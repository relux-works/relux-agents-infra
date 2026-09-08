# TASK-260830-bq9daj: version-the-local-runtime-status-payload

## Description
Add an explicit contract version to the SharedRuntimeStatus payload delivered by TASK-260829-2t5xmi, and define the compatibility rule a consumer may rely on. Bound the failure and backoff evidence the payload carries so it cannot grow without limit across a long unattended run.

## Scope
(define task scope)

## Acceptance Criteria
The payload carries an explicit contract version. A consumer reading an unknown newer version fails closed with an error naming the version it saw and the range it supports, rather than parsing what it recognizes and ignoring the rest. Failure and backoff evidence is bounded by an enforced limit, demonstrated by a test that drives production through more events than the bound and asserts the retained count and the eviction order. Removing the bound makes that test red.
