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
        /// Absolute path to the model directory both runtimes were pointed at.
        public let modelPath: String
        /// Digest over the model's `config.json` and safetensors index, so a
        /// re-quantized or re-sharded directory at the same path is a mismatch.
        public let modelDigest: String
        /// Quantization as the model's own config declares it, for example
        /// `8bit/group64/affine`.
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
            modelPath: String,
            modelDigest: String,
            quantization: String,
            promptSuiteDigest: String,
            contextPolicy: String,
            maxOutputTokens: Int,
            temperature: Double,
            topP: Double,
            seed: Int
        ) {
            self.hostIdentity = hostIdentity
            self.modelPath = modelPath
            self.modelDigest = modelDigest
            self.quantization = quantization
            self.promptSuiteDigest = promptSuiteDigest
            self.contextPolicy = contextPolicy
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
        func firstMismatch(against other: Pins) -> (field: String, mine: String, theirs: String)? {
            let fields: [(String, String, String)] = [
                ("hostIdentity", hostIdentity, other.hostIdentity),
                ("modelPath", modelPath, other.modelPath),
                ("modelDigest", modelDigest, other.modelDigest),
                ("quantization", quantization, other.quantization),
                ("promptSuiteDigest", promptSuiteDigest, other.promptSuiteDigest),
                ("contextPolicy", contextPolicy, other.contextPolicy),
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
        /// Peak physical footprint **within this scenario's window only**.
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
            hostLoadAverageMax: Double? = nil,
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
            self.hostLoadAverageMax = hostLoadAverageMax
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
        /// Whole-process peak, sampled from the Mach physical footprint rather
        /// than from `ps` RSS.
        ///
        /// Resident size is not a usable size for an MLX process: three
        /// identical loads of this model reported 2 650, 10 774 and 14 056 MiB
        /// resident while the physical footprint stayed within 16 MiB and MLX's
        /// own active-bytes figure was byte-identical.
        public let peakPhysicalFootprintBytes: Int?
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
    /// * the KV bound. `mlx_lm.server` has no flag for one at all and is always
    ///   unbounded; this runtime takes `--max-kv-size`. Absent on both sides
    ///   means the same thing, so absence is a legitimate `unbounded` reading.
    /// * the prefill chunk. `mlx_lm.server` defaults `--prefill-step-size` to
    ///   `2048`; `MLXLMCommon.GenerateParameters` defaults `prefillStepSize` to
    ///   `512`. Absence here does **not** mean the same thing on both sides, so
    ///   it is reported as `unpinned` and refused rather than read as a value.
    ///   Measuring 512-token chunks against 2048-token chunks and calling the
    ///   difference a runtime difference is exactly the comparison this pin
    ///   exists to prevent.
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
    public static func contextPolicy(derivedFrom argv: [String]) -> String {
        func value(of flag: String) -> String? {
            for (index, token) in argv.enumerated() {
                if token == flag, index + 1 < argv.count { return argv[index + 1] }
                if token.hasPrefix(flag + "=") { return String(token.dropFirst(flag.count + 1)) }
            }
            return nil
        }
        // Spelled two different ways by the two runtimes and meaning the same
        // thing, so the derivation reads both and reports the value rather
        // than the spelling. `mlx_lm.server` takes a JSON blob of chat-template
        // kwargs; this runtime takes the one kwarg it supports as a flag.
        func reasoningEffort() -> String {
            if let effort = value(of: "--reasoning-effort") { return effort }
            guard let raw = value(of: "--chat-template-args"),
                let data = raw.data(using: .utf8),
                let object = try? JSONSerialization.jsonObject(with: data),
                let mapping = object as? [String: Any],
                let effort = mapping["reasoning_effort"] as? String
            else { return "unpinned" }
            return effort
        }
        let kv = value(of: "--max-kv-size").map { "max-kv-size=\($0)" } ?? "unbounded"
        let prefill = value(of: "--prefill-step-size") ?? "unpinned"
        return "kv=\(kv);prefill-step=\(prefill);reasoning=\(reasoningEffort())"
    }

    /// Conditions the derived policy must not leave to a runtime default.
    static let unpinnableConditions = ["prefill-step=unpinned", "reasoning=unpinned"]

    private static func isSHA256(_ value: String) -> Bool {
        value.count == 64 && value.allSatisfy { $0.isHexDigit && !$0.isUppercase }
    }

    /// Refuse a record that is not tied to a run it could have come from.
    ///
    /// Every clause here is a cross-check between two things the record says,
    /// not a re-execution: the gate cannot prove a benchmark happened, but it
    /// can refuse a document whose declared conditions contradict the launch it
    /// reports, or that reports no launch at all.
    static func admitProvenance(_ record: RunRecord) throws {
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
        let derived = contextPolicy(derivedFrom: provenance.launchArgv)
        guard record.pins.contextPolicy == derived else {
            throw AdmissionError.contextPolicyNotDerived(
                runtime: record.runtime, declared: record.pins.contextPolicy, derived: derived)
        }
        for condition in unpinnableConditions where derived.contains(condition) {
            throw AdmissionError.unpinnedLaunchCondition(
                runtime: record.runtime, condition: condition)
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
        gateBinaryDigest: String
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
        try admitProvenance(baseline)
        try admitProvenance(candidate)
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
            lines.append("hostLoad=\(number(scenario.hostLoadAverageMax))")
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
            let comparable = skew.map { $0 <= skewBound } ?? false
            if let skew {
                if !comparable {
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
                        verdict: comparable ? "within" : "outside"))
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
                &deltas, &blockers, scenario: name, metric: "peak_physical_footprint_bytes",
                baseline: baseline.peakPhysicalFootprintBytes.map(Double.init),
                candidate: candidate.peakPhysicalFootprintBytes.map(Double.init),
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
            &deltas, &blockers, scenario: "process", metric: "peak_physical_footprint_bytes",
            baseline: comparison.baseline.peakPhysicalFootprintBytes.map(Double.init),
            candidate: comparison.candidate.peakPhysicalFootprintBytes.map(Double.init),
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
