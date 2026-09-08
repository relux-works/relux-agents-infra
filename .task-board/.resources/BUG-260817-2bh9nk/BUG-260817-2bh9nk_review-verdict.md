# BUG-260817-2bh9nk Review Verdict

## Verdict

Accepted for Change Request `CR-BUG-260817-2bh9nk-2`, revision 2. No review findings remain.

The candidate adds `ValidatePiExecutionEnvironment(environ)` in `RunSharedRuntimeLauncher` immediately before the sole Darwin `sharedRuntimeExecve` call. The production CLI dispatch at `tools/agents-infra/main.go:628` supplies the actual process environment to this boundary. Exact `HF_ENDPOINT` and `MODEL_ENDPOINT` names are therefore refused before the configured llama.cpp runtime execs; refusal text contains the name but not its value. The clean control still execs the configured target. Tokens, cache variables, lowercase lookalikes, and unrelated names remain outside this exact-name policy as required.

Operator policy and regression gates already exist in `README.md`, `SKILL.md`, `pi_operator_docs_test.go`, and the installed global/local launcher test. Revision 2 closes the independently callable `runtime runtime-launch` bypass found in the prior review and adds both an internal process-entry test and a real built-CLI test.

## Candidate integrity

- Base OID: `5c9b4e4f7a88e1eb937b80851af522e4fa4b066f`
- Candidate tree OID: `bea562ff014770b3a723fd48fea9f157f8479ae8`
- Review patch SHA-256: `b850ec8261db19d962e0b89861e3d0cf123422bde3428459c8792c75fe8c6014`, exactly matching the handed revision
- Reviewed delta: four paths, 214 insertions and one deletion; `git diff --check` passed

## Independent attack evidence

- Restored source: `TestSharedRuntimeLauncherRefusesModelOriginEnvironmentAtProductionEntry`, the model-origin helper negative, and the clean `RunPi` control passed uncached.
- Restored source: `TestProductionRuntimeLaunchRefusesModelOriginEnvironment`, installed global/local launcher probes, and README/SKILL documentation gates passed uncached.
- HF-only narrowing mutant: expected red in the built production binary test because `MODEL_ENDPOINT` reached the target; exit 1.
- MODEL-only narrowing mutant: expected red in the built production binary test because `HF_ENDPOINT` reached the target; exit 1.
- Both mutants also produced the corresponding expected red through the internal process entry.
- A first outer-only Go overlay appeared to survive because `buildInstalledBinary` performs a nested `go build`; this was not treated as product evidence. Re-running with inherited `GOFLAGS=-overlay=...` mutated the composed production binary and killed both mutants.
- Full independent `go test ./... -count=1` passed. Root package: `197.973s`; `internal/infra`: `353.161s`; attachments and modelharness also passed.
- Independent `go vet ./...`, `go build ./...`, `gofmt -d` on changed Go files, and diff hygiene all passed.
- Producer tree-bound validation was also inspected: full uncached module suite and `go vet ./...` passed for revision 2.

The reviewer supplies no `commit_ack`; the commit-owning orchestrator owns checkpoint/integration after acceptance.
