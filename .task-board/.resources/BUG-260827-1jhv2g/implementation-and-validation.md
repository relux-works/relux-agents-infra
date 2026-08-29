# Implementation and validation

## Remediation

- Installed an isolated pipx environment `mlx-lm-qwenfix` from official upstream commit `11a6ce75589f59809d6d79b28efa03c50896c18b`, which contains merged PR #1632.
- Updated `/Users/alexis/src/.agents/.configs/model-harness.toml` to execute the pinned `mlx_lm-qwenfix.server` entry point.
- Added local model-harness supervision with literal fatal-output detection, non-zero exit restart, a rolling restart budget, and unchanged stdout/stderr forwarding.
- Configured Qwen for at most 3 restarts per 3600 seconds with a 1000ms delay.
- Installed agents-infra/model-harness from commit `d91d6fc`.

## Validation

| Check | Result |
| --- | --- |
| Patched ArraysCache 256-step graph regression | PASS; lengths and left_padding each retained 0 graph edges |
| Focused model-harness suite | PASS |
| `go test ./... -count=1` | PASS |
| `go vet ./...` | PASS |
| Installed supervisor process smoke | PASS; fatal child killed, one restart, second launch exited 0 |
| Exact Pi session startup-only restore | PASS; session `01a03e8e-7c6d-7973-876a-a392202cdd57`, 59.9%/75k, no prompt appended |
| Patched 50k synthetic stress | PASS; observed 50000/50000 tokens, 527336ms prefill, 25157664768-byte peak RSS, 36.61% of 64GiB host memory, 2199 samples |
| Cleanup | PASS; stress runtime, Pi, broker, and port 18011 were reaped |

The shell wrapper used around the stress command attempted to assign the zsh read-only `status` variable after model-harness had completed, so that wrapper returned 1. The already-written versioned JSON report is complete and states `status=passed`; `jq` validation and runtime cleanup checks passed.
