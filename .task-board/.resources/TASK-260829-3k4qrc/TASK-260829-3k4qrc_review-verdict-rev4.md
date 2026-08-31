# TASK-260829-3k4qrc revision 4 review verdict

Verdict: **accepted**. Recorded with `accept_cr`, which parks the element at
`to-review` for the orchestrator.

Reviewed Change Request `CR-TASK-260829-3k4qrc-4` revision 4, base
`3272e3ae60b53eb0fb61f6d427a883487711b48f`, candidate tree
`bafa676bada78f5cdb6481326977086a20aaf7e7`. The attached patch reproduces
SHA-256 `f0011b033f1a86b5c103dc1de5f3930c876528f0b013a90e798511cee45514c3`, and
the review worktree writes exactly tree `bafa676b` both before and after the
review, so every finding below is against the exact reviewed revision and no
repository file was modified by the reviewer.

**Answer to the standing question: yes.** The instrumentation is now trustworthy
enough to spend the hour on, and the carried-forward cached-tokens unknown is
closed favourably by live probe (below), so nothing is left that would sink the
pair before it starts.

**Scope caveat, stated so the orchestrator does not over-read this acceptance.**
This accepts the *instrumentation revision*, not the comparison numbers. The
task's acceptance criterion — the full pinned suite actually run on corrected
instrumentation — is still outstanding by design; the brief forbade both the
producer and me from running the hour-scale pair. Do not transition the element
to `done` on the strength of this revision alone.

## 1. Did the pendulum swing back? No — but the output shape is identical, so this needs care

The revision-2 attack was re-run against the exact candidate sources, compiled
unmodified in reviewer scratch (`RuntimeMemoryAccounting.swift` SHA-256
`1ff3e6ad…`, byte-identical to the candidate). A 256 MiB file-backed region
mapped, faulted fully resident, and released strictly between two mapped
observations:

```
directMappedWhileResident=268645172
scoredMapped=82944
status=measured issues=[] scored=Optional(2475608)
mappedStamps=2 widestMappedGap=2.250  statedLimit=Optional(7.0)
note=mapped-file resident transients shorter than 7.0s are not observable; ...
```

Those are the *same two numbers revision 2 produced* (`268645172` /`82944`,
`measured`, `issues=[]`). That resemblance is the trap in this review, and it is
superficial. Revision 2's defect was a **stale mapped value minted as fresh**: a
5 s-old `vmmap` reading carried a fresh Mach timestamp and thereby claimed
sub-cadence resolution it never had. Here nothing is stale. The mapped
timestamps are independent and truthful (2.25 s apart), and `82944` is a
genuine, current observation taken at the closing boundary read, when the region
really was gone. What is missed is a transient the instrument openly declares it
cannot see, in the contract, in `mappedFileObservabilityNote`, and in every
emitted record.

This is direction 1 of the two the revision-3 verdict authorized, executed as
specified. The missed class is also not decision-relevant: the mapped component
this comparison scores is the model weights, resident for the whole run;
sub-cadence *anonymous* growth is the real risk and is covered separately by the
Mach series at 20 Hz with its own unchanged 125 ms bound.

## 2. Is the bound derived or reverse-engineered? Derived — and more tightly than the comment claims

Reader cost measured independently, through a fork/read-to-EOF/wait shape
against a 1.6 GB, ~400-mapping target: **0.663–0.957 s, mean 0.731** over 8
consecutive calls. The producer's figures (0.608–0.672 s at ~3 MB, 0.783–0.850 s
at ~0.9 GiB) reproduce and are consistent. `1.0 s` is an honest, modest
round-up.

The formula reproduces exactly: `samplingIntervalSeconds (5) + 2 x
observedMappedFileReadCostSeconds (1.0) = 7.0`.

The `2 x` is better justified than its own doc comment says. The comment calls
the second reader cost "scheduling headroom". It is not — it is read latency at
the far end. `mappedFileSampledAt` is stamped *after* the read completes
(`BenchmarkFootprintSampler.swift:65`), so a value may already be up to one
reader cost old when stamped; consecutive stamps are `interval + C` apart; the
true maximum window in which a mapped transient is invisible is therefore
`interval + 2C` — exactly 7.0, and exactly what the note tells the reader. Same
constant, stronger reason. Non-blocking, but the comment understates its own
rigor.

