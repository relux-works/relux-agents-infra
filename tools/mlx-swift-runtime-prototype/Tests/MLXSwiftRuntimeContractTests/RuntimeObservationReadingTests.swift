import Foundation
import Testing

@testable import MLXSwiftRuntimeContract

/// F2 and F3: an absence and a failure to read are different facts.
///
/// Both findings are one defect wearing two costumes. Something the gate could
/// not establish was spent as if it had been established, because a malformed
/// answer and a failed request were both folded into the *absence* case — and
/// the absence case is the one that reaches the permissive fallback.
///
/// * **F2.** A present but malformed or non-positive `meta.n_ctx` became
///   ``RuntimeContextWindow/notReported``, which
///   ``RuntimeBenchmark/contextPolicy(derivedFrom:observing:)`` spends as
///   permission to derive `unbounded`. Review drove a runtime with a finite
///   32768-token window that answered the *string* `"32768"`; its record
///   asserted `kv=unbounded` and the shipped entry accepted it against an
///   unbounded baseline with exit 0.
/// * **F3.** Every non-200 from `/slots` became
///   ``RuntimeSpeculation/notReported``, and the argv fallback then derived
///   `off`. Review drove a speculative fixture whose `/slots` answered HTTP
///   500; it was scored as MTP-off and the comparison was accepted.
///
/// The readings are exercised through `JSONSerialization` on real bytes rather
/// than through hand-built dictionaries, because the bridging is the bug: a
/// JSON string, a JSON boolean and a JSON float all reach `as? Int` in shapes
/// that either fail silently or succeed wrongly.
///
/// **Production call sites.** `BenchmarkRunCommand.servingAnswer` calls
/// ``RuntimeContextWindow/read(fromModelsEntry:)`` on the `/v1/models` entry it
/// matched by model ID, and `BenchmarkRunCommand.speculationAnswer` calls
/// ``RuntimeSpeculation/read(slotsStatus:body:)`` on the `GET /slots` answer,
/// both inside the gate's own observation window against the process it
/// spawned. `scripts/benchmark-gate-smoke.sh` section 3 drives a malformed
/// `n_ctx` and section 4 an HTTP 500 `/slots` through the shipped subcommand.
@Suite("runtime observation readings")
struct RuntimeObservationReadingTests {
    // ------------------------------------------------------ F2: the KV bound

    /// One `/v1/models` entry, parsed the way the driver parses it.
    static func entry(_ json: String) throws -> [String: Any] {
        let object = try JSONSerialization.jsonObject(with: Data(json.utf8))
        return try #require(object as? [String: Any])
    }

    static func window(_ json: String) throws -> RuntimeContextWindow {
        RuntimeContextWindow.read(fromModelsEntry: try entry(json))
    }

    /// The two MLX runtimes' measured answer: an entry with no `meta` block at
    /// all. This is the one reading that may reach the argv fallback, and a
    /// narrowing fix that refused it would refuse `mlx_lm.server`, the
    /// incumbent baseline.
    @Test("a runtime that answers and names no bound is an absence")
    func absentMetaIsNotReported() throws {
        #expect(try Self.window(#"{"id": "m", "object": "model"}"#) == .notReported)
        #expect(try Self.window(#"{"id": "m", "meta": {"n_ctx_train": 32768}}"#) == .notReported)
    }

    /// `llama-server`'s measured answer.
    @Test("a runtime that names its bound pins that number")
    func reportedBound() throws {
        #expect(try Self.window(#"{"id": "m", "meta": {"n_ctx": 8192}}"#) == .reported(8192))
        #expect(try Self.window(#"{"id": "m", "meta": {"n_ctx": 32768}}"#) == .reported(32768))
    }

    /// The exact shape review used, and the ones beside it.
    ///
    /// Every one of these is a field that is *there* and that the gate could
    /// not get a bound out of. None of them is a runtime declining to answer,
    /// and none of them may reach the argv fallback.
    @Test(
        "a present but unusable n_ctx is a failed read, never an absence",
        arguments: [
            #"{"id": "m", "meta": {"n_ctx": "32768"}}"#,
            #"{"id": "m", "meta": {"n_ctx": true}}"#,
            #"{"id": "m", "meta": {"n_ctx": false}}"#,
            #"{"id": "m", "meta": {"n_ctx": 8192.5}}"#,
            #"{"id": "m", "meta": {"n_ctx": 0}}"#,
            #"{"id": "m", "meta": {"n_ctx": -1}}"#,
            #"{"id": "m", "meta": {"n_ctx": null}}"#,
            #"{"id": "m", "meta": {"n_ctx": [32768]}}"#,
            #"{"id": "m", "meta": {"n_ctx": {"value": 32768}}}"#,
            #"{"id": "m", "meta": "32768"}"#,
            #"{"id": "m", "meta": [{"n_ctx": 32768}]}"#,
        ])
    func malformedContextLengthIsUnread(json: String) throws {
        let reading = try Self.window(json)
        #expect(reading == .unread)
        #expect(reading != .notReported)
    }

