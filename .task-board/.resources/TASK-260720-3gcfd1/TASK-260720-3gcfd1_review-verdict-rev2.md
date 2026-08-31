# TASK-260720-3gcfd1 — review verdict, round two: CHANGES REQUESTED

Change Request `CR-TASK-260720-3gcfd1-2` rev 2 · reviewed 2026-08-31
Delta `5feebbb..5f0165f`, 5 files, `.md` only. Round-two rework: `7b70fc5..c7d20fa`,
+581/−125 in the ADR, 873 → 1336 lines.

**Verdict: changes requested → `analysis`.** The rework did the thing it was
sent back for, and did it properly: I re-read every cited first-party source
myself, and **the Claude reversal is fully supported by primary evidence quoted
verbatim and correctly**. The codex half is where the problems are. Three
findings, all on that side, all pushing the same direction — they make the codex
NO-GO look better-evidenced than it is. None of them flips a verdict. Two of them
are restatements, one is a ledger fix. No re-research is needed.

---

## 1. What survived attack — quote-checked, not summary-checked

I fetched every cited source independently. `curl`, `gh api`, and one
search-index query phrased differently from both the producer's and the previous
reviewer's.

| Claim | Independent check | Result |
| --- | --- | --- |
| **N8** — Anthropic documents the macOS Keychain keying | `curl https://code.claude.com/docs/en/authentication.md` → 200, grep `CLAUDE_CONFIG_DIR` | **Verbatim, one match, line 178.** *"…keys the macOS Keychain entry to that directory too, so a session with a different `CLAUDE_CONFIG_DIR` reads a different entry."* Exactly as quoted, in the *Credential management* section, macOS-specific |
| **N8** — the plaintext-fallback sentence | same page | **Verbatim.** Keychain-rejects-write → `~/.claude/.credentials.json` at 0600 |
| **N9** — multiple accounts side by side, with a worked alias | `curl https://code.claude.com/docs/en/env-vars.md` → 200 | **Verbatim**, the `CLAUDE_CONFIG_DIR` row, including `alias claude-work='CLAUDE_CONFIG_DIR=~/.claude-work claude'` |
| **N10** — terms restrict sharing, not count | fetch of `anthropic.com/legal/consumer-terms` | **Verbatim**, §2. And I asked the adverse question directly: **no clause limits how many accounts one person may hold** |
| **N12** — codex doc silent on multi-account | `curl https://learn.chatgpt.com/docs/auth` → 200, tag-stripped, grepped | **Reproduced.** The `file`/`keyring`/`auto` text is verbatim. `CODEX_HOME` appears **only** as the location of `auth.json`. Zero hits for `multiple account`, `account switch`, `switching` in substantive text. The silence is established by a *successful read of the page that would carry the statement* — which is the right shape |
| **N12b** — the ten cited issues | `gh api repos/openai/codex/issues/N` on all ten | **All ten exist, all ten open**, titles match the ADR's descriptions exactly |
| Ledger **23** — no merged upstream auth-profile PR | the recorded query, run verbatim | **Zero results. Reproduced** |
| Ledger **24** — 403 is a failed read, not an absence | `curl -A <browser UA>` on both origins | **403 reproduced** on `help.openai.com/en/articles/20001068` and `openai.com/policies/terms-of-use/` |
| Ledger **25** — the switching text reproduces independently | one search query, phrased differently from the two the producer used | **Reproduced a third time**, same substantive text — see F1 for the qualifier |

**Brief item 3 — the `CONDITIONAL GO` condition. Passes, on all three tests.**
Q1 is *"enrol a second Anthropic account under its own state root, leave the first
live, verify both still work after 24 h"*. It is runnable: concrete steps, a
stated duration, a pass/fail. It does **not** require creating or purchasing an
account — §12.1 scopes it to *"an operator who already holds two accounts"* and
Q2c carves out the buy-a-second-subscription case explicitly as a purchasing
decision the ADR declines to make. Enrolment of an already-held account is
irreducible: the question *is* whether concurrent enrolment is durable, and no
restatement removes that. It is stated as an operator experiment, not an agent
action, which keeps it inside the task's boundary. And it cannot be read as a
licence: §9 *"Until Q1 runs, this stays unstarted — a likely outcome is not a
finding"*; §11.3 *"Do not begin H1–H12 before §12 is satisfied. For B1-claude that
means Q1 has actually been run, not that it is expected to pass"*; §12.1 *"It is
not a formality."* Three independent statements of the same bar.

