# BUG-260818-1s1lka review verdict — RUN-260830-78a722

## Verdict

**ACCEPTED** — Change Request `CR-BUG-260818-1s1lka-1` revision 1 is the correct additive rework and may be checkpointed by the commit-owning orchestrator.

The reviewed delta is exactly base `fb7789634d5f423c5ae0563a90dd1ff70f682800` to candidate tree `3fff57c2cbdb144bd8a30c82c46b945ad1ba3b48`. The materialized board patch hashes to `8ebefc659b3ec01bb0834bedfbba4c8fc762c3f0daef5d2f89f7004333ba417d`, matches the Git-generated binary patch byte-for-byte, and changes only:

- `tools/agents-infra/internal/infra/pi_test.go`
- `tools/agents-infra/installed_binary_setup_test.go`

Both files add `llama_api_key=case-sensitive-lookalike` to clean admitted-control fixtures. No production code changes are present in this revision. That is the required correction for the prior EqualFold surviving mutant.

## Gate and production-path review

The production gate remains the case-sensitive exact comparison in `ValidatePiExecutionEnvironment` (`internal/infra/pi_catalog.go`). `RunPi` invokes it before execution identity inspection, managed state creation, listener/runtime initialization, or runtime spawn. Production grep also found the shared runtime/client revalidation sites; the installed bootstrap-global alias and project-local wrapper both enter the real `RunPi` path. No second managed entry point bypassing the gate was found. Unmanaged passthrough and the Windows no-op stub remain outside the supported managed darwin/arm64 lane.

Pristine candidate behavior proves:

- exact non-empty `LLAMA_API_KEY` is refused with only the name in the diagnostic;
- the refusal occurs before managed state/runtime spawn;
- `HF_TOKEN`, Hugging Face cache variables, `LLAMA_API_KEY_SUFFIX`, lowercase `llama_api_key`, `UNRELATED_SERVICE_API_KEY`, and `GGML_METAL_PATH` reach runtime initialization;
- both installed launcher surfaces enforce the exact refusal and admit the same clean controls;
- README and installed-skill operator contracts state the no-ambient-auth and no-value-leak rules.

## Independent gate attacks

All mutants were applied only to disposable copies of the immutable candidate and run with `-count=1` and the pinned Pi fixture present.

| Mutant | Helper | Production `RunPi` | Bootstrap alias | Project-local wrapper | Result |
| --- | ---: | ---: | ---: | ---: | --- |
| Broaden exact `LLAMA_API_KEY` to `strings.EqualFold` | FAIL | FAIL | FAIL | FAIL | Lowercase control is rejected; all required surfaces redden |
| Narrow refusal to empty values only | FAIL | FAIL | FAIL | FAIL | Non-empty secret is admitted; exact refusal/pre-state assertions redden |

The EqualFold failures are attributable to `llama_api_key=case-sensitive-lookalike`: the helper reports that clean input was rejected, production `RunPi` refuses before its runtime marker, and both installed clean-control markers remain absent. The narrow-empty failures are attributable to admission of the non-empty exact key: the helper sees `<nil>`, production reaches the runtime path instead of the pre-state refusal, and both installed `LLAMA_API_KEY` subtests report admission. This closes both the earlier case-broadening gap and the task's narrowing requirement; it is not delete-only evidence.

The earlier independent reviewer evidence additionally covers prefix broadening, gate reordering after state creation, and value-leak mutants. Production code is unchanged by revision 1, so those results remain applicable; this run independently re-exercised the pristine non-leak and pre-state tests.

## Validation

Rerun by this reviewer against an archive of the exact candidate tree:

- focused helper exact-refusal and clean-control tests: exit 0;
- production pre-state refusal plus clean runtime-initialization tests: exit 0, no skips after installing the pinned fixture into the disposable copy;
- real installed bootstrap-global/project-local launcher suite: exit 0;
- operator README/skill contract tests: exit 0;
- `go build ./...`: exit 0;
- `gofmt -l` on both changed files: empty, exit 0;
- `git diff --check <base> <candidate>`: exit 0;
- candidate-copy blob IDs for production gate and both changed test files exactly match the candidate tree after all review runs.

Accepted from tree-bound Change Request publication evidence rather than rerun in this headless review:

- `go test ./... -count=1`: all packages pass (`internal/infra` 226.756s);
- `go vet ./...`: exit 0.

Accepted from already-attached implementation and prior independent review evidence because revision 1 changes test fixtures only and does not change setup/bootstrap/verification code:

- canonical `setup.sh`: exit 0;
- installed `agents-infra verify global`: exit 0;
- isolated `setup local` and `verify local`: exit 0.

Two scratch probes were deliberately excluded from evidence: an initial focused test whose tests passed but whose `tee` destination was wrong, and an initial integrity loop that used zsh's tied `path` variable and therefore lost `PATH`. Both were rerun correctly; neither touched the candidate.

## Conclusion

Revision 1 directly fixes the sole blocking review finding. The exact-name lower and upper bounds now have production and installed-surface negative evidence, admitted controls remain admitted, pristine validation is green, and no acceptance criterion is outstanding. Reviewer archetype supplies no `commit_ack`; acceptance must park the element at `to-review` for orchestrator checkpoint/integration.
