# TASK-260826-fcu5pe review verdict — CR-TASK-260826-fcu5pe-3 rev3

Reviewer run: `RUN-260826-c56491` (claude-opus-5)
Verdict: **accepted** (`accept_cr`, revision 3)

Candidate: commit `7ef425820de3174b535907fbc085a091be1de8c0`, tree `e3b84df47ed9c1dec2404fe4146dac12ed56a509`
Base: `b3cb84550a60f7f4df92a287c573bfc692cd26e0` · current main: `e70f953969d46e451892d9f16e7401b879910b6b`
Candidate worktree clean and unmodified throughout this review (`git status --short` empty at start and end).
Every mutant ran in `/tmp/rev3-mut/`, never in the candidate tree.

The rev2 refusal — the broker's own client-admission chain unbound — is **closed**,
and I re-derived that by attack rather than by reading the rework note. Every row
rev2 recorded as SURVIVED now dies under both its delete and its narrowing form,
and rev2's composite (nine admission comparisons neutralised in one build, whole
package green) is now killed by nine named subtests. No new instance of the class
was found on the surfaces I attacked.

---

## 1. Published candidate is the tree I reviewed

- `git rev-parse HEAD^{tree}` = `e3b84df47ed9c1dec2404fe4146dac12ed56a509` = the CR's declared candidate tree.
- `git diff <base> <candidate>` reproduced byte-for-byte to sha256
  `2b0f8e07…96ec976`, identical to the published
  `TASK-260826-fcu5pe_change-request_rev3.patch`. The patch is the delta, not a summary of it.

## 2. Scope of rev2 → rev3 — exactly the four claimed paths

`git diff --name-only 63bd538 HEAD` yields precisely `LOGBOOK.md`,
`pi_shared_broker_admission_test.go` (new), `pi_shared_broker_darwin.go`,
`pi_shared_launcher_test.go`.

Blob-hash identity confirms the accepted §1 work is untouched, not merely
"unchanged in the stat": `pi_shared_client_darwin.go`, `pi_shared_attestation_test.go`,
`pi_shared_launcher_darwin.go`, `pi_shared_operator_darwin.go`,
`pi_shared_protocol.go`, `pi_shared_shape_oracle_test.go`,
`pi_shared_integration_test.go`, `main.go`, `pi_config.go`, `pi_state.go` all have
identical object IDs at `63bd538` and `HEAD`. rev2's §1 certification therefore
still applies without re-running it.

The production change in `pi_shared_broker_darwin.go` replaces **evidence sources
only** — `sharedBrokerAdmissionDependencies` (peer identity, process inspection,
exec identity) and `sharedRuntimeReclaimDependencies` (kernel/process inspection,
kill, wait). Every predicate is character-for-character the one rev2 named.
`newSharedBrokerServer` is the single production construction site
(`pi_shared_broker_darwin.go:297`) and it wires the real bundle; the test uses that
same constructor and overrides only the evidence functions. No nil-seam fallback,
no admission bypass.

## 3. The rev2 refusal — closed, verified by my own mutants

Method: copy the module to a scratch tree, patch exactly one predicate,
`go build ./...` (compile failure ⇒ invalid mutant, discarded), then
`go test ./internal/infra -count=1 -json`. A kill counts **only** when the
expected **named** test/subtest emits its own `Action=fail` — exit code alone was
never accepted. 22/22 compile-valid mutants killed by the named witness.

