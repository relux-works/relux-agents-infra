# TASK-260825-2osq67 review verdict — ACCEPTED

Reviewer run: RUN-260825-1cc557 (claude-opus-5/high). Read-only; no code, config,
or board file was modified by this review other than the board lifecycle calls.

Every claim below was re-executed by the reviewer against the **installed**
runtime (`/Users/alexis/.local/bin/agents-infra`, `v1.6.1-30-g5b081ad`), not read
from the producer artifact.

## 1. Accepted source and installed runtime — verified

| Fact | Producer claim | Reviewer observation |
| --- | --- | --- |
| Story worktree HEAD | `5b081ad` | `5b081ad Support native Qwen thinking and reject Pi yolo` |
| Worktree cleanliness | clean, 0 behind main, 1 ahead | `git status --short` empty; `rev-list --left-right --count main...HEAD` = `0 1` |
| Installed binary | `v1.6.1-30-g5b081ad commit=5b081ad` | identical |
| `agents-infra verify global` | exit 0 | exit 0, `verified global agent runtime: /Users/alexis/.agents` |

Installed tree really carries the accepted source, not a stale copy: `README.md`,
`SKILL.md`, and `LOGBOOK.md` under `~/.agents` are byte-identical to the worktree
at `5b081ad`, and `~/.agents/.skills/relux-agents-infra/SKILL.md` matches too.
`agents-infra doctor global` reports links/render state healthy.

## 2. Canonical config delta — exactly the mandated four lines

The producer spawn log carries the real pre/post diff of
`/Users/alexis/src/.agents/.configs/project-config.toml`
(`325de464… → b8139eb8…`). It is exactly:

- `+ agents.pi.primary_session.yolo_mode = false`
- profile `reasoning = false → true`
- profile `thinking = "off" → "medium"`
- target `qwen-mlx-8bit` `reasoning = "off" → "medium"`

Nothing else. The pre-existing `agents.codex.primary_session.yolo_mode = true`
and `agents.claude.primary_session.yolo_mode = true` lines sit above the first
hunk and were **not** touched by this rollout — no unattended-execution policy
was widened as a side effect. Live file content matches the post-image.

## 3. AC — production output re-run by the reviewer

`qwen-infra --print-config` from `/Users/alexis/src`, exit 0:

```
target_source: /Users/alexis/src/.agents/.configs/project-config.toml
reasoning: medium
effective_reasoning: medium
effective_reasoning_source: /Users/alexis/src/.agents/.configs/project-config.toml
provider_argv: --provider local-qwen --model <qwen> --thinking medium
```

`agents-infra compose --mode primary-session --entrypoint qwen-infra --project
/Users/alexis/src --schema-version 1 --json`, exit 0:

- `producer.commit = 5b081ad`
- `target.reasoning = "medium"`
- `resolved.reasoning = {value: "medium", source: <canonical config>}`
- `resolved.yolo = {value: false, source: <canonical config>}`
- `interactive.argv` and `managed_host.argv` both end in `--thinking medium`

The plan's argv is not a diagnostic-only fiction: `BuildManagedPiArguments`
emits one canonical `--provider/--model/--thinking <effectiveThinking>` triple
into **both** `Argv` (what `pi_launch_posix.go:236` actually execs) and
`DiagnosticArgv` (what the plan reports), and with no user args the two are
identical. `managed_client.argv` is empty by design for Pi — Pi carries argv on
the PTY host variant, not a separate client.

## 4. Gate attack — the licence was exercised, not skipped

### 4.1 `pi_yolo_mode_unsupported` reachable from every production entry

Fixtures under `/Users/alexis/src/.temp/TASK-260825-2osq67-review/` inherit the
canonical config and override only `agents.pi.primary_session.yolo_mode = true`.

| Attack | Entry point | Result |
| --- | --- | --- |
| direct child sets true | `compose --mode primary-session --agent pi` | exit 1, `pi_yolo_mode_unsupported` |
| same, canonical entrypoint | `compose --entrypoint qwen-infra` | exit 1, `pi_yolo_mode_unsupported` |
| nested child true under parent explicitly false | `compose --agent pi` | exit 1, `pi_yolo_mode_unsupported` (last-wins does not launder true) |
| true with **no** profile selected anywhere | `compose --agent pi` | exit 1, `pi_yolo_mode_unsupported` (gate is not profile-gated) |
| **real launch entry** | `agents-infra pi` | exit 1, `pi_yolo_mode_unsupported` |
| **real launch entry** | `agents-infra target qwen-infra` | exit 1, `pi_yolo_mode_unsupported` |
| launcher print path | `target qwen-infra --print-config` | exit 1, `pi_yolo_mode_unsupported` |

