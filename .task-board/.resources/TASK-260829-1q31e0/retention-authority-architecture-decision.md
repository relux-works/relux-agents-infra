# TASK-260829-1q31e0 retention authority architecture decision

The user requires a local-model harness that remains stable for hours, days, and weeks, with bounded retained diagnostics. Existing draft abstractions may be redesigned. No live model/runtime/service/socket/endpoint may be contacted.

Revision 6 proved that passive facts stored in a same-UID user-owned filesystem tree cannot be unforgeable to another process of that UID. Before expanding the product with a new daemon, separate the actual safety boundary from an impossible or meaningless one.

## Questions the architecture must answer

1. What is the real threat model? A same-UID process that can mint every filesystem fact can also directly unlink or rewrite the target file. Determine whether preventing that process from asking our retention code to perform the same deletion creates meaningful additional authority, or whether same UID is already the OS trust boundary.
2. Distinguish malicious same-UID construction from accidental collision, stale/corrupt metadata, symlink/path substitution, ownership changes, and our own buggy overbroad scanner. The latter must remain fail-closed even if the former is explicitly outside the security contract.
3. Compare at least:
   - a profile-scoped retention-owner process holding kernel-backed identities;
   - a dedicated launcher-owned namespace plus cryptographically random collision-resistant records while explicitly trusting same UID;
   - zero-persistence semantics;
   - any cleaner existing agents-infra/session-manager/shared-broker ownership boundary already present in source.
4. Evaluate each option against week-scale availability, crash/restart recovery, aggregate count/byte/age bounds, retained diagnostics, implementation complexity, Darwin MLX priority, Linux/Windows behavior, status observability, and denial-of-service/fail-closed behavior.
5. If an owner process is chosen, specify exact lifecycle, singleton/lease authority, protocol, upgrade and crash recovery, orphan handling, status fields, and why it does not become a new long-running single point of failure.
6. If same UID is explicitly trusted, define a narrow contract that still prevents ordinary retention code from deleting unrelated files: dedicated root, no-follow/path containment, identity revalidation, random ownership records, atomic creation/update, corruption/unknown preservation, and production-entry negative tests. State plainly that the marker prevents accidental collision and code bugs, not a malicious same-UID actor.

## Required output

- One recommended architecture and rejected alternatives with evidence from current source.
- A precise threat model and ownership/deletion-authority contract.
- State/lifecycle and recovery design sufficient for implementation without guessing.
- A minimally schedulable implementation/review plan and exact adversarial tests.
- Updates to the task description/acceptance/checklist if the current impossible requirement must be corrected.
- Persist the decision as a task outcome and route the task to `to-dev` only when the design is implementation-ready. Do not edit production code or publish a code Change Request in this architecture pass.

