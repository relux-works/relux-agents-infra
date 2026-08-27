import Foundation

/// Why generation stopped, in OpenAI vocabulary.
public enum FinishReason: String, Sendable {
    case stop
    case length
    case toolCalls = "tool_calls"
}

public struct CompletionUsage: Sendable, Equatable {
    public let promptTokens: Int
    public let completionTokens: Int

    public init(promptTokens: Int, completionTokens: Int) {
        self.promptTokens = promptTokens
        self.completionTokens = completionTokens
    }

    public var json: JSONValue {
        .object([
            "prompt_tokens": .int(promptTokens),
            "completion_tokens": .int(completionTokens),
            "total_tokens": .int(promptTokens + completionTokens),
        ])
    }
}

/// Builds the OpenAI-shaped bodies the Pi profile consumes.
///
/// Field names track `mlx_lm.server` rather than a generic OpenAI client: the
/// Pi profile is configured against that runtime today, so reasoning is emitted
/// under `reasoning` (not `reasoning_content`) and `system_fingerprint` is
/// present on every packet.
public struct ChatCompletionResponseBuilder: Sendable {
    public let requestID: String
    public let modelID: String
    public let created: Int
    public let systemFingerprint: String

    public init(requestID: String, modelID: String, created: Int, systemFingerprint: String) {
        self.requestID = requestID
        self.modelID = modelID
        self.created = created
        self.systemFingerprint = systemFingerprint
    }

    private func envelope(object: String) -> [String: JSONValue] {
        [
            "id": .string(requestID),
            "system_fingerprint": .string(systemFingerprint),
            "object": .string(object),
            "model": .string(modelID),
            "created": .int(created),
        ]
    }

    /// The single body of a non-streaming completion.
    public func completion(
        content: String,
        reasoning: String,
        toolCalls: [ToolCallPayload],
        finishReason: FinishReason,
        usage: CompletionUsage
    ) -> JSONValue {
        var message: [String: JSONValue] = ["role": .string("assistant")]
        // The schema requires `content` to be present even when empty, so an
        // empty completion is reported as an explicit null rather than omitted.
        message["content"] = content.isEmpty ? .null : .string(content)
        if !reasoning.isEmpty {
            message["reasoning"] = .string(reasoning)
        }
        if !toolCalls.isEmpty {
            message["tool_calls"] = .array(toolCalls.map(\.json))
        }

        var body = envelope(object: "chat.completion")
        body["choices"] = .array([
            .object([
                "index": .int(0),
                "finish_reason": .string(finishReason.rawValue),
                "message": .object(message),
            ])
        ])
        body["usage"] = usage.json
        return .object(body)
    }

    /// An incremental `chat.completion.chunk`.
    public func chunk(
        content: String,
        reasoning: String,
        toolCalls: [ToolCallPayload],
        finishReason: FinishReason?
    ) -> JSONValue {
        var delta: [String: JSONValue] = ["role": .string("assistant")]
        if !content.isEmpty {
            delta["content"] = .string(content)
        }
        if !reasoning.isEmpty {
            delta["reasoning"] = .string(reasoning)
        }
        if !toolCalls.isEmpty {
            delta["tool_calls"] = .array(toolCalls.map(\.json))
        }

        var body = envelope(object: "chat.completion.chunk")
        body["choices"] = .array([
            .object([
                "index": .int(0),
                "finish_reason": finishReason.map { .string($0.rawValue) } ?? .null,
                "delta": .object(delta),
            ])
        ])
        return .object(body)
    }

    /// The choice-less usage packet emitted when `stream_options.include_usage`
    /// is set. `supports_usage_in_streaming = true` on the Pi profile, so this
    /// packet is part of the contract, not an extra.
    public func streamingUsage(_ usage: CompletionUsage) -> JSONValue {
        var body = envelope(object: "chat.completion")
        body["choices"] = .array([])
        body["usage"] = usage.json
        return .object(body)
    }

    public static func error(message: String, type: String, code: String) -> JSONValue {
        .object([
            "error": .object([
                "message": .string(message),
                "type": .string(type),
                "code": .string(code),
            ])
        ])
    }
}

public enum JSONEncoding {
    /// Encode a body for the wire.
    ///
    /// `withoutEscapingSlashes` matters here: the configured model ID is an
    /// absolute filesystem path, and escaping its separators would make the ID
    /// the launcher compares against differ from the one it configured.
    public static func data(_ value: JSONValue) throws -> Data {
        let encoder = JSONEncoder()
        encoder.outputFormatting = [.sortedKeys, .withoutEscapingSlashes]
        return try encoder.encode(value)
    }

    public static func string(_ value: JSONValue) throws -> String {
        String(decoding: try data(value), as: UTF8.self)
    }
}
