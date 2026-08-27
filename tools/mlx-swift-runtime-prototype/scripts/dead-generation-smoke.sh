#!/usr/bin/env bash
# Dead-generation-worker health regression, end to end (TASK-260827-2h39ya).
#
# Carries `mlx_lm.server`'s dead-generation-thread regression (BUG-260827-1jhv2g,
# BUG-260827-2tul5n) into the MLX Swift runtime's acceptance suite: a runtime
# that cannot produce a token must not answer `GET /health` with 200.
#
# Runs the real Release binary under the real `model-harness`, against a real
# model, and checks four things:
#
#   1. control      -- a healthy run of the same build answers /health 200
#   2. fault        -- an invalidating generation failure flips /health to 503,
#                      unsupervised, so the window is observable rather than
#                      raced against the supervisor's kill; run once through
#                      the buffered path and once through the streaming one,
#                      because they are two separate production call sites
#   3. supervision  -- the same fault under a fatal-substring policy: the
#                      harness restarts it and the replacement answers 200
#   4a. benign      -- a request-scoped failure does NOT flip /health or restart
#   4b. overbroad   -- a request-scoped failure that REUSES a fragment of the
#                      backend message (`Resource limit`, no Metal allocator
#                      context) also does NOT flip /health or restart
#
# (4a) is what makes (2) mean anything. Without it, a runtime that condemned
# itself on every error would pass 1-3 and be strictly worse than the bug.
#
# (4b) is what makes (4a) mean anything. A classifier that condemns on any
# single loose fragment of the incident message passes 1-4a and still takes a
# healthy runtime out of rotation -- that was the shipped defect on revision 1.
# Deleting the gate and narrowing it too little are different failures, and only
# (4b) separates them.
#
# The fault is injected with `serve --fault-inject-generation-error <message>`,
# which throws that exact text out of the real generation path. Upstream mlx-lm
# proved its own generation-thread recovery the same way. Nothing about the
# verdict is injected: production classifies the message, so the benign run
# below is a real negative and not a rehearsed one.
#
# Uses a small model on purpose -- the failure path never reaches the weights,
# so paying 29 GB and six seconds of load to observe it would only mean the
# check cannot run while the default Python runtime is resident.
#
#   BINARY=./DerivedData/Build/Products/Release/mlx-swift-runtime-prototype \
#   HARNESS=/Users/alexis/.local/bin/model-harness \
#   MODEL=/Users/alexis/.cache/huggingface/hub/models--mlx-community--Qwen1.5-0.5B-Chat-4bit/snapshots/659d8dafc39202a6688bb46242d60440702489b1 \
#   PORT=18019 OUT=./dead-generation-out \
#   scripts/dead-generation-smoke.sh

set -uo pipefail
# Job control, so each managed run becomes its own process group and teardown
# can signal the whole tree. `model-harness` does not forward signals to the
# runtime it spawns, so signalling the harness alone orphans a live listener --
# and an orphan holding this port is exactly how a later check could pass by
# talking to the wrong process.
set -m

BINARY="${BINARY:?set BINARY to the xcodebuild Release product}"
HARNESS="${HARNESS:?set HARNESS to the model-harness executable}"
MODEL="${MODEL:?set MODEL to an absolute local model directory}"
PORT="${PORT:-18019}"
HOST=127.0.0.1
OUT="${OUT:-./dead-generation-out}"
BASE="http://${HOST}:${PORT}"

# Verbatim from the BUG-260827-1jhv2g incident record.
INCIDENT='RuntimeError: [metal::malloc] Resource limit (499000) exceeded'
# A failure that belongs to the request, not to the backend.
BENIGN='generation ended without completion info; token usage is unknown'
# The over-broad neighbour. Shares half of the incident message -- the words
# `Resource limit` -- and none of its Metal allocator context. Found by review
# on revision 1, where a bare `Resource limit` substring was fatal on its own
# and this message condemned a healthy runtime through the production entry
# point. Kept verbatim from that reproduction.
OVERBROAD='RequestError: Resource limit for this request is 8 tokens'
MARKER='generation_worker_unavailable'

