import CryptoKit
import Foundation

/// The comparison layer for the Python-vs-Swift runtime migration decision.
///
/// A benchmark number only means something next to the conditions it was taken
/// under. This type exists because the interesting failure of a migration
/// benchmark is not a slow runtime — it is a fast-looking one, measured against
/// a baseline that was quietly given a different model, a different prompt, a
/// different output bound, or a host that was busy holding the other runtime's
/// weights at the time. None of those show up in the numbers themselves.
///
/// So the numbers are never compared directly. Two ``RuntimeBenchmark/RunRecord``
/// documents are admitted for comparison only if every pinned condition in them
/// is identical, they name different runtimes, and their wall-clock intervals do
/// not overlap. Anything else is refused with the exact field that differs, and
/// no report is produced at all.
///
/// Production call site: `BenchmarkCompareCommand.run(arguments:)` in the
/// `mlx-swift-runtime-prototype` executable target, reached from `Main.main()`
/// for the `benchmark-compare` subcommand.
public enum RuntimeBenchmark {
    /// Every condition that must be identical for two runs to be comparable.
    ///
    /// All properties are non-optional and have no defaults, so a record that
    /// omits one fails to decode rather than comparing under an invented value.
    /// That distinction matters: a `0` default for ``maxOutputTokens`` would
    /// make two records with *no* output bound agree with each other.
    public struct Pins: Sendable, Equatable, Codable {
        /// Hardware model, physical memory and OS build, joined. Two runs on
        /// different hosts are not a comparison, they are two measurements.
        public let hostIdentity: String
        /// The model both runs are about, identified by the thing they share
        /// rather than by the file each of them opened.
        ///
        /// This is the G4 pin. `modelPath` and `modelDigest` below are still
        /// recorded and still checked, but they are no longer what two runs
        /// must agree on, because for a cross-format comparison they can never
        /// agree: MLX 8-bit group64 and GGUF Q8_0 are two files derived from
        /// one upstream model, and demanding byte identity of them refuses
        /// forever on a premise that does not fit the question.
        ///
        /// Not free text and not the driver's opinion. It is
        /// ``RuntimeBenchmark/modelOfRecord(artifactDigest:observing:)`` applied
        /// to the equivalence reading the *gate* wrote onto
        /// ``RuntimeAttestation/observedModelEquivalence``, and
        /// ``RuntimeBenchmark/admitProvenance(_:observing:)`` re-derives it and
        /// refuses the record when the two disagree. Two forms, and only two:
        ///
        /// * `artifact:<modelDigest>` when no equivalence verdict was named.
        ///   Nothing is relaxed by this: the pin *is* the artifact digest, so
        ///   the same-format pair still has to be byte-identical to compare
        ///   equal, exactly as it did when `modelDigest` was the pin.
        /// * `source:<sourceOfRecord>` when a verdict was read. The pair then
        ///   additionally has to satisfy
        ///   ``RuntimeBenchmark/admitModelIdentity(baseline:baselineAttestation:candidate:candidateAttestation:)``,
        ///   which is strictly more than the equality it replaces.
        public let modelOfRecord: String
        /// Absolute path to the weight artifact *this* runtime was pointed at:
        /// an MLX weight directory, or a `.gguf` file.
        ///
        /// Per-run rather than shared since G4. It is checked against the
        /// launch argv in ``RuntimeBenchmark/admitProvenance(_:observing:)`` and
        /// against the other run in
        /// ``RuntimeBenchmark/admitModelIdentity(baseline:baselineAttestation:candidate:candidateAttestation:)``;
        /// it is not compared for equality in ``firstMismatch(against:)``,
        /// because ``modelOfRecord`` is what carries that job now.
        public let modelPath: String
        /// Digest over the artifact at ``modelPath``: `config.json` plus the
        /// safetensors index for a weight directory, the whole file for a
        /// `.gguf`. A re-quantized or re-sharded artifact at the same path is a
        /// different digest and therefore a different ``modelOfRecord``.
        public let modelDigest: String
        /// Quantization as the artifact declares it, for example
        /// `8bit/group64/affine`, or as the equivalence verdict names it for
        /// this artifact's digest, for example `Q8_0`.
        public let quantization: String
        /// Digest over the pinned prompt suite file. The prompts themselves are
        /// not carried here; what is pinned is that both runs read the same
        /// bytes.
        public let promptSuiteDigest: String
        /// The prompt-evaluation policy both runtimes were configured with,
        /// spelled the same way for both.
        ///
        /// Not free text, and not the driver's opinion: it is
        /// ``RuntimeBenchmark/contextPolicy(derivedFrom:)`` applied to the
        /// *rendered launch argv* recorded in ``RunRecord/provenance``, and
        /// ``RuntimeBenchmark/admit(baseline:baselineAttestation:candidate:candidateAttestation:requiredScenarios:gateBinaryDigest:)``
        /// re-derives it and refuses the pair when the two disagree. A record
        /// can therefore no longer declare a policy its launch did not carry.
        ///
        /// Review found the previous shape: `runtime-benchmark.py` assigned a
        /// module-level constant, so two records minted by hand with no launch
        /// behind them agreed on this pin and were admitted.
        public let contextPolicy: String
        /// Whether this runtime was speculating, in a spelling both runtimes
        /// share.
        ///
        /// Derived, like ``contextPolicy``, by
        /// ``RuntimeBenchmark/speculationPolicy(derivedFrom:observing:)`` from
        /// the launch argv and from what the *running* process answered, and
        /// re-derived at admission. Anything but `off` is refused outright by
        /// ``RuntimeBenchmark/AdmissionError/speculativeDecodingActive(runtime:reading:)``
        /// rather than merely required to match, because speculation is not a
        /// condition two runs can share: `llama-server` can speculate off an
        /// MTP head and the MLX baseline has no MTP head at all, so a tokens/s
        /// measured with it on is a different decoding algorithm rather than a
        /// faster runtime. TASK-260828-3g87i4's equivalence verdict names this
        /// as the one way the Qwen3.8-27B pair genuinely stops being
        /// comparable.
        public let speculation: String
        /// Output bound applied to every scored scenario.
        ///
        /// Scenarios that need a different bound — a 75k-context capacity probe
        /// has no reason to also generate 256 tokens — carry their own in the
        /// prompt suite, which is pinned by ``promptSuiteDigest`` above. So both
        /// halves are pinned: the scored bound by value here, the per-scenario
        /// overrides by digest.
        public let maxOutputTokens: Int
        /// Sampler settings, sent explicitly on every request.
        ///
        /// Pinned rather than defaulted because the two runtimes do not agree
        /// on defaults: `MLXLMCommon.GenerateParameters` starts at
        /// `temperature = 0.6`, `mlx_lm.server` at `1.0`. A benchmark that
        /// omitted them would be comparing two samplers.
        public let temperature: Double
        public let topP: Double
        public let seed: Int

        public init(
            hostIdentity: String,
            modelOfRecord: String,
            modelPath: String,
            modelDigest: String,
            quantization: String,
            promptSuiteDigest: String,
            contextPolicy: String,
            speculation: String,
            maxOutputTokens: Int,
            temperature: Double,
            topP: Double,
            seed: Int
        ) {
            self.hostIdentity = hostIdentity
            self.modelOfRecord = modelOfRecord
            self.modelPath = modelPath
            self.modelDigest = modelDigest
            self.quantization = quantization
            self.promptSuiteDigest = promptSuiteDigest
            self.contextPolicy = contextPolicy
            self.speculation = speculation
            self.maxOutputTokens = maxOutputTokens
            self.temperature = temperature
            self.topP = topP
            self.seed = seed
        }

        /// Field-by-field comparison, in a fixed order, reporting the first
        /// disagreement by name.
        ///
        /// Written out rather than derived from `Equatable` so the refusal can
        /// say *which* pin differs. "These runs are not comparable" sends a
        /// reader back to the raw records; "modelPath differs" does not.
        ///
        /// ``modelPath``, ``modelDigest`` and ``quantization`` are deliberately
        /// absent from this list since G4, and their absence relaxes nothing.
        /// For a pair with no equivalence evidence ``modelOfRecord`` *is*
        /// `artifact:<modelDigest>`, so digest equality is still demanded here
        /// by an equality pin, and the remaining two are demanded by
        /// ``RuntimeBenchmark/admitModelIdentity(baseline:baselineAttestation:candidate:candidateAttestation:)``
        /// with the same `pinMismatch` refusal. What changed is only that a
        /// cross-format pair now has somewhere to go other than a refusal it
        /// could never have satisfied — and where it goes is strictly harder.
        func firstMismatch(against other: Pins) -> (field: String, mine: String, theirs: String)? {
            let fields: [(String, String, String)] = [
                ("hostIdentity", hostIdentity, other.hostIdentity),
                ("modelOfRecord", modelOfRecord, other.modelOfRecord),
                ("promptSuiteDigest", promptSuiteDigest, other.promptSuiteDigest),
                ("contextPolicy", contextPolicy, other.contextPolicy),
                ("speculation", speculation, other.speculation),
                ("maxOutputTokens", String(maxOutputTokens), String(other.maxOutputTokens)),
                ("temperature", String(temperature), String(other.temperature)),
                ("topP", String(topP), String(other.topP)),
                ("seed", String(seed), String(other.seed)),
            ]
            for (name, mine, theirs) in fields where mine != theirs {
                return (name, mine, theirs)
            }
            return nil
        }
    }

    /// The HTTP exchanges one scenario actually performed, recorded by the
    /// process that performed them.
    ///
    /// This type exists because of the third consecutive review finding on the
    /// same class, and the third one was the sharpest: the gate would attest
    /// two placeholder HTTP servers that answered `GET /v1/models` and nothing
    /// else, and then admit a caller-authored set of measurements taken against
    /// them. Everything the previous revision checked was true of that pass —
    /// two live processes, two kernel-observed pids, two served model IDs, one
    /// judging binary. What none of it established is that a benchmark had
    /// happened. `/v1/models` plus an elapsed window is not a transcript.
    ///
    /// So a measurement now travels with the exchanges it came from, and the
    /// observing process seals them: ``RuntimeAttestation/transcriptDigest`` is
    /// computed by the same invocation that drove these requests, over
    /// ``RuntimeBenchmark/transcriptDigest(of:)`` of the record it built.
    /// A record whose scenarios were edited — or authored — no longer digests
    /// to what the observer sealed, and a pass in which no scenario ever
    /// completed a chat completion is refused outright rather than scored as a
    /// runtime that lost.
    public struct ScenarioTranscript: Sendable, Equatable, Codable {
        /// One request/response pair, as the driver performed it.
        public struct Exchange: Sendable, Equatable, Codable {
            public let method: String
            /// Request path, so a `/v1/models` probe cannot pass for a served
            /// completion.
            public let path: String
            /// SHA-256 over the exact request body bytes that went on the wire.
            public let requestDigest: String
            public let requestByteCount: Int
            /// HTTP status, or `0` when the request never got one — a refused
            /// connection and a rejected request are different facts.
            public let status: Int
            /// SHA-256 over the exact response bytes that came back.
            public let responseDigest: String
            public let responseByteCount: Int
            public let sentAtUnixSeconds: Double
            /// `nil` when nothing ever came back.
            public let firstByteAtUnixSeconds: Double?
            public let lastByteAtUnixSeconds: Double

            public init(
                method: String,
                path: String,
                requestDigest: String,
                requestByteCount: Int,
                status: Int,
                responseDigest: String,
                responseByteCount: Int,
                sentAtUnixSeconds: Double,
                firstByteAtUnixSeconds: Double?,
                lastByteAtUnixSeconds: Double
            ) {
                self.method = method
                self.path = path
                self.requestDigest = requestDigest
                self.requestByteCount = requestByteCount
                self.status = status
                self.responseDigest = responseDigest
                self.responseByteCount = responseByteCount
                self.sentAtUnixSeconds = sentAtUnixSeconds
                self.firstByteAtUnixSeconds = firstByteAtUnixSeconds
                self.lastByteAtUnixSeconds = lastByteAtUnixSeconds
            }
        }

        public let exchanges: [Exchange]

        public init(exchanges: [Exchange]) {
            self.exchanges = exchanges
        }

        /// The path every scored measurement in this harness comes from.
        ///
        /// Named here rather than at each call site so "did this scenario
        /// actually serve a completion" has one answer, and so a probe of some
        /// other endpoint can never satisfy it.
        public static let completionPath = "/v1/chat/completions"

        /// `true` when at least one exchange was a completion that returned
        /// 200 with bytes in it.
        ///
        /// A 200 with an empty body is not a served completion: that is what a
        /// placeholder returns.
        public var carriesServedCompletion: Bool {
            exchanges.contains {
                $0.path == Self.completionPath && $0.status == 200 && $0.responseByteCount > 0
            }
        }

        /// Every timestamp in this transcript, for the window check.
        var instants: [Double] {
            exchanges.flatMap { exchange -> [Double] in
                [exchange.sentAtUnixSeconds, exchange.lastByteAtUnixSeconds]
                    + (exchange.firstByteAtUnixSeconds.map { [$0] } ?? [])
            }
        }
    }

    /// One measured scenario, as the driver observed it.
    ///
    /// Every measurement is optional and every optional means *not measured*.
    /// A scenario that failed reports `succeeded == false` and a
    /// ``failureMode``; it does not report zeros. Zero tokens per second and
    /// "we never found out" are different facts and the decision treats them
    /// differently.
    public enum CacheReuseState: String, Sendable, Equatable, Codable {
        case hit
        case miss
        case unknown
        case notApplicable = "not-applicable"
    }

    /// Runtime-reported cache reuse for one scenario. The observation is part
    /// of the sealed run record; timing never stands in for this fact.
    public struct CacheReuseObservation: Sendable, Equatable, Codable {
        public static let usageSource =
            "openai-usage.prompt_tokens_details.cached_tokens"

