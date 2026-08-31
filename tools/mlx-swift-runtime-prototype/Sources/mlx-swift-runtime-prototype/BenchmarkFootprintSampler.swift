import Darwin
import Foundation
import MLXSwiftRuntimeContract

/// One other process's physical footprint, or `nil` when it could not be read.
///
/// Physical footprint, never `ps` RSS. Three identical loads of this model
/// reported 2 650, 10 774 and 14 056 MiB resident while the footprint stayed
/// within 16 MiB of itself, so resident size cannot answer "is the candidate
/// heavier" at all.
///
/// `nil` is never turned into `0` by a caller: a process that could not be
/// sampled and a process that used no memory are different facts, and only one
/// of them is good news.
func physicalFootprintBytes(pid: Int) -> Int? {
    var usage = rusage_info_current()
    let status = withUnsafeMutablePointer(to: &usage) { pointer in
        pointer.withMemoryRebound(to: rusage_info_t?.self, capacity: 1) { rebound in
            proc_pid_rusage(Int32(pid), RUSAGE_INFO_CURRENT, rebound)
        }
    }
    guard status == 0 else { return nil }
    return Int(usage.ri_phys_footprint)
}

/// One complete cross-runtime resident-memory reading.
///
/// The Mach component is exact. vmmap's resident mapped-file component is a
/// rounded display bucket, so the parser supplies its upper edge. Adding the
/// two can double-count some file-backed pages on runtimes whose allocator
/// already charges them; that is why the resulting score is named and recorded
/// as a conservative upper bound rather than as RSS or physical footprint.
func residentMemorySample(pid: Int) -> RuntimeMemorySampleRead {
    let process = Process()
    let output = Pipe()
    process.executableURL = URL(fileURLWithPath: "/usr/bin/vmmap")
    process.arguments = ["-summary", String(pid)]
    process.standardOutput = output
    process.standardError = FileHandle.nullDevice
    do {
        try process.run()
    } catch {
        return .readFailed("vmmap-launch-failed")
    }
    let data = output.fileHandleForReading.readDataToEndOfFile()
    process.waitUntilExit()
    guard process.terminationReason == .exit, process.terminationStatus == 0 else {
        return .readFailed("vmmap-read-failed")
    }
    guard let text = String(data: data, encoding: .utf8) else {
        return .malformed("vmmap-summary-not-utf8")
    }
    let raw: String?
    let mappedUpperBound: Int
    switch RuntimeVMMapSummary.read(text) {
    case .reported(let residentToken, let upperBound):
        raw = residentToken
        mappedUpperBound = upperBound
    case .notPresent:
        raw = nil
        mappedUpperBound = 0
    case .malformed(let issue):
        return .malformed(issue)
    }
    let mappedFileSampledAt = Date().timeIntervalSince1970
    guard let physical = physicalFootprintBytes(pid: pid) else {
        return .readFailed("mach-physical-footprint-unreadable")
    }
    let physicalSampledAt = Date().timeIntervalSince1970
    guard
        let components = RuntimeMemoryComponents(
            machPhysicalFootprintBytes: physical,
            vmmapResidentMappedFileRaw: raw,
            residentMappedFileBytesUpperBound: mappedUpperBound,
            sampledAtUnixSeconds: physicalSampledAt,
            machSampledAtUnixSeconds: physicalSampledAt,
            mappedFileSampledAtUnixSeconds: mappedFileSampledAt)
    else {
        return .malformed("resident-memory-component-overflow")
    }
    return .measured(components)
}

/// Samples one pid's resident-memory upper bound until stopped, keeping both the
/// process-wide peak and a scenario-local one.
///
/// The two are kept apart because conflating them decided a verdict once. A
/// running maximum never falls, so once the 75k-context probe has pushed a
/// process to 49 GiB every later scenario reports 49 GiB whatever it actually
/// cost — and a candidate that aborted before the expensive scenario reports a
/// *lower* whole-pass maximum than the baseline that completed it. Review
/// caught exactly that: 1.094x on whole-pass maxima from different completed
/// work, against 1.399x on the one scenario both runtimes finished.
final class BenchmarkFootprintSampler: @unchecked Sendable {
    private let pid: Int
    private let accounting: RuntimeMemoryAccounting
    private let lock = NSLock()
    private var stoppedFullLoop = false
    private var stoppedPhysicalLoop = false
    private var processReads: [RuntimeMemorySampleRead] = []
    private var windowReads: [RuntimeMemorySampleRead] = []
    private var windowLoadMax: Double?
    private var passLoadMax: Double?
    private var fullThread: Thread?
    private var physicalThread: Thread?
    private var latestMappedFile: (raw: String?, upperBound: Int, sampledAt: Double)?

    /// Effective mapped-file coverage bound. Never wider than the contract's:
    /// a caller may only *narrow* it, so no call site can buy a laxer gate, and
    /// the coverage-refusal probe can drive the production class with the old
    /// unreachable 125 ms claim and require it to refuse.
    private let maximumMappedFileGapSeconds: TimeInterval

