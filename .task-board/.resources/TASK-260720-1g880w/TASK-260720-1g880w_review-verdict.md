# TASK-260720-1g880w — review verdict: ACCEPTED

Reviewer run: `RUN-260831-a8bcd4`. Change Request `CR-TASK-260720-1g880w-1` rev 1.
Reviewed at branch head `4f92abf3`, tree `ac2a815f`.

## Verdict

**Accepted.** Every load-bearing claim in
`TASK-260720-1g880w_keychain-custody-and-refresh-semantics.md` was re-derived
independently against current source or reproduced on this machine. Nothing was
taken on the producer's word except one attestation that artifacts cannot
falsify, named explicitly below.

## Why an empty `repository_delta` is the right outcome here

`repository_delta=empty` is a **snapshot-timing artifact, not an absent
delivery.** The producer committed its work *before* the CR was cut, and the CR
base OID `4f92abf3` **is** the producer's own second commit. Verified:

| Check | Result |
| --- | --- |
| `git log --oneline main..HEAD` | `1b379e2` research doc (376 lines), `4f92abf` logbook (17 lines) |
| `git rev-parse HEAD^{tree}` | `ac2a815f…` — identical to the CR candidate tree |
| Branch tip | `4f92abf3` == CR base OID |

So the leaf did produce 393 lines of committed repository change; the diff
window simply opens after them. The footprint is also the *correct* one for this
leaf: the task is feasibility research whose scope says "documentation, provider
source/contracts, empty synthetic profiles" and whose deliverable is a matrix
plus proof gates for `TASK-260720-3gcfd1`. A research leaf that had modified
production code would have been the finding.

## Claim-by-claim verification

Every row below was re-established by the reviewer, not read off the report.

| # | Claim | How verified | Result |
| --- | --- | --- | --- |
| 1 | `CLAUDE_CONFIG_DIR` namespaces the macOS Keychain **service** name | Independent run: fresh synthetic dir, hash predicted *before* execution, shim-observed | Predicted `47c640ee`, observed `Claude Code-credentials-47c640ee` — **exact** |
| 2 | Suffix is `sha256(NFC(dir)).hex[:8]` | Recomputed the producer's own dir digest independently | `31c6920d` — exact match to the report and its shim log |
| 3 | Suffix tracks the directory | Second dir in the same run | Different dir → different suffix, as the formula predicts |
| 4 | The **NFC** in the formula is real, not decorative | Ran a **NFD-composed** non-ASCII config dir | Observed `450cba77` = NFC digest, **not** `047ac4b0` = raw/NFD digest. Formula correct for non-ASCII paths; proof gate 2's pin is correctly specified |
| 5 | `ok()` / `ME()` source quotation | `strings` on the 2.1.248 artifact | Byte-for-byte as quoted, including minified identifiers `e/t/r/c`, `ge()`, `a("sha256")` |
| 6 | Live default-namespace item untouched | `cdat`/`mdat` before **and after** the reviewer's own experiment | `20260819081931Z` / `20260831002314Z` unchanged. Service exactly `Claude Code-credentials`, no suffix. Account `alexis`, length 6 — corroborates the shim's `<redacted:len=6>` |
| 7 | Claude fails **open** to plaintext | Composite `T(e,t)` in 2.1.248 | On non-transient primary failure: writes plaintext at mode `384` (0o600) with `Warning: Storing credentials in plaintext.`, then `if(s!==null) await e.delete()` — **deletes the Keychain item**. Confirmed, and the report is if anything *conservative*: `transient` is set only on `u.timedOut`, so an ACL/permission denial — the exact enforcement mechanism external custody would use — is non-transient and **does** trigger the fallback |
| 8 | Refresh is an in-place `-U` overwrite, argv hazard above 4032 B | Same chunk | `["add-generic-password","-U","-a",a,"-s",r,"-X",s]` on the argv branch, guarded by `U=4032`, with a `using argv` warn. Exact |
| 9 | Cross-process `.storage-write` lock | Same chunk | `retries:10, minTimeout:100, maxTimeout:1000, stale:15000`, `realpath:!1`. Exact, including the numbers |
| 10 | 30 s read cache, 1 s failure backoff, stale-cache-on-failure | Constant resolution | `V6t=30000`, `lyn=1000`, and the literal `[keychain] read failed; serving stale cache`. Exact |
| 11 | Codex default custody is **file** | `strings` on the vendored 0.150.1 native binary | `cli_auth_credentials_store = "file"` present in the fixed-defaults block |
| 12 | Codex service `Codex Auth`, account prefix `cli\|` | Same binary | Both literals present, in `login/src/auth/storage.rs` |
| 13 | `home_env` layer 1 — normalized, not validated | `skill-project-management/pkg/remoteconfig/spawn_runtimes.go` (source repo, not just vendor) | Field at `:39`, `strings.TrimSpace` at `:99`, and nothing else — while `agentic_system` and `broker` both get `Parse*` + closed-vocabulary refusal with typed errors. **Confirmed** |
| 14 | `home_env` has exactly four non-test references | Repo-wide grep across both skill repos | Field, doc comment, trim, and the read-only `auth.go:578-579` copy into `project_config()`. **Confirmed** |
| 15 | `home_env` layer 2 — `Plan.Home` discarded | `plan.go:176-190`, `internal/spawn/spawn.go:938-945` | `planCommand` consumes Binary, Argv, Env, Stdin, WorkDir. `grep '\.Home\b'` over `internal/spawn` non-test returns **zero** hits — even stronger than "not among them" |
| 16 | No plugin injects `HomeEnvVar` into the child env | grep `CODEX_HOME`/`CLAUDE_CONFIG_DIR` non-test in skill-agents-management | Only comments and the two `Capabilities` declarations. `Home` flows `SpawnRequest → LaunchRequest → Plan` as inert data (`vendorplugin/plugin.go:75` is a lossless projection, not an injection). **Confirmed** |
| 17 | `home_env` layer 3 — limit plane reads the **parent** env | `providerlimits/identity.go:112-123` | `os.Getenv(capabilities.HomeEnvVar)`. **Confirmed**, including the report's correction that the plane is not blind but wired to the wrong source |
| 18 | `codex_manager.go:314` is a different mechanism | Read | `managedCodexClientEnvironment` sets `CODEX_HOME` for the session-manager context fork. Correctly excluded from the multi-account claim |

