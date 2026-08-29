import Foundation
import Testing

@testable import MLXSwiftRuntimeContract

@Suite("generation batch recovery plan")
struct GenerationBatchRecoveryPlanTests {
    /// Verbatim from the `BUG-260827-1jhv2g` incident record.
    static let incident = "RuntimeError: [metal::malloc] Resource limit (499000) exceeded"
    /// The allocator giving up *after* its own partial reclaim. The one class
    /// where clearing `buffer_cache_` can change the next attempt.
    static let allocationFailed =
        "RuntimeError: [malloc] Unable to allocate 268435456 bytes."
    /// `allocator.cpp` line ~111: thrown before the cache lock is taken, on
    /// `size > device_->maxBufferLength()`. Reads like pressure, is not.
    static let oversize =
        "RuntimeError: [metal::malloc] Attempting to allocate 4294967296 bytes which is "
        + "greater than the maximum allowed buffer size"
    /// A failure that belongs entirely to the request.
    static let requestScoped = "generation ended without completion info; token usage is unknown"

    // MARK: - The always-true half

    @Test(
        "every failure releases its batch, whatever the verdict",
        arguments: [
            incident, oversize, allocationFailed, requestScoped, "",
            "something nobody has ever seen",
        ])
    func alwaysReleasesBatch(description: String) {
        // The per-request KV cache belongs to the request. Any verdict under
        // which it survived the request would be the recorded leak.
        #expect(GenerationBatchRecovery.plan(after: description).releasesBatch)
    }

    // MARK: - The conditional half

    @Test("an ordinary request-scoped failure does not touch the shared pool")
    func requestScopedLeavesSharedCacheAlone() {
        let plan = GenerationBatchRecovery.plan(after: Self.requestScoped)
        // NARROWING. Everything else in this suite is satisfied by a runtime
        // that clears the shared pool on every error, which would pay a
        // cold-pool latency cost on every subsequent generation for a failure
        // that never touched the pool. Only this expectation separates
        // "rebuilds when implicated" from "rebuilds always".
        #expect(!plan.rebuildsSharedCache)
        #expect(!plan.condemnsWorker)
        #expect(plan.condemnedSignature == nil)
    }

    @Test("an exhausted allocation rebuilds the shared pool immediately but keeps the worker")
    func allocationFailureRebuildsWithoutCondemning() {
        let plan = GenerationBatchRecovery.plan(after: Self.allocationFailed)
        // `[malloc] Unable to allocate` is thrown only after the allocator's
        // own `release_cached_buffers` slice already failed to make room.
        // `clear_cache()` empties the pool outright, so it can hand back what
        // that partial reclaim kept -- the one case where a rebuild changes
        // the next attempt's outcome.
        #expect(plan.sharedCacheRebuild == .immediate)
        // The engine survives, so the clear happens while the request unwinds
        // rather than waiting for a teardown that is never coming.
        #expect(!plan.condemnsWorker)
    }

    @Test("an oversize rejection does not rebuild the pool, because the pool cannot fix it")
    func oversizeIsNotSharedCachePressure() {
        // NARROWING, and the revision-1 defect. `MetalAllocator::malloc` throws
        // this before `std::unique_lock lk(mutex_)`, comparing against
        // `device_->maxBufferLength()`. `clear_cache()` only empties
        // `buffer_cache_`; it cannot move `maxBufferLength()`, so rebuilding
        // the pool cannot make the same request succeed. Shipping it as a
        // pressure class charged every later generation a cold pool to recover
        // from a failure the pool can neither cause nor repair.
        let plan = GenerationBatchRecovery.plan(after: Self.oversize)
        #expect(plan.sharedCacheRebuild == .none)
        #expect(!plan.rebuildsSharedCache)
        // Still request-scoped: it rejects one oversized buffer and leaves the
        // backend able to serve.
        #expect(!plan.condemnsWorker)
        #expect(plan.releasesBatch)
    }

    @Test("the recorded exhaustion condemns the worker and rebuilds the pool")
    func incidentCondemnsAndRebuilds() {
        let plan = GenerationBatchRecovery.plan(after: Self.incident)
        #expect(plan.condemnedSignature == "metal-allocator-resource-limit")
        // Deferred, not immediate. The weights are still held by the request
        // that is unwinding; clearing now empties the pool moments before the
        // whole model lands back in it.
        #expect(plan.sharedCacheRebuild == .afterWorkerTeardown)
        #expect(plan.releasesBatch)
    }

    @Test("a condemned shader-library failure still releases and rebuilds")
    func shaderLibraryCondemns() {
        let plan = GenerationBatchRecovery.plan(
            after: "MLXError: Failed to load the default metallib after 4 lookups")
        #expect(plan.condemnedSignature == "metal-shader-library-unreachable")
        #expect(plan.sharedCacheRebuild == .afterWorkerTeardown)
        #expect(plan.releasesBatch)
    }

    // MARK: - Narrowing the pressure signature

    @Test(
        "a message carrying only one fragment of the pressure signature does not rebuild",
        arguments: [
            // The allocator tag alone.
            "RuntimeError: [malloc] something else went wrong",
            // The prose alone, from a message with nothing to do with the
            // allocator.
            "RequestError: Unable to allocate a tool call id",
            // The condemning signature's tag. `[metal::malloc]` must not be
            // read as carrying `[malloc]`: if it were, every condemnation
            // would also match the *immediate* pressure class and clear the
            // pool at the exact moment the deferral exists to avoid.
            "RuntimeError: [metal::malloc] Unable to allocate 1 bytes.",
        ])
    func pressureSignatureIsConjunctive(description: String) {
        let plan = GenerationBatchRecovery.plan(after: description)
        #expect(!plan.rebuildsSharedCache)
        #expect(!plan.condemnsWorker)
    }

    @Test("a pressure signature that lost its fragments matches nothing")
    func emptyPressureSignatureFailsClosed() {
        // Same fail-closed rule as GenerationWorkerHealth: `allSatisfy` is
        // vacuously true on an empty sequence, so a signature stripped by a bad
        // edit would otherwise rebuild the pool on literally every failure.
        let signature = GenerationWorkerHealth.BackendFailureSignature(
            name: "stripped", requiredFragments: [])
        #expect(!signature.matches(Self.allocationFailed))
        #expect(!signature.matches(""))
    }

    @Test("the pressure list does not overlap the condemning list")
    func pressureAndCondemnationStayDistinct() {
        // The two lists are read by different gates with opposite consequences
        // — keep serving versus take out of rotation. A name appearing in both
        // would mean one message with two verdicts.
        let condemning = Set(GenerationWorkerHealth.invalidatingSignatures.map(\.name))
        let pressure = Set(GenerationBatchRecovery.sharedCachePressureSignatures.map(\.name))
        #expect(condemning.isDisjoint(with: pressure))
    }

    @Test("the plan's condemnation verdict is the health gate's verdict")
    func condemnationDelegatesToTheHealthGate() {
        // Pins the two classifiers together. The readiness transition is owned
        // by GenerationWorkerHealth alone; if this plan started reporting a
        // condemnation the health gate does not make (or missing one it does),
        // the released-batch record would name a reason that never reached
        // /health.
        for description in [
            Self.incident, Self.allocationFailed, Self.requestScoped,
            "MLXError: Failed to load the default metallib after 4 lookups",
            "RequestError: Resource limit for this request is 8 tokens", "",
        ] {
            let plan = GenerationBatchRecovery.plan(after: description)
            switch GenerationWorkerHealth.classify(failure: description) {
            case .requestScoped:
                #expect(plan.condemnedSignature == nil, "\(description)")
            case .workerInvalidated(let signature):
                #expect(plan.condemnedSignature == signature, "\(description)")
            }
        }
    }

