import Foundation

/// What an observed generation failure says about the runtime's ability to
/// serve the *next* request.
public enum GenerationWorkerVerdict: Sendable, Equatable {
    /// The failure belongs to the request that provoked it. The worker is
    /// still able to serve, so readiness must not change: flipping `/health`
    /// on a bad request would take a healthy runtime out of rotation.
    case requestScoped
    /// The generation backend itself is broken. The worker cannot be trusted
    /// with another request, so `/health` must stop answering `200` and the
    /// supervisor must recover the process.
    case workerInvalidated(signature: String)
}

/// Liveness of the thing that actually produces tokens, as distinct from
/// whether weights are resident.
///
/// This carries `mlx_lm.server`'s dead-generation-thread regression across to
/// the Swift runtime. In Python, `ResponseGenerator` runs generation on a
/// long-lived thread; an uncaught exception killed that thread while the HTTP
/// listener stayed up, so `/health` kept answering `200 {"status": "ok"}` for a
/// runtime that could no longer produce a single token, and every client
/// discovered it as a request timeout instead. Recorded as `BUG-260827-1jhv2g`
/// and fixed upstream by making `/health` report
/// `503 {"status": "unavailable"}` when `_generation_thread.is_alive()` is
/// false.
///
/// Swift has no equivalent silent death — an error thrown inside a structured
/// task propagates to its caller rather than tearing a worker down — so the
/// same regression lands here as *invalidation*: the runtime observes a
/// generation failure whose signature means the MLX backend, not the request,
/// is broken, and takes itself out of rotation deliberately.
///
/// **The boundary of that claim, stated rather than implied.** This contract
/// covers failures that are *delivered as throws*. `GenerationEngine.run` wraps
/// generation in `MLX.withError`, so an MLX C++ error raised on the task that
/// is awaiting the stream becomes a Swift error and reaches the invalidation
/// path above. An MLX error raised on an MLX-owned `asyncEval` thread does not:
/// it reaches MLX's global default handler and traps the process. A trapped
/// process cannot emit `503`, cannot emit `generation_worker_unavailable`, and
/// cannot invalidate its own readiness — recovery there begins with the
/// supervisor observing that the process died, and `model-harness` replacement
/// is the whole of it.
///
/// Established by TASK-260827-2v13w8 revision 2, which provoked a real
/// 255,904,140,288-byte allocation on the vision-first path: delivered as a
/// throw it produced `HTTP 500`, a surviving process and a served next request;
/// the same class of error on the eval thread ends the process instead. Nothing
/// in this type should be read as evidence that arbitrary MLX backend faults are
/// survivable in process.
///
/// Production call sites: `RuntimeState.recordGenerationFailure(_:)` for the
/// readiness transition and `Router.health()` via ``HealthReport/make(readiness:)``
/// for the endpoint, both in the `mlx-swift-runtime-prototype` executable target.
public enum GenerationWorkerHealth {
    /// The literal substring `model-harness` matches against the managed
    /// child's forwarded output to trigger the configured supervised restart.
    ///
    /// It is a stable token rather than prose because a supervision policy
    /// pins it by value in `fatal_output_substrings`: reworded log text would
    /// silently stop matching and the runtime would sit dead-but-listening,
    /// which is the exact failure this whole path exists to end.
    public static let supervisionMarker = "generation_worker_unavailable"

    /// A failure signature that condemns the backend rather than the request.
    ///
    /// Every fragment must be present. The conjunction is the whole point: the
    /// only recorded backend death reports *two* things at once — that the
    /// message came from MLX's Metal allocator, and that the allocator is out
    /// of resources — and either half on its own is a phrase an ordinary
    /// request-scoped error can carry. `Resource limit` matched independently
    /// condemns a healthy runtime on, for example,
    /// `RequestError: Resource limit for this request is 8 tokens`.
    public struct BackendFailureSignature: Sendable, Equatable {
        /// Stable identifier for the condition, reported as the verdict's
        /// signature. Not the matched text: the matched text is a fragment of
        /// a message, and naming the condition is what a reader of the verdict
        /// actually needs.
        public let name: String
        /// Fragments that must *all* appear in the failure description.
        public let requiredFragments: [String]

        public init(name: String, requiredFragments: [String]) {
            self.name = name
            self.requiredFragments = requiredFragments
        }

        /// - Returns: `false` for an empty fragment list. `allSatisfy` is
        ///   vacuously true on an empty sequence, so a signature that lost its
        ///   fragments — through a bad edit or a decoding default — would
        ///   otherwise match *every* failure and condemn the runtime on the
        ///   first one. Failing closed here means a malformed signature
        ///   condemns nothing rather than everything.
        public func matches(_ description: String) -> Bool {
            guard !requiredFragments.isEmpty else { return false }
            return requiredFragments.allSatisfy { description.contains($0) }
        }
    }

