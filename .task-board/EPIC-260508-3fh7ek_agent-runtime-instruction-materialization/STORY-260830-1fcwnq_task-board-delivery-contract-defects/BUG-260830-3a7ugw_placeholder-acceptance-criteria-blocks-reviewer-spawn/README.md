# BUG-260830-3a7ugw: placeholder-acceptance-criteria-blocks-reviewer-spawn

## Description
Elements can be created and parked with literal placeholder acceptance criteria such as '(define acceptance criteria)'. The board accepts that state at creation, then refuses the reviewer spawn later, so the defect surfaces only at review time on three separate tasks. Worse, the natural repair is to write criteria after the work exists, which invites fitting the criteria to the delivered result; the criteria here had to be reconstructed deliberately from stated intent instead. This bug was reported once before and its record did not persist.

## Scope
(define bug scope / affected area)

## Acceptance Criteria
The board refuses to park or advance an element whose acceptance criteria are empty or are an unedited template placeholder, at the point the state is entered rather than at reviewer spawn, and the refusal names the element and the exact field to populate. Detection matches the template placeholders the board itself emits, anchored as the whole field value; prose that merely mentions the word placeholder is not flagged. A false positive on legitimate criteria is a defect of equal severity, because it would block delivery rather than merely permit a bad state.
