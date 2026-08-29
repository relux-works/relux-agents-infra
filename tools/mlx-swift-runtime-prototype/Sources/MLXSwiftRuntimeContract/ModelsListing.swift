import Foundation

/// Serving state of the single model this process serves.
///
/// This is readiness to *answer*, not merely to have loaded: a runtime whose
/// generation worker has been invalidated is no more usable than one whose
/// weights never arrived, and both must stop advertising the model.
public enum RuntimeReadiness: Sendable, Equatable {
    case loading
    case ready
    case failed(String)
    /// The weights loaded, but the generation backend was condemned by
    /// ``GenerationWorkerHealth``. Terminal for this process; recovery is the
    /// supervisor's job, not this runtime's.
    case generationWorkerFailed(String)
    case shuttingDown
}

/// The `GET /v1/models` answer.
///
/// The managed contract in `agents-infra` polls this endpoint and treats
/// `503` as "keep waiting", any other non-200 as a hard readiness failure, and
/// a `200` whose `data[]` lacks the exact configured model ID as
/// `runtime_model_unavailable`. Advertising the model before the weights are
/// resident would therefore hand the launcher a false ready signal, so the
/// listing is derived from ``RuntimeReadiness`` and never from configuration
/// alone.
///
/// Production call site: `ModelsRoute.respond()` in the
/// `mlx-swift-runtime-prototype` executable target.
public struct ModelsListing: Sendable, Equatable {
    public let status: Int
    public let body: JSONValue

    public static func make(
        modelID: String, readiness: RuntimeReadiness, created: Int
    ) -> ModelsListing {
        switch readiness {
        case .ready:
            let entry = JSONValue.object([
                "id": .string(modelID),
                "object": .string("model"),
                "created": .int(created),
                "owned_by": .string("mlx-swift-runtime-prototype"),
            ])
            return ModelsListing(
                status: 200,
                body: .object(["object": .string("list"), "data": .array([entry])]))
        case .loading, .shuttingDown:
            // 503 with an empty list: still an OpenAI model list, so a poller
            // that ignores the status code still cannot find the model ID.
            return ModelsListing(
                status: 503,
                body: .object(["object": .string("list"), "data": .array([])]))
        case .failed(let reason):
            return ModelsListing(status: 503, body: unavailable(reason, code: "model_load_failed"))
        case .generationWorkerFailed(let reason):
            // Reported under its own code: this runtime did load the model, and
            // calling it a load failure would send whoever reads the receipt
            // looking at the weights instead of at the generation backend.
            return ModelsListing(
                status: 503,
                body: unavailable(reason, code: GenerationWorkerHealth.supervisionMarker))
        }
    }

    private static func unavailable(_ reason: String, code: String) -> JSONValue {
        .object([
            "object": .string("list"),
            "data": .array([]),
            "error": .object([
                "message": .string(reason),
                "type": .string("server_error"),
                "code": .string(code),
            ]),
        ])
    }
}
