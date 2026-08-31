import Darwin
import Foundation
import MLXSwiftRuntimeContract

/// Emitted diagnostics shared by the memory probes below, so a failing probe
/// leaves the record it judged rather than only an exit code.
private func emit(_ payload: [String: Any]) {
    guard let data = try? JSONSerialization.data(withJSONObject: payload) else { return }
    FileHandle.standardOutput.write(data)
    FileHandle.standardOutput.write(Data([0x0a]))
}

private func encodedPeak(_ peak: RuntimeMemoryPeak) -> Any {
    guard let data = try? JSONEncoder().encode(peak),
        let object = try? JSONSerialization.jsonObject(with: data)
    else { return NSNull() }
    return object
}

/// The mapped-file series' widest observed gap, after collapsing the reused
/// timestamps the fast Mach loop carries forward.
private func widestMappedFileGap(in peak: RuntimeMemoryPeak) -> Double? {
    guard let samples = peak.rawSamples else { return nil }
    let stamps = Array(Set(samples.compactMap(\.mappedFileSampledAtUnixSeconds))).sorted()
    guard stamps.count >= 2 else { return nil }
    return zip(stamps, stamps.dropFirst()).map { $1 - $0 }.max()
}

/// Every scored peak has to carry the cadence it was produced under, and that
/// cadence has to be at least as wide as the series actually delivered.
private func statesItsOwnObservationLimit(_ peak: RuntimeMemoryPeak) -> Bool {
    guard
        peak.mappedFileObservationLimitSeconds
            == RuntimeMemoryAccounting.maximumMappedFileSampleGapSeconds,
        peak.mappedFileObservabilityNote == RuntimeMemoryAccounting.mappedFileObservabilityNote,
        let limit = peak.mappedFileObservationLimitSeconds
    else { return false }
    guard let widest = widestMappedFileGap(in: peak) else { return false }
    return widest <= limit
}

/// Production-entry positive for the anonymous half of the scored quantity.
///
/// A 128 MiB anonymous allocation lives for 150 ms, far below the 5 s `vmmap`
/// cadence, and the 20 Hz Mach series has to catch it *and* the window has to
/// end up scored. Revision 3 required this entry to refuse instead, which is
/// how a total loss of the memory dimension read as a green suite.
enum BenchmarkMemorySamplerProbe {
    static let name = "benchmark-memory-sampler-probe"

    static func run() -> Int32 {
        let bytes = 128 * 1_024 * 1_024
        let sampler = BenchmarkFootprintSampler(pid: Int(getpid()))
        sampler.start()
        sampler.beginWindow()
        guard
            let address = mmap(
                nil, bytes, PROT_READ | PROT_WRITE, MAP_PRIVATE | MAP_ANON, -1, 0),
            address != MAP_FAILED
        else { return 2 }
        memset(address, 0x5a, bytes)
        usleep(150_000)
        munmap(address, bytes)
        usleep(150_000)
        let peaks = sampler.capturePeaks()
        sampler.stop()
        let window = peaks.window
        emit([
            "probe": name,
            "peak": encodedPeak(window),
            "widestMappedFileGapSeconds": widestMappedFileGap(in: window) ?? NSNull(),
        ])
        guard
            let samples = window.rawSamples,
            samples.count >= 3,
            // The dimension must be scoreable, not merely refused honestly.
            window.status == .measured,
            window.issues.isEmpty,
            window.validatedScoredBytes != nil,
            statesItsOwnObservationLimit(window),
            samples.allSatisfy({ $0.machSampledAtUnixSeconds != nil }),
            zip(samples, samples.dropFirst()).allSatisfy({ earlier, later in
                later.machSampledAtUnixSeconds! - earlier.machSampledAtUnixSeconds!
                    <= RuntimeMemoryAccounting.maximumPhysicalFootprintSampleGapSeconds
            }),
            let minimum = samples.map(\.machPhysicalFootprintBytes).min(),
            let maximum = samples.map(\.machPhysicalFootprintBytes).max(),
            maximum - minimum >= bytes / 2
        else { return 1 }
        return 0
    }
}

