import Darwin
import Foundation
import MLXSwiftRuntimeContract

/// The `benchmark-run` subcommand: launch, drive, measure, record and judge, in
/// one invocation of one binary.
///
/// This is the revision-4 answer to a finding review made three times, each
/// time against a construction that had raised the cost of the previous
/// forgery without removing it:
///
/// 1. A driver wrote a record and the gate checked the record against itself.
///    Review minted a self-consistent pair; `accepted=true`.
/// 2. The gate additionally checked the record against provenance the driver
///    recorded off its own launch. Review minted both; `accepted=true`.
/// 3. The gate observed each pass through `benchmark-attest open|close`, from
///    the kernel and over the wire. Review started two placeholder HTTP servers
///    that answered `GET /v1/models`, had the production commands attest them —
///    correctly, the processes were real — typed the measurements, and got
///    `accepted=true` in 7.2 seconds.
///
/// The common shape is a seam: some other program measured, and the gate was
/// asked to believe a document about it. Revision 4 closes the seam rather than
/// policing it. There is no `benchmark-attest` subcommand any more, so no
/// caller can direct this binary to attest a process it chose; the only code
/// that produces an attestation is the code below, about a process it spawned
/// itself, sealing the record it built from exchanges it performed.
/// `benchmark-compare` still exists for replaying an archived session, and it
/// cannot return an acceptance at all.
///
/// **What this still does not prove.** A modified build of this binary can
/// report anything it likes; nothing a program says about itself can survive
/// the program being changed. What has changed is the class of the attack: the
/// three findings above were all *ordinary use of shipped commands*, and that
/// is now closed. Making this gate accept fiction requires editing and
/// rebuilding it, and the acceptance suite mutates exactly that — see
/// `scripts/benchmark-gate-smoke.sh`, whose narrowing mutant N-P swaps the
/// scenario driver below for the placeholder path and is caught.
@MainActor
enum BenchmarkRunCommand {
    nonisolated static let name = "benchmark-run"

    /// Exit codes are distinct on purpose. A caller has to be able to tell "the
    /// candidate lost" from "the question was never asked", because only the
    /// second one means the benchmark is broken.
    enum ExitCode: Int32 {
        case accepted = 0
        case usage = 2
        case rejected = 3
        case inadmissible = 4
        case aborted = 5
    }

    enum RunError: Error, CustomStringConvertible {
        case unusableInput(String)
        case aborted(String)

        var description: String {
            switch self {
            case .unusableInput(let detail): return detail
            case .aborted(let detail): return detail
            }
        }
    }

    static let usage = """
        usage: mlx-swift-runtime-prototype benchmark-run \
        --config <model-harness.toml> --model <dir-or-gguf> --prompts <suite.json> \
        --thresholds <thresholds.json> --session <dir> --harness <model-harness> \
        --baseline-runtime <id> --baseline-profile <name> \
        --candidate-runtime <id> --candidate-profile <name> \
        [--candidate-model <dir-or-gguf>] [--equivalence <verdict.json>] \
        [--port <n>] [--python-bin <path>] [--candidate-binary <path>] \
        [--skip <scenario>]... [--baseline-declare <text>]... \
        [--candidate-declare <text>]... [--startup-timeout <s>] \
        [--request-timeout <s>] [--settle-seconds <s>]
        """

    /// Environment variables that configure a runtime's speculative decoding
    /// without appearing in any argv this gate records.
    ///
    /// `llama-server` reads all of these; `--help` for the pinned build lists
    /// the variable beside every speculative flag. They are refused rather than
    /// stripped because this process's environment is what it hands the
    /// launcher, so an inherited `LLAMA_ARG_SPEC_TYPE` would put the runtime
    /// under test into speculative decoding while the recorded launch showed
    /// nothing — the same "absent flag read as a policy" defect the KV bound
    /// was rebuilt to remove, one condition along.
    private static let speculativeEnvironmentPrefixes = ["LLAMA_ARG_SPEC", "LLAMA_ARG_DRAFT"]

    private static let repeatable: Set<String> = [
        "--skip", "--baseline-declare", "--candidate-declare",
    ]

