import Foundation
import Testing

@testable import MLXSwiftRuntimeContract

@Suite("runtime benchmark comparison gate")
struct RuntimeBenchmarkTests {
    static let modelPath = "/Users/alexis/src/local-models/Qwen3.8-27B-Uncensored-MLX-8bit"

    /// A launch that requests the same KV bound the fake live observation
    /// reports, plus explicit prefill and reasoning conditions.
    static let launchArgv = [
        "serve", "--model", modelPath, "--host", "127.0.0.1", "--port", "18031",
        "--model-factory", "text-only", "--max-kv-size", "76800",
        "--prefill-step-size", "2048",
        "--reasoning-effort", "medium",
    ]

    static let pythonLaunchArgv = [
        "mlx_lm.server", "--model", modelPath, "--host", "127.0.0.1", "--port", "18031",
        "--max-kv-size", "76800", "--prefill-step-size", "2048",
        "--chat-template-args", #"{"reasoning_effort":"medium"}"#,
    ]

    static let generationConfiguration = RuntimeGenerationConfiguration.reported(
        prefillStepSize: 2_048, reasoningEffort: "medium")

    static func digest(_ seed: String) -> String {
        String(repeating: seed, count: 64 / seed.count)
    }

    static func provenance(
        launchArgv: [String] = RuntimeBenchmarkTests.launchArgv,
        configDigest: String = digest("ab"),
        executableDigest: String = digest("cd"),
        configPath: String = "/tmp/model-harness.benchmark.toml",
        profile: String = "qwen-benchmark",
        harnessCommand: [String]? = nil,
        driverDigest: String = digest("ef")
    ) -> RuntimeBenchmark.LaunchProvenance {
        RuntimeBenchmark.LaunchProvenance(
            driverCommand: ["/usr/bin/python3", "runtime-benchmark.py", "--runtime", "x"],
            driverDigest: driverDigest,
            harnessCommand: harnessCommand
                ?? ["model-harness", "run", profile, "--config", configPath],
            configPath: configPath,
            configDigest: configDigest,
            profile: profile,
            launchExecutable: "/usr/local/bin/runtime",
            launchExecutableDigest: executableDigest,
            launchArgv: launchArgv,
            runtimeProcessID: 4242)
    }

    static let pins = RuntimeBenchmark.Pins(
        hostIdentity: "MacBookPro18,2/68719476736/25F80",
        modelPath: modelPath,
        modelDigest: "9f2c1a",
        quantization: "8bit/group64/affine",
        promptSuiteDigest: "aa11bb",
        contextPolicy: RuntimeBenchmark.contextPolicy(
            observing: .reported(76_800),
            generationConfiguration: generationConfiguration),
        maxOutputTokens: 256,
        temperature: 0.0,
        topP: 1.0,
        seed: 1234)

    /// A transcript that looks like one served completion.
    ///
    /// Every scenario helper below carries one, because a record without one is
    /// now inadmissible by name and the tests that attack *other* clauses
    /// should not all trip that one first. The tests that attack this clause
    /// build their scenarios without it, on purpose.
    static func transcript(
        at instant: Double, path: String = RuntimeBenchmark.ScenarioTranscript.completionPath,
        status: Int = 200, responseBytes: Int = 4096
    ) -> RuntimeBenchmark.ScenarioTranscript {
        RuntimeBenchmark.ScenarioTranscript(exchanges: [
            RuntimeBenchmark.ScenarioTranscript.Exchange(
                method: "POST", path: path, requestDigest: digest("1a"), requestByteCount: 900,
                status: status, responseDigest: digest("2b"), responseByteCount: responseBytes,
                sentAtUnixSeconds: instant, firstByteAtUnixSeconds: instant,
                lastByteAtUnixSeconds: instant)
        ])
    }

    static func scenario(
        _ name: String,
        succeeded: Bool = true,
        failureMode: String? = nil,
        ttft: Double? = 1.0,
        prefill: Double? = 100,
        decode: Double? = 10,
        promptTokens: Int? = 512,
        footprint: Int? = 29_000_000_000,
        transcript: RuntimeBenchmark.ScenarioTranscript? = nil
    ) -> RuntimeBenchmark.ScenarioResult {
        RuntimeBenchmark.ScenarioResult(
            name: name, succeeded: succeeded, failureMode: failureMode,
            promptTokens: promptTokens, completionTokens: 64,
            timeToFirstTokenSeconds: ttft, prefillTokensPerSecond: prefill,
            decodeTokensPerSecond: decode, wallClockSeconds: 7,
            peakPhysicalFootprintBytes: footprint,
            processPeakSoFarBytes: 29_000_000_000,
            transcript: transcript ?? Self.transcript(at: 0))
    }

    /// The same scenario with its transcript moved inside a given pass.
    ///
    /// Applied by ``record(runtime:...)`` so a helper-built scenario always sits
    /// inside the interval of whatever record it ends up in. Without it every
    /// test would have to hand-place its exchanges, and the clause under attack
    /// would be ambiguous with the window clause.
    static func anchored(
        _ scenario: RuntimeBenchmark.ScenarioResult, at instant: Double
    ) -> RuntimeBenchmark.ScenarioResult {
        guard let existing = scenario.transcript, !existing.exchanges.isEmpty else {
            return scenario
        }
        let moved = existing.exchanges.map { exchange in
            RuntimeBenchmark.ScenarioTranscript.Exchange(
                method: exchange.method, path: exchange.path,
                requestDigest: exchange.requestDigest,
                requestByteCount: exchange.requestByteCount, status: exchange.status,
                responseDigest: exchange.responseDigest,
                responseByteCount: exchange.responseByteCount, sentAtUnixSeconds: instant,
                firstByteAtUnixSeconds: exchange.firstByteAtUnixSeconds.map { _ in instant },
                lastByteAtUnixSeconds: instant)
        }
        return RuntimeBenchmark.ScenarioResult(
            name: scenario.name, succeeded: scenario.succeeded,
            failureMode: scenario.failureMode, promptTokens: scenario.promptTokens,
            completionTokens: scenario.completionTokens,
            timeToFirstTokenSeconds: scenario.timeToFirstTokenSeconds,
            prefillTokensPerSecond: scenario.prefillTokensPerSecond,
            decodeTokensPerSecond: scenario.decodeTokensPerSecond,
            wallClockSeconds: scenario.wallClockSeconds,
            peakPhysicalFootprintBytes: scenario.peakPhysicalFootprintBytes,
            processPeakSoFarBytes: scenario.processPeakSoFarBytes,
            transcript: RuntimeBenchmark.ScenarioTranscript(exchanges: moved))
    }

    static func record(
        runtime: String,
        pins: RuntimeBenchmark.Pins = RuntimeBenchmarkTests.pins,
        startedAt: Double,
        finishedAt: Double,
        footprint: Int? = 29_000_000_000,
        scenarios: [RuntimeBenchmark.ScenarioResult] = [scenario("short_prompt")],
        asymmetries: [String] = [],
        revisions: [String: String] = ["runtime": "x"],
        provenance: RuntimeBenchmark.LaunchProvenance? = nil
    ) -> RuntimeBenchmark.RunRecord {
        // Distinct by runtime, because two records naming different runtimes
        // that launched the same executable are refused -- correctly.
        let bound =
            provenance
            ?? RuntimeBenchmarkTests.provenance(
                launchArgv: runtime == "python-mlx-lm" ? pythonLaunchArgv : launchArgv,
                executableDigest: digest(runtime == "mlx-swift" ? "12" : "cd"))
        return RuntimeBenchmark.RunRecord(
            runtime: runtime, revisions: revisions, command: bound.harnessCommand,
            provenance: bound, pins: pins,
            startedAtUnixSeconds: startedAt, finishedAtUnixSeconds: finishedAt,
            peakPhysicalFootprintBytes: footprint,
            scenarios: scenarios.map { anchored($0, at: startedAt + 1) },
            declaredAsymmetries: asymmetries)
    }

    static let baseline = record(runtime: "python-mlx-lm", startedAt: 100, finishedAt: 200)
    static let candidate = record(
        runtime: "mlx-swift", startedAt: 300, finishedAt: 400,
        provenance: provenance(executableDigest: digest("12")))

    /// The digest the tests pretend this binary has.
    ///
    /// Every attestation below carries it and every comparison below is made
    /// with it, so the "the judging binary is not the observing binary" clause
    /// is exercised by the test that deliberately breaks it rather than
    /// accidentally by all of them.
    static let gateDigest = digest("7a")

    /// A gate observation consistent with a record.
    ///
    /// Defaults are derived from the record rather than restated, so a test
    /// that wants to attack one clause changes exactly that argument and every
    /// other clause stays satisfied. A helper that let the two drift silently
    /// would make each refusal below ambiguous.
    static func attestation(
        for record: RuntimeBenchmark.RunRecord,
        openedAt: Double? = nil,
        closedAt: Double?? = nil,
        servedModelID: String?? = nil,
        gateBinaryDigest: String? = nil,
        processID: Int? = nil,
        executableDigest: String? = nil,
        configDigest: String? = nil,
        profile: String? = nil,
        processStart: Double? = nil,
        contextWindow: RuntimeContextWindow? = nil,
        generationConfiguration: RuntimeGenerationConfiguration? = nil,
        transcriptDigest: String?? = nil
    ) -> RuntimeAttestation {
        RuntimeAttestation(
            runtime: record.runtime,
            processID: processID ?? record.provenance.runtimeProcessID,
            processStartUnixSeconds: processStart ?? record.startedAtUnixSeconds,
            observedExecutablePath: record.provenance.launchExecutable,
            observedExecutableDigest: executableDigest
                ?? record.provenance.launchExecutableDigest,
            configPath: record.provenance.configPath,
            configDigest: configDigest ?? record.provenance.configDigest,
            profile: profile ?? record.provenance.profile,
            openedAtUnixSeconds: openedAt ?? record.startedAtUnixSeconds,
            closedAtUnixSeconds: closedAt ?? record.finishedAtUnixSeconds,
            servedModelID: servedModelID ?? record.pins.modelPath,
            observedContextWindow: contextWindow ?? .reported(76_800),
            observedGenerationConfiguration: generationConfiguration
                ?? Self.generationConfiguration,
            gateBinaryDigest: gateBinaryDigest ?? gateDigest,
            // Sealed over the record it is being minted for, which is what the
            // observing invocation does in production. A test that attacks the
            // seal passes its own value here.
            transcriptDigest: transcriptDigest
                ?? RuntimeBenchmark.transcriptDigest(of: record))
    }

