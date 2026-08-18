# BUG-260818-1s1lka reviewer verdict — accepted

Reviewer run: RUN-260817-8fb08d (claude-opus-5). Read-only review; no product code modified.
All mutation work was done in two disposable copies under `.temp/BUG-260818-1s1lka/review-mutant`
and `.../review-mutant2` (rsync of the dirty working tree, `.temp/TASK-260817-2h8hn4` Pi asset
symlinked in), removed after the review. `internal/infra/pi_catalog.go` in the main checkout was
byte-verified unchanged afterwards.

## Verdict

Accepted. Every acceptance criterion was independently re-established by attacking the gate, not
by rerunning the developer's logs. One residual coverage gap is recorded below; it does not
falsify any AC and does not affect production behavior.

## Gate under review

`tools/agents-infra/internal/infra/pi_catalog.go:295` — exact, case-sensitive `LLAMA_API_KEY`
added to the existing `HF_ENDPOINT` / `MODEL_ENDPOINT` / `GGML_BACKEND_PATH` exact-name member of
`ValidatePiExecutionEnvironment`.

Production call site: `tools/agents-infra/internal/infra/pi_launch_posix.go:94`, inside `RunPi`,
after `BuildManagedPiArguments` and before `VerifyPiExecutionIdentity`,
`inspectRuntimeExecutable`, `ResolvePiStatePaths`, `CreatePiStateTree`, `AcquirePiProfileLock`,
`GeneratePiModelsJSON`/`WritePiModelsJSON`, `preflightPiListener`, and both `exec` spawns.
Reached from the single production caller `main.go:423` (`agents-infra pi` -> `infra.RunPi`),
which the bootstrap-global `pi-infra` alias and the setup-generated project-local `pi-infra`
wrapper both exec. Re-validated post-readiness at `pi_launch_posix.go:197`.

## Attacks executed by the reviewer (not reruns of developer logs)

| # | Mutant applied to a disposable copy | Production `internal/infra` | Installed-launcher suite | Result |
| --- | --- | ---: | ---: | --- |
| 1 | Narrow: refuse `LLAMA_API_KEY` only when its value is empty | FAIL | FAIL | Class bound is real, not delete-only |
| 2 | Broaden: `name == "LLAMA_API_KEY"` -> `strings.HasPrefix(name, "LLAMA_API_KEY")` | FAIL | not run | Family upper bound is real |
| 3 | Reorder: move the gate from pre-state to after `CreatePiStateTree` | FAIL | not run | "before managed state" is asserted |
| 4 | Leak: append the denied value to the refusal message | FAIL | not run | Value non-disclosure is asserted |
| 5 | Broaden (case): `name == "LLAMA_API_KEY"` -> `strings.EqualFold(...)` | **PASS (survives)** | not run | See residual gap |
| — | Pristine baseline, same copies | PASS | PASS | Mutants, not the harness, caused the red |

Mutant 1 red detail — the runtime actually spawned with the key admitted, so the gate is what stops
the spawn, not an incidental failure:
- `internal/infra`: `TestPiExecutionEnvironmentRejectsExactLlamaAPIKeyWithoutExposingValue` →
  `LLAMA_API_KEY was admitted: <nil>`; `TestPiLaunchRejectsLoaderAndInboundPiEnvironmentBeforeState/llama_api_key`
  → `production environment shape=llama api key err=runtime exited before readiness`.
- `TestInstalledPiLaunchersRejectExactEnvironmentNamesBeforeRuntimeSpawn`: both
  `bootstrap global alias/LLAMA_API_KEY` and `project local wrapper/LLAMA_API_KEY` FAIL with
  `admitted LLAMA_API_KEY: exit status 1 / runtime exited before readiness`, while all four clean
  controls and the three sibling names stayed green in the same run.

Mutant 2 red detail: `LLAMA_API_KEY_SUFFIX=not-the-exact-auth-control` is refused on both the
helper clean control and the full in-process lifecycle test
(`TestPiLaunchCleanEnvironmentReachesRuntimeBackendInitializationAndPreservesGlobalState`), so the
"exact name, not the `LLAMA_API_KEY*` family" boundary is pinned in both directions.

Mutant 3 red detail: `environment refusal created managed state: [d agents-infra/]` for every
member of the refusal table including `llama_api_key`, not only the new one.

Mutant 4 red detail: the helper test pins the exact refusal string, and the production table test
fails with `production RunPi refusal exposed an inherited environment value` for `llama_api_key`
alongside the three sibling names.

