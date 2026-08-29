# TASK-260828-3fgca3 Review Verdict

- Verdict: **changes requested**
- Route: `to-dev`
- Change Request: `CR-TASK-260828-3fgca3-2`, revision 2
- Reviewed candidate tree: `d829b94ac01a69052642e23e65752a2500a03da2`
- Base: `5a287e4bcb454b53bc432cb7788a03792d1c96f6`
- Reviewer run: `RUN-260828-36e21d`
- Reviewed at: 2026-08-28 23:25 MSK

The candidate is not acceptable. Three production-entry attacks make the real
`benchmark-run` admit claims it must refuse. This is ordinary implementation
rework, not a stop-the-line boundary.

## Findings

### F1 — Cross-format equivalence evidence is caller-self-minted

Severity: blocking.

`BenchmarkRunCommand.equivalenceReading(path:)` reads and hashes any JSON path
supplied through `--equivalence`; it establishes no trusted producer, expected
verdict digest, or other provenance
(`Sources/mlx-swift-runtime-prototype/BenchmarkRunPins.swift:171-189`). The
admission clause proves only that both attestations contain the same bytes, the
document says `comparable`, the document names the two gate-computed artifact
digests, and its non-equivalence list is non-empty
(`Sources/MLXSwiftRuntimeContract/RuntimeBenchmark.swift:1267-1308`). A caller
can therefore mint every fact the gate later treats as evidence.

Production attack: a caller-created verdict named an arbitrary source of
record, the real artifact digests, `comparable`, and only one generic note.
The shipped entry accepted it with exit 0 and `accepted=true`. Both records and
the decision carried only that invented note. Thus the required dropped MTP
head, vision placement, and F32-versus-bf16 norms can all be omitted while the
comparison passes. Hashing and observer-sealing attacker-authored bytes does
not authenticate the verdict.

Evidence:

- `.temp/review-TASK-260828-3fgca3/production-attacks-record-proof-01.log`
- `.temp/review-TASK-260828-3fgca3/production-attacks-02.log`
- `.temp/review-TASK-260828-3fgca3/smoke-control/review2-forged-one-note.log`

Negative shape: **forged or self-minted evidence**.

Required rework: bind cross-format admission to a trusted, pre-existing
equivalence decision that an invocation cannot author for itself. For example,
make the expected verdict identity/digest part of trusted configuration or a
task-owned immutable artifact and compare the observed bytes against it. Add a
production-entry negative test where the caller creates a well-shaped verdict
covering the real files; it must refuse. The trusted decision must also bind the
three required non-equivalences, and both records plus the observer seal must
carry those exact decision contents.

### F2 — A malformed runtime KV answer becomes `notReported` and then `unbounded`

Severity: blocking.

After a successful `/v1/models` response, `servingAnswer` maps every missing,
wrongly typed, or non-positive `meta.n_ctx` to `.notReported`
(`Sources/mlx-swift-runtime-prototype/BenchmarkRunCommand.swift:718-734`).
`contextPolicy` then spends `.notReported` as permission to derive
`--max-kv-size` or `unbounded`
(`Sources/MLXSwiftRuntimeContract/RuntimeBenchmark.swift:898-906`). This
collapses a malformed/partial read into a legitimate absence.

Production attack: the candidate runtime had a finite 32768-token window but
returned `meta.n_ctx` as the malformed string `"32768"`. The attestation said
`notReported`, the record asserted `kv=unbounded`, and the real entry accepted
the pair with exit 0 and `accepted=true` against an unbounded baseline.

Evidence:

- `.temp/review-TASK-260828-3fgca3/production-attacks-record-proof-01.log`
- `.temp/review-TASK-260828-3fgca3/production-attacks-02.log`
- `.temp/review-TASK-260828-3fgca3/smoke-control/review2-malformed-nctx.log`

Negative shape: **failed or malformed read presented as absence**.

Required rework: distinguish an absent runtime field from a present but
malformed/non-positive field. The latter must be `.unread` and inadmissible.
Drive that distinction through the production entry with a finite candidate;
the pair must not acquire an unbounded pin or reach scoring.