    /// The production admission, called with attestations that agree with the
    /// records.
    ///
    /// The tests below are about every *other* clause, so they should not each
    /// have to restate an observation; the attestation clauses have their own
    /// suite. This shim mints one from each record, which is exactly the thing
    /// a real caller cannot do — and the point of the attestation is that the
    /// gate binary, not the caller, writes it.
    static func admit(
        baseline: RuntimeBenchmark.RunRecord,
        candidate: RuntimeBenchmark.RunRecord,
        requiredScenarios: [String]
    ) throws -> RuntimeBenchmark.Comparison {
        try RuntimeBenchmark.admit(
            baseline: baseline, baselineAttestation: attestation(for: baseline),
            candidate: candidate, candidateAttestation: attestation(for: candidate),
            requiredScenarios: requiredScenarios, gateBinaryDigest: gateDigest)
    }

    static let thresholds = RuntimeBenchmark.Thresholds(
        maxTimeToFirstTokenRatio: 1.10,
        minPrefillThroughputRatio: 0.90,
        minDecodeThroughputRatio: 0.90,
        maxPeakFootprintRatio: 1.10,
        maxPromptTokenSkewRatio: 1.10,
        paritySuccessScenarios: ["short_prompt"],
        scoredScenarios: ["short_prompt"])

    // MARK: - Admission, positive

    @Test("two sequential runs with identical pins are admitted")
    func admitsIdenticalPins() throws {
        let comparison = try Self.admit(
            baseline: Self.baseline, candidate: Self.candidate,
            requiredScenarios: ["short_prompt"])
        #expect(comparison.pins == Self.pins)
        #expect(comparison.sharedScenarios == ["short_prompt"])
    }

    @Test("runs that merely touch at an instant do not overlap")
    func admitsAdjacentIntervals() throws {
        // The boundary the overlap refusal is written against. Sequential runs
        // on this host are back to back; refusing a zero-length touch would
        // refuse every honest pass.
        let candidate = Self.record(runtime: "mlx-swift", startedAt: 200, finishedAt: 300)
        _ = try Self.admit(
            baseline: Self.baseline, candidate: candidate, requiredScenarios: [])
    }

