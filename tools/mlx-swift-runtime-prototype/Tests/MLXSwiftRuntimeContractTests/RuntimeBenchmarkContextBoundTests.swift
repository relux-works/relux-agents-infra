import Foundation
import Testing

@testable import MLXSwiftRuntimeContract

/// The KV bound, for a third runtime that is never unbounded.
///
/// `llama-server` has no unbounded mode. Measured on the pinned Homebrew
/// `llama.cpp 0.3.0`, build `b10621-c1d0e7a00`, with a `Qwen2.5-0.5B-Instruct`
/// Q8_0 fixture: `GET /v1/models` reports `data[0].meta.n_ctx` as **8192**
/// under `--ctx-size 8192` and as **32768** — the model's `n_ctx_train` — with
/// no context flag at all.
///
/// The derivation used to read the absence of `--max-kv-size` as `unbounded`.
/// Applied to that runtime it produced a pin that was false, and because
/// ``RuntimeBenchmark/Pins/firstMismatch(against:)`` demands equality, a false
/// pin that *matched* a genuinely unbounded MLX baseline. The gate would have
/// stayed green over a comparison between a 32,768-token context window and no
/// window at all.
///
/// So these tests are mostly negative, and the acceptance question is
/// ``refusesTheFalseMatchTheOldDerivationWouldHaveGranted``: a llama.cpp record
/// must not be able to match an MLX baseline on a bound it does not satisfy.
/// The positive case is here too, because a gate that refuses everything is not
/// a gate either.
@Suite("runtime benchmark context bound")
struct RuntimeBenchmarkContextBoundTests {
    typealias Fixture = RuntimeBenchmarkTests

    /// A `llama-server` launch as `examples/model-harness.benchmark.toml`
    /// spells it: a finite context, the prompt-evaluation chunk in llama.cpp's
    /// own spelling, and the reasoning effort in the spelling it shares with
    /// the Swift runtime.
    static func llamaCPPArgv(
        contextSize: String? = "8192",
        microBatch: (flag: String, value: String)? = (
            "--ubatch-size", "2048"
        )
    ) -> [String] {
        var argv = ["--model", Fixture.modelPath, "--host", "127.0.0.1", "--port", "18031"]
        if let contextSize { argv += ["--ctx-size", contextSize] }
        if let microBatch { argv += [microBatch.flag, microBatch.value] }
        argv += ["--reasoning-effort", "medium", "--no-webui"]
        return argv
    }

    /// The deployed-default `mlx_lm.server` launch: no KV bound is requested,
    /// so `unbounded` is a true reading of this profile.
    static let unboundedMLXArgv = [
        "--model", Fixture.modelPath, "--host", "127.0.0.1", "--port", "18031",
        "--prefill-step-size", "2048",
        "--chat-template-args", #"{"reasoning_effort": "medium"}"#,
    ]

    /// The benchmark-only Python launch. Unlike the deployed default, this
    /// launch asks for the same finite bound as the llama.cpp candidate.
    static let boundedPythonArgv = unboundedMLXArgv + ["--max-kv-size", "76800"]

    /// A bounded Swift launch retained for cross-runtime derivation coverage.
    static let boundedSwiftArgv = [
        "serve",
        "--model", Fixture.modelPath, "--host", "127.0.0.1", "--port", "18031",
        "--reasoning-effort", "medium",
        "--prefill-step-size", "2048",
        "--max-kv-size", "8192",
    ]

