# TASK-260825-rtmcsw: implement-bounded-model-check

## Description
Implement the reusable agents-infra local-model behavior check command and its production-entrypoint tests.

## Scope
Follow existing agents-infra CLI conventions. Resolve a configured canonical entrypoint/target; execute Pi or the selected environment with a default bounded deadline; write raw JSONL and stderr into an explicit output directory; parse events into a sanitized machine-readable and human-readable summary containing exit/timeout, duration, provider/model, event counts, tool calls and failures, bounded final response, expectation results, and managed-runtime cleanup state; support generic prompt, expected tool, expected text, and deadline overrides; never print secrets. Reuse the existing managed local-runtime lifecycle rather than duplicating a shell launcher.

## Acceptance Criteria
The production CLI accepts a configured target and prompt, applies a safe default deadline, persists evidence, reports a stable summary, exits non-zero for timeout/expectation failure/malformed event stream, and confirms cleanup. Tests drive the real CLI entrypoint and include happy path plus narrowed negative cases for timeout, missing expected tool/text, malformed JSONL, failed tool execution, and cleanup.
