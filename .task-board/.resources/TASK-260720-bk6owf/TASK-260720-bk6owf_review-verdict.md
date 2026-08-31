# TASK-260720-bk6owf — review verdict

Reviewer run: `TASK_BOARD_RUN_ID` (reviewer archetype)
Change Request: `CR-TASK-260720-bk6owf-1` revision 1
Artifact under review: `.research/260831_extensible-auth-method-lifecycle.design.md` (682 lines)
Date: 2026-08-31

## Verdict: CHANGES REQUESTED → `analysis`

Four defects. One of them (F3) is the design's own `unknown`-never-collapses
invariant failing at the single point where the consequence is irreversible.
The rework is bounded — four targeted edits, no restructuring. Everything else
in this document is strong and independently checked out.

---

## 0. On the empty repository delta

The Change Request reports `repository_delta=empty`. **This is a snapshotting
artifact, not an absent deliverable.** The CR's base OID `fba9ee4` *is* the
producer's own commit:

```
fba9ee4 TASK-260720-bk6owf: design extensible auth-method and credential lifecycle
 .research/260831_extensible-auth-method-lifecycle.design.md | 682 +++++++++
 LOGBOOK.md                                                  |  14 +
```

The producer committed before the candidate tree was snapshotted, so base tree
== candidate tree. The repository change exists, is on
`task-board/story/STORY-260831-yr0x81`, and is 682 lines of design plus a
logbook entry. Reviewed against that commit, not against the empty patch.

Design-only with no production-code change is the *correct* shape for this leaf:
the task scope says "Architecture model and CLI semantics only", the DoD says
open questions ready for `TASK-260720-3gcfd1`, and §9 establishes that none of
it is implementable today because three named repositories lack the contracts.
Writing code here would have been the failure.

---

## 1. Independent re-derivation of §12

§12 claims seven facts "re-derived here, because every one of these is
load-bearing for a prerequisite in section 9". I re-ran all seven against the
live checkouts rather than trusting the table.

| # | Claim | My check | Result |
| --- | --- | --- | --- |
| 1 | `runClaude`/`runCodex` never set `cmd.Env` | Read `tools/agents-infra/main.go:417-472` directly | **Conclusion holds.** Both build `exec.Command`, set Stdin/Stdout/Stderr, and never assign `.Env`. The *enumeration* is wrong — see F1. |
| 2 | claude `ChildEnv` injects nothing runtime-specific | `pkg/agentic/systems/claude/env.go:112` | Holds. `childEnv` = `agentic.WithRunContext(filterRuntimeEnv(parent), req)`; `runtimeEnvKeys = []string{sessionMarkerEnv}` — one key. Parent's `CLAUDE_CONFIG_DIR` passes straight through. |
| 3 | `planCommand` already carries `Plan.Env` | `internal/spawn/spawn.go:938-940` | Holds. `cmd.Env = plan.Env`. P-PM-3 is correctly a no-op. |
| 4 | `Plan.Home` resolved and unread | `pkg/agentic/plan.go:176-189`, then grepped every `.Home` reader | Holds, and is *stronger* than stated. `req.Home` is a pure passthrough (`vendorplugin/plugin.go:75` → `LaunchRequest.Home` → `plan.go:176`), consumed by nothing. `vendorplugin/spawn.go:80` even documents Home as "load-bearing beyond the launch: on-disk [limit state]" while `providerlimits` reads the parent env instead. The comment and the code disagree; the design's P-AM-2/P-AM-3 pair is right. |
| 5 | `providerlimits` reads the parent environment | `pkg/providerlimits/identity.go:115-118` | Holds. `os.Getenv(capabilities.HomeEnvVar)`. |
| 6 | `home_env` trimmed, not validated | `pkg/remoteconfig/spawn_runtimes.go:39,99` | Holds. `declared.HomeEnv = strings.TrimSpace(declared.HomeEnv)` on line 99, sandwiched between `ParseBrokerID` (typed refusal) and the builtin-contradiction check. Siblings get parsing and refusal; this gets whitespace removal. Only other non-test reference is the read-only copy at `cmd/auth.go:578-579`, as claimed. |
| 7 | qwen declares no home variable | `pkg/agentic/systems/qwen/qwen.go:121` | Holds. `HomeEnvVar: ""`, `DefaultHome: ""`. Same for muse, gemini, agy, pi. |

