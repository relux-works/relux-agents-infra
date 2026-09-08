# TASK-260830-ter72z — revision 4 developer handoff

## Outcome

- Owning repository: `skill-agents-management`.
- Signed commit: `37dc5c0b2abd82cd369042daab9087dfc845e880` (`Refuse missing inference-engine facts`). Raw commit object contains an SSH signature; human author and committer remain `alexis <alexis@relux.works>`.
- Published branch: `codex/TASK-260830-ter72z-inference-engine-plugin`.
- Pull request: <https://github.com/relux-works/skill-agents-management/pull/8>, body updated to revision 4, remote head matches the signed commit.
- Release contract: signed `v0.4.4` tag only after independent review accepts this exact head; published tags are immutable.
- `relux-agents-infra` Story worktree tracked delta: empty, as required. `ExecutionOwner` remains the literal `agents-infra`; no OS process, SSH, polling, readiness, or supervision execution moved.

## Contract fix and audit

- `pkg/inferenceengine.ValidateCandidateValue` now routes every structured fact through one strict object decoder.
- Every non-`omitempty` JSON field must be present and non-null before semantic validation. Errors from the public entry point name the fact and missing field.
- Explicit legitimate zeros remain valid: speculative `active=false` and restart `max_restarts=0`.
- Audit: 10 struct shapes, 11 JSON fact schemas, 35 distinct required JSON tags, 38 fact-field occurrences. Load and unload share the three-field transition shape.
- Only the two reviewer-demonstrated omissions previously returned nil: `speculative-decoding.active` and `profile-restart-supervision-policy.max_restarts`. The other 36 omissions already failed downstream semantic validation but did not identify the missing evidence.
- Required `null` is also refused by name; Go otherwise decodes scalar null into the same zero values.

## Public-entry attack evidence

- Pre-fix reproducer through `inferenceengine.ValidateCandidateValue`: exit 0, both attacks returned canonical zero-filled JSON with `error=<nil>`.
- Tests-first omission matrix: expected-red exit 1. All 38 omissions failed the new naming assertion; the two zero-valued fields were admitted with nil.
- Post-fix reproducer through the same public entry: exit 0 as a probe; both calls returned empty canonical output plus `ErrObservationMalformed`, naming `active` or `max_restarts`.
- Exhaustive external-package test removes and nulls all 38 required occurrences.
- Two paired tests admit explicit zero and refuse the otherwise-identical omitted field.
- Mutation harness exit 0: 5/5 compile-clean narrowed mutants killed. The two new mutants exempt only `active` or only `max_restarts`; each named `go test` exits 1. No delete-only mutant is used.

## Validation — real exit codes

- `gofmt -l pkg internal tools`: exit 0, no output.
- `git diff --check`: exit 0, no output.
- `go test -mod=mod ./pkg/inferenceengine -count=1`: exit 0, 15 named top-level tests.
- `go test -mod=mod -race ./pkg/inferenceengine -count=1`: exit 0.
- `go test -mod=mod -cover ./pkg/inferenceengine -count=1`: exit 0, 94.5% statements.
- `python3 .scripts/verify-inference-engine-contract.py`: exit 0; 5/5 mutants killed, each internal named test exit 1 as required.
- `make vet`: exit 0.
- `make build BIN=.temp/TASK-260830-ter72z/agents-management`: exit 0.
- `env -u TASK_BOARD_DIR go test -mod=mod ./... -count=1`: exit 0, 29 package lines, 0 FAIL.
- `make regress`: exit 0.
- `env -u TASK_BOARD_DIR go test -mod=mod -race ./... -count=1`: exit 0, 29 package lines, 0 FAIL.
- `env -u TASK_BOARD_DIR go test -mod=mod -cover ./... -count=1`: exit 0, 29 package lines, inferenceengine 94.5%.
- `skill-creator/scripts/validate-skill.sh` against staged install name `agents-management`: exit 0, 220 body lines.

## Operational anomalies

- First `git commit -S` attempt: exit 128 because Git selected GPG and no GPG secret key exists. No commit object was created. Re-run used the already-loaded SSH signing identity through invocation-local `gpg.format=ssh`: exit 0.
- First pre-push ancestry command: exit 1 because unquoted `HEAD^` was expanded by zsh as a glob. No push ran. Re-run quoted `HEAD^`, proved remote head equaled the candidate parent, then push exited 0.

## Files changed

Nine owning-repository paths, 216 insertions / 25 deletions:

- `.scripts/verify-inference-engine-contract.py`
- `LOGBOOK.md`
- `README.md`
- `SKILL.md`
- `docs/architecture.md`
- `docs/consuming-the-module.md`
- `docs/shipped-state.md`
- `pkg/inferenceengine/contract.go`
- `pkg/inferenceengine/contract_test.go`
