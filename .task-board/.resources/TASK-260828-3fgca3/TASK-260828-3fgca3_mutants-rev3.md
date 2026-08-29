# TASK-260828-3fgca3 — revision 3 mutants

Twelve mutants, each applied to the shipped source, built, run, reverted.
All twelve killed, zero survivors. The three below restore exactly the reading
review exploited and reproduce its exit 0 from the shipped binary.

| Mutant | What it does | swift test | smoke |
| --- | --- | --- | --- |
| B3+A1 | F1 undone at the production entry: trust lookup deleted from equivalenceReading and neutered in admitModelIdentity | exit 1, 2 red | exit 1, 5 FAIL — the caller's own verdict is accepted, exit 0 |
| B1+B1b | F2 undone: a malformed or non-positive n_ctx is an absence again | exit 1, 23 red | exit 1, 3 FAIL — the 32768 window pinned unbounded and was accepted, exit 0 |
| B2 | F3 undone: every non-200 from /slots is an absence again | exit 1, 20 red | exit 1, 3 FAIL — the HTTP 500 pair was scored as MTP-off, exit 0 |
| A1 | the trust guard in admitModelIdentity alone stops refusing | exit 1, 2 red | — |
| A2 (M11, narrowing) | byte identity restored on top of the evidence: modelDigest back in firstMismatch | exit 1, 18 red | — |
| A3 (M12, narrowing) | any runtime that answers /slots is treated as speculating | exit 1, 16 red | — |
| A4 (drift) | the shipped trust store requires a note its own document does not carry | exit 1, 4 red | — |
| A5 | the declared non-equivalences stop being demanded of the records | exit 1, 4 red | — |
| A6 | a context flag the gate cannot read goes back to being an absence | exit 1, 15 red | — |
| A7 (narrowing) | no status means route-absent, so every MLX runtime reads unread | exit 1, 3 red | — |
| A8 (narrowing) | rev 2's M3 re-run here: kv=unbounded declared unpinnable, so an MLX-against-MLX pair is refused | exit 1, 73 red | — |
| A9 | rev 2's M4 re-run here -- THE RELAXATION THIS TASK WAS TOLD NOT TO MAKE: prefill-step=unpinned dropped from unpinnableConditions | exit 1, 3 red | — |

## Raw smoke output of the three production-entry mutants

### B3+A1 — F1 undone
```
FAIL  a caller-authored verdict is refused however well-shaped it is: expected exit 5, got 0
FAIL  the minted-verdict pair was still scored
FAIL  a caller-authored equivalence-not-comparable is refused as untrusted, not on its contents: expected exit 5, got 4
FAIL  a caller-authored equivalence-other-artifact is refused as untrusted, not on its contents: exit 5 but the output never mentions is not an equivalence decision this repository took
FAIL  a caller-authored equivalence-no-notes is refused as untrusted, not on its contents: expected exit 5, got 4
----------------------------------------
BENCHMARK GATE SMOKE FAILED (5 failures)
```

### B1+B1b — F2 undone
```
FAIL  a malformed n_ctx cannot buy an unbounded pin: expected exit 4, got 0
FAIL  the malformed-window pair was still scored
FAIL  the malformed window was recorded as something other than unread
----------------------------------------
BENCHMARK GATE SMOKE FAILED (3 failures)
```

### B2 — F3 undone
```
FAIL  a failed /slots observation cannot be spent as MTP-off: expected exit 4, got 0
FAIL  the failed-observation pair was still scored
FAIL  the failed /slots observation was recorded as something other than unread
----------------------------------------
BENCHMARK GATE SMOKE FAILED (3 failures)
```

