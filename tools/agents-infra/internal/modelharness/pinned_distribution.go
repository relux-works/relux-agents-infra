package modelharness

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// directURLMetadata is the subset of PEP 610 direct_url.json this package
// reads. pip and pipx write this file into the installed distribution's
// dist-info directory for any non-PyPI-index install (VCS URL or local
// directory), recording exactly what was requested and resolved.
type directURLMetadata struct {
	URL     string `json:"url"`
	VCSInfo *struct {
		VCS               string `json:"vcs"`
		CommitID          string `json:"commit_id"`
		RequestedRevision string `json:"requested_revision"`
	} `json:"vcs_info"`
	DirInfo *struct {
		Editable bool `json:"editable"`
	} `json:"dir_info"`
}

// VerifyPinnedDistribution refuses an editable install, a non-VCS install, and
// a commit that does not exactly match pin.Commit. It is the negative gate
// behind `model-harness doctor`: a profile that declares pinned_distribution
// only passes doctor when the installed backend is a non-editable git checkout
// pinned to the configured commit.
func VerifyPinnedDistribution(pin PinnedDistribution) error {
	matches, err := filepath.Glob(filepath.Join(pin.SitePackages, pin.Package+"-*.dist-info"))
	if err != nil {
		return fmt.Errorf("scan %s for pinned distribution %s: %w", pin.SitePackages, pin.Package, err)
	}
	if len(matches) == 0 {
		return fmt.Errorf("pinned distribution %s not found under %s", pin.Package, pin.SitePackages)
	}
	if len(matches) > 1 {
		return fmt.Errorf("pinned distribution %s is ambiguous under %s (%d dist-info directories)", pin.Package, pin.SitePackages, len(matches))
	}
	directURLPath := filepath.Join(matches[0], "direct_url.json")
	raw, err := os.ReadFile(directURLPath)
	if err != nil {
		return fmt.Errorf("pinned distribution %s has no direct_url.json (not an explicit VCS install): %w", pin.Package, err)
	}
	var meta directURLMetadata
	if err := json.Unmarshal(raw, &meta); err != nil {
		return fmt.Errorf("decode %s: %w", directURLPath, err)
	}
	if meta.DirInfo != nil && meta.DirInfo.Editable {
		return fmt.Errorf("pinned distribution %s is an editable install; refuse editable installs for a pinned fork", pin.Package)
	}
	if meta.VCSInfo == nil || meta.VCSInfo.VCS != "git" {
		return fmt.Errorf("pinned distribution %s is not a git VCS install; cannot verify its commit pin", pin.Package)
	}
	if meta.VCSInfo.CommitID != pin.Commit {
		return fmt.Errorf("pinned distribution %s is drifted: installed commit %s does not match configured pin %s", pin.Package, meta.VCSInfo.CommitID, pin.Commit)
	}
	return nil
}
