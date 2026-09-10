# BUG-260830-1rths2: fix timing-sensitive readiness test flakes

## Root causes (two distinct races, as described in the bug)

1. `times_out_while_runtime_remains_alive` read a `readiness-count` side-effect
   file written by the fake runtime's HTTP handler. The production readiness
   poller can (and under the fixed timeout usually does) fire the deadline
   before the file exists or before its first write lands, causing
   `os.ReadFile` "no such file" or `strconv.Atoi("")` on an empty read.
2. `refuses_after_owned_runtime_exits` used a runtime that answered 503 for a
   short window and then exited via a timer, racing two independently-timed
   terminal conditions (deadline vs. delayed self-exit) against each other.

## Fix

- Removed the `readiness-count` file and its read entirely. The
  `times_out_while_runtime_remains_alive` case now proves the deadline was
  honored (not returned early) with a wall-clock lower bound: elapsed time
  must be >= the configured `startupTimeoutSeconds`. This is a lower bound,
  not a race \u2014 a real timeout cannot fire before its duration has elapsed,
  regardless of scheduler load.
- The `refuses_after_owned_runtime_exits` fake runtime now exits immediately
  on start (before it could ever answer an HTTP poll) instead of after a
  fixed delay while serving 503s. This removes the race between two timed
  conditions: OS-level child-exit detection is inherently faster than the
  readiness deadline, so `runtime_exited_early` is deterministic.
- Did not widen `startupTimeoutSeconds` (kept at 1s) to "fix" flakes; the AC
  explicitly forbids that shortcut.

## Additional race found and fixed during load verification (not in the original report)

While stress-testing the fix at higher concurrency than the original repro,
`times_out_while_runtime_remains_alive` intermittently failed at
`assertRecordedPIDsGone` with "no such file or directory" for `runtime.pid`
(~1 failure in 45 runs under 3 concurrent processes x 15 iterations). Cause:
the fake runtime wrote its pidfile only after `import http.server, os, sys,
threading, time` \u2014 under heavy host load, that stdlib import (which pulls in
`socketserver`, `email`, `mimetypes`, `html`, etc.) could occasionally take
long enough that the production 1s readiness deadline fired and the process
was reaped before the pidfile write ever executed. This is the same class of
bug the AC calls out: "must not read a side-effect file that the production
path may not have written yet."

Fix: reordered the fake runtime script so the pidfile write happens
immediately after `import os, sys` (cheap, near-free imports), before the
expensive `http.server`/`threading` imports. This collapses the exposed race
window down to bare interpreter startup, independent of unrelated import
cost. Scoped to this one test's fixture script only \u2014 did not touch the
shared `assertRecordedPIDsGone` helper (used by 9 other tests) or production
code, since a broader rewrite is out of proportion for this ticket. Noted:
the same "side-effect file read races production path" shape also appears in
an unrelated, pre-existing test (`TestPinnedPiNoModelDirectRPCBashBypassesToolCallHookWhileStandaloneExcludesRPC`,
which failed the same way in a single serial full-suite run, no concurrency
needed) \u2014 that is a separate pre-existing defect, out of scope here.

## Evidence

- `go build ./...`, `go vet ./internal/infra/...`, `gofmt -l` \u2014 all clean.
- Isolated run: `go test -run TestPiLaunchReadinessServiceUnavailableStillHonorsRuntimeBoundsAtProductionEntry -count=1` \u2014 PASS.
- All `TestPiLaunch*` tests (28 tests) \u2014 PASS, single run.
- Concurrency stress (matching and exceeding "3 concurrent runs" from the
  original report): 3 concurrent processes x 15 iterations = 45 runs, then a
  second round of 5 concurrent processes x 20 iterations = 100 runs. Total
  145 top-level runs (290 subtests), 0 flakes after the pidfile-write-order
  fix. (Before that fix: 1/45 failures reproduced, matching the pidfile race
  described above.)

## Mutant evidence

| Mutant | Narrows | Failing test | Bound stated by survival |
|---|---|---|---|
| Test-side: weaken `elapsed < startupTimeoutSeconds*time.Second` to `elapsed < 0` | The lower-bound assertion itself | none (mutant survives \u2014 see note) | Confirms the check is the only guard; see production-side mutant below for the real proof |
| Production-side: `pi_launch_posix.go` `waitPiRuntimeReady` deadline timer halved (`timeout/2`) | The actual readiness deadline duration | `TestPiLaunchReadinessServiceUnavailableStillHonorsRuntimeBoundsAtProductionEntry/times_out_while_runtime_remains_alive` (pi_test.go:1740) | None \u2014 mutant is caught: "runtime_readiness_timeout fired before the configured bound elapsed: elapsed=874.814208ms bound=1s" |

The test-side mutant (weakening the assertion to `elapsed < 0`, an
always-false condition) does not itself fail under correct production
behavior \u2014 that's expected, since it doesn't inject a wrong input, it only
disables the check. The production-side mutant is the real narrowing proof:
it injects exactly the class of bug the assertion exists to catch (readiness
timeout firing early), and the named subtest fails with a message pinpointing
the violation. Both mutations were reverted; `git diff` confirms
`pi_launch_posix.go` is unchanged in the final state.