| # | Gate (production site) | Mutant | Killing named test |
| --- | --- | --- | --- |
| m01 | `attestClient` peer uid `:816` | delete | `…RejectsEveryGateDeleteAndNarrowWitness/peer_uid_root_narrowing` |
| m02 | same | `&& uid != 0` | same |
| m03 | announced client PID `:821` | delete | `/announced_client_pid_zero_narrowing` |
| m04 | same | `ClientPID != 0 &&` | same |
| m05 | client liveness `:825` | delete `!observation.live()` | `/client_zombie_narrowing` |
| m06 | client exec identity `:836` | delete | `/client_executable_same_inode_wrong_device` |
| m07 | same | `Ino` only, `Dev` dropped | same |
| m08 | protocol version `:839` | delete | `/future_protocol_version_range_narrowing` |
| m09 | same | `<` instead of `!=` | same |
| m10 | runtime key `:842` | delete | `/empty_runtime_key` |
| m11 | same | `!= "" &&` | same |
| m12 | profile digest `:845` | delete | `/empty_profile_digest` |
| m13 | same | `!= "" &&` | same |
| m14 | `acquireLease` draining `:857` | delete | `TestSharedBrokerAcquireLeaseRefusesDrainingBeforeGrant` |
| m15 | `readSharedWireMessage` bound `:956` | delete | `TestSharedBrokerProductionConnectionRejectsWireFrameBoundWidening` |
| m16 | same | `> max+1` widening | same |
| m17 | broker recomputed key `:184` | delete | `TestRunSharedRuntimeBrokerRefusesRecomputedRuntimeKeyAtProductionEntry` |
| m18 | same | first-8-chars prefix compare | same |
| m19 | `reclaimSharedRuntime` uid `:436` | delete | `…BeforeSignal/reclaim_uid_root_narrowing` |
| m20 | same | `&& UID != 0` | same |
| m21 | reclaim pgid `:445` | delete | `…BeforeSignal/reclaim_pgid_zero_narrowing` |
| m22 | same | `PGID != 0 &&` | same |

**Composite.** All nine `attestClient` / `acquireLease` / `readSharedWireMessage`
comparisons neutralised in a single build, whole package run
(`go test ./internal/infra -count=1 -run .`):

```
go build ./...   -> BUILD_OK
go test          -> exit 1, killed by 9 named subtests
```

rev2 recorded this exact composite as `ok … 87.026s (exit 0)` with the broker's
admission wide open. It is now red for the right reason.

**Production entry, not helpers.** The seven evidence witnesses drive
`server.attestClient`, which `handleConnection` calls at
`pi_shared_broker_darwin.go:830`; the frame-bound witness drives
`handleConnection` itself over a real AF_UNIX socket pair with a 65,537-byte
valid-JSON hello; the recomputed-key witness forks the real
`runtime broker` subcommand and additionally asserts the refusal left no
`broker.json` and no rendezvous socket. Every refusal asserts the exact
`SharedRuntimeError` code, a zero observation, and zero leases — refusal proven by
the protected state not being reached, not only by an error string.

**Calibration harness (rev2 item 3).** The producer's named-failure rule is the
right fix and it is load-proof in the way that matters: when a control did flake
during my composite run (§5), my driver classified it `RED_WRONG_TEST`, not a
kill. An exit-code-only harness would have recorded a false kill there.

## 4. Reconciliation — re-derived, unchanged from rev2

- `git merge-base main HEAD` = `b3cb845`; branch is 4 behind / 23 ahead. Those 4 are
  main's two "Record board state" commits plus its two story merges, whose product
  content is carried on the branch as `3d203b1` / `5e3bbe0`.
- `git diff --name-status main HEAD | grep '^D'` → **38 deletions, all under
  `.task-board/`, zero source deletions.** Those artifacts are absent because the
  branch forked before they existed, not because it removed them.
  **Integration must be a merge, not a tree reset.**
- Non-board paths differing from main are exactly the broker delta plus
  `LOGBOOK.md`, `README.md`, and the research spec. `model_check*`, `SKILL.md`,
  `canonical_target*`, `pi_run_report.go`, `pi_platform_windows.go` do not appear in
  `git diff --stat main HEAD` at all — the overlap was composed, not clobbered.
- No task-board Pi adapter path and no standalone yolo policy path is touched
  anywhere in `git diff --name-only main HEAD`.
- README gains only additive `runtime status` / `runtime stop` / `runtime.sharing`
  documentation; no main content is displaced.

## 5. Validation — every command rerun by this reviewer, foreground, unpiped exits

