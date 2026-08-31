import Foundation

/// The one memory quantity every benchmark runtime is sampled with.
///
/// `ri_phys_footprint` is an allocator/dirty-page accounting and does not see
/// clean resident pages from llama.cpp's mmap-loaded GGUF. The scored quantity
/// is therefore an explicitly conservative upper bound:
///
///     Mach physical footprint + vmmap "mapped file" resident upper bound
///
/// Both runtimes use the same sum. The components stay in the record so a
/// consumer never has to reverse-engineer what the score means, and the old
/// Mach-only number remains raw evidence rather than silently becoming the
/// score again.
public enum RuntimeMemoryAccounting: String, Codable, Equatable, Sendable {
    case residentMemoryUpperBound =
        "mach-physical-footprint-plus-vmmap-resident-mapped-file-upper-bound"

    /// Kept at the production profile-resolution call site so tests can prove
    /// executable/model spelling cannot narrow one runtime back to Mach-only.
    public static func forExecutable(
        _: String, modelPath _: String? = nil, launchArgv _: [String] = []
    ) -> RuntimeMemoryAccounting {
        .residentMemoryUpperBound
    }

    public static let scoreSemantics = "conservative-upper-bound"

    /// Production cadence for the external `vmmap` reader.
    ///
    /// The legacy 250 ms cadence was chosen for the in-process Mach read. A
    /// `vmmap -summary` launch walks another process's mappings and materially
    /// stalls this 28 GiB workload when repeated at that frequency. Scenario
    /// boundaries still take synchronous samples, so a five-second periodic
    /// cadence retains at least one attempt per short window without turning
    /// the observer into most of the benchmark load.
    public static let samplingIntervalSeconds: TimeInterval = 5
    /// Cheap kernel reads are bounded to 20 Hz; `vmmap` remains at 0.2 Hz.
    public static let physicalFootprintSamplingIntervalSeconds: TimeInterval = 0.05

    /// A 150 ms resident-memory transient is only a measured capability when
    /// the persisted series proves there was no sampling hole large enough to
    /// hide it. Keep the admitted gap strictly below that fixture duration.
    ///
    /// This bound belongs to the in-process kernel read only. It is deliberately
    /// not shared with the mapped-file component below, whose reader is three
    /// orders of magnitude more expensive.
    public static let maximumPhysicalFootprintSampleGapSeconds: TimeInterval = 0.125

    /// Measured cost of one production `/usr/bin/vmmap -summary` fork, read
    /// through the same `Process`/`Pipe`/`waitUntilExit` shape the sampler uses.
    ///
    /// Measured on this host (Apple silicon, macOS 25.5, 2026-08-30), eight
    /// consecutive calls per target:
    ///
    ///   * ~3 MB target:    0.608-0.672 s, mean 0.625 s
    ///   * ~0.9 GiB target: 0.783-0.850 s, mean 0.802 s
    ///
    /// The cost rises with the target's mapping count, so the 28 GiB runtimes
    /// this benchmark compares are not read any faster than the larger figure.
    /// Rounded up to a whole second, which is the headroom this constant
    /// carries over the largest target actually timed.
    public static let observedMappedFileReadCostSeconds: TimeInterval = 1.0

    /// Largest admitted gap between two consecutive *mapped-file* observations.
    ///
    /// Derived, not asserted. The mapped component is produced only by the
    /// `vmmap` reader above, which runs once per ``samplingIntervalSeconds`` on
    /// the periodic loop plus once at each window boundary. Two mapped
    /// observations are therefore at best one sampling interval apart, and in
    /// practice that interval plus the reader's own cost; the second reader
    /// cost is scheduling headroom for a host under the benchmark's own load.
    ///
    /// Revisions before this one claimed 125 ms here. That claim was never
    /// reachable: a 0.68-1.0 s reader cannot deliver two observations 125 ms
    /// apart, so the memory dimension had an empty admissible set and every
    /// comparison returned `unmeasured` rather than a number.
    public static let maximumMappedFileSampleGapSeconds: TimeInterval =
        samplingIntervalSeconds + 2 * observedMappedFileReadCostSeconds

