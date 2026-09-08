# TASK-260830-tvy8q5: ship-legacy-retirement-and-multiweek-soak

## Description
Finish the explicit bounded legacy lifecycle-log retirement operator surface and prove the complete current-trunk retention plane across deterministic hours, days, weeks, host restarts, crashes, leases, health loss, reload and pressure transitions.

## Scope
Non-launching legacy status pagination, dry-run and exact plan-hash/confirm-delete flow, odd/even legacy fencing, per-candidate revalidation, external-only results, deterministic soak simulations, consumer/status documentation and final Story release evidence. No automatic legacy mutation or live runtime contact.

## Acceptance Criteria
1. Legacy inspection and retirement are explicit CLI operations with bounded dry-run, stable full-plan hash, exact confirmation, generation fencing, resumable unlink and immediate pre-delete authority revalidation. 2. Automatic setup/launch/status paths never mutate legacy evidence. 3. Fresh complete scans alone publish within_policy and soak_ready after retirement. 4. Deterministic simulations cover hours, days and at least eight weeks including crash loops, stale leases, corruption, backend/reload/host restart and pressure composition without wall-clock sleeps. 5. Full repository and cross-platform gates, independent review, PR merge, installed-runtime parity and no-live-runtime evidence pass.
