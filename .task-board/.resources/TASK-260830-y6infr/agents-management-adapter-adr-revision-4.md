# ADR: Public engine observation adapters and the Pi one-turn boundary

- Task: `TASK-260830-ivbodb` (`specify-public-observer-and-pi-turn-boundary`)
- Story: `STORY-260830-29753p` (`publish-production-local-runtime-adapter-contract`)
- Decision revision: 4
- Date: 2026-08-30
- Status: proposed for independent exact-revision review

## 1. Requirements and evidence base

This decision implements the task acceptance criteria named **public adapter**, **Process-B ownership**, **Pi argv/stdin/env/result/cancellation**, **status-version compatibility**, **fail-closed error taxonomy**, **consumer migration**, **adversarial tests**, **no identity branching**, and **no live probes**. It enables the Story requirements **versioned sanitized observations**, **pinned Pi transport**, **missing/unknown/stale evidence refusal**, **generic dispatch**, and **release before activation**.

Static sources inspected:

- `skill-agents-management` tag `v0.5.0`, commit `b74f758a90a422304d55460422831da80e3d6cc8`.
- The task precondition `cross-repo-v050-consumer-audit.md`.
- `relux-agents-infra` commit `fe3818209c9861fcafa1f2e68efe078cc0f96f30`; its `tools/agents-infra` subtree had no working-tree delta during inspection.
- The pinned Pi `v0.84.2` Darwin/arm64 release identity `github-release:earendil-works/pi@v0.84.2:darwin-arm64#sha256-c996e888b7f7dce44bcf24f69176ac646c44139d3916bd49a6b28e5a8c5e3a65` and 217-record tree manifest SHA-256 `2f68ab1b3f28a9c4b8995f91984f8f47001a79735da7e57aa7fe6d223f90378b`.

No runtime, process, status command, service, socket, model, network endpoint, or user configuration was contacted.

## 2. v0.5.0 blockers

The following package-private surfaces make a real non-dry-run engine launch impossible for an ordinary consumer:

1. `pkg/vendorplugin/engine.go` defines private `engineObservationQuery`, `engineFactSource`, `productionEngineFactSource`, and `resolveEngineObservations`.
2. `pkg/vendorplugin/registry.go` stores private `Registry.engineFacts`; `NewRegistry` always installs `productionEngineFactSource`; only private `newRegistryWithEngineFactSource` can replace it.
3. `productionEngineFactSource` special-cases `mlx` and returns `NotObservedUnsupported("agents-infra has not supplied the concrete MLX observation adapter")` for all 17 measured facts.
4. `inferenceengine.NewReading` and `ValidateReadings` are public but intentionally schema-only. They cannot install a production source or authorize `BuildLaunch`.
5. `pkg/agentic/systems/pi` pins only `agents-infra pi --profile <profile> --`, then uses the prompt-file/prompt-bytes stdin fallback. Its own source explicitly says the real Pi turn grammar is unresolved.

The public seam must therefore be added at registry construction, not to `SpawnRequest` or `BuildLaunch`. Adding observations to either per-launch input would let the launch caller mint the evidence checked by the gate.

## 3. Decision: exact public observation API

Add the following public API in `pkg/vendorplugin`:

```go
const EngineObservationAdapterContract = "agents-management.engine-observation-adapter"
const EngineObservationAdapterSchemaVersion = 1

type EngineObservationAdapterDeclaration struct {
    Contract        string
    SchemaVersion   int
    Engine          plugin.Ref
    EngineKind      inferenceengine.EngineKind
    EngineContract  string
}

type EngineObservationQuery struct {
    Engine   plugin.Ref
    Runtime  RuntimeID
    Model    ModelID
    Profile  string
}

type EngineObservation struct {
    Contract       string
    SchemaVersion  int
    Engine         plugin.Ref
    Runtime        RuntimeID
    Model          ModelID
    Profile        string
    ObservedAt     time.Time
    ValidUntil     time.Time
    Readings       []inferenceengine.Reading
}

type EngineObservationAdapter interface {
    EngineObservationAdapterDeclaration() EngineObservationAdapterDeclaration
    ObserveEngine(context.Context, EngineObservationQuery) (EngineObservation, error)
}

func NewRegistryWithEngineObservationAdapters(
    systems *agentic.Registry,
    adapters ...EngineObservationAdapter,
) (*Registry, error)
```

Add these public sentinel errors in `pkg/vendorplugin`:

