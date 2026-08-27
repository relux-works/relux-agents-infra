#!/usr/bin/env bash
# Bounded end-to-end smoke for the MLX Swift LM prototype runtime.
#
# Drives the prototype the way the managed contract does: start it through
# `model-harness run`, poll `/v1/models` with the launcher's readiness rules,
# exercise the OpenAI-compatible surface the Pi profile uses, then stop it with
# SIGTERM to the process group and confirm the endpoint is released.
#
# Every request is time-bounded and every check reports PASS or FAIL. The script
# exits non-zero if any check fails.

set -uo pipefail

MODEL="${MODEL:-/Users/alexis/src/local-models/Qwen3.8-27B-Uncensored-MLX-8bit}"
PORT="${PORT:-18017}"
HOST=127.0.0.1
PROFILE="${PROFILE:-qwen-mlx-swift-prototype}"
HARNESS="${HARNESS:?set HARNESS to a model-harness binary}"
HARNESS_CONFIG="${HARNESS_CONFIG:?set HARNESS_CONFIG to an absolute config path}"
OUT="${OUT:-./smoke-out}"
STARTUP_TIMEOUT="${STARTUP_TIMEOUT:-900}"
SHUTDOWN_TIMEOUT="${SHUTDOWN_TIMEOUT:-30}"
REQUEST_TIMEOUT="${REQUEST_TIMEOUT:-300}"
MAX_TOKENS="${MAX_TOKENS:-192}"

# The inline python helpers below read these from the environment.
export MODEL MAX_TOKENS

BASE="http://${HOST}:${PORT}/v1"
mkdir -p "$OUT"
RUNTIME_LOG="$OUT/runtime.log"
: > "$RUNTIME_LOG"

FAILURES=0
pass() { printf 'PASS  %s\n' "$*"; }
fail() { printf 'FAIL  %s\n' "$*"; FAILURES=$((FAILURES + 1)); }
info() { printf 'INFO  %s\n' "$*"; }

json_check() { # json_check <file> <python-expr-returning-bool> <label>
    if python3 -c '
import json, sys
path, expr, = sys.argv[1], sys.argv[2]
try:
    body = json.load(open(path))
except Exception as error:
    print(f"    unparseable JSON: {error}")
    sys.exit(1)
sys.exit(0 if eval(expr, {"body": body, "json": json}) else 1)
' "$1" "$2"; then
        pass "$3"
    else
        fail "$3"
        printf '    body: %s\n' "$(head -c 600 "$1")"
    fi
}

post() { # post <name> <json-body> ; writes $OUT/<name>.json, echoes status
    curl -sS --max-time "$REQUEST_TIMEOUT" -o "$OUT/$1.json" -w '%{http_code}' \
        -H 'Content-Type: application/json' -X POST "$BASE/chat/completions" \
        --data-binary "$2" 2>>"$OUT/curl.err"
}

# ---------------------------------------------------------------- preflight --
if lsof -nP -iTCP:"$PORT" -sTCP:LISTEN >/dev/null 2>&1; then
    fail "port $PORT is already in use; refusing to smoke against a foreign listener"
    exit 1
fi
pass "port $PORT is free before start"

info "rendering the model-harness plan"
if "$HARNESS" render "$PROFILE" --config "$HARNESS_CONFIG" --host "$HOST" --port "$PORT" --json \
    > "$OUT/plan.json" 2>"$OUT/plan.err"; then
    pass "model-harness render resolved the profile"
    json_check "$OUT/plan.json" \
        "body['contract'] == 'model-harness.launch-plan' and body['endpoint'] == '$BASE'" \
        "plan endpoint is $BASE"
else
    fail "model-harness render failed: $(cat "$OUT/plan.err")"
    exit 1
fi

# ------------------------------------------------------------------- start --
info "starting the runtime through model-harness run"
set -m
"$HARNESS" run "$PROFILE" --config "$HARNESS_CONFIG" --host "$HOST" --port "$PORT" \
    >>"$RUNTIME_LOG" 2>&1 &
HARNESS_PID=$!
set +m
info "model-harness pid $HARNESS_PID"

cleanup() {
    if kill -0 "$HARNESS_PID" 2>/dev/null; then
        kill -TERM -- "-$HARNESS_PID" 2>/dev/null || kill -TERM "$HARNESS_PID" 2>/dev/null
        sleep 2
        kill -KILL -- "-$HARNESS_PID" 2>/dev/null || true
    fi
}
trap cleanup EXIT

