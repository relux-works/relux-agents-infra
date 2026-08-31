# TASK-260720-bk6owf — rework rev3: the two invariants, swept document-wide

Producer run: developer archetype · Branch: `task-board/story/STORY-260831-yr0x81`
Commit: `d7c2de5` · Artifact: `.research/260831_extensible-auth-method-lifecycle.design.md`
(908 → 1047 lines) · Date: 2026-08-31

## The count, since the brief asked for it explicitly

**11 sites, not the 3 the reviewer cited.** That is the answer to the brief's
question, and it is not a rhetorical flourish: 8 of the 11 are in sections no
finding named, and 5 of those 8 carry the *same two claims* the reviewer said
were closed.

The brief said to report whether it came out exactly at the two cited. It did
not, and the gap is the point — the third round of this pattern would have been
the third round of fixing a citation.

### Invariant A — the custodian class is not declarable anywhere (6 sites)

| # | Site | Cited? | What it said |
| --- | --- | --- | --- |
| A1 | §8.3, the declares-clause | yes | "the custodian class is a property the method declares" — rev1 text, verbatim, in the section governing how a *new* method enters the system |
| A2 | §8.3, the add-a-method recipe | yes | "one registry entry and one adapter" — omits the §3.2 mapping arm, so an implementer does two of three required edits |
| A3 | §4.1 `describe()` | yes (as compounding) | listed custodian class as `MethodDescriptor` content the adapter supplies |
| A4 | §4.1 `refresh_capability()` | **no** | let the adapter declare *who refreshes* — which is the "Who refreshes" column of §3.2's class table, i.e. the class's own definition. An adapter returning `host` for `subscription-oauth` restores the pairing C1 removed under a different noun |
| A5 | §4.3 **Refresh** column | **no** | sat un-annotated directly beside a **Custodian** column explicitly marked "rendered", carrying the same three values |
| A6 | §6.1's eliminated-surface enumeration | **no** (finding 3's territory) | "No config file, flag, registry column, project table, remote-config key, plugin or environment variable" — the profile record is in none of those categories and is where the function's input lives |

A4 and A5 are the ones that matter beyond bookkeeping. C1's claim is about the
class; the *refresh owner* is the class restated as a verb, and it was still
declarable in two places after five sections had been corrected.

### Invariant B — `unknown` never collapses into `absent` or `active` (5 sites)

| # | Site | Cited? | What it said |
| --- | --- | --- | --- |
| B1 | §3.6 state machine | yes | `active\|degraded\|unknown ──logout──▶ retired-local ──remove──▶ removed`, unconditional |
| B2 | §3.5 `custody_state` | implied by B1's fix | `retired-local` was defined nowhere in 1047 lines — an undefined state with an unconditional edge to deletion |
| B3 | §3.3 **H2** | **no** | "is `unknown`, never `absent`" — omits "never `active`", while §10's H2 says both. The invariant's own statement was weaker in one of its two homes |
| B4 | §6.2 table, `active` row | **no** | the `absent` row was hardened to "every observation succeeded"; the `active` row directly above it was not. A denied `stat` of the fallback path reads as "no plaintext artifact" → `active` |
| B5 | §5.1 step 8 | **no** | "Presence … and absence of the fallback file. Present + no fallback → `active`" — same asymmetry, at enrol |
| B6 | §7.2 codex file-branch negative | **no** | "no `Codex Auth` item appears" is satisfied by a lookup that *failed* — a negative pin assertion forged by a read error, on the branch the whole file-store custody argument rests on |