    @Test("declared asymmetries from both records are carried through, deduplicated")
    func carriesDeclaredAsymmetries() throws {
        let baseline = Self.record(
            runtime: "python-mlx-lm", startedAt: 100, finishedAt: 200,
            asymmetries: ["prompt cache enabled", "lazy load"])
        let candidate = Self.record(
            runtime: "mlx-swift", startedAt: 300, finishedAt: 400,
            asymmetries: ["prompt cache enabled", "single generation at a time"])
        let comparison = try Self.admit(
            baseline: baseline, candidate: candidate, requiredScenarios: [])
        #expect(
            comparison.declaredAsymmetries == [
                "prompt cache enabled", "lazy load", "single generation at a time",
            ])
    }

    // MARK: - Admission, negative

    /// Every pinned field, mutated one at a time. The gate must refuse each of
    /// them *by name*.
    ///
    /// Written as a table rather than as one representative case because the
    /// cheap way to break this gate is not to delete it — it is to drop one
    /// field from the list it walks, which a single-field test would never see.
    /// Every pinned field, mutated one at a time. The gate must refuse each of
    /// them *by name*.
    ///
    /// Written as a table rather than as one representative case because the
    /// cheap way to break this gate is not to delete it -- it is to drop one
    /// field from the list it walks, which a single-field test would never see.
    @Test(
        "a difference in any pinned condition refuses the comparison",
        arguments: mismatchedPins)
    func refusesAnyPinMismatch(field: String, pins: RuntimeBenchmark.Pins) {
        let candidate = Self.record(
            runtime: "mlx-swift", pins: pins, startedAt: 300, finishedAt: 400)
        do {
            _ = try Self.admit(
                baseline: Self.baseline, candidate: candidate, requiredScenarios: [])
            Issue.record("pin \(field) mismatch was admitted")
        } catch let error as RuntimeBenchmark.AdmissionError {
            guard case .pinMismatch(let named, _, _) = error else {
                Issue.record("expected a pin mismatch for \(field), got \(error)")
                return
            }
            #expect(named == field)
        } catch {
            Issue.record("unexpected error \(error)")
        }
    }

    @Test("a runtime compared against itself is refused")
    func refusesSameRuntime() {
        let candidate = Self.record(runtime: "python-mlx-lm", startedAt: 300, finishedAt: 400)
        #expect(throws: RuntimeBenchmark.AdmissionError.sameRuntimeIdentity("python-mlx-lm")) {
            _ = try Self.admit(
                baseline: Self.baseline, candidate: candidate, requiredScenarios: [])
        }
    }

    @Test("overlapping runs are refused because they measured each other")
    func refusesOverlappingRuns() {
        // The host holds ~35 GiB free and one copy of a 28 GB model. Two runs
        // that overlap by even a second were paging against each other, and
        // whichever one lost is not the runtime that is slower.
        let candidate = Self.record(runtime: "mlx-swift", startedAt: 150, finishedAt: 400)
        do {
            _ = try Self.admit(
                baseline: Self.baseline, candidate: candidate, requiredScenarios: [])
            Issue.record("an overlapping pair was admitted")
        } catch let error as RuntimeBenchmark.AdmissionError {
            guard case .overlappingRuns(_, _, let seconds) = error else {
                Issue.record("expected an overlap refusal, got \(error)")
                return
            }
            #expect(seconds == 50)
        } catch {
            Issue.record("unexpected error \(error)")
        }
    }

    @Test("an overlap of one second is still an overlap")
    func refusesMinimalOverlap() {
        // The narrowing case. A gate that only refuses *large* overlaps admits
        // exactly the pair where one runtime's teardown is still returning
        // 28 GB while the other one starts loading.
        let candidate = Self.record(runtime: "mlx-swift", startedAt: 199, finishedAt: 400)
        #expect(throws: RuntimeBenchmark.AdmissionError.self) {
            _ = try Self.admit(
                baseline: Self.baseline, candidate: candidate, requiredScenarios: [])
        }
    }

    @Test("a record that finished before it started cannot prove sequencing")
    func refusesReversedInterval() {
        let candidate = Self.record(runtime: "mlx-swift", startedAt: 400, finishedAt: 300)
        #expect(throws: RuntimeBenchmark.AdmissionError.reversedInterval(runtime: "mlx-swift")) {
            _ = try Self.admit(
                baseline: Self.baseline, candidate: candidate, requiredScenarios: [])
        }
    }

    @Test("a required scenario missing from either record refuses the comparison")
    func refusesMissingRequiredScenario() {
        #expect(
            throws: RuntimeBenchmark.AdmissionError.missingScenario(
                runtime: "python-mlx-lm", scenario: "context_75k")
        ) {
            _ = try Self.admit(
                baseline: Self.baseline, candidate: Self.candidate,
                requiredScenarios: ["context_75k"])
        }
        let candidate = Self.record(
            runtime: "mlx-swift", startedAt: 300, finishedAt: 400,
            scenarios: [Self.scenario("short_prompt")])
        let baseline = Self.record(
            runtime: "python-mlx-lm", startedAt: 100, finishedAt: 200,
            scenarios: [Self.scenario("short_prompt"), Self.scenario("context_75k")])
        #expect(
            throws: RuntimeBenchmark.AdmissionError.missingScenario(
                runtime: "mlx-swift", scenario: "context_75k")
        ) {
            _ = try Self.admit(
                baseline: baseline, candidate: candidate, requiredScenarios: ["context_75k"])
        }
    }

    // MARK: - Reading a record

    @Test("bytes that could not be read are a read failure, never an empty record")
    func distinguishesUnreadableFromEmpty() {
        #expect(
            throws: RuntimeBenchmark.AdmissionError.unreadable(
                path: "/tmp/missing.json", detail: "no bytes were read")
        ) {
            _ = try RuntimeBenchmark.decodeRecord(path: "/tmp/missing.json", data: nil)
        }
    }

    @Test("a record missing a pinned field is malformed, not a record with a default pin")
    func refusesRecordMissingPin() throws {
        // Dropping `seed` is the interesting one: a decoder that defaulted it
        // would make two runs that pinned *no* seed agree with each other, and
        // the sampler difference the pin exists to catch would go unreported.
        let complete = try JSONEncoder().encode(Self.baseline)
        var object = try #require(
            try JSONSerialization.jsonObject(with: complete) as? [String: Any])
        var pins = try #require(object["pins"] as? [String: Any])
        pins.removeValue(forKey: "seed")
        object["pins"] = pins
        let mutilated = try JSONSerialization.data(withJSONObject: object)
        do {
            _ = try RuntimeBenchmark.decodeRecord(path: "/tmp/r.json", data: mutilated)
            Issue.record("a record with no pinned seed decoded")
        } catch let error as RuntimeBenchmark.AdmissionError {
            guard case .malformed = error else {
                Issue.record("expected malformed, got \(error)")
                return
            }
        } catch {
            Issue.record("unexpected error \(error)")
        }
    }

    @Test("truncated bytes are malformed rather than absent")
    func refusesTruncatedRecord() {
        let truncated = Data("{\"runtime\":\"mlx-swift\",\"pins\":".utf8)
        do {
            _ = try RuntimeBenchmark.decodeRecord(path: "/tmp/r.json", data: truncated)
            Issue.record("truncated bytes decoded")
        } catch let error as RuntimeBenchmark.AdmissionError {
            guard case .malformed = error else {
                Issue.record("expected malformed, got \(error)")
                return
            }
        } catch {
            Issue.record("unexpected error \(error)")
        }
    }

    @Test("a record round-trips through JSON unchanged")
    func roundTripsRecord() throws {
        let encoded = try JSONEncoder().encode(Self.baseline)
        let decoded = try RuntimeBenchmark.decodeRecord(path: "/tmp/r.json", data: encoded)
        #expect(decoded == Self.baseline)
    }

    // MARK: - Decision

    @Test("a candidate inside every threshold is accepted")
    func acceptsCandidateWithinThresholds() throws {
        let comparison = try Self.admit(
            baseline: Self.baseline, candidate: Self.candidate,
            requiredScenarios: ["short_prompt"])
        let decision = RuntimeBenchmark.decide(
            comparison: comparison, thresholds: Self.thresholds)
        #expect(decision.accepted)
        #expect(decision.blockers.isEmpty)
    }

    @Test("a ratio exactly on the bound is inside it")
    func acceptsBoundaryRatio() throws {
        // 1.10x TTFT and 0.90x throughput are admissible by definition of the
        // threshold; pinning the boundary keeps a later `<` / `<=` edit from
        // silently moving the accepted band.
        let candidate = Self.record(
            runtime: "mlx-swift", startedAt: 300, finishedAt: 400,
            footprint: Int(29_000_000_000.0 * 1.10),
            scenarios: [Self.scenario("short_prompt", ttft: 1.10, prefill: 90, decode: 9)])
        let comparison = try Self.admit(
            baseline: Self.baseline, candidate: candidate, requiredScenarios: [])
        let decision = RuntimeBenchmark.decide(
            comparison: comparison, thresholds: Self.thresholds)
        #expect(decision.accepted, "blockers: \(decision.blockers)")
    }

    @Test(
        "a candidate outside any single threshold is rejected",
        arguments: [
            ("time_to_first_token_seconds", 1.5, 100.0, 10.0, 29_000_000_000),
            ("prefill_tokens_per_second", 1.0, 50.0, 10.0, 29_000_000_000),
            ("decode_tokens_per_second", 1.0, 100.0, 5.0, 29_000_000_000),
            ("peak_physical_footprint_bytes", 1.0, 100.0, 10.0, 40_000_000_000),
        ] as [(String, Double, Double, Double, Int)])
    func rejectsOutsideThreshold(
        metric: String, ttft: Double, prefill: Double, decode: Double, footprint: Int
    ) throws {
        let candidate = Self.record(
            runtime: "mlx-swift", startedAt: 300, finishedAt: 400, footprint: footprint,
            scenarios: [
                Self.scenario("short_prompt", ttft: ttft, prefill: prefill, decode: decode)
            ])
        let comparison = try Self.admit(
            baseline: Self.baseline, candidate: candidate, requiredScenarios: [])
        let decision = RuntimeBenchmark.decide(
            comparison: comparison, thresholds: Self.thresholds)
        #expect(!decision.accepted)
        #expect(decision.blockers.contains { $0.contains(metric) })
        #expect(decision.deltas.contains { $0.metric == metric && $0.verdict == "outside" })
    }

    @Test("a scenario that claims success with no measurements is a blocker, not a pass")
    func refusesUnmeasuredSuccess() throws {
        // The self-minted-evidence shape. `succeeded: true` is the driver's own
        // word for its own work; the numbers are the evidence. A decision that
        // scored the flag and skipped the missing numbers would accept a
        // migration on a driver that measured nothing at all.
        let candidate = Self.record(
            runtime: "mlx-swift", startedAt: 300, finishedAt: 400,
            scenarios: [
                // Carries a served completion, so the refusal under test is the
                // one about missing *measurements* rather than the one about a
                // scenario that never served anything.
                RuntimeBenchmark.ScenarioResult(
                    name: "short_prompt", succeeded: true,
                    transcript: Self.transcript(at: 0))
            ])
        let comparison = try Self.admit(
            baseline: Self.baseline, candidate: candidate, requiredScenarios: [])
        let decision = RuntimeBenchmark.decide(
            comparison: comparison, thresholds: Self.thresholds)
        #expect(!decision.accepted)
        for metric in [
            "time_to_first_token_seconds", "prefill_tokens_per_second",
            "decode_tokens_per_second",
        ] {
            #expect(
                decision.blockers.contains { $0.contains(metric) && $0.contains("not measured") },
                "\(metric) was not reported unmeasured")
        }
    }

    @Test("an unmeasured process footprint is a blocker, not a pass")
    func refusesUnmeasuredFootprint() throws {
        let candidate = Self.record(
            runtime: "mlx-swift", startedAt: 300, finishedAt: 400, footprint: nil)
        let comparison = try Self.admit(
            baseline: Self.baseline, candidate: candidate, requiredScenarios: [])
        let decision = RuntimeBenchmark.decide(
            comparison: comparison, thresholds: Self.thresholds)
        #expect(!decision.accepted)
        #expect(
            decision.blockers.contains {
                $0.contains("peak_physical_footprint_bytes") && $0.contains("not measured")
            })
    }

    @Test("a zero baseline yields no ratio and blocks rather than reading as infinite headroom")
    func refusesZeroBaseline() throws {
        let baseline = Self.record(
            runtime: "python-mlx-lm", startedAt: 100, finishedAt: 200,
            scenarios: [Self.scenario("short_prompt", ttft: 0, prefill: 0, decode: 0)])
        let comparison = try Self.admit(
            baseline: baseline, candidate: Self.candidate, requiredScenarios: [])
        let decision = RuntimeBenchmark.decide(
            comparison: comparison, thresholds: Self.thresholds)
        #expect(!decision.accepted)
        #expect(decision.blockers.contains { $0.contains("no usable ratio") })
        // `prompt_tokens` still divides -- the prompt lengths were measured and
        // are not zero. What must have no ratio is every metric whose baseline
        // reading was the zero.
        #expect(
            decision.deltas.allSatisfy {
                $0.ratio == nil || $0.metric.contains("footprint") || $0.metric == "prompt_tokens"
            })
    }

    @Test("a parity scenario the baseline won and the candidate lost is a blocker")
    func refusesLostParity() throws {
        let candidate = Self.record(
            runtime: "mlx-swift", startedAt: 300, finishedAt: 400,
            scenarios: [
                Self.scenario("short_prompt"),
                Self.scenario(
                    "context_75k", succeeded: false, failureMode: "metal OOM at 61k tokens",
                    ttft: nil, prefill: nil, decode: nil),
            ])
        let baseline = Self.record(
            runtime: "python-mlx-lm", startedAt: 100, finishedAt: 200,
            scenarios: [Self.scenario("short_prompt"), Self.scenario("context_75k")])
        let thresholds = RuntimeBenchmark.Thresholds(
            maxTimeToFirstTokenRatio: 1.10, minPrefillThroughputRatio: 0.90,
            minDecodeThroughputRatio: 0.90, maxPeakFootprintRatio: 1.10,
            maxPromptTokenSkewRatio: 1.10,
            paritySuccessScenarios: ["context_75k"], scoredScenarios: ["short_prompt"])
        let comparison = try Self.admit(
            baseline: baseline, candidate: candidate, requiredScenarios: [])
        let decision = RuntimeBenchmark.decide(comparison: comparison, thresholds: thresholds)
        #expect(!decision.accepted)
        #expect(
            decision.blockers.contains {
                $0.contains("context_75k") && $0.contains("metal OOM at 61k tokens")
            })
    }

    @Test("a parity scenario the baseline also lost is not held against the candidate")
    func toleratesParityTheIncumbentNeverCleared() throws {
        let failed = Self.scenario(
            "context_75k", succeeded: false, failureMode: "metal OOM", ttft: nil,
            prefill: nil, decode: nil)
        let baseline = Self.record(
            runtime: "python-mlx-lm", startedAt: 100, finishedAt: 200,
            scenarios: [Self.scenario("short_prompt"), failed])
        let candidate = Self.record(
            runtime: "mlx-swift", startedAt: 300, finishedAt: 400,
            scenarios: [Self.scenario("short_prompt"), failed])
        let thresholds = RuntimeBenchmark.Thresholds(
            maxTimeToFirstTokenRatio: 1.10, minPrefillThroughputRatio: 0.90,
            minDecodeThroughputRatio: 0.90, maxPeakFootprintRatio: 1.10,
            maxPromptTokenSkewRatio: 1.10,
            paritySuccessScenarios: ["context_75k"], scoredScenarios: ["short_prompt"])
        let comparison = try Self.admit(
            baseline: baseline, candidate: candidate, requiredScenarios: [])
        let decision = RuntimeBenchmark.decide(comparison: comparison, thresholds: thresholds)
        #expect(decision.accepted, "blockers: \(decision.blockers)")
    }

    @Test("a scored scenario that failed on either runtime blocks instead of being scored")
    func refusesScoringAFailedScenario() throws {
        let candidate = Self.record(
            runtime: "mlx-swift", startedAt: 300, finishedAt: 400,
            scenarios: [
                Self.scenario(
                    "short_prompt", succeeded: false, failureMode: "connection reset",
                    ttft: 0.1, prefill: 9999, decode: 9999)
            ])
        let comparison = try Self.admit(
            baseline: Self.baseline, candidate: candidate, requiredScenarios: [])
        let decision = RuntimeBenchmark.decide(
            comparison: comparison, thresholds: Self.thresholds)
        // The numbers a failed run leaves behind are the numbers it had when it
        // died. Scoring them would report a 100x speed-up for crashing early.
        #expect(!decision.accepted)
        #expect(decision.blockers.contains { $0.contains("did not succeed on both runtimes") })
        #expect(!decision.deltas.contains { $0.scenario == "short_prompt" })
    }

    // MARK: - Provenance: a record has to be tied to a run

    @Test("live effective configuration is sufficient without argv parsing")
    func derivesContextPolicyFromLiveConfiguration() {
        #expect(
            RuntimeBenchmark.contextPolicy(
                observing: .reported(76_800),
                generationConfiguration: .reported(
                    prefillStepSize: 999, reasoningEffort: "medium"))
                == "kv=76800;prefill-step=999;reasoning=medium")
    }

    @Test("a live context window overrides argv and malformed live values stay unread")
    func derivesContextPolicyFromLiveRuntime() {
        #expect(
            RuntimeBenchmark.contextPolicy(
                observing: .reported(76_800),
                generationConfiguration: Self.generationConfiguration)
                == "kv=76800;prefill-step=2048;reasoning=medium")
        #expect(
            RuntimeBenchmark.contextPolicy(
                observing: .unread,
                generationConfiguration: Self.generationConfiguration)
                == "kv=unread;prefill-step=2048;reasoning=medium")
        #expect(
            RuntimeBenchmark.contextPolicy(
                observing: .notReported,
                generationConfiguration: Self.generationConfiguration)
                == "kv=not-reported;prefill-step=2048;reasoning=medium")
        #expect(
            RuntimeContextWindow.read(fromModelsEntry: ["meta": ["n_ctx": "76800"]])
                == .unread)
        #expect(
            RuntimeContextWindow.read(fromModelsEntry: ["meta": ["n_ctx": 76_800]])
                == .reported(76_800))
        #expect(RuntimeContextWindow.reported(76_800).observation == .observed(76_800))
        #expect(RuntimeContextWindow.notReported.observation == .observedAbsent)
        #expect(RuntimeContextWindow.unread.observation == .notObserved)
        #expect(
            RuntimeGenerationConfiguration.read(fromModelsEntry: [
                "meta": [
                    "runtime_config": [
                        "prefill_step_size": 2_048, "reasoning_effort": "medium",
                    ]
                ]
            ]) == Self.generationConfiguration)
        #expect(
            RuntimeGenerationConfiguration.read(fromModelsEntry: [
                "meta": ["runtime_config": ["prefill_step_size": "2048"]]
            ]).prefillStepSize == .unread)
    }

    @Test("admission refuses a failed live context read instead of falling back to argv")
    func refusesUnreadLiveContextAtProductionAdmission() {
        #expect(
            throws: RuntimeBenchmark.AdmissionError.contextPolicyNotDerived(
                runtime: "python-mlx-lm",
                declared: "kv=76800;prefill-step=2048;reasoning=medium",
                derived: "kv=unread;prefill-step=2048;reasoning=medium")
        ) {
            try RuntimeBenchmark.admit(
                baseline: Self.baseline,
                baselineAttestation: Self.attestation(
                    for: Self.baseline, contextWindow: .unread),
                candidate: Self.candidate,
                candidateAttestation: Self.attestation(for: Self.candidate),
                requiredScenarios: [], gateBinaryDigest: Self.gateDigest)
        }
    }

    @Test("admission refuses an omitted live context bound despite any argv assertion")
    func refusesAbsentLiveContextBoundAtProductionAdmission() {
        #expect(
            throws: RuntimeBenchmark.AdmissionError.contextPolicyNotDerived(
                runtime: "python-mlx-lm",
                declared: "kv=76800;prefill-step=2048;reasoning=medium",
                derived: "kv=not-reported;prefill-step=2048;reasoning=medium")
        ) {
            try RuntimeBenchmark.admit(
                baseline: Self.baseline,
                baselineAttestation: Self.attestation(
                    for: Self.baseline, contextWindow: .notReported),
                candidate: Self.candidate,
                candidateAttestation: Self.attestation(for: Self.candidate),
                requiredScenarios: [], gateBinaryDigest: Self.gateDigest)
        }
    }

    @Test("a caller-authored context policy the server did not report is refused")
    func refusesUndrivedContextPolicy() {
        #expect(
            throws: RuntimeBenchmark.AdmissionError.contextPolicyNotDerived(
                runtime: "mlx-swift",
                declared: "kv=76800;prefill-step=2048;reasoning=medium",
                derived: "kv=76800;prefill-step=512;reasoning=medium")
        ) {
            try RuntimeBenchmark.admit(
                baseline: Self.baseline,
                baselineAttestation: Self.attestation(for: Self.baseline),
                candidate: Self.candidate,
                candidateAttestation: Self.attestation(
                    for: Self.candidate,
                    generationConfiguration: .reported(
                        prefillStepSize: 512, reasoningEffort: "medium")),
                requiredScenarios: [], gateBinaryDigest: Self.gateDigest)
        }
    }

    @Test("an omitted live prefill report is refused, not defaulted from argv")
    func refusesUnpinnedPrefillStep() {
        let launch = ["serve", "--model", Self.modelPath]
        let policy = RuntimeBenchmark.contextPolicy(
            observing: .reported(76_800),
            generationConfiguration: .notReported)
        var pins = Self.pins
        pins = RuntimeBenchmark.Pins(
            hostIdentity: pins.hostIdentity, modelPath: pins.modelPath,
            modelDigest: pins.modelDigest, quantization: pins.quantization,
            promptSuiteDigest: pins.promptSuiteDigest, contextPolicy: policy,
            maxOutputTokens: pins.maxOutputTokens, temperature: pins.temperature,
            topP: pins.topP, seed: pins.seed)
        let baseline = Self.record(
            runtime: "python-mlx-lm", pins: pins, startedAt: 100, finishedAt: 200,
            provenance: Self.provenance(launchArgv: launch))
        let candidate = Self.record(
            runtime: "mlx-swift", pins: pins, startedAt: 300, finishedAt: 400,
            provenance: Self.provenance(launchArgv: launch, executableDigest: Self.digest("12")))
        #expect(
            throws: RuntimeBenchmark.AdmissionError.unpinnedLaunchCondition(
                runtime: "python-mlx-lm", condition: "prefill-step=not-reported")
        ) {
            try RuntimeBenchmark.admit(
                baseline: baseline,
                baselineAttestation: Self.attestation(
                    for: baseline, generationConfiguration: .notReported),
                candidate: candidate,
                candidateAttestation: Self.attestation(
                    for: candidate, generationConfiguration: .notReported),
                requiredScenarios: [], gateBinaryDigest: Self.gateDigest)
        }
    }

    @Test("a record that names no revisions is refused")
    func refusesEmptyRevisions() {
        let candidate = Self.record(
            runtime: "mlx-swift", startedAt: 300, finishedAt: 400, revisions: [:],
            provenance: Self.provenance(executableDigest: Self.digest("12")))
        #expect(throws: RuntimeBenchmark.AdmissionError.missingRevisions(runtime: "mlx-swift")) {
            try Self.admit(
                baseline: Self.baseline, candidate: candidate, requiredScenarios: [])
        }
    }

    @Test("two runs configured by different launcher documents are not a comparison")
    func refusesConfigDigestMismatch() {
        let candidate = Self.record(
            runtime: "mlx-swift", startedAt: 300, finishedAt: 400,
            provenance: Self.provenance(
                configDigest: Self.digest("99"), executableDigest: Self.digest("12")))
        #expect(
            throws: RuntimeBenchmark.AdmissionError.configDigestMismatch(
                baseline: Self.digest("ab"), candidate: Self.digest("99"))
        ) {
            try Self.admit(
                baseline: Self.baseline, candidate: candidate, requiredScenarios: [])
        }
    }

    @Test("two runtime identities behind one executable are refused")
    func refusesSameLaunchExecutable() {
        let candidate = Self.record(
            runtime: "mlx-swift", startedAt: 300, finishedAt: 400,
            provenance: Self.provenance(executableDigest: Self.digest("cd")))
        #expect(
            throws: RuntimeBenchmark.AdmissionError.sameLaunchExecutable(
                digest: Self.digest("cd"))
        ) {
            try Self.admit(
                baseline: Self.baseline, candidate: candidate, requiredScenarios: [])
        }
    }

    @Test("a pinned model path the launch never received is refused")
    func refusesModelPathNotInLaunch() {
        let launch = ["serve", "--model", "/other", "--prefill-step-size", "2048"]
        let candidate = Self.record(
            runtime: "mlx-swift", startedAt: 300, finishedAt: 400,
            provenance: Self.provenance(launchArgv: launch, executableDigest: Self.digest("12")))
        #expect(
            throws: RuntimeBenchmark.AdmissionError.launchDoesNotCarryModel(
                runtime: "mlx-swift", modelPath: Self.modelPath)
        ) {
            try Self.admit(
                baseline: Self.baseline, candidate: candidate, requiredScenarios: [])
        }
    }

    @Test("a launcher command that does not carry the recorded config is refused")
    func refusesUnboundHarnessCommand() {
        let candidate = Self.record(
            runtime: "mlx-swift", startedAt: 300, finishedAt: 400,
            provenance: Self.provenance(
                executableDigest: Self.digest("12"),
                harnessCommand: [
                    "model-harness", "run", "qwen-benchmark", "--config", "/elsewhere",
                ]
            ))
        #expect(
            throws: RuntimeBenchmark.AdmissionError.harnessCommandUnbound(
                runtime: "mlx-swift", missing: "/tmp/model-harness.benchmark.toml")
        ) {
            try Self.admit(
                baseline: Self.baseline, candidate: candidate, requiredScenarios: [])
        }
    }

    @Test(
        "a digest that is not a digest is refused, field by field",
        arguments: [
            "provenance.configDigest", "provenance.driverDigest",
            "provenance.launchExecutableDigest",
        ])
    func refusesMalformedDigest(field: String) {
        let bogus = "not-a-digest"
        let provenance = RuntimeBenchmark.LaunchProvenance(
            driverCommand: ["python3", "runtime-benchmark.py"],
            driverDigest: field == "provenance.driverDigest" ? bogus : Self.digest("ef"),
            harnessCommand: [
                "model-harness", "run", "qwen-benchmark", "--config",
                "/tmp/model-harness.benchmark.toml",
            ],
            configPath: "/tmp/model-harness.benchmark.toml",
            configDigest: field == "provenance.configDigest" ? bogus : Self.digest("ab"),
            profile: "qwen-benchmark",
            launchExecutable: "/usr/local/bin/runtime",
            launchExecutableDigest: field == "provenance.launchExecutableDigest"
                ? bogus : Self.digest("12"),
            launchArgv: Self.launchArgv,
            runtimeProcessID: 4242)
        let candidate = Self.record(
            runtime: "mlx-swift", startedAt: 300, finishedAt: 400, provenance: provenance)
        #expect(
            throws: RuntimeBenchmark.AdmissionError.malformedDigest(
                runtime: "mlx-swift", field: field, value: bogus)
        ) {
            try Self.admit(
                baseline: Self.baseline, candidate: candidate, requiredScenarios: [])
        }
    }

    @Test("an uppercase hex digest is not a digest either")
    func refusesUppercaseDigest() {
        let candidate = Self.record(
            runtime: "mlx-swift", startedAt: 300, finishedAt: 400,
            provenance: Self.provenance(
                executableDigest: String(repeating: "AB", count: 32)))
        #expect(throws: (any Error).self) {
            try Self.admit(
                baseline: Self.baseline, candidate: candidate, requiredScenarios: [])
        }
    }

    @Test("a pass cannot contain more scenario time than it lasted")
    func refusesImpossibleTiming() {
        // Two seconds of pass, seven seconds of scenario.
        let candidate = Self.record(
            runtime: "mlx-swift", startedAt: 300, finishedAt: 302,
            provenance: Self.provenance(executableDigest: Self.digest("12")))
        #expect(
            throws: RuntimeBenchmark.AdmissionError.impossibleTiming(
                runtime: "mlx-swift", scenarioSeconds: 7, intervalSeconds: 2)
        ) {
            try Self.admit(
                baseline: Self.baseline, candidate: candidate, requiredScenarios: [])
        }
    }

    @Test("a record without provenance does not decode at all")
    func refusesRecordWithoutProvenance() throws {
        var object =
            try JSONSerialization.jsonObject(
                with: try JSONEncoder().encode(Self.baseline)) as! [String: Any]
        object.removeValue(forKey: "provenance")
        let data = try JSONSerialization.data(withJSONObject: object)
        #expect(throws: (any Error).self) {
            try RuntimeBenchmark.decodeRecord(path: "/tmp/forged.json", data: data)
        }
    }

    // MARK: - Comparability

    @Test("prompt-token skew is symmetric and at least one")
    func measuresPromptSkewSymmetrically() {
        let short = Self.scenario("short_prompt", promptTokens: 41)
        let long = Self.scenario("short_prompt", promptTokens: 79)
        let forward = RuntimeBenchmark.promptTokenSkew(baseline: long, candidate: short)
        let backward = RuntimeBenchmark.promptTokenSkew(baseline: short, candidate: long)
        #expect(forward != nil && backward != nil)
        #expect(abs((forward ?? 0) - (backward ?? 0)) < 1e-9)
        #expect((forward ?? 0) > 1.9)
        // Never invented from a missing count.
        #expect(
            RuntimeBenchmark.promptTokenSkew(
                baseline: Self.scenario("x", promptTokens: nil), candidate: short) == nil)
    }

    @Test("a scored scenario whose runtimes rendered different prompts is not scored")
    func refusesToScoreSkewedPrompts() throws {
        // Review's short-prompt observation: 41 Swift tokens against 79 Python
        // tokens, with a "0.751x TTFT win" on top of it.
        let baseline = Self.record(
            runtime: "python-mlx-lm", startedAt: 100, finishedAt: 200,
            scenarios: [Self.scenario("short_prompt", ttft: 1.234, promptTokens: 79)])
        let candidate = Self.record(
            runtime: "mlx-swift", startedAt: 300, finishedAt: 400,
            scenarios: [Self.scenario("short_prompt", ttft: 0.926, promptTokens: 41)],
            provenance: Self.provenance(executableDigest: Self.digest("12")))
        let comparison = try Self.admit(
            baseline: baseline, candidate: candidate, requiredScenarios: [])
        let decision = RuntimeBenchmark.decide(
            comparison: comparison, thresholds: Self.thresholds)
        #expect(!decision.accepted)
        let ttft = try #require(
            decision.deltas.first { $0.metric == "time_to_first_token_seconds" })
        // The favourable ratio is still reported -- and given no verdict.
        #expect(ttft.verdict == "non-comparable")
        #expect((ttft.ratio ?? 0) < 0.8)
        #expect(decision.blockers.contains { $0.contains("not a comparison") })
    }

    // MARK: - Memory is scored on work both runtimes completed

    @Test("the whole-process peak is not scored when the two passes did different work")
    func refusesWholeProcessPeakAfterAParityFailure() throws {
        // The exact shape review caught: the candidate aborts the expensive
        // scenario, so its whole-pass maximum is *lower* than the baseline that
        // completed it, and the old scoring called that "within".
        let thresholds = RuntimeBenchmark.Thresholds(
            maxTimeToFirstTokenRatio: 1.10, minPrefillThroughputRatio: 0.90,
            minDecodeThroughputRatio: 0.90, maxPeakFootprintRatio: 1.10,
            maxPromptTokenSkewRatio: 1.10,
            paritySuccessScenarios: ["context_75k"], scoredScenarios: [])
        let baseline = Self.record(
            runtime: "python-mlx-lm", startedAt: 100, finishedAt: 200,
            footprint: 48_438_825_960,
            scenarios: [Self.scenario("context_75k")])
        let candidate = Self.record(
            runtime: "mlx-swift", startedAt: 300, finishedAt: 400,
            footprint: 53_000_017_448,
            scenarios: [
                Self.scenario(
                    "context_75k", succeeded: false, failureMode: "process aborted",
                    ttft: nil, prefill: nil, decode: nil)
            ],
            provenance: Self.provenance(executableDigest: Self.digest("12")))
        let comparison = try Self.admit(
            baseline: baseline, candidate: candidate, requiredScenarios: [])
        let decision = RuntimeBenchmark.decide(comparison: comparison, thresholds: thresholds)
        let process = try #require(decision.deltas.first { $0.scenario == "process" })
        #expect(process.verdict == "non-comparable")
        #expect(decision.blockers.contains { $0.contains("did not complete the same parity") })
        #expect(!decision.accepted)
    }

    @Test("scenario-local footprints are scored, and a scenario-local blow-out is caught")
    func scoresScenarioLocalFootprint() throws {
        // Whole-process maxima that pass the band, scenario-local ones that do
        // not: 44.01 GiB against 31.47 GiB is 1.399x on the one scenario both
        // runtimes finished. A gate reading only the process figure calls this
        // 1.094x and accepts it.
        let baseline = Self.record(
            runtime: "python-mlx-lm", startedAt: 100, finishedAt: 200,
            footprint: 48_438_825_960,
            scenarios: [Self.scenario("short_prompt", footprint: 33_788_449_720)])
        let candidate = Self.record(
            runtime: "mlx-swift", startedAt: 300, finishedAt: 400,
            footprint: 53_000_017_448,
            scenarios: [Self.scenario("short_prompt", footprint: 47_257_540_040)],
            provenance: Self.provenance(executableDigest: Self.digest("12")))
        let comparison = try Self.admit(
            baseline: baseline, candidate: candidate, requiredScenarios: [])
        let decision = RuntimeBenchmark.decide(
            comparison: comparison, thresholds: Self.thresholds)
        let scenarioDelta = try #require(
            decision.deltas.first {
                $0.scenario == "short_prompt" && $0.metric == "peak_physical_footprint_bytes"
            })
        #expect(scenarioDelta.verdict == "outside")
        #expect((scenarioDelta.ratio ?? 0) > 1.39)
        let processDelta = try #require(
            decision.deltas.first { $0.scenario == "process" })
        #expect(processDelta.verdict == "within")
        // The process axis passing does not rescue the scenario axis failing.
        #expect(!decision.accepted)
    }

    @Test("an unmeasured scenario-local footprint blocks rather than falling back")
    func refusesUnmeasuredScenarioFootprint() throws {
        let candidate = Self.record(
            runtime: "mlx-swift", startedAt: 300, finishedAt: 400,
            scenarios: [Self.scenario("short_prompt", footprint: nil)],
            provenance: Self.provenance(executableDigest: Self.digest("12")))
        let comparison = try Self.admit(
            baseline: Self.baseline, candidate: candidate, requiredScenarios: [])
        let decision = RuntimeBenchmark.decide(
            comparison: comparison, thresholds: Self.thresholds)
        #expect(!decision.accepted)
        #expect(
            decision.blockers.contains {
                $0.contains("short_prompt/peak_physical_footprint_bytes")
                    && $0.contains("not measured")
            })
    }

    @Test("thresholds round-trip through JSON")
    func roundTripsThresholds() throws {
        let encoded = try JSONEncoder().encode(Self.thresholds)
        let decoded = try JSONDecoder().decode(RuntimeBenchmark.Thresholds.self, from: encoded)
        #expect(decoded == Self.thresholds)
    }
}

