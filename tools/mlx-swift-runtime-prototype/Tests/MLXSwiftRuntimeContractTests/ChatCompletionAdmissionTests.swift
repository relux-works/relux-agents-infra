import Foundation
import Testing

@testable import MLXSwiftRuntimeContract

@Suite("chat completion admission")
struct ChatCompletionAdmissionTests {
    static let modelID = "/Users/alexis/src/local-models/Qwen3.8-27B-Uncensored-MLX-8bit"

    static var configuration: ChatCompletionAdmission.Configuration {
        // Mirrors the live Pi profile compat table.
        ChatCompletionAdmission.Configuration(
            modelID: modelID,
            defaultMaxTokens: 2048,
            supportsDeveloperRole: false,
            supportsReasoningEffort: false)
    }

    static func body(_ json: String) throws -> ChatCompletionRequest {
        try ChatCompletionRequest.decode(from: Data(json.utf8))
    }

    static func admit(_ json: String) throws -> AdmittedChatCompletion {
        try ChatCompletionAdmission.admit(try body(json), configuration: configuration)
    }

    static func refusal(_ json: String) -> ChatCompletionRefusal? {
        do {
            _ = try admit(json)
            return nil
        } catch let refusal as ChatCompletionRefusal {
            return refusal
        } catch {
            return nil
        }
    }

    // MARK: - Positive path

    @Test("a well formed request is admitted")
    func admitsValidRequest() throws {
        let admitted = try Self.admit(
            """
            {"model":"\(Self.modelID)","messages":[{"role":"user","content":"hi"}],
             "max_tokens":64,"temperature":0.7,"stream":true,
             "stream_options":{"include_usage":true}}
            """)
        #expect(admitted.modelID == Self.modelID)
        #expect(admitted.messages.map(\.role) == [.user])
        #expect(admitted.messages.first?.content == "hi")
        #expect(admitted.maxTokens == 64)
        #expect(admitted.stream)
        #expect(admitted.includeUsage)
    }

