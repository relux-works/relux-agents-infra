# TASK-260831-1bt8f4: replay-accepted-retention-after-adapter

## Description
After TASK-260831-26b034 is fully integrated, provision this new Story from exact current origin/main and replay only the independently accepted retention/restart/quarantine/rotation/status/soak delta. Use old revision 1 and revision 6 evidence as immutable intent; do not reuse the contaminated old Story branch or widened revision 2.

## Scope
Accepted retention implementation paths plus minimum conflicts in files also changed by the landed generic Pi adapter. CR path count must be derived against this new workspace selected base and every path beyond the original accepted set must be named and justified.

## Acceptance Criteria
1. Hard dependency on TASK-260831-26b034 remains until its Story integration is done. 2. New workspace selected/local/upstream base OIDs all equal fetched current origin/main. 3. Old 26-path accepted functional delta is reconstructed without the 110-path widening. 4. Every adapter-overlap path is semantically unioned and listed. 5. Full/race/vet/build/format/diff, lifecycle mutants, cross-platform, isolated parity, eight-week soak, and no-live-runtime gates pass. 6. Independent reviewer accepts exact fresh-trunk CR and signed PR head lands.
