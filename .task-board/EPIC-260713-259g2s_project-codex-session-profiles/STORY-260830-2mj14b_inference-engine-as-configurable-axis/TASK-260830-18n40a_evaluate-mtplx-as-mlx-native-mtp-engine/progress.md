## Status
backlog

## Review
required

## Task Class
code

## Estimate
notEstimated

## Blocked By
- TASK-260830-3euwsu

## Blocks
- (none)

## Checklist
- [ ] Establish the weights question first: our MLX 8-bit build DROPPED the MTP head, so determine which artifact MTPLX needs to speculate at all
- [ ] Verify the vendor claim of 1.6x-2.24x independently on this host and this model rather than citing it
- [ ] Speculative decoding must be scored separately from a parity comparison, exactly as MTP-off parity was required for llama.cpp
- [ ] Characterise its OpenAI-compatible surface against the Pi contract: tool calls, reasoning content, streaming shape, health and readiness
- [ ] Report exact rejection sampling as a correctness property, not only a speed one — a faster wrong answer is not a win
- [ ] If MTPLX cannot serve the configured Qwen with an MTP head, say so with evidence; that is a valid result

## Notes

## Precondition Resources
(none)

## Outcome Resources
- [mtplx-first-look.md](file://TASK-260830-18n40a/mtplx-first-look.md)

## Created
2026-08-29T22:25:07Z

## Last Update
2026-08-30T09:15:47Z
