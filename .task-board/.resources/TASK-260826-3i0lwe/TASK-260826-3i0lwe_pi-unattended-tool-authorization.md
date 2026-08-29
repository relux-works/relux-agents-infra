# Pi Unattended Tool Authorization Research

Task: `TASK-260825-1q1987`
Story: `STORY-260825-2l6axn`
Research date: 2026-08-25

## 1. Decision summary

Pinned Pi `0.84.2` has no native tool-approval mode to enable or bypass. Active
model tools execute automatically. `--approve` is unrelated: it only permits
Pi to load project-local settings and resources for that run.

Pi does have two useful enforcement surfaces:

1. a strict launch-time tool registry allowlist through `--tools` (or an empty
   registry through `--no-tools`); and
2. a central pre-execution `beforeToolCall` hook exposed to coding-agent
   extensions as `tool_call`, where `{ block: true }` prevents execution.

The smallest sound agents-infra contract is therefore **launch-time
authorization**, not emulation of an approval prompt:

- keep `yolo_mode = false` as the default;
- require an explicit tracked-Pi opt-in plus a non-empty exact tool allowlist;
- on opt-in, inject `--tools <exact-list> --no-extensions --no-approve` and own
  every tool/extension/trust argument so caller ordering cannot change it;
- without opt-in, use `--no-tools` only for an explicitly text-only tracked run,
  otherwise refuse before executable lookup or process start;
- do not expose Pi's direct RPC `bash` command to an untrusted tracked-agent
  client, because it does not traverse the model `tool_call` hook;
- retain OS/container isolation because Pi tools and extensions run with the
  Pi process's full user permissions.

This uses the pinned binary's native controls and requires no Pi fork or policy
extension. A bundled `tool_call` extension is appropriate only if a later
requirement needs argument/path policy, interactive RPC confirmation, or
per-call audit. It is not required to express the current coarse unattended
grant.

## 2. Scope, snapshots, and evidence standard

Primary sources only were used:

- official Pi monorepo tag `v0.84.2`, commit
  `914cf1472e715297caa30db4b9535d534a9eb718`;
- official current upstream snapshot at
  `8fa7eebd235355522c8104166b4f1f959b4e2f10` (`0.84.3`, 2026-08-25);
- the official `v0.84.2` darwin-arm64 standalone artifact already retained by
  agents-infra; and
- agents-infra source in this Story worktree.

The local probes used `--offline --mode rpc --no-session`, sent no model prompt,
and started no MLX or other model runtime. Upstream tool-gate tests used fake
assistant streams. The exact pinned standalone returned `0.84.2` from
`pi --version` with exit `0`.

The coding-agent test checkout required Pi's official `hydrate-model-data`
setup command because generated provider JSON is not retained in the source
tree. That command's fetched catalog values were used only to make module
imports collect; no research claim or citation is derived from those external
catalog services.

Permanent upstream references in this document use the pinned commit rather
than a moving branch. The current snapshot is cited separately where behavior
changed.

## 3. Exact pinned execution and authorization path

### 3.1 Production call path

```text
model AssistantMessage.toolCall
        |
        v
executeToolCalls()
  sequential: prepareToolCall() at agent-loop.ts:452
  parallel:   prepareToolCall() at agent-loop.ts:507
        |
        v
resolve tool -> prepare arguments -> validate arguments
        |
        v
config.beforeToolCall?                         agent-loop.ts:619-628
  absent / returns undefined ------------------------------+
  returns { block: true } -> error ToolResult, no execute   |
  throws -> caught as error ToolResult, no execute          |
                                                           v
                                             prepared.tool.execute()
                                             agent-loop.ts:679-697
```

