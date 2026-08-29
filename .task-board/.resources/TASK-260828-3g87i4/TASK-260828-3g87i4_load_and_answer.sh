#!/bin/bash
# Bounded load-and-answer check for the staged GGUF through llama-server.
#
# This is an ATTESTATION, not a demo: it must exit non-zero unless a real server
# really answered the bounded question correctly. It fails closed on a curl
# error, a malformed or unexpected response body, a finish_reason other than
# "stop", an empty completion, a completion that does not contain the expected
# answer, and on an early server exit. test_load_answer_gate.sh drives this exact
# script with fakes for each of those and requires non-zero every time.
#
# It also refuses to start while another run holds the host.
#
# Overridable ONLY for that negative test (defaults are the real check):
#   LLAMA_SERVER_BIN   server binary to launch          (default: the pinned llama.cpp)
#   MODEL              model path                       (default: the staged Q8_0)
#   PORT               port to bind                     (default: 18901)
#   OUT                output directory                 (default: this script's dir)
#   REQUIRE_FREE_GIB   reclaimable-memory floor         (default: 35)
#   CONTENTION_SCAN    port pattern the guard scans     (default: the whole 18000-18999
#                                                        range shared by local runtimes)
#   CONTENTION_PROCS   process pattern the guard scans  (default: the local runtimes)
# The three CONTENTION_*/REQUIRE_* knobs exist because the fakes load no model and
# no GPU, so they must neither fight nor be blocked by real runtimes sharing this
# host. The negative suite narrows them to sentinels it plants itself, and runs
# extra cases at the UNTOUCHED defaults -- with a held port and with a matching
# process -- to prove the guard still refuses there.
# The prompt and the answer the response must contain are NOT overridable --
# there is no way to make this script accept a wrong answer.
set -u
MODEL=${MODEL:-/Users/alexis/src/local-models/Qwen3.8-27B-Uncensored-GGUF-Q8_0/Qwen3.8-27B-Uncensored-OrcaRouter-Q8_0.gguf}
PORT=${PORT:-18901}
OUT=${OUT:-$(cd "$(dirname "$0")" && pwd)}
LOG="$OUT/llama-server-01.log"
BIN=${LLAMA_SERVER_BIN:-/opt/homebrew/opt/llama.cpp/bin/llama-server}
REQUIRE_FREE_GIB=${REQUIRE_FREE_GIB:-35}
CONTENTION_SCAN=${CONTENTION_SCAN:-18[0-9][0-9][0-9]}
CONTENTION_PROCS=${CONTENTION_PROCS:-mlx_lm|mlx-swift-runtime-prototype|runtime-benchmark.py}
mkdir -p "$OUT"

echo "== host contention check =="
BUSY=$(lsof -nP -iTCP -sTCP:LISTEN 2>/dev/null | awk -v pat=":${CONTENTION_SCAN}\$" '$9 ~ pat {print $1, $9}')
PROC=$(pgrep -fl "$CONTENTION_PROCS" | grep -v grep)
if [ -n "$BUSY" ] || [ -n "$PROC" ]; then
  echo "HOST BUSY — refusing to load."
  [ -n "$BUSY" ] && echo "listeners: $BUSY"
  [ -n "$PROC" ] && echo "processes: $PROC"
  exit 3
fi
FREE_GIB=$(vm_stat | awk '/Pages free/{f=$3} /Pages inactive/{i=$3} /Pages speculative/{s=$3} END{gsub(/\./,"",f);gsub(/\./,"",i);gsub(/\./,"",s); printf "%.1f", (f+i+s)*16384/1073741824}')
echo "free+inactive+speculative: ${FREE_GIB} GiB (floor ${REQUIRE_FREE_GIB})"
awk -v g="$FREE_GIB" -v need="$REQUIRE_FREE_GIB" 'BEGIN{ if (g < need) { print "INSUFFICIENT MEMORY — refusing to load."; exit 1 } }' || exit 4

