import Foundation

/// The generated text carried by one OpenAI-compatible streamed delta.
///
/// The two deployed runtimes spell reasoning differently: `mlx_lm.server`
/// uses `reasoning`, while `llama-server` uses `reasoning_content`.  Timing
/// code must use this reader for both the first and last generated event so a
/// field-name difference cannot move only one side's clock boundary.
public struct RuntimeStreamDelta: Equatable, Sendable {
    public let content: String
    public let reasoning: String

    public var generatedText: String { content + reasoning }
    public var carriesGeneratedText: Bool { !generatedText.isEmpty }

    public static func read(_ delta: [String: Any]) -> RuntimeStreamDelta {
        let content = (delta["content"] as? String) ?? ""
        let reasoning =
            ((delta["reasoning"] as? String) ?? "")
            + ((delta["reasoning_content"] as? String) ?? "")
        return RuntimeStreamDelta(content: content, reasoning: reasoning)
    }
}
