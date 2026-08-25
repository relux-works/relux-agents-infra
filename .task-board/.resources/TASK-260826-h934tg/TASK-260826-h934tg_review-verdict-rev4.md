# TASK-260826-h934tg — Reviewer Verdict, CR revision 4

- Reviewer run: `RUN-260826-2dd5f9`
- Change Request: `CR-TASK-260826-h934tg-4`, revision `4`, state `ready`
- Base OID: `355a156276080b6994080f8e9a767e7416a5b357`
- Candidate tree OID: `8c8096878aa5a7b9bb2fefa129d2407dd620e009`
- Worktree: `.temp/STORY-260825-2l6axn/worktree`, branch `task-board/story/STORY-260825-2l6axn`, HEAD `eaefc6f4f41a9e2c2c0f0464a8fb632ab13910ff`
- Toolchain: `go1.25.5 darwin/arm64`

## Verdict

**accepted.** Revision 3's single finding (F1 — the shared-runtime standalone
terminal guard had no witness) is independently closed on this tree. No new
findings. No production code changed since revision 3.

## Candidate identity

| Check | Command | Result |
| --- | --- | --- |
| Attached patch is the real delta | `git diff 355a156 8c80968` vs `TASK-260826-h934tg_change-request_rev4.patch` | byte-identical (`diff` exit 0) |
| Patch digest | `shasum -a 256` | `9432708681568c904014ca430265f33263ef99bbf75d1a10302391252b828299` — matches the CR |
| Candidate tree reproduces | temp-index `git read-tree HEAD && git add -A && git write-tree` | `8c8096878aa5a7b9bb2fefa129d2407dd620e009` — exact |

## Revision 3 → revision 4 delta is exactly what was required

`git diff --stat b59f8c94 8c80968`:

```
 LOGBOOK.md                                      |  4 ++--
 .../internal/infra/pi_standalone_shared_test.go | 13 +++++++++++++
 2 files changed, 15 insertions(+), 2 deletions(-)
```

Two files, both non-production. **Every production file is byte-identical to
revision 3** — `pi_launch_posix.go`, `pi_shared_client_darwin.go`,
`pi_standalone.go`, `pi_config.go`, `canonical_target.go`,
`pi_platform_windows.go`, `project_config.go`, `main.go` do not appear in the
delta at all.

The test delta is exactly the reviewer-specified fix: a forced-positive
`piTerminalFDProbe` override with `t.Cleanup` restoration, a zero-probe-call
assertion, and a production-call-site comment naming
`RunPi -> runSharedPiSession` in `pi_shared_client_darwin.go`.

The LOGBOOK delta corrects both overstated EVIDENCE lines called out in the
revision 3 verdict (1528 and 1411), naming both witnesses and both production
launches. Both corrected claims are true on this tree — verified below.

## F1 is closed: the shared guard is now witnessed, and the witness is load-bearing

`TestRunPiStandaloneConcurrentWorkersShareOnlyRuntimeAndCrashCleanupPreservesPeer`
drives the real `RunPi` entry point on a `runtime.sharing.mode = "shared"`
profile with `Stdin: strings.NewReader("shared-readable-stdin-witness-…")` and a
probe forced to `(0, true)`, then asserts zero probe calls **after both
concurrent workers have launched**, plus `StdinEOF` per worker.

Mutants applied to and restored from the candidate tree:

| # | Mutant | Expectation | Result |
| --- | --- | --- | --- |
| M4 | `pi_shared_client_darwin.go`: narrow the guard — standalone keeps closed stdin but calls `configurePiProcessTerminal(piCmd, opts.Stdin)` | must now die | **KILLED** — `FAIL`, worker a `fork/exec …: operation not supported by device` (forced-positive probe puts `Ctty` on a non-tty fd) |
| M4-rev3shape | Same M4 mutant, with the revision-4 probe override and assertion **removed** from the test (i.e. revision 3's test shape) | reproduces rev3 F1 | **SURVIVES** — `ok`, `PASS` in 1.32s |
| M4c | `pi_shared_client_darwin.go`: standalone branch calls `configurePiProcessTerminal` but `SysProcAttr` is then restored to `Setpgid: true` (process starts fine, stdin still closed, PGIDs still independent) | only the new assertion can catch this | **KILLED** by the new assertion: `shared standalone workers attempted interactive terminal detection 2 time(s)` |
| M3 | `pi_shared_client_darwin.go`: standalone inherits `opts.Stdin` | retained | **KILLED** — `worker 0 inherited readable stdin: … StdinEOF:false` |
| M1c | `pi_launch_posix.go`: exclusive standalone branch leaks the probe, `Setpgid` restored | other half of the split | **KILLED** by `TestRunPiStandaloneExclusiveWorkerClosesReadableStdin`: `exclusive standalone worker attempted interactive terminal detection 1 time(s)` |
| M5 | `pi_standalone.go`: allowlist gate widened to `if tool == ""` (any non-empty tool name admitted) | authorization boundary | **KILLED** — `TestStandalonePiAuthorizationRejectsNarrowedAndInvalidAllowlists`, `want "pi_tool_allowlist_invalid"` × 2 |

M4-rev3shape is the decisive one: it proves the seven added lines are not
decoration. With them removed the exact regression this revision exists to
prevent passes silently; with them present it dies. M4c proves the
`terminalProbeCalls != 0` assertion is itself live and covers **both**
concurrent workers (it reports `2 time(s)`), not merely a side effect of `Ctty`
on a non-terminal fd.

Both witnesses drive the production entry point `RunPi`, not a helper: the
exclusive one reaches `pi_launch_posix.go:304-310`, the shared one reaches
`pi_shared_client_darwin.go:648-654` via `runtime.sharing.mode = "shared"`.

### Exact restoration

After every mutant the file was restored from a byte backup and re-hashed:
`pi_launch_posix.go` `84a09900…`, `pi_shared_client_darwin.go` `b5e1adac…`,
`pi_standalone_shared_test.go` `593f4f97…`, `pi_standalone.go` `e9622b77…`.
Post-review temp-index `git write-tree` returns
`8c8096878aa5a7b9bb2fefa129d2407dd620e009` — the candidate tree is untouched.
Focused witnesses re-run clean after restoration (4/4 PASS, `ok` 5.4s).

## Ancestry — real main ancestry, not a replay

| Check | Result |
| --- | --- |
| `git rev-parse main` | `355a156276080b6994080f8e9a767e7416a5b357` — identical to the CR base OID |
| `git merge-base --is-ancestor 355a156 HEAD` | exit 0 |
| `git rev-list --count HEAD..main` | `0` |
| `git rev-parse HEAD^1 HEAD^2` | `3a52ec76…` / `355a156276…` — main is literally the second parent of the merge |

## Scope

- 17 changed paths, all within the standalone-Pi surface plus additive docs.
- `pi_standalone.go` and `main.go` are **byte-identical** to the accepted
  checkpoint `8f81371` — no CLI surface change, no allowlist widening.
- Every main-owned interactive/session file is **unchanged versus main**:
  `pi_terminal_darwin.go`, `pi_terminal_linux.go`, `pi_terminal_other_posix.go`,
  `pi_session_log.go`, `pi_state.go`, `pi_run_report.go`, `pi_args.go`,
  `pi_plan.go`, `pi_test.go`, `pi_operator_docs_test.go` (zero diff lines each).
- `piTerminalFDProbe` is a package-private seam defaulting to `piTerminalFD`;
  it is assigned only at its declaration and inside tests, each with
  `t.Cleanup` restoration. No exported surface added.
- Allowlist stays the pinned built-in set `read,bash,edit,write,grep,find,ls`,
  with empty / duplicate / unknown names refused (M5 confirms the gate is live).
- Pinned argv still carries `--no-approve --no-extensions --tools … --mode json
  --no-session --print`; the concurrency test refuses `rpc`, `--approve`,
  `--extension`, `-e` in the observed child argv.
- No task-board adapter: only the `task_board_adapter:
  "deferred_not_implemented"` diagnostic string, plus
  `TestStandalonePiStateNeverDerivesFromTaskBoardRunID` proving client state
  never derives from `TASK_BOARD_RUN_ID` (the concurrency test injects
  `TASK_BOARD_RUN_ID=RUN-forged-board-id` and still requires isolated state).
- No sudo / setuid / seteuid / root escalation anywhere in
  `tools/agents-infra/**/*.go` (grep clean); `sudo` appears only in `.research/`
  and in LOGBOOK/SKILL prose that rejects it.
- `runPiStandaloneCLI` (`main.go:542-549`) passes **no** `Stdin` field at all.
- Documentation is additive. README's only two deleted lines are two table rows
  rewritten in place; a token-level superset check confirms **no** documented
  command or artifact token was dropped. SKILL/LOGBOOK/.research are
  insertion-only (`+19/-0`, `+36/-0`, `+637/-0`).

## Full landing suite on the exact revision-4 tree

Test cache cleared with `go clean -testcache` first; every run `-count=1`.

| Gate | Result |
| --- | --- |
| `go test ./internal/infra/ -count=1` | `ok` 115.8s |
| `go test ./ ./internal/attachments/ -count=1` | `ok` 72.6s / `ok` 1.1s |
| `go vet ./...` (darwin) | exit 0 |
| `GOOS=linux go vet ./...` | exit 0 |
| `GOOS=windows go vet ./...` | exit 0 |
| `gofmt -l .` | empty |
| `go build ./...` × darwin/arm64, darwin/amd64, linux/amd64, linux/arm64, windows/amd64 | all exit 0 |

## Acceptance criteria

| AC | Status |
| --- | --- |
| Final candidate contains the exact accepted standalone YOLO behavior | met — `pi_standalone.go`/`main.go` byte-identical to `8f81371` |
| Current main is an ancestor of HEAD | met — `355a156` is HEAD's second parent, `HEAD..main` = 0 |
| Additive documentation preserved | met — superset check on the only two rewritten README rows; all other doc edits insertion-only |
| No unrelated paths change | met — 17 paths, all in scope; main's interactive/session files untouched |
| Full Go tests / vet / build checks pass on the merged tree | met — table above |

## Reviewer hygiene

Six mutants applied and byte-restored; the board index was never touched; no
repository file was modified by this review. Command logs are attached as
`TASK-260826-h934tg_rev4-review-evidence.tgz`.