```go
var (
    ErrEngineObservationAdapterMissing = errors.New("vendorplugin: inference-engine observation adapter is missing")
    ErrEngineObservationAdapterInvalid = errors.New("vendorplugin: inference-engine observation adapter is invalid")
    ErrEngineObservationVersion        = errors.New("vendorplugin: inference-engine observation version is unsupported")
    ErrEngineObservationIdentity       = errors.New("vendorplugin: inference-engine observation identity does not match the query")
    ErrEngineObservationStale          = errors.New("vendorplugin: inference-engine observation is stale")
)
```

### 3.1 Construction and immutability

- Each adapter declares exactly one normalized `plugin.Ref` of kind `inferenceengine.Kind`, one valid `EngineKind`, exact adapter contract/schema, and exact `inferenceengine.ContractVersion` (`observed-process/v2` in v0.5.0).
- Construction rejects nil and typed-nil adapters, malformed declarations, wrong kind, duplicate engine refs, unknown adapter contract/schema, and unknown engine contract. Registration is atomic: one bad adapter installs none of the batch.
- Construction calls `EngineObservationAdapterDeclaration()` exactly once, validates it, copies the value into registry-owned storage, and never consults the method again. Later mutation or a declaration method that would return a different value cannot change adapter identity, selection, or validation.
- The registry stores an immutable map keyed only by the declared engine ref. There is no setter and no adapter field on `SpawnRequest`, `EngineObservationQuery`, or `BuildLaunch`.
- Existing `NewRegistry(systems) *Registry` remains source-compatible and installs no positive adapter. A non-dry-run engine-bound launch through it returns `ErrEngineObservationAdapterMissing` (wrapping `inferenceengine.ErrEngineContractMissing`) after authoritative pure value construction but before Pi preflight, plan, child, state, lease, or Process-B effects. Dry run remains observation-free.
- Remove the `mlx` branch from generic `productionEngineFactSource`; concrete identity-to-implementation mapping comes from the registered adapter declaration. Generic core must contain no conditional on runtime, model, publisher, family, profile, or engine literal.

### 3.2 Production-entry validation order

For non-dry-run engine-bound `BuildLaunch`, after graph/model/effort resolution:

1. Check the caller context. A pre-cancelled call returns its original context class before vendor or adapter work.
2. Construct the authoritative `agentic.LaunchRequest`: call the vendor's pure value-producing `Spawn` (or the existing system-only passthrough), apply `checkLaunchFidelity`, and set authoritative runtime. This resolves/fills the trusted profile before observation and is permitted to allocate only local values—no preflight, plan, child, state, lease, or Process-B effect.
3. If caller profile conflicts with the vendor declaration, refuse before adapter lookup. If absent, the adapter sees only the trusted profile filled by the vendor.
4. Look up the adapter by the already-resolved engine `plugin.Ref`.
5. Derive an observation context with `engineObservationTimeout = 10 * time.Second`; an earlier caller deadline wins. `EngineObservationAdapter` is a trusted cooperative interface: implementations must return promptly after `ctx.Done()`. The module does not claim that a context-ignoring in-process implementation can be forcibly stopped.
6. Call the adapter once with authoritative `{Engine, Runtime, Model, launch.Profile}`. Preserve context errors; any other read error wraps `inferenceengine.ErrObservationRead`. The concrete agents-infra adapter must additionally bound every underlying read.
7. Check the original caller context again after return and before response validation, preflight, or plan; cancellation never admits a late response.
8. Require response contract/schema `agents-management.engine-observation-adapter`/`1`.
9. Require response engine/runtime/model/profile to equal the authoritative query byte-for-byte. The response never redirects dispatch.
10. Require non-zero timestamps, `ObservedAt <= now`, `ValidUntil > ObservedAt`, and `now < ValidUntil`. The trusted adapter owns the validity interval; the module invents no TTL. Expired, future-dated, inverted, or zero intervals return `ErrEngineObservationStale`.
11. Call `inferenceengine.ValidateReadings` with the snapshotted declaration's engine kind and a copied reading slice. The response cannot select its own implementation kind or contract.
12. Continue to preflight and `agentic.BuildPlan` only after every gate accepts.

The adapter may obtain facts only through bounded, read-only facilities owned by agents-infra. Its public result is sanitized to the fields above and the existing closed `Reading` values. It must not return PIDs, sockets, credentials, environment values, raw status JSON, SSH secrets, process handles, callbacks, or lifecycle operations.

## 4. Ownership split

