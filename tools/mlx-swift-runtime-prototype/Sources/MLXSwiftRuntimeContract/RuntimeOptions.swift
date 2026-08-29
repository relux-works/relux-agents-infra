import Foundation

/// Command-line configuration for the prototype runtime.
///
/// The shape mirrors what `model-harness` substitutes into a local profile's
/// `argv`: literal `{host}` / `{port}` tokens are replaced before exec, so the
/// runtime is always told its endpoint rather than choosing one.
public struct RuntimeOptions: Sendable, Equatable {
    /// Loopback is the only address a model-harness plan can present.
    public static let requiredHost = "127.0.0.1"

    /// Absolute path to the local model directory (weights + tokenizer).
    public let modelPath: String

    /// The canonical model ID advertised on `/v1/models` and required on
    /// `/v1/chat/completions`. Defaults to `modelPath`, matching what
    /// `mlx_lm.server` publishes for a `--model <local dir>` launch.
    public let modelID: String

    public let host: String
    public let port: Int

    /// How many prompt tokens are evaluated per prefill chunk.
    ///
    /// `nil` leaves `MLXLMCommon.GenerateParameters.prefillStepSize` at its own
    /// default of `512`.
    ///
    /// Exposed because it is not a tuning knob for this work, it is a pinnable
    /// condition. `mlx_lm.server` takes `--prefill-step-size` and defaults it
    /// to `2048`; this runtime's default is `512`. Comparing the two as shipped
    /// measures a 4x difference in chunk size and reports it as a difference
    /// between runtimes. ``RuntimeBenchmark/contextPolicy(derivedFrom:)`` reads
    /// this flag off the rendered launch and refuses a benchmark pair that left
    /// it to either default.
    public let prefillStepSize: Int?

    /// Which MLX Swift model factory this runtime is allowed to build the
    /// model with.
    ///
    /// Not cosmetic, and not a tuning knob. The two factories that both accept
    /// `model_type` `qwen3_5` implement *different prompt-evaluation
    /// strategies*, and the difference decides whether a long prompt is served
    /// or aborts the process:
    ///
    /// * `MLXLLM.LLMModelFactory` builds `MLXLLM.Qwen35Model`, whose prepare
    ///   step comes from the `LLMModel` extension and evaluates the prompt in
    ///   `windowSize ?? 512` chunks.
    /// * `MLXVLM.VLMModelFactory` builds `MLXVLM.Qwen35VLModel`, whose
    ///   `prepare(_:cache:windowSize:)` declares `windowSize _: Int?` and
    ///   discards it, evaluating the whole prompt in one call. On a 73k-token
    ///   prompt that is a single 255,904,140,288-byte attention allocation
    ///   against a 41,747,087,360-byte Metal buffer limit, and MLX traps.
    ///
    /// ``ModelFactoryPreference/textOnly`` is the default because this
    /// executable serves a text-only surface: ``ChatCompletionRequest`` refuses
    /// image and audio content parts, so a vision tower is weight this runtime
    /// loads and can never reach. `MLXLLM.Qwen35Model.sanitize` drops those
    /// weights outright. Preferring the vision factory here would mean paying
    /// for a capability the HTTP contract rejects and losing chunked prefill to
    /// buy it.
    public let modelFactory: ModelFactoryPreference

    /// Rotating KV-cache bound, `nil` for an unbounded `KVCacheSimple`.
    public let maxKVSize: Int?

    /// Applied when a request omits `max_tokens`.
    public let defaultMaxTokens: Int

    /// Chat-template `reasoning_effort` kwarg. The Qwen3.5 template accepts
    /// `low`, `medium` and `xhigh` and raises on anything else, so the value is
    /// validated here rather than at first token.
    public let reasoningEffort: String?

