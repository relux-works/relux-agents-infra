# TASK-260830-u8nd0b: pin-stable-agents-management-release-and-validate-adapter

## Description
After the exact adapter candidate is independently accepted and agents-management publishes the stable release containing commit 046baef, replace the development pseudo-version with that immutable tag and validate the cumulative consumer surface.

## Scope
go.mod/go.sum dependency pin, exact public API compile, adapter/static-fake integration gates, docs and release evidence only. No adapter redesign and no live runtime/model/process/service/socket/endpoint/user-config access.

## Acceptance Criteria
1. Blocked until TASK-260830-y6infr is accepted and checkpointed and the upstream stable tag contains exact commit 046baef. 2. go.mod/go.sum contain the immutable tag and no local replace or pseudo-version for agents-management. 3. Exact observer/Pi turn external compile plus adapter production-entry, full, race, vet, build, cross-platform, mutation and no-live-runtime gates pass. 4. Independent review accepts the final Story CR. 5. Canonical PR merges before any local-Qwen activation.
