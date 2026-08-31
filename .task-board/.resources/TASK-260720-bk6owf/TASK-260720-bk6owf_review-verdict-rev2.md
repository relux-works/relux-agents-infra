# TASK-260720-bk6owf — review verdict, revision 2

Reviewer run: `RUN-260831-4bba37` (reviewer archetype)
Change Request: `CR-TASK-260720-bk6owf-2` revision 2
Artifact: `.research/260831_extensible-auth-method-lifecycle.design.md` (908 lines)
Branch: `task-board/story/STORY-260831-yr0x81` · Date: 2026-08-31

## Verdict: CHANGES REQUESTED → `analysis`

The five rev1 findings: **F1, F4, F5 fully closed. F3 closed where it was
found and leaking one line below the edit. F2 closed in five sections and
contradicted verbatim in a sixth the producer never opened.**

Three findings. All three are the shape the rev2 brief predicted — a
true-sounding claim standing next to its own contradiction — and none of them
needs new research. This is a short rework, not a restructuring.

---

## 0. On the empty repository delta

`repository_delta=empty` is a snapshotting artifact again, not an absent
deliverable. Base OID `7a4b52d` is the producer's own last commit on this
branch. The work is `c3a467a` (+277/-51 on the design) and `7a4b52d`
(LOGBOOK). Reviewed against those commits.

Design-only remains the correct shape for this leaf: scope says "architecture
model and CLI semantics only", §9 establishes that three named repositories
lack the contracts, and this feeds `TASK-260720-3gcfd1` as a decision input.
Writing code here would have been the failure. The branch touches
`.research/*.md` and `LOGBOOK.md` and nothing else — verified with
`git diff --stat $(git merge-base main HEAD) HEAD`. `go build ./...` and
`go vet ./...` in `tools/agents-infra`: both clean. No Go file is touched by
this branch, so "tests green" is trivially satisfied and the producer's
attached `go test ./... -count=1` exit-0 log is accepted rather than rerun.

---

## 1. The five rev1 findings, checked as fixes rather than as rewordings

### F1 — closed, and independently re-derived

I re-ran the enumeration myself rather than reading the corrected row:

```
$ grep -rnE '\.Env\s*=' --include="*.go" tools/agents-infra/ | grep -v _test.go
pi_shared_broker_darwin.go:384   command.Env    = scrubSharedRuntimeEnvironment(environ)
pi_shared_client_darwin.go:516   command.Env    = scrubSharedRuntimeEnvironment(environ)
pi_shared_client_darwin.go:693   piCmd.Env      = managedEnv
pi_launch_posix.go:223           runtimeCmd.Env = opts.Environ
pi_launch_posix.go:306           piCmd.Env      = managedEnv
pi_launch_posix.go:642           cmd.Env        = env
pi_platform_windows.go:140       cmd.Env        = opts.Environ
```

Seven child-process env compositions, exactly as §12 row 1 now states, all pi
launches. (The raw pattern also hits `attachments.go:91-92`, which sets an
env-*lookup* function field on an options struct, not a child environment —
correctly excluded.) §9.3's P-AI-1 carries the same correction. The row now
says something true and stronger than what it replaced.

### F4 — closed, thoroughly

The `file` default is carried into every place that needed it: `store_selector`
as a first-class handle field (§3.3) with the runtime × store → coordinates
table; H3 taking it as a re-derivation input; §5.1 step 6 naming who sets it
and refusing `auto` with `E_AUTH_CUSTODY_AMBIGUOUS`; §5.3 stating that
`auth.json` in an agents-infra-created root is still `vendor-opaque` because
custody follows the method not the filesystem; §5.4's file-store retire branch
with the *different* hazard named (no orphan, but `leave-vendor-state` ends no
server-side session); §6.2 re-keyed on the recorded store with
"`degraded:plaintext` means *unexplained* plaintext"; §7.1 explaining why the
version range still binds a branch with no hash; and Q12 surfacing the new
unknown rather than assuming keyring's posture in either direction.

§7.2's file branch carries the sharpest negative in the document: *the four
keyring assertions, run against a `file`-store profile, must fail — a pin that
passes on both branches is selecting neither.* That is a narrowing mutant, not
a delete mutant, and it is the correct answer to a vacuous-pin hazard.

### F5 — closed

The `enrolling` self-edge is drawn, and the prose distinguishes an `unknown`
observation *result* from the `unknown` *state* with the two different ways
forward. Correct.