    init(
        pid: Int,
        accounting: RuntimeMemoryAccounting = .residentMemoryUpperBound,
        narrowedMappedFileGapSeconds: TimeInterval? = nil
    ) {
        self.pid = pid
        self.accounting = accounting
        let contract = RuntimeMemoryAccounting.maximumMappedFileSampleGapSeconds
        maximumMappedFileGapSeconds =
            narrowedMappedFileGapSeconds.map { min($0, contract) }
            ?? contract
    }

    func currentMemoryReading() -> RuntimeMemorySampleRead {
        switch accounting {
        case .residentMemoryUpperBound:
            return residentMemorySample(pid: pid)
        }
    }

    func start() {
        appendFull(currentMemoryReading())
        let full = Thread { [weak self] in self?.fullLoop() }
        full.name = "benchmark-vmmap-\(pid)"
        full.start()
        fullThread = full
        let physical = Thread { [weak self] in self?.physicalLoop() }
        physical.name = "benchmark-physical-footprint-\(pid)"
        physical.start()
        physicalThread = physical
    }

    private func fullLoop() {
        while true {
            // `start()` seeds the first complete reading synchronously. Delay
            // the expensive vmmap refresh so observation cost stays at one
            // invocation per configured interval rather than two at startup.
            Thread.sleep(forTimeInterval: RuntimeMemoryAccounting.samplingIntervalSeconds)
            lock.lock()
            let done = stoppedFullLoop
            lock.unlock()
            if done { return }
            let reading = currentMemoryReading()
            // Sampled beside the footprint rather than once per pass: what
            // matters is whether the machine was busy *while this scenario was
            // being timed*, and a single reading taken at the end would miss a
            // neighbour that ran for the first half of it.
            let load = hostLoadAverage()
            lock.lock()
            if let load {
                if windowLoadMax == nil || load > windowLoadMax! { windowLoadMax = load }
                if passLoadMax == nil || load > passLoadMax! { passLoadMax = load }
            }
            if case .measured(let sample) = reading {
                latestMappedFile = (
                    sample.vmmapResidentMappedFileRaw,
                    sample.residentMappedFileBytesUpperBound,
                    sample.mappedFileSampledAtUnixSeconds!
                )
            }
            processReads.append(reading)
            windowReads.append(reading)
            lock.unlock()
        }
    }

    private func physicalLoop() {
        while true {
            lock.lock()
            let done = stoppedPhysicalLoop
            let mapped = latestMappedFile
            lock.unlock()
            if done { return }
            let reading: RuntimeMemorySampleRead
            if let physical = physicalFootprintBytes(pid: pid), let mapped {
                let physicalSampledAt = Date().timeIntervalSince1970
                guard
                    let sample = RuntimeMemoryComponents(
                        machPhysicalFootprintBytes: physical,
                        vmmapResidentMappedFileRaw: mapped.raw,
                        residentMappedFileBytesUpperBound: mapped.upperBound,
                        sampledAtUnixSeconds: physicalSampledAt,
                        machSampledAtUnixSeconds: physicalSampledAt,
                        mappedFileSampledAtUnixSeconds: mapped.sampledAt)
                else {
                    reading = .readFailed("resident-memory-component-overflow")
                    lock.lock()
                    processReads.append(reading)
                    windowReads.append(reading)
                    lock.unlock()
                    Thread.sleep(
                        forTimeInterval:
                            RuntimeMemoryAccounting.physicalFootprintSamplingIntervalSeconds)
                    continue
                }
                reading = .measured(sample)
            } else if mapped == nil {
                reading = .readFailed("resident-mapped-file-observation-unavailable")
            } else {
                reading = .readFailed("mach-physical-footprint-unreadable")
            }
            lock.lock()
            processReads.append(reading)
            windowReads.append(reading)
            lock.unlock()
            Thread.sleep(
                forTimeInterval: RuntimeMemoryAccounting.physicalFootprintSamplingIntervalSeconds)
        }
    }

    private func appendFull(_ reading: RuntimeMemorySampleRead) {
        lock.lock()
        if case .measured(let sample) = reading {
            latestMappedFile = (
                sample.vmmapResidentMappedFileRaw,
                sample.residentMappedFileBytesUpperBound,
                sample.mappedFileSampledAtUnixSeconds!
            )
        }
        processReads.append(reading)
        windowReads.append(reading)
        lock.unlock()
    }

