# TASK-260829-3k4qrc revision 3 review verdict

Verdict: **changes requested**. Route: `to-dev`.

Reviewed Change Request `CR-TASK-260829-3k4qrc-3` revision 3, base
`3272e3ae60b53eb0fb61f6d427a883487711b48f`, candidate tree
`a193e60e29fb1046ebb7dfe73a81c91594928b9a`. The attached patch reproduces
SHA-256 `d12688156d7aac13536d3d64c427eec0482d71abbbcd8ab192ebc25a54cffe95`, and
the review worktree writes exactly tree `a193e60e`, so every finding below is
against the exact reviewed revision.

Answer to the brief's closing question: **no.** Do not spend the hour on this
revision. Under revision 3 the run would return no memory number at all.

## F3 — the scored resident-memory dimension has an empty admissible set

Severity: blocking. This is the revision-2 F1 finding closed in the wrong
direction: the stale value is no longer minted as fresh, but nothing can be
measured in its place.

`samplingCoverageIssue` now requires the mapped-file timestamp series to have
all consecutive distinct gaps `<= maximumMappedFileSampleGapSeconds = 0.125`
(`RuntimeMemoryAccounting.swift:44-108`). The mapped-file component is produced
only by `residentMemorySample`, which forks `/usr/bin/vmmap -summary`. In
production it runs at window boundaries and on the 5 s `fullLoop`
(`BenchmarkFootprintSampler.swift:131-163, 239-292`); `physicalLoop` reuses the
last mapped value and, correctly, its original timestamp.

One `vmmap -summary` costs **0.668–0.687 s** against a ~3 MB process on this
host (8 consecutive production `residentMemorySample` calls). Two mapped
observations therefore cannot be closer than ~0.68 s, which is **5.4x the
0.125 s the gate admits**. There is no window length that satisfies the bound.

Reviewer production-source attack: the exact candidate `RuntimeMemoryAccounting`
and `BenchmarkFootprintSampler` sources were compiled unmodified in task-scoped
scratch and the production class was driven directly.

```
window=0.05s status=partial scored=nil issues=["resident-mapped-file-sampling-gap"] samples=15
window=0.10s status=partial scored=nil issues=["resident-mapped-file-sampling-gap"] samples=16
window=0.12s status=partial scored=nil issues=["resident-mapped-file-sampling-gap"] samples=16
window=0.20s status=partial scored=nil issues=["resident-mapped-file-sampling-gap"] samples=18
window=0.50s status=partial scored=nil issues=["resident-mapped-file-sampling-gap"] samples=23
window=1.00s status=partial scored=nil issues=["resident-mapped-file-sampling-gap"] samples=31
window=3.00s status=partial scored=nil issues=["resident-mapped-file-sampling-gap"] samples=68
window=7.00s status=partial scored=nil issues=["resident-mapped-file-sampling-gap"] samples=136

zero-length window (beginWindow immediately followed by capturePeaks):
  status=partial issues=["resident-mapped-file-sampling-gap"] distinctMapped=2 span=0.688s
```

Both shipped production entries agree. `benchmark-mapped-file-sampler-probe`
exits 0 by taking its refusal branch, not its measurement branch:

```
directMappedFileBytesUpperBound=268645172
peak.status=partial  issues=["resident-mapped-file-sampling-gap"]
peak.peakSample.residentMappedFileBytesUpperBound=82944
distinct mapped timestamps=2, separated by 2.126s
```

`benchmark-memory-sampler-probe` is now written to *require* the refusal
(`BenchmarkMemorySamplerProbe.swift:32-34`: `status == .partial` and
`issues.contains("resident-mapped-file-sampling-gap")`).

Consequence at the decision entry. `appendDelta` is fed
`peakResidentMemory.validatedScoredBytes`, which is `nil` for a partial peak
(`RuntimeBenchmark.swift:2233-2262`). Every per-scenario
`peak_resident_memory_upper_bound_bytes` delta and the process-wide delta
therefore become `unmeasured` blockers, and `accepted = blockers.isEmpty` can
never be true. The hour-scale pair would return `context_75k` memory as
`unmeasured` for both runtimes and exit 3.

This is not "the dimension was refused because it is not comparable" — the
dimension is refused because the instrumentation cannot observe it at the
resolution it claims. The DoD line "corrected memory reproduces or overturns
the provisional 9 percent advantage at 75000 tokens" is unreachable as shipped.

Counterfactual, same production sources, single constant changed to size the
mapped bound to the cadence the mapped component is actually read at:

