# Extensible auth-method and credential lifecycle

Task: `TASK-260720-bk6owf`
Feeds: `TASK-260720-3gcfd1` (multi-account auth architecture decision)
Builds on: `TASK-260720-1g880w` (keychain custody and refresh semantics, accepted
2026-08-31), `TASK-260720-3moaky` (native auth isolation contracts, 2026-08-30)

Date: 2026-08-31

## 0. What this document is

A design. It defines a data model, an adapter contract, a lifecycle, a CLI
grammar, and the prerequisites those need in the repositories that own the
launch plane. It implements nothing, enrols nothing, migrates nothing, and
changes no production code.

Everything here is constrained by evidence from `TASK-260720-1g880w`, which was
independently re-derived by its reviewer. Where that evidence says *unknown*,
this design says unknown and refuses rather than guessing. Where it names a
hazard, the design says which half of it is eliminated by the shape of the
model and which half is only held by a checked gate, rather than averaging the
two into one reassuring word (6.1).

## 1. Safety boundary held by this task

No credential, token, cookie or Keychain secret was read, printed, exported or
persisted. No Keychain item was queried, created, modified or deleted. No
`security` invocation of any kind was made. No login, logout, revoke, rotation
or re-authentication ran. The only commands executed were source reads over
three checked-out repositories and a Go build over this one.

## 2. The five facts this design is built on

Restated compactly, because every later section refuses something on account of
one of them.

1. **There is no adopt-an-existing-login operation, either provider.** Custody
   can only be chosen *before* the credential exists, by selecting the state
   root the login lands in.
2. **Claude Code fails OPEN to plaintext.** On a non-transient primary-store
   failure it writes `<state root>/.credentials.json` at mode 0600 and *deletes
   the Keychain item*. `transient` is set only on timeout, so an ACL or
   permission denial — the exact mechanism external custody would use — takes
   the fallback path. Any design that enforces custody by denying the CLI access
   to its own credential converts a Keychain item into a plaintext file.
3. **The namespace derivations work and are undocumented.** Claude:
   service `Claude Code-credentials-<sha256(NFC(configDir)).hex[:8]>`, account
   `$USER`. Codex: service `Codex Auth`, account
   `cli|<sha256(canonical CODEX_HOME).hex[:16]>`. Neither is a promised
   contract; a minor bump can orphan every enrolled profile silently.
4. **The two providers do not share a custody model.** Codex's packaged default
   is `cli_auth_credentials_store = "file"` (`~/.codex/auth.json`, 0600); Claude
   defaults to the macOS Keychain.
5. **The plumbing to run two accounts at once does not exist.** `home_env` is
   normalized and never used; the launch plane computes a home and discards it;
   the limit plane reads the *parent's* environment. Section 9 names the fix.

## 3. Data model

### 3.1 The five separations

| Entity | What it is | What it is NOT |
| --- | --- | --- |
| **Provider** | The authentication authority: `anthropic`, `openai`. Owns identities, issues and revokes credentials. | Not the CLI. Not the subscription plan. |
| **Runtime** | The vendor CLI whose state root carries the namespace: `claude`, `codex`. A provider may gain more than one; a runtime belongs to exactly one provider. | Not an account. Not a profile. |
| **Profile alias** | An operator-chosen local name, unique per provider, stable for the profile's whole life. The only thing an operator ever has to type. | Not derived from the identity. Not a display name that can be edited freely — see the immutability rule in 3.6. |
| **Account identity** | The provider-side identity as a *claim*: email, account id, organization, plan. Carries its own verification state. | Not a secret. Not authoritative unless the vendor confirmed it. Not a selector on its own for destructive verbs. |
| **Auth method** | How the credential was obtained and who therefore owns its refresh: `subscription-oauth`, `browser-oauth`, `device-code`, `api-key`, `enterprise-sso`, `workload-identity`, `coding-plan`. | Not a credential. Not a provider capability — a method exists per provider or does not. |
| **Credential handle** | An opaque *locator* for where the vendor keeps the secret, plus the non-secret coordinates needed to observe presence. | **Never a container.** It holds no secret, and no operation dereferences it to a value. |

Six rows for five separations: `runtime` is separated out because the namespace
attaches to the CLI's state root, not to the authority. Today the mapping is
1:1 in both directions; the model does not assume it stays that way.

### 3.2 Custodian classes, and the invariant that makes the model safe

The custodian class is **not a declared property of anything**. It is a total
function of the auth method, evaluated in code:

```
custodian_class : Method -> vendor-opaque | host-owned | provider-delegated
```

| Class | Who holds the bytes | Who refreshes | Methods |
| --- | --- | --- | --- |
| `vendor-opaque` | The vendor's own store, inside a state root agents-infra allocated. agents-infra never touches the bytes. | Vendor | `subscription-oauth`, `browser-oauth`, `device-code`, `coding-plan` |
| `host-owned` | agents-infra's own secret lifecycle, for a credential agents-infra minted. | agents-infra | `api-key` |
| `provider-delegated` | An external authority; agents-infra rotates an input file or names an env var. | External authority | `workload-identity`, `enterprise-sso` |

The table above is the *rendering* of that function, not its input. A method
registry entry has no custodian field. There is no custodian column in a config
file, no `--custodian` flag, no environment variable, no remote-config key and
no plugin hook. The registry declares which methods exist for a provider; the
class comes back from the function.

**Invariant C1.** `subscription-oauth`, `browser-oauth`, `device-code` and
`coding-plan` map to `vendor-opaque`, and no input the running system accepts
can produce a different pairing — because the pairing is not an input. "the host holds the native login" is not a state that gets declared and then
rejected at load; it is a state the type has no field to hold. That is what
closes the fail-open hazard: not a warning, and not a load-time check that
a future entry point could forget to call or that a bypass path could route
around. There is no check to bypass.

**What is checked rather than structural, stated plainly.** Three things. Two
are not surfaces the running system exposes. The third is, and it is the one an
earlier revision of this section missed by enumerating the surfaces it had
already thought of:

- The function is total by construction: its fallthrough arm returns *no class
  established*, and `describe()` refuses a method that reaches it. A method
  added to the enum without a mapping arm refuses at first use instead of
  defaulting to anything. That is D1's shape (6.2) applied to custody.
- A test asserts the mapping is exhaustive over the method enum and that the
  four native-OAuth methods map to `vendor-opaque`. Its negative variant matters
  more than its positive one: the test must fail when an arm is *changed* to
  another class, not only when the mapping is deleted (6.4).
- **The function's sole input lives on an editable on-disk surface.**
  `custodian_class = f(method)`, and `method` is a field of the profile record
  (3.5) — the same kind of artifact 3.3 refused to store the *class* in, and for
  the reason 3.3 states: an operator or a migration can edit it. Deleting the
  derived value while keeping its only input in that same file does not remove
  the input surface; it adds one level of indirection. 3.5 marks `method`
  immutable after enrol, and on its own that is an assertion with no enforcement
  — the same kind of assertion 3.3 declined to trust. A record rewritten from
  `subscription-oauth` to `api-key` passes H3 (the coordinates did not change),
  passes the pin (the derivation did not change) and reads `active` in 6.2 (the
  credential is still there), and would then be evaluated on the `host-owned`
  branch, where 5.3's prohibitions do not apply by their own scoping sentence.
  C2 below is the gate that stops it; before C2 there was none.

**Invariant C2 — the computed class must agree with the recorded backend.** At
enrol and at every launch, `f(method)` is compared against the recorded `backend`
and `store_selector` (3.3). `vendor-opaque` admits exactly the backends a vendor
store uses — claude's `keychain`, and codex's `file` or `keyring` as named by
`store_selector`. `host-owned` and `provider-delegated` admit only
`external-file` or `env`, the backends agents-infra or an external authority owns.
Any other pairing refuses with `E_AUTH_CUSTODY_INCONSISTENT`, non-zero exit, no
child process. A record whose `method` alone was rewritten therefore refuses at
its next launch rather than silently changing branch. Its test has to be the
narrowing one: flip `method` in a synthetic record, leave every other field
byte-identical, and the launch must refuse — a test that only fails when
`backend` is *also* cleared proves the fields are read and says nothing about
their agreement.

