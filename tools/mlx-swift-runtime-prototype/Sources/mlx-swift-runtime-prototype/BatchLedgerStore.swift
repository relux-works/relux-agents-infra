import Foundation
import MLX
import MLXSwiftRuntimeContract

/// The runtime's single ``GenerationBatchLedger``, and the place recovery is
/// actually performed.
///
/// Deliberately *not* owned by `GenerationEngine`. Condemnation drops the
/// engine — that is the whole point of `RuntimeState.recordGenerationFailure` —
/// and an operator asking "did the worker that just died leave anything behind"
/// is asking after the engine is gone. A ledger stored inside the engine would
/// answer that question by disappearing, which reads exactly like a clean
/// runtime.
///
/// One process, one store, created in `Main` and shared by `GenerationEngine`
/// (which begins, finishes and fails slots) and `Router` (which publishes it on
/// `GET /debug/generation-state`).
actor GenerationBatchLedgerStore {
    private var ledger = GenerationBatchLedger()

    /// A condemned worker's pool rebuild, waiting for the engine to actually go.
    ///
    /// Set when a failure condemns the backend, discharged by
    /// ``completeWorkerTeardown()`` once the request that observed it has
    /// unwound and `RuntimeState` has dropped the engine. Until then the model's
    /// weight buffers are still held, and clearing the pool would empty it just
    /// before ~261 MB of freed weights land back in it.
    private var rebuildPendingWorkerTeardown = false

    func snapshot() -> GenerationBatchLedger { ledger }

    /// Generations still open, for the teardown's attribution veto.
    ///
    /// Read by ``WeightReleaseBarrier`` on every poll rather than captured
    /// once: a reading taken while another generation is allocating and
    /// freeing says nothing about whose bytes came back, and the release gate
    /// refuses on it. See
    /// ``GenerationBatchRecovery/WeightReleaseObservation/generationsInFlight``.
    func generationsInFlight() -> Int { ledger.active }

    func begin() -> GenerationBatchLedger.Slot { ledger.begin() }

    func finish(_ slot: GenerationBatchLedger.Slot) {
        ledger.finish(slot)
    }

    /// Close a failed generation and carry out the plan its failure implies.
    ///
    /// The release of the batch entry itself is the caller's: the KV cache
    /// hangs off the `ChatSession` that `GenerationEngine.generate` holds in a
    /// local, and unwinding out of that scope is what drops it. What happens
    /// here is the part that is *not* automatic — returning MLX's shared
    /// buffer pool to the system, which nothing unwinds — plus the record that
    /// makes both observable.
    ///
    /// - Returns: the plan that was applied, or `nil` if the slot was already
    ///   closed, in which case nothing was applied and nothing was counted.
    @discardableResult
    func fail(_ slot: GenerationBatchLedger.Slot, observing description: String)
        -> GenerationRecoveryPlan?
    {
        guard let plan = ledger.fail(slot, observing: description) else { return nil }
        switch plan.sharedCacheRebuild {
        case .none:
            break
        case .immediate:
            // Request-scoped pressure. The engine survives, so everything this
            // failure freed is already back in the pool and clearing now is
            // both correct and effective.
            clearSharedCache()
        case .afterWorkerTeardown:
            // The weights are still held by the frame that is unwinding.
            // Deferred to ``completeWorkerTeardown(releaseObserved:attempt:maxAttempts:)``.
            rebuildPendingWorkerTeardown = true
            ledger.deferSharedCacheRebuild()
        }
        StandardOutput.shared.event(
            RuntimeEvent(
                name: "generation_batch_released",
                fields: [
                    "slot": .int(slot.id),
                    "active": .int(ledger.active),
                    "failed": .int(ledger.failed),
                    "batches_released": .int(ledger.batchesReleased),
                    "shared_cache_rebuilds": .int(ledger.sharedCacheRebuilds),
                    "rebuilt_shared_cache": .bool(plan.sharedCacheRebuild == .immediate),
                    "shared_cache_rebuild_deferred": .bool(
                        plan.sharedCacheRebuild == .afterWorkerTeardown),
                    "condemned_signature": plan.condemnedSignature.map { .string($0) } ?? .null,
                    "detail": .string(description),
                ]))
        return plan
    }

    /// Drop MLX's shared buffer pool and record that it happened.
    ///
    /// The count is taken here, one line after the call it describes, so a
    /// deleted `Memory.clearCache()` cannot leave the counter behind to vouch
    /// for it.
    private func clearSharedCache() {
        Memory.clearCache()
        ledger.recordSharedCacheRebuild()
    }

    /// Whether a condemned worker still owes the shared pool a rebuild.
    func hasPendingWorkerTeardown() -> Bool { rebuildPendingWorkerTeardown }

    /// Discharge — or refuse to discharge — a condemned worker's deferred pool
    /// rebuild.
    ///
    /// Production call site: `GenerationEngine.deinit`, which schedules this
    /// once the engine has been deallocated and ``WeightReleaseBarrier`` has
    /// taken a reading of whether every weight buffer has actually been
    /// returned to `buffer_cache_`. *Then* this empties it.
    ///
    /// Doing it any earlier returns nothing, and both earlier attempts were
    /// measured. Clearing from the failing request left the pool empty while
    /// the request still held 303,782,980 active bytes of model, which landed
    /// in the pool moments later; review measured the same shape at 299,129,824
    /// cache bytes on a condemned runtime whose ledger reported the pool
    /// rebuilt.
    ///
    /// The release verdict is a gate, not a hint, and it is not taken on
    /// trust: this asks ``GenerationBatchRecovery/weightsReleased(_:)`` about
    /// the observation rather than accepting a boolean somebody else computed.
    /// Revision 2 waited for MLX's process-global active bytes to fall below
    /// half their condemnation-time reading, discarded the timeout, and cleared
    /// regardless. Revision 3 fixed the timeout but answered the release
    /// question from the outer container's `weak` reference alone, which review
    /// showed can read `nil` while the weight-owning `ModelContext` is still
    /// being destroyed — a narrowed mutant attested a completed rebuild with
    /// 262,361,760 bytes still active. Now neither an unobserved release nor a
    /// merely-deallocated wrapper can reach the clear.
    ///
    /// Idempotent: a second failure from a request already in flight arrives
    /// here again and must not clear, or count, twice.
    ///
    /// - Returns: what this attempt concluded. ``WorkerTeardownOutcome/retry``
    ///   means the caller should wait again; the pending flag is still raised.
    @discardableResult
    func completeWorkerTeardown(
        observation: GenerationBatchRecovery.WeightReleaseObservation,
        attempt: Int,
        maxAttempts: Int
    ) -> WorkerTeardownOutcome {
        guard rebuildPendingWorkerTeardown else { return .rebuilt }
        let releaseObserved = GenerationBatchRecovery.weightsReleased(observation)
        let outcome = GenerationBatchRecovery.teardownOutcome(
            releaseObserved: releaseObserved, attempt: attempt, maxAttempts: maxAttempts)
        switch outcome {
        case .retry:
            // Nothing is cleared and nothing is counted. The only thing that
            // happens is that the runtime says, on the record, that it looked
            // and did not see the release.
            StandardOutput.shared.event(
                RuntimeEvent(
                    name: "generation_shared_cache_rebuild_deferred",
                    fields: [
                        "reason": .string("worker_teardown"),
                        "release_observed": .bool(false),
                        "attempt": .int(attempt),
                        "max_attempts": .int(maxAttempts),
                        // The reading the verdict was taken from, not just the
                        // verdict. `container_deallocated=true` beside
                        // `returned_bytes` short of the footprint is exactly
                        // the interval revision 3 could not see: the wrapper is
                        // gone and the weights are not.
                        "container_deallocated": .bool(observation.containerDeallocated),
                        "weight_owner_count": .int(observation.weightOwnerCount),
                        "live_weight_owners": .int(observation.liveWeightOwners),
                        "generations_in_flight": .int(observation.generationsInFlight),
                        "stable_active_samples": .int(observation.stableActiveSamples),
                        "weight_footprint_bytes": .int(observation.weightFootprintBytes),
                        "baseline_active_bytes": .int(observation.baselineActiveBytes),
                        "returned_bytes": .int(observation.returnedBytes),
                        "active_bytes": .int(observation.activeBytes),
                        "observed_active_bytes": .int(observation.activeBytes),
                        "residual_non_weight_allowance_bytes": .int(
                            GenerationBatchRecovery.residualNonWeightAllowanceBytes),
                        "cache_bytes": .int(Memory.snapshot().cacheMemory),
                    ]))
        case .abandoned:
            // Terminal, and deliberately loud. The pending flag stays raised
            // and `shared_cache_rebuilds` stays where it was, so
            // `GET /debug/generation-state` keeps reporting the pool as owed
            // rather than returned. The supervision marker is re-emitted
            // because the condition this describes -- a condemned model still
            // holding its buffers -- is exactly the one that must not be left
            // competing with a replacement process for the host's memory.
            ledger.abandonSharedCacheRebuild()
            let usage = memoryUsage()
            StandardOutput.shared.event(
                RuntimeEvent(
                    name: "generation_shared_cache_rebuild_abandoned",
                    fields: [
                        "reason": .string("worker_teardown"),
                        "release_observed": .bool(false),
                        "attempts": .int(maxAttempts),
                        "container_deallocated": .bool(observation.containerDeallocated),
                        // The attribution reading, and the one the byte figures
                        // beside it cannot supply. `live_weight_owners > 0`
                        // beside a `returned_bytes` that clears the footprint is
                        // exactly the interval revision 4 could not see.
                        "weight_owner_count": .int(observation.weightOwnerCount),
                        "live_weight_owners": .int(observation.liveWeightOwners),
                        "generations_in_flight": .int(observation.generationsInFlight),
                        "stable_active_samples": .int(observation.stableActiveSamples),
                        "weight_footprint_bytes": .int(observation.weightFootprintBytes),
                        "baseline_active_bytes": .int(observation.baselineActiveBytes),
                        "returned_bytes": .int(observation.returnedBytes),
                        "shared_cache_rebuilds": .int(ledger.sharedCacheRebuilds),
                        "shared_cache_rebuilds_abandoned": .int(
                            ledger.sharedCacheRebuildsAbandoned),
                        "shared_cache_rebuild_pending": .bool(ledger.sharedCacheRebuildPending),
                        // The residue the refusal was taken on, published
                        // beside the allowance it was compared against so an
                        // acceptance run can tell a refusal caused by residue
                        // from one caused by ownership or by the clock. The
                        // allowance is zero, so any residue at all lands here.
                        "observed_active_bytes": .int(observation.activeBytes),
                        "residual_non_weight_allowance_bytes": .int(
                            GenerationBatchRecovery.residualNonWeightAllowanceBytes),
                        "active_bytes": .int(usage.activeBytes),
                        "cache_bytes": .int(usage.cacheBytes),
                        "detail": .string(
                            "condemned worker's weight release was never observed; shared pool "
                                + "not rebuilt"),
                    ]))
            StandardOutput.shared.log(
                "\(GenerationWorkerHealth.supervisionMarker): condemned worker did not release "
                    + "its weights within \(maxAttempts) teardown attempts; the shared buffer "
                    + "pool was not rebuilt and this process must be replaced")
        case .rebuilt:
            rebuildPendingWorkerTeardown = false
            let before = Memory.snapshot()
            clearSharedCache()
            let usage = memoryUsage()
            StandardOutput.shared.event(
                RuntimeEvent(
                    name: "generation_shared_cache_rebuilt",
                    fields: [
                        "reason": .string("worker_teardown"),
                        "release_observed": .bool(true),
                        "attempt": .int(attempt),
                        "container_deallocated": .bool(observation.containerDeallocated),
                        "weight_owner_count": .int(observation.weightOwnerCount),
                        "live_weight_owners": .int(observation.liveWeightOwners),
                        "generations_in_flight": .int(observation.generationsInFlight),
                        "stable_active_samples": .int(observation.stableActiveSamples),
                        "weight_footprint_bytes": .int(observation.weightFootprintBytes),
                        "baseline_active_bytes": .int(observation.baselineActiveBytes),
                        "returned_bytes": .int(observation.returnedBytes),
                        // The absolute residue the verdict was taken from, as
                        // opposed to the `active_bytes` pair below, which is
                        // read around the clear. Published because this is the
                        // number review's revision-4 finding turns on: a drop
                        // of any size means nothing while this is at or above
                        // the footprint.
                        "observed_active_bytes": .int(observation.activeBytes),
                        "residual_non_weight_allowance_bytes": .int(
                            GenerationBatchRecovery.residualNonWeightAllowanceBytes),
                        "shared_cache_rebuilds": .int(ledger.sharedCacheRebuilds),
                        // Both sides of the clear. The `before` pair is what
                        // makes a mis-ordered teardown legible instead of
                        // merely wrong: high active bytes here means the
                        // weights had not been released yet and the clear
                        // returned nothing.
                        "active_bytes_before": .int(before.activeMemory),
                        "cache_bytes_before": .int(before.cacheMemory),
                        "active_bytes": .int(usage.activeBytes),
                        "cache_bytes": .int(usage.cacheBytes),
                    ]))
        }
        return outcome
    }

    /// Allocator counters as of now, for `GET /debug/generation-state`.
    func memoryUsage() -> GenerationBatchReport.MemoryUsage {
        let snapshot = Memory.snapshot()
        return GenerationBatchReport.MemoryUsage(
            activeBytes: snapshot.activeMemory,
            cacheBytes: snapshot.cacheMemory,
            peakBytes: snapshot.peakMemory)
    }
}

