# TASK-260720-3gcfd1 — review round four verdict

**Change Request:** `CR-TASK-260720-3gcfd1-4` revision 4
**Reviewer run:** `RUN-260831-9b5586`
**Base → candidate:** `5feebbb1` → `e4e3a4eb` (candidate tree == branch `HEAD` `5c84ee78`; worktree 0 commits behind `main`)
**Verdict: CHANGES REQUESTED → `analysis`.**

---

## Summary

Round four does what the brief asked. The sweep was run, the six unrun free reads
were run, the two contradictions (N18, N19) were **absorbed into conclusions**
rather than footnoted, the 403 holds as a failed read everywhere, the growth is
the reads and their consequences, and **every note I spot-checked reproduces at
the exact cited line numbers**.

It fails on the mirror image of the defect it was chartered to fix. Round four
propagated its two **adverse** findings thoroughly and left its two **favourable**
ones (N13, N16) standing-as-open at **five sites**, three of them in sections that
are named acceptance-criteria deliverables and one of them a self-contradiction
inside §13, the corrections table itself. The document therefore reads more
pessimistic about codex than its own round-four evidence supports, at exactly the
places a reader goes for the comparison and the hazard.

That is the same shape as "a refutation recorded as a footnote next to the
unchanged claim" — with the polarity flipped. §13's sqlite correction row
enumerates the nine sites it fixed; §6 is not in that list, and §6 received **no**
round-four edit at all.

---

## What passed

### Attack 2 — the two contradictions changed conclusions. CONFIRMED.

- **N19 / §5.3.** Not a note. §5.3 is rewritten head to tail (`@@ -838,26 +1367,56`),
  qwen is promoted from footnote to a row in §0's decision table, the verdict text
  changes from *"not modellable"* to *"provisional-with-a-mechanism"*, the plugin
  defect becomes **W22** as a B0 prerequisite with gate **G-B0-9**, the standing
  limit (`main`, no shipped-build corroboration) is stated explicitly and carried
  as **Q18**, and the parallel claim about muse/gemini/agy/pi is withdrawn as
  *unaudited* rather than silently kept. §13 carries the correction row.
- **N18 / Q7.** Answers against the design and is acted on: Q7's own row is
  rewritten to say the proposed test *"cannot pass as written"*, replaced by
  **H14**, and §13 carries the correction. It is not softened — the ADR states the
  honest split (file-level disjoint, directory-level nested) rather than picking
  the flattering half.

**Reproduced N18 locally, exactly:** `scripts/setup.sh:69` is
`CONFIG_DIR="$HOME/Library/Application Support/agents-infra"`;
`scripts/setup.ps1:13` is the `%APPDATA%\agents-infra` line;
`source_dir.go:387-404` is `configDir()` with the same three branches;
`write_install_state` at `setup.sh:208-211` does `mkdir -p "$CONFIG_DIR"` and
writes `$CONFIG_DIR/install.json`. `grep -n CONFIG_DIR scripts/setup.sh` returns
six hits and **no recursive operation** — the "file-level disjoint today" half
checks out too.

### Attack 3 — the 403 stays a failed read. CONFIRMED, everywhere.

Checked all six named surfaces. §0 (line 165), §2.3 (523-532), §2.4 sweep row
(824), §4 Q2b-b (1284), §11.2 W18 (2071), §13 (2397), ledger 24/25/26/42.
`README.md:1690-1691` — *"second-hand at that, since `help.openai.com` returns 403
from this host. Entitlement there is **unestablished**"*. `LOGBOOK.md:33,48` —
*"A successful read returning nothing is an absence; a 403 is not."* No use has
regressed to "no evidence". Round four also **extends** the standard to a new
tool: ledger 39 labels a `gh api search/code` `total_count: 0` as a failed read
and re-labels ledger 23 accordingly, having caught it returning zero for a
constant that is demonstrably in the target file.

### Attack 4 — growth is reads and consequences. CONFIRMED.

