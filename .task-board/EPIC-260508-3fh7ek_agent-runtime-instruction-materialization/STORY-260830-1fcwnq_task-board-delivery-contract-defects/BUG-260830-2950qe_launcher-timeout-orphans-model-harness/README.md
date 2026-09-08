# BUG-260830-2950qe: launcher-timeout-orphans-model-harness

## Description
When a spawn run exceeds its launcher timeout and is terminated with exit 124, model-harness child processes survive as orphans reparented to init. Observed twice on TASK-260829-3k4qrc. The first occurrence retained model-harness plus a Python child holding about 31 GiB until external process-group signalling, and its evidence archive lacked a raw process snapshot. The second occurrence, from RUN-260830-332081 exiting 124 after a 90 minute timeout, was captured: PID 60219, PPID 1, own process group 60219, RSS 10672 KB, elapsed 13:14, running 'model-harness run gate-smoke-baseline' against a decoy profile. Snapshot preserved at .temp/TASK-260829-3k4qrc/b7-orphan-snapshot-120755.txt. The orphan holds a listening port, so a rerun on the same port fails with EADDRINUSE. This is blocker B7 reproduced in production.

## Scope
(define bug scope / affected area)

## Acceptance Criteria
A launcher timeout terminates the entire owned process tree, not only the direct child. No model-harness or runtime child survives with PPID 1 after a run exits on timeout. The termination is proven by a test that starts a run which spawns a harness child, forces the timeout path, and asserts no surviving descendant and no listening socket on the run's ports. When a survivor is nevertheless detected, the run records a raw process snapshot automatically rather than relying on an operator to capture one.
