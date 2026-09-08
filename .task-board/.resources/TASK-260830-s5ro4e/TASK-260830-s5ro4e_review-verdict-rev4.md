# TASK-260830-s5ro4e review verdict — revision 4

Verdict: **changes requested**. Route to `analysis` for a corrected plan and a
new Change Request revision.

Reviewed candidate: `CR-TASK-260830-s5ro4e-4` revision 4, base
`5c9b4e4f7a88e1eb937b80851af522e4fa4b066f`, candidate tree
`7b96eb3269f1b3c1262ef8334a7a4f520e934d63`, patch SHA-256
`d364efd02d5f2db067ca3a1aa42d1263b9e8c068200fc44ddfc24df4cbc730be` (matches the
handed value). Reviewer run `RUN-260830-95b876`.

Revision 4 fixes both revision-3 findings. I reproduced that independently, with
my own probes rather than the producer's validator — see "Confirmed fixes". The
rejection is a new finding of the same shape the last three rounds were about,
one installer further along: the task-board half of the migration has never had
the boundary analysis the agents-infra half received.

## Findings

### F1 — High: `W2`/`W5` omit the project-management installer's `install_skills` stage, and the Step-2/Step-5 rollback cannot restore it

The plan enumerates the Step-2 artifact sequence as "the first atomic
replacement of `task-board`; then `tb-sessiond`, TUI, both board launchers,
installed skill, roles and install state"
(`.research/260830_agents-management-lockstep-release-and-rollback.md:722-728`,
inherited verbatim by `W5` at `:860-862` and by the window-table row at `:952`).
That enumeration is incomplete.

Production `skill-project-management` (`f1319eff`) `setup_main` runs, in order:
`install_go`, three builds, five `install_binary` calls, `register_skill`,
`install_roles`, `write_install_state`, `check_path`, `configure_project`,
`check_agents_infra_compose`, **`install_skills`**, `verify`
(`scripts/setup.sh:891-931`). `install_skills` is unconditional — it is not
behind `--configure-project` — and for each of eight unrelated skills
(`SKILL_REPOS`, `scripts/setup.sh:821-831`) it:

- `git clone`s from a remote GitHub/SSH URL when `~/.agents/skills/<skill>` is
  absent or a symlink, after having already `rm -f`'d that symlink
  (`:843-858`); a failed clone hits `continue`, leaving the skill **removed**;
- runs `scrub_git_metadata`, which `rm -rf`s `.git`, `.gitignore`,
  `.gitattributes`, `.gitmodules` inside the installed tree (`:861`, `:308-328`);
- `rm -rf`s `~/.claude/skills/<skill>` and `~/.codex/skills/<skill>` when they
  are real directories, then re-links them (`:873-886`).

None of `~/.agents/skills/<the other eight>`, `~/.claude/skills/*` or
`~/.codex/skills/*` beyond `project-management` appears anywhere in the plan.
Grep over the whole document for `install_skills`, `SKILL_REPOS`, `git clone`
or any of the eight skill names returns nothing. They are not in the
`current-pair`/`bridge-pair` snapshot (`:449-490`, which saves seven binaries,
the project-management skill, nine named roles and `install.json`) and not in
`restore_project_management_surface` (`:585-608`). So a Step-2 or Step-5 install
mutates an artifact class that the rollback provably cannot return to its
previous state, and the plan asserts a complete artifact list that excludes it.

Measured host state (read-only): all eight are real directories under
`~/.agents/skills` with symlinks under `~/.claude/skills` and `~/.codex/skills`,
and their git metadata is already absent. So on *this* host today the
non-mutating branch is selected and the residual delta is sixteen recreated
symlinks. That is a measurement, not the plan's claim — the plan performed no
measurement of this surface at all, which is exactly the shape rejected in
revision 1 (`bypass path around the check` / absence inferred rather than read).
The destructive branch is reachable by a state the installer itself treats as
expected: if any of the eight is a symlink or missing when Step 2 runs, the
release installer performs a network clone, and a clone failure in a
non-interactive release shell without SSH agent leaves that skill deleted with a
dangling `~/.claude`/`~/.codex` link and no rollback path.

This is the direct analogue of the accepted revision-2 finding about
`install_lldb_mcp`. That one was resolved by mandating
`AGENTS_INFRA_SKIP_LLDB_MCP=1` plus a before/after identity guard. The
project-management installer received no equivalent analysis, no skip, and no
guard.

