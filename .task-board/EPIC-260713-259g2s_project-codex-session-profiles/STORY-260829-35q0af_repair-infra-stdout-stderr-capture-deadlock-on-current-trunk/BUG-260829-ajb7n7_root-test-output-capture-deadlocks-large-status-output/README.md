# BUG-260829-ajb7n7: root-test-output-capture-deadlocks-large-status-output

## Description
tools/agents-infra/main_test.go captureStdout and captureStderr invoke the producer before draining os.Pipe, so output beyond pipe capacity blocks the callback and prevents the reader from starting. Fix both helpers with concurrent draining and safe descriptor restoration; add deterministic multi-megabyte regressions and rerun the exact root status test that hung during TASK-260829-3fozxa review.

## Scope
(define bug scope / affected area)

## Acceptance Criteria
Both stdout and stderr capture helpers drain concurrently, restore global descriptors on every tested path, and return byte-identical output beyond pipe capacity; regressions fail on the old implementation; the exact hung root status test and complete root test package exit 0 within existing timeouts without skips, sleeps, or timeout inflation.
