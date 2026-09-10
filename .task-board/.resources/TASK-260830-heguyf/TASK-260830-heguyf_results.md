# TASK-260830-heguyf — bound deployed qwen-local KV size and keep context_window consistent

## What changed

- `tools/agents-infra/internal/infra/pi_kv_bound.go` (new): `validatePiModelHarnessKVBound`
  runs inside `parsePiProfile` (the production `parseProjectConfig` entry point). For a Pi
  profile whose `runtime.argv` invokes `model-harness run PROFILE [--config PATH] ...`, it
  resolves the referenced model-harness profile through `modelharness.Resolve` (the same
  static, side-effect-free resolution `--print-config` uses — no process/socket/file
  mutation) and refuses when the resolved profile's `--max-kv-size` is absent or below
  `context_window`, naming both values. SSH-mode model-harness profiles and non-model-harness
  Pi runtimes are out of scope (documented in the function's doc comment) and pass through
  unchanged.
- `tools/agents-infra/internal/infra/pi_config.go`: wires the new check into `parsePiProfile`
  right after the existing runtime/endpoint argv validation.
- `tools/agents-infra/internal/infra/pi_kv_bound_test.go` (new): drives `parseProjectConfig`
  directly (see Coverage table below).
- `SKILL.md` / `README.md`: documented the `context_window` vs `--max-kv-size` relationship
  next to the existing operator-facing `model-harness stress` / context-window guidance,
  including the `--prompt-cache-bytes` vs `--max-kv-size` distinction.
- `LOGBOOK.md`: recorded the root cause, fix, scope exclusions, and gate evidence.

No change to any deployed/live config. `/Users/alexis/src/.agents/.configs/model-harness.toml`
and `/Users/alexis/src/.agents/.configs/project-config.toml` were only read (to confirm the
golden 75000/76800 pair and the exact `runtime.argv` shape used in production), never written.

## AC coverage — 5 of 6 rows driven by a named test; 1 of 6 is a stated documentation bound

| # | AC row | Driven by | Production call site |
|---|---|---|---|
| 1 | Deployed profile's KV bound (76800) is at least its context_window (75000) | `TestPiProfileAdmitsContextWindowAtOrBelowModelHarnessKVBoundAtProductionEntry/deployed_pair` (exact deployed numbers) | `parseProjectConfig` → `parsePiProfile` → `validatePiModelHarnessKVBound` |
| 2 | context_window > KV bound refused at production entry, error names both values | `TestPiProfileRefusesContextWindowAboveModelHarnessKVBoundAtProductionEntry/bound_below_context_window` and `/bound_exactly_one_below_context_window` | same |
| 3 | Raising context_window without raising bound fails closed (no silent truncation path exists) | same two subtests + `/missing_--max-kv-size_entirely` | same |
| 4 | Test drives real resolution path, window-above-bound → refusal | `TestPiProfileRefusesContextWindowAboveModelHarnessKVBoundAtProductionEntry` (all 3 subtests) | same |
| 5 | Test drives real resolution path, consistent pair → admission | `TestPiProfileAdmitsContextWindowAtOrBelowModelHarnessKVBoundAtProductionEntry` (equal-bound + deployed-pair subtests) | same |
| 6 | Relationship documented where an operator setting context_window will see it | **Stated bound, not test-drivable**: `SKILL.md` (next to the `model-harness stress` context-window paragraph) and `README.md` (next to the `agents.pi.profiles."qwen-harness".runtime` example) | n/a — documentation |

## Mutant evidence

| Mutant | Narrows the gate to | Failing test | Bound survival would state |
|---|---|---|---|
| Delete the `validatePiModelHarnessKVBound` call from `parsePiProfile` | No KV-bound check at all | `TestPiProfileRefusesContextWindowAboveModelHarnessKVBoundAtProductionEntry/missing_--max-kv-size_entirely` and `/bound_below_context_window` (both fail) | n/a — no survivor |
| `if bound < contextWindow` → `if bound+1 < contextWindow` (accepts exactly one token of slack, preserves the `--max-kv-size` token search) | Admits `context_window` up to 1 above the real bound | `TestPiProfileRefusesContextWindowAboveModelHarnessKVBoundAtProductionEntry/bound_exactly_one_below_context_window` | An initial version of this test used a 5000-token gap (70000 vs 75000) and did **not** catch this mutant — it survived. Added the exact off-by-one case (74999 vs 75000) specifically to close that gap; re-ran and the mutant now fails. This is the required narrowing mutant, and also satisfies "gate that inspects source text attacked by a mutant that preserves the searched-for token" since `--max-kv-size` presence detection is unchanged, only the numeric comparison is weakened. |

Both mutants were applied to the actual working tree, confirmed red, then reverted (`diff` verified byte-identical restore) before this handoff.

## Commands run

```
cd tools/agents-infra
go build ./...                                                    # exit 0
go vet ./...                                                       # exit 0
gofmt -l internal/infra/pi_kv_bound.go internal/infra/pi_kv_bound_test.go internal/infra/pi_config.go   # empty output (clean)
go test ./internal/infra/... -run 'TestPiProfile.*KVBound' -v      # exit 0, all subtests pass
go test -race ./internal/infra/... -run 'TestPiProfile.*KVBound'   # exit 0
go test ./...                                                      # exit 0 (full tools/agents-infra module: root, internal/attachments, internal/infra, internal/modelharness)
```

Full suite runtime: `tools/agents-infra` 104.9s, `internal/infra` 179.9s, `internal/attachments` 1.1s, `internal/modelharness` 14.1s — all green.