**C2's residual, stated rather than averaged away.** C2 is a checked gate, not a
structural impossibility, and it belongs in 6.1's second half for that reason. It
converts a one-field edit into a two-field consistent rewrite: a real reduction,
not an elimination, and no defence at all against an actor who already has write
access to both fields. What it does not weaken is C1 — there is still no field
anywhere that names a custodian class, and the rewrite above forges a *method*,
which is a lie the gate can catch, rather than declaring a *custody*, which the
type has no place to hold. Q13 asks whether sealing the record is worth its key.

Changing a native method's custody therefore requires editing that function in a
reviewed commit against a failing test. It is not reachable from configuration,
argv, environment or plugin at all, and the one remaining path — forging the
record's `method` — is what C2 gates.

### 3.3 The credential handle is a locator

```
CredentialHandle {
  # no custodian_class field: the class is computed from the profile's `method`
  # by the total function in 3.2 every time it is needed. Storing it would put
  # the pairing back on an input surface — an on-disk record an operator or a
  # migration could edit — which is precisely what C1 removes.
  runtime         : claude | codex
  state_root      : absolute path, byte-exact as recorded at enrol
  backend         : keychain | keyring | file | memory | external-file | env
  store_selector  : {                       # the non-secret vendor config that picks the backend
      key    : string                       # codex: "cli_auth_credentials_store"; claude: n/a
      value  : string                       # codex: "file" (packaged default) | "keyring"
      source : packaged-default | state-root-config | operator-override
  }
  observed_coords : {                       # non-secret, for presence checks only
      keychain_service : string             # e.g. "Claude Code-credentials-<8 hex>"
      keychain_account : string             # e.g. "$USER", or "cli|<16 hex>"
      file_path        : string             # e.g. "<state_root>/auth.json"
      fallback_path    : string             # "<state_root>/.credentials.json"
  }
}
```

**Which fields of `observed_coords` are populated is decided by `backend`, and
for codex `backend` is decided by `store_selector` — not by the runtime.** The
two providers diverge here (2, fact 4) and the divergence has to be carried into
the derivation, or an implementer would populate the wrong fields and H3 would
compare the wrong thing:

| Runtime | `store_selector.value` | Populated coordinates |
| --- | --- | --- |
| claude | n/a — the backend is the macOS Keychain | `keychain_service`, `keychain_account`, `fallback_path` |
| codex | `"file"` — **the packaged default** | `file_path = <state_root>/auth.json`; keychain fields **empty**, because no keyring item is created |
| codex | `"keyring"` | `keychain_service = "Codex Auth"`, `keychain_account = "cli\|<sha256(canonical CODEX_HOME).hex[:16]>"`; `file_path` empty |

`cli_auth_credentials_store` lives in a config file *inside* the state root, so
its value can change after enrol without the state root's path changing. That is
exactly why it is recorded at enrol and re-read at every launch (H3), and why
`auto` is refused at enrol (5.1 step 6).

`observed_coords` exists so `status` can answer *is there a credential, and is
custody healthy* using `stat` and metadata-only lookups. Three rules bound it:

- **H1.** No operation reads a secret value. Concretely: no `security
  find-generic-password -w`, no read of `auth.json` or `.credentials.json`
  contents, no `-X` payload constructed by agents-infra, ever.
- **H2.** A failed, partial or permission-denied observation is `unknown`, never
  `absent` **and never `active`**. Absence and failure-to-read are different
  facts, and so are health and failure-to-read; all three take different branches
  (6.2). The `active` half is stated here because it is the easier one to lose:
  `absent` is obviously a claim, while `active` reads like the default.
- **H3.** `observed_coords` is *derived and recorded* at enrol and *re-derived
  and compared* at every launch. A mismatch is `namespace-drift` and refuses.
  For codex the re-derivation takes `store_selector` as an input alongside
  `state_root`: a profile enrolled on `file` whose store now reads `keyring`
  derives an entirely different coordinate set, and re-deriving without that
  input would compare a stale shape against itself and report a false match. A
  changed store refuses as `namespace-drift` with a message naming the
  transition; the ways forward are restoring the recorded store value or
  re-enrolling — the same two as any other drift. A store value that cannot be
  read is `unknown` and refuses (H2); it is never assumed to be the default.

### 3.4 Email and OTP are claims and inputs, never primitives

- **Email is an identity claim.** It carries `verification: declared |
  vendor-confirmed | unverified`. `declared` means the operator typed it;
  `vendor-confirmed` means a non-secret vendor status output stated it.
  Verification is never inferred from a proxy signal such as "the login
  succeeded, so the email must have been right".
- **An OTP is a transient method input and can never become a credential
  handle.** The model has no field it could be stored in. On both providers
  today, email and OTP live *inside* the vendor's own browser or device flow and
  agents-infra never observes either. If a provider later ships a native
  CLI email+OTP flow, the adapter's `continue` step streams the code to the
  vendor process's stdin and retains nothing.
- **Invariant S1.** Any method input declared `secret: true` in the method
  registry — OTP, password, API key material — is refused on argv, refused in
  config, refused in environment variables agents-infra composes, and never
  written to a log, a board resource or a profile record. Section 8.4 states the
  CLI-level enforcement.

### 3.5 The profile record

One JSON document per profile, no secrets, machine-scoped:

```
~/Library/Application Support/agents-infra/auth/profiles/<provider>/<alias>.json
```

| Field | Notes |
| --- | --- |
| `provider`, `runtime`, `alias` | Immutable after enrol. |
| `identity` | `{ email?, account_id?, org?, plan?, verification }`. Mutable; `verification` downgrades to `unverified` on any edit. |
| `method` | Immutable after enrol. Changing the method is a new enrolment. The immutability is not self-enforcing — this file is editable, and `method` is the sole input to the custodian class — so it is checked by C2 (3.2), not asserted by the record. |
| `credential_handle` | Per 3.3. `state_root` immutable. |
| `version_pin` | `{ runtime_version_at_enrol, supported_range, pin_test_id, pin_verified_at }`. |
| `custody_state` | One of `declared`, `enrolling`, `active`, `degraded:plaintext`, `degraded:version-unpinned`, `degraded:namespace-drift`, `unknown`, `retired-local`. Per 3.6. `removed` is terminal and is not a value this field ever holds: the record is gone, and only the tombstone may remain. |
| `acknowledgements` | e.g. `plaintext_custody_accepted: { at, reason, operator }`. Absent by default. |
| `enrolled_at`, `last_launch_at`, `retired_at` | Timestamps. |
| `tombstone` | Written by any `remove` that did not positively confirm local invalidation; see 5.4. |

State roots live beside the records:

```
~/Library/Application Support/agents-infra/auth/roots/<provider>/<alias>/
```

**Invariant P1 — profile ↔ state root is a bijection.** Two profiles never
share a state root, and a state root is never reused after `remove`. Enrol
refuses a root that already exists.

**Invariant P2 — the recorded `state_root` string is passed byte-verbatim.**
Codex canonicalizes its home before hashing (current source, 0.150.1). Whether
Claude canonicalizes the config dir beyond NFC normalization is **unknown**.
Under that uncertainty the only safe rule is to never re-derive the path: the
exact string recorded at enrol is the exact string exported at launch. A
different spelling of the same directory — symlink, trailing slash, relative
form — may hash differently and would silently enrol into a second namespace.

### 3.6 Custody state machine

```
declared ──enrol──▶ enrolling ──vendor login observed──▶ active
                        │                                  │
                        ├──observation failed──▶ enrolling  ├──▶ degraded:plaintext
                        │      (observation result:         ├──▶ degraded:version-unpinned
                        │       unknown; 5.1 step 8)        ├──▶ degraded:namespace-drift
                        └──refused/failed──────▶ declared   └──▶ unknown  (observation failed)

active|degraded|unknown ──logout(local: invalidated|already-absent)──▶ retired-local
active|degraded|unknown ──logout(local: unknown|failed)──────────────▶ (self-edge; refuses)
retired-local ──remove(local-logout)──▶ removed
active|degraded|unknown ──remove(leave-vendor-state, --confirm)──────▶ removed
```

- `active` is the only state that launches without an explicit acknowledgement.
  Every other state — `declared`, `enrolling`, every `degraded:*`, `unknown` and
  `retired-local` — **refuses to launch**. Only `degraded:plaintext` has an
  acknowledgement path (6.3).
