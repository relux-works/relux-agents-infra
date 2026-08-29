## Status
done

## Review
light

## Task Class
code

## Estimate
estimated(fibonacci(1))

## Blocked By
- (none)

## Blocks
- (none)

## Checklist
(empty)

## Notes
Implemented source and installed runtime update from 1M/900K to 272K/245K. Rationale: OpenAI Docs list GPT-5.6 Sol long-context pricing as starting when prompts exceed 272K input tokens; no official compact-limit recommendation was found, so 245000 keeps the same approximate 90% headroom pattern as the prior 900000/1000000 config while staying below the threshold. Changed .configs/codex-config.toml root keys to model_context_window=272000 and model_auto_compact_token_limit=245000 before any TOML table; GPT-5.6 Sol remains selected. README documents the values and headroom. Updated setup tests to assert the new defaults. Preserved pre-existing installed local state by adding the agent-session-manager trust entry and tui.status_line settings to source before syncing. Synced with agents-infra setup global --source-dir /Users/iv/Developer/IV/relux-agents-infra. Validation: python tomllib source+installed ok; focused go test ./internal/infra ok; go test ./... -count=1 ok; go vet ./... ok; git diff --check ok; agents-infra doctor global ok. task-board validate logs 480 pre-existing MISSING_RESOURCE_PAYLOAD records already tracked as board debt, unrelated to this config change.

## Precondition Resources
(none)

## Outcome Resources
(none)

## Created
2026-08-29T23:56:34Z

## Last Update
2026-08-30T00:01:58Z

## Assigned To
codex
