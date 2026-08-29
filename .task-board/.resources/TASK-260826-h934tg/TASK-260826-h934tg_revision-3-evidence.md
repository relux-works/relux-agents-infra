# TASK-260826-h934tg — Revision 3 fresh-base reconciliation evidence

## Outcome

The accepted standalone Pi YOLO candidate is reconciled with current `main`
commit `355a156276080b6994080f8e9a767e7416a5b357` (`Keep Pi in the terminal
foreground`). The Story branch now has real current-main ancestry and preserves
both behavior sets.

During semantic review of the clean Git merge, one composition defect was found
and repaired: the new foreground-terminal helper was still called with caller
stdin for standalone launches. Standalone already left `cmd.Stdin` nil, but a
TTY caller could still cause `Foreground`/`Ctty` configuration. Both exclusive
and shared-runtime production paths now attach stdin and terminal foreground
only when `opts.Standalone == nil`; standalone retains closed stdin and its own
process group.

No task-board adapter, allowlist widening, sudo/root path, or unrelated feature
was added.

## Fresh-base ancestry

- Accepted standalone checkpoint: `8f81371d93c75552580bb1530281ea5627f429a1`
- Prior reconciliation merge: `3a52ec762b93149b6db541612f28bf1a6ccef5ed`
- Required current main: `355a156276080b6994080f8e9a767e7416a5b357`
- Fresh merge leaf: `eaefc6f4f41a9e2c2c0f0464a8fb632ab13910ff`
- Fresh merge parents: `3a52ec762b93149b6db541612f28bf1a6ccef5ed`
  and `355a156276080b6994080f8e9a767e7416a5b357`
- `git merge-base --is-ancestor main HEAD`: exit `0`
- `git rev-list --left-right --count main...HEAD`: `0 3`
- Final current-main identity check: local `main` remained exactly `355a156...`

The accepted implementation task `TASK-260826-3i0lwe` was already checkpointed
as CR revision 2 before this merge, as shown by `task-board worktree status`.

## Reconciliation paths

Git's `ort` merge exited `0`. It auto-merged overlapping `README.md`, `SKILL.md`,
`pi_launch_posix.go`, `pi_platform_windows.go`, and
`pi_shared_client_darwin.go` without textual conflicts. Therefore the literal
hand-resolved conflict path set is empty.

Post-merge semantic reconciliation was intentionally limited to:

- `tools/agents-infra/internal/infra/pi_launch_posix.go`
- `tools/agents-infra/internal/infra/pi_shared_client_darwin.go`
- `tools/agents-infra/internal/infra/pi_standalone_shared_test.go`
- `LOGBOOK.md`

`README.md` and `SKILL.md` retain the accepted standalone contracts and current
main's foreground/session-log documentation. Their existing operator contract
tests pass in the full suite. Historical logbook entry `1411` remains unchanged;
new append-only entry `1528` records the terminal-ownership regression and fix.

## Negative evidence

Production call site:
`RunPi` -> exclusive/shared Pi launch -> terminal configuration.

`TestRunPiStandaloneExclusiveWorkerClosesReadableStdin` now supplies readable
stdin and replaces the terminal probe with a forced-positive probe. A valid
standalone launch must observe stdin EOF and invoke that probe zero times.

The production file was copied before mutation. Its SHA-256 was
`84a099004458566137bb9fe4b33f2944533b910c86aef7b47bcd85bab1df1e54`.
The bounded mutant widened the interactive branch from
`opts.Standalone == nil` to `opts.Standalone == nil || opts.Stdin != nil`.
The witness exited `1` because the standalone child attempted terminal
foreground attachment (`operation not supported by device`). Production was
restored from the copy; `cmp` exited `0`, the SHA-256 returned exactly to the
value above, and the restored witness exited `0`.

The previously accepted authorization witness
`TestRunPiStandaloneNeverInheritsPrimarySessionProjectTrust` remains present.
It drives the real `RunPi` entry point with primary `yolo_mode = true`, requires
exactly one standalone-owned `--no-approve`, and rejects inherited
`--approve`/`-a`. Existing shared-runtime attestation and authorization tests
exercise delete/narrowing, absent evidence, wrong identity, and malformed-shape
refusals at production entry points.

## Scope proof

The immutable revision-2 candidate tree is
`7cfbbd938587b2054e112440678d6bb317d834b4`. The exact-scope gate compares the
current candidate against that tree, excludes `.task-board/**` as required by
Change Request construction, and requires equality with the union of:

1. the 13 paths changed by `fd80bd8..355a156`;
2. additive `LOGBOOK.md`; and
3. the strengthened standalone production witness.

The first scope invocation exited `1` because its checker incorrectly used the
accepted implementation checkpoint and included the linked worktree's board
checkout artifact. This was a checker-baseline defect, not accepted as product
evidence. The corrected immutable-candidate invocation exited `0` and reported
`scope exact: 15 paths`. No unrelated repository path is present.

## Direct validation

Every command ran as a standalone foreground process without `tee` or a
status-hiding pipe. Exit codes are the real process results.

| Command | Exit | Evidence |
| --- | ---: | --- |
| `git merge --no-edit main` | 0 | Fresh merge leaf created |
| Initial focused terminal/session-log tests | 0 | Pre-semantic-fix merge smoke |
| Initial focused standalone tests | 0 | Pre-semantic-fix merge smoke |
| Initial focused shared-runtime/negative tests | 0 | Pre-semantic-fix merge smoke |
| Initial `go test ./... -count=1` | 0 | Pre-semantic-fix configured suite |
| Initial `go vet ./...` | 0 | Pre-semantic-fix configured vet |
| Initial native and four cross builds | 0 each | Pre-semantic-fix compile evidence |
| Initial exact-scope checker | 1 | Rejected: wrong baseline included board artifact |
| Corrected pre-fix exact-scope checker | 0 | Immutable revision-2 baseline |
| Strengthened terminal/standalone witness, baseline | 0 | Production guard passed |
| Same witness under narrowed terminal guard | 1 | Expected red; mutant killed |
| Same witness after exact restoration | 0 | Restored production passed |
| Post-fix focused terminal/session-log tests | 0 | `focused-terminal-02.log` |
| Post-fix focused standalone tests | 0 | `focused-standalone-02.log` |
| Post-fix focused shared-runtime/negative tests | 0 | `focused-shared-runtime-02.log` |
| Post-fix `go test ./... -count=1` | 0 | root `80.133s`; attachments `1.038s`; infra `127.922s` |
| Post-fix `go vet ./...` | 0 | Configured landing vet |
| Post-fix `go build ./...` | 0 | Native Darwin/arm64 |
| Post-fix `GOOS=darwin GOARCH=amd64 go build ./...` | 0 | Cross build |
| Post-fix `GOOS=linux GOARCH=amd64 go build ./...` | 0 | Cross build |
| Post-fix `GOOS=linux GOARCH=arm64 go build ./...` | 0 | Cross build |
| Post-fix `GOOS=windows GOARCH=amd64 go build ./...` | 0 | Cross build |
| `gofmt -w` on the three touched Go files | 0 | Formatter applied |
| `gofmt -l` over all project Go files + empty-output assertion | 0 / 0 | Formatting clean |
| `git diff --check` | 0 | Diff hygiene clean |
| Corrected final exact-scope checker | 0 | `scope exact: 15 paths` |
| Current-main identity assertion | 0 | `main == 355a156...` |
| `git merge-base --is-ancestor main HEAD` | 0 | Real main ancestry |

Revision 3 is ready for independent review.