### Attempted refutations that failed

The following were pursued as attacks on the report and did **not** land:

- **"`authority_schema.go:157` says a specialized validator applies to
  `spawn.runtimes.*.home_env`, contradicting layer 1."** No: that string is the
  `NullBehavior` positional argument, shared verbatim with sibling leaves. It
  makes no claim about name-syntax validation. Report stands.
- **"The minified `ok()` only normalizes NFC on the `CLAUDE_SECURESTORAGE_CONFIG_DIR`
  branch, so `sha256(NFC(configDir))` may be wrong for the `CLAUDE_CONFIG_DIR`
  branch on non-ASCII paths."** Refuted empirically by row 4 — the NFD dir hashed
  as NFC. Report's formula stands as written.
- **"Fail-open may not fire for a permission denial."** Refuted — `transient` is
  set only on timeout, so denials take the fallback path. Report understates the
  hazard rather than overstating it.

## Findings

### F1 — the version boundary was asked for and not established (accepted, closed here)

The spawn brief called this out as mattering more than the rest: *"the claim is
specific to 2.1.248. Establish whether it holds only there, and say so."* The
report does not do that empirically. Its nearest check — inspecting the constant
across the installed versions under `~/.local/share/claude/versions/` — is filed
in the unknowns table as **"Free. Not run here."**

The reviewer ran it. Across the four other artifacts on disk:

| Version | `CLAUDE_SECURESTORAGE_CONFIG_DIR` | Namespacing construction |
| --- | --- | --- |
| 2.1.234 | present | `…normalize("NFC")…sha256…substring(0,8)` — same shape |
| 2.1.236 | present | same |
| 2.1.247 | present | same |
| 2.1.248 | present | same |

**So the behaviour is not 2.1.248-specific: it holds unchanged across at least
2.1.234 → 2.1.248.**

