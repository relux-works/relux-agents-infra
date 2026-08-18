# Pi Project-Local Model Launch Contract

Task: `TASK-260817-2h8hn4`
Story: `STORY-260817-1on8ex`
Decision revision: cycle 10 collision-resistant profile state keys under the cycle 8 practical trust boundary

## 1. Decision summary

`agents-infra pi` shall add Pi as a project-aware primary-session launcher without reading or writing the user's normal `~/.pi/agent` tree for managed profiles.

When a project selects a managed local profile, agents-infra shall:

1. compose `.agents/.configs/project-config.toml` files from filesystem root to the current working directory;
2. select one complete named profile by nearest explicit `agents.pi.primary_session.profile`, or by wrapper `--profile` override;
3. validate the complete selected profile and Pi arguments before any managed side effect;
4. verify the exact supported standalone Pi execution closure and parser contract selected by `pi_compatibility`;
5. treat the configured absolute `runtime.executable` plus literal `runtime.argv` as reviewed executable policy that authorizes that local program to run;
6. refuse a non-loopback endpoint and refuse before spawn when the configured listener is already occupied;
7. derive a contained profile-state directory from collision-resistant hashes rather than raw profile text, atomically generate one isolated Pi `models.json`, start the runtime as a direct child in a new owned process group without a shell, wait for direct-child liveness plus exact `/v1/models` discovery, and then start Pi;
8. invoke the verified Pi executable by absolute path with the managed exact provider/model identity and a production-faithful argv bridge;
9. forward signals, terminate and reap the runtime process group when Pi exits, and never intentionally attach to an existing listener or silently fall back to another profile, port, runtime, model, or target-only mode.

The launcher does not claim to prove that a trusted runtime executable is internally honest. Qwen `text`/`tools` and Muse `text`/`tools`/`dflash` are requested/configured capabilities, not independently verified facts. Runtime responses may be reported only as unverified diagnostic provenance. For Muse, launch admission requires the exact configured target/draft argv, a live selected runtime child, and the exact target in `/v1/models`; Pi smoke and benchmark evidence remain explicit operator verification steps.

`agents-infra pi --print-config` and `agents-infra compose --mode primary-session --agent pi ...` run the same resolution and static validation but start no process, create no file, acquire no lock, bind/connect to no socket, and mutate no Pi trust/settings/auth state.

With genuine policy absence, `agents-infra pi` is native passthrough. An unreadable, partial, or malformed policy is not absence and fails closed.

## 2. Requirements traceability

| Contract section | Requirement |
| --- | --- |
| Sections 3-4 | Official Pi behavior; exact nearest-project TOML; Qwen and Muse profile inputs |
| Section 5 | Precedence; explicit CLI overrides; exact managed identity; production-faithful argv bridge |
| Section 6 | Non-launching launch-plan diagnostics |
| Sections 7-8 | Process ownership/lifecycle; Qwen text/tools; Muse plus DFlash practical acceptance |
| Sections 9-11 | Security boundary; failure semantics; rejected alternatives |
| Section 12 | Executable positive, negative, and narrowing acceptance scenarios |
| Section 13 | Smallest development-ready board decomposition, dependencies, and gap verification |

## 3. Verified Pi behavior

The three task-linked official documents were re-read from redirected official repository revision `10acee6045e9025a22dff7e5220ed0d7538f12aa` on 2026-08-17. SHA-256 values are:

| Document | SHA-256 |
| --- | --- |
| `models.md` | `3ab68dd46af081d3a11a2d705048f2fbde93a87e29891c677191e24cec2840f3` |
| `settings.md` | `f36d3a918d87d18e13d22ca98f4e429428b5f2bc06316ff7d2e7adace59a973b` |
| `usage.md` | `a6a76e733c50ea8a08701456858d3937e1d60caa8709a7f15125e6f5ba6cabca` |

Verified behavior:

1. Custom providers/models are configured in `~/.pi/agent/models.json`. A local OpenAI-compatible provider uses `baseUrl`, `api: "openai-completions"`, a dummy `apiKey`, and model entries. Without configured auth, custom models load but remain unavailable in `/model` and `--list-models`.
2. Model entries support `id`, `name`, `api`, `reasoning`, `thinkingLevelMap`, `input`, `contextWindow`, `maxTokens`, `samplingParams`, `cost`, and `compat`. OpenAI compatibility includes developer-role, reasoning-effort, streaming-usage, finish-reason, token-field, tool-result, strict-tool, and thinking-format controls.
3. `models.json` `apiKey` and header values may execute shell commands or interpolate environment variables. The generated managed catalog deliberately uses neither surface and contains a fixed non-secret dummy key.
4. Pi settings are global `~/.pi/agent/settings.json` plus current-directory `.pi/settings.json`; project settings override global settings and nested objects merge.
5. Pi project resources are trust-gated. Interactive Pi may prompt. Non-interactive modes use saved/default trust unless `--approve` or `--no-approve` overrides one run. The wrapper never injects either flag or persists a trust decision.
6. Native model controls are `--provider`, `--model`, `--api-key`, `--thinking`, `--models`, and `--list-models`. `--model` accepts a provider-qualified selector and optional thinking suffix; `--api-key` overrides environment credentials.
7. Pi documents `--session-dir`; current official source derives `PI_CODING_AGENT_DIR` and `PI_CODING_AGENT_SESSION_DIR` and resolves the agent directory before the default `~/.pi/agent`. Managed launch uses those narrow paths rather than overriding `HOME`.
8. At the verified source revision, production `parseArgs()` has no end-of-options state. Literal `--` enters the unknown-long-flag branch, can consume a following non-option token, and does not stop later flags from being parsed. Recognized model/provider/value options use spaced forms; a recognized-looking `--flag=value` reaches the unknown-extension branch.
9. Pi `v0.84.2` provides an official darwin-arm64 standalone asset and release-owned checksums. The retained managed Pi compatibility gate binds launch to that exact standalone release tree and parser grammar; it does not trust a shared version string or an npm/shebang entrypoint.