    /// One record with its pin derived exactly as the driver derives it.
    ///
    /// `contextPolicy` is never typed in these tests. Writing the string by
    /// hand would make every case below a test of the string rather than of the
    /// derivation, and the pin the driver writes is the derivation's output by
    /// construction — `BenchmarkRunCommand.drive` builds `Pins` from this same
    /// call, after the runtime has answered.
    static func record(
        runtime: String, argv: [String], window: RuntimeContextWindow, startedAt: Double,
        finishedAt: Double, executableDigest: String
    ) -> RuntimeBenchmark.RunRecord {
        let pins = Fixture.variantPins(
            contextPolicy: RuntimeBenchmark.contextPolicy(derivedFrom: argv, observing: window))
        return Fixture.record(
            runtime: runtime, pins: pins, startedAt: startedAt, finishedAt: finishedAt,
            provenance: Fixture.provenance(
                launchArgv: argv, executableDigest: executableDigest))
    }

    /// Admission with each record observed under its own window.
    ///
    /// The window is the gate's reading, not the record's, so it is supplied
    /// per attestation and never derived from the record being judged.
    static func admit(
        baseline: RuntimeBenchmark.RunRecord, baselineWindow: RuntimeContextWindow,
        candidate: RuntimeBenchmark.RunRecord, candidateWindow: RuntimeContextWindow
    ) throws -> RuntimeBenchmark.Comparison {
        try RuntimeBenchmark.admit(
            baseline: baseline,
            baselineAttestation: Fixture.attestation(for: baseline, contextWindow: baselineWindow),
            candidate: candidate,
            candidateAttestation: Fixture.attestation(
                for: candidate, contextWindow: candidateWindow),
            requiredScenarios: [], gateBinaryDigest: Fixture.gateDigest)
    }

    // MARK: - G2: the bound comes from the process, not from a missing flag