    static func run(arguments: [String]) async -> Int32 {
        var single: [String: String] = [:]
        var multiple: [String: [String]] = [:]
        var known: Set<String> = [
            "--config", "--model", "--prompts", "--thresholds", "--session", "--harness",
            "--baseline-runtime", "--baseline-profile", "--candidate-runtime",
            "--candidate-profile", "--port", "--python-bin", "--candidate-binary",
            "--startup-timeout", "--request-timeout", "--settle-seconds",
            "--candidate-model", "--equivalence",
        ]
        known.formUnion(repeatable)
        var index = 0
        while index < arguments.count {
            let flag = arguments[index]
            guard known.contains(flag), index + 1 < arguments.count else {
                StandardOutput.shared.log("bad argument at \(flag.debugDescription)")
                StandardOutput.shared.log(usage)
                return ExitCode.usage.rawValue
            }
            if repeatable.contains(flag) {
                multiple[flag, default: []].append(arguments[index + 1])
            } else {
                guard single[flag] == nil else {
                    StandardOutput.shared.log("flag \(flag.debugDescription) was given twice")
                    return ExitCode.usage.rawValue
                }
                single[flag] = arguments[index + 1]
            }
            index += 2
        }
        let required = [
            "--config", "--model", "--prompts", "--thresholds", "--session", "--harness",
            "--baseline-runtime", "--baseline-profile", "--candidate-runtime",
            "--candidate-profile",
        ]
        for flag in required where single[flag] == nil {
            StandardOutput.shared.log("missing required flag \(flag.debugDescription)")
            StandardOutput.shared.log(usage)
            return ExitCode.usage.rawValue
        }

        do {
            return try await execute(single: single, multiple: multiple)
        } catch let error as RunError {
            StandardOutput.shared.log("\(error)")
            return ExitCode.aborted.rawValue
        } catch {
            StandardOutput.shared.log("\(error)")
            return ExitCode.aborted.rawValue
        }
    }

