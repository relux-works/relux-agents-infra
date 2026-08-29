import Foundation
import MLX
import MLXLMCommon
import MLXSwiftRuntimeContract

enum GenerationEvent: Sendable {
    case delta(reasoning: String, content: String)
    case toolCall(ToolCallPayload)
    case completed(finishReason: FinishReason, usage: CompletionUsage)
}

enum GenerationError: Error, CustomStringConvertible {
    /// Generation ended without the completion-info packet that carries the
    /// token counts. Reported instead of substituting zeros: a fabricated usage
    /// block would be read downstream as a real measurement.
    case missingCompletionInfo
    case cancelled
    /// Raised by the `--fault-inject-generation-error` acceptance seam.
    case injected(String)
    /// An error MLX's C++ layer reported during this generation, verbatim.
    ///
    /// Exists because without it the message never becomes a Swift error at
    /// all. `MLX/ErrorHandler.swift` calls `fatalError` when no handler is
    /// installed, so a real allocator failure killed the process outright and
    /// every contract downstream of a *thrown* generation failure --
    /// classification, the batch ledger, `/health`, the supervision marker --
    /// was unreachable for exactly the failures they exist for.
    case mlx(String)

    var description: String {
        switch self {
        case .missingCompletionInfo:
            return "generation ended without completion info; token usage is unknown"
        case .cancelled:
            return "generation was cancelled before it finished"
        case .injected(let message):
            // Verbatim, with no wrapper text: the classifier must see exactly
            // what a real backend failure would report, or the injected run
            // would be testing a different string than production does.
            return message
        case .mlx(let message):
            // Also verbatim, and for the same reason: this is the real backend
            // failure the injected seam imitates, and the classifier has to see
            // the same bytes from both.
            return message
        }
    }
}