**Brief item 4 — Q2 is genuinely retired.** No conclusion leans on it. §4 marks
it *"largely mooted"* and *"the verdict no longer waits on it"*; §5.2 and §9 both
say the codex verdict does not need it. The only surviving reference is Q12's
*research path* (*"the induced-failure half is blocked on Q2"*) — that is an
unknown's cost column, not a conclusion, and Q12 is itself carried as unresolved
and adverse. Clean.

**Brief item 5 — the LOGBOOK correction replaced, it does not sit beside.** The
new 1140 entry carries **THE REUSABLE ONE**, and it records the *mechanism*, not
the outcome: *"An unperformed read is not an absence, and the tell is the cost
column"* — a question filed as *"a decision, and it is not the agent's to make"*
while eight siblings in the same table were correctly filed as free source
audits. That is the transferable part, stated as the transferable part. The 0842
entry is retitled with a **CORRECTION NOTICE**, its false claim struck inline and
marked `STRUCK`, its verdict line struck and superseded, and its price paragraph
annotated *"Partly superseded at 1140."* The false claim is no longer asserted as
true anywhere; the history is preserved as history. That is the right shape.

**The "Also" list, all three.** Growth is the read and the split, not
elaboration: the hunk map puts +217 in the new §2.3 documentary read, +54 in the
§0 rewrite, +38 in the new §7.3, and the rest in the per-provider rewrites of
§§4, 5, 9, 11.3, 12, 13 and ledger rows 15–26. §11 is unstarted and says so four
times, and the delta touches no code file. §8's plaintext hazard is **untouched**
— one hunk crosses the §8/§9 boundary and it changes only the §9 headline; the
three-way split, the "three halves is the wrong word and the right count" line
and the two-thirds-at-build-or-launch-time budgeting all survive byte-identical,
with the two adverse additions (N4, N1) intact.

**Brief item 2 — the direction that surprised you. It is the sharper reading,
with one real defect.** You expected OpenAI's documented switching to make Codex
permissive; the ADR concludes the opposite. It is *not* the conflation you
feared — the ADR never treats ChatGPT-web switching as Codex CLI entitlement, and
it does not rest the codex verdict on the ChatGPT feature at all. Its
codex-scoped leg is N12: a **successful** read of the actual Codex auth
documentation, which I reproduced, showing `CODEX_HOME` documented purely as a
file location and multi-account nowhere. That is a genuine documentary silence
against Anthropic's documentary *endorsement*, and the asymmetry is real and runs
the way the ADR says. **But** there is a residual conflation, on a different
axis than you guessed, and it is F1.

---

## 2. F1 — BLOCKING. The quote's scope word is dropped everywhere the quote is used

The ADR quotes N11 once, verbatim and correctly, at line 514:

> Account switching is currently available on ChatGPT web; it is **not yet
> supported in Codex desktop** or the native ChatGPT mobile apps.

I reproduced that sentence independently. The word `desktop` then appears
**nowhere else in the document** — I grepped; there are exactly two occurrences
of "desktop" in 1336 lines, this quote and an issue title. Every *use* of the
quote compresses it:

- §0: *"one of its two halves is closed by the vendor's own published position"*
- §5.2: *"Vendor position on multi-account: **Negative.**"*
- §9: *"OpenAI publishes that Codex does not support account switching"*
- §12.2: *"Today it publishes the opposite"*
- §12.3: *"no-go, on the vendor's own published position"*
- `README.md`: *"OpenAI publishes that account switching is not supported in
  Codex, so that half is closed on the vendor's own position"*
