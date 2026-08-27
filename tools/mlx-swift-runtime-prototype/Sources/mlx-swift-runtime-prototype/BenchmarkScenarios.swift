import Foundation
import MLXSwiftRuntimeContract

/// The pinned scenario suite, and the code that drives it.
///
/// Every runner here returns a ``RuntimeBenchmark/ScenarioResult`` whose
/// measurements and whose ``RuntimeBenchmark/ScenarioTranscript`` are produced
/// together, from the same requests. There is no path by which a result can
/// carry a number that did not come from an exchange in its own transcript, and
/// no argument through which a caller can supply one.
@MainActor
enum BenchmarkScenarios {
    /// Sent explicitly on every request because the two runtimes do not agree
    /// on defaults: `MLXLMCommon.GenerateParameters` starts at temperature 0.6,
    /// `mlx_lm.server` at 1.0. A benchmark that omitted them would be comparing
    /// two samplers and calling the difference a runtime difference.
    static let temperature = 0.0
    static let topP = 1.0
    static let seed = 1234
    static let defaultMaxOutputTokens = 256

    /// Fixed order, and the capacity probe is last on purpose: it is the run
    /// most likely to exhaust the allocator, and putting it last means an
    /// exhaustion takes down a process whose other measurements are already
    /// recorded rather than one that has not made any.
    static let order = [
        "short_prompt",
        "long_prompt_8k",
        "tool_call",
        "multiturn_prefix_reuse",
        "stability_soak",
        "context_75k",
    ]

    struct Suite {
        let fillerParagraph: String
        let systemPrompt: String
        let scenarios: [String: [String: Any]]

        init(path: String) throws {
            guard let data = try? Data(contentsOf: URL(fileURLWithPath: path)),
                let document = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                let filler = document["filler_paragraph"] as? String,
                let system = document["system_prompt"] as? String,
                let scenarios = document["scenarios"] as? [String: [String: Any]]
            else {
                throw BenchmarkRunCommand.RunError.unusableInput(
                    "prompt suite \(path.debugDescription) could not be read as a suite with "
                        + "filler_paragraph, system_prompt and scenarios")
            }
            fillerParagraph = filler
            systemPrompt = system
            self.scenarios = scenarios
        }

        func prefix(repeats: Int) -> String {
            String(repeating: fillerParagraph, count: repeats)
        }
    }

    /// The request body every scenario builds on.
    ///
    /// Serialized with sorted keys so the transcript's request digest is a
    /// function of the request rather than of dictionary ordering.
    static func payload(
        model: String, messages: [[String: Any]], maxTokens: Int, streaming: Bool = false,
        extra: [String: Any] = [:]
    ) -> [String: Any] {
        var body: [String: Any] = [
            "model": model,
            "messages": messages,
            "max_tokens": maxTokens,
            "temperature": temperature,
            "top_p": topP,
            "seed": seed,
        ]
        if streaming {
            // Both flags, always together. Without `stream_options.include_usage`
            // neither runtime emits a usage frame, and a scenario with no token
            // counts has no prefill and no decode rate — it reports that it
            // measured nothing, which is correct and useless.
            body["stream"] = true
            body["stream_options"] = ["include_usage": true]
        }
        for (key, value) in extra { body[key] = value }
        return body
    }

    static func encode(_ body: [String: Any]) throws -> Data {
        try JSONSerialization.data(withJSONObject: body, options: [.sortedKeys])
    }

    /// Prefill and decode tokens per second from one streamed completion.
    ///
    /// Prefill is prompt tokens divided by time to first token: everything
    /// before the first token is tokenization plus prefill, and on the long
    /// prompts prefill dominates it. Decode divides the tokens *after* the first
    /// by the time after the first, so prefill is not counted twice.
    static func rates(
        _ stream: BenchmarkHTTPDriver.StreamedCompletion
    ) -> (prefill: Double?, decode: Double?) {
        var prefill: Double?
        var decode: Double?
        if let prompt = stream.promptTokens, prompt > 0,
            let ttft = stream.timeToFirstTokenSeconds, ttft > 0
        {
            prefill = Double(prompt) / ttft
        }
        if let completion = stream.completionTokens, completion > 1,
            let ttft = stream.timeToFirstTokenSeconds, stream.totalSeconds > ttft
        {
            decode = Double(completion - 1) / (stream.totalSeconds - ttft)
        }
        return (prefill, decode)
    }

