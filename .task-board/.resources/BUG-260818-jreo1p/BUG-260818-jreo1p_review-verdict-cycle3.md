# BUG-260818-jreo1p — reviewer verdict (cycle 3)

Run: `RUN-260830-12b6af`

Verdict: **accepted**. Reviewer-archetype run; no `commit_ack` supplied.

## Production behavior

- `RunPi` calls `waitPiRuntimeReady` at `tools/agents-infra/internal/infra/pi_launch_posix.go:260`.
- The readiness loop keeps one configured timer, checks owned-child exit both through `childWait.done` and signal 0, retries connection errors and exact HTTP 503, rejects every other non-200 as `runtime_readiness_invalid`, and requires the exact configured model before Pi spawn.
- README and the installed `relux-agents-infra` skill describe the same exact 503-only boundary.
- The implementation commit `97ebda4a` is contained in `main` and the active Story branch. The current shared dirty paths belong to later Story work; this review changed none of them.

## Adversarial production-entry evidence

Mutants ran only in `.temp/review/mutant-RUN-260830-12b6af`, an isolated copy with the official Pi acceptance asset present. Baseline and final restored runs did not skip and passed.

| Mutant | Expected binding | Result |
| --- | --- | --- |
| Widen exact `503` retry to every `>=500` response | 502 must remain fatal before Pi | RED: `bad_gateway` reached Pi and returned nil instead of `runtime_readiness_invalid` |
| Delete the exact-503 retry branch | 503 must poll through readiness/timeout/child exit | RED: ready-after-503, persistent-503 timeout, and child-exit cases all returned `runtime_readiness_invalid` |
| Delete both in-loop child-liveness guards | Retry must stop when the owned runtime dies | RED: `runtime_readiness_timeout` instead of `runtime_exited_early` |
| Reset the startup timer on every 503 | Configured timeout must remain an upper bound | RED: the persistent-503 production test hit its 5s Go-test watchdog |

After every mutant, `pi_launch_posix.go` was restored from the unchanged shared source. Final isolated and shared SHA-256 both equal `ecdba258a39936662ecf4453d5d1a444e696d2ba419f892f2ec93e33967c19af`. No matching fixture process survived.

## Green validation rerun

- Focused production readiness tests: PASS, `4.484s` in the shared checkout.
- Isolated baseline: PASS, `4.786s`; post-mutant restored run: PASS, `4.271s`.
- `go test ./... -count=1 -timeout=8m`: PASS; root `104.205s`, attachments `2.080s`, infra `188.767s`, modelharness `1.722s`.
- `go build ./...`: PASS.
- `go vet ./...`: PASS.
- `gofmt -l .`: no output.
- `agents-infra verify global`: PASS.
- `agents-infra verify local /Users/alexis/src/local-models`: PASS.
- `/Users/alexis/src/local-models` `pi-infra --print-config`: PASS and resolves the reviewed Qwen profile, exact readiness endpoint, direct-child-process-group ownership, and 120s startup timeout.
- Attached controlled Qwen archive was re-parsed without printing raw transcript: final text is exactly `QWEN_TEXT_OK`; one exact `bash` call returns `QWEN_TOOL_VALUE=42` with `isError=false`; final tool response is exactly `QWEN_TOOL_OK:42`; both stderr records name `127.0.0.1:18011`.
- Final cleanup check found no listener on port 18011 and no matching managed llama/Pi process.
- Final shared `git status --short` is byte-for-byte identical to the initial review status; no code, index, or tracked documentation was modified by this run.

## Evidence notes

- The first focused command executed passing tests but used an invalid `tee` path; it is not counted. The identical rerun with the correct path is the recorded gate.
- The first archive assertion assumed a scalar tool result; the archive uses structured `content`. It is not counted. The corrected schema-aware assertion passed.
- Cycle-2 evidence records that a `timeout * 10` magnitude mutant survives the Go test assertions, while an installed 3s production run timed out in 3.17s. This is a useful follow-up for a tighter elapsed-time assertion, but the current implementation uses the configured timer directly and the sliding-timeout mutant that would remove the upper bound is RED.

## Handoff

Acceptance criteria and reviewer negative-evidence gates are satisfied. Park the Bug at `to-review` as the accepted handoff. The commit-owning Orchestrator may perform the enforced final `done` transition with `commit_ack=scope_committed`; this reviewer must not supply that acknowledgement.
