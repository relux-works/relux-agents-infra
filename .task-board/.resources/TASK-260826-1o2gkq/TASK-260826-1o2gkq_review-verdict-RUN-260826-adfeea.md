# TASK-260826-1o2gkq — Reviewer verdict, CR rev4 (story_final)

- Run: `RUN-260826-adfeea` (claude-opus-5/high, reviewer archetype)
- Change Request: `CR-TASK-260826-1o2gkq-4` revision 4
- Base OID: `e70f953969d46e451892d9f16e7401b879910b6b` (= current local `main`)
- Candidate tree: `281a72e1b96ca8c08ca62ea54f6f2d2557c1e33d`
- Verdict: **ACCEPTED**
- Nothing in the reviewed worktree was modified. Every mutant ran in a
  `git archive`-materialized scratch extraction under
  `.temp/STORY-260825-1r7z9o/review-RUN-260826-adfeea/cand`, which re-hashed to
  `281a72e…` after each restore (asserted 4 times).

## 1. CR integrity — reconstructed, not trusted

| Check | Result |
| --- | --- |
| Patch sha256 | `b1fe19f8…ba0481` — matches the declared value |
| Declared paths | 25 `diff --git` headers — matches |
| Reconstruction | `read-tree e70f953` → `git apply --cached rev4.patch` → `git write-tree` = **`281a72e1b96ca8c08ca62ea54f6f2d2557c1e33d`**, exact |
| Base identity | `e70f953` == `git rev-parse main`, and `git merge-base main HEAD` |
| Candidate == worktree | `git diff --quiet 281a72e --` exit 0; zero untracked non-ignored files |
| Scratch materialization | `git archive 281a72e` re-hashed to `281a72e` |

Applying the published patch to the published base yields the published tree
byte for byte. The CR is self-consistent.

## 2. Fresh-main ancestry and `integration_base_moved`

- `git merge-base --is-ancestor main HEAD` → exit 0.
- `git rev-list --count HEAD..main` → 0.
- `git merge-tree --write-tree main HEAD` → `5a15d04…` = `HEAD^{tree}` exactly,
  exit 0. Clean fast-forward, no conflict, refusal closed.
- Merge `c40f0c2` has exactly two parents: accepted Story tip `91adc73` and
  current main `e70f953`.

## 3. Zero product drift vs the accepted rev5 tree

`git diff --name-status 40a83fe(=91adc73^{tree}) 281a72e -- ':(exclude).task-board'`
returns exactly five paths, and nothing else:

```
M LOGBOOK.md
M tools/agents-infra/internal/infra/pi_shared_attestation_test.go
M tools/agents-infra/internal/infra/pi_shared_broker_admission_test.go
M tools/agents-infra/internal/infra/pi_shared_integration_test.go
M tools/agents-infra/internal/infra/pi_shared_launcher_test.go
```

The four test blobs in the candidate tree are **byte-identical to the accepted
witness-leaf post-images** in `TASK-260826-12laby_change-request_rev2.patch`:

| Path | rev2 post-image | candidate blob |
| --- | --- | --- |
| `pi_shared_attestation_test.go` | `3e9bbee` | `3e9bbee` |
| `pi_shared_broker_admission_test.go` | `7aa33c1` | `7aa33c1` |
| `pi_shared_integration_test.go` | `c7a037a` | `c7a037a` |
| `pi_shared_launcher_test.go` | `b4f3f75` | `b4f3f75` |
| `LOGBOOK.md` | `fcba0b2` | `e745abf` (the one intended correction) |

`git diff HEAD^{tree} 281a72e` is a single hunk: 1 insertion, 1 deletion, in
`LOGBOOK.md`. No production file differs from the accepted rev5 tree at all.

## 4. Trunk preservation — attacked separately

Tree-identity with rev5 would also pass if the merge had silently reverted a
trunk-only change, so this was checked independently. Fork point is
`b3cb8455`. Of the 18 non-board paths main changed since then:

- 11 are **byte-identical to `main`**,
- 7 differ from both merge-base and main; **none equals the merge-base blob**,
  so nothing was reverted,
- of those 7, 5 have **zero deletions** against main (`LOGBOOK.md` +178/-0,
  `pi_launch_posix.go` +3/-0, `pi_plan.go` +22/-0, `pi_test.go` +88/-0,
  `main.go` +103/-0) — main's lines survive as an in-order subsequence,
- the 2 with deletions are single-line strict extensions, verified by character
  diff: the README `agents-infra` row loses **nothing** and gains
  `runtime status` / `runtime stop`; `rejectUnknownFields` keeps every existing
  field name and gains `"sharing"`.

`pi_state.go` (-24) is not in main's changed set — main never touched it since
the fork point; those deletions are rev5 refactors of merge-base code.

