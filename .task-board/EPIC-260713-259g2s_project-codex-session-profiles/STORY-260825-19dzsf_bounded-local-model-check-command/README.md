# STORY-260825-19dzsf: bounded-local-model-check-command

## Description
Provide a reusable agents-infra command for observing and validating configured local-model behavior without ad-hoc shell scripts.

## Scope
Add a generic bounded checker over configured agents-infra targets that runs a prompt through the target environment, persists raw JSONL and stderr evidence under an explicit output directory, emits a compact sanitized summary, supports expectation assertions, and verifies managed runtime cleanup. Keep the command provider-aware and avoid exposing credentials or ambient secret values.

## Acceptance Criteria
A documented command can run qwen-infra or another configured target with a default deadline, capture and summarize model/tool behavior, assert expected tools/text and return a useful non-zero result on mismatch or timeout, terminate its managed runtime cleanly, and pass positive plus negative production-entrypoint tests.
