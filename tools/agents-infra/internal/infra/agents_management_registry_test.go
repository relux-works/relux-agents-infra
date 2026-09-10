package infra

import (
	"os"
	"strings"
	"testing"

	"github.com/relux-works/skill-agents-management/pkg/agentic"

	// Registered ONLY in this test binary, never in the production import
	// graph (agents_management_registry.go blank-imports codex and claude,
	// never gemini). That asymmetry is the point: it makes gemini-cli a real
	// agentic.Default registration this program still has no launcher for,
	// which is exactly the "registered but unlaunchable" case
	// ValidateLaunchableEnvironment must refuse.
	geminisystem "github.com/relux-works/skill-agents-management/pkg/agentic/systems/gemini"
)

// TestValidateLaunchableEnvironmentGoldenText locks the exact wording
// project_config.go's environment field refusal has always used. The
// refactor that routes admission through ValidateLaunchableEnvironment must
// not change what an operator sees for the same bad input.
func TestValidateLaunchableEnvironmentGoldenText(t *testing.T) {
	err := ValidateLaunchableEnvironment("other")
	if err == nil {
		t.Fatal("ValidateLaunchableEnvironment accepted an unregistered identifier")
	}
	const want = "must be one of codex, claude-code, pi"
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
}

// TestValidateLaunchableProviderGoldenText locks the exact wording
// primary_session_launch_plan.go's provider refusal has always used.
func TestValidateLaunchableProviderGoldenText(t *testing.T) {
	err := ValidateLaunchableProvider("gemini")
	if err == nil {
		t.Fatal("ValidateLaunchableProvider accepted an unregistered provider")
	}
	const want = `unsupported provider "gemini"`
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
}

// TestValidateLaunchableEnvironmentAdmitsExactlyTheDeclaredThree is the
// positive control every negative case below needs: the three environments
// this program actually launches must still be admitted against the real,
// production agentic.Default registry.
func TestValidateLaunchableEnvironmentAdmitsExactlyTheDeclaredThree(t *testing.T) {
	for _, environment := range []string{"codex", "claude-code", "pi"} {
		if err := ValidateLaunchableEnvironment(environment); err != nil {
			t.Errorf("ValidateLaunchableEnvironment(%q) = %v, want nil", environment, err)
		}
	}
}

// TestValidateLaunchableEnvironmentRefusesRegisteredButUnlaunchableSystem is
// the case named in the acceptance criteria: gemini-cli is a REAL
// agentic.Default registration in this test binary (see the geminisystem
// import above), not a typo or an absence. Admission must still refuse it,
// because registry membership is necessary but never sufficient — this
// program has no launcher for it.
func TestValidateLaunchableEnvironmentRefusesRegisteredButUnlaunchableSystem(t *testing.T) {
	geminiID := geminisystem.New().ID()
	if _, ok := agentic.Default.Lookup(geminiID); !ok {
		t.Fatalf("test setup: %q is not registered in agentic.Default; the geminisystem import did not run its init", geminiID)
	}
	if err := ValidateLaunchableEnvironment(string(geminiID)); err == nil {
		t.Fatalf("ValidateLaunchableEnvironment(%q) admitted a registered-but-unlaunchable system", geminiID)
	}
}

// TestValidateProjectTargetRefusesRegisteredButUnlaunchableEnvironmentAtTheRealCallSite
// drives project_config.go's actual validateProjectTarget entry point, not
// ValidateLaunchableEnvironment in isolation. This is the test that closes
// the gap a source-text scanner cannot: a mutant that stops calling
// ValidateLaunchableEnvironment and instead inlines a DIFFERENTLY content'd
// literal set — for example one that also admits "gemini-cli" — cannot be
// distinguished from the real code by grepping source text (the forbidden
// three-item literal this file's source-scan test looks for never appears),
// but it changes what an operator seeing this exact call site can get
// admitted. Driving validateProjectTarget itself, with gemini-cli genuinely
// registered in this test binary, is what still catches it.
//
// The assertion is on the SPECIFIC environment-gate wording, not merely on
// err != nil: any environment/vendor pair the trailing switch does not
// explicitly admit is refused there too ("vendor/environment pair ... is not
// admitted"), which would mask a mutant that let gemini-cli past the
// environment gate itself. Asserting the environment gate's own wording is
// what proves THIS gate, not a downstream one, did the refusing.
func TestValidateProjectTargetRefusesRegisteredButUnlaunchableEnvironmentAtTheRealCallSite(t *testing.T) {
	geminiID := geminisystem.New().ID()
	if _, ok := agentic.Default.Lookup(geminiID); !ok {
		t.Fatalf("test setup: %q is not registered in agentic.Default", geminiID)
	}
	err := validateProjectTarget(ProjectTarget{Vendor: "openai", Environment: string(geminiID)})
	if err == nil {
		t.Fatalf("validateProjectTarget admitted a registered-but-unlaunchable environment %q", geminiID)
	}
	const wantSubstring = "must be one of codex, claude-code, pi"
	if !strings.Contains(err.Error(), wantSubstring) {
		t.Fatalf("error = %q, want it to contain %q (the environment gate's own wording, not a downstream refusal)", err.Error(), wantSubstring)
	}
}

