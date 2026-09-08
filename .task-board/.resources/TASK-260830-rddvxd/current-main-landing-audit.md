# TASK-260830-27u51n current-main landing audit

Date: 2026-08-30, Europe/Moscow

## Verdict

**Do not integrate or publish CR revision 3. Its independent review requested changes. Close the Claude include-chain bypass, then use a fresh board-managed replacement lane from exact current `origin/main`, replay the four policy paths by three-way semantic composition, obtain a fresh exact-tree review, and deliver through a signed feature-branch PR.**

The old candidate remains useful implementation and negative-review evidence, but neither its verdict nor its tree identity can authorize the composed current-main result. This is a small replay plus one focused test correction, not a redesign.

No versioned/source/user file, board record, repository Git ref/index/worktree, GitHub state, installed runtime, or live service was mutated. Only task-scoped `.temp` diagnostics, an alternate disposable index, and this report were written. No dirty-file contents or environment values were persisted or printed.

## Exact observed state

- CR: `CR-TASK-260830-27u51n-3`, revision `3`, state `ready`, but reviewer `RUN-260830-137cad` completed with **changes requested**.
- Current Task status: `development`; rework run `RUN-260830-0fd3d0` is active.
- Review finding: removing `INSTRUCTIONS_WORKFLOW.md` from `.instructions/INSTRUCTIONS.md` leaves the focused parity test green. Rework must prove the full Claude `CLAUDE.md -> INSTRUCTIONS.md -> INSTRUCTIONS_WORKFLOW.md` production chain and keep the existing Codex master-control mutant.
- Kind: `task_delta`.
- Historical base: `5c9b4e4f7a88e1eb937b80851af522e4fa4b066f`.
- Candidate tree: `b01edfee93a4599f7a3386ce32354048fc84018f`.
- Patch SHA-256: `b4b8f2e1b50339b9139e29c27ff341a25740748170367e3dbc03bb26a9f3d75b`.
- Local `HEAD` and local `main`: `5c9b4e4...`.
- Local tracking ref `origin/main`: `3295c7da7151de128f176cf7560a57d54c8f6c0d`.
- Local main is exactly 8 commits behind and 0 commits ahead; `5c9b4e4...` is an ancestor of `3295c7d...`.
- The root checkout is dirty, including both candidate-overlap paths described below. It must not be fast-forwarded, rebased, staged, stashed wholesale, or used as the landing workspace.

## Exact CR paths

1. `.instructions/INSTRUCTIONS_WORKFLOW.md`
2. `LOGBOOK.md`
3. `README.md`
4. `tools/agents-infra/internal/infra/infra_test.go`

## Semantic overlap with `5c9b4e4..3295c7d`

The trunk range contains eight commits: automatic signed delivery policy, removal of residual no-commit language, and two Codex Sol context-configuration changes, each followed by its board-state close commit.

Three of the four CR paths changed on trunk:

| Path | Trunk change | Required composition |
| --- | --- | --- |
| `.instructions/INSTRUCTIONS_WORKFLOW.md` | The complete Version Control section was replaced with automatic signed delivery, current-main freshness, exact-head landing, signature verification, and re-review after signed rebase. | Keep the new trunk contract intact. Insert External-CI local mirror fallback as a subordinate section after canonical delivery. It may supplement evidence only; it must not weaken signed exact-head delivery, current-main freshness, review, checks, merge queue, or protection. |
| `README.md` | Codex defaults changed to `gpt-5.6-sol`, 272K context, and 245K auto-compaction. | Preserve the new model/config documentation and add the independent External-CI policy section. Hunks are semantically independent. |
| `tools/agents-infra/internal/infra/infra_test.go` | Setup fixtures and assertions changed to the new Codex model/context configuration. | Preserve those fixture changes and add the external-CI clause/parity test. The test additions are in separate logical regions. |
| `LOGBOOK.md` | No committed change in this trunk range. | Add the policy/LLVM observations only after preserving the existing newest-first history and any separately owned dirty entries. |

Evidence:

- A direct index-only `git apply --check` of revision 3 against `origin/main` exits `1` at `.instructions/INSTRUCTIONS_WORKFLOW.md`; therefore raw patch application is not canonical.
- A read-only three-way merge using `origin/main` as current, `5c9b4e4` as base, and the revision-3 Story candidate as other exits `0` for all four paths. The replay is mechanically small, but its resulting tree is necessarily different from `b01edfee...` and must receive fresh review.

## Pre-existing dirty overlap

The root checkout overlaps the CR in exactly two paths:

