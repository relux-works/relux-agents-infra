# TASK-260826-fcu5pe review verdict — CR-TASK-260826-fcu5pe-5 rev5

Reviewer run: `RUN-260826-ae2fa5` (claude-opus-5)
Verdict: **accepted** → `accept_cr`, element parks at `to-review` for the orchestrator's
`done` transition with `commit_ack=scope_committed`. No `commit_ack` supplied by this run.

Candidate: commit `91adc73`, tree `40a83fe6f3b1544494969edc861f3fe23ffc4757`
Base: `b3cb84550a60f7f4df92a287c573bfc692cd26e0` · current main: `e70f953`
Candidate worktree clean and unmodified at the start and the end of this review. Every mutant
ran in `/tmp/rev5-full` (a `git archive` of the candidate), never in the candidate tree.

The previous cycle (`RUN-260826-6697f3`, rev4) requested six specific pieces of evidence and
refused on one thing: the force-stop identity chain was bound by deletion only, and six gate
comparisons could be neutralised in one build with the landing gate still exiting 0. **All six
items are delivered, and I re-derived every kill myself.** The composite that rev4 published
green now exits 1 on exactly the ten expected named witnesses and nothing else.

---

## 1. Published candidate is the tree I reviewed

- `git rev-parse HEAD^{tree}` = `40a83fe6f3b1544494969edc861f3fe23ffc4757` = the CR's declared
  candidate tree.
- `git diff <base> <candidate> | shasum -a 256` =
  `29ad4f2faaede0ea46671054f3a430915845cb7837b8ae770fae1b77a52830f1`, byte-identical to the
  published `TASK-260826-fcu5pe_change-request_rev5.patch` and to the CR's declared hash.
- rev4 → rev5 touches five files: `LOGBOOK.md`, three test files, and **one production file**,
  `pi_shared_operator_darwin.go` (+26/-8). rev4's rework note called for test-only work; the
  production edit is section 4 below and it is behaviour-preserving.

## 2. Reconciliation onto current main — re-derived independently, correct

Derived from git, not read from the rework note.

- `git merge-base HEAD main` = `b3cb845`. Branch is **4 behind / 25 ahead**. Those 4 are main's
  two "Record board state" commits plus its two story merges (`STORY-260825-19dzsf`,
  `STORY-260825-7oqacp`).
- **Main's story content was composed, not clobbered.** `git diff 0a0f8fc 3d203b1` and
  `git diff 945da5e 5e3bbe0` (main's story commits vs the branch's carriers, restricted to
  `tools/ README.md SKILL.md .agents/`) show **only additive broker files** — zero divergence in
  main's own product content. `model_check.go`, `model_check_test.go`, `model_check_main_test.go`,
  `model_check_docs_test.go`, `SKILL.md`, `canonical_target*.go`, `pi_run_report.go`,
  `pi_platform_windows.go`, `pi_operator_docs_test.go` do not appear in `git diff --stat main HEAD`
  at all.
- `git diff --name-status main HEAD | grep '^D'` → **38 deletions, every one under
  `.task-board/`, zero source deletions.** Those board artifacts postdate the fork.
  **Integration must be a merge, not a tree reset.**
- `git diff --name-status HEAD main | grep '^A'` restricted to non-board paths → **empty**. Main
  has no source file the candidate lacks.
- Source delta relative to main is exactly 22 files: the `pi_shared_*` broker set plus its four
  integration points (`pi_config.go`, `pi_launch_posix.go`, `pi_plan.go`, `pi_state.go`) and
  `main.go` / `pi_test.go` / `runtime_main_darwin_test.go`.
- **Scope guards hold.** `git diff --name-only b3cb845 HEAD | grep -iE 'task.?board|yolo|adapter'`
  → nothing. `git diff main HEAD -- tools/ | grep -iE '^[+-].*yolo'` → nothing: yolo policy is
  byte-identical to main. No task-board Pi adapter is added.

Acceptance criterion "the Story candidate contains the accepted broker behavior composed with
current main; all overlaps are resolved deliberately" — met.

## 3. Validation — every command rerun by this reviewer, foreground, exit codes captured

Run in `/tmp/rev5-full`, a `git archive HEAD` extraction verified byte-identical to the candidate
worktree for every tracked file (`diff -rq` differs only in untracked `.temp/` directories), and
verified byte-identical again after every mutant was reverted.

| Command | Where | Exit | Time |
| --- | --- | ---: | ---: |
| `gofmt -l .` (empty output) | candidate | 0 | — |
| `git diff --check` | candidate | 0 | — |
| `go vet ./...` (configured landing gate) | candidate | 0 | — |
| `go build ./...` (darwin) | candidate | 0 | — |
| `GOOS=linux GOARCH=amd64 go build ./...` | candidate | 0 | — |
| `GOOS=windows GOARCH=amd64 go build ./...` | candidate | 0 | — |
| `go test ./... -count=1` (configured landing gate) | `/tmp/rev5-full` pristine | 0 | 122.5s / 4.7s / 203.1s |
| `go test -race ./internal/infra -run '^(TestShared\|TestConnectAndAttestSharedRuntime\|TestRunSharedRuntimeBroker\|TestReclaimSharedRuntime)'` | `/tmp/rev5-full` | 0 | 59.6s |