Official references:

- https://github.com/badlogic/pi-mono/blob/main/packages/coding-agent/docs/models.md
- https://github.com/badlogic/pi-mono/blob/main/packages/coding-agent/docs/settings.md
- https://github.com/badlogic/pi-mono/blob/main/packages/coding-agent/docs/usage.md
- Supplemental official source at the verified revision: `packages/coding-agent/src/cli/args.ts` and `packages/coding-agent/src/config.ts`
- Official standalone release: https://github.com/earendil-works/pi/releases/tag/v0.84.2

## 4. Exact project TOML contract

The schema extends `.agents/.configs/project-config.toml`:

```toml
[agents.pi.primary_session]
profile = "qwen-3.8-27b"
pi_compatibility = "github-release:earendil-works/pi@v0.84.2:darwin-arm64#sha256-c996e888b7f7dce44bcf24f69176ac646c44139d3916bd49a6b28e5a8c5e3a65"

[agents.pi.profiles."qwen-3.8-27b"]
provider = "local-qwen"
model = "Qwen-3.8-27B"
base_url = "http://127.0.0.1:18011/v1"
api = "openai-completions"
reasoning = false
input = ["text"]
context_window = 131072
max_tokens = 16384
thinking = "off"
requested_capabilities = ["text", "tools"]

[agents.pi.profiles."qwen-3.8-27b".compat]
supports_developer_role = false
supports_reasoning_effort = false
supports_usage_in_streaming = true
supports_finish_reason = true
max_tokens_field = "max_tokens"
thinking_format = "qwen-chat-template"

[agents.pi.profiles."qwen-3.8-27b".runtime]
executable = "/absolute/path/to/reviewed-runtime"
argv = ["serve", "--model", "Qwen-3.8-27B", "--host", "127.0.0.1", "--port", "18011"]
readiness_path = "/models"
startup_timeout_seconds = 120
shutdown_timeout_seconds = 10

[agents.pi.profiles."muse-glimmer-30b-dflash"]
provider = "local-muse"
model = "Muse-Glimmer-30B"
base_url = "http://127.0.0.1:18012/v1"
api = "openai-completions"
reasoning = false
input = ["text"]
context_window = 131072
max_tokens = 16384
thinking = "off"
requested_capabilities = ["dflash", "text", "tools"]

[agents.pi.profiles."muse-glimmer-30b-dflash".compat]
supports_developer_role = false
supports_reasoning_effort = false
supports_usage_in_streaming = true
supports_finish_reason = true
max_tokens_field = "max_tokens"

[agents.pi.profiles."muse-glimmer-30b-dflash".runtime]
executable = "/absolute/path/to/reviewed-runtime"
argv = ["serve", "--model", "Muse-Glimmer-30B", "--draft", "Muse-Glimmer-30B-DFlash", "--host", "127.0.0.1", "--port", "18012"]
readiness_path = "/models"
startup_timeout_seconds = 180
shutdown_timeout_seconds = 10

[agents.pi.profiles."muse-glimmer-30b-dflash".runtime.dflash]
target_model = "Muse-Glimmer-30B"
draft_model = "Muse-Glimmer-30B-DFlash"
target_argv = ["--model", "Muse-Glimmer-30B"]
draft_argv = ["--draft", "Muse-Glimmer-30B-DFlash"]
```

The model/runtime names and executable paths above are exact operator-supplied deployment inputs, not shipped artifact coordinates. This story does not guess downloads, quantization, conversion, licensing, runtime-specific flags, or hardware sizing.

### 4.1 Composition and validation

- `primary_session` accepts only optional non-empty `profile` and `pi_compatibility`. A selected managed profile requires both after composition.
- Profile names are exact non-empty TOML string keys after TOML escape decoding. Their UTF-8 bytes are the logical identity: no Unicode normalization, case folding, path cleaning, or lossy sanitization participates in lookup or state-key derivation. A child definition of the same exact decoded name atomically replaces the ancestor definition; profile fields never merge across files. A child may still select a complete ancestor profile by exact name.
- Every shown profile/runtime field is required except individual `compat` members and `runtime.dflash`, which is forbidden for Qwen and required for Muse. Unknown fields fail closed.
- `api` is exactly `openai-completions`; `input` is exactly `["text"]`; numeric limits/timeouts are positive; `max_tokens <= context_window`.
- `requested_capabilities` is exactly `['text','tools']` for ordinary managed profiles or byte-sorted `['dflash','text','tools']` when `runtime.dflash` is present. These are reported as `requested`, never `verified`.
- `provider` is non-empty and excludes ASCII `/`, `:`, whitespace, Pi glob characters, and Unicode separator lookalikes. `model` is preserved byte-for-byte. Together they define the only generated Pi identity.
- `base_url` must be `http`, host exactly `127.0.0.1`, no user info/query/fragment, path exactly `/v1`, and a non-zero explicit port. `localhost`, IPv6 loopback, wildcard, non-loopback, and URL normalization are refused.
- `runtime.executable` must be a non-empty absolute path with no NUL. Static diagnostics distinguish absent, non-regular, non-executable, unreadable, and present; launch requires a present executable. It is never resolved through `PATH`, `env`, a package manager, or a shell.
- `runtime.argv` is a non-empty literal string array and must contain exactly one spaced `--host 127.0.0.1` pair plus exactly one spaced `--port <base_url-port>` pair. Missing, duplicate, attached, wildcard, or divergent endpoint options fail. Accepted tokens are passed byte-for-byte after `runtime.executable`; empty tokens and NUL fail. No interpolation, globbing, tilde expansion, command substitution, quoting layer, or implicit flag injection occurs.
- Selecting `runtime.executable` and its literal argv is explicit trusted executable policy. agents-infra validates and reproduces the launch, but does not inspect or certify runtime semantics.
- `readiness_path` is exactly `/models`, yielding exact `<base_url>/models`. A successful response must have the OpenAI list shape and contain an item whose `id` is byte-equal to the configured profile `model` for Qwen or `runtime.dflash.target_model` for Muse.
- Muse `target_argv` and `draft_argv` are each non-empty literal token arrays and must occur exactly once as contiguous subsequences of `runtime.argv`; their last token must equal `target_model` and `draft_model` respectively. They may not overlap. This proves only which exact argv was launched.
- Project TOML has no adapter, observer, backend-catalog, digest, attestation endpoint, authority key, fallback, attach, shell, environment, or working-directory fields.
- The generated `models.json` contains one provider and one model with fixed dummy key `agents-infra-local`, zero cost, configured metadata, and mapped compatibility. It contains no secret, header, command, or environment reference.