    @Test("a runtime that reports its bound pins that number, however argv is spelled")
    func readsTheBoundOffTheRunningRuntime() {
        // The flag is absent and the runtime is bounded anyway. This is the
        // exact input the old derivation answered `kv=unbounded` to.
        #expect(
            RuntimeBenchmark.contextPolicy(
                derivedFrom: Self.llamaCPPArgv(contextSize: nil), observing: .reported(32768))
                == "kv=32768;prefill-step=2048;reasoning=medium")
        #expect(
            RuntimeBenchmark.contextPolicy(
                derivedFrom: Self.llamaCPPArgv(contextSize: "76800"),
                observing: .reported(76800))
                == "kv=76800;prefill-step=2048;reasoning=medium")
        #expect(
            RuntimeBenchmark.contextPolicy(
                derivedFrom: Self.boundedPythonArgv, observing: .reported(76800))
                == "kv=76800;prefill-step=2048;reasoning=medium")
    }

    @Test("no llama.cpp launch derives an unbounded KV pin, with or without --ctx-size")
    func neverDerivesUnboundedForABoundedRuntime() {
        for contextSize in [nil, "8192", "32768"] {
            for reported in [8192, 32768] {
                let derived = RuntimeBenchmark.contextPolicy(
                    derivedFrom: Self.llamaCPPArgv(contextSize: contextSize),
                    observing: .reported(reported))
                #expect(!derived.contains("kv=unbounded"))
                #expect(derived.hasPrefix("kv=\(reported);"))
            }
        }
    }

    @Test("a runtime that answered and named no bound still reads argv")
    func keepsTheArgvReadingForRuntimesThatReportNothing() {
        // The deployed-default Python launch and the Swift runtime's current
        // listing emit no `meta` block, so this fallback remains covered.
        #expect(
            RuntimeBenchmark.contextPolicy(
                derivedFrom: Self.unboundedMLXArgv, observing: .notReported)
                == "kv=unbounded;prefill-step=2048;reasoning=medium")
        #expect(
            RuntimeBenchmark.contextPolicy(
                derivedFrom: Self.boundedSwiftArgv, observing: .notReported)
                == "kv=8192;prefill-step=2048;reasoning=medium")
    }

    @Test("a bound the gate could not read is unpinned, not unbounded")
    func neverReadsAFailedReadAsAnAbsence() {
        let derived = RuntimeBenchmark.contextPolicy(
            derivedFrom: Self.unboundedMLXArgv, observing: .unread)
        #expect(derived == "kv=unread;prefill-step=2048;reasoning=medium")
        #expect(RuntimeBenchmark.unpinnableConditions.contains("kv=unread"))
    }

    // MARK: - The acceptance question

    @Test("a bounded candidate cannot match an unbounded baseline")
    func refusesTheFalseMatchTheOldDerivationWouldHaveGranted() {
        // The baseline is genuinely unbounded and says so by reporting nothing.
        // The candidate runs a 32,768-token window and says so. Under the old
        // derivation both sides produced the byte-identical string
        // `kv=unbounded;prefill-step=2048;reasoning=medium` and the pair was
        // admitted; the comparison then scored a 32k window against no window
        // and called the difference a runtime difference.
        let baseline = Self.record(
            runtime: "python-mlx-lm", argv: Self.unboundedMLXArgv, window: .notReported,
            startedAt: 100, finishedAt: 200, executableDigest: Fixture.digest("cd"))
        let candidate = Self.record(
            runtime: "llama-cpp", argv: Self.llamaCPPArgv(contextSize: nil),
            window: .reported(32768), startedAt: 300, finishedAt: 400,
            executableDigest: Fixture.digest("12"))
        #expect(
            throws: RuntimeBenchmark.AdmissionError.pinMismatch(
                field: "contextPolicy",
                baseline: "kv=unbounded;prefill-step=2048;reasoning=medium",
                candidate: "kv=32768;prefill-step=2048;reasoning=medium")
        ) {
            try Self.admit(
                baseline: baseline, baselineWindow: .notReported,
                candidate: candidate, candidateWindow: .reported(32768))
        }
    }

    @Test("a llama.cpp candidate is admitted against a baseline pinned to the same bound")
    func admitsALlamaCPPCandidateWithNoClauseRelaxed() throws {
        // The other half of the acceptance question. A gate that refused every
        // llama.cpp record would satisfy the negative above and be useless, so
        // the pair that genuinely shares every condition has to get through —
        // and it does so with `unpinnableConditions` untouched.
        let baseline = Self.record(
            runtime: "python-mlx-lm", argv: Self.boundedPythonArgv, window: .reported(76800),
            startedAt: 100, finishedAt: 200, executableDigest: Fixture.digest("cd"))
        let candidate = Self.record(
            runtime: "llama-cpp", argv: Self.llamaCPPArgv(contextSize: "76800"),
            window: .reported(76800),
            startedAt: 300, finishedAt: 400, executableDigest: Fixture.digest("12"))
        let comparison = try Self.admit(
            baseline: baseline, baselineWindow: .reported(76800),
            candidate: candidate, candidateWindow: .reported(76800))
        #expect(comparison.pins.contextPolicy == "kv=76800;prefill-step=2048;reasoning=medium")
    }

    @Test(
        "the llama.cpp candidate is refused against the Python incumbent, at any bound",
        arguments: [512, 4096, 8192, 32768, 76800, 262144])
    func refusesTheLlamaCPPCandidateAgainstThePythonIncumbent(bound: Int) {
        // The pair TASK-260828-2wcrph was sent to measure, and the reason its
        // report carries no score. `mlx_lm-relux.server` has no KV bound flag,
        // so its launch can only derive `kv=unbounded`; `llama-server` has no
        // unbounded mode, so its launch can only derive `kv=<n>`. There is no
        // argv on either side that makes the two agree, which is why the
        // argument list sweeps the bound rather than naming one: this is a
        // refusal of the whole class, not of one badly chosen `--ctx-size`.
        //
        // It is also the correct refusal rather than a configuration mistake:
        // `llama-server` allocates its whole KV arena at load, so the bound is
        // resident before the first token, while `mlx_lm.server` grows its
        // cache per request. Measured on the 27B Q8_0 GGUF, the arena is
        // 65,536 bytes per token -- 5.03 GB at 76,800 -- and it is paid on the
        // 15-token prompt too.
        let baseline = Self.record(
            runtime: "python-mlx-lm", argv: Self.unboundedMLXArgv, window: .notReported,
            startedAt: 100, finishedAt: 200, executableDigest: Fixture.digest("cd"))
        let candidate = Self.record(
            runtime: "llama-cpp", argv: Self.llamaCPPArgv(contextSize: String(bound)),
            window: .reported(bound), startedAt: 300, finishedAt: 400,
            executableDigest: Fixture.digest("12"))
        #expect(
            throws: RuntimeBenchmark.AdmissionError.pinMismatch(
                field: "contextPolicy",
                baseline: "kv=unbounded;prefill-step=2048;reasoning=medium",
                candidate: "kv=\(bound);prefill-step=2048;reasoning=medium")
        ) {
            try Self.admit(
                baseline: baseline, baselineWindow: .notReported,
                candidate: candidate, candidateWindow: .reported(bound))
        }
    }

    @Test("the shipped benchmark pair pins 76800 without changing the deployed default")
    func theShippedConfigBoundsOnlyTheBenchmarkPair() throws {
        // Reads the versioned file, the way `trustedDecisionMatchesItsDocument`
        // reads the equivalence document, so the comment beside the profile is
        // an artifact the build checks rather than prose next to the code.
        //
        let configPath = URL(fileURLWithPath: #filePath)
            .deletingLastPathComponent()
            .deletingLastPathComponent()
            .deletingLastPathComponent()
            .appendingPathComponent("examples/model-harness.benchmark.toml")
        let text = try String(contentsOf: configPath, encoding: .utf8)

        func profileBlock(_ name: String) throws -> String {
            let header = "[profiles.\(name)]"
            let start = try #require(text.range(of: header))
            let rest = text[start.upperBound...]
            guard let next = rest.range(of: "\n[profiles.") else { return String(rest) }
            return String(rest[..<next.lowerBound])
        }

        let python = try profileBlock("qwen-benchmark-python")
        #expect(
            python.contains(
                #"executable = "/Users/alexis/.local/bin/mlx_lm-kv76800-45a472f.server""#))
        #expect(
            !python.contains(
                #"executable = "/Users/alexis/.local/bin/mlx_lm-relux.server""#))
        #expect(python.contains(#""--max-kv-size", "76800""#))
        #expect(!python.contains("--ctx-size"))
        #expect(!python.contains("\"-c\""))
        // The conditions it does state, so this is not passing because the
        // block was sliced empty.
        #expect(python.contains("--prefill-step-size"))
        #expect(python.contains("--chat-template-args"))

        let llamacpp = try profileBlock("qwen-benchmark-llamacpp")
        #expect(llamacpp.contains(#""--ctx-size", "76800""#))
        #expect(llamacpp.contains("--ubatch-size"))

        // The deployed profile is outside this benchmark-only file.
        #expect(!text.contains("[profiles.qwen-local]"))
    }

    @Test("a record whose bound the gate never read is refused by name")
    func refusesAnUnreadBound() {
        let baseline = Self.record(
            runtime: "python-mlx-lm", argv: Self.unboundedMLXArgv, window: .unread,
            startedAt: 100, finishedAt: 200, executableDigest: Fixture.digest("cd"))
        let candidate = Self.record(
            runtime: "llama-cpp", argv: Self.llamaCPPArgv(), window: .unread,
            startedAt: 300, finishedAt: 400, executableDigest: Fixture.digest("12"))
        #expect(
            throws: RuntimeBenchmark.AdmissionError.unpinnedLaunchCondition(
                runtime: "python-mlx-lm", condition: "kv=unread")
        ) {
            try Self.admit(
                baseline: baseline, baselineWindow: .unread,
                candidate: candidate, candidateWindow: .unread)
        }
    }

    // MARK: - The launch and the process have to agree about the bound

    @Test("a --ctx-size the process did not honour is refused")
    func refusesAContextSizeTheProcessDidNotHonour() {
        let baseline = Self.record(
            runtime: "mlx-swift", argv: Self.boundedSwiftArgv, window: .notReported,
            startedAt: 100, finishedAt: 200, executableDigest: Fixture.digest("cd"))
        // Asked for 8192, running 4096. The pin takes the process's number, so
        // the pins still agree and nothing above this clause can see it.
        let candidate = Self.record(
            runtime: "llama-cpp", argv: Self.llamaCPPArgv(contextSize: "4096"),
            window: .reported(8192), startedAt: 300, finishedAt: 400,
            executableDigest: Fixture.digest("12"))
        #expect(
            throws: RuntimeBenchmark.AdmissionError.contextBoundNotHonoured(
                runtime: "llama-cpp", flag: "--ctx-size", declared: 4096, reported: "8192")
        ) {
            try Self.admit(
                baseline: baseline, baselineWindow: .reported(8192),
                candidate: candidate, candidateWindow: .reported(8192))
        }
    }

    @Test("a --max-kv-size the Python process did not honour is refused")
    func refusesAMaxKVSizeTheProcessDidNotHonour() {
        let baseline = Self.record(
            runtime: "python-mlx-lm", argv: Self.boundedPythonArgv,
            window: .reported(4096), startedAt: 100, finishedAt: 200,
            executableDigest: Fixture.digest("cd"))
        let candidate = Self.record(
            runtime: "llama-cpp", argv: Self.llamaCPPArgv(contextSize: "4096"),
            window: .reported(4096), startedAt: 300, finishedAt: 400,
            executableDigest: Fixture.digest("12"))
        #expect(
            throws: RuntimeBenchmark.AdmissionError.contextBoundNotHonoured(
                runtime: "python-mlx-lm", flag: "--max-kv-size", declared: 76800,
                reported: "4096")
        ) {
            try Self.admit(
                baseline: baseline, baselineWindow: .reported(4096),
                candidate: candidate, candidateWindow: .reported(4096))
        }
    }

    @Test("a --ctx-size launch whose runtime reports nothing cannot reach the argv fallback")
    func refusesASilentRuntimeThatPinnedAContextSize() {
        // The residual the `notReported` fallback would otherwise leave open: a
        // bounded runtime that answered `/v1/models` and declined to say so
        // would be read as unbounded. A launch carrying llama.cpp's own context
        // flag and a server that will not confirm the bound is a contradiction,
        // and is refused rather than defaulted.
        let baseline = Self.record(
            runtime: "mlx-swift", argv: Self.boundedSwiftArgv, window: .reported(8192),
            startedAt: 100, finishedAt: 200, executableDigest: Fixture.digest("cd"))
        let candidate = Fixture.record(
            runtime: "llama-cpp", pins: baseline.pins, startedAt: 300, finishedAt: 400,
            provenance: Fixture.provenance(
                launchArgv: Self.llamaCPPArgv(), executableDigest: Fixture.digest("12")))
        #expect(
            throws: RuntimeBenchmark.AdmissionError.contextBoundNotHonoured(
                runtime: "llama-cpp", flag: "--ctx-size", declared: 8192,
                reported: "no bound at all")
        ) {
            try Self.admit(
                baseline: baseline, baselineWindow: .reported(8192),
                candidate: candidate, candidateWindow: .notReported)
        }
    }

    @Test("a --max-kv-size launch whose runtime reports nothing is its normal case")
    func acceptsASilentRuntimeThatPinnedMaxKVSize() throws {
        // The asymmetry is deliberate and measured: `--max-kv-size` belongs to
        // runtimes that report no bound, so silence there is their answer, not
        // a contradiction. Collapsing the two arms would refuse the incumbent.
        // Both sides are Swift-shaped launches on purpose: `--max-kv-size` is
        // the flag of a runtime that reports no bound, and this test is about
        // that arm, so a pair in which only one side can spell it would never
        // reach the clause. `admit` refuses two records naming one runtime, so
        // the two builds are named apart.
        let baseline = Self.record(
            runtime: "mlx-swift", argv: Self.boundedSwiftArgv, window: .notReported,
            startedAt: 100, finishedAt: 200, executableDigest: Fixture.digest("cd"))
        let candidate = Self.record(
            runtime: "mlx-swift-next", argv: Self.boundedSwiftArgv, window: .notReported,
            startedAt: 300, finishedAt: 400, executableDigest: Fixture.digest("12"))
        let comparison = try Self.admit(
            baseline: baseline, baselineWindow: .reported(8192),
            candidate: candidate, candidateWindow: .reported(8192))
        #expect(comparison.pins.contextPolicy.hasPrefix("kv=8192;"))
    }

    // MARK: - G1: a third spelling of the prefill pin, additively

    @Test(
        "the llama.cpp spelling of the prefill chunk derives the same value",
        arguments: ["--ubatch-size", "-ub"])
    func readsTheMicroBatchSpelling(flag: String) {
        #expect(
            RuntimeBenchmark.contextPolicy(
                derivedFrom: Self.llamaCPPArgv(microBatch: (flag, "2048")),
                observing: .reported(8192))
                == "kv=8192;prefill-step=2048;reasoning=medium")
        // `=`-joined, as a launcher config may render it.
        #expect(
            RuntimeBenchmark.contextPolicy(
                derivedFrom: ["--model", Fixture.modelPath, "\(flag)=512", "--ctx-size", "8192"]
                    + ["--reasoning-effort", "medium"], observing: .reported(8192))
                == "kv=8192;prefill-step=512;reasoning=medium")
    }

    @Test("--batch-size is not the prefill chunk and does not pin it")
    func doesNotReadTheLogicalBatchAsThePrefillChunk() {
        // `llama-server` takes both. `--batch-size` is the logical batch and
        // defaults to 2048; the physical prompt-evaluation chunk is
        // `--ubatch-size` and defaults to 512. Reading the first as the second
        // would pin a condition the launch never stated, at a value four times
        // the one in effect.
        let derived = RuntimeBenchmark.contextPolicy(
            derivedFrom: ["--model", Fixture.modelPath, "--batch-size", "2048"]
                + ["--ctx-size", "8192", "--reasoning-effort", "medium"],
            observing: .reported(8192))
        #expect(derived == "kv=8192;prefill-step=unpinned;reasoning=medium")
    }

    @Test("a llama.cpp launch that left the prefill chunk to its default is still refused")
    func refusesAnUnpinnedMicroBatch() {
        // The narrowing direction. `unpinnableConditions` was not relaxed to
        // admit llama.cpp — the derivation was widened — so a llama.cpp launch
        // that states no chunk is refused exactly as an MLX one is, at
        // llama.cpp's own default of 512 against `mlx_lm.server`'s 2048.
        let baseline = Self.record(
            runtime: "python-mlx-lm", argv: Self.unboundedMLXArgv, window: .notReported,
            startedAt: 100, finishedAt: 200, executableDigest: Fixture.digest("cd"))
        let candidate = Self.record(
            runtime: "llama-cpp", argv: Self.llamaCPPArgv(microBatch: nil),
            window: .reported(8192), startedAt: 300, finishedAt: 400,
            executableDigest: Fixture.digest("12"))
        #expect(
            throws: RuntimeBenchmark.AdmissionError.pinMismatch(
                field: "contextPolicy",
                baseline: "kv=unbounded;prefill-step=2048;reasoning=medium",
                candidate: "kv=8192;prefill-step=unpinned;reasoning=medium")
        ) {
            try Self.admit(
                baseline: baseline, baselineWindow: .notReported,
                candidate: candidate, candidateWindow: .reported(8192))
        }
        // And with the pins forced into agreement, so the mismatch above cannot
        // be what refuses it, the unpinnable clause is what fires.
        let unreportedGeneration = RuntimeGenerationConfiguration.notReported
        let matchedPins = Fixture.variantPins(
            contextPolicy: RuntimeBenchmark.contextPolicy(
                observing: .reported(8192),
                generationConfiguration: unreportedGeneration))
        let matched = Fixture.record(
            runtime: "python-mlx-lm", pins: matchedPins,
            startedAt: 100, finishedAt: 200,
            provenance: Fixture.provenance(
                launchArgv: Self.llamaCPPArgv(microBatch: nil),
                executableDigest: Fixture.digest("cd")))
        let matchedCandidate = Fixture.record(
            runtime: "llama-cpp", pins: matchedPins,
            startedAt: 300, finishedAt: 400,
            provenance: Fixture.provenance(
                launchArgv: Self.llamaCPPArgv(microBatch: nil),
                executableDigest: Fixture.digest("12")))
        #expect(
            throws: RuntimeBenchmark.AdmissionError.unpinnedLaunchCondition(
                runtime: "python-mlx-lm", condition: "prefill-step=not-reported")
        ) {
            try RuntimeBenchmark.admit(
                baseline: matched,
                baselineAttestation: Fixture.attestation(
                    for: matched, contextWindow: .reported(8192),
                    generationConfiguration: unreportedGeneration),
                candidate: matchedCandidate,
                candidateAttestation: Fixture.attestation(
                    for: matchedCandidate, contextWindow: .reported(8192),
                    generationConfiguration: unreportedGeneration),
                requiredScenarios: [], gateBinaryDigest: Fixture.gateDigest)
        }
    }

    @Test("the unpinnable clause list only ever grew")
    func doesNotRelaxTheUnpinnableConditions() {
        // The measured cost of relaxing it is on TASK-260828-2jbufw: dropping
        // `prefill-step=unpinned` admits an unpinned mlx-swift launch (512
        // tokens) against an unpinned `mlx_lm.server` one (2048), because all
        // three runtimes derive the byte-identical string. This asserts the
        // list as a whole so a future edit that removes a clause has to remove
        // it here too, in the open.
        #expect(
            RuntimeBenchmark.unpinnableConditions == [
                "kv=not-reported", "kv=unread",
                "prefill-step=not-reported", "prefill-step=unread",
                "reasoning=not-reported", "reasoning=unread",
            ])
    }

    // MARK: - The window is a document field, and absence is not a reading

    @Test("an attestation that omits the context window is malformed, not silent")
    func refusesToDecodeAnAttestationWithoutAWindow() throws {
        let attestation = Fixture.attestation(
            for: Fixture.candidate, contextWindow: .reported(8192))
        let encoded = try JSONEncoder().encode(attestation)
        var document =
            try JSONSerialization.jsonObject(with: encoded) as? [String: Any] ?? [:]
        #expect(document["observedContextWindow"] != nil)
        document["observedContextWindow"] = nil
        let stripped = try JSONSerialization.data(withJSONObject: document)
        #expect(throws: (any Error).self) {
            try JSONDecoder().decode(RuntimeAttestation.self, from: stripped)
        }
    }

    @Test(
        "every window state round-trips by name",
        arguments: [
            RuntimeContextWindow.reported(8192), .reported(32768), .notReported, .unread,
        ])
    func roundTripsEveryState(window: RuntimeContextWindow) throws {
        let encoded = try JSONEncoder().encode(window)
        #expect(try JSONDecoder().decode(RuntimeContextWindow.self, from: encoded) == window)
    }

    @Test("an unknown window state is refused rather than read as one of the known ones")
    func refusesAnUnknownWindowState() {
        let document = Data(#"{"state": "assumed", "length": 8192}"#.utf8)
        #expect(throws: (any Error).self) {
            try JSONDecoder().decode(RuntimeContextWindow.self, from: document)
        }
    }
}
