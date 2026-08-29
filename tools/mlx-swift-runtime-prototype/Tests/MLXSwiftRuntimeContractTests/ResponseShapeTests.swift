import Foundation
import Testing

@testable import MLXSwiftRuntimeContract

@Suite("OpenAI response shape")
struct ResponseShapeTests {
    static let modelID = "/Users/alexis/src/local-models/Qwen3.8-27B-Uncensored-MLX-8bit"

    static var builder: ChatCompletionResponseBuilder {
        ChatCompletionResponseBuilder(
            requestID: "chatcmpl-test", modelID: modelID, created: 1_756_000_000,
            systemFingerprint: "prototype")
    }

    static func object(_ value: JSONValue) -> [String: JSONValue] { value.objectValue ?? [:] }

    static func firstChoice(_ body: JSONValue) -> [String: JSONValue] {
        guard case .array(let choices)? = object(body)["choices"], let first = choices.first
        else { return [:] }
        return object(first)
    }

    @Test("a plain completion carries content, usage and a stop reason")
    func nonStreamingShape() {
        let body = Self.builder.completion(
            content: "4", reasoning: "2+2", toolCalls: [], finishReason: .stop,
            usage: CompletionUsage(promptTokens: 11, completionTokens: 3))
        #expect(Self.object(body)["object"] == .string("chat.completion"))
        #expect(Self.object(body)["model"] == .string(Self.modelID))
        #expect(Self.object(body)["system_fingerprint"] == .string("prototype"))

        let choice = Self.firstChoice(body)
        #expect(choice["finish_reason"] == .string("stop"))
        let message = Self.object(choice["message"] ?? .null)
        #expect(message["role"] == .string("assistant"))
        #expect(message["content"] == .string("4"))
        // mlx_lm.server publishes reasoning under `reasoning`; the Pi profile is
        // configured against that runtime, so the key must match.
        #expect(message["reasoning"] == .string("2+2"))
        #expect(message["tool_calls"] == nil)

        let usage = Self.object(Self.object(body)["usage"] ?? .null)
        #expect(usage["prompt_tokens"] == .int(11))
        #expect(usage["completion_tokens"] == .int(3))
        #expect(usage["total_tokens"] == .int(14))
    }

    @Test("empty content is an explicit null, not an omitted field")
    func emptyContentIsNull() {
        let body = Self.builder.completion(
            content: "", reasoning: "", toolCalls: [], finishReason: .stop,
            usage: CompletionUsage(promptTokens: 1, completionTokens: 0))
        let message = Self.object(Self.firstChoice(body)["message"] ?? .null)
        #expect(message["content"] == .some(.null))
        #expect(message["reasoning"] == nil)
    }

    @Test("a tool call completion finishes with tool_calls and a JSON argument string")
    func toolCallShape() throws {
        let call = ToolCallPayload(
            id: "call_1", name: "write_file",
            argumentsJSON: ToolCallPayload.encodeArguments([
                "path": .string("/tmp/a.txt"), "content": .string("hi"),
            ]))
        let body = Self.builder.completion(
            content: "", reasoning: "", toolCalls: [call], finishReason: .toolCalls,
            usage: CompletionUsage(promptTokens: 20, completionTokens: 12))

        #expect(Self.firstChoice(body)["finish_reason"] == .string("tool_calls"))
        let message = Self.object(Self.firstChoice(body)["message"] ?? .null)
        guard case .array(let calls)? = message["tool_calls"], let first = calls.first else {
            Issue.record("expected tool_calls")
            return
        }
        let entry = Self.object(first)
        #expect(entry["id"] == .string("call_1"))
        #expect(entry["type"] == .string("function"))
        let function = Self.object(entry["function"] ?? .null)
        #expect(function["name"] == .string("write_file"))
        // arguments is a JSON *string*, per the OpenAI contract.
        guard case .string(let arguments)? = function["arguments"] else {
            Issue.record("arguments must be a JSON string")
            return
        }
        let decoded = try JSONDecoder().decode(
            [String: JSONValue].self, from: Data(arguments.utf8))
        #expect(decoded["path"] == .string("/tmp/a.txt"))
        #expect(decoded["content"] == .string("hi"))
    }

    @Test("tool-call arguments do not escape path separators")
    func argumentsKeepSlashes() {
        let encoded = ToolCallPayload.encodeArguments(["path": .string("/tmp/a.txt")])
        #expect(encoded.contains("/tmp/a.txt"))
        #expect(!encoded.contains("\\/"))
    }

