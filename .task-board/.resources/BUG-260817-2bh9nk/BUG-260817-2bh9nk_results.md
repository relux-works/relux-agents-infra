# BUG-260817-2bh9nk — developer handoff

## Outcome

- Managed `infra.RunPi` refuses exact inherited `HF_ENDPOINT` and `MODEL_ENDPOINT` names before managed state or runtime spawn.
- Refusal diagnostics include only the environment name; tests verify origin values do not appear.
- `HF_HOME`, `HF_TOKEN`, `HUGGING_FACE_HUB_TOKEN`, and lowercase lookalikes remain admitted controls. Tokens and cache-location variables are intentionally outside this model-origin rule pending separate effect analysis.
- README and the source-managed `relux-agents-infra` skill document the operator policy.
- The bootstrap-owned global binary was rebuilt with `./setup.sh`; both the real global `pi-infra` alias and a real generated local wrapper refused both names before a runtime marker could be created.

## Production evidence

- Production call site: `tools/agents-infra/main.go:423` passes `os.Environ()` to `infra.RunPi`.
- Shared pre-spawn gate: `tools/agents-infra/internal/infra/pi_launch_posix.go:94` calls `ValidatePiExecutionEnvironment` before state creation and runtime spawn.
- Exact-name enforcement: `tools/agents-infra/internal/infra/pi_catalog.go:295`.
- Unit and production-entry regressions: `tools/agents-infra/internal/infra/pi_test.go`.
- Installed global/local launcher regression: `tools/agents-infra/installed_binary_setup_test.go:633`.

## Validation

| Command / gate | Exit | Result |
| --- | ---: | --- |
| Pre-fix targeted environment and production-entry tests | 1 | Expected red: both model-origin names were admitted and reached runtime readiness. |
| Pre-fix docs regression test | 1 | Expected red: README and skill lacked the policy. |
| HF-only narrowing mutant, `go test -count=1` | 1 | Expected red on `MODEL_ENDPOINT` in unit and production `RunPi`. |
| MODEL-only narrowing mutant, `go test -count=1` | 1 | Expected red on `HF_ENDPOINT` in unit and production `RunPi`. |
| Targeted environment/production tests | 0 | Passed. |
| Operator docs regression tests | 0 | Passed. |
| Installed global/local launcher integration test | 0 | Passed. |
| `go test ./... -count=1` | 0 | Passed all packages. |
| `go test -race ./internal/infra -count=1 -run 'Test.*Pi'` | 0 | Passed focused Pi race gate. |
| `gofmt -l` empty check | 0 | Clean. |
| `go vet ./...` | 0 | Clean. |
| `go build ./...` | 0 | Passed. |
| `git diff --check` | 0 | Passed. |
| `task-board validate` | 0 | Passed. |
| `./setup.sh` | 0 | Rebuilt and installed bootstrap-owned global launcher. |
| `agents-infra verify global` after bootstrap | 0 | Passed. |
| Disposable `agents-infra setup local` / `verify local` | 0 / 0 | Passed. |
| Real global/local `HF_ENDPOINT` and `MODEL_ENDPOINT` probes | 1 each | Expected refusal; exact-name diagnostics, no values, no runtime marker. |

Logs are under `.temp/BUG-260817-2bh9nk/`. Existing unrelated dirty worktree changes were preserved; nothing was staged or committed.
