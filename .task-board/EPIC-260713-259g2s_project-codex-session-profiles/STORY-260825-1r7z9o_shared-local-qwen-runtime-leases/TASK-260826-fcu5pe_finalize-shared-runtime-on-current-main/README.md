# TASK-260826-fcu5pe: finalize-shared-runtime-on-current-main

## Description
Reconcile the accepted shared-runtime broker Story branch with the current main branch, resolve overlapping agents-infra Pi/runtime changes, and publish the final Story candidate for integration.

## Scope
Use the accepted broker implementation and reviewer rev2 verdict as fixed inputs. Bring the Story candidate onto current main without weakening the 13-gate attestation chain, runtime-launch authorization, shared lease semantics, mutation calibration, or CLI contract. Preserve unrelated main changes. Run the configured landing validation plus focused shared-runtime race and production-entry mutant suites. Do not add the deferred task-board Pi adapter or change standalone yolo policy.

## Acceptance Criteria
The Story candidate contains the accepted broker behavior composed with current main; all overlaps are resolved deliberately; focused, race, mutation, build, vet, formatting, and configured landing validation pass on the reconciled tree; a story_final Change Request is published with evidence suitable for independent review and integration.
