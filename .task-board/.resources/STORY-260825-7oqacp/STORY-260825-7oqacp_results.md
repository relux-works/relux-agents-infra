# STORY-260825-7oqacp developer handoff evidence

Date: 2026-08-25 (Europe/Moscow)
Candidate: `5b081ad3ffd002634ed1e992c70a35dba839174f`
Branch: `task-board/story/STORY-260825-7oqacp`

## Outcome

The exact Story candidate preserves the independently reviewed Qwen/Pi
contract from `TASK-260825-kpky8f`:

- canonical Qwen reasoning `medium` requires Pi-native reasoning plus
  `qwen-chat-template` compatibility and reaches provider argv as
  `--thinking medium`;
- omitted or explicit-safe Pi yolo remains false with provenance;
- `yolo_mode = true` is refused as `pi_yolo_mode_unsupported` because pinned
  Pi `0.84.2` has no native unattended tool-execution policy and `--approve`
  controls project-local input trust instead;
- README, source `SKILL.md`, tests, and LOGBOOK remain in the same coherent
  13-path change already accepted by the leaf reviewer.

`main...HEAD` is `0 1`: this Story candidate is one commit ahead and has no
unrelated working-tree changes. `git diff --check main...HEAD` exited 0.

## Validation run directly on the exact candidate

All commands were standalone processes; no validation was piped through
`tee` or a status-masking pipeline.

| Command | Exit | Result |
| --- | ---: | --- |
| `go test ./internal/infra -run 'TestParsePi|TestPiPrimaryYolo|TestPiYoloTrue|TestCanonicalQwen|TestGeneratedPiCatalog' -count=1` | 0 | Focused Pi/Qwen config, native-thinking, yolo refusal, and catalog coverage passed. |
| `go test . -run 'TestRunComposeCanonicalQwen|TestPiOperatorContract|TestReluxAgentsInfraSkill|TestBoundedModelCheckREADME|TestReluxAgentsInfraSkillPinsBounded' -count=1` | 0 | Production compose and documentation-contract coverage passed. |
| `go test ./... -count=1` | 0 | Full module passed: root `84.049s`, attachments `1.702s`, infra `102.649s`. |
| `go vet ./...` | 0 | Vet clean. |
| `gofmt -l .` plus empty-output assertion | 0 / 0 | Formatting clean. |
| `go build ./...` | 0 | Native Darwin/arm64 build passed. |
| `GOOS=windows GOARCH=amd64 go build ./...` | 0 | Windows build passed. |
| `GOOS=linux GOARCH=amd64 go build ./...` | 0 | Linux build passed. |
| `agents-infra verify global` | 0 | Installed global runtime verified. |

The installed binary reports
`agents-infra v1.6.1-30-g5b081ad commit=5b081ad`, matching the Story
candidate.

## Canonical `/Users/alexis/src` rollout

The canonical config was already rolled forward by the accepted implementation
and was inspected without rewriting it. It contains:

- target reasoning `medium`;
- managed profile `reasoning = true`, `thinking = "medium"`, and
  `compat.thinking_format = "qwen-chat-template"`;
- explicit `agents.pi.primary_session.yolo_mode = false`;
- `qwen-infra` mapped to the canonical Qwen target.

A binary built from exact `5b081ad` ran the production non-launching compose
entry point:

```text
agents-infra compose --mode primary-session --entrypoint qwen-infra \
  --project /Users/alexis/src --schema-version 1 --json
```

It exited 0. A standalone `jq -e` assertion also exited 0 and proved:

- target and resolved reasoning are `medium`;
- reasoning provenance is
  `/Users/alexis/src/.agents/.configs/project-config.toml`;
- interactive and managed-host argv both end with `--thinking medium`;
- resolved yolo is explicit `false` from that same canonical config.

The source-built production `target qwen-infra --print-config` entry point also
exited 0 and reported the same target, reasoning provenance, and provider argv.

`pgrep -fl 'mlx_lm.server|llama-server'` exited 1 before and after these checks,
with empty output both times: no local model runtime existed or was launched.

## Negative evidence

The focused suites drive the production call sites named in the accepted leaf
review: `BuildPrimarySessionLaunchPlan`,
`buildPiPrimarySessionLaunchPlan`, `RunPi`,
`BuildCanonicalTargetLaunchPlan`, `ResolveCanonicalTarget`, and
`GeneratePiModelsJSON`. The accepted leaf-review evidence also records three
independent narrowing mutants covering the compose-only yolo call site, the
canonical Qwen capability condition, and the Pi parse gate.

This run additionally drove a minimal `yolo_mode = true` fixture through the
source-built production compose entry point. The command exited **1** as
expected-red, emitted code `pi_yolo_mode_unsupported`, named the configuration
source, explained that `--approve` is not unattended execution, and prescribed
false/omission. This is reported as a failing command by design, not as a pass.

## Handoff boundary

The prior Story review requested no code changes; it requested the missing
managed-worktree `story_final` Change Request. This developer handoff publishes
the complete one-commit Story candidate for a new independent reviewer. Story
integration remains the orchestrator's step after that Change Request is
accepted.

## Stop-The-Line blocker discovered during handoff

The required final command was run exactly:

```text
task-board handoff STORY-260825-7oqacp --role developer
```

It exited **1** and changed no status:

```text
cannot hand off STORY-260825-7oqacp: unchecked checklist items [4]
(A story-final Change Request is produced and independently accepted before
trunk integration.): handoff evidence missing
```

This is a lifecycle cycle, not missing implementation evidence:

1. Story checklist item 4 requires the `story_final` Change Request to have
   already been produced and independently accepted.
2. `task-board handoff` requires every checklist item to be checked before it
   moves the producer to `to-review`.
3. The task-board production completion path publishes a Change Request only
   after a managed-workspace producer reaches its configured end status and has
   attached a new outcome.
4. There is no public `task-board worktree` or Change Request command that lets
   this producer publish a `story_final` revision before handoff.
5. Independent acceptance additionally requires a reviewer run bound to that
   published revision, so it cannot exist before publication.

The failed assumptions were that the required handoff command would first
publish the candidate, or that a supported pre-handoff CR publication command
existed. CLI help and the production task-board source disprove both.

Rejected workarounds:

- checking item 4 while it is factually false would forge evidence;
- using direct `set_status(..., status=to-review)` would bypass the mandatory
  handoff evidence gate and violate the assigned final-command contract;
- changing relux-agents-infra product code cannot solve this board lifecycle
  cycle and would add unrelated repository changes.

Viable ownership choices are:

- move/split checklist item 4 so producer handoff requires publication-ready
  evidence, while independent `story_final` acceptance remains a reviewer /
  integration gate; this is the recommended clean contract;
- or add a supported task-board operation that publishes the producer CR before
  the handoff checklist is evaluated, while preserving reviewer authorization.

The exact external input needed is an orchestrator/task-board-owner decision
and board repair implementing one of those two contracts. Once repaired, rerun
the same developer handoff; no Qwen/Pi code rework is required.
