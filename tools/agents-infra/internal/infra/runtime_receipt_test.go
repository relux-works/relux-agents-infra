package infra

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func readReceipt(t *testing.T, agentsDir string) RuntimeReceipt {
	t.Helper()
	data, err := os.ReadFile(runtimeReceiptPath(agentsDir))
	if err != nil {
		t.Fatalf("read install receipt: %v", err)
	}
	var receipt RuntimeReceipt
	if err := json.Unmarshal(data, &receipt); err != nil {
		t.Fatalf("decode install receipt: %v", err)
	}
	return receipt
}

func TestSetupMintsReceiptAndVerifiesTheRuntimeItInstalled(t *testing.T) {
	source := seedSourceRepo(t)
	project := t.TempDir()
	layout, err := LocalLayout(source, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}

	if err := Setup(Options{Layout: layout, Stdout: io.Discard}); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	if err := VerifyInstalledRuntime(layout); err != nil {
		t.Fatalf("VerifyInstalledRuntime after a successful Setup: %v", err)
	}
	receipt := readReceipt(t, layout.AgentsDir)
	if receipt.Schema != runtimeReceiptSchema {
		t.Fatalf("receipt schema = %d, want %d", receipt.Schema, runtimeReceiptSchema)
	}
	if !samePath(receipt.AgentsDir, layout.AgentsDir) {
		t.Fatalf("receipt agentsDir = %q, want %q", receipt.AgentsDir, layout.AgentsDir)
	}
}

func TestSetupRepairsAndVerifyNarrowsEveryCanonicalTargetAlias(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX alias file-kind and executable-mode contract")
	}
	source := seedSourceRepo(t)
	project := t.TempDir()
	layout, err := LocalLayout(source, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}
	if err := Setup(Options{Layout: layout, Stdout: io.Discard}); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	mutations := []struct {
		entrypoint string
		mutate     func(t *testing.T, path string)
	}{
		{
			entrypoint: "openai-infra",
			mutate: func(t *testing.T, path string) {
				if err := os.Remove(path); err != nil {
					t.Fatalf("Remove(%s): %v", path, err)
				}
			},
		},
		{
			entrypoint: "anthropic-infra",
			mutate: func(t *testing.T, path string) {
				if err := os.Remove(path); err != nil {
					t.Fatalf("Remove(%s): %v", path, err)
				}
				if err := os.Mkdir(path, 0o755); err != nil {
					t.Fatalf("Mkdir(%s): %v", path, err)
				}
			},
		},
		{
			entrypoint: "qwen-infra",
			mutate: func(t *testing.T, path string) {
				if err := os.Chmod(path, 0o644); err != nil {
					t.Fatalf("Chmod(%s): %v", path, err)
				}
			},
		},
	}
	for _, mutation := range mutations {
		t.Run(mutation.entrypoint, func(t *testing.T) {
			path := filepath.Join(layout.BinDir, canonicalTargetWrapperName(mutation.entrypoint, runtime.GOOS))
			mutation.mutate(t, path)
			err := VerifyInstalledRuntime(layout)
			if err == nil || !strings.Contains(err.Error(), mutation.entrypoint) {
				t.Fatalf("VerifyInstalledRuntime error = %v, want %s-specific refusal", err, mutation.entrypoint)
			}
			if err := Setup(Options{Layout: layout, Stdout: io.Discard}); err != nil {
				t.Fatalf("Setup repair: %v", err)
			}
			if err := VerifyInstalledRuntime(layout); err != nil {
				t.Fatalf("VerifyInstalledRuntime after %s repair: %v", mutation.entrypoint, err)
			}
		})
	}
}

