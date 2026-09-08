# STORY-260829-3rs679: publish-resource-pressure-status-on-fresh-trunk

## Description
Add a typed resource-pressure and inference-busy observation contract independently of restart/backoff work.

## Scope
Static and fake-provider infra work only; no live local-model access.

## Acceptance Criteria
The owned task passes independent review and lands from exact current trunk.
