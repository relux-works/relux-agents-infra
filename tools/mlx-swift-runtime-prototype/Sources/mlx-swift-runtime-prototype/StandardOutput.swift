import Foundation
import MLXSwiftRuntimeContract

/// Line-buffered, serialized writers for the two streams `model-harness`
/// forwards from a managed child.
final class StandardOutput: @unchecked Sendable {
    static let shared = StandardOutput()

    private let lock = NSLock()

    func event(_ event: RuntimeEvent) {
        guard let line = try? event.line() else { return }
        write(line + "\n", to: FileHandle.standardOutput)
    }

    func log(_ message: String) {
        write("mlx-swift-runtime-prototype: \(message)\n", to: FileHandle.standardError)
    }

    private func write(_ text: String, to handle: FileHandle) {
        lock.lock()
        defer { lock.unlock() }
        try? handle.write(contentsOf: Data(text.utf8))
    }
}