    /// Acceptance-suite fault seam: when set, every generation attempt fails
    /// immediately with this exact text instead of reaching MLX.
    ///
    /// It exists so the dead-generation-worker path — classification, the
    /// readiness transition, `/health`, the supervision marker and the
    /// supervised restart — can be driven end to end on a real build, on
    /// demand, without waiting for a Metal allocator to actually leak. The
    /// text is injected verbatim so the classifier sees exactly the message a
    /// real backend failure would produce, which is what makes the check
    /// meaningful rather than circular: nothing here forces a verdict, and
    /// injecting a request-scoped message must leave `/health` at `200`.
    ///
    /// Upstream `mlx-lm` proved its own generation-thread recovery the same
    /// way, by injecting a `RuntimeError` into the live generation loop.
    ///
    /// Off unless the flag is given, so a profile that does not ask for it
    /// cannot be affected by it.
    public let faultInjectedGenerationError: String?

    /// How many generation attempts the fault seam is allowed to fail, or
    /// `nil` for every attempt.
    ///
    /// `nil` is the default because it is what the dead-generation-worker
    /// suite already depends on: that regression is terminal, so an injection
    /// that healed itself after one request would be testing a different
    /// runtime than the one being pinned.
    ///
    /// A bound is what makes *recovery* observable at all. The acceptance
    /// question here is whether a request that follows a failed one succeeds
    /// on the same process, and a seam that fails every attempt can only ever
    /// answer "no" — not because the runtime failed to recover, but because
    /// the seam refused to let it.
    public let faultInjectedGenerationErrorCount: Int?

    /// How many generated tokens must reach the client before the fault seam
    /// fires. `0` fires before MLX is touched at all.
    ///
    /// `0` is the default, matching the dead-generation-worker suite: that
    /// path never reaches the weights, which is what lets it run on a 261 MB
    /// model instead of a 29 GB one.
    ///
    /// A non-zero threshold is what makes *batch* recovery observable. Failing
    /// before a `ChatSession` exists releases nothing, so a runtime that leaked
    /// every KV cache it ever built would pass a check written that way. Firing
    /// after N tokens means the batch entry, its cache, and partial output
    /// already delivered to the client all exist at the moment of failure —
    /// which is also the only way to establish that partial output is *not*
    /// then returned as a truncated success.
    public let faultInjectedGenerationErrorAfterTokens: Int

    /// Retain the condemned worker's model container across the deferred
    /// teardown, so its weights are never released.
    ///
    /// The acceptance seam for the branch review found unguarded: a teardown
    /// that never observes the release must not clear the shared pool and must
    /// not attest a rebuild. It cannot be provoked by asking the runtime to
    /// pretend — the interesting state is one where the buffers really are
    /// still held — so this arms a real strong reference on the teardown path
    /// and lets ``WeightReleaseBarrier`` genuinely time out.
    ///
    /// Off unless the flag is given, and refused unless
    /// ``faultInjectedGenerationError`` is given too: only a condemned worker
    /// has a deferred teardown to retain anything across.
    public let faultRetainWeightsOnTeardown: Bool

    /// Retain the condemned worker's *weight-owning* state — `ModelContext.model`,
    /// below the container — across the deferred teardown, while letting the
    /// container itself be deallocated normally.
    ///
    /// The seam for the interval review found in revision 3, and it is a
    /// different interval from ``faultRetainWeightsOnTeardown``. That one parks
    /// the wrapper, so its `weak` reference never reads `nil` and a runtime
    /// that answered the release question from the wrapper alone would still
    /// pass. This one lets the wrapper die on schedule and keeps the weights,
    /// which is precisely the state a wrapper-only barrier reports as a
    /// completed release: review's narrowed mutant attested a rebuild with
    /// 262,361,760 bytes still active.
    ///
    /// Like the container seam it holds for the lifetime of the process, so
    /// what the acceptance suite measures is a reference that is genuinely
    /// stuck rather than one that quietly lets go once the wait it defeats has
    /// expired.
    ///
    /// Off unless the flag is given, and refused unless
    /// ``faultInjectedGenerationError`` is given too.
    public let faultRetainWeightsBelowContainerOnTeardown: Bool

