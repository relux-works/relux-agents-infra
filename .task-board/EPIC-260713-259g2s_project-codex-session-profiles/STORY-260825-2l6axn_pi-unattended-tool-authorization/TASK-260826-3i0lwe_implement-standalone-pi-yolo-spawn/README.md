# TASK-260826-3i0lwe: implement-standalone-pi-yolo-spawn

## Description
Implement a board-agnostic standalone qwen-infra/Pi spawn command that launches one independent non-interactive worker against the configured local Qwen runtime and executes its authorized tools without human approval prompts.

## Scope
Change agents-infra and its source-of-truth Pi configuration/docs/tests only. Add an explicit standalone spawn or headless-session contract; do not add a task-board runtime adapter, task-board provider registration, root execution, unrestricted sudo, raw RPC bash, implicit extension discovery, or caller-controlled overrides of the yolo safety arguments. Preserve the interactive qwen-infra path. Reuse the shared runtime lease broker when configured so multiple independent workers can share one MLX runtime while keeping separate Pi processes, sessions, and state.

## Acceptance Criteria
A caller can launch qwen-infra in standalone unattended mode with a prompt and receive a deterministic terminal result; the launcher owns --no-approve, --no-extensions, and an exact validated tool allowlist; conflicting Pi authorization or extension flags are refused before launch; stdin and UI approval are not required; two concurrently launched workers remain independent while reusing one verified local runtime and do not tear it down prematurely; reasoning medium from the effective qwen profile is preserved; failures are typed and sanitized; interactive behavior remains compatible; tests include real no-model Pi tool-side-effect proof, extension replacement rejection, direct-RPC bypass exclusion, conflicting-flag refusals, concurrent worker/runtime reuse, crash cleanup, and no task-board dependency or integration claim.