    /// One streamed completion, scored.
    static func single(
        pass: BenchmarkPass, name: String, spec: [String: Any]
    ) async -> RuntimeBenchmark.ScenarioResult {
        var content = (spec["prompt"] as? String) ?? ""
        if let repeats = spec["prefix_repeats"] as? Int {
            content = pass.suite.prefix(repeats: repeats) + "\n\n" + content
        }
        let messages: [[String: Any]] = [
            ["role": "system", "content": pass.suite.systemPrompt],
            ["role": "user", "content": content],
        ]
        let maxTokens = (spec["max_tokens"] as? Int) ?? defaultMaxOutputTokens
        guard
            let body = try? encode(
                payload(
                    model: pass.modelID, messages: messages, maxTokens: maxTokens,
                    streaming: true))
        else {
            return pass.failed(name, "the request body could not be encoded", exchanges: [])
        }
        let started = Date().timeIntervalSince1970
        let stream = await pass.stream(body: body)
        let elapsed = Date().timeIntervalSince1970 - started
        if let failure = stream.failure {
            return pass.failed(name, failure, exchanges: [stream.exchange], wallClock: elapsed)
        }
        guard stream.promptTokens != nil, stream.completionTokens != nil else {
            return pass.failed(
                name, "the stream ended without a usage frame, so nothing was measured",
                exchanges: [stream.exchange], wallClock: elapsed)
        }
        let derived = rates(stream)
        return pass.succeeded(
            name, exchanges: [stream.exchange], promptTokens: stream.promptTokens,
            completionTokens: stream.completionTokens,
            timeToFirstToken: stream.timeToFirstTokenSeconds, prefill: derived.prefill,
            decode: derived.decode, wallClock: elapsed)
    }

    /// Tool-call parity: a well-formed call, not merely a 200.
    ///
    /// A runtime that answers prose to a tool prompt returns 200 and is a parity
    /// failure, so the check is on `finish_reason == "tool_calls"` and on the
    /// arguments parsing into the declared shape.
    static func tool(
        pass: BenchmarkPass, name: String, spec: [String: Any]
    ) async -> RuntimeBenchmark.ScenarioResult {
        let messages: [[String: Any]] = [
            ["role": "system", "content": pass.suite.systemPrompt],
            ["role": "user", "content": (spec["prompt"] as? String) ?? ""],
        ]
        guard let tools = spec["tools"] as? [[String: Any]], let declared = tools.first,
            let function = declared["function"] as? [String: Any],
            let declaredName = function["name"] as? String
        else {
            return pass.failed(name, "the suite declares no tool for this scenario", exchanges: [])
        }
        let maxTokens = (spec["max_tokens"] as? Int) ?? defaultMaxOutputTokens
        guard
            let body = try? encode(
                payload(
                    model: pass.modelID, messages: messages, maxTokens: maxTokens,
                    extra: ["tools": tools]))
        else {
            return pass.failed(name, "the request body could not be encoded", exchanges: [])
        }
        let completion = await pass.post(body: body)
        let exchanges = [completion.exchange]
        let elapsed = completion.totalSeconds
        if let failure = completion.failure {
            return pass.failed(name, failure, exchanges: exchanges, wallClock: elapsed)
        }
        guard
            let document = try? JSONSerialization.jsonObject(with: completion.body)
                as? [String: Any],
            let choice = (document["choices"] as? [[String: Any]])?.first
        else {
            return pass.failed(
                name, "undecodable completion", exchanges: exchanges, wallClock: elapsed)
        }
        let finish = choice["finish_reason"] as? String
        let calls = ((choice["message"] as? [String: Any])?["tool_calls"] as? [[String: Any]]) ?? []
        guard finish == "tool_calls", let call = calls.first else {
            return pass.failed(
                name,
                "finish_reason=\(String(describing: finish)) with \(calls.count) tool call(s); "
                    + "the runtime answered instead of calling the declared tool",
                exchanges: exchanges, wallClock: elapsed)
        }
        let calledFunction = (call["function"] as? [String: Any]) ?? [:]
        guard (calledFunction["name"] as? String) == declaredName else {
            return pass.failed(
                name,
                "called \(String(describing: calledFunction["name"])), not the declared tool",
                exchanges: exchanges, wallClock: elapsed)
        }
        guard let argumentText = calledFunction["arguments"] as? String,
            let argumentData = argumentText.data(using: .utf8),
            let arguments = try? JSONSerialization.jsonObject(with: argumentData) as? [String: Any]
        else {
            return pass.failed(
                name, "tool arguments are not JSON", exchanges: exchanges, wallClock: elapsed)
        }
        let required =
            ((function["parameters"] as? [String: Any])?["required"] as? [String]) ?? []
        let missing = required.filter { arguments[$0] == nil }
        guard missing.isEmpty else {
            return pass.failed(
                name, "tool arguments omitted required key(s) \(missing)", exchanges: exchanges,
                wallClock: elapsed)
        }
        let usage = (document["usage"] as? [String: Any]) ?? [:]
        return pass.succeeded(
            name, exchanges: exchanges, promptTokens: usage["prompt_tokens"] as? Int,
            completionTokens: usage["completion_tokens"] as? Int, timeToFirstToken: nil,
            prefill: nil, decode: nil, wallClock: elapsed)
    }