The constant is pinned in both directions by the contract suite
(`RuntimeStreamDeltaTests.swift`): `== interval + 2*readCost`, `>= interval +
readCost` (unreachable-from-below), `< 2 * interval` (not open-ended), and
`readCost >= 0.85`.

## 3. Does the new positive actually fail? Yes, on every mutant

Every check was executed against the candidate sources, not read.

| Mutant | anon | mapped | refusal | stop |
| --- | ---: | ---: | ---: | ---: |
| unmodified candidate | 0 | 0 | 0 | 0 |
| mapped reader broken (`vmmap` repointed at a nonexistent path) | 1 | 2 | 1 | 1 |
| contract constant narrowed to 0.125 | 1 | 1 | 1 | 1 |
| contract constant widened to 600.0 | 0 | 0 | 0 | 0 |

The broken-reader row is the one the brief asked for: nothing passes through a
refusal branch. The refusal probe fails too, because its own control half must
score — which is precisely the property revision 3 lacked.

The widened row is not a gap: widening is caught by the contract suite's
`< 2 * samplingIntervalSeconds` expectation rather than by the probes. Placement
is adequate. (That one is read-and-reasoned from a literal numeric comparison in
shipped test code, not executed; every other row in the table was executed.)

**The override cannot buy a laxer gate**, proven behaviourally rather than by
reading the `min`. With `narrowedMappedFileGapSeconds: 3600` and a real 11.67 s
mapped hole injected by slowing the reader, the peak is still `partial`,
`resident-mapped-file-sampling-gap`, `scored=nil`, and the record still states
7.0. The same hole under the contract bound refuses identically. This also shows
the 7.0 bound remains useful: a stalled reader is still caught.

## 4. The smoke FAIL — replay not accepted; I ran the suite

I did not accept the replay argument. I ran the full suite myself:

```
BENCHMARK GATE SMOKE OK (0 failures)   —   120 PASS / 0 FAIL, exit 0
```

The `warmupMemory` FAIL is gone, and the acceptance path is genuinely taken, not
merely reachable:

```
accepted: True   blockers: []
  short_prompt  verdict=within  baseline=11993424  candidate=16548296
  process       verdict=within  baseline=12009808  candidate=16548296
```

That is exactly what F4 required and exactly what every revision-3 check failed
to reach. The replay question is moot.

The assertion scoping is also correct on its merits, verified statically:
`recordWarmupMemory` summarizes exactly one reading
(`BenchmarkPass.swift:99-101`), and `RuntimeBenchmark.decide` references neither
`warmupMemory` nor `soakMemory`. The `if len(raw) > 1` guard cannot mask a real
window, because the sampler's own coverage gate refuses any series with fewer
than two samples.

**My own error, reported rather than omitted.** My first smoke run showed 13
failures. The cause was mine, not the change's: I left `swift test -c release`
running, which relinked `.build/release` mid-run, so sessions were attested by
gate binary `5b7ef74d…` while replays were made by `a3f11a9c…`. The rerun above
used a snapshotted binary (`a3f11a9c…`) with nothing else touching `.build`.

## 5. The carried-forward unknown — closed, by live probe against the real binary

The producer flagged this without closing it. It is now closed, and closed
favourably. One short streamed request against the pinned
`llama-server 0.3.0 (build 10621, commit c1d0e7a00)`:

```
streamed, stream_options.include_usage=true, cold:
  "usage":{...,"prompt_tokens_details":{"cached_tokens":0}}
streamed, same prompt repeated:
  "usage":{...,"prompt_tokens_details":{"cached_tokens":54}}
streamed, no stream_options:
  0 usage frames at all
```

So the candidate does report it, and reports a real hit. The driver always sends
`stream_options: {"include_usage": true}` whenever it streams
(`BenchmarkScenarios.swift:89-96`), so `cacheReuseObservation` will return
`reported`, not `unknown`, and the F3-shaped catastrophe applied to every
dimension does not occur.

