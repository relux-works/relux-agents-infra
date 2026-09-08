# TASK-260830-ter72z — developer rework outcome

## Owning repository delivery

- Repository: `github.com/relux-works/skill-agents-management`
- Branch: `codex/TASK-260830-ter72z-inference-engine-plugin`
- Signed rework commit: `8f4a8483abefed9671eb048a2547bb3553f2950c`
- Parent: signed initial specification commit `4fa4e25c4c7cd314643d021dc0c8cf576c38c543`
- Signature: good SSH signature for `alexis@relux.works`, ECDSA fingerprint `SHA256:60fPOOw38n4bW1moyoPfQ/TZIRwPFLgYf7BfR49npaU`
- Commit time: `2026-08-29T20:44:00+03:00`, matching the owning repository's backdating policy
- Pull request: <https://github.com/relux-works/skill-agents-management/pull/8>
- PR state after publication: open, non-draft, clean; remote head equals the signed commit; no hosted checks are configured

## Rework delivered

- Removed the caller-owned `Observer` parameter and public caller-fillable provenance/value shape from production `ResolveObserved`; the entry point now accepts only context, registry, and engine ID.
- Bound derivation to `Engine.DeriveObservation`; agents-infra composes the concrete engine kind while retaining OS process, SSH, polling, and supervision execution ownership.
- Added a sealed three-outcome result model: canonical `ObservedValue`, successful typed `ObservedAbsent`, and refusing `NotObserved`.
- Distinguished read failure, malformed derivation, unsupported derivation, and unknown cause inside `NotObserved`; none can fall back to absence.
- Added closed value contracts and production revalidation: argv token JSON, JSON field path, clean absolute path, and canonical structured JSON object.
- Covered measured absence for speculative-decoding capability and mutually exclusive local executable versus SSH-forwarding profile expansion.
- Updated README, skill, architecture, consumer, shipped-state, and logbook specifications.

## Validation

| Command | Exit | Result |
| --- | ---: | --- |
| `go test ./pkg/inferenceengine -count=1` | 0 | focused production-entry suite passes |
| `go test ./pkg/inferenceengine -cover -count=1` | 0 | 83.8% statement coverage |
| `env -u TASK_BOARD_DIR go test -mod=mod ./... -count=1` | 0 | final full repository suite passes |
| `env -u TASK_BOARD_DIR go test -mod=mod -race ./... -count=1` | 0 | full race suite passes |
| `go test -race ./pkg/inferenceengine -count=1` | 0 | final focused race pass after the last test change |
| `make vet` | 0 | vet/lint clean |
| `make build BIN=.temp/TASK-260830-ter72z/agents-management` | 0 | CLI builds |
| `make regress` | 0 | regression gate passes |
| `test -z "$(gofmt -l pkg internal tools)"` | 0 | formatting clean |
| `git diff --check` | 0 | whitespace validation clean |

## Negative proof

- Narrowing production `ObservedValue` validation from canonical revalidation to non-empty-only made `TestResolveObservedIndependentlyRefusesAProcessLabeledForgedValue` fail with `ResolveObserved admitted process-labeled forged argv: <nil>`; real test exit `1`.
- Removing only the `Unsupported == refuse` contract predicate made `TestResolveObservedRefusesAnIncompleteOrFallbackContract/unsupported_is_silently_dropped` fail with a nil error; real test exit `1`.
- The unmodified exact tests then exited `0`.
- One earlier mutant invocation exited `1` before `go test` because its redirect targeted a missing directory; it was explicitly discarded and rerun with an existing log path. It is not counted as mutation evidence.

## Scope boundary

No repository files changed in relux-agents-infra. No OS process, SSH, readiness polling, or supervision execution moved out of agents-infra.
