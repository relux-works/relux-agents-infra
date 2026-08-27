import Foundation

/// What a failed generation must do to the state it was holding.
///
/// The three fields are deliberately independent. A runtime that collapsed
/// them into one flag would have to pick between two wrong behaviours: never
/// dropping the shared buffer pool, so an allocation failure repeats on every
/// later request, or always dropping it, so one malformed request throws away
/// the pool that makes every *other* request fast.
public struct GenerationRecoveryPlan: Sendable, Equatable {
    /// Release this generation's batch entry and the KV cache hanging off it.
    ///
    /// Always true, and that is the point: the per-request cache belongs to the
    /// request, so it must not survive the request under *any* verdict. The
    /// field exists so the invariant is asserted at the plan rather than
    /// assumed at the call site — the recorded regression is precisely a
    /// runtime that kept per-request state alive after the request died.
    public let releasesBatch: Bool

    /// Whether MLX's shared buffer pool must be dropped, and *when*.
    ///
    /// Conditional, and narrow. The pool is shared across every request, and
    /// clearing it is the documented remedy for exactly one class of failure:
    /// the allocator refusing a buffer after its own partial reclaim, while the
    /// pool still holds freed buffers it has not returned to the system.
    /// Clearing it for an ordinary request-scoped error would be a
    /// self-inflicted latency regression on every subsequent generation, paid
    /// for a failure that never touched the pool.
    ///
    /// The *timing* is part of the plan rather than the call site's business
    /// because getting it wrong is silent. A condemned worker's weights are
    /// freed when the engine is dropped, which happens after the failing
    /// request has unwound; a clear performed before that returns an empty pool
    /// to a process that is about to refill it with the entire model. Review
    /// measured exactly that: 299,129,824 cache bytes left resident after a
    /// condemnation whose ledger reported the pool rebuilt.
    public let sharedCacheRebuild: SharedCacheRebuild

    /// When the shared pool may be dropped.
    public enum SharedCacheRebuild: Sendable, Equatable {
        /// The failure does not implicate the pool. Leave it alone.
        case none
        /// Request-scoped pressure. The engine survives, so the buffers freed
        /// by this failure are already back in the pool by the time the error
        /// reaches the ledger, and clearing now is both correct and effective.
        case immediate
        /// The worker is condemned. The engine — and with it every weight
        /// buffer — is released only after the request that observed the
        /// failure has finished unwinding, so the clear has to wait for that
        /// teardown or it returns the pool before the model refills it.
        case afterWorkerTeardown
    }

    /// Whether the pool is dropped at all, under any timing.
    public var rebuildsSharedCache: Bool { sharedCacheRebuild != .none }

    /// The condemning signature, or `nil` when the failure belongs to the
    /// request.
    ///
    /// Reported, not acted on. The readiness transition has exactly one owner —
    /// `RuntimeState.recordGenerationFailure` via
    /// ``GenerationWorkerHealth/readiness(after:observing:)`` — and a second
    /// place that could condemn the worker would be a second place that could
    /// get it wrong. This field exists so the released-batch record says *why*
    /// the shared pool was rebuilt, and so the two classifiers can be pinned
    /// to each other by test rather than drifting apart.
    public let condemnedSignature: String?

    public var condemnsWorker: Bool { condemnedSignature != nil }
}

