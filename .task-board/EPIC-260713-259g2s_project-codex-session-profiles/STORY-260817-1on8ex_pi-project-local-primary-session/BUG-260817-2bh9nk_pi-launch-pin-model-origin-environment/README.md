# BUG-260817-2bh9nk: pi-launch-pin-model-origin-environment

## Description
llama.cpp build 10470 honors HF_ENDPOINT and MODEL_ENDPOINT for -hf resolution. A managed profile can keep identical argv and model IDs while resolving weights from an unreviewed origin.

## Scope
Define and enforce managed Pi policy for model-origin environment variables at the shared launch boundary. Cover exact verified endpoint names first; treat tokens and cache variables separately unless their effect is established.

## Acceptance Criteria
Managed Pi launch refuses or explicitly binds HF_ENDPOINT and MODEL_ENDPOINT before llama.cpp spawn; diagnostics leak no environment values; clean controls and independent narrowing tests prove the gate; operator docs state the model-origin policy; bootstrap-installed and local-wrapper behavior are verified.
