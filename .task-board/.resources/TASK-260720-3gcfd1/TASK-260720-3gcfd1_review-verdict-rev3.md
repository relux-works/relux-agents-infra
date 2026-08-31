# TASK-260720-3gcfd1 — review verdict, round three: CHANGES REQUESTED

Change Request `CR-TASK-260720-3gcfd1-3` rev 3 · reviewed 2026-08-31
Base `5feebbb` → candidate tree `4fcae1e` (verified identical to `f9a6c5b^{tree}`).
Round-three rework: `c7d20fa..f9a6c5b`, +599/−147, ADR 1336 → 1754 lines,
`.md` only — no repository code file is touched in the round or in the whole CR.

**Verdict: changes requested → `analysis`.** One blocking finding, one
should-fix, one minor. **The landing is honest and the symmetry holds** — every
one of the brief's five questions passes, and I attacked them rather than read
them. The blocking finding is not about what round three concluded. It is that
the ADR's own **highest-priority free source audit**, the one it says could still
close B1-codex, is answerable from first-party public source in three read-only
calls; the answer is favourable; and it surfaces a **second codex ambient
environment input that B0 — the option this ADR says adopt now — has no
prerequisite for**, falsifying the ADR's stated completeness claim about B0's
dependencies. No verdict in the document moves as a result. One "adopt now"
work breakdown does.

---

## 1. The brief's five questions, attacked

### 1.1 Is `HELD — UNESTABLISHED` the honest landing, or a retreat? **Honest.**

I re-read the material myself rather than accepting the ADR's reading, and I
looked specifically for a conclusion the evidence *does* support for the CLI
surface, because declining to conclude would be its own error.

Every CLI-scoped leg reproduced exactly, from the sources the ADR names:

| Claim | My independent check | Result |
| --- | --- | --- |
| `developers.openai.com/codex/auth.md` is a real markdown source endpoint | `curl -sSL`, 15 440 bytes | **HTTP 200.** Round three's "better source" is real |
| It is surface-scoped by `ContentModeSwitch` | grep | **Reproduced.** `group="codex-surface"` with `ids="app,cli,ide"`, `id="web"`, `id="app"`, `id="cli"`, `id="ide"` — a CLI claim can be read off the vendor's own scoping, exactly as N12 says |
| `CODEX_HOME` appears **exactly once** | `grep -o … \| wc -l` | **1.** In `- \`file\` stores credentials in \`auth.json\` under \`CODEX_HOME\`…`, as a file location |
| `multiple account` / `account switch` / `switching`: **zero** | case-insensitive grep | **0 each.** Documentary silence established by a *successful* read of the page that would carry the statement |
| The *Login caching* quote, `app,cli,ide`-scoped | line 233 | **Verbatim**, every word incl. "The CLI and extension share the same cached login details" |
| "Codex caches login details locally in a **plaintext file**" | line 235 | **Verbatim.** The vendor does call its own default store plaintext |
| The credential-storage block quote | lines 255–257 | **Verbatim**, including the `auto` sentence, which round three corrected from round two's paraphrase |
| The only worked example sets `cli_auth_credentials_store = "keyring"` | lines 249–252 | **Reproduced** — and it is not the shipped default (ledger 11), so ledger 30 holds |
| `config-reference.md`: `cli_auth_credentials_store` has **no default** | JSON key extraction | **Reproduced.** `{key, type: "file \| keyring \| auto", description}` — no default field, no default in prose |
| N12b — 27 issues, 21 open, 6 closed | the recorded method verbatim, `-f per_page=100`, count by `state` | **27 / 21 / 6. Reproduces exactly.** Row 21 is fixed |
| The six closed, with `state_reason` | `gh api` | **Exact match:** `#14730` completed, `#2833` completed, `#17349`/`#19756`/`#15384` duplicate, `#3573` not_planned. **None closed as already-supported** |
| All ten cited issues open, titles matching | ten direct `gh api` reads | **All ten open, all titles match** |
| N12c — Discussion `#25630` has no vendor reply | `gh api graphql`, `answer` + `authorAssociation` | **`answer: null`; 6 comments, all `NONE`.** I went further than the ledger and read **nested replies too** — the one reply is also `NONE`. No maintainer, member or collaborator anywhere in the thread |
| N11 is still second-hand | `curl` with a browser UA | **403 reproduced** on both `help.openai.com/en/articles/20001068-…` and `openai.com/policies/terms-of-use/`. The label is correct and must stay |

