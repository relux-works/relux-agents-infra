# BUG-260901-2iidgq: primary-session-compose-validates-unselected-pi-profile

## Description
In /Users/alexis/src/bsim, agents-infra compose --mode primary-session --agent codex fails with invalid_project_configuration because an existing Pi profile lacks the newly required publisher field. The selected provider is Codex, so strict Pi profile validation crosses the provider boundary and blocks openai-board even though no Pi runtime is being selected or launched.

## Scope
Make primary-session configuration loading and validation provider-local. Codex and Claude composition must ignore incomplete or unavailable Pi-only profile details that they do not select. Pi direct/canonical/standalone paths must retain strict profile validation and actionable errors.

## Acceptance Criteria
1. The exact bsim-shaped config with a Codex primary session and a Pi profile missing publisher successfully composes a Codex primary-session plan. 2. The same shape successfully composes Claude when Claude policy is selected. 3. Selecting Pi/qwen with that profile still fails closed and names agents.pi.profiles.<name>.publisher. 4. Malformed shared/top-level or selected-provider fields still fail; the fix does not silently accept an invalid selected profile. 5. Negative production-callsite tests prove validation cannot widen back across providers, full Go tests pass, and docs state provider-local validation.
