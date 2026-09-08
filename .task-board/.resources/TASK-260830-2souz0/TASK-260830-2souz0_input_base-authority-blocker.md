# TASK-260830-1ugbb1 base-authority blocker

Observed 2026-08-30 04:33 MSK before any product-code change by run `RUN-260830-8c0ecb`.

## Constraint

The Task and accepted revision-3 architecture require fetched `origin/main`, direct GitHub `main`, the managed Story selected base, and workspace `HEAD` to equal protected post-pressure commit `5c9b4e4f7a88e1eb937b80851af522e4fa4b066f` before implementation. Fresh remote authority does not satisfy that gate.

## Evidence

| Authority | Observed OID | Command/result |
| --- | --- | --- |
| Workspace `HEAD` | `5c9b4e4f7a88e1eb937b80851af522e4fa4b066f` | `git rev-parse HEAD`, exit 0 |
| Local `main` | `5c9b4e4f7a88e1eb937b80851af522e4fa4b066f` | `git rev-parse main`, exit 0 |
| Managed Story `selected_base_oid` | `5c9b4e4f7a88e1eb937b80851af522e4fa4b066f` | `task-board worktree status --json`, exit 0 |
| Fetched `origin/main` | `3295c7da7151de128f176cf7560a57d54c8f6c0d` | `git fetch origin main && git rev-parse origin/main`, exit 0 |
| Direct GitHub `main` | `3295c7da7151de128f176cf7560a57d54c8f6c0d` | `git ls-remote origin refs/heads/main`, exit 0 |
| Remote-to-required ancestry | not established | `git merge-base --is-ancestor 3295c7d 5c9b4e4`, exit 1 |

The managed workspace already contains the preserved revision-2 candidate and `CR-TASK-260830-1ugbb1-2`; this run made no production-code or test changes and did not merge, replay, reset, rebase, or overwrite those bytes.

## Failed assumption

The fresh authoritative GitHub branch was expected to remain at the exact accepted post-pressure commit. Both fetch and a direct remote read disprove that assumption.

## Viable paths

1. Authorized repository owner restores and verifies protected GitHub `main` at exact `5c9b4e4f7a88e1eb937b80851af522e4fa4b066f`. This preserves the accepted architecture and current Story workspace.
2. If `3295c7da7151de128f176cf7560a57d54c8f6c0d` is intentional, explicitly revise the accepted base contract, independently re-establish the pressure dependency on the intended trunk, and provision a fresh managed Story workspace from the newly approved exact OID. This requires replay/overlap review and all validation gates again.

## Recommendation and exact unblock input

Prefer path 1 because the Task names `5c9b4e4` as the protected post-pressure authority. Resume only after an authorized owner supplies one of:

- remote evidence that GitHub `main` is restored to exact `5c9b4e4f7a88e1eb937b80851af522e4fa4b066f`; or
- an explicit architecture/base decision approving a different exact OID plus a reprovisioned Story workspace from that OID.

No Go test, race, vet, build, format, or cross-compilation gate was run by this run: the pre-implementation authority gate failed first, and running candidate acceptance gates would not cure it.
