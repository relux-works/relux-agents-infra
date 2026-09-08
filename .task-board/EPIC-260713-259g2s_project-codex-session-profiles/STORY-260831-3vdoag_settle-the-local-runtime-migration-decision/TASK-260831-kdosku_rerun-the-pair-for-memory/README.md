# TASK-260831-kdosku: rerun-the-pair-for-memory

## Description
Re-run the pinned pair once the memory instrument is fixed at its design, and only then. Section 7.2 item 2.

## Scope
(define task scope)

## Acceptance Criteria
The full pinned pair runs sequentially on an idle host with MTP and speculation off, and peak resident memory is scored on both runtimes for context_75k, or the dimension is refused for a reason that is not instrument coverage. The result is compared against the study's stated weighting and the outcome recorded whichever way it falls.
