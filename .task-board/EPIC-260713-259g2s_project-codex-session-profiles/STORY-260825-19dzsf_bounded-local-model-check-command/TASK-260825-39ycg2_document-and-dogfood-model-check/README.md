# TASK-260825-39ycg2: document-and-dogfood-model-check

## Description
Document the bounded model checker and use it to reproduce the local Qwen skill-discovery smoke.

## Scope
Update the canonical README and relux-agents-infra skill documentation with concise command examples, artifact locations, exit semantics, timeout/cleanup behavior, and a skill-discovery example. Run the installed command against qwen-infra with a bounded skill-read prompt and preserve sanitized evidence proving whether the model discovers and reads relux-agents-infra/SKILL.md.

## Acceptance Criteria
README and skill docs expose the exact command and artifacts. Setup/install remains valid. A real qwen-infra smoke is run through the new command, the evidence is recorded without secrets, and the observed skill behavior plus any timeout is reported accurately.
