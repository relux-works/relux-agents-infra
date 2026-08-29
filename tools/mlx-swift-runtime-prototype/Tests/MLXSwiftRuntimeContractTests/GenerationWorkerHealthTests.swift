import Foundation
import Testing

@testable import MLXSwiftRuntimeContract

/// The `mlx_lm.server` dead-generation-thread regression, carried over.
///
/// The bug being prevented is a single sentence: a runtime that cannot produce
/// a token answered `GET /health` with `200 {"status": "ok"}`, so every caller
/// found out through a request timeout instead. Every test here fails if that
/// becomes possible again, and the negative half fails if the cure is worse
/// than the disease and a bad request can take a healthy runtime down.
@Suite("generation worker health")
struct GenerationWorkerHealthTests {
    /// Verbatim from the `BUG-260827-1jhv2g` incident record.
    static let incidentFailure = "RuntimeError: [metal::malloc] Resource limit (499000) exceeded"

    // MARK: - classification

    @Test(
        "a condemned backend is classified as worker invalidation",
        arguments: [
            GenerationWorkerHealthTests.incidentFailure,
            "[metal::malloc] Resource limit (499000) exceeded",
            "MLXError: Failed to load the default metallib after 4 lookups",
        ])
    func condemnsBackendFailures(description: String) {
        guard case .workerInvalidated = GenerationWorkerHealth.classify(failure: description)
        else {
            Issue.record("\(description) was not classified as worker invalidation")
            return
        }
    }

    /// A request-scoped failure that reuses a fragment of a backend message.
    ///
    /// The over-broad class. Found by review on revision 1, where
    /// `invalidatingSignatures` listed the bare phrase `Resource limit`
    /// independently: this exact message condemned a healthy runtime through
    /// the production entry point, flipped `/health` and `/v1/models` to 503,
    /// and emitted the fatal supervision marker.
    static let overBroadNeighbour = "RequestError: Resource limit for this request is 8 tokens"

    /// MLX's *other* Metal allocator throw
    /// (`mlx/backend/metal/allocator.cpp:111-117`). It rejects one oversized
    /// allocation and leaves the pool intact, so it belongs to the request
    /// that asked for it — carrying the allocator tag is not on its own proof
    /// that the backend is gone.
    static let oversizedAllocation =
        "[metal::malloc] Attempting to allocate 91268055040 bytes which is greater than"
        + " the maximum allowed buffer size of 68719476736 bytes."

