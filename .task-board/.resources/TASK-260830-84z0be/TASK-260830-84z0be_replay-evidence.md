# TASK-260830-84z0be — replay of accepted CR-TASK-260830-tvy8q5-6 onto exact trunk 4270549

Date: 2026-08-31
Role: developer
Workspace: managed Story worktree `.temp/STORY-260830-7tt3gf/worktree` on `task-board/story/STORY-260830-7tt3gf`

## 1. Base authority verified before any edit (AC1)

`task-board worktree status --json` for `WS-99ca75a4fdac` / `STORY-260830-7tt3gf`:

| Field | Value |
| --- | --- |
| `selected_base_oid` | `4270549dd17c010599e2083bf3ec7672af60ea29` |
| `local_base_oid` | `4270549dd17c010599e2083bf3ec7672af60ea29` |
| `upstream_oid` (`refs/remotes/origin/main`, `authority_source=fetched_upstream`) | `4270549dd17c010599e2083bf3ec7672af60ea29` |
| `initial_base_oid` / `current_base_oid` / `checkpoint_oid` | `4270549dd17c010599e2083bf3ec7672af60ea29` |
| `branch_tip_oid` | `4270549dd17c010599e2083bf3ec7672af60ea29` |
| worktree `git rev-parse HEAD` | `4270549dd17c010599e2083bf3ec7672af60ea29` |

All seven agree on the exact base named by the task.

**Observation the orchestrator must own, not a blocker for this replay.** The
shared repository's live `refs/heads/main` and `refs/remotes/origin/main` have
since advanced three commits to `5feebbb170ea9a9ef884899c846a897d58f02fc5`
(`bb857fe`, `bf9717b`, `5feebbb`). Those three commits touch `LOGBOOK.md`,
`README.md`, `task-board.config.json`, `.research/**`, and
`articles/260831_local-qwen-runtime-comparison-study/**`; they touch no
`tools/agents-infra` path. The task pins this replay to exact `4270549`, so the
candidate is based there. Landing it will need one further union reconciliation
of `LOGBOOK.md` and `README.md` against `5feebbb`. No code path overlaps.

## 2. Accepted patch identity reproduced (AC2)

| Property | Expected (accepted CR review surface) | Reproduced |
| --- | --- | --- |
| Patch sha256 | `1ed35314955527822a6211f11510c10582aa9f588e9558731d1321b837c117ad` | identical |
| Base OID | `b78498bf98c05175db10bb341aee621e53de4881` | present, tree `3fd838f65ea0dbd22c664780ceeb5cb9b390feeb` |
| Candidate tree OID | `57e2a9f3e43d75e806d792fe4675a2572345063a` | `git read-tree b78498b` + `git apply --cached` + `git write-tree` = `57e2a9f3e43d75e806d792fe4675a2572345063a` |
| Changed paths | 26 | 26 |

The identity check ran against a scratch `GIT_INDEX_FILE`, so it never touched
the working tree or the real index.

## 3. Reconciliation onto trunk (AC3)

Trunk moved `b78498b -> 4270549` over five paths:
`.instructions/INSTRUCTIONS_WORKFLOW.md`, `LOGBOOK.md`, `README.md`,
`task-board.config.json`, `tools/agents-infra/internal/infra/infra_test.go`.
Only `LOGBOOK.md` and `README.md` intersect the accepted 26-path scope, which is
exactly the `integration_base_moved` overlap recorded in the refusal note.

`git merge-tree --write-tree --merge-base=b78498b 4270549 57e2a9f` reported one
content conflict (`LOGBOOK.md`) and auto-merged `README.md`.

**Proof that no invariant was dropped:**

- All 25 non-`LOGBOOK.md` paths in the delivered worktree are byte-identical to
  the three-way merge result (`git diff <merge-tree> <candidate-scope-tree>`
  reports only `LOGBOOK.md`).
- `SKILL.md` is untouched on trunk and is byte-identical to the accepted
  candidate.
- `README.md`: 0 lines deleted relative to the accepted candidate; the 10 lines
  it deletes relative to trunk are the *same set* the accepted candidate itself
  deletes from `b78498b` (sorted deleted-line sets compare equal). Trunk's
  `### External-CI local mirror fallback` section is present and complete.
- `LOGBOOK.md`: 0 lines deleted relative to trunk and 0 lines deleted relative
  to the accepted candidate — a strict union of both sides.

**Two defects in a naive union were corrected:**

1. The conflict resolution left the trunk block (`1220`, `1218`, `1154`) above
   the candidate block (`1440` …), violating the file's own
   "Newest entries first" rule stated at `LOGBOOK.md:4`. Entries were reordered
   to `1440, 1440, 1345, 1320, 1253, 1242, 1220, 1218, 1154, 1138, 0705, …`
   with every entry preserved verbatim.
2. The blank separator before `### 0705` had been consumed with the `>>>>>>>`
   marker. It was restored; the only remaining missing separators in the file
   (`### 2245`, `### 1449`) are pre-existing on trunk and out of scope.

One new task-scoped entry was added under a new `## 2026-08-31` header:
`### 0807 — Stale Integration Is Replayed, Not Rebased`.

## 4. Gates (AC4)

Every command ran directly as a standalone process; real exit codes are in
`TASK-260830-84z0be_change-request_rev1-validation.log`. All of them ran against
the exact published candidate tree `913168e4ad563edd38551f8d88cdf00665149536`.

