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
    nonisolated static let order = [
        "short_prompt",
        "long_prompt_8k",
        "tool_call",
        "multiturn_prefix_reuse",
        "stability_soak",
        "context_75k",
    ]

    /// The pinned suite, validated in full before this type exists.
    ///
    /// F5: the readers that used to live in the scenario drivers are gone. This
    /// initializer either produces a suite every field of which the gate
    /// understood, or it throws — and it is called at the top of
    /// `BenchmarkRunCommand.execute`, before the session directory is created
    /// and long before either runtime is launched, so a malformed suite cannot
    /// reach a measurement.
    struct Suite {
        let validated: PromptSuiteSchema.Suite

        var fillerParagraph: String { validated.fillerParagraph }
        var systemPrompt: String { validated.systemPrompt }
        var scenarios: [String: PromptSuiteSchema.Scenario] { validated.scenarios }

        init(path: String) throws {
            guard let data = try? Data(contentsOf: URL(fileURLWithPath: path)) else {
                throw BenchmarkRunCommand.RunError.unusableInput(
                    "prompt suite \(path.debugDescription) could not be read")
            }
            do {
                // The driver's own scenario list is what "known" means: a
                // scenario this loop never visits is pinned into both records
                // by the suite digest and then silently not run.
                validated = try PromptSuiteSchema.validate(
                    data: data, knownScenarioNames: Set(BenchmarkScenarios.order))
            } catch let failure as PromptSuiteSchema.Failure {
                throw BenchmarkRunCommand.RunError.unusableInput(
                    "prompt suite \(path.debugDescription) was refused before any runtime was "
                        + "launched -- \(failure)")
            }
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
            let first = stream.timeToFirstTokenSeconds,
            let last = stream.timeToLastTokenSeconds, last > first
        {
            decode = Double(completion - 1) / (last - first)
        }
        return (prefill, decode)
    }

    /// One streamed completion, scored.
    static func single(
        pass: BenchmarkPass, name: String, maxTokens: Int, prompt: String, prefixRepeats: Int?
    ) async -> RuntimeBenchmark.ScenarioResult {
        // No cast, no default. `prompt` is a validated non-empty string and
        // `prefixRepeats` is either a validated count or a field the suite did
        // not write; the string `"2027"` cannot arrive here as either.
        var content = prompt
        if let prefixRepeats {
            content = pass.suite.prefix(repeats: prefixRepeats) + "\n\n" + content
        }
        let messages: [[String: Any]] = [
            ["role": "system", "content": pass.suite.systemPrompt],
            ["role": "user", "content": content],
        ]
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
            return pass.failed(
                name, failure, exchanges: [stream.exchange], wallClock: elapsed,
                cacheReuse: cacheReuseObservation([stream.cacheReuse]))
        }
        guard stream.promptTokens != nil, stream.completionTokens != nil else {
            return pass.failed(
                name, "the stream ended without a usage frame, so nothing was measured",
                exchanges: [stream.exchange], wallClock: elapsed,
                cacheReuse: cacheReuseObservation([stream.cacheReuse]))
        }
        let derived = rates(stream)
        return pass.succeeded(
            name, exchanges: [stream.exchange], promptTokens: stream.promptTokens,
            completionTokens: stream.completionTokens,
            timeToFirstToken: stream.timeToFirstTokenSeconds, prefill: derived.prefill,
            decode: derived.decode, wallClock: elapsed,
            cacheReuse: cacheReuseObservation([stream.cacheReuse]))
    }

    /// Tool-call parity: a well-formed call, not merely a 200.
    ///
    /// A runtime that answers prose to a tool prompt returns 200 and is a parity
    /// failure, so the check is on `finish_reason == "tool_calls"` and on the
    /// arguments parsing into the declared shape.
    static func tool(
        pass: BenchmarkPass, name: String, maxTokens: Int, prompt: String,
        tools: [PromptSuiteSchema.ToolDeclaration]
    ) async -> RuntimeBenchmark.ScenarioResult {
        let messages: [[String: Any]] = [
            ["role": "system", "content": pass.suite.systemPrompt],
            ["role": "user", "content": prompt],
        ]
        // Validated non-empty, so there is no "the suite declares no tool"
        // failure to report at measurement time any more: that shape is
        // refused before either runtime starts.
        let declared = tools[0]
        let declaredName = declared.name
        guard
            let body = try? encode(
                payload(
                    model: pass.modelID, messages: messages, maxTokens: maxTokens,
                    // Forwarded verbatim: the suite's own bytes, not a
                    // re-shaped copy.
                    extra: ["tools": tools.map { $0.value.sendableValue }]))
        else {
            return pass.failed(name, "the request body could not be encoded", exchanges: [])
        }
        let completion = await pass.post(body: body)
        let exchanges = [completion.exchange]
        let elapsed = completion.totalSeconds
        if let failure = completion.failure {
            return pass.failed(
                name, failure, exchanges: exchanges, wallClock: elapsed,
                cacheReuse: cacheReuseObservation([completion.cacheReuse]))
        }
        guard
            let document = try? JSONSerialization.jsonObject(with: completion.body)
                as? [String: Any],
            let choice = (document["choices"] as? [[String: Any]])?.first
        else {
            return pass.failed(
                name, "undecodable completion", exchanges: exchanges, wallClock: elapsed,
                cacheReuse: cacheReuseObservation([completion.cacheReuse]))
        }
        let finish = choice["finish_reason"] as? String
        let calls = ((choice["message"] as? [String: Any])?["tool_calls"] as? [[String: Any]]) ?? []
        guard finish == "tool_calls", let call = calls.first else {
            return pass.failed(
                name,
                "finish_reason=\(String(describing: finish)) with \(calls.count) tool call(s); "
                    + "the runtime answered instead of calling the declared tool",
                exchanges: exchanges, wallClock: elapsed,
                cacheReuse: cacheReuseObservation([completion.cacheReuse]))
        }
        let calledFunction = (call["function"] as? [String: Any]) ?? [:]
        guard (calledFunction["name"] as? String) == declaredName else {
            return pass.failed(
                name,
                "called \(String(describing: calledFunction["name"])), not the declared tool",
                exchanges: exchanges, wallClock: elapsed,
                cacheReuse: cacheReuseObservation([completion.cacheReuse]))
        }
        guard let argumentText = calledFunction["arguments"] as? String,
            let argumentData = argumentText.data(using: .utf8),
            let arguments = try? JSONSerialization.jsonObject(with: argumentData) as? [String: Any]
        else {
            return pass.failed(
                name, "tool arguments are not JSON", exchanges: exchanges, wallClock: elapsed,
                cacheReuse: cacheReuseObservation([completion.cacheReuse]))
        }
        let missing = declared.requiredArguments.filter { arguments[$0] == nil }
        guard missing.isEmpty else {
            return pass.failed(
                name, "tool arguments omitted required key(s) \(missing)", exchanges: exchanges,
                wallClock: elapsed,
                cacheReuse: cacheReuseObservation([completion.cacheReuse]))
        }
        let usage = (document["usage"] as? [String: Any]) ?? [:]
        return pass.succeeded(
            name, exchanges: exchanges, promptTokens: usage["prompt_tokens"] as? Int,
            completionTokens: usage["completion_tokens"] as? Int, timeToFirstToken: nil,
            prefill: nil, decode: nil, wallClock: elapsed,
            cacheReuse: cacheReuseObservation([completion.cacheReuse]))
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
        pass: BenchmarkPass, name: String, maxTokens: Int, prefixRepeats: Int?, turns: [String]
    ) async -> RuntimeBenchmark.ScenarioResult {
        let prefix = prefixRepeats.map { pass.suite.prefix(repeats: $0) } ?? ""
        var messages: [[String: Any]] = [
            ["role": "system", "content": pass.suite.systemPrompt]
        ]
        var exchanges: [BenchmarkHTTPDriver.Exchange] = []
        var totalElapsed = 0.0
        var firstTimeToFirstToken: Double?
        var lastTimeToFirstToken: Double?
        var promptTokens: Int?
        var completionTokens = 0
        var cacheReads: [BenchmarkHTTPDriver.CacheReuseRead] = []
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
                    wallClock: totalElapsed,
                    cacheReuse: cacheReuseObservation(cacheReads))
            }
            let started = Date().timeIntervalSince1970
            let stream = await pass.stream(body: body)
            totalElapsed += Date().timeIntervalSince1970 - started
            exchanges.append(stream.exchange)
            cacheReads.append(stream.cacheReuse)
            if let failure = stream.failure {
                return pass.failed(
                    name, "turn \(index + 1) failed: \(failure)", exchanges: exchanges,
                    wallClock: totalElapsed,
                    cacheReuse: cacheReuseObservation(cacheReads))
            }
            guard stream.promptTokens != nil else {
                return pass.failed(
                    name, "turn \(index + 1) ended without a usage frame", exchanges: exchanges,
                    wallClock: totalElapsed,
                    cacheReuse: cacheReuseObservation(cacheReads))
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
            wallClock: totalElapsed,
            cacheReuse: cacheReuseObservation(cacheReads))
    }

    /// Convert response telemetry into one sealed scenario fact. A positive
    /// reported count proves a hit. A miss is stronger: every turn must report
    /// zero, otherwise absence/malformed telemetry remains unknown.
    static func cacheReuseObservation(
        _ reads: [BenchmarkHTTPDriver.CacheReuseRead]
    ) -> RuntimeBenchmark.CacheReuseObservation {
        let reported = reads.compactMap { read -> Int? in
            guard case .reported(let cached) = read else { return nil }
            return cached
        }
        if reported.contains(where: { $0 > 0 }) {
            return .reported(cachedPromptTokens: reported)
        }
        if reads.contains(.malformed) {
            return .unknown("cached-token telemetry was malformed")
        }
        if reads.contains(.notReported) || reported.count != reads.count || reads.isEmpty {
            return .unknown("cached-token telemetry was not reported for every turn")
        }
        return .reported(cachedPromptTokens: reported)
    }

    /// Sequential distinct requests, watching for drift rather than for speed.
    ///
    /// Prompts differ by index so the prompt cache cannot serve a repeat; what
    /// is being measured is whether a runtime that has served twenty requests is
    /// the same runtime that served the first one.
    static func soak(
        pass: BenchmarkPass, name: String, maxTokens: Int, iterations: Int, template: String
    ) async -> RuntimeBenchmark.ScenarioResult {
        var exchanges: [BenchmarkHTTPDriver.Exchange] = []
        var latencies: [Double] = []
        var memoryReads: [RuntimeMemorySampleRead] = []
        var promptTokens = 0
        var completionTokens = 0
        var failures: [String] = []
        var cacheReads: [BenchmarkHTTPDriver.CacheReuseRead] = []
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
            cacheReads.append(stream.cacheReuse)
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
            memoryReads.append(pass.currentMemoryReading())
        }
        let elapsed = Date().timeIntervalSince1970 - startedAll
        if !failures.isEmpty {
            return pass.failed(
                name, failures.prefix(5).joined(separator: "; "), exchanges: exchanges,
                wallClock: elapsed, cacheReuse: cacheReuseObservation(cacheReads))
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
            latencies: latencies, memoryReads: memoryReads)
        return pass.succeeded(
            name, exchanges: exchanges, promptTokens: promptTokens,
            completionTokens: completionTokens, timeToFirstToken: nil, prefill: nil, decode: nil,
            wallClock: elapsed, cacheReuse: cacheReuseObservation(cacheReads))
    }

    /// The one entry the driver calls, and it cannot decline.
    ///
    /// The previous signature returned `nil` for a `kind` string it did not
    /// recognise and the driver loop skipped it, so a misspelled kind silently
    /// removed a scenario from a pass. There is no unrecognised kind left to
    /// return `nil` for: ``PromptSuiteSchema/Kind`` is closed and a document
    /// that names anything else never becomes a ``PromptSuiteSchema/Scenario``.
    static func run(
        pass: BenchmarkPass, scenario: PromptSuiteSchema.Scenario
    ) async -> RuntimeBenchmark.ScenarioResult {
        // The one supported absence on every kind, applied in one place: a
        // scenario that names no cap gets the driver's own default, which is
        // the value pinned into both records as `maxOutputTokens`.
        let maxTokens = scenario.maxTokens ?? defaultMaxOutputTokens
        switch scenario.body {
        case .single(let prompt, let prefixRepeats):
            return await single(
                pass: pass, name: scenario.name, maxTokens: maxTokens, prompt: prompt,
                prefixRepeats: prefixRepeats)
        case .tool(let prompt, let tools):
            return await tool(
                pass: pass, name: scenario.name, maxTokens: maxTokens, prompt: prompt,
                tools: tools)
        case .multiturn(let prefixRepeats, let turns):
            return await multiturn(
                pass: pass, name: scenario.name, maxTokens: maxTokens,
                prefixRepeats: prefixRepeats, turns: turns)
        case .soak(let iterations, let promptTemplate):
            return await soak(
                pass: pass, name: scenario.name, maxTokens: maxTokens, iterations: iterations,
                template: promptTemplate)
        }
    }
}
