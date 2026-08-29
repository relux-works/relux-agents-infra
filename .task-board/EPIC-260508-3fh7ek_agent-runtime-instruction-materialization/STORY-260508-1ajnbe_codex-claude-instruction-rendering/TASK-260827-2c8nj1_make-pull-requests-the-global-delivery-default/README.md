# TASK-260827-2c8nj1: make-pull-requests-the-global-delivery-default

## Description
Add a global agent instruction that makes branch-to-pull-request-to-self-review-to-green-checks-to-self-merge the canonical publication workflow for hosted repositories so real review work is represented by platform events.

## Scope
Global INSTRUCTIONS_WORKFLOW.md source, generated instruction surfaces, task evidence, and installation verification. Preserve task-board authority over commit authorization and repository-specific stronger policies.

## Acceptance Criteria
Global instructions prefer PR or MR delivery over direct default-branch pushes, require the owning agent to inspect the complete remote diff and checks and submit a real review verdict, require merge only after acceptance, explain the self-approval limitation without fake identities, and preserve stricter repository rules.
