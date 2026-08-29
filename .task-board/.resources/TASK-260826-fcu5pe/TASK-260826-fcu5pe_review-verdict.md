# TASK-260826-fcu5pe review verdict — CR-TASK-260826-fcu5pe-1 rev1

Reviewer run: `RUN-260825-1c3f46` (claude-opus-5/high)
Verdict: **changes_requested → `to-dev`**

Candidate: commit `94752fd`, tree `0ad11d216421593781e3f7c67977e5a4a5dfa21d`
Base: `b3cb84550a60f7f4df92a287c573bfc692cd26e0` · current main: `e70f953969d4`
Repository delta: `present` (35 paths vs base, 24 non-board paths vs current main)

The reconciliation itself is correct and fully reproduced. The candidate is
refused on a single class: the shared-runtime **client attestation chain and
launcher authorization channel ship with no negative coverage**, and ten of
their comparisons can be deleted outright with the whole `internal/infra`
package suite green.

---

## 1. Reconciliation — verified correct, reproduced independently

Every claim in `TASK-260826-fcu5pe_reconciliation-and-validation.md` was
re-derived from the repository rather than accepted.

| Claim | Reviewer check | Result |
| --- | --- | --- |
| 18-path overlap between `b3cb845..main` and `b3cb845..HEAD` | `comm -12` over both name lists | exactly 18, matches |
| 11 overlap paths byte-identical to current main | `git diff --exit-code main HEAD -- <path>` per path | 11/11 identical |
| 7 overlap paths deliberately composed | read each `git diff main HEAD -- <path>` | all strictly **additive**; no main-introduced line removed |
| No unrelated main change discarded | `git diff --name-status main HEAD \| grep ^D \| grep -v .task-board/` | **empty** — every deletion vs main is a `.task-board/` checkout artifact, zero source deletions |
| Standalone Pi yolo policy unchanged | read `RunPi` | `validatePiPrimarySessionYolo` still runs at the top of `RunPi`, before profile lookup, argument build, identity verification, and before the new shared dispatch — the shared branch cannot bypass it |
| Broker/launcher spawn matches the CLI contract | compared `pi_shared_client_darwin.go:445` / `pi_shared_broker_darwin.go:336` argv against `main.go` `runRuntime` | flag names and arity agree |
| No task-board Pi adapter change | path audit of the delta | confirmed; delta confined to `tools/agents-infra`, README/SKILL/LOGBOOK, and the research spec |

The branch is 4 commits behind main only in main's two "Record board state"
commits plus their story merges; the two current-main product commits are
carried as `3d203b1` / `5e3bbe0`, and their code content is byte-identical to
main. Composition, not a rebase — deliberate, documented, and correct.

## 2. Validation — reran on the candidate, all green

Not accepted from the artifact; every command below was run by this reviewer in
the candidate worktree.

| Command | Exit | Time |
| --- | ---: | ---: |
| `gofmt -l .` (empty) | 0 | — |
| `git diff --check` | 0 | — |
| `go vet ./...` | 0 | — |
| `go build ./...` (darwin) | 0 | — |
| `GOOS=linux GOARCH=amd64 go build ./...` | 0 | — |
| `GOOS=windows GOARCH=amd64 go build ./...` | 0 | — |
| `go test ./internal/infra -run '^(TestSharedRuntime\|TestSharedAuthorization)' -count=1` | 0 | 12.5s |
| `go test -race ./internal/infra -run '^(TestSharedRuntime\|TestSharedAuthorization)' -count=1` | 0 | 27.5s |
| focused oracle/calibration/production-entry/reject-all suite | 0 | 2.0s |
| `go test ./internal/infra -count=1` | 0 | 95.4s |
| `go test . ./internal/attachments -count=1` | 0 | 71.9s / 1.0s |

## 3. Refusal — the attestation chain is unbound by tests