**LOGBOOK resolution is additive**: +178/-0 against main, so main's LOGBOOK is a
line subsequence of the candidate's, and byte-identical to the accepted Story
tip apart from the single corrected sentence.

## 5. The LOGBOOK correction is factually accurate

Old: *"Four production `Dev != … || Ino != …` gates had wrong-device witnesses
but no same-device/wrong-inode witness…"*

New: *"Three production `Dev != … || Ino != …` gates had wrong-device witnesses
but no same-device/wrong-inode witness; `sharedRuntimeBrokerCandidates` had
neither witness, three inode-removal mutants survived, and the force-stop gate
was killed only incidentally by a foreign-binary test."*

There are exactly four such gates in production:

| # | Site | Function |
| --- | --- | --- |
| 1 | `pi_shared_broker_darwin.go:836` | `(*sharedBrokerServer).attestClient` |
| 2 | `pi_shared_client_darwin.go:341` | `connectAndAttestSharedRuntime` |
| 3 | `pi_shared_operator_darwin.go:363` | `sharedRuntimeBrokerCandidates` |
| 4 | `pi_shared_operator_darwin.go:470` | `stopRecordedSharedRuntimeWithDependencies` |

Each clause of the corrected sentence checks out:

- *"`sharedRuntimeBrokerCandidates` had neither witness"* — reproduced directly
  (**mutant B**, §6). The device half is still unwitnessed in the delivered
  tree, and the witness leaf only added tests, so it cannot have been witnessed
  before either.
- *"three inode-removal mutants survived"* = sites 1, 2, 3; independently
  recorded by `RUN-260826-1b682b` (survived at `:836`, `:341`, `:363`).
- *"the force-stop gate was killed only incidentally by a foreign-binary test"*
  = site 4, via `TestSharedRuntimeForceStopRefusesForgedBrokerIdentityWithoutSignal`,
  same source.
- *"Three … had wrong-device witnesses"* = sites 1, 2, 4 — the complement of the
  named exception, consistent with the five Dev-perturbing witnesses recorded in
  the same prior verdict.

The sentence is accurate. The AC clause "the LOGBOOK sentence accurately states
prior gate coverage" is met.

## 6. Gates attacked, not read

Seven narrowing mutants against the exact candidate tree. Every one narrows the
gate rather than deleting it, so a survivor is a real coverage hole.

| # | Mutation | Scope run | Result |
| --- | --- | --- | --- |
| A | `pi_shared_operator_darwin.go:363` — drop the **inode** clause | named witness | **KILLED**, exit 1, `TestSharedRuntimeBrokerCandidatesRejectSameDeviceWrongInodeAtProductionEntry` |
| B | `pi_shared_operator_darwin.go:363` — drop the **device** clause | focused, then **full `go test ./...`** | **SURVIVED**, exit 0 (focused 26.4s; full suite: root 60.0s, attachments 2.2s, infra 108.7s) — see F1 |
| C | `pi_config.go:359` — `>=` → `>` (admit heartbeat == lease_stale) | named witness | **KILLED**, exit 1, only `TestParsePiRuntimeSharingIsStrictAndOptIn/heartbeat_equals_stale`; other 11 subtests green |
| E | `pi_shared_client_darwin.go:379` — `!=` → `>` (admit below-current protocol) | focused | **KILLED**, exit 1, only `…/past_protocol_version_range_narrowing` |
| F | `pi_shared_broker_darwin.go:839` — `!=` → `>` (admit below-current protocol) | focused: **exit 0**; named witness: **exit 1** | **KILLED** by `TestSharedBrokerAttestClientRejectsEveryGateDeleteAndNarrowWitness/past_protocol_version_range_narrowing`, all 9 sibling subtests green — see F2 |
| G | `pi_shared_broker_darwin.go:836` — drop the **inode** clause | named witness | **KILLED**, exit 1, only `…/client_executable_same_device_wrong_inode` |

Mutants E and F together close the "weaker lead" left open by
`RUN-260826-1b682b` (`!=` → `>` admitting every version below 6, including the
zero value of an omitted field). Both the client and the broker side now refuse
the below-current direction under a named witness. Mutants A and G confirm the
witness leaf's inode closure is real at two independent production entries.

## 7. Reviewer-run validation — nothing accepted from producer logs

All commands run by this reviewer against the materialized candidate tree.

| Gate | Runs | Result |
| --- | ---: | --- |
| `gofmt -l .` | 1 | 0, empty output |
| `go build ./...` | 1 | 0 |
| `go vet ./...` (configured) | 2 | 0, empty output |
| Focused shared runtime | 4 | 0 / 0 / 0 / 0 — `33.7s`, `32.3s`, `33.2s`, `32.7s` |
| Focused shared runtime `-race` | 1 | 0 — `56.8s` |
| **Configured landing** `go test ./... -count=1` | 2 | 0 / 0 — root `64.7s`/`63.7s`, attachments `1.2s`/`1.3s`, infra `110.9s`/`109.4s` |
| Landing split by package (bounded calls) | 1 | 0 — root `68.0s`, attachments `1.6s`, infra `125.0s` |