    // swiftlint:disable:next function_body_length
    private static func execute(
        single: [String: String], multiple: [String: [String]]
    ) async throws -> Int32 {
        let configPath = single["--config"]!
        let modelPath = single["--model"]!
        // Defaults to the baseline's artifact, so a same-format comparison is
        // spelled exactly as it was before G4 and pins byte identity exactly as
        // it did.
        let candidateModelPath = single["--candidate-model"] ?? modelPath
        let promptsPath = single["--prompts"]!
        let thresholdsPath = single["--thresholds"]!
        let sessionPath = single["--session"]!
        let harness = single["--harness"]!
        let port = single["--port"].flatMap(Int.init) ?? 18031
        let startupTimeout = single["--startup-timeout"].flatMap(Double.init) ?? 900
        let requestTimeout = single["--request-timeout"].flatMap(Double.init) ?? 2400
        let settleSeconds = single["--settle-seconds"].flatMap(Double.init) ?? 20
        let skip = Set(multiple["--skip"] ?? [])

        let suite = try BenchmarkScenarios.Suite(path: promptsPath)
        guard let thresholdData = try? Data(contentsOf: URL(fileURLWithPath: thresholdsPath)),
            let thresholds = try? JSONDecoder().decode(
                RuntimeBenchmark.Thresholds.self, from: thresholdData)
        else {
            throw RunError.unusableInput(
                "thresholds \(thresholdsPath.debugDescription) could not be read")
        }
        guard let gateDigest = GateBinary.digest() else {
            throw RunError.aborted(
                "this binary could not be digested, so it cannot attest to anything it observes")
        }
        guard let configDigest = fileDigest(configPath) else {
            throw RunError.unusableInput(
                "launcher config \(configPath.debugDescription) could not be read")
        }
        guard let promptSuiteDigest = fileDigest(promptsPath) else {
            throw RunError.unusableInput("prompt suite could not be digested")
        }

        let recordsDirectory = (sessionPath as NSString).appendingPathComponent("records")
        let attestDirectory = (sessionPath as NSString).appendingPathComponent("attest")
        let logsDirectory = (sessionPath as NSString).appendingPathComponent("logs")
        for directory in [recordsDirectory, attestDirectory, logsDirectory] {
            try? FileManager.default.createDirectory(
                atPath: directory, withIntermediateDirectories: true)
        }

        // Refused before anything is launched, and refused rather than scrubbed.
        // Silently unsetting it would make the gate's environment differ from
        // the operator's for reasons no record shows; refusing says which
        // variable, so the operator fixes the shell that would have changed the
        // decoding algorithm under test.
        for (name, value) in ProcessInfo.processInfo.environment
        where speculativeEnvironmentPrefixes.contains(where: name.hasPrefix) {
            throw RunError.unusableInput(
                "the environment this gate would hand the launcher sets \(name)="
                    + "\(value.debugDescription); that configures speculative decoding without "
                    + "appearing in any recorded argv, and speculation is not a condition the "
                    + "two runtimes can share -- unset it and run again")
        }

        // Read before the pins, because the artifact's quantization label may
        // have to come out of it, and read once for both passes: a cross-format
        // comparison rests on one verdict about both artifacts.
        let equivalence = equivalenceReading(path: single["--equivalence"])
        if case .unread(let path) = equivalence {
            throw RunError.unusableInput(
                "equivalence verdict \(path.debugDescription) could not be read or decoded; a "
                    + "verdict the gate cannot read is a failed read rather than an absence of "
                    + "one, and this run has no model of record")
        }
        // F1. The document read and decoded perfectly and is not one of the
        // equivalence decisions this repository took. Refused here, before any
        // launch, and refused for what it is: not unreadable, not absent, but
        // authored by the invocation that is asking to be believed.
        if case .untrusted(let path, let documentDigest) = equivalence {
            throw RunError.unusableInput(
                "equivalence verdict \(path.debugDescription) read cleanly at digest "
                    + "\(documentDigest.debugDescription) and is not an equivalence decision this "
                    + "repository took; admission is bound to the decisions compiled into this "
                    + "gate from versioned source, because a verdict this invocation supplies is "
                    + "one this invocation could have written -- add the decision to "
                    + "TrustedEquivalenceDecisions.shipped and rebuild, or pass the document that "
                    + "decision names")
        }
        let common = CommonPins(
            hostIdentity: try hostIdentity(),
            promptSuiteDigest: promptSuiteDigest,
            maxOutputTokens: BenchmarkScenarios.defaultMaxOutputTokens,
            equivalence: equivalence)
        let baselineModel = try modelPins(artifact: modelPath, equivalence: equivalence)
        let candidateModel =
            candidateModelPath == modelPath
            ? baselineModel : try modelPins(artifact: candidateModelPath, equivalence: equivalence)
        // Said here, at the production entry, before any weights are loaded.
        // `RuntimeBenchmark.admitModelIdentity` refuses the same pair after the
        // fact; saying it now costs an hour less.
        if baselineModel.digest != candidateModel.digest, equivalence.equivalence == nil {
            throw RunError.unusableInput(
                "the two passes would serve \(modelPath.debugDescription) and "
                    + "\(candidateModelPath.debugDescription), which are different weight "
                    + "artifacts; pass --equivalence with a verdict naming the upstream model "
                    + "they share, because two different files are comparable only under declared "
                    + "equivalence and never by default")
        }

        // Every non-equivalence the verdict declared, on both passes, before
        // whatever the caller declared. `RuntimeBenchmark.admitModelIdentity`
        // refuses a cross-format pair whose records do not carry all of them,
        // so this is what makes them travel rather than an act of politeness --
        // and putting them in here rather than at admission is what makes them
        // land in `decision.json` and in every report taken from it.
        // The union of what the document declared and what the trusted decision
        // requires. They coincide for a document that matches its anchor; the
        // anchor's list is taken as well so the three measured differences
        // travel even under a trust store whose document drifted, and
        // `RuntimeBenchmark.admitModelIdentity` demands both lists back out of
        // both records.
        var mandated = equivalence.equivalence?.declaredNonEquivalences ?? []
        if let documentDigest = equivalence.evidenceDigest,
            let anchor = TrustedEquivalenceDecisions.decision(documentDigest: documentDigest)
        {
            for entry in anchor.requiredNonEquivalences where !mandated.contains(entry) {
                mandated.append(entry)
            }
        }
        let fixedOrderLimitation =
            "pass order is fixed baseline then candidate: residual heat or unreleased host "
            + "pressure can disadvantage the candidate, while shared host cache state can "
            + "favour it; direction is indeterminate and the per-pass host load is diagnostic, "
            + "not an admission gate"
        let parityPolicyDirection =
            "parity policy is intentionally one-way: baseline success plus candidate failure "
            + "blocks, while the reverse does not; direction favours the incumbent, and this is "
            + "a migration acceptance policy rather than a measured performance metric"
        let mtpOffDirection =
            "MTP/speculative decoding is forced off for algorithmic parity because the MLX "
            + "incumbent drops the MTP head: direction is against llama.cpp, which cannot use "
            + "a product capability the incumbent lacks"
        let residentMemoryBoundDirection =
            "memory is scored as a conservative upper bound: Mach physical footprint plus the "
            + "upper edge of vmmap resident mapped-file bytes; double-counting can overstate a "
            + "runtime whose footprint already charges mapped pages, so residual direction is "
            + "runtime-dependent and the raw components are retained"
        let baselinePromptCachePolicy =
            "baseline cache policy: --prompt-cache-size 1 --prompt-cache-bytes 8GB; a reuse hit "
            + "can reduce repeated-prefix TTFT and retained cache can increase memory; direction "
            + "is unknown without an observed hit"
        let candidateSlotCachePolicy =
            "candidate cache policy: llama.cpp per-slot KV reuse; a reuse hit can reduce "
            + "repeated-prefix TTFT and retained slot state can increase memory; direction is "
            + "unknown without an observed hit"
        let cacheComparabilityRule =
            "cache comparability: a scenario with an observed reuse hit on only one runtime is "
            + "non-comparable and must be refused rather than scored"
        func declared(_ caller: [String]) -> [String] {
            var entries = mandated
            for limitation in [
                fixedOrderLimitation, parityPolicyDirection, mtpOffDirection,
                residentMemoryBoundDirection, baselinePromptCachePolicy,
                candidateSlotCachePolicy, cacheComparabilityRule,
            ]
            where !entries.contains(limitation) {
                entries.append(limitation)
            }
            for entry in caller where !entries.contains(entry) { entries.append(entry) }
            return entries
        }

        var passes: [String: PassOutcome] = [:]
        var order:
            [(
                role: String, runtime: String, profile: String, declare: [String],
                model: ModelPins
            )] = [
                (
                    "baseline", single["--baseline-runtime"]!, single["--baseline-profile"]!,
                    declared(multiple["--baseline-declare"] ?? []), baselineModel
                ),
                (
                    "candidate", single["--candidate-runtime"]!, single["--candidate-profile"]!,
                    declared(multiple["--candidate-declare"] ?? []), candidateModel
                ),
            ]
        guard order[0].runtime != order[1].runtime else {
            throw RunError.unusableInput(
                "both passes name runtime \(order[0].runtime.debugDescription); a runtime "
                    + "compared against itself cannot decide a migration")
        }

        for (offset, pass) in order.enumerated() {
            StandardOutput.shared.log("[\(pass.runtime)] starting pass")
            let outcome = try await drive(
                role: pass.role, runtime: pass.runtime, profile: pass.profile,
                assertedPython: single["--python-bin"],
                assertedCandidateBinary: single["--candidate-binary"], configPath: configPath,
                configDigest: configDigest, harness: harness, port: port, suite: suite,
                skip: skip, common: common, model: pass.model,
                declaredAsymmetries: pass.declare, gateDigest: gateDigest,
                startupTimeout: startupTimeout, requestTimeout: requestTimeout,
                logPath: (logsDirectory as NSString)
                    .appendingPathComponent("\(pass.runtime)-runtime.log"))
            passes[pass.role] = outcome
            try write(
                outcome.record,
                to: (recordsDirectory as NSString)
                    .appendingPathComponent("\(pass.runtime).json"))
            try write(
                outcome.attestation,
                to: (attestDirectory as NSString)
                    .appendingPathComponent(RuntimeAttestation.fileName(runtime: pass.runtime)))
            // Sequential with a gap, never concurrent. This host holds one copy
            // of a 28 GB model; two overlapping passes would be measuring each
            // other's memory pressure, and the comparison refuses a pair whose
            // intervals touch.
            if offset == 0, settleSeconds > 0 {
                StandardOutput.shared.log(
                    "settling \(settleSeconds)s before the next pass so the host releases the "
                        + "first runtime's pages")
                try? await Task.sleep(nanoseconds: UInt64(settleSeconds * 1_000_000_000))
            }
        }
        order.removeAll()

        guard let baseline = passes["baseline"], let candidate = passes["candidate"] else {
            throw RunError.aborted("a pass produced no outcome")
        }

        try writeSession(
            path: (sessionPath as NSString).appendingPathComponent("session.json"),
            baseline: baseline, candidate: candidate)

        let comparison: RuntimeBenchmark.Comparison
        do {
            comparison = try RuntimeBenchmark.admit(
                baseline: baseline.record, baselineAttestation: baseline.attestation,
                candidate: candidate.record, candidateAttestation: candidate.attestation,
                requiredScenarios: thresholds.paritySuccessScenarios + thresholds.scoredScenarios,
                gateBinaryDigest: gateDigest)
        } catch {
            StandardOutput.shared.log("\(error)")
            return ExitCode.inadmissible.rawValue
        }
        let decision = RuntimeBenchmark.decide(comparison: comparison, thresholds: thresholds)
        let encoder = JSONEncoder()
        encoder.outputFormatting = [.prettyPrinted, .sortedKeys]
        if let encoded = try? encoder.encode(decision) {
            try? encoded.write(
                to: URL(
                    fileURLWithPath: (sessionPath as NSString)
                        .appendingPathComponent("decision.json")))
            FileHandle.standardOutput.write(encoded)
            FileHandle.standardOutput.write(Data("\n".utf8))
        }
        return decision.accepted ? ExitCode.accepted.rawValue : ExitCode.rejected.rawValue
    }