    /// What this instrument cannot see, stated in the contract and carried in
    /// every emitted ``RuntimeMemoryPeak`` rather than left for a reader to
    /// infer.
    ///
    /// The mapped-file component that matters for this comparison is the model
    /// weights, which stay resident for the whole run. A file-backed region
    /// that appears and is released strictly between two mapped observations is
    /// not represented in the scored peak, and no part of this record claims
    /// otherwise. Sub-cadence *anonymous* growth is a different risk and is
    /// covered by the Mach component's own 20 Hz series and its own
    /// ``maximumPhysicalFootprintSampleGapSeconds`` bound.
    public static var mappedFileObservabilityNote: String {
        "mapped-file resident transients shorter than "
            + "\(maximumMappedFileSampleGapSeconds)s are not observable; the "
            + "mapped-file component is read by an external vmmap fork costing "
            + "~\(observedMappedFileReadCostSeconds)s per call at a "
            + "\(samplingIntervalSeconds)s cadence. Anonymous growth is covered "
            + "separately by the Mach series at "
            + "\(physicalFootprintSamplingIntervalSeconds)s."
    }

    /// Refusal reason for a timestamp series that cannot support the claimed
    /// sub-cadence observation, or `nil` when its coverage is sufficient.
    ///
    /// Explicit read failures are handled separately by ``RuntimeMemoryPeak``.
    /// This check closes the silent path where successful samples are too sparse
    /// or lack timestamps, while still looking like a measured maximum.
    public static func samplingCoverageIssue(
        in reads: [RuntimeMemorySampleRead],
        maximumPhysicalGapSeconds: TimeInterval,
        maximumMappedFileGapSeconds: TimeInterval
    ) -> String? {
        let samples = reads.compactMap { reading -> RuntimeMemoryComponents? in
            guard case .measured(let sample) = reading else { return nil }
            return sample
        }
        guard samples.count >= 2 else {
            return "resident-memory-sampling-coverage-insufficient"
        }
        let physicalTimestamps = samples.compactMap(\.machSampledAtUnixSeconds)
        guard physicalTimestamps.count == samples.count,
            physicalTimestamps.allSatisfy({ $0.isFinite && $0 > 0 })
        else {
            return "mach-physical-footprint-sampling-timestamp-unreadable"
        }
        let orderedPhysical = physicalTimestamps.sorted()
        guard
            zip(orderedPhysical, orderedPhysical.dropFirst()).allSatisfy({
                $1 - $0 <= maximumPhysicalGapSeconds
            })
        else {
            return "mach-physical-footprint-sampling-gap"
        }

        let mappedTimestamps = samples.compactMap(\.mappedFileSampledAtUnixSeconds)
        guard mappedTimestamps.count == samples.count,
            mappedTimestamps.allSatisfy({ $0.isFinite && $0 > 0 })
        else {
            return "resident-mapped-file-sampling-timestamp-unreadable"
        }
        // Reused mapped values deliberately retain their original timestamp.
        // Deduplicate those timestamps before judging coverage so twenty fast
        // Mach samples backed by one vmmap read still count as one mapped-file
        // observation, not twenty.
        let orderedMapped = Array(Set(mappedTimestamps)).sorted()
        guard orderedMapped.count >= 2 else {
            return "resident-mapped-file-sampling-coverage-insufficient"
        }
        let physicalStart = orderedPhysical[0]
        let physicalEnd = orderedPhysical[orderedPhysical.count - 1]
        guard orderedMapped[0] <= physicalStart + maximumMappedFileGapSeconds,
            orderedMapped[orderedMapped.count - 1]
                >= physicalEnd - maximumMappedFileGapSeconds,
            zip(orderedMapped, orderedMapped.dropFirst()).allSatisfy({
                $1 - $0 <= maximumMappedFileGapSeconds
            })
        else {
            return "resident-mapped-file-sampling-gap"
        }
        return nil
    }
}

