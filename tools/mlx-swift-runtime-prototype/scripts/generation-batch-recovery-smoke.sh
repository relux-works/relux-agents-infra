#!/usr/bin/env bash
# Generation-batch failure recovery, end to end (TASK-260827-2q77g8).
#
# Carries the *recoverable* half of the mlx-lm generation regression into the
# MLX Swift runtime's acceptance suite. `scripts/dead-generation-smoke.sh`
# already pins the terminal half: a backend that is gone must stop answering
# `/health` 200. It says nothing about the far more common case -- a generation
# that blows up mid-batch on a runtime that is still perfectly able to serve the
# next caller.
#
# In Python that case was not survivable: the exception escaped the batch loop,
# killed the generation thread, and took the batch entry and its KV cache with
# it into a process that kept listening. Here the error propagates to its
# caller, so the runtime *can* recover -- and this script is what establishes
# that it does, observably, rather than by hoping ARC gets there first.
#
# Fifteen phases. Every resource claim is anchored to MLX's own allocator figures
# from `GET /debug/generation-state`, never to a counter the ledger minted --
# review defeated a counter-only version of this suite with two production
# mutants that left all 63 checks green:
#
#   1. recovery      -- a failure injected AFTER real tokens have reached the
#                       client ends that request with 500 (never a truncated
#                       200), leaves /health at 200, returns the batch slot,
#                       and the NEXT request succeeds on the same process with
#                       no restart
#   2. streaming     -- the same, through the SSE call site, which is a separate
#                       production path into the same state
#   3. multi-fault   -- --fault-inject-generation-error-count 2 fails exactly
#                       two requests and serves the third, so the seam's own
#                       arithmetic is checked rather than assumed
#   4. leak          -- ALLOCATOR-BOUND. mlx.active_bytes must not grow across
#                       failed generations. Catches a runtime that closes the
#                       ledger slot while retaining the failed ChatSession:
#                       +25,165,824 B of KV per failure, invisible to counters
#   5a. no-rebuild   -- NARROWING, and the control for 5b: an ordinary
#                       request-scoped failure releases the batch and leaves the
#                       shared pool alone
#   5b. rebuild      -- ALLOCATOR-BOUND. An exhausted allocation ALSO drops the
#                       shared pool, measured as cache_bytes well below 5a's
#                       control. Catches deletion of the production
#                       Memory.clearCache() while its counter and event survive
#   5c. oversize     -- NARROWING. A maxBufferLength rejection must NOT drop the
#                       pool: the allocator throws it before taking the cache
#                       lock, and clear_cache() cannot move that limit
#   6a. condemned    -- NARROWING, and the CLEAN-PATH cost of the zero-residue
#                       allowance. The recorded exhaustion still condemns:
#                       /health 503, later requests refused 503, marker emitted,
#                       batch still released. What changed in revision 6 is the
#                       verdict: this teardown is genuinely clean -- container
#                       gone, every owner gone, idle, at rest, the whole
#                       footprint handed back -- and it STILL abandons, because
#                       2,720 B of process-global residue is not attributable to
#                       anything and the allowance is zero. The phase asserts
#                       every other clause green, so the refusal is pinned to
#                       the residue, and asserts the cost: the pool is left
#                       holding the freed model and the marker demands
#                       replacement
#   6b. supervised   -- the marker still produces a replacement process
#   6c. teardown     -- NEGATIVE, ALLOCATOR-BOUND. --fault-inject-teardown-retain
#                       holds the condemned container so its weights are really
#                       never released. The runtime must then NOT clear the pool
#                       and NOT attest a rebuild: shared_cache_rebuilds stays 0,
#                       shared_cache_rebuild_pending stays true, and the residue
#                       is visible in mlx.active_bytes. Revision 2 discarded the
#                       timeout and attested a rebuild it had not performed
#   6e. long-context -- NEGATIVE, and review's own reproduction. The same inner
#                       retention as 6d, driven by a six-thousand-word prompt so
#                       the failed request's KV state outweighs the model. The
#                       process-global returned-byte comparison is then
#                       SATISFIED with every weight still resident -- revision 4
#                       attested a completed rebuild here, twice. The phase
#                       asserts that the bypass condition is present and that
#                       the runtime abandons anyway
#   6f. subset-hold  -- NEGATIVE, NARROWED, and the class no byte comparison can
#                       reach. --fault-inject-teardown-retain-weight-modules
#                       parks a STRICT SUBSET of the module tree, so the residue
#                       lands BELOW the model's footprint -- what a released
#                       model looks like to a process-global counter -- while
#                       this model's weights are still owned. Every byte-derived
#                       clause of the release gate reads green; only ownership
#                       refuses
#   6d. inner-hold   -- NEGATIVE, NARROWED. The interval 6c CANNOT reach.
#                       --fault-inject-teardown-retain-weights parks
#                       ModelContext.model, the weight-owning state BELOW the
#                       container, and lets the container itself be deallocated
#                       on schedule. So the wrapper's weak reference really does
#                       read nil while the whole model is still active -- the
#                       state revision 3 reported as a completed release,
#                       clearing the pool and attesting a rebuild with
#                       262,361,760 bytes resident. 6c cannot produce it,
#                       because parking the wrapper means weak-nil never happens
#                       at all
#
#   6g. array-hold   -- NEGATIVE, NARROWED, and the mirror of 6f. It holds no
#                       object of the model tree at all: the flattened parameter
#                       ARRAYS are copied out, so every Module dies, ownership
#                       reports the model released, and MLX still calls the
#                       whole footprint active. Ownership cannot refuse this;
#                       only the ABSOLUTE RESIDUE clause can, which is what
#                       makes it that clause's production negative

#   6h. subset-arrays -- NEGATIVE, NARROWED, and review's revision-5 bypass kept
#                       as a maintained production input. The largest HALF of
#                       the parameter arrays by nbytes, copied out: every Module
#                       dies, ownership reports the model released, the
#                       process-global delta clears the footprint, and the
#                       residue lands SIGNIFICANT but strictly BELOW the
#                       footprint -- the interval revision 5 attested a
#                       completed release over, at 255,724,192 B of 262,361,760 B.
#                       6g cannot reach this interval (its residue is at or
#                       above the footprint) and 6f cannot either (ownership
#                       refuses first)
#
# 5a is what makes 5b mean anything: a runtime that cleared the shared pool on
# every error would pass 5b and pay a cold-pool cost on every later generation.
# 5c is what keeps the pressure class honest in the other direction.
#
# 6 is what makes the whole script mean anything. Everything in 1-5 is
# satisfied by a runtime that recovered from *every* failure -- including the
# one that means the backend is gone. That runtime would hand the next caller a
# dead worker while reporting 200, which is precisely the incident 2h39ya
# exists to end. (6) requires the recovery path to have left that contract
# intact.
#
#   BINARY=./DerivedData/Build/Products/Release/mlx-swift-runtime-prototype \
#   HARNESS=/Users/alexis/.local/bin/model-harness \
#   MODEL=/Users/alexis/.cache/huggingface/hub/models--mlx-community--Qwen1.5-0.5B-Chat-4bit/snapshots/659d8dafc39202a6688bb46242d60440702489b1 \
#   PORT=18021 OUT=./batch-recovery-out \
#   scripts/generation-batch-recovery-smoke.sh
#
# Unlike the dead-generation smoke, this one DOES reach the weights: the fault
# fires after real tokens, which is the only way the batch has anything to
# release. The 261 MB model is still enough, and still leaves the host able to
# hold whatever else is resident.

set -uo pipefail
# Job control, so each managed run becomes its own process group. `model-harness`
# does not forward signals to the runtime it spawns, so signalling the harness
# alone orphans a live listener -- and an orphan holding this port is exactly how
# a later phase could pass by talking to the wrong process.
set -m

BINARY="${BINARY:?set BINARY to the xcodebuild Release product}"
HARNESS="${HARNESS:?set HARNESS to the model-harness executable}"
MODEL="${MODEL:?set MODEL to an absolute local model directory}"
PORT="${PORT:-18021}"
HOST=127.0.0.1
OUT="${OUT:-./batch-recovery-out}"
BASE="http://${HOST}:${PORT}"

# A failure that belongs to the request. Shares nothing with any signature.
BENIGN='RuntimeError: mid-batch generation step failed'
# The allocator giving up AFTER its own partial reclaim
# (`release_cached_buffers(mem_required - gc_limit_)`) already failed to make
# room. `clear_cache()` empties `buffer_cache_` outright rather than in a slice,
# so it can hand back what that reclaim kept -- the one class where a rebuild
# can change the next attempt. Does NOT condemn: one allocation failed and the
# backend can still serve.
ALLOC_FAILED='RuntimeError: [malloc] Unable to allocate 268435456 bytes.'
# Reads like allocation pressure and is not. `MetalAllocator::malloc` throws
# this before it takes the cache lock, on `size > device_->maxBufferLength()`;
# `clear_cache()` cannot move `maxBufferLength()`. Carried here as a NEGATIVE:
# revision 1 shipped it as a pressure class and charged every later generation a
# cold pool for a failure the pool can neither cause nor repair.
OVERSIZE='RuntimeError: [metal::malloc] Attempting to allocate 4294967296 bytes which is greater than the maximum allowed buffer size'
# Verbatim from the BUG-260827-1jhv2g incident record. Condemns.
INCIDENT='RuntimeError: [metal::malloc] Resource limit (499000) exceeded'
MARKER='generation_worker_unavailable'
# Tokens that must reach the client before the fault fires. Non-zero on purpose:
# a fault thrown before the ChatSession exists releases nothing, so a runtime
# that leaked every KV cache it ever built would pass a check written that way.
#
# One, not more. By the first chunk the whole prompt is already in the KV cache,
# the session exists and output has reached the client -- everything the
# recovery has to give back. A higher threshold buys nothing and costs
# robustness: this 0.5B chat model answers some prompts in a single token, and a
# threshold it never reaches turns a phase into a run where the fault simply did
# not happen. That fails loudly rather than silently -- each phase below demands
# its 500 -- but a check whose outcome depends on how chatty a model feels is
# not a check.
AFTER_TOKENS=1
MAX_TOKENS=24
# A retained KV cache for this model and prompt measured 25,165,824 B (24 MiB)
# per failure in review. The tolerance sits well under one increment, so a
# single leaked session is caught, while ordinary allocator jitter between
# identical requests is not.
LEAK_TOLERANCE=8388608
# The cache-drop margin for the rebuild A/B. Review measured 34,729,220 B with
# the clear against 67,955,820 B without it; anything at or below this margin
# would not distinguish them.
CACHE_DROP_MARGIN=16777216
# What MLX must report as still *active* once a correct teardown has run. A
# released 0.5B 4-bit model leaves single-digit KB active -- review measured
# 2,720 B. The ceiling is generous against that and still two orders of
# magnitude below the ~261 MB a retained model leaves behind, so it separates
# "released" from "substantially still held" rather than only from "entirely
# still held".
RELEASED_ACTIVE_CEILING=16777216
# ...and the floor the retention negative must clear, so that phase measures a
# runtime that really is still holding the weights. Sits above the ceiling
# above by a wide margin: no reading can satisfy both.
RETAINED_ACTIVE_FLOOR=134217728
# Long enough for the whole bounded teardown wait to expire:
# GenerationBatchRecovery.workerTeardownAttempts (3) x 100 polls x 20 ms, plus
# slack. Under-waiting here would read an abandonment that has not happened yet
# as one that never will.
TEARDOWN_BUDGET_SECONDS=12
# Fixed, and forwarded to the sampler. Without it each phase generates a
# different answer of a different length, so what the runtime is asked to
# recover from varies run to run, and a phase that passed yesterday can fail
# today for a reason that has nothing to do with the runtime.
SEED=20260827