| Gate | Command | Exit |
| --- | --- | ---: |
| build | `go build ./...` | 0 |
| vet | `go vet ./...` | 0 |
| format | `gofmt -l <25 changed .go files>` (no output) | 0 |
| diff | `git diff --check` (no output) | 0 |
| production entry | `go test . -run '^TestRunPiLifecycle' -count=1` | 0 |
| **foreign mutant (expected red)** | narrowed overlay dropping only `status.ForeignCount == 0` | **1** |
| **legacy mutant (expected red)** | narrowed overlay dropping only `status.LegacyCount == 0` | **1** |
| legacy authority + crash recovery + bounded retention + eight-week soak + pressure composition | `go test ./internal/infra -run '^(TestPiLifecycle\|TestPiLegacy\|TestPiAutomatic\|TestPiRetentionPlane\|TestParsePiLifecycle\|TestRunPiLifecycle\|TestRunPiRejectsMissingLifecycle)' -count=1` (47 tests) | 0 |
| full suite (main, attachments, modelharness, cmd) | `go test . ./internal/attachments ./internal/modelharness ./cmd/model-harness -count=1` | 0 |
| full suite (internal/infra) | `go test ./internal/infra -count=1` | 0 |
| full race (main, attachments, modelharness) | `CGO_ENABLED=1 go test -race … -count=1` | 0 |
| full race (internal/infra) | `CGO_ENABLED=1 go test -race ./internal/infra -count=1` | 0 |
| cross-platform linux/amd64 | `GOOS=linux GOARCH=amd64 go test -exec=true ./... -run '^$' -count=1` | 0 |
| cross-platform linux/arm64 | `GOOS=linux GOARCH=arm64 go build ./...` | 0 |
| cross-platform windows/amd64 | `GOOS=windows GOARCH=amd64 go test -exec=true ./... -run '^$' -count=1` | 0 |
| isolated installed parity | `CGO_ENABLED=0 go build`, `setup global --home-dir <fresh /tmp HOME>`, `verify global`, `doctor global` | 0, 0, 0, 0 |

### Mutant detail — the gates are narrowed, not deleted

Both overlays replace exactly one line of `internal/infra/pi_session_log.go:2109`
and remove exactly one clause from the production `WithinPolicy` predicate; every
other clause and the whole `SoakReady` derivation stay intact. Both are driven
through the real production entry point `runPi -> runPiLifecycleCLI ->
PiLifecycleOperatorStatus -> PiLifecycleStatus`.

- Foreign mutant failure text: `ForeignCount:1 … WithinPolicy:true SoakReady:true`
  — the test fails because the gate admitted foreign evidence, not because of
  setup or compilation.
- Legacy mutant failure text: `LegacyCount:1 LegacyBytes:3` admitted as within
  policy at the pre-retirement assertion.

The `windows/amd64` compile that the revision-6 reviewer could not execute
(missing Windows-only `github.com/natefinch/atomic` archive in their offline
proxy) ran green here, closing that stated evidence gap.

## 5. Runtime boundary

No Pi executable, configured runtime, MLX/Qwen model or provider process,
service, socket, endpoint, or network service was contacted. `GOPROXY=off` and
`GOSUMDB=off` were set for every Go invocation, so no module endpoint was
reached either. Installed parity ran against a freshly created
`mktemp -d /tmp/TASK-260830-84z0be-home.XXXXXX` HOME; the user's `~/.agents`
was not written (`find ~/.agents -maxdepth 1 -newermt '-3 hours'` returned
nothing). Test configuration used temporary HOME/PATH values and nonexistent
runtime paths throughout.

## 6. Candidate identity for review (AC5)

| Property | Value |
| --- | --- |
| Base OID | `4270549dd17c010599e2083bf3ec7672af60ea29` |
| Candidate tree OID | `913168e4ad563edd38551f8d88cdf00665149536` |
| `tools/agents-infra` subtree OID | `88ef8270021ffdc0ac30ea545201462c818c098f` |
| Patch sha256 | `0c63c3bc5d9ea0496fc2c26c112f9361ee092791681950eff38cbb0023478afb` |
| Patch bytes | 336966 |
| Changed paths | 26 |
| Round trip | `git read-tree 4270549` + `git apply --cached <patch>` + `git write-tree` = `913168e4ad563edd38551f8d88cdf00665149536` |

`git status --porcelain --untracked-files=all` outside gitignored `.temp/`
contains exactly those 26 paths — no out-of-scope file was created or modified.
The candidate tree was recomputed after every gate had run and is unchanged.

Handed off to review. This developer evidence makes no review, acceptance, or
integration claim.

## 7. What "published" means for this Change Request

The board registers a Change Request record under
`.temp/changerequests/<TASK-ID>/` bound to the producing run; the record is
written by the task-board runtime when this producer run terminates, and there
is no operator-facing command that creates one (the mutation DSL exposes none,
and `task-board worktree` only inspects, checkpoints, or integrates existing
records). What this run owns and has completed:

- The candidate is immutable and self-verifying: base
  `4270549dd17c010599e2083bf3ec7672af60ea29`, tree
  `913168e4ad563edd38551f8d88cdf00665149536`, patch sha256
  `0c63c3bc5d9ea0496fc2c26c112f9361ee092791681950eff38cbb0023478afb`, and the
  patch round-trips back to that exact tree from that exact base.
- The diff is attached under the runtime's own naming convention as
  `TASK-260830-84z0be_change-request_rev1.patch`, alongside
  `TASK-260830-84z0be_change-request_rev1-validation.log` and this document.
- The candidate tree was recomputed after the last gate finished and is
  unchanged, so the record the runtime registers at run termination will bind
  the same tree these artifacts describe.

A reviewer can therefore verify the candidate end to end from the attached
artifacts alone, without trusting this run's summary.
