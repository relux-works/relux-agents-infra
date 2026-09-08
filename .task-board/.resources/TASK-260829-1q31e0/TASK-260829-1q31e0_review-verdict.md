# TASK-260829-1q31e0 independent review verdict

## Verdict

**Changes requested** for `CR-TASK-260829-1q31e0-2` revision 2.

Route: `to-dev` after dependency `BUG-260829-ajb7n7` reaches `done`. This is ordinary repository-composition rework, not Stop-The-Line.

- Retention base: `675f77ed63376320ed1213f46f9462a299c0abaf`
- Retention candidate tree: `bcdf1c032276742220b3ae2cb27d51d470796ead`
- Retention patch SHA-256: `ec96ceb8ab23a78e4074b96210c9de538ade2d6e1aaaa6cc5cf00cd745bd8bab`
- Accepted blocker CR: `CR-BUG-260829-ajb7n7-1` revision 1
- Reviewer run: `RUN-260829-1e5e00`

## Blocking finding

### P1 — Immutable revision 2 conflicts with the accepted refreshed-main prerequisite

The lifecycle-retention implementation itself passes review, but the immutable candidate cannot be accepted against the refreshed landing order requested by the orchestrator.

`BUG-260829-ajb7n7` fixes the pre-existing root test-output deadlock found during this review. Its accepted patch and retention revision 2 share two paths:

- `LOGBOOK.md`
- `tools/agents-infra/main_test.go`

A reviewer-owned three-way composition probe built two commits from exact base `675f77ed`: one from the accepted bug patch and one from the immutable retention patch. `git merge-tree --write-tree <bug> <retention>` auto-merged `tools/agents-infra/main_test.go` but returned a content conflict in `LOGBOOK.md`:

```text
Auto-merging LOGBOOK.md
CONFLICT (content): Merge conflict in LOGBOOK.md
Auto-merging tools/agents-infra/main_test.go
```

Both candidates insert a new `2026-08-29` entry at the same location. Accepting revision 2 before checking this would attest a candidate that does not compose with the accepted prerequisite/refreshed main. The correct recovery is a new immutable revision, not an acceptance retry.

## Required rework

1. Let `BUG-260829-ajb7n7` integrate to refreshed main.
2. Refresh the retention Story worktree and republish revision 3 from that exact main.
3. Preserve both LOGBOOK entries: the lifecycle aggregate decision and the root capture deadlock fix.
4. Preserve the bug's concurrent `capturePipe` implementation/tests while retaining the lifecycle table added to root test configuration.
5. Run the configured full `go test ./... -count=1` suite on the refreshed candidate. The accepted bug fix makes the previously deadlocking root tests executable, so a complete green result is now required rather than a base-control exception.
6. Re-run the lifecycle focused/race suite and the aggregate-narrowing mutant, then publish a new CR revision for independent review.

## Functional retention evidence retained

No functional defect was found in the lifecycle-retention implementation:

- all 17 reviewed path blobs match candidate tree `bcdf1c...`;
- the patch digest reproduces exactly;
- focused lifecycle/config/status tests pass uncached;
- eight simulated weeks finish at `managed_count=5/5`, `managed_bytes=405/420`, `expired_count=0`, one foreign file preserved, `within_policy=true`;
- focused `-race`, `go build ./...`, `go vet ./...`, formatting, and diff checks pass;
- a reviewer narrowing mutant that bypasses `runs/<hash>/logs/` fails the production-shaped aggregate test with expected exit 1;
- active overflow, unreadable status, identity replacement, foreign entries, and whole-record byte refusal pass.

These results should be replayed after reparenting because repository composition, not retention semantics, blocks revision 2.

No live Pi/model process, service, socket, or endpoint was used.