/// Recovery of batch and cache state after a generation fails.
///
/// This carries the second half of the `mlx_lm.server` generation regression
/// into the Swift runtime. ``GenerationWorkerHealth`` covers the terminal case:
/// a backend that is gone must stop answering `/health` `200`. It says nothing
/// about the far more common case — a generation that fails, mid-batch, on a
/// runtime that is still perfectly able to serve the next caller.
///
/// In Python that case was not survivable at all: the exception escaped the
/// batch loop, killed the generation thread, and took the batch entry and its
/// KV cache with it into a process that kept listening. Here the error
/// propagates to its caller, which means the runtime *can* recover — and
/// therefore must, deliberately and observably, rather than by hoping ARC gets
/// to the session before the next request needs the memory.
///
/// **Scoped to failures that unwind through Swift.** "The error propagates to
/// its caller" is true for a generation error delivered as a throw, which is
/// what `MLX.withError` in `GenerationEngine.run` arranges for the awaiting
/// task. It is not true for an MLX error raised on an MLX-owned `asyncEval`
/// thread: that reaches MLX's global default handler and traps the process, so
/// no batch release is attested, no deferred pool rebuild is discharged, and
/// the cleanup is performed by process termination. Replacement by the
/// supervisor is the recovery path there, and this ledger has nothing to say
/// about it.
///
/// Measured in TASK-260827-2v13w8 revision 2 on the real model: the same
/// 255,904,140,288-byte allocation returned `HTTP 500` with
/// `generation_batch_released` emitted and the process alive when it arrived as
/// a throw, and ended the process when it did not.
///
/// Production call sites: `GenerationEngine.generate(_:emit:)` runs every
/// generation through ``GenerationBatchLedger``, and `Router.route` publishes
/// the ledger on `GET /debug/generation-state` via ``GenerationBatchReport``,
/// both in the `mlx-swift-runtime-prototype` executable target.
public enum GenerationBatchRecovery {
    /// Failure signatures that mean the *shared* buffer pool is implicated and
    /// should be dropped before the next request is attempted.
    ///
    /// Conjunctive for the same reason ``GenerationWorkerHealth`` is: each
    /// entry names a condition that asserts two facts at once, and either half
    /// alone is a phrase an unrelated failure can carry.
    ///
    /// Membership is decided from the pinned `mlx-swift` allocator source, not
    /// from how allocation-shaped the message reads. The question an entry has
    /// to answer is narrow: *can `clear_cache()` change the outcome of the next
    /// attempt?* For most allocator errors it cannot, and a rebuild is then a
    /// pure cold-pool cost charged to every later generation.
    ///
    /// - `metal-buffer-allocation-failed` — `mlx/backend/metal/allocator.cpp`
    ///   throws `[malloc] Unable to allocate N bytes.` when `newBuffer`
    ///   returns null. Reaching that throw means the allocator already ran its
    ///   own reclaim, `release_cached_buffers(mem_required - gc_limit_)`, and
    ///   still came up empty. That reclaim is a *slice*: it frees only enough
    ///   to get back under `gc_limit_`. `clear_cache()` empties
    ///   `buffer_cache_` outright, so it can hand back memory the allocator's
    ///   own partial reclaim deliberately kept — which is exactly the state
    ///   where the next attempt's outcome can change.
    ///
    ///   It does **not** condemn: one allocation failed, and the backend is
    ///   still able to serve. The tag is `[malloc]`, which is not a substring
    ///   of `[metal::malloc]`, so this signature and the condemning one cannot
    ///   collide through their tags.
    ///
    /// Deliberately **not** here — `[metal::malloc] Attempting to allocate N
    /// bytes which is greater than the maximum allowed buffer size`. It reads
    /// like allocation pressure and is not. The throw is the first thing
    /// `MetalAllocator::malloc` does, before `std::unique_lock lk(mutex_)` and
    /// before any cache is consulted, and its test is `size >
    /// device_->maxBufferLength()`. `clear_cache()` clears `buffer_cache_` and
    /// cannot move `maxBufferLength()`, so no amount of pool rebuilding makes
    /// that request succeed. Shipping it as a pressure class charged every
    /// subsequent generation a cold pool to recover from a failure the pool can
    /// neither cause nor repair. Found in review of revision 1; the
    /// neighbouring negative in the test suite and the smoke's oversize phase
    /// exist to keep it out.
    public static let sharedCachePressureSignatures:
        [GenerationWorkerHealth.BackendFailureSignature] = [
            GenerationWorkerHealth.BackendFailureSignature(
                name: "metal-buffer-allocation-failed",
                requiredFragments: ["[malloc]", "Unable to allocate"])
        ]

