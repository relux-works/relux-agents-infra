# TASK-260824-3rl3ws revision 3 rework validation

Validated at 2026-08-24 18:37 MSK in Story worktree commit
`cf21665dde35274cc14e66e26a93574e0c18c15c`.

## Reviewer findings closed

- G1: Contract Sections 3, 5, and 7 now preserve `resolved.profile` semantics
  per provider. Claude uses `not_applicable`; Codex uses explicit CLI
  provenance when a legacy invocation supplies a profile and `native` when
  Codex configuration decides. A canonical Codex alias rejects `--profile`
  and `-c profile=...` because hosted targets have no profile coordinate.
- G1: Codex `--model`, `--model-reasoning-effort`, `-c model=...`, and
  `-c model_reasoning_effort=...` are explicitly part of the alias identity
  lock: exact repeats pass and divergent values fail.
- G1 production anchors were inspected directly:
  `tools/agents-infra/internal/infra/primary_session_launch_plan.go:108`
  defines `native` versus `not_applicable`, lines 277-281 preserve Codex
  profile provenance, and lines 459-460 mark Claude profile as
  `not_applicable`. `tools/agents-infra/internal/infra/codex_launch.go:798`
  records `-c model=` and `-c model_reasoning_effort=` selections.
- G2: The dead `grep -Fv | grep -Fq` loop was removed. The validator first
  proves each marker exists, creates a changed contract with the complete
  marker-bearing clause removed, and requires the real `check_contract`
  function to return non-zero.
- G2: `check_contract` now propagates each failed marker check explicitly with
  `|| return 1`. This is required because a shell function invoked as an `if`
  condition does not stop at an intermediate failed command under `set -e`.

## Contract, traceability, and board shape

- Revision 1 findings F1-F5 remain closed; Sections 1-4, 6, and 8 are
  unchanged except for the focused Codex selector clarification in Section 3.
- The source contract and the implementation, rollout, and deployment
  preconditions are byte-identical at SHA-256
  `c0db2515ed3a1055d34d0f9384889deea51268b8f06b8ad5dd402d3bb498717b`.
- The Story remains the smallest proportional four-task decomposition:
  contract, implementation, explicit operator rollout, and deployment smoke.
  No diagram, research task, or additional quality-gate task is justified.
- Dependencies remain contract -> implementation -> rollout -> deployment;
  deployment also keeps its explicit implementation prerequisite.

## Evidence executed in revision 3

- `validate-contract.sh`: passed. Nine real clause-removal mutants were
  rejected: open Codex reasoning, composite Pi model identity, endpoint
  equality, non-magic Pi provider labels, legacy-plan output isolation,
  config-read-only runtime behavior, actionable JSON remediation, provider-
  specific Codex profile provenance, and Codex alias profile refusal. Log:
  `.temp/TASK-260824-3rl3ws/contract-validation-06.log`.
- The validator was also challenged with an absent marker during diagnosis;
  the corrected precondition rejects it before claiming mutation evidence.
- `git diff --check`: passed. Log:
  `.temp/TASK-260824-3rl3ws/git-diff-check-03.log`.
- `task-board validate`: run after publishing the four byte-identical contract
  resources; log attached separately as revision-3 validation evidence.

No Go implementation changed in revision 3. This run did not rerun the full Go
suite. It accepts the immediately preceding reviewer run
`RUN-260824-444ecb` evidence: `go build ./...`, `go vet ./...`, and
`go test ./... -count=1` all passed against the same code commit. Revision 3
reran the checks capable of failing on its actual contract and logbook delta.
