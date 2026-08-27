import Darwin
import Foundation
import MLXSwiftRuntimeContract

/// Everything `benchmark-run` reads off the host and the model to build its
/// pins, plus the two subprocess calls it makes to name the code that ran.
///
/// All of it is read in the gate's own process. The pins are what make two
/// measurements a comparison, so a pin computed by some other program is a pin
/// that can disagree with the launch it claims to describe — which is the
/// defect the whole `contextPolicy` derivation exists to prevent.
extension BenchmarkRunCommand {
    /// Hardware model, physical memory, OS build and architecture, joined.
    ///
    /// Two runs on different hosts are not a comparison, they are two
    /// measurements. Read through `sysctl` rather than by shelling out, so the
    /// string is a function of this process's view of the machine.
    static func hostIdentity() -> String {
        [
            sysctlString("hw.model") ?? "unknown-model",
            sysctlUInt64("hw.memsize").map(String.init) ?? "unknown-memsize",
            sysctlString("kern.osversion") ?? "unknown-build",
            sysctlString("hw.machine") ?? "unknown-arch",
        ].joined(separator: "/")
    }

    static func sysctlString(_ name: String) -> String? {
        var size = 0
        guard sysctlbyname(name, nil, &size, nil, 0) == 0, size > 0 else { return nil }
        var buffer = [CChar](repeating: 0, count: size)
        guard sysctlbyname(name, &buffer, &size, nil, 0) == 0 else { return nil }
        return String(cString: buffer)
    }

    static func sysctlUInt64(_ name: String) -> UInt64? {
        var value: UInt64 = 0
        var size = MemoryLayout<UInt64>.size
        guard sysctlbyname(name, &value, &size, nil, 0) == 0 else { return nil }
        return value
    }

    /// Digest over the files that decide what the weights are.
    ///
    /// `config.json` fixes the architecture and the quantization; the
    /// safetensors index fixes the shard set and every tensor's shape.
    /// Re-quantizing or re-sharding the directory changes one of them, so the
    /// same path with different contents is a pin mismatch rather than a silent
    /// comparison.
    static func modelDigest(directory: String) throws -> String {
        var accumulated = Data()
        for name in ["config.json", "model.safetensors.index.json"] {
            let path = (directory as NSString).appendingPathComponent(name)
            guard let bytes = try? Data(contentsOf: URL(fileURLWithPath: path)) else {
                throw RunError.unusableInput(
                    "\(path.debugDescription) could not be read; the model cannot be pinned")
            }
            accumulated.append(Data(name.utf8))
            accumulated.append(bytes)
        }
        return digest(of: accumulated)
    }

    static func quantizationLabel(directory: String) throws -> String {
        let path = (directory as NSString).appendingPathComponent("config.json")
        guard let bytes = try? Data(contentsOf: URL(fileURLWithPath: path)),
            let document = try? JSONSerialization.jsonObject(with: bytes) as? [String: Any],
            let quantization = document["quantization"] as? [String: Any],
            let bits = quantization["bits"] as? Int,
            let group = quantization["group_size"] as? Int,
            let mode = quantization["mode"] as? String
        else {
            throw RunError.unusableInput(
                "\(path.debugDescription) does not declare bits/group_size/mode; the "
                    + "quantization cannot be pinned and the run would not be comparable")
        }
        return "\(bits)bit/group\(group)/\(mode)"
    }

    struct CapturedProcess {
        let status: Int32
        let standardOutput: String?
    }

    /// Run a short-lived helper and read its stdout.
    ///
    /// Only ever used to *name* revisions, never to measure anything. Every
    /// number in a record comes from an exchange this process performed.
    static func capture(executable: String, arguments: [String]) throws -> CapturedProcess {
        let process = Process()
        process.executableURL = URL(fileURLWithPath: executable)
        process.arguments = arguments
        let pipe = Pipe()
        process.standardOutput = pipe
        process.standardError = pipe
        do {
            try process.run()
        } catch {
            throw RunError.unusableInput(
                "could not run \(executable.debugDescription): \(error)")
        }
        let data = pipe.fileHandleForReading.readDataToEndOfFile()
        process.waitUntilExit()
        return CapturedProcess(
            status: process.terminationStatus, standardOutput: String(data: data, encoding: .utf8))
    }

