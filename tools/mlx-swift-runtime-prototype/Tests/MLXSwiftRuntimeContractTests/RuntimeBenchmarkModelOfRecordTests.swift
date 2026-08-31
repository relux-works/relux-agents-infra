import Foundation
import Testing

@testable import MLXSwiftRuntimeContract

/// G4: what "the same model" means when the two runtimes cannot read the same
/// weight file.
///
/// The pair under test throughout is the real one. `mlx_lm.server` serves
/// `Qwen3.8-27B-Uncensored-MLX-8bit`, a weight *directory* at
/// `8bit/group64/affine`; `llama-server` serves a single `Q8_0` `.gguf`. Two
/// files, two paths, two digests, one upstream BF16 model. Before this change
/// `modelPath` and `modelDigest` were pins compared for equality, so that
/// comparison was refused forever — not because it was unsound, but because the
/// pin had been written about the local file rather than about the model.
///
/// Every test below is about the replacement being *harder* than what it
/// replaced, not softer. The same-format pair still has to be byte-identical
/// (``admitsASameArtifactPairUnchanged`` and the `modelDigest` row of the pin
/// table in `RuntimeBenchmarkTests`), and the cross-format pair has to carry
/// evidence that survives six separate checks. Two of these tests are positive
/// and pin the admitted class so a *narrowing* mutant reddens them; the rest
/// are negative and fail if the gate admits what it must reject.
///
/// **Production call site.** `BenchmarkRunCommand.execute` reads the verdict
/// with `equivalenceReading(path:)`, refuses an unreadable one before any
/// launch, computes each artifact's digest itself, writes the reading onto both
/// attestations, copies the verdict's declared non-equivalences into both
/// records, and hands the pair to `RuntimeBenchmark.admit`, which calls
/// `admitModelIdentity`. `scripts/benchmark-gate-smoke.sh` section 4 drives
/// that path through the shipped subcommand for the shapes this suite cannot
/// express — an absent `--equivalence`, and an unreadable one.
@Suite("runtime benchmark model of record")
struct RuntimeBenchmarkModelOfRecordTests {
    typealias Fixture = RuntimeBenchmarkTests

    /// The shared upstream model. Both 8-bit builds are quantized from it and
    /// neither is derived from the other, which is why it is the only thing
    /// they can be pinned on.
    static let source = "hf:orcarouter/Qwen3.8-27B-Uncensored-BF16@a855f377"

    static let mlxDigest = Fixture.modelDigest
    static let ggufPath =
        "/Users/alexis/src/local-models/Qwen3.8-27B-Uncensored-GGUF-Q8_0/"
        + "Qwen3.8-27B-Uncensored-OrcaRouter-Q8_0.gguf"
    static let ggufDigest = "31756fca"

    /// The three differences TASK-260828-3g87i4 measured and did not dissolve.
    ///
    /// They are load-bearing: a cross-format pair is admitted only while both
    /// records carry all three, so a report of the comparison cannot be read
    /// without them.
    static let nonEquivalences = [
        "the MLX build drops the MTP head the GGUF carries",
        "the vision tower is in both files and resident in neither on the text path",
        "GGUF norms are F32 where the MLX build keeps bf16",
    ]

    /// Both launches pin the same bound, in their own spellings, so the
    /// `contextPolicy` pin agrees and the tests below are about the model pin
    /// rather than about the KV one.
    static let mlxArgv = Fixture.pythonLaunchArgv
    static let llamaArgv = [
        "--model", ggufPath, "--host", "127.0.0.1", "--port", "18031",
        "--ctx-size", "76800", "--ubatch-size", "2048", "--reasoning-effort", "medium",
    ]

    static let evidenceDigest = Fixture.digest("3b")

    /// The trust anchor these fixtures stand under.
    ///
    /// Every clause below the trust lookup in ``RuntimeBenchmark/admitModelIdentity``
    /// reads a field out of a document, and none of them is reachable until the
    /// document is a decision this repository took. This suite is about those
    /// clauses, so it hands admission a decision naming its own fixture digest.
    ///
    /// The trust lookup itself is *not* tested with this: it is tested against
    /// the shipped store, unstubbed, in `RuntimeBenchmarkTrustedEquivalenceTests`,
    /// and at the production entry by `scripts/benchmark-gate-smoke.sh`. An
    /// injected registry that could stand in for the shipped one is exactly the
    /// F1 defect, so nothing here calls admission without saying which registry
    /// it means.
    static let trust = [
        TrustedEquivalenceDecision(
            sourceOfRecord: source, documentDigest: evidenceDigest,
            requiredNonEquivalences: nonEquivalences,
            provenance: "contract-suite fixture, not shipped")
    ]

