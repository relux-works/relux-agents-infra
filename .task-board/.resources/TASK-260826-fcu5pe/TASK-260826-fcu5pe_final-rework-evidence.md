# TASK-260826-fcu5pe final rework evidence

## Candidate identity

- Candidate commit: `7ef425820de3174b535907fbc085a091be1de8c0`
- Candidate tree: `e3b84df47ed9c1dec2404fe4146dac12ed56a509`
- Current main: `e70f953969d46e451892d9f16e7401b879910b6b`
- Story fork point: `b3cb84550a60f7f4df92a287c573bfc692cd26e0`
- Divergence at handoff preparation: 4 commits behind main, 23 ahead.
- Worktree status after commit: clean.
- Commit author and committer time: `2026-08-25T23:59:00+03:00`, matching the repository backdating policy.

The independently accepted current-main reconciliation was not redone. The
delta from reviewed revision 2 commit `63bd5381` is limited to:

- `LOGBOOK.md`
- `tools/agents-infra/internal/infra/pi_shared_broker_admission_test.go`
- `tools/agents-infra/internal/infra/pi_shared_broker_darwin.go`
- `tools/agents-infra/internal/infra/pi_shared_launcher_test.go`

No task-board Pi adapter, standalone Pi yolo policy, Pi config, Pi plan, CLI
entrypoint, or accepted client/launcher attestation predicate changed.

## Reviewer rev2 refusal closure

`sharedBrokerServer.attestClient` now reads kernel/process evidence through a
server-owned dependency bundle while retaining the exact production predicates.
The new negative table drives that production admission function and covers:

- peer UID with a root narrowing witness;
- announced PID with a zero-value narrowing witness;
- zombie liveness;
- same-inode/wrong-device executable identity;
- future protocol-version range narrowing;
- empty runtime key and profile digest.

Every refusal asserts its exact `SharedRuntimeError` code, a zero observation,
and zero broker leases. The same test has a valid-evidence control.

Additional production surfaces are attacked directly:

- `sharedBrokerServer.handleConnection` receives a valid JSON hello of exactly
  65,537 bytes over a real AF_UNIX connection and must answer
  `protocol_violation` without a lease;
- `sharedBrokerServer.acquireLease` must refuse a draining broker before
  constructing or retaining a lease;
- the forked `runtime broker` entry receives a syntactically valid runtime key
  sharing 63 of 64 characters with the recomputed key and must refuse before
  runtime/rendezvous side effects;
- `reclaimSharedRuntime` receives forged root UID and zero PGID evidence and
  must refuse before its process-group signal seam is reached.

Launcher positive controls now wait for the actual exec marker/process
observation with a 15-second load-safe bound. Ten repeated clean launcher
control/guard runs completed in `11.439s` with exit 0.

## Strict mutation calibration

The task-scoped scratch harness copies the Go module for every mutant, replaces
exactly one source predicate, and requires all three facts:

1. `go build ./...` exits 0;
2. the uncached focused `go test -json` exits 1;
3. the expected named test or subtest emits its own `Action=fail` event.

`mutant-run-02` killed 23/23 compile-valid delete-or-narrow mutants under that
rule. Coverage includes every broker admission comparison, draining refusal,
wire-bound deletion and widening, broker runtime-key deletion and prefix-only
narrowing, and reclaim UID/PGID deletion and zero-value narrowing. Every
constituent mutant test exited 1 for the expected-red reason: the mutated guard
admitted evidence that its named production test requires it to reject. The
group harness exited 0. The attached mutation log records each real build/test
exit and failing test name.

The earlier `mutant-run-01` exited 1 because two delete mutants did not compile
after making evidence variables unused. They were rejected, not counted. The
replacement forms retain an explicit `_ = evidence` while deleting the guard;
both compile and are killed in `mutant-run-02`.

## Clean candidate validation

Every gate below ran as a standalone foreground process without `tee` or a
background handoff.

| Command | Exit | Result |
| --- | ---: | --- |
| `gofmt -l .` | 0 | empty output |
| `git diff --check` | 0 | clean |
| focused new broker/reclaim suite, 10 repetitions | 0 | `1.128s` |
| focused launcher control/guard suite, 10 repetitions | 0 | `11.439s` |
| focused shared-runtime suite | 0 | `23.064s` |
| focused shared-runtime race suite | 0 | `37.587s` |
| production-entry/oracle/calibration suite | 0 | `17.023s` |
| `go test ./... -count=1` | 0 | root `77.627s`; attachments `1.028s`; infra `115.575s` |
| `go vet ./...` | 0 | configured landing gate |
| `go build ./...` | 0 | Darwin |
| `GOOS=linux GOARCH=amd64 go build ./...` | 0 | Linux |
| `GOOS=windows GOARCH=amd64 go build ./...` | 0 | Windows |
| scoped `git add`, cached diff check, and commit | 0 | four intended paths only |

Expected development-red runs, reported rather than laundered as passes:

- initial tests-first compile run: exit 1 because the production dependency
  seams did not yet exist (and two test-only struct comparisons needed repair);
- first post-seam focused run: exit 1 because the test AF_UNIX path exceeded the
  Darwin socket-path bound; the fixture moved to a short `/tmp` path;
- witness-strengthening run: exit 1 on one stale unused import, then the exact
  focused command passed after removal;
- mutation run 01: harness exit 1 because two compile-invalid mutants were
  refused as evidence; immutable run 02 supersedes it and exits 0.

## Scope conclusion

The revision binds the broker-side mirror of the accepted 13-gate client
attestation chain and the reclaim authorization gates without weakening shared
lease semantics, runtime-launch authorization, CLI behavior, or the accepted
current-main composition. The candidate is suitable for an independent
`story_final` review and later orchestrator integration.
