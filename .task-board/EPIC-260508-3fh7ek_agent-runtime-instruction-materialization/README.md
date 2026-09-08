# EPIC-260508-3fh7ek: agent-runtime-instruction-materialization

## Description
Развести исходные модульные инструкции и runtime-представления для Claude/Codex: .agents как source of truth, .claude как Claude-совместимая раскладка, .codex как Codex-совместимая плоская AGENTS.md раскладка без @include.

## Scope
(define epic scope)

## Acceptance Criteria
Every supported agent runtime receives its instructions through one materialization path, verified on the installed artifact and not on source bytes, with the entrypoint and include chain intact for each runtime. Adding a runtime requires no runtime-specific branch in shared composition code. Global and project instruction layers compose in a defined precedence, and a missing or malformed layer fails closed with an error naming the layer.
