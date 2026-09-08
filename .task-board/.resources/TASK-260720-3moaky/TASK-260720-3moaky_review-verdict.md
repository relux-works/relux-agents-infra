# Review verdict: accepted

Task: `TASK-260720-3moaky`

Change Request: `CR-TASK-260720-3moaky-1`, revision 1

Reviewer run: `RUN-260830-117271`

## Scope reviewed

- Candidate contains exactly three repository paths: `.research/260830_native-auth-isolation-contracts.md`, `LOGBOOK.md`, and `README.md`.
- The attached research resource is byte-identical to the candidate research file.
- No production code is changed; the repository delta is a research artifact plus its README and logbook indexes.

## Verdict

Accepted. The audit satisfies the task acceptance criteria and preserves the no-secret/no-live-auth-mutation boundary. It distinguishes vendor-supported contracts, exact-version Codex source behavior, internal/forbidden interfaces, observations, and unknowns. The resulting hybrid recommendation is supported by the evidence: vendor-native human OAuth remains vendor-owned; agents-infra custody is limited to supported injected credentials or workload identity.

## Evidence and adversarial checks

- Re-fetched the official OpenAI authentication, configuration, access-token, and workload-identity Markdown pages and the official Anthropic Claude Code authentication, WIF, and `ant` authentication pages. The cited enrollment, persistence, credential precedence, refresh ownership, logout, and injection claims reproduced.
- Verified the vendored official Codex checkout is exactly tag `rust-v0.150.1`, commit `90854393966b21e9ebfd21b122334eb09a20c93d`, matching installed `codex-cli 0.150.1`; installed Claude is `2.1.248`.
- Attacked the plaintext-fallback boundary: the production `AutoAuthStorage` falls back to file storage on both Keyring load and save errors, and existing source tests cover the failure shapes. This supports the audit's refusal to use `auto` for a Keychain-only policy.
- Attacked per-home isolation: production direct-Keyring and encrypted-secrets backends both derive their account identity from canonical `CODEX_HOME`; callers construct these backends through the real auth-storage factory. This remains correctly labelled current-source behavior requiring an upgrade gate, not a public guarantee.
- Attacked concurrent refresh: the refresh semaphore belongs to one `AuthManager` instance/process; the guarded reload can adopt a prior writer, but there is no cross-process lock around the authority call. No exact simultaneous two-process refresh proof exists. The audit correctly reports this as unknown/risky instead of inferring safety.
- Checked production reachability: CLI and App Server paths call `logout_with_revoke` and `refresh_token`; the internal `chatgptAuthTokens` protocol is explicitly marked unstable/internal-only and is correctly rejected as an architecture dependency.
- Verified the producer's empty-profile commands used `env -i` with task-scoped `HOME`/`CODEX_HOME` or `CLAUDE_CONFIG_DIR`. Codex printed only `Not logged in` and created no `auth.json`. Claude `--bare` printed only its unauthenticated message; installed help explicitly says bare mode never reads OAuth or Keychain. Its created state/transcript files are mode `0600` and contain no credential artifact.
- Re-ran a bounded secret-shape scan over all changed deliverables: no API-key, JWT, bearer-token, private-key, or GitHub-token shape matched. No real credential payload or existing Keychain item was read.
- Re-ran `go test ./... -count=1` on the candidate tree: exit 0; packages completed successfully, with the longest package at 304.854s.
- `git diff --check`: exit 0. `task-board validate`: exit 0.

## Deliberate limits

- No live login completion, logout, revoke, refresh, rotation, authenticated status query, or existing Keychain-item inspection was performed.
- Claude native OAuth refresh/write-back/logout-revocation and both providers' exact simultaneous native-refresh outcomes remain unknown for the reasons recorded in the audit.

## Board routing result

The reviewer spawn was not handed a Change Request revision (`revision 0` in its immutable run binding), although `CR-TASK-260720-3moaky-1` revision 1 was ready by review time. `accept_cr(... revision=1 ...)` therefore failed closed with `change_request_acceptance_unauthorized`; it was not retried or bypassed. Because the assigned prompt contained no `Change Request Under Review` section, the non-CR accepted branch was used and `set_status(... status=done)` succeeded without `commit_ack`. At handoff, the task is `done`, the board validates, and CR revision 1 remains `ready`; the orchestrator should reconcile that pre-existing CR/worktree record before Story integration.
