# Change Request CR-TASK-260830-y6infr-1

Immutable snapshot of the agents-infra consumer implementation for
TASK-260830-y6infr. Revision 1.

| Field | Value |
| --- | --- |
| Repository | `relux-agents-infra` |
| Branch | `task-board/story/STORY-260830-3imc9f` |
| Base commit | `4270549dd17c010599e2083bf3ec7672af60ea29` |
| Candidate tree | `f9a32d68dd903c593df14adcee41cde58276f8ea` |
| Patch | `CR-TASK-260830-y6infr-1.patch` (`git diff --binary <base>`, intent-to-add applied) |
| Patch bytes | 126159 |
| Patch SHA-256 | `105c3c4d3bb8b640f5d37e7f18aebcb478401d667455e49ee27efc71f14386f7` |
| `git diff --check` | exit 0 |
| Scope | 25 files, +2324 / -43 |

## Upstream pin under review

`github.com/relux-works/skill-agents-management v0.5.1-0.20260830114459-046baef11790`
— immutable pseudo-version of the independently accepted commit
`046baef11790e93a7967230eec760c4563432270` (tree `454e2aae8f975f7991edd34cb8cd96171afde790`,
reviewed patch SHA-256 `471ebb6ce1b3805bcc8213248ebf0f8ac3c67c7ee215eb279c2f4490c76d96cb`).
No `replace` directive, no copied interface. `go mod verify`: all modules verified.

## Reviewer entry points

- Trusted assembly: `internal/infra/agents_management_registry.go` → `BuildPiPluginGraph`
- Concrete observation adapter: `internal/infra/agents_management_observer.go`
- Production child-launch/result call site: `internal/infra/agents_management_process_a.go` → `BuildAndRunPiTurn`
- Schema-1 producer (Process A): `internal/infra/pi_turn_result.go` → `RunPiTurnProcessA`
- Schema-1 CLI selection: `main.go` → `selectsPiTurnResultSchema1` / `runPiTurnSchema1CLI`
- Boundary guards: `internal/infra/agents_management_boundary_test.go`

## Evidence

Validation exit codes, the narrowing-mutant table, and the explicit
not-run list are in `TASK-260830-y6infr_results.md`. The raw mutant log is
`TASK-260830-y6infr_mutants.log`; the reproducible gate is
`TASK-260830-y6infr_mutants.sh`.

Every gate ran against this exact tree. No live runtime, model, process, service,
socket, network endpoint, status command, or user configuration was contacted.

## Not covered by this revision

- The stable upstream semver tag and the final pin (AC7 assigns them to the
  dedicated delivery task).
- CLI activation of `BuildAndRunPiTurn`; it ships as a library seam pending that tag.
- Independent review, PR, and merge gates, which this handoff requests.
