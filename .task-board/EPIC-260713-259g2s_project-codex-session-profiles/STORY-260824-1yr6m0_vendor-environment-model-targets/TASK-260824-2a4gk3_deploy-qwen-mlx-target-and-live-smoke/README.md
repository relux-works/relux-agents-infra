# TASK-260824-2a4gk3: deploy-qwen-mlx-target-and-live-smoke

## Description
Deploy contract Section 2 required targets (R2-R3, R5-R6) into casual-talks and establish Section 5 provenance plus Section 7 end-to-end evidence for the local Qwen/Pi path.

## Scope
Project config for OpenAI/Codex gpt-5.6-sol high, Anthropic/Claude Code claude-opus-5 high, and Qwen/Pi Qwen3.8 27B MLX 8-bit via the existing managed Pi profile; source setup/install; alias print/compose diagnostics; real Pi text and safe task-scoped filesystem tool round trips.

## Acceptance Criteria
All three required aliases resolve the exact Section 2 tuples with target and effective provenance. The Qwen profile model/reasoning/profile-provider/endpoint assertions and Section 5 qualified-model/endpoint invariants match, runtime binds loopback only, readiness passes, one real text response and one safe task-scoped filesystem tool call succeed, and cleanup proves no orphan listener/runtime. No secret or arbitrary environment value is persisted.