func TestCanonicalConfigurationFailurePreventsSetupAndVerifyMutation(t *testing.T) {
	source := seedSourceRepo(t)
	project := t.TempDir()
	configPath := writeCanonicalConfig(t, project, "[agents.targets.openai]\nvendor=7\n")
	before := mustReadRuntimeFile(t, configPath)
	layout, err := LocalLayout(source, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}

	err = Setup(Options{Layout: layout, Stdout: io.Discard})
	if err == nil || !strings.Contains(err.Error(), PrimarySessionErrorInvalidProjectConfiguration) || !strings.Contains(err.Error(), "agents.targets.openai.vendor") || !strings.Contains(err.Error(), configPath) || !strings.Contains(err.Error(), "Remediation:") {
		t.Fatalf("Setup error = %v", err)
	}
	if got := mustReadRuntimeFile(t, configPath); string(got) != string(before) {
		t.Fatalf("Setup rewrote project config: before=%q after=%q", before, got)
	}
	for _, entrypoint := range canonicalTargetLauncherNames {
		if _, statErr := os.Lstat(filepath.Join(layout.BinDir, canonicalTargetWrapperName(entrypoint, runtime.GOOS))); !os.IsNotExist(statErr) {
			t.Fatalf("Setup created %s before rejecting config: %v", entrypoint, statErr)
		}
	}

	if err := os.Remove(configPath); err != nil {
		t.Fatalf("Remove(%s): %v", configPath, err)
	}
	if err := Setup(Options{Layout: layout, Stdout: io.Discard}); err != nil {
		t.Fatalf("baseline Setup: %v", err)
	}
	mustWrite(t, configPath, "[agents.targets.openai]\nvendor=7\n")
	beforeVerify := mustReadRuntimeFile(t, configPath)
	err = VerifyInstalledRuntime(layout)
	if err == nil || !strings.Contains(err.Error(), "agents.targets.openai.vendor") || !strings.Contains(err.Error(), configPath) || !strings.Contains(err.Error(), "Remediation:") {
		t.Fatalf("VerifyInstalledRuntime error = %v", err)
	}
	if got := mustReadRuntimeFile(t, configPath); string(got) != string(beforeVerify) {
		t.Fatalf("VerifyInstalledRuntime rewrote project config: before=%q after=%q", beforeVerify, got)
	}
}

// Negative: the generated launcher builds its recorded source dir on every
// invocation. Once that source can no longer provide the backend, the runtime
// is broken and verification must say so, even though every installed file is
// still exactly where setup left it.
func TestVerifyInstalledRuntimeRefusesLauncherWhoseBackendDisappeared(t *testing.T) {
	source := seedSourceRepo(t)
	project := t.TempDir()
	layout, err := LocalLayout(source, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}
	if err := Setup(Options{Layout: layout, Stdout: io.Discard}); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	if err := os.RemoveAll(filepath.Join(source, "tools")); err != nil {
		t.Fatalf("RemoveAll(tools): %v", err)
	}

	err = VerifyInstalledRuntime(layout)
	if err == nil {
		t.Fatal("VerifyInstalledRuntime accepted a launcher whose backend is gone")
	}
	for _, want := range []string{"agents-infra launcher", "tools/agents-infra/go.mod"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q missing %q", err, want)
		}
	}
}

