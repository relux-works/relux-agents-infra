# BUG-260817-3nk7yf — Reviewer Verdict

## Verdict

**Changes requested → `to-dev`.**

The intended production path fixes the reported root literal directory and the canonical `relux-agents-infra` ancestor link, and all positive gates pass. The broader acceptance criterion that setup emit no escaping or self-referential skill symlink is still bypassable.

## Blocking finding — source skill symlinks bypass containment

Production `setup local` copies source symlinks verbatim at `tools/agents-infra/internal/infra/infra.go:307-312`. The new repository-skill materialization protects only the canonical `relux-agents-infra` link. Neither setup's postcondition nor `verify local` inspects the remaining `.agents/.skills` symlinks for containment or cycles.

Two installed-binary attacks against the real `setup local` entry point reproduced the gap:

| Negative shape | Setup | Verify | Protected invariant defeated |
| --- | ---: | ---: | --- |
| `.skills/escape-probe -> <outside-runtime>` | 0 | 0 | `find -L` reached `OUTSIDE_MARKER` outside the installed runtime |
| `.skills/cycle-probe -> ..` | 0 | 0 | Installed `.agents/.skills/cycle-probe` resolves to its `.agents` ancestor |

This is `bypass path around the check` and `positive-path-only evidence`. `TestInstalledBinarySetupLocalScrubsLiteralSourceDirAndAvoidsRepoSkillCycle` calls `assertContainedAcyclicSymlinks`, but its fixture contains only ordinary skill directories. It never injects the escaping or ancestor-cycle inputs that the production copier accepts.

## Required rework

1. Enforce a fail-closed skill-link containment/acyclicity invariant at the production setup boundary before a runtime receipt is minted. Prefer rejecting invalid source `.skills` links before destination mutation; safe omission is acceptable only if explicitly specified and tested.
2. Make `verify local` reject installed-runtime drift containing an escaping or ancestor/self-cycle skill link across the managed skill surfaces.
3. Add installed-binary production-entry negatives for both `.skills/escape -> outside` and `.skills/cycle -> ..`; require refusal or specified safe omission. Keep the clean canonical `relux-agents-infra` case as the narrowing control.
4. Re-run focused/full tests, vet/build/gofmt, scratch setup/verify/find, source/local-models refresh/verify, diff checks, and board validation.

## Positive evidence retained

- Root and nested literal `$AGENTS_INFRA_SOURCE_DIR` artifacts are absent after production setup; nested `.temp` is scrubbed.
- Canonical `.agents/skills/relux-agents-infra` resolves to the contained `.agents/.skills/relux-agents-infra` package on scratch and `local-models`.
- Focused tests pass.
- `go test ./... -count=1`, `go vet ./...`, `go build ./...`, focused `gofmt -d`, and `git diff --check` pass.
- Source repo and `/Users/alexis/src/local-models` both pass `agents-infra verify local`; `local-models` has no literal artifact, recursive `find -L` exits 0, and `pi-infra` exists.
- No staged diff; `task-board validate` reports no issues.

## Evidence files

Reviewer probes and logs are under `.temp/BUG-260817-3nk7yf/`, notably `escape-setup-01.log`, `escape-verify-01.log`, `escape-find-01.log`, `cycle-setup-01.log`, `cycle-verify-01.log`, `cycle-find-01.log`, `full-tests-01.log`, `local-models-verify-01.log`, and `board-validate-01.log`.
