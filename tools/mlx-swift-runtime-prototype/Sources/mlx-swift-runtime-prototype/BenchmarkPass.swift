import Darwin
import Foundation
import MLXSwiftRuntimeContract

/// One runtime's benchmark pass, from the process this invocation spawned to
/// the record it hands to the judgement.
///
/// The class exists to make one thing structurally true: a measurement and the
/// exchange it came from are produced by the same object, in the same process,
/// during the window that object is being observed in. Revision 3 had these
/// three things in three places — a Python driver that measured, a gate
/// subcommand that attested whatever pid it was handed, and a compare
/// subcommand that read both off disk — and review walked between them with two
/// placeholder HTTP servers.
///
/// Nothing here takes a measurement from an argument.
@MainActor
final class BenchmarkPass {
    let runtime: String
    let suite: BenchmarkScenarios.Suite
    let modelID: String
    let runtimeProcessID: Int
    let endpoint: URL
    let requestTimeout: TimeInterval
    private let session: URLSession
    private let sampler: BenchmarkFootprintSampler

    /// Scenario-level detail that is reported beside the decision rather than
    /// scored. Kept out of the record's scored fields on purpose — see the
    /// soak runner for why the aggregate is not a decode rate.
    private(set) var soakDetail: [String: Double?] = [:]
    private(set) var lifecycle: [String: Double?] = [:]

    init(
        runtime: String,
        suite: BenchmarkScenarios.Suite,
        modelID: String,
        runtimeProcessID: Int,
        endpoint: URL,
        requestTimeout: TimeInterval,
        session: URLSession,
        sampler: BenchmarkFootprintSampler
    ) {
        self.runtime = runtime
        self.suite = suite
        self.modelID = modelID
        self.runtimeProcessID = runtimeProcessID
        self.endpoint = endpoint
        self.requestTimeout = requestTimeout
        self.session = session
        self.sampler = sampler
    }

    func beginScenarioWindow() { sampler.beginWindow() }

    func stream(body: Data) async -> BenchmarkHTTPDriver.StreamedCompletion {
        await BenchmarkHTTPDriver.stream(
            session: session, endpoint: endpoint.appendingPathComponent("chat/completions"),
            path: RuntimeBenchmark.ScenarioTranscript.completionPath, body: body,
            timeout: requestTimeout)
    }

    func post(body: Data) async -> BenchmarkHTTPDriver.Completion {
        await BenchmarkHTTPDriver.post(
            session: session, endpoint: endpoint.appendingPathComponent("chat/completions"),
            path: RuntimeBenchmark.ScenarioTranscript.completionPath, body: body,
            timeout: requestTimeout)
    }

    func models(timeout: TimeInterval = 10) async -> (status: Int, body: Data) {
        await BenchmarkHTTPDriver.get(
            session: session, url: endpoint.appendingPathComponent("models"), timeout: timeout)
    }

    func recordLifecycle(_ key: String, _ value: Double?) { lifecycle[key] = value }

    func recordSoak(
        iterations: Int, elapsed: Double, completionTokens: Int, latencies: [Double],
        footprints: [Int]
    ) {
        let sorted = latencies.sorted()
        soakDetail = [
            "iterations": Double(iterations),
            "aggregate_output_tokens_per_second": elapsed > 0
                ? Double(completionTokens) / elapsed : nil,
            "first_latency_seconds": latencies.first,
            "last_latency_seconds": latencies.last,
            "median_latency_seconds": sorted.isEmpty ? nil : sorted[sorted.count / 2],
            "first_footprint_bytes": footprints.first.map(Double.init),
            "last_footprint_bytes": footprints.last.map(Double.init),
            "footprint_drift_bytes": footprints.count >= 2
                ? Double(footprints[footprints.count - 1] - footprints[0]) : nil,
        ]
    }

    /// A scenario that produced no usable measurement.
    ///
    /// It reports what went wrong and the exchanges it got that far on. It does
    /// **not** report zeros: zero tokens per second and "we never found out"
    /// are different facts and the decision treats them differently.
    func failed(
        _ name: String, _ failureMode: String,
        exchanges: [BenchmarkHTTPDriver.Exchange], wallClock: Double? = nil
    ) -> RuntimeBenchmark.ScenarioResult {
        RuntimeBenchmark.ScenarioResult(
            name: name, succeeded: false, failureMode: failureMode, wallClockSeconds: wallClock,
            peakPhysicalFootprintBytes: sampler.currentWindowPeak(),
            processPeakSoFarBytes: sampler.processPeakSoFar(),
            hostLoadAverageMax: sampler.currentWindowLoadMax(),
            transcript: RuntimeBenchmark.ScenarioTranscript(exchanges: exchanges))
    }

