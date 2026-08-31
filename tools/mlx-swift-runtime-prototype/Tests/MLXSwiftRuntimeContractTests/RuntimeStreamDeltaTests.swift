import Foundation
import Testing

@testable import MLXSwiftRuntimeContract

@Suite("Cross-runtime stream timing boundary")
struct RuntimeStreamDeltaTests {
    @Test(
        "both deployed reasoning spellings carry generated text",
        arguments: ["reasoning", "reasoning_content"])
    func bothReasoningSpellingsCarryGeneratedText(field: String) {
        let reading = RuntimeStreamDelta.read([field: "thinking"])
        #expect(reading.content.isEmpty)
        #expect(reading.reasoning == "thinking")
        #expect(reading.generatedText == "thinking")
        #expect(reading.carriesGeneratedText)
    }

    @Test("content and both reasoning spellings share one event definition")
    func allGeneratedFieldsShareOneEventDefinition() {
        let reading = RuntimeStreamDelta.read([
            "content": "answer",
            "reasoning": "first",
            "reasoning_content": "second",
        ])
        #expect(reading.content == "answer")
        #expect(reading.reasoning == "firstsecond")
        #expect(reading.generatedText == "answerfirstsecond")
    }

    @Test("absent, empty, and malformed fields do not mint a timing event")
    func noGeneratedTextMeansNoTimingEvent() {
        for delta: [String: Any] in [
            [:],
            ["content": "", "reasoning": "", "reasoning_content": ""],
            ["content": NSNull(), "reasoning": 7, "reasoning_content": false],
        ] {
            #expect(!RuntimeStreamDelta.read(delta).carriesGeneratedText)
        }
    }
}

