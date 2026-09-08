# TASK-260830-tvy8q5 review verdict — revision 5

## Verdict

Changes requested. Route to `to-dev`.

Immutable review surface:

- Change Request: `CR-TASK-260830-tvy8q5-5`, revision 5
- Base OID: `b78498bf98c05175db10bb341aee621e53de4881`
- Candidate tree OID: `78fd4e6ef09d69f9258b20e3cf027e98c36a5a04`
- Patch SHA-256: `e22844e482999afdc36e3702aab158074d33bec05e14d12f2c2442675c2629ae`
- A reviewer-owned alternate-index snapshot of the managed worktree produced the exact candidate tree OID. The recomputed binary patch matched the attached patch byte-for-byte, and `git diff --check` passed.

## F1 — Foreign-evidence health refusal is unproved

Severity: blocking test-evidence gap.

Negative shape: **prove a bound by narrowing, not only by deleting it**. Revision 5 correctly adds both `status.LegacyCount == 0` and `status.ForeignCount == 0` to the `WithinPolicy` predicate in `PiLifecycleStatus`, but the tests exercise only the legacy half. No test in `tools/agents-infra` names or asserts `ForeignCount`/`foreign_count`.

Production call site: `runPi -> runPiLifecycleCLI -> PiLifecycleOperatorStatus -> PiLifecycleStatus`, with the attestation at `tools/agents-infra/internal/infra/pi_session_log.go:2109`.

The reviewer removed only `&& status.ForeignCount == 0` through a Go overlay, leaving the legacy condition and every other health condition intact. With `-count=1` and task-scoped HOME/cache plus the repository-local file proxy:

- `go test . -run '^TestRunPiLifecycle' -count=1 -overlay ...`: exit 0 (`0.899s`).
- The focused status/continuation/retirement/automatic-nonmutation/eight-week-soak suite: exit 0 (`8.464s`).

The narrowed mutant therefore survives every relevant shipped test. This is not evidence that current production behavior is wrong; it is evidence that the foreign-evidence attestation can regress while all claimed gates stay green. The prior revision-4 verdict explicitly required foreign evidence through the same production status entry point, and the live reviewer checklist requires negative tests for attesting behavior.

Required revision 6:

- Add a production-entry CLI status fixture containing foreign legacy-tree evidence and assert a fresh complete scan reports the explicit foreign count while both `within_policy` and `soak_ready` remain false.
- Assert that the status path preserves the foreign evidence and remains non-launching/non-mutating.
- Run an uncached narrowed mutant that removes only the foreign-count clause and require the named production-entry test to fail; then restore production and rerun the focused lifecycle gates.
- Keep the existing legacy pre-retirement and post-retirement assertions unchanged.

## Reviewer validation

- Original candidate `go test . -run '^TestRunPiLifecycle' -count=1`: exit 0 (`1.512s`).
- Original focused status/continuation/retirement/automatic-nonmutation/eight-week-soak suite: exit 0 (`6.533s`).
- Attached Change Request validation records full `go test ./...` and `go vet ./...` success; these were accepted as producer evidence and not rerun wholesale.
- No Pi executable, installed runtime, model/provider process, service, socket, endpoint, setup/install flow, network proxy, or user-HOME runtime/config state was contacted. All reviewer commands used deterministic filesystem fixtures and task-scoped caches.
- No tracked repository source, test, documentation, or logbook file was modified by the reviewer. The overlay, mutant copy, and transcripts are ignored `.temp/` evidence only.
