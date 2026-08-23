# STORY-260824-9exx3n: remove-codex-fast-profile

## Description
Deliver the owner-requested removal of the retained Codex fast profile through an isolated Story that can land independently of unrelated legacy review backlog.

## Scope
Replay the already accepted CR-TASK-260824-1qm60c-2 candidate onto a fresh Story workspace, preserve its exact eight-path source/config/docs/test scope, integrate it to trunk, then synchronize supported global and source-repository local runtimes.

## Acceptance Criteria
Source and installed managed Codex configs contain no profiles.fast and select service_tier=default; repeat setup preserves projects, notice, non-fast custom profiles, project-local Codex config, and primary-session policy; malformed installed TOML fails without clobbering; full Go tests, vet, doctor, and verify pass; the Story candidate is reviewed and integrated to trunk.