    @Test("the over-broad neighbour is neither condemned nor rebuilt")
    func overbroadNeighbourStaysRequestScoped() {
        // The revision-1 defect message from `TASK-260827-2h39ya`. It carries
        // `Resource limit` and no allocator context; a second gate that matched
        // it would reintroduce the same regression through a new door.
        let plan = GenerationBatchRecovery.plan(
            after: "RequestError: Resource limit for this request is 8 tokens")
        #expect(!plan.condemnsWorker)
        #expect(!plan.rebuildsSharedCache)
    }
}

@Suite("generation batch ledger")
struct GenerationBatchLedgerTests {
    static let requestScoped = "generation ended without completion info; token usage is unknown"
    static let allocationFailed = "RuntimeError: [malloc] Unable to allocate 268435456 bytes."

    @Test("a fresh ledger holds nothing")
    func startsEmpty() {
        let ledger = GenerationBatchLedger()
        #expect(ledger.active == 0)
        #expect(ledger.started == 0)
        #expect(ledger.completed == 0)
        #expect(ledger.failed == 0)
        #expect(ledger.batchesReleased == 0)
        #expect(ledger.sharedCacheRebuilds == 0)
    }

    @Test("a completed generation leaves nothing in flight")
    func completionClosesTheSlot() {
        var ledger = GenerationBatchLedger()
        let slot = ledger.begin()
        #expect(ledger.active == 1)
        let closed = ledger.finish(slot)
        #expect(closed)
        #expect(ledger.active == 0)
        #expect(ledger.completed == 1)
        // Nothing failed, so nothing was released. A ledger that counted a
        // release here would report a leak-free runtime by construction.
        #expect(ledger.batchesReleased == 0)
    }

    @Test("a failed generation leaves nothing in flight and records the release")
    func failureClosesTheSlotAndReleases() {
        var ledger = GenerationBatchLedger()
        let slot = ledger.begin()
        let plan = ledger.fail(slot, observing: Self.requestScoped)
        #expect(plan?.releasesBatch == true)
        // THE INVARIANT. `active` is the only non-monotonic counter, so a
        // runtime that failed a generation without giving the slot back shows
        // up here and nowhere else.
        #expect(ledger.active == 0)
        #expect(ledger.failed == 1)
        #expect(ledger.batchesReleased == 1)
        #expect(ledger.sharedCacheRebuilds == 0)
    }

    @Test("planning a rebuild does not count one")
    func planningARebuildDoesNotCountOne() {
        // THE SELF-MINTED-EVIDENCE NEGATIVE. Review deleted the production
        // `Memory.clearCache()` call and the old ledger went on reporting
        // `shared_cache_rebuilds=1` while MLX held 67,955,820 cache bytes. The
        // plan may say the pool *should* be dropped; only whoever drops it may
        // say that it *was*.
        var ledger = GenerationBatchLedger()
        let slot = ledger.begin()
        let plan = ledger.fail(slot, observing: Self.allocationFailed)
        #expect(plan?.sharedCacheRebuild == .immediate)
        #expect(ledger.batchesReleased == 1)
        #expect(ledger.sharedCacheRebuilds == 0)
    }

    @Test("a rebuild is counted when it is performed")
    func performedRebuildIsCounted() {
        var ledger = GenerationBatchLedger()
        let slot = ledger.begin()
        _ = ledger.fail(slot, observing: Self.allocationFailed)
        ledger.recordSharedCacheRebuild()
        #expect(ledger.sharedCacheRebuilds == 1)
        #expect(ledger.batchesReleased == 1)
    }

    @Test("a condemned failure does not count a rebuild until the teardown performs one")
    func condemnedRebuildIsDeferred() {
        var ledger = GenerationBatchLedger()
        let slot = ledger.begin()
        let plan = ledger.fail(
            slot, observing: "RuntimeError: [metal::malloc] Resource limit (499000) exceeded")
        #expect(plan?.sharedCacheRebuild == .afterWorkerTeardown)
        // Nothing has been cleared yet: the request that observed the failure
        // still holds the weights.
        #expect(ledger.sharedCacheRebuilds == 0)
        ledger.recordSharedCacheRebuild()
        #expect(ledger.sharedCacheRebuilds == 1)
    }

    @Test("a second failure for the same slot changes nothing")
    func doubleFailIsRefused() {
        var ledger = GenerationBatchLedger()
        let slot = ledger.begin()
        _ = ledger.fail(slot, observing: Self.allocationFailed)
        let after = ledger
        // An `emit` throwing while the runtime unwinds from the error that
        // caused it arrives here twice for one generation. Counting it twice
        // would inflate `batchesReleased` past `failed` and clear the shared
        // pool a second time for a batch that no longer exists.
        let second = ledger.fail(slot, observing: Self.allocationFailed)
        #expect(second == nil)
        #expect(ledger == after)
    }

    @Test("finishing an already-failed slot changes nothing")
    func finishAfterFailIsRefused() {
        var ledger = GenerationBatchLedger()
        let slot = ledger.begin()
        _ = ledger.fail(slot, observing: Self.requestScoped)
        let after = ledger
        let reclosed = ledger.finish(slot)
        #expect(!reclosed)
        #expect(ledger == after)
        // Specifically: the failure was not laundered into a completion.
        #expect(ledger.completed == 0)
    }

    @Test("a slot from another ledger is not honoured")
    func foreignSlotIsRefused() {
        var issuing = GenerationBatchLedger()
        let slot = issuing.begin()
        var other = GenerationBatchLedger()
        let untouched = other
        let foreignFail = other.fail(slot, observing: Self.requestScoped)
        #expect(foreignFail == nil)
        let foreignFinish = other.finish(slot)
        #expect(!foreignFinish)
        #expect(other == untouched)
    }

    @Test("slot identity is not recycled")
    func slotsAreNotRecycled() {
        var ledger = GenerationBatchLedger()
        let first = ledger.begin()
        _ = ledger.finish(first)
        let second = ledger.begin()
        // If the second generation reused the first's id, a late failure from
        // the first would close the second — a live generation silently
        // released while it is still producing tokens.
        #expect(first != second)
        let stale = ledger.fail(first, observing: Self.requestScoped)
        #expect(stale == nil)
        #expect(ledger.active == 1)
    }

    @Test("concurrent generations are accounted for independently")
    func interleavedSlots() {
        var ledger = GenerationBatchLedger()
        let a = ledger.begin()
        let b = ledger.begin()
        #expect(ledger.active == 2)
        _ = ledger.fail(a, observing: Self.requestScoped)
        #expect(ledger.active == 1)
        let closedB = ledger.finish(b)
        #expect(closedB)
        #expect(ledger.active == 0)
        #expect(ledger.started == 2)
        #expect(ledger.completed == 1)
        #expect(ledger.failed == 1)
        #expect(ledger.batchesReleased == 1)
    }

