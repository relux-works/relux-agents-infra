## Status
backlog

## Review
required

## Task Class
code

## Estimate
notEstimated

## Blocked By
- TASK-260830-1jpse1
- TASK-260830-s5ro4e
- TASK-260830-11ajl2
- TASK-260831-1pfnxx

## Blocks
- (none)

## Checklist
- [ ] Sequenced so the board can spawn and route at every step; a window where this board cannot run its own migration is a failed plan, not an inconvenience
- [ ] Prefer a compatibility surface over the new graph first, so task-board keeps working unreleased-unchanged, then migrate it deliberately
- [ ] task-board own test suite green on the new contract, plus one real spawn and one real review routed on this live board
- [ ] Version bump and release order stated explicitly: which repository releases first and what happens to a consumer pinned to the previous version
- [ ] Rollback path written down before the breaking change lands, not after

## Notes

## Precondition Resources
(none)

## Outcome Resources
(none)

## Created
2026-08-29T22:47:16Z

## Last Update
2026-08-31T02:58:23Z