case "$BINARY" in /*) ;; *) BINARY="$PWD/${BINARY#./}" ;; esac
case "$OUT" in /*) ;; *) OUT="$PWD/${OUT#./}" ;; esac

mkdir -p "$OUT"
CONFIG="$OUT/model-harness-batch-recovery.toml"

FAILURES=0
pass() { printf 'PASS  %s\n' "$*"; }
fail() { printf 'FAIL  %s\n' "$*"; FAILURES=$((FAILURES + 1)); }
info() { printf 'INFO  %s\n' "$*"; }

status_of() { curl -sS --max-time 15 -o "$2" -w '%{http_code}' "$1" 2>/dev/null; }

chat() {
    curl -sS --max-time 120 -o "$1" -w '%{http_code}' \
        -H 'Content-Type: application/json' -X POST "$BASE/v1/chat/completions" \
        --data-binary "{\"model\":\"$MODEL\",\"messages\":[{\"role\":\"user\",\"content\":\"Count from one to twenty.\"}],\"max_tokens\":$MAX_TOKENS,\"seed\":$SEED}" \
        2>/dev/null
}

stream_chat() {
    curl -sS --max-time 120 -o "$1" \
        -H 'Content-Type: application/json' -X POST "$BASE/v1/chat/completions" \
        --data-binary "{\"model\":\"$MODEL\",\"messages\":[{\"role\":\"user\",\"content\":\"Count from one to twenty.\"}],\"max_tokens\":$MAX_TOKENS,\"stream\":true,\"seed\":$SEED}" \
        >/dev/null 2>&1
}

state_of() { status_of "$BASE/debug/generation-state" "$1"; }

# The long-context request, from review's own reproduction of the revision-4
# bypass. About six thousand ordinary words makes the failed request's KV state
# LARGER than the model, which is the whole point: the process-global
# `returned_bytes` comparison is then satisfied by releasing the request alone,
# with every weight still resident. Phases 6e and 6f drive it so the suite
# measures the bypass condition rather than describing it.
#
# Built once, to a file, so both phases post byte-identical bodies and a
# difference between them can only come from the seam.
LONG_BODY="$OUT/long-context-request.json"
write_long_body() {
    python3 -c '
import json, sys
model, seed, path = sys.argv[1:]
with open(path, "w") as handle:
    json.dump(
        {
            "model": model,
            "messages": [{"role": "user", "content": "hello " * 6000}],
            "max_tokens": 2,
            "seed": int(seed),
        },
        handle,
    )
' "$MODEL" "$SEED" "$LONG_BODY"
}

chat_long() {
    curl -sS --max-time 240 -o "$1" -w '%{http_code}' \
        -H 'Content-Type: application/json' -X POST "$BASE/v1/chat/completions" \
        --data-binary "@$LONG_BODY" \
        2>/dev/null
}

# Assert one integer field of the batch ledger. Reads the endpoint's own JSON --
# nothing here recomputes what the runtime should have counted.
expect_batch() {
    local file=$1 field=$2 want=$3 label=$4 got
    got=$(python3 -c '
import json, sys
print(json.load(open(sys.argv[1]))["batch"][sys.argv[2]])
' "$file" "$field" 2>/dev/null)
    if [ "$got" = "$want" ]; then
        pass "$label ($field=$got)"
    else
        fail "$label ($field=${got:-unreadable}, want $want)"
    fi
}

# Assert one boolean field of the batch ledger. Separate from `expect_batch`
# because Python renders JSON booleans as True/False and a string compare
# against "true" would pass for every value that is not exactly that -- which is
# how a gate reporting the wrong answer reads as a gate reporting nothing.
expect_batch_bool() {
    local file=$1 field=$2 want=$3 label=$4 got
    got=$(python3 -c '
import json, sys
value = json.load(open(sys.argv[1]))["batch"][sys.argv[2]]
assert isinstance(value, bool), value
print("true" if value else "false")
' "$file" "$field" 2>/dev/null)
    if [ "$got" = "$want" ]; then
        pass "$label ($field=$got)"
    else
        fail "$label ($field=${got:-unreadable}, want $want)"
    fi
}

# Assert an allocator figure is at or above a floor -- that residue really is
# still resident. The mirror of `expect_no_growth`, and the half a negative
# phase needs: without it, a seam that quietly failed to retain anything would
# leave the phase asserting "no rebuild attested" about a runtime that had
# nothing to rebuild.
expect_at_least() {
    local got=$1 floor=$2 label=$3
    case "$got" in *[!0-9]*) fail "$label (unreadable allocator figures)"; return;; esac
    if [ "$got" -ge "$floor" ]; then
        pass "$label (${got}B >= floor ${floor}B)"
    else
        fail "$label (${got}B is below the ${floor}B floor; the seam retained nothing)"
    fi
}

# Read one MLX allocator figure from the endpoint. These are the numbers the
# ledger CANNOT mint: they come from `Memory.snapshot()`, i.e. from MLX's own
# `get_active_memory()` / `get_cache_memory()`. Review defeated a counter-only
# suite with two production mutants that left every counter and event intact, so
# every resource claim below is anchored here instead.
mlx_of() {
    local file=$1 field=$2
    python3 -c '
import json, sys
mlx = json.load(open(sys.argv[1]))["mlx"]
print("absent" if mlx is None else mlx[sys.argv[2]])
' "$file" "$field" 2>/dev/null || echo unreadable
}

# Read one field from the LAST occurrence of a named runtime event. The event
# log is where the teardown records the reading its verdict came from --
# container_deallocated, weight_footprint_bytes, baseline_active_bytes,
# returned_bytes -- so a phase can check the verdict against the measurement
# instead of taking the verdict's word for it.
event_field() {
    local log=$1 event=$2 field=$3
    python3 -c '
import json, sys
found = None
for line in open(sys.argv[1]):
    try:
        row = json.loads(line)
    except ValueError:
        continue
    if row.get("event") == sys.argv[2] and sys.argv[3] in row:
        found = row[sys.argv[3]]
print("absent" if found is None else ("true" if found is True else "false" if found is False else found))
' "$log" "$event" "$field" 2>/dev/null || echo unreadable
}

# Wait, bounded, for a condemned worker's deferred teardown to reach a terminal
# verdict.
#
# The teardown runs on a task the failing request scheduled and finishes AFTER
# the response is written, so a phase that reads `/debug/generation-state` the
# instant the request returns is racing it. Losing that race looks exactly like
# a runtime that refused to rebuild -- `shared_cache_rebuilds` 0 and
# `shared_cache_rebuild_pending` true -- which is a false negative, and worse, a
# false negative that would go away again the moment the machine got faster.
# This waits for the verdict to EXIST; every assertion about what the verdict
# says is still made afterwards, and a runtime that never settles fails here.
wait_teardown_settled() {
    local log=$1 limit=${2:-$TEARDOWN_BUDGET_SECONDS}
    for _ in $(seq 1 $((limit * 4))); do
        grep -q '"event":"generation_shared_cache_rebuilt"' "$log" && return 0
        grep -q '"event":"generation_shared_cache_rebuild_abandoned"' "$log" && return 0
        sleep 0.25
    done
    return 1
}

expect_equals() {
    local got=$1 want=$2 label=$3
    if [ "$got" = "$want" ]; then
        pass "$label ($got)"
    else
        fail "$label (got ${got:-unreadable}, want $want)"
    fi
}

# Assert an allocator figure is strictly below a ceiling AND above zero -- the
# shape of a residue that looks like a released model to a process-global
# counter while some of the model is still owned. Both halves matter: without
# the floor a seam that retained nothing would pass, and without the ceiling the
# phase would be indistinguishable from the whole-model retention above it.
expect_between() {
    local got=$1 ceiling=$2 label=$3
    case "$got$ceiling" in *[!0-9]*) fail "$label (unreadable allocator figures)"; return;; esac
    if [ "$got" -gt 0 ] && [ "$got" -lt "$ceiling" ]; then
        pass "$label (0 < ${got}B < ${ceiling}B)"
    else
        fail "$label (${got}B is not strictly between 0 and ${ceiling}B)"
    fi
}

# Assert an allocator figure has not grown beyond a tolerance.
expect_no_growth() {
    local got=$1 base=$2 tol=$3 label=$4 delta
    case "$got$base" in *[!0-9]*) fail "$label (unreadable allocator figures)"; return;; esac
    delta=$((got - base))
    if [ "$delta" -lt "$tol" ]; then
        pass "$label (delta=${delta}B, tolerance=${tol}B)"
    else
        fail "$label (delta=${delta}B exceeds tolerance ${tol}B; base=$base got=$got)"
    fi
}

# ------------------------------------------------------------------ config --
# Supervision is attached wherever a phase has to prove that no restart
# happened: a policy that is not configured cannot fire, so "no restart"
# measured without one measures nothing.
#
# It is deliberately OFF for the condemnation window in 6a. With
# `fatal_output_substrings` attached, model-harness kills the runtime within
# milliseconds of the marker reaching its stdout -- correct behaviour, and it
# destroys the very 503 the phase exists to observe. 6b attaches the policy and
# measures the other half. Recorded in LOGBOOK.md at 2205 and rediscovered here
# as six `000` connection-refused failures.
write_config() {
    local profile=$1 injection=$2 count=$3 after=$4 supervised=${5:-supervised} retain=${6:-}
    local retain_weights=${7:-}
    local retain_weight_modules=${8:-}
    local retain_weight_arrays=${9:-}
    local retain_weight_array_subset=${10:-}
    local injection_argv=""
    if [ -n "$injection" ]; then
        injection_argv=$(printf '    "--fault-inject-generation-error", "%s",\n' "$injection")
        if [ -n "$count" ]; then
            injection_argv+=$(printf '    "--fault-inject-generation-error-count", "%s",\n' "$count")
        fi
        injection_argv+=$(printf '    "--fault-inject-generation-error-after-tokens", "%s",\n' "$after")
        if [ -n "$retain" ]; then
            injection_argv+=$(printf '    "--fault-inject-teardown-retain", "%s",\n' "$retain")
        fi
        if [ -n "$retain_weights" ]; then
            injection_argv+=$(printf '    "--fault-inject-teardown-retain-weights", "%s",\n' "$retain_weights")
        fi
        if [ -n "$retain_weight_modules" ]; then
            injection_argv+=$(printf '    "--fault-inject-teardown-retain-weight-modules", "%s",\n' "$retain_weight_modules")
        fi
        if [ -n "$retain_weight_arrays" ]; then
            injection_argv+=$(printf '    "--fault-inject-teardown-retain-weight-arrays", "%s",\n' "$retain_weight_arrays")
        fi
        if [ -n "$retain_weight_array_subset" ]; then
            injection_argv+=$(printf '    "--fault-inject-teardown-retain-weight-array-subset", "%s",\n' "$retain_weight_array_subset")
        fi
    fi
    cat > "$CONFIG" <<TOML
# Task-scoped acceptance config for TASK-260827-2q77g8. NOT installed.
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
    local log=$1 limit=${2:-240} code
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

# ===================================== 1. the acceptance criterion itself ====
info "recovery: a mid-batch failure, then the next request on the same process"
write_config batch-recovery "$BENIGN" 1 "$AFTER_TOKENS"
start_harness batch-recovery "$OUT/recovery.log"
if wait_ready "$OUT/recovery.log"; then
    pass "recovery run reached ready"

    CODE=$(status_of "$BASE/health" "$OUT/recovery-health-before.json")
    [ "$CODE" = "200" ] && pass "recovery run /health is 200 before the fault" \
        || fail "recovery run /health answered $CODE before the fault (want 200)"

    state_of "$OUT/recovery-state-before.json" > /dev/null
    expect_batch "$OUT/recovery-state-before.json" active 0 \
        "nothing is in flight before any traffic"

    # ---- the affected request terminates with an explicit error -------------
    CODE=$(chat "$OUT/recovery-chat-1.json")
    if [ "$CODE" = "500" ]; then
        pass "RECOVERY GATE: the mid-batch failure answers the request with 500"
    else
        fail "RECOVERY GATE: the mid-batch failure answered $CODE (want 500)"
    fi
    # THE TRUNCATION NEGATIVE. By this point the client has already received
    # real tokens. A runtime that returned what it had accumulated would answer
    # 200 with a short but well-formed completion, and every other check in this
    # phase would still pass -- the caller would simply believe a truncated
    # answer was the whole answer.
    python3 -c '
import json, sys
body = json.load(open(sys.argv[1]))
assert "choices" not in body, "partial generation was returned as a completion"
assert body["error"]["code"] == "generation_failed", body
assert sys.argv[2] in body["error"]["message"], body
' "$OUT/recovery-chat-1.json" "$BENIGN" 2>/dev/null \
        && pass "the 500 names the failure and returns no partial completion" \
        || fail "the failed request did not report an explicit error (or returned a partial completion)"

    # ---- the worker is kept ------------------------------------------------
    CODE=$(status_of "$BASE/health" "$OUT/recovery-health-after.json")
    if [ "$CODE" = "200" ]; then
        pass "RECOVERY GATE: /health stays 200 after a request-scoped batch failure"
    else
        fail "RECOVERY GATE: /health answered $CODE after a request-scoped batch failure (want 200)"
    fi
    CODE=$(status_of "$BASE/v1/models" "$OUT/recovery-models.json")
    [ "$CODE" = "200" ] && pass "the model is still advertised on /v1/models" \
        || fail "/v1/models answered $CODE after a batch failure (want 200)"

    # ---- the invalid batch state is released -------------------------------
    state_of "$OUT/recovery-state-after.json" > /dev/null
    expect_batch "$OUT/recovery-state-after.json" active 0 \
        "RECOVERY GATE: no batch slot is left in flight after the failure"
    expect_batch "$OUT/recovery-state-after.json" failed 1 \
        "the failure is recorded"
    expect_batch "$OUT/recovery-state-after.json" batches_released 1 \
        "RECOVERY GATE: the failed batch was released"
    grep -q '"event":"generation_batch_released"' "$OUT/recovery.log" \
        && pass "the runtime emitted a generation_batch_released event" \
        || fail "no generation_batch_released event in the runtime output"

    # ---- and the next request succeeds -------------------------------------
    CODE=$(chat "$OUT/recovery-chat-2.json")
    if [ "$CODE" = "200" ]; then
        pass "RECOVERY GATE: the next request completes on the same process"
    else
        fail "RECOVERY GATE: the next request answered $CODE (want 200)"
    fi
    # Content or reasoning: the 0.5B chat model this runs on emits no
    # `</think>`, so the splitter files its whole answer under `reasoning`.
    # Requiring `content` specifically would fail a perfectly recovered
    # generation for a property of the model, not of the recovery.
    python3 -c '
import json, sys
body = json.load(open(sys.argv[1]))
message = body["choices"][0]["message"]
assert (message.get("content") or message.get("reasoning")), "recovered completion is empty"
assert body["usage"]["completion_tokens"] > 0, body["usage"]
assert body["choices"][0]["finish_reason"], body["choices"][0]
' "$OUT/recovery-chat-2.json" 2>/dev/null \
        && pass "the recovered completion carries real text and token counts" \
        || fail "the recovered completion is empty or has no usage"

    state_of "$OUT/recovery-state-final.json" > /dev/null
    expect_batch "$OUT/recovery-state-final.json" completed 1 "the second generation completed"
    expect_batch "$OUT/recovery-state-final.json" active 0 "nothing is left in flight"

    # ---- without restarting a healthy process ------------------------------
    grep -q "$MARKER" "$OUT/recovery.log" \
        && fail "a request-scoped batch failure emitted the supervision marker" \
        || pass "a request-scoped batch failure emitted no supervision marker"
    grep -q 'restarting profile' "$OUT/recovery.log" \
        && fail "RECOVERY GATE: model-harness restarted a runtime that recovered on its own" \
        || pass "RECOVERY GATE: no restart -- the recovery happened in the original process"
    if [ "$(grep -c '"event":"listening"' "$OUT/recovery.log")" -eq 1 ]; then
        pass "exactly one listener was ever bound (no replacement process)"
    else
        fail "more than one listening event: the recovery was a restart in disguise"
    fi
else
    fail "recovery run never became ready"
fi
stop_harness

# ======================== 2. the streaming call site recovers as well =======
# `Router.complete` and `RuntimeHTTPHandler.sendStream` are separate production
# paths into the same engine. A runtime that recovered only the buffered one
# would leak a batch per stream on exactly the path Pi uses.
info "streaming: the same mid-batch failure, driven through SSE"
start_harness batch-recovery "$OUT/stream.log"
if wait_ready "$OUT/stream.log"; then
    pass "streaming run reached ready"

    # SSE opens with a 200 head before generation starts, so the failure can
    # only be reported as a terminal frame; the status code says nothing here.
    stream_chat "$OUT/stream-body-1.txt"
    grep -q 'generation_failed' "$OUT/stream-body-1.txt" \
        && pass "the stream reports the batch failure as a terminal frame" \
        || fail "the stream did not report the batch failure"
    # The partial chunks are real, and they are what the truncation negative in
    # phase 1 is about: the stream delivered content and then failed, so a
    # client that treated the frames as a complete answer would be wrong. The
    # error frame is what tells it otherwise.
    grep -q '"delta"' "$OUT/stream-body-1.txt" \
        && pass "partial chunks reached the client before the failure" \
        || fail "no chunks reached the client; the fault fired before generation"
    python3 -c '
import json, sys
frames = [json.loads(line[6:]) for line in open(sys.argv[1])
          if line.startswith("data: ") and not line.startswith("data: [DONE]")]
finished = [f for f in frames
            if f.get("choices") and f["choices"][0].get("finish_reason")]
assert not finished, f"failed stream still sent a finish_reason: {finished}"
' "$OUT/stream-body-1.txt" 2>/dev/null \
        && pass "the failed stream sent no finish_reason frame" \
        || fail "the failed stream sent a finish_reason, reading as a clean completion"

    CODE=$(status_of "$BASE/health" "$OUT/stream-health.json")
    [ "$CODE" = "200" ] && pass "/health stays 200 after a failed stream" \
        || fail "/health answered $CODE after a failed stream (want 200)"
    state_of "$OUT/stream-state.json" > /dev/null
    expect_batch "$OUT/stream-state.json" active 0 \
        "RECOVERY GATE (streaming): no batch slot is left in flight"
    expect_batch "$OUT/stream-state.json" batches_released 1 \
        "RECOVERY GATE (streaming): the failed batch was released"

    stream_chat "$OUT/stream-body-2.txt"
    python3 -c '
import json, sys
frames = [json.loads(line[6:]) for line in open(sys.argv[1])
          if line.startswith("data: ") and not line.startswith("data: [DONE]")]
assert frames, "no frames at all"
assert not any("error" in f for f in frames), "recovered stream carried an error frame"
assert any(f.get("choices") and f["choices"][0].get("finish_reason") for f in frames), \
    "recovered stream never finished"
' "$OUT/stream-body-2.txt" 2>/dev/null \
        && pass "RECOVERY GATE (streaming): the next stream completes cleanly" \
        || fail "RECOVERY GATE (streaming): the next stream did not complete cleanly"
    grep -q 'restarting profile' "$OUT/stream.log" \
        && fail "(streaming) model-harness restarted a runtime that recovered" \
        || pass "(streaming) no restart"
else
    fail "streaming run never became ready"
fi
stop_harness

# ================================ 3. the seam's own arithmetic is checked ====
# Everything above rests on --fault-inject-generation-error-count actually
# bounding the injection. A seam that always failed exactly one request would
# pass phases 1 and 2 while making the count flag a lie, and any later phase
# written against a different count would be measuring nothing.
info "multi-fault: count=2 must fail exactly two requests and serve the third"
write_config batch-multi "$BENIGN" 2 "$AFTER_TOKENS"
start_harness batch-multi "$OUT/multi.log"
if wait_ready "$OUT/multi.log"; then
    pass "multi-fault run reached ready"
    CODE1=$(chat "$OUT/multi-chat-1.json")
    CODE2=$(chat "$OUT/multi-chat-2.json")
    CODE3=$(chat "$OUT/multi-chat-3.json")
    [ "$CODE1" = "500" ] && pass "request 1 of 3 failed as configured" \
        || fail "request 1 answered $CODE1 (want 500)"
    [ "$CODE2" = "500" ] && pass "request 2 of 3 failed as configured" \
        || fail "request 2 answered $CODE2 (want 500)"
    [ "$CODE3" = "200" ] && pass "SEAM GATE: request 3 succeeded -- the count bounds the seam" \
        || fail "SEAM GATE: request 3 answered $CODE3 (want 200); the count does not bound the seam"
    state_of "$OUT/multi-state.json" > /dev/null
    expect_batch "$OUT/multi-state.json" failed 2 "exactly two generations failed"
    expect_batch "$OUT/multi-state.json" completed 1 "exactly one generation completed"
    expect_batch "$OUT/multi-state.json" batches_released 2 "both failed batches were released"
    expect_batch "$OUT/multi-state.json" active 0 "nothing is left in flight"
else
    fail "multi-fault run never became ready"
fi
stop_harness

# ================ 4. the KV state is actually gone, per the allocator =======
# THE LEAK PHASE. Every counter in this suite is minted by the ledger, so a
# runtime that closed its slot while retaining the failed `ChatSession` reports
# a spotless recovery: `active=0`, `batches_released=N`, next request fine. That
# mutant survived all 63 checks of revision 1 while MLX carried one KV-sized
# increment per failure (+25,165,824 B each). Only `mlx.active_bytes` sees it.
info "leak (allocator-bound): failed generations must not retain their KV state"
write_config batch-leak "$BENIGN" 2 "$AFTER_TOKENS"
start_harness batch-leak "$OUT/leak.log"
if wait_ready "$OUT/leak.log"; then
    pass "leak run reached ready"
    state_of "$OUT/leak-state-0.json" > /dev/null
    BASE_ACTIVE=$(mlx_of "$OUT/leak-state-0.json" active_bytes)
    if [ "$BASE_ACTIVE" = "absent" ] || [ "$BASE_ACTIVE" = "unreadable" ]; then
        fail "no allocator baseline: mlx figures are $BASE_ACTIVE with a model resident"
    else
        pass "allocator baseline read with the model resident (active_bytes=$BASE_ACTIVE)"
    fi
    info "baseline active_bytes=$BASE_ACTIVE"

    CODE=$(chat "$OUT/leak-chat-1.json")
    [ "$CODE" = "500" ] && pass "first generation failed as configured" \
        || fail "first generation answered $CODE (want 500)"
    state_of "$OUT/leak-state-1.json" > /dev/null
    A1=$(mlx_of "$OUT/leak-state-1.json" active_bytes)
    info "after 1 failure active_bytes=$A1"
    expect_no_growth "$A1" "$BASE_ACTIVE" "$LEAK_TOLERANCE" \
        "LEAK GATE: one failed generation retained no KV state"

    CODE=$(chat "$OUT/leak-chat-2.json")
    [ "$CODE" = "500" ] && pass "second generation failed as configured" \
        || fail "second generation answered $CODE (want 500)"
    state_of "$OUT/leak-state-2.json" > /dev/null
    A2=$(mlx_of "$OUT/leak-state-2.json" active_bytes)
    info "after 2 failures active_bytes=$A2"
    # Two failures, not one, because a leak is cumulative and a single sample
    # can always be argued to be jitter. A retained session shows up here as a
    # second increment of the same size.
    expect_no_growth "$A2" "$BASE_ACTIVE" "$LEAK_TOLERANCE" \
        "LEAK GATE: two failed generations retained no KV state"
    expect_batch "$OUT/leak-state-2.json" batches_released 2 \
        "the ledger agrees both batches were released"

    # And a success afterwards leaves the same footprint, so the release is not
    # an artefact of the failure path alone.
    CODE=$(chat "$OUT/leak-chat-3.json")
    [ "$CODE" = "200" ] && pass "the third request completes on the same process" \
        || fail "the third request answered $CODE (want 200)"
    state_of "$OUT/leak-state-3.json" > /dev/null
    A3=$(mlx_of "$OUT/leak-state-3.json" active_bytes)
    info "after a success active_bytes=$A3"
    expect_no_growth "$A3" "$BASE_ACTIVE" "$LEAK_TOLERANCE" \
        "LEAK GATE: a completed generation retained no KV state either"
else
    fail "leak run never became ready"
fi
stop_harness

# ====== 5a. NARROWING: the pool is NOT dropped for an unrelated failure =====
# Runs BEFORE the rebuild phase because it is the control the rebuild phase is
# measured against: same model, same prompt, same pinned seed, same single
# failed request. The only difference is the injected message, so the two
# `cache_bytes` readings differ only by whether the pool was cleared.
info "no-rebuild (narrowing): an ordinary failure releases the batch and nothing else"
write_config batch-no-rebuild "$BENIGN" 1 "$AFTER_TOKENS"
start_harness batch-no-rebuild "$OUT/no-rebuild.log"
NO_REBUILD_CACHE=""
if wait_ready "$OUT/no-rebuild.log"; then
    pass "no-rebuild run reached ready"
    CODE=$(chat "$OUT/no-rebuild-chat.json")
    [ "$CODE" = "500" ] && pass "the request-scoped failure answers the request with 500" \
        || fail "the request-scoped failure answered $CODE (want 500)"
    state_of "$OUT/no-rebuild-state.json" > /dev/null
    expect_batch "$OUT/no-rebuild-state.json" batches_released 1 \
        "the batch is still released"
    expect_batch "$OUT/no-rebuild-state.json" shared_cache_rebuilds 0 \
        "NARROWING GATE: the shared buffer pool was left alone"
    grep -q '"rebuilt_shared_cache":true' "$OUT/no-rebuild.log" \
        && fail "NARROWING GATE: the runtime dropped the shared pool for an unrelated failure" \
        || pass "NARROWING GATE: no rebuild was recorded for an unrelated failure"
    NO_REBUILD_CACHE=$(mlx_of "$OUT/no-rebuild-state.json" cache_bytes)
    info "control cache_bytes with the pool left alone=$NO_REBUILD_CACHE"
else
    fail "no-rebuild run never became ready"
fi
stop_harness

# ============ 5b. cache state is rebuilt when the failure implicates it ======
info "rebuild (allocator-bound): an exhausted allocation drops the shared buffer pool"
write_config batch-rebuild "$ALLOC_FAILED" 1 "$AFTER_TOKENS"
start_harness batch-rebuild "$OUT/rebuild.log"
if wait_ready "$OUT/rebuild.log"; then
    pass "rebuild run reached ready"
    CODE=$(chat "$OUT/rebuild-chat-1.json")
    [ "$CODE" = "500" ] && pass "the allocation failure answers the request with 500" \
        || fail "the allocation failure answered $CODE (want 500)"

    state_of "$OUT/rebuild-state.json" > /dev/null
    expect_batch "$OUT/rebuild-state.json" batches_released 1 "the failed batch was released"
    expect_batch "$OUT/rebuild-state.json" shared_cache_rebuilds 1 \
        "the runtime reports the shared buffer pool was dropped"
    grep -q '"rebuilt_shared_cache":true' "$OUT/rebuild.log" \
        && pass "the released-batch event records the rebuild" \
        || fail "the released-batch event does not record a rebuild"

    # THE MUTANT THAT SURVIVED REVISION 1. Deleting the production
    # `Memory.clearCache()` leaves the counter above, the event above, and every
    # other check in this phase untouched. Only the allocator disagrees.
    REBUILD_CACHE=$(mlx_of "$OUT/rebuild-state.json" cache_bytes)
    info "cache_bytes after the rebuild=$REBUILD_CACHE (control=$NO_REBUILD_CACHE)"
    case "$REBUILD_CACHE$NO_REBUILD_CACHE" in
        *[!0-9]*)
            fail "REBUILD GATE: allocator cache figures unreadable (rebuild=$REBUILD_CACHE control=$NO_REBUILD_CACHE)"
            ;;
        *)
            DROP=$((NO_REBUILD_CACHE - REBUILD_CACHE))
            if [ "$DROP" -ge "$CACHE_DROP_MARGIN" ]; then
                pass "REBUILD GATE: MLX actually returned the pool (${DROP}B below the control)"
            else
                fail "REBUILD GATE: the pool was not returned (only ${DROP}B below the control, want >= ${CACHE_DROP_MARGIN}B); the counter says it was"
            fi
            ;;
    esac

    # NARROWING against the condemning gate: this message is request-scoped and
    # must not take a healthy runtime out of rotation.
    CODE=$(status_of "$BASE/health" "$OUT/rebuild-health.json")
    if [ "$CODE" = "200" ]; then
        pass "NARROWING GATE: an allocation failure does not condemn the worker"
    else
        fail "NARROWING GATE: an allocation failure moved /health to $CODE (want 200)"
    fi
    grep -q "$MARKER" "$OUT/rebuild.log" \
        && fail "an allocation failure emitted the supervision marker" \
        || pass "an allocation failure emitted no supervision marker"

    CODE=$(chat "$OUT/rebuild-chat-2.json")
    [ "$CODE" = "200" ] && pass "REBUILD GATE: the next request completes after the pool was dropped" \
        || fail "REBUILD GATE: the next request answered $CODE (want 200)"
else
    fail "rebuild run never became ready"
fi
stop_harness

# ===== 5c. NARROWING: an oversize rejection is not shared-cache pressure =====
# The revision-1 defect. `MetalAllocator::malloc` throws this before it takes
# the cache lock, comparing against `device_->maxBufferLength()`, and
# `clear_cache()` cannot move that limit. Classifying it as pressure charged
# every later generation a cold pool to recover from a failure the pool can
# neither cause nor repair -- and the suite could not tell, because the next
# request it made was small enough to succeed either way.
info "oversize (narrowing): a limit rejection must not drop the shared pool"
write_config batch-oversize "$OVERSIZE" 1 "$AFTER_TOKENS"
start_harness batch-oversize "$OUT/oversize.log"
if wait_ready "$OUT/oversize.log"; then
    pass "oversize run reached ready"
    CODE=$(chat "$OUT/oversize-chat.json")
    [ "$CODE" = "500" ] && pass "the oversize rejection answers the request with 500" \
        || fail "the oversize rejection answered $CODE (want 500)"
    state_of "$OUT/oversize-state.json" > /dev/null
    expect_batch "$OUT/oversize-state.json" batches_released 1 \
        "the batch is still released for an oversize rejection"
    expect_batch "$OUT/oversize-state.json" shared_cache_rebuilds 0 \
        "NARROWING GATE: an oversize rejection does not drop the shared pool"
    OVERSIZE_CACHE=$(mlx_of "$OUT/oversize-state.json" cache_bytes)
    info "cache_bytes after an oversize rejection=$OVERSIZE_CACHE (control=$NO_REBUILD_CACHE)"
    case "$OVERSIZE_CACHE$NO_REBUILD_CACHE" in
        *[!0-9]*) fail "NARROWING GATE: allocator cache figures unreadable" ;;
        *)
            DROP=$((NO_REBUILD_CACHE - OVERSIZE_CACHE))
            if [ "$DROP" -lt "$CACHE_DROP_MARGIN" ]; then
                pass "NARROWING GATE: MLX kept the pool for an oversize rejection (${DROP}B from the control)"
            else
                fail "NARROWING GATE: the pool was dropped for a failure it cannot repair (${DROP}B below the control)"
            fi
            ;;
    esac
    CODE=$(status_of "$BASE/health" "$OUT/oversize-health.json")
    [ "$CODE" = "200" ] && pass "an oversize rejection leaves the worker serving" \
        || fail "an oversize rejection moved /health to $CODE (want 200)"
else
    fail "oversize run never became ready"
fi
stop_harness

# ============ 6. NARROWING: an unrecoverable death is still unrecoverable ===
# THE LOAD-BEARING PHASE. Everything above is satisfied by a runtime that
# recovers from every failure, including the one that means the backend is
# gone. That runtime hands the next caller a dead worker while answering
# /health 200 -- the exact incident TASK-260827-2h39ya exists to end. This
# phase requires the recovery path to have left that contract intact, and the
# batch to be released even so.
#
# Split in two for the same reason `dead-generation-smoke.sh` is: the 503 window
# belongs to the runtime and the restart belongs to the supervisor, and with a
# policy attached the second destroys the first within milliseconds. Measuring
# them in one process only measures which won the race.
info "condemned 6a (narrowing, unsupervised): the recorded exhaustion still takes the runtime out of rotation"
write_config batch-condemned "$INCIDENT" 1 "$AFTER_TOKENS" unsupervised
start_harness batch-condemned "$OUT/condemned.log"
if wait_ready "$OUT/condemned.log"; then
    pass "condemned run reached ready"

    CODE=$(status_of "$BASE/health" "$OUT/condemned-health-before.json")
    [ "$CODE" = "200" ] && pass "condemned run /health is 200 before the fault" \
        || fail "condemned run /health answered $CODE before the fault (want 200)"

    CODE=$(chat "$OUT/condemned-chat-1.json")
    [ "$CODE" = "500" ] && pass "the exhaustion answers the provoking request with 500" \
        || fail "the exhaustion answered $CODE (want 500)"

    CODE=$(status_of "$BASE/health" "$OUT/condemned-health-after.json")
    if [ "$CODE" = "503" ]; then
        pass "NARROWING GATE: an unrecoverable failure still moves /health to 503"
    else
        fail "NARROWING GATE: /health answered $CODE for a condemned worker (want 503); recovery swallowed the condemnation"
    fi
    python3 -c '
import json, sys
body = json.load(open(sys.argv[1]))
assert body["status"] == "unavailable", body
assert "metal::malloc" in body["detail"], body
' "$OUT/condemned-health-after.json" 2>/dev/null \
        && pass "condemned /health body is still {\"status\":\"unavailable\"}" \
        || fail "condemned /health body is not {\"status\":\"unavailable\"}"

    # The seam has budget for exactly one injection, so a runtime that merely
    # rearmed its engine would serve this one. It must be refused instead: the
    # condemnation is terminal regardless of whether the fault would repeat.
    CODE=$(chat "$OUT/condemned-chat-2.json")
    if [ "$CODE" = "503" ]; then
        pass "NARROWING GATE: the next request is refused 503, not served by a condemned worker"
    else
        fail "NARROWING GATE: the next request answered $CODE (want 503); the condemned worker was reused"
    fi

    grep -q "$MARKER" "$OUT/condemned.log" \
        && pass "the supervision marker is still emitted for an unrecoverable failure" \
        || fail "no supervision marker for an unrecoverable failure"

    # ...and the dead worker still gave its batch back. Condemnation is not an
    # excuse to leak: the supervisor is about to start a replacement that needs
    # the host's memory.
    wait_teardown_settled "$OUT/condemned.log" \
        && pass "the deferred teardown reached a terminal verdict" \
        || fail "the deferred teardown never reached a terminal verdict within ${TEARDOWN_BUDGET_SECONDS}s"

    state_of "$OUT/condemned-state.json" > /dev/null
    expect_batch "$OUT/condemned-state.json" active 0 \
        "a condemned worker leaves no batch slot in flight"
    expect_batch "$OUT/condemned-state.json" batches_released 1 \
        "a condemned worker still releases its batch"
    expect_batch "$OUT/condemned-state.json" shared_cache_rebuilds 0 \
        "CLEAN-PATH GATE: a condemned worker attests no rebuild it cannot prove"
    expect_batch "$OUT/condemned-state.json" shared_cache_rebuilds_abandoned 1 \
        "CLEAN-PATH GATE: the deferred rebuild is counted as abandoned"
    expect_batch_bool "$OUT/condemned-state.json" shared_cache_rebuild_pending true \
        "CLEAN-PATH GATE: the pool stays reported as owed"

    # THE CLEAN-PATH GATE, and the change this revision is about.
    #
    # Until revision 5 this phase asserted a COMPLETED rebuild: the container was
    # gone, every owner was gone, the allocator was idle and at rest, the whole
    # footprint had been handed back, and the residue -- 2,720 B -- was below the
    # model's footprint, so the gate attested a release and cleared the pool.
    #
    # Review then walked a production input into that interval five rounds
    # running, most recently a strict subset of this model's own copied
    # parameter arrays sitting at 255,724,192 B of a 262,361,760 B footprint with
    # zero live owners. The interval is the defect, not its width: MLX's
    # counters are process-global, so no threshold above zero can say what is
    # underneath it. The allowance is now zero
    # (GenerationBatchRecovery.residualNonWeightAllowanceBytes), and 2,720 B is
    # not zero -- so the clean teardown abandons too.
    #
    # That is the point of this phase now. It measures the cost of the fix on
    # the one path that used to succeed, and it fails if the runtime ever goes
    # back to attesting over a residue it cannot attribute.
    CONDEMNED_CACHE=$(mlx_of "$OUT/condemned-state.json" cache_bytes)
    CONDEMNED_ACTIVE=$(mlx_of "$OUT/condemned-state.json" active_bytes)
    info "condemned cache_bytes=$CONDEMNED_CACHE active_bytes=$CONDEMNED_ACTIVE"

    grep -q '"event":"generation_shared_cache_rebuilt"' "$OUT/condemned.log" \
        && fail "CLEAN-PATH GATE: a rebuild was attested over an unattributable residue" \
        || pass "CLEAN-PATH GATE: no completed-rebuild event on the clean teardown either"
    grep -q '"event":"generation_shared_cache_rebuild_abandoned"' "$OUT/condemned.log" \
        && pass "CLEAN-PATH GATE: the clean teardown recorded an abandoned rebuild" \
        || fail "CLEAN-PATH GATE: the clean teardown recorded no terminal verdict"
    python3 -c '
import json, sys
for line in open(sys.argv[1]):
    try:
        row = json.loads(line)
    except ValueError:
        continue
    if row.get("event") == "generation_shared_cache_rebuild_abandoned":
        assert row["release_observed"] is False, row
        break
else:
    raise SystemExit("no abandoned event")
' "$OUT/condemned.log" 2>/dev/null \
        && pass "the abandonment records that no release was observed" \
        || fail "the abandonment does not record an unobserved release"

    # THE MEASUREMENT GATE. The refusal above has to be attributable to the
    # residue and to nothing else, or this phase would pass for a runtime that
    # abandoned because its registry was empty, its clock ran out, or its
    # container never died. Every other clause is asserted GREEN here, so the
    # only thing left refusing is the byte that is still resident.
    CLEAN_DEALLOC=$(event_field "$OUT/condemned.log" generation_shared_cache_rebuild_abandoned container_deallocated)
    CLEAN_FOOTPRINT=$(event_field "$OUT/condemned.log" generation_shared_cache_rebuild_abandoned weight_footprint_bytes)
    CLEAN_RETURNED=$(event_field "$OUT/condemned.log" generation_shared_cache_rebuild_abandoned returned_bytes)
    CLEAN_BASELINE=$(event_field "$OUT/condemned.log" generation_shared_cache_rebuild_abandoned baseline_active_bytes)
    CLEAN_OWNERS=$(event_field "$OUT/condemned.log" generation_shared_cache_rebuild_abandoned weight_owner_count)
    CLEAN_LIVE=$(event_field "$OUT/condemned.log" generation_shared_cache_rebuild_abandoned live_weight_owners)
    CLEAN_INFLIGHT=$(event_field "$OUT/condemned.log" generation_shared_cache_rebuild_abandoned generations_in_flight)
    CLEAN_STABLE=$(event_field "$OUT/condemned.log" generation_shared_cache_rebuild_abandoned stable_active_samples)
    CLEAN_ACTIVE_AT_VERDICT=$(event_field "$OUT/condemned.log" generation_shared_cache_rebuild_abandoned observed_active_bytes)
    CLEAN_ALLOWANCE=$(event_field "$OUT/condemned.log" generation_shared_cache_rebuild_abandoned residual_non_weight_allowance_bytes)
    info "clean teardown container_deallocated=$CLEAN_DEALLOC owners=$CLEAN_OWNERS live=$CLEAN_LIVE in_flight=$CLEAN_INFLIGHT stable=$CLEAN_STABLE footprint=$CLEAN_FOOTPRINT baseline=$CLEAN_BASELINE returned=$CLEAN_RETURNED observed_active=$CLEAN_ACTIVE_AT_VERDICT allowance=$CLEAN_ALLOWANCE"
    expect_equals "$CLEAN_DEALLOC" true \
        "MEASUREMENT GATE: the clean teardown saw the container deallocated"
    case "$CLEAN_FOOTPRINT" in
        *[!0-9]*|0) fail "MEASUREMENT GATE: the weight footprint was never measured (got ${CLEAN_FOOTPRINT:-unreadable})" ;;
        *) pass "MEASUREMENT GATE: the model's weight footprint was measured at load (${CLEAN_FOOTPRINT}B)" ;;
    esac
    case "$CLEAN_OWNERS" in
        *[!0-9]*|0) fail "MEASUREMENT GATE: the weight-owner registry was never populated (got ${CLEAN_OWNERS:-unreadable})" ;;
        *) pass "MEASUREMENT GATE: this model's module tree was registered at load (${CLEAN_OWNERS} owners)" ;;
    esac
    expect_equals "$CLEAN_LIVE" 0 \
        "MEASUREMENT GATE: every weight owner of this model was deallocated"
    expect_equals "$CLEAN_INFLIGHT" 0 \
        "MEASUREMENT GATE: nothing else was allocating when the reading was taken"
    case "$CLEAN_STABLE" in
        *[!0-9]*) fail "MEASUREMENT GATE: the stability run is unreadable" ;;
        *)
            if [ "$CLEAN_STABLE" -ge 3 ]; then
                pass "MEASUREMENT GATE: the allocator had come to rest before the verdict (${CLEAN_STABLE} identical samples)"
            else
                fail "MEASUREMENT GATE: the verdict was taken on a reading that was still moving (${CLEAN_STABLE} samples)"
            fi
            ;;
    esac
    case "$CLEAN_RETURNED$CLEAN_FOOTPRINT" in
        *[!0-9]*) fail "MEASUREMENT GATE: teardown figures unreadable" ;;
        *)
            if [ "$CLEAN_RETURNED" -ge "$CLEAN_FOOTPRINT" ]; then
                pass "MEASUREMENT GATE: MLX gave back at least the whole model (${CLEAN_RETURNED}B >= ${CLEAN_FOOTPRINT}B)"
            else
                fail "MEASUREMENT GATE: only ${CLEAN_RETURNED}B of ${CLEAN_FOOTPRINT}B came back; the refusal is then not attributable to the residue"
            fi
            ;;
    esac
    # ...and the model really is gone, by the absolute reading as well as by the
    # delta. Both halves matter: this is a genuinely CLEAN teardown, which is
    # what makes its abandonment a cost rather than a catch.
    expect_no_growth "$CLEAN_ACTIVE_AT_VERDICT" 0 "$RELEASED_ACTIVE_CEILING" \
        "MEASUREMENT GATE: the condemned worker's weights were actually released"
    expect_no_growth "$CONDEMNED_ACTIVE" 0 "$RELEASED_ACTIVE_CEILING" \
        "MEASUREMENT GATE: and they are still gone when the endpoint is asked"

    # THE ALLOWANCE GATE. The one number the verdict turns on, read from the
    # runtime's own event rather than assumed, plus the fact that this run's
    # residue is outside it. A revision that raised the allowance to admit
    # 2,720 B would redden both halves.
    expect_equals "$CLEAN_ALLOWANCE" 0 \
        "ALLOWANCE GATE: the runtime admits no residue at all"
    case "$CLEAN_ACTIVE_AT_VERDICT$CLEAN_ALLOWANCE" in
        *[!0-9]*) fail "ALLOWANCE GATE: residue figures unreadable" ;;
        *)
            if [ "$CLEAN_ACTIVE_AT_VERDICT" -gt "$CLEAN_ALLOWANCE" ]; then
                pass "ALLOWANCE GATE: the clean teardown's residue is outside the allowance (${CLEAN_ACTIVE_AT_VERDICT}B > ${CLEAN_ALLOWANCE}B), which is why it abandoned"
            else
                fail "ALLOWANCE GATE: the residue was inside the allowance (${CLEAN_ACTIVE_AT_VERDICT}B <= ${CLEAN_ALLOWANCE}B); this phase then proves nothing about the refusal"
            fi
            ;;
    esac

    # THE COST, asserted rather than described. Refusing to attest means
    # refusing to clear, so the pool this runtime never rebuilt is still holding
    # the freed model. That is the price of the allowance above, and it is only
    # payable because the abandonment re-announces the supervision marker: the
    # supervisor replaces the process and the host gets everything back. A
    # revision that quietly cleared here without attesting would drop this
    # figure and redden the check.
    case "$CONDEMNED_CACHE$CLEAN_FOOTPRINT" in
        *[!0-9]*) fail "COST GATE: condemned allocator figures unreadable" ;;
        *)
            if [ "$CONDEMNED_CACHE" -ge $((CLEAN_FOOTPRINT / 2)) ]; then
                pass "COST GATE: the unattested pool is still held (${CONDEMNED_CACHE}B of a ${CLEAN_FOOTPRINT}B model), and the marker demands replacement"
            else
                fail "COST GATE: the pool was cleared (${CONDEMNED_CACHE}B) on a teardown that attested nothing; a clear and an attestation must not come apart"
            fi
            ;;
    esac

    python3 -c '
import json, sys
body = json.load(open(sys.argv[1]))
assert body["readiness"] == "generation_worker_failed", body
' "$OUT/condemned-state.json" 2>/dev/null \
        && pass "the state endpoint still answers after condemnation and names it" \
        || fail "the state endpoint did not report the condemned readiness"

    grep -q "$MARKER.*did not release" "$OUT/condemned.log" \
        && pass "CLEAN-PATH GATE: the abandonment re-announces the process as unusable" \
        || fail "CLEAN-PATH GATE: an abandoned rebuild left no supervision marker naming it"

else
    fail "condemned run never became ready"
fi
stop_harness

# ================= 6b. and the supervisor still gets the last word ==========
# The other half of 6a, with the policy attached. Fresh process: the
# condemnation above is terminal, so the restart has to be provoked again.
info "condemned 6b (supervised): the marker still produces a replacement process"
write_config batch-condemned-supervised "$INCIDENT" 1 "$AFTER_TOKENS" supervised
start_harness batch-condemned-supervised "$OUT/condemned-supervised.log"
if wait_ready "$OUT/condemned-supervised.log"; then
    pass "supervised condemned run reached ready"
    # The response is not asserted: the marker reaches the harness while the
    # reply is still being written, so the kill may land first. That the client
    # loses this request is the point of restarting.
    chat "$OUT/condemned-supervised-chat.json" > "$OUT/condemned-supervised-chat.code"
    info "provoking request returned HTTP $(cat "$OUT/condemned-supervised-chat.code")"
    RESTARTED=0
    for _ in $(seq 1 240); do
        if grep -q 'restarting profile "batch-condemned-supervised"' \
            "$OUT/condemned-supervised.log"; then
            RESTARTED=1
            break
        fi
        sleep 0.25
    done
    [ "$RESTARTED" -eq 1 ] \
        && pass "NARROWING GATE: model-harness still restarts a condemned worker" \
        || fail "NARROWING GATE: model-harness never restarted the condemned runtime"
    grep -q 'fatal output "'"$MARKER"'"' "$OUT/condemned-supervised.log" \
        && pass "the harness names the marker as the reason it restarted" \
        || fail "the harness restart does not name the marker"
    if wait_ready "$OUT/condemned-supervised.log"; then
        CODE=$(status_of "$BASE/health" "$OUT/condemned-supervised-health.json")
        [ "$CODE" = "200" ] && pass "the replacement runtime answers /health 200 again" \
            || fail "the replacement runtime answered $CODE (want 200)"
    else
        fail "the restarted runtime never became ready"
    fi
else
    fail "supervised condemned run never became ready"
fi
stop_harness

# ====== 6c. NEGATIVE: a teardown that never sees the release must not lie =====
# THE FAIL-CLOSED PHASE, and the one revision 2 did not survive. That revision
# waited for MLX's process-global active bytes to fall below half their
# condemnation-time reading, threw the timeout away, and cleared the pool and
# attested the rebuild either way. So a teardown that observed nothing took the
# same transition as one that observed a release: on the 29 GB target model the
# crossing admits the clear with ~14.5 GB still active, and on a timeout the
# ledger reported a rebuild that had returned nothing at all.
#
# `--fault-inject-teardown-retain true` does not ask the runtime to report a
# timeout. It holds a strong reference to the real ModelContainer across the
# teardown, so the weights genuinely are never released and the barrier
# genuinely never observes one. The residue is measured here, from MLX's own
# active_bytes, so the phase cannot pass by driving a seam that retained nothing.
#
# Unsupervised, for the same reason 6a is: the marker this path emits would
# otherwise have the harness kill the process mid-measurement.
info "condemned 6c (negative, unsupervised): a teardown that never observes the release must not attest a rebuild"
write_config batch-teardown-retained "$INCIDENT" 1 "$AFTER_TOKENS" unsupervised true
start_harness batch-teardown-retained "$OUT/teardown-retained.log"
if wait_ready "$OUT/teardown-retained.log"; then
    pass "retained-teardown run reached ready"

    CODE=$(chat "$OUT/teardown-retained-chat.json")
    [ "$CODE" = "500" ] && pass "the exhaustion still answers the provoking request with 500" \
        || fail "the exhaustion answered $CODE (want 500)"

    # The whole bounded wait has to expire before the abandonment exists.
    info "waiting ${TEARDOWN_BUDGET_SECONDS}s for the bounded teardown wait to expire"
    sleep "$TEARDOWN_BUDGET_SECONDS"

    state_of "$OUT/teardown-retained-state.json" > /dev/null

    # First: establish that the seam did what it claims. Without this the rest
    # of the phase is a runtime with nothing to release being congratulated for
    # not releasing it.
    RETAINED_ACTIVE=$(mlx_of "$OUT/teardown-retained-state.json" active_bytes)
    RETAINED_CACHE=$(mlx_of "$OUT/teardown-retained-state.json" cache_bytes)
    info "retained teardown active_bytes=$RETAINED_ACTIVE cache_bytes=$RETAINED_CACHE"
    expect_at_least "$RETAINED_ACTIVE" "$RETAINED_ACTIVE_FLOOR" \
        "TEARDOWN GATE: the seam really is holding the condemned model"

    # ...and now the gate. Substantial residue is still resident, so the
    # runtime must report the pool as owed, not returned.
    expect_batch "$OUT/teardown-retained-state.json" shared_cache_rebuilds 0 \
        "TEARDOWN GATE: an unobserved release attests no rebuild"
    expect_batch "$OUT/teardown-retained-state.json" shared_cache_rebuilds_abandoned 1 \
        "TEARDOWN GATE: the abandoned rebuild is counted as abandoned"
    expect_batch_bool "$OUT/teardown-retained-state.json" shared_cache_rebuild_pending true \
        "TEARDOWN GATE: the pool is still reported as owed"

    # The batch entry is a separate fact and is released under every verdict,
    # including this one. A teardown that could not return the shared pool is
    # not a licence to leak the per-request state as well.
    expect_batch "$OUT/teardown-retained-state.json" active 0 \
        "a retained teardown still leaves no batch slot in flight"
    expect_batch "$OUT/teardown-retained-state.json" batches_released 1 \
        "a retained teardown still releases its batch"

    grep -q '"event":"generation_shared_cache_rebuilt"' "$OUT/teardown-retained.log" \
        && fail "TEARDOWN GATE: the runtime announced a rebuild it never performed" \
        || pass "TEARDOWN GATE: no completed-rebuild event for an unobserved release"
    python3 -c '
import json, sys
for line in open(sys.argv[1]):
    try:
        row = json.loads(line)
    except ValueError:
        continue
    if row.get("event") == "generation_shared_cache_rebuild_abandoned":
        assert row["release_observed"] is False, row
        assert row["shared_cache_rebuilds"] == 0, row
        assert row["shared_cache_rebuild_pending"] is True, row
        break
else:
    raise SystemExit("no abandoned event")
' "$OUT/teardown-retained.log" 2>/dev/null \
        && pass "TEARDOWN GATE: the timeout is reported explicitly, with release_observed=false" \
        || fail "TEARDOWN GATE: no explicit release_observed=false abandonment event"

    # ...and this is the interval THIS seam covers, stated so it cannot be
    # confused with 6d's. Parking the wrapper means its weak reference never
    # reads nil, so a barrier that consulted nothing but that reference would
    # also refuse here. That is exactly why this phase alone is not enough, and
    # why 6d exists.
    RETAINED_DEALLOC=$(event_field "$OUT/teardown-retained.log" generation_shared_cache_rebuild_abandoned container_deallocated)
    expect_equals "$RETAINED_DEALLOC" false \
        "TEARDOWN GATE: the container seam holds the wrapper itself, so weak-nil never happens"

    # Requirement 4 of the rework: the 2h39ya contract survives all of this.
    CODE=$(status_of "$BASE/health" "$OUT/teardown-retained-health.json")
    [ "$CODE" = "503" ] && pass "a failed teardown still leaves /health at 503" \
        || fail "a failed teardown moved /health to $CODE (want 503)"

    # A host holding a condemned model it could not return must not be left to
    # compete with a replacement for it. The marker is the only thing that ends
    # that, and it has to be emitted for the abandonment, not merely for the
    # condemnation that preceded it.
    grep -q "$MARKER.*did not release" "$OUT/teardown-retained.log" \
        && pass "TEARDOWN GATE: the abandonment re-announces the process as unusable" \
        || fail "TEARDOWN GATE: an abandoned rebuild left no supervision marker naming it"
else
    fail "retained-teardown run never became ready"
fi
stop_harness

# == 6d. NEGATIVE, NARROWED: the wrapper dies and the weights do not ==========
# THE INTERVAL 6c CANNOT REACH, and the one revision 3 did not survive.
#
# Revision 3 answered "are the weights gone?" from a single weak reference to
# the outer ModelContainer. Review narrowed the pinned dependency -- three
# seconds of delay in SerialAccessContainer<ModelContext> destruction, nothing
# else touched -- and every phase above stayed green while 6a produced
# `release_observed=true shared_cache_rebuilds=1 cache_bytes=0` with
# active_bytes=262361760. The runtime cleared the pool and attested a rebuild
# with the entire model resident, because ModelContainer is a wrapper and a
# Swift weak reference may read nil while destruction of the state below it is
# still running.
#
# 6c cannot catch that. It parks the wrapper, so weak-nil never happens and a
# wrapper-only barrier refuses for the wrong reason. This phase parks
# `ModelContext.model` INSTEAD -- the LanguageModel and its arrays, below the
# container -- and lets the container be deallocated exactly on schedule. So
# the disputed interval is not simulated: the wrapper really is gone, the
# weights really are still active, and the runtime has to answer from something
# other than the wrapper to get this right.
#
# Unsupervised, for the same reason 6a and 6c are.
info "condemned 6d (narrowed negative, unsupervised): the container is deallocated and the weights are not"
write_config batch-teardown-inner "$INCIDENT" 1 "$AFTER_TOKENS" unsupervised "" true
start_harness batch-teardown-inner "$OUT/teardown-inner.log"
if wait_ready "$OUT/teardown-inner.log"; then
    pass "inner-retention run reached ready"

    CODE=$(chat "$OUT/teardown-inner-chat.json")
    [ "$CODE" = "500" ] && pass "the exhaustion still answers the provoking request with 500" \
        || fail "the exhaustion answered $CODE (want 500)"

    info "waiting ${TEARDOWN_BUDGET_SECONDS}s for the bounded teardown wait to expire"
    sleep "$TEARDOWN_BUDGET_SECONDS"

    state_of "$OUT/teardown-inner-state.json" > /dev/null

    # First half of the seam's claim: the weights really are still resident.
    INNER_ACTIVE=$(mlx_of "$OUT/teardown-inner-state.json" active_bytes)
    INNER_CACHE=$(mlx_of "$OUT/teardown-inner-state.json" cache_bytes)
    info "inner-retention active_bytes=$INNER_ACTIVE cache_bytes=$INNER_CACHE"
    expect_at_least "$INNER_ACTIVE" "$RETAINED_ACTIVE_FLOOR" \
        "INNER GATE: the seam really is holding the condemned model's weights"

    # Second half, and the one that makes this phase a NARROWING of 6c rather
    # than a repeat of it: the wrapper this time really was deallocated. Without
    # this the phase would be indistinguishable from 6c, and a runtime that had
    # simply failed to arm the seam would pass both.
    INNER_DEALLOC=$(event_field "$OUT/teardown-inner.log" generation_shared_cache_rebuild_abandoned container_deallocated)
    expect_equals "$INNER_DEALLOC" true \
        "INNER GATE: the container itself WAS deallocated, so weak-nil alone would have said released"

    # ...and the runtime still refused. This is the assertion that kills a
    # wrapper-only barrier: with container_deallocated=true above, the only
    # thing left to refuse from is MLX's own accounting.
    INNER_FOOTPRINT=$(event_field "$OUT/teardown-inner.log" generation_shared_cache_rebuild_abandoned weight_footprint_bytes)
    INNER_RETURNED=$(event_field "$OUT/teardown-inner.log" generation_shared_cache_rebuild_abandoned returned_bytes)
    info "inner-retention footprint=$INNER_FOOTPRINT returned=$INNER_RETURNED"
    case "$INNER_RETURNED$INNER_FOOTPRINT" in
        *[!0-9]*) fail "INNER GATE: teardown figures unreadable" ;;
        *)
            if [ "$INNER_RETURNED" -lt "$INNER_FOOTPRINT" ]; then
                pass "INNER GATE: MLX never gave the weights back (${INNER_RETURNED}B of ${INNER_FOOTPRINT}B), and the runtime said so"
            else
                fail "INNER GATE: the runtime claims ${INNER_RETURNED}B of ${INNER_FOOTPRINT}B returned while ${INNER_ACTIVE}B is still active"
            fi
            ;;
    esac

    expect_batch "$OUT/teardown-inner-state.json" shared_cache_rebuilds 0 \
        "INNER GATE: a deallocated wrapper over live weights attests no rebuild"
    expect_batch "$OUT/teardown-inner-state.json" shared_cache_rebuilds_abandoned 1 \
        "INNER GATE: the rebuild is counted as abandoned"
    expect_batch_bool "$OUT/teardown-inner-state.json" shared_cache_rebuild_pending true \
        "INNER GATE: the pool is still reported as owed"

    grep -q '"event":"generation_shared_cache_rebuilt"' "$OUT/teardown-inner.log" \
        && fail "INNER GATE: the runtime announced a rebuild with the weights still active" \
        || pass "INNER GATE: no completed-rebuild event while the weights are still active"

    # The batch entry is a separate fact and is released under every verdict.
    expect_batch "$OUT/teardown-inner-state.json" active 0 \
        "an inner-retained teardown still leaves no batch slot in flight"
    expect_batch "$OUT/teardown-inner-state.json" batches_released 1 \
        "an inner-retained teardown still releases its batch"

    # Requirement 4 of the rework, on this path too.
    CODE=$(status_of "$BASE/health" "$OUT/teardown-inner-health.json")
    [ "$CODE" = "503" ] && pass "an inner-retained teardown still leaves /health at 503" \
        || fail "an inner-retained teardown moved /health to $CODE (want 503)"

    grep -q "$MARKER.*did not release" "$OUT/teardown-inner.log" \
        && pass "INNER GATE: the abandonment re-announces the process as unusable" \
        || fail "INNER GATE: an abandoned rebuild left no supervision marker naming it"
else
    fail "inner-retention run never became ready"
fi
stop_harness

# = 6e. NEGATIVE, LONG CONTEXT: the request outweighs the model ===============
# REVIEW'S OWN REPRODUCTION, carried into the maintained suite.
#
# Revision 4 answered "are the weights gone?" from a PROCESS-GLOBAL byte delta:
# baseline_active_bytes - active_bytes >= weight_footprint_bytes. Review drove
# the same production path with a six-thousand-word prompt, which makes the
# failed request's own KV state larger than the model, and the release of that
# request alone satisfied the subtraction. Two consecutive runs attested a
# completed rebuild -- 608,909,592 B "returned" against a 262,361,760 B
# footprint -- with post-teardown active_bytes at exactly 262,361,760: every
# weight still resident.
#
# 6d cannot reach this. Its short control prompt returns less than the footprint,
# so the revision-4 gate refused there for the right reason by accident. What
# separates this phase is the assertion that the bypass condition really is
# present: returned_bytes must CLEAR the footprint here, and the runtime must
# still abandon.
info "condemned 6e (negative, long context, unsupervised): a request larger than the model must not pay for its release"
write_long_body
write_config batch-teardown-long "$INCIDENT" 1 "$AFTER_TOKENS" unsupervised "" true
start_harness batch-teardown-long "$OUT/teardown-long.log"
if wait_ready "$OUT/teardown-long.log"; then
    pass "long-context retention run reached ready"

    CODE=$(chat_long "$OUT/teardown-long-chat.json")
    [ "$CODE" = "500" ] && pass "the exhaustion answers the long-context request with 500" \
        || fail "the long-context exhaustion answered $CODE (want 500)"

    info "waiting ${TEARDOWN_BUDGET_SECONDS}s for the bounded teardown wait to expire"
    sleep "$TEARDOWN_BUDGET_SECONDS"

    state_of "$OUT/teardown-long-state.json" > /dev/null
    LONG_ACTIVE=$(mlx_of "$OUT/teardown-long-state.json" active_bytes)
    LONG_FOOTPRINT=$(event_field "$OUT/teardown-long.log" generation_shared_cache_rebuild_abandoned weight_footprint_bytes)
    LONG_RETURNED=$(event_field "$OUT/teardown-long.log" generation_shared_cache_rebuild_abandoned returned_bytes)
    LONG_BASELINE=$(event_field "$OUT/teardown-long.log" generation_shared_cache_rebuild_abandoned baseline_active_bytes)
    LONG_DEALLOC=$(event_field "$OUT/teardown-long.log" generation_shared_cache_rebuild_abandoned container_deallocated)
    info "long-context footprint=$LONG_FOOTPRINT baseline=$LONG_BASELINE returned=$LONG_RETURNED active=$LONG_ACTIVE"

    # The seam's claim: the weights really are still resident.
    expect_at_least "$LONG_ACTIVE" "$RETAINED_ACTIVE_FLOOR" \
        "LONG-CONTEXT GATE: the seam really is holding the condemned model's weights"
    expect_equals "$LONG_DEALLOC" true \
        "LONG-CONTEXT GATE: the container itself WAS deallocated"

    # THE BYPASS CONDITION. Without this the phase would be a slower 6d: a run
    # where the request happened to be small would abandon for the old reason
    # and the suite would report the new one as proven.
    case "$LONG_RETURNED$LONG_FOOTPRINT" in
        *[!0-9]*) fail "LONG-CONTEXT GATE: teardown figures unreadable" ;;
        *)
            if [ "$LONG_RETURNED" -ge "$LONG_FOOTPRINT" ]; then
                pass "LONG-CONTEXT GATE: the process-global delta WAS satisfied (${LONG_RETURNED}B >= ${LONG_FOOTPRINT}B) -- revision 4 would have attested here"
            else
                fail "LONG-CONTEXT GATE: the request did not outweigh the model (${LONG_RETURNED}B < ${LONG_FOOTPRINT}B); this phase did not reproduce the bypass and proves nothing"
            fi
            ;;
    esac
    # ...and the absolute residue that must beat it.
    case "$LONG_ACTIVE$LONG_FOOTPRINT" in
        *[!0-9]*) fail "LONG-CONTEXT GATE: allocator figures unreadable" ;;
        *)
            if [ "$LONG_ACTIVE" -ge "$LONG_FOOTPRINT" ]; then
                pass "LONG-CONTEXT GATE: the whole model is still resident (${LONG_ACTIVE}B >= ${LONG_FOOTPRINT}B), and the runtime refused anyway"
            else
                fail "LONG-CONTEXT GATE: the seam did not hold the model (${LONG_ACTIVE}B < ${LONG_FOOTPRINT}B)"
            fi
            ;;
    esac

    grep -q '"event":"generation_shared_cache_rebuilt"' "$OUT/teardown-long.log" \
        && fail "LONG-CONTEXT GATE: MEASUREMENT GATE BYPASS -- a rebuild was attested while the retained model weights remain active" \
        || pass "LONG-CONTEXT GATE: no completed-rebuild event while the retained weights remain active"
    expect_batch "$OUT/teardown-long-state.json" shared_cache_rebuilds 0 \
        "LONG-CONTEXT GATE: no rebuild attested"
    expect_batch "$OUT/teardown-long-state.json" shared_cache_rebuilds_abandoned 1 \
        "LONG-CONTEXT GATE: the rebuild is counted as abandoned"
    expect_batch_bool "$OUT/teardown-long-state.json" shared_cache_rebuild_pending true \
        "LONG-CONTEXT GATE: the pool is still reported as owed"
    expect_batch "$OUT/teardown-long-state.json" active 0 \
        "a long-context failure still leaves no batch slot in flight"
    expect_batch "$OUT/teardown-long-state.json" batches_released 1 \
        "a long-context failure still releases its batch"

    CODE=$(status_of "$BASE/health" "$OUT/teardown-long-health.json")
    [ "$CODE" = "503" ] && pass "a long-context abandoned teardown still leaves /health at 503" \
        || fail "a long-context abandoned teardown moved /health to $CODE (want 503)"

    grep -q "$MARKER.*did not release" "$OUT/teardown-long.log" \
        && pass "LONG-CONTEXT GATE: the abandonment re-announces the process as unusable" \
        || fail "LONG-CONTEXT GATE: an abandoned rebuild left no supervision marker naming it"
else
    fail "long-context retention run never became ready"
fi
stop_harness

# = 6f. NEGATIVE, NARROWED: every byte clause is green and the weights are held
# THE CLASS NO BYTE COMPARISON CAN REACH, and the reason the runtime registers
# this model's module tree at all.
#
# 6e leaves the WHOLE model resident, so an absolute-residue check refuses it on
# the bytes. This phase parks a strict subset -- the second half of the
# flattened module tree, with the container and the root model object dying on
# schedule. What MLX then reports is a residue BELOW the model's load footprint,
# which is exactly what a fully released model looks like to a process-global
# counter, while this model's weights are demonstrably still owned. With the
# long-context request behind it the returned-byte comparison clears the
# footprint too.
#
# So every byte-derived clause of the release gate reads green here, and the
# only thing left to refuse from is ownership. A runtime that dropped the
# registry would pass 6c, 6d and 6e and fail this phase alone.
info "condemned 6f (narrowed negative, unsupervised): part of the model is still owned and every byte clause says released"
write_config batch-teardown-modules "$INCIDENT" 1 "$AFTER_TOKENS" unsupervised "" "" true
start_harness batch-teardown-modules "$OUT/teardown-modules.log"
if wait_ready "$OUT/teardown-modules.log"; then
    pass "module-subset retention run reached ready"

    CODE=$(chat_long "$OUT/teardown-modules-chat.json")
    [ "$CODE" = "500" ] && pass "the exhaustion answers the provoking request with 500" \
        || fail "the exhaustion answered $CODE (want 500)"

    info "waiting ${TEARDOWN_BUDGET_SECONDS}s for the bounded teardown wait to expire"
    sleep "$TEARDOWN_BUDGET_SECONDS"

    state_of "$OUT/teardown-modules-state.json" > /dev/null
    SUBSET_ACTIVE=$(mlx_of "$OUT/teardown-modules-state.json" active_bytes)
    SUBSET_FOOTPRINT=$(event_field "$OUT/teardown-modules.log" generation_shared_cache_rebuild_abandoned weight_footprint_bytes)
    SUBSET_RETURNED=$(event_field "$OUT/teardown-modules.log" generation_shared_cache_rebuild_abandoned returned_bytes)
    SUBSET_OBSERVED=$(event_field "$OUT/teardown-modules.log" generation_shared_cache_rebuild_abandoned active_bytes)
    SUBSET_DEALLOC=$(event_field "$OUT/teardown-modules.log" generation_shared_cache_rebuild_abandoned container_deallocated)
    SUBSET_OWNERS=$(event_field "$OUT/teardown-modules.log" generation_shared_cache_rebuild_abandoned weight_owner_count)
    SUBSET_LIVE=$(event_field "$OUT/teardown-modules.log" generation_shared_cache_rebuild_abandoned live_weight_owners)
    SUBSET_INFLIGHT=$(event_field "$OUT/teardown-modules.log" generation_shared_cache_rebuild_abandoned generations_in_flight)
    SUBSET_STABLE=$(event_field "$OUT/teardown-modules.log" generation_shared_cache_rebuild_abandoned stable_active_samples)
    info "module-subset owners=$SUBSET_OWNERS live=$SUBSET_LIVE in_flight=$SUBSET_INFLIGHT stable=$SUBSET_STABLE footprint=$SUBSET_FOOTPRINT returned=$SUBSET_RETURNED observed_active=$SUBSET_OBSERVED state_active=$SUBSET_ACTIVE"

    # First: the state really is the one this phase claims. The wrapper died,
    # a strict subset of the tree did not, and the retention is real rather
    # than a seam that quietly let go.
    expect_equals "$SUBSET_DEALLOC" true \
        "SUBSET GATE: the container itself WAS deallocated"
    case "$SUBSET_OWNERS" in
        *[!0-9]*|0) fail "SUBSET GATE: the weight-owner registry was never populated (got ${SUBSET_OWNERS:-unreadable})" ;;
        *) pass "SUBSET GATE: this model's module tree was registered (${SUBSET_OWNERS} owners)" ;;
    esac
    case "$SUBSET_LIVE$SUBSET_OWNERS" in
        *[!0-9]*) fail "SUBSET GATE: owner figures unreadable" ;;
        *)
            if [ "$SUBSET_LIVE" -gt 0 ] && [ "$SUBSET_LIVE" -lt "$SUBSET_OWNERS" ]; then
                pass "SUBSET GATE: a STRICT subset of this model's weights is still owned (${SUBSET_LIVE} of ${SUBSET_OWNERS})"
            else
                fail "SUBSET GATE: the seam did not retain a strict subset (${SUBSET_LIVE} of ${SUBSET_OWNERS} live); this phase is then either 6e or nothing"
            fi
            ;;
    esac

    # Second, and this is what makes the phase a NARROWING: every byte-derived
    # clause of the gate is satisfied. Residue below the footprint, a
    # process-global drop that clears the footprint, nothing in flight, and a
    # reading that has come to rest. A gate without ownership attests here.
    expect_between "$SUBSET_OBSERVED" "$SUBSET_FOOTPRINT" \
        "SUBSET GATE: the ABSOLUTE RESIDUE clause is satisfied -- less than a model is resident"
    case "$SUBSET_RETURNED$SUBSET_FOOTPRINT" in
        *[!0-9]*) fail "SUBSET GATE: teardown figures unreadable" ;;
        *)
            if [ "$SUBSET_RETURNED" -ge "$SUBSET_FOOTPRINT" ]; then
                pass "SUBSET GATE: the process-global delta clause is satisfied (${SUBSET_RETURNED}B >= ${SUBSET_FOOTPRINT}B)"
            else
                fail "SUBSET GATE: the delta clause was not satisfied (${SUBSET_RETURNED}B < ${SUBSET_FOOTPRINT}B); ownership is then not the only clause refusing and the narrowing is not proven"
            fi
            ;;
    esac
    expect_equals "$SUBSET_INFLIGHT" 0 \
        "SUBSET GATE: the in-flight clause is satisfied"
    case "$SUBSET_STABLE" in
        *[!0-9]*) fail "SUBSET GATE: the stability run is unreadable" ;;
        *)
            if [ "$SUBSET_STABLE" -ge 3 ]; then
                pass "SUBSET GATE: the at-rest clause is satisfied (${SUBSET_STABLE} identical samples)"
            else
                fail "SUBSET GATE: the reading never came to rest (${SUBSET_STABLE} samples); ownership is then not the only clause refusing"
            fi
            ;;
    esac

    # Third: the runtime refused anyway. With every other clause green above,
    # this can only have come from ownership.
    grep -q '"event":"generation_shared_cache_rebuilt"' "$OUT/teardown-modules.log" \
        && fail "SUBSET GATE: a rebuild was attested while ${SUBSET_LIVE} of this model's weight owners are still alive" \
        || pass "SUBSET GATE: no completed-rebuild event while part of the model is still owned"
    expect_batch "$OUT/teardown-modules-state.json" shared_cache_rebuilds 0 \
        "SUBSET GATE: no rebuild attested"
    expect_batch "$OUT/teardown-modules-state.json" shared_cache_rebuilds_abandoned 1 \
        "SUBSET GATE: the rebuild is counted as abandoned"
    expect_batch_bool "$OUT/teardown-modules-state.json" shared_cache_rebuild_pending true \
        "SUBSET GATE: the pool is still reported as owed"
    expect_batch "$OUT/teardown-modules-state.json" active 0 \
        "a partially-retained teardown still leaves no batch slot in flight"
    expect_batch "$OUT/teardown-modules-state.json" batches_released 1 \
        "a partially-retained teardown still releases its batch"

    CODE=$(status_of "$BASE/health" "$OUT/teardown-modules-health.json")
    [ "$CODE" = "503" ] && pass "a partially-retained teardown still leaves /health at 503" \
        || fail "a partially-retained teardown moved /health to $CODE (want 503)"

    grep -q "$MARKER.*did not release" "$OUT/teardown-modules.log" \
        && pass "SUBSET GATE: the abandonment re-announces the process as unusable" \
        || fail "SUBSET GATE: an abandoned rebuild left no supervision marker naming it"
else
    fail "module-subset retention run never became ready"
fi
stop_harness

# = 6g. NEGATIVE, NARROWED: ownership says released and the model is resident ==
# THE MIRROR OF 6f, and the production negative for the ABSOLUTE RESIDUE clause.
#
# 6c, 6d, 6e and 6f all keep some object of the model tree alive, so the
# ownership clause refuses them and no byte reading is ever the deciding vote.
# This phase keeps NOTHING of the tree: --fault-inject-teardown-retain-weight-arrays
# copies the flattened parameter arrays out and holds those. `MLXArray` is a
# value type over a shared buffer handle, so every Module -- and the container
# above them -- is deallocated exactly on schedule while the buffers stay alive.
#
# The runtime therefore sees container_deallocated=true, live_weight_owners=0,
# nothing in flight, a reading at rest, and a process-global drop that clears
# the footprint. Every clause except one says "released". MLX says the whole
# 262 MB model is still active, and that has to be enough on its own.
#
# It is not a hypothetical shape either: anything that caches, snapshots or
# exports a model's parameters produces it, and to a process-global counter it
# is indistinguishable from a released model.
info "condemned 6g (narrowed negative, unsupervised): ownership says released and the whole model is still resident"
write_config batch-teardown-arrays "$INCIDENT" 1 "$AFTER_TOKENS" unsupervised "" "" "" true
start_harness batch-teardown-arrays "$OUT/teardown-arrays.log"
if wait_ready "$OUT/teardown-arrays.log"; then
    pass "weight-array retention run reached ready"

    CODE=$(chat_long "$OUT/teardown-arrays-chat.json")
    [ "$CODE" = "500" ] && pass "the exhaustion answers the provoking request with 500" \
        || fail "the exhaustion answered $CODE (want 500)"

    info "waiting ${TEARDOWN_BUDGET_SECONDS}s for the bounded teardown wait to expire"
    sleep "$TEARDOWN_BUDGET_SECONDS"

    state_of "$OUT/teardown-arrays-state.json" > /dev/null
    ARRAY_ACTIVE=$(mlx_of "$OUT/teardown-arrays-state.json" active_bytes)
    ARRAY_FOOTPRINT=$(event_field "$OUT/teardown-arrays.log" generation_shared_cache_rebuild_abandoned weight_footprint_bytes)
    ARRAY_RETURNED=$(event_field "$OUT/teardown-arrays.log" generation_shared_cache_rebuild_abandoned returned_bytes)
    ARRAY_OBSERVED=$(event_field "$OUT/teardown-arrays.log" generation_shared_cache_rebuild_abandoned active_bytes)
    ARRAY_DEALLOC=$(event_field "$OUT/teardown-arrays.log" generation_shared_cache_rebuild_abandoned container_deallocated)
    ARRAY_OWNERS=$(event_field "$OUT/teardown-arrays.log" generation_shared_cache_rebuild_abandoned weight_owner_count)
    ARRAY_LIVE=$(event_field "$OUT/teardown-arrays.log" generation_shared_cache_rebuild_abandoned live_weight_owners)
    ARRAY_INFLIGHT=$(event_field "$OUT/teardown-arrays.log" generation_shared_cache_rebuild_abandoned generations_in_flight)
    ARRAY_STABLE=$(event_field "$OUT/teardown-arrays.log" generation_shared_cache_rebuild_abandoned stable_active_samples)
    info "weight-array owners=$ARRAY_OWNERS live=$ARRAY_LIVE in_flight=$ARRAY_INFLIGHT stable=$ARRAY_STABLE footprint=$ARRAY_FOOTPRINT returned=$ARRAY_RETURNED observed_active=$ARRAY_OBSERVED state_active=$ARRAY_ACTIVE"

    # Every clause except the residue is satisfied, and each one is asserted so
    # the refusal below cannot be attributed to any of them.
    expect_equals "$ARRAY_DEALLOC" true \
        "ARRAY GATE: the container itself WAS deallocated"
    case "$ARRAY_OWNERS" in
        *[!0-9]*|0) fail "ARRAY GATE: the weight-owner registry was never populated (got ${ARRAY_OWNERS:-unreadable})" ;;
        *) pass "ARRAY GATE: this model's module tree was registered (${ARRAY_OWNERS} owners)" ;;
    esac
    expect_equals "$ARRAY_LIVE" 0 \
        "ARRAY GATE: the OWNERSHIP clause is satisfied -- no Module of this model is alive"
    expect_equals "$ARRAY_INFLIGHT" 0 \
        "ARRAY GATE: the in-flight clause is satisfied"
    case "$ARRAY_STABLE" in
        *[!0-9]*) fail "ARRAY GATE: the stability run is unreadable" ;;
        *)
            if [ "$ARRAY_STABLE" -ge 3 ]; then
                pass "ARRAY GATE: the at-rest clause is satisfied (${ARRAY_STABLE} identical samples)"
            else
                fail "ARRAY GATE: the reading never came to rest (${ARRAY_STABLE} samples)"
            fi
            ;;
    esac
    case "$ARRAY_RETURNED$ARRAY_FOOTPRINT" in
        *[!0-9]*) fail "ARRAY GATE: teardown figures unreadable" ;;
        *)
            if [ "$ARRAY_RETURNED" -ge "$ARRAY_FOOTPRINT" ]; then
                pass "ARRAY GATE: the process-global delta clause is satisfied (${ARRAY_RETURNED}B >= ${ARRAY_FOOTPRINT}B)"
            else
                fail "ARRAY GATE: the delta clause was not satisfied (${ARRAY_RETURNED}B < ${ARRAY_FOOTPRINT}B); the residue clause is then not the only one refusing"
            fi
            ;;
    esac

    # ...and the one clause that must refuse: the model is still resident.
    case "$ARRAY_OBSERVED$ARRAY_FOOTPRINT" in
        *[!0-9]*) fail "ARRAY GATE: allocator figures unreadable" ;;
        *)
            if [ "$ARRAY_OBSERVED" -ge "$ARRAY_FOOTPRINT" ]; then
                pass "ARRAY GATE: the whole model is still resident (${ARRAY_OBSERVED}B >= ${ARRAY_FOOTPRINT}B), and it is the only clause left to refuse"
            else
                fail "ARRAY GATE: the seam did not hold the weights (${ARRAY_OBSERVED}B < ${ARRAY_FOOTPRINT}B); this phase then proves nothing"
            fi
            ;;
    esac
    expect_at_least "$ARRAY_ACTIVE" "$RETAINED_ACTIVE_FLOOR" \
        "ARRAY GATE: the residue is still there when the endpoint is asked"

    grep -q '"event":"generation_shared_cache_rebuilt"' "$OUT/teardown-arrays.log" \
        && fail "ARRAY GATE: a rebuild was attested with the whole model still active and no owner alive to say so" \
        || pass "ARRAY GATE: no completed-rebuild event while the model is still resident"
    expect_batch "$OUT/teardown-arrays-state.json" shared_cache_rebuilds 0 \
        "ARRAY GATE: no rebuild attested"
    expect_batch "$OUT/teardown-arrays-state.json" shared_cache_rebuilds_abandoned 1 \
        "ARRAY GATE: the rebuild is counted as abandoned"
    expect_batch_bool "$OUT/teardown-arrays-state.json" shared_cache_rebuild_pending true \
        "ARRAY GATE: the pool is still reported as owed"
    expect_batch "$OUT/teardown-arrays-state.json" active 0 \
        "an array-retained teardown still leaves no batch slot in flight"
    expect_batch "$OUT/teardown-arrays-state.json" batches_released 1 \
        "an array-retained teardown still releases its batch"

    CODE=$(status_of "$BASE/health" "$OUT/teardown-arrays-health.json")
    [ "$CODE" = "503" ] && pass "an array-retained teardown still leaves /health at 503" \
        || fail "an array-retained teardown moved /health to $CODE (want 503)"

    grep -q "$MARKER.*did not release" "$OUT/teardown-arrays.log" \
        && pass "ARRAY GATE: the abandonment re-announces the process as unusable" \
        || fail "ARRAY GATE: an abandoned rebuild left no supervision marker naming it"
else
    fail "weight-array retention run never became ready"
fi
stop_harness

# = 6h. NEGATIVE, NARROWED: review's revision-5 bypass, kept as a live input ==
# THE PHASE THIS REVISION EXISTS FOR.
#
# 6g holds EVERY parameter array, so the residue lands at or above the model's
# footprint and any footprint-relative check refuses it. 6f holds a strict
# subset of the module tree, so ownership refuses first and no byte reading is
# ever the deciding vote. Neither can produce the combination review found:
#
#   * zero live Module owners      -- ownership says the model is released
#   * container deallocated        -- the weak veto says the same
#   * nothing in flight, at rest   -- the reading is attributable and quiet
#   * returned_bytes >= footprint  -- the process-global delta clause is met
#   * residue SIGNIFICANT but BELOW the footprint
#
# --fault-inject-teardown-retain-weight-array-subset produces exactly that: the
# largest half of the flattened parameter arrays by nbytes, copied out, so every
# Module dies on schedule while ~255 MB of a ~262 MB model stays resident.
# Review measured 255,724,192 B against a 262,361,760 B footprint and revision 5
# attested a completed release over it.
#
# Driven with the long request, so returned_bytes clears the footprint and the
# ONLY clause capable of refusing is the residue.
info "condemned 6h (narrowed negative, unsupervised): a strict subset of copied weight arrays, below the footprint"
write_config batch-teardown-array-subset "$INCIDENT" 1 "$AFTER_TOKENS" unsupervised "" "" "" "" true
start_harness batch-teardown-array-subset "$OUT/teardown-array-subset.log"
if wait_ready "$OUT/teardown-array-subset.log"; then
    pass "narrowed weight-array retention run reached ready"

    CODE=$(chat_long "$OUT/teardown-array-subset-chat.json")
    [ "$CODE" = "500" ] && pass "the exhaustion answers the provoking request with 500" \
        || fail "the exhaustion answered $CODE (want 500)"

    info "waiting ${TEARDOWN_BUDGET_SECONDS}s for the bounded teardown wait to expire"
    sleep "$TEARDOWN_BUDGET_SECONDS"

    state_of "$OUT/teardown-array-subset-state.json" > /dev/null
    SUBSET_ACTIVE=$(mlx_of "$OUT/teardown-array-subset-state.json" active_bytes)
    SUBSET_FOOTPRINT=$(event_field "$OUT/teardown-array-subset.log" generation_shared_cache_rebuild_abandoned weight_footprint_bytes)
    SUBSET_RETURNED=$(event_field "$OUT/teardown-array-subset.log" generation_shared_cache_rebuild_abandoned returned_bytes)
    SUBSET_OBSERVED=$(event_field "$OUT/teardown-array-subset.log" generation_shared_cache_rebuild_abandoned observed_active_bytes)
    SUBSET_DEALLOC=$(event_field "$OUT/teardown-array-subset.log" generation_shared_cache_rebuild_abandoned container_deallocated)
    SUBSET_OWNERS=$(event_field "$OUT/teardown-array-subset.log" generation_shared_cache_rebuild_abandoned weight_owner_count)
    SUBSET_LIVE=$(event_field "$OUT/teardown-array-subset.log" generation_shared_cache_rebuild_abandoned live_weight_owners)
    SUBSET_INFLIGHT=$(event_field "$OUT/teardown-array-subset.log" generation_shared_cache_rebuild_abandoned generations_in_flight)
    SUBSET_STABLE=$(event_field "$OUT/teardown-array-subset.log" generation_shared_cache_rebuild_abandoned stable_active_samples)
    SUBSET_ALLOWANCE=$(event_field "$OUT/teardown-array-subset.log" generation_shared_cache_rebuild_abandoned residual_non_weight_allowance_bytes)
    info "array-subset owners=$SUBSET_OWNERS live=$SUBSET_LIVE in_flight=$SUBSET_INFLIGHT stable=$SUBSET_STABLE footprint=$SUBSET_FOOTPRINT returned=$SUBSET_RETURNED observed_active=$SUBSET_OBSERVED allowance=$SUBSET_ALLOWANCE state_active=$SUBSET_ACTIVE"

    # Every clause except the residue is asserted GREEN, so the refusal below
    # cannot be attributed to any of them. This is what makes the phase a
    # narrowing of the residue clause rather than a restatement of 6f.
    expect_equals "$SUBSET_DEALLOC" true \
        "SUBSET GATE: the container itself WAS deallocated"
    case "$SUBSET_OWNERS" in
        *[!0-9]*|0) fail "SUBSET GATE: the weight-owner registry was never populated (got ${SUBSET_OWNERS:-unreadable})" ;;
        *) pass "SUBSET GATE: this model's module tree was registered (${SUBSET_OWNERS} owners)" ;;
    esac
    expect_equals "$SUBSET_LIVE" 0 \
        "SUBSET GATE: the OWNERSHIP clause is satisfied -- no Module of this model is alive"
    expect_equals "$SUBSET_INFLIGHT" 0 \
        "SUBSET GATE: the in-flight clause is satisfied"
    case "$SUBSET_STABLE" in
        *[!0-9]*) fail "SUBSET GATE: the stability run is unreadable" ;;
        *)
            if [ "$SUBSET_STABLE" -ge 3 ]; then
                pass "SUBSET GATE: the at-rest clause is satisfied (${SUBSET_STABLE} identical samples)"
            else
                fail "SUBSET GATE: the reading never came to rest (${SUBSET_STABLE} samples)"
            fi
            ;;
    esac
    case "$SUBSET_RETURNED$SUBSET_FOOTPRINT" in
        *[!0-9]*) fail "SUBSET GATE: teardown figures unreadable" ;;
        *)
            if [ "$SUBSET_RETURNED" -ge "$SUBSET_FOOTPRINT" ]; then
                pass "SUBSET GATE: the process-global delta clause is satisfied (${SUBSET_RETURNED}B >= ${SUBSET_FOOTPRINT}B)"
            else
                fail "SUBSET GATE: the delta clause was not satisfied (${SUBSET_RETURNED}B < ${SUBSET_FOOTPRINT}B); the residue clause is then not the only one refusing"
            fi
            ;;
    esac

    # THE INTERVAL ITSELF. The residue has to be strictly BELOW the footprint --
    # otherwise this is 6g under another name and says nothing about the band
    # revision 5 admitted -- and large enough to be weights rather than jitter.
    case "$SUBSET_OBSERVED$SUBSET_FOOTPRINT" in
        *[!0-9]*) fail "SUBSET GATE: allocator figures unreadable" ;;
        *)
            if [ "$SUBSET_OBSERVED" -lt "$SUBSET_FOOTPRINT" ]; then
                pass "SUBSET GATE: the residue is INSIDE the interval revision 5 admitted (${SUBSET_OBSERVED}B < ${SUBSET_FOOTPRINT}B)"
            else
                fail "SUBSET GATE: the residue reached the footprint (${SUBSET_OBSERVED}B >= ${SUBSET_FOOTPRINT}B); the seam did not narrow and this phase repeats 6g"
            fi
            ;;
    esac
    expect_at_least "$SUBSET_OBSERVED" "$RETAINED_ACTIVE_FLOOR" \
        "SUBSET GATE: the residue is weight-sized, not allocator jitter"
    expect_at_least "$SUBSET_ACTIVE" "$RETAINED_ACTIVE_FLOOR" \
        "SUBSET GATE: the residue is still there when the endpoint is asked"
    expect_equals "$SUBSET_ALLOWANCE" 0 \
        "SUBSET GATE: the allowance the residue was refused against is zero"

    # ...and the runtime refuses anyway. Under revision 5 every assertion above
    # held and THIS is what came out as `generation_shared_cache_rebuilt`.
    grep -q '"event":"generation_shared_cache_rebuilt"' "$OUT/teardown-array-subset.log" \
        && fail "SUBSET GATE: a rebuild was attested with ${SUBSET_OBSERVED}B of weights still resident and no owner alive to say so" \
        || pass "SUBSET GATE: no completed-rebuild event for a below-footprint weight residue"
    expect_batch "$OUT/teardown-array-subset-state.json" shared_cache_rebuilds 0 \
        "SUBSET GATE: no rebuild attested"
    expect_batch "$OUT/teardown-array-subset-state.json" shared_cache_rebuilds_abandoned 1 \
        "SUBSET GATE: the rebuild is counted as abandoned"
    expect_batch_bool "$OUT/teardown-array-subset-state.json" shared_cache_rebuild_pending true \
        "SUBSET GATE: the pool is still reported as owed"
    expect_batch "$OUT/teardown-array-subset-state.json" active 0 \
        "a narrowed array-retained teardown still leaves no batch slot in flight"
    expect_batch "$OUT/teardown-array-subset-state.json" batches_released 1 \
        "a narrowed array-retained teardown still releases its batch"

    CODE=$(status_of "$BASE/health" "$OUT/teardown-array-subset-health.json")
    [ "$CODE" = "503" ] && pass "a narrowed array-retained teardown still leaves /health at 503" \
        || fail "a narrowed array-retained teardown moved /health to $CODE (want 503)"

    grep -q "$MARKER.*did not release" "$OUT/teardown-array-subset.log" \
        && pass "SUBSET GATE: the abandonment re-announces the process as unusable" \
        || fail "SUBSET GATE: an abandoned rebuild left no supervision marker naming it"
else
    fail "narrowed weight-array retention run never became ready"
fi
stop_harness

printf '\n%s\n' "----------------------------------------"
if [ "$FAILURES" -eq 0 ]; then
    printf 'GENERATION-BATCH-RECOVERY SMOKE OK (0 failures)\n'
    exit 0
fi
printf 'GENERATION-BATCH-RECOVERY SMOKE FAILED (%d failures)\n' "$FAILURES"
exit 1