    /// Classify a failed generation into the state it must give back.
    ///
    /// - Parameter description: the failure exactly as it will be reported to
    ///   the caller. Matched rather than typed, because MLX surfaces these as
    ///   opaque backend errors with nothing to switch on.
    public static func plan(after description: String) -> GenerationRecoveryPlan {
        var condemnedSignature: String?
        if case .workerInvalidated(let name) = GenerationWorkerHealth.classify(
            failure: description)
        {
            condemnedSignature = name
        }
        // A condemned backend rebuilds the pool too. The one recorded
        // condemnation *is* pool exhaustion, and returning the buffers before
        // the supervisor replaces the process is the difference between the
        // replacement loading weights into a freed host and racing a corpse
        // for them. It rebuilds *later*, though: the weights are still held by
        // the request that is unwinding, so a clear issued here would be
        // undone by the model landing in the pool moments afterwards.
        let rebuild: GenerationRecoveryPlan.SharedCacheRebuild
        if condemnedSignature != nil {
            rebuild = .afterWorkerTeardown
        } else if sharedCachePressureSignatures.contains(where: { $0.matches(description) }) {
            rebuild = .immediate
        } else {
            rebuild = .none
        }
        return GenerationRecoveryPlan(
            releasesBatch: true,
            sharedCacheRebuild: rebuild,
            condemnedSignature: condemnedSignature)
    }

    /// How many times the deferred teardown may wait for the condemned worker's
    /// weights before it gives up.
    ///
    /// Bounded because the wait is on a `deinit`-scheduled task in a process
    /// the supervisor is already replacing; an unbounded wait would be a
    /// non-terminating task in a runtime that is trying to die. Retries rather
    /// than a single longer wait so each attempt re-reads the barrier and the
    /// event log carries the attempt that finally observed it.
    public static let workerTeardownAttempts = 3

    /// How many consecutive readings, taken one poll apart, must report the
    /// same `activeMemory` before the allocator is treated as at rest.
    ///
    /// A sampling parameter, not a memory threshold: it says nothing about how
    /// many bytes are acceptable, only that the number has stopped moving.
    /// Needed because a destruction that is *still running* produces a falling
    /// count, and any single reading taken during that fall is a partial
    /// release being read as a finished one — which is the shape of the
    /// revision-3 defect one level down. Greater than one on purpose: a bound
    /// of one would be satisfied by the first sample and assert nothing.
    public static let minimumStableActiveSamples = 3

    /// The only residue a completed release may be attested over, in bytes.
    ///
    /// **Zero, and zero on purpose.**
    ///
    /// Every earlier revision of this gate admitted a *band* of residue and
    /// review found a production input inside it, five rounds running. The
    /// band was `activeBytes < weightFootprintBytes` last time, and a strict
    /// subset of the model's own parameter arrays — copied out, so every
    /// `Module` dies and ownership reports the model released — sat inside it
    /// at 255,724,192 B of 262,361,760 B and was attested as a completed
    /// release. Moving the threshold again would only move the band.
    ///
    /// So the band is removed rather than narrowed. MLX's allocator counters
    /// are process-global: `activeMemory` is a single number for the whole
    /// process and cannot say whose bytes it is counting. That makes *every*
    /// non-zero allowance `A` an admission that some weight buffer of `A`
    /// bytes or fewer may be hiding underneath it — there is no reading that
    /// distinguishes `A` bytes of sampler state from `A` bytes of retained
    /// weights, and "the smallest tensor in this model is bigger than `A`" is
    /// a claim about one checkpoint, not about the runtime.
    ///
    /// At zero the question does not arise. `activeMemory == 0` means no MLX
    /// buffer of any kind is alive in the process, so no *weight* buffer is
    /// alive in it either. It is the one reading a process-global counter can
    /// carry an attribution claim on, because it leaves nothing to attribute.
    ///
    /// The cost is stated rather than engineered around: this runtime's own
    /// clean condemned teardown leaves 2,720 B of post-generation MLX state
    /// (sampler and RNG arrays) active, so it does **not** reach zero and it
    /// abandons like every other condemned teardown. A clean-path rebuild is
    /// therefore essentially never attested here. That is the intended
    /// outcome: abandoning re-announces the supervision marker and the
    /// supervisor replaces the process, which returns every byte the pool was
    /// holding — while a false attestation tells an operator the host is free
    /// with a condemned model still on it. The first costs a restart that was
    /// already going to happen. The second is the incident.
    ///
    /// Kept as a named constant with a `<=` comparison rather than folded into
    /// an `== 0`, so that the quantity being claimed — *the maximum residue
    /// this runtime will attest a release over* — is one greppable number with
    /// a negative test on either side of it, and so raising it is a visible
    /// edit rather than an emergent property of an inequality.
    public static let residualNonWeightAllowanceBytes = 0

