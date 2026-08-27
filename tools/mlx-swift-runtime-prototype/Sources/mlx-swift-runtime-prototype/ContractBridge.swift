import Foundation
import MLXLMCommon
import MLXSwiftRuntimeContract

/// Conversions between the transport-facing contract types and MLX Swift LM.
enum ContractBridge {
    static func mlxValue(_ value: MLXSwiftRuntimeContract.JSONValue) -> MLXLMCommon.JSONValue {
        switch value {
        case .null: return .null
        case .bool(let value): return .bool(value)
        case .int(let value): return .int(value)
        case .double(let value): return .double(value)
        case .string(let value): return .string(value)
        case .array(let value): return .array(value.map(mlxValue))
        case .object(let value): return .object(value.mapValues(mlxValue))
        }
    }

    static func contractValue(_ value: MLXLMCommon.JSONValue) -> MLXSwiftRuntimeContract.JSONValue {
        switch value {
        case .null: return .null
        case .bool(let value): return .bool(value)
        case .int(let value): return .int(value)
        case .double(let value): return .double(value)
        case .string(let value): return .string(value)
        case .array(let value): return .array(value.map(contractValue))
        case .object(let value): return .object(value.mapValues(contractValue))
        }
    }

    /// Translate an admitted OpenAI transcript into MLX chat messages.
    ///
    /// Assistant tool calls and tool results are carried across so the Qwen chat
    /// template can re-render a multi-turn tool exchange; dropping them would
    /// make the model re-issue a call it has already been answered for.
    static func chatMessages(_ messages: [AdmittedChatCompletion.Message]) -> [Chat.Message] {
        messages.map { message in
            switch message.role {
            case .system:
                return .system(message.content)
            case .user:
                return .user(message.content)
            case .assistant:
                let calls = message.toolCalls.map(toolCall)
                return .assistant(message.content, toolCalls: calls.isEmpty ? nil : calls)
            case .tool:
                return .tool(message.content, id: message.toolCallID)
            }
        }
    }

    private static func toolCall(_ payload: ToolCallPayload) -> MLXLMCommon.ToolCall {
        var arguments: [String: MLXLMCommon.JSONValue] = [:]
        if let data = payload.argumentsJSON.data(using: .utf8),
            let decoded = try? JSONDecoder().decode(
                [String: MLXSwiftRuntimeContract.JSONValue].self, from: data)
        {
            arguments = decoded.mapValues(mlxValue)
        }
        return MLXLMCommon.ToolCall(
            function: .init(name: payload.name, arguments: arguments),
            id: payload.id.isEmpty ? nil : payload.id)
    }

    /// Convert an emitted MLX tool call into the OpenAI wire shape.
    ///
    /// The model never supplies a call ID in the Qwen XML format, so one is
    /// minted here; a tool result can only be correlated back if the ID the
    /// client sees is the ID it echoes.
    static func toolCallPayload(
        _ call: MLXLMCommon.ToolCall, fallbackID: @autoclosure () -> String
    ) -> ToolCallPayload {
        let arguments = call.function.arguments.mapValues(contractValue)
        return ToolCallPayload(
            id: call.id ?? fallbackID(),
            name: call.function.name,
            argumentsJSON: ToolCallPayload.encodeArguments(arguments))
    }

    /// Convert OpenAI tool declarations into the `ToolSpec` dictionaries the
    /// chat template renders.
    static func toolSpecs(_ tools: [MLXSwiftRuntimeContract.JSONValue]) -> [ToolSpec]? {
        guard !tools.isEmpty else { return nil }
        return tools.compactMap { tool in
            guard let object = tool.objectValue else { return nil }
            let sendable = object.mapValues { $0.sendableValue }
            return sendable as ToolSpec
        }
    }
}
