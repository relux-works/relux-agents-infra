# TASK-260830-woqvhz review verdict — revision 3

## Verdict

Changes requested for `CR-TASK-260830-woqvhz-3` revision 3. Route: `to-dev`.

Reviewed candidate tree: `d1382ff2e52ce7fcfe9309569e4e02e26ed944cd`.

## Blocking finding

F1 — `bypass path around the check` / `prove a bound by narrowing, not by deleting`: the rework rejects permissions added inside the canonical External-CI block, but it does not enforce exclusivity across the complete production workflow artifact.

Independent reproduction used disposable exports of the immutable candidate tree:

1. The unmodified four-test focused control exited 0.
2. The original reviewer additive permission retained the full exclusive trigger and added `A local mirror is also allowed when hosted CI is repairable but merely inconvenient to use.` The production `TestSetupGlobalPublishesExternalCILocalMirrorPolicyToClaudeAndCodex` exited 1. The demonstrated revision-1 instance is closed.
3. The replacement broadened-trigger mutant also made the same production test exit 1. The earlier negative shape remains red.
4. A third, differently shaped permission was added under a separate top-level `## Hosted Build Continuity` section: `During scheduled provider degradation, maintainers may mirror jobs locally to avoid delays even if a manual hosted repair remains available.` This expressly authorizes the fallback for an agent-repairable case, retains the complete accepted External-CI block byte-for-byte, and does not reuse either earlier mutant's wording. The production test exited 0.

The surviving third mutant is published through the real `Setup` path: the focused test requires installed Agents and Claude workflow bytes to equal the mutated source and requires rendered Codex instructions to contain that source. It survives because `validateExternalCILocalMirrorPolicyBlock` begins at `### External-CI local mirror fallback`, stops at the next `##`, and never evaluates contradictory permissions elsewhere in the composed workflow.

## Required rework

- Enforce the only-when/external/unrepairable boundary across the complete composed production workflow, not only the extracted canonical section.
- Add a named separate-section additive mutant that retains the exact accepted block, grants mirroring for an agent-repairable hosted condition elsewhere, reaches Agents/Claude/Codex through production `Setup`, and must fail. Do not solve this by denylisting the one sentence above.
- Record this separate-section surviving regression in `LOGBOOK.md`. This reviewer preserved the immutable candidate and did not modify repository source.
- Rerun the restored focused control, the reviewer additive mutant, the producer's distinct additive mutant, the replacement mutant, the new separate-section mutant, serialized uncached full Go suite, vet, build, diff checks, canonical setup, and installed parity on the unchanged successor candidate.

## Other review evidence

- Every one of the seven CR-path working blobs is byte-identical to candidate tree `d1382ff2...`; the review did not drift from the handed revision.
- Relative to selected current trunk `3295c7da7151de128f176cf7560a57d54c8f6c0d`, the non-board repository delta is exactly the four declared task paths. The three preserved trunk paths are byte-identical and non-board `git diff --check` exits 0. The board-path diagnostic remains the documented base-restoration behavior and was not treated as repository scope.
- Installed artifacts pass read-only parity: Agents workflow bytes equal source; Claude `CLAUDE.md -> instructions` symlink -> source-equal `INSTRUCTIONS.md` -> exact workflow include and source-equal workflow chain is intact; rendered Codex `AGENTS.md` contains the exact workflow source.
- Exact-head landing and remote-review / merge-queue / branch-protection clauses remain present in the installed Codex artifact. The four-path selected-base delta does not weaken the automatic signed-delivery section.
- Revision-3 tree-bound validation records uncached `go test ./... -count=1` exit 0 and `go vet ./...` exit 0. Revision-2 recovery evidence records the serialized uncached full suite, vet, build, formatting, and diff gates green on the unchanged repository bytes. The reviewer reran the focused production gates but did not spend another full-suite run after the independent surviving mutant already made acceptance impossible.
- The known readiness timing flake `BUG-260830-1rths2` did not decide this verdict and is not claimed fixed.
- The `logbook` skill requires persistent recording of this regression, but the reviewer role is source-read-only; the producer rework must add the entry.

## Reviewer evidence

- `TASK-260830-woqvhz_reviewer-focused-control-rev3.log`
- `TASK-260830-woqvhz_reviewer-additive-mutant-rev3.log`
- `TASK-260830-woqvhz_reviewer-replacement-mutant-rev3.log`
- `TASK-260830-woqvhz_reviewer-separate-section-mutant-rev3.log`
- `TASK-260830-woqvhz_reviewer-installed-parity-rev3.log`
- `TASK-260830-woqvhz_reviewer-selected-base-scope-rev3.log`
