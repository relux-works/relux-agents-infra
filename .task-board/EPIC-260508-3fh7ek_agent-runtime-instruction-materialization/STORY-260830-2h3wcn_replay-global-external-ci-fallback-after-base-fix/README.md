# STORY-260830-2h3wcn: replay-global-external-ci-fallback-after-base-fix

## Description
Replay the global external-CI local-mirror policy on the exact fetched protected trunk after installing selected-base enforcement.

## Scope
Global instruction source, generated-surface tests, README and logbook only; preserve unrelated infra work and do not edit installed runtime locations directly.

## Acceptance Criteria
Fresh workspace branch and CR base equal fetched origin/main; policy triggers only when hosted CI cannot execute repository steps for a verified external cause the agent cannot repair; exact clean PR head, all jobs and explicit env/services are mirrored locally; SHA/platform/tool/command/exit evidence is durable; repository failures block; remote statuses are never forged; Claude and Codex generated instruction surfaces are tested; independent review accepts and source setup installs the result.