# --------------------------------------------------------------- readiness --
# Replicates waitPiRuntimeReady: 503 means keep waiting, 200 must contain the
# exact configured model ID, anything else is a hard readiness failure.
info "polling $BASE/models (timeout ${STARTUP_TIMEOUT}s)"
READY=0
SAW_503=0
LOAD_START=$SECONDS
while [ $((SECONDS - LOAD_START)) -lt "$STARTUP_TIMEOUT" ]; do
    if ! kill -0 "$HARNESS_PID" 2>/dev/null; then
        fail "runtime exited before readiness"
        break
    fi
    CODE=$(curl -sS --max-time 5 -o "$OUT/models.json" -w '%{http_code}' "$BASE/models" 2>/dev/null)
    case "$CODE" in
        503)
            if [ "$SAW_503" -eq 0 ]; then
                SAW_503=1
                cp "$OUT/models.json" "$OUT/models-loading.json"
                info "observed 503 while loading after $((SECONDS - LOAD_START))s"
            fi
            ;;
        200)
            if python3 -c '
import json, sys
body = json.load(open(sys.argv[1]))
sys.exit(0 if any(m.get("id") == sys.argv[2] for m in body.get("data", [])) else 1)
' "$OUT/models.json" "$MODEL"; then
                READY=1
                READY_SECONDS=$((SECONDS - LOAD_START))
                break
            fi
            fail "200 from /v1/models without the configured model ID"
            break
            ;;
        000) ;;  # not listening yet
        *)
            fail "unexpected readiness status $CODE"
            break
            ;;
    esac
    sleep 2
done

if [ "$READY" -ne 1 ]; then
    fail "runtime never became ready"
    printf -- '--- runtime log (tail) ---\n%s\n' "$(tail -40 "$RUNTIME_LOG")"
    exit 1
fi
pass "runtime ready after ${READY_SECONDS}s"

if [ "$SAW_503" -eq 1 ]; then
    json_check "$OUT/models-loading.json" \
        "body['object'] == 'list' and body['data'] == []" \
        "loading listing is an empty OpenAI model list"
else
    info "load completed before the first poll; no 503 sample captured"
fi

json_check "$OUT/models.json" \
    "body['object'] == 'list' and [m['id'] for m in body['data']] == ['$MODEL']" \
    "ready listing advertises exactly the configured model"

grep -h '"event":"model_loaded"' "$RUNTIME_LOG" | tail -1 > "$OUT/model-loaded.json"
if [ -s "$OUT/model-loaded.json" ]; then
    json_check "$OUT/model-loaded.json" \
        "body['load_seconds'] > 0 and body['resident_bytes'] is not None" \
        "runtime reported load time and resident memory"
    info "load event: $(cat "$OUT/model-loaded.json")"
else
    fail "no model_loaded event on stdout"
fi