// TestValidateLaunchableEnvironmentRefusesCaseAndAliasVariants proves the
// cross-check requires an EXACT spelling match, not agentic.NormalizeSystemID's
// folded one: a plugin's own id must already be its canonical spelling, but
// admission here must not fold an operator's or a configuration's differently
// spelled input onto that canonical form and admit it anyway.
func TestValidateLaunchableEnvironmentRefusesCaseAndAliasVariants(t *testing.T) {
	for _, variant := range []string{"Codex", "CODEX", "Claude-Code", "claude", "CLAUDE-CODE", "Pi", " codex", "codex "} {
		if err := ValidateLaunchableEnvironment(variant); err == nil {
			t.Errorf("ValidateLaunchableEnvironment(%q) admitted a case/alias variant", variant)
		}
	}
}

// TestValidateLaunchableProviderRefusesUnadmittedProvider covers the provider
// label surface primary_session_launch_plan.go dispatches on, which is
// spelled differently from the environment id for Claude (claude vs
// claude-code) and therefore needs its own case/unregistered coverage.
func TestValidateLaunchableProviderRefusesUnadmittedProvider(t *testing.T) {
	for _, provider := range []string{"Codex", "claude-code", "gemini", "", "PI"} {
		if err := ValidateLaunchableProvider(provider); err == nil {
			t.Errorf("ValidateLaunchableProvider(%q) admitted an unadmitted provider", provider)
		}
	}
	for _, provider := range []string{"codex", "claude", "pi"} {
		if err := ValidateLaunchableProvider(provider); err != nil {
			t.Errorf("ValidateLaunchableProvider(%q) = %v, want nil", provider, err)
		}
	}
}

// TestValidateLaunchableEnvironmentRegistryCrossCheckIsLoadBearing is the
// mutant-resistance test for the registry half of admission. It drives the
// unexported core function directly with an EMPTY injected registry so it
// cannot be satisfied by agentic.Default's real, always-populated state.
//
// "codex" is declared in launchableSystems, so a version of
// validateLaunchableEnvironment that dropped the `lookup` call — keeping only
// the declaration-membership check — would admit it here. That is exactly the
// narrowing mutant this test exists to fail: the registry cross-check is
// supposed to be able to refuse a declared identifier that never actually
// registered, and an implementation that silently trusts the declaration
// alone cannot express that refusal.
func TestValidateLaunchableEnvironmentRegistryCrossCheckIsLoadBearing(t *testing.T) {
	emptyRegistry := agentic.NewRegistry()
	if err := validateLaunchableEnvironment("codex", emptyRegistry.Lookup); err == nil {
		t.Fatal("validateLaunchableEnvironment admitted a declared identifier against an empty registry; the registry cross-check is not being consulted")
	}
	// The declaration-membership half must still run first and independently:
	// an identifier absent from launchableSystems is refused even against a
	// registry that (hypothetically) carries it.
	fullRegistry := agentic.NewRegistry()
	if err := fullRegistry.Register(geminisystem.New()); err != nil {
		t.Fatalf("registering gemini into an isolated registry: %v", err)
	}
	if err := validateLaunchableEnvironment("gemini-cli", fullRegistry.Lookup); err == nil {
		t.Fatal("validateLaunchableEnvironment admitted an identifier absent from launchableSystems merely because a registry carried it")
	}
}

// TestValidateLaunchableProviderRegistryCrossCheckIsLoadBearing mirrors the
// environment-side mutant-resistance test for the provider admission path.
func TestValidateLaunchableProviderRegistryCrossCheckIsLoadBearing(t *testing.T) {
	emptyRegistry := agentic.NewRegistry()
	if err := validateLaunchableProvider("codex", emptyRegistry.Lookup); err == nil {
		t.Fatal("validateLaunchableProvider admitted a declared provider against an empty registry; the registry cross-check is not being consulted")
	}
}

// TestLaunchableSystemsAccessorsMatchDeclarationOrder pins the declaration
// order LaunchableEnvironments and LaunchableProviders expose, since
// project_config.go's error text joins LaunchableEnvironments() directly.
func TestLaunchableSystemsAccessorsMatchDeclarationOrder(t *testing.T) {
	if got, want := LaunchableEnvironments(), []string{"codex", "claude-code", "pi"}; !equalStringSlices(got, want) {
		t.Fatalf("LaunchableEnvironments() = %v, want %v", got, want)
	}
	if got, want := LaunchableProviders(), []string{"codex", "claude", "pi"}; !equalStringSlices(got, want) {
		t.Fatalf("LaunchableProviders() = %v, want %v", got, want)
	}
}

// TestLaunchableSystemSourceHasNoReintroducedLiteralList is a source-scan
// guard mirroring skill-agents-management's own singlesource_guard_test.go:
// behavioral tests cannot distinguish "calls the shared declaration" from "a
// verbatim copy of the same three strings restated inline", because both
// admit and refuse exactly the same inputs. This test reads the admission and
// dispatch sites' own source text and fails if the specific literal
// constructs this task removed ever reappear.
func TestLaunchableSystemSourceHasNoReintroducedLiteralList(t *testing.T) {
	forbidden := []string{
		`[]string{"codex", "claude-code", "pi"}`,
		`provider != "codex" && provider != "claude" && provider != "pi"`,
		"case \"claude-code\":\n\t\treturn \"claude\"",
		"case \"codex\":\n\t\treturn lockCodexTargetArguments",
		"case \"claude\":\n\t\terr = buildClaudePrimarySessionLaunchPlan",
	}
	for _, path := range []string{"project_config.go", "canonical_target.go", "primary_session_launch_plan.go"} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)
		}
		text := string(data)
		for _, literal := range forbidden {
			if strings.Contains(text, literal) {
				t.Errorf("%s re-introduces the hardcoded launchable-system construct %q; route admission and dispatch through launchableSystems in agents_management_registry.go instead", path, literal)
			}
		}
	}
}
