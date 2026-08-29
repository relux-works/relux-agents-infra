# TASK-260828-3fgca3: integrate-llamacpp-into-benchmark-gate

## Description
Make the comparison gate able to judge llama.cpp as a candidate on the single-invocation construction delivered by TASK-260827-2v13w8, including a runtime-aware KV bound so an unbounded-by-absence derivation cannot assert something false about llama.cpp.

## Scope
(define task scope)

## Acceptance Criteria
llama.cpp is admissible as a benchmark candidate under the single-invocation construction with no admission clause relaxed, the KV-bound derivation no longer reads absence of an argv flag as unbounded for a runtime that is never unbounded, and a narrowing mutant proves the bound.
