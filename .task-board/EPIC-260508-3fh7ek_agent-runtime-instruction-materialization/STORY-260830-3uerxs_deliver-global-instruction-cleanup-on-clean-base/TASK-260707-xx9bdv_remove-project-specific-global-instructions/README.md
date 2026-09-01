# TASK-260707-xx9bdv: remove-project-specific-global-instructions

## Description
Remove x-platform-airdrop/Tap2Cash-specific material from global relux-agents-infra instructions, preserve the removed material under task temp artifacts for later project-local placement, and refresh the installed global runtime through agents-infra setup global.

## Scope
(define task scope)

## Acceptance Criteria
No project-specific x-platform-airdrop or Tap2Cash material remains in the global relux-agents-infra instructions, the removed material is preserved under a task-scoped temp artifact rather than deleted, and the remaining global instructions still read as complete without it.
