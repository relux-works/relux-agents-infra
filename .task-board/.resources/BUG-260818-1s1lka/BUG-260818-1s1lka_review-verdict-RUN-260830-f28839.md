# BUG-260818-1s1lka review verdict — RUN-260830-f28839

## Verdict

**CHANGES REQUESTED — workflow re-review only; route to `analysis`.** No source change is requested. The candidate satisfies the task acceptance criteria, but this reviewer run cannot legally record acceptance because it was created before Change Request revision 1 existed and was not handed that revision.

## Operational finding

- Reviewer run `RUN-260830-f28839` was created at `2026-08-30T01:25:53.520281Z`.
- `BUG-260818-1s1lka_change-request_rev1.patch` was persisted at `2026-08-30T01:28:33.217759Z`.
- The run prompt contains no `Change Request Under Review` section; run status/manifest output contains no CR binding.
- `task-board worktree status STORY-260817-1on8ex` reports `BUG-260818-1s1lka rev 1 ready` with `repository_delta=present` and two changed paths.
- The Change Request contract permits `accept_cr` only from the reviewer run handed the exact revision. Calling it here would be an unauthorized self-minted acceptance, so it was not attempted.

Required next action: return the task to its review routing path and spawn a new reviewer after CR rev 1 is ready, so its prompt/manifest is bound to `CR-BUG-260818-1s1lka-1` revision 1. That reviewer may reuse this evidence but must independently read the candidate and attach its own verdict before `accept_cr`.

## Technical assessment

No implementation finding remains. CR rev 1 is limited to the requested lower-case lookalike controls in:

- `tools/agents-infra/internal/infra/pi_test.go`
- `tools/agents-infra/installed_binary_setup_test.go`

The controls preserve `HF_TOKEN`, Hugging Face cache variables, `LLAMA_API_KEY_SUFFIX`, unrelated names, and exact case-sensitive policy while closing the previous `EqualFold` broadening gap.

## Independent gate attacks

- Restored candidate focused helper, production `RunPi`, and shared-runtime production-entry tests: exit 0.
- Restored installed bootstrap-global and project-local launcher test, verbose with all subtests executed: exit 0.
- Narrow-to-empty mutant (`LLAMA_API_KEY` rejected only for an empty value): helper and production `RunPi` exit 1; both installed launcher LLAMA subtests exit 1 because the forbidden key reaches runtime.
- `strings.EqualFold` broadening mutant: helper clean control and production `RunPi` clean lifecycle exit 1 on `llama_api_key`; both installed clean controls exit 1 before their runtime marker.
- Disposable direct `runtime runtime-launch` probe with a valid authorization frame and exact `LLAMA_API_KEY`: restored production exit 0; deleting the pre-`sharedRuntimeExecve` validation call makes the probe exit 1 with an empty refusal and reachable target path.

These attacks cover narrowing, over-broadening, the production call site, installed launcher surfaces, value non-leakage, pre-state refusal, and the independently callable shared-runtime path.

## Validation

- Operator README/SKILL docs tests: exit 0.
- `go vet ./...`: exit 0.
- `go build ./...`: exit 0.
- `gofmt -l` on both CR paths: empty output, exit 0.
- `git diff --check HEAD`: exit 0.
- Tree-bound CR publication validation: `go test ./... -count=1` and `go vet ./...`, exit 0.
- Producer rework evidence additionally records `go test -p 1 ./... -count=1`, build, vet, formatting, and diff checks at exit 0.
- Earlier task evidence records canonical `setup.sh`, global verify, isolated local setup, and isolated local verify at exit 0. They were not rerun for this additive test-only CR by the read-only reviewer.

One combined search/lint probe had a shell quoting error and was not counted; every affected gate was rerun separately with independently observed exit 0. Details are in `.temp/BUG-260818-1s1lka-review/tool-failure-RUN-260830-f28839.log`.

## Scope integrity

The reviewer changed no repository file. All mutants and the LLAMA direct-entry probe live only in a detached disposable worktree under `.temp/`. The repository logbook was not edited because the reviewer role is read-only; this task-scoped outcome is the persistent anomaly record.