    @Test("streaming deltas omit finish_reason until the final chunk")
    func streamingShape() {
        let delta = Self.builder.chunk(
            content: "he", reasoning: "", toolCalls: [], finishReason: nil)
        #expect(Self.object(delta)["object"] == .string("chat.completion.chunk"))
        #expect(Self.firstChoice(delta)["finish_reason"] == .some(.null))
        #expect(Self.object(Self.firstChoice(delta)["delta"] ?? .null)["content"] == .string("he"))

        let final = Self.builder.chunk(
            content: "", reasoning: "", toolCalls: [], finishReason: .length)
        #expect(Self.firstChoice(final)["finish_reason"] == .string("length"))
        #expect(Self.object(Self.firstChoice(final)["delta"] ?? .null)["content"] == nil)
    }

    @Test("streaming tool-call deltas carry an index")
    func streamingToolCallIndex() {
        let call = ToolCallPayload(id: "call_1", name: "read", argumentsJSON: "{}").indexed(2)
        let chunk = Self.builder.chunk(
            content: "", reasoning: "", toolCalls: [call], finishReason: nil)
        guard
            case .array(let calls)? =
                Self.object(Self.firstChoice(chunk)["delta"] ?? .null)["tool_calls"],
            let first = calls.first
        else {
            Issue.record("expected streaming tool_calls")
            return
        }
        #expect(Self.object(first)["index"] == .int(2))
    }

    @Test("the streaming usage packet has no choices")
    func streamingUsagePacket() {
        // `supports_usage_in_streaming = true`: the trailing usage frame is part
        // of the contract and must not carry a phantom empty choice.
        let body = Self.builder.streamingUsage(
            CompletionUsage(promptTokens: 5, completionTokens: 7))
        #expect(Self.object(body)["choices"] == .array([]))
        let usage = Self.object(Self.object(body)["usage"] ?? .null)
        #expect(usage["total_tokens"] == .int(12))
    }

    @Test("server-sent events are framed with a blank-line terminator")
    func sseFraming() throws {
        let frame = try ServerSentEvent.data(.object(["a": .int(1)]))
        #expect(frame.hasPrefix("data: "))
        #expect(frame.hasSuffix("\n\n"))
        #expect(!frame.dropLast(2).contains("\n"))
        #expect(ServerSentEvent.done == "data: [DONE]\n\n")
    }

    @Test("finish reasons use the OpenAI spellings")
    func finishReasonSpellings() {
        #expect(FinishReason.stop.rawValue == "stop")
        #expect(FinishReason.length.rawValue == "length")
        #expect(FinishReason.toolCalls.rawValue == "tool_calls")
    }
}

@Suite("runtime events")
struct RuntimeEventTests {
    @Test("a load event reports seconds and both memory readings")
    func modelLoadedEvent() throws {
        let event = RuntimeEvent.modelLoaded(
            modelID: "/models/Qwen", modelPath: "/models/Qwen", loadSeconds: 12.3456,
            residentBytes: 30_000_000_000, physicalFootprintBytes: 31_000_000_000,
            modelType: "qwen3_5")
        let fields = event.json.objectValue ?? [:]
        #expect(fields["event"] == .string("model_loaded"))
        #expect(fields["model_type"] == .string("qwen3_5"))
        #expect(fields["load_seconds"] == .double(12.346))
        #expect(fields["resident_bytes"] == .int(30_000_000_000))
        #expect(fields["physical_footprint_bytes"] == .int(31_000_000_000))
    }

    @Test("an unavailable memory reading is null, never zero")
    func unknownMemoryIsNull() {
        // A zero would be read downstream as a measured value. `task_info`
        // failing means the footprint is unknown, and that is what is reported.
        let event = RuntimeEvent.modelLoaded(
            modelID: "/models/Qwen", modelPath: "/models/Qwen", loadSeconds: 1,
            residentBytes: nil, physicalFootprintBytes: nil, modelType: "qwen3_5")
        let fields = event.json.objectValue ?? [:]
        #expect(fields["resident_bytes"] == .some(.null))
        #expect(fields["resident_mib"] == .some(.null))
        #expect(fields["physical_footprint_bytes"] == .some(.null))
        #expect(fields["resident_bytes"] != .some(.int(0)))
    }

    @Test("an event serializes to a single line")
    func eventIsOneLine() throws {
        let line = try RuntimeEvent(name: "ready", fields: ["port": .int(18011)]).line()
        #expect(!line.contains("\n"))
        #expect(line.contains("\"event\":\"ready\""))
    }
}
