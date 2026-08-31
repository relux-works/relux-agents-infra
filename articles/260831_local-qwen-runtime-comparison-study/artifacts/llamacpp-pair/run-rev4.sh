#!/bin/bash
# TASK-260829-3k4qrc revision 4: one foreground `benchmark-run` invocation that
# owns both passes, bracketed by host sweeps.
#
# Process hygiene (blocker B7): this script terminates only the process group it
# created itself. It never kills a process by name, so another run's processes
# are never reaped by it. If an orphan survives, a raw process-list snapshot is
# written before anything is signalled.
set -u

WT=/Users/alexis/src/relux-works/relux-agents-infra/.temp/STORY-260828-2faxgm/worktree
PROTO="$WT/tools/mlx-swift-runtime-prototype"
OUT="$WT/.temp/TASK-260829-3k4qrc/rev4"
GATE="$PROTO/.build/release/mlx-swift-runtime-prototype"
CONFIG="$OUT/TASK-260829-3k4qrc-rev4.benchmark.toml"
SESSION="$OUT/session-rev4"

sweep() {
    local label="$1"
    {
        echo "=== $label @ $(date -u +%FT%TZ) epoch=$(date +%s) ==="
        echo "--- uptime ---"; uptime
        echo "--- vm_stat ---"; vm_stat
        echo "--- runtime processes ---"
        if ! ps -Ao pid,ppid,pgid,rss,etime,command | grep -Ei 'llama-server|mlx_lm|mlx-swift|model-harness run' | grep -v grep; then
            echo "(none)"
        fi
    } >> "$OUT/run-rev4-sweeps.log" 2>&1
}

sweep "PRE-RUN SWEEP"

# Full raw process list before the run, so an orphan found afterwards can be
# attributed rather than guessed at (BUG-260830-2950qe).
ps -Ao pid,ppid,pgid,user,rss,etime,command > "$OUT/raw-processes-before.txt" 2>&1

START_EPOCH=$(date +%s)
echo "driver started $(date -u +%FT%TZ) epoch=$START_EPOCH" > "$OUT/run-rev4-interval.txt"

# The invocation. Its children live in this script's own process group.
"$GATE" benchmark-run \
    --config "$CONFIG" \
    --model /Users/alexis/src/local-models/Qwen3.8-27B-Uncensored-MLX-8bit \
    --candidate-model /Users/alexis/src/local-models/Qwen3.8-27B-Uncensored-GGUF-Q8_0/Qwen3.8-27B-Uncensored-OrcaRouter-Q8_0.gguf \
    --equivalence "$PROTO/equivalence/qwen3-8-27b-uncensored.equivalence.json" \
    --prompts "$PROTO/examples/benchmark-prompts.json" \
    --thresholds "$PROTO/examples/benchmark-thresholds.json" \
    --session "$SESSION" \
    --harness /Users/alexis/.local/bin/model-harness \
    --baseline-runtime python-mlx-lm --baseline-profile qwen-benchmark-python \
    --candidate-runtime llamacpp --candidate-profile qwen-benchmark-llamacpp \
    --port 18341 \
    --python-bin /Users/alexis/.local/pipx/venvs/mlx-lm-kv76800-45a472f/bin/python \
    --baseline-declare 'Python-only prompt cache: launched with --prompt-cache-size 1 --prompt-cache-bytes 8GB; it can reduce TTFT on a repeated prompt and can retain up to the configured cache budget. Direction for a cold, unique prompt is unknown.' \
    --candidate-declare 'llama.cpp per-slot KV reuse: it can reduce TTFT when a slot reuses a matching prefix and can retain slot-local KV state across requests and therefore across scenarios. Direction without an observed reuse hit is unknown.' \
    --candidate-declare 'llama-server allocates its whole --ctx-size KV arena at load, so the 76800-token window is a resident cost paid before the first token rather than a limit' \
    --startup-timeout 900 \
    --request-timeout 10800 \
    --settle-seconds 20 \
    > "$OUT/run-rev4.log" 2>&1
EXIT=$?

END_EPOCH=$(date +%s)
{
    echo "driver finished $(date -u +%FT%TZ) epoch=$END_EPOCH"
    echo "driver exit $EXIT"
    echo "driver wall seconds $((END_EPOCH - START_EPOCH))"
} >> "$OUT/run-rev4-interval.txt"

sweep "POST-RUN SWEEP"
ps -Ao pid,ppid,pgid,user,rss,etime,command > "$OUT/raw-processes-after.txt" 2>&1

# Orphan check, scoped to this run's own config path. Snapshot first, always.
ORPHANS=$(pgrep -f "$CONFIG" | tr '\n' ' ')
if [ -n "${ORPHANS// /}" ]; then
    echo "ORPHANS naming this run's config: $ORPHANS" >> "$OUT/run-rev4-interval.txt"
    ps -Ao pid,ppid,pgid,user,rss,etime,command > "$OUT/raw-processes-orphan-snapshot.txt" 2>&1
else
    echo "no orphan names this run's config" >> "$OUT/run-rev4-interval.txt"
fi

echo "BENCHMARK_RUN_EXIT=$EXIT" > "$OUT/run-rev4.exit"
exit "$EXIT"
