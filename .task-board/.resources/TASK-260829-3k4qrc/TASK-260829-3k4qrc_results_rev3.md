# TASK-260829-3k4qrc — Revision 3 developer outcome

Date: 2026-08-30
Role: developer
Base: `3272e3a` (`task-board/story/STORY-260828-2faxgm`), intentionally 7 commits behind `main` as required by the rework brief

## Scope delivered

Revision 3 fixes both blocking revision-2 findings without changing the bounded-KV merge:

1. Mach physical-footprint and mapped-file observations now carry independent timestamps. Reusing a mapped value preserves its original timestamp, coverage is checked independently for both components, and insufficient or stale mapped coverage yields an explicit partial result with no score.
2. Every scenario seals structured runtime-reported cached-token telemetry (`hit`, `miss`, or `unknown`). `RuntimeBenchmark.decide` consumes it: one-sided hits, unknown/malformed facts, and applicability mismatches make the scenario non-comparable; symmetric reported hits or misses remain scoreable.

The implementation, rationale, and regression evidence are recorded in `LOGBOOK.md` under the 2026-08-30 1432 entries. The package README, repository tool table, and comparison research header describe the revision-3 contract.

## Negative production evidence

### F1 — file-backed transient

The real `benchmark-mapped-file-sampler-probe` mapped and touched 256 MiB for less than the mapped-file refresh interval:

- direct mapped-file observation: `268645172 B`
- stale periodic mapped-file component: `82944 B`
- production window: `partial`
- issue: `resident-mapped-file-sampling-gap`
- scored bytes: absent
- probe exit: `0`

The anonymous 128 MiB probe remains separate. Its 20 Hz Mach series captured the transient, while the independently stale mapped component caused the expected explicit refusal; exit `0`, 25 raw samples in the final smoke.

### F2 — one-sided cache reuse

The real `benchmark-run` entry received identical short-prompt token counts and asymmetric sealed reuse telemetry:

- baseline: 14 prompt tokens, cache `miss`, cached tokens `[0]`
- candidate: 14 prompt tokens, cache `hit`, cached tokens `[14]`
- blocker: `short_prompt/cache_reuse is one-sided (baseline miss, candidate hit); the scenario is not comparable`
- TTFT, prefill, and decode verdicts: `non-comparable`
- memory verdict: `unmeasured`
- production decision exit: `3` (expected rejection)

The symmetric no-hit control remains scoreable on timing dimensions. Unit coverage also proves symmetric hits and misses stay comparable, unknown refuses, malformed structured facts refuse, and cache-reuse edits change the sealed transcript digest.

## Validation

All commands were run as standalone processes; exit codes are the real process exits.

- `swift test -c release --filter RuntimeMemoryAccountingTests` — exit `0`, 6 tests
- `swift test -c release --filter RuntimeBenchmarkTests` — exit `0`, 76 tests
- `swift test -c release` — exit `0`, 407 tests in 32 suites
- `swift build -c release` — exit `0`; existing unrelated `quantization` deprecation warning only
- `xcrun swift-format lint --strict --recursive Sources Tests` — exit `0`
- `bash -n scripts/benchmark-gate-smoke.sh` — exit `0`
- `shellcheck -S warning scripts/benchmark-gate-smoke.sh` — exit `0`
- `git diff --check` — exit `0`
- `.build/release/mlx-swift-runtime-prototype benchmark-memory-sampler-probe` — exit `0`
- `.build/release/mlx-swift-runtime-prototype benchmark-mapped-file-sampler-probe` — exit `0`
- full `benchmark-gate-smoke.sh`, run 1 — exit `1`, 117 production checks passed and one smoke assertion rejected a valid Mach coverage refusal because it allowed only mapped-file coverage refusals
- isolated revised mmap-session assertion against that exact production output — exit `0`
- full `benchmark-gate-smoke.sh`, run 2 — exit `0`, 118 production checks, 0 failures

Logs:

- `.temp/TASK-260829-3k4qrc/swift-test-release-rev3.log`
- `.temp/TASK-260829-3k4qrc/swift-build-release-rev3.log`
- `.temp/TASK-260829-3k4qrc/benchmark-gate-smoke-rev3.log` (preserved red run)
- `.temp/TASK-260829-3k4qrc/benchmark-gate-smoke-rev3-02.log` (green run)

## Deliberately not run

The hour-scale pinned llama.cpp versus Python mlx-lm pair was not run, exactly as required by the revision-3 rework brief. Consequently revision 3 publishes no replacement decode, TTFT, or 75k memory number. Revision-1 numbers remain historical and outside the decision. The next measurement run must keep MTP off, run the two runtimes sequentially on a host with no other model resident, and either score both `context_75k` memory observations under the corrected freshness contract or explicitly refuse that dimension.
