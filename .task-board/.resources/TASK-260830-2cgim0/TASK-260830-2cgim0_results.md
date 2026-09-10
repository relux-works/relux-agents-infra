rev2 = accepted rev1 re-applied on trunk 67e9c43, kind story_final, product delta unchanged

## Steps taken
1. Fresh managed worktree provisioned at trunk 67e9c43 (STORY-260830-112ewt landed).
2. Applied CR-TASK-260830-2cgim0 rev1 (TASK-260830-2cgim0_change-request_rev1.patch, identical copy .temp/goal-audit/2cgim0-candidate-full.patch) via git apply --3way.
3. All 5 non-LOGBOOK files (README.md, tools/agents-infra/internal/modelharness/config.go, tools/agents-infra/internal/modelharness/engine_knobs.go, tools/agents-infra/internal/modelharness/engine_knobs_test.go, tools/agents-infra/engine_knob_profile_docs_test.go) applied CLEANLY (git apply reported clean/direct application, no conflict markers) -- byte-identical to rev1 hunks, product delta unchanged.
4. LOGBOOK.md had a 3-way merge conflict (both trunk 1730 entry and candidate 1743 entry are newest-first under 2026-09-11). Resolved manually by inserting the candidate 1743 entry above the trunk 1730 entry, newest-first, removing no trunk line.

## Proof 1: product files byte-identical to rev1 (no product code changed)
git apply reported README.md and config.go applied cleanly; the three new files (engine_knobs.go, engine_knobs_test.go, engine_knob_profile_docs_test.go) are wholly new-file additions from the patch with no fallback/fuzz needed. No manual edits touched any of these 5 files.

## Proof 2: LOGBOOK.md diff against trunk is additions-only
Command: diff <(git show 67e9c43:LOGBOOK.md) LOGBOOK.md
Result: every changed line is a "> " addition (the new 1743 entry, 6 lines incl. blank separator). Zero "<" deletion lines. No trunk content removed.

## Validation run (bounded, tools/agents-infra module)
- go build ./... -> exit 0
- go vet ./... -> exit 0
- go test -count=1 ./internal/modelharness/... . -> ok (13.063s), ok (89.312s), exit 0
- Targeted: go test -run '^TestREADME' -v . -> all 6 TestREADME* subtests PASS, including TestREADMEEngineKnobProfilesResolveAsDocumented/qwen-local-mlx-lm and /qwen-local-llama-cpp, TestREADMECrossReferencesTheEngineAdapterSpec

## Conclusion
rev2 = accepted rev1 re-applied on trunk 67e9c43, kind story_final, product delta unchanged.