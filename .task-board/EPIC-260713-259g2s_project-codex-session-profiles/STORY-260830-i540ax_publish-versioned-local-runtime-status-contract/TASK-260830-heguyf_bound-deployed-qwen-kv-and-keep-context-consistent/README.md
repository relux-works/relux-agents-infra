# TASK-260830-heguyf: bound-deployed-qwen-kv-and-keep-context-consistent

## Description
The deployed qwen-local model-harness profile ran without --max-kv-size, so the active generation KV cache was unbounded while context_window was 75000. The orchestrator added --max-kv-size 76800 inline on 2026-08-30 (backup at .temp/model-harness.toml.bak-kvbound). This task makes that consistent and enforced rather than a manual edit: the KV bound and the profile's context_window must move together, because a context_window larger than the KV bound silently truncates and a bound far above the window wastes memory the host does not have. Host is 64 GB; a 73k-token prompt already peaked around 47.8 GB resident.

## Scope
(define task scope)

## Acceptance Criteria
The deployed profile's KV bound is at least its configured context_window, and a configuration where context_window exceeds the KV bound is refused at the production entry point with an error naming both values. Raising context_window without raising the bound fails closed rather than silently truncating. A test drives the real resolution path with a window above the bound and asserts refusal, and with a consistent pair and asserts admission. The relationship is documented where an operator setting context_window will see it.
