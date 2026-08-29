# Incident root cause

## Local evidence

- Active session: `01a03e8e-7c6d-7973-876a-a392202cdd57`.
- Effective generated Pi model context is `75000`; compaction policy is `reserveTokens=25000`, `keepRecentTokens=8192`, so automatic compaction begins at `50000`.
- The persisted session contains two successful compaction records followed by repeated assistant errors at roughly four-to-five-minute intervals.
- On the latest turn, mlx-lm prefills exactly `25032` tokens and then its generation thread throws `RuntimeError: [metal::malloc] Resource limit (499000) exceeded`.
- mlx-lm keeps TCP port 18011 and the accepted HTTP connection alive after the generation thread dies. Pi later records `Request timed out.`, which is a downstream symptom rather than the original failure.
- Runtime: mlx-lm 0.31.3, MLX 0.32.1, model type `qwen3_5`, 64 layers, full-attention interval 4.
- Byte-memory pressure is not the trigger: the machine still reported 37% free memory after the crash. The failing limit is a Metal buffer-object count.

## Upstream mapping

- ml-explore/mlx-lm#1641 identifies the same qwen3_5/qwen3_next batch-path `ArraysCache.advance()` lazy graph leak and the exact 499000-resource failure.
- ml-explore/mlx-lm#1632 merged the minimal upstream fix on 2026-08-22 as commit `11a6ce7`: bind cache metadata to evaluated cache state so the lazy graph remains bounded.
- mlx-lm 0.31.3 predates that merge; the fix is on upstream main but not in the installed release.
- ml-explore/mlx-lm#1505 documents the second defect visible here: an uncaught generation-loop exception leaves the HTTP server apparently alive while completions hang.

## Conclusion

Changing prompt-cache byte limits, wired memory, or the 75k/50k Pi thresholds cannot fix this incident. The primary remediation is to run a reproducibly pinned mlx-lm revision containing #1632. The harness also needs a liveness/response watchdog so a future generation-thread failure cannot masquerade as a healthy runtime and consume repeated client timeouts.

Sources:
- https://github.com/ml-explore/mlx-lm/issues/1641
- https://github.com/ml-explore/mlx-lm/pull/1632
- https://github.com/ml-explore/mlx-lm/issues/1505