#### 4.1.1 Profile state-key and containment contract

Raw profile names are configuration identities only and are never filesystem path components. For the selected decoded profile name, let `profile_bytes` be its exact UTF-8 encoding and let `profile_key = lowercase_hex(SHA256(profile_bytes))`, exactly 64 ASCII characters in `[0-9a-f]`. The launcher performs no normalization, case folding, separator replacement, trimming, or path cleaning before hashing. Thus `qwen`, `./qwen`, `nested/../qwen`, case variants, NFC/NFD variants, `/`, `\\`, `.`, `..`, absolute-looking strings, and Unicode separator/lookalike strings remain distinct logical names and distinct state keys unless their exact UTF-8 bytes are equal. A decoded duplicate name is one TOML key and ordinary duplicate-key handling applies.

The project key is likewise `lowercase_hex(SHA256(UTF8(canonical_project_path)))`. The intended profile root is exactly `<canonical-cache-root>/agents-infra/pi/<project_key>/<profile_key>`, where `canonical-cache-root` is the successfully resolved absolute `os.UserCacheDir()`. Before lock acquisition, directory creation, file creation, or process start, the production entry point must:

1. compute every effective profile key and refuse `profile_state_key_collision` if two byte-distinct effective names produce the same key;
2. prove lexically that the intended root is the exact four-component relative suffix `agents-infra/pi/<project_key>/<profile_key>` beneath the canonical cache root, with both keys matching `[0-9a-f]{64}` and no absolute or `..` result;
3. walk/create the managed suffix from an opened canonical-cache-root directory handle, never following symlinks in managed components; every existing component must be the expected directory, and each created component must be reopened and revalidated through that anchored handle before use; and
4. treat any cache-root lookup, canonicalization, stat/open, containment, collision, symlink, type, permission, partial-read, or revalidation failure as `profile_state_path_invalid`, not as absence and not as permission to fall back.

Diagnostics may show the exact logical profile name and its provenance, but all managed paths use only the two hashes. They report `project_state_key`, `profile_state_key`, `canonical_cache_root`, and the derived contained paths. No raw profile name is interpolated into a path. Lock identity is therefore exact `(canonical_project_path bytes, profile name bytes)`, and distinct accepted names never intentionally share `models.json`, sessions, or `session.lock`.

### 4.2 Retained managed Pi compatibility identity

The cycle-8 runtime simplification does not weaken the wrapper's Pi parser boundary. Managed launch remains supported only for an agents-infra-catalogued official standalone Pi execution closure selected by `pi_compatibility`.

Initial production entry:

| Field | Exact value |
| --- | --- |
| ID | `github-release:earendil-works/pi@v0.84.2:darwin-arm64#sha256-c996e888b7f7dce44bcf24f69176ac646c44139d3916bd49a6b28e5a8c5e3a65` |
| Asset | Official `pi-darwin-arm64.tar.gz` at `v0.84.2` |
| Asset SHA-256 | `c996e888b7f7dce44bcf24f69176ac646c44139d3916bd49a6b28e5a8c5e3a65` |
| Host | `darwin/arm64` |
| Regular-file count | `217` |
| Canonical tree-manifest SHA-256 | `2f68ab1b3f28a9c4b8995f91984f8f47001a79735da7e57aa7fe6d223f90378b` |
| Authoritative manifest | `TASK-260817-2h8hn4_pi-v0.84.2-darwin-arm64-tree-manifest.txt` |
| Entrypoint | `<release-root>/pi`, native arm64 Mach-O |
| Entrypoint SHA-256 | `d5de3fe32f9e109324f32d6e393554fb2ce10bbc82e8ff935ab2e072f5e2f044` |
| Parser contract | Pinned option arities, unknown-flag consumption, no separator, equal-form behavior |

The catalog input is exactly the release asset identified above. Its archive must contain exactly one top-level `pi/` directory and no peer entry. For an installed launch, `<release-root>` is the canonical parent directory of the fully resolved standalone entrypoint `<release-root>/pi`; the installed directory name itself is not trusted as identity. The authoritative manifest attachment is generated relative to that root from every regular file, with no exclusions, by these exact rules:

1. Validate names as UTF-8 and preserve their encoded bytes exactly. Reject NUL, CR, LF, backslash, empty, `.`, and `..` path components; absolute paths; path escape; Unicode normalization/case folding; and any observed name that is not byte-equal to its catalog path.
2. Sort slash-separated relative file paths by unsigned byte value in `C`-locale order. Do not use directory enumeration order or locale-sensitive collation.
3. Encode each record as `<64 lowercase hexadecimal SHA-256 bytes><two ASCII spaces>./<relative-path><LF>`, with no BOM, header, trailer, or final omission of `LF`. Hash the concatenated 217 records with SHA-256.
4. Treat the 217 attached file paths as exhaustive. The only permitted directories are `<release-root>` and the exact proper-prefix closure of those file paths: 34 directories including the root. A missing or extra file or directory fails identity.
5. Require every catalogued file to be a regular file with link count exactly one. Reject every symbolic link, hard-link alias, socket, FIFO, device, mount crossing, or other entry type at or below the root. No symlink may participate in the canonical root or entrypoint after resolution.
6. Require POSIX permission bits exactly `0755` for every directory and for `./pi`, `./examples/extensions/doom-overlay/doom/build.sh`, `./examples/extensions/doom-overlay/doom/build/doom.wasm`, and `./native/darwin/prebuilds/darwin-arm64/darwin-modifiers.node`; require `0644` for the other 213 regular files. Ownership, timestamps, ACLs, and extended attributes are not catalog identity inputs, but they cannot excuse an unreadable tree or a non-executable entrypoint.
7. Require `./pi` content SHA-256 and native arm64 Mach-O kind as recorded above in addition to the full tree comparison. The release asset checksum, entrypoint hash, file count, manifest bytes/digest, exact path inventory, entry types/link counts, and permission map are one indivisible compiled catalog entry; a project supplies only its catalog ID and cannot replace any member.

Before any managed side effect, the launcher resolves `pi` once, canonicalizes the link chain, verifies every field of the complete compiled-catalog release tree, captures the canonical root and entrypoint identity, and rejects absent, malformed, unreadable/unknown, unsupported, or mismatched results. It admits no npm/shebang, Node/Bun host, wrapper, copied standalone binary without its exact tree, project-supplied digest, or regular-files-only digest approximation. Immediately before Pi spawn, after runtime readiness, it repeats the same exhaustive verification from the canonical root and requires the same canonical root/entrypoint identity plus catalog bytes, paths, types, links, and modes. A failed, partial, or raced re-read is `pi_execution_identity_changed`, never absence or success.

Managed launch also rejects duplicate environment names and loader-affecting names in the catalogued deny set (`DYLD_*`, `LD_*`, `NODE_*`, `BUN_*`, including exact Node/Bun configuration names) before side effects. It then sets isolated Pi paths plus `PI_SKIP_VERSION_CHECK=1` and `PI_TELEMETRY=0`. It invokes only the captured canonical standalone entrypoint by absolute path. Native passthrough is outside this gate.

## 5. Composition, precedence, and argv bridge

Selection precedence:

| Priority | Selection | Behavior |
| ---: | --- | --- |
| 1 | Wrapper `agents-infra pi --profile NAME` before wrapper delimiter | Selects one named profile; conflicting repeats fail, equal repeats normalize |
| 2 | Nearest explicit `agents.pi.primary_session.profile` | Nearest root-to-cwd field wins |
| 3 | Genuine absence | Native Pi passthrough; no runtime or isolated state |

`pi_compatibility` has nearest-field precedence and no CLI override. No CLI option overrides `runtime.executable`, runtime argv, endpoint, requested capabilities, target, or draft.

Within a managed profile:

- `--provider VALUE` is accepted only when byte-equal to configured `provider`; otherwise `managed_profile_identity_mismatch`.
- `--model VALUE` is accepted only when it resolves to the configured exact provider/model identity. Accepted forms are exact `model` or ASCII `provider/model`, optionally followed by an ASCII `:` plus a documented thinking level. Globs, other IDs, Unicode slash/colon lookalikes, case folding, or fuzzy matching never select a managed identity.
- A provider embedded in `--model` and an explicit `--provider` must both equal configured `provider`.
- `--thinking VALUE` overrides configured thinking; an unequal model-suffix level conflicts and fails.
- `--api-key VALUE` is preserved and wins by Pi's native contract; its value is always redacted from diagnostics.
- `--approve` and `--no-approve` are preserved. The wrapper injects neither.

The wrapper parses the full pre-delimiter argv with the pinned Pi grammar. The first ASCII `--` is a wrapper-only operand delimiter and is never forwarded because Pi does not implement end-of-options. A second delimiter fails. Every suffix token is an intended Pi message operand and is appended byte-for-byte only if it begins with neither ASCII `-` nor `@`; unsafe tokens fail before runtime start. The wrapper never prefixes, quotes, joins, or otherwise changes message bytes.

A value-taking prefix option cannot consume from the suffix. Unknown long flags are accepted only as self-contained `--name=value` or as a complete Pi-style flag/value pair entirely before the delimiter; a bare unknown flag adjacent to suffix operands is boundary-ambiguous and fails.

For wrapper-recognized `--profile`, `--provider`, `--model`, `--thinking`, and `--api-key`, both spaced and `--flag=value` forms are wrapper syntax. `--profile` is removed; other equal forms normalize to Pi's spaced form and pass the same validation/redaction. Unsupported known Pi `--flag=value` forms fail rather than becoming extension flags.