Green — with the determinism caveat in section 6.4.

## 4. The production change in revision 5 is a minimal, bound seam

`stopRecordedSharedRuntime` now delegates to `stopRecordedSharedRuntimeWithDependencies`, passing
`inspectSharedProcess`, `ownResolvedExecutableIdentity`, `processExecIdentity` and `syscall.Kill`
through an immutable per-call struct. **Not one comparison changed** — the identity chain at
`:459` and `:470` and both signal sites at `:476`/`:478` are character-for-character the rev4
logic. The seam exists because two classes could not be expressed through record inputs: the
kernel-observed UID, and one executable with the same inode on a different device.

I attacked the seam itself, because injectable wiring is a narrowable surface that did not exist
before (mutation log Part C):

- `processExecIdentity` → always return the caller's own identity: **KILLED** by
  `TestSharedRuntimeForceStopRefusesForgedBrokerIdentityWithoutSignal`, which drives the real
  `StopSharedRuntime` production entry. The wiring is bound end-to-end for executable identity.
- `kill` → no-op: **SURVIVED** the whole package (`ok`, 171.3s). See 6.2 — a positive-control
  gap, not a bypass.

A package-global would have been the cheaper seam and would have added a race surface; the
per-call struct is the right call. `LOGBOOK.md` records that reasoning honestly.

## 5. Every rev4 survivor is now killed by a named witness — re-derived, not accepted on report

Ten mutants, each compile-checked, run individually, reverted, and scored KILLED only when the
run exited non-zero **and** the expected named subtest failed. A `valid`/positive-control failure
was scored `RED_WRONG_TEST`, never a kill. Full table in the attached mutation log, Part A.

| rev4 finding | Mutation | Result | Named witness |
| --- | --- | --- | --- |
| §4 O1f | drop recorded broker argv comparison | KILLED | `…NarrowingBeforeSignal/recorded_argv` |
| §4 O2f (PID reuse) | drop recorded broker start time | KILLED | `…NarrowingBeforeSignal/pid_reuse_start_time` |
| §4 O3f | narrow executable identity to `Ino`, drop `Dev` | KILLED | `…NarrowingBeforeSignal/broker_executable_same_inode_wrong_device` |
| §4 uid | drop `brokerObservation.UID != euid` | KILLED | `…NarrowingBeforeSignal/recorded_uid_root` |
| §4 O5f | drop `sharedRecordedBrokerIsLive` start time | KILLED | `TestSharedRuntimeStatusMarksReusedRecordedBrokerPIDUnverifiedStale` |
| §5 | client `profile_digest` exempts the empty value | KILLED | `…RejectsEveryGateDeleteAndNarrowWitness/empty_profile_digest` |
| §6.1 | drop `\|\| response.EffectiveSharing == nil` from `hello_ok` | KILLED | `…/hello_effective_sharing_absent` |
| §5 | launcher `schema` exempts the empty value | KILLED | `…ComparesEveryAuthorizationValueAtProductionEntry/schema_empty` |
| §5 | launcher `runtime_key` exempts the empty value | KILLED | `…/runtime_key_empty` |
| §6.2 | launcher recomputed key compares first 8 chars | KILLED | `…RejectsAuthorizationChannelGuardBypassesAtProductionEntry/runtime_key_argument_differs_only_after_shared_prefix` |

**rev4 §9 item 6 — the composite, re-run and attached red.** All ten applied in one build of the
whole repository:

```
go build ./...           -> BUILD_OK
go vet ./internal/infra  -> VET_OK
go test ./... -count=1   -> COMPOSITE_EXIT=1   (4m44s)
```

The ten named witnesses above are the **complete** failure set. Zero positive-control failures,
zero collateral failures. This is the exact inverse of rev4 §7, where the same six comparisons
were neutralised and `go test ./...` still exited 0.

Two details worth naming because they answer rev4 directly:

- `hello effective sharing absent` wraps the call in a `recover()` and fails with "attestation
  gate panicked instead of refusing" if the clause is gone. rev4's complaint was that removing
  that clause converted a `protocol_violation` refusal into a nil dereference; the witness now
  distinguishes the two outcomes instead of collapsing them into "test failed".
- The launcher recomputed-key witness was strengthened from a wholly different key
  (`strings.Repeat("0", 64)`) to one differing **only in the last character**, which is what makes
  the prefix narrowing die. That is the right shape.

## 6. Findings recorded, none blocking

Everything below is **correct as shipped**. Nothing here is a live bypass, and none of it is in
this task's delta — all four sit in accepted broker behaviour that this task's scope names as a
fixed input. They are recorded so the next cycle does not have to rediscover them, and so the
review record does not claim these surfaces were attacked and cleared.

