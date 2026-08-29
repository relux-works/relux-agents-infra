# TASK-260830-1xscas Developer Results

## Outcome

The validated revision-2 policy replay was applied byte-for-byte on the freshly
fetched fast-off trunk. Production setup composition now pins the complete
External-CI policy section and a test-owned Claude entrypoint expectation.

Production call site under test:
`infra.Setup(Options{Layout: layout})` in
`TestSetupGlobalPublishesExternalCILocalMirrorPolicyToClaudeAndCodex`.

## Base, Replay, Scope, And Configuration

- Pre-edit and final fresh fetches exited 0.
- Pre-edit and final `HEAD`, Story branch, local `main`, `origin/main`,
  `FETCH_HEAD`, selected/current/checkpoint base all equal
  `d69a435945758ea1cd5dfa62395ca32498e712c7`.
- Materialized replay input SHA-256 equals the declared
  `388e2cd095f69a613733ab03c91c03a65ed1090b0ae82f3499cf48b0742db3e9`.
- Final `git diff --binary` has that same SHA-256: the candidate is the exact
  validated replay.
- Exactly four paths change:
  - `.instructions/INSTRUCTIONS_WORKFLOW.md` — installed workflow policy.
  - `README.md` — operator-facing policy documentation.
  - `LOGBOOK.md` — regression, fix, and predecessor-base history.
  - `tools/agents-infra/internal/infra/infra_test.go` — production composition
    and negative tests.
- `task-board.config.json` is outside the delta and contains no `fast_mode`
  key. Spawn preflight reports `fast_mode=false`, source `default`.

## Validation

Every gate command was executed directly as a standalone foreground process.
Expected-red entries below report the real `go test` exit code.

| Gate | Exit | Evidence |
| --- | ---: | --- |
| Tool readiness | 0 | `tool-readiness-01.log` |
| Initial fresh `git fetch origin main` | 0 | `git-fetch-origin-main-01.log` |
| Pre-edit exact-base gate | 0 | `base-equality-pre-edit-01.log` |
| Replay digest | 0 | `replay-digest-01.log` |
| `git apply --check` | 0 | `git-apply-check-01.log` |
| Invalid preflight task class (`implementation`) | 1 | Expected CLI refusal; corrected to task class `code` |
| Correct fast-off spawn preflight | 0 | `spawn-preflight-fast-off-02.json` |
| First focused invocation | 1 | Shell log-path error before `go test`; corrected immediately |
| Pristine focused suite | 0 | `focused-policy-pristine-02.log` |
| Additive broadened-trigger live mutant | 1, expected red | `mutant-additive-trigger-01.log` |
| Claude production-entrypoint bypass live mutant | 1, expected red | `mutant-claude-entrypoint-01.log` |
| Claude workflow-index include bypass live mutant | 1, expected red | `mutant-claude-index-include-01.log` |
| Codex workflow-index include bypass live mutant | 1, expected red | `mutant-codex-index-include-01.log` |
| Byte-for-byte mutant restoration | 0 | Direct `cmp` of all four mutant targets |
| Focused suite after restoration | 0 | `focused-policy-restored-01.log` |
| Full uncached `go test ./... -count=1` | 0 | `go-test-all-01.log` |
| `go vet ./...` | 0 | `go-vet-all-01.log` |
| `go build ./...` | 0 | `go-build-all-01.log` |
| `gofmt` diff check | 0 | `gofmt-check-01.log` |
| `git diff --check` | 0 | `git-diff-check-01.log` |
| `AGENTS_INFRA_SKIP_LLDB_MCP=1 ./setup.sh` | 0 | `setup-global-skip-lldb-01.log` |
| `agents-infra verify global` | 0 | `verify-global-01.log` |
| Installed Agents/Claude/Codex parity | 0 | `installed-parity-01.log` |
| Final fresh `git fetch origin main` | 0 | `git-fetch-origin-main-final-01.log` |
| Final exact-base gate | 0 | `base-equality-final-01.log` |
| Final four-path/digest/config gate | 0 | `final-scope-and-digest-01.log` |

The four live mutants were applied to real versioned production inputs or the
production Claude entrypoint constant, then driven through `infra.Setup`. Each
mutant was restored before the next gate; source/index/entrypoint restoration
was verified with byte comparisons.

## Handoff Boundary

This developer run publishes the immutable Change Request through
`task-board handoff`. Independent review, canonical PR publication, hosted
checks, and exact-head landing are orchestrator/reviewer lifecycle steps and
are not claimed as already performed here.
