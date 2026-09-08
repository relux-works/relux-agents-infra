# TASK-260830-r1uh4v Developer Rework Blocker

## Verdict

Revision-2 rework closes both reviewer bypasses and passes the requested code,
test, build, setup, verification, and installed-parity gates. It cannot publish
an exact-current-main Change Request because protected `main` advanced after
validation, and the advancing commit invalidates a separate acceptance premise.

## Implemented Rework

- `tools/agents-infra/internal/infra/infra_test.go` now compares Claude's
  installed entrypoint with a test-owned expected consumer path instead of the
  production constant that generated it.
- The production `Setup` composition test now pins the complete External-CI
  Markdown section as one independently owned expectation. An additive
  contradictory authorization inside that section is rejected.
- The production-path negative test covers replacement and additive broadened
  trigger shapes after real `Setup` publication to Agents, Claude, and Codex.
- `LOGBOOK.md` records the reviewer regressions, the fixes, and the later
  protected-main conflict.
- The candidate still changes only the four authorized paths. It does not
  modify `task-board.config.json`.

Production call site under test: `infra.Setup(Options{Layout: layout})` in
`TestSetupGlobalPublishesExternalCILocalMirrorPolicyToClaudeAndCodex`.

## Validation Evidence

Every gate command ran directly as a standalone process. Expected-red mutants
retain their real non-zero result.

| Gate | Exit | Evidence |
| --- | ---: | --- |
| Fresh pre-edit fetch | 0 | `git-fetch-origin-main.log` |
| Pre-edit fetched/base/HEAD equality | 0 | `base-equality-pre-edit.log` |
| Initial focused invocation from the repository root | 1 | `focused-policy-rework-01.log`; expected module-root invocation error, then corrected |
| Correct focused rework suite | 0 | `focused-policy-rework-02.log` |
| Additive broadened-trigger live mutant | 1, expected red | `mutant-additive-broadened-01.log` |
| Claude production-entrypoint bypass live mutant | 1, expected red | `mutant-claude-entrypoint-bypass-01.log` |
| Claude workflow-index include bypass live mutant | 1, expected red | `mutant-claude-index-include-bypass-01.log` |
| Codex workflow-index include bypass live mutant | 1, expected red | `mutant-codex-index-include-bypass-01.log` |
| Focused suite after byte-for-byte restoration | 0 | `focused-policy-restored-01.log` |
| Full uncached `go test ./... -count=1` | 0 | `go-test-all-01.log` |
| `go vet ./...` | 0 | `go-vet-all-01.log` |
| `go build ./...` | 0 | `go-build-all-01.log` |
| `AGENTS_INFRA_SKIP_LLDB_MCP=1 ./setup.sh` | 0 | `setup-global-skip-lldb-01.log` |
| `agents-infra verify global` | 0 | `verify-global-01.log` |
| Installed Agents/Claude/Codex parity | 0 | `installed-parity-01.log` |
| Focused final-candidate suite | 0 | `focused-policy-final-01.log` |
| Final formatting check | 0 | `gofmt-check-final.log` |
| Final diff check after blocker logbook entry | 0 | `git-diff-check-blocked.log` |
| Final fresh fetch | 0 | `git-fetch-origin-main-final.log` |
| Final exact-base gate | 1, blocking | `base-equality-final-blocker.log`, `upstream-drift-final.log`, and `worktree-status-final.json` |

The full test/vet/build/install gates ran after the code and test changes. The
only later repository edit was the blocker entry in `LOGBOOK.md`; the final
focused suite and diff/format checks cover the resulting candidate state.

## Stop-The-Line Constraint

- Before editing, `FETCH_HEAD`, `origin/main`, local `main`, workspace `HEAD`,
  Story branch `HEAD`, selected/current/checkpoint base, and CR revision-1 base
  all equalled `fe3818209c9861fcafa1f2e68efe078cc0f96f30`.
- The final fresh fetch advanced `FETCH_HEAD`, `origin/main`, and local `main`
  to `d69a435945758ea1cd5dfa62395ca32498e712c7` while the managed workspace and
  all recorded bases remain `fe3818209c9861fcafa1f2e68efe078cc0f96f30`.
- Upstream `d69a435` changes only `task-board.config.json` and explicitly
  removes Codex `fast_mode` from spawn ceilings.
- This task simultaneously requires exact current main, preservation of the
  landed Codex-only fast-mode configuration, and a four-path scope excluding
  `task-board.config.json`. Those requirements can no longer all be satisfied.
- Manual switch, merge, or rebase of the managed Story branch is prohibited.
  Path-disjoint drift is not substituted for the exact-OID gate.

## Required Decision And Options

1. Treat current trunk `d69a435` as authoritative, revise the fast-mode
   preservation criterion to preserve its intentional absence, and provision a
   fresh managed Story workspace before replaying this four-path rework.
2. Intentionally restore fast mode, expand the authorized scope to include
   `task-board.config.json`, and resolve ownership of the explicit upstream
   removal before provisioning a fresh workspace.

Recommendation: option 1. It preserves the four-path policy scope and does not
silently undo a newer explicit protected-trunk decision.

Exact input required: the orchestrator must select one option, update the task
acceptance/scope accordingly, and reprovision from the freshly fetched selected
base. Existing CR revision 1 remains immutable and changes-requested; no
revision-2 Change Request was published from the stale workspace.
