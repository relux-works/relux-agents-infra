//go:build !windows

package infra

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// Production call sites: PiLegacyRetirementDryRun and PiLegacyRetire. The
// caller must confirm the stable hash of the complete bounded candidate plan;
// an arbitrary or stale caller-minted hash must preserve every legacy file.
func TestPiLegacyRetirementRequiresStableFullPlanHash(t *testing.T) {
	paths := newPiLifecycleTestState(t)
	policy := testPiLifecyclePolicy()
	seedClosedPiLifecycleEntry(t, paths, policy)
	legacy := []string{
		filepath.Join(paths.LogsDir, "legacy-a.jsonl"),
		filepath.Join(paths.LogsDir, "legacy-b.jsonl"),
	}
	for _, path := range legacy {
		if err := os.WriteFile(path, []byte("{}\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	first, err := PiLegacyRetirementDryRun(context.Background(), paths, policy, "test")
	if err != nil {
		t.Fatal(err)
	}
	second, err := PiLegacyRetirementDryRun(context.Background(), paths, policy, "test")
	if err != nil {
		t.Fatal(err)
	}
	if first.PlanHash == "" || first.PlanHash != second.PlanHash || !reflect.DeepEqual(first.Candidates, second.Candidates) {
		t.Fatalf("dry-run plan is not stable: first=%+v second=%+v", first, second)
	}
	if len(first.Candidates) != len(legacy) || !first.ScanComplete || !first.DryRun {
		t.Fatalf("dry-run did not project the exact bounded plan: %+v", first)
	}

	if _, err := PiLegacyRetire(context.Background(), paths, policy, "test", "caller-minted-plan-hash"); !piErrorIs(err, "lifecycle_legacy_confirmation_mismatch") {
		t.Fatalf("caller-minted confirmation admitted: %v", err)
	}
	for _, path := range legacy {
		if _, err := os.Lstat(path); err != nil {
			t.Fatalf("wrong confirmation mutated %s: %v", path, err)
		}
	}

	result, err := PiLegacyRetire(context.Background(), paths, policy, "test", first.PlanHash)
	if err != nil {
		t.Fatal(err)
	}
	if result.PlanHash != first.PlanHash || result.RetiredCount != len(legacy) || !result.Status.ScanComplete || !result.Status.WithinPolicy || !result.Status.SoakReady {
		t.Fatalf("confirmed retirement result=%+v", result)
	}
	for _, path := range legacy {
		if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("confirmed candidate survived %s: %v", path, err)
		}
	}
}

// Production call site: PiLegacyRetire -> retirePiLegacyCandidate. The held
// file and its descriptor-relative path must be revalidated immediately before
// unlink; a same-mode replacement is unknown evidence and must survive.
func TestPiLegacyRetirementPreservesLastGapCandidateSubstitution(t *testing.T) {
	paths := newPiLifecycleTestState(t)
	policy := testPiLifecyclePolicy()
	seedClosedPiLifecycleEntry(t, paths, policy)
	first := filepath.Join(paths.LogsDir, "legacy-a-first.jsonl")
	if err := os.WriteFile(first, []byte("first\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	legacy := filepath.Join(paths.LogsDir, "legacy-z-substitute.jsonl")
	if err := os.WriteFile(legacy, []byte("original\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	plan, err := PiLegacyRetirementDryRun(context.Background(), paths, policy, "test")
	if err != nil {
		t.Fatal(err)
	}

	originalHook := piLifecycleBeforeLegacyCandidateUnlink
	t.Cleanup(func() { piLifecycleBeforeLegacyCandidateUnlink = originalHook })
	piLifecycleBeforeLegacyCandidateUnlink = func(parentFD int, name string, index int) error {
		if index != 1 {
			return nil
		}
		preserved := legacy + ".preserved"
		if err := os.Rename(legacy, preserved); err != nil {
			return err
		}
		return os.WriteFile(legacy, []byte("replacement\n"), 0o600)
	}

	if _, err := PiLegacyRetire(context.Background(), paths, policy, "test", plan.PlanHash); !piErrorIs(err, "lifecycle_log_evidence_unknown") {
		t.Fatalf("last-gap substitution admitted: %v", err)
	}
	content, err := os.ReadFile(legacy)
	if err != nil || string(content) != "replacement\n" {
		t.Fatalf("replacement evidence was not preserved: content=%q err=%v", content, err)
	}
	if _, err := os.Lstat(legacy + ".preserved"); err != nil {
		t.Fatalf("original evidence was not preserved: %v", err)
	}
}

// Production call site: PiLegacyRetire rebuilds the complete plan while both
// lifecycle locks are held. Candidate or directory changes after dry-run make
// the caller's former hash stale and preserve all observed evidence.
func TestPiLegacyRetirementRejectsStaleCandidateAndDirectoryPlan(t *testing.T) {
	for _, mutate := range []string{"candidate", "directory"} {
		t.Run(mutate, func(t *testing.T) {
			paths := newPiLifecycleTestState(t)
			policy := testPiLifecyclePolicy()
			seedClosedPiLifecycleEntry(t, paths, policy)
			legacy := filepath.Join(paths.LogsDir, "legacy-stale.jsonl")
			if err := os.WriteFile(legacy, []byte("before\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			plan, err := PiLegacyRetirementDryRun(context.Background(), paths, policy, "test")
			if err != nil {
				t.Fatal(err)
			}
			if mutate == "candidate" {
				if err := os.WriteFile(legacy, []byte("after-longer\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			} else {
				preserved := paths.LogsDir + ".preserved"
				if err := os.Rename(paths.LogsDir, preserved); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(paths.LogsDir, 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(paths.LogsDir, filepath.Base(legacy)), []byte("replacement\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := PiLegacyRetire(context.Background(), paths, policy, "test", plan.PlanHash); !piErrorIs(err, "lifecycle_legacy_confirmation_mismatch") {
				t.Fatalf("stale %s plan admitted: %v", mutate, err)
			}
			if _, err := os.Lstat(filepath.Join(paths.LogsDir, filepath.Base(legacy))); err != nil {
				t.Fatalf("stale %s evidence was not preserved: %v", mutate, err)
			}
		})
	}
}

// Production call site: PiLegacyRetirementDryRun. Narrowing the mutation cap
// from two candidates to one must refuse the full plan instead of silently
// hashing or deleting a prefix.
func TestPiLegacyRetirementPlanRefusesNarrowedCandidateBound(t *testing.T) {
	paths := newPiLifecycleTestState(t)
	policy := testPiLifecyclePolicy()
	seedClosedPiLifecycleEntry(t, paths, policy)
	for _, name := range []string{"legacy-a.jsonl", "legacy-b.jsonl"} {
		if err := os.WriteFile(filepath.Join(paths.LogsDir, name), []byte("{}\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	policy.MaxMutationsPerOperation = 3 // one rename+unlink pair, not two
	if plan, err := PiLegacyRetirementDryRun(context.Background(), paths, policy, "test"); !piErrorIs(err, "lifecycle_legacy_plan_exhausted") || len(plan.Candidates) != 0 {
		t.Fatalf("narrowed candidate bound admitted: plan=%+v err=%v", plan, err)
	}
}

// Production call site: PiLegacyRetirementDryRun. A candidate set inside the
// mutation count but outside the bounded odd-generation document is refused at
// dry-run; confirmation must never discover that the emitted plan is not
// persistable.
func TestPiLegacyRetirementDryRunRefusesOversizePersistedAuthority(t *testing.T) {
	paths := newPiLifecycleTestState(t)
	policy := testPiLifecyclePolicy()
	seedClosedPiLifecycleEntry(t, paths, policy)
	policy.MaxMutationsPerOperation = 100
	for index := 0; index < 20; index++ {
		name := fmt.Sprintf("legacy-oversize-%02d.jsonl", index)
		if err := os.WriteFile(filepath.Join(paths.LogsDir, name), []byte("{}\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	plan, err := PiLegacyRetirementDryRun(context.Background(), paths, policy, "test")
	if !piErrorIs(err, "lifecycle_legacy_plan_exhausted") || !plan.ScanComplete || plan.PlanHash == "" || len(plan.Candidates) != 20 {
		t.Fatalf("oversize persisted authority was emitted as executable: plan=%+v err=%v", plan, err)
	}
	for _, candidate := range plan.Candidates {
		if _, err := os.Lstat(filepath.Join(paths.ProfileRoot, filepath.FromSlash(candidate.Path))); err != nil {
			t.Fatalf("oversize dry-run mutated %s: %v", candidate.Path, err)
		}
	}
}

// Production call site: PiLegacyRetirementDryRun. A foreign, linked, or
// mode-narrowed legacy-looking file is unknown evidence, never an absent or
// deletable candidate.
func TestPiLegacyRetirementPreservesUnknownLegacyEvidence(t *testing.T) {
	for _, shape := range []string{"mode", "symlink"} {
		t.Run(shape, func(t *testing.T) {
			paths := newPiLifecycleTestState(t)
			policy := testPiLifecyclePolicy()
			seedClosedPiLifecycleEntry(t, paths, policy)
			legacy := filepath.Join(paths.LogsDir, "legacy-unknown.jsonl")
			if shape == "mode" {
				if err := os.WriteFile(legacy, []byte("{}\n"), 0o640); err != nil {
					t.Fatal(err)
				}
			} else {
				target := legacy + ".target"
				if err := os.WriteFile(target, []byte("{}\n"), 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, legacy); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := PiLegacyRetirementDryRun(context.Background(), paths, policy, "test"); !piErrorIs(err, "lifecycle_log_evidence_unknown") {
				t.Fatalf("unknown %s evidence admitted: %v", shape, err)
			}
			if _, err := os.Lstat(legacy); err != nil {
				t.Fatalf("unknown %s evidence was not preserved: %v", shape, err)
			}
		})
	}
}

// Production call site: PiLegacyRetire -> resumePiLegacyRetirement. A crash
// after the operation-bound rename leaves an odd generation that the same
// exact confirmation resumes without minting new authority.
func TestPiLegacyRetirementResumesCrashAfterFencedRename(t *testing.T) {
	paths := newPiLifecycleTestState(t)
	policy := testPiLifecyclePolicy()
	seedClosedPiLifecycleEntry(t, paths, policy)
	legacy := filepath.Join(paths.LogsDir, "legacy-crash.jsonl")
	if err := os.WriteFile(legacy, []byte("crash\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	plan, err := PiLegacyRetirementDryRun(context.Background(), paths, policy, "test")
	if err != nil {
		t.Fatal(err)
	}

	originalHook := piLifecycleAfterLegacyCandidateRename
	t.Cleanup(func() { piLifecycleAfterLegacyCandidateRename = originalHook })
	crash := errors.New("simulated crash after fenced rename")
	piLifecycleAfterLegacyCandidateRename = func(int) error { return crash }
	if _, err := PiLegacyRetire(context.Background(), paths, policy, "test", plan.PlanHash); !errors.Is(err, crash) {
		t.Fatalf("simulated crash error=%v", err)
	}
	if next, err := openPiSessionLog(context.Background(), paths, policy); !piErrorIs(err, "lifecycle_log_evidence_unknown") || next != nil {
		t.Fatalf("automatic launch path mutated or bypassed odd legacy fencing: next=%v err=%v", next != nil, err)
	}
	piLifecycleAfterLegacyCandidateRename = nil

	result, err := PiLegacyRetire(context.Background(), paths, policy, "test", plan.PlanHash)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Resumed || result.RetiredCount != 1 || !result.Status.SoakReady {
		t.Fatalf("crash resume result=%+v", result)
	}
	if _, err := os.Lstat(legacy); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("resumed candidate survived: %v", err)
	}
}

// Production call site: PiLegacyRetire -> resumePiLegacyRetirement. The full
// plan hash covers policy provenance, so an odd-generation retry must bind the
// persisted source as well as the numeric policy digest. A different source
// with byte-identical policy values cannot reuse the old confirmation.
func TestPiLegacyRetirementResumeRejectsChangedPolicySource(t *testing.T) {
	paths := newPiLifecycleTestState(t)
	policy := testPiLifecyclePolicy()
	seedClosedPiLifecycleEntry(t, paths, policy)
	legacy := filepath.Join(paths.LogsDir, "legacy-policy-source.jsonl")
	if err := os.WriteFile(legacy, []byte("policy source\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	plan, err := PiLegacyRetirementDryRun(context.Background(), paths, policy, "source-a")
	if err != nil {
		t.Fatal(err)
	}

	originalHook := piLifecycleAfterLegacyCandidateRename
	t.Cleanup(func() { piLifecycleAfterLegacyCandidateRename = originalHook })
	crash := errors.New("simulated crash before rename progress persisted")
	piLifecycleAfterLegacyCandidateRename = func(int) error { return crash }
	if _, err := PiLegacyRetire(context.Background(), paths, policy, "source-a", plan.PlanHash); !errors.Is(err, crash) {
		t.Fatalf("simulated policy-source crash error=%v", err)
	}
	piLifecycleAfterLegacyCandidateRename = nil

	if _, err := PiLegacyRetire(context.Background(), paths, policy, "source-b", plan.PlanHash); !piErrorIs(err, "lifecycle_legacy_confirmation_mismatch") {
		t.Fatalf("changed policy source reused odd-generation authority: %v", err)
	}
	tombstones, err := filepath.Glob(filepath.Join(paths.LogsDir, ".retiring-*.jsonl"))
	if err != nil || len(tombstones) != 1 {
		t.Fatalf("changed policy source did not preserve operation evidence: tombstones=%v err=%v", tombstones, err)
	}
	encoded, err := os.ReadFile(filepath.Join(paths.LifecycleLogsRoot, "legacy-generation.json"))
	var persisted piLifecycleLegacyGeneration
	if err == nil {
		err = json.Unmarshal(encoded, &persisted)
	}
	if err != nil || persisted.State != "odd" || persisted.PolicySource != "source-a" || persisted.PlanHash != plan.PlanHash || persisted.NextCandidate != 0 {
		t.Fatalf("changed policy source altered persisted authority: generation=%s err=%v", encoded, err)
	}

	result, err := PiLegacyRetire(context.Background(), paths, policy, "source-a", plan.PlanHash)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Resumed || result.RetiredCount != 1 || !result.Status.SoakReady {
		t.Fatalf("original policy source could not resume: result=%+v", result)
	}
}

// Production call site: PiLegacyRetire -> resumePiLegacyRetirement. External
// removal before the operation-bound rename leaves neither source nor
// tombstone, but cannot mint retirement authority from that absence.
func TestPiLegacyRetirementRefusesExternalAbsenceWithoutPersistedRenameAuthority(t *testing.T) {
	paths := newPiLifecycleTestState(t)
	policy := testPiLifecyclePolicy()
	seedClosedPiLifecycleEntry(t, paths, policy)
	legacy := filepath.Join(paths.LogsDir, "legacy-external-absence.jsonl")
	if err := os.WriteFile(legacy, []byte("external\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	plan, err := PiLegacyRetirementDryRun(context.Background(), paths, policy, "test")
	if err != nil {
		t.Fatal(err)
	}

	originalHook := piLifecycleBeforeLegacyCandidateUnlink
	t.Cleanup(func() { piLifecycleBeforeLegacyCandidateUnlink = originalHook })
	piLifecycleBeforeLegacyCandidateUnlink = func(_ int, _ string, _ int) error {
		return os.Remove(legacy)
	}
	if _, err := PiLegacyRetire(context.Background(), paths, policy, "test", plan.PlanHash); !piErrorIs(err, "lifecycle_log_evidence_unknown") {
		t.Fatalf("external removal before rename admitted: %v", err)
	}
	piLifecycleBeforeLegacyCandidateUnlink = nil

	if result, err := PiLegacyRetire(context.Background(), paths, policy, "test", plan.PlanHash); !piErrorIs(err, "lifecycle_log_evidence_unknown") || result.RetiredCount != 0 || result.Status.SoakReady {
		t.Fatalf("external absence minted retirement authority: result=%+v err=%v", result, err)
	}
	generation, err := os.ReadFile(filepath.Join(paths.LifecycleLogsRoot, "legacy-generation.json"))
	var persisted piLifecycleLegacyGeneration
	if err == nil {
		err = json.Unmarshal(generation, &persisted)
	}
	if err != nil || persisted.State != "odd" || persisted.CandidateRenamed || persisted.NextCandidate != 0 {
		t.Fatalf("external absence did not preserve odd unknown evidence: generation=%s err=%v", generation, err)
	}
}

// Production call site: PiLegacyRetire -> resumePiLegacyRetirement. A crash
// after unlink is resumable only because the odd generation persisted the
// operation-bound rename before unlink made both paths absent.
func TestPiLegacyRetirementResumesCrashAfterUnlinkWithPersistedRenameAuthority(t *testing.T) {
	paths := newPiLifecycleTestState(t)
	policy := testPiLifecyclePolicy()
	seedClosedPiLifecycleEntry(t, paths, policy)
	legacy := filepath.Join(paths.LogsDir, "legacy-post-unlink-crash.jsonl")
	if err := os.WriteFile(legacy, []byte("crash\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	plan, err := PiLegacyRetirementDryRun(context.Background(), paths, policy, "test")
	if err != nil {
		t.Fatal(err)
	}

	originalHook := piLifecycleAfterLegacyCandidateUnlink
	t.Cleanup(func() { piLifecycleAfterLegacyCandidateUnlink = originalHook })
	crash := errors.New("simulated crash after legacy unlink")
	piLifecycleAfterLegacyCandidateUnlink = func(int) error { return crash }
	if _, err := PiLegacyRetire(context.Background(), paths, policy, "test", plan.PlanHash); !errors.Is(err, crash) {
		t.Fatalf("simulated post-unlink crash error=%v", err)
	}
	piLifecycleAfterLegacyCandidateUnlink = nil

	result, err := PiLegacyRetire(context.Background(), paths, policy, "test", plan.PlanHash)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Resumed || result.RetiredCount != 1 || !result.Status.SoakReady {
		t.Fatalf("persisted post-unlink authority did not resume: result=%+v", result)
	}
}

// Production call site: PiLegacyRetire -> resumePiLegacyRetirement. A retry
// that proves the exact operation tombstone must persist equivalent rename
// authority before unlink, so a second crash cannot strand a legitimate
// retirement as indistinguishable from external dual absence.
func TestPiLegacyRetirementResumesTwoProgressWindowCrashes(t *testing.T) {
	paths := newPiLifecycleTestState(t)
	policy := testPiLifecyclePolicy()
	seedClosedPiLifecycleEntry(t, paths, policy)
	legacy := filepath.Join(paths.LogsDir, "legacy-two-crashes.jsonl")
	if err := os.WriteFile(legacy, []byte("crash twice\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	plan, err := PiLegacyRetirementDryRun(context.Background(), paths, policy, "test")
	if err != nil {
		t.Fatal(err)
	}

	originalRenameHook := piLifecycleAfterLegacyCandidateRename
	originalUnlinkHook := piLifecycleAfterLegacyCandidateUnlink
	t.Cleanup(func() {
		piLifecycleAfterLegacyCandidateRename = originalRenameHook
		piLifecycleAfterLegacyCandidateUnlink = originalUnlinkHook
	})
	firstCrash := errors.New("simulated crash after legacy rename")
	piLifecycleAfterLegacyCandidateRename = func(int) error { return firstCrash }
	if _, err := PiLegacyRetire(context.Background(), paths, policy, "test", plan.PlanHash); !errors.Is(err, firstCrash) {
		t.Fatalf("first progress-window crash error=%v", err)
	}
	piLifecycleAfterLegacyCandidateRename = nil

	secondCrash := errors.New("simulated crash after resumed legacy unlink")
	piLifecycleAfterLegacyCandidateUnlink = func(int) error { return secondCrash }
	if _, err := PiLegacyRetire(context.Background(), paths, policy, "test", plan.PlanHash); !errors.Is(err, secondCrash) {
		t.Fatalf("second progress-window crash error=%v", err)
	}
	piLifecycleAfterLegacyCandidateUnlink = nil

	encoded, err := os.ReadFile(filepath.Join(paths.LifecycleLogsRoot, "legacy-generation.json"))
	var persisted piLifecycleLegacyGeneration
	if err == nil {
		err = json.Unmarshal(encoded, &persisted)
	}
	if err != nil || persisted.State != "odd" || !persisted.CandidateRenamed || persisted.NextCandidate != 0 {
		t.Fatalf("resumed unlink lacked durable rename authority: generation=%s err=%v", encoded, err)
	}

	result, err := PiLegacyRetire(context.Background(), paths, policy, "test", plan.PlanHash)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Resumed || result.RetiredCount != 1 || !result.Status.SoakReady {
		t.Fatalf("two-crash retirement did not resume: result=%+v", result)
	}
}

// Production call sites: CreatePiStateTree, openPiSessionLog and
// PiLifecycleStatus. Automatic setup, launch lifecycle maintenance, and status
// inspection may change managed aggregate evidence but never legacy evidence or
// the explicit legacy generation.
func TestPiAutomaticSetupLaunchAndStatusNeverMutateLegacyEvidence(t *testing.T) {
	paths := newPiLifecycleTestState(t)
	policy := testPiLifecyclePolicy()
	legacy := filepath.Join(paths.LogsDir, "legacy-preserved.jsonl")
	if err := os.WriteFile(legacy, []byte("preserve\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	seedClosedPiLifecycleEntry(t, paths, policy)
	beforeLegacy, err := os.ReadFile(legacy)
	if err != nil {
		t.Fatal(err)
	}
	beforeGeneration, err := os.ReadFile(filepath.Join(paths.LifecycleLogsRoot, "legacy-generation.json"))
	if err != nil {
		t.Fatal(err)
	}

	if err := CreatePiStateTree(paths); err != nil {
		t.Fatal(err)
	}
	log, err := openPiSessionLog(context.Background(), paths, policy)
	if err != nil {
		t.Fatal(err)
	}
	if err := log.close(context.Background()); err != nil {
		t.Fatal(err)
	}
	_, _ = PiLifecycleStatus(context.Background(), paths, policy, "test", "")

	afterLegacy, err := os.ReadFile(legacy)
	if err != nil {
		t.Fatal(err)
	}
	afterGeneration, err := os.ReadFile(filepath.Join(paths.LifecycleLogsRoot, "legacy-generation.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(beforeLegacy, afterLegacy) || !reflect.DeepEqual(beforeGeneration, afterGeneration) {
		t.Fatalf("automatic path mutated legacy evidence: legacy=%q generation_before=%s generation_after=%s", afterLegacy, beforeGeneration, afterGeneration)
	}
}

func seedClosedPiLifecycleEntry(t *testing.T, paths PiStatePaths, policy PiLifecycleLogRetention) {
	t.Helper()
	log, err := openPiSessionLog(context.Background(), paths, policy)
	if err != nil {
		t.Fatal(err)
	}
	if err := log.close(context.Background()); err != nil {
		t.Fatal(err)
	}
}
