import Foundation
import MLXSwiftRuntimeContract

/// What the HTTP layer should send back.
enum RouterOutcome: Sendable {
    case json(status: Int, body: JSONValue)
    /// A streaming chat completion; the HTTP layer sends SSE headers and then
    /// pumps `run` with a writer.
    case stream(AdmittedChatCompletion, GenerationEngine, ChatCompletionResponseBuilder)
}

/// Maps HTTP requests onto runtime behaviour.
///
/// This is the production call site for every admission gate in
/// `MLXSwiftRuntimeContract`:
/// - ``ModelsListing/make(modelID:readiness:created:)`` for `GET /v1/models`
/// - ``ChatCompletionAdmission/admit(_:configuration:)`` for
///   `POST /v1/chat/completions`, called before any model work
struct Router: Sendable {
    let options: RuntimeOptions
    let state: RuntimeState
    let ledger: GenerationBatchLedgerStore
    let created: Int
    let systemFingerprint: String

    var admissionConfiguration: ChatCompletionAdmission.Configuration {
        // Mirrors the Pi profile's `compat` table: this profile declares
        // `supports_developer_role = false` and `supports_reasoning_effort =
        // false`, so both are refused rather than accepted and ignored.
        ChatCompletionAdmission.Configuration(
            modelID: options.modelID,
            defaultMaxTokens: options.defaultMaxTokens,
            supportsDeveloperRole: false,
            supportsReasoningEffort: false)
    }

    func route(method: String, path: String, body: Data) async -> RouterOutcome {
        let route = path.split(separator: "?", maxSplits: 1).first.map(String.init) ?? path
        switch (method, route) {
        case ("GET", "/health"):
            return await health()
        case ("GET", "/v1/models"):
            let listing = ModelsListing.make(
                modelID: options.modelID,
                readiness: await state.currentReadiness(),
                created: created)
            return .json(status: listing.status, body: listing.body)
        case ("POST", "/v1/chat/completions"):
            return await chatCompletions(body: body)
        case ("GET", "/debug/generation-state"):
            return await generationState()
        default:
            return .json(
                status: 404,
                body: ChatCompletionResponseBuilder.error(
                    message: "unknown route \(method) \(route)",
                    type: "invalid_request_error", code: "not_found"))
        }
    }

    private func health() async -> RouterOutcome {
        let report = HealthReport.make(readiness: await state.currentReadiness())
        return .json(status: report.status, body: report.body)
    }

    /// Production call site for ``GenerationBatchReport``.
    ///
    /// Read-only, loopback-only, and separate from `/health` on purpose:
    /// `/health` answers "should you send me traffic", which is a question a
    /// condemned runtime answers `503`, and this answers "what are you still
    /// holding", which a condemned runtime has to be able to answer at all.
    private func generationState() async -> RouterOutcome {
        // Allocator figures only once MLX has actually been driven. Before
        // that they are not a low reading, they are no reading, and the report
        // omits them rather than publishing zeros somebody would size a host
        // from.
        let memory = await state.hasInitializedBackend() ? await ledger.memoryUsage() : nil
        let report = GenerationBatchReport.make(
            readiness: await state.currentReadiness(),
            ledger: await ledger.snapshot(),
            memory: memory)
        return .json(status: report.status, body: report.body)
    }

    /// Report a generation failure to the runtime's health state.
    ///
    /// Production call site for ``GenerationWorkerHealth``. Called from both
    /// completion paths — the buffered one below and the SSE one in
    /// `RuntimeHTTPHandler.sendStream` — because a worker condemned mid-stream
    /// is exactly as dead as one condemned mid-request, and a health endpoint
    /// that only noticed the buffered case would still answer `200` for the
    /// runtime that Pi actually streams from.
    func recordGenerationFailure(_ error: any Error) async {
        let description = String(describing: error)
        guard await state.recordGenerationFailure(description) else { return }
        // Written to stdout as a normal runtime event *and* carrying the
        // literal supervision marker, because `model-harness` matches fatal
        // substrings against the forwarded stream rather than parsing it.
        StandardOutput.shared.event(
            RuntimeEvent(
                name: "generation_worker_failed",
                fields: [
                    "marker": .string(GenerationWorkerHealth.supervisionMarker),
                    "detail": .string(description),
                ]))
        StandardOutput.shared.log(
            "\(GenerationWorkerHealth.supervisionMarker): \(description)")
    }

