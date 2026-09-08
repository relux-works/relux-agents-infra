# TASK-260829-1qh0ud revision 3 implementation report

## Scope and result

Revision 3 preserves the revision-2 resource-pressure/status contract on base
`675f77ed63376320ed1213f46f9462a299c0abaf` and closes the independently
reproduced stale provider-observation race. No live model, user-owned runtime,
service, socket, or endpoint was probed or mutated; every new behavior test uses
the deterministic fake provider through the real broker wire handler.

## Implementation

- `sharedBrokerServer` assigns a monotonic generation before each provider read.
- A completion superseded by a newer read returns versioned
  `unknown/refused` facts with reason `resource_observation_stale`; it cannot
  mutate the pressure latch.
- `acquireLease` rechecks the generation under the broker mutex and reserves the
  lease in that same critical section. A newer observation therefore cannot
  enter between admission validation and ownership creation.
- Pressure/recovery events carry the generation that changed the latch. The
  serve loop ignores a delayed event when a newer latch transition owns state.
- Existing leases and the single broker-owned runtime remain untouched by a
  pressure refusal. The revision-2 read-only status recovery rule and
  `draining` precedence remain intact.
- README, runtime skill guidance, and LOGBOOK now document the concurrency and
  explicit-unknown contract.

Production sites:

- `tools/agents-infra/internal/infra/pi_shared_broker_darwin.go:996`
  (`handleConnection -> acquireLease` admission and atomic reservation)
- `tools/agents-infra/internal/infra/pi_shared_broker_darwin.go:1055`
  (monotonic provider observation application)
- `tools/agents-infra/internal/infra/pi_shared_broker_darwin.go:816`
  (generation-bound pressure/recovery event application)

## Negative evidence

The new tests drive `sharedBrokerServer.handleConnection`, not the classifier
helper:

- `TestSharedBrokerProductionStaleRecoveryObservationCannotClearNewerPressure`
  blocks generation 1, applies generation 2 pressure, then releases generation
  1 and requires refusal, a preserved latch, zero leases, and no new runtime.
- `TestSharedBrokerProductionStaleStatusObservationCannotLaunderNewerPressure`
  repeats the same ordering through the wire `status` surface and requires
  `unknown/refused`, never `healthy/admitted`.
- `TestSharedBrokerProductionSupersededAdmissionCannotGrantBeforeLeaseReservation`
  pauses after a healthy read has completed but before lease reservation, then
  applies newer pressure and proves the post-observation gate refuses.

Observed expected-red evidence:

- Pre-fix production race: exit 1; the stale request received `Type: "lease"`.
- Adjacent-generation narrowing (`!=` changed so generation N is accepted after
  N+1): exit 1; both acquire and status tests reproduced laundering.
- Post-observation reservation-gate narrowing only: exit 1; the paused stale
  admission received `Type: "lease"`.
- After each mutant, the production file was restored and compared byte-for-byte
  with its pre-mutant copy before the green rerun.

## Validation and real exit codes

| Command | Exit | Evidence |
| --- | ---: | --- |
| `go test ./internal/infra -run '^TestSharedBrokerProductionStaleRecoveryObservationCannotClearNewerPressure$' -count=1 -v` before fix | 1 | `go-test-stale-observation-red-rev3.log` |
| Three stale-generation production tests under `go test -race` after both mutant restores | 0 | `go-test-stale-generation-race-post-mutants-rev3.log` |
| Adjacent-generation narrowed mutant, acquire + status | 1 (expected red) | `go-test-stale-adjacent-generation-mutant-rev3.log` |
| Lease-reservation narrowed mutant | 1 (expected red) | `go-test-lease-reservation-generation-mutant-rev3.log` |
| `go test -p 1 ./internal/infra -count=1` | 0 (`153.133s`) | `go-test-infra-serial-rev3.log` |
| Relevant root runtime/Pi/status subset | 0 (`447.587s`) | `go-test-root-resource-relevant-rev3.log` |
| `go test -p 1 ./cmd/model-harness ./internal/attachments ./internal/modelharness -count=1` | 0 | `go-test-short-packages-rev3.log` |
| `go vet ./...` | 0 | `go-vet-all-rev3.log` |
| `go build ./...` on Darwin arm64 | 0 | `go-build-darwin-rev3.log` |
| `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ./...` | 0 | `go-build-linux-amd64-rev3.log` |
| `CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ./...` | 0 | `go-build-windows-amd64-rev3.log` |
| `gofmt -d` on changed Go files | 0, empty diff | direct gate |
| `git diff --check` | 0 | direct gate |

Two broader commands are deliberately not reported as passing. A foreign
`go test -p 1 ./...` overlapped on the host. The combined remaining-package run
was interrupted at 8:36 and the unfiltered root-only rerun at 9:23, both with
exit 1 and no package result, to respect the headless single-command bound.
Their raw logs are retained. The changed package, relevant root consumers,
short packages, vet, and all supported compile shapes were then executed as
bounded standalone commands with the exits above.

## Review focus

Attack both generation checks independently, event-generation comparison,
read-only status recovery, provider read failures, unsupported-platform
`unknown` shapes, and preservation of `RestartNotBefore`, `HalfOpen`, and
`Resources` on the current-trunk status surfaces.
