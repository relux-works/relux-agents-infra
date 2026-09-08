# Native authentication isolation contracts

Task: `TASK-260720-3moaky`

Date: 2026-08-30

## Context and safety boundary

This audit determines whether agents-infra can isolate multiple Claude Code and
Codex CLI accounts and take macOS Keychain custody after a vendor-native login.
The target workstation normally runs concurrent sessions, so refresh and
write-back behavior is part of the decision.

This document contains paths, field names, backend semantics, and public
protocol shapes only. The audit did not read a real credential file, query an
existing Keychain item, inspect an authenticated account, run logout, revoke a
token, rotate a token, or complete a login. Behavior that would require any of
those actions is `unknown`.

Claim labels:

- **Supported**: stated in current public vendor documentation.
- **Current source**: implemented in the official Codex source at the exact
  installed release, but not necessarily promised as a stable public contract.
- **Internal / forbidden**: present in source but explicitly unstable,
  internal-only, or unsuitable as an architecture dependency.
- **Observed**: reproduced with an empty profile and no credential material.
- **Unknown**: not documented and not safely established within this audit.

Audit versions:

- Codex CLI `0.150.1`; official source tag `rust-v0.150.1`, commit
  `90854393966b21e9ebfd21b122334eb09a20c93d`.
- Claude Code `2.1.248`.

## Decision summary

Native OAuth credentials cannot be transferred into an agents-infra-owned
Keychain item without reading and copying the secret. Neither vendor publishes
a supported "adopt this native login into another credential store" operation.
That transfer is therefore forbidden for this architecture.

Use a hybrid model:

1. **Native human login stays vendor-owned.** agents-infra selects the vendor
   state boundary before login and never handles credential bytes.
2. **Agents-infra custody is limited to vendor-supported injected credentials**
   such as API keys, access tokens, or workload identity inputs that originate
   in an agents-infra-controlled secret lifecycle.
3. **Codex native login can be isolated per `CODEX_HOME` with a version gate.**
   `keyring` is a supported store, and Codex 0.150.1 derives its Keychain account
   from canonical `CODEX_HOME`. The derivation is current-source behavior, not an
   explicit public compatibility promise.
4. **Claude native macOS login has no supported per-account selector.**
   `/login` uses one macOS Keychain-backed credential surface. The documented
   `CLAUDE_CONFIG_DIR` credential relocation applies only on Linux and Windows.
   Multiple simultaneous native Claude subscription accounts are therefore not
   a supported agents-infra isolation boundary on macOS.
5. Prefer **workload identity** for managed automation. It avoids long-lived
   vendor credentials and gives the host an explicit refresh boundary. It does
   not solve isolation for personal subscription accounts.

## Provider matrix

| Concern | Codex CLI | Claude Code |
| --- | --- | --- |
| Native enrollment | **Supported:** `codex login` opens ChatGPT browser login; device-code login is available for headless use. API keys and enterprise access tokens can be piped on stdin. | **Supported:** first launch or `claude auth login` opens browser login; callback failure falls back to pasting a code. Account types include Claude subscription, Team/Enterprise, Console, cloud providers, and gateway. |
| Native macOS persistence | **Supported:** `file`, `keyring`, or `auto`. File is `CODEX_HOME/auth.json`; default home is `~/.codex`. | **Supported:** macOS Keychain. Linux uses `~/.claude/.credentials.json` mode `0600`; Windows uses the user-profile equivalent. |
| Per-account native boundary | **Current source:** canonical `CODEX_HOME` is hashed into the Keychain account key. Distinct homes therefore select distinct entries in 0.150.1. | **Unsupported / unknown:** no public native-login Keychain service/account selector. `CLAUDE_CONFIG_DIR` relocates credential files only on Linux and Windows. |
| Native refresh | **Supported:** ChatGPT tokens refresh automatically before expiry. **Current source:** proactive refresh performs an account-matched reload, refreshes only if unchanged, writes the returned token set through the selected storage backend, then reloads memory. A 401 performs reload then refresh. | **Partially supported:** docs describe expiry warnings and failure when a stored login expires and cannot be refreshed. Exact native OAuth refresh timing, write-back, rotation, and cross-process coordination are not documented; record them as **unknown**. |
| Logout | **Supported docs:** clears current stored credentials. **Current source:** CLI calls best-effort OAuth revocation first, then deletes the selected Keyring/file stores even if revocation fails. No live test was run. | **Supported docs:** `/logout` removes managed login state and resets first-launch setup. Server-side token revocation and the exact set of Keychain metadata removed are **unknown**. No live test was run. |
| Host-managed injection | **Supported:** `CODEX_API_KEY` on documented non-interactive surfaces; `CODEX_ACCESS_TOKEN` for ephemeral use; persistent access-token login via stdin; workload identity through a protected token file; custom providers through a named environment variable. | **Supported:** `ANTHROPIC_API_KEY`, `ANTHROPIC_AUTH_TOKEN`, `apiKeyHelper`, `CLAUDE_CODE_OAUTH_TOKEN`, named Anthropic profiles/WIF, and cloud-provider credentials. |
| Agents-infra Keychain custody after native login | **No supported transfer.** Start native login inside its final vendor-owned home instead. An existing login can only be moved by reading/copying cached credentials, which this design forbids. | **No supported transfer.** Native subscription OAuth remains in Claude's Keychain item. `apiKeyHelper` can expose an agents-infra-owned API key to Claude, but cannot adopt Claude's native subscription OAuth cache. |
| Same-provider concurrent sessions | Shared native storage is not documented as a cross-process coordination contract. Codex has mitigations but still has an unclosed simultaneous-refresh window; details below. | Native macOS OAuth concurrency and refresh write-back are **unknown**. API-key helper and WIF have explicit external refresh owners; details below. |

