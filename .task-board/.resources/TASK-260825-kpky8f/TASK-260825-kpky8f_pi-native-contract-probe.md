# TASK-260825-kpky8f Pi native contract probe

Date: 2026-08-25 (Europe/Moscow)

## Pinned identity

- `command -v pi`: exit 0, installed standalone launcher found.
- `pi --version`: exit 0, version `0.84.2`.
- `qwen-infra --print-config`: exit 0, baseline target resolved from the
  project-local `.agents/.configs/project-config.toml`; baseline reasoning was
  `off`, its source was reported, and provider argv contained `--thinking off`.
- `agents-infra compose --mode primary-session --entrypoint qwen-infra ...`:
  exit 0; verified standalone Pi identity, 217-file manifest, and the same
  reasoning/source/argv. Host-specific model and runtime paths are omitted
  from this persisted projection.

## Native reasoning evidence

- `pi --help`: exit 0; `--thinking <level>` accepts `off`, `minimal`, `low`,
  `medium`, `high`, `xhigh`, and `max`.
- Pinned `docs/models.md` states that `reasoning=true` advertises extended
  thinking and `compat.thinkingFormat=qwen-chat-template` is the local Qwen
  request mechanism.
- Pinned executable contents show the production request branch:
  `thinkingFormat === "qwen-chat-template" && model.reasoning` produces
  `chat_template_kwargs.enable_thinking = !!reasoningEffort` and
  `preserve_thinking = true`.
- Therefore a real medium Qwen contract requires all three coordinates:
  generated model `reasoning=true`, profile thinking `medium`, and launch argv
  `--thinking medium`, with `qwen-chat-template` compatibility.

## Native unattended-execution evidence

- Pinned `docs/usage.md` and `docs/security.md` define `--approve`/`-a` only as
  one-run trust for project-local settings/resources/instructions/packages.
- Pinned executable parsing maps `--approve` to `projectTrustOverride=true`.
- No native Pi option or policy in the pinned help/docs represents an
  unattended tool-execution bypass. Pi tool execution has no separate native
  approval mode that `agents.pi.primary_session.yolo_mode` can select.
- Decision: never translate Pi yolo to `--approve`; reject explicit
  `yolo_mode=true` as unsupported before executable lookup/launch. Preserve
  omitted/false behavior.

## Failed probes

- `task-board version`: exit 1 because this CLI has no version subcommand;
  board readiness was established through successful query/mutation calls.
- Reading the composed `models.json` path: exit 1 because non-launching compose
  correctly did not create managed state; absence is expected and was not
  treated as evidence of generated contents.