Six of seven clean. §9's prerequisites are real, correctly located, and each
carries a negative test that would fail against the current broken code. That
part of the document is exactly right and I am not asking for changes to it.

---

## 2. Findings

### F1 — §12 row 1 states a false enumeration (low severity, but it is in the fact table)

**The claim:** "the only `cmd.Env` assignments in the whole module are
`pi_launch_posix.go:642` and `pi_platform_windows.go:140`."

**Reproduced the producer's grep:**
```
$ grep -rn "cmd.Env" --include="*.go" tools/agents-infra/ | grep -v _test.go
tools/agents-infra/internal/infra/pi_platform_windows.go:140:   cmd.Env = opts.Environ
tools/agents-infra/internal/infra/pi_launch_posix.go:642:       cmd.Env = env
```
Two hits, as stated. **But there are seven child-environment assignments in
that module:**
```
pi_shared_broker_darwin.go:384:  command.Env    = scrubSharedRuntimeEnvironment(environ)
pi_shared_client_darwin.go:516:  command.Env    = scrubSharedRuntimeEnvironment(environ)
pi_shared_client_darwin.go:693:  piCmd.Env      = managedEnv
pi_launch_posix.go:223:          runtimeCmd.Env = opts.Environ
pi_launch_posix.go:306:          piCmd.Env      = managedEnv
pi_launch_posix.go:642:          cmd.Env        = env
pi_platform_windows.go:140:      cmd.Env        = opts.Environ
```
The pattern `cmd.Env` is case-sensitive and matches neither `command.Env` nor
`piCmd.Env`/`runtimeCmd.Env`.

**Why it is only low severity:** I chased the five missed sites. All are pi
runtime launches, and `pi` declares `HomeEnvVar: ""` (`systems/pi/pi.go:87`),
so it is not a home-bearing runtime and Invariant L1 does not reach them. **No
design conclusion changes and P-AI-1 has no scope gap.** I am reporting the
defect, not inflating it.

**Why it still needs fixing:** this row sits in a table introduced with "Not
taken from the research on trust — re-derived here", and `TASK-260720-3gcfd1`
will consume it as established fact. A narrow literal grep reported as an
exhaustive module-wide enumeration is the *proxy-signal-as-fact* shape. Restate
as "neither `runClaude` nor `runCodex` assigns `cmd.Env` (read directly at
`main.go:417-472`); the module's seven env-composing sites are all pi launches,
and pi declares no home variable" — which is both true and a stronger argument.

---

### F2 — §6.1's "structural" is an overclaim; the mechanism is three checked gates (medium)

**The claim (§6.1):** "The enforcement mechanism that would trigger the fallback
does not exist in the design, rather than existing and being discouraged."
Repeated in the commit message: "not expressible".

**What the document actually specifies:**

1. **C1 (§3.2)** says the pairing "cannot be declared anything else" and "the
   model has no way to express" it — and then, one sentence later: "A method
   registry entry that pairs a native-OAuth method with `host-owned` is a config
   error **refused at load**." If it is refused at load, it is expressible. The
   custodian class is a declared column of a registry entry (§4.3 renders it as
   one), and correctness is a load-time validation. That is a *checked*
   prohibition, not an unsayable state.
2. **§5.3** is explicitly "stated as a prohibition list because each item is a
   plausible good-intention mistake". A list of things not to do is
   discouragement by construction.
3. **§6.4** upgrades that to greppable-absent plus negative tests. Real
   enforcement — but it is CI enforcement, not structure.

**Why the distinction is load-bearing here and not pedantry:** §6.1 as written
tells a reader the hazard needs no test, while §6.4 requires tests for exactly
this. The two sections disagree about what protects the most damaging outcome
the design can produce. And the scoping leaks: §5.3's prohibition table is
introduced "For `vendor-opaque`". A registry entry that got past the load check
with a native method declared `host-owned` would land on the host-owned branch
— "agents-infra rotates on its own schedule and the vendor reads what it is
given" — where §5.3's prohibitions, by their own scoping sentence, do not apply.
Only §6.4's module-wide grep still stands in the way.

**What would make the claim true as stated:** make the custodian class a
property of the method *type* in code — a total function from the method enum,
with no registry field to set — so there is no load check because there is no
input. Then §6.1's sentence is accurate. Alternatively keep the registry field
and rewrite §6.1 honestly: "closed by three independent checked gates (load-time
registry validation, module-wide greppable prohibition, negative tests)", and
re-scope §5.3's table to all custodian classes rather than to `vendor-opaque`.
Either is fine. The current text claims the first and specifies the second.

