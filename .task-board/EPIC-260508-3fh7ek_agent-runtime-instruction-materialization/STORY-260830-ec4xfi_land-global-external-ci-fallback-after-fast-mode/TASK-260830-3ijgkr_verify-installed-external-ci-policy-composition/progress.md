## Status
closed

## Review
required

## Task Class
code

## Estimate
notEstimated

## Blocked By
- (none)

## Blocks
- (none)

## Checklist
- [ ] Start from the accepted checkpoint and confirm its candidate tree contains both policy and fast-mode changes
- [ ] Run canonical setup without editing installed runtime files directly
- [ ] Prove source, generated Claude and Codex, and installed instruction surfaces are byte-consistent
- [ ] Kill broadened-trigger and both include-bypass mutants on the exact final candidate
- [ ] Attach validation evidence and publish an empty or corrective final Change Request for independent review

## Notes
2026-09-11 goal audit: closed as superseded duplicate. Global external-CI local-mirror fallback policy landed via STORY-260830-11fnea / PR #23 (4270549); candidate text byte-identical to main INSTRUCTIONS_WORKFLOW.md. Verification tests exist on main (infra_test.go broadened/additive mutants).

## Precondition Resources
(none)

## Outcome Resources
(none)

## Created
2026-08-30T08:43:24Z

## Last Update
2026-09-11T12:39:35Z