/// Every object that owns a piece of this model's weights, held weakly.
///
/// The attribution primitive, and the answer to review's revision-4 finding.
/// MLX's allocator counters are process-global: they can say that 608 MB
/// stopped being active and cannot say whether those were the model's bytes or
/// a 6,000-word prompt's KV cache. This can. Every weight array of an MLX Swift
/// model is a stored property of some `Module` in the tree rooted at
/// `ModelContext.model`, so registering that whole tree — `modules()` returns
/// it flat, root included — and watching all of it die is a statement about
/// *these* weights rather than about the process's byte total.
///
/// `weak` throughout, so populating the registry cannot itself be what keeps
/// the model alive. ``count`` is the size the registry was populated to and
/// never changes; ``live`` is how many of those references still read
/// non-`nil`.
///
/// A registry that was never populated reports `count == 0`, and the release
/// gate fails closed on that rather than reading "no live owners" as an
/// attestation — it would be true forever. Absence and failure to read are
/// different facts.
final class WeightOwnerRegistry: @unchecked Sendable {
    private final class WeakBox {
        weak var object: AnyObject?
        init(_ object: AnyObject) { self.object = object }
    }

    private let lock = NSLock()
    private var boxes: [WeakBox] = []

    /// Register the model's whole module tree. Called once, at load.
    func register(_ objects: [AnyObject]) {
        lock.lock()
        defer { lock.unlock() }
        boxes = objects.map(WeakBox.init)
    }

