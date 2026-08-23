# TASK-260824-1glviz: carry-accepted-vendor-target-candidate

## Description
Reconstruct the accepted CR-TASK-260824-2a4gk3-2 delta on a fresh worktree based on current main. Use base cf21665dde35274cc14e66e26a93574e0c18c15c and candidate tree 95d12fb4c2aaf6050ff51f2c74fee7a81041acff as authoritative inputs. Preserve current-main changes on README.md, SKILL.md, infra.go, and infra_test.go through a semantic three-way merge; candidate LOGBOOK.md is a reviewed strict superset and may be retained. Do not redesign accepted behavior.

## Scope
Fresh-base mechanical reconciliation, focused/full validation, source setup/install verification, alias provenance checks, and final story integration evidence.

## Acceptance Criteria
Fresh candidate semantically equals the merge of current main and accepted candidate 95d12fb4, all accepted tests and three alias provenance checks pass, current-main fast-profile work is preserved, no listener/process/lock remains, and a story_final CR is accepted.