    /// One reading of a condemned worker's weight state, taken by
    /// `WeightReleaseBarrier` and judged by ``weightsReleased(_:)``.
    ///
    /// A record rather than a bare `Bool` because review defeated the bare
    /// `Bool` twice, one level apart.
    ///
    /// Revision 3 answered "are the weights gone?" from a single `weak`
    /// reference to the outer `ModelContainer`, and a narrowed production
    /// mutant — three seconds of delay in the pinned
    /// `SerialAccessContainer<ModelContext>` destruction, nothing else touched
    /// — showed the runtime attesting a completed rebuild with the whole
    /// 262,361,760-byte model still active.
    ///
    /// Revision 4 replaced that with a *process-global* byte delta,
    /// `baselineActiveBytes - activeBytes >= weightFootprintBytes`, and review
    /// defeated that too: driving the same production path with a 6,000-word
    /// prompt made the failed request's own KV state larger than the model, so
    /// its release alone satisfied the subtraction. Two consecutive runs
    /// attested a completed rebuild — `returned_bytes` 608,909,592 against a
    /// 262,361,760 footprint — while post-teardown `active_bytes` sat at
    /// exactly 262,361,760: every weight still resident. A process-global
    /// delta cannot say *whose* bytes came back.
    ///
    /// So the fields below are what the current gate needs to answer that
    /// without attribution guesswork: who still owns weights, whether anything
    /// else is allocating, whether the allocator has come to rest, and what is
    /// *absolutely* still resident rather than what merely moved.
    public struct WeightReleaseObservation: Sendable, Equatable {
        /// Whether ARC has deallocated the exact `ModelContainer` this worker
        /// served from.
        ///
        /// Necessary and *not* sufficient. `ModelContainer` is a wrapper around
        /// `SerialAccessContainer<ModelContext>`, and the weights live two
        /// levels below it in `ModelContext.model`; a Swift `weak` reference
        /// may read `nil` while destruction of that stored state is still in
        /// progress. Review's scratch probe observed the order directly:
        /// `payload-deinit-start`, `weak-nil`, `payload-deinit-finish`.
        public let containerDeallocated: Bool

        /// How many weight-owning `Module` objects were registered when this
        /// model was loaded — the whole `model.modules()` tree, root included.
        ///
        /// `0` means the registry was never populated. That is a *failure to
        /// read*, not a model with no weights, and ``weightsReleased(_:)``
        /// refuses on it rather than treating "none live" as an attestation:
        /// an empty registry reports zero live owners forever.
        public let weightOwnerCount: Int

        /// How many of those objects are still alive, counted through `weak`
        /// references so counting cannot itself keep them alive.
        ///
        /// This is the *attribution* half, and it is what a process-global byte
        /// count cannot supply. Every weight array of this model is a stored
        /// property of one of these objects, so while any of them is alive some
        /// of this model's weights are certainly still owned — no matter how
        /// many unrelated bytes the process handed back in the meantime. It is
        /// also the only reading that separates "half the weights are retained"
        /// from "the weights are gone and half a gigabyte of request state came
        /// back", which are numerically identical to the allocator.
        public let liveWeightOwners: Int

        /// What this model's weights cost MLX when they were loaded, measured
        /// as the `activeMemory` delta across the load.
        ///
        /// Measured rather than configured: it is the only number that makes
        /// "the weights came back" checkable without a threshold somebody
        /// picked. `0` means the measurement is unavailable, which is a
        /// *failure to read* and never a licence to attest — see
        /// ``weightsReleased(_:)``.
        public let weightFootprintBytes: Int

