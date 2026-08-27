#!/usr/bin/env bash
set -uo pipefail
cd /Users/alexis/src/relux-works/relux-agents-infra/.temp/STORY-260827-m30k8z/worktree/tools/mlx-swift-runtime-prototype
exec /opt/homebrew/bin/python3 scripts/runtime-benchmark.py \
    --runtime python-mlx-lm --profile qwen-benchmark-python \
    --config /Users/alexis/src/relux-works/relux-agents-infra/.temp/TASK-260827-2v13w8-rework/model-harness.benchmark.toml --port 18031 \
    --model /Users/alexis/src/local-models/Qwen3.8-27B-Uncensored-MLX-8bit \
    --prompts examples/benchmark-prompts.json \
    --out /Users/alexis/src/relux-works/relux-agents-infra/.temp/TASK-260827-2v13w8-rework/records/python-mlx-lm.json --log-dir /Users/alexis/src/relux-works/relux-agents-infra/.temp/TASK-260827-2v13w8-rework/logs \
    --python-bin /Users/alexis/.local/pipx/venvs/mlx-lm-relux/bin/python \
    --declare "deployed with --prompt-cache-size 1 --prompt-cache-bytes 8GB; multi-turn prefix reuse is served from that cache" \
    --declare "loads the model in the generation thread at startup, so /v1/models answers 200 about a second after launch with no weights resident" \
    --declare "/v1/models lists every MLX model in the local Hugging Face cache and appends the configured one last, so data[0].id is not the configured model" \
    --declare "installed revision 9150698 still answers /health with an unconditional 200 (BUG-260827-1jhv2g); the generation-thread liveness tie is not installed"
