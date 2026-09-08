# TASK-260829-1q31e0 revision 6 architecture blocker

## Constraint

Revision 6 requires durable deletion authority that an ordinary same-UID foreign caller cannot construct, while retaining lifecycle logs across launcher process lifetimes and pruning them later. The current design stores every authority fact in a user-owned profile tree. Under discretionary POSIX/macOS filesystem authority, another process with the same UID can create or mutate every passive fact the launcher can create there: names, regular files, links, modes, ACLs, xattrs, user flags, lock files, sidecars, and readable secrets.

The only authority in the current implementation that is not durable and self-mintable is the live open file description plus its held lock. It disappears when the launcher exits. A later launcher therefore cannot distinguish a prior launcher-created closed file from a caller-created file without trusting another same-UID-mintable record.

## Evidence

- Immutable revision-5 review drove production `openPiSessionLogAt` and proved that a caller-created lifecycle name, mode, inode/link count, and ownership-namespace hard link are deleted. Reviewer log: `TASK-260829-1q31e0_review-negative-self-minted-provenance-rev5.log`.
- The local task-scoped probe `same-uid-filesystem-authority-probe.zsh` exits 0 and proves the same UID can construct the hard-link pair, reopen a mode-0500 namespace, restore a mode-0000 file to 0600, mint an xattr, and clear the user immutable flag. Evidence: `TASK-260829-1q31e0_same-uid-filesystem-probe-rev6.log`.
- `piSessionLog.event` can safely authorize only its current open fd/path identity. `scanPiLifecycleLogs` and `removePiLifecycleLog` operate after arbitrary process boundaries and currently have no authority source outside the same-UID filesystem tree.

## Failed assumptions and rejected workarounds

- Deterministic or random hard-link sidecars: the same UID can create them.
- A hidden token, HMAC key, or manifest stored below the profile/cache tree: the same UID can read or replace it; moving the self-minted claim does not strengthen it.
- Mode 0500/0400, ACL, xattr, or `uchg`: the owning UID can reopen, rewrite, mint, or clear them; the attached probe demonstrates representative cases.
- Birth time, inode generation, or other kernel-assigned file metadata: a foreign caller can create a new file and receive valid metadata for its own inode; without an independent trusted manifest those values do not identify the launcher.
- Conservatively preserving every unverifiable closed file while retaining normal history: safe from foreign deletion, but old launcher logs become unverifiable after process exit, cannot be pruned, and eventually violate count/byte/age retention or permanently refuse new launches.
- Adding another filesystem check/test around the revision-5 pair: forced fit; it raises attack cost without creating an independent owner.

## Viable options

1. Profile-scoped retention owner process. It exclusively creates lifecycle files, retains their open file descriptions/locks after clients close, prunes only those kernel-held identities, and returns log descriptors to clients. After owner death, prior files become unknown and are conservatively preserved/refused. This is portable in principle and preserves retained diagnostics, but adds a daemon protocol, lifecycle, recovery, status, and denial-of-service boundary beyond the assigned configuration/filesystem-test scope.
2. Platform-backed identity/signing authority. Use a code-identity-bound Keychain/keystore or privileged helper to sign inode-bound provenance. This is platform-specific, needs install/upgrade/key rotation policy, and conflicts with current pure user-space cross-platform behavior unless separately designed.
3. Zero-persistence lifecycle logs. Keep the active log bounded, then unlink it on verified close. This is safe and minimal but removes retained per-turn diagnostics and makes count/age retention configuration largely meaningless.
4. Relax the threat boundary. Declare same-UID processes trusted and retain the revision-5 shape. This directly contradicts the revision-6 requirement and the reviewer attack model.

## Recommendation and exact decision needed

Recommend option 1 if retained lifecycle diagnostics remain a product requirement; it provides a real owner rather than passive evidence. Choose option 3 only if bounded active diagnostics are sufficient and retained history can be dropped.

Required decision: authorize and specify the profile-scoped retention-owner architecture (supported platforms, startup/upgrade/crash recovery, client protocol, and availability behavior), or explicitly choose zero-persistence semantics, or relax the same-UID attacker requirement. Product-code rework cannot honestly continue until one boundary is selected.