- **The destructive edge is gated, and `retired-local` is a positive assertion.**
  `retired-local` means a *successful* metadata-only observation showed the local
  credential invalidated or already absent (5.4). A `logout` resolving `local:
  unknown` or `local: failed` does not reach it: the profile stays in whatever
  state it was already in, `E_AUTH_LOGOUT_UNCONFIRMED`, nothing deleted. `remove`
  under `--logout-policy local-logout` is reachable **only** from
  `retired-local`, so on that policy it is reachable only from a
  positively-confirmed logout. Drawing the `retired-local` edge from `unknown`
  would manufacture a claim of local retirement out of a read that failed — the
  same forgery shape as reporting `already-absent` for a denied `stat` — and
  unlike that one it authorizes a deletion.
- **`leave-vendor-state` is the one edge to `removed` that does not pass through
  `retired-local`, and it is not a hole.** It deletes without ever claiming the
  local credential was invalidated: no positive assertion is produced from an
  `unknown`, because none is produced at all. The operator states in the verb
  itself that vendor state is being left behind, `--confirm` is mandatory, and a
  tombstone is written unconditionally with the coordinates the orphan will be
  findable by (5.4). What D1 forbids is *inferring* retirement; declaring
  deliberately that you are not retiring anything is the opposite move.
- `unknown` is reached only by a failed or malformed observation, and only from
  `active|degraded`. It is never collapsed into `active` or into `absent`.
- **An observation that fails during enrol does not produce the `unknown`
  state.** It leaves the profile in `enrolling` with an `unknown` observation
  *result* (5.1 step 8) — the self-edge drawn above. The two are deliberately
  distinct: `enrolling` says the enrolment never completed, `unknown` says an
  enrolled profile could not be observed, and they have different ways forward
  (re-run the vendor login, versus repair the observation). Both refuse to
  launch, so the distinction is diagnostic, not a safety boundary.
- `alias`, `provider`, `runtime`, `method` and `state_root` are immutable across
  the whole machine, so a record can never quietly come to describe a different
  account than the one enrolled.

## 4. AuthMethod adapter contract

### 4.1 Operations

| Operation | Signature | Semantics |
| --- | --- | --- |
| `describe()` | → `MethodDescriptor` | Static: capability vocabulary, declared inputs with `secret` flags, supported runtimes, supported version range — plus the custodian class **rendered** from the total function in 3.2. The framework computes the class and fills that field; the adapter neither supplies nor overrides it, because a descriptor field an adapter author writes is the registry column C1 removed wearing a different name. A method whose mapping arm is missing reaches the fallthrough and `describe()` refuses it (3.2). |
| `start(profile)` | → `StartOutcome` | Begins enrolment. For `vendor-opaque` the only legal outcome is `handoff-to-vendor`: agents-infra prepares the state root, gates the version, then runs the vendor's own login command with the home variable set. agents-infra never proxies, intercepts or parses a login flow. |
| `continue(profile, input)` | → `ContinueOutcome` | Second leg for methods that have one — device code, OTP confirmation. Inputs marked `secret` are streamed to the vendor process and retained nowhere. Methods without a second leg declare `continue: unsupported`. |
| `status(profile)` | → `ProfileStatus` | Derived from non-secret observation only: version, pin comparison, presence at `observed_coords`, fallback-file presence. Returns `custody_state`, `identity` with its verification state, and a per-field `unknown` where the observation could not be made. |
| `refresh_capability()` | → `vendor \| host \| external-authority \| none \| unknown` | *Who* refreshes. The owner is **rendered from the custodian class**, not declared: `vendor-opaque` → `vendor`, `host-owned` → `host`, `provider-delegated` → `external-authority` — that is the "Who refreshes" column of 3.2's table, which is the class's own definition. The adapter may only narrow *within* the rendered owner: `none` where the method never refreshes at all, `unknown` where it is not established. It can never name a different owner, because "agents-infra refreshes this native login" is precisely the pairing C1 removed, and an adapter allowed to say it would have restored the declaration under another word. Not an operation: for `vendor-opaque` there is no agents-infra refresh verb, by design (5.3). |
| `logout(profile)` | → `LogoutOutcome` | Local invalidation of one profile. Reports `local: {invalidated \| already-absent \| unknown \| failed}` and `remote_revoke: {revoked \| not-attempted \| unsupported \| unknown}` as **separate** fields. Both carry `unknown`; neither is inferred from the other or from an exit code (5.4). |
| `revoke_capability()` | → `vendor-side-on-logout \| host-side \| unsupported \| unknown` | Declared, never assumed. |
| `local_delete(profile)` | → `DeleteOutcome` | Enumerates exactly what local bytes go away and what provably does not (5.4). |

### 4.2 Capability vocabulary

Four values, and the distinction between the last two is load-bearing:

- `supported` — in current public vendor documentation.
- `current-source` — implemented at the pinned version, no compatibility promise.
- `unsupported` — the vendor does not offer it. A negative fact we can act on.
- `unknown` — not established. **Never** rendered as either `supported` or
  `unsupported`, never defaulted, never inferred from a successful adjacent
  operation. An `unknown` capability that a verb depends on makes that verb
  refuse and say which capability is unknown.

### 4.3 The registry as of today

| Provider | Method | Custodian | Refresh | Remote revoke | Status |
| --- | --- | --- | --- | --- | --- |
| anthropic | `subscription-oauth` | vendor-opaque | vendor (in-place `-U` Keychain overwrite, current-source) | **unknown** | available, version-gated |
| anthropic | `api-key` | host-owned | host | host-side | available |
| anthropic | `enterprise-sso` / named profiles / WIF | provider-delegated | external | external | available for managed automation |
| anthropic | `coding-plan` | vendor-opaque | vendor | unknown | placeholder; no distinct native flow established |
| openai | `subscription-oauth` (ChatGPT browser) | vendor-opaque | vendor | current-source: `logout_with_revoke` best-effort | available, version-gated |
| openai | `device-code` | vendor-opaque | vendor | current-source, as above | available where enabled |
| openai | `api-key` | host-owned | host | host-side | available |
| openai | `workload-identity` | provider-delegated | external | external | available for managed automation |
| qwen | — | — | — | — | **not modellable**: the qwen plugin declares `HomeEnvVar: ""`, so there is no state root to namespace and `vendor-opaque` has nothing to attach to. Provisional, per epic AC8. |

The **Custodian** and **Refresh** columns are both rendered, not stored in these
rows: Custodian from the total function in 3.2, Refresh from the class that
function returns (4.1). A row supplies a provider, a method, its remote-revoke
capability and its status, and says nothing about custody or refresh duty. There
is no way to author a row of this table that disagrees with either column —
including by authoring a plausible-looking `host` in the Refresh column beside a
native method, which is the same claim as a custodian column and is unauthorable
for the same reason. The `openai` rows are custody-independent for the same reason: a codex
profile is `vendor-opaque` on the `file` store and on the `keyring` store alike,
because custody class follows the method, not the filesystem (5.3).

`browser-oauth` is listed in the model as a distinct method for providers whose
browser flow is separable from a subscription; on both current providers the
browser flow *is* the subscription flow, so it is not registered twice.

## 5. Lifecycle

### 5.1 Enrol

Ordered, and the order is the whole safety argument: custody is established
before the credential exists, because afterwards it cannot be.

1. **Validate the request.** Provider, runtime, method and alias must resolve;
   the method must be registered for that provider; the alias must be unused.
2. **Version gate.** Resolve the installed runtime version and compare against
   the method's supported range. Outside it → refuse (7.1). Not readable →
   `unknown` → refuse.
3. **Pin check.** Re-run the namespace derivation pin test for that runtime
   (7.2). Mismatch → refuse.
4. **Environment check.** For claude, `CLAUDE_SECURESTORAGE_CONFIG_DIR` set in
   the operator environment → refuse: it repoints the credential namespace
   independently of the config dir and would silently detach the profile from
   its own root (7.3).
5. **Allocate the state root.** Create it at mode 0700 owned by the operator.
   Mode 0700 on a directory the CLI's own user owns is not a denial to the CLI
   and does not engage the fail-open path; it is the ordinary private-directory
   posture. Refuse if the path exists.
