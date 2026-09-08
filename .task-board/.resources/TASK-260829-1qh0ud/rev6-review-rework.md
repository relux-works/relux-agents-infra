# TASK-260829-1qh0ud revision 6 review rework

Implement revision 6 from the immutable revision 5 candidate on exact base `6d051f54440d36e3ca3d132f8d9d1e78d46289de`.

## Blocking findings to close

1. Record-derived status currently combines caller-configured resource policy with a different record-effective sharing policy. A configured-disabled caller can therefore publish `sharing.effective=provider` while resources claim `disabled`, `resource_pressure_disabled`, and `not-enforced`.
2. The production policy mismatch table does not prove equality for independently variable fields. A mutant comparing only pressure threshold survives because the existing case changes pressure and recovery thresholds together.

## Required implementation and evidence

- Bind record-derived resource status to coherent provenance. Never synthesize effective enforcement from the caller policy when the persisted record says otherwise.
- Preserve `record-derived-unverified` and explicit unknown observation semantics. Do not publish disabled/not-enforced unless that fact is established for the same policy surface.
- Add production-entry `SharedRuntimeStatusReport` cases for both configured/effective mismatch directions and a pre-extension record without sharing data.
- Add real `handleConnection -> acquireLease` mismatch cases that vary independently every policy field; at minimum, recovery-threshold-only mismatch must refuse before observation or lease.
- Prove the reviewer attacks red before the fix and green after it, then run focused race tests, `go test ./... -count=1`, `go vet ./...`, `go build ./...`, formatting/diff integrity, and supported cross-platform compile gates.
- Update README, SKILL, and LOGBOOK truthfully. Do not contact or mutate a live model runtime, service, socket, endpoint, or user-owned model state.
- Publish a new immutable Change Request revision; do not overwrite revision 5 evidence.

