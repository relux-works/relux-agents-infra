import Darwin
import Foundation
import MLX
import MLXNN
import MLXSwiftRuntimeContract

@main
struct Main {
    static func main() async {
        let arguments = Array(CommandLine.arguments.dropFirst())

        // Dispatched before `RuntimeOptions.parse` because it is a different
        // argument grammar entirely: it names no model, binds nothing and
        // touches no GPU, so neither the model-directory admission nor the
        // Metal shader library admission below applies to it. Both of those
        // gate *serving*, and there is nothing here to serve.
        if arguments.first == BenchmarkCompareCommand.name {
            exit(BenchmarkCompareCommand.run(arguments: Array(arguments.dropFirst())))
        }
        // The migration decision itself: it spawns the two runtimes, drives
        // every scenario against them, seals what it measured and judges the
        // pair, all in this process. It serves nothing of its own — the
        // candidate runtime is a *separate* invocation of this same binary,
        // launched through `model-harness` — so the serving admissions below do
        // not apply to it either.
        //
        // There is deliberately no `benchmark-attest` subcommand beside these
        // two. Revision 3 had one, and review used it exactly as documented to
        // have this binary attest two placeholder HTTP servers it had started
        // itself; the attestations were correct and the pass they certified had
        // served nothing. An observation a caller can direct at a process of
        // their choosing is not evidence about a benchmark, so the command is
        // gone rather than guarded.
        if arguments.first == BenchmarkRunCommand.name {
            exit(await BenchmarkRunCommand.run(arguments: Array(arguments.dropFirst())))
        }

        let subcommand: RuntimeOptions.Subcommand
        let options: RuntimeOptions
        do {
            (subcommand, options) = try RuntimeOptions.parse(arguments: arguments)
        } catch {
            StandardOutput.shared.log("\(error)")
            StandardOutput.shared.log(usage)
            exit(2)
        }

        // Fail closed on the model directory before binding anything. There is
        // no downloader in this executable, so a bad path has no fallback and
        // must not become a running server that answers for nothing.
        do {
            try ModelDirectoryCheck.admit(
                path: options.modelPath,
                observation: ModelDirectoryCheck.observe(path: options.modelPath))
        } catch {
            StandardOutput.shared.log("\(error)")
            exit(2)
        }

        if subcommand == .preflight {
            // Preflight never binds and never loads weights, so it is safe to
            // run while another runtime holds the host's memory. It reports the
            // shader library as a named stage rather than refusing, because
            // every check it runs is CPU-only and stays useful on a build that
            // cannot reach the GPU.
            exit(await Preflight.run(options: options))
        }

        // Fail closed on the Metal shader library before binding. `swift build`
        // cannot compile mlx-swift's shaders, so a SwiftPM-built binary aborts
        // deep inside MLX's C++ at the first GPU touch — which here is halfway
        // through a multi-GiB weight load, long after the port is bound and the
        // managed launcher has begun polling for readiness. Refusing up front
        // reports the real condition instead of a mid-load crash.
        do {
            try MetalShaderLibraryCheck.admit(
                MetalShaderLibraryCheck.classify(
                    MetalShaderLibraryCheck.inspect(
                        roots: MetalShaderLibraryCheck.defaultSearchRoots())))
        } catch {
            StandardOutput.shared.log("\(error)")
            exit(2)
        }

        let state = RuntimeState()
        // One ledger per process, shared by the engine that fills it and the
        // router that publishes it. Created here rather than with the engine
        // because it has to outlive the engine: condemnation drops the engine,
        // and what the dead worker released is only worth asking about after
        // that.
        let ledger = GenerationBatchLedgerStore()
        let router = Router(
            options: options,
            state: state,
            ledger: ledger,
            created: Int(Date().timeIntervalSince1970),
            systemFingerprint: "mlx-swift-runtime-prototype-\(mlxSwiftLMRevision)")

        let server = RuntimeHTTPServer()
        do {
            try await server.start(router: router, host: options.host, port: options.port)
        } catch {
            StandardOutput.shared.log("failed to bind \(options.host):\(options.port): \(error)")
            exit(1)
        }

        StandardOutput.shared.event(
            RuntimeEvent(
                name: "listening",
                fields: [
                    "host": .string(options.host),
                    "port": .int(options.port),
                    "model_id": .string(options.modelID),
                    "mlx_swift": .string(mlxSwiftRevision),
                    "mlx_swift_lm": .string(mlxSwiftLMRevision),
                ]))

        let shutdown = ShutdownSignal()
        shutdown.install()

        let loadTask = Task {
            do {
                let loaded = try await ModelLoader.load(
                    path: options.modelPath, preference: options.modelFactory)
                // The acceptance seam for the interval review found in
                // revision 3. Armed here rather than in the engine's `deinit`
                // because reaching the weights needs `perform`, which is
                // `async`, and a `deinit` body cannot await. Holding
                // `context.model` retains only the `LanguageModel` and its
                // arrays: the `ModelContainer` above it is untouched and still
                // dies with the engine, which is exactly the state a
                // wrapper-only release barrier misreads as a completed release.
                if options.faultRetainWeightsBelowContainerOnTeardown {
                    await loaded.container.perform { context in
                        RetainedWeights.shared.hold(context.model)
                    }
                }
                // The acceptance seam for the interval review found in
                // revision 4, and the one no byte comparison can close. It
                // parks the SECOND HALF of the flattened module tree and lets
                // everything above it -- the container, the root model object,
                // the first half of the tree -- be deallocated on schedule.
                // What the allocator then reports is a residue *below* this
                // model's load footprint, which is exactly what a released
                // model looks like to a process-global counter, while this
                // model's weights are demonstrably still owned. Dropping the
                // root is what makes the retention strict: holding it would
                // retain the whole tree transitively and reproduce the seam
                // above instead.
                if options.faultRetainWeightModulesOnTeardown {
                    await loaded.container.perform { context in
                        let modules = context.model.modules()
                        for module in modules.dropFirst(modules.count / 2) {
                            RetainedWeights.shared.hold(module)
                        }
                    }
                }
                // The acceptance seam that isolates the ABSOLUTE RESIDUE
                // clause, and the only one that holds no object of the model
                // tree. Copying the parameter arrays keeps their buffers alive
                // while every Module -- and the container above them -- is
                // deallocated exactly on schedule, so the ownership registry
                // reports a fully released model with the whole footprint still
                // active. Nothing but the residue itself can refuse that.
                if options.faultRetainWeightArraysOnTeardown {
                    await loaded.container.perform { context in
                        RetainedWeights.shared.holdArrays(
                            context.model.parameters().flattened().map { $0.1 })
                    }
                }
                // Review's revision-5 bypass, kept as a maintained production
                // input rather than as a scratch mutant. Same mechanism as the
                // seam above -- copied arrays, every Module dead, ownership
                // reporting the model released -- but NARROWED to the largest
                // half by nbytes, so the residue lands significant and yet
                // strictly BELOW the model's load footprint. That interval is
                // what every footprint-relative residue check admits, and it is
                // the state revision 5 attested a completed release over with
                // 255,724,192 B of a 262,361,760 B model still resident.
                if options.faultRetainWeightArraySubsetOnTeardown {
                    await loaded.container.perform { context in
                        let arrays = context.model.parameters().flattened()
                            .map { $0.1 }
                            .sorted { $0.nbytes > $1.nbytes }
                        RetainedWeights.shared.holdArrays(
                            Array(arrays.prefix(arrays.count / 2)))
                    }
                }
                let engine = GenerationEngine(
                    model: loaded, options: options, ledger: ledger)
                await state.markReady(engine: engine)
                StandardOutput.shared.event(
                    RuntimeEvent.modelLoaded(
                        modelID: options.modelID,
                        modelPath: options.modelPath,
                        loadSeconds: loaded.loadSeconds,
                        residentBytes: MemorySampler.residentBytes(),
                        physicalFootprintBytes: MemorySampler.physicalFootprintBytes(),
                        modelType: loaded.modelType))
                StandardOutput.shared.event(
                    RuntimeEvent(
                        name: "ready",
                        fields: [
                            "model_id": .string(options.modelID),
                            "factory": .string(loaded.factory),
                            // Both halves, on purpose. The preference is what
                            // the process was asked for and the factory is what
                            // actually built the model; a record that carried
                            // only the request could not tell a text-only
                            // launch that got the text factory from one whose
                            // text factory refused and silently fell back to
                            // the vision one -- which is the exact difference
                            // between chunked and unchunked prefill.
                            "factory_preference": .string(options.modelFactory.rawValue),
                            "factory_order": .string(
                                options.modelFactory.factoryOrder.joined(separator: ",")),
                            "host_memory_bytes": MemorySampler.hostMemoryBytes()
                                .map { .int(Int(clamping: $0)) } ?? .null,
                            // The measurement the condemned-worker teardown
                            // later has to see come back. Published at load so
                            // an acceptance run can tell an unmeasured
                            // footprint -- which fails the release gate closed
                            // -- from a measured one that was never returned.
                            "weight_footprint_bytes": .int(loaded.weightFootprintBytes),
                            "mlx_active_bytes": .int(Memory.snapshot().activeMemory),
                            "mlx_cache_bytes": .int(Memory.snapshot().cacheMemory),
                            "mlx_peak_bytes": .int(Memory.snapshot().peakMemory),
                        ]))
            } catch {
                await state.markFailed(String(describing: error))
                StandardOutput.shared.event(
                    RuntimeEvent(
                        name: "model_load_failed",
                        fields: [
                            "model_path": .string(options.modelPath),
                            "detail": .string(String(describing: error)),
                        ]))
                StandardOutput.shared.log("model load failed: \(error)")
            }
        }

        let signalName = await shutdown.wait()
        StandardOutput.shared.event(
            RuntimeEvent(name: "shutting_down", fields: ["signal": .string(signalName)]))
        await state.markShuttingDown()
        loadTask.cancel()
        await server.shutdown()
        StandardOutput.shared.event(RuntimeEvent(name: "stopped", fields: [:]))
        exit(0)
    }

