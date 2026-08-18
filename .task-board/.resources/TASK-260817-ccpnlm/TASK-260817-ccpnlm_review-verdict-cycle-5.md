# TASK-260817-ccpnlm reviewer verdict — cycle 5

Verdict branch: changes_requested -> to-dev.

## Accepted rework

The cycle-5 managed-session fix closes the prior export bypass. Production runPi rejects --export before managed cache/runtime/Pi side effects; the sentinel/global-session regression passed 5/5 under the race detector. Pinned args.ts audit confirms the remaining session selectors are either refused when path-shaped or remain rooted in PI_CODING_AGENT_SESSION_DIR.

## Finding

1. P1 — Required signal lifecycle evidence is flaky under suite load.

The first exact reviewer command:

go test -race ./internal/infra -run Test.*Pi -count=1

failed in TestPiLaunchForwardsSignalsThenCleansRuntime/terminated after 37.581s. RunPi returned signal: killed, meaning the Pi child did not complete the graceful SIGTERM path before waitPiGroup escalated to SIGKILL. This contradicts the task-scoped claim that the required focused race suite is green.

Reproduction rate observed during review:
- first focused suite: fail;
- isolated signal test: pass 3/3, then pass 10/10;
- second focused suite: pass;
- observed total for the signal test: 1 failure in 12 runs.

This is the standard capability claim that does not reproduce shape. A later green rerun does not erase the exact required command failure.

Required rework:
- make the production signal/shutdown path and its test deterministic under focused-suite load;
- retain SIGINT and SIGTERM forwarding, graceful exit, group cleanup, lock release, and timeout escalation evidence;
- rerun the exact race-focused command enough times to establish stable behavior;
- preserve the accepted --export/session isolation regressions.

## Other validation

- go test -race ./internal/infra -run Test.*Pi -count=1: first run failed; second run passed.
- go test -race . -run TestRunPi -count=1: passed.
- go test -race . -run TestRunPiRejectsManagedSessionPathOverridesBeforeSideEffects -count=5: passed.
- go test ./... -count=1: passed.
- go vet ./...: passed.
- go build ./...: passed.
- gofmt -l .: no output.
- git diff --check: passed.
- task-board validate: passed.

No code was modified by the reviewer.