/// Field-wise copy helper, so the pin table above can change exactly one thing.
private func mutate(
    _ pins: RuntimeBenchmark.Pins,
    hostIdentity: String? = nil,
    modelPath: String? = nil,
    modelDigest: String? = nil,
    quantization: String? = nil,
    promptSuiteDigest: String? = nil,
    contextPolicy: String? = nil,
    maxOutputTokens: Int? = nil,
    temperature: Double? = nil,
    topP: Double? = nil,
    seed: Int? = nil
) -> RuntimeBenchmark.Pins {
    RuntimeBenchmark.Pins(
        hostIdentity: hostIdentity ?? pins.hostIdentity,
        modelPath: modelPath ?? pins.modelPath,
        modelDigest: modelDigest ?? pins.modelDigest,
        quantization: quantization ?? pins.quantization,
        promptSuiteDigest: promptSuiteDigest ?? pins.promptSuiteDigest,
        contextPolicy: contextPolicy ?? pins.contextPolicy,
        maxOutputTokens: maxOutputTokens ?? pins.maxOutputTokens,
        temperature: temperature ?? pins.temperature,
        topP: topP ?? pins.topP,
        seed: seed ?? pins.seed)
}

/// One entry per pinned field: the field's name, and a copy of the pins that
/// differs from ``RuntimeBenchmarkTests/pins`` in that field alone.
///
/// A field missing from this table is a field the suite never checks, so the
/// table is the coverage claim: it must name every property of
/// ``RuntimeBenchmark/Pins``.
private let mismatchedPins: [(String, RuntimeBenchmark.Pins)] = {
    let base = RuntimeBenchmarkTests.pins
    return [
        ("hostIdentity", mutate(base, hostIdentity: "Mac16,6/137438953472/25F80")),
        ("modelPath", mutate(base, modelPath: "/Users/alexis/src/local-models/other")),
        ("modelDigest", mutate(base, modelDigest: "deadbeef")),
        ("quantization", mutate(base, quantization: "6bit/group64/affine")),
        ("promptSuiteDigest", mutate(base, promptSuiteDigest: "cc22dd")),
        ("contextPolicy", mutate(base, contextPolicy: "kv-4096")),
        ("maxOutputTokens", mutate(base, maxOutputTokens: 512)),
        ("temperature", mutate(base, temperature: 0.6)),
        ("topP", mutate(base, topP: 0.95)),
        ("seed", mutate(base, seed: 4321)),
    ]
}()

