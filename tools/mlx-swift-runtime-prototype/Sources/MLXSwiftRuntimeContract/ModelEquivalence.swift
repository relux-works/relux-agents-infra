import Foundation

/// A declared verdict that two *different* local weight artifacts are derived
/// from one upstream model and are comparable to each other.
///
/// This type exists because of G4. `modelPath` and `modelDigest` were pins
/// compared for byte equality, which asks a cross-format comparison for
/// something that can never be true of it: MLX 8-bit group64 and GGUF Q8_0 are
/// two different files, at two different paths, with two different digests,
/// derived from the same upstream BF16 weights. A gate that demands equality
/// there refuses forever on a premise that does not fit the question — not
/// because the comparison is unsound, but because the pin was written about the
/// local file rather than about the model.
///
/// The decision taken for this story is therefore: **the pin identifies the
/// shared source of record, and byte-identity of the local artifact is replaced
/// by declared, digest-bound equivalence evidence — never by nothing.**
///
/// What that buys, and what it costs, stated together:
///
/// * ``RuntimeBenchmark/Pins/modelOfRecord`` is what the two runs must agree
///   on. For a same-format pair it is `artifact:<digest>`, so byte identity is
///   still exactly what is pinned and nothing is relaxed. For a cross-format
///   pair it is `source:<sourceOfRecord>`, taken from a verdict artifact the
///   gate read itself.
/// * A cross-format pair must additionally satisfy
///   ``RuntimeBenchmark/admitModelIdentity(baseline:baselineAttestation:candidate:candidateAttestation:trusting:)``,
///   which is strictly more work than the equality it replaces: the document
///   must be one of the equivalence decisions this repository took (see
///   ``TrustedEquivalenceDecision``), the same document on both sides, a
///   `comparable` verdict, both gate-computed artifact digests named *by that
///   document*, a non-empty set of declared non-equivalences, and every one of
///   them carried in both records'
///   ``RuntimeBenchmark/RunRecord/declaredAsymmetries``.
/// * Absence of the evidence is a refusal, not a default pass. So is a failed
///   read of it, and so is a document the invocation authored for itself — the
///   three are different facts and have three different refusals. See
///   ``ModelEquivalenceReading``.
///
/// The document this describes is the one TASK-260828-3g87i4 produced for the
/// Qwen3.8-27B pair: both schemes cost 8.5 bits per weight, the quantized
/// tensor sets match apart from the MTP block, and mean relative RMS against
/// the shared BF16 source is 0.766. Its verdict is `comparable` with named
/// conditions, and its declared non-equivalences are what
/// ``declaredNonEquivalences`` carries into every record and every report.
///
/// **What this does not claim.** The verdict is evidence somebody produced, not
/// a measurement this gate performed; the gate binds it to the artifacts by
/// digest and refuses when it does not fit, and that is all. A wrong verdict
/// about the right files is outside what any admission clause can see, which is
/// why the non-equivalences travel with the record instead of being consumed
/// and dropped.
public struct ModelEquivalence: Sendable, Equatable, Codable {
    /// Whether the two artifacts may be compared at all.
    ///
    /// Only ``comparable`` admits. The other two are carried rather than
    /// collapsed into "not comparable" because they are different facts: a
    /// measured non-equivalence and an unfinished analysis call for different
    /// work, and a gate that reported one as the other would be guessing.
    public enum Verdict: String, Sendable, Equatable, Codable {
        case comparable
        case notComparable
        case incomplete
    }

    /// One local artifact the verdict is about.
    public struct Artifact: Sendable, Equatable, Codable {
        /// Where the verdict found it. Recorded for a reader; never matched on,
        /// because a path is not an identity.
        public let path: String
        /// SHA-256 of the artifact as the verdict computed it. **This** is what
        /// binds the verdict to a file: admission requires the digest the gate
        /// computed for the artifact under test to appear here.
        public let digest: String
        /// The quantization scheme this artifact carries, in its own runtime's
        /// spelling — `8bit/group64/affine`, `Q8_0`.
        public let quantization: String

        public init(path: String, digest: String, quantization: String) {
            self.path = path
            self.digest = digest
            self.quantization = quantization
        }
    }

    /// The upstream model both artifacts descend from, named the same way in
    /// both records. This is the thing the pin identifies.
    public let sourceOfRecord: String
    public let verdict: Verdict
    /// Every artifact the verdict covers. A comparison is admitted only when
    /// both sides' gate-computed digests appear here.
    public let artifacts: [Artifact]
    /// The differences the verdict found and did **not** dissolve.
    ///
    /// These are not commentary. They are copied into both records'
    /// ``RuntimeBenchmark/RunRecord/declaredAsymmetries`` by the gate and
    /// re-checked at admission, so a decision produced from this pair cannot be
    /// read without them. For the Qwen3.8-27B pair they are the dropped MTP
    /// head, the vision-tower placement, and F32 versus bf16 norms.
    ///
    /// Required non-empty for a cross-format admission: a verdict that found
    /// two differently quantized artifacts identical in every respect has not
    /// looked.
    public let declaredNonEquivalences: [String]

