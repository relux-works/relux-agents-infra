# Revision 4 independent review scope

Review the immutable latest Change Request for `TASK-260829-3fozxa`; do not
review the moving worktree and do not select an older revision.

In addition to the Task checklist, prove that the candidate composes with
current `origin/main` commit `675f77ed63376320ed1213f46f9462a299c0abaf`.
The candidate was produced from historical protected base
`891de4427bb7de6885b8b221f0e2b24a49a8fdc2`, while current trunk already owns
persisted restart/quarantine status fields. The review must inspect the exact
overlap and refuse if replay would lose, shadow, or report stale recovery state.

Required verdict evidence:

- exact CR revision, base OID, tree OID, and patch digest;
- complete changed-path review;
- focused rotation/retention/zero-limit/overflow tests;
- replay or three-way composition check against exact current trunk;
- full configured validation commands, or explicit rejection if they cannot
  be reproduced without weakening a gate.
