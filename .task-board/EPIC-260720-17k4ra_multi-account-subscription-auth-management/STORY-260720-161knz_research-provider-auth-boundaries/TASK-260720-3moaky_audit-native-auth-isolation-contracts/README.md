# TASK-260720-3moaky: audit-native-auth-isolation-contracts

## Description
Collect sanitized evidence for Claude Code and Codex CLI enrollment, credential persistence, secure-storage behavior, token refresh/rotation, logout, and supported injection boundaries. Determine how agents-infra can take Keychain custody after native login without logging or exposing secrets.

## Scope
Read-only CLI/source/official-doc research and empty-profile smoke tests. Do not inspect any real credential payload or existing Keychain item.

## Acceptance Criteria
Research identifies provider-native login flows, credential artifacts/backends, supported secure injection or isolated runtime mechanisms, refresh/write-back semantics, concurrency risks, and blockers to agents-infra Keychain ownership. Evidence contains no secrets.
