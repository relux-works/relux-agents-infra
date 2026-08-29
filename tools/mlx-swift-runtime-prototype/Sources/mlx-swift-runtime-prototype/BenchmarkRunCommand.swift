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
        --config <model-harness.toml> --model <dir> --prompts <suite.json> \
        --thresholds <thresholds.json> --session <dir> --harness <model-harness> \
        --baseline-runtime <id> --baseline-profile <name> \
        --candidate-runtime <id> --candidate-profile <name> \
        [--port <n>] [--python-bin <path>] [--candidate-binary <path>] \
        [--skip <scenario>]... [--baseline-declare <text>]... \
        [--candidate-declare <text>]... [--startup-timeout <s>] \
        [--request-timeout <s>] [--settle-seconds <s>]
        """

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

        let common = CommonPins(
            hostIdentity: hostIdentity(),
            modelPath: modelPath,
            modelDigest: try modelDigest(directory: modelPath),
            quantization: try quantizationLabel(directory: modelPath),
            promptSuiteDigest: promptSuiteDigest,
            maxOutputTokens: BenchmarkScenarios.defaultMaxOutputTokens)

        var sharedRevisions: [String: String] = [:]
        if let version = try? capture(executable: harness, arguments: ["version"]).standardOutput {
            sharedRevisions["model_harness"] = version.trimmingCharacters(
                in: .whitespacesAndNewlines)
        }

        var passes: [String: PassOutcome] = [:]
        var order: [(role: String, runtime: String, profile: String, declare: [String])] = [
            (
                "baseline", single["--baseline-runtime"]!, single["--baseline-profile"]!,
                multiple["--baseline-declare"] ?? []
            ),
            (
                "candidate", single["--candidate-runtime"]!, single["--candidate-profile"]!,
                multiple["--candidate-declare"] ?? []
            ),
        ]
        guard order[0].runtime != order[1].runtime else {
            throw RunError.unusableInput(
                "both passes name runtime \(order[0].runtime.debugDescription); a runtime "
                    + "compared against itself cannot decide a migration")
        }

        for (offset, pass) in order.enumerated() {
            var revisions = sharedRevisions
            if pass.role == "baseline", let python = single["--python-bin"] {
                revisions.merge(pythonRevisions(python: python)) { current, _ in current }
            }
            if pass.role == "candidate", let binary = single["--candidate-binary"] {
                revisions.merge(swiftRevisions(binary: binary, model: modelPath)) { current, _ in
                    current
                }
            }
            StandardOutput.shared.log("[\(pass.runtime)] starting pass")
            let outcome = try await drive(
                runtime: pass.runtime, profile: pass.profile, configPath: configPath,
                configDigest: configDigest, harness: harness, port: port, suite: suite,
                skip: skip, common: common, revisions: revisions,
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

    struct CommonPins {
        let hostIdentity: String
        let modelPath: String
        let modelDigest: String
        let quantization: String
        let promptSuiteDigest: String
        let maxOutputTokens: Int
    }

    struct PassOutcome {
        let record: RuntimeBenchmark.RunRecord
        let attestation: RuntimeAttestation
        let lifecycle: [String: Double?]
        let soak: [String: Double?]
        /// Highest 1-minute host load average seen anywhere in this pass.
        ///
        /// Reported beside the decision, never scored. A pass measured while
        /// something else owned the machine is a measurement of that something
        /// else, and the first revision-4 attempt was exactly that.
        let hostLoadAverageMax: Double?
        let footprintSamples: (successful: Int, failed: Int)
        let harnessExitStatus: Int32?
    }

    /// One pass, end to end, inside this process.
    // swiftlint:disable:next function_body_length
    private static func drive(
        runtime: String,
        profile profileName: String,
        configPath: String,
        configDigest: String,
        harness: String,
        port: Int,
        suite: BenchmarkScenarios.Suite,
        skip: Set<String>,
        common: CommonPins,
        revisions: [String: String],
        declaredAsymmetries: [String],
        gateDigest: String,
        startupTimeout: TimeInterval,
        requestTimeout: TimeInterval,
        logPath: String
    ) async throws -> PassOutcome {
        let host = "127.0.0.1"
        let profile = try BenchmarkLaunchConfig.profile(
            named: profileName, in: configPath, host: host, port: port)
        guard profile.argv.contains(common.modelPath) else {
            throw RunError.unusableInput(
                "profile \(profileName.debugDescription) does not pass "
                    + "\(common.modelPath.debugDescription) to the runtime; the modelPath pin "
                    + "would not be bound to the process under test")
        }
        let contextPolicy = RuntimeBenchmark.contextPolicy(derivedFrom: profile.argv)
        let pins = RuntimeBenchmark.Pins(
            hostIdentity: common.hostIdentity, modelPath: common.modelPath,
            modelDigest: common.modelDigest, quantization: common.quantization,
            promptSuiteDigest: common.promptSuiteDigest, contextPolicy: contextPolicy,
            maxOutputTokens: common.maxOutputTokens,
            temperature: BenchmarkScenarios.temperature, topP: BenchmarkScenarios.topP,
            seed: BenchmarkScenarios.seed)

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

        // Opened before readiness rather than after it, so the observation
        // covers the model load as well as the scenarios. A window opened after
        // warm-up would leave the most expensive part of the pass outside the
        // only observation anybody can check.
        guard let observation = ProcessObservation.of(pid: Int(runtimePID)) else {
            _ = launcher.terminate()
            throw RunError.aborted("pid \(runtimePID) could not be observed from the kernel")
        }
        guard let observedExecutableDigest = fileDigest(observation.executablePath) else {
            _ = launcher.terminate()
            throw RunError.aborted(
                "could not digest \(observation.executablePath.debugDescription)")
        }
        let openedAt = Date().timeIntervalSince1970

        let sampler = BenchmarkFootprintSampler(pid: Int(runtimePID))
        sampler.start()
        let session = BenchmarkHTTPDriver.session(requestTimeout: requestTimeout)
        let pass = BenchmarkPass(
            runtime: runtime, suite: suite, modelID: common.modelPath,
            runtimeProcessID: Int(runtimePID),
            endpoint: URL(string: "http://\(host):\(port)/v1")!, requestTimeout: requestTimeout,
            session: session, sampler: sampler)

        var scenarios: [RuntimeBenchmark.ScenarioResult] = []
        var servedModelID: String?
        do {
            try await awaitReady(
                pass: pass, launcher: launcher, modelID: common.modelPath,
                startupTimeout: startupTimeout)
            for name in BenchmarkScenarios.order {
                guard !skip.contains(name), let spec = suite.scenarios[name],
                    let kind = spec["kind"] as? String
                else { continue }
                StandardOutput.shared.log("[\(runtime)] \(name) ...")
                // Opened here and nowhere else: the window has to start when the
                // scenario starts, or its peak is the previous scenario's.
                pass.beginScenarioWindow()
                guard
                    let result = await BenchmarkScenarios.run(
                        kind: kind, pass: pass, name: name, spec: spec)
                else { continue }
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
        servedModelID = await servedModel(pass: pass, expecting: common.modelPath)

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
                launchArgv: profile.argv, runtimeProcessID: Int(runtimePID)),
            pins: pins, startedAtUnixSeconds: startedAt, finishedAtUnixSeconds: finishedAt,
            peakPhysicalFootprintBytes: processPeak, scenarios: scenarios,
            declaredAsymmetries: declaredAsymmetries)

        // The seal, computed here and nowhere else, over the record this
        // invocation has just built from the exchanges it performed. A pass the
        // gate could not close honestly — a recycled pid, a runtime that never
        // answered what it was serving — seals nothing, and the comparison
        // refuses a record whose observation seals nothing rather than scoring
        // it. Absence and failure are different facts; this is the failure one.
        let sealed: String? =
            stillTheSameProcess && servedModelID != nil
            ? RuntimeBenchmark.transcriptDigest(of: record) : nil
        let attestation = RuntimeAttestation(
            runtime: runtime, processID: Int(runtimePID),
            processStartUnixSeconds: observation.startUnixSeconds,
            observedExecutablePath: observation.executablePath,
            observedExecutableDigest: observedExecutableDigest, configPath: configPath,
            configDigest: configDigest, profile: profileName, openedAtUnixSeconds: openedAt,
            closedAtUnixSeconds: stillTheSameProcess ? closedAt : nil,
            servedModelID: servedModelID, gateBinaryDigest: gateDigest,
            transcriptDigest: sealed)

        return PassOutcome(
            record: record, attestation: attestation, lifecycle: pass.lifecycle,
            soak: pass.soakDetail, hostLoadAverageMax: passLoadMax,
            footprintSamples: sampleCounts, harnessExitStatus: harnessExit)
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
        pass.recordLifecycle(
            "footprint_after_warmup_bytes",
            physicalFootprintBytes(pid: pass.runtimeProcessID).map(Double.init))
    }

    /// The model the runtime says it is serving, asked over the wire.
    ///
    /// `nil` when the question could not be asked or the answer did not contain
    /// the pinned model. An unread answer is not an answer, and it is never
    /// read as one: a `nil` here leaves the attestation unclosed, which the
    /// comparison refuses.
    private static func servedModel(pass: BenchmarkPass, expecting modelID: String) async
        -> String?
    {
        let answer = await pass.models(timeout: 30)
        guard answer.status == 200,
            let document = try? JSONSerialization.jsonObject(with: answer.body) as? [String: Any],
            let entries = document["data"] as? [[String: Any]],
            entries.contains(where: { ($0["id"] as? String) == modelID })
        else { return nil }
        return modelID
    }
}
