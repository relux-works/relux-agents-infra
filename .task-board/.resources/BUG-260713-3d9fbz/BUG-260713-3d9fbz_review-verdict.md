ACCEPTED (CR-BUG-260713-3d9fbz-1, revision 1).

Scope reviewed: `git diff HEAD` (infra.go, source_dir.go, source_dir_test.go,
installed_binary_setup_test.go, README.md, SKILL.md, LOGBOOK.md). The other
files in the CR's changed-path list (dange launchers, hosted-primary-session
validation, canonical target tests) belong to already-committed prior commits
(fd29856, ce3db14) on this branch and are out of scope for this bug.

## What the fix does

`infra.Setup`, for `ModeLocal` with a non-empty `SourceDir`, now calls
`resolveLocalSetupLayout` as the very first step, before
`ValidateCanonicalProjectConfiguration`, `preparePrimarySessionSetup`,
`requireLayoutSourceDir`, or any `invalidateRuntimeReceipt`/`syncRepo`
mutation. It resolves both source and project roots through `filepath.Abs` +
`filepath.EvalSymlinks`, then walks the project ancestry comparing
`os.SameFile` identity (`directoryContainsByIdentity`) against the resolved
source. Equality or source-contains-project is refused with an error naming
both resolved paths. Because the guard lives inside `infra.Setup` itself, it
applies uniformly regardless of whether `SourceDir` originated from
`--source-dir`, `AGENTS_INFRA_SOURCE_DIR`, install-state `repoPath`, or the
installed `~/.agents` fallback (all resolved into `layout.SourceDir` by
`ResolveSourceDir` before `Setup` is invoked).

## Design decision verified against docs

`--no-sync` does NOT bypass the guard. An explicit self/ancestor
`--source-dir` is refused even with `--no-sync`; README/SKILL/LOGBOOK
explicitly document this as deliberate ("an apparently harmless invocation
cannot later become a recursive sync when its flags change"), and
`TestSetupRefusesSourceDirContainingItsOwnDestination` asserts the refusal
fires with `NoSync: true`. This replaces the bug's suggested `--no-sync`
workaround with a requirement to point `--source-dir` at an external tree.

## Independent verification performed

1. Reproduced the original bug against the pre-fix base commit (`08a094c`)
   with a symlinked `--source-dir` pointing at the project itself: base
   commit synced and created `project/.agents` (confirms the bug is real and
   that the pre-existing lexical `dirContains` check in
   `requireLayoutSourceDir`/`evaluateSourceDirCandidate` already caught
   literal-equal/relative/trailing-slash forms but not symlink or
   case-alias forms).
2. Ran the new
   `TestInstalledBinarySetupLocalRefusesRecursiveSourceBeforeFilesystemMutation`
   matrix (equal, relative paths, trailing separators, symlink alias,
   source-contains-project, case-insensitive equality on darwin) against the
   candidate: all 6 subtests pass, driving the real installed binary through
   `main()->runSetup->infra.Setup`, asserting the exact refusal message names
   both resolved paths, and asserting no `.agents`/`.claude`/`.codex`/`.local`/
   `AGENTS.md` path exists afterward.
3. Mutation-killed the guard myself: commented out the
   `resolveLocalSetupLayout` call in `infra.Setup` and reran the same test —
   all 6 subtests failed red (the symlink and case-insensitive cases actually
   completed a full sync into the project, reproducing the original bug),
   confirming this is a real production-entrypoint-killing negative, not a
   vacuous/helper-only test.
4. Manually verified the happy path still works: `setup local` against a
   fresh existing project dir with a valid external `--source-dir` completes
   normally end to end.
5. Confirmed a pre-existing, unrelated behavior (`setup local` requires the
   project directory to already exist) is unchanged by this fix — the same
   failure reproduces identically on the base commit, so it is not a
   regression introduced here.
6. `go build ./...`, `go vet ./...` clean. Full `go test ./...` green
   (`agents-infra` 131.7s, `internal/infra` 226.0s, others cached/no tests).

No findings. AC fully met: refusal fires before any filesystem mutation,
survives relative paths/trailing separators/symlinks/macOS case-insensitive
equality, the regression test drives the production entrypoint and is red
when the guard is removed, and no nested `.agents` is created on the refused
path.
