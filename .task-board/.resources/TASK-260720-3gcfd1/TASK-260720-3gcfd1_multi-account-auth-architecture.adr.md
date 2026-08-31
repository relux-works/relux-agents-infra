# ADR — Multi-account subscription auth management

Task: `TASK-260720-3gcfd1`
Story: `STORY-260831-yr0x81` · Epic: `EPIC-260720-17k4ra`
Builds on: `TASK-260720-3moaky` (native auth isolation contracts, 2026-08-30),
`TASK-260720-1g880w` (keychain custody and refresh semantics, 2026-08-31),
`TASK-260720-bk6owf` (extensible auth-method lifecycle, 2026-08-31)

Date: 2026-08-31 · Status: **decided, split verdict — revised after review**

Revision 2 replaced round one's single "no evidence at all" entitlement verdict
with a per-provider verdict derived from a documentary read that round one
skipped. **The B1 verdict changed for one provider.** See §2.3 for the read, §0
for the reversal, §7.3 for the cost that no longer applies to it, and §13 for the
corrections this document makes to its own previous rounds.

Revision 3 corrects revision 2 on the codex side, and **it does move a verdict**:
**B1-codex goes from NO-GO to HELD — UNESTABLISHED.** Not because new permission
was found, but because all three of the no-go's stated reasons failed inspection
and nothing replaced them. Revision 2 called B1-codex closed by "the vendor's own
published position", resting that on a sentence scoped to **Codex desktop** whose
scope word it then dropped in all seven places it used the quote; the sentence
does not reach the Codex CLI (N11, demoted). Revision 3 re-reads the CLI-scoped
documentation from a better primary source (N12,
`developers.openai.com/codex/auth.md`), applies §7.3's property-versus-derivation
distinction **symmetrically** rather than to Claude alone — which retires the cost
argument for codex's default branch — and checks the premise review offered to
rescue that argument, which the design document it cites contradicts (§7.3.2).
What survives is one relocated and escapable version-gate cost (§7.3.3) and one
genuinely open, free, unrun isolation audit (Q15). See §7.3, §12.2, §12.4 and §13.

Revision 4 is **the sweep revision**. Review found that the same defect had
recurred a third time — a read this document itself ranked first, called free,
and did not run — and asked for the general fix rather than the instance: **list
every read this ADR calls free and confirm each was actually run.** §4 names
eight. One had been run, one was network-blocked, and **six had not been run in
three rounds**. All six are now run (§2.4).

**Four of the six moved something, and one verdict moves: qwen.**
`sqlite_home` **defaults inside `CODEX_HOME`** (N13), so the risk this document
called *"the substantive one"* across nine sites is **closed, favourably**.
Codex's keyring path has **no** plaintext fallback, so N12a's doc-versus-binary
conflict is **withdrawn** (N16). The store vocabulary has **four** values, not
three (N17). **qwen has a state root — `QWEN_HOME` — so §5.3's "not modellable"
was our own plugin defect reported as a vendor limitation** (N19). And Q7
answered **against** the design: the proposed auth root is nested inside the
installer-managed config dir, not outside it (N18).

**The blocking finding is on the adopted option, not the held one.** The read
that closed Q15 surfaced **`CODEX_SQLITE_HOME`** — a second codex ambient input
present in no round of this ADR, `bk6owf`, the custody research, `README.md` or
`LOGBOOK.md` — and the enumeration method that found it then found **six more**
codex environment inputs (N15) and an entire **second, profile-named claude
credential store** (N20). §5.2's *"Namespace inputs: **One**"* and §11.2's
**"W13 is the only one B0 depends on"** were both false: they sized each
runtime's ambient-input set from its *credential-namespace* input count. **B0 is
still GO** — every one of these is a further argument for composing the child
environment — but its prerequisite list goes from one item to five (W13, W20,
W21, W22, W24), and §10.1 gains three per-variable negative gates.

**B1-codex does not move, and round four is explicit about why that is not a
disappointment.** Round three said *"most of what would settle it is ours and
free"* and named Q15, Q12 and H13. The two reads were run; both came back
favourable; **neither settled anything**, because both were mechanism questions
and what is open is **entitlement**. The only thing left that can decide
B1-codex is a second account (Q2). See §2.4, §12.2, §12.4 and §13.

**Verdicts after round four.** A permanently NO-GO. **B0 GO** — with a larger
prerequisite list. C GO. **B1-claude CONDITIONAL GO.** **B1-codex HELD —
UNESTABLISHED.** **qwen: provisional, but no longer "not modellable"** — a
mechanism exists. Both halves of B1 remain **unstarted**.

Revision 5 is **the mirror-image sweep**. Round four propagated its two adverse
corrections (N18, N19) thoroughly and left its two **favourable** ones — N13
(`sqlite_home` defaults inside `CODEX_HOME`) and N16 (the keyring→file fallback
is gated on `Auto`) — standing as open at five sites, three of them named
acceptance-criteria deliverables and one a self-contradiction inside §13's own
corrections table. §6's B1 verdict row called `sqlite_home` "a live cross-account
leakage risk" after N13 had closed it; the B0 Security row still sized the
namespace at N1's three variables after N15/N20 widened it; §8 counted the N4
fallback hazard as adverse after N16 withdrew it; §13 asserted Q12 "still not
answered" three rows below the entry withdrawing it; §11.3 still promised Q15
"could still close B1-codex" after round four ran it and found it did not. All
five are corrected below, using the same `~~struck~~ — withdrawn` convention
round four introduced. **This did not move a verdict** — every outcome from
round four stands. Round five also ran the two documentary halves the sweep's
own reclassification skipped: codex's store-selector default is stable across
seven `rust-v*` tags (N21), and claude's config-dir derivation applies NFC
normalization only, nothing more (N22) — both partial answers, neither closing
their unknown. See §6, §8, §11.3, §13 and §2.4.1.

---

## 0. The decision, up front

**Six** things are being decided, not one, because the epic's question
decomposes into parts with genuinely different answers — and, after the
documentary read in §2.3, the multi-account part decomposes again **per
provider**. Forcing one verdict across them would be the forced fit the epic's
own notes warn against; round one avoided it on the option axis and committed it
on the provider axis. **Round three found the opposite failure on the same axis**
— an asymmetry asserted more sharply than the evidence carried — and shrank it.
The providers do differ. They differ less, and elsewhere, than round two said.

**Round four adds the sixth row and shrinks the asymmetry once more.** qwen was
carried for three rounds as a footnote reading *"not modellable, no mechanism"*,
on evidence that was our own plugin declaration; the free audit that would have
checked it was listed in every round and run in none. It has a state root. It is
now a row rather than a footnote, because a runtime with a mechanism and no
entitlement evidence is the same shape as B1-codex, not the same shape as
nothing.

| | What it is | Verdict |
| --- | --- | --- |
| **A** | agents-infra owns the credential record — native login adopted into an agents-infra Keychain item | **NO-GO. Permanent, not conditional.** |
| **B0** | agents-infra owns the *state root* and composes the child environment deterministically — **one** account per runtime | **GO. Adopt now.** Fixes a live defect; depends on no undocumented derivation. **Round four: the prerequisite list is five items, not one** — the ambient-input set was sized from the wrong enumeration in every previous round (N14, N15, N19, N20). |
| **B1-claude** | The same mechanism extended to **multiple simultaneously enrolled** Claude accounts | **CONDITIONAL GO.** Vendor-documented mechanism, vendor-documented use case, one empirical gate left. |
| **B1-codex** | The same, for Codex | **HELD — UNESTABLISHED.** *Round three changed this from NO-GO; round four does not move it.* **Round four discharged both free reads that round three said could settle it** — `sqlite_home` defaults inside the root, and the keyring path has no plaintext fallback; both favourable, neither decisive. What remains needs a **second account** (Q2). §12.2. |
| **C** | Provider-native switching and delegation — API key, `apiKeyHelper`, named profiles, workload identity | **GO for managed automation.** Already supported; does not solve B1. **Round four adds a concrete instance nobody had documented here:** Claude Code 2.1.248 ships a file-based **named-profile** credential store (`ANTHROPIC_CONFIG_DIR`/`ANTHROPIC_PROFILE`, `configs/<n>.json` + `credentials/<n>.json`, `user_oauth` \| `oidc_federation`). Its enrolment path is **unread** (Q17), so it is recorded as a mechanism sighting, not as a delivered option. |
| **qwen** | Per epic AC8 | **PROVISIONAL — but no longer "not modellable".** *Round four changed this.* `QWEN_HOME` exists and namespaces the credential; the empty `HomeEnvVar` was **our** plugin defect (N19). Entitlement, refresh and revocation remain unaddressed. §5.3. |

**The epic's headline capability — "select provider plus identity, authorize
natively, then switch among multiple Claude *and* Codex accounts" — is not
deliverable as a single unified capability today. Neither half is built. Neither
half is closed.** Round three wrote that the two halves sit at different
evidentiary maturity: *"Claude is one experiment from a decision, Codex is
several free reads from even having one."* **Round four ran those free reads, so
the second clause is now wrong — and the halves turn out to be closer than that
sentence claimed.** Both are one experiment from a decision, and it is the same
experiment on each side: does a second concurrently-enrolled account leave the
first working, server-side, over time (Q1 for claude, Q2 for codex). The real
difference is not distance; it is that Anthropic **documents and recommends** the
workflow while OpenAI documents the credential's location and stops.

**This is a reversal of the previous round of this decision, and the reason is
worth stating plainly.** The previous round asserted that the deciding axis had
*"no evidence at all"* — that nobody had established whether either vendor
permits one human to hold two concurrently enrolled logins. That assertion was
false. It described a **read that had not been performed**, not an absence, and
it was propagated into `README.md` and the logbook as a durable claim about the
state of the evidence. §4 had classified the read as "a decision, and it is not
the agent's to make", which is what licensed skipping it; reading published
terms and product documentation is a free documentary read, and the *judgement*
about whether to rely on what they say is the only part that was ever the
operator's. The read has now been performed (§2.3), against published
first-party material only, with no account created or enrolled on either
provider.

What it found is **asymmetric**, which is why one verdict across both providers
was the wrong shape:

- **Anthropic is not silent. It documents this exact mechanism for this exact
  use case.** `code.claude.com/docs/en/authentication` states that Claude Code
  *"keys the macOS Keychain entry to that directory too, so a session with a
  different `CLAUDE_CONFIG_DIR` reads a different entry"* (N8), and the
  environment-variable reference calls `CLAUDE_CONFIG_DIR` *"useful for running
  multiple accounts side by side"* with a worked `claude-work` alias (N9). The
  Consumer Terms restrict **sharing** an account with other people, not the
  number of accounts one person holds (N10). The property B1 needs is not an
  undocumented derivation on this provider — it is a published product contract.
- **OpenAI publishes nothing that addresses the Codex CLI — and rounds two and
  three both had to correct themselves here.** OpenAI ships and documents
  concurrent dual-account sign-in for ChatGPT web — two accounts, all plan types,
  explicitly for keeping work and personal separate — which is entitlement
  evidence in the **permissive** direction. The same article adds that switching
  is *"not yet supported in **Codex desktop** or the native ChatGPT mobile
  apps"* (N11). Round two treated that as the vendor closing B1-codex.
  **It is not.** It names two GUI surfaces and the Codex CLI is not among them;
  *"not yet"* is a roadmap note, not a prohibition; and the absence of a
  session-switcher **UI** from a command-line tool says nothing about how many
  logins an operator may hold. N11 is demoted to corroboration and decides
  nothing. What *is* established, from successful primary reads: the CLI-scoped
  auth documentation mentions multi-account **nowhere** and describes a
  **single** cached login per state root (N12); the capability is requested and
  unshipped across Codex surfaces, with 27 tracker issues, 21 open and none
  closed as already-supported (N12b); and OpenAI's own discussion thread on
  exactly this question has **no vendor reply at all** (N12c).

  **So the honest status of entitlement for the Codex CLI is *unestablished*.**
  That is a third thing, different from round one's "no evidence" (an unperformed
  read) and round two's "closed by the vendor's published position" (a widened
  quote). Nobody has published a yes and nobody has published a no. **Round four
  leaves this untouched and removes everything that was standing next to it**:
  the isolation objection (Q15) and the keyring-fallback objection (Q12) were
  both free reads, both were run, and both came back in codex's favour. Stripping
  them out is what makes the remaining sentence short enough to be honest —
  entitlement, and nothing else, and it needs an account.

- **And once entitlement stopped carrying the codex verdict, the cost argument
  had to carry it — and it does not. That is why B1-codex moved from NO-GO to
  HELD.** §7.3's property-versus-derivation distinction, the thing that rescued
  Claude, was applied to Codex only in round three, and applying it symmetrically
  broke round two's second reason too. Codex's **packaged default** store is
  `file`, on which the credential is simply `auth.json` inside `CODEX_HOME` — a
  property the vendor **documents** — with **no derivation on the path** at all;
  the `sha256` account derivation exists only on the keyring branch. The premise
  offered to rescue the cost argument, that §8's plaintext refusal forces codex
  onto keyring, is contradicted by the design it cites: `bk6owf` §6.2 classifies
  codex-on-`file` as **`active`**, explicitly *"the vendor's packaged default
  custody, not a fallback"* (§7.3.2). What survives is a **real but different and
  escapable** cost — a version gate on the store *selector*, whose default the
  vendor documents nowhere (§7.3.3). ~~plus a genuine open isolation gap, Q15's
  undocumented `sqlite_home` default~~ — **withdrawn in round four: it defaults
  inside `CODEX_HOME`** (N13). That is a reason to leave B1-codex unstarted. It
  is not a reason to tell a later reader it is closed.

The single unknown that genuinely survives on the Anthropic side is narrower
than the one the previous round claimed: whether one human may hold two
concurrently **billed consumer subscriptions**. Nothing published addresses it.
It is also mostly beside the point — the documented scenario is work plus
personal, which in practice is a subscription plus an organization seat, or a
subscription plus Console, and both of those are explicitly supported (N10). An
operator who already holds two accounts they are entitled to does not need this
question answered; an operator who would have to buy a second Pro to find out
does, and that is a purchasing decision, not a research one.

**B1-claude is therefore CONDITIONAL GO, not GO.** The remaining gate is
empirical, not documentary: whether enrolling a second account under its own
state root leaves the first working, server-side, over time (Q1). Vendor
documentation recommending the workflow makes that outcome likely; likely is not
established, and this ADR does not promote an inference to a finding. What has
changed is that the gate is now **one experiment an operator can run with
accounts they already have**, rather than an unanswered question about whether
the capability is permitted at all.


**B0 is the reason this ADR is not a bare "no".** The plumbing gap that blocks
B1 is also a live correctness defect today, independent of multi-account, and
fixing it requires none of the machinery B1 would need — no version gate, no
namespace pin, no second account, no credential of any kind. It is worth doing
on its own merits and it is the exact prerequisite B1 would need later.

---

## 1. Safety boundary held by this task

No credential, token, cookie or Keychain secret value was read, printed,
exported or persisted. **No `security` invocation of any kind was made**, so no
Keychain item was queried, created, modified or deleted — not even for metadata.
No login, logout, revoke, rotation or re-authentication ran. No vendor CLI was
executed except `claude --version` and `codex --version`.

New evidence in this task comes from two rounds, both read-only:

1. **Byte inspection of installed vendor binaries** — five Claude Code builds
   under `~/.local/share/claude/versions/` and the Codex 0.150.1 native binary
   under Homebrew — plus reads of the three input documents (§2.2).
2. **A documentary read of published first-party vendor material** (§2.3):
   Anthropic's Claude Code documentation, consumer terms and support articles;
   OpenAI's reachable Codex documentation; and the `openai/codex` repository via
   `gh api`. **No account was created, enrolled or authenticated on either
   provider.** Entitlement was attacked with documents, never with an experiment.

Nothing was written outside this repository.

---

## 2. Evidence base

### 2.1 Inherited, and its standing

