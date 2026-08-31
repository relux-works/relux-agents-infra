# TASK-260720-3gcfd1 — CR rev1 change-request evidence (round two)

Commit `cedd18b`, 3 files, +601 / -122, documentation only. No Go source, test
or config file touched.

## The correction, in one line

The four findings compound into one: **round one reported an unperformed read as
an absence, and that absence was the load-bearing reason for the headline
verdict.** Doing the read reversed the verdict for one provider.

## F1 — BLOCKING — "no evidence at all" is false; it is unresearched → FIXED

The read was performed on 2026-08-31 against published first-party material
only. **No account was created, enrolled or authenticated on either provider.**
It is recorded as a new §2.3 (N8–N12) with a read-status table (§2.3.1) that
states the reachability of every origin before any finding.

What it found — and the reviewer's characterisation of *which* vendor is silent
turns out to be inverted on the axis that matters:

**Anthropic is not silent. Its Claude Code product docs state the mechanism and
recommend the use case.** `code.claude.com/docs/en/authentication`, verbatim:

> …and keys the macOS Keychain entry to that directory too, so a session with a
> different `CLAUDE_CONFIG_DIR` reads a different entry.

`code.claude.com/docs/en/env-vars`, the `CLAUDE_CONFIG_DIR` row, verbatim:

> Useful for running multiple accounts side by side: for example,
> `alias claude-work='CLAUDE_CONFIG_DIR=~/.claude-work claude'`.

Consumer Terms restrict **sharing** an account with other people and place no
limit on how many accounts one person holds. The review checked the Consumer
Terms and the support articles — correctly finding them silent — but not the
Claude Code product documentation, which is where the answer was.

**OpenAI, confirmed and extended from reachable sources.** The account-switching
article reproduced on a second, independently-phrased query. Beyond it:
`learn.chatgpt.com/docs/auth` (HTTP 200, the canonical target of
`developers.openai.com/codex/auth`, which is where `openai/codex`'s own
`docs/authentication.md` points) documents `CODEX_HOME` **only** as the location
of `auth.json` and mentions multi-account nowhere; `gh api` on `openai/codex`
returns 20 account-switching issues, **19 open, none closed as already-
supported**, including reports of projects, conversations and usage quota
surviving a switch across accounts.

Restated per provider in §0, §2.1, §4, §5.1, §5.2, §7.3, §9, §11.3, §12, §13,
§14, §15, `README.md` and `LOGBOOK.md`.

## F2 — BLOCKING — one verdict over two asymmetric vendors → FIXED, and the verdict moved

The B1 row is split into **B1-claude** and **B1-codex** throughout, and it is not
cosmetic — the two halves now have different verdicts:

- **B1-claude: CONDITIONAL GO.** Documented mechanism (N8), vendor-recommended
  use case (N9), no count limit in the terms (N10). One gate remains: **Q1**,
  whether a second concurrent enrolment leaves the first working server-side over
  24 h. Empirical, and runnable by an operator who already holds two accounts.
- **B1-codex: NO-GO, on affirmative vendor evidence** rather than on absence —
  which is the stronger no-go the review asked for. It does not depend on Q2
  ever being run.

**§7's cost argument is scoped, not repriced (new §7.3).** This is the part the
review did not anticipate and it is why the claude verdict could move. §7 prices
the `sha256(NFC(dir))[:8]` **derivation**. B1 never needed the derivation — it
needs the **property** "different dir ⇒ different Keychain entry", which N8 shows
is documented. B1 selects a profile by choosing the directory; the vendor
computes the service name. So the every-two-days allowlist operation is dropped
for claude (H3 not required) and stands **in full** for codex. §7.1/§7.2 are
unedited apart from one qualifier; §7.3 scopes them.

Honest limits stated in §7.3 rather than glossed: a documented behaviour can
still change, but a documented change is announced and universal; only the
Keychain **keying** is documented, so N1's two extra namespace inputs stay behind
a **fail-closed presence refusal** (H5, unchanged and now more important).

The review's §6 note — "no change to the B1 NO-GO outcome" — is respectfully not
followed, because the spawn brief directed re-derivation over defence and the
evidence found on the Anthropic side is materially stronger than what was
available when that note was written. §13 records the disagreement explicitly.

## F3 — SHOULD FIX — Q2b's cost column licensed skipping the read → FIXED

Q2b is split into **Q2b-a** (anthropic terms — ANSWERED, free read),
**Q2b-b** (openai terms — partly answered, 403-blocked) and **Q2c** (two billed
consumer subscriptions — residual, unaddressed anywhere, and a purchasing
decision). §4's closing paragraph names the misclassification as the *mechanism*
of the failure, not merely as an error: eight sibling rows were filed as free
source audits and this one was not. The free half is recorded in §11.2 as
**W17, complete**, alongside new W18 (re-read the 403-blocked pages from an
unblocked network) and W19 (resolve the doc-vs-binary keyring conflict).

## F4 — MINOR — evidence-ledger row 9 does not reproduce → FIXED

Corrected to **10 in every build**, with the method that re-runs:
`LC_ALL=C grep -ao 'CLAUDE_CODE_CUSTOM_OAUTH_URL' <build> | wc -l`. The reason
7,7,7,8,8 does not reproduce is now stated: the builds are Mach-O arm64
executables, so `grep -c` without `-a` returns **exit 1 and no count at all**.
Twelve new ledger rows (15–26) cover round two.

## Read failures recorded as read failures, not absences

`help.openai.com`, `openai.com/policies` and `chatgpt.com` return **HTTP 403 to
every request from this host** — `curl` with a browser UA and the fetch tool
alike; the body is OpenAI's own block page (ledger row 24). Consequences carried
explicitly:

- Everything sourced from those origins is labelled **second-hand** at every
  point of use. It reached this task through the search index on two
  independently-phrased queries.
- The OpenAI ToU free-tier multi-account clause the review quoted **could not be
  corroborated from here at all**. It is carried as reviewer-supplied and
  unverified, and **nothing in the ADR depends on it**.
- The codex verdict deliberately does **not** rest on the second-hand quote: N12
  corroborates it from `learn.chatgpt.com` and `gh api`, both reachable.

## Adverse finding the read produced, carried rather than buried

`learn.chatgpt.com/docs/auth` documents a plaintext fallback for `auto` **only**;
`keyring` is documented as storing in the OS credential store, full stop. N4
found keyring→file fallback strings on **both** legs. The two sources conflict,
so `keyring` cannot be relied on to mean "never plaintext". This **strengthens**
§8's hazard and weakens `3moaky`'s "force keyring" implication further. §5.2's
failure-posture row and §13 both record it.

## What was left alone, per §6.5 of the verdict

§§6 (except the B1 split the review itself demanded), 8, 10, 11.1, 13's existing
rows — unedited. §8's three-way plaintext split is carried verbatim. §10 adds one
paragraph explaining why a *conditional* go still gets no CLI UX: the condition
is an unrun experiment, and specifying UX would convert a likely outcome into a
committed one.

## Logbook

The 0842 entry's headline claim is **struck in place** with a correction notice
rather than quietly rewritten — the false claim stays visible with a pointer to
the correction, because silently editing a durable lesson is how the next reader
loses the ability to see that it was ever wrong. The binary-inspection findings
in that entry all reproduce and stand. A new 1140 entry carries the corrected
reusable lesson: *an unperformed read is not an absence, and the tell is the cost
column.*

## Boundary held

No account created, enrolled or authenticated on either provider. No credential,
token, cookie or Keychain value read, printed, exported or persisted. **No
`security` invocation of any kind.** No login, logout, revoke, rotation or
re-authentication. Vendor CLI execution: none in this round. §11's implementation
breakdown remains **unstarted for both providers**.

## Validation

Documentation-only delta, so no test was added or modified; the existing suite
was re-run to confirm nothing regressed.

| Command | Exit | Result |
| --- | ---: | --- |
| `go build ./...` | 0 | ok |
| `go vet ./...` | 0 | ok |
| `go test -count=1 ./...` | 0 | all 4 packages with tests `ok`; `cmd/model-harness` has no test files |

The combined `-count=1` run was killed by the shell's 2-minute call limit (exit
143) after two packages had reported; the remaining two were rerun in a separate
longer call and both reported `ok`. Both halves are in
`TASK-260720-3gcfd1_validation-rev2.log`, including that note.

## Checklist item 16 — "attacked, not read" — and why it is now checked

I first left this unchecked, reasoning that it is about shipped gate behavior and
that a documentation-only delta ships no gate. On reflection that reading is too
narrow for where the item actually sits: it is on a decision task, next to items
about evidence, assumptions and unknowns. Its subject here is the ADR's own
claims and the gates it specifies — and read that way it is the item this entire
round is about.

**A passing suite means nothing unless something in it would have failed.
Something did.**

- The ADR's **load-bearing claim was attacked and broke.** "No evidence at all"
  was the reason carrying the headline verdict, and one free documentary read
  falsified it. Round one was the positive-path-only version of this document;
  this round is the attack on it, and the verdict moved as a result.
- **A ledger row was re-run and did not reproduce.** §14 row 9's 7,7,7,8,8 is
  wrong; the real count is 10 in every build, and the reason the original method
  fails silently — `grep -c` without `-a` on a Mach-O binary returns exit 1 and
  no count — is now recorded so the row re-runs.
- **A failed read was refused as evidence.** `help.openai.com` 403s. Rather than
  let that origin's content carry the codex verdict, the verdict was
  re-established from two sources that actually returned 200
  (`learn.chatgpt.com`, `gh api` on `openai/codex`). An absence and a failure to
  read are different facts, and the ADR now states which is which per origin
  before stating any finding.
- **Evidence that cuts against the conclusion was surfaced, not buried.** The
  codex docs promise a plaintext fallback for `auto` only while the binary
  carries it on the keyring path too. That conflict makes `keyring` *less*
  trustworthy and strengthens the hazard section; it would have been easier to
  leave N4 as the earlier "narrowing".
- **Evidence that cuts against the new conclusion is also stated.** §7.3 lists
  the limits of the documentation argument — a documented behaviour can still
  change, only the Keychain keying is documented, the two extra namespace inputs
  are not — rather than letting the reversal run unqualified.
- **The specified gates carry their mutants.** §10.1's G-B0-1..6 each name the
  narrowing that must make them fail (G-B0-1 must fail even when the variable is
  set to the *correct* default path; G-B0-3 must fail when the parent's value
  leaks through and happens to match). §11.3 keeps H5's refusal **fail-closed**
  and states why over-refusal is the safe direction.
- **A likely outcome was refused as a finding.** Anthropic recommending the
  workflow makes Q1 likely to pass. B1-claude is CONDITIONAL GO, not GO, and §10
  withholds CLI UX on exactly that ground. Promoting the inference would have
  been the same error as round one's, run in the opposite direction.

What is *not* claimed: no Go source, test or config file was touched, so no gate
was implemented and none was executed. If the reviewer holds that this item can
only be satisfied by shipped code, it should be reopened — the reasoning is
stated here rather than buried so that call is available.