Two things worth recording. First, **the mechanism is not the one that was
feared**: the risk was never that llama.cpp omits the field, it is that the
whole usage frame only exists if the caller asks for it (`include_usage`
defaults to `false`), and the driver already asks. Second, this was settled
against the **binary**, not the source: the only llama.cpp tree on this host is
0.1.1, which does not match the shipped 0.3.0, so source reading alone would not
have been legitimate evidence.

## 6. Bounded-KV closures

Hold, in my own green run: the `--python-bin` decoy exits 5 and cannot mint an
accepted record, missing live KV stays inadmissible at exit 4 despite a finite
launch flag, malformed `n_ctx` cannot buy an unbounded pin, and an unreadable
verdict is refused rather than spent as an absence. No regression from this
round.

## Validation reproduced by the reviewer

| Check | Result |
| --- | --- |
| patch SHA-256 and candidate tree OID | both reproduce exactly |
| `swift build -c release` | exit 0 |
| `swift test -c release` | exit 0, **410 tests / 32 suites** (matches claim) |
| `benchmark-gate-smoke.sh`, full run | **120 PASS / 0 FAIL, exit 0** |
| four production-entry probes, unmodified | exit 0 each |
| broken-reader / narrowed-constant mutants | fail as required |
| independent `vmmap` cost measurement | 0.663–0.957 s, consistent with 1.0 s |
| live `llama-server` streamed cached-tokens probe | reported, `0` then `54` |

Decision chain re-verified: `validatedScoredBytes == nil` → `appendDelta`'s
`guard let baseline, let candidate` → blocker → `accepted = blockers.isEmpty`
false → the control's `assert accepted is True` goes red. F4's fix is
load-bearing, not decorative.

## Non-blocking items for the record

1. **The one thing that could still cost the memory dimension on the real pair.**
   The bound refuses when the real reader cost exceeds 2.0 s, since the mapped
   gap is `interval + C` against a 7.0 bound. Measured `C` tops out near 0.96 s
   at 1.6 GB; the 28 GiB runtimes are untested and `vmmap` cost rises with
   mapping count. This fails *safe* — a visible refusal, not a wrong number —
   but it is the single most likely way `context_75k` memory comes back
   `unmeasured`. Worth one `vmmap -summary` timing against each runtime once
   loaded, before committing to the hour.
2. **`warmupMemory` / `soakMemory` bypass the coverage gate.** Both are built by
   direct `RuntimeMemoryPeak(summarizing:)` construction rather than through
   `coveredPeak`, so they can be `measured` with no coverage judgement at all.
   Harmless today because `decide` reads neither; it becomes a trap the moment
   anything scores them.
3. **Cache-policy symmetry is a run-design item.** My probe showed llama-server
   reuses prompt cache across identical requests by default (`cached_tokens=54`
   on the repeat). If the pinned baseline's policy differs, F2 will correctly
   refuse repeated-prompt scenarios as one-sided rather than score them. That is
   the gate working, not a defect — but it should be settled deliberately before
   the pair, or scenarios will come back non-comparable for a reason that had
   nothing to do with the runtimes' speed.
4. The `2 x readCost` doc comment attributes the second term to scheduling
   headroom; it is actually read latency at the far end (see section 2).

## Definition of Done status

- Corrected instrumentation reaches a scored memory dimension at the production
  entry — **met**, demonstrated by a fresh accepted run.
- Non-comparable dimensions refused rather than scored, and refusals reported —
  **met**; memory is no longer refused unconditionally, and the refusal branch
  keeps its own fixture.
- Negative tests that fail when the gate admits what it must reject, with the
  production call site named — **met**; broken-reader and narrowed-constant
  mutants both fail, and the override cannot widen.
- Tests / lint / build green — **met**, reproduced.
- MTP off for scored comparisons — unchanged, not regressed.
- Both runtimes run sequentially on a clean host; corrected decode/TTFT and
  corrected memory reproduce or overturn the provisional deficits — **still
  outstanding**, deliberately, per the brief. These belong to the rerun, not to
  this revision.

Reviewer artifacts (fresh smoke log, decision document, probe outputs, mutant
outputs, vmmap timings, llama-server transcripts) are under
`.temp/TASK-260829-3k4qrc/reviewer-r4/`.