```
maximumMappedFileSampleGapSeconds = 6.0
window=0.5s  status=measured scored=2097536 issues=[]
window=1.0s  status=measured scored=2360728 issues=[]
window=3.0s  status=measured scored=2606488 issues=[]
window=7.0s  status=measured scored=2934168 issues=[]
window=12.0s status=measured scored=3147184 issues=[]
```

The incoherence is a 125 ms coverage claim asserted over a component obtained
at 0.2 Hz by a reader that costs 0.68 s per call.

Required rework, either direction, not both:

- Size the mapped-file coverage bound to the cadence at which the mapped
  component is actually observed, and say plainly in the contract and in the
  record that mapped-file transients shorter than that cadence are not
  observable. The Mach component keeps its independent 20 Hz claim and its own
  125 ms bound; that separation is the part of revision 3 worth keeping.
- Or obtain the mapped-file component from a reader cheap enough to support a
  125 ms claim, and keep the bound.

Whichever is chosen, add a production-entry positive that fails when the
scored memory dimension cannot be measured for a realistic scenario window.
Revision 3 has negatives for the refusal and none for the score, which is why
a total loss of the dimension read as green.

## F4 — the smoke can no longer tell "refused correctly" from "never scores"

Severity: blocking, same root cause, separate defect in the evidence.

The control accepts exit 0 or exit 3 and only requires that every blocker
mention `peak_resident_memory_upper_bound_bytes was not measured`
(`benchmark-gate-smoke.sh:663-691`). Scenario and process peaks are asserted
with `assert_peak_or_coverage_refusal` (`:2274-2299`). The tolerance itself
predates this revision, but revision 3 is where it started absorbing a total
loss of the dimension.

In the producer's own attached green run, **every** admitted-path check is
exit 3:

```
rev3 green: PASS a measured pair reaches an admitted decision (exit 3)
            PASS the thresholds a caller names reach the production entry (exit 3)
            PASS a bound reported by the process and a bound pinned in argv compare equal (exit 3)
            PASS a tool that deliberately requires no arguments is measured and admitted (exit 3)
            PASS mmap-loaded and anonymous runtime shapes carry the same scored quantity (exit 3)
            ... 118 PASS / 0 FAIL

rev2 green: PASS a measured pair reaches an admitted decision (exit 0)
            ... 115 PASS / 0 FAIL
```

No full pass anywhere in revision 3 reaches `accepted=true`. The comment at
`:498-502` still says the wide-band thresholds exist so "an honestly measured
pair reach `accepted=true` so the acceptance path is exercised at all"; that
is no longer true of any check in the file. A green suite in which the
production acceptance path is never taken cannot report that the acceptance
path broke.

Required: at least one smoke check must require `accepted=true` from a full
measured pass with scored memory on both records, and fail when memory is
refused. Keep the coverage-refusal branch as its own separate check with its
own fixture.

## Suite growth: real, but the growth is not where the loss is

52 -> 118 executed checks is real, and no assertion was dropped between
revision 2 and revision 3. Comparing `pass` statements: base 45, revision 2 48,
revision 3 51. Revision 3 renames one (`production sampler catches a 150 ms
transient...` -> `anonymous probe catches the Mach transient while mapped
coverage refuses separately`) and adds three, all genuine:
`one-sided cache reuse reaches a rejected decision (exit 3)`,
`production decision consumes the sealed cache fact and keeps the no-hit
control scoreable`, `file-backed transient is captured or explicitly refused
for mapped-file coverage`. Nothing previously failing was removed rather than
fixed. The three acceptance-path checks weakened relative to `base` were
weakened in revisions 1-2, not here.

## F1 (revision 2) — closed as stated, wrong direction

The revision-2 attack reproduces and no longer scores. Revision 2 reported
`directMapped=268645172 scoredMapped=82944 missed=268562228 status=measured
issues=[]`. Revision 3 reports the same two quantities with `status=partial`,
`scoredBytes` absent, `issues=[resident-mapped-file-sampling-gap]`. Independent
Mach and mapped timestamps are carried correctly, reuse preserves the original
mapped timestamp (`BenchmarkFootprintSampler.swift:152-158, 209-221, 239-252,
276-292`), and `RuntimeMemoryComponents` no longer lets one timestamp stand in
for the other. That mechanism is right. Only its calibration is not.

## F2 (revision 2) — closed, and load-bearing

