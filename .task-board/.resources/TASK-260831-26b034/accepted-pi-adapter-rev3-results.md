# Revision 3 producer evidence

Closes the two blocking findings and the evidence anomaly from
`TASK-260830-y6infr_review-verdict-rev2.md` (CR-TASK-260830-y6infr-2 revision 2,
base `4270549`, candidate tree `7653c1af`).

## F1 — Pi JSONL turn/message lifecycle was still bypassable

### What was wrong

`parsePiTurnJSONL` in `tools/agents-infra/internal/infra/pi_turn_result.go`
tracked only the outer `agent_start`/`agent_end` lifecycle plus "some
assistant `message_end` exists" (`sawFinalAssistantMessage`, closed in
revision 2 for a different finding). It had no state for whether a turn was
currently open, whether a message was currently open, or whether a second
`turn_start` arrived while the first turn was still open. A reviewer overlay
driving the real `parsePiTurnJSONL` entry point admitted all three attacks
(missing `turn_start`, missing `message_start`, duplicate `turn_start`) as
success with final text `"admitted"`.

### The fix

Added two closed booleans, `turnOpen` and `messageOpen`, threaded through
every turn/message-scoped event case in the same switch statement that already
enforced the outer agent lifecycle:

- `turn_start` refuses if a turn is already open (`turnOpen`); otherwise opens
  it.
- `turn_end` refuses unless a turn is open and no message is still open
  (`!turnOpen || messageOpen`); otherwise closes it.
- `message_start` refuses unless a turn is open and no message is already open
  (`!turnOpen || messageOpen`); otherwise opens it.
- `message_update` and `message_end` both refuse unless a turn is open and a
  message is open (`!turnOpen || !messageOpen`); `message_end` closes it.
- `agent_end` additionally refuses while a turn is still open (`turnOpen`).
- The trailing structural EOF check additionally refuses if `turnOpen` or
  `messageOpen` is still true when the stream ends (defense in depth on top of
  the per-event gates above).

Production call site: `tools/agents-infra/internal/infra/pi_turn_result.go`,
`parsePiTurnJSONL`, the sole raw-Pi-to-schema-1 translator invoked from
`classifyPiTurn` / `RunPiTurnProcessA`.

### Reproduced reviewer attacks and additional invariants

`TestPiTurnTranslatorRefusesTurnAndMessageLifecycleViolations`
(`pi_turn_result_test.go`) drives `parsePiTurnJSONL` directly with seven cases,
each a single targeted change to the otherwise-valid fixture:

1. missing `turn_start` (reviewer attack 1)
2. missing `message_start` (reviewer attack 2)
3. duplicate `turn_start` (reviewer attack 3)
4. `turn_end` without an open turn (duplicate `turn_end`)
5. duplicate `message_start` while the first message is still open
6. `turn_end` while its message is still open
7. `agent_end` while a turn is still open

All seven now return a non-nil error (refused) against the fixed code; all
seven returned `nil` (admitted, `final_text: "admitted"`-equivalent) against
the pre-fix code shipped in revision 2.

## F2 — Fake Process A and fake-backed Process B were not composed in one production graph

### What was wrong

`pi_shared_engine_observation_darwin_test.go` drove the real Process-B reader
(`SharedRuntimeSanitizedEngineObservationReader`) directly against a real
(fake-backed) broker and asserted a real refusal, then **switched** to
`managementGraphFixture` with `recordingSanitizedObservationReader` — a
self-minted, always-succeeding fake — before calling `BuildAndRunPiTurn`. The
Process-A launch/classifier path therefore consumed self-minted observation
facts, not the observation read from the broker the test had just
established. The broker/lease stayed alive beside an unrelated fake
Process-A run; no Process-A-owned lease lifecycle was exercised or attacked.

### The fix

- `tools/agents-infra/internal/infra/pi_shared_integration_test.go`: the
  shared fake-runtime source template gained an unconditional
  `/agents-infra/resources` HTTP handler returning a genuine, schema-`v1`
  provider resource observation. It is inert for every pre-existing test
  (all use `resource_pressure_mode = "disabled"`, which never queries this
  endpoint).
- `tools/agents-infra/internal/infra/pi_shared_engine_observation_darwin_test.go`:
  the profile now configures `resource_pressure_mode = "provider"` with an
  absolute `--model` weight path (needed for the weight-artifact fact and
  compatible with the existing readiness-matching string comparison). The
  real `SharedRuntimeSanitizedEngineObservationReader`, bound to the real
  fake-backed broker, now returns a genuine successful observation (asserted
  directly) instead of a refusal.
