# Review verdict: CR-TASK-260830-heguyf-1 rev1 — ACCEPTED

## Scope reviewed
LOGBOOK.md, README.md, SKILL.md, tools/agents-infra/internal/infra/{pi_config.go,pi_kv_bound.go,pi_kv_bound_test.go}
Base b138ebc -> candidate tree 1c48958.

## AC verification
- Production entry point: parsePiProfile (pi_config.go) now calls validatePiModelHarnessKVBound(p.ContextWindow, p.BaseURL, p.Runtime) right after the existing runtime-argv endpoint validator. Diff to pi_config.go is exactly the 4-line call-site addition -- no unrelated changes.
- Real resolution path: validatePiModelHarnessKVBound recognizes a model-harness-backed Pi runtime (model-harness run PROFILE [--config PATH]), resolves the referenced profile through modelharness.Resolve (pre-existing package, static parse, no process/socket side effects), and reads --max-kv-size from the resolved local-mode argv. No stubs/mocks in the production path.
- Refusal: missing --max-kv-size, and bound < context_window (including exact off-by-one), both refuse naming both values. Verified live: parseProjectConfig returns an error containing both numbers in each case.
- Admission: bound == context_window, and the deployed pair context_window=75000 / --max-kv-size=76800 (the qwen-local production pair) are both admitted.
- SSH-mode and non-model-harness runtimes are explicitly out of scope and unaffected (dedicated negative-control test), matching the stated reasoning that the bound lives on a host a local parse cannot observe.
- Deployed config verified directly (not just via test fixture): /Users/alexis/src/.agents/.configs/model-harness.toml already carries --max-kv-size 76800 for profiles.qwen-local; /Users/alexis/src/.agents/.configs/project-config.toml carries context_window = 75000 for the Pi profile that launches it via model-harness run qwen-local --config .../model-harness.toml. Consistent pair, AC satisfied on the actual deployed artifact, not only in a unit-test fixture.
- Documentation: README.md and SKILL.md both gained a section next to existing model-harness/context-window guidance explaining the context_window vs --max-kv-size relationship and the --prompt-cache-bytes distinction, where an operator setting context_window will see it. LOGBOOK.md records root cause, fix, scope, and gate evidence.
- context_window values and other profile settings are unchanged by this task; the deployed configs were read only, not modified in this CR (confirmed: qwen-local/model-harness.toml and project-config.toml are untouched by the diff, matches LOGBOOK claim).

## Adversarial mutant testing (gate defeat attempts)
1. Off-by-one narrowing mutant: changed `bound < contextWindow` to `bound+1 < contextWindow` (wrongly admits bound == contextWindow-1). Result: the dedicated off-by-one boundary subtest (bound exactly one below context_window, 74999 vs 75000) went red as claimed in the LOGBOOK gate note. Restored original after.
2. True bypass mutant: changed the !present branch to `return nil` (admit a profile with no --max-kv-size at all instead of refusing). Result: the missing-bound refusal subtest went red immediately. Restored original after.
3. A third exploratory mutant (widening the !present branch condition to `!present && contextWindow > 80000`) did not flip the overall pass/fail because the code falls through to the bound<contextWindow check with bound=0, which still refuses (with a different message) for the tested context_window=75000 -- so the actual admit/reject decision was unchanged and this is not a real gate weakness, just a alternate message path for one unexercised sub-condition. Not a defect worth blocking on; noted for completeness.

## Build/test evidence
- go build ./... : clean
- go vet ./... : clean
- gofmt -l on all 3 touched/added infra files : no output (clean)
- go test ./internal/infra/... -run TestPiProfile -v : all new tests PASS
- go test ./... (full module, all packages) : ok, exit 0 (agents-infra pkg 189.6s, internal/infra pkg 363.6s, others cached ok)

## Worktree hygiene
git status matches the CR change list exactly (6 paths, all uncommitted as required); no stray files from mutant testing left behind; git write-tree confirms candidate tree oid dc543c5e (worktree state at time of review, unchanged from CR revision).

## Verdict
ACCEPTED. Gate is real, narrowing-mutant-resistant, and bypass-mutant-resistant; deployed production config independently verified consistent; docs and logbook meet the AC; full test suite green.
