import CryptoKit
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
    ///
    /// **A failed `sysctl` refuses rather than substituting a placeholder.**
    /// This used to join `"unknown-model"` and friends, which is the same shape
    /// as F2 one pin along: `hostIdentity` is compared for equality, and two
    /// records from two different machines that both failed to read `hw.model`
    /// carry the byte-identical string and compare *equal*. A placeholder is
    /// not a reading, and a pin built out of placeholders says two runs shared
    /// a host that neither of them could name.
    static func hostIdentity() throws -> String {
        var components: [String] = []
        for (name, value) in [
            ("hw.model", sysctlString("hw.model")),
            ("hw.memsize", sysctlUInt64("hw.memsize").map(String.init)),
            ("kern.osversion", sysctlString("kern.osversion")),
            ("hw.machine", sysctlString("hw.machine")),
        ] {
            guard let value, !value.isEmpty else {
                throw RunError.aborted(
                    "sysctl \(name.debugDescription) could not be read, so this gate cannot name "
                        + "the host it is measuring on; a placeholder there would compare equal "
                        + "to every other host that failed the same read")
            }
            components.append(value)
        }
        return components.joined(separator: "/")
    }

    static func sysctlString(_ name: String) -> String? {
        var size = 0
        guard sysctlbyname(name, nil, &size, nil, 0) == 0, size > 0 else { return nil }
        var buffer = [CChar](repeating: 0, count: size)
        guard sysctlbyname(name, &buffer, &size, nil, 0) == 0 else { return nil }
        let bytes = buffer.prefix { $0 != 0 }.map { UInt8(bitPattern: $0) }
        return String(decoding: bytes, as: UTF8.self)
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

    /// SHA-256 over a single weight file, read in bounded chunks.
    ///
    /// The whole file, not a header or a prefix. A GGUF has no `config.json`
    /// and no safetensors index to stand in for its contents, and a partial
    /// digest would let two differently quantized files share a pin — which is
    /// the one thing ``RuntimeBenchmark/Pins/modelDigest`` exists to prevent.
    /// The equivalence verdict binds to this number and to nothing else, so it
    /// has to cover the bytes the runtime loads.
    ///
    /// Streamed at 8 MiB because the artifact this was written for is 29 GB and
    /// reading it into memory beside a model that already does not fit twice is
    /// not an option.
    static func wholeFileDigest(of path: String) throws -> String {
        guard let handle = FileHandle(forReadingAtPath: path) else {
            throw RunError.unusableInput(
                "\(path.debugDescription) could not be opened; the model cannot be pinned")
        }
        defer { try? handle.close() }
        var hasher = SHA256()
        while true {
            let chunk: Data?
            do {
                chunk = try handle.read(upToCount: 8 * 1024 * 1024)
            } catch {
                throw RunError.unusableInput(
                    "\(path.debugDescription) could not be read to the end (\(error)); a partial "
                        + "digest is not a digest and the model cannot be pinned")
            }
            guard let chunk, !chunk.isEmpty else { break }
            hasher.update(data: chunk)
        }
        return hasher.finalize().map { String(format: "%02x", $0) }.joined()
    }

    /// The digest of whichever shape the artifact is.
    ///
    /// A weight *directory* is digested over the two files that decide what its
    /// weights are; a single *file* is digested whole. Anything else — a
    /// missing path, a symlink to nothing — is refused rather than defaulted,
    /// because a model that cannot be pinned cannot be compared.
    static func modelDigest(artifact path: String) throws -> String {
        var isDirectory: ObjCBool = false
        guard FileManager.default.fileExists(atPath: path, isDirectory: &isDirectory) else {
            throw RunError.unusableInput(
                "\(path.debugDescription) does not exist; the model cannot be pinned")
        }
        return isDirectory.boolValue
            ? try modelDigest(directory: path) : try wholeFileDigest(of: path)
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

    /// The quantization this artifact carries, however it can be read.
    ///
    /// A weight directory declares it in its own `config.json` and that is the
    /// only source used for one. A single weight file does not: a GGUF states
    /// its file type in a binary header this gate has no parser for, so the
    /// label comes from the equivalence verdict — **matched on the digest this
    /// gate computed**, never on the path. That is the whole reason a
    /// cross-format artifact needs a verdict at all: identity comes from a
    /// digest the gate took itself, and the verdict may then describe the file
    /// that digest identifies.
    ///
    /// A file with no *trusted* verdict covering its digest is refused here,
    /// before any launch: an untrusted reading carries no equivalence at all,
    /// so there is nothing to read a label out of. So is a directory whose verdict disagrees with its own
    /// `config.json` — the two are read independently and have to say the same
    /// thing.
    static func quantizationLabel(
        artifact path: String, equivalence reading: ModelEquivalenceReading, digest: String
    ) throws -> String {
        var isDirectory: ObjCBool = false
        _ = FileManager.default.fileExists(atPath: path, isDirectory: &isDirectory)
        let declared = reading.equivalence?.artifact(digest: digest)?.quantization
        guard isDirectory.boolValue else {
            guard let declared else {
                throw RunError.unusableInput(
                    "\(path.debugDescription) is a single weight file and no equivalence verdict "
                        + "names an artifact at digest \(digest.debugDescription); its "
                        + "quantization cannot be pinned, so the run would not be comparable")
            }
            return declared
        }
        let fromConfig = try quantizationLabel(directory: path)
        if let declared, declared != fromConfig {
            throw RunError.unusableInput(
                "\(path.debugDescription) declares quantization "
                    + "\(fromConfig.debugDescription) and the equivalence verdict names "
                    + "\(declared.debugDescription) for the same digest; the verdict and the "
                    + "weights are not describing the same artifact")
        }
        return fromConfig
    }

    /// Read an equivalence verdict the caller named, digest the bytes it was
    /// read from, and decide whether that digest names a decision this
    /// repository took.
    ///
    /// Four outcomes, and they are the four cases of
    /// ``ModelEquivalenceReading``. No path given is
    /// ``ModelEquivalenceReading/noneDeclared``; a path that will not read or
    /// decode is ``ModelEquivalenceReading/unread(path:)``; a path that reads
    /// and decodes but whose SHA-256 appears in no entry of
    /// ``TrustedEquivalenceDecisions/shipped`` is
    /// ``ModelEquivalenceReading/untrusted(path:digest:)``; only a document the
    /// trust store already names is ``ModelEquivalenceReading/read(_:digest:)``.
    ///
    /// **The trust lookup is why this function exists in this shape.** Its
    /// previous version read and hashed whatever JSON the caller pointed
    /// `--equivalence` at, and review minted a well-shaped verdict naming an
    /// arbitrary source of record, the two artifact digests the gate had itself
    /// computed, `comparable`, and one generic note — and the shipped
    /// `benchmark-run` accepted the pair with exit 0. Every step of that read
    /// was correct and none of it authenticated anything: a digest over
    /// attacker-authored bytes proves only that they did not change between the
    /// read and the seal. The caller still supplies the document, because its
    /// contents have to travel into both records; what the caller cannot do is
    /// decide that it counts.
    static func equivalenceReading(path: String?) -> ModelEquivalenceReading {
        guard let path else { return .noneDeclared }
        guard let bytes = try? Data(contentsOf: URL(fileURLWithPath: path)),
            let decoded = try? JSONDecoder().decode(ModelEquivalence.self, from: bytes)
        else { return .unread(path: path) }
        let documentDigest = digest(of: bytes)
        guard TrustedEquivalenceDecisions.decision(documentDigest: documentDigest) != nil else {
            return .untrusted(path: path, digest: documentDigest)
        }
        return .read(decoded, digest: documentDigest)
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

    static func canonicalPath(_ path: String) -> String? {
        guard FileManager.default.fileExists(atPath: path) else { return nil }
        return URL(fileURLWithPath: path).resolvingSymlinksInPath().standardized.path
    }

    /// Resolve the executable image a Python interpreter path becomes after
    /// macOS launcher/framework handoff, using a live child rather than path
    /// spelling. `sys.executable` is not enough on Xcode Python: it names the
    /// framework binary while `proc_pidpath` names `Python.app`.
    static func observedInterpreterExecutable(_ python: String) throws -> String? {
        let process = Process()
        process.executableURL = URL(fileURLWithPath: python)
        process.arguments = ["-c", "import time; time.sleep(10)"]
        try process.run()
        defer {
            process.terminate()
            process.waitUntilExit()
        }
        return ProcessObservation.settled(pid: Int(process.processIdentifier)).flatMap {
            canonicalPath($0.executablePath)
        }
    }

    /// Exact revisions of the incumbent's numerical stack, derived from the
    /// Python process that actually served the pass.
    ///
    /// The caller's `--python-bin` is only an assertion. The authority is the
    /// observed process argv, the package-owned `mlx_lm.server` entry point it
    /// executed, that entry point's shebang, and the interpreter selected by
    /// the shebang. Any missing or contradictory link refuses the run.
    static func pythonRevisions(
        observation: ProcessObservation.Reading,
        profile: BenchmarkLaunchConfig.Profile,
        assertedPython: String?
    ) throws -> [String: String] {
        guard let profileExecutable = canonicalPath(profile.executable) else {
            throw RunError.unusableInput(
                "baseline profile executable \(profile.executable.debugDescription) is unreadable")
        }
        guard let arguments = observation.arguments else {
            throw RunError.unusableInput(
                "the observed baseline process argv could not be read; runtime revision is unknown")
        }
        // Darwin keeps the interpreter image at argv[0] after a shebang exec
        // and puts the executed script in argv[1]. Require that exact slot so
        // a decoy cannot smuggle the expected path as an unused later argument.
        let observedEntryPoint = arguments.dropFirst().first
        guard observedEntryPoint.flatMap(canonicalPath) == profileExecutable else {
            throw RunError.unusableInput(
                "observed baseline pid \(observation.executablePath.debugDescription) did not "
                    + "execute the configured entry point \(profile.executable.debugDescription); "
                    + "observed argv[1] was \((observedEntryPoint ?? "<missing>").debugDescription); "
                    + "runtime revision cannot be attributed to the process that served")
        }
        guard
            let entryPointText = try? String(
                contentsOfFile: profileExecutable, encoding: .utf8),
            let firstLine = entryPointText.split(separator: "\n", maxSplits: 1).first,
            firstLine.hasPrefix("#!")
        else {
            throw RunError.unusableInput(
                "baseline profile executable \(profile.executable.debugDescription) is not a "
                    + "Python console-script entry point")
        }
        let shebang = String(firstLine.dropFirst(2))
            .trimmingCharacters(in: .whitespacesAndNewlines)
        guard !shebang.contains(" "), let interpreter = canonicalPath(shebang),
            let observedInterpreter = canonicalPath(observation.executablePath)
        else {
            throw RunError.unusableInput(
                "baseline entry point has no readable interpreter; runtime revision is unknown")
        }
        let interpreterImage = try observedInterpreterExecutable(shebang)
        guard interpreterImage == observedInterpreter
        else {
            throw RunError.unusableInput(
                "baseline entry point interpreter does not match the executable observed for "
                    + "the process that served (entry point becomes: "
                    + "\((interpreterImage ?? shebang).debugDescription), observed: "
                    + "\(observation.executablePath.debugDescription)); runtime revision is unknown"
            )
        }
        if let assertedPython {
            guard let assertion = canonicalPath(assertedPython), assertion == interpreter else {
                throw RunError.unusableInput(
                    "--python-bin \(assertedPython.debugDescription) is not the interpreter "
                        + "behind the observed baseline process; refusing caller-supplied "
                        + "runtime provenance")
            }
        }

        let script = """
            import base64, hashlib, json, pathlib, platform, re
            from importlib.metadata import distribution, version

            entry = pathlib.Path(__import__('sys').argv[1]).resolve(strict=True)
            d = distribution('mlx-lm')
            points = [e for e in d.entry_points
                      if e.group == 'console_scripts' and e.name == 'mlx_lm.server']
            if len(points) != 1 or points[0].value != 'mlx_lm.server:main':
                raise RuntimeError('mlx-lm does not own the expected server entry point')
            owned = []
            for item in d.files or ():
                target = pathlib.Path(d.locate_file(item)).resolve(strict=True)
                if target == entry:
                    owned.append(item)
                if item.hash is None:
                    continue
                algorithm, expected = item.hash.mode, item.hash.value
                digest = hashlib.new(algorithm, target.read_bytes()).digest()
                actual = base64.urlsafe_b64encode(digest).decode().rstrip('=')
                if actual != expected:
                    raise RuntimeError(f'installed mlx-lm file differs from RECORD: {item}')
            if len(owned) != 1 or owned[0].hash is None:
                raise RuntimeError('observed server entry point is not hash-owned by mlx-lm')
            direct = json.loads(d.read_text('direct_url.json') or '')
            vcs = direct.get('vcs_info') or {}
            commit = vcs.get('commit_id')
            requested = vcs.get('requested_revision')
            if vcs.get('vcs') != 'git' or commit != requested or not isinstance(commit, str) \
                    or re.fullmatch(r'[0-9a-f]{40}', commit) is None:
                raise RuntimeError('mlx-lm direct_url is not one immutable git revision')
            url = json.dumps(direct, sort_keys=True, separators=(',', ':'))
            print(json.dumps({'mlx_lm': version('mlx-lm'), 'mlx': version('mlx'),
             'mlx_metal': version('mlx-metal'), 'transformers': version('transformers'),
             'python': platform.python_version(), 'mlx_lm_direct_url': url,
             'mlx_lm_commit': commit}))
            """
        let captured = try capture(
            executable: shebang, arguments: ["-c", script, profileExecutable])
        guard captured.status == 0,
            let output = captured.standardOutput,
            let line = output.split(separator: "\n").last(where: { $0.hasPrefix("{") }),
            let data = line.data(using: .utf8),
            let document = try? JSONSerialization.jsonObject(with: data) as? [String: String]
        else {
            let detail = captured.standardOutput?.suffix(800) ?? "no output"
            throw RunError.unusableInput(
                "the observed baseline process could not attest its installed mlx-lm revision: "
                    + detail)
        }
        return document
    }

    /// The compiled-in MLX revisions, read off the candidate binary's own
    /// preflight rather than off `Package.resolved`, so the record names the
    /// code that actually ran. Preflight binds no port and loads no weights.
    ///
    /// Empty means the preflight could not be run or said nothing this gate
    /// recognised, and `execute` refuses on that for the same reason
    /// ``pythonRevisions(python:)`` is refused: `--candidate-binary` was given.
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
        func memoryObject(_ peak: RuntimeMemoryPeak) throws -> Any {
            let data = try JSONEncoder().encode(peak)
            guard let object = try? JSONSerialization.jsonObject(with: data) else {
                throw RunError.aborted("failed to encode a session memory reading")
            }
            return object
        }
        func block(_ outcome: PassOutcome) throws -> [String: Any] {
            var soak: [String: Any] = [:]
            for (key, value) in outcome.soak { soak[key] = value ?? NSNull() }
            var lifecycle: [String: Any] = [:]
            for (key, value) in outcome.lifecycle { lifecycle[key] = value ?? NSNull() }
            return [
                "runtime": outcome.record.runtime,
                "lifecycle": lifecycle,
                "soak": soak,
                "warmupMemory": try memoryObject(outcome.warmupMemory),
                "soakMemory": try memoryObject(outcome.soakMemory),
                "hostLoadAverageMax": outcome.hostLoadAverageMax ?? NSNull(),
                "memorySamplesSuccessful": outcome.memorySamples.successful,
                "memorySamplesReadFailed": outcome.memorySamples.readFailed,
                "memorySamplesMalformed": outcome.memorySamples.malformed,
                "harnessExitStatus": outcome.harnessExitStatus.map { Int($0) } ?? NSNull(),
                "transcriptSealed": outcome.attestation.transcriptDigest ?? NSNull(),
            ]
        }
        let document: [String: Any] = [
            "baseline": try block(baseline),
            "candidate": try block(candidate),
        ]
        guard
            let data = try? JSONSerialization.data(
                withJSONObject: document, options: [.prettyPrinted, .sortedKeys])
        else { throw RunError.aborted("failed to encode the session summary") }
        try data.write(to: URL(fileURLWithPath: path))
    }
}