        public let state: CacheReuseState
        public let source: String
        public let cachedPromptTokens: [Int]?
        public let issue: String?

        public init(
            state: CacheReuseState,
            source: String,
            cachedPromptTokens: [Int]? = nil,
            issue: String? = nil
        ) {
            self.state = state
            self.source = source
            self.cachedPromptTokens = cachedPromptTokens
            self.issue = issue
        }

        public static let notApplicable = CacheReuseObservation(
            state: .notApplicable, source: "scenario-not-reuse-sensitive")

        public static func reported(cachedPromptTokens: [Int]) -> CacheReuseObservation {
            CacheReuseObservation(
                state: cachedPromptTokens.contains(where: { $0 > 0 }) ? .hit : .miss,
                source: usageSource, cachedPromptTokens: cachedPromptTokens)
        }

        public static func unknown(_ issue: String) -> CacheReuseObservation {
            CacheReuseObservation(state: .unknown, source: usageSource, issue: issue)
        }

        /// Reject a decoded observation whose state disagrees with its facts.
        public var validatedState: CacheReuseState? {
            switch state {
            case .hit:
                guard source == Self.usageSource, issue == nil,
                    let cachedPromptTokens, !cachedPromptTokens.isEmpty,
                    cachedPromptTokens.allSatisfy({ $0 >= 0 }),
                    cachedPromptTokens.contains(where: { $0 > 0 })
                else { return nil }
            case .miss:
                guard source == Self.usageSource, issue == nil,
                    let cachedPromptTokens, !cachedPromptTokens.isEmpty,
                    cachedPromptTokens.allSatisfy({ $0 == 0 })
                else { return nil }
            case .unknown:
                guard source == Self.usageSource, cachedPromptTokens == nil,
                    issue?.isEmpty == false
                else { return nil }
            case .notApplicable:
                guard source == "scenario-not-reuse-sensitive", cachedPromptTokens == nil,
                    issue == nil
                else { return nil }
            }
            return state
        }
    }

    public struct ScenarioResult: Sendable, Equatable, Codable {
        public let name: String
        public let succeeded: Bool
        public let failureMode: String?
        public let promptTokens: Int?
        public let completionTokens: Int?
        public let timeToFirstTokenSeconds: Double?
        public let prefillTokensPerSecond: Double?
        public let decodeTokensPerSecond: Double?
        public let wallClockSeconds: Double?
        /// Raw Mach component from the sample that produced this scenario's
        /// resident-memory upper-bound peak. Never scored by itself.
        ///
        /// Scenario-local rather than peak-so-far, and the distinction decided
        /// a verdict. The sampler's running maximum never falls, so once a
        /// 75k-context probe has pushed a process to 49 GiB every later
        /// scenario reports 49 GiB whatever it actually cost — and a candidate
        /// that *aborted* before the expensive scenario reports a lower
        /// whole-pass maximum than the baseline that completed it. Review
        /// caught exactly that: 1.094x on whole-pass maxima from different
        /// completed work, against 1.399x on the one 8k scenario both runtimes
        /// finished.
        public let peakPhysicalFootprintBytes: Int?
        /// The process-wide running peak at the moment this scenario ended.
        ///
        /// Kept beside the scenario-local figure rather than instead of it, so
        /// the growth of the whole process is still readable. Never scored: it
        /// is a different quantity and mixing the two is the defect above.
        public let processPeakSoFarBytes: Int?
        /// The scored scenario-local quantity, including resident mapped files.
        /// Its status and raw components make absence, failed reads, malformed
        /// reads and partial coverage distinguishable and fail-closed.
        public let peakResidentMemory: RuntimeMemoryPeak
        /// Process-wide running peak at this scenario boundary, under the same
        /// accounting and with the same evidence shape.
        public let processResidentMemoryPeakSoFar: RuntimeMemoryPeak
        /// The highest 1-minute host load average seen inside this scenario's
        /// window.
        ///
        /// Recorded, never scored, and it exists because the first revision-4
        /// pass was ruined by something this record could not show. A concurrent
        /// agent session on the same machine spent the whole baseline pass
        /// loading models, and `mlx_lm.server` came out **2.7x slower on
        /// `short_prompt` and 3.2x slower at 8k** than the same runtime,
        /// same config, same prompts one revision earlier — while the candidate
        /// pass, which ran after that session went quiet, reproduced its own
        /// revision-3 numbers to within 1%. The comparison looked like a large
        /// candidate win. It was a measurement of somebody else's workload.
        ///
        /// Nothing in the record showed it. The contamination was caught by
        /// comparing absolute values against the previous revision, which is
        /// not a check a gate can make and not one a reader should have to.
        /// So the figure is in the record now, sealed with everything else.
        ///
        /// It is **not** scored, and that is a deliberate boundary rather than
        /// an oversight: what load ratio makes two passes non-comparable is a
        /// policy decision about this host, and a bar invented here would be
        /// the gate asserting something it cannot support.
        public let hostLoadAverageMax: Double?
        /// Cache reuse is independent evidence, never inferred from TTFT.
        public let cacheReuse: CacheReuseObservation
        /// The HTTP exchanges the measurements above were taken from.
        ///
        /// Optional in the type and required by admission: a record that
        /// decodes without one is a record from an older driver, and
        /// ``RuntimeBenchmark/admit(baseline:baselineAttestation:candidate:candidateAttestation:requiredScenarios:gateBinaryDigest:)``
        /// refuses it by name rather than letting it decode into a scenario
        /// with nothing behind it. Review's placeholder pass is the shape this
        /// exists for: two processes that answered `/v1/models`, and a set of
        /// numbers that came from nowhere.
        public let transcript: ScenarioTranscript?

        public init(
            name: String,
            succeeded: Bool,
            failureMode: String? = nil,
            promptTokens: Int? = nil,
            completionTokens: Int? = nil,
            timeToFirstTokenSeconds: Double? = nil,
            prefillTokensPerSecond: Double? = nil,
            decodeTokensPerSecond: Double? = nil,
            wallClockSeconds: Double? = nil,
            peakPhysicalFootprintBytes: Int? = nil,
            processPeakSoFarBytes: Int? = nil,
            peakResidentMemory: RuntimeMemoryPeak = .absent,
            processResidentMemoryPeakSoFar: RuntimeMemoryPeak = .absent,
            hostLoadAverageMax: Double? = nil,
            cacheReuse: CacheReuseObservation = .notApplicable,
            transcript: ScenarioTranscript? = nil
        ) {
            self.name = name
            self.succeeded = succeeded
            self.failureMode = failureMode
            self.promptTokens = promptTokens
            self.completionTokens = completionTokens
            self.timeToFirstTokenSeconds = timeToFirstTokenSeconds
            self.prefillTokensPerSecond = prefillTokensPerSecond
            self.decodeTokensPerSecond = decodeTokensPerSecond
            self.wallClockSeconds = wallClockSeconds
            self.peakPhysicalFootprintBytes = peakPhysicalFootprintBytes
            self.processPeakSoFarBytes = processPeakSoFarBytes
            self.peakResidentMemory = peakResidentMemory
            self.processResidentMemoryPeakSoFar = processResidentMemoryPeakSoFar
            self.hostLoadAverageMax = hostLoadAverageMax
            self.cacheReuse = cacheReuse
            self.transcript = transcript
        }
    }

    /// What actually ran, recorded from the launch rather than asserted beside
    /// it.
    ///
    /// The gate cannot re-execute a benchmark, so it cannot prove a number was
    /// measured. What it can do is refuse a record whose declared conditions
    /// are not *tied to anything*. Every field here is something the driver
    /// read off the launch it performed — the config bytes it passed, the argv
    /// the profile rendered to, the executable those bytes named — and every
    /// one of them is cross-checked in
    /// ``RuntimeBenchmark/admit(baseline:baselineAttestation:candidate:candidateAttestation:requiredScenarios:gateBinaryDigest:)``
    /// against the pins the record claims.
    ///
    /// This exists because review minted two records by hand with empty
    /// `revisions`, different `--config` paths and matching self-authored
    /// pins, and the production `benchmark-compare` accepted them and exited 0
    /// with `accepted=true`. Nothing in the record referred to a run, so there
    /// was nothing for the gate to disagree with.
    public struct LaunchProvenance: Sendable, Equatable, Codable {
        /// Complete argv of the driver process that produced this record.
        public let driverCommand: [String]
        /// SHA-256 over the driver script's own bytes.
        public let driverDigest: String
        /// Complete argv of the managed-launcher invocation the driver ran.
        public let harnessCommand: [String]
        /// Path of the launcher config the profile was read from.
        public let configPath: String
        /// SHA-256 over that config file's bytes. Both records must carry the
        /// same one: the two profiles live in a single file, so two runs that
        /// disagree here were not configured by the same document.
        public let configDigest: String
        /// Profile name selected out of that config.
        public let profile: String
        /// Executable the profile named.
        public let launchExecutable: String
        /// SHA-256 over that executable's bytes, so "the Swift Release
        /// product" is a specific 80 MB file and not a path that has been
        /// rebuilt since.
        public let launchExecutableDigest: String
        /// The profile's argv with `{host}` and `{port}` substituted — the
        /// tokens the runtime process actually received.
        public let launchArgv: [String]
        /// The runtime child's pid, as the driver resolved it for sampling.
        public let runtimeProcessID: Int

        public init(
            driverCommand: [String],
            driverDigest: String,
            harnessCommand: [String],
            configPath: String,
            configDigest: String,
            profile: String,
            launchExecutable: String,
            launchExecutableDigest: String,
            launchArgv: [String],
            runtimeProcessID: Int
        ) {
            self.driverCommand = driverCommand
            self.driverDigest = driverDigest
            self.harnessCommand = harnessCommand
            self.configPath = configPath
            self.configDigest = configDigest
            self.profile = profile
            self.launchExecutable = launchExecutable
            self.launchExecutableDigest = launchExecutableDigest
            self.launchArgv = launchArgv
            self.runtimeProcessID = runtimeProcessID
        }
    }

    /// Everything one runtime's benchmark pass produced.
    public struct RunRecord: Sendable, Equatable, Codable {
        /// Stable identity of the runtime under test, for example
        /// `python-mlx-lm` or `mlx-swift`.
        public let runtime: String
        /// Exact revisions of the runtime and its numerical stack, verbatim.
        public let revisions: [String: String]
        /// The command line the driver actually executed.
        ///
        /// Retained for readability; it is ``LaunchProvenance/harnessCommand``
        /// and the gate checks that it still is. On its own it named only
        /// `model-harness run`, which is why it could not establish anything:
        /// the driver invocation, the config bytes and the rendered argv were
        /// all outside it.
        public let command: [String]
        /// What the record is bound to. Non-optional: a record without it does
        /// not decode, so a hand-minted document fails at
        /// ``RuntimeBenchmark/decodeRecord(path:data:)`` rather than reaching a
        /// comparison.
        public let provenance: LaunchProvenance
        public let pins: Pins
        /// Wall-clock bounds of the whole pass, as Unix seconds. Used to prove
        /// the two runs did not overlap on a host that cannot hold both.
        public let startedAtUnixSeconds: Double
        public let finishedAtUnixSeconds: Double
        /// Raw Mach component from the sample that produced the whole-process
        /// resident-memory upper-bound peak. Never scored by itself.
        ///
        /// Resident size is not a usable size for an MLX process: three
        /// identical loads of this model reported 2 650, 10 774 and 14 056 MiB
        /// resident while the physical footprint stayed within 16 MiB and MLX's
        /// own active-bytes figure was byte-identical.
        public let peakPhysicalFootprintBytes: Int?
        /// Whole-process scored memory quantity. This is explicitly a
        /// conservative upper bound, not physical footprint or RSS.
        public let peakResidentMemory: RuntimeMemoryPeak
        public let scenarios: [ScenarioResult]
        /// Runtime-level differences that could not be pinned away, declared by
        /// the driver so they appear in the report instead of being discovered
        /// in it. Required — an empty list is a claim that there are none, and
        /// an absent list is a record that never considered the question.
        public let declaredAsymmetries: [String]

        public init(
            runtime: String,
            revisions: [String: String],
            command: [String],
            provenance: LaunchProvenance,
            pins: Pins,
            startedAtUnixSeconds: Double,
            finishedAtUnixSeconds: Double,
            peakPhysicalFootprintBytes: Int?,
            peakResidentMemory: RuntimeMemoryPeak = .absent,
            scenarios: [ScenarioResult],
            declaredAsymmetries: [String]
        ) {
            self.runtime = runtime
            self.revisions = revisions
            self.command = command
            self.provenance = provenance
            self.pins = pins
            self.startedAtUnixSeconds = startedAtUnixSeconds
            self.finishedAtUnixSeconds = finishedAtUnixSeconds
            self.peakPhysicalFootprintBytes = peakPhysicalFootprintBytes
            self.peakResidentMemory = peakResidentMemory
            self.scenarios = scenarios
            self.declaredAsymmetries = declaredAsymmetries
        }

        public func scenario(named name: String) -> ScenarioResult? {
            scenarios.first { $0.name == name }
        }
    }