**6.1 `sharedProcessGone` — "a read failure is not an absence" is unbound at three production
sites.** Replacing `sharedProcessGone(err)` with `(err != nil)` at
`pi_shared_operator_darwin.go:522` (`waitRecordedBrokerGone`), `pi_shared_broker_darwin.go:412`
(`reclaimSharedRuntime`) and `:457` (`waitRecordedRuntimeGone`), in one build, **survives**:
`go test ./internal/infra -count=1` → `ok`, 202.8s (with the section 6.4 flake excluded so the
measurement is clean). The shipped predicate is right — `errors.Is(err, ESRCH) || errors.Is(err,
EIO)`, and any other inspection error is refused as `broker_stop_identity_mismatch` /
`shared_runtime_orphan_unidentifiable`. But nothing proves it: no test injects a non-gone
inspection error. **Failure scenario if it regressed:** an `inspectSharedProcessKernel` failure
that is not ESRCH/EIO would be read as "the process is gone", and `StopSharedRuntime` would
return `BrokerTerminated: true` for a broker that is still running — a capability claim that does
not reproduce. `pi_shared_client_darwin.go:199` carries the same shape and was not mutated by this
pass; I report it as **unknown**, not as bound.

**6.2 `stopRecordedSharedRuntime`'s signal has no positive control.** With `kill` stubbed to a
no-op the whole package stays green (`ok`, 171.3s). The reason is structural:
`StopSharedRuntime` reaches this function **only** through the unreachable-broker fallback, and
the one force-stop success test (`TestSharedRuntimeOperatorStopRefusesActiveLeasesThenForceDrains`)
takes the rendezvous branch instead, so the seam never executes. Severity is low — a suppressed
signal makes `waitRecordedBrokerGone` fail and the error propagates, so the operator sees a failed
stop, not a false success — but no test asserts that `runtime stop --force` against a recorded,
unreachable broker actually terminates it.

**6.3 `sharedRecordedBrokerIsLive`'s euid clause is unbound.** Dropping
`observation.UID == uint32(os.Geteuid())` at `:410` survives. Single caller, `:226`, status
classification only: a recorded PID recycled to another user's process would report as live rather
than `unverified-stale`. Its sibling start-time clause **is** now bound (section 5), so this is one
residual clause, not an open surface.

**6.4 The 15 s launcher positive control is flakier than rev4 §8 estimated, and it makes the
configured landing gate non-deterministic.** rev4 saw one failure and six green repetitions and
called the rate low. On this host I saw **2 failures in ~14 executions of that test on a pristine
candidate tree**, plus a 14.52 s run against a 15.00 s bound
(`sharedLauncherPositiveControlTimeout`, `pi_shared_launcher_test.go:23`). One of the failures
landed inside a mutant run whose mutation is provably unreachable from the launcher path, so it
was scored `RED_WRONG_TEST` rather than a kill. Load average during this review was **228-261**;
the same subtest runs 1.7-2.4 s on a quiet box and 15 s+ under contention. The failure mode is a
**false red**, so it cannot admit a bad change. But the orchestrator should expect
`go test ./... -count=1` to fail intermittently on a loaded host for no code reason, and a
wall-clock positive control inside a mutation-calibration suite can mis-score mutants — rev4 §8
hit exactly that. Converting the wait to observation-driven is still the right fix and now has a
measured rate behind it.

## 7. Why this is accepted rather than reworked again

The charter for this task is reconciliation onto current main plus the rev4 rework list. Both are
delivered, and I verified them independently rather than reading the rework note: section 2
re-derives the reconciliation from git, section 5 re-derives all ten kills and publishes the
composite red. The gates this Story ships — the 13-gate attestation chain, runtime-launch
authorization, the broker admission chain, and now the force-stop identity chain — each die on a
named narrowing witness, not on a delete-only mutant.

The four findings in section 6 are in accepted upstream broker behaviour that this task's scope
names as a fixed input, none is a live defect, and 6.4 can only produce a false red. Holding a
fifth cycle to bind them would reopen a scope this task was told not to reopen. They belong in a
follow-up against the broker implementation, with 6.1 first: it is the one whose regression would
produce a false success rather than a visible failure.

## 8. Reviewer reproduction

`/tmp/rev5-full` (pristine archive of the candidate, restored after every mutant),
`/tmp/rev5-mut/` (`drive.py`, `composite.py`, `cat-*.json` mutant catalogs, `res-*.json` per-mutant
results, `composite.json` raw `go test -json`, `flake-*.json`). Consolidated table attached as
`TASK-260826-fcu5pe_review-mutation-log-RUN-260826-ae2fa5.txt`.

## 9. Handoff

`accept_cr(TASK-260826-fcu5pe, revision=5, evidence=TASK-260826-fcu5pe_review-verdict.md)`.
CR-TASK-260826-fcu5pe-5 is accepted; the element parks at `to-review`. The orchestrator commits
the scope and makes the `done` transition with `commit_ack=scope_committed`. **Integration must be
a merge, not a tree reset** — the branch is 4 commits behind main and those 38 board artifacts
must survive (section 2).