    /// The negative half of the gate.
    ///
    /// A classifier that condemned every generation error would pass all the
    /// positive tests above and still be wrong: a single malformed request
    /// would take a healthy runtime out of rotation and, with supervision
    /// configured, restart it. These are the failures that must *not* move
    /// readiness.
    ///
    /// The last two entries are the *narrowing* cases rather than the
    /// unrelated ones. A gate that merely exists passes on messages sharing no
    /// words with a backend failure; these two share half of one and must
    /// still be request-scoped, so widening either signature back to a single
    /// loose fragment reddens this test.
    @Test(
        "request-scoped failures never condemn the worker",
        arguments: [
            "generation ended without completion info; token usage is unknown",
            "generation was cancelled before it finished",
            "invalid_body: max_tokens must be between 1 and 1048576",
            "unsupported_role: developer",
            "The operation could not be completed",
            "",
            GenerationWorkerHealthTests.overBroadNeighbour,
            GenerationWorkerHealthTests.oversizedAllocation,
            // The allocator tag alone, with no exhaustion evidence at all.
            "[metal::malloc] buffer cache trimmed",
            // The limit phrase alone, from a subsystem that is not the GPU.
            "HTTPError: Resource limit reached for this connection",
        ])
    func spareseRequestScopedFailures(description: String) {
        #expect(GenerationWorkerHealth.classify(failure: description) == .requestScoped)
        #expect(
            GenerationWorkerHealth.readiness(after: .ready, observing: description) == nil,
            "a request-scoped failure moved a ready runtime out of rotation")
    }

    /// Both halves of the incident, spelled out against the neighbour.
    ///
    /// Stated as a pair so the property under test is legible: the *same*
    /// words condemn or do not condemn depending on whether the Metal
    /// allocator context is present. A classifier narrowed until it matches
    /// nothing would fail the first expectation; one widened to a bare
    /// `Resource limit` would fail the second.
    @Test("the allocator context, not the limit phrase, is what condemns")
    func allocatorContextIsRequired() {
        #expect(
            GenerationWorkerHealth.classify(failure: Self.incidentFailure)
                == .workerInvalidated(signature: "metal-allocator-resource-limit"))
        #expect(GenerationWorkerHealth.classify(failure: Self.overBroadNeighbour) == .requestScoped)
    }

    /// Fails closed on a malformed signature.
    ///
    /// `allSatisfy` is vacuously true on an empty sequence, so a signature
    /// that lost its fragments would match every failure and condemn the
    /// runtime on the first one — a condemn-all gate arrived at by accident
    /// rather than by decision.
    @Test("a signature with no fragments matches nothing")
    func emptySignatureMatchesNothing() {
        let malformed = GenerationWorkerHealth.BackendFailureSignature(
            name: "malformed", requiredFragments: [])
        #expect(!malformed.matches(Self.incidentFailure))
        #expect(!malformed.matches(""))
    }

    /// Every shipped signature has to be able to tell the incident class from
    /// its neighbour, so a future entry cannot be added as one loose word.
    @Test("no shipped signature is a single loose fragment of a common message")
    func shippedSignaturesAreSpecific() {
        for signature in GenerationWorkerHealth.invalidatingSignatures {
            #expect(!signature.requiredFragments.isEmpty, "\(signature.name) has no fragments")
            #expect(
                !signature.matches(Self.overBroadNeighbour),
                "\(signature.name) condemns a request-scoped failure")
            #expect(
                !signature.matches(Self.oversizedAllocation),
                "\(signature.name) condemns a request-scoped allocation refusal")
        }
    }

    // MARK: - readiness transition

    @Test("a ready runtime that observes a condemned backend leaves rotation")
    func readyRuntimeLeavesRotation() {
        let next = GenerationWorkerHealth.readiness(
            after: .ready, observing: Self.incidentFailure)
        #expect(next == .generationWorkerFailed(Self.incidentFailure))
    }

    /// Only `ready` transitions.
    ///
    /// A generation in flight when `SIGTERM` arrives fails with whatever the
    /// backend was doing at the time. Letting that overwrite `shuttingDown`
    /// would report an orderly stop as a dead worker and — with the
    /// supervision marker attached — ask the harness to restart a process that
    /// was deliberately asked to stop. `loading` and the two terminal states
    /// are excluded for the same reason: the reason already being reported is
    /// the true one.
    @Test(
        "a runtime that is not ready keeps the state it already reports",
        arguments: [
            RuntimeReadiness.loading,
            RuntimeReadiness.shuttingDown,
            RuntimeReadiness.failed("no factory accepted model_type qwen3_5"),
            RuntimeReadiness.generationWorkerFailed("already condemned"),
        ])
    func nonReadyRuntimesDoNotTransition(current: RuntimeReadiness) {
        #expect(
            GenerationWorkerHealth.readiness(after: current, observing: Self.incidentFailure) == nil
        )
    }

    // MARK: - the endpoint

    @Test("a ready runtime is the only thing that answers health 200")
    func healthyRuntimeAnswers200() {
        let report = HealthReport.make(readiness: .ready)
        #expect(report.status == 200)
        #expect(report.body == .object(["status": .string("ok")]))
    }

    /// The regression itself.
    ///
    /// This is the assertion the upstream `mlx-lm` fix added, in the runtime it
    /// was carried into: with the generation worker gone, `/health` must be
    /// `503` and must say `unavailable`, not `ok`.
    @Test("a condemned worker answers health 503 unavailable")
    func condemnedWorkerAnswers503() {
        let report = HealthReport.make(readiness: .generationWorkerFailed(Self.incidentFailure))
        #expect(report.status == 503)
        #expect(report.body.objectValue?["status"] == .string("unavailable"))
        #expect(report.body.objectValue?["detail"] == .string(Self.incidentFailure))
    }

    @Test(
        "no unready state can answer health 200",
        arguments: [
            RuntimeReadiness.loading,
            RuntimeReadiness.shuttingDown,
            RuntimeReadiness.failed("no factory accepted model_type qwen3_5"),
            RuntimeReadiness.generationWorkerFailed(GenerationWorkerHealthTests.incidentFailure),
        ])
    func unreadyStatesNeverAnswer200(readiness: RuntimeReadiness) {
        let report = HealthReport.make(readiness: readiness)
        #expect(report.status == 503)
        #expect(report.body.objectValue?["status"] != .string("ok"))
    }

    // MARK: - the listing

    /// `/health` and `/v1/models` must agree.
    ///
    /// The managed launcher polls the listing, not health, and accepts a `200`
    /// whose `data[]` carries the exact model ID. A condemned runtime that kept
    /// advertising would be handed traffic by the launcher no matter what
    /// `/health` said.
    @Test("a condemned worker stops advertising the model")
    func condemnedWorkerStopsAdvertising() {
        let modelID = "/Users/alexis/src/local-models/Qwen3.8-27B-Uncensored-MLX-8bit"
        let listing = ModelsListing.make(
            modelID: modelID,
            readiness: .generationWorkerFailed(Self.incidentFailure),
            created: 1)
        #expect(listing.status == 503)
        #expect(listing.body.objectValue?["data"] == .array([]))
        let error = listing.body.objectValue?["error"]?.objectValue
        // Not `model_load_failed`: the weights did load. Reporting the wrong
        // code sends whoever reads the receipt to the wrong subsystem.
        #expect(error?["code"] == .string(GenerationWorkerHealth.supervisionMarker))
        #expect(error?["message"] == .string(Self.incidentFailure))
        guard let encoded = try? JSONEncoding.string(listing.body) else {
            Issue.record("listing did not encode")
            return
        }
        #expect(!encoded.contains(modelID), "the model ID leaked into a condemned listing")
    }

    // MARK: - the supervision contract

    /// The marker is a wire value shared with `model-harness`.
    ///
    /// A profile pins it by literal substring in `fatal_output_substrings`.
    /// Renaming it here without updating every such profile would leave the
    /// runtime dead-but-listening with no supervisor coming — the original
    /// incident, with extra steps. Pinned so that rename has to be deliberate.
    @Test("the supervision marker is a stable literal")
    func supervisionMarkerIsPinned() {
        #expect(GenerationWorkerHealth.supervisionMarker == "generation_worker_unavailable")
    }

    @Test("the emitted supervision event carries the marker verbatim")
    func supervisionEventCarriesMarker() throws {
        // The shape `Router.recordGenerationFailure(_:)` emits.
        let event = RuntimeEvent(
            name: "generation_worker_failed",
            fields: [
                "marker": .string(GenerationWorkerHealth.supervisionMarker),
                "detail": .string(Self.incidentFailure),
            ])
        let line = try event.line()
        #expect(line.contains(GenerationWorkerHealth.supervisionMarker))
        #expect(!line.contains("\n"), "a multi-line event would break substring matching")
    }
}

