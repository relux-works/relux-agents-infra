# BUG-260818-jreo1p — reviewer verdict: CHANGES REQUESTED (-> to-dev)

Reviewer run: RUN-260818-f9bbb1 (not goal-bound). Read-only review; no production
code was left modified. Every mutant below was applied to a backup-verified copy
and restored byte-for-byte (`cmp` clean, sha256 `fe5c597bc8026411fcf141f74069ee3ff008b0a25dde56cf0269707802213b29`).

## What is confirmed correct

Implementation reviewed at `tools/agents-infra/internal/infra/pi_launch_posix.go:352`
(`waitPiRuntimeReady`), sole production call site `pi_launch_posix.go:165`. A repo-wide
grep for readiness polling shows exactly one such loop — there is no second surface
that could carry a divergent rule.

| Check | Result |
| --- | --- |
| `TestPiLaunchReadinessRetriesOnlyServiceUnavailableAtProductionEntry` | PASS (1.67s); both subtests |
| Widening mutant `== http.StatusServiceUnavailable` -> `>= 500` | RED — `bad_gateway` subtest fails: `code="" want="runtime_readiness_invalid"` |
| Delete mutant (503 branch removed entirely) | RED — `service_unavailable_then_ready` fails: `code="runtime_readiness_invalid" want=""` |
| `go test ./... -count=1` (module) | PASS — root 76.5s, attachments 3.6s, infra 112.1s |
| `go vet ./...` | exit 0 |
| `gofmt -l` on changed Go files | clean |
| Installed global binary carries the fix | `~/.local/bin/agents-infra` is byte-identical to a `-trimpath` rebuild of current source with the recorded ldflags — sha256 `6e74c363c2fcea21b56efe72f1f355738f83f32971b589ac7e2e5265ff4d837d` on both |
| Docs agree with code | `README.md:595-596`, `SKILL.md:309`, `LOGBOOK.md` entry 0401 |

The two branches named in the AC are genuinely bound at the production entry, by a
real `RunPi` call driving the real Pi asset — not a helper. The 502 subtest asserts
request count `1` (no retry), `runtime_readiness_invalid`, owned PIDs gone, and lock
released. Read failure, non-list JSON, missing exact model, and redirect all remain
fatal, with `TestPiRuntimeReadinessDoesNotFollowRedirect` covering the redirect case.

## Finding 1 (blocking) — the retry loop's timeout bound is unbound

DoD line 1 and the AC both require 503 to be polled "**until** exact-model readiness
**or timeout**". Nothing binds the timeout side.

Mutant applied — inside the 503 branch:

```go
if resp.StatusCode == http.StatusServiceUnavailable {
        deadline.Reset(timeout)   // <-- injected: retry forever while 503
        continue
}
```

Result: `go test ./internal/infra -count=1` -> **`ok ... 70.703s`**. Entirely green.

Failure scenario this admits: a llama.cpp that binds the exact loopback port and then
serves 503 forever (stuck or failed model load — precisely the class this bug is
about). `RunPi` never returns, holds the profile lock and the owned process group
indefinitely, and the operator never sees `runtime_readiness_timeout`. Before this
change a 503 was fatal, so the loop could not be long-lived; the change is what makes
the deadline load-bearing, and it shipped without a bound.

No test fixture in the package ever sustains 503 — `writePiSequencedReadinessServer`
flips to 200 on request 3, and `writePiReadinessServer` only ever answers 200.

## Finding 2 (blocking) — the in-loop owned-child liveness gate is unbound

DoD line 1 and the AC require the retry to happen "**only while the owned runtime
remains alive**". Nothing binds that either.

Mutant applied — both liveness guards deleted from `waitPiRuntimeReady`:

```go
case <-childWait.done:                                     // deleted
        return piError("runtime_exited_early", ...)
...
if err := child.Signal(syscall.Signal(0)); err != nil {    // deleted
        return piError("runtime_exited_early", err)
}
```

Result: `go build ./...` clean, `go test ./internal/infra -count=1` -> **`ok ... 71.280s`**.
Entirely green.

`TestPiLaunchRefusesForeignReadyListenerForDeadChild` does not cover this: it serves
200, so readiness returns `nil` and the *post*-readiness liveness check at
`pi_launch_posix.go:176-186` produces `runtime_exited_early`. The in-loop gate is
never the thing under test.

Failure scenario this admits: the owned runtime dies during model load. The loop now
spins on connection errors for the full `startup_timeout_seconds` and reports
`runtime_readiness_timeout` instead of `runtime_exited_early` — wrong operator
diagnostic plus a long hang, on the exact path this bug lengthened.

## Finding 3 (minor, non-blocking) — always-nil `read=` in the non-200 message

`pi_launch_posix.go:389` formats `read=%v` with `readErr`, but the `readErr != nil`
case already returned at line 383. Operators always see `read=<nil>` on the rejection
path. Cosmetic; fix while in the file.

## Finding 4 (minor, evidence) — live smoke claim has no artifact

`BUG-260818-jreo1p_results.md` states "Installed Qwen text and tool live smokes
subsequently exited 0" with no attached transcript or log resource. That is an
unreproducible assertion. Attach the smoke output as a task resource, or report the
property as unknown rather than asserting it.

## Required rework (small, additive — production code is correct by inspection)

Extend `TestPiLaunchReadinessRetriesOnlyServiceUnavailableAtProductionEntry` (or add
sibling production-entry cases) with:

1. **Sustained 503 + short `startup_timeout_seconds`** -> `runtime_readiness_timeout`,
   plus `assertRecordedPIDsGone` and `assertPiLockReleased`. Needs a fixture variant
   that never flips to 200.
2. **Owned child exits while the endpoint is answering 503** -> `runtime_exited_early`,
   plus the same cleanup assertions.

Then confirm both mutants above go red, and record which named test fails for each.
Fix finding 3 in the same pass and attach or withdraw the smoke claim in finding 4.

No production-path code change is required for findings 1 and 2 — the implementation
is correct as written. What is missing is the evidence that keeps it correct.
