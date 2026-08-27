import Foundation
import Testing

@testable import MLXSwiftRuntimeContract

@Suite("serve argument parsing")
struct RuntimeOptionsTests {
    static let modelPath = "/Users/example/models/Qwen"

    static func serve(_ extra: [String] = []) -> [String] {
        ["serve", "--model", modelPath, "--port", "18011"] + extra
    }

    @Test("a model-harness style argv resolves to loopback defaults")
    func parsesHarnessArgv() throws {
        let (subcommand, options) = try RuntimeOptions.parse(
            arguments: [
                "serve", "--model", Self.modelPath, "--host", "127.0.0.1", "--port", "18011",
            ])
        #expect(subcommand == .serve)
        #expect(options.modelPath == Self.modelPath)
        // The advertised ID defaults to the path, matching what mlx_lm.server
        // publishes for a local --model directory.
        #expect(options.modelID == Self.modelPath)
        #expect(options.host == "127.0.0.1")
        #expect(options.port == 18011)
        #expect(options.maxKVSize == nil)
        #expect(options.defaultMaxTokens == 2048)
        #expect(options.reasoningEffort == nil)
    }

    @Test("--model-id overrides the advertised identity")
    func modelIDOverride() throws {
        let (_, options) = try RuntimeOptions.parse(arguments: Self.serve(["--model-id", "qwen"]))
        #expect(options.modelID == "qwen")
        #expect(options.modelPath == Self.modelPath)
    }

    // MARK: - Loopback gate