    /// Why a pair of records was refused.
    public enum AdmissionError: Error, Equatable, CustomStringConvertible {
        case unreadable(path: String, detail: String)
        case malformed(path: String, detail: String)
        case pinMismatch(field: String, baseline: String, candidate: String)
        case sameRuntimeIdentity(String)
        case overlappingRuns(
            baseline: String, candidate: String, overlapSeconds: Double)
        case reversedInterval(runtime: String)
        case missingScenario(runtime: String, scenario: String)
        case missingRevisions(runtime: String)
        case malformedDigest(runtime: String, field: String, value: String)
        case configDigestMismatch(baseline: String, candidate: String)
        case sameLaunchExecutable(digest: String)
        case launchDoesNotCarryModel(runtime: String, modelPath: String)
        case harnessCommandUnbound(runtime: String, missing: String)
        case contextPolicyNotDerived(runtime: String, declared: String, derived: String)
        case unpinnedLaunchCondition(runtime: String, condition: String)
        case modelOfRecordNotDerived(runtime: String, declared: String, derived: String)
        case modelOfRecordUnread(runtime: String, path: String)
        case modelOfRecordUntrusted(runtime: String, path: String, digest: String)
        case speculationNotDerived(runtime: String, declared: String, derived: String)
        case speculativeDecodingActive(runtime: String, reading: String)
        case speculationNotHonoured(runtime: String, flag: String, declared: String)
        case equivalenceEvidenceDiffers(baseline: String, candidate: String)
        case equivalenceVerdictNotComparable(sourceOfRecord: String, verdict: String)
        case equivalenceDoesNotCoverArtifact(
            runtime: String, artifact: String, digest: String, sourceOfRecord: String)
        case equivalenceDeclaresNoNonEquivalences(sourceOfRecord: String)
        case equivalenceQuantizationDisagrees(runtime: String, pinned: String, declared: String)
        case declaredNonEquivalenceNotCarried(runtime: String, entry: String)
        case equivalenceEvidenceUnused(runtime: String, sourceOfRecord: String)
        case equivalenceEvidenceUntrusted(sourceOfRecord: String, digest: String)
        case trustedDecisionDisagrees(sourceOfRecord: String, detail: String)
        case contextBoundNotHonoured(
            runtime: String, flag: String, declared: Int, reported: String)
        case contextBoundUnreadable(runtime: String, flag: String, raw: String)
        case impossibleTiming(runtime: String, scenarioSeconds: Double, intervalSeconds: Double)
        case attestationAbsent(runtime: String, directory: String)
        case attestationUnreadable(path: String, detail: String)
        case attestationMalformed(path: String, detail: String)
        case attestationDisagrees(
            runtime: String, field: String, observed: String, declared: String)
        case attestationNeverClosed(runtime: String)
        case attestationOutsideRun(runtime: String, detail: String)
        case attestationDoesNotCoverScenarios(
            runtime: String, scenarioSeconds: Double, observedSeconds: Double)
        case attestationsShareProcess(processID: Int)
        case judgingBinaryDidNotObserve(runtime: String, observing: String, judging: String)
        case scenarioWithoutTranscript(runtime: String, scenario: String)
        case scenarioSuccessWithoutCompletion(runtime: String, scenario: String)
        case transcriptCarriesNoCompletion(runtime: String)
        case attestationSealsNoTranscript(runtime: String)
        case transcriptNotObserved(runtime: String, recomputed: String, observed: String)
        case transcriptOutsideObservation(runtime: String, detail: String)

        public var description: String {
            switch self {
            case .unreadable(let path, let detail):
                return "could not read benchmark record \(path.debugDescription): \(detail)"
            case .malformed(let path, let detail):
                return "benchmark record \(path.debugDescription) is malformed: \(detail)"
            case .pinMismatch(let field, let baseline, let candidate):
                return
                    "pinned condition \(field.debugDescription) differs: baseline "
                    + "\(baseline.debugDescription) vs candidate \(candidate.debugDescription); "
                    + "these runs are not a comparison"
            case .sameRuntimeIdentity(let runtime):
                return
                    "both records name runtime \(runtime.debugDescription); a runtime "
                    + "compared against itself cannot decide a migration"
            case .overlappingRuns(let baseline, let candidate, let overlap):
                return
                    "runs \(baseline.debugDescription) and \(candidate.debugDescription) overlap "
                    + "by \(overlap)s of wall clock; this host cannot hold two copies of the "
                    + "model, so overlapping runs measured each other's memory pressure"
            case .reversedInterval(let runtime):
                return
                    "record \(runtime.debugDescription) finished before it started; its interval "
                    + "cannot establish that the runs were sequential"
            case .missingScenario(let runtime, let scenario):
                return
                    "record \(runtime.debugDescription) has no scenario "
                    + "\(scenario.debugDescription); a scenario one runtime never ran is not a "
                    + "scenario the other one won"
            case .missingRevisions(let runtime):
                return
                    "record \(runtime.debugDescription) declares no revisions; a record that "
                    + "cannot name the code that ran is not evidence about a runtime"
            case .malformedDigest(let runtime, let field, let value):
                return
                    "record \(runtime.debugDescription) field \(field.debugDescription) is "
                    + "\(value.debugDescription), which is not a SHA-256 digest; the record is "
                    + "not bound to the bytes it names"
            case .configDigestMismatch(let baseline, let candidate):
                return
                    "the two runs were launched from different launcher configurations "
                    + "(baseline digest \(baseline.debugDescription), candidate "
                    + "\(candidate.debugDescription)); both profiles live in one file, so a "
                    + "digest mismatch means these runs were not configured together"
            case .sameLaunchExecutable(let digest):
                return
                    "both records launched executable digest \(digest.debugDescription); two "
                    + "records naming different runtimes cannot have run the same binary"
            case .launchDoesNotCarryModel(let runtime, let modelPath):
                return
                    "record \(runtime.debugDescription) pins modelPath "
                    + "\(modelPath.debugDescription) but its rendered launch argv never "
                    + "mentions it; the pin is not bound to the process that ran"
            case .harnessCommandUnbound(let runtime, let missing):
                return
                    "record \(runtime.debugDescription) has a launcher command that does not "
                    + "carry \(missing.debugDescription); the recorded configuration is not the "
                    + "one the launcher was given"
            case .contextPolicyNotDerived(let runtime, let declared, let derived):
                return
                    "record \(runtime.debugDescription) declares contextPolicy "
                    + "\(declared.debugDescription) but its rendered launch argv derives "
                    + "\(derived.debugDescription); the pin is the caller's claim, not the "
                    + "run's condition"
            case .unpinnedLaunchCondition(let runtime, let condition):
                return
                    "record \(runtime.debugDescription) left \(condition.debugDescription) to "
                    + "the runtime's own default; the two runtimes do not share defaults, so an "
                    + "unstated condition is an unpinned one"
            case .modelOfRecordNotDerived(let runtime, let declared, let derived):
                return
                    "record \(runtime.debugDescription) declares model of record "
                    + "\(declared.debugDescription) where the equivalence evidence the gate read "
                    + "for it derives \(derived.debugDescription); a record cannot name the model "
                    + "two runs share by writing a string"
            case .modelOfRecordUnread(let runtime, let path):
                return
                    "record \(runtime.debugDescription) names equivalence evidence at "
                    + "\(path.debugDescription) that the gate could not read or decode; an "
                    + "unreadable verdict is a failed read and is never spent as an absence of "
                    + "one, so this pass has no model of record at all"
            case .modelOfRecordUntrusted(let runtime, let path, let digest):
                return
                    "record \(runtime.debugDescription) names equivalence evidence at "
                    + "\(path.debugDescription), which read and decoded cleanly at digest "
                    + "\(digest.debugDescription) and is not an equivalence decision this "
                    + "repository took; a verdict the invocation authored for itself "
                    + "authenticates nothing, however carefully the gate then hashes and seals "
                    + "it, so this pass has no model of record at all"
            case .speculationNotDerived(let runtime, let declared, let derived):
                return
                    "record \(runtime.debugDescription) declares speculation "
                    + "\(declared.debugDescription) where its launch and the runtime the gate "
                    + "observed derive \(derived.debugDescription)"
            case .speculativeDecodingActive(let runtime, let reading):
                return
                    "record \(runtime.debugDescription) reads \(reading.debugDescription) for "
                    + "speculative decoding, and only \"off\" is comparable; speculation is not a "
                    + "condition two runtimes can share here -- llama.cpp can draft off an MTP "
                    + "head and the MLX baseline has none -- so a rate measured with it on is a "
                    + "different decoding algorithm rather than a faster runtime"
            case .speculationNotHonoured(let runtime, let flag, let declared):
                return
                    "record \(runtime.debugDescription) launched with \(flag) "
                    + "\(declared.debugDescription) and the runtime it observed reported that it "
                    + "was not speculating; the launch and the process disagree about which "
                    + "algorithm ran, and neither reading can be preferred over the other"
            case .equivalenceEvidenceDiffers(let baseline, let candidate):
                return
                    "the two passes cite different equivalence evidence "
                    + "(\(baseline.debugDescription) and \(candidate.debugDescription)); a "
                    + "cross-format comparison rests on one verdict about both artifacts, not on "
                    + "two documents that happen to agree"
            case .equivalenceVerdictNotComparable(let sourceOfRecord, let verdict):
                return
                    "the equivalence verdict for \(sourceOfRecord.debugDescription) is "
                    + "\(verdict.debugDescription), not \"comparable\"; only a comparable verdict "
                    + "admits, and \"incomplete\" is an unfinished analysis rather than a "
                    + "measured non-equivalence"
            case .equivalenceDoesNotCoverArtifact(
                let runtime, let artifact, let digest, let sourceOfRecord):
                return
                    "the equivalence verdict for \(sourceOfRecord.debugDescription) names no "
                    + "artifact at digest \(digest.debugDescription), which is what the gate "
                    + "computed over \(artifact.debugDescription) for record "
                    + "\(runtime.debugDescription); a verdict is bound to the files it was "
                    + "measured on by their digests and to nothing else"
            case .equivalenceDeclaresNoNonEquivalences(let sourceOfRecord):
                return
                    "the equivalence verdict for \(sourceOfRecord.debugDescription) declares no "
                    + "non-equivalences; two differently quantized artifacts found identical in "
                    + "every respect is an analysis that did not look, and there would be nothing "
                    + "for a later report to have to state"
            case .equivalenceQuantizationDisagrees(let runtime, let pinned, let declared):
                return
                    "record \(runtime.debugDescription) pins quantization "
                    + "\(pinned.debugDescription) where the equivalence verdict names "
                    + "\(declared.debugDescription) for the same artifact digest; the verdict and "
                    + "the record are not describing the same weights"
            case .declaredNonEquivalenceNotCarried(let runtime, let entry):
                return
                    "record \(runtime.debugDescription) does not carry declared non-equivalence "
                    + "\(entry.debugDescription); the differences a cross-format verdict did not "
                    + "dissolve travel with both records so that no report of this comparison can "
                    + "be read without them"
            case .equivalenceEvidenceUnused(let runtime, let sourceOfRecord):
                return
                    "record \(runtime.debugDescription) cites equivalence evidence for "
                    + "\(sourceOfRecord.debugDescription) while both passes served the same "
                    + "artifact; there is nothing for the two to be equivalent to, and an unused "
                    + "verdict beside a same-format pair is a claim nobody checked"
            case .equivalenceEvidenceUntrusted(let sourceOfRecord, let digest):
                return
                    "the equivalence verdict for \(sourceOfRecord.debugDescription) at digest "
                    + "\(digest.debugDescription) is not an equivalence decision this repository "
                    + "took; admission is bound to the decisions compiled into this gate, "
                    + "because a verdict the invocation supplies is one the invocation could "
                    + "have written -- hashing and sealing attacker-authored bytes proves only "
                    + "that they did not change between the read and the seal"
            case .trustedDecisionDisagrees(let sourceOfRecord, let detail):
                return
                    "the trusted equivalence decision for \(sourceOfRecord.debugDescription) and "
                    + "the document offered under its digest disagree: \(detail); the decision "
                    + "states what it measured where a reviewer reads it, so a document that "
                    + "hashes to it and says something else is a drift rather than evidence"
            case .contextBoundNotHonoured(let runtime, let flag, let declared, let reported):
                return
                    "record \(runtime.debugDescription) launched with \(flag) \(declared) but "
                    + "the runtime it observed reported \(reported); the context bound the "
                    + "launch pinned is not the one the process ran"
            case .contextBoundUnreadable(let runtime, let flag, let raw):
                return
                    "record \(runtime.debugDescription) launched with \(flag) "
                    + "\(raw.debugDescription), which is not a context length this gate can "
                    + "read; a bound the gate cannot read out of the launch is a failed read of "
                    + "what was asked for, and reading it as \"nothing was asked for\" is the "
                    + "one thing that lets a launch reach the argv fallback unchecked"
            case .impossibleTiming(let runtime, let scenarioSeconds, let intervalSeconds):
                return
                    "record \(runtime.debugDescription) reports \(scenarioSeconds)s of scenario "
                    + "wall clock inside a \(intervalSeconds)s pass; the scenarios it claims "
                    + "could not have run in the interval it claims"
            case .attestationAbsent(let runtime, let directory):
                return
                    "no attestation for runtime \(runtime.debugDescription) exists in "
                    + "\(directory.debugDescription); the gate never observed this pass, so the "
                    + "record is the only witness to itself and is not evidence"
            case .attestationUnreadable(let path, let detail):
                return
                    "could not read attestation \(path.debugDescription): \(detail); a failed "
                    + "read is not an absent attestation and is never treated as one"
            case .attestationMalformed(let path, let detail):
                return "attestation \(path.debugDescription) is malformed: \(detail)"
            case .attestationDisagrees(let runtime, let field, let observed, let declared):
                return
                    "record \(runtime.debugDescription) declares \(field.debugDescription) "
                    + "\(declared.debugDescription) but the gate observed "
                    + "\(observed.debugDescription); the record describes a run the gate did "
                    + "not watch"
            case .attestationNeverClosed(let runtime):
                return
                    "the attestation for \(runtime.debugDescription) was opened and never "
                    + "closed; the gate saw the pass begin and never saw it end, which is not "
                    + "the same as a pass that ended"
            case .attestationOutsideRun(let runtime, let detail):
                return
                    "the gate's observation window for \(runtime.debugDescription) does not sit "
                    + "inside the interval the record claims: \(detail)"
            case .attestationDoesNotCoverScenarios(
                let runtime, let scenarioSeconds, let observedSeconds):
                return
                    "record \(runtime.debugDescription) claims \(scenarioSeconds)s of scenario "
                    + "wall clock, but the gate watched the runtime for only "
                    + "\(observedSeconds)s; the scenarios it reports did not happen under "
                    + "observation"
            case .attestationsShareProcess(let processID):
                return
                    "both attestations observed pid \(processID) started at the same instant; "
                    + "two runtimes cannot be one process"
            case .judgingBinaryDidNotObserve(let runtime, let observing, let judging):
                return
                    "the attestation for \(runtime.debugDescription) was written by gate binary "
                    + "\(observing.debugDescription) and this comparison is being made by "
                    + "\(judging.debugDescription); a binary that did not watch these runs "
                    + "cannot certify that it was the one measured"
            case .scenarioWithoutTranscript(let runtime, let scenario):
                return
                    "record \(runtime.debugDescription) reports scenario "
                    + "\(scenario.debugDescription) with no transcript; a measurement that "
                    + "does not carry the requests it came from is a number, not an observation"
            case .scenarioSuccessWithoutCompletion(let runtime, let scenario):
                return
                    "record \(runtime.debugDescription) reports scenario "
                    + "\(scenario.debugDescription) as succeeded, but its transcript contains "
                    + "no served completion; answering some other endpoint is not serving this "
                    + "scenario"
            case .transcriptCarriesNoCompletion(let runtime):
                return
                    "record \(runtime.debugDescription) contains no scenario that ever completed "
                    + "a chat completion; the process under observation answered other endpoints "
                    + "and served nothing, so there is no benchmark here to judge"
            case .attestationSealsNoTranscript(let runtime):
                return
                    "the attestation for \(runtime.debugDescription) seals no transcript; it "
                    + "witnesses that a process existed, which is not a witness to what was "
                    + "measured"
            case .transcriptNotObserved(let runtime, let recomputed, let observed):
                return
                    "record \(runtime.debugDescription) digests to "
                    + "\(recomputed.debugDescription) and the observation seals "
                    + "\(observed.debugDescription); these measurements are not the ones the "
                    + "gate watched being taken"
            case .transcriptOutsideObservation(let runtime, let detail):
                return
                    "record \(runtime.debugDescription) reports work outside the window it was "
                    + "observed in: \(detail)"
            }
        }
    }