# ------------------------------------------------------- refusal behaviour --
CODE=$(post refuse-model "$(python3 -c '
import json, sys
print(json.dumps({"model": "gpt-4o",
                  "messages": [{"role": "user", "content": "hi"}]}))')")
[ "$CODE" = "404" ] && pass "unknown model refused with 404" || fail "unknown model returned $CODE"
json_check "$OUT/refuse-model.json" "body['error']['code'] == 'model_not_found'" \
    "unknown model reports model_not_found"

CODE=$(post refuse-role "$(python3 -c '
import json, os, sys
print(json.dumps({"model": os.environ["MODEL"],
                  "messages": [{"role": "developer", "content": "be terse"}]}))')")
[ "$CODE" = "400" ] && pass "developer role refused with 400" || fail "developer role returned $CODE"
json_check "$OUT/refuse-role.json" "body['error']['code'] == 'unsupported_role'" \
    "developer role reports unsupported_role"

CODE=$(post refuse-effort "$(python3 -c '
import json, os
print(json.dumps({"model": os.environ["MODEL"],
                  "messages": [{"role": "user", "content": "hi"}],
                  "reasoning_effort": "high"}))')")
[ "$CODE" = "400" ] && pass "reasoning_effort refused with 400" || fail "reasoning_effort returned $CODE"
json_check "$OUT/refuse-effort.json" "body['error']['code'] == 'unsupported_parameter'" \
    "reasoning_effort reports unsupported_parameter"

CODE=$(post refuse-template "$(python3 -c '
import json, os
print(json.dumps({"model": os.environ["MODEL"],
                  "messages": [{"role": "user", "content": "hi"}],
                  "chat_template_kwargs": {"enable_thinking": False}}))')")
[ "$CODE" = "400" ] && pass "chat_template_kwargs refused with 400" \
    || fail "chat_template_kwargs returned $CODE"

# An integral JSON number too large for Int used to reach `Int(_:)` and abort the
# whole process, so the runtime died before any refusal could be sent. Probed
# against the live ready endpoint, not only in unit tests: the refusal must be a
# bounded 400 AND the runtime must still be answering afterwards.
for LITERAL in 1e300 -1e300 9223372036854775808; do
    NAME="refuse-range-$(printf '%s' "$LITERAL" | tr -d '.-')"
    CODE=$(post "$NAME" "$(LITERAL="$LITERAL" python3 -c '
import json, os
print(json.dumps({"model": os.environ["MODEL"],
                  "messages": [{"role": "user", "content": "hi"}]})[:-1]
      + ", \"max_tokens\": " + os.environ["LITERAL"] + "}")')")
    [ "$CODE" = "400" ] && pass "max_tokens $LITERAL refused with 400" \
        || fail "max_tokens $LITERAL returned $CODE"
    json_check "$OUT/$NAME.json" \
        "body['error']['code'] == 'invalid_body' and 'representable integer range' in body['error']['message']" \
        "max_tokens $LITERAL reports an out-of-range body, not a crash"
    # Survival check: a dead runtime cannot answer this.
    HEALTH=$(curl -sS --max-time 5 -o "$OUT/$NAME-health.json" -w '%{http_code}' \
        "http://${HOST}:${PORT}/health" 2>/dev/null)
    [ "$HEALTH" = "200" ] && pass "runtime still healthy after max_tokens $LITERAL" \
        || fail "runtime unhealthy ($HEALTH) after max_tokens $LITERAL"
done

# The gate rejects unrepresentable values, not large ones: the boundary is still
# a normal request, refused only by the ordinary positive-integer rule.
CODE=$(post accept-boundary "$(python3 -c '
import json, os
print(json.dumps({"model": os.environ["MODEL"],
                  "messages": [{"role": "user", "content": "hi"}],
                  "max_tokens": 0}))')")
[ "$CODE" = "400" ] && pass "max_tokens 0 still hits the ordinary bound" \
    || fail "max_tokens 0 returned $CODE"
json_check "$OUT/accept-boundary.json" "body['error']['code'] == 'invalid_max_tokens'" \
    "max_tokens 0 reports invalid_max_tokens, not an out-of-range body"

# ------------------------------------------------------------- text smokes --
info "non-streaming completion"
CODE=$(post text "$(python3 -c '
import json, os
print(json.dumps({"model": os.environ["MODEL"],
                  "messages": [{"role": "user",
                                "content": "Reply with exactly: TEXT_RESPONSE_OK"}],
                  "max_tokens": int(os.environ["MAX_TOKENS"]),
                  "temperature": 0.0}))')")
[ "$CODE" = "200" ] && pass "non-streaming completion returned 200" \
    || fail "non-streaming completion returned $CODE"
json_check "$OUT/text.json" \
    "body['object'] == 'chat.completion' and body['choices'][0]['message']['role'] == 'assistant' and body['usage']['total_tokens'] == body['usage']['prompt_tokens'] + body['usage']['completion_tokens'] and body['choices'][0]['finish_reason'] in ('stop', 'length')" \
    "non-streaming body matches the OpenAI chat.completion shape"
json_check "$OUT/text.json" \
    "'TEXT_RESPONSE_OK' in (body['choices'][0]['message'].get('content') or '')" \
    "non-streaming completion produced the requested marker"
json_check "$OUT/text.json" \
    "isinstance(body['choices'][0]['message'].get('reasoning'), str) and len(body['choices'][0]['message']['reasoning']) > 0" \
    "reasoning is reported separately from content"

info "streaming completion"
curl -sS --max-time "$REQUEST_TIMEOUT" -N -H 'Content-Type: application/json' \
    -X POST "$BASE/chat/completions" --data-binary "$(python3 -c '
import json, os
print(json.dumps({"model": os.environ["MODEL"],
                  "messages": [{"role": "user",
                                "content": "Reply with exactly: STREAM_RESPONSE_OK"}],
                  "max_tokens": int(os.environ["MAX_TOKENS"]),
                  "temperature": 0.0,
                  "stream": True,
                  "stream_options": {"include_usage": True}}))')" \
    > "$OUT/stream.sse" 2>>"$OUT/curl.err"

if python3 - "$OUT/stream.sse" <<'PY'
import json, sys
frames = []
done = False
for line in open(sys.argv[1]):
    line = line.strip()
    if not line.startswith("data: "):
        continue
    payload = line[6:]
    if payload == "[DONE]":
        done = True
        continue
    frames.append(json.loads(payload))

chunks = [f for f in frames if f.get("object") == "chat.completion.chunk"]
usage_frames = [f for f in frames if f.get("object") == "chat.completion" and not f.get("choices")]
text = "".join(
    c["choices"][0]["delta"].get("content", "") for c in chunks if c.get("choices"))
finals = [c for c in chunks if c["choices"][0].get("finish_reason")]

problems = []
if not done:
    problems.append("no [DONE] terminator")
if not chunks:
    problems.append("no chat.completion.chunk frames")
if len(finals) != 1:
    problems.append(f"expected exactly one finish_reason frame, got {len(finals)}")
if len(usage_frames) != 1:
    problems.append(f"expected exactly one usage frame, got {len(usage_frames)}")
elif usage_frames[0]["usage"]["completion_tokens"] <= 0:
    problems.append("usage frame reports no completion tokens")
if "STREAM_RESPONSE_OK" not in text:
    problems.append(f"marker missing from streamed content: {text!r}")
if any("</think>" in (c["choices"][0]["delta"].get("content") or "") for c in chunks):
    problems.append("reasoning marker leaked into streamed content")

print(json.dumps({"frames": len(frames), "chunks": len(chunks),
                  "finish": finals[0]["choices"][0]["finish_reason"] if finals else None,
                  "text": text[:200]}, indent=2))
for problem in problems:
    print("    " + problem)
sys.exit(1 if problems else 0)
PY
then
    pass "streaming completion matches the SSE contract"
else
    fail "streaming completion did not match the SSE contract"
fi

# ------------------------------------------------------------- tool smoke --
info "tool-call completion"
CODE=$(post tool "$(python3 -c '
import json, os
print(json.dumps({
  "model": os.environ["MODEL"],
  "messages": [
    {"role": "user",
     "content": "Write the exact text TOOL_ROUNDTRIP_OK to the file /tmp/mlx-swift-smoke.txt. Use the write_file tool."}],
  "tools": [{"type": "function", "function": {
      "name": "write_file",
      "description": "Write text content to an absolute file path.",
      "parameters": {"type": "object",
                     "properties": {"path": {"type": "string"},
                                    "content": {"type": "string"}},
                     "required": ["path", "content"]}}}],
  "max_tokens": int(os.environ["MAX_TOKENS"]) * 3,
  "temperature": 0.0}))')")
[ "$CODE" = "200" ] && pass "tool-call completion returned 200" \
    || fail "tool-call completion returned $CODE"
json_check "$OUT/tool.json" \
    "body['choices'][0]['finish_reason'] == 'tool_calls'" \
    "tool-call completion finishes with tool_calls"
json_check "$OUT/tool.json" \
    "(lambda calls: bool(calls) and calls[0]['type'] == 'function' and bool(calls[0]['id']) and calls[0]['function']['name'] == 'write_file' and isinstance(json.loads(calls[0]['function']['arguments']), dict))(body['choices'][0]['message'].get('tool_calls') or [])" \
    "tool_calls payload is well formed"
if [ -s "$OUT/tool.json" ]; then
    info "tool_calls: $(python3 -c '
import json, sys
body = json.load(open(sys.argv[1]))
print(json.dumps(body["choices"][0]["message"].get("tool_calls"), indent=None))' "$OUT/tool.json")"
fi

# --------------------------------------------------------------- shutdown --
info "sending SIGTERM to the process group"
STOP_START=$SECONDS
kill -TERM -- "-$HARNESS_PID" 2>/dev/null || kill -TERM "$HARNESS_PID"
STOPPED=0
while [ $((SECONDS - STOP_START)) -lt "$SHUTDOWN_TIMEOUT" ]; do
    if ! kill -0 "$HARNESS_PID" 2>/dev/null; then
        STOPPED=1
        break
    fi
    sleep 1
done
wait "$HARNESS_PID" 2>/dev/null
RUNTIME_EXIT=$?
if [ "$STOPPED" -eq 1 ]; then
    pass "runtime exited $((SECONDS - STOP_START))s after SIGTERM (status $RUNTIME_EXIT)"
else
    fail "runtime still alive ${SHUTDOWN_TIMEOUT}s after SIGTERM"
fi
trap - EXIT

sleep 1
if lsof -nP -iTCP:"$PORT" -sTCP:LISTEN >/dev/null 2>&1; then
    fail "port $PORT still has a listener after shutdown"
else
    pass "port $PORT released after shutdown"
fi

if grep -q '"event":"stopped"' "$RUNTIME_LOG"; then
    pass "runtime emitted a stopped event"
else
    fail "runtime did not emit a stopped event"
fi

printf '\n%s\n' "----------------------------------------"
if [ "$FAILURES" -eq 0 ]; then
    printf 'SMOKE OK (0 failures)\n'
    exit 0
fi
printf 'SMOKE FAILED (%d failures)\n' "$FAILURES"
exit 1
