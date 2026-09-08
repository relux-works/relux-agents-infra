# TASK-260830-84z0be review verdict — Change Request revision 2

## Verdict

Changes requested. Do not accept or integrate `CR-TASK-260830-84z0be-2`
revision 2.

The functional replay may be correct, but the immutable Change Request is not
the 26-path current-trunk candidate described by the producer evidence. It is a
110-path candidate recorded against the obsolete base `4270549`, so it violates
the task scope and cannot survive the production integration gate.

## Immutable identity reproduced

- CR base: `4270549dd17c010599e2083bf3ec7672af60ea29`.
- CR candidate tree: `78f283949dda1e4cab8b36b30e33695dc57909af`.
- CR patch SHA-256: `d0582b5f6c20d539d41497f06f3e16d530eb9365f430b69aacfe87ac1e568807`.
- Reviewer-owned alternate-index application of the CR patch to the recorded
  base reproduced candidate tree `78f283949dda1e4cab8b36b30e33695dc57909af`.
- Fresh `git fetch origin main` resolved `origin/main`, local `main`, and the
  workspace `HEAD` to `0d1641a0ab8fe47a98d6a54a81524a37e1cc6ead`.
- The accepted revision-6 source remains exactly 26 paths and its attachment
  remains SHA-256 `1ed35314955527822a6211f11510c10582aa9f588e9558731d1321b837c117ad`.

## Blocking finding

The CR contains all 26 accepted paths, but also 84 paths introduced by the nine
trunk commits between `4270549` and `0d1641a`. The widened set includes unrelated
research, article artifacts, `task-board.config.json`, model-harness changes,
and the MLX Swift prototype. This independently reproduces the board's 110-path
metadata; no accepted path is missing.

The producer artifact instead claims base `0d1641a`, candidate tree `5e9ae12d`,
patch SHA-256 `708a2009...`, and 26 changed paths. Those are not the immutable CR
identity handed to this reviewer. This is the standard negative-evidence shape
"capability claim that does not reproduce": the claimed scoped publication is
not what the production publication path emitted.

Under the Change Request contract, `accept_cr` would record reviewed trunk OID
`4270549`. Integration against current trunk `0d1641a` would compare the 110 CR
paths with the paths added since `4270549`; the 84 widened paths intersect by
construction, so the candidate returns to `integration_base_moved`/stale. The
revision therefore neither satisfies the 26-path scope nor produces an
integrable fresh-trunk Story candidate.

Positive build, test, race, mutant, cross-platform, and isolated-parity evidence
does not repair this identity failure. I did not rerun the long suites because
the immutable artifact fails before those results can authorize landing.

## Required rework

Provision a fresh Story workspace whose selected base, local base, upstream
base, branch tip, and `HEAD` all equal current `origin/main`
`0d1641a0ab8fe47a98d6a54a81524a37e1cc6ead`. Replay the accepted retention delta
there and publish a new immutable `story_final` Change Request whose:

1. recorded base is `0d1641a` (or a newer exactly verified current trunk if it
   advances before provisioning);
2. changed-path set is exactly the accepted 26-path set;
3. patch round-trip reproduces the advertised candidate tree; and
4. validation and no-live-runtime evidence are bound to that exact tree.

Evidence files from this review are under `.temp/TASK-260830-84z0be/`, notably
`scope-roundtrip.log`, `cr-paths.txt`, `accepted-paths.txt`, and
`widened-paths.txt`.