    /// How many owners were registered. `0` means the registry was never
    /// populated, which the release gate treats as a failure to read.
    var count: Int {
        lock.lock()
        defer { lock.unlock() }
        return boxes.count
    }

    /// How many registered owners are still alive.
    var live: Int {
        lock.lock()
        defer { lock.unlock() }
        return boxes.reduce(into: 0) { total, box in
            if box.object != nil { total += 1 }
        }
    }
}

/// A deterministic answer to "are this worker's weights gone yet?"
///
/// Four readings, taken together, and no one of them is sufficient. Each was
/// added because review defeated the set without it.
///
/// 1. A `weak` reference to the exact `ModelContainer` the condemned engine
///    served from. While that reads non-`nil` the weights are certainly still
///    held, so it is a cheap and exact *veto*. Revision 3 shipped it as an
///    attestation, and review showed the wrapper reaching `nil` while the
///    `ModelContext` below it was still being destroyed.
/// 2. The ``WeightOwnerRegistry`` — every `Module` in this model's tree, held
///    weakly. This is the reading that *attributes*: while any of them is
///    alive, some of this model's weights are still owned, however many
///    unrelated bytes the process handed back.
/// 3. MLX's own `activeMemory`, read as an ABSOLUTE residue against a fixed
///    allowance of zero rather than against the model's footprint. Revision 4
///    compared a process-global delta, and review made the failed request's own
///    KV state larger than the model to satisfy it with every weight resident.
///    Revision 5 compared the residue against the footprint, and review put a
///    strict subset of the model's copied parameter arrays — 255,724,192 B of
///    262,361,760 B — inside that interval with zero live owners. Every
///    threshold above zero is an interval, and MLX cannot say what is inside
///    one; see ``GenerationBatchRecovery/residualNonWeightAllowanceBytes``.
/// 4. The ledger's in-flight count and the stability of the byte reading, so a
///    concurrent generation or a destruction still in progress cannot be
///    mistaken for a finished, quiet release.
///
/// The verdict itself is not taken here. This type only measures; whether a
/// measurement means "released" is
/// ``GenerationBatchRecovery/weightsReleased(_:)``, which is pure and has its
/// own negative tests.
///
/// - Note: `@unchecked Sendable` with a single-consumer contract. The barrier is
///   created in `GenerationEngine.deinit` and read only by the one `Task` that
///   `deinit` schedules, so the weak slot is never read concurrently. It is not
///   stored anywhere that could hand it to a second reader.
final class WeightReleaseBarrier: @unchecked Sendable {
    private weak var container: AnyObject?
    private let owners: WeightOwnerRegistry
    private let weightFootprintBytes: Int
    private let baselineActiveBytes: Int
    private let generationsInFlight: @Sendable () async -> Int

