# TASK-260824-2a4gk3 — deployment and live-smoke evidence

## Result

`casual-talks` now has verified global and project-local canonical target
entrypoints for OpenAI/Codex, Anthropic/Claude Code, and Qwen/Pi. The Qwen
deployment produced a real text response, a successful `write` tool result, a
successful `read` tool result, and deterministic listener/process/lock cleanup.

The predecessor rollout `TASK-260824-1jjze0` authored the target configuration
before this deployment task started. Per its operator-approved architecture
decision, the Qwen target/profile model identity is the exact resolved MLX
weights path, because that is the identity used by `mlx_lm.server` for load,
request, and `/v1/models` readiness:

```text
/Users/alexis/src/local-models/Qwen3.8-27B-Uncensored-MLX-8bit
```

No project-config bytes changed during setup, diagnostics, compose, or live
launch. SHA-256 remained:

```text
464c699f4dfe505203bc0ac80abb05238f6e04ef645ff36dba65e86b6f26b7b6
```

## Canonical target and provenance assertions

All human `--print-config` and schema-v1 JSON compose commands exited 0.

| Entrypoint | Vendor / environment | Model | Reasoning | Target / effective source |
| --- | --- | --- | --- | --- |
| `openai-infra` | `openai` / `codex` | `gpt-5.6-sol` | `high` | `/Users/alexis/src/casual-talks/.agents/.configs/project-config.toml` |
| `anthropic-infra` | `anthropic` / `claude-code` | `claude-opus-5` | `high` | same project config |
| `qwen-infra` | `qwen` / `pi` | exact absolute MLX weights path above | `off` | same project config |

Qwen Section 5 invariants were parsed from the production compose document:

- target profile: `qwen-3.8-27b-mlx-8bit`
- target/effective profile provider: `local-qwen`
- target/effective/runtime endpoint: `http://127.0.0.1:18011/v1`
- effective model: `local-qwen//Users/alexis/src/local-models/Qwen3.8-27B-Uncensored-MLX-8bit`
- profile and all effective Qwen coordinates source from the selected complete
  project profile definition
- runtime argv contains the exact model path, `--host 127.0.0.1`, and
  `--port 18011`
- Pi catalog: v0.84.2 darwin-arm64, 217 files, manifest SHA-256
  `2f68ab1b3f28a9c4b8995f91984f8f47001a79735da7e57aa7fe6d223f90378b`
- requested capabilities remain `text` and `tools`; verification is correctly
  reported as `not-claimed` until the live smoke below

Commands:

```text
openai-infra --print-config                                      exit 0
anthropic-infra --print-config                                   exit 0
qwen-infra --print-config                                        exit 0
agents-infra compose ... --entrypoint openai-infra ... --json    exit 0
agents-infra compose ... --entrypoint anthropic-infra ... --json exit 0
agents-infra compose ... --entrypoint qwen-infra ... --json      exit 0
```

Before and after these non-launching diagnostics, `lsof` found no listener on
TCP 18011 and `pgrep` found no `mlx_lm.server`; both commands exited 1, the
expected-red result that proves absence.

## Installed alias artifacts

Global and local target alias scripts are byte-identical, regular executable
files. Each wrapper contains only its entrypoint and exact sibling
`agents-infra` dispatch. The local `agents-infra` launcher records the Story
worktree as `AGENTS_INFRA_SOURCE_DIR`; the global sibling is the installed
arm64 binary.

| Artifact | Size | SHA-256 |
| --- | ---: | --- |
| global `agents-infra` | 10,630,802 bytes | `98cc4dad62fc898af44ee9b81bfc3180cdcfe8fe3ccd1b573a6b8aa1f2532826` |
| local `agents-infra` launcher | 475 bytes | `f3982633685fb85d541af53ed1c07b57101f0708197344282c6f00d7ff9b57b8` |
| global/local `pi-infra` | 221 bytes | `f3c6313bba49fbc941f847a207f5d1cb011e14f3e2caf27657b773d4f170f57b` |
| global/local `openai-infra` | 292 bytes | `fa0596ed7a039dbe513f49d30fc5a0115cad9f50f84c4e3c7930bbfd7938f173` |
| global/local `anthropic-infra` | 298 bytes | `d1acbb048939dfcbbbd924f83dcd7cf879b77328e3e61cbcd818ea4fe40c59c7` |
| global/local `qwen-infra` | 288 bytes | `9ee57860a93ea473c1481029dd34998fa52614eb106c8d5edd6ec0acd349b60a` |

Installed producer evidence:

```text
agents-infra v1.6.1-25-gf81197f commit=f81197f
```

## Setup gate and source fix

The first local setup invocation exited 1 after exposing a real reusable
infra defect: local verification inspected every provider-owned top-level
skill link, although the documented ownership boundary only covers names
managed by `.agents/.skills`. This rejected the pre-existing
`casual-talks` `mac-infra` links into the global runtime.

The source fix:

- derives the managed-name set from `.agents/.skills` in both global and local
  modes;
- preserves provider-owned names outside that set;
- continues recursively validating every source-managed package and every
  managed fanout surface;
