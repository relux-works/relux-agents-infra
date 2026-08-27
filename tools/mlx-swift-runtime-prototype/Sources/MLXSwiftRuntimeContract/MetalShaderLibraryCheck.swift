import Foundation

/// What inspecting one search root observed about the MLX Metal shader library.
///
/// `absent` and `unreadable` are deliberately distinct. Failing to inspect a
/// directory is not evidence that the library is missing from it, so an
/// unreadable root can never be counted towards a "the build is broken" verdict.
public enum MetalShaderRootOutcome: Sendable, Equatable {
    /// The root holds the shader bundle and a real library file inside it.
    case containsLibrary(path: String)
    /// Something sits at the expected library path but it is not a regular
    /// file, so it cannot be the library MLX loads. Proven non-evidence: it
    /// counts towards absence and is named in the refusal.
    case libraryNotAFile(path: String, kind: String)
    /// The root was inspected successfully and does not hold the library.
    case inspected
    /// Nothing exists at the root. It contributes no evidence either way.
    case notPresent
    /// The root exists but could not be inspected. The reason is carried along.
    case unreadable(String)
}

/// One search root paired with what inspecting it observed.
public struct MetalShaderRootFinding: Sendable, Equatable {
    public let root: String
    public let outcome: MetalShaderRootOutcome

    public init(root: String, outcome: MetalShaderRootOutcome) {
        self.root = root
        self.outcome = outcome
    }
}

/// The verdict over every search root.
public enum MetalShaderLibraryObservation: Sendable, Equatable {
    /// The library was found. Generation can reach the GPU.
    case present(path: String)
    /// Every root was inspected cleanly and none held a library file.
    /// `rejected` names any object that occupied the library path without
    /// being one, so the refusal can say what is actually there.
    case absent(searched: [String], rejected: [String])
    /// The library was not found, but at least one root could not be inspected,
    /// so its absence was never established.
    case undetermined(reasons: [String])
}

public enum MetalShaderLibraryError: Error, Equatable, CustomStringConvertible {
    case absent(searched: [String], rejected: [String])

    public var description: String {
        switch self {
        case .absent(let searched, let rejected):
            let roots = searched.isEmpty ? "<no search roots>" : searched.joined(separator: ", ")
            let forged =
                rejected.isEmpty
                ? ""
                : " An object occupies the expected library path without being a library file: "
                    + rejected.joined(separator: "; ")
                    + ". Existence at that path is not the same as a loadable library."
            return """
                MLX Metal shader library \
                \(MetalShaderLibraryCheck.bundleName)/Contents/Resources/\
                \(MetalShaderLibraryCheck.libraryName) was not found next to this executable \
                (searched: \(roots)). This binary was almost certainly produced by \
                `swift build`, which cannot compile the Metal shaders; upstream mlx-swift \
                documents that the build has to be done via Xcode. Rebuild with \
                `xcodebuild build -scheme mlx-swift-runtime-prototype -configuration Release \
                -destination 'platform=macOS,arch=arm64' -derivedDataPath <dir> \
                -skipPackagePluginValidation` and run the product from \
                <derivedDataPath>/Build/Products/Release. \
                A Metal Toolchain is required: `xcodebuild -downloadComponent MetalToolchain`.\
                \(forged)
                """
        }
    }
}

/// Fail-closed admission of the Metal shader library the MLX backend needs.
///
/// Without it MLX aborts the whole process deep inside C++ (`Failed to load the
/// default metallib`) at the first GPU touch — which for this runtime is halfway
/// through a multi-GiB weight load, long after the port is bound and the managed
/// launcher has started polling for readiness. Naming the condition up front
/// turns that opaque abort into a specific, actionable refusal.
public enum MetalShaderLibraryCheck {
    /// The SwiftPM resource bundle mlx-swift compiles its shaders into. Matches
    /// the `SWIFTPM_BUNDLE` define in mlx-swift's `Package.swift`.
    public static let bundleName = "mlx-swift_Cmlx.bundle"
    /// Matches the `METAL_PATH` define in mlx-swift's `Package.swift`.
    public static let libraryName = "default.metallib"

    /// Where the shader library sits relative to a search root.
    public static func libraryPath(inRoot root: String) -> String {
        (root as NSString)
            .appendingPathComponent(bundleName)
            .appending("/Contents/Resources/\(libraryName)")
    }

