# TASK-260830-s5ro4e — review verdict, document revision 12 (CR revision 10)

**Verdict: ACCEPTED.** Both blocking findings from revision 11 are closed, closed by
*fixing* rather than by trimming, and closed in a way I could break on purpose and watch
fail. Nothing was displaced to stay under the 800-line cap — the plan **grew** 762 → 800.
The three minor items from the last verdict are fixed too.

Reviewed: `51675d6` (plan 800 lines, runbook 348, review record 238) against `6e26e0b`
(revision 11). The plan delta is seven hunks and I read all seven; every property the
previous compression audit cleared in revision 11 is in untouched text.

## On the empty repository delta

`repository_delta=empty` because the CR's base OID `51675d6` **is** the head that contains
revision 12 (`0feb3a8` plan/runbook/record, `51675d6` logbook). The reviewable delta is
`git diff 6e26e0b 51675d6` — 162 insertions, 27 deletions across three research documents
and `LOGBOOK.md`. Same snapshot-timing artifact as revision 11, not a producer that
changed nothing. The right outcome for this leaf is a document, and the document changed.

## F1 — CLOSED. P2 no longer fails open under zsh, and silence is no longer a pass

The fix is the right one at both levels. The shared `scripts/lib/`\*`.zsh` glob is gone and
each installer has a **named** operand list; `set_of` refuses an operand it cannot read;
and the pass condition is a positive `P2-SET-OK <n>` sigil instead of absence of output.
That last part is what actually kills the class — a check whose pass condition is silence
cannot distinguish success from not running, whatever the operand list looks like.

Driven against the real `relux-agents-infra` and `skill-project-management` installers,
per-branch, with one injected undecided override `NEW_UNDECIDED_INPUT="${SOME_BRAND_NEW_OVERRIDE:-x}"`:

| Case | bash | zsh |
| --- | --- | --- |
| agents-infra, identical pair | `P2-SET-OK 19` — passes | `P2-SET-OK 19` — passes |
| agents-infra, injected override | `> SOME_BRAND_NEW_OVERRIDE`, no sigil — **stops** | identical — **stops** |
| task-board, identical pair | `P2-SET-OK 38` — passes | `P2-SET-OK 38` — passes |
| task-board, override in the *sourced* lib file | `> SOME_BRAND_NEW_OVERRIDE`, no sigil — **stops** | identical — **stops** |
| task-board, named operand unreadable | `STOP P2 unreadable operand …`, no sigil — **stops** | identical — **stops** |

Ten runs, two shells, identical verdicts throughout. Revision 11's text on the same
injected case, for contrast: bash reports the name, zsh prints `no matches found` on stderr
and the `diff` never runs — the undecided input is invisible. The fix, not the environment,
is what changed the answer.

Part II also now names its interpreter (`Run every Part II command under bash`), which was
the root enabler and was previously unstated anywhere.

## F2 — CLOSED. Step 0d reads `attached_clients` strictly, and the cross-check earns its place

`rows=d["sessions"]`, a list check, a row-count cross-check against
`session_count + quarantined_count`, `int(r["attached_clients"])` indexed strictly, and a
`|| STOP … is unknown, not zero` arm. Isolated against the shipped slice with a live fake
daemon, a stubbed census so no real board is ever enumerated, and a `session status` that
reflects the pid's real fate:

| `session list` payload | daemon | record written |
| --- | --- | --- |
| `{"sessions":[{"attached_clients":0}]}`, count 1 | terminated | `attached=0` — control, and the record is a fact that *was* read |
| `{"sessions":[{"attached_clients":2}]}` | alive | none — `STOP … has 2 attached client(s)` |
| `{"sessions":[{"session_id":"s1"}]}` | **alive** | none — `KeyError: 'attached_clients'` → unknown |
| `{"items":[{"attached_clients":9}]}` | **alive** | none — `KeyError: 'sessions'` → unknown |
| `{"sessions":null}` | **alive** | none — `sessions is not a list` → unknown |
| `{"sessions":[]}` vs count 1 | alive | none — `0 rows vs status 1` |
| 2 rows vs count 1 | alive | none — `2 rows vs status 1` |
| 2 rows vs `session_count 1 + quarantined_count 1` | terminated | correct — the cross-check does not over-refuse a quarantined row |
| status lacks `quarantined_count` | alive | none — `KeyError` aborts under `set -euo pipefail` |
| `attached_clients: "0"` (string) | terminated | tolerated by `int()`; matches the Go type's wire shape |
| `attached_clients: null` | **alive** | none — `TypeError` → unknown |

The three shapes I reported as blocking last round now all stop, and none of them writes
`attached=0` into `r1-daemons-stopped.txt`.

**Two mutants, because one would not have bounded it.**

- *Delete* — restore revision 11's `d.get("sessions") or []` / `r.get("attached_clients") or 0`:
  all three shapes go back to `rc=0`, daemon terminated, `attached=0` recorded. The gate
  covers exactly the class it claims.
- *Narrow* — keep the strict index, remove **only** the row-count cross-check: the absent
  key is still caught, but four rows against `session_count: 1` are admitted and the daemon
  is stopped. So the cross-check carries its own class and is not redundant with strict
  indexing. A delete-only mutant would have credited everything to the index.

**And the half that cannot be fixed here is recorded rather than papered over.** Limit 7
states plainly that against a protocol-mismatched daemon — the only situation 0d exists
for — a lenient Go-side `json.Unmarshal` can zero a renamed or dropped field before the
shell sees it, so a well-formed `attached_clients: 0` may be a decode artifact; PRE-1 is
the structural fix, and until it ships such a board is dispositioned by its owner. That is
the honest answer, and it is the one I asked for.

