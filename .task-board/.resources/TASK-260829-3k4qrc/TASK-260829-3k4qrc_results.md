# TASK-260829-3k4qrc corrected llama.cpp comparison

Date: 2026-08-30 MSK

## Revision 2: corrected instrumentation and production-gate closure

Revision 2 fixes the remaining mmap-aware smoke contract and publishes the
reviewable instrumentation change. Per the third-run brief, it **does not**
repeat the hour-scale six-scenario model pair. The sequential, same-host,
MTP-off real-runtime measurements and every refused dimension remain preserved
below as revision-1 evidence; no with-MTP result exists.

The production `benchmark-gate-smoke.sh` now exits **0** with
`BENCHMARK GATE SMOKE OK (0 failures)`. Its mmap-shaped and anonymous runtimes
both use `peak_resident_memory_upper_bound_bytes`. Scenario, warm-up, and soak
windows carry scored conservative resident upper bounds. A process-wide window
whose timestamp coverage is incomplete carries `status=partial`, raw samples,
the `resident-memory-sampling-gap` issue, and no `scoredBytes`; the decision
reports that delta as `unmeasured` rather than inventing a number. The 150 ms
production sampler probe also passes and retains timestamped samples.

The smoke fixture now preserves the real runtime shapes instead of sharing a
synthetic default: bounded Python/MLX defaults to a 2,048-token prefill step,
while llama.cpp defaults to a 512-token ubatch. An unpinned pair is therefore
refused with both effective values; an explicitly equal pair reaches admission.
Runtime records are opened by their production names (`python-mlx-lm` and
`mlx-swift`), and replay must reproduce the control decision byte-for-byte at
the decoded JSON level.

A second production defect surfaced during the negative provenance attack.
`BenchmarkRunCommand.drive` could throw after the runtime had served every
scenario but before one of its scattered shutdown calls, leaving the owned
`model-harness` group alive. A scope-exit termination now begins immediately
after successful spawn. The shipped decoy attack requires exit 5, no accepted
record, and no process matching its exact task-scoped config after the refusal.
A focused run reproduced the refusal at the real `benchmark-run` call site;
the subsequent exact `pgrep` exited 1, which is the expected proof of absence.

No admission, attestation, MTP-off, equivalence, context, memory, or parity
clause was weakened to obtain the green suite. Negative production cases still
refuse forged equivalence, missing or malformed live facts, unread
attestations, replay without attestations, context mismatch, speculative
decoding, HTTP-500 `/slots`, malformed prompt suites, and malformed tool
declarations. Absence and failed reads remain distinct.

### Revision-2 command outcomes

| Command | Exit | Evidence |
| --- | ---: | --- |
| focused decoy `benchmark-run` | 5 | expected provenance refusal; no decision |
| exact post-refusal `pgrep -f <task config>` | 1 | expected absence; no owned survivor |
| `swift test -c release` | 0 | 404 tests / 32 suites |
| `swift build -c release` | 0 | Release product built |
| `xcrun swift-format lint --strict --recursive Sources Tests` | 0 | clean |
| `bash -n scripts/benchmark-gate-smoke.sh` | 0 | syntax valid |
| default `shellcheck` | 1 | existing info-level SC2015/SC2086 findings; not reported green |
| `shellcheck -S warning scripts/benchmark-gate-smoke.sh` | 0 | repository warning-level lint clean |
| `git diff --check` | 0 | clean |
| full smoke attempt 03 | 1 | four stale exact-acceptance/lifecycle assertions exposed |
| full smoke attempt 04 | 1 | one assertion used a required key for an omitted optional `scoredBytes` |
| focused partial-memory assertion | 0 | one explicit partial record; omitted score and gap evidence verified |
| final full smoke attempt 05 | 0 | zero failures; mmap accounting and transient probe pass |
| post-smoke process snapshot | 0 | no task-owned model runtime or harness remained |

The first release-test invocation completed green but its initial 30-second
yield lost the command status, so it is not counted. The same command was run
again directly and produced the exit-0 evidence above.

