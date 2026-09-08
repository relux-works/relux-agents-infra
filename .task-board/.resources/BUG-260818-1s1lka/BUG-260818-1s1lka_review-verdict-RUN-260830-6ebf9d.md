# BUG-260818-1s1lka review verdict — RUN-260830-6ebf9d

## Verdict

**CHANGES REQUESTED** — route to `to-dev` for additive negative-test coverage. The production equality gate is correct; no production-code rewrite is requested.

## Blocking finding

### F1 — Exact-name upper bound is not protected on production and installed surfaces

`tools/agents-infra/internal/infra/pi_catalog.go:295` correctly refuses case-sensitive exact `LLAMA_API_KEY`, but the shipped tests do not carry the lowercase lookalike `llama_api_key`. Replacing only that equality with `strings.EqualFold(name, "LLAMA_API_KEY")` survived all shipped task-relevant gates:

- internal helper and production `RunPi` focused suite: exit 0;
- installed bootstrap-global alias and project-local wrapper suite: exit 0 with every named subtest passing.

That mutant violates the source contract in `SKILL.md:508-513`: `LLAMA_API_KEY` lookalikes remain admitted unless separately justified. It also violates the task's exact-name scope and admitted-unrelated-name control. `LLAMA_API_KEY_SUFFIX` pins only the prefix/family dimension; it cannot detect case broadening.

Adding `llama_api_key=case-sensitive-lookalike` only in the disposable review copy made the same mutant fail at every required surface:

- `TestPiExecutionEnvironmentAcceptsExactCleanEnvironment` rejected the lowercase lookalike;
- `TestPiLaunchCleanEnvironmentReachesRuntimeBackendInitializationAndPreservesGlobalState` refused before runtime initialization;
- both installed clean controls failed before their runtime marker was created.

Required rework: add that lowercase control to the helper clean environment, the production `RunPi` clean lifecycle, and the installed global/local launcher clean environment. Re-run the `EqualFold` mutant with `-count=1` and the pinned Pi asset present; it must redden the named production and installed tests. Keep the existing suffix, `HF_TOKEN`, cache-variable, and unrelated-name controls.

## Gate attacks and validation

- Baseline focused helper/RunPi tests: exit 0.
- Baseline installed global/local plus README/SKILL docs tests: exit 0.
- Narrow-to-empty mutant: exact helper test and production `RunPi` test exit 1; with the pinned Pi asset present, both installed launcher `LLAMA_API_KEY` subtests exit 1 because the runtime becomes reachable.
- Important harness note: the first disposable installed-mutant run reported package PASS only because `mainTestOfficialPiAsset` called `t.Skip`; after copying the byte-verified pinned asset, the same mutant correctly reddened. A skipped gate is not positive evidence.
- Full pristine `go test ./... -count=1`: exit 0 (`internal/infra` 328.656s).
- `go vet ./...`, `go build ./...`, `gofmt -l .`, and `git diff --check HEAD`: exit 0/clean.
- Canonical `setup.sh` in a clean external source copy, isolated global verify, isolated local setup, and isolated local verify: exit 0.
- Installed runtime `verify global`: exit 0. Direct `verify local` against this source worktree refused because a source checkout is not an installed runtime; isolated canonical local setup/verify then passed.

## Scope and state

- Production call site reviewed: `RunPi` calls `ValidatePiExecutionEnvironment` before identity/state/runtime work; the shared-runtime launcher also calls it immediately before `sharedRuntimeExecve`.
- Run is not goal-bound and no Change Request was handed to this reviewer.
- `git diff HEAD` is empty. Existing staged/unstaged `MM` entries belong to checkpointed sibling `BUG-260817-2bh9nk` and were preserved unchanged.
- Mutants and reviewer controls existed only under task-scoped `.temp/` disposable copies.
- The case-control gap is already recorded in `LOGBOOK.md` entry `2026-08-29 0205`; no duplicate logbook entry was added.

