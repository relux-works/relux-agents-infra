# TASK-260825-2osq67 rollout evidence

Recorded: 2026-08-25 20:47:13 +0300

## Accepted source and installation

- Managed Story workspace: `/Users/alexis/src/relux-works/relux-agents-infra/.temp/STORY-260825-7oqacp/worktree`
- Branch: `task-board/story/STORY-260825-7oqacp`
- Accepted source commit: `5b081ad` (`Support native Qwen thinking and reject Pi yolo`)
- Workspace state before installation: clean; `0` commits behind `main`, `1` commit ahead.
- `agents-infra setup global --source-dir /Users/alexis/src/relux-works/relux-agents-infra/.temp/STORY-260825-7oqacp/worktree`: exit `0`. This synchronized the runtime tree but, by contract, did not replace the bootstrap-owned CLI binary.
- `./setup.sh` from the accepted Story workspace: exit `0`. It rebuilt and installed the CLI, wrote install state, and ran the supported global setup flow.
- Installed version: `agents-infra v1.6.1-30-g5b081ad commit=5b081ad`.
- `agents-infra verify global`: exit `0`, `verified global agent runtime: /Users/alexis/.agents`.

## Canonical `/Users/alexis/src` policy

Updated only `/Users/alexis/src/.agents/.configs/project-config.toml` outside generated installation artifacts:

- `agents.pi.primary_session.yolo_mode = false`
- selected Qwen Pi profile `reasoning = true`
- selected Qwen Pi profile `thinking = "medium"`
- canonical Qwen target `reasoning = "medium"`
- retained `supports_reasoning_effort = false`; native reasoning is carried by `reasoning=true`, `compat.thinking_format="qwen-chat-template"`, and `--thinking medium`, not by that legacy capability flag.

## Non-launching production checks

`qwen-infra --print-config` from `/Users/alexis/src`: exit `0`.

Observed production fields:

```text
entrypoint_source: /Users/alexis/src/.agents/.configs/project-config.toml
target_source: /Users/alexis/src/.agents/.configs/project-config.toml
reasoning: medium
effective_reasoning: medium
effective_reasoning_source: /Users/alexis/src/.agents/.configs/project-config.toml
effective_profile_source: /Users/alexis/src/.agents/.configs/project-config.toml
provider_argv:
  - "--thinking"
  - "medium"
```

`agents-infra compose --mode primary-session --entrypoint qwen-infra --project /Users/alexis/src --schema-version 1 --json`: exit `0`.

Observed schema-v1 launch-plan facts:

- producer commit: `5b081ad`
- `target.reasoning = "medium"`
- `resolved.reasoning.value = "medium"`
- `resolved.reasoning.source = /Users/alexis/src/.agents/.configs/project-config.toml`
- interactive and managed-host argv contain exactly `--thinking`, `medium`
- `resolved.yolo.value = false`
- `resolved.yolo.source = /Users/alexis/src/.agents/.configs/project-config.toml`
- managed Pi executable identity and 217-file manifest report `observed_state = "verified"`
- the command only composed a launch plan; it did not launch Pi or the local model runtime.

## Negative configuration gate

A child fixture inherited the canonical Pi profile and overrode only:

```toml
[agents.pi.primary_session]
yolo_mode = true
```

Direct command:

```text
agents-infra compose --mode primary-session --agent pi --project /Users/alexis/src/.temp/TASK-260825-2osq67/pi-yolo-true-fixture --schema-version 1 --json
```

Real result: exit `1` (expected-red refusal), contract status `error`, error code `pi_yolo_mode_unsupported`. The diagnostic states that pinned Pi v0.84.2 `--approve` controls project-local input trust rather than unattended tool execution. This was configuration-only compose; no launch occurred.

Focused source assertions were also rerun directly:

```text
go test ./internal/infra -run 'TestPiPrimaryYoloSafeDefaultAndNearestFalseMask|TestPiYoloTrueFailsClosedBeforeComposeOrLaunchLookup|TestCanonicalQwenPlanProvesProfileDerivedIdentityAndEndpointInvariants|TestCanonicalQwenProfileAssertionsFailClosed' -count=1
```

Result: exit `0`, package `ok` in `0.614s`.

## No-runtime evidence

`pgrep -fl 'mlx_lm.server|llama-server'` ran before and after the non-launching rollout checks. Both invocations returned exit `1` with no output, the expected absence result. No local model process was started.

## Logbook

The accepted Story source already records the governing findings in `LOGBOOK.md` under `2026-08-25`: “Qwen Thinking Is Native, Pi Yolo Is Not” and “Pi Approve Is Project Trust, Not Yolo.” No duplicate repository logbook entry was added by this metadata-only rollout.
