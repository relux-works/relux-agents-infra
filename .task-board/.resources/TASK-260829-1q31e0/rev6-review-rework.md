# TASK-260829-1q31e0 revision 6 review rework

Implement revision 6 from the immutable revision 5 candidate on exact base `6d051f54440d36e3ca3d132f8d9d1e78d46289de`.

## Blocking finding to close

Revision 5 treats a deterministic same-UID-mintable hard-link pair as launcher provenance. A foreign caller can construct the public filename, mode, link count, inode relation, and ownership-namespace link, then the real `openPiSessionLogAt` pruning path deletes that foreign file.

## Required implementation and evidence

- Replace the caller-mintable evidence with deletion authority that an ordinary same-UID foreign caller cannot construct, or conservatively preserve files whenever launcher provenance cannot be established.
- Keep the production `openPiSessionLogAt` standalone/shared entry path in the proof.
- Add a negative test that constructs every public shape accepted by revision 5 and requires preservation/refusal.
- Add a narrowing mutant against the replacement authority. A test that only passes when ownership checks are deleted is insufficient.
- Preserve active-file safety, aggregate count/byte/age bounds, deterministic ordering, unknown/fail-closed status, concurrency behavior, and the eight-week soak proof.
- Prove the reviewer attack red before the fix and green after it, then run focused race and concurrency tests, `go test ./... -count=1`, `go vet ./...`, `go build ./...`, formatting/diff integrity, and supported cross-platform compile gates.
- Reconcile README/SKILL claims and append a corrective LOGBOOK entry. Do not contact or mutate a live model runtime, service, socket, endpoint, or user-owned model state.
- Publish a new immutable Change Request revision; do not overwrite revision 5 evidence.

