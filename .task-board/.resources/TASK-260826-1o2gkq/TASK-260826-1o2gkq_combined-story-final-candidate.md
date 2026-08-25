# TASK-260826-1o2gkq combined Story candidate evidence

## Outcome

The managed Story candidate now carries all three required layers without
reopening accepted behavior:

1. accepted shared-runtime revision 5 at commit `91adc7328d6a122fbbbb40f42a1d9b6aad5f2ac0`
   and tree `40a83fe6f3b1544494969edc861f3fe23ffc4757`;
2. the fresh-main merge `c40f0c26e071a9b466f1b856bebe91f19fb7390b`;
3. the accepted identity/version witness checkpoint
   `52270bf613bf0ba287f6231c921780cab1a51906`.

The only new worktree delta is the reviewer-requested correction to the
`LOGBOOK.md` root-cause sentence. It now says that three gates had prior
wrong-device witnesses and that `sharedRuntimeBrokerCandidates` had neither
witness. No production file, test file, standalone yolo path, or other
documentation changed in this run.

The completion-hook candidate tree, reconstructed with an alternate Git index,
is `281a72e1b96ca8c08ca62ea54f6f2d2557c1e33d`.

## Ancestry and blob identity

Every command below ran as a standalone foreground process and reported its
real exit code.

| Command / proof | Exit | Result |
| --- | ---: | --- |
| `git merge-base --is-ancestor main HEAD` | 0 | Current `main` is a real Story ancestor. |
| `git rev-list --count HEAD..main` | 0 | Output `0`. |
| `git rev-parse c40f0c2^1 c40f0c2^2` | 0 | Parents are accepted rev5 `91adc73...` and current main `e70f953...`. |
| `git rev-parse 52270bf^1` | 0 | Witness checkpoint is directly atop `c40f0c2...`. |
| `git diff --name-status HEAD -- . ':(exclude).task-board/**'` | 0 | Only `M LOGBOOK.md`. |
| `git diff --quiet HEAD -- <four witness test paths>` | 0 | All checkpointed witness test bytes remain identical to `52270bf`. |
| `git diff --name-status 91adc73 HEAD` excluding `LOGBOOK.md`, `.task-board/**`, and the four accepted witness paths | 0 | Empty output: every other product blob remains identical to accepted rev5. |
| Same rev5 comparison without exclusions except `.task-board/**` | 0 | Exactly `LOGBOOK.md` plus the four accepted witness test paths differ. |
| `git diff --numstat HEAD -- LOGBOOK.md` | 0 | Exactly `1` insertion / `1` deletion. |
| `git diff HEAD --check -- . ':(exclude).task-board/**'` before and after validation | 0 / 0 | No whitespace errors. |
| `git ls-files --others --exclude-standard` after validation | 0 | Empty output. |
| Alternate-index `git write-tree` before and after validation | 0 / 0 | Stable tree `281a72e1...`. |

The visible `MM` status on `LOGBOOK.md` and four witness files is intentional
managed-checkpoint state. The worktree's four test blobs equal `HEAD`; the
managed index still contains the prior candidate snapshot. This run did not
stage, reset, or overwrite that index. Change Request construction uses its own
alternate index and snapshots the worktree bytes.

## Validation run in this developer process

Project platform is the Darwin/arm64 Go CLI module under
`tools/agents-infra`. The configured landing suite in
`task-board.config.json` is `go test ./... -count=1` followed by
`go vet ./...`; both exact commands ran.

| Gate | Exit | Result |
| --- | ---: | --- |
| Focused shared runtime | 0 | `go test ./internal/infra -count=1 -run '^(TestSharedRuntime|TestSharedAuthorization|TestConnectAndAttestSharedRuntime|TestRunSharedRuntimeBroker|TestReclaimSharedRuntime)'`; `32.127s`. |
| Focused shared runtime race | 0 | Same mask with `go test -race`; `55.701s`. |
| Configured landing test | 0 | `go test ./... -count=1`; root `107.161s`, attachments `1.984s`, infra `156.895s`. |
| Configured landing vet | 0 | `go vet ./...`; empty output. |
| Build | 0 | `go build ./...`; empty output. |
| Formatting | 0 | `gofmt -l .`; empty output. |

The previously recorded load-sensitive launcher false red did not reproduce in
this run. This is one fresh green sample, not a claim that historical flakiness
never existed.

## Negative evidence binding

No gate or test behavior changed in this run, so production mutants were not
reapplied to the managed worktree. The checkpoint's independently accepted
review packet `TASK-260826-12laby_review-verdict-rev2.md` remains binding because
all four witness test blobs are byte-identical to checkpoint `52270bf`. That
review narrowed the real production call sites one at a time and recorded real
exit-1 kills for:

- executable identity at `pi_shared_broker_darwin.go:836`,
  `pi_shared_client_darwin.go:341`, and
  `pi_shared_operator_darwin.go:363` / `:470`;
- below-current protocol versions at the broker, client, and launcher gates;
- the fork-plus-exec replacement of `sharedRuntimeExecve`, killed by the
  exact-version launcher control while clean `unix.Exec` stayed green.

This developer process reran the unchanged focused suite and its race variant
through those production entry points. The LOGBOOK correction records the
reviewer's remaining historical-coverage qualification without overstating the
accepted fix.

## Handoff

After the developer handoff, the managed completion hook must publish a new
`story_final` Change Request bound to candidate tree `281a72e1...`. Independent
review must reconstruct that exact tree from the published patch and accept the
new revision before Story integration. Publication occurs after this spawned
process exits; there is no supported child-side manual publish command.
