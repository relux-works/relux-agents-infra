# TASK-260829-1qh0ud revision 5 review verdict

## Verdict

**Changes requested -> `to-dev`.** Change Request `CR-TASK-260829-1qh0ud-5` revision 5 is not accepted.

Review identity:

- base OID: `6d051f54440d36e3ca3d132f8d9d1e78d46289de`
- candidate tree OID: `bd49d59fed93ad7b11a425cedf8728b78dda16b0`
- patch SHA-256: `ba5072d68fb1e7075df9bda5927f44ee8515113ef0fa2711afecec23c5852718`
- repository delta: `present`
- independently reconstructed worktree tree: `bd49d59fed93ad7b11a425cedf8728b78dda16b0`

## Acceptance-blocking finding: record fallback launders effective provider policy into disabled/not-enforced

Severity: correctness defect in AC1 and the versioned consumer handoff.

The attested live-broker status surface publishes resources from the effective broker policy, but the unreachable/stale-record production fallback does not. `sharedRuntimeRecordStatus` constructs `report.Resources` from caller `resolved.Sharing` at `tools/agents-infra/internal/infra/pi_shared_operator_darwin.go:281`, then publishes the record's different effective policy at lines 285-289. `disabledSharedRuntimeResourceStatus` turns that caller policy into `reason=resource_pressure_disabled` and `admission=not-enforced` at `pi_shared_resources.go:102-118`.

Because sharing is deliberately outside runtime identity, this is a normal reachable composition. A configured-disabled caller can inspect a persisted record fixed by a provider-policy broker. The exact production `SharedRuntimeStatusReport` response then contradicts itself:

- `sharing.configured.resource_pressure_mode = disabled`
- `sharing.effective.resource_pressure_mode = provider`
- `broker.state = unverified-stale`, source `record-derived-unverified`
- `resources.policy.mode = disabled`
- `resources.reason = resource_pressure_disabled`
- `resources.admission = not-enforced`

The provider policy and pressure admission were not proved disabled; they were copied from the caller as a proxy. This violates explicit unknown semantics and gives a consumer a false non-enforcement fact precisely on the non-attested fallback surface.

Negative-evidence shapes: **bypass path around the check** (attested status is effective-policy-bound, record fallback is not) and **prove, or report nothing** (configured policy is used as a proxy for effective policy).

### Attack evidence

A scratch-only test drove `SharedRuntimeStatusReport -> sharedRuntimeRecordStatus` with configured disabled and record-effective provider. Candidate code failed expected-red:

```text
go test ./internal/infra -run '^TestReviewAttackRecordDerivedStatusDoesNotLaunderEffectiveProviderPolicy$' -count=1 -v
--- FAIL: TestReviewAttackRecordDerivedStatusDoesNotLaunderEffectiveProviderPolicy (0.02s)
resources: source=record-derived-unverified policy.mode=disabled reason=resource_pressure_disabled admission=not-enforced
sharing: configured=disabled effective=provider
FAIL .../internal/infra 0.451s
```

Full output: `TASK-260829-1qh0ud_record-status-attack-rev5.log`.

## Negative gate coverage gap: complete policy-table equality is not mutation-resistant

Revision 5 implements exact struct equality, but the production mismatch test changes pressure and recovery thresholds together. In a scratch copy, equality was narrowed to compare only `PressureThresholdBytes`. All three named `handleConnection -> acquireLease` mismatch cases still passed, including the alleged complete-table witness:

```text
go test ./internal/infra -run '^TestSharedBrokerProductionResourcePolicyMismatchRefusesBeforeObservationOrLease$' -count=1 -v
PASS
ok .../internal/infra 0.430s
```

This surviving narrowed mutant can admit a different recovery threshold, observation path/timeout, or eviction grace. The current positive result therefore does not prove the documented complete-table gate. Full output: `TASK-260829-1qh0ud_policy-narrowing-mutant-rev5.log`.

## Required rework

1. Make record-derived resource status use one coherent policy provenance. If `record.Sharing` is present, the resource policy view must not be synthesized from caller configuration. Preserve `record-derived-unverified` and unknown observation facts; report effective provider admission conservatively, or represent the policy itself as explicitly unknown if it cannot be established. Do not emit `resource_pressure_disabled/not-enforced` unless disabled enforcement is actually established for that surface.
2. Add deterministic production-entry tests through `SharedRuntimeStatusReport` for both configured/effective mismatch directions and for a pre-extension record with no sharing field. Prove no combination publishes mutually contradictory `sharing.effective`, `resources.policy`, reason, and admission fields.
3. Strengthen the real `handleConnection -> acquireLease` negative table so single-field differences in the provider policy table are refused. At minimum, a recovery-threshold-only difference must kill the pressure-threshold-only mutant; preferably enumerate every independently variable field.
4. Add the record-fallback root cause/fix/evidence to `LOGBOOK.md` during producer rework. The reviewer did not mutate the immutable candidate under the read-only role.

## Independent validation

All positive gates were rerun on the exact candidate and are green; they do not override the reproduced negative-path defect.

| Gate | Result |
| --- | --- |
| Exact worktree tree reconstruction | `bd49d59...` equals candidate tree |
| Patch digest | matches `ba5072d...` |
| `git diff --check <base> <candidate>` | exit 0 |
| Focused production pressure/status/policy suite under `-race -count=1` | exit 0, package 2.860s |
| `go test ./... -count=1` | exit 0; root 96.922s, attachments 2.229s, infra 172.052s, modelharness 1.038s |
| `go vet ./...` | exit 0 |
| `go build ./...` | exit 0 |
| Linux amd64 cross-build | exit 0 |
| Windows amd64 cross-build | exit 0 |
| `gofmt -d internal/infra/*.go *.go` | exit 0, empty output |

Cross-platform type shapes compile and retain `Resources`, but Darwin's record-derived public status remains acceptance-blocking.

## Logbook handling

The `logbook` skill was consulted after discovering the regression. Its write contract conflicts with the reviewer role's immutable/read-only candidate constraint, so this reviewer did not alter `LOGBOOK.md`; required rework item 4 assigns the entry to the next producer revision.