- The test builds `PiPluginGraph` via the production `BuildPiPluginGraph`
  constructor directly (not the `managementGraphFixture` test double),
  passing this exact real reader. `BuildAndRunPiTurn` is driven twice (success
  and mid-flight cancellation) through this exact graph with a fake Process A
  script — the observation the classifier's launch decision is built on now
  comes from the same broker this test independently established, closing the
  self-minted-evidence seam.
- A second, independent, real shared-runtime lease is acquired directly
  through the production `acquireSharedRuntimeLease` client (the same call
  `RunPi`'s shared-mode path uses), representing the lease lifecycle a real
  (non-fake) Process A would hold. `SharedRuntimeStatusReport` is read before
  acquisition (1 lease), while it is held alongside the pre-existing
  Process-B peer lease (2 leases, both surviving Process-A's fake success and
  cancelled runs unchanged), and after its release (back to 1 lease, broker
  still `serving`, same runtime PID) — proving Process-A activity and its own
  lease acquire/release lifecycle never disturb the independently-held peer
  lease or the runtime.

Production call site: `tools/agents-infra/internal/infra/agents_management_registry.go`
`BuildPiPluginGraph` and `tools/agents-infra/internal/infra/agents_management_process_a.go`
`BuildAndRunPiTurn`, both driven for real by
`TestSharedRuntimeEngineObservationReaderReadsRealBrokerAndProcessASpawnsNeverTouchProcessB`.

`recordingSanitizedObservationReader` remains defined for the platform-generic,
no-live-runtime unit tests in `agents_management_contract_test.go` /
`agents_management_consumer_test.go` (which have no broker to attach to and
are explicitly meant to be hermetic/static); it is no longer used on this
darwin real-broker production-graph path.

## Evidence anomaly — mutant log exit codes

`TASK-260830-y6infr_mutants-rev2.log` printed `exit=0` for every entry
regardless of the underlying test's real pass/fail outcome (including a
compile-failure case that clearly needed a non-zero exit).
`.temp/TASK-260830-y6infr/mutants.sh`, as currently checked into this task's
`.temp` working directory, was inspected line by line: `run_mutant()` captures
`local code=$?` as a plain statement immediately after `go test ...`, with no
intervening command and no negation (`!`) anywhere in the invocation chain.

To confirm this exact script's exit-code capture is correct rather than
assume it, two direct reproductions were run against it in isolation:

- A `go test -run <nonexistent-name>` (no tests match, real exit 0):
  `run_mutant` logged `exit=0` — correct.
- A deliberately introduced Go syntax error in `pi_turn_result.go` (forces a
  real build failure, real exit 1): `run_mutant` logged `exit=1` — correct.

The published rev2 log's `exit=0`-for-everything pattern therefore does not
reproduce against the exact script present now; it must have been produced by
an earlier, different revision of the script (the rev2 log's raw-output
format — unlabelled test output followed by a bare `exit=0` line for most
mutants, with no `name=`/`test=` prefix on mutant 1 at all — does not match
the current script's `MUTANT_KILLED name=... test=... exit=$code` format
either, corroborating that it was a different generator). No behavioral change
to `run_mutant` was needed or made; the fix that was required — and is applied
here — is republishing a freshly regenerated, verified-correct log
(`TASK-260830-y6infr_mutants-rev3.log`) from this exact, verified-correct
script, so the published exit codes are guaranteed to be the real status of
the process that ran for every mutant in this revision, including the 6 new
F1 mutants below.

## New narrowing mutants for the F1 lifecycle guards (18 total, all killed)

Mutants 1-11 are unchanged from revision 2 (production-plane composition,
tool-failure/exit-laundering, exact-profile, stale/identity observation
guards); all still kill cleanly. Mutants 12-18 are new for F1:

| # | Mutant | Attack it must admit | Test | Real exit |
| - | --- | --- | --- | --- |
| 12 | `turn_start` drops its "already open" check | duplicate `turn_start` | `.../duplicate_turn_start` | 1 (killed) |
| 13 | drop `!turnOpen` from every turn-scoped event (`message_start`, `message_update`, `message_end`, `turn_end`) | missing `turn_start` | `.../missing_turn_start` | 1 (killed) |
| 14 | drop `!messageOpen` from every message-content event (`message_update`, `message_end`) | missing `message_start` | `.../missing_message_start` | 1 (killed) |
| 15 | `turn_end` drops its `!turnOpen` check (keeps `messageOpen`) | `turn_end` without an open turn | `.../turn_end_without_turn_start` | 1 (killed) |
| 16 | `turn_end` drops its `messageOpen` check, plus the trailing EOF `messageOpen` backstop | `turn_end` while message open | `.../turn_end_while_message_open` | 1 (killed) |
| 17 | `agent_end` drops its `turnOpen` check, plus the trailing EOF `turnOpen` backstop | `agent_end` while turn open | `.../agent_end_while_turn_open` | 1 (killed) |
| 18 | `message_start` drops its "already open" check | duplicate `message_start` | `.../duplicate_message_start` | 1 (killed) |