    /// The pins that are the same for both passes by construction.
    ///
    /// The model left this struct at G4. It is per-pass now, because a
    /// cross-format comparison serves two different artifacts and pinning them
    /// jointly was exactly the assumption that made llama.cpp inadmissible.
    struct CommonPins {
        let hostIdentity: String
        let promptSuiteDigest: String
        let maxOutputTokens: Int
        /// The one equivalence verdict, read once, carried into both
        /// attestations so admission can require both sides to cite the same
        /// document by digest.
        let equivalence: ModelEquivalenceReading
    }

    /// One pass's weight artifact, as the gate read it.
    struct ModelPins {
        let path: String
        let digest: String
        let quantization: String
        /// ``RuntimeBenchmark/modelOfRecord(artifactDigest:observing:)`` over
        /// the two, computed here so the pin and the attestation the gate
        /// writes beside it cannot be built from different readings.
        let ofRecord: String
    }

    static func modelPins(
        artifact path: String, equivalence: ModelEquivalenceReading
    ) throws -> ModelPins {
        let digest = try modelDigest(artifact: path)
        return ModelPins(
            path: path, digest: digest,
            quantization: try quantizationLabel(
                artifact: path, equivalence: equivalence, digest: digest),
            ofRecord: RuntimeBenchmark.modelOfRecord(
                artifactDigest: digest, observing: equivalence))
    }