        /// MLX's `activeMemory` at the moment the teardown began, with the
        /// model still held.
        ///
        /// Kept for the record and for one *necessary* condition below. It is
        /// no longer sufficient for anything: this is the contaminated baseline
        /// review defeated.
        public let baselineActiveBytes: Int

        /// MLX's `activeMemory` now.
        public let activeBytes: Int

        /// Generations the ledger still has open at the moment of the reading.
        ///
        /// Non-zero means something else is allocating and freeing while the
        /// teardown measures, so nothing this reading says about *whose* bytes
        /// are resident can be trusted. Review named that case directly:
        /// "concurrent in-flight allocation has the same attribution problem".
        /// The gate vetoes on it rather than guessing.
        public let generationsInFlight: Int

        /// How many consecutive readings, one poll apart, have reported this
        /// exact ``activeBytes``. `1` is the first reading of a value.
        ///
        /// A destruction in progress is a falling count; a finished one holds
        /// still. See ``minimumStableActiveSamples``.
        public let stableActiveSamples: Int

        public init(
            containerDeallocated: Bool,
            weightOwnerCount: Int,
            liveWeightOwners: Int,
            weightFootprintBytes: Int,
            baselineActiveBytes: Int,
            activeBytes: Int,
            generationsInFlight: Int,
            stableActiveSamples: Int
        ) {
            self.containerDeallocated = containerDeallocated
            self.weightOwnerCount = weightOwnerCount
            self.liveWeightOwners = liveWeightOwners
            self.weightFootprintBytes = weightFootprintBytes
            self.baselineActiveBytes = baselineActiveBytes
            self.activeBytes = activeBytes
            self.generationsInFlight = generationsInFlight
            self.stableActiveSamples = stableActiveSamples
        }

        /// How many bytes MLX has stopped calling active since the teardown
        /// began. Floored at zero: a *rise* is not a negative release.
        ///
        /// Reported, and used only as a *necessary* condition. On its own it is
        /// the revision-4 defect.
        public var returnedBytes: Int { max(0, baselineActiveBytes - activeBytes) }

        /// Whether every registered weight owner has been deallocated, from a
        /// registry that was actually populated.
        public var weightOwnersDeallocated: Bool {
            weightOwnerCount > 0 && liveWeightOwners == 0
        }
    }

