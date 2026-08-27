#!/usr/bin/env bash
# Bounded lifecycle smoke that never loads real weights.
#
# Drives the startup / not-ready / load-failure / shutdown path using a fixture
# model directory that passes directory admission but that no MLX Swift LM
# factory can load. It costs no GPU memory, so it can run while the default
# Python runtime for the same model is resident.
#
# It also records verbatim what MLX Swift LM says when it refuses an
# architecture, which is the raw material for the unsupported-model gap list.

set -uo pipefail

BINARY="${BINARY:?set BINARY to the prototype executable}"
PORT="${PORT:-18018}"
HOST=127.0.0.1
OUT="${OUT:-./lifecycle-out}"
MODEL_TYPE="${MODEL_TYPE:-not_a_real_architecture}"
BASE="http://${HOST}:${PORT}/v1"

mkdir -p "$OUT"
FIXTURE="$OUT/fixture-model"
rm -rf "$FIXTURE"
mkdir -p "$FIXTURE"
printf '{"model_type": "%s", "hidden_size": 8}\n' "$MODEL_TYPE" > "$FIXTURE/config.json"
printf '{"tokenizer_class": "PreTrainedTokenizerFast"}\n' > "$FIXTURE/tokenizer_config.json"
: > "$FIXTURE/model.safetensors"

FAILURES=0
pass() { printf 'PASS  %s\n' "$*"; }
fail() { printf 'FAIL  %s\n' "$*"; FAILURES=$((FAILURES + 1)); }
info() { printf 'INFO  %s\n' "$*"; }

status_of() { curl -sS --max-time 5 -o "$2" -w '%{http_code}' "$1" 2>/dev/null; }

# ------------------------------------------------- refusals before binding --
info "checking startup refusals"
"$BINARY" serve --model "$FIXTURE" --host 0.0.0.0 --port "$PORT" > "$OUT/refuse-host.log" 2>&1
[ $? -eq 2 ] && pass "non-loopback host exits 2" || fail "non-loopback host did not exit 2"
grep -q "host must equal 127.0.0.1" "$OUT/refuse-host.log" \
    && pass "non-loopback refusal names the required host" \
    || fail "non-loopback refusal message missing"

"$BINARY" serve --model "$OUT/does-not-exist" --host "$HOST" --port "$PORT" \
    > "$OUT/refuse-missing.log" 2>&1
[ $? -eq 2 ] && pass "missing model directory exits 2" || fail "missing model directory did not exit 2"
grep -q "does not exist" "$OUT/refuse-missing.log" \
    && pass "missing directory is reported as missing" \
    || fail "missing directory message missing"

"$BINARY" serve --model "$FIXTURE" --host "$HOST" --port "$PORT" --reasoning-effort high \
    > "$OUT/refuse-effort.log" 2>&1
[ $? -eq 2 ] && pass "unsupported reasoning effort exits 2" \
    || fail "unsupported reasoning effort did not exit 2"

# ---------------------------------------------------------------- lifecycle --
if lsof -nP -iTCP:"$PORT" -sTCP:LISTEN >/dev/null 2>&1; then
    fail "port $PORT already in use"
    exit 1
fi

info "starting against the unloadable fixture"
"$BINARY" serve --model "$FIXTURE" --host "$HOST" --port "$PORT" > "$OUT/runtime.log" 2>&1 &
PID=$!
trap 'kill -KILL "$PID" 2>/dev/null || true' EXIT

# The listener must come up before the load resolves.
BOUND=0
for _ in $(seq 1 100); do
    if lsof -nP -iTCP:"$PORT" -sTCP:LISTEN >/dev/null 2>&1; then BOUND=1; break; fi
    sleep 0.1
done
[ "$BOUND" -eq 1 ] && pass "listener bound before the model resolved" \
    || fail "listener never bound"

CODE=$(status_of "$BASE/models" "$OUT/models-notready.json")
if [ "$CODE" = "503" ]; then
    pass "/v1/models answers 503 while not ready"
else
    fail "/v1/models answered $CODE while not ready"