This does not change the verdict, for two reasons. First, it makes the report's
framing *conservative* rather than wrong — it under-claims stability while
recommending exactly the right mitigation. Second, and more importantly, the
brief's stated *purpose* for the ask ("the decision must know whether it is
relying on [an undocumented implementation detail]") **is** discharged, squarely
and in the right places: hard blocker 5 names both derivations as current-source
with no compatibility promise and warns that a minor bump can orphan every
enrolled profile silently; proof gate 2 requires a pin test; proof gate 3
requires a version gate that refuses the launch outside the pinned range. The
decision is not being handed an unmarked dependency.

Recording the breadth here rather than routing a rework cycle for one `strings`
invocation. `TASK-260720-3gcfd1` should carry the 2.1.234–2.1.248 span into the
version gate's initial range instead of pinning it at 2.1.248 alone.

### F2 — one safety claim rests on attestation, not artifact (reported as unknown)

Reported rather than inferred, per the evidence rules.

**Verifiable and verified:** no credential-shaped material exists in any
artifact. The research doc's only `sk-`-shaped string is the fabricated
`sk-SYNTHETIC-NOT-A-REAL-KEY-a`; it contains no hex blob ≥ 20 chars; the shim
log carries only `<redacted:len=6>` and service names; the added logbook lines
contain only `c2e40b5f045197b0` / `c8c1b4ad6d576395` / `31c6920d`, which are
*path*-derived namespace digests, not secrets; the implementer spawn log has
zero secret-shaped matches and records no `-w`; the board resource is
byte-identical (`be470bcd…`) to the committed doc. The shim source execs through
and logs only `-a`/`-s`.

**Verifiable and verified:** nothing live was mutated. Live item `cdat`/`mdat`
unchanged; `~/.codex/auth.json` unchanged at `2026-08-27T10:52:45Z` / 4003 B; no
`Codex Auth` item remains, independently confirming the producer's cleanup
(ledger 9); a full keychain sweep finds exactly one `Claude Code*` service and
no hash-suffixed residue from either the producer's or the reviewer's synthetic
profiles.

**Not verifiable from artifacts:** the claim that no `security … -w` was *ever*
issued against the live item. A `-w` read leaves no trace in `cdat`/`mdat`, and
the attached spawn log is an 8.5 KB summary rather than a command transcript. The
strictly stronger and decision-relevant property — that no credential was
printed, exported or persisted anywhere — **is** established by the scans above.
The narrower "no `-w` ever executed" is an unfalsifiable attestation and is
recorded as such rather than reported as confirmed.

## Reviewer's own experiment

Held to the report's own proof gate 6 (reversible, cleanup verified). Two
synthetic empty config dirs, the producer's shim reused verbatim, `claude -p`
run against each; both reported `Not logged in`. No Keychain item was created,
none was read with `-w`, and the live item's `cdat`/`mdat` were re-read after and
were unchanged. Synthetic dirs removed; log retained at
`.temp/TASK-260720-1g880w-review/reviewer-independent-falsification.log`.

## Definition of Done

| Item | Status |
| --- | --- |
| Per-provider feasibility matrix (custody, refresh, multi-account, breakage) | Met — verified rows 7-12 |
| Hard blockers separated from unknowns, each unknown paired with its experiment | Met — 6 blockers, 7 unknowns each with experiment + cost/risk |
| `home_env` gap confirmed or refuted against current source | Met — all three layers confirmed at exact cited lines, with two corrections that themselves check out |
| No credential/token/cookie/keychain value printed, exported or persisted | Met — see F2 |
| No logout/revoke/rotation/re-auth against a live session | Met — live item and `auth.json` integrity re-verified by the reviewer |
| Findings attached for `TASK-260720-3gcfd1` | Met — board resource byte-identical to the committed doc |
| Implementation matches AC | Met — matrix, blockers, unknowns, custody-model comparison (A/B/C), and 7 proof gates all present |
| Gates attacked, not read | Met — see "Attempted refutations that failed"; three independent attacks mounted, all failed |

Tests: none applicable. This leaf is feasibility research and ships no code; the
`home_env` remediation it scopes is explicitly deferred to `TASK-260720-3gcfd1`,
with the report correctly requiring a *negative* test per layer ("fails when the
home is dropped") rather than a positive one, since all three current failures
are silent.

## For the orchestrator

Accepted head `4f92abf3`. The scope is already committed on
`task-board/story/STORY-260831-yr0x81` as `1b379e2` + `4f92abf`; there is no
uncommitted delta to stage.
