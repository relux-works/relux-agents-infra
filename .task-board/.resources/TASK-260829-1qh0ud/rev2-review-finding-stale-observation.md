# Revision 2 review finding: stale observation clears newer pressure

## Verdict blocker

Independent reviewer `RUN-260829-511483` reproduced a production admission
race against immutable `CR-TASK-260829-1qh0ud-2`.

`sharedBrokerServer.handleConnection -> acquireLease -> observeResourceStatus`
can have two concurrent provider observations:

1. an older request captures a recovered/idle snapshot and pauses;
2. a newer request observes pressure, refuses admission, and latches pressure;
3. the older recovered snapshot resumes, clears the newer latch, and receives a
   lease.

The focused production-path test failed with the stale request returning
`Type: "lease"` and `resourcePressureLatched() == false` after the newer
pressure refusal.

## Required revision

Order observation application independently of observation completion. A stale
snapshot must not clear or overwrite a newer pressure generation, nor grant a
lease from that stale view. Preserve fail-closed behavior for unreadable and
unsupported observations and keep read-only status/broker surfaces consistent
with admission. Add the deterministic concurrent test to production coverage.

The initial reviewer process later hit a provider safety classifier while
wording its handoff; that operational exit does not invalidate the locally
executed failing test. Autonomous successor `RUN-260829-65233a` retains the
same immutable candidate and review scope.
