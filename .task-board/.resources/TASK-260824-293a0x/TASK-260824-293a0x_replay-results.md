# TASK-260824-293a0x replay evidence

## Outcome

Replayed the accepted `CR-TASK-260824-1qm60c-2` revision 2 patch into the
dedicated `STORY-260824-9exx3n` worktree without unrelated edits.

- Accepted resource: `TASK-260824-1qm60c_change-request_rev2.patch`
- Resource SHA-256: `1c097fabe0b37b93c8696840e63db8d50faf8867dc21a63cff70f44546ae372b`
- Accepted CR state: `accepted`
- Accepted base OID: `cf21665dde35274cc14e66e26a93574e0c18c15c`
- Accepted candidate tree OID: `a1f084fb0def43889cc3a517bc00072033c81817`
- Dedicated Story base commit: `266bc112b9e0cafb91b46a63a66f575460bb96e3`
- Dedicated Story candidate tree OID: `936edb22b562389051dcc8c23dd6850920680b80`

## Scope and identity

The accepted base and the dedicated Story base have identical content at all
eight reviewed paths (`git diff --quiet`, exit 0). Reconstructing the accepted
base plus the downloaded patch through an alternate Git index produced exactly
`a1f084fb0def43889cc3a517bc00072033c81817` (exit 0). The current
eight-path `git diff --binary` is byte-identical to the accepted patch (`cmp`,
exit 0).

Changed paths:

1. `.configs/codex-config.toml`
2. `LOGBOOK.md`
3. `README.md`
4. `SKILL.md`
5. `tools/agents-infra/internal/infra/codex_config.go`
6. `tools/agents-infra/internal/infra/infra.go`
7. `tools/agents-infra/internal/infra/infra_test.go`
8. `tools/agents-infra/setup_test.go`

## Validation

| Command | Exit | Evidence |
| --- | ---: | --- |
| `git apply --check <accepted-rev2.patch>` | 0 | Patch applies cleanly to the dedicated Story base. |
| `go test ./... -count=1` | 0 | Full configured Go suite passed; main 54.538s, attachments 3.104s, infra 76.931s. |
| Focused migration/refusal `go test ./internal/infra ... -count=1` | 0 | Three named production-path tests passed in 2.312s. |
| `go vet ./...` | 0 | Configured vet gate passed. |
| `go build ./...` | 0 | All Go packages build. |
| Empty `gofmt -l` assertion on four changed Go files | 0 | Formatting is clean. |
| `git diff --check` | 0 | No whitespace errors. |

## Negative evidence and production call site

Production path: `infra.Setup` -> `syncRepo` -> `syncManagedCodexConfig`.

- `TestSetupGlobalMigratesManagedCodexConfigPreservingUserState` rejects the
  withdrawn fast tier/profile while preserving user trust, notice state, and a
  custom profile through the real global setup path.
- `TestSetupLocalPreservesExistingNativeAgentConfigsOnResync` attacks the old
  local repeat-sync bypass through the real local setup path.
- `TestSetupGlobalRejectsMalformedExistingCodexConfigWithoutReplacingIt`
  proves a malformed read is not treated as absence and its bytes are not
  replaced.

The prior independent acceptance is recorded in
`TASK-260824-1qm60c_review-verdict-rev2.md`. The replay adds no new behavioral
delta; the accepted `LOGBOOK.md` entries preserve the root cause, migration
decision, and production-path evidence.
