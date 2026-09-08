# TASK-260831-2byv3q: explain-the-anonymous-soak-climb

## Description
The 2026-08-31 pair recorded a +6,201,119,136 B anonymous climb across the candidate soak window. It is a two-point drift off a window that never faced the coverage gate, so it is not established as a leak, but it is unexplained and the study requires it explained before any deployment decision, whichever way the rest goes.

## Scope
(define task scope)

## Acceptance Criteria
The climb is either reproduced with coverage-gated instrumentation and explained by a named mechanism, or shown to be an artifact of the ungated two-point measurement. A deployment decision is not taken on this runtime while the observation stands unexplained.