| Concern | `agents-management` | `agents-infra` / consumer |
| --- | --- | --- |
| Engine/plugin identity | Register and resolve generic `plugin.Ref`; reject mismatch | Select and construct the concrete adapter at trusted registry assembly |
| Observation | Validate version, identity, freshness, closed facts, and cross-fact invariants | Perform the bounded read and sanitize the result |
| Process B | Describe identity/provenance only | Start, supervise, poll, signal, stop, restart, quarantine, and reap |
| Process A / Pi plan | Produce the deterministic agents-infra standalone command and Process-A environment | Validate exact profile/policy, build the inner Pi argv/environment, execute Pi, and own process group, deadline, capture, and cleanup |
| Result | Own the public `pi.ValidateTurnResult` classifier, its closed types, and exit/document/intervention/cleanup precedence | Translate the pinned raw Pi output into the schema-1 document; raw JSONL remains private to agents-infra |

Neither the adapter call nor Pi planning may start, stop, signal, attach to, or mutate Process B. `BuildLaunch` remains a value-producing boundary.

## 5. Decision: Pi one-turn transport

Revision 1 incorrectly placed standalone flags after ordinary `agents-infra pi --profile ... --`. Static source inspection shows that only `agents-infra pi spawn` enters the standalone path; the ordinary path inherits stdin and does not own the standalone policy, state, process-group, or cleanup guarantees. The current standalone command also has no exact-profile selector and exposes raw Pi output rather than a versioned sanitized success result. Copying inner flags onto the ordinary path would be a forced fit.

The smallest clean contract adds two backwards-compatible flags to agents-infra's existing standalone entry point:

```text
agents-infra pi spawn --profile <exact-profile> --prompt <exact-prompt> --deadline 30m --result-schema 1
```

`--profile` is an exact assertion over the runtime-resolved profile: no aliasing, case folding, normalization, or fallback is allowed. Absent, unknown, or mismatched profile refuses before Pi or runtime effects. `--result-schema 1` requests the versioned result translation below; callers that omit it retain the current raw-output behavior.

For `LaunchModeExec`, `pkg/agentic/systems/pi` resolves the agents-infra binary and emits exactly these `Plan.Argv` entries, in order:

```text
pi
spawn
--profile
<exact profile>
--prompt
<exact prompt UTF-8 as one argv value>
--deadline
30m
--result-schema
1
```

The module does not construct Pi's inner `--tools`, approval, extension, mode, session, provider, model, thinking, auth, state, or telemetry arguments. Agents-infra's standalone policy remains the sole authority for those values.

### 5.1 Prompt, stdin, environment, and modes

- Source prompt bytes use the existing `PromptPath`-before-`Prompt` precedence. A file read failure is an error, not an empty prompt.
- Decode bytes as UTF-8 and reject empty, invalid UTF-8, or NUL-containing prompts with public `pi.ErrTurnPromptInvalid`. Leading `-` and `@` are legal because the prompt is the value of a named flag, not a free operand.
- The prompt is one argv value; it is never copied to stdin. `Plan.Stdin` is the zero `StdinPayload` (`Attached=false`, no bytes), Process A observes EOF, and agents-infra closes Pi stdin.
- `LaunchModeDryRun` uses the same outer argv grammar but replaces the prompt value with the literal `<prompt>` and performs no prompt-file read, observation, preflight, executable action, or other side effect.
- Pi supports only Exec and DryRun. ManagedSession remains refused.
- Process A receives parent-environment passthrough followed by `agentic.WithRunContext`, with vendor-supplied `AGENTS_INFRA_CALLER_CWD` preserved. Agents-infra then independently validates Process A's input, constructs Pi's child environment, and owns `PI_CODING_AGENT_DIR`, `PI_CODING_AGENT_SESSION_DIR`, `PI_SKIP_VERSION_CHECK`, and `PI_TELEMETRY`.
- The exact effective denial contract is: malformed or duplicate environment entries; exact names `HF_ENDPOINT`, `MODEL_ENDPOINT`, `GGML_BACKEND_PATH`, and `LLAMA_API_KEY`; case-insensitive prefixes `DYLD_`, `LD_`, `NODE_`, `BUN_`, and `LLAMA_ARG_`; and exact inbound managed names `PI_CODING_AGENT_DIR`, `PI_CODING_AGENT_SESSION_DIR`, `PI_SKIP_VERSION_CHECK`, and `PI_TELEMETRY`. The first three groups are enforced by `ValidatePiExecutionEnvironment`; the four managed names are separately rejected by the existing Process-A execution path before identity or child work. Other `PI_*` names are not claimed to be denied by this contract. A denied value is never logged or included in a diagnostic.
- Missing/blank profile returns public `pi.ErrProfileMissing`. Caller-supplied Pi flags are impossible because the System constructs the entire suffix.

### 5.2 Result and cancellation semantics

