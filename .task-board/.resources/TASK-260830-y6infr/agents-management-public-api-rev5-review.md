# Review verdict: CR-TASK-260830-1fy32f-5 revision 5

## Verdict

ACCEPTED. Exact revision 5 satisfies the agents-management public observer and Pi Process-A contract and closes revision-4 finding F1.

Reviewed base `45c7bd801845ae029d6e78bfa3e2cbef2a146b53`, candidate tree `454e2aae8f975f7991edd34cb8cd96171afde790`, and binary patch SHA-256 `471ebb6ce1b3805bcc8213248ebf0f8ac3c67c7ee215eb279c2f4490c76d96cb`. A temporary alternate-index reconstruction of the current worktree produced the exact candidate tree, so all reviewer gates ran against the immutable CR snapshot.

## Revision-4 closure

The no-live observer guard now discovers every regular non-test Go source in `pkg/vendorplugin` instead of trusting a fixed production filename list. The exact called-helper bypass is covered: a newly discovered `observer_live.go` imports `os` under an alias and is called from `resolveEngineObservations`; the guard reports the helper's forbidden import. Narrowing the composed scan still fails. This closes the prior **bypass path around the check** without weakening the import-aware user-config refusal.

## Contract and adversarial review

- The public adapter types and constructor match ADR revision 4. Nil/typed-nil, malformed, duplicate, wrong-kind, unknown contract/schema/engine-contract declarations refuse atomically. Declarations are read once and copied into the registry-owned engine-ref map.
- `vendorplugin.BuildLaunch` resolves graph/model/effort, honors pre-cancellation, constructs the pure vendor-owned launch/profile, refuses a caller/vendor profile conflict, snapshots Pi prompt input, then performs exactly one cooperatively bounded observation. Missing, read-failed, unknown-version, identity-drifted, stale, malformed, unsupported, and contradictory observations refuse before preflight or plan materialization.
- Dry-run remains observation/preflight free. Generic production core contains no Pi/Qwen/MLX/model/profile dispatch branch; metamorphic identity-renaming and literal-placement tests pass.
- Pi emits the exact `pi spawn --profile <exact> --prompt <one argv value> --deadline 30m --result-schema 1` suffix, keeps stdin detached/EOF, accepts leading `-` and `@` in the named prompt value, refuses missing/invalid UTF-8/NUL/unreadable prompts before observation/preflight, and emits no inner Pi flags.
- `pi.ValidateTurnResult` owns the closed schema-1 parser/classifier. Attacks cover leading bytes, duplicate and unknown fields, second/trailing values, invalid UTF-8, unknown contract/schema/status/code, exit/document disagreement, signal without intervention, invalid enums, truncation, the 1 MiB bound, and cancellation/deadline/cleanup precedence. Ordinary, aliased, and dot-imported permissive JSON parser mutants are rejected.
- Process-B lifecycle and raw Pi translation remain in agents-infra. Per the accepted four-task ADR sequence and this task's agents-management-only scope, the real agents-infra child-launch call-site/one-call integration proof remains mandatory in downstream `TASK-260830-izr4hp`; this CR does not fake that consumer evidence or edit agents-infra.

## Reviewer verification

- Exact binary patch digest: match.
- Alternate-index candidate-tree reconstruction: exact `454e2aae8f975f7991edd34cb8cd96171afde790`.
- `git diff --check <base> <candidate>`: exit 0.
- `go test ./pkg/vendorplugin ./pkg/agentic/systems/pi -count=1`: exit 0.
- Full uncached `env -u TASK_BOARD_DIR go test -mod=mod ./... -count=1`: exit 0.
- Focused adversarial replay across observer, parser, profile, prompt, missing-adapter, identity-renaming and literal-placement cases: exit 0.
- `go test -race ./pkg/vendorplugin ./pkg/agentic/systems/pi -count=1`: exit 0.
- `make vet`, `make build BIN=.temp/review-TASK-260830-1fy32f/agents-management`, and `make regress`: exit 0.
- Repository `gofmt -l` assertion: zero output; exit 0.
- No live runtime, process, service, socket, model, network endpoint, status command, or user configuration was contacted.

No blocking or non-blocking review finding remains for revision 5.
