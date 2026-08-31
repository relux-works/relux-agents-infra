# TASK-260720-3gcfd1 — response to `CR-TASK-260720-3gcfd1-3` (round four)

Commit `5c84ee7`, base `5feebbb`. 3 files, +993/−200, **`.md` only** — no
repository code, test or config file is touched, in this round or in the whole CR.

---

## 1. The sweep, which the brief said *is* the fix

> *"Before you finalise: enumerate every read this document calls free, and
> confirm each was actually run. Report the count and the result of each."*

**§4's cost column names twelve items as free. Four are not reads** — Q3 is free
only once a second enrolled account exists, and Q4/Q5/Q6 are free *synthetic
experiments* needing a harness. **That leaves eight free reads.**

| # | Free read | Entering round four | Now | Result |
| --- | --- | --- | --- | --- |
| Q2b-a | Anthropic terms / product docs | **run** (round two) | not re-run | Answered; N9/N10 stand |
| Q2b-b | OpenAI terms / help centre | attempted, **403** | **re-attempted** | **Still 403** on both URLs. Failed read, not an absence |
| Q7 | Installer write set vs. proposed auth root | **UNRUN** | **RUN** | **Answers against the design (N18)** |
| Q10 | Does qwen have a state root? | **UNRUN** | **RUN** | **Refutes §5.3 (N19)** |
| Q11 | Do legitimate setups set claude's extra inputs? | **UNRUN** | **RUN, host-scoped** | Partly answered (N20a); stays open at reduced scope |
| Q12 | Is the keyring→file fallback gated on `auto`? | **UNRUN** | **RUN** | **Yes (N16).** Retires N12a's "conflict" |
| Q14 | Store-selector accepted values | **UNRUN** | **RUN** | **Four values, not three (N17)** |
| Q15 | Does `sqlite_home` default inside `CODEX_HOME`? | **UNRUN** | **RUN** | **Yes — closed favourably (N13)** |

**Count: eight named · one run · one blocked · six unrun for three rounds · all
six now run · the blocked one re-attempted and still blocked.**

Cost of the six: 14 `gh api` reads, 2 `curl` probes, 4 local file reads, one
`strings` pass. **Four of the six moved something. One moved a verdict.**

Recorded in the ADR as **§2.4**, and §11.2's free-work table now carries a
**status column** — a table of free work without one reports intention as
evidence, which is how the same item stayed unrun three rounds running.

**Method correction made mid-round and recorded (§2.4.2, ledger closing note).**
The first pass read `openai/codex` at `main`. That is a different program from
the installed one. Everything was re-read at tag **`rust-v0.150.1`**
(`0eb410ad…`, matching `codex-cli 0.150.1`) and every environment-variable
literal was corroborated in the installed Mach-O binary. Tag and build agree.
The one claim that cannot be pinned this way — qwen, not installed here — is
labelled unpinned rather than levelled up.

---

## 2. (c) The blocker — `CODEX_SQLITE_HOME`, and what the same method then found

**Confirmed at the installed tag, all three reads.** `config_toml.rs:329-331`
doc comment; `core/src/config/mod.rs:3918-3923` production resolution;
`state/src/lib.rs:106-107` `SQLITE_HOME_ENV = "CODEX_SQLITE_HOME"`. Precedence
`config.toml` › `$CODEX_SQLITE_HOME` › `$CODEX_HOME`. Ledger **31**.

**Added to B0's prerequisites, and the count corrected.** §5.2's *"Namespace
inputs: **One** (N5)"* and §11.2's **"W13 is the only one B0 depends on"** are
both refuted in place (N14; §13 rows; §9 item 2). N5 is precise about the
**credential namespace** and was spent at the scope of **state isolation and B0's
prerequisites** — the same precise-where-derived / wider-where-used shape §12.4
tabulates, on the axis nobody enumerated.

**"Check whether any other ambient input was missed by the same method."** Done,
by switching method: enumerate what each program reads from its environment
instead of grepping the literal expected.