The final Pi argv contains no fake separator, appends no option after operand content, preserves accepted operand bytes/order, and contains exactly one effective managed provider/model selection. Inbound `PI_CODING_AGENT_DIR` or `PI_CODING_AGENT_SESSION_DIR` fails for managed profiles and remains untouched for native passthrough.

## 6. Non-launching diagnostics

Pi uses `agents-infra.primary-session-launch-plan` schema version 1. `provider` is `pi`; `launch_variants.interactive.argv` is the exact normalized Pi argv; `managed_host.kind` is `pi-pty` with the same argv; `managed_client.argv` is empty.

The plan reports:

- `resolved.profile`, `resolved.model`, `resolved.reasoning`, and `resolved.pi_compatibility` with exact source provenance;
- root-to-cwd `sources` for every successfully read project config;
- generated `models.json` path and canonical-content SHA-256 without writing it;
- the retained Pi compatibility catalog identity, statically observed state, and point-of-use recheck requirement;
- `sidecars.local_model.ownership = "direct-child-process-group"`;
- exact configured runtime executable/argv with source path, endpoint/readiness URL, timeouts, lock/state paths, and static executable state;
- exact logical profile name plus `project_state_key` and `profile_state_key`; every reported lock/state path is the successfully validated hash-only contained path, never a raw-name-derived path;
- `capabilities.requested` equal to configured capability labels;
- `capabilities.verified = []` and `capabilities.verification = "not-claimed"`;
- for Muse, exact configured target/draft and their exact argv subsequences, with `dflash.status = "configured-unverified"`;
- runtime-reported facts, if later exposed by an implementation, only beneath `runtime_reported` with `trust = "unverified"`; they never populate `verified` fields or authorize launch;
- only environment variable names and non-secret managed paths; secrets and arbitrary environment values never appear.

`--print-config` and compose do not statically convert an inability to inspect into absence. Failed/partial reads are `unknown` or an error. They never run Pi/runtime, bind/connect, create a lock/file/pipe/nonce, download anything, or claim listener ownership, tool calling, or active DFlash.

## 7. Process ownership and lifecycle

1. Resolve project, selected profile, normalized Pi argv, exact Pi identity, exact runtime executable/argv, endpoint, and isolated paths.
2. Perform all static validation and retained Pi compatibility checks. Stop before side effects on any error.
3. Resolve the canonical cache root and derive state exactly as section 4.1.1: `<canonical-cache-root>/agents-infra/pi/<project_key>/<profile_key>/`, containing `agent/`, `sessions/`, and `session.lock`. Prove containment, collision freedom, managed-component no-follow behavior, and complete reads before any managed side effect.
4. Acquire `session.lock` non-blocking. A second launch for the same exact project/profile byte identity fails `pi_profile_busy`; byte-distinct profile names have distinct state keys and independent locks.
5. Atomically replace only managed `agent/models.json` with mode `0600`. Preserve profile-local settings/trust/sessions/auth and never copy from `~/.pi/agent`.
6. Preflight the exact `127.0.0.1:<port>` by attempting an exclusive bind. If occupied, refuse without connecting and before runtime spawn. Close the probe socket before spawn because the arbitrary trusted runtime must bind its own listener.
7. Recheck runtime path presence/executability, then spawn `[runtime.executable] + runtime.argv` directly with no shell, interpolation, implicit environment additions, or `PATH` lookup. Make it the leader of a new process group and retain its direct-child handle.
8. During startup, require the direct child to remain alive and poll only the configured loopback readiness URL. A connection failure is retryable until timeout. A successful malformed response, missing list, or absence of the exact expected model is hard failure.
9. Readiness is discovery, not ownership or capability proof. Preflight plus child liveness does not cryptographically bind the listener to the child. The contract intentionally excludes a malicious selected runtime and a malicious same-UID process that wins the post-preflight bind race.
10. For Muse, independently validate before spawn that the exact configured target/draft subsequences occur once in literal runtime argv. At readiness, require the exact target ID. Do not inspect, infer, or claim DFlash active state.
11. Reverify Pi identity/environment and spawn only the canonical standalone Pi path with normalized argv and isolated Pi directories.
12. If the runtime direct child exits at any time, terminate Pi if started, terminate/reap the runtime group, and return non-zero. A ready foreign listener does not compensate for a dead selected child.
13. On SIGINT/SIGTERM, forward to Pi first, then terminate the runtime process group. On Pi normal exit or failed spawn, terminate/reap the runtime group. Escalate to kill after `shutdown_timeout_seconds`.
14. Release the lock only after Pi and every owned runtime-group process are reaped. No error path intentionally attaches, leaves a daemon, or falls back.

## 8. Product profile requirements

### Qwen 3.8 27B text/tool-calling

- Pi receives one text-only OpenAI Chat Completions model with exact operator-supplied ID.
- The generated Pi model disables incompatible developer-role/reasoning-effort behavior as configured and may use Pi's documented Qwen thinking mapping. The reference profile is non-reasoning/off.
- `requested_capabilities = ["text", "tools"]` is a request/configuration label. The launcher verifies only runtime child liveness and exact model discovery.
- Operator acceptance drives Pi through one real text response and one real function-tool call/result round trip against the selected runtime. This evidence verifies the deployment, not a reusable launcher trust root.

### Muse Glimmer 30B plus DFlash

