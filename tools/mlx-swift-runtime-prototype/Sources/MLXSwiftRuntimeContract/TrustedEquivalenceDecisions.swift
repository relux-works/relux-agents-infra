import Foundation

/// One equivalence decision this repository trusts, named here and nowhere the
/// invocation can reach.
///
/// This type exists because of F1. `--equivalence` used to name a JSON path,
/// and the gate read it, hashed it, carried the hash onto both attestations and
/// sealed it. Every one of those steps is real and none of them authenticates
/// anything: review minted a well-shaped verdict naming an arbitrary source of
/// record, the two artifact digests the gate had itself computed, `comparable`,
/// and one generic note, and the shipped `benchmark-run` accepted the pair with
/// exit 0. Hashing attacker-authored bytes proves the bytes did not change
/// between the read and the seal. It says nothing about who decided.
///
/// So admission is bound to a decision the invocation **cannot author for
/// itself**: a fixed list, compiled into the gate binary from versioned
/// repository source, of the equivalence decisions that have actually been
/// taken. A caller may still hand the gate a document — it has to, because the
/// verdict's contents travel into both records — but the only documents that
/// admit are the ones whose SHA-256 already appears here. Changing that list
/// means changing this file and rebuilding the gate, which is the same class
/// boundary the rest of this gate rests on: producing a false acceptance
/// requires modifying the gate rather than using it.
///
/// ``requiredNonEquivalences`` is carried explicitly rather than left implicit
/// in the document the digest fixes. The digest already pins the document's
/// contents, so in a run where the two agree these entries are an equality that
/// holds; what they guard is the case where they *stop* agreeing — a later
/// decision added here with its digest updated and one of the measured
/// differences quietly dropped to a single generic note. That is precisely the
/// substitution F1 demonstrated, so the required entries are stated where a
/// reviewer reads them, demanded of the document, and demanded again of both
/// records at
/// ``RuntimeBenchmark/admitModelIdentity(baseline:baselineAttestation:candidate:candidateAttestation:trusting:)``.
public struct TrustedEquivalenceDecision: Sendable, Equatable {
    /// The upstream model the decision is about. Must match the document's.
    public let sourceOfRecord: String
    /// SHA-256 over the exact bytes of the decision document.
    ///
    /// This is the whole trust anchor. A document that hashes to this is the
    /// decision; anything else is a claim the invocation made.
    public let documentDigest: String
    /// The measured differences the decision found and did not dissolve, every
    /// one of which must appear in the document and in both records.
    public let requiredNonEquivalences: [String]
    /// Where the decision came from, for a human reading a refusal. Never
    /// matched on.
    public let provenance: String

    public init(
        sourceOfRecord: String,
        documentDigest: String,
        requiredNonEquivalences: [String],
        provenance: String
    ) {
        self.sourceOfRecord = sourceOfRecord
        self.documentDigest = documentDigest
        self.requiredNonEquivalences = requiredNonEquivalences
        self.provenance = provenance
    }
}

/// The equivalence decisions this repository has actually taken.
///
/// Deliberately short, and deliberately holding no fixture entry. A trusted
/// anchor written so that a smoke script could reach an acceptance would be a
/// decision nobody made, sitting in the production trust store — the F1 defect
/// with a test's name on it. `scripts/benchmark-gate-smoke.sh` therefore proves
/// the *refusals* at the production entry and proves that the trusted path is
/// live there by driving the real document past the trust clause into the
/// artifact-coverage one; the admitted class is pinned by the contract suite
/// and by the one-off run against the real 29 GB pair recorded in
/// `.research/260828_llamacpp-in-the-benchmark-gate.md`.
public enum TrustedEquivalenceDecisions {
    /// The canonical document for each entry lives in `equivalence/` beside the
    /// package, versioned with it. `trustedDecisionsMatchTheirDocuments` in the
    /// contract suite reads those files and refuses a drift between the bytes
    /// on disk and the anchors here.
    public static let shipped: [TrustedEquivalenceDecision] = [
        TrustedEquivalenceDecision(
            sourceOfRecord: "hf:orcarouter/Qwen3.8-27B-Uncensored-BF16",
            documentDigest:
                "106edbf472177b055a149dda5cff3c8c86e13d1278a8ec508789f1197f09f962",
            requiredNonEquivalences: [
                "the MLX build drops the MTP head the GGUF carries as blk.64: 8 quantized "
                    + "tensors, 451319808 bytes on disk, skipped at load",
                "the vision tower is placed differently -- 333 bf16 tensors inside the MLX "
                    + "safetensors shards against a separate 931145984-byte GGUF mmproj -- and "
                    + "is resident in neither on the default text path",
                "GGUF upcasts norms and 1-D tensors to F32 where the MLX build keeps the source "
                    + "bf16: 10686464 bytes of extra resident memory, no fidelity difference",
            ],
            provenance:
                "TASK-260828-3g87i4 revision 3, quantization-equivalence.md section 6; document "
                + "at equivalence/qwen3-8-27b-uncensored.equivalence.json")
    ]

    /// The trusted decision a document's digest names, or `nil`.
    ///
    /// Matched on the digest and on nothing else. Matching on the source of
    /// record would let a caller mint a document that names a trusted upstream
    /// and says whatever it likes about the artifacts, which is the attack.
    public static func decision(
        documentDigest: String, in decisions: [TrustedEquivalenceDecision] = shipped
    ) -> TrustedEquivalenceDecision? {
        decisions.first { $0.documentDigest == documentDigest }
    }
}
