# Owner requirements

Define a configuration-driven target identity with separate vendor, agent environment/harness, model, reasoning, and optional endpoint/profile coordinates.

Required resolved targets:

- Qwen vendor; Pi environment; local OpenAI-compatible model URL/profile; Qwen3.8 27B MLX 8-bit model.
- OpenAI vendor; Codex environment; hosted gpt-5.6-sol model; high reasoning.
- Anthropic vendor; Claude Code environment; hosted claude-opus-5 model; high reasoning.

Required installed entrypoints:

- openai-infra
- anthropic-infra
- qwen-infra

Each entrypoint must resolve the environment and model from agents-infra project configuration rather than hardcoding the harness/model pair. Preserve existing agents.codex, agents.claude, and agents.pi configurations through explicit backward-compatible precedence or migration behavior.