## Codex CLI

### Enrollment and login

**Supported.** ChatGPT subscription login runs a browser OAuth flow. The browser
returns credentials to Codex; `codex login --device-auth` is the preferred
headless alternative when enabled. API-key and enterprise access-token logins
read from stdin. An access token may instead remain ephemeral in
`CODEX_ACCESS_TOKEN`.

An empty-profile smoke used a separate `HOME`, separate `CODEX_HOME`, and
`cli_auth_credentials_store = "file"`:

| Command shape | Exit | Sanitized observation |
| --- | ---: | --- |
| `codex login --help` | 0 | Lists browser/device login plus stdin API-key and access-token modes. |
| `codex login status` | 1 | Prints only `Not logged in`; created no `auth.json`. |
| `codex logout --help` | 0 | Describes removal of stored authentication; no mutation occurs for help. |

### Persistence and secure storage

**Supported.** `cli_auth_credentials_store` accepts:

- `file`: `auth.json` under `CODEX_HOME` (default `~/.codex`).
- `keyring`: operating-system credential store.
- `auto`: Keyring when available, otherwise `auth.json`.

**Current source, 0.150.1.** File storage creates mode `0600` on Unix but writes
by truncating the target in place. Direct Keyring storage serializes the auth
record into service `Codex Auth`; its account key is a hash of canonical
`CODEX_HOME`. The optional encrypted-secrets backend stores encrypted auth under
`CODEX_HOME/secrets/codex_auth.age` and stores its passphrase in the OS Keyring,
also under an account derived from canonical `CODEX_HOME`.

Consequences:

- Distinct canonical homes produce distinct Keychain account names in this
  release without agents-infra touching credential bytes.
- Symlink aliases that canonicalize to the same home intentionally share auth.
- Public docs promise `keyring` as a store, but do not explicitly promise that
  its namespace is keyed by `CODEX_HOME`. agents-infra must pin/test this current
  implementation property before relying on it after a Codex upgrade.
- `auto` silently falls back to a plaintext credential file if Keyring access or
  save fails. A Keychain-only policy must select `keyring`, not `auto`.

### Refresh, rotation, and concurrent sessions

**Supported docs.** ChatGPT tokens refresh automatically before expiration.

**Current source, 0.150.1.** Each process has a one-permit refresh semaphore.
Before a managed refresh it reloads storage only if the account ID matches the
process's cached account. If another writer already changed the token set, the
process adopts it and skips refresh. Otherwise it calls the authority, persists
the returned ID/access/optional replacement refresh token through the selected
backend, records refresh time, and reloads the cache. A 401 recovery follows the
same reload-then-refresh sequence.

This mitigates, but does not prove safe, two-process refresh:

- The semaphore is process-local, not cross-process.
- Two processes can both reload the same unchanged token before either writes,
  then both call the authority.
- File mode has no cross-process lock and its write is truncate-in-place. The
  exact overlap can produce last-writer-wins state and can expose a partial JSON
  read. Keyring writes avoid partial file bytes but still do not establish a
  refresh-token rotation winner.
- Source tests prove adoption when storage changed before the guarded reload;
  they do not prove the exact simultaneous-authority-call schedule.

Therefore:

- Two sessions sharing one `CODEX_HOME` are **not established safe for the exact
  simultaneous refresh/rotation schedule**. Normal concurrent inference may
  work, but that is not the required proof.
