# TASK-260826-fcu5pe revision 5 witness and landing evidence

## Candidate identity and scope

- Candidate commit: `91adc7328d6a122fbbbb40f42a1d9b6aad5f2ac0`
- Candidate tree: `40a83fe6f3b1544494969edc861f3fe23ffc4757`
- Accepted rev4 input commit: `727a45b019ba2ee98a75577565e6e8f5a1f212a1`
- Current local `main`: `e70f953969d46e451892d9f16e7401b879910b6b`
- Candidate/main graph at validation: `25` candidate-only commits / `4` main-only commits.

Reviewer `RUN-260826-6697f3` certified the rev4 reconciliation and every
overlapping current-main blob; that fixed input was not redone. The delta from
rev4 to this candidate is exactly five paths: three test files, `LOGBOOK.md`,
and one operator production file. No task-board Pi adapter or standalone yolo
path changed. Integration must remain a merge, not a tree reset, because the
Story checkout's deletions relative to main are board artifacts.

The only non-test implementation delta is a private, immutable per-call
dependency helper behind `stopRecordedSharedRuntime`. A literal test-only
witness cannot synthesize a kernel-observed root UID, and one executable
identity cannot naturally have the same inode on a different device. The
production wrapper passes the exact original system functions on every call;
there is no mutable package global and the race suite is green.

## Required negative witnesses

- `stopRecordedSharedRuntime`: PID-reuse start time, recorded argv, observed
  root UID paired with the root record, and same-inode/wrong-device executable
  identity all refuse with `broker_stop_identity_mismatch` before the injected
  signal boundary.
- `sharedRecordedBrokerIsLive`: a reused PID/start-time mismatch reports
  `unverified-stale` and leaves the live process untouched.
- Client attestation: empty `profile_digest` refuses; absent
  `effective_sharing` returns `protocol_violation` rather than panicking.
- Launcher: empty `schema`, empty `runtime_key`, and a recomputed runtime key
  that differs only after a shared prefix refuse at the production entry.

## Clean candidate validation

Every command was a standalone foreground process. Redirection only captured
stdout/stderr; no command was piped through `tee`.

| Command | Exit | Result |
| --- | ---: | --- |
| `cd tools/agents-infra && gofmt -l .` plus empty-output assertion | 0 / 0 | No unformatted module Go files |
| `git diff --check` | 0 | Clean committed tree |
| focused production-entry/oracle/calibration suite | 0 | `24.696s` |
| `go test ./internal/infra -count=1 -run '^(TestSharedRuntime\|TestSharedAuthorization\|TestConnectAndAttestSharedRuntime\|TestRunSharedRuntimeBroker\|TestReclaimSharedRuntime)'` | 0 | `36.050s` |
| same focused suite with `-race` | 0 | `55.351s` |
| `go vet ./...` | 0 | Configured landing vet gate |
| `go build ./...` | 0 | Darwin |
| `GOOS=linux GOARCH=amd64 go build ./...` | 0 | Linux |
| `GOOS=windows GOARCH=amd64 go build ./...` | 0 | Windows |
| `go test ./... -count=1` | 0 | Root `144.625s`; attachments `3.205s`; infra `200.399s` |

## Standalone mutant attribution

Each mutant was created from candidate `91adc73`, compiled by its named test,
and run with `-count=1`. Every test process exited `1`. A separate attribution
check exited `0` after requiring exactly the named failure and rejecting any
`valid` subtest failure.

| Mutant | Exit | Required named witness |
| --- | ---: | --- |
| client empty-profile exemption | 1 | `empty_profile_digest` |
| client missing-sharing guard deletion | 1 | `hello_effective_sharing_absent` |
| launcher empty-schema exemption | 1 | `schema_empty` |
| launcher empty-key exemption | 1 | `runtime_key_empty` |
| launcher first-eight-key prefix comparison | 1 | `runtime_key_argument_differs_only_after_shared_prefix` |
| operator start-time deletion | 1 | `pid_reuse_start_time` |
| operator argv deletion | 1 | `recorded_argv` |
| operator root-UID exemption | 1 | `recorded_uid_root` |
| operator executable device deletion | 1 | `broker_executable_same_inode_wrong_device` |
| status start-time deletion | 1 | `TestSharedRuntimeStatusMarksReusedRecordedBrokerPIDUnverifiedStale` |

## Full-repository composite

The final scratch tree was copied from candidate `91adc73`. It simultaneously:

- reduced the operator kernel identity gate to live/current-UID only;
- narrowed executable identity to inode only;
- removed status start-time binding;
- exempted empty client profile digest;
- removed the client `effective_sharing` completeness guard; and
- prefix-compared only the first eight launcher runtime-key characters.

`go build ./...` exited `0`. `go test ./... -count=1` exited `1` as required:
root (`162.802s`) and attachments (`5.404s`) stayed green; infra (`236.366s`)
failed only on the two client witnesses, three operator witnesses, status
witness, and launcher prefix witness. The exact-count attribution check exited
`0`; no `valid` subtest failed. The root-UID mutant is not part of this six-group
composite and is covered by its standalone named run above.

## Excluded development runs

- A record-only root-UID test initially exited `1` because production correctly
  ignores the record UID and reads the kernel UID. This demonstrated why the
  observation seam is necessary; it is not a passing gate.
- The first module-only composite exited `1` on missing repository fixtures and
  a package panic. It is excluded. The full-repository composite above replaces
  it.
- One focused run wrote an `ok` line but its execution handle was lost after a
  yield. Its exit is treated as unknown and excluded; the counted focused run
  above reran the same command and exited `0`.
- A repository-root format assertion exited `1` because `gofmt -l .` traversed
  historical and deliberately mutated `.temp` fixtures. It is excluded; the
  project module-scoped format command and empty-output assertion both exited
  `0`.

## Handoff

The accepted current-main reconciliation, 13-gate attestation chain,
runtime-launch authorization, shared lease semantics, mutation calibration,
CLI contract, standalone yolo refusal, and deferred task-board adapter boundary
remain intact. Candidate `91adc73` is suitable for independent review and
Story integration by merge.