### F3 — closed in §5.4, leaking in §3.6 (see finding 2)

§5.4 itself is right: `local` gains `unknown`; the classification table
requires a *successful* metadata-only observation for both positive values;
"silence is `unknown`, not `already-absent`"; `remove` refuses `unknown` and
`failed` with `E_AUTH_LOGOUT_UNCONFIRMED` and deletes nothing; the tombstone
is re-scoped to every `remove` that did not positively confirm invalidation.
§4.1 and §10's D1 row match. That is the fix asked for.

The brief said a single fix in one place is not evidence, and it was right —
finding 2.

### F2 — closed in five sections, contradicted in a sixth (see finding 1)

The mechanism change is real and I checked each surface. §3.2 makes the class
a total function; §3.3 **deletes** `custodian_class` from the handle with the
reason inline; §4.3 annotates the Custodian column as rendered; §6.1 splits
the hazard into an eliminated half and a CI-mitigated half and refuses to
average them; §10's C1 is restated. The §6.1 split is the honest version of a
claim that was previously false, and the recorded dependency — §5.3's
prohibition table is scoped "for `vendor-opaque`" and that scoping is safe
*only because of* C1 — is exactly the kind of thing that gets rediscovered
painfully later. Good.

Then finding 1.

---

## 2. Findings

### Finding 1 — §8.3 still says the method declares its custodian class (medium)

`.research/260831_extensible-auth-method-lifecycle.design.md:798`:

> because the method is a *value* of `--method`, **the custodian class is a
> property the method declares**, and the capability vocabulary already
> carries `unsupported` and `unknown`…

