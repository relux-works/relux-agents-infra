package infra

import (
	"fmt"
	"os"
	"path/filepath"
)

// validateSourceSkillLinks protects syncRepo's verbatim symlink copy. Relative
// links retain their topology in the installed tree; absolute links would keep
// pointing back outside the installed runtime and are therefore always invalid.
func validateSourceSkillLinks(layout Layout) error {
	root := filepath.Join(layout.SourceDir, ".skills")
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("inspect source skill surface %s: %w", root, err)
	}
	failures := inspectSkillLinks(root, layout.SourceDir, true)
	if len(failures) == 0 {
		return nil
	}
	return fmt.Errorf("source skill links are not safe to materialize: %s", failures[0])
}

// managedSkillLinkFailures validates every surface setup owns. All provider
// links must ultimately resolve into AgentsDir; otherwise verify would attest a
// runtime that recursive tools can use to escape or re-enter an ancestor.
func managedSkillLinkFailures(layout Layout) []string {
	hiddenSkills := filepath.Join(layout.AgentsDir, ".skills")
	failures := inspectSkillLinks(hiddenSkills, layout.AgentsDir, false)
	managedNames := map[string]bool(nil)
	if layout.Mode == ModeGlobal {
		managedNames = make(map[string]bool)
		entries, err := os.ReadDir(hiddenSkills)
		if err != nil {
			return append(failures, fmt.Sprintf("cannot identify managed skill names in %s: %v", hiddenSkills, err))
		}
		for _, entry := range entries {
			managedNames[entry.Name()] = true
		}
	}
	for _, root := range []string{
		filepath.Join(layout.AgentsDir, "skills"),
		filepath.Join(layout.ClaudeDir, "skills"),
		filepath.Join(layout.CodexDir, "skills"),
	} {
		failures = append(failures, inspectTopLevelSkillLinks(root, layout.AgentsDir, managedNames, false)...)
	}
	return failures
}

func inspectSkillLinks(root, containmentRoot string, rejectAbsolute bool) []string {
	if info, err := os.Stat(root); err != nil {
		return []string{fmt.Sprintf("cannot inspect managed skill surface %s: %v", root, err)}
	} else if !info.IsDir() {
		return []string{fmt.Sprintf("managed skill surface is not a directory: %s", root)}
	}
	canonicalContainmentRoot, err := filepath.EvalSymlinks(containmentRoot)
	if err != nil {
		return []string{fmt.Sprintf("cannot resolve managed skill containment root %s: %v", containmentRoot, err)}
	}
	return inspectSkillLinkTree(root, containmentRoot, canonicalContainmentRoot, rejectAbsolute)
}

func inspectTopLevelSkillLinks(root, containmentRoot string, managedNames map[string]bool, rejectAbsolute bool) []string {
	entries, err := os.ReadDir(root)
	if err != nil {
		return []string{fmt.Sprintf("cannot inspect managed skill surface %s: %v", root, err)}
	}
	canonicalContainmentRoot, err := filepath.EvalSymlinks(containmentRoot)
	if err != nil {
		return []string{fmt.Sprintf("cannot resolve managed skill containment root %s: %v", containmentRoot, err)}
	}
	var failures []string
	for _, entry := range entries {
		if managedNames != nil && !managedNames[entry.Name()] {
			continue
		}
		path := filepath.Join(root, entry.Name())
		if entry.Type()&os.ModeSymlink != 0 || entry.IsDir() {
			failures = append(failures, inspectSkillLinkTree(path, containmentRoot, canonicalContainmentRoot, rejectAbsolute)...)
		}
	}
	return failures
}

// inspectSkillLinkTree follows contained directory links with an explicit DFS
// state instead of asking a recursive filesystem walker to follow them. That
// proves the composed graph is acyclic without letting a hostile graph trap the
// validator itself. A completed directory may be reached by several links; that
// is a legitimate DAG, while re-entering a visiting directory is a cycle.
func inspectSkillLinkTree(root, containmentRoot, canonicalContainmentRoot string, rejectAbsolute bool) []string {
	inspector := skillLinkGraphInspector{
		containmentRoot:          containmentRoot,
		canonicalContainmentRoot: canonicalContainmentRoot,
		rejectAbsolute:           rejectAbsolute,
		visiting:                 make(map[string]bool),
		done:                     make(map[string]bool),
	}
	inspector.walkPath(root, "")
	return inspector.failures
}

type skillLinkGraphInspector struct {
	containmentRoot          string
	canonicalContainmentRoot string
	rejectAbsolute           bool
	visiting                 map[string]bool
	done                     map[string]bool
	failures                 []string
}