    @Test("active is exactly what has not been closed")
    func activeMatchesTheArithmetic() {
        var ledger = GenerationBatchLedger()
        var open: [GenerationBatchLedger.Slot] = []
        for index in 0 ..< 12 {
            let slot = ledger.begin()
            if index % 3 == 0 {
                _ = ledger.fail(slot, observing: Self.requestScoped)
            } else if index % 3 == 1 {
                _ = ledger.finish(slot)
            } else {
                open.append(slot)
            }
        }
        #expect(ledger.active == open.count)
        #expect(ledger.active == ledger.started - ledger.completed - ledger.failed)
        #expect(ledger.batchesReleased == ledger.failed)
    }
}

@Suite("GET /debug/generation-state")
struct GenerationBatchReportTests {
    static func body(
        readiness: RuntimeReadiness = .ready,
        ledger: GenerationBatchLedger = GenerationBatchLedger(),
        memory: GenerationBatchReport.MemoryUsage? = nil
    ) -> [String: JSONValue] {
        let report = GenerationBatchReport.make(
            readiness: readiness, ledger: ledger, memory: memory)
        #expect(report.status == 200)
        guard case .object(let fields) = report.body else {
            Issue.record("report body is not an object")
            return [:]
        }
        return fields
    }

    @Test("a condemned runtime still answers 200 with its state")
    func condemnedRuntimeStillReports() {
        // The endpoint reports state, not serving capacity. Mirroring
        // /health's 503 here would black it out at exactly the moment its
        // answer matters — right after the worker was condemned, when the
        // question is what the dead worker gave back.
        var ledger = GenerationBatchLedger()
        let slot = ledger.begin()
        _ = ledger.fail(
            slot, observing: "RuntimeError: [metal::malloc] Resource limit (499000) exceeded")
        // The teardown has run by the time an operator asks: the engine is gone
        // and the pool was cleared after it, which is what makes the reported
        // rebuild a measurement rather than an intention.
        ledger.recordSharedCacheRebuild()
        let fields = Self.body(
            readiness: .generationWorkerFailed("condemned"), ledger: ledger)
        #expect(fields["readiness"] == .string("generation_worker_failed"))
        guard case .object(let batch) = fields["batch"] else {
            Issue.record("no batch object")
            return
        }
        #expect(batch["active"] == .int(0))
        #expect(batch["failed"] == .int(1))
        #expect(batch["batches_released"] == .int(1))
        #expect(batch["shared_cache_rebuilds"] == .int(1))
    }

    @Test(
        "readiness is labelled for every case",
        arguments: [
            (RuntimeReadiness.loading, "loading"),
            (.ready, "ready"),
            (.failed("boom"), "failed"),
            (.generationWorkerFailed("boom"), "generation_worker_failed"),
            (.shuttingDown, "shutting_down"),
        ])
    func labelsReadiness(readiness: RuntimeReadiness, label: String) {
        #expect(Self.body(readiness: readiness)["readiness"] == .string(label))
    }

    @Test("allocator figures are absent, not zero, before MLX has been driven")
    func memoryIsNullWhenUnavailable() {
        // A zeroed reading is a number somebody would size a host from. A null
        // is not. Absence and a low value are different facts.
        #expect(Self.body(memory: nil)["mlx"] == .null)
    }

    @Test("allocator figures are published when available")
    func memoryIsPublishedWhenAvailable() {
        let fields = Self.body(
            memory: GenerationBatchReport.MemoryUsage(
                activeBytes: 11, cacheBytes: 22, peakBytes: 33))
        #expect(
            fields["mlx"]
                == .object([
                    "active_bytes": .int(11), "cache_bytes": .int(22), "peak_bytes": .int(33),
                ]))
    }
}

@Suite("bounded generation fault injection seam")
struct BoundedFaultInjectionTests {
    static let base = ["serve", "--model", "/models/qwen", "--port", "18011"]
    static let message = "RuntimeError: [metal::malloc] Resource limit (499000) exceeded"

    static func serve(_ extra: [String]) -> [String] { base + extra }

    @Test("both bounds are unset unless asked for")
    func defaultsPreserveTheTerminalSeam() throws {
        // The dead-generation-worker suite depends on this: that regression is
        // terminal, so an injection that silently healed after one request
        // would be pinning a different runtime.
        let (_, options) = try RuntimeOptions.parse(
            arguments: Self.serve(["--fault-inject-generation-error", Self.message]))
        #expect(options.faultInjectedGenerationErrorCount == nil)
        #expect(options.faultInjectedGenerationErrorAfterTokens == 0)
    }

    @Test("a bounded, deferred injection parses")
    func parsesBounds() throws {
        let (_, options) = try RuntimeOptions.parse(
            arguments: Self.serve([
                "--fault-inject-generation-error", Self.message,
                "--fault-inject-generation-error-count", "2",
                "--fault-inject-generation-error-after-tokens", "4",
            ]))
        #expect(options.faultInjectedGenerationErrorCount == 2)
        #expect(options.faultInjectedGenerationErrorAfterTokens == 4)
    }

    @Test(
        "a bound without an injection to bound is refused",
        arguments: [
            "--fault-inject-generation-error-count",
            "--fault-inject-generation-error-after-tokens",
        ])
    func refusesModifierWithoutInjection(flag: String) {
        // NEGATIVE. Accepted-and-ignored is the dangerous outcome here: an
        // acceptance run that meant to inject a fault and mistyped the message
        // flag would watch a runtime that never fails and read that as
        // recovery.
        #expect(throws: RuntimeOptionsError.faultInjectionModifierWithoutInjection(flag)) {
            try RuntimeOptions.parse(arguments: Self.serve([flag, "1"]))
        }
    }

    @Test("a zero or negative count is refused", arguments: ["0", "-1"])
    func refusesNonPositiveCount(value: String) {
        // A count of zero arms and disarms the seam in one argv. Whatever it
        // was meant to say, the runtime it produces is indistinguishable from
        // one with no seam at all.
        #expect(
            throws: RuntimeOptionsError.nonPositiveInteger(
                flag: "--fault-inject-generation-error-count", value: Int(value)!)
        ) {
            try RuntimeOptions.parse(
                arguments: Self.serve([
                    "--fault-inject-generation-error", Self.message,
                    "--fault-inject-generation-error-count", value,
                ]))
        }
    }

    @Test("a negative token threshold is refused")
    func refusesNegativeThreshold() {
        #expect(
            throws: RuntimeOptionsError.negativeInteger(
                flag: "--fault-inject-generation-error-after-tokens", value: -1)
        ) {
            try RuntimeOptions.parse(
                arguments: Self.serve([
                    "--fault-inject-generation-error", Self.message,
                    "--fault-inject-generation-error-after-tokens", "-1",
                ]))
        }
    }

    @Test("a zero token threshold is accepted and means immediately")
    func acceptsZeroThreshold() throws {
        let (_, options) = try RuntimeOptions.parse(
            arguments: Self.serve([
                "--fault-inject-generation-error", Self.message,
                "--fault-inject-generation-error-after-tokens", "0",
            ]))
        #expect(options.faultInjectedGenerationErrorAfterTokens == 0)
    }

    @Test(
        "a non-integer bound is refused",
        arguments: [
            "--fault-inject-generation-error-count",
            "--fault-inject-generation-error-after-tokens",
        ])
    func refusesNonIntegerBound(flag: String) {
        #expect(throws: RuntimeOptionsError.invalidInteger(flag: flag, value: "lots")) {
            try RuntimeOptions.parse(
                arguments: Self.serve([
                    "--fault-inject-generation-error", Self.message, flag, "lots",
                ]))
        }
    }

    @Test(
        "a repeated bound is refused",
        arguments: [
            "--fault-inject-generation-error-count",
            "--fault-inject-generation-error-after-tokens",
        ])
    func refusesDuplicateBound(flag: String) {
        #expect(throws: RuntimeOptionsError.duplicateFlag(flag)) {
            try RuntimeOptions.parse(
                arguments: Self.serve([
                    "--fault-inject-generation-error", Self.message, flag, "1", flag, "2",
                ]))
        }
    }
}