**So: does the published material support a conclusion for the CLI surface?**
It supports several, and the ADR states all of them: the capability is unshipped
across Codex surfaces, the CLI-scoped documentation is silent on multi-account,
the documented login model is a *single* cached login per state root, and the
vendor has not answered its own thread on the question. What it does **not**
support is a statement about entitlement — whether one operator may hold two
Codex CLI logins. Feature availability is not permission; documentary silence is
not prohibition; an open feature request is not a vendor saying no. `UNESTABLISHED`
is the only status those four legs carry.

And this is not the comfortable landing. The comfortable landing was available
and was explicitly recommended to the producer: round two's review closed F2 with
*"Both routes leave B1-codex a NO-GO"*. Round three refused it, checked the
premise that review supplied, found the design document contradicts it, and moved
the verdict against the reviewer's stated expectation. **That is the opposite of
retreating to safe wording.** §7.3.2 records the failed reviewer premise rather
than quietly dropping it, which is the right call — a correction carries the same
standard as a claim.

### 1.2 Scope restoration, meaningfully. **Yes — the conclusions moved with it.**

`desktop` is now 23 in the ADR, 1 in `README.md`, 4 in `LOGBOOK.md`. I checked
every site, not the count. It is not a word inserted beside unchanged sentences:
at each of round two's seven compressed uses the **claim itself changed**.

| Site | Round two | Round three |
| --- | --- | --- |
| §0 | "closed by the vendor's own published position" | "OpenAI publishes nothing that addresses the Codex CLI" |
| §5.2 vendor-position row | "**Negative.**" | "**Unestablished**", row retitled *"…, CLI surface"* |
| §5.2 B1 verdict | "No-go — on affirmative vendor evidence" | "HELD — UNESTABLISHED" |
| §6 entitlement row | "a published negative" | "unestablished for the CLI surface" |
| §9 | "Reject B1-codex on published vendor evidence" | "Hold … it is not rejected" |
| §12.2 | "Every item is something **the vendor** would have to do" | inverted: three of five are ours and free |
| §12.3 | "no-go, on the vendor's own published position" | "held, unestablished — not a no-go" |
| `README.md` | "that half is closed on the vendor's own position" | "**The Codex half is not closed — an earlier revision of this README said it was, and that was wrong**" |
| `LOGBOOK.md` 1140 | "NO-GO, now on affirmative vendor evidence" | struck inline, superseded by 1210 |

I grepped adversarially for any *surviving* compressed use — `not supported in
Codex`, `vendor's published position`, `Vendor position … Negative`, `a published
negative` — across all three files. **Every hit is inside a correction or a
quotation of the withdrawn claim.** Nothing still asserts it.

### 1.3 F2 symmetry. **Applied, and it changed the finding.**

§7.3.1 puts claude, codex-`file` and codex-`keyring` in one table on four axes and
lets the result fall where it falls. It falls against round two. I verified the
load-bearing premise at its source rather than trusting the quotation:

`bk6owf` §6.2, `.research/260831_extensible-auth-method-lifecycle.design.md:668,676-678,689` — **verbatim, and the ADR's abbreviation is faithful**:

> **`degraded:plaintext` means *unexplained* plaintext, not plaintext.** It is a
> claim that a store failed open, not a claim about file custody as such.

> | codex `file` | `<state_root>/auth.json` present at 0600 | **`active`** — this is the vendor's packaged default custody, not a fallback |

> Calling codex-on-`file` `degraded:plaintext` would refuse every default codex
> enrolment for a hazard that did not occur — the credential is in a 0600 file
> *by design*.