    /// The last `activeMemory` value seen, and how many consecutive polls have
    /// reported it. Instance state rather than per-wait state so the stability
    /// run survives across teardown attempts: the allocator does not restart
    /// when the caller retries.
    private var lastActiveBytes: Int?
    private var stableActiveSamples = 0

    /// - Parameters:
    ///   - container: the exact container this worker served from. Held weakly,
    ///     so constructing a barrier cannot itself keep the weights alive.
    ///   - owners: this model's module tree, registered at load. An empty
    ///     registry fails the gate closed.
    ///   - weightFootprintBytes: the `activeMemory` delta MLX reported across
    ///     this model's load. `0` means unmeasured, and the gate fails closed
    ///     on it rather than treating any reading as satisfying it.
    ///   - generationsInFlight: the ledger's open-slot count, re-read on every
    ///     poll. Non-zero vetoes the reading.
    init(
        observing container: AnyObject,
        owners: WeightOwnerRegistry,
        weightFootprintBytes: Int,
        generationsInFlight: @escaping @Sendable () async -> Int
    ) {
        self.container = container
        self.owners = owners
        self.weightFootprintBytes = weightFootprintBytes
        self.generationsInFlight = generationsInFlight
        // Taken now, inside `deinit`, while the model is provably still held —
        // a `deinit` body runs before its stored properties are released. This
        // is the top of the fall the release has to produce. It is a
        // *necessary* reference point and never a sufficient one: it is
        // process-global, and review defeated a gate that trusted it alone.
        self.baselineActiveBytes = Memory.snapshot().activeMemory
    }