- **Codex: eight, not one** (N15, ledger 34/35). `CODEX_HOME`,
  `CODEX_SQLITE_HOME`, `OPENAI_API_KEY`, `CODEX_API_KEY`, `CODEX_ACCESS_TOKEN`,
  `CODEX_REFRESH_TOKEN_URL_OVERRIDE`, `CODEX_REVOKE_TOKEN_URL_OVERRIDE`,
  `CODEX_APP_SERVER_LOGIN_CLIENT_ID`. Two are worse than a count implies:
  `manager.rs:1456-1462` carries the vendor's own comment *"API key via env var
  takes precedence over any other auth method"* — an ambient key makes a composed
  launch **run as the wrong identity while reporting the right one** — and
  `refresh_token_endpoint()` (`:1717-1720`) accepts **any** value with no
  allowlist, where claude's `CLAUDE_CODE_CUSTOM_OAUTH_URL` is checked against
  three endpoints and **throws**. On this axis the ADR's claude-is-the-larger-
  surface framing inverts; *"cleaner than claude"* is withdrawn.
- **Claude: 174 `process.env` read sites**, and among them an entire **second,
  file-based, named-profile credential store** — `ANTHROPIC_CONFIG_DIR` /
  `ANTHROPIC_PROFILE` → `configs/<n>.json` + `credentials/<n>.json`, accepting
  `authentication.type = "user_oauth"` (N20, ledger 40). Unknown to three rounds.
  **Recorded as a mechanism sighted in shipped code with its enrolment path
  unread (Q17), not as an option** — no writing verb was found, the subsystem is
  internally named for workload-identity federation, and `~/.config/anthropic`
  does not exist here.
- **qwen: `QWEN_HOME` + `QWEN_RUNTIME_DIR`** (N19).

**Consequences carried into the work breakdown, unstarted:** §11.1 gains
**W20/W21/W22**; §11.2 gains **W24**; §10.1 gains **G-B0-7/8/9**, each written as
a *per-variable* negative — the defect was a set sized from the wrong
enumeration, and a gate that admits one member of a set it claims to cover is
that defect in test form.

---

## 3. (a) The risk that is closed, favourably

`sqlite_home` defaults **inside** `CODEX_HOME`. Corrected at every site the
verdict named, plus three the verdict did not:

§0 (preamble, table, codex bullet) · §2.2 N7 · §2.3 N12b standing · §4 Q15 and
the recount · §5.2 *Other state*, *Failure posture*, B0 and B1 verdicts · §7.3
closing bullet · §7.3.3 · §9 items 2/4 and both "why not build" paragraphs ·
§10 preamble · §11.2 (W10 → DONE, promotion dropped, re-ranked) · §12.2 item 1 ·
§12.3 · §12.4 · §13 · §15 AC6 · `README.md` · `LOGBOOK.md` 1210.

**Q12 closed too, and it was not on the verdict's list.** The keyring→file
fallback is gated on `Auto`; `Keyring` dispatches to a backend with **no file
leg** (N16, ledger 33). **N12a's doc-versus-binary conflict is withdrawn — there
never was one.** This ADR reached the cautious conclusion from string literals,
labelled it correctly as *"a narrowing, not an answer"*, and then carried the
narrowing as adverse for two rounds while the control flow sat unread.

**The verdict does not move, and §12.4 records why that is the round's second
lesson.** Round three staked B1-codex's reopening list on Q15 and Q12 —
*"most of what would settle it is ours and free"*. Both were run, both came back
favourable, **neither settled anything**, because both were *mechanism* questions
and what is open is *entitlement*. **Cheapness is a reason to run a read; it is
never evidence the read is load-bearing.** The only thing left that can decide
B1-codex is a second account (Q2).

---

## 4. (b) AC6 for codex — evaluated

§15 no longer records `unevaluated`. AC6 names **two** properties and conflating
them is what let the label stand:

- **Leakage prevention — evaluated, and codex passes.** `sqlite_home` and
  `log_dir` both default inside `CODEX_HOME` (N13), so credential-isolated
  profiles share neither state DB nor logs. The residual is *ambient*
  (`CODEX_SQLITE_HOME`), and escapable per-profile because `config.toml`
  outranks the variable.
- **Multiple independently named accounts — not met**, on entitlement, which
  needs an account (Q2).

Also updated: **AC8** (qwen), **AC9** (round-four additions recorded, not begun).

---

## 5. F2 and F3

**F2 — §12.2 item 6.** Corrected. `#22419`'s body reads *"After switching
accounts in Codex App **/ Codex CLI**"* — it names the CLI. The surface list was
narrower than its own sources, in the direction that made adverse evidence look
less relevant. The load-bearing half is kept and stated without it: every
reporter used the **vendor's own switcher**, not two `CODEX_HOME` roots. Also
corrected at §2.3's N12b standing paragraph, which had leaned on the same
reports as *"direct corroboration of Q15's worry"* — Q15 is answered.

