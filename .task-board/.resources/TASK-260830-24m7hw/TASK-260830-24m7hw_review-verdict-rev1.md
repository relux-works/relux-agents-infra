# TASK-260830-24m7hw — Review verdict rev1 — ACCEPT

Adversarial review of CR-TASK-260830-24m7hw rev 1, verified against real state
(not the producer's report). Did not restart the runtime; all checks were
read-only or self-contained (doctor doesn't touch the network in local mode;
mutant tests run in a scratch checkout copy, restored before finishing).

## Scope clarification

The CR diffs base b9ffcf2 → candidate tree (9 paths), but b9ffcf2 predates the
already-landed TASK-260830-heguyf commit (554b6a4) on this same branch. This
task's actual uncommitted delta (`git status --short`) is only 4 files:
`LOGBOOK.md`, `pi_launch_posix.go` (modified), `pi_kv_bound_alert.go` +
`pi_kv_bound_alert_test.go` (new). `pi_kv_bound.go`/`pi_kv_bound_test.go`/
`README.md`/`SKILL.md` changes in the CR diff belong to heguyf, already
reviewed and committed separately — verified they are byte-identical to
554b6a4's committed content and untouched by this candidate.

## Verified independently (not just read)

1. **Fork HEAD/origin**: `mlx-lm` repo HEAD = `6d2df63`, confirmed pushed
   (`git merge-base --is-ancestor 6d2df63 origin/task/TASK-260830-2hc5r2-bounded-kv`
   → ancestor). `_report_kv_cache_bound_if_exceeded` is called exactly once,
   inside `APIHandler.handle_completion`, immediately after the
   `for gen in response:` loop finishes — outside `stream_generate` and
   `RotatingKVCache._update_*`, common to both streaming and non-streaming
   branches. Confirmed via `git show 6d2df63 -- mlx_lm/server.py`.
2. **Bound source**: `active_max_kv_size` is read from
   `ResponseGenerator._active_max_kv_size`, itself populated by
   `_cache_max_kv_size(cache)` reading `.max_size` off the actual created
   cache object, not `cli_args.max_kv_size` directly — satisfies "bound
   actually in force ... not the launch argument."
3. **pipx pin**: `direct_url.json` for the `mlx-lm-relux` pipx venv shows
   `vcs_info.commit_id = 6d2df63...`, no `dir_info` (non-editable). Matches.
4. **model-harness.toml pin + doctor**: live
   `/Users/alexis/src/.agents/.configs/model-harness.toml`
   `pinned_distribution.commit = 6d2df63...`; backup
   `.bak-TASK-260830-24m7hw` holds the prior value `06e2f0f`, diff isolated to
   that one field. `model-harness doctor qwen-local --host 127.0.0.1 --port
   8080 --config <live path>` → `status=ok` (ran myself; local mode, no
   network touched, no runtime restarted).
5. **Go narrowing mutant**: reproduced the claimed mutant myself — replaced
   `scanLine`'s regex match with a `bytes.Contains(line,
   []byte("kv_cache_bound_exceeded"))` guard (source-text-preserving, per the
   DoD's mutant-on-a-text-inspecting-gate requirement) plus a fallback
   `report("mutant", 0, 0)` on non-matching-but-containing lines. Ran
   `go test ./internal/infra/... -run TestPiKVBoundAlertWriterIgnoresNearMissLine
   -v`: the two near-miss subtests that must stay ignored now FAIL as
   claimed, while the plain-HTTP-log subtest still passes. Restored the file,
   re-ran — green again. Full `tools/agents-infra` suite green after restore
   (`go test ./...`, 128s root + 222s internal/infra).
6. **Python narrowing mutant**: reproduced independently — removed the
   `_report_kv_cache_bound_if_exceeded(...)` call from `handle_completion` in
   a scratch copy, ran `python -m unittest
   tests.test_server.TestKVCacheBoundExceededLogging -v` via the leftover
   `.temp/test-venv` (editable install, real small model, real
   `ThreadingHTTPServer`+`APIHandler`+`ResponseGenerator`):
   `test_request_exceeding_bound_logs_once` → FAIL (`0 != 1`),
   `test_request_under_bound_logs_nothing` → still PASS. Restored, re-ran
   both green. `black --check` and `isort --check --profile=black` clean on
   both changed Python files (ran myself).
7. **LOGBOOK additions-only**: `git diff b9ffcf2 -- LOGBOOK.md | grep '^-' |
   grep -v '^--- '` → empty (additions-only, reproduced the producer's exact
   claim). The heguyf entry ("### 1620 — Pi context_window And model-harness
   --max-kv-size Are Now Bound Together") is present in the current worktree
   byte-for-byte identical to what 554b6a4 committed (diffed the exact
   section, empty diff) — survived the base-refresh recovery intact.
8. **Production wiring**: `RunPi` (`pi_launch_posix.go`) wraps
   `runtimeOutput` in `newPiKVBoundAlertWriter` before assigning to
   `runtimeCmd.Stdout`/`Stderr`; the `report` callback calls
   `sessionLog.event(opts.Context, "runtime_kv_cache_bound_exceeded", ...)`
   — the same session log surface as `runtime_started`/`runtime_ready`, so an
   operator sees it without reading raw server output. Confirmed by reading
   the diff, not just the report.

## Judgment call requested by the spawn brief: throwaway profile for the live E2E run

The live end-to-end run (`kv-alert-check-24m7hw`, a throwaway local profile
with a small model) is **not** the deployed `qwen-3.8-27b-mlx-8bit` profile.
Judged **acceptable** as evidence for the AC's "drives the real server" /
decode-throughput clauses:

- The AC's "test drives the real server ... removing the check makes the
  first test red" is satisfied by the *committed* `TestKVCacheBoundExceededLogging`
  Python test, which runs the real `ThreadingHTTPServer` + real `APIHandler`
  + real `ResponseGenerator` classes from the pinned `6d2df63` commit — same
  binary/distribution the deployed profile also runs, independent of which
  model is loaded. I reran this test myself; it passes and the mutant
  reproduction confirms it is a real, narrowing gate.
- The live E2E run and the decode-throughput A/B are *additional* manual
  verification beyond the AC-mandated automated test, not a substitute for
  it. Using a small model here is the conservative choice, not a weaker one:
  the per-request overhead this check adds is O(1) (one string format + one
  log call), independent of model size, while a small/fast model has a
  *higher* baseline tok/s than the large deployed model — so any fixed
  overhead shows up as a *larger* relative regression on the small model than
  it would on the real 27B profile. A result "within noise" on the harder
  case is stronger, not weaker, evidence that the large-model case is fine.
  Producer's rationale (avoiding a many-minutes 76800-token prefill against
  the live profile) is reasonable and consistent with the spawn brief's
  "do not restart the runtime" constraint on my own review pass.
- Disclosure was explicit in `TASK-260830-24m7hw_results.md` and the LOGBOOK
  entry ("not the deployed qwen-3.8-27b-mlx-8bit profile") — not glossed
  over.

## AC coverage ratio

Producer reported 6 of 7 AC rows driven by a named committed test at a named
production call site, with row 3 (decode throughput) explicitly declared a
stated bound (continuous measurement, not a boolean gate) rather than
claimed as test-driven. I independently verified rows 1, 2, 6, 7 (mutant
reproduction) and row 5 (Go writer wiring + test). Row 4 (bound is
server-introspected, not launch arg) verified by reading
`ResponseGenerator._active_max_kv_size` / `_cache_max_kv_size`. This is a
real ratio backed by evidence, not prose standing in for one.

## Other checks

- `git status --short` in the worktree matches exactly the expected
  uncommitted candidate delta; no stray files.
- Candidate left uncommitted at the recorded checkpoint (no commits of the
  producer's own on the Story branch).
- `mlx-lm` fork checkout: `git diff --stat` clean after my mutant
  reproduction and restore (only a pre-existing untracked `.temp/` scratch
  dir left by the producer, unrelated to tracked state).
- `tools/agents-infra`: `gofmt -l` clean, `go vet ./...` clean, `go build
  ./...` clean, full `go test ./...` green (reran myself, not just trusting
  the attached validation log — my timings were consistent with the attached
  log's).

## Verdict

**ACCEPT.** No forced fits, no positive-path-only evidence, no fabricated or
reused-without-attack evidence — every load-bearing claim in the CR (fork
commit, pin, doctor, both narrowing mutants, LOGBOOK additions-only, heguyf
entry survival) was reproduced independently in this review pass and matched
the producer's report exactly.