6. **Resolve and record the store selector, then derive and record
   `observed_coords`.** For codex, resolve the effective
   `cli_auth_credentials_store` for the fresh state root and record its value
   and source. agents-infra never writes it: a root it just created holds no
   config, so the value is the vendor's packaged default `"file"` (2, fact 4)
   unless the operator set it through codex's own configuration, and agents-
   infra composes no `-c` override of its own. `"auto"` is **refused** —
   `E_AUTH_CUSTODY_AMBIGUOUS` — because it names no single custody and its
   documented failure path is a fallback to a plaintext `auth.json`, which is
   the fail-open shape section 6 exists to exclude. A store value that cannot be
   read is `unknown` and refuses; it is not assumed to be the default. For
   claude the selector is not applicable and the backend is the Keychain. Then
   derive `observed_coords` from the runtime and the resolved backend, per 3.3.
   All non-secret. Refuse if a credential already exists at those coordinates —
   enrol never overwrites a namespace it did not just create.
7. **Hand off to the vendor.** Run the vendor's own login command as a child
   with the home variable set to the recorded `state_root` string, verbatim.
   agents-infra does not read the flow, the callback, the code or the result
   payload.
8. **Observe.** Presence at `observed_coords` and absence of the fallback file,
   each by a *successful* metadata-only observation. Present + a successful
   observation showing no fallback → `active`. Fallback present →
   `degraded:plaintext` (6.3). Either observation failing, denied or malformed →
   `unknown` result, and the profile stays `enrolling`. An unread fallback path
   is not an absent one, and `active` is a claim about it (H2).
9. **Write the record.** Identity claim as `declared` unless a non-secret vendor
   status output confirmed it.

There is no `adopt` verb and there will not be one. Taking custody of an
existing login requires reading the secret, which H1 forbids.

### 5.2 Use

A launch that selects a profile:

1. Resolve the selector (8.2). Ambiguity refuses before anything happens.
2. Re-run the version gate and the pin check. Either failing → refuse.
3. Re-derive `observed_coords` from the recorded `state_root` and compare
   (H3). Drift → `degraded:namespace-drift` → refuse.
4. Observe custody state. Anything but `active` refuses, unless a matching
   acknowledgement is recorded (6.3).
5. **Write the home variable into the child environment explicitly** —
   `CLAUDE_CONFIG_DIR` or `CODEX_HOME` set to the recorded string.

**Invariant L1 — the home variable is written on every launch of a home-bearing
runtime, never inherited.** This is not a convenience. Today a spawned claude
child inherits the parent's environment minus `CLAUDECODE`
(`skill-agents-management/pkg/agentic/systems/claude/env.go:112`,
`filterRuntimeEnv` strips exactly one key), so a parent that happens to have
`CLAUDE_CONFIG_DIR` set propagates its account to every child regardless of what
that child's runtime declared. Inheritance silently decides the account. When no
profile is selected, the launch writes the *default* home explicitly, so the
variable's value always comes from a decision and never from an ancestor.

**Invariant L2 — one state root, one live enrolment.** Two profiles never share
a root (P1), so the vendors' own concurrency machinery — Claude's
`<storageDir>/.storage-write` lock, Codex's process-local refresh semaphore —
is never asked to arbitrate between two accounts. It only ever arbitrates
between processes of the same account, which is what it was built for.

### 5.3 Refresh

For `vendor-opaque`, agents-infra has **no refresh verb and no refresh duty**.
Its entire obligation is non-interference, stated as a prohibition list because
each item is a plausible good-intention mistake that triggers a real hazard:

| Forbidden | Why |
| --- | --- |
| Copying the credential anywhere | H1; and the copy is stale from the next refresh onward, silently, because Claude's write is an in-place `-U` overwrite. |
| Setting an ACL, changing ownership, or making the item or store unwritable | Non-transient failure → Claude writes plaintext and **deletes the item**. This is the design's own worst outcome. |
| Taking the vendor's `.storage-write` lock | It is the vendor's advisory lock over its own writes. An external holder can only delay or deadlock a refresh. |
| Pre-emptively refreshing on the vendor's behalf | There is no supported operation, and it would race the vendor's own refresh. |
| Deleting or moving the state root while a session may be live | Removes the credential's namespace out from under a running process. |

**On the `file` store the credential is a file inside a directory agents-infra
created, and that changes nothing.** For a codex profile on the packaged default
the secret is `<state_root>/auth.json` at mode 0600 — owned by the operator,
sitting in a root agents-infra allocated, which makes it look like host state.
It is not. Custody follows the method, not the filesystem: `subscription-oauth`
and `device-code` are `vendor-opaque` on either store (C1), so every row of the
table above applies byte-for-byte to `auth.json`. It is never read, copied,
backed up, chmod'ed, moved, staged, or written by anything but the vendor, and
there is still no agents-infra refresh verb. Whether codex rewrites `auth.json`
on every refresh is **not established**; the design depends on no answer,
because it neither reads nor writes the file. Detection under this store is
`stat`-only presence, mode and mtime, exactly as for claude (6.2).

For `host-owned`, agents-infra rotates on its own schedule and the vendor reads
what it is given. For `provider-delegated`, the external authority rotates and
agents-infra atomically replaces the identity-token file it owns.

### 5.4 Retire — `logout` and `remove` are different verbs

**`logout(selector)`** invalidates one local credential profile and nothing
else. It runs the *vendor's own* logout command inside the profile's state root,
because that is the only supported way to make the vendor delete its own item.

Its result has two independent fields. Neither is inferred from the other, and
neither is inferred from the vendor's exit code:

```
local        : invalidated | already-absent | unknown | failed
remote_revoke: revoked | not-attempted | unsupported | unknown
```

**Classification rule for `local`.** `invalidated` and `already-absent` are both
*positive assertions*, and each requires a **successful** metadata-only
observation at `observed_coords` (6.2). The vendor's exit status classifies
nothing on its own.

| Pre-logout observation | Vendor logout | Post-logout observation | `local` |
| --- | --- | --- | --- |
| succeeded, credential present | ran | succeeded, credential absent | `invalidated` |
| succeeded, credential absent | ran or skipped | — | `already-absent` |
| succeeded, credential present | ran | succeeded, credential still present | `failed` |
| failed, denied, or malformed | any | any | `unknown` |
| any | any | failed, denied, or malformed | `unknown` |
| not attempted | exited 0 or non-zero, stating nothing classifiable | not attempted | `unknown` |

**Silence is `unknown`, not `already-absent`.** A vendor logout that exits 0 and
prints nothing has told us nothing: a locked login keychain, a permission error
the vendor swallowed, and a genuine no-op are indistinguishable in that output.
This is D1 (6.2) applied to the one verb whose consequence is irreversible. An
absence and a failure to read are different facts, and the fallback defined for
absence must not fire on a read failure — least of all here, where "absent"
authorizes destruction.

For openai, `revoke_capability` is `vendor-side-on-logout` (current source:
best-effort revoke, then local delete even if revoke failed) — so
`remote_revoke` reports what the vendor's own output stated, and `unknown` when
it stated nothing. For anthropic, `revoke_capability` is `unknown`, so
`remote_revoke` reports `unknown`. It does not report `revoked`, and the CLI
does not print a sentence implying the server-side session ended.

`logout` never touches another profile, never touches the default namespace, and
refuses a selector that resolves to more than one profile.

**`remove(selector)`** is explicitly destructive: it deletes the state root and
the profile record. It is gated on the logout policy being *resolved*, not
merely offered:

```
agents-infra auth remove --provider P --alias A \
    --logout-policy {local-logout | leave-vendor-state} --confirm
```

- `local-logout` runs `logout` first and deletes **only** on `local:
  invalidated | already-absent`. `unknown` and `failed` both refuse:
  `E_AUTH_LOGOUT_UNCONFIRMED`, non-zero exit, nothing deleted, the profile left
  exactly as it was. `unknown` is not a weaker `already-absent` — it is
  precisely the state in which deleting the derivation input would produce an
  orphan nothing can name again, so it is the state `remove` exists to refuse.
- `leave-vendor-state` skips it and is recorded (see the tombstone below).
- Without `--logout-policy`, `remove` refuses. Without `--confirm`, `remove`
  refuses.

