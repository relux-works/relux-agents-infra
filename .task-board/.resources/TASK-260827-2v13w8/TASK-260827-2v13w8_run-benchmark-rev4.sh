#!/usr/bin/env bash
# Revision 4: ONE invocation of ONE binary launches both runtimes, drives every
# scenario, measures, seals and judges. There is no argument here through which
# a measurement can be supplied.
set -uo pipefail
cd /Users/alexis/src/relux-works/relux-agents-infra/.temp/STORY-260827-m30k8z/worktree/tools/mlx-swift-runtime-prototype
exec ./DerivedData/Build/Products/Release/mlx-swift-runtime-prototype benchmark-run \
    --config /Users/alexis/src/relux-works/relux-agents-infra/.temp/TASK-260827-2v13w8-rev4/model-harness.benchmark.toml \
    --model /Users/alexis/src/local-models/Qwen3.8-27B-Uncensored-MLX-8bit \
    --prompts /Users/alexis/src/relux-works/relux-agents-infra/.temp/STORY-260827-m30k8z/worktree/tools/mlx-swift-runtime-prototype/examples/benchmark-prompts.json \
    --thresholds /Users/alexis/src/relux-works/relux-agents-infra/.temp/STORY-260827-m30k8z/worktree/tools/mlx-swift-runtime-prototype/examples/benchmark-thresholds.json \
    --session /Users/alexis/src/relux-works/relux-agents-infra/.temp/TASK-260827-2v13w8-rev4/session \
    --harness /Users/alexis/.local/bin/model-harness \
    --baseline-runtime python-mlx-lm --baseline-profile qwen-benchmark-python \
    --candidate-runtime mlx-swift --candidate-profile qwen-benchmark-swift \
    --port 18031 \
    --python-bin /Users/alexis/.local/pipx/venvs/mlx-lm-relux/bin/python \
    --candidate-binary /Users/alexis/src/relux-works/relux-agents-infra/.temp/STORY-260827-m30k8z/worktree/tools/mlx-swift-runtime-prototype/DerivedData/Build/Products/Release/mlx-swift-runtime-prototype \
    --baseline-declare "prompt cache enabled as deployed: --prompt-cache-size 1 --prompt-cache-bytes 8GB, the incumbent's installed configuration, kept rather than tuned away" \
    --baseline-declare "/health is a static 200 in the deployed revision 9150698; the generation-thread liveness fix is in the fork checkout but not installed" \
    --baseline-declare "/v1/models answers 200 about a second after launch with no weights resident; the model loads on first completion" \
    --candidate-declare "no prompt cache across requests: every request builds a fresh KV cache, so a shared multi-turn prefix is re-prefilled every turn" \
    --candidate-declare "one generation at a time: GenerationEngine is an actor, so requests serialize even though the profile allows concurrent leases" \
    --candidate-declare "readiness is gated on a resident model: /v1/models answers 503 until the weights are loaded, and advertises only the configured model" \
    --candidate-declare "served by MLXLLM.LLMModelFactory (text-only); the vision tower is not loaded and the HTTP contract refuses image and audio content"
