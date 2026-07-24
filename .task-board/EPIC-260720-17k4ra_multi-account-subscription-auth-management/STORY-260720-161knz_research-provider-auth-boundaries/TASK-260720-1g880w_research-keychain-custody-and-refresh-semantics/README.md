# TASK-260720-1g880w: research-keychain-custody-and-refresh-semantics

## Description
Evaluate whether agents-infra can safely own provider credential records in macOS Keychain after native login. Treat this as a hypothesis. Determine whether credential capture, opaque storage, injection, refresh write-back, revocation, and concurrency are supported for each provider; recommend no-go where external custody violates provider contracts or cannot be made reliable.

## Scope
Feasibility research only using documentation, provider source/contracts, empty synthetic profiles, and synthetic Keychain payloads. Never access real credentials.

## Acceptance Criteria
Research produces a per-provider feasibility matrix for external Keychain custody, lists hard blockers and unknowns, compares custody against opaque native-profile isolation, and defines proof gates required before any real credential experiment.