@Suite("generation fault injection seam")
struct FaultInjectionOptionsTests {
    static let base = [
        "serve", "--model", "/models/qwen", "--port", "18017",
    ]

    @Test("the seam is off unless asked for")
    func offByDefault() throws {
        let (_, options) = try RuntimeOptions.parse(arguments: Self.base)
        #expect(options.faultInjectedGenerationError == nil)
    }

    @Test("the injected message is carried verbatim")
    func carriesMessageVerbatim() throws {
        let message = GenerationWorkerHealthTests.incidentFailure
        let (_, options) = try RuntimeOptions.parse(
            arguments: Self.base + ["--fault-inject-generation-error", message])
        #expect(options.faultInjectedGenerationError == message)
        // The end-to-end point of "verbatim": what production classifies is
        // exactly what the operator asked to inject.
        guard
            case .workerInvalidated = GenerationWorkerHealth.classify(
                failure: options.faultInjectedGenerationError ?? "")
        else {
            Issue.record("the injected incident message did not condemn the worker")
            return
        }
    }

    @Test("an empty injection is refused at parse time")
    func refusesEmptyInjection() {
        #expect(throws: RuntimeOptionsError.emptyFaultInjection) {
            try RuntimeOptions.parse(
                arguments: Self.base + ["--fault-inject-generation-error", ""])
        }
    }
}