/// Raw components from one simultaneous memory sample.
public struct RuntimeMemoryComponents: Codable, Equatable, Sendable {
    /// Exact kernel `ri_phys_footprint` reading.
    public let machPhysicalFootprintBytes: Int
    /// The literal resident token printed on vmmap's `mapped file` row.
    /// `nil` means a complete summary contained no such row, which is a
    /// measured zero rather than a failed read.
    public let vmmapResidentMappedFileRaw: String?
    /// Upper edge of vmmap's displayed resident-value bucket, in bytes.
    /// The display is rounded, so treating its centre as exact could make an
    /// alleged upper bound smaller than the actual resident mapping.
    public let residentMappedFileBytesUpperBound: Int
    /// The scored sum. Named as a bound, never as physical footprint or RSS.
    public let residentMemoryUpperBoundBytes: Int
    /// Wall-clock timestamp for audit of the raw sample series.
    public let sampledAtUnixSeconds: Double?
    /// Timestamp of the kernel physical-footprint read.
    public let machSampledAtUnixSeconds: Double?
    /// Timestamp of the vmmap mapped-file observation. Reusing its value must
    /// reuse this timestamp too; otherwise stale evidence looks fresh.
    public let mappedFileSampledAtUnixSeconds: Double?

    public init?(
        machPhysicalFootprintBytes: Int,
        vmmapResidentMappedFileRaw: String?,
        residentMappedFileBytesUpperBound: Int,
        sampledAtUnixSeconds: Double? = nil,
        machSampledAtUnixSeconds: Double? = nil,
        mappedFileSampledAtUnixSeconds: Double? = nil
    ) {
        guard machPhysicalFootprintBytes >= 0, residentMappedFileBytesUpperBound >= 0 else {
            return nil
        }
        let (sum, overflow) = machPhysicalFootprintBytes.addingReportingOverflow(
            residentMappedFileBytesUpperBound)
        guard !overflow else { return nil }
        self.machPhysicalFootprintBytes = machPhysicalFootprintBytes
        self.vmmapResidentMappedFileRaw = vmmapResidentMappedFileRaw
        self.residentMappedFileBytesUpperBound = residentMappedFileBytesUpperBound
        residentMemoryUpperBoundBytes = sum
        self.machSampledAtUnixSeconds = machSampledAtUnixSeconds ?? sampledAtUnixSeconds
        self.mappedFileSampledAtUnixSeconds =
            mappedFileSampledAtUnixSeconds ?? sampledAtUnixSeconds
        self.sampledAtUnixSeconds = sampledAtUnixSeconds ?? machSampledAtUnixSeconds
    }

    /// Re-derive the scored component after decoding. `Codable` evidence may
    /// carry all four stored fields, so comparing its two derived fields to
    /// each other is not validation: both can be forged back to Mach-only while
    /// a non-zero mapped-file component remains in the same document.
    public var validatedResidentMemoryUpperBoundBytes: Int? {
        guard machPhysicalFootprintBytes >= 0, residentMappedFileBytesUpperBound >= 0 else {
            return nil
        }
        let (derived, overflow) = machPhysicalFootprintBytes.addingReportingOverflow(
            residentMappedFileBytesUpperBound)
        guard !overflow, residentMemoryUpperBoundBytes == derived else { return nil }
        return derived
    }
}

/// One sampling attempt before it is folded into a window peak.
public enum RuntimeMemorySampleRead: Equatable, Sendable {
    case measured(RuntimeMemoryComponents)
    case readFailed(String)
    case malformed(String)
}

/// Whether a window can honestly be scored.
public enum RuntimeMemoryPeakStatus: String, Codable, Equatable, Sendable {
    /// Every attempted sample was complete and at least one existed.
    case measured
    /// The window ended before any sampling attempt reached it.
    case absent
    /// Every attempt failed at the OS/tool boundary.
    case readFailed = "read-failed"
    /// Every attempt returned bytes that did not form the required reading.
    case malformed
    /// Some evidence existed, but not one uninterrupted complete sample set.
    case partial
}

