# BUG-260818-76hkcb: pi-launch-deny-ggml-backend-path

## Description
llama.cpp build 10470 passes inherited GGML_BACKEND_PATH to dlopen() during backend discovery. Managed Pi currently admits the variable and forwards it to the runtime, bypassing the intent of existing DYLD_/LD_ loader-injection gates.

## Scope
Refuse exact GGML_BACKEND_PATH at the shared managed Pi execution-environment boundary before state/runtime spawn; add production, installed-global, and local-wrapper controls and narrowing evidence; document the established dlopen effect without broad speculative GGML_* denial.

## Acceptance Criteria
Managed Pi refuses inherited GGML_BACKEND_PATH before runtime spawn and leaks no value; a clean control reaches runtime backend initialization; production-entry and installed launcher negatives redden if the exact gate is removed; existing endpoint, LLAMA_ARG_*, loader, token, and cache policies remain unchanged; bootstrap and local verification pass.