    /// The prompt-evaluation policy a rendered launch actually carries.
    ///
    /// The single source of truth for the ``Pins/contextPolicy`` pin: the
    /// driver writes what this returns for the launch it performed, and
    /// ``admit(baseline:baselineAttestation:candidate:candidateAttestation:requiredScenarios:gateBinaryDigest:)``
    /// re-derives it from the
    /// same recorded argv and refuses any record whose pin has drifted from it.
    /// A record therefore cannot declare a policy by writing a string.
    ///
    /// Three conditions are read, and all three are conditions the two runtimes do
    /// **not** default to the same way:
    ///
    /// * the KV bound, and this one is **not** read off argv. Earlier revisions
    ///   read the absence of `--max-kv-size` as `unbounded`, on the grounds
    ///   that "absent on both sides means the same thing". That premise held
    ///   for two runtimes and is false for the third: `llama-server` has no
    ///   unbounded mode, and measured on build `b10621-c1d0e7a00` it reports
    ///   `n_ctx` 8192 under `--ctx-size 8192` and 32768 with no context flag at
    ///   all. There is no additive argv spelling that rescues the premise —
    ///   absence still means "finite, taken from the model" for llama.cpp and
    ///   "no bound" for an unbounded `mlx_lm.server` — so the bound comes from
    ///   ``RuntimeContextWindow``, which the gate reads from the running
    ///   process. A runtime that names its bound pins that number; a runtime
    ///   that answered and named none falls back to `--max-kv-size` or
    ///   `unbounded`, which remains the two MLX runtimes' default case; the
    ///   bounded Python benchmark instead reports its constructed cache bound. A
    ///   bound the gate could not read at all is `unread`, which is
    ///   unpinnable and refused. The value is rendered as a bare number on
    ///   every path, so a llama.cpp `n_ctx` of 8192 and an MLX
    ///   `--max-kv-size 8192` are the same reading of the same condition
    ///   rather than two spellings that can never compare equal.
    /// * the prefill chunk. `mlx_lm.server` defaults `--prefill-step-size` to
    ///   `2048`; `MLXLMCommon.GenerateParameters` defaults `prefillStepSize` to
    ///   `512`; `llama-server` spells the same physical prompt-evaluation chunk
    ///   `-ub` / `--ubatch-size` and defaults it to `512` (its `--batch-size` is
    ///   the logical batch and is a different condition). All three spellings
    ///   are read and the *value* is what the pin carries. Absence here does
    ///   **not** mean the same thing on any two of them, so it is reported as
    ///   `unpinned` and refused rather than read as a value. Measuring
    ///   512-token chunks against 2048-token chunks and calling the difference
    ///   a runtime difference is exactly the comparison this pin exists to
    ///   prevent, and it is why the fix for llama.cpp widened the *derivation*
    ///   instead of relaxing ``unpinnableConditions``: dropping
    ///   `prefill-step=unpinned` from that list was measured to admit an
    ///   unpinned mlx-swift launch (512) against an unpinned `mlx_lm.server`
    ///   one (2048), because all three derive the byte-identical string.
    /// * the chat template's `reasoning_effort`. This model's template defaults
    ///   it to `xhigh` and injects an extra system instruction at that setting,
    ///   so a profile that states nothing renders a *different prompt* from one
    ///   that states `medium`. Review measured the consequence: 79 baseline
    ///   tokens against 41 candidate ones for the same messages, a constant +38
    ///   on every prompt in the suite, reported by revision 2 as a 1.93x
    ///   runtime skew. The two runtimes spell the same condition differently —
    ///   `--reasoning-effort medium` here, `--chat-template-args
    ///   '{"reasoning_effort": "medium"}'` for `mlx_lm.server` — so both
    ///   spellings are read and the *value* is what the pin carries. Absent on
    ///   either side is `unpinned` and refused: absence there is the template
    ///   default, which is not a shared default at all.
    /// - Parameter window: what the running runtime said its context bound was,
    ///   read by the gate over the wire inside its own observation window. Has
    ///   no default: a default here would be the very thing this parameter
    ///   exists to remove — a KV reading nobody asked the runtime for.
    public static func contextPolicy(
        observing window: RuntimeContextWindow,
        generationConfiguration: RuntimeGenerationConfiguration = .notReported
    ) -> String {
        let kv: String
        switch window.observation {
        case .observed(let length): kv = String(length)
        case .observedAbsent: kv = "not-reported"
        case .notObserved: kv = "unread"
        }
        return
            "kv=\(kv);prefill-step=\(generationConfiguration.prefillStepSize.policyValue);"
            + "reasoning=\(generationConfiguration.reasoningEffort.policyValue)"
    }

    /// Legacy argv derivation used only by compatibility fixtures that exercise
    /// historical refusal shapes. Production admission derives the effective
    /// policy from the two live observations above and never calls this helper.
    static func contextPolicy(
        derivedFrom argv: [String], observing window: RuntimeContextWindow
    ) -> String {
        func value(_ flag: String) -> String? {
            for (index, token) in argv.enumerated() {
                if token == flag, index + 1 < argv.count { return argv[index + 1] }
                if token.hasPrefix(flag + "=") { return String(token.dropFirst(flag.count + 1)) }
            }
            return nil
        }
        func reasoning() -> String {
            if let direct = value("--reasoning-effort") { return direct }
            guard let raw = value("--chat-template-args"), let data = raw.data(using: .utf8),
                let object = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                let effort = object["reasoning_effort"] as? String
            else { return "unpinned" }
            return effort
        }
        let kv: String
        switch window {
        case .reported(let length): kv = String(length)
        case .notReported: kv = value("--max-kv-size") ?? "unbounded"
        case .unread: kv = "unread"
        }
        let prefill =
            value("--prefill-step-size") ?? value("--ubatch-size") ?? value("-ub")
            ?? "unpinned"
        return "kv=\(kv);prefill-step=\(prefill);reasoning=\(reasoning())"
    }

    /// The model two runs are about, derived from what the gate read rather
    /// than from what either record says.
    ///
    /// The whole G4 decision is in these three lines:
    ///
    /// * no verdict named — the local artifact *is* the record, so the pin is
    ///   its digest and byte identity is still exactly what two runs must
    ///   share;
    /// * a verdict read — the pin is the upstream model both artifacts descend
    ///   from, and ``admitModelIdentity(baseline:baselineAttestation:candidate:candidateAttestation:)``
    ///   then has to bind that verdict to both artifacts by digest;
    /// * a verdict named and unreadable — `unread`, refused by name. A failed
    ///   read is not an absence, and spending it as one would turn an
    ///   unreadable file into a same-format pass over two different models.
    ///
    /// - Parameter artifactDigest: ``Pins/modelDigest``, computed by the gate
    ///   over the artifact this pass actually served.
    public static func modelOfRecord(
        artifactDigest: String, observing reading: ModelEquivalenceReading
    ) -> String {
        switch reading {
        case .noneDeclared: return "artifact:\(artifactDigest)"
        case .read(let equivalence, _): return "source:\(equivalence.sourceOfRecord)"
        case .unread: return "unread"
        // A document the invocation authored derives neither an artifact pin
        // nor a source of record. It is not the same fact as a failed read and
        // does not share its string, so a refusal names which of the two
        // happened.
        case .untrusted: return "untrusted"
        }
    }

    /// The value ``modelOfRecord(artifactDigest:observing:)`` produces when the
    /// gate was handed a verdict it could not read.
    ///
    /// Named rather than spelled inline at each site so the refusal and the
    /// derivation cannot drift apart.
    static let unreadModelOfRecord = "unread"

    /// The value ``modelOfRecord(artifactDigest:observing:)`` produces when the
    /// gate was handed a well-formed verdict no trusted decision covers.
    static let untrustedModelOfRecord = "untrusted"

    /// Whether the launch asked for speculative decoding, in whichever spelling
    /// it used.
    ///
    /// Every flag here was read off `llama-server --help` for the pinned build
    /// `b10621-c1d0e7a00`, not guessed. `--spec-type` is the switch — a
    /// comma-separated list defaulting to `none` — and a draft model is the
    /// other way in, under three spellings of one flag. `--draft` and
    /// `--draft-min` are listed by that build as removed, so they are not read:
    /// a launch carrying one gets a hard error from the runtime before this
    /// gate sees anything.
    ///
    /// Returns `nil` when the launch declares no speculation, and the flag's
    /// value otherwise — including an explicit `--spec-type none`, which
    /// reports `none` and is treated as no declaration by
    /// ``speculationPolicy(derivedFrom:observing:)``.
    static func declaredSpeculation(inArgv argv: [String]) -> (flag: String, value: String)? {
        func value(of flag: String) -> String? {
            for (index, token) in argv.enumerated() {
                if token == flag, index + 1 < argv.count { return argv[index + 1] }
                if token.hasPrefix(flag + "=") { return String(token.dropFirst(flag.count + 1)) }
            }
            return nil
        }
        // A flag that is there and whose value is not readable -- trailing,
        // or spelled `--spec-type=` with nothing after it -- is a launch this
        // gate cannot read rather than a launch that asked for nothing. It is
        // reported as a declaration, which refuses, because that is the
        // conservative direction and because the alternative is the F3 shape:
        // a failed read of what the launch asked for, spent as "it asked for
        // nothing".
        func present(_ flag: String) -> Bool {
            argv.contains { $0 == flag || $0.hasPrefix(flag + "=") }
        }
        if let requested = value(of: "--spec-type") {
            let kinds = requested.split(separator: ",").map {
                $0.trimmingCharacters(in: .whitespaces)
            }
            let active = kinds.filter { !$0.isEmpty && $0 != "none" }
            if !active.isEmpty { return ("--spec-type", active.joined(separator: ",")) }
            // An explicit `none` is the launch saying it wants no drafting.
            // Anything else that leaves `active` empty -- `--spec-type=` with
            // nothing after it, or a list of separators -- is a value this gate
            // could not read, which is not the same statement.
            return kinds.contains("none")
                ? nil : ("--spec-type", unreadableSpeculationValue)
        }
        if present("--spec-type") { return ("--spec-type", unreadableSpeculationValue) }
        for flag in ["--spec-draft-model", "--model-draft", "-md"] {
            if let draft = value(of: flag), !draft.isEmpty { return (flag, draft) }
            if present(flag) { return (flag, unreadableSpeculationValue) }
        }
        return nil
    }

    /// What ``declaredSpeculation(inArgv:)`` reports for a speculative flag
    /// whose value it could not read.
    static let unreadableSpeculationValue = "<unreadable>"

    /// Whether this pass was speculating, spelled the same way for every
    /// runtime.
    ///
    /// Same shape as ``contextPolicy(derivedFrom:observing:)`` and for the same
    /// reason: the reading comes from the running process, and argv is consulted
    /// only in the one case where the process answered and named nothing.
    ///
    /// * `off` — the process said it was not speculating, or said nothing and
    ///   the launch asked for nothing.
    /// * `on` — the process said it was.
    /// * `declared:<flag>=<value>` — the launch asked for speculation and the
    ///   process did not confirm it was off. Refused, and it names the flag, so
    ///   a `/slots`-less runtime cannot be launched into speculation and read
    ///   as quiet.
    /// * `unread` — no answer at all. Refused; a failed read is not an absence.
    ///
    /// Only `off` is admitted, and that is a refusal rather than a pin
    /// comparison on purpose — see ``Pins/speculation``.
    public static func speculationPolicy(
        derivedFrom argv: [String], observing speculation: RuntimeSpeculation
    ) -> String {
        switch speculation {
        // The process settles the question when it answers, in both
        // directions. A launch that declared speculation against a process
        // reporting `false` is a contradiction rather than a second reading of
        // the pin, and it is refused by name in
        // ``admitProvenance(_:observing:)`` — the same split
        // ``AdmissionError/contextBoundNotHonoured(runtime:flag:declared:reported:)``
        // makes, for the same reason: collapsing the two would leave one of
        // them invisible.
        case .reported(let active): return active ? "on" : "off"
        // The only case argv can speak to. `mlx_lm.server` and the Swift
        // prototype serve no `/slots`, so this is their normal answer, and a
        // launch that asked for speculation there cannot be read as quiet.
        case .notReported:
            guard let declared = declaredSpeculation(inArgv: argv) else { return "off" }
            return "declared:\(declared.flag)=\(declared.value)"
        case .unread: return "unread"
        }
    }

