# TASK-260817-3bgn2f — Reviewer Verdict

**Verdict: changes requested → `to-dev`**

Six of seven acceptance criteria hold and were verified by attacking the gate, not
by reading it. One AC — `agents-infra verify local passes` — does not reproduce at
review time.

## Blocking finding

`agents-infra verify local "$PWD"` exits **1**:

```
installed agent runtime at /Users/alexis/src/local-models/.agents is not usable:
  no generated pi-infra launcher at /Users/alexis/src/local-models/.local/bin/pi-infra
Rerun agents-infra setup against a complete relux-agents-infra source tree.
```

Root cause is dependency drift, not a defect in the delivered files:

| Fact | Evidence |
| --- | --- |
| Verify passed at implementation time | Implementer spawn log: `agents-infra verify local "$PWD"` → `verified local agent runtime` |
| Project runtime installed | `.agents/` files mtime `18:35` |
| Global `agents-infra` binary rebuilt after that | `/Users/alexis/.local/bin/agents-infra` mtime `18:37` |
| The pi-infra requirement is uncommitted WIP upstream | `git diff` in `relux-works/relux-agents-infra` adds `pi-infra` launcher generation + `runtime_receipt.go` verify clause |

The installed runtime predates the launcher requirement, so the delivered state is
now behind the toolchain that judges it. The runtime is reported *not usable* — this
is not a cosmetic gap.

## Remedy is cheap and proven

A clean `agents-infra setup local` into a scratch dir with the current binary:

- `SETUP_EXIT=0`, printed `Installed Pi launcher: .../.local/bin/pi-infra`
- `agents-infra verify local <scratch>` → `VERIFY_EXIT=0`
- scratch `project-config.toml` contains **only** the two `primary_session` blocks —
  no Pi runtime/model profiles were added, so the refresh does not violate the
  orchestrator's deferral of Pi profiles to a follow-up task.

### Rework scope

1. Re-run `agents-infra setup local "$PWD" --codex-primary-model gpt-5.6-sol --codex-primary-reasoning-effort high --codex-yolo-mode=true --claude-primary-model claude-opus-5 --claude-yolo-mode=true`; confirm `.local/bin/pi-infra` exists.
2. `agents-infra verify local "$PWD"` → exit 0 (capture real exit code, not a piped one).
3. Re-confirm both `--print-config` outputs are unperturbed by the refresh.
4. Re-confirm `AGENTS.md`, `.codex/AGENTS.md`, and `.claude/instructions/INSTRUCTIONS.md` still carry the board-first block — setup regenerates these.
5. Re-confirm nothing staged/committed.
6. Update `TASK-260817-3bgn2f_results.md` with the re-run evidence and the drift note.

Caveat for the producer: the upstream source tree is dirty WIP. If it moves again the
refresh may need repeating once the Pi schema lands.

## Verified — gates attacked, not read

### Model admission ceiling (refusal proven, fires before side effects)

| Probe | Result |
| --- | --- |
| codex `gpt-5.6-terra` (out of set) | **REFUSED**: `not admitted by spawn.ceilings.codex.allowed_models`; `No task or run state was changed` |
| claude `claude-sonnet-5` (out of set) | **REFUSED**: `not admitted by spawn.ceilings.claude.allowed_models` |
| codex `gpt-5.6-sol` (in set), bogus task | falls through to task lookup — refusal is model-specific, not blanket |

Out-of-set refusal precedes task resolution, so the gate cannot be reached around by
a valid task ID.

### Reasoning-effort ceiling (clamp proven at the real launch entry point)

Executed on an isolated scratch board copy, not the live board:

| Requested | Resolved |
| --- | --- |
| `gpt-5.6-sol/max` | `-> gpt-5.6-sol/medium` |
| `gpt-5.6-sol/high` | `-> gpt-5.6-sol/medium` |

Proven at the composed production artifact, not the projection: the launched process
carried `model_reasoning_effort="medium"`. Both scratch runs were killed; the real
board, the real tree, and the git index were confirmed untouched afterward.

### Ignore policy (positive and negative)

- 26 representative generated paths ignored, including nested `sub/.temp/x`,
  `nested/checkpoints/c.ckpt`, `sub/dir/model.gguf`.
- Trackability negative checked by **exit code**: `README.md`, `AGENTS.md`,
  `task-board.config.json`, `.agents/.configs/project-config.toml`,
  `.env.example`, `src/`, `scripts/`, `docs/`, `.task-board/` all exit 1 (not ignored).
  `.env.example` is correctly rescued by the `!` negation despite `.env.*`.
- `git add -An` composed set: 123 files, no weights/datasets/caches.

### Primary-session policy

`agents-infra codex --print-config` → `gpt-5.6-sol`, `high`, yolo `true`, one native
danger flag. `agents-infra claude --print-config` → `claude-opus-5`, yolo `true`, one
native danger flag. Both sourced from `.agents/.configs/project-config.toml`.

### Instruction overlay on both surfaces

The board-first block is present in the Codex surfaces (`AGENTS.md`, `.codex/AGENTS.md`,
both generated from `.agents/.instructions/AGENTS.md`) **and** the Claude surface
(`.claude/CLAUDE.md` → symlinked `instructions/INSTRUCTIONS.md`). No profile carries the
rule while another omits it. Live confirmation: this reviewer session loaded the project
overlay from `.claude/instructions/INSTRUCTIONS.md` verbatim.

### Git state

`git status` exit 0; `git ls-files` empty; `git rev-list --all --count` = 0;
`git diff --cached` empty — after all review probing. Nothing staged, nothing committed.
`task-board validate` → `Board is valid. No issues found.`

## Non-blocking observations

1. **Upstream agents-infra bug (not this task's rework).** A clean `setup local` into a
   pristine scratch dir reproduces both artifacts, so neither was introduced here:
   - a literal directory named `$AGENTS_INFRA_SOURCE_DIR` created under `.agents/`
     (unexpanded shell variable);
   - a self-referential symlink `.agents/skills/relux-agents-infra -> .agents`, i.e. an
     infinite `skills/relux-agents-infra/skills/...` cycle.
   Git does not follow symlinks, so the repo is safe, but a committed symlink cycle is a
   footgun for `find -L`, `rsync`, `tar`, and build tools. Belongs in `relux-agents-infra`.
2. **Repo bloat.** The 669 KB implementer spawn log under `.task-board/.resources/` is
   currently trackable. Fine if board evidence is meant to be versioned; worth a
   deliberate decision before the first commit.