- adds an installed production CLI regression for preserved unmanaged links;
- changes the existing negative test to replace the actual managed
  `relux-agents-infra` name rather than an unowned probe name;
- updates README ownership wording and records the root cause in `LOGBOOK.md`.

The two observed `mac-infra` links remained exactly
`/Users/alexis/.agents/skills/mac-infra` after the successful setup and verify.

Final setup gates:

| Command | Exit | Evidence |
| --- | ---: | --- |
| `AGENTS_INFRA_SKIP_LLDB_MCP=1 ./setup.sh` | 0 | built and installed current source; verified binary/global setup |
| `agents-infra setup global --source-dir <Story worktree>` | 0 | installed all canonical aliases |
| `agents-infra verify global` | 0 | verified `/Users/alexis/.agents` |
| `agents-infra setup local /Users/alexis/src/casual-talks --source-dir <Story worktree>` | 0 | installed/repaired local runtime and aliases |
| `agents-infra verify local /Users/alexis/src/casual-talks` | 0 | verified local runtime, including after live smoke |

LLDB bootstrap was explicitly skipped because it is unrelated to this target
deployment.

## Source validation and negative evidence

Production call site: installed `agents-infra verify local` reaching
`managedSkillLinkFailures`.

| Command | Exit | Result |
| --- | ---: | --- |
| first focused test compile | 1 | honest development failure: test referenced unexported package helpers; corrected before evidence |
| focused installed CLI tests after correction | 0 | unmanaged external link preserved; managed links checked on every surface |
| narrowing mutant: remove `relux-agents-infra` from managed-name gate | 1 | expected-red; negative test reported that `.agents/skills` was wrongly admitted |
| byte-for-byte source restore (`cmp`) | 0 | mutant removed using saved copy, not Git restore |
| focused installed CLI tests after restore | 0 | both tests passed |
| `go test ./... -count=1` | 0 | all three packages passed; named package durations 82.385s, 1.887s, 122.626s |
| `go vet ./...` | 0 | clean |
| `go build ./...` | 0 | build succeeded |
| `gofmt -l <changed Go files>` | 0 | empty output |
| `git diff --check` | 0 | clean |

One earlier full-suite invocation lost its process exit from the tool wrapper;
it was not accepted as gate evidence. The exact command was rerun in the
foreground and the exit-0 result above was captured through terminal process
completion.

## Live Qwen/Pi text and filesystem-tool smoke

Runtime/platform evidence:

| Component | Value |
| --- | --- |
| macOS | 26.5.1 (25F80), arm64 |
| Hardware | Apple M1 Max, 64 GiB |
| Pi | 0.84.2 |
| mlx-lm | 0.31.3 |
| agents-infra | v1.6.1-25-gf81197f |

The foreground command used `qwen-infra`, Pi JSON print mode, no session,
context/extensions/skills/templates/themes disabled, and an exact
`write,read` tool allowlist. It exited 0. Runtime logs show bind only on
`127.0.0.1:18011`, readiness `GET /v1/models` HTTP 200, and subsequent
`POST /v1/chat/completions` HTTP 200 responses.

Observed production events:

- assistant text: `TEXT_RESPONSE_OK`
- `write` tool call on
  `/Users/alexis/src/casual-talks/.temp/TASK-260824-2a4gk3/qwen-tool-roundtrip.txt`
- write result: successful, 39 bytes
- `read` tool call on the same task-scoped path
- read result: `TASK-260824-2a4gk3 qwen tool roundtrip\n`, `isError=false`
- final assistant text: `TOOL_ROUNDTRIP_OK`
- provider/model: `local-qwen` and the exact absolute target model
- usage reported `reasoning=0`; launch plan and Pi argv both resolved
  reasoning/thinking to `off`

Independent on-disk verification:

```text
size:   39 bytes
sha256: e94776d59d7a2ac9289024d3e0bd409bf1e821cf62ee08b8fe7e082fa65f03d7
tail:   one LF byte (0a)
```

Post-run cleanup probes:

| Probe | Exit | Meaning |
| --- | ---: | --- |
| `lsof -nP -iTCP:18011 -sTCP:LISTEN` | 1 | expected-red: no listener |
| `pgrep -lf mlx_lm.server` | 1 | expected-red: no runtime process |
| `lsof <profile session.lock>` | 1 | expected-red: no lock holder |
| project-config SHA-256 | 0 | unchanged from preflight |

Only the names of runtime-affecting environment variables were inspected.
`HF_ENDPOINT`, `MODEL_ENDPOINT`, `GGML_BACKEND_PATH`, and `LLAMA_API_KEY` were
all unset before and after launch. No environment values, credentials, or
secrets were persisted in this artifact.

## Repository scope

Source changes in the Story worktree:

- `tools/agents-infra/internal/infra/skill_link_validation.go`
- `tools/agents-infra/installed_binary_setup_test.go`
- `README.md`
- two append-only entries in `LOGBOOK.md`

`casual-talks` had substantial pre-existing tracked and untracked work. Its
`git status --short` line set was identical before and after this deployment;
the config hash and external `mac-infra` link targets were also unchanged.
No foreign work was staged, committed, reset, or removed.
