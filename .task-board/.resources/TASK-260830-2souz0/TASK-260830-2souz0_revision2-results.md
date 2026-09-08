# TASK-260830-2souz0 developer rework outcome — revision 2

## Handoff

Revision 1's deterministic delete-recovery substitution gap is repaired and the candidate is ready for independent review.
No live model, Pi runtime, daemon, service endpoint, readiness URL, or broker socket was launched, contacted, or inspected.

## Authority and scope

- Before production edits, selected Story base, workspace `HEAD`, fetched `origin/main`, `FETCH_HEAD`, and direct remote `main` all resolved to `3295c7da7151de128f176cf7560a57d54c8f6c0d`.
- Preserved stale-Story patch input was independently verified at 184182 bytes and SHA-256 `5c162f2050c288039651599e5f8c7fc252e1455412e03d145a7441d5faece553`; it remained non-authorizing review input.
- Revision 1 immutable candidate tree was `db78f48c1722b18cc1ee34e60f15b706d37303f9`.
- An alternate index seeded from that tree proves the revision 2 rework changes only `LOGBOOK.md`, `pi_session_log.go`, and `pi_lifecycle_authority_test.go`; resulting candidate tree is `0dc5c8912792291a2e9a374161ddcf011e5fe59b`.

## Production repair

- One `revalidatePiLifecycleDeleteChild` helper now owns both initial child admission and the repeated pre-unlink proof.
- After tombstone revalidation and the deterministic substitution scheduling boundary, recovery repeats held-descriptor plus `Fstatat(..., AT_SYMLINK_NOFOLLOW)` path checks for exact type, mode, trusted effective UID, link count, device, and inode.
- A mismatch or failed read preserves the original and replacement, leaves the odd generation unresolved, and returns typed `lifecycle_log_evidence_unknown` before mutation.
- Final tombstone `AT_REMOVEDIR` failures are also wrapped as typed unknown instead of leaking raw `ENOTEMPTY`.
- `LOGBOOK.md` records why revision 1's “immediately before every unlink” claim was incomplete and the exact production fix.

## Negative evidence

All commands were direct standalone processes; no gate output was piped through `tee`.

| Command / gate | Exit | Evidence |
| --- | ---: | --- |
| Initial red command attempt | 1 | Shell failed before `go test` because the redirect path had one excess `..`; excluded from test evidence. |
| Exact-gap production negative before fix | 1 | Replacement was deleted and recovery leaked raw `ENOTEMPTY`; `red-delete-unlink-gap-02.log`. |
| Final-tombstone typed-error negative before fix | 1 | Late evidence caused raw `ENOTEMPTY`; `red-delete-final-rmdir-03.log`. |
| Both new negatives after fix | 0 | Replacement/original preserved and failures typed unknown; `green-delete-boundary-04.log`. |
| Focused close/delete/generation/crash/soak authority suite | 0 | Package `2.351s`; `focused-authority-05.log`. |
| Focused lifecycle plus current pressure suite under `-race` | 0 | Package `7.273s`; `focused-race-06.log`. |

## Repository validation

| Command / gate | Exit | Result |
| --- | ---: | --- |
| `go test ./... -count=1` | 1 | First parallel aggregate: only the known readiness timing fixture missed its marker under suite load; all other packages green. Preserved in `go-test-all-07.log`. |
| Exact readiness fixture, `-count=3` | 1 | First repetition red, next two green; preserved honestly in `readiness-timing-rerun-08.log`. |
| Exact readiness fixture, uncached standalone | 0 | Both subtests green, package `5.498s`; `readiness-timing-isolated-09.log`. |
| `go test ./... -count=1` rerun | 1 | Same sole timing fixture under parallel package load; all other packages green. Preserved in `go-test-all-rerun-10.log`. |
| Uncached `internal/infra` excluding exactly that fixture | 0 | Package `231.231s`; `infra-skip-timing-11.log`. |
| `go test -p 1 ./... -count=1` | 0 | Full serial repository suite green: root `125.228s`, infra `204.534s`, attachments `0.883s`, modelharness `0.729s`; `go-test-all-serial-18.log`. |
| `go vet ./...` | 0 | Clean. |
| `gofmt -l .` | 0 | Empty output. |
| `git diff --check` | 0 | Empty output. |
| Darwin `go build ./...` | 0 | Clean. |
| Linux/amd64 `CGO_ENABLED=0 go build ./...` | 0 | Clean. |
| Windows/amd64 `CGO_ENABLED=0 go build ./...` | 0 | Clean. |

The two parallel aggregate failures and mixed `-count=3` run remain failures; the later serial full-suite exit 0 and exact standalone exit 0 are separate evidence, not relabeling.

## No-live-runtime evidence

Only repository inspection, Git authority reads, board operations, Go compilation/tests/vet, formatting, and filesystem-backed test fixtures were used. No `pi-infra`, `qwen-infra`, model checker, runtime executable, HTTP readiness endpoint, live daemon, live broker, or service command was invoked.
