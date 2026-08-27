import Foundation

/// A chat request that passed every admission gate, normalized for generation.
public struct AdmittedChatCompletion: Sendable, Equatable {
    public enum Role: String, Sendable {
        case system
        case user
        case assistant
        case tool
    }

    public struct Message: Sendable, Equatable {
        public let role: Role
        public let content: String
        public let toolCalls: [ToolCallPayload]
        public let toolCallID: String?
    }

    public let modelID: String
    public let messages: [Message]
    public let maxTokens: Int
    public let temperature: Double?
    public let topP: Double?
    public let stream: Bool
    public let includeUsage: Bool
    public let tools: [JSONValue]
    public let seed: UInt64?
}

/// Why a chat request was refused.
///
/// Each case carries the HTTP status and OpenAI error `type`/`code` the
/// prototype answers with, so the refusal reaching the client and the refusal
/// asserted in tests cannot drift apart.
public enum ChatCompletionRefusal: Error, Equatable, CustomStringConvertible {
    /// `model` did not equal the single model this process loaded.
    case unknownModel(requested: String, configured: String)
    /// A role this profile declares unsupported (`supports_developer_role = false`).
    case unsupportedRole(String)
    /// `reasoning_effort` is not honoured (`supports_reasoning_effort = false`).
    case reasoningEffortUnsupported(String)
    /// Fields the runtime does not implement and refuses to silently drop.
    case unsupportedParameters([String])
    case emptyMessages
    case invalidMaxTokens(Int)
    case invalidTemperature(Double)
    case invalidTopP(Double)
    /// A tool entry that is not a `{"type":"function","function":{...}}` object.
    case malformedTool(index: Int)

    public var httpStatus: Int {
        switch self {
        case .unknownModel: return 404
        default: return 400
        }
    }

    public var errorType: String {
        switch self {
        case .unknownModel: return "invalid_request_error"
        default: return "invalid_request_error"
        }
    }

    public var errorCode: String {
        switch self {
        case .unknownModel: return "model_not_found"
        case .unsupportedRole: return "unsupported_role"
        case .reasoningEffortUnsupported: return "unsupported_parameter"
        case .unsupportedParameters: return "unsupported_parameter"
        case .emptyMessages: return "empty_messages"
        case .invalidMaxTokens: return "invalid_max_tokens"
        case .invalidTemperature: return "invalid_temperature"
        case .invalidTopP: return "invalid_top_p"
        case .malformedTool: return "invalid_tool"
        }
    }

    public var description: String {
        switch self {
        case .unknownModel(let requested, let configured):
            return
                "model \(requested.debugDescription) is not available; this runtime serves only \(configured.debugDescription)"
        case .unsupportedRole(let role):
            return
                "role \(role.debugDescription) is not supported by this profile; supported roles are system, user, assistant, tool"
        case .reasoningEffortUnsupported(let value):
            return
                "reasoning_effort \(value.debugDescription) is not supported; this runtime selects reasoning effort through the chat template at startup"
        case .unsupportedParameters(let fields):
            return
                "this runtime does not implement \(fields.map { $0.debugDescription }.joined(separator: ", ")); the request is refused rather than answered as if the field had been applied"
        case .emptyMessages:
            return "messages must contain at least one message"
        case .invalidMaxTokens(let value):
            return "max_tokens must be a positive integer, got \(value)"
        case .invalidTemperature(let value):
            return "temperature must be between 0 and 2, got \(value)"
        case .invalidTopP(let value):
            return "top_p must be greater than 0 and at most 1, got \(value)"
        case .malformedTool(let index):
            return "tools[\(index)] must be an object with type \"function\" and a function object"
        }
    }
}

/// The admission gate for `POST /v1/chat/completions`.
///
/// Production call site: `ChatCompletionsRoute.respond(to:)` in the
/// `mlx-swift-runtime-prototype` executable target, which calls
/// ``ChatCompletionAdmission/admit(_:configuration:)`` before any tokenizer or
/// model work happens and turns a thrown ``ChatCompletionRefusal`` into the
/// HTTP error response.
public enum ChatCompletionAdmission {
    public struct Configuration: Sendable, Equatable {
        /// The one model ID this process loaded and may answer for.
        public let modelID: String
        public let defaultMaxTokens: Int
        /// Mirrors `supports_developer_role` on the Pi profile.
        public let supportsDeveloperRole: Bool
        /// Mirrors `supports_reasoning_effort` on the Pi profile.
        public let supportsReasoningEffort: Bool

        public init(
            modelID: String,
            defaultMaxTokens: Int,
            supportsDeveloperRole: Bool = false,
            supportsReasoningEffort: Bool = false
        ) {
            self.modelID = modelID
            self.defaultMaxTokens = defaultMaxTokens
            self.supportsDeveloperRole = supportsDeveloperRole
            self.supportsReasoningEffort = supportsReasoningEffort
        }
    }

    public static func admit(
        _ request: ChatCompletionRequest, configuration: Configuration
    ) throws -> AdmittedChatCompletion {
        guard request.model == configuration.modelID else {
            throw ChatCompletionRefusal.unknownModel(
                requested: request.model, configured: configuration.modelID)
        }

        if let effort = request.reasoningEffort, !configuration.supportsReasoningEffort {
            throw ChatCompletionRefusal.reasoningEffortUnsupported(effort)
        }

        if !request.unsupportedFields.isEmpty {
            throw ChatCompletionRefusal.unsupportedParameters(request.unsupportedFields)
        }

        guard !request.messages.isEmpty else {
            throw ChatCompletionRefusal.emptyMessages
        }

        let messages = try request.messages.map { message -> AdmittedChatCompletion.Message in
            guard let role = AdmittedChatCompletion.Role(rawValue: message.rawRole) else {
                // `developer` is a real OpenAI role, so it decodes fine and is
                // refused here rather than being quietly folded into `system`.
                throw ChatCompletionRefusal.unsupportedRole(message.rawRole)
            }
            return AdmittedChatCompletion.Message(
                role: role,
                content: message.content,
                toolCalls: message.toolCalls,
                toolCallID: message.toolCallID)
        }

        let maxTokens = request.maxTokens ?? configuration.defaultMaxTokens
        guard maxTokens > 0 else {
            throw ChatCompletionRefusal.invalidMaxTokens(maxTokens)
        }

        if let temperature = request.temperature, !(0 ... 2).contains(temperature) {
            throw ChatCompletionRefusal.invalidTemperature(temperature)
        }
        if let topP = request.topP, !(topP > 0 && topP <= 1) {
            throw ChatCompletionRefusal.invalidTopP(topP)
        }

        for (index, tool) in request.tools.enumerated() {
            guard let object = tool.objectValue,
                case .string("function")? = object["type"],
                let function = object["function"]?.objectValue,
                case .string? = function["name"]
            else {
                throw ChatCompletionRefusal.malformedTool(index: index)
            }
        }

        return AdmittedChatCompletion(
            modelID: configuration.modelID,
            messages: messages,
            maxTokens: maxTokens,
            temperature: request.temperature,
            topP: request.topP,
            stream: request.stream,
            includeUsage: request.includeUsage,
            tools: request.tools,
            seed: request.seed)
    }
}