    struct PassOutcome {
        let record: RuntimeBenchmark.RunRecord
        let attestation: RuntimeAttestation
        let lifecycle: [String: Double?]
        let soak: [String: Double?]
        let warmupMemory: RuntimeMemoryPeak
        let soakMemory: RuntimeMemoryPeak
        /// Highest 1-minute host load average seen anywhere in this pass.
        ///
        /// Reported beside the decision, never scored. A pass measured while
        /// something else owned the machine is a measurement of that something
        /// else, and the first revision-4 attempt was exactly that.
        let hostLoadAverageMax: Double?
        let memorySamples: (successful: Int, readFailed: Int, malformed: Int)
        let harnessExitStatus: Int32?
    }

    /// One pass, end to end, inside this process.
    // swiftlint:disable:next function_body_length
    private static func drive(
        role: String,
        runtime: String,
        profile profileName: String,
        assertedPython: String?,
        assertedCandidateBinary: String?,
        configPath: String,
        configDigest: String,
        harness: String,
        port: Int,
        suite: BenchmarkScenarios.Suite,
        skip: Set<String>,
        common: CommonPins,
        model: ModelPins,
        declaredAsymmetries: [String],
        gateDigest: String,
        startupTimeout: TimeInterval,
        requestTimeout: TimeInterval,
        logPath: String
    ) async throws -> PassOutcome {
        let host = "127.0.0.1"
        let profile = try BenchmarkLaunchConfig.profile(
            named: profileName, in: configPath, host: host, port: port)
        let memoryAccounting = RuntimeMemoryAccounting.forExecutable(
            profile.executable, modelPath: model.path, launchArgv: profile.argv)
        guard profile.argv.contains(model.path) else {
            throw RunError.unusableInput(
                "profile \(profileName.debugDescription) does not pass "
                    + "\(model.path.debugDescription) to the runtime; the modelPath pin "
                    + "would not be bound to the process under test")
        }
        let harnessCommand = [
            harness, "run", profileName, "--host", host, "--port", String(port),
            "--config", configPath,
        ]
        let startedAt = Date().timeIntervalSince1970
        guard
            let launcher = SpawnedProcess.spawn(
                executable: harness, arguments: Array(harnessCommand.dropFirst()),
                standardOutputPath: logPath)
        else {
            throw RunError.aborted("could not spawn \(harnessCommand.joined(separator: " "))")
        }
        // Every refusal below owns this launch just as surely as the happy path
        // does. In particular, Python provenance is resolved late in the pass
        // and can throw after the runtime has served every scenario; without a
        // scope guard that error leaves model-harness and its model child in
        // their detached session after benchmark-run has already returned.
        defer { _ = launcher.terminate() }

        // The runtime under test, not the launcher that owns it.
        var runtimePID: pid_t = 0
        let childDeadline = Date().addingTimeInterval(60)
        while Date() < childDeadline {
            let children = SpawnedProcess.children(of: launcher.pid)
            if children.count == 1 {
                runtimePID = children[0]
                break
            }
            if children.count > 1 {
                _ = launcher.terminate()
                throw RunError.aborted(
                    "the launcher has \(children.count) children; the runtime under test cannot "
                        + "be identified")
            }
            if let status = launcher.exitStatusIfFinished() {
                throw RunError.aborted("the launcher exited with status \(status) before serving")
            }
            try? await Task.sleep(nanoseconds: 200_000_000)
        }
        guard runtimePID > 0 else {
            _ = launcher.terminate()
            throw RunError.aborted("the launcher never spawned a runtime child")
        }

        guard let launcherObservation = ProcessObservation.of(pid: Int(launcher.pid)),
            let observedHarness = canonicalPath(launcherObservation.executablePath),
            let requestedHarness = canonicalPath(harness), observedHarness == requestedHarness
        else {
            _ = launcher.terminate()
            throw RunError.aborted(
                "the model-harness revision cannot be attributed to the launcher process")
        }

        // Opened before readiness rather than after it, so the observation
        // covers the model load as well as the scenarios. A window opened after
        // warm-up would leave the most expensive part of the pass outside the
        // only observation anybody can check.
        guard let openingObservation = ProcessObservation.of(pid: Int(runtimePID)) else {
            _ = launcher.terminate()
            throw RunError.aborted("pid \(runtimePID) could not be observed from the kernel")
        }
        let openedAt = Date().timeIntervalSince1970

        let sampler = BenchmarkFootprintSampler(
            pid: Int(runtimePID), accounting: memoryAccounting)
        sampler.start()
        let session = BenchmarkHTTPDriver.session(requestTimeout: requestTimeout)
        let pass = BenchmarkPass(
            runtime: runtime, suite: suite, modelID: model.path,
            runtimeProcessID: Int(runtimePID),
            endpoint: URL(string: "http://\(host):\(port)/v1")!, requestTimeout: requestTimeout,
            session: session, sampler: sampler)

        var scenarios: [RuntimeBenchmark.ScenarioResult] = []
        do {
            try await awaitReady(
                pass: pass, launcher: launcher, modelID: model.path,
                startupTimeout: startupTimeout)
            for name in BenchmarkScenarios.order {
                // The only two reasons a scenario is not run: the caller asked
                // for it to be skipped, or the suite does not define it. A
                // scenario the suite *does* define is always driven, because
                // every field of it was understood before this loop existed --
                // the previous `kind` cast could drop a defined scenario here
                // without a word.
                guard !skip.contains(name), let scenario = suite.scenarios[name] else { continue }
                StandardOutput.shared.log("[\(runtime)] \(name) ...")
                // Opened here and nowhere else: the window has to start when the
                // scenario starts, or its peak is the previous scenario's.
                pass.beginScenarioWindow()
                let result = await BenchmarkScenarios.run(pass: pass, scenario: scenario)
                StandardOutput.shared.log(
                    "[\(runtime)] \(name): "
                        + (result.succeeded
                            ? "ok" : "FAILED \(result.failureMode ?? "no failure mode")"))
                scenarios.append(result)
                if launcher.exitStatusIfFinished() != nil {
                    // A scenario took the runtime down. Every scenario after it
                    // would report a connection error that says nothing about
                    // the runtime's behaviour, so the pass stops here and the
                    // record simply lacks them — which the comparison refuses
                    // rather than scores.
                    StandardOutput.shared.log(
                        "[\(runtime)] the runtime exited during \(name); stopping the pass")
                    break
                }
            }
        } catch {
            StandardOutput.shared.log("[\(runtime)] pass aborted: \(error)")
        }
        // Asked while the runtime is still up, because a question put after
        // teardown could only report that nothing answered — and asked even
        // when the pass aborted, so a runtime that stayed alive and served
        // nothing is refused for *that*, by name, rather than for an
        // observation the gate declined to close. Review's placeholder is
        // exactly this case: two processes that answered `/v1/models` happily
        // and never served a completion, and the refusal they deserve says so.
        let serving = await servingAnswer(pass: pass, expecting: model.path)

        // Resolve provenance from the process that actually served. Caller
        // paths are assertions only and cannot mint runtime evidence.
        guard let observation = ProcessObservation.settled(pid: Int(runtimePID)),
            observation.startUnixSeconds == openingObservation.startUnixSeconds
        else {
            _ = launcher.terminate()
            throw RunError.aborted(
                "the process that served could not be re-observed for runtime provenance")
        }
        guard let observedLaunchArgv = observation.arguments else {
            _ = launcher.terminate()
            throw RunError.aborted("the process that served exposed no observable runtime argv")
        }
        guard let observedExecutableDigest = fileDigest(observation.executablePath) else {
            _ = launcher.terminate()
            throw RunError.aborted(
                "could not digest \(observation.executablePath.debugDescription)")
        }
        var revisions: [String: String] = [:]
        let harnessVersion = try capture(
            executable: launcherObservation.executablePath, arguments: ["version"])
        guard harnessVersion.status == 0,
            let version = harnessVersion.standardOutput?
                .trimmingCharacters(in: .whitespacesAndNewlines), !version.isEmpty
        else {
            _ = launcher.terminate()
            throw RunError.aborted(
                "the observed model-harness process could not report its revision")
        }
        revisions["model_harness"] = version
        if role == "baseline" {
            revisions.merge(
                try pythonRevisions(
                    observation: observation, profile: profile,
                    assertedPython: assertedPython)
            ) { current, _ in current }
        } else if let assertedCandidateBinary {
            guard
                canonicalPath(assertedCandidateBinary) == canonicalPath(observation.executablePath),
                canonicalPath(profile.executable) == canonicalPath(observation.executablePath)
            else {
                _ = launcher.terminate()
                throw RunError.unusableInput(
                    "--candidate-binary is not the executable observed serving the candidate; "
                        + "refusing caller-supplied runtime provenance")
            }
            let observed = swiftRevisions(binary: observation.executablePath, model: model.path)
            guard !observed.isEmpty else {
                _ = launcher.terminate()
                throw RunError.unusableInput(
                    "the observed candidate process could not report its compiled revisions")
            }
            revisions.merge(observed) { current, _ in current }
        }

        // Re-read before close. A pid that was recycled is a different process,
        // and an observation spanning both attests to neither.
        let closingObservation = ProcessObservation.of(pid: Int(runtimePID))
        let stillTheSameProcess =
            closingObservation?.startUnixSeconds == observation.startUnixSeconds
            && closingObservation?.executablePath == observation.executablePath
        let closedAt = Date().timeIntervalSince1970
        sampler.stop()
        let processPeak = sampler.processPeakSoFar()
        let passLoadMax = sampler.passLoadAverageMax()
        let sampleCounts = sampler.sampleCounts()
        let harnessExit = launcher.terminate()
        session.invalidateAndCancel()
        let finishedAt = Date().timeIntervalSince1970

        // Built here rather than before the launch, because the KV bound is the
        // runtime's answer and not the launch's. `contextPolicy` reads the same
        // window the attestation below carries, which is what lets admission
        // re-derive this pin from a document the record did not author.
        let pins = RuntimeBenchmark.Pins(
            hostIdentity: common.hostIdentity, modelOfRecord: model.ofRecord,
            modelPath: model.path,
            modelDigest: model.digest, quantization: model.quantization,
            promptSuiteDigest: common.promptSuiteDigest,
            contextPolicy: RuntimeBenchmark.contextPolicy(
                observing: serving.contextWindow,
                generationConfiguration: serving.generationConfiguration),
            speculation: RuntimeBenchmark.speculationPolicy(
                derivedFrom: profile.argv, observing: serving.speculation),
            maxOutputTokens: common.maxOutputTokens,
            temperature: BenchmarkScenarios.temperature, topP: BenchmarkScenarios.topP,
            seed: BenchmarkScenarios.seed)

        let record = RuntimeBenchmark.RunRecord(
            runtime: runtime, revisions: revisions, command: harnessCommand,
            provenance: RuntimeBenchmark.LaunchProvenance(
                driverCommand: CommandLine.arguments,
                // The driver *is* the gate here, so the two digests are the same
                // file on purpose: one binary launched this runtime, drove it,
                // measured it and will judge it.
                driverDigest: gateDigest,
                harnessCommand: harnessCommand, configPath: configPath,
                configDigest: configDigest, profile: profileName,
                // Read off what the kernel was running, not off the profile's
                // `executable` field. Those are not the same file for a
                // script-launched runtime: the Python profile names a shebang
                // script and the process the kernel runs is the venv
                // interpreter, so a record that digested the script would
                // disagree with every observation of the process.
                launchExecutable: observation.executablePath,
                launchExecutableDigest: observedExecutableDigest,
                launchArgv: observedLaunchArgv, runtimeProcessID: Int(runtimePID)),
            pins: pins, startedAtUnixSeconds: startedAt, finishedAtUnixSeconds: finishedAt,
            peakPhysicalFootprintBytes: processPeak.peakSample?.machPhysicalFootprintBytes,
            peakResidentMemory: processPeak, scenarios: scenarios,
            declaredAsymmetries: declaredAsymmetries)

        // The seal, computed here and nowhere else, over the record this
        // invocation has just built from the exchanges it performed. A pass the
        // gate could not close honestly — a recycled pid, a runtime that never
        // answered what it was serving — seals nothing, and the comparison
        // refuses a record whose observation seals nothing rather than scoring
        // it. Absence and failure are different facts; this is the failure one.
        let sealed: String? =
            stillTheSameProcess && serving.modelID != nil
            ? RuntimeBenchmark.transcriptDigest(of: record) : nil
        let attestation = RuntimeAttestation(
            runtime: runtime, processID: Int(runtimePID),
            processStartUnixSeconds: observation.startUnixSeconds,
            observedExecutablePath: observation.executablePath,
            observedExecutableDigest: observedExecutableDigest, configPath: configPath,
            configDigest: configDigest, profile: profileName, openedAtUnixSeconds: openedAt,
            closedAtUnixSeconds: stillTheSameProcess ? closedAt : nil,
            servedModelID: serving.modelID, observedContextWindow: serving.contextWindow,
            observedGenerationConfiguration: serving.generationConfiguration,
            observedSpeculation: serving.speculation,
            // The verdict the gate read for itself, carried onto the document
            // the gate authored. `admitProvenance` re-derives `modelOfRecord`
            // from it and `admitModelIdentity` requires both sides to cite the
            // same bytes, so the record never gets to claim its own model
            // identity.
            observedModelEquivalence: common.equivalence,
            gateBinaryDigest: gateDigest,
            transcriptDigest: sealed)

        return PassOutcome(
            record: record, attestation: attestation, lifecycle: pass.lifecycle,
            soak: pass.soakDetail, warmupMemory: pass.warmupMemory,
            soakMemory: pass.soakMemory, hostLoadAverageMax: passLoadMax,
            memorySamples: sampleCounts, harnessExitStatus: harnessExit)
    }