    static func verdict(
        sourceOfRecord: String = source,
        verdict: ModelEquivalence.Verdict = .comparable,
        artifacts: [ModelEquivalence.Artifact]? = nil,
        nonEquivalences: [String]? = nil
    ) -> ModelEquivalence {
        ModelEquivalence(
            sourceOfRecord: sourceOfRecord,
            verdict: verdict,
            artifacts: artifacts
                ?? [
                    ModelEquivalence.Artifact(
                        path: Fixture.modelPath, digest: mlxDigest,
                        quantization: "8bit/group64/affine"),
                    ModelEquivalence.Artifact(
                        path: ggufPath, digest: ggufDigest, quantization: "Q8_0"),
                ],
            declaredNonEquivalences: nonEquivalences ?? declaredByDefault)
    }

    /// The verdict's own list, aliased so the parameter above can shadow the
    /// type-level name without losing access to it.
    static let declaredByDefault = nonEquivalences

    /// The MLX baseline: the artifact it serves, the policy its launch pins.
    static func mlxRecord(
        equivalence: ModelEquivalenceReading,
        asymmetries: [String]? = nil,
        quantization: String = "8bit/group64/affine"
    ) -> RuntimeBenchmark.RunRecord {
        Fixture.record(
            runtime: "python-mlx-lm",
            pins: Fixture.variantPins(
                modelOfRecord: RuntimeBenchmark.modelOfRecord(
                    artifactDigest: mlxDigest, observing: equivalence),
                quantization: quantization,
                contextPolicy: RuntimeBenchmark.contextPolicy(
                    derivedFrom: mlxArgv, observing: .reported(76800))),
            startedAt: 100, finishedAt: 200,
            asymmetries: asymmetries ?? nonEquivalences,
            provenance: Fixture.provenance(
                launchArgv: mlxArgv, executableDigest: Fixture.digest("cd")))
    }

    /// The llama.cpp candidate, serving a different file.
    static func llamaRecord(
        equivalence: ModelEquivalenceReading,
        asymmetries: [String]? = nil,
        quantization: String = "Q8_0",
        digest: String = ggufDigest,
        argv: [String]? = nil
    ) -> RuntimeBenchmark.RunRecord {
        let launch = argv ?? llamaArgv
        return Fixture.record(
            runtime: "llamacpp",
            pins: Fixture.variantPins(
                modelOfRecord: RuntimeBenchmark.modelOfRecord(
                    artifactDigest: digest, observing: equivalence),
                modelPath: ggufPath, modelDigest: digest, quantization: quantization,
                contextPolicy: RuntimeBenchmark.contextPolicy(
                    derivedFrom: launch, observing: .reported(76800))),
            startedAt: 300, finishedAt: 400,
            asymmetries: asymmetries ?? nonEquivalences,
            provenance: Fixture.provenance(
                launchArgv: launch, executableDigest: Fixture.digest("12")))
    }

    /// Admission with each record observed under its own reading.
    ///
    /// The equivalence reading is supplied per attestation, never derived from
    /// the record being judged — it is the gate's document, and that is exactly
    /// what stops a record minting its own model identity.
    static func admit(
        baseline: RuntimeBenchmark.RunRecord,
        baselineEvidence: ModelEquivalenceReading,
        candidate: RuntimeBenchmark.RunRecord,
        candidateEvidence: ModelEquivalenceReading,
        candidateSpeculation: RuntimeSpeculation = .reported(false),
        trusting: [TrustedEquivalenceDecision]? = nil
    ) throws -> RuntimeBenchmark.Comparison {
        try RuntimeBenchmark.admit(
            baseline: baseline,
            baselineAttestation: Fixture.attestation(
                for: baseline, contextWindow: .reported(76800), speculation: .notReported,
                equivalence: baselineEvidence),
            candidate: candidate,
            candidateAttestation: Fixture.attestation(
                for: candidate, contextWindow: .reported(76800),
                speculation: candidateSpeculation, equivalence: candidateEvidence),
            requiredScenarios: ["short_prompt"], gateBinaryDigest: Fixture.gateDigest,
            trusting: trusting ?? trust)
    }