    /// Turns over one shared long prefix.
    ///
    /// This is where the prompt cache lives. `mlx_lm.server` is deployed with
    /// `--prompt-cache-size 1 --prompt-cache-bytes 8GB` and can serve later
    /// turns without re-prefilling the prefix; the Swift prototype builds a
    /// fresh KV cache per request and cannot. The asymmetry is declared in the
    /// record rather than pinned away, because there is no way to run the
    /// incumbent in its production configuration and also remove it.
    static func multiturn(
        pass: BenchmarkPass, name: String, spec: [String: Any]
    ) async -> RuntimeBenchmark.ScenarioResult {
        let prefix = pass.suite.prefix(repeats: (spec["prefix_repeats"] as? Int) ?? 0)
        let turns = (spec["turns"] as? [String]) ?? []
        let maxTokens = (spec["max_tokens"] as? Int) ?? defaultMaxOutputTokens
        var messages: [[String: Any]] = [
            ["role": "system", "content": pass.suite.systemPrompt]
        ]
        var exchanges: [BenchmarkHTTPDriver.Exchange] = []
        var totalElapsed = 0.0
        var firstTimeToFirstToken: Double?
        var lastTimeToFirstToken: Double?
        var promptTokens: Int?
        var completionTokens = 0
        for (index, turn) in turns.enumerated() {
            let content = index == 0 ? "\(prefix)\n\n\(turn)" : turn
            messages.append(["role": "user", "content": content])
            guard
                let body = try? encode(
                    payload(
                        model: pass.modelID, messages: messages, maxTokens: maxTokens,
                        streaming: true))
            else {
                return pass.failed(
                    name, "turn \(index + 1) could not be encoded", exchanges: exchanges,
                    wallClock: totalElapsed)
            }
            let started = Date().timeIntervalSince1970
            let stream = await pass.stream(body: body)
            totalElapsed += Date().timeIntervalSince1970 - started
            exchanges.append(stream.exchange)
            if let failure = stream.failure {
                return pass.failed(
                    name, "turn \(index + 1) failed: \(failure)", exchanges: exchanges,
                    wallClock: totalElapsed)
            }
            guard stream.promptTokens != nil else {
                return pass.failed(
                    name, "turn \(index + 1) ended without a usage frame", exchanges: exchanges,
                    wallClock: totalElapsed)
            }
            if index == 0 {
                firstTimeToFirstToken = stream.timeToFirstTokenSeconds
                promptTokens = stream.promptTokens
            }
            lastTimeToFirstToken = stream.timeToFirstTokenSeconds
            completionTokens += stream.completionTokens ?? 0
            messages.append(["role": "assistant", "content": stream.content])
        }
        var prefill: Double?
        if let promptTokens, promptTokens > 0, let first = firstTimeToFirstToken, first > 0 {
            prefill = Double(promptTokens) / first
        }
        return pass.succeeded(
            name, exchanges: exchanges, promptTokens: promptTokens,
            completionTokens: completionTokens,
            // The last turn's time to first token is the number this scenario
            // is for: what a caller pays to continue a conversation whose prefix
            // the runtime has already seen.
            timeToFirstToken: lastTimeToFirstToken, prefill: prefill, decode: nil,
            wallClock: totalElapsed)
    }

