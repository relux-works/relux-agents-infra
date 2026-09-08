# TASK-260901-1ixi9y: install-direct-provider-yolo-aliases

## Description
Install source-managed openai-dange and anthropic-dange executable aliases beside the existing canonical provider entrypoints so operators can launch each provider directly in YOLO mode with normal argument forwarding.

## Scope
Relux agents-infra setup, verification, launcher bytes, tests, README, and relux-agents-infra skill documentation. Preserve openai-infra and anthropic-infra behavior; the new names are convenience aliases only.

## Acceptance Criteria
1. Global and local setup install executable regular files named openai-dange and anthropic-dange. 2. openai-dange delegates to the OpenAI canonical entrypoint with exactly one danger/YOLO selection and forwards caller arguments byte-for-byte. 3. anthropic-dange does the same for Anthropic. 4. verify detects missing, drifted, symlinked, or non-executable aliases and setup repairs them. 5. Focused and full Go tests pass, and README/SKILL document the commands.
