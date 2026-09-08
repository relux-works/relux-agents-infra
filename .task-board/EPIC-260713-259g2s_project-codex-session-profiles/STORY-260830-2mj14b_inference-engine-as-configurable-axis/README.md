# STORY-260830-2mj14b: inference-engine-as-configurable-axis

## Description
Make the local-model configuration three-axis — model artifact, runtime environment and inference engine — so engine-specific argv spellings, stream semantics, health contracts, weight formats and memory accounting live behind an adapter instead of being baked into a profile executable and argv string.

## Scope
Do NOT build a second plugin system here. skill-agents-management already owns the agentic-system and vendor plugin layers and documents a planned local-model plugin with load/unload awareness, inference-busy state and memory-pressure sequencing. The inference-engine axis belongs there, and agents-infra becomes a consumer alongside task-board so extension happens in one place.

## Acceptance Criteria
The inference engine is a first-class configurable axis implemented as a local-model plugin layer in skill-agents-management, with agents-infra consuming it rather than carrying its own engine handling, so adding an engine touches one repository.