Against line 77 ("The custodian class is **not a declared property of
anything**") and line 864 (C1: "not a declared field"). §8 was never opened by
the rework — the diff hunks jump from §7.2 (`@@ -487,6 +694,24 @@`) straight to
§9 (`@@ -614,7 +839,7 @@`) — so this is rev1 text surviving verbatim inside the
one section that governs **how a new method enters the system**.

Two things are wrong, not one:

1. The sentence is the rev1 defect in its original words. The rev2 brief was
   explicit: *any* surface that sets or influences a custodian class, "including
   a prerequisite it names, a plugin contract, or a migration path", falsifies
   the claim. §8.3 is the adapter/registry extensibility contract.
2. §8.3's add-a-method recipe is "**one registry entry and one adapter**". §3.2
   makes a third edit mandatory — an arm in the mapping function — and says a
   method added without one "refuses at first use instead of defaulting". So
   §8.3 tells an implementer to do two of the three required things, and the
   design's own safety property is what turns that into a runtime refusal.

**This compounds with §4.1.** `describe()` → `MethodDescriptor` is documented
as "Static: **custodian class**, capability vocabulary, declared inputs…". The
adapter implements `describe()`. §3.2 says `describe()` refuses a method that
reaches the fallthrough arm, which implies the framework computes the class and
`describe()` renders it — but §4.1 does not say that, and §8.3 says the
opposite. An implementer reading the adapter contract and the extensibility
section together builds a `MethodDescriptor` whose author supplies the class,
which is the registry field C1 removed, wearing a different name.

**Failure scenario:** an implementer adds `coding-plan` per §8.3 — registry
entry plus adapter — and the adapter's `describe()` returns
`custodian_class: host-owned` because §8.3 told it the method declares one and
§4.1 gave it the field. Nothing in the specified system contradicts that: the
mapping function is never consulted, so its fallthrough never fires, and a
native subscription credential is evaluated on the `host-owned` branch where
§5.3's prohibitions do not apply by their own scoping sentence. That is exactly
the chain §6.1 says C1 eliminates.

**Fix:** rewrite §8.3's clause to "the custodian class is a total function of
the method (3.2), so a new method changes no verb and no record field"; state
the add-a-method edit set as three items including the mapping arm; and make
§4.1 say `describe()` *renders* the class from the 3.2 function rather than
listing it as descriptor content the adapter supplies.

### Finding 2 — §3.6's state machine draws the destructive edge ungated (medium, destructive consequence)

`…design.md:248`:

```
active|degraded|unknown ──logout──▶ retired-local ──remove──▶ removed
```

Unconditional, from `unknown`, to a state named `retired-local`, and onward to
`removed`. §5.4 says the opposite: a `logout` resolving `local: unknown` or
`failed` **refuses** `remove`, deletes nothing, and leaves the profile exactly
as it was.

The producer was inside §3.6 for F5 — the hunk is `@@ -205,7 +276,7 @@` and this
line is its context — and edited the `enrolling` edge one line above while
leaving the destructive edge untouched. This is precisely the leak the brief
predicted: the invariant held in §5.4's table and leaked in §3.6's diagram.

It is worse than a diagram-prose mismatch for two reasons. `retired-local` is a
**positive assertion of local retirement**, and the machine produces it from
`unknown` — the same forgery shape as `already-absent` from a failed
observation, which is the finding that rejected rev1. And `retired-local`
appears exactly once in all 908 lines: it is not in §3.5's `custody_state`
field, not in §5.4, not in §6.2's detection table, not in §10. An implementer
building the state machine from §3.6 has an undefined state with an
unconditional edge to deletion and no other section to correct them.

**Failure scenario:** an implementer builds `remove` from §3.6 rather than from
§5.4's prose. `logout` on a profile whose Keychain observation was
permission-denied returns `local: unknown`; the machine says the profile is now
`retired-local`; `remove` follows the drawn edge and deletes the state root. The
Keychain item survives, its service name derived from a path that no longer
exists. §5.4's `E_AUTH_LOGOUT_UNCONFIRMED` never fires because the
implementation followed the diagram.

**Fix:** gate the edge in the diagram —
`active|degraded|unknown ──logout(local: invalidated|already-absent)──▶ retired-local`,
with `logout(local: unknown|failed)` a self-edge that refuses — define
`retired-local` in §3.5's `custody_state` vocabulary, and add a §3.6 bullet
stating that `remove` is reachable only from a positively-confirmed logout.

### Finding 3 — the "what is checked rather than structural" list is not complete (medium)

§3.2's admission list has two entries: the mapping function's totality and its
exhaustiveness test. The brief asked whether that list is complete or merely
convenient. It is missing the one that matters most.

**The class function's only input is stored on an editable on-disk surface, and
§3.3 deletes a field for exactly that reason.** §3.3's inline comment:

> Storing it would put the pairing back on an input surface — an on-disk record
> an operator or a migration could edit — which is precisely what C1 removes.

§3.5 puts `method` in that same JSON record. `custodian_class = f(method)`.
Deleting the derived value and keeping its sole input on the same file does not
remove the input surface; it adds one level of indirection. §3.5 marks `method`
"Immutable after enrol", but that is an assertion with no stated enforcement —
no signature, no checksum, no re-derivation from enrol-time evidence — and it is
the same kind of assertion §3.3 just refused to trust.

§6.1's enumeration is "No config file, flag, registry column, project table,
remote-config key, plugin or environment variable". The profile record is in
none of those categories, and it is the one place the input actually lives.

**Failure scenario, and nothing in the design catches it:** a migration (§3.3's
own named concern) rewrites `method` from `subscription-oauth` to `api-key` in
`…/auth/profiles/anthropic/work.json`. The computed class flips to `host-owned`.
Every other gate passes: `state_root` and `store_selector` are unchanged so H3
finds no drift; the version gate and pin are unaffected; §6.2 sees a credential
present at `observed_coords` with no plaintext artifact and reports `active`.
The profile launches, agents-infra now believes it owns rotation of a credential
in the vendor's own store, and §5.3's prohibitions — scoped "For
`vendor-opaque`" — no longer apply to it. That is the fail-open hazard reached
without touching a config file, a flag, or an environment variable.

**Fix:** either add the profile record to §3.2's checked list and state what
protects `method` (and note that H3, the pin and §6.2 all pass through a flipped
method, so nothing currently does), or make the record's `method` non-
authoritative — bind the class to enrol-time evidence the record cannot
contradict. Do not delete finding 3 by asserting immutability harder; §3.3
already established that the document does not accept that argument.

---

## 3. Growth: warranted

682 → 908, twenty hunks, every one inside a section named by a finding. No new
top-level sections, no restructuring, §9 untouched but for P-AI-1's corrected
sentence. §3.2's "what is checked" block and §6.1's split are prose rather than
mechanism, but they are the prose rev1's verdict explicitly demanded, and §6.1
is now shorter in claim and longer in honesty, which is the right trade. The
+226 buys five fixes, one type-shape change, one new open question and two new
negative-test families. I would not cut any of it.

The document remains usable under pressure: §10 is thirteen checkable rows, §11
carries what-settles-it and a cost per question, and §7.2's negatives are
specific enough to write tests from. That property is intact and worth
protecting through the next rework — all three findings above are edits of
tens of lines, not sections.

## 4. AC and DoD

All seven ACs are substantively addressed (§3.1; §4.1; §3.4 + S1; §5.4's two
independent fields; §5.4's gated `remove`; §8.3 — see finding 1, which is a
correctness defect in its *justification*, not an extensibility gap; §8.2 + A1).
DoD items on the codex custody model, the plaintext-fallback hazard, the
version-gated `CLAUDE_CONFIG_DIR` dependency with a refusing gate, the
`skill-agents-management` prerequisites, and the open-questions set are all met.

Not accepted on findings 1, 2 and 3.

## 5. Safety boundary

Held, by the producer and by this review. No credential, token, cookie or
Keychain secret is read, printed, exported or persisted anywhere in the
artifact or the validation log. No `security` invocation, no login, logout,
revoke or rotation. My checks were source reads, greps, `go build`, `go vet`
and git reads on this worktree.

## 6. What rev3 needs

1. §8.3 — restate the extensibility justification in terms of the 3.2 function;
   make the add-a-method edit set three items; align §4.1's `describe()` row.
2. §3.6 — gate the `logout → retired-local → remove` edge on
   `local: invalidated|already-absent`, define `retired-local` in §3.5, and
   state the refusal in a bullet.
3. §3.2 — admit the profile record as a checked surface and say what protects
   `method`, or remove it as the authoritative input.

Nothing else. F1, F4 and F5 are done, §5.4 is done, §9 and §12 are verified and
should not be reworked.

---

## 7. Logbook entry for the next producer to commit

The reviewer role is read-only on the repository, so this is recorded here
rather than written into `LOGBOOK.md`. The rev3 producer should commit it
alongside the three fixes.

### HHmm — A Fixed Invariant Leaks In The Section The Rework Never Opened
- FINDING (reviewer, CR rev2 of `TASK-260720-bk6owf`): both rev1 defects were fixed correctly *at the site they were found* and both survived elsewhere. The rework's own hunk map is the tell — 20 hunks, every one inside a section a finding named, and §8 was never opened.
- SHAPE 1 — the corrected claim survives verbatim where nobody looked. §3.2/§3.3/§4.3/§6.1/§10 all now say custodian class is a total function with no declared field. `design.md:798` (§8.3, extensibility) still says "the custodian class is a property the method declares" — rev1 text, in the one section that governs how a *new* method enters the system. Its add-a-method recipe is "one registry entry and one adapter", omitting the mapping arm §3.2 makes mandatory. §4.1 compounds it by listing custodian class as `describe()` descriptor content, which the adapter implements.
- SHAPE 2 — the invariant leaks one line from its own fix. §5.4 now refuses `remove` on `local: unknown|failed`. `design.md:248` (§3.6) still draws `active|degraded|unknown --logout--> retired-local --remove--> removed`, unconditional, and the producer edited that diagram one line above for F5. `retired-local` is a positive assertion of local retirement produced from `unknown` — the same forgery shape as `already-absent` from a failed read — and it appears exactly once in 908 lines, in no vocabulary and no other section.
- SHAPE 3 — deleting a derived field does not remove its input surface. §3.3 dropped stored `custodian_class` because "an on-disk record an operator or a migration could edit" is an input surface. `custodian_class = f(method)` and `method` is a field of that same on-disk record (§3.5). A migration flipping `method` to `api-key` passes H3 (coords unchanged), passes the pin, and reads `active` in §6.2 — landing a native credential on the `host-owned` branch where §5.3's prohibitions do not apply by their own scoping.
- ROOT CAUSE (all three): a claim of the form "X is structural, not checked" is a claim about the *whole document*, but it gets verified against the paragraph that makes it. Fixing the paragraph does not make the claim true.
- REUSABLE: when a rework's justification is "we made the bad state inexpressible", grep the whole artifact for the verb — `declares`, `set`, `configured` — near the concept, not for the sentence that was rewritten. And read the diff's hunk map: sections with zero hunks are where the old claim still lives.
- SCOPE: `.research/260831_extensible-auth-method-lifecycle.design.md` (rev2, 682->908). STATUS: changes requested -> `analysis`.