- Agents-infra owns the raw Pi parser and translation. Before implementing it, agents-infra must add a content-addressed static fixture or source excerpt from the pinned Pi `v0.84.2` tree that proves the raw event grammar. A consumer-authored adjacent fake fixture is not evidence of that grammar.
- With `--result-schema 1`, Process-A stdout is exactly one UTF-8 JSON object, bounded to 1 MiB including permitted trailing ASCII space, tab, CR, or LF. Object-member order is irrelevant; duplicate keys, unknown fields, a second value, non-whitespace trailing bytes, invalid UTF-8, or an over-limit document are refused. Raw Pi JSONL remains private to agents-infra.
- Success has exactly `{"contract":"agents-infra.pi-turn-result","schema_version":1,"status":"ok","final_text":"..."}`. Error has exactly `{"contract":"agents-infra.pi-turn-result","schema_version":1,"status":"error","error":{"code":"<code>"}}`; `final_text` is forbidden on error and `error` is forbidden on success.
- Schema 1 has the following closed error vocabulary. Every code in the first three groups exits `1`; cancellation/deadline exit `2`; result-protocol invalidity exits `3`.

  | Class | Exact schema-1 codes |
  | --- | --- |
  | cleanup failed | `pi_turn_cleanup_failed` |
  | Process-A pre-child refusal | `pi_turn_request_invalid`, `pi_turn_profile_missing`, `pi_turn_profile_unknown`, `pi_turn_profile_mismatch`, `pi_turn_environment_malformed`, `pi_turn_environment_denied`, `pi_turn_configuration_invalid`, `pi_turn_authorization_denied`, `pi_turn_identity_invalid`, `pi_turn_runtime_refused` |
  | Pi child/tool failure | `pi_turn_child_failed`, `pi_turn_tool_failed` |
  | cancellation/deadline | `pi_turn_cancelled`, `pi_turn_deadline_exceeded` |
  | result protocol invalid | `pi_turn_result_invalid` |

- Process A installs the schema-1 result writer immediately after it has positively parsed `--result-schema 1`, before request, profile, environment, project configuration, authorization, Pi identity, or runtime admission/preflight validation. From that point until exit, exactly one result document is mandatory for every uninterrupted path, including every pre-child refusal. Failure to write it is observed by the consumer as `pi_turn_result_invalid`; it is never inferred back into the intended preflight class. OS failure to start Process A is outside schema 1 because no command started and is returned by the consumer's process-start boundary.
- `pi_turn_request_invalid` covers only malformed outer request fields after schema 1 was selected: invalid prompt, deadline, or conflicting/unknown standalone flags. Profile absence, unknown identity, and byte mismatch have their own codes. A failed/partial/malformed project-config read is `pi_turn_configuration_invalid`, never absence. Missing/invalid yolo or tool allowlist and denied tool policy are `pi_turn_authorization_denied`. Missing, non-regular, non-executable, wrong-version, wrong-digest, copied, or point-of-use-changed pinned Pi is `pi_turn_identity_invalid`. A broker/readiness/admission refusal before Pi child creation is `pi_turn_runtime_refused`. Error documents and stderr contain codes and sanitized diagnostics only; profile names, environment values, paths, credentials, raw configuration, status payloads, and child output are forbidden.
- Result precedence is: cleanup failure; consumer or self-reported cancellation/deadline; Process-A pre-child refusal; non-zero Pi child exit; malformed, duplicated, contradictory, over-limit, or premature-EOF upstream output; tool failure; success. Agents-infra emits the single highest-precedence code. Cleanup discards every lower class; cancellation/deadline discards pre-child/child/result/tool classes; a pre-child refusal is possible only before a Pi child exists and discards result/tool classes; child failure discards result/tool classes.
- For an uninterrupted Process A, exit/document absence or disagreement is refused: `0` requires `ok`; `1`, `2`, and `3` require the matching error-code class above. Stderr is diagnostic-only and never substitutes for a result.
- Context cancellation before or during adapter/preflight returns the original context class and starts no child. After a child exists, the consumer records cancellation first, sends bounded terminate-then-kill only to Process A's owned process group, and never signals Process B directly. If that cleanup succeeds, the recorded consumer cancellation remains authoritative even when the induced signal exit has no result document or disagrees with schema 1; no post-kill write is required. If bounded cleanup itself fails, cleanup failure takes precedence. A self-reported Process-A deadline/cancellation without consumer intervention still requires exit `2` and its matching document. Releasing Process A releases only its shared-runtime lease; agents-infra's broker decides Process-B lifetime.
- Cancellation/deadline, child exit, malformed/incomplete JSONL, tool failure, and cleanup failure remain distinct result classes. No class falls back to a previous result or to success.

Process execution remains a consumer responsibility; raw Pi parsing belongs to agents-infra. Versioned-result classification has one authoritative implementation in agents-management, specified below. Pi package tests pin the Process-A plan and classifier surfaces. Agents-infra tests drive the real standalone entry point against the content-addressed Pi fixture and a fake child; consumer integration tests drive the real child-launch/result entry point against fake Process-A output and must invoke the exported classifier.

