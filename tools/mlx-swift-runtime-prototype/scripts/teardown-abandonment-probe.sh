#!/usr/bin/env bash
# What a condemned-worker teardown costs on the full 28 GB model.
#
# Every prior measurement of this path used the 261 MB `Qwen1.5-0.5B-Chat-4bit`
# fixture, because the paths it exercises never reach the weights. The one thing
# that fixture cannot report is the *size* of what the abandoned rebuild leaves
# behind, and that number is the whole risk: under MLX's process-global
# counters this runtime abandons the shared-cache rebuild whenever the weight
# release cannot be proved, and an abandoned rebuild does not clear the pool. On
# a 261 MB model that is a rounding error. On this one it is the host.
#
# The probe arms the real fault seam on the real Release binary, lets a real
# generation deliver tokens and then fail with the recorded `metal::malloc`
# signature, and measures what the teardown does about it.
#
#   BINARY=./DerivedData/Build/Products/Release/mlx-swift-runtime-prototype \
#   MODEL=/Users/alexis/src/local-models/Qwen3.8-27B-Uncensored-MLX-8bit \
#   PORT=18033 OUT=./teardown-out \
#   scripts/teardown-abandonment-probe.sh
set -uo pipefail

BINARY="${BINARY:?BINARY is required}"
MODEL="${MODEL:?MODEL is required}"
PORT="${PORT:-18033}"
OUT="${OUT:-./teardown-out}"
FAULT="${FAULT:-RuntimeError: [metal::malloc] Resource limit (499000) exceeded}"
AFTER_TOKENS="${AFTER_TOKENS:-8}"

mkdir -p "$OUT"
LOG="$OUT/runtime.log"
: > "$LOG"

fail() { echo "PROBE FAILED: $*" >&2; exit 1; }

if lsof -nP -iTCP:"$PORT" -sTCP:LISTEN >/dev/null 2>&1; then
    fail "port $PORT is occupied; a stale listener could satisfy every check below"
fi

now() { python3 -c 'import time;print(repr(time.time()))'; }
footprint() {
    python3 - "$1" <<'PY'
import ctypes, sys
libc = ctypes.CDLL("/usr/lib/libSystem.B.dylib")
buf = ctypes.create_string_buffer(4096)
if libc.proc_pid_rusage(ctypes.c_int(int(sys.argv[1])), ctypes.c_int(4), ctypes.byref(buf)) != 0:
    print("null")
else:
    print(int.from_bytes(buf.raw[72:80], "little"))
PY
}

# Armed deliberately with `--fault-inject-generation-error-after-tokens`: firing
# before MLX is touched releases nothing, so a runtime that leaked every KV
# cache it ever built would still look clean. Firing after N delivered tokens
# means the batch entry, its cache and partial output all exist at the moment
# of failure.
"$BINARY" serve --model "$MODEL" --host 127.0.0.1 --port "$PORT" \
    --reasoning-effort medium --default-max-tokens 2048 \
    --fault-inject-generation-error "$FAULT" \
    --fault-inject-generation-error-after-tokens "$AFTER_TOKENS" \
    >>"$LOG" 2>&1 &
RUNTIME_PID=$!
trap 'kill -TERM "$RUNTIME_PID" 2>/dev/null; wait "$RUNTIME_PID" 2>/dev/null' EXIT

