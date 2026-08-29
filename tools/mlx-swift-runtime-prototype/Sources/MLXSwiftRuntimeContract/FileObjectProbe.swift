import Foundation

/// What inspecting one filesystem path established about the object at it.
///
/// `absent` and `unreadable` are deliberately distinct. Every caller in this
/// module refuses a launch on proven absence, so folding a failed stat into
/// absence would turn an EACCES on a perfectly good tree into a confident
/// "your build is broken" verdict. A failure to read is never evidence of
/// absence.
enum FileObjectProbe: Sendable, Equatable {
    /// A regular file, or a symlink resolving to one.
    case regularFile
    /// Something exists at the path but it is not a regular file. `kind`
    /// describes what was found, so the refusal can name it.
    case otherObject(kind: String)
    /// Nothing exists at the path. Proven, not assumed.
    case absent
    /// The object's type was never established. The reason is carried along.
    case unreadable(String)
}

enum FileObjects {
    /// Establish whether `path` holds a regular file.
    ///
    /// Existence alone is not enough for any caller here: a directory, socket
    /// or dangling symlink sitting at the expected path is forged evidence, not
    /// a usable file, and `FileManager.fileExists(atPath:)` cannot tell them
    /// apart — it answers `true` for all of them.
    ///
    /// POSIX `stat` is used directly rather than
    /// `FileManager.attributesOfItem(atPath:)` because the distinction that
    /// matters here is exactly the one Foundation blurs: `stat` follows the
    /// symlink and reports the target, and its `errno` separates "there is
    /// nothing here" (`ENOENT`) from "I was not allowed to look" (`EACCES`).
    static func probe(path: String, fileManager: FileManager = .default) -> FileObjectProbe {
        var status = stat()
        guard stat(path, &status) != 0 else {
            return kind(ofMode: status.st_mode)
        }
        let followError = errno
        guard followError == ENOENT else {
            return .unreadable(describe(errno: followError))
        }
        // `stat` reports ENOENT for a symlink pointing at nothing as well as for
        // a path with nothing on it. `lstat` tells the two apart, and they are
        // different facts: a dangling link is an object that exists and is not
        // a file, which is a refusal, not an absence.
        var linkStatus = stat()
        guard lstat(path, &linkStatus) != 0 else {
            return .otherObject(kind: "a symbolic link that resolves to nothing")
        }
        let linkError = errno
        guard linkError == ENOENT else {
            return .unreadable(describe(errno: linkError))
        }
        return .absent
    }

    private static func kind(ofMode mode: mode_t) -> FileObjectProbe {
        switch mode & S_IFMT {
        case S_IFREG: return .regularFile
        case S_IFDIR: return .otherObject(kind: "a directory")
        case S_IFLNK: return .otherObject(kind: "a symbolic link")
        case S_IFSOCK: return .otherObject(kind: "a socket")
        case S_IFIFO: return .otherObject(kind: "a FIFO")
        case S_IFCHR: return .otherObject(kind: "a character device")
        case S_IFBLK: return .otherObject(kind: "a block device")
        default: return .otherObject(kind: "an object of mode \(String(mode & S_IFMT, radix: 8))")
        }
    }

    private static func describe(errno code: Int32) -> String {
        "\(String(cString: strerror(code))) (errno \(code))"
    }
}