    /// Retain a *strict subset* of the condemned worker's weight-owning
    /// `Module` objects across the deferred teardown, letting the container and
    /// the rest of the model tree be deallocated normally.
    ///
    /// The seam for the class review's revision-4 finding opened up and no
    /// byte count can close. Both seams above leave the whole model resident,
    /// so an absolute-residue check alone refuses them. This one leaves *part*
    /// of it resident: `activeBytes` lands below the model's load footprint
    /// while some of this model's weights are still owned, and with a large
    /// enough request behind it the process-global `returnedBytes` comparison
    /// is satisfied too. Every byte-derived clause of the release gate is then
    /// green, and only ``GenerationBatchRecovery/WeightReleaseObservation/liveWeightOwners``
    /// — ownership read from the model tree itself — refuses.
    ///
    /// Like the other two it holds for the lifetime of the process, so what the
    /// acceptance suite measures is retention that is genuinely stuck.
    ///
    /// Off unless the flag is given, and refused unless
    /// ``faultInjectedGenerationError`` is given too.
    public let faultRetainWeightModulesOnTeardown: Bool

    /// Retain this model's weight *arrays* across the deferred teardown while
    /// letting the container and every `Module` that owned them be deallocated
    /// normally.
    ///
    /// The seam for the one clause the three above cannot isolate. They all
    /// keep some object of the model tree alive, so the ownership clause
    /// refuses them and the byte clauses are never the deciding vote. This one
    /// copies the flattened parameter arrays out of the tree and holds those:
    /// every module dies, ownership reports the model released, and MLX still
    /// calls the whole weight footprint active because the buffers those arrays
    /// reference are still referenced.
    ///
    /// It is not a hypothetical shape. Anything that caches, snapshots or
    /// exports a model's parameters produces exactly this state, and to a
    /// process-global counter it is indistinguishable from a released model
    /// unless the residue itself is checked.
    ///
    /// Off unless the flag is given, and refused unless
    /// ``faultInjectedGenerationError`` is given too.
    public let faultRetainWeightArraysOnTeardown: Bool

    /// Retain a *strict subset* of this model's weight arrays — the largest
    /// half by `nbytes` — across the deferred teardown, while letting the
    /// container and every `Module` be deallocated normally.
    ///
    /// Review's revision-5 bypass, promoted from a scratch mutant to a
    /// maintained seam. It is the exact combination none of the four seams
    /// above can produce: zero live `Module` owners (so the ownership clause
    /// is satisfied), a residue that is *significant* but strictly **below**
    /// the model's load footprint (so a footprint-relative residue check is
    /// satisfied), and — behind a large enough request — a process-global
    /// `returnedBytes` that clears the footprint as well. Under revision 5's
    /// gate every clause read green and the runtime attested a completed
    /// release with 255,724,192 B of a 262,361,760 B model still active.
    ///
    /// The maintained all-array seam above cannot cover this class, because it
    /// leaves the residue *at or above* the footprint, and the module-subset
    /// seam cannot cover it either, because keeping a `Module` alive makes
    /// ownership refuse first. Only a strict subset of raw arrays puts the
    /// residue inside the interval a footprint-relative gate admits.
    ///
    /// Off unless the flag is given, and refused unless
    /// ``faultInjectedGenerationError`` is given too.
    public let faultRetainWeightArraySubsetOnTeardown: Bool

