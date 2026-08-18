# BUG-260817-3nk7yf — Reviewer Verdict Cycle 2

## Verdict

**Changes requested → `to-dev`.**

The cycle-1 top-level escape and ancestor-cycle inputs are now refused through the production setup path, and the focused suite is green. The acceptance criterion still has a bypass: the new validator inspects only top-level entries, while `syncRepo` copies symlinks at every depth.

## Blocking finding — nested skill symlinks bypass containment

`inspectSkillLinks` delegates directly to `inspectTopLevelSkillLinks` at `tools/agents-infra/internal/infra/skill_link_validation.go:53-60`; non-symlink directory entries are skipped at lines 72-78 without walking their contents. Both production gate call sites therefore attest only top-level links:

- setup preflight: `tools/agents-infra/internal/infra/infra.go:134-139`
- setup postcondition / `verify local`: `tools/agents-infra/internal/infra/runtime_receipt.go:158-174`

Two source-built installed-binary attacks drove the real `setup local` and `verify local` entry points:

| Negative shape | Setup | Verify | Installed result |
| --- | ---: | ---: | --- |
| `.skills/nested-probe/escape -> <absolute outside>` | 0 | 0 | `.agents/.skills/nested-probe/escape` resolves outside `.agents` |
| `.skills/nested-probe/cycle -> ../..` | 0 | 0 | `.agents/.skills/nested-probe/cycle` resolves to the `.agents` ancestor |

This is `bypass path around the check`. The focused production tests still pass because every hostile fixture is top-level; no test places a hostile link inside an otherwise ordinary skill directory.

## Required rework

1. Recursively inspect every symlink that `syncRepo` can copy beneath source `.skills`, before destination mutation.
2. Recursively inspect every setup-owned installed skill package for escape, dangling/cyclic, self, and ancestor links in setup postcondition and `verify local`.
3. Preserve the global ownership boundary deliberately: provider skill names not managed by setup may remain out of scope, but all descendants of a setup-managed skill package are in scope.
4. Add installed-binary negatives for nested absolute escape and nested ancestor-cycle inputs, plus a contained nested relative-link narrowing control.
5. Re-run focused/full tests, vet/build/gofmt, scratch/source/local-models setup+verify, recursive-safe inspection, diff checks, and board validation.

## Evidence

- Reviewer scratch: `.temp/BUG-260817-3nk7yf/nested-escape-review/` and `.temp/BUG-260817-3nk7yf/nested-cycle-review/`.
- Source-built reviewer binary: `.temp/BUG-260817-3nk7yf/agents-infra-review`.
- Existing focused production tests passed with the bypass present: `go test . -count=1 -run 'TestInstalledBinarySetupLocal(ScrubsLiteralSourceDirAndAvoidsRepoSkillCycle|RefusesUnsafeSourceSkillLinksBeforeDestinationMutation)|TestInstalledBinaryVerifyLocal(RefusesUnsafeManagedSkillLinkDrift|InspectsEveryManagedSkillSurface)'`.
- No source code was modified by the reviewer; only this verdict artifact and the required logbook entry were added.
