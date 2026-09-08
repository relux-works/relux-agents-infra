# TASK-260830-ter72z — Revision 3 developer evidence

## Delivery

- Owning repository: `skill-agents-management`
- Pull request: https://github.com/relux-works/skill-agents-management/pull/8
- Revision-3 head: `58a063c72ffc298023219d660f1cf7167bc2cb59`
- Branch: `codex/TASK-260830-ter72z-inference-engine-plugin`
- Commit signature: verified SSH signature for the configured human author.
- Remote PR head: verified equal to the reviewed local head.

## Blocking findings addressed

### F1 — caller-minted canonical observations

The public `Engine` contract is declaration-only. It no longer accepts or invokes an observer/derivation method, and `ResolveContract` returns no observed values. An external-package negative test registers an engine with the old `DeriveObservation`-shaped extra method and proves the resolver never calls it and never exposes forged argv.

### F2 — generic JSON was not a fact contract

All 17 facts now have closed, fact-specific value contracts and exact trusted derivation-source declarations. Shape validation rejects the reviewer's false-readiness case (`endpoint_answering=true` while `weights_resident=false`), naive non-mapping-aware memory accounting, invalid load/unload and busy transitions, invalid memory-pressure ordering, and inconsistent speculative-decoding state. Absence, read failure, malformed evidence, and unsupported derivation are distinct and all require refusal where the engine cannot express the fact.

### F3 — unused production gate

The package no longer documents `ResolveObserved` as a production gate and removes that API. The specification explicitly states that neither repository currently composes a runtime observation gate; production admission remains unimplemented until agents-infra supplies the non-replaceable process-owned evidence path. OS process, SSH, polling, and supervision execution ownership remains pinned to `agents-infra`.

The repeated trust-boundary regression, root cause, decision, and protecting test are recorded in the owning repository's `LOGBOOK.md`.

## Verification evidence

Every command below ran directly as a standalone process.

| Command | Exit | Result |
| --- | ---: | --- |
| `gofmt -l pkg internal tools` | 0 | No output |
| `git diff --check` | 0 | No whitespace errors |
| `go test ./pkg/inferenceengine -count=1` | 0 | 13 named top-level package tests pass |
| `go test -race ./pkg/inferenceengine -count=1` | 0 | Race pass |
| `go test -cover ./pkg/inferenceengine -count=1` | 0 | 91.9% statement coverage |
| `make vet` | 0 | Clean |
| `make build BIN=.temp/TASK-260830-ter72z/agents-management` | 0 | Build succeeds |
| `env -u TASK_BOARD_DIR go test -mod=mod ./... -count=1` | 0 | All 29 packages pass |
| `make regress` | 0 | Regression suite passes |
| `env -u TASK_BOARD_DIR go test -mod=mod -race ./... -count=1` | 0 | Full race suite passes |
| `env -u TASK_BOARD_DIR go test -mod=mod -cover ./... -count=1` | 0 | Full coverage suite passes; inferenceengine 91.9% |
| staged `quick_validate.py` against install basename `agents-management` | 0 | Valid skill, 218 body lines |
| `make contract-mutants` | 0 | 3/3 narrowed mutants killed |

The three compile-clean mutants independently weaken: the trusted interface boundary, readiness conjunction validation, and unsupported-refusal validation. Each mutant caused the external-package suite to exit 1 before the pristine suite reran green.

## Release state

Revision 3 is published on PR #8. The next signed patch tag is intentionally deferred until the exact revision-3 head is accepted in review.