So the rescue premise round two's review supplied is genuinely contradicted by the
design it cited, §8 does not force codex onto `keyring`, and §7.3.1's retirement
of the derivation cost for the default branch is correct. The replacement cost
found in its place (§7.3.3, the store *selector*'s undocumented default) is real
— I confirmed independently that `config-reference.md` states no default for
`cli_auth_credentials_store` and that `auth.md`'s only example shows `"keyring"`,
which is not what ships. And it is correctly reported as **escapable** by H13,
the same move §7.3 makes for claude. Symmetric in both directions.

`"overdetermined"` is withdrawn outright; `"three independent reasons"` is
re-derived in §9 rather than left standing. Both are the honest accounting.

### 1.4 The logbook. **Third, correct, replaced not accumulated, and it records the mechanism.**

The 1210 entry supersedes rather than sits beside: 1140's codex claim carries a
`CORRECTION NOTICE (1210)`, its verdict line is struck with `~~…~~` and marked
`STRUCK at 1210`, its "THE THIRD LESSON" bullet has *"negative about the surface
that matters"* struck and replaced inline with *"silent about the surface that
matters"*, and the 0842 price paragraph is annotated *"Partly superseded at 1140,
and further at 1210"* with the codex half struck. The false claim is asserted as
true nowhere; the history survives as history.

It records the **transferable mechanism**, not the corrected conclusion:

> A claim stated precisely once and used in a widened form thereafter reads as
> evidence-backed, and **the tell is that the precise form stays in the evidence
> section while the widened form goes into the summary, the README and the
> logbook**. … **When you cite a quote more than once, grep your own document for
> the quote's narrowest word; if it appears once, every other site is a paraphrase
> you have not checked.**

That is a runnable check, not a moral. The second and third bullets are equally
mechanism-shaped (apply your best distinction to the case it hurts *before*
publishing the case it helps; a reviewer-supplied premise arrives with authority
and no citation check — run one, especially when it lets you keep your
conclusion). Correct shape.

### 1.5 Growth 1336 → 1754. **Restoration and symmetry, not accumulation.**

I attributed every `-U0` hunk to its enclosing section in the pre-image:

| Net | Section |
| ---: | --- |
| **+115** | §7.3 — the new §7.3.1/7.3.2/7.3.3 symmetry work |
| **+83** | §2.3.1 / N11 / N12 / N12c — the scope analysis, the better primary source, the new read |
| +35 / +34 / +33 / +32 | §12.3, §12.2, §0, §9 — conclusions re-derived from the above |
| +18 | preamble |
| ≤+17 each | §11.2, §11.3, §4, §14, §10, §13, §12 |
| **0** | §5.2, §6 B1, §15 — pure substitutions, no growth |

The two largest blocks are exactly the two things the brief sent it back for.
`§8`, `§2.1` and `§7.0–7.2` are **byte-identical** to round two — I diffed them
whole, not by grep — so the plaintext three-way split, the 3moaky-item-4
refutation and the `CLAUDE_CONFIG_DIR` pricing are carried forward untouched as
required. The one place it accumulates is **§12.4**, a ~20-line three-rounds
retrospective that overlaps §13's correction rows and the logbook's 1210 entry.
Not a defect; worth trimming if anything else in this file is rewritten.

---

## 2. F1 — BLOCKING. The free read the ADR ranks first is three public-source calls away, and running it falsifies a completeness claim about the option being adopted now

This is not a finding about the codex landing, which is right. It is the
round-one shape one notch milder again, on a third axis — and unlike round one it
does not misreport an absence, so the defect is in what the document leaves
**unevaluated and tells operators is the substantive risk**, plus what that
unrun read conceals in **B0**.

### 2.1 What the ADR says

- §11.2: *"**W10 is now the highest-value item on this table, and round three
  promoted it from housekeeping to the front.** … the only free read in this ADR
  that could still **close** B1-codex on a demonstrated ground."*
- §12.2 item 1: *"This is the substantive risk and it should be first. … A source
  audit; **free**; W10."*
- §12.3: *"one genuinely open isolation question (`sqlite_home`, Q15) that is a
  **free source audit nobody has run**."*
- `README.md`, operator-facing: *"and — **the substantive one** — `sqlite_home`
  has no documented default, so nobody has yet checked whether two
  credential-isolated Codex profiles share a state database."*
- §15 AC6: *"**AC6 for codex is *unevaluated*, and the free source audit that
  would evaluate it is §12.2 item 1.**"*

### 2.2 I ran it. Three read-only calls against public first-party source.

```
gh api -X GET search/code -f q='repo:openai/codex sqlite_home' -f per_page=20
gh api repos/openai/codex/contents/codex-rs/config/src/config_toml.rs --jq .content | base64 -d
gh api repos/openai/codex/contents/codex-rs/state/src/lib.rs        --jq .content | base64 -d
gh api repos/openai/codex/contents/codex-rs/core/src/config/mod.rs  --jq .content | base64 -d
```

`codex-rs/config/src/config_toml.rs:335-337` — **the vendor's own doc comment**:

> ```rust
> /// Directory where Codex stores the SQLite state DB.
> /// Defaults to `$CODEX_SQLITE_HOME` when set. Otherwise uses `$CODEX_HOME`.
> pub sqlite_home: Option<AbsolutePathBuf>,
> ```

`codex-rs/core/src/config/mod.rs:3951-3956`, the **production** resolution path,
not a helper:

> ```rust
> let sqlite_home = cfg.sqlite_home.as_ref().cloned()
>     .or(sqlite_home_env)
>     .unwrap_or_else(|| codex_home.clone());
> ```

with `sqlite_home_env` from `resolve_sqlite_home_env` (`mod.rs:247-256`), which
reads `codex_state::SQLITE_HOME_ENV` from the process environment, and

`codex-rs/state/src/lib.rs:108-109`:

> ```rust
> /// Environment variable for overriding the SQLite state database home directory.
> pub const SQLITE_HOME_ENV: &str = "CODEX_SQLITE_HOME";
> ```

`log_dir` resolves at `mod.rs:3940-3943` to `codex_home.join("log")`, which
independently confirms ledger row 29's documented half from source.

**Answer: `sqlite_home` defaults to `$CODEX_HOME`.** Precedence is
`config.toml` › `CODEX_SQLITE_HOME` › `CODEX_HOME`.

### 2.3 What follows, and why it blocks

**(a) The ADR's largest stated objection to B1-codex dissolves.** §12.2 item 1
states the fork itself: *"If it defaults inside, the largest open objection goes
away."* It defaults inside. Two credential-isolated codex profiles do **not**
share a state DB by default. The verdict does not move — entitlement is still
unestablished and Q2 is still unrun, so B1-codex stays HELD — but §0, §9, §12.2,
§12.3, §5.2's *Other state* row (still `Unknown`), §11.2's promotion of W10, §10's
UX reasoning (*"a UX … specified before anyone has checked whether those profiles
share a state database"*) and the README paragraph all currently describe an open
risk that is closed, favourably, for free.

**(b) AC6 for codex is recorded `unevaluated` when the evaluation was in hand.**
That is an epic acceptance criterion, and the DoD asks this task for a
per-provider feasibility verdict. Leaving it unevaluated is defensible only while
the evaluation is expensive; the ADR itself says it is free, ranks it first, and
does not run it.

**(c) The part that actually blocks: `CODEX_SQLITE_HOME` is a second codex
ambient environment input, and B0's prerequisite list has no entry for it.**
It appears **nowhere** in the ADR, `bk6owf`, the keychain-custody research,
`README.md` or `LOGBOOK.md` — I grepped all five.

- §5.2: *"Namespace inputs: **One** (N5)."* §5.2's B0 verdict: *"**Go**, and
  cleaner than claude: one input."* N5 derives that from `Codex Auth` occurring
  exactly once with no environment-derived affix — which is precise about the
  **credential** namespace and true. It is then used at the wider scope of
  **state isolation and B0's prerequisites**, where it does not hold.
- Claude's three extra namespace inputs (N1) get a presence refusal *and* a
  dedicated prerequisite: **W13**, *"Survey whether any legitimate operator setup
  sets `CLAUDE_SECURESTORAGE_CONFIG_DIR` or `CLAUDE_CODE_CUSTOM_OAUTH_URL`"*,
  marked *"and it is a **B0 prerequisite**"*. Codex has no counterpart, because
  the document believes codex has one input.
- §11.2 then states completeness outright: **"W13 is the only one B0 depends
  on."** With an ambient `CODEX_SQLITE_HOME` set, a B0-composed codex launch with
  a per-profile `CODEX_HOME` puts the state DB **outside** that root — the exact
  cross-account state sharing AC6 forbids, on the option this ADR says **adopt
  now** and calls *"ready to schedule"*. That sentence is false in a checkable
  way.

**This is the round-three defect in the round-three shape:** a claim precise where
it is derived (`Codex Auth` occurs once ⇒ one *credential* namespace input) and
wider where it is used (⇒ codex needs no ambient-input survey for B0). The
document diagnosed that pattern three times and committed it on the axis it did
not grep.

### 2.4 What to change — no re-research needed beyond confirming the three reads

1. **Q15: answered, not half-answered.** Record both halves from source with the
   file/line citations above as new ledger rows. Note the precedence chain and
   that `config.toml`'s `sqlite_home` **beats** the env var — the same per-profile
   escape hatch as H13, and worth saying so, because it is the third instance of
   the same move.
2. **§5.2 *Other state* row** — `Unknown` → answered, with the ambient-override
   hazard named.
3. **New B0 prerequisite W-item, the codex twin of W13**: survey whether any
   legitimate operator setup sets `CODEX_SQLITE_HOME`; have B0's explicit child
   environment pin or clear it, and have `--print-config` render it in the
   composed namespace-variable set (§10.2 / W2). **Correct "W13 is the only one
   B0 depends on."**
4. **W10 → complete**; drop its promotion in §11.2 and its first place in §12.2,
   and re-rank what is left (Q12/W19, H13, Q2).
5. **§0, §9, §12.2, §12.3, §10, AC6, `README.md`** — remove "the substantive one"
   / "the largest open objection" framing and state what actually remains: no
   vendor documentation of the use case, entitlement unestablished, Q2 unrun,
   and one ambient env input to pin. **B1-codex stays HELD.**
6. **`LOGBOOK.md`** — the 1210 entry's *"WHAT ACTUALLY MIGHT CLOSE IT, and nobody
   has run it"* bullet and its *"three rounds argued about a vendor sentence while
   the one free read that could decide the question went unrun"* line need the
   fourth-round version: the read was run, it did not close it, and it found a
   second ambient input. The transferable lesson is stronger, not weaker — *a
   free read you rank first and do not run is still a free read you did not run,
   and correctly labelling it "unknown" does not discharge it.*

**Not requested:** no change to any verdict. A stays permanent NO-GO, B0 GO,
B1-claude CONDITIONAL GO, **B1-codex HELD — UNESTABLISHED**, C GO, qwen
not-modellable. §7.3.1–7.3.3, §2.3's N11 demotion and N12/N12b/N12c, §8, §2.1,
§7.0–7.2, §12.1's Q1 gate, §13 and both logbook entries' structure all survived
attack and should not be rewritten.

---

## 3. F2 — SHOULD FIX. §12.2 item 6's surface enumeration is narrower than its own sources

> "these are **third-party reports** about the vendor's **own switching
> implementations on the app and IDE surfaces**, not about running two
> `CODEX_HOME` roots"

`#22419`'s body says: *"After switching accounts in Codex App **/ Codex CLI**,
local Codex sessions work, but SSH remote sessions…"*. It names the CLI. The
others check out — `#39698` and `#26628` are the Codex app, `#16894` is the VS
Code extension.

The load-bearing half of the sentence survives: `#22419`'s reporter used the
app's account switcher, not two `CODEX_HOME` roots, so it is still not evidence
about B1's mechanism. But the enumeration is a scope word narrowed in the
direction that makes adverse evidence look less relevant — the mirror of the
defect this round corrected. Say "at least one (`#22419`) names the CLI, and it
is still a report about the vendor's switcher rather than about two state roots",
or drop the surface list.

---

## 4. F3 — MINOR. The logbook asserts a cause for row 21 that does not reproduce

`LOGBOOK.md` 1210: *"The cause was method, not phrasing: `gh api search/issues`
**pages at 30 by default** and the row recorded no page size or counting step."*

The query returns **27** items. I ran it both ways: with `-f per_page=100` and
with the default page size. **Both return 27 / 21 open / 6 closed.** Twenty-seven
fits inside a thirty-item page, so default paging cannot have truncated round
two's result to 20/19/1. Round two's reviewer's alternative — the narrower
phrasing `account switching in:title` — gives 16/13/3, also not 20/19/1. Neither
candidate cause reproduces; the honest status is **cause unestablished**.

The ADR's ledger row 21 is better: it hedges with *"a count taken off a truncated
**or differently-phrased** run does not reproduce"*, and its recorded method does
re-run. The logbook does not hedge, and it states the unverified cause one bullet
after *"a correction carries the same standard as a claim"*. State the corrected
numbers and the method that reproduces; say the cause of the original miscount is
unknown.

---

## 5. Definition of Done — where it stands

| DoD item | Status |
| --- | --- |
| ADR records evidence, assumptions, unknowns; 3moaky item 4 marked refuted, not the audit as a whole | **Met.** §2.1 byte-identical to the version already accepted on this point |
| Per-provider feasibility verdict, both custody models, Claude Keychain namespacing and Codex file store | **Met for A/B0/B1-claude/C.** For B1-codex the verdict is sound but rests on one input recorded `unevaluated` that is free — F1 |
| Options compared on security, concurrency, refresh, revocation, and what each gives up | **Met.** §6, both directions, including for accepted options |
| Recommended architecture or explicit no-go, no-go a legitimate outcome | **Met**, and round three demonstrates the converse discipline: it *withdrew* a no-go once its reasons failed rather than defending it |
| `CLAUDE_CONFIG_DIR` dependency priced explicitly | **Met.** §7.0–7.2 byte-identical; §7.3 correctly removes it for the property B1 needs |
| Plaintext-fallback hazard carried forward with its three-way split | **Met.** §8 byte-identical |
| Proof-of-concept gates and CLI UX for viable paths only | **Met**, with §10's B1-codex reasoning to be restated once F1 lands |
| Implementation work breakdown recorded and left unstarted | **Not met.** Unstarted: yes. Recorded completely: no — B0 is missing the `CODEX_SQLITE_HOME` prerequisite and asserts "W13 is the only one B0 depends on" (F1c) |
| No credential printed/exported/persisted; no session logged out, revoked or rotated | **Met**, by the producer and by this review |
| New outcome artifact attached with a task-scoped name | **Met** — this file |
| Gates attacked, not read | **Met by the producer** (§7.3.2 attacks a reviewer-supplied premise and defeats it) **and by this review** |

---

## 6. Why this is `analysis` and not an accept

Three rounds have each found a real defect and each moved the conclusion closer to
what the evidence supports, and this round's conclusion is where I independently
land too. I considered accepting with F1 as a note. I am not, for one reason:
**F1c is a false completeness claim about the option the ADR says to schedule
now.** "W13 is the only one B0 depends on" will be read as the prerequisite list,
and B0-codex has a second ambient input that defeats per-profile state isolation.
That is a decision the epic acts on, not a documentation nicety — and it was
reachable by the read this document ranks first and left unrun.

The fix is bounded: three source reads already done and cited above, one new
ledger block, one new W-item, one corrected sentence, and a re-ranking of §12.2.
**No verdict changes and no re-research is required.** Everything in §§1.1–1.5
survived attack and must not be rewritten.

---

## 7. Safety boundary held by this review

**No account was created, enrolled or authenticated on either provider**, and the
Q1 experiment was **not run**. No credential, token, cookie or Keychain value was
read, printed, exported or persisted. **No `security` invocation of any kind.**
No login, logout, revoke, rotation or re-authentication. No `codex` or `claude`
subcommand that touches state was executed — vendor CLI execution was limited to
`codex --version`. Evidence gathering was unauthenticated `curl` of public vendor
documentation, `gh api` reads of a public repository, read-only `strings`/`grep`
of an installed binary, and read-only reads of the worktree and `git diff`.
**No repository file was modified by this review.**
