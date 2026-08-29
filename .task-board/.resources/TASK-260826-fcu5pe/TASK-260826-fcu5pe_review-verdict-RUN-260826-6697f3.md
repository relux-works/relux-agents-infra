# TASK-260826-fcu5pe review verdict — CR-TASK-260826-fcu5pe-4 rev4

Reviewer run: `RUN-260826-6697f3` (claude-opus-5)
Verdict: **changes requested** → `to-dev`

Candidate: commit `727a45b019ba2ee98a75577565e6e8f5a1f212a1`, tree `e2951b27d3e2863cde557887a9c0614f3d1ea0c0`
Base: `b3cb84550a60f7f4df92a287c573bfc692cd26e0` · current main: `e70f953969d46e451892d9f16e7401b879910b6b`
Candidate worktree clean and unmodified at start and end of this review. Every mutant ran in
`/tmp/rev4-mut/` (module-only) and `/tmp/rev4-full/` (whole repo), never in the candidate tree.

**The reconciliation this task was created to do is correct and is not what is being refused.**
Section 2 certifies it in full so the rework does not redo it. The refusal is section 4: the one
production path in this change that sends `SIGTERM`/`SIGKILL` has its identity chain proven by a
single delete-shaped witness, and every narrowing of that chain survives the configured landing
gate. Six gate comparisons can be neutralised in one build with `go test ./... -count=1` still
exiting 0.

---

## 1. Published candidate is the tree I reviewed

- `git rev-parse HEAD^{tree}` = `e2951b27d3e2863cde557887a9c0614f3d1ea0c0` = the CR's declared candidate tree.
- `git diff <base> <candidate>` reproduces to sha256
  `3387224bd8bfe5dc7f8f2aa209a44b7a8b1cbc727e6e0504a862151f89cc1505`, byte-identical to the
  published `TASK-260826-fcu5pe_change-request_rev4.patch` and to the CR's declared hash.
- rev3 → rev4 is **one blob**: `git ls-tree -r 7ef4258` vs `git ls-tree -r HEAD` (non-board paths)
  differ only in `LOGBOOK.md` (`f5c8ae0` → `4b0ad16`), a single restored blank line at
  `LOGBOOK.md:222`. Every source blob accepted at revision 3 is byte-identical here.

## 2. Reconciliation onto current main — certified, re-derived by this reviewer

Re-derived independently, not read from the rework note.

- `git merge-base HEAD main` = `b3cb845`. Branch is **4 behind / 24 ahead**. Those 4 are main's two
  "Record board state" commits plus its two story merges (`STORY-260825-19dzsf`,
  `STORY-260825-7oqacp`), whose product content is carried on the branch as `3d203b1` / `5e3bbe0`,
  applied on top of the broker work at `8c47c13`.
- **Overlap set is exactly 7 files** (`comm -12` of `b3cb845..8c47c13` and `b3cb845..main`
  non-board paths): `LOGBOOK.md`, `README.md`, `pi_config.go`, `pi_launch_posix.go`, `pi_plan.go`,
  `pi_test.go`, `main.go`. I read `git diff main HEAD` for each. **Every hunk is additive broker
  work; no main content is displaced.** `pi_config.go`'s apparent deletions are the
  `rejectUnknownFields` allow-list gaining `"sharing"` and `clonePiProfiles` gaining deep copies;
  README's is main's tools-table row extended in place, with main's `model-check` content intact.
- **Main's story content was composed, not clobbered.** `SKILL.md`, `model_check.go`,
  `model_check_test.go`, `model_check_main_test.go`, `model_check_docs_test.go`,
  `canonical_target.go`, `canonical_target_pi_test.go`, `canonical_target_pi_main_test.go`,
  `pi_run_report.go`, `pi_platform_windows.go`, `pi_operator_docs_test.go` do **not appear in
  `git diff --stat main HEAD` at all** — they are byte-identical to main.
- `git diff --name-status main HEAD | grep '^D'` → **38 deletions, all under `.task-board/`, zero
  source deletions.** Those artifacts are absent because the branch forked before they existed.
  **Integration must be a merge, not a tree reset.**
