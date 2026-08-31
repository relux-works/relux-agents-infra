# TASK-260829-3cwcb6 Review Verdict — Revision 3

Verdict: **accepted**.

Reviewed immutable delta: `4332e1dddd0164876b4da3ec0340ba9320aec1e9`
to candidate tree `666272309815b07718c5bdc494896e12001e7f31` for
`CR-TASK-260829-3cwcb6-3`, revision 3. The regenerated binary patch SHA-256 is
`420d4b73c65bb35f59973db899426bec81db7625bb8b141f3f02af2ba611d922`,
matching the Change Request resource.

## F1 — production decision is bound to the composite

I changed only the final measured return in
`RuntimeMemoryPeak.validatedScoredBytes` from the validated composite to
`peakSample.machPhysicalFootprintBytes`. The record shape, accounting name,
raw components, stored composite, and `scoredBytes` were unchanged.

The Release production binary built successfully, then
`scripts/benchmark-gate-smoke.sh` exited 1 with exactly one failure. The
production call path was `benchmark-run` -> `RuntimeBenchmark.decide` ->
`RuntimeMemoryPeak.validatedScoredBytes`; the failure was the new
decision-to-record equality assertion.

For both the `short_prompt` and `process` decision deltas in my run:

| Candidate value | Bytes |
| --- | ---: |
| Decision value under the mutant | 13,763,016 |
| Mach component | 13,763,016 |
| Resident mapped-file upper component | 11,114,906 |
| Re-derived/stored/scored record composite | 24,877,922 |

The generated record remained internally correct while only the decision value
narrowed, so this is a real narrowing kill bound to the consumed delta rather
than to a record field or metric-name check. The pristine smoke on the same
candidate tree exited 0 with `BENCHMARK GATE SMOKE OK (0 failures)`.

## F1b — forged decoded components fail closed

The permanent negative encoded a measured peak with Mach `100`, mapped `2,048`,
and composite `2,148`, rewrote both stored composite and `scoredBytes` to the
old Mach-only value `100`, decoded it, and drove `RuntimeBenchmark.decide`.
The focused test passed: `validatedScoredBytes` exposed no value, the decision
was not accepted, and the memory axis carried the named unmeasured blocker.

The implementation re-derives Mach plus mapped bytes after decode with sign and
overflow checks. Both production consumers of the metric — scenario-local and
whole-process deltas — call `validatedScoredBytes`; no raw
`peakPhysicalFootprintBytes` call site remains in scoring.

## Producer evidence discipline

The producer's claimed separation is accurate. Its pristine flake log has the
same 14 attestation-close cascade failures as its mutant log. The mutant log
adds one distinct failure at the memory decision-to-record assertion and ends
with 15 failures. The producer did not count the shared 14 toward the kill.
My independent mutant run had no cascade and ended with only the memory mismatch,
which independently confirms the attribution.

## Revision-2 properties retained

- `reasoning` and `reasoning_content` flow through one generated-event reader;
  TTFT starts on the first such event and decode ends on the last. The pristine
  production smoke recorded four completion tokens and non-null TTFT/decode for
  both spellings.
- Both runtime shapes resolve to the same
  `mach-physical-footprint-plus-vmmap-resident-mapped-file-upper-bound`
  accounting across warm-up, scenario, soak, and process windows.
- Records retain Mach, literal vmmap token, mapped-file upper component,
  composite, status, counts, issues, and `conservative-upper-bound` semantics.
- Absent, read-failed, malformed, and partial windows expose no score; decoded
  component inconsistency now does the same.
- Machine-readable records and `decision.json` carry MTP-off as against
  llama.cpp. Remaining limitations state their directions: fixed order is
  indeterminate, parity policy favours the incumbent, and conservative-memory
  residual bias is runtime-dependent.
- The task's explicit audit for defects biased against llama.cpp remains in the
  task outcome and repository logbook; no additional silent adverse default or
  permissive measurement path was found in revision 3.

## Reviewer validation

| Check | Result |
| --- | --- |
| Exact CR patch hash and `git diff --check` | pass |
| Focused decoded Mach-only forgery negative | 1 test / 1 suite passed |
| Full `swift test -c release --skip-build` | 401 tests / 32 suites passed |
| Pristine Release production smoke | exit 0, 0 failures |
| Mach-only mutant Release build | exit 0 |
| Mach-only mutant production smoke | exit 1, exactly 1 memory mismatch |
| `swift-format lint --strict --recursive Sources Tests` | pass |
| `shellcheck -S warning scripts/benchmark-gate-smoke.sh` | pass |
| Mutant source restoration hash | restored to `3a621858...a4e` |

The producer's attached macOS arm64 Xcode Release build also reports
`BUILD SUCCEEDED`. I accepted that already-attached build evidence rather than
rerunning the same broad build; all negative gates and the complete Swift suite
above were rerun independently against the immutable candidate tree.

No repository code was modified by review. The task is accepted for the
orchestrator's checkpoint/integration step.