### 5.3 Exact public result-validation API

Add the following public API in `pkg/agentic/systems/pi`:

```go
const TurnResultContract = "agents-infra.pi-turn-result"
const TurnResultSchemaVersion = 1
const TurnResultMaxStdoutBytes = 1 << 20

type TurnResultClass string

const (
    TurnResultOK               TurnResultClass = "ok"
    TurnResultProcessARefused  TurnResultClass = "process-a-refused"
    TurnResultChildFailed      TurnResultClass = "child-failed"
    TurnResultToolFailed       TurnResultClass = "tool-failed"
    TurnResultCancelled        TurnResultClass = "cancelled"
    TurnResultDeadlineExceeded TurnResultClass = "deadline-exceeded"
    TurnResultInvalid          TurnResultClass = "result-invalid"
    TurnResultCleanupFailed    TurnResultClass = "cleanup-failed"
)

type TurnResultCode string

const (
    TurnCodeCleanupFailed       TurnResultCode = "pi_turn_cleanup_failed"
    TurnCodeRequestInvalid      TurnResultCode = "pi_turn_request_invalid"
    TurnCodeProfileMissing      TurnResultCode = "pi_turn_profile_missing"
    TurnCodeProfileUnknown      TurnResultCode = "pi_turn_profile_unknown"
    TurnCodeProfileMismatch     TurnResultCode = "pi_turn_profile_mismatch"
    TurnCodeEnvironmentMalformed TurnResultCode = "pi_turn_environment_malformed"
    TurnCodeEnvironmentDenied   TurnResultCode = "pi_turn_environment_denied"
    TurnCodeConfigurationInvalid TurnResultCode = "pi_turn_configuration_invalid"
    TurnCodeAuthorizationDenied TurnResultCode = "pi_turn_authorization_denied"
    TurnCodeIdentityInvalid     TurnResultCode = "pi_turn_identity_invalid"
    TurnCodeRuntimeRefused      TurnResultCode = "pi_turn_runtime_refused"
    TurnCodeChildFailed         TurnResultCode = "pi_turn_child_failed"
    TurnCodeToolFailed          TurnResultCode = "pi_turn_tool_failed"
    TurnCodeCancelled           TurnResultCode = "pi_turn_cancelled"
    TurnCodeDeadlineExceeded    TurnResultCode = "pi_turn_deadline_exceeded"
    TurnCodeResultInvalid       TurnResultCode = "pi_turn_result_invalid"
)

type TurnIntervention string

const (
    TurnInterventionNone      TurnIntervention = "none"
    TurnInterventionCancel    TurnIntervention = "cancel"
    TurnInterventionDeadline  TurnIntervention = "deadline"
)

type ProcessACleanupOutcome string

const (
    ProcessACleanupNotRequired ProcessACleanupOutcome = "not-required"
    ProcessACleanupSucceeded   ProcessACleanupOutcome = "succeeded"
    ProcessACleanupFailed      ProcessACleanupOutcome = "failed"
)

type ProcessAExit struct {
    Code     int
    Signaled bool
}

type TurnResultInput struct {
    Stdout          []byte
    StdoutTruncated bool
    Exit            ProcessAExit
    Intervention    TurnIntervention
    Cleanup         ProcessACleanupOutcome
}

type TurnResult struct {
    Class     TurnResultClass
    Code      TurnResultCode
    FinalText string
}

type TurnResultError struct {
    Class TurnResultClass
    Code  TurnResultCode
}

func (*TurnResultError) Error() string
func (*TurnResultError) Unwrap() error

func ValidateTurnResult(TurnResultInput) (TurnResult, error)
```

Add public sentinels `ErrTurnResultInvalid`, `ErrTurnProcessARefused`, `ErrTurnChildFailed`, `ErrTurnToolFailed`, and `ErrTurnCleanupFailed`. `TurnResultError.Unwrap` maps its class to the corresponding sentinel; cancellation and deadline map to `context.Canceled` and `context.DeadlineExceeded`. Its `Error` method contains only class and code, never stdout, final text, environment, profile, path, or cleanup detail.

`ValidateTurnResult` copies the caller's stdout before parsing and is the sole schema-1 parser/classifier. `StdoutTruncated`, more than 1 MiB, an invalid input enum combination, a signal exit without recorded consumer intervention, an unknown exit, or any document/schema/code disagreement returns `TurnResultInvalid` / `TurnCodeResultInvalid`. With no intervention, `Cleanup` must be `not-required` and the exact exit/document table is enforced. With cancellation/deadline intervention after Process A started, cleanup must be `succeeded` or `failed`: failed cleanup returns `TurnResultCleanupFailed` regardless of exit/document; successful cleanup returns the original cancellation/deadline class regardless of the induced signal exit or absent/disagreeing document. Pre-start cancellation returns directly from the process-start boundary and never calls this validator.

