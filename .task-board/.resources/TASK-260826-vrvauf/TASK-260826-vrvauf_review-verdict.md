# TASK-260826-vrvauf Review Verdict

Verdict: **changes requested** (`to-dev`)

Reviewer run: `RUN-260830-530745`

## Findings

### F1 — Acceptance value is absent from the production configuration

`/Users/alexis/src/.agents/.configs/project-config.toml` currently defines the inherited `qwen-3.8-27b-mlx-8bit` profile with `context_window = 75000`, not `50000`.

The production `qwen-infra --print-config` and `agents-infra compose --mode primary-session --entrypoint qwen-infra --project /Users/alexis/src --schema-version 1 --json` calls both resolve the ancestor profile from that exact source. The compose result derives `compactAtTokens=50000` and `reserveTokens=25000`, which is consistent with a `75000` window and inconsistent with the requested `50000` window.

The task note claims `131072 -> 50000`, but the current file and `/Users/alexis/src/.agents/.configs/project-config.toml.bak-260830` (created at 02:59) both contain `75000`. No task-scoped producer outcome independently establishes the claimed transition. A failed or contradictory read is not accepted as absence.

### F2 — A context-only change to 50000 is invalid under the current compaction policy

A narrowed fixture copied from the production config changed only `context_window` from `75000` to `50000`. The real `qwen-infra --print-config` entry point exited 1:

`compact_at_tokens: must be less than context_window`

The current threshold is `compact_at_tokens = 50000`; therefore the producer note's stated context-only edit cannot satisfy both the requested window and the compaction acceptance criterion. A control fixture using `context_window = 50000` and `compact_at_tokens = 33000` exited 0, deriving a `17000` reserve that remains above `max_tokens = 16384`; `keep_recent_tokens = 8192` remains below the threshold.

This attacks the production validation path and narrows the bound rather than deleting the gate. It also rules out the “check present but uncalled from production” bypass shape.

## Policy comparison

Against the 02:59 backup, the relevant inherited policy fields remain:

- model: `/Users/alexis/src/local-models/Qwen3.8-27B-Uncensored-MLX-8bit`
- profile reasoning capability: `true`
- target reasoning / thinking: `medium`
- endpoint: `http://127.0.0.1:18011/v1`
- Pi primary-session yolo: `true`
- deployed default profile: `qwen-3.8-27b-mlx-8bit`

The benchmark-only `qwen-remote-tiny` identity remains `Qwen3-0.6B-GGUF`, reasoning `false` / thinking `off`, endpoint `http://127.0.0.1:18012/v1`, and context window `32768`. The backup/current diff contains unrelated shared-runtime supervision fields, but no 50K context-window candidate.

## Validation

- Production static resolution from `/Users/alexis/src`: exit 0, but resolves the 75K policy described above.
- Negative `50000/50000` window/threshold fixture through `qwen-infra --print-config`: exit 1 as required.
- Positive `50000/33000` control fixture through the same entry point: exit 0; model/reasoning/endpoint/yolo unchanged.
- Focused Pi/canonical-target tests: exit 0 (`0.735s`).
- Full `go test ./internal/infra -count=1`: exit 0 (`150.327s`).

## Required rework

1. Set the inherited source profile's `context_window` to `50000` in `/Users/alexis/src/.agents/.configs/project-config.toml`.
2. Adjust compaction coherently: `compact_at_tokens < 50000`, `50000 - compact_at_tokens >= 16384`, and `keep_recent_tokens < compact_at_tokens`. The validated `33000/8192` threshold/tail pair is one passing example; the producer should use the intended policy value.
3. Preserve and name the exact model, target reasoning/thinking, endpoint, primary yolo, deployed-default profile, and benchmark-only profile values above.
4. Rerun the production `qwen-infra` static resolution and attach task-scoped outcome evidence. Do not rely on a board note that contradicts the resolved state.

No reviewed source code was modified. Review probes and logs are confined to `.temp/TASK-260826-vrvauf/`; the separate required logbook entry records the anomaly.
