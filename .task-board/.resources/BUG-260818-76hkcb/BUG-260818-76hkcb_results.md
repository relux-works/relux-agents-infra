# BUG-260818-76hkcb implementation evidence

## Outcome

- `ValidatePiExecutionEnvironment`, called by production `RunPi` before `ResolvePiStatePaths`, `CreatePiStateTree`, and runtime spawn, now refuses exact case-sensitive `GGML_BACKEND_PATH` without including its value in diagnostics.
- Existing `DYLD_*`, `LD_*`, `NODE_*`, `BUN_*`, `LLAMA_ARG_*`, `HF_ENDPOINT`, and `MODEL_ENDPOINT` behavior remains intact.
- Lowercase `ggml_backend_path`, `GGML_BACKEND_PATH_SUFFIX`, and unrelated `GGML_METAL_PATH` remain admitted; a clean `GGML_METAL_PATH` control reaches runtime backend initialization.
- Bootstrap-global and setup-generated project-local `pi-infra` launchers both exercise the production gate and prove no runtime marker is created for the denied exact name.
- README and skill operator contracts record llama.cpp build 10470's inherited-value `dlopen()` effect and explicitly reject speculative broad `GGML_*` denial.

## Changed scope

- `tools/agents-infra/internal/infra/pi_catalog.go`
- `tools/agents-infra/internal/infra/pi_test.go`
- `tools/agents-infra/installed_binary_setup_test.go`
- `tools/agents-infra/pi_operator_docs_test.go`
- `README.md`
- `SKILL.md`
- `LOGBOOK.md`

The checkout already contained the broader uncommitted Pi rollout and unrelated concurrent board/source changes. This task made only the exact gate, its focused controls, operator wording, and logbook entry listed above; no pre-existing changes were reset or staged.

## Validation evidence

| Gate | Exit | Evidence |
| --- | ---: | --- |
| Focused production environment/RunPi suite after restoration | 0 | `go-test-production-gate-restored-01.log` |
| Installed global/local launcher + docs suite after restoration | 0 | `go-test-installed-docs-restored-01.log` |
| Full uncached Go suite: `go test -count=1 ./...` | 0 | `go-test-all-01.log` |
| Go vet: `go vet ./...` | 0 | `go-vet-all-01.log` |
| Go build: `go build ./...` | 0 | `go-build-all-01.log` |
| Go formatting check | 0 | `gofmt-check-01.log` |
| Git diff whitespace check | 0 | `git-diff-check-01.log` |
| Bootstrap: `./setup.sh` | 0 | `bootstrap-global-01.log` |
| Installed global verification | 0 | `verify-global-01.log` |
| Task-scoped local setup | 0 | `setup-local-01.log` |
| Installed local verification | 0 | `verify-local-01.log` |
| Board structural validation | 0 | `task-board-validate-01.log` |

## Negative and narrowing evidence

The narrowing mutant changed only the policy member from exact `GGML_BACKEND_PATH` to exact `GGML_BACKEND_PATH_V2`; it did not delete the environment gate. Both real entrypoint suites were rerun with `-count=1` before restoring the source byte-for-byte.

| Mutant gate | Exit | Required red observation |
| --- | ---: | --- |
| Production validator + `RunPi` | 1 | Exact variable admitted; production reached runtime instead of the pre-state refusal |
| Bootstrap-global + project-local installed launchers | 1 | Both launchers admitted exact variable and reached runtime |

Logs: `narrowing-mutant-source-01.log`, `narrowing-mutant-production-red-01.log`, and `narrowing-mutant-installed-red-01.log`.

Two earlier installed-control attempts exited 1 while the fixture was being corrected: the first used a non-authoritative fake Pi and the second lacked the canonical cache root. They are retained as `go-test-installed-launchers-01.log`, `go-test-installed-launchers-02.log`, and `go-test-installed-control-diagnostic-01.log`; the corrected installed suite subsequently exited 0.