    /// Sequential distinct requests, watching for drift rather than for speed.
    ///
    /// Prompts differ by index so the prompt cache cannot serve a repeat; what
    /// is being measured is whether a runtime that has served twenty requests is
    /// the same runtime that served the first one.
    static func soak(
        pass: BenchmarkPass, name: String, spec: [String: Any]
    ) async -> RuntimeBenchmark.ScenarioResult {
        let iterations = (spec["iterations"] as? Int) ?? 0
        let template = (spec["prompt_template"] as? String) ?? ""
        let maxTokens = (spec["max_tokens"] as? Int) ?? defaultMaxOutputTokens
        var exchanges: [BenchmarkHTTPDriver.Exchange] = []
        var latencies: [Double] = []
        var footprints: [Int] = []
        var promptTokens = 0
        var completionTokens = 0
        var failures: [String] = []
        let startedAll = Date().timeIntervalSince1970
        for index in 0 ..< iterations {
            let messages: [[String: Any]] = [
                ["role": "system", "content": pass.suite.systemPrompt],
                [
                    "role": "user",
                    "content": template.replacingOccurrences(
                        of: "{index}", with: String(index)),
                ],
            ]
            guard
                let body = try? encode(
                    payload(
                        model: pass.modelID, messages: messages, maxTokens: maxTokens,
                        streaming: true))
            else {
                failures.append("iteration \(index): the request body could not be encoded")
                continue
            }
            let started = Date().timeIntervalSince1970
            let stream = await pass.stream(body: body)
            latencies.append(Date().timeIntervalSince1970 - started)
            exchanges.append(stream.exchange)
            if let failure = stream.failure {
                failures.append("iteration \(index): \(failure)")
                continue
            }
            guard stream.promptTokens != nil else {
                failures.append("iteration \(index): no usage frame")
                continue
            }
            promptTokens += stream.promptTokens ?? 0
            completionTokens += stream.completionTokens ?? 0
            if let value = physicalFootprintBytes(pid: pass.runtimeProcessID) {
                footprints.append(value)
            }
        }
        let elapsed = Date().timeIntervalSince1970 - startedAll
        if !failures.isEmpty {
            return pass.failed(
                name, failures.prefix(5).joined(separator: "; "), exchanges: exchanges,
                wallClock: elapsed)
        }
        // Deliberately NOT reported as decode throughput. `elapsed` covers every
        // request's tokenization, prefill and time to first token as well as its
        // decode, so `completionTokens / elapsed` is aggregate output throughput
        // for the whole soak. Review found the previous labelling presenting it
        // as `decodeTokensPerSecond`, which is a per-token rate this scenario
        // never measured; the scored field stays unmeasured and the aggregate
        // goes into the session's soak block under its own name.
        pass.recordSoak(
            iterations: iterations, elapsed: elapsed, completionTokens: completionTokens,
            latencies: latencies, footprints: footprints)
        return pass.succeeded(
            name, exchanges: exchanges, promptTokens: promptTokens,
            completionTokens: completionTokens, timeToFirstToken: nil, prefill: nil, decode: nil,
            wallClock: elapsed)
    }

    static func run(
        kind: String, pass: BenchmarkPass, name: String, spec: [String: Any]
    ) async -> RuntimeBenchmark.ScenarioResult? {
        switch kind {
        case "single": return await single(pass: pass, name: name, spec: spec)
        case "tool": return await tool(pass: pass, name: name, spec: spec)
        case "multiturn": return await multiturn(pass: pass, name: name, spec: spec)
        case "soak": return await soak(pass: pass, name: name, spec: spec)
        default: return nil
        }
    }
}
