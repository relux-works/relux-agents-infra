# Keychain custody and refresh semantics

Task: `TASK-260720-1g880w`
Feeds: `TASK-260720-3gcfd1` (multi-account auth architecture decision)
Builds on: `TASK-260720-3moaky` (native auth isolation contracts, 2026-08-30)

Date: 2026-08-31

## Safety boundary actually held

No credential, token, cookie or Keychain secret value was read, printed,
exported or persisted. Every Keychain read in this research used
`security find-generic-password` **without** `-w`, so only item metadata
(service, account, class, creation/modification date) was ever returned.

No logout, revoke, rotation or re-auth ran against a live authenticated
session. The live Claude Keychain item and `~/.codex/auth.json` were verified
byte-for-byte unchanged (same `cdat`/`mdat`, same size/mtime) after every
experiment.

Two synthetic Codex Keychain items were created from a **fabricated
non-credential string** (`sk-SYNTHETIC-NOT-A-REAL-KEY-a/b`) under two synthetic
`CODEX_HOME` directories, inspected for metadata only, and deleted. There was
no pre-existing `Codex Auth` item to collide with, and none remains.

Claim labels: **Proven here** (reproduced on this machine in this task),
**Current source** (read out of the installed 2.1.248 / 0.150.1 artifacts),
**Supported** (public vendor documentation), **Unknown**.

Versions: Codex CLI `0.150.1`, Claude Code `2.1.248` — identical to the
versions `TASK-260720-3moaky` audited, so that audit's findings remain
version-current.

## Headline: one prior finding is refuted

`TASK-260720-3moaky` decision summary item 4 states that Claude native macOS
login has **no** supported per-account selector, that `CLAUDE_CONFIG_DIR`
relocates credentials on Linux and Windows only, and therefore that multiple
simultaneous native Claude accounts are not an isolation boundary on macOS.

That is correct about the **documentation** and wrong about the **implementation
at 2.1.248**. `CLAUDE_CONFIG_DIR` namespaces the macOS Keychain service name.

Proven here, by running the real binary against a synthetic empty config dir
with a logging shim in front of `security` that records the non-secret `-a`/`-s`
selectors and nothing else:

```
CALL find-generic-password acct=<redacted:len=6> service=Claude Code-credentials-31c6920d
CALL find-generic-password acct=<redacted:len=6> service=Claude Code-31c6920d
```

`31c6920d` is exactly `sha256(NFC(<synthetic dir>)).hex[:8]`, computed
independently. The child reported `Not logged in` while the live default-
namespace item sat untouched in the same Keychain.

The corresponding construction in the 2.1.248 bundle:

```js
var eq = "-credentials";
function ok(n = "") {
  let e = process.env.CLAUDE_SECURESTORAGE_CONFIG_DIR,
      t = e !== void 0 ? !e : !process.env.CLAUDE_CONFIG_DIR,
      r = e !== void 0 ? e.normalize("NFC") : ge(),
      c = t ? "" : `-${a("sha256").update(r).digest("hex").substring(0, 8)}`;
  return `Claude Code${Gt().OAUTH_FILE_SUFFIX}${n}${c}`;
}
function ME() {                       // the Keychain ACCOUNT
  let n;
  try { n = process.env.USER || u().username } catch { n = "claude-code-user" }
  if (!/^[a-zA-Z0-9._-]+$/.test(n)) return "claude-code-user";
  return n;
}
```

The live item corroborates the other half: its service is exactly
`Claude Code-credentials`, with no hash suffix, which is what `ok()` produces
when `CLAUDE_CONFIG_DIR` is unset.

This does **not** promote the mechanism to *supported*. Anthropic documents
`CLAUDE_CONFIG_DIR` as a Linux/Windows credential-file relocation. The macOS
Keychain namespacing is current-source behaviour with no compatibility promise,
so it needs the same version gate as the Codex equivalent — see the proof
gates.

## Per-provider feasibility matrix

