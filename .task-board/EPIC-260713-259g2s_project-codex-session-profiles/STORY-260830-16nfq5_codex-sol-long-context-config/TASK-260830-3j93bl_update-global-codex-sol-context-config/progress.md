## Status
to-review

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
Implemented in source and installed runtime. OpenAI Docs checked: config reference documents ~/.codex/config.toml and the model_context_window/model_auto_compact_token_limit keys; GPT-5.6 Sol docs list model id gpt-5.6-sol, 1,050,000 context window, and >272K long-context pricing threshold. Changed .configs/codex-config.toml root keys to model=gpt-5.6-sol, model_context_window=1000000, model_auto_compact_token_limit=900000 before any TOML table; restored pre-existing installed trust entries /Users/iv and /Users/iv/Developer/relux-tunnel in source so setup global preserves them. Synced with agents-infra setup global --source-dir /Users/iv/Developer/IV/relux-agents-infra. Validation: python tomllib source+installed ok; go test ./internal/infra focused ok; go test ./... -count=1 ok; go vet ./... ok; git diff --check ok; agents-infra doctor global ok. task-board validate logs 480 pre-existing MISSING_RESOURCE_PAYLOAD issues already tracked as board debt, unrelated to this config change.

## Precondition Resources
(none)

## Outcome Resources
(none)

## Created
2026-08-29T23:39:49Z

## Last Update
2026-08-29T23:46:28Z

## Assigned To
codex
