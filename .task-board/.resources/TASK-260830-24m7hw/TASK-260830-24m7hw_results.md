# TASK-260830-24m7hw — alert when KV cache saturates — results

## What changed

**mlx-lm fork** (`/Users/alexis/src/relux-works/mlx-lm`, branch `task/TASK-260830-2hc5r2-bounded-kv`, commit `6d2df63ab33e97d39e4740a4cc913d7d5838161b`, pushed to `origin/task/TASK-260830-2hc5r2-bounded-kv`):
- `mlx_lm/server.py`: `_report_kv_cache_bound_if_exceeded(request_id, active_max_kv_size, observed_tokens)` (new, near `_cache_max_kv_size`) logs one `kv_cache_bound_exceeded request_id=%s max_kv_size=%d observed_tokens=%d` warning when `observed_tokens > active_max_kv_size`. Called once from `APIHandler.handle_completion`, immediately after the `for gen in response:` generation loop completes (common to both the streaming and non-streaming branches, so every completion request is covered from one call site), with `observed_tokens = len(ctx.prompt) + len(tokens)` and `active_max_kv_size = self.response_generator.active_max_kv_size` (the live-introspected bound `/v1/models` `meta.n_ctx` already reports, not the launch argument).
- `tests/test_server.py`: `DummyModelProvider` gained an optional `max_kv_size` constructor arg; new `TestKVCacheBoundExceededLogging` spins up a real `ThreadingHTTPServer` + real small model (`mlx-community/Qwen1.5-0.5B-Chat-4bit`, same fixture the rest of the file already uses) with `max_kv_size=8`, and asserts the warning appears for an over-bound completion and not for an under-bound one.

**relux-agents-infra** (this worktree, `tools/agents-infra`):
- `internal/infra/pi_kv_bound_alert.go` (new): `piKVBoundAlertWriter` wraps an `io.Writer`, forwards every byte to it verbatim, and line-buffers a regex scan (`kv_cache_bound_exceeded request_id=(\S+) max_kv_size=(\d+) observed_tokens=(\d+)`) across write boundaries; on a match it calls a `report(requestID, maxKVSize, observedTokens)` callback.
- `internal/infra/pi_launch_posix.go`: `RunPi`'s runtime-child stdout/stderr writer (`runtimeOutput`, previously wired straight to `runtimeCmd.Stdout`/`Stderr`) is now wrapped by `newPiKVBoundAlertWriter`, whose `report` callback calls `sessionLog.event(opts.Context, "runtime_kv_cache_bound_exceeded", {request_id, max_kv_size, observed_tokens})` — the same Pi session log (`pi_session_log.go`) that already records `session_start`/`runtime_started`/`runtime_ready`/etc. This is the `infra` package (Pi launch), not `modelharness.Run`; the latter's `run is a pipe, not a recorder` characterization tests (`run_observability_test.go`) are untouched.
- `internal/infra/pi_kv_bound_alert_test.go` (new): direct writer-level tests (no subprocess needed) for verbatim forwarding, cross-write-boundary matching, near-miss rejection, ordinary-output silence, and multiple matches per write.

## AC coverage — 6 of 7 rows driven by a committed test at a named production call site

| # | AC clause | Driven by | Production call site |
|---|---|---|---|
| 1 | Over-bound request produces a record naming request id, bound, observed tokens | `TestKVCacheBoundExceededLogging.test_request_exceeding_bound_logs_once` | `mlx_lm/server.py: APIHandler.handle_completion` → `_report_kv_cache_bound_if_exceeded` |
| 2 | Check runs once per request, not in the per-layer/per-token path | Same test (one log line per request) + code location (post-loop, outside `stream_generate`/`RotatingKVCache`) | same call site |
| 3 | Decode throughput unchanged within noise | **Not a pass/fail test** — a stated, measured bound (see below) | n/a |
| 4 | Bound reported is the live server-introspected value, not the launch arg | `TestResponseGeneratorKVBound.*` (pre-existing, proves `active_max_kv_size` sources from the real cache object) + this task's call reads `response_generator.active_max_kv_size` directly | same call site |
| 5 | agents-infra surfaces the record through the runtime's normal log path | `TestPiKVBoundAlertWriterReportsMatchAndForwardsVerbatim` (Go) + live end-to-end run (below) | `internal/infra/pi_launch_posix.go: RunPi` → `newPiKVBoundAlertWriter` → `sessionLog.event` |
| 6 | Under-bound request produces no record | `TestKVCacheBoundExceededLogging.test_request_under_bound_logs_nothing` + `TestPiKVBoundAlertWriterReportsNothingForOrdinaryOutput` (Go) | both call sites above |
| 7 | A test drives the real server; removing the check turns it red | `TestKVCacheBoundExceededLogging` (both subtests) — narrowing mutant evidence below | `mlx_lm/server.py: APIHandler.handle_completion` |

