# TASK-260830-r1uh4v Developer Outcome

## Verdict

The independently validated External-CI local-mirror policy was replayed on the
fresh managed Story workspace. The candidate is ready for independent review.

## Exact Base And Scope

- Freshly fetched `FETCH_HEAD`, `origin/main`, local `main`, selected/initial/current
  workspace base, checkpoint, Story branch HEAD, and workspace HEAD all equal
  `fe3818209c9861fcafa1f2e68efe078cc0f96f30`.
- Equality was proven before editing and again immediately before handoff.
- Changed paths are exactly:
  - `.instructions/INSTRUCTIONS_WORKFLOW.md`
  - `LOGBOOK.md`
  - `README.md`
  - `tools/agents-infra/internal/infra/infra_test.go`
- `task-board.config.json` has no diff and its landed Codex-only
  `"fast_mode": true` configuration remains present.

## Behavior And Negative Evidence

Production call site: `infra.Setup`, driven by
`TestSetupGlobalPublishesExternalCILocalMirrorPolicyToClaudeAndCodex` through
`Setup(Options{Layout: layout})`.

The production composition test proves source-to-installed reachability for the
Agents workflow, Claude entrypoint/index/include chain, and rendered Codex
instructions. The narrowed trigger validator rejects repairable or merely
inconvenient CI disruption while preserving the formerly asserted interior
phrase.

| Gate | Exit | Evidence |
| --- | ---: | --- |
| Pre-edit fetched-base/workspace equality | 0 | `base-equality-pre-edit-01.log` |
| Focused production policy tests | 0 | `focused-policy-01.log` |
| Broadened-trigger live mutant | 1, expected red | `mutant-broadened-trigger-01.log` |
| Claude workflow-include bypass live mutant | 1, expected red | `mutant-claude-include-bypass-01.log` |
| Codex workflow-include bypass live mutant | 1, expected red | `mutant-codex-include-bypass-01.log` |
| Focused tests after byte-for-byte restoration | 0 | `focused-policy-restored-01.log` |
| Full uncached `go test ./... -count=1` | 0 | `go-test-all-01.log` |
| `go vet ./...` | 0 | `go-vet-all-01.log` |
| `go build ./...` | 0 | `go-build-all-01.log` |
| `AGENTS_INFRA_SKIP_LLDB_MCP=1 ./setup.sh` | 0 | `setup-global-skip-lldb-01.log` |
| `agents-infra verify global` | 0 | `verify-global-01.log` |
| Installed Agents/Claude/Codex parity | 0 | `installed-parity-01.log` |
| Focused final-candidate tests | 0 | `focused-policy-final-01.log` |
| Final fetched-base, four-path scope, fast-mode, and diff check | 0 | `final-candidate-gate-01.log` |

Every validation command ran directly as a standalone process. Expected-red
mutants retain their real exit 1 results and were restored from task-local byte
copies before pristine reruns.

## Policy Boundary

- Local mirroring is authorized only for verified external hosted-CI
  non-execution that the agent cannot repair.
- Exact clean PR head, affected-job reproduction, toolchain/environment/service/
  target equivalence, and auditable exit evidence remain mandatory.
- Repository failures, real hosted status, independent remote review, merge
  queues, and branch protection remain authoritative.
- No hosted check, review verdict, merge result, or protection outcome was
  forged or synthesized.

## Handoff

The managed producer handoff publishes the immutable exact-base Change Request;
the orchestrator/reviewer owns independent acceptance and later integration.
