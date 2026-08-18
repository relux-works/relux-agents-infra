# BUG-260818-1s1lka: pi-launch-deny-ambient-llama-api-key

## Description
llama.cpp build 10470 documents LLAMA_API_KEY as the environment backing --api-key. Managed profiles do not declare --api-key and generated Pi model configuration has no matching credential, so inherited LLAMA_API_KEY silently changes runtime authentication and causes unreviewed or misleading live behavior.

## Scope
Refuse exact LLAMA_API_KEY at the shared managed Pi execution-environment boundary before state/runtime spawn; preserve HF_TOKEN and cache-variable policy; add production, installed launcher, clean, narrowing, non-leak, and docs evidence.

## Acceptance Criteria
Managed Pi refuses inherited LLAMA_API_KEY before runtime spawn without leaking its value; clean controls with HF_TOKEN and unrelated names reach runtime; production and installed launcher suites redden if the exact gate is narrowed or removed; docs state the no-ambient-auth policy; bootstrap and local verification pass.