Row 3 is declared a stated bound rather than a test because decode throughput is a continuous runtime measurement, not a boolean gate; it is reported as measured numbers in the "Decode throughput" section below.

## Negative tests / narrowing mutants

**Python (`mlx_lm`)** — removed the `_report_kv_cache_bound_if_exceeded(...)` call from `handle_completion`:
- `test_request_exceeding_bound_logs_once` → **FAIL** (`AssertionError: 0 != 1`)
- `test_request_under_bound_logs_nothing` → still PASS
- Restored; both green again.

**Go (`agents-infra`)** — narrowed `piKVBoundAlertWriter.scanLine` from the full regex match to a bare `bytes.Contains(line, []byte("kv_cache_bound_exceeded"))` check (preserves the searched-for token, changes the admitted class — the DoD's source-text-inspecting-gate mutant shape), reporting placeholder/zero fields on any line containing the substring:
- `TestPiKVBoundAlertWriterIgnoresNearMissLine/kv_cache_bound_exceeded_for_request_chatcmpl-abc123_` → **FAIL**
- `TestPiKVBoundAlertWriterIgnoresNearMissLine/kv_cache_bound_exceeded_request_id=...max_kv_size=notanumber...` → **FAIL**
- The plain HTTP-access-log near-miss subtest and every positive/verbatim/multi-line test → still PASS
- Restored; full `internal/infra` suite green again. The mutant harness ran the actual Go test binary (`go test ./internal/infra/...`), not a static checker.

## Live end-to-end run (real re-pinned distribution, real agents-infra binary)

Throwaway Pi profile `kv-alert-check-24m7hw` (`.temp/TASK-260830-24m7hw/live-project/.agents/.configs/project-config.toml`, not the deployed `qwen-3.8-27b-mlx-8bit` profile — a small local model was used so the run completes in seconds instead of the many minutes a genuine 76800-token prefill against the real 27B profile would take), launched via `agents-infra pi --profile kv-alert-check-24m7hw -p --no-tools --system-prompt "x" -- "hi"`, executable `/Users/alexis/.local/bin/mlx_lm-relux.server` (the pipx `mlx-lm-relux` distribution, re-pinned to `6d2df63` for this run), Go binary rebuilt from this worktree via `scripts/setup.sh`.

- `--max-kv-size 100`, one turn (164 prompt + 1 generated = 165 tokens): server stderr showed `kv_cache_bound_exceeded request_id=chatcmpl-d23e02b5-... max_kv_size=100 observed_tokens=165`; the Pi session log (`lifecycle-logs/entries/.../log.jsonl`) recorded `runtime_kv_cache_bound_exceeded {"max_kv_size":100,"observed_tokens":165,"request_id":"chatcmpl-d23e02b5-..."}`.
- `--max-kv-size 300`, same-shaped turn (164 prompt tokens, well under 300): no warning on stderr, no `runtime_kv_cache_bound_exceeded` event in the session log — only `session_start`/`runtime_started`/`runtime_ready`/`pi_started`/`pi_exited`/`pi_cleanup`/`runtime_cleanup`/`session_end`.
- No process left listening on the test port after either run (`lsof -i :18097` empty); the throwaway profile is not `sharing.mode = "shared"`, so it tore down fully each time — no shared broker or lease state touched.

## Decode throughput (before/after, mean of 3 runs each)

Measured against the local `mlx-community/Qwen1.5-0.5B-Chat-4bit` model directly under `mlx_lm.server` (editable checkout in `.temp/test-venv`), same chat prompt each run, `max_tokens=220` minus a `max_tokens=1` request on the same prompt to isolate decode time from prefill (`nN - n1` completion tokens over `tN - t1` wall-clock seconds). "Before" = `mlx_lm/server.py` at parent commit `06e2f0f` (temporarily restored, then reverted — `git diff HEAD` empty afterward, verified). "After" = this task's committed `6d2df63`.

| | run 1 | run 2 | run 3 | mean |
| --- | ---: | ---: | ---: | ---: |
| Before (tok/s) | 135.36 | 131.53 | 130.09 | 132.33 |
| After (tok/s) | 134.78 | 129.58 | 127.48 | 130.61 |

Δ mean = -1.72 tok/s (-1.3%), within the run-to-run spread observed in both sets (≈5 tok/s). No decode-path regression, consistent with the check running once post-loop rather than per-token/per-layer.

## Re-pin and redeploy

- `mlx-lm-relux` pipx distribution: uninstalled and reinstalled non-editable from `git+https://github.com/relux-works/mlx-lm.git@6d2df63ab33e97d39e4740a4cc913d7d5838161b`; `mlx_lm-0.32.0.dist-info/direct_url.json` confirms `vcs_info.commit_id = 6d2df63...`, no `dir_info` (non-editable).
- `/Users/alexis/src/.agents/.configs/model-harness.toml`: `[profiles.qwen-local.pinned_distribution] commit` updated `06e2f0f... → 6d2df63...`; backup `.bak-TASK-260830-24m7hw` holds the prior value.
- `model-harness doctor qwen-local --config /Users/alexis/src/.agents/.configs/model-harness.toml --host 127.0.0.1 --port 18021` → `status=ok`.
- `agents-infra`/`model-harness` binaries redeployed from this worktree via `scripts/setup.sh` (Go source changed this task, unlike TASK-260911-2zcdqe which changed no Go files).

## Validation run

- Python: `black --check`, `isort --check --profile=black` on `mlx_lm/server.py` + `tests/test_server.py` — clean. `python -m unittest tests.test_server -v` (all 37 tests, editable install in `.temp/test-venv`) — `OK`.
- Go: `gofmt -l .` — clean (no output). `go vet ./...` — clean. `go build ./...` — clean. `go test ./...` (full `tools/agents-infra`, no cache) — `ok` for the root package (102.4s), `internal/infra` (176.8s), `internal/attachments` (cached), `internal/modelharness` (13.5s). Exit code 0.

## LOGBOOK proof

```
$ diff <(git show b9ffcf26da3e210b84cbbd56189cfc27680a4002:LOGBOOK.md) LOGBOOK.md | grep -c '^<'
0
```

Additions-only relative to trunk `b9ffcf2` (this task's two 2026-09-11 entries: the base-refresh recovery, documented separately below, and this feature's summary).

## Base-refresh recovery (before implementation)

The Story worktree checkpoint (`900a447`) was 2-3 commits behind trunk when this task started. `task-board worktree refresh-candidate` has no working invocation for a clean, no-uncommitted-diff worktree (a genuine tool defect distinct from the previously-logged dirty-candidate revert bug — see LOGBOOK `### 1936`); routed around it by dry-running the rebase in a scratch `git worktree`, hand-merging the one real conflict (`LOGBOOK.md`, both sides append-only) additions-only, and submitting it as an explicit `--replay-resolutions` packet, which the tool's `refreshPreservedCandidate` path does support. The post-refresh working tree then silently reverted ~50 unrelated `.task-board/`/`tools/agents-infra` paths to the pre-refresh snapshot (the already-documented entry-1815/1830 bug); restored via `git checkout HEAD -- <path>` per path plus removing one stray untracked directory left from a since-relocated task. Verified clean: `git status` empty, `git diff --stat <trunk> HEAD` shows only the prior checkpoint's own 6 files (additions-only), `git merge-base <trunk> HEAD` equals `<trunk>`.
