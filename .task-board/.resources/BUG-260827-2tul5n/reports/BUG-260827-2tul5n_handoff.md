# mlx-lm fork and resilience handoff

## Fork and local source

- Fork: https://github.com/relux-works/mlx-lm
- Stable checkout: `/Users/alexis/src/relux-works/mlx-lm`
- Runtime branch: `runtime/generation-recovery`
- Runtime commit: `91506981056172f937e7bdca4ab0d3b7459c7fab`
- Health branch: `fix/generation-health-readiness`
- Health delta is intentionally uncommitted because upstream `AGENTS.md` forbids agents from writing commit messages, pushing, creating pull requests, or writing pull-request prose.

The runtime commit is a current-main cherry-pick of upstream PR #1513 commit `58d38e4f57ba4f3d2495a3ed9b8a0da7ba129960`. It preserves Rajan Sharma as the author. Current upstream main also contains the merged ArraysCache fix from PR #1632.

## Local runtime selection

- Isolated pipx environment: `/Users/alexis/.local/pipx/venvs/mlx-lm-relux`
- Executable: `/Users/alexis/.local/bin/mlx_lm-relux.server`
- PEP 610 source: `file:///Users/alexis/src/relux-works/mlx-lm/.git`
- Exact installed revision: `91506981056172f937e7bdca4ab0d3b7459c7fab`
- Root model-harness config now selects `mlx_lm-relux.server` for the next Qwen start.
- The currently running Qwen process still uses the old `mlx_lm-qwenfix.server`; it was left untouched.

## Validation

- Main-branch health regression failed as expected: one missing property error and one `200 != 503` assertion.
- Health regression after the patch: 2 tests passed.
- Runtime recovery injection: passed; the in-flight request received `RuntimeError("boom")`, the generation thread survived, and a later request completed.
- Mixed logits-processor regression: passed.
- Full `tests.test_server` plus `tests.test_generate`: 58 tests passed in 37.577 seconds.
- Black and isort hooks passed for changed health files.
- Installed runtime source contains both generation-loop recovery and the ArraysCache advance fix.
- `model-harness render` resolves the next launch to `/Users/alexis/.local/bin/mlx_lm-relux.server` with the configured 50k stress profile and restart supervision.

## Human-only upstream publication

The repository policy requires the human contributor to understand and author the commit and pull-request prose. From the stable checkout:

```bash
cd /Users/alexis/src/relux-works/mlx-lm
git push origin runtime/generation-recovery

git add mlx_lm/server.py tests/test_server.py
git commit
git push -u origin fix/generation-health-readiness
gh pr create --repo ml-explore/mlx-lm --base main --head relux-works:fix/generation-health-readiness --web
```

The health change is complementary to open PR #1513: it makes `/health` return HTTP 503 with `{"status": "unavailable"}` if the generation thread is not alive. It does not duplicate #1513's recovery code.