- Pi sees one text/tool OpenAI Chat Completions target. Pi receives no draft-model configuration.
- The launcher reproduces the exact configured target/draft argv subsequences, requires the selected runtime child alive, and requires the exact target in readiness.
- `requested_capabilities = ["dflash", "text", "tools"]` and `dflash.status = "configured-unverified"` never mean independently verified active DFlash.
- No silent target-only fallback is added by agents-infra. If the trusted runtime silently disables DFlash while keeping the child alive and target ready, the launcher cannot detect it without an authoritative runtime API and does not claim otherwise.
- Operator acceptance runs a Pi text/tool smoke plus a runtime-appropriate benchmark/telemetry check that distinguishes target-only from DFlash. Automation of that smoke/benchmark and interpretation of runtime-specific telemetry are out of story scope.

## 9. Security and threat boundary

In scope:

- exact config provenance and fail-closed parsing;
- exact managed Pi execution-closure/parser gate;
- reviewed absolute runtime executable plus literal argv policy;
- direct argv-only spawn without shell/interpolation/PATH lookup;
- exact loopback endpoint and occupied-listener preflight refusal;
- direct-child liveness, exact model discovery, owned runtime process group, cleanup, and no intentional attach/fallback;
- isolated Pi agent/session state and secret-redacted diagnostics;
- negative tests that drive the real launcher and narrow each gate.

Explicitly out of threat model:

- a malicious or compromised `runtime.executable` selected by reviewed project policy;
- false internal claims, proxying, or silent DFlash disablement performed by that trusted runtime;
- a malicious same-UID process winning the bind race after preflight closes its socket and before the runtime binds;
- compromised kernel, platform libraries, or same-UID mutation outside the retained deterministic Pi point-of-use checks.

The launcher does not call preflight plus readiness listener attestation, cryptographic ownership, or capability verification. Runtime status, logs, throughput, argv, `/v1/models`, and child PID may be useful provenance but are not an independent authority root.

## 10. Failure semantics

| Code | Condition | Managed processes started? |
| --- | --- | --- |
| `invalid_project_configuration` | Unreadable/malformed/partial TOML, unknown field, invalid type/domain, unsafe endpoint, invalid runtime/DFlash argv relation | No |
| `invalid_provider_arguments` | Missing/conflicting/invalid recognized Pi argument | No |
| `unsafe_pi_operand` | Fake/repeated separator, unsafe suffix, or ambiguous cross-boundary consumption | No |
| `managed_profile_identity_mismatch` | Explicit provider/model is not the selected exact generated identity | No |
| `unknown_pi_profile` | Selected profile absent after composition | No |
| `profile_state_key_collision` | Byte-distinct effective profile names derive the same SHA-256 state key | No |
| `profile_state_path_invalid` | Cache root or managed path cannot be completely resolved, contained, opened without following managed symlinks, or revalidated | No |
| `provider_executable_not_found` | Pi executable absent | No |
| `pi_compatibility_unsupported` | No compiled Pi entry for selected ID/host | No |
| `pi_execution_identity_unavailable` | Pi closure cannot be read completely | No |
| `pi_execution_identity_malformed` | Pi tree cannot form the catalogued shape | No |
| `pi_execution_identity_mismatch` | Complete observed Pi tree differs | No |
| `pi_execution_environment_invalid` | Duplicate or denied loader-affecting environment name | No |
| `runtime_executable_not_found` | Configured absolute runtime path absent | No |
| `runtime_executable_invalid` | Runtime path is relative, unreadable, non-regular, non-executable, or changes before spawn | No |
| `pi_profile_busy` | Project/profile lock already held | No |
| `runtime_listener_occupied` | Exact loopback bind preflight reports address in use | No |
| `runtime_listener_check_failed` | Preflight cannot determine vacancy; never treated as vacant | No |
| `runtime_start_failed` | Direct spawn fails | No owned process remains; Pi not started |
| `runtime_exited_early` | Direct child exits before or during Pi session | Runtime group reaped; Pi not started or terminated |
| `runtime_readiness_timeout` | Exact endpoint never becomes ready | Runtime group terminated; Pi not started |
| `runtime_readiness_invalid` | Successful response is malformed/partial | Runtime group terminated; Pi not started |
| `runtime_model_unavailable` | Exact configured target absent | Runtime group terminated; Pi not started |
| `pi_execution_identity_changed` | Pi point-of-use recheck changes or fails | Runtime group terminated; Pi not started |
| `pi_start_failed` | Pi cannot spawn after readiness | Runtime group terminated |
| `runtime_shutdown_timeout` | Group ignores graceful termination | Group killed; launcher non-zero |

No selected managed profile falls back to native Pi, another runtime/profile/port, an occupied listener, or target-only decoding.

There are deliberately no `runtime_attestation_*`, backend-catalog, observer, adapter, or DFlash-authority errors in cycle 8. Such gates would promise evidence agents-infra does not possess and would prevent the requested profiles from launching.

## 11. Rejected alternatives

