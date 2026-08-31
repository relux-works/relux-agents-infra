# TASK-260720-3gcfd1 — response to CR-TASK-260720-3gcfd1-2 (round three)

Commit `f9a6c5b`. Delta vs `main`: 3 files, `.md` only, +599/−147.
`.research/260831_multi-account-auth-architecture.adr.md` 1336 → 1780 lines.

**Headline, stated first because it exceeds what was requested: F1 and F2 both
moved, and the codex verdict moved with them. B1-codex is no longer a NO-GO.**
It is **HELD — UNESTABLISHED**. Not because new permission was found — because
all three reasons round two gave failed inspection, *including the premise this
review supplied to rescue one of them*, and nothing replaced them. Both halves
of B1 remain unstarted. No other verdict moves.

---

## F1 — BLOCKING. The scope word survives once and is dropped everywhere it is used

**Accepted in full, and it went further than restoring the word.**

`desktop` now appears at every site that *uses* the quote, and the quote is used
differently. Specifically:

- **§2.3 / N11 rewritten.** The heading no longer claims OpenAI "publishes that
  Codex does not have it". The finding now states the scope analysis explicitly
  in three numbered parts — **surface** (two GUI apps enumerated; the CLI is not
  among them, and `openai/codex#31778` *"…in the **Codex desktop app**"* confirms
  "Codex desktop" is a distinct named product, not loose phrasing), **kind**
  (feature availability of a session-switcher UI, not an entitlement position —
  a switcher UI is trivially absent from a command-line tool), and **tense**
  (*"not yet"* presupposes intent to ship; it is the opposite shape from a
  prohibition).
- **N11 is demoted to corroboration.** Nothing in the verdict rests on it. The
  direction of support in N12b's standing paragraph is also reversed: N11
  corroborates the reachable first-party tracker evidence, not the other way
  round.
- **The renamed verdict.** Asking the question again gave the answer the brief
  anticipated: the CLI surface is unaddressed by that sentence, so the codex leg
  is **not** "closed on the vendor's published position". Entitlement for the
  Codex CLI is **UNESTABLISHED** — a third status, explicitly distinguished
  throughout from round one's "no evidence" (an unperformed read) and round two's
  "closed by published position" (a widened quote).
- **A better primary source was found in the process**, and it should have been
  found in round two. OpenAI publishes **markdown source endpoints** at
  `developers.openai.com/codex/*.md` (HTTP 200) — the same documents without a
  tag-stripping step, and explicitly **surface-scoped**: the body is divided into
  `ContentModeSwitch` blocks tagged `app`/`cli`/`ide`/`web`, so a CLI-scoped claim
  can be read off the vendor's own scoping instead of inferred. `CODEX_HOME`
  occurs **exactly once** in the auth doc, as a file location; `multiple account`
  / `account switch` / `switching` occur **zero** times. Ledger row 19 re-run.
- **New N12c.** OpenAI's own discussion thread on exactly this question
  (`openai/codex#25630`, *Switch Between Accounts*) has **no accepted answer and
  every comment `authorAssociation: NONE`** — no maintainer, member or
  collaborator has replied. A successful read of a place a vendor statement would
  appear, returning none. Ledger row 28.
- **`README.md` and `LOGBOOK.md` 1140 corrected**, per §5 of the verdict — see the
  "durable surfaces" section below.
- **§12.4 added**: a four-row table recording all three rounds of the one
  recurring shape, so the pattern is inherited rather than re-derived.

## F2 — BLOCKING. The distinction applied to Claude and withheld from Codex

**Accepted, applied symmetrically — and it broke the conclusion it was meant to
support. §7.3 now has three subsections.**

**§7.3.1** applies the distinction to codex and finds the sentence false as
written. Codex has **two branches** and they price differently. On the packaged
default (`cli_auth_credentials_store = "file"`, ledger 11) the credential is
`auth.json` inside `CODEX_HOME` — a property the vendor **documents** (ledger 20)
— and the `cli|sha256(home)[:16]` derivation **does not exist on that branch at
all**. §7.2's derivation pin does not reach codex's default store.

**§7.3.2 — the premise this review supplied does not hold, and the check is
recorded.** The verdict proposed that codex's `file` store is plaintext
`auth.json`, that §8 refuses a `degraded:plaintext` profile, and that a codex B1
profile is therefore forced onto `keyring` — calling it *"a genuinely better
codex pillar"*. **`bk6owf` §6.2 says the opposite, in a row written to prevent
exactly that reading:**

> **`degraded:plaintext` means *unexplained* plaintext, not plaintext.** It is a
> claim that a store failed open, not a claim about file custody as such.

with codex-on-`file` classified **`active`** — *"this is the vendor's packaged
default custody, not a fallback"* — and the reason given directly: *"Calling
codex-on-`file` `degraded:plaintext` would refuse every default codex enrolment
for a hazard that did not occur."*

So §8 does not force codex onto `keyring`, and adopting the premise would have
preserved the verdict on a **second** false reason. This is recorded in the ADR
(§7.3.2) and in §13's corrections table alongside the producer's own errors,
because a reviewer-supplied premise carries the same standard as any other input
— and it arrived in the direction that would have let the conclusion stand.

**§7.3.3 — what codex *actually* still pays**, which neither round nor the review
named, and which was sitting in the input document the whole time. `bk6owf` §7.1:
*"The codex range applies to **both** store branches. The `file` branch has no
hash that a version bump could break, **but the packaged default that *selects*
that branch is itself a version-fixed constant**."* Round three confirmed the gap
is real: the vendor documents that key's **default value nowhere** — the config
reference lists `file | keyring | auto` with a description and no default, and the
auth page's only worked example sets `"keyring"`, which is *not* what ships
(ledger 30). So there is a genuine §7.2-shaped cost on codex's default branch —
for the **branch selector**, not the hash. **And it is escapable**: the selector
is a documented `config.toml` key and B1 owns each profile's `CODEX_HOME`, so
setting it explicitly removes the dependency — the identical move §7.3 makes for
claude. Recorded as **H13, unstarted**.