- `LOGBOOK.md` 1140: *"NO-GO, now on affirmative vendor evidence: OpenAI
  publishes that account switching is 'not yet supported in Codex'"*

Three things go wrong at once, and they compound:

1. **Surface.** B1-codex decides the **Codex CLI**. The sentence is about the
   Codex **desktop app**. The ADR never states that it is extending the scope,
   and never argues why it may.
2. **Kind.** The sentence reports *feature availability* — a ChatGPT-web session
   switcher has not been extended to other surfaces. The ADR promotes it to a
   *vendor position on multi-account*, which is a different kind of claim. A
   switcher UI is trivially absent from a CLI; that absence is not the vendor
   taking a position on whether an operator may hold two Codex logins.
3. **Provenance, on the surfaces that matter most.** The ADR is scrupulous
   internally — §2.3.1's read-status table, the *(second-hand)* label on N11,
   ledger rows 24–26, W18 as the fix. **`README.md` and `LOGBOOK.md` carry
   neither the qualifier nor the second-hand label.** Those are the two durable
   operator-facing surfaces the previous round was returned for, for a claim of
   exactly this family: a statement about what the vendor evidence *is*.

This is milder than round one — the quote is real, I verified it three ways, and
the conclusion is right. But it is the same class, and it lands on the same two
files. An operator reading the README believes OpenAI has stated a position about
the Codex CLI. It has stated that a web switcher is not in the desktop app.

**Fix, and it makes the ADR stronger rather than weaker:** demote N11 from pillar
to corroboration. Say what it is — *the ChatGPT web switcher has not been
extended to Codex surfaces; second-hand, `help.openai.com` is 403 from here* —
and let the CLI-scoped leg carry the weight, because that leg is a successful
primary read and it is the better argument. Carry the qualifier and the
second-hand label into `README.md` and the `LOGBOOK.md` 1140 entry. The logbook
is the durable one and it is the one that must be right.

---

## 3. F2 — BLOCKING. The rework's own key distinction is applied to Claude and withheld from Codex

§7.3 is the best thing in this revision: it separates the **derivation**
(`sha256(NFC(dir))[:8]`, binary-inspected, no compatibility promise, needs a
per-release pin) from the **property** B1 actually needs (*different config dir ⇒
different entry*, published). That distinction is what correctly dissolves the
version-gate cost for Claude. It is then withheld from Codex, in one sentence:

> **Codex pays §7.2 in full.** One verified version, an undocumented derivation,
> and refusal on the very next upgrade. §7.2's cost argument survives intact for
> the provider whose entitlement evidence is also negative — which is what makes
> the codex no-go **overdetermined rather than marginal**.

Three facts in this same document contradict that, and I did not have to leave
the file to find them:

- §5.2, *Default custody*: **`cli_auth_credentials_store = "file"` is the
  packaged default.** Ledger row 11 records the byte window.
- §5.2, *Namespace mechanism*: the `cli|sha256(home)[:16]` derivation exists
  **only on the keyring branch** — *"on the file branch the credential simply
  lives in the root."*
- Ledger row 20, quoting the page the ADR itself successfully read: *"`file`
  stores credentials in `auth.json` under `CODEX_HOME`."* That is the **property**
  — different `CODEX_HOME` ⇒ different credential file — **vendor-documented**,
  in the same class as N8, on the default store.

So on the default branch, codex has no derivation to pin and the property is
published. §5.2's own B0 verdict already says as much: *"Go, and cleaner than
claude: one input."* The §7.2 cost cannot apply *in full* to a branch that has no
derivation in it.

There is a premise that would rescue the sentence, and the ADR never states it:
codex's `file` store *is* the plaintext `auth.json`, and §8 ends with *"a
`degraded:plaintext` profile refuses to launch"* — so a codex B1 profile would
have to use `keyring`, which does have the derivation, and whose posture is
Q12-unresolved with the docs and the binary in direct conflict (N12a). **That
chain is a genuinely better codex pillar than the desktop-switcher quote: it is
CLI-scoped, first-party, and reproducible.** The ADR assembles every piece of it
and never connects them.