    /// Whether a condemned worker's weights have actually been returned to MLX.
    ///
    /// Conjunctive, and deliberately so: every clause below covers a case that
    /// review reached through one of the others.
    ///
    /// 1. `containerDeallocated` — the cheap, exact veto. While the wrapper is
    ///    alive the weights are certainly still held, and no byte count may
    ///    out-vote that. Not an attestation: revision 3 shipped it as one.
    /// 2. `weightOwnersDeallocated` — the ATTRIBUTION clause. Every weight
    ///    array of this model is a stored property of one of the registered
    ///    `Module` objects, so "none of them is alive" is a statement about
    ///    *this model's* weights rather than about the process's byte total. A
    ///    registry that was never populated (`weightOwnerCount == 0`) fails
    ///    closed, because "zero live owners" would otherwise be true forever.
    /// 3. `weightFootprintBytes > 0` — an unmeasured footprint is a failure to
    ///    read, not a model that cost nothing.
    /// 4. `generationsInFlight == 0` — nothing else may be allocating while the
    ///    reading is taken.
    /// 5. `stableActiveSamples >= minimumStableActiveSamples` — the allocator
    ///    has come to rest, so a destruction still running is not mistaken for
    ///    one that finished.
    /// 6. `activeBytes <= residualNonWeightAllowanceBytes` — the ABSOLUTE
    ///    RESIDUE clause. It asks what is still resident, not what moved, and
    ///    it admits no band: the allowance is zero, so the only reading it
    ///    accepts is an allocator holding nothing at all. Revisions 4 and 5
    ///    compared the residue against the model's footprint instead, and
    ///    review walked a production input into the gap both times — most
    ///    recently a strict subset of the model's own parameter arrays,
    ///    copied out so that every `Module` died and ownership reported the
    ///    model released, sitting at 255,724,192 B of a 262,361,760 B
    ///    footprint. Any threshold above zero is a promise that nothing
    ///    weight-sized fits underneath it, and a process-global counter cannot
    ///    keep that promise. See ``residualNonWeightAllowanceBytes``.
    /// 7. `returnedBytes >= weightFootprintBytes` — kept from revision 4 as a
    ///    *necessary* condition. Not sufficient, and never again used alone:
    ///    it establishes that the fall happened inside this teardown window.
    ///
    /// What this cannot do, stated rather than papered over: MLX's counters are
    /// process-global, so no clause here can attribute an individual byte. The
    /// gate is therefore built to refuse everything it cannot account for —
    /// ownership is read directly from the model tree, the allocator must be
    /// idle and at rest before it is believed, and any residue at all is a
    /// refusal. The practical consequence, measured rather than assumed: this
    /// runtime's clean condemned teardown leaves 2,720 B of post-generation
    /// MLX state active and therefore abandons too, so a completed rebuild is
    /// essentially never attested on any path. That is the honest end of a
    /// process-global counter, and it is the cheaper end. A false abandonment
    /// costs a supervision marker and a replacement process the condemnation
    /// had already made necessary; a false attestation tells an operator the
    /// host is free while a condemned model is still holding it.
    public static func weightsReleased(_ observation: WeightReleaseObservation) -> Bool {
        guard observation.containerDeallocated else { return false }
        guard observation.weightOwnersDeallocated else { return false }
        guard observation.weightFootprintBytes > 0 else { return false }
        guard observation.generationsInFlight == 0 else { return false }
        guard observation.stableActiveSamples >= minimumStableActiveSamples else { return false }
        guard observation.activeBytes <= residualNonWeightAllowanceBytes else { return false }
        return observation.returnedBytes >= observation.weightFootprintBytes
    }

    /// What a teardown attempt is allowed to conclude.
    ///
    /// Split out as a pure function because it is a gate, and the gate is the
    /// part that was wrong: revision 2 discarded the release observation and
    /// took the success transition either way. Everything this returns is
    /// decided from `releaseObserved`, so a caller cannot reach ``rebuilt``
    /// without having seen the release.
    ///
    /// - Parameters:
    ///   - releaseObserved: whether the condemned worker's weights were
    ///     observed to be returned. Not "probably", not "the wrapper is gone":
    ///     the caller derives it from ``weightsReleased(_:)``, which requires
    ///     both the container's destruction and MLX's own allocator giving back
    ///     at least what this model's weights cost to load.
    ///   - attempt: 1-based attempt number.
    ///   - maxAttempts: the bound, normally ``workerTeardownAttempts``.
    public static func teardownOutcome(
        releaseObserved: Bool, attempt: Int, maxAttempts: Int
    ) -> WorkerTeardownOutcome {
        guard releaseObserved else {
            // Fail closed. An unobserved release is not a release, and the two
            // must not share a transition: the buffers this clear would have
            // returned are still held, so clearing now empties a pool the
            // condemned model is about to refill, and attesting a completed
            // rebuild tells an operator the host is free when it is not.
            return attempt >= maxAttempts ? .abandoned : .retry
        }
        return .rebuilt
    }
}

/// The result of one attempt to discharge a condemned worker's deferred pool
/// rebuild.
public enum WorkerTeardownOutcome: Sendable, Equatable {
    /// The weights were observed released and the pool was dropped.
    case rebuilt
    /// The release was not observed and attempts remain.
    case retry
    /// The release was never observed and the bound expired. The rebuild is
    /// **not** performed and **not** attested; the pending flag stays raised.
    case abandoned
}