**The orphaned-item hazard, and why ordering is load-bearing.** On macOS the
credential does not live inside the state root — the Keychain item lives in the
login keychain under a service name *derived from* the root's path. Deleting the
root therefore does not delete the credential; it destroys the derivation input
and leaves an item that nothing can name any more. So:

1. Vendor logout runs **before** the state root is deleted, while the vendor CLI
   can still find its own item.
2. **A tombstone is written by every `remove` that did not positively confirm
   local invalidation** — `leave-vendor-state` always, and `local-logout`
   resolving to `already-absent`. Only `local: invalidated` deletes without one,
   because only it observed the credential go away. `unknown` and `failed`
   produce no tombstone because they produce no deletion. The tombstone carries
   the recorded store, the resolved `local` and `remote_revoke` values, the
   reason, and the non-secret coordinates for the store the profile actually
   used: `keychain_service` and `keychain_account` for a keychain or keyring
   store, `state_root` plus the credential's relative path for a file store. An
   orphan therefore stays auditable and the operator can clean it up
   deliberately.
3. **agents-infra never deletes a Keychain item.** It did not create the item and
   a wrong derivation would delete some other account's credential. It prints
   the exact non-secret coordinates and lets the operator or the vendor act.

**On the `file` store the orphan hazard does not arise, and a different one
does.** For a codex profile on the packaged default the credential *is* inside
the state root, so deleting the root deletes it and nothing survives that cannot
be named. The ordering above still holds, for a different reason: the vendor's
own logout is what performs openai's best-effort server-side revoke
(`revoke_capability: vendor-side-on-logout`), so `leave-vendor-state` on a
file-store profile destroys the local credential while leaving a *server-side*
session nothing ever attempted to end. The tombstone records that as store
`file`, the state root, the credential's relative path, and `remote_revoke:
not-attempted`. The `local` classification rule is unchanged: a `stat` of
`<state_root>/auth.json` before and after, and a `stat` that fails or is denied
is `unknown`, which refuses.

`local_delete` reports what went away and what provably did not, so a `remove`
that left an orphan says so rather than reporting clean success.

## 6. The plaintext-fallback hazard

### 6.1 What is eliminated, and what is only checked

The hazard needs an *enforcement mechanism* to fire: something has to deny the
CLI access to its own credential. Three separate things stand in the way, and no
two of them are the same kind of thing: one is eliminated, two are checked.
Separating them is the point of this section, because a decision that reads this
document budgets its residual risk from it, and a single reassuring word would
hide the fact that two thirds of the defence runs at build or launch time.

**Eliminated — C1 (3.2).** Custodian class is a total function of the method,
not a field on anything. No config file, flag, registry column, project table,
remote-config key, plugin or environment variable can pair a native-OAuth method
with `host-owned`, because there is nothing to set. This is not a validation
that refuses a bad input; it is the absence of the input. A future entry point
cannot forget to call it and no bypass path can route around it, because there
is no check to bypass. The one way to reach the pairing is to edit the mapping
function in a commit, against a test that fails when an arm changes class.

**One surface that enumeration misses, and it is not a rhetorical one.** The
profile record (3.5) is not a config file, a flag, a registry column, a project
table, a remote-config key, a plugin or an environment variable — it is in none
of the categories the paragraph above enumerates — and it is where the class
function's *input* lives. Rewriting its `method` declares no custody; C1 holds,
there is still no field to hold one. It reaches the `host-owned` branch by
forging the input instead, and every other gate passes through it unchanged
(3.2). That path is closed by C2, which is a **checked** gate and therefore
belongs in this section's second half. Counting it as eliminated would be the
averaging this section exists to refuse, and the enumeration above is exactly how
a claim of the form "nothing can set X" gets verified against the list of things
its author already thought of.

**Mitigated, not eliminated — 5.3, 6.4 and C2 (3.2).** The prohibition list in
5.3 is a list of operations no code path performs: setting an ACL, changing ownership,
making a vendor store unwritable. Nothing in the type system stops an
implementer writing `os.Chmod` against a state root's credential store. What
stops it is 6.4 — a module-wide greppable-absent assertion plus negative tests,
enforced in CI. That is a real gate and it is the primary defence for this half
of the hazard, but it is a *gate*: it runs at build time, and it protects only
what its pattern set names. A new way to spell the same operation, added without
extending 6.4's set, would pass. Widening that set is part of reviewing any
change that touches a state root.

So: the *declaration* half of the hazard is eliminated, the *implementation* half
is mitigated by CI, and the *input* half — forging the method the class is
computed from — is mitigated by a launch-time consistency gate that a two-field
rewrite still passes. Three halves is the wrong word and the right count. They
carry different residual risk and this document does not average them.

5.3's prohibition table is scoped "For `vendor-opaque`", and that scoping is
safe **only because of C1**: a method whose credential lives in a vendor store
can never be evaluated on the `host-owned` branch, where "agents-infra rotates
on its own schedule" would leave the prohibitions silently inapplicable. Remove
C1 and the scoping becomes a hole. It is stated here so the dependency is not
rediscovered later by someone relaxing one of the two.

### 6.2 Detection distinguishes absence from failure to read

The fallback is observable without reading anything: for claude,
`<state_root>/.credentials.json` present means the fallback fired. Detection is
`stat` only — presence, mode and mtime. Contents are never read.

**`degraded:plaintext` means *unexplained* plaintext, not plaintext.** It is a
claim that a store failed open, not a claim about file custody as such. It is
therefore keyed on the profile's **recorded store** (3.3), not on the runtime
and not on the mere presence of a credential file:

| Recorded store | Observation | Custody state |
| --- | --- | --- |
| any | Credential present at `observed_coords`, no unexplained plaintext artifact, **every observation succeeded** | `active` |
| claude Keychain | `<state_root>/.credentials.json` present | `degraded:plaintext` — the vendor failed open and deleted the Keychain item |
| codex `file` | `<state_root>/auth.json` present at 0600 | **`active`** — this is the vendor's packaged default custody, not a fallback |
| codex `keyring` | `<state_root>/auth.json` present | `degraded:plaintext` — codex's keyring failure posture is **unknown**, so unexplained plaintext beside a keyring store is not classified as healthy |
| any | Nothing at `observed_coords` and no plaintext artifact, every observation succeeded | `absent` — the profile is not enrolled |
| any | Any observation failed, was denied, or returned malformed output — **including the store selector** | `unknown` |

Rows one and five carry the same qualifier for the same reason, in the two
directions D1 forbids: `active` is as much a positive claim as `absent` is, and
a fallback path that could not be read has not been shown to be missing. The
`absent` row is the one people remember to harden, because absence obviously
needs evidence; `active` reads like the default and needs exactly as much.

Rows three and four are the same evidence read in two directions, and both are
needed. Calling codex-on-`file` `degraded:plaintext` would refuse every default
codex enrolment for a hazard that did not occur — the credential is in a 0600
file *by design* (2, fact 4). Calling codex-on-`keyring` with an `auth.json`
`active` would be a guess: the research established that codex's `auto` mode
falls back to a plaintext `auth.json` and left the `keyring` mode's failure
posture unknown, so the design refuses rather than inferring in either
direction.

**Invariant D1.** `unknown` never falls back to `absent` and never falls back to
`active`. A permission error reading the state root is not evidence that the
fallback did not fire.

### 6.3 Response

A `degraded:plaintext` profile **refuses to launch** by default, with an error
that names the file, states that the Keychain item was deleted by the vendor's
fallback, and gives the two ways forward: re-enrol (a fresh vendor login into a
fresh state root), or acknowledge.

Acknowledgement is explicit, per-profile, recorded and dated:

```
agents-infra auth launch ... --allow-plaintext-custody --reason "<text>"
```

which writes `acknowledgements.plaintext_custody_accepted` into the record. A
blanket config flag is deliberately not offered: the acknowledgement has to name
one profile and one reason. This satisfies the research's proof gate 4 by taking
its first branch — detect and refuse — while leaving the operator an audited
escape hatch rather than a silent one.

### 6.4 Checkable prohibitions

The following must be greppable-absent from any implementation of this design,
and each deserves a negative test that fails if the operation appears:

`security ... -w`, `security add-generic-password`, `security
delete-generic-password`, `chmod`/`chown`/`chflags` against a state root's
credential store or a Keychain item, reads of `auth.json` or `.credentials.json`
contents, any `-X` payload construction, any acquisition of `.storage-write`.

The two gates this design added after review need their own negatives, and both
have to be **narrowing** rather than deleting, for the reason 7.2 states:

- **C2 (3.2).** Flip `method` from `subscription-oauth` to `api-key` in a
  synthetic record and leave every other field byte-identical: the launch must
  refuse with `E_AUTH_CUSTODY_INCONSISTENT`. A test that only fails when
  `backend` is cleared as well proves the fields are read and says nothing about
  their agreement. Assert also that the *consistent* two-field rewrite is
  **admitted** — C2 claims to catch a one-field forgery and nothing more, and a
  test that pretends otherwise would overstate the gate 6.1 already budgets as
  mitigated.
- **The 3.6 destructive edge.** A `logout` whose post-observation is denied must
  leave the profile in its prior state, must not produce `retired-local`, and
  `remove --logout-policy local-logout` must then refuse and delete nothing.
  Drive it by making the observation *fail* rather than by making the credential
  absent: a test that supplies an absent credential exercises `already-absent`,
  which is the path that is allowed to delete, and would pass against an
  implementation that collapses the two.
- **`describe()` (4.1).** An adapter that returns a `MethodDescriptor` with a
  custodian class disagreeing with 3.2's function must fail to compile or be
  rejected at registration. If the descriptor type makes the field settable at
  all, this test is the only thing standing between the design and the registry
  column C1 removed.

## 7. Version gate and namespace pins

### 7.1 Supported ranges

| Runtime | Initial supported range | Basis |
| --- | --- | --- |
| claude | **2.1.234 – 2.1.248** | The namespacing construction is byte-identical across 2.1.234, 2.1.236, 2.1.247, 2.1.248 (reviewer finding F1 on `TASK-260720-1g880w`). |
| codex | **0.150.1 only** | The account derivation was proven at 0.150.1 and at no other version. No range has been established, and this design does not invent one. Widening it requires running the pin test at the other versions. |

The gate **refuses the launch**; it does not warn. Refusing costs an operator one
error message. Warning and proceeding enrols a profile into an orphaned
namespace, or silently shares one account across two profiles, and neither
failure announces itself. Error code `E_AUTH_VERSION_UNPINNED`, non-zero exit,
no child process started.

A version that cannot be read is `unknown` and refuses. It is not treated as
in-range.

The codex range applies to **both** store branches (3.3). The `file` branch has
no hash that a version bump could break, but the packaged default that *selects*
that branch is itself a version-fixed constant, so a build outside the range can
silently change which branch is in use — and with it, which coordinates every
enrolled codex profile should be observed at.

### 7.2 Pin tests, with the negative variants that give them meaning

Both pins are reproducible with no credential, using the synthetic-root and
observation-shim technique from `TASK-260720-1g880w`. Each needs its negative
variants, because a single positive assertion proves only that the derivation is
*reachable*:

**claude pin**
- Positive: a synthetic empty config dir produces the queried service
  `Claude Code-credentials-<sha256(NFC(dir)).hex[:8]>`, with the digest computed
  *before* the run.
- Negative — different root: a second dir must produce a *different* suffix. A
  pin that passes for two different roots is measuring nothing.
- Negative — NFC is real: an NFD-composed non-ASCII path must hash as its NFC
  form, not its raw bytes. The reviewer's independent run established this
  distinction empirically; a pin that cannot tell the two apart would pass
  against a build that dropped the normalization and orphan every non-ASCII
  root.
- Negative — narrowing, not deleting: truncating the digest to 7 or 9 hex
  characters must fail the pin. A pin that only fails when the whole suffix is
  removed proves the suffix exists and says nothing about its derivation.

**codex pin.** Two branches, and the pin selects the branch from the profile's
recorded `store_selector`, not from the runtime. Under the packaged default the
keyring branch is the one **not** in use, so a pin that ran the keyring
derivation against a `file`-store profile would be asserting against coordinates
nothing populates — vacuously, forever. That is this pin's sharpest negative.

*keyring branch*
- Positive: a synthetic `CODEX_HOME` selects account
  `cli|<sha256(canonical home).hex[:16]>`, digest computed before the run.
- Negative — different home: a second home produces a different account.
- Negative — canonicalization is real: a symlink alias resolving to the same
  home must select the *same* account; if it selects a different one, the
  canonicalization assumption is wrong and P2's byte-verbatim rule must extend
  to codex too.
- Negative — narrowing: 15 or 17 hex characters must fail.

*file branch — the packaged default*
- There is no hash to pin. Disjointness comes from P1 instead: profile ↔ state
  root is a bijection, so the pin asserts that the credential lands at
  `<state_root>/auth.json` and that the profile's coordinates name that path and
  **no** keychain service. Simpler than the keyring branch, not absent.
- Negative — wrong branch: the four keyring assertions above, run against a
  `file`-store profile, must **fail**. A pin that passes on both branches is
  selecting neither.
- Negative — no keyring item appears: with the store resolved to `file`, a
  synthetic home must produce no `Codex Auth` item for its derived account, and
  the lookup must have **succeeded** for that to count. A denied, errored or
  malformed lookup fails the pin; it does not satisfy it (H2). A negative
  assertion satisfied by a failed read asserts nothing, and this one runs on the
  branch where the whole file-store custody argument rests. If an item does
  appear, store selection is not doing what this design assumes and the file
  branch is invalid.
- Negative — outside the root: an `auth.json` under a *different* synthetic home
  must not satisfy this profile's pin. Asserting "some `auth.json` exists"
  measures nothing.
- Negative — store drift: a profile pinned on `file` whose state-root config now
  reads `keyring` must fail the pin (H3), not silently re-derive.

Pins run at enrol and at every launch. `pin_test_id` and `pin_verified_at` are
recorded per profile so a stale pin is visible.

### 7.3 `CLAUDE_SECURESTORAGE_CONFIG_DIR`

This variable repoints the credential namespace *independently of the config
dir*. If it is set, the profile's state root no longer determines where its
credential lives, and every guarantee in this design detaches. Treated as a
refusal condition at enrol and at launch: `E_AUTH_NAMESPACE_OVERRIDDEN`.
agents-infra never sets it.

### 7.4 Widening a range

Deliberate and evidenced, not a config edit: install the new version, run the
full pin test including every negative variant, record the result, extend the
range in the method registry, and re-verify one existing profile's
`observed_coords` against the new build. A range widened without a passing pin
run is the failure mode the gate exists to prevent.

## 8. CLI grammar

### 8.1 Verbs

```
agents-infra auth providers
agents-infra auth methods            [--provider P] [--json]
agents-infra auth enroll             --provider P --alias A --method M
                                     [--identity EMAIL] [--input KEY=VALUE ...]