Attack method: mutants applied to a scratch copy of `tools/agents-infra`
(`.temp/review-TASK-260826-fcu5pe/scratch`, candidate never modified), each
rebuilt and run against `go test ./internal/infra -run '^TestShared'`.
"SURVIVED" = the gate was deleted and the suite stayed green.

### 3.1 `connectAndAttestSharedRuntime` — 10 of 16 comparisons delete silently

`pi_shared_client_darwin.go:282` documents this as "the single production call
site for the 13-gate client attestation chain". It is the same-uid peer
authentication for the AF_UNIX rendezvous socket — the only thing standing
between a client and a hostile local process impersonating the broker.

| Gate | Source | Delete-mutant |
| --- | --- | --- |
| `peer_uid` | `uid != uint32(os.Geteuid())` (l.313) | **SURVIVED** |
| `peer_pid_liveness` | `!peer.live()` (l.320) | **SURVIVED** |
| `broker_executable` | `peerIdentity.Dev/Ino != ownIdentity` (l.334) | **SURVIVED** |
| `broker_build` | `response.Broker.ExecutableIdentity != ownIdentity` (l.363) | **SURVIVED** |
| `broker_start_time` | `Broker.PID != peerPID \|\| Broker.StartTime != peer.StartTime` (l.366) | **SURVIVED** |
| `protocol_version` | `response.ProtocolVersion != SharedRuntimeProtocolVersion` (l.369) | **SURVIVED** |
| `runtime_key` | `response.RuntimeKey != resolved.RuntimeKey` (l.372) | **SURVIVED** |
| `profile_digest` | `response.ProfileDigest != resolved.ProfileDigest` (l.375) | killed |
| `endpoint` | `response.Runtime.Endpoint != resolved.Profile.BaseURL` (l.378) | **SURVIVED** |
| `runtime_executable` | `runtimeIdentity != response.Runtime.ExecutableIdentity` (l.382) | **SURVIVED** |
| `runtime_process` / uid | `runtimeProcess.UID != Geteuid()` (l.393) | **SURVIVED** |
| `runtime_process` / exec path | `runtimeProcess.ExecPath != …Runtime.Executable` (l.393) | **SURVIVED** |
| `runtime_process` / start time | `runtimeProcess.StartTime != response.Runtime.StartTime` (l.393) | killed |
| `runtime_process` / argv | `!equalStrings(runtimeProcess.Argv, wantArgv)` (l.393) | killed |
| `runtime_liveness` | `!runtimeProcess.live()` (l.396) | killed |
| `model_discovery` | `checkSharedRuntimeModel(...)` (l.403) | killed |

Composite proof — all six of `broker_executable`, `broker_build`,
`broker_start_time`, `protocol_version`, `runtime_key`, `endpoint` neutralised
in one build:

```
go build ./...                          -> BUILD_OK
go test ./internal/infra -count=1       -> ok  ... 80.217s   (exit 0)
```

The **entire** `internal/infra` package passes with six named attestation gates
removed. The only two tests that touch this function at all are
`TestSharedRuntimeClientAttestationRejectsProfileRuntimeArgvAndModelDivergence`
(profile digest, kernel argv, model discovery) and
`TestSharedRuntimeForceStopCannotBypassReachableBrokerAttestation`. Everything
else in the chain is positive-path only.

Narrowings survive too, so the bound is not merely "delete-only" untested — the
class is untested:

- `uid != Geteuid() && uid != 0` — a root-owned peer is admitted: SURVIVED
- `peerIdentity.Ino` only, `Dev` dropped — cross-filesystem inode collision: SURVIVED
- `Endpoint != "" && Endpoint != BaseURL` — empty endpoint admitted: SURVIVED

### 3.2 `RunSharedRuntimeLauncher` — authorization channel guards delete silently