func TestVerifyInstalledRuntimeRefusesMissingAndDriftedPiInfraAlias(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(t *testing.T, layout Layout)
		want   string
	}{
		{
			name: "missing alias",
			mutate: func(t *testing.T, layout Layout) {
				if err := os.Remove(filepath.Join(layout.BinDir, piInfraWrapperName(runtime.GOOS))); err != nil {
					t.Fatalf("Remove(pi-infra): %v", err)
				}
			},
			want: "no generated pi-infra launcher",
		},
		{
			name: "drifted alias target",
			mutate: func(t *testing.T, layout Layout) {
				path := filepath.Join(layout.BinDir, piInfraWrapperName(runtime.GOOS))
				mustWrite(t, path, strings.ReplaceAll(piInfraWrapperBody(runtime.GOOS, piInfraTargetName(layout.Mode, runtime.GOOS)), piInfraTargetName(layout.Mode, runtime.GOOS), "other-infra"))
			},
			want: "has drifted from the managed",
		},
		{
			name: "byte-identical symlink alias",
			mutate: func(t *testing.T, layout Layout) {
				path := filepath.Join(layout.BinDir, piInfraWrapperName(runtime.GOOS))
				external := filepath.Join(t.TempDir(), piInfraWrapperName(runtime.GOOS))
				mustWrite(t, external, string(mustReadRuntimeFile(t, path)))
				if err := os.Remove(path); err != nil {
					t.Fatalf("Remove(pi-infra): %v", err)
				}
				if err := os.Symlink(external, path); err != nil {
					t.Fatalf("Symlink(pi-infra): %v", err)
				}
			},
			want: "pi-infra launcher",
		},
		{
			name: "missing sibling target",
			mutate: func(t *testing.T, layout Layout) {
				if err := os.Remove(filepath.Join(layout.BinDir, piInfraTargetName(layout.Mode, runtime.GOOS))); err != nil {
					t.Fatalf("Remove(agents-infra): %v", err)
				}
			},
			want: "pi-infra launcher target is missing",
		},
		{
			name: "byte-identical symlink sibling target",
			mutate: func(t *testing.T, layout Layout) {
				path := filepath.Join(layout.BinDir, piInfraTargetName(layout.Mode, runtime.GOOS))
				external := filepath.Join(t.TempDir(), piInfraTargetName(layout.Mode, runtime.GOOS))
				mustWrite(t, external, string(mustReadRuntimeFile(t, path)))
				if err := os.Remove(path); err != nil {
					t.Fatalf("Remove(agents-infra): %v", err)
				}
				if err := os.Symlink(external, path); err != nil {
					t.Fatalf("Symlink(agents-infra): %v", err)
				}
			},
			want: "pi-infra launcher target is not a regular file",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			source := seedSourceRepo(t)
			project := t.TempDir()
			layout, err := LocalLayout(source, project)
			if err != nil {
				t.Fatalf("LocalLayout: %v", err)
			}
			if err := Setup(Options{Layout: layout, Stdout: io.Discard}); err != nil {
				t.Fatalf("Setup: %v", err)
			}
			test.mutate(t, layout)
			err = VerifyInstalledRuntime(layout)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("VerifyInstalledRuntime error = %v, want %q", err, test.want)
			}
		})
	}
}

func mustReadRuntimeFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}
	return data
}

func TestSetupRepairsPiInfraAliasDrift(t *testing.T) {
	source := seedSourceRepo(t)
	project := t.TempDir()
	layout, err := LocalLayout(source, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}
	if err := Setup(Options{Layout: layout, Stdout: io.Discard}); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	aliasPath := filepath.Join(layout.BinDir, piInfraWrapperName(runtime.GOOS))
	mustWrite(t, aliasPath, "drifted alias\n")
	if err := VerifyInstalledRuntime(layout); err == nil {
		t.Fatal("VerifyInstalledRuntime accepted drifted pi-infra alias")
	}
	if err := Setup(Options{Layout: layout, Stdout: io.Discard}); err != nil {
		t.Fatalf("Setup repair: %v", err)
	}
	if err := VerifyInstalledRuntime(layout); err != nil {
		t.Fatalf("VerifyInstalledRuntime after setup repair: %v", err)
	}
}