    /// Pure policy: turn per-root findings into a single verdict.
    ///
    /// A root that could not be inspected suppresses the `absent` verdict. The
    /// runtime would rather launch and let MLX fail loudly than refuse to start
    /// on the strength of a directory it was merely unable to read.
    public static func classify(
        _ findings: [MetalShaderRootFinding]
    ) -> MetalShaderLibraryObservation {
        for finding in findings {
            if case .containsLibrary(let path) = finding.outcome {
                return .present(path: path)
            }
        }
        let reasons = findings.compactMap { finding -> String? in
            guard case .unreadable(let reason) = finding.outcome else { return nil }
            return "\(finding.root): \(reason)"
        }
        guard reasons.isEmpty else {
            return .undetermined(reasons: reasons)
        }
        let rejected = findings.compactMap { finding -> String? in
            guard case .libraryNotAFile(let path, let kind) = finding.outcome else { return nil }
            return "\(path) is \(kind)"
        }
        return .absent(searched: findings.map(\.root), rejected: rejected)
    }

    /// Inspect real directories on disk.
    ///
    /// The terminal object must be a regular file. `FileManager.fileExists` is
    /// true for a directory too, so a `mkdir default.metallib` would otherwise
    /// mint its own admission evidence and let the launch bind a port on a build
    /// that still cannot reach the GPU.
    public static func inspect(
        roots: [String], fileManager: FileManager = .default
    ) -> [MetalShaderRootFinding] {
        roots.map { root in
            let libraryPath = libraryPath(inRoot: root)
            switch FileObjects.probe(path: libraryPath, fileManager: fileManager) {
            case .regularFile:
                return MetalShaderRootFinding(
                    root: root, outcome: .containsLibrary(path: libraryPath))
            case .otherObject(let kind):
                return MetalShaderRootFinding(
                    root: root, outcome: .libraryNotAFile(path: libraryPath, kind: kind))
            case .unreadable(let reason):
                // The library path could not be stat'ed. Its absence was never
                // established, so this root must not count towards `absent`.
                return MetalShaderRootFinding(
                    root: root, outcome: .unreadable("\(libraryPath): \(reason)"))
            case .absent:
                break
            }
            var isDirectory: ObjCBool = false
            guard fileManager.fileExists(atPath: root, isDirectory: &isDirectory),
                isDirectory.boolValue
            else {
                return MetalShaderRootFinding(root: root, outcome: .notPresent)
            }
            do {
                _ = try fileManager.contentsOfDirectory(atPath: root)
                return MetalShaderRootFinding(root: root, outcome: .inspected)
            } catch {
                return MetalShaderRootFinding(
                    root: root, outcome: .unreadable(String(describing: error)))
            }
        }
    }

    /// Pure composition of the search roots, in priority order, de-duplicated.
    ///
    /// The executable directory is a distinct root from the bundle URL and not
    /// a redundant one: `swift build` publishes `.build/release` as a symlink to
    /// `.build/<triple>/release`, so the symlink-resolved executable directory
    /// is a path the bundle URL alone never yields.
    public static func composeSearchRoots(
        bundlePath: String, resourcePath: String?, executableDirectory: String
    ) -> [String] {
        var roots = [bundlePath]
        if let resourcePath {
            roots.append(resourcePath)
        }
        roots.append(executableDirectory)
        var seen = Set<String>()
        return roots.filter { seen.insert($0).inserted }
    }

    /// The directories MLX itself searches for a command-line executable: the
    /// main bundle URL (the directory holding the binary), its resource URL and
    /// the symlink-resolved directory the executable actually lives in.
    public static func defaultSearchRoots(bundle: Bundle = .main) -> [String] {
        composeSearchRoots(
            bundlePath: bundle.bundleURL.path,
            resourcePath: bundle.resourceURL?.path,
            executableDirectory: URL(fileURLWithPath: CommandLine.arguments[0])
                .resolvingSymlinksInPath()
                .deletingLastPathComponent()
                .path)
    }

    /// Throw only when the library's absence was actually established.
    public static func admit(_ observation: MetalShaderLibraryObservation) throws {
        switch observation {
        case .present, .undetermined:
            return
        case .absent(let searched, let rejected):
            throw MetalShaderLibraryError.absent(searched: searched, rejected: rejected)
        }
    }
}
