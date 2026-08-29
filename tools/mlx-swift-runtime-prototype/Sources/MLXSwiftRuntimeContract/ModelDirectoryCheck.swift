import Foundation

/// What an inspection of the configured model directory observed.
///
/// `missing` and `unreadable` are deliberately distinct: a directory we failed
/// to read is not evidence that the model is absent, and neither outcome may be
/// downgraded into "load something else instead".
public enum ModelDirectoryObservation: Sendable, Equatable {
    /// A readable directory containing every required file.
    case complete
    /// A readable directory that is missing at least one required file.
    case incomplete(missing: [String])
    /// A required entry exists by name but is not a regular file. Existing in a
    /// directory listing is not the same as being a loadable file.
    case notRegularFiles(details: [String])
    /// Nothing exists at the path.
    case missing
    /// The path exists but is not a directory.
    case notADirectory
    /// The path could not be inspected. The description carries the reason.
    case unreadable(String)
}

public enum ModelDirectoryError: Error, Equatable, CustomStringConvertible {
    case missing(path: String)
    case notADirectory(path: String)
    case incomplete(path: String, missing: [String])
    case notRegularFiles(path: String, details: [String])
    case unreadable(path: String, reason: String)

    public var description: String {
        switch self {
        case .missing(let path):
            return "model directory \(path.debugDescription) does not exist"
        case .notADirectory(let path):
            return "model path \(path.debugDescription) is not a directory"
        case .incomplete(let path, let missing):
            return
                "model directory \(path.debugDescription) is missing required files: \(missing.joined(separator: ", "))"
        case .notRegularFiles(let path, let details):
            return
                "model directory \(path.debugDescription) has required entries that are not regular files: \(details.joined(separator: ", "))"
        case .unreadable(let path, let reason):
            return "model directory \(path.debugDescription) could not be inspected: \(reason)"
        }
    }
}

/// Fail-closed admission of a local model directory.
///
/// The prototype links no downloader, so a rejected directory cannot be
/// silently replaced by a Hugging Face snapshot; it can only stop the launch.
public enum ModelDirectoryCheck {
    /// Files an MLX Swift LM local load needs before it can even start.
    public static let requiredFiles = ["config.json", "tokenizer_config.json"]

    /// Convert an observation into an admission decision.
    ///
    /// Every non-`complete` observation throws. In particular `unreadable`
    /// throws its own error instead of collapsing into `missing`, so an
    /// EPERM/EIO on the model directory can never be reported as "no model".
    public static func admit(path: String, observation: ModelDirectoryObservation) throws {
        switch observation {
        case .complete:
            return
        case .incomplete(let missing):
            throw ModelDirectoryError.incomplete(path: path, missing: missing)
        case .notRegularFiles(let details):
            throw ModelDirectoryError.notRegularFiles(path: path, details: details)
        case .missing:
            throw ModelDirectoryError.missing(path: path)
        case .notADirectory:
            throw ModelDirectoryError.notADirectory(path: path)
        case .unreadable(let reason):
            throw ModelDirectoryError.unreadable(path: path, reason: reason)
        }
    }

    /// Inspect a real directory on disk.
    public static func observe(
        path: String, fileManager: FileManager = .default
    ) -> ModelDirectoryObservation {
        var isDirectory: ObjCBool = false
        guard fileManager.fileExists(atPath: path, isDirectory: &isDirectory) else {
            return .missing
        }
        guard isDirectory.boolValue else {
            return .notADirectory
        }
        let entries: [String]
        do {
            entries = try fileManager.contentsOfDirectory(atPath: path)
        } catch {
            return .unreadable(String(describing: error))
        }
        // A name in a directory listing is not a file. `mkdir config.json`
        // would otherwise mint its own admission evidence, so every required
        // entry is stat'ed and must turn out to be a regular file.
        var missing: [String] = []
        var notRegular: [String] = []
        var unreadable: [String] = []
        for required in requiredFiles {
            switch probe(entry: required, inDirectory: path, fileManager: fileManager) {
            case .regularFile: continue
            case .absent: missing.append(required)
            case .otherObject(let kind): notRegular.append("\(required) is \(kind)")
            case .unreadable(let reason): unreadable.append("\(required): \(reason)")
            }
        }

        // A weight file is required as well, but its name varies between a
        // single `model.safetensors` and a sharded index, so it is matched by
        // extension rather than by an exact name.
        let weightNames = entries.filter { $0.hasSuffix(".safetensors") }
        var weightFound = false
        var weightRejected: [String] = []
        for candidate in weightNames where !weightFound {
            switch probe(entry: candidate, inDirectory: path, fileManager: fileManager) {
            case .regularFile: weightFound = true
            case .otherObject(let kind): weightRejected.append("\(candidate) is \(kind)")
            case .unreadable(let reason): unreadable.append("\(candidate): \(reason)")
            case .absent: continue
            }
        }
        if !weightFound {
            if weightRejected.isEmpty && unreadable.isEmpty {
                missing.append("*.safetensors")
            } else {
                notRegular.append(contentsOf: weightRejected)
            }
        }

        // A failed stat is not proof of anything, so it outranks both verdicts
        // below rather than being laundered into "missing".
        guard unreadable.isEmpty else {
            return .unreadable(unreadable.joined(separator: "; "))
        }
        guard missing.isEmpty else {
            return .incomplete(missing: missing)
        }
        guard notRegular.isEmpty else {
            return .notRegularFiles(details: notRegular)
        }
        return .complete
    }

    private static func probe(
        entry: String, inDirectory path: String, fileManager: FileManager
    ) -> FileObjectProbe {
        FileObjects.probe(
            path: (path as NSString).appendingPathComponent(entry), fileManager: fileManager)
    }
}