    static let usage = """
        usage: mlx-swift-runtime-prototype serve --model <absolute-dir> --port <port> \
        [--model-id <id>] [--host 127.0.0.1] [--max-kv-size <n>] \
        [--prefill-step-size <n>] \
        [--model-factory text-only|vision-first|text-only-strict] \
        [--default-max-tokens <n>] [--reasoning-effort low|medium|xhigh] \
        [--fault-inject-generation-error <message>] \
        [--fault-inject-generation-error-count <n>] \
        [--fault-inject-generation-error-after-tokens <n>] \
        [--fault-inject-teardown-retain true|false] \
        [--fault-inject-teardown-retain-weights true|false] \
        [--fault-inject-teardown-retain-weight-modules true|false] \\
        [--fault-inject-teardown-retain-weight-arrays true|false]
               mlx-swift-runtime-prototype preflight --model <absolute-dir> \
        [--reasoning-effort low|medium|xhigh]
               mlx-swift-runtime-prototype benchmark-run \
        --config <model-harness.toml> --model <absolute-dir> --prompts <suite.json> \
        --thresholds <thresholds.json> --session <dir> --harness <model-harness> \
        --baseline-runtime <id> --baseline-profile <name> \
        --candidate-runtime <id> --candidate-profile <name> [--port <n>] \
        [--python-bin <path>] [--candidate-binary <path>] [--skip <scenario>]... \
        [--baseline-declare <text>]... [--candidate-declare <text>]... \
        [--startup-timeout <s>] [--request-timeout <s>] [--settle-seconds <s>]
               mlx-swift-runtime-prototype benchmark-compare \
        --baseline <record.json> --candidate <record.json> \
        --thresholds <thresholds.json> --attestations <dir> [--output <decision.json>]
        """
}

