# TASK-260829-1qh0ud implementation report — revision 2

## Outcome

Revision 1 was replayed semantically on current trunk `675f77ed63376320ed1213f46f9462a299c0abaf`. The additive restart status fields `restart_not_before` and `half_open` remain present beside the new `resources` handoff on Darwin, POSIX-unsupported, and Windows-unsupported shapes.

The review defect is fixed at the production status/admission boundary. `sharedBrokerServer.handleConnection` status remains read-only: a recovery-threshold sample cannot publish `healthy/admitted` while the broker still enforces a pressure latch and has pressure eviction armed. It publishes `pressured/refused` until `sharedBrokerServer.acquireLease` applies recovery and the serve-loop cancels eviction. Broker `draining` remains the stronger state.

The resource-pressure vertical slice now includes:

- explicit `disabled|provider` configuration with no implicit mode;
- strict timeout, pressure/recovery hysteresis, grace, and action validation;
- bounded strict provider observation with explicit unknown semantics;
- protocol v7 and versioned observation/status JSON fixtures;
- typed healthy, busy, pressured, draining, and unknown status;
- pressure/unknown refusal without disturbing existing connection-bound leases;
- final-release drain/eviction and same-runtime recovery without duplicate ownership;
- current-trunk restart status composition in operator and unsupported-platform projections.

No user-owned live runtime, model, endpoint, or socket was probed or mutated. All provider/runtime evidence came from disposable deterministic test fixtures.

## Review-defect attack evidence

Production call sites under test:

- `sharedBrokerServer.handleConnection` `status` case;
- `sharedBrokerServer.handleConnection` `acquire` case;
- `sharedBrokerServer.acquireLease` recovery transition;
- `sharedBrokerServer.serve` pressure eviction event owner.

`TestSharedBrokerProductionStatusCannotBypassLatchedPressureRecovery` starts with a pressure latch and eviction armed, obtains a recovery sample through the real wire status entry, proves the published status/latch/broker state remain coherent, then acquires through the real wire path and proves recovery cancels the old eviction deadline while preserving the same runtime PID and one lease owner.

A narrowed production mutant changed preservation from all latched recovery to `brokerState == "serving"` only. The named test exited 1 and reproduced `broker state=pressured` with a hypothetical recovered resource handoff. The exact pre-mutant file was restored byte-for-byte; the test then exited 0.

## Validation and real exit codes

Final-state green gates:

- full serial module suite, all tests and no skips, `go test -p 1 ./... -count=1`: exit 0;
- focused resource/config/production tests, uncached: exit 0;
- focused production resource tests under `-race`: exit 0;
- `go test ./internal/infra -count=1` excluding exactly the two unrelated host-timing tests named below: exit 0 (`278.667s`);
- root package, attachments, and modelharness packages in the final full run: exit 0;
- `go vet ./...`: exit 0;
- `go build ./...` on Darwin: exit 0;
- `GOOS=linux GOARCH=amd64 go build ./...`: exit 0;
- `GOOS=windows GOARCH=amd64 go build ./...`: exit 0;
- `git diff --check`: exit 0;
- changed-Go-file `gofmt -l` gate: exit 0;
- narrowed status-bypass mutant: exit 1, expected red;
- restored status-bypass regression test: exit 0.

Full-suite history is reported without relabeling failures:

- `go test ./... -count=1` before the final draining-precedence self-review: exit 0;
- final `go test ./... -count=1`: exit 1. Only these unrelated unchanged host-timing tests failed:
  - `TestPinnedPiNoModelDirectRPCBashBypassesToolCallHookWhileStandaloneExcludesRPC`: fixed 500ms stdin-close fixture produced no RPC output/side effect;
  - `TestPiLaunchReadinessServiceUnavailableStillHonorsRuntimeBoundsAtProductionEntry/refuses_after_owned_runtime_exits`: fixed 1s fixture returned `runtime_readiness_timeout` instead of observing child exit.
- isolated `-count=3` reruns of each timing test: exit 1;
- later isolated single reruns of each timing test: exit 1.
- final serial all-package run with no skips: exit 0. Serial package execution removed the suite's own package-level contention without omitting either timing test.

Read-only `mac-load-profile snapshot` recorded 929 processes, concurrent foreign Go integration binaries, and repository searches above 177% CPU. No foreign process was stopped. The two red timing runs are retained as failures in evidence rather than relabeled; their exact full-run log is attached. The later serial command executed every test and exited 0, so the final all-tests gate is green without a skip or fixture change.

Early regression-test development runs are also recorded honestly:

- regression run 01: exit 1 because the fixture did not mark a prior lease and first-lease grace drained the broker;
- regression run 02: exit 1 because the helper overwrote the intended one-second eviction grace with zero;
- corrected regression runs 03 and 04: exit 0.

## Changed surface

- `README.md`, `SKILL.md`, `LOGBOOK.md`;
- `tools/agents-infra/internal/infra/pi_config.go`;
- shared broker/client/operator/protocol and unsupported-platform status files;
- shared-runtime config fixtures across existing tests;
- `pi_shared_resources.go`, `pi_shared_resources_test.go`;
- versioned resource observation/status JSON fixtures.

## Handoff

Implementation matches the task acceptance criteria and is ready for independent review. Review should retain the parallel full-suite exit 1 as reported host-timing history, verify the later no-skip serial full-suite exit 0, and independently attack the wire status/admission latch boundary.