    /// Exact revisions of the incumbent's numerical stack, read from the
    /// interpreter that will serve the baseline.
    static func pythonRevisions(python: String) -> [String: String] {
        let script = """
            import json, platform
            from importlib.metadata import version, distribution
            d = distribution('mlx-lm')
            url = ''.join((d.read_text('direct_url.json') or '').splitlines())
            print(json.dumps({'mlx_lm': version('mlx-lm'), 'mlx': version('mlx'),
             'mlx_metal': version('mlx-metal'), 'transformers': version('transformers'),
             'python': platform.python_version(), 'mlx_lm_direct_url': url}))
            """
        guard let captured = try? capture(executable: python, arguments: ["-c", script]),
            let output = captured.standardOutput,
            let line = output.split(separator: "\n").last(where: { $0.hasPrefix("{") }),
            let data = line.data(using: .utf8),
            let document = try? JSONSerialization.jsonObject(with: data) as? [String: String]
        else { return [:] }
        return document
    }

    /// The compiled-in MLX revisions, read off the candidate binary's own
    /// preflight rather than off `Package.resolved`, so the record names the
    /// code that actually ran. Preflight binds no port and loads no weights.
    static func swiftRevisions(binary: String, model: String) -> [String: String] {
        guard
            let captured = try? capture(
                executable: binary, arguments: ["preflight", "--model", model]),
            let output = captured.standardOutput
        else { return [:] }
        var revisions: [String: String] = [:]
        for line in output.split(separator: "\n") {
            let trimmed = line.trimmingCharacters(in: .whitespaces)
            guard trimmed.hasPrefix("{"), let data = trimmed.data(using: .utf8),
                let event = try? JSONSerialization.jsonObject(with: data) as? [String: Any]
            else { continue }
            for key in ["mlx_swift", "mlx_swift_lm"] {
                if let value = event[key] as? String { revisions[key] = value }
            }
        }
        return revisions
    }

    static func write<T: Encodable>(_ value: T, to path: String) throws {
        let encoder = JSONEncoder()
        encoder.outputFormatting = [.prettyPrinted, .sortedKeys]
        guard let encoded = try? encoder.encode(value) else {
            throw RunError.aborted("failed to encode \(path.debugDescription)")
        }
        do {
            try encoded.write(to: URL(fileURLWithPath: path))
        } catch {
            throw RunError.aborted("failed to write \(path.debugDescription): \(error)")
        }
    }

    /// Everything measured that is reported beside the decision rather than
    /// scored by it: readiness timings, the soak's aggregate behaviour, sampler
    /// coverage and how each launcher exited.
    ///
    /// Separate from the record on purpose. The record carries what the
    /// comparison scores; a figure that is interesting but not comparable — the
    /// soak's aggregate output rate, which is not a decode rate — belongs where
    /// it cannot be mistaken for one.
    static func writeSession(path: String, baseline: PassOutcome, candidate: PassOutcome) throws {
        func block(_ outcome: PassOutcome) -> [String: Any] {
            var soak: [String: Any] = [:]
            for (key, value) in outcome.soak { soak[key] = value ?? NSNull() }
            var lifecycle: [String: Any] = [:]
            for (key, value) in outcome.lifecycle { lifecycle[key] = value ?? NSNull() }
            return [
                "runtime": outcome.record.runtime,
                "lifecycle": lifecycle,
                "soak": soak,
                "hostLoadAverageMax": outcome.hostLoadAverageMax ?? NSNull(),
                "footprintSamplesSuccessful": outcome.footprintSamples.successful,
                "footprintSamplesFailed": outcome.footprintSamples.failed,
                "harnessExitStatus": outcome.harnessExitStatus.map { Int($0) } ?? NSNull(),
                "transcriptSealed": outcome.attestation.transcriptDigest ?? NSNull(),
            ]
        }
        let document: [String: Any] = [
            "baseline": block(baseline),
            "candidate": block(candidate),
        ]
        guard
            let data = try? JSONSerialization.data(
                withJSONObject: document, options: [.prettyPrinted, .sortedKeys])
        else { throw RunError.aborted("failed to encode the session summary") }
        try data.write(to: URL(fileURLWithPath: path))
    }
}