    /// Wait until the *pinned* model answers a completion.
    ///
    /// Two separate facts are recorded: when `/v1/models` first listed the
    /// pinned model, and when that model first produced a token. For the Swift
    /// runtime they are nearly the same; for `mlx_lm.server` the first is about
    /// a second and the second includes the whole 28 GB load, because it loads
    /// on demand. A readiness check that stopped at the first 200 would report a
    /// cold runtime as ready.
    private static func awaitReady(
        pass: BenchmarkPass, launcher: SpawnedProcess, modelID: String,
        startupTimeout: TimeInterval
    ) async throws {
        let started = Date()
        let deadline = started.addingTimeInterval(startupTimeout)
        var listed = false
        while Date() < deadline {
            if let status = launcher.exitStatusIfFinished() {
                throw RunError.aborted(
                    "the runtime exited with status \(status) before readiness")
            }
            let answer = await pass.models(timeout: 5)
            if answer.status == 200,
                let document = try? JSONSerialization.jsonObject(with: answer.body)
                    as? [String: Any],
                let entries = document["data"] as? [[String: Any]],
                entries.contains(where: { ($0["id"] as? String) == modelID })
            {
                listed = true
                pass.recordLifecycle(
                    "models_listing_seconds", Date().timeIntervalSince(started))
                break
            }
            try? await Task.sleep(nanoseconds: 500_000_000)
        }
        guard listed else {
            throw RunError.aborted("the pinned model never appeared on /v1/models")
        }
        // The warm-up. Its cost is the model load for a lazily loading runtime,
        // so it is timed and reported rather than hidden, and it is excluded
        // from every scenario so no scenario carries a one-off load.
        let body = try BenchmarkScenarios.encode(
            BenchmarkScenarios.payload(
                model: modelID,
                messages: [["role": "user", "content": "Say OK."]], maxTokens: 4))
        let warmup = await pass.post(body: body)
        guard warmup.status == 200 else {
            throw RunError.aborted(
                "warm-up completion failed with HTTP \(warmup.status): "
                    + String(decoding: warmup.body.prefix(400), as: UTF8.self))
        }
        pass.recordLifecycle("first_completion_seconds", Date().timeIntervalSince(started))
        pass.recordWarmupMemory(pass.currentMemoryReading())
    }

