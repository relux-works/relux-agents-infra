# Review verdict — TASK-260828-2wcrph — revision 2

Change Request: `CR-TASK-260828-2wcrph-2`, revision 2  
Reviewed candidate tree: `c0672fdd399c905f6e0f2238dcb6fe3f692ec53b`  
Verdict: **accepted**

## Verdict

Revision 2 is honest interim evidence and resolves the revision-1 blocking
finding. The replacement paired probe obtains authoritative streamed usage from
both runtimes, applies a symmetric request and timing boundary, and limits the
flipped decode result to one bounded observation. Gate-recorded llama.cpp
latency remains excluded; `TASK-260829-3cwcb6` owns the production
`reasoning_content` fix and full rerun.

## Independent raw-SSE verification

The board-owned archive
`TASK-260828-2wcrph_rework-evidence.tgz` has SHA-256
`551f3eeece2c43a4927b004f7fc5f0416f59b0f40d82184d84cabb01fc409361`.
The raw streams, not their summaries, establish:

| Runtime | Prompt | Completion | Total | First text | Last text | Prefill | Decode |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| Python mlx-lm | 7,784 | 106 | 7,890 | 103.309696292s | 115.736693917s | 75.346267382288 tok/s | 8.449345784759 tok/s |
| llama.cpp | 7,784 | 106 | 7,890 | 76.784443167s | 88.754397500s | 101.374701423183 tok/s | 8.771963290664 tok/s |

Each stream contains one non-empty usage packet carrying
`completion_tokens: 106`; the count was not inferred from the coincidentally
equal 106 SSE frames. I independently recomputed prefill as
`prompt_tokens / first_text` and decode as
`(completion_tokens - 1) / (last_text - first_text)`. The results reproduce the
published summaries.

## Symmetry and the flipped direction

- After deleting only the runtime-specific `model` field, the two request
  objects are byte-identical: same messages, `stream: true`,
  `stream_options.include_usage: true`, `max_tokens: 256`, `temperature: 0.0`,
  and `top_p: 1.0`.
- Both runtimes use the first SSE delta carrying generated text and the last
  such delta. The probe treats Python's `reasoning` and llama.cpp's
  `reasoning_content` identically, then also counts `content` on both sides.
- The raw shape corroborates that boundary: Python has 89 reasoning and 15
  content frames; llama.cpp has 89 reasoning-content and 14 content frames.
  These counts remain diagnostics only.
- The corrected observation favours llama.cpp on TTFT, prefill, and decode.
  Because the decode direction flipped from the unsupported revision-1 claim,
  I checked every request field and timing boundary above rather than relying on
  the summaries. No asymmetry explaining the flip was found.

The report, README, and logbook consistently qualify 1.038x decode as one
bounded observation that establishes no general or suite-level decode
direction. The unsupported 8.88/9.85 tok/s replacement figures and the
“decodes far faster” attribution are absent. The historical gate value 80.79
tok/s remains only where explicitly labelled an artifact and excluded. All
three documents state the mixed-direction defect correctly: omitting
`reasoning_content` inflates llama.cpp TTFT, understates prefill, and overstates
decode.

## Acceptance-criteria evidence

The attached round-1 benchmark evidence and the prior review establish that
all pinned scenarios ran sequentially on the same `MacBookPro18,2`, with model
process sweeps reporting none before and after each run. I did not rerun the
roughly hour-scale 28/29 GB suite in this headless review. The retained evidence
records exact prompt-token parity on all six scenarios, successful 73,016-token
capacity runs on both runtimes, tool-call parity, 20/20 stability iterations,
and peak-memory treatment that does not use `resident_bytes` or
`ri_phys_footprint` alone for sizing. All three production comparisons were
correctly refused and left unscored on the irreconcilable `contextPolicy`
mismatch.

## Validation performed in this review

- Candidate/current blob identity: all seven changed paths match candidate tree
  `c0672fdd399c905f6e0f2238dcb6fe3f692ec53b`.
- Change Request patch SHA-256 matches the handed-off
  `ef0f5b5b61630ee093e380e57fcf2b04ec5543b1fb14c5b335c075f1b1c5538f`.
- Independent raw-SSE recomputation: passed for both runtimes.
- Probe negative tests: 4/4 passed; absent, malformed, and inconsistent usage
  refuse a decode rate, and frame count is not substituted for token count.
- `swift test -c release --filter RuntimeObservationReadingTests`: 21/21 passed.
- `swift test -c release --filter RuntimeBenchmarkContextBoundTests`: 19/19 passed.
- `git diff --check` on the exact Change Request range: passed.
- The producer's full candidate-tree evidence reports 392 Swift tests / 30
  suites, release build, Xcode release build, format lint, shellcheck, and the
  production benchmark smoke all passing. Revision 2 changes only the report,
  README, and logbook relative to the already-reviewed revision 1 code.

No blocking or rework finding remains.
