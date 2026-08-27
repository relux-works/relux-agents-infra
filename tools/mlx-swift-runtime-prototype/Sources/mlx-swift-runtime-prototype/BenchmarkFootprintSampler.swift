import Darwin
import Foundation

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

/// Samples one pid's physical footprint until stopped, keeping both the
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
    private let interval: TimeInterval
    private let lock = NSLock()
    private var stopped = false
    private var processPeak: Int?
    private var windowPeak: Int?
    private var successfulSamples = 0
    private var failedSamples = 0
    private var windowLoadMax: Double?
    private var passLoadMax: Double?
    private var thread: Thread?

    init(pid: Int, interval: TimeInterval = 0.25) {
        self.pid = pid
        self.interval = interval
    }

    func start() {
        let thread = Thread { [weak self] in self?.loop() }
        thread.name = "benchmark-footprint-\(pid)"
        thread.start()
        self.thread = thread
    }

    private func loop() {
        while true {
            lock.lock()
            let done = stopped
            lock.unlock()
            if done { return }
            let value = physicalFootprintBytes(pid: pid)
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
            if let value {
                successfulSamples += 1
                if processPeak == nil || value > processPeak! { processPeak = value }
                if windowPeak == nil || value > windowPeak! { windowPeak = value }
            } else {
                failedSamples += 1
            }
            lock.unlock()
            Thread.sleep(forTimeInterval: interval)
        }
    }

    func stop() {
        lock.lock()
        stopped = true
        lock.unlock()
    }

    /// Start a fresh scenario-local window.
    ///
    /// Taken under the same lock the sampling thread writes through, so a sample
    /// landing during the reset cannot be counted into both windows.
    func beginWindow() {
        lock.lock()
        windowPeak = nil
        windowLoadMax = nil
        lock.unlock()
    }

    /// Peak within the window opened by the last ``beginWindow()``.
    ///
    /// `nil` when the window caught no successful sample — a scenario short
    /// enough to fall between two ticks. That stays `nil`: the comparison treats
    /// an unmeasured footprint as a blocker, and substituting the process peak
    /// would answer the question with a different quantity's number.
    func currentWindowPeak() -> Int? {
        lock.lock()
        defer { lock.unlock() }
        return windowPeak
    }

    func processPeakSoFar() -> Int? {
        lock.lock()
        defer { lock.unlock() }
        return processPeak
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

    func sampleCounts() -> (successful: Int, failed: Int) {
        lock.lock()
        defer { lock.unlock() }
        return (successfulSamples, failedSamples)
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
