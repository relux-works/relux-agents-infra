# TASK-260720-3gcfd1 — review verdict: CHANGES REQUESTED

Change Request `CR-TASK-260720-3gcfd1-1` rev 1 · reviewed 2026-08-31
Delta: `5feebbb..e3ef896`, 5 files, +2371, documentation only.

**Verdict: changes requested → `analysis`.** The verdict *outcome* (B1 NO-GO)
survives attack. The **stated reason carrying it** does not, and it has been
propagated into `README.md` and `LOGBOOK.md` as a durable claim about the state
of the evidence that a free documentary check falsifies.

---

## 1. What survived attack

I independently re-derived the ADR's own findings rather than reading them.
Everything below reproduced on this machine, read-only, with no `security`
invocation and no credential value read.

| Claim | Independent check | Result |
| --- | --- | --- |
| N1 — service name is `Claude Code${OAUTH_FILE_SUFFIX}${n}${c}` | byte grep, 2.1.248 | **Reproduced verbatim** |
| N1 — three env inputs | grep across all 5 builds | **Reproduced** |
| N1 — `ALLOWED_OAUTH_BASE_URLS` = 3 endpoints, throws otherwise | grep 2.1.247/2.1.248 | **Reproduced exactly**: `beacon.claude-ai.staging.ant.dev`, `claude.fedstart.com`, `claude-staging.fedstart.com`; `if(!l.includes(a))throw Error(...)` |
| N2 — `t = e!==void 0 ? !e : !process.env.CLAUDE_CONFIG_DIR` | grep all 5 builds | **Reproduced in all five**, incl. `sha256...substring(0,8)` |
| N3 — `OAUTH_FILE_SUFFIX` non-empty values shipped | grep 2.1.248 | **Reproduced**: `""`, `-custom-oauth`, `-local-oauth`, `-staging-oauth` |
| N4 — codex keyring→file fallback, both legs | `strings` on 0.150.1 native binary | **Reproduced** both literals + `failed to remove CLI auth fallback file:` |
| N5 — `Codex Auth` single literal | occurrence count | **Reproduced**: exactly 1, `cli\|` exactly 1 |
| N6 — 2.1.235 same construction | grep | **Reproduced** |
| N7 — `sqlite_home`, `log_dir`, `forced_login_method` | config key table | **Reproduced**; `cli_auth_credentials_store = "file"` / `mcp_oauth_credentials_store = "auto"` re-confirmed |

**Repository-defect claims — all four reproduced at the cited sites:**

- `runClaude`/`runCodex` (`tools/agents-infra/main.go`) build `exec.Command` and
  **never assign `cmd.Env`**. Confirmed; the only `cmd.Env` assignments in the
  module are the two pi paths.
- claude `runtimeEnvKeys = []string{sessionMarkerEnv}` — **exactly one key**
  (`pkg/agentic/systems/claude/env.go:72`).
- `providerlimits/identity.go:116` — `os.Getenv(capabilities.HomeEnvVar)`, the
  parent's environment. The file's own header comment names the drift.
- `home_env` gets `strings.TrimSpace` at `spawn_runtimes.go:99` and nothing else,
  while `agentic_system`/`broker` are parsed against closed vocabularies.
  "Validated rather than trimmed" is a real gap.
- `spawn.go:940` — `cmd.Env = plan.Env`. W6's no-op entry is correct.
- qwen `HomeEnvVar: ""` at `qwen.go:121`, exactly as cited.

**Brief items 3, 4, 5 and the "Also" list — all pass:**

- **B0's independence from the undocumented derivation (item 3): holds under
  attack.** I tried to make it fail. B0 never computes a hash; it needs only
  "unset/empty means the default namespace". Even if `ok()` changed so that
  writing the default path became a no-op, *removing* the variable would still be
  correct. B0 smuggles in nothing B1 was rejected for: no version gate, no pin,
  no second account, no credential. The §6 claim stands as written.
- **§8 (item 4): carried as the three-way split**, not the earlier two-way
  version. Verified against `bk6owf` §6.1 line by line, including "Three halves
  is the wrong word and the right count" and the "two thirds runs at build or
  launch time" budgeting. Not softened.