(Six rows, five sites: B2 is B1's vocabulary half.)

**The pattern in that list is worth more than the list.** Every uncited leak is
on the *reassuring* side of its invariant. `absent` gets hardened because absence
obviously needs evidence; `active` reads like a default and gets missed — B3, B4
and B5 are all that same asymmetry. A declared *custodian* gets deleted; a
declared *refresh owner*, the identical claim with a different noun, survives —
A4 and A5. Sweeping for the rewritten sentence finds neither class.

## Finding 3 — the input surface, answered rather than asserted away

The reviewer's two acceptable branches were "admit the record to §3.2's checked
list and say what protects `method`" or "make the record non-authoritative", with
an explicit prohibition on asserting immutability harder. Taken the first branch,
plus a gate, because the honest answer to "what protects `method`" was *nothing*:

- **§3.2 gains a third checked bullet** stating the surface plainly: the class's
  sole input is a field of the same on-disk record §3.3 refused to store the
  class in, H3 compares coordinates, the pin compares a derivation, §6.2 compares
  presence, and a `method` rewritten to `api-key` passes all three unchanged.
- **Invariant C2**: at enrol and every launch, `f(method)` must agree with the
  recorded `backend`/`store_selector`. `vendor-opaque` admits only vendor stores
  (claude `keychain`, codex `file`/`keyring`); `host-owned` and
  `provider-delegated` admit only `external-file`/`env`. Mismatch →
  `E_AUTH_CUSTODY_INCONSISTENT`, non-zero, no child process.
- **C2 is labelled checked, not structural**, and its residual is stated rather
  than averaged: it converts a one-field edit into a two-field consistent
  rewrite, and defends against nothing that can write both. §6.1 now budgets
  **three** halves — eliminated (C1), CI-mitigated (5.3/6.4), launch-gated (C2)
  — and says so in its opening sentence.
- **Q13** carries the seal-the-record decision as a threat-model question, with
  what settles it, rather than a guess.

C1 is not weakened: there is still no field anywhere that names a custodian
class. Forging `method` lies about a *method*, which a gate can catch; it does
not declare a *custody*, which the type has no place to hold. The document says
that distinction explicitly so the next reader does not collapse it.

## Negative tests added (§6.4)

All three are narrowing mutants, not deletions:

- **C2** — flip `method` in a synthetic record leaving every other field
  byte-identical; the launch must refuse. A test that only fails when `backend`
  is also cleared proves the fields are read and nothing about their agreement.
  And assert the consistent two-field rewrite is **admitted**, so the test does
  not overstate a gate §6.1 books as mitigated.
- **§3.6 edge** — drive it by making the post-logout observation *fail*, not by
  making the credential absent. An absent credential exercises `already-absent`,
  which is the path allowed to delete, and would pass against an implementation
  that collapses the two.
- **`describe()`** — an adapter returning a descriptor whose class disagrees with
  §3.2 must fail to compile or be rejected at registration.

## One contradiction introduced and removed mid-rework

Gating the §3.6 edge first produced "`remove` is reachable only from
`retired-local`", which is false against §5.4's `leave-vendor-state` — the policy
that deliberately skips logout. Corrected: `leave-vendor-state` is drawn as the
one edge to `removed` that bypasses `retired-local`, and stated as *not* a hole,
because it produces no positive assertion at all rather than producing one from
an `unknown`. What D1 forbids is inferring retirement; declaring that you are not
retiring anything is the opposite move, and `--confirm` plus an unconditional
tombstone is what makes it auditable.

## Growth

908 → 1047 (+139 net; +193/-40). The brief said usefulness under pressure is a
property the decision depends on, so: no new top-level sections, no
restructuring, §9 and §12 untouched, §5.4 untouched. The additions are one new
invariant (C2, which the reviewer's finding 3 required), one open question, three
negative-test families, and the corrected claims themselves. §10 is now fourteen
checkable rows and §11 thirteen questions.

## Validation

| Check | Command | Exit |
| --- | --- | ---: |
| Cross-reference integrity | all 14 invariant ids + Q13 + both error codes resolve | 0 |
| Invariant A sweep | `grep -i custodian` minus negations/renderings → empty | 0 |
| Invariant B sweep | no ungated `unknown ──verb──▶` edge remains | 0 |
| §3.6 ↔ §3.5 vocabulary | every drawn state is defined; `removed` handled as terminal | 0 |
| Safety scan | no credential/token/key material in the artifact | 0 |
| Build | `go build ./...` (`tools/agents-infra`) | 0 |
| Vet | `go vet ./...` | 0 |
| Tests | `go test ./... -count=1` | 0 |

`go test`: 4 packages ok (83.8s / 2.5s / 146.8s / 15.1s), 1 with no test files.
No Go file is touched by this change, so the suite proves the tree still builds
and nothing regressed — it does not evidence the design, and nothing here claims
it does.

## Safety boundary

Held. No credential, token, cookie or Keychain secret read, printed, exported or
persisted. No `security` invocation. No login, logout, revoke, rotation or
re-authentication. Commands run: file reads, greps, a python rewrite of one
markdown file, `go build`/`go vet`/`go test`, and git reads and one commit on
this worktree.

**Commit signing:** unavailable in this environment — `gpg` reports "No secret
key" for `alexis <alexis@relux.works>`. `d7c2de5` is unsigned, as are this
branch's three prior commits (`%G?` = `N` on all four). Stated rather than worked
around.