`Agent.createLoopConfig()` copies `Agent.beforeToolCall` into the loop config.
Both sequential and parallel execution call `prepareToolCall()` before any
tool's `execute()` method. The hook runs only after schema validation. A block
returns an immediate error tool result; a hook exception is also converted to
an immediate error result. Only a prepared result reaches `tool.execute()`.
See the pinned [Agent hook wiring](https://github.com/badlogic/pi-mono/blob/914cf1472e715297caa30db4b9535d534a9eb718/packages/agent/src/agent.ts#L445-L459),
[sequential/parallel preflight](https://github.com/badlogic/pi-mono/blob/914cf1472e715297caa30db4b9535d534a9eb718/packages/agent/src/agent-loop.ts#L411-L425),
[pre-execution decision](https://github.com/badlogic/pi-mono/blob/914cf1472e715297caa30db4b9535d534a9eb718/packages/agent/src/agent-loop.ts#L600-L668),
and [actual execution](https://github.com/badlogic/pi-mono/blob/914cf1472e715297caa30db4b9535d534a9eb718/packages/agent/src/agent-loop.ts#L670-L711).

The coding-agent layer installs that hook once in the `AgentSession`
constructor. At execution time it invokes the current extension runner's
`tool_call` handlers. With no handler, it returns `undefined`, which means
allow. Extension handlers run in extension order and the first block wins.
See [AgentSession installation and bridge](https://github.com/badlogic/pi-mono/blob/914cf1472e715297caa30db4b9535d534a9eb718/packages/coding-agent/src/core/agent-session.ts#L377-L402),
[the installed hook](https://github.com/badlogic/pi-mono/blob/914cf1472e715297caa30db4b9535d534a9eb718/packages/coding-agent/src/core/agent-session.ts#L471-L499),
and [extension dispatch](https://github.com/badlogic/pi-mono/blob/914cf1472e715297caa30db4b9535d534a9eb718/packages/coding-agent/src/core/extensions/runner.ts#L932-L953).

### 3.2 What the path means

- There is **no built-in approval decision** in the default path. No handler
  means execute.
- `tool_execution_start` is emitted before `prepareToolCall()`. It is an event,
  not an authorization boundary; an observer cannot authorize by watching it.
- `tool_call` is a genuine pre-execution veto for every model tool in the
  active registry, including extension/custom tools and same-name replacements.
- A throwing `tool_call` handler fails that model call closed because the throw
  becomes an error result before `execute()`.
- `terminate: true` may stop the agent loop after the blocked batch; blocking
  itself does not depend on termination. The typed contract is documented in
  [agent types](https://github.com/badlogic/pi-mono/blob/914cf1472e715297caa30db4b9535d534a9eb718/packages/agent/src/types.ts#L52-L69)
  and [extension result types](https://github.com/badlogic/pi-mono/blob/914cf1472e715297caa30db4b9535d534a9eb718/packages/coding-agent/src/core/extensions/types.ts#L1071-L1080).

The official pinned regression tests drive fake tool calls through the real
loop/session paths. They assert that a blocked tool's `execute()` method is not
called and that an error tool result is emitted. See the
[agent-core negative](https://github.com/badlogic/pi-mono/blob/914cf1472e715297caa30db4b9535d534a9eb718/packages/agent/test/agent-loop.test.ts#L1253-L1310)
and [coding-agent extension negative](https://github.com/badlogic/pi-mono/blob/914cf1472e715297caa30db4b9535d534a9eb718/packages/coding-agent/test/suite/agent-session-model-extension.test.ts#L116-L156).

## 4. Project trust is not tool permission

The distinction is explicit in both source and official documentation:

- CLI parsing stores `--approve` and `--no-approve` in
  `projectTrustOverride`; it does not set a tool field
  ([args.ts:205-208](https://github.com/badlogic/pi-mono/blob/914cf1472e715297caa30db4b9535d534a9eb718/packages/coding-agent/src/cli/args.ts#L205-L208)).
- `resolveProjectTrusted()` applies that override before saved/default/UI trust
  resolution
  ([project-trust.ts:46-95](https://github.com/badlogic/pi-mono/blob/914cf1472e715297caa30db4b9535d534a9eb718/packages/coding-agent/src/core/project-trust.ts#L46-L95)).
- Project trust determines whether Pi loads project settings, packages,
  extensions, skills, prompts, and themes. It is explicitly “not a sandbox”
  and “does not restrict what the model can ask tools to do”
  ([security.md:5-37](https://github.com/badlogic/pi-mono/blob/914cf1472e715297caa30db4b9535d534a9eb718/packages/coding-agent/docs/security.md#L5-L37)).
- In non-interactive modes, `--approve` only overrides project trust for one
  run; no trust prompt is shown
  ([security.md:27-29](https://github.com/badlogic/pi-mono/blob/914cf1472e715297caa30db4b9535d534a9eb718/packages/coding-agent/docs/security.md#L27-L29)).

Local pinned-binary evidence makes the separation concrete:

| Launch | Active/all tools observed at `session_start` | Meaning |
| --- | --- | --- |
| `--no-approve` | defaults active: `read,bash,edit,write`; all seven built-ins present | Declining project trust does not disable tools. |
| `--approve` | same defaults plus project `project_probe` extension tool | Trust admits project code/resources. |
| `--approve --no-tools` | active/all both empty | `--no-tools`, not trust, controls tool availability. |
| `--approve --tools read` | only `read` exists and is active | `--tools` is a strict registry allowlist. |

Therefore agents-infra must never map `yolo_mode` to `--approve`, and a
diagnostic must report project trust and tool authorization as separate fields.
The existing agents-infra rejection at
`tools/agents-infra/internal/infra/pi_config.go:583-595` correctly identifies
this mismatch, but its claim that Pi exposes no unattended mechanism is too
broad: Pi lacks an approval policy, while strict tool selection plus automatic
execution is sufficient for a wrapper-owned unattended grant.

## 5. Native and programmable control inventory

### 5.1 CLI and settings

Pinned and current upstream expose these controls:

| Surface | Exact behavior | Authorization value | Caveat |
| --- | --- | --- | --- |
| `--tools a,b` | Strict allowlist across built-in, extension, and SDK custom tools. | Strong launch-time name scope. | A permitted name can refer to an extension replacement unless extension loading is separately controlled. |
| `--exclude-tools a,b` | Denylist after the initial selection. | Useful narrowing. | Harder to audit than a positive allowlist; new tools can remain enabled. |
| `--no-tools` | Empty allowed registry. | Strong text-only mode. | No model tools at all. |
| `--no-builtin-tools` | Removes built-ins, retains extension/custom tools. | Not a safe deny-all. | Custom tools still execute automatically. |
| `defaultTools` setting | Initial built-in selection. | Convenience only. | Extension/custom tools remain enabled; project setting is trust-controlled. |
| `--no-extensions` | Disables discovery but preserves explicit `-e` paths. | Essential for a wrapper-owned registry. | Caller-supplied `-e` must also be rejected/owned. |
| `--approve` / `--no-approve` | One-run project-input trust override. | None for tool calls. | Trusted project extensions execute arbitrary TypeScript. |

The CLI help defines the all-tool allowlist and explicit-extension exception
([args.ts:281-303](https://github.com/badlogic/pi-mono/blob/914cf1472e715297caa30db4b9535d534a9eb718/packages/coding-agent/src/cli/args.ts#L281-L303)).
The SDK derives a default active set of `read,bash,edit,write`, converts
`--no-tools` into an empty allowlist, and gives an explicit `tools` list
precedence
([sdk.ts:247-254](https://github.com/badlogic/pi-mono/blob/914cf1472e715297caa30db4b9535d534a9eb718/packages/coding-agent/src/core/sdk.ts#L247-L254)).
The registry applies the allowlist before merging built-ins and custom tools;
custom tools replace the same built-in name
([agent-session.ts:2463-2529](https://github.com/badlogic/pi-mono/blob/914cf1472e715297caa30db4b9535d534a9eb718/packages/coding-agent/src/core/agent-session.ts#L2463-L2529)).
Official settings documentation confirms that `defaultTools` preserves custom
tools while `--tools` is strict
([settings.md:221-231](https://github.com/badlogic/pi-mono/blob/914cf1472e715297caa30db4b9535d534a9eb718/packages/coding-agent/docs/settings.md#L221-L231)).

### 5.2 Extensions and custom tool replacement

An extension may:

- veto a model call in `pi.on("tool_call", ...)`;
- use `ctx.ui.confirm()` for supervised approval, including over the RPC UI
  request/response protocol;
- register a new tool through `pi.registerTool()`; or
- replace a built-in by registering the same name.

Official extension docs warn that extensions run with full process permissions
([extensions.md:110-120](https://github.com/badlogic/pi-mono/blob/914cf1472e715297caa30db4b9535d534a9eb718/packages/coding-agent/docs/extensions.md#L110-L120)).
The official [tool override example](https://github.com/badlogic/pi-mono/blob/914cf1472e715297caa30db4b9535d534a9eb718/packages/coding-agent/examples/extensions/tool-override.ts#L1-L20)
explicitly lists access control and sandbox routing as uses for replacement.

The pinned binary reproduced both sides:

- `--approve` loaded a project `bash` replacement and the registry reported
  `bash` from the project extension rather than `builtin`;
- `--approve --no-extensions -e <probe>` suppressed the project extension but
  still loaded the explicit CLI probe.

This makes `--tools` plus unrestricted extensions insufficient as a stable
tool-implementation identity. The wrapper must disable discovery and reject
caller-supplied explicit extensions. A future agents-infra-owned policy
extension may be the sole explicit `-e` path, with its identity catalogued and
rechecked immediately before spawn.

### 5.3 Skills `allowed-tools` metadata

Pinned/current `docs/skills.md` describe an experimental `allowed-tools`
frontmatter field as “pre-approved tools.” A repository-wide source/test search
at both inspected snapshots found `allowed-tools` only in that documentation
row and found no consumer or enforcement path. It must be treated as
non-authoritative metadata, not as a tool permission. This is an upstream
documentation/implementation anomaly, not evidence of a hidden approval flow.
See the pinned [skills documentation](https://github.com/badlogic/pi-mono/blob/914cf1472e715297caa30db4b9535d534a9eb718/packages/coding-agent/docs/skills.md#L137-L149).

### 5.4 RPC mode

RPC is a transport, not a model-tool approval service:

- its command union has prompts, state controls, and a direct `bash` command,
  but no authorize/approve-tool command
  ([rpc-types.ts:20-73](https://github.com/badlogic/pi-mono/blob/914cf1472e715297caa30db4b9535d534a9eb718/packages/coding-agent/src/modes/rpc/rpc-types.ts#L20-L73));
- model-generated tool calls still traverse `AgentSession` and `tool_call`;
- an extension may block on `ctx.ui.confirm()`, which RPC translates into an
  `extension_ui_request` and matching response
  ([rpc.md:1155-1164](https://github.com/badlogic/pi-mono/blob/914cf1472e715297caa30db4b9535d534a9eb718/packages/coding-agent/docs/rpc.md#L1155-L1164));
- the direct RPC `bash` command instead emits `user_bash` and calls
  `session.executeBash()`; it never invokes `tool_call`
  ([rpc-mode.ts:559-580](https://github.com/badlogic/pi-mono/blob/914cf1472e715297caa30db4b9535d534a9eb718/packages/coding-agent/src/modes/rpc/rpc-mode.ts#L559-L580)).

The local probe loaded a `tool_call` extension that always blocks, then sent a
direct RPC `bash` request. Pi emitted `rpc-direct-bash-ok` and returned
`success:true`, `exitCode:0`. This is an intentional trusted-controller path,
but it is a bypass if a product claims that one `tool_call` policy controls all
execution.

`user_bash` interception is also weaker as a fail-closed gate: its result type
has no `block` field, and runner exceptions are reported then ignored before
returning `undefined`. A handler can synthesize a denial result, but agents-infra
should gate or omit the RPC command at its controller boundary instead of
depending on that hook.

## 6. Current upstream check

At current snapshot `8fa7eebd235355522c8104166b4f1f959b4e2f10`:

- the three agent-core files carrying `beforeToolCall` and the execution path
  are byte-identical to `v0.84.2` (`git diff --exit-code`, exit `0`);
- coding-agent still installs `tool_call` before execution and exposes the same
  CLI tool/trust/extension controls;
- no new native tool-approval flag, approval setting, or RPC authorization
  command was found;
- RPC adds `clear_queue`; and
- current Pi adds an optional `powershell` tool, illustrating why an exact
  positive allowlist is safer than “all current tools except...” policy.

Current source references:

- [AgentSession hook](https://github.com/badlogic/pi-mono/blob/8fa7eebd235355522c8104166b4f1f959b4e2f10/packages/coding-agent/src/core/agent-session.ts#L478-L506)
- [RPC command union](https://github.com/badlogic/pi-mono/blob/8fa7eebd235355522c8104166b4f1f959b4e2f10/packages/coding-agent/src/modes/rpc/rpc-types.ts#L20-L76)
- [current CLI help](https://github.com/badlogic/pi-mono/blob/8fa7eebd235355522c8104166b4f1f959b4e2f10/packages/coding-agent/src/cli/args.ts)
- [upstream PowerShell tool change](https://github.com/badlogic/pi-mono/commit/80e62761f7251a104f1b21d9c73920c720f0ec00)

### 6.1 Tracked task-board spawn is a separate integration boundary

This research proves how agents-infra can authorize and launch Pi. It does
**not** prove that current task-board tracked spawn can select Pi.

The current task-board source binds built-in runtime ID `qwen` to agentic
system `qwen-code`, whose adapter resolves the `qwen` binary and emits Qwen Code
stream-JSON grammar. Pi is not one of the shipped agentic systems. Primary
local sources at `skill-project-management@f6e7898f02d372d804bba727c59f7bb2f4c0ac51`
are `pkg/remoteconfig/runtimeid/builtins.go:73-84` and
`tools/board-cli/vendor/github.com/relux-works/skill-agents-management/pkg/agentic/systems/qwen/qwen.go:47-68`.

`spawn.runtimes` cannot manufacture a new harness. It declares a binding to
one of the launch adapters already shipped by the release, and the project's
own contract states that it adds vocabulary/selectability rather than behavior
(`skill-project-management@f6e7898f02d372d804bba727c59f7bb2f4c0ac51:references/task-board-config.md:347-419`).

Actual tracked Pi spawn therefore needs cross-repository adapter work in
skill-agents-management/task-board: a shipped Pi agentic-system plugin with
binary/argv/stdin/stdout/environment semantics, a runtime/model registration,
and production spawn tests that invoke the agents-infra Pi boundary. Only then
may a `spawn.runtimes`/ceiling entry select it. The agents-infra authorization
change is necessary for that integration but is not sufficient evidence of it.

## 7. Local reproduction matrix

Every row below was a standalone command against the exact `0.84.2`
standalone/source. No row launched a model runtime.

| Check | Exit | Observed result |
| --- | ---: | --- |
| `pi --version` | `0` | Exact binary reports `0.84.2`. |
| RPC probe: `--no-approve` | `0` | Default model tools remain active; project extension absent. |
| RPC probe: `--approve` | `0` | Project extension tool becomes active; built-in defaults unchanged. |
| RPC probe: `--approve --no-tools` | `0` | Active and complete registries are empty. |
| RPC probe: `--approve --tools read` | `0` | Only `read` exists and is active. |
| RPC probe: `--approve --no-extensions -e <probe>` | `0` | Project discovery suppressed; explicit policy probe loaded. |
| RPC probe: project replaces `bash` | `0` | Registry reports project source for allowed name `bash`. |
| RPC direct `bash` with blocking `tool_call` hook | `0` | Command executes and response reports `exitCode:0`; hook is not on this path. |
| Agent-core blocked-tool test | `0` | 1 passed, 22 skipped; tool side-effect flag remains false. |
| Coding-agent extension block test, initial | `1` | Test collection failed because source checkout lacked generated provider JSON after `npm ci --ignore-scripts`; no test ran. |
| Official `hydrate-model-data` setup step | `0` | Required generated provider JSON materialized. |
| Same coding-agent extension block test, rerun | `0` | 1 passed, 11 skipped; throwing tool implementation was not reached. |
| Pinned-to-current agent-core hook diff | `0` | No source delta in `agent.ts`, `agent-loop.ts`, or `types.ts`. |

The initial exit `1` is retained as a real setup failure, not counted as a
passing gate. Evidence logs are under `.temp/TASK-260825-1q1987/` in this
worktree.

## 8. Alternatives and tradeoffs

| Mechanism | Autonomy | Security property | Pinned/current compatibility | Maintenance |
| --- | --- | --- | --- | --- |
| Wrapper gate + `--tools` + `--no-extensions` | Fully unattended | Explicit launch admission and exact name scope; stable implementations because discovery is off | Native in 0.84.2 and current | Lowest; argv/schema/tests only |
| Bundled `tool_call` policy extension | Unattended or RPC-confirmed | Central per-call argument/path veto; handler throw blocks model call | Native extension API in 0.84.2/current | Medium; catalogue extension, load marker, reload/resume tests, TS API drift |
| Replace every built-in/custom tool | Fully unattended | Full executor control per replacement | Supported in both | High; duplicate tool contracts/rendering and track new tools such as `powershell` |
| RPC host + confirmation extension | Supervised, not unattended | Human per-call decision for model tools | Supported in both | High; request correlation, timeout/cancel/recovery; direct RPC bash remains separate |
| SDK embedding or Pi core patch | Any desired policy | Strongest unskippable in-process boundary | SDK hook exists; fork must track upstream | Highest; abandons simple standalone launch or carries a fork |
| `--approve`, saved trust, skill `allowed-tools` | N/A | No tool authorization | Present/documented | Unsound for this purpose |

The first option is sufficient for the requested coarse “may this tracked Pi
run tools unattended?” decision. The extension option becomes justified only
when the policy must inspect arguments or ask an external approver per call.

### 8.1 Privileged execution candidate

Privilege is a separate axis from Pi tool authorization. Base Pi already has
no model-tool approval popup, so wrapping commands in `sudo` does not make an
unattended mode possible; it only changes the OS identity and damage ceiling.

| Candidate | Effective boundary | Assessment for tracked unattended Pi |
| --- | --- | --- |
| Run the whole Pi/agents-infra process as root or Administrator | Every built-in tool, extension, dependency, and process child receives administrator authority. | Reject. Pi officially runs with the permissions of its process; this turns prompt/model/extension compromise into unrestricted host compromise. |
| `NOPASSWD: ALL` or equivalent unrestricted elevation | Any authorized `bash` call can obtain arbitrary root execution without a human gate. | Reject; materially equivalent to a root-capable agent. |
| `NOPASSWD` command allowlist | `NOPASSWD` removes authentication only; safety depends entirely on exact executable and argument matching. | Do not include in the generic contract. It is acceptable only when the listed command is itself a narrow audited helper with a closed argument language. |
| Root-owned allowlisted helper/daemon | Pi remains unprivileged and requests a small typed capability from separately installed privileged code. | The only acceptable future shape, but it requires its own design, user/admin installation approval, threat model, and negative tests. Not enabled by `yolo_mode`. |

The official sudoers grammar says that a listed command with no command-line
arguments may be run with any arguments, and that argument rules use either
regular expressions or shell wildcards
([sudoers source](https://github.com/sudo-project/sudo/blob/2e18923c2e959fff57b60207a261080daa2ebee9/docs/sudoers.man.in#L1090-L1136)).
It separately defines `NOPASSWD` as changing whether authentication is
required for following commands
([sudoers source](https://github.com/sudo-project/sudo/blob/2e18923c2e959fff57b60207a261080daa2ebee9/docs/sudoers.man.in#L2065-L2088)).
The maintainers explicitly warn that argument wildcards match whitespace and
that regular expressions are generally safer
([sudoers source](https://github.com/sudo-project/sudo/blob/2e18923c2e959fff57b60207a261080daa2ebee9/docs/sudoers.man.in#L2216-L2332)).
These controls can narrow a command, but they do not turn an arbitrary
privileged utility into a narrow capability.

On macOS, Apple's supported Service Management surface can install/register a
launch daemon subject to user/admin approval. Apple describes a LaunchDaemon
as a root process that responds to low-level IPC requests, and `SMAppService`
registers helper executables from the app bundle
([Service Management](https://developer.apple.com/documentation/servicemanagement),
[SMAppService](https://developer.apple.com/documentation/servicemanagement/smappservice),
[registration approval](https://developer.apple.com/documentation/servicemanagement/smappservice/register%28%29)).
That is an installation/lifecycle mechanism, not authorization logic. A future
helper must still expose only named typed operations, reject arbitrary command
strings and paths, authenticate the local client, validate path containment,
run no shell, keep its executable/config root-owned and non-writable by the Pi
user, and audit allow/deny outcomes.

No privileged path belongs in the current unattended-spawn contract. In
particular, authorizing Pi's `bash` tool cannot support a claim of unprivileged
execution when the host account already has unrestricted passwordless sudo;
that absence must come from the OS account/container boundary, not a Pi flag.

## 9. Recommended agents-infra contract

### 9.1 Configuration and defaults

Add a tracked/headless Pi policy owned by agents-infra, conceptually:

```toml
[agents.pi.tracked_session]
yolo_mode = true
tool_allowlist = ["read", "bash", "edit", "write"]
```

Exact field placement may reuse the existing composed Pi session policy, but
the semantics must be these:

1. `yolo_mode` defaults to `false` with explicit nearest-field false masking.
2. The grant applies only to independently tracked/headless Pi launches. It is
   not inferred from project trust, a primary interactive session, a task-board
   status, or a parent agent's policy.
3. `yolo_mode = true` requires a present, non-empty, duplicate-free exact
   `tool_allowlist`. Unknown names fail before spawn; no “all” wildcard exists.
4. `yolo_mode = false` or absence plus a tool-capable tracked request returns a
   typed `pi_tool_authorization_required` error before executable lookup,
   runtime startup, state writes, locks, or process spawn.
5. An explicitly text-only tracked request launches with `--no-tools`; it is not
   a silent fallback from a refused tool-capable request.
6. Project trust remains independently defaulted to false for tracked runs.
   Do not emit `--approve` as part of tool authorization.

### 9.2 Canonical argv and ownership

For an authorized run, agents-infra emits exactly one canonical set before the
message operand boundary:

```text
--no-approve
--no-extensions
--tools read,bash,edit,write
```

The existing argument composer currently passes all three classes through
(`pi_args.go:51-68`, `180-228`). The implementation must instead reserve and
normalize these tool-boundary options for tracked authorization:

- `--tools` / `-t`
- `--exclude-tools` / `-xt`
- `--no-tools` / `-nt`
- `--no-builtin-tools` / `-nbt`
- `--extension` / `-e`
- `--no-extensions` / `-ne`
- `--approve` / `-a`
- `--no-approve` / `-na`

Caller-supplied occurrences, duplicates, attached unknown spellings, or
post-delimiter flag-shaped operands must not override the managed selection.
Prefer typed rejection over last-wins composition. Existing Pi identity and
environment revalidation immediately before spawn remains in force
(`pi_launch_posix.go:220-243`).

### 9.3 RPC boundary

Prefer `--print` or JSON mode when a tracked child needs one autonomous prompt.
If agents-infra uses RPC for lifecycle control:

- the controller must expose only the commands required by the tracked-agent
  protocol;
- direct `bash` and equivalent user-command injection must be rejected unless
  separately authorized as controller actions;
- `tool_execution_*` events are audit evidence only, never authorization;
- a future confirmation extension must receive and answer
  `extension_ui_request` messages, but that is supervised mode and must not be
  labelled unattended.

### 9.4 Diagnostics

Non-launching compose/print diagnostics should report at least:

```json
{
  "tool_authorization": {
    "mode": "unattended_allowlist",
    "effective": true,
    "source": "/canonical/project/.agents/.configs/project-config.toml",
    "scope": "tracked_session",
    "allowed_tools": ["read", "bash", "edit", "write"],
    "native_enforcement": "pi_cli_strict_allowlist",
    "extension_discovery": "disabled",
    "project_trust": "declined",
    "rpc_direct_bash": "not_exposed"
  }
}
```

Report the pinned Pi compatibility identity and final normalized diagnostic
argv nearby. Never report secrets or claim a sandbox. Missing, unreadable,
partial, malformed, ambiguous, or unsupported policy is an error, not policy
absence.

### 9.5 Security boundary

This contract limits which Pi model-tool implementations are reachable and
requires explicit unattended admission. It does not confine those tools after
admission. Pi's official security contract says tools and extensions run with
the process user's permissions and recommends a container, VM, micro-VM, or
policy-controlled sandbox for unattended work
([security.md:31-53](https://github.com/badlogic/pi-mono/blob/914cf1472e715297caa30db4b9535d534a9eb718/packages/coding-agent/docs/security.md#L31-L53)).
Tracked Pi should retain agents-infra's isolated state, restricted environment,
process ownership, and task-scoped workspace boundary.

### 9.6 Privilege default

The tracked-Pi contract grants no privileged capability and emits no `sudo`,
administrator, setuid, helper, or daemon configuration. Diagnostics should say
`"privilege_boundary":"calling_user"` and must not claim that passwordless
elevation is absent unless the containing OS policy proves it. A later
privileged helper is a separate explicit capability contract, never implied by
`yolo_mode`, `tool_allowlist`, project trust, or task status.

## 10. Minimal patch points

No upstream Pi patch is recommended.

The smallest agents-infra implementation touches:

1. Pi tracked/session policy parsing and composition: replace
   `pi_yolo_mode_unsupported` for the tracked scope with the authorization and
   allowlist validation above.
2. `BuildManagedPiArguments`: reserve tool, extension, and trust options and
   append one canonical managed set.
3. launch-plan diagnostics: expose separate tool-authorization and project
   trust fields.
4. `RunPi`: retain the existing fail-before-lookup ordering and point-of-use
   identity/environment rechecks.
5. production-entry tests at both non-launching compose and direct `RunPi`.

If future requirements add argument/path policy, the next-smallest patch is an
agents-infra-owned extension loaded as the sole explicit `-e` path. Catalogue
its entire execution closure, verify it before side effects and immediately
before spawn, require a `session_start` identity marker, and prove that reload,
resume, and cwd changes preserve the handler. Absence or load failure must stop
the run; a missing handler must never mean allow.

## 11. Negative test plan

Tests must drive the real compose/launch entry points and preserve real exit
codes.

### 11.1 Admission and evidence

1. Omitted/false `yolo_mode` plus a tool-capable tracked request fails with
   `pi_tool_authorization_required` before executable lookup. Assert lookup,
   runtime, state, lock, and Pi process sentinels are untouched.
2. Explicit text-only plus false/absent yolo reaches Pi with exactly
   `--no-tools`; a tool-capable request never silently downgrades to this lane.
3. True yolo without an allowlist, with an empty/duplicate/unknown allowlist,
   or with an unreadable/malformed config fails closed. A failed read is not
   absence.
4. `--approve` alone cannot satisfy authorization and is never emitted as its
   translation. Project trust and tool authorization diagnostics remain
   independent.
5. Caller-supplied tool/trust/extension flags cannot override the canonical
   set before or after the wrapper delimiter.

### 11.2 Registry and bypasses

6. Drive a fake assistant tool call through a launched pinned AgentSession:
   allowlisted tool changes a task-scoped sentinel only when yolo is true.
7. The same fake call with yolo false or an unlisted name leaves the sentinel
   unchanged and yields refusal/error evidence.
8. Place project and global extensions that register a new tool and replace
   allowed name `bash`; authorized tracked launch loads neither. The registry
   must report only built-in sources.
9. Add a new upstream-like built-in (for example `powershell`) in a fixture;
   it remains absent unless explicitly allowlisted.
10. If RPC is used, model-generated `bash` traverses the policy, while a raw
    RPC `bash` command is rejected by the controller and cannot produce a
    sentinel.

### 11.3 Attacks on the proof

11. A caller cannot self-mint diagnostics, a policy marker, or an allowlist
    hash; agents-infra derives evidence from resolved config and normalized
    argv.
12. Revalidate Pi and any future policy asset immediately before spawn; mutate
    between compose and spawn and require refusal.
13. Resume, reload, and cwd-switch cases preserve the managed registry and
    extension-discovery-off state.
14. Narrowing mutant: apply the strict allowlist only to built-ins, leaving
    custom/replacement tools admitted. The project replacement negative must
    fail. This proves the bounded class, unlike a delete-only mutant.
15. Narrowing mutant: stop ownership parsing at the first wrapper delimiter so
    a later tool/extension flag reaches Pi. The real launch-entry bypass test
    must fail.
16. If a policy extension is later added, its thrown handler must leave the
    tool sentinel unchanged; missing handler/load marker must refuse the whole
    run rather than fall back to Pi's default allow behavior.
17. A tracked spawn configured to execute Pi must fail while task-board has no
    shipped Pi agentic-system adapter; declaring a `spawn.runtimes` row alone
    must not make the capability claim reproduce.
18. The ordinary contract never emits `sudo` and never launches Pi as a
    privileged helper. A root/admin test fixture must be refused or explicitly
    classified outside this contract rather than reported as ordinary yolo.
19. If a privileged helper is later introduced, send unknown operation,
    arbitrary command, traversal/symlink path, forged client identity, and
    malformed/partial request shapes through the real IPC entry point; each
    must fail without privileged side effects.
20. For any later sudoers-based prototype, prove the exact installed rule with
    both allowed and adjacent denied argv. A wildcard-narrowing mutant that
    admits whitespace/additional arguments must make the negative fail.

## 12. Answered questions and residual unknowns

- **Exact authorization call path:** Section 3. Pi has an optional pre-tool
  veto but no default approval decision; absent handler means allow.
- **Project trust versus tool permission:** Section 4. They are orthogonal.
- **Pinned/current flags, settings, extensions, custom tools, RPC:** Sections
  5-6.
- **Candidate behavior without MLX:** Section 7.
- **Autonomy, security, compatibility, maintenance:** Section 8.
- **Concrete agents-infra implementation, defaults, opt-in, diagnostics, and
  negative tests:** Sections 9-11.
- **Privilege candidates:** Section 8.1. Root/unrestricted sudo are rejected;
  only a separately designed narrow helper is acceptable if future privileged
  capability is required.
- **Actual task-board tracked spawn:** Section 6.1. Current `qwen` is qwen-code;
  Pi needs shipped cross-repository adapter work before the authorization
  contract can be claimed end to end.

No product or upstream capability question remains unresolved for the coarse
tracked-session unattended grant. Per-command/path policy and human RPC
confirmation are deliberately separate future modes, not hidden requirements
of this recommendation.

## 13. Primary source index

- [Official Pi `v0.84.2` source tree](https://github.com/badlogic/pi-mono/tree/914cf1472e715297caa30db4b9535d534a9eb718)
- [Official Pi `v0.84.2` release](https://github.com/badlogic/pi-mono/releases/tag/v0.84.2)
- [Official standalone `v0.84.2` release](https://github.com/earendil-works/pi/releases/tag/v0.84.2)
- [Pinned agent execution loop](https://github.com/badlogic/pi-mono/blob/914cf1472e715297caa30db4b9535d534a9eb718/packages/agent/src/agent-loop.ts)
- [Pinned coding-agent session](https://github.com/badlogic/pi-mono/blob/914cf1472e715297caa30db4b9535d534a9eb718/packages/coding-agent/src/core/agent-session.ts)
- [Pinned CLI parser](https://github.com/badlogic/pi-mono/blob/914cf1472e715297caa30db4b9535d534a9eb718/packages/coding-agent/src/cli/args.ts)
- [Pinned official security documentation](https://github.com/badlogic/pi-mono/blob/914cf1472e715297caa30db4b9535d534a9eb718/packages/coding-agent/docs/security.md)
- [Pinned official RPC documentation](https://github.com/badlogic/pi-mono/blob/914cf1472e715297caa30db4b9535d534a9eb718/packages/coding-agent/docs/rpc.md)
- [Pinned official extension documentation](https://github.com/badlogic/pi-mono/blob/914cf1472e715297caa30db4b9535d534a9eb718/packages/coding-agent/docs/extensions.md)
- [Current upstream snapshot](https://github.com/badlogic/pi-mono/tree/8fa7eebd235355522c8104166b4f1f959b4e2f10)
- [Official sudoers source snapshot](https://github.com/sudo-project/sudo/blob/2e18923c2e959fff57b60207a261080daa2ebee9/docs/sudoers.man.in)
- [Apple Service Management documentation](https://developer.apple.com/documentation/servicemanagement)
- Task-board local primary source: `skill-project-management@f6e7898f02d372d804bba727c59f7bb2f4c0ac51:pkg/remoteconfig/runtimeid/builtins.go:73-84`
- Task-board local primary source: `skill-project-management@f6e7898f02d372d804bba727c59f7bb2f4c0ac51:references/task-board-config.md:347-419`