Grep confirms the guard is called from all three production paths and from
nowhere else: `pi_plan.go:118`, `pi_launch_posix.go:73`,
`pi_platform_windows.go:101`. It is not a unit-tested orphan.

### 4.2 Ordering proven with a control, not asserted

`agents-infra pi` was run twice with `PATH=/usr/bin:/bin` (no `pi` on PATH):

- yolo=true fixture → `pi_yolo_mode_unsupported`
- canonical yolo=false → `exec: "pi": executable file not found in $PATH`

The identical invocation reaches executable lookup only when yolo is false, so
the refusal genuinely precedes lookup and launch. No process was spawned in
either case. There is no `-d/--danger/--yolo` flag on the `pi` launcher (that
escape hatch exists only for `codex`/`claude`), so there is no CLI bypass.

### 4.3 The reasoning settings are load-bearing, not decorative (narrowing mutants)

Three single-field mutants of the canonical config, each failing closed with a
distinct code — a delete-only mutant would not have shown this:

| Mutant | Result |
| --- | --- |
| target `reasoning="high"`, profile `thinking="medium"` | exit 1 `invalid_target`, field `agents.targets.qwen-mlx-8bit.reasoning` |
| profile `reasoning=false`, target `reasoning="medium"` | exit 1 `invalid_project_configuration`, "must be true when thinking is not off" |
| `compat.thinking_format="anthropic-chat"` | exit 1 `invalid_target`, field `…compat.thinking_format` |

And the flag value tracks config rather than being hardcoded: an `off/off`
mutant composes `status: ok` with `resolved.reasoning=off` and argv
`--thinking off`, while canonical composes `medium`.

### 4.4 Operator override cannot silently contradict reported provenance

`compose --entrypoint qwen-infra -- --thinking high` → exit 1,
`target_identity_conflict` on `provider_args.reasoning`. Repeating the
configured value exactly (`-- --thinking medium`) composes `ok`. So the plan can
never report `reasoning: medium` while launching something else.

## 5. No local model runtime started

`pgrep -fl 'mlx_lm.server|llama-server'` before, between, and after every check:
exit 1, no output; `ps aux | grep '[m]lx_lm'` empty. Every launch-path probe was
either a compose (plan only) or a launch entry deliberately run with `pi` off
PATH, so nothing could have reached `mlx_lm.server`.

## 6. Tests

Full module suite re-run by the reviewer in the accepted worktree
(`tools/agents-infra`, `go test ./...`): exit 0 — `agents-infra 81.722s`,
`internal/attachments 1.784s`, `internal/infra 103.533s`.

## 7. Blast radius

Bounded scan of `/Users/alexis/src` (`find -maxdepth 4 -name project-config.toml`,
excluding `node_modules` and `.temp`; 28 files) for `agents.pi.primary_session`
yolo: the canonical config is the only one that sets it, and it sets `false`. No
sibling project is newly fail-closed by this rollout. This is a depth-4 scan, not
an exhaustive tree walk — deeper configs were not enumerated.
There is no `~/.agents/.configs/project-config.toml`, so nothing at the home
layer competes with the canonical file.

## 8. Follow-up for the commit-owning mover (does not block acceptance)

The install pointed the machine at a **disposable** source tree, and the
producer artifact does not mention this:

- `~/Library/Application Support/agents-infra/install.json` →
  `repoPath = /Users/alexis/src/relux-works/relux-agents-infra/.temp/STORY-260825-7oqacp/worktree`
- `~/.agents/.agents-infra-install.json` → same path as `sourceDir`

The DoD explicitly required installing from the Story worktree, so this is
intended state, and `ResolveSourceDir` treats install-state `repoPath` as a
*discovered* candidate that is skipped when unusable and falls through to the
installed `~/.agents` runtime — so removing the worktree degrades gracefully
rather than breaking setup. Still: after STORY-260825-7oqacp merges to `main` and
the worktree is removed, re-run `./setup.sh` from the main checkout so the
machine-scoped `repoPath` and the runtime receipt name a durable source.

## Verdict

**Accepted.** Every DoD item is satisfied and independently reproduced; the
refusal, validation, and provenance behavior was attacked from five production
entry points plus four config mutants and an operator-flag override, and failed
closed every time with a control proving the ordering. Reviewer-archetype run —
no `commit_ack` supplied; §8 is handed to the commit-owning mover.

Reviewer reproduction fixtures (disposable):
`/Users/alexis/src/.temp/TASK-260825-2osq67-review/`