agents-infra auth list               [--provider P] [--json]
agents-infra auth status             [SELECTOR] [--json]
agents-infra auth logout             SELECTOR [--revoke auto|skip]
agents-infra auth remove             SELECTOR --logout-policy {local-logout|leave-vendor-state} --confirm
agents-infra auth doctor             [SELECTOR]
agents-infra auth pin verify         --provider P [--runtime R]
```

Profile selection at launch is a value, not a new verb:

```
agents-infra claude --auth-profile A
agents-infra codex  --auth-profile A
```

and per project, alongside the existing `[agents.*]` tables:

```toml
[agents.claude.auth]
profile = "work"
```

parsed with the same closed-vocabulary refusal its siblings get: an unknown
alias refuses the launch rather than falling back to the default namespace.

### 8.2 Selector resolution — ambiguity fails safe

A `SELECTOR` is one of:

- `--provider P --alias A` — exact, always unambiguous. **Required for
  `logout` and `remove`**, or an `--identity` that resolves to exactly one
  profile across all providers.
- `--identity EMAIL` — matched against recorded identity claims.
- a bare token — matched against aliases first, then identity claims, across all
  providers.

Resolution rules:

| Matches | Outcome |
| --- | --- |
| exactly 1 | proceed |
| 0 | `E_AUTH_NO_MATCH`, non-zero exit, no action |
| >1 | `E_AUTH_AMBIGUOUS`, non-zero exit, no action, prints every candidate as `provider/alias` |

**Invariant A1.** Ambiguity is resolved before any mutation, and it is never
resolved *by* a heuristic. Not "the only one for this provider", not "the most
recently used", not "the one matching the current project", not "the first".
A destructive verb that guessed which account to log out would be
indistinguishable from working, right up until it was not.

### 8.3 Extensibility

Adding `device-code`, `enterprise-sso`, `api-key` or `coding-plan` is **three**
edits, not two:

1. a registry entry, declaring that the method exists for that provider;
2. an adapter implementing 4.1;
3. **an arm in the custodian-class mapping function (3.2).**

The third is not a formality. A method added without it reaches the function's
fallthrough, `describe()` refuses it, and the method is unusable from its first
call — the safe failure, and the reason 3.2 makes the function total. But a
recipe that names only the first two hands an implementer two thirds of the work
and lets the design's own safety property deliver the remainder as a runtime
refusal. State the edit set instead.

It changes no verb, no selector, no record field, and nothing about the
provider/profile model — because the method is a *value* of `--method`, the
custodian class is a **total function of that value (3.2, C1)** rather than
anything the method declares, and the capability vocabulary already carries
`unsupported` and `unknown` for the operations a new method does not have. A
method that cannot refresh declares `refresh_capability: none`; a method with no
second leg declares `continue: unsupported`; neither needs a new verb to say so.
Neither declaration names a custodian class or a refresh *owner*: both come back
from the class (4.1). Extensibility here means a new method cannot express a new
custody, which is the property that makes adding one cheap to review.

### 8.4 Secret-bearing inputs are structurally kept off argv

`--input KEY=VALUE` is refused when the method registry marks `KEY` as
`secret: true` — `E_AUTH_SECRET_ON_ARGV`, before parsing continues. Secret
inputs arrive only by the vendor's own interactive prompt or by a stdin stream
agents-infra passes through without retaining. This makes S1 (3.4) a property of
the parser rather than a rule in a document: an OTP cannot reach the process
table, the shell history, a log or a board resource, because the flag that would
carry it refuses.

## 9. Named prerequisites

None of this is implementable today. Each item below is a contract change in a
named repository, and each carries the negative test that gives it meaning —
because all three current failures are *silent*, so a positive test would pass
against the broken code.

### 9.1 `skill-agents-management` — the launch plane

| # | Change | Negative test |
| --- | --- | --- |
| **P-AM-1** | `claude`'s and `codex`'s `ChildEnv` must write `HomeEnvVar=<Plan.Home>` into the child environment. Today `claude.ChildEnv` is `agentic.WithRunContext(filterRuntimeEnv(parent), req)` (`pkg/agentic/systems/claude/env.go:112`) and injects nothing runtime-specific; the home is inherited from the parent instead of decided (Invariant L1). | Drive `BuildPlan` with an explicit `Home` and assert the child env carries it; the test must **fail** when the injection is removed, and separately when the plan's `Home` is dropped and the parent's value leaks through. Inheriting the right value by accident must not pass. |
| **P-AM-2** | `Plan.Home` must become the *input* to that injection rather than inert data. `plan.go:176-190` resolves it and no production code reads it. If it does not become load-bearing, it should be removed — an inert field that looks configured is worse than an absent one. | A plan built with `Home` set and a plugin that ignores it must fail a contract test, not pass silently. |
| **P-AM-3** | `providerlimits` needs an explicit-home entry point. `identity.go:112-123` resolves the home via `os.Getenv(capabilities.HomeEnvVar)` — the *parent orchestrator's* environment — so two children under different homes key their rate-limit state under the same identity. | Two launches with distinct homes must produce distinct `IdentityKey`s and distinct on-disk state files. The test must fail if identity is resolved from the process environment. |
| **P-AM-4** | `Capabilities` must expose the supported version range and the namespace derivation id, so the gate (7.1) and the pin (7.2) have a declared source rather than a hardcoded constant. | A capability declaring a range that does not cover the installed binary must refuse the launch. |

### 9.2 `skill-project-management` — config and spawn

| # | Change | Negative test |
| --- | --- | --- |
| **P-PM-1** | `home_env` must be **validated**, not trimmed. `pkg/remoteconfig/spawn_runtimes.go:99` does `strings.TrimSpace` and nothing else, while its siblings `agentic_system` and `broker` get parsing, closed-vocabulary checks and typed refusals. Required: environment-variable-name syntax, agreement with the agentic system's declared `HomeEnvVar`, and refusal on mismatch. | A typo'd `home_env`, and a `home_env` naming a variable the declared system does not use, must both be refused at config parse with a typed error. Today both are accepted in silence, which is strictly worse than being unused, because they look configured. |
| **P-PM-2** | `home_env` must be routed into `LaunchRequest.Home`. Its only non-test references today are the field, the doc comment, the trim, and a read-only copy into `project_config()` at `tools/board-cli/cmd/auth.go:578-579`. | A runtime declaring `home_env` whose spawned child's environment lacks that variable must fail. |
| **P-PM-3** | `internal/spawn` needs **no** change, *provided* P-AM-1 carries the home in `Plan.Env`: `planCommand` already sets `cmd.Env = plan.Env` (`spawn.go:940`). This is recorded so the fix is not made twice, in two places that can later disagree. | Covered by P-AM-1's test at the plan boundary. |

### 9.3 `relux-agents-infra` — the primary-session launcher

| # | Change | Negative test |
| --- | --- | --- |
| **P-AI-1** | `runClaude` and `runCodex` in `tools/agents-infra/main.go` build `exec.Command(...)` and never assign a child environment — verified in this task by reading both functions directly (`main.go:417-472`). The module's seven env-composing sites are all pi launches, and pi declares no home variable, so none of them covers this gap (12, row 1). A primary session therefore always lands in whatever namespace the shell happened to carry. Both launchers must compose an explicit child environment carrying the resolved home. | Launching with `--auth-profile` whose child environment lacks the home variable must fail; and a launch with no profile must set the *default* home explicitly rather than inheriting one. |
| **P-AI-2** | The launch plans (`BuildClaudeLaunchPlan`, `BuildCodexLaunchPlan`) must carry a resolved state root, and `--print-config` must render it, so the selected account is visible before the child starts. | A plan built with a selected profile that renders no state root must fail. |
| **P-AI-3** | `[agents.<runtime>.auth].profile` must parse with the same typed refusal as the other `agents.*` fields (`internal/infra/project_config.go`), and an unknown alias must refuse rather than default. | An unknown alias must refuse the launch. A test asserting it "falls back to default" is asserting the bug. |
| **P-AI-4** | The `auth` verb group itself (8.1), the profile store (3.5), the version gate (7.1) and the pin tests (7.2). | Per 6.4 and 7.2. |

### 9.4 Placement caveat

`~/Library/Application Support/agents-infra/auth/` is chosen because it is
outside every directory `setup.sh` installs, symlinks or rewrites — `~/.agents`,
`~/.claude`, `~/.codex` are all managed and would put enrolled state roots at
risk of being replaced by an install. Because the state root's *path* is the
namespace input, a path that setup can move is a path that can orphan every
profile. This needs confirming against the current installer before it is
committed to (Q7).

## 10. Security invariants, as a checkable list

| Id | Invariant |
| --- | --- |
| H1 | No secret value is ever read, printed, exported, copied or persisted. |
| H2 | A failed or malformed observation is `unknown`, never `absent` and never `active`. |
| H3 | `observed_coords` is re-derived and compared at every launch; drift refuses. |
| C1 | Custodian class is a total function of the method, not a declared field. Native-OAuth methods map to `vendor-opaque`, and no config, flag, registry column or environment input can express otherwise — there is nothing to set (3.2, 6.1). |
| C2 | The class computed from `method` must agree with the recorded `backend`/`store_selector`; a rewritten `method` refuses at the next launch, `E_AUTH_CUSTODY_INCONSISTENT` (3.2). **Checked, not structural** — a consistent two-field rewrite still passes (6.1). |
| S1 | Secret-marked inputs are refused on argv, config and composed environment. |
| P1 | Profile ↔ state root is a bijection; roots are never reused. |
| P2 | The recorded state-root string is passed byte-verbatim, never re-derived. |
| L1 | The home variable is written on every launch, never inherited. |
| L2 | One state root, one enrolment; vendor concurrency never arbitrates two accounts. |
| D1 | `unknown` custody never collapses into `absent` or `active` — including `logout`'s `local` field, where `unknown` and `failed` both refuse `remove` and nothing is deleted (5.4); including 3.6's `retired-local`, which no `unknown` observation reaches and which is the only state `remove` is reachable from; and including 6.2's `active` row and 5.1's step 8, which both require every observation to have succeeded. |
| A1 | Ambiguous selectors refuse; no heuristic ever picks a profile. |
| V1 | Version outside the pinned range, or unreadable, refuses the launch. |
| R1 | agents-infra never deletes a Keychain item and never runs a vendor's revoke on its own. |

## 11. Open questions for `TASK-260720-3gcfd1`

Questions, each with what would settle it. None is answerable from the evidence
now in hand, and none is guessed at above.

| # | Question | What settles it | Cost |
| --- | --- | --- | --- |
| Q1 | Does Anthropic permit two concurrently-enrolled native subscription logins for one human, or does the second invalidate the first server-side? | Enrol a **disposable second** Anthropic account under its own state root, leave the first live, verify both work for 24 h. Namespaces are provably disjoint, so the live account is not at risk. | Needs a second real account. Not answerable synthetically. |
| Q2 | Same question for OpenAI/ChatGPT under `CODEX_HOME` with `cli_auth_credentials_store = "keyring"`. | Same shape, disposable ChatGPT account. | Same. |
| Q3 | What is Claude's actual `revoke_capability`? The design currently reports `unknown`, which means `logout` can never tell an operator their server-side session ended. | Disposable second account: run the vendor logout and observe whether the session is still usable from another namespace. | Free once Q1's account exists. |
| Q4 | Is the codex namespace derivation stable across versions, or is 0.150.1 the only point we have? The range is currently a single version, which will refuse on the next codex upgrade. | Run the codex pin test with its negative variants against every installed codex version, as the reviewer did for claude. | Free, synthetic only. |
| Q5 | Does Claude canonicalize the config dir before NFC-normalizing it, or only normalize? P2's byte-verbatim rule is the safe answer under uncertainty; the answer would let it relax. | Shim run with a symlinked and a trailing-slash spelling of one synthetic dir; compare observed service suffixes. | Free, synthetic only. |
| Q6 | Does Claude's `.storage-write` lock protect against a writer that does not take it? It cannot in principle — it is advisory — but the design's L2 leans on the vendor machinery never arbitrating two accounts, and that assumption is worth failing early. | Two synthetic profiles, one lock-taking writer and one not, both writing the same synthetic item; observe interleaving. | Free, synthetic only. Research proof gate 7. |
| Q7 | Is `~/Library/Application Support/agents-infra/auth/` genuinely outside everything `setup.sh` manages, on macOS and on the Windows path? A state root that setup can move orphans every profile. | Read the installer's write set; assert the auth root is disjoint from it, with a test that fails if setup grows a write into it. | Free. |
| Q8 | Are profile records machine-scoped (as designed) or may a project pin an alias that does not exist on another machine? A project config referencing an unknown alias currently refuses — correct, but it makes a shared repo config machine-specific. | Operator/product decision, informed by whether these configs are shared across machines. | Decision, not experiment. |
| Q9 | Is a `degraded:plaintext` profile repairable in place, or must it be re-enrolled? Current source says a later Keychain success deletes the plaintext file, which suggests self-healing — but the Keychain item was already deleted, so what heals is unclear. | Synthetic profile, induce a non-transient store failure, then remove the cause and observe. Requires a credential-shaped payload, so it needs a disposable account (blocked with Q1). | Blocked on Q1. |
| Q10 | Does qwen get a state root at all? Its plugin declares `HomeEnvVar: ""`, so `vendor-opaque` has nothing to attach to and the epic's qwen ambition has no mechanism. | Audit the qwen CLI for a state-root environment variable and a credential store, as `TASK-260720-3moaky` did for the other two. | Free, a source audit. |
| Q11 | Is refusing on `CLAUDE_SECURESTORAGE_CONFIG_DIR` (7.3) right, or does some legitimate operator setup set it? | Survey the operator environment and the vendor's docs; if legitimate, honour it as an explicit override with its own recorded coordinate. | Free. |
| Q12 | What is codex's failure posture on the `keyring` store? The research proved `auto` falls back to a plaintext `auth.json` and left `keyring` unestablished, so 6.2 classifies plaintext-beside-keyring as `degraded:plaintext` rather than guess. If keyring also fails open, codex inherits Claude's fail-open hazard and 5.3's prohibitions become load-bearing on that branch too; if it fails closed, the row can be relaxed. | Synthetic profile with `cli_auth_credentials_store = "keyring"`; induce a non-transient keyring write failure and observe whether an `auth.json` appears. Needs a credential-shaped payload, so it needs the disposable ChatGPT account from Q2. | Blocked on Q2. |
| Q13 | Is the profile record worth cryptographically sealing? C2 (3.2) turns a `method` forgery into a two-field consistent rewrite, and stops nothing that can write both fields. A seal over the immutable fields, verified at launch, would — at the cost of a key living on the same machine as the record it protects. | A threat-model decision: does the model include local write access to `~/Library/Application Support/agents-infra/auth/`? If it does, C2 is insufficient and the seal is required, with its key custody a question of its own. If it does not, C2 is the right stopping point and the seal is ceremony. | Decision, not experiment. |

## 12. What this task verified itself

Not taken from the research on trust — re-derived here, because every one of
these is load-bearing for a prerequisite in section 9.

| # | Claim | How | Result |
| --- | --- | --- | --- |
| 1 | `runClaude`/`runCodex` never assign a child environment | Read `main.go:417-472` directly, then enumerated every `<recv>.Env =` assignment in the module, non-test — not a `cmd.Env` literal grep | Neither launcher assigns `cmd.Env`: both build `exec.Command`, set Stdin/Stdout/Stderr and run. The module has **seven** env-composing sites — `pi_shared_broker_darwin.go:384`, `pi_shared_client_darwin.go:516,693`, `pi_launch_posix.go:223,306,642`, `pi_platform_windows.go:140` — and all seven are pi launches, where `HomeEnvVar` is `""` (row 7), so L1 does not reach them and P-AI-1 has no scope gap. A literal `grep "cmd.Env"` finds only two of the seven: it is case-sensitive and matches neither `command.Env` nor `piCmd.Env`/`runtimeCmd.Env`. Reporting that narrow grep as a module-wide enumeration was wrong and is corrected here. **New finding**, not in the research; it is P-AI-1. |
| 2 | claude's `ChildEnv` injects nothing runtime-specific | `pkg/agentic/systems/claude/env.go:112` | `agentic.WithRunContext(filterRuntimeEnv(parent), req)`; `runtimeEnvKeys` is `["CLAUDECODE"]`. The child inherits the parent's `CLAUDE_CONFIG_DIR` — inheritance, not defaulting. Sharpens the research's "not injected" into Invariant L1. |
| 3 | `planCommand` already carries `Plan.Env` | `skill-project-management/tools/board-cli/internal/spawn/spawn.go:940` | `cmd.Env = plan.Env`. So P-PM-3 is a no-op *if* the home is written into `Env` — the fix belongs in one place, not two. |
| 4 | `Plan.Home` is still resolved and still unread | `skill-agents-management/pkg/agentic/plan.go:176-190` | Resolved from `req.Home`/`caps.DefaultHome`, returned in the plan, read by nothing. |
| 5 | `providerlimits` still reads the parent environment | `pkg/providerlimits/identity.go:115-118` | `os.Getenv(capabilities.HomeEnvVar)`. |
| 6 | `home_env` is still trimmed, not validated | `pkg/remoteconfig/spawn_runtimes.go:39,99` | Field declared, `strings.TrimSpace` applied, siblings parsed and refused with typed errors. |
| 7 | qwen declares no home variable | `pkg/agentic/systems/qwen/qwen.go:121` | `HomeEnvVar: ""`. Also muse, gemini, agy, pi. |
