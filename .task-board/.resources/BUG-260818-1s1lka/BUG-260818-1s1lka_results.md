# BUG-260818-1s1lka — Developer Evidence

## Implementation

- Production call site: `RunPi` calls `ValidatePiExecutionEnvironment` before `VerifyPiExecutionIdentity`, `ResolvePiStatePaths`, `CreatePiStateTree`, profile lock acquisition, and runtime `exec.Command`.
- Exact `LLAMA_API_KEY` is refused with `pi_execution_environment_invalid`; the diagnostic contains the environment name only.
- `HF_TOKEN`, `HF_HOME`, `HUGGINGFACE_HUB_CACHE`, `TRANSFORMERS_CACHE`, `LLAMA_API_KEY_SUFFIX`, `UNRELATED_SERVICE_API_KEY`, and the existing unrelated controls remain admitted through runtime initialization.
- README and installed relux-agents-infra skill document the no-ambient-auth policy.

## Negative and narrowing evidence

A temporary mutant narrowed the gate to reject `LLAMA_API_KEY` only when its value was empty. Tests used non-empty values and were run with `-count=1`.

- `go test ./internal/infra -count=1 -run 'TestPiExecutionEnvironmentRejectsExactLlamaAPIKeyWithoutExposingValue|TestPiLaunchRejectsLoaderAndInboundPiEnvironmentBeforeState'`: expected red, exit 1. Both helper and production `RunPi` cases admitted the non-empty key.
- `go test . -count=1 -run '^TestInstalledPiLaunchersRejectExactEnvironmentNamesBeforeRuntimeSpawn$'`: expected red, exit 1. Bootstrap-global alias and project-local wrapper both admitted the non-empty key.
- Source restored from a task-scoped copy; `cmp` over production and test files exited 0 before green reruns.

## Passing validation

- Focused helper, production refusal, pre-state, and admitted-control suite: exit 0.
- Installed global/local launcher suite: exit 0.
- Operator README/skill docs suite: exit 0.
- `go test ./... -count=1`: exit 0 (`main` 152.153s; `internal/attachments` 4.833s; `internal/infra` 192.827s).
- `go vet ./...`: exit 0.
- `go build ./...`: exit 0.
- `git diff --check`: exit 0.
- `./setup.sh`: exit 0; rebuilt and installed the bootstrap-owned global binary.
- `agents-infra verify global`: exit 0.
- `agents-infra setup local .temp/BUG-260818-1s1lka/local-project`: exit 0.
- `agents-infra verify local .temp/BUG-260818-1s1lka/local-project`: exit 0.

No files were staged or committed. The checkout already contained the parent story's uncommitted managed-Pi implementation; this change was applied additively without resetting or absorbing unrelated work.
