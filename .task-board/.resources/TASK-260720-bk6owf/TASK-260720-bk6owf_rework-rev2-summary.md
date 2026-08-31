# TASK-260720-bk6owf — rework against CR rev1 review verdict

Change Request: `CR-TASK-260720-bk6owf-1` **revision 2**
Base OID: `fba9ee4` · Candidate OID: `c3a467a`
Branch: `task-board/story/STORY-260831-yr0x81`
Artifact: `.research/260831_extensible-auth-method-lifecycle.design.md` (682 → 908 lines)
Date: 2026-08-31

Five findings, all addressed. Scope held to the five: no new sections, no
restructuring, and §9 — which the reviewer independently verified and asked not
to be reworked — is untouched except for the P-AI-1 sentence F1 names.

---

## F2 — "structural" was an overclaim (medium)

The reviewer's charge was precise: §6.1 claimed the enforcement mechanism "does
not exist in the design", while §3.2 said the same pairing is "a config error
refused at load". If it is refused at load, it is expressible, and the claim was
a checked gate wearing the word structural.

**Took the first branch: made it genuinely inexpressible.**

- §3.2 no longer says a method *declares* a custodian class. The class is a
  total function `custodian_class : Method -> class`, evaluated in code. The
  class table is stated to be the *rendering* of that function, not its input.
- Every input surface is enumerated and closed: no registry field, no config
  column, no `--custodian` flag, no environment variable, no remote-config key,
  no plugin hook.
- §3.3's `CredentialHandle` **loses its `custodian_class` field**, with the
  reason stated inline. This was the leak the first branch would otherwise have
  kept: the profile record is JSON on disk, so a stored class is an input
  surface an operator or a migration could edit. Storing a derived value would
  have put the pairing straight back where C1 removes it from.
- §4.3's Custodian column is annotated as rendered, not authored.
- C1 is restated: the pairing is not validated and rejected, it is a state the
  type has no field to hold.

**And said plainly what is still only checked**, because the reviewer's point
was that an honest gate beats a false structural claim:

- The mapping's totality: its fallthrough arm returns *no class established* and
  `describe()` refuses a method that reaches it, so a method added without an
  arm refuses at first use rather than defaulting. That is D1's shape applied to
  custody.
- The exhaustiveness test, with the negative that matters named: it must fail
  when an arm is **changed** to another class, not only when the mapping is
  deleted. A delete-only mutant proves the mapping exists and says nothing about
  the class it covers.

§6.1 is retitled **"What is eliminated, and what is only checked"** and splits
the hazard into its two halves, which have different residual risk:

- *Eliminated* — the declaration half, by C1. No check to bypass, no entry point
  that can forget to call one.
- *Mitigated, not eliminated* — the implementation half, by §6.4's module-wide
  greppable-absent assertion plus negative tests. Stated as what it is: a
  build-time gate that protects only what its pattern set names, and a new way
  to spell `chmod` added without extending that set would pass.

It also records the dependency the reviewer found: §5.3's prohibition table is
scoped "For `vendor-opaque`", and that scoping is safe **only because of C1** —
remove C1 and a native method could be evaluated on the `host-owned` branch
where the prohibitions do not apply. Stated so nobody relaxes one of the two
without seeing the other.

§0's matching sentence ("closes it structurally") is corrected to say the
document names which half is eliminated and which is held by a gate, rather than
averaging the two into one reassuring word.

## F3 — `unknown` leaked in the one destructive verb (medium, destructive)

§5.4's `local` vocabulary was `invalidated | already-absent | failed`. Nothing
stopped a failed or denied observation producing `already-absent`, which is a
*positive assertion of absence*, and `remove` gated destruction on it.

- `local` gains **`unknown`**.
- A **classification table** now states what produces each value. Both
  `invalidated` and `already-absent` require a *successful* metadata-only
  observation at `observed_coords`; the vendor's exit status classifies nothing
  on its own. A failed, denied or malformed observation on either side is
  `unknown`.
- Stated explicitly: **silence is `unknown`, not `already-absent`**. A vendor
  logout that exits 0 and prints nothing is indistinguishable between a locked
  login keychain, a swallowed permission error, and a genuine no-op.
- `remove --logout-policy local-logout` deletes only on `invalidated |
  already-absent`. `unknown` and `failed` both **refuse**:
  `E_AUTH_LOGOUT_UNCONFIRMED`, non-zero exit, nothing deleted, profile
  unchanged. The text names why: `unknown` is precisely the state in which
  deleting the derivation input produces an orphan nothing can name again.
- The **tombstone** is now written by every `remove` that did not positively
  confirm local invalidation — `leave-vendor-state` always, and `local-logout`
  resolving to `already-absent`. Only `invalidated` deletes without one.
  `unknown` and `failed` produce no tombstone because they produce no deletion.
- The tombstone's contents are made store-aware (keychain coordinates for a
  keychain/keyring store; state root plus relative path for a file store), which
  F4 needed anyway.
- §4.1's `logout` row and §10's D1 row are updated to match.

## F4 — the codex branch assumed the wrong default (medium)

