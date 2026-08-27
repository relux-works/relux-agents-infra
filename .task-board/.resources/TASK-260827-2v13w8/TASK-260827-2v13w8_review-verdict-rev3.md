# TASK-260827-2v13w8 review verdict — CR revision 3

Verdict: **CHANGES REQUESTED**. Route `TASK-260827-2v13w8` to `to-dev`.

Reviewed candidate: `CR-TASK-260827-2v13w8-3`, revision 3, candidate tree
`11e765482a7513a1d8e7ef8c78f9313c065cb23c`, binary SHA-256
`ba1c7451a7d035ecfd786cfd93510dc2b28da915a30aba02ea755ece21112695`.
The worktree matched the candidate tree before and after review.

## Critical finding R3-A — forged/self-minted evidence remains admissible

The attestation redesign raises the cost of the revision-2 forgery, but does not
close it. The remaining forgery is cheap and uses the shipped production entry
points exactly as documented; it does not edit, forge, or tamper with an
attestation file.

Independent reproduction against the final `ba1c7451...` binary:

1. `scripts/benchmark-gate-smoke.sh` started two placeholder HTTP processes,
   one under `/usr/bin/python3` and one under `/opt/homebrew/bin/python3`. They
   only answered `GET /v1/models`; neither loaded the model, ran a prompt, or
   measured a benchmark (`benchmark-gate-smoke.sh:72-99`, `:102-142`).
2. The real `benchmark-attest open|close` commands wrote both attestations.
   `open` accepts caller-supplied runtime/profile/config beside a live pid
   (`BenchmarkAttestCommand.swift:81-131`). `close` accepts a caller-supplied
   endpoint and model id and does not bind that endpoint or any benchmark
   traffic to the observed pid (`BenchmarkAttestCommand.swift:134-203`).
3. The same shipped smoke then created both records from those attestations and
   invented every measurement: 512 prompt tokens, 1.0 s TTFT, 100 tok/s
   prefill, 10 tok/s decode, 29,000,000,000 B peak
   (`benchmark-gate-smoke.sh:156-239`).
4. Production `benchmark-compare` accepted the pair with `accepted=true`, no
   blockers, unit ratios, and exit 0. The full attack completed in 7.2 s.

This is the standard negative shape **forged or self-minted evidence**. The
gate proves that two processes stayed alive and that some endpoints named the
model. It does not prove that the processes were the declared runtimes, that
the declared launcher/argv ran, that any scenario request happened, or that any
reported measurement was observed. The report's statement that a caller would
need to “fabricate the attestation too” understates the bypass: the final gate
binary willingly mints both attestations for the caller's placeholder
processes. The current smoke's “THE CONTROL IS NOT FABRICATED” claim is false
for the evidence being scored; `write_record` fabricates it explicitly.

The candidate therefore has no negative production-entry witness for the
actual R2-A class. Its no-attestation and field-mismatch negatives prove that
attestations are required and cross-checked, not that an admitted attestation
corresponds to a benchmark.

## Required rework

- Put runtime launch, scenario driving/measurement, record construction, and
  judgement under one trusted production invocation, or provide an equivalent
  independently protected observer that the ordinary caller cannot direct to
  attest placeholder processes. The report already identifies the single-
  process construction as the only residue-free boundary; that is the clean
  recommendation.
- Bind the measured transcript to the observed runtime and the decision. A
  `/v1/models` response and elapsed window are not a benchmark transcript.
- Turn the reproduction above into a negative production-entry test: two live
  `/v1/models` placeholders plus caller-authored measurements must be
  inadmissible (exit 4). Prove the bound with a narrowing mutant that replaces
  the real scenario driver/measurement path with this placeholder path while
  preserving valid attestations.
- Because the final judging binary will change, preserve R2-C by rerunning the
  final evidence on the binary that serves/observes/judges.
- Add this review finding to `LOGBOOK.md` during producer rework. The reviewer
  did not modify the candidate tree.

## Findings closed or accepted in revision 3

- **R2-B/R2-C:** closed. The attached executed config pins
  `reasoning=medium` and `prefill-step=2048` in both profiles. Both records
  report 41 / 7,784 / 73,016 prompt tokens. Config digest
  `dfa80a60...` matches both records and attestations. The current candidate
  binary digest is exactly `ba1c7451...`; it served the Swift pass, observed
  both passes, and independently re-judged the attached real pair as
  `accepted=false`, exit 3.
- **8k reasoning:** sound. The old mismatch was a constant 38 tokens, only
  0.49% of the corrected 7,784-token 8k prompt. Equal-policy rerun ratios
  (1.210 TTFT, 0.826 prefill, 1.144 footprint) remain materially outside their
  1.10/0.90/1.10 bars and moved by less than two percentage points.
- **Short prompt:** the equal-token point estimates are marginally favorable to
  Swift (0.974 TTFT, 1.026 prefill, 1.000 decode, 0.990 footprint). With one
  observation and no variance they are not a robust performance win; the
  report appropriately calls them marginal and does not base the REJECT on
  them.
- **75k capacity:** recording rather than scoring the scenario is defensible
  for this conservative REJECT because the task's capacity criterion requires
  executed outcome/capability evidence and the report exposes the 1.485x TTFT
  gap rather than hiding it. It must remain an explicit migration risk; a later
  acceptance should make an explicit policy decision before treating capacity
  success alone as sufficient.
- **R2-D:** recorded at the contract sites. `GenerationWorkerHealth.swift:35-44`
  and `GenerationBatchRecovery.swift:87-95` explicitly limit in-process 503,
  batch release, and teardown evidence to errors delivered as throws; an
  `asyncEval` trap recovers only by process death and supervisor replacement.
- **Migration decision:** still conservatively correct. Python `mlx-lm` remains
  default; the real final pair produces the three reported 8k blockers.

## Independent validation

- `swift test`: exit 0, 271 tests in 23 Swift Testing suites.
- `scripts/benchmark-gate-smoke.sh` on final binary: exit 0; its accepted
  control independently reproduced R3-A (`accepted=true` on invented metrics).
  The first invocation failed before any check because the reviewer supplied an
  incorrect doubled binary path; the corrected invocation above is the result.
- `benchmark-compare` on the attached real revision-3 records/attestations:
  exit 3, `accepted=false`, exactly three 8k blockers.
- `xcrun swift-format lint --strict --recursive Sources Tests`: exit 0.
- `shellcheck -S warning scripts/*.sh`: exit 0.
- `python3 -m py_compile scripts/runtime-benchmark.py`: exit 0.
- Exact CR diff check and candidate/worktree equality: exit 0.

Raw independent attack evidence is attached separately: the two gate-written
attestations, the two fabricated records, and the accepted decision.