- Distinct homes and distinct native logins eliminate local write contention,
  but provider allowance for multiple independently issued refresh credentials
  for the same ChatGPT identity remains **unknown** here.
- A future auth broker could serialize refresh for host-managed credentials,
  but must use a supported credential source rather than the internal protocol
  described below.

### Logout semantics

Public docs say `codex logout` clears current stored credentials. In official
0.150.1 source, the CLI uses `logout_with_revoke`: it attempts to revoke the
managed ChatGPT refresh token (or access token when no refresh token is present)
with a ten-second timeout, then deletes local auth even if revocation fails.
Deletion covers the selected backend plus fallback file and legacy Keyring form.
Workload-identity auth rejects login/logout because the host environment owns it.

This behavior was not exercised because logout would mutate the live session.
Treat server revocation as current-source behavior until public docs promise it.

### Supported injection and isolation boundaries

1. **Per-home managed login:** supported storage selection plus current-source
   Keyring namespace isolation. Best fit for human subscription accounts when
   version-gated.
2. **Ephemeral API key:** documented `CODEX_API_KEY` on `exec`, `review`, SDK,
   and remote exec-server surfaces. No refresh or write-back; the host rotates.
3. **Ephemeral access token:** documented `CODEX_ACCESS_TOKEN`. No persistent
   login is required; host rotates/revokes.
4. **Persistent access-token login:** stdin to `codex login
   --with-access-token`; Codex stores an agent identity credential. This is not
   native ChatGPT OAuth adoption.
5. **Workload identity:** `OPENAI_FEDERATION_RULE_ID` plus a protected absolute
   `OPENAI_IDENTITY_TOKEN_FILE`. Codex keeps the exchanged OpenAI access token in
   memory and writes neither input nor exchanged token to auth storage. The host
   atomically rotates the identity-token file. Separate Codex processes perform
   separate exchanges and need assertions the identity provider permits them to
   use.
6. **Custom model-provider `env_key`:** supported for provider-specific API
   keys; separate from OpenAI ChatGPT login.
7. **Headless `auth.json` copy:** publicly documented as a fallback, but it
   explicitly copies a password-equivalent credential cache. It is forbidden by
   this architecture's custody and no-exposure requirements.
8. **`--remote-auth-token-env`:** transport authentication between a remote
   client and app server, not OpenAI provider authentication. It must never carry
   or reuse a Codex access token.

### Internal boundary that must not be used

Codex 0.150.1 contains an App Server `chatgptAuthTokens` login shape and a
server-to-client refresh callback. It would let the client own access-token
refresh and keep provider credentials process-local. However, the protocol type
is explicitly marked **unstable, for OpenAI internal use only, do not use**. It
also accepts an already obtained access token; it does not provide a supported
way to adopt a native login without exposing credential bytes. This path is
forbidden as an agents-infra dependency.

## Claude Code

### Enrollment and login

**Supported.** First launch or `claude auth login` opens a browser flow. When a
local callback is unavailable, the user pastes the returned code. CLI help in
2.1.248 exposes subscription, Console, and SSO selection. `/logout` or
`claude auth logout` signs out and causes first-launch setup to run again.

Empty-profile commands ran with separate `HOME` and `CLAUDE_CONFIG_DIR`:

| Command shape | Exit | Sanitized observation |
| --- | ---: | --- |
| `claude auth --help` | 0 | Lists login, logout, and status subcommands without accessing auth. |
| `claude auth login --help` | 0 | Lists subscription, Console, email hint, and SSO options. |
| `claude setup-token --help` | 0 | Identifies the long-lived subscription-token flow. |
| `claude --bare -p <fixed smoke prompt>` | 1 | Prints only `Not logged in`; 2.1.248 help promises `--bare` never reads OAuth or Keychain and accepts only API key or `apiKeyHelper` auth. |

No ordinary `claude auth status` was run: on macOS, `CLAUDE_CONFIG_DIR` does not
isolate `/login` credentials from the real Keychain, so status could have read a
live item.

### Persistence and secure storage

**Supported.** Native Claude Code credentials are stored:

- macOS: encrypted macOS Keychain.
- Linux: `~/.claude/.credentials.json`, mode `0600`.
- Windows: `%USERPROFILE%\.claude\.credentials.json` under user-profile ACLs.

The docs say `CLAUDE_CONFIG_DIR` relocates `.credentials.json` on Linux and
Windows only. They do not document a macOS Keychain service/account selector,
an alternate keychain, or a per-profile native `/login` store. Therefore a
different `CLAUDE_CONFIG_DIR` is not evidence of macOS credential isolation.

### Refresh, rotation, and concurrent sessions