The Story worktree remains deliberately unmerged as instructed. It was three
commits behind `main` in the spawn brief and was observed seven commits behind
at handoff because trunk advanced during the run. Revision 2 stays based on
`3272e3a`; integration owns the later trunk reconciliation.

### Revision-2 artifacts

- `benchmark-gate-smoke-r3-final-05.log`: green production smoke.
- `focused-decoy-cleanup-r3-01.log` and its empty pgrep result: expected-red
  provenance refusal plus cleanup evidence.
- `focused-mmap-partial-assertion-r3-02.log`: explicit partial-memory shape.
- `swift-test-release-r3-03.log`, `swift-build-release-r3-02.log`,
  `swift-format-lint-r3-03.log`, `shellcheck-warning-r3-02.log`,
  `bash-n-r3-03.log`, and `git-diff-check-r3-03.log`: validation evidence.
- `b8-before-final-smoke-05-*` and `b8-after-final-smoke-05-*`: raw host
  process and VM snapshots.

## Revision 1 measurement (preserved)

## Outcome

The pinned six-scenario suite completed for both runtimes in one
`benchmark-run` invocation. The Python baseline ran first and llama.cpp ran
second; their measured windows were non-overlapping, with the configured 20 s
settle between them. The start gate found no other resident model or listener
on ports 18000-18999. The post-run check found no owned model process and no
listener in that range. An idle Ollama service held no model and was not
touched.

The production command exited **4**. This is an expected-red refusal, not a
pass: Python reports `kv=unbounded`, while the capacity-capable llama.cpp
profile reports `kv=76800`. The gate refused `contextPolicy` and wrote no
`decision.json`. Both sealed records and attestations exist because both passes
completed before admission compared their pins.

Both records state `speculation=off`; llama.cpp also launched with
`--spec-type none`. No with-MTP number was taken. MTP-off is intentionally
against llama.cpp because the MLX artifact lacks the MTP head.

## Direct answers to the provisional claims

- **The provisional 10% decode deficit is overturned.** On `long_prompt_8k`,
  where both runtimes consumed 7,784 prompt and produced 106 completion tokens,
  corrected decode is 6.598 tok/s Python versus 7.880 tok/s llama.cpp:
  llama.cpp is **19.42% faster**, not slower. `short_prompt` is +23.73% and
  `context_75k` is +189.45%, but the latter has only 16 completion tokens and
  is a weaker decode-rate estimate. Every single-stream scenario with a
  measured corrected decode points in the same direction.
- **Corrected TTFT also favours llama.cpp in every measured single-stream
  scenario:** -75.87% on `short_prompt`, -37.66% on `long_prompt_8k`, and
  -37.18% on `context_75k`.
- **The provisional 9% capacity-memory advantage keeps its direction but not
  its magnitude.** Corrected conservative resident upper bounds at
  `context_75k` are 44.509 GiB Python versus 42.000 GiB llama.cpp, so llama.cpp
  is **5.64% lower**. The same values are the pass peaks. llama.cpp is not
  generally more frugal: it is 0.78% to 48.58% higher in the other five
  scenario windows.

These are measurements under refused context policies, not an accepted gate
decision. The comparison is useful evidence, but it cannot authorize migration
as though a fixed 76,800-token arena and Python's growing unbounded cache were
the same condition.

## Corrected measurements

All ratios are candidate / baseline. Lower is better for TTFT, wall clock, and
memory; higher is better for prefill and decode.