/// Waits for SIGTERM or SIGINT.
///
/// `model-harness` and the Pi launcher both stop a managed runtime by signalling
/// its process group and then waiting `shutdown_timeout_seconds`; a runtime that
/// ignores SIGTERM gets SIGKILLed and looks like a crash in the receipt.
final class ShutdownSignal: @unchecked Sendable {
    private let queue = DispatchQueue(label: "runtime.shutdown")
    private var sources: [DispatchSourceSignal] = []
    private var continuation: CheckedContinuation<String, Never>?
    private var delivered: String?
    private let lock = NSLock()

    func install() {
        for (number, name) in [(SIGTERM, "SIGTERM"), (SIGINT, "SIGINT")] {
            signal(number, SIG_IGN)
            let source = DispatchSource.makeSignalSource(signal: number, queue: queue)
            source.setEventHandler { [weak self] in self?.deliver(name) }
            source.resume()
            sources.append(source)
        }
    }

    func wait() async -> String {
        await withCheckedContinuation { continuation in
            lock.lock()
            if let delivered {
                lock.unlock()
                continuation.resume(returning: delivered)
                return
            }
            self.continuation = continuation
            lock.unlock()
        }
    }

    private func deliver(_ name: String) {
        lock.lock()
        guard delivered == nil else {
            lock.unlock()
            return
        }
        delivered = name
        let continuation = self.continuation
        self.continuation = nil
        lock.unlock()
        continuation?.resume(returning: name)
    }
}