§2 fact 4 states codex's packaged default is `cli_auth_credentials_store =
"file"`, and the lifecycle then specified only the keyring path.

- **§3.3** gains `store_selector` as a first-class field of the handle, and a
  table mapping runtime × store → which coordinates are populated. Under `file`
  the keychain fields are **empty**; under `keyring` `file_path` is empty. The
  text states that the value lives in a config file *inside* the state root, so
  it can change after enrol without the path changing.
- **H3** takes `store_selector` as an input to the re-derivation. Without it,
  re-deriving compares a stale shape against itself and reports a false match —
  the vacuous-check shape. A changed store refuses as `namespace-drift` naming
  the transition; an unreadable store value is `unknown` and refuses, never
  assumed to be the default.
- **§5.1 step 6** states who sets it: the operator, through codex's own config.
  agents-infra never writes it and composes no `-c` override, so a fresh root
  resolves to the packaged `file`. **`auto` is refused**
  (`E_AUTH_CUSTODY_AMBIGUOUS`) — it names no single custody and its documented
  failure path is a fallback to plaintext `auth.json`, the exact fail-open shape
  §6 exists to exclude.
- **§5.3 (refresh)** states that a `file`-store credential sitting in a root
  agents-infra created is still `vendor-opaque`: custody follows the method, not
  the filesystem, so every prohibition applies byte-for-byte to `auth.json`.
  Whether codex rewrites it on every refresh is stated as not established, and
  the design depends on no answer because it neither reads nor writes the file.
- **§5.4 (retire)** gains the file-store branch: the orphan hazard does not
  arise because the credential is inside the root, but `leave-vendor-state`
  destroys the local credential while leaving a *server-side* session nothing
  attempted to end, since the vendor's own logout is what performs openai's
  best-effort revoke. The tombstone records that.
- **§6.2** is re-keyed on the **recorded store**, with the principle stated:
  `degraded:plaintext` means *unexplained* plaintext, not plaintext. codex-on-
  `file` is **`active`** — the vendor's documented default custody. codex-on-
  `keyring` with an `auth.json` present is `degraded:plaintext`, because the
  research established that `auto` falls open and left `keyring`'s posture
  unknown, so the design refuses rather than guessing in either direction. Store
  selector unreadable is `unknown`.
- **§7.1** states the codex range applies to both branches: the `file` branch
  has no hash to break, but the packaged default that *selects* it is itself a
  version-fixed constant, so a build outside the range can silently change which
  branch is in use.
- **§7.2** gains the file-store pin branch, with the negatives that give it
  meaning: running the keyring assertions against a `file`-store profile must
  **fail** (a pin that passes on both branches selects neither); no `Codex Auth`
  item may appear for a `file`-store synthetic home; an `auth.json` under a
  different home must not satisfy this profile's pin; and store drift must fail
  the pin rather than silently re-derive.
- **Q12** added: codex's `keyring` failure posture, what settles it, blocked on
  Q2's disposable account. This is a new `unknown` the fix surfaced, and §6.2
  now refuses on it rather than assuming either answer.

## F1 — §12 row 1 stated a false enumeration (low)

Re-derived in this worktree. `grep "cmd.Env"` is case-sensitive and matches
neither `command.Env` nor `piCmd.Env`/`runtimeCmd.Env`; it found two of the
module's **seven** child-environment assignments.

Row 1 now: read `main.go:417-472` directly (neither launcher assigns `cmd.Env`),
enumerate all seven env-composing sites by name, note that all seven are pi
launches where `HomeEnvVar` is `""`, and state that the narrow grep reported as
a module-wide enumeration was wrong. §9.3's P-AI-1 cell carried the same false
sentence and is corrected the same way. The conclusion and P-AI-1's scope are
unchanged — the reviewer's assessment that this is a reporting defect, not a
design defect, is correct. Raw command output is in the CR rev2 validation log.

## F5 — §3.6 omitted a transition §5.1 produces (minor)

The diagram gains the `enrolling -> enrolling` edge labelled "observation
failed", and the prose states the distinction the two readings collapsed:
`enrolling` with an `unknown` observation *result* is not the `unknown` *state*.
`enrolling` means the enrolment never completed; `unknown` means an enrolled
profile could not be observed; they have different ways forward (re-run the
vendor login versus repair the observation). Both refuse to launch, so the
distinction is diagnostic, not a safety boundary. The launch-refusal bullet is
widened from "every `degraded:*` and `unknown`" to every state but `active`.

---

## What this rework does not claim

- **No gate in this document is executable today.** §9 establishes that three
  named repositories lack the contracts, so the negative tests specified in
  §3.2, §5.4, §6.4 and §7.2 are prerequisites for `TASK-260720-3gcfd1`, not
  things that ran here. `go test ./...` in `tools/agents-infra` exits 0, which
  establishes that nothing regressed — not that this change is covered.
- **The attack that happened was on the specification, not on running code.**
  Three of the five findings are gate defects of exactly the shapes the evidence
  standard names: a positive assertion of absence produced by a read failure
  (F3), a check whose claimed structure was a validation that a future entry
  point could miss (F2), and a pin that would pass vacuously against coordinates
  the default configuration does not populate (F4). Each is fixed in the
  specification and each now carries the narrowing negative that would catch it.
- **Commit `c3a467a` is unsigned.** This worktree has no `user.signingkey` or
  `gpg.format` configured, and every commit on this branch and its base is
  unsigned (`git log --format=%G?` reports `N` throughout). Signing would
  require choosing an identity on the operator's behalf, which a spawned run
  should not do. Flagged rather than worked around.

## Safety boundary

Unchanged and held. No credential, token, cookie or Keychain secret was read,
printed, exported or persisted. No Keychain item was queried, created, modified
or deleted. No `security` invocation of any kind was made. No login, logout,
revoke, rotation or re-authentication ran. Every command in this rework was a
source read, a grep, a Go build/test/vet, or a git operation on this worktree.