The rule now executes at the decision entry. `RuntimeBenchmark.decide` computes
`cacheComparabilityIssue` and ANDs it into scenario comparability before any
metric is scored (`RuntimeBenchmark.swift:2176-2190, 2286-2320`). The fact is
real telemetry, not a declaration: `BenchmarkHTTPDriver` reads
`usage.prompt_tokens_details.cached_tokens` from both the streaming usage frame
and the non-streaming body, distinguishing `reported` / `notReported` /
`malformed` (`BenchmarkHTTPDriver.swift:222-238`), and every scenario seals it
(`BenchmarkScenarios.swift`, all failure and success paths). `validatedState`
rejects a decoded observation that disagrees with its own facts.

Attacked, not read. A **narrowing** mutant on the exact candidate sources — the
one-sided `hit`/`miss` branch removed while the unknown, malformed and
applicability branches were left intact — fails 3 tests in
`runtime benchmark comparison gate` (`swift test -c release --filter
RuntimeBenchmarkTests`, 76 tests). The smoke's F2 check
(`benchmark-gate-smoke.sh:695-729`) drives the real `benchmark-run`, asserts
equal `promptTokens`, baseline `miss`, candidate `hit`, exit 3, an explicit
`short_prompt/cache_reuse is one-sided` blocker and `non-comparable` verdicts on
TTFT, prefill and decode; the symmetric control asserts no `cache_reuse` blocker
and a scoreable TTFT. That is the shape revision 2 lacked.

## Unknown, and worth ten minutes before the hour: F2's refusal breadth

I could not establish that both real runtimes report `cached_tokens` on the real
scenarios, and neither did revision 3 — the F2 evidence is a fake runtime that
emits the field on demand. If either side omits it, `cacheReuseObservation`
returns `unknown`, and `cacheComparabilityIssue` then makes **every** scenario
non-comparable. That is the same failure shape as F3, applied to every
dimension at once.

What is established:

- Baseline reports it. The pinned fork
  `/Users/alexis/src/relux-works/mlx-lm/mlx_lm/server.py` populates
  `usage.prompt_tokens_details.cached_tokens` on both the non-streaming path
  (`:1325-1333`) and the streaming usage frame (`:1573-1582`), and
  `prompt_cache_count` is assigned unconditionally on both generation paths
  (`:722`, `:945`), so it is never left at its `-1` sentinel. A real signed
  baseline completion in `TASK-260830-2hc5r2` carries
  `"prompt_tokens_details": {"cached_tokens": 0}`.
- Candidate: unverified. `libllama-server-impl.dylib` (llama.cpp 0.3.0) contains
  `prompt_tokens_details` and `cached_tokens` adjacent to `prompt_tokens`, but
  whether they appear in the **streaming** usage frame this driver reads is not
  established. The revision-1 records store transcript digests only, so the
  archived evidence cannot answer it.

Report this as unknown rather than assuming it. Confirm it with one short
streamed request against `llama-server` before committing to the pair.

## Verified closures and validation

- Bounded-KV closures hold. In the attached green run: `--python-bin` decoy
  exits 5 and cannot mint an accepted record, its owned process group is reaped,
  missing live KV stays inadmissible at exit 4 despite a finite launch flag,
  malformed `n_ctx` cannot buy an unbounded pin and produces no decision, and an
  unreadable verdict is refused rather than spent as an absence. No bounded-KV
  merge regression was found.
- `swift test -c release`: **407 tests / 32 suites, exit 0**, reproduced by the
  reviewer. Matches the claim.
- `swift build -c release`: exit 0.
- Patch SHA-256 and candidate tree OID both reproduce exactly.
- Producer's smoke counts confirmed from the attached transcripts: revision 3
  green 118 PASS / 0 FAIL; revision 2 115 PASS / 0 FAIL. The preserved
  revision-3 red run is real and its cause is documented.

## Definition of Done status

- Both runtimes sequential on a clean host — not applicable this revision (rerun
  deliberately not performed, per brief).
- MTP off for scored comparisons — unchanged, not regressed.
- Non-comparable dimensions refused and reported — **fails**: memory is refused
  for a reason that is not non-comparability, and the refusal is unconditional.
- Corrected decode / TTFT reproduce or overturn the 10 percent deficit — pending
  the rerun.
- Corrected memory reproduces or overturns the 9 percent advantage at 75k —
  **unreachable as shipped** (F3).
- Negative tests with production call site named — met for F2, met for the F1
  refusal, **missing** for the F3 positive direction.
- Lint / build / tests green — met.
- Logbook and artifacts — met.

No repository file was modified by the reviewer; the review worktree still
writes tree `a193e60e`. Scratch attacks, the vendored-source window scan, the
narrowing mutant and both probe outputs are under
`.temp/TASK-260829-3k4qrc/reviewer-r3/`. `accept_cr` was not called.
