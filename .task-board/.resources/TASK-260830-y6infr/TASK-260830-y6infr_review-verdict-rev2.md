# Review verdict: CR-TASK-260830-y6infr-2 revision 2

## Verdict

CHANGES REQUESTED. Route `TASK-260830-y6infr` to `to-dev`.

Reviewed base `4270549dd17c010599e2083bf3ec7672af60ea29`, candidate tree
`7653c1af0cc48fcf8d0e1bc9894c3a9bc9277584`, and patch SHA-256
`9f754f582272efe2f0fa597bf765590621485b1edafe6f1457db4ea71ea664ba`.
An alternate-index reconstruction of the current worktree produced the exact
candidate tree. The upstream dependency resolves to immutable pseudo-version
`v0.5.1-0.20260830114459-046baef11790` with no `replace`.

## Blocking findings

### F1 — Pi JSONL translation still admits missing and duplicate lifecycle events

`tools/agents-infra/internal/infra/pi_turn_result.go:162-265` tracks only the
outer agent lifecycle and the existence of some assistant `message_end`. It
does not track whether a turn is open, whether a message was started, or
whether a second `turn_start` arrived while the first turn was still open.

A reviewer-only overlay drove the real `parsePiTurnJSONL` entry and required
each malformed stream to be refused. All three were admitted as success with
final text `"admitted"`:

- missing `turn_start`;
- missing `message_start`;
- duplicate `turn_start`.

Command:

```text
go test -overlay .temp/TASK-260830-y6infr-review/parser-overlay.json \
  ./internal/infra -run '^TestReviewerAttackTurnAndMessageLifecycle$' -count=1
```

Exit `1`, because every negative assertion failed. Full output is preserved in
`.temp/TASK-260830-y6infr-review/parser-attack-02.log` for this run.

This is a narrowed **bypass path around the check**. Revision 2 closed the two
revision-1 examples but still does not implement the pinned turn/message state
machine required by the missing/duplicate-event attack matrix. A schema-1
success can therefore be minted from an incomplete or contradictory raw Pi
event stream.

Required rework: model the closed Pi v0.84.2 turn and message lifecycle, refuse
missing/reordered/duplicated start/end events, and add production-translation
negative tests plus narrowing mutants for each state transition.

### F2 — Fake Process A and fake-backed Process B are still not composed through one production graph

`tools/agents-infra/internal/infra/pi_shared_engine_observation_darwin_test.go:96-107`
drives the real Process-B reader directly and expects it to refuse. The test
then switches at lines `112-118` to `managementGraphFixture` with
`recordingSanitizedObservationReader` before calling `BuildAndRunPiTurn`.
Consequently the Process-A launch/classifier path consumes self-minted
observation facts, not the observation read from the fake-backed broker that
the test established. The broker and its independent lease merely remain alive
beside an unrelated fake Process-A process.

The targeted test passes, but it does not reproduce the capability claimed by
AC3 and the task DoD: the real Registry/BuildLaunch/observation/child-launch
path never composes fake Process A and fake Process B in the same execution,
and no Process-A-owned shared-runtime lease is acquired/released or attacked.

Negative shape: **capability claim that does not reproduce**, with a remaining
**forged or self-minted evidence** seam at `recordingSanitizedObservationReader`.

Required rework: drive the exact production graph with a fake-backed broker
whose sanitized facts permit BuildLaunch, then execute/cancel fake Process A
through `BuildAndRunPiTurn`; assert the Process-A lease lifecycle and prove
cleanup signals only Process A while the independently held Process-B peer
lease/runtime survive unchanged.

## Evidence anomaly

`TASK-260830-y6infr_mutants-rev2.log` shows the expected failing test output but
prints `exit=0` for each killed mutant. The mutation behavior is visible, but
the script loses the real exit status (consistent with reading `$?` after a
negated command). Fix the evidence script so published exit-code claims match
the process that actually ran.

## Verification performed

- Exact CR patch: 164654 bytes; SHA-256 match.
- Alternate-index candidate reconstruction: exact tree match.
- `git diff --check <base> <candidate>`: exit 0.
- Upstream `go list -m -json`: exact immutable pseudo-version and commit suffix.
- Focused contract/adversarial suite: exit 0.
- Real fake-backed broker test: exit 0; static inspection above shows the graph split.
- Full uncached `env -u TASK_BOARD_DIR go test -mod=mod ./... -count=1`: exit 0.
- `go vet ./...`, `go build ./...`, focused `go test -race`: exit 0.
- Cross-builds: linux/amd64, windows/amd64, darwin/amd64: exit 0.
- Reviewer lifecycle overlay: exit 1 because all three invalid streams were admitted.
- No real model, external service, production socket, or user configuration was contacted.

The green baseline does not override F1 or F2: the shipped negative suite omits
the turn/message lifecycle states, and the Process-B test substitutes a fake
observation reader before entering the production child-launch path.