@Suite("condemned worker teardown gate")
struct WorkerTeardownGateTests {
    /// The gate itself, attacked from the side that matters.
    ///
    /// Revision 2 shipped a teardown that waited for a proportional allocator
    /// crossing, discarded the timeout, and cleared the shared pool either way.
    /// The failure it admitted is exactly this: a run where the release was
    /// never observed, taking the same transition as one where it was. Every
    /// case below drives `releaseObserved: false` and demands something other
    /// than ``WorkerTeardownOutcome/rebuilt``.
    @Test(
        "an unobserved release never reaches a completed rebuild",
        arguments: 1 ... GenerationBatchRecovery.workerTeardownAttempts)
    func unobservedReleaseNeverRebuilds(attempt: Int) {
        let outcome = GenerationBatchRecovery.teardownOutcome(
            releaseObserved: false,
            attempt: attempt,
            maxAttempts: GenerationBatchRecovery.workerTeardownAttempts)
        #expect(outcome != .rebuilt)
    }

    @Test("attempts before the bound ask to be retried")
    func retriesWhileAttemptsRemain() {
        #expect(
            GenerationBatchRecovery.teardownOutcome(
                releaseObserved: false, attempt: 1, maxAttempts: 3) == .retry)
        #expect(
            GenerationBatchRecovery.teardownOutcome(
                releaseObserved: false, attempt: 2, maxAttempts: 3) == .retry)
    }

    @Test("the last unobserved attempt abandons rather than clearing")
    func abandonsAtTheBound() {
        #expect(
            GenerationBatchRecovery.teardownOutcome(
                releaseObserved: false, attempt: 3, maxAttempts: 3) == .abandoned)
        // A caller that overshoots the bound must not fall back through
        // `.retry` into an unbounded wait.
        #expect(
            GenerationBatchRecovery.teardownOutcome(
                releaseObserved: false, attempt: 9, maxAttempts: 3) == .abandoned)
    }

    @Test("an observed release rebuilds, on any attempt including the last")
    func observedReleaseRebuilds() {
        for attempt in 1 ... 3 {
            #expect(
                GenerationBatchRecovery.teardownOutcome(
                    releaseObserved: true, attempt: attempt, maxAttempts: 3) == .rebuilt)
        }
    }

    @Test("the bound is more than one attempt")
    func boundLeavesRoomToRetry() {
        // A bound of 1 collapses `.retry` out of existence: the first
        // unobserved look would abandon, and a teardown that is simply slower
        // than one poll window would be reported as a runtime that never
        // released its weights.
        #expect(GenerationBatchRecovery.workerTeardownAttempts > 1)
    }
}

@Suite("abandoned shared cache rebuild accounting")
struct AbandonedRebuildLedgerTests {
    static let incident = "RuntimeError: [metal::malloc] Resource limit (499000) exceeded"

    @Test("a condemning failure leaves the rebuild pending, not done")
    func condemnationDefersTheRebuild() {
        var ledger = GenerationBatchLedger()
        let slot = ledger.begin()
        let plan = ledger.fail(slot, observing: Self.incident)
        #expect(plan?.sharedCacheRebuild == .afterWorkerTeardown)
        // `fail` does not defer on its own -- the store does, at the same place
        // it raises its own flag. What matters here is that nothing has been
        // attested yet.
        #expect(ledger.sharedCacheRebuilds == 0)
        ledger.deferSharedCacheRebuild()
        #expect(ledger.sharedCacheRebuildPending)
        #expect(ledger.sharedCacheRebuilds == 0)
    }

    @Test("abandoning attests nothing and leaves the pool owed")
    func abandoningDoesNotAttest() {
        var ledger = GenerationBatchLedger()
        ledger.deferSharedCacheRebuild()
        ledger.abandonSharedCacheRebuild()
        // The whole point. An abandoned rebuild must be distinguishable from a
        // completed one by every field an operator can read.
        #expect(ledger.sharedCacheRebuilds == 0)
        #expect(ledger.sharedCacheRebuildsAbandoned == 1)
        #expect(ledger.sharedCacheRebuildPending)
    }

    @Test("only performing the rebuild clears the pending flag")
    func rebuildClearsPending() {
        var ledger = GenerationBatchLedger()
        ledger.deferSharedCacheRebuild()
        ledger.recordSharedCacheRebuild()
        #expect(ledger.sharedCacheRebuilds == 1)
        #expect(ledger.sharedCacheRebuildsAbandoned == 0)
        #expect(!ledger.sharedCacheRebuildPending)
    }

    @Test("an ordinary failure owes the pool nothing")
    func requestScopedFailureIsNotPending() {
        var ledger = GenerationBatchLedger()
        let slot = ledger.begin()
        _ = ledger.fail(slot, observing: "RuntimeError: mid-batch generation step failed")
        #expect(!ledger.sharedCacheRebuildPending)
        #expect(ledger.sharedCacheRebuildsAbandoned == 0)
    }

    @Test("the state endpoint publishes an abandoned rebuild")
    func reportPublishesAbandonment() throws {
        var ledger = GenerationBatchLedger()
        let slot = ledger.begin()
        _ = ledger.fail(slot, observing: Self.incident)
        ledger.deferSharedCacheRebuild()
        ledger.abandonSharedCacheRebuild()
        let report = GenerationBatchReport.make(
            readiness: .generationWorkerFailed(Self.incident), ledger: ledger, memory: nil)
        let batch = try #require(report.body.objectValue?["batch"]?.objectValue)
        #expect(batch["shared_cache_rebuilds"] == .int(0))
        #expect(batch["shared_cache_rebuilds_abandoned"] == .int(1))
        #expect(batch["shared_cache_rebuild_pending"] == .bool(true))
    }

    @Test("a completed rebuild reports as completed and not owed")
    func reportPublishesCompletion() throws {
        var ledger = GenerationBatchLedger()
        let slot = ledger.begin()
        _ = ledger.fail(slot, observing: Self.incident)
        ledger.deferSharedCacheRebuild()
        ledger.recordSharedCacheRebuild()
        let report = GenerationBatchReport.make(
            readiness: .generationWorkerFailed(Self.incident), ledger: ledger, memory: nil)
        let batch = try #require(report.body.objectValue?["batch"]?.objectValue)
        #expect(batch["shared_cache_rebuilds"] == .int(1))
        #expect(batch["shared_cache_rebuilds_abandoned"] == .int(0))
        #expect(batch["shared_cache_rebuild_pending"] == .bool(false))
    }
}