    /// What one `GET /v1/models` against the live runtime said.
    struct ServingAnswer {
        /// The pinned model, if the runtime listed it. `nil` when the question
        /// could not be asked or the answer did not contain it.
        let modelID: String?
        /// The context bound that same answer named.
        let contextWindow: RuntimeContextWindow
        /// Effective prefill and reasoning parameters from the same answer.
        let generationConfiguration: RuntimeGenerationConfiguration
        /// Whether the process said it was speculating, from `GET /slots`.
        let speculation: RuntimeSpeculation
    }

    /// What the runtime says it is serving, and under what context bound, asked
    /// over the wire while the process is still up.
    ///
    /// One exchange produces both readings on purpose. A `nil` ``modelID``
    /// leaves the attestation unclosed and the pair refused, and the window
    /// that goes with it is ``RuntimeContextWindow/unread`` rather than
    /// ``RuntimeContextWindow/notReported``: an unread answer is not an answer,
    /// and a bound nobody could read is not a runtime without one.
    ///
    /// `meta.n_ctx` is `llama-server`'s spelling and is measured, not assumed —
    /// build `b10621-c1d0e7a00` reports 8192 under `--ctx-size 8192` and 32768
    /// with no context flag. Unbounded MLX launches emit no `meta` block and
    /// answer ``RuntimeContextWindow/notReported``. The bounded Python benchmark
    /// reports its active cache bound after construction, so its KV pin comes
    /// from this live answer rather than argv.
    ///
    /// A malformed field is not an absent field, and the two are separated in
    /// ``RuntimeContextWindow/read(fromModelsEntry:)`` — see there for the F2
    /// finding this shape exists to close.
    private static func servingAnswer(pass: BenchmarkPass, expecting modelID: String) async
        -> ServingAnswer
    {
        let answer = await pass.models(timeout: 30)
        guard answer.status == 200,
            let document = try? JSONSerialization.jsonObject(with: answer.body) as? [String: Any],
            let entries = document["data"] as? [[String: Any]],
            let entry = entries.first(where: { ($0["id"] as? String) == modelID })
        else {
            return ServingAnswer(
                modelID: nil, contextWindow: .unread,
                generationConfiguration: .unread, speculation: .unread)
        }
        let speculation = await speculationAnswer(pass: pass)
        // The reading itself lives in `RuntimeContextWindow` so it can be
        // attacked directly by the contract suite; this is its only production
        // call site, and it is inside the one exchange that also produces
        // `servedModelID`.
        return ServingAnswer(
            modelID: modelID, contextWindow: RuntimeContextWindow.read(fromModelsEntry: entry),
            generationConfiguration: RuntimeGenerationConfiguration.read(
                fromModelsEntry: entry),
            speculation: speculation)
    }