1754 → 2512 (+758, +43%). The diff is 946 added / 188 removed across 25 hunks.
The single largest hunk is `@@ -729,6 +798,453 @@` — 453 lines, which is §2.4 and
its seven notes, i.e. the reads themselves. The remaining ~490 are spread over
§0, §4, §5.1, §5.2, §5.3, §7.3, §9, §10, §11.1, §11.2, §12.2, §12.4, §13, §14,
§15 — consequence propagation, not accumulation. Removals include real deletions
(§9's old decision paragraph, §12.2's long codex list). Not padding.

### Attack 5 — the notes reproduce. CONFIRMED, better than required.

Six checked against primary sources, all at the cited line numbers.
`rust-v0.150.1` → `0eb410ad0dd161ea323b05452f978de01cd63430`, matching the ADR.

| Note | Claim | Reproduced |
| --- | --- | --- |
| **N13** | `config_toml.rs:329-331` doc comment; `mod.rs:3918-3923` resolution; `mod.rs:242-252` `resolve_sqlite_home_env`; `state/src/lib.rs:106-107` `SQLITE_HOME_ENV`; `log_dir` at `mod.rs:3906-3910` | **Exact, all five.** Precedence `config.toml` › `$CODEX_SQLITE_HOME` › `$CODEX_HOME` confirmed |
| **N15** | env constants at `manager.rs:199-201` and `:910-912`; API-key precedence at `:1456-1462` with the vendor comment; `CODEX_ACCESS_TOKEN` at `:1492-1497`; `refresh_token_endpoint()` at `:1717-1720` with no allowlist | **Exact, all five.** The `// API key via env var takes precedence over any other auth method.` comment is verbatim |
| **N16** | both fallback `warn!` legs only inside `impl AuthStorageBackend for AutoAuthStorage`; `Keyring` → `create_keyring_auth_storage` with no file leg | **Exact.** `grep -n 'falling back to file storage'` over the whole file returns **only** `:437` and `:447`, both in `AutoAuthStorage` |
| **N17** | `types.rs:107-118`, four variants, `#[default]` on `File` | **Exact** |
| **N18** | installer write set vs. auth root | **Exact** (above) |
| **N19** | `storage.ts` `getGlobalQwenDir():193-203`, `getOAuthCredsPath():640-642`, `QWEN_RUNTIME_DIR:172-190` | **Exact.** `QWEN_HOME` present; `HomeEnvVar: ""` confirmed at `skill-agents-management/pkg/agentic/systems/qwen/qwen.go:121` |
| **N20** | 174 distinct `process.env.CLAUDE_*`/`ANTHROPIC_*` reads in the 2.1.248 bundle; the named-profile store | **Exact 174.** `ANTHROPIC_PROFILE`, `active_config`, `WIF profile`, `credentials-file`, `user_oauth`, `oidc_federation` all present in the installed build |

### Boundary and safety. HELD.

No credential value anywhere in the delta — the only key-shaped string is
`sk-SYNTHETIC-NOT-A-REAL-KEY-a`. The two `security find-generic-password` rows in
the delta belong to `1g880w`'s inherited custody research, not to this task's
ledger; the ADR's own §14 header claim (*"No `security` invocation was made in
this task"*) is consistent with its rows 1-42. No logout, revoke or rotation.

---

## Findings — why this is not accepted

### F1 (blocking) — §6 "Options compared" was not swept at all, and asserts a closed risk as live

`.research/260831_multi-account-auth-architecture.adr.md:1469`, B1 verdict row:

> …the asymmetry … sits somewhere else: not entitlement, not the version gate, but
> that Anthropic **documents and recommends** the workflow … while OpenAI documents
> neither — **and leaves `sqlite_home`'s default unstated, which is a live
> cross-account leakage risk (Q15)**.

Both halves are false as of round four. N13 establishes the default **is** stated,
in the vendor's own doc comment, and that it resolves **inside** `CODEX_HOME`;
§5.2 says *"the objection this ADR called 'the substantive risk' for a full round
does not exist"*; §4's Q15 row says ANSWERED, favourably.

