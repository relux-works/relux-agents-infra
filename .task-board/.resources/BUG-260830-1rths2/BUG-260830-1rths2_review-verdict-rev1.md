# Review Verdict — BUG-260830-1rths2 rev1 — ACCEPTED

## Scope
`LOGBOOK.md`, `tools/agents-infra/internal/infra/pi_test.go`. No production code touched.

## What changed
1. Removed the `countFile` side-effect-file read (raced the production path — file could not exist yet under load). Replaced, for the `runtime_readiness_timeout` case only, with a wall-clock lower-bound assertion (`elapsed >= startupTimeoutSeconds`), which is monotonic under load in the correct direction: a real Go timer cannot fire early, so this can never flake from scheduler jitter, only from a genuine early-return bug.
2. Removed the racing-terminal-conditions design in `refuses_after_owned_runtime_exits`: the fixture no longer starts an HTTP server and waits 20ms via `threading.Timer` before exiting (racing a completed poll round-trip). It now exits immediately after writing the pidfile, before importing `http.server`/`threading` at all, so `runtime_exited_early` no longer competes with a poll cycle.
3. Reordered the persistent-unavailable fixture to write the pidfile before importing `http.server`/`threading` — a third, previously undocumented race the author found via their own 45-run stress pass (heavy stdlib import occasionally outran the 1s production deadline, so `assertRecordedPIDsGone` read a pidfile that didn't exist yet).
4. Logbook entry recorded under 2026-09-11 documenting root cause, fix, evidence, and an out-of-scope anomaly (a same-shape race in an unrelated test) flagged for its own ticket rather than silently fixed here.

## Independent verification performed by this review (not reused from the producer)
- `go build ./...`, `go vet ./...`, `gofmt -l` on both changed Go-adjacent files: clean.
- Full package suite `go test ./...`: PASS in 353.6s (exit 0).
- Reproduced the original repro load myself: 3 concurrent `go test -run TestPiLaunchReadinessServiceUnavailableStillHonorsRuntimeBoundsAtProductionEntry -count=15 ./...` processes (45 total runs) — **0 failures**, matching the concurrency (3x) that originally reproduced the flake.
- Attacked the new elapsed-time assertion with a narrowing mutant on the production side: halved the deadline timer in `waitPiRuntimeReady` (`pi_launch_posix.go:560`, `timeout` → `timeout / 2`). Re-ran the target test: `times_out_while_runtime_remains_alive` FAILED as expected (`elapsed=787ms bound=1s`), `refuses_after_owned_runtime_exits` still passed (unaffected, as expected — it doesn't depend on the timeout deadline). Reverted the mutant; confirmed `git diff --stat` on the production file is empty and the build is clean again.
- Confirmed `assertRecordedPIDsGone` (shared helper, 9 call sites) was correctly left untouched — the fix scoped itself to the one fixture rather than widening a shared helper's tolerance.
- Confirmed the logbook-flagged anomaly test (`TestPinnedPiNoModelDirectRPCBashBypassesToolCallHookWhileStandaloneExcludesRPC`) exists in a separate file (`pi_standalone_real_pi_test.go`) and was correctly left out of this CR's scope.

## AC check
- "must not read a side-effect file production may not have written yet" — countFile read removed; pidfile read is now provably safe (write happens before the only import that could plausibly exceed the 1s deadline). ✓
- "must not depend on which of two racing terminal conditions fires first" — exit-early fixture no longer starts an HTTP server or waits on a timer; it wins the race unconditionally by construction. ✓
- "not fixed by widening a timeout" — `startupTimeoutSeconds` stays at 1; no timeout value changed anywhere in this diff. ✓
- "demonstrated under at least the concurrency that reproduced the failure, repeatedly, no flake" — reproduced independently at 3x/15 (45 runs), 0 flakes, on top of the producer's own 145-run evidence. ✓

## Verdict
ACCEPTED. Fix addresses the root cause (test asserts on deterministic conditions instead of a side-effect file or race outcome), correctly scoped to the test fixture only, evidenced by both reused and independently-reproduced concurrent runs, and the new assertion is proven to have teeth via a narrowing production-side mutant.
