#!/usr/bin/env bash
set -uo pipefail
cd /Users/alexis/src/relux-works/relux-agents-infra/.temp/STORY-260827-m30k8z/worktree/tools/mlx-swift-runtime-prototype
exec /opt/homebrew/bin/python3 scripts/runtime-benchmark.py \
    --runtime mlx-swift --profile qwen-benchmark-swift \
    --config /Users/alexis/src/relux-works/relux-agents-infra/.temp/TASK-260827-2v13w8-rev3/model-harness.benchmark.toml --port 18031 \
    --model /Users/alexis/src/local-models/Qwen3.8-27B-Uncensored-MLX-8bit \
    --prompts examples/benchmark-prompts.json \
    --out /Users/alexis/src/relux-works/relux-agents-infra/.temp/TASK-260827-2v13w8-rev3/records/mlx-swift.json --log-dir /Users/alexis/src/relux-works/relux-agents-infra/.temp/TASK-260827-2v13w8-rev3/logs \
    --swift-binary /Users/alexis/src/relux-works/relux-agents-infra/.temp/STORY-260827-m30k8z/worktree/tools/mlx-swift-runtime-prototype/DerivedData/Build/Products/Release/mlx-swift-runtime-prototype \
    --attest-binary /Users/alexis/src/relux-works/relux-agents-infra/.temp/STORY-260827-m30k8z/worktree/tools/mlx-swift-runtime-prototype/DerivedData/Build/Products/Release/mlx-swift-runtime-prototype \
    --attest-dir /Users/alexis/src/relux-works/relux-agents-infra/.temp/TASK-260827-2v13w8-rev3/attest \
    --declare "no prompt cache across requests: every request builds a fresh KV cache, so a shared multi-turn prefix is re-prefilled every turn" \
    --declare "one generation at a time: GenerationEngine is an actor, so requests serialize even though the Pi profile allows max_leases = 8" \
    --declare "readiness is gated on a resident model: /v1/models answers 503 until the weights are loaded, and advertises only the configured model" \
    --declare "served by MLXLLM.LLMModelFactory (text-only); the vision tower is not loaded and the HTTP contract refuses image and audio content"