/// Guards the table above against silently falling behind ``RuntimeBenchmark/Pins``.
@Suite("runtime benchmark pin coverage")
struct RuntimeBenchmarkPinCoverageTests {
    @Test("the mismatch table names every pinned field")
    func tableCoversEveryPin() throws {
        let encoded = try JSONEncoder().encode(RuntimeBenchmarkTests.pins)
        let object = try #require(
            try JSONSerialization.jsonObject(with: encoded) as? [String: Any])
        let named = Set(mismatchedPins.map(\.0))
        // A pin that nothing in the table differs on is a pin the comparison
        // gate is never tested against, which is how a dropped field survives.
        #expect(named == Set(object.keys), "untested pins: \(Set(object.keys).subtracting(named))")
    }
}

/// The clauses that check a record against something it did not author.
///
/// Every test here is a narrowing attack, not a deletion: the record pair is
/// the same pair the suite above admits, fully populated and internally
/// consistent, and exactly one thing about what the gate *observed* is moved.
/// A gate that only refused missing attestations would pass none of them.
///
/// Production call site: `BenchmarkCompareCommand.run(arguments:)` loads both
/// documents from `--attestations` and passes them, with its own binary's
/// digest, into `RuntimeBenchmark.admit`. The one shape that cannot be reached
/// from here — a directory with no attestation in it at all — is driven through
/// that subcommand by `scripts/benchmark-gate-smoke.sh`, because the parameter
/// is non-optional and absence is therefore not expressible in this suite.
@Suite("runtime benchmark attestation gate")
struct RuntimeBenchmarkAttestationTests {
    typealias Fixture = RuntimeBenchmarkTests