Required rework:

1. Extend the `W2`/`W5` artifact enumeration and the window table to name the
   `install_skills` stage explicitly, including the eight skill trees and both
   link trees, and state where in the sequence it runs relative to `verify`.
2. Measure that surface the way the LLDB surface was measured — which branch the
   target host selects, for each of the eight — and record it as a measured fact
   with a failed read distinguished from an absence.
3. Either add a before/after identity guard for those paths on the same
   fail-closed construction as `snapshot_lldb_surface`, or add them to the
   snapshot and to `restore_project_management_surface`, or establish a
   supported way to skip the stage during a lockstep release. Do not leave the
   third option implicit.
4. State the in-flight behaviour and recovery authority for an interruption
   inside `install_skills`, as Steps 3/6 do for their installer.

### F2 — Medium: Steps 2 and 5 have no install command, so the mutation scope of `W2`/`W5` is operator-chosen

Steps 3 and 6 pin the exact install invocation, the mandatory environment flag
and the guard (`:763-770`, `:903-910`). Steps 2 and 5 contain no install command
at all — only prose about canaries. `scripts/setup.sh` accepts
`--configure-project`, `--bootstrap-local-agents`, `--project-dir`, `--mode`,
`--remote-*` (`scripts/setup.sh:56-128`), and `--configure-project` additionally
enables `configure_project`, `check_agents_infra_compose` and
`bootstrap_local_agents` inside the same window. A window whose artifact set
depends on flags the plan never fixes is not a named window.

Required rework: give Steps 2 and 5 the exact invocation, and state that a bare
machine-scoped install is mandatory (or name the flags that are permitted and
fold their extra artifacts into the window and the snapshot).

### F3 — Medium: the nine-role list is a maintained literal, the exact defect class revision 4 just eliminated for the baseline

The snapshot loop (`:476-480`) and `restore_project_management_surface`
(`:596-603`) both hard-code the same nine role names, while production
`install_roles` iterates `"$SKILL_DIR/.roles"/*` (`scripts/setup.sh:373-401`).
The list matches the current source today (verified: nine roles in
`skill-project-management/.roles`, nine in `~/.roles`). But `0.25.0` and
`0.25.1` are new releases of the repository that owns that directory. A role
added by either release is installed by Step 2/Step 5 and is *not* removed by
the rollback, so the "previous working state" it claims to restore contains a
role from the abandoned release. A renamed role fails the snapshot's
`test -e` closed, which is correct; an added one fails open.

Revision 4's own headline lesson is that a maintained literal in a rollback path
is a defect that fails silently (`LOGBOOK.md` entry 1730). The same argument
applies here: derive the role set from the saved installed surface at snapshot
time, and make the restore remove roles that are present after the release but
absent from the snapshot.

## Confirmed fixes (reproduced independently, not accepted from the producer)

I extracted the guard functions verbatim from plan lines 227-358 and the
identity/derivation functions from 408-445 and 496-519, and drove them with my
own probes and my own fake `brew`, under both `bash` (3.2.57) and `zsh`.
Verdict-F1 (revision 3) is genuinely dead:

| Probe | bash | zsh | Required |
| --- | ---: | ---: | --- |
| `brew --prefix` fails | 70 | 70 | refuse |
| `brew list --formula` fails | 70 | 70 | refuse |
| `brew list --versions llvm` fails | 70 | 70 | refuse |
| `brew --prefix llvm` fails | 70 | 70 | refuse |
| `brew --prefix llvm` returns empty on exit 0 | 70 | 70 | refuse |
| `brew` entirely absent from PATH | 70 | 70 | refuse |
| `stat` fails on a present target | 70 | 70 | refuse |
| reader invoked as an `if` condition, read fails | 70 | 70 | refuse |
| `capture` as `if` condition, read fails | 70 | 70 | refuse, publish nothing |
| llvm genuinely not installed | 0 | 0 | pass, `LLVM_NOT_INSTALLED`, no collapsed `/bin` probe |
| readable surface, unchanged | 0 | 0 | pass |

End-to-end `guarded_agents_infra_install`, with a stale `passed` planted in the
guard dir beforehand and a marker file proving whether the installer executed:

| Scenario | exit | `passed` | installer ran | recorded status |
| --- | ---: | --- | --- | --- |
| unreadable surface | 1 | absent (stale one cleared) | **no** | `before=70 install=not-run after=not-run cmp=not-run` |
| installer mutates the wrapper | 1 | absent | yes | `0/0/0/cmp=1` |
| installer exits 9 | 1 | absent | yes | `0/9/0/0` |
| surface becomes unreadable mid-install | 1 | absent | yes | `0/0/after=70/not-run` |
| clean happy path | 0 | present | yes | all zero |

Identical results in both shells. The measured-absence probe passing is the
narrowing check: the fix refuses failed reads without refusing legitimate
absences, so it is not a delete-only mutant.

Verdict-F2 (revision 3) is also fixed. `extract_reported_commit` parses both
real installed formats and refuses `commit=unknown`, no-commit, two matching
lines, a `-dirty` suffix and a 6-hex string. `materialize_recovery_source`
derives the worktree OID from the saved binary's reported commit (derived OID
equals the repository's resolution of it), and refuses with **no worktree
created** on unparseable and unresolvable input, in both shells.
`require_saved_binary_source_identity` refuses a dirty tree and a mismatched
reported commit. A grep of every code fence in the plan finds zero 7-40-hex
literals: the baseline is derived, not pinned. Both installed reported commits
(`063197b1`, `4270549`) resolve read-only in their owning repositories today.

## Other independent checks that passed

- The plan file is byte-identical to the board plan resource
  (`5c04797e…`), and the working tree matches the candidate tree for both
  changed paths. `git diff --check` over the exact range exits 0.
- The delta touches only the plan and `LOGBOOK.md` — zero Go, shell, or module
  files — so the producer's decision not to re-run the full `tools/agents-infra`
  suite for this revision is sound; the attached CR validation log records
  `go test ./...` exit 0 and `go vet ./...` exit 0 from the prior revision, and
  the targeted `internal/infra` tests green here.
- The `LOGBOOK.md` correction has the required shape: `1245` and `1428` carry the
  corrected 25/36 census, 20-package graph and 555 symbols; `1528` and `1529` are
  explicitly marked SUPERSEDED with the reason, and `1730`/`1731` state the
  corrected finding. The false claim is not left standing as a current entry.
- The four consumption instruments and the `v0.6.0` removal-manifest gate are
  unchanged from revision 3 (diffed: the measurement section carries only the
  compatibility-matrix row edit). They were independently reproduced by the
  revision-2 review; I did not re-derive the 555-symbol count in this run and do
  not claim to have.
- `AGENTS_INFRA_SKIP_LLDB_MCP=1` still returns before any Homebrew or wrapper
  mutation in production `relux-agents-infra/scripts/setup.sh:93-97`, and
  `install_lldb_mcp` is still called before the managed-binary replacements
  (`:258-259`). `install_go` (`:75-91`) returns early when `go` is present, which
  it is on the target host; the plan does not name it, but it selects the
  non-mutating branch here.
- All operational rollbacks remain explicitly labelled **UNTESTED**.
- No release, tag, install, migration, Homebrew change or installed-runtime
  mutation was performed by this review. All probes ran against a fake `brew`
  and a disposable scratch Git repository under `.temp/`; no process belonging to
  another run was signalled.

## Review diagnostics

- My first extraction of the identity block took plan lines 406-432 and cut a
  function in half. `bash` aborted on the syntax error and produced no output at
  all while `zsh` still printed results, because zsh had already defined the
  earlier function before reaching EOF. I treated the empty bash output as a
  failed read, not as a refusal by the plan, found the truncation, re-extracted
  the complete function bodies (408-445 and 496-519) and re-ran both shells.
- One probe initially set `FAKE_BREW_MODE` without exporting it, so the fake
  `brew` subprocess never saw the failure mode and the reader correctly returned
  0. That was my harness bug, not a fail-open; it is excluded from the tables
  above, which use the corrected exported form.

## Evidence

- `TASK-260830-s5ro4e_review-evidence-rev4.log` (attached): all probe output in
  both shells, real version strings, and the F1 source citations.
- Scratch: `.temp/TASK-260830-s5ro4e-review2/` (verbatim extractions, fake brew,
  probe scripts).
