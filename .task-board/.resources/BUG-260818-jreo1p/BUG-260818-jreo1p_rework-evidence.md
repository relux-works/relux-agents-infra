# BUG-260818-jreo1p — termination-bound rework evidence

## Rework

- Added `TestPiLaunchReadinessServiceUnavailableStillHonorsRuntimeBoundsAtProductionEntry` at the production `RunPi` entry.
- Persistent exact HTTP 503 must end with `runtime_readiness_timeout` at the configured deadline.
- Exact HTTP 503 followed by owned-child exit must end with `runtime_exited_early`.
- Both cases prove at least one 503 poll, owned PID reap, and profile-lock release.
- The Python fixture watches its test parent and self-terminates if a deliberate unbounded mutant kills the Go test process.

## Negative evidence

Mutants ran in `.temp/BUG-260818-jreo1p/mutant-repo`, an rsync snapshot of the dirty source with the official standalone Pi acceptance asset linked read-only.

- `deadline.Reset(timeout)` on every exact 503: named timeout subtest failed by Go test timeout; exit 1.
- Deleted both in-loop liveness checks (`childWait.done` and `child.Signal(0)`): named child-exit subtest returned `runtime_readiness_timeout` instead of `runtime_exited_early`; exit 1.
- After restoration, isolated production test exited 0; `cmp` against the snapshot baseline and shared production source exited 0; both orphan-process checks exited 0.
- An earlier isolated attempt exited 0 only because the official Pi asset was absent and the test reported `SKIP`; it is not counted as a green gate.
- The first asset copy attempt was killed with exit 137 after copying 2.3 MiB of an 81 MiB tree. The isolated snapshot instead used a read-only link to the unchanged acceptance asset.

The same mutants briefly ran in the shared checkout before the concurrency nudge was observed. Both were restored byte-for-byte and repeated in isolation. The later bootstrap rebuilt and installed only the restored production source.

## Green validation

- Focused production readiness tests: exit 0 (`3.921s`).
- Full `go test ./... -count=1`: exit 0; root `114.453s`, attachments `6.248s`, infra `147.499s`.
- `go build ./...`: exit 0.
- `go vet ./...`: exit 0.
- `gofmt -d tools/agents-infra/internal/infra/pi_test.go`: exit 0, no output.
- `git diff --check`: exit 0.
- `./setup.sh`: exit 0.
- `agents-infra verify global`: exit 0.
- `agents-infra setup local /Users/alexis/src/local-models`: exit 0.
- `agents-infra verify local /Users/alexis/src/local-models`: exit 0.
- Installed project `pi-infra --print-config`: exit 0; exact Qwen profile, reviewed llama.cpp executable, port 18011, and 120s startup timeout resolved.

One early combined formatter/test invocation exited 2 because a repo-root file path was used from the Go module directory; neither formatter nor tests ran in that invocation. They were rerun as separate commands with the results above.

## Controlled Qwen live re-smokes

The accepted `TASK-260817-300nun` task HOME contains the fully cached reviewed Qwen target and mmproj. Both smokes were rerun from `/Users/alexis/src/local-models` through the project-local `pi-infra` under the same `env -i` boundary: task-scoped `HOME`, explicit `PATH`, `TMPDIR=/tmp`, and `/bin/zsh -f`.

- Text smoke: exit 0. JSON assertion exit 0 proves final assistant text exactly `QWEN_TEXT_OK`.
- Tool smoke: exit 0. Combined JSON assertion exit 0 proves exactly one `bash` call with `printf QWEN_TOOL_VALUE=42`, exact successful tool result `QWEN_TOOL_VALUE=42`, `isError=false`, and final assistant text exactly `QWEN_TOOL_OK:42`.
- Post-smoke cleanup: exit 0; port 18011 had no listener and no matching managed llama or `pi-infra` process remained.
- Raw outputs: `BUG-260818-jreo1p_qwen-controlled-smokes.tar.gz`.

Before the task HOME was identified, three installed project-launcher text-smoke attempts used the ordinary HOME and each exited 1 with `runtime readiness timed out` at the configured 120s. After every failed attempt:

- port 18011 had no listener;
- no matching managed llama process remained;
- `ollama ps` was empty;
- system memory reported 77% free;
- setup and installed verification remained green.

The ordinary-HOME failures are retained as honest bounded-timeout/cleanup evidence, not capability evidence. The controlled-cache successes reproduce the accepted deployment and close the live gate without changing runtime policy or profile timeout.

No files were staged or committed.
