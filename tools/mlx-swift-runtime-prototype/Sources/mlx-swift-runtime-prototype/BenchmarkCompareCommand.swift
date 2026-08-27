import Foundation
import MLXSwiftRuntimeContract

/// The `benchmark-compare` subcommand: replay an archived session's judgement.
///
/// It is no longer the production entry point for the migration decision, and
/// the demotion is the point. Three review rounds ended the same way: a caller
/// assembled files that this subcommand read, and it answered `accepted=true`
/// with exit 0. The third time the files were not even sloppy — two placeholder
/// HTTP servers really did run, this binary really did observe them, and the
/// measurements beside them were simply typed. Every clause the gate had was
/// satisfied by a pass in which nothing was served.
///
/// The decision now comes from `benchmark-run`, which launches both runtimes,
/// drives every scenario, records what it observed, seals it and judges it
/// inside one invocation. There is no argument to that command through which a
/// measurement can be supplied.
///
/// What is left here is replay: read an archived session, re-apply every
/// admission clause and re-score it, and report what it finds. **It cannot
/// return an acceptance.** Its best outcome is a reproduced rejection, exit 3;
/// a pair it will not admit is exit 4. That is not a cosmetic cap. These are
/// ordinary files owned by the ordinary user, so anything read off disk can in
/// principle be authored — and a command that could turn authored files into
/// `accepted=true, exit 0` is the exact bypass review found three times. A
/// replay that can only ever confirm or refuse cannot be that bypass, whatever
/// it is handed.
///
/// It stays in the binary because reading an archived decision back is worth
/// having: the session artifacts of this task are what a reviewer inspects, and
/// re-deriving the verdict from them independently of the report is a check the
/// report cannot perform on itself.
///
/// `--attestations` still has no default and no fallback. A directory that
/// holds no attestation for a record's runtime is an explicit refusal, not a
/// comparison made without one. A directory entry that exists and cannot be
/// read is reported as a read failure, which is a different fact from absence
/// and is never collapsed into it.
///
/// The subcommand loads no model and touches no GPU, so `Main` dispatches to it
/// before the model-directory and Metal shader library admissions — those gate
/// serving, and there is nothing here to serve.
enum BenchmarkCompareCommand {
    static let name = "benchmark-compare"

    /// Exit codes are distinct on purpose, and `0` is deliberately not one of
    /// them. A caller has to be able to tell "the candidate lost" from "the
    /// question was never asked", because only the second one means the
    /// benchmark is broken — and neither of them is an acceptance, because a
    /// replay of files cannot grant one.
    enum ExitCode: Int32 {
        case usage = 2
        case replayed = 3
        case inadmissible = 4
    }

    static let usage = """
        usage: mlx-swift-runtime-prototype benchmark-compare \
        --baseline <record.json> --candidate <record.json> \
        --thresholds <thresholds.json> --attestations <dir> [--output <decision.json>]

        Replays an archived session's judgement. Never returns an acceptance:
        exit 3 means the pair was admitted and re-scored, exit 4 means it was
        refused. Acceptance comes only from `benchmark-run`.
        """

