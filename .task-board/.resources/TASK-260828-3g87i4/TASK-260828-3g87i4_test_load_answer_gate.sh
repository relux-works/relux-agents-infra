#!/bin/bash
# Negative tests for load_and_answer.sh -- the attestation that says the staged
# GGUF really loads and really answers.
#
# Every case drives the REAL script (./load_and_answer.sh, the exact artifact
# attached to the board), swapping only the server binary via LLAMA_SERVER_BIN,
# exactly as the reviewer did when they proved the rev-1 script exited 0 on a
# wrong answer. The prompt and the required answer are not reachable from here;
# the script decides.
#
# An attestation that exits 0 for a wrong answer attests nothing, so each mutant
# below MUST be rejected -- and rejected FOR THE RIGHT REASON: every case asserts
# the EXACT exit code. A case that trips the host-contention guard (3) instead of
# the answer gate is a failure here, not a pass.
#
# Contention scoping. This host is shared with other tracked runs, and the real
# guard refuses whenever anything listens in 18000-18999 or a local model runtime
# is alive -- which is correct for a 29 GB load and fatal for fakes that load
# nothing. The answer-gate cases therefore narrow CONTENTION_SCAN/CONTENTION_PROCS
# to sentinels this suite plants itself, so cross-case leakage is still caught
# while a neighbouring run is neither disturbed nor able to turn a case green.
# Cases 9a and 9c then run at the UNTOUCHED defaults and must refuse, and 9b is
# their control.
#
#   ./test_load_answer_gate.sh
#
# No model is loaded and no llama.cpp binary is needed: the fakes answer instantly.
set -u
HERE="$(cd "$(dirname "$0")" && pwd)"
SCRIPT="$HERE/load_and_answer.sh"
FAKE="$HERE/fake_llama_server.py"
DEAD="$HERE/dead_llama_server.sh"
WORK="$HERE/.load-gate-work"
rm -rf "$WORK"; mkdir -p "$WORK"
FAILURES=0
CASES=0

# This suite's own port block, and a process pattern nothing on this host matches.
SUITE_SCAN='1893[0-9]'
SUITE_PROCS='load-gate-sentinel-4f1a9c'

port_for() { echo $((18930 + $1)); }

wait_suite_ports_free() {
  local held
  for _ in $(seq 1 80); do
    held=$(lsof -nP -iTCP -sTCP:LISTEN 2>/dev/null | awk -v pat=":${SUITE_SCAN}\$" '$9 ~ pat {print $9}')
    [ -z "$held" ] && return 0
    sleep 0.25
  done
  echo "    WARNING: suite ports still held by: $held"
}

report() { # name rc want
  if [ "$2" = "$3" ]; then
    echo "  [PASS] $1: exit=$2 (expected $3)"
  else
    echo "  [FAIL] $1: exit=$2, expected exactly $3"
    [ "$2" = "3" ] && echo "         exit 3 is the host-contention guard, NOT the answer gate"
    FAILURES=$((FAILURES + 1))
  fi
}

run_case() { # name expected_exit bin fake_mode case_index
  local name="$1" want="$2" bin="$3" mode="$4" idx="$5"
  local port; port=$(port_for "$idx")
  local out="$WORK/case-$idx"
  CASES=$((CASES + 1))
  mkdir -p "$out"
  wait_suite_ports_free
  FAKE_MODE="$mode" LLAMA_SERVER_BIN="$bin" MODEL=/dev/null PORT="$port" \
    OUT="$out" REQUIRE_FREE_GIB=0 CONTENTION_SCAN="$SUITE_SCAN" \
    CONTENTION_PROCS="$SUITE_PROCS" bash "$SCRIPT" > "$out/run.log" 2>&1
  local rc=$?
  wait_suite_ports_free
  report "$name" "$rc" "$want"
  [ "$rc" = "$want" ] || sed -n '/^FAIL\|^curl rc\|^finish_reason\|^content\|^HOST BUSY\|^listeners:\|^processes:/p' \
    "$out/run.log" | head -4 | sed 's/^/         /'
}

# exit codes: 3 host busy  5 server died  7 curl failed  8 bad body  9 finish_reason  10 answer
echo "load_and_answer.sh must fail closed -- driving the real script with fakes"
run_case "confident WRONG answer is rejected"        10 "$FAKE" wrong      1
run_case "finish_reason 'length' is rejected"         9 "$FAKE" truncated  2
run_case "empty completion is rejected"              10 "$FAKE" empty      3
run_case "response without choices[] is rejected"     8 "$FAKE" malformed  4
run_case "non-JSON body is rejected"                  8 "$FAKE" notjson    5
run_case "zero completion tokens is rejected"         8 "$FAKE" zerotokens 6
run_case "server that exits immediately is rejected"  5 "$DEAD" right      7

