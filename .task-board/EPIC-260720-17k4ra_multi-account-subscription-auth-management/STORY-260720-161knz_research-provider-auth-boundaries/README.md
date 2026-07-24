# STORY-260720-161knz: research-provider-auth-boundaries

## Description
Research provider-native login, credential storage boundaries, supported state-directory overrides, logout behavior, and concurrent-session implications for Claude Code and Codex CLI. Define the evidence needed before a Qwen adapter can be accepted. No implementation.

## Scope
Read-only local CLI inspection and official documentation research. Do not inspect credential contents or export secrets.

## Acceptance Criteria
1. Claude and Codex native auth entry points and isolation mechanisms are documented.
2. Unsupported assumptions are clearly separated from verified behavior.
3. Qwen is represented as an explicit future research gap, not guessed behavior.
4. Findings are attached as a board resource and feed the ADR.
