# TASK-260826-fcu5pe review verdict — CR-TASK-260826-fcu5pe-2 rev2

Reviewer run: `RUN-260826-6b7916` (claude-opus-5)
Verdict: **changes_requested → `to-dev`**

Candidate: commit `63bd5381aafa86212f40bbd3aada0acb71fb4b6d`, tree `685a204677ec4b137415cbded9441d50902fbde6`
Base: `b3cb84550a60f7f4df92a287c573bfc692cd26e0` · current main: `e70f953969d46e451892d9f16e7401b879910b6b`
Repository delta: `present` (35 paths vs base; 24 non-board paths vs current main)
Candidate worktree clean and unmodified throughout this review (`git status --short` empty at start and end).

The rev1 refusal is **fully closed** — every gate it named is now bound, and I
re-derived that independently rather than reading the rework note. The candidate
is refused on a **new** instance of the same class the rev1 refusal named, on the
surface the rev1 review did not attack: the **broker's own client-admission
chain**. Nine of its comparisons can be deleted in a single build with the whole
`internal/infra` package green.

---

## 1. rev1 refusal — closed, verified by attack

Method: copy `tools/agents-infra` to a scratch tree, patch exactly one
comparison, `go build ./...` (compile failure ⇒ invalid mutant, discarded), then
`go test ./internal/infra -count=1 -run <regex>`. A kill counts **only** when the
expected named negative subtest is the one that failed — an exit code alone was
not accepted (see §3).

### 1.1 Client attestation chain — 21/21 mutants killed by the named subtest

Every row the rev1 verdict recorded as SURVIVED now dies, and the narrowings die
with the deletions:

| Gate | Delete mutant | Narrow mutant | Killing subtest |
| --- | --- | --- | --- |
| `peer_uid` | killed | `&& uid != 0` killed | `peer_uid_root_narrowing` |
| `peer_pid_liveness` | killed | — | `peer_zombie_narrowing` |
| `broker_executable` | killed | `Ino` only, `Dev` dropped killed | `broker_executable_same_inode_wrong_device` |
| `broker_build` | killed | — | `broker_build_same_inode_wrong_device` |
| `broker_start_time` | killed | — | `broker_start_time_zero` |
| `protocol_version` | killed | `<` instead of `!=` killed | `future_protocol_version_range_narrowing` |
| `runtime_key` | killed | `!= "" &&` killed | `empty_runtime_key` |
| `profile_digest` | killed | — | `profile_digest` |
| `endpoint` | killed | `!= "" &&` killed | `empty_endpoint` |
| `runtime_executable` | killed | — | `runtime_executable_same_inode_wrong_device` |
| `runtime_process` uid | — | `&& UID != 0` killed | `runtime_process_uid_root_narrowing` |
| `runtime_process` start time | killed | — | `runtime_process_start_time_zero` |
| `runtime_process` exec path | killed | — | `runtime_process_empty_executable_path` |
| `runtime_process` argv | killed | — | `runtime_process_argv` |
| `runtime_liveness` | killed | — | `runtime_zombie_narrowing` |
| `model_discovery` | killed | — | `model_discovery` |

The `sharedRuntimeAttestationSystem` seam replaces only the *evidence source*
(kernel peer identity, process inspection, file identity, model probe). The gate
logic under test is the production one, and `connectAndAttestSharedRuntime` is
driven directly. That is the right shape for forging kernel evidence.

### 1.2 Launcher authorization channel — 10/10 killed by the named subtest

FIFO-only descriptor 3, recomputed-key equality, trailing content after the
newline, the 65536-byte bound (delete **and** `*2` widening), and all five frame
comparisons including both zero-value narrowings (`launcher_pid == 0`,
`exec_plan_digest == ""`). Every attack has a plain-valid control and asserts
`carried_target=false` plus an absent exec marker, so refusal is proven by the
target not being reached, not only by an error string.

### 1.3 Operator attestation report — now derived, and bound at both surfaces

`passedSharedRuntimeGateOutcomes()` is gone. `connectAndAttestSharedRuntime`
appends each `SharedRuntimeGateOutcome` only after the corresponding production
check passes, and `pi_shared_operator_darwin.go:112` copies that list into
`SharedRuntimeStatus.Attestation`.

- Dropping a single `passed("endpoint")` while keeping its check → killed by
  `TestConnectAndAttestSharedRuntimeReportsOnlyTheExactPassedGateSet`.
