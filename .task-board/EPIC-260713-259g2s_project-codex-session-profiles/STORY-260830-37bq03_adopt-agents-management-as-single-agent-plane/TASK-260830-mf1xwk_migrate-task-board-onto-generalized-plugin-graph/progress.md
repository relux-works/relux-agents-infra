## Status
done

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

## Blocks
- (none)

## Checklist
- [ ] Sequenced so the board can spawn and route at every step; a window where this board cannot run its own migration is a failed plan, not an inconvenience
- [ ] Prefer a compatibility surface over the new graph first, so task-board keeps working unreleased-unchanged, then migrate it deliberately
- [ ] task-board own test suite green on the new contract, plus one real spawn and one real review routed on this live board
- [ ] Version bump and release order stated explicitly: which repository releases first and what happens to a consumer pinned to the previous version
- [ ] Rollback path written down before the breaking change lands, not after

## Notes
2026-09-11 goal audit: delivered in skill-project-management (board-cli v0.5.2..v0.5.10, b815801e..de938adf); this board already spawns on it. Not agents-infra work. Set done.

## Precondition Resources
(none)

## Outcome Resources
(none)

## Created
2026-08-29T22:47:16Z

## Last Update
2026-09-11T12:40:44Z
