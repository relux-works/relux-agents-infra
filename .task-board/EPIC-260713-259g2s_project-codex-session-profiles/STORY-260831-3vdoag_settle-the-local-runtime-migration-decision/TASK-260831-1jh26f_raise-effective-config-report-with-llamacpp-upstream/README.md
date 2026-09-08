# TASK-260831-1jh26f: raise-effective-config-report-with-llamacpp-upstream

## Description
Build b10621-c1d0e7a00 serialises neither its effective prefill chunk (n_ubatch) nor its effective reasoning effort on any of its 44 routes, verified by direct probe including /props, /slots, /v1/models and --metrics. That is why the comparison gate refused the pair with exit 4. A live report of those two values is the single change that makes this pair scoreable without weakening the gate.

## Scope
(define task scope)

## Acceptance Criteria
An upstream issue or discussion is opened with llama.cpp describing the need and the exact two values, referencing the measured route survey. The request is for the runtime to report what it actually parsed, not for a way to read them from argv. The outcome, including a refusal or an existing mechanism we missed, is recorded here. No local workaround that infers these values from launch arguments is introduced, because that is the exact bypass the gate exists to refuse.
