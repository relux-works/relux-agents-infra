# TASK-260829-3cwcb6: correct-cross-runtime-measurement-asymmetries

## Description
Correct the cross-runtime measurement defects that made llama.cpp look better than it is, and audit for the symmetric class that would make it look worse, so the migration decision rests on instrumentation with no known directional bias.

## Scope
(define task scope)

## Acceptance Criteria
Reasoning-delta field naming, mmap-invisible memory accounting and any further directional measurement defect are corrected or explicitly declared unmeasurable, an audit for defects biased AGAINST llama.cpp is performed and reported, and every corrected metric carries a production-entry negative proving the old reading is refused.