    /// Failure signatures that condemn the backend rather than the request.
    ///
    /// Deliberately short, and deliberately conjunctive. Every entry names a
    /// condition that was actually observed to leave MLX unable to generate; a
    /// speculative entry, or an entry loose enough to match a neighbouring
    /// message, would turn a recoverable request error into a restart loop,
    /// and this list is the only thing standing between the two.
    ///
    /// - `metal-allocator-resource-limit` — the `BUG-260827-1jhv2g` incident.
    ///   MLX's Metal allocator hit its buffer-object limit through the
    ///   `qwen3_5` `ArraysCache` leak; the Python generation thread died on it
    ///   and never recovered. Emitted verbatim as
    ///   `[metal::malloc] Resource limit (N) exceeded.` by
    ///   `mlx/backend/metal/allocator.cpp:141-144` in the vendored MLX, which
    ///   is why the bracketed allocator tag is required alongside the limit
    ///   phrase rather than trusted to appear on its own.
    ///
    ///   The allocator's *other* throw at `allocator.cpp:111-117`
    ///   (`[metal::malloc] Attempting to allocate N bytes which is greater
    ///   than the maximum allowed buffer size`) is deliberately **not** here:
    ///   that one rejects a single oversized allocation and leaves the pool
    ///   intact, so it belongs to the request that asked for it.
    /// - `metal-shader-library-unreachable` — MLX cannot reach the GPU at all.
    ///   Recorded in `LOGBOOK.md` for the `swift build` product; the startup
    ///   gate refuses that build up front, so seeing it *during* generation
    ///   means the shader library went away under a running process.
    public static let invalidatingSignatures: [BackendFailureSignature] = [
        BackendFailureSignature(
            name: "metal-allocator-resource-limit",
            requiredFragments: ["[metal::malloc]", "Resource limit"]),
        BackendFailureSignature(
            name: "metal-shader-library-unreachable",
            requiredFragments: ["Failed to load the default metallib"]),
    ]

    /// Classify a generation failure by its reported description.
    ///
    /// Matching is on the message because MLX surfaces these as opaque backend
    /// errors with no Swift type to switch on. Both directions are dangerous:
    /// treating *every* generation error as fatal would restart the runtime
    /// whenever a single request tripped over its own parameters, and matching
    /// too loosely does the same thing for any request-scoped error that
    /// happens to reuse a word from a backend message.
    public static func classify(failure description: String) -> GenerationWorkerVerdict {
        for signature in invalidatingSignatures where signature.matches(description) {
            return .workerInvalidated(signature: signature.name)
        }
        return .requestScoped
    }

    /// The readiness a runtime should move to after observing a generation
    /// failure, or `nil` when it must stay where it is.
    ///
    /// Both guards matter and both are load-bearing:
    ///
    /// - a request-scoped failure never changes readiness, or a malformed
    ///   request would be able to take a healthy runtime down;
    /// - only a `ready` runtime transitions. A failure seen while loading,
    ///   already failed, or shutting down must not overwrite the reason that
    ///   is already being reported — in particular, a generation cancelled by
    ///   `SIGTERM` must not be reported as a dead worker and must not provoke
    ///   a restart of a process that was asked to stop.
    public static func readiness(
        after current: RuntimeReadiness, observing description: String
    ) -> RuntimeReadiness? {
        guard case .workerInvalidated = classify(failure: description) else { return nil }
        guard current == .ready else { return nil }
        return .generationWorkerFailed(description)
    }
}

/// The `GET /health` answer.
///
/// Split out of the router so the status/body pairing is testable without a
/// model, a port or MLX. `model-harness` and the Pi launcher both read this
/// endpoint, and the whole point of the carried regression is that `200` here
/// must mean "this runtime can generate", not "this process is running".
///
/// Production call site: `Router.health()` in the `mlx-swift-runtime-prototype`
/// executable target.
public struct HealthReport: Sendable, Equatable {
    public let status: Int
    public let body: JSONValue

    public static func make(readiness: RuntimeReadiness) -> HealthReport {
        switch readiness {
        case .ready:
            return HealthReport(status: 200, body: .object(["status": .string("ok")]))
        case .loading:
            return HealthReport(status: 503, body: .object(["status": .string("loading")]))
        case .shuttingDown:
            return HealthReport(status: 503, body: .object(["status": .string("shutting_down")]))
        case .failed(let reason):
            return HealthReport(
                status: 503,
                body: .object(["status": .string("failed"), "detail": .string(reason)]))
        case .generationWorkerFailed(let reason):
            // `unavailable` verbatim from the upstream mlx-lm health patch, so
            // a client that already distinguishes the Python runtime's dead
            // worker sees the same word from this one.
            return HealthReport(
                status: 503,
                body: .object(["status": .string("unavailable"), "detail": .string(reason)]))
        }
    }
}
