# BUG-260818-jreo1p: pi-readiness-retries-service-unavailable

## Description
Managed Pi production launch must treat HTTP 503 from the exact readiness endpoint as transient while the owned runtime is alive, poll until exact-model readiness or timeout, and continue to reject other non-200 responses without spawning Pi.

## Scope
Update the shared waitPiRuntimeReady production path, its RunPi regression coverage, operator documentation, and installed global/project runtimes; preserve fatal handling for every non-200 status other than exact 503 and preserve process-group cleanup and lock release.

## Acceptance Criteria
A live llama.cpp 503 during model loading is polled until exact-model readiness or configured timeout while the owned child remains alive; 502 and every other non-200 remain runtime_readiness_invalid and never spawn Pi; production-entry tests and a widening mutant bind both branches; full tests, build, vet, setup, verify, and the controlled Qwen text/tool smokes pass with no orphan runtime or listener.