| Scenario | Metric | Python mlx-lm | llama.cpp | Ratio / delta |
| --- | --- | ---: | ---: | ---: |
| `short_prompt` | TTFT | 3.018s | 0.728s | 0.241x / -75.87% |
|  | prefill | 13.584 tok/s | 56.307 tok/s | 4.145x / +314.50% |
|  | decode | 6.659 tok/s | 8.239 tok/s | 1.237x / +23.73% |
|  | wall | 11.079s | 7.614s | 0.687x / -31.27% |
| `long_prompt_8k` | TTFT | 111.402s | 69.451s | 0.623x / -37.66% |
|  | prefill | 69.873 tok/s | 112.079 tok/s | 1.604x / +60.40% |
|  | decode | 6.598 tok/s | 7.880 tok/s | 1.194x / +19.42% |
|  | wall | 127.421s | 82.868s | 0.650x / -34.97% |
| `tool_call` | wall | 17.835s | 7.117s | 0.399x / -60.09% |
| `multiturn_prefix_reuse` | TTFT | 108.066s | 0.721s | 0.0067x / -99.33% |
|  | prefill | 73.529 tok/s | 428.529 tok/s | 5.828x / +482.80% |
|  | wall | 349.047s | 44.752s | 0.128x / -87.18% |
| `stability_soak` | wall | 299.032s | 218.245s | 0.730x / -27.02% |
| `context_75k` | TTFT | 1331.359s | 836.343s | 0.628x / -37.18% |
|  | prefill | 54.843 tok/s | 87.304 tok/s | 1.592x / +59.19% |
|  | decode | 2.957 tok/s | 8.558 tok/s | 2.895x / +189.45% |
|  | wall | 1336.448s | 838.098s | 0.627x / -37.29% |

The multi-request and semantic scenarios deliberately do not report a synthetic
decode rate: `tool_call` has no TTFT/prefill/decode, `multiturn_prefix_reuse`
has no aggregate decode, and `stability_soak` has no TTFT/prefill/decode. Those
dimensions are **unmeasured**, not omitted or inferred.

## Corrected resident-memory upper bound

Every value below is `ri_phys_footprint + upper edge of vmmap resident mapped
file`. All 286 baseline and 174 candidate samples were measured; read-failed and
malformed counts were zero. The raw components remain in the records.

| Scenario | Python mlx-lm | llama.cpp | Candidate delta |
| --- | ---: | ---: | ---: |
| `short_prompt` | 27.289 GiB | 32.347 GiB | +18.53% |
| `long_prompt_8k` | 32.825 GiB | 33.079 GiB | +0.78% |
| `tool_call` | 27.780 GiB | 34.149 GiB | +22.93% |
| `multiturn_prefix_reuse` | 31.614 GiB | 34.465 GiB | +9.02% |
| `stability_soak` | 28.232 GiB | 41.948 GiB | +48.58% |
| `context_75k` | 44.509 GiB | 42.000 GiB | **-5.64%** |
| pass peak | 44.509 GiB | 42.000 GiB | **-5.64%** |

The old report helper's `peak footprint` column reads the legacy Mach-only
diagnostic and therefore shows llama.cpp near 15.8 GiB. It is explicitly
excluded. The sealed candidate capacity sample is 15.80 GiB Mach plus a 26.1G
resident-mapped display bucket, producing the 42.000 GiB conservative upper
bound above.

## Parity and stability

- Prompt-token parity is exact on all six scenarios: 41, 7,784, 313, 7,784,
  910, and 73,016 tokens on both runtimes.
- `tool_call` succeeded on both production paths with 313 prompt tokens.
- `stability_soak` completed all 20 iterations on both runtimes. Aggregate
  output rate was 4.280 tok/s Python and 5.718 tok/s llama.cpp; median request
  latency was 12.532s versus 8.395s.
- Candidate soak memory climbed by 6,888,002,432 bytes (6.41 GiB), while the
  baseline declined by 498,204,624 bytes. The candidate's soak peak is therefore
  materially worse even though its 75k capacity peak is lower.

## Every refusal and declared non-comparability

1. Production admission refused `contextPolicy`: `kv=unbounded` versus
   `kv=76800`; exit 4, no decision document.
2. The MLX artifact drops the GGUF MTP head: 8 quantized tensors / 451,319,808
   bytes on disk. MTP was forced and observed off; direction is against
   llama.cpp.
3. Vision placement differs: 333 bf16 tensors in the MLX shards versus a
   separate 931,145,984-byte GGUF mmproj; neither is resident on this text path.
4. GGUF keeps norms and 1-D tensors in F32 where MLX keeps bf16, adding
   10,686,464 resident bytes on the GGUF side without a fidelity difference.
