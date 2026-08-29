import Foundation

/// A decoded `POST /v1/chat/completions` body.
///
/// Decoding is deliberately permissive about fields the runtime ignores and
/// strict about fields whose silent acceptance would misrepresent the runtime's
/// capabilities (see ``ChatCompletionAdmission``).
public struct ChatCompletionRequest: Sendable, Equatable {
    public struct Message: Sendable, Equatable {
        /// The role exactly as the client wrote it, before any mapping.
        public let rawRole: String
        public let content: String
        public let toolCalls: [ToolCallPayload]
        public let toolCallID: String?

        public init(
            rawRole: String, content: String, toolCalls: [ToolCallPayload] = [],
            toolCallID: String? = nil
        ) {
            self.rawRole = rawRole
            self.content = content
            self.toolCalls = toolCalls
            self.toolCallID = toolCallID
        }
    }

    public let model: String
    public let messages: [Message]
    public let maxTokens: Int?
    public let temperature: Double?
    public let topP: Double?
    public let stream: Bool
    public let includeUsage: Bool
    public let tools: [JSONValue]
    /// Present only so the runtime can refuse it; never forwarded.
    public let reasoningEffort: String?
    /// Deterministic sampling seed, forwarded to the sampler when set.
    public let seed: UInt64?
    /// Unsupported fields observed in the body, in stable order.
    public let unsupportedFields: [String]
}

public enum ChatCompletionDecodingError: Error, Equatable, CustomStringConvertible {
    case malformedJSON(String)
    case notAnObject
    case missingField(String)
    case wrongType(field: String, expected: String)
    case numberOutOfRange(field: String, value: Double)
    case unsupportedContentPart(index: Int, type: String)

    public var description: String {
        switch self {
        case .malformedJSON(let reason): return "request body is not valid JSON: \(reason)"
        case .notAnObject: return "request body must be a JSON object"
        case .missingField(let field): return "missing required field \(field.debugDescription)"
        case .wrongType(let field, let expected):
            return "field \(field.debugDescription) must be \(expected)"
        case .numberOutOfRange(let field, let value):
            return
                "field \(field.debugDescription) value \(value) is outside the representable integer range"
        case .unsupportedContentPart(let index, let type):
            return
                "message content part \(index) has unsupported type \(type.debugDescription); only text is supported"
        }
    }
}

extension ChatCompletionRequest {
    public static func decode(from data: Data) throws -> ChatCompletionRequest {
        let root: JSONValue
        do {
            root = try JSONDecoder().decode(JSONValue.self, from: data)
        } catch {
            throw ChatCompletionDecodingError.malformedJSON(String(describing: error))
        }
        guard let object = root.objectValue else {
            throw ChatCompletionDecodingError.notAnObject
        }

        guard let modelValue = object["model"] else {
            throw ChatCompletionDecodingError.missingField("model")
        }
        guard case .string(let model) = modelValue else {
            throw ChatCompletionDecodingError.wrongType(field: "model", expected: "a string")
        }

        guard let messagesValue = object["messages"] else {
            throw ChatCompletionDecodingError.missingField("messages")
        }
        guard case .array(let rawMessages) = messagesValue else {
            throw ChatCompletionDecodingError.wrongType(field: "messages", expected: "an array")
        }
        let messages = try rawMessages.map(decodeMessage)

        var includeUsage = false
        if case .object(let streamOptions)? = object["stream_options"],
            case .bool(let flag)? = streamOptions["include_usage"]
        {
            includeUsage = flag
        }

        var tools: [JSONValue] = []
        if let toolsValue = object["tools"], toolsValue != .null {
            guard case .array(let list) = toolsValue else {
                throw ChatCompletionDecodingError.wrongType(field: "tools", expected: "an array")
            }
            tools = list
        }

        var reasoningEffort: String?
        if let value = object["reasoning_effort"], value != .null {
            guard case .string(let effort) = value else {
                throw ChatCompletionDecodingError.wrongType(
                    field: "reasoning_effort", expected: "a string")
            }
            reasoningEffort = effort
        }

        var seed: UInt64?
        if let value = object["seed"], value != .null {
            guard case .int(let raw) = value, raw >= 0 else {
                throw ChatCompletionDecodingError.wrongType(
                    field: "seed", expected: "a non-negative integer")
            }
            seed = UInt64(raw)
        }

        return ChatCompletionRequest(
            model: model,
            messages: messages,
            maxTokens: try optionalInt(object["max_tokens"], field: "max_tokens"),
            temperature: try optionalDouble(object["temperature"], field: "temperature"),
            topP: try optionalDouble(object["top_p"], field: "top_p"),
            stream: try optionalBool(object["stream"], field: "stream") ?? false,
            includeUsage: includeUsage,
            tools: tools,
            reasoningEffort: reasoningEffort,
            seed: seed,
            unsupportedFields: UnsupportedParameters.present(in: object))
    }

