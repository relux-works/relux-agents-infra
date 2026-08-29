# TASK-260827-qyebv8 — blocker

## Exact input needed

**Free the host of its resident copy of the Qwen model, then re-run one script.**
Either:

- close the interactive `agents-infra target qwen-infra -- --continue` session
  (pid 12116, in a zellij pane) and let the broker's `linger_seconds = 15`
  expire, **or**
- point this task at a machine that is not already serving this model.

No product, architecture or design decision is needed. Nothing about the
prototype changes based on the answer.

## What is blocked

Checklist items 2, 3, 4 and 5 — the full-model load and the non-streaming,
streaming and tool-call generation smokes, plus the ready-state `200` model
listing.

## The constraint, with evidence

The model is ~29 GB of 8-bit weights and needs roughly **27.5 GiB resident,
mostly wired** (MLX wires its Metal buffers, so they cannot be paged out).
A 64 GiB host cannot hold two copies.

Six samples taken across this task:

| Reading | Range observed |
| --- | --- |
| Host memory | 64 GiB |
| Python `mlx_lm.server` (pid 78542) RSS | 23.1 GiB |
| System wired | 40.1 – 43.1 GiB |
| Free + inactive | **5.6 – 10.7 GiB** |

`agents-infra runtime status --project /Users/alexis/src --profile qwen-3.8-27b-mlx-8bit --json`
reported throughout: broker pid 78540 `serving`, runtime pid 78541, and **one
lease in state `held`**, heartbeat age 0.2–4.7 s. Lease owner:

```
12116  62670  /Users/alexis/.local/bin/agents-infra target qwen-infra -- --continue
62670  62665  /bin/zsh
62665      1  /opt/homebrew/bin/zellij --server ...
```

That is a live interactive operator session, running for 1h45m+ at last check.

## Why it was not resolved autonomously

Attempting the load would push the host past its wired limit. It would not just
fail this task — it would destabilise the runtime that interactive session is
using. Stopping that runtime, even through the supported
`agents-infra runtime stop` path, kills someone's in-progress work.

## Attempts made, and what they bought

Rather than wait idle, everything provable without resident weights was proven:

1. **`preflight` subcommand** (new) — exercises architecture registry,
   configuration decode, tokenizer and chat template at zero GPU cost. Exit 0 on
   the exact model. It establishes that `qwen3_5` is implemented by mlx-swift-lm
   3.31.4 in both `MLXVLM` and `MLXLLM`, that this model's `config.json` decodes
   into `MLXVLM.Qwen35Configuration`, that the tokenizer loads, and that the chat
   template renders with tool declarations and ends inside an open `<think>`
   block. **There is no compatibility gap for this model** — what remains
   unknown is runtime behaviour.
2. **`scripts/lifecycle-smoke.sh`** (new) — 17/17 checks against a fixture no
   factory can load, so it costs no GPU memory: startup refusals, listener bound
   before load, `503` while not ready, model ID absent from the not-ready
   listing, `model_not_ready` on completions, load-failure reporting, SIGTERM
   exit 0, port released.
3. **Managed-path run** — `model-harness run` against that fixture: direct-child
   ownership, unchanged stdout forwarding, readiness answer, SIGTERM group
   shutdown, port released.

## To finish, once the host is free

```bash
cd tools/mlx-swift-runtime-prototype
HARNESS=<repo>/.temp/TASK-260827-qyebv8/model-harness \
HARNESS_CONFIG=<repo>/.temp/TASK-260827-qyebv8/model-harness-prototype.toml \
PORT=18017 OUT=<repo>/.temp/TASK-260827-qyebv8/smoke-out \
scripts/smoke.sh
```

`scripts/smoke.sh` is written and syntax-checked. It renders the plan, starts
the runtime through `model-harness run`, polls `/v1/models` with the launcher's
own readiness rules, verifies the four refusals against the live server, runs
bounded non-streaming / streaming / tool-call completions, then SIGTERMs the
process group and confirms the port is released. It exits non-zero on any
failure and needs no further code changes.