- **§13 / 3moaky item 4 (item 5): handled correctly.** §2.1 refutes item 4
  specifically and cites the rest of the audit as whole and current, which is
  what the DoD asks for. The "survives as a statement about *support*, not
  mechanism" distinction is right and is the honest reading.
- **§7 prices the dependency, not defers it.** The operational cost is stated
  concretely — a manual pin-and-extend roughly every two days at the observed
  cadence, on the critical path of the primary tool — and the allowlist choice is
  made knowingly worse than the interval. That is owning the consequence.
- **§11 is unstarted.** The delta is `.md`-only; no code file is touched.
- **§6 states what each option gives up for accepted options too**, not only
  rejected ones — B0 gives up the epic's actual capability plus ambient
  configuration; C gives up the subscription.
- **N4's label is right, and arguably too conservative in the ADR's own favour.**
  The binary carries *both* `failed to load CLI auth from keyring, falling back
  to file storage:` **and** a bare `failed to load CLI auth from keyring:`. Two
  variants is mild evidence the fallback *is* gated. "A narrowing, not an answer"
  remains the correct label; noted so the producer does not over-read N4 later.

The four-way split (item 1) is **honest, not evasive**. Each boundary carries
weight: A and B1 differ on who holds the credential — a difference the fail-open
analysis makes decisive, not cosmetic — and cannot collapse. B0/B1 differ on
account count, and B0's verdict is carried by a dependency argument that is
categorically different in kind, not merely in degree.

---

## 2. Finding F1 — BLOCKING. The load-bearing NO-GO reason is a non-read reported as an absence

§0: *"the question that decides the **product** has produced **no evidence at
all**"* · *"The entitlement half is untouched."*
`README.md`: *"because **no evidence** establishes that either vendor permits
one human to hold two concurrently enrolled subscription logins."*
`LOGBOOK.md` 0842, as THE REUSABLE ONE: *"it has **no** evidence, not weak
evidence."*

The brief directed me to check exactly this: *"Verify it is genuinely
unestablished rather than merely unresearched — is there published vendor terms
material that answers it, which the ADR did not consult?"*

**It is merely unresearched.** Free, published, first-party vendor material
exists and speaks to Q1/Q2/Q2b in **both** directions:

**OpenAI — permissive direction.** OpenAI ships a first-party *account
switching* feature: stay signed in to **two ChatGPT accounts at the same time**,
switch instantly without logging out, with *"conversations, billing, workspaces,
and settings"* kept separate, across **all plan types**, max two per session.
A vendor that builds and documents concurrent dual-account sign-in has said
something substantial about whether one human may hold two logins.
`help.openai.com/en/articles/20001068-use-multiple-accounts-with-account-switching`

**OpenAI — restrictive direction, and this one favours the ADR's verdict.** The
same page states account switching is *"currently available on ChatGPT web; it is
**not yet supported in Codex** desktop or the native ChatGPT mobile apps."*
That is **published vendor evidence closing the codex half of B1** — far stronger
than "unestablished", and the ADR left its best available argument on the table.

