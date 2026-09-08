# TASK-260829-1q31e0 architecture revision 2 validation

Date: 2026-08-29, Europe/Moscow.

## Evidence

- `task-board` readiness and `git version` were recorded in
  `.temp/TASK-260829-1q31e0/tool-readiness.log`.
- Exact source `HEAD` is
  `6d051f54440d36e3ca3d132f8d9d1e78d46289de`; `HEAD..main` count is `0`.
- The Story worktree already contains an uncommitted producer candidate. This
  architecture run did not edit, stage, reset, or absorb its implementation
  files.
- Current-trunk evidence was read with `git show HEAD:<path>` for
  `pi_state.go`, `pi_session_log.go`, and `main.go`; board evidence was read
  through compact task-specific projections and materialized resources.
- Revision 2 closes both orchestrator findings: bounded lock/scan liveness and
  an explicit safe legacy upgrade path.
- No diagram was produced. Operation and state tables in the architecture
  decision express the relevant relationships more precisely than a separate
  C4/UML artifact.
- No build/test was run because this pass changes board architecture only and
  does not claim implementation behavior. Negative production-path gates are
  specified for the developer and reviewer Tasks.
- No live model, runtime, service, socket, endpoint, Git ref, or external
  system was contacted or mutated.

## Decomposition verdict

Two code Tasks are proportional:

1. the existing aggregate retention engine;
2. dependent `TASK-260829-2v7x1u`, the explicit legacy retirement operator
   flow.

The second Task is a justified gap, not ceremonial scope: without it existing
clean installations can never regain `soak_ready=true`. Research, diagrams,
documentation-only, soak-only, daemon, and review Tasks remain rejected.
