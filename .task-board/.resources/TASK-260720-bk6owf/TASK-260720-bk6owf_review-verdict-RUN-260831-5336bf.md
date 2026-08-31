# TASK-260720-bk6owf — review verdict, revision 3: ACCEPTED

Reviewer run: reviewer archetype · CR `CR-TASK-260720-bk6owf-3` revision 3
Base `d7c2de5` · candidate tree `bcc3725` · repository delta **empty**
Artifact reviewed: `.research/260831_extensible-auth-method-lifecycle.design.md`
(1049 lines) at `d7c2de5` · Date: 2026-08-31

## 0. Why `repository_delta=empty` is the right outcome here

It is not "the producer changed nothing". The producer committed its rework as
`d7c2de5` *before* the CR snapshot was taken, so `d7c2de5` became both the CR's
base OID and the commit carrying the work. The candidate tree is byte-identical
to its base because the delta is already in it.

Verified rather than assumed:

- `git show --stat d7c2de5` → `.research/260831_extensible-auth-method-lifecycle.design.md`
  (+181/-40) and `LOGBOOK.md` (+12). Two files, 193 insertions, 40 deletions.
- `git show --format= d7c2de5` is line-for-line the attached
  `TASK-260720-bk6owf_rev3-delta.patch` (389 lines both).
- `git status --short` in the worktree: clean. No unstaged rework hiding outside
  the snapshot.

So the reviewable delta exists and I reviewed it; it is reachable at `d7c2de5`
rather than at `base..candidate`. This is a snapshot-ordering artifact, not an
empty deliverable. Accepting it means accepting `d7c2de5`.

## 1. The brief's four questions

### Q1 — Is the sweep real and complete? **Yes.** No twelfth site.

I ran my own sweep before reading the producer's, on the rev3 text, and did not
take the producer's site list as the search space.

**Invariant A (the custodian class is not declarable anywhere).** Two passes,
because the producer's own two best finds (A4/A5) were *synonym* leaks that no
keyword sweep can reach:

- literal: every one of the 55 lines matching `custod*`, read in context. Every one
  is either a denial of declarability, a rendering, the admitted input surface
  (3.2's third bullet), or C2's gate.
- synonym: `refresh|rotat|owns|owner|holds the bytes`, on the theory that the
  class is the "Who refreshes" column of 3.2's table restated as a verb. 30 lines.
  §3.1's `Auth method` row says "how the credential was obtained and who
  *therefore* owns its refresh" — derived, not declared. §8.3's "a method that
  cannot refresh declares `refresh_capability: none`" is the narrowing §4.1
  explicitly permits. Nothing else names an owner an author could choose.

**Invariant B (`unknown` never collapses into `absent` or `active`).** Every
line matching `absen*` (33) and `active` (23), each read in context: 3.3 H2,
3.6's five bullets, 4.1 `status()`, 4.2's vocabulary, 5.1 step 8, 5.2 step 4,
5.4's classification table and its `Silence is unknown` paragraph, 6.2's six
rows, 6.4's edge test, 7.1, 7.2's file-branch negative, §10 H2/D1.

One candidate I examined and rejected: **6.2 row three** (`codex file` +
`auth.json` present at 0600 → `active`) does **not** carry row one's "every
observation succeeded" qualifier. It does not need it, and the asymmetry is
principled rather than missed. The qualifier exists to stop a *negative* conjunct
being forged by a denied read — row one's "no unexplained plaintext artifact",
row five's "no plaintext artifact". Row three has no negative conjunct: on the
`file` store `auth.json` *is* the credential, there is no separate fallback whose
absence must be established, and "present at 0600" cannot be concluded from a
`stat` that failed. Row six ("**including the store selector**") covers the one
other read the row depends on. The doc's own sentence — "rows one and five carry
the same qualifier for the same reason, in the two directions D1 forbids" —
states exactly that scoping.

Two independently-run sweeps converging on eleven is the strongest corroboration
available short of an implementation. **Sweep confirmed complete.**

### Q2 — Sound, or lucky? **Sound — but the soundness is human, not tooled.**

What makes it sound: the producer did not sweep for the rewritten sentence, it
derived a transferable predicate and then applied it. Two predicates, both stated
in the rework summary:

- *the class restated under a different noun* — which is why A4 (`refresh_capability`
  letting an adapter name the refresh owner) and A5 (4.3's un-annotated **Refresh**
  column) were caught. These are the same claim as a custodian column, and a grep
  for "custodian" cannot reach either.
- *every uncited leak is on the reassuring side of its invariant* — `absent`
  obviously needs evidence and gets hardened; `active` reads like a default and
  gets missed. B3 (3.3's H2 saying "never absent" but not "never active"), B4
  (6.2's `active` row), B5 (5.1 step 8) are three instances of that one shape.

A predicate that predicts unseen sites is an argument for exhaustiveness. Eleven
found by chance would not have clustered like that.

**The caveat, stated because the producer's own evidence overstates itself.** The
automated checks in `TASK-260720-bk6owf_rev3-validation.log` do *not* implement
those predicates:

- Check C2 greps the literal string `custodian`. By construction it cannot find
  A4 or A5 — the two finds that make the method worth anything.
- Its caption reads `(empty above == no declaring surface remains)` and sits
  under **four** printed lines (92, 118, 254, 749). All four are benign on
  inspection — a denial, the admitted input surface, C2's note, a negative test —
  but the check did not return empty, and the rework summary's validation table
  reports it as "`grep -i custodian` minus negations/renderings → empty | 0".

So the sweep's soundness lives in a manual read whose predicate is recorded only
in the summary artifact. Not a defect in the design; a limit on how much the
attached log evidences. Named here so the next round does not mistake the grep
for the method.

### Q3 — Did any fix weaken a claim to make it consistent? **No.**

I read all 40 deleted lines against their replacements. Fifteen distinct
reconciliations; every one strengthens or admits, none softens:

| Site | Direction |
| --- | --- |
| 3.2 "two things, neither a surface" → "three things, the third is" | admits a surface it had denied |
| 3.2 "argv, environment or plugin at all" | + names the one remaining path and its gate |
| 3.3 H2 "never `absent`" → "never `absent` **and never `active`**" | strengthened |
| 3.5 `method` row | admits immutability is unenforced, names C2 |
| 3.5 `custody_state` "Per 3.6" → full enumeration | B2's undefined state now defined |
| 3.6 `unknown ──logout──▶ retired-local ──remove──▶ removed` | unconditional edge → gated, with a refusing self-edge |
| 3.6 launch-refusal list | `retired-local` added to it |
| 4.1 `describe()` | class supplied by adapter → rendered by framework |
| 4.1 `refresh_capability()` | owner declared → rendered, adapter may only narrow |
| 4.3 Custodian column → Custodian **and** Refresh | closes A5 |
| 5.1 step 8 | requires a *successful* observation |
| 6.1 two halves → three | see below |
| 6.2 `active` row | + "every observation succeeded" |
| 7.2 codex file-branch negative | + "the lookup must have **succeeded**" |
| 8.3 "one registry entry and one adapter" + "a property the method declares" | → three edits, total function |

**The one that looks like a retreat and is not.** 6.1 went from *"the declaration
half is eliminated, the implementation half is mitigated"* to three halves, moving
newly-accounted risk into the mitigated column. That is a downgrade of the
document's safety picture — and it is the correct direction, because it is a
downgrade toward truth, and it is loud rather than invisible: 6.1 says "Three
halves is the wrong word and the right count", §10's C2 row is stamped
"**Checked, not structural** — a consistent two-field rewrite still passes", and
the residual is routed to Q13 as a threat-model decision rather than absorbed. The
brief's failure mode is a retreat the decision cannot see; this one is the only
thing the section is shouting.

**C1 was not weakened, and it survives attack.** C1 still reads "no input the
running system accepts can produce a different pairing". I tested the obvious
counter: rewriting the record's `method` from `subscription-oauth` to `api-key`
*does* reach the `host-owned` branch through an accepted input. It does not
falsify C1, because C1's pairing is (method → class) and that function is
unchanged — the record now lies about *which method it is*. The document draws
exactly that distinction at 3.2 ("forges a *method*, which is a lie the gate can
catch, rather than declaring a *custody*, which the type has no place to hold")
rather than leaving the reader to rescue it. That is a real distinction, argued,
not a rhetorical patch.

**The added gate is specified with the right negative.** C2's test in 6.4 is a
narrowing mutant — flip `method`, leave every other field byte-identical, launch
must refuse — with the delete-only variant explicitly rejected ("a test that only
fails when `backend` is also cleared proves the fields are read and says nothing
about their agreement"), *and* an upper bound asserted (the consistent two-field
rewrite must be **admitted**, so the test cannot overstate a gate 6.1 books as
mitigated). Same for the 3.6 edge test: drive it by making the observation *fail*,
not by making the credential absent, because absence exercises `already-absent` —
the path that is allowed to delete — and would pass against an implementation that
collapses the two. These are the shapes this review chain has been asking for.

### Q4 — Growth 908 → 1049: **reconciliation, not elaboration.**

Verified by hunk map. Sixteen hunks, last at old-line 891 (Q13's append). §9's
prerequisite tables (old 800–862), §11's Q1–Q12 and all of §12 are untouched. No
new top-level section; no restructuring. Every added block traces to a finding or
to an invariant: 3.2's third bullet + C2 + C2's residual, 3.6's gated machine,
6.1's three halves, 6.4's three negative-test families, 8.3's third edit, Q13,
and the corrected claims themselves.

## 2. Discrepancies found (recorded, non-blocking)

- **D1 — the mirrored deliverable on the board is stale.** Board resource
  `TASK-260720-bk6owf_extensible-auth-method-lifecycle.design.md` is still the
  **rev2, 908-line** document — verified by `diff` against the repo file: 221
  differing lines, all of rev3's reconciliations absent. The rev3 design exists
  only in the repo at `d7c2de5` and inside the delta patch. Anyone who reads the
  deliverable *from the board* — which is where `TASK-260720-3gcfd1` would look —
  reads the version with all eleven contradictions still in it. **This must be
  refreshed before the architecture decision is spawned.** Not blocking the
  verdict, because the accepted artifact is the committed one and the fix is one
  `resource update`, but it is the single highest-consequence item in this review.
- **D2 — the rework summary's numbers are slightly off its own artifact.** It
  says 1047 lines twice; the file is 1049. It reports "+193/-40" for the design
  file; that is the two-file total — the design file is +181/-40 and `LOGBOOK.md`
  is +12. Small, and in a document whose entire subject is not approximating
  facts you could measure.
- **D3 — validation check C2's caption contradicts its output**, as described in
  Q2. Self-disclosing (the four lines are printed), so nothing is hidden.

## 3. Carry-forwards for `TASK-260720-3gcfd1` (new; no prior round covered these)

None is an invariant leak and none blocks acceptance. All are fail-safe in
direction. Recorded so the decision prices them.

- **CF1 — the lifecycle is specified for `vendor-opaque` only, but §4.3 markets
  three classes as "available".** 3.3's population table gives `observed_coords`
  for claude-keychain, codex-`file` and codex-`keyring` — every row is a
  `vendor-opaque` profile. H3 ("re-derived and compared at every launch; drift
  refuses") therefore has **no defined derivation** for a `host-owned` or
  `provider-delegated` profile, and 5.1's enrol is written end-to-end as a vendor
  handoff (step 7) with no host-owned branch, though 4.1's `start()` implies one
  exists ("*For `vendor-opaque`* the only legal outcome is `handoff-to-vendor`").
  A decision reading `anthropic | api-key | host-owned | host | host-side |
  available` would price it as modelled; it needs a whole agents-infra secret
  lifecycle that §9 does not list as a prerequisite. Either scope §4.3's "Status"
  column to grammar-level extensibility (AC6, which *is* satisfied), or add the
  host-owned lifecycle.
- **CF2 — `memory` is a backend no class admits.** 3.3's enum is `keychain |
  keyring | file | memory | external-file | env`; C2 admits `keychain`/`file`/
  `keyring` for `vendor-opaque` and `external-file`/`env` for the other two.
  `memory` is admitted by nothing, so any record carrying it refuses at every
  launch. Fail-safe, but dangling: drop it from the enum or say which class takes it.
- **CF3 — 3.6 draws no recovery edge out of `unknown` or `degraded:*`.**
  `custody_state` is simultaneously a stored record field (3.5) and recomputed at
  launch (5.2 step 4). Read as a stored latch — which the drawn machine invites,
  since every drawn edge into `unknown` is one-way — a single denied `stat` pins a
  profile at a state that refuses to launch forever. 3.6's prose intends recovery
  ("repair the observation"), it just is not in the machine. Same class as B2
  (`retired-local` defined nowhere), which this round fixed.
- **CF4 — the version range has two declared sources and no precedence rule.**
  4.1's `describe()` returns a "supported version range"; 9.1's P-AM-4 requires
  `Capabilities` to expose one "so the gate (7.1) and the pin (7.2) have a
  declared source rather than a hardcoded constant". V1 is a refusing gate; two
  authors for its bound, and nothing says which wins.
- **CF5 — one negative variant in 7.2 is still satisfiable by an error.**
  "*Negative — wrong branch*: the four keyring assertions above, run against a
  `file`-store profile, must **fail**" is satisfied by *any* failure, including a
  denied keyring lookup on a build where store selection is broken. Same family as
  B6, which sits one bullet below and *was* hardened with "the lookup must have
  **succeeded**". Not an instance of invariant B — it makes no `absent`/`active`
  claim — which is why I did not count it as a twelfth site, but it is the same
  shape and deserves the same sentence.

## 4. Acceptance-criteria and DoD check

| AC | Status | Where |
| --- | --- | --- |
| 1 provider / alias / identity / method / handle separated | met | 3.1 (six rows for five separations, `runtime` split out and justified) |
| 2 adapter: start, continue, status, refresh, logout, revoke, local-delete | met | 4.1, all eight operations |
| 3 email/OTP as claims and inputs, never primitives or secrets | met | 3.4, S1, 8.4 (`E_AUTH_SECRET_ON_ARGV` makes it a parser property) |
| 4 logout scoped to one local profile; remote revoke reported separately | met | 4.1, 5.4 — two independent fields, neither inferred from the other or from an exit code |
| 5 remove explicitly destructive, gated on resolved logout policy | met | 5.4, `--logout-policy` + `--confirm`, tombstone rules |
| 6 grammar extensible to OAuth / device code / API key / SSO / Coding Plan | met at the grammar level | 8.1–8.3; see CF1 on the lifecycle level |
| 7 ambiguous identity fails safe | met | 8.2, A1 — no heuristic, prints every candidate |

DoD: data model separated ✔; lifecycle covers both providers' real custody models
rather than an idealised shared one (5.3's `file`-store paragraph and 6.2's rows
three and four are the whole point) ✔; the plaintext-fallback hazard addressed
with the "no mechanism denies the CLI its own credential" rule explicit (fact 2,
5.1 step 5, 5.3) ✔; `CLAUDE_CONFIG_DIR` treated as version-gated and undocumented
with a stated range and a refusing gate (7.1, `E_AUTH_VERSION_UNPINNED`, "refuses,
does not warn") ✔; `skill-agents-management` prerequisites named as prerequisites
(9.1, P-AM-1..4, each with the negative test that gives it meaning) ✔; open
questions with what settles each, ready for 3gcfd1 (§11, thirteen) ✔; no credential
value printed, exported or persisted and no live session touched — held by the
producer per its log, and independently by this run, which executed only file
reads, greps and git reads ✔.

## 5. Verdict

**ACCEPT.** The sweep is real, independently reproduced, and complete at eleven.
The method is sound rather than lucky, on the strength of two transferable
predicates that predict the uncited sites — with the caveat that the attached
automated checks do not implement them. Nothing was reconciled by retreating; the
one downgrade is a downgrade toward truth, stated loudly and routed to an open
question. Growth is reconciliation, not elaboration, and the sections the brief
cared about are untouched.

Required before `TASK-260720-3gcfd1` is spawned: refresh the stale board copy of
the design document (D1). It is the version the decision would otherwise read.
