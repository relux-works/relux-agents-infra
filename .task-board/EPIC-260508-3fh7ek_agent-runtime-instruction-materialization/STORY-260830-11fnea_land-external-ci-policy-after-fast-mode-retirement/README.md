# STORY-260830-11fnea: land-external-ci-policy-after-fast-mode-retirement

## Description
Replay the independently attacked external-CI fallback policy on exact current protected trunk after fast_mode was deliberately removed.

## Scope
Four policy paths plus board evidence only; task-board.config.json remains authoritative fast_mode-absent and outside the candidate delta.

## Acceptance Criteria
Fresh workspace selected base equals fetched origin/main d69a435 or newer authoritative trunk; exact four policy paths only; additive trigger and Claude/Codex include-entrypoint bypass mutants fail; fast_mode remains absent and preflight resolves false/default; full tests, vet, build, setup, installed parity, independent review, PR and merge pass.
