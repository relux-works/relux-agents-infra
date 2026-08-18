# BUG-260817-161m6u: pi-launch-deny-llama-arg-environment

## Description
Managed Pi launch currently passes LLAMA_ARG_* environment variables through to llama.cpp. For options absent from runtime argv, llama.cpp silently gives those variables effect, allowing profile model and runtime parameters to drift from the reviewed plan.

## Scope
Deny runtime-affecting LLAMA_ARG_* names at the shared managed Pi execution-environment boundary, cover production launch paths with negative tests, document the contract, and refresh installed runtime through setup.

## Acceptance Criteria
Managed Pi launch refuses inherited LLAMA_ARG_* variables before spawning llama.cpp; refusal names only sanitized variable names and leaks no values; focused tests cover LLAMA_ARG_MODEL and another absent-option variable plus an exact clean-environment control; existing DYLD_/LD_/NODE_/BUN_/PI_* gates remain intact; setup and verification pass.