    /// The one admitted value of ``Pins/speculation``.
    static let admittedSpeculation = "off"

    /// What the *launch* asked for as a context bound, and whether that could
    /// be read at all.
    ///
    /// Three cases rather than an optional, for the reason every other reading
    /// in this gate has three: a flag that is absent and a flag whose value the
    /// gate could not parse are different facts. Folding the second into the
    /// first is the F2 shape one level up — with `--ctx-size abc` read as "no
    /// bound was asked for", the
    /// ``AdmissionError/contextBoundNotHonoured(runtime:flag:declared:reported:)``
    /// clause stops firing, and that clause is the only thing that keeps a
    /// llama.cpp launch away from the argv fallback when its server names no
    /// bound.
    enum DeclaredContextBound: Equatable {
        case none
        case pinned(flag: String, value: Int)
        case unreadable(flag: String, raw: String)
    }

    /// The context bound the *launch* asked for, in whichever spelling it used.
    ///
    /// Read only to be checked against what the runtime reported. It is never a
    /// source for the pin: `--ctx-size 8192` is a request, and `n_ctx` is what
    /// the process is running.
    ///
    /// An unreadable occurrence wins over a readable one wherever both appear,
    /// because the conservative direction here is the refusing one.
    static func declaredContextBound(inArgv argv: [String]) -> DeclaredContextBound {
        var pinned: DeclaredContextBound = .none
        for flag in ["--max-kv-size", "--ctx-size", "-c"] {
            for (index, token) in argv.enumerated() {
                var raw: String?
                if token == flag { raw = index + 1 < argv.count ? argv[index + 1] : "" }
                if token.hasPrefix(flag + "=") { raw = String(token.dropFirst(flag.count + 1)) }
                guard let raw else { continue }
                guard let parsed = Int(raw), parsed > 0 else {
                    return .unreadable(flag: flag, raw: raw)
                }
                if case .none = pinned { pinned = .pinned(flag: flag, value: parsed) }
            }
        }
        return pinned
    }

    /// Conditions the derived policy must not leave to a runtime default.
    ///
    /// `kv=unread` joins the list rather than replacing anything in it: a bound
    /// the gate failed to read is exactly as unusable as a prefill chunk left
    /// to a default, and it is the one KV reading that must never be scored.
    static let unpinnableConditions = [
        "kv=not-reported", "kv=unread", "prefill-step=not-reported",
        "prefill-step=unread", "reasoning=not-reported", "reasoning=unread",
    ]

    private static func isSHA256(_ value: String) -> Bool {
        value.count == 64 && value.allSatisfy { $0.isHexDigit && !$0.isUppercase }
    }

    /// Refuse a record that is not tied to a run it could have come from.
    ///
    /// Every clause here is a cross-check between two things the record says,
    /// not a re-execution: the gate cannot prove a benchmark happened, but it
    /// can refuse a document whose declared conditions contradict the launch it
    /// reports, or that reports no launch at all.
    /// - Parameter attestation: the gate's own observation of this pass, for
    ///   the one input the argv cannot supply — the context bound the running
    ///   runtime named. Its other fields are checked in
    ///   ``admitAttestation(_:for:gateBinaryDigest:)``, which runs later; a
    ///   forged window read here can therefore change *which* refusal a bad
    ///   pair gets, and cannot produce an acceptance, because the same document
    ///   still has to carry this binary's digest and the seal over the record.
    static func admitProvenance(
        _ record: RunRecord, observing attestation: RuntimeAttestation
    ) throws {
        guard !record.revisions.isEmpty else {
            throw AdmissionError.missingRevisions(runtime: record.runtime)
        }
        let provenance = record.provenance
        for (field, value) in [
            ("provenance.configDigest", provenance.configDigest),
            ("provenance.driverDigest", provenance.driverDigest),
            ("provenance.launchExecutableDigest", provenance.launchExecutableDigest),
        ] where !isSHA256(value) {
            throw AdmissionError.malformedDigest(
                runtime: record.runtime, field: field, value: value)
        }
        guard provenance.launchArgv.contains(record.pins.modelPath) else {
            throw AdmissionError.launchDoesNotCarryModel(
                runtime: record.runtime, modelPath: record.pins.modelPath)
        }
        for required in [provenance.configPath, provenance.profile]
        where !provenance.harnessCommand.contains(required) {
            throw AdmissionError.harnessCommandUnbound(
                runtime: record.runtime, missing: required)
        }
        guard record.command == provenance.harnessCommand else {
            throw AdmissionError.harnessCommandUnbound(
                runtime: record.runtime, missing: provenance.harnessCommand.joined(separator: " "))
        }
        // A declared bound contradicted by the live server is a more specific
        // refusal than the policy string mismatch it necessarily also causes.
        switch declaredContextBound(inArgv: provenance.launchArgv) {
        case .unreadable(let flag, let raw):
            throw AdmissionError.contextBoundUnreadable(
                runtime: record.runtime, flag: flag, raw: raw)
        case .pinned(let flag, let value):
            switch attestation.observedContextWindow {
            case .reported(let length) where length != value:
                throw AdmissionError.contextBoundNotHonoured(
                    runtime: record.runtime, flag: flag, declared: value,
                    reported: String(length))
            case .notReported where flag != "--max-kv-size":
                throw AdmissionError.contextBoundNotHonoured(
                    runtime: record.runtime, flag: flag, declared: value,
                    reported: "no bound at all")
            default: break
            }
        case .none: break
        }
        let derived = contextPolicy(
            observing: attestation.observedContextWindow,
            generationConfiguration: attestation.observedGenerationConfiguration ?? .unread)
        guard record.pins.contextPolicy == derived else {
            throw AdmissionError.contextPolicyNotDerived(
                runtime: record.runtime, declared: record.pins.contextPolicy, derived: derived)
        }
        for condition in unpinnableConditions where derived.contains(condition) {
            throw AdmissionError.unpinnedLaunchCondition(
                runtime: record.runtime, condition: condition)
        }
        // Re-derived from the gate's own reading, exactly as `contextPolicy` is
        // and for the same reason: a record must not be able to declare that
        // two different weight files are the same model by writing a string.
        let derivedModel = modelOfRecord(
            artifactDigest: record.pins.modelDigest,
            observing: attestation.observedModelEquivalence)
        guard record.pins.modelOfRecord == derivedModel else {
            throw AdmissionError.modelOfRecordNotDerived(
                runtime: record.runtime, declared: record.pins.modelOfRecord,
                derived: derivedModel)
        }
        if case .unread(let path) = attestation.observedModelEquivalence {
            throw AdmissionError.modelOfRecordUnread(runtime: record.runtime, path: path)
        }
        // F1. A document that read and decoded perfectly and is not a decision
        // this repository took. Refused here, at the same place as the failed
        // read and with its own name, so a caller is told which of the two
        // happened rather than being handed the generic pin mismatch that the
        // derived `untrusted` string would otherwise produce.
        if case .untrusted(let path, let digest) = attestation.observedModelEquivalence {
            throw AdmissionError.modelOfRecordUntrusted(
                runtime: record.runtime, path: path, digest: digest)
        }
        let derivedSpeculation = speculationPolicy(
            derivedFrom: provenance.launchArgv, observing: attestation.observedSpeculation)
        guard record.pins.speculation == derivedSpeculation else {
            throw AdmissionError.speculationNotDerived(
                runtime: record.runtime, declared: record.pins.speculation,
                derived: derivedSpeculation)
        }
        // A launch that asked for speculation against a process reporting it was
        // not speculating. Checked before the blanket refusal below so the pair
        // is told the specific thing -- the two disagree about which algorithm
        // ran -- rather than the general one, and checked at all because
        // `speculationPolicy` takes the process's answer and the launch's
        // request then disappears, the same way a `--ctx-size` does.
        if let declared = declaredSpeculation(inArgv: provenance.launchArgv),
            case .reported(false) = attestation.observedSpeculation
        {
            throw AdmissionError.speculationNotHonoured(
                runtime: record.runtime, flag: declared.flag, declared: declared.value)
        }
        guard derivedSpeculation == admittedSpeculation else {
            throw AdmissionError.speculativeDecodingActive(
                runtime: record.runtime, reading: derivedSpeculation)
        }
    }

    /// Decide whether the two passes were about the same model, and on what
    /// grounds.
    ///
    /// This is the G4 clause. It replaces "``Pins/modelPath`` and
    /// ``Pins/modelDigest`` must be equal", and it replaces it in a direction
    /// that admits one new class and refuses everything that class could be
    /// confused with.
    ///
    /// **Same artifact.** Both passes served one file, so there is nothing to
    /// be equivalent to. ``Pins/modelOfRecord`` already had to be equal in
    /// ``Pins/firstMismatch(against:)`` and it *is* `artifact:<digest>` here, so
    /// the digests are already known equal; what is left to demand is the path
    /// and the quantization, refused with the same `pinMismatch` as before.
    /// Evidence cited beside such a pair is refused rather than ignored: an
    /// unused verdict is a claim nobody checked.
    ///
    /// **Different artifacts.** Admissible only under evidence, and the
    /// evidence has to survive eight separate things, every one of which is
    /// more than the equality it replaces:
    ///
    /// 0. the document is an equivalence decision **this repository took** —
    ///    its SHA-256 appears in ``TrustedEquivalenceDecisions/shipped``, and
    ///    the decision's stated upstream model and required non-equivalences
    ///    are what the document carries. This is F1's clause and it is checked
    ///    first, because every clause after it reads a field out of a document
    ///    and none of them means anything until the document is evidence rather
    ///    than something the invocation wrote;
    /// 1. both passes carry a verdict — absence on either side is a refusal,
    ///    never a default pass;
    /// 2. it is the *same* verdict, by the digest the gate computed over the
    ///    document, so "the same evidence" is a fact about bytes rather than
    ///    two documents that agree;
    /// 3. its verdict is `comparable`;
    /// 4. it names an artifact at each side's gate-computed digest, so a
    ///    verdict about some other pair of files cannot be pointed at these;
    /// 5. the quantization each record pins agrees with what the verdict
    ///    records for that digest;
    /// 6. it declares at least one non-equivalence, and **every** one of them
    ///    is carried in **both** records' ``RunRecord/declaredAsymmetries`` —
    ///    which is what makes the dropped MTP head, the vision-tower placement
    ///    and the F32-versus-bf16 norms travel into every report of the
    ///    comparison instead of being consumed here and dropped. The trusted
    ///    decision's own required entries are demanded of both records
    ///    alongside the document's, so the three cannot be replaced by one
    ///    generic note even if the trust store and its document drift apart.
    ///
    /// The evidence is read off the two attestations rather than off the two
    /// records, because the attestation is the document the *gate* wrote. A
    /// record that minted its own verdict has already been refused by
    /// ``AdmissionError/modelOfRecordNotDerived(runtime:declared:derived:)``
    /// before this runs.
    ///
    /// **What is still not proven.** That the verdict is correct. The gate
    /// binds it to these two files by digest and checks its shape; whether 8.5
    /// bits per weight two different ways really are comparable is a
    /// measurement somebody else made, and it stays visible in the record
    /// rather than being turned into a silent pass.
    /// - Parameter trust: the equivalence decisions this repository took.
    ///   Defaults to ``TrustedEquivalenceDecisions/shipped``, which is what
    ///   every production call site uses; the parameter exists so the contract
    ///   suite can drive the clauses below with decisions of its own, and
    ///   `admissionUsesTheShippedTrustStoreByDefault` asserts the default is
    ///   the shipped one so a test registry cannot become the gate's.
    static func admitModelIdentity(
        baseline: RunRecord,
        baselineAttestation: RuntimeAttestation,
        candidate: RunRecord,
        candidateAttestation: RuntimeAttestation,
        trusting trust: [TrustedEquivalenceDecision] = TrustedEquivalenceDecisions.shipped
    ) throws {
        let baselineEvidence = baselineAttestation.observedModelEquivalence.equivalence
        let candidateEvidence = candidateAttestation.observedModelEquivalence.equivalence
        let sameArtifact =
            baseline.pins.modelDigest == candidate.pins.modelDigest
            && baseline.pins.modelPath == candidate.pins.modelPath

        if sameArtifact {
            for (record, evidence) in [
                (baseline, baselineEvidence), (candidate, candidateEvidence),
            ] {
                if let evidence {
                    throw AdmissionError.equivalenceEvidenceUnused(
                        runtime: record.runtime, sourceOfRecord: evidence.sourceOfRecord)
                }
            }
            guard baseline.pins.quantization == candidate.pins.quantization else {
                throw AdmissionError.pinMismatch(
                    field: "quantization", baseline: baseline.pins.quantization,
                    candidate: candidate.pins.quantization)
            }
            return
        }

        // Two different artifacts. Absence of evidence has already refused this
        // pair before control reaches here, and it did so *structurally* rather
        // than by a clause somebody has to remember to call: with no verdict
        // read, `modelOfRecord` derives to `artifact:<digest>`, the two digests
        // differ by definition of this branch, and
        // ``Pins/firstMismatch(against:)`` refuses them. A separate
        // "evidence absent" refusal here could therefore never fire, and a
        // clause that cannot fail is not a second opinion -- it is a line that
        // makes a gate look more careful than it is. What is left below is what
        // it costs to compare two artifacts once evidence *is* present.
        guard let baselineEvidence, candidateEvidence != nil else {
            throw AdmissionError.pinMismatch(
                field: "modelOfRecord", baseline: baseline.pins.modelOfRecord,
                candidate: candidate.pins.modelOfRecord)
        }
        // By the digest the gate computed over the document, not by its
        // contents: two separately authored files that happen to decode to the
        // same struct are two claims, and this clause is about there being one.
        guard
            case .read(_, let baselineDigest) = baselineAttestation.observedModelEquivalence,
            case .read(_, let candidateDigest) = candidateAttestation.observedModelEquivalence,
            baselineDigest == candidateDigest
        else {
            throw AdmissionError.equivalenceEvidenceDiffers(
                baseline: baselineAttestation.observedModelEquivalence.evidenceDigest
                    ?? "no evidence",
                candidate: candidateAttestation.observedModelEquivalence.evidenceDigest
                    ?? "no evidence")
        }
        // F1, and it is checked before anything the document says about
        // itself. Everything below this line reads fields out of a document;
        // this is the clause that decides whether the document is evidence at
        // all. `admitProvenance` already refused an ``ModelEquivalenceReading/untrusted(path:digest:)``
        // reading, which is where an invocation of `benchmark-run` is stopped;
        // this repeats the lookup against the digest carried on the attestation
        // so a hand-authored attestation claiming ``ModelEquivalenceReading/read(_:digest:)``
        // over a minted verdict is refused too.
        guard
            let anchor = TrustedEquivalenceDecisions.decision(
                documentDigest: baselineDigest, in: trust)
        else {
            throw AdmissionError.equivalenceEvidenceUntrusted(
                sourceOfRecord: baselineEvidence.sourceOfRecord, digest: baselineDigest)
        }
        guard anchor.sourceOfRecord == baselineEvidence.sourceOfRecord else {
            throw AdmissionError.trustedDecisionDisagrees(
                sourceOfRecord: anchor.sourceOfRecord,
                detail: "the document names upstream model "
                    + baselineEvidence.sourceOfRecord.debugDescription)
        }
        // The digest already fixes the document's contents, so in a run where
        // the anchor and the document agree this loop is an equality that
        // holds. What it guards is the drift: a later decision added to the
        // trust store with its digest updated and one of the measured
        // differences quietly replaced by a single generic note. That
        // substitution is exactly what F1 demonstrated, so the required entries
        // are stated in the trust store where a reviewer reads them and are
        // demanded of the document here and of both records below.
        for entry in anchor.requiredNonEquivalences
        where !baselineEvidence.declaredNonEquivalences.contains(entry) {
            throw AdmissionError.trustedDecisionDisagrees(
                sourceOfRecord: anchor.sourceOfRecord,
                detail: "the document does not declare " + entry.debugDescription)
        }
        guard baselineEvidence.verdict == .comparable else {
            throw AdmissionError.equivalenceVerdictNotComparable(
                sourceOfRecord: baselineEvidence.sourceOfRecord,
                verdict: baselineEvidence.verdict.rawValue)
        }
        for record in [baseline, candidate] {
            guard let artifact = baselineEvidence.artifact(digest: record.pins.modelDigest) else {
                throw AdmissionError.equivalenceDoesNotCoverArtifact(
                    runtime: record.runtime, artifact: record.pins.modelPath,
                    digest: record.pins.modelDigest,
                    sourceOfRecord: baselineEvidence.sourceOfRecord)
            }
            guard artifact.quantization == record.pins.quantization else {
                throw AdmissionError.equivalenceQuantizationDisagrees(
                    runtime: record.runtime, pinned: record.pins.quantization,
                    declared: artifact.quantization)
            }
        }
        guard !baselineEvidence.declaredNonEquivalences.isEmpty else {
            throw AdmissionError.equivalenceDeclaresNoNonEquivalences(
                sourceOfRecord: baselineEvidence.sourceOfRecord)
        }
        for record in [baseline, candidate] {
            // The union of what the document declared and what the trusted
            // decision requires. The two coincide for a document that matches
            // its anchor; the anchor's list is demanded separately so a record
            // cannot lose one of the three measured differences even under a
            // trust store whose document drifted.
            for entry in baselineEvidence.declaredNonEquivalences
                + anchor.requiredNonEquivalences
            where !record.declaredAsymmetries.contains(entry) {
                throw AdmissionError.declaredNonEquivalenceNotCarried(
                    runtime: record.runtime, entry: entry)
            }
        }
    }

