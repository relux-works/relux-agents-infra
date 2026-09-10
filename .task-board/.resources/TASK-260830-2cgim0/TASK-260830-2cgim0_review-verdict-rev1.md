# Review verdict — TASK-260830-2cgim0 rev 1 — ACCEPTED

## Scope reviewed
CR-TASK-260830-2cgim0-1, base b138ebc -> candidate 0a8dc6a. 6 files: LOGBOOK.md,
README.md, tools/agents-infra/engine_knob_profile_docs_test.go,
tools/agents-infra/internal/modelharness/{config.go,engine_knobs.go,engine_knobs_test.go}.

## AC coverage — 5 of 5 rows driven through `modelharness.Resolve`
1. Engine/model/environment independent, engine defaults to mlx-lm, golden
   qwen-local profile unchanged — `TestREADMEQwenLocalProfileResolvesWithDefaultEngine`
   (hands the exact README bytes to `Resolve`, asserts byte-identical argv) +
   `TestResolveDefaultsEngineToMLXLM`.
2. Canonical knobs (spec Knobs 1-4, the only ones with a launch argv spelling)
   translated per engine through one table (`knobEngineTable` in engine_knobs.go)
   — `TestResolveTranslatesKnobsPerEngine`, cross-checked against the
   documented README table via `TestREADMEEngineKnobProfilesResolveAsDocumented`.
3. Unexpressible knob refuses naming knob+engine instead of being dropped —
   `TestResolveRejectsUnexpressibleKnobValue` (llama-cpp + `kv_context_tokens=unbounded`).
4. Three named negative tests present and each independently verified by me
   with a narrowing mutant (not just read):
   - unknown engine: `TestResolveRejectsUnknownEngine` — mutant admitting
     `Engine("vllm")` into `knownEngines` caught it (`error = <nil>, want unknown engine refusal`).
   - unexpressible knob: `TestResolveRejectsUnexpressibleKnobValue` — mutant
     making `formatKVContextTokensAsCtxSize` accept `"unbounded"` silently
     caught it.
   - knob valid only for another engine: `TestResolveRejectsKnobOnlyValidForAnotherEngine`
     — mutant adding `EngineMLXLM: formatSpeculativeDecoding` to the
     `speculative_decoding` row caught the mlx-lm subtest (mlx-swift subtest
     unaffected, as expected of a targeted mutant).
   All three mutants were hand-applied and reverted by me in this review
   session, not taken on the producer's word.
5. Spec cross-referenced from profile docs — `TestREADMECrossReferencesTheEngineAdapterSpec`
   asserts the exact path string and that the file exists.

## Spec fidelity
Checked engine_knobs.go's knob table against
`.research/260831_engine-adapter-contract-and-canonical-knob-set.spec.md`
Knobs 1-4 directly:
- Knob 1 (KV/context): mlx-lm/mlx-swift `--max-kv-size` (omitted when
  unbounded), llama-cpp `--ctx-size` (unbounded refuses, matches spec's
  "always finite... absent falls back to trained context, not unbounded").
- Knob 2 (prefill chunk): mlx-lm/mlx-swift `--prefill-step-size`, llama-cpp
  `--ubatch-size` (not the false-friend `--batch-size`/`-b`) — matches spec exactly.
- Knob 3 (reasoning effort): mlx-lm `--chat-template-args '{"reasoning_effort":...}'`,
  mlx-swift/llama-cpp `--reasoning-effort` — matches spec exactly.
- Knob 4 (speculative decoding): llama-cpp only, `--spec-type`; mlx-lm/mlx-swift
  refuse. Verified independently against
  `tools/mlx-swift-runtime-prototype/Sources/.../RuntimeOptions.swift`
  that mlx-swift does in fact parse `--max-kv-size`/`--prefill-step-size`/
  `--reasoning-effort` (so those three rows correctly admit mlx-swift) and
  carries no `--spec-type` flag (so the refusal is correct, not a guess).
- Correctly scoped to Knobs 1-4 only (launch-argv-expressible); Knobs 5-10
  (wire-protocol field naming, cache/memory telemetry, health, argv-parsing
  precedence) are live-report/benchmark-gate concerns with no launch spelling,
  out of this task's scope per its own scope note, and the code comment says so.

## Build/test evidence (rerun by me, not reused)
- `go build ./...` — exit 0
- `go vet ./...` — exit 0
- `gofmt -l` on changed files — clean
- `go test ./internal/modelharness/... -v` — all pass (14.3s)
- `go test . -run 'Engine|Qwen|README' -v` — all pass
- `go test ./... -count=1` (full tools/agents-infra suite: root, cmd/model-harness,
  internal/attachments, internal/infra, internal/modelharness) — all pass (root
  109.2s, infra 183.5s, modelharness 15.0s)
- `git diff --check` on the exact reviewed range — clean, no whitespace errors

## Scope discipline
No engine adapter implementation added (out of scope, belongs to
TASK-260830-1e9gse). Deployed qwen-local profile semantics unchanged — proven
by golden test, not merely asserted. LOGBOOK entry documents the design
decision (environment axis reuses existing executable/argv, no new field) and
the one finding worth recording (mlx-swift prototype's argv parser accepts
Knobs 1-3 contrary to a naive reading of the live-reportability table).

## Verdict
ACCEPTED. All AC rows have named driving tests through the production entry
point; three independently-verified narrowing mutants confirm the negative
tests are real, not decorative; spec fidelity checked line-by-line against
the cited knobs; full build/vet/test suite green under my own rerun.
