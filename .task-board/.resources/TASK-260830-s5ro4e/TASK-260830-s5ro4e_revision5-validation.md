# TASK-260830-s5ro4e — Change Request revision 5 validation

Board CR revision 5 == plan document revision 6. The document keeps its own
counter, which runs one ahead of the board's; the plan states the mapping in its
header.

Base `5c9b4e4f7a88e1eb937b80851af522e4fa4b066f`, candidate tree
`b1bc6fd1ef94ace92ea21077ca81d9c9a608d8f2`, patch SHA-256
`fec721791ef587fac2ec7db682ddc7e29d92121b4b474d1641062c72499fa70e`, plan SHA-256
`083a602386b78ed8b6e5542b5aa4422e47e603d34cffd9a2b0c2bb9b28cddbaa`.

Delta: `.research/260830_agents-management-lockstep-release-and-rollback.md`
(new file, 1990 lines) and `LOGBOOK.md` (one new entry). No Go, shell, module or
configuration file changed. No release, tag, install, migration, Homebrew change
or installed-runtime mutation was performed.

## What the revision-4 verdict asked for, and what was done instead

The verdict raised three findings and one instruction: do not fix them one at a
time, enumerate every place the plan states a literal that production derives,
and say how many. **Fourteen were found.** They are listed in the plan's
"Derived-versus-restated census". Nine of them were not named by the verdict.

| Verdict item | Resolution |
| --- | --- |
| F1 — `install_skills` omitted from the `W2`/`W5` artifact enumeration, and unrollbackable | Modelled, not declared out of scope. New section "task-board installer boundary: derived artifact surface" derives the stage sequence from the release's own `setup_main`, documents both branches of `install_skills`, records a read-only measurement of all eight `SKILL_REPOS` skills on the target host, adds a pre-window precondition that makes the network clone branch unreachable, and extends the snapshot and restore to all nine managed skills and both link trees. In-flight behaviour and recovery authority for an interruption inside the stage are stated. |
| F2 — Steps 2 and 5 have no install command | Both now carry the exact invocation: a bare `scripts/setup.sh` under an `env -u` list derived from, and kept in sync with, `classify_pm_env_override`. No flag is permitted, with the reason stated. Steps 3 and 6 additionally state that `guarded_agents_infra_install` passes no arguments and that `--with-pdf-tools` is not permitted. |
| F3 — the nine-role list is a maintained literal | Snapshot saves the whole installed role surface and records its names; the rollback restores those and removes any role the release owns that the snapshot did not contain. `derive_pm_role_names` reads `.roles/*` the way `install_roles` does. |

## The fourteen places

See the plan's census table for the full list. Summary: executable name sets (4
sites), role name sets (2 sites), the managed-skill set and its two link trees,
repository paths, the install directory, the installer stage sequence, the
agents-infra machine-scoped install state and its config-directory rule, the
LLDB guard's target set, and the installers' environment-override surface.

Four kinds of value remain written in the document — release identities, paths
the plan itself owns, paths production writes as its own literals, and one
disposition per environment-override name. Each is guarded: release identities
by `git describe --exact-match`, production-literal paths by a mirror comparison
between the saved source and the release source, and the override dispositions
by a classifier that exits 75 on an unclassified name.

## Evidence

All probes extract the functions verbatim from the plan document named on the
command line, so no probe can pass against a copy the plan does not contain. The
end-to-end suite builds a disposable HOME plus two disposable Git repositories
carrying the **real production `scripts/setup.sh`** of both installers, so every
derivation runs against the bytes production ships.

| Command | Exit | Result |
| --- | ---: | --- |
| `bash validate-revision6.sh <rev6>` | 0 | 54 probes, 0 failures |
| `zsh validate-revision6.sh <rev6>` | 0 | 54 probes, 0 failures |
| `bash validate-revision6.sh <rev5>` | 1 | Expected red: 12 probes reached, 8 failures; the other 42 report the function as absent rather than passing |
| `zsh validate-revision6.sh <rev5>` | 1 | Expected red, same shape |
| `bash probe-rev5-failopen.sh <rev5>` | 1 | Expected red: 4 failures — revision 5's restore leaves a release-added role, a release-added skill and its `~/.claude` link in place, and does not restore a rewritten dependency skill |
| `zsh probe-rev5-failopen.sh <rev5>` | 1 | Expected red, same shape |
| `bash validate-revision5.bash <rev6>` | 0 | `overall_fail=0` — the previously accepted reader/guard/identity/baseline fixes are still green |
| `zsh validate-revision5.zsh <rev6>` | 0 | `overall_fail=0` — same |
| `cd tools/agents-infra && go test ./... -count=1` | 0 | Full suite green |
| `cd tools/agents-infra && go vet ./...` | 0 | Clean |
| `cd tools/agents-infra && go build ./...` | 0 | Compiles |

### Negative probes that are red on the previous revision

| Probe | rev5 | rev6 | Narrowing check that stays green on both |
| --- | ---: | ---: | --- |
| Rollback removes a role the release added | present | absent | An existing role's content is still restored on both revisions |
| Rollback removes a skill the release added, and both its links | present ×3 | absent ×3 | Unrelated roles and skills are untouched on both |
| Rollback restores a dependency skill the release rewrote | `v2-from-release` | `v1` | — |
| Derived stage sequence contains `install_skills` | absent | present at position 13, before `verify` | — |
| LLDB guard detects a mutated `.agents-infra.bak` | identical | different | The guard still reads a clean fake Homebrew surface with exit 0 on both |
| `materialize_recovery_source` with a relative destination | 0 | 1 | Absolute destination still exits 0, and an unparseable version still refuses with no worktree created, on both |

The narrowing column matters: each fix refuses a new class without refusing the
legitimate cases, so none of them is a delete-only mutant.

### Refusal probes on the new code

`70` on unreadable trees, empty derived sets, missing/malformed/keyless install
state, an unreadable installer and an absent mirrored function; `73` on a
changed mirrored function; `74` on a symlinked, absent or non-directory
dependency skill; `75` on an unclassified environment override. Refusals hold
when the function is invoked as an `if` condition, and hold identically under
`bash` and `zsh`.

## Validator diagnostic

The first end-to-end run failed because the sandbox HOME was a relative path and
`git -C repo worktree add` resolves a relative destination against the
repository rather than the caller, while exiting zero. That was a harness bug,
but it exposed a real fail-open in `materialize_recovery_source`, which now
confirms the destination exists before publishing the resolved commit and is
covered by its own probe. Excluding that, no probe result in the tables above
was obtained from a harness that stubbed the component under attack.

## Scope discipline

No migration step, release, install or tag was performed. The installer was read,
never executed against the real runtime. All probes ran against a disposable
HOME, disposable Git repositories and a fake `brew`. Every operational rollback
remains labelled **UNTESTED**; the sandbox evidence is explicitly stated in the
plan as not a substitute for an operational rehearsal. No process belonging to
another run was signalled. No credential, token, cookie or keychain value was
read, printed or persisted.
