# TASK-260829-ivybt9 revision 2 independent review verdict

## Verdict

**Accepted.** Change Request `CR-TASK-260829-ivybt9-2` revision 2 satisfies the replacement-lane contract and closes the revision-1 finding. No acceptance-blocking defect remains.

## Immutable identity

- Base OID: `6d051f54440d36e3ca3d132f8d9d1e78d46289de`.
- Candidate tree: `eb94a41aefe919622f6031450c936df8ffe20ac1`.
- Board patch SHA-256: `b5d3ffd6b9123d459716bcd336300b06d5b0dc4e222bc328b7918c207e79f6d5`.
- Repository delta: present, exactly the 22 declared paths.
- `HEAD`, local `main`, `origin/main`, and GitHub `refs/heads/main` all resolve to the exact base OID.
- Applying the board patch to the base through an alternate index reconstructs the exact candidate tree and passes `git diff --check`.
- Revision 1 to revision 2 changes only `tools/agents-infra/internal/infra/pi_shared_resources_test.go`: 60 inserted test lines and no production-byte change.

## Revision-1 finding closure

The new `TestSharedRuntimeStatusReportRefusesAttestedStatusWithoutResources` drives the production entry point `SharedRuntimeStatusReport`, completes the fake attestation exchange, omits the `resources` evidence from an otherwise valid status response, and requires typed `protocol_violation`.

Reviewer-owned narrowed mutant in an isolated archive:

```go
if message.Resources != nil {
    report.Resources = *message.Resources
}
```

This permissive absent-evidence branch made the named uncached test fail with `error=<nil> want protocol_violation`. The exact candidate then passed the same named test. This closes the standard negative shape **absent evidence treated as satisfied** at the production call site.

## Gate-defeat evidence

Four further reviewer-owned narrowed mutants were applied only in the isolated archive and each reached its intended production path before failing by assertion:

| Boundary | Narrowed mutant | Killing production-entry test |
| --- | --- | --- |
| Final status publication revalidation | Ignore admission-generation drift during publication | `TestSharedBrokerProductionPressureCannotSupersedeStatusBeforePublication` |
| Starvation decoupling | Let every diagnostic status poll invalidate admission | `TestSharedBrokerProductionHealthyStatusPollingCannotStarveAdmission` |
| Strict policy equality | Compare only `PressureThresholdBytes` | `TestSharedBrokerProductionResourcePolicyMismatchRefusesBeforeObservationOrLease` |
| Record-derived provenance | Derive unavailable-resource policy from caller config instead of persisted effective sharing | `TestSharedRuntimeStatusReportRecordDerivedResourcePolicyUsesCoherentProvenance` |

All scratch mutations were restored byte-for-byte against the immutable candidate. One discarded mutant setup failed to compile because of a bad scratch-only field/restore edit; it is not counted as evidence and was corrected before the valid assertion-level reruns.

## Independent validation

| Command / gate | Result |
| --- | --- |
| Named absent-resources production test, `-count=1` | PASS |
| Prior 14-test production resource/status slice, `-race -count=1` | PASS (`3.031s`) |
| Both revision-6 concurrency schedules, `-race -count=20` | PASS (`20/20` each) |
| Provenance, ownership, attestation, restart, quarantine, malformed-read, and recovery composition slice, `-race -count=1` | PASS (`30.477s`) |
| `go test ./... -count=1` | PASS: root `84.297s`, attachments `1.641s`, infra `142.028s`, modelharness `1.655s` |
| `go vet ./...` | PASS, no diagnostics |
| `gofmt -d internal/infra/*.go *.go` | PASS, empty diff |
| `git diff --check <base> <candidate>` | PASS |
| Darwin `go build ./...` | PASS |
| Linux amd64 `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ./...` | PASS |
| Windows amd64 `CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ./...` | PASS |

The focused attestation/ownership slice includes every delete-or-narrow witness on both client and broker production paths, forged and narrowed force-stop identities, persisted manual-quarantine refusal before runtime launch, automatic restart across brokers, restart deadline provenance, and malformed ledger refusal. Linux and Windows builds verify the additive `resources` status field on unsupported-platform surfaces without changing their typed refusal behavior.

## Preservation and scope

- The current managed worktree snapshots exactly to candidate tree `eb94a41aefe919622f6031450c936df8ffe20ac1`; reviewer scratch is ignored and did not enter the tree.
- All 114 historical/control-root hashes from the preservation manifest still pass, including the historical Task/Story resources and CRs, move journal, root dirty `LOGBOOK.md`, and shared instruction files.
- No reparent, checkpoint, integration, historical-lane mutation, or root-dirty-file mutation was performed.
- No live model/runtime/daemon/service/endpoint or user-owned socket was contacted. Tests used static fixtures, fake providers, test subprocesses, and task-owned local transports only. The only external read was the required GitHub branch-ref verification.

## Residual risk

Per the review scope, this verdict is static and fake-backed. It does not claim live provider/runtime interoperability or operational capacity evidence. That exclusion is intentional and is not an acceptance blocker for this Change Request.

## Handoff

Accept revision 2 with `accept_cr`; this parks `TASK-260829-ivybt9` at `to-review` for the Orchestrator's managed Story integration. The reviewer supplies no `commit_ack`.