/// Accounting for in-flight generations and the state each one holds.
///
/// A value type on purpose: the arithmetic is the contract, and it is worth
/// testing without an actor, a model or a port. The runtime keeps exactly one
/// of these, and it outlives the engine — after the worker is condemned the
/// engine is dropped, and "did the dead worker leak its batch" is a question
/// that only becomes interesting *after* there is no engine left to ask.
public struct GenerationBatchLedger: Sendable, Equatable {
    /// A handle to one in-flight generation.
    ///
    /// Opaque and non-reusable. Slot identity is what lets ``fail(_:observing:)``
    /// refuse to close a generation twice; an index into a pool would be
    /// recycled and a late second failure would silently close somebody else's
    /// live generation.
    public struct Slot: Sendable, Equatable, Hashable {
        public let id: Int
        fileprivate init(id: Int) { self.id = id }
    }

    private var open: Set<Int> = []
    private var nextID = 1

    /// Generations begun.
    public private(set) var started = 0
    /// Generations that ran to a completion packet.
    public private(set) var completed = 0
    /// Generations that ended in an error, of any verdict.
    public private(set) var failed = 0
    /// Batch entries released after a failure.
    ///
    /// Tracked separately from ``failed`` rather than derived from it. Equal
    /// counts are the invariant being checked, so deriving one from the other
    /// would make the check unfalsifiable: a runtime that forgot to release
    /// would still report that it had.
    public private(set) var batchesReleased = 0
    /// Times the shared MLX buffer pool was actually dropped.
    ///
    /// Incremented by ``recordSharedCacheRebuild()`` at the moment the clear is
    /// performed, never by ``fail(_:observing:)`` when the clear is merely
    /// *planned*. Review defeated the earlier arrangement with a one-line
    /// mutant: deleting the production `Memory.clearCache()` call left this
    /// counter, the emitted event and all 63 smoke checks intact while the
    /// allocator went on holding 67,955,820 cache bytes. A counter incremented
    /// beside the action rather than by it is self-minted evidence.
    public private(set) var sharedCacheRebuilds = 0

    /// Deferred rebuilds that were never carried out because the condemned
    /// worker's weights were never observed released.
    ///
    /// Counted, and counted *separately*, because it is the failure this whole
    /// accounting exists to make visible. Folding it into
    /// ``sharedCacheRebuilds`` would restore the exact shape review rejected —
    /// a failure to observe taking the same transition as an observation.
    public private(set) var sharedCacheRebuildsAbandoned = 0

    /// A condemned worker's pool rebuild that has been planned but not yet
    /// performed.
    ///
    /// Raised by ``deferSharedCacheRebuild()``, lowered *only* by
    /// ``recordSharedCacheRebuild()`` — that is, only by the clear actually
    /// happening. ``abandonSharedCacheRebuild()`` deliberately leaves it
    /// raised: the pool is still owed, and a report that lowered the flag on
    /// giving up would describe an abandoned rebuild the same way it describes
    /// a finished one.
    public private(set) var sharedCacheRebuildPending = false

    public init() {}

    /// Generations currently in flight.
    ///
    /// The number that must return to zero once the traffic stops. A leak shows
    /// up here and nowhere else — every other counter is monotonic and a
    /// forgotten release leaves them all looking plausible.
    public var active: Int { open.count }

    public mutating func begin() -> Slot {
        let slot = Slot(id: nextID)
        nextID += 1
        open.insert(slot.id)
        started += 1
        return slot
    }

    /// Close a generation that produced its completion packet.
    ///
    /// - Returns: `false` when the slot is not open, leaving every counter
    ///   untouched.
    @discardableResult
    public mutating func finish(_ slot: Slot) -> Bool {
        guard open.remove(slot.id) != nil else { return false }
        completed += 1
        return true
    }

    /// Close a generation that failed, and say what its state owes back.
    ///
    /// - Returns: `nil` when the slot is not open. A generation is closed once:
    ///   a second failure arriving for a slot that already fell over — an
    ///   `emit` throwing while unwinding from the error that caused it — must
    ///   not be counted, must not release a batch that is already gone, and
    ///   must not clear the shared pool a second time.
    public mutating func fail(_ slot: Slot, observing description: String)
        -> GenerationRecoveryPlan?
    {
        guard open.remove(slot.id) != nil else { return nil }
        failed += 1
        let plan = GenerationBatchRecovery.plan(after: description)
        if plan.releasesBatch { batchesReleased += 1 }
        // `sharedCacheRebuilds` is deliberately NOT touched here. The plan says
        // the pool should be dropped; only whoever drops it may say that it
        // was. See ``recordSharedCacheRebuild()``.
        return plan
    }