/// A window-local or process-wide peak with the evidence needed to interpret it.
///
/// A partial peak retains its highest complete raw sample for diagnosis but has
/// no `scoredBytes`. That is the fail-closed distinction between "we measured
/// some of it" and "the whole window was measured".
public struct RuntimeMemoryPeak: Codable, Equatable, Sendable {
    public let accounting: RuntimeMemoryAccounting
    public let scoreSemantics: String
    public let status: RuntimeMemoryPeakStatus
    public let scoredBytes: Int?
    public let peakSample: RuntimeMemoryComponents?
    public let successfulSampleCount: Int
    public let readFailureCount: Int
    public let malformedSampleCount: Int
    public let issues: [String]
    /// Every complete observation, in acquisition order, not only its maximum.
    public let rawSamples: [RuntimeMemoryComponents]?
    /// The mapped-file observation cadence this peak was produced under.
    /// Optional only so records written before the limit was stated still
    /// decode; a peak missing it cannot be scored.
    public let mappedFileObservationLimitSeconds: Double?
    /// Plain-language statement of what that cadence cannot see.
    public let mappedFileObservabilityNote: String?

    public static let absent = RuntimeMemoryPeak(summarizing: [])

    public init(summarizing reads: [RuntimeMemorySampleRead]) {
        var complete: [RuntimeMemoryComponents] = []
        var readFailures = 0
        var malformed = 0
        var issues: [String] = []
        for read in reads {
            switch read {
            case .measured(let sample):
                complete.append(sample)
            case .readFailed(let issue):
                readFailures += 1
                if !issues.contains(issue) { issues.append(issue) }
            case .malformed(let issue):
                malformed += 1
                if !issues.contains(issue) { issues.append(issue) }
            }
        }
        let peak = complete.max {
            $0.residentMemoryUpperBoundBytes < $1.residentMemoryUpperBoundBytes
        }
        let status: RuntimeMemoryPeakStatus
        if reads.isEmpty {
            status = .absent
        } else if !complete.isEmpty && readFailures == 0 && malformed == 0 {
            status = .measured
        } else if !complete.isEmpty || (readFailures > 0 && malformed > 0) {
            status = .partial
        } else if malformed > 0 {
            status = .malformed
        } else {
            status = .readFailed
        }
        accounting = .residentMemoryUpperBound
        scoreSemantics = RuntimeMemoryAccounting.scoreSemantics
        self.status = status
        scoredBytes = status == .measured ? peak?.residentMemoryUpperBoundBytes : nil
        peakSample = peak
        successfulSampleCount = complete.count
        readFailureCount = readFailures
        malformedSampleCount = malformed
        self.issues = issues
        mappedFileObservationLimitSeconds =
            RuntimeMemoryAccounting.maximumMappedFileSampleGapSeconds
        mappedFileObservabilityNote = RuntimeMemoryAccounting.mappedFileObservabilityNote
        rawSamples = complete.sorted {
            ($0.sampledAtUnixSeconds ?? 0) < ($1.sampledAtUnixSeconds ?? 0)
        }
    }

    /// A concise production refusal reason. `nil` only for a scored window.
    public var refusalReason: String? {
        guard status != .measured else { return nil }
        return
            "resident memory upper bound is \(status.rawValue) "
            + "(complete=\(successfulSampleCount), read-failed=\(readFailureCount), "
            + "malformed=\(malformedSampleCount))"
    }