func TestVerifyInstalledRuntimeRefusesIncorrectGlobalPiInfraTarget(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX executable identity probe")
	}
	source := seedSourceRepo(t)
	home := t.TempDir()
	layout, err := GlobalLayout(source, home)
	if err != nil {
		t.Fatalf("GlobalLayout: %v", err)
	}
	seedGlobalAgentsInfraTarget(t, layout)
	if err := Setup(Options{Layout: layout, Stdout: io.Discard}); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	target := filepath.Join(layout.BinDir, piInfraTargetName(ModeGlobal, runtime.GOOS))
	mustWrite(t, target, "#!/usr/bin/env sh\nexit 0\n")
	if err := os.Chmod(target, 0o755); err != nil {
		t.Fatalf("Chmod(%s): %v", target, err)
	}
	err = VerifyInstalledRuntime(layout)
	if err == nil || !strings.Contains(err.Error(), "does not start as agents-infra") || !strings.Contains(err.Error(), "does not identify it as agents-infra") {
		t.Fatalf("VerifyInstalledRuntime error = %v, want incorrect global target refusal", err)
	}
}

func TestVerifyInstalledRuntimeRefusesPiCatalogManifestDrift(t *testing.T) {
	source := seedSourceRepo(t)
	project := t.TempDir()
	layout, err := LocalLayout(source, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}
	if err := Setup(Options{Layout: layout, Stdout: io.Discard}); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	manifest := filepath.Join(layout.AgentsDir, "tools", "agents-infra", "internal", "infra", "pi-v0.84.2-darwin-arm64-tree-manifest.txt")
	data, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatalf("ReadFile(manifest): %v", err)
	}
	data[0] ^= 1
	if err := os.WriteFile(manifest, data, 0o644); err != nil {
		t.Fatalf("WriteFile(manifest): %v", err)
	}
	err = VerifyInstalledRuntime(layout)
	if err == nil || !strings.Contains(err.Error(), "release-tree catalog") || !strings.Contains(err.Error(), "has drifted") {
		t.Fatalf("VerifyInstalledRuntime error = %v, want catalog drift refusal", err)
	}
}

// Negative: a receipt is a claim about one destination. Copying it next to a
// different tree must not transfer the claim, or the receipt becomes exactly the
// kind of self-minted evidence it exists to replace.
func TestVerifyInstalledRuntimeRefusesReceiptMintedForAnotherDestination(t *testing.T) {
	source := seedSourceRepo(t)
	installed := t.TempDir()
	installedLayout, err := LocalLayout(source, installed)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}
	if err := Setup(Options{Layout: installedLayout, Stdout: io.Discard}); err != nil {
		t.Fatalf("Setup: %v", err)
	}

	// A hand-assembled destination carrying a complete tree and a receipt lifted
	// from the real install.
	forged := t.TempDir()
	forgedLayout, err := LocalLayout(source, forged)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}
	if err := copyTree(installedLayout.AgentsDir, forgedLayout.AgentsDir); err != nil {
		t.Fatalf("copy installed runtime: %v", err)
	}
	if err := copyTree(installedLayout.BinDir, forgedLayout.BinDir); err != nil {
		t.Fatalf("copy installed bin dir: %v", err)
	}

	err = VerifyInstalledRuntime(forgedLayout)
	if err == nil {
		t.Fatal("VerifyInstalledRuntime accepted a receipt minted for another destination")
	}
	if !strings.Contains(err.Error(), "was minted for") {
		t.Fatalf("error %q does not name the destination mismatch", err)
	}
}

