# TASK-260825-39ycg2 Documentation and Qwen Evidence

## Documentation

- `README.md` documents exact CLI syntax, repeatable expectations, artifact
  names/modes, secret handling, exit precedence, the `1ms..30m` Go-duration
  range, the default `5m` deadline, process-group cleanup reporting, retained
  evidence, and a Qwen skill-discovery command.
- `SKILL.md` documents when the checker is appropriate, the required
  setup/verify/print-config order, the bounded command, and the evidence rule
  that an exact file read must be established from a completed non-error raw
  event rather than inferred from a tool-name expectation or response marker.
- `LOGBOOK.md` records the observed read/timeout distinction without raw
  provider data.

## Installed-path gates

| Command | Exit | Result |
| --- | ---: | --- |
| `./setup.sh` | `0` | Built and installed `agents-infra` at commit `ac759d9`; bootstrap verified the global runtime. Homebrew refreshed metadata, found LLVM already current, and refreshed the existing `lldb-mcp` wrapper. |
| `agents-infra setup global --source-dir <story-worktree>` | `0` | Synced the final documentation bytes after the last prose correction. |
| `agents-infra verify global` | `0` | Installed global runtime verified. |
| `agents-infra version` | `0` | `v1.6.1-28-gac759d9`, commit `ac759d9`. |
| `qwen-infra --print-config` | `0` | Resolved canonical `qwen-infra` to managed Pi target `qwen-mlx-8bit`, profile `qwen-3.8-27b-mlx-8bit`, provider `local-qwen`, reasoning `off`, loopback endpoint `127.0.0.1:18011`. Host-specific source/model paths are omitted here. |
| `cmp -s SKILL.md $HOME/.agents/skills/relux-agents-infra/SKILL.md` | `0` | Installed skill is byte-identical to the source skill. |

Two help probes were diagnostics, not green gates: `qwen-infra --print-config
--help` and `agents-infra help` each returned exit `1` after printing their Go
flag/usage text. The installed general help included the new `model-check`
syntax.

## Source checks

| Command | Exit | Result |
| --- | ---: | --- |
| `go test . -run '^(TestPiOperatorContractDocumentsCycle10Boundary\|TestReluxAgentsInfraSkillRoutesSafePiWorkflowToSource)$' -count=1` | `0` | Documentation contract tests passed. |
| `go test . -run '^TestModelCheckProductionEntrypoint$' -count=1` | `0` | Production CLI behavior, artifacts, exits, timeout, overwrite refusal, and cleanup fixtures passed. |
| `go test ./internal/infra -run '^TestModelCheckCleanupAttestationRefusesUnconfirmedStates$' -count=1` | `0` | Cleanup evaluator refused unconfirmed states. |
| `git diff --check -- README.md SKILL.md LOGBOOK.md` | `0` | Documentation diff has no whitespace errors. |
| Per-file `git hash-object` versus `git rev-parse HEAD:<path>` for all eight checker/runtime code files visible in Story-worktree status | `0` | Every current code file is byte-identical to `HEAD`; this doc-writer changed no code. The staged-delete/untracked pairs are pre-existing Story Change Request index bookkeeping. |

## Real Qwen skill-discovery smoke

Executed from the Story worktree through the installed production command:

```bash
agents-infra model-check \
  --target qwen-infra \
  --prompt 'Discover the applicable installed skill for updating shared agent infrastructure. Use the read tool to read its SKILL.md. Reply with RELUX_SKILL_READ_CONFIRMED and one source-of-truth rule learned from that file.' \
  --output-dir .temp/TASK-260825-39ycg2/qwen-skill-discovery-01 \
  --deadline 5m \
  --expect-tool read \
  --expect-text RELUX_SKILL_READ_CONFIRMED
```

Revision ordering: this smoke read the installed `SKILL.md` revision immediately
before the last source/installed prose sync. That later revision was 42 bytes
longer and changed only wrapping in the same bounded-check section; the observed
completed read and the skill-discovery conclusion are unaffected. The final
source and installed skill were subsequently verified byte-identical with
`cmp -s` as reported above, but the smoke itself was not rerun against those
last prose bytes.

Persisted checker outcome:

| Field | Observed value |
| --- | --- |
| `status` / `exit_code` | `timed_out` / `2` |
| `deadline_ms` / `duration_ms` | `300000` / `300192` |
| `process_exit_code` | `unknown` (the managed call ended by context deadline, not an `exec.ExitError`) |
| Event stream | Valid, incomplete |
| Tool call | `read`, completed, `failed=false` |
| Tool expectation | `read`: met |
| Text expectation | `RELUX_SKILL_READ_CONFIRMED`: unmet |
| Managed cleanup | Confirmed |
| Pi/runtime process-group cleanup | `confirmed` / `confirmed` |

Sanitized projection of the only tool call, correlated by the same raw call ID:

```text
tool_execution_start toolName=read path=$HOME/.agents/skills/relux-agents-infra/SKILL.md
tool_execution_end   toolName=read isError=false
```

Conclusion: the model discovered and successfully read the installed
`relux-agents-infra/SKILL.md`. The overall behavior check did not pass: the
final marker was not produced and the lifecycle remained incomplete before the
five-minute deadline. The successful read is therefore proven; final-response
behavior is not proven. The timeout is a failure, not a passing smoke.

Raw mode-`0600` `events.jsonl` and `stderr.log` remain only in the local
task-scoped output directory. They are not attached because they contain raw
provider/tool bytes. Artifact SHA-256 values retained for local correlation:

```text
events.jsonl  f089f23d617c670b3c4af91a2cbd11a07f0cf1e2fd1003ecee995720c76bcbc6
stderr.log    a9da0b2d7bf5325258199604cc00f0ef40fbaa82990fcb99e012fa32efa4055f
summary.json  c320bf117c109ef4c83ccef265d785dc0f097a08246fcbfcf65ca3500ba60568
summary.txt   89286380f2acffb1555a33fbb4db368a4aa672d7178cbce0521624a2cc696a49
```