@Suite("Runtime memory accounting")
struct RuntimeMemoryAccountingTests {
    @Test("external vmmap sampling does not inherit the legacy 250 ms Mach cadence")
    func productionVMMapCadenceIsBounded() {
        #expect(RuntimeMemoryAccounting.samplingIntervalSeconds == 5)
        #expect(RuntimeMemoryAccounting.samplingIntervalSeconds > 0.25)
        #expect(RuntimeMemoryAccounting.physicalFootprintSamplingIntervalSeconds == 0.05)
        #expect(
            RuntimeMemoryAccounting.physicalFootprintSamplingIntervalSeconds
                < RuntimeMemoryAccounting.samplingIntervalSeconds)
        #expect(RuntimeMemoryAccounting.maximumPhysicalFootprintSampleGapSeconds < 0.15)
    }

    @Test("the mapped-file coverage bound is reachable by the reader that produces it")
    func mappedFileCoverageBoundIsDerivedFromTheReaderCost() {
        // The bound the contract admits must be derivable from the cadence the
        // mapped component is actually read at: one vmmap fork per sampling
        // interval, each costing the measured read cost, plus one read cost of
        // scheduling headroom. Revisions 1-3 asserted 125 ms here against a
        // 0.6-1.0 s reader, which left the memory dimension with an empty
        // admissible set.
        #expect(RuntimeMemoryAccounting.observedMappedFileReadCostSeconds >= 0.85)
        #expect(
            RuntimeMemoryAccounting.maximumMappedFileSampleGapSeconds
                == RuntimeMemoryAccounting.samplingIntervalSeconds
                + 2 * RuntimeMemoryAccounting.observedMappedFileReadCostSeconds)
        // Two consecutive mapped observations cannot be closer than one
        // sampling interval plus one reader cost. A bound below that is
        // unreachable by construction, whatever the workload does.
        #expect(
            RuntimeMemoryAccounting.maximumMappedFileSampleGapSeconds
                >= RuntimeMemoryAccounting.samplingIntervalSeconds
                + RuntimeMemoryAccounting.observedMappedFileReadCostSeconds)
        // And it is not open-ended: the admitted hole stays within two periodic
        // cadences, so a stalled vmmap thread is still refused.
        #expect(
            RuntimeMemoryAccounting.maximumMappedFileSampleGapSeconds
                < 2 * RuntimeMemoryAccounting.samplingIntervalSeconds)
        // The blind spot the wider bound buys is stated, not implied.
        #expect(RuntimeMemoryAccounting.mappedFileObservabilityNote.contains("not observable"))
        #expect(
            RuntimeMemoryAccounting.mappedFileObservabilityNote.contains(
                "\(RuntimeMemoryAccounting.maximumMappedFileSampleGapSeconds)"))
    }

    @Test("a series at the production mapped cadence is admitted, one at twice it is not")
    func mappedFileCoverageAdmitsTheProductionCadenceOnly() throws {
        func reading(mach: Double, mapped: Double) throws -> RuntimeMemorySampleRead {
            .measured(
                try #require(
                    RuntimeMemoryComponents(
                        machPhysicalFootprintBytes: 1,
                        vmmapResidentMappedFileRaw: nil,
                        residentMappedFileBytesUpperBound: 0,
                        sampledAtUnixSeconds: mach,
                        machSampledAtUnixSeconds: mach,
                        mappedFileSampledAtUnixSeconds: mapped)))
        }

        let mappedBound = RuntimeMemoryAccounting.maximumMappedFileSampleGapSeconds
        let machBound = RuntimeMemoryAccounting.maximumPhysicalFootprintSampleGapSeconds
        let cadence =
            RuntimeMemoryAccounting.samplingIntervalSeconds
            + RuntimeMemoryAccounting.observedMappedFileReadCostSeconds

        // One vmmap read every production cadence, with the fast Mach loop
        // carrying the same mapped timestamp forward between them.
        func series(mappedEvery period: Double) throws -> [RuntimeMemorySampleRead] {
            var reads: [RuntimeMemorySampleRead] = []
            var lastMapped = 100.0
            var now = 100.0
            while now <= 100.0 + 2 * period {
                if now - lastMapped >= period { lastMapped = now }
                reads.append(try reading(mach: now, mapped: lastMapped))
                now += machBound
            }
            return reads
        }

        #expect(
            RuntimeMemoryAccounting.samplingCoverageIssue(
                in: try series(mappedEvery: cadence),
                maximumPhysicalGapSeconds: machBound,
                maximumMappedFileGapSeconds: mappedBound) == nil)
        #expect(
            RuntimeMemoryAccounting.samplingCoverageIssue(
                in: try series(mappedEvery: 2 * cadence),
                maximumPhysicalGapSeconds: machBound,
                maximumMappedFileGapSeconds: mappedBound)
                == "resident-mapped-file-sampling-gap")
    }

    @Test("a scored peak that does not state its observation limit is not scoreable")
    func scoredPeakMustCarryItsObservationLimit() throws {
        let sample = try #require(
            RuntimeMemoryComponents(
                machPhysicalFootprintBytes: 10_000,
                vmmapResidentMappedFileRaw: "2K",
                residentMappedFileBytesUpperBound: 3_072,
                sampledAtUnixSeconds: 100))
        let peak = RuntimeMemoryPeak(summarizing: [.measured(sample)])
        #expect(peak.validatedScoredBytes == 13_072)
        #expect(
            peak.mappedFileObservationLimitSeconds
                == RuntimeMemoryAccounting.maximumMappedFileSampleGapSeconds)
        #expect(peak.mappedFileObservabilityNote != nil)

        // A document that omits the stated limit, or claims a cadence this
        // instrument was not run at, is not this comparison's evidence.
        let encoder = JSONEncoder()
        let decoder = JSONDecoder()
        var object = try #require(
            try JSONSerialization.jsonObject(with: encoder.encode(peak)) as? [String: Any])
        for mutation: (String, Any?) in [
            ("mappedFileObservationLimitSeconds", nil),
            ("mappedFileObservationLimitSeconds", 0.125),
            ("mappedFileObservabilityNote", nil),
            ("mappedFileObservabilityNote", "everything is observable"),
        ] {
            var mutated = object
            if let value = mutation.1 {
                mutated[mutation.0] = value
            } else {
                mutated.removeValue(forKey: mutation.0)
            }
            let decoded = try decoder.decode(
                RuntimeMemoryPeak.self,
                from: try JSONSerialization.data(withJSONObject: mutated))
            #expect(
                decoded.validatedScoredBytes == nil,
                "\(mutation.0) = \(String(describing: mutation.1)) was still scored")
        }
        // Control: the untouched document still scores, so the four mutants
        // above fail for their own reason and not because re-encoding breaks.
        object["scoredBytes"] = 13_072
        let untouched = try decoder.decode(
            RuntimeMemoryPeak.self, from: try JSONSerialization.data(withJSONObject: object))
        #expect(untouched.validatedScoredBytes == 13_072)
    }

    @Test("timestamp coverage refuses a hole capable of hiding the 150 ms fixture")
    func refusesSubCadenceCoverageHole() throws {
        func reading(mach: Double?, mapped: Double?) throws -> RuntimeMemorySampleRead {
            .measured(
                try #require(
                    RuntimeMemoryComponents(
                        machPhysicalFootprintBytes: 1,
                        vmmapResidentMappedFileRaw: nil,
                        residentMappedFileBytesUpperBound: 0,
                        sampledAtUnixSeconds: mach,
                        machSampledAtUnixSeconds: mach,
                        mappedFileSampledAtUnixSeconds: mapped)))
        }

        let maximumGap = RuntimeMemoryAccounting.maximumPhysicalFootprintSampleGapSeconds
        #expect(
            RuntimeMemoryAccounting.samplingCoverageIssue(
                in: try [reading(mach: 100, mapped: 100), reading(mach: 100.1, mapped: 100.1)],
                maximumPhysicalGapSeconds: maximumGap,
                maximumMappedFileGapSeconds: maximumGap) == nil)
        #expect(
            RuntimeMemoryAccounting.samplingCoverageIssue(
                in: try [reading(mach: 100, mapped: 100), reading(mach: 100.15, mapped: 100.1)],
                maximumPhysicalGapSeconds: maximumGap,
                maximumMappedFileGapSeconds: maximumGap)
                == "mach-physical-footprint-sampling-gap")
        #expect(
            RuntimeMemoryAccounting.samplingCoverageIssue(
                in: try [reading(mach: nil, mapped: 100), reading(mach: 100.05, mapped: 100.05)],
                maximumPhysicalGapSeconds: maximumGap,
                maximumMappedFileGapSeconds: maximumGap)
                == "mach-physical-footprint-sampling-timestamp-unreadable")
        #expect(
            RuntimeMemoryAccounting.samplingCoverageIssue(
                in: [try reading(mach: 100, mapped: 100)],
                maximumPhysicalGapSeconds: maximumGap,
                maximumMappedFileGapSeconds: maximumGap)
                == "resident-memory-sampling-coverage-insufficient")
        #expect(
            RuntimeMemoryAccounting.samplingCoverageIssue(
                in: try [reading(mach: 100, mapped: 95), reading(mach: 100.1, mapped: 95)],
                maximumPhysicalGapSeconds: maximumGap,
                maximumMappedFileGapSeconds: maximumGap)
                == "resident-mapped-file-sampling-coverage-insufficient")
        #expect(
            RuntimeMemoryAccounting.samplingCoverageIssue(
                in: try [reading(mach: 100, mapped: 100), reading(mach: 100.1, mapped: 100.2)],
                maximumPhysicalGapSeconds: maximumGap,
                maximumMappedFileGapSeconds: maximumGap)
                == "resident-mapped-file-sampling-gap")
    }

    @Test(
        "every runtime shape uses the same resident upper bound",
        arguments: [
            ("/opt/homebrew/bin/llama-server", "/models/qwen.gguf", ["--model", "q.gguf"]),
            ("/tmp/runtime-wrapper", "/models/qwen.gguf", ["--model", "q.gguf"]),
            ("/Users/test/bin/mlx_lm-relux.server", "/models/mlx", ["--model", "mlx"]),
            ("/Applications/mlx-swift-runtime-prototype", "/models/mlx", ["serve"]),
        ])
    func runtimeNamesAndArtifactsCannotNarrowTheAccounting(
        executable: String, model: String, argv: [String]
    ) {
        #expect(
            RuntimeMemoryAccounting.forExecutable(
                executable, modelPath: model, launchArgv: argv)
                == .residentMemoryUpperBound)
    }

    @Test("a complete vmmap mapped-file row produces a rounded upper component")
    func parsesMappedFileResidencyAsAnUpperBound() {
        let summary = """
            Analysis Tool:   /usr/bin/vmmap
                                                VIRTUAL RESIDENT DIRTY
            REGION TYPE                        SIZE     SIZE     SIZE
            mapped file                       26.8G    26.6G       0K
            TOTAL                             30.0G    28.0G       1G
            """
        #expect(
            RuntimeVMMapSummary.read(summary)
                == .reported(raw: "26.6G", bytesUpperBound: 28_668_906_701))
    }

    @Test("complete absence differs from partial and malformed vmmap output")
    func vmmapAbsenceDoesNotLaunderReadDefects() {
        let noMappedFiles = """
            Analysis Tool: /usr/bin/vmmap
            VIRTUAL RESIDENT DIRTY
            REGION TYPE SIZE SIZE
            MALLOC 1M 1M
            TOTAL 1M 1M
            """
        #expect(RuntimeVMMapSummary.read(noMappedFiles) == .notPresent)
        #expect(
            RuntimeVMMapSummary.read(noMappedFiles.replacingOccurrences(of: "TOTAL", with: ""))
                == .malformed("vmmap-summary-incomplete"))
        #expect(
            RuntimeVMMapSummary.read(
                noMappedFiles.replacingOccurrences(of: "MALLOC 1M 1M", with: "mapped file"))
                == .malformed("vmmap-mapped-file-row-partial"))
    }

    @Test("failed, malformed, partial and absent windows all refuse a score")
    func incompleteWindowsFailClosedByName() throws {
        let components = try #require(
            RuntimeMemoryComponents(
                machPhysicalFootprintBytes: 100,
                vmmapResidentMappedFileRaw: "1K",
                residentMappedFileBytesUpperBound: 2_048))
        let cases: [(RuntimeMemoryPeakStatus, RuntimeMemoryPeak)] = [
            (.absent, RuntimeMemoryPeak(summarizing: [])),
            (.readFailed, RuntimeMemoryPeak(summarizing: [.readFailed("vmmap-read-failed")])),
            (.malformed, RuntimeMemoryPeak(summarizing: [.malformed("vmmap-summary-incomplete")])),
            (
                .partial,
                RuntimeMemoryPeak(summarizing: [
                    .measured(components), .readFailed("vmmap-read-failed"),
                ])
            ),
        ]
        for (status, peak) in cases {
            #expect(peak.status == status)
            #expect(peak.scoredBytes == nil)
            #expect(peak.validatedScoredBytes == nil)
            #expect(peak.refusalReason?.contains(status.rawValue) == true)
        }
    }
}