| Command | Exit | Time |
| --- | ---: | ---: |
| `gofmt -l .` (empty output) | 0 | — |
| `git diff --check` | 0 | — |
| `go vet ./...` (configured landing gate) | 0 | — |
| `go build ./...` (darwin) | 0 | — |
| `GOOS=linux GOARCH=amd64 go build ./...` | 0 | — |
| `GOOS=windows GOARCH=amd64 go build ./...` | 0 | — |
| `go test ./... -count=1` (configured landing gate) | 0 | 161.5s / 6.0s / 250.1s |
| `go test -race ./internal/infra -run '^(TestShared\|TestConnectAndAttestSharedRuntime\|TestRunSharedRuntimeBroker\|TestReclaimSharedRuntime)'` | 0 | 58.2s |
| `go test ./internal/infra -count=1` on a pristine scratch copy, ×2 | 0 | 88.5s / 90.6s |
| `go test ./internal/infra -count=8 -run '^TestSharedRuntimeLauncher'` under 10 CPU hogs | 0 | 35.9s |

## 6. Residual observations — recorded, not blocking

**6.1 The launcher positive control is still wall-clock, not event-driven.**
rev2 item 4 offered "a bound that does not fail under sweep load, **or** retry";
the producer took the bound (3s → 15s, `pi_shared_launcher_test.go:23`).
It held under every clean condition I could construct, including 8 repetitions of
the whole launcher suite against 10 competing CPU hogs (§5). It failed **once**, at
exactly 15.02s, in
`…RejectsAuthorizationChannelGuardBypassesAtProductionEntry/descriptor_three_is_a_socket_rather_than_a_fifo`,
inside the §3 composite run — i.e. only with the broker's admission gates removed,
a state that never ships. Two clean full-package runs and the landing gate were
green. Because the harness now requires the named failure, this flake can no longer
manufacture a false kill; it can only cost a rerun. Worth converting to an
observation-driven wait the next time this file is opened.

**6.2 `stopRecordedSharedRuntime` uid check is redundant under its own successor.**
Narrowing `brokerObservation.UID != uint32(os.Geteuid())` to
`… && UID != 0` (`pi_shared_operator_darwin.go:441`) survives the package
(`s02_stop_uid_narrow_root`, exit 0). I attacked it and it opens no hole: any record
naming a root process still fails the executable-identity comparison two lines down,
which **is** bound — deleting that comparison is killed by
`TestSharedRuntimeForceStopRefusesForgedBrokerIdentityWithoutSignal`
(`s01`, exit 1). Observationally equivalent defence-in-depth, same category rev2
dismissed for the re-frozen report constant. Not a finding.

**6.3 Non-darwin fail-closed is proven by construction, not by test.**
`pi_shared_unsupported_posix.go` / `pi_shared_unsupported_windows.go` return
`shared_runtime_platform_unsupported` from every shared entry point including
`runSharedPiSession`, and both cross-compile clean. No darwin test can execute them;
stated as a known coverage boundary rather than inferred as covered.

**6.4 Environment handling on the shared path is bound by the pre-existing gate.**
`scrubSharedRuntimeEnvironment` has no direct test, but it is defence-in-depth:
`ValidatePiExecutionEnvironment` refuses `DYLD_*`, `LD_*`, `NODE_*`, `BUN_*`,
`LLAMA_ARG_*`, `HF_ENDPOINT`, `GGML_BACKEND_PATH` at `pi_launch_posix.go:110`
before `runSharedPiSession` is reached and again at
`pi_shared_client_darwin.go:609`, and that gate has negative tests in `pi_test.go`.

## 7. Reviewer reproduction

Mutant catalog, driver, and per-mutant `go test -json` logs:
`/tmp/rev3-mut/` (`mutants-gates.json`, `composite.json`, `drive.py`, `logs/`).
Consolidated table attached as
`TASK-260826-fcu5pe_review-mutation-log-RUN-260826-c56491.txt`.

## 8. Handoff

Accepted via `accept_cr(TASK-260826-fcu5pe, revision=3, evidence=TASK-260826-fcu5pe_review-verdict-RUN-260826-c56491.md)`,
which parks the element at `to-review`. The orchestrator integrates the accepted
revision **as a merge** (§4) and makes the `done` transition with
`commit_ack=scope_committed`. No `commit_ack` supplied by this reviewer.
