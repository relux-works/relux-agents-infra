# TASK-260825-rtmcsw — Developer Rework Evidence

## Scope

The production call path remains `main -> run -> runModelCheck ->
infra.RunModelCheck -> infra.RunPi`. The rework preserves the existing managed
Pi/runtime lifecycle and addresses review findings F1–F4 without adding another
launcher.

- Provider fixtures now persist the actual OpenAI-compatible request body and
  prove the caller's exact prompt reached the provider boundary.
- Cleanup attestation tests pin `pending` and `failed` as unconfirmed and verify
  they select production exit code 1; `confirmed`,
  `confirmed_after_sigkill`, and `not_started` are pinned separately.
- Production-entrypoint tests cover deadline-range, existing-evidence,
  raw-event-only overwrite, and non-managed-target refusals.
- A plan refusal no longer fabricates an event-stream lifecycle error when no
  event was observed, and `duration_ms` now reports the measured value including
  a legitimate zero.
- README documents the stable exit-code contract.

## Expected-red evidence

Every command below was run directly with `-count=1`; non-zero results are
expected failures and are not reported as passing gates.

| Attack | Command scope | Exit | Result |
| --- | --- | ---: | --- |
| Pre-fix honesty assertion | Full production model-check suite | 1 | New non-managed-target case rejected the invented event-stream error. |
| Prompt replaced with fixed literal | Happy-path production entrypoint in throwaway copy | 1 | Exact provider request lacked the caller prompt. |
| Cleanup predicate forced true | Pending/failed cleanup cases in throwaway copy | 1 | Both false attestations were detected. |
| Maximum deadline widened to 3h | Out-of-range production refusal in throwaway copy | 1 | A 2h run was admitted and the test rejected it. |
| Raw overwrite protections narrowed | Raw-event-only production refusal in throwaway copy | 1 | The prior JSONL was appended and the test rejected it. |
| Codex target admitted | Non-managed production refusal in throwaway copy | 1 | Execution passed the intended plan gate and the test rejected the wrong reason. |

One earlier prompt-mutant attempt exited 0 only because the production test was
skipped after the throwaway copy lost its relative pinned-Pi path. It was not
counted. After a task-local read-only symlink restored fixture discovery, the
same mutant ran and exited 1.

## Green validation

| Command | Exit | Evidence |
| --- | ---: | --- |
| `go test . -run TestModelCheckProductionEntrypoint -count=1 -v` | 0 | 12/12 production-entrypoint subtests passed; 11.23s suite body. |
| `go test ./internal/infra -run TestModelCheckCleanupAttestationRefusesUnconfirmedStates -count=1 -v` | 0 | 5/5 lifecycle states passed. |
| `go test . -count=1` | 0 | Main package passed in 77.699s. |
| `go test ./internal/... -count=1` | 0 | Latest run after test self-review: `internal/attachments` 1.421s; `internal/infra` 116.356s. |
| `go vet ./...` | 0 | No diagnostics. |
| `go build ./...` | 0 | Native build passed. |
| `GOOS=windows GOARCH=amd64 go build ./...` | 0 | Windows compatibility stub compiled. |
| `git diff --check` | 0 | No whitespace errors before final evidence assembly. |

`go list ./...` reports exactly the main, `internal/attachments`, and
`internal/infra` packages, so the two bounded full-suite commands cover the
module. Tests use the pinned Pi asset and a local Python fixture; no external
model was downloaded.
