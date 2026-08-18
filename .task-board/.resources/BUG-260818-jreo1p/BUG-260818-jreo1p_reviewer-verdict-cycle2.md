# BUG-260818-jreo1p — reviewer verdict (cycle 2, RUN-260818-a40b1d)

Verdict: **accepted**. Reviewer-archetype run; no `commit_ack` supplied.

## Isolation

The shared checkout was never mutated. All mutants ran in
`.temp/BUG-260818-jreo1p/review2`, an rsync copy of the dirty
`tools/agents-infra` module placed inside the repo so the module's
repo-root-relative acceptance-asset path resolves through a symlinked
`.temp/TASK-260817-2h8hn4`. `internal/infra/pi_launch_posix.go` in the shared
checkout hashed `fe5c597bc8026411fcf141f74069ee3ff008b0a25dde56cf0269707802213b29`
before and after the review; `pi_test.go` hashed
`f94500ddd04f8275726719a78622c24762e7cbeb6b7ba9891d988bd4b7959d5b`. Nothing was
staged or committed. Caveat for anyone reusing this sandbox: `officialPiAsset`
resolves through the symlink while production canonicalizes it, so path-comparing
tests (`TestPiLaunchForwardsSignalsThenCleansRuntime`) fail there as a sandbox
artifact — only the readiness tests are valid in that copy, and the full suite was
run in the main checkout instead.

## Baseline

`go test ./internal/infra -run TestPiLaunchReadiness -count=1 -v` in the isolated
copy: 3 tests / 6 subtests PASS, `ok 6.274s`, no SKIP (the official Pi asset is
present, so the gates actually ran).

## Mutants

| # | Mutant on `waitPiRuntimeReady` | Result |
| --- | --- | --- |
| 1 | `deadline.Reset(timeout)` on every exact 503 | RED — package hit the 60s `-timeout` bound and panicked; retry became unbounded |
| 2 | delete both in-loop liveness checks (`<-childWait.done` and `child.Signal(0)`) | RED — `…HonorsRuntimeBounds…/refuses_after_owned_runtime_exits`: got `runtime_readiness_timeout`, want `runtime_exited_early` |
| 3 | widen `== http.StatusServiceUnavailable` to `>= 500` | RED — `…RetriesOnlyServiceUnavailable…/bad_gateway`: got `""`, want `runtime_readiness_invalid` |
| 4 | delete the exact-503 retry branch | RED — `…/service_unavailable_then_ready` plus both new bound subtests |
| 5 | `time.NewTimer(timeout * 10)` | **SURVIVES** — all readiness tests PASS; the timeout subtest merely ran 10.13s instead of 1.13s |

Source was restored from a pre-mutation backup after every run and re-hashed to
`fe5c597b…`. No fixture process or listener survived any run, including the
mutant-1 test-timeout panic (the fixture's parent watchdog works).

## Green gates re-run independently (main checkout, read-only)

- `go test ./... -count=1`: exit 0 — root `122.568s`, attachments `6.362s`, infra `170.243s`.
- `go vet ./...`: exit 0. `gofmt -l .`: no output.
- `agents-infra verify global`: exit 0. `agents-infra verify local /Users/alexis/src/local-models`: exit 0.

## Installed-runtime production proof (new evidence)

`~/.local/bin/agents-infra pi --version` run under `env -i` against Python
readiness fixtures, `startup_timeout_seconds = 3`, `pi` resolved to the official
v0.84.2 asset:

| Case | Exit | Message | Elapsed | Readiness requests | Runtime PID |
| --- | ---: | --- | ---: | ---: | --- |
| persistent 503 | 1 | `runtime readiness timed out` | 3.17s | 26 | gone |
| 502 | 1 | `invalid readiness response: status=502 read=<nil>` | 0.26s | 1 | gone |

This binds both AC branches at the installed global runtime, and incidentally
shows the *configured* timeout magnitude is honored in production even though no
test asserts it.

## Provenance

`~/.local/bin/agents-infra` (sha256 `58f178e6139cba18b709edb83b642973a14cd30763b4430a185ddf4f0c0682eb`)
is byte-identical to
`go build -trimpath -ldflags "-X main.Version=v1.6.1-14-gccf0daf -X main.Commit=ccf0daf -X main.BuildDate=2026-08-18T01:32:47Z" .`
over the current source. A plain `-trimpath` rebuild differs by 96 bytes; a
symbol-table diff shows the only delta is `main.Version/Commit/BuildDate`, which
`scripts/setup.sh:176` stamps. Cycle one's plain-`-trimpath` comparison method is
therefore no longer valid and should not be repeated as-is.

`/Users/alexis/src/local-models/.local/bin/agents-infra` is a shell launcher that
runs `go build` from this source repo on every invocation, so the project-local
`pi-infra` used for the Qwen smokes necessarily executed current production
source.

## Live Qwen smokes re-verified from the artifact

Unpacked `BUG-260818-jreo1p_qwen-controlled-smokes.tar.gz` and re-parsed:

- text: final assistant text exactly `QWEN_TEXT_OK`.
- tool: exactly one `toolCall` — `bash` with `{"command":"printf QWEN_TOOL_VALUE=42"}`;
  `tool_execution_end` `isError=false` with result exactly `QWEN_TOOL_VALUE=42`;
  final assistant text exactly `QWEN_TOOL_OK:42`.
- stderr shows the reviewed profile listening on `127.0.0.1:18011` with model and
  mmproj under the `TASK-260817-300nun` task HOME.

Honest scope limit: in that log llama.cpp bound the port only after
`model loaded` (13.97s), so the Qwen smoke did not itself traverse a 503 window.
The 503 semantics are bound by the Go production-entry tests and by the installed
binary run above, not by the live smoke.

## Documentation

`README.md` (managed Pi operator contract) and `SKILL.md` state exactly
"connection failure and HTTP 503 Service Unavailable retry until timeout …
every other non-200 is fatal", matching the implementation and the error table
row for `runtime_readiness_timeout` / `runtime_readiness_invalid`.

## Non-blocking findings

1. **Narrowing survivor (mutant 5).** The suite binds "the loop terminates on
   some deadline", not "on the configured `startup_timeout_seconds`". A
   `timeout * 10` (or a swap to `ShutdownTimeoutSeconds`) ships green. Cheap fix:
   assert elapsed within a window in
   `…HonorsRuntimeBounds…/times_out_while_runtime_remains_alive`. Production
   behavior is correct — verified above at 3.17s for a configured 3s.
2. **Cosmetic.** `internal/infra/pi_launch_posix.go:389` formats `read=%v` with an
   always-nil `readErr` on the non-200 branch; the operator sees
   `status=502 read=<nil>`. Flagged in cycle one, still present.
3. **Redundant guard, not a hole.** Deleting only one of the two in-loop liveness
   checks survives, because the other covers the same class. Deleting both is RED
   (mutant 2), which is the binding that matters.

None of these admit what the gate must reject, so none block acceptance.

## Handoff

Acceptance evidence is recorded. The commit-owning mover commits this scope and
performs the final `done` transition with `commit_ack=scope_committed`.
