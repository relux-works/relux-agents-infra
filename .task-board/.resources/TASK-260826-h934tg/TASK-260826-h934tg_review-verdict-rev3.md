# TASK-260826-h934tg — Reviewer Verdict, CR revision 3

- Reviewer run: `RUN-260826-9c9036`
- Change Request: `CR-TASK-260826-h934tg-3`, revision `3`
- Base OID: `355a156276080b6994080f8e9a767e7416a5b357`
- Candidate tree OID: `b59f8c9418476b62799c9653f5d23ee7b535329e`
- Worktree: `.temp/STORY-260825-2l6axn/worktree`, branch `task-board/story/STORY-260825-2l6axn`, HEAD `eaefc6f4f41a9e2c2c0f0464a8fb632ab13910ff`

## Verdict

**changes_requested → `to-dev`.** One finding (F1). Product behavior is correct;
the defect is missing negative evidence for one of the two production standalone
terminal guards this revision introduced. No product-code change is required to
close it.

## Independently verified (all pass)

### Ancestry and candidate identity

| Check | Command | Result |
| --- | --- | --- |
| Current main tip | `git rev-parse main` | `355a156…` — identical to the CR base OID |
| Main is an ancestor of HEAD | `git merge-base --is-ancestor 355a156 HEAD` | exit 0 |
| Branch is not behind main | `git rev-list --count HEAD..main` | `0` |
| Candidate tree reproduces from the worktree | temp-index `git read-tree HEAD && git add -A && git write-tree` | `b59f8c9418476b62799c9653f5d23ee7b535329e` — exact match |

HEAD `eaefc6f` is a real merge whose second parent is main `355a156`; this is
genuine main ancestry, not a replay-equivalent tree.

### Main's interactive behavior is preserved byte-for-byte

`git diff 355a156 b59f8c94` is **empty** on every file main `355a156` owns for
foreground-terminal and session-log behavior: `pi_terminal_darwin.go`,
`pi_terminal_linux.go`, `pi_terminal_other_posix.go`, `pi_session_log.go`,
`pi_state.go`, `pi_run_report.go`, `pi_args.go`, `pi_plan.go`, `pi_test.go`,
`pi_operator_docs_test.go`.

In both launch paths the interactive branch is main's code verbatim:

```go
if opts.Standalone == nil {
    piCmd.Stdin = opts.Stdin
    foreground = configurePiProcessTerminal(piCmd, opts.Stdin)
} else {
    piCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}
```

`configurePiProcessTerminal` is unchanged apart from routing through the new
`piTerminalFDProbe` seam, which defaults to `piTerminalFD`. Session-log events
(`session_start`, `pi_started`/`foreground`, `pi_cleanup`, `session_end`, …) are
emitted in both paths exactly as main emits them.

### Standalone attaches neither caller stdin nor foreground terminal state

Both production call sites confirmed:
`pi_launch_posix.go:304-310` (`RunPi`, exclusive managed runtime) and
`pi_shared_client_darwin.go:648-654` (`runSharedPiSession`, shared runtime,
reached from `RunPi` when `runtime.sharing.mode = "shared"`). In the standalone
branch `piCmd.Stdin` is never assigned (child gets `/dev/null` → EOF) and
`SysProcAttr` carries only `Setpgid: true` — never `Foreground`/`Ctty`.
`runPiStandaloneCLI` in `main.go` likewise passes no `Stdin`.

### Retained guarantees (re-run, all green)

Primary-yolo non-inheritance (`--no-approve` count exactly 1, no `--approve`/`-a`/`-na`),
stdin EOF in both paths, exact allowlist/argv composition, prompt redaction in
`DiagnosticArgv` and in every typed CLI failure, deadline bounds `(0, 30m]`,
per-run random hash-contained client state that never derives from
`TASK_BOARD_RUN_ID`, shared-runtime lease concurrency, crash cleanup that
preserves a live peer, and final-lease runtime reaping.

### Scope

`pi_standalone.go` and `main.go` are **identical** between the accepted
checkpoint `8f81371` and this candidate — no allowlist widening, no CLI surface
change. Allowlist stays the pinned built-in set
`read,bash,edit,write,grep,find,ls` with empty/duplicate/unknown/wildcard all
refused. No task-board adapter (only the `task_board_adapter:
"deferred_not_implemented"` diagnostic and a test proving state never derives
from a board run id). No sudo/root/setuid path in any production file —
`sudo` appears only in `.research/` and in LOGBOOK/SKILL prose that rejects it.
LOGBOOK, README and SKILL edits are additive.