Native `/login` behavior is only partially documented. Claude Code warns near
login expiry; after expiry and failed refresh, requests fail until `/login` is
run again. The public docs do not state:

- when native OAuth refresh is attempted;
- whether a refresh token rotates;
- whether refreshed state is rewritten in place or replaced;
- whether two processes coordinate Keychain refresh/write-back;
- whether one process can invalidate the other's cached native credential.

Those properties remain **unknown** because testing them requires live auth
mutation or credential inspection.

Supported external sources have clearer ownership:

- `apiKeyHelper` is invoked after five minutes or after HTTP 401 by default;
  the TTL is configurable. The helper owns retrieval/rotation; Claude consumes
  its stdout and does not write the key back.
- `ANTHROPIC_API_KEY` and `ANTHROPIC_AUTH_TOKEN` are per-process inputs; no local
  refresh/write-back exists.
- `CLAUDE_CODE_OAUTH_TOKEN` is a one-year subscription token generated by
  `claude setup-token`. The flow prints the token and saves nothing. It is
  inference-only and lacks Remote Control and claude.ai connectors. It therefore
  does not meet the "take custody without exposure" requirement.
- Named `ANTHROPIC_PROFILE` and federation credentials rank above `/login` and
  provide supported account selection. User OAuth profiles are file-backed by
  the separate `ant` CLI; WIF refresh re-reads the identity-token file. These
  profiles do not provide subscription-login Keychain custody, and some
  claude.ai features are unavailable while a profile is active.
- Anthropic WIF uses short-lived host identity tokens. With replay protection,
  every exchange needs a fresh `jti`; two processes reading one unchanged token
  file can race or reuse the assertion. Give each process a token source that
  can mint/rotate fresh assertions or serialize exchanges outside Claude Code.

### Logout semantics

Public docs state that `/logout` manages the stored credential and resets
first-launch setup. They do not state whether logout revokes a server token,
which Keychain records or account metadata it removes, or whether another
running process retains a usable in-memory credential. These are **unknown** and
were not tested against the live machine.

### Supported injection and isolation boundaries

1. **`apiKeyHelper`: recommended for agents-infra-owned API credentials.** A
   small helper can read one task-selected agents-infra Keychain item, emit the
   credential only to Claude's captured stdout pipe, and never log it. Use one
   immutable helper/profile selection per account. This is vendor-supported for
   API keys, not native subscription OAuth.
2. **Environment API/bearer tokens:** supported but visible in the process
   environment and inherited by children unless scrubbed. Prefer a helper.
3. **`CLAUDE_CODE_OAUTH_TOKEN`:** supported for headless subscription inference,
   but the vendor generation flow displays the secret and requires manual
   custody. Not suitable for no-exposure adoption.
4. **Named Anthropic profiles and WIF:** supported for Console/API organization
   identities. They provide explicit selection and externally owned refresh,
   but do not represent a macOS native subscription `/login` account.
5. **Bedrock, Vertex, Foundry, and Claude apps gateway:** supported provider
   boundaries with their own credential/SSO lifecycle. Gateway session token is
   the only credential for that selected session.
6. **Native Keychain manipulation or copying to `.credentials.json`:** not a
   supported Claude Code interface and forbidden here.
7. **`--bare`:** a supported hard boundary that never reads OAuth/Keychain and
   only admits `ANTHROPIC_API_KEY` or `apiKeyHelper` via explicit settings. It is
   useful for an agents-infra-owned API-key worker, with the documented feature
   reductions of bare mode.

## agents-infra architecture implications

### What is supported

- Select a unique canonical Codex home before each native login; force
  `cli_auth_credentials_store = "keyring"`; let Codex own enrollment, refresh,
  and logout inside that home.
- For managed Codex automation, prefer workload identity; for enterprise access
  tokens, inject from an external secret lifecycle or persist deliberately.
- For Claude API billing, configure an account-specific `apiKeyHelper` backed by
  agents-infra Keychain storage. Use `--bare` when its reduced surface is
  acceptable and a hard no-native-Keychain-read boundary is desired.
- For managed Anthropic organizations, use named profiles or WIF with explicit
  per-process identity-token rotation.

### What merely works today or needs a version gate

- Codex Keyring account names are derived from canonical `CODEX_HOME` in
  0.150.1. The public setting contract does not promise this derivation.
- Multiple Codex processes often share a home successfully, and source reloads
  a winner before refreshing, but the exact simultaneous refresh race has no
  cross-process lock or proof.
- Any assumption about Claude's native Keychain item naming, refresh write-back,
  or multi-process coordination is undocumented.

### What is forbidden

