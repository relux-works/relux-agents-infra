import Foundation

/// Splits a Qwen chat-template generation stream into reasoning and content.
///
/// The bundled Qwen3.5 chat template ends its generation prompt with a bare
/// `<think>\n`, so generation *starts inside* the reasoning block and the model
/// only ever emits the closing `</think>`. Everything before that marker is
/// reasoning; everything after is the answer.
///
/// Markers can straddle chunk boundaries, so text that could still become a
/// marker prefix is held back rather than emitted. This mirrors the
/// hold-back behaviour of `mlx_lm.generate.TextStateMachine`, which the Python
/// runtime uses for the same job.
public struct ReasoningSplitter: Sendable {
    public enum Mode: Sendable, Equatable {
        /// Generation begins inside `<think>` (`enable_thinking` on).
        case startsInReasoning
        /// Generation begins in normal content (`enable_thinking` off).
        case startsInContent
    }

    public struct Output: Sendable, Equatable {
        public let reasoning: String
        public let content: String

        public var isEmpty: Bool { reasoning.isEmpty && content.isEmpty }
    }

    static let thinkStart = "<think>"
    static let thinkEnd = "</think>"

    private enum State {
        case reasoning
        case content
    }

    private var state: State
    private var buffer = ""

    public init(mode: Mode) {
        state = mode == .startsInReasoning ? .reasoning : .content
    }

    public var isInReasoning: Bool { state == .reasoning }

    /// Consume a decoded chunk and return whatever is now safe to emit.
    public mutating func consume(_ chunk: String) -> Output {
        buffer += chunk
        var reasoning = ""
        var content = ""

        while true {
            let marker = state == .reasoning ? Self.thinkEnd : Self.thinkStart
            if let range = buffer.range(of: marker) {
                let head = String(buffer[buffer.startIndex ..< range.lowerBound])
                if state == .reasoning {
                    reasoning += head
                    state = .content
                } else {
                    content += head
                    state = .reasoning
                }
                buffer = String(buffer[range.upperBound...])
                continue
            }

            // No complete marker. Hold back any suffix that could still grow
            // into one; emit the rest.
            let held = Self.longestMarkerPrefixSuffix(of: buffer, marker: marker)
            let emittable = String(buffer.dropLast(held))
            if state == .reasoning {
                reasoning += emittable
            } else {
                content += emittable
            }
            buffer = String(buffer.suffix(held))
            break
        }

        return Output(reasoning: reasoning, content: content)
    }

    /// Emit whatever is still buffered.
    ///
    /// Called when generation ends without the marker completing (for example a
    /// `length` stop mid-`</think`). The held-back bytes are real model output
    /// and are attributed to the state they were produced in — dropping them
    /// would silently truncate the answer.
    public mutating func flush() -> Output {
        let remainder = buffer
        buffer = ""
        guard !remainder.isEmpty else { return Output(reasoning: "", content: "") }
        return state == .reasoning
            ? Output(reasoning: remainder, content: "")
            : Output(reasoning: "", content: remainder)
    }

    /// Length of the longest suffix of `text` that is a proper prefix of `marker`.
    static func longestMarkerPrefixSuffix(of text: String, marker: String) -> Int {
        let maximum = min(text.count, marker.count - 1)
        guard maximum > 0 else { return 0 }
        for length in stride(from: maximum, through: 1, by: -1) {
            if marker.hasPrefix(text.suffix(length)) {
                return length
            }
        }
        return 0
    }
}
