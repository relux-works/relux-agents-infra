# Revision 2 independent review scope

Review only the immutable latest Change Request for `TASK-260829-1qh0ud`.
Do not infer acceptance from the producer worktree or its handoff summary.

The review must prove that pressure/resource snapshots cannot be laundered into
healthy availability through either status or broker surfaces while the
pressure latch is active. Inspect cross-platform unsupported and unreadable
states, stale-snapshot behavior, recovery clearing, all public status fields,
and the exact production bypass mutant. Re-run the configured full validation
commands and record exact CR revision/base/tree/patch identities.
