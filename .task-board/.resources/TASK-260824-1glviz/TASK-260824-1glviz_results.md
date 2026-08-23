# TASK-260824-1glviz — reconciled vendor-target candidate

## Result

The accepted candidate tree `95d12fb4c2aaf6050ff51f2c74fee7a81041acff`
was mechanically reconstructed on current `main`
`2f74fb0c757c3d3038d744a054e0ce1b68656df7`, using authoritative base
`cf21665dde35274cc14e66e26a93574e0c18c15c`. The resulting tree is
`dd84306b4db3edf703c787971f24bca6bc09e81e`.

The result changes exactly the accepted 20 repository paths and no
`.task-board/**` path. No accepted behavior was redesigned.

## Reconciliation proof

- 15 production/test paths untouched by current main are byte-identical to the
  accepted candidate tree.
- `LOGBOOK.md` is also byte-identical to the accepted candidate. The upstream
  reviewer proved it is a strict superset of current main and includes the
  `1815` and `1753` entries byte-for-byte.
- `README.md`, `SKILL.md`, `tools/agents-infra/internal/infra/infra.go`, and
  `tools/agents-infra/internal/infra/infra_test.go` are the conflict-free
  file-level three-way results of current main / authoritative base / accepted
  candidate.
- Current-main-only `.configs/codex-config.toml`,
  `internal/infra/codex_config.go`, and `setup_test.go` remain byte-identical to
  current main.
- Alternate-index tree construction reports 20 changed paths, zero board
  paths, and leaves the working index untouched.

Detailed blob IDs are in `identity-proof-02.log`; final tree identity and path
set are in `reconciled-tree-identity-01.log` and
`reconciled-changed-paths-01.txt`.

## Source validation

Every command below ran directly and completed with the stated exit code.

| Command | Exit | Result |
| --- | ---: | --- |
| focused installed-binary unmanaged/managed skill-link tests | 0 | Production setup/verify containment tests passed. |
| `go test -count=1 ./...` | 0 | Packages passed in 81.719s, 1.399s, and 130.079s. |
| `go vet ./...` | 0 | Clean. |
| `go build ./...` | 0 | Build succeeded. |
| `gofmt -l .` plus empty-output assertion | 0 / 0 | No formatting drift. |
| `git diff --check` | 0 | Clean. |
| focused current-main fast-profile root tests | 0 | Standard tier/no fast profile and README removal preserved. |
| focused current-main managed-config migration tests | 0 | User-state preservation and malformed-config refusal preserved. |

## Setup and installed runtime

| Command | Exit |
| --- | ---: |
| `AGENTS_INFRA_SKIP_LLDB_MCP=1 ./setup.sh` | 0 |
| `agents-infra setup global --source-dir <worktree>` | 0 |
| `agents-infra verify global` | 0 |
| `agents-infra setup local /Users/alexis/src/casual-talks --source-dir <worktree>` | 0 |
| `agents-infra verify local /Users/alexis/src/casual-talks` | 0 |

LLDB bootstrap was intentionally skipped because it is unrelated to canonical
vendor targets. The `casual-talks` project-config SHA-256 remained
`464c699f4dfe505203bc0ac80abb05238f6e04ef645ff36dba65e86b6f26b7b6`, and
its complete `git status --short` line set was unchanged.

Global/local `pi-infra`, `openai-infra`, `anthropic-infra`, and `qwen-infra`
are regular executable files and byte-identical to their matching sibling.
All three target aliases passed `--print-config` and schema-v1 primary-session
compose. Machine assertions passed for:

- OpenAI / Codex / `gpt-5.6-sol` / `high`;
- Anthropic / Claude Code / `claude-opus-5` / `high`;
- Qwen / Pi / the exact MLX weights path / `off`, including profile/provider,
  endpoint, runtime argv, 217-file Pi identity, and requested-but-not-attested
  capability semantics.

All target and effective identity fields resolve to the `casual-talks` project
config. No `env` node or required environment name appears in the compose
documents.

## Identity-lock and cleanup evidence

The installed production aliases were attacked directly. Nine refusal probes
returned real exit `1` as expected: divergent OpenAI model, post-delimiter
OpenAI profile, Codex config model override, post-delimiter Anthropic model,
divergent Anthropic effort, divergent Qwen thinking, divergent Qwen provider,
unknown entrypoint, and conflicting compose selectors. Stable refusal codes
were `target_identity_conflict` or `unknown_entrypoint`; the selector conflict
named the mutually exclusive selectors. Exact OpenAI model and Anthropic effort
repeats each returned exit `0`. The aggregate assertion returned exit `0`.

Post-probe absence commands are intentionally reported as failing commands:

| Probe | Exit | Expected observation |
| --- | ---: | --- |
| `lsof -nP -iTCP:18011 -sTCP:LISTEN` | 1 | No listener. |
| `pgrep -af 'mlx_lm\\.server.*--port 18011'` | 1 | No runtime process. |
| `lsof <profile session.lock>` | 1 | No lock holder. |

The separate cleanup assertion, project-config hash comparison, and project
status comparison all returned exit `0`.

## Evidence notes

Two diagnostic harness mistakes were not counted as validation: an initial zsh
probe used the special variable name `path` and lost command lookup, and an
initial identity summary glob included unrelated logs. Both were recorded,
corrected, and rerun with direct commands and explicit real exit codes. No
repository file was changed by either failed harness.

The accepted upstream independent review is attached separately. It proves
candidate identity, attacks the production gate with three narrowing mutants,
recomputes alias provenance and identity locks, and independently reproduces a
live Qwen text/tool round trip. This carry task intentionally reran the full
source/setup/verify/provenance/lock/cleanup surface rather than claiming the old
validation applied to the new tree.
