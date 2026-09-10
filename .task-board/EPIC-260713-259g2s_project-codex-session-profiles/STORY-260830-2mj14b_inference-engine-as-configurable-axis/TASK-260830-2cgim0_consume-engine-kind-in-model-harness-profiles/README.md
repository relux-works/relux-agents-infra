# TASK-260830-2cgim0: implement-engine-axis-in-model-harness-profiles

## Description
Make model, environment and engine independently configurable in local-model profiles, with per-engine translation of the canonical knobs.

## Scope
tools/agents-infra/internal/modelharness (profile config, knob translation) and the agents-infra local-model target resolution that reads profiles; the canonical knob set is the landed .research/260831_engine-adapter-contract-and-canonical-knob-set.spec.md. No engine adapter implementation (upstream, TASK-260830-1e9gse); no change to the deployed qwen-local profile semantics beyond adding an explicit engine field defaulting to the current mlx-lm behaviour.

## Acceptance Criteria
A local-model profile declares model, environment and engine as independent fields; engine defaults to the current mlx-lm server behaviour so existing profiles resolve unchanged (golden test on the deployed qwen-local profile); each canonical knob from the spec is translated per engine through one table, and a knob the selected engine cannot express refuses at profile resolution with an error naming the knob and engine instead of being dropped; negative tests cover an unknown engine, an unexpressible knob, and a profile that sets a knob only valid for another engine; the spec is cross-referenced from the profile documentation.