    /// The whole point: this pair used to be impossible.
    ///
    /// Positive, and it is the case a narrowing mutant has to break. A gate
    /// that tightened any clause here into refusing legitimate evidence — one
    /// that demanded byte-identical artifacts after all, or that refused a
    /// verdict naming more artifacts than the two under test — would redden
    /// this test and only this kind of test.
    @Test("a llama.cpp candidate on a GGUF is admitted against an MLX baseline under evidence")
    func admitsACrossFormatPairUnderEvidence() throws {
        let reading = ModelEquivalenceReading.read(Self.verdict(), digest: Self.evidenceDigest)
        let comparison = try Self.admit(
            baseline: Self.mlxRecord(equivalence: reading), baselineEvidence: reading,
            candidate: Self.llamaRecord(equivalence: reading), candidateEvidence: reading)
        // The pin is the upstream model, not either file.
        #expect(comparison.pins.modelOfRecord == "source:\(Self.source)")
        // And the three differences travel, so no report of this decision can
        // be written without stating them.
        for entry in Self.nonEquivalences {
            #expect(comparison.declaredAsymmetries.contains(entry))
        }
    }

    /// The other half, and the reason the replacement relaxes nothing.
    ///
    /// Also positive, also a narrowing target: a gate that started demanding
    /// evidence from every pair would break the incumbent MLX-vs-MLX
    /// comparison this whole benchmark exists to decide.
    @Test("a same-artifact pair with no evidence is admitted exactly as before")
    func admitsASameArtifactPairUnchanged() throws {
        let comparison = try Fixture.admit(
            baseline: Fixture.baseline, candidate: Fixture.candidate,
            requiredScenarios: ["short_prompt"])
        #expect(comparison.pins.modelOfRecord == "artifact:\(Self.mlxDigest)")
    }

    // ---------------------------------------------------------------- absence

