# BUG-260817-2bh9nk — reviewer verdict (cycle 1)

**Verdict: accepted.** Independent attack evidence below; no rework requested.
Reviewer archetype supplied no `commit_ack`; the commit-owning mover still owns
the final `done` transition.

## Scope reviewed

Managed Pi launch must refuse or explicitly bind the exact llama.cpp
model-origin environment names `HF_ENDPOINT` and `MODEL_ENDPOINT` before the
llama.cpp runtime is spawned, without leaking values.

Enforcement: `tools/agents-infra/internal/infra/pi_catalog.go:295` inside
`ValidatePiExecutionEnvironment`.
Production call chain: `tools/agents-infra/main.go:423` (`os.Environ()`) →
`infra.RunPi` → `tools/agents-infra/internal/infra/pi_launch_posix.go:94`, which
runs before `ResolvePiStatePaths`/`CreatePiStateTree` (line 110-115) and before
`runtimeCmd.Start()` (line 148).

## What I attacked (not read)

### 1. Independent narrowing mutants, re-run by review

Mutated only `pi_catalog.go:295` in a pristine copy at
`.temp/BUG-260817-2bh9nk/review-copy` (main checkout never modified).

| Mutant | Unit gate test | Production `RunPi` entry test | Installed launcher test |
| --- | --- | --- | --- |
| `name == "HF_ENDPOINT"` only | FAIL on `MODEL_ENDPOINT` | FAIL `.../model_origin` | not run |
| `name == "MODEL_ENDPOINT"` only | FAIL on `HF_ENDPOINT` | FAIL `.../hf_model_origin` | FAIL on both real surfaces (`bootstrap global alias/HF_ENDPOINT`, `project local wrapper/HF_ENDPOINT`) |
| pristine | PASS | PASS | PASS |

Both endpoint names are independently bound. A single-name gate cannot survive.

### 2. Docs regression gate is a real gate, not a comment

| Mutant | Result |
| --- | --- |
| README narrowed to `HF_ENDPOINT` only | `TestPiOperatorContractDocumentsCycle10Boundary` FAILs with the exact missing fragment |
| SKILL.md narrowed to `HF_ENDPOINT` only | `TestReluxAgentsInfraSkillRoutesSafePiWorkflowToSource` FAILs |

Both operator surfaces carry the clause; neither is a one-sided binding.

### 3. Real installed bootstrap-owned launcher, with a positive clean control

Probed `~/.local/bin/pi-infra` → `~/.local/bin/agents-infra`
(sha256 `9859d591820a6232002cd31d4ab20a67608857eb85cbac4de9710beacbe9a058`,
mtime 2026-08-17 23:48) against a scratch project with a marker-writing runtime
and the official Pi asset on `PATH`, under `env -i`.

| Run | Exit | Diagnostic | Value leaked | Runtime child started |
| --- | ---: | --- | --- | --- |
| Clean control | non-zero (`runtime readiness timed out`) | — | — | **yes** (`runtime-started` marker created) |
| `HF_ENDPOINT=https://leak-canary-HF_ENDPOINT.invalid/weights` | 1 | `runtime-affecting environment name "HF_ENDPOINT" is denied` | no | no |
| `MODEL_ENDPOINT=https://leak-canary-MODEL_ENDPOINT.invalid/weights` | 1 | `runtime-affecting environment name "MODEL_ENDPOINT" is denied` | no | no |

The control is what makes this an ordering proof: the same fixture reaches
runtime spawn when the names are absent, so "no marker" is refusal, not an
unrelated early failure.

### 4. Proxy signal explicitly rejected

`strings ~/.local/bin/agents-infra | grep -c HF_ENDPOINT` returns `0` even
though the installed binary demonstrably enforces the gate — Go compiles short
literal string equality into immediate-constant comparisons, so the literal is
absent from rodata. The `strings` check used in the earlier cycle
(LOGBOOK 2310) is a proxy that would have reported this gate as missing.
Only the behavioral probe in section 3 establishes the property.

### 5. Bypass and forgery search

- No config-side injection channel: `PiRuntime` (`pi_config.go:56`) has no env
  field, and `rejectUnknownFields` blocks unknown TOML keys.
- Both managed spawn sites inherit `opts.Environ` verbatim
  (`pi_launch_posix.go:142` runtime, `:206` Pi via `managedEnv`); both are
  downstream of the gate. `piExecCommand`/`exec.Command` never re-read
  `os.Environ()`.
- Malformed / empty-name / duplicate entries are rejected before the name test,
  so `HF_ENDPOINT` cannot be smuggled as a nameless or duplicated entry; an
  empty value (`HF_ENDPOINT=`) is denied too, because the rule is name-only.
- Diagnostics surface: `--print-config` builds the plan without any environment
  input, so `BuildPrimarySessionLaunchPlan` has no leak path.
- Unmanaged pass-through (`pi_launch_posix.go:78`, no profile selected) is not
  gated, which is correct: no managed llama.cpp runtime is started there and the
  policy is scoped to managed profiles.

### 6. Scope honesty checked

`HF_HOME`, `HF_TOKEN`, `HUGGING_FACE_HUB_TOKEN` and lowercase lookalikes stay
admitted, and the clean-control test pins that. This matches the task scope
("cover exact verified endpoint names first; treat tokens and cache variables
separately"), and README/SKILL.md both state the limit explicitly rather than
implying wider coverage. POSIX `getenv` is case-sensitive, so exact-name
matching is the correct shape for the reported llama.cpp behavior.

## Suite state (main checkout, pristine)

| Gate | Exit | Result |
| --- | ---: | --- |
| `gofmt -l .` | 0 | clean |
| `go vet ./...` | 0 | clean |
| `go build ./...` | 0 | ok |
| `go test ./... -count=1` | 0 | 3/3 packages ok (52.8s / 1.3s / 81.1s) |

Working tree after review is byte-identical to the pre-review snapshot; all
review scratch lives under `.temp/BUG-260817-2bh9nk/`.

## Non-blocking observation (not rework for this task)

`officialPiAsset` (`pi_test.go:1364`) skips when the gitignored
`.temp/TASK-260817-2h8hn4/...` asset is absent, so the in-package production
entry test `TestPiLaunchRejectsLoaderAndInboundPiEnvironmentBeforeState`
silently skips on a clean clone. This gate is not left uncovered — 
`TestInstalledPiLaunchersRejectModelOriginEnvironmentBeforeRuntimeSpawn`
(`installed_binary_setup_test.go:633`) needs no such asset, drives both real
launcher surfaces, and reddens under the narrowing mutant. The skip pattern is
pre-existing from TASK-260817-2h8hn4 and is out of scope here; worth a separate
item if the acceptance asset should become fetchable or the skip should become a
hard failure in CI.