    /// The only value scoring may consume. A decoded document whose status,
    /// counts, components and score disagree is malformed evidence, not a
    /// measured peak, so it returns no value and the decision blocks.
    public var validatedScoredBytes: Int? {
        guard accounting == .residentMemoryUpperBound,
            scoreSemantics == RuntimeMemoryAccounting.scoreSemantics,
            successfulSampleCount >= 0, readFailureCount >= 0, malformedSampleCount >= 0
        else { return nil }
        switch status {
        case .measured:
            // A scored peak has to say, in the record, which mapped-file
            // observation cadence produced it. A document that omits the limit
            // or claims a different one was produced under a different
            // instrument and is not this comparison's evidence.
            guard successfulSampleCount > 0, readFailureCount == 0, malformedSampleCount == 0,
                issues.isEmpty, let peakSample,
                mappedFileObservationLimitSeconds
                    == RuntimeMemoryAccounting.maximumMappedFileSampleGapSeconds,
                mappedFileObservabilityNote == RuntimeMemoryAccounting.mappedFileObservabilityNote,
                let validatedComposite = peakSample.validatedResidentMemoryUpperBoundBytes,
                scoredBytes == validatedComposite
            else { return nil }
            return validatedComposite
        case .absent:
            guard successfulSampleCount == 0, readFailureCount == 0, malformedSampleCount == 0,
                issues.isEmpty, scoredBytes == nil, peakSample == nil
            else { return nil }
            return nil
        case .readFailed:
            guard successfulSampleCount == 0, readFailureCount > 0, malformedSampleCount == 0,
                !issues.isEmpty, scoredBytes == nil, peakSample == nil
            else { return nil }
            return nil
        case .malformed:
            guard successfulSampleCount == 0, readFailureCount == 0, malformedSampleCount > 0,
                !issues.isEmpty, scoredBytes == nil, peakSample == nil
            else { return nil }
            return nil
        case .partial:
            guard scoredBytes == nil,
                readFailureCount > 0 || malformedSampleCount > 0,
                (peakSample != nil) == (successfulSampleCount > 0), !issues.isEmpty
            else { return nil }
            return nil
        }
    }
}

/// Parses the resident mapped-file component out of a complete `vmmap -summary`.
public enum RuntimeVMMapSummary {
    public enum Reading: Equatable, Sendable {
        /// Complete summary with a mapped-file row.
        case reported(raw: String, bytesUpperBound: Int)
        /// Complete summary with no mapped-file row: a measured zero.
        case notPresent
        /// Output existed but did not constitute a complete, parseable summary.
        case malformed(String)
    }

    public static func read(_ text: String) -> Reading {
        let lines = text.split(whereSeparator: \Character.isNewline).map(String.init)
        guard lines.contains(where: { $0.contains("Analysis Tool:") && $0.contains("vmmap") }),
            lines.contains(where: { $0.contains("VIRTUAL") && $0.contains("RESIDENT") }),
            lines.contains(where: { $0.hasPrefix("REGION TYPE") }),
            lines.contains(where: { $0.hasPrefix("TOTAL") })
        else {
            return .malformed("vmmap-summary-incomplete")
        }
        let mapped = lines.filter { line in
            line.split(whereSeparator: \Character.isWhitespace).prefix(2).map(String.init)
                == ["mapped", "file"]
        }
        guard mapped.count <= 1 else { return .malformed("vmmap-mapped-file-row-ambiguous") }
        guard let row = mapped.first else { return .notPresent }
        let columns = row.split(whereSeparator: \Character.isWhitespace).map(String.init)
        guard columns.count >= 4 else { return .malformed("vmmap-mapped-file-row-partial") }
        let raw = columns[3]
        guard let upperBound = displayBucketUpperBound(raw) else {
            return .malformed("vmmap-mapped-file-resident-malformed")
        }
        return .reported(raw: raw, bytesUpperBound: upperBound)
    }

    /// vmmap prints human-readable buckets. Add one display quantum so the
    /// persisted component is truly an upper bound even if vmmap truncates.
    private static func displayBucketUpperBound(_ raw: String) -> Int? {
        guard let suffix = raw.last else { return nil }
        let multiplier: Double
        switch suffix {
        case "K": multiplier = 1_024
        case "M": multiplier = 1_024 * 1_024
        case "G": multiplier = 1_024 * 1_024 * 1_024
        case "T": multiplier = 1_024 * 1_024 * 1_024 * 1_024
        default: return nil
        }
        let numberText = String(raw.dropLast())
        guard let value = Double(numberText), value >= 0, value.isFinite else { return nil }
        let decimalPlaces =
            numberText.split(separator: ".", omittingEmptySubsequences: false)
            .dropFirst().first?.count ?? 0
        let quantum = multiplier / pow(10, Double(decimalPlaces))
        let upper = value * multiplier + quantum
        guard upper.isFinite, upper <= Double(Int.max) else { return nil }
        return Int(ceil(upper))
    }
}