    /// Absence of evidence refuses *structurally*, not by a clause somebody has
    /// to remember to call.
    ///
    /// With no verdict read, `modelOfRecord` is `artifact:<digest>` on both
    /// sides by derivation, so two different files are two different models and
    /// the ordinary pin comparison refuses them. There is deliberately no
    /// separate "evidence absent" clause: it could never fire, and a clause
    /// that cannot fail only makes the gate look more careful than it is.
    @Test("two different artifacts with no evidence at all are refused")
    func refusesCrossFormatWithoutEvidence() {
        #expect(
            throws: RuntimeBenchmark.AdmissionError.pinMismatch(
                field: "modelOfRecord", baseline: "artifact:\(Self.mlxDigest)",
                candidate: "artifact:\(Self.ggufDigest)")
        ) {
            try Self.admit(
                baseline: Self.mlxRecord(equivalence: .noneDeclared),
                baselineEvidence: .noneDeclared,
                candidate: Self.llamaRecord(equivalence: .noneDeclared),
                candidateEvidence: .noneDeclared)
        }
    }

    @Test("evidence on one side only is refused, naming the side that has none")
    func refusesOneSidedEvidence() {
        let reading = ModelEquivalenceReading.read(Self.verdict(), digest: Self.evidenceDigest)
        // The baseline carries the verdict; the candidate does not. Its
        // `modelOfRecord` is then `artifact:…`, which already differs from the
        // baseline's `source:…`, so the pair is refused before this clause --
        // by the pin comparison, which is the stricter of the two.
        #expect(
            throws: RuntimeBenchmark.AdmissionError.pinMismatch(
                field: "modelOfRecord", baseline: "source:\(Self.source)",
                candidate: "artifact:\(Self.ggufDigest)")
        ) {
            try Self.admit(
                baseline: Self.mlxRecord(equivalence: reading), baselineEvidence: reading,
                candidate: Self.llamaRecord(equivalence: .noneDeclared),
                candidateEvidence: .noneDeclared)
        }
    }

    /// A named verdict the gate could not read is not a verdict that was not
    /// named.
    @Test("evidence the gate could not read is refused, not spent as an absence")
    func refusesUnreadEvidence() {
        let unread = ModelEquivalenceReading.unread(path: "/tmp/equivalence.json")
        #expect(
            throws: RuntimeBenchmark.AdmissionError.modelOfRecordUnread(
                runtime: "python-mlx-lm", path: "/tmp/equivalence.json")
        ) {
            try Self.admit(
                baseline: Self.mlxRecord(equivalence: unread), baselineEvidence: unread,
                candidate: Self.llamaRecord(equivalence: unread), candidateEvidence: unread)
        }
    }

    // -------------------------------------------------------- forged evidence

    /// The self-minted-evidence shape: a record that names the upstream model
    /// while the gate read no verdict for it.
    @Test("a record that mints its own model of record is refused")
    func refusesSelfMintedModelOfRecord() {
        let forged = Fixture.record(
            runtime: "llamacpp",
            pins: Fixture.variantPins(
                modelOfRecord: "source:\(Self.source)", modelPath: Self.ggufPath,
                modelDigest: Self.ggufDigest, quantization: "Q8_0",
                contextPolicy: RuntimeBenchmark.contextPolicy(
                    derivedFrom: Self.llamaArgv, observing: .reported(76800))),
            startedAt: 300, finishedAt: 400, asymmetries: Self.nonEquivalences,
            provenance: Fixture.provenance(
                launchArgv: Self.llamaArgv, executableDigest: Fixture.digest("12")))
        let reading = ModelEquivalenceReading.read(Self.verdict(), digest: Self.evidenceDigest)
        #expect(
            throws: RuntimeBenchmark.AdmissionError.modelOfRecordNotDerived(
                runtime: "llamacpp", declared: "source:\(Self.source)",
                derived: "artifact:\(Self.ggufDigest)")
        ) {
            try Self.admit(
                baseline: Self.mlxRecord(equivalence: reading), baselineEvidence: reading,
                candidate: forged, candidateEvidence: .noneDeclared)
        }
    }

    /// Two documents that decode to the same struct are still two documents.
    ///
    /// This is the clause that makes "the same evidence" a fact about bytes.
    /// Without it a second, separately authored verdict could be pointed at one
    /// side of a pair and would look identical to the first.
    @Test("two separately authored verdicts that agree are still two verdicts")
    func refusesTwoDistinctVerdictDocuments() {
        let first = ModelEquivalenceReading.read(Self.verdict(), digest: Self.evidenceDigest)
        let second = ModelEquivalenceReading.read(Self.verdict(), digest: Fixture.digest("4c"))
        #expect(
            throws: RuntimeBenchmark.AdmissionError.equivalenceEvidenceDiffers(
                baseline: Self.evidenceDigest, candidate: Fixture.digest("4c"))
        ) {
            try Self.admit(
                baseline: Self.mlxRecord(equivalence: first), baselineEvidence: first,
                candidate: Self.llamaRecord(equivalence: second), candidateEvidence: second)
        }
    }

    // ------------------------------------------------- the verdict's contents

    @Test(
        "a verdict that is not comparable is refused",
        arguments: [ModelEquivalence.Verdict.notComparable, .incomplete])
    func refusesNonComparableVerdict(verdict: ModelEquivalence.Verdict) {
        let reading = ModelEquivalenceReading.read(
            Self.verdict(verdict: verdict), digest: Self.evidenceDigest)
        #expect(
            throws: RuntimeBenchmark.AdmissionError.equivalenceVerdictNotComparable(
                sourceOfRecord: Self.source, verdict: verdict.rawValue)
        ) {
            try Self.admit(
                baseline: Self.mlxRecord(equivalence: reading), baselineEvidence: reading,
                candidate: Self.llamaRecord(equivalence: reading), candidateEvidence: reading)
        }
    }

    /// A comparable verdict about two *other* files, pointed at these ones.
    @Test("a verdict that does not name this artifact's digest is refused")
    func refusesVerdictAboutOtherArtifacts() {
        let elsewhere = Self.verdict(artifacts: [
            ModelEquivalence.Artifact(
                path: Fixture.modelPath, digest: Self.mlxDigest,
                quantization: "8bit/group64/affine"),
            ModelEquivalence.Artifact(
                path: "/some/other-model-Q8_0.gguf", digest: "aaaaaaaa", quantization: "Q8_0"),
        ])
        let reading = ModelEquivalenceReading.read(elsewhere, digest: Self.evidenceDigest)
        #expect(
            throws: RuntimeBenchmark.AdmissionError.equivalenceDoesNotCoverArtifact(
                runtime: "llamacpp", artifact: Self.ggufPath, digest: Self.ggufDigest,
                sourceOfRecord: Self.source)
        ) {
            try Self.admit(
                baseline: Self.mlxRecord(equivalence: reading), baselineEvidence: reading,
                candidate: Self.llamaRecord(equivalence: reading), candidateEvidence: reading)
        }
    }

    @Test("a verdict that declares no non-equivalences is refused")
    func refusesVerdictWithNoDeclaredNonEquivalences() {
        let silent = Self.verdict(nonEquivalences: [])
        let reading = ModelEquivalenceReading.read(silent, digest: Self.evidenceDigest)
        #expect(
            throws: RuntimeBenchmark.AdmissionError.equivalenceDeclaresNoNonEquivalences(
                sourceOfRecord: Self.source)
        ) {
            try Self.admit(
                baseline: Self.mlxRecord(equivalence: reading, asymmetries: []),
                baselineEvidence: reading,
                candidate: Self.llamaRecord(equivalence: reading, asymmetries: []),
                candidateEvidence: reading,
                // An anchor that requires nothing, so the refusal under test is
                // the verdict's own emptiness rather than a drift from the
                // trusted decision. Both clauses exist and they refuse
                // different things.
                trusting: [
                    TrustedEquivalenceDecision(
                        sourceOfRecord: Self.source, documentDigest: Self.evidenceDigest,
                        requiredNonEquivalences: [],
                        provenance: "contract-suite fixture, not shipped")
                ])
        }
    }

    @Test("a record whose quantization disagrees with the verdict is refused")
    func refusesQuantizationDisagreement() {
        let reading = ModelEquivalenceReading.read(Self.verdict(), digest: Self.evidenceDigest)
        #expect(
            throws: RuntimeBenchmark.AdmissionError.equivalenceQuantizationDisagrees(
                runtime: "llamacpp", pinned: "Q4_K_M", declared: "Q8_0")
        ) {
            try Self.admit(
                baseline: Self.mlxRecord(equivalence: reading), baselineEvidence: reading,
                candidate: Self.llamaRecord(equivalence: reading, quantization: "Q4_K_M"),
                candidateEvidence: reading)
        }
    }

    /// The non-equivalences have to *travel*, not merely exist in the verdict.
    @Test(
        "a record that drops a declared non-equivalence is refused",
        arguments: ["python-mlx-lm", "llamacpp"])
    func refusesRecordThatDropsANonEquivalence(runtime: String) {
        let reading = ModelEquivalenceReading.read(Self.verdict(), digest: Self.evidenceDigest)
        let short = Array(Self.nonEquivalences.dropLast())
        let dropped = Self.nonEquivalences[Self.nonEquivalences.count - 1]
        let baseline = Self.mlxRecord(
            equivalence: reading, asymmetries: runtime == "python-mlx-lm" ? short : nil)
        let candidate = Self.llamaRecord(
            equivalence: reading, asymmetries: runtime == "llamacpp" ? short : nil)
        #expect(
            throws: RuntimeBenchmark.AdmissionError.declaredNonEquivalenceNotCarried(
                runtime: runtime, entry: dropped)
        ) {
            try Self.admit(
                baseline: baseline, baselineEvidence: reading,
                candidate: candidate, candidateEvidence: reading)
        }
    }

    /// Evidence beside a pair that does not need it is a claim nobody checked.
    @Test("a same-artifact pair that cites equivalence evidence is refused")
    func refusesUnusedEvidence() {
        let sameArtifactVerdict = Self.verdict(artifacts: [
            ModelEquivalence.Artifact(
                path: Fixture.modelPath, digest: Self.mlxDigest,
                quantization: "8bit/group64/affine")
        ])
        let reading = ModelEquivalenceReading.read(
            sameArtifactVerdict, digest: Self.evidenceDigest)
        let baseline = Self.mlxRecord(equivalence: reading)
        let candidate = Fixture.record(
            runtime: "mlx-swift",
            pins: baseline.pins, startedAt: 300, finishedAt: 400,
            asymmetries: Self.nonEquivalences,
            provenance: Fixture.provenance(
                launchArgv: Self.mlxArgv, executableDigest: Fixture.digest("12")))
        #expect(
            throws: RuntimeBenchmark.AdmissionError.equivalenceEvidenceUnused(
                runtime: "python-mlx-lm", sourceOfRecord: Self.source)
        ) {
            try Self.admit(
                baseline: baseline, baselineEvidence: reading,
                candidate: candidate, candidateEvidence: reading)
        }
    }

    // ---------------------------------------------------------- the seal

    /// The non-equivalences are inside the observer's seal since G4, so a
    /// record cannot lose one after the pass that produced it.
    @Test("deleting a declared non-equivalence breaks the observation's seal")
    func sealCoversDeclaredNonEquivalences() {
        let reading = ModelEquivalenceReading.read(Self.verdict(), digest: Self.evidenceDigest)
        let full = Self.llamaRecord(equivalence: reading)
        let stripped = Self.llamaRecord(
            equivalence: reading, asymmetries: Array(Self.nonEquivalences.dropLast()))
        #expect(
            RuntimeBenchmark.transcriptDigest(of: full)
                != RuntimeBenchmark.transcriptDigest(of: stripped))
    }
}