- The §3.3 composite — delete the endpoint check **and** re-freeze the report to
  a hardcoded 13-entry constant so it still claims all thirteen attested → killed
  by `empty_endpoint`.
- `requireExactSharedRuntimeAttestation` is asserted both on `attested.gates`
  and on `status.Attestation` at the operator surface
  (`pi_shared_integration_test.go:351`).

Re-freezing the report to the equivalent constant *without* touching any gate
survives, but that mutant is observationally equivalent — the reported set and
the evaluated set are identical when every gate passes — and the composite that
makes it lie is killed. Not a finding.

## 2. Refusal — the broker's client-admission chain is unbound

`sharedBrokerServer.attestClient` (`pi_shared_broker_darwin.go:782`) is the
broker's admission gate for lease acquisition, reached from `handleConnection` in
the forked `runtime broker` production entry. It is the mirror image of the
client chain rev1 refused, and it has **no negative coverage at all**.

| Gate | Source | Delete-mutant |
| --- | --- | --- |
| peer uid | `uid != uint32(os.Geteuid())` (l.784) | **SURVIVED** |
| announced client PID | `hello.ClientPID != pid` (l.790) | **SURVIVED** |
| client liveness | `!observation.live()` (l.793) | **SURVIVED** |
| client executable identity | `clientExec.Dev/Ino != Broker.ExecutableIdentity` (l.804) | **SURVIVED** |
| protocol version | `hello.ProtocolVersion != SharedRuntimeProtocolVersion` (l.807) | **SURVIVED** |
| runtime key | `hello.RuntimeKey != server.resolved.RuntimeKey` (l.810) | **SURVIVED** |
| profile digest | `hello.ProfileDigest != server.resolved.ProfileDigest` (l.813) | **SURVIVED** |
| draining refusal | `server.state == "draining"` in `acquireLease` (l.822) | **SURVIVED** |
| wire frame bound | 65536-byte check in `readSharedWireMessage` (l.925) | **SURVIVED** |
| broker recomputed key | `resolved.RuntimeKey != options.RuntimeKey` (l.185) | **SURVIVED** |
| reclaim uid | `observation.UID != uint32(os.Geteuid())` (l.403) | **SURVIVED** |
| reclaim pgid | `full.PGID != runtimeRecord.PGID` (l.413) | **SURVIVED** |

**Composite proof.** All nine `attestClient` / `acquireLease` /
`readSharedWireMessage` comparisons neutralised in one build:

```
go build ./...                       -> BUILD_OK
go test ./internal/infra -count=1    -> ok  ... 87.026s   (exit 0)
```

The entire package passes with the broker's client admission wide open: any
same-uid process — regardless of executable, liveness, protocol version, runtime
key, or profile digest — is admitted and granted a lease on the shared runtime,
and a draining broker still hands out new leases.

Two of these are the *same rule* the launcher already enforces with a test, and
the asymmetry is the tell:

- the launcher's 65536-byte frame bound is killed by
  `authorization_frame_is_one_byte_over_the_bound`; the broker's identical wire
  bound has no witness;
- the launcher's `resolved.RuntimeKey != options.RuntimeKey` is killed by
  `runtime_key_argument_differs_from_recomputed_profile`; the broker's identical
  recompute check at `RunSharedRuntimeBroker` has no witness.

That is the "clause present in one context profile and absent from the other"
shape, and the whole class here is positive-path-only: the integration tests
drive well-formed clients from the same test binary, which every one of these
gates admits by construction.

Bound and to be preserved: lease limit `>=` (killed by
`TestSharedRuntimeStarterPolicyGovernsLeaseLimitStatusAndCleanup`), the reclaim
identity-shape gate (killed by
`TestSharedRuntimeReclamationRefusesUnknownShapeAndReapsVerifiedGroup`), the
session-leader gate (killed by `.../not_session_leader`), plus everything in §1
and the sharing-config parser and shape-gate work rev1 already certified.

## 3. Secondary — the calibration harness accepts false kills

The rework evidence states the driver "treated only test exit `1` as a kill". An
exit code is not a kill: it does not establish that the *named* gate's witness is
what failed.