    /// The consequence, spelled out: this is what F2 actually cost.
    ///
    /// A malformed answer must not be able to derive the pin that a genuinely
    /// unbounded runtime derives, because ``RuntimeBenchmark/Pins/firstMismatch(against:)``
    /// compares that pin for equality and a false equality is worse than a
    /// refusal.
    @Test("a malformed bound cannot derive the unbounded pin an MLX baseline derives")
    func malformedBoundCannotMatchAnUnboundedBaseline() throws {
        let argv = ["--prefill-step-size", "2048", "--reasoning-effort", "medium"]
        let unbounded = RuntimeBenchmark.contextPolicy(derivedFrom: argv, observing: .notReported)
        let malformed = RuntimeBenchmark.contextPolicy(
            derivedFrom: argv,
            observing: try Self.window(#"{"id": "m", "meta": {"n_ctx": "32768"}}"#))
        #expect(unbounded.contains("kv=unbounded"))
        #expect(malformed.contains("kv=unread"))
        #expect(malformed != unbounded)
        // And `kv=unread` is unpinnable, so it cannot be scored at all.
        #expect(RuntimeBenchmark.unpinnableConditions.contains("kv=unread"))
    }

    // -------------------------------------------------- F3: speculative decoding

    static func speculation(_ status: Int, _ body: String = "") -> RuntimeSpeculation {
        RuntimeSpeculation.read(slotsStatus: status, body: Data(body.utf8))
    }

    /// The only two statuses that say **this route is not here**.
    ///
    /// `mlx_lm.server` and this prototype's router answer 404 for an unknown
    /// path; `llama-server --no-slots` answers 501. Both are runtimes answering
    /// and naming no speculation state, and only they may reach the argv
    /// reading. A narrowing fix that refused them would refuse both MLX
    /// runtimes.
    @Test("route absence is an absence", arguments: [404, 501])
    func routeAbsenceIsNotReported(status: Int) {
        #expect(Self.speculation(status, #"{"error": "not found"}"#) == .notReported)
    }

    /// The exact status review used, and every other failed observation.
    ///
    /// A 500 does not say the runtime is not speculating. It says the gate
    /// could not find out, and the gate has to *establish* that MTP is off
    /// rather than fail to establish that it is on.
    @Test(
        "a failed observation is never a negative observation",
        arguments: [0, 400, 401, 403, 429, 500, 502, 503, 301, 302])
    func failedObservationIsUnread(status: Int) {
        let reading = Self.speculation(status, #"{"error": "boom"}"#)
        #expect(reading == .unread)
        #expect(reading != .notReported)
    }

    @Test("a 200 the gate cannot parse is a failed read")
    func unparseableSlotsBodyIsUnread() {
        #expect(Self.speculation(200, "not json at all") == .unread)
        #expect(Self.speculation(200, #"{"slots": []}"#) == .unread)
    }

    /// A server that describes its slots without naming the field has answered
    /// the request and not the question.
    @Test("slots that name no speculation state are a failed read, not an absence")
    func slotsWithoutTheFieldAreUnread() {
        let reading = Self.speculation(200, #"[{"id": 0, "state": 1}]"#)
        #expect(reading == .unread)
        #expect(reading != .notReported)
    }

    @Test("a runtime with no slots at all has nothing to be speculating")
    func emptySlotsAreNotReported() {
        #expect(Self.speculation(200, "[]") == .notReported)
    }

    @Test("a slot that names the field settles it")
    func reportedSpeculation() {
        #expect(Self.speculation(200, #"[{"params": {"speculative": false}}]"#) == .reported(false))
        #expect(Self.speculation(200, #"[{"params": {"speculative": true}}]"#) == .reported(true))
        // Any single slot drafting settles it: a rate measured across slots
        // that disagree is not a rate this comparison can score.
        #expect(
            Self.speculation(
                200,
                #"[{"params": {"speculative": false}}, {"params": {"speculative": true}}]"#)
                == .reported(true))
    }

    // MARK: - Where the field is looked for (TASK-260828-2wcrph)

    /// The exact `/slots` body this build serves once traffic has touched a
    /// slot, transcribed from `.temp/TASK-260828-2wcrph/probe-slots/`.
    ///
    /// `speculative` is a top-level slot field on `llama.cpp 0.3.0` build
    /// `b10621-c1d0e7a00`; `params` appears on a slot that has already served a
    /// request and carries sampling settings that never name it.
    static func usedSlots(speculative: Bool) -> String {
        """
        [{"id": 0, "n_ctx": 8192, "speculative": \(speculative), "is_processing": false,
          "id_task": 53, "params": {"seed": 4294967295, "temperature": 1.0, "top_k": 20,
          "top_p": 0.95, "n_predict": 8}}]
        """
    }

    @Test("a params block that does not name the field cannot hide the slot's own reading")
    func paramsDoesNotShadowTheTopLevelField() {
        // The regression TASK-260828-2wcrph measured. Under the previous
        // `(slot["params"] as? [String: Any]) ?? slot`, a slot carrying a
        // `params` object was consulted ONLY there, the field was not in it,
        // and once the soak had touched every slot the whole array read
        // `unread` -- refusing a runtime that had answered the question.
        let reading = Self.speculation(200, Self.usedSlots(speculative: false))
        #expect(reading == .reported(false))
        #expect(reading != .unread)
    }

    @Test("a speculating runtime is still caught when its params block is silent")
    func paramsDoesNotHideASpeculatingRuntime() {
        // The direction that must never regress. `--spec-type ngram-mod` was
        // measured to set the top-level field `true` on used and unused slots
        // alike, so a reader that stopped consulting the slot would report a
        // drafting server as quiet -- and MTP-off is a precondition of every
        // scored comparison in this story.
        #expect(Self.speculation(200, Self.usedSlots(speculative: true)) == .reported(true))
    }

    @Test("either placement reporting true settles it, however the other reads")
    func eitherPlacementCanSettleIt() {
        // Neither reading is allowed to overrule the other towards `false`. A
        // disagreement means at least one slot is drafting, which is the
        // conservative direction and the only safe one.
        #expect(
            Self.speculation(200, #"[{"speculative": false, "params": {"speculative": true}}]"#)
                == .reported(true))
        #expect(
            Self.speculation(200, #"[{"speculative": true, "params": {"speculative": false}}]"#)
                == .reported(true))
        #expect(
            Self.speculation(200, #"[{"speculative": false, "params": {"speculative": false}}]"#)
                == .reported(false))
    }

    @Test("the class was not widened: a slot naming the field nowhere is still a failed read")
    func neitherPlacementIsStillUnread() {
        // The narrowing counterpart. Reading two places must not turn "the
        // server described its slots and never answered the question" into an
        // answer, so a slot whose `params` block is present and silent AND
        // whose top level is silent stays `unread`.
        let reading = Self.speculation(
            200, #"[{"id": 0, "state": 1, "params": {"temperature": 1.0}}]"#)
        #expect(reading == .unread)
        #expect(reading != .reported(false))
        #expect(reading != .notReported)
    }

    @Test("a non-boolean speculative field is a failed read in either placement")
    func nonBooleanSpeculativeIsUnread() {
        // `as? Bool` is the whole guard here, and the F2 lesson applies: a
        // field that is present and unusable is not a field that is absent.
        #expect(Self.speculation(200, #"[{"speculative": "false"}]"#) == .unread)
        #expect(Self.speculation(200, #"[{"params": {"speculative": "true"}}]"#) == .unread)
        #expect(Self.speculation(200, #"[{"speculative": null}]"#) == .unread)
    }

    /// The consequence, spelled out: this is what F3 actually cost.
    ///
    /// The gate admits exactly one speculation reading, and a failed `/slots`
    /// observation must not be able to produce it however quiet the launch was.
    @Test("a 500 from /slots cannot derive the one admitted reading")
    func failedSlotsReadCannotDeriveOff() {
        let quietLaunch = ["--model", "/m", "--reasoning-effort", "medium"]
        let absent = RuntimeBenchmark.speculationPolicy(
            derivedFrom: quietLaunch, observing: Self.speculation(404))
        let failed = RuntimeBenchmark.speculationPolicy(
            derivedFrom: quietLaunch, observing: Self.speculation(500))
        #expect(absent == RuntimeBenchmark.admittedSpeculation)
        #expect(failed == "unread")
        #expect(failed != RuntimeBenchmark.admittedSpeculation)
    }

    // ------------------------------------- the same shape, read off the launch

    /// The audit the rework brief asked for, turned into tests.
    ///
    /// F2 and F3 are readings of the *process*. The same collapse exists one
    /// step earlier, in what the gate reads off the launch: a flag that is
    /// there and whose value cannot be parsed used to fall through to "the
    /// launch asked for nothing", which is the permissive branch in both cases.
    @Test(
        "a context flag the gate cannot read is not a launch that asked for nothing",
        arguments: [
            ["--ctx-size", "abc"], ["--ctx-size=abc"], ["--ctx-size", "0"],
            ["--ctx-size", "-1"], ["-c", "eight-thousand"], ["--max-kv-size", ""],
            ["--reasoning-effort", "medium", "--ctx-size"],
        ])
    func unreadableContextFlagIsNotAnAbsence(argv: [String]) {
        let reading = RuntimeBenchmark.declaredContextBound(inArgv: argv)
        #expect(reading != .none)
        guard case .unreadable = reading else {
            Issue.record("\(argv) read as \(reading) rather than unreadable")
            return
        }
    }

    /// And the readable ones still read, so the clause narrows nothing that was
    /// working.
    @Test("a context flag with a value still pins that value")
    func readableContextFlagStillReads() {
        #expect(
            RuntimeBenchmark.declaredContextBound(inArgv: ["--ctx-size", "8192"])
                == .pinned(flag: "--ctx-size", value: 8192))
        #expect(
            RuntimeBenchmark.declaredContextBound(inArgv: ["--max-kv-size=4096"])
                == .pinned(flag: "--max-kv-size", value: 4096))
        #expect(RuntimeBenchmark.declaredContextBound(inArgv: ["--model", "/m"]) == .none)
    }

    /// An unreadable context flag reaches the refusal through the production
    /// admission path, not just through the reader.
    @Test("an unreadable context flag refuses the record by name")
    func unreadableContextFlagRefusesTheRecord() {
        let argv = RuntimeBenchmarkTests.launchArgv + ["--ctx-size", "abc"]
        let record = RuntimeBenchmarkTests.record(
            runtime: "llamacpp",
            pins: RuntimeBenchmarkTests.variantPins(
                contextPolicy: RuntimeBenchmark.contextPolicy(
                    derivedFrom: argv, observing: .reported(8192))),
            startedAt: 100, finishedAt: 200,
            provenance: RuntimeBenchmarkTests.provenance(launchArgv: argv))
        #expect(
            throws: RuntimeBenchmark.AdmissionError.contextBoundUnreadable(
                runtime: "llamacpp", flag: "--ctx-size", raw: "abc")
        ) {
            try RuntimeBenchmark.admitProvenance(
                record,
                observing: RuntimeBenchmarkTests.attestation(
                    for: record, contextWindow: .reported(8192), speculation: .notReported))
        }
    }

    @Test(
        "a speculative flag the gate cannot read is a declaration, not a silence",
        arguments: [
            ["--spec-type"], ["--spec-type="], ["--spec-type", ""], ["--spec-type", ",,"],
            ["--spec-draft-model"], ["--model-draft"], ["-md"], ["-md="],
        ])
    func unreadableSpeculativeFlagIsADeclaration(argv: [String]) {
        let declared = RuntimeBenchmark.declaredSpeculation(inArgv: argv)
        #expect(declared != nil)
        #expect(declared?.value == RuntimeBenchmark.unreadableSpeculationValue)
        // And it refuses: a `/slots`-less runtime cannot be launched into
        // drafting through a flag the gate failed to parse.
        let policy = RuntimeBenchmark.speculationPolicy(
            derivedFrom: argv, observing: .notReported)
        #expect(policy != RuntimeBenchmark.admittedSpeculation)
    }

    /// The other direction, so the clause is not reject-all: the two readings
    /// that genuinely mean "no drafting was asked for" still mean that.
    @Test("an explicit none and an absent flag are still silence")
    func readableSpeculativeSilenceStillReads() {
        #expect(RuntimeBenchmark.declaredSpeculation(inArgv: ["--spec-type", "none"]) == nil)
        #expect(RuntimeBenchmark.declaredSpeculation(inArgv: ["--spec-type=none"]) == nil)
        #expect(RuntimeBenchmark.declaredSpeculation(inArgv: ["--model", "/m"]) == nil)
        #expect(
            RuntimeBenchmark.speculationPolicy(
                derivedFrom: ["--spec-type", "none"], observing: .notReported)
                == RuntimeBenchmark.admittedSpeculation)
    }
}
