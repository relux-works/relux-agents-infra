# BUG-260830-1rths2: timing-sensitive-readiness-test-flakes-under-concurrent-load

## Description
TestPiLaunchReadinessServiceUnavailableStillHonorsRuntimeBoundsAtProductionEntry fails under concurrent load and passes in isolation. First observed failing in a 297s suite under 3 concurrent runs, then passing 5/5 alone. Observed again on 2026-08-30 under three concurrent spawns, where it failed Change Request rev2 validation for TASK-260830-woqvhz at command 1 of 2, cancelled run RUN-260830-c4b4e1, and forced an automatic successor run. Two distinct subtest failures show the shape is a race in the test design, not an insufficient timeout: 'times_out_while_runtime_remains_alive' read the side-effect file readiness-count before the production path created it (no such file or directory, then strconv.Atoi of an empty string), and 'refuses_after_owned_runtime_exits' observed code runtime_readiness_timeout where it wanted runtime_exited_early, meaning it asserts on which of two racing conditions fires first.

## Scope
The timing-sensitive readiness assertions and their bounds. Do not fix this by widening a timeout until it stops failing; a bound chosen to outrun today's load is the same defect with a larger number.

## Acceptance Criteria
The test asserts on a condition that is deterministic under load: it must not read a side-effect file that the production path may not have written yet, and it must not depend on which of two racing terminal conditions is observed first. Do not fix this by widening a timeout until it stops failing. The fix is demonstrated by running the test under at least the concurrency that reproduced the failure, repeatedly, with no flake.