@Suite("condemned worker weight-release gate")
struct WeightReleaseGateTests {
    /// The 0.5B 4-bit acceptance model, from the smoke's own logs. Real
    /// numbers, so a reading that review actually produced can be replayed as a
    /// case rather than paraphrased.
    static let footprint = 262_361_760

    /// What MLX still called active after a *correct* teardown in review.
    ///
    /// Small per-process state — sampler and RNG arrays — that outlived the
    /// request and is almost certainly not this model's weights. "Almost
    /// certainly" is the whole problem: MLX's counters are process-global, so
    /// nothing in the runtime can tell 2,720 B of sampler state from 2,720 B
    /// of retained weights. The gate therefore refuses this reading too, which
    /// is why the clean teardown now abandons like every other one.
    static let residue = 2_720

    /// Review's revision-5 bypass, in bytes: a strict subset of this model's
    /// copied parameter arrays, significant and yet below the load footprint,
    /// with every `Module` already dead.
    static let subsetResidue = 255_724_192

    /// The teardown baseline that goes with those two: the model plus the
    /// residue, which is what `deinit` reads while the weights are still held.
    static let baseline = WeightReleaseGateTests.footprint + WeightReleaseGateTests.residue

    /// A populated registry. The exact size is the model tree's business; what
    /// matters to the gate is that it is greater than zero, so "no live owners"
    /// is an observation rather than an empty container's default answer.
    static let ownerCount = 128

    static func observation(
        containerDeallocated: Bool = true,
        owners: Int = WeightReleaseGateTests.ownerCount,
        live: Int = 0,
        footprint: Int = WeightReleaseGateTests.footprint,
        baseline: Int = WeightReleaseGateTests.baseline,
        active: Int,
        inFlight: Int = 0,
        stable: Int = GenerationBatchRecovery.minimumStableActiveSamples
    ) -> GenerationBatchRecovery.WeightReleaseObservation {
        GenerationBatchRecovery.WeightReleaseObservation(
            containerDeallocated: containerDeallocated,
            weightOwnerCount: owners,
            liveWeightOwners: live,
            weightFootprintBytes: footprint,
            baselineActiveBytes: baseline,
            activeBytes: active,
            generationsInFlight: inFlight,
            stableActiveSamples: stable)
    }

    @Test("a released model: the owners are gone and the allocator is holding nothing")
    func releasedModel() {
        // The only reading that attests. Not "small residue" -- NO residue:
        // `activeMemory == 0` means no MLX buffer of any kind is alive, so no
        // weight buffer is alive either, and that is the one claim a
        // process-global counter can carry without attributing a byte.
        let observed = Self.observation(baseline: Self.footprint, active: 0)
        #expect(observed.returnedBytes == Self.footprint)
        #expect(observed.weightOwnersDeallocated)
        #expect(GenerationBatchRecovery.weightsReleased(observed))
    }

