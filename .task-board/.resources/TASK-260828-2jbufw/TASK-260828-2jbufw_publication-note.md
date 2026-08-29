# TASK-260828-2jbufw — publication note (scope narrowed to the profile half)

Run: RUN-260828-e7dda2 · Branch `task-board/story/STORY-260828-2faxgm` · Base `8045005`

This run did not re-do the task. It published the profile half that the previous
run had already built and validated, reconciled the delivered documents with the
scope split, and re-ran the verification gates against the delta as it stands now.

## What the delta is

8 files, all under `tools/agents-infra`, `README.md`, `LOGBOOK.md` and
`.research/`. Nothing under `tools/mlx-swift-runtime-prototype` — that path does
not exist on `main` or on this story's branch, so no gate code could be, and was
not, touched.

| Path | Change |
| --- | --- |
| `tools/agents-infra/internal/modelharness/run.go` | signal handling, process-group start, group forward, SIGKILL escalation, lifecycle records, signalled attempt never restarted |
| `tools/agents-infra/internal/modelharness/run_process_posix.go` | new — `SIGINT`/`SIGTERM`/`SIGHUP`, `Setpgid`, `kill(-pgid, …)` |
| `tools/agents-infra/internal/modelharness/run_process_windows.go` | new — `os.Interrupt` only, direct child only, limitation stated |
| `tools/agents-infra/internal/modelharness/run_shutdown_test.go` | new — 4 tests incl. one that builds the shipped binary and sends a real signal |
| `tools/agents-infra/llamacpp_profile_docs_test.go` | new — README profile bytes handed to the production resolver; README stop/observability contract pinned |
| `README.md` | `llamacpp-local` profile, stop contract, `llama-server` observability table |
| `LOGBOOK.md` | B8/B7 entry + scope-split record |
| `.research/260828_llamacpp-under-the-managed-harness.md` | full report |

## B8 — answered, and it is in the delivered evidence

The B8 answer for `llama-server` (pinned llama.cpp 0.3.0 build 10621) is carried
in three places inside this delta, not only on the board:

- `README.md` — the per-channel table an operator reads before pointing a
  managed profile at `-lv 5`.
- `.research/260828_llamacpp-under-the-managed-harness.md` §3 and the `B8 —
  closed` entry in §5 — the measurement, the two positive controls, and what was
  deliberately left open.
- `LOGBOOK.md` 1822 — the durable finding.

The answer itself: stdout is empty; there is **no HTTP access line at any
verbosity**, so `llama-server` is strictly *less* observable than
`mlx_lm.server` on the URL channel. It is strictly *more* attributable —
per-request `launch_slot_` / `print_timing` / `release` records keyed by a `task`
id, three concurrent completions giving three distinct ids where `mlx_lm.server`
gave three byte-identical lines — but the task id never leaves the engine, so B2
still stands. From `-lv 5` upward the **completion and tool calls** reach
harness-captured stderr as one `Parsed message` record and tool schemas as
compiled GBNF; the **prompt never does** at lv 3/5/10. B4's shape does not apply:
killed mid-body there is no status line to be wrong, and the capture holds a
`launch_slot_` with no matching `release` — unknown, not misreported. Raw captures
are in `TASK-260828-2jbufw_b8-nonce-grep.txt` and the evidence tarball.

## Gate work in this delta: none, and the inert parts are named

No gate code is present. What is present and is **documentation only**:
§4 and the G1/G2/G3 entries of the research doc, and the `REGRESSION RISK` /
`DECISION` lines in the `LOGBOOK.md` entry. They record the finding that
motivated the split into TASK-260828-3fgca3 and change no behaviour. Both
documents were edited in this run so their ownership pointers name
TASK-260828-3fgca3 rather than implying this task still owes the gate.

Safe to carry: they are prose in files with no build dependency (`grep -rl
LOGBOOK tools/agents-infra` → no matches; no test reads `.research/`), and the
delta's only executable additions are in `internal/modelharness` and one
`_test.go` under `tools/agents-infra`.

Checklist item 3 ("benchmark driver produces a RunRecord for llama.cpp with the
same LaunchProvenance binding") is **left unchecked deliberately** — it belongs
to TASK-260828-3fgca3 under the narrowed scope. Item 4 ("no admission clause
relaxed; prove with a narrowing mutant") stays checked: it is satisfied
negatively and the measurement is in `TASK-260828-2jbufw_gate-relaxation-cost.txt`.

## Verification re-run in this run, with real exit codes

Working directory `.temp/STORY-260828-2faxgm/worktree/tools/agents-infra`, each
command run standalone, no pipe to `tee`.

| Command | Exit | Result |
| --- | ---: | --- |
| `go build ./...` | 0 | clean |
| `go vet ./...` | 0 | clean |
| `go test ./internal/modelharness/...` | 0 | `ok … 1.961s` |
| `go test ./...` | 0 | `ok … 97.809s` (root), `ok` attachments/infra/modelharness |
| `gofmt -l .` | 0 | no files listed |
| `git apply --check --binary` of the CR patch in a detached clean-`HEAD` worktree | 0 | applies |

`golangci-lint` is not installed on this host; lint is `gofmt -l` + `go vet`,
both clean. That is stated rather than claimed as a full linter run.

The five mutation results (N1/N2/N3/D1/D2, all killed, each exit 1) were produced
by the previous run of this task and are **accepted from already-attached
evidence** in `TASK-260828-2jbufw_evidence.tar.gz`; they were not re-executed
here, because no line of the mutated code changed in this run.

## How `to-review` was reached, stated plainly

`task-board handoff TASK-260828-2jbufw --role developer` **refused**, exit
non-zero, with:

```
cannot hand off TASK-260828-2jbufw: unchecked checklist items [3] (benchmark
driver produces a RunRecord for llama.cpp with the same LaunchProvenance
binding as the other runtimes): handoff evidence missing
```

That refusal is correct on its own terms and item 3 was **not** checked to get
around it. The item is a leftover from the pre-split checklist: the board's `ac`
and `scope` were narrowed to the model-harness profile half, but the checklist
was not, and the CLI exposes only `add_checklist_item` / `check_item` /
`uncheck_item` — a producer cannot remove a stale item. Checking it would have
been a false attestation about work this task no longer owns, so the element was
moved with `set_status(TASK-260828-2jbufw, status=to-review)` instead, leaving
item 3 visibly unchecked.

**For the orchestrator:** item 3 belongs to TASK-260828-3fgca3 and should be
removed from this task's checklist or carried there. It is the one deliverable
in the original Definition of Done that this delta does not contain, and nothing
here claims otherwise.