**OpenAI ToU §4(e)** restricts multiple accounts *only* for free-tier credit
farming (*"You may not create more than one account to benefit from credits
provided in the free tier"*). A prohibition scoped that narrowly is informative
about the unscoped paid case.

**Anthropic — genuinely silent, and I verified this rather than assuming it.**
I fetched the Consumer Terms directly. They prohibit *sharing* credentials and
making the account available to others — a different question — and are silent on
one person holding two accounts. Support material permits a Claude account plus a
Console account under one email, and a personal Pro/Max account coexisting with a
Team account via a toggle; neither addresses two Pro/Max subscriptions.

**Read status, stated honestly.** `help.openai.com` returns HTTP 403 to direct
fetch — a *failed read, not an absence*. The article's content reached me through
the search index on two independent queries and is corroborated by a secondary
report (piunikaweb, 2026-03-05). The producer must confirm against the primary
source; I am reporting this as a strong documentary lead, not as settled fact.

**Why this blocks.** The ADR is transparent that Q2b is unread, so this is not
forgery. But "no evidence at all" is a claim *about the state of the evidence*,
and it is false — and it is the claim the entire headline verdict rests on, now
published in the operator-facing README and written into the durable LOGBOOK as
the reusable lesson. The ADR applies exactly the right standard everywhere else
(N4: "string literals, not control flow"; the 0748 entry correcting "a narrow
grep reported as an enumeration") and misses it at the one point that decides the
product.

---

## 3. Finding F2 — BLOCKING. Both vendors lumped into one entitlement verdict where the evidence is asymmetric

§0, §4 (Q1/Q2/Q2b), §5.1, §5.2 and §12 all treat entitlement as one undivided
unknown covering "either vendor". Post-check the two providers are **sharply
asymmetric**: OpenAI has substantial published material in both directions;
Anthropic is genuinely silent.

This is the ADR's own diagnosed error mode, committed on a different axis. N5
says it directly: *"Any design that says 'namespace the state root' without
naming which provider it is talking about is under-specified."* The same applies
to *"nobody has established that either vendor permits…"*. The document earns its
per-provider rigour in §5 and then discards it on the axis that carries the
verdict.

---

## 4. Finding F3 — SHOULD FIX. Q2b's cost column is what licensed skipping the read

§4 Q2b: *"What settles it: A human reading the subscription terms. Not an
experiment. Cost: A decision, and it is not the agent's to make."*

Reading published terms and help-centre pages is a **free documentary read**, not
a decision. §4 classifies eight other unknowns as free source audits; this one
belongs in that class. The *judgement* about whether to rely on what the terms
say is the human's; the *reading* was not. That misclassification is the
mechanism by which the single load-bearing input went unresearched while
substantial effort went into binary inspection already largely settled by the
inputs.

Split Q2b into the free read (agent-performable, now partly done above) and the
residual human judgement, and move the read into §11.2's free-investigations
table.

---

## 5. Finding F4 — MINOR. Evidence-ledger row 9 does not reproduce

§14 row 9 records `CLAUDE_CODE_CUSTOM_OAUTH_URL` occurrence counts as
**7, 7, 7, 8, 8** across the five builds. I count **10 in every build**
(`grep -ao … | wc -l`). The material claim — present in all five — is unaffected,
but a ledger exists so a later reader can re-run it, and this row does not re-run.
Either restate the counting method or correct the numbers.

---

## 6. What the producer should do

1. Perform the free documentary read for both vendors. Confirm the OpenAI
   account-switching article against the primary source (note the 403 on direct
   fetch and say how it was obtained). Restate §0, §4 Q1/Q2/Q2b, §5.1, §5.2, §12,
   `README.md` and the `LOGBOOK.md` 0842 entry to replace "no evidence at all"
   with what the evidence actually is, **per provider**.
2. Fold the *"not yet supported in Codex"* statement into §5.2's B1 verdict. It
   converts the codex half from "unestablished" to published-vendor-evidence
   backed — a **stronger** NO-GO than the one currently written.
3. Reclassify Q2b per F3 and route the free half into §11.2.
4. Fix or re-methodise §14 row 9.
5. Leave §§6–8, 10, 11, 13 alone. They survived attack and should not be
   rewritten.

**Not requested:** no change to the B1 NO-GO outcome, to B0 GO, to A permanent
NO-GO, or to C GO. §7's recurring version-gate cost stands on its own and the
ADR is right that it sinks B1 on cost-benefit independently of entitlement.
This is a correction to the *evidence claim*, not to the decision.

## 7. Safety boundary held by this review

No credential, token, cookie or Keychain secret value was read, printed,
exported or persisted. **No `security` invocation of any kind was made.** No
login, logout, revoke, rotation or re-authentication ran. **No account was
created, enrolled or authenticated** — the entitlement question was attacked with
documentary evidence only, exactly as the brief required. Vendor CLI execution
was limited to `claude --version` and `codex --version`. All other evidence is
read-only byte inspection of installed binaries, read-only source reads, and
fetches of public vendor documentation. No repository file was modified.
