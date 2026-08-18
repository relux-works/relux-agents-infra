# BUG-260817-3nk7yf — Reviewer Verdict Cycle 4

## Verdict

**Changes requested → `to-dev`.**

Cycle-3 correctly closes nested absolute escapes and direct ancestor/self links. The broader cycle invariant remains bypassable: the validator proves each symlink is individually contained but does not prove that the composed symlink graph is acyclic.

## Blocking finding — contained transitive cycle passes setup and verify

The production validator physically walks directories and evaluates links one at a time at `tools/agents-infra/internal/infra/skill_link_validation.go:92-150`. `filepath.EvalSymlinks(path)` resolves the link's target path, but it does not traverse that target directory's descendants. Therefore two links can each resolve to a contained directory while mutually re-entering one another during recursive traversal.

A fresh source copy was given this graph:

- `.skills/transitive-cycle-probe -> ../cycle-target`
- `cycle-target/back -> ../.skills/transitive-cycle-probe`

The source-built production artifact was then driven through the real call sites:

- setup preflight: `tools/agents-infra/internal/infra/infra.go:134-139`
- verbatim materialization: `tools/agents-infra/internal/infra/infra.go:313-318`
- setup postcondition and `verify local`: `tools/agents-infra/internal/infra/runtime_receipt.go:148-174`

Results:

| Probe | Result |
| --- | ---: |
| `agents-infra setup local` | exit 0 |
| `agents-infra verify local` | exit 0 |
| installed links retained | yes |
| `rsync -aL` over installed `.agents` | exit 0 with `directory cycle` warnings on both graph edges |
| focused production tests | pass |

This is the standard negative shape **bypass path around the check**. The focused tests cover direct and nested single-link ancestor cycles, but no test constructs a cycle through two individually contained targets; `rg` finds no transitive-cycle regression.

## Required rework

1. Enforce graph acyclicity for every symlink path setup owns, including cycles formed through multiple contained directories, before destination mutation and in setup/verify postconditions.
2. Preserve the intended global ownership boundary: traverse descendants of setup-managed skill packages without claiming unrelated provider-owned top-level packages.
3. Add an installed-binary production-entry negative for the two-link transitive cycle above. Require setup refusal before mutation and `verify local` refusal for equivalent installed drift.
4. Add a narrowing control where multiple contained relative links form a DAG but no cycle, so the fix does not reject legitimate contained topology.
5. Re-run focused/full tests, vet/build/gofmt, pristine/source/local-models setup+verify, symlink-following recursive inspection, diff/index checks, and board validation.

## Evidence

- `.temp/BUG-260817-3nk7yf/transitive-cycle-summary-04b.log`
- `.temp/BUG-260817-3nk7yf/transitive-cycle-setup-04b.log`
- `.temp/BUG-260817-3nk7yf/transitive-cycle-verify-04b.log`
- `.temp/BUG-260817-3nk7yf/transitive-cycle-links-04b.log`
- `.temp/BUG-260817-3nk7yf/transitive-cycle-follow-rsync-04b.log`
- `.temp/BUG-260817-3nk7yf/focused-review-tests-04.log`
- `.temp/BUG-260817-3nk7yf/board-validate-review-04.log`

No source code was modified by this reviewer.
