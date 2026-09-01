# TASK-260707-xx9bdv — rework: refresh accepted CR onto current trunk

## Why this refresh happened

`task-board worktree integrate STORY-260830-3uerxs --cr TASK-260707-xx9bdv --revision 2`
refused with `integration_base_moved` on 2026-09-02: revision 2 was accepted
against base `71230fc187e6b11eb9aea5520616b20967e223e3` (2026-08-30), but trunk
had since advanced through `.configs/codex-config.toml`
(model bumped to `gpt-5.6-sol`, context-window/auto-compact overrides changed,
new trusted-project entries added), a file the candidate's cumulative Story
snapshot also touched. No one had reviewed that combination.

## Base refresh

- Old base: `71230fc187e6b11eb9aea5520616b20967e223e3`
- New base: `5a78932b449fec5aa07b4bb7a13c54ea97784d53` (current `origin/main`,
  verified equal via `git rev-parse HEAD` / `git rev-parse origin/main` and
  `git rev-list --left-right --count HEAD...origin/main` → `0 0`)
- No rebase/merge was needed: the Story worktree branch
  (`task-board/story/STORY-260830-3uerxs`) was already parked exactly at
  current trunk when this rework started.

## `.configs/codex-config.toml` overlap — resolution

This task's own scope never intentionally touches `.configs/codex-config.toml`.
The prior accepted revision 2's Change Request carried a diff against that
file only because a CR for this Story is built from the cumulative Story
snapshot, which picks up sibling tasks' already-landed commits (Codex
model/context-window changes from unrelated Story work) relative to an
increasingly stale pinned base. Those sibling changes are already on trunk
verbatim as of `5a78932`.

Resolution: keep trunk's `.configs/codex-config.toml` unchanged (verified no
diff between the worktree's copy and trunk's) and re-express nothing on top of
it, because this task never had a genuine intent for that file. There is no
conflicting intent to reconcile — the previous staleness was a base-pin
artifact, not a real design disagreement.

## This task's actual repository delta: still empty

The x-platform-airdrop / Tap2Cash workflow-bullet removal from
`.instructions/INSTRUCTIONS_WORKFLOW.md` was already committed to trunk in
`d1c8d7d5649c37df394d3401101a9650491b4893` (2026-07-06), well before this task
existed. This task's own deliverable has always been the preservation artifact
and the installed-runtime refresh, not a source-tree edit. That remains true
on the refreshed base — confirmed no repository delta is required this round.

## Re-verification on the refreshed base

- `.instructions/` (source, this worktree) and `~/.agents/.instructions`
  (installed): re-synced via `agents-infra setup global --source-dir <this
  worktree>`; `diff -rq .instructions ~/.agents/.instructions` → identical,
  no output.
- `agents-infra verify global` → exit 0 (`verified global agent runtime:
  /Users/alexis/.agents`).
- `agents-infra doctor global` → exit 0, all managed-link/config fields report
  healthy (`git_free=true`, `claude_linked=true`, `codex_rendered=true`,
  `codex_config_linked=true`, `helpers_linked=true`, `infra_skill_link=true`).
- Negative search (case-insensitive, separator-flexible) for
  `tap[ -]?2?[ -]?cash|swipe[ -]?2?[ -]?cash|x-platform-airdrop|x[
  -]?platform[ -]?airdrop|xpairdrop` across all four production surfaces:
  - source `.instructions` → ripgrep exit 1 (no match)
  - installed `~/.agents/.instructions` → ripgrep exit 1 (no match)
  - rendered `~/.codex/AGENTS.md` → ripgrep exit 1 (no match)
  - rendered `~/.claude/CLAUDE.md` → ripgrep exit 1 (no match)
  (exit 1 = clean no-match; any exit >1 would mean a read failure, not an
  absence — none occurred.)
- `INSTRUCTIONS_WORKFLOW.md`'s "Research & Knowledge Persistence" section was
  read in full: it is continuous and complete after the earlier bullet
  removal (generic artifact location, persistence rationale, child-agent
  persistence, task/worklog linkage all present).

## Validation commands (this rework, run from `tools/agents-infra`)

| Command | Exit | Notes |
| --- | ---: | --- |
| `go vet ./...` | 0 | clean |
| `go test ./... -count=1` | 0 | root `159.106s`, `internal/attachments` `2.149s`, `internal/infra` `265.146s`, `internal/modelharness` `15.155s`; `cmd/model-harness` has no test files (expected) |
| `go run . setup global --source-dir <worktree>` | 0 | resynced `~/.agents`, re-rendered `~/.codex/AGENTS.md`, refreshed launchers |
| `go run . verify global` | 0 | |
| `go run . doctor global` | 0 | |

## Outcome

The accepted design is unchanged. This rework only re-pins the Change
Request to current trunk (`5a78932`) and re-confirms every acceptance
criterion and validation command against that exact base, so a fresh CR
revision can be published without a stale-base conflict.
