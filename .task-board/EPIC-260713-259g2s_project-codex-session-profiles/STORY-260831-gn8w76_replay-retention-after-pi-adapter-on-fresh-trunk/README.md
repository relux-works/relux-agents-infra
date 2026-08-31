# STORY-260831-gn8w76: replay-retention-after-pi-adapter-on-fresh-trunk

## Description
Replay the independently accepted Pi lifecycle retention and long-horizon status delta only after the generic Pi adapter has landed, using a new managed Story workspace provisioned from the then-current signed origin/main.

## Scope
Accepted retention revision 6 functional paths and the minimum semantic composition with the landed generic Pi adapter. No inherited old Story workspace, unrelated research/article/config paths, or live local runtime access.

## Acceptance Criteria
The sole implementation task is hard-blocked by the fresh-trunk Pi adapter landing; its workspace selected base equals fetched origin/main after that dependency completes; the immutable CR contains only the accepted retention path set plus explicitly reviewed adapter composition paths; all retention, race, soak, cross-platform, parity, and no-live gates pass; independent review accepts and canonical integration lands the exact signed head.