- No task-board Pi adapter path and no standalone yolo policy path appears anywhere in
  `git diff --name-only main HEAD`.

This part of the acceptance criteria is met. Do not re-do it.

## 3. Validation — every command rerun by this reviewer, foreground, exit codes captured

| Command | Where | Exit | Time |
| --- | --- | ---: | ---: |
| `gofmt -l .` (empty output) | candidate | 0 | — |
| `git diff --check` | candidate | 0 | — |
| `go vet ./...` (configured landing gate) | candidate | 0 | — |
| `go build ./...` (darwin) | candidate | 0 | — |
| `GOOS=linux GOARCH=amd64 go build ./...` | candidate | 0 | — |
| `GOOS=windows GOARCH=amd64 go build ./...` | candidate | 0 | — |
| `go test ./internal/infra -count=1` | candidate | ok | 156.6s |
| `go test ./... -count=1` (configured landing gate) | candidate | 0 | 127.3s / 2.4s / 187.1s |
| `go test -race ./internal/infra -run '^(TestShared\|TestConnectAndAttestSharedRuntime\|TestRunSharedRuntimeBroker\|TestReclaimSharedRuntime)'` | candidate | 0 | 46.8s |
| `go test ./... -count=1` on a pristine whole-repo scratch copy | `/tmp/rev4-full` | 0 | 108.4s / 1.8s / 259.5s |
| `go test ./internal/infra -run '^TestSharedRuntimeLauncher'` ×6 | `/tmp/rev4-full` pristine | 0 ×6 | — |

Green. The suite passing is not in dispute; what it would catch is.

## 4. REFUSAL — the force-stop identity chain is bound by deletion only

**Production call site.** `main.go runRuntime "stop"` → `infra.StopSharedRuntime`
(`pi_shared_operator_darwin.go:124`) → `forceStopUnreachableSharedRuntime:406` → **`:414
stopRecordedSharedRuntime`** → the chain at `:442`/`:454` → **`syscall.Kill(record.Broker.PID,
SIGTERM)` at `:460` and `SIGKILL` at `:462`.** This is the only production site in the change that
signals a PID it did not fork. Its entire job is to guarantee a stale `broker.json` cannot make the
operator kill an innocent process.

**Its sole witness distinguishes on inode and nothing else.**
`TestSharedRuntimeForceStopRefusesForgedBrokerIdentityWithoutSignal`
(`pi_shared_integration_test.go:913`) starts `/bin/sleep 10`, then writes a broker record carrying
that process's **true** `PID`, `PGID`, `SID`, `StartTime`, `UID`, `ExecPath` and `Argv`. UID matches,
StartTime matches, Argv matches, `live()` is true. The only field that can refuse it is the
executable inode. So the witness proves the executable comparison is *present*; it says nothing
about the class any other comparison covers, and nothing about the executable comparison's own
shape.

**Every narrowing survives the whole package** (`go test ./internal/infra -count=1 -run .`, scratch
tree, compile-valid, revert-verified):

| # | Gate (production site) | Mutant | Package result |
| --- | --- | --- | --- |
| O1f | recorded broker argv `:442` | drop `!equalStrings(brokerObservation.Argv, record.Broker.Argv)` | **SURVIVED** 145.6s |
| O2f | recorded broker start time `:442` | drop `brokerObservation.StartTime != record.Broker.StartTime` | **SURVIVED** 136.6s |
| O3f | recorded broker executable `:454` | narrow to `Ino` only, drop `Dev` | **SURVIVED** 211.1s |
| O5f | `sharedRecordedBrokerIsLive:403` | drop `observation.StartTime == record.StartTime` | **SURVIVED** 216.3s |