    func succeeded(
        _ name: String,
        exchanges: [BenchmarkHTTPDriver.Exchange],
        promptTokens: Int?,
        completionTokens: Int?,
        timeToFirstToken: Double?,
        prefill: Double?,
        decode: Double?,
        wallClock: Double
    ) -> RuntimeBenchmark.ScenarioResult {
        RuntimeBenchmark.ScenarioResult(
            name: name, succeeded: true, failureMode: nil, promptTokens: promptTokens,
            completionTokens: completionTokens, timeToFirstTokenSeconds: timeToFirstToken,
            prefillTokensPerSecond: prefill, decodeTokensPerSecond: decode,
            wallClockSeconds: wallClock,
            peakPhysicalFootprintBytes: sampler.currentWindowPeak(),
            processPeakSoFarBytes: sampler.processPeakSoFar(),
            hostLoadAverageMax: sampler.currentWindowLoadMax(),
            transcript: RuntimeBenchmark.ScenarioTranscript(exchanges: exchanges))
    }
}

/// A child process this invocation spawned into its own session.
///
/// `posix_spawn` with `POSIX_SPAWN_SETSID` rather than `Process`, for one
/// reason: the pass has to be able to tear down the launcher *and* the runtime
/// it owns, and the only safe way to signal a whole tree is a process group
/// that is not this process's own. Without the new session, `killpg` at the end
/// of a pass would signal the benchmark itself.
final class SpawnedProcess {
    let pid: pid_t
    let argv: [String]
    private var reaped: Int32?

    private init(pid: pid_t, argv: [String]) {
        self.pid = pid
        self.argv = argv
    }

    static func spawn(
        executable: String, arguments: [String], standardOutputPath: String
    ) -> SpawnedProcess? {
        var attributes: posix_spawnattr_t?
        posix_spawnattr_init(&attributes)
        defer { posix_spawnattr_destroy(&attributes) }
        posix_spawnattr_setflags(&attributes, Int16(POSIX_SPAWN_SETSID))

        var actions: posix_spawn_file_actions_t?
        posix_spawn_file_actions_init(&actions)
        defer { posix_spawn_file_actions_destroy(&actions) }
        posix_spawn_file_actions_addopen(
            &actions, 1, standardOutputPath, O_WRONLY | O_CREAT | O_TRUNC, 0o644)
        posix_spawn_file_actions_adddup2(&actions, 1, 2)

        let argv = [executable] + arguments
        var cArgv: [UnsafeMutablePointer<CChar>?] = argv.map { strdup($0) }
        cArgv.append(nil)
        defer { for pointer in cArgv where pointer != nil { free(pointer) } }

        var pid: pid_t = 0
        let status = posix_spawn(&pid, executable, &actions, &attributes, cArgv, environ)
        guard status == 0 else { return nil }
        return SpawnedProcess(pid: pid, argv: argv)
    }

    /// Exit status if the child has already been reaped, `nil` while it is
    /// still running. Never blocks.
    func exitStatusIfFinished() -> Int32? {
        if let reaped { return reaped }
        var status: Int32 = 0
        let result = waitpid(pid, &status, WNOHANG)
        guard result == pid else { return nil }
        reaped = status
        return status
    }

    /// Terminate the whole session, escalating once.
    func terminate(gracePeriod: TimeInterval = 60) -> Int32? {
        if let reaped { return reaped }
        killpg(pid, SIGTERM)
        let deadline = Date().addingTimeInterval(gracePeriod)
        while Date() < deadline {
            if let status = exitStatusIfFinished() { return status }
            usleep(200_000)
        }
        killpg(pid, SIGKILL)
        let hardDeadline = Date().addingTimeInterval(30)
        while Date() < hardDeadline {
            if let status = exitStatusIfFinished() { return status }
            usleep(200_000)
        }
        return nil
    }

    /// The single child of a pid, resolved from the kernel process table.
    ///
    /// The runtime under test is the launcher's child, not the launcher.
    /// Sampling the launcher would report its few megabytes and call it the
    /// model's footprint.
    static func children(of parent: pid_t) -> [pid_t] {
        var name: [Int32] = [CTL_KERN, KERN_PROC, KERN_PROC_ALL, 0]
        var size = 0
        guard sysctl(&name, UInt32(name.count) - 1, nil, &size, nil, 0) == 0, size > 0 else {
            return []
        }
        let count = size / MemoryLayout<kinfo_proc>.stride
        var table = [kinfo_proc](repeating: kinfo_proc(), count: count + 16)
        var actual = table.count * MemoryLayout<kinfo_proc>.stride
        let status = table.withUnsafeMutableBufferPointer { buffer -> Int32 in
            sysctl(&name, UInt32(name.count) - 1, buffer.baseAddress, &actual, nil, 0)
        }
        guard status == 0 else { return [] }
        let found = actual / MemoryLayout<kinfo_proc>.stride
        return table.prefix(found)
            .filter { $0.kp_eproc.e_ppid == parent }
            .map(\.kp_proc.p_pid)
    }
}
