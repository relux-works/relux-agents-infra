package infra

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
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
