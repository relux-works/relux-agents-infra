import CryptoKit
import Foundation
import Testing

@testable import MLXSwiftRuntimeContract

/// F1: who decided that two different weight files are one model.
///
/// The gate used to read the answer out of a JSON path the caller supplied. It
/// read it correctly, digested it, carried the digest onto both attestations
/// and sealed it — and none of that authenticates anything, because the caller
/// wrote the bytes. Review minted a well-shaped verdict naming an arbitrary
/// source of record, the two artifact digests the gate had itself computed,
/// `comparable`, and one generic note, and the shipped `benchmark-run` accepted
/// the pair with exit 0 while both records carried only that invented note.
///
/// So admission is bound to ``TrustedEquivalenceDecisions/shipped``: a fixed
/// list, compiled into the gate from versioned repository source, of the
/// equivalence decisions that have actually been taken. This suite drives that
/// store **unstubbed**. Every `admit` call below omits `trusting:`, so what is
/// under test is the store the production call sites use and not a registry a
/// test handed in.
///
/// **Production call site.** `BenchmarkRunCommand.equivalenceReading(path:)`
/// performs the same lookup on the document the caller named and returns
/// ``ModelEquivalenceReading/untrusted(path:digest:)`` when it misses;
/// `BenchmarkRunCommand.execute` refuses that before any launch, and
/// `RuntimeBenchmark.admitProvenance` refuses it again from the attestation, so
/// a hand-authored attestation cannot reintroduce it. `scripts/benchmark-gate-smoke.sh`
/// section 4 drives a caller-authored verdict over the real fixture files
/// through the shipped subcommand and requires the refusal.
@Suite("runtime benchmark trusted equivalence")
struct RuntimeBenchmarkTrustedEquivalenceTests {
    typealias Fixture = RuntimeBenchmarkTests

    /// The one decision this repository has taken, as the gate carries it.
    static let anchor = TrustedEquivalenceDecisions.shipped[0]

    static let documentPath = URL(fileURLWithPath: #filePath)
        .deletingLastPathComponent()
        .deletingLastPathComponent()
        .deletingLastPathComponent()
        .appendingPathComponent("equivalence/qwen3-8-27b-uncensored.equivalence.json")

    static let mlxPath = Fixture.modelPath
    static let ggufPath =
        "/Users/alexis/src/local-models/Qwen3.8-27B-Uncensored-GGUF-Q8_0/"
        + "Qwen3.8-27B-Uncensored-OrcaRouter-Q8_0.gguf"

    /// The digests the gate computes over the two real artifacts, recomputed on
    /// this host: the MLX directory over `config.json` and the safetensors
    /// index, the GGUF over all 29 047 084 416 of its bytes.
    static let mlxDigest = "1b10f3fe1c1097c909fa35e112b943255c44be4a5f332f45e0af57a96188460b"
    static let ggufDigest = "31756fca94beca71ea4b8706d6fdc896dab2a3c6376ab0c1863b98512a24f8d6"

    static func documentBytes() throws -> Data {
        try Data(contentsOf: documentPath)
    }

    static func documentDigest() throws -> String {
        SHA256.hash(data: try documentBytes()).map { String(format: "%02x", $0) }.joined()
    }

    static func document() throws -> ModelEquivalence {
        try JSONDecoder().decode(ModelEquivalence.self, from: try documentBytes())
    }

    // --------------------------------------------- the store and its document

    /// The trust store and the file it names must not drift apart.
    ///
    /// Both are versioned repository source and neither is generated from the
    /// other, so this is the check that keeps them one decision. It also
    /// asserts the three measured differences by name: F1's substitution was a
    /// single generic note standing in for all of them, and the whole point of
    /// stating them in the trust store is that a reviewer reads them there.
    @Test("the shipped decision is the document it names, non-equivalences and all")
    func trustedDecisionMatchesItsDocument() throws {
        #expect(try Self.documentDigest() == Self.anchor.documentDigest)
        let document = try Self.document()
        #expect(document.sourceOfRecord == Self.anchor.sourceOfRecord)
        #expect(document.verdict == .comparable)
        for entry in Self.anchor.requiredNonEquivalences {
            #expect(document.declaredNonEquivalences.contains(entry))
        }
        #expect(Self.anchor.requiredNonEquivalences.count == 3)
        #expect(document.artifact(digest: Self.mlxDigest)?.quantization == "8bit/group64/affine")
        #expect(document.artifact(digest: Self.ggufDigest)?.quantization == "Q8_0")
    }

    /// A decision is found by the digest of its document and by nothing else.
    @Test("the trust lookup matches on the document digest, never on the upstream name")
    func lookupIsByDigestOnly() {
        #expect(
            TrustedEquivalenceDecisions.decision(documentDigest: Self.anchor.documentDigest)
                != nil)
        #expect(TrustedEquivalenceDecisions.decision(documentDigest: Fixture.digest("3b")) == nil)
        // A minted document naming the trusted upstream is still not the
        // decision: the store holds no entry at its digest.
        #expect(TrustedEquivalenceDecisions.decision(documentDigest: "") == nil)
    }

