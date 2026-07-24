## Status
backlog

## Review
required

## Task Class
code

## Blocked By
- (none)

## Blocks
- (none)

## Checklist
(empty)

## Notes
Product clarification: Keychain ownership is an exploratory sketch, not a fixed requirement. The architecture must first establish whether Claude, Codex, and Qwen permit reliable external credential management. Prefer a provider-specific hybrid or no-go verdict over a forced fit.
Lifecycle clarification: authentication must be method-parameterized. Initial UX may use email plus OTP/confirmation, but provider adapters declare supported methods. logout(provider, exact alias/identity) must invalidate and delete only that profile local credentials; remote revoke and metadata removal are distinct capabilities/actions.

## Precondition Resources
(none)

## Outcome Resources
(none)

## Created
2026-07-20T15:59:12Z

## Last Update
2026-07-20T16:02:12Z
