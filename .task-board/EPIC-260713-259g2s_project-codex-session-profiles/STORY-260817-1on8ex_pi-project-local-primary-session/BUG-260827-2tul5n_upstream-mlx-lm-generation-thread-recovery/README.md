# BUG-260827-2tul5n: upstream-mlx-lm-generation-thread-recovery

## Description
mlx_lm.server leaves its HTTP listener alive and future completions hanging after an uncaught BatchGenerator exception kills the generation thread.

## Scope
Create and maintain the relux-works/mlx-lm fork, implement generation-thread exception propagation and continued service on a current-upstream branch, validate focused and upstream tests, open a pull request to ml-explore/mlx-lm, and repin the local Qwen pipx runtime to the relux-works fork commit.

## Acceptance Criteria
The relux-works fork contains the exact fix commit; an injected generation exception fails the affected request without wedging the HTTP service; a subsequent request can complete; focused upstream tests pass; a public upstream PR links the reproduction and validation; local qwen config resolves to an isolated pipx environment installed from the relux-works fork commit.