    /// A pass cannot contain more scenario time than it lasted.
    ///
    /// Cheap, and the one clause that costs a wholesale fabrication something:
    /// invented scenario timings have to be small enough to fit inside the
    /// interval the same document claims for the whole run.
    ///
    /// Checked after ``AdmissionError/reversedInterval(runtime:)``, because a
    /// negative interval is a more specific thing to be told than an
    /// impossible one.
    static func admitTiming(_ record: RunRecord) throws {
        let scenarioSeconds = record.scenarios.compactMap(\.wallClockSeconds).reduce(0, +)
        let intervalSeconds = record.finishedAtUnixSeconds - record.startedAtUnixSeconds
        guard scenarioSeconds <= intervalSeconds else {
            throw AdmissionError.impossibleTiming(
                runtime: record.runtime, scenarioSeconds: scenarioSeconds,
                intervalSeconds: intervalSeconds)
        }
    }

    /// Refuse a record the gate did not watch being produced.
    ///
    /// Everything in ``admitProvenance(_:)`` is a cross-check between two
    /// things the *same document* says. That is worth having and it is not
    /// enough: review supplied a pair where all of it agreed, none of it had
    /// happened, and the production subcommand exited 0 with `accepted=true`.
    /// The clauses here compare the record against a document it did not
    /// author — one the gate binary wrote during the pass, from the kernel and
    /// from the wire, while the runtime process was alive.
    ///
    /// The residual is stated on ``RuntimeAttestation`` rather than hidden
    /// here: this makes a record insufficient on its own, it does not make a
    /// determined fabrication impossible on a host where the same user owns
    /// both files.
    static func admitAttestation(
        _ attestation: RuntimeAttestation,
        for record: RunRecord,
        gateBinaryDigest: String
    ) throws {
        // First, because everything below is a statement about a run this
        // binary claims to have watched. If it was not the watcher, the rest
        // of the document is somebody else's observation.
        guard attestation.gateBinaryDigest == gateBinaryDigest else {
            throw AdmissionError.judgingBinaryDidNotObserve(
                runtime: record.runtime, observing: attestation.gateBinaryDigest,
                judging: gateBinaryDigest)
        }
        for (field, observed, declared) in [
            ("runtime", attestation.runtime, record.runtime),
            (
                "provenance.runtimeProcessID", String(attestation.processID),
                String(record.provenance.runtimeProcessID)
            ),
            (
                "provenance.launchExecutableDigest", attestation.observedExecutableDigest,
                record.provenance.launchExecutableDigest
            ),
            ("provenance.configPath", attestation.configPath, record.provenance.configPath),
            ("provenance.configDigest", attestation.configDigest, record.provenance.configDigest),
            ("provenance.profile", attestation.profile, record.provenance.profile),
        ] where observed != declared {
            throw AdmissionError.attestationDisagrees(
                runtime: record.runtime, field: field, observed: observed, declared: declared)
        }
        guard let closedAt = attestation.closedAtUnixSeconds else {
            throw AdmissionError.attestationNeverClosed(runtime: record.runtime)
        }
        // Asked over the wire by the gate at close, so it is the model the
        // runtime was actually serving rather than the one the record names.
        guard let served = attestation.servedModelID else {
            throw AdmissionError.attestationNeverClosed(runtime: record.runtime)
        }
        guard served == record.pins.modelPath else {
            throw AdmissionError.attestationDisagrees(
                runtime: record.runtime, field: "pins.modelPath", observed: served,
                declared: record.pins.modelPath)
        }
        guard attestation.openedAtUnixSeconds >= record.startedAtUnixSeconds else {
            throw AdmissionError.attestationOutsideRun(
                runtime: record.runtime,
                detail:
                    "the gate opened at \(attestation.openedAtUnixSeconds), before the pass "
                    + "claims to have started at \(record.startedAtUnixSeconds)")
        }
        guard closedAt <= record.finishedAtUnixSeconds else {
            throw AdmissionError.attestationOutsideRun(
                runtime: record.runtime,
                detail:
                    "the gate closed at \(closedAt), after the pass claims to have finished at "
                    + "\(record.finishedAtUnixSeconds)")
        }
        guard closedAt >= attestation.openedAtUnixSeconds else {
            throw AdmissionError.attestationOutsideRun(
                runtime: record.runtime,
                detail: "the gate closed at \(closedAt) before it opened at "
                    + "\(attestation.openedAtUnixSeconds)")
        }
        let scenarioSeconds = record.scenarios.compactMap(\.wallClockSeconds).reduce(0, +)
        let observedSeconds = closedAt - attestation.openedAtUnixSeconds
        guard scenarioSeconds <= observedSeconds else {
            throw AdmissionError.attestationDoesNotCoverScenarios(
                runtime: record.runtime, scenarioSeconds: scenarioSeconds,
                observedSeconds: observedSeconds)
        }
    }

    /// Decode a record, keeping "could not read" and "read something invalid"
    /// apart from each other and from any notion of absence.
    ///
    /// A caller that collapsed these would let a permissions error, a truncated
    /// write or a half-flushed file read as a record with no scenarios — which
    /// the decision below would then score as a runtime that failed everything,
    /// or, worse, as one with nothing to object to.
    public static func decodeRecord(path: String, data: Data?) throws -> RunRecord {
        guard let data else {
            throw AdmissionError.unreadable(path: path, detail: "no bytes were read")
        }
        do {
            return try JSONDecoder().decode(RunRecord.self, from: data)
        } catch {
            throw AdmissionError.malformed(path: path, detail: String(describing: error))
        }
    }

    /// The admitted pair, with the pins they agree on hoisted out.
    public struct Comparison: Sendable, Equatable {
        public let baseline: RunRecord
        public let candidate: RunRecord
        public let pins: Pins
        /// Scenario names present in both records, in baseline order.
        public let sharedScenarios: [String]
        /// Every asymmetry either record declared, deduplicated, in the order
        /// baseline-then-candidate.
        public let declaredAsymmetries: [String]
    }

    /// Admit two records for comparison, or refuse and say exactly why.
    ///
    /// - Parameter requiredScenarios: scenario names that must be present in
    ///   *both* records. A decision that silently skipped the 75k-context run
    ///   because one runtime never recorded it would be reporting a narrower
    ///   comparison than the one it claims.
    /// - Parameters:
    ///   - baselineAttestation: what the gate binary itself observed of the
    ///     baseline pass. Non-optional, and separately for both records: an
    ///     optional here would be a bypass with a default value, and the only
    ///     shape review's forged pair needed was a gate that would compare
    ///     without one.
    ///   - gateBinaryDigest: SHA-256 of the binary making this comparison. Both
    ///     attestations must name it, so the binary that judges is the binary
    ///     that watched — which is also the defect review found in revision 2,
    ///     where `19c54c…` served and `3e5fdcc…` judged.
    public static func admit(
        baseline: RunRecord,
        baselineAttestation: RuntimeAttestation,
        candidate: RunRecord,
        candidateAttestation: RuntimeAttestation,
        requiredScenarios: [String],
        gateBinaryDigest: String,
        trusting trust: [TrustedEquivalenceDecision] = TrustedEquivalenceDecisions.shipped
    ) throws -> Comparison {
        guard baseline.runtime != candidate.runtime else {
            throw AdmissionError.sameRuntimeIdentity(baseline.runtime)
        }
        if let mismatch = baseline.pins.firstMismatch(against: candidate.pins) {
            throw AdmissionError.pinMismatch(
                field: mismatch.field, baseline: mismatch.mine, candidate: mismatch.theirs)
        }
        // After the pin comparison, because a pin that simply differs deserves
        // the more specific refusal. The interesting case is the one the pin
        // comparison *cannot* see: two records that agree on every pin because
        // one caller typed the same values into both. Nothing above this line
        // can tell that apart from two runs.
        try admitProvenance(baseline, observing: baselineAttestation)
        try admitProvenance(candidate, observing: candidateAttestation)
        // After `admitProvenance`, because that is where each record's
        // `modelOfRecord` is re-derived from the evidence the gate read; this
        // clause then compares the two, and it can only be reached by a pair
        // whose pins are the readings the gate actually made.
        try admitModelIdentity(
            baseline: baseline, baselineAttestation: baselineAttestation,
            candidate: candidate, candidateAttestation: candidateAttestation,
            trusting: trust)
        guard baseline.provenance.configDigest == candidate.provenance.configDigest else {
            throw AdmissionError.configDigestMismatch(
                baseline: baseline.provenance.configDigest,
                candidate: candidate.provenance.configDigest)
        }
        guard
            baseline.provenance.launchExecutableDigest
                != candidate.provenance.launchExecutableDigest
        else {
            throw AdmissionError.sameLaunchExecutable(
                digest: baseline.provenance.launchExecutableDigest)
        }
        for record in [baseline, candidate]
        where record.finishedAtUnixSeconds < record.startedAtUnixSeconds {
            throw AdmissionError.reversedInterval(runtime: record.runtime)
        }
        try admitTiming(baseline)
        try admitTiming(candidate)
        let overlap =
            min(baseline.finishedAtUnixSeconds, candidate.finishedAtUnixSeconds)
            - max(baseline.startedAtUnixSeconds, candidate.startedAtUnixSeconds)
        if overlap > 0 {
            throw AdmissionError.overlappingRuns(
                baseline: baseline.runtime, candidate: candidate.runtime,
                overlapSeconds: overlap)
        }
        // And then against something neither record wrote. Everything above
        // this line is the two documents agreeing with themselves and with each
        // other; that is what review's forged pair also did. The clauses below
        // are the only ones a caller cannot satisfy by writing a record.
        //
        // Ordered last among the refusals so that a record which is *internally*
        // broken — a reversed interval, scenario time that does not fit, two
        // passes that overlapped — is still told the specific thing that is
        // wrong with it rather than being told the gate did not watch it.
        try admitAttestation(
            baselineAttestation, for: baseline, gateBinaryDigest: gateBinaryDigest)
        try admitAttestation(
            candidateAttestation, for: candidate, gateBinaryDigest: gateBinaryDigest)
        // And then the clause the three previous revisions did not have. Above
        // this line every check is satisfied by two live processes and a set of
        // numbers typed beside them — which is precisely the pass review built:
        // two placeholder servers answering `GET /v1/models`, attested end to
        // end, scored `accepted=true` in 7.2 seconds. What was missing is any
        // link between the process that was watched and the work that was
        // reported. These two clauses are that link: the measurements have to
        // be the ones the observer sealed, and the pass has to contain a
        // completion that was actually served.
        try admitTranscripts(baseline, requiredScenarios: requiredScenarios)
        try admitTranscripts(candidate, requiredScenarios: requiredScenarios)
        try admitTranscriptObservation(baselineAttestation, for: baseline)
        try admitTranscriptObservation(candidateAttestation, for: candidate)
        // Pid *and* start time, not pid alone. The two passes are sequential by
        // construction, so the second runtime can legitimately be handed the
        // number the first one released; what cannot happen is two runtimes
        // being the same process, and only the start time can tell those apart.
        guard
            !(baselineAttestation.processID == candidateAttestation.processID
                && baselineAttestation.processStartUnixSeconds
                    == candidateAttestation.processStartUnixSeconds)
        else {
            throw AdmissionError.attestationsShareProcess(
                processID: baselineAttestation.processID)
        }
        // Sequencing is not re-checked here on purpose. Each observation
        // window is required to sit inside its own record's interval and the
        // two records are already refused when they overlap, so a separate
        // overlap check over the windows could never fire. A clause that cannot
        // fail is not a second opinion; it is a line that makes a gate look
        // more careful than it is.
        for name in requiredScenarios {
            if baseline.scenario(named: name) == nil {
                throw AdmissionError.missingScenario(runtime: baseline.runtime, scenario: name)
            }
            if candidate.scenario(named: name) == nil {
                throw AdmissionError.missingScenario(runtime: candidate.runtime, scenario: name)
            }
        }
        let shared = baseline.scenarios.map(\.name).filter { name in
            candidate.scenario(named: name) != nil
        }
        var asymmetries: [String] = []
        for entry in baseline.declaredAsymmetries + candidate.declaredAsymmetries
        where !asymmetries.contains(entry) {
            asymmetries.append(entry)
        }
        return Comparison(
            baseline: baseline, candidate: candidate, pins: baseline.pins,
            sharedScenarios: shared, declaredAsymmetries: asymmetries)
    }
}