/// Speculative decoding: not a condition two runtimes can share.
///
/// `llama-server` can draft off this model's MTP head and the MLX baseline has
/// no MTP head at all, so a tokens/s measured with speculation on is a
/// different decoding algorithm rather than a faster runtime.
/// TASK-260828-3g87i4's equivalence verdict names this as the one way the pair
/// genuinely stops being comparable, and the gate therefore *refuses* it rather
/// than requiring the two sides to match — two speculating runtimes would agree
/// on the pin and still not be a migration result.
///
/// Every flag read here was taken off `llama-server --help` for the pinned
/// build `b10621-c1d0e7a00`. `--spec-type` defaults to `none` and `/slots`
/// reports `params.speculative` `true` under `--spec-type ngram-mod`, measured
/// on that build with a `Qwen2.5-0.5B-Instruct` Q8_0 fixture; `/props` reports
/// `"none"` in both cases and is deliberately not the source.
///
/// Production call site: `BenchmarkRunCommand.speculationAnswer` performs the
/// `GET /slots`, `drive` writes it onto the attestation and derives the pin
/// from it, and `RuntimeBenchmark.admitProvenance` re-derives and refuses.
@Suite("runtime benchmark speculative decoding")
struct RuntimeBenchmarkSpeculationTests {
    typealias Fixture = RuntimeBenchmarkTests