**`TASK-260720-3moaky`** — native auth isolation contracts. Its findings stand
**except decision-summary item 4**, which is **refuted**. Item 4 states that
Claude native macOS login has no supported per-account selector, that
`CLAUDE_CONFIG_DIR` relocates credentials on Linux and Windows only, and that
multiple simultaneous native Claude accounts are therefore not an isolation
boundary on macOS. It is **refuted on both counts**: `CLAUDE_CONFIG_DIR`
namespaces the macOS Keychain **service name** (implementation, `1g880w`), *and*
Anthropic documents that it does so, for macOS specifically, on the Claude Code
documentation site (N8) — while the environment-variable reference names running
multiple accounts side by side as the variable's use, with a worked example (N9).
The audit's related forbidding line — "Treating `CLAUDE_CONFIG_DIR` as macOS
native credential isolation" — is therefore **refuted outright**. The previous
round of this ADR let that line survive as a statement about *support* ("the
mechanism works and is not promised"); the documentary read removes that escape.
Everything else in that audit is cited here as whole and current.

**`TASK-260720-1g880w`** — keychain custody and refresh, accepted and
independently re-derived by its reviewer. Its per-provider matrix, hard blockers
and evidence ledger are taken as established. Two of its listed unknowns are
settled below.

**`TASK-260720-bk6owf`** — the auth-method and credential lifecycle design, 1049
lines, reworked against a review verdict and swept document-wide for its two
invariants. Its data model, adapter contract, lifecycle and CLI grammar are the
design this ADR decides *whether and when to build*. Its own honest downgrade is
carried forward unchanged: §6.1 splits the plaintext hazard **three** ways, not
two, and one third is mitigated rather than eliminated (§8 below).

### 2.2 Verified in this task

Seven findings, all from read-only binary inspection. Four are new; two settle
questions the inputs left open; one closes a gap in a stated version range.

---

**N1 — Claude's Keychain service name has three environment inputs, not one.**

The construction, byte-identical in substance across all five installed builds
(2.1.234, 2.1.235, 2.1.236, 2.1.247, 2.1.248; identifier names differ by
minifier revision):

```js
function s(){ return "prod" }                       // build channel: a hardcoded constant
function pct(){                                     // fileSuffixForOauthConfig
  if (process.env.CLAUDE_CODE_CUSTOM_OAUTH_URL) return "-custom-oauth";
  switch (s()) { case "local": return "-local-oauth";
                 case "staging": return "-staging-oauth";
                 case "prod": return "" }
}
function F(){                                       // getOauthConfig
  let t = (prod config);
  let o = process.env.CLAUDE_CODE_CUSTOM_OAUTH_URL;
  if (o) { let a = o.replace(/\/$/,"");
           if (!l.includes(a)) throw Error("CLAUDE_CODE_CUSTOM_OAUTH_URL is not an approved endpoint.");
           t = { ...t, /* ... */ OAUTH_FILE_SUFFIX: "-custom-oauth" } }
  return t
}
function Gv(){                                      // storage dir
  let n = process.env.CLAUDE_SECURESTORAGE_CONFIG_DIR;
  if (n !== void 0) return (n || join(homedir(), ".claude")).normalize("NFC");
  return ge()
}
function ok(n = ""){                                // the Keychain SERVICE name
  let e = process.env.CLAUDE_SECURESTORAGE_CONFIG_DIR,
      t = e !== void 0 ? !e : !process.env.CLAUDE_CONFIG_DIR,
      r = e !== void 0 ? e.normalize("NFC") : ge(),
      c = t ? "" : `-${sha256(r).hex.slice(0,8)}`;
  return `Claude Code${Gt().OAUTH_FILE_SUFFIX}${n}${c}`
}
var eq = "-credentials";
```

So the service name is
`Claude Code` + `OAUTH_FILE_SUFFIX` + `-credentials` + `[-<8 hex>]`,
and **three** separate environment variables feed it:

| Variable | Effect on the namespace | Modelled by `bk6owf`? |
| --- | --- | --- |
| `CLAUDE_CONFIG_DIR` | appends `-<sha256(NFC(dir)).hex[:8]>` | yes — it is the design's whole mechanism |
| `CLAUDE_SECURESTORAGE_CONFIG_DIR` | **replaces** the dir input entirely, independent of the config dir | yes — refused at enrol and launch (§7.3) |
| `CLAUDE_CODE_CUSTOM_OAUTH_URL` | inserts the literal `-custom-oauth` infix | **no** |

The third is new. Its reachability is narrower than the other two and worth
stating precisely rather than dramatically: `getOauthConfig` **throws** unless
the value is one of three approved endpoints — `ALLOWED_OAUTH_BASE_URLS` =
`["https://beacon.claude-ai.staging.ant.dev", "https://claude.fedstart.com",
"https://claude-staging.fedstart.com"]`. So an arbitrary value does not silently
repoint the namespace; it makes Claude Code throw at OAuth-config time. Either
outcome is launch-affecting, neither is modelled, and both arrive by
**inheritance** from an ancestor process today (N2, and `bk6owf` §12 row 1).

`fileSuffixForOauthConfig` — which returns `-custom-oauth` for *any* non-empty
value, with no allowlist check — is exported separately and drives the
`OAUTH_GLOBAL_FILE_SUFFIXES` set `["", "-staging-oauth", "-local-oauth",
"-custom-oauth"]`, i.e. file-side naming rather than the Keychain service. The
two paths agree on the suffix and disagree on the gate. That divergence is
itself a reason not to treat "the namespace is the config dir hash" as the whole
truth.

---

**N2 — Setting `CLAUDE_CONFIG_DIR` to the default path is not a no-op, and this
corrects `bk6owf` Invariant L1.**

From `ok()`: `t = e !== void 0 ? !e : !process.env.CLAUDE_CONFIG_DIR`. The suffix
is empty **only** when `CLAUDE_CONFIG_DIR` is unset or empty. Any non-empty value
— including the literal default `~/.claude` — produces `-<hash>`.

`1g880w` proved both halves empirically: a synthetic dir produced the queried
service `Claude Code-credentials-31c6920d` = `sha256(NFC(dir)).hex[:8]`, and the
live item's service is exactly `Claude Code-credentials` with **no** suffix,
which is what `ok()` produces when the variable is unset. Combining that proven
experiment with the source above, the conclusion follows without a new run:

> The default namespace is addressable **only** by leaving `CLAUDE_CONFIG_DIR`
> unset or empty. Writing it explicitly — even to the exact default path — lands
> the launch in a different, empty namespace.

`bk6owf` §5.2 states Invariant L1 as "the home variable is written on every
launch of a home-bearing runtime, never inherited … When no profile is selected,
the launch writes the *default* home explicitly." For codex that is right. **For
claude, as written, it would move every default launch into an empty namespace
and report the operator as not logged in.** The invariant's intent — the value
comes from a decision, never from an ancestor — is correct and important. Its
expression must become:

> **L1′.** On every launch the child environment is **composed** over the full
> namespace-input set, not inherited. For a selected profile that means writing
> the recorded state root byte-verbatim. For the default profile it means
> explicitly **removing** `CLAUDE_CONFIG_DIR` from the child environment.
> `CLAUDE_SECURESTORAGE_CONFIG_DIR` and `CLAUDE_CODE_CUSTOM_OAUTH_URL` are
> removed in both cases unless a profile records them deliberately.

Removal is as much a decision as assignment; inheritance is the thing L1 was
built to forbid, and an explicit `unset` satisfies it. This is the single most
consequential correction in this ADR, because L1 as written would have been
implemented literally and would have broken the default launch on the operator's
primary tool.

---

**N3 — `OAUTH_FILE_SUFFIX` is non-empty in a shipped production build. Settles a
`1g880w` unknown, positively.**

`1g880w` listed as an open unknown: "Whether `OAUTH_FILE_SUFFIX` is ever
non-empty in a shipped build (staging/internal channels), which would move the
service name. Cost: free. Not run here." Run here. Answer: **yes.**

All five installed builds carry all four suffix values in their OAuth config
module. The build channel is the hardcoded constant `"prod"`, so `-local-oauth`
and `-staging-oauth` are unreachable without patching the binary — but
`-custom-oauth` is selected from a plain environment variable at runtime (N1).
The unknown resolves to: the suffix is empty on the default path of the shipped
binary, and is operator-reachable, and the design must therefore treat it as an
input rather than a constant.

---

**N4 — Codex has a keyring→file fallback. Narrows `bk6owf` Q12 from unknown to
"a fallback path exists; its gating is unestablished."**

Adjacent string literals in `login/src/auth/storage.rs` within the Codex 0.150.1
native binary:

```
failed to load CLI auth from keyring, falling back to file storage:
failed to save auth to keyring, falling back to file storage:
failed to remove CLI auth fallback file:
failed to delete auth from keyring:
failed to load CLI auth from encrypted auth storage:
failed to deserialize CLI auth from encrypted auth storage:
failed to write OAuth tokens to encrypted auth stora…
```

Q12 asked what codex's failure posture is on the `keyring` store, and the prior
research could only establish that `auto` falls back to a plaintext `auth.json`.
There is now current-source evidence that a keyring→file fallback path exists in
the login auth-storage module on **both** the load and the save leg, with a
matching "fallback file" removal path.

**The honest label matters here.** These are string literals, not control flow.
String inspection cannot establish whether the fallback is gated on
`store == auto` or fires on the `keyring` selector too. So Q12 moves from
*unknown* to *a fallback exists; its gating is unestablished* — a narrowing, not
an answer. Its consequence for the decision is asymmetric and therefore worth
acting on: if the fallback is ungated, codex on `keyring` inherits Claude's
fail-open posture, `bk6owf` §6.2's `degraded:plaintext` classification for
codex-keyring is right for the right reason, and §5.3's prohibition table
becomes load-bearing on that branch too. `bk6owf` chose to refuse rather than
guess in either direction. That choice is now better supported than when it was
made.

A third backend also appears — "encrypted auth storage", with `CODEX_AUTH`
described as a secret name — which is **not** in the design's
`file | keyring | auto` store vocabulary. Unmodelled; see Q14.

---

**N5 — Codex's service name is a single literal. The providers are asymmetric.**

`Codex Auth` occurs exactly once in the binary, with no environment-derived
prefix, infix or suffix; the account prefix `cli|` likewise. So codex's namespace
has **one** input (`CODEX_HOME`, canonicalized and hashed into the account) where
claude has three (N1). Any design that says "namespace the state root" without
naming which provider it is talking about is under-specified.

---

**N6 — 2.1.235 verified; the claude version range is four-then-five verified
points with an interpolated interval between them.**

`bk6owf` §7.1 sets the claude supported range to **2.1.234 – 2.1.248** on the
basis that the construction is byte-identical across 2.1.234, 2.1.236, 2.1.247
and 2.1.248. 2.1.235 is installed on this machine and was not in that set. It is
now verified: 2.1.235 carries the same `rte()`/`ok()` construction with the same
`sha256 … substring(0,8)` derivation and the same
`!process.env.CLAUDE_CONFIG_DIR` test.

That closes the local gap and exposes the general one. Five points are verified;
the interval **2.1.234 – 2.1.248** asserts ten further versions
(2.1.237 – 2.1.246) that nobody has seen. A range stated as an interval quietly
claims them. The gate should therefore be an **allowlist of verified versions**,
not an interval — or, if an interval is kept for operational reasons, the
document must say plainly that it interpolates. This ADR chooses the allowlist
(§9), and the choice makes the version-gate cost *worse*, which is the point:
the cost is real and pricing it honestly is what §7 is for.

---

**N7 — `CODEX_HOME` is not codex's only state location.**

The packaged config surface includes `sqlite_home`, `log_dir`, `history`,
`forced_login_method` and `forced_chatgpt_workspace_id` alongside
`cli_auth_credentials_store`. Whether `sqlite_home` and `log_dir` default inside
`CODEX_HOME` was unestablished when this note was written. **Answered in round
four, and both default inside** (N13, ledger 31): `log_dir` → `$CODEX_HOME/log`,
`sqlite_home` → `$CODEX_HOME` unless overridden. The residual is *ambient*, not
architectural — `CODEX_SQLITE_HOME` relocates the DB out of a composed root
(N14) — and it is escapable per-profile because `config.toml` outranks the
variable. `forced_login_method` is still directly relevant to the auth-method
registry and is still unmodelled. See Q15 (answered) and Q16 (open).

Independently re-confirmed here: the packaged fixed-defaults block contains
`cli_auth_credentials_store = "file"` and `mcp_oauth_credentials_store = "auto"`.
The `1g880w` finding that codex's effective default custody is **file**, not
Keychain, stands.

---

### 2.3 Verified in this task, round two — the documentary read

The first round of this task treated vendor entitlement as unresearched-because-
unresearchable and said so in §0 as *"no evidence at all"*. Review found that
claim false and directed the read. It was performed on 2026-08-31 against
published first-party vendor material only. **No account was created, enrolled
or authenticated on either provider**; the boundary in §1 held unchanged.

The read did not merely add evidence. It **changed a verdict**, and it changed
it on the provider the previous round called the silent one.

**Round three re-ran the codex half of this read**, after review found that N11's
scope word had been dropped everywhere the quote was used. The re-read used the
markdown source endpoints rather than the rendered HTML, and it added one source
round two did not consult (the vendor's own discussion thread on exactly this
question). It moved no verdict. It changed what the codex verdict rests on, and
it retired one of the three reasons round two gave. The findings below are marked
where round three altered them.

#### 2.3.1 Read status, stated before the findings

An absence and a failure to read are different facts, so the reachability of
each source is recorded first.

| Origin | Reachable from this host | Consequence |
| --- | --- | --- |
| `code.claude.com/docs/**` | **Yes**, HTTP 200, including the `.md` source endpoints | Primary-source quotes below are exact |
| `support.claude.com/**`, `anthropic.com/legal/**` | **Yes**, HTTP 200 | Primary-source |
| `learn.chatgpt.com/docs/**` (canonical target of `developers.openai.com/codex/*`) | **Yes**, HTTP 200 | Primary-source |
| `developers.openai.com/codex/*.md` — the **markdown source endpoints** | **Yes**, HTTP 200. *Found in round three; round two used only the rendered HTML* | Primary-source, and better: no tag-stripping step between the vendor's text and this ADR |
| `github.com/openai/codex` | **Yes**, via `gh api` | Primary-source repo; issue bodies are third-party |
| `help.openai.com/**` | **No. HTTP 403 to every request**, `curl` with a browser UA and the fetch tool alike | **Failed read, not an absence** |
| `openai.com/policies/**`, `chatgpt.com/**` | **No. HTTP 403** | **Failed read, not an absence** |

Everything sourced from `help.openai.com` below is therefore **second-hand**:
its text reached this task through the search index on two independently-phrased
queries, and independently through the review that raised the finding. It is
recorded as a **strong documentary lead, not as a verified primary quote**, and
every claim resting on it is labelled. The OpenAI Terms-of-Use clause the review
quoted (multiple accounts restricted only for free-tier credit farming) **could
not be corroborated from here at all** — the origin is 403 and the index did not
return the clause text. It is carried as reviewer-supplied and unverified, and
nothing below depends on it.

---

**N8 — Anthropic documents the macOS Keychain namespacing. The mechanism this
ADR called undocumented is a published product contract.**

`code.claude.com/docs/en/authentication`, under *Credential management*,
verbatim:

> If you've set the `CLAUDE_CONFIG_DIR` environment variable, Claude Code keeps
> the `.credentials.json` file under that directory instead, including the file
> the macOS fallback writes, **and keys the macOS Keychain entry to that
> directory too, so a session with a different `CLAUDE_CONFIG_DIR` reads a
> different entry.**

That sentence states the exact property B1 needs — *distinct config dir ⇒
distinct macOS Keychain entry* — as vendor documentation, on the vendor's own
Claude Code documentation site, for macOS specifically.

The same page also documents the plaintext fallback §8 is built around:

> On macOS, credentials are stored in the encrypted macOS Keychain. When the
> Keychain rejects the write, such as when it's locked in an SSH session, Claude
> Code stores your login in `~/.claude/.credentials.json` with file mode `0600`
> instead, the same storage it uses on Linux.

---

**N9 — Anthropic documents multiple accounts side by side as the intended use
of `CLAUDE_CONFIG_DIR`, with a worked example.**

`code.claude.com/docs/en/env-vars`, the `CLAUDE_CONFIG_DIR` row, verbatim:

> Override the configuration directory (default: `~/.claude`). All settings,
> session history, and plugins are stored under this path. For credentials, see
> where Claude Code stores credentials. **Useful for running multiple accounts
> side by side: for example, `alias claude-work='CLAUDE_CONFIG_DIR=~/.claude-work claude'`.**
> Set it in your shell, user settings, or managed settings. Ignored in project
> and local settings.

The alias in the vendor's own example is named `claude-work`. The documented
scenario is **one operator, a work account and a personal account, side by
side** — which is the epic's scenario, published by the vendor as a supported
use of the variable.

This does not by itself license *any* pair of accounts, and it is not a billing
statement (see N10). What it does do is destroy the previous round's premise:
Anthropic is not silent about one human running multiple Claude accounts through
Claude Code. It documents how.

---

**N10 — Anthropic's terms restrict sharing, not count.**

`anthropic.com/legal/consumer-terms`, verbatim:

> You may not share your Account login information, Anthropic API key, or
> Account credentials with anyone else. You also may not make your Account
> available to anyone else.

The restriction is on making one account available to **other people**. The
terms contain **no clause limiting how many accounts one person may hold**.
Corroborating support material, all first-party and reachable:

- *Can I have a Claude account and a Console account?* — "You can have a Claude
  account (free, Pro, Max, Team, or Enterprise) and a Console account (to access
  the Claude API) with the same email address… These two accounts will operate
  independently."
- Personal-account/organization coexistence: a personal Claude account on the
  same email as an organization stays active and the member switches between
  them from the account menu.
- *Account management FAQs* and *Use Claude Code with your Pro or Max plan*
  address **neither** multiple concurrent personal subscriptions nor account
  switching. Checked; genuinely silent on that narrower question.

The same terms also carry an automated-access clause — access by "bot, script,
or otherwise" is prohibited "**except** when you are accessing our Services via
an Anthropic API Key **or where we otherwise explicitly permit it**". Claude Code
on a Pro/Max login is an explicitly-supported first-party client, and B0/B1
launch that client rather than substituting for it, so this sits inside the
exception. It is recorded because it is the clause an agents-infra launcher
would violate if it ever stopped launching the vendor's own CLI.

---

**N11 — OpenAI publishes account switching for ChatGPT web. It says nothing
about the Codex CLI.** *(Second-hand: `help.openai.com` is 403 from here.
**Demoted in round three from a pillar to corroboration** — see the scope
analysis below.)*

Article *Use multiple accounts with account switching*, as returned by the
search index on three independently-phrased queries across two rounds:

> Account switching lets you stay signed in to two ChatGPT accounts at the same
> time and switch between them instantly — no logging out required. This makes
> it easy to keep your **work and personal** usage separate while using the same
> browser or device.

> You can have a maximum of two accounts active in the switcher per session…
> You can still create additional accounts, but you'll need to log out to access
> more than two at once.

> Account switching is currently available on ChatGPT web; it is not yet
> supported in **Codex desktop** or the native ChatGPT mobile apps.

The first two quotes are entitlement evidence in the **permissive** direction —
a vendor that ships, documents and supports concurrent dual sign-in for work and
personal use is not treating two logins per human as a violation.

**The third quote is where round two went wrong, and the correction matters more
than the quote.** Round two reproduced the sentence accurately here and then used
it in a compressed form in seven places — §0, §5.2, §9, §12.2, §12.3, `README.md`
and the `LOGBOOK.md` entry — none of which carried the word **desktop**. Read at
its own scope, the sentence will not support what was built on it:

1. **Surface.** It enumerates two surfaces: **Codex desktop** and the native
   ChatGPT mobile apps. Both are GUI applications. B1-codex decides the **Codex
   CLI**, which is not in the list, and the ADR never argued that it could
   extend the scope. `openai/codex#31778` — *Multi-account support / Account
   switching in the **Codex desktop app*** — confirms that "Codex desktop" is a
   distinct named product surface rather than loose phrasing for "Codex".
2. **Kind.** It reports the *availability of a feature*: a session-switcher UI
   built for ChatGPT web has not been extended to other apps. A switcher UI is
   trivially absent from a command-line tool, and that absence is not a vendor
   position on how many logins one operator may hold. Promoting feature
   availability to an entitlement statement is a category change the sentence
   does not license.
3. **Tense.** *"not yet"* presupposes intent to ship. It is the opposite shape
   from a prohibition.

**N11 is therefore demoted to corroboration**: it is good evidence that account
switching is *unshipped* on OpenAI's client surfaces generally, and it is a
useful cross-check on N12b. It is **not** the vendor closing the Codex CLI, and
nothing in this ADR's verdict rests on it any more. The CLI-scoped legs — N12, a
successful primary read, and §7.3's symmetric cost analysis — carry it instead,
and they are the better arguments.

**This is the same defect the ADR spent two rounds correcting**, committed in its
own headline: a claim stated precisely once and used in a widened form
thereafter. Round one asserted an absence it had not checked; round two checked
it, quoted it correctly, and then spent the quote at a scope it does not have.

---

**N12 — OpenAI's reachable first-party Codex material is silent on multi-account,
and its own issue tracker shows account switching unshipped and leaking.**

**Round three re-read this from a better source.** Round two used the rendered
HTML at `learn.chatgpt.com/docs/auth` and had to strip tags. The vendor also
publishes the **markdown source** at `developers.openai.com/codex/auth.md`
(HTTP 200), which is the same document without an extraction step, and which is
explicitly **surface-scoped**: its body is divided into `ContentModeSwitch`
blocks tagged `app`, `cli`, `ide` and `web`, so a claim about the **CLI** can be
read off the vendor's own scoping rather than inferred. Both endpoints were read;
they agree. Quotes below are from the `.md` source.

It documents credential storage, in a block scoped to `app,cli,ide`:

> `file` stores credentials in `auth.json` under `CODEX_HOME` (defaults to
> `~/.codex`). `keyring` stores credentials in your operating system credential
> store. `auto` uses the OS credential store when available, otherwise falls
> back to `auth.json`.

and it **does not mention multiple accounts, account switching, or `CODEX_HOME`
as an account-isolation boundary anywhere**. `CODEX_HOME` appears **exactly
once** in the whole document — in the sentence above, as the location of
`auth.json`. Zero occurrences of "multiple account", "account switch" or
"switching". This is a genuine documentary silence, established by a successful
read of the page that would carry the statement, on the surface that decides the
question.

**What the page does say about the CLI's login model, and it is not neutral.**
From the *Login caching* block, scoped `app,cli,ide`, verbatim:

> When you sign in to the ChatGPT desktop app, Codex CLI, or IDE extension using
> either ChatGPT or an API key, **your login details are cached and reused. The
> CLI and extension share the same cached login details. If you log out from
> either one, you'll need to sign in again** the next time you start the CLI or
> extension.

> Codex caches login details locally in a **plaintext file** at
> `~/.codex/auth.json` or in your OS-specific credential store.

Two things follow that round two did not have. First, the vendor describes
**one** cached login per state root, singular throughout, with no notion of a
second one held alongside it — that is not a prohibition, but it is the model the
documentation actually describes, and it is the closest thing to a first-party
statement about the CLI surface that exists. Second, **the vendor itself calls
the default store plaintext**, in its own words. §7.3 and §8 both turn on that,
and it is now a quotation rather than an inference.

Three consequences follow.

**(a) A documentation-versus-binary discrepancy on the keyring fallback,
adverse to `keyring`.** The doc promises a fallback to `auth.json` for `auto`
only; `keyring` is documented as storing in the OS credential store, full stop.
N4 found fallback strings on **both** the load and the save leg of the keyring
path. So either the fallback is gated on `auto` and the strings are shared, or
`keyring` silently falls back to plaintext in a way the documentation does not
disclose. **Q12 is therefore not merely unresolved — the two available sources
disagree**, and until that is settled `keyring` cannot be relied on to mean
"never plaintext". This is adverse to `3moaky`'s "force `keyring`" implication
and strengthens §8's hazard rather than softening it.

**(b) Codex account switching is an open, unshipped, and buggy area.** In
`github.com/openai/codex`, first-party repository, read via `gh api`. A title
search for account switching returns **27 issues, 21 of them open and 6
closed** *(corrected in round three; ledger row 21 recorded 20/19/1, which does
not re-run — see §14)*, among them `#31778` *Multi-account support / Account
switching in the Codex desktop app*, `#30684` *Add account/workspace switching
support*, `#18806` *Support switching ChatGPT/OpenAI accounts without losing the
current session*, and `#34111`. All four verified open by direct `gh api` read.

The six closed issues, with their `state_reason` as GitHub reports it: `#14730`
*completed* (a usage-caching bug in the Codex **app**), `#2833` *completed* (a
web-login/API-key switching 403), `#17349`, `#19756` and `#15384` *duplicate*,
and `#3573` *not_planned* (an unrelated Azure failover bug). **None is closed as
already-supported**, which is the material claim, and it survives the corrected
count intact — six closed issues make the "requested and unshipped" reading
stronger, not weaker, because none of them closed by shipping it.

More decisive for the isolation claim, the tracker also carries reports of
**cross-account state surviving a switch**: `#35657` (previous account's projects
and conversations shown), `#21314` (credits consumed from the original account
after switching), `#16894` (previous account's usage limit warning), `#26628`
(plugins disappear after switching on Windows), `#22419` (SSH sessions keep stale
auth), `#39698` (remote stops working between personal and work accounts).

**Standing, stated exactly:** the *existence, count and open state* of these
issues is first-party repository fact. Their *contents* are third-party user
reports and are not vendor statements. **The direction of support runs the
opposite way from how round two wrote it**: N12b is the reachable first-party
evidence and N11 corroborates *it*, not the reverse. Together they establish that
account switching is unshipped across OpenAI's Codex surfaces. ~~and the leak
reports are direct corroboration of Q15's worry that codex state outside the
credential is not isolated by the same boundary~~ — **narrowed in round four.**
Q15 is answered and codex's state DB and logs *do* default inside `CODEX_HOME`
(N13), so these reports are not corroboration of a defect in the mechanism B1
would use. They concern the vendor's **own account switcher** — and §12.2's
enumeration of which surfaces is itself corrected: `#22419` names the **CLI**,
not only the app and IDE. What they remain is third-party evidence that the
vendor's switcher does not cleanly partition account state, on a code path B1
does not take. **What none of them establishes is entitlement** — an unshipped
feature request is not a vendor saying no.

**(c) N12c — the vendor's own forum thread on exactly this question has no
vendor answer.** `openai/codex` Discussion `#25630`, *Switch Between Accounts*, opened
2026-06-01, read via `gh api graphql`. It has **no accepted answer**, and
**every comment is from an account with `authorAssociation: NONE`** — no
maintainer, member or collaborator has replied. Users describe the CLI case
directly (one work account, one personal, on one machine) and circulate a
community tool that swaps `auth.json` files. This is a *successful read of a
place a vendor statement would appear*, returning none — the same shape as N12
and the correct way to establish a silence. It is **not** evidence of a
prohibition, and it is not treated as any.

One more item, load-bearing for §12: a community fork proposes `--auth-profile
NAME` / `CODEX_PROFILE=NAME` with a per-profile home under
`$CODEX_HOME/profiles/NAME` (linked from `#18806`). It is **unmerged**, and no
matching merged PR exists in `openai/codex`. If it or an equivalent lands
upstream, the codex half of B1 stops depending on an undocumented derivation
entirely — which is why it is now a named reopening condition rather than a
curiosity.

---

### 2.4 Verified in this task, round four — the free-read sweep

Round three's review found the same defect a third time and named the pattern
rather than the instance: **each round, this document itself ranked a read as
important and free, and each round the read was not run.** Round three's §11.2
called W10 *"the highest-value item on this table"*, §12.2 called it *"the
substantive risk and it should be first"*, and §12.3 called it *"a free source
audit nobody has run"* — and it stayed unrun. The reviewer ran it in three
public-source calls.

So the fix is not another argument. It is a sweep: **enumerate every read this
document calls free, and confirm each was actually run.**

#### 2.4.1 The sweep, as a count

§4's cost column, **as it stood entering round four**, named **twelve** items as
free. (Q16/Q17/Q18 did not exist yet — this sweep created them, and they are
also free, so a reader counting today's table gets fifteen; twelve is the
pre-sweep count.) Four of the twelve were flagged as **not reads**: Q3 is free
only once a second enrolled account exists; Q4, Q5 and Q6 were filed as free
*synthetic experiments* that need a harness, not documents.

**That reclassification needed sweeping too, and two of the four had a free
documentary half routed to the wrong class** — the same move that licensed
skipping Q2b in round one, applied here without checking for a documentary
half first:

- **Q3 — correct as filed.** Genuinely needs a second enrolled account; no
  documentary half exists.
- **Q6 — correct as filed.** The ADR already answers it in principle (*"it
  cannot in principle"*); the experiment is confirmatory only.
- **Q4 — split.** *"Is the codex derivation stable across versions?"* has a
  free documentary half: reading `AuthCredentialsStoreMode`'s `#[default]` at
  several `rust-v*` tags, the exact method §2.4.2 already establishes and uses
  seven times. **Run here (N21):** `#[default] File` is unchanged across
  `rust-v0.147.0`, `0.148.0`, `0.149.0`, `0.149.1`, `0.150.0`, `0.151.0` and the
  unreleased `0.152.0-alpha.6` — seven tags, stable at the source level across
  that whole span. This does not close Q4: the vendor documents the default
  nowhere (§7.3.3 stands), and stability so far is not a guarantee against the
  next release — which is exactly what H13 exists to stop depending on. The
  **harness** half — pinning against every version *installed on this host*,
  which W14 asks for — still cannot run: one codex is installed. That half
  stays in the experiment class.
- **Q5 — split.** *"Does Claude canonicalize the config dir beyond NFC?"* has a
  free documentary half: the derivation is readable JS in the installed
  bundle, the same method N20 uses to extract function bodies. **Run here
  (N22):** two independent occurrences in the 2.1.248 bundle —
  `var ge=Ko(()=>(s()??i(g(),".claude")).normalize("NFC"),s)` and `ok()`'s
  `r=e!==void 0?e.normalize("NFC"):ge()` — both apply exactly one transform,
  `.normalize("NFC")`, to the raw `CLAUDE_CONFIG_DIR` value or to the
  homedir-joined default. No `path.resolve`, no trailing-slash trim, no
  symlink resolution. **The documentary answer is no**: two spellings of the
  same directory that still differ after NFC — a trailing slash, an unresolved
  symlink — hash to different Keychain service names. The **harness** half —
  actually pointing `CLAUDE_CONFIG_DIR` at a symlinked or trailing-slash path
  and confirming the running CLI observes two separate Keychain entries, which
  W16 asks for — still needs a live shim run and stays in the experiment
  class.

That leaves **eight free reads**, unaffected by this correction — Q4 and Q5
were never members of that list; they move from "experiment-only" to
"experiment, with a documentary half now answered above (N21, N22)".

| # | Free read | Status entering round four | Round four | Result |
| --- | --- | --- | --- | --- |
| Q2b-a | Anthropic terms, support and product docs | **Run** (round two) | not re-run | Answered; N9/N10 stand |
| Q2b-b | OpenAI terms and help centre | **Attempted, HTTP 403** | re-attempted | **Still 403** on both URLs from this host. A failed read, not an absence — unchanged |
| Q7 | Installer write set vs. the proposed auth root | **Unrun** | **run** | **Answered adversely (N18).** The proposed root is *inside* the installer-managed dir, not outside it |
| Q10 | Does qwen have a state root? | **Unrun** | **run** | **Answered, and it overturns §5.3 (N19).** `QWEN_HOME` exists and namespaces the credential |
| Q11 | Does a legitimate setup set claude's extra namespace inputs? | **Unrun** | **run, host-scoped** | Answered for this host only (N20). Not generalisable, and labelled that way |
| Q12 | Is codex's keyring→file fallback gated on `auto`? | **Unrun** (read half) | **run** | **Answered: yes (N16).** Retires the doc-versus-binary "conflict" |
| Q14 | Codex store-selector accepted values | **Unrun** | **run** | **Answered (N17).** Four variants, not three; `Ephemeral` was unmodelled |
| Q15 | Does `sqlite_home` default inside `CODEX_HOME`? | **Unrun** | **run** | **Answered favourably (N13).** It defaults inside |

**Eight named. One was run. One was attempted and blocked. Six were unrun. All
six are now run, and the blocked one was re-attempted and is still blocked.**

The cost of the six: fourteen `gh api` reads of a public repository, two `curl`
attempts, four local file reads and one `strings` pass over an already-installed
binary. No account, no credential, no `security` call, no network write.

**Four of the six moved something.** Q15 closed the largest stated open risk
favourably; Q12 retired a conflict this document had carried as unresolved
across two rounds; Q14 found a fourth store mode nobody had modelled; Q10
overturned a per-provider verdict. Q7 answered against the design. Only Q11
returned what was expected. That ratio is the argument for the sweep.

#### 2.4.2 Method correction: read at the installed tag, not at `main`

Round four's first pass read `openai/codex` at the default branch. That is a
different program from the one installed. Every codex source claim below was
re-read at **`rust-v0.150.1`** — the tag matching `codex-cli 0.150.1`, resolved
to `0eb410ad0dd161ea323b05452f978de01cd63430` — and every environment-variable
literal was independently corroborated in the **installed** Mach-O binary with
`strings`. Where the two would have disagreed the tag wins; they do not disagree.

Recorded because "read the vendor's source" is not one method. Reading `main`
and reporting it as evidence about the installed CLI is the same widening this
document has now diagnosed four times.

---

**N13 — `sqlite_home` defaults *inside* `CODEX_HOME`. Q15 is answered, and it
answers in B1-codex's favour.**

`codex-rs/config/src/config_toml.rs:329-331`, the vendor's own doc comment:

> ```rust
> /// Directory where Codex stores the SQLite state DB.
> /// Defaults to `$CODEX_SQLITE_HOME` when set. Otherwise uses `$CODEX_HOME`.
> pub sqlite_home: Option<AbsolutePathBuf>,
> ```

`codex-rs/core/src/config/mod.rs:3918-3923`, the production resolution — not a
test helper, not a default-impl:

> ```rust
> let sqlite_home = cfg.sqlite_home.as_ref().cloned()
>     .or(sqlite_home_env)
>     .unwrap_or_else(|| codex_home.clone());
> ```

`sqlite_home_env` comes from `resolve_sqlite_home_env` (`mod.rs:242-252`), which
reads `codex_state::SQLITE_HOME_ENV` — `"CODEX_SQLITE_HOME"`, declared at
`codex-rs/state/src/lib.rs:106-107`. **Precedence: `config.toml`'s `sqlite_home`
› `$CODEX_SQLITE_HOME` › `$CODEX_HOME`.**

`log_dir` resolves at `mod.rs:3906-3910` to `codex_home.join("log")`, which
confirms ledger 29's documented half from source rather than from prose.

**What this settles.** Two credential-isolated codex profiles do **not** share a
state database by default. §12.2 item 1 stated the fork itself — *"If it defaults
inside, the largest open objection goes away"* — and it defaults inside. The
cross-account state-sharing risk this ADR has called *the substantive one* since
round three is **closed, favourably, and it was three reads away the whole time.**

**What it does not settle.** Entitlement is still unestablished and Q2 is still
unrun, so **B1-codex stays HELD**. A closed objection is not permission.

**And note the third instance of the same escape hatch.** `config.toml`'s
`sqlite_home` **beats** the environment variable, exactly as
`cli_auth_credentials_store` can be set explicitly (H13) and exactly as claude's
namespace is pinned by the config dir rather than by a derivation. A B1 profile
that owns its own `CODEX_HOME/config.toml` can pin its state DB the same way it
pins its store. Three separate hazards, one shape of answer.

---

**N14 — `CODEX_SQLITE_HOME` is a second codex ambient input, and B0 had no
prerequisite for it.**

The variable appears **nowhere** in this ADR's previous three rounds, in
`bk6owf`, in the keychain-custody research, in `README.md` or in `LOGBOOK.md`.
§5.2 said *"Namespace inputs: **One** (N5)"* and §5.2's B0 verdict said
*"**Go**, and cleaner than claude: one input."*

N5 is not wrong — it is *narrow*. `Codex Auth` occurs exactly once with no
environment-derived affix, which is a precise and true statement about the
**credential namespace**. It was then used at the wider scope of **state
isolation and B0's prerequisite list**, where it does not hold. That is the same
precise-where-derived / wider-where-used shape §12.4 tabulates, committed on the
axis nobody grepped.

**Why it blocks B0 rather than B1.** B0's entire premise is deterministic
child-environment composition. With an ambient `CODEX_SQLITE_HOME` set, a
B0-composed launch that sets a per-profile `CODEX_HOME` puts the state DB
**outside** that root — the exact cross-account state sharing AC6 forbids, on
the option this ADR says *adopt now* and calls *"ready to schedule"*. §11.2's
**"W13 is the only one B0 depends on"** was false in a checkable way.

Codex is aware of the collision: the installed binary carries
`` `CODEX_SQLITE_HOME` is overridden by an exact requirement `` and
``Environment value for `$CODEX_SQLITE_HOME` is overridden by the required
`sqlite_home`…`` from `core/src/config/requirements.rs`. The vendor treats
ambient-versus-required as a case worth warning about. So should B0.

---

**N15 — and there are six more. Codex's ambient input count is eight, not one.**

The method that missed `CODEX_SQLITE_HOME` was grepping for the credential
namespace literal instead of enumerating what codex reads from its environment.
Applying the second method at `rust-v0.150.1`, `codex-rs/login/src/auth/manager.rs`
declares:

> ```rust
> pub const REFRESH_TOKEN_URL_OVERRIDE_ENV_VAR: &str = "CODEX_REFRESH_TOKEN_URL_OVERRIDE"; // :199
> pub const REVOKE_TOKEN_URL_OVERRIDE_ENV_VAR:  &str = "CODEX_REVOKE_TOKEN_URL_OVERRIDE";  // :200
> pub const CLIENT_ID_OVERRIDE_ENV_VAR:         &str = "CODEX_APP_SERVER_LOGIN_CLIENT_ID"; // :201
> pub const OPENAI_API_KEY_ENV_VAR:             &str = "OPENAI_API_KEY";                   // :910
> pub const CODEX_API_KEY_ENV_VAR:              &str = "CODEX_API_KEY";                    // :911
> pub const CODEX_ACCESS_TOKEN_ENV_VAR:         &str = "CODEX_ACCESS_TOKEN";               // :912
> ```

All six are present in the installed 0.150.1 binary. With `CODEX_HOME` and
`CODEX_SQLITE_HOME` that is **eight** ambient inputs that relocate codex's state
root, supply its credential, or redirect its OAuth endpoints.

Two of them are worse than the count suggests.

**(a) The credential env vars *outrank the composed profile's own credential*.**
`manager.rs:1456-1462`, with the vendor's own comment:

> ```rust
> // API key via env var takes precedence over any other auth method.
> if enable_codex_api_key_env && auth_mode_is_allowed(…, AuthMode::ApiKey)
>     && let Some(api_key) = read_codex_api_key_from_env()
> { return Ok(Some(CodexAuth::from_api_key(api_key.as_str()))); }
> ```

`CODEX_ACCESS_TOKEN` follows the same shape at `:1492-1497`, after the ephemeral
store and before any persisted credential. An ambient key therefore silently
defeats profile selection: the launch would run under the *wrong identity while
reporting the right one*. That is not a leak of one profile into another; it is
the identity claim being false.

**(b) The refresh-endpoint override has no allowlist.** `manager.rs:1717-1720`:

> ```rust
> fn refresh_token_endpoint() -> String {
>     std::env::var(REFRESH_TOKEN_URL_OVERRIDE_ENV_VAR)
>         .unwrap_or_else(|_| REFRESH_TOKEN_URL.to_string())
> }
> ```

Any value is accepted. The refresh POST carries the refresh token, so an ambient
`CODEX_REFRESH_TOKEN_URL_OVERRIDE` redirects a live credential to an arbitrary
host. `revoke.rs:135-139` does the same for revocation and *falls back to the
refresh override* when its own is unset.

**Compare claude, and note the direction.** N1 recorded that
`CLAUDE_CODE_CUSTOM_OAUTH_URL` is checked against `ALLOWED_OAUTH_BASE_URLS` and
**throws** on anything else. This ADR treated that as claude's larger attack
surface and codex's single literal as the cleaner story. On this axis the
comparison inverts: claude validates its OAuth override and codex does not.
§5.2's *"cleaner than claude: one input"* is withdrawn.

**Standing.** Source at the installed tag plus literal corroboration in the
shipped binary. What is **not** established: whether `enable_codex_api_key_env`
is on by default in a subscription login, and whether any of the three override
variables is gated elsewhere in the call graph. Those are read as constants and
call sites, not as a whole-program reachability proof — recorded as **Q16**
rather than asserted.

---

**N16 — codex's keyring→file fallback *is* gated on `Auto`. Q12's read half is
answered, and it retires a conflict this ADR carried for two rounds.**

`codex-rs/login/src/auth/storage.rs:511-528` at `rust-v0.150.1`:

> ```rust
> match mode {
>     AuthCredentialsStoreMode::File      => Arc::new(FileAuthStorage::new(codex_home)),
>     AuthCredentialsStoreMode::Keyring   => create_keyring_auth_storage(codex_home, …),
>     AuthCredentialsStoreMode::Auto      => Arc::new(AutoAuthStorage::new(codex_home, …)),
>     AuthCredentialsStoreMode::Ephemeral => Arc::new(EphemeralAuthStorage::new(codex_home)),
> }
> ```

Both fallback warnings — *"failed to load CLI auth from keyring, falling back to
file storage"* and *"failed to save auth to keyring, falling back to file
storage"* — live in `impl AuthStorageBackend for AutoAuthStorage`
(`:431-449`), and nowhere else. `create_keyring_auth_storage` (`:531-545`)
dispatches to `DirectKeyringAuthStorage` or `SecretsKeyringAuthStorage` with **no
file leg at all**.

**So the docs and the binary never disagreed.** N4 found the fallback strings by
byte inspection and this ADR concluded *"the two sources disagree, so `keyring`
cannot be relied on to mean 'never plaintext'"*. They agree: the vendor
documents a fallback for `auto`, and the strings N4 found belong to `auto`'s
implementation. N12a's conflict is **withdrawn**; `keyring` does mean never
plaintext, on the load and the save leg both.

This is worth stating plainly because the ADR reached the *cautious* conclusion
from string literals and was right to label it *"a narrowing, not an answer"* —
and then carried the narrowing as if it were adverse for two rounds. **A
correctly-labelled unknown still costs something while it stays unread.**

The induced-failure half of Q12 remains unrun and still needs a synthetic
profile; it is no longer interesting, because the control flow answers the
question the experiment was for.

---

**N17 — the codex store vocabulary has four values, not three, and the default
is `File` in first-party source. Q14 answered.**

`codex-rs/config/src/types.rs:107-118`:

> ```rust
> pub enum AuthCredentialsStoreMode {
>     #[default]
>     /// Persist credentials in CODEX_HOME/auth.json.
>     File,
>     /// Persist credentials in the keyring. Fail if unavailable.
>     Keyring,
>     /// Use keyring when available; otherwise, fall back to a file in CODEX_HOME.
>     Auto,
>     /// Store credentials in memory only for the current process.
>     Ephemeral,
> }
> ```

Three consequences.

1. **`Ephemeral` is a fourth mode neither `bk6owf` §3.3 nor this ADR modelled.**
   It is not a file and not the keyring — it is a process-lifetime in-memory map
   (`storage.rs`, `EPHEMERAL_AUTH_STORE`), and `manager.rs:1465-1471` consults it
   **before any persisted credential**. Any custody classifier that is total over
   `file | keyring | auto` is not total.
2. **The "encrypted auth storage" of N4/Q14 is `AuthKeyringBackendKind`**, a
   *separate* axis from the store mode: `Direct` (payload straight into the OS
   keyring) or `Secrets` (*"local encrypted secrets file, with the file key in
   the OS keyring"*), defaulting to `Secrets` on Windows and `Direct` elsewhere.
   It is not selectable through `cli_auth_credentials_store`. Q14 asked exactly
   that and the answer is **no**.
3. **The default is `File`, established in first-party source** by `#[default]`
   at the exact installed tag — not only by byte inspection of one build.

Point 3 narrows §7.3.3 without removing it. The default is now known from the
vendor's own source rather than from a fixed-defaults byte window, which is
better evidence; it is still documented **nowhere operator-facing** (ledger 30
stands), and a `#[default]` attribute is exactly the kind of constant that moves
between releases without a doc change. The cost is smaller and its escape is
unchanged: **H13** — set the key explicitly per profile.

---

**N18 — Q7 answers against the design: the proposed auth root is *inside* the
installer-managed config directory, on all three platforms.**

`scripts/setup.sh:69` sets `CONFIG_DIR="$HOME/Library/Application Support/agents-infra"`
on macOS and `${XDG_CONFIG_HOME:-$HOME/.config}/agents-infra` otherwise;
`scripts/setup.ps1:13` uses `%APPDATA%\agents-infra`;
`tools/agents-infra/internal/infra/source_dir.go:387-404` resolves the same three
paths in Go. `write_install_state` (`setup.sh:208-211`) does
`mkdir -p "$CONFIG_DIR"` and writes `$CONFIG_DIR/install.json`.

`bk6owf`'s proposed credential root is
`~/Library/Application Support/agents-infra/auth/` — a **child** of that
directory. Q7 asked whether it is *"genuinely outside everything `setup.sh`
manages"*. **It is not.** The honest statement is narrower than either "safe" or
"broken":

- **File-level:** disjoint today. The installer's only write inside `CONFIG_DIR`
  is `install.json`; nothing recursive, nothing that removes the directory.
- **Directory-level:** nested. Any future uninstall, prune or repair that treats
  `CONFIG_DIR` as owned would take enrolled credentials with it, and it would be
  a *reasonable* thing for an installer to do to its own config directory.

This does not reopen A (which is rejected for reasons untouched by path choice)
and it does not gate B1. It changes what the B1 prerequisite must be: Q7's
proposed test — *"assert disjointness"* — cannot pass as written. Replace it with
an assertion that the installer's write set never grows a **recursive** operation
on `CONFIG_DIR`, or move the auth root out from under it. Recorded as **H14**.

---

**N19 — qwen has a state root. §5.3's "not modellable" is refuted, and the cause
was our own plugin declaration, not a vendor limitation.**

`QwenLM/qwen-code`, `packages/core/src/config/storage.ts`:

> ```ts
> static getGlobalQwenDir(): string {            // :193-203
>   const envDir = process.env['QWEN_HOME'];
>   if (envDir) return Storage.resolvePath(envDir);
>   const homeDir = os.homedir();
>   if (!homeDir) return path.join(os.tmpdir(), '.qwen');
>   return path.join(homeDir, QWEN_DIR);
> }
> static getOAuthCredsPath(): string {           // :640-642
>   return path.join(Storage.getGlobalQwenDir(), OAUTH_FILE);   // 'oauth_creds.json'
> }
> ```

`QWEN_HOME` is a state root **and** it namespaces the credential, because the
OAuth credentials file is resolved through it. A second ambient input,
`QWEN_RUNTIME_DIR` (`:172-190`), relocates runtime output with its own
precedence chain.

**So qwen is the same shape as codex-on-`file`:** an env-selected state root with
a plaintext credential file inside it. That is B0-modellable immediately and
B1-shaped in principle. `HomeEnvVar: ""` at `skill-agents-management`
`pkg/agentic/systems/qwen/qwen.go:121` is a **repository defect**, not a fact
about qwen.

**Standing, stated exactly, because this is the read that would be easiest to
over-spend.** This is first-party source at `main`. **qwen is not installed on
this host**, so unlike every claude and codex claim in this document there is no
shipped-build corroboration, and no version is pinned. The finding is strong
enough to retire *"there is no state root"* and to schedule the plugin fix; it is
**not** strong enough to promote qwen into B1, and §5.3 says so. Verification
against an installed build is **Q18**.

**And a method note that is the point of this whole section.**
`gh api search/code -f q='repo:QwenLM/qwen-code QWEN_CODE_HOME OR QWEN_HOME'`
returned **`total_count: 0`**. The constant is right there in
`packages/core/src/config/storage.ts`. It was found by reading the file that the
*credential-path* query pointed at, not by the name query. **A zero from
GitHub code search is a failed or partial read, not an absence** — which is this
ADR's own standard for `curl` 403s, applied to a tool whose zero looks like a
result. Anything in this document resting on a code-search zero would need
re-running; nothing does, but ledger 23 is now labelled accordingly.

---

**N20 — claude's ambient-input class is far larger than its namespace-derivation
inputs, and one member of it is a second, vendor-shipped, file-based
named-profile credential store.**

The same enumerate-the-environment method, applied to Claude Code 2.1.248,
extracts **174** distinct `process.env.CLAUDE_*` / `process.env.ANTHROPIC_*`
read sites from the installed bundle. That is a read-site extraction, not a bare
literal count: each is a place the program reads the ambient environment.

N1's "three inputs" remains correct and remains narrow, in exactly N5's way: it
is the input set of the **Keychain service-name derivation**. The set that
matters to **B0's determinism** is much larger, and includes at least:

| Class | Members |
| --- | --- |
| State roots | `CLAUDE_CONFIG_DIR`, `ANTHROPIC_CONFIG_DIR`, `CLAUDE_SECURESTORAGE_CONFIG_DIR`, `XDG_CONFIG_HOME`, `HOME`, `CLAUDE_CODE_TMPDIR`, `CLAUDE_TMPDIR`, `CLAUDE_JOB_DIR`, `CLAUDE_MEMORY_STORES` |
| Ambient credentials | `ANTHROPIC_API_KEY`, `ANTHROPIC_AUTH_TOKEN`, `CLAUDE_CODE_OAUTH_TOKEN`, `CLAUDE_CODE_OAUTH_REFRESH_TOKEN`, `CLAUDE_CODE_SESSION_ACCESS_TOKEN`, `CLAUDE_TRUSTED_DEVICE_TOKEN`, `CLAUDE_CODE_API_KEY_FILE_DESCRIPTOR`, `CLAUDE_BG_AUTH_SNAPSHOT_PATH`, `CLAUDE_SESSION_INGRESS_TOKEN_FILE` |
| Endpoint / client redirection | `ANTHROPIC_BASE_URL`, `CLAUDE_CODE_API_BASE_URL`, `CLAUDE_CODE_CUSTOM_OAUTH_URL`, `CLAUDE_CODE_OAUTH_CLIENT_ID`, `CLAUDE_LOCAL_OAUTH_API_BASE`, `CLAUDE_LOCAL_OAUTH_APPS_BASE`, `CLAUDE_LOCAL_OAUTH_CONSOLE_BASE` |
| Identity selection | `ANTHROPIC_PROFILE`, `ANTHROPIC_ORGANIZATION_ID`, `ANTHROPIC_FEDERATION_RULE_ID`, `ANTHROPIC_WORKSPACE_ID`, `CLAUDE_CODE_ACCOUNT_UUID` |

**And the identity-selection row is not a list of knobs. It is a mechanism.**
2.1.248 ships a complete file-based named-profile credential store, internally
labelled *"WIF profile"*:

> ```js
> function C(){ let e=process.env.ANTHROPIC_CONFIG_DIR?.trim();
>   if(e) return {dir:e,space:"userNamed"};
>   let t=process.env.XDG_CONFIG_HOME?.trim();
>   if(t) return {dir:i(t,"anthropic"),space:"home"};
>   let r=process.env.HOME?.trim();
>   return r?{dir:i(r,".config","anthropic"),space:"home"}:null }
> function T(){ return process.env.ANTHROPIC_PROFILE?.trim() }
> function L_t(e){ return f(i(e,"active_config"))?.trim()||"default" }
> function O(e,t){ … JSON.parse(f(i(e,"configs",`${t}.json`))) … o?.authentication?.type … }
> function z(e,t,r){ return r?.authentication?.credentials_path ?? i(e,"credentials",`${t}.json`) }
> ```

- Root: `ANTHROPIC_CONFIG_DIR` › `$XDG_CONFIG_HOME/anthropic` › `$HOME/.config/anthropic`
- Profile: `ANTHROPIC_PROFILE` (explicit) or the contents of `<root>/active_config` (implicit), defaulting to `"default"`
- Per-profile config `<root>/configs/<name>.json`; credentials at
  `authentication.credentials_path`, defaulting to `<root>/credentials/<name>.json`
- Accepted `authentication.type`: **`user_oauth`** and `oidc_federation`
- Precedence: explicit profile › "env-quad"
  (`ANTHROPIC_FEDERATION_RULE_ID` **and** `ANTHROPIC_ORGANIZATION_ID` both set)
  › implicit profile
- Operator-visible status string: `` credentials-file · <authType> · profile <name> ``

**Why this matters to §5.1 and §12.1.** B1-claude's whole design rests on
namespacing the *Keychain* by config dir. Here is a **second** vendor-shipped
credential path — file-based, explicitly profile-named, selected by an
environment variable, accepting a `user_oauth` credential — that this document
did not know existed. If a consumer subscription login can be materialised as a
`user_oauth` profile, B1-claude may have a cleaner mechanism than the one it is
holding an experiment for.

**What is established, and what is not.** Established: the mechanism exists in
the installed build, is read from the ambient environment in production code,
and accepts `user_oauth`. **Not established:** how such a profile is *created* —
no CLI verb that writes `configs/<name>.json` was found, the subsystem's internal
name points at workload-identity federation, and `~/.config/anthropic` does not
exist on this host. **Nothing here is promoted to a verdict.** It is recorded as
**Q17**, and as W23 in §11.2, at exactly the standing the evidence carries: a
mechanism sighted in shipped code, with its enrolment path unread.

**N20a — Q11, answered for this host and only this host.** None of
`CLAUDE_CONFIG_DIR`, `CLAUDE_SECURESTORAGE_CONFIG_DIR`,
`CLAUDE_CODE_CUSTOM_OAUTH_URL`, `ANTHROPIC_CONFIG_DIR`, `ANTHROPIC_PROFILE`,
`XDG_CONFIG_HOME`, the env-quad pair, `ANTHROPIC_API_KEY`, `ANTHROPIC_AUTH_TOKEN`,
`CLAUDE_CODE_OAUTH_TOKEN`, `CODEX_HOME`, `CODEX_SQLITE_HOME`, `CODEX_API_KEY`,
`CODEX_ACCESS_TOKEN`, `OPENAI_API_KEY` or the three codex URL/client overrides is
set in this process, and none is mentioned in `~/.zshrc`, `~/.zshenv`,
`~/.zprofile`, `~/.bash_profile` or `~/.profile`. `~/.config/anthropic` is
absent. **Presence was tested by name; no value was read or printed.**

This is one host and one shell context, and this process is itself a composed
child environment, so it is weak evidence about what a *legitimate operator
setup* does elsewhere. It supports "the refusal will not fire spuriously here".
It does not support "no legitimate setup sets these". W13 stays open at that
reduced scope rather than being closed.

**N21 — codex's store-selector default is stable across seven tags spanning
`0.147.0` to an unreleased alpha.** `codex-rs/config/src/types.rs`'s
`AuthCredentialsStoreMode` at `rust-v0.147.0`, `0.148.0`, `0.149.0`, `0.149.1`,
`0.150.0`, `0.151.0` and `0.152.0-alpha.6` all carry `#[default]` on `File`, at
the same line range in every tag. This is the free documentary half of Q4
(§2.4.1): it establishes source-level stability across the sampled span, not a
guarantee for the next release, and it does not remove the version-gate cost
§7.3.3 prices — H13 still exists because a doc comment is not a contract.

**N22 — claude applies exactly one transform to the config-dir value, and it
is not enough to make two spellings collide.** Two independent sites in the
installed 2.1.248 bundle compute the Keychain namespace input:
`var ge=Ko(()=>(s()??i(g(),".claude")).normalize("NFC"),s)` (one chunk) and
`ok()`'s `r=e!==void 0?e.normalize("NFC"):ge()` (another). Both take
`process.env.CLAUDE_CONFIG_DIR`, or the homedir-joined default, and apply
`.normalize("NFC")` and nothing else — no `path.resolve`, no trailing-slash
trim, no symlink dereference. This is the free documentary half of Q5
(§2.4.1): a trailing-slash spelling or a symlink pointing at the same
directory is NFC-equal to the canonical spelling only if it was already
byte-identical before normalization, so such a pair produces two different
service names. Whether that pair actually reaches this code path from a live
shell is the harness half W16 still asks for.

---

## 3. Assumptions

Stated so a later reader can attack them rather than inherit them.

1. **The target is macOS.** Every derivation, custody model and hazard here is
   macOS-specific. Linux and Windows are out of scope and would need their own
   evidence; `CLAUDE_CONFIG_DIR` is documented as a credential-file relocation
   there, which is a *different* mechanism from the one this ADR rests on.
2. **The threat model excludes an adversary with local write access to
   `~/Library/Application Support/agents-infra/`.** `bk6owf` C2 is a checked
   gate that a consistent two-field rewrite passes; it stops accident and
   migration drift, not an attacker who already has the write. Q13 asks whether
   this assumption should hold. This ADR adopts it, and it is the assumption
   most worth revisiting first.
3. **The vendor CLIs remain the sole writers of their own credentials.**
   Everything in B rests on agents-infra choosing a directory and then keeping
   its hands off. The moment that stops being true, the analysis in §8 restarts.
4. **One human operator, one macOS user account.** Claude's Keychain *account*
   field is `$USER` (`1g880w`); namespacing is carried entirely by the service
   name. Multi-user hosts are unanalysed.
5. **Observed versions are representative.** Claude Code 2.1.248 and Codex
   0.150.1 are what is installed. Every "current source" claim expires at the
   next upgrade — which, per §7, is a matter of days.

---

## 4. Unknowns

Carried forward from the inputs, plus two added here. None is guessed at
anywhere in this document.

| # | Unknown | Blocks | What settles it | Cost |
| --- | --- | --- | --- | --- |
| **Q1** | *(claude, narrowed)* Does a second concurrently-enrolled Anthropic login leave the first working server-side over time? Anthropic **documents** the side-by-side workflow (N9), so the software question is settled; server-side durability is not. | **B1-claude**, and it is the only remaining gate | Enrol a second Anthropic account under its own state root, leave the first live, verify both for 24 h. Namespaces are provably disjoint (N8, documented), so the live account is not at risk. | A second real account the operator is already entitled to. Not answerable synthetically. |
| **Q2** | *(codex)* Same, for OpenAI/ChatGPT under `CODEX_HOME`. **Still unanswered, and still not blocking** — but for a corrected reason. Round two called it "largely mooted" because "the vendor has already answered in the negative"; the vendor has not answered at all for the CLI surface (§2.3, N11 demoted). It is non-blocking because the codex no-go rests on §7.3.1, not on entitlement. | nothing — the verdict does not wait on it | Same shape, disposable ChatGPT account. | Same. Not worth spending, because the answer would not move the verdict either way. |
| **Q2b-a** | *(anthropic, terms)* **ANSWERED — free documentary read, performed in §2.3.** The Consumer Terms restrict sharing an account with other people and place **no limit on how many accounts one person holds** (N10); the product docs recommend running multiple accounts side by side (N9). | nothing, now | Done. Reclassified from "a decision" to a free source read, which is what it always was. | Free. The misclassification is why it went unread. |
| **Q2b-b** | *(openai, terms)* Same question. **Partly answered for ChatGPT web, unestablished for the Codex CLI, partly unreadable from here.** The account-switching material is *permissive* for ChatGPT web; its negative clause is scoped to **Codex desktop** and the native mobile apps and **does not reach the CLI** (N11, corrected in round three). `help.openai.com` and `openai.com/policies` return **HTTP 403 to every request from this host** — a failed read, not an absence. | nothing — the codex no-go rests on §7.3.1, not on entitlement | A read from a host that is not 403-blocked, or a human opening the pages in a browser (W18). | Free from an unblocked network. |
| **Q2c** | *(both, residual)* May one human hold two concurrently **billed consumer subscriptions** — two Pro, or Pro plus Max? **Nothing published addresses this**, on either vendor; checked in §2.3. | only the buy-a-second-subscription variant of B1 | Vendor sales or support, or an operator who already holds two. Mostly moot: the documented scenario is work + personal, i.e. subscription + org seat or subscription + Console, both explicitly supported (N10). | A purchasing decision the operator owns. |
| Q3 | Claude's actual `revoke_capability`. `logout` currently can never tell an operator their server-side session ended. | B1 `logout` semantics | Disposable second account: run vendor logout, observe usability from another namespace. | Free once Q1's account exists. |
| **Q4** | **PARTLY ANSWERED (N21).** The documentary half: `#[default] File` is unchanged across seven `rust-v*` tags, `0.147.0` through the unreleased `0.152.0-alpha.6`. Source-stable so far; not a guarantee for the next release. | B1 gate economics | The harness half: run the codex pin test with all negative variants against every *installed* codex version — still needs more than the one codex this host has. | Documentary half free, done. Harness half free, synthetic, unrun. |
| **Q5** | **PARTLY ANSWERED (N22).** The documentary half: two independent occurrences in the 2.1.248 bundle apply exactly `.normalize("NFC")` to the config-dir value, nothing else — no `resolve`, no trailing-slash trim, no symlink dereference. So a trailing-slash or symlinked spelling that survives NFC unchanged does **not** collide with the canonical spelling. | B1 pin precision | The harness half: shim run with symlinked and trailing-slash spellings of one synthetic dir, confirming the running CLI actually produces two Keychain entries. | Documentary half free, done. Harness half free, synthetic, unrun. |
| Q6 | Does Claude's advisory `.storage-write` lock protect against a writer that does not take it? It cannot in principle. | B1 concurrency claim | Two synthetic profiles, one lock-taking writer and one not, same synthetic item. | Free, synthetic. |
| **Q7** | **ANSWERED IN ROUND FOUR, adversely (N18).** The proposed auth root is a **child** of the installer-managed config dir on all three platforms (`setup.sh:69`, `setup.ps1:13`, `source_dir.go:387-404`), not outside it. File-level disjoint today — the installer's only write inside it is `install.json` — but directory-level nested, so any future recursive uninstall or prune would take enrolled credentials with it. | B1 root placement | Done. The originally-proposed test ("assert disjointness") cannot pass as written; replaced by H14 — assert the installer's write set never grows a **recursive** operation on `CONFIG_DIR`, or move the auth root out from under it. | Free. Was unrun for three rounds. |
| Q8 | Are profile records machine-scoped, or may a shared repo config pin an alias that does not exist on another machine? | B1 UX | Operator/product decision. | Decision. |
| Q9 | Is a `degraded:plaintext` profile repairable in place, or must it be re-enrolled? | B1 recovery | Synthetic profile, induce a non-transient store failure, remove the cause, observe. Needs a credential-shaped payload. | Blocked on Q1. |
| **Q10** | **ANSWERED IN ROUND FOUR, and it refutes §5.3 (N19).** qwen has `QWEN_HOME`, and it namespaces the credential: `getOAuthCredsPath()` = `$QWEN_HOME/oauth_creds.json` (`packages/core/src/config/storage.ts:193-203, 640-642`). A second input, `QWEN_RUNTIME_DIR`, relocates runtime output. `HomeEnvVar: ""` is **our** plugin defect, not a vendor limitation. | qwen, epic AC8 | Done, at `main`. **No shipped-build corroboration** — qwen is not installed on this host; see Q18. | Free. Was unrun for three rounds. |
| **Q11** | **PARTLY ANSWERED IN ROUND FOUR, host-scoped only (N20a).** None of the claude or codex namespace, credential or endpoint-override variables is set in this process or named in this host's shell profiles, and `~/.config/anthropic` is absent. Presence tested by name; no value read. | B0 and B1 | Supports "the refusal will not fire spuriously here". Does **not** support "no legitimate setup sets these" — one host, one shell, and this process is itself a composed child env. Stays open at reduced scope. | Free. |
| **Q12** | **ANSWERED IN ROUND FOUR (N16): the fallback is gated on `Auto`.** Both fallback legs live in `impl AuthStorageBackend for AutoAuthStorage` and nowhere else; `Keyring` dispatches to `create_keyring_auth_storage` with **no file leg** (`storage.rs:431-449, 511-545` at `rust-v0.150.1`). **N12a's doc-versus-binary conflict is withdrawn** — the two sources never disagreed. | B1 hazard scope | Done. The induced-failure half is now uninteresting: the control flow answers what the experiment was for. | Free. The read half was unrun for two rounds. |
| Q13 | Is the profile record worth cryptographically sealing? | B1 threat model | A threat-model decision: does the model include local write access to the auth root? | Decision. |
| **Q14** | **ANSWERED IN ROUND FOUR (N17).** `AuthCredentialsStoreMode` has **four** variants — `File` (`#[default]`), `Keyring`, `Auto`, **`Ephemeral`** (`config/src/types.rs:107-118`). "Encrypted auth storage" is `AuthKeyringBackendKind::{Direct,Secrets}`, a **separate axis**, `Secrets` on Windows and `Direct` elsewhere — so the answer to "selectable through `cli_auth_credentials_store`" is **no**. `Ephemeral` was unmodelled by `bk6owf` §3.3 *and* by this ADR, and it is consulted **before** any persisted credential. | B1 store modelling | Done. Any custody classifier total over `file \| keyring \| auto` is not total. | Free. Was unrun for two rounds. |
| **Q15** | **ANSWERED IN ROUND FOUR, favourably (N13).** `sqlite_home` **defaults inside `CODEX_HOME`**. Precedence: `config.toml`'s `sqlite_home` › `$CODEX_SQLITE_HOME` › `$CODEX_HOME` (`config_toml.rs:329-331`, `core/src/config/mod.rs:3918-3923`, `state/src/lib.rs:106-107`, all at `rust-v0.150.1`). `log_dir` confirmed from source at `mod.rs:3906-3910`. **Two credential-isolated codex profiles do not share a state DB by default.** | B1 isolation claim | Done. This closes the largest stated open objection to B1-codex. It does **not** move the verdict: entitlement is still unestablished and Q2 is still unrun. | Free. It was ranked first for a full round and stayed unrun. |
| **Q16** | **New (N15).** Are codex's six auth-affecting environment overrides gated anywhere in the call graph, and is `enable_codex_api_key_env` on by default for a subscription login? `CODEX_API_KEY` is documented in source as taking *"precedence over any other auth method"*, and `CODEX_REFRESH_TOKEN_URL_OVERRIDE` is accepted with **no allowlist** — unlike claude's `ALLOWED_OAUTH_BASE_URLS`, which throws. | **B0 determinism**, B1 isolation | Whole-program reachability read, or a synthetic launch with the variable set against a throwaway `CODEX_HOME`. Round four read the constants and the call sites, not the whole graph, and says so. | Free for the read; the launch half needs a synthetic profile. |
| **Q17** | **New (N20).** Claude Code 2.1.248 ships a second, file-based, **named-profile** credential store — `ANTHROPIC_CONFIG_DIR`/`ANTHROPIC_PROFILE` → `configs/<name>.json` + `credentials/<name>.json`, accepting `user_oauth`. Can a consumer subscription login be materialised as such a profile, and by which verb? No writing verb was found; the subsystem is internally named for workload-identity federation. | **B1-claude mechanism** — potentially a cleaner one than the Keychain route, potentially enterprise-only | Source audit of the enrolment path in the installed bundle, plus the vendor's own documentation for `ANTHROPIC_PROFILE`. | Free. W23. |
| **Q18** | **New (N19).** Does the *installed* qwen build carry `QWEN_HOME`, and at what version? Round four read `QwenLM/qwen-code` at `main`; qwen is **not installed on this host**, so unlike every claude and codex claim here there is no shipped-build corroboration and no version pin. | Promoting qwen past B0 | Install or obtain a build and repeat the `strings`/source cross-check this ADR uses for the other two runtimes. | Free once a build exists. |

**Recount, after the documentary read.** The previous round counted three
product-deciding unknowns (Q1, Q2, Q2b) and said all three needed something this
machine did not have. That was true of one of them. Q2b was a **free documentary
read misfiled as a decision** — its cost column read "a decision, and it is not
the agent's to make", when reading published terms and product documentation is
neither a decision nor an experiment. Eight other rows in this table were filed
correctly as free source audits; this one was not, and that misfiling is the
mechanism by which the single load-bearing input went unread while substantial
effort went into binary inspection the inputs had already largely settled. Only
the *judgement* about whether to rely on what the documents say was ever the
operator's. The read is now performed and recorded in §11.2 as **W17, complete**.

**Recount again, after round four's sweep.** The table now holds twenty
entries. **Seven are answered** (Q2b-a, Q7, Q10, Q12, Q14, Q15, and Q11 at
reduced scope); **two are partly answered by a documentary half, with only a
harness half remaining** (Q4, Q5 — N21, N22); three are decisions (Q8, Q13,
Q2c); one is a failed read that stays failed (Q2b-b); one is a free synthetic
experiment nobody needs yet (Q6); three are new and free (Q16, Q17, Q18); Q3
and Q9 wait on an account; and exactly **one — Q1, on Anthropic only — still
gates a live path**.

**Round four changes which unknowns matter, on both sides, and the change is
not the one round three predicted.** Round three ranked Q15 first and called it
*"the one open item that could still close B1-codex on a demonstrated ground"*.
It was three source reads away and it **closed favourably**: `sqlite_home`
defaults inside `CODEX_HOME`, so credential-isolated codex profiles do not share
a state DB. Q12 answered too, and *in codex's favour* — the keyring path has no
plaintext fallback at all. Neither moves the verdict, because entitlement is
unestablished and **Q2** is still unrun; Q2 is now the only thing left on the
codex side that could decide anything, and it needs an account.

**What replaced them is heavier than what they were.** Q15's read surfaced
`CODEX_SQLITE_HOME` (N14) and, by the enumeration method that found it, six more
codex environment inputs (N15) and a whole second claude credential store (N20).
Those are **B0** problems, not B1 problems — B0 is the option this ADR says
adopt now. The unknown carrying the most weight is no longer on the held path.
It is **Q16**, on the adopted one.

---

## 5. Per-provider feasibility verdict

### 5.1 Claude Code 2.1.248 — macOS

| Dimension | Finding | Standing |
| --- | --- | --- |
| Default custody | macOS Keychain, service `Claude Code-credentials`, account `$USER`. `~/.claude/.credentials.json` absent. | Proven (`1g880w`) |
| Namespace mechanism | Config-dir hash in the **service** name. Works. | Proven (`1g880w`), refutes `3moaky` item 4 |
| Namespace inputs | **Three** env variables feed the Keychain **service-name derivation** (N1). That is the derivation's input set, and it is *not* B0's input set: **174** distinct `process.env.CLAUDE_*`/`ANTHROPIC_*` read sites exist in the installed bundle, including nine that relocate a state root, nine that supply an ambient credential and seven that redirect an endpoint (N20). | N1 verified across 5 builds; N20 by read-site extraction, 2.1.248 |
| Second credential store | **Yes, and it was unknown to rounds one to three (N20).** A file-based **named-profile** store: `ANTHROPIC_CONFIG_DIR` › `$XDG_CONFIG_HOME/anthropic` › `$HOME/.config/anthropic`, profile from `ANTHROPIC_PROFILE` or `active_config`, config at `configs/<name>.json`, credential at `credentials/<name>.json`, accepted `authentication.type` = `user_oauth` \| `oidc_federation`. Internally "WIF profile". | Production code in the installed bundle. **How a profile is created is unread (Q17)** — no writing verb found; not promoted to a verdict |
| Default namespace addressing | Reachable only with `CLAUDE_CONFIG_DIR` unset/empty (N2). | Verified here + `1g880w` |
| Documented? | **Yes**, and this is a reversal (N8/N9). `code.claude.com/docs/en/authentication`: Claude Code *"keys the macOS Keychain entry to that directory too, so a session with a different `CLAUDE_CONFIG_DIR` reads a different entry"*. `code.claude.com/docs/en/env-vars`: *"Useful for running multiple accounts side by side"*, with a `claude-work` alias example. | **Vendor documentation, read 2026-08-31**. Replaces the previous round's "No". |
| Documented? — the parts that are **not** | The `sha256(NFC(dir))[:8]` derivation itself; the `CLAUDE_SECURESTORAGE_CONFIG_DIR` and `CLAUDE_CODE_CUSTOM_OAUTH_URL` inputs (N1); the unset-vs-set-to-default asymmetry (N2). None appears in the docs. | Binary inspection only. See §7.3 for why B1 does not need them. |
| Refresh | Vendor-owned, in-place `security add-generic-password -U` overwrite. Live item `cdat` 2026-08-19 vs `mdat` 2026-08-31 without re-login. | Proven + current source |
| Concurrency | Cross-process `proper-lockfile` on `<storageDir>/.storage-write`; lock scope and namespace derive from the same dir, so they cannot disagree. Advisory only (Q6). | Current source |
| Failure posture | **Fails OPEN.** Non-transient store failure → plaintext `<storageDir>/.credentials.json` at 0600, **and the Keychain item is deleted**. `transient` is set only on timeout. | Current source |
| Remote revoke | **Unknown** (Q3). | Unknown |
| External custody (A) | **No supported transfer.** Also: the item's ACL necessarily trusts `/usr/bin/security`, so custody is not even exclusive today. | Proven blocker |
| **Verdict — A** | **No-go.** Requires reading the secret to establish; invalidated by the next refresh; and the enforcement mechanism *triggers the fail-open*, actively downgrading protection. | |
| **Verdict — B0** | **Go, and the prerequisite list is larger than round three's.** No derivation dependency: composing the child env and removing the three variables for the default profile needs only "unset means default", which is vendor fallback semantics rather than an undocumented formula. **Round four correction:** the composed set must also cover `ANTHROPIC_CONFIG_DIR`, `ANTHROPIC_PROFILE`, `XDG_CONFIG_HOME` and the ambient-credential variables (N20), or an ambient profile selection silently decides which identity a "deterministic" launch runs as. W21. | |
| **Verdict — B1** | **Conditional go — reversed from the previous round's no-go.** The mechanism is proven *and documented* (N8), the use case is vendor-recommended (N9), and the terms place no count limit (N10). The version-gate cost §7 priced does **not** apply to the property B1 needs on this provider (§7.3), because that property is documented rather than derived. The larger namespace surface (N1) is handled by a fail-closed presence check that needs no version pin. **One gate remains: Q1, server-side durability of a second concurrent enrolment — empirical, and runnable by an operator who already holds two accounts.** Q2c (two billed consumer subscriptions) is unaddressed by anything published and is a purchasing decision, not a research one. | |
| **Verdict — C** | **Go.** `ANTHROPIC_API_KEY`, `ANTHROPIC_AUTH_TOKEN`, `apiKeyHelper`, `CLAUDE_CODE_OAUTH_TOKEN`, named profiles/WIF are all documented with explicit refresh owners. | |

### 5.2 Codex CLI 0.150.1 — macOS

| Dimension | Finding | Standing |
| --- | --- | --- |
| Default custody | **File.** `~/.codex/auth.json`, 0600. `cli_auth_credentials_store = "file"` in the packaged fixed defaults. No `Codex Auth` Keychain item exists. | Proven (`1g880w`), re-confirmed here |
| Namespace mechanism | **Branch-dependent, and the branches price differently (§7.3.1).** *file (default):* the credential is `auth.json` inside `CODEX_HOME` — **vendor-documented**, no derivation. *keyring:* account `cli\|<sha256(canonical home).hex[:16]>` — undocumented, binary-inspected at one version. | Proven (`1g880w`); the file-branch property also vendor-documented (N12) |
| Namespace inputs | **One for the credential namespace** (N5) — and that is the *only* thing N5 establishes. **Eight ambient environment inputs** affect codex's state root, credential or OAuth endpoints: `CODEX_HOME`, **`CODEX_SQLITE_HOME`** (N14), `OPENAI_API_KEY`, `CODEX_API_KEY`, `CODEX_ACCESS_TOKEN`, `CODEX_REFRESH_TOKEN_URL_OVERRIDE`, `CODEX_REVOKE_TOKEN_URL_OVERRIDE`, `CODEX_APP_SERVER_LOGIN_CLIENT_ID` (N15). Rounds one to three used the credential-namespace count as the ambient-input count. | N5 verified here; N14/N15 from source at `rust-v0.150.1`, every literal corroborated in the installed binary |
| Documented? | **Split.** *The isolation property on the `file` branch:* **yes** — *"`file` stores credentials in `auth.json` under `CODEX_HOME`"* (N12). *Multi-account as a use case, and any isolation contract on `CODEX_HOME`:* **no**, established by a *successful* primary read of the CLI-scoped doc (`developers.openai.com/codex/auth.md`, HTTP 200), where `CODEX_HOME` appears **exactly once**, as a file location. The doc describes **one** cached login per state root. | Documentary silence on multi-account, verified; the file-branch property is documented |
| Vendor position on multi-account, **CLI surface** | **Unestablished — corrected in round three.** OpenAI publishes concurrent dual sign-in for ChatGPT web (permissive), and adds that switching is *"not yet supported in **Codex desktop** or the native ChatGPT mobile apps"* (N11) — two **GUI** surfaces, not the CLI, in the tense of a roadmap note. Round two dropped `desktop` and read that as the vendor closing the CLI; it does not. Nothing OpenAI publishes addresses the CLI. What *is* established: the capability is unshipped across Codex surfaces (27 issues, 21 open, none closed as already-supported), the vendor's own discussion thread on it has no vendor reply, and reported switches leak cross-account state (N12b). | Unshipped: first-party, reproduced. Entitlement: **neither permitted nor prohibited in anything published**. N11 second-hand and demoted to corroboration |
| Refresh | Vendor-owned. Process-local semaphore, **no cross-process lock**; file mode truncates in place. | Current source |
| Failure posture | **Resolved in round four, in codex's favour (N16).** `auto` falls back to plaintext `auth.json`; `keyring` does **not** — both fallback legs live in `AutoAuthStorage` and nowhere else, and `Keyring` dispatches to a backend with no file leg (`storage.rs:431-449, 511-545`). **N12a's "the two sources disagree" is withdrawn**: the strings N4 found by byte inspection belong to `auto`'s implementation, so `keyring` *does* mean never-plaintext. The store vocabulary is **four** values, not three — `File` (default), `Keyring`, `Auto`, **`Ephemeral`** — and the "encrypted auth storage" is a separate axis, `AuthKeyringBackendKind::{Direct,Secrets}` (N17). | Source at `rust-v0.150.1`; doc and binary now agree |
| Remote revoke | `vendor-side-on-logout`: best-effort revoke, then local delete even if revoke failed. | Current source |
| Other state | **Answered in round four (N13). Both default inside.** `log_dir` → `$CODEX_HOME/log` (`mod.rs:3906-3910`); `sqlite_home` → `$CODEX_HOME` unless `config.toml` or `$CODEX_SQLITE_HOME` says otherwise (`mod.rs:3918-3923`). Two credential-isolated profiles do **not** share a state DB by default. **The residual is an ambient one**: `CODEX_SQLITE_HOME` set in the parent environment relocates the DB out of a composed root (N14) — a B0 prerequisite, and escapable per-profile because `config.toml` outranks the variable. | Source at `rust-v0.150.1` |
| **Verdict — A** | **No-go**, same three reasons. Codex re-creates its own item on the next refresh; an externally held copy goes stale silently. | |
| **Verdict — B0** | **Go — and round four withdraws "cleaner than claude".** The L1/N2 asymmetry genuinely does not arise: setting `CODEX_HOME` to the default path *is* a no-op. But "one input" was the credential-namespace count used as the ambient-input count, and there are eight (N14, N15). Two of them are worse than a count suggests: `CODEX_API_KEY` is documented **in the vendor's own comment** as taking *"precedence over any other auth method"*, so an ambient key defeats profile selection while the launch still reports the profile's identity; and `CODEX_REFRESH_TOKEN_URL_OVERRIDE` is accepted with **no allowlist**, where claude's equivalent throws. On this axis codex is the *less* defended runtime. B0 must pin or clear all eight — **W20**, and it is a B0 prerequisite exactly as W13 is. | |
| **Verdict — B1** | **HELD — UNESTABLISHED. Re-derived in round three, and this is a change: it is no longer a no-go.** Both of round two's reasons failed inspection. *Entitlement:* **unestablished**, not negative — the sentence carrying "the vendor's published position" is scoped to Codex **desktop** and the native mobile apps and does not reach the CLI (N11, demoted). *Cost:* §7.2's derivation pin does **not** reach the packaged `file` branch, which has no derivation and whose isolation property the vendor **documents** (§7.3.1); and the premise proposed to rescue that argument — that §8 forces codex onto `keyring` — is contradicted by `bk6owf` §6.2, which classifies codex-on-`file` as **`active`**, not `degraded:plaintext` (§7.3.2). **Round four ran the free reads §12.2 listed and two of the three survivors went away.** *Q15 is closed, favourably:* `sqlite_home` defaults inside `CODEX_HOME` (N13), so credential-isolated profiles do not share a state DB — the objection this ADR called *"the substantive risk"* for a full round does not exist. *Q12 is closed, also favourably:* the keyring path has no plaintext fallback (N16). *The store-selector cost is narrowed:* the default is `File`, established by `#[default]` in first-party source at the installed tag rather than by byte inspection of one build (N17) — still operator-facing-undocumented, still escapable by H13. **What actually remains is thinner than three rounds of this document claimed, and it is exactly two things:** **no** vendor documentation of multi-account as a use case, against a doc describing a *single* cached login (N12); and **entitlement, unestablished** — nobody has published a yes and nobody has published a no. Every remaining route to a decision needs a **second account** (Q2). Nothing here closes B1-codex; nothing here establishes it. It stays **held**, and §12.2 now says so with a much shorter list. | |
| **Verdict — C** | **Go.** `CODEX_API_KEY`, `CODEX_ACCESS_TOKEN`, stdin access-token login, workload identity, per-provider `env_key`. Workload identity is the preferred managed-automation path. | |

### 5.3 Qwen — provisional, per epic AC8

**Round four ran Q10 and this section is wrong. Corrected.**

Rounds one to three recorded qwen as **not modellable**, on the ground that the
plugin declares `HomeEnvVar: ""` (`skill-agents-management`
`pkg/agentic/systems/qwen/qwen.go:121`) so there is no state root for
`vendor-opaque` custody to attach to. Q10 — the audit that would have checked
that — was listed as **free** in every round and run in none.

**qwen has a state root.** `QwenLM/qwen-code`,
`packages/core/src/config/storage.ts:193-203`: `getGlobalQwenDir()` reads
`process.env['QWEN_HOME']` and falls back to `~/.qwen`. It namespaces the
**credential**, because `getOAuthCredsPath()` (`:640-642`) resolves
`oauth_creds.json` through it. A second input, `QWEN_RUNTIME_DIR` (`:172-190`),
relocates runtime output on its own precedence chain.

**So qwen is the same shape as codex-on-`file`:** an environment-selected state
root with a plaintext credential file inside it. The empty `HomeEnvVar` is a
**defect in our plugin**, not a fact about qwen, and it is the reason five
rounds of analysis treated a modellable runtime as unmodellable. Fixing it is
**W22**, and it is B0 work — small, and it needs no vendor cooperation.

**Standing, and the limit is real.** This is first-party source at `main`.
**qwen is not installed on this host**, so there is no shipped-build
corroboration and no version pin — unlike every claude and codex claim in this
document, all of which are cross-checked against an installed binary. That gap
is **Q18**.

**What this does and does not change.** It retires *"there is no mechanism"* and
it schedules the plugin fix. It does **not** promote qwen into B1: nothing here
establishes entitlement, refresh semantics, revocation or concurrency for qwen,
and this ADR has spent three rounds learning what happens when a mechanism
finding is spent as a permission finding. Per epic AC8 qwen stays **provisional**
— but provisional-with-a-mechanism, which is a different statement from the one
this section made before. muse, gemini, agy and pi are **unaudited**, not
established as rootless; the same claim was made about qwen on the same evidence
and it did not survive one read.

---

## 6. Options compared

Security, concurrency, refresh and revocation for each — and, per the brief,
**what each one gives up**, stated for the accepted options too and not only for
the rejected ones.

### A — agents-infra-owned Keychain credentials

| | |
| --- | --- |
| **Security** | Negative on Claude. Enforcing custody means denying the CLI write access to its own item; that denial is non-transient (`transient` is set only on timeout), which triggers the fail-open: plaintext file, Keychain item deleted. The mechanism intended to strengthen custody is the mechanism that destroys it. |
| **Concurrency** | Undefined. Two writers to one item with no shared lock. |
| **Refresh** | Broken by construction. Both vendors write refreshed tokens through their own backend with no change notification; Claude's write is an in-place `-U` overwrite. Any external copy is stale from the next refresh onward, **silently**. |
| **Revocation** | Not improved. agents-infra cannot revoke server-side for either provider. |
| **Establishment** | Impossible without reading the secret. Neither vendor has an adopt-an-existing-login operation. |
| **What it gives up** | Not applicable — it delivers nothing to trade against. |
| **Verdict** | **No-go, permanent.** Three independent blockers, each individually sufficient. It remains viable *only* for credentials agents-infra minted end to end — which is option C, not custody of a native login. |

### B0 — environment determinism, one account per runtime

| | |
| --- | --- |
| **Security** | Positive and immediate. Today `runClaude`/`runCodex` build `exec.Command` and never assign `cmd.Env` (`bk6owf` §12 row 1), and claude's spawn `ChildEnv` strips exactly one key (`CLAUDECODE`). So **every** namespace variable is inherited from whatever the ancestor shell carried. **Round four widens the sizing past the three-variable framing this row used through round three** (N15, N20): claude alone has 174 `CLAUDE_*`/`ANTHROPIC_*` read sites, including a second named-profile credential store selected by `ANTHROPIC_CONFIG_DIR`/`ANTHROPIC_PROFILE`; codex has eight ambient inputs, two of them load-bearing. The worst case in that set is not a wrong namespace — it is `CODEX_API_KEY`, which the vendor's own comment says takes *"precedence over any other auth method"*, so an ambient key silently makes a launch run **as a different identity while reporting the profile's**. B0 closes the whole class this enumerates, not the three-variable slice round one named. |
| **Concurrency** | Improved. `providerlimits` currently keys rate-limit identity off `os.Getenv(capabilities.HomeEnvVar)` — the **parent orchestrator's** environment — so two children under different homes collide on one state file. `identity.go`'s own header comment names exactly this failure. B0 keys identity by the home the launch actually used. |
| **Refresh** | Unchanged. Vendor-owned, and B0 touches nothing. |
| **Revocation** | Unchanged. |
| **Version dependency** | **None on the hash derivation.** B0 needs only "unset/empty means the default namespace" (N2), which is the vendor's own fallback semantics, verified across five builds, and whose failure mode is *loud and universal* rather than silent and specific to us: if it changed, every default launch everywhere would break, not just ours. That is a categorically cheaper dependency than the derivation B1 needs. |
| **What it gives up** | It gives up the capability the epic actually asked for. B0 delivers **one** deterministic account per runtime, not switching among several. It is correctness, not a feature. It also gives up the convenience of ambient configuration: an operator who deliberately sets `CLAUDE_CONFIG_DIR` in their shell to steer agents-infra launches loses that, because B0 removes it. That is the intended trade — ambient configuration is exactly the silent decision-making being removed — but it is a real behaviour change, and Q11 exists to check nobody depends on it. **Partly answered, this host only (N20a):** none of the surveyed namespace, credential or endpoint-override variables is set in this process or named in this host's shell profiles. One host is weak evidence about every operator's setup, so the question stays open at reduced scope rather than closed. |
| **Verdict** | **Go. Adopt now.** |

### B1 — multiple simultaneously enrolled native profiles

**Split per provider.** The security, concurrency, refresh and revocation
analysis below is common to both; the two rows that decide the option — version
dependency and entitlement — differ sharply, and collapsing them was the error
the documentary read exposed.

| | |
| --- | --- |
| **Security** | Good *in the model*: the credential never leaves the vendor's store, agents-infra never touches the bytes, and the fail-open path is never engaged because nothing is ever denied. Mode 0700 on a directory the CLI's own user owns is not a denial. `bk6owf`'s C1 makes "the host holds the native login" unexpressible rather than merely rejected. |
| **Concurrency** | Good, and for a structural reason: P1 makes profile ↔ state root a bijection, so the vendors' own concurrency machinery is never asked to arbitrate between two *accounts* — only between processes of the same account, which is what it was built for. Claude's lock and namespace derive from the same directory and cannot disagree. Codex has no cross-process lock, but with one root per profile it does not need one for cross-account safety. Q6 remains the place this is most likely to be wrong. |
| **Refresh** | Best available: **not our problem.** No refresh verb, no refresh duty, an explicit prohibition list. Nothing to keep in sync is the property that makes the model robust. |
| **Revocation** | Split and honest. `logout` and `remove` are different verbs; `local` and `remote_revoke` are independent fields; silence is `unknown`, not `already-absent`; `remove --logout-policy local-logout` refuses on `unknown` or `failed` and deletes nothing. For anthropic `remote_revoke` is `unknown` and the CLI says so (Q3). For openai it is best-effort-on-logout. **The orphan hazard is real**: on macOS deleting the state root destroys the derivation input and leaves a Keychain item nothing can name again, which is why a tombstone is written by every `remove` that did not positively confirm invalidation, and why agents-infra never deletes a Keychain item itself. |
| **Version dependency** | **Split.** *claude:* **none for the property B1 needs.** N8 documents "different `CLAUDE_CONFIG_DIR` ⇒ different Keychain entry"; the `sha256[:8]` formula is what §7 priced, and B1 never needed the formula. The extra namespace inputs (N1) are handled by a *presence* refusal, which is version-independent and fails closed. See §7.3. *codex:* **branch-dependent, and smaller than round two claimed (§7.3.1–7.3.3).** On the packaged `file` default there is **no derivation** and the property is vendor-documented, so §7.2's pin does not apply. §8 does **not** force codex onto `keyring` — `bk6owf` §6.2 classifies codex-on-`file` as `active` (§7.3.2). What remains is a version gate on the store **selector**, whose default the vendor never documents (§7.3.3) — real, but escapable by setting the key explicitly (H13), which is the same move §7.3 makes for claude. |
| **Entitlement dependency** | **Split, and this is the reversal.** *claude:* Anthropic documents running multiple accounts side by side through this exact variable (N9) and its terms limit sharing, not count (N10). What remains is Q1 — server-side durability — which is empirical and runnable. *codex:* **unestablished for the CLI surface**, and this is a round-three correction. Round two called it "a published negative" on the strength of a sentence scoped to **Codex desktop** (N11). Nothing OpenAI publishes addresses whether an operator may hold two Codex CLI logins; the CLI-scoped doc is silent (N12) and describes a single cached login, the tracker shows the feature requested and unshipped (N12b), and the vendor's own discussion thread on it has no vendor reply. Unestablished is neither "no evidence" nor "the vendor said no". **The codex no-go does not rest on this row** — §7.3.1 does. |
| **What it gives up** | Three things. (i) **Freedom to upgrade.** Every profile-selected launch becomes conditional on a version allowlist someone must extend by hand after each release (§7). (ii) **Operator-visible account attribution from the system of record.** Claude's Keychain account field is `$USER`, not the identity; nothing in the item says which Anthropic account it holds, so attribution lives only in agents-infra's own profile record and is only as good as that record. (iii) **A clean uninstall.** Because the credential lives outside the state root on Claude, any `remove` that cannot positively confirm invalidation leaves an auditable orphan behind by design. |
| **Verdict** | **claude: conditional go**, gated on Q1 alone (§12.1). **codex: held — unestablished** (§12.2), *changed from no-go in round three*. Still not one verdict, but the asymmetry is much smaller than round two claimed and sits somewhere else: not entitlement, not the version gate, but that Anthropic **documents and recommends** the workflow and documents what state the config dir carries, while OpenAI documents neither as a multi-account use case. ~~and leaves `sqlite_home`'s default unstated, which is a live cross-account leakage risk (Q15)~~ — **withdrawn in round four: it defaults inside `CODEX_HOME`** (N13). What remains on the codex side is the absence of documented *intent*, not an isolation gap: nothing OpenAI publishes names running two `CODEX_HOME` roots as a supported pattern, while Anthropic's own documentation does exactly that for `CLAUDE_CONFIG_DIR`. |

### C — provider-native switching and delegation

| | |
| --- | --- |
| **Security** | Strong where it applies. The credential's whole lifecycle is owned by an authority with an explicit refresh boundary — agents-infra's own secret lifecycle for an API key it minted, or an external IdP for workload identity. No undocumented derivation anywhere. |
| **Concurrency** | Trivial. Injected credentials are stateless per process; there is no shared mutable store to race. |
| **Refresh** | Explicit owner by construction: `host` for `api-key`, `external-authority` for WIF/SSO. `bk6owf` renders the owner from the custodian class rather than letting an adapter declare it, so a native method cannot claim host refresh. |
| **Revocation** | Real. A host-minted API key can actually be revoked by the host; a WIF identity token expires on the authority's schedule. This is the only option where agents-infra can genuinely revoke. |
| **What it gives up** | **The subscription.** This is the whole cost and it is not small: C routes inference through API/console billing or an enterprise org instead of the operator's personal Claude or ChatGPT subscription. It solves managed-automation identity well and personal-subscription multi-account **not at all**. It also gives up the native session's feature surface where the two differ — `--bare` is named in `3moaky` as the reduced-surface variant with a hard no-native-Keychain-read boundary. |
| **Verdict** | **Go, for managed automation. Not a substitute for B1.** |

---

## 7. The undocumented derivation dependency, priced — and scoped

The brief requires this explicitly: what breaks on a Claude minor bump, and what
the version gate costs operationally. Deferring it would be the decision
declining to own its own consequence.

**Read §7.1 and §7.2 as written, then §7.3, which scopes them.** They price the
dependency on the *`sha256[:8]` service-name derivation*. That pricing is correct
and is unchanged by the documentary read. What the read changed is **which
provider still has to pay it**, because on Claude the property B1 actually needs
turns out to be documented rather than derived.

### 7.1 What breaks

The derivation `service = "Claude Code" + suffix + "-credentials" + "-" +
sha256(NFC(dir))[:8]` is current-source with no compatibility promise, and N1
shows the composition has more inputs than the hash. If any part changes in a
release:

- The CLI queries a service name that does not exist.
- It reports **not logged in** and offers to log in.
- A login there creates a *new* credential under the *new* derivation.
- The old item stays in the login keychain under a name nothing derives any
  more — an orphan, invisible, holding a live credential.

**Every layer of this is silent.** No error, no warning, no diagnostic. That is
why `bk6owf` §7.1 makes the gate *refuse* rather than warn, and that choice is
correct. Warning and proceeding enrols a profile into an orphaned namespace or
silently shares one account across two profiles, and neither announces itself.

### 7.2 What the gate costs

Observed on this machine, from `~/.local/share/claude/versions/`:

| Version | Installed |
| --- | --- |
| 2.1.234 | 2026-08-17 23:34 |
| 2.1.235 | 2026-08-19 10:49 |
| 2.1.236 | 2026-08-19 23:32 |
| 2.1.247 | 2026-08-27 13:34 |
| 2.1.248 | 2026-08-28 01:16 |

| Metric | Value |
| --- | --- |
| Upstream patch versions across the window | 14 (2.1.234 → 2.1.248) in 11 days ≈ **1.3/day** |
| Builds actually landing on this host | 5 in 11 days ≈ **one every 2.2 days** |
| Verified derivation points | **5** |
| Versions the interval range asserts but nobody verified | **10** (2.1.237 – 2.1.246) |
| Codex verified points | **1** (0.150.1) |

So, with B1 enrolled and an honest allowlist gate (N6):

- **Every Claude Code upgrade refuses every profile-selected launch** until
  someone installs the new build, runs the pin test with all four negative
  variants, records the result, extends the allowlist, and re-verifies one
  existing profile's coordinates against the new build. At the observed cadence
  that is a **manual gate operation roughly every two days**, on the critical
  path of the operator's primary tool.
- **Codex refuses on its very next upgrade**, full stop. The range is one
  version. Q4 would widen it to the installed set for free, but the structural
  problem — that each release needs a human-run pin — does not go away.
- Choosing an interval instead of an allowlist trades that cost for a claim
  about ten unexamined versions. It buys latency, not safety, and it buys it
  with exactly the currency the gate exists to protect.

**As priced, this is the single largest cost in the whole decision and it is
recurring.**
The mechanism works; the mechanism is free; keeping the mechanism *trusted* is
not. A B1 that depends on the derivation does not merely need to be built once —
it needs to be re-verified faster than the vendors ship. That is the reason B0's
*absence* of this dependency (§6) is the load-bearing argument for the split
rather than a consolation prize.

### 7.3 Which provider still pays this, after the documentary read

The previous round applied §7.2's cost to both providers and concluded B1 failed
on cost-benefit independently of entitlement. That conclusion **does not survive
for Claude**, and the reason is a distinction §7.1 never drew: between the
*derivation* and the *property*.

| | The derivation | The property B1 needs |
| --- | --- | --- |
| Statement | `service = "Claude Code" + suffix + "-credentials" + "-" + sha256(NFC(dir))[:8]` | *A session with a different `CLAUDE_CONFIG_DIR` reads a different Keychain entry* |
| Source | Binary inspection, five builds, no compatibility promise | **`code.claude.com/docs/en/authentication`, published** (N8) |
| Needed to enrol and select a profile? | **No** | **Yes** |
| Needed to *predict a specific service name*? | Yes | No |
| Version gate required? | Yes — §7.2's every-two-days operation | **No.** A documented behaviour is a supported contract; it is not re-derived per release |

B1 selects a profile by *choosing the directory the child runs under*. It never
needs to compute the service name — the vendor computes it. The pin test existed
to catch a silent change in a formula nobody promised; there is no formula in the
path once the property itself is promised. The residual undocumented items on
Claude (N1's two extra namespace inputs, N2's unset-vs-default asymmetry) are
handled by **refusing when an unexpected namespace variable is present**, which
is a presence check, is version-independent, and fails closed. Over-refusal is
safe; the failure §7.1 describes is silent enrolment into an orphaned namespace,
and a presence check cannot produce it.

Honest limits on this, stated rather than glossed:

- A documented behaviour can still be changed by the vendor. The difference is
  that a documented change is announced and applies to everyone, which is the
  same argument §6 makes for B0's dependency being categorically cheaper — and it
  is now the same *class* of dependency, not merely a smaller one.
- Only the **macOS Keychain keying** is documented. Nothing about the two extra
  inputs is, which is exactly why the refusal set stays and stays fail-closed.
#### 7.3.1 The same distinction, applied to codex — which round two declined to do

Round two ended §7.3 with one sentence: *"Codex pays §7.2 in full. One verified
version, an undocumented derivation, and refusal on the very next upgrade."*
**That is false as written.** An argument that makes one verdict cheap and is
then withheld from the other is exactly where a reader should look for motivated
reasoning, so it is applied symmetrically here — and the result is stated
whichever way it falls. It falls against round two's conclusion.

Codex does not have one branch. It has two, and they price differently.

| | claude (Keychain) | **codex `file` — the packaged default** | codex `keyring` |
| --- | --- | --- | --- |
| The property B1 needs | different `CLAUDE_CONFIG_DIR` ⇒ different Keychain entry | different `CODEX_HOME` ⇒ different `auth.json` | different `CODEX_HOME` ⇒ different keyring account |
| Vendor-documented? | **Yes** (N8) | **Yes** — *"`file` stores credentials in `auth.json` under `CODEX_HOME`"* (N12, ledger 20) | **No.** The docs say only "your operating system credential store" |
| Derivation on the path? | No | **No — there is none.** The credential is a file inside the root; `cli\|sha256(canonical home)[:16]` exists **only** on the keyring branch (§5.2) | Yes, and binary-inspected at exactly **one** version |
| §7.2's every-release pin, *for the derivation*? | **No** | **No** | **Yes** |
| Does the launch still gate on version? | No | **Yes — but for a different constant**, see below | Yes |

So the sentence round two wrote does not describe codex's default branch at all.
On `file` the property is published and there is **no formula to pin**, because
the credential is simply a file in a directory the operator chooses. §5.2's own
B0 verdict already says this — *"Go, and cleaner than claude: one input"* — and
B0 and B1 need the same property from the same mechanism.

#### 7.3.2 A premise was proposed to rescue the sentence. It does not hold, and the check is recorded because the failure is the finding

Review proposed the rescue explicitly: codex's `file` store *is* the plaintext
`auth.json`; §8 ends with *"a `degraded:plaintext` profile refuses to launch"*;
therefore a codex B1 profile is forced onto `keyring`, which does carry the
derivation — so the cost survives after all.

**The design it cites says the opposite, in a row written to prevent exactly this
reading.** `bk6owf` §6.2, verbatim:

> **`degraded:plaintext` means *unexplained* plaintext, not plaintext.** It is a
> claim that a store failed open, not a claim about file custody as such.

| Recorded store | Observation | Custody state |
| --- | --- | --- |
| claude Keychain | `<state_root>/.credentials.json` present | `degraded:plaintext` |
| **codex `file`** | `<state_root>/auth.json` present at 0600 | **`active`** — *"this is the vendor's packaged default custody, not a fallback"* |
| codex `keyring` | `<state_root>/auth.json` present | `degraded:plaintext` |

And the design states the reason directly: *"Calling codex-on-`file`
`degraded:plaintext` would refuse every default codex enrolment for a hazard that
did not occur — the credential is in a 0600 file **by design**."*

So **§8 does not refuse a codex `file` profile**, nothing forces B1-codex onto
`keyring`, and the proposed rescue fails. It is recorded rather than quietly
dropped because a reviewer-supplied premise carries the same standard as any
other input: this ADR's §13 exists precisely to record inputs that did not
survive being checked, and an unchecked correction is the same failure mode as
an unchecked claim, wearing the reviewer's authority instead of the producer's.

#### 7.3.3 What codex actually still pays, which is neither of the above

Applying the distinction properly finds a real cost, and it is a **third**
constant that neither round two nor the review named. `bk6owf` §7.1 already
identified it and this ADR walked past it twice:

> The codex range applies to **both** store branches. The `file` branch has no
> hash that a version bump could break, **but the packaged default that *selects*
> that branch is itself a version-fixed constant**, so a build outside the range
> can silently change which branch is in use — and with it, which coordinates
> every enrolled codex profile should be observed at.

That is exactly right, and it is the property/derivation distinction applied one
level up. The chain is:

1. The property *"`file` stores credentials in `auth.json` under `CODEX_HOME`"*
   is **documented** (N12) — so *given* the file branch, coordinates are safe.
2. **Which branch is in use** is decided by `cli_auth_credentials_store`, whose
   **default value the vendor never documents.** Verified in round three: the
   config reference lists the key as `file | keyring | auto` with a description
   and **no default**; the auth page's only worked example sets `"keyring"` —
   which is *not* the shipped default (ledger 30).
   **Round four narrows this, and the narrowing is worth stating precisely.**
   That `file` is the default is no longer known *only* from byte inspection of
   one build: it is `#[default]` on `AuthCredentialsStoreMode::File` in
   first-party source at the exact installed tag (N17, ledger 32). That is
   better evidence than a fixed-defaults byte window. It is **not** a
   documented contract — a `#[default]` attribute is precisely the kind of
   constant that moves between releases without a doc change, and the config
   reference still states no default. The cost is smaller; its shape is
   unchanged. Also unchanged by round four: the accepted-value list is **four**
   long, not three (`Ephemeral`), so a gate written against `file | keyring |
   auto` is not total (N17).
3. A build that flips that default moves every enrolled codex profile's
   credential from `auth.json` to the keyring silently — the same silent-orphan
   shape §7.1 describes, reached by a different route.

**So codex does pay a §7.2-shaped cost on its default branch — for the branch
selector, not for the hash.** Round two reached a defensible-sounding conclusion
by an argument that does not hold, and the correct argument was sitting in its
own input document.

**And, stated because the symmetric treatment demands it: this cost is escapable,
and cheaply.** The selector is a documented, vendor-supported `config.toml` key,
and B1 owns each profile's `CODEX_HOME` — so a profile can **set
`cli_auth_credentials_store` explicitly at enrol** instead of inheriting an
undocumented constant. That is the identical move §7.3 makes for claude: rely on
the published contract, not on the derived value. Take it and codex's
version-gate cost goes to **zero on the file branch**, leaving only the
documented property. `bk6owf` §3.3 already carries a `store_selector` field on
the profile record, but it *records* the observed selector rather than *setting*
it; closing that gap is a design change, not research, and §11.3 carries it as
**H13, unstarted**.

**Consequences for claims made elsewhere, all corrected:**

- §9's *"three independent reasons"* for a codex no-go: N11 is retired to
  corroboration (§2.3), the §7.2 derivation cost does not apply to the default
  branch, and what survives is a *different* cost that is escapable by a design
  change. The count does not survive and neither does the conclusion it
  supported. §9 and §12.2 are re-derived.
- §7.3's earlier word **"overdetermined"** is withdrawn outright. Nothing about
  B1-codex is overdetermined; the honest word for what remains is
  **unestablished**.
- **The asymmetry between the providers is real but much smaller than round two
  claimed, and it sits somewhere else.** It is not entitlement (unestablished for
  codex, permissive for claude), and it is not the version gate (escapable on
  both). It is that Anthropic **documents and recommends the multi-account
  workflow** with a worked example (N9) and documents that settings, session
  history and plugins live under the config dir, while OpenAI documents the
  credential's location and stops.
  **Round four shrinks it again, and corrects the sentence that used to end this
  bullet.** That sentence read: *"`sqlite_home` has no documented default at all
  … a live cross-account leakage risk of exactly the kind AC6 forbids, and it is
  a **free** source audit nobody has run."* The audit has now been run.
  `sqlite_home` **defaults inside `CODEX_HOME`** (N13), so the leakage risk does
  not exist by default and the asymmetry loses its sharpest edge. What is left of
  it is documentation, not mechanism: OpenAI documents *where the credential
  lives* and does not document *the use case*. On the mechanism itself the two
  runtimes are now closer than any round of this document has said.

## 8. The plaintext-fallback hazard, carried forward

Carried as `bk6owf` §6.1 states it after its own downgrade, **not** as the
earlier cleaner version. The hazard splits three ways and they are not averaged:

1. **Eliminated — the declaration half (C1).** Custodian class is a total
   function `Method -> class` evaluated in code, not a field on anything. No
   config file, flag, registry column, project table, remote-config key, plugin
   or environment variable can pair a native-OAuth method with `host-owned`,
   because there is nothing to set. This is the absence of an input, not a
   validation that refuses one; there is no check to bypass and no entry point
   that can forget to call it.

2. **Mitigated by CI — the implementation half (5.3, 6.4).** The prohibition
   list is a list of operations no code path performs: setting an ACL, changing
   ownership, making a vendor store unwritable. Nothing in the type system stops
   an implementer writing `os.Chmod` against a state root's credential store.
   What stops it is a module-wide greppable-absent assertion plus negative
   tests, enforced in CI. That is a real gate and it is the primary defence for
   this half — but it runs at **build time** and protects only what its pattern
   set names. A new way to spell the same operation, added without extending the
   set, passes.

3. **Mitigated by a launch-time gate — the input half (C2).** The class
   function's sole input, `method`, lives on the profile record, which is an
   editable on-disk file. Rewriting it declares no custody — C1 still holds —
   but reaches the `host-owned` branch by forging the input. C2 compares the
   computed class against the recorded `backend`/`store_selector` at enrol and
   every launch and refuses a mismatch. **It is a checked gate, not a structural
   impossibility**: it converts a one-field edit into a two-field consistent
   rewrite, which is a real reduction and no defence at all against an actor who
   can already write both fields. Q13 asks whether sealing the record is worth
   its key.

"Three halves is the wrong word and the right count." Two thirds of the defence
runs at build or launch time, and a decision budgeting residual risk from this
document should budget it that way.

**One thing this ADR adds to that picture, adverse — and one it later withdrew:**

- ~~**N4.** Codex's `keyring` store has a load-and-save fallback path to file
  storage. If it is ungated, codex inherits the same fail-open shape and §5.3's
  prohibitions become load-bearing on the codex-keyring branch too — a branch
  the design currently reasons about as merely *unknown*. `bk6owf` §6.2's
  choice to classify plaintext-beside-keyring as `degraded:plaintext` rather
  than guess is now better supported than when it was made.~~ — **withdrawn in
  round four** (N16). The fallback **is** gated: both legs live in
  `impl AuthStorageBackend for AutoAuthStorage` and nowhere else, `Keyring`
  dispatches to a backend with no file leg, and `bk6owf` §6.2's
  `degraded:plaintext` classification for codex-keyring gains no support from
  this — there was no conflict to support it with.
- **N1.** The namespace has more environment inputs than the hazard analysis
  modelled. This does not create a new fail-open path — none of the three
  variables denies the CLI anything — but it widens the set of ways a profile
  can end up observing the wrong coordinates, which is what H3's drift check and
  §7.3's refusal exist for. The refusal set must grow from
  `CLAUDE_SECURESTORAGE_CONFIG_DIR` alone to include
  `CLAUDE_CODE_CUSTOM_OAUTH_URL`.

**Detection stays as designed and is not weakened here.** `stat`-only, three-
valued, and `unknown` never collapses into `absent` or `active` (D1). Rows one
and five of §6.2's table carry the same "every observation succeeded" qualifier
in both directions, because `active` is as much a positive claim as `absent` is.

**The proof-gate-4 branch is taken as written**: detect and refuse, with a
per-profile, dated, reasoned acknowledgement as an audited escape hatch. This
ADR does not exercise the "accepted in writing" branch. Plaintext custody is
**not** tolerated by default; a `degraded:plaintext` profile refuses to launch.

---

## 9. Decision

**Adopt B0 and C now — B0 with the five prerequisites round four established,
not the one round three recorded. Reject A permanently. Hold B1-claude as
CONDITIONAL GO behind one experiment. Hold B1-codex as UNESTABLISHED behind one
experiment too — round four spent the free reads and they did not decide it, so
what is left needs a second account. It is not rejected, and round three withdrew
the no-go that said it was. Carry qwen as provisional-with-a-mechanism, which is
a change from "not modellable".**

Concretely:

1. **A is rejected for native logins, permanently.** Not a preference — the
   enforcement mechanism is actively harmful on Claude. This is not revisited by
   new evidence about namespacing; only a vendor-published adopt-an-existing-
   login operation would reopen it, and neither vendor has one.

2. **B0 is adopted — and round four enlarges what "the full namespace-input
   set" means.** The launch plane composes the child environment
   deterministically over that set, for both runtimes, on every launch.
   `home_env` is validated rather than trimmed; `Plan.Home` becomes load-bearing
   or is removed; `providerlimits` keys identity by the launch's home. Invariant
   **L1′** (§2, N2) replaces `bk6owf`'s L1: compose, and for the default claude
   profile that means *removing* `CLAUDE_CONFIG_DIR`, not writing it. No version
   gate. No pin. No credential.

   **The correction round four forces.** Rounds one to three sized this set from
   each runtime's *credential-namespace* inputs — three for claude (N1), one for
   codex (N5) — and §11.2 stated outright that *"W13 is the only one B0 depends
   on."* **That was false.** Codex has **eight** ambient inputs that relocate its
   state root, supply its credential or redirect its OAuth endpoints (N14, N15);
   claude has a state-root, ambient-credential and endpoint-redirect class of
   comparable size plus an entire **second, profile-named credential store**
   selected by `ANTHROPIC_CONFIG_DIR`/`ANTHROPIC_PROFILE` (N20); qwen has
   `QWEN_HOME` and `QWEN_RUNTIME_DIR` (N19). Two of codex's are not merely
   untidy: `CODEX_API_KEY` outranks *any* persisted credential by the vendor's
   own comment, so an ambient key makes a composed launch run as the wrong
   identity while reporting the right one; and `CODEX_REFRESH_TOKEN_URL_OVERRIDE`
   is accepted with no allowlist. **B0 gains three prerequisites — W20 (codex),
   W21 (claude), W22 (qwen) — alongside W13.** None needs a vendor, a version
   gate or a credential. B0 remains adopted; it is *more* worth doing after this
   read than before, because an ambient input nobody composes is exactly the
   defect B0 exists to remove.

3. **C is adopted for managed automation.** Workload identity preferred for
   Codex; named profiles / WIF / `apiKeyHelper` for Anthropic. Refresh owner
   recorded explicitly per profile; a profile with no single owner is refused.

4. **B1 is split.** `bk6owf`'s design is accepted as the design B1 *would* use —
   it is not rejected and should not be rewritten. **Neither half is built, and
   the work breakdown in §11 is recorded and remains unstarted**, but the two
   halves are held for different reasons and should not be un-held together.
   - **B1-claude: conditional go.** The mechanism is documented (N8), the use
     case is vendor-recommended (N9), the terms do not limit account count (N10),
     and §7.2's recurring version-gate cost does not apply to the documented
     property (§7.3). The single remaining gate is **Q1**, an experiment an
     operator can run with two accounts they already hold. Until Q1 runs, this
     stays unstarted — a likely outcome is not a finding.
   - **B1-codex: held, unestablished. Round three withdrew round two's no-go**,
     because all three of its stated reasons failed inspection and no
     replacement closes it (§7.3.1–7.3.3, §13).
     **Retired — N11.** It is scoped to Codex *desktop* and the native mobile
     apps and does not reach the CLI; entitlement there is **unestablished**,
     neither published yes nor published no.
     **Retired — §7.2's derivation cost.** The packaged `file` branch has no
     derivation and its isolation property is vendor-documented.
     **Retired — the proposed rescue.** §8 does not force codex onto `keyring`:
     `bk6owf` §6.2 classifies codex-on-`file` as `active`, *"the vendor's
     packaged default custody, not a fallback"*.
     **Retired in round four — Q15.** `sqlite_home` **defaults inside
     `CODEX_HOME`** (N13). The item this ADR called *"the substantive risk"* and
     *"a free source audit nobody has run"* has now been run, and it closed
     **favourably**: credential-isolated codex profiles do not share a state DB.
     **Retired in round four — Q12.** The keyring path has no plaintext
     fallback; N12a's doc-versus-binary conflict is withdrawn (N16).
     **Narrowed in round four — the store-selector gate.** The default is
     `#[default] File` in first-party source at the installed tag (N17), not
     merely a byte window; still operator-facing-undocumented, still escapable by
     H13.
     **What survives, and it is two things:** no vendor documentation of
     multi-account as a use case, against a doc describing a **single** cached
     login; and **entitlement, unestablished**. Every remaining route to a
     decision needs a **second account** (Q2). B1-codex stays unstarted for the
     same reason B1-claude does — the evidence is not in — and §12.2 now lists a
     much shorter set of what would bring it in.

5. **Qwen remains provisional per epic AC8 — but a mechanism *is* now
   established, and round four corrects this line.** Rounds one to three said
   *"with no mechanism established"* on the strength of our own plugin declaring
   `HomeEnvVar: ""`. qwen has `QWEN_HOME`, and it namespaces the credential
   (N19). The empty declaration is a **repository defect**; fixing it is B0 work
   (W22). qwen stays provisional because nothing establishes its entitlement,
   refresh, revocation or concurrency — not because it has no state root.
   muse, gemini, agy and pi are **unaudited**, and this ADR no longer asserts
   they are rootless: that claim was made about qwen on identical evidence and
   did not survive one read.

**Why not just build B1-claude now, given the design is done, the mechanism is
documented and the vendor recommends the use case?** Because AC1 requires that
unsupported behaviour is not inferred, and *"a second concurrent enrolment leaves
the first working server-side"* would be inferred. Anthropic documenting the
side-by-side workflow makes that outcome likely; it does not state it. The gap
between "the vendor recommends this" and "we observed it hold for 24 h" is
exactly the gap Q1 closes, and it is one cheap experiment wide. Promoting an
inference to a finding here would be the same error the previous round made in
the other direction — reporting a state of the evidence that had not been
checked.

**And why not build B1-codex on the strength of B1-claude?** Because that is the
forced fit AC5 exists to permit refusing. Anthropic documents the workflow, names
the use case, and documents that settings, session history and plugins live under
the config dir. OpenAI documents the credential's location and stops.
**Round four removes the second half of the sentence that used to be here** — it
read *"leaving `sqlite_home`'s default unstated, which is the one place
cross-account state could still be shared"*, and `sqlite_home` defaults inside
the root (N13). The remaining reason is narrower and is documentation, not
mechanism: extending one provider's *entitlement* evidence over the other is
exactly N5's under-specification, and it is the error round two committed in the
restrictive direction and round three would have committed in the permissive
one.

**A caution about how this hold should be read**, because round two's framing
invited the wrong one and a stale no-go is worse than an honest unknown.
**B1-codex is not rejected. Nothing found in four rounds closes it.** It is
unstarted because the evidence for it is not in. Round three said *"most of what
would settle it is ours and free"* and named three items: Q15, Q12 and H13.
**Round four ran the two reads. Both came back in codex's favour and neither
settled it** — because neither was ever about entitlement, which is the thing
that is actually open. H13 remains a design change and stays unstarted.

That is the round-four lesson and it is not the one round three expected: the
free reads were worth running, they removed two stated objections, **and the
verdict did not move**, because the objections they removed were not the reason
for the verdict. A reader who inherits "NO-GO, the vendor closed it" would never
have run them; a reader who inherits "three free reads away from a decision"
would have expected them to decide something. Both readings are wrong, and the
honest one is now short enough to state in a sentence: **the only thing left
that could decide B1-codex is a second account.**

The epic's AC6 — a chosen model supporting multiple independently named accounts
— remains **not met as a whole**, because it is written across both providers
and one of them is closed. It is *reachable* for Claude alone once Q1 runs. §15
records that split rather than a single pass/fail.

---

## 10. Proof-of-concept gates and CLI UX — viable paths only

Per the acceptance criteria, gates and UX are specified for B0 and C. **No CLI
UX is specified for B1's enrol/switch/logout verbs, on either provider.**

For **B1-codex** the conclusion is unchanged and round four changes the reasoning
a second time. Round three's version rested on Q15: *"a UX for enrolling two
codex profiles specified before anyone has checked whether those profiles share a
state database would be committing to a shape the evidence might refuse."* That
check has been run and it **passed** — profiles do not share a state DB (N13) —
so that particular reason is withdrawn. What remains is the *weaker* form of the
rule that applies to B1-claude below, and it applies harder: B1-codex is further
from a decision, not closer. The open item is no longer a mechanism question this
side could answer; it is **entitlement**, and it needs an account (Q2).
Specifying enrol/switch UX for a runtime whose permission to hold two logins is
unestablished would commit to a shape the evidence has not licensed. For **B1-claude** the reasoning
is
different and worth stating, because "conditional go" could be misread as
licensing UX work now. It does not. The condition is a single unrun experiment
(Q1), and a conditional go whose condition has not been met is still not a path
to build against; specifying its UX would convert a likely outcome into a
committed one. `bk6owf` §8 already holds the grammar for both, and it stays
there, unimplemented, until §12.1 is satisfied.

### 10.1 B0 gates

Each is a **negative** test, because all three current failures are silent and a
positive test passes against the broken code.

| # | Gate | The negative that gives it meaning |
| --- | --- | --- |
| **G-B0-1** | The composed child environment for a default claude launch contains **no** `CLAUDE_CONFIG_DIR`. | Must fail when the variable is present in the child env — including when it is set to the *correct default path*. A test that only fails on a wrong path asserts nothing about N2. |
| **G-B0-2** | A parent process carrying `CLAUDE_CONFIG_DIR`, `CLAUDE_SECURESTORAGE_CONFIG_DIR` or `CLAUDE_CODE_CUSTOM_OAUTH_URL` must not propagate any of them to the child. | Must fail when the child inherits. Run once per variable; a test that only covers the first proves filtering exists and says nothing about the set. |
| **G-B0-3** | A codex launch's child environment carries `CODEX_HOME` from the plan, not from the parent. | Must fail when `Plan.Home` is dropped **and separately** when the parent's value leaks through and happens to match. Inheriting the right value by accident must not pass. |
| **G-B0-4** | Two launches with distinct homes produce distinct `providerlimits` `IdentityKey`s and distinct on-disk state files. | Must fail if identity is resolved from the process environment. |
| **G-B0-5** | A typo'd `home_env`, and a `home_env` naming a variable the declared agentic system does not use, are both refused at config parse with a typed error. | Must fail if either is accepted. Today both are accepted in silence. |
| **G-B0-6** | An agentic system declaring `HomeEnvVar: ""` (muse, gemini, agy, pi — **no longer qwen**, see W22/N19) is launched with no home variable written and no error. | Must fail if an empty `HomeEnvVar` causes a variable named `""` to be composed. |
| **G-B0-7** | **New (N14, N15).** A parent carrying any of `CODEX_SQLITE_HOME`, `OPENAI_API_KEY`, `CODEX_API_KEY`, `CODEX_ACCESS_TOKEN`, `CODEX_REFRESH_TOKEN_URL_OVERRIDE`, `CODEX_REVOKE_TOKEN_URL_OVERRIDE` or `CODEX_APP_SERVER_LOGIN_CLIENT_ID` must not propagate it, and an ambient credential or endpoint-override variable must **refuse** the launch rather than warn. | Must fail **per variable** — seven cases, not one; a single-variable test proves filtering exists and says nothing about the set, which is the exact error that let `CODEX_SQLITE_HOME` go three rounds unnoticed. Must **separately** fail if the refusal is downgraded to a warning: an ambient `CODEX_API_KEY` outranks any persisted credential, so admitting it launches the wrong identity under the right label. Narrow the gate — admit exactly one variable — and it must still fail. |
| **G-B0-8** | **New (N20).** A parent carrying `ANTHROPIC_CONFIG_DIR` or `ANTHROPIC_PROFILE` must not propagate either to a claude child. | Must fail when either leaks. These select a **different credential store** from the Keychain one B0 reasons about, so a gate that covers only `CLAUDE_*` variables passes while the child authenticates from a file profile nobody composed. |
| **G-B0-9** | **New (N19).** A qwen launch's child environment carries `QWEN_HOME` from the plan, and `QWEN_RUNTIME_DIR` is composed rather than inherited. | Must fail if `HomeEnvVar` is empty for qwen, and separately if the parent's `QWEN_HOME` leaks through and happens to match. |

**None of these needs a credential, a Keychain call, a version gate or a second
account.** That is the argument for B0 in one line.

**G-B0-7 through G-B0-9 are round-four additions**, and they are the reason §2.4
matters to the *adopted* option rather than only to the held one. Each is written
as a per-variable negative on purpose: the defect that produced them was a set
sized from the wrong enumeration, and a gate that admits one member of a set it
claims to cover is the same defect in test form.

### 10.2 B0 CLI UX

B0 adds **no verbs**. It changes what an existing launch does and makes the
result visible:

```
agents-infra claude --print-config
agents-infra codex  --print-config
```

must render the resolved state root and the composed namespace-variable set —
`CLAUDE_CONFIG_DIR: <unset — default namespace>` or
`CODEX_HOME: /Users/…/.codex`, plus the removal of the other namespace
variables — so the account a launch will use is inspectable **before** the child
starts. Today it is not, which is how the defect stayed invisible.

`agents-infra auth doctor` (read-only) reports, per runtime: installed version,
whether any namespace variable is present in the operator environment, and
whether the launch would compose or inherit. Metadata only; it makes no
`security` call and reads no credential file.

### 10.3 C gates

| # | Gate | The negative |
| --- | --- | --- |
| **G-C-1** | A profile whose refresh owner is not exactly one of `host` / `external-authority` is refused. | Must fail when a profile with no owner, or two, is admitted. |
| **G-C-2** | An `api-key` or WIF profile's secret never appears in argv, config, composed environment agents-infra writes, logs, board resources or diagnostics. | Must fail when a secret-marked input is supplied on argv — `E_AUTH_SECRET_ON_ARGV`, before parsing continues. |
| **G-C-3** | `apiKeyHelper` stdout/stderr is redacted by construction in every diagnostic path. | Must fail when a diagnostic renders helper output unredacted. |
| **G-C-4** | Provider-auth and app-server transport-auth remain distinct types in schema, CLI, logs and tests. | Must fail when a transport token is accepted where a provider credential is expected. |

### 10.4 C CLI UX

Uses the existing `[agents.<runtime>]` project-config surface plus the
host-owned secret lifecycle already in the repo. No new native-login verbs. A
managed profile declares its refresh owner explicitly and is refused without
one.

---

## 11. Implementation work breakdown — recorded, unstarted

**Nothing below is started. This ADR implements nothing.** Sizes are relative,
not estimates.

### 11.1 B0 — adopted, ready to schedule

| # | Repository | Change | Gate |
| --- | --- | --- | --- |
| **W1** | `relux-agents-infra` | `runClaude` / `runCodex` (`tools/agents-infra/main.go:417-472`) compose an explicit child environment instead of inheriting. Implements L1′ including the *removal* case for claude's default profile. | G-B0-1, G-B0-2 |
| **W2** | `relux-agents-infra` | `BuildClaudeLaunchPlan` / `BuildCodexLaunchPlan` carry the resolved state root; `--print-config` renders it and the composed namespace-variable set. | §10.2 |
| **W3** | `skill-agents-management` | `claude`'s and `codex`'s `ChildEnv` write `HomeEnvVar=<Plan.Home>` — and, for claude, remove `CLAUDE_CONFIG_DIR` when the plan selects the default. `Plan.Home` becomes the input rather than inert data. | G-B0-3 |
| **W4** | `skill-agents-management` | `providerlimits` gains an explicit-home entry point; identity is keyed by the launch's home, not `os.Getenv`. | G-B0-4 |
| **W5** | `skill-project-management` | `home_env` validated, not trimmed: env-var-name syntax, agreement with the declared `HomeEnvVar`, typed refusal on mismatch. Routed into `LaunchRequest.Home`. | G-B0-5 |
| **W6** | `skill-project-management` | `internal/spawn` needs **no** change *provided* W3 lands: `planCommand` already sets `cmd.Env = plan.Env`. Recorded so the fix is not made twice in two places that can later disagree. | covered by G-B0-3 |
| **W7** | `relux-agents-infra` | `agents-infra auth doctor`, read-only, metadata only. | §10.2 |
| **W20** | `relux-agents-infra` + `skill-agents-management` | **New (N14, N15). Codex ambient-input closure.** The composed child environment pins or clears all eight: `CODEX_HOME`, `CODEX_SQLITE_HOME`, `OPENAI_API_KEY`, `CODEX_API_KEY`, `CODEX_ACCESS_TOKEN`, `CODEX_REFRESH_TOKEN_URL_OVERRIDE`, `CODEX_REVOKE_TOKEN_URL_OVERRIDE`, `CODEX_APP_SERVER_LOGIN_CLIENT_ID`. `--print-config` renders the whole set. Ambient credential or endpoint-override variables **refuse**, they do not warn — `CODEX_API_KEY` outranks any persisted credential by the vendor's own comment, so admitting it means launching as the wrong identity while reporting the right one. | **G-B0-7** (new) |
| **W21** | `relux-agents-infra` + `skill-agents-management` | **New (N20). Claude ambient-input closure.** Extend the composed set beyond N1's three to the state-root class (`ANTHROPIC_CONFIG_DIR`, `XDG_CONFIG_HOME`, `CLAUDE_CODE_TMPDIR`, `CLAUDE_TMPDIR`, `CLAUDE_JOB_DIR`, `CLAUDE_MEMORY_STORES`), the ambient-credential class, the endpoint-redirect class, and **`ANTHROPIC_PROFILE`** — which selects a different credential store entirely. | **G-B0-8** (new) |
| **W22** | `skill-agents-management` | **New (N19).** qwen's plugin declares `HomeEnvVar: ""` (`pkg/agentic/systems/qwen/qwen.go:121`); qwen reads `QWEN_HOME`. Set it, and compose `QWEN_RUNTIME_DIR` too. Blocked on Q18 only for the version pin, not for the fix. | **G-B0-9** (new) |

W1 and W3 are the load-bearing pair. W6 is deliberately a no-op entry.
**W20, W21 and W22 are round-four additions and all three are B0 prerequisites.**
They are the direct consequence of §2.4's sweep: the enumeration method that
found `CODEX_SQLITE_HOME` found six more codex inputs, a second claude credential
store and a qwen state root, none of which any previous round composed.

### 11.2 Free investigations — unblock later decisions at no risk

**Round four ran every read on this table that was runnable. Status is now
recorded per row, because a table of free work with no status column is how
three rounds each left the same item unrun.**

| # | Work | Settles | Status |
| --- | --- | --- | --- |
| **W8** | Read the control flow around codex's keyring→file fallback | Q12 (half of it) | **DONE** (round four, N16). Superseded by W19 and answered with it |
| **W9** | Audit codex's `cli_auth_credentials_store` accepted values, incl. "encrypted auth storage" | Q14 | **DONE** (round four, N17). Four values, not three; the encrypted backend is a separate axis |
| **W10** | Audit whether `sqlite_home` defaults inside `CODEX_HOME` | Q15 | **DONE** (round four, N13). **It defaults inside.** Ranked first for a full round and unrun for that whole round |
| **W11** | Audit the qwen CLI for a state-root variable and credential store | Q10, epic AC8 | **DONE** (round four, N19). `QWEN_HOME` exists; §5.3 rewritten; W22 scheduled |
| **W12** | Read `setup.sh`'s write set; assert disjointness from the proposed auth root | Q7 | **DONE** (round four, N18). Disjointness **fails** — the root is nested. Replaced by H14 |
| **W13** | Survey whether any legitimate operator setup sets `CLAUDE_SECURESTORAGE_CONFIG_DIR` or `CLAUDE_CODE_CUSTOM_OAUTH_URL` | Q11, **B0 prerequisite** | **PARTLY DONE** (round four, N20a). This host only. Weak evidence about other setups; stays open at reduced scope |
| **W14** | Run the codex pin test with negative variants against every installed codex version | Q4 | **Documentary half done (N21): `#[default] File` is stable across seven `rust-v*` tags.** The harness half is unrun — only one codex version is installed, so it cannot run as written here |
| **W15** | Synthetic two-writer race against the `.storage-write` lock | Q6 | Unrun. Synthetic experiment. §5.1 already answers it in principle: an advisory lock cannot bind a writer that does not take it |
| **W16** | Shim run with symlinked and trailing-slash spellings of one synthetic dir | Q5 | **Claude documentary half done (N22): the derivation applies `.normalize("NFC")` only — no resolve, no trailing-slash trim, no symlink dereference.** The claude harness half is unrun. Codex is partly answered as a side effect of N13's reads: `find_codex_home` **canonicalizes** `CODEX_HOME` and errors if it cannot (`utils/home-dir/src/lib.rs:43-47`) |
| **W17** | Read both vendors' published terms, help-centre and product documentation | Q2b-a, Q2b-b, Q2c | **DONE** (round two, §2.3) |
| **W18** | Re-read `help.openai.com/en/articles/20001068` and `openai.com/policies/terms-of-use` from a network that is not 403-blocked | Q2b-b | **Re-attempted round four: still `403` on both.** A failed read, not an absence. Cannot be done from this host |
| **W19** | Resolve the doc-versus-binary conflict on codex's `keyring` fallback | Q12, supersedes W8 | **DONE** (round four, N16). **There was no conflict** — the strings belong to `AutoAuthStorage`. N12a withdrawn |
| **W23** | **New (N20).** Audit the enrolment path of claude's file-based named-profile credential store: which verb, if any, writes `configs/<name>.json` with `authentication.type = "user_oauth"`, and whether a consumer subscription can be materialised that way | **Q17**, and potentially a cleaner **B1-claude mechanism** | Unrun. Free |
| **W24** | **New (N15).** Establish whether codex's six auth-affecting env overrides are gated anywhere in the call graph, and whether `enable_codex_api_key_env` is on by default for a subscription login | **Q16**, and it is a **B0 prerequisite** | Unrun. Free for the read half |

**Seven rows moved to DONE in one round.** W17 was already complete. The six that
were not — W8/W9/W10/W11/W12/W13 — had been on this table since round two,
every one of them labelled free, and none of them had been run. That is the
finding §2.4 exists to record, and this status column is the mechanism that
prevents it recurring: **a free-work table without a status column reports
intention as if it were evidence.**

**Round four corrects the sentence that used to sit here.** It read: *"W13 is
the only one B0 depends on."* **That was false.** B0 now depends on **W13, W20,
W21, W22 and W24** — the ambient-input closures for codex, claude and qwen, and
the gating question behind the codex credential overrides. The false version was
derived by sizing each runtime's ambient-input set from its *credential-namespace*
input count (N1's three, N5's one), which is the precise-where-derived /
wider-where-used shape §12.4 tabulates, committed on the axis nobody enumerated.

**What is left on this table that could still move something.** **W23** is the
highest-value unrun read now, and it is on the *claude* side, not the codex one:
a second vendor-shipped credential store with a `user_oauth` type would be a
different and possibly better mechanism for B1-claude than the Keychain route the
design assumes. **W24** ranks second and is a B0 prerequisite. Nothing left on
this table can settle B1-codex — round three believed W10 and W19 could, both
have now been run, and **both came back favourable without deciding anything**,
because neither was ever about entitlement. Only Q2 is, and Q2 needs an account.

W18 remains worth doing for evidence quality — it would make N11 primary — but
**it can no longer move anything**, because N11 is demoted to corroboration and
no verdict rests on it. Recorded that way so nobody spends an unblocked network
on it expecting a decision to follow.

W14, W15 and W16 are synthetic experiments rather than reads, and are listed
because they are cheap, not because they are scheduled.

### 11.3 B1 — held, unstarted, not to be scheduled

Recorded so the scope is known if §12's conditions are met. `bk6owf` §9 holds
the full prerequisite tables; this is the summary.

**The split verdict changes what this table would cost, and only for Claude.**
Two entries exist purely to defend an undocumented derivation, and §7.3 removes
that need on the Claude side:

- **H3** (version allowlist) is **not required for B1-claude**. The property it
  guards is documented (N8). **Round three: nor is it required for B1-codex on
  the packaged `file` branch**, for the same reason — that branch has no
  derivation and its isolation property is documented (§7.3.1). What it must
  still guard on codex is the store **selector**'s undocumented default
  (§7.3.3), and **H13 removes even that** by setting the key explicitly. Round
  two cited H3 as one of the reasons B1-codex was a no-go; that reason is
  withdrawn.
- **H4** (namespace pin tests) shrinks for Claude to asserting the *documented*
  property — different dir, different entry — rather than pinning a service-name
  formula per release. For codex it stays as written.
- **H5** (refusal on `CLAUDE_SECURESTORAGE_CONFIG_DIR` and
  `CLAUDE_CODE_CUSTOM_OAUTH_URL`) **stays exactly as written and becomes more
  important**, because it is now the whole of what carries the undocumented
  namespace inputs on Claude, and it must fail closed.

Nothing here is started, on either provider, and H1–H12 are not scheduled by
this ADR.

| # | Work |
| --- | --- |
| **H1** | Profile store: record schema, state roots, P1 bijection, immutability |
| **H2** | Custodian-class total function + C1 + C2, with narrowing negative tests |
| **H3** | Version gate as an **allowlist** (N6), refusing rather than warning, per runtime |
| **H4** | Namespace pin tests, both runtimes, both codex branches, with every negative variant |
| **H5** | Refusal conditions: `CLAUDE_SECURESTORAGE_CONFIG_DIR` **and** `CLAUDE_CODE_CUSTOM_OAUTH_URL` (N1) |
| **H6** | `auth` verb group: `enroll`, `list`, `status`, `logout`, `remove`, `doctor`, `pin verify` |
| **H7** | Selector resolution with A1 (ambiguity refuses, never a heuristic) |
| **H8** | Custody detection, three-valued, `stat`-only, D1 |
| **H9** | Retire: `logout`/`remove` split, tombstones, ordering, R1 |
| **H10** | 6.4's greppable-absent CI assertions and their negative tests |
| **H11** | `Capabilities` exposes the version allowlist and derivation id |
| **H12** | `[agents.<runtime>.auth].profile` parsing with typed refusal on unknown alias |
| **H13** | **New (§7.3.3).** Set `cli_auth_credentials_store` **explicitly** in each codex profile's `CODEX_HOME/config.toml` at enrol, and record it, rather than inheriting a packaged default the vendor documents nowhere. Removes codex's last version-gate dependency. `bk6owf` §3.3 already carries a `store_selector` field but *observes* it rather than *setting* it. **Round four:** the same profile-owned `config.toml` should also pin **`sqlite_home`**, because `config.toml` outranks `$CODEX_SQLITE_HOME` (N13) — one file closes both hazards. And the vocabulary it validates against must be **four** values including `Ephemeral`, not three (N17) |
| **H14** | **New (N18).** Q7's proposed disjointness assertion **cannot pass as written**: the auth root is a child of the installer-managed config dir on all three platforms. Either move the auth root out from under `CONFIG_DIR`, or assert that the installer's write set never grows a **recursive** operation on it. The negative that gives it meaning: the test must fail if a future `rm -rf "$CONFIG_DIR"` or `Remove-Item -Recurse` is added, not merely if a new file appears |

**Status: unstarted by design. Do not begin H1–H13 before §12 is satisfied.**
For B1-claude that means Q1 has actually been run, not that it is expected to
pass. **For B1-codex, round three corrected what this sentence used to say, and
round four ran the item it pointed at.** It previously read *"it means the
vendor has shipped something that does not exist today"* — that followed from
the no-go, and the no-go is withdrawn. Round three then said §12.2's items 1–3
were **ours and free**, and that one of them (Q15's `sqlite_home` half) could
still close B1-codex on a real ground. **Round four ran it: Q15 came back
favourable (N13), and so did Q12 (N16) — and neither settled anything**,
because what remained open was never the mechanism, it was entitlement. §12.2's
items 1–3 are now **discharged**: two reads, both favourable and neither
decisive, plus H13 as an unstarted design change. The only thing left that
could close B1-codex is **Q2**, and Q2 needs a second account. Neither provider
is scheduled by this ADR.

---

## 12. Conditions for B1, per provider

Round one listed one set of six conditions for an undivided B1. The documentary
read split them per provider. **Round three then rewrote the codex half
entirely**, because the conditions round two wrote were derived from a no-go that
did not survive: they were all things OpenAI would have to do, and most of what
would actually settle B1-codex turns out to be ours and free. Neither half is
scheduled; both are unstarted.

### 12.1 B1-claude — conditional go. One condition remains.

Entitlement is no longer among them. Anthropic documents the mechanism (N8),
documents the multiple-accounts use case with a worked example (N9), and its
Consumer Terms restrict sharing rather than count (N10).

1. **Q1 — server-side durability, observed.** Enrol a second Anthropic account
   under its own state root, leave the first live, and verify **both** still work
   after 24 h. The namespaces are provably and now documentedly disjoint, so the
   live account is not at risk. This is the only blocking condition, and it is an
   experiment rather than a question about permission.
   **It is not a formality.** If the second enrolment invalidates the first
   server-side, B1-claude is closed regardless of what the documentation says,
   and this ADR would rather find that out in one 24-hour observation than in
   production.
2. **W13** complete, so B0's environment composition is known not to break a
   legitimate setup. Free, and already a B0 prerequisite. **Round four: partly
   done and honestly short of complete** — the survey ran on this host only
   (N20a), and one host is weak evidence about what other setups do. It now also
   has to cover `ANTHROPIC_CONFIG_DIR` and `ANTHROPIC_PROFILE`, which round four
   found select a **different credential store** (N20).
3. **Q13 answered** — whether the threat model includes local write access to the
   auth root — because it decides whether `bk6owf` C2 is the right stopping point
   or the profile record must be sealed. Free; it is a decision, not research.
4. **Q17 read (W23) — new in round four, and it is a condition on the *design*,
   not on the go.** Claude Code 2.1.248 ships a second credential store that is
   *already* profile-named: `ANTHROPIC_PROFILE` selects
   `configs/<name>.json` + `credentials/<name>.json`, and it accepts
   `authentication.type = "user_oauth"` (N20). B1-claude's whole design assumes
   the Keychain route. **If a consumer subscription can be enrolled as such a
   profile, the vendor has already shipped most of B1-claude** — file-based,
   explicitly named, environment-selected, no `sha256` derivation anywhere near
   it. If it cannot, the store is enterprise-federation machinery and the
   Keychain route stands.
   **This is not treated as good news.** No verb that writes those files was
   found, the subsystem is internally named for workload-identity federation, and
   `~/.config/anthropic` does not exist on this host. It is a **mechanism sighted
   in shipped code with its enrolment path unread** — which is exactly the
   standing this ADR spent three rounds learning not to over-spend. It is listed
   as a condition because building B1-claude's Keychain design *before* reading
   this would risk implementing the harder of two mechanisms.

**Deliberately not conditions any more:**

- *The §7.2 version-gate cost.* §7.3 shows it prices the derivation, and B1-claude
  does not use the derivation. H3 is dropped for this provider.
- *Q2b/entitlement in writing.* Answered by W17 (Q2b-a).
- *Q2c, two billed consumer subscriptions.* Unaddressed by anything published,
  and not required: the vendor's own documented scenario is work plus personal,
  which is a subscription plus an organization seat or a Console account, both
  explicitly supported. An operator who holds two accounts they are entitled to
  can proceed; one who would have to buy a second Pro to find out is making a
  purchasing decision this ADR does not make for them.

### 12.2 B1-codex — held, unestablished. What would settle it.

**Round three replaced this section.** Round two titled it *"no-go — what would
have to change"* and opened with *"Every item is something **the vendor** would
have to do, which is what distinguishes a no-go from a hold."* Both halves are
withdrawn: it is a hold, not a no-go (§7.3.1–7.3.3), and **most of the list is
ours**. That inversion is the practical consequence of the correction — a reader
who believed OpenAI had closed this would not have run any of items 1–3, and all
three are free.

**Round four ran items 1 and 2. This is what they returned.**

1. **Q15 — does `sqlite_home` default inside `CODEX_HOME`? ✅ DONE. It does.**
   Precedence `config.toml` › `$CODEX_SQLITE_HOME` › `$CODEX_HOME`
   (`config_toml.rs:329-331`, `core/src/config/mod.rs:3918-3923`,
   `state/src/lib.rs:106-107`, at `rust-v0.150.1`; N13). Round three called this
   *"the substantive risk"* and staked the section on it: *"If it defaults
   inside, the largest open objection goes away."* **It defaults inside, and the
   objection is gone.** Two credential-isolated codex profiles do not share a
   state database. Cost: three `gh api` calls.
   **It did not close B1-codex, and it was never going to** — it was a mechanism
   question, and what is open is entitlement. Recorded because that mismatch is
   the round-four lesson, not a footnote to it.
2. **Q12 — is the keyring→file fallback gated on `auto`? ✅ DONE. Yes.** Both
   fallback legs live in `AutoAuthStorage`; `Keyring` dispatches to a backend
   with no file leg (`storage.rs:431-449, 511-545`; N16). **N12a's
   doc-versus-binary conflict is withdrawn — there never was one.** Also
   favourable, also not decisive.
3. **H13 — set `cli_auth_credentials_store` explicitly per profile** instead of
   inheriting a default the vendor documents nowhere. Still the right move, and
   **round four narrows what it is worth**: the default is `#[default] File` in
   first-party source at the installed tag (N17), so this now guards against a
   *silent release-to-release change* rather than against an unknown. The same
   profile-owned `config.toml` should pin `sqlite_home` too, since it outranks
   the environment variable. A design change, not research; **unstarted**,
   recorded in §11.3.

**So the "ours and free" list is discharged.** Two reads run, both favourable,
neither decisive; one design change left, and it is unstarted rather than
unknown. Round three's framing — *"most of what would settle it is ours and
free"* — was **wrong in a way worth recording**: those items were ours and free,
but they were not what would settle it. The only thing that settles B1-codex is
**Q2**, and Q2 needs a second account.

**Not ours, and none of them blocking:**

4. **Q2 — server-side durability of a second concurrent codex enrolment.** The
   codex counterpart of Q1. Needs a disposable ChatGPT account. It was never run
   because round two believed the vendor had answered; the vendor has not
   answered (§2.3), so Q2 is a genuine open experiment again — though items 1–3
   should come first, because they are free and one of them could close the
   question without it.
5. **OpenAI documents multi-account for the Codex CLI**, or ships switching, or
   documents `CODEX_HOME` as an account-isolation boundary rather than only as
   the location of `auth.json`. It publishes none of these today (N12), and
   **it publishes no prohibition either** — the sentence round two cited is
   scoped to Codex *desktop* and the native mobile apps (N11) and does not reach
   the CLI, so this condition is *unmet*, not *contradicted*. The natural shape
   already exists as an unmerged community proposal — `--auth-profile NAME` /
   `CODEX_PROFILE` with a per-profile home under `$CODEX_HOME/profiles/NAME`,
   linked from `openai/codex#18806`. If it or an equivalent lands upstream, this
   stops being a question at all. **This would strengthen the case; its absence
   does not close it**, which is the distinction round two collapsed.
6. **The cross-account state-leak reports are resolved** — `#35657`, `#21314`,
   `#16894`, `#26628`, `#22419`, `#39698`. Stated carefully, because this is the
   same widening trap as N11 — and **round three's own statement of it was itself
   too narrow, in the direction that made adverse evidence look less relevant.**
   It said these were reports about the vendor's switching *"on the app and IDE
   surfaces"*. `#22419`'s body says *"After switching accounts in Codex App **/
   Codex CLI**…"* — it names the CLI. The surface list was wrong; the
   load-bearing half of the sentence survives without it: every one of these
   reporters used **the vendor's own account switcher**, not two `CODEX_HOME`
   roots, so none is evidence about the mechanism B1 would use. **At least one
   (`#22419`) names the CLI, and it is still a report about the vendor's switcher
   rather than about two state roots.** They are third-party reports on a
   first-party tracker, not vendor statements. They were part of why item 1 was
   ranked first; item 1 has now been run and came back favourable (N13), which
   does not make these reports go away — it means they concern a mechanism B1
   does not use.
7. **Q4 — a codex version range wider than one point**, if any version-dependent
   path survives H13. Free, and moot if H13 lands.

### 12.3 The honest label

**B1-claude: conditional go — one experiment from a decision, on a mechanism the
vendor documents and a use case the vendor recommends.**

**B1-codex: held, unestablished — not a no-go, and round three withdrew the one
that was written here.** Every reason round two gave failed inspection: N11 is
scoped to Codex *desktop*; §7.2's derivation cost does not reach the packaged
`file` branch; and the premise offered to rescue that cost is contradicted by the
design document it cites.

**Round four ran the reads round three said were left, and the list shrank
again.** The clause that used to end this paragraph — *"one genuinely open
isolation question (`sqlite_home`, Q15) that is a free source audit nobody has
run"* — is **discharged**: the audit was run, and `sqlite_home` defaults inside
`CODEX_HOME` (N13). Q12 was run too and also came back favourable (N16). What is
left is **two** things: no vendor documentation of multi-account as a use case,
and entitlement. Entitlement for the Codex CLI is **unestablished** — a third
status distinct from round one's "no evidence" and round two's "closed by
published position". Neither vendor has said yes; neither has said no.

**And the shape of the hold has changed, which is the thing a later reader most
needs.** For three rounds B1-codex was held behind work *we* could do. It is now
held behind exactly one thing we cannot do from a document: **Q2, a second
account.** Every free read is spent. Nothing on §11.2's table can move it.

Both halves are unstarted. **They are unstarted for the same reason — the
evidence is not in — and that is the honest symmetry**, rather than one being
gated and the other closed. The difference between them is real but narrower than
two rounds of this document claimed: Anthropic documents the workflow, names the
use case and documents what the config dir carries; OpenAI documents the
credential's location and stops.

The epic's headline capability — switching among multiple Claude *and* Codex
accounts — is **not deliverable as one feature today**. That is different from
saying either half is closed, and after three rounds it is the only formulation
that has survived.

### 12.4 Three rounds, one defect, three scales

Recorded because the pattern is the transferable part and it recurred after being
diagnosed, twice.

| Round | The claim | What it actually was |
| --- | --- | --- |
| 1 | *"The question that decides the product has produced **no evidence at all**."* | A **read nobody performed**, reported as an absence. The tell was in the ADR's own unknowns table: that question's cost column said *"a decision, and it is not the agent's to make"* while eight sibling rows were correctly filed as free source audits. |
| 2 | *"One of its two halves is closed by the vendor's own published position."* | A **quote used at a wider scope than it has**. `desktop` was reproduced once, in the evidence section, and dropped in all seven places the quote was *used* — including the README and the logbook. The tell was that the narrowest word in the quote appeared exactly once in 1336 lines. |
| 2 | *"Codex pays §7.2 in full… **overdetermined** rather than marginal."* | The round's **own best distinction, applied to one provider and withheld from the other**. Property-versus-derivation rescued claude and was declined for codex in a single sentence that three facts in the same document contradicted. The tell was that the distinction was never run against the case it would hurt. |
| 3 (review) | *"§8 forces a codex B1 profile onto `keyring`."* | A **reviewer-supplied premise that does not survive being checked**, in the direction that would have preserved the existing verdict. `bk6owf` §6.2 classifies codex-on-`file` as `active` in a row written to prevent exactly that reading. Recorded here because a correction carries the same standard as a claim. |
| 3 | *"W10 is the highest-value item on this table"* · *"the substantive risk and it should be first"* · *"a free source audit nobody has run"* · *"W13 is the only one B0 depends on."* | A **free read the document itself ranked first, and did not run** — for the third round running, and this time after diagnosing the pattern in §12.4. The tell was that all three superlatives were about the same unrun item and none of them was a result. Running it took three `gh api` calls, closed the risk favourably, and surfaced `CODEX_SQLITE_HOME` — a second ambient input that falsified the completeness claim in the next sentence, about the option this ADR says **adopt now**. |
| 4 | *(this round)* — | The counter-shape, recorded so it is not over-learned: **six free reads were run and four moved something, but none moved the verdict they were ranked against.** Round three ranked W10 and W19 as what could settle B1-codex; both returned favourable and neither settled it, because both were **mechanism** questions and what is open is **entitlement**. A free read ranked first should be run *and* checked against what it could actually decide — cheapness is a reason to run it, never a reason to believe it is load-bearing. |

**The shape is one thing each time: a statement that is precise where it is
derived and wider where it is used.** The check that catches all four is the same
— take the claim back to its source, at the source's own scope, before spending
it. It cost this decision two rounds to learn and it recurred inside the round
that diagnosed it.

**What did not move across four rounds:** A permanent NO-GO, B0 GO, C GO.
**What moved:** B1-claude NO-GO → CONDITIONAL GO (round two, on evidence).
B1-codex NO-GO → NO-GO-for-a-different-reason → **HELD** (round three, on the
reasons failing rather than on new permission). **qwen not-modellable →
provisional-with-a-mechanism** (round four, on the free audit that three rounds
listed and none ran — `QWEN_HOME` exists, and the empty `HomeEnvVar` was our
defect, not the vendor's limit).

**And what moved on the adopted option, which is the part that costs something.**
B0's prerequisite list went from one item to five, because rounds one to three
sized each runtime's ambient-input set from its credential-namespace input count.
B0 is still GO — nothing found weakens the case for composing the child
environment, and the new inputs are further arguments for it — but *"ready to
schedule"* meant a smaller job than the document claimed.

## 13. Corrections this ADR makes to its inputs

Listed together so a later reader does not have to reconstruct them.

| Input | Claim | Correction |
| --- | --- | --- |
| `3moaky` decision summary item 4 | Claude has no per-account selector on macOS; `CLAUDE_CONFIG_DIR` relocates credentials on Linux/Windows only; multi-account is not an isolation boundary on macOS. | **Refuted** by `1g880w`: `CLAUDE_CONFIG_DIR` namespaces the macOS Keychain **service** name. The audit's other findings stand. Its "forbidden" line about `CLAUDE_CONFIG_DIR` survives only as a statement about *support*, not mechanism. |
| `3moaky` implication "force `cli_auth_credentials_store = \"keyring\"`" | Keyring is the safe codex store. | **Weakened** by N4: keyring has a fallback path to file storage on both load and save. Forcing keyring may not avoid plaintext, and the packaged default is `file` anyway. |
| `1g880w` unknown: is `OAUTH_FILE_SUFFIX` ever non-empty in a shipped build? | Listed as free, not run. | **Settled here: yes** (N3), and it is operator-reachable in the prod binary via `CLAUDE_CODE_CUSTOM_OAUTH_URL`. |
| `bk6owf` §5.2 Invariant L1 | "When no profile is selected, the launch writes the *default* home explicitly." | **Corrected to L1′** (N2). For claude this would move every default launch into an empty namespace. The default is expressed by *removing* the variable. |
| `bk6owf` §7.1 claude range `2.1.234 – 2.1.248` | Stated as an interval on four verified points. | **Sharpened** (N6): 2.1.235 now verified, making five; the interval still asserts ten unverified versions. The gate should be an allowlist. |
| `bk6owf` §7.3 refusal set | `CLAUDE_SECURESTORAGE_CONFIG_DIR` only. | **Widened** (N1) to include `CLAUDE_CODE_CUSTOM_OAUTH_URL`. |
| `bk6owf` §3.3 codex store vocabulary | `file \| keyring \| auto`. | **Incomplete** (N4/Q14): an "encrypted auth storage" backend exists in 0.150.1. |
| `bk6owf` Q12 | Codex keyring failure posture unknown. | **Narrowed and then sharpened adversely** (N4, N12a): a fallback path exists on both legs, and the vendor documents a fallback for `auto` **only**. The two sources conflict, so `keyring` cannot be relied on to mean "never plaintext". **Superseded below (N16, round four): the conflict was never real.** |
| **This ADR, previous round**, §0 / §4 / §5 / §12 | "The question that decides the product has produced **no evidence at all**"; entitlement is one undivided unknown covering "either vendor". | **Refuted, and it was the load-bearing claim.** It described a **read that had not been performed**, not an absence. Free published first-party material exists on both providers and speaks in both directions (§2.3). Restated per provider throughout. |
| **This ADR, previous round**, §4 Q2b | "What settles it: a human reading the subscription terms. Not an experiment. Cost: a decision, and it is not the agent's to make." | **Misclassified.** Reading published terms and product documentation is a **free documentary read**; eight sibling rows in the same table were filed that way. Only the judgement about whether to rely on the documents was ever the operator's. The misfiling is why the single load-bearing input went unread. Split into Q2b-a (answered), Q2b-b (403-blocked) and Q2c (residual); the read is recorded as W17. |
| **This ADR, previous round**, §5.1 | Claude's namespace mechanism is undocumented; Anthropic documents `CLAUDE_CONFIG_DIR` as Linux/Windows credential-file relocation only. | **Refuted** (N8, N9). `code.claude.com` documents the macOS Keychain keying *and* names running multiple accounts side by side as the variable's use, with a worked alias. |
| **This ADR, previous round**, §7 | The recurring version-gate cost sinks B1 on cost-benefit, for both providers, independently of entitlement. | **Scoped** (§7.3). It prices the `sha256[:8]` *derivation*; B1 needs the *property*, which on Claude is documented. The cost does not apply to B1-claude. **Further corrected in round three** — see the next two rows: it does not apply "in full" to codex either, and it is not independent of the plaintext policy. |
| **This ADR, round two**, §0 / §5.2 / §9 / §12.2 / §12.3 / `README.md` / `LOGBOOK.md` 1140 | B1-codex is "closed by the vendor's own published position"; "Vendor position on multi-account: **Negative**"; "OpenAI publishes that Codex does not support account switching". | **Refuted, and it is the same defect one notch milder than round one's.** The supporting quote is scoped to *"**Codex desktop** or the native ChatGPT mobile apps"* — two GUI surfaces, not the CLI — and *"not yet"* is a roadmap note, not a prohibition. The word `desktop` was reproduced once, in the quote, and dropped in all seven places the quote was **used**, including both durable operator-facing files. N11 is demoted to corroboration; entitlement for the Codex CLI is **unestablished**, not negative (§2.3, §12.3). |
| **This ADR, round two**, §7.3 final bullet | "Codex pays §7.2 in full. One verified version, an undocumented derivation, and refusal on the very next upgrade" — making the codex no-go "**overdetermined** rather than marginal". | **Refuted** (§7.3.1). `cli_auth_credentials_store = "file"` is the packaged default (ledger 11); on that branch the credential is `auth.json` inside `CODEX_HOME` — **vendor-documented** (ledger 20) — and the `cli\|sha256(home)[:16]` derivation **does not exist**. §7.2's pin does not reach codex's default branch at all. "Overdetermined" is **withdrawn**, "three independent reasons" is withdrawn, and with both entitlement and cost retired **the codex no-go had nothing left carrying it** — hence the hold. The distinction round two used to rescue claude was simply never run against codex. |
| **CR-TASK-260720-3gcfd1-2 §3**, reviewer-supplied | "There is a premise that would rescue the sentence… codex's `file` store *is* the plaintext `auth.json`, and §8 ends with 'a `degraded:plaintext` profile refuses to launch' — so a codex B1 profile would have to use `keyring`." Offered as "a genuinely better codex pillar". | **Refuted by the document it cites** (§7.3.2). `bk6owf` §6.2 classifies codex-on-`file` as **`active`** — *"this is the vendor's packaged default custody, not a fallback"* — and states the reason: *"Calling codex-on-`file` `degraded:plaintext` would refuse every default codex enrolment for a hazard that did not occur."* `degraded:plaintext` means **unexplained** plaintext. §8 does not force codex onto `keyring`, and adopting the premise would have preserved a verdict on a second false reason. Recorded because a reviewer-supplied premise carries the same standard as any other input. |
| **This ADR, round two**, §7.3 / §11.3 H3 | The codex version gate is a derivation cost, and it is one of the reasons B1-codex is a no-go. | **Relocated, not removed** (§7.3.3). The cost is real but it guards a **third** constant neither round named: `cli_auth_credentials_store`'s **default value**, which the vendor documents nowhere (ledger 30) and which is known only from byte inspection at one version (ledger 11). `bk6owf` §7.1 had already identified this and both rounds walked past it. It is **escapable** by setting the key explicitly (H13), which is the same move §7.3 makes for claude. |
| **This ADR, rounds one–three**, §5.2 / §11.2 / §9 | Codex has *"Namespace inputs: **One** (N5)"*; B0 is *"cleaner than claude: one input"*; **"W13 is the only one B0 depends on."** | **Refuted, and this is the round-four blocker** (N14, N15). N5 is precise about the **credential namespace** and was spent at the scope of **state isolation and B0's prerequisites**. Codex reads **eight** ambient inputs affecting its state root, credential or OAuth endpoints, including `CODEX_SQLITE_HOME` — which appeared nowhere in this ADR, `bk6owf`, the custody research, `README.md` or `LOGBOOK.md`. Two are materially worse than a count implies: `CODEX_API_KEY` outranks *"any other auth method"* by the vendor's own comment, and `CODEX_REFRESH_TOKEN_URL_OVERRIDE` has **no allowlist** where claude's equivalent throws. B0 gains W20/W21/W22/W24. |
| **This ADR, rounds one–three**, §5.1 | Claude's namespace surface is the three env inputs of N1. | **Widened** (N20). Those three are the *derivation's* inputs. The installed bundle has **174** `process.env.CLAUDE_*`/`ANTHROPIC_*` read sites, and among them a **second, vendor-shipped, file-based named-profile credential store** — `ANTHROPIC_CONFIG_DIR`/`ANTHROPIC_PROFILE` → `configs/<n>.json` + `credentials/<n>.json`, accepting `user_oauth`. Unknown to three rounds of this document. Its enrolment path is unread (**Q17**), so it is recorded as a mechanism sighting, not a verdict. |
| **This ADR, rounds one–three**, §0 / §5.2 / §6 / §9 / §10 / §11.2 / §12.2 / §12.3 / §15 / `README.md` | `sqlite_home` has no documented default; two credential-isolated codex profiles may share a state DB; *"the substantive risk"*; *"a free source audit nobody has run"*. | **Answered, favourably** (N13). `sqlite_home` defaults **inside** `CODEX_HOME`; precedence is `config.toml` › `$CODEX_SQLITE_HOME` › `$CODEX_HOME`. The risk described as open across these ten sites is **closed**, and it cost three `gh api` calls. (Nine were caught in round four; §6 was the tenth, missed by the same sweep and corrected there directly.) B1-codex's verdict does not move, because the objection was never what was carrying it. |
| **This ADR, rounds two–three**, §5.2 / N12a / Q12 | Docs promise a keyring→file fallback for `auto` only, the binary carries fallback strings on the keyring path, *"the two sources disagree"*, so `keyring` cannot be relied on to mean "never plaintext". | **Withdrawn — there was no conflict** (N16). Both fallback legs are in `impl AuthStorageBackend for AutoAuthStorage` and nowhere else; `Keyring` dispatches to a backend with no file leg. The strings N4 found by byte inspection belong to `auto`'s implementation. A correctly-labelled unknown *"a narrowing, not an answer"* was nevertheless carried as adverse for two rounds while the control flow sat unread. |
| **This ADR, round two**, §5.2 / `bk6owf` §3.3 | The codex store vocabulary is `file \| keyring \| auto`, and an unmodelled "encrypted auth storage" exists (Q14). | **Completed** (N17). Four values: `File` (`#[default]`), `Keyring`, `Auto`, **`Ephemeral`** — the last consulted *before* any persisted credential and modelled by neither document. The "encrypted auth storage" is `AuthKeyringBackendKind::{Direct,Secrets}`, a **separate axis** not selectable through `cli_auth_credentials_store`, so Q14's actual answer is **no**. Any custody classifier total over three values is not total. |
| **This ADR / `bk6owf`**, §5.3 / §12 row 7 | qwen is **not modellable**: `HomeEnvVar: ""`, so there is no state root; *"the epic's qwen ambition has no mechanism"*. | **Refuted** (N19). qwen reads `QWEN_HOME` and resolves `oauth_creds.json` through it (`storage.ts:193-203, 640-642`), plus `QWEN_RUNTIME_DIR`. The empty declaration is a **defect in `skill-agents-management`**, not a fact about qwen. Q10 was listed as free in every round and run in none. qwen stays *provisional* per AC8 — nothing establishes its entitlement, refresh or revocation — but *provisional-with-a-mechanism*. The parallel claim about muse, gemini, agy and pi is **withdrawn as unaudited**, on the ground that the identical claim about qwen died to one read. |
| **`bk6owf` / this ADR**, Q7 | The proposed auth root `~/Library/Application Support/agents-infra/auth/` is outside everything `setup.sh` manages; the gate is to *"assert disjointness"*. | **Refuted** (N18). It is a **child** of the installer-managed `CONFIG_DIR` on macOS (`setup.sh:69`), Windows (`setup.ps1:13`) and Linux, per `source_dir.go:387-404`. File-level disjoint today — only `install.json` is written there — but directory-level nested, so a future recursive uninstall would take credentials with it. The proposed assertion cannot pass as written; replaced by **H14**. |
| **This ADR, round three**, §12.2 item 6 | The codex cross-account leak reports are *"about the vendor's own switching implementations on the app and IDE surfaces"*. | **Corrected** (F2, §12.2). `#22419`'s body reads *"After switching accounts in Codex App **/ Codex CLI**"* — it names the CLI. The surface enumeration was narrower than its own sources, in the direction that made adverse evidence look less relevant. The load-bearing half survives unchanged: every reporter used the **vendor's switcher**, not two `CODEX_HOME` roots. |
| **This ADR, ledger 23** | `gh api search/issues 'repo:openai/codex is:pr auth-profile OR CODEX_PROFILE in:title'` → *"zero results"*. | **Re-labelled, not refuted** (N19's method note). Round four found `gh api search/code` returning `total_count: 0` for a constant that demonstrably exists in the target file. **A zero from GitHub search is a failed or partial read, not an absence** — this ADR's own standard for `curl` 403s, applied to a tool whose zero looks like a result. Row 23's material claim is unaffected (no such PR is merged) but its zero is now labelled as what it is. |

---

## 14. Evidence ledger

All commands read-only. Gathered on this machine 2026-08-31. No `security`
invocation was made in this task; no credential value was read.

| # | Establishes | Method | Result |
| --- | --- | --- | --- |
| 1 | Installed versions | `claude --version`, `codex --version` | 2.1.248, codex-cli 0.150.1 — same as the versions `1g880w` and `3moaky` audited |
| 2 | Release cadence | `ls -la ~/.local/share/claude/versions/` | 5 builds in 11 days; 14 upstream patch versions across the window |
| 3 | Service-name construction, 2.1.248 | byte window around `CLAUDE_SECURESTORAGE_CONFIG_DIR` | `ok()`, `Gv()`, `ME()`, `eq="-credentials"` as quoted in N1 |
| 4 | Same construction at 2.1.235 (N6) | same technique | `rte()`, `tte()`, `rX()` — identical semantics, minified names differ |
| 5 | Suffix is empty only when `CLAUDE_CONFIG_DIR` is unset/empty (N2) | source read of `t = e!==void 0 ? !e : !process.env.CLAUDE_CONFIG_DIR` + `1g880w`'s proven live item having no suffix | Derived, not run; the run half is `1g880w` evidence #5/#6 |
| 6 | `OAUTH_FILE_SUFFIX` values (N3) | byte windows around `OAUTH_FILE_SUFFIX` in all 5 builds | `""` (prod), `-local-oauth`, `-staging-oauth`, `-custom-oauth` |
| 7 | Suffix selector and its allowlist (N1) | readable source window in 2.1.247 | `fileSuffixForOauthConfig`, `getOauthConfig`, `ALLOWED_OAUTH_BASE_URLS` = 3 fedstart/staging endpoints; throws otherwise |
| 8 | Channel is hardcoded prod | `function s(){return"prod"}` / `function _nu(){return"prod"}` | 2.1.235 and 2.1.247; `-local`/`-staging` unreachable without patching |
| 9 | `CLAUDE_CODE_CUSTOM_OAUTH_URL` present in all 5 builds | `LC_ALL=C grep -ao 'CLAUDE_CODE_CUSTOM_OAUTH_URL' <build> \| wc -l`, per build | **10 in every build** (2.1.234, .235, .236, .247, .248). **Corrected**: the previous round recorded 7,7,7,8,8, which does not reproduce. The builds are Mach-O arm64 executables, so a `grep -c` without `-a` returns exit 1 and no count at all; the method above is the one that re-runs. The material claim — present in all five — is unaffected |
| 10 | Codex service name is a single literal (N5) | `grep -c 'Codex Auth'` + byte window | exactly 1 occurrence, no env-derived variation; `cli\|` likewise 1 |
| 11 | Codex packaged default custody | byte window at the fixed-defaults block | `cli_auth_credentials_store = "file"`, `mcp_oauth_credentials_store = "auto"` |
| 12 | Codex keyring→file fallback (N4) | byte window in `login/src/auth/storage.rs` strings | `failed to load CLI auth from keyring, falling back to file storage:`, `failed to save auth to keyring, falling back to file storage:`, `failed to remove CLI auth fallback file:` |
| 13 | Codex "encrypted auth storage" backend (Q14) | same window | load / deserialize / write error strings; `CODEX_AUTH should be a valid secret name` |
| 14 | Codex config keys outside `CODEX_HOME` (N7, Q15) | config-key string table | `sqlite_home`, `log_dir`, `history`, `forced_login_method`, `forced_chatgpt_workspace_id` |

| **15** | Anthropic documents the macOS Keychain keying (N8) | `curl -sSL https://code.claude.com/docs/en/authentication.md`, HTTP 200; grep `CLAUDE_CONFIG_DIR` | one match, line 178: *"…keys the macOS Keychain entry to that directory too, so a session with a different `CLAUDE_CONFIG_DIR` reads a different entry."* |
| **16** | Anthropic documents multiple accounts side by side (N9) | `curl -sSL https://code.claude.com/docs/en/env-vars.md`, HTTP 200; grep `CLAUDE_CONFIG_DIR` | the `CLAUDE_CONFIG_DIR` row: *"Useful for running multiple accounts side by side: for example, `alias claude-work='CLAUDE_CONFIG_DIR=~/.claude-work claude'`."* |
| **17** | Anthropic terms restrict sharing, not count (N10) | fetch of `anthropic.com/legal/consumer-terms` | *"You may not share your Account login information… You also may not make your Account available to anyone else."* No account-count clause |
| **18** | Anthropic support pages are silent on two concurrent subscriptions (Q2c) | fetches of support articles 8987223, 13325567, 11145838 | 8987223: Claude + Console on one email, operating independently. 13325567 and 11145838: neither addresses multiple personal subscriptions or account switching |
| **19** | Codex documentation is silent on multi-account (N12) | **Round three re-read from the markdown source**: `curl -sSL https://developers.openai.com/codex/auth.md`, HTTP 200 — same document, no tag-stripping step, and surface-scoped via `ContentModeSwitch` blocks tagged `app`/`cli`/`ide`/`web`. Round two's rendered-HTML read at `https://learn.chatgpt.com/docs/auth` (HTTP 200) was re-run and agrees. Grep for `CODEX_HOME`, `multiple account`, `account switch`, `switching` | `CODEX_HOME`: **exactly 1** occurrence, as the location of `auth.json`. `multiple account`, `account switch`, `switching`: **0 each**. Also captured this round, CLI-scoped and load-bearing for §7.3.1: *"your login details are cached and reused. The CLI and extension share the same cached login details"* and *"Codex caches login details locally in a **plaintext file** at `~/.codex/auth.json`"* |
| **20** | Vendor documents a fallback for `auto` only (N12a) | same page | *"`keyring` stores credentials in your operating system credential store. `auto` uses the OS credential store when available, otherwise falls back to `auth.json`."* Conflicts with N4's keyring-path fallback strings |
| **21** | Codex account switching is unshipped (N12b) | `gh api -X GET search/issues -f q='repo:openai/codex is:issue "account switching" OR "switch account" in:title' -f per_page=100`, then count `items` by `state`. **The `per_page=100` and the explicit count are the method**: the default page size is 30 and a count taken off a truncated or differently-phrased run does not reproduce | **27 returned, 21 open, 6 closed.** **Corrected in round three**: round two recorded 20/19/1, which does not re-run under its own recorded query. Cited issues `#31778`, `#30684`, `#18806`, `#34111` re-verified **open** by direct `gh api repos/openai/codex/issues/N`. Closed set with `state_reason`: `#14730` completed (usage-caching bug in the Codex **app**), `#2833` completed (web-login/API 403), `#17349`/`#19756`/`#15384` duplicate, `#3573` not_planned. **None closed as already-supported** — the material claim, unaffected |
| **22** | Cross-account state survives a codex switch (N12b, Q15) | same query; `gh issue view` on the hits | `#35657`, `#21314`, `#16894`, `#26628`, `#22419`, `#39698`. Third-party reports on a first-party tracker; labelled as such |
| **23** | No merged upstream auth-profile support (§12.2) | `gh api search/issues 'repo:openai/codex is:pr auth-profile OR CODEX_PROFILE in:title'` | **zero results**. The `--auth-profile` / `CODEX_PROFILE` work linked from `#18806` is an unmerged community branch |
| **24** | `help.openai.com` / `openai.com` / `chatgpt.com` unreachable — **failed read, not absence** | `curl -sS -o /dev/null -w '%{http_code}'` with a browser UA, plus the fetch tool | **403** on `help.openai.com/en/articles/20001068`, `.../11369540`, `openai.com/policies/terms-of-use/`, `openai.com/policies/eu-terms-of-use/`, `chatgpt.com/policies/terms-of-use`. Body is OpenAI's own block page. Everything sourced from these origins in §2.3 is labelled second-hand |
| **25** | The account-switching text reproduces independently (N11) | three search-index queries with disjoint phrasing across two rounds, one restricted to `help.openai.com`; plus a round-three re-attempt on the primary origin | All returned the same substantive text: two accounts simultaneously, work/personal, max two per session, and — **at its full scope, which round two dropped** — *"not yet supported in **Codex desktop** or the native ChatGPT mobile apps"*. Primary origin re-attempted round three: `403` on both `.../20001068-use-multiple-accounts-with-account-switching` and `.../20001068`. **Still second-hand; W18 would make it primary. Demoted to corroboration — nothing in the verdict rests on it** |
| **26** | OpenAI ToU free-tier multi-account clause | attempted `curl` (403) and a verbatim-phrase search | **Not corroborated from this host.** Carried as reviewer-supplied and unverified; nothing in this ADR depends on it |

| **27** | The `file` branch's isolation property is **vendor-documented**, and the vendor calls that store plaintext in its own words (§7.3.1–7.3.2) | `curl -sSL https://developers.openai.com/codex/auth.md`, HTTP 200; *Login caching* and *Credential storage* blocks, scoped `app,cli,ide` | *"Codex caches login details locally in a **plaintext file** at `~/.codex/auth.json` or in your OS-specific credential store."* and *"`file` stores credentials in `auth.json` under `CODEX_HOME` (defaults to `~/.codex`)."* Together with ledger 11 (`cli_auth_credentials_store = "file"` is the packaged default) this establishes that codex's default branch has **no derivation on the path** and a **documented** isolation property. Note the vendor's "plaintext" wording does **not** make such a profile `degraded:plaintext`: `bk6owf` §6.2 classifies codex-on-`file` as **`active`**, *"the vendor's packaged default custody, not a fallback"* — see §7.3.2, where the contrary premise is checked and fails |
| **28** | The vendor's own forum thread on Codex account switching has **no vendor reply** (N12c) | `gh api graphql` on `repository(openai/codex).discussion(number:25630)`; read `answer`, and `authorAssociation` on every comment | *Switch Between Accounts*, opened 2026-06-01. **No accepted answer.** Every comment `authorAssociation: NONE` — no maintainer, member or collaborator. A **successful read of a place a vendor statement would appear**, returning none. Comment contents are third-party and are not treated as vendor statements |
| **29** | `log_dir` defaults inside `CODEX_HOME`; `sqlite_home` has no documented default (Q15, half-answered) | `curl -sSL https://developers.openai.com/codex/config-reference.md`, HTTP 200; grep the key table | `log_dir`: *"Directory where Codex writes log files; **defaults to `$CODEX_HOME/log`**"*. `sqlite_home`: *"Directory where Codex stores the SQLite-backed state DB used by agent jobs and other resumable runtime state"* — **no default stated**. Half of Q15 answered, half still open; reported as half rather than rounded to either end |
| **30** | The **default value** of `cli_auth_credentials_store` is documented **nowhere** (§7.3.3) | `curl -sSL https://developers.openai.com/codex/config-reference.md` and `.../codex/auth.md`, both HTTP 200; locate the key and look for a stated default | Config reference: `{ key: "cli_auth_credentials_store", type: "file \| keyring \| auto", description: "Control where the CLI stores cached credentials (file-based auth.json vs OS keychain)." }` — **no default field, no default in prose**. The auth page's only worked example sets `cli_auth_credentials_store = "keyring"`, which is **not** the shipped default (ledger 11 proves the packaged default is `"file"`). So the constant that selects which branch every codex profile uses is known only from byte inspection at one version, and the vendor's sole example shows a different value |

| **31** | `sqlite_home` defaults **inside** `CODEX_HOME`; precedence `config.toml` › `$CODEX_SQLITE_HOME` › `$CODEX_HOME` (N13, Q15 answered) | `gh api repos/openai/codex/contents/<path>?ref=rust-v0.150.1 --jq .content \| base64 -d`, on `codex-rs/config/src/config_toml.rs`, `codex-rs/core/src/config/mod.rs`, `codex-rs/state/src/lib.rs`. Tag `rust-v0.150.1` = `0eb410ad0dd161ea323b05452f978de01cd63430`, matching the installed `codex-cli 0.150.1` | Doc comment `config_toml.rs:329-331`: *"Defaults to `$CODEX_SQLITE_HOME` when set. Otherwise uses `$CODEX_HOME`."* Production resolution `mod.rs:3918-3923`: `cfg.sqlite_home … .or(sqlite_home_env).unwrap_or_else(\|\| codex_home.clone())`. `resolve_sqlite_home_env` at `mod.rs:242-252`; `SQLITE_HOME_ENV = "CODEX_SQLITE_HOME"` at `state/src/lib.rs:106-107`. `log_dir` → `codex_home.join("log")` at `mod.rs:3906-3910`, confirming ledger 29's documented half **from source** |
| **32** | Codex's store vocabulary is four values and the default is `File` in source (N17, Q14 answered) | same method, `codex-rs/config/src/types.rs?ref=rust-v0.150.1` | `:107-118` — `enum AuthCredentialsStoreMode { #[default] File, Keyring, Auto, Ephemeral }`, each with a vendor doc comment. `AuthKeyringBackendKind::{Direct,Secrets}` at `:139-153`, `Secrets` on Windows and `Direct` elsewhere — a **separate axis**, so "is the encrypted backend selectable through `cli_auth_credentials_store`?" answers **no**. `Ephemeral` is modelled by neither `bk6owf` §3.3 nor this ADR |
| **33** | The keyring→file fallback is gated on `Auto` (N16, Q12 read-half answered) | same method, `codex-rs/login/src/auth/storage.rs?ref=rust-v0.150.1` | `:511-528` dispatches `File`→`FileAuthStorage`, `Keyring`→`create_keyring_auth_storage`, `Auto`→`AutoAuthStorage`, `Ephemeral`→`EphemeralAuthStorage`. Both fallback `warn!` lines are inside `impl AuthStorageBackend for AutoAuthStorage` (`:431-449`). `create_keyring_auth_storage` (`:531-545`) has **no file leg**. **N12a's doc-versus-binary conflict is withdrawn** |
| **34** | Codex reads six more auth-affecting environment variables, and two of them are load-bearing (N15) | same method, `codex-rs/login/src/auth/manager.rs` and `.../revoke.rs?ref=rust-v0.150.1` | `manager.rs:199-201, 910-912` declare `CODEX_REFRESH_TOKEN_URL_OVERRIDE`, `CODEX_REVOKE_TOKEN_URL_OVERRIDE`, `CODEX_APP_SERVER_LOGIN_CLIENT_ID`, `OPENAI_API_KEY`, `CODEX_API_KEY`, `CODEX_ACCESS_TOKEN`. `:1456-1462` carries the vendor's own comment *"API key via env var takes precedence over any other auth method"*; `:1492-1497` does the same for `CODEX_ACCESS_TOKEN`. `refresh_token_endpoint()` at `:1717-1720` is `env::var(…).unwrap_or_else(…)` — **no allowlist**, unlike claude's `ALLOWED_OAUTH_BASE_URLS` which throws (ledger 7). `revoke.rs:135-139` accepts its own override and falls back to the refresh one |
| **35** | All eight codex environment names are present in the **installed** binary, not only at the tag | `strings -a <codex 0.150.1 Mach-O> \| grep -c <name>`, per name | Present, all eight. Rust concatenates string literals, so whole-line matches are 0 and substring matches are the correct method — `CODEX_SQLITE_HOME` 3, `CODEX_REFRESH_TOKEN_URL_OVERRIDE` 2, `CODEX_REVOKE_TOKEN_URL_OVERRIDE` 1, `CODEX_APP_SERVER_LOGIN_CLIENT_ID` 1, `CODEX_ACCESS_TOKEN` 8, `CODEX_API_KEY` 5, `OPENAI_API_KEY` 23, `CODEX_HOME` 68. Context confirms semantics: `` `CODEX_SQLITE_HOME` is overridden by an exact requirement `` (`core/src/config/requirements.rs`) and `CODEX_REFRESH_TOKEN_URL_OVERRIDE` adjacent to `https://auth.openai.com/oauth/`. **The tag and the installed build agree** |
| **36** | Codex canonicalizes `CODEX_HOME` and errors if it cannot (partial answer to Q5's codex half) | same method, `codex-rs/utils/home-dir/src/lib.rs?ref=rust-v0.150.1` | `find_codex_home()` reads `CODEX_HOME`, requires the path to **exist** and be a **directory**, and `path.canonicalize()`s it (`:43-47`), erroring otherwise. A non-existent `CODEX_HOME` is **fatal**, not silently defaulted — relevant to B0's refusal design |
| **37** | The proposed auth root is **inside** the installer-managed config dir (N18, Q7 answered) | read `scripts/setup.sh`, `scripts/setup.ps1`, `tools/agents-infra/internal/infra/source_dir.go` in this repository | `setup.sh:69` → `$HOME/Library/Application Support/agents-infra`; `:72` → `${XDG_CONFIG_HOME:-$HOME/.config}/agents-infra`; `setup.ps1:13` → `%APPDATA%\agents-infra`; `source_dir.go:387-404` resolves the same three in Go. `write_install_state` (`setup.sh:208-211`) does `mkdir -p "$CONFIG_DIR"` and writes `install.json`. **Write set inside `CONFIG_DIR` is exactly one file; no recursive operation exists today** |
| **38** | qwen has a state root that namespaces its credential (N19, Q10 answered) | `gh api repos/QwenLM/qwen-code/contents/packages/core/src/config/storage.ts --jq .content \| base64 -d`, default branch `main` | `getGlobalQwenDir()` `:193-203` reads `process.env['QWEN_HOME']`, falling back to `~/.qwen`. `getOAuthCredsPath()` `:640-642` = `getGlobalQwenDir()/oauth_creds.json`. `getRuntimeBaseDir()` `:172-190` adds `QWEN_RUNTIME_DIR` with its own precedence chain. **Standing: `main`, no version pin, and qwen is not installed on this host — no shipped-build corroboration, unlike every other runtime claim in this ledger (Q18)** |
| **39** | A GitHub code-search zero is not an absence (N19 method note) | `gh api -X GET search/code -f q='repo:QwenLM/qwen-code QWEN_CODE_HOME OR QWEN_HOME'` | **`total_count: 0`**, for a constant that is present in `packages/core/src/config/storage.ts`. Found instead by reading the file that the `oauth_creds.json` query returned. Recorded because this ADR labels `curl` 403s as failed reads and would otherwise have spent this zero as a finding. Ledger 23's zero is re-labelled accordingly |
| **40** | Claude 2.1.248 ships a second, file-based, named-profile credential store (N20) | `strings -a ~/.local/share/claude/versions/2.1.248 \| grep -oE 'process\.env\.(CLAUDE\|ANTHROPIC)_[A-Z0-9_]+' \| sort -u`, then byte-window extraction around the `ANTHROPIC_CONFIG_DIR` read site | **174** distinct read sites. The profile machinery: root `ANTHROPIC_CONFIG_DIR` › `$XDG_CONFIG_HOME/anthropic` › `$HOME/.config/anthropic`; profile from `ANTHROPIC_PROFILE` or `<root>/active_config`, default `"default"`; config `<root>/configs/<n>.json`; credential `authentication.credentials_path` defaulting to `<root>/credentials/<n>.json`; accepted `authentication.type` = `user_oauth` \| `oidc_federation`; precedence explicit › env-quad (`ANTHROPIC_FEDERATION_RULE_ID` **and** `ANTHROPIC_ORGANIZATION_ID`) › implicit; status string `` credentials-file · <authType> · profile <name> ``. Internal label *"WIF profile read-ahead"*. **No verb that writes `configs/<n>.json` was found, and `~/.config/anthropic` is absent on this host — the enrolment path is unread (Q17)** |
| **41** | Ambient-variable survey, this host only (N20a, Q11 partly answered) | `env \| grep -q "^NAME="` per name; `grep -l` over `~/.zshrc`, `~/.zshenv`, `~/.zprofile`, `~/.bash_profile`, `~/.profile`; `[ -d ~/.config/anthropic ]` | All nineteen surveyed names **unset**; none mentioned in any shell profile; `~/.config/anthropic` **absent**. **Presence tested by name — no value was read, printed or persisted.** One host, one shell, and this process is itself a composed child environment: weak evidence about other operators' setups, and labelled as such |
| **42** | The 403 wall is unchanged (Q2b-b, W18) | `curl -sS -o /dev/null -w '%{http_code}' -A '<browser UA>'` on both URLs, re-run in round four | **403** on `help.openai.com/en/articles/20001068-use-multiple-accounts-with-account-switching` and `openai.com/policies/terms-of-use/`. Still a **failed read, not an absence**; N11 stays second-hand and stays demoted |
| **43** | Codex's store-selector default is stable across seven tags (N21, Q4 documentary half) | `gh api repos/openai/codex/tags --jq '.[].name'` to enumerate stable `rust-v0.1*` tags; `gh api repos/openai/codex/contents/codex-rs/config/src/types.rs?ref=<tag>` on `rust-v0.147.0`, `0.148.0`, `0.149.0`, `0.149.1`, `0.150.0`, `0.151.0`, `0.152.0-alpha.6` | `#[default]` on `File` in `AuthCredentialsStoreMode`, at the same line range, in all seven tags — the earliest stable release through an unreleased alpha |
| **44** | Claude applies only NFC normalization to the config-dir value (N22, Q5 documentary half) | Byte-window extraction around two independent `.normalize("NFC")` sites in the installed 2.1.248 bundle, the same method as ledger 40 | `var ge=Ko(()=>(s()??i(g(),".claude")).normalize("NFC"),s)` and `ok()`'s `r=e!==void 0?e.normalize("NFC"):ge()`. No `path.resolve`, no trailing-slash trim, no symlink dereference in either |

Inherited evidence is not restated here; see `1g880w`'s 13-row ledger and
`3moaky`'s ledger, both of which stand except where §13 corrects them.

Rows 15–26 were gathered in round two, 2026-08-31, after review found the
entitlement claim unresearched. **Rows 27–30 were gathered in round three**, and
rows 19, 21 and 25 were **re-run and corrected** in round three after review
found that N11's scope word had been dropped everywhere the quote was used and
that row 21 did not reproduce. **Rows 31–42 were gathered in round four**, when
review found that six reads this document had labelled free were still unrun.

**Round four's method note, recorded because it changes how these rows should be
re-run.** Rows 31–36 read `openai/codex` at the tag **`rust-v0.150.1`**
(`0eb410ad0dd161ea323b05452f978de01cd63430`), which matches the installed
`codex-cli 0.150.1` — not at the default branch. A first pass read `main` and was
discarded: `main` is a different program from the one installed, and reporting it
as evidence about the installed CLI is the same widening this document has now
diagnosed four times. Every environment-variable literal was additionally
corroborated in the installed Mach-O binary (row 35); the tag and the build agree.
**Row 38 is the exception and is labelled as one** — qwen is not installed here,
so it is `main` with no shipped-build cross-check and no version pin.

**No account was created, enrolled or authenticated on either provider** in any
round, no credential value was read, and **no `security` invocation was made in
any round**. Round four's evidence gathering was limited to `gh api` reads of two
public repositories, two unauthenticated `curl` status probes, read-only
`strings`/`grep` over already-installed binaries, read-only reads of this
repository, and an environment survey that tested variable **presence by name
only**. No vendor CLI subcommand that touches state was executed.

---

## 15. Traceability to epic acceptance criteria

| AC | Status |
| --- | --- |
| **AC1** Research proves what each provider allows; unsupported behaviour not inferred | Met **in round three**; not met in round one, and only partly met in round two. §5 labels every claim proven / current-source / **vendor-documented** / unknown, per provider. Round one inferred an absence of evidence from an unperformed read. Round two did the read and then inferred a *vendor position on the Codex CLI* from a sentence scoped to Codex **desktop** — the same violation in the restrictive direction. Round three restores the scope and records codex CLI entitlement as **unestablished**, which is what the evidence supports (§2.3, §12.3). Q1 remains held as unknown rather than inferred in the permissive direction. |
| **AC2** Model separates provider, identity, alias, parameterized method; email+OTP a candidate not an assumption | Met by `bk6owf` §3, adopted here as the design B1 would use. Email is an identity *claim* with verification state; OTP is a transient input with no field to be stored in. |
| **AC3** At least three custody models compared | Met. §6 compares A, B (split B0/B1) and C on security, concurrency, refresh, revocation, version dependency and what each gives up. |
| **AC4** Targeted logout invalidates only that profile; server-side revoke and metadata removal are separate | Met in design (`bk6owf` §5.4), carried in §6. Not implemented — B1 is held. `remote_revoke` for anthropic is `unknown` (Q3) and the CLI must say so. |
| **AC5** Verdict may be full-go, hybrid or no-go; must not force one design across incompatible providers | Met. Split verdict: A no-go permanently, B0 go, **B1-claude conditional go, B1-codex held/unestablished**, C go for managed automation, qwen provisional. AC5 permits a no-go; **it does not require one**, and round three withdrew a no-go rather than defending it once its reasons failed — which is the same discipline in the other direction. Round one satisfied this on the option axis and violated it on the provider axis by writing one entitlement verdict for "either vendor". Round three adds the converse discipline: §7.3's property-versus-derivation distinction is applied to **both** providers rather than used to rescue one, which is what surfaced that the codex cost argument holds only via `keyring` and only because of §8 (§7.3.1). An asymmetric verdict has to be re-derived symmetrically or it is just a preference. |
| **AC6** Chosen model supports multiple independently named accounts and prevents cross-account leakage | **Not met as a whole. Round four evaluates the codex half that round three left `unevaluated`, and it splits cleanly in two.** AC6 names two properties, and conflating them is what let "unevaluated" stand. *(i) **Leakage prevention — evaluated, and it passes for codex.*** `sqlite_home` defaults **inside** `CODEX_HOME` and `log_dir` → `$CODEX_HOME/log` (N13, ledger 31), so two credential-isolated codex profiles do **not** share a state database or logs by default. The residual is ambient, not architectural: `CODEX_SQLITE_HOME` set in the parent relocates the DB out of a composed root (N14) — and it is escapable per-profile, because `config.toml`'s `sqlite_home` outranks the variable. With that pinned, **codex satisfies AC6's leakage half.** *(ii) **Multiple independently named accounts — not met, and not because of leakage.*** It is not met because entitlement is unestablished (nothing OpenAI publishes addresses the CLI; N11 is scoped to Codex *desktop*), and the only remaining route to establishing it needs a second account (Q2). Third-party leak reports (N12b) concern the vendor's **own switcher**, not two `CODEX_HOME` roots — at least one (`#22419`) names the CLI, corrected in §12.2 — and are not evidence about this mechanism. **So AC6 for codex is no longer *unevaluated*: half is evaluated and passes, half is blocked on an account.** *Claude:* vendor-level support **is** established (N8, N9); AC6 becomes met once Q1 confirms server-side durability. Round four adds an unread alternative on this side — a second, profile-named credential store (N20, Q17) — which is recorded as a mechanism sighting, not as progress toward AC6. *qwen:* leakage is now modellable at all, because `QWEN_HOME` namespaces the credential (N19); entitlement is unaddressed. B0 prevents leakage for the one-account case on all three, **once W20/W21/W22 land** — which rounds one to three did not know were needed. |
| **AC7** Raw credentials and OTP values never enter repo files, board resources, config, argv, logs, shell history | Met, in both rounds. §1 and §14's closing note record the boundary; no `security` call was made in either round, and **no account was created, enrolled or authenticated** — the entitlement question was attacked with documentary evidence only. S1 and §8.4 are the design-level enforcement for B1. |
| **AC8** Claude and Codex conclusions evidence-backed; qwen explicitly provisional | **Met — and round four had to run the audit to keep it met.** Rounds one to three recorded qwen as *"not modellable"* with *"no mechanism"*, resting that on our own plugin's `HomeEnvVar: ""`, and listed the audit (Q10/W11) as **free** in every round without running it. Run in round four: qwen has `QWEN_HOME`, and it namespaces the credential (N19, ledger 38). §5.3 is rewritten. qwen remains **provisional** — nothing establishes its entitlement, refresh, revocation or concurrency, and it is not installed here so there is no shipped-build corroboration (Q18) — but the reason is now honest. The parallel *"the same is true of muse, gemini, agy and pi"* is **withdrawn**: those are unaudited, and the identical claim about qwen did not survive a single read. |
| **AC9** No implementation before feasibility findings and ADR receive review | Met. Nothing implemented, in any round. §11's B1 breakdown is explicitly unstarted, and round four's additions to §11.1 (W20–W22) and §10.1 (G-B0-7 to G-B0-9) are **recorded, not begun** — no repository code, test or config file is touched by this round. B0 is recommended for scheduling *after* this ADR is reviewed. |
