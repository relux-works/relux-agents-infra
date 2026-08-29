# STORY-260824-1yr6m0: vendor-environment-model-targets

## Description
Implement Owner Requirements R1-R6 through TASK-260824-3rl3ws vendor target contract Sections 2-7: explicit vendor/environment/model/reasoning identity, configuration-driven vendor aliases, and additive legacy compatibility.

## Scope
Contract and implementation for agents-infra project-config parsing/composition/diagnostics; installed openai-infra, anthropic-infra, and qwen-infra; compatibility with agents.codex, agents.claude, and agents.pi; deployment of the three Section 2 target tuples.

## Acceptance Criteria
Traceability: contract R1-R6 and Sections 2-7. The three configured targets resolve unambiguously with provenance; legacy direct launch behavior remains unchanged; aliases are configuration-driven and identity-locked; production negatives fail closed; setup/verify and required Qwen text/tool smokes pass.