O2f is the PID-reuse guard. **Failure scenario:** `broker.json` names PID *N*; the broker dies
uncleanly; the kernel recycles *N* to another `agents-infra` process of the same user (another
profile's broker, or a plain CLI invocation). With O2f applied, uid matches, `live()` is true, the
executable identity matches because it is the same binary — `agents-infra runtime stop --force`
sends it SIGTERM and then SIGKILL. `go test ./...` stays green throughout.

O3f is the same shape the client and broker chains both close explicitly with
`broker_executable_same_inode_wrong_device` and `client_executable_same_inode_wrong_device`
witnesses. The operator chain has no equivalent.

**This also refutes rev3 §6.2.** That section dismissed a `stopRecordedSharedRuntime` uid narrowing
as safe because "any record naming a root process still fails the executable-identity comparison two
lines down, **which is bound** — deleting that comparison is killed by
`…ForgedBrokerIdentityWithoutSignal`." Deleting it is killed. **Narrowing it is not** (O3f). The
successor relied on to make the uid narrowing safe is itself only delete-bound, so the safety
argument does not close. Recorded so the review record does not keep saying this surface was
attacked and cleared.

## 5. Three equality gates leave the empty-value class open where the suite closes it everywhere else

| Gate | Production site | Its witness uses | Narrowing | Result |
| --- | --- | --- | --- | --- |
| client `profile_digest` | `pi_shared_client_darwin.go:387` | `strings.Repeat("0", 64)` | `&& response.ProfileDigest != ""` | **SURVIVED** whole package, 139.4s |
| launcher `schema` | `pi_shared_launcher_darwin.go:71` | `"future"` | `\|\| frame.Schema == ""` | **SURVIVED**, 2/2 clean runs |
| launcher `runtime_key` | `pi_shared_launcher_darwin.go:74` | `strings.Repeat("f", 64)` | `\|\| frame.RuntimeKey == ""` | **SURVIVED**, 1 clean green + 1 unrelated flake (§7) |

This is not a general gap — it is three exceptions to a rule the suite otherwise keeps. The client's
immediate neighbours both close it (`empty runtime key`, `empty endpoint`), and the launcher closes
it for two of its five comparisons (`launcher pid zero`, `exec plan digest empty`). An absent or
zero-valued field is the cheapest thing a broken or hostile peer emits; it is the one value these
three gates are not proven against.

For contrast, the other twelve client gates and all four remaining launcher gates I attacked die on
their named narrowing witness — full table in the attached mutation log. The chain is in good shape;
these are the holes in it.

## 6. Two smaller unbound gates

**6.1 `hello_ok` completeness has no witness and its removal is a panic, not a refusal.**
`pi_shared_client_darwin.go:370` refuses an incomplete hello response. Dropping
`|| response.EffectiveSharing == nil` **survives the whole package (205.7s)**, and with it gone the
client dereferences `*response.EffectiveSharing` at `:429`. A validation whose removal converts a
`protocol_violation` refusal into a nil-pointer dereference should not be the one clause in that
condition with no negative test.

**6.2 The launcher's recomputed-runtime-key gate has no witness.**
`pi_shared_launcher_darwin.go:45`. A first-8-characters prefix comparison **survives the whole
package (143.3s)**. The broker's identical gate is bound by
`TestRunSharedRuntimeBrokerRefusesRecomputedRuntimeKeyAtProductionEntry`; the launcher's is not.
Its successor `exec_plan_digest` is bound and would catch the divergence, so this is defence in
depth — but the asymmetry is unexplained and the gate is unbound.

## 7. Composite — the landing gate stays green with six comparisons neutralised

Whole-repo scratch tree `/tmp/rev4-full` (pristine baseline green, §3). Applied in one build:
operator `:442` reduced to `uid` + `live()`, operator `:454` reduced to `Ino`,
`sharedRecordedBrokerIsLive:403` stripped of start time, client `profile_digest:387` exempting the
empty value, client `hello_ok:370` stripped of the `EffectiveSharing` clause, launcher
`:45` prefix-comparing 8 characters.

```
go build ./...            -> BUILD_OK
go test ./... -count=1    -> COMPOSITE_EXIT=0
  ok  .../tools/agents-infra                 124.276s
  ok  .../tools/agents-infra/internal/attachments 3.652s
  ok  .../tools/agents-infra/internal/infra  201.938s
```

This is the same shape rev2 recorded as a failure for the broker's admission chain and rev3 then
turned red with nine named subtests. Same class, different surface, still open.

## 8. Observation — the 15 s launcher positive control flakes on a clean tree

Recorded, not blocking, but it corrects rev3 §6.1.

`…RejectsAuthorizationChannelGuardBypassesAtProductionEntry/descriptor_three_is_a_socket_rather_than_a_fifo`
failed at exactly **15.02 s** (`sharedLauncherPositiveControlTimeout`,
`pi_shared_launcher_test.go:23`) in a run whose only mutation was the launcher `runtime_key`
empty-value exemption. **That mutation is provably unreachable in that subtest**: the case is
refused at `unix.Fstat(3)` (`pi_shared_launcher_darwin.go:34`) before any frame is read, so no
frame comparison executes. The failure is therefore a pure wall-clock flake, not a kill. Six
pristine repetitions afterwards were green, so the rate is low.

rev3 §6.1 recorded this flake as firing "only with the broker's admission gates removed, a state
that never ships". It fired here on a clean tree with every admission gate intact. Converting the
wait to observation-driven is still the right fix; the justification for deferring it is weaker
than recorded.

It also caught a weakness in **my own** harness: two mutants (launcher `schema`, launcher
`runtime_key`) were initially scored KILLED because the driver accepted any named failure. Both
"kills" were the `valid` positive control, which those mutations cannot affect. Re-run with subtest
inspection, both are survivors — they are in §5 because of that recheck, not despite it.

## 9. What the rework needs — test-only, no production change

Every gate in this change is **correct as shipped**. Nothing in §4–§6 is a live bypass. What is
missing is the evidence that any of them is load-bearing. Required:

1. A narrowing-witness table for `stopRecordedSharedRuntime`, in the shape the client and broker
   chains already use. Minimum cases, each asserting `broker_stop_identity_mismatch` **and** that
   the recorded PID was not signalled: recorded `StartTime` differs from the live process (PID
   reuse); recorded `Argv` differs; broker executable same inode, different device; recorded UID is
   root. Name `pi_shared_operator_darwin.go:460`/`:462` as the protected site.
2. A witness for `sharedRecordedBrokerIsLive` start time, asserting the reported broker state, since
   it feeds status classification at `:226`.
3. An empty-value witness for client `profile_digest`, launcher `schema`, launcher `runtime_key`,
   matching the `empty runtime key` / `empty endpoint` / `exec plan digest empty` cases already
   present.
4. A witness for `hello_ok` with `effective_sharing` absent → `protocol_violation`, asserting the
   refusal rather than a panic.
5. A production-entry witness for the launcher's recomputed runtime key, mirroring
   `TestRunSharedRuntimeBrokerRefusesRecomputedRuntimeKeyAtProductionEntry`.
6. Re-run the §7 composite and attach it **red**, with the named subtest that kills each of the six.
   An exit code alone is not a kill; require the named witness, and treat a `valid`/positive-control
   failure as `RED_WRONG_TEST` (§8).

Do not touch the §2 reconciliation, the accepted broker behaviour, the task-board Pi adapter, or
standalone yolo policy.

## 10. Reviewer reproduction

Mutant catalogs, driver and per-mutant `go test -json` results:
`/tmp/rev4-mut/` (`cat-client.json`, `cat-b.json`, `cat-c.json`, `cat-d.json`, `drive.py`,
`res-*.json`) and `/tmp/rev4-full/` (whole-repo baseline and composite). Consolidated table attached
as `TASK-260826-fcu5pe_review-mutation-log-RUN-260826-6697f3.txt`.

## 11. Handoff

`changes_requested` → `to-dev`. CR-TASK-260826-fcu5pe-4 is **not** accepted; no `accept_cr` recorded
and no `commit_ack` supplied. When the rework republishes, the next reviewer can take §1–§3 of this
verdict as re-derived and should spend its attack budget on §9 items 1 and 6.
