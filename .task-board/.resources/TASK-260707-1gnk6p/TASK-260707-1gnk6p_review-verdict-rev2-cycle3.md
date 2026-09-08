# TASK-260707-1gnk6p Review Verdict — CR Revision 2, Cycle 3

## Verdict

**Changes requested.** Route the task to `to-dev` and publish a new Change
Request revision whose repository delta contains only this task's LLDB MCP
bootstrap scope.

The LLDB rework itself is acceptable under the exercised paths. The blocking
finding is that CR revision 2 is not a task-scoped candidate: it also carries
the separate, closed `TASK-260824-1qm60c` source-managed Codex configuration
work that its own board note says will be replayed through a dedicated delivery
Story.

## Blocking Finding: Candidate Scope Includes A Closed Sibling Delta

The reviewed candidate is:

- base OID: `cf21665dde35274cc14e66e26a93574e0c18c15c`
- candidate tree OID: `1edd36ea14ea4adacb53cb6917fbb0eee8a5c055`
- patch SHA-256: `83af40133816266867fcb920ebf7bedb80adf1ce5703b5582594b6fa89e5d433`
- repository delta: 10 paths, 689 insertions/deletions in aggregate

`task-board worktree status STORY-260508-1ajnbe` reports both this ready CR and
accepted-but-uncheckpointed sibling CR revision 2 for `TASK-260824-1qm60c`.
The sibling patch has eight changed paths. Every one of those paths also appears
in this task's ten-path candidate:

- `.configs/codex-config.toml`
- `LOGBOOK.md`
- `README.md`
- `SKILL.md`
- `tools/agents-infra/internal/infra/codex_config.go`
- `tools/agents-infra/internal/infra/infra.go`
- `tools/agents-infra/internal/infra/infra_test.go`
- `tools/agents-infra/setup_test.go`

Five of those paths are wholly unrelated to LLDB setup and implement/test the
withdrawn Codex fast profile and installed-config preservation. The three shared
documentation/evidence paths contain both the sibling work and the LLDB changes.
The sibling task is already `closed`, and its own note explicitly says its
accepted patch was not checkpointed and will be replayed through a dedicated
delivery Story. Accepting this CR would therefore attest and potentially land
that closed sibling scope through `TASK-260707-1gnk6p`, outside this task's
description and acceptance criteria.

Required rework: remove the sibling delta from the managed worktree/candidate
and republish revision 3 from the same task with only the LLDB setup changes.
Preserve the LLDB-specific hunks in `scripts/setup.sh`,
`tools/agents-infra/bootstrap_lldb_mcp_test.go`, `README.md`, `SKILL.md`, and
`LOGBOOK.md`; do not retain the Codex fast-profile/config-preservation code or
tests in this task's candidate. The sibling patch remains recoverable from its
own accepted CR resource.

## Gate Attack And Validation

The production call site under review is root `./setup.sh`, which execs
`scripts/setup.sh`; that script invokes `install_lldb_mcp` directly at line 263.
I reran the following uncached production-entry tests against the exact working
tree that realizes candidate tree `1edd36e`:

- missing Homebrew and absent `lldb-mcp` refuses rather than reaching success;
- two setup runs issue one `brew install llvm` and preserve the managed wrapper;
- a marker-bearing wrapper with the wrong helper target is rejected/repaired;
- a self-minted wrapper containing the expected `exec` after an earlier
  `exit 0` is rejected/repaired.

Command:

```text
go test . -run '^TestBootstrapSetup(RefusesMissingHomebrewAndLLDBMCP|InstallsLLDBMCPOnceAcrossTwoRealRuns|RepairsManagedLLDBMCPWrapperWithWrongTarget|RepairsManagedLLDBMCPWrapperWithUnreachableExpectedTarget)$' -count=1 -v
```

Result: PASS, 4/4 tests. This attacks the `absent evidence treated as
satisfied` and `forged or self-minted evidence` shapes through the production
entry point rather than calling the helper directly. `zsh -n scripts/setup.sh`
and exact-delta `git diff --check` also passed.

I accepted, rather than redundantly reran, the tree-bound producer log for the
full suite: `go test ./... -count=1` and `go vet ./...` both exited 0. The patch
resource digest independently matches the digest named in the review prompt.

Evidence logs from this reviewer run are under
`.temp/TASK-260707-1gnk6p-review/`, notably `bootstrap-attack-01.log`,
`static-validation-01.log`, `cr2-validation.log`, and the downloaded candidate
and sibling patch resources. No repository source file was modified during
review.
