# TASK-260830-27u51n: publish-global-external-ci-local-mirror-policy

## Description
Add the external-CI fallback to the global workflow source and prove generated surfaces and installed runtime stay synchronized.

## Scope
Edit only the versioned source in an isolated Story worktree, preserve unrelated dirty main-checkout instruction work, regenerate via the repository setup path, and test source-to-output parity.

## Acceptance Criteria
Fallback applies only when hosted CI cannot execute repository steps for a verified external cause the agent cannot repair; every affected job is reproduced on exact clean PR head with matching commands/toolchain/environment/services/target or documented equivalent; evidence records SHA job platform tool versions commands and exits; repository failures block; hosted status is never forged; remote review and protection remain authoritative; setup and tests prove generated Claude/Codex surfaces contain the policy.