## The size line — nothing was displaced

Revision 11 was 762 lines, revision 12 is 800. The fixes were paid for out of headroom, not
out of other properties. The plan diff is seven hunks: the 0d prose, the Part II shell
directive, P2, P5, the R1 0d summary line, the two step-table cells, and Limit 7. All seven
are additions or in-place strengthenings. There is no hunk that removes a property, and the
compression audit that cleared revision 11 therefore still holds for everything outside
those seven.

The three minors from the last verdict are all fixed and I checked each: Step 3 regained
`agents-infra full tests/build/verify plus the real target/compose/Pi refusal gates on the
exact release head`; Step 5's rollback regained the full recovery-PATH `env -u
AGENTS_INFRA_SOURCE_DIR` form; and P5's phantom `release-set.txt` is gone.

**P5 is now a real command sequence, and I drove it too** — control silent; added binary,
dropped role, unreadable `setup.sh` and missing `.roles` each produce visible diff output
plus the read error on stderr. A partial read cannot look like a match: with task-board's
`setup.sh` removed the binaries diff reports all five task-board names as missing.

## Still true, checked directly

- **UNTESTED is in the documents, not only in the handoff.** `Status: UNTESTED (P8)` on R1;
  the four-row exercised/real/status table with `R2 | no | no | UNTESTED`; "**R2 is more
  unknown than R1**" and "R2 has no execution evidence at all"; UNTESTED in all seven Part
  IV step rows (Step 1 correctly `PARTIALLY TESTED (source build/test only)`); the runbook
  header's "**Every script here is UNTESTED against a real installed pair.**"
- **Every step row carries** repository+version, must-remain-working capabilities, window,
  and a concrete rollback command sequence. All six windows (`W2`, `W3`, `W2d`, `Wimg`,
  `W5`, `W6`) carry duration and impact, `W2d` still with "no upper bound" and "the impact
  is silence".
- **The corrected consumption census survives in the correct direction** — "task-board
  imports compatibility packages but directly imports neither `pkg/plugin` nor
  `pkg/inferenceengine`", and Step 4 still says "A zero direct-import count cannot satisfy
  this gate."
- **Derived-not-literal** is intact: the installed-pair row is explicitly "the recovery
  baseline is derived from the installed artifact at execution time, not from this row";
  the daemon-starting command set is "**derived, never restated here**".
- **The downgrade direction is still handled by stopping the daemon**, not by a third prose
  prohibition: prevention first (P7c stops the step if `controlProtocolVersion` changed),
  then R1 step 0d ends the daemon through its own SIGTERM/`Drain` arm before any executable
  moves, never escalating to SIGKILL, so the dangerous state is never created.

## Minor, non-blocking, for whoever touches this next

1. `unexpected-bin-entries.txt` is built with `… | grep -vxF -f "$MANIFEST/bin-inventory.txt" > … || true`
   (runbook `:252-253`). `|| true` is needed for grep's exit 1 (nothing unexpected) but also
   swallows exit 2, and step 0c validates `binaries.txt`/`roles.txt`/`install.json` but not
   `bin-inventory.txt` — so an unreadable inventory yields an empty "nothing unexpected"
   report. That is a failed read presented as an absence, two scripts after RB-1's own
   "Never add `|| true`". Nothing gates on the file, so it corrupts a record, not an action,
   and it is pre-existing since revision 9 — not a revision-12 regression.
2. Part II names `bash`; RB-2, RB-3 and RB-4 still name no interpreter (only RB-1 does).
   The F1 lesson was "name the interpreter"; it was applied to Part II and not to three of
   the four scripts. One line each.
3. P5's pass condition is still absence of output rather than a sigil. Unlike revision 11's
   P2 it fails loudly on stderr and carries `test -s` on the binaries half, so it is not the
   F1 shape — but P2's sigil form is the better pattern if the document is ever touched again.

None of these changes a rollback decision, and none is worth a thirteenth cycle.

## Definition of Done

| Item | Status |
| --- | --- |
| Release order with repository, version, required capabilities, rollback commands per step | met — Part IV table, 7 rows |
| Every unsupported-contract window named with duration and impact | met — 6 windows, incl. the two daemon-skew windows |
| Untested rollbacks labelled UNTESTED in the document | met — R1 `UNTESTED (P8)`, R2 `no execution evidence at all` |
| In-flight spawn behaviour stated per step | met — `W2`/`W3`/`W5`/`W6` stage tables and the `w*-*.txt` capture |
| task-board's consumption of the agents-management contract measured, not assumed | met — census, corrected direction, live derivations |
| Plan attached as a board resource | met |
| No migration step, release or tag performed | met — by the producer, and by this review |
| Gates attacked, not read | met — 10 P2 runs, 11 step-0d payloads, 2 mutants, 10 P5 runs |

## Evidence

- `TASK-260830-s5ro4e_review-evidence-rev12.log` — every run above, verbatim.
- Probes: `.temp/TASK-260830-s5ro4e/rev12-review/{p2-probe.sh,f2-probe2.sh,p5-probe.sh}`,
  driving `p2-{tb,ai}.frag`, `r1-0d.sh` and `p5.frag`, all extracted programmatically from
  the shipped documents at `51675d6`.
- Independently reproduced, not accepted from the producer: my results match
  `TASK-260830-s5ro4e_validate-revision12.log` on every overlapping case.
- No migration step, release, install or tag was performed. No live daemon was signalled —
  the census was stubbed for the isolated run, and the two real `tb-sessiond` processes on
  this host hold the same pids before and after every probe. No credential, token or
  environment value was read, printed or persisted.