    @Test("the maximum admissible residue is zero bytes")
    func allowanceAdmitsNothing() {
        // The number this whole gate turns on, pinned as a value rather than
        // left implicit in an inequality. Raising it re-opens a band, and a
        // band is what review walked a production input into five rounds
        // running -- so raising it has to redden this.
        #expect(GenerationBatchRecovery.residualNonWeightAllowanceBytes == 0)
        // ...and nothing fits underneath it. One byte of unattributable
        // residue is refused with every other clause green, so there is no
        // interval in which a weight buffer of any size can hide.
        #expect(
            GenerationBatchRecovery.weightsReleased(
                Self.observation(baseline: Self.footprint, active: 0)))
        #expect(
            !GenerationBatchRecovery.weightsReleased(
                Self.observation(baseline: Self.baseline, active: 1)))
    }

    @Test("MEASURED: the runtime's own clean teardown residue is refused")
    func cleanPathResidueRefused() {
        // The cost of the allowance above, asserted rather than described. This
        // is the real reading from a correct condemned teardown of the
        // acceptance model -- container gone, every owner gone, nothing in
        // flight, the allocator at rest, the whole footprint handed back -- and
        // it is refused, because 2,720 B of process-global residue is not
        // attributable to anything. A clean-path rebuild is therefore
        // essentially never attested in this runtime, and the runtime abandons
        // and demands replacement instead of claiming a host it cannot prove is
        // free.
        let observed = Self.observation(active: Self.residue)
        #expect(observed.containerDeallocated)
        #expect(observed.weightOwnersDeallocated)
        #expect(observed.returnedBytes == Self.footprint)
        #expect(!GenerationBatchRecovery.weightsReleased(observed))
    }

    @Test("REVIEW REPRODUCTION F1e-R5: a strict subset of copied weight arrays")
    func arraySubsetDoesNotAttest() {
        // Review's revision-5 bypass, verbatim from the verdict's table. A
        // strict subset of the model's parameter arrays was copied out, so all
        // 316 registered `Module` owners died and ownership reported the model
        // released, while 255,724,192 B of a 262,361,760 B model stayed
        // resident -- below the footprint, which is what a released model looks
        // like to a footprint-relative check, and with returned_bytes clearing
        // the footprint as well. Revision 5 called that a completed release.
        let observed = Self.observation(
            owners: 316, live: 0, baseline: 871_607_252, active: Self.subsetResidue)
        #expect(observed.containerDeallocated)
        #expect(observed.weightOwnersDeallocated)
        #expect(observed.activeBytes < observed.weightFootprintBytes)
        #expect(observed.returnedBytes == 615_883_060)
        #expect(observed.returnedBytes >= observed.weightFootprintBytes)
        #expect(!GenerationBatchRecovery.weightsReleased(observed))
        // ...and it does not become a rebuild through the transition either.
        #expect(
            GenerationBatchRecovery.teardownOutcome(
                releaseObserved: GenerationBatchRecovery.weightsReleased(observed),
                attempt: GenerationBatchRecovery.workerTeardownAttempts,
                maxAttempts: GenerationBatchRecovery.workerTeardownAttempts) == .abandoned)
    }

    @Test("REVIEW REPRODUCTION F1d-R4: a request larger than the model paid the delta")
    func processGlobalDeltaDoesNotAttest() {
        // Review's revision-4 bypass, verbatim, from the two consecutive runs
        // in the verdict: a 6,000-word prompt made the failed request's own KV
        // state larger than the model, so its release alone satisfied the
        // process-global subtraction -- 608,909,592 B returned against a
        // 262,361,760 B footprint -- while post-teardown active_bytes sat at
        // exactly the footprint. Every weight was still resident and revision 4
        // called it a completed release.
        //
        // Ownership is set to "all gone" on purpose: this pins the ABSOLUTE
        // RESIDUE clause on its own, so the case still fails if a later change
        // ever loosens it while ownership happens to look clean.
        let observed = Self.observation(
            live: 0, baseline: 871_271_352, active: Self.footprint)
        #expect(observed.containerDeallocated)
        #expect(observed.weightOwnersDeallocated)
        #expect(observed.returnedBytes == 608_909_592)
        #expect(observed.returnedBytes > observed.weightFootprintBytes)
        #expect(!GenerationBatchRecovery.weightsReleased(observed))
    }

    @Test("NARROWING: the residue bound is the allowance, and the allowance is zero")
    func absoluteResidueBoundary() {
        // The bound proved by tightening rather than by deletion, on both sides
        // of the one number it turns on: exactly the allowance is accepted, one
        // byte more is refused. Nothing else about the reading changes between
        // the two, so the refusal is attributable to the residue clause and to
        // nothing else.
        #expect(
            GenerationBatchRecovery.weightsReleased(
                Self.observation(
                    baseline: 871_271_352,
                    active: GenerationBatchRecovery.residualNonWeightAllowanceBytes)))
        #expect(
            !GenerationBatchRecovery.weightsReleased(
                Self.observation(
                    baseline: 871_271_352,
                    active: GenerationBatchRecovery.residualNonWeightAllowanceBytes + 1)))
        // The two readings revisions 4 and 5 accepted, kept as explicit
        // negatives: at the footprint, and one byte below it. Revision 5's gate
        // read `activeBytes < weightFootprintBytes`, so the second of these was
        // an ACCEPT -- and a strict subset of this model's copied weight arrays
        // lives in exactly that interval.
        #expect(
            !GenerationBatchRecovery.weightsReleased(
                Self.observation(baseline: 871_271_352, active: Self.footprint)))
        #expect(
            !GenerationBatchRecovery.weightsReleased(
                Self.observation(baseline: 871_271_352, active: Self.footprint - 1)))
    }

    @Test("NARROWING: live weight owners refuse a reading every byte clause accepts")
    func liveOwnersRefuseByteGreenReading() {
        // The class no byte comparison can reach, and the reason the registry
        // exists. Half the model tree is retained: MLX reports roughly half the
        // footprint active -- BELOW it, which is what a released model looks
        // like to a process-global counter -- and a large request behind it
        // makes returned_bytes clear the footprint too. Ownership is the only
        // clause left, and it must be the one that refuses.
        let half = Self.footprint / 2
        let byteGreen = Self.observation(live: 1, baseline: 871_271_352, active: half)
        #expect(byteGreen.containerDeallocated)
        #expect(byteGreen.returnedBytes >= byteGreen.weightFootprintBytes)
        #expect(!GenerationBatchRecovery.weightsReleased(byteGreen))
        // Ownership is a VETO and never a proof: the identical reading with
        // every owner gone is refused too, because half a model's worth of
        // unattributable residue is still resident. Revision 5 accepted this
        // second reading, and that is the defect -- copied `MLXArray` values
        // outlive their modules, which this candidate's own array seams
        // demonstrate, so "no live `Module`" cannot mean "no weights".
        #expect(
            !GenerationBatchRecovery.weightsReleased(
                Self.observation(live: 0, baseline: 871_271_352, active: half)))
        // The refusal above is still attributable to ownership, proved by
        // moving ONLY the owner count on a reading the residue clause accepts.
        #expect(
            !GenerationBatchRecovery.weightsReleased(
                Self.observation(live: 1, baseline: 871_271_352, active: 0)))
        #expect(
            GenerationBatchRecovery.weightsReleased(
                Self.observation(live: 0, baseline: 871_271_352, active: 0)))
    }

    @Test("an unpopulated owner registry attests nothing")
    func emptyRegistryFailsClosed() {
        // Absence versus failure to read. A registry that was never populated
        // reports zero live owners, and would go on reporting it for the life
        // of the process. That is not an observation that the weights are gone.
        let unread = Self.observation(owners: 0, live: 0, active: 0)
        #expect(!unread.weightOwnersDeallocated)
        #expect(!GenerationBatchRecovery.weightsReleased(unread))
        // ...and the same reading from a registry that WAS populated is
        // accepted, so the refusal is attributable to the failure to read.
        #expect(GenerationBatchRecovery.weightsReleased(Self.observation(active: 0)))
    }

    @Test("REVIEW REPRODUCTION F1c-R3: the container is gone and the weights are still active")
    func containerGoneWeightsHeld() {
        // The revision-3 narrowed production mutant: delaying only the pinned
        // SerialAccessContainer<ModelContext> destruction let the wrapper reach
        // weak-nil with the whole model still resident. Here the owners are
        // still live too, which is what the registry would report during that
        // destruction.
        let observed = Self.observation(live: 1, active: Self.baseline)
        #expect(observed.containerDeallocated)
        #expect(observed.returnedBytes == 0)
        #expect(!GenerationBatchRecovery.weightsReleased(observed))
    }

    @Test("a wrapper that is still alive is not a release, whatever the bytes say")
    func containerStillAlive() {
        // The byte test must not be able to out-vote the veto: while the
        // container lives the weights are certainly held, and a drop this large
        // can only have come from something else.
        #expect(
            !GenerationBatchRecovery.weightsReleased(
                Self.observation(containerDeallocated: false, active: 0)))
    }

    @Test("NARROWING: returning most of the model is not returning the model")
    func partialReleaseRefused() {
        // The class revision 2's proportional threshold admitted. At the 29 GB
        // target model the half-baseline crossing cleared the pool with about
        // 14.5 GB still resident; scaled to the acceptance model that is this.
        // Here the teardown baseline is the ordinary one, so the shortfall is
        // visible in `returnedBytes` as well as in the absolute residue.
        #expect(
            !GenerationBatchRecovery.weightsReleased(
                Self.observation(active: Self.baseline / 2)))
    }

    @Test("NARROWING: one byte short of the footprint is short")
    func oneByteShortRefused() {
        // The revision-4 clause, kept as a NECESSARY condition: the fall has to
        // have happened inside this teardown window. It is no longer sufficient
        // for anything -- see `processGlobalDeltaDoesNotAttest` -- but a
        // shortfall of a single byte is still a byte of weights that did not
        // come back. Isolated by holding the residue at the allowance and
        // moving only the baseline, so the returned-byte clause is the one
        // deciding.
        #expect(
            !GenerationBatchRecovery.weightsReleased(
                Self.observation(baseline: Self.footprint - 1, active: 0)))
        #expect(
            GenerationBatchRecovery.weightsReleased(
                Self.observation(baseline: Self.footprint, active: 0)))
    }

    @Test("an unmeasured footprint attests nothing", arguments: [0, -1])
    func unmeasuredFootprintFailsClosed(footprint: Int) {
        // Absence versus failure to read. A footprint of zero is not a model
        // that cost nothing; it is a measurement that did not happen, and a
        // comparison against it would be satisfied by every reading including
        // one taken while the whole model is resident.
        #expect(
            !GenerationBatchRecovery.weightsReleased(
                Self.observation(
                    footprint: footprint, baseline: Self.baseline, active: Self.baseline)))
        #expect(
            !GenerationBatchRecovery.weightsReleased(
                Self.observation(footprint: footprint, baseline: Self.baseline, active: 0)))
    }

    @Test("NARROWING: a generation still in flight makes the reading unattributable")
    func concurrentGenerationRefused() {
        // Review named this case beside the one it blocked on: while another
        // request is allocating and freeing, nothing the allocator reports can
        // be assigned to the model. The gate vetoes rather than guessing, and
        // the same reading with nothing in flight is accepted -- so the refusal
        // is attributable to the in-flight clause.
        #expect(
            !GenerationBatchRecovery.weightsReleased(
                Self.observation(active: 0, inFlight: 1)))
        #expect(
            GenerationBatchRecovery.weightsReleased(
                Self.observation(active: 0, inFlight: 0)))
    }

    @Test("NARROWING: a reading that has not come to rest is not a finished release")
    func unstableReadingRefused() {
        // A destruction still running produces a falling count, and any single
        // sample taken during that fall is a partial release read as a finished
        // one. One sample short of the bound is refused; the bound itself is
        // accepted.
        #expect(
            !GenerationBatchRecovery.weightsReleased(
                Self.observation(
                    active: 0,
                    stable: GenerationBatchRecovery.minimumStableActiveSamples - 1)))
        #expect(
            GenerationBatchRecovery.weightsReleased(
                Self.observation(
                    active: 0,
                    stable: GenerationBatchRecovery.minimumStableActiveSamples)))
    }

    @Test("the stability bound asserts something")
    func stabilityBoundIsMoreThanOneSample() {
        // A bound of one would be satisfied by the first reading of any value
        // and would assert nothing at all.
        #expect(GenerationBatchRecovery.minimumStableActiveSamples > 1)
    }

    @Test("active memory rising is not a negative release")
    func growthIsNotRelease() {
        let observed = Self.observation(active: Self.baseline + 1_000_000)
        #expect(observed.returnedBytes == 0)
        #expect(!GenerationBatchRecovery.weightsReleased(observed))
    }

    @Test("a reading the gate refuses cannot reach a completed rebuild")
    func refusedReadingAbandons() {
        // The composition the runtime actually performs: the barrier measures,
        // this gate judges, and `teardownOutcome` transitions. Pinned here so a
        // future caller cannot reintroduce an earlier revision by feeding
        // `teardownOutcome` a boolean of its own making.
        let held = Self.observation(live: 1, active: Self.baseline)
        let observed = GenerationBatchRecovery.weightsReleased(held)
        #expect(
            GenerationBatchRecovery.teardownOutcome(
                releaseObserved: observed, attempt: 1, maxAttempts: 3) == .retry)
        #expect(
            GenerationBatchRecovery.teardownOutcome(
                releaseObserved: observed, attempt: 3, maxAttempts: 3) == .abandoned)
        #expect(
            GenerationBatchRecovery.teardownOutcome(
                releaseObserved: GenerationBatchRecovery.weightsReleased(
                    Self.observation(active: 0)),
                attempt: 3, maxAttempts: 3) == .rebuilt)
    }
}

