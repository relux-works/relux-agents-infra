package modelharness

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testPinnedCommit = "45a472f2d0cda166b7ffe1a80fe50dd9621f4303"

// writeDistInfo materializes a fake dist-info directory with a direct_url.json
// so VerifyPinnedDistribution can be exercised without a real pip/pipx install.
func writeDistInfo(t *testing.T, sitePackages, distDirName, directURLBody string) {
	t.Helper()
	dir := filepath.Join(sitePackages, distDirName)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "direct_url.json"), []byte(directURLBody), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

// TestVerifyPinnedDistributionAcceptsNonEditablePinnedCommit proves the
// positive control: a non-editable git install whose direct_url.json commit
// matches the configured pin passes.
func TestVerifyPinnedDistributionAcceptsNonEditablePinnedCommit(t *testing.T) {
	sitePackages := t.TempDir()
	writeDistInfo(t, sitePackages, "mlx_lm-0.32.0.dist-info", `{
		"url": "https://github.com/relux-works/mlx-lm.git",
		"vcs_info": {"vcs": "git", "commit_id": "`+testPinnedCommit+`", "requested_revision": "`+testPinnedCommit+`"}
	}`)
	pin := PinnedDistribution{Package: "mlx_lm", SitePackages: sitePackages, Commit: testPinnedCommit}
	if err := VerifyPinnedDistribution(pin); err != nil {
		t.Fatalf("VerifyPinnedDistribution: %v", err)
	}
}

// TestVerifyPinnedDistributionRefusesEditableInstall is the negative control
// named by the task: an editable install (dir_info.editable=true) must fail
// verification even when installed from the same source tree.
func TestVerifyPinnedDistributionRefusesEditableInstall(t *testing.T) {
	sitePackages := t.TempDir()
	writeDistInfo(t, sitePackages, "mlx_lm-0.32.0.dist-info", `{
		"url": "file:///Users/alexis/src/relux-works/mlx-lm",
		"dir_info": {"editable": true}
	}`)
	pin := PinnedDistribution{Package: "mlx_lm", SitePackages: sitePackages, Commit: testPinnedCommit}
	err := VerifyPinnedDistribution(pin)
	if err == nil || !strings.Contains(err.Error(), "editable") {
		t.Fatalf("error = %v, want editable-install refusal", err)
	}
}

func TestVerifyPinnedDistributionRefusesDriftedCommit(t *testing.T) {
	sitePackages := t.TempDir()
	writeDistInfo(t, sitePackages, "mlx_lm-0.32.0.dist-info", `{
		"url": "https://github.com/relux-works/mlx-lm.git",
		"vcs_info": {"vcs": "git", "commit_id": "91506981056172f937e7bdca4ab0d3b7459c7fab", "requested_revision": "91506981056172f937e7bdca4ab0d3b7459c7fab"}
	}`)
	pin := PinnedDistribution{Package: "mlx_lm", SitePackages: sitePackages, Commit: testPinnedCommit}
	err := VerifyPinnedDistribution(pin)
	if err == nil || !strings.Contains(err.Error(), "drifted") {
		t.Fatalf("error = %v, want drifted-commit refusal", err)
	}
}

// TestVerifyPinnedDistributionRefusesCommitSharingOnlyAPrefix proves the
// comparison is against the full 40-character commit, not merely the
// abbreviated 7-character form operators typically write in prose (the task
// itself names the pin as both "45a472f" and its full SHA). A narrowing
// mutant that compared only a short prefix would admit this installed commit.
func TestVerifyPinnedDistributionRefusesCommitSharingOnlyAPrefix(t *testing.T) {
	sitePackages := t.TempDir()
	installedCommit := "45a472f2aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	writeDistInfo(t, sitePackages, "mlx_lm-0.32.0.dist-info", `{
		"url": "https://github.com/relux-works/mlx-lm.git",
		"vcs_info": {"vcs": "git", "commit_id": "`+installedCommit+`", "requested_revision": "`+installedCommit+`"}
	}`)
	pin := PinnedDistribution{Package: "mlx_lm", SitePackages: sitePackages, Commit: testPinnedCommit}
	err := VerifyPinnedDistribution(pin)
	if err == nil || !strings.Contains(err.Error(), "drifted") {
		t.Fatalf("error = %v, want drifted refusal for a commit sharing only a 7-char prefix", err)
	}
}

func TestVerifyPinnedDistributionRefusesMissingDistribution(t *testing.T) {
	pin := PinnedDistribution{Package: "mlx_lm", SitePackages: t.TempDir(), Commit: testPinnedCommit}
	err := VerifyPinnedDistribution(pin)
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("error = %v, want not-found refusal", err)
	}
}

func TestVerifyPinnedDistributionRefusesAmbiguousDistribution(t *testing.T) {
	sitePackages := t.TempDir()
	writeDistInfo(t, sitePackages, "mlx_lm-0.32.0.dist-info", `{"url": "https://github.com/relux-works/mlx-lm.git", "vcs_info": {"vcs": "git", "commit_id": "`+testPinnedCommit+`"}}`)
	writeDistInfo(t, sitePackages, "mlx_lm-0.31.3.dist-info", `{"url": "https://github.com/relux-works/mlx-lm.git", "vcs_info": {"vcs": "git", "commit_id": "`+testPinnedCommit+`"}}`)
	pin := PinnedDistribution{Package: "mlx_lm", SitePackages: sitePackages, Commit: testPinnedCommit}
	err := VerifyPinnedDistribution(pin)
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("error = %v, want ambiguous refusal", err)
	}
}

func TestVerifyPinnedDistributionRefusesNonVCSInstall(t *testing.T) {
	sitePackages := t.TempDir()
	writeDistInfo(t, sitePackages, "mlx_lm-0.32.0.dist-info", `{"url": "https://pypi.org/simple/mlx-lm/"}`)
	pin := PinnedDistribution{Package: "mlx_lm", SitePackages: sitePackages, Commit: testPinnedCommit}
	err := VerifyPinnedDistribution(pin)
	if err == nil || !strings.Contains(err.Error(), "not a git VCS install") {
		t.Fatalf("error = %v, want non-VCS refusal", err)
	}
}

func TestVerifyPinnedDistributionRefusesMissingDirectURLMetadata(t *testing.T) {
	sitePackages := t.TempDir()
	if err := os.MkdirAll(filepath.Join(sitePackages, "mlx_lm-0.32.0.dist-info"), 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	pin := PinnedDistribution{Package: "mlx_lm", SitePackages: sitePackages, Commit: testPinnedCommit}
	err := VerifyPinnedDistribution(pin)
	if err == nil || !strings.Contains(err.Error(), "no direct_url.json") {
		t.Fatalf("error = %v, want missing-metadata refusal", err)
	}
}