5. Pass order is fixed baseline then candidate. Baseline host-load maximum was
   22.59 versus candidate 10.16, so residual host load can disadvantage the
   baseline; direction is not normalized away.
6. Migration parity is one-way by policy: baseline success plus candidate
   failure blocks, while the reverse does not. That favours the incumbent and
   is not a measured performance dimension.
7. The memory score can double-count file-backed pages on a runtime whose Mach
   footprint already charges them. Residual direction is runtime-dependent;
   raw components are retained.
8. llama.cpp slot KV reuse persists across requests and fixed suite order can
   warm later prefixes. The 0.128x multiturn wall-clock ratio is therefore not
   a clean cold-prefill runtime ratio.
9. Tool-call and multi-request first-token/decode dimensions listed above are
   unmeasured and were refused from numeric interpretation rather than filled
   from proxy signals.

## Instrumentation regression fixed before the accepted run

The composite reader initially inherited the old 250 ms cadence used by the
cheap in-process Mach read. It now launched blocking `vmmap -summary` at that
cadence. Two attempts produced no baseline record before timeout; the latest
reached only 32,768/73,016 capacity prompt tokens.

`RuntimeMemoryAccounting.samplingIntervalSeconds` now owns a 5 s production
cadence, and `BenchmarkFootprintSampler` has no caller override. Every scenario
still takes a synchronous boundary sample. A focused contract test pins the
cadence above the retired 250 ms value. On the rerun, baseline capacity reached
73,016 tokens in about 22 minutes and both passes completed in 57m03s total.

Separately, termination of the previous timed-out RUN-260829-d6b68d left
`model-harness` pid 40394 and Python child 40395 holding about 31 GiB. External
process-group signalling was required. This reproduces TASK-260828-28gdmq B7
in production; the accepted rerun's post-run verification found no survivor.

## Gate evidence and validation

Production call sites named by the negative evidence:

- `BenchmarkRunCommand.run` derives live context/speculation evidence and calls
  `RuntimeBenchmark.admit`; `refusesTheLlamaCPPCandidateAgainstThePythonIncumbent`
  narrows across six llama.cpp bounds and requires the context refusal.
- `BenchmarkRunCommand.speculationAnswer` reads `/slots`; tests require a 500,
  malformed body, missing field, contradictory placement, and a true field in
  either placement to remain non-admissible.
- `RuntimeBenchmark.decide` consumes re-derived memory components;
  `refusesDecodedMachOnlyComposite` forges both derived fields back to Mach-only
  and requires the decision to block.
- `BenchmarkHTTPDriver.stream` consumes `RuntimeStreamDelta.read`; tests require
  `reasoning` and `reasoning_content` to share the generated-event definition.

Command outcomes, reported literally:

| Command | Exit | Evidence |
| --- | ---: | --- |
| first focused-test attempt | 1 | command did not start; log redirect used a wrong relative path |
| `swift test -c release --filter productionVMMapCadenceIsBounded` | 0 | focused cadence test passed |
| `swift test -c release` | 0 | 402 tests / 32 suites passed |
| `swift build -c release` | 0 | Release benchmark binary built; existing deprecation warnings retained |
| `xcrun swift-format lint --strict --recursive Sources Tests` | 0 | clean |
| `git diff --check` | 0 | clean before the run |
| full `benchmark-run` | 4 | expected-red `contextPolicy` refusal after both full passes |
| post-run process/listener/artifact verification | 0 | no model listener/survivor; both records and attestations present |

## Artifacts

- `session-c76800-corrected-cadence5/records/*.json`: sealed corrected records.
- `session-c76800-corrected-cadence5/attest/*.json`: live-process attestations.
- `session-c76800-corrected-cadence5/logs/*.log`: runtime logs.
- `session-c76800-corrected-cadence5/session.json`: session evidence.
- `benchmark-c76800-corrected-cadence5.log`: command progress and final refusal.
- `post-run-verification-01.log`: teardown/listener check and artifact inventory.
- `swift-test-release-01.log`, `swift-build-release-01.log`,
  `swift-format-lint-01.log`, `git-diff-check-01.log`: validation evidence.