## Bypass-path search

- Managed spawn sites enumerated: `pi_launch_posix.go:140` (runtime, `Env = opts.Environ`) and
  `:204` (Pi, `Env = opts.Environ` + `PI_*` additions). Both consume the validated slice.
- `PiProfile`/`PiRuntime` carry no environment field, so no profile-supplied environment is merged
  in after validation.
- `agents-infra pi --print-config` takes the `BuildPrimarySessionLaunchPlan` branch and never
  execs; not a launch surface.
- `RunPi` has exactly one production caller (`main.go:423`); no second managed entry point exists.
- Duplicate `LLAMA_API_KEY` entries hit the exact-name branch on the first occurrence; an entry
  with no `=` and an entry with an empty name are refused as malformed; `LLAMA_API_KEY=a=b` splits
  on the first `=` and is refused.
- Unmanaged passthrough (`selected == ""`, `runPiProcess` at `:78`) is deliberately outside the
  managed boundary and spawns no llama.cpp runtime — pre-existing shape shared with the
  `HF_ENDPOINT`/`MODEL_ENDPOINT`/`GGML_BACKEND_PATH` policy, not a regression introduced here.
- Windows `ValidatePiExecutionEnvironment` is a no-op stub, but Windows `RunPi` refuses any managed
  profile with `pi_compatibility_unsupported` before reaching it. Not reachable.

## Residual gap (accepted, recommended follow-up)

Mutant 5 — replacing the exact compare with `strings.EqualFold` — **survives the entire
`internal/infra` package** (`go test ./internal/infra -count=1` under the mutant: the only failures
are the two `TestPiLaunchForwardsSignalsThenCleansRuntime` subtests, which also fail on a pristine
copy of the same tree and are the known rsync/symlink copy artifact recorded in `LOGBOOK.md`
2026-08-18 0119; the whole package is green in the main checkout).

Cause: `TestPiExecutionEnvironmentAcceptsExactCleanEnvironment` carries a lowercase case-sensitivity
control for every sibling exact name (`hf_endpoint`, `model_endpoint`, `ggml_backend_path`) but not
for `llama_api_key`. `SKILL.md` shipped by this change asserts `LLAMA_API_KEY` lookalikes remain
admitted; the suffix dimension of that claim is tested, the case dimension is not.

Severity: low. A case-insensitive regression would over-refuse a name llama.cpp never reads
(`getenv` is exact), i.e. a usability regression, not an authentication bypass. It falsifies no AC.

Recommended one-line follow-up: add `"llama_api_key=case-sensitive-lookalike"` to the clean list in
`tools/agents-infra/internal/infra/pi_test.go` (`TestPiExecutionEnvironmentAcceptsExactCleanEnvironment`).

## Premise verification

The llama.cpp b10470 binary that the prior cycle used to verify the `GGML_BACKEND_PATH` premise
(`~/.local/share/llama.cpp/llama-b10470`) is no longer present on this host — a filesystem search
for `llama-server`/the b10470 tree returned nothing. The "`LLAMA_API_KEY` is the environment backing
`--api-key` in build 10470" premise is therefore **carried forward from the task description and the
prior cycle's artifact evidence, not independently re-established in this run**. Reported as
unverified-in-this-run rather than inferred from a proxy signal.

Corollary, also unknown rather than answered: whether b10470 exposes any other ambient-auth
environment name (e.g. an env backing `--api-key-file`). The repo policy requires an established
runtime effect before adding a name, so this is a separate investigation, not a defect here.

## Independent validation in the main checkout

| Gate | Exit |
| --- | ---: |
| `go test ./... -count=1` (main 104.6s, attachments 2.9s, internal/infra 169.7s) | 0 |
| `go vet ./...` | 0 |
| `go build ./...` | 0 |
| `gofmt -l .` (no output) | 0 |
| `git diff --check` | 0 |
| `agents-infra verify global` | 0 |
| `agents-infra verify local .temp/BUG-260818-1s1lka/local-project` | 0 |

Docs surfaces confirmed present and pinned by `pi_operator_docs_test.go`: `README.md:609` and
`SKILL.md:325` state the no-ambient-auth policy, the b10470 `--api-key` backing, name-only refusal,
and that `HF_TOKEN`/cache variables/lookalikes stay admitted.

## Handoff

Reviewer archetype must not supply `commit_ack`. Acceptance evidence is recorded here; the task is
routed `to-review` for the commit-owning mover, which commits this scope and then makes the final
`done` transition with `commit_ack=scope_committed`.
