# TASK-260828-2jbufw review verdict

- Change Request: `CR-TASK-260828-2jbufw-1`, revision 1
- Base: `804500529d613d4a3cff182376c8f7fdc6c26c1f`
- Candidate tree: `84f403399a7cc93e804d0ec10463dc1986ff424f`
- Verdict: **changes requested**
- Route: `to-dev`

The narrowed scope was honored. This review does not require benchmark-driver or benchmark-gate implementation from this task; those remain with `TASK-260828-3fgca3` after `TASK-260827-2v13w8` lands.

## Findings

### F1 — the harness attests group shutdown after only the direct child exits

`shutdownRuntime` says it waits for the process group to be reaped, but `tools/agents-infra/internal/modelharness/run.go:93-95` returns immediately when `exec.Cmd.Wait` reports the direct child. It does not establish that the process group has no remaining members before logging `process group ... stopped` and returning success.

The review narrowed the shipped fixture to a same-group grandchild that:

- explicitly ignores `SIGTERM` inside the process;
- closes stdin/stdout/stderr, so `exec.Cmd.Wait` cannot observe it through inherited pipes;
- keeps a loopback TCP listener open;
- remains in the runtime child's process group.

The direct shell handles `SIGTERM` and exits. Candidate revision 1 then returns `nil` and logs:

```text
model-harness: received terminated; stopping profile "detached-port-holder" process group 76147
model-harness: profile "detached-port-holder" process group 76147 stopped after terminated (child: <nil>)
```

At that point the grandchild is alive and rebinding its port fails with `address already in use`. The review-only contract test therefore fails:

```text
harness attested a stopped process group while its detached member survived:
grandchild_alive=true port_bind_error=... address already in use
```

This is the **bypass path around the check** negative shape. The existing tests prove a group whose members all accept the forwarded signal, and the N1 mutant proves `-pgid` rather than `pid`; neither proves the claimed group-empty condition before success. The production path is `Run -> run -> runWithSignals -> runOnce -> shutdownRuntime`, with the false success at `run.go:93-95`.

Required rework:

1. Do not return a successful group-shutdown attestation solely because the direct child's `Wait` completed. Establish that the process group is empty; if a member remains through the grace period, escalate the whole group and do not report success until the group is gone or the kill grace fails.
2. Add a production-entry negative test using `model-harness run`, a same-group SIGTERM-ignoring member with closed harness streams, and a held port. It must fail against revision 1 and prove both group disappearance and port release before exit 0.
3. Add the root cause and regression test to `LOGBOOK.md`. The reviewer did not mutate the candidate tree.

Review artifacts: `TASK-260828-2jbufw_review-group-bypass-test.go` and `TASK-260828-2jbufw_review-group-bypass.log`.

### F2 — the new POSIX lifecycle tests do not compile for the shipped Windows surface

`run_process_windows.go` adds and documents Windows behavior, but `run_shutdown_test.go` has no POSIX build constraint and calls `syscall.Kill`, which is unavailable on Windows. Production cross-build succeeds; cross-compiling the package tests fails at lines 261, 262, 271, 331, 332 and 407:

```text
GOOS=windows GOARCH=amd64 go test -c ./internal/modelharness
internal/modelharness/run_shutdown_test.go:261:17: undefined: syscall.Kill
...
internal/modelharness/run_shutdown_test.go:407:18: undefined: syscall.Kill
```

Required rework: make the POSIX-only test body platform-scoped (or split the platform helpers) and keep a Windows-compilable test surface. Re-run the Windows test cross-compile.

Review artifact: `TASK-260828-2jbufw_review-windows-cross-test.log`.

## Accepted evidence and scope checks

### Real llama-server lifecycle

The exact candidate tree was materialized separately and its `model-harness` binary was built. Against real `/opt/homebrew/bin/llama-server` `0.3.0` build 10621 and the recorded 676 MB Qwen2.5 fixture:

| Check | Reviewer result |
| --- | --- |
| startup/readiness | `/v1/models` reached 200 in `1.137s` |
| health | 200, `{"status":"ok"}` |
| configured KV | `/v1/models` reported `n_ctx=8192` |
| process group | runtime PGID differed from harness PGID |
| directed SIGTERM | harness exit 0; direct `llama-server` gone |
| port | free after harness exit |
| captured stdout | 0 bytes |
| lifecycle records | 2 |

This accepts the real-binary startup, readiness, health, direct llama-server stop and port-release evidence. F1 concerns the stronger group-shutdown attestation the implementation and tests explicitly claim.

### B8

The attached raw captures answer B8. At default verbosity stdout is empty; stderr carries `launch_slot_`, `print_timing` and `release` records keyed by task/slot; no HTTP request/status line is emitted. In the mid-body kill, the client received `http=000`, no status was logged, and task 91 has `launch_slot_` without `release`. Therefore the B4 false-200 shape does not reproduce for llama.cpp; the outcome is unknown rather than falsely successful.

At verbosity 5/10 the completion/tool-call output is present in `Parsed message`. The raw verbosity summary labels one nonce occurrence as a `prompt body` hit because the same nonce was also the requested completion; exact-line inspection shows the full user prompt is absent and the nonce occurrence is the completion record. This does not invalidate the report's prompt-absence conclusion, but future probes should use distinct prompt and expected-output nonces.

### G1/G2 and unchanged gate

- The attached `RuntimeBenchmark.swift` SHA-256 is `d8377708ae4e893cb4f65b8aa4c524a9929c72f032992403b0bc808dd2291e18`, matching its recorded source digest.
- Reviewer compilation reproduced G1: `--ubatch-size` and `-ub` both derive `prefill-step=unpinned` and are refused.
- Reviewer compilation reproduced the narrowing cost: removing only `prefill-step=unpinned` also admits unpinned mlx-swift (default 512) and python-mlx-lm (default 2048).
- G2 is sound: the production derivation maps absence of `--max-kv-size` to `kv=unbounded`, while real llama.cpp reported finite `n_ctx=32768` by default and `8192` when pinned. A llama.cpp record can therefore state a false bound and falsely match an MLX record.
- None of the eight changed paths is under the benchmark driver/gate; this delta cannot reintroduce caller-authored benchmark records.
- The `profiles.qwen-local` lines have zero additions/deletions in the exact README diff.

## Validation

Run against a pristine archive of candidate tree `84f40339...` on Darwin arm64:

| Command | Result |
| --- | --- |
| `gofmt -l .` | exit 0, no output |
| `go test ./internal/modelharness -count=1 -v` | pass |
| `go vet ./...` | pass |
| `go build ./...` | pass |
| `go test ./... -count=1` | pass; slowest package 127.684s |
| `GOOS=windows GOARCH=amd64 go build ./...` | pass |
| `GOOS=windows GOARCH=amd64 go test -c ./internal/modelharness` | fail, F2 |
| review narrowing test | fail as intended against revision 1, F1 |

No `accept_cr` mutation was issued because revision 1 does not satisfy the whole-process-group lifecycle contract.