    /// Record that the shared buffer pool was just dropped.
    ///
    /// Called by the owner of the `Memory.clearCache()` call, immediately after
    /// it returns, so the count cannot outrun the action it claims.
    public mutating func recordSharedCacheRebuild() {
        sharedCacheRebuilds += 1
        sharedCacheRebuildPending = false
    }

    /// Record that a condemned worker's pool rebuild is owed but not yet due.
    public mutating func deferSharedCacheRebuild() {
        sharedCacheRebuildPending = true
    }

    /// Record that a deferred rebuild was given up on without being performed.
    ///
    /// Increments nothing that could be read as success and leaves
    /// ``sharedCacheRebuildPending`` raised, so `GET /debug/generation-state`
    /// keeps reporting the pool as owed for as long as the process lives.
    public mutating func abandonSharedCacheRebuild() {
        sharedCacheRebuildsAbandoned += 1
    }
}

/// The `GET /debug/generation-state` answer.
///
/// Exists because "the batch was released" is otherwise only assertable from a
/// log line, and a log line records that a release was *announced*. This
/// records that nothing is still held, at whatever moment the operator asks.
///
/// Always `200`, including once the worker is condemned. The endpoint reports
/// state rather than serving capacity, and mirroring `/health`'s `503` here
/// would black it out at exactly the moment its answer matters — which is also
/// why the ledger is kept outside the engine that condemnation drops.
///
/// Production call site: `Router.route(method:path:body:)` in the
/// `mlx-swift-runtime-prototype` executable target.
public struct GenerationBatchReport: Sendable, Equatable {
    /// MLX allocator counters, passed in rather than read here so this type
    /// stays free of the MLX dependency and testable under `swift test`.
    public struct MemoryUsage: Sendable, Equatable {
        public let activeBytes: Int
        public let cacheBytes: Int
        public let peakBytes: Int

        public init(activeBytes: Int, cacheBytes: Int, peakBytes: Int) {
            self.activeBytes = activeBytes
            self.cacheBytes = cacheBytes
            self.peakBytes = peakBytes
        }
    }

    public let status: Int
    public let body: JSONValue

    public static func make(
        readiness: RuntimeReadiness,
        ledger: GenerationBatchLedger,
        memory: MemoryUsage?
    ) -> GenerationBatchReport {
        var fields: [String: JSONValue] = [
            "readiness": .string(label(readiness)),
            "batch": .object([
                "active": .int(ledger.active),
                "started": .int(ledger.started),
                "completed": .int(ledger.completed),
                "failed": .int(ledger.failed),
                "batches_released": .int(ledger.batchesReleased),
                "shared_cache_rebuilds": .int(ledger.sharedCacheRebuilds),
                // Published, not merely logged. A rebuild that was owed and
                // never performed is a property of the runtime's current
                // state, and an operator deciding whether this host can take a
                // replacement process has to be able to ask for it rather than
                // reconstruct it from a stream of events.
                "shared_cache_rebuilds_abandoned": .int(ledger.sharedCacheRebuildsAbandoned),
                "shared_cache_rebuild_pending": .bool(ledger.sharedCacheRebuildPending),
            ]),
        ]
        // Absent rather than zero when unavailable. Zeroed allocator counters
        // are a reading a caller would act on; a missing key is not.
        fields["mlx"] =
            memory.map {
                .object([
                    "active_bytes": .int($0.activeBytes),
                    "cache_bytes": .int($0.cacheBytes),
                    "peak_bytes": .int($0.peakBytes),
                ])
            } ?? .null
        return GenerationBatchReport(status: 200, body: .object(fields))
    }

    private static func label(_ readiness: RuntimeReadiness) -> String {
        switch readiness {
        case .loading: return "loading"
        case .ready: return "ready"
        case .failed: return "failed"
        case .generationWorkerFailed: return "generation_worker_failed"
        case .shuttingDown: return "shutting_down"
        }
    }
}