    @Test("max_tokens falls back to the configured default")
    func appliesDefaultMaxTokens() throws {
        let admitted = try Self.admit(
            #"{"model":"\#(Self.modelID)","messages":[{"role":"user","content":"hi"}]}"#)
        #expect(admitted.maxTokens == 2048)
    }

    @Test("multipart text content is joined")
    func joinsTextParts() throws {
        let admitted = try Self.admit(
            """
            {"model":"\(Self.modelID)","messages":[{"role":"user",
             "content":[{"type":"text","text":"a"},{"type":"text","text":"b"}]}]}
            """)
        #expect(admitted.messages.first?.content == "ab")
    }

    // MARK: - Model identity gate

    @Test(
        "a request for any other model is refused with model_not_found",
        arguments: [
            "gpt-4o",
            // Basename only: a `lastPathComponent` comparison would admit this.
            "Qwen3.8-27B-Uncensored-MLX-8bit",
            // Sibling checkout that exists on this host.
            "/Users/alexis/src/local-models/Qwen3.8-27B-Uncensored-MLX-6bit",
            // Trailing separator.
            "/Users/alexis/src/local-models/Qwen3.8-27B-Uncensored-MLX-8bit/",
            // Proper prefixes: a `configured.hasPrefix(requested)` match admits both.
            "/Users/alexis/src/local-models/",
            "/Users/alexis/src/local-models/Qwen3.8-27B-Uncensored-MLX-8bi",
            "/",
            // Superstring: a `requested.contains(configured)` match admits this.
            "/Users/alexis/src/local-models/Qwen3.8-27B-Uncensored-MLX-8bit-draft",
            // Case-shifted: APFS is case-insensitive, the identity check is not.
            "/users/alexis/src/local-models/Qwen3.8-27B-Uncensored-MLX-8bit",
            "",
        ])
    func refusesForeignModel(model: String) throws {
        let refusal = Self.refusal(
            #"{"model":"\#(model)","messages":[{"role":"user","content":"hi"}]}"#)
        #expect(refusal == .unknownModel(requested: model, configured: Self.modelID))
        #expect(refusal?.httpStatus == 404)
        #expect(refusal?.errorCode == "model_not_found")
    }

    @Test("only the exact configured ID is admitted")
    func admitsOnlyExactIdentity() throws {
        // Narrowing checks. Each neighbour below is admitted by a plausible
        // weakening of the gate — prefix match, substring match, basename
        // match, case-insensitive match — and each must still be refused.
        let neighbours = [
            "/Users/alexis/src/local-models/Qwen3.8-27B-Uncensored-MLX-6bit",
            "/Users/alexis/src/local-models/Qwen3.8-27B-Uncensored-MLX-8bi",
            "/Users/alexis/src/local-models/Qwen3.8-27B-Uncensored-MLX-8bitt",
            "/Users/alexis/src/local-models",
            "Qwen3.8-27B-Uncensored-MLX-8bit",
            Self.modelID.uppercased(),
        ]
        for neighbour in neighbours {
            #expect(
                Self.refusal(
                    #"{"model":"\#(neighbour)","messages":[{"role":"user","content":"hi"}]}"#)
                    == .unknownModel(requested: neighbour, configured: Self.modelID),
                "\(neighbour) must not be served by the 8-bit runtime")
        }
        // The exact ID is still admitted, so the gate is not simply closed.
        #expect(
            Self.refusal(
                #"{"model":"\#(Self.modelID)","messages":[{"role":"user","content":"hi"}]}"#)
                == nil)
    }

    // MARK: - Unsupported role gate

    @Test(
        "roles this profile does not support are refused",
        arguments: ["developer", "function", "Assistant", "SYSTEM", "bot"])
    func refusesUnsupportedRole(role: String) throws {
        let refusal = Self.refusal(
            #"{"model":"\#(Self.modelID)","messages":[{"role":"\#(role)","content":"hi"}]}"#)
        #expect(refusal == .unsupportedRole(role))
        #expect(refusal?.httpStatus == 400)
        #expect(refusal?.errorCode == "unsupported_role")
    }

    @Test("a developer role buried after a valid user turn is still refused")
    func refusesDeveloperRoleAnywhere() throws {
        // The gate must scan every message, not just the first: a transcript
        // whose second turn is `developer` would otherwise be silently
        // rewritten into a role the chat template does not define.
        let refusal = Self.refusal(
            """
            {"model":"\(Self.modelID)","messages":[
              {"role":"user","content":"hi"},
              {"role":"developer","content":"be terse"}]}
            """)
        #expect(refusal == .unsupportedRole("developer"))
    }

    @Test(
        "the four supported roles are admitted", arguments: ["system", "user", "assistant", "tool"])
    func admitsSupportedRoles(role: String) throws {
        let admitted = try Self.admit(
            #"{"model":"\#(Self.modelID)","messages":[{"role":"\#(role)","content":"x"}]}"#)
        #expect(admitted.messages.first?.role.rawValue == role)
    }

    // MARK: - reasoning_effort gate

    @Test(
        "reasoning_effort is refused rather than silently ignored",
        arguments: ["low", "medium", "high", "xhigh", "none"])
    func refusesReasoningEffort(effort: String) throws {
        // `supports_reasoning_effort = false` on the profile. Accepting the
        // field and dropping it would let a caller believe it changed the
        // model's reasoning budget when nothing happened.
        let refusal = Self.refusal(
            """
            {"model":"\(Self.modelID)","messages":[{"role":"user","content":"hi"}],
             "reasoning_effort":"\(effort)"}
            """)
        #expect(refusal == .reasoningEffortUnsupported(effort))
        #expect(refusal?.errorCode == "unsupported_parameter")
    }

    @Test("reasoning_effort: null is treated as absent")
    func nullReasoningEffortIsAbsent() throws {
        let admitted = try Self.admit(
            """
            {"model":"\(Self.modelID)","messages":[{"role":"user","content":"hi"}],
             "reasoning_effort":null}
            """)
        #expect(admitted.messages.count == 1)
    }

    @Test("a profile that does support reasoning_effort admits it")
    func honoursCapabilityFlag() throws {
        // Proves the refusal is driven by the profile flag, not hardcoded.
        let configuration = ChatCompletionAdmission.Configuration(
            modelID: Self.modelID, defaultMaxTokens: 2048,
            supportsDeveloperRole: false, supportsReasoningEffort: true)
        let request = try Self.body(
            """
            {"model":"\(Self.modelID)","messages":[{"role":"user","content":"hi"}],
             "reasoning_effort":"low"}
            """)
        let admitted = try ChatCompletionAdmission.admit(request, configuration: configuration)
        #expect(admitted.maxTokens == 2048)
    }

    // MARK: - Bounds

    @Test("non-positive max_tokens is refused", arguments: [0, -1, -4096])
    func refusesNonPositiveMaxTokens(value: Int) throws {
        let refusal = Self.refusal(
            """
            {"model":"\(Self.modelID)","messages":[{"role":"user","content":"hi"}],
             "max_tokens":\(value)}
            """)
        #expect(refusal == .invalidMaxTokens(value))
    }

    /// A number too large for `Int` decodes as a `Double`, and `Int(_:)` on it
    /// is a trap, not a throw. Before this was made total, a single ordinary
    /// request body killed the whole managed runtime — a bypass around every
    /// refusal below, because the process died before any of them could answer.
    ///
    /// Production call site: `Router.chatCompletions(body:)` ->
    /// `ChatCompletionRequest.decode(from:)` -> `optionalInt(_:field:)`, whose
    /// throw the router turns into a bounded `400 invalid_body`.
    @Test(
        "max_tokens outside the Int range is refused, not fatal",
        arguments: ["1e300", "-1e300", "1e19", "-1e19", "9223372036854775808"])
    func refusesUnrepresentableMaxTokens(literal: String) throws {
        let json = """
            {"model":"\(Self.modelID)","messages":[{"role":"user","content":"hi"}],
             "max_tokens":\(literal)}
            """
        #expect(throws: (any Error).self) {
            try ChatCompletionRequest.decode(from: Data(json.utf8))
        }
        do {
            _ = try ChatCompletionRequest.decode(from: Data(json.utf8))
            Issue.record("unrepresentable max_tokens \(literal) was accepted")
        } catch let error as ChatCompletionDecodingError {
            guard case .numberOutOfRange(let field, _) = error else {
                Issue.record("expected numberOutOfRange, got \(error)")
                return
            }
            #expect(field == "max_tokens")
        }
    }

    /// The boundary itself must still be accepted: the gate rejects
    /// unrepresentable values, not merely large ones.
    @Test("max_tokens at the representable boundary still decodes")
    func acceptsRepresentableBoundary() throws {
        let request = try Self.body(
            """
            {"model":"\(Self.modelID)","messages":[{"role":"user","content":"hi"}],
             "max_tokens":9223372036854775807}
            """)
        #expect(request.maxTokens == Int.max)
    }

    /// An integral `Double` inside the range is still a valid integer and must
    /// not be swept up by the range gate.
    @Test("an integral double max_tokens is accepted", arguments: ["64.0", "1e3"])
    func acceptsIntegralDoubleMaxTokens(literal: String) throws {
        let request = try Self.body(
            """
            {"model":"\(Self.modelID)","messages":[{"role":"user","content":"hi"}],
             "max_tokens":\(literal)}
            """)
        #expect(request.maxTokens == Int(Double(literal)!))
    }

    /// A fractional value is a wrong type, not an out-of-range one; the two
    /// refusals must not be collapsed.
    @Test("a fractional max_tokens is refused as a wrong type")
    func refusesFractionalMaxTokens() throws {
        let json = """
            {"model":"\(Self.modelID)","messages":[{"role":"user","content":"hi"}],
             "max_tokens":1.5}
            """
        do {
            _ = try ChatCompletionRequest.decode(from: Data(json.utf8))
            Issue.record("fractional max_tokens was accepted")
        } catch let error as ChatCompletionDecodingError {
            #expect(error == .wrongType(field: "max_tokens", expected: "an integer"))
        }
    }

    @Test("out-of-range temperature is refused", arguments: [-0.1, 2.5, 100.0])
    func refusesInvalidTemperature(value: Double) throws {
        let refusal = Self.refusal(
            """
            {"model":"\(Self.modelID)","messages":[{"role":"user","content":"hi"}],
             "temperature":\(value)}
            """)
        #expect(refusal == .invalidTemperature(value))
    }

    @Test("out-of-range top_p is refused", arguments: [0.0, -1.0, 1.5])
    func refusesInvalidTopP(value: Double) throws {
        let refusal = Self.refusal(
            """
            {"model":"\(Self.modelID)","messages":[{"role":"user","content":"hi"}],
             "top_p":\(value)}
            """)
        #expect(refusal == .invalidTopP(value))
    }

    @Test("an empty transcript is refused")
    func refusesEmptyMessages() throws {
        #expect(Self.refusal(#"{"model":"\#(Self.modelID)","messages":[]}"#) == .emptyMessages)
    }

    @Test("a tool entry that is not a named function is refused")
    func refusesMalformedTool() throws {
        let cases = [
            #"[{"type":"function"}]"#,
            #"[{"type":"retrieval","function":{"name":"x"}}]"#,
            #"[{"function":{"name":"x"}}]"#,
            #"[{"type":"function","function":{}}]"#,
            #"["write_file"]"#,
        ]
        for tools in cases {
            let refusal = Self.refusal(
                """
                {"model":"\(Self.modelID)","messages":[{"role":"user","content":"hi"}],
                 "tools":\(tools)}
                """)
            #expect(refusal == .malformedTool(index: 0), "tools payload \(tools) must be refused")
        }
    }

    @Test("a well formed tool declaration is admitted")
    func admitsValidTool() throws {
        let admitted = try Self.admit(
            """
            {"model":"\(Self.modelID)","messages":[{"role":"user","content":"hi"}],
             "tools":[{"type":"function","function":{"name":"write_file",
             "parameters":{"type":"object","properties":{"path":{"type":"string"}}}}}]}
            """)
        #expect(admitted.tools.count == 1)
    }
}