// Negative: this is the partial-write shape. Setup fails after it has already
// rewritten the destination; the caller sees a populated .agents directory. The
// run must have dropped the previous receipt before mutating anything, so what
// is left cannot pass verification and cannot be presented as a usable runtime.
func TestSetupFailingLateLeavesNoRuntimeThatVerifies(t *testing.T) {
	source := seedSourceRepo(t)
	project := t.TempDir()
	layout, err := LocalLayout(source, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}
	if err := Setup(Options{Layout: layout, Stdout: io.Discard}); err != nil {
		t.Fatalf("first Setup: %v", err)
	}
	if err := VerifyInstalledRuntime(layout); err != nil {
		t.Fatalf("VerifyInstalledRuntime after the first Setup: %v", err)
	}

	// Make a step that runs after the sync fail: .claude cannot be a directory.
	if err := os.RemoveAll(layout.ClaudeDir); err != nil {
		t.Fatalf("RemoveAll(.claude): %v", err)
	}
	mustWrite(t, layout.ClaudeDir, "not a directory\n")

	if err := Setup(Options{Layout: layout, Stdout: io.Discard}); err == nil {
		t.Fatal("Setup succeeded despite a destination it cannot write")
	}
	if _, statErr := os.Stat(layout.AgentsDir); statErr != nil {
		t.Fatalf("expected the failed run to have already written the destination: %v", statErr)
	}
	if _, statErr := os.Stat(runtimeReceiptPath(layout.AgentsDir)); !os.IsNotExist(statErr) {
		t.Fatalf("a failed run left the previous receipt vouching for a half-rewritten runtime: %v", statErr)
	}
	if err := VerifyInstalledRuntime(layout); err == nil {
		t.Fatal("a partially installed runtime verified as usable")
	}
}

// Negative: a usable source is not a finished install. With --no-sync against a
// destination that was only ever partly populated, every input check passes and
// the tree that ends up installed is still incomplete. Setup must run its
// postcondition against what it produced, not against what it was handed.
func TestSetupRefusesToMintAReceiptForARuntimeItDidNotFinishInstalling(t *testing.T) {
	source := seedSourceRepo(t)
	project := t.TempDir()
	layout, err := LocalLayout(source, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}
	// A stale destination: instructions, configs and rules are there, the module
	// the generated launcher builds is not.
	mustMkdir(t, filepath.Join(layout.AgentsDir, ".instructions"))
	mustMkdir(t, filepath.Join(layout.AgentsDir, ".configs"))
	mustMkdir(t, filepath.Join(layout.AgentsDir, ".rules"))
	mustWrite(t, filepath.Join(layout.AgentsDir, ".instructions", "INSTRUCTIONS.md"), "# Instructions\n")
	mustWrite(t, filepath.Join(layout.AgentsDir, ".instructions", "AGENTS.md"), "# Agents\n")
	mustWrite(t, filepath.Join(layout.AgentsDir, "SKILL.md"), "# relux-agents-infra\n")
	mustWrite(t, filepath.Join(layout.AgentsDir, "README.md"), "# relux-agents-infra\n")

	err = Setup(Options{Layout: layout, NoSync: true, Stdout: io.Discard})
	if err == nil {
		t.Fatal("Setup reported success for a runtime it never finished installing")
	}
	if !strings.Contains(err.Error(), "installed runtime is missing tools/agents-infra") {
		t.Fatalf("error %q does not name the incomplete installed tree", err)
	}
	if _, statErr := os.Stat(runtimeReceiptPath(layout.AgentsDir)); !os.IsNotExist(statErr) {
		t.Fatalf("Setup minted a receipt for an incomplete runtime: %v", statErr)
	}
}

// A source tree cannot ship a receipt: sync must never carry one into a
// destination, or any tree could vouch for whatever it is copied into.
func TestSyncNeverCopiesAnInstallReceiptFromTheSource(t *testing.T) {
	source := seedSourceRepo(t)
	mustWrite(t, filepath.Join(source, runtimeReceiptFileName), `{"schema":1,"agentsDir":"/anywhere"}`)
	project := t.TempDir()
	layout, err := LocalLayout(source, project)
	if err != nil {
		t.Fatalf("LocalLayout: %v", err)
	}

	if err := Setup(Options{Layout: layout, Stdout: io.Discard}); err != nil {
		t.Fatalf("Setup: %v", err)
	}
	receipt := readReceipt(t, layout.AgentsDir)
	if !samePath(receipt.AgentsDir, layout.AgentsDir) {
		t.Fatalf("receipt agentsDir = %q, want the destination %q; the source's receipt was copied through", receipt.AgentsDir, layout.AgentsDir)
	}
}

func copyTree(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode())
	})
}