**Credit where due:** I traced every lifecycle operation for a path to a
permission or ownership change, as the brief asked. There is exactly one —
§5.1 step 5, "Create it at mode 0700 owned by the operator" — and the design
*names it and defends it correctly*: 0700 on a directory owned by the same user
the CLI runs as is not a denial to the CLI and does not engage the fail-open
path. That is the right analysis and it is not a finding. Enrol, use, refresh,
retire and the `degraded:plaintext` recovery path reach nothing else.

---

### F3 — D1 leaks in `logout`, and the leak gates the only destructive verb (medium, destructive consequence)

This is the finding I would not accept the document with.

**§5.4 specifies the logout result as:**
```
local        : invalidated | already-absent | failed
remote_revoke: revoked | not-attempted | unsupported | unknown
```
`remote_revoke` carries `unknown`. **`local` does not.** And `remove` gates
destruction on it: "`local-logout` runs `logout` first and requires
`local: invalidated | already-absent` before deleting anything."

`already-absent` is a positive assertion of absence. The document has no rule
for classifying an observation it could not make. §4.1's `status` contract
promises "a per-field `unknown` where the observation could not be made", and
§6.2/H2/D1 are rigorous about it — and then §5.4 drops it.

**Concrete failure:** operator runs
`agents-infra auth remove --provider anthropic --alias work --logout-policy local-logout --confirm`.
The vendor logout runs inside the state root and exits 0 having printed nothing
— because the login keychain is locked, or the item query returned a permission
error the vendor swallowed, or the vendor simply says nothing on a no-op. The
natural implementation of the stated vocabulary reads exit 0 + nothing observed
as `already-absent`, which is in the permitted set, so `remove` proceeds and
**deletes the state root**. The Keychain item still exists. Its service name is
`Claude Code-credentials-<sha256(NFC(configDir)).hex[:8]>` and the derivation
input has just been destroyed, so nothing can name it again.

That is precisely the orphaned-item hazard §5.4 itself raises three paragraphs
later — and the auditability mechanism the design built for it, the tombstone,
is written only on the **other** branch (`leave-vendor-state`). Under
`local-logout` the orphan is silent.

**Fix:** add `unknown` to the `local` vocabulary; exclude it from the set that
permits deletion; state the classification rule (what distinguishes
`invalidated` from `already-absent` from `unknown` in the vendor's observable
output, and that silence is `unknown`, not `already-absent`); and write the
tombstone on any `remove` that did not positively confirm local invalidation,
not only on `leave-vendor-state`.

---

### F4 — the codex branch is specified only for the keyring store, while §2 states the default is `file` (medium)

The brief asked me to confirm the lifecycle does not quietly assume the Claude
shape for Codex. It mostly does not — §2 fact 4, §4.3's registry and §7.1's
separate range are all real. But the divergence is *stated* in §2 and then not
carried into the three sections that need it:

- **§3.3 `observed_coords`** lists `keychain_service`, `keychain_account`,
  `file_path`, `fallback_path` as one flat struct. For codex, which of these is
  populated depends on `cli_auth_credentials_store`, whose packaged default is
  `"file"` (§2 fact 4) — a config value living *inside* the state root that can
  change after enrol without the path changing. The document never names it as
  an input to the codex derivation, so an implementer would not know to include
  it, and H3's re-derive-and-compare would silently compare the wrong thing.
- **§7.2's codex pin** verifies `cli|<sha256(canonical home).hex[:16]>` — the
  **keyring** account derivation. Under the packaged default that derivation is
  not in use at all. So the pin that gates every codex launch targets a store
  the default configuration does not select.
- **§6.2's detection table** is written generically but keys `degraded:plaintext`
  on `<state_root>/.credentials.json`, which is Claude's fallback path. For a
  codex profile on the default `file` store the credential *is* plaintext at
  `<state_root>/auth.json`, and the correct answer is `active` — that is the
  vendor's documented default, not a fail-open. The document never says so, and
  a reader can reasonably reach the opposite conclusion from a table that
  declares plaintext custody degraded.

**Fix:** state which store an enrolled codex profile uses and who sets it; make
`cli_auth_credentials_store` an explicit input to the codex `observed_coords`
derivation and to H3's comparison; give §7.2 a `file`-store branch (where P1's
state-root bijection already provides the disjointness the hash provides for
keyring, so the pin is simpler, not absent); and say explicitly in §6.2 that
codex-on-`file` is `active` and not `degraded:plaintext`.

