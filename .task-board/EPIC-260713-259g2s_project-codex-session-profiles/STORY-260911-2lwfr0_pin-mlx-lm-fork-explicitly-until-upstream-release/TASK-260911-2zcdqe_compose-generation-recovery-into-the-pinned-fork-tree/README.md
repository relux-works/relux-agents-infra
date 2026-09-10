# TASK-260911-2zcdqe: compose-generation-recovery-into-the-pinned-fork-tree

## Description
Found during TASK-260911-1diuhg review: the pinned fork commit 45a472f (branch task/TASK-260830-2hc5r2-bounded-kv in /Users/alexis/src/relux-works/mlx-lm) does NOT contain 9150698 (fix: keep the server generation loop alive when a batched request fails, the BUG-260827-2tul5n recovery that upstream merged as ml-explore/mlx-lm#1791). The running qwen server therefore lacks the generation-loop recovery the goal assumes. Compose both fixes in the fork (merge or cherry-pick 9150698 onto the bounded-KV branch, or rebase onto upstream main now that #1791 is merged), push the fork branch, reinstall mlx-lm-relux non-editable from the new commit, update the pinned_distribution commit in the live model-harness.toml, and restart the shared qwen runtime through the managed lifecycle. Coordinate the restart: it interrupts any session using the local model.

## Scope
relux-works/mlx-lm fork (branch composition + push), pipx reinstall, /Users/alexis/src/.agents/.configs/model-harness.toml pinned_distribution commit, agents-infra README pin note; managed runtime restart via agents-infra lifecycle commands

## Acceptance Criteria
The pinned commit contains both 45a472f and 9150698 by git merge-base; the fork branch is pushed; mlx-lm-relux direct_url.json names the new commit; model-harness doctor qwen-local passes against the live config; the shared runtime was restarted through the managed path and its status reports healthy; the negative control (old commit) fails doctor; README documents the new pin and the retirement condition (upstream PyPI release carrying #1791 and the KV bound).