Mutants 13, 14, 16, and 17 are compound (touch more than one line) by
necessity: the state machine has deliberate defense-in-depth (e.g. every
turn-scoped event independently re-checks `turnOpen`, and the trailing EOF
check independently re-checks both flags), so narrowing only one of several
redundant checks for these specific attack shapes leaves the attack caught by
a neighboring, unrelated check and the mutant survives (this was verified by
hand — an earlier attempt at single-line mutants 13-17 against the original
custom-stream fixtures produced exactly this false "survived" outcome, which
is why the fixtures for tests 4/6/7 above were redesigned as targeted
single-event insertions/duplications against `validPiTurnJSONL`, and mutants
13/14/16/17 narrow the complete redundant set for their fixture). Mutants 12,
15, and 18 are true single-line narrowings, isolated because the violation in
their fixture is detected at the exact mutated line with no earlier or later
check able to catch the same condition.

Full transcript: `TASK-260830-y6infr_mutants-rev3.log`, produced by
`.temp/TASK-260830-y6infr/mutants.sh` (unchanged from what's described above).

## Validation (real exit codes)

- `go build ./...`: exit 0.
- `go vet ./...`: exit 0.
- `go test . -count=1`: exit 0 (89.1s).
- `go test ./internal/infra -count=1`: exit 0 (157.3s).
- `env -u TASK_BOARD_DIR go test -mod=mod ./... -count=1`: exit 0, every
  package (`.`, `cmd/model-harness`, `internal/attachments`, `internal/infra`,
  `internal/modelharness`).
- `go test ./internal/infra -race -count=1 -run
  'TestPiTurnTranslator|TestSharedRuntimeEngineObservationReader|TestConsumer|TestPiPluginGraph|TestObservationPlane|TestGenericPlanes|TestProcessBLifecycle'`:
  exit 0 (12.2s).
- Cross-builds `go build ./...`: linux/amd64, windows/amd64, darwin/amd64 —
  all exit 0.
- `gofmt -l tools/agents-infra`: zero output, exit 0.
- `git diff --check 4270549 -- .`: exit 0.
- `.temp/TASK-260830-y6infr/mutants.sh`: 18/18 narrowing mutants
  `MUTANT_KILLED` (real exit 1), 1 discovery-scope control
  `DISCOVERY_NARROWING_ADMITTED` (real exit 0, expected).
- No real model, external service, production socket, or user configuration
  was contacted. The darwin F2 test's "runtime" and its
  `/agents-infra/resources` endpoint are a test-built Go HTTP stub, launched
  only under this test's own temp `HOME`/project directories; the second
  shared-runtime lease is acquired against that same test-local broker.

## Rev2 evidence superseded by this revision

- `TASK-260830-y6infr_mutants-rev2.log` is superseded by
  `TASK-260830-y6infr_mutants-rev3.log` (corrected exit codes, plus the 6 new
  F1 mutants 13-18; renumbered from the reviewer's F1/F2 finding order).
- The `TestSharedRuntimeEngineObservationReaderReadsRealBrokerAndProcessASpawnsNeverTouchProcessB`
  entry in `TASK-260830-y6infr_results-rev2.md` (F2 section) is superseded by
  the F2 section above: the rev2 version of this test still substituted
  `recordingSanitizedObservationReader` right before `BuildAndRunPiTurn`; the
  rev3 version does not.
- `pi_turn_result.go`'s state machine described in `TASK-260830-y6infr_results-rev2.md`
  (F3 section, `sawFinalAssistantMessage` and post-`agent_end` tool guards) is
  still correct and unchanged; F1 above is additive to it, not a replacement.

## Not run, stated plainly

- Independent review, PR, and merge gates belong to the review and delivery
  roles.
- The stable upstream `skill-agents-management` release tag and its final pin
  remain out of scope for this checkpoint (AC7); `TASK-260830-u8nd0b` owns
  that.