/// Runs one generation at a time against the single loaded model.
///
/// Serialization is deliberate. Each concurrent `ChatSession` allocates its own
/// KV cache, and two 75k-context caches beside a 27B 8-bit model do not fit in
/// the host's working set. Concurrency is a named limitation of this prototype,
/// not an oversight.
actor GenerationEngine {
    private let model: LoadedModel
    private let options: RuntimeOptions
    private let ledger: GenerationBatchLedgerStore

    /// Fault-seam budget left, or `nil` when the seam fires on every attempt.
    ///
    /// Engine state rather than a re-read of `options`, because it is consumed.
    /// It is decremented at the moment the seam actually throws, never when a
    /// generation merely *becomes eligible* to be failed: a threshold set higher
    /// than the request's `max_tokens` must leave the budget intact and let that
    /// request succeed, so an acceptance run that mis-sizes its injection
    /// reports a `200` where it demanded a `500` instead of silently spending
    /// the injection on nothing.
    private var faultInjectionsRemaining: Int?

    init(model: LoadedModel, options: RuntimeOptions, ledger: GenerationBatchLedgerStore) {
        self.model = model
        self.options = options
        self.ledger = ledger
        self.faultInjectionsRemaining = options.faultInjectedGenerationErrorCount
    }

    /// Take one unit of fault-seam budget, if the seam is armed and has any.
    ///
    /// - Returns: the message to throw, or `nil` when the seam is not armed or
    ///   is spent. Spending the budget is what lets the *next* request through
    ///   on the same process, which is the property the recovery suite exists
    ///   to establish.
    private func consumeFaultInjection() -> String? {
        guard let message = options.faultInjectedGenerationError else { return nil }
        guard let remaining = faultInjectionsRemaining else { return message }
        guard remaining > 0 else { return nil }
        faultInjectionsRemaining = remaining - 1
        return message
    }

    /// Polls per teardown attempt, and the interval between them. 100 x 20 ms
    /// bounds one attempt at ~2 s; ``GenerationBatchRecovery/workerTeardownAttempts``
    /// bounds the whole wait. Bounded because this runs on a `deinit`-scheduled
    /// task in a process the supervisor is already replacing.
    private static let teardownPolls = 100
    private static let teardownPollInterval: UInt64 = 20_000_000

    /// Discharge a condemned worker's deferred pool rebuild, once the engine —
    /// and with it every weight buffer — is actually gone.
    ///
    /// This is the only correct trigger, and finding that out cost a review
    /// cycle. Clearing the pool from the failing request returns nothing: the
    /// request still holds the engine, so MLX reported 303,782,980 *active*
    /// bytes at the moment of the clear and the model landed in the pool
    /// immediately afterwards, leaving a condemned runtime sitting on it.
    /// Clearing it after the response is written was no better, for the same
    /// reason. Only the engine's own deallocation orders this correctly.
    ///
    /// The work is handed to a `Task` rather than done inline because a
    /// `deinit` body runs *before* stored properties are released: the model is
    /// still alive on this line, which is precisely why ``WeightReleaseBarrier``
    /// can take a `weak` reference to the container here and then watch it go.
    ///
    /// What the task waits for is a reading that
    /// ``GenerationBatchRecovery/weightsReleased(_:)`` accepts, and the shape
    /// of that reading is the history of this review. The weak reference alone
    /// was revision 3, and it is not enough — `ModelContainer` is a wrapper,
    /// and review's narrowed mutant showed it reaching `nil` while the
    /// `ModelContext` holding the weights was still being destroyed. The weak
    /// reference plus a process-global byte *delta* was revision 4, and that is
    /// not enough either — review made a 6,000-word prompt's KV state larger
    /// than the model, so releasing the request alone satisfied the
    /// subtraction while every weight stayed resident. Comparing the residue
    /// against the *footprint* was revision 5, and review put a strict subset
    /// of the model's copied parameter arrays inside that interval with every
    /// `Module` dead. What the barrier now carries is ownership read from the
    /// model tree itself, a residue measured against a fixed allowance of
    /// ZERO, and the idle-and-at-rest conditions that make either meaningful.
    /// If those facts never arrive the
    /// teardown fails closed: no clear, no attestation, and a supervision
    /// marker, because a condemned model still holding its buffers must not be
    /// left competing with its replacement for the host.
    ///
    /// Correct ordering is not assumed here — the acceptance suite's ORDERING
    /// GATE reads MLX's own `cache_bytes` after condemnation and fails if the
    /// pool was left holding the model, and its TEARDOWN GATE drives the
    /// unobserved-release path from three sides:
    /// `--fault-inject-teardown-retain` parks the wrapper so `weak`-`nil` never
    /// happens, `--fault-inject-teardown-retain-weights` parks the whole model
    /// below a wrapper that *does* die, and
    /// `--fault-inject-teardown-retain-weight-modules` parks a strict subset so
    /// every byte-derived clause reads green and only ownership refuses. Two
    /// further seams attack the residue clause from the side ownership cannot
    /// see: `--fault-inject-teardown-retain-weight-arrays` copies every
    /// parameter array out of a tree that then dies completely, and
    /// `--fault-inject-teardown-retain-weight-array-subset` narrows that to the
    /// largest half, which is review's revision-5 bypass. All of them fail if
    /// the runtime attests a rebuild it did not perform.
    deinit {
        let ledger = self.ledger
        let barrier = WeightReleaseBarrier(
            observing: model.container,
            owners: model.weightOwners,
            weightFootprintBytes: model.weightFootprintBytes,
            // Re-read on every poll rather than captured. A reading taken while
            // another generation is allocating cannot attribute the bytes it
            // sees move, and the gate vetoes on it.
            generationsInFlight: { await ledger.generationsInFlight() })
        // The acceptance seam for the unobserved-release branch. It parks the
        // real container for the lifetime of the process, so the release
        // genuinely never happens and the barrier genuinely never observes one
        // — the runtime is not told to report a timeout, it is put in the state
        // that produces one, and stays there while the suite measures it.
        if options.faultRetainWeightsOnTeardown {
            RetainedWeights.shared.hold(model.container)
        }
        let attempts = GenerationBatchRecovery.workerTeardownAttempts
        let polls = Self.teardownPolls
        let interval = Self.teardownPollInterval
        Task {
            guard await ledger.hasPendingWorkerTeardown() else { return }
            for attempt in 1 ... attempts {
                let observation = await barrier.waitForRelease(
                    polls: polls, intervalNanoseconds: interval)
                let outcome = await ledger.completeWorkerTeardown(
                    observation: observation, attempt: attempt, maxAttempts: attempts)
                if outcome != .retry { break }
            }
        }
    }

    var modelType: String { model.modelType }
    var factory: String { model.factory }

    /// Run one generation, accounted for in the shared ledger.
    ///
    /// The `do`/`catch` is the production call site for
    /// ``GenerationBatchRecovery``. Every exit from `run` closes the slot
    /// exactly once — the failing path through ``GenerationBatchLedgerStore/fail(_:observing:)``,
    /// which also returns MLX's shared buffer pool when the failure implicates
    /// it, and the succeeding path through `finish`. The error is rethrown
    /// unchanged: recovering the runtime's own state must not turn a failed
    /// generation into a partial answer, so the caller still gets the failure
    /// and still reports it as one.
    func generate(
        _ request: AdmittedChatCompletion,
        emit: @Sendable (GenerationEvent) async throws -> Void
    ) async throws {
        let slot = await ledger.begin()
        do {
            try await run(request, emit: emit)
        } catch {
            await ledger.fail(slot, observing: String(describing: error))
            throw error
        }
        await ledger.finish(slot)
    }

    /// Run one generation with MLX's C++ errors scoped into Swift `throws`.
    ///
    /// The wrapper is the production call site for MLX error handling, and it
    /// is not defensive decoration. With no handler installed,
    /// `MLX/ErrorHandler.swift` dispatches to `fatalError`, so the first real
    /// allocator failure took the process down before ``GenerationEngine/generate``
    /// could reach its `catch`. Everything the two accepted supervision
    /// contracts specify -- `generation_worker_failed`, the dead-worker
    /// readiness transition, `/health` 503, the batch release -- is downstream
    /// of that `catch`, and therefore did not run for the one class of failure
    /// those contracts exist to survive.
    ///
    /// ``MLX/ErrorBox`` is checked after every streamed item rather than only
    /// on exit. MLX's C++ does not unwind: the handler is called, it returns,
    /// and the failed operation yields an unusable array that the next
    /// operation consumes. Checking per item stops that chain at the first
    /// reported error instead of generating from corrupt state until something
    /// else notices.
    ///
    /// What it cannot cover is stated rather than assumed: the handler is a
    /// task local, so an error raised on a thread MLX owns -- an `asyncEval`
    /// worker with no task context -- still reaches the global default and
    /// still traps. ``GenerationEngine`` cannot install a process-global
    /// handler that returns, because returning is what leaves the C++ side
    /// running on state it already declared invalid.
    private func run(
        _ request: AdmittedChatCompletion,
        emit: @Sendable (GenerationEvent) async throws -> Void
    ) async throws {
        // `@Sendable` and an explicit `await` hop: the scope has to be entered
        // from outside this actor's isolation, because the task local it
        // installs must also be visible to the `Task` `ChatSession.streamMap`
        // creates for the generation itself.
        try await MLX.withError { @Sendable mlxError in
            try await self.generateUnguarded(request, emit: emit, mlxError: mlxError)
        }
    }

    private func generateUnguarded(
        _ request: AdmittedChatCompletion,
        emit: @Sendable (GenerationEvent) async throws -> Void,
        mlxError: MLX.ErrorBox
    ) async throws {
        let injectAfterTokens = options.faultInjectedGenerationErrorAfterTokens

        // With no token threshold the seam fires before any MLX call, so an
        // injected run exercises the failure path without touching the GPU and
        // without depending on how far into generation the real fault would
        // have got. This is the dead-generation-worker suite's path and its
        // behaviour is unchanged.
        if injectAfterTokens == 0, let injected = consumeFaultInjection() {
            throw GenerationError.injected(injected)
        }

        var parameters = GenerateParameters(
            maxTokens: request.maxTokens, maxKVSize: options.maxKVSize)
        // Assigned only when the launch asked for it, so an unset flag leaves
        // `GenerateParameters` on its own default rather than on one this
        // runtime invented. What the benchmark pins is the *stated* value; an
        // unstated one is refused by the comparison gate, not defaulted here.
        if let prefillStepSize = options.prefillStepSize {
            parameters.prefillStepSize = prefillStepSize
        }
        if let temperature = request.temperature {
            parameters.temperature = Float(temperature)
        }
        if let topP = request.topP {
            parameters.topP = Float(topP)
        }
        if let seed = request.seed {
            parameters.seed = seed
        }

        var additionalContext: [String: any Sendable] = [:]
        if let effort = options.reasoningEffort {
            // The Qwen3.5 chat template reads `reasoning_effort` as a template
            // kwarg; there is no OpenAI request field wired to it, which is why
            // `reasoning_effort` on a request is refused rather than forwarded.
            additionalContext["reasoning_effort"] = effort
        }

        let session = ChatSession(
            model.container,
            generateParameters: parameters,
            additionalContext: additionalContext.isEmpty ? nil : additionalContext,
            tools: ContractBridge.toolSpecs(request.tools),
            toolDispatch: nil)

        var splitter = ReasoningSplitter(mode: .startsInReasoning)
        var generatedTokens = 0
        var toolCallIndex = 0
        var sawToolCall = false
        var usage: CompletionUsage?
        var finishReason = FinishReason.stop

        for try await item in session.streamDetails(
            to: ContractBridge.chatMessages(request.messages))
        {
            try Self.rethrowMLX(mlxError)
            switch item {
            case .chunk(let text):
                let split = splitter.consume(text)
                if !split.isEmpty {
                    try await emit(.delta(reasoning: split.reasoning, content: split.content))
                }
                generatedTokens += 1
                // Fired *after* the chunk is emitted, on purpose. By this point
                // the session's KV cache holds real state and the client has
                // already received real output -- which is what makes both
                // halves of the recovery contract observable: the batch has
                // something to release, and the partial output must still not
                // come back as a truncated success.
                if injectAfterTokens > 0, generatedTokens >= injectAfterTokens,
                    let injected = consumeFaultInjection()
                {
                    throw GenerationError.injected(injected)
                }
            case .toolCall(let call):
                sawToolCall = true
                let payload = ContractBridge.toolCallPayload(
                    call, fallbackID: "call_\(UUID().uuidString)"
                ).indexed(toolCallIndex)
                toolCallIndex += 1
                try await emit(.toolCall(payload))
            case .info(let info):
                usage = CompletionUsage(
                    promptTokens: info.promptTokenCount,
                    completionTokens: info.generationTokenCount)
                switch info.stopReason {
                case .length: finishReason = .length
                case .stop: finishReason = .stop
                case .cancelled: throw GenerationError.cancelled
                }
            }
        }

        let tail = splitter.flush()
        if !tail.isEmpty {
            try await emit(.delta(reasoning: tail.reasoning, content: tail.content))
        }

        if sawToolCall, finishReason == .stop {
            finishReason = .toolCalls
        }

        // Before `missingCompletionInfo`, not after. A generation MLX killed
        // mid-prefill also ends without a completion-info packet, and reporting
        // the missing packet would name the symptom this runtime observed
        // instead of the allocator failure that caused it.
        try Self.rethrowMLX(mlxError)

        guard let usage else {
            throw GenerationError.missingCompletionInfo
        }
        try await emit(.completed(finishReason: finishReason, usage: usage))
    }

    /// Turn whatever MLX recorded in this scope into a `GenerationError`.
    ///
    /// The message is carried verbatim so the classifier sees the same bytes it
    /// would see from the fault seam, and `MLXError.caught`'s own prefix is
    /// stripped for the same reason.
    private static func rethrowMLX(_ box: MLX.ErrorBox) throws {
        guard let error = box.firstError else { return }
        if case MLX.MLXError.caught(let message) = error {
            throw GenerationError.mlx(message)
        }
        throw GenerationError.mlx(String(describing: error))
    }
}