    /// Both passes launched and observed the same way, on purpose.
    ///
    /// A pair in which only one side speculates is already refused by the
    /// ``RuntimeBenchmark/Pins/speculation`` equality pin, which is the more
    /// general refusal. The case that needs a clause of its own is the one the
    /// pin comparison cannot see: **both** runtimes speculating, agreeing on
    /// every pin, and still not a migration result. That is what these tests
    /// drive, and it is why the gate refuses this reading rather than merely
    /// requiring the two sides to match.
    ///
    /// `speculationPin` overrides what the records *declare* without changing
    /// what the gate observed, which is the record-mints-its-own-policy shape.
    static func pair(
        argv: [String] = Fixture.launchArgv,
        speculation: RuntimeSpeculation,
        speculationPin: String? = nil
    ) throws -> RuntimeBenchmark.Comparison {
        let pins = Fixture.variantPins(
            contextPolicy: RuntimeBenchmark.contextPolicy(
                derivedFrom: argv, observing: .notReported),
            speculation: speculationPin
                ?? RuntimeBenchmark.speculationPolicy(
                    derivedFrom: argv, observing: speculation))
        func record(_ runtime: String, at start: Double, executable: String)
            -> RuntimeBenchmark.RunRecord
        {
            Fixture.record(
                runtime: runtime, pins: pins, startedAt: start, finishedAt: start + 100,
                provenance: Fixture.provenance(
                    launchArgv: argv, executableDigest: Fixture.digest(executable)))
        }
        let baseline = record("python-mlx-lm", at: 100, executable: "cd")
        let candidate = record("mlx-swift", at: 300, executable: "12")
        return try RuntimeBenchmark.admit(
            baseline: baseline,
            baselineAttestation: Fixture.attestation(for: baseline, speculation: speculation),
            candidate: candidate,
            candidateAttestation: Fixture.attestation(for: candidate, speculation: speculation),
            requiredScenarios: ["short_prompt"], gateBinaryDigest: Fixture.gateDigest)
    }