echo "== starting llama-server =="
"$BIN" -m "$MODEL" --host 127.0.0.1 --port "$PORT" -c 4096 --jinja -n 128 > "$LOG" 2>&1 &
SRV=$!
trap 'kill $SRV 2>/dev/null' EXIT
echo "pid=$SRV log=$LOG"

echo "== waiting for readiness (max 600s) =="
READY=0
for i in $(seq 1 120); do
  if ! kill -0 $SRV 2>/dev/null; then echo "FAIL: server exited early"; tail -30 "$LOG"; exit 5; fi
  CODE=$(curl -s -o /dev/null -w '%{http_code}' "http://127.0.0.1:$PORT/health" 2>/dev/null)
  if [ "$CODE" = "200" ]; then READY=1; echo "ready after ~$((i*5))s"; break; fi
  sleep 5
done
[ "$READY" = "1" ] || { echo "FAIL: NOT READY within 600s"; tail -30 "$LOG"; exit 6; }

echo "== one bounded prompt =="
cat > "$OUT/request.json" <<'JSON'
{
  "model": "local",
  "messages": [{"role": "user", "content": "Reply with exactly one sentence: what is the capital of Armenia?"}],
  "max_tokens": 96,
  "temperature": 0,
  "chat_template_kwargs": {"enable_thinking": false}
}
JSON
curl -s --max-time 300 -H 'Content-Type: application/json' \
  -d @"$OUT/request.json" "http://127.0.0.1:$PORT/v1/chat/completions" > "$OUT/response.json"
RC=$?
echo "curl rc=$RC"
if [ "$RC" -ne 0 ]; then echo "FAIL: curl exited $RC"; kill $SRV 2>/dev/null; exit 7; fi

# The server must still be alive: a body written by a process that has since died
# is not evidence that the runtime works.
if ! kill -0 $SRV 2>/dev/null; then
  echo "FAIL: server died before the response could be validated"; tail -30 "$LOG"; exit 5
fi

# Assert the answer. Every branch below exits non-zero; nothing here is advisory.
#   8  malformed / unexpected response body
#   9  finish_reason is not "stop"
#  10  empty completion, or the completion does not contain the expected answer
python3 - "$OUT/response.json" <<'PY'
import json, re, sys

EXPECT = re.compile(r"\byerevan\b", re.I)   # the bounded question has exactly one right answer

try:
    with open(sys.argv[1]) as fh:
        d = json.load(fh)
    ch = d["choices"][0]
    content = ch["message"]["content"]
except Exception as e:
    print(f"FAIL: malformed response body: {type(e).__name__}: {e}")
    sys.exit(8)

fin = ch.get("finish_reason")
usage = d.get("usage") or {}
print("finish_reason:", fin)
print("content:", repr(content))
print("usage:", usage)

if fin != "stop":
    print(f"FAIL: finish_reason is {fin!r}, expected 'stop'")
    sys.exit(9)
if not isinstance(content, str) or not content.strip():
    print("FAIL: empty completion")
    sys.exit(10)
if not EXPECT.search(content):
    print(f"FAIL: completion does not contain the expected answer ({EXPECT.pattern}): {content!r}")
    sys.exit(10)
if int(usage.get("completion_tokens") or 0) <= 0:
    print(f"FAIL: usage reports {usage.get('completion_tokens')!r} completion tokens")
    sys.exit(8)
print("OK: bounded answer asserted")
PY
VRC=$?
if [ "$VRC" -ne 0 ]; then echo "answer assertion failed (exit $VRC)"; kill $SRV 2>/dev/null; exit "$VRC"; fi

echo "== stopping =="
kill $SRV 2>/dev/null; wait $SRV 2>/dev/null
grep -iE 'load_tensors|llama_model_loader: - type|print_info: file (type|size)|n_ctx|Metal|MTP|nextn' "$LOG" | head -40
echo "== PASS: load-and-answer attested =="
exit 0
