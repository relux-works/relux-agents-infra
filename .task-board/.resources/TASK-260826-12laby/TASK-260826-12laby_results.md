# TASK-260826-12laby — developer results

## Outcome

Closed the final Story review's test-evidence gaps without modifying shared-runtime
production code.

- Added same-device/wrong-inode witnesses at all four literal production
  `Dev`/`Ino` comparisons:
  - `sharedBrokerServer.attestClient`
  - `connectAndAttestSharedRuntime`
  - `sharedRuntimeBrokerCandidates`
  - `stopRecordedSharedRuntimeWithDependencies`
- Added below-current protocol-version refusals to the broker, client, and
  runtime-launcher production gates, alongside named exact-version controls.
- Replaced launcher positive-control marker polling and its 15-second deadline
  with an event emitted by the executed test target over stdout. The filesystem
  marker remains a negative-test side-effect witness.
- Recorded closure in `LOGBOOK.md`.

## Negative evidence

Seven scratch-tree narrowing mutants were run independently with `-count=1`.
All exited `1` on the named witness. For each version mutant, the exact-version
control passed in the same expected-red invocation.

Detailed commands, production call sites, and discriminating failures are in
`TASK-260826-12laby_mutation-evidence.md`.

The broker-candidate witness uses two live processes with production-shaped
argv: one executes the exact test binary inode and must be returned; the other
executes a byte copy on the same device with a different inode and must be
excluded. This makes the production scan test non-vacuous without adding a
production dependency seam.

## Validation

All commands below were run directly and reported their real exit codes.

| Gate | Exit | Evidence |
| --- | ---: | --- |
| Four bounded focused changed-scope groups | 0 each | New inode/version witnesses, exact controls, and event controls passed. |
| `go test ./internal/infra -run '^(TestSharedRuntimeLauncherRejectsAuthorizationChannelGuardBypassesAtProductionEntry|TestSharedRuntimeLauncherComparesEveryAuthorizationValueAtProductionEntry)$' -count=10` | 0 | `23.052s`; event control passed ten repetitions. |
| `go test ./internal/infra -run 'SharedRuntime|SharedBroker' -count=1` | 0 | `45.328s`. |
| `go test -race ./internal/infra -run 'SharedRuntime|SharedBroker' -count=1` | 0 | `67.989s`. |
| `go test . -count=1` | 0 | `163.570s`. |
| `go test ./internal/attachments -count=1` | 0 | `2.285s`. |
| `go test ./internal/infra -count=1` | 0 | `333.802s`; prior load-flaky 15s control did not recur. |
| `go build ./...` | 0 | Empty output. |
| `go vet ./...` | 0 | Empty output. |
| `gofmt -l .` | 0 | Empty output. |
| `git diff --check` | 0 | Empty output. |

The three package-level test commands are the bounded full-module equivalent of
`go test ./... -count=1`. An initial over-wide focused invocation yielded before
its exit could be retained and is excluded from evidence; every reported scope
was rerun through bounded standalone commands with explicit exit codes.

## Scope

Repository changes:

- `LOGBOOK.md`
- `tools/agents-infra/internal/infra/pi_shared_attestation_test.go`
- `tools/agents-infra/internal/infra/pi_shared_broker_admission_test.go`
- `tools/agents-infra/internal/infra/pi_shared_integration_test.go`
- `tools/agents-infra/internal/infra/pi_shared_launcher_test.go`

No production file changed.
