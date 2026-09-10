# STORY-260911-2lwfr0: pin-mlx-lm-fork-explicitly-until-upstream-release

## Description
2026-09-11 goal audit finding: the pipx venv mlx-lm-relux is an *editable* install that follows the local fork checkout HEAD (45a472f, branch task/TASK-260830-2hc5r2-bounded-kv, 3 commits ahead of fork main) while its pipx metadata claims commit 9150698. The goal clause retain the pinned mlx-lm fork is therefore not actually pinned. Upstream ml-explore/mlx-lm#1791 (generation-thread recovery) merged 2026-09-05 but PyPI latest is still 0.31.3 without it, so the fork stays required for now.

## Scope
pipx install spec for mlx-lm-relux, the agents-infra local-model target/setup that references it, README operator note; no runtime behaviour change

## Acceptance Criteria
mlx-lm-relux is installed non-editable from an explicit fork commit or tag whose tree carries bounded KV (--max-kv-size), the generation-loop recovery (9150698) and the effective-config report (45a472f); pipx metadata and the running server agree on that commit; setup/verify refuses an editable or drifted install; a retirement condition is written: switch to upstream once a PyPI release carries #1791 and the KV bound. Negative test: an editable install or a mismatched commit fails verify local.
