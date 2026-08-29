import Foundation
import Testing

@testable import MLXSwiftRuntimeContract

@Suite("metal shader library admission")
struct MetalShaderLibraryCheckTests {

    // MARK: - policy

    @Test("a root holding the library admits the launch")
    func presentAdmits() throws {
        let observation = MetalShaderLibraryCheck.classify([
            MetalShaderRootFinding(root: "/a", outcome: .inspected),
            MetalShaderRootFinding(root: "/b", outcome: .containsLibrary(path: "/b/lib")),
        ])
        #expect(observation == .present(path: "/b/lib"))
        try MetalShaderLibraryCheck.admit(observation)
    }

    /// The gate's whole purpose: a `swift build` product, where every search
    /// root is readable and none carries the shader bundle, must be refused
    /// before the port binds — not left to abort mid-load inside MLX's C++.
    @Test("every root inspected and none holding the library is refused")
    func absentRefuses() {
        let observation = MetalShaderLibraryCheck.classify([
            MetalShaderRootFinding(root: "/a", outcome: .inspected),
            MetalShaderRootFinding(root: "/b", outcome: .notPresent),
        ])
        #expect(observation == .absent(searched: ["/a", "/b"], rejected: []))
        #expect(throws: MetalShaderLibraryError.absent(searched: ["/a", "/b"], rejected: [])) {
            try MetalShaderLibraryCheck.admit(observation)
        }
    }

    /// An absence and a failure to read are different facts. A root we could not
    /// inspect is not evidence that the library is missing from it, so it must
    /// suppress the `absent` verdict rather than count towards it.
    @Test("an unreadable root suppresses the absent verdict")
    func unreadableIsNotAbsence() throws {
        let observation = MetalShaderLibraryCheck.classify([
            MetalShaderRootFinding(root: "/a", outcome: .inspected),
            MetalShaderRootFinding(root: "/b", outcome: .unreadable("EPERM")),
        ])
        #expect(observation == .undetermined(reasons: ["/b: EPERM"]))
        // Must not throw: absence was never established.
        try MetalShaderLibraryCheck.admit(observation)
    }

    @Test("a present library wins over an unreadable sibling root")
    func presentBeatsUnreadable() throws {
        let observation = MetalShaderLibraryCheck.classify([
            MetalShaderRootFinding(root: "/a", outcome: .unreadable("EPERM")),
            MetalShaderRootFinding(root: "/b", outcome: .containsLibrary(path: "/b/lib")),
        ])
        #expect(observation == .present(path: "/b/lib"))
        try MetalShaderLibraryCheck.admit(observation)
    }

    /// A search that inspected nothing at all has established nothing. It must
    /// not read as a clean "we looked everywhere and it is gone".
    @Test("no search roots is refused rather than silently admitted")
    func emptySearchRefuses() {
        let observation = MetalShaderLibraryCheck.classify([])
        #expect(observation == .absent(searched: [], rejected: []))
        #expect(throws: MetalShaderLibraryError.absent(searched: [], rejected: [])) {
            try MetalShaderLibraryCheck.admit(observation)
        }
    }

    // MARK: - path layout

    /// The layout is not free-form: it must match the `SWIFTPM_BUNDLE` and
    /// `METAL_PATH` defines mlx-swift compiles into `device.cpp`, which resolves
    /// `<root>/mlx-swift_Cmlx.bundle/Contents/Resources/default.metallib`.
    @Test("library path matches the layout mlx-swift resolves")
    func libraryLayout() {
        #expect(
            MetalShaderLibraryCheck.libraryPath(inRoot: "/products/Release")
                == "/products/Release/mlx-swift_Cmlx.bundle/Contents/Resources/default.metallib")
        #expect(MetalShaderLibraryCheck.bundleName == "mlx-swift_Cmlx.bundle")
        #expect(MetalShaderLibraryCheck.libraryName == "default.metallib")
    }

    // MARK: - real filesystem

    @Test("a real directory holding the bundle is observed as present")
    func inspectFindsRealLibrary() throws {
        let root = FileManager.default.temporaryDirectory
            .appendingPathComponent("qyebv8-metal-\(UUID().uuidString)")
        let resources = root.appendingPathComponent(
            "\(MetalShaderLibraryCheck.bundleName)/Contents/Resources")
        try FileManager.default.createDirectory(
            at: resources, withIntermediateDirectories: true)
        defer { try? FileManager.default.removeItem(at: root) }
        let library = resources.appendingPathComponent(MetalShaderLibraryCheck.libraryName)
        try Data([0x00]).write(to: library)

        let observation = MetalShaderLibraryCheck.classify(
            MetalShaderLibraryCheck.inspect(roots: [root.path]))
        #expect(observation == .present(path: library.path))
        try MetalShaderLibraryCheck.admit(observation)
    }

    /// A readable directory that simply lacks the bundle — the exact shape of a
    /// `swift build` `.build/release` output directory.
    @Test("a real readable directory without the bundle is observed as absent")
    func inspectRefusesRealSwiftBuildLayout() throws {
        let root = FileManager.default.temporaryDirectory
            .appendingPathComponent("qyebv8-metal-\(UUID().uuidString)")
        try FileManager.default.createDirectory(at: root, withIntermediateDirectories: true)
        defer { try? FileManager.default.removeItem(at: root) }
        // A sibling resource bundle is present, as in a real SwiftPM build, but
        // it is not the MLX shader bundle.
        try FileManager.default.createDirectory(
            at: root.appendingPathComponent("swift-nio_NIOPosix.bundle"),
            withIntermediateDirectories: true)

        let observation = MetalShaderLibraryCheck.classify(
            MetalShaderLibraryCheck.inspect(roots: [root.path]))
        #expect(observation == .absent(searched: [root.path], rejected: []))
        #expect(throws: MetalShaderLibraryError.absent(searched: [root.path], rejected: [])) {
            try MetalShaderLibraryCheck.admit(observation)
        }
    }

    /// The bundle directory existing is not the same as the library existing;
    /// a truncated or half-copied bundle must not read as present.
    @Test("a bundle directory without the metallib inside is still absent")
    func bundleWithoutLibraryIsAbsent() throws {
        let root = FileManager.default.temporaryDirectory
            .appendingPathComponent("qyebv8-metal-\(UUID().uuidString)")
        try FileManager.default.createDirectory(
            at: root.appendingPathComponent(
                "\(MetalShaderLibraryCheck.bundleName)/Contents/Resources"),
            withIntermediateDirectories: true)
        defer { try? FileManager.default.removeItem(at: root) }

        let observation = MetalShaderLibraryCheck.classify(
            MetalShaderLibraryCheck.inspect(roots: [root.path]))
        #expect(observation == .absent(searched: [root.path], rejected: []))
    }

    /// Forged evidence. `mkdir default.metallib` puts an object at exactly the
    /// path the gate looks at, and `FileManager.fileExists` says yes to it. A
    /// directory is not a Metal library: MLX would still abort at the first GPU
    /// touch, but the port would already be bound and the managed launcher
    /// already polling. The terminal object must be a regular file.
    ///
    /// Production call site: `Main.main()` -> `MetalShaderLibraryCheck.inspect`
    /// -> `classify` -> `admit`, before the listener binds.
    @Test("a directory named default.metallib is refused, not admitted")
    func forgedLibraryDirectoryIsRefused() throws {
        let root = FileManager.default.temporaryDirectory
            .appendingPathComponent("qyebv8-metal-\(UUID().uuidString)")
        let library = root.appendingPathComponent(
            "\(MetalShaderLibraryCheck.bundleName)/Contents/Resources/"
                + MetalShaderLibraryCheck.libraryName)
        try FileManager.default.createDirectory(at: library, withIntermediateDirectories: true)
        defer { try? FileManager.default.removeItem(at: root) }

        let findings = MetalShaderLibraryCheck.inspect(roots: [root.path])
        #expect(
            findings == [
                MetalShaderRootFinding(
                    root: root.path,
                    outcome: .libraryNotAFile(path: library.path, kind: "a directory"))
            ])

        let observation = MetalShaderLibraryCheck.classify(findings)
        #expect(
            observation
                == .absent(
                    searched: [root.path], rejected: ["\(library.path) is a directory"]))
        #expect(
            throws: MetalShaderLibraryError.absent(
                searched: [root.path], rejected: ["\(library.path) is a directory"])
        ) {
            try MetalShaderLibraryCheck.admit(observation)
        }
        // The refusal must say what is actually sitting there, not just "not found".
        #expect(
            String(
                describing: MetalShaderLibraryError.absent(
                    searched: [root.path], rejected: ["\(library.path) is a directory"])
            ).contains("is a directory"))
    }

    /// A symlink pointing at a real library is a usable library; a symlink
    /// pointing at nothing is not. Both must be classified on what they resolve
    /// to, not on the fact that something exists at the path.
    @Test("a symlink is judged by what it resolves to")
    func symlinkedLibrary() throws {
        let root = FileManager.default.temporaryDirectory
            .appendingPathComponent("qyebv8-metal-\(UUID().uuidString)")
        let resources = root.appendingPathComponent(
            "\(MetalShaderLibraryCheck.bundleName)/Contents/Resources")
        try FileManager.default.createDirectory(at: resources, withIntermediateDirectories: true)
        defer { try? FileManager.default.removeItem(at: root) }
        let library = resources.appendingPathComponent(MetalShaderLibraryCheck.libraryName)

        // Dangling: refused.
        try FileManager.default.createSymbolicLink(
            at: library, withDestinationURL: root.appendingPathComponent("nowhere.metallib"))
        var findings = MetalShaderLibraryCheck.inspect(roots: [root.path])
        guard case .libraryNotAFile = findings.first?.outcome else {
            Issue.record("dangling symlink was not rejected: \(String(describing: findings))")
            return
        }

        // Resolving to a real file: admitted.
        let real = root.appendingPathComponent("real.metallib")
        try Data([0x00]).write(to: real)
        try FileManager.default.removeItem(at: library)
        try FileManager.default.createSymbolicLink(at: library, withDestinationURL: real)
        findings = MetalShaderLibraryCheck.inspect(roots: [root.path])
        #expect(
            findings == [
                MetalShaderRootFinding(
                    root: root.path, outcome: .containsLibrary(path: library.path))
            ])
        try MetalShaderLibraryCheck.admit(MetalShaderLibraryCheck.classify(findings))
    }

    /// The library path itself may be unreadable while the root lists fine. A
    /// failed stat is not proven absence, so it must land in `unreadable` and
    /// suppress the refusal rather than be reported as a broken build.
    @Test("an unreadable library path is undetermined, not absent")
    func unreadableLibraryPathIsNotAbsence() throws {
        try #require(getuid() != 0, "root can read any directory, so this cannot be exercised")
        let root = FileManager.default.temporaryDirectory
            .appendingPathComponent("qyebv8-metal-\(UUID().uuidString)")
        let resources = root.appendingPathComponent(
            "\(MetalShaderLibraryCheck.bundleName)/Contents/Resources")
        try FileManager.default.createDirectory(at: resources, withIntermediateDirectories: true)
        defer {
            try? FileManager.default.setAttributes(
                [.posixPermissions: 0o755], ofItemAtPath: resources.path)
            try? FileManager.default.removeItem(at: root)
        }
        try Data([0x00]).write(
            to: resources.appendingPathComponent(MetalShaderLibraryCheck.libraryName))
        try FileManager.default.setAttributes(
            [.posixPermissions: 0o000], ofItemAtPath: resources.path)

        let findings = MetalShaderLibraryCheck.inspect(roots: [root.path])
        guard case .unreadable = findings.first?.outcome else {
            Issue.record("expected unreadable, got \(String(describing: findings.first))")
            return
        }
        guard case .undetermined = MetalShaderLibraryCheck.classify(findings) else {
            Issue.record("an unreadable library path was collapsed into a verdict")
            return
        }
        try MetalShaderLibraryCheck.admit(MetalShaderLibraryCheck.classify(findings))
    }

    @Test("a root that does not exist contributes no evidence")
    func missingRootIsNotPresent() {
        let root = FileManager.default.temporaryDirectory
            .appendingPathComponent("qyebv8-absent-\(UUID().uuidString)").path
        let findings = MetalShaderLibraryCheck.inspect(roots: [root])
        #expect(findings == [MetalShaderRootFinding(root: root, outcome: .notPresent)])
    }

    /// Real permission failure: an existing but unreadable root must surface as
    /// `unreadable`, which is what keeps it from being counted as absence.
    @Test("a real unreadable directory is reported unreadable, not absent")
    func realUnreadableRoot() throws {
        try #require(getuid() != 0, "root can read any directory, so this cannot be exercised")
        let root = FileManager.default.temporaryDirectory
            .appendingPathComponent("qyebv8-metal-\(UUID().uuidString)")
        try FileManager.default.createDirectory(at: root, withIntermediateDirectories: true)
        defer {
            try? FileManager.default.setAttributes(
                [.posixPermissions: 0o755], ofItemAtPath: root.path)
            try? FileManager.default.removeItem(at: root)
        }
        try FileManager.default.setAttributes(
            [.posixPermissions: 0o000], ofItemAtPath: root.path)

        let findings = MetalShaderLibraryCheck.inspect(roots: [root.path])
        guard case .unreadable = findings.first?.outcome else {
            Issue.record(
                "expected an unreadable outcome, got \(String(describing: findings.first))")
            return
        }
        // And therefore the launch is not refused.
        try MetalShaderLibraryCheck.admit(MetalShaderLibraryCheck.classify(findings))
    }

    /// The executable's own directory must be searched as a root in its own
    /// right. `swift build` publishes `.build/release` as a symlink to
    /// `.build/<triple>/release`, so the symlink-resolved executable directory
    /// is a path the bundle URL never yields — dropping it would silently
    /// shrink the search. Asserted on the pure composition because in a test
    /// host the bundle URL and the executable directory coincide, which would
    /// hide the difference.
    @Test("the executable directory is a search root distinct from the bundle")
    func executableDirectoryIsItsOwnRoot() {
        let roots = MetalShaderLibraryCheck.composeSearchRoots(
            bundlePath: "/app/.build/release",
            resourcePath: nil,
            executableDirectory: "/app/.build/arm64-apple-macosx/release")
        #expect(roots == ["/app/.build/release", "/app/.build/arm64-apple-macosx/release"])
    }

    @Test("the resource path is searched when the bundle publishes one")
    func resourcePathIsSearched() {
        let roots = MetalShaderLibraryCheck.composeSearchRoots(
            bundlePath: "/app", resourcePath: "/app/Contents/Resources",
            executableDirectory: "/app/bin")
        #expect(roots == ["/app", "/app/Contents/Resources", "/app/bin"])
    }

    @Test("coinciding roots collapse to one entry")
    func coincidingRootsDeduplicate() {
        let roots = MetalShaderLibraryCheck.composeSearchRoots(
            bundlePath: "/app", resourcePath: "/app", executableDirectory: "/app")
        #expect(roots == ["/app"])
    }

    @Test("default search roots are de-duplicated and never empty")
    func defaultRoots() {
        let roots = MetalShaderLibraryCheck.defaultSearchRoots()
        #expect(roots.count == Set(roots).count)
        #expect(!roots.isEmpty)
    }
}
