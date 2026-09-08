# TASK-260830-2souz0 review verdict — Change Request revision 2

## Verdict

**Accepted.** Revision 2 repairs the deterministic revision-1 delete-recovery substitution gap and typed-error leak. The exact current-trunk candidate satisfies the close-evidence, delete-recovery, strict-generation, status, and next-writer acceptance boundaries through the production lifecycle path.

Reviewed Change Request: `CR-TASK-260830-2souz0-2`, revision `2`.

- Base OID: `3295c7da7151de128f176cf7560a57d54c8f6c0d`
- Candidate tree OID: `0dc5c8912792291a2e9a374161ddcf011e5fe59b`
- Patch SHA-256: `b8d853144ec339b6de2c86c269b861bf92b6e33c3188b16c6964555607f3d99f`
- Repository delta: present, 21 paths
- Reviewer run: `RUN-260830-5b02e4`
- Spawn goal: none; this reviewer run is not goal-bound.

## Authority and provenance

- A fresh `git fetch origin main` completed successfully. Workspace `HEAD`, fetched `origin/main`, `FETCH_HEAD`, and direct `git ls-remote origin refs/heads/main` all equal the CR base `3295c7da7151de128f176cf7560a57d54c8f6c0d`.
- Board workspace evidence independently reports selected, initial, current, and checkpoint base plus branch tip at the same OID. Local `main` remains stale at `5c9b4e4f7a88e1eb937b80851af522e4fa4b066f` and was not used as authority.
- An isolated alternate index seeded from the exact base and populated with the CR's explicit 21-path scope wrote tree `0dc5c8912792291a2e9a374161ddcf011e5fe59b`, proving the reviewed worktree bytes equal the immutable candidate. An initial broad-path snapshot attempt was refused because `.temp` is ignored; it produced no equality claim and was replaced by the successful explicit-scope proof.
- `git diff --binary <base> <candidate> | shasum -a 256` independently reproduced the board patch digest exactly.
- The preserved stale-Story patch was materialized and independently verified at 184182 bytes and SHA-256 `5c162f2050c288039651599e5f8c7fc252e1455412e03d145a7441d5faece553`. Its patch shape, preserved review verdict, negative probes, base-authority blocker, and producer result were inspected only as non-authorizing historical input.

## Gate attacks and findings

### Exact close authority

Production call site: `openPiSessionLog` -> `recoverPiLifecycle` -> `recoverPiLifecycleClose`.

Recovery accepts an already-published close only when `record.ClosedAt == generation.StartedAt`; an unrelated well-formed timestamp returns typed `lifecycle_log_evidence_unknown`. The positive crash phase now publishes the operation's issued timestamp. The forged-close production negative passes uncached.

### Delete recovery authority

Production call site: `openPiSessionLog` -> `recoverPiLifecycle` -> `recoverPiLifecycleDelete`.

The odd delete generation carries exact tombstone and child identities. Tombstone recovery proves mode, trusted effective UID, device, inode, type, and stored identity through held descriptors. Every recognized child is initially validated and then revalidated again, after tombstone revalidation and the deterministic scheduling boundary, by comparing its held descriptor and `Fstatat(..., AT_SYMLINK_NOFOLLOW)` path identity for exact type, mode, UID, link count, device, and inode immediately before the bounded name-based unlink.

Revision-1 red evidence was independently inspected: the exact-gap replacement was deleted and raw `ENOTEMPTY` leaked. Revision 2 moves the shared child authority proof to the unlink boundary. The exact same production-entry shape now preserves both original and replacement evidence, leaves the generation unresolved, and returns typed unknown. Narrowed tombstone, narrowed child mode, replaced child inode, substituted symlink type, exact-gap substitution, and late foreign evidence are all refused. Final tombstone removal failures are wrapped as typed unknown.

Negative shapes exercised: **bypass path around the check**, **forged or self-minted evidence**, and **failure to read is not absence**.

### Strict generation and status/next-writer refusal

Production call sites: `PiLifecycleStatus` -> `readPiLifecycleGenerationPair` and `openPiSessionLog` -> `recoverPiLifecycle`.

Even generations reject every operation-only field, including `StartedAt`, counters, append bytes, names, operation metadata, and delete authority. Odd generations require canonical timestamps, operation IDs, entry/staging grammar, parity, and exact create/append/close/delete field contracts; delete identities must be trusted and exact. The production negatives prove malformed even and odd records are unknown to both status and the next writer, with `WithinPolicy` and `SoakReady` remaining false.

Caller inspection found the strict generation validator and delete-recovery guard on the production lifecycle path; no second production implementation bypasses the decision.

## Validation

Reviewer-rerun commands:

| Gate | Exit | Result |
| --- | ---: | --- |
| Focused forged-close, narrowed/substituted delete, exact-gap, final-rmdir, malformed even/odd, and crash-close suite with `-count=1 -v` | 0 | Eight top-level tests plus subtests passed; package `5.611s` |
| Focused lifecycle plus current pressure suite under `go test -race -count=1` | 0 | Package `27.994s` |
| `go vet ./...` | 0 | Clean |
| Darwin `go build ./...` | 0 | Clean |
| Linux/amd64 `CGO_ENABLED=0 go build ./...` | 0 | Clean |
| Windows/amd64 `CGO_ENABLED=0 go build ./...` | 0 | Clean |
| `gofmt -l tools/agents-infra` | 0 | Empty output |
| `git diff --check <base> <candidate>` | 0 | Clean |

Already-attached exact-candidate evidence inspected rather than redundantly rerun:

- CR publication validation: uncached `go test ./... -count=1` passed, including root `164.069s` and `internal/infra` `286.272s`; `go vet ./...` passed.
- Producer serial no-skip repository suite: `go test -p 1 ./... -count=1` passed, including root `125.228s`, `internal/infra` `204.534s`, attachments `0.883s`, and modelharness `0.729s`.
- Earlier parallel aggregate timing failures remain preserved as failures. They are not relabeled; the later CR-bound parallel publication suite and serial full suite are separate green evidence.

No live Pi runtime, model, daemon, service endpoint, readiness URL, broker socket, or external runtime process was launched, contacted, or inspected during this review.

## Handoff

Accept Change Request revision 2 and park `TASK-260830-2souz0` at `to-review` for the commit-owning Orchestrator to checkpoint. This reviewer supplies no `commit_ack`.
