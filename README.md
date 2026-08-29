# relux-agents-infra

Source repo for shared AI agent configurations, instructions, skills, and rules.

Works with:
- **Claude Code** (`~/.claude/`)
- **Codex CLI** (`~/.codex/`)

## Quick Start

```bash
# Bootstrap the launcher, then immediately sync the global runtime
cd /path/to/relux-agents-infra
./setup.sh

# Windows bootstrap
.\setup.ps1

# Bootstrap and also install the optional PDF toolchain
./setup.sh --with-pdf-tools

# Use the installed CLI after bootstrap
agents-infra setup global
agents-infra setup local /path/to/project
agents-infra doctor global
agents-infra doctor local /path/to/project
agents-infra compose --agent codex --project /path/to/project --schema-version 1 --json
openai-infra --print-config
pi-infra --print-config
qwen-infra spawn --prompt "Complete the bounded task" --deadline 10m
agents-infra model-check --target qwen-infra --prompt "Reply with READY" --output-dir .temp/model-check
agents-infra version
```

`setup.sh` and `setup.ps1` are bootstrap wrappers. They delegate into
`scripts/setup.sh` and `scripts/setup.ps1`, build the `agents-infra` launcher
and the separate `model-harness` runtime binary with version metadata, install
them into the user-local bin directory, write install-state metadata, and then
immediately run `agents-infra setup global`.

Install-state metadata lives under the standard user config directory:

- macOS: `~/Library/Application Support/agents-infra/install.json`
- Windows: `%AppData%\agents-infra\install.json`

The canonical interface after bootstrap is:
- `agents-infra setup global`
- `agents-infra setup local [PATH]`
- `agents-infra doctor global|local`
- `agents-infra compose --agent codex|claude --project DIR --schema-version 1 --json`
- `agents-infra compose --mode primary-session --agent codex|claude|pi --project DIR --schema-version 1 --json [-- PROVIDER_ARGS...]`
- `agents-infra compose --mode primary-session --entrypoint openai-infra|anthropic-infra|qwen-infra --project DIR --schema-version 1 --json [-- PROVIDER_ARGS...]`
- `agents-infra target openai-infra|anthropic-infra|qwen-infra [--print-config] [-- PROVIDER_ARGS...]`
- `agents-infra prepare --agent codex|claude --project DIR --schema-version 1 --json`
- `agents-infra codex [--print-config] [-d] [CODEX_ARGS...]`
- `agents-infra claude [--print-config] [-d] [CLAUDE_ARGS...]`
- `agents-infra pi [--print-config] [--profile NAME] [PI_ARGS...] [-- MESSAGE...]`
- `agents-infra pi spawn --prompt TEXT [--deadline DURATION] [--print-config]`
- `agents-infra runtime status [--project DIR] [--profile NAME] [--json]`
- `agents-infra runtime stop [--project DIR] [--profile NAME] [--force] [--timeout SECONDS]`
- `agents-infra runtime quarantine [--project DIR] [--profile NAME]`
- `agents-infra runtime unquarantine [--project DIR] [--profile NAME]`
- `pi-infra [--print-config] [--profile NAME] [PI_ARGS...] [-- MESSAGE...]`
- `openai-infra|anthropic-infra|qwen-infra [--print-config] [-- PROVIDER_ARGS...]`
- `qwen-infra spawn --prompt TEXT [--deadline DURATION] [--print-config]`
- `agents-infra model-check --target ENTRYPOINT --prompt TEXT --output-dir DIR [--deadline DURATION] [--expect-tool NAME] [--expect-text TEXT]`
- `agents-infra version`
- `model-harness render PROFILE --host 127.0.0.1 --port PORT --json [--config PATH]`
- `model-harness doctor PROFILE --host 127.0.0.1 --port PORT [--config PATH]`
- `model-harness run PROFILE --host 127.0.0.1 --port PORT [--config PATH]`