The consumer's production child-launch/result entry must pass its bounded capture, actual waited Process-A exit, recorded intervention, and bounded-cleanup outcome to `pi.ValidateTurnResult` exactly once. It must not independently unmarshal schema 1, classify an exit, or accept a result around this call. Agents-infra is the document producer, not a second consumer-side classifier.

## 6. Version compatibility

| Surface | Accepted | Refused |
| --- | --- | --- |
| Observation adapter declaration | exact contract `agents-management.engine-observation-adapter`, schema `1`, engine contract `observed-process/v2` | unknown contract/schema/engine contract; no optimistic forward compatibility |
| Observation response | exact schema `1`, exact query identity, live validity interval | missing/unknown version, identity drift, zero/future/expired/inverted interval |
| `localruntime.Status` | adapter-owned `local-runtime.status`, schema `1`; complete legacy, pre-deadline, or current restart cohorts decode to v1 with presence flags | partial cohort, malformed present field, or unknown adapter contract/schema |
| Pi binary | pinned agents-infra compatibility for `earendil-works/pi@v0.84.2` and the exact manifest/asset digests above | different, missing, copied, partially matching, or point-of-use changed tree |
| Process-A Pi command | exact `pi spawn --profile ... --prompt ... --deadline 30m --result-schema 1`, EOF stdin, and two-hop environment ownership in §5 | ordinary `pi`, profile fallback/normalization, inner Pi flags from the module, stdin attachment, or environment-authority drift |
| Pi turn result | exact contract `agents-infra.pi-turn-result`, schema `1`; exact closed pre-child/child/tool/cancellation/deadline/cleanup codes; `pi.ValidateTurnResult` over bounded stdout, actual exit, intervention, and cleanup | raw JSONL, local permissive parser, unknown fields/version/status/code, duplicate/trailing document, invalid input enum, signal without intervention, or exit/document disagreement |

The local-runtime decoder's three accepted producer generations remain a wire compatibility detail. Every successful value is stamped as `local-runtime.status` schema 1. Absence of an additive cohort and failure/partial presence are never equivalent.

## 7. Fail-closed production-entry attack matrix

Every row must drive `vendorplugin.BuildLaunch` or the consumer's real child-launch/result entry point, assert the typed class, and assert zero effects appropriate to the stage.