    static let baseline = Fixture.baseline
    static let candidate = Fixture.candidate

    static func admit(
        baselineAttestation: RuntimeAttestation,
        candidateAttestation: RuntimeAttestation,
        gateBinaryDigest: String = Fixture.gateDigest
    ) throws -> RuntimeBenchmark.Comparison {
        try RuntimeBenchmark.admit(
            baseline: baseline, baselineAttestation: baselineAttestation,
            candidate: candidate, candidateAttestation: candidateAttestation,
            requiredScenarios: ["short_prompt"], gateBinaryDigest: gateBinaryDigest)
    }

    static func expectRefusal(
        _ expected: RuntimeBenchmark.AdmissionError,
        baselineAttestation: RuntimeAttestation = Fixture.attestation(for: Self.baseline),
        candidateAttestation: RuntimeAttestation? = nil,
        gateBinaryDigest: String = Fixture.gateDigest
    ) {
        #expect(
            throws: expected,
            performing: {
                try admit(
                    baselineAttestation: baselineAttestation,
                    candidateAttestation: candidateAttestation
                        ?? Fixture.attestation(for: Self.candidate),
                    gateBinaryDigest: gateBinaryDigest)
            })
    }

    @Test("a pair the gate watched from open to close is admitted")
    func admitsObservedPair() throws {
        let comparison = try Self.admit(
            baselineAttestation: Fixture.attestation(for: Self.baseline),
            candidateAttestation: Fixture.attestation(for: Self.candidate))
        #expect(comparison.sharedScenarios == ["short_prompt"])
    }

    @Test("a binary that did not observe the runs cannot judge them")
    func refusesForeignGateBinary() {
        // The revision-2 defect in one line: `19c54c…` served and `3e5fdcc…`
        // decided. The attestations name their observer, so the mismatch is
        // now a refusal rather than a footnote.
        Self.expectRefusal(
            .judgingBinaryDidNotObserve(
                runtime: "python-mlx-lm", observing: Fixture.gateDigest,
                judging: Fixture.digest("99")),
            gateBinaryDigest: Fixture.digest("99"))
    }

    @Test("an attestation the gate opened and never closed is not a watched pass")
    func refusesOpenAttestation() {
        Self.expectRefusal(
            .attestationNeverClosed(runtime: "mlx-swift"),
            candidateAttestation: Fixture.attestation(for: Self.candidate, closedAt: .some(nil)))
    }

    @Test("an attestation that never established what was being served is refused")
    func refusesUnaskedServedModel() {
        Self.expectRefusal(
            .attestationNeverClosed(runtime: "mlx-swift"),
            candidateAttestation: Fixture.attestation(
                for: Self.candidate, servedModelID: .some(nil)))
    }

    @Test("a runtime the gate watched serving a different model is refused")
    func refusesOtherServedModel() {
        Self.expectRefusal(
            .attestationDisagrees(
                runtime: "mlx-swift", field: "pins.modelPath",
                observed: "/Users/alexis/src/local-models/other",
                declared: Fixture.modelPath),
            candidateAttestation: Fixture.attestation(
                for: Self.candidate, servedModelID: "/Users/alexis/src/local-models/other"))
    }

    @Test(
        "a record that disagrees with the observation on any bound field is refused",
        arguments: [
            (
                "provenance.runtimeProcessID", "9999", "4242",
                Fixture.attestation(for: Self.candidate, processID: 9999)
            ),
            (
                "provenance.launchExecutableDigest", Fixture.digest("aa"),
                Fixture.digest("12"),
                Fixture.attestation(for: Self.candidate, executableDigest: Fixture.digest("aa"))
            ),
            (
                "provenance.configDigest", Fixture.digest("bb"), Fixture.digest("ab"),
                Fixture.attestation(for: Self.candidate, configDigest: Fixture.digest("bb"))
            ),
            (
                "provenance.profile", "some-other-profile", "qwen-benchmark",
                Fixture.attestation(for: Self.candidate, profile: "some-other-profile")
            ),
        ])
    func refusesObservationMismatch(
        field: String, observed: String, declared: String, attestation: RuntimeAttestation
    ) {
        Self.expectRefusal(
            .attestationDisagrees(
                runtime: "mlx-swift", field: field, observed: observed, declared: declared),
            candidateAttestation: attestation)
    }

    @Test("an attestation for the wrong runtime is refused")
    func refusesRuntimeMismatch() {
        let foreign = RuntimeAttestation(
            runtime: "some-third-runtime",
            processID: Self.candidate.provenance.runtimeProcessID,
            processStartUnixSeconds: 300,
            observedExecutablePath: Self.candidate.provenance.launchExecutable,
            observedExecutableDigest: Self.candidate.provenance.launchExecutableDigest,
            configPath: Self.candidate.provenance.configPath,
            configDigest: Self.candidate.provenance.configDigest,
            profile: Self.candidate.provenance.profile,
            openedAtUnixSeconds: 300, closedAtUnixSeconds: 400,
            servedModelID: Fixture.modelPath,
            observedContextWindow: .reported(76_800),
            observedGenerationConfiguration: Fixture.generationConfiguration,
            gateBinaryDigest: Fixture.gateDigest,
            transcriptDigest: RuntimeBenchmark.transcriptDigest(of: Self.candidate))
        Self.expectRefusal(
            .attestationDisagrees(
                runtime: "mlx-swift", field: "runtime", observed: "some-third-runtime",
                declared: "mlx-swift"),
            candidateAttestation: foreign)
    }

    @Test("an observation that began before the pass it certifies is refused")
    func refusesObservationBeforeRun() {
        Self.expectRefusal(
            .attestationOutsideRun(
                runtime: "mlx-swift",
                detail:
                    "the gate opened at 250.0, before the pass claims to have started at 300.0"),
            candidateAttestation: Fixture.attestation(for: Self.candidate, openedAt: 250))
    }

    @Test("an observation that ended after the pass it certifies is refused")
    func refusesObservationAfterRun() {
        Self.expectRefusal(
            .attestationOutsideRun(
                runtime: "mlx-swift",
                detail:
                    "the gate closed at 500.0, after the pass claims to have finished at 400.0"),
            candidateAttestation: Fixture.attestation(for: Self.candidate, closedAt: 500))
    }

    @Test("scenarios that did not fit inside the watched window are refused")
    func refusesScenariosOutsideObservation() {
        // The window is narrowed to five seconds and the single scenario claims
        // seven. Narrowing rather than deleting: the clause has to refuse a
        // window that is merely *too short*, not only one that is absent.
        Self.expectRefusal(
            .attestationDoesNotCoverScenarios(
                runtime: "mlx-swift", scenarioSeconds: 7, observedSeconds: 5),
            candidateAttestation: Fixture.attestation(
                for: Self.candidate, openedAt: 300, closedAt: 305))
    }

    @Test("one process cannot be two runtimes")
    func refusesSharedProcess() {
        Self.expectRefusal(
            .attestationsShareProcess(processID: 4242),
            baselineAttestation: Fixture.attestation(for: Self.baseline, processStart: 42),
            candidateAttestation: Fixture.attestation(for: Self.candidate, processStart: 42))
    }

    @Test("a pid the operating system handed out twice is not one process")
    func admitsReusedProcessIdentifier() throws {
        // The narrowing that keeps the clause above honest. The two passes are
        // sequential, so the second runtime can legitimately receive the number
        // the first one released; refusing on the pid alone would reject a
        // correct benchmark on a coincidence.
        let comparison = try Self.admit(
            baselineAttestation: Fixture.attestation(for: Self.baseline, processStart: 100),
            candidateAttestation: Fixture.attestation(for: Self.candidate, processStart: 300))
        #expect(comparison.sharedScenarios == ["short_prompt"])
    }
}

/// The reasoning/chat-template half of the context policy.
///
/// Review measured the defect this closes: the Swift profile passed
/// `--reasoning-effort medium` and the Python profile passed nothing, so this
/// model's template resolved `xhigh`, injected an extra system instruction and
/// added a constant 38 tokens to every prompt in the suite. Revision 2 reported
/// the consequence as a 1.93x runtime skew on `short_prompt`.
@Suite("runtime benchmark reasoning policy derivation")
struct RuntimeBenchmarkReasoningPolicyTests {
    typealias Fixture = RuntimeBenchmarkTests

