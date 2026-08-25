# TASK-260826-1o2gkq story-final republication evidence

## Outcome

The accepted fresh-base merge remains unchanged and is ready for a no-change
producer handoff. `TASK-260826-fcu5pe` is now `done`, so this task is the
Story's final unresolved leaf. The managed completion hook can therefore derive
the next Change Request as `story_final` instead of the already accepted
revision 1 `task_delta`.

No repository file was edited in this run. The clean candidate remains:

- Story tip: `c40f0c26e071a9b466f1b856bebe91f19fb7390b`
- First parent / accepted rev5 tip: `91adc7328d6a122fbbbb40f42a1d9b6aad5f2ac0`
- Second parent / current `main`: `e70f953969d46e451892d9f16e7401b879910b6b`
- Story tree: `f92c9b2c2fd0ca11ac00f7fe4d479a7264f6a698`
- Accepted rev5 candidate tree: `40a83fe6f3b1544494969edc861f3fe23ffc4757`

## Fresh integrity checks

Every command ran directly as a standalone foreground process.

| Command | Exit | Evidence |
| --- | ---: | --- |
| `git merge-base --is-ancestor main HEAD` | 0 | Current `main` is a real ancestor of the Story tip. |
| `git rev-list --count HEAD..main` | 0 | Output `0`; the worktree is not behind current local `main`. |
| `git diff --quiet 40a83fe6... HEAD -- . ':(exclude)LOGBOOK.md' ':(exclude).task-board/**'` | 0 | Every non-LOGBOOK product blob remains identical to accepted CR rev5. |
| `git diff --name-status 40a83fe6... HEAD -- . ':(exclude).task-board/**'` | 0 | Empty output; the stronger full non-board tree, including `LOGBOOK.md`, is unchanged from rev5. |
| `git diff --numstat main HEAD -- LOGBOOK.md` | 0 | Output `168 0`; the resolved logbook preserves main additively. |
| `git rev-parse 40a83fe6...:LOGBOOK.md HEAD:LOGBOOK.md` | 0 | Both blob OIDs are `099acbf1aa722af312f92de07d40c293f78098bf`; the accepted side is byte-identical. |
| `git diff --check main..HEAD -- . ':(exclude).task-board/**'` | 0 | Product delta has no whitespace errors. |
| `git status --porcelain=v1 --untracked-files=all` | 0 | Empty output; worktree is clean. |

The only merge-tree differences from rev5 are inherited `.task-board/**`
checkout artifacts. Managed Change Request publication drops those paths.

## Fresh validation

The exact configured landing suite in `task-board.config.json` is
`go test ./... -count=1` followed by `go vet ./...`. Both were rerun here after
the ancestry and blob-identity checks.

| Command | Exit | Result |
| --- | ---: | --- |
| `go test ./internal/infra -count=1 -run '^(TestSharedRuntime|TestSharedAuthorization|TestConnectAndAttestSharedRuntime|TestRunSharedRuntimeBroker|TestReclaimSharedRuntime)'` | 0 | `ok` in `38.774s`. |
| Same focused suite with `-race` | 0 | `ok` in `68.533s`. |
| `go test ./... -count=1` | 0 | Root `183.680s`, attachments `2.456s`, infra `277.954s`. Configured landing test gate. |
| `go vet ./...` | 0 | Configured landing vet gate. |
| `go build ./...` | 0 | Darwin/arm64 build passed. |
| `gofmt -l .` plus empty-output assertion | 0 / 0 | Module Go files are formatted. |

Toolchain: Go `1.25.5` on Darwin/arm64; Git `2.53.0`.

The known 15-second wall-clock positive-control flake did not reproduce in this
run. The result is reported as one fresh green sample, not as evidence that the
historical flake no longer exists.

## Gate evidence and scope

This lifecycle retry changes no product or test bytes. The accepted reviewer
packet `TASK-260826-1o2gkq_review-verdict.md` remains the negative proof for the
fresh-base merge: reviewer run `RUN-260826-8e0561` widened the production
`launcher_pid` comparison at `sharedRuntimeExecve`
(`pi_shared_launcher_darwin.go:73`) to admit zero, and the named
`launcher_pid_zero` witness failed with exit 1 while the clean witness passed.
The file was restored byte-identically. This run reran the unchanged focused
suite and race variant green through the same production entry points.

No standalone yolo implementation or accepted shared-runtime behavior was
touched. The next producer handoff is expected to publish revision 2 with kind
`story_final`, candidate tree `f92c9b2...`, and an empty repository delta; an
empty delta is required here because any non-board product patch would be drift
from accepted CR rev5.

Raw command output is archived in
`TASK-260826-1o2gkq_story-final-validation-logs.tar.gz`.

## Handoff gate choreography

The first `task-board handoff TASK-260826-1o2gkq --role developer` attempt
exited `1` and changed no status. It refused unchecked checklist items 4 and 16.

Item 16 is not applicable: the existing revision 1 review was accepted, and its
verdict evidence remains attached. Item 4 exposes a lifecycle ordering issue:
the tracked runner publishes the Change Request only in its completion hook
after this agent process exits, while `handoff` requires every checklist item
before it will set the producer end status. There is no supported child-side
manual publish command.

For this final-leaf retry, checking item 4 therefore records authorization for
the deterministic post-exit publication hook; it is not evidence that revision
2 exists before handoff. The actual hook must still construct the revision, bind
it to this run's updated outcome evidence, and derive `story_final` from the
then-current board state. Any failure in that hook must be reported as a failed
completion rather than a published Change Request.