| Attack / mutant | Required refusal | Required side-effect proof |
| --- | --- | --- |
| No adapter for resolved engine | `ErrEngineObservationAdapterMissing` | pure vendor value construction permitted; zero Pi Preflight, plan, child, state, lease, or Process-B operation |
| Nil/typed-nil, malformed, wrong-kind, duplicate adapter declaration | `ErrEngineObservationAdapterInvalid` at registry construction | atomic: no adapter from the batch installed |
| Adapter declaration method would mutate after its first read | no runtime identity change; copied first declaration remains authoritative | method called exactly once; later state cannot redirect lookup or validation |
| Unknown adapter/schema/engine-contract version | `ErrEngineObservationVersion` | zero launch effects |
| Response changes engine/runtime/model/profile | `ErrEngineObservationIdentity` | zero launch effects |
| Zero/future/expired/inverted observation interval | `ErrEngineObservationStale` | zero launch effects |
| Adapter returns read error | `ErrObservationRead`, preserving context/error cause | never treated as observed absence |
| Cooperative adapter blocks until `ctx.Done()` | `context.DeadlineExceeded` at the named 10-second module bound | vendor may have constructed a pure value; zero preflight, plan, child, state, lease, or Process-B effect |
| Concrete adapter underlying read exceeds its own bound | preserved deadline/read refusal | no preflight, plan, child, state, lease, or Process-B effect; interface does not claim forcible interruption of a ctx-ignoring implementation |
| Missing, reordered, duplicated, unknown, or malformed fact | existing `ErrObservationMalformed` | zero launch effects |
| Unsupported fact | existing `ErrObservationUnsupported` | zero launch effects |
| Wrong configured/caller/resolved engine ref or graph kind | existing engine mismatch/graph refusal | adapter is not called when identity resolution already failed |
| Caller profile absent and vendor fills the declared profile | adapter receives only the filled authoritative profile | pure vendor construction precedes exactly one adapter call |
| Caller profile conflicts with vendor declaration | vendor profile-conflict refusal | adapter not called; zero preflight, plan, child, state, lease, or Process-B effect |
| Runtime/model/profile/engine literal branch inserted in generic core | metamorphic identity-renaming and literal-placement guard fails | no special-case dispatch ships |
| Prompt missing/unreadable/invalid UTF-8/NUL | `pi.ErrTurnPromptInvalid` | no observation, preflight, child, or runtime side effect at initial gate; leading `-`/`@` remain accepted flag values |
| Ordinary `pi` path, outer argv reorder/drop/duplication, inner Pi flags from module, or stdin reattached | launch-surface golden/parity failure | exact binary/argv/Process-A env/stdin comparison |
| Exact profile absent, unknown, normalized, or mismatched at the real standalone entry | exact `pi_turn_profile_missing`, `pi_turn_profile_unknown`, or `pi_turn_profile_mismatch` document and exit `1` | zero Pi child, lease, state, or Process-B operation; diagnostic does not expose the supplied profile |
| Malformed/duplicate environment entry at the real standalone entry | exact `pi_turn_environment_malformed` document and exit `1` | zero Pi child/runtime effect and no environment bytes in output |
| Exact denied name, denied prefix family, or one of four managed Pi names at the real standalone entry | exact `pi_turn_environment_denied` document and exit `1` | zero Pi child/runtime effect and no denied name/value in diagnostic |
| Failed/partial/malformed configuration read, invalid tool authorization, pinned Pi missing/mismatch/change, or runtime preflight refusal | exact `pi_turn_configuration_invalid`, `pi_turn_authorization_denied`, `pi_turn_identity_invalid`, or `pi_turn_runtime_refused` document and exit `1` | real standalone entry starts no Pi child and performs no unauthorized runtime mutation; sanitized code-only result |
| Consumer real result entry receives each missing/unknown/mismatched/malformed/denied standalone result with exit `1` | `pi.ValidateTurnResult` returns `TurnResultProcessARefused` and the exact closed code | no fallback to `TurnResultInvalid`, child/tool class, previous result, or success |
| Consumer production entry bypasses `pi.ValidateTurnResult`, invokes it only on a helper path, or adds a permissive parser around it | static production-call-site guard and narrowed mutant fail; malformed/unknown/exit-disagreement integration cases are admitted by the mutant and therefore fail the named tests | no result persistence or success before the one authoritative call returns |
| Zero exit plus absent/malformed/duplicate/contradictory/trailing/unknown result document | versioned-result refusal | no success/result persistence |
| Error exit plus `ok`, wrong/unknown error code, missing required document, duplicate key, >1 MiB, or other uninterrupted exit/document disagreement | versioned-result refusal | no false success or class laundering |
| Consumer records cancellation then causes signal exit/no document | original cancellation class after successful bounded cleanup | absence/disagreement is not laundered into protocol failure; Process B is never signalled |
| Pinned Pi parser accepts an invalid/missing/duplicate/tool-error event | content-addressed Pi v0.84.2 fixture/source test fails | raw grammar is proved only at agents-infra's real standalone translation entry point |
| Pre-cancelled context | `context.Canceled` or `context.DeadlineExceeded` | zero adapter read when already cancelled; zero child/runtime effects |
| Cancellation after child start | cancellation class after bounded Process-A cleanup | zero direct Process-B signal; shared peer lease remains valid |
| Live probe/process/socket/model access inserted into static/fake tests | static import/call guard and fixture test fail | test suite remains hermetic |

Narrowing mutants must target one admitted class, not merely delete the whole gate. Tests must obtain the artifact through the production call site rather than supplying already-resolved helper inputs.

## 8. Consumer migration sequence

1. In agents-management, implement the public adapter types, single-read copied declaration, atomic registry constructor, pure authoritative launch/profile construction, cooperative pre/post-cancellation and 10-second production-entry gates, sentinels, and static/fake attack matrix while preserving `NewRegistry` and all zero-engine launch bytes.
2. In the same reviewed agents-management change, replace the v0.5.0 Pi stdin fallback with the exact Process-A `pi spawn` Plan and implement the exact public `pi.ValidateTurnResult` API in §5.3. Do not copy inner Pi flags or parse raw Pi JSONL in the module. Publish the exact reviewable commit/API artifact before a concrete adapter tries to compile against it.
3. In agents-infra, compile against that exact agents-management artifact; add exact-profile standalone selection, install the schema-1 result writer before all post-selection validation, map every pre-child refusal to §5.2's exact code/exit, add the closed/bounded `agents-infra.pi-turn-result` translation backed by content-addressed Pi v0.84.2 grammar evidence, enforce the exact environment refusals in §5.1, and implement the concrete cooperative bounded read-only MLX adapter. Process-B lifecycle ownership does not move.
4. At trusted consumer assembly, construct the registry with `NewRegistryWithEngineObservationAdapters`; never pass an adapter through spawn request/config/provenance. Remote/server and dry-run paths install no live adapter and perform no local observation.
5. Validate the cumulative cross-repository contract and release a new immutable agents-management semver tag. Consumers upgrade, regenerate dependency snapshots, and rerun the full negative matrix before enabling local Qwen.
6. Activate local Qwen only after the new tag and the matching agents-infra adapter/result contract are pinned. Configuration drift requires a fresh explicit spawn; persisted evidence is never rewritten.