`requireSharedLauncherValidControl` and the `valid` control in
`TestSharedRuntimeLauncherComparesEveryAuthorizationValueAtProductionEntry` wait
on `waitForSharedTest(t, 3*time.Second, ...)` for the forked launcher to exec the
target. On an idle machine that is stable — 0 failures in 8 consecutive
launcher-suite runs. Under the load a mutation sweep creates it times out, and I
hit it seven times during this review (`pi_shared_launcher_test.go:339` and
`:424`, always at exactly 3.01s, "plain valid authorization did not reach the
production launcher target").

This is not cosmetic. The **first** full-package run of the §2 composite exited
`1` — and its only failure was that flake. An exit-code-only harness records that
as a kill and reports the broker chain as bound; the rerun is green and the
mutant survives. Four of my own intermediate results (`b01`, `b03`, `r01`, `r04`)
were false kills for the same reason and had to be re-run to get the truth.

## 4. Validation — reran on the candidate, all green

Every command run by this reviewer in the candidate worktree, foreground, no pipe.

| Command | Exit | Time |
| --- | ---: | ---: |
| `gofmt -l .` (empty output) | 0 | — |
| `git diff --check` | 0 | — |
| `go vet ./...` | 0 | — |
| `go build ./...` (darwin) | 0 | — |
| `GOOS=linux GOARCH=amd64 go build ./...` | 0 | — |
| `GOOS=windows GOARCH=amd64 go build ./...` | 0 | — |
| `go test ./internal/infra -count=1` | 0 | 142.5s |
| `go test . ./internal/attachments -count=1` | 0 | 69.4s / 1.2s |
| `go test -race ./internal/infra -run '^(TestShared\|TestConnectAndAttestSharedRuntime)'` | 0 | 37.8s |

## 5. Reconciliation — still correct, not re-litigated

Re-derived, not accepted from the artifact:

- `git merge-base main HEAD` = `b3cb845`; branch is 4 commits behind main and 22
  ahead. Those 4 are main's two "Record board state" commits plus the two story
  merges; their **product** content is carried on the branch as `3d203b1` /
  `5e3bbe0` and is byte-identical to main.
- `git diff --name-status main HEAD | grep ^D` yields **only** `.task-board/`
  checkout artifacts — zero source deletions. Those artifacts are absent from the
  branch because it forked before they existed, not because it removed them; a
  merge into main preserves them. Integration must be a merge, not a tree reset.
- Non-board paths differing from main are exactly the broker delta plus
  `LOGBOOK.md`, `README.md`, and the research spec. `model_check*`, `SKILL.md`,
  `canonical_target*`, `pi_run_report.go`, `pi_platform_windows.go` are
  byte-identical to main — the overlap was composed, not clobbered.
- Non-darwin profiles fail closed identically: `pi_shared_unsupported_posix.go`
  and `pi_shared_unsupported_windows.go` both return
  `shared_runtime_platform_unsupported` from every shared entry point, including
  `runSharedPiSession`.
- No task-board Pi adapter change; standalone Pi yolo policy untouched.

## 6. Required rework (scope-limited)

Do **not** redo the reconciliation (§5) or disturb §1 — both are verified and
must survive byte-for-byte.

1. Add per-gate negative coverage for every SURVIVED row in §2, driven through a
   real broker connection (the forked `runtime broker` entry, or `attestClient`
   with a seam for the evidence source, mirroring
   `sharedRuntimeAttestationSystem`). Each must assert the specific
   `SharedRuntimeError` code **and** that no lease was granted. Each must fail
   when its gate is deleted *and* when it is narrowed — `uid != geteuid && uid != 0`,
   `Ino` without `Dev`, `!= "" &&` on runtime key and profile digest,
   `<` instead of `!=` on protocol version, `>` instead of `>=` on the wire bound.
2. Cover the broker's own `resolved.RuntimeKey != options.RuntimeKey` recompute
   and the `reclaimSharedRuntime` uid / pgid gates with the same discipline the
   launcher's equivalents already have.
3. Make the calibration harness require the **named** subtest to be the failing
   one, not merely exit `1`, and re-derive the rev2 calibration table under that
   rule. Report any result that changes.
4. Remove the wall-clock dependence from the launcher positive control — wait on
   the marker/exec observation with a bound that does not fail under sweep load,
   or retry the control before declaring it failed. A control that flakes cannot
   certify anything.
5. Rerun the §4 command set and republish the Change Request.

## 7. Reviewer reproduction

Scratch trees, mutant catalog, and per-mutant logs:
`.temp/review-TASK-260826-fcu5pe-rev2/` (candidate worktree never modified).
Consolidated results: `TASK-260826-fcu5pe_review-mutation-log-RUN-260826-6b7916.txt`,
attached as a task-scoped outcome resource.
