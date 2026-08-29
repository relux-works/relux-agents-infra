import Foundation
import MLXHuggingFace
import MLXLLM
import MLXLMCommon
import MLXSwiftRuntimeContract
import MLXVLM
import Tokenizers

/// Checks everything about a local model that can be checked without
/// materializing its weights.
///
/// This exists because the weight load needs the whole model resident, while the
/// parts most likely to be *incompatible* — the architecture registry entry, the
/// configuration decoder, the tokenizer and the chat template — cost almost
/// nothing to exercise. Running them separately turns "we could not try" into a
/// specific, named result.
///
/// Preflight never proves the model runs. It only proves which of these
/// stages accept it; the load and generation smokes remain the evidence for
/// that.
enum Preflight {
    static func run(options: RuntimeOptions) async -> Int32 {
        var report: [String: MLXSwiftRuntimeContract.JSONValue] = [
            "model_path": .string(options.modelPath),
            "model_id": .string(options.modelID),
            "mlx_swift": .string(mlxSwiftRevision),
            "mlx_swift_lm": .string(mlxSwiftLMRevision),
        ]
        var failed = false

        func record(_ stage: String, _ outcome: String, detail: String? = nil) {
            var entry: [String: MLXSwiftRuntimeContract.JSONValue] = [
                "outcome": .string(outcome)
            ]
            if let detail {
                entry["detail"] = .string(detail)
            }
            report[stage] = .object(entry)
            if outcome == "failed" { failed = true }
        }

        // 0. Metal shader library: whether this build can reach the GPU at all.
        //    Reported rather than fatal, so the CPU-only stages below still run
        //    and still say something useful about a `swift build` product.
        switch MetalShaderLibraryCheck.classify(
            MetalShaderLibraryCheck.inspect(
                roots: MetalShaderLibraryCheck.defaultSearchRoots()))
        {
        case .present(let path):
            record("metal_shader_library", "passed", detail: path)
        case .absent(let searched, let rejected):
            record(
                "metal_shader_library", "failed",
                detail: String(
                    describing: MetalShaderLibraryError.absent(
                        searched: searched, rejected: rejected)))
        case .undetermined(let reasons):
            // Not proven either way. Reported as unknown rather than inferred
            // from a proxy signal.
            record(
                "metal_shader_library", "unknown",
                detail: "could not establish presence: \(reasons.joined(separator: "; "))")
        }

        // 1. Base configuration: the model_type the registries are keyed on.
        let configURL = URL(fileURLWithPath: options.modelPath)
            .appendingPathComponent("config.json")
        guard let configData = try? Data(contentsOf: configURL) else {
            record("base_configuration", "failed", detail: "cannot read \(configURL.path)")
            emit(report, failed: true)
            return 1
        }

        var modelType = ""
        do {
            let base = try JSONDecoder.json5().decode(BaseConfiguration.self, from: configData)
            modelType = base.modelType
            var detail = "model_type=\(base.modelType)"
            if let quantization = base.quantization {
                detail += " quantization=\(quantization.bits)bit/group\(quantization.groupSize)"
            }
            record("base_configuration", "passed", detail: detail)
            report["model_type"] = .string(base.modelType)
        } catch {
            record("base_configuration", "failed", detail: String(describing: error))
            emit(report, failed: true)
            return 1
        }

        // 2. Architecture registry: is this model_type implemented at all?
        let inVLM = await VLMTypeRegistry.shared.contains(modelType)
        let inLLM = await LLMTypeRegistry.shared.contains(modelType)
        if inVLM || inLLM {
            let registries = [inVLM ? "MLXVLM" : nil, inLLM ? "MLXLLM" : nil].compactMap { $0 }
            record(
                "architecture_registry", "passed",
                detail: "\(modelType) implemented by \(registries.joined(separator: ", "))")
        } else {
            record(
                "architecture_registry", "failed",
                detail: "\(modelType) is not registered in MLXVLM or MLXLLM")
        }

        // 3. Concrete configuration decode — the step that fails when an
        //    architecture is registered but its config schema has moved on.
        if inVLM {
            do {
                _ = try JSONDecoder.json5().decode(
                    MLXVLM.Qwen35Configuration.self, from: configData)
                record("vlm_configuration_decode", "passed", detail: "MLXVLM.Qwen35Configuration")
            } catch {
                record("vlm_configuration_decode", "failed", detail: String(describing: error))
            }
        }

        // 4. Tokenizer: reads tokenizer.json only, no weights.
        let tokenizerLoader = #huggingFaceTokenizerLoader()
        var tokenizer: (any MLXLMCommon.Tokenizer)?
        do {
            tokenizer = try await tokenizerLoader.load(
                from: URL(fileURLWithPath: options.modelPath))
            record("tokenizer_load", "passed")
        } catch {
            record("tokenizer_load", "failed", detail: String(describing: error))
        }

        // 5. Chat template, with tools and with the configured reasoning effort.
        //    This is what decides whether generation starts inside `<think>`,
        //    which the reasoning splitter depends on.
        if let tokenizer {
            let messages: [[String: any Sendable]] = [
                ["role": "system", "content": "You are terse."],
                ["role": "user", "content": "What is 2+2?"],
            ]
            let tools: [[String: any Sendable]] = [
                [
                    "type": "function",
                    "function": [
                        "name": "write_file",
                        "description": "Write text to a path.",
                        "parameters": [
                            "type": "object",
                            "properties": ["path": ["type": "string"]],
                        ] as [String: any Sendable],
                    ] as [String: any Sendable],
                ]
            ]
            var context: [String: any Sendable] = [:]
            if let effort = options.reasoningEffort {
                context["reasoning_effort"] = effort
            }

            do {
                let tokens = try tokenizer.applyChatTemplate(
                    messages: messages, tools: tools,
                    additionalContext: context.isEmpty ? nil : context)
                let rendered = tokenizer.decode(tokenIds: tokens, skipSpecialTokens: false)
                record(
                    "chat_template", "passed",
                    detail: "\(tokens.count) tokens with 1 tool declaration")
                report["chat_template_tail"] = .string(String(rendered.suffix(120)))
                // The splitter assumes the prompt ends inside an open think
                // block. Verifying it here means a template change is caught as
                // a preflight failure instead of as garbled `content` at runtime.
                let opensThinkBlock =
                    rendered.hasSuffix("<think>\n") || rendered.hasSuffix("<think>")
                record(
                    "generation_starts_in_reasoning",
                    opensThinkBlock ? "passed" : "failed",
                    detail: opensThinkBlock
                        ? "prompt ends with an open <think> block"
                        : "prompt does not end with an open <think> block; the reasoning splitter's startsInReasoning mode would misfile output"
                )
                report["tool_declaration_rendered"] = .bool(rendered.contains("write_file"))
            } catch {
                record("chat_template", "failed", detail: String(describing: error))
            }
        }

        // 6. Tool-call format inference for this architecture.
        if let format = ToolCallFormat.infer(from: modelType) {
            record("tool_call_format", "passed", detail: format.rawValue)
        } else {
            record(
                "tool_call_format", "failed",
                detail:
                    "no tool-call format is inferred for \(modelType); tool calls would fall back to bare JSON"
            )
        }

        emit(report, failed: failed)
        return failed ? 1 : 0
    }

    private static func emit(
        _ report: [String: MLXSwiftRuntimeContract.JSONValue], failed: Bool
    ) {
        var body = report
        body["preflight"] = .string(failed ? "failed" : "passed")
        StandardOutput.shared.event(RuntimeEvent(name: "preflight", fields: body))
    }
}
