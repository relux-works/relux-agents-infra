# TASK-260720-3gcfd1 — response to `CR-TASK-260720-3gcfd1-4` (round five)

Commit `e7a7212`, base `5c84ee7`. 1 file, +126/−26, **`.md` only** — no
repository code, test or config file is touched.

Round four's own verdict: *"It fails on the mirror image of the defect it was
chartered to fix"* — the two adverse findings (N18, N19) were propagated
thoroughly; the two favourable ones (N13, N16) were left standing-as-open at
five sites, three of them named acceptance-criteria deliverables, one a
self-contradiction inside §13's own corrections table. This round fixes those
five sites and runs F5's two documentary halves. **No verdict moves.**

---

## F1 (blocking) — §6 "Options compared" was not swept

Fixed at three points in §6:

1. **B1 verdict row.** Struck *"leaves `sqlite_home`'s default unstated, which
   is a live cross-account leakage risk (Q15)"* with the `~~…~~ — withdrawn`
   convention round four introduced, and stated N13's answer in its place: the
   remaining asymmetry is documented *intent*, not an isolation gap.
2. **B0 Security row.** Widened past N1's three-variable framing to name the
   N15/N20 ambient-input classes (174 claude read sites, eight codex inputs),
   and named `CODEX_API_KEY`'s wrong-identity-under-the-right-label consequence
   explicitly, since that is the worst member of the set the row now covers.
3. **B0 "What it gives up" row.** Updated the Q11 reference to N20a's actual
   standing — partly answered, this host only, open at reduced scope — instead
   of leaving it phrased as a bare open question.

Added `§6` to §13's N13 site list (line ~2494, ten sites now, not nine) and
corrected the "open across nine sites" sentence to say ten and name why: §6
was missed by round four's own sweep and is fixed directly here, not folded
silently into the nine.

## F2 (blocking) — §8 carried a withdrawn finding as adverse

The N4 bullet is struck with the same convention and marked withdrawn per N16:
both fallback legs live in `AutoAuthStorage` and nowhere else, so `bk6owf`
§6.2's `degraded:plaintext` classification gains no support from this finding
— there was no conflict to support it with. Heading changed from "two things,
both adverse" to "one thing adverse, one withdrawn" so the section total is no
longer overcounted.

## F3 (blocking) — §13 self-contradiction on Q12

The `bk6owf` Q12 row (asserting "Still not answered") now ends with
*"Superseded below (N16, round four): the conflict was never real"*, pointing
forward to the row that already carried the correct, current answer. Both rows
are kept — the table's stated purpose is a reconstructable history — but a
reader landing on the stale row is no longer left with an unresolved
contradiction.

## F4 — §11.3's "do not begin" block still promised Q15 could close B1-codex

Rewritten to state what round four actually found: Q15 and Q12 were both run,
both came back favourable, and neither settled anything, because what was open
was never the mechanism, it was entitlement. §12.2's items 1–3 are now stated
as discharged rather than pending, and the paragraph names Q2 as the only
remaining path to closing B1-codex.

## F5 — the reclassification's undocumented free half

Both documentary halves identified in the brief are now run and recorded:

- **Q4 (N21).** `AuthCredentialsStoreMode`'s `#[default]` on `File` is
  unchanged across seven `rust-v*` tags — `0.147.0`, `0.148.0`, `0.149.0`,
  `0.149.1`, `0.150.0`, `0.151.0`, and the unreleased `0.152.0-alpha.6` —
  read via `gh api repos/openai/codex/contents/...?ref=<tag>`. Source-stable
  across that span; does not remove §7.3.3's version-gate cost, since a doc
  comment absent from vendor docs is not a contract for the next release.
- **Q5 (N22).** The claude 2.1.248 bundle contains two independent sites
  computing the config-dir input to the Keychain service-name derivation.
  Both apply exactly `.normalize("NFC")` to `process.env.CLAUDE_CONFIG_DIR`
  (or the homedir-joined default) and nothing else — no `path.resolve`, no
  trailing-slash trim, no symlink dereference. Documentary answer: **no**, it
  does not canonicalize beyond NFC. A trailing-slash or symlinked spelling
  that survives NFC unchanged produces a different Keychain service name.

Both are recorded as partial answers in §4 (Q4/Q5 rows), in §11.2 (W14/W16),
in a formal N21/N22 note pair after N20a, and as ledger rows 43/44. The
harness halves (pin-test against every *installed* codex version; a live shim
run with a symlinked/trailing-slash `CLAUDE_CONFIG_DIR`) are unchanged and
stay in the experiment class, named as such. §2.4.1's twelve/fifteen count is
corrected to say which table state it counts (pre-sweep vs. today).

A **Revision 5** paragraph was added to the document's own revision log (§0
preamble) summarising this round, consistent with how revisions 2–4 are
recorded there.

---

## Boundary held

No credential, token, cookie or Keychain secret value was read, printed,
exported or persisted. No `security` invocation of any kind. No login,
logout, revoke, rotation or re-authentication. No account created, enrolled
or authenticated. Vendor CLI execution: none this round (N21/N22 come from
`gh api` reads of the public `openai/codex` repository and byte-window
extraction from the already-installed, already-analysed claude binary). §11's
implementation breakdown (H1–H14) remains unstarted — this round is `.md`-only,
verified by `git status --short`.

## Validation run

- `go vet ./...` — exit 0.
- `go test ./... -count=1` — exit 0, all packages `ok` (log attached).

Both are sanity checks confirming the docs-only change touched nothing in
`tools/agents-infra`; neither is expected to be sensitive to this delta, and
neither was.
