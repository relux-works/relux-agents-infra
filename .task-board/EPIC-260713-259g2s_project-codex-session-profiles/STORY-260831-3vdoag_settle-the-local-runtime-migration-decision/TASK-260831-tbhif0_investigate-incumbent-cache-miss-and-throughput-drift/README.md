# TASK-260831-tbhif0: investigate-incumbent-cache-miss-and-throughput-drift

## Description
Two incumbent findings that matter regardless of any migration. The baseline is deployed with --prompt-cache-size 1 --prompt-cache-bytes 8GB, yet sealed cached_tokens show it never fired: llama.cpp hit ([5736, 7780, 7809] and [18]x20) while the baseline missed ([0]). Separately, the incumbent's own throughput moved 1.46-1.51x between measurement campaigns, a drift larger than the decode advantage the candidate showed.

## Scope
(define task scope)

## Acceptance Criteria
The prompt cache either fires under a configuration we can state, or the reason it cannot is established and the flags are corrected or removed rather than left as decoration. The throughput drift is attributed to a named cause. Both findings are recorded whether or not they favour the incumbent.