    /// Stop the two loops *in order*, and never the fast one first.
    ///
    /// A vmmap read can still be in flight when the stop flag is set, and it
    /// lands with a Mach timestamp one reader-cost later than the sample before
    /// it. Stopping both loops at once let the 20 Hz Mach loop exit immediately
    /// while that read finished, so every process-wide series ended with a hole
    /// exactly one vmmap cost wide -- 0.36-0.49 s against the real baseline
    /// server in the gate smoke, which is 3-4x the 125 ms the Mach component
    /// admits. Every process-wide peak in that run came back `partial` and the
    /// whole-process memory delta was unmeasurable, for a reason that was an
    /// artifact of teardown rather than of the workload.
    ///
    /// So: retire the slow loop first and wait for its in-flight read to land,
    /// with the fast loop still covering that interval, and only then retire
    /// the fast loop. This costs nothing extra -- the old code already waited
    /// for the same thread.
    func stop() {
        lock.lock()
        stoppedFullLoop = true
        lock.unlock()
        while fullThread?.isExecuting == true {
            usleep(10_000)
        }
        lock.lock()
        stoppedPhysicalLoop = true
        lock.unlock()
        while physicalThread?.isExecuting == true {
            usleep(10_000)
        }
    }

    /// Start a fresh scenario-local window.
    ///
    /// Taken under the same lock the sampling thread writes through, so a sample
    /// landing during the reset cannot be counted into both windows.
    func beginWindow() {
        let reading = currentMemoryReading()
        lock.lock()
        if case .measured(let sample) = reading {
            latestMappedFile = (
                sample.vmmapResidentMappedFileRaw,
                sample.residentMappedFileBytesUpperBound,
                sample.mappedFileSampledAtUnixSeconds!
            )
        }
        windowReads = [reading]
        windowLoadMax = nil
        lock.unlock()
    }

    /// Peak within the window opened by the last ``beginWindow()``.
    ///
    /// An explicit absent/failed/malformed/partial result when the window did
    /// not yield one uninterrupted set of complete samples. The comparison
    /// refuses every such state instead of substituting a different window's
    /// reading.
    func currentWindowPeak() -> RuntimeMemoryPeak {
        lock.lock()
        defer { lock.unlock() }
        return coveredPeak(summarizing: windowReads)
    }

    func processPeakSoFar() -> RuntimeMemoryPeak {
        lock.lock()
        defer { lock.unlock() }
        return coveredPeak(summarizing: processReads)
    }

    /// Take the end-of-window sample synchronously and return both peaks from
    /// the same completed read set. This gives even a sub-interval scenario a
    /// real attempt while the periodic sampler continues to catch transient
    /// peaks during longer work.
    func capturePeaks() -> (window: RuntimeMemoryPeak, process: RuntimeMemoryPeak) {
        let reading = currentMemoryReading()
        lock.lock()
        if case .measured(let sample) = reading {
            latestMappedFile = (
                sample.vmmapResidentMappedFileRaw,
                sample.residentMappedFileBytesUpperBound,
                sample.mappedFileSampledAtUnixSeconds!
            )
        }
        processReads.append(reading)
        windowReads.append(reading)
        let window = coveredPeak(summarizing: windowReads)
        let process = coveredPeak(summarizing: processReads)
        lock.unlock()
        return (window, process)
    }

    /// A sampled maximum is a score only when the timestamped production series
    /// proves it had no hole larger than the cadence each component is actually
    /// read at: 125 ms for the in-process Mach series, which is what hides the
    /// 150 ms anonymous fixture, and the derived vmmap cadence for the
    /// mapped-file series. Otherwise the synthetic read failure makes the peak
    /// partial and the ordinary admission path refuses it.
    private func coveredPeak(summarizing reads: [RuntimeMemorySampleRead]) -> RuntimeMemoryPeak {
        guard
            let issue = RuntimeMemoryAccounting.samplingCoverageIssue(
                in: reads,
                maximumPhysicalGapSeconds:
                    RuntimeMemoryAccounting.maximumPhysicalFootprintSampleGapSeconds,
                maximumMappedFileGapSeconds: maximumMappedFileGapSeconds)
        else {
            return RuntimeMemoryPeak(summarizing: reads)
        }
        return RuntimeMemoryPeak(summarizing: reads + [.readFailed(issue)])
    }

    /// Highest 1-minute load average inside the window opened by the last
    /// ``beginWindow()``, or `nil` when the window caught no reading.
    func currentWindowLoadMax() -> Double? {
        lock.lock()
        defer { lock.unlock() }
        return windowLoadMax
    }

    func passLoadAverageMax() -> Double? {
        lock.lock()
        defer { lock.unlock() }
        return passLoadMax
    }

    func sampleCounts() -> (successful: Int, readFailed: Int, malformed: Int) {
        lock.lock()
        defer { lock.unlock() }
        let peak = RuntimeMemoryPeak(summarizing: processReads)
        return (
            peak.successfulSampleCount, peak.readFailureCount,
            peak.malformedSampleCount
        )
    }
}

/// The host's 1-minute load average, or `nil` when it could not be read.
///
/// Read for the same reason the runtime's footprint is: a measurement taken on
/// a busy machine is a measurement of the machine. `nil` is never turned into
/// `0` by a caller — "the host was idle" and "we could not tell" are different
/// facts, and only one of them supports a comparison.
func hostLoadAverage() -> Double? {
    var samples = [Double](repeating: 0, count: 3)
    guard getloadavg(&samples, 3) == 3 else { return nil }
    return samples[0]
}