| Dimension | Codex CLI 0.150.1 | Claude Code 2.1.248 |
| --- | --- | --- |
| Default custody on this machine | **Proven here:** file. `~/.codex/auth.json`, mode `0600`, 4003 bytes. No `Codex Auth` Keychain item exists. The packaged client bakes `cli_auth_credentials_store = "file"` into its fixed defaults, so file is the effective default, not `auto`. | **Proven here:** macOS Keychain. `~/.claude/.credentials.json` is absent; a generic-password item exists in the login keychain. |
| Keychain item coordinates | **Proven here:** service `Codex Auth`, account `cli\|<sha256(canonical CODEX_HOME).hex[:16]>`. Two synthetic homes produced `cli\|c2e40b5f045197b0` and `cli\|c8c1b4ad6d576395`, matching the independently computed digests exactly. | **Proven here:** service `Claude Code${OAUTH_FILE_SUFFIX}-credentials[-<sha256(NFC(configDir)).hex[:8]>]`, account `$USER` (fallback `claude-code-user`). `OAUTH_FILE_SUFFIX` is empty in this build. A second item `Claude Code[-<suffix>]` is read alongside it. |
| Multi-account mechanism | **Proven here:** `CODEX_HOME`. Two homes → two Keychain items → two simultaneously "logged in" identities; a third, never-enrolled home correctly reported `Not logged in` (negative control). | **Proven here:** `CLAUDE_CONFIG_DIR`, or the undocumented `CLAUDE_SECURESTORAGE_CONFIG_DIR`, which repoints the credential namespace independently of the config dir. Read routing proven; a second *live subscription* account was not enrolled (see unknowns). |
| Refresh write-back | **Current source:** proactive refresh reloads storage, adopts a competing writer's token if it changed, otherwise calls the authority and persists the returned token set through the selected backend. File mode truncates in place. | **Proven here + current source:** `security add-generic-password -U -a <acct> -s <svc> -X <hex(json)>` — `-U` updates the existing item in place. The live item's `cdat` is 2026-08-19 and `mdat` is 2026-08-31, i.e. the same item has been rewritten without a re-login. |
| Write-back transport hazard | File mode writes bytes to disk; keyring mode goes through the `keyring` crate. | **Current source:** the hex payload is fed on `security -i` stdin only while the command line is ≤ 4032 bytes; above that it is passed **as argv**, putting the hex-encoded credential in the process table for the duration of the call. |
| Refresh concurrency | **Current source:** a process-local one-permit semaphore plus a guarded reload. No cross-process lock. File mode is truncate-in-place, so the exact simultaneous-refresh schedule can yield last-writer-wins and partial reads. | **Current source, and better than the prior audit assumed:** writes are serialized cross-process by a `proper-lockfile` on `<storageDir>/.storage-write` (10 retries, 100–1000 ms backoff, 15 s stale reclaim). Lock scope and Keychain namespace are both derived from the same storage dir, so they cannot disagree. |
| Read caching | Not established in this task. | **Current source:** 30 s in-memory cache; on read failure it logs `[keychain] read failed; serving stale cache` and re-serves the stale value rather than failing closed. A 1 s failure backoff suppresses retries. |
| Failure posture under custody interference | **Unknown** for keyring mode. `auto` (not the packaged default) falls back to a plaintext `auth.json`. | **Current source: fails OPEN.** The store is a composite `keychain-with-plaintext-fallback`. On a non-transient Keychain write failure it writes `<storageDir>/.credentials.json` at mode `0600` with `Warning: Storing credentials in plaintext.`, **and deletes the Keychain item if one existed**. On a later Keychain success it deletes the plaintext file. |
| External custody of the native credential | **No supported transfer.** Moving an existing login requires reading and copying the secret. | **No supported transfer.** Same. Additionally, because Claude Code shells out to `/usr/bin/security`, its item's ACL necessarily trusts `security`, so any process running as the same user could read it with `-w` and no prompt — that is a custody *hazard* to defend against, not a mechanism to use. |
| Supported host-owned injection | `CODEX_API_KEY`, `CODEX_ACCESS_TOKEN`, stdin access-token login, workload identity, per-provider `env_key`. | `ANTHROPIC_API_KEY`, `ANTHROPIC_AUTH_TOKEN`, `apiKeyHelper`, `CLAUDE_CODE_OAUTH_TOKEN`, named profiles / WIF, cloud providers. |
| What breaks if agents-infra takes custody of the item | Codex re-creates its own item on the next refresh under the `CODEX_HOME`-derived account. An externally held copy silently becomes a stale credential. | Worse. The next refresh overwrites the item in place (`-U`); an externally held copy goes stale within one refresh interval. If agents-infra makes the item unwritable to enforce custody, Claude does not fail closed — it **migrates the credential to a plaintext file and deletes the Keychain item**, which is the exact opposite of the intended security posture. |