    private static func decodeMessage(_ value: JSONValue) throws -> Message {
        guard let object = value.objectValue else {
            throw ChatCompletionDecodingError.wrongType(
                field: "messages[]", expected: "an array of objects")
        }
        guard case .string(let role)? = object["role"] else {
            throw ChatCompletionDecodingError.missingField("messages[].role")
        }

        var content = ""
        switch object["content"] {
        case .some(.string(let text)):
            content = text
        case .some(.array(let parts)):
            // OpenAI multipart content: only `text` parts can reach a text-only
            // profile, so an image/audio part is refused rather than dropped.
            var joined = ""
            for (index, part) in parts.enumerated() {
                guard let partObject = part.objectValue else {
                    throw ChatCompletionDecodingError.wrongType(
                        field: "messages[].content[]", expected: "an array of objects")
                }
                guard case .string(let type)? = partObject["type"] else {
                    throw ChatCompletionDecodingError.missingField("messages[].content[].type")
                }
                guard type == "text", case .string(let text)? = partObject["text"] else {
                    throw ChatCompletionDecodingError.unsupportedContentPart(
                        index: index, type: type)
                }
                joined += text
            }
            content = joined
        case .some(.null), .none:
            content = ""
        case .some:
            throw ChatCompletionDecodingError.wrongType(
                field: "messages[].content", expected: "a string or an array of text parts")
        }

        var toolCalls: [ToolCallPayload] = []
        if case .array(let rawCalls)? = object["tool_calls"] {
            toolCalls = try rawCalls.map(ToolCallPayload.decode)
        }

        var toolCallID: String?
        if case .string(let id)? = object["tool_call_id"] {
            toolCallID = id
        }

        return Message(
            rawRole: role, content: content, toolCalls: toolCalls, toolCallID: toolCallID)
    }

    /// Decode an optional integer field without ever trapping.
    ///
    /// A JSON number too large for `Int` — `1e300`, or any literal past
    /// `Int.max` — decodes as a `Double`, and `Int(_:)` on it is a fatal error,
    /// not a thrown one. That killed the whole managed runtime from an ordinary
    /// request body, bypassing every refusal downstream. The conversion is
    /// therefore total: out of range is refused, exactly like a wrong type.
    private static func optionalInt(_ value: JSONValue?, field: String) throws -> Int? {
        switch value {
        case .none, .some(.null): return nil
        case .some(.int(let value)): return value
        case .some(.double(let value)) where value == value.rounded():
            guard let exact = Int(exactly: value) else {
                throw ChatCompletionDecodingError.numberOutOfRange(field: field, value: value)
            }
            return exact
        default: throw ChatCompletionDecodingError.wrongType(field: field, expected: "an integer")
        }
    }

    private static func optionalDouble(_ value: JSONValue?, field: String) throws -> Double? {
        switch value {
        case .none, .some(.null): return nil
        case .some(.int(let value)): return Double(value)
        case .some(.double(let value)): return value
        default: throw ChatCompletionDecodingError.wrongType(field: field, expected: "a number")
        }
    }

    private static func optionalBool(_ value: JSONValue?, field: String) throws -> Bool? {
        switch value {
        case .none, .some(.null): return nil
        case .some(.bool(let value)): return value
        default: throw ChatCompletionDecodingError.wrongType(field: field, expected: "a boolean")
        }
    }
}