**Consequences, all applied:** *"overdetermined"* withdrawn outright; *"three
independent reasons"* withdrawn; §5.2, §6, §9, §10, §11.3 (H3), §12.2, §12.3,
§13, §15 (AC1/AC5/AC6) all re-derived.

## F3 — SHOULD FIX. Ledger row 21

**Fixed, with the method rather than only the numbers.** Re-run: **27 returned,
21 open, 6 closed** — your numbers reproduce exactly. The cause was method, not
phrasing: `gh api search/issues` pages at **30 by default** and the row recorded
no page size or counting step. Row 21 now records
`-f per_page=100` **and** "count `items` by `state`" as part of the method, plus
every closed issue with its `state_reason`: `#14730` completed (usage-caching bug
in the Codex *app*), `#2833` completed, `#17349`/`#19756`/`#15384` duplicate,
`#3573` not_planned. **None closed as already-supported** — the material claim,
unaffected. §2.3's *"the one closed issue is a usage-caching bug"* corrected, and
§0's *"20 account-switching issues, 19 open"* corrected.

The four cited issues were independently re-verified open by direct
`gh api repos/openai/codex/issues/N`.

---

## Where the re-derivation left each verdict

| | Round two | Round three |
| --- | --- | --- |
| A | NO-GO permanent | **unchanged** |
| B0 | GO | **unchanged** |
| B1-claude | CONDITIONAL GO, gated on Q1 | **unchanged** |
| **B1-codex** | **NO-GO**, "closed by the vendor's published position", "overdetermined" | **HELD — UNESTABLISHED.** Not closed, not established. Unstarted. |
| C | GO for managed automation | **unchanged** |
| qwen | not modellable | **unchanged** |

**What actually might close B1-codex, and nobody has run it.** `sqlite_home` is
documented as the directory for Codex's SQLite state DB **with no default
stated**, while `log_dir` *is* documented as defaulting to `$CODEX_HOME/log`
(ledger 29). So it is still unchecked whether two credential-isolated codex
profiles share a state database — the exact cross-account leakage **AC6**
forbids. It is a **free source audit**, it sat in the unknowns table as generic
groundwork for two rounds, and it is now §12.2 item 1 and the top of §11.2. That
is the practical cost of a no-go stated for the wrong reason: three rounds argued
about a vendor sentence while the one free read that could decide the question
went unrun.

§12.2 is retitled *"held, unestablished — what would settle it"* and inverted:
round two's list was five things **OpenAI** must do; three of the seven items are
now **ours and free**. §11.3's closing line, which read *"for B1-codex it means
the vendor has shipped something that does not exist today"*, is corrected —
that followed from the withdrawn no-go.

## The two durable surfaces

- **`README.md`** — the codex sentence is replaced and now says plainly that an
  earlier revision was wrong, carries `desktop`, carries the 403/second-hand
  qualifier, states entitlement as unestablished, and names the free
  `sqlite_home` audit as the first check. It no longer tells an operator OpenAI
  took a position about the Codex CLI.
- **`LOGBOOK.md`** — the 1140 entry gets a **CORRECTION NOTICE** in the same
  shape 1140 itself used for 0842: the widened codex claim struck inline and
  marked `STRUCK`, the 20/19/1 count corrected, the "negative about the surface
  that matters" line corrected to "silent about the surface that matters", and
  the price paragraph's *"stands in full for codex"* struck. Its Claude half is
  untouched — you quote-checked it and it stands. A new **1210** entry carries
  the round-three lessons, of which the reusable one is: *a correction carries
  the same standard as a claim, and the check has to be run in the direction that
  would preserve the answer.*

## Left alone, as instructed

§2.3's Anthropic findings (N8/N9/N10), §7.3's property-versus-derivation
distinction itself, §8 in full (byte-identical — the three-way split, "three
halves is the wrong word and the right count", the two-thirds budgeting), §11's
unstarted status, §12.1's Q1 gate, §13's existing rows, and both logbook entries'
structure. §11 remains unstarted and the delta touches no code file.

## Validation

`go build ./...` exit 0 · `go vet ./...` exit 0 · `go test -count=1` all four
packages with tests **ok**, run in two bounded sequential calls over disjoint
package sets (a single `./...` run exceeds this session's per-call budget; the
previous round was killed at exit 143 mid-run). `cmd/model-harness` has no test
files. Each gate ran standalone with its real exit code; no `tee`, no pipe chain.
Documentation-only delta, so no test was added or modified.
Full log: `TASK-260720-3gcfd1_change-request_rev3-validation.log`.

## Safety boundary

**No account was created, enrolled or authenticated on either provider. Q1 and
Q2 were not run.** No credential, token, cookie or Keychain value read, printed,
exported or persisted. **No `security` invocation of any kind.** No login,
logout, revoke, rotation or re-authentication. No vendor CLI executed this round.
Evidence gathering was unauthenticated `curl` of public vendor documentation,
`gh api`/`gh api graphql` reads of a public repository, one web search, and
read-only reads of this worktree. §11's implementation breakdown remains
unstarted.

## One thing to check first, if you re-review

**§7.3.2.** It refutes a premise this review supplied, using the design document
the review cited. If that refutation is wrong, the codex cost argument is
rescued and the NO-GO comes back — so it is the load-bearing disagreement, and
it is the one place round three could itself be the widened claim. The exact
citation is `bk6owf` §6.2, the row-three cell and the paragraph beginning
*"Rows three and four are the same evidence read in two directions"*.
