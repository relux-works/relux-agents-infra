# TASK-260825-rtmcsw — Reviewer Verdict (CR revision 3)

**Verdict: ACCEPTED**

- Change Request: `CR-TASK-260825-rtmcsw-3`, revision `3`
- Base OID: `b3cb84550a60f7f4df92a287c573bfc692cd26e0`
- Candidate tree OID: `bf1f208724002da517e7ff9f89bdafe4c0650e9d`
- Reviewer run: `RUN-260825-bce1e8`
- Repository delta: `present` (10 files, +1685/-33)
- Worktree tree OID recomputed after every mutation experiment: `bf1f208724002da517e7ff9f89bdafe4c0650e9d` (identical to candidate — no reviewer residue)

## Scope reviewed

`agents-infra model-check` (`tools/agents-infra/main.go`, `internal/infra/model_check.go`,
`internal/infra/pi_run_report.go`) plus the `RunPi` lifecycle changes that make the
managed Pi run bounded and cleanup-observable (`internal/infra/pi_launch_posix.go`,
`pi_platform_windows.go`), the production-entrypoint suite
(`model_check_main_test.go`), the lifecycle unit tests (`internal/infra/model_check_test.go`,
`pi_test.go`), and the README/LOGBOOK updates.

## Gate attack (not a read-through)

Seven narrowing/behavioral mutants were applied to the candidate working tree, run,
and reverted. Every one was caught by a named test. This is the evidence that the
suite's refusals bind rather than merely execute.

| # | Mutant (narrowing, not delete-only) | Test that failed | Observed failure |
| --- | --- | --- | --- |
| M1 | Deadline gate lower bound removed (`< time.Millisecond` → `< 0`), upper bound kept | `TestModelCheckProductionEntrypoint/zero_deadline_refuses_before_launch` | `--deadline 0` launched the managed runtime and exited 2 instead of refusing pre-launch |
| M2 | Final-response accumulation widened (`=` → `+=` across assistant `message_end`) | `.../expected_text_is_limited_to_the_final_assistant_response` | earlier assistant text satisfied a final-response expectation |
| M3 | Malformed JSONL record skipped instead of marked invalid | `.../malformed_JSONL_refuses_instead_of_treating_it_as_absence` | corrupted stream reported as a clean pass (absence treated as satisfied) |
| M4 | `contextDone` branch no longer records `DeadlineExceeded` | `.../deadline_override_terminates_both_owned_process_groups` | timeout misreported as exit 3 malformed-stream |
| M5 | `modelCheckAssignmentSecret` redaction removed | `.../happy_path_persists_raw_and_sanitized_evidence` | fixture secret leaked into `summary.json` and terminal stdout |
| M6 | SIGKILL escalation removed from `terminateProcessGroupWithSignal` | `.../deadline_override_terminates_both_owned_process_groups` | `RuntimeProcessGroupCleanup="failed"`, `CleanupConfirmed=false` — attestation correctly refused to certify a surviving group |
| M7 | Nominal `*infra.ModelCheckFailure` match replaced by structural `interface{ ExitCode() int }` | `TestMainKeepsProviderChildFailuresAtLegacyExitOne` | Codex child exiting 42 changed the wrapper exit from legacy 1 to 42 |

M6 is the one that matters most: the cleanup attestation is not cosmetic. It is
derived from a live `kill(-pgid, 0)` probe, and when the group actually survives
the summary reports `failed` / `cleanup_confirmed=false` rather than certifying
success. M5 proves the sanitizer is load-bearing on the terminal surface while the
raw provider bytes are preserved in mode-0600 `events.jsonl`.

## Independent verification run by this reviewer

```
go build ./...                                     -> ok
go vet ./...                                       -> ok
gofmt -l .                                         -> clean
GOOS=windows GOARCH=amd64 go build/vet ./...       -> ok
GOOS=linux   GOARCH=amd64 go build ./...           -> ok
go test -run TestModelCheckProductionEntrypoint -v -> PASS, 14/14 subtests, 13.63s (NOT skipped)
go test ./internal/...                             -> ok (infra 99.15s)
go test .                                          -> ok (80.94s)
```

The production suite did **not** skip on this host: darwin/arm64 with the pinned
official Pi 0.84.2 asset resolved from the primary checkout. The runtime is a
local Python OpenAI-completions fixture — no external model or weight download
occurs at test time.