extension RuntimeBenchmark {
    /// The canonical seal over everything a pass measured.
    ///
    /// Computed by the observing process over the record it has just built, and
    /// written into ``RuntimeAttestation/transcriptDigest``; recomputed here at
    /// admission over the record on disk. The two must agree, so a scenario
    /// block cannot be edited or authored after the observation that covers it.
    ///
    /// It covers the measurements *and* the exchanges, not just the exchanges.
    /// Sealing only the transcript would leave the reported TTFT free to be
    /// anything while the requests behind it stayed honest — which is the same
    /// bypass one level down.
    ///
    /// The encoding is written out by hand rather than taken from `JSONEncoder`
    /// because a digest has to be stable against key ordering, floating-point
    /// formatting and any future field added to these structs for readability.
    /// `String(describing:)` on a `Double` is exact for the values here: they
    /// come from `Date().timeIntervalSince1970` and from arithmetic on it, and
    /// Swift prints the shortest representation that round-trips.
    public static func transcriptDigest(of record: RunRecord) -> String {
        var lines: [String] = ["runtime=\(record.runtime)"]
        func appendMemory(_ label: String, _ peak: RuntimeMemoryPeak) {
            lines.append("\(label).accounting=\(peak.accounting.rawValue)")
            lines.append("\(label).semantics=\(peak.scoreSemantics)")
            lines.append("\(label).status=\(peak.status.rawValue)")
            lines.append("\(label).scored=\(number(peak.scoredBytes))")
            lines.append("\(label).complete=\(peak.successfulSampleCount)")
            lines.append("\(label).readFailed=\(peak.readFailureCount)")
            lines.append("\(label).malformed=\(peak.malformedSampleCount)")
            lines.append(
                "\(label).mach=\(number(peak.peakSample?.machPhysicalFootprintBytes))")
            lines.append(
                "\(label).mappedRaw=\(peak.peakSample?.vmmapResidentMappedFileRaw ?? "-")")
            lines.append(
                "\(label).mappedUpper="
                    + number(peak.peakSample?.residentMappedFileBytesUpperBound))
            if let rawSamples = peak.rawSamples {
                lines.append("\(label).rawSamples=\(rawSamples.count)")
                for sample in rawSamples {
                    lines.append(
                        "\(label).raw="
                            + [
                                number(sample.sampledAtUnixSeconds),
                                number(sample.machSampledAtUnixSeconds),
                                number(sample.mappedFileSampledAtUnixSeconds),
                                String(sample.machPhysicalFootprintBytes),
                                sample.vmmapResidentMappedFileRaw ?? "-",
                                String(sample.residentMappedFileBytesUpperBound),
                            ].joined(separator: ","))
                }
            } else {
                lines.append("\(label).rawSamples=-")
            }
            for issue in peak.issues { lines.append("\(label).issue=\(issue)") }
        }
        // Sealed with the measurements since G4, because they stopped being
        // commentary. A cross-format pair is admitted only while both records
        // carry every non-equivalence its verdict declared, so an entry deleted
        // from a record after the pass would otherwise change what a report has
        // to state without disturbing anything the observer signed.
        for entry in record.declaredAsymmetries { lines.append("asymmetry=\(entry)") }
        appendMemory("processMemory", record.peakResidentMemory)
        for scenario in record.scenarios {
            lines.append("scenario=\(scenario.name)")
            lines.append("succeeded=\(scenario.succeeded)")
            lines.append("failureMode=\(scenario.failureMode ?? "-")")
            lines.append("promptTokens=\(number(scenario.promptTokens))")
            lines.append("completionTokens=\(number(scenario.completionTokens))")
            lines.append("ttft=\(number(scenario.timeToFirstTokenSeconds))")
            lines.append("prefill=\(number(scenario.prefillTokensPerSecond))")
            lines.append("decode=\(number(scenario.decodeTokensPerSecond))")
            lines.append("wall=\(number(scenario.wallClockSeconds))")
            lines.append("windowPeak=\(number(scenario.peakPhysicalFootprintBytes))")
            lines.append("processPeak=\(number(scenario.processPeakSoFarBytes))")
            appendMemory("scenarioMemory", scenario.peakResidentMemory)
            appendMemory("scenarioProcessMemory", scenario.processResidentMemoryPeakSoFar)
            lines.append("hostLoad=\(number(scenario.hostLoadAverageMax))")
            lines.append("cacheReuse.state=\(scenario.cacheReuse.state.rawValue)")
            lines.append("cacheReuse.source=\(scenario.cacheReuse.source)")
            lines.append(
                "cacheReuse.cachedPromptTokens="
                    + (scenario.cacheReuse.cachedPromptTokens?.map(String.init).joined(
                        separator: ",")
                        ?? "-"))
            lines.append("cacheReuse.issue=\(scenario.cacheReuse.issue ?? "-")")
            guard let transcript = scenario.transcript else {
                lines.append("transcript=absent")
                continue
            }
            lines.append("exchanges=\(transcript.exchanges.count)")
            for exchange in transcript.exchanges {
                lines.append(
                    [
                        exchange.method, exchange.path, exchange.requestDigest,
                        String(exchange.requestByteCount), String(exchange.status),
                        exchange.responseDigest, String(exchange.responseByteCount),
                        number(exchange.sentAtUnixSeconds),
                        number(exchange.firstByteAtUnixSeconds),
                        number(exchange.lastByteAtUnixSeconds),
                    ].joined(separator: "|"))
            }
        }
        let payload = Data(lines.joined(separator: "\n").utf8)
        return SHA256.hash(data: payload).map { String(format: "%02x", $0) }.joined()
    }

    /// `-` for absent, so a missing value and a zero never digest alike.
    private static func number(_ value: Int?) -> String {
        value.map(String.init) ?? "-"
    }

    private static func number(_ value: Double?) -> String {
        value.map { String(describing: $0) } ?? "-"
    }

    /// Refuse a record whose numbers are not attached to the requests they came
    /// from, or whose requests were never served.
    ///
    /// Three separate refusals, because they are three different things to be
    /// told:
    ///
    /// * a required scenario with no transcript at all — the driver reported a
    ///   measurement and not what it measured;
    /// * a scenario that claims success and carries no served completion — the
    ///   shape a `/v1/models` placeholder produces;
    /// * a whole pass in which nothing was ever completed — review's
    ///   reproduction, where two placeholder servers answered one `GET` each
    ///   and the record beside them reported six scenarios.
    ///
    /// The last one is inadmissible rather than a rejection on the numbers, and
    /// the distinction is the point. "The candidate lost" is a comparison this
    /// gate agreed to make; a pass in which no runtime ever served anything is
    /// not a comparison at all, and answering it with a verdict would be the
    /// gate certifying its own blindness.
    static func admitTranscripts(_ record: RunRecord, requiredScenarios: [String]) throws {
        for name in requiredScenarios {
            guard let scenario = record.scenario(named: name) else { continue }
            guard scenario.transcript != nil else {
                throw AdmissionError.scenarioWithoutTranscript(
                    runtime: record.runtime, scenario: name)
            }
        }
        for scenario in record.scenarios where scenario.succeeded {
            guard let transcript = scenario.transcript, transcript.carriesServedCompletion else {
                throw AdmissionError.scenarioSuccessWithoutCompletion(
                    runtime: record.runtime, scenario: scenario.name)
            }
        }
        let served = record.scenarios.contains {
            $0.transcript?.carriesServedCompletion ?? false
        }
        guard served else {
            throw AdmissionError.transcriptCarriesNoCompletion(runtime: record.runtime)
        }
    }

    /// Bind the record's measurements to the observation that covers them.
    ///
    /// Two clauses. The seal has to be the one the observer computed over this
    /// record, and every request in it has to have happened inside the window
    /// the observer was watching. Neither is satisfiable by a caller who was
    /// handed an attestation for a process it did not drive.
    static func admitTranscriptObservation(
        _ attestation: RuntimeAttestation, for record: RunRecord
    ) throws {
        guard let sealed = attestation.transcriptDigest else {
            throw AdmissionError.attestationSealsNoTranscript(runtime: record.runtime)
        }
        let recomputed = transcriptDigest(of: record)
        guard sealed == recomputed else {
            throw AdmissionError.transcriptNotObserved(
                runtime: record.runtime, recomputed: recomputed, observed: sealed)
        }
        guard let closedAt = attestation.closedAtUnixSeconds else {
            throw AdmissionError.attestationNeverClosed(runtime: record.runtime)
        }
        for scenario in record.scenarios {
            guard let transcript = scenario.transcript else { continue }
            for instant in transcript.instants
            where instant < attestation.openedAtUnixSeconds || instant > closedAt {
                throw AdmissionError.transcriptOutsideObservation(
                    runtime: record.runtime,
                    detail:
                        "scenario \(scenario.name.debugDescription) records an exchange at "
                        + "\(instant), outside the window the gate watched "
                        + "(\(attestation.openedAtUnixSeconds) to \(closedAt))")
            }
        }
    }
}

extension RuntimeBenchmark {
    /// Ratios the candidate must stay inside to be eligible for the default
    /// profile, plus the parity scenarios it must not lose.
    public struct Thresholds: Sendable, Equatable, Codable {
        /// Candidate TTFT divided by baseline TTFT, at most.
        public let maxTimeToFirstTokenRatio: Double
        /// Candidate prefill tokens/s divided by baseline, at least.
        public let minPrefillThroughputRatio: Double
        /// Candidate decode tokens/s divided by baseline, at least.
        public let minDecodeThroughputRatio: Double
        /// Candidate peak physical footprint divided by baseline, at most.
        ///
        /// Applied to the **scenario-local** peak of each scored scenario. The
        /// whole-process peak is reported against the same bound only when
        /// every parity scenario succeeded on both runtimes; otherwise the two
        /// maxima come from different completed work and the axis is reported
        /// as non-comparable instead of as a pass.
        public let maxPeakFootprintRatio: Double
        /// How far the two runtimes' rendered prompt lengths may diverge on a
        /// scored scenario before its latency and throughput stop being a
        /// comparison.
        ///
        /// The two runtimes render the same messages through the same chat
        /// template with different tokenizer front ends, and on the short
        /// scenario review measured 41 tokens against 79 — a 1.93x workload
        /// difference underneath a "0.751x TTFT win". Below this bound the
        /// scenario is scored; above it the scenario's ratios are reported
        /// without a verdict and the divergence is a blocker, because a faster
        /// number on a smaller prompt is not a faster runtime.
        public let maxPromptTokenSkewRatio: Double
        /// Scenarios where a baseline success and a candidate failure is a
        /// blocker on its own, whatever the ratios say.
        public let paritySuccessScenarios: [String]
        /// Scenarios whose throughput and latency ratios are scored.
        public let scoredScenarios: [String]

