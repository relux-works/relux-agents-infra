# STORY-260830-i540ax: publish-versioned-local-runtime-status-contract

## Description
The shared runtime already emits restart_count, quarantined_until, last_readiness_match and manual_quarantine in SharedRuntimeStatus JSON, delivered by TASK-260829-2t5xmi. What is missing is the versioned, externally consumable contract around that payload, and the registration that makes local-qwen a first-class client-executable runtime for project-management.

## Scope
(define story scope)

## Acceptance Criteria
The status payload carries an explicit contract version, and a documented compatibility rule states what a consumer may assume across versions. agents-management can consume the contract without reading agents-infra internals. project-management resolves local-qwen as a first-class client-executable runtime through that contract. Failure and backoff evidence in the payload is explicitly bounded, and the bound is enforced rather than documented.