func (inspector *skillLinkGraphInspector) walkPath(path, viaLink string) {
	info, err := os.Lstat(path)
	if err != nil {
		inspector.failures = append(inspector.failures, fmt.Sprintf("cannot inspect managed skill path %s: %v", path, err))
		return
	}
	if info.Mode()&os.ModeSymlink != 0 {
		failures := inspectManagedSkillLink(path, inspector.containmentRoot, inspector.canonicalContainmentRoot, inspector.rejectAbsolute)
		inspector.failures = append(inspector.failures, failures...)
		if len(failures) != 0 {
			return
		}
		resolved, err := filepath.EvalSymlinks(path)
		if err != nil {
			return // inspectManagedSkillLink already recorded the resolution failure.
		}
		resolvedInfo, err := os.Stat(resolved)
		if err != nil {
			inspector.failures = append(inspector.failures, fmt.Sprintf("cannot inspect managed skill link target %s: %v", path, err))
			return
		}
		if resolvedInfo.IsDir() {
			inspector.walkDirectory(resolved, path)
		}
		return
	}
	if info.IsDir() {
		inspector.walkDirectory(path, viaLink)
	}
}

func (inspector *skillLinkGraphInspector) walkDirectory(path, viaLink string) {
	canonical, err := filepath.EvalSymlinks(path)
	if err != nil {
		inspector.failures = append(inspector.failures, fmt.Sprintf("cannot resolve managed skill directory %s: %v", path, err))
		return
	}
	canonical, err = filepath.Abs(canonical)
	if err != nil {
		inspector.failures = append(inspector.failures, fmt.Sprintf("cannot resolve managed skill directory %s: %v", path, err))
		return
	}
	if inspector.visiting[canonical] {
		edge := viaLink
		if edge == "" {
			edge = path
		}
		inspector.failures = append(inspector.failures, fmt.Sprintf("managed skill graph contains a transitive symlink cycle: %s re-enters %s", edge, canonical))
		return
	}
	if inspector.done[canonical] {
		return
	}
	inspector.visiting[canonical] = true
	entries, err := os.ReadDir(canonical)
	if err != nil {
		inspector.failures = append(inspector.failures, fmt.Sprintf("cannot inspect managed skill directory %s: %v", canonical, err))
		delete(inspector.visiting, canonical)
		return
	}
	for _, entry := range entries {
		path := filepath.Join(canonical, entry.Name())
		if entry.Type()&os.ModeSymlink != 0 || entry.IsDir() {
			inspector.walkPath(path, "")
		}
	}
	delete(inspector.visiting, canonical)
	inspector.done[canonical] = true
}

func inspectManagedSkillLink(path, containmentRoot, canonicalContainmentRoot string, rejectAbsolute bool) []string {
	var failures []string
	rawTarget, err := os.Readlink(path)
	if err != nil {
		return []string{fmt.Sprintf("cannot read managed skill link %s: %v", path, err)}
	}
	if rejectAbsolute && filepath.IsAbs(rawTarget) {
		return []string{fmt.Sprintf("managed skill link is absolute and would escape the installed runtime: %s -> %s", path, rawTarget)}
	}
	lexicalTarget := rawTarget
	if !filepath.IsAbs(lexicalTarget) {
		lexicalTarget = filepath.Join(filepath.Dir(path), lexicalTarget)
	}
	lexicalTarget, err = filepath.Abs(lexicalTarget)
	if err != nil {
		return []string{fmt.Sprintf("cannot resolve managed skill link %s: %v", path, err)}
	}
	// macOS commonly exposes the same temporary tree through both /var and
	// /private/var. Graph traversal uses canonical directory names, while the
	// setup layout retains the caller spelling, so accept containment through
	// either spelling before the resolved-target check below proves identity.
	if !dirContains(containmentRoot, lexicalTarget) && !dirContains(canonicalContainmentRoot, lexicalTarget) {
		return []string{fmt.Sprintf("managed skill link escapes runtime containment: %s -> %s", path, rawTarget)}
	}
	pathAbs, err := filepath.Abs(path)
	if err != nil {
		return []string{fmt.Sprintf("cannot resolve managed skill link path %s: %v", path, err)}
	}
	if dirContains(lexicalTarget, pathAbs) {
		return []string{fmt.Sprintf("managed skill link points to itself or an ancestor: %s -> %s", path, rawTarget)}
	}
	resolvedTarget, err := filepath.EvalSymlinks(path)
	if err != nil {
		return []string{fmt.Sprintf("managed skill link is dangling or cyclic: %s -> %s: %v", path, rawTarget, err)}
	}
	if !dirContains(canonicalContainmentRoot, resolvedTarget) {
		failures = append(failures, fmt.Sprintf("managed skill link resolves outside runtime containment: %s -> %s", path, resolvedTarget))
	}
	return failures
}
