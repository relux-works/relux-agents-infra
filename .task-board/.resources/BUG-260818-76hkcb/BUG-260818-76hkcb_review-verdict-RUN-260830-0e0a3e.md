# BUG-260818-76hkcb reviewer verdict — accepted

Reviewer run: `RUN-260830-0e0a3e` (`reviewer`, Codex).

## Verdict

Accepted. Exact `GGML_BACKEND_PATH` is refused at the shared managed Pi
environment boundary before state creation and runtime spawn, the value is not
reported, and clean unrelated `GGML_*` controls remain admitted. The gate was
attacked rather than inferred from positive tests.

This run was not handed a Change Request revision, so `accept_cr` does not
apply. The task-attributable implementation and operator wording are already
committed ancestors of both `HEAD` and `main` (`97ebda4` and `0413aee`), and
the current filesystem is byte-identical to `HEAD`; no repository change is
required for this leaf. The commit-owning mover should acknowledge the already
committed scope and perform the final `done` transition. This reviewer supplies
no `commit_ack`.

## Production boundary and bypass search

- Policy: `tools/agents-infra/internal/infra/pi_catalog.go:284-304`.
  `GGML_BACKEND_PATH` is an exact, case-sensitive member; `GGML_*` is not a
  denied prefix.
- Primary managed entry: `RunPi` calls the policy at
  `tools/agents-infra/internal/infra/pi_launch_posix.go:132`, before identity
  checks, state resolution/creation, lock acquisition, and runtime/Pi spawn.
- Shared Pi repeats the check before client Pi spawn at
  `pi_shared_client_darwin.go:686`.
- Independently callable `runtime runtime-launch` calls the same policy at
  `pi_shared_launcher_darwin.go:94`, immediately before `sharedRuntimeExecve`.
  A reviewer-added disposable production test proved exact
  `GGML_BACKEND_PATH` is refused there and that `GGML_METAL_PATH` reaches the
  exec target.
- The managed launcher surfaces are the bootstrap-global `pi-infra` alias and
  the setup-generated project-local wrapper. Both rebuild and drive the real
  CLI entry, not a helper-only fake.
- Unmanaged Pi passthrough remains deliberately outside this boundary and does
  not launch the managed llama.cpp runtime.

## Reviewer attacks

All mutations were applied only in
`.temp/BUG-260818-76hkcb-review/attack-copy`; repository source was not edited.
Every test invocation used `-count=1`.

| Attack | RunPi production | Installed global/local | Direct shared-runtime | Result |
| --- | ---: | ---: | ---: | --- |
| Narrow exact member to `GGML_BACKEND_PATH_ZZZ` | FAIL | FAIL | FAIL | Exact lower bound is enforced |
| Broaden policy to `GGML_*` | FAIL | FAIL | FAIL | Exact upper bound is enforced by `GGML_METAL_PATH` controls |
| Append denied value to diagnostic | FAIL | FAIL | FAIL | Non-disclosure is asserted |
| Move validation after `CreatePiStateTree` | FAIL | n/a | n/a | Pre-state ordering is asserted |

The narrowing mutant did not merely remove the guard. Its direct shared-runtime
failure observed the target process instead of the refusal; the RunPi and
installed failures reached runtime initialization. The ordering mutant created
`agents-infra/` under the cache and was caught by the existing pre-state
assertion. The broadening mutant rejected `GGML_METAL_PATH` on all three
production surfaces. The leakage mutant exposed the canary and was caught on
all three surfaces.

## Factual premise

The local installed runtime reports `build 10470, commit 34af94cd9`.
`libggml.0.20.1.dylib` imports `_getenv`, `_dlopen`, and `_dlsym` and contains
the `GGML_BACKEND_PATH` literal. `GGML_METAL_PATH` is absent from the installed
b10470 tree. An initial probe selected `libggml-base.0.20.1.dylib`, which is the
wrong component for backend discovery; that partial read was not treated as
absence. The corrected inspection enumerated all `libggml*.dylib` files and
identified the loader-bearing library explicitly.

## Validation

Pristine current checkout:

| Gate | Result |
| --- | ---: |
| Focused production lifecycle and direct shared launcher | PASS |
| Rebuilt bootstrap-global/project-local launcher controls and docs | PASS |
| `go test -count=1 ./...` | PASS |
| `go vet ./...` | PASS |
| `go build ./...` | PASS |
| `gofmt -l .` | clean |
| `git diff --check HEAD` | PASS |
| Filesystem compared with `HEAD` | identical |

The task's earlier attached raw evidence was also re-read. It records successful
`./setup.sh`, `agents-infra verify global`, task-scoped local setup, and
`agents-infra verify local` runs. This review did not repeat those external
installation mutations; the current-source installed-wrapper fixture rebuilt
both launcher surfaces and passed.

## Scope note for the mover

The managed worktree index differs from `HEAD`, while the filesystem does not.
The staged delta is the inverse of already checkpointed sibling BUG changes
(`BUG-260817-2bh9nk` and `BUG-260818-1s1lka`), not this task's candidate. Do not
commit or checkpoint that index as `BUG-260818-76hkcb` scope. The target gate,
tests, README, skill guidance, and prior logbook record are already present in
committed `main` history.
