# BUG-260818-76hkcb reviewer verdict — accepted

Reviewer run: RUN-260817-7f9739 (claude-opus-5). Read-only review; no product code modified.
All mutation work was done in a disposable copy at `.temp/BUG-260818-76hkcb/review-copy`
(rsync of the dirty working tree, `.temp/TASK-260817-2h8hn4` Pi asset symlinked in), which was
removed after the review. The main checkout was byte-verified unchanged afterwards.

## Verdict

Accepted. The gate was attacked from four independent directions and held; the premise the
operator docs assert was independently established against the real llama.cpp artifact.

## Gate under review

`tools/agents-infra/internal/infra/pi_catalog.go:295` — exact, case-sensitive
`GGML_BACKEND_PATH` added to the existing `HF_ENDPOINT` / `MODEL_ENDPOINT` exact-name member.

Production call site: `tools/agents-infra/internal/infra/pi_launch_posix.go:94`, inside `RunPi`,
after `BuildManagedPiArguments` and before `VerifyPiExecutionIdentity`,
`inspectRuntimeExecutable`, `ResolvePiStatePaths`, `CreatePiStateTree`, `AcquirePiProfileLock`,
`preflightPiListener`, and both `exec` spawns. Reached from `main.go:423`
(`agents-infra pi` -> `infra.RunPi`), which the bootstrap-global `pi-infra` alias and the
setup-generated project-local `pi-infra` wrapper both exec.

## Attacks executed by the reviewer (not reruns of developer logs)

| # | Mutant applied to the disposable copy | Production suite | Installed-launcher suite | Result |
| --- | --- | ---: | ---: | --- |
| 1 | Narrow: `GGML_BACKEND_PATH` -> `GGML_BACKEND_PATH_ZZZ` (gate kept, class narrowed) | FAIL | FAIL | Class bound is real |
| 2 | Broaden: add `"GGML_"` to the loader prefix list | FAIL | FAIL | Upper bound is real |
| 3 | Reorder: move the gate from pre-state to after `CreatePiStateTree` | FAIL | n/a | "before managed state" is asserted |
| 4 | Leak: append the denied value to the refusal message | FAIL | FAIL | Value non-disclosure is asserted |
| — | Pristine baseline in the same copy | PASS | PASS | Mutants, not the harness, caused the red |

Mutant 1 red detail: production reached `runtime exited before readiness` — the runtime actually
spawned with the variable admitted, so the gate is what stops the spawn, not an incidental failure.
Mutant 3 red detail: `environment refusal created managed state: [d agents-infra/]` for every
member of the environment-refusal table, not just the new one.
Mutant 2 red detail: `GGML_METAL_PATH` clean control refused on both the in-process lifecycle test
and both installed launcher surfaces — the "no speculative `GGML_*` prefix" boundary is pinned in
both directions, so this is not a delete-only mutant result.

## Premise verification (docs claim treated as a claim, not as given)

The operator contract asserts "llama.cpp build 10470 passes its inherited value to `dlopen()`
during backend discovery" and that other `GGML_*` names have no established effect. Verified
directly against the installed artifact rather than accepted from the task description:

- `~/.local/bin/llama` resolves to `~/.local/share/llama.cpp/llama-b10470/llama`;
  `--version` reports `build 10470, commit 34af94cd9`.
- `libggml.0.20.1.dylib` in that tree imports `_getenv`, `_dlopen`, and `_dlsym`.
- `GGML_BACKEND_PATH` is the only uppercase `GGML_*` env-shaped literal present in that library.
- `GGML_METAL_PATH` does not appear anywhere in the b10470 tree, so the clean control is a
  genuinely unestablished-effect name and the exact-name policy is justified, not lazy.

## Bypass-path search

- All managed spawn sites enumerated: `pi_launch_posix.go:140` (runtime, `Env = opts.Environ`)
  and `:204` (Pi, `Env = opts.Environ` + `PI_*` additions). Both consume the validated slice;
  no profile-supplied environment is merged in.
- `BuildPrimarySessionLaunchPlan("pi", ...)` / `compose --mode primary-session` carry
  `argsPlan.DiagnosticArgv` and no environment, and never exec. Not a launch surface.
- Trailing/leading-whitespace and lowercase lookalike names cannot be read by `getenv`
  (exact compare up to `=`), so case-sensitivity here is correct rather than a hole.
- Duplicate `GGML_BACKEND_PATH` entries hit the exact-name branch on the first occurrence.
- Malformed entries with no `=` are refused as `pi_execution_environment_invalid`.
- Unmanaged passthrough (`selected == ""`, `runPiProcess` at `:78`) is deliberately outside the
  managed boundary and spawns no llama.cpp runtime — pre-existing shape shared with the
  `HF_ENDPOINT`/`MODEL_ENDPOINT` policy, not a regression introduced here.
- Windows `ValidatePiExecutionEnvironment` is a no-op stub, but the Windows `RunPi` refuses any
  managed profile with `pi_compatibility_unsupported` before reaching it. Not reachable.

## Independent validation in the main checkout

| Gate | Exit |
| --- | ---: |
| `go test -count=1 ./...` (72.9s + 2.8s + 104.6s) | 0 |
| `go vet ./...` | 0 |
| `go build ./...` | 0 |
| `gofmt -l .` (no output) | 0 |
| `agents-infra verify global` | 0 |
| `agents-infra verify local .` | 0 |

`TestPiLaunchForwardsSignalsThenCleansRuntime` fails only inside the disposable review copy
because `officialPiAsset` resolves through the symlinked `.temp` asset while `identity.Entrypoint`
resolves symlinks. It passes in the main checkout. Copy artifact, not a defect.

## Scope and regression check

Task-attributable delta is exactly the seven files claimed: `pi_catalog.go` (one member added),
`pi_test.go`, `installed_binary_setup_test.go`, `pi_operator_docs_test.go`, `README.md`,
`SKILL.md`, `LOGBOOK.md`. `installed_binary_setup_test.go` is 704 additions / 2 deletions with no
test function removed — no coverage was traded away. Existing `DYLD_*`, `LD_*`, `NODE_*`, `BUN_*`,
`LLAMA_ARG_*`, `HF_ENDPOINT`, `MODEL_ENDPOINT`, duplicate-name, inbound `PI_*`, token, and
cache-location policies are unchanged and still covered.

## Acceptance criteria

| AC | Status |
| --- | --- |
| Refuses inherited `GGML_BACKEND_PATH` before runtime spawn | met (mutant 1, mutant 3) |
| Leaks no value | met (mutant 4) |
| Clean control reaches runtime backend initialization | met (marker + pid file on both installed surfaces and in-process) |
| Production-entry and installed launcher negatives redden when the exact gate is removed | met — proved by narrowing, stronger than removal |
| Endpoint, `LLAMA_ARG_*`, loader, token, cache policies unchanged | met |
| Bootstrap and local verification pass | met (verify global/local re-run at exit 0) |

## Handoff

Reviewer-archetype run supplies no `commit_ack`. The commit-owning mover should commit this scope
and then make the final `done` transition with `commit_ack=scope_committed`.
