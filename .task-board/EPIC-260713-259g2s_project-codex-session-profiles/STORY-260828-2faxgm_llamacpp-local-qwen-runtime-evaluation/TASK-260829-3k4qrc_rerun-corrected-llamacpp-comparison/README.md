# TASK-260829-3k4qrc: rerun-corrected-llamacpp-comparison

## Description
Re-run the full pinned comparison of llama.cpp against the Python mlx-lm baseline on corrected instrumentation and produce the trustworthy number set the decision will rest on.

## Scope
(define task scope)

## Acceptance Criteria
The full pinned scenario suite runs on corrected instrumentation with MTP off, producing prefill, decode, TTFT, 75000-token capacity, tool-call parity, stability and memory for both runtimes, with every non-comparable dimension refused rather than scored.
