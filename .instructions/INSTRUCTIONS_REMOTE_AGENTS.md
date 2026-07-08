# Remote Agent Workers

Use this pattern when a remote machine has an authenticated agent runtime
(for example Claude Code) and the local host has the project checkout that must
remain the source of truth.

## Core Model

- Treat the remote machine as a disposable worker, not as the owner of the
  project.
- Run the remote agent in an isolated remote copy or worktree.
- Bring changes back to the local checkout as a patch, commit, or reviewed file
  transfer.
- Verify locally before accepting the work.
- Keep the workflow host-agnostic: do not hardcode a specific host alias, user,
  install path, shell profile, board setup, or sync tool in reusable
  instructions.

## Variables

Use explicit variables in notes, scripts, and task resources:

```bash
REMOTE_SSH="user@host-or-ssh-alias"
REMOTE_CLAUDE="claude"                  # or an absolute path discovered on the remote
LOCAL_PROJECT="$PWD"
RUN_ID="$(date +%Y%m%d-%H%M%S)-task"
LOCAL_RUN_DIR=".temp/remote-agent/$RUN_ID"
REMOTE_RUN_ROOT="/tmp/remote-agent-runs"
REMOTE_PROJECT="$REMOTE_RUN_ROOT/$RUN_ID/project"
REMOTE_PROMPT="$REMOTE_RUN_ROOT/$RUN_ID/prompt.md"
LOCAL_PATCH="$LOCAL_RUN_DIR/remote.patch"
```

The exact transfer mechanism is intentionally interchangeable. `git clone`,
`git worktree`, `rsync`, `scp`, `tar` over SSH, shared volumes, or a board/server
artifact flow are all acceptable when they preserve the same contract:

1. Remote agent gets a task-scoped copy of the needed project files.
2. Remote agent writes only inside that copy.
3. Local host receives a reviewable change artifact.

## Readiness Check

Before delegating real work, verify the remote access and runtime without
printing credentials or private config:

```bash
ssh -o BatchMode=yes -o ConnectTimeout=10 "$REMOTE_SSH" 'hostname; whoami; pwd'
ssh -o BatchMode=yes "$REMOTE_SSH" "command -v '$REMOTE_CLAUDE' || true"
ssh -o BatchMode=yes "$REMOTE_SSH" "'$REMOTE_CLAUDE' --version"
```

If non-interactive SSH does not load the same `PATH` as an interactive shell,
use an absolute remote binary path or prefix `PATH` inside the remote command.
Do not inspect or print OAuth tokens, API keys, cookie stores, keychain exports,
or files such as agent credential JSON.

Run a one-line non-interactive smoke before asking for edits:

```bash
ssh "$REMOTE_SSH" "cd /tmp && '$REMOTE_CLAUDE' -p 'Reply with remote-agent-ok.'"
```

## Prepare A Remote Copy

Start from a clean local state:

```bash
git status --short
git diff --check
mkdir -p "$LOCAL_RUN_DIR"
```

Prefer a task-scoped local worktree or archive so the remote agent cannot see
unrelated untracked files:

```bash
git worktree add "$LOCAL_RUN_DIR/source" HEAD
```

Then transfer the project copy using any suitable tool. This tar-over-SSH shape
is portable and does not require the remote host to have repository access:

```bash
ssh "$REMOTE_SSH" "mkdir -p '$REMOTE_RUN_ROOT/$RUN_ID'"
tar \
  --exclude='.git' \
  --exclude='.temp' \
  --exclude='DerivedData' \
  --exclude='node_modules' \
  -C "$LOCAL_RUN_DIR/source" \
  -czf "$LOCAL_RUN_DIR/source.tgz" .
scp "$LOCAL_RUN_DIR/source.tgz" "$REMOTE_SSH:$REMOTE_RUN_ROOT/$RUN_ID/source.tgz"
ssh "$REMOTE_SSH" "
  mkdir -p '$REMOTE_PROJECT' &&
  tar -xzf '$REMOTE_RUN_ROOT/$RUN_ID/source.tgz' -C '$REMOTE_PROJECT' &&
  cd '$REMOTE_PROJECT' &&
  git init -q &&
  git add -A &&
  git commit -q -m baseline
"
```