    // ------------------------------------------------------- the four readings

    /// An untrusted reading is its own fact and derives its own pin.
    ///
    /// Collapsing it into any of the other three is the F1 defect in a
    /// different shape, so the derivation is asserted against all three.
    @Test("a well-formed verdict nobody trusts derives neither an artifact pin nor a source")
    func untrustedDerivesItsOwnModelOfRecord() {
        let untrusted = ModelEquivalenceReading.untrusted(
            path: "/tmp/minted.json", digest: Fixture.digest("3b"))
        let derived = RuntimeBenchmark.modelOfRecord(
            artifactDigest: Self.ggufDigest, observing: untrusted)
        #expect(derived == RuntimeBenchmark.untrustedModelOfRecord)
        #expect(derived != RuntimeBenchmark.unreadModelOfRecord)
        #expect(derived != "artifact:\(Self.ggufDigest)")
        #expect(derived != "source:\(Self.anchor.sourceOfRecord)")
    }

    @Test("an untrusted reading survives a round trip as itself")
    func untrustedRoundTrips() throws {
        let reading = ModelEquivalenceReading.untrusted(
            path: "/tmp/minted.json", digest: Fixture.digest("3b"))
        let encoded = try JSONEncoder().encode(reading)
        #expect(try JSONDecoder().decode(ModelEquivalenceReading.self, from: encoded) == reading)
    }

    // ------------------------------------------------------------- the records

    static let mlxArgv = Fixture.pythonLaunchArgv
    static let llamaArgv = [
        "--model", ggufPath, "--host", "127.0.0.1", "--port", "18031",
        "--ctx-size", "76800", "--ubatch-size", "2048", "--reasoning-effort", "medium",
    ]

    static func mlxRecord(
        equivalence: ModelEquivalenceReading, asymmetries: [String]
    ) -> RuntimeBenchmark.RunRecord {
        Fixture.record(
            runtime: "python-mlx-lm",
            pins: Fixture.variantPins(
                modelOfRecord: RuntimeBenchmark.modelOfRecord(
                    artifactDigest: mlxDigest, observing: equivalence),
                modelDigest: mlxDigest, quantization: "8bit/group64/affine",
                contextPolicy: RuntimeBenchmark.contextPolicy(
                    derivedFrom: mlxArgv, observing: .reported(76800))),
            startedAt: 100, finishedAt: 200, asymmetries: asymmetries,
            provenance: Fixture.provenance(
                launchArgv: mlxArgv, executableDigest: Fixture.digest("cd")))
    }

    static func llamaRecord(
        equivalence: ModelEquivalenceReading, asymmetries: [String]
    ) -> RuntimeBenchmark.RunRecord {
        Fixture.record(
            runtime: "llamacpp",
            pins: Fixture.variantPins(
                modelOfRecord: RuntimeBenchmark.modelOfRecord(
                    artifactDigest: ggufDigest, observing: equivalence),
                modelPath: ggufPath, modelDigest: ggufDigest, quantization: "Q8_0",
                contextPolicy: RuntimeBenchmark.contextPolicy(
                    derivedFrom: llamaArgv, observing: .reported(76800))),
            startedAt: 300, finishedAt: 400, asymmetries: asymmetries,
            provenance: Fixture.provenance(
                launchArgv: llamaArgv, executableDigest: Fixture.digest("12")))
    }

    /// Admission against the **shipped** trust store. No registry is injected
    /// anywhere in this suite; `trusting:` is deliberately never passed.
    static func admit(
        baselineEvidence: ModelEquivalenceReading,
        candidateEvidence: ModelEquivalenceReading,
        asymmetries: [String],
        candidateAsymmetries: [String]? = nil
    ) throws -> RuntimeBenchmark.Comparison {
        let baseline = mlxRecord(equivalence: baselineEvidence, asymmetries: asymmetries)
        let candidate = llamaRecord(
            equivalence: candidateEvidence, asymmetries: candidateAsymmetries ?? asymmetries)
        return try RuntimeBenchmark.admit(
            baseline: baseline,
            baselineAttestation: Fixture.attestation(
                for: baseline, contextWindow: .reported(76800), speculation: .notReported,
                equivalence: baselineEvidence),
            candidate: candidate,
            candidateAttestation: Fixture.attestation(
                for: candidate, contextWindow: .reported(76800), speculation: .reported(false),
                equivalence: candidateEvidence),
            requiredScenarios: ["short_prompt"], gateBinaryDigest: Fixture.gateDigest)
    }