## The mutant definitions
```python
import sys, pathlib
name = sys.argv[1]
RB = pathlib.Path("Sources/MLXSwiftRuntimeContract/RuntimeBenchmark.swift")
RA = pathlib.Path("Sources/MLXSwiftRuntimeContract/RuntimeAttestation.swift")
TE = pathlib.Path("Sources/MLXSwiftRuntimeContract/TrustedEquivalenceDecisions.swift")
BP = pathlib.Path("Sources/mlx-swift-runtime-prototype/BenchmarkRunPins.swift")

M = {
 # widening: the trust lookup stops deciding anything
 "A1": (RB, """        guard let anchor = TrustedEquivalenceDecisions.decision(
            documentDigest: baselineDigest, in: trust)
        else {
            throw AdmissionError.equivalenceEvidenceUntrusted(
                sourceOfRecord: baselineEvidence.sourceOfRecord, digest: baselineDigest)
        }""",
  """        let anchor = TrustedEquivalenceDecisions.decision(
            documentDigest: baselineDigest, in: trust)
            ?? TrustedEquivalenceDecision(
                sourceOfRecord: baselineEvidence.sourceOfRecord,
                documentDigest: baselineDigest, requiredNonEquivalences: [],
                provenance: "mutant")"""),
 # narrowing (M11): byte identity restored on top of the evidence
 "A2": (RB, """                ("modelOfRecord", modelOfRecord, other.modelOfRecord),""",
  """                ("modelOfRecord", modelOfRecord, other.modelOfRecord),
                ("modelDigest", modelDigest, other.modelDigest),"""),
 # narrowing (M12): any runtime that answers /slots is treated as speculating
 "A3": (RB, """        case .reported(let active): return active ? "on" : "off\"""",
  """        case .reported: return "on\""""),
 # drift: the shipped store requires a note its own document does not carry
 "A4": (TE, """                "the MLX build drops the MTP head the GGUF carries as blk.64: 8 quantized "
                    + "tensors, 451319808 bytes on disk, skipped at load",""",
  """                "the two builds differ in quantization scheme","""),
 # the declared non-equivalences stop being demanded of the records
 "A5": (RB, """            for entry in baselineEvidence.declaredNonEquivalences
                + anchor.requiredNonEquivalences
            where !record.declaredAsymmetries.contains(entry) {
                throw AdmissionError.declaredNonEquivalenceNotCarried(
                    runtime: record.runtime, entry: entry)
            }""", """            _ = record"""),
 # widening: a context flag the gate cannot read goes back to being an absence
 "A6": (RB, """                guard let parsed = Int(raw), parsed > 0 else {
                    return .unreadable(flag: flag, raw: raw)
                }""", """                guard let parsed = Int(raw), parsed > 0 else { continue }"""),
 # narrowing: no status means "this route is absent", so every MLX runtime is unread
 "A7": (RA, """    static let routeAbsentStatuses = [404, 501]""",
  """    static let routeAbsentStatuses: [Int] = []"""),
 # widening (F2 undone): a malformed n_ctx is an absence again
 "B1": (RA, """        guard let meta = rawMeta as? [String: Any] else { return .unread }
        guard let rawLength = meta["n_ctx"] else { return .notReported }""",
  """        guard let meta = rawMeta as? [String: Any] else { return .notReported }
        guard let rawLength = meta["n_ctx"] else { return .notReported }"""),
 "B1b": (RA, """        guard !(rawLength is Bool), let number = rawLength as? NSNumber,
            CFNumberIsFloatType(number) == false
        else { return .unread }
        let length = number.intValue
        guard length > 0 else { return .unread }""",
  """        guard let length = rawLength as? Int, length > 0 else { return .notReported }"""),
 # widening (F3 undone): every non-200 is an absence again
 "B2": (RA, """        guard status == 200 else {
            return routeAbsentStatuses.contains(status) ? .notReported : .unread
        }""",
  """        guard status != 0 else { return .unread }
        guard status == 200 else { return .notReported }"""),
 # widening (F1 undone at the production entry): the caller's document is trusted
 "B3": (BP, """        let documentDigest = digest(of: bytes)
        guard TrustedEquivalenceDecisions.decision(documentDigest: documentDigest) != nil else {
            return .untrusted(path: path, digest: documentDigest)
        }
        return .read(decoded, digest: documentDigest)""",
  """        return .read(decoded, digest: digest(of: bytes))"""),
}
path, old, new = M[name]
s = path.read_text()
assert old in s, f"{name}: anchor text not found"
path.write_text(s.replace(old, new, 1))
print(f"applied {name} to {path.name}")
```
