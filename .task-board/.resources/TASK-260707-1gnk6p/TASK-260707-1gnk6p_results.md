# TASK-260707-1gnk6p implementation evidence

## Scope

- Production call site: root `./setup.sh` -> `scripts/setup.sh:install_lldb_mcp`.
- Story worktree base: `cf21665dde35274cc14e66e26a93574e0c18c15c`.
- Current local trunk checked before implementation: `main@5c9b4e4f7a88e1eb937b80851af522e4fa4b066f` (Story base was 63 commits behind).
- The LLDB install block had the same fail-open and repeat-install behavior on both bases.

## Implementation

- Reuse an executable `lldb-mcp` already present on `PATH`, even when Homebrew is absent.
- Fail non-zero when macOS has neither `lldb-mcp` nor Homebrew; remediation names both alternatives and the exact `brew install llvm` command.
- Treat the Homebrew wrapper as installed only when it is a regular executable with the managed marker, exact expected helper target, executable helper, and executable sibling `lldb`.
- Preserve a valid managed wrapper across repeat setup runs without a second `brew install llvm` or inode change.
- Document command, helper path, wrapper path, failure behavior, and repeat-run behavior in `README.md` and the source-managed `SKILL.md`.

## Negative and production-entry evidence

- Before the fix, `go test -count=1 ./... -run 'TestBootstrapSetup(...)'` exited `1`: real `./setup.sh` completed with `=== Done ===` when Homebrew/helper were absent, failed to recognize an external helper without Homebrew, and replaced the wrapper on run two.
- `TestBootstrapSetupRefusesMissingHomebrewAndLLDBMCP` now requires non-zero exit, both prerequisite names, exact remediation, and absence of `=== Done ===`.
- `TestBootstrapSetupRepairsManagedLLDBMCPWrapperWithWrongTarget` is the narrowing control: a marker-only wrapper targeting a forged helper must be replaced and must trigger one real install path.
- `TestBootstrapSetupInstallsLLDBMCPOnceAcrossTwoRealRuns` runs the real root entry point twice, compares `os.SameFile` inode identity, and requires exactly one logged `brew install llvm`.
- `TestBootstrapSetupAcceptsExistingLLDBMCPWithoutHomebrew` proves the refusal is bounded to missing-both, not all no-Homebrew hosts.

## Validation

| Command | Base | Exit | Result |
| --- | --- | ---: | --- |
| `go test -v -count=1 . -run '^TestBootstrapSetup'` | Story worktree | 0 | Four production-entry cases passed |
| `go test -count=1 ./...` | Story worktree | 0 | Full Go suite passed uncached |
| `go vet ./...` | Story worktree | 0 | Go lint/vet passed |
| `go build -trimpath -o ../../.temp/TASK-260707-1gnk6p/agents-infra .` | Story worktree | 0 | CLI compiled |
| `zsh -n setup.sh scripts/setup.sh` | Story worktree | 0 | Bootstrap shell syntax valid |
| `git diff --check` | Story worktree | 0 | Diff hygiene passed |
| `go test -count=1 ./... -run '^TestBootstrapSetup'` | Clean snapshot of `main@5c9b4e4f` plus scoped production/test delta | 0 | Current-trunk compatibility passed |

The first current-main attempt exited `1` because the validation fixture had not actually applied the production patch; its output showed the unchanged warn-and-return block. After applying the same scoped patch with `patch -p1` and inspecting the copied production call site, the rerun above exited `0`. No live global setup was run from the 63-commit-behind Story branch, avoiding replacement of the operator's installed runtime with stale unrelated code.
