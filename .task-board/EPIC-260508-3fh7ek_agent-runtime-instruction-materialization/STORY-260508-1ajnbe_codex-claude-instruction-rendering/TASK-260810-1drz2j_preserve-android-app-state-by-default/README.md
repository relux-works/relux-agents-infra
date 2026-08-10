# TASK-260810-1drz2j: preserve-android-app-state-by-default

## Description
Add a global Android workflow instruction that preserves installed app and user state by default and avoids unnecessary reinstall, uninstall, clear-data, and force-stop operations that create avoidable human re-interaction.

## Scope
Shared instruction source module, generated entrypoints, setup sync, and focused verification only.

## Acceptance Criteria
Global instructions require non-destructive Android verification first; destructive or state-resetting app operations need demonstrated necessity, exact scope, and awareness of human approval/re-interaction cost; installed global and project runtime copies are synced and verified.