### A near-miss worth naming

The Codex binary contains `credential_broker` symbols and
`network-proxy/src/credential_broker/providers/{github,openai}.rs`. This is
**not** a custody hook for Codex's own provider authentication — it is the
sandbox network proxy's MITM header-injection feature for the agent's outbound
traffic. Do not build a custody design on it.

## Hard blockers

1. **No supported adopt-an-existing-login operation, either provider.** Taking
   custody of a native login that already exists requires reading the secret.
   Forbidden by this architecture and by this task's constraints. Custody can
   only ever be established *before* enrolment, by choosing the state root the
   login lands in — never after.
2. **Claude Code fails open, not closed, under custody interference.** A
   read-only or externally-locked Keychain item does not stop Claude Code; it
   makes Claude Code write the credential to disk in plaintext and delete the
   Keychain item. Any design in which agents-infra "holds" the item and denies
   Claude write access actively *downgrades* the credential's protection.
3. **Refresh invalidates an externally-held copy by construction.** Both
   providers write refreshed tokens through their own storage backend. There is
   no publish/subscribe, no change notification, and for Claude the write is an
   in-place `-U` overwrite. Any external copy is stale from the next refresh
   onward, silently.
4. **Claude Code's Keychain account is `$USER`, not the identity.** Namespacing
   is carried entirely by the *service* name. Two accounts under one macOS user
   are separated only by the config-dir hash; nothing in the item identifies
   which Anthropic account it holds. Operator-visible account attribution has to
   come from agents-infra's own profile record, not from the Keychain.
5. **Neither namespacing derivation is a promised contract.** Codex's
   `cli|sha256(CODEX_HOME)[:16]` and Claude's `sha256(configDir)[:8]` service
   suffix are both current-source. A minor version bump can orphan every
   enrolled profile with no error at any layer.
6. **The plumbing to run two accounts simultaneously does not exist in
   agents-infra today.** See the `home_env` section below. This is a hard
   blocker for the *product* capability, and it is the one blocker on this list
   that is entirely ours to remove.

## Unknowns, each with the experiment that settles it

| Unknown | Experiment that settles it | Cost / risk |
| --- | --- | --- |
| Does Anthropic permit two concurrently-enrolled native subscription logins for one human, or does enrolling the second invalidate the first server-side? | Enrol a **second, disposable** Anthropic account under a distinct `CLAUDE_CONFIG_DIR`, leave the first live, and check that both keep working for 24 h. | Needs a second real account. Cannot be answered with synthetic material. Does not risk the existing session, because the namespaces are provably disjoint. |
| Same question for OpenAI/ChatGPT and `CODEX_HOME`. | Same shape, with a disposable ChatGPT account and `cli_auth_credentials_store = "keyring"`. | Same. |
| Claude Code native OAuth refresh *cadence* — how far before expiry, and whether the refresh token rotates. | Instrument a **synthetic** profile holding a fabricated credential shaped like the real payload, and watch which `security` subcommands the shim records over time. Requires knowing the payload schema, which requires reading a real one. **Blocked by the no-read constraint** unless the schema is obtained from a disposable second account. | Free once a disposable account exists; otherwise blocked. |
| Whether Codex file-mode auth is rewritten on every refresh. `~/.codex/auth.json` has an mtime of 2026-08-27 while the machine has been in use since. That is consistent with "no refresh happened" (Codex credits were exhausted and spawns were routed to Claude) *and* with "refresh does not always write". | Watch the mtime across a window in which Codex is definitely exercised, under a disposable account. | Free with a disposable account. |
| Whether `security delete-generic-password` on a live item makes Claude Code re-login or fail. | Only answerable by deleting a live item. **Do not run.** Approximate it with a disposable account. | Destroys a live session if run against the real item. |
| Whether Claude's cross-process `.storage-write` lock also covers the *Keychain* against a non-Claude writer. It cannot — the lock is advisory and file-based, so an external writer that does not take it will race. | Two synthetic profiles, one Claude-shaped writer taking the lock and one not, both writing the same synthetic item; observe interleaving. | Free, synthetic only. Worth running before any design depends on the lock. |
| Whether `OAUTH_FILE_SUFFIX` is ever non-empty in a shipped build (staging/internal channels), which would move the service name. | Inspect the constant across the last few installed versions under `~/.local/share/claude/versions/`. | Free. Not run here. |

