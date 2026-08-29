# TASK-260828-28gdmq: verify-model-io-observability-through-harness

## Description
Establish what the agents-infra managed harness path actually captures of model input and output, and whether every prompt, completion, tool call, error and lifecycle transition can be attributed to its originating request for each managed runtime.

## Scope
(define task scope)

## Acceptance Criteria
For the full chain from a Pi session turn through the agents-infra managed harness to the inference engine and back, evidence shows exactly which inputs and outputs are captured and which are not, whether stdout/stderr proxying and HTTP-borne request and response bodies are both covered, whether a Pi turn can be correlated to the engine requests it caused, and every gap is recorded as a named blocker rather than assumed covered.