    @Test(
        "a non-loopback host is refused",
        arguments: ["0.0.0.0", "::", "localhost", "192.168.1.10", "127.0.0.2"])
    func refusesNonLoopbackHost(host: String) {
        #expect(throws: RuntimeOptionsError.nonLoopbackHost(host)) {
            try RuntimeOptions.parse(arguments: Self.serve(["--host", host]))
        }
    }

    @Test("0.0.0.0 is refused even though it is a valid bind address")
    func refusesWildcardBind() {
        // The gate must reject by exact-match, not by "looks local": 0.0.0.0
        // binds every interface and would expose the model off-host.
        var thrown: RuntimeOptionsError?
        do {
            _ = try RuntimeOptions.parse(arguments: Self.serve(["--host", "0.0.0.0"]))
        } catch let error as RuntimeOptionsError {
            thrown = error
        } catch {}
        #expect(thrown == .nonLoopbackHost("0.0.0.0"))
    }

    // MARK: - Other admission failures

    @Test("a relative model path is refused")
    func refusesRelativeModelPath() {
        #expect(throws: RuntimeOptionsError.relativeModelPath("models/Qwen")) {
            try RuntimeOptions.parse(
                arguments: ["serve", "--model", "models/Qwen", "--port", "18011"])
        }
    }

    @Test("out-of-range ports are refused", arguments: ["0", "65536", "-1", "eighteen"])
    func refusesInvalidPort(port: String) {
        #expect(throws: RuntimeOptionsError.invalidPort(port)) {
            try RuntimeOptions.parse(
                arguments: ["serve", "--model", Self.modelPath, "--port", port])
        }
    }

    @Test("--port is required")
    func requiresPort() {
        #expect(throws: RuntimeOptionsError.missingRequiredFlag("--port")) {
            try RuntimeOptions.parse(arguments: ["serve", "--model", Self.modelPath])
        }
    }

    @Test("--model is required")
    func requiresModel() {
        #expect(throws: RuntimeOptionsError.missingRequiredFlag("--model")) {
            try RuntimeOptions.parse(arguments: ["serve", "--port", "18011"])
        }
    }

    @Test("unknown flags are refused rather than ignored")
    func refusesUnknownFlag() {
        #expect(throws: RuntimeOptionsError.unknownFlag("--trust-remote-code")) {
            try RuntimeOptions.parse(arguments: Self.serve(["--trust-remote-code", "true"]))
        }
    }

    @Test("a repeated flag is refused instead of silently last-wins")
    func refusesDuplicateFlag() {
        #expect(throws: RuntimeOptionsError.duplicateFlag("--model")) {
            try RuntimeOptions.parse(arguments: Self.serve(["--model", "/other"]))
        }
    }

    @Test("a flag without a value is refused")
    func refusesDanglingFlag() {
        #expect(throws: RuntimeOptionsError.missingValue("--model-id")) {
            try RuntimeOptions.parse(arguments: Self.serve(["--model-id"]))
        }
    }

    @Test("only the declared subcommands are accepted")
    func refusesUnknownSubcommand() {
        for unknown in ["run", "stress", "Serve", "--model"] {
            #expect(throws: RuntimeOptionsError.unknownSubcommand(unknown)) {
                try RuntimeOptions.parse(arguments: [unknown, "--model", Self.modelPath])
            }
        }
        #expect(throws: RuntimeOptionsError.missingSubcommand) {
            try RuntimeOptions.parse(arguments: [])
        }
    }

    @Test("preflight does not require a port because it binds nothing")
    func preflightNeedsNoPort() throws {
        let (subcommand, options) = try RuntimeOptions.parse(
            arguments: ["preflight", "--model", Self.modelPath])
        #expect(subcommand == .preflight)
        #expect(options.port == 0)
    }

    @Test("serve still requires a port")
    func serveStillRequiresPort() {
        #expect(throws: RuntimeOptionsError.missingRequiredFlag("--port")) {
            try RuntimeOptions.parse(arguments: ["serve", "--model", Self.modelPath])
        }
    }

    @Test("a port given to preflight is still validated")
    func preflightValidatesGivenPort() {
        // The relaxation is "port not required", not "port not checked": the
        // same argv is used for both subcommands, so a bad port must still be
        // caught rather than silently accepted by the preflight path.
        #expect(throws: RuntimeOptionsError.invalidPort("70000")) {
            try RuntimeOptions.parse(
                arguments: ["preflight", "--model", Self.modelPath, "--port", "70000"])
        }
    }

    @Test("preflight enforces the loopback and model-path gates too")
    func preflightKeepsOtherGates() {
        #expect(throws: RuntimeOptionsError.nonLoopbackHost("0.0.0.0")) {
            try RuntimeOptions.parse(
                arguments: ["preflight", "--model", Self.modelPath, "--host", "0.0.0.0"])
        }
        #expect(throws: RuntimeOptionsError.relativeModelPath("models/Qwen")) {
            try RuntimeOptions.parse(arguments: ["preflight", "--model", "models/Qwen"])
        }
    }

    @Test("non-positive integer options are refused", arguments: ["0", "-4"])
    func refusesNonPositiveInteger(value: String) {
        #expect(
            throws: RuntimeOptionsError.nonPositiveInteger(
                flag: "--max-kv-size", value: Int(value)!)
        ) {
            try RuntimeOptions.parse(arguments: Self.serve(["--max-kv-size", value]))
        }
    }

    @Test(
        "a reasoning effort the chat template does not define is refused",
        arguments: ["high", "none", "off", "XHIGH", ""])
    func refusesUnsupportedReasoningEffort(effort: String) {
        // The bundled Qwen3.5 template raises on anything outside
        // {low, medium, xhigh}; catching it here turns a mid-request template
        // exception into a startup refusal.
        #expect(throws: RuntimeOptionsError.unsupportedReasoningEffort(effort)) {
            try RuntimeOptions.parse(arguments: Self.serve(["--reasoning-effort", effort]))
        }
    }

    @Test("supported reasoning efforts pass", arguments: ["low", "medium", "xhigh"])
    func acceptsSupportedReasoningEffort(effort: String) throws {
        let (_, options) = try RuntimeOptions.parse(
            arguments: Self.serve(["--reasoning-effort", effort]))
        #expect(options.reasoningEffort == effort)
    }
    // MARK: - Model factory and prefill chunk

    @Test("the default factory preference is text-only")
    func defaultsToTextOnlyFactory() throws {
        let (_, options) = try RuntimeOptions.parse(arguments: Self.serve([]))
        // Not cosmetic. Both MLX Swift LM factories accept this model's
        // `model_type` `qwen3_5`, and only `MLXLLM`'s evaluates the prompt in
        // chunks; the registry's own order would silently pick the other one.
        #expect(options.modelFactory == .textOnly)
    }

    @Test(
        "every factory preference parses to itself",
        arguments: RuntimeOptions.ModelFactoryPreference.allCases)
    func parsesEveryFactoryPreference(
        preference: RuntimeOptions.ModelFactoryPreference
    ) throws {
        let (_, options) = try RuntimeOptions.parse(
            arguments: Self.serve(["--model-factory", preference.rawValue]))
        #expect(options.modelFactory == preference)
    }

    @Test("an unknown factory preference is refused at parse time")
    func refusesUnknownFactoryPreference() {
        #expect(throws: RuntimeOptionsError.unsupportedModelFactory("vlm")) {
            try RuntimeOptions.parse(arguments: Self.serve(["--model-factory", "vlm"]))
        }
    }

    @Test("the prefill chunk is unset unless the launch states it")
    func leavesPrefillStepUnsetByDefault() throws {
        let (_, options) = try RuntimeOptions.parse(arguments: Self.serve([]))
        // `nil` rather than 512. The value MLXLMCommon defaults to is not this
        // runtime's claim about what it ran, and the comparison gate refuses a
        // benchmark record whose launch left the condition unstated.
        #expect(options.prefillStepSize == nil)
    }

    @Test("a stated prefill chunk is carried through")
    func parsesPrefillStep() throws {
        let (_, options) = try RuntimeOptions.parse(
            arguments: Self.serve(["--prefill-step-size", "2048"]))
        #expect(options.prefillStepSize == 2048)
    }

    @Test("a non-positive prefill chunk is refused")
    func refusesNonPositivePrefillStep() {
        #expect(
            throws: RuntimeOptionsError.nonPositiveInteger(flag: "--prefill-step-size", value: 0)
        ) {
            try RuntimeOptions.parse(arguments: Self.serve(["--prefill-step-size", "0"]))
        }
    }

    @Test("the text-only preferences put the chunked-prefill factory first")
    func ordersTextFactoryFirst() {
        #expect(
            RuntimeOptions.ModelFactoryPreference.textOnly.factoryOrder
                == ["MLXLLM.LLMModelFactory", "MLXVLM.VLMModelFactory"])
        // Strict has no fallback on purpose: a directory the text factory
        // refuses must fail the load rather than quietly becoming a vision
        // model whose prepare step discards `windowSize`.
        #expect(
            RuntimeOptions.ModelFactoryPreference.textOnlyStrict.factoryOrder
                == ["MLXLLM.LLMModelFactory"])
        #expect(
            RuntimeOptions.ModelFactoryPreference.visionFirst.factoryOrder
                == ["MLXVLM.VLMModelFactory", "MLXLLM.LLMModelFactory"])
    }

    @Test("no preference offers a factory this executable does not link")
    func namesOnlyLinkedFactories() {
        let known: Set<String> = ["MLXLLM.LLMModelFactory", "MLXVLM.VLMModelFactory"]
        for preference in RuntimeOptions.ModelFactoryPreference.allCases {
            #expect(!preference.factoryOrder.isEmpty)
            #expect(Set(preference.factoryOrder).isSubset(of: known))
            // A duplicate would make the loader try the same factory twice and
            // report its refusal twice, which reads as two architectures
            // refusing the directory.
            #expect(Set(preference.factoryOrder).count == preference.factoryOrder.count)
        }
    }

}