    public init(
        modelPath: String,
        modelID: String,
        host: String,
        port: Int,
        maxKVSize: Int?,
        prefillStepSize: Int? = nil,
        modelFactory: ModelFactoryPreference = .textOnly,
        defaultMaxTokens: Int,
        reasoningEffort: String?,
        faultInjectedGenerationError: String? = nil,
        faultInjectedGenerationErrorCount: Int? = nil,
        faultInjectedGenerationErrorAfterTokens: Int = 0,
        faultRetainWeightsOnTeardown: Bool = false,
        faultRetainWeightsBelowContainerOnTeardown: Bool = false,
        faultRetainWeightModulesOnTeardown: Bool = false,
        faultRetainWeightArraysOnTeardown: Bool = false,
        faultRetainWeightArraySubsetOnTeardown: Bool = false
    ) {
        self.modelPath = modelPath
        self.modelID = modelID
        self.host = host
        self.port = port
        self.maxKVSize = maxKVSize
        self.prefillStepSize = prefillStepSize
        self.modelFactory = modelFactory
        self.defaultMaxTokens = defaultMaxTokens
        self.reasoningEffort = reasoningEffort
        self.faultInjectedGenerationError = faultInjectedGenerationError
        self.faultInjectedGenerationErrorCount = faultInjectedGenerationErrorCount
        self.faultInjectedGenerationErrorAfterTokens = faultInjectedGenerationErrorAfterTokens
        self.faultRetainWeightsOnTeardown = faultRetainWeightsOnTeardown
        self.faultRetainWeightsBelowContainerOnTeardown =
            faultRetainWeightsBelowContainerOnTeardown
        self.faultRetainWeightModulesOnTeardown = faultRetainWeightModulesOnTeardown
        self.faultRetainWeightArraysOnTeardown = faultRetainWeightArraysOnTeardown
        self.faultRetainWeightArraySubsetOnTeardown = faultRetainWeightArraySubsetOnTeardown
    }
}

public enum RuntimeOptionsError: Error, Equatable, CustomStringConvertible {
    case missingSubcommand
    case unknownSubcommand(String)
    case unknownFlag(String)
    case missingValue(String)
    case duplicateFlag(String)
    case missingRequiredFlag(String)
    case nonLoopbackHost(String)
    case invalidPort(String)
    case relativeModelPath(String)
    case invalidInteger(flag: String, value: String)
    case nonPositiveInteger(flag: String, value: Int)
    case unsupportedReasoningEffort(String)
    case unsupportedModelFactory(String)
    case emptyFaultInjection
    case negativeInteger(flag: String, value: Int)
    case faultInjectionModifierWithoutInjection(String)
    case invalidBoolean(flag: String, value: String)

    public var description: String {
        switch self {
        case .missingSubcommand:
            return "expected a subcommand: serve or preflight"
        case .unknownSubcommand(let value):
            return
                "unknown subcommand \(value.debugDescription); expected serve or preflight"
        case .unknownFlag(let flag):
            return "unknown flag \(flag.debugDescription)"
        case .missingValue(let flag):
            return "flag \(flag.debugDescription) requires a value"
        case .duplicateFlag(let flag):
            return "flag \(flag.debugDescription) was given more than once"
        case .missingRequiredFlag(let flag):
            return "flag \(flag.debugDescription) is required"
        case .nonLoopbackHost(let host):
            return
                "host must equal \(RuntimeOptions.requiredHost); refusing to bind \(host.debugDescription)"
        case .invalidPort(let value):
            return "port must be an integer between 1 and 65535, got \(value.debugDescription)"
        case .relativeModelPath(let path):
            return "--model must be an absolute path, got \(path.debugDescription)"
        case .invalidInteger(let flag, let value):
            return
                "flag \(flag.debugDescription) requires an integer, got \(value.debugDescription)"
        case .nonPositiveInteger(let flag, let value):
            return "flag \(flag.debugDescription) requires a positive integer, got \(value)"
        case .unsupportedReasoningEffort(let value):
            return
                "--reasoning-effort must be one of low, medium, xhigh; got \(value.debugDescription)"
        case .unsupportedModelFactory(let value):
            return
                "--model-factory must be one of "
                + RuntimeOptions.ModelFactoryPreference.allCases.map(\.rawValue)
                .joined(separator: ", ") + "; got \(value.debugDescription)"
        case .emptyFaultInjection:
            return "--fault-inject-generation-error requires a non-empty failure message"
        case .negativeInteger(let flag, let value):
            return "flag \(flag.debugDescription) requires a non-negative integer, got \(value)"
        case .faultInjectionModifierWithoutInjection(let flag):
            return
                "flag \(flag.debugDescription) requires --fault-inject-generation-error; "
                + "on its own it configures a fault seam that is not armed"
        case .invalidBoolean(let flag, let value):
            return "\(flag) expects true or false, got \(value.debugDescription)"
        }
    }
}