| Path | Safe metadata | Risk |
| --- | --- | --- |
| `.instructions/INSTRUCTIONS_WORKFLOW.md` | Tracked dirty delta: 11 additions, 2 deletions; its hunk is in the Version Control area. The working, old-HEAD, and current-origin blob identities are all distinct. | It overlaps both the CR policy location and the eight-commit trunk rewrite. Any root fast-forward/apply would risk overwriting or absorbing unrelated user instruction work. |
| `LOGBOOK.md` | Tracked dirty delta: 12 additions, 0 deletions in two separate hunks. The working blob differs from old HEAD; committed origin equals old HEAD. | The CR also prepends logbook entries. Preserve the root entries independently; never use a whole-file candidate copy or whole-checkout stash as the landing mechanism. |

The root also contains dirty board state that overlaps committed board-state changes in the eight-commit trunk range. That reinforces the clean-worktree requirement; none of it belongs in this four-path policy PR.

## Smallest canonical landing path

1. Complete `RUN-260830-0fd3d0` rework and independent review. Revision 3 must remain rejected evidence; do not checkpoint or accept it. Any new old-base CR is still only a source for the subsequent current-main replay.
2. Do **not** checkpoint/integrate the stale Story as the delivery unit. The CR is only `task_delta`, while sibling `TASK-260830-12w5gq` remains backlog awaiting task-board `fast_mode` schema support; the current Story cannot land this policy independently.
3. Create a minimal replacement Story/Task whose only deliverable is the corrected four-path policy replay. Provision its managed worktree from exact freshly fetched protected `origin/main`; record selected/local/upstream OID equality at the then-current commit.
4. Three-way replay the corrected policy semantics. Preserve current trunk wholesale and resolve only the four paths listed above. Do not read from or copy the dirty root versions.
5. Run the focused source-to-installed-surface test, the Claude include-chain expected-red mutant, the existing Codex master-control and broadened-policy mutants, full uncached Go tests, `go vet ./...`, `go build ./...`, formatting/diff integrity, and setup-parity checks in the fresh workspace. Do not claim the historical logs cover the new composed tree.
6. Publish a new immutable `story_final` CR from the fresh Story, obtain independent review of that exact candidate tree, and integrate through the board-managed path.
7. Create a signed commit on a non-default feature branch from current main, verify the signature, push the feature branch, open/update the PR, inspect the complete remote diff, wait for real required checks and review, and land only the exact accepted head under the repository's automatic signed-delivery policy. If main advances, signed-rebase the feature branch, rerun review/checks, and never force main.
8. After remote merge, refresh installed parity from the merged source, not the stale worktree or an older installed CLI. Until the LLVM bug lands, the documented explicit temporary lane is `AGENTS_INFRA_SKIP_LLDB_MCP=1 ./setup.sh`, followed by `agents-infra verify global` and byte/content parity checks for Agents, Claude, and rendered Codex instructions. This audit did not run setup or touch the installed runtime.
9. Only after the replacement is merged should the historical policy Task/Story be administratively superseded through normal board mutations; preserve all old CR/review evidence.

This replacement lane is the smallest canonical route because it avoids three unsafe expansions: rebasing a dirty historical Story, coupling policy delivery to the unrelated future `fast_mode` task, and touching the dirty root checkout.

## LLVM 23 bug is separate

`BUG-260830-j1o8av` — `repair-lldb-mcp-bootstrap-after-homebrew-llvm-23-split` — is a distinct backlog bug under `STORY-260508-1ajnbe`.

- Its problem: default `./setup.sh` expects `/opt/homebrew/opt/llvm/bin/lldb-mcp`, but Homebrew LLVM 23 split packaging no longer provides that helper at the old location.
- Its scope: bootstrap/package detection, supported helper discovery or typed actionable refusal, wrapper verification, docs, and static/fake tests; it preserves `AGENTS_INFRA_SKIP_LLDB_MCP=1` as an explicit bypass.
- It does not change when an External-CI mirror is permitted and must not broaden that policy.
- The policy task's successful skip-LLDB setup proves instruction installation parity only. It does **not** prove default LLDB MCP bootstrap compatibility and does not close the bug.
- Conversely, the LLVM bug does not invalidate the policy text or justify forging/waiving hosted checks. It is an independently tracked local setup defect.

## Final recommendation

Proceed only through the fresh replacement lane above. The semantic replay is low-risk and auto-merges cleanly, but direct landing from revision 3 would conflate an obsolete base, a non-final Task CR, overlapping new trunk policy, and unrelated dirty user work. Fresh exact-tree review is the required boundary.
