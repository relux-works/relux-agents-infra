# STORY-260830-2vrhg1: deliver-bounded-kv-in-pinned-mlx-lm-fork

## Description
Deliver the bounded KV window in the pinned mlx-lm fork as its own change, because it is a prerequisite of the runtime comparison rather than part of it, and because sharing a Story with the comparison task deadlocked both producers through mutually unaccepted Change Requests.

## Scope
The relux-works/mlx-lm fork and the benchmark provenance that reads it. Delivery vehicle separated from STORY-260828-2faxgm so the comparison task and this prerequisite stop blocking each other.

## Acceptance Criteria
A real bounded KV window works in the pinned Python baseline through qwen3_5 cache construction, the runtime revision is derived from the process that actually served rather than from a caller-supplied interpreter, and the attestation refuses rather than pins a commit it cannot attribute.