`model-check` exits `0` when the managed check and all expectations pass, `1`
for launch, validation, or cleanup failure, `2` on deadline expiry, `3` for a
malformed or incomplete JSONL stream, `4` for unmet tool/text expectations,
and `5` when the model reports a failed tool execution. Raw provider output is
kept in mode-`0600` files inside the mode-`0700` evidence directory; terminal
output is sanitized. The command passes Pi `--approve` so reviewed
project-local inputs are loaded for that run. Pi has no separate native
tool-execution approval policy, so this flag is not a yolo control; use only a
reviewed target and a controlled prompt. See [Bounded model behavior checks](#bounded-model-behavior-checks)
for the full evidence, timeout, and cleanup contract.

Setup syncs the repo into `.agents`, treats `.skills/` as the authoritative
source-managed skill tree, refreshes the managed links it owns inside `skills/`,
and then refreshes symlinks in `.claude/`, `.codex/`, and `.local/bin`. Scratch
`.temp/` trees are excluded at every source depth and removed from installed
runtimes, so nested development artifacts cannot become runtime content.

### Source tree resolution

`setup global` and `setup local` need a source tree to sync from. An installed
binary carries no path of its own, so it resolves one — first usable wins:

1. `--source-dir DIR`
2. `AGENTS_INFRA_SOURCE_DIR`
3. `repoPath` from the machine-scoped `install.json` the installer writes
4. the installed `~/.agents` runtime

Callers therefore do not need a host-specific checkout path:
`agents-infra setup local /path/to/project` works from a globally installed
binary.

#### What makes a tree usable

The contract is derived from what setup installs, not from a set of
recognisable file names. A usable source tree carries:

| Asset | Needed by |
| --- | --- |
| `.instructions/INSTRUCTIONS.md` | Claude instructions entrypoint |
| `.instructions/AGENTS.md` | rendered Codex instructions entrypoint |
| `.configs` | linked agent config tree |
| `.rules` | linked agent rules tree |
| `SKILL.md`, `README.md` | materialized `relux-agents-infra` skill package and its reference |
| `tools/agents-infra/go.mod`, `tools/agents-infra/main.go` | the Go module the generated local `agents-infra` launcher builds on every invocation |
| `tools/agents-infra/internal/infra/pi-v0.84.2-darwin-arm64-tree-manifest.txt` | authoritative 217-record managed Pi release-tree catalog; exact SHA-256 `2f68ab1b3f28a9c4b8995f91984f8f47001a79735da7e57aa7fe6d223f90378b` |

…plus every instruction module the entrypoints `@include`. That closure is
resolved up front, so a tree that references modules it does not ship is
refused before the destination is touched rather than failing half way through
the render.

The launcher backend is part of the contract for a concrete reason: a tree with
only the instruction and config markers used to pass, and setup would exit 0,
print a full install log, and leave a launcher that failed the first time it
ran.

For `setup local` and `verify local`, the launcher backend is not checked by
listing files, and not by compiling it either. The generated launcher runs
`go build .` in `SOURCE/tools/agents-infra` into its own output path and then
executes the result, on every invocation — so setup performs that whole
operation and requires the built binary to answer `version` as `agents-infra`.

Each narrower check was satisfied by a tree that then failed on first use. A
module can carry `go.mod` and `main.go` and still be missing the packages the
build needs. A module can compile cleanly into a program that exits non-zero, or
into a program that is not agents-infra at all. And a build to some unrelated
temporary directory says nothing about whether the launcher's real output path
can be written. A refusal carries the command output naming what failed.

The launcher builds into `PROJECT/.local/bin/.agents-infra-build/`, inside the
target it was installed for — never into the shared source checkout. That path
would otherwise be contended by every project installed from the same source,
would break for a read-only source, and could not be reproduced by verification
without mutating the tree being verified.

Two consequences worth knowing:

- `setup local` and `verify local` need a Go toolchain. The generated launcher
  already needs one on every invocation, so this is not a new requirement for
  the runtime — but it does mean a host without Go cannot install a local
  runtime instead of installing a broken one.
- Setup and `verify local` build to the launcher's own output path, so the
  launcher's first invocation finds an artifact that is already there. Nothing
  is written into the source tree.

`setup global` generates no launcher — the bootstrap owns
`~/.local/bin/agents-infra` — so it makes no claim about a build and does not
run one.

Both setup modes install sibling-only `pi-infra`, `openai-infra`,
`anthropic-infra`, and `qwen-infra` launchers: global setup writes
it beside the bootstrap-owned `agents-infra`, and local setup writes it beside
the generated project launcher. `pi-infra` delegates as `agents-infra pi`; the
three vendor aliases delegate as `agents-infra target <exact-alias>` without
changing the caller's working directory or argument bytes/order, including a
literal wrapper delimiter and its following operands. It never searches `PATH`
for a substitute target. Setup repairs a drifted managed alias; setup's
postcondition and `verify` refuse a missing alias, changed bytes or mode, a
wrong embedded target, and a missing/non-regular/non-executable sibling target.
The managed alias and sibling target must each be a regular file at its own
pathname: setup replaces a symlinked alias even when its target has identical
bytes, and verification rejects symlinks for either artifact instead of
following them.

Three things are refused instead of guessed:

- An explicit `--source-dir`/`AGENTS_INFRA_SOURCE_DIR` that is not an
  agents-infra tree fails and names every missing asset and the component that
  needs it. It never falls back to a discovered candidate, so a wrong path
  cannot install something else.
- A candidate that contains the destination is rejected rather than synced into
  itself.
- A tree that satisfies the markers but not the full contract is rejected, not
  installed part way.

When nothing resolves, the error lists every candidate and why each was
unusable.

### Verifying an installed runtime

Exit code zero and the existence of `.agents` are not evidence that a runtime
works. A setup run only marks its destination complete after its postconditions
pass, by writing `.agents/.agents-infra-install.json`. That receipt is dropped
before the run mutates anything and rewritten only at the end, so a run that
fails part way through leaves nothing vouching for what it wrote. Sync never
copies a receipt out of a source tree, and a receipt naming a different
destination is rejected.

```bash
agents-infra verify local /path/to/project
agents-infra verify global
```

`verify` re-runs the same postcondition: the receipt must have been minted for
this destination, the installed tree must carry every asset above, and the
generated launcher's recorded source must still produce a binary that starts —
`verify local` builds it to the launcher's recorded output path and runs it,
rather than checking that the module's files are present. A receipt on its own
proves nothing — it is always checked against the live artifacts.

The postcondition also hashes the installed Pi manifest and checks the exact
generated `pi-infra` body and sibling target. Therefore missing catalog bytes,
catalog drift, alias drift, or a missing target invalidates an otherwise intact
receipt.

Consumers that bootstrap a repo-local runtime should gate on `verify` rather
than on the exit status of `setup` alone.

Author shared changes in this source repo. Do **not** edit `~/.agents/`
directly.
The installed `~/.agents/` copy is runtime state and should not keep git metadata.

For project-local installs, use `agents-infra setup local /abs/path/to/project`.
That creates a local runtime layout under the project root:
- `.agents/`: the installed runtime copy and project-owned instruction space
- `.claude/`: thin Claude shim that points into `.agents`
- `.codex/`: thin Codex shim that points into `.agents`
- `.local/bin/`: helper CLIs for the local setup, including `agents-infra`
- `.local/bin/pi-infra`: managed sibling alias for `agents-infra pi`
- `.local/bin/openai-infra`, `.local/bin/anthropic-infra`, `.local/bin/qwen-infra`: canonical target aliases

Local setup reproduces the global runtime topology, not the global instruction
content. It does not copy source `.instructions/` modules into the project.
Instead it creates `.agents/.instructions/AGENTS.md` and
`.agents/.instructions/INSTRUCTIONS.md` as minimal project-owned entrypoints
when they are missing. Existing local entrypoints and modules are never
overwritten during resync. Add only project-specific guidance there; global
policy continues to come from the global agent runtime.

Project-local setup intentionally does not create `.codex/config.toml`. Codex
model, reasoning effort, service tier, trusted projects, and TUI notices are
owned by the global `~/.codex/config.toml` link by default. This prevents stale
project-local configs from silently overriding the current global model.

Both global and local source sync refresh the managed
`.agents/.configs/codex-config.toml` defaults while merging existing Codex
user state: project trust decisions, TUI notice acknowledgements, and custom
profiles other than the withdrawn `fast` profile. Source remains authoritative
for model, reasoning, and `service_tier = "default"`. A malformed installed
Codex config refuses synchronization without replacing that config.

During local setup in the default `preserve` mode, agents-infra removes managed
project-local Codex config artifacts it created: either the legacy symlink or a
rendered managed regular file. A custom project-local config is left in place
because project-specific model/reasoning overrides must be explicit and
intentional, not silently destroyed.

Use `--codex-config` when local setup should make an explicit decision:

- `--codex-config=preserve` keeps custom project-local config files and removes
  a managed local config from a prior explicit `local` setup. This is the
  default.
- `--codex-config=global` removes `.codex/config.toml`, making the global
  `~/.codex/config.toml` authoritative for the project.
- `--codex-config=local` renders a managed regular `.codex/config.toml` from
  `.agents/.configs/codex-config.toml`, making its supported settings
  project-local by explicit choice. The renderer omits the user-level-only
  top-level `profiles` table while preserving all other valid TOML settings.
  Invalid installed TOML fails before the existing project config is replaced.

These modes alone govern project Codex config state. A primary
`agents-infra codex` launch—or an external owner using the preparation contract
below—refreshes instructions, skills, and rules while preserving an absent,
managed, custom, or linked `.codex/config.toml` exactly.

### Child launch MCP composition contract

Automation that owns child process policy can request the project MCP subset
without launching Codex or Claude:

```bash
agents-infra compose --agent codex --project /abs/path/to/project --schema-version 1 --json
agents-infra compose --agent claude --project /abs/path/to/project --schema-version 1 --json
```

The command writes exactly one
`agents-infra.child-launch-composition` JSON document to stdout. Schema version
`1` contains the canonical project directory, build version/commit, an MCP-only
`argv_prefix`, ordered safe server provenance, and referenced environment
variable names. Codex receives only `-c mcp_servers.*` pairs. Claude receives
either an empty prefix or exactly one `--mcp-config` pair.

This is deliberately not a primary-session launch plan. The composition never
includes project model/reasoning/yolo policy, approval or permission flags,
provider user arguments, prompts, profiles, service tiers, or base Codex config
paths. It also never resolves a `bearer_token_env_var`: Codex receives the env
variable name and Claude receives the literal `Bearer ${ENV_NAME}` reference,
so no environment value is read or serialized.

Recognized composition failures return nonzero and emit a safe version-1 error
envelope with a stable `error.code`; human diagnostics go only to stderr.
Unsupported versions use `unsupported_schema_version`, while malformed or
invalid discovered project configuration uses
`invalid_project_configuration`. Consumers must reject a contract/version
mismatch rather than partially applying its arguments.

### Primary-session project preparation contract

Session owners that launch a primary provider outside the agents-infra process
must refresh the same installed project surface as the direct launcher:

```bash
agents-infra prepare --agent codex --project /abs/path/to/project --schema-version 1 --json
agents-infra prepare --agent claude --project /abs/path/to/project --schema-version 1 --json
```

The command is non-launching and board-agnostic. It walks from `--project`
toward the filesystem root, selects the nearest installed project `.agents/`
runtime, and never treats the user's global `~/.agents` runtime as a project.
Codex preparation refreshes the managed `.codex/AGENTS.md`, project-root
`AGENTS.md`, and skills/rules links. It never chooses a Codex config mode:
an absent `.codex/config.toml` stays absent, while an existing managed, custom,
or linked config is preserved byte-for-byte or target-for-target. The explicit
`setup local --codex-config=preserve|global|local` operation remains the only
owner of that choice. Claude preparation refreshes `.claude/CLAUDE.md`,
instruction/settings links, and managed skill links. When no project-local
runtime is installed, the command succeeds as an explicit no-op so the
provider-native/global surface remains authoritative.

Stdout contains exactly one
`agents-infra.primary-session-preparation` schema-version-1 JSON report. It
identifies the provider and canonical requested project, the selected
`runtime_project_dir`, whether a local runtime was present, verified
provider-state booleans such as
`codex_project_rendered`/`codex_config_generated`, and an ordered artifact list
with regular-file SHA-256 values or symlink targets. The Codex config artifact
is reported as `absent` or `preserved`; `codex_config_generated` describes an
existing preserved managed file and does not authorize rendering one.
Unsupported schemas and render failures return nonzero with a safe error
envelope.

`agents-infra codex` and `agents-infra claude` call this same preparation
function immediately before provider launch. `--print-config` remains a
read-only inspection path. External session owners must call `prepare` after
successful composition and before starting or connecting to their provider
host; composition still supplies MCP and primary-session policy through the
versioned launch plan.

### Task-board Session Manager wrappers and shell aliases

The primary launch path for a board-scoped session is the Task-board Session
Manager, not a direct `agents-infra` launch. `task-board codex` and
`task-board claude` are thin clients: each resolves launch policy exclusively
through `agents-infra compose --mode primary-session --agent codex|claude
--project DIR --schema-version 1 --json`, invokes the matching `agents-infra
prepare` contract, then owns the provider process, session/thread identity, and
native board-goal binding. agents-infra stays board-agnostic: it renders the
provider project surface and launch plan without reading board state.

Because of that, the recommended personal shortcut aliases now target the
task-board wrappers rather than the raw launchers:

```zsh
alias codexD="task-board codex"
alias claudeD="task-board claude"
```

Use `agents-infra codex` / `agents-infra claude` directly only for a launch that
must bypass the Session Manager (no board-goal binding, no managed session).

### Provider-specific primary session policies

Projects may set primary Codex and Claude sessions independently in
`.agents/.configs/project-config.toml`. This is optional policy for the
matching `agents-infra` launcher, not a replacement for either provider's own
configuration. A missing table—or a missing Codex individual field—leaves that
dimension to the provider-native project/profile/user/system/default resolution.

Pi additionally supports an exact project-local managed-profile contract. It
verifies the catalogued standalone Pi tree, derives hash-only isolated state,
owns the configured local-model runtime process group, and launches Pi only
after exact loopback model discovery. `agents-infra pi --print-config` and
`compose --mode primary-session --agent pi` emit the same schema-v1 plan without
creating state, acquiring a lock, opening a socket, or starting a process. With
no Pi policy, `agents-infra pi` is native passthrough; malformed or unreadable
policy fails closed. Managed launches reject `--export`, `--session-dir`, and
path-shaped `--session`/`--fork` selectors so Pi cannot read or write session
state outside the generated isolated session directory; ID selectors plus
`--continue` and `--resume` remain scoped to that directory.

Managed Pi diagnostics report Qwen text/tools and Muse text/tools/DFlash only
as requested or configured, with `verified: []` and `verification:
"not-claimed"`. Exact loopback preflight, direct-child liveness, and model
discovery do not prove that a reviewed runtime is internally honest or close
the same-UID post-preflight bind race; those two adversaries are explicit
non-claims, not launcher authorization evidence.

### Managed Pi local-model operator contract

Author Pi policy only in this source repository or a project's
`.agents/.configs/project-config.toml`; never patch the installed `~/.agents`
copy. Install and verify the command surface, then inspect the non-launching
plan before every new or changed deployment:

```bash
# Global runtime and alias.
agents-infra setup global --source-dir /path/to/relux-agents-infra
agents-infra verify global

# Project-local runtime and alias.
agents-infra setup local /abs/path/to/project
agents-infra verify local /abs/path/to/project

cd /abs/path/to/project
pi-infra --print-config
# Launch only after the plan has the intended profile, executable, literal
# runtime argv, endpoint, state keys, and exact Pi catalog identity.
pi-infra
```

`pi-infra` is installed by both setup modes and is exactly a sibling delegation
to `agents-infra pi`. It preserves the caller cwd and every argument in order.
The first ASCII `--` is a wrapper-only message boundary: it is removed before
Pi because pinned Pi has no end-of-options parser state, while safe suffix
operands are appended byte-for-byte. A second delimiter or a suffix beginning
with ASCII `-` or `@` is refused before the runtime starts.

The cycle-10 reference policy is exactly:

```toml
[agents.pi.primary_session]
profile = "qwen-3.8-27b"
pi_compatibility = "github-release:earendil-works/pi@v0.84.2:darwin-arm64#sha256-c996e888b7f7dce44bcf24f69176ac646c44139d3916bd49a6b28e5a8c5e3a65"
yolo_mode = false

[agents.pi.profiles."qwen-3.8-27b"]
provider = "local-qwen"
model = "Qwen-3.8-27B"
base_url = "http://127.0.0.1:18011/v1"
api = "openai-completions"
reasoning = true
input = ["text"]
context_window = 131072
max_tokens = 16384
thinking = "medium"
requested_capabilities = ["text", "tools"]

# Optional. If present, every field is required. This profile-managed policy
# is merged into the isolated Pi settings before every launch.
[agents.pi.profiles."qwen-3.8-27b".compaction]
enabled = true
compact_at_tokens = 106496
keep_recent_tokens = 8192

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

# Optional. If this table is absent, the existing exclusive direct-child
# runtime behavior is unchanged. If present, every field is required.
[agents.pi.profiles."qwen-3.8-27b".runtime.sharing]
mode = "shared"
linger_seconds = 15
max_leases = 8
max_segment_bytes = 104857600
max_segments = 7
heartbeat_interval_seconds = 5
lease_stale_seconds = 30
restart_limit = 3
restart_initial_backoff_seconds = 1
restart_max_backoff_seconds = 4
stable_run_seconds = 10
quarantine_seconds = 30
broker_start_timeout_seconds = 160
resource_pressure_mode = "disabled"

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

The model names, runtime executable, argv, ports, limits, and timeouts are
operator-supplied deployment inputs. agents-infra does not acquire, convert,
quantize, license, size, or securely distribute models or runtimes. Its bounded
synthetic prefill probe is a host-capacity check, not a throughput, quality, or
production benchmark.

`model-harness` is the machine-facing runtime boundary for new deployments. It
lives in this repository as a separate binary so it can later move to its own
repository without changing the agent-facing contract. Its config is separate
from `project-config.toml` and defaults to the platform user config directory:

- macOS: `~/Library/Application Support/model-harness/config.toml`
- Linux: `${XDG_CONFIG_HOME:-~/.config}/model-harness/config.toml`

`MODEL_HARNESS_CONFIG` or `--config /absolute/path` selects another file. A
local profile replaces exact `{host}` and `{port}` argv tokens and starts the
backend without a shell:

```toml
[profiles.qwen-local]
mode = "local"
executable = "/absolute/path/to/python"
argv = ["-c", "from mlx_lm.server import main; main()", "--model", "/models/Qwen", "--host", "{host}", "--port", "{port}"]

[profiles.qwen-local.supervision]
fatal_output_substrings = ["RuntimeError: [metal::malloc] Resource limit ("]
restart_on_failure = true
max_restarts = 3
restart_window_seconds = 3600
restart_delay_milliseconds = 1000

[profiles.qwen-local.stress]
prompt_tokens = 50000
max_output_tokens = 1
startup_timeout_seconds = 120
request_timeout_seconds = 600
sample_interval_milliseconds = 250
```

Local supervision is an explicit fail-closed policy. The harness forwards the
child's stdout and stderr unchanged while matching literal fatal substrings,
kills a runtime that remains listening after a fatal background-thread error,
and starts a fresh child within the configured rolling restart budget. A normal
zero exit is never restarted. `restart_on_failure` additionally covers an
unexpected non-zero child exit. Supervision is configured on the remote
machine's local profile rather than on the local SSH forwarding profile.

As of 2026-08-27, the released `mlx-lm 0.31.3` predates the upstream fix for
the Qwen3.5 `ArraysCache` Metal buffer-object leak. A reproducible temporary
installation can pin the merged upstream fix without replacing the released
environment:

```bash
pipx install \
  --suffix=-qwenfix \
  --python python3.14 \
  'git+https://github.com/ml-explore/mlx-lm.git@11a6ce75589f59809d6d79b28efa03c50896c18b'
```

Point the local profile's `executable` at the resulting absolute
`mlx_lm-qwenfix.server` path. Remove this temporary pin after a released
`mlx-lm` version containing upstream PR #1632 has passed the same capacity and
long-session checks.

All stress bounds are explicit profile policy. `model-harness stress` refuses
profiles without this table and currently supports local mode only. It starts
the reviewed backend on an unoccupied loopback endpoint, discovers the exact
model ID, calibrates a repeated synthetic token unit against real API usage,
prefills approximately `prompt_tokens`, requests only `max_output_tokens`, and
samples the backend process RSS. Its versioned JSON report contains observed
tokens, target delta/tolerance, startup and prefill time, baseline/peak RSS,
physical host memory, and peak RSS as a percentage of host memory. The runtime
is killed and reaped before the command returns, including failure paths.

A remote profile starts the same CLI over SSH and forwards the remote loopback
endpoint to local loopback:

```toml
[profiles.qwen-remote]
mode = "ssh"
ssh_target = "dedicated-mac"
remote_executable = "/Users/remote/.local/bin/model-harness"
remote_config = "/Users/remote/.config/model-harness/config.toml"
remote_profile = "qwen-tiny"
remote_host = "127.0.0.1"
remote_port = 18011
```

Both presented endpoints are strictly `127.0.0.1`. The remote backend also
binds only to its loopback interface; SSH is the transport and access-control
boundary. Remote runs force a dedicated SSH PTY so disconnect delivers HUP to
the foreground remote process group instead of orphaning a model server. The
MVP intentionally refuses direct LAN/public exposure of MLX or llama.cpp HTTP
servers.

Inspect before launch:

```bash
model-harness render qwen-local --host 127.0.0.1 --port 18011 --json
model-harness doctor qwen-remote --host 127.0.0.1 --port 18011
model-harness stress qwen-local --host 127.0.0.1 --port 18011 --json
```

An agents-infra Pi profile can use the harness without a new launcher mode:

```toml
[agents.pi.profiles."qwen-harness".runtime]
executable = "/Users/example/.local/bin/model-harness"
argv = ["run", "qwen-local", "--host", "127.0.0.1", "--port", "18011"]
readiness_path = "/models"
startup_timeout_seconds = 120
shutdown_timeout_seconds = 10
```

`runtime.sharing` is an explicit, strict opt-in. Its table has no field
defaults: unknown or missing members fail closed, `mode` is `exclusive` or
`shared`, the heartbeat interval must be below the stale-reporting threshold,
and the broker-start timeout must cover startup, shutdown, and 30 seconds of
coordination overhead. `mode = "exclusive"` uses the established direct-child
path. `mode = "shared"` gives each tracked RUN its own Pi state, session, lock,
and process group while byte-identical profiles can lease one broker-owned MLX
runtime across independent launcher sessions and project roots.

Shared runtime output is bounded by the operator-supplied `max_segment_bytes`
and `max_segments`; neither has a code default. The broker splits writes at the
exact byte boundary, keeps `runtime.log` as the active segment, archives full
segments with a monotonic sequence and UTC timestamp, and prunes the oldest
archive first. On restart, any managed archive larger than the current cap is
refused before runtime launch rather than admitted from an older configuration.
The active file and retained archives therefore occupy at most
`max_segment_bytes * max_segments` bytes of runtime output. A pre-existing
active segment or managed archive larger than the configured cap is refused
rather than silently discarded or reported as bounded.

`compaction` is also an explicit, strict opt-in. Configure exactly one threshold:
prefer the operator-facing `compact_at_tokens`, or retain the Pi-native
`reserve_tokens` compatibility form. agents-infra derives the other value as
`context_window - threshold`. The resulting reserve must be at least
`max_tokens`, and `keep_recent_tokens` must be below the compaction threshold.
Pi starts automatic compaction when current context exceeds the configured
threshold, then preserves approximately `keep_recent_tokens` of the newest
conversation while summarizing older work. Smaller retained tails and an
earlier threshold are appropriate for local models that must keep one session
alive across many days. agents-infra writes Pi's native `reserveTokens` and
`keepRecentTokens` fields into the profile's isolated `agent/settings.json`,
preserving unrelated Pi preferences, and fails without overwriting when the
existing settings file is malformed or unsafe.

The first broker fixes the effective sharing policy for its lifetime. Inspect
both configured and effective values, the attested broker/runtime identities,
and live leases without starting or connecting from `pi --print-config`:

Shared profiles also require explicit restart supervision values. Runtime
failures increment a mode-0600 `restart-ledger.json` under the resolved
shared-runtime root. Each elected broker honors the persisted, exponentially
increasing `restart_initial_backoff_seconds` delay capped by
`restart_max_backoff_seconds`; a run that remains ready for
`stable_run_seconds` resets the count. Reaching `restart_limit` quarantines the
runtime for `quarantine_seconds`; the next broker attempt after that deadline
is the automatic half-open probe. `runtime quarantine` and
`runtime unquarantine` mutate the same ledger only while no broker owns it, so
stop an active broker before changing manual quarantine. Status JSON always
includes `restart_count`, `restart_not_before`, `quarantined_until`,
`last_readiness_match`, `manual_quarantine`, and `half_open`.
`restart_not_before` is the ledger's exact RFC3339 deadline or JSON `null`; it
is never synthesized from `restart_count`, readiness history, or broker state.
A non-zero restart count is historical and may coexist with a serving runtime.
`half_open` is also copied directly from the ledger. It is lifecycle evidence,
not an availability gate: it can remain true after readiness while the stable
run timer is still pending.

The consumer handoff is exact and presence-aware. A post-extension parser must
require the `restart_not_before` key, accept only JSON `null` or a valid RFC3339
timestamp, and refuse malformed timestamps as a failed status read. A
pre-extension fixture with the key absent is compatibility input with no
backoff-deadline evidence; it must not fall back to `restart_count`. Given a
successful read observed at `checked_at`, only a non-null deadline strictly
after `checked_at` maps to:

```go
vendorplugin.LimitedUntil(
    restartNotBefore,
    vendorplugin.Observation{
        Source: "agents-infra runtime status --json.restart_not_before",
        Detail: "shared runtime restart backoff deadline",
        At: checkedAt,
    },
)
```

That shape is validator-safe: `Until` is evidence-derived and non-zero, while
`Checked` and `Observed` are populated by `LimitedUntil`. A missing legacy key
maps to `vendorplugin.UnknownAfterCheck` for that source. A `null` or elapsed
deadline contributes no backoff verdict; consumers continue with attested
broker/runtime facts, so a serving runtime is never relabeled from a historical
restart count. `half_open` alone likewise contributes no availability verdict.

`last_failure` and `last_failure_at` are explicitly deferred and absent from
this status schema. Restart-ledger v1 persists neither a failure reason nor a
failure timestamp, so publishing placeholders or deriving them from another
field would fabricate provenance. Adding them requires a separately reviewed
ledger/event write contract and new pre/post fixtures.
Every configured seconds field is bounded before conversion to `time.Duration`;
coupled handoff and doubled lease-stale windows are bounded as effective
durations too, so overflow is refused during config resolution before launch.

Shared profiles must also make an explicit resource-pressure choice. The
`disabled` value in the example preserves legacy admission but publishes
resource facts as `unknown` with `admission = "not-enforced"`; it is an
explicit operator opt-out, not a default. To enforce provider-backed
observation, replace it with `resource_pressure_mode = "provider"` and add the
complete strict table:

```toml
[agents.pi.profiles."qwen-3.8-27b".runtime.sharing.resource_pressure]
observation_path = "/agents-infra/resources"
observation_timeout_milliseconds = 250
pressure_threshold_bytes = 50000000000
recovery_threshold_bytes = 45000000000
eviction_grace_seconds = 15
pressure_action = "refuse-new-drain-idle"
unknown_action = "refuse-new"
busy_action = "observe"
```

There are no threshold or action defaults. Recovery must be below pressure to
provide explicit hysteresis. The only admitted actions preserve existing
connection-bound leases: pressure refuses new leases, waits for the final lease
to release, then drains and evicts after `eviction_grace_seconds`; an unknown
observation refuses a new lease but never guesses that eviction is safe; busy
is an independently observed fact and is not inferred from lease count.

The configured absolute observation path must return at most 16 KiB of strict
JSON before `observation_timeout_milliseconds` expires:

```json
{
  "schema": "agents-infra.pi.shared-runtime.resource-observation.v1",
  "model": "Qwen-3.8-27B",
  "loaded_model_memory": {"state": "observed", "bytes": 47244640256},
  "inference": {"state": "busy"}
}
```

Each provider fact may instead use `state = "unknown"` with a non-empty
`reason` and no value. A failed, partial, oversized, wrong-schema, wrong-model,
or malformed response is also `unknown`; it is never treated as absence or
healthy. Protocol v7 carries the versioned
`agents-infra.pi.shared-runtime.resource-status.v1` consumer handoff. Runtime
status always publishes independent loaded-model-memory and inference facts,
the effective thresholds/actions, admission, source, and one aggregate state:
`healthy`, `busy`, `pressured`, `draining`, or `unknown`.

An attested live broker is the authoritative source for that effective policy.
When the broker cannot be reached but its persisted record is readable, status
keeps `source = "record-derived-unverified"` and derives both
`sharing.effective` and `resources.policy` from the same recorded sharing
value. Provider observations remain explicitly unknown/refused. A
pre-extension record without sharing evidence reports
`resources.policy.mode = "unknown"`, reason
`resource_pressure_policy_unknown`, and refused admission; caller
configuration is never substituted as a guess for effective enforcement.

Admission and diagnostic provider reads use separate monotonic broker-local
generations. Healthy or busy status polling advances only diagnostic freshness,
so observability cannot repeatedly supersede a healthy lease at its reservation
boundary. Direct pressure observed by status still advances admission
invalidation and prevents an older healthy observation from granting a lease;
status does not take ownership of the pressure latch or eviction timer.

Immediately before building a status wire response, the broker revalidates the
diagnostic and admission generations while atomically snapshotting the pressure
latch, broker state, and leases. A superseded response is explicit
`unknown/refused` with `resource_observation_stale`, never stale
`healthy/admitted`. The admission generation check and lease reservation share
the same broker lock, and pressure/recovery events carry the admission
generation that changed the latch, so completion, publication, or event
scheduling cannot reverse a newer admission decision.

Because sharing policy is not part of the runtime identity, a caller may find
an already-running broker with different settings. Status keeps both
`sharing.configured` and `sharing.effective` visible, but lease acquisition
requires the resource-pressure mode and complete policy table to match exactly.
The compared table includes observation path and timeout, pressure and recovery
thresholds, eviction grace, and all three actions. Any single-field difference,
including either disabled/provider direction or a recovery-threshold-only
change, refuses before provider observation or lease reservation with
`shared_runtime_resource_policy_mismatch`; the existing broker/runtime and its
leases are preserved.

When a profile change produces a new runtime identity while the previous
shared runtime is still releasing the same fixed listener, acquisition retries
the broker's structured `runtime_listener_occupied` exit. The same bounded
handoff applies when an installed agents-infra upgrade leaves an older broker
inode alive long enough to refuse the new client with
`broker_executable_identity_mismatch`. Both retries last at most the configured
linger plus shutdown timeout and a two-second handoff grace. The client never
adopts or signals the old or occupying runtime; a refusal that persists after
that bounded window still fails closed.

```bash
agents-infra pi --print-config --profile qwen-3.8-27b
agents-infra runtime status --profile qwen-3.8-27b
agents-infra runtime status --profile qwen-3.8-27b --json
agents-infra runtime stop --profile qwen-3.8-27b
agents-infra runtime stop --profile qwen-3.8-27b --force --timeout 30
agents-infra runtime quarantine --profile qwen-3.8-27b
agents-infra runtime unquarantine --profile qwen-3.8-27b
```

Ordinary stop refuses while leases are active. Forced stop first attests a
reachable broker; if rendezvous is unavailable, it verifies the recorded
broker against the kernel before signalling it and reuses the broker's strict
orphan-reclamation checks. A held election lock without an attributable owner
is reported as `starting-unverified` and is never guessed or signalled.

#### Composition, identity, and CLI precedence

Configs compose from filesystem root to cwd. The nearest explicit
`primary_session.profile` wins unless wrapper `--profile NAME` selects a
profile; `pi_compatibility` and `yolo_mode` have nearest-field precedence and
no CLI override. An explicit child `yolo_mode = false` masks an inherited
`true`.
One child definition of the same exact decoded profile name atomically replaces
the ancestor definition—profile fields never merge—while a child may select a
complete ancestor profile. Unreadable, malformed, partial, or unknown-field
policy is a failure, not policy absence; only genuine absence enables native Pi
passthrough.

Pinned Pi `0.84.2` exposes no per-tool approval policy: enabled tools execute
without a confirmation prompt. Its `--approve`/`-a` flag controls one-run trust
for project-local files. Therefore omitted or explicit `yolo_mode = false`
preserves native project-trust behavior, while `yolo_mode = true` injects
exactly one `--approve` so project `AGENTS.md`, skills, extensions, and other
reviewed local resources load without a trust prompt. Explicit
`--no-approve`/`-na` conflicts with effective yolo and fails closed.

That primary-session rule is intentionally separate from the board-agnostic
standalone worker contract. To authorize an unattended worker explicitly,
configure both fields in the same composed project policy:

```toml
[agents.pi.primary_session]
profile = "qwen-3.8-27b"
pi_compatibility = "github-release:earendil-works/pi@v0.84.2:darwin-arm64#sha256-c996e888b7f7dce44bcf24f69176ac646c44139d3916bd49a6b28e5a8c5e3a65"
yolo_mode = false

[agents.pi.standalone_session]
yolo_mode = true
tool_allowlist = ["read", "bash", "edit", "write", "grep", "find", "ls"]
```

The required managed profile and canonical `qwen-infra` target definitions are
the same reviewed deployment inputs described by the reference policy and
canonical-target contract in this surrounding operator section; standalone
authorization does not duplicate or weaken them.

Then inspect or launch only through the owned standalone command:

```bash
qwen-infra spawn --prompt "Complete the bounded task" --print-config
qwen-infra spawn --prompt "Complete the bounded task" --deadline 10m
```

Standalone policy is fail-closed and does not change interactive
`qwen-infra`/`pi-infra` behavior. Both `yolo_mode = true` and a non-empty exact
built-in allowlist are required. The launcher owns Pi's `--no-approve`,
`--no-extensions`, `--tools`, `--mode json`, `--no-session`, and `--print`
arguments, closes stdin, refuses every caller-controlled Pi argument, and
keeps the prompt out of diagnostics. `--no-approve` declines project trust;
tool authorization comes from the wrapper's reviewed built-in allowlist, not
from Pi project trust. Raw RPC mode is never exposed because its direct `bash`
request bypasses Pi's model `tool_call` hook. Extension discovery is disabled
so a project or global extension cannot replace an allowed built-in tool.

Each invocation owns an independent Pi process group and a fresh random,
hash-contained, non-persistent client state. With `runtime.sharing.mode =
"shared"`, concurrent standalone workers lease one verified local runtime;
releasing or crashing one worker releases only its lease, and the final lease
reaps the runtime. The command is deliberately board-agnostic: no task-board
runtime adapter, registration, run identity, or lifecycle dependency is
implemented here. A board adapter may later consume this stable primitive.

Profile-name identity is its exact post-TOML-decoding UTF-8 bytes. There is no
normalization, case folding, trimming, path cleaning, or lossy sanitization.
Explicit `--provider`, `--model`, and `--thinking` must resolve to the exact
managed identity; equal repeats normalize and conflicts fail. `--api-key` is
redacted in diagnostics, and an explicit `--approve` remains idempotent with
effective yolo. Wrapper-recognized equal forms normalize
to Pi's pinned spaced syntax. Options cannot consume a value across the removed
delimiter; ambiguous unknown flags and option-looking message suffixes fail.
The resulting argv has one provider/model selection, no fake separator, no
option after operand content, and preserves accepted message bytes/order.

The managed provider is `openai-completions`, input is exactly `['text']`, and
the endpoint must be literal `http://127.0.0.1:<nonzero-port>/v1` with no user
info, query, or fragment. `localhost`, IPv6, wildcard, and remote endpoints are
refused. Runtime argv must also contain exactly one spaced `--host 127.0.0.1`
pair and exactly one spaced `--port <base_url-port>` pair. Missing, duplicate,
attached (`--host=...`/`--port=...`), wildcard, or divergent endpoint options
are refused by the same production resolver used for compose, diagnostics, and
launch. The runtime executable is an absolute literal reviewed path and argv is
a non-empty literal token vector: no shell, `PATH` lookup, interpolation,
globbing, tilde expansion, command substitution, or implicit flag injection.
For Muse, target and draft token subsequences must each occur exactly once,
contiguously and without overlap, and end in their declared model. This proves
the launched argv, not active DFlash.

#### Non-launching diagnostics and contained state

`pi-infra --print-config` and
`agents-infra compose --mode primary-session --agent pi --project "$PWD"
--schema-version 1 --json` run the same resolution and static validation. They
do not run Pi/runtime, create state or locks, bind/connect a socket, download,
or mutate Pi settings, auth, or trust. An uninspectable value is `unknown` or an
error, never absence. The plan reports exact sources, normalized Pi argv,
runtime executable/argv and static state, loopback readiness URL, timeouts,
generated `models.json` digest/path, catalog identity, requested capabilities,
and hash-only state/lock/log paths. Secret values and arbitrary environment
values are omitted.

The emitted contract is `agents-infra.primary-session-launch-plan` schema 1
with provider `pi`. `launch_variants.interactive.argv` is the exact normalized
Pi argv; `managed_host.kind` is `pi-pty` with the same argv; managed-client argv
is empty. The generated one-provider/one-model `models.json` uses fixed
non-secret dummy key `agents-infra-local`, zero costs, and configured metadata;
it contains no executable credential command, header, secret, or environment
reference.

Let `profile_bytes` be the exact UTF-8 profile-name bytes. The profile key is
`lowercase_hex(SHA256(profile_bytes))`; the project key is
`lowercase_hex(SHA256(UTF8(canonical_project_path)))`. State is only:

```text
<canonical-cache-root>/agents-infra/pi/<64-hex-project-key>/<64-hex-profile-key>/
  agent/models.json
  agent/settings.json  # present/managed when profile compaction is configured
  sessions/
  logs/<UTC-start>-<random>.jsonl
  session.lock
```

Each managed launch creates a distinct mode-`0600` lifecycle log. It records
session/runtime/Pi start, PID and process-group identity, runtime readiness,
foreground-terminal ownership, exit or received signal, and bounded cleanup.
It never records environment values, API keys, or prompt/argument contents.
Pi's own conversation and tool transcript remains in `sessions/`; the launcher
log exists to diagnose orchestration failures such as a TUI child stopped by
terminal job control. The launcher prints the exact log path before starting
Pi, and `PiRunReport.session_log` carries it for managed callers.

Resume from the same canonical project directory so the project/profile state
keys resolve to the same isolated session directory:

```bash
cd /the/original/project

# Continue the newest session directly.
qwen-infra -- --continue

# Open Pi's session selector.
qwen-infra -- --resume

# Resume one exact session ID from the JSONL filename.
qwen-infra -- --session 01a03e8e-7c6d-7973-876a-a392202cdd57
```

The first `--` belongs to the canonical-target launcher and is not forwarded.
Compaction is lossy only for active model context: Pi retains the complete
conversation/tool JSONL, so resume does not require restating the task. A
different cwd or profile intentionally resolves another isolated session tree.

The canonical cache root is a successfully resolved absolute
`os.UserCacheDir()`. Raw profile text is never a path component. `/`, `\\`,
`.`, `..`, `../qwen`, `nested/../qwen`, absolute-looking names, case variants,
Unicode separator lookalikes, and NFC/NFD variants remain byte-distinct names
and keys. Before side effects, the launcher refuses any byte-distinct key
collision, proves the exact four-component suffix, and walks/creates it from an
opened cache-root handle without following managed symlinks. Existing and
created components are reopened and revalidated. Cache lookup,
canonicalization, stat/open/read, containment, collision, symlink, type,
permission, partial-read, or revalidation failures fail closed. Distinct names
have independent `session.lock` files and cannot intentionally share catalog or
session state. The production-entry `TestPiLaunchProfileStateKeyIsolation`
guards this boundary; narrowing to raw names, slash replacement, case folding,
Unicode normalization, or another lossy key must make it fail.

#### Standalone Pi catalog

Managed launch accepts only the official `v0.84.2` darwin-arm64 standalone
closure selected by the exact compatibility ID above. Its asset checksum is
`c996e888b7f7dce44bcf24f69176ac646c44139d3916bd49a6b28e5a8c5e3a65`;
the native arm64 Mach-O entrypoint is `<release-root>/pi` with SHA-256
`d5de3fe32f9e109324f32d6e393554fb2ce10bbc82e8ff935ab2e072f5e2f044`.
The release archive contains exactly one top-level `pi/`; the installed
`<release-root>` is the canonical parent of the fully resolved entrypoint, not
a trusted directory name.

The authoritative attachment is
`TASK-260817-3a0zr3_pi-v0.84.2-darwin-arm64-tree-manifest.txt`; the shipped
source copy is
`tools/agents-infra/internal/infra/pi-v0.84.2-darwin-arm64-tree-manifest.txt`.
It contains exactly 217 exhaustive regular-file records and hashes to
`2f68ab1b3f28a9c4b8995f91984f8f47001a79735da7e57aa7fe6d223f90378b`.
Paths are exact UTF-8, slash-separated, unsigned-byte sorted in C-locale order,
with records encoded as `<64 lowercase hex><two spaces>./<path><LF>` and a
required final LF. NUL/CR/LF/backslash, empty/`.`/`..` components, absolute or
escaping paths, and case/normalization changes are invalid.

The permitted directory set is exactly the root plus proper-prefix closure: 34
directories. Every file is regular with link count one; symlinks, hard-link
aliases, sockets, FIFOs, devices, mount crossings, extras, and omissions fail.
Directory mode is `0755`; `./pi`,
`./examples/extensions/doom-overlay/doom/build.sh`,
`./examples/extensions/doom-overlay/doom/build/doom.wasm`, and
`./native/darwin/prebuilds/darwin-arm64/darwin-modifiers.node` are `0755`; the
other 213 files are `0644`. Ownership, timestamps, ACLs, and xattrs are not
identity inputs and cannot excuse unreadability. Asset digest, complete path
inventory, types, links, modes, manifest bytes/digest, file count, entrypoint
digest, and Mach-O kind are one compiled catalog entry; project TOML cannot
replace any member. The full gate runs before managed side effects and again
immediately before Pi spawn with the same canonical root/entrypoint required.
Missing, partial, raced, or changed point-of-use reads fail.

#### Runtime lifecycle, capabilities, and operator verification

After all static gates, launch acquires the profile lock, atomically writes the
isolated `agent/models.json` as `0600`, and, when configured, safely merges the
profile compaction policy into isolated `agent/settings.json`. It preserves
unrelated profile-local settings, trust, auth, and sessions without reading or
writing normal `~/.pi/agent`. It exclusively preflights the exact loopback port and refuses an
occupied listener without connecting. It then closes the probe, rechecks the
runtime path, and starts `[runtime.executable] + runtime.argv` directly as a
direct child leading a new owned process group.

Startup requires that direct child to stay alive and the exact readiness URL to
return OpenAI list JSON containing the exact configured model (Muse requires
the exact target). Connection failure and HTTP 503 Service Unavailable retry
until timeout because llama.cpp uses 503 while loading; every other non-200
response, malformed success, missing data, wrong model, or child exit is fatal.
A ready foreign listener does not compensate for a dead selected child. After readiness the Pi catalog and
environment are rechecked, then only the captured absolute standalone Pi path
starts with isolated `PI_CODING_AGENT_DIR`/session state,
`PI_SKIP_VERSION_CHECK=1`, and `PI_TELEMETRY=0`. Duplicate or runtime-affecting
`DYLD_*`, `LD_*`, `NODE_*`, `BUN_*`, or `LLAMA_ARG_*` environment names are
refused before llama.cpp starts. `HF_ENDPOINT` and `MODEL_ENDPOINT` are refused
before llama.cpp starts because they can redirect `-hf` model resolution while
leaving the reviewed argv and model ID unchanged. Exact `GGML_BACKEND_PATH` is
refused before managed state or runtime spawn because llama.cpp build 10470
passes its inherited value to `dlopen()` during backend discovery. Other
`GGML_*` names remain outside this exact-name policy; they require an established
runtime effect before denial. Refusals identify only the quoted environment name
and never expose its value. Exact `LLAMA_API_KEY` is refused before managed
state or runtime spawn because llama.cpp build 10470 uses it as the environment
backing for `--api-key`; managed profiles must not acquire ambient runtime
authentication that is absent from reviewed profile configuration. `HF_TOKEN`,
cache-location variables, and unrelated environment names remain admitted;
`LLAMA_API_KEY` values are never reported. Managed
`--export`, `--session-dir`, and path-shaped `--session`/`--fork` are refused;
plain IDs, `--continue`, and `--resume` remain isolated.

If Pi exits, spawn fails, the runtime exits, or SIGINT/SIGTERM arrives, the
launcher forwards/terminates as appropriate, reaps the entire runtime process
group, escalates after the configured shutdown timeout, and releases the lock
only after owned processes are reaped. It never intentionally attaches to an
existing listener or silently chooses another profile, runtime, port, model,
listener, or Muse target-only mode.

Qwen `text`/`tools` and Muse `dflash`/`text`/`tools` are
requested/configured—not independently verified. Runtime reports, child PID,
argv, logs, throughput, and `/v1/models` are unverified provenance and never
populate `capabilities.verified`. Muse admission requires the exact configured
target/draft argv, live selected child, and exact target readiness; agents-infra
does not invent an attestation API and cannot detect a trusted runtime silently
disabling DFlash.

Operator acceptance must therefore record runtime/version/environment and run:

1. One real Pi text response and one function-tool call/result round trip for
   Qwen.
2. One real Pi text response and one function-tool call/result round trip for
   Muse.
3. A runtime-specific Muse benchmark or telemetry check that distinguishes
   target-only decoding from DFlash.

That evidence verifies the deployment; benchmark automation and interpretation
remain operator work and do not authorize later launches.

#### Bounded model behavior checks

Use `model-check` after setup/verification and `qwen-infra --print-config` to
exercise a configured canonical target through the real managed Pi lifecycle:

```text
agents-infra model-check --target ENTRYPOINT --prompt TEXT --output-dir DIR \
  [--deadline DURATION] [--expect-tool NAME]... [--expect-text TEXT]...
```

Run it from the project whose config and files the model should see; the caller
working directory is the project directory. `--target` is a configured
canonical entrypoint such as `qwen-infra`, not a provider executable or model
name, and it must resolve to a managed local Pi profile. `--expect-tool` matches
an exact tool name. `--expect-text` matches a substring of the final assistant
response only. Both flags are repeatable. With no expectations, exit `0` proves
only a clean, complete managed lifecycle, not a particular model behavior.

For a skill-discovery smoke, use a fresh output directory for every run:

```bash
agents-infra setup global --source-dir /path/to/relux-agents-infra
agents-infra verify global

cd /abs/path/to/project
qwen-infra --print-config

agents-infra model-check \
  --target qwen-infra \
  --prompt 'Discover the applicable installed skill for updating shared agent infrastructure. Use the read tool to read its SKILL.md. Reply with RELUX_SKILL_READ_CONFIRMED and one source-of-truth rule learned from that file.' \
  --output-dir .temp/model-check/qwen-skill-discovery-01 \
  --deadline 5m \
  --expect-tool read \
  --expect-text RELUX_SKILL_READ_CONFIRMED
```

The checker creates or secures `DIR` as `0700` and writes four new regular
files as `0600`. It refuses to overwrite any of these names, so reuse requires
a different directory:

| Artifact | Contents | Handling |
| --- | --- | --- |
| `events.jsonl` | Raw Pi JSONL provider/tool event stream | Sensitive raw evidence; may contain prompts, tool arguments, tool results, or secrets. |
| `stderr.log` | Raw managed runtime and Pi stderr | Sensitive raw diagnostics; do not publish without review and redaction. |
| `summary.json` | Schema-1 machine-readable outcome, expectations, event/tool counts, target identity, timing, and cleanup report | Sanitized; the final response is capped at 4096 bytes and its full SHA-256 is recorded. |
| `summary.txt` | Deterministic human-readable rendering of the same bounded summary | Sanitized; also printed to stdout when a summary is produced. |

`--expect-tool read` proves that an exact-name `read` tool event occurred; it
does not prove which file was read. To claim that the skill was discovered and
read, inspect `events.jsonl` for a completed, non-error `read` of the installed
`relux-agents-infra/SKILL.md`, then persist only a sanitized projection plus
`summary.json`/`summary.txt`. A response marker alone is self-reported evidence,
not proof of the tool target. A failed, partial, or malformed read is not
legitimate absence; report the result as failed or unknown.

The default managed-execution deadline is `5m`; accepted Go duration values are
`1ms` through `30m`. It covers managed runtime readiness and the Pi agent run
after static target resolution and output preparation. On expiry, the checker
cancels the managed lifecycle and runs bounded TERM-to-SIGKILL cleanup for its
owned Pi and runtime process groups, using the profile's
`shutdown_timeout_seconds`, before releasing the profile lock and returning.
Evidence files remain for the operator; the checker does not delete them.
`summary.json` reports
`deadline_ms`, `duration_ms`, `timed_out`, both process-group cleanup states,
and `cleanup_confirmed`.

Exit semantics are stable and ordered:

| Exit | Meaning |
| ---: | --- |
| `0` | Complete valid stream, managed cleanup confirmed, no failed tools, and every supplied expectation met. |
| `1` | Target/launch/validation/assistant/managed-cleanup failure. Early option validation may fail before summary artifacts exist. |
| `2` | Managed-execution deadline expired. The summary separately reports whether cleanup was confirmed. |
| `3` | Provider JSONL is malformed or an otherwise successful process produced an incomplete agent lifecycle. |
| `4` | One or more `--expect-tool` or `--expect-text` assertions were not observed. |
| `5` | A tool execution completed with `isError=true`; this takes precedence over unmet expectations. |

Provider stdout/stderr is never mirrored to the terminal. Sanitized summaries
redact recognized secret shapes, but an operator must still inspect them before
attaching or publishing evidence. The checker supplies Pi `--approve` only to
load reviewed project-local inputs. Pi tools have no separate native approval
policy, so this is not a yolo selection; keep prompts bounded and run only
reviewed targets in controlled projects.

#### Security boundary, diagnostics, and failures

The reproducible guarantees are exact config provenance, standalone Pi closure
and parser identity, reviewed absolute runtime executable plus literal argv,
shell-free spawn, exact loopback preflight, direct-child liveness, exact model
discovery, owned group cleanup, isolated state, redacted diagnostics, and no
intentional attach/fallback. The selected runtime code is trusted policy.
Explicitly excluded threats are a malicious/compromised selected runtime and a
malicious same-UID process winning the bind race after preflight closes and
before the runtime binds, plus compromised kernel/platform libraries or
same-UID mutation outside the retained Pi point-of-use checks. Preflight plus
readiness is not listener attestation, cryptographic ownership, runtime honesty,
tool verification, or DFlash verification.

There is no project-config surface or launch gate for model acquisition,
conversion, benchmark automation, secure runtime distribution, backend
catalogs, compiled observers, adapter/proxy layers, private-pipe authorities,
runtime/DFlash attestation, nonce, or cryptographic ownership. A selected
runtime cannot self-mint authoritative HTTP/stdout/config/argv/environment
evidence. Unknown or unverified remains diagnostics-only and never becomes a
satisfied capability gate; a no-observer deployment is supported precisely as
configured/unverified, not rejected or upgraded to verified.

Managed failures are named and fail closed without fallback:

| Error | Meaning |
| --- | --- |
| `invalid_project_configuration`, `invalid_provider_arguments`, `unsafe_pi_operand` | Invalid TOML/domain, Pi arguments, delimiter, or message boundary. |
| `managed_profile_identity_mismatch`, `unknown_pi_profile` | Explicit identity differs or selected profile is absent. |
| `profile_state_key_collision`, `profile_state_path_invalid` | Exact-name hash collision, or cache/path/read/no-follow/containment/revalidation failure. |
| `provider_executable_not_found`, `pi_compatibility_unsupported` | Pi is absent or no compiled entry matches ID/host. |
| `pi_execution_identity_unavailable`, `pi_execution_identity_malformed`, `pi_execution_identity_mismatch` | Pi closure cannot be fully read, has invalid shape, or differs from the catalog. |
| `pi_execution_environment_invalid`, `pi_execution_identity_changed` | Environment is denied/ambiguous, or point-of-use identity changed/read failed. |
| `runtime_executable_not_found`, `runtime_executable_invalid` | Runtime is absent or not the exact absolute readable executable. |
| `pi_profile_busy` | Exact project/profile lock is already held. |
| `runtime_listener_occupied`, `runtime_listener_check_failed` | Loopback port is occupied or vacancy cannot be established. |
| `runtime_start_failed`, `runtime_exited_early` | Direct spawn failed or selected child died. |
| `runtime_readiness_timeout`, `runtime_readiness_invalid`, `runtime_model_unavailable` | Exact readiness timed out after connection/503 retries, returned another non-200 or malformed response, or lacked the exact model. |
| `pi_start_failed`, `runtime_shutdown_timeout` | Pi spawn failed, or the runtime group required forced kill; launcher returns nonzero. |

Start troubleshooting with `pi-infra --print-config`, then `agents-infra verify
local "$PWD"`. Check exact provenance, profile/model, literal runtime argv,
loopback URL, static executable state, `project_state_key`,
`profile_state_key`, contained paths, requested/unverified capability labels,
and Pi catalog identity before permitting a process launch.

Create the local runtime, then either edit the TOML manually or use the setup
flags below:

```bash
PROJECT=/abs/path/to/project
agents-infra setup local "$PROJECT"
```

```toml
[mcp]
enabled_servers = ["figma"]

[agents.codex.primary_session]
model = "gpt-5.6-terra"
reasoning_effort = "xhigh"
yolo_mode = false

[agents.claude.primary_session]
model = "claude-opus-4-6"
yolo_mode = false
```

`[agents.codex.primary_session]` must contain at least one of these optional
fields. `model` and `reasoning_effort` are non-empty TOML strings after
trimming; `yolo_mode` is a real TOML boolean, not a quoted string. Codex—not
agents-infra—remains responsible for whether a model is available and whether a
model/effort pair is compatible.

`[agents.claude.primary_session]` accepts optional non-empty `model` and
`yolo_mode` fields. `yolo_mode` is an unquoted TOML boolean. Claude reasoning
remains provider-native. Codex fields never configure `agents-infra claude`,
and Claude fields never configure `agents-infra codex`; `[mcp]` remains the one
intentional provider-shared project section.

### Canonical vendor target entrypoints

Canonical targets are an additive, strict launch path for installed vendor
aliases. They separate vendor identity from the provider harness and map each
public entrypoint explicitly:

```toml
[agents.targets."openai-sol-high"]
vendor = "openai"
environment = "codex"
model = "gpt-5.6-sol"
reasoning = "high"

[agents.targets."anthropic-opus-high"]
vendor = "anthropic"
environment = "claude-code"
model = "claude-opus-5"
reasoning = "high"

[agents.targets."qwen-mlx-8bit"]
vendor = "qwen"
environment = "pi"
model = "Qwen3.8-27B-MLX-8bit"
reasoning = "medium"
profile = "qwen-3.8-27b-mlx-8bit"
profile_provider = "local-qwen" # optional assertion
endpoint = "http://127.0.0.1:18011/v1" # optional assertion

[agents.entrypoints]
openai-infra = "openai-sol-high"
anthropic-infra = "anthropic-opus-high"
qwen-infra = "qwen-mlx-8bit"
```

Only `openai/codex`, `anthropic/claude-code`, and `qwen/pi` are admitted.
Claude reasoning is limited to `low`, `medium`, `high`, `xhigh`, and `max`;
Pi uses its documented thinking levels; Codex accepts any non-empty reasoning
token and leaves model/effort compatibility to Codex. A Qwen target must name
an existing complete managed Pi profile with `api = "openai-completions"`.
Its model and thinking must match the profile. A non-`off` Qwen target also
requires profile `reasoning = true` and
`compat.thinking_format = "qwen-chat-template"`; pinned Pi then carries the
selected native level as `--thinking medium` and enables the Qwen chat-template
thinking path. Optional provider and endpoint
fields are exact assertions; the effective qualified model, provider, and
endpoint always come from the selected profile definition.

Target definitions compose atomically from filesystem root to current working
directory: a nearer complete target replaces the same exact name, and a nearer
entrypoint mapping replaces that alias mapping. Missing mappings, unknown
targets, malformed fields, alias/vendor disagreement, invalid Pi profiles, or
conflicting identity selectors fail before provider/runtime side effects. No
alias infers a target from legacy tables and no setup, verify, compose, print,
or launch operation rewrites project configuration.

Alias invocations identity-lock model and reasoning. Exact repeats are
accepted; divergent repeats fail with `target_identity_conflict`. Codex also
locks `-c model=...` and `-c model_reasoning_effort=...` and refuses profile
selectors. Pi decodes `model`, `provider/model`, `model:thinking`, and
`provider/model:thinking`, refusing a divergent provider or thinking suffix
before the legacy Pi composer can normalize it.

Inspect or compose the selected target without launching it:

```bash
openai-infra --print-config
anthropic-infra --print-config
qwen-infra --print-config
qwen-infra spawn --prompt "Complete the bounded task" --print-config

agents-infra compose --mode primary-session \
  --entrypoint qwen-infra --project "$PWD" --schema-version 1 --json \
  -- --model 'local-qwen/Qwen3.8-27B-MLX-8bit:off'
```

Alias JSON plans retain schema version 1 and add `target`, plus Pi-only
`resolved.profile_provider` and `resolved.endpoint` provenance. Legacy
`compose --agent ...` plans omit those fields byte-for-byte. Direct
`agents-infra codex`, `agents-infra claude`, `agents-infra pi`, and `pi-infra`
keep their existing CLI/project/provider precedence; merely declaring targets
does not change them. `doctor local` reports configured mappings and their
resolved coordinates with sources.

When launched from a project directory, `agents-infra codex` walks from the
filesystem root to the current directory and reads each
`.agents/.configs/project-config.toml` it finds. It parses every discovered
file once, retains absolute paths as provenance, and fails before launch when
any discovered file is invalid. `~/.agents/.configs/project-config.toml` is
never a project-policy source. Each primary-session field composes
independently: the nearest file that explicitly supplies that field wins. In
particular, a child `yolo_mode = false` masks a parent's `true`; omission means
inheritance.

Claude model and yolo provenance compose independently with the same
root-to-leaf rule: the nearest explicitly configured field wins, and
`yolo_mode = false` masks an inherited `true`. Claude never inherits Codex
model, effort, or yolo values and does not affect Codex resolution.

For model and reasoning effort, the per-field launch precedence is:

| Priority | Source | Notes |
| --- | --- | --- |
| 1 | Explicit Codex selection | `--model`/`-m`, top-level `-c model=...`, or top-level `-c model_reasoning_effort=...` passed through this launcher. |
| 2 | Effective project primary-session field | Emitted as a Codex CLI/config override only when selected. |
| 3 | Codex native resolution | Project/profile/user/system/default configuration. |

An explicit `--profile` or `-p` suppresses project model and project reasoning
so the profile can resolve them; an explicit model or effort passed alongside a
profile still wins for its own field. Equal duplicate explicit values collapse
to one override; conflicting explicit values fail before Codex is executed.
Model, reasoning, and profile wrapper arguments participate only before `--`.

Yolo is an independent safety decision per provider. For either launcher,
explicit `-d`, `--danger`, `--yolo`, or its native dangerous flag wins over
project policy; otherwise effective project `yolo_mode = true` enables it, and
an explicit project `false` or an absent value emits no dangerous flag. Codex
uses `--dangerously-bypass-approvals-and-sandbox`; Claude uses
`--dangerously-skip-permissions`. Each launch emits that provider's native
dangerous flag at most once, and `--print-config` records the effective source
and whether project policy was suppressed by explicit CLI input. Persistent
yolo applies only to its matching `agents-infra codex` or `agents-infra claude`
primary launch; it never affects `task-board spawn`, task-board manifests, or
child-run selection.

For Claude primary-session policy, `agents-infra claude` applies an effective
project model through native `--model MODEL` and an effective yolo value through
`--dangerously-skip-permissions`. An explicit Claude `--model` or
`--model=MODEL` before `--` wins and suppresses only the project model for that
launch; explicit Claude danger input suppresses only the project yolo policy.
The Claude wrapper does not infer model or yolo values from Codex policy,
profiles, or task-board settings.

Use the supported local setup surface to update a project without replacing
MCP settings, unrelated TOML tables, comments, or unspecified primary fields:

```bash
agents-infra setup local /abs/path/to/project \
  --codex-primary-model gpt-5.6-terra \
  --codex-primary-reasoning-effort xhigh \
  --codex-yolo-mode=false \
  --claude-primary-model claude-opus-4-6 \
  --claude-yolo-mode=false

agents-infra setup local /abs/path/to/project \
  --clear-codex-primary-session

agents-infra setup local /abs/path/to/project \
  --claude-primary-model claude-opus-4-6

agents-infra setup local /abs/path/to/project \
  --clear-claude-primary-session
```

No provider primary-session flags leave `project-config.toml` byte-identical.
Set flags update only the supplied provider fields, so
`--codex-yolo-mode=false` is an explicit value and
`--claude-primary-model` never edits the Codex, MCP, comments, or unrelated
TOML content. Each clear removes only its own table. Clear cannot be combined
with a set flag for the same provider, all primary-session flags are local-only,
and a target that resolves to `~/.agents/.configs/project-config.toml` is
rejected. Validation or atomic-write failures preserve the original file.
Supported Unix targets use an atomic rename; Windows uses
[`atomic.ReplaceFile`](https://pkg.go.dev/github.com/natefinch/atomic#ReplaceFile)
(`MoveFileExW` replace/write-through); unsupported replacement targets fail
closed.

Inspect either provider invocation without launching an agent:

```bash
cd /abs/path/to/project
agents-infra codex --print-config
agents-infra codex --print-config --model gpt-5.6-terra -c 'model_reasoning_effort="xhigh"'

agents-infra claude --print-config
agents-infra claude --print-config --model claude-opus-4-6

# Start an actual primary Codex session from the same project.
agents-infra codex
```

`--print-config` prints every discovered project-config path; effective and
project values with their sources; explicit-CLI/profile suppression state where
applicable; wrapper yolo expansion; and the exact provider args to be executed.
It is the first diagnostic when a launch does not use the expected values.

`doctor local` uses the same resolver for persistent configuration evidence:

```bash
agents-infra doctor local /abs/path/to/project
```

For a configured project, its stable primary-session fields include:

```text
codex_primary_config_valid: true
codex_primary_model: gpt-5.6-terra
codex_primary_model_source: /abs/path/to/project/.agents/.configs/project-config.toml
codex_primary_reasoning_effort: xhigh
codex_primary_reasoning_effort_source: /abs/path/to/project/.agents/.configs/project-config.toml
codex_primary_yolo_mode: false
codex_primary_yolo_mode_source: /abs/path/to/project/.agents/.configs/project-config.toml
claude_primary_config_valid: true
claude_primary_model: claude-opus-4-6
claude_primary_model_source: /abs/path/to/project/.agents/.configs/project-config.toml
claude_primary_yolo_mode: false
claude_primary_yolo_mode_source: /abs/path/to/project/.agents/.configs/project-config.toml
```

When a provider model or Codex effort is absent, doctor renders an empty value
with source `native`; absent Codex or Claude yolo renders `false` with source
`default`. Invalid ancestor TOML makes doctor nonzero, reports the exact path
and field, and sets both provider validation flags false without printing
partial provider policy.

Existing `.codex/config.toml` behavior is unchanged: no automatic migration
or deletion occurs in the default `preserve` mode, and custom local config
remains an intentional Codex-native project layer.
`codex_config_shadowing_global: true` means it overrides the global
`~/.codex/config.toml`; use `--codex-config=global` to remove an unwanted local
config, or `--codex-config=local` to render the managed project-safe local one.
The managed local file intentionally has no `profiles` table because Codex does
not support profiles at project scope. Project primary-session overrides and
`.codex/config.toml` can coexist; profiles remain available through the full
global config.

Troubleshooting:

- Unexpected model or effort: run `agents-infra codex --print-config`; check
  `effective_source`, `project_application`, an explicit `-c`/`--model`, and
  profile suppression.
- Unexpected danger flag: inspect that launcher's `wrapper_expansions` and
  `yolo_mode`; set the nearest provider field explicitly to `false` to mask an
  inherited `true`.
- Unexpected Claude model: run `agents-infra claude --print-config`; check its
  `effective_source`, `project_application`, and any explicit `--model`.
- Invalid configuration: use unquoted strings for model/effort and an unquoted
  boolean for yolo; the launcher reports the source path and field.
- Global model appears ignored: run `agents-infra doctor local PROJECT` and
  resolve any `codex_config_shadowing_global: true` state deliberately.

Task-board child spawn ceilings belong to the separate
[task-board spawn-ceiling contract](https://github.com/relux-works/skill-project-management/blob/main/.specs/project-agent-selection-policy.md#task-board-spawn-ceiling-contract).
Agents-infra neither reads nor validates that `task-board.config.json` policy;
it owns only this primary-session TOML.

### Parent primary-goal alignment

The installed workflow instructions also tell an eligible primary parent to
keep the active task-board's provider-neutral `primary-goal-v1` record aligned
with materially actionable user requirements. This applies only when
`TASK_BOARD_RUN_ID` is absent and an active task-board is available.

The parent reads `task-board goal get`, then uses `goal set-primary` when no
record is active or `goal update --if-revision N` when the complete objective
materially changes. Later goal-changing turns replace the complete objective
while retaining unresolved prior requirements. The policy permits at most one
successful write per user turn, skips chatter and semantic no-ops, and retries
one revision conflict after re-reading. Routine successful synchronization is
silent.

Spawned owners never mutate this record; they continue to use
`task-board spawn goal`. A primary update does not alter a spawned delivery
goal, so delivery-scope changes still require the explicit spawn-goal upsert or
reroute workflow. Version 1 does not invoke native Codex or Claude goal APIs and
does not clear the primary automatically on session exit.

This is instruction-only integration. Agents-infra stores no board goal state
and has no task-board library dependency; the parent invokes the external CLI.
After authoring-source changes, install and verify the policy with:

```bash
agents-infra setup global --source-dir /path/to/relux-agents-infra
rg -n "Primary Parent Goal Actualization" \
  ~/.agents/.instructions/INSTRUCTIONS_WORKFLOW.md \
  ~/.codex/AGENTS.md
```

## Tooling

| Tool | Purpose | Command | Outputs |
|------|---------|---------|---------|
| `./setup.sh` / `./setup.ps1` | Bootstrap the `agents-infra` and `model-harness` CLIs and sync the global runtime | `./setup.sh`, `.\setup.ps1` | `~/.local/bin/agents-infra`, `~/.local/bin/model-harness`, `~/.agents/`, `~/.claude/`, `~/.codex/`, install-state metadata |
| `agents-infra` | Set up or inspect global/project-local agent runtimes; prepare provider project surfaces without launching; compose non-launching MCP-only or primary-session launch plans; launch isolated primary Codex, Claude, managed Pi, and standalone unattended Pi workers; inspect, stop, quarantine, or unquarantine shared local runtimes; run bounded managed local-model behavior checks; run the Go attachment helper | `agents-infra setup global`, `agents-infra setup local /path/to/project --codex-primary-model MODEL --codex-primary-reasoning-effort EFFORT --codex-yolo-mode=true\|false --claude-primary-model MODEL`, `agents-infra setup local /path/to/project --clear-codex-primary-session`, `agents-infra setup local /path/to/project --clear-claude-primary-session`, `agents-infra doctor local /path/to/project`, `agents-infra prepare --agent codex --project /path/to/project --schema-version 1 --json`, `agents-infra compose --agent codex --project /path/to/project --schema-version 1 --json`, `agents-infra compose --mode primary-session --agent pi --project /path/to/project --schema-version 1 --json`, `agents-infra target qwen-infra spawn --prompt "Complete the bounded task" --deadline 10m`, `agents-infra model-check --target qwen-infra --prompt "Reply with READY" --output-dir .temp/model-check`, `agents-infra attachments list`, `agents-infra codex --print-config`, `agents-infra claude --print-config`, `agents-infra pi --print-config`, `agents-infra runtime status --profile NAME --json`, `agents-infra runtime stop --profile NAME --force --timeout 30`, `agents-infra runtime quarantine --profile NAME`, `agents-infra runtime unquarantine --profile NAME` | Runtime directories and rendered provider artifacts under the target root; deterministic preparation/compose or standalone launch-plan JSON, standalone Pi JSONL output and exit status, hash-contained Pi client and shared-runtime state under the user cache directory, mode-0600 model-check `events.jsonl`, `stderr.log`, `summary.json`, and `summary.txt` under the explicit output directory, attachment manifests/staged images, or printed diagnostics on stdout |
| `model-harness` | Resolve and run machine-local or SSH-forwarded model server profiles, plus bounded local synthetic-prefill capacity checks, while keeping agent configuration separate from backend-specific lifecycle details | `model-harness render PROFILE --host 127.0.0.1 --port PORT --json`, `model-harness doctor PROFILE --host 127.0.0.1 --port PORT`, `model-harness run PROFILE --host 127.0.0.1 --port PORT`, `model-harness stress PROFILE --host 127.0.0.1 --port PORT --json` | Exact side-effect-free launch-plan JSON, readiness diagnostics, a foreground backend/SSH process owned by `agents-infra`, or a versioned stress report with observed prompt tokens, timing, and process RSS evidence |
| `pipx` | Install an isolated, reproducibly pinned model-server runtime when a required upstream fix has not reached PyPI | `pipx install --suffix=-qwenfix --python python3.14 'git+https://github.com/ml-explore/mlx-lm.git@COMMIT'` | Isolated virtual environment under the pipx home and suffixed entry points under the pipx bin directory |
| `mlx-swift-runtime-prototype` | Task-scoped MLX Swift LM prototype that serves the configured local Qwen model over the same OpenAI-compatible surface the Pi profile uses, so an MLX Swift migration can be measured without changing the default Python `mlx-lm` runtime | `cd tools/mlx-swift-runtime-prototype`, then `xcodebuild -downloadComponent MetalToolchain` once per host, `xcodebuild build -scheme mlx-swift-runtime-prototype -configuration Release -destination 'platform=macOS,arch=arm64' -derivedDataPath ./DerivedData -skipPackagePluginValidation -skipMacroValidation` (SwiftPM cannot compile mlx-swift's Metal shaders, so `swift build` yields a binary that refuses to start), `swift test -c release` for the contract suite, `DerivedData/Build/Products/Release/mlx-swift-runtime-prototype serve --model /abs/model --host 127.0.0.1 --port PORT`, `HARNESS=... HARNESS_CONFIG=... scripts/smoke.sh`, `BINARY=... scripts/lifecycle-smoke.sh` and `BINARY=... scripts/metallib-gate-probe.sh` for the weight-free lifecycle and startup-gate probes, and `BINARY=... HARNESS=... MODEL=... scripts/dead-generation-smoke.sh` for the dead-generation-worker health regression (`/health` must answer 503 once the generation worker is condemned, `model-harness` must restart it on the `generation_worker_unavailable` marker, and a request-scoped failure must change neither), and `BINARY=... HARNESS=... MODEL=... scripts/generation-batch-recovery-smoke.sh` for the generation-batch failure recovery regression (a mid-batch failure must end its request with an explicit error rather than a truncated success, release the batch and any implicated cache state, and let the next request complete on the same unrestarted process, while an unrecoverable failure still reaches 503), plus `DerivedData/Build/Products/Release/mlx-swift-runtime-prototype benchmark-run --config ... --model ... --prompts examples/benchmark-prompts.json --thresholds examples/benchmark-thresholds.json --session ... --harness ... --baseline-runtime python-mlx-lm --baseline-profile ... --candidate-runtime mlx-swift --candidate-profile ... --python-bin ... --candidate-binary ...` for the Python-vs-Swift migration decision, with `BINARY=... scripts/benchmark-gate-smoke.sh` driving the decision and replay entry points through the real subcommands (ONE invocation spawns both runtimes through `model-harness`, drives every scenario against them, samples the Mach physical footprint of the pid it spawned, seals the record it built with a transcript digest and judges the pair; the two runtimes are measured sequentially because a 64 GiB host cannot hold two copies of a 28 GB model; there is no `benchmark-attest` subcommand and no flag through which a caller can supply a measurement, because review obtained `accepted=true` three times by handing the previous gates documents about work nobody did — most recently two placeholder HTTP servers that answered only `GET /v1/models`; and admission refuses any pair whose pinned host, model, quantization, prompt suite, context policy — KV bound, prefill chunk *and* chat-template reasoning effort — output bound or sampler differs, whose wall-clock intervals overlap, which no attestation covers, which was observed by a different build than the one judging, whose measurements do not digest to what the observation sealed, whose scenarios carry no served completion, or that leaves a scored metric unmeasured; `benchmark-compare` replays an archived session and can never return an acceptance) | A release binary plus its `mlx-swift_Cmlx.bundle` shader bundle under `tools/mlx-swift-runtime-prototype/DerivedData/Build/Products/Release/` (both gitignored), one-line JSON lifecycle events on stdout carrying load time, physical footprint and MLX active bytes, smoke transcripts under the caller's `OUT` directory, and a benchmark session directory under the caller's `--session` path holding `records/`, `attest/`, `logs/`, `session.json` and `decision.json` |
| `pi-infra` | Stable global/project-local alias for the managed Pi production entry point; preserves caller cwd and every argument and refuses a missing sibling target | `pi-infra --print-config`, `pi-infra --profile qwen-3.8-27b -- "ordinary prompt"`, `pi-infra` | Non-launching `agents-infra.primary-session-launch-plan` JSON or an isolated Pi/runtime session under the canonical user cache root |
| `openai-infra`, `anthropic-infra`, `qwen-infra` | Strict sibling-only aliases for configured canonical vendor targets; preserve cwd/argv and lock target identity; `qwen-infra` additionally exposes the explicit standalone unattended worker primitive | `openai-infra --print-config`, `anthropic-infra --print-config`, `qwen-infra --print-config`, `qwen-infra spawn --prompt "Complete the bounded task" --deadline 10m`; machine consumers use `agents-infra compose --mode primary-session --entrypoint NAME --project DIR --schema-version 1 --json` | Alias launch, standalone Pi JSONL result stream plus deterministic process status, or non-launching schema-v1 plan with target and effective-coordinate provenance; no project-config mutation or task-board dependency |
| `agents-attachments` | Backwards-compatible launcher for the Go attachment helper | `agents-attachments list`, `agents-attachments path screenshot.png`, `agents-attachments stage-images ./photo.heic --out-dir .temp/image-intake` | `.temp/agents-attachments-manifest.json`, `.temp/agents-attachments/`, staged images and `image-stage-map.json` under caller-selected `.temp/` |
| `sips` / ImageMagick `magick` | Normalize HEIC/HEIF image inputs for staged inspection | `sips -s format png input.heic --out output.png`, `magick input.heic output.png` | Normalized staged images under caller-selected `.temp/` |
| `go` | Build, test, and vet the Go CLI in `tools/agents-infra` | `cd tools/agents-infra && go test ./...`, `cd tools/agents-infra && go vet ./...` | Go test cache; task-scoped logs should be written under `.temp/` |
| `task-board` | Track project work, checklist state, and outcome resources | `task-board q --format compact 'get(TASK-ID) { full }'`, `task-board m 'set_status(TASK-ID, status=development)'` | `.task-board/` and `.task-board/.resources/` |
| `git` | Inspect repo state and validate diff hygiene | `git status --short`, `git diff --check` | No repo artifact; task-scoped command logs should be written under `.temp/` |
| `ssh` / `scp` / `tar` | Validate and document host-agnostic remote agent worker handoff patterns | `ssh "$REMOTE_SSH" 'hostname'`, `scp prompt.md "$REMOTE_SSH:/tmp/run/prompt.md"`, `tar -czf source.tgz .` | Remote task copies and local scratch artifacts under `.temp/remote-agent/` |

## Structure

```
~/.agents/
├── .instructions/          # Global instructions (modular .md files)
│   ├── INSTRUCTIONS.md     # Entry point (loads all modules)
│   ├── AGENTS.md           # Entry point for Codex CLI
│   ├── INSTRUCTIONS_ATTACHMENTS.md
│   ├── INSTRUCTIONS_BROWSER_AUTOMATION.md
│   ├── INSTRUCTIONS_REMOTE_AGENTS.md
│   ├── INSTRUCTIONS_PLATFORM.md
│   ├── INSTRUCTIONS_STRUCTURE.md
│   ├── INSTRUCTIONS_TOOLS.md
│   ├── INSTRUCTIONS_SKILLS.md
│   ├── INSTRUCTIONS_DIAGRAMS.md
│   ├── INSTRUCTIONS_TESTING.md
│   ├── INSTRUCTIONS_WORKFLOW.md
│   ├── INSTRUCTIONS_DOCS.md
│   └── INSTRUCTIONS_STYLE.md
│
├── .skills/                # Source-managed shared skills versioned in this repo
│   ├── algorithmic-art/
│   ├── architecture-diagrams/
│   ├── brand-guidelines/
│   ├── canvas-design/
│   ├── doc-coauthoring/
│   ├── docx/
│   ├── frontend-design/
│   ├── internal-comms/
│   ├── ios-ui-validation/
│   ├── mcp-builder/
│   ├── pdf/
│   ├── pptx/
│   ├── skill-creator/
│   ├── slack-gif-creator/
│   ├── theme-factory/
│   ├── web-artifacts-builder/
│   ├── web-search/
│   ├── webapp-testing/
│   └── xlsx/
│
├── skills/                 # External skills/tooling area in installed runtime; not versioned by this repo
│
├── scripts/                # Cross-platform bootstrap entrypoints
│   ├── setup.sh
│   └── setup.ps1
│
├── .scripts/               # Setup and utility scripts
│   └── setup-symlinks.sh   # Internal compatibility wrapper over agents-infra
│
├── .configs/               # Tool configurations
│   ├── claude-settings.json    # Claude Code settings (reference)
│   ├── codex-config.toml       # Codex CLI config
│   └── codex-mcp-servers.toml  # Known Codex MCP server definitions
│
├── tools/
│   └── agents-infra/       # Go CLI source
│
└── .rules/                 # Codex CLI rules
    └── default.rules       # Pre-approved commands
```

## Instructions

Modular instruction files in `.instructions/`:

| File | Purpose |
|------|---------|
| `INSTRUCTIONS.md` | Entry point for Claude Code |
| `AGENTS.md` | Entry point for Codex CLI |
| `INSTRUCTIONS_PLATFORM.md` | Target platform preferences (iOS > macOS) |
| `INSTRUCTIONS_STRUCTURE.md` | Project structure conventions |
| `INSTRUCTIONS_ATTACHMENTS.md` | Generic attachment manifest, image staging, inspection, OCR fallback, and redaction workflow |
| `INSTRUCTIONS_BROWSER_AUTOMATION.md` | No-focus browser scripting and authenticated browser-session rules |
| `INSTRUCTIONS_REMOTE_AGENTS.md` | Host-agnostic workflow for using remote Claude/agent workers through isolated project copies and patch handoff |
| `INSTRUCTIONS_TOOLS.md` | Allowed CLI tools |
| `INSTRUCTIONS_SKILLS.md` | Skills system usage |
| `INSTRUCTIONS_DIAGRAMS.md` | C4/PlantUML diagram rules |
| `INSTRUCTIONS_TESTING.md` | Swift Testing, refactoring workflow |
| `INSTRUCTIONS_WORKFLOW.md` | Task tracking, parent primary-goal actualization, model fallback, autonomous completion, forced-fit escalation, Git, and logging |
| `INSTRUCTIONS_DOCS.md` | Documentation requirements |
| `INSTRUCTIONS_STYLE.md` | Communication style |

## Skills

Each skill follows the structure:

```
skill-name/
├── SKILL.md              # Required: frontmatter + instructions
├── scripts/              # Optional: executable code
├── references/           # Optional: docs/schemas
└── assets/               # Optional: templates/resources
```

### Available Skills

| Skill | Description |
|-------|-------------|
| `ios-ui-validation` | UI testing with screenshot validation, Page Object pattern |
| `skill-creator` | Scaffold new skills |
| `architecture-diagrams` | C4/PlantUML diagrams |
| `frontend-design` | Production-grade frontend interfaces |
| `pdf` | Markdown/HTML to PDF rendering with shared themes |
| `webapp-testing` | Playwright-based web testing |
| `mcp-builder` | Build MCP servers |
| `web-search` | Web search integration |
| `canvas-design` | Visual art in PNG/PDF |
| `algorithmic-art` | p5.js generative art |
| `theme-factory` | Artifact styling toolkit |
| `brand-guidelines` | Anthropic brand colors/typography |
| `internal-comms` | Internal communications templates |
| `slack-gif-creator` | Animated GIFs for Slack |
| `doc-coauthoring` | Documentation co-authoring workflow |
| `web-artifacts-builder` | Multi-component HTML artifacts |

## Optional PDF Toolchain

Install the PDF renderer stack with:

```bash
./setup.sh --with-pdf-tools
```

Or without rerunning the whole bootstrap:

```bash
./.scripts/setup-pdf-tools.sh
./.scripts/setup-pdf-tools.sh --check
```

Managed dependencies:

- `pandoc`
- `weasyprint`
- `poppler` (`pdftotext`, `pdfinfo`)

The shared PDF skill lives at `.skills/pdf/` and includes:

- `scripts/render-pdf.sh`
- `assets/template.html5`
- `assets/themes/prose-classic.css`
- `assets/themes/report-clean.css`

Example:

```bash
./.skills/pdf/scripts/render-pdf.sh notes/report.md \
  -o .temp/report.pdf \
  --theme prose-classic \
  --title "Research Report"
```

Quick preflight and discovery:

```bash
./.scripts/setup-pdf-tools.sh --check
./.skills/pdf/scripts/render-pdf.sh --list-themes
```

## Configs

### Claude Code (`claude-settings.json`)

Reference config with:
- Allowed tools (Bash, Read, Edit, Write, etc.)
- Default model: `sonnet` (currently Sonnet 4.6)
- Enabled plugins: `swift-lsp`

### Codex CLI (`codex-config.toml`)

- Model: `gpt-5.6-sol`
- Context window override: `1000000`
- Auto-compaction token limit: `900000`
- Reasoning effort: `xhigh`
- Service tier: `default` (Standard).
- Project docs byte limit: `131072`
- The approaching-rate-limit model switch reminder is suppressed with `[notice].hide_rate_limit_model_nudge = true` so Codex does not ask to move to a lower-credit model.
- As of the audited Codex CLI `0.144.1`, the separate safety-buffering chooser (`Retry with a faster model` / `Keep waiting`) has no supported `config.toml` setting for suppression, a default choice, or automatic waiting. It is runtime UI shown before agent instructions can act; terminal key automation or a patched Codex binary is intentionally out of scope.
- Global workflow instructions treat temporary model unavailability as an operational condition: retry the preferred model at least three times, then choose the best viable fallback autonomously and escalate only a real blocker.
- Trusted projects list
- Global setup owns `~/.codex/config.toml`; project-local setup deliberately does not create `.codex/config.toml` so the global model/settings remain authoritative.
- Local setup removes legacy managed project-local config symlinks but preserves custom `.codex/config.toml` files.
- Explicit project-local config is available with `agents-infra setup local /path/to/project --codex-config=local`; it is rendered atomically from the installed config without the unsupported top-level `profiles` table.
- Enforce global config with `agents-infra setup local /path/to/project --codex-config=global`.
- `agents-infra doctor local` reports `codex_config_generated: true` for the managed rendered file and `codex_config_shadowing_global: true` whenever a project-local `.codex/config.toml` overrides the global config.

### Project-Local MCP Opt-In (Codex + Claude Code)

Agents-infra does not enable MCP servers in the global Codex or Claude Code
config by default. Projects opt in explicitly through a single,
agent-agnostic list in `.agents/.configs/project-config.toml`:

```toml
[mcp]
enabled_servers = ["figma"]
```

There is one list per project, not one per agent — `enabled_servers` decides
which servers are available regardless of whether you launch Codex or Claude
Code. Known MCP server definitions live in `.configs/codex-mcp-servers.toml`
and are synced into project runtimes. Definitions can describe streamable
HTTP servers with `url` or stdio servers with `command` and optional `args`.

Start Codex through `agents-infra codex` from inside the project tree. The
launcher walks upward from the current directory, composes every discovered
`.agents/.configs/project-config.toml`, resolves enabled MCP definitions from
project registries plus the global registry, logs where each part came from,
then starts Codex with the resulting `-c` overrides:

```bash
agents-infra codex
agents-infra codex -d -
agents-infra codex exec "check the Figma node"
agents-infra codex --print-config
```

Start Claude Code the same way through `agents-infra claude` — same
`enabled_servers` list, same registries, same ancestor walk — but rendered as
a single Claude Code `--mcp-config` JSON payload instead of Codex `-c`
overrides (streamable HTTP servers become `{"type":"http","url":...}`, with
`bearer_token_env_var` mapped to an `Authorization: Bearer ${VAR}` header for
Claude Code to expand at launch; stdio servers become
`{"type":"stdio","command":...,"args":[...]}`). That payload is added on top
of whatever MCP servers are already configured at the user/project level —
the launcher does not pass `--strict-mcp-config`, so existing `.mcp.json` /
`claude mcp add` servers keep working unchanged:

```bash
agents-infra claude
agents-infra claude -d
agents-infra claude --print-config
```

For a child runner that already owns model, safety, prompt, and lifecycle
arguments, use the non-launching composition contract instead of either primary
launcher:

```bash
agents-infra compose --agent codex --project "$PWD" --schema-version 1 --json
agents-infra compose --agent claude --project "$PWD" --schema-version 1 --json
```

Its `argv_prefix` is only the provider rendering of the resolved
`enabled_servers` set; safe metadata repeats no URL, command, args, or headers.
Bearer token values are never read or emitted.

For a session manager that wants to own the primary provider process itself
(for example the task-board Session Manager), use the primary-session
composition mode. It resolves exactly the launch plan `agents-infra codex`,
`agents-infra claude`, or `agents-infra pi` would execute — same project-config precedence, same
executable lookup, same argument ordering, including provider user args passed
after `--` — but performs no launch and emits one machine-readable
`agents-infra.primary-session-launch-plan` schema v1 document:

```bash
agents-infra compose --mode primary-session --agent codex --project "$PWD" --schema-version 1 --json
agents-infra compose --mode primary-session --agent claude --project "$PWD" --schema-version 1 --json -- --continue
agents-infra compose --mode primary-session --agent pi --project "$PWD" --schema-version 1 --json -- --version
```

The response contains:

- `executable` — the resolved provider binary path (fails closed with error
  code `provider_executable_not_found` when the provider is not on `PATH`);
- `launch_variants.interactive.argv` — the exact argv the launching wrapper
  would pass for a terminal session;
- `launch_variants.managed_host` — the argv for a manager-owned host process.
  For Codex the kind is `codex-app-server` and the argv is derived from the
  same normalized interactive argv with an explicit, total three-way
  classification (host argv, managed client argv, resolved session policy):
  every config-level global option class keeps its relative order (arbitrary
  `-c`/`--config` overrides, `--enable`, `--disable`, `--strict-config`,
  `--profile`/`-p`/`-pVALUE`, `--oss`, `--local-provider`, `--search`,
  `--dangerously-bypass-hook-trust`), the effective `--model`/`-m`/`-mVALUE`
  converts to its `-c model=` override, and a terminal `app-server` token
  ends the argv; the consumer appends `--listen <url>`. Approval/sandbox
  policy is intentionally absent from this argv (the bypass flag,
  `--sandbox`/`--ask-for-approval` selections, and
  `-c sandbox_mode=`/`approval_policy=` overrides) — the manager applies it
  per thread over the app-server RPC using the `resolved` block. For Claude
  the kind is `claude-pty` and the argv equals the interactive composition,
  run under a manager-owned PTY;
- `launch_variants.managed_client` — every remaining interactive token, in
  interactive order: thread/client options (`-C`/`--cd`, `--add-dir`,
  `-i`/`--image`, `--no-alt-screen`, `--remote`, `--remote-auth-token-env`),
  subcommands with their flags, prompt text, and everything after `--`.
  Unrecognized future provider flags also land here instead of being silently
  dropped; the session manager applies these on its client or thread layer
  (thread cwd, writable roots, initial prompt, session selection) and must
  fail closed on any token it cannot represent. The split is total: every
  interactive token appears in `managed_host.argv`, in `managed_client.argv`,
  or as session policy in `resolved`. For Claude this fragment is always
  empty because the whole interactive argv runs in the managed PTY;
- `resolved` — model, reasoning, yolo, sandbox, profile, approval, and MCP
  with per-field provenance (`value` is null when the provider's native
  configuration decides or the field does not apply to the provider);
- `required_env_names` and `sources` — environment variable names (never
  values) and the project-config/registry files that contributed. The names
  cover composed MCP `bearer_token_env_var` references and, for Codex, the
  environment variable named by a valid `--remote-auth-token-env VALUE` or
  `--remote-auth-token-env=VALUE` selection, in deterministic order (MCP names
  first, then the remote auth name) and de-duplicated. Tokens after the
  provider `--` are never interpreted as environment options, and Codex's
  accepted empty name (`--remote-auth-token-env=`) contributes no requirement.
  A repeated occurrence or a missing value fails closed exactly like the Codex
  parser rejects it.

Explicit Codex model and profile selections resolve in every form the
provider parser accepts — `--model`/`-m` and `--profile`/`-p` spaced, `=`,
and attached (`-mVALUE`, `-pVALUE`) — with `cli:<flag>` provenance, and an
explicit CLI model suppresses the composed project-config model. Profile
values are validated against the Codex plain profile-name syntax (ASCII
letters, digits, dashes, underscores; non-empty) in every spelling, so an ok
plan never carries a profile value the provider rejects at launch; whether the
named profile exists remains provider-native config resolution.

Pass-through native policy selections are reflected into `resolved` with
provider-faithful precedence. Codex `--sandbox`/`-s` and
`--ask-for-approval`/`-a` (spaced, `=`, and attached short forms) resolve with
`cli:<flag>` provenance; `-c sandbox_mode=` and `-c approval_policy=` resolve
with `cli:-c <key>` provenance, a typed flag wins over a `-c` override
regardless of order, and repeated `-c` overrides keep last-wins semantics.
Repeated Codex policy flags, a flag without a value, and combining the bypass
flag (or `-d`) with an explicit policy flag all fail closed exactly like the
Codex parser rejects them. Policy values are validated against the
provider-accepted domains before they are serialized as effective, so an ok
plan never carries a value the provider rejects at launch: typed
`--sandbox`/`--ask-for-approval` values must be in the clap flag enums, while
`-c sandbox_mode=`/`approval_policy=` values must be in the config
deserialization domains (`on-failure` and `granular` are config-only approval
variants the typed flag rejects). Matching the provider, only the last `-c`
override per policy key is validated — earlier repeats are masked by
last-wins — and a typed flag does not mask an invalid `-c` override. Claude
`--effort` and `--permission-mode` resolve into `resolved.reasoning` and
`resolved.approval` with last-wins duplicate semantics; `--permission-mode`
validates every occurrence against the provider's case-sensitive choices and
fails closed on an unknown mode, while `--effort` matches case-insensitively
and canonicalizes to its lowercase domain value. An unknown effort token is
not rejected — Claude launches, warns, and applies its own default — so the
token stays in the composed argv but `resolved.reasoning` reports the
provider-native fallback (`value` null, source `native`) instead of claiming
the ignored value is effective. An effective yolo resolves approval to
`bypass-permissions` because `--dangerously-skip-permissions` overrides an
explicit mode at Claude runtime.
An explicit sandbox/approval/permission-mode selection suppresses
project-config `yolo_mode` (reported as `suppressed_by_explicit_cli`), so the
composed argv never pairs a bypass flag with the user's explicit policy.

Error envelopes mirror the child contract: `unsupported_schema_version`,
`invalid_project_configuration`, `invalid_provider_arguments` (a pass-through
provider argument the provider's own parser would reject),
`provider_executable_not_found`. The default
`--mode child` contract above is unchanged. agents-infra stays board-agnostic:
it renders the plan; the caller owns board discovery, goals, and the launched
process.

`-d` expands to Codex `--dangerously-bypass-approvals-and-sandbox` or Claude
Code `--dangerously-skip-permissions` respectively. Each launcher can also
apply its own persistent `[agents.<provider>.primary_session].yolo_mode`
policy. If no project opt-in is found while walking upward, neither launcher
mounts anything — no `-c` overrides for Codex, no `--mcp-config` flag for
Claude Code.

LLDB MCP is available as an opt-in stdio server:

```toml
[mcp]
enabled_servers = ["lldb"]
```

LLDB's MCP integration uses `lldb-mcp`, which bridges stdio to the LLDB MCP
server socket. On macOS, `./setup.sh` installs Homebrew `llvm` when `lldb-mcp`
is missing and writes a narrow `$(brew --prefix)/bin/lldb-mcp` wrapper that
execs Homebrew's helper without overriding `LLDB_EXE_PATH`. This lets
`lldb-mcp` use the `lldb` binary next to itself by default, matching LLDB's
documented behavior. The wrapper also prunes dead-PID
`~/.lldb/lldb-mcp-*.json` discovery files before launch so stale sockets do not
break the MCP initialize handshake. Set `AGENTS_INFRA_SKIP_LLDB_MCP=1` to skip
that bootstrap. If a project uses an LLDB build with the helper elsewhere,
override the definition in the project-local
`.agents/.configs/codex-mcp-servers.toml`:

```toml
[servers.lldb]
command = "/path/to/lldb-mcp"
```

Safari MCP is available as an opt-in stdio server backed by Safari Technology
Preview's `safaridriver`:

```toml
[mcp]
enabled_servers = ["safari"]
```

The shared definition launches:

```toml
[servers.safari]
command = "/Applications/Safari Technology Preview.app/Contents/MacOS/safaridriver"
args = ["--mcp"]
```

Prerequisites:

- Install Safari Technology Preview 247 or newer.
- Enable `Safari Settings > Advanced > Show features for web developers`.
- Enable `Safari Settings > Developer > Enable remote automation and external agents`.

Safari remains project-local opt-in only. Do not add it to a global Codex or
Claude Code MCP config unless the user explicitly wants a user-managed global
server.

During `agents-infra setup local`, a non-empty `enabled_servers` list also
installs `.local/bin/codex-local` as a backward-compatible shim that delegates
to `agents-infra codex`. The project-local `agents-infra` helper preserves the
caller's working directory before it runs the source checkout with `go run`, so
`codex-local --print-config` should report the directory where the user invoked
it, not `.agents/tools/agents-infra`.

User-managed global MCP servers in the base Codex config, or in Claude Code's
own user/project scopes, remain that agent's own responsibility, not
agents-infra project opt-in state. The global Codex model/settings config
remains authoritative.

## Attachments

This repo defines a generic agent attachment contract:

- manifest file name: `agents-attachments-manifest.json`
- env var: `AGENTS_ATTACHMENTS_MANIFEST`
- helper CLI: `agents-attachments`, backed by `agents-infra attachments`

The repo does not itself ingest chat attachments. A separate runtime or launcher
must materialize files locally, write the manifest, and export the env var before
starting the agent process.

For Codex sessions, the helper can bootstrap a local manifest from rollout
history when `CODEX_THREAD_ID` is available:

```bash
agents-attachments materialize
```

For image intake, stage explicit paths or manifest references before inspection:

```bash
agents-attachments stage-images ./photo.heic screenshot.png --out-dir .temp/image-intake
agents-attachments stage-images --all --manifest .temp/agents-attachments-manifest.json --out-dir .temp/image-intake
```

`stage-images` keeps originals read-only, writes normalized/copied images under
the selected scratch directory, and emits `image-stage-map.json` with redacted
source labels, content hashes, staged filenames, and HEIC normalization details.
HEIC/HEIF inputs normalize to PNG with macOS `sips` first, then ImageMagick
(`magick`, then `convert`) as the portable fallback; missing converters fail
clearly.

Agents should inspect staged images directly through runtime vision first. OCR
is only a bounded fallback when direct inspection is insufficient. Observations
must cite the staged filename, evidence, confidence, uncertainty, and redactions;
do not persist raw ICCID, IMSI, QR payloads, activation codes, tokens, keys, or
password-like values extracted from images.

## Rules

`.rules/default.rules`: pre-approved Codex CLI commands:
- PlantUML download and rendering
- Temporary directory creation

## How It Works

After running `agents-infra setup global`:

```
~/.agents/
├── skills/
│   ├── relux-agents-infra -> ~/.agents/.skills/relux-agents-infra
│   ├── skill-creator -> ~/.agents/.skills/skill-creator
│   └── ...
├── .skills/
│   └── relux-agents-infra/  # Materialized SKILL.md + README.md; no ancestor-link cycle

~/.claude/
├── CLAUDE.md           # Loads @instructions/INSTRUCTIONS.md
├── instructions/ -> ~/.agents/.instructions/
└── skills/
    ├── relux-agents-infra -> ~/.agents/skills/relux-agents-infra
    ├── skill-creator/ -> ~/.agents/skills/skill-creator
    └── ...

~/.codex/
├── AGENTS.md           # Rendered from ~/.agents/.instructions/AGENTS.md
├── config.toml -> ~/.agents/.configs/codex-config.toml
├── skills/
│   └── ... -> ~/.agents/skills/...
└── rules/
    └── default.rules -> ~/.agents/.rules/default.rules
```

`~/.agents` is the installed runtime copy. It should not be used as a git checkout.

Meaning of the two skill trees:
- `.skills/` is the authoritative skill content that belongs to this repo, lives under its version control, and is synced into the installed runtime.
- `skills/` is the external runtime area for public skills and tooling. It may contain content that does not belong to `relux-agents-infra`. `setup` only refreshes the managed links it owns there and must not treat that directory as repo-owned content.

Before mutation, setup recursively validates every source-managed skill link it
can materialize. Setup's postcondition and `verify` repeat that check across the
managed installed surfaces. Links must remain contained, resolve successfully,
and form an acyclic directory graph; multiple contained links may share a target
when they form a DAG. Provider-owned top-level skill packages remain outside
this ownership boundary unless setup manages their name, in both global and
project-local runtimes.

Project-local install example:

```
project-root/
├── .agents/
│   ├── .instructions/       # Project-owned; not copied from global modules
│   ├── .configs/
│   ├── .scripts/
│   ├── .skills/
│   └── skills/
├── .claude/
│   ├── CLAUDE.md
│   ├── instructions/ -> .agents/.instructions/
│   └── skills/ -> .agents/skills/...
├── .codex/
│   ├── AGENTS.md       # Rendered Codex instructions
│   └── skills/ -> .agents/skills/...
├── AGENTS.md           # Rendered project-root Codex instructions
└── .local/bin/
    ├── agents-attachments # launcher for agents-infra attachments
    └── agents-infra       # launcher for the Go CLI
```

In local-project mode, treat `.agents/` as the installed source/runtime-common
tree, with `.agents/.instructions/` reserved for project-owned guidance.
`setup local` skips the source repo's global `.instructions/` tree, creates only
missing local entrypoints, and preserves every existing local instruction file
across resyncs. `.claude/` and `.codex/` are agent-specific runtime outputs.
Codex does not expand Claude-style `@...` include indexes, so `setup`
materializes `.codex/AGENTS.md` and project-root `AGENTS.md` as flattened
markdown. If a hand-written project-root `AGENTS.md` exists, `setup local`
preserves it as `.agents/.instructions/AGENTS.project.md` before rendering the
Codex-visible file.

## Adding New Skills

1. Create skill in `.skills/<skill-name>/`
2. Add `SKILL.md` with frontmatter
3. Run `agents-infra setup global` to propagate
4. `setup` will refresh the managed link in the installed external `skills/` area without replacing unrelated external skills

Use `./setup.sh` only as bootstrap when the `agents-infra` launcher is missing
or needs reinstalling. On Windows, use `.\setup.ps1` for the same bootstrap flow.

Or use the `skill-creator` skill:

```
/skill-creator
```

## Updating Instructions

Edit files in this source repo, then run `agents-infra setup global` to sync them
into `~/.agents` and refresh the installed runtime state.

## Git

This repo is version-controlled. Commit your changes:

```bash
cd /path/to/relux-agents-infra
git add -A
git commit -m "Update skills/instructions"
git push
agents-infra setup global
```

<!-- relux-ecosystem:start -->

## About Relux Works

This project is part of the open-source ecosystem of
[Relux Works](https://relux.works), an AI-native software development studio.
We build fixed-price MVPs, rescue vibe-coded apps, run local AI inference, and
train teams to work with coding agents. Much of the infrastructure behind that
work is open source.

- Full catalog: [relux.works/en/open-source](https://relux.works/en/open-source/)
- Agentic enablement: [agent harnesses & team training](https://relux.works/en/agentic-enablement/)
- Hire us the agent-native way: point your assistant at `https://api.relux.works/mcp`
- Contact: ivan@relux.works

<!-- relux-ecosystem:end -->
