# BUG-260830-2uo1vq: concurrent-stories-on-one-base-restale-cascade

## Description
The board permits N Stories to hold accepted Change Requests against the same base simultaneously, and every integration moves trunk and invalidates all remaining ones. Observed with four concurrent Stories, all ahead=0: STORY-260830-2vrhg1 at main, STORY-260830-3uerxs 4 behind, STORY-260830-tk737f 2 behind, STORY-260829-26nbbv 13 behind. Each restale currently costs a full producer plus reviewer cycle even when the delta is provably byte-identical across the base move, because a stale Change Request must be republished and re-reviewed. This is the dominant delivery cost observed over two days and it grows with the number of parallel Stories, which directly opposes the instruction to run independent Stories in parallel.

## Scope
(define bug scope / affected area)

## Acceptance Criteria
Either integration does not invalidate sibling Change Requests whose delta does not overlap the integrated paths, or a stale Change Request whose delta is byte-identical across the base move can be revalidated without a full re-review cycle. The board states the chosen contract explicitly and reports, at integration time, which sibling Stories it invalidated and why.
