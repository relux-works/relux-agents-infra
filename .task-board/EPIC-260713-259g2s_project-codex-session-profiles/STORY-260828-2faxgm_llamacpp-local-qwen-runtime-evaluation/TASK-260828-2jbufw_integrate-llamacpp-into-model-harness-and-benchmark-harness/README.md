# TASK-260828-2jbufw: integrate-llamacpp-into-model-harness-and-benchmark-harness

## Description
Add a model-harness profile for llama-server and make the existing benchmark driver and benchmark-compare gate able to run llama.cpp as a candidate without weakening any admission clause.

## Scope
model-harness profile and managed lifecycle for llama-server only. Benchmark-gate admission for llama.cpp is split out to its own task because the gate is being rebuilt on a single-invocation construction by TASK-260827-2v13w8.

## Acceptance Criteria
llama-server runs under a model-harness profile with the managed lifecycle, health and readiness contracts, and blocker B8 from TASK-260828-28gdmq is answered with evidence for what llama-server emits on its captured streams.