extension RuntimeOptions {
    /// Chat-template efforts the bundled Qwen3.5 template accepts. Anything else
    /// makes the template raise mid-request, so it is refused at startup.
    public static let supportedReasoningEfforts: Set<String> = ["low", "medium", "xhigh"]

    /// What the process was asked to do.
    public enum Subcommand: String, Sendable, Equatable {
        /// Load the model and serve the OpenAI-compatible endpoints.
        case serve
        /// Check configuration, tokenizer and chat template without loading weights.
        case preflight
    }

    /// Which factories ``ModelLoader`` may try, and in which order.
    ///
    /// Spelled as an ordered preference rather than as a single factory name so
    /// a directory the preferred factory refuses still has somewhere to go: an
    /// architecture that only one of the two registers must still load. What
    /// the option fixes is which implementation gets *first* refusal, and
    /// therefore which one serves a model both accept.
    public enum ModelFactoryPreference: String, Sendable, Equatable, CaseIterable, Codable {
        /// `MLXLLM.LLMModelFactory` first, `MLXVLM.VLMModelFactory` as
        /// fallback. The default, and the only order whose prompt evaluation
        /// is chunked for this model.
        case textOnly = "text-only"
        /// `MLXVLM.VLMModelFactory` first, `MLXLLM.LLMModelFactory` as
        /// fallback. Retained because it is the order every measurement before
        /// this option was taken under, so a comparison against those numbers
        /// is still expressible.
        case visionFirst = "vision-first"
        /// `MLXLLM.LLMModelFactory` only. No fallback: a directory the text
        /// factory refuses fails the load rather than quietly becoming a vision
        /// model, which is what makes "this model served text-only" checkable
        /// instead of merely likely.
        case textOnlyStrict = "text-only-strict"

        /// Fully-qualified factory type names, in the order `ModelLoader` will
        /// try them.
        ///
        /// Lives here rather than beside the loader so the order is checkable
        /// without linking MLX or loading 28 GB of weights. The order *is* the
        /// behaviour of this option — for `model_type` `qwen3_5` it decides
        /// between chunked and unchunked prompt evaluation — and a property
        /// that could only be observed by loading the model would be one this
        /// suite never observes.
        public var factoryOrder: [String] {
            switch self {
            case .textOnly:
                return ["MLXLLM.LLMModelFactory", "MLXVLM.VLMModelFactory"]
            case .visionFirst:
                return ["MLXVLM.VLMModelFactory", "MLXLLM.LLMModelFactory"]
            case .textOnlyStrict:
                return ["MLXLLM.LLMModelFactory"]
            }
        }
    }