    private func chatCompletions(body: Data) async -> RouterOutcome {
        // Readiness is checked before parsing: answering a completion while the
        // weights are still loading would contradict the 503 that `/v1/models`
        // is returning at the same moment.
        guard let engine = await state.currentEngine() else {
            let readiness = await state.currentReadiness()
            let detail: String
            switch readiness {
            case .loading: detail = "model is still loading"
            case .shuttingDown: detail = "runtime is shutting down"
            case .failed(let reason): detail = "model load failed: \(reason)"
            case .generationWorkerFailed(let reason):
                detail = "generation worker is unavailable: \(reason)"
            case .ready: detail = "model is ready but no engine is installed"
            }
            return .json(
                status: 503,
                body: ChatCompletionResponseBuilder.error(
                    message: detail, type: "server_error", code: "model_not_ready"))
        }

        let request: ChatCompletionRequest
        do {
            request = try ChatCompletionRequest.decode(from: body)
        } catch {
            return .json(
                status: 400,
                body: ChatCompletionResponseBuilder.error(
                    message: String(describing: error),
                    type: "invalid_request_error", code: "invalid_body"))
        }

        let admitted: AdmittedChatCompletion
        do {
            admitted = try ChatCompletionAdmission.admit(
                request, configuration: admissionConfiguration)
        } catch let refusal as ChatCompletionRefusal {
            return .json(
                status: refusal.httpStatus,
                body: ChatCompletionResponseBuilder.error(
                    message: refusal.description, type: refusal.errorType,
                    code: refusal.errorCode))
        } catch {
            return .json(
                status: 400,
                body: ChatCompletionResponseBuilder.error(
                    message: String(describing: error),
                    type: "invalid_request_error", code: "invalid_body"))
        }

        let builder = ChatCompletionResponseBuilder(
            requestID: "chatcmpl-\(UUID().uuidString)",
            modelID: admitted.modelID,
            created: Int(Date().timeIntervalSince1970),
            systemFingerprint: systemFingerprint)

        if admitted.stream {
            return .stream(admitted, engine, builder)
        }
        return await complete(admitted, engine: engine, builder: builder)
    }

    private func complete(
        _ request: AdmittedChatCompletion,
        engine: GenerationEngine,
        builder: ChatCompletionResponseBuilder
    ) async -> RouterOutcome {
        let accumulator = CompletionAccumulator()
        do {
            try await engine.generate(request) { event in
                await accumulator.apply(event)
            }
        } catch {
            await recordGenerationFailure(error)
            return .json(
                status: 500,
                body: ChatCompletionResponseBuilder.error(
                    message: String(describing: error), type: "server_error",
                    code: "generation_failed"))
        }
        guard let usage = await accumulator.usage, let finish = await accumulator.finishReason
        else {
            return .json(
                status: 500,
                body: ChatCompletionResponseBuilder.error(
                    message: String(describing: GenerationError.missingCompletionInfo),
                    type: "server_error", code: "generation_failed"))
        }
        return .json(
            status: 200,
            body: builder.completion(
                content: await accumulator.content,
                reasoning: await accumulator.reasoning,
                toolCalls: await accumulator.toolCalls,
                finishReason: finish,
                usage: usage))
    }
}

/// Collects a full generation for the non-streaming response.
actor CompletionAccumulator {
    private(set) var content = ""
    private(set) var reasoning = ""
    private(set) var toolCalls: [ToolCallPayload] = []
    private(set) var usage: CompletionUsage?
    private(set) var finishReason: FinishReason?

    func apply(_ event: GenerationEvent) {
        switch event {
        case .delta(let reasoning, let content):
            self.reasoning += reasoning
            self.content += content
        case .toolCall(let call):
            toolCalls.append(call)
        case .completed(let finishReason, let usage):
            self.finishReason = finishReason
            self.usage = usage
        }
    }
}