### Full landing suite on the exact revision-3 tree

| Gate | Result |
| --- | --- |
| `go test ./... -count=1` | `ok` × 3 packages (root 82.7s, attachments 1.7s, infra 131.3s) |
| `go vet ./...` (darwin) | clean |
| `go vet ./...` (`GOOS=linux`, `GOOS=windows`) | clean |
| `gofmt -l .` | clean |
| `go build ./...` × darwin/arm64, darwin/amd64, linux/amd64, linux/arm64, windows/amd64 | all OK |

## F1 — the shared-runtime standalone terminal guard has no witness

**Category:** test-coverage / unwitnessed gate.
**Severity:** blocking for this revision (an explicitly required verification item).
**Production code is correct — do not change it.**

`pi_shared_client_darwin.go:648` and `pi_launch_posix.go:304` are two distinct
production standalone terminal guards. Only the exclusive one is witnessed.

Mutants run against the candidate tree:

| # | Mutant | Suite result |
| --- | --- | --- |
| M1 | `pi_launch_posix.go`: narrow the guard so `configurePiProcessTerminal` also runs for standalone (stdin still closed) | **KILLED** — `TestRunPiStandaloneExclusiveWorkerClosesReadableStdin` fails |
| M2 | `pi_launch_posix.go`: widen so standalone inherits `opts.Stdin` | **KILLED** — same test, `StdinEOF:false` |
| M3 | `pi_shared_client_darwin.go`: widen so standalone inherits `opts.Stdin` | **KILLED** — `TestRunPiStandaloneConcurrentWorkersShareOnlyRuntimeAndCrashCleanupPreservesPeer` fails, `StdinEOF:false` |
| M4 | `pi_shared_client_darwin.go`: narrow **only** the terminal guard — standalone keeps closed stdin but calls `configurePiProcessTerminal(piCmd, opts.Stdin)` | **SURVIVES** — `go test ./internal/infra/ -count=1` returns `ok` in 117.0s |

M4 is the exact regression this revision exists to fix, on the other half of the
split. It survives because `TestRunPiStandaloneConcurrentWorkersShareOnlyRuntime…`
supplies a `strings.Reader` stdin, so the real `piTerminalFD` returns `false`
regardless, and the test never overrides `piTerminalFDProbe` or asserts a probe
count. The PGID assertion cannot discriminate either: on a non-terminal stdin the
narrowed guard still lands on `Setpgid: true`. Shipped, a future edit could give
a `qwen-infra spawn` worker on a `runtime.sharing.mode = "shared"` profile
`Foreground: true, Ctty: <caller tty>` from an interactive terminal, with the
whole suite green.

Two evidence claims in the delta overstate what is proven:

- LOGBOOK 1528 EVIDENCE cites only the exclusive witness while the FIX line names
  both `pi_launch_posix.go` and `pi_shared_client_darwin.go`.
- LOGBOOK 1411 asserts "Shared and exclusive production paths use the same split"
  with the concurrency test as evidence; that test does not witness the shared
  terminal half of the split.

### Required fix (verified by this reviewer, ~7 lines, tests only)

Give the shared path the same forced-positive probe the exclusive witness uses.
In `TestRunPiStandaloneConcurrentWorkersShareOnlyRuntimeAndCrashCleanupPreservesPeer`
(or a new focused shared witness), after the `piExecCommand` override:

```go
originalTerminalFDProbe := piTerminalFDProbe
var terminalProbeCalls atomic.Int32
piTerminalFDProbe = func(io.Reader) (int, bool) {
    terminalProbeCalls.Add(1)
    return 0, true
}
t.Cleanup(func() { piTerminalFDProbe = originalTerminalFDProbe })
```

and assert before the per-worker loop:

```go
if calls := terminalProbeCalls.Load(); calls != 0 {
    t.Fatalf("shared standalone workers attempted interactive terminal detection %d time(s)", calls)
}
```

Reviewer-confirmed on this tree:

- against correct production → `ok` (2.5s);
- against M4 → `FAIL`, `fork/exec …: operation not supported by device` (the
  narrowed guard sets `Ctty` on a non-terminal fd before the probe assertion is
  even reached).

Then correct the LOGBOOK 1528/1411 EVIDENCE lines to name the shared witness, and
re-run the full landing suite.

## Reviewer hygiene

All mutants were applied to and restored from the worktree. Post-review
`git read-tree HEAD && git add -A && git write-tree` on a temp index returns
`b59f8c9418476b62799c9653f5d23ee7b535329e` — the candidate tree is byte-identical
to what the CR snapshotted. The board index was never touched.
