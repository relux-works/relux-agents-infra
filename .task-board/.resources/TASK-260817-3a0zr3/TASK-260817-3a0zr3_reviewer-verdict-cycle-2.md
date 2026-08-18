# TASK-260817-3a0zr3 reviewer verdict — cycle 2

Verdict: **accepted**, pending the commit-owning mover's enforced final `done` transition.

## Cycle-1 findings closed

- F1 is closed: production verification inspects both the managed `pi-infra` alias and its sibling `agents-infra` target with pathname-level `Lstat` semantics and rejects symlink/non-regular replacements.
- F2 is closed: production `setup local` repairs a byte-identical alias whose POSIX mode drifted to `0644`, and repairs a byte-identical symlink alias to a regular `0755` artifact.

## Gate-defeat evidence

The reviewer built the source production binary, installed a real local runtime in an isolated `/tmp` project, and attacked the real `setup local` / `verify local` entry points:

1. Changed the installed alias from `0755` to `0644`; setup restored a regular `0755` file.
2. Replaced the alias with a symlink to an external byte-identical copy; verify refused `is not a regular file`, then setup restored a regular `0755` file.
3. Replaced the sibling target with a symlink to an external byte-identical copy; verify refused `pi-infra launcher target is not a regular file`.

This closes the cycle-1 **bypass path around the check**. A follow-link/content-only narrowing would admit the exact production attack and fail the named regression.

Evidence logs:

- Board outcome `TASK-260817-3a0zr3_reviewer-manual-attack-02.log` contains the complete production attack transcript.
- Board outcome `TASK-260817-3a0zr3_reviewer-go-test-full-01.log` contains the uncached full-suite result.
- Detailed alias/target refusal streams remain in the task-scoped local review directory and are summarized verbatim by the manual attack transcript.

## Validation

| Check | Result |
| --- | --- |
| Named global/local production setup tests, uncached | Pass (`8.230s`) |
| `go test ./... -count=1` | Pass (`31.900s`, `2.297s`, `95.799s`) |
| `go vet ./...` | Pass |
| `go build ./...` | Pass |
| `git diff --check` | Pass |
| `task-board validate` | Pass |
| Spawn directives | None |
| Spawn goal | Run is not goal-bound |

The retained caller cwd/argv/post-separator contract and operator documentation tests pass in the full suite. No acceptance-blocking findings remain. The reviewer did not modify product code.

## Commit handoff

Board version-control confirmation is enabled. This task is the last active child of `STORY-260817-1on8ex`, so accepting it promotes the Story to `done` and requires mover-local `commit_ack=scope_committed`. Reviewer-archetype runs must not provide that acknowledgement. The commit-owning mover must inspect and commit its scoped changes, then make the final task `done` transition with the required acknowledgement.
