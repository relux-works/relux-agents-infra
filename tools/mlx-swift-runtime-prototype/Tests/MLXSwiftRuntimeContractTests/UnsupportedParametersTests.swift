import Foundation
import Testing

@testable import MLXSwiftRuntimeContract

@Suite("unsupported parameter refusal")
struct UnsupportedParametersTests {
    static let modelID = "/Users/alexis/src/local-models/Qwen3.8-27B-Uncensored-MLX-8bit"

    static var configuration: ChatCompletionAdmission.Configuration {
        ChatCompletionAdmission.Configuration(modelID: modelID, defaultMaxTokens: 2048)
    }

    static func refusal(extra: String) -> ChatCompletionRefusal? {
        let json = """
            {"model":"\(modelID)","messages":[{"role":"user","content":"hi"}]\(extra)}
            """
        do {
            let request = try ChatCompletionRequest.decode(from: Data(json.utf8))
            _ = try ChatCompletionAdmission.admit(request, configuration: configuration)
            return nil
        } catch let refusal as ChatCompletionRefusal {
            return refusal
        } catch {
            Issue.record("unexpected error: \(error)")
            return nil
        }
    }

    @Test(
        "a field the runtime does not implement is refused, not dropped",
        arguments: [
            (#","stop":["\n"]"#, "stop"),
            (#","logit_bias":{"123":-100}"#, "logit_bias"),
            (#","top_logprobs":5"#, "top_logprobs"),
            (#","response_format":{"type":"json_object"}"#, "response_format"),
            (#","chat_template_kwargs":{"enable_thinking":false}"#, "chat_template_kwargs"),
            (#","n":2"#, "n"),
            (#","logprobs":true"#, "logprobs"),
            (#","tool_choice":"required""#, "tool_choice"),
            (#","tool_choice":"none""#, "tool_choice"),
            (#","tool_choice":{"type":"function","function":{"name":"f"}}"#, "tool_choice"),
        ])
    func refusesUnsupportedField(payload: String, field: String) {
        // Each of these would change the answer if honoured. Returning 200
        // while ignoring them reports a constraint that was never applied.
        let refusal = Self.refusal(extra: payload)
        #expect(refusal == .unsupportedParameters([field]))
        #expect(refusal?.errorCode == "unsupported_parameter")
        #expect(refusal?.httpStatus == 400)
    }

    @Test("enable_thinking cannot be flipped per request")
    func refusesThinkingOverride() {
        // Narrowing check for the reasoning split: the `</think>` splitter is
        // built from the startup template state. A request that turned thinking
        // off would have its whole answer filed as `reasoning` and its
        // `content` reported as null.
        #expect(
            Self.refusal(extra: #","chat_template_kwargs":{"enable_thinking":false}"#)
                == .unsupportedParameters(["chat_template_kwargs"]))
    }

    @Test("several unsupported fields are all named")
    func namesEveryUnsupportedField() {
        let refusal = Self.refusal(extra: #","stop":["x"],"n":3,"logprobs":true"#)
        #expect(refusal == .unsupportedParameters(["logprobs", "n", "stop"]))
    }

    @Test(
        "inert forms of the same fields are admitted",
        arguments: [
            #","n":1"#,
            #","logprobs":false"#,
            #","tool_choice":"auto""#,
            #","stop":null"#,
            #","response_format":null"#,
            #","user":"pi""#,
            #","seed":42"#,
        ])
    func admitsInertForms(payload: String) {
        // The gate must reject the class, not every mention of the key: `n: 1`
        // and `logprobs: false` request nothing the runtime fails to deliver.
        #expect(Self.refusal(extra: payload) == nil, "payload \(payload) should be admitted")
    }

    @Test("seed is forwarded rather than refused")
    func forwardsSeed() throws {
        let request = try ChatCompletionRequest.decode(
            from: Data(
                #"{"model":"\#(Self.modelID)","messages":[{"role":"user","content":"hi"}],"seed":7}"#
                    .utf8))
        let admitted = try ChatCompletionAdmission.admit(
            request, configuration: Self.configuration)
        #expect(admitted.seed == 7)
    }

    @Test("detection is driven by the declared field list")
    func detectionMatchesDeclaredList() {
        for field in UnsupportedParameters.alwaysRefused {
            #expect(
                UnsupportedParameters.present(in: [field: .string("x")]) == [field],
                "\(field) must be detected")
            #expect(
                UnsupportedParameters.present(in: [field: .null]).isEmpty,
                "an explicit null \(field) is absent, not present")
        }
    }
}
