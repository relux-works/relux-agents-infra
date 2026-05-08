# STORY-260508-1ajnbe: codex-claude-instruction-rendering

## Description
Добавить в agents-infra setup renderer, который из source modules в .agents/.instructions строит разные runtime-представления: Codex получает materialized AGENTS.md без include-директив, Claude получает Claude-совместимую раскладку.

## Scope
Встроить agent-specific instruction rendering в существующие agents-infra команды setup global, setup local и refresh-links. Source modules остаются в .agents/.instructions; runtime outputs пишутся отдельно: Codex получает materialized AGENTS.md/config в .codex, Claude получает Claude-compatible CLAUDE.md/instructions в .claude. Не полагаться на то, что Codex или Claude напрямую интерпретируют .agents как общий формат.

## Acceptance Criteria
- agents-infra setup global генерирует корректные ~/.codex/AGENTS.md и ~/.claude runtime instructions из source modules.
- agents-infra setup local PROJECT_DIR генерирует корректные PROJECT/.codex и PROJECT/.claude runtime instructions из project-local source modules.
- agents-infra refresh-links переиспользует тот же renderer и не откатывает output к symlink на .agents/.instructions/AGENTS.md.
- Codex output не содержит raw @~/.agents include lines и проходит проверку через codex debug prompt-input.
- Claude output остается совместим с подтвержденным Claude contract.
- doctor показывает состояние instruction rendering отдельно от простых symlink checks.