**The prior chain's carried honest-red is closed.** `RUN-260826-8e0561` measured
the focused gate failing 2-of-3 and `RUN-260826-1b682b` measured the infra
package failing 1-of-2, both on the 15-second wall-clock positive control. The
cause is structurally gone, not merely lucky: `sharedLauncherPositiveControlTimeout
= 15 * time.Second` exists at `91adc73:pi_shared_launcher_test.go:23` and does
**not** exist in the candidate — the witness leaf replaced marker polling with a
target-emitted stdout event. Nine green measurements here, including two runs of
the literal configured command under package-parallel load, and zero reds.

## 8. Findings

### F1 — carried, non-blocking, Story-scoped: the candidates **device** clause is unwitnessed repo-wide

Dropping `identity.Dev != ownIdentity.Dev` at
`pi_shared_operator_darwin.go:363` leaves the **entire configured landing suite
green** (`go test ./... -count=1`, exit 0; root 60.0s, attachments 2.2s, infra 108.7s). A same-inode/different-device
executable would be admitted as a broker candidate by
`sharedRuntimeBrokerCandidates`, called in production at `:214` and `:438`.

This is out of this element's scope — its AC pins the rev5 test blobs
byte-identical, so routing to `to-dev` would order a change that fails its own
acceptance criteria. It reproduces the carried finding from
`RUN-260826-589f55` (mutant I) at full-suite rather than focused scope.

What is *not* recorded anywhere: the LOGBOOK correctly says this gate "had
neither witness" in the past tense, and the FIX line says witnesses "now cover …
broker-candidate scanning" — true for the inode direction, silent on the fact
that the device direction is **still** open after the fix. Recommend the Story
owner either open one leaf authorized to touch rev5 test blobs adding a
`same_inode_wrong_device` witness at `:363`, or record the gap consciously
before Story integration.

### F2 — new: the "focused shared-runtime" mask omits the broker-admission and config-strictness witness tables

The mask used as the focused gate throughout this Story's evidence chain,
`^(TestSharedRuntime|TestSharedAuthorization|TestConnectAndAttestSharedRuntime|TestRunSharedRuntimeBroker|TestReclaimSharedRuntime)`,
does not match:

- `TestSharedBrokerAttestClientRejectsEveryGateDeleteAndNarrowWitness` (the
  10-case broker admission witness table)
- `TestSharedBrokerAcquireLeaseRefusesDrainingBeforeGrant`
- `TestSharedBrokerProductionConnectionRejectsWireFrameBoundWidening`
- `TestParsePiRuntimeSharingIsStrictAndOptIn` (the 12-case sharing config
  strictness table)

Demonstrated, not inferred: **mutant F** — narrowing the broker protocol gate to
admit every below-current version — exits **0** under the focused mask (27.8s)
and exits **1** on its named witness. No gate is actually unwitnessed; the
configured landing suite covers all of them and is green. But "focused
shared-runtime: exit 0" is a materially weaker signal than this chain has been
treating it as, and any future rework that reports only the focused gate can
ship a broker-admission or config-strictness regression. Recommend widening the
mask to `^(TestShared|TestConnectAndAttestSharedRuntime|TestReclaimSharedRuntime|TestParsePiRuntimeSharing)`
or equivalent when it is next authored.

### F3 — orchestrator action required before integration: the accepted tree is not committed

The accepted candidate tree `281a72e…` exists **only as uncommitted working-tree
state**. Branch tip `52270bf` carries tree `5a15d04…`, which still contains the
**uncorrected** "Four production gates" sentence — the exact defect this element
exists to fix. `git merge-tree --write-tree main HEAD` returns `5a15d04…`,
confirming that a fast-forward of the branch as it stands today would land the
uncorrected wording.

The one-line LOGBOOK correction must be committed as part of the checkpoint
before the `done` transition with `commit_ack=scope_committed`. This reviewer
does not commit and does not supply `commit_ack`.

## 9. Bounds on this review

- Full-package `-race` was not run; `-race` was run against the focused mask
  only (exit 0, 56.8s). The configured landing suite does not specify `-race`.
- Mutants A, C, E, G were scoped to their named witness or the focused mask
  after the mutation's kill site was located; only mutant B, the one survivor
  I report as a finding, was escalated to the full configured landing suite.
- The CR `kind` is **inferred** as `story_final` from the fork-point base, the
  25-path Story-wide scope, and the orchestrator note at `progress.md:86`. It
  was not read: no child-side CR read verb exists on the board CLI.
- No board files and no reviewed-worktree bytes were modified by this run.