fi
python3 -c '
import json, sys
body = json.load(open(sys.argv[1]))
assert body["object"] == "list", body
assert body["data"] == [], body
' "$OUT/models-notready.json" 2>/dev/null \
    && pass "not-ready listing is an empty OpenAI model list" \
    || fail "not-ready listing is not an empty OpenAI model list"

if ! grep -q "$(cd "$FIXTURE" && pwd)" "$OUT/models-notready.json"; then
    pass "the model ID is absent from the not-ready listing"
else
    fail "the model ID leaked into the not-ready listing"
fi

CODE=$(curl -sS --max-time 5 -o "$OUT/chat-notready.json" -w '%{http_code}' \
    -H 'Content-Type: application/json' -X POST "$BASE/chat/completions" \
    --data-binary "{\"model\":\"$FIXTURE\",\"messages\":[{\"role\":\"user\",\"content\":\"hi\"}]}" \
    2>/dev/null)
[ "$CODE" = "503" ] && pass "chat completions answers 503 while not ready" \
    || fail "chat completions answered $CODE while not ready"
python3 -c '
import json, sys
body = json.load(open(sys.argv[1]))
assert body["error"]["code"] == "model_not_ready", body
' "$OUT/chat-notready.json" 2>/dev/null \
    && pass "not-ready completion reports model_not_ready" \
    || fail "not-ready completion did not report model_not_ready"

# --------------------------------------------------- unsupported architecture --
info "waiting for the load to be refused"
FAILED=0
for _ in $(seq 1 100); do
    if grep -q '"event":"model_load_failed"' "$OUT/runtime.log"; then FAILED=1; break; fi
    sleep 0.2
done
if [ "$FAILED" -eq 1 ]; then
    pass "unloadable architecture produced a model_load_failed event"
    grep '"event":"model_load_failed"' "$OUT/runtime.log" | tail -1 > "$OUT/load-failed.json"
    info "refusal recorded in $OUT/load-failed.json"
else
    fail "no model_load_failed event within 20s"
fi

CODE=$(status_of "$BASE/models" "$OUT/models-failed.json")
[ "$CODE" = "503" ] && pass "/v1/models stays 503 after a failed load" \
    || fail "/v1/models answered $CODE after a failed load"
python3 -c '
import json, sys
body = json.load(open(sys.argv[1]))
assert body["data"] == [], body
assert body["error"]["code"] == "model_load_failed", body
' "$OUT/models-failed.json" 2>/dev/null \
    && pass "failed listing reports model_load_failed and advertises nothing" \
    || fail "failed listing shape is wrong"

# ----------------------------------------------------------------- shutdown --
info "sending SIGTERM"
START=$SECONDS
kill -TERM "$PID"
STOPPED=0
while [ $((SECONDS - START)) -lt 10 ]; do
    kill -0 "$PID" 2>/dev/null || { STOPPED=1; break; }
    sleep 0.2
done
wait "$PID" 2>/dev/null
EXIT_STATUS=$?
trap - EXIT
if [ "$STOPPED" -eq 1 ] && [ "$EXIT_STATUS" -eq 0 ]; then
    pass "SIGTERM exited 0 in $((SECONDS - START))s"
else
    fail "SIGTERM handling failed (stopped=$STOPPED status=$EXIT_STATUS)"
fi
grep -q '"event":"stopped"' "$OUT/runtime.log" \
    && pass "runtime emitted a stopped event" \
    || fail "runtime did not emit a stopped event"

sleep 0.5
if lsof -nP -iTCP:"$PORT" -sTCP:LISTEN >/dev/null 2>&1; then
    fail "port $PORT still has a listener"
else
    pass "port $PORT released"
fi

printf '\n%s\n' "----------------------------------------"
if [ "$FAILURES" -eq 0 ]; then
    printf 'LIFECYCLE SMOKE OK (0 failures)\n'
    exit 0
fi
printf 'LIFECYCLE SMOKE FAILED (%d failures)\n' "$FAILURES"
exit 1