    @Test("reasoning is read from the running server")
    func readsLiveReasoning() {
        #expect(
            RuntimeBenchmark.contextPolicy(
                observing: .reported(76_800),
                generationConfiguration: .reported(
                    prefillStepSize: 2_048, reasoningEffort: "medium"))
                == "kv=76800;prefill-step=2048;reasoning=medium")
    }

    @Test("missing and malformed live reasoning are distinct refusals")
    func refusesMissingAndMalformedLiveReasoning() {
        let missing = RuntimeGenerationConfiguration(
            prefillStepSize: .reported("2048"), reasoningEffort: .notReported)
        let malformed = RuntimeGenerationConfiguration(
            prefillStepSize: .reported("2048"), reasoningEffort: .unread)
        #expect(
            RuntimeBenchmark.contextPolicy(
                observing: .reported(76_800), generationConfiguration: missing)
                == "kv=76800;prefill-step=2048;reasoning=not-reported")
        #expect(
            RuntimeBenchmark.contextPolicy(
                observing: .reported(76_800), generationConfiguration: malformed)
                == "kv=76800;prefill-step=2048;reasoning=unread")
    }

    @Test("two runtimes on different reasoning policies are not a comparison")
    func refusesDifferingReasoningPolicies() {
        let baselineArgv = [
            "--model", RuntimeBenchmarkTests.modelPath, "--prefill-step-size", "2048",
            "--chat-template-args", #"{"reasoning_effort": "xhigh"}"#,
        ]
        let candidateArgv = [
            "serve", "--model", RuntimeBenchmarkTests.modelPath, "--prefill-step-size", "2048",
            "--reasoning-effort", "medium",
        ]
        func pins(reasoning: String) -> RuntimeBenchmark.Pins {
            RuntimeBenchmark.Pins(
                hostIdentity: "MacBookPro18,2/68719476736/25F80",
                modelPath: RuntimeBenchmarkTests.modelPath,
                modelDigest: "9f2c1a", quantization: "8bit/group64/affine",
                promptSuiteDigest: "aa11bb",
                contextPolicy: RuntimeBenchmark.contextPolicy(
                    observing: .reported(76_800),
                    generationConfiguration: .reported(
                        prefillStepSize: 2_048, reasoningEffort: reasoning)),
                maxOutputTokens: 256, temperature: 0, topP: 1, seed: 1234)
        }
        let baseline = RuntimeBenchmarkTests.record(
            runtime: "python-mlx-lm",
            pins: pins(reasoning: "xhigh"),
            startedAt: 100, finishedAt: 200,
            provenance: RuntimeBenchmarkTests.provenance(
                launchArgv: baselineArgv, executableDigest: RuntimeBenchmarkTests.digest("cd")))
        let candidate = RuntimeBenchmarkTests.record(
            runtime: "mlx-swift",
            pins: pins(reasoning: "medium"),
            startedAt: 300, finishedAt: 400,
            provenance: RuntimeBenchmarkTests.provenance(
                launchArgv: candidateArgv, executableDigest: RuntimeBenchmarkTests.digest("12")))
        #expect(
            throws: RuntimeBenchmark.AdmissionError.pinMismatch(
                field: "contextPolicy",
                baseline: "kv=76800;prefill-step=2048;reasoning=xhigh",
                candidate: "kv=76800;prefill-step=2048;reasoning=medium"),
            performing: {
                try RuntimeBenchmark.admit(
                    baseline: baseline,
                    baselineAttestation: Fixture.attestation(
                        for: baseline,
                        generationConfiguration: .reported(
                            prefillStepSize: 2_048, reasoningEffort: "xhigh")),
                    candidate: candidate,
                    candidateAttestation: Fixture.attestation(for: candidate),
                    requiredScenarios: ["short_prompt"],
                    gateBinaryDigest: Fixture.gateDigest)
            })
    }
}

/// The clause that closes review's third reproduction.
///
/// Rounds 1, 2 and 3 all ended the same way: a caller assembled documents that
/// agreed with each other, and the production gate scored them `accepted=true`.
/// Round 3's version used no forgery at all — two real placeholder HTTP servers
/// that answered `GET /v1/models`, two real attestations minted for them by the
/// shipped `benchmark-attest` subcommand, and a set of measurements the caller
/// simply typed. Everything the gate checked was true; nothing it checked was
/// about a benchmark.
///
/// These tests are about the link that was missing: a measurement has to carry
/// the exchanges it came from, those exchanges have to include a completion the
/// runtime actually served, and the whole thing has to digest to what the
/// observing invocation sealed.
@Suite("runtime benchmark transcript binding")
struct RuntimeBenchmarkTranscriptTests {
    typealias Fixture = RuntimeBenchmarkTests

    static let baseline = Fixture.baseline

    /// A candidate built with a chosen scenario, sealed honestly.
    ///
    /// The seal is recomputed from whatever record the test builds, so a test
    /// that wants to attack the *seal* has to break it deliberately; the rest
    /// attack the transcript itself with the seal intact.
    static func pair(
        candidateScenarios: [RuntimeBenchmark.ScenarioResult],
        seal: String?? = nil
    ) -> (RuntimeBenchmark.RunRecord, RuntimeAttestation) {
        let candidate = RuntimeBenchmark.RunRecord(
            runtime: "mlx-swift", revisions: ["runtime": "x"],
            command: Fixture.provenance(executableDigest: Fixture.digest("12")).harnessCommand,
            provenance: Fixture.provenance(executableDigest: Fixture.digest("12")),
            pins: Fixture.pins, startedAtUnixSeconds: 300, finishedAtUnixSeconds: 400,
            peakPhysicalFootprintBytes: 29_000_000_000, scenarios: candidateScenarios,
            declaredAsymmetries: [])
        return (candidate, Fixture.attestation(for: candidate, transcriptDigest: seal))
    }

    static func admit(
        candidate: RuntimeBenchmark.RunRecord, attestation: RuntimeAttestation
    ) throws -> RuntimeBenchmark.Comparison {
        try RuntimeBenchmark.admit(
            baseline: baseline, baselineAttestation: Fixture.attestation(for: baseline),
            candidate: candidate, candidateAttestation: attestation,
            requiredScenarios: ["short_prompt"], gateBinaryDigest: Fixture.gateDigest)
    }

