# BUG-260713-3d9fbz implementation and validation evidence

## Outcome

- `infra.Setup` now resolves the setup-local source and project roots before destination validation or mutation.
- It compares filesystem identity while walking project ancestry, refusing source/project equality and source containment across relative paths, trailing separators, symlinks, and macOS case aliases.
- Refusals name both resolved paths, explain the recursive `PROJECT/.agents` hazard, and direct users to an external source (plus `--no-sync` for config-only changes).
- Resolved identities are used for the safety decision without changing caller-visible path spellings used by valid installations.
- README, source-managed `SKILL.md`, and `LOGBOOK.md` document the contract and the macOS compatibility decision.

## Production negative and filesystem evidence

Production call site: installed binary `setup local` -> `runSetup` -> `infra.Setup` -> `resolveLocalSetupLayout`, before `ValidateCanonicalProjectConfiguration`, project-config preparation, receipt invalidation, `syncRepo`, or link/render writes.

`go test . -run '^TestInstalledBinarySetupLocalRefusesRecursiveSourceBeforeFilesystemMutation$' -count=1 -v`

- Exit: 0.
- Passed exact equality, relative paths, trailing separators, symlink alias, source-containing-project, and macOS case-insensitive equality; the Darwin case ran and was not skipped.
- Every case asserted absence of `PROJECT/.agents`, `.claude`, `.codex`, `.local`, and root `AGENTS.md` after refusal.
- The test comment names the production dispatch and explains that removing/narrowing the guard enters `syncRepo`; the no-mutation assertions therefore kill that mutant.

`go test ./internal/infra -run 'TestSetupRefusesSourceDirContainingItsOwnDestination|TestResolveSourceDirRejectsCandidateContainingTheDestination' -count=1`

- Exit: 0.

Valid external-source controls:

- `go test . -run 'TestInstalledBinarySetupLocal(RefusesRecursiveSourceBeforeFilesystemMutation|ScrubsLiteralSourceDirAndAvoidsRepoSkillCycle)$' -count=1` — exit 0.
- Focused rerun of the pre-existing setup/install compatibility cases — exit 0.

## Full validation

- `go test ./... -count=1` — exit 0. Main package 127.731s; internal/infra 230.043s; all packages green.
- `go vet ./...` — exit 0.
- `go build ./...` — exit 0.
- Focused README/SKILL operator documentation tests — exit 0.
- Scoped `git diff --check` across all seven task-owned files — exit 0.
- Diff scope: 7 files, 242 insertions, 3 deletions.
- No repository-specific linter, `golangci-lint`, or `staticcheck` was available; `go vet ./...` was the applicable lint/static-analysis gate.
- `task-board validate` — exit 0, while printing 83 pre-existing `MISSING_ACTIVITY` findings for unrelated historical elements; none names `BUG-260713-3d9fbz`.

## Honest non-green iterations

- The first combined focused command exited 1 because the older lexical source resolver produced its legacy destination-only error before the new canonical production gate. The resolver check was narrowed to global setup; local setup now reaches the stronger source-versus-project gate. Focused reruns exited 0.
- The first full `go test ./... -count=1` exited 1 because using resolved `/private/var/...` spelling for installation changed pre-existing `/var/...` symlink/receipt bytes. The implementation was corrected to use resolved identity only for the safety decision while preserving the caller's cleaned absolute spelling. Focused compatibility reruns and the subsequent full suite exited 0.

## Worktree base and unrelated state

- Refreshed `origin/main`; upstream OID and `merge-base(HEAD, origin/main)` both resolve to `08a094c4d9942586bd9492cfb36bd7eccfbdc378`.
- Story branch remained `task-board/story/STORY-260713-3et7b3`; no switch, rebase, merge, or integration was performed.
- `.task-board/.board-write-ledger.json` was already modified at turn start and was left untouched as unrelated pre-existing state.
