# TASK-260825-imfmgz: design-shared-runtime-owner-contract

## Description
Specify the trusted local runtime owner and lease protocol that permits multiple Pi clients to share one verified Qwen model server.

## Scope
Produce an implementation-ready specification for a terminal-independent local runtime broker: owner identity, single-flight startup, same-profile attestation, loopback endpoint handoff, RUN/process-bound leases and heartbeats, crash recovery across independent spawn-runner process groups, per-Pi session/state isolation, final-release shutdown, operator status/stop, and refusal of unrelated listeners. The required reference scenario is an orchestrator issuing two separate tracked spawns, not two Pi children of one shell.

## Acceptance Criteria
A focused .spec document defines states, transitions, persistence, IPC and OS/process boundaries, integration with task-board/agents-infra child launch, failure recovery, security assumptions, typed refusals, CLI surfaces, and tests. It names the exact proof: two independent RUN handles and Pi PIDs, overlapping execution, one attested Qwen runtime PID, and cleanup after both terminal states.