    static func expectRefusal(
        _ expected: RuntimeBenchmark.AdmissionError,
        candidateScenarios: [RuntimeBenchmark.ScenarioResult],
        seal: String?? = nil
    ) {
        let (candidate, attestation) = pair(candidateScenarios: candidateScenarios, seal: seal)
        #expect(
            throws: expected,
            performing: { try admit(candidate: candidate, attestation: attestation) })
    }

    /// One exchange, placed inside the candidate's window.
    static func exchange(
        path: String = RuntimeBenchmark.ScenarioTranscript.completionPath,
        status: Int = 200,
        responseBytes: Int = 4096,
        at instant: Double = 350
    ) -> RuntimeBenchmark.ScenarioTranscript.Exchange {
        RuntimeBenchmark.ScenarioTranscript.Exchange(
            method: "POST", path: path, requestDigest: Fixture.digest("1a"),
            requestByteCount: 900, status: status, responseDigest: Fixture.digest("2b"),
            responseByteCount: responseBytes, sentAtUnixSeconds: instant,
            firstByteAtUnixSeconds: instant, lastByteAtUnixSeconds: instant)
    }

    static func scenario(
        _ name: String = "short_prompt",
        succeeded: Bool = true,
        exchanges: [RuntimeBenchmark.ScenarioTranscript.Exchange]?
    ) -> RuntimeBenchmark.ScenarioResult {
        RuntimeBenchmark.ScenarioResult(
            name: name, succeeded: succeeded, promptTokens: 512, completionTokens: 64,
            timeToFirstTokenSeconds: 1.0, prefillTokensPerSecond: 100,
            decodeTokensPerSecond: 10, wallClockSeconds: 7,
            peakPhysicalFootprintBytes: 29_000_000_000,
            processPeakSoFarBytes: 29_000_000_000,
            transcript: exchanges.map { RuntimeBenchmark.ScenarioTranscript(exchanges: $0) })
    }

    @Test("a pass whose measurements carry the completions they came from is admitted")
    func admitsSealedTranscript() throws {
        let (candidate, attestation) = Self.pair(
            candidateScenarios: [Self.scenario(exchanges: [Self.exchange()])])
        let comparison = try Self.admit(candidate: candidate, attestation: attestation)
        #expect(comparison.sharedScenarios == ["short_prompt"])
    }

    @Test("a required scenario with no transcript at all is refused")
    func refusesMissingTranscript() {
        Self.expectRefusal(
            .scenarioWithoutTranscript(runtime: "mlx-swift", scenario: "short_prompt"),
            candidateScenarios: [Self.scenario(exchanges: nil)])
    }

    @Test("review's reproduction: a pass that only ever answered /v1/models is inadmissible")
    func refusesModelsOnlyPass() {
        // The exact shape of round 3's attack, as a record. Two placeholder
        // servers answered this one endpoint; the numbers beside them were
        // typed. The refusal is inadmissible rather than a rejection on the
        // numbers, and the distinction is the whole finding: "the candidate
        // lost" is a comparison this gate agreed to make, and there was no
        // comparison here to make.
        Self.expectRefusal(
            .scenarioSuccessWithoutCompletion(runtime: "mlx-swift", scenario: "short_prompt"),
            candidateScenarios: [
                Self.scenario(exchanges: [Self.exchange(path: "/v1/models")])
            ])
    }

    @Test(
        "a pass in which nothing was ever served is inadmissible even when nothing claims success")
    func refusesPassWithNoServedCompletion() {
        // The same attack with the success flags dropped — a caller who reads
        // the refusal above and tries the honest-looking version of it. There
        // is still no benchmark here, so there is still nothing to judge.
        Self.expectRefusal(
            .transcriptCarriesNoCompletion(runtime: "mlx-swift"),
            candidateScenarios: [
                Self.scenario(
                    succeeded: false, exchanges: [Self.exchange(path: "/v1/models")])
            ])
    }

    @Test("a 200 with an empty body is not a served completion")
    func refusesEmptyCompletionBody() {
        // Narrower than the endpoint check above: the placeholder that learns to
        // answer the right *path* still has to return something.
        Self.expectRefusal(
            .scenarioSuccessWithoutCompletion(runtime: "mlx-swift", scenario: "short_prompt"),
            candidateScenarios: [Self.scenario(exchanges: [Self.exchange(responseBytes: 0)])])
    }

    @Test("a completion the runtime refused is not a served completion")
    func refusesNon200Completion() {
        Self.expectRefusal(
            .scenarioSuccessWithoutCompletion(runtime: "mlx-swift", scenario: "short_prompt"),
            candidateScenarios: [Self.scenario(exchanges: [Self.exchange(status: 500)])])
    }

    @Test("an observation that seals no transcript witnesses a process, not a measurement")
    func refusesUnsealedAttestation() {
        Self.expectRefusal(
            .attestationSealsNoTranscript(runtime: "mlx-swift"),
            candidateScenarios: [Self.scenario(exchanges: [Self.exchange()])],
            seal: .some(nil))
    }

    @Test("measurements edited after the observation no longer digest to what was sealed")
    func refusesResealedMeasurements() throws {
        // The narrowing case. Everything here is real: a real pass, a real
        // observation, a real served completion. One number was changed
        // afterwards — a 1.0s time to first token became 0.1s — and that is
        // enough, because the seal covers the measurements and not only the
        // exchanges. Sealing the exchanges alone would leave exactly this edit
        // free, which is the same bypass one level down.
        let honest = Self.scenario(exchanges: [Self.exchange()])
        let sealedOverHonest = RuntimeBenchmark.transcriptDigest(
            of: Self.pair(candidateScenarios: [honest]).0)
        let edited = RuntimeBenchmark.ScenarioResult(
            name: honest.name, succeeded: true, promptTokens: honest.promptTokens,
            completionTokens: honest.completionTokens, timeToFirstTokenSeconds: 0.1,
            prefillTokensPerSecond: honest.prefillTokensPerSecond,
            decodeTokensPerSecond: honest.decodeTokensPerSecond,
            wallClockSeconds: honest.wallClockSeconds,
            peakPhysicalFootprintBytes: honest.peakPhysicalFootprintBytes,
            processPeakSoFarBytes: honest.processPeakSoFarBytes,
            transcript: honest.transcript)
        let (candidate, _) = Self.pair(candidateScenarios: [edited])
        let attestation = Fixture.attestation(
            for: candidate, transcriptDigest: .some(sealedOverHonest))
        #expect(throws: RuntimeBenchmark.AdmissionError.self) {
            try Self.admit(candidate: candidate, attestation: attestation)
        }
        #expect(RuntimeBenchmark.transcriptDigest(of: candidate) != sealedOverHonest)
    }

    @Test("an exchange outside the window it was observed in is refused")
    func refusesExchangeOutsideWindow() {
        let (candidate, _) = Self.pair(
            candidateScenarios: [Self.scenario(exchanges: [Self.exchange(at: 500)])])
        let attestation = Fixture.attestation(for: candidate)
        #expect(throws: RuntimeBenchmark.AdmissionError.self) {
            try Self.admit(candidate: candidate, attestation: attestation)
        }
    }

    @Test("an exchange that began inside the window and ended outside it is refused")
    func refusesExchangeStraddlingWindowEnd() {
        // Narrower than the case above, and the one a window check written
        // against the send time alone would admit: the request went out while
        // the gate was watching and came back long after it stopped. Whatever
        // that response was measured against, it was not the observed process.
        let straddling = RuntimeBenchmark.ScenarioTranscript.Exchange(
            method: "POST", path: RuntimeBenchmark.ScenarioTranscript.completionPath,
            requestDigest: Fixture.digest("1a"), requestByteCount: 900, status: 200,
            responseDigest: Fixture.digest("2b"), responseByteCount: 4096,
            sentAtUnixSeconds: 350, firstByteAtUnixSeconds: 350,
            lastByteAtUnixSeconds: 5000)
        let (candidate, _) = Self.pair(
            candidateScenarios: [Self.scenario(exchanges: [straddling])])
        let attestation = Fixture.attestation(for: candidate)
        #expect(throws: RuntimeBenchmark.AdmissionError.self) {
            try Self.admit(candidate: candidate, attestation: attestation)
        }
    }

    @Test("the seal is a function of the record and nothing else")
    func sealIsDeterministic() {
        let (first, _) = Self.pair(candidateScenarios: [Self.scenario(exchanges: [Self.exchange()])]
        )
        let (second, _) = Self.pair(
            candidateScenarios: [Self.scenario(exchanges: [Self.exchange()])])
        #expect(
            RuntimeBenchmark.transcriptDigest(of: first)
                == RuntimeBenchmark.transcriptDigest(of: second))
    }

    /// Rebuild a scenario with one field replaced.
    ///
    /// Written out rather than done with a mutating copy because the type is a
    /// value with `let` members, and the point of the test below is that the
    /// seal notices *each* of these fields individually.
    static func replacing(
        _ base: RuntimeBenchmark.ScenarioResult,
        name: String? = nil,
        succeeded: Bool? = nil,
        failureMode: String?? = nil,
        promptTokens: Int?? = nil,
        completionTokens: Int?? = nil,
        timeToFirstToken: Double?? = nil,
        prefill: Double?? = nil,
        decode: Double?? = nil,
        wallClock: Double?? = nil,
        windowPeak: Int?? = nil,
        processPeak: Int?? = nil,
        hostLoad: Double?? = nil
    ) -> RuntimeBenchmark.ScenarioResult {
        RuntimeBenchmark.ScenarioResult(
            name: name ?? base.name, succeeded: succeeded ?? base.succeeded,
            failureMode: failureMode ?? base.failureMode,
            promptTokens: promptTokens ?? base.promptTokens,
            completionTokens: completionTokens ?? base.completionTokens,
            timeToFirstTokenSeconds: timeToFirstToken ?? base.timeToFirstTokenSeconds,
            prefillTokensPerSecond: prefill ?? base.prefillTokensPerSecond,
            decodeTokensPerSecond: decode ?? base.decodeTokensPerSecond,
            wallClockSeconds: wallClock ?? base.wallClockSeconds,
            peakPhysicalFootprintBytes: windowPeak ?? base.peakPhysicalFootprintBytes,
            processPeakSoFarBytes: processPeak ?? base.processPeakSoFarBytes,
            hostLoadAverageMax: hostLoad ?? base.hostLoadAverageMax,
            transcript: base.transcript)
    }

    @Test("the seal covers every measurement a decision reads, one field at a time")
    func sealCoversEveryScoredField() {
        // A seal that covered only the exchanges would leave every number in
        // this list free to be anything, which is the same bypass one level
        // down. `succeeded` is in the list for its own reason: `decide` reads
        // it before it reads any measurement, so a flag flipped after the fact
        // turns a scenario the runtime failed into one it won without touching
        // a single number.
        let base = Self.scenario(exchanges: [Self.exchange()])
        let sealed = RuntimeBenchmark.transcriptDigest(of: Self.pair(candidateScenarios: [base]).0)
        let variants: [(String, RuntimeBenchmark.ScenarioResult)] = [
            ("name", Self.replacing(base, name: "some_other_scenario")),
            ("succeeded", Self.replacing(base, succeeded: false)),
            ("failureMode", Self.replacing(base, failureMode: .some("it did not"))),
            ("promptTokens", Self.replacing(base, promptTokens: .some(1))),
            ("completionTokens", Self.replacing(base, completionTokens: .some(1))),
            ("timeToFirstTokenSeconds", Self.replacing(base, timeToFirstToken: .some(0.001))),
            ("prefillTokensPerSecond", Self.replacing(base, prefill: .some(999_999))),
            ("decodeTokensPerSecond", Self.replacing(base, decode: .some(999_999))),
            ("wallClockSeconds", Self.replacing(base, wallClock: .some(0.001))),
            ("peakPhysicalFootprintBytes", Self.replacing(base, windowPeak: .some(1))),
            ("processPeakSoFarBytes", Self.replacing(base, processPeak: .some(1))),
            ("hostLoadAverageMax", Self.replacing(base, hostLoad: .some(41.0))),
        ]
        for (field, variant) in variants {
            let digest = RuntimeBenchmark.transcriptDigest(
                of: Self.pair(candidateScenarios: [variant]).0)
            #expect(digest != sealed, "the seal does not cover \(field)")
        }
    }

    @Test("the seal covers every field of every exchange, one at a time")
    func sealCoversEveryExchangeField() {
        let base = Self.exchange()
        let sealed = RuntimeBenchmark.transcriptDigest(
            of: Self.pair(candidateScenarios: [Self.scenario(exchanges: [base])]).0)
        func exchange(
            method: String? = nil, path: String? = nil, requestDigest: String? = nil,
            requestBytes: Int? = nil, status: Int? = nil, responseDigest: String? = nil,
            responseBytes: Int? = nil, sentAt: Double? = nil, firstByteAt: Double?? = nil,
            lastByteAt: Double? = nil
        ) -> RuntimeBenchmark.ScenarioTranscript.Exchange {
            RuntimeBenchmark.ScenarioTranscript.Exchange(
                method: method ?? base.method, path: path ?? base.path,
                requestDigest: requestDigest ?? base.requestDigest,
                requestByteCount: requestBytes ?? base.requestByteCount,
                status: status ?? base.status,
                responseDigest: responseDigest ?? base.responseDigest,
                responseByteCount: responseBytes ?? base.responseByteCount,
                sentAtUnixSeconds: sentAt ?? base.sentAtUnixSeconds,
                firstByteAtUnixSeconds: firstByteAt ?? base.firstByteAtUnixSeconds,
                lastByteAtUnixSeconds: lastByteAt ?? base.lastByteAtUnixSeconds)
        }
        let variants: [(String, RuntimeBenchmark.ScenarioTranscript.Exchange)] = [
            ("method", exchange(method: "GET")),
            ("path", exchange(path: "/v1/models")),
            ("requestDigest", exchange(requestDigest: Fixture.digest("cc"))),
            ("requestByteCount", exchange(requestBytes: 1)),
            ("status", exchange(status: 500)),
            ("responseDigest", exchange(responseDigest: Fixture.digest("dd"))),
            ("responseByteCount", exchange(responseBytes: 1)),
            ("sentAtUnixSeconds", exchange(sentAt: 351)),
            ("firstByteAtUnixSeconds", exchange(firstByteAt: .some(nil))),
            ("lastByteAtUnixSeconds", exchange(lastByteAt: 352)),
        ]
        for (field, variant) in variants {
            let digest = RuntimeBenchmark.transcriptDigest(
                of: Self.pair(candidateScenarios: [Self.scenario(exchanges: [variant])]).0)
            #expect(digest != sealed, "the seal does not cover exchange.\(field)")
        }
    }

    @Test("an absent measurement and a zero one do not seal alike")
    func sealDistinguishesAbsentFromZero() {
        // `-` for absent rather than an empty field, so a record that drops a
        // measurement cannot digest as one that measured zero. The two are
        // different facts everywhere else in this gate and they are different
        // here too.
        let base = Self.scenario(exchanges: [Self.exchange()])
        func with(ttft: Double?) -> RuntimeBenchmark.RunRecord {
            Self.pair(candidateScenarios: [
                RuntimeBenchmark.ScenarioResult(
                    name: base.name, succeeded: true, promptTokens: base.promptTokens,
                    completionTokens: base.completionTokens, timeToFirstTokenSeconds: ttft,
                    prefillTokensPerSecond: base.prefillTokensPerSecond,
                    decodeTokensPerSecond: base.decodeTokensPerSecond,
                    wallClockSeconds: base.wallClockSeconds,
                    peakPhysicalFootprintBytes: base.peakPhysicalFootprintBytes,
                    processPeakSoFarBytes: base.processPeakSoFarBytes,
                    transcript: base.transcript)
            ]).0
        }
        #expect(
            RuntimeBenchmark.transcriptDigest(of: with(ttft: nil))
                != RuntimeBenchmark.transcriptDigest(of: with(ttft: 0)))
    }
}
