# STORY-260830-1fa5vy: land-codex-fast-policy-on-fresh-trunk

## Description
Replay the validated task-board.config.json-only Codex fast-mode policy on the exact freshly fetched protected trunk without inheriting the rejected stale Story workspace.

## Scope
(define story scope)

## Acceptance Criteria
Fresh Story workspace selected_base_oid equals the fetched upstream main OID; the only production diff is task-board.config.json; spawn-policy-v4 remains exclusive Codex with exactly gpt-5.6-sol/high, fast_mode true, and all workload classes including unified aligned; preflight and repository validation pass; an independent reviewer accepts the exact CR before PR delivery.
