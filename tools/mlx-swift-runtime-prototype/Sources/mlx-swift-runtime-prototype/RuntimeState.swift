import Foundation
import MLXSwiftRuntimeContract

/// Readiness and the loaded engine, in one place.
///
/// Readiness is stored, never derived from configuration. `/v1/models` reads it
/// so the endpoint cannot advertise a model that is not resident yet.
actor RuntimeState {
    private var readiness: RuntimeReadiness = .loading
    private var engine: GenerationEngine?
    private var backendInitialized = false

    func currentReadiness() -> RuntimeReadiness { readiness }

    func currentEngine() -> GenerationEngine? { engine }

    /// Whether MLX has ever been driven in this process.
    ///
    /// Tracked separately from the engine because the engine is dropped on
    /// condemnation while MLX's allocator counters remain both readable and
    /// interesting -- "how much did the dead worker give back" is a question
    /// asked after there is no engine left. Kept as a latch rather than derived
    /// from readiness, so a load that failed *before* touching MLX reports no
    /// allocator figures instead of reporting zeros as measurements.
    func hasInitializedBackend() -> Bool { backendInitialized }

    func markReady(engine: GenerationEngine) {
        self.engine = engine
        backendInitialized = true
        readiness = .ready
    }

    func markFailed(_ reason: String) {
        engine = nil
        readiness = .failed(reason)
    }

    func markShuttingDown() {
        readiness = .shuttingDown
    }

    /// Production call site for
    /// ``GenerationWorkerHealth/readiness(after:observing:)``.
    ///
    /// Drops the engine as well as flipping readiness, so a request that
    /// arrives after the worker is condemned is refused with `503` at the same
    /// gate that refuses one arriving before the weights land, rather than
    /// being handed an engine already known to be broken.
    ///
    /// - Returns: `true` when this call is the one that condemned the worker,
    ///   so the caller emits the supervision marker exactly once. A second
    ///   failure from a request already in flight must not re-announce it.
    func recordGenerationFailure(_ description: String) -> Bool {
        guard
            let next = GenerationWorkerHealth.readiness(
                after: readiness, observing: description)
        else { return false }
        engine = nil
        readiness = next
        return true
    }
}