Process hygiene after the review: `ps` shows no surviving `model-check-runtime`,
`sleep 60`, or `agents-infra model-check` processes.

## Acceptance criteria

| AC | Verdict | Evidence |
| --- | --- | --- |
| Production CLI accepts configured target + prompt | met | `main -> run -> runModelCheck -> infra.RunModelCheck -> infra.RunPi`; every subtest drives a freshly built binary against `qwen-infra`. The provider request body is captured and asserted to contain the caller's prompt — the prompt is proven to reach the model, not assumed |
| Safe default deadline | met | `DefaultModelCheckDeadline = 5m` asserted as `deadline_ms=300000` on the happy path; `MaximumModelCheckDeadline = 30m` and a 1ms floor refuse pre-launch with no runtime spawned |
| Persists evidence | met | `events.jsonl`, `stderr.log`, `summary.json`, `summary.txt` each asserted regular + mode `0600`; `O_EXCL` creation and an `Lstat` pre-check refuse to overwrite prior evidence (both "full prior run" and "raw events only" cases tested, prior bytes verified unchanged) |
| Stable summary | met | schema-versioned JSON + deterministic sorted text render; exit/timeout, duration, provider/model, event counts, tool calls/failures, bounded 4096-byte final response with sha256 of the pre-truncation value, expectation results, and cleanup state |
| Non-zero for timeout / expectation failure / malformed stream | met | exits 2 / 4 / 3 asserted through the real binary; plus 5 for failed tool execution and 1 for launch-validation failure |
| Confirms cleanup | met | `pi_process_group_cleanup` and `runtime_process_group_cleanup` derived from live `ESRCH` probes; the timeout fixture deliberately parks a `trap '' TERM` descendant in each owned group and the test then probes every recorded PID directly for `ESRCH` |
| Tests drive the real CLI entrypoint | met | no in-process `run()` shortcut anywhere in the model-check suite |
| Reuse managed lifecycle, no duplicate shell launcher | met | `RunModelCheck` composes `BuildCanonicalTargetLaunchPlan` + `RunPi`; the bounding was added *inside* `RunPi` via optional `Context`/`Report`, so ordinary `pi`/`pi-infra` launches inherit the same bounded TERM→SIGKILL group cleanup |

## Negative-shape audit

- **Absence vs failure to read** — correctly separated. An empty event stream from a
  target that never launched yields exit 1 with the resolution error and an *empty*
  `event_stream.error`; it is not laundered into a malformed-stream verdict.
  `.../non-managed_canonical_target_refuses_without_inventing_stream_failure` pins this.
- **Forged evidence** — cleanup state cannot be self-minted; it is an OS probe (M6).
- **Check present but uncalled** — every gate is exercised through the built binary,
  and `TestProcessGroupCleanupStateReflectsLiveAndReapedGroups` pins the producer
  against a real live-then-reaped process group rather than a synthetic report struct.
- **Delete-only mutants avoided** — M1 narrows one bound and keeps the other; M2 widens
  a search surface; M7 loosens a type match. All caught.

## Non-blocking observations (recorded, not rework)

1. `validateModelCheckPlan` has four conjuncts; only the non-`pi` environment branch
   has a production negative test. A `pi` target with a non-managed profile is
   unproven. Low risk — the plan builder already refuses most of that space.
2. The entire production battery is skip-gated on darwin/arm64 plus a gitignored,
   task-scoped Pi asset path. This matches the existing repo pattern
   (`main_test.go`, `canonical_target_pi_main_test.go`), but it means the negative
   coverage silently disappears on a host without that asset. Pre-existing, not
   introduced here.
3. `prepareModelCheckOutputDir` chmods the output directory through a symlink if
   `--output-dir` is one. The artifact files themselves are safe (`O_EXCL`, no
   symlink follow). Minor hardening opportunity.
4. On a timeout with unconfirmed cleanup the exit code stays 2 rather than
   escalating; the unconfirmed state and its reason are still recorded in the
   summary, so no information is lost.

## Handoff

Accepted via `accept_cr(TASK-260825-rtmcsw, revision=3)`. Element parks at
`to-review`. No `commit_ack` supplied by this reviewer run — the orchestrator
commits the scope and makes the `done` transition.