| Alternative | Reason rejected |
| --- | --- |
| Write/merge `~/.pi/agent/models.json` | Mutates global state, races user edits, and exposes executable credential-value surfaces |
| Put custom models only in `.pi/settings.json` | Official custom provider/model registration belongs in `models.json` |
| Override `HOME` | Redirects unrelated tools/credentials; Pi exposes narrower directories |
| Inject `--approve` | Silently authorizes project resources outside model launch |
| Forward wrapper `--` | Pi does not implement end-of-options at the pinned parser revision |
| Trust Pi semver, `--version`, PATH, npm/shebang, or copied binary | Does not bind the parser/execution closure that receives managed argv |
| Let project TOML mint Pi digests | Self-minted evidence; project may select only reviewed Pi compatibility IDs |
| Shell command string for runtime | Adds quoting, expansion, and executable-selection ambiguity |
| Attach to an occupied listener | Breaks process ownership and cleanup; preflight must refuse |
| Auto-pick another port/profile/runtime or target-only mode | Hides policy mismatch and defeats exact provenance |
| Use raw, cleaned, or sanitized profile text as a state-directory component | Allows traversal or aliases byte-distinct profiles onto shared state and locks; hash exact UTF-8 identity instead |
| Cycle-7 backend catalog + compiled observer + internal proxy | Unspecified scope expansion, requires future agents-infra backend integration, and leaves Muse unsupported today |
| Runtime attestation schema supplied by the selected runtime | The trusted executable can self-mint expected fields; exact shape is not independent authority |
| Require an independent DFlash attestation API | No such contract is provided; it would make the requested Muse profile unlaunchable |
| Claim preflight + liveness + `/v1/models` proves listener ownership | False under the post-preflight bind race and trusted-proxy threat shapes |
| Automate model acquisition, conversion, or benchmarking | Explicitly outside this story and lacks artifact/hardware/runtime inputs |

Cycle-8 directive supersedes the older task checklist sentence requiring an authoritative DFlash attestation schema. The replacement is the practical, explicitly limited acceptance boundary in sections 7-9 and 12; no attestation schema is invented.

## 12. Executable acceptance scenarios

All gate/refusal tests drive the production `agents-infra pi` or production compose entry point. Helper-only and positive-only evidence is insufficient.

### Positive paths

1. With genuine Pi policy absence, launch preserves cwd/environment/argv and executes native Pi without runtime/state creation. An unreadable file control must fail instead.
2. `--print-config` and compose use sentinel Pi/runtime executables and filesystem/socket sentinels; neither executable runs and no lock, file, listener, connection, or process appears.
3. Root and child configs select different profiles; child wins. A child complete profile definition replaces the same ancestor profile atomically. A child may select a different complete ancestor profile.
4. Named production-entry case `TestPiLaunchProfileStateKeyIsolation` selects adversarial exact profile names and proves every resulting state/lock path uses the expected 64-hex key beneath the canonical project root. Byte-distinct case and Unicode normalization variants receive distinct keys and can hold independent locks.
5. Qwen real-entry fixture records exact absolute runtime executable and token vector, binds only the configured loopback port, keeps the direct child alive, returns OpenAI model-list JSON containing exact target, then observes exact normalized Pi argv and isolated paths. Pi exit causes complete runtime-group reap.
6. Muse fixture additionally proves exact configured target/draft token subsequences were passed byte-for-byte, exact target readiness admitted launch, requested/unverified DFlash diagnostics remained correctly labelled, and no fallback argv was introduced.
7. Operator deployment verification runs Pi text plus function-tool round trips for both profiles and a Muse runtime-specific benchmark/telemetry check for DFlash. The artifact records environment/runtime/version and result; it is not launcher authorization evidence.

### Negative and narrowing paths

1. Malformed, partial, unreadable, and unknown-field nearest config each fail before native fallback. Deleting the policy is the control that enables passthrough.
2. Narrow endpoint validation and prove `localhost`, `0.0.0.0`, `[::1]`, remote hosts, user-info, query, and fragment still fail before side effects.
3. Relative runtime executable, directory, non-executable, unreadable, disappearing path, NUL argv, and empty argv each refuse. A path/argv containing shell metacharacters reaches a recording child as literal bytes and executes no shell side effect.
4. Named production-entry case `TestPiLaunchProfileStateKeyIsolation` covers exact names `/`, `\\`, `.`, `..`, `../qwen`, `nested/../qwen`, an absolute-looking name, Unicode slash/backslash lookalikes, case variants, and NFC/NFD variants. Each accepted name remains beneath the canonical hash-only root; byte-distinct variants neither collide nor share locks. Simulated state-key collision, cache-root resolution failure, partial/stat/open failure, non-directory or symlink managed component, and post-create revalidation failure each return the named pre-side-effect error. Narrow state derivation to raw profile text or any slash-replacement, case-folded, Unicode-normalized, or otherwise lossy sanitizer and require this named test to fail.
5. Occupy the exact listener before launch: the production entry refuses before runtime/Pi and never connects. Make the bind probe return a non-`EADDRINUSE` error: it fails `runtime_listener_check_failed`, not vacant.
6. Let a foreign listener expose the exact model while the selected child exits: launch refuses on direct-child death. Narrow the liveness gate to endpoint-only and require this test to fail.
7. Return connection failures until timeout, malformed success JSON, missing `data`, or only a different model: each terminates/reaps the runtime group before Pi. Narrow exact ID comparison to case-folded/fuzzy/substring and require a named mismatch case to fail.
8. Have a live trusted runtime proxy a foreign backend or self-report `dflash=true`: diagnostics still show only configured/unverified capabilities. No `verified` or attested field may appear. This is a reporting-boundary test, not an attempted malicious-runtime rejection.
9. Change/remove/reorder Muse target/draft argv or make the declared model differ from the subsequence terminal token: static validation refuses before listener/runtime. Narrow validation to token-presence-only and require wrong adjacency/duplicate cases to fail.
10. Conflicting/missing managed Pi selections, different provider/model, provider-qualified mismatch, glob selector, ASCII suffix conflict, and Unicode slash/colon lookalikes fail before runtime. Equal recognized `--flag=value` forms normalize and pass the same identity checks.
11. Invoke `-- --provider other`, `-- --model other/model`, `-- --api-key secret`, `-- --thinking high`, `-- --approve`, `-- -prompt`, `-- @file`, and repeated `--`: each fails before runtime. Safe `-- "ordinary prompt"` reaches Pi without a literal separator and preserves bytes.
12. Narrow the bridge by forwarding `--`, scanning only the prefix, admitting one option-looking suffix class, or consuming a prefix value across the removed boundary; a named real-entry case must fail for each mutant.
13. `--api-key secret` and `--api-key=secret` diagnostics preserve normalized/redacted shape but never the value.
14. Seed sentinel global Pi `models.json`, settings, auth, and trust files; require byte identity across managed success/failure. Narrow the generated local catalog to permit command-prefixed keys/headers and require the security test to fail.
15. Exercise absent/malformed/mismatched Pi standalone trees, npm/shebang Pi, copied binary, denied loader environment, and deterministic post-readiness Pi tree mutation. Every case refuses at its named Pi identity gate, reaps runtime where already started, and never starts Pi.
16. Named production-entry case `TestPiLaunchRejectsCatalogCanonicalizationNarrowing` starts from the exact official asset with the configured asset checksum and `./pi` bytes/hash unchanged. Its subcases introduce an extra empty directory, omit a catalogued non-entrypoint file, change one non-entrypoint permission, substitute a symlink or hard-link alias, alter case/normalization of a path where the filesystem permits it, and change record ordering/encoding. Each must fail the execution-closure gate. Narrow the production verifier in turn to a regular-files-only walk, locale/enumeration ordering, content-only digest, ignored extra/missing prefix closure, ignored link/type, or entrypoint-only mode checks; the named real-entry case must fail for each mutant rather than merely proving the whole gate exists.
17. Runtime spawn failure, runtime early exit, Pi spawn failure, SIGINT/SIGTERM, graceful shutdown, timeout escalation, and descendant process fixture each prove complete group cleanup and lock release. Narrow cleanup to direct PID only and require the descendant-survival test to fail.
18. Prove no backend catalog, adapter, observer, attestation endpoint/schema, nonce, capability-verification gate, proxy, or cryptographic-ownership field exists in accepted TOML or launch-plan verified state. Unknown/unverified never becomes satisfied.
19. Document the two excluded adversaries with non-claims: a malicious selected runtime and a malicious same-UID post-preflight bind winner. Tests must not pretend they are rejected by liveness/readiness; operator output must state the limit.