For repositories the remote can access directly, `git clone` plus checkout of
the exact branch/commit is fine. The important part is preserving a baseline so
the remote can produce a clean diff.

## Prompt Contract

Write the prompt to a file and transfer it as an artifact. Include:

- task goal and acceptance criteria
- exact files or directories in scope
- commands the remote agent should run
- "do not read or modify files outside this project copy"
- expected handoff: summary, tests run, known failures, and no commit unless
  explicitly requested

Example:

```bash
cat > "$LOCAL_RUN_DIR/prompt.md" <<'PROMPT'
You are working in a disposable remote copy of a project.
Do not read or modify files outside this directory.

Task:
- Implement the requested change with minimal scope.
- Preserve existing behavior unless the task says otherwise.
- Run the relevant tests or build checks.
- Leave changes uncommitted.

Handoff:
- Summarize files changed.
- List validation commands and results.
- Mention blockers or skipped checks explicitly.
PROMPT

scp "$LOCAL_RUN_DIR/prompt.md" "$REMOTE_SSH:$REMOTE_PROMPT"
```

## Spawn Claude Remotely

Use non-interactive Claude for remote worker runs:

```bash
ssh "$REMOTE_SSH" "
  cd '$REMOTE_PROJECT' &&
  '$REMOTE_CLAUDE' -p --permission-mode bypassPermissions \"\$(cat '$REMOTE_PROMPT')\"
"
```

Use `--permission-mode bypassPermissions` only for disposable remote copies where
the agent is intentionally allowed to edit and run local validation commands.
For less isolated environments, prefer a narrower permission mode or
`--allowedTools` set that permits only the required read/edit/test tools.

If the remote Claude binary is not on non-interactive `PATH`, set it explicitly:

```bash
REMOTE_CLAUDE="$HOME/.local/bin/claude"
```

or prefix the remote command:

```bash
PATH="$HOME/.local/bin:$PATH" claude -p ...
```

## Bring Changes Back

Never trust only the remote agent's summary. Export and inspect the actual diff:

```bash
ssh "$REMOTE_SSH" "
  cd '$REMOTE_PROJECT' &&
  git status --short &&
  git diff --binary > '$REMOTE_RUN_ROOT/$RUN_ID/remote.patch'
"

scp "$REMOTE_SSH:$REMOTE_RUN_ROOT/$RUN_ID/remote.patch" "$LOCAL_PATCH"
git apply --check "$LOCAL_PATCH"
git apply "$LOCAL_PATCH"
```

If the patch does not apply cleanly, inspect it manually and either rerun the
remote agent with a tighter prompt or apply the useful parts by hand.

## Local Verification

After applying remote changes locally:

- inspect `git diff`
- run the local build/test/lint commands appropriate for the project
- verify no unrelated files changed
- document remote run path, commands, and logs in the normal task notes or
  `.temp/` scratch notes

The local host remains responsible for final acceptance, commit, and push.

## Cleanup

Clean up only after the patch and logs are captured:

```bash
ssh "$REMOTE_SSH" "rm -rf '$REMOTE_RUN_ROOT/$RUN_ID'"
git worktree remove "$LOCAL_RUN_DIR/source"
```

Keep local `.temp/remote-agent/$RUN_ID/` logs until the task is complete.

## Failure Modes

- `claude: command not found`: non-interactive SSH PATH differs from the
  interactive shell. Discover the remote binary path and use it explicitly.
- auth prompt or subscription failure: stop and ask the human to fix remote
  Claude auth on that machine. Do not scrape credentials.
- patch is empty: inspect remote status and the prompt; the agent may have only
  answered without editing.
- patch conflicts locally: rebase/refresh the remote copy or apply manually.
- remote tests pass but local tests fail: trust local verification and reopen or
  rerun with the local failure context.
- task needs secrets or private services: pass only task-scoped credentials
  through an approved secure channel; never copy broad local dotfiles.