    /// Take one reading, advancing the stability run.
    private func sample(generationsInFlight inFlight: Int)
        -> GenerationBatchRecovery.WeightReleaseObservation
    {
        let active = Memory.snapshot().activeMemory
        if active == lastActiveBytes {
            stableActiveSamples += 1
        } else {
            lastActiveBytes = active
            stableActiveSamples = 1
        }
        return GenerationBatchRecovery.WeightReleaseObservation(
            containerDeallocated: container == nil,
            weightOwnerCount: owners.count,
            liveWeightOwners: owners.live,
            weightFootprintBytes: weightFootprintBytes,
            baselineActiveBytes: baselineActiveBytes,
            activeBytes: active,
            generationsInFlight: inFlight,
            stableActiveSamples: stableActiveSamples)
    }

    /// Wait, bounded, for a reading that ``GenerationBatchRecovery/weightsReleased(_:)``
    /// accepts.
    ///
    /// A wait rather than an assumed ordering because the ordering cannot be
    /// assumed. `GenerationEngine.deinit` runs *before* its stored properties
    /// are released, so the container is still alive on the line that creates
    /// this barrier, and MLX returns buffers through its own scoped pools. Two
    /// earlier attempts — clearing from the failing request, then clearing
    /// right after the response was written — both measured ~303 MB still
    /// active at the moment of the clear.
    ///
    /// - Returns: the last reading taken, released or not. A reading that does
    ///   not satisfy the gate is a refusal, not a hint: see
    ///   ``GenerationBatchRecovery/teardownOutcome(releaseObserved:attempt:maxAttempts:)``,
    ///   which cannot reach a completed rebuild from it.
    func waitForRelease(polls: Int, intervalNanoseconds: UInt64) async
        -> GenerationBatchRecovery.WeightReleaseObservation
    {
        var latest = sample(generationsInFlight: await generationsInFlight())
        for _ in 0 ..< polls {
            if GenerationBatchRecovery.weightsReleased(latest) { return latest }
            try? await Task.sleep(nanoseconds: intervalNanoseconds)
            latest = sample(generationsInFlight: await generationsInFlight())
        }
        return latest
    }
}

