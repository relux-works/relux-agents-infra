# TASK-260830-24m7hw: alert-when-kv-cache-saturates

## Description
The deployed qwen-local profile now runs with --max-kv-size 76800 (previously the flag was absent, so the active generation KV was unbounded; --prompt-cache-bytes 8GB caps the stored prompt-cache pool, which is a different mechanism on a different code path). A bounded rotating KV cache silently overwrites the oldest context once it wraps: in RotatingKVCache._update_in_place the write index resets to keep when it reaches max_size, and _update_concat trims by _idx - max_size + 1. Nothing currently reports that this happened, so a request whose context was truncated is indistinguishable from one that fit. Emit an explicit signal when a request's token count exceeds the active KV bound, and surface it through agents-infra logging so long-context degradation is visible instead of silent.

## Scope
(define task scope)

## Acceptance Criteria
A request whose prompt plus generated tokens exceed the active max KV size produces an explicit, greppable log record naming the request, the active bound, and the observed token count. The check runs once per request, never inside the per-layer per-token cache update path, and the measured decode throughput is unchanged within noise by its presence. The signal reports the bound actually in force as reported by the running server, not the launch argument. agents-infra surfaces the record through the runtime's normal log path so an operator sees it without reading raw server output. A request that fits within the bound produces no record. A test drives the real server with a prompt exceeding the bound and asserts the record appears, and with a prompt under the bound and asserts it does not; removing the check makes the first test red.