START="$(now)"
READY=""
for _ in $(seq 1 600); do
    kill -0 "$RUNTIME_PID" 2>/dev/null || fail "runtime exited before readiness; see $LOG"
    if [ "$(curl -s -o /dev/null -w '%{http_code}' "http://127.0.0.1:$PORT/v1/models")" = "200" ]; then
        READY="$(now)"; break
    fi
    sleep 1
done
[ -n "$READY" ] || fail "runtime never became ready"

LOAD_SECONDS="$(python3 -c "print(round($READY-$START,3))")"
FOOTPRINT_READY="$(footprint "$RUNTIME_PID")"
echo "load_seconds=$LOAD_SECONDS footprint_ready_bytes=$FOOTPRINT_READY"

REQUEST_AT="$(now)"
curl -s -N -X POST "http://127.0.0.1:$PORT/v1/chat/completions" \
    -H 'Content-Type: application/json' \
    -d "{\"model\":\"$MODEL\",\"messages\":[{\"role\":\"user\",\"content\":\"Describe one routine coolant loop check.\"}],\"max_tokens\":128,\"temperature\":0,\"top_p\":1,\"seed\":1234,\"stream\":true}" \
    > "$OUT/condemning-request.sse" 2>&1
echo "condemning request returned after $(python3 -c "print(round($(now)-$REQUEST_AT,3))")s"

# The teardown runs on a `deinit`-scheduled task: three attempts, each bounded
# at about two seconds. Waiting well past that bound and then reporting what was
# actually seen keeps a slow machine from being recorded as a missing event.
OUTCOME=""
OUTCOME_AT=""
for _ in $(seq 1 400); do
    if grep -q 'generation_shared_cache_rebuild_abandoned' "$LOG"; then
        OUTCOME="abandoned"; OUTCOME_AT="$(now)"; break
    fi
    if grep -q 'generation_shared_cache_rebuilt' "$LOG"; then
        OUTCOME="rebuilt"; OUTCOME_AT="$(now)"; break
    fi
    sleep 0.05
done
[ -n "$OUTCOME" ] || fail "no teardown outcome event appeared; see $LOG"

TEARDOWN_SECONDS="$(python3 -c "print(round($OUTCOME_AT-$REQUEST_AT,3))")"
FOOTPRINT_AFTER="$(footprint "$RUNTIME_PID")"
HEALTH="$(curl -s -o "$OUT/health.json" -w '%{http_code}' "http://127.0.0.1:$PORT/health")"
MODELS="$(curl -s -o "$OUT/models.json" -w '%{http_code}' "http://127.0.0.1:$PORT/v1/models")"

grep -E 'generation_shared_cache_rebuild_abandoned|generation_shared_cache_rebuilt|generation_shared_cache_rebuild_deferred|generation_worker_failed' "$LOG" \
    > "$OUT/teardown-events.jsonl"

python3 - "$OUT" "$OUTCOME" "$TEARDOWN_SECONDS" "$LOAD_SECONDS" \
    "$FOOTPRINT_READY" "$FOOTPRINT_AFTER" "$HEALTH" "$MODELS" <<'PY'
import json, sys
out, outcome, teardown_s, load_s, fp_ready, fp_after, health, models = sys.argv[1:]
events = []
with open(f"{out}/teardown-events.jsonl") as handle:
    for line in handle:
        line = line.strip()
        if line.startswith("{"):
            try:
                events.append(json.loads(line))
            except json.JSONDecodeError:
                pass
terminal = next(
    (e for e in events if e.get("event", "").startswith("generation_shared_cache_rebuild")
     and e.get("event") != "generation_shared_cache_rebuild_deferred"),
    None,
)
report = {
    "outcome": outcome,
    "teardown_seconds_from_condemning_request": float(teardown_s),
    "load_seconds": float(load_s),
    "footprint_at_ready_bytes": None if fp_ready == "null" else int(fp_ready),
    "footprint_after_teardown_bytes": None if fp_after == "null" else int(fp_after),
    "health_status": int(health),
    "models_status": int(models),
    "deferred_attempts": sum(
        1 for e in events if e.get("event") == "generation_shared_cache_rebuild_deferred"
    ),
    "terminal_event": terminal,
}
with open(f"{out}/teardown-abandonment.json", "w") as handle:
    json.dump(report, handle, indent=2, sort_keys=True)
print(json.dumps(report, indent=2, sort_keys=True))
PY

FAILURES=0
check() {
    if [ "$2" = "$3" ]; then echo "PASS  $1"; else echo "FAIL  $1 (expected $3, got $2)"; FAILURES=$((FAILURES+1)); fi
}
# The probe measures a cost; it does not get to decide the outcome. What it
# asserts is only that the runtime really was condemned, because a teardown
# measured on a runtime that stayed healthy is measuring nothing.
check "/health reports the condemned worker" "$HEALTH" 503
check "/v1/models stops advertising" "$MODELS" 503
echo "teardown outcome: $OUTCOME after ${TEARDOWN_SECONDS}s"
[ "$FAILURES" -eq 0 ] || fail "$FAILURES check(s) failed"
echo "TEARDOWN ABANDONMENT PROBE OK"
