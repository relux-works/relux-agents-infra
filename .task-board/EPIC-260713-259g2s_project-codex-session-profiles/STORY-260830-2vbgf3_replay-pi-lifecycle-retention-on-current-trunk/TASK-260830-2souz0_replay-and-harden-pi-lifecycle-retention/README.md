# TASK-260830-2souz0: replay-and-harden-pi-lifecycle-retention

## Description
Reconcile the preserved retention revision with current protected trunk and implement the reviewer-proven close, delete-recovery, and generation-attestation authority fixes through the production lifecycle path.

## Scope
Current-trunk replay of Pi lifecycle aggregate retention, strict filesystem/generation recovery, status and continuation evidence, pressure composition, tests, docs and logbook. Do not reuse the stale Story branch as an integration base and do not contact a live runtime.

## Acceptance Criteria
1. Fresh selected base equals fetched protected main and the preserved patch is treated only as review input. 2. Exact completed close evidence binds ClosedAt to the issued odd generation StartedAt; forged timestamps preserve unknown. 3. Tombstone and every child are descriptor-relatively revalidated for mode, trusted UID, device, inode, type and substitution immediately before bounded unlink. 4. Even and odd generation records enforce exact state/kind-specific field contracts; malformed residual authority is unknown to status and next writers. 5. Reviewer probes plus stronger substitution/narrowing/status/next-writer negatives fail before and pass after production fixes. 6. Focused race, uncached suites, vet, format, Darwin build and Linux/Windows compile gates pass with outcome evidence and independent review.