## Custody model comparison

The decision `TASK-260720-3gcfd1` has to make is between three shapes. Only one
of them survives the evidence.

**A. External Keychain custody — agents-infra owns the credential record.**
No-go for native subscription logins, both providers. It requires reading the
secret to establish, it is invalidated by the next refresh, and for Claude the
enforcement mechanism (denying the CLI write access) triggers a documented
fail-open to plaintext. It remains viable *only* for credentials whose lifecycle
agents-infra already owns end to end — an API key it minted, a WIF identity
token file it rotates — which is not custody of a native login at all.

**B. Opaque native-profile isolation — agents-infra owns the state root, the
vendor owns the credential.** This is the model the evidence supports. The
credential never leaves the vendor's own store; agents-infra selects
`CODEX_HOME` / `CLAUDE_CONFIG_DIR` before enrolment and never touches the bytes.
Both providers namespace their Keychain items by that root, proven here for
both. Refresh, rotation, revocation and concurrency all stay the vendor's
problem, which is precisely what makes the model robust: there is nothing for
agents-infra to keep in sync. Its costs are the two version gates and the fact
that operator-visible account attribution must be maintained separately.

**C. Provider-native switching / delegation.** Codex workload identity and
Anthropic named profiles / WIF are genuinely supported and have explicit refresh
owners. They solve managed-automation identity well and personal-subscription
multi-account not at all.

The honest recommendation is B for subscription accounts, C for managed
automation, A never for native logins — which is a sharpening of, not a
departure from, `TASK-260720-3moaky`'s hybrid recommendation. The material
change is that B is now known to work on macOS for **both** providers, where the
prior audit could only establish it for Codex.

## The `home_env` gap — confirmed, with one correction

The gap is real and it is worse-shaped than the brief describes. It exists at
three layers, and the brief's own framing is slightly off at each.

### Layer 1 — the config field is not validated, only trimmed

`pkg/remoteconfig/spawn_runtimes.go:39` declares
`HomeEnv string \`json:"home_env,omitempty"\``. The only thing
`normalizeAndValidateSpawnRuntimesConfig` does with it is line 99:

```go
declared.HomeEnv = strings.TrimSpace(declared.HomeEnv)
```

The sibling fields `agentic_system` and `broker` are parsed, range-checked
against closed vocabularies, and refused with typed errors. `home_env` gets
none of that: no environment-variable-name syntax check, no check that it
agrees with the agentic system's declared `HomeEnvVar`, no check that the
variable resolves to anything. So the brief's "validated at config-parse time"
is **refuted** — it is *normalized* at parse time. A typo'd `home_env` is
accepted in silence, which is strictly worse than being validated-and-unused,
because it looks configured.

The field's only other non-test reference in the whole repository is
`tools/board-cli/cmd/auth.go:578-579`, which copies it into the read-only
`project_config()` report:

```go
if declaration.HomeEnv != "" {
    entry["home_env"] = declaration.HomeEnv
}
```

Declaration, trim, report. That is the complete list.

### Layer 2 — the launch plane computes a home and throws it away

`skill-agents-management` does carry a per-launch home.
`agentic.LaunchRequest.Home` is documented as "overrides the harness
configuration home for this launch … On-disk limit state is keyed by
(provider, home), so this value is load-bearing beyond the launch", and
`pkg/agentic/plan.go:176-190` resolves it:

```go
home := strings.TrimSpace(req.Home)
if home == "" {
    home = caps.DefaultHome
}
```

`Plan.Home` is then never read by any production code. `ChildEnv` is built from
`sys.ChildEnv(req.Env, req)` and neither the codex nor the claude plugin's
`childEnv` writes `HomeEnvVar` into it — grep for `CODEX_HOME` /
`CLAUDE_CONFIG_DIR` across non-test source in that repo returns only comments
and the two `Capabilities` declarations. And the actual launch site,
`skill-project-management` `tools/board-cli/internal/spawn/spawn.go:938-945`,
consumes exactly five fields:

```go
cmd := exec.Command(plan.Binary, plan.Argv...)
cmd.Env = plan.Env
if plan.Stdin.Attached { cmd.Stdin = bytes.NewReader(plan.Stdin.Bytes) }
cmd.Dir = plan.WorkDir
```

`plan.Home` is not among them, and there is no other `.Home` read anywhere in
`internal/spawn`. **Confirmed: not injected into the child environment.**

The one place a provider home *is* injected into a child is
`tools/board-cli/cmd/codex_manager.go:314`, and that is the session-manager
private-`CODEX_HOME` context-fork feature — a different mechanism serving a
different purpose, and not a multi-account boundary.

### Layer 3 — the limit plane reads the parent's environment

`pkg/providerlimits/identity.go:112-123` resolves the provider home like this:

```go
if capabilities.HomeEnvVar != "" {
    if envValue := os.Getenv(capabilities.HomeEnvVar); strings.TrimSpace(envValue) != "" {
        return envValue, nil
    }
}
return capabilities.DefaultHome, nil
```

So the brief's "not consulted by limit-plane identity resolution" needs a
correction: the limit plane **does** consult a home environment variable — the
agentic system's declared `HomeEnvVar` — but it reads it from `os.Getenv`, i.e.
from the **parent orchestrator's own process environment**. It never sees the
launch's `Home`, and it never sees the config's `home_env`. The plumbing exists
and is wired to the wrong source.

