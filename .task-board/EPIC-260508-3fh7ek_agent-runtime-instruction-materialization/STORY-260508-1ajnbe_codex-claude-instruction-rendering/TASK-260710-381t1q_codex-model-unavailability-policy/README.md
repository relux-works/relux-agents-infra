# Codex model unavailability policy

## Description
Prevent model-capacity prompts from becoming human operational decisions. Prefer waiting and retrying the requested top-tier model, then choose the best autonomous fallback path after repeated failures.

## Scope
Audit the installed Codex CLI runtime chooser and supported config schema; configure a non-interactive wait/retry preference if the runtime exposes one; add source-managed global instructions for autonomous retries and fallback; update renderer/setup tests and generated artifacts; reinstall and verify runtime drift.

## Acceptance Criteria
Codex runtime chooser is suppressed or pre-answered to wait when a supported config exists, otherwise the platform limitation is documented; global instructions forbid asking the human whether to wait or use a weaker model; agents retry the preferred model several times before selecting the best alternate path; setup and tests regenerate both Codex and Claude outputs without drift.