## 9. Board decomposition and traceability check

Static source review proved one cross-repository prerequisite that the original three-task Story did not own. The smallest complete decomposition is four tasks:

1. `TASK-260830-ivbodb` answers the exact public-observer and Pi-wire questions left open by v0.5.0 and the cross-repo audit.
2. `TASK-260830-1fy32f` implements and independently reviews the agents-management public API, Process-A Pi Plan, exact `pi.ValidateTurnResult` API, and static/fake production-entry/bypass attacks; it is blocked by task 1 and publishes the exact Go artifact needed downstream.
3. `TASK-260830-izr4hp` compiles against task 2's exact artifact and publishes exact-profile standalone selection, the pre-validation schema writer, every exact pre-child code/exit, the pinned versioned Pi result translation, and the concrete read-only MLX adapter in agents-infra; it is blocked by task 2.
4. `TASK-260830-2qjzto` validates the exact cumulative cross-repository public release surface and immutable handoff; it is blocked by task 3.

**Justified gap.** The missing piece is an agents-infra production surface that can (a) select the runtime-resolved profile through its true standalone path, (b) translate pinned Pi output to a versioned sanitized success/error result, and (c) implement the new public MLX observation interface. Without it, the Story requirements **pinned Pi transport/result**, **versioned sanitized observations**, and **production adapter before activation** can be met only by using the wrong ordinary path, self-minting a raw grammar, or leaving every real launch refused. One companion-contract task closes the gap at the owning source without moving Process-B lifecycle into this module.

The gap was self-verified before board creation against this task's Description, Scope, and Acceptance Criteria; the parent Story's public-adapter, pinned-transport, and release requirements; the complete attached audit, especially **Hard blockers**, **Story B**, and its out-of-scope/no-live constraints; and the repository ownership rules. The audit explicitly assigns these production contracts to agents-management plus agents-infra. Nothing checked excludes this work; live runtime/model/socket/service probing, lifecycle transfer, cloud/network work, board-local callbacks, and identity-specific bypasses remain excluded. There is no unresolved fact to research, so no research task is added. No separate documentation, quality-gate, or diagram task is justified.

## 10. Rejected alternatives

- **Adapter on `SpawnRequest` or `BuildLaunch`:** lets the launch caller self-mint evidence.
- **Public registry setter:** permits post-construction source replacement and races.
- **Board-local callback, environment escape hatch, bypass flag, or fake positive production source:** defeats the prerequisite rather than supplying it.
- **Branch on `local-qwen`, Qwen, MLX, publisher/family, or model substring:** creates a second dispatch authority and fails metamorphic genericity.
- **Copy standalone flags after ordinary `agents-infra pi --profile ... --`:** does not enter standalone ownership and falsely claims EOF/process-group/policy guarantees.
- **Invent a raw Pi JSONL parser from a consumer-authored fake fixture:** self-mints the evidence; the grammar must be tied to a content-addressed Pi v0.84.2 source or fixture and translated by agents-infra.
- **Have agents-management execute or supervise Process B:** violates the established ownership boundary and makes dry-run planning effectful.
- **Optimistically accept unknown versions or partial status cohorts:** launders failed/unknown evidence into compatibility.
- **Perform live verification during this Story:** violates scope; all acceptance uses static files and fake adapters/children.

## 11. Review acceptance checklist

An independent reviewer must verify this exact file digest and explicitly accept or reject:

- the API names/signatures and registry-construction-only trust boundary;
- immutable adapter identity/version/freshness validation before launch effects;
- Process-B ownership remaining entirely in agents-infra;
- authoritative vendor-resolved profile before the adapter and single-read copied adapter declaration;
- exact standalone Process-A argv, complete two-hop environment denial, EOF stdin, closed/bounded versioned result, and cancellation rules;
- the exact public `pi.ValidateTurnResult` input/output/error surface, its sole-parser production call, and the complete pre-child refusal code/exit/document/precedence mapping;
- exact compatibility table and fail-closed attack matrix;
- absence of identity branches, live probes, invented board scope, and unnecessary tasks.
