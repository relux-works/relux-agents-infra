# TASK-260828-2wcrph: benchmark-llamacpp-against-python-mlx-baseline

## Description
Run the pinned scenario suite against llama.cpp on the same host and record prefill, decode, TTFT, 75k capacity, tool-call parity, stability and memory alongside the existing baseline.

## Scope
(define task scope)

## Acceptance Criteria
The pinned scenario suite runs against llama.cpp on the same host as the existing baseline, with prefill, decode, TTFT, 75000-token capacity, tool-call parity, stability and peak physical footprint recorded, and non-comparable pairs refused rather than scored.