/// Production-entry positive for the mapped-file half of the scored quantity.
///
/// The mapped-file component this comparison exists to measure is the model
/// weights: resident for the whole run, not a sub-second transient. So the
/// fixture is a sustained one — a 256 MiB file-backed region held resident
/// across the window boundary — and the probe requires it to reach the scored
/// bytes, not merely to be refused politely.
///
/// It also requires the emitted record to state its own mapped-file
/// observation cadence, and requires that cadence to be no narrower than what
/// the series actually delivered. Tightening the contract constant back toward
/// the unreachable 125 ms claim fails here.
enum BenchmarkMappedFileSamplerProbe {
    static let name = "benchmark-mapped-file-sampler-probe"

    static func run() -> Int32 {
        let bytes = 256 * 1_024 * 1_024
        let path = FileManager.default.temporaryDirectory
            .appendingPathComponent("benchmark-mapped-probe-\(UUID().uuidString)").path
        let descriptor = open(path, O_RDWR | O_CREAT | O_EXCL, S_IRUSR | S_IWUSR)
        guard descriptor >= 0 else { return 2 }
        defer {
            close(descriptor)
            unlink(path)
        }
        guard ftruncate(descriptor, off_t(bytes)) == 0 else { return 2 }
        guard
            let writable = mmap(nil, bytes, PROT_READ | PROT_WRITE, MAP_SHARED, descriptor, 0),
            writable != MAP_FAILED
        else { return 2 }
        memset(writable, 0x5a, bytes)
        guard msync(writable, bytes, MS_SYNC) == 0 else {
            munmap(writable, bytes)
            return 2
        }
        munmap(writable, bytes)

        let sampler = BenchmarkFootprintSampler(pid: Int(getpid()))
        sampler.start()
        // Window opens before the region exists, so the scored growth has to
        // come from an observation taken inside the window.
        sampler.beginWindow()
        guard
            let mapped = mmap(nil, bytes, PROT_READ, MAP_PRIVATE, descriptor, 0),
            mapped != MAP_FAILED
        else { return 2 }
        var checksum: UInt8 = 0
        let readable = mapped.assumingMemoryBound(to: UInt8.self)
        for offset in stride(from: 0, to: bytes, by: Int(getpagesize())) {
            checksum ^= readable[offset]
        }
        guard case .measured(let direct) = residentMemorySample(pid: Int(getpid())) else {
            munmap(mapped, bytes)
            return 2
        }
        // The region stays resident across the closing observation. That is the
        // shape of the weights this benchmark actually scores.
        let peak = sampler.capturePeaks().window
        sampler.stop()
        munmap(mapped, bytes)

        let scoredMapped = peak.peakSample?.residentMappedFileBytesUpperBound ?? 0
        emit([
            "probe": name,
            "checksum": Int(checksum),
            "directMappedFileBytesUpperBound": direct.residentMappedFileBytesUpperBound,
            "scoredMappedFileBytesUpperBound": scoredMapped,
            "widestMappedFileGapSeconds": widestMappedFileGap(in: peak) ?? NSNull(),
            "peak": encodedPeak(peak),
        ])

        guard direct.residentMappedFileBytesUpperBound >= bytes / 2 else { return 2 }
        guard peak.status == .measured, peak.issues.isEmpty,
            peak.validatedScoredBytes != nil,
            statesItsOwnObservationLimit(peak),
            scoredMapped >= direct.residentMappedFileBytesUpperBound - bytes / 8
        else { return 1 }
        return 0
    }
}

/// Production-entry negative for the same gate the two probes above pass.
///
/// The identical production class, on the identical workload, with the
/// mapped-file coverage bound narrowed to the unreachable 125 ms claim
/// revisions 1-3 shipped. It must refuse, and the unnarrowed control on the
/// same shape must score — otherwise "refused correctly" and "never scores"
/// are indistinguishable, which is exactly how revision 3 read as green.
///
/// The override can only narrow: ``BenchmarkFootprintSampler`` clamps it to the
/// contract bound, so this is not a path a caller can use to buy a laxer gate.
enum BenchmarkMemoryCoverageRefusalProbe {
    static let name = "benchmark-memory-coverage-refusal-probe"