@Suite("teardown weight-retention seam")
struct TeardownRetentionSeamTests {
    static let base = ["serve", "--model", "/models/qwen", "--port", "18011"]
    static let message = "RuntimeError: [metal::malloc] Resource limit (499000) exceeded"

    static func serve(_ extra: [String]) -> [String] { base + extra }

    @Test("off unless asked for")
    func defaultsOff() throws {
        let (_, options) = try RuntimeOptions.parse(
            arguments: Self.serve(["--fault-inject-generation-error", Self.message]))
        #expect(!options.faultRetainWeightsOnTeardown)
    }

    @Test("parses both values")
    func parsesBothValues() throws {
        for (text, expected) in [("true", true), ("false", false)] {
            let (_, options) = try RuntimeOptions.parse(
                arguments: Self.serve([
                    "--fault-inject-generation-error", Self.message,
                    "--fault-inject-teardown-retain", text,
                ]))
            #expect(options.faultRetainWeightsOnTeardown == expected)
        }
    }

    @Test("refused without an injection to condemn the worker")
    func refusesWithoutInjection() {
        // Same rule as the other modifiers, for the same reason: only a
        // condemned worker has a deferred teardown, so on its own this flag
        // configures a seam that can never fire. An acceptance run that meant
        // to drive the unobserved-release branch and mistyped the message flag
        // would otherwise watch a clean teardown and read it as the negative.
        #expect(
            throws: RuntimeOptionsError.faultInjectionModifierWithoutInjection(
                "--fault-inject-teardown-retain")
        ) {
            try RuntimeOptions.parse(
                arguments: Self.serve(["--fault-inject-teardown-retain", "true"]))
        }
    }

    @Test("a non-boolean value is refused", arguments: ["yes", "1", "TRUE", ""])
    func refusesNonBoolean(value: String) {
        #expect(
            throws: RuntimeOptionsError.invalidBoolean(
                flag: "--fault-inject-teardown-retain", value: value)
        ) {
            try RuntimeOptions.parse(
                arguments: Self.serve([
                    "--fault-inject-generation-error", Self.message,
                    "--fault-inject-teardown-retain", value,
                ]))
        }
    }

    @Test("the below-container seam is off unless asked for")
    func belowContainerDefaultsOff() throws {
        let (_, options) = try RuntimeOptions.parse(
            arguments: Self.serve(["--fault-inject-generation-error", Self.message]))
        #expect(!options.faultRetainWeightsBelowContainerOnTeardown)
    }

    @Test("the below-container seam parses both values")
    func belowContainerParsesBothValues() throws {
        for (text, expected) in [("true", true), ("false", false)] {
            let (_, options) = try RuntimeOptions.parse(
                arguments: Self.serve([
                    "--fault-inject-generation-error", Self.message,
                    "--fault-inject-teardown-retain-weights", text,
                ]))
            #expect(options.faultRetainWeightsBelowContainerOnTeardown == expected)
        }
    }

    @Test("the five retention seams are independent, not modes of one flag")
    func seamsAreIndependent() throws {
        // They park different objects and defeat different clauses of the
        // release gate: the container seam defeats the weak-nil veto, the
        // weights seam defeats a barrier that trusts it, the module seam
        // defeats every byte-derived clause at once, and the array seam defeats
        // ownership by holding nothing that owns anything. Collapsing them
        // would let an acceptance run silently drive one negative while
        // claiming another.
        let (_, container) = try RuntimeOptions.parse(
            arguments: Self.serve([
                "--fault-inject-generation-error", Self.message,
                "--fault-inject-teardown-retain", "true",
            ]))
        #expect(container.faultRetainWeightsOnTeardown)
        #expect(!container.faultRetainWeightsBelowContainerOnTeardown)
        #expect(!container.faultRetainWeightModulesOnTeardown)

        let (_, weights) = try RuntimeOptions.parse(
            arguments: Self.serve([
                "--fault-inject-generation-error", Self.message,
                "--fault-inject-teardown-retain-weights", "true",
            ]))
        #expect(!weights.faultRetainWeightsOnTeardown)
        #expect(weights.faultRetainWeightsBelowContainerOnTeardown)
        #expect(!weights.faultRetainWeightModulesOnTeardown)

        let (_, modules) = try RuntimeOptions.parse(
            arguments: Self.serve([
                "--fault-inject-generation-error", Self.message,
                "--fault-inject-teardown-retain-weight-modules", "true",
            ]))
        #expect(!modules.faultRetainWeightsOnTeardown)
        #expect(!modules.faultRetainWeightsBelowContainerOnTeardown)
        #expect(modules.faultRetainWeightModulesOnTeardown)
        #expect(!modules.faultRetainWeightArraysOnTeardown)

        let (_, arrays) = try RuntimeOptions.parse(
            arguments: Self.serve([
                "--fault-inject-generation-error", Self.message,
                "--fault-inject-teardown-retain-weight-arrays", "true",
            ]))
        #expect(!arrays.faultRetainWeightsOnTeardown)
        #expect(!arrays.faultRetainWeightsBelowContainerOnTeardown)
        #expect(!arrays.faultRetainWeightModulesOnTeardown)
        #expect(arrays.faultRetainWeightArraysOnTeardown)
        #expect(!arrays.faultRetainWeightArraySubsetOnTeardown)

        // The fifth seam is review's revision-5 bypass, and it is deliberately
        // NOT a mode of the fourth: the all-array seam leaves the residue at or
        // above the footprint, this one strictly below it, and those are two
        // different intervals of the residue clause. A shared flag would let an
        // acceptance run believe it had driven the narrowed input while it drove
        // the wide one.
        let subset = try RuntimeOptions.parse(
            arguments: Self.serve([
                "--fault-inject-generation-error", Self.message,
                "--fault-inject-teardown-retain-weight-array-subset", "true",
            ])
        ).1
        #expect(!subset.faultRetainWeightsOnTeardown)
        #expect(!subset.faultRetainWeightsBelowContainerOnTeardown)
        #expect(!subset.faultRetainWeightModulesOnTeardown)
        #expect(!subset.faultRetainWeightArraysOnTeardown)
        #expect(subset.faultRetainWeightArraySubsetOnTeardown)
    }

    @Test("the array-subset seam is off unless asked for")
    func arraySubsetDefaultsOff() throws {
        let (_, options) = try RuntimeOptions.parse(
            arguments: Self.serve(["--fault-inject-generation-error", Self.message]))
        #expect(!options.faultRetainWeightArraySubsetOnTeardown)
    }

    @Test("the array-subset seam parses both values")
    func arraySubsetParsesBothValues() throws {
        for (text, expected) in [("true", true), ("false", false)] {
            let (_, options) = try RuntimeOptions.parse(
                arguments: Self.serve([
                    "--fault-inject-generation-error", Self.message,
                    "--fault-inject-teardown-retain-weight-array-subset", text,
                ]))
            #expect(options.faultRetainWeightArraySubsetOnTeardown == expected)
        }
    }

    @Test("the array-subset seam is refused without an injection")
    func arraySubsetRefusesWithoutInjection() {
        #expect(
            throws: RuntimeOptionsError.faultInjectionModifierWithoutInjection(
                "--fault-inject-teardown-retain-weight-array-subset")
        ) {
            try RuntimeOptions.parse(
                arguments: Self.serve([
                    "--fault-inject-teardown-retain-weight-array-subset", "true",
                ]))
        }
    }

    @Test(
        "the array-subset seam refuses a non-boolean value",
        arguments: ["yes", "1", "TRUE", ""])
    func arraySubsetRefusesNonBoolean(value: String) {
        #expect(
            throws: RuntimeOptionsError.invalidBoolean(
                flag: "--fault-inject-teardown-retain-weight-array-subset", value: value)
        ) {
            try RuntimeOptions.parse(
                arguments: Self.serve([
                    "--fault-inject-generation-error", Self.message,
                    "--fault-inject-teardown-retain-weight-array-subset", value,
                ]))
        }
    }

    @Test("the weight-array seam is off unless asked for")
    func weightArraysDefaultOff() throws {
        let (_, options) = try RuntimeOptions.parse(
            arguments: Self.serve(["--fault-inject-generation-error", Self.message]))
        #expect(!options.faultRetainWeightArraysOnTeardown)
    }

    @Test("the weight-array seam parses both values")
    func weightArraysParseBothValues() throws {
        for (text, expected) in [("true", true), ("false", false)] {
            let (_, options) = try RuntimeOptions.parse(
                arguments: Self.serve([
                    "--fault-inject-generation-error", Self.message,
                    "--fault-inject-teardown-retain-weight-arrays", text,
                ]))
            #expect(options.faultRetainWeightArraysOnTeardown == expected)
        }
    }

    @Test("the weight-array seam is refused without an injection")
    func weightArraysRefuseWithoutInjection() {
        #expect(
            throws: RuntimeOptionsError.faultInjectionModifierWithoutInjection(
                "--fault-inject-teardown-retain-weight-arrays")
        ) {
            try RuntimeOptions.parse(
                arguments: Self.serve(["--fault-inject-teardown-retain-weight-arrays", "true"]))
        }
    }

    @Test(
        "the weight-array seam refuses a non-boolean value",
        arguments: ["yes", "1", "TRUE", ""])
    func weightArraysRefuseNonBoolean(value: String) {
        #expect(
            throws: RuntimeOptionsError.invalidBoolean(
                flag: "--fault-inject-teardown-retain-weight-arrays", value: value)
        ) {
            try RuntimeOptions.parse(
                arguments: Self.serve([
                    "--fault-inject-generation-error", Self.message,
                    "--fault-inject-teardown-retain-weight-arrays", value,
                ]))
        }
    }

    @Test("the module-subset seam is off unless asked for")
    func moduleSubsetDefaultsOff() throws {
        let (_, options) = try RuntimeOptions.parse(
            arguments: Self.serve(["--fault-inject-generation-error", Self.message]))
        #expect(!options.faultRetainWeightModulesOnTeardown)
    }

    @Test("the module-subset seam parses both values")
    func moduleSubsetParsesBothValues() throws {
        for (text, expected) in [("true", true), ("false", false)] {
            let (_, options) = try RuntimeOptions.parse(
                arguments: Self.serve([
                    "--fault-inject-generation-error", Self.message,
                    "--fault-inject-teardown-retain-weight-modules", text,
                ]))
            #expect(options.faultRetainWeightModulesOnTeardown == expected)
        }
    }

    @Test("the module-subset seam is refused without an injection")
    func moduleSubsetRefusesWithoutInjection() {
        #expect(
            throws: RuntimeOptionsError.faultInjectionModifierWithoutInjection(
                "--fault-inject-teardown-retain-weight-modules")
        ) {
            try RuntimeOptions.parse(
                arguments: Self.serve(["--fault-inject-teardown-retain-weight-modules", "true"]))
        }
    }

    @Test(
        "the module-subset seam refuses a non-boolean value",
        arguments: ["yes", "1", "TRUE", ""])
    func moduleSubsetRefusesNonBoolean(value: String) {
        #expect(
            throws: RuntimeOptionsError.invalidBoolean(
                flag: "--fault-inject-teardown-retain-weight-modules", value: value)
        ) {
            try RuntimeOptions.parse(
                arguments: Self.serve([
                    "--fault-inject-generation-error", Self.message,
                    "--fault-inject-teardown-retain-weight-modules", value,
                ]))
        }
    }

    @Test("the below-container seam is refused without an injection")
    func belowContainerRefusesWithoutInjection() {
        #expect(
            throws: RuntimeOptionsError.faultInjectionModifierWithoutInjection(
                "--fault-inject-teardown-retain-weights")
        ) {
            try RuntimeOptions.parse(
                arguments: Self.serve(["--fault-inject-teardown-retain-weights", "true"]))
        }
    }

    @Test(
        "the below-container seam refuses a non-boolean value",
        arguments: ["yes", "1", "TRUE", ""])
    func belowContainerRefusesNonBoolean(value: String) {
        #expect(
            throws: RuntimeOptionsError.invalidBoolean(
                flag: "--fault-inject-teardown-retain-weights", value: value)
        ) {
            try RuntimeOptions.parse(
                arguments: Self.serve([
                    "--fault-inject-generation-error", Self.message,
                    "--fault-inject-teardown-retain-weights", value,
                ]))
        }
    }
}