## 13. Board decomposition, dependencies, and gap verification

The existing three-task board is the smallest proportional decomposition:

1. `TASK-260817-2h8hn4` — this verified decision contract; blocks implementation.
2. `TASK-260817-ccpnlm` — implement one cohesive launcher/config/diagnostics/lifecycle/test entry point from sections 4-12.
3. `TASK-260817-3a0zr3` — after the launcher, install/verify `pi-infra` and document operator configuration, verification, and limits.

Dependencies remain decision -> implementation -> alias/documentation. No new story, task, research task, quality-gate task, or diagram is justified. Splitting parser, lifecycle, state, or tests would divide one production entry point without producing independently usable deliverables.

Cycle-10 self-verification before updating existing elements:

- Checked Story description/scope/AC; Task description/scope/AC and DoD; three official Pi documents; current parser/config source; all existing Story children and dependencies; cycle-8 directive; and the explicit exclusions.
- The directive resolves the earlier trust-root question: runtime executable + literal argv is trusted reviewed policy, and agents-infra makes only reproducible claims. Therefore no research task remains open.
- The prior backend catalog/observer/proxy/attestation work is not a justified gap. It is explicitly out of scope, depends on unspecified future integration, and prevents Muse/Qwen launch. It is rejected rather than decomposed.
- Model artifact coordinates and benchmark implementation remain unspecified and explicitly out of scope. The launcher accepts exact operator inputs; operator verification is documented, not automated by a new task.
- Existing downstream tasks are sufficient once their descriptions, AC, checklists, and precondition copy point to this revision.
- Reviewer cycle 8 exposed a genuine completeness gap inside the already-required managed Pi identity gate: the digest alone did not define a reproducible catalog. The exact manifest and deterministic rules above close that gap for the existing implementation task. Checked the Story and task requirements, cycle-8 directive, and every explicit exclusion: this adds no runtime observer, backend catalog, proxy, attestation, acquisition, conversion, benchmark automation, or new product behavior, so no new board element is justified.
- Reviewer cycle 9 exposed a genuine completeness gap inside the Story's required isolated-state boundary: raw profile text could escape or alias the managed cache root. Exact UTF-8 SHA-256 state keys, anchored no-follow containment, explicit read-failure semantics, and production-entry narrowing cases close that gap in the existing implementation/documentation tasks. Rechecked the Story scope and AC, cycle-8 directive, all three official Pi documents, downstream task scopes, and every explicit exclusion; the correction adds no model/runtime acquisition, backend catalog, observer, proxy, attestation, benchmark automation, or new board deliverable.

What remains after this architecture handoff:

- `TASK-260817-ccpnlm`: implement and attack the exact project schema, collision-resistant contained profile state, Pi identity/argv gates, non-launching diagnostics, direct-child runtime lifecycle, loopback/readiness boundary, cleanup, and requested/unverified capability reporting.
- `TASK-260817-3a0zr3`: install/verify the alias and document exact TOML, hash-only profile state paths, operator smoke/benchmark steps, failure behavior, and threat-boundary non-claims.

## 14. Validation and handoff scope

This task is architecture research with explicit `No implementation`. Applicable evidence is the verified English decision artifact, exact board updates, official-source capture, negative acceptance design, Markdown/diff validation, task-board validation, and outcome-resource parity. Product builds/tests remain the downstream implementation task's responsibility.