This is not an oversight the document can absorb, because **§13 enumerated its own
correction sites and missed this one.** Line 2406 lists nine:
`§0 / §5.2 / §9 / §10 / §11.2 / §12.2 / §12.3 / §15 / README.md`. §6 is absent —
and hunk analysis confirms §6 received **zero** round-four edits (the diff jumps
from `@@ -838,26 +1367,56 @@` to `@@ -1101,13 +1660,21 @@`, skipping §6 entirely).
Round four invented the enumerate-and-confirm discipline for *reads* and did not
apply it to its own *corrections*.

**Same section, second staleness.** The B0 "Security" row still frames the whole
problem as *"a stray `CLAUDE_CONFIG_DIR`, `CLAUDE_SECURESTORAGE_CONFIG_DIR` or
`CLAUDE_CODE_CUSTOM_OAUTH_URL`"* — the three-input sizing that §0, §5.1, §5.2, §9
and §13 all now say was *"the wrong enumeration in every previous round"*. After
N15 the worst member of that set is `CODEX_API_KEY`, which does not merely pick
the wrong account but makes a composed launch **run as the wrong identity while
reporting the right one**. §6's B0 "What it gives up" row also still says
*"Q11 exists to check nobody depends on it"*, after N20a partly answered Q11.

§6 is the section that satisfies the AC *"compared options with security,
concurrency, refresh and revocation tradeoffs, and what each option gives up"*.
A reader who opens the options comparison — the natural place — gets a codex
leakage risk the rest of the document calls closed and a B0 threat model the rest
of the document calls under-sized.

### F2 (blocking) — §8 carries a withdrawn finding as one of "two things, both adverse"

`…adr.md:1763-1770`, §8 (*"The plaintext-fallback hazard, carried forward"* — an
explicit Definition-of-Done item):

> **Two things this ADR adds to that picture, both adverse:**
> - **N4.** Codex's `keyring` store has a load-and-save fallback path to file
>   storage. **If it is ungated**, codex inherits the same fail-open shape and
>   §5.3's prohibitions become load-bearing on the codex-keyring branch too…

N16 answered the conditional: the fallback **is** gated on `Auto`, both legs live
in `AutoAuthStorage` and nowhere else, and §13 line 2407 records the finding as
**"Withdrawn — there was no conflict"**. §8 was untouched in round four (verified:
the diff contains no `+`/`-` line matching `ungated` or `both adverse`). So the
hazard section still counts a retired item toward its adverse total, and still
tells a reader that `bk6owf` §6.2's `degraded:plaintext` classification for
codex-keyring is *"now better supported than when it was made"* — which N16
removes the support for.

The document demonstrably knows how to fix this in place without rewriting
history: §0 line 183-184 strikes the retired Q15 clause with `~~…~~` and states
the withdrawal inline, and `LOGBOOK.md:29,30,35` does the same. §8 got neither.

### F3 (blocking) — §13's corrections table contradicts itself on Q12

Two rows of the same table, both present-tense:

- `:2395` — `bk6owf` Q12: *"**Narrowed and then sharpened adversely** (N4, N12a):
  a fallback path exists on both legs … **The two sources conflict**, so `keyring`
  cannot be relied on to mean 'never plaintext'. **Still not answered.**"*
- `:2407` — *"**Withdrawn — there was no conflict** (N16). Both fallback legs are
  in `impl AuthStorageBackend for AutoAuthStorage` and nowhere else."*

§13's stated purpose is *"Listed together so a later reader does not have to
reconstruct them."* A reader who consults it on Q12 gets both answers and no
ordering. This is the one finding with no interpretive latitude at all.

### F4 — §11.3's "do not begin" block still promises Q15 could close B1-codex

`…adr.md:2153-2160`:

> It now means §12.2's items 1–3 have been done: they are **ours and free**, and
> **one of them (Q15's `sqlite_home` half) could still close B1-codex on a real
> ground.**

Round four ran it. §0 says *"both came back favourable; **neither settled
anything**"*; §12.2 item 1 is marked *"✅ DONE"*; §12.4 is built on exactly this
lesson. This paragraph is a live instruction block governing when H1–H13 may
start, not narration, and it still points at a discharged item as the thing that
could decide the verdict.

### F5 — the sweep's reclassification: 2 of the 4 have a free documentary half

§2.4.1 moves four of twelve items out of "free read": *"Q3 is free only once a
second enrolled account exists; Q4, Q5 and Q6 are free synthetic experiments that
need a harness, not documents."*

- **Q3 — correct.** Genuinely needs a second enrolled account.
- **Q6 — correct.** The ADR already answers it in principle (*"It cannot in
  principle"*); the experiment is confirmatory only.
- **Q4 — not correct.** *"Is the codex derivation stable across versions?"* is
  answerable by reading the derivation at several `rust-v*` tags — the exact
  method §2.4.2 establishes and then uses seven times. The reclassification
  routes the item to the **only half that cannot run on this host**: the pin test
  needs *"every installed codex version"* and exactly one codex is installed. The
  free half is decision-relevant, because N17 itself observes that
  `#[default]` *"is exactly the kind of constant that moves between releases
  without a doc change"* — and §7.3.3's surviving codex version-gate cost rests on
  that constant.
- **Q5 — not correct.** *"Does Claude canonicalize the config dir beyond NFC?"*
  The derivation path is readable JS in the installed bundle; N20 demonstrates the
  method by extracting five verbatim function bodies from that same file.

Two of four is not a fabricated reclassification, and the sweep is broadly honest
— but "this is not a documentary read" is precisely the move that licensed
skipping Q2b in round one, and it was applied here without checking whether a
documentary half existed. Naming the free half of Q4 and Q5 (run or not) is
enough to discharge this.

*Minor, non-blocking:* §2.4.1's *"§4's cost column names **twelve** items as
free"* no longer re-runs against the table as it now stands — Q16/Q17/Q18 were
created **by** the sweep and are also marked free, so a reader counting today gets
fifteen. The twelve is right for the pre-sweep table; say so.

---

## The pattern worth recording

Round four's own thesis is that a document must be audited against the work it
ranks for itself. It applied that to *reads* and produced a genuinely better
document. It did not apply it to *corrections*: §13 enumerated nine sites for the
N13 correction and there were at least eleven, and the N16 withdrawal got a §13
row and a §5.2 row but no sweep at all.

**The residue is directional.** N18 and N19 — the two findings that make the
document look worse — were driven into §5.3, §0, §4, H14, W22, G-B0-9 and §13.
N13 and N16 — the two that make codex look better — reached §0, §4, §5.2, §12.2
and §13 and stopped. Every one of the five stale sites overstates a codex hazard.
A document that propagates bad news faster than good news is biased even when
every individual note is correct, and this one has spent four rounds establishing
that its own asymmetries are worth measuring.

None of this moves a verdict. A permanently NO-GO, B0 GO, C GO, B1-claude
CONDITIONAL GO, B1-codex HELD, qwen provisional-with-a-mechanism all survive
these findings intact, and the evidence base under them is sound. The rework is
a correction sweep, not new research.

---

## Rework

1. **§6** — strike the *"live cross-account leakage risk (Q15)"* clause from the
   B1 verdict row and state N13's answer; widen the B0 Security row past N1's
   three variables to the N15/N20 ambient-input classes, naming the wrong-identity
   consequence of `CODEX_API_KEY`; update the Q11 reference to N20a's reduced
   scope. Add §6 to §13's site list.
2. **§8** — mark the N4 bullet withdrawn per N16 (the `~~…~~` convention §0 and
   the logbook already use), and correct *"both adverse"*.
3. **§13** — reconcile the `bk6owf` Q12 row at `:2395` with the withdrawal row at
   `:2407`; one of them must stop asserting *"Still not answered."*
4. **§11.3** — correct the *"could still close B1-codex on a real ground"*
   sentence to what round four found.
5. **§2.4.1** — either run the documentary halves of Q4 and Q5 or record them as
   free reads that exist and were not run, with the reason. Do not leave them
   classified as experiment-only. Fix the twelve/fifteen count to say which table
   it counts.
6. **Then re-run the sweep discipline on the corrections themselves**, not just
   the reads: for every round-four note, grep the document for the claim it
   retires and confirm each site was updated. That is the general fix, and it is
   the same fix round four applied one level up.

`README.md` and `LOGBOOK.md` are already correct on all of this and need no
change.