case "$BINARY" in /*) ;; *) BINARY="$PWD/${BINARY#./}" ;; esac
case "$OUT" in /*) ;; *) OUT="$PWD/${OUT#./}" ;; esac

mkdir -p "$OUT"
CONFIG="$OUT/model-harness-dead-generation.toml"

FAILURES=0
pass() { printf 'PASS  %s\n' "$*"; }
fail() { printf 'FAIL  %s\n' "$*"; FAILURES=$((FAILURES + 1)); }
info() { printf 'INFO  %s\n' "$*"; }

status_of() { curl -sS --max-time 10 -o "$2" -w '%{http_code}' "$1" 2>/dev/null; }

chat() {
    curl -sS --max-time 60 -o "$2" -w '%{http_code}' \
        -H 'Content-Type: application/json' -X POST "$BASE/v1/chat/completions" \
        --data-binary "{\"model\":\"$MODEL\",\"messages\":[{\"role\":\"user\",\"content\":\"hi\"}],\"max_tokens\":8}" \
        2>/dev/null
}

# ------------------------------------------------------------------ config --
# Supervision is deliberately narrow: `restart_on_failure = false` means the
# ONLY thing that can restart this runtime is the fatal marker. A restart
# observed below therefore proves the marker did it, not that the process
# happened to exit.
write_config() {
    local profile=$1 injection=$2 supervised=$3
    local injection_argv=""
    if [ -n "$injection" ]; then
        injection_argv=$(printf '    "--fault-inject-generation-error", "%s",\n' "$injection")
    fi
    cat > "$CONFIG" <<TOML
# Task-scoped acceptance config for TASK-260827-2h39ya. NOT installed.
[profiles.$profile]
mode = "local"
executable = "$BINARY"
argv = [
    "serve",
    "--model", "$MODEL",
    "--host", "{host}",
    "--port", "{port}",
    "--default-max-tokens", "64",
$injection_argv]
TOML
    [ "$supervised" = "supervised" ] || return 0
    cat >> "$CONFIG" <<TOML

[profiles.$profile.supervision]
fatal_output_substrings = ["$MARKER"]
restart_on_failure = false
max_restarts = 2
restart_window_seconds = 120
restart_delay_milliseconds = 250
TOML
}

port_free() { ! lsof -nP -iTCP:"$PORT" -sTCP:LISTEN >/dev/null 2>&1; }

wait_ready() {
    local log=$1 limit=${2:-120} code
    for _ in $(seq 1 "$limit"); do
        code=$(status_of "$BASE/v1/models" /dev/null)
        [ "$code" = "200" ] && return 0
        sleep 0.5
    done
    info "runtime never became ready; last /v1/models code=${code:-none}"
    tail -20 "$log" >&2
    return 1
}

start_harness() {
    local profile=$1 log=$2
    # Refuse to start on an occupied port. Every check below reads the port, so
    # a leftover listener from an earlier phase would answer for a runtime that
    # was never launched and turn this suite into theatre.
    if ! port_free; then
        fail "port $PORT still has a listener; refusing to start $profile"
        exit 1
    fi
    "$HARNESS" run "$profile" --config "$CONFIG" --host "$HOST" --port "$PORT" > "$log" 2>&1 &
    HARNESS_PID=$!
}

stop_harness() {
    [ -n "${HARNESS_PID:-}" ] || return 0
    local children
    children=$(pgrep -P "$HARNESS_PID" 2>/dev/null)
    kill -TERM -- "-$HARNESS_PID" 2>/dev/null \
        || kill -TERM "$HARNESS_PID" $children 2>/dev/null
    for _ in $(seq 1 80); do
        port_free && ! kill -0 "$HARNESS_PID" 2>/dev/null && break
        sleep 0.25
    done
    kill -KILL -- "-$HARNESS_PID" 2>/dev/null \
        || kill -KILL "$HARNESS_PID" $children 2>/dev/null
    wait "$HARNESS_PID" 2>/dev/null
    HARNESS_PID=""
    for _ in $(seq 1 40); do port_free && break; sleep 0.25; done
}

trap 'stop_harness' EXIT

if ! port_free; then
    fail "port $PORT already has a listener"
    exit 1
fi

# ================================================== 1. healthy control run ==
info "control: same build, no injected fault"
write_config prototype-control "" supervised
start_harness prototype-control "$OUT/control.log"
if wait_ready "$OUT/control.log"; then
    pass "control run reached ready"

    CODE=$(status_of "$BASE/health" "$OUT/control-health.json")
    if [ "$CODE" = "200" ]; then
        pass "control /health answers 200"
    else
        fail "control /health answered $CODE (want 200)"
    fi
    python3 -c '
import json, sys
body = json.load(open(sys.argv[1]))
assert body == {"status": "ok"}, body
' "$OUT/control-health.json" 2>/dev/null \
        && pass "control /health body is {\"status\":\"ok\"}" \
        || fail "control /health body is not {\"status\":\"ok\"}"

    CODE=$(chat "$MODEL" "$OUT/control-chat.json")
    [ "$CODE" = "200" ] && pass "control completion answers 200" \
        || fail "control completion answered $CODE (want 200)"

    CODE=$(status_of "$BASE/health" "$OUT/control-health-after.json")
    [ "$CODE" = "200" ] && pass "control /health still 200 after a real generation" \
        || fail "control /health answered $CODE after a real generation"
else
    fail "control run never became ready"
fi
grep -q "$MARKER" "$OUT/control.log" \
    && fail "control run emitted the supervision marker" \
    || pass "control run never emitted the supervision marker"
stop_harness

# ======================== 2. injected backend failure: the health semantics ==
# Deliberately UNSUPERVISED. With a fatal-substring policy attached,
# model-harness kills the runtime within milliseconds of the marker reaching
# its stdout -- correct behaviour, and it destroys the very window this phase
# has to observe. The 503 belongs to the runtime and the restart belongs to the
# supervisor; measuring them in one process would only measure which won the
# race. Phase 3 attaches supervision and measures the other half.
info "fault: injecting the BUG-260827-1jhv2g backend failure (unsupervised)"
write_config prototype-fault "$INCIDENT" unsupervised
start_harness prototype-fault "$OUT/fault.log"
if wait_ready "$OUT/fault.log"; then
    pass "fault run reached ready"

    # Same process, before the fault is provoked. This is what makes the 503
    # below a transition rather than a runtime that was never healthy.
    CODE=$(status_of "$BASE/health" "$OUT/fault-health-before.json")
    [ "$CODE" = "200" ] && pass "fault run /health is 200 before the fault" \
        || fail "fault run /health answered $CODE before the fault (want 200)"

    CODE=$(chat "$MODEL" "$OUT/fault-chat.json")
    [ "$CODE" = "500" ] && pass "the injected failure answers the request with 500" \
        || fail "the injected failure answered $CODE (want 500)"
    python3 -c '
import json, sys
body = json.load(open(sys.argv[1]))
assert body["error"]["code"] == "generation_failed", body
assert "metal::malloc" in body["error"]["message"], body
' "$OUT/fault-chat.json" 2>/dev/null \
        && pass "the 500 names the backend failure verbatim" \
        || fail "the 500 did not name the backend failure"

    # -------------------------------------------------- the regression itself
    CODE=$(status_of "$BASE/health" "$OUT/fault-health-after.json")
    if [ "$CODE" = "503" ]; then
        pass "REGRESSION GATE: /health answers 503 with the worker condemned"
    else
        fail "REGRESSION GATE: /health answered $CODE with the worker condemned (want 503)"
    fi
    python3 -c '
import json, sys
body = json.load(open(sys.argv[1]))
assert body["status"] == "unavailable", body
assert "metal::malloc" in body["detail"], body
' "$OUT/fault-health-after.json" 2>/dev/null \
        && pass "condemned /health body is {\"status\":\"unavailable\"}" \
        || fail "condemned /health body is not {\"status\":\"unavailable\"}"

    # The condemnation is terminal, not a one-request blip: a second poll must
    # still be 503. A runtime that healed itself would hand the next caller a
    # backend already known to be broken.
    CODE=$(status_of "$BASE/health" "$OUT/fault-health-again.json")
    [ "$CODE" = "503" ] && pass "the condemned runtime stays 503 on a later poll" \
        || fail "the condemned runtime answered $CODE on a later poll (want 503)"

    CODE=$(status_of "$BASE/v1/models" "$OUT/fault-models.json")
    [ "$CODE" = "503" ] && pass "a condemned runtime stops advertising on /v1/models" \
        || fail "/v1/models answered $CODE with the worker condemned (want 503)"
    if [ ! -s "$OUT/fault-models.json" ]; then
        fail "no listing body was captured from the condemned runtime"
    elif grep -q "$MODEL" "$OUT/fault-models.json"; then
        fail "the model ID leaked into the condemned listing"
    else
        pass "the model ID is absent from the condemned listing"
    fi

    # A subsequent completion is refused at the readiness gate rather than
    # handed the engine that was just condemned.
    CODE=$(chat "$MODEL" "$OUT/fault-chat-after.json")
    [ "$CODE" = "503" ] && pass "a later completion is refused with 503" \
        || fail "a later completion answered $CODE (want 503)"

    grep -q '"event":"generation_worker_failed"' "$OUT/fault.log" \
        && pass "the runtime emitted a generation_worker_failed event" \
        || fail "no generation_worker_failed event in the runtime output"
    grep -q "$MARKER" "$OUT/fault.log" \
        && pass "the emitted output carries the supervision marker" \
        || fail "the emitted output does not carry the supervision marker"
    grep '"event":"generation_worker_failed"' "$OUT/fault.log" | tail -1 \
        > "$OUT/fault-marker.json"
else
    fail "fault run never became ready"
fi
stop_harness

# ============== 2b. the streaming call site condemns the worker as well =====
# `Router.complete` and `RuntimeHTTPHandler.sendStream` are two separate
# production call sites into the same health state. A worker condemned
# mid-stream is exactly as dead as one condemned mid-request, and a runtime
# that only noticed the buffered case would keep answering 200 for the path Pi
# actually streams from. Fresh process: the condemnation above is terminal.
info "fault: same injection, driven through the streaming path"
start_harness prototype-fault "$OUT/fault-stream.log"
if wait_ready "$OUT/fault-stream.log"; then
    pass "streaming fault run reached ready"

    CODE=$(status_of "$BASE/health" "$OUT/stream-health-before.json")
    [ "$CODE" = "200" ] && pass "streaming fault run /health is 200 before the fault" \
        || fail "streaming fault run /health answered $CODE before the fault (want 200)"

    # SSE opens with a 200 head before generation starts, so the failure can
    # only be reported as a terminal frame; the status code says nothing here.
    curl -sS --max-time 60 -o "$OUT/stream-body.txt" \
        -H 'Content-Type: application/json' -X POST "$BASE/v1/chat/completions" \
        --data-binary "{\"model\":\"$MODEL\",\"messages\":[{\"role\":\"user\",\"content\":\"hi\"}],\"max_tokens\":8,\"stream\":true}" \
        >/dev/null 2>&1
    grep -q 'metal::malloc' "$OUT/stream-body.txt" \
        && pass "the stream reports the backend failure as a terminal frame" \
        || fail "the stream did not report the backend failure"

    CODE=$(status_of "$BASE/health" "$OUT/stream-health-after.json")
    if [ "$CODE" = "503" ]; then
        pass "REGRESSION GATE (streaming): /health answers 503 after a condemned stream"
    else
        fail "REGRESSION GATE (streaming): /health answered $CODE (want 503)"
    fi
    grep -q '"event":"generation_worker_failed"' "$OUT/fault-stream.log" \
        && pass "the streaming path emitted a generation_worker_failed event" \
        || fail "no generation_worker_failed event from the streaming path"
else
    fail "streaming fault run never became ready"
fi
stop_harness

# ========================= 3. the same fault under the supervision policy ====
info "supervised: same fault, with fatal_output_substrings configured"
write_config prototype-supervised "$INCIDENT" supervised
start_harness prototype-supervised "$OUT/supervised.log"
if wait_ready "$OUT/supervised.log"; then
    pass "supervised run reached ready"

    # The response to this request is not asserted: the marker reaches the
    # harness while the reply is still being written, so the kill may land
    # first. That the client loses this request is the point of restarting.
    chat "$MODEL" "$OUT/supervised-chat.json" > "$OUT/supervised-chat.code"
    info "provoking request returned HTTP $(cat "$OUT/supervised-chat.code")"

    RESTARTED=0
    for _ in $(seq 1 240); do
        if grep -q 'restarting profile "prototype-supervised"' "$OUT/supervised.log"; then
            RESTARTED=1
            break
        fi
        sleep 0.25
    done
    if [ "$RESTARTED" -eq 1 ]; then
        pass "RECOVERY GATE: model-harness performed the configured supervised restart"
        grep 'restarting profile "prototype-supervised"' "$OUT/supervised.log" | tail -1 \
            > "$OUT/supervised-restart.txt"
        info "restart line recorded in $OUT/supervised-restart.txt"
    else
        fail "RECOVERY GATE: model-harness never restarted the condemned runtime"
    fi
    grep -q 'fatal output "'"$MARKER"'"' "$OUT/supervised.log" \
        && pass "the harness names the marker as the reason it restarted" \
        || fail "the harness restart does not name the marker"

    if wait_ready "$OUT/supervised.log"; then
        pass "the restarted runtime became ready again"
        CODE=$(status_of "$BASE/health" "$OUT/supervised-health-recovered.json")
        [ "$CODE" = "200" ] && pass "the recovered runtime answers /health 200 again" \
            || fail "the recovered runtime answered $CODE (want 200)"
        # Counted only once the replacement is up, or this races the restart.
        if [ "$(grep -c '"event":"listening"' "$OUT/supervised.log")" -ge 2 ]; then
            pass "the restarted runtime bound its listener again"
        else
            fail "no second listening event after the restart"
        fi
    else
        fail "the restarted runtime never became ready"
    fi
else
    fail "supervised run never became ready"
fi
stop_harness

# ========================================= 4. negative: request-scoped faults ==
# Narrowing checks. Everything above is satisfied by a runtime that condemns
# itself on ANY generation error, which would be a worse bug than the one being
# fixed. Each run below injects a failure that belongs to the request and
# requires the runtime to stay in rotation.
#
# Supervision is attached on purpose: these phases have to prove that no
# restart happens, and a policy that is not configured cannot fire.
negative_phase() {
    local profile=$1 injection=$2 slug=$3 label=$4
    info "negative ($slug): $label"
    write_config "$profile" "$injection" supervised
    start_harness "$profile" "$OUT/$slug.log"
    if ! wait_ready "$OUT/$slug.log"; then
        fail "negative run ($slug) never became ready"
        stop_harness
        return
    fi
    pass "negative run ($slug) reached ready"

    CODE=$(chat "$MODEL" "$OUT/$slug-chat.json")
    [ "$CODE" = "500" ] \
        && pass "($slug) the request-scoped failure answers the request with 500" \
        || fail "($slug) the request-scoped failure answered $CODE (want 500)"

    CODE=$(status_of "$BASE/health" "$OUT/$slug-health.json")
    if [ "$CODE" = "200" ]; then
        pass "NARROWING GATE ($slug): a request-scoped failure leaves /health at 200"
    else
        fail "NARROWING GATE ($slug): a request-scoped failure moved /health to $CODE (want 200)"
    fi
    python3 -c '
import json, sys
body = json.load(open(sys.argv[1]))
assert body == {"status": "ok"}, body
' "$OUT/$slug-health.json" 2>/dev/null \
        && pass "($slug) /health body is still {\"status\":\"ok\"}" \
        || fail "($slug) /health body is not {\"status\":\"ok\"}"

    CODE=$(status_of "$BASE/v1/models" "$OUT/$slug-models.json")
    [ "$CODE" = "200" ] && pass "($slug) a request-scoped failure keeps the model advertised" \
        || fail "($slug) /v1/models answered $CODE after a request-scoped failure (want 200)"

    grep -q "$MARKER" "$OUT/$slug.log" \
        && fail "($slug) a request-scoped failure emitted the supervision marker" \
        || pass "($slug) a request-scoped failure emitted no supervision marker"

    # Give supervision the same wall-clock it needed to act in the fault run.
    sleep 2
    grep -q 'restarting profile' "$OUT/$slug.log" \
        && fail "($slug) model-harness restarted a runtime that was still healthy" \
        || pass "($slug) model-harness left the healthy runtime alone"

    # A completion after the refused one must still be served. A runtime that
    # merely deferred the condemnation would pass every check above.
    CODE=$(chat "$MODEL" "$OUT/$slug-chat-after.json")
    [ "$CODE" = "500" ] && pass "($slug) the runtime is still serving requests afterwards" \
        || fail "($slug) a later completion answered $CODE (want 500, the same request-scoped failure)"

    stop_harness
}

# 4a. A failure sharing nothing with a backend message. Proves the gate is not
# condemn-all.
negative_phase prototype-benign "$BENIGN" benign \
    "a failure that belongs to the request"

# 4b. THE NARROWING CASE. Shares the words `Resource limit` with the incident
# and carries no Metal allocator context. A classifier that matches the incident
# by any single loose fragment passes 4a and fails here, which is exactly the
# revision-1 defect: this message took a healthy runtime to 503 and emitted the
# fatal marker. Deleting a signature cannot be told apart from narrowing one
# without this phase.
negative_phase prototype-overbroad "$OVERBROAD" overbroad \
    "a request-scoped failure that reuses a fragment of the backend message"

printf '\n%s\n' "----------------------------------------"
if [ "$FAILURES" -eq 0 ]; then
    printf 'DEAD-GENERATION SMOKE OK (0 failures)\n'
    exit 0
fi
printf 'DEAD-GENERATION SMOKE FAILED (%d failures)\n' "$FAILURES"
exit 1
