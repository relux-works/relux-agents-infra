# TASK-260720-bk6owf: design-extensible-auth-method-lifecycle

## Description
Define a provider-neutral authentication method and credential lifecycle contract. Separate provider, account identity, auth method, and local profile alias. Initial desired method is email plus OTP/confirmation where provider-native flows support it; future methods include browser OAuth, device code, API key, enterprise SSO, and subscription-specific plans. Specify enrollment, activation, status, logout, revocation, and removal semantics without implementation.

## Scope
Architecture model and CLI semantics only. Do not assume every provider supports email/OTP, remote revocation, or externally managed credentials.

## Acceptance Criteria
1. Data model separates provider, profile alias, account identity, auth method, and credential handle.
2. AuthMethod adapter defines start, continue/confirm, status, refresh capability, logout, revoke capability, and local-delete behavior.
3. Email and OTP are treated as method inputs/claims, never as universal provider primitives or credential secrets.
4. logout(provider, identity-or-alias) removes or invalidates only the selected local credential profile; server-side revoke is reported separately when supported.
5. remove is explicitly destructive and deletes remaining local metadata/state after logout policy is resolved.
6. CLI grammar remains extensible to browser OAuth, device code, API key, SSO, and Coding Plan without changing the provider/profile model.
7. Ambiguous identity matches fail safely and require provider plus alias or an exact unique identity.
