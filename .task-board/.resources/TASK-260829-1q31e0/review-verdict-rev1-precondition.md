# TASK-260829-1q31e0 review verdict

## Verdict

**changes_requested** for CR `CR-TASK-260829-1q31e0-1` revision 1, candidate tree `6cc89089ce65f3dfe25302b8a02d1d2d4c24b888`.

Route: `to-dev`. This is ordinary implementation rework, not a Stop-The-Line boundary.

## Blocking finding

### P1 — Retention is per run-state directory, so aggregate count, bytes, and age are unbounded

The policy is profile-scoped, but production shared and standalone paths create a distinct state root for each run ID:

- `tools/agents-infra/internal/infra/pi_state.go:82-90` appends `runs/<run-key>`.
- `tools/agents-infra/internal/infra/pi_shared_client_darwin.go:585-617` always resolves that client state for shared sessions, then opens the lifecycle log there.
- `tools/agents-infra/internal/infra/pi_launch_posix.go:151-179` does the same for standalone sessions.
- `tools/agents-infra/internal/infra/pi_lifecycle_log_posix.go:48-76,176-185` observes and prunes only the one supplied `paths.LogsDir`; it never scans sibling run-state directories.

Therefore each new turn/run gets an independent full allowance. Old run directories are never revisited by later run IDs. This is the standard **bypass path around the check** shape: exclusive non-standalone state is bounded, while shared/standalone production paths bypass the intended profile aggregate.

The fake-clock soak does not cover the production shape: `tools/agents-infra/internal/infra/pi_lifecycle_log_test.go:150-180,231-243` repeatedly uses one `ResolvePiStatePaths` directory. It cannot fail when retention is narrowed to one directory.

An expected-red reviewer probe used the same `ResolvePiClientStatePaths -> openPiSessionLog` path selected by both production callers. With one profile policy (`max_count=1`, `max_bytes=120`, `max_age_seconds=60`) and three distinct run IDs, it observed:

```text
profile aggregate managed_count=3/1 managed_bytes=183/120 expired_count=3
```

The second probe showed that diagnostics use the non-run root (`tools/agents-infra/internal/infra/pi_plan.go:184,217`) and reported `observation="absent", managed_count=0` while a run-state lifecycle log existed. AC 3 and AC 5 are therefore both unmet, and AC 1's explicit bounds do not actually bind all production paths.

## Required rework

1. Establish one profile-aggregate ownership/pruning boundary covering exclusive, shared, and standalone lifecycle logs while retaining run-state isolation for agent/session data.
2. Serialize or otherwise make aggregate pruning deterministic across concurrent run IDs; preserve every active file and every foreign entry across all participating directories.
3. Make status/`--print-config` observe the same aggregate that pruning governs; read failures must report `unknown`, never `absent` or `within_policy=true`.
4. Replace or extend the eight-week test so every simulated turn uses a fresh run ID through the production state-selection path, and assert aggregate count/bytes/age plus foreign and active preservation.
5. Add a negative test that narrows pruning/status back to one run directory and fails at the real shared/standalone production entry points.
6. Append a corrective `LOGBOOK.md` entry; the current 1818 entry claims aggregate bounds that the reviewer probe disproved. Reviewer did not edit it because CR review is read-only and the candidate snapshot must remain immutable.

## Validation evidence

- `git diff --check <base> <candidate>`: exit 0.
- Focused candidate suite with `-count=1`: exit 0, 9.896s. It confirms same-directory behavior but misses the bypass above.
- Expected-red overlay probe with `-count=1`: exit 1, 1.415s; both aggregate enforcement and status tests failed with the values above.
- Producer CR validation log reports `go test ./... -count=1` and `go vet ./...` exit 0; not rerun broadly because the focused negative reproduction is sufficient to reject revision 1.
- No live Pi/model process or socket used. No repository source modified by review; probe and readiness artifacts are under ignored `.temp/TASK-260829-1q31e0/`.
