# BUG-260830-s187vu: acceptance-unrecordable-when-verdict-predates-accepting-run

## Description
accept_cr refuses with change_request_evidence_missing when the reviewer verdict predates the accepting run, so a correct verdict cannot be recorded by a later run and the review must be redone. The mirror variant is a reviewer spawned before the Change Request revision exists, which produces a verdict the board will not bind. A related constraint already recorded separately is that reusing or updating a pre-existing verdict file also fails this check. This bug was reported once before and its record did not persist.

## Scope
(define bug scope / affected area)

## Acceptance Criteria
A reviewer verdict that is valid for a specific Change Request revision can be recorded against that revision regardless of which run performs the acceptance, or the board refuses the reviewer spawn up front when the revision it would review does not yet exist. The error names which of the two conditions failed.