    /// Parse arguments. Pure: performs no filesystem or network access.
    ///
    /// - Parameter arguments: argv without the executable name.
    public static func parse(arguments: [String]) throws -> (Subcommand, RuntimeOptions) {
        guard let raw = arguments.first else {
            throw RuntimeOptionsError.missingSubcommand
        }
        guard let subcommand = Subcommand(rawValue: raw) else {
            throw RuntimeOptionsError.unknownSubcommand(raw)
        }

        var values: [String: String] = [:]
        var index = 1
        let known: Set<String> = [
            "--model", "--model-id", "--host", "--port", "--max-kv-size", "--prefill-step-size",
            "--model-factory",
            "--default-max-tokens",
            "--reasoning-effort", "--fault-inject-generation-error",
            "--fault-inject-generation-error-count",
            "--fault-inject-generation-error-after-tokens",
            "--fault-inject-teardown-retain",
            "--fault-inject-teardown-retain-weights",
            "--fault-inject-teardown-retain-weight-modules",
            "--fault-inject-teardown-retain-weight-arrays",
            "--fault-inject-teardown-retain-weight-array-subset",
        ]
        while index < arguments.count {
            let flag = arguments[index]
            guard known.contains(flag) else {
                throw RuntimeOptionsError.unknownFlag(flag)
            }
            guard index + 1 < arguments.count else {
                throw RuntimeOptionsError.missingValue(flag)
            }
            guard values[flag] == nil else {
                throw RuntimeOptionsError.duplicateFlag(flag)
            }
            values[flag] = arguments[index + 1]
            index += 2
        }

        guard let modelPath = values["--model"] else {
            throw RuntimeOptionsError.missingRequiredFlag("--model")
        }
        guard modelPath.hasPrefix("/") else {
            throw RuntimeOptionsError.relativeModelPath(modelPath)
        }

        let host = values["--host"] ?? requiredHost
        guard host == requiredHost else {
            throw RuntimeOptionsError.nonLoopbackHost(host)
        }

        // `preflight` binds nothing, so it does not require a port; when one is
        // given it is still validated so the same argv works for both.
        let portText = values["--port"]
        if subcommand == .serve, portText == nil {
            throw RuntimeOptionsError.missingRequiredFlag("--port")
        }
        var port = 0
        if let portText {
            guard let parsed = Int(portText), (1 ... 65535).contains(parsed) else {
                throw RuntimeOptionsError.invalidPort(portText)
            }
            port = parsed
        }

        let maxKVSize = try values["--max-kv-size"].map {
            try positiveInteger(flag: "--max-kv-size", value: $0)
        }
        let prefillStepSize = try values["--prefill-step-size"].map {
            try positiveInteger(flag: "--prefill-step-size", value: $0)
        }

        let modelFactory: ModelFactoryPreference
        if let raw = values["--model-factory"] {
            guard let parsed = ModelFactoryPreference(rawValue: raw) else {
                throw RuntimeOptionsError.unsupportedModelFactory(raw)
            }
            modelFactory = parsed
        } else {
            modelFactory = .textOnly
        }

        let defaultMaxTokens =
            try values["--default-max-tokens"].map {
                try positiveInteger(flag: "--default-max-tokens", value: $0)
            } ?? 2048

        if let effort = values["--reasoning-effort"], !supportedReasoningEfforts.contains(effort) {
            throw RuntimeOptionsError.unsupportedReasoningEffort(effort)
        }

        // An empty injection would arm the seam with a message that matches no
        // signature, producing a runtime that fails every request while
        // reporting itself healthy — the precise shape this work exists to
        // remove. Refused at parse time rather than discovered at first request.
        if let injected = values["--fault-inject-generation-error"], injected.isEmpty {
            throw RuntimeOptionsError.emptyFaultInjection
        }

        // Both modifiers shape a seam they cannot arm. Given without
        // `--fault-inject-generation-error` they would be accepted and do
        // nothing, so an acceptance run that meant to bound its injection and
        // mistyped the message flag would observe a runtime that never failed
        // and read it as recovery. Refused at parse time, like the empty
        // message above.
        for modifier in [
            "--fault-inject-generation-error-count",
            "--fault-inject-generation-error-after-tokens",
            "--fault-inject-teardown-retain",
            "--fault-inject-teardown-retain-weights",
            "--fault-inject-teardown-retain-weight-modules",
            "--fault-inject-teardown-retain-weight-arrays",
            "--fault-inject-teardown-retain-weight-array-subset",
        ] where values[modifier] != nil && values["--fault-inject-generation-error"] == nil {
            throw RuntimeOptionsError.faultInjectionModifierWithoutInjection(modifier)
        }

        // Positive: a count of zero is a seam that is armed and disarmed at the
        // same time. Whatever it was meant to say, it is not what it says.
        let faultCount = try values["--fault-inject-generation-error-count"].map {
            try positiveInteger(flag: "--fault-inject-generation-error-count", value: $0)
        }
        // Non-negative: zero is meaningful here and is the default -- fire
        // before MLX is touched.
        let faultAfterTokens =
            try values["--fault-inject-generation-error-after-tokens"].map {
                try nonNegativeInteger(
                    flag: "--fault-inject-generation-error-after-tokens", value: $0)
            } ?? 0

        // Spelled with an explicit value rather than as a bare switch so the
        // parser stays one shape -- every flag here is `--flag value` -- and so
        // a profile can carry `false` explicitly instead of expressing "off" by
        // deleting a line.
        let retainWeights =
            try values["--fault-inject-teardown-retain"].map {
                try boolean(flag: "--fault-inject-teardown-retain", value: $0)
            } ?? false
        // Deliberately independent of the flag above rather than a mode of it.
        // The two park different objects and drive different halves of the
        // release gate, and a single tri-state would let an acceptance run
        // silently swap one negative for the other.
        let retainWeightsBelowContainer =
            try values["--fault-inject-teardown-retain-weights"].map {
                try boolean(flag: "--fault-inject-teardown-retain-weights", value: $0)
            } ?? false
        // A third independent seam rather than a mode of the second. It parks a
        // strict subset of the model tree, which is the only one of the three
        // that leaves the allocator reporting *less* than the model's footprint
        // while this model's weights are still owned.
        let retainWeightModules =
            try values["--fault-inject-teardown-retain-weight-modules"].map {
                try boolean(flag: "--fault-inject-teardown-retain-weight-modules", value: $0)
            } ?? false
        // A fourth independent seam. It holds no object of the model tree at
        // all, so ownership reports the model released and the absolute-residue
        // clause is the only thing left to refuse.
        let retainWeightArrays =
            try values["--fault-inject-teardown-retain-weight-arrays"].map {
                try boolean(flag: "--fault-inject-teardown-retain-weight-arrays", value: $0)
            } ?? false
        // A fifth independent seam, and review's revision-5 bypass kept as a
        // maintained input rather than as a scratch mutant. It holds the
        // largest half of the parameter arrays by `nbytes`, so ownership
        // reports the model released AND the residue lands strictly below the
        // model's footprint -- the interval every footprint-relative residue
        // check admits.
        let retainWeightArraySubset =
            try values["--fault-inject-teardown-retain-weight-array-subset"].map {
                try boolean(
                    flag: "--fault-inject-teardown-retain-weight-array-subset", value: $0)
            } ?? false

        return (
            subcommand,
            RuntimeOptions(
                modelPath: modelPath,
                modelID: values["--model-id"] ?? modelPath,
                host: host,
                port: port,
                maxKVSize: maxKVSize,
                prefillStepSize: prefillStepSize,
                modelFactory: modelFactory,
                defaultMaxTokens: defaultMaxTokens,
                reasoningEffort: values["--reasoning-effort"],
                faultInjectedGenerationError: values["--fault-inject-generation-error"],
                faultInjectedGenerationErrorCount: faultCount,
                faultInjectedGenerationErrorAfterTokens: faultAfterTokens,
                faultRetainWeightsOnTeardown: retainWeights,
                faultRetainWeightsBelowContainerOnTeardown: retainWeightsBelowContainer,
                faultRetainWeightModulesOnTeardown: retainWeightModules,
                faultRetainWeightArraysOnTeardown: retainWeightArrays,
                faultRetainWeightArraySubsetOnTeardown: retainWeightArraySubset)
        )
    }

    private static func boolean(flag: String, value: String) throws -> Bool {
        switch value {
        case "true": return true
        case "false": return false
        default: throw RuntimeOptionsError.invalidBoolean(flag: flag, value: value)
        }
    }

    private static func nonNegativeInteger(flag: String, value: String) throws -> Int {
        guard let parsed = Int(value) else {
            throw RuntimeOptionsError.invalidInteger(flag: flag, value: value)
        }
        guard parsed >= 0 else {
            throw RuntimeOptionsError.negativeInteger(flag: flag, value: parsed)
        }
        return parsed
    }

    private static func positiveInteger(flag: String, value: String) throws -> Int {
        guard let parsed = Int(value) else {
            throw RuntimeOptionsError.invalidInteger(flag: flag, value: value)
        }
        guard parsed > 0 else {
            throw RuntimeOptionsError.nonPositiveInteger(flag: flag, value: parsed)
        }
        return parsed
    }
}
