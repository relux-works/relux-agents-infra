import Foundation
import Testing

@testable import MLXSwiftRuntimeContract

@Suite("model directory admission")
struct ModelDirectoryCheckTests {
    static let path = "/models/Qwen"

    static func error(_ observation: ModelDirectoryObservation) -> ModelDirectoryError? {
        do {
            try ModelDirectoryCheck.admit(path: path, observation: observation)
            return nil
        } catch let error as ModelDirectoryError {
            return error
        } catch {
            return nil
        }
    }

    @Test("a complete directory is admitted")
    func admitsComplete() throws {
        try ModelDirectoryCheck.admit(path: Self.path, observation: .complete)
    }

    @Test("an absent directory is refused")
    func refusesMissing() {
        #expect(Self.error(.missing) == .missing(path: Self.path))
    }

    @Test("a path that is not a directory is refused")
    func refusesFile() {
        #expect(Self.error(.notADirectory) == .notADirectory(path: Self.path))
    }

    @Test("a directory missing required files is refused and names them")
    func refusesIncomplete() {
        #expect(
            Self.error(.incomplete(missing: ["tokenizer_config.json"]))
                == .incomplete(path: Self.path, missing: ["tokenizer_config.json"]))
    }

    @Test("an unreadable directory is refused as unreadable, not as absent")
    func doesNotDowngradeReadFailureToAbsence() {
        // A failed read is not evidence of absence. If this collapsed into
        // `.missing`, an EPERM on the model directory would be reported as
        // "no model configured" instead of "cannot inspect the model".
        let error = Self.error(.unreadable("EPERM"))
        #expect(error == .unreadable(path: Self.path, reason: "EPERM"))
        #expect(error != .missing(path: Self.path))
    }

    @Test("every non-complete observation stops the launch")
    func failsClosed() {
        let observations: [ModelDirectoryObservation] = [
            .missing, .notADirectory, .incomplete(missing: ["config.json"]), .unreadable("EIO"),
            .notRegularFiles(details: ["config.json is a directory"]),
        ]
        for observation in observations {
            #expect(
                Self.error(observation) != nil,
                "observation \(observation) must not be admitted")
        }
    }

    @Test("a real directory missing its weights is observed as incomplete")
    func observesMissingWeights() throws {
        let directory = FileManager.default.temporaryDirectory
            .appendingPathComponent("mlx-proto-\(UUID().uuidString)")
        try FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)
        defer { try? FileManager.default.removeItem(at: directory) }
        for name in ModelDirectoryCheck.requiredFiles {
            try Data("{}".utf8).write(to: directory.appendingPathComponent(name))
        }

        let observation = ModelDirectoryCheck.observe(path: directory.path)
        #expect(observation == .incomplete(missing: ["*.safetensors"]))

        try Data().write(to: directory.appendingPathComponent("model.safetensors"))
        #expect(ModelDirectoryCheck.observe(path: directory.path) == .complete)
    }

    /// The same forged-evidence shape as the Metal library gate: a name in a
    /// directory listing is not a file. `mkdir config.json` must not be able to
    /// mint its own admission.
    ///
    /// Production call site: `Main.main()` -> `ModelDirectoryCheck.observe` ->
    /// `admit`, before anything binds.
    @Test("a directory standing in for a required file is refused")
    func refusesForgedRequiredFile() throws {
        let directory = FileManager.default.temporaryDirectory
            .appendingPathComponent("mlx-proto-\(UUID().uuidString)")
        try FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)
        defer { try? FileManager.default.removeItem(at: directory) }
        try Data("{}".utf8).write(
            to: directory.appendingPathComponent("tokenizer_config.json"))
        try Data().write(to: directory.appendingPathComponent("model.safetensors"))
        try FileManager.default.createDirectory(
            at: directory.appendingPathComponent("config.json"),
            withIntermediateDirectories: true)

        let observation = ModelDirectoryCheck.observe(path: directory.path)
        #expect(observation == .notRegularFiles(details: ["config.json is a directory"]))
        #expect(Self.error(observation) != nil)
    }

    /// Same shape on the weights, which are matched by suffix rather than by
    /// name and so are the easier of the two to forge.
    @Test("a directory standing in for the weights is refused")
    func refusesForgedWeights() throws {
        let directory = FileManager.default.temporaryDirectory
            .appendingPathComponent("mlx-proto-\(UUID().uuidString)")
        try FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)
        defer { try? FileManager.default.removeItem(at: directory) }
        for name in ModelDirectoryCheck.requiredFiles {
            try Data("{}".utf8).write(to: directory.appendingPathComponent(name))
        }
        try FileManager.default.createDirectory(
            at: directory.appendingPathComponent("model.safetensors"),
            withIntermediateDirectories: true)

        let observation = ModelDirectoryCheck.observe(path: directory.path)
        #expect(observation == .notRegularFiles(details: ["model.safetensors is a directory"]))
        #expect(Self.error(observation) != nil)

        // A real shard alongside the forged one is enough: the gate wants one
        // loadable weight file, not the absence of odd entries.
        try Data().write(to: directory.appendingPathComponent("model-00001.safetensors"))
        #expect(ModelDirectoryCheck.observe(path: directory.path) == .complete)
    }

    /// A symlinked model tree — how a Hugging Face snapshot lays its files out —
    /// must still be admitted. The gate rejects non-files, not indirection.
    @Test("symlinked required files are admitted")
    func admitsSymlinkedFiles() throws {
        let root = FileManager.default.temporaryDirectory
            .appendingPathComponent("mlx-proto-\(UUID().uuidString)")
        let blobs = root.appendingPathComponent("blobs")
        let snapshot = root.appendingPathComponent("snapshot")
        try FileManager.default.createDirectory(at: blobs, withIntermediateDirectories: true)
        try FileManager.default.createDirectory(at: snapshot, withIntermediateDirectories: true)
        defer { try? FileManager.default.removeItem(at: root) }
        for name in ModelDirectoryCheck.requiredFiles + ["model.safetensors"] {
            let blob = blobs.appendingPathComponent("blob-\(name)")
            try Data("{}".utf8).write(to: blob)
            try FileManager.default.createSymbolicLink(
                at: snapshot.appendingPathComponent(name), withDestinationURL: blob)
        }
        #expect(ModelDirectoryCheck.observe(path: snapshot.path) == .complete)
    }

    /// A required entry that cannot be stat'ed was never proven missing. It
    /// must surface as `unreadable`, which throws its own error, rather than be
    /// laundered into `incomplete`.
    ///
    /// The directory itself stays listable, so this exercises the per-entry
    /// probe rather than the directory-level read that already had coverage:
    /// `config.json` is a symlink into a subdirectory with no search
    /// permission, so the name resolves fine and stat'ing it fails with EACCES.
    @Test("an unreadable required entry is not reported as missing")
    func unreadableEntryIsNotMissing() throws {
        try #require(getuid() != 0, "root can read anything, so this cannot be exercised")
        let root = FileManager.default.temporaryDirectory
            .appendingPathComponent("mlx-proto-\(UUID().uuidString)")
        let locked = root.appendingPathComponent("locked")
        try FileManager.default.createDirectory(at: locked, withIntermediateDirectories: true)
        defer {
            try? FileManager.default.setAttributes(
                [.posixPermissions: 0o755], ofItemAtPath: locked.path)
            try? FileManager.default.removeItem(at: root)
        }
        let blob = locked.appendingPathComponent("blob")
        try Data("{}".utf8).write(to: blob)
        try Data("{}".utf8).write(to: root.appendingPathComponent("tokenizer_config.json"))
        try Data().write(to: root.appendingPathComponent("model.safetensors"))
        try FileManager.default.createSymbolicLink(
            at: root.appendingPathComponent("config.json"), withDestinationURL: blob)
        try FileManager.default.setAttributes(
            [.posixPermissions: 0o000], ofItemAtPath: locked.path)

        // Precondition: the directory itself is still perfectly readable, so an
        // `unreadable` verdict here can only have come from the entry probe.
        #expect(
            Set(try FileManager.default.contentsOfDirectory(atPath: root.path))
                == ["locked", "tokenizer_config.json", "model.safetensors", "config.json"])

        let observation = ModelDirectoryCheck.observe(path: root.path)
        guard case .unreadable(let reason) = observation else {
            Issue.record("expected unreadable, got \(observation)")
            return
        }
        #expect(reason.contains("config.json"))
        #expect(Self.error(observation) != nil)
    }

    @Test("a real missing path is observed as missing")
    func observesMissingPath() {
        let path = FileManager.default.temporaryDirectory
            .appendingPathComponent("absent-\(UUID().uuidString)").path
        #expect(ModelDirectoryCheck.observe(path: path) == .missing)
    }
}
