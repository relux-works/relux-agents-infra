# TASK-260830-s5ro4e revision 4 validation (corrected)

Date: 2026-08-30

Scope: rework answering `CR-TASK-260830-s5ro4e-3` verdict F1 and F2.
No migration, release, tag, Homebrew mutation, install, or real installed-runtime
setup was performed by this run.

## Correction to the previous revision-4 attempt

The prior attempt at this revision (`RUN-260830-288c90`) terminated at exit 1 on
a provider usage limit. It had attached a plan and a validation resource
claiming both findings were fixed. Re-attacking that plan showed **both findings
survived it**, so this document replaces those claims.

| Claimed in prior attempt | Actual state when re-attacked |
| --- | --- |
| "Guard refuses identical partial bytes ... independent of caller shell context" | The *orchestration* refused, but the reader beneath it returned 0 after three failed `brew` calls. All five probes stubbed `snapshot_lldb_surface`, so none of them tested the reader. |
| "Recovery baseline is exact ... `4270549dd17c010599e2083bf3ec7672af60ea29`" | Still a hard-coded literal, which the spawn brief explicitly ruled out. Rebinding one literal to another does not fix the defect class. |

## F1 — the guard admits a failed read

Two distinct fail-open mechanisms were hidden under the stub:

1. `errexit` is suppressed inside a **braced** function body invoked as an `if`
   condition — exactly how `capture_lldb_surface` calls the reader. Reproduced:
   the same failing read returns 1 when called directly and 0 through `if`.
2. `var=$(cmd)` never trips `errexit` in zsh at all, so status checks written
   that way were inert in one of the two shells in use.

Additionally, `brew list --versions llvm` exits 1 **both** when llvm is absent
and when the query fails, so it cannot separate a measured absence from a failed
read; and deriving target paths from an unread prefix collapsed them to
`/bin/lldb-mcp` and `/bin/lldb`, emitting `ABSENT` lines for paths that were
never the intended targets — a failed read wearing the shape of a measured
absence.

Fix: reader and capture functions have subshell bodies with an explicit status
test on every read (`if ! var="$(...)"; then exit 70; fi`); llvm state is read
via `brew list --formula`, authoritative on exit 0 and refusing otherwise;
llvm-prefixed targets are probed only after the prefix was actually read.

## F2 — the recovery baseline

Fix: `materialize_recovery_source` parses the commit the **saved** executable
reports, resolves it in the owning repository, and creates the detached worktree
at exactly that OID. Baseline and binary are the same pair by construction.
Unparseable version output or an unresolvable commit refuses and produces no
baseline — the plan reports `unknown` and stops rather than guessing.

Zero baseline literals remain in the plan's executable blocks, down from 2.

## Negative evidence

The validator extracts the guard and identity functions **verbatim from the plan
document** and takes the plan path as an argument, so revision 4 and revision 5
are attacked by the identical suite. Every negative probe below is red against
revision 4 and green against revision 5.

| Probe | rev4 | rev5 | Required |
| --- | ---: | ---: | --- |
| Real reader, `brew --prefix llvm` / `list --formula` fails | `0` | `70` | refuse |
| Same failure invoked via an `if` condition | `0` | `70` | refuse |
| `brew` entirely absent | `0` | `70` | refuse |
| `brew --prefix llvm` fails | `0` | `70` | refuse |
| `brew list --versions llvm` fails | `0` | `70` | refuse |
| llvm genuinely not installed (measured absence) | n/a | `0` | **pass**, records `LLVM_NOT_INSTALLED`, no collapsed `/bin` probe |
| End-to-end production guard, unreadable surface | `0` + `passed` created + installer ran | `1`, no `passed`, installer never invoked | refuse |
| End-to-end production guard, real host | `0` | `0` | pass |
| Guard: before/after snapshots differ | `1` | `1` | refuse |
| Guard: installer exits 9 | `1` | `1` | refuse |
| Baseline derived from installed artifact | absent | `0` | pass, derived OID == reported commit |
| Unparseable reported commit | n/a | `1` | refuse, no worktree |
| Unresolvable reported commit | n/a | `1` | refuse, no worktree |
| Baseline literals in executable blocks | `2` | `0` | none |
| **Full suite** | `1` | `0` | — |

Run under **both shells**, same verdict — confirming the fail-open was real in
`bash`, the shell the plan's code fences declare, not a zsh artifact:

| Shell | rev4 | rev5 |
| --- | ---: | ---: |
| `zsh` | 1 | 0 |
| `bash` | 1 | 0 |

The measured-absence probe is the narrowing check: the fix refuses failed reads
without refusing legitimate absences, so it is not a delete-only mutant.

## Real command exits

| Command/gate | Exit | Result |
| --- | ---: | --- |
| `go build ./...` (tools/agents-infra) | 0 | Compiles. |
| `go vet ./...` (tools/agents-infra) | 0 | Clean. |
| `go test ./internal/infra -run 'TestSetupGlobal\|TestVerifyInstalledRuntimeRefusesIncorrectGlobalPiInfraTarget' -count=1` | 0 | 4.303s. |
| `go test ./internal/attachments ./internal/modelharness -count=1` | 0 | 1.460s / 1.281s. |
| `git diff --check` | 0 | No whitespace defects. |
| `validate-revision5.zsh` against rev5 plan | 0 | All contracts hold. |
| `validate-revision5.bash` against rev5 plan | 0 | Same under bash. |
| `validate-revision5.zsh` against rev4 plan | 1 | **Expected red.** Negative probes fail against the previous revision. |
| `validate-revision5.bash` against rev4 plan | 1 | **Expected red.** Same under bash. |
| Real-host reader snapshot | 0 | `llvm 23.1.0`, prefix `/opt/homebrew/opt/llvm`, both llvm targets measured absent. |

### Not run, with reason

`go test ./...` over the whole `tools/agents-infra` module was not re-run. The
root package and `./internal/infra` dominate at ~182s and ~312s, and this
revision's delta is markdown-only (`.research/260830_...md` and `LOGBOOK.md`),
changing zero Go files — the full suite cannot observe it. The targeted infra
tests that previous revisions used as the load-bearing gate were rerun and are
green above.

## Deliverables

- `.research/260830_agents-management-lockstep-release-and-rollback.md`
- `LOGBOOK.md` entries `1730` and `1731`; entries `1528` and `1529` marked SUPERSEDED
- Board plan resource (byte-identical to the source file)
- `TASK-260830-s5ro4e_change-request_rev4.patch` + validation log
- Validators: `validate-revision5.zsh`, `validate-revision5.bash`

All operational rollback procedures remain explicitly **UNTESTED** until a
release operator rehearses them on a disposable target environment.