    // ---------------------------------------------------- the admitted class

    /// The positive, and it stands on the shipped decision alone.
    ///
    /// The real MLX 8-bit directory against the real Q8_0 GGUF, at the digests
    /// this host computed for both, under the document
    /// `equivalence/qwen3-8-27b-uncensored.equivalence.json` read off disk and
    /// digested here. Nothing is stubbed: the trust store is the one the gate
    /// ships. A *narrowing* mutant — one that stops trusting the shipped
    /// decision, demands byte identity after all, or drops a required
    /// non-equivalence from the anchor — reddens this and only this kind of
    /// test.
    @Test("the real cross-format pair is admitted under the decision this repository took")
    func admitsTheShippedDecision() throws {
        let reading = ModelEquivalenceReading.read(
            try Self.document(), digest: try Self.documentDigest())
        let comparison = try Self.admit(
            baselineEvidence: reading, candidateEvidence: reading,
            asymmetries: Self.anchor.requiredNonEquivalences)
        #expect(comparison.baseline.pins.modelOfRecord == "source:\(Self.anchor.sourceOfRecord)")
        #expect(comparison.candidate.pins.modelOfRecord == comparison.baseline.pins.modelOfRecord)
    }

    // ------------------------------------------------------------ the attacks

    /// F1 itself, at the default trust store: the caller mints a verdict that
    /// is right in every respect the old gate checked.
    ///
    /// `comparable`, naming the real upstream, naming both artifacts at the
    /// digests the gate computed, with the correct quantization labels, and
    /// with a note. Every clause the previous revision had is satisfied. It is
    /// still not a decision anybody took.
    @Test("a caller-minted verdict that satisfies every other clause is refused")
    func refusesASelfMintedVerdict() throws {
        let minted = ModelEquivalence(
            sourceOfRecord: Self.anchor.sourceOfRecord,
            verdict: .comparable,
            artifacts: [
                ModelEquivalence.Artifact(
                    path: Self.mlxPath, digest: Self.mlxDigest,
                    quantization: "8bit/group64/affine"),
                ModelEquivalence.Artifact(
                    path: Self.ggufPath, digest: Self.ggufDigest, quantization: "Q8_0"),
            ],
            declaredNonEquivalences: ["the two builds differ in quantization scheme"])
        let reading = ModelEquivalenceReading.read(minted, digest: Fixture.digest("3b"))
        #expect(
            throws: RuntimeBenchmark.AdmissionError.equivalenceEvidenceUntrusted(
                sourceOfRecord: Self.anchor.sourceOfRecord, digest: Fixture.digest("3b"))
        ) {
            try Self.admit(
                baselineEvidence: reading, candidateEvidence: reading,
                asymmetries: ["the two builds differ in quantization scheme"])
        }
    }

    /// The same minted verdict carrying the three real notes verbatim.
    ///
    /// Copying the decision's words is not taking the decision, and this is the
    /// case that would survive a trust check written against the
    /// non-equivalences instead of against the document.
    @Test("a minted verdict that copies the required non-equivalences is still refused")
    func refusesAMintedVerdictThatQuotesTheDecision() throws {
        let minted = ModelEquivalence(
            sourceOfRecord: Self.anchor.sourceOfRecord,
            verdict: .comparable,
            artifacts: try Self.document().artifacts,
            declaredNonEquivalences: Self.anchor.requiredNonEquivalences)
        let reading = ModelEquivalenceReading.read(minted, digest: Fixture.digest("4c"))
        #expect(
            throws: RuntimeBenchmark.AdmissionError.equivalenceEvidenceUntrusted(
                sourceOfRecord: Self.anchor.sourceOfRecord, digest: Fixture.digest("4c"))
        ) {
            try Self.admit(
                baselineEvidence: reading, candidateEvidence: reading,
                asymmetries: Self.anchor.requiredNonEquivalences)
        }
    }

    /// An untrusted *reading* — what `equivalenceReading(path:)` returns — is
    /// refused by name from the attestation, so a hand-authored attestation
    /// cannot smuggle one past the production entry's refusal.
    @Test("an untrusted reading on the attestation is refused by name")
    func refusesAnUntrustedReadingOnTheAttestation() throws {
        let reading = ModelEquivalenceReading.untrusted(
            path: "/tmp/minted.json", digest: Fixture.digest("3b"))
        #expect(
            throws: RuntimeBenchmark.AdmissionError.modelOfRecordUntrusted(
                runtime: "python-mlx-lm", path: "/tmp/minted.json",
                digest: Fixture.digest("3b"))
        ) {
            try Self.admit(
                baselineEvidence: reading, candidateEvidence: reading,
                asymmetries: Self.anchor.requiredNonEquivalences)
        }
    }

    /// The three measured differences have to reach both records, and the
    /// trusted decision is what says which three.
    ///
    /// This is F1's other half: the accepted pair carried a single generic note
    /// where the decision names a dropped MTP head, a vision-tower placement
    /// and F32-versus-bf16 norms, and a report taken from that decision could
    /// be read without any of them.
    @Test(
        "a record that carries a generic note instead of the decision's is refused",
        arguments: ["python-mlx-lm", "llamacpp"])
    func refusesRecordsThatLoseTheDecisionsNonEquivalences(runtime: String) throws {
        let reading = ModelEquivalenceReading.read(
            try Self.document(), digest: try Self.documentDigest())
        let generic = ["the two builds differ in quantization scheme"]
        let dropped = Self.anchor.requiredNonEquivalences[0]
        #expect(
            throws: RuntimeBenchmark.AdmissionError.declaredNonEquivalenceNotCarried(
                runtime: runtime, entry: dropped)
        ) {
            try Self.admit(
                baselineEvidence: reading, candidateEvidence: reading,
                asymmetries: runtime == "python-mlx-lm"
                    ? generic : Self.anchor.requiredNonEquivalences,
                candidateAsymmetries: runtime == "llamacpp"
                    ? generic : Self.anchor.requiredNonEquivalences)
        }
    }

    // --------------------------------------------- drift between the two files

    /// A trusted decision and a document that hashes to it but says something
    /// else is a drift, and it is refused as one.
    ///
    /// Reachable only by editing the trust store, which is why these two use an
    /// injected registry: they are about the store's own self-consistency
    /// rather than about what a caller can do, and there is no way to express
    /// them against the shipped entry without breaking it.
    @Test("a document naming an upstream its trusted decision does not is refused")
    func refusesADocumentThatRenamesTheUpstream() throws {
        let document = try Self.document()
        let drifted = [
            TrustedEquivalenceDecision(
                sourceOfRecord: "hf:somebody-else/Qwen3.8-27B",
                documentDigest: try Self.documentDigest(),
                requiredNonEquivalences: Self.anchor.requiredNonEquivalences,
                provenance: "contract-suite fixture, not shipped")
        ]
        let reading = ModelEquivalenceReading.read(document, digest: try Self.documentDigest())
        let baseline = Self.mlxRecord(
            equivalence: reading, asymmetries: Self.anchor.requiredNonEquivalences)
        let candidate = Self.llamaRecord(
            equivalence: reading, asymmetries: Self.anchor.requiredNonEquivalences)
        #expect(
            throws: RuntimeBenchmark.AdmissionError.trustedDecisionDisagrees(
                sourceOfRecord: "hf:somebody-else/Qwen3.8-27B",
                detail: "the document names upstream model "
                    + document.sourceOfRecord.debugDescription)
        ) {
            try RuntimeBenchmark.admitModelIdentity(
                baseline: baseline,
                baselineAttestation: Fixture.attestation(for: baseline, equivalence: reading),
                candidate: candidate,
                candidateAttestation: Fixture.attestation(for: candidate, equivalence: reading),
                trusting: drifted)
        }
    }

    @Test("a document that dropped a required non-equivalence is refused")
    func refusesADocumentThatDroppedARequiredNonEquivalence() throws {
        let required = "the MTP head is dropped and this document forgot to say so"
        let drifted = [
            TrustedEquivalenceDecision(
                sourceOfRecord: Self.anchor.sourceOfRecord,
                documentDigest: try Self.documentDigest(),
                requiredNonEquivalences: [required],
                provenance: "contract-suite fixture, not shipped")
        ]
        let reading = ModelEquivalenceReading.read(
            try Self.document(), digest: try Self.documentDigest())
        let baseline = Self.mlxRecord(equivalence: reading, asymmetries: [required])
        let candidate = Self.llamaRecord(equivalence: reading, asymmetries: [required])
        #expect(
            throws: RuntimeBenchmark.AdmissionError.trustedDecisionDisagrees(
                sourceOfRecord: Self.anchor.sourceOfRecord,
                detail: "the document does not declare " + required.debugDescription)
        ) {
            try RuntimeBenchmark.admitModelIdentity(
                baseline: baseline,
                baselineAttestation: Fixture.attestation(for: baseline, equivalence: reading),
                candidate: candidate,
                candidateAttestation: Fixture.attestation(for: candidate, equivalence: reading),
                trusting: drifted)
        }
    }
}
