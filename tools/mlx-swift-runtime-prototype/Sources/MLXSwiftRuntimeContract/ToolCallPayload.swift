import Foundation

/// An OpenAI `tool_calls[]` entry.
///
/// `arguments` stays a JSON *string* in both directions, exactly as the OpenAI
/// contract specifies and as `mlx_lm.server` emits it.
public struct ToolCallPayload: Sendable, Equatable {
    public let id: String
    public let name: String
    public let argumentsJSON: String
    /// Populated only on streaming deltas.
    public let index: Int?

    public init(id: String, name: String, argumentsJSON: String, index: Int? = nil) {
        self.id = id
        self.name = name
        self.argumentsJSON = argumentsJSON
        self.index = index
    }

    public func indexed(_ index: Int) -> ToolCallPayload {
        ToolCallPayload(id: id, name: name, argumentsJSON: argumentsJSON, index: index)
    }

    public var json: JSONValue {
        let function: [String: JSONValue] = [
            "name": .string(name),
            "arguments": .string(argumentsJSON),
        ]
        var entry: [String: JSONValue] = [
            "id": .string(id),
            "type": .string("function"),
            "function": .object(function),
        ]
        if let index {
            entry["index"] = .int(index)
        }
        return .object(entry)
    }

    static func decode(_ value: JSONValue) throws -> ToolCallPayload {
        guard let object = value.objectValue else {
            throw ChatCompletionDecodingError.wrongType(
                field: "messages[].tool_calls[]", expected: "an array of objects")
        }
        guard let function = object["function"]?.objectValue else {
            throw ChatCompletionDecodingError.missingField("messages[].tool_calls[].function")
        }
        guard case .string(let name)? = function["name"] else {
            throw ChatCompletionDecodingError.missingField("messages[].tool_calls[].function.name")
        }
        var arguments = "{}"
        if case .string(let raw)? = function["arguments"] {
            arguments = raw
        }
        var id = ""
        if case .string(let raw)? = object["id"] {
            id = raw
        }
        return ToolCallPayload(id: id, name: name, argumentsJSON: arguments)
    }

    /// Encode tool-call arguments to the JSON string form.
    ///
    /// Uses sorted keys so a given argument set always serializes identically —
    /// smoke output and golden comparisons would otherwise be order-dependent.
    public static func encodeArguments(_ arguments: [String: JSONValue]) -> String {
        let encoder = JSONEncoder()
        encoder.outputFormatting = [.sortedKeys, .withoutEscapingSlashes]
        guard let data = try? encoder.encode(JSONValue.object(arguments)),
            let text = String(data: data, encoding: .utf8)
        else {
            return "{}"
        }
        return text
    }
}
