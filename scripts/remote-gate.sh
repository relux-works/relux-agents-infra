#!/bin/sh
# Landing gate for task-board Story workspaces: runs this repository's CI
# workflow on GitHub (hosted ubuntu + macOS matrix) instead of on the
# operator host, whose per-call budget the full Go suite cannot meet while
# producers and reviewers share it.
# Adapted from relux-works/curator scripts/remote-gate.sh;
# the push trigger for gate/** branches lives in .github/workflows/ci.yml.
# Landing gate that runs the repository's CI workflow on the GitHub runner
# instead of on this host.
#
# Run from inside a Story worktree (or any checkout). It snapshots the working
# tree exactly as it is (tracked, modified and untracked-but-not-ignored files)
# into a detached commit on top of HEAD, pushes that commit to a throwaway
# gate/<name>/<stamp> branch, waits for the CI workflow run that the push
# triggers, prints the failed step logs when it is red, deletes the branch and
# exits with the run's status. The worktree, its index and its HEAD are never
# touched.
#
# Requirements on the caller's host: git with push access to origin, gh
# authenticated for the repository, SSH_AUTH_SOCK reachable when origin is ssh.
set -eu
export GODEBUG="${GODEBUG:-netdns=go}"

WORKFLOW="${REMOTE_GATE_WORKFLOW:-ci.yml}"
POLL="${REMOTE_GATE_POLL_SECONDS:-30}"
DISPATCH_WAIT="${REMOTE_GATE_DISPATCH_WAIT_SECONDS:-300}"

root=$(git rev-parse --show-toplevel)
cd "$root"

# .temp/STORY-x/worktree -> STORY-x; otherwise the checkout directory name.
name=$(basename "$(dirname "$root")")
case "$name" in STORY-*|TASK-*|BUG-*|EPIC-*) ;; *) name=$(basename "$root") ;; esac
name=$(printf '%s' "$name" | tr -c 'A-Za-z0-9._-' '-')

head=$(git rev-parse HEAD)
tmpindex=$(mktemp "${TMPDIR:-/tmp}/remote-gate-index.XXXXXX")
trap 'rm -f "$tmpindex"' EXIT
GIT_INDEX_FILE="$tmpindex" git read-tree "$head"
GIT_INDEX_FILE="$tmpindex" git add -A .
tree=$(GIT_INDEX_FILE="$tmpindex" git write-tree)
if [ "$tree" = "$(git rev-parse "$head^{tree}")" ]; then
  commit=$head
else
  commit=$(printf 'remote gate snapshot of %s\n\nworking tree of %s on top of %s\n' "$name" "$root" "$head" \
    | GIT_COMMITTER_NAME="remote-gate" GIT_COMMITTER_EMAIL="remote-gate@localhost" \
      GIT_AUTHOR_NAME="remote-gate" GIT_AUTHOR_EMAIL="remote-gate@localhost" \
      git -c commit.gpgsign=false commit-tree "$tree" -p "$head")
fi

ATTEMPTS="${REMOTE_GATE_ATTEMPTS:-3}"
attempt=1
while :; do
  branch="gate/$name/$(date -u +%y%m%d-%H%M%S)-$$-$attempt"
  echo "remote gate: attempt $attempt/$ATTEMPTS: pushing $commit as $branch"
  git push -q origin "$commit:refs/heads/$branch"
  cleanup() {
    git push -q origin --delete "$branch" 2>/dev/null || true
    rm -f "$tmpindex"
  }
  trap cleanup EXIT INT TERM

  run_id=""
  waited=0
  while [ -z "$run_id" ]; do
    run_id=$(gh run list --workflow "$WORKFLOW" --branch "$branch" --limit 1 --json databaseId --jq '.[0].databaseId // empty' 2>/dev/null || true)
    [ -n "$run_id" ] && break
    if [ "$waited" -ge "$DISPATCH_WAIT" ]; then
      echo "remote gate: no $WORKFLOW run appeared for $branch within ${DISPATCH_WAIT}s" >&2
      exit 2
    fi
    sleep 5; waited=$((waited + 5))
  done
  echo "remote gate: run $run_id ($(gh run view "$run_id" --json url --jq .url 2>/dev/null || echo url-unavailable))"

  status=""
  while :; do
    status=$(gh run view "$run_id" --json status,conclusion --jq '.status + "/" + (.conclusion // "")' 2>/dev/null || echo "unknown/")
    case "$status" in
      completed/*) break ;;
    esac
    sleep "$POLL"
  done
  conclusion=${status#completed/}
  echo "remote gate: run $run_id finished: $conclusion"
  gh run view "$run_id" --json jobs --jq '.jobs[] | "  " + .name + ": " + (.conclusion // .status) + "  " + ((.steps // []) | map(select(.conclusion != "success" and .conclusion != "skipped" and .conclusion != null) | .name) | join(", "))' 2>/dev/null || true
  if [ "$conclusion" = "success" ]; then
    exit 0
  fi
  # A cancelled run carries no verdict about the tree: GitHub cancels runs for
  # queue reasons (a runner loss, a concurrency rule, an operator click). Push
  # the same snapshot again under a fresh branch rather than fail the gate.
  if [ "$conclusion" = "cancelled" ] && [ "$attempt" -lt "$ATTEMPTS" ]; then
    git push -q origin --delete "$branch" 2>/dev/null || true
    attempt=$((attempt + 1))
    sleep 20
    continue
  fi
  echo "remote gate: failed step logs (tail):"
  gh run view "$run_id" --log-failed 2>/dev/null | tail -n 400 || true
  exit 1
done