### F3 — A `/slots` server failure becomes “MTP off”

Severity: blocking.

`speculationAnswer` maps every nonzero HTTP status other than 200 to
`.notReported`, including 5xx and authorization failures
(`Sources/mlx-swift-runtime-prototype/BenchmarkRunCommand.swift:761-777`).
The argv fallback then derives speculation `off`, so a failed observation can
satisfy the MTP-off admission rule.

Production attack: the candidate fixture was configured as speculative while
`GET /slots` returned HTTP 500. Its attestation said `notReported`, its record
pinned `speculation=off`, and the real entry accepted the scored comparison
with exit 0 and `accepted=true`. The gate therefore did not establish that MTP
was off.

Evidence:

- `.temp/review-TASK-260828-3fgca3/production-attacks-record-proof-01.log`
- `.temp/review-TASK-260828-3fgca3/production-attacks-02.log`
- `.temp/review-TASK-260828-3fgca3/smoke-control/review2-slots-http-500.log`

Negative shape: **failed read presented as absence**.

Required rework: reserve `.notReported` for explicitly supported route-absence
responses; network failure, 5xx, unexpected status, malformed 200, and other
failed observations must be `.unread` and refuse scoring. Add a production
negative test for HTTP 500 and prove it cannot derive `off`.

### F4 — The changed smoke script adds lint findings

Severity: non-blocking by itself, required by the task DoD.

`swift-format lint --strict --recursive Sources Tests` is clean. `shellcheck
scripts/benchmark-gate-smoke.sh` exits 1: the base already had 12 SC2015/SC2181
findings, and revision 2 adds four at candidate lines 892, 929, 930, and 1110.
Avoid adding new lint debt while repairing the blocking gate findings.

Evidence:

- `.temp/review-TASK-260828-3fgca3/swift-format-lint-01.log`
- `.temp/review-TASK-260828-3fgca3/shellcheck-01.log`
- `.temp/review-TASK-260828-3fgca3/shellcheck-base-01.log`

## Evidence that passed

- Candidate index tree exactly matched revision 2:
  `d829b94ac01a69052642e23e65752a2500a03da2`; `git diff --cached --check`
  passed. Evidence: `final-tree-hygiene-01.log`.
- Clean candidate test run: 324 tests in 27 suites passed. Evidence:
  `swift-test-release-02.log`.
- Candidate production smoke: 59 checks, 0 failures. Evidence:
  `benchmark-gate-smoke-control-01.log`. Its positive paths do not cover F1-F3.
- M1 (old argv-only KV derivation) failed 36 assertions. Its private mutated
  production binary nevertheless reproduced the historic false match: the
  candidate attestation reported 32768, both records asserted `kv=unbounded`,
  and the entry returned exit 0 with `accepted=true`. Evidence:
  `mutant-M1-swift-test-01.log`, `mutant-M1-production-02.log`.
- M11 (restored raw model-digest equality) failed 12 assertions. Evidence:
  `mutant-M11-swift-test-01.log`.
- M12 (narrowed speculation admission to always-on) failed 11 assertions.
  Evidence: `mutant-M12-swift-test-01.log`.
- G1 recognises `--prefill-step-size`, `--ubatch-size`, and `-ub` additively.
- `unpinnableConditions` remains unchanged; the candidate tests and M1/M11/M12
  mutants exercise the intended boundaries.
- No 28 GB model was loaded during review. The production attacks used only the
  tiny fake runtime/model fixture on ports 28901, 28911, 28921, and 28931.
  Final process and listener checks were empty. Evidence:
  `final-validation-summary-01.log`.

## Review integrity

All mutations were made only in a private ignored copy under
`.temp/review-TASK-260828-3fgca3/mutant-package`. Its source was restored to the
pristine SHA-256
`b9f8b5cfed633fdd8da6f1fefc1e67ff370c6e3a873c5e9c9e451b2e9466a9e1`.
The reviewed repository index remained exactly the candidate tree. The
versioned `LOGBOOK.md` was not modified by the reviewer because it is itself
inside the CR snapshot; the persistent board outcome is the reviewer-owned
finding record.