    private static func window(narrowedTo bound: TimeInterval?) -> RuntimeMemoryPeak {
        let sampler = BenchmarkFootprintSampler(
            pid: Int(getpid()), narrowedMappedFileGapSeconds: bound)
        sampler.start()
        sampler.beginWindow()
        usleep(300_000)
        let peak = sampler.capturePeaks().window
        sampler.stop()
        return peak
    }

    static func run() -> Int32 {
        let narrowed = window(narrowedTo: 0.125)
        let control = window(narrowedTo: nil)
        emit([
            "probe": name,
            "narrowedMappedFileGapSeconds": 0.125,
            "narrowed": encodedPeak(narrowed),
            "control": encodedPeak(control),
        ])
        guard narrowed.status == .partial,
            narrowed.validatedScoredBytes == nil,
            narrowed.issues.contains(where: {
                $0 == "resident-mapped-file-sampling-gap"
                    || $0 == "resident-mapped-file-sampling-coverage-insufficient"
            }),
            !narrowed.issues.contains(where: { $0.hasPrefix("mach-physical-footprint") })
        else { return 1 }
        guard control.status == .measured, control.issues.isEmpty,
            control.validatedScoredBytes != nil,
            statesItsOwnObservationLimit(control)
        else { return 1 }
        return 0
    }
}

/// Production-entry check for the sampler's *teardown*, which is where the
/// process-wide series is finalised.
///
/// `BenchmarkRunCommand` calls `sampler.stop()` and then
/// `sampler.processPeakSoFar()` (`BenchmarkRunCommand.swift:674-675`), so any
/// coverage hole created by stopping is carried straight into the whole-process
/// memory delta. Stopping the 20 Hz Mach loop while the 5 s vmmap loop still
/// had a read in flight left a hole one reader-cost wide at the tail of every
/// process series, and every process-wide peak came back `partial`.
///
/// The probe reproduces that race deliberately: it waits until the periodic
/// vmmap read is in flight, stops, and requires the finalised process peak to
/// be scoreable. It also requires the race to have actually happened, so a
/// mistimed attempt reports a failure rather than a vacuous pass.
enum BenchmarkStopCoverageProbe {
    static let name = "benchmark-memory-stop-coverage-probe"

    private struct Attempt {
        let caughtReadInFlight: Bool
        let peak: RuntimeMemoryPeak
    }

    private static func attempt() -> Attempt {
        let sampler = BenchmarkFootprintSampler(pid: Int(getpid()))
        sampler.start()
        // `fullLoop` sleeps one sampling interval before its first periodic
        // read, so this lands inside that read rather than between reads.
        Thread.sleep(forTimeInterval: RuntimeMemoryAccounting.samplingIntervalSeconds + 0.15)
        let stopRequestedAt = Date().timeIntervalSince1970
        sampler.stop()
        let peak = sampler.processPeakSoFar()
        let landedAfterStop =
            (peak.rawSamples ?? []).contains { sample in
                (sample.mappedFileSampledAtUnixSeconds ?? 0) >= stopRequestedAt
            }
        return Attempt(caughtReadInFlight: landedAfterStop, peak: peak)
    }

    static func run() -> Int32 {
        var caught = false
        var attempts: [Any] = []
        for _ in 0 ..< 3 {
            let result = attempt()
            attempts.append([
                "caughtReadInFlight": result.caughtReadInFlight,
                "status": result.peak.status.rawValue,
                "issues": result.peak.issues,
                "scored": result.peak.validatedScoredBytes ?? NSNull(),
            ])
            // Every finalised process peak must be scoreable, whether or not
            // this attempt caught the race.
            guard result.peak.status == .measured, result.peak.issues.isEmpty,
                result.peak.validatedScoredBytes != nil,
                statesItsOwnObservationLimit(result.peak)
            else {
                emit(["probe": name, "attempts": attempts, "caughtReadInFlight": caught])
                return 1
            }
            caught = caught || result.caughtReadInFlight
            if caught { break }
        }
        emit(["probe": name, "attempts": attempts, "caughtReadInFlight": caught])
        // A pass that never reproduced the race proves nothing about it.
        return caught ? 0 : 1
    }
}
