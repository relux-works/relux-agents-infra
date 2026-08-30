# STORY-260831-2829gr: replay-accepted-pi-adapter-on-current-trunk

## Description
Replay the independently accepted generic Pi adapter revision 3 onto the exact current signed agents-infra trunk without changing its reviewed architecture or contacting a live local model.

## Scope
agents-infra adapter and dependency files from accepted CR revision 3 only; board state and unrelated dirty files remain outside the candidate.

## Acceptance Criteria
The accepted revision 3 patch is reproduced on exact current origin/main, all tree-bound static/fake/full/race/vet/build/cross-platform/mutation/no-live-runtime gates pass, an independent reviewer accepts the exact fresh-trunk CR, and the signed reviewed head lands through the canonical PR flow.