echo
echo "positive control -- the gate is not 'always reject'"
run_case "correct answer is ACCEPTED"                 0 "$FAKE" right      8

echo
echo "host-contention guard -- at the UNTOUCHED defaults"
wait_suite_ports_free

# 9a: a listener inside the default 18000-18999 scan. The assertion names THIS
# port, so an unrelated runtime elsewhere in the range cannot make it pass.
HOLD_PORT=18950
python3 -c "
import socket, sys, time
s = socket.socket(); s.bind(('127.0.0.1', $HOLD_PORT)); s.listen(1)
sys.stderr.write('holding\n'); sys.stderr.flush()
time.sleep(120)
" 2>"$WORK/hold.err" &
HOLD=$!
# 9c: a process matching the default CONTENTION_PROCS pattern.
sleep 120 &
SLEEPER=$!
PLANT="$WORK/runtime-benchmark.py"
printf '#!/bin/bash\nsleep 120\n' > "$PLANT"; chmod +x "$PLANT"
"$PLANT" &
PROCPLANT=$!
trap 'kill $HOLD $SLEEPER $PROCPLANT 2>/dev/null' EXIT
for _ in $(seq 1 40); do
  lsof -nP -iTCP:$HOLD_PORT -sTCP:LISTEN >/dev/null 2>&1 && break
  sleep 0.25
done

CASES=$((CASES + 1))
FAKE_MODE=right LLAMA_SERVER_BIN="$FAKE" MODEL=/dev/null PORT=18931 \
  OUT="$WORK/case-9a" REQUIRE_FREE_GIB=0 CONTENTION_PROCS="$SUITE_PROCS" \
  bash "$SCRIPT" > "$WORK/case-9a.log" 2>&1
RC=$?
if [ "$RC" -eq 3 ] && grep -q ":$HOLD_PORT" "$WORK/case-9a.log"; then
  echo "  [PASS] default port scan refuses and names :$HOLD_PORT: exit=3"
else
  echo "  [FAIL] default port scan must refuse and name :$HOLD_PORT: exit=$RC"
  sed -n '/listeners:/p' "$WORK/case-9a.log" | sed 's/^/         /'
  FAILURES=$((FAILURES + 1))
fi

CASES=$((CASES + 1))
FAKE_MODE=right LLAMA_SERVER_BIN="$FAKE" MODEL=/dev/null PORT=18931 \
  OUT="$WORK/case-9c" REQUIRE_FREE_GIB=0 CONTENTION_SCAN="$SUITE_SCAN" \
  bash "$SCRIPT" > "$WORK/case-9c.log" 2>&1
RC=$?
if [ "$RC" -eq 3 ] && grep -q "runtime-benchmark.py" "$WORK/case-9c.log"; then
  echo "  [PASS] default process scan refuses and names the planted runtime: exit=3"
else
  echo "  [FAIL] default process scan must refuse and name the planted runtime: exit=$RC"
  sed -n '/processes:/p' "$WORK/case-9c.log" | head -2 | sed 's/^/         /'
  FAILURES=$((FAILURES + 1))
fi

# 9b: control. Same invocation, both scans narrowed away from the plants. Without
# this, 9a and 9c only show the script exiting 3 in the presence of noise.
CASES=$((CASES + 1))
wait_suite_ports_free
FAKE_MODE=right LLAMA_SERVER_BIN="$FAKE" MODEL=/dev/null PORT=18931 \
  OUT="$WORK/case-9b" REQUIRE_FREE_GIB=0 CONTENTION_SCAN="$SUITE_SCAN" \
  CONTENTION_PROCS="$SUITE_PROCS" bash "$SCRIPT" > "$WORK/case-9b.log" 2>&1
RC=$?
wait_suite_ports_free
report "with the plants outside both scans the check runs" "$RC" 0
[ "$RC" = "0" ] || sed -n '/^HOST BUSY\|^listeners:\|^processes:/p' "$WORK/case-9b.log" | head -3 | sed 's/^/         /'

kill $HOLD $SLEEPER $PROCPLANT 2>/dev/null
wait $HOLD $SLEEPER $PROCPLANT 2>/dev/null

echo
echo "$CASES case(s), $FAILURES failure(s)"
[ "$FAILURES" -eq 0 ] || exit 1
exit 0