**F3 — `LOGBOOK.md` 1210 row-21 cause.** Struck. The query returns **27**, which
fits inside a thirty-item page, and returns 27/21/6 **both** with `-f
per_page=100` and with the default page size; the narrower phrasing gives
16/13/3. Neither candidate reproduces. The entry now records the cause as
**unknown** and keeps only the method that does re-run — with the point made
explicitly, since the false cause was asserted one bullet after *"a correction
carries the same standard as a claim"*.

---

## 6. What else the sweep produced, reported because a sweep is not supposed to only return good news

- **Q7 answers against the design (N18, ledger 37).** The proposed auth root is a
  **child** of the installer-managed `CONFIG_DIR` on macOS (`setup.sh:69`),
  Windows (`setup.ps1:13`) and Linux (`source_dir.go:387-404`). File-level
  disjoint today — one file, `install.json` — but directory-level nested, so a
  future recursive uninstall takes credentials with it. Q7's proposed test
  ("assert disjointness") **cannot pass as written**; replaced by **H14**.
- **A verdict moves: qwen (N19, ledger 38).** `QWEN_HOME` exists and namespaces
  the credential (`getOAuthCredsPath()` resolves through it). `HomeEnvVar: ""` is
  **our** plugin defect reported for three rounds as a vendor limitation. §5.3 is
  rewritten; §0 gains a sixth row; W22 is scheduled. **Standing is capped
  honestly:** `main`, no version pin, qwen not installed here, so no shipped-build
  corroboration — unlike every other runtime claim in the ledger (Q18). It is
  *provisional-with-a-mechanism*, not promoted toward B1. The parallel claim about
  muse/gemini/agy/pi is **withdrawn as unaudited**.
- **A method anomaly worth more than the finding (ledger 39).**
  `gh api search/code … QWEN_HOME` returned **`total_count: 0`** for a constant
  plainly present in the target file. **A code-search zero is a failed or partial
  read, not an absence** — this ADR's own standard for `curl` 403s, applied to a
  tool whose zero looks like a result. Ledger 23's zero is re-labelled.

**This contradicts the verdict's "Not requested: no change to any verdict …
qwen not-modellable."** Stated plainly rather than buried: that list was written
before the free read existed, the brief directed the read, and the read refutes
the claim. Everything the verdict asked to leave alone — §§1.1–1.5's subjects,
§7.3.1–7.3.3, §2.3's N11/N12/N12b/N12c, §8, §2.1, §7.0–7.2, §12.1's Q1 gate, §13
and both logbook entries' structure — is left alone except where round four's
evidence directly falsifies a sentence, and each such edit is marked as a
correction rather than a rewrite.

---

## 7. Verdicts after round four

**A** permanent NO-GO · **B0 GO** (prerequisites 1 → 5) · **C GO** ·
**B1-claude CONDITIONAL GO** · **B1-codex HELD — UNESTABLISHED** ·
**qwen provisional, no longer "not modellable"**. Both halves of B1 unstarted;
§11 unstarted; the delta is `.md` only.

---

## 8. Validation

| Command | Exit |
| --- | ---: |
| `cd tools/agents-infra && go vet ./...` | **0** |
| `cd tools/agents-infra && go test ./... -count=1` | **0** (5 packages, all `ok`) |

Full output in `TASK-260720-3gcfd1_change-request_rev4-validation.log`. No test
was added or changed, because no behaviour was: the delta is three markdown
files, and §11's breakdown remains unstarted by design.

**The commit is UNSIGNED.** `git commit -S` fails with gpg *"No secret key"* for
`alexis <alexis@relux.works>`; no signing key is configured on this host, and the
three prior commits on this story branch are unsigned as well. Reported, not
worked around.

---

## 9. Safety boundary

**No account created, enrolled or authenticated on either provider. The Q1/Q2
experiments were not run.** No credential, token, cookie or Keychain value read,
printed, exported or persisted. **No `security` invocation of any kind.** No
login, logout, revoke, rotation or re-authentication. No vendor CLI subcommand
that touches state — `codex --version` and `claude --version` only.

Evidence gathering was: `gh api` reads of two public repositories, two `curl`
status probes, read-only `strings`/`grep` over already-installed binaries,
read-only reads of this repository, and an environment survey that tested
variable **presence by name only** — no value was read or printed. §11's
implementation breakdown remains **unstarted**.