- Reading, exporting, copying, logging, or rewriting either vendor's native
  credential payload to transfer it into agents-infra custody.
- Calling live logout, revoke, or rotation as a discovery mechanism.
- Relying on Codex App Server `chatgptAuthTokens`; the protocol explicitly says
  internal-only and unstable.
- Treating Codex remote transport tokens as OpenAI provider credentials.
- Treating `CLAUDE_CONFIG_DIR` as macOS native credential isolation.
- Using `auto` when a policy requires Keychain-only Codex storage; `auto` may
  create a plaintext fallback file.

### Required implementation contracts for the architecture task

1. Add an immutable auth-profile identity to agents-infra launch composition.
   The profile selects provider, canonical state root, credential mode, and
   permitted injection mechanism; no secret value enters config or diagnostics.
2. Codex native profile: distinct canonical `CODEX_HOME`, forced `keyring`, and
   a startup version/capability check that refuses if per-home Keyring
   namespacing can no longer be established from the supported/current source.
3. Claude native profile: support at most one macOS vendor-native `/login`
   identity until Anthropic publishes a per-account selector. Do not claim
   multi-account isolation.
4. Claude managed profile: `apiKeyHelper` or WIF only; helper executable and
   Keychain item identity are policy, while stdout/stderr and all diagnostics
   redact secret bytes by construction.
5. Serialize login/logout/profile mutation per auth profile. Do not serialize
   ordinary inference unless provider evidence requires it.
6. Record refresh ownership explicitly: `vendor`, `agents-infra-helper`,
   `identity-provider`, or `manual`. Refuse a profile with no single owner.
7. Keep provider-auth and app-server transport-auth types separate in schema,
   CLI, logs, and tests.
8. Add upgrade smokes using empty profiles and synthetic credentials only.
   Never use a live personal credential as a fixture.

## Unknowns retained deliberately

- Claude native OAuth refresh cadence, rotation semantics, Keychain write-back,
  logout revocation, and same-account concurrent-process behavior.
- Whether OpenAI guarantees per-`CODEX_HOME` Keychain namespacing beyond current
  source, and the exact outcome of two processes refreshing the same rotating
  token at the same instant.
- Whether either subscription provider permits an unlimited number of
  independently enrolled native refresh credentials for one human account.
- Any native-login-to-external-secret-store migration semantics; neither vendor
  documents such a transfer.

## Evidence ledger

### Official public contracts

- [OpenAI Codex authentication](https://learn.chatgpt.com/docs/auth)
- [OpenAI Codex configuration reference](https://learn.chatgpt.com/docs/config-file/config-reference)
- [OpenAI Codex access tokens](https://learn.chatgpt.com/docs/enterprise/access-tokens)
- [OpenAI Codex workload identity](https://learn.chatgpt.com/docs/enterprise/workload-identity)
- [Claude Code authentication](https://code.claude.com/docs/en/authentication)
- [Claude Platform workload identity](https://platform.claude.com/docs/en/manage-claude/workload-identity-federation)
- [Anthropic `ant` CLI authentication](https://platform.claude.com/docs/en/cli-sdks-libraries/cli/authentication)

### Official Codex 0.150.1 source

- [`CODEX_HOME`-derived Keyring storage and file semantics](https://github.com/openai/codex/blob/90854393966b21e9ebfd21b122334eb09a20c93d/codex-rs/login/src/auth/storage.rs#L154-L228)
- [Direct and encrypted Keyring backends](https://github.com/openai/codex/blob/90854393966b21e9ebfd21b122334eb09a20c93d/codex-rs/login/src/auth/storage.rs#L230-L456)
- [Refresh reload guard, write-back, and logout/revoke path](https://github.com/openai/codex/blob/90854393966b21e9ebfd21b122334eb09a20c93d/codex-rs/login/src/auth/manager.rs#L2764-L3028)
- [Best-effort OAuth revocation](https://github.com/openai/codex/blob/90854393966b21e9ebfd21b122334eb09a20c93d/codex-rs/login/src/auth/revoke.rs#L1-L132)
- [Explicitly internal-only external token protocol](https://github.com/openai/codex/blob/90854393966b21e9ebfd21b122334eb09a20c93d/codex-rs/app-server-protocol/src/protocol/v2/account.rs#L83-L103)

### Local sanitized evidence

Task-scoped logs are under `.temp/TASK-260720-3moaky/`. Only the sanitized
observations above are deliverable; raw empty-profile logs are not attached to
the board. The profiles contained no real auth material. The Claude bare smoke
created only mode-`0600` empty-profile state/transcript files. No credential
artifact was created by the Codex status smoke.

