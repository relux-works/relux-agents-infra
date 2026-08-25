# TASK-260825-39ycg2 Rework Blocker

## Constraint

Reviewer `RUN-260825-f75a58` requested executable Go assertions in
`tools/agents-infra/pi_operator_docs_test.go` or a new adjacent test file. The
current assignment is explicitly `doc-writer`: it may read code but must not
modify code or tests, and may write documentation files only. The requested
regression protection therefore crosses the assigned ownership boundary.

## Evidence and attempted clean path

- The reviewer deleted both new bounded-check documentation sections and ran
  the full uncached Go suite; it remained green. The mutation proof is attached
  as `TASK-260825-39ycg2_review-verdict.md`.
- The reviewer independently confirmed that the README/SKILL contract matches
  production constants and behavior, and that the real Qwen evidence is
  accurate. Repeating the five-minute smoke or editing more prose cannot close
  the uncovered-test finding.
- This run inspected the production constants and existing doc-contract tests,
  updated `TASK-260825-39ycg2_results.md` to state the smoke/setup revision
  ordering explicitly, and made no repository code or test changes.

## Viable options

1. **Recommended:** reroute the narrow rework to a role permitted to modify Go
   tests. Add mutation-resistant README/SKILL fragment assertions, derive the
   deadline/exit fragments from `infra` constants where practical, run the
   focused doc-contract test and full uncached suite, then return to review.
2. Expand this run's role authorization to include test code. This would permit
   the same narrow change, but contradicts the current explicit role contract
   unless the orchestrator changes the assignment.
3. Waive the reviewer finding. This leaves an unattended `--approve` operator
   safety contract able to disappear or drift while the suite stays green and
   is not recommended.

## Required external decision

The orchestrator must either reroute the requested Go test change to an
authorized code/test role or explicitly revise this run's role scope. No
product or documentation wording decision remains.
