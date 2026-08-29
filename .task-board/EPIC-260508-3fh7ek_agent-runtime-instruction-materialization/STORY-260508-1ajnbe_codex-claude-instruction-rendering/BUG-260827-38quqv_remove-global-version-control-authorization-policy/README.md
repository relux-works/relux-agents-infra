# BUG-260827-38quqv: remove-global-version-control-authorization-policy

## Description
Global generated agent instructions duplicate commit and push authorization policy that task-board version_control already governs for tracked work.

## Scope
Remove global commit/push authorization and commit-style prose from the source workflow instructions; update tool cross-references; preserve dirty-checkout and worktree safety; regenerate/install the managed global instruction surface; verify generated AGENTS no longer contains the removed policy and board version_control remains authoritative.

## Acceptance Criteria
Source workflow instructions contain no global commit/push authorization or commit-message style policy; tooling instructions no longer reference the removed section; dirty checkout and worktree safeguards remain; generated root and installed global AGENTS surfaces reflect the change; focused/full infra checks and task-board validation pass.
