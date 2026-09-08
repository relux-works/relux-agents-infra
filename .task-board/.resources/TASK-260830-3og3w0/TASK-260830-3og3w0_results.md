# TASK-260830-3og3w0 Revision 2 Implementation Evidence

## Review rework

- Addressed reviewer finding F1 from `CR-TASK-260830-3og3w0-1` revision 1.
- Added `TestPiLifecycleDeleteRecoveryPreservesModeCorrectTombstoneSubstitution` at the production entry `openPiSessionLog -> recoverPiLifecycle -> recoverPiLifecycleDelete -> revalidatePiLifecycleDeleteDirectory`.
- The test issues a real odd delete generation, substitutes the tombstone with a different inode while preserving mode, effective UID, and the three stored child identities, and requires typed unknown plus preservation of the replacement evidence.
- The production gate was already exact; no production weakening or compensating special case was added.
- Added the root cause and discriminating proof to `LOGBOOK.md` at `2026-08-30 1138`.

## Base and accepted patch authority

- Before the accepted patch was applied, `git fetch origin main` exited 0 and the managed Story `selected_base_oid`, workspace `HEAD`, and freshly fetched `origin/main` were all `b78498bf98c05175db10bb341aee621e53de4881` (`0/0` ahead/behind). The board note recorded this before patch bytes were applied.
- Accepted input remains exactly 219471 bytes with SHA-256 `b8d853144ec339b6de2c86c269b861bf92b6e33c3188b16c6964555607f3d99f`.
- All 21 accepted paths remain classified in `TASK-260830-3og3w0_path-classification.tsv`: 18 `applied_exact`, 3 `semantically_reconciled`, 0 `already_upstream`.
- `LOGBOOK.md`, `README.md`, and `pi_lifecycle_authority_test.go` are semantic reconciliations that retain current Story material while preserving the accepted retention contract and revision-2 review rework.
- A fresh fetch during this rework resolved `origin/main` to `fe3818209c9861fcafa1f2e68efe078cc0f96f30`, three commits after the already-applied candidate base. The advance is path-disjoint from all 21 candidate paths. No patch was reapplied, and the managed Story branch was not switched, rebased, or merged; Change Request integration owns re-parenting and exact-tree revalidation.
- Current 21-path binary candidate patch: 227512 bytes, SHA-256 `f197c3e09c9d83789877de6466ba25733538e0f3e80ac79d5548ce7adbdbb872`.

## Negative evidence

| Probe | Exit | Result | Evidence |
| --- | ---: | --- | --- |
| New test against exact production gate | 0 | Refused mode-correct tombstone substitution and preserved all replacement evidence | `tombstone-substitution-green-01.log` |
| Gate narrowed to `expected != nil` plus generic validation | 1 | Expected red: replacement admitted, `next=true`, `err=nil` | `tombstone-substitution-mutant-red-01.log` and board resource `TASK-260830-3og3w0_tombstone-mutant.log` |
| New test after exact identity restoration | 0 | Exact device/inode/mode/UID authority restored | `tombstone-substitution-green-restored-01.log` |
| Named forged-close, tombstone/child substitution, late-evidence, malformed even/odd status and next-writer negatives | 0 | All production-entry negatives passed uncached | `authority-negatives-rev2-01.log` |

The expected-red mutant was temporary and was restored before every green gate. `git diff` shows no residual production mutation.

## Validation

Every command ran directly as a standalone process; exit codes are the real command exits.

| Gate | Exit | Evidence |
| --- | ---: | --- |
| `go test -race ./internal/infra -run '(PiLifecycle|Pressure)' -count=1` | 0 | `focused-race-rev2-01.log` |
| `go test . -count=1` | 0 | `full-root-rev2-01.log` |
| `go test ./internal/infra -count=1` | 0 | `full-infra-rev2-01.log` |
| `go test ./cmd/model-harness ./internal/attachments ./internal/modelharness -count=1` | 0 | `full-aux-rev2-01.log` |
| `go vet ./...` | 0 | `go-vet-rev2-01.log` |
| `gofmt -l tools/agents-infra/internal/infra/*.go tools/agents-infra/*.go` | 0 and empty output | `gofmt-rev2-01.log` |
| `git diff HEAD --check` | 0 | `git-diff-check-rev2-01.log` |
| `GOOS=darwin GOARCH=arm64 go build ./...` | 0 | `go-build-darwin-arm64-rev2-01.log` |
| `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ./...` | 0 | `go-build-linux-amd64-rev2-01.log` |
| `CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ./...` | 0 | `go-build-windows-amd64-rev2-01.log` |

All changed behavior was exercised with task-local static filesystem fixtures. This run did not contact, inspect, start, stop, signal, or mutate a live runtime, service, or socket.

## Handoff

The accepted retention patch and current setup, launch, pressure, instruction, and test surfaces remain reconciled. Strict close evidence, descriptor-relative delete, tombstone and child identity, even/odd generation validation, status refusal, and next-writer refusal remain in force. Revision 2 is ready for an independent reviewer to attack the exact stored tombstone identity gate and the complete authority slice.