    /// Whether the runtime says it is speculating, asked over the wire while it
    /// is still up.
    ///
    /// `GET /slots` and deliberately not `GET /props`. Measured on
    /// `llama.cpp 0.3.0` build `b10621-c1d0e7a00` with a `Qwen2.5-0.5B-Instruct`
    /// Q8_0 fixture: launched with `--spec-type ngram-mod`, `/slots` reports
    /// `params.speculative` **true** and `/props` still reports
    /// `default_generation_settings.params["speculative.types"]` as `"none"`.
    /// `/props` describes the compiled default and does not move with the
    /// launch, so reading it would report a speculating server as quiet — a
    /// property inferred from a proxy signal, which a caller then acts on.
    ///
    /// A failed observation is not a negative observation, and the two are
    /// separated in ``RuntimeSpeculation/read(slotsStatus:body:)`` — see there
    /// for the F3 finding this shape exists to close.
    private static func speculationAnswer(pass: BenchmarkPass) async -> RuntimeSpeculation {
        let answer = await pass.slots(timeout: 30)
        // As with the context window: the reading lives in `RuntimeSpeculation`
        // where the contract suite can drive every branch of it, and this is
        // its only production call site.
        return RuntimeSpeculation.read(slotsStatus: answer.status, body: answer.body)
    }
}