As written, *"three independent reasons"* (§9) and *"overdetermined rather than
marginal"* (§7.3) are the document's own characterisation of its confidence, and
one of the three does not hold in the form stated. An ADR is a thing later
readers rely on; the next person to reopen B1-codex will re-derive the cost
argument, find the default store has no derivation, and be left wondering what
else drifted.

**Fix:** either state the keyring premise explicitly and route the cost through
it, or drop the derivation leg for codex and let the no-go stand on the two legs
that are clean. Adjust *"three independent reasons"* and *"overdetermined"* to
whatever survives. Both routes leave B1-codex a NO-GO.

---

## 4. F3 — SHOULD FIX. Ledger row 21 does not re-run, and this revision knows better

Row 21 records its query verbatim, which is right, and then records a result the
query does not produce.

| | ADR row 21 | Recorded query, re-run |
| --- | ---: | ---: |
| Issues returned | 20 | **27** |
| Open | 19 | **21** |
| Closed | 1 | **6** |

`gh api search/issues 'repo:openai/codex is:issue "account switching" OR "switch
account" in:title'`. The narrower phrasing (`account switching in:title`) gives
16/13/3 — also not 20/19/1. The narrative in §2.3 goes further than the row:
*"The one closed issue is a usage-caching bug"*. There are six closed under the
recorded query — `#14730` (the usage-caching one), `#17349`, `#19756`, `#15384`
duplicates, `#2833` a 403 bug, `#3573` not-planned. §0 repeats *"20
account-switching issues, 19 open"* in the up-front summary.

**The material claim is unaffected and I confirmed it separately:** none of the
six is closed as already-supported, and all ten issues the ADR cites by number
are open with matching titles. This is a ledger defect, not an evidence defect.

It is called out because of what else this revision did: it corrected row 9 for
exactly this — 7,7,7,8,8 that did not reproduce — and wrote the lesson into the
logbook as *"A ledger exists so a later reader can re-run it — a row that does not
re-run is worse than no row."* Row 21 was written in the same commit as that
sentence. Restate the counting method or correct the numbers, and correct
*"the one closed issue"* in §2.3 and the count in §0.

---

## 5. What the producer should do

1. **F1.** Demote N11 to corroboration; carry `desktop` and the second-hand label
   wherever the quote is used, and **into `README.md` and `LOGBOOK.md` 1140**.
   Let N12 — the successful, CLI-scoped, primary read — carry the codex leg.
2. **F2.** State the keyring premise, or drop the derivation leg for codex.
   Reconcile *"three independent reasons"* and *"overdetermined"* with whatever
   survives. If you take the keyring route, say plainly that the default `file`
   store's property **is** vendor-documented and that what closes it is §8's
   plaintext refusal — that is a better argument than the one currently there.
3. **F3.** Fix row 21's numbers or its method; fix *"the one closed issue"* in
   §2.3 and the count in §0.
4. **Leave everything else alone.** §2.3's Anthropic findings, §7.3's
   property-vs-derivation distinction, §8, §11, §12.1's Q1 gate, §13, and both
   `LOGBOOK.md` entries' *structure* all survived attack. Do not rewrite them.

**Not requested:** no change to B1-claude CONDITIONAL GO, B1-codex NO-GO, B0 GO,
A permanent NO-GO, or C GO. Every verdict in this ADR is correct. This is a
correction to how two of the codex no-go's reasons are stated, and to one ledger
row — the same category as round one's finding, one notch milder, and on the same
two durable files.

---

## 6. Safety boundary held by this review

**No account was created, enrolled or authenticated on either provider**, and the
Q1 experiment was **not run**. No credential, token, cookie or Keychain value was
read, printed, exported or persisted. **No `security` invocation of any kind.** No
login, logout, revoke, rotation or re-authentication. Evidence gathering was
limited to unauthenticated `curl`/fetch of public vendor documentation, `gh api`
reads of a public repository, one web search, read-only reads of the worktree,
and `git diff`. No repository file was modified.
