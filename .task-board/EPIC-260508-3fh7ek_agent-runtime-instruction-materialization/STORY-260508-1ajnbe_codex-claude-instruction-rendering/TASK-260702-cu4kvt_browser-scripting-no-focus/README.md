# TASK-260702-cu4kvt: browser-scripting-no-focus

## Description
Document that browser automation and scripting must avoid focusing or activating the user browser by default. Use background/browser-session tooling and only focus windows when the user explicitly requests it.

## Scope
(define task scope)

## Acceptance Criteria
Global instructions state that browser automation and page inspection are no-focus by default, name the background-capable tooling to use, forbid the AppleScript activate and frontmost-window patterns, and say when a human must take over instead — with the browser-secret handling rules preserved.
