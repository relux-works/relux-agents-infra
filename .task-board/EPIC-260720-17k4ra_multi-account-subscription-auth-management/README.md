# EPIC-260720-17k4ra: multi-account-subscription-auth-management

## Description
Investigate and, only if provider contracts permit it, design multi-account subscription authentication management through agents-infra. Desired UX: select provider plus identity/email, complete native authorization, then switch among multiple Claude, Codex, and eventually Qwen accounts. Direct agents-infra ownership of per-profile credentials in macOS Keychain is a candidate hypothesis, not a committed solution.

## Scope
Feasibility research and architecture decision for provider auth boundaries, native profile isolation, optional Keychain custody, account selection, concurrent sessions, refresh/revocation semantics, diagnostics and future adapters. No implementation in this phase.

## Acceptance Criteria
1. Research proves what auth state each provider allows an external manager to isolate, store, inject, refresh, revoke, and switch; unsupported behavior is not inferred.
2. The model separates provider, account identity, local alias, and parameterized auth method; email-plus-OTP is an initial candidate method, not a universal assumption.
3. At least three custody models are compared: agents-infra-owned Keychain credentials, agents-infra-managed opaque native profiles, and provider-native switching/delegation.
4. logout targeted by provider plus exact alias/identity invalidates and deletes only that profile local credentials; server-side revoke and metadata removal have explicit separate semantics.
5. The decision may be full-go, provider-specific hybrid, or no-go; it must not force one design across incompatible providers.
6. Any chosen model supports multiple independently named accounts and prevents cross-account leakage in concurrent launches.
7. Raw credentials and OTP values never enter repository files, board resources, normal config, command-line arguments, logs, or shell history.
8. Claude and Codex conclusions are evidence-backed; Qwen remains explicitly provisional until verified.
9. No implementation begins before feasibility findings and ADR receive review.