/// The acceptance seam's parking place for a condemned worker's state.
///
/// Armed only by `--fault-inject-teardown-retain` (which parks the outer
/// `ModelContainer`, so the wrapper never reaches `weak`-`nil`), by
/// `--fault-inject-teardown-retain-weights` (which parks the whole weight-owning
/// `ModelContext.model` *below* it, so the wrapper does reach `weak`-`nil`
/// while the weights stay resident), or by
/// `--fault-inject-teardown-retain-weight-modules` (which parks a *strict
/// subset* of the model tree), or by
/// `--fault-inject-teardown-retain-weight-arrays` (which parks the weight
/// arrays and no object of the tree at all). The parser refuses any of them
/// without a generation injection to condemn the worker in the first place.
///
/// Four seams rather than one because they defeat four different clauses.
///
/// The first proves the timeout branch. The second is the interval revision 3
/// missed: a runtime answering the release question from the wrapper alone
/// passes every check the first seam can make and still attests a rebuild with
/// the entire model active. The third is the interval revision 4 missed, and
/// it is the only one no byte comparison can catch — a partial retention puts
/// `activeBytes` *below* the model's footprint while this model's weights are
/// still owned, and with a large enough request behind it the process-global
/// `returnedBytes` clears the footprint as well. Only ownership read from the
/// model tree refuses it.
///
/// The fourth is the mirror of the third, and the only one that isolates the
/// absolute-residue clause. It copies the flattened parameter arrays out of the
/// tree, so every `Module` dies and ownership reports the model released while
/// MLX goes on calling the whole footprint active. Ownership cannot refuse
/// that; only reading what is still resident can.
///
/// Holds for the lifetime of the process rather than for the duration of the
/// teardown wait. A seam whose whole claim is "these weights are never
/// released" must not quietly release them the moment the wait it was defeating
/// gives up: the first revision of this phase did, and the residue it was
/// supposed to demonstrate had already drained into `buffer_cache_` by the time
/// the acceptance suite read the allocator. What is being reproduced is a
/// reference that is genuinely stuck, so the seam gets genuinely stuck.
final class RetainedWeights: @unchecked Sendable {
    static let shared = RetainedWeights()

    private let lock = NSLock()
    private var held: [AnyObject] = []
    private var heldArrays: [MLXArray] = []

    func hold(_ object: AnyObject) {
        lock.lock()
        defer { lock.unlock() }
        held.append(object)
    }

    /// Park weight arrays without parking anything that owns them.
    ///
    /// `MLXArray` is a value type over a shared buffer handle, so a copy keeps
    /// the buffer alive while every `Module` that used to hold it dies on
    /// schedule. That is the point: this is the seam that leaves the ownership
    /// registry reporting a fully released model with weight buffers still
    /// active — the whole footprint when it is handed every array, and a
    /// significant *strict subset* of it when the caller narrows the set, which
    /// is review's revision-5 bypass.
    func holdArrays(_ arrays: [MLXArray]) {
        lock.lock()
        defer { lock.unlock() }
        heldArrays.append(contentsOf: arrays)
    }
}