    /// Positive, and the narrowing target: a gate that refused every runtime
    /// that answers `/slots` at all, or that refused an explicit
    /// `--spec-type none`, would break the case llama.cpp is supposed to reach.
    @Test("a runtime that reports it is not speculating is admitted")
    func admitsARuntimeThatReportsNoSpeculation() throws {
        let comparison = try Self.pair(speculation: .reported(false))
        #expect(comparison.pins.speculation == "off")
    }

    @Test("an explicit --spec-type none is a launch that asked for nothing")
    func admitsAnExplicitlyDisabledSpeculation() throws {
        let argv = Fixture.launchArgv + ["--spec-type", "none"]
        let comparison = try Self.pair(argv: argv, speculation: .notReported)
        #expect(comparison.pins.speculation == "off")
    }

    @Test("a runtime measured to be speculating is refused")
    func refusesAReportedSpeculatingRuntime() {
        #expect(
            throws: RuntimeBenchmark.AdmissionError.speculativeDecodingActive(
                runtime: "python-mlx-lm", reading: "on")
        ) {
            try Self.pair(speculation: .reported(true))
        }
    }

    /// A failed read is not an absence, so it does not become `off`.
    @Test("a speculation state the gate could not read is refused, not defaulted")
    func refusesUnreadSpeculation() {
        #expect(
            throws: RuntimeBenchmark.AdmissionError.speculativeDecodingActive(
                runtime: "python-mlx-lm", reading: "unread")
        ) {
            try Self.pair(speculation: .unread)
        }
    }

    /// The `/slots`-less bypass: launch into speculation against a runtime that
    /// will not answer the question. The argv reading closes it.
    @Test(
        "a launch that asks for speculation against a silent runtime is refused",
        arguments: [
            (["--spec-type", "ngram-mod"], "declared:--spec-type=ngram-mod"),
            (["--spec-type", "none,draft-mtp"], "declared:--spec-type=draft-mtp"),
            (
                ["--spec-draft-model", "/tmp/draft.gguf"],
                "declared:--spec-draft-model=/tmp/draft.gguf"
            ),
            (["-md", "/tmp/draft.gguf"], "declared:-md=/tmp/draft.gguf"),
        ])
    func refusesDeclaredSpeculationWithoutAnAnswer(argv: [String], reading: String) {
        #expect(
            throws: RuntimeBenchmark.AdmissionError.speculativeDecodingActive(
                runtime: "python-mlx-lm", reading: reading)
        ) {
            try Self.pair(argv: Fixture.launchArgv + argv, speculation: .notReported)
        }
    }

    /// The launch and the process disagree about which algorithm ran. Neither
    /// reading can be preferred, so the pair is refused for *that* rather than
    /// having the launch's request quietly disappear into the process's answer.
    @Test("a launch that asked for speculation against a process reporting none is refused")
    func refusesSpeculationTheProcessDidNotHonour() {
        #expect(
            throws: RuntimeBenchmark.AdmissionError.speculationNotHonoured(
                runtime: "python-mlx-lm", flag: "--spec-type", declared: "ngram-mod")
        ) {
            try Self.pair(
                argv: Fixture.launchArgv + ["--spec-type", "ngram-mod"],
                speculation: .reported(false))
        }
    }

    /// A record cannot declare itself quiet.
    @Test("a record that pins speculation off while the process reported it on is refused")
    func refusesADeclaredSpeculationPin() {
        #expect(
            throws: RuntimeBenchmark.AdmissionError.speculationNotDerived(
                runtime: "python-mlx-lm", declared: "off", derived: "on")
        ) {
            try Self.pair(speculation: .reported(true), speculationPin: "off")
        }
    }

    /// `--batch-size` is not `--ubatch-size` and `--spec-draft-threads` is not
    /// `--spec-type`: a flag that merely *configures* a draft path does not by
    /// itself say speculation was asked for, and reading it as one would refuse
    /// launches that are not speculating.
    @Test("a speculative tuning flag with no draft path is not a declaration")
    func doesNotReadTuningFlagsAsADeclaration() {
        #expect(
            RuntimeBenchmark.declaredSpeculation(
                inArgv: ["--spec-draft-threads", "4", "--spec-draft-n-max", "3"]) == nil)
    }
}