    public init(
        sourceOfRecord: String,
        verdict: Verdict,
        artifacts: [Artifact],
        declaredNonEquivalences: [String]
    ) {
        self.sourceOfRecord = sourceOfRecord
        self.verdict = verdict
        self.artifacts = artifacts
        self.declaredNonEquivalences = declaredNonEquivalences
    }

    /// The artifact this verdict names at a given digest, or `nil`.
    ///
    /// Matched on the digest and on nothing else. A path match would let a
    /// verdict about yesterday's file describe today's.
    public func artifact(digest: String) -> Artifact? {
        artifacts.first { $0.digest == digest }
    }
}

/// What the gate found when it went looking for equivalence evidence.
///
/// Four cases, kept apart for the same reason ``RuntimeContextWindow`` keeps
/// its three apart: collapsing any two of them is the defect in a different
/// shape.
///
/// * ``noneDeclared`` — no verdict artifact was named. The local artifact is
///   its own source of record, which is the same-format case and pins byte
///   identity exactly as before.
/// * ``read(_:digest:)`` — the gate read and decoded a verdict document, and
///   carries the SHA-256 it computed over those bytes. Both sides of a
///   cross-format pair must carry the *same* digest here, so "two records
///   referring to the same evidence" is a fact about bytes rather than a
///   coincidence of two documents that agree.
/// * ``unread(path:)`` — a verdict artifact was named and could not be read or
///   decoded. **Never** read as ``noneDeclared``: a failed read is not an
///   absence, and reading it as one would turn an unreadable file into a
///   same-format pass over two different models. It derives the model-of-record
///   `unread`, which is refused by name.
/// * ``untrusted(path:digest:)`` — a verdict artifact was named, read and
///   decoded perfectly, and is **not a decision this repository took**. Its
///   SHA-256 appears in no entry of ``TrustedEquivalenceDecisions/shipped``,
///   which means the invocation authored it. Kept apart from all three of the
///   others because it is the F1 case and every collapse of it is the F1
///   defect: as ``read(_:digest:)`` it is the self-minted verdict review
///   accepted with exit 0, as ``noneDeclared`` it is a caller silently
///   downgraded to a same-format pin, and as ``unread(path:)`` it would blame
///   the file for being unreadable when it read fine and simply is not
///   evidence. It derives the model-of-record `untrusted` and is refused by
///   name, before any launch.
public enum ModelEquivalenceReading: Sendable, Equatable, Codable {
    case noneDeclared
    case read(ModelEquivalence, digest: String)
    case unread(path: String)
    case untrusted(path: String, digest: String)

    private enum CodingKeys: String, CodingKey {
        case state
        case equivalence
        case digest
        case path
    }

    private enum State: String, Codable {
        case noneDeclared
        case read
        case unread
        case untrusted
    }

    /// Decoded by name, with no default for a missing `state`.
    ///
    /// A document that omits the field is malformed rather than "no evidence
    /// was declared" — the same distinction the cases themselves carry.
    public init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        switch try container.decode(State.self, forKey: .state) {
        case .noneDeclared:
            self = .noneDeclared
        case .read:
            self = .read(
                try container.decode(ModelEquivalence.self, forKey: .equivalence),
                digest: try container.decode(String.self, forKey: .digest))
        case .unread:
            self = .unread(path: try container.decode(String.self, forKey: .path))
        case .untrusted:
            self = .untrusted(
                path: try container.decode(String.self, forKey: .path),
                digest: try container.decode(String.self, forKey: .digest))
        }
    }

    public func encode(to encoder: Encoder) throws {
        var container = encoder.container(keyedBy: CodingKeys.self)
        switch self {
        case .noneDeclared:
            try container.encode(State.noneDeclared, forKey: .state)
        case .read(let equivalence, let digest):
            try container.encode(State.read, forKey: .state)
            try container.encode(equivalence, forKey: .equivalence)
            try container.encode(digest, forKey: .digest)
        case .unread(let path):
            try container.encode(State.unread, forKey: .state)
            try container.encode(path, forKey: .path)
        case .untrusted(let path, let digest):
            try container.encode(State.untrusted, forKey: .state)
            try container.encode(path, forKey: .path)
            try container.encode(digest, forKey: .digest)
        }
    }

    /// The verdict, when one was read. `nil` for both the absence and the
    /// failure, which is why every caller of this also has to handle the two
    /// separately rather than branching on `nil`.
    public var equivalence: ModelEquivalence? {
        if case .read(let equivalence, _) = self { return equivalence }
        return nil
    }

    /// The digest the gate computed over the verdict document, when one was
    /// read. Used only to name the two sides in a refusal; the comparison that
    /// matters is made on the ``read(_:digest:)`` payload directly, so this
    /// returning `nil` can never be mistaken for two digests agreeing.
    public var evidenceDigest: String? {
        if case .read(_, let digest) = self { return digest }
        return nil
    }
}