    static func run(arguments: [String]) -> Int32 {
        var values: [String: String] = [:]
        let known: Set<String> = [
            "--baseline", "--candidate", "--thresholds", "--attestations", "--output",
        ]
        var index = 0
        while index < arguments.count {
            let flag = arguments[index]
            guard known.contains(flag) else {
                StandardOutput.shared.log("unknown flag \(flag.debugDescription)")
                StandardOutput.shared.log(usage)
                return ExitCode.usage.rawValue
            }
            guard index + 1 < arguments.count else {
                StandardOutput.shared.log("flag \(flag.debugDescription) requires a value")
                return ExitCode.usage.rawValue
            }
            guard values[flag] == nil else {
                StandardOutput.shared.log("flag \(flag.debugDescription) was given more than once")
                return ExitCode.usage.rawValue
            }
            values[flag] = arguments[index + 1]
            index += 2
        }
        guard let baselinePath = values["--baseline"],
            let candidatePath = values["--candidate"],
            let thresholdsPath = values["--thresholds"],
            let attestationDirectory = values["--attestations"]
        else {
            StandardOutput.shared.log(usage)
            return ExitCode.usage.rawValue
        }

        let baseline: RuntimeBenchmark.RunRecord
        let candidate: RuntimeBenchmark.RunRecord
        let thresholds: RuntimeBenchmark.Thresholds
        do {
            baseline = try RuntimeBenchmark.decodeRecord(
                path: baselinePath, data: readBytes(at: baselinePath))
            candidate = try RuntimeBenchmark.decodeRecord(
                path: candidatePath, data: readBytes(at: candidatePath))
            guard let thresholdData = readBytes(at: thresholdsPath) else {
                throw RuntimeBenchmark.AdmissionError.unreadable(
                    path: thresholdsPath, detail: "no bytes were read")
            }
            do {
                thresholds = try JSONDecoder().decode(
                    RuntimeBenchmark.Thresholds.self, from: thresholdData)
            } catch {
                throw RuntimeBenchmark.AdmissionError.malformed(
                    path: thresholdsPath, detail: String(describing: error))
            }
        } catch {
            StandardOutput.shared.log("\(error)")
            return ExitCode.inadmissible.rawValue
        }

        let baselineAttestation: RuntimeAttestation
        let candidateAttestation: RuntimeAttestation
        let gateDigest: String
        do {
            baselineAttestation = try attestation(
                runtime: baseline.runtime, directory: attestationDirectory)
            candidateAttestation = try attestation(
                runtime: candidate.runtime, directory: attestationDirectory)
            guard let digest = GateBinary.digest() else {
                throw RuntimeBenchmark.AdmissionError.unreadable(
                    path: GateBinary.path() ?? "<this binary>",
                    detail: "this binary could not be digested, so it cannot claim to be the "
                        + "one that observed these runs")
            }
            gateDigest = digest
        } catch {
            StandardOutput.shared.log("\(error)")
            return ExitCode.inadmissible.rawValue
        }

        let comparison: RuntimeBenchmark.Comparison
        do {
            comparison = try RuntimeBenchmark.admit(
                baseline: baseline, baselineAttestation: baselineAttestation,
                candidate: candidate, candidateAttestation: candidateAttestation,
                requiredScenarios: thresholds.paritySuccessScenarios
                    + thresholds.scoredScenarios,
                gateBinaryDigest: gateDigest)
        } catch {
            StandardOutput.shared.log("\(error)")
            return ExitCode.inadmissible.rawValue
        }

        let decision = RuntimeBenchmark.decide(
            comparison: comparison, thresholds: thresholds)

        let encoder = JSONEncoder()
        encoder.outputFormatting = [.prettyPrinted, .sortedKeys]
        guard let encoded = try? encoder.encode(decision) else {
            StandardOutput.shared.log("failed to encode the decision")
            return ExitCode.inadmissible.rawValue
        }
        if let outputPath = values["--output"] {
            do {
                try encoded.write(to: URL(fileURLWithPath: outputPath))
            } catch {
                StandardOutput.shared.log("failed to write \(outputPath): \(error)")
                return ExitCode.inadmissible.rawValue
            }
        }
        FileHandle.standardOutput.write(encoded)
        FileHandle.standardOutput.write(Data("\n".utf8))
        StandardOutput.shared.log(
            "replayed: accepted=\(decision.accepted), \(decision.blockers.count) blocker(s). "
                + "This is a replay of archived files and is never an acceptance; only "
                + "\(BenchmarkRunCommand.name), which launches and measures the runtimes it "
                + "judges, can return one.")
        return ExitCode.replayed.rawValue
    }

    /// The attestation this binary wrote for one runtime, or a refusal that says
    /// which of the two failures happened.
    ///
    /// A missing file and an unreadable one are reported separately on purpose.
    /// They are different facts about the world — nobody observed this pass
    /// versus somebody observed it and the evidence cannot be read — and a
    /// caller that collapsed them would let a truncated write, a permissions
    /// error or a half-flushed file read as "no attestation", which some future
    /// fallback would then be tempted to treat as a pass with nothing to check.
    private static func attestation(
        runtime: String, directory: String
    ) throws -> RuntimeAttestation {
        let path = (directory as NSString)
            .appendingPathComponent(RuntimeAttestation.fileName(runtime: runtime))
        guard FileManager.default.fileExists(atPath: path) else {
            throw RuntimeBenchmark.AdmissionError.attestationAbsent(
                runtime: runtime, directory: directory)
        }
        guard let data = readBytes(at: path) else {
            throw RuntimeBenchmark.AdmissionError.attestationUnreadable(
                path: path, detail: "the file exists and its bytes could not be read")
        }
        do {
            return try JSONDecoder().decode(RuntimeAttestation.self, from: data)
        } catch {
            throw RuntimeBenchmark.AdmissionError.attestationMalformed(
                path: path, detail: String(describing: error))
        }
    }

    /// `nil` only when the bytes could not be obtained at all. The caller turns
    /// that into an explicit `unreadable` refusal rather than into an empty
    /// document, so a permissions error or a truncated write can never be
    /// mistaken for a record that simply had nothing in it.
    private static func readBytes(at path: String) -> Data? {
        try? Data(contentsOf: URL(fileURLWithPath: path))
    }
}
