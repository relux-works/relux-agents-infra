# STORY-260830-884cmp: replay-global-external-ci-fallback-on-fresh-trunk

## Description
Replay the independently audited external-hosted-CI fallback policy on the exact freshly fetched protected trunk, preserving generated Claude and Codex instruction parity.

## Scope
(define story scope)

## Acceptance Criteria
Fresh Story selected base equals fetched upstream main; fallback triggers only for verified external unrepairable hosted-CI non-execution; exact clean PR head mirrors every workflow job and explicit environment/service locally; durable evidence records SHA, platform, tools, commands, and exits; repository failures block; remote statuses are never forged; generated instruction surfaces and negative mutants pass; independent review accepts the exact CR.