The consequence is the sharp one. If a child were launched under a different
`CODEX_HOME` today, `IdentityKey(provider, home)` in the parent would still hash
the *parent's* home, so both accounts' rate-limit state would collide on one
on-disk file — and `identity.go`'s own header comment names exactly this failure
("a home that drifted between the launch and the limit lookup keys state under
a name the launch never uses, silently").

### Verdict

**Confirmed:** an operator cannot get two genuinely separate simultaneous
accounts for one agentic system today. `home_env` is inert, `Plan.Home` is
discarded at the launch site, and limit identity keys off the parent's
environment.

**Corrected on two points:** `home_env` is normalized rather than validated, and
the limit plane is not blind to home env vars — it reads the wrong one.

**What closing it requires** (for `TASK-260720-3gcfd1` to scope, not for this
task to implement): route `home_env` → `LaunchRequest.Home`; have each system's
`ChildEnv` write `HomeEnvVar=<Plan.Home>` into the child environment; and give
`providerlimits` an explicit-home entry point on the spawn path so identity is
keyed by the home the launch actually used. Every one of those three needs a
negative test that fails when the home is dropped, because all three current
failures are silent.

## Proof gates before any real-credential experiment

No experiment involving a real credential runs until all of these hold.

1. **A disposable second account exists** for the provider under test, and the
   experiment is written so that the *existing* live account is never the
   subject. Namespace disjointness is verified first, by metadata only, exactly
   as done in this task.
2. **The namespace derivation is pinned by a test** that fails when the
   derivation changes: for Codex, that `cli|sha256(canonical CODEX_HOME)[:16]`
   still selects the item the CLI writes; for Claude, that
   `sha256(NFC(CLAUDE_CONFIG_DIR))[:8]` still appears in the service name the
   CLI queries. Both are reproducible without any credential, using the shim
   and synthetic-item techniques from this task.
3. **A version gate refuses the launch** when the installed CLI version is
   outside the range those pins were established against, rather than silently
   sharing one account across profiles.
4. **The fail-open path is closed or accepted in writing.** For Claude, a
   profile that cannot write its Keychain item will write plaintext and delete
   the item. Either agents-infra detects that state and refuses the run, or the
   architecture record states explicitly that plaintext fallback is tolerated.
   Silence is not an option here.
5. **No secret crosses a boundary agents-infra controls.** No `-w`, no
   `-X`-bearing argv constructed by us, no credential in a log, an env var, a
   board resource, a diagnostic, or a test fixture. Synthetic payloads only.
6. **Every experiment is reversible and its cleanup is verified**, as this task
   verified: items deleted, absence re-checked, and the live items' `cdat`/
   `mdat` and file mtimes re-read to prove they were not touched.
7. **Concurrency claims are proven by a race, not by a single-process run.** The
   `.storage-write` lock and Codex's process-local semaphore both look adequate
   in a sequential test and are the two places a two-account design is most
   likely to be wrong.

## Evidence ledger

All commands are read-only unless marked. Reproduced on this machine
2026-08-31.

| # | What it establishes | Command shape | Result |
| --- | --- | --- | --- |
| 1 | Codex default custody is file, not Keychain | `stat ~/.codex/auth.json`; `security find-generic-password -s "Codex Auth"` | file present `0600` 4003 B; no Keychain item |
| 2 | Packaged Codex fixes `file` as the default store | byte-scan of the vendored `codex` binary at the fixed-defaults block | `cli_auth_credentials_store = "file"` |
| 3 | Claude custody is Keychain, not file | `stat ~/.claude/.credentials.json`; `security find-generic-password -s "Claude Code-credentials"` | file absent; generic-password item present |
| 4 | Claude refreshes in place | item `cdat` `20260819081931Z` vs `mdat` `20260831002314Z` | same item rewritten, no re-login |
| 5 | Claude service name is config-dir-namespaced | `security` shim + `CLAUDE_CONFIG_DIR=<synthetic>` + `claude -p` | queried `Claude Code-credentials-31c6920d`; `Not logged in` |
| 6 | The suffix is `sha256(NFC(dir))[:8]` | independent digest of the synthetic path | `31c6920d`, exact match |
| 7 | Codex Keychain account derivation (**mutating, synthetic, reverted**) | two synthetic `CODEX_HOME`s + `codex login --with-api-key` on a fabricated string | `cli\|c2e40b5f045197b0`, `cli\|c8c1b4ad6d576395`, both matching computed digests |
| 8 | Two Codex identities coexist | `codex login status` per synthetic home | home a and home b each report their own key; a third home reports `Not logged in` |
| 9 | Cleanup verified | `security delete-generic-password` ×2, then re-query | no `Codex Auth` item remains |
| 10 | Nothing live was touched | re-read live item `cdat`/`mdat` and `~/.codex/auth.json` mtime after every step | unchanged throughout |
| 11 | `home_env` is inert | `grep -rn HomeEnv` over non-vendor, non-test source | 4 references: field, doc, trim, report |
| 12 | `Plan.Home` is discarded at launch | `spawn.go:938-945`; `grep '\.Home'` over `internal/spawn` | five fields consumed, `Home` not among them; no other reader |
| 13 | Limit identity reads the parent env | `pkg/providerlimits/identity.go:112-123` | `os.Getenv(capabilities.HomeEnvVar)` |

Local scratch, including the `security` observation shim and its log, is under
`.temp/TASK-260720-1g880w/`. The shim logs only the subcommand and, for reads,
the service name and the *length* of the account — never a value, never a
payload, never stdin.

## Recommendation for `TASK-260720-3gcfd1`

Adopt model **B**, opaque native-profile isolation, for both Codex and Claude
subscription accounts on macOS, with model **C** for managed automation. Reject
model **A** for native logins outright — not as a preference, but because
Claude Code's plaintext fallback makes the enforcement mechanism actively
harmful.

The gating work is not on the provider side. It is the three-layer `home_env`
gap: until `home_env` reaches the child environment and the limit plane keys
identity by the launch's home, the isolation mechanism that both vendors
provably support cannot be used by agents-infra at all.