        public init(
            maxTimeToFirstTokenRatio: Double,
            minPrefillThroughputRatio: Double,
            minDecodeThroughputRatio: Double,
            maxPeakFootprintRatio: Double,
            maxPromptTokenSkewRatio: Double,
            paritySuccessScenarios: [String],
            scoredScenarios: [String]
        ) {
            self.maxTimeToFirstTokenRatio = maxTimeToFirstTokenRatio
            self.minPrefillThroughputRatio = minPrefillThroughputRatio
            self.minDecodeThroughputRatio = minDecodeThroughputRatio
            self.maxPeakFootprintRatio = maxPeakFootprintRatio
            self.maxPromptTokenSkewRatio = maxPromptTokenSkewRatio
            self.paritySuccessScenarios = paritySuccessScenarios
            self.scoredScenarios = scoredScenarios
        }
    }

    /// One scored metric, with the ratio spelled out so a reader can check the
    /// arithmetic against the raw records.
    public struct MetricDelta: Sendable, Equatable, Codable {
        public let scenario: String
        public let metric: String
        public let baseline: Double?
        public let candidate: Double?
        public let ratio: Double?
        public let admissibleRatio: String
        public let verdict: String

        public init(
            scenario: String, metric: String, baseline: Double?, candidate: Double?,
            ratio: Double?, admissibleRatio: String, verdict: String
        ) {
            self.scenario = scenario
            self.metric = metric
            self.baseline = baseline
            self.candidate = candidate
            self.ratio = ratio
            self.admissibleRatio = admissibleRatio
            self.verdict = verdict
        }
    }

    /// The decision itself.
    public struct Decision: Sendable, Equatable, Codable {
        /// `true` only when every scored metric is inside its threshold, every
        /// parity scenario the baseline won is also won by the candidate, and
        /// nothing needed was left unmeasured.
        public let accepted: Bool
        public let blockers: [String]
        public let deltas: [MetricDelta]
        public let declaredAsymmetries: [String]
    }

    /// Score an admitted comparison against thresholds.
    ///
    /// Fails closed on every axis. An unmeasured metric is a blocker, not a
    /// pass: the question "is the candidate no worse" has no affirmative answer
    /// from a number nobody took, and a decision that treated `nil` as "nothing
    /// to object to" would accept a migration on the strength of a broken
    /// driver. A baseline that itself failed a parity scenario is *not* a
    /// blocker against the candidate — the candidate cannot be required to beat
    /// a bar the incumbent never cleared — but it is reported.
    public static func decide(
        comparison: Comparison,
        thresholds: Thresholds
    ) -> Decision {
        var blockers: [String] = []
        var deltas: [MetricDelta] = []

        for name in thresholds.paritySuccessScenarios {
            guard let baseline = comparison.baseline.scenario(named: name),
                let candidate = comparison.candidate.scenario(named: name)
            else {
                blockers.append(
                    "parity scenario \(name.debugDescription) is missing from a record, so "
                        + "parity is unknown rather than met")
                continue
            }
            if baseline.succeeded && !candidate.succeeded {
                blockers.append(
                    "parity scenario \(name.debugDescription): baseline succeeded and candidate "
                        + "failed (\(candidate.failureMode ?? "no failure mode recorded"))")
            }
        }

        for name in thresholds.scoredScenarios {
            guard let baseline = comparison.baseline.scenario(named: name),
                let candidate = comparison.candidate.scenario(named: name)
            else {
                blockers.append(
                    "scored scenario \(name.debugDescription) is missing from a record")
                continue
            }
            // A scenario that did not succeed has no throughput to score. Say
            // so rather than dividing whatever numbers survived the failure.
            guard baseline.succeeded, candidate.succeeded else {
                blockers.append(
                    "scored scenario \(name.debugDescription) did not succeed on both runtimes "
                        + "(baseline \(baseline.succeeded ? "ok" : "failed"), candidate "
                        + "\(candidate.succeeded ? "ok" : "failed")), so its metrics are unknown")
                continue
            }
            // Scored first, because its verdict decides whether the three
            // below are a comparison at all.
            //
            // The ratio here is ``promptTokenSkew(baseline:candidate:)``, which
            // is symmetric and never below 1. The raw `candidate / baseline`
            // would pass a `<=` bound whenever the candidate rendered the
            // *shorter* prompt — which is the direction the defect actually
            // took: 41 candidate tokens against 79 baseline ones is 0.519, and
            // 0.519 is inside every upper bound.
            let skew = promptTokenSkew(baseline: baseline, candidate: candidate)
            let skewBound = thresholds.maxPromptTokenSkewRatio
            let promptComparable = skew.map { $0 <= skewBound } ?? false
            let cacheIssue = cacheComparabilityIssue(
                scenario: name, baseline: baseline.cacheReuse, candidate: candidate.cacheReuse)
            if let cacheIssue { blockers.append(cacheIssue) }
            let comparable = promptComparable && cacheIssue == nil
            if let skew {
                if !promptComparable {
                    blockers.append(
                        "\(name)/prompt_tokens skew \(skew) is outside the admissible band "
                            + "<= \(skewBound) (baseline "
                            + "\(baseline.promptTokens.map(String.init) ?? "unmeasured"), "
                            + "candidate "
                            + "\(candidate.promptTokens.map(String.init) ?? "unmeasured")); the "
                            + "two runtimes rendered materially different prompts, so this "
                            + "scenario's latency and throughput are not a comparison")
                }
                deltas.append(
                    MetricDelta(
                        scenario: name, metric: "prompt_tokens",
                        baseline: baseline.promptTokens.map(Double.init),
                        candidate: candidate.promptTokens.map(Double.init),
                        ratio: skew, admissibleRatio: "<= \(skewBound)",
                        verdict: promptComparable ? "within" : "outside"))
            } else {
                blockers.append(
                    "\(name)/prompt_tokens was not measured on both runtimes, so whether the "
                        + "two ran the same workload is unknown; unknown is not within threshold")
                deltas.append(
                    MetricDelta(
                        scenario: name, metric: "prompt_tokens",
                        baseline: baseline.promptTokens.map(Double.init),
                        candidate: candidate.promptTokens.map(Double.init),
                        ratio: nil, admissibleRatio: "<= \(skewBound)", verdict: "unmeasured"))
            }
            appendDelta(
                &deltas, &blockers, scenario: name, metric: "time_to_first_token_seconds",
                baseline: baseline.timeToFirstTokenSeconds,
                candidate: candidate.timeToFirstTokenSeconds,
                bound: thresholds.maxTimeToFirstTokenRatio, lowerIsBetter: true,
                comparable: comparable)
            appendDelta(
                &deltas, &blockers, scenario: name, metric: "prefill_tokens_per_second",
                baseline: baseline.prefillTokensPerSecond,
                candidate: candidate.prefillTokensPerSecond,
                bound: thresholds.minPrefillThroughputRatio, lowerIsBetter: false,
                comparable: comparable)
            appendDelta(
                &deltas, &blockers, scenario: name, metric: "decode_tokens_per_second",
                baseline: baseline.decodeTokensPerSecond,
                candidate: candidate.decodeTokensPerSecond,
                bound: thresholds.minDecodeThroughputRatio, lowerIsBetter: false,
                comparable: comparable)
            // Scenario-local, so it is the cost of *this* work rather than the
            // running maximum of everything the process has done so far.
            appendDelta(
                &deltas, &blockers, scenario: name,
                metric: "peak_resident_memory_upper_bound_bytes",
                baseline: baseline.peakResidentMemory.validatedScoredBytes.map(Double.init),
                candidate: candidate.peakResidentMemory.validatedScoredBytes.map(Double.init),
                bound: thresholds.maxPeakFootprintRatio, lowerIsBetter: true,
                comparable: comparable)
        }

        // The whole-process maximum is only a comparison when both runtimes
        // completed the same work. A candidate that aborted the 75k probe never
        // paid for it, so its lower whole-pass peak is not a smaller appetite —
        // it is a shorter pass. Reported either way; scored only when the pass
        // it summarises is the same pass.
        let completedSameWork = thresholds.paritySuccessScenarios.allSatisfy { name in
            guard let baseline = comparison.baseline.scenario(named: name),
                let candidate = comparison.candidate.scenario(named: name)
            else { return false }
            return baseline.succeeded == candidate.succeeded
        }
        appendDelta(
            &deltas, &blockers, scenario: "process",
            metric: "peak_resident_memory_upper_bound_bytes",
            baseline: comparison.baseline.peakResidentMemory.validatedScoredBytes.map(Double.init),
            candidate: comparison.candidate.peakResidentMemory.validatedScoredBytes.map(
                Double.init),
            bound: thresholds.maxPeakFootprintRatio, lowerIsBetter: true,
            comparable: completedSameWork,
            outsideDetail:
                "the two passes did not complete the same parity scenarios, so their whole-process "
                + "maxima summarise different work")

        return Decision(
            accepted: blockers.isEmpty, blockers: blockers, deltas: deltas,
            declaredAsymmetries: comparison.declaredAsymmetries)
    }

    /// How far apart the two runtimes' rendered prompts were, as a ratio at
    /// least `1.0`, or `nil` when either side never reported a count.
    ///
    /// Symmetric on purpose: which runtime rendered the longer prompt does not
    /// change whether the two workloads were the same one.
    static func promptTokenSkew(
        baseline: ScenarioResult, candidate: ScenarioResult
    ) -> Double? {
        guard let baselineTokens = baseline.promptTokens,
            let candidateTokens = candidate.promptTokens,
            baselineTokens > 0, candidateTokens > 0
        else { return nil }
        let ratio = Double(candidateTokens) / Double(baselineTokens)
        return ratio >= 1 ? ratio : 1 / ratio
    }

    /// Cache reuse is a workload fact, not a latency inference. A one-sided
    /// hit and every unknown/malformed observation make the scenario
    /// non-comparable even when prompt-token counts are byte-for-byte equal.
    static func cacheComparabilityIssue(
        scenario: String,
        baseline: CacheReuseObservation,
        candidate: CacheReuseObservation
    ) -> String? {
        guard let baselineState = baseline.validatedState else {
            return
                "\(scenario)/cache_reuse baseline observation is malformed; unknown is not comparable"
        }
        guard let candidateState = candidate.validatedState else {
            return
                "\(scenario)/cache_reuse candidate observation is malformed; unknown is not comparable"
        }
        if baselineState == .unknown || candidateState == .unknown {
            return
                "\(scenario)/cache_reuse is unknown (baseline \(baselineState.rawValue), candidate "
                + "\(candidateState.rawValue)); unknown is not comparable"
        }
        if baselineState == .notApplicable || candidateState == .notApplicable {
            return baselineState == candidateState
                ? nil
                : "\(scenario)/cache_reuse applicability differs (baseline "
                    + "\(baselineState.rawValue), candidate \(candidateState.rawValue)); the "
                    + "scenario is not comparable"
        }
        if baselineState != candidateState {
            return
                "\(scenario)/cache_reuse is one-sided (baseline \(baselineState.rawValue), "
                + "candidate \(candidateState.rawValue)); the scenario is not comparable"
        }
        return nil
    }

    /// - Parameters:
    ///   - comparable: `false` when something outside this metric has already
    ///     established that the two numbers are not measuring the same thing.
    ///     The ratio is still reported — a reader wants to see it — but no
    ///     verdict is claimed for it, and the metric becomes a blocker. A
    ///     non-comparable metric is a species of unknown, and unknown has never
    ///     been within threshold here.
    private static func appendDelta(
        _ deltas: inout [MetricDelta],
        _ blockers: inout [String],
        scenario: String,
        metric: String,
        baseline: Double?,
        candidate: Double?,
        bound: Double,
        lowerIsBetter: Bool,
        comparable: Bool = true,
        outsideDetail: String? = nil
    ) {
        let admissible = lowerIsBetter ? "<= \(bound)" : ">= \(bound)"
        guard let baseline, let candidate else {
            blockers.append(
                "\(scenario)/\(metric) was not measured on "
                    + (baseline == nil ? "the baseline" : "the candidate")
                    + "; unknown is not within threshold")
            deltas.append(
                MetricDelta(
                    scenario: scenario, metric: metric, baseline: baseline, candidate: candidate,
                    ratio: nil, admissibleRatio: admissible, verdict: "unmeasured"))
            return
        }
        // A zero or negative baseline cannot produce a ratio. Reporting one
        // anyway would print `inf` and score it as a pass on the
        // higher-is-better axis.
        guard baseline > 0, candidate.isFinite, baseline.isFinite else {
            blockers.append(
                "\(scenario)/\(metric) has no usable ratio (baseline \(baseline), candidate "
                    + "\(candidate)); unknown is not within threshold")
            deltas.append(
                MetricDelta(
                    scenario: scenario, metric: metric, baseline: baseline, candidate: candidate,
                    ratio: nil, admissibleRatio: admissible, verdict: "unmeasured"))
            return
        }
        let ratio = candidate / baseline
        guard comparable else {
            blockers.append(
                "\(scenario)/\(metric) ratio \(ratio) is not comparable"
                    + (outsideDetail.map { ": \($0)" } ?? "")
                    + " (baseline \(baseline), candidate \(candidate))")
            deltas.append(
                MetricDelta(
                    scenario: scenario, metric: metric, baseline: baseline, candidate: candidate,
                    ratio: ratio, admissibleRatio: admissible, verdict: "non-comparable"))
            return
        }
        let within = lowerIsBetter ? ratio <= bound : ratio >= bound
        if !within {
            blockers.append(
                "\(scenario)/\(metric) ratio \(ratio) is outside the admissible band "
                    + "\(admissible) (baseline \(baseline), candidate \(candidate))"
                    + (outsideDetail.map { "; \($0)" } ?? ""))
        }
        deltas.append(
            MetricDelta(
                scenario: scenario, metric: metric, baseline: baseline, candidate: candidate,
                ratio: ratio, admissibleRatio: admissible,
                verdict: within ? "within" : "outside"))
    }
}