---

### F5 — §3.6 state machine omits a transition §5.1 produces (minor)

§5.1 step 8: "Observation failed → `unknown`, and the profile stays `enrolling`."
The §3.6 diagram shows `enrolling` going only to `active` or back to `declared`,
and reaches `unknown` only from `active|degraded`. Two readings of what a failed
enrol-time observation leaves behind. D1 is not violated either way — neither
`enrolling` nor `unknown` is `absent` or `active` — so this is a diagram/prose
consistency fix, not a safety defect.

---

## 3. D1 trace (the brief asked for every transition and every consumer)

| Site | Handles `unknown` correctly? |
| --- | --- |
| §3.3 H2 | Yes — "failed, partial or permission-denied observation is `unknown`, never `absent`". |
| §3.6 machine | Yes — `unknown` is its own state; see F5 for the `enrolling` edge. |
| §4.1 `status` | Yes — per-field `unknown` where observation failed. |
| §4.2 capability vocabulary | Yes, and explicitly: never rendered as `supported`/`unsupported`, never defaulted, never inferred from an adjacent success. |
| §5.1 step 2 (version) | Yes — unreadable → `unknown` → refuse. |
| §5.1 step 8 (enrol observe) | Yes semantically; see F5. |
| §5.2 step 4 (launch) | Yes — "anything but `active` refuses". |
| §5.4 `remote_revoke` | Yes — anthropic reports `unknown`, and explicitly does not print a sentence implying the session ended. |
| **§5.4 `local`** | **No — F3.** |
| §6.2 table | Yes — four rows, failure and absence distinct. |
| §7.1 version gate | Yes — unreadable is not in-range. |

One leak out of eleven, at the destructive verb.

## 4. Version gate — confirmed as the brief specified

§7.1 claude range is **2.1.234 – 2.1.248**, the full span the custody reviewer's
F1 established, not 2.1.248 alone. The gate **refuses** (`E_AUTH_VERSION_UNPINNED`,
non-zero exit, no child process started), with the cost argument stated
correctly: refusing costs one error message, warning enrols into a namespace
that can be orphaned silently. Unreadable version → refuse, not in-range. Codex
is honestly held at 0.150.1-only with Q4 naming what would widen it, and §7.4
requires a passing pin run with every negative variant before any widening.

§7.2's pin tests carry the narrowing negatives, not just delete-mutants: 7-or-9
hex must fail for claude, 15-or-17 for codex, plus different-root and
NFC-vs-NFD discrimination. That is the right shape and it is the part of this
document I would hold up as the example.

§7.3's `CLAUDE_SECURESTORAGE_CONFIG_DIR` refusal is correct and Q11 keeps it
honest by asking whether some legitimate setup sets it.

## 5. AC and DoD status

All seven ACs are substantively addressed (§3.1; §4.1's eight operations; §3.4;
§5.4's two independent fields; §5.4's gated `remove`; §8.3; §8.2 + A1).
§10 is checkable rather than aspirational — each row maps to a grep or a test.
§11's eleven questions each carry "what settles it" and a cost, and correctly
mark Q9 blocked on Q1 rather than guessing.

Safety boundary held: no credential, token or Keychain secret is printed,
exported or persisted anywhere in the artifact; no live session was touched. I
verified the same for this review — my checks were source reads only.

Not accepted on F2, F3, F4 (F1 and F5 to be swept while in there).

## 6. What the next `analysis` pass needs

1. §12 row 1 — restate the enumeration truthfully (F1).
2. §6.1 + §3.2 — either make custodian class a code-level property of the method
   type so the state is genuinely unsayable, or rewrite §6.1 as three checked
   gates and re-scope §5.3's prohibition table to all custodian classes (F2).
3. §5.4 — add `unknown` to `local`, exclude it from the deletion-permitting set,
   state the classification rule, and write the tombstone whenever local
   invalidation was not positively confirmed (F3).
4. §3.3 / §6.2 / §7.2 — carry `cli_auth_credentials_store` into the codex coords
   derivation, add the `file`-store pin branch, and state that codex-on-`file`
   is `active` (F4).
5. §3.6 — reconcile the `enrolling` failure edge with §5.1 step 8 (F5).

Everything else stands. §9 in particular is verified and should not be reworked.
