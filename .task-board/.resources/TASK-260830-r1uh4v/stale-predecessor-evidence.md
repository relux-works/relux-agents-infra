# TASK-260830-rddvxd Developer Blocker Outcome

## Verdict

The audited External-CI fallback replay is implemented and validated in the
managed Story workspace, but it is not eligible for a new candidate snapshot or
developer handoff. A final fresh fetch advanced `main`, `origin/main`, and
`FETCH_HEAD` from the workspace base `b78498bf98c05175db10bb341aee621e53de4881`
to `fe3818209c9861fcafa1f2e68efe078cc0f96f30`. The required exact-OID equality
gate therefore exited 1.

Manual switch, merge, or rebase of this managed Story branch is prohibited.
`task-board worktree` exposes no supported refresh command. The orchestrator
must preserve this evidence and reroute the replay through a newly provisioned
managed Story workspace from exact freshly fetched current main.

## Scoped Replay Input

- Changed paths: `.instructions/INSTRUCTIONS_WORKFLOW.md`, `LOGBOOK.md`,
  `README.md`, `tools/agents-infra/internal/infra/infra_test.go`.
- Diff: 4 files, 203 insertions, 5 deletions.
- Non-authorizing replay patch:
  `TASK-260830-rddvxd_stale-base-replay-input.patch`.
- Patch SHA-256:
  `42c0b768ec25ab22275c1b9a5e48f39fa1f56c73f15dae1f691a1861838b9d5d`.
- The three new upstream commits do not modify any of the four scoped paths;
  they modify `task-board.config.json` and board state. Exact OID equality is
  still mandatory and was not weakened to path non-overlap or tree similarity.

## Validation Evidence

Every command below ran directly as a standalone process. Expected-red mutants
and unsuccessful timing attempts retain their real exit 1 results.

| Gate | Exit | Evidence |
| --- | ---: | --- |
| Focused production `Setup` policy tests after pristine restoration | 0 | `focused-policy-restored-02.log` |
| Broadened repairable-or-inconvenient trigger live mutant | 1, expected red | `mutant-broadened-trigger-01.log` |
| Claude workflow-include bypass live mutant | 1, expected red | `mutant-claude-include-bypass-01.log` |
| Codex workflow-include bypass live mutant | 1, expected red | `mutant-codex-include-bypass-01.log` |
| First configured `go test ./... -count=1` rerun | 1 | `go-test-all-configured-01.log`; unrelated readiness timing failure |
| Exact readiness reproduction, five uncached runs | 1 | `readiness-repro-01.log`; 5/5 red under concurrent host Go-test load |
| Exact readiness reproduction after partial load reduction | 1 | `readiness-repro-02.log`; 5/5 red |
| Exact readiness recovery check | 0 | `readiness-repro-03.log`; one uncached run |
| Authoritative configured `go test ./... -count=1` | 0 | `go-test-all-configured-02.log` |
| Serialized uncached `go test -p 1 ./... -count=1` | 0 | `go-test-all-serial-01.log` |
| `go vet ./...` | 0 | `go-vet-all-01.log` |
| `go build ./...` | 0 | `go-build-all-01.log` |
| `AGENTS_INFRA_SKIP_LLDB_MCP=1 ./setup.sh` | 0 | `setup-global-skip-lldb-01.log` |
| `agents-infra verify global` | 0 | `verify-global-01.log` |
| Installed Agents/Claude/Codex parity | 0 | `installed-parity-02.log` |
| `git diff --check` | 0 | `git-diff-check-final.log` |
| Final fetched-base equality gate | 1, blocking | `base-equality-final.log`; fresh refs advanced before board-registry assertions |

The earlier CR revision-1 validation failure is preserved at
`change-request-rev1-validation.log`: it exited 1 on a different unrelated Pi
side-effect timing test, whose exact five-run reproduction later exited 0 in
`flaky-repro-01.log`.

## Policy Boundary

- Local mirroring remains authorized only for verified external hosted-CI
  non-execution that the agent cannot repair.
- The replay requires exact clean PR head, every affected hosted job, matching
  toolchain/environment/services/target or a documented equivalent, and
  auditable SHA/job/platform/tool/command/exit evidence.
- Repository failures, real hosted status, independent remote review, merge
  queues, and branch protection remain authoritative.
- No hosted status, review verdict, merge result, or protection outcome was
  forged or synthesized.

## Required Reroute

1. Orchestrator provisions a fresh managed Story workspace from exact fetched
   current main (`fe3818209c9861fcafa1f2e68efe078cc0f96f30` at observation time,
   or the newer exact fetched head if it advances again).
2. Replay the attached four-path patch semantically, preserving newer trunk
   policy and config wholesale.
3. Repeat focused tests, all three live mutants, full tests, vet, build,
   canonical skip-LLDB setup, global verification, installed parity, and the
   final exact-base equality gate.
4. Publish a fresh immutable CR and obtain independent review of that exact
   candidate and base. Reviewer ownership cannot be substituted by this
   producer outcome.
