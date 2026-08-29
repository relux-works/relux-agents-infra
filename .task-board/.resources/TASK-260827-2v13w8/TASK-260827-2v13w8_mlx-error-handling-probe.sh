#!/usr/bin/env bash
# Production test for MLX error handling around generation.
#
# Before this revision an MLX C++ failure called `fatalError` and killed the
# process, so `GenerationEngine.generate`'s catch, the batch ledger, the health
# transition and the supervision marker were all unreachable for the one class
# of failure they exist for. Review reproduced exactly that with a 73k-token
# prompt.
#
# The failure is provoked here on purpose, on a supported path:
# `--model-factory vision-first` selects `MLXVLM.Qwen35VLModel`, whose
# `prepare(_:cache:windowSize:)` discards its window and evaluates the whole
# prompt in one call. On 73k tokens that is a single 255,904,140,288-byte
# allocation against a 41,747,087,360-byte Metal buffer limit.
#
# What is under test is not that the allocation fails -- it must -- but what the
# runtime does about it:
#
#   * the request gets a 500 rather than the process getting a SIGTRAP;
#   * `/health` stays 200, because an oversized single allocation is
#     request-scoped: it leaves MLX's pool intact and belongs to the request
#     that asked for it (`GenerationWorkerHealth.invalidatingSignatures`
#     deliberately does not list it);
#   * the process is still serving afterwards.
set -uo pipefail
BINARY="${BINARY:?set BINARY}"
MODEL="${MODEL:-/Users/alexis/src/local-models/Qwen3.8-27B-Uncensored-MLX-8bit}"
PORT="${PORT:-18042}"
OUT="${OUT:-./mlx-error-out}"
PROMPTS="${PROMPTS:-examples/benchmark-prompts.json}"
mkdir -p "$OUT"

FAILURES=0
pass() { printf 'PASS  %s\n' "$*"; }
fail() { printf 'FAIL  %s\n' "$*"; FAILURES=$((FAILURES + 1)); }

"$BINARY" serve --model "$MODEL" --port "$PORT" \
    --model-factory vision-first --prefill-step-size 2048 \
    --reasoning-effort medium --default-max-tokens 2048 > "$OUT/runtime.log" 2>&1 &
RUNTIME_PID=$!
trap 'kill -TERM $RUNTIME_PID 2>/dev/null' EXIT

for _ in $(seq 1 120); do
    [ "$(curl -s -m 5 -o /dev/null -w '%{http_code}' "http://127.0.0.1:$PORT/v1/models")" = "200" ] && break
    sleep 1
done
grep -q '"factory":"MLXVLM.VLMModelFactory"' "$OUT/runtime.log" \
    && pass "vision-first selected MLXVLM.VLMModelFactory" \
    || fail "vision-first did not select the VLM factory"

python3 - "$PORT" "$PROMPTS" "$MODEL" > "$OUT/request.json" 2>&1 <<'DROP'
import json, sys, urllib.request, urllib.error, time
port, prompts, model = sys.argv[1], sys.argv[2], sys.argv[3]
suite = json.load(open(prompts))
spec = suite["scenarios"]["context_75k"]
content = suite["filler_paragraph"] * spec["prefix_repeats"] + "\n\n" + spec["prompt"]
body = {"model": model,
        "messages": [{"role": "system", "content": suite["system_prompt"]},
                     {"role": "user", "content": content}],
        "max_tokens": spec["max_tokens"], "temperature": 0.0, "top_p": 1.0, "seed": 1234}
req = urllib.request.Request(f"http://127.0.0.1:{port}/v1/chat/completions",
                             data=json.dumps(body).encode(),
                             headers={"Content-Type": "application/json"}, method="POST")
started = time.time()
try:
    with urllib.request.urlopen(req, timeout=3600) as r:
        print(json.dumps({"status": r.status, "body": r.read().decode()[:400],
                          "seconds": time.time() - started}))
except urllib.error.HTTPError as e:
    print(json.dumps({"status": e.code, "body": e.read().decode()[:600],
                      "seconds": time.time() - started}))
except Exception as e:
    print(json.dumps({"status": 0, "body": f"{type(e).__name__}: {e}",
                      "seconds": time.time() - started}))
DROP

STATUS="$(python3 -c "import json,sys;print(json.load(open(sys.argv[1]))['status'])" "$OUT/request.json" 2>/dev/null)"
if [ "$STATUS" = "500" ]; then
    pass "the oversized allocation came back as HTTP 500, not a process trap"
else
    fail "expected HTTP 500 from the oversized allocation, got '$STATUS'"
fi
grep -qF "metal::malloc" "$OUT/request.json" \
    && pass "the 500 carries MLX's own allocator message verbatim" \
    || fail "the 500 does not carry the allocator message"

if kill -0 "$RUNTIME_PID" 2>/dev/null; then
    pass "the runtime process survived the MLX failure"
else
    fail "the runtime process died (this is the pre-fix behaviour)"
fi
grep -q "Fatal error" "$OUT/runtime.log" \
    && fail "the runtime still trapped" \
    || pass "no fatalError in the runtime log"

HEALTH="$(curl -s -m 5 -o "$OUT/health.json" -w '%{http_code}' "http://127.0.0.1:$PORT/health")"
if [ "$HEALTH" = "200" ]; then
    pass "/health stayed 200: an oversized single allocation is request-scoped"
else
    fail "/health answered $HEALTH; the allocator's oversized-request throw must not condemn the worker"
fi

FOLLOWUP="$(curl -s -m 120 -o "$OUT/followup.json" -w '%{http_code}' \
    -H 'Content-Type: application/json' \
    -d "{\"model\":\"$MODEL\",\"messages\":[{\"role\":\"user\",\"content\":\"Say OK.\"}],\"max_tokens\":4,\"temperature\":0.0}" \
    "http://127.0.0.1:$PORT/v1/chat/completions")"
if [ "$FOLLOWUP" = "200" ]; then
    pass "a following request on the same process succeeded"
else
    fail "the following request answered $FOLLOWUP"
fi

printf '\n%s\n' "----------------------------------------"
if [ "$FAILURES" -eq 0 ]; then
    printf 'MLX ERROR HANDLING PROBE OK (0 failures)\n'; exit 0
fi
printf 'MLX ERROR HANDLING PROBE FAILED (%d failures)\n' "$FAILURES"; exit 1
