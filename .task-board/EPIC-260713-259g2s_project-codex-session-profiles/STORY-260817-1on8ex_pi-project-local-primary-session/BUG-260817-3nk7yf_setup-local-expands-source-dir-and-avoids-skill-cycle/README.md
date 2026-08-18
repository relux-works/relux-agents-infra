# BUG-260817-3nk7yf: setup-local-expands-source-dir-and-avoids-skill-cycle

## Description
Fix reusable setup artifacts reproduced by the local-models reviewer: a literal $AGENTS_INFRA_SOURCE_DIR directory and a self-referential relux-agents-infra skill symlink.

## Scope
Correct source-dir expansion/materialization and skill-link generation in relux-agents-infra; add production setup-local regressions; reinstall and verify a pristine scratch project and local-models. Do not patch generated runtime files directly.

## Acceptance Criteria
Fresh setup local creates no literal variable-named directory; no self-referential or escaping skill symlink is emitted; recursive-safe inspection succeeds; setup/verify and focused/full tests pass; local-models is refreshed from source with those artifacts absent.