| Guard | Source | Delete-mutant |
| --- | --- | --- |
| descriptor 3 must be a FIFO | `stat.Mode&S_IFMT != S_IFIFO` (l.34) | **SURVIVED** — any inherited fd 3 (regular file, socket, tty) becomes the authorization channel |
| recomputed key must equal `--runtime-key` | `resolved.RuntimeKey != options.RuntimeKey` (l.44) | **SURVIVED** |
| no content after the frame's newline | `len(TrimSpace(buffer[newline+1:])) != 0` (l.130) | **SURVIVED** |
| 65536-byte frame bound | `buffer.Len()+count > sharedRuntimeMaxFrameBytes` (l.126) | **SURVIVED** |

`TestSharedRuntimeLauncherComparesEveryAuthorizationValueAtProductionEntry`
alters each of the five compared values to a divergent **non-empty** value. It
does not cover the zero value, and the shape gate admits a present-but-empty
member, so these narrowings survive:

- `frame.ExecPlanDigest == execPlanDigest || frame.ExecPlanDigest == ""`: SURVIVED
- `frame.LauncherPID == os.Getpid() || frame.LauncherPID == 0`: SURVIVED

This is the "absent evidence treated as satisfied" shape applied to the exec-plan
binding.

### 3.3 The operator attestation report is a constant, not a derivation

`passedSharedRuntimeGateOutcomes()` (`pi_shared_client_darwin.go:417`) returns a
hardcoded list of 13 `{"outcome":"passed","source":"attested"}` entries, and
`pi_shared_operator_darwin.go:112` copies it verbatim into
`SharedRuntimeStatus.Attestation`, i.e. into `agents-infra runtime status --json`.
Under the six-gate deletion in §3.1 the report still asserts all thirteen gates
attested-passed. Nothing ties the reported gate set to the gates the function
actually evaluates. Today the list is accurate; by construction it cannot detect
when it stops being accurate — a self-minted attestation on the operator surface.

### 3.4 What *is* well bound (for contrast, and to keep)

The refusal is narrow. Two surfaces in this delta meet the bar and must not be
disturbed by the rework:

- **Sharing config parser** — 4/4 mutants killed by
  `TestParsePiRuntimeSharingIsStrictAndOptIn`: `heartbeat >= lease_stale`
  narrowed to `>`, broker-start-timeout minimum removed, mode widened to
  any-non-empty, `rejectUnknownFields` disabled.
- **Authorization frame shape gate** — an 18-mutant catalog plus a `reject_all`
  harness negative, every one driven through the real `runtime runtime-launch`
  entry in a forked process, each with a plain-valid control frame. This is
  exactly the discipline the attestation chain is missing.

## 4. Required rework (scope-limited)

Do **not** redo the reconciliation. The composition in §1 is verified correct
and must be preserved byte-for-byte; the reconciliation evidence artifact stays
valid.

1. Add per-gate negative coverage for every SURVIVED row in §3.1 and §3.2, driven
   through the production call site (`connectAndAttestSharedRuntime` for the
   chain, the forked `runtime runtime-launch` entry for the launcher), each
   asserting the specific `SharedRuntimeError` code and that no runtime/target
   was reached. Each test must fail when its gate is deleted **and** when it is
   narrowed — mirror the calibration discipline already used for the shape gate
   in §3.4, including a plain-valid control per mutant.
2. Cover the zero-value class at the launcher: `exec_plan_digest: ""` and
   `launcher_pid: 0` must refuse with `runtime_authorization_mismatch` on the
   named field.
3. Derive `SharedRuntimeStatus.Attestation` from the gates actually evaluated in
   `connectAndAttestSharedRuntime`, or bind the constant with a test that turns
   red when a gate is removed from the function. A hardcoded "13 passed" list
   must not outlive the checks it names.
4. Rerun the §2 command set and republish the Change Request.

## 5. Reviewer reproduction

Scratch tree and mutant driver: `.temp/review-TASK-260826-fcu5pe/scratch`
(candidate worktree never modified; `git status --short` clean throughout).
Each mutant: patch one comparison, `go build ./...`, then
`go test ./internal/infra -count=1 -run '^TestShared'`; green output means the
gate is unbound. The composite six-gate build additionally ran the full
`go test ./internal/infra -count=1`.
