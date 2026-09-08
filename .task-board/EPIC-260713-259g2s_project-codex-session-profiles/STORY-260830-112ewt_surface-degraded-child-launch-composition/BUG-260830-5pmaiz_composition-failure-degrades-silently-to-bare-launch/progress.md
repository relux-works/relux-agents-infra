## Status
backlog

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
- [ ] Degradation is visible without reading a spawn note line by line; a run that lost its MCP composition is not indistinguishable from a healthy one
- [ ] One invocation reports every missing required field for the profile, not the first one and then stop
- [ ] The error names the field path and what a valid value looks like, as the board-side error contract requires
- [ ] Fail-closed validation is preserved; nothing here adds an automatic migration or a silent default
- [ ] Regression test drives the real compose path with a config missing several required fields

## Notes

## Precondition Resources
(none)

## Outcome Resources
(none)

## Created
2026-08-30T00:00:44Z

## Last Update
2026-08-30T00:00:48Z
