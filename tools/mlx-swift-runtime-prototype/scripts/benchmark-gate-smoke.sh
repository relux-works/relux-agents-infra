#!/usr/bin/env bash
# Production-entry checks for the migration benchmark, at the real subcommands.
#
# Every case here drives the shipped binary as an unmodified caller would: files
# and flags in, exit code and stdout out. No source is edited and no seam is
# armed. The contract suite in
# `Tests/MLXSwiftRuntimeContractTests/RuntimeBenchmarkTests.swift` proves that
# `RuntimeBenchmark.admit` refuses these shapes; only this script proves the
# shipped subcommands actually call it, and only this script can drive the
# shapes the contract suite cannot express — an attestation directory with
# nothing in it, an omitted flag, and a runtime process that answers one
# endpoint and serves nothing.
#
# THE CONTROL IS NOT FABRICATED, AND THIS TIME THAT CLAIM IS ABOUT THE
# MEASUREMENTS. Review round 3 was right that the previous version's claim was
# false: its two attestations were real, and every number beside them was
# invented by the script. Here the control is one `benchmark-run` invocation
# that spawns both runtimes through `model-harness`, drives every scenario
# against them itself, times and samples them itself, and seals and judges what
# it measured — the same code path the real 28 GB comparison uses. The script
# supplies configuration and prompts; it cannot supply a measurement, because
# the subcommand has no argument that takes one.
#
# The runtimes are stand-ins, not the real model: `fake-runtime.py` serves
# `/v1/models` and `/v1/chat/completions` in about a second, where the real pass
# takes an hour. Two stand-ins are used, one that serves and one that only
# lists, and the second is review's own reproduction.
#
# Exit codes under test:
#   benchmark-run      0 accepted   2 usage   3 rejected   4 inadmissible
#   benchmark-compare               2 usage   3 replayed   4 inadmissible
#                                   (never 0: a replay cannot grant acceptance)
#
# Usage: BINARY=/path/to/mlx-swift-runtime-prototype scripts/benchmark-gate-smoke.sh

set -uo pipefail

BINARY="${BINARY:?set BINARY to the prototype executable}"
HARNESS="${HARNESS:-/Users/alexis/.local/bin/model-harness}"
OUT="${OUT:-./benchmark-gate-out}"
PORT="${PORT:-18771}"
BASELINE_PYTHON="${BASELINE_PYTHON:-/usr/bin/python3}"
CANDIDATE_PYTHON="${CANDIDATE_PYTHON:-/opt/homebrew/bin/python3}"
rm -rf "$OUT"
mkdir -p "$OUT"

FAILURES=0
pass() { printf 'PASS  %s\n' "$*"; }
fail() { printf 'FAIL  %s\n' "$*"; FAILURES=$((FAILURES + 1)); }
info() { printf 'INFO  %s\n' "$*"; }

MODEL="$OUT/model"
mkdir -p "$MODEL"
cat > "$MODEL/config.json" <<'JSON'
{"model_type": "gate-smoke", "quantization": {"bits": 8, "group_size": 64, "mode": "affine"}}
JSON
cat > "$MODEL/model.safetensors.index.json" <<'JSON'
{"metadata": {"total_size": 1}, "weight_map": {"a": "model-00001.safetensors"}}
JSON

# ------------------------------------------------------------ the stand-ins --
#
# One serves; one only lists. The second is the process review used to obtain
# `accepted=true` from the previous revision in 7.2 seconds.
cat > "$OUT/fake-runtime.py" <<'PY'
import json, sys, time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

PORT = int(sys.argv[1])
MODEL = sys.argv[2]
MODE = sys.argv[3] if len(sys.argv) > 3 else "serving"


class Handler(BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"

    def _json(self, document, status=200):
        body = json.dumps(document).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def do_GET(self):
        if self.path.endswith("/models"):
            self._json({"data": [{"id": MODEL, "object": "model"}]})
            return
        self._json({"error": "not found"}, status=404)

    def do_POST(self):
        # Drained before answering, always. A keep-alive connection whose
        # request body is left in the socket makes the next request
        # unparseable, which would look like the gate failing to read rather
        # than like this placeholder refusing to serve.
        length = int(self.headers.get("Content-Length") or 0)
        raw = self.rfile.read(length)
        if MODE == "models-only":
            self._json({"error": "this process only lists models"}, status=404)
            return
        payload = json.loads(raw or b"{}")
        if payload.get("tools"):
            self._tool_call(payload)
        elif payload.get("stream"):
            self._stream(payload)
        else:
            self._json(self._completion(payload))

    def _prompt_tokens(self, payload):
        return sum(len(m.get("content") or "") // 4 + 1 for m in payload.get("messages", []))

    def _completion(self, payload):
        return {
            "id": "cmpl-fake", "object": "chat.completion", "model": MODEL,
            "choices": [{"index": 0, "finish_reason": "stop",
                         "message": {"role": "assistant", "content": "ok"}}],
            "usage": {"prompt_tokens": self._prompt_tokens(payload), "completion_tokens": 4,
                      "total_tokens": self._prompt_tokens(payload) + 4},
        }

    def _tool_call(self, payload):
        tool = payload["tools"][0]["function"]
        arguments = {key: 7 for key in tool.get("parameters", {}).get("required", [])}
        self._json({
            "id": "cmpl-fake", "object": "chat.completion", "model": MODEL,
            "choices": [{"index": 0, "finish_reason": "tool_calls",
                         "message": {"role": "assistant", "content": None,
                                     "tool_calls": [{"id": "call_1", "type": "function",
                                                     "function": {"name": tool["name"],
                                                                  "arguments": json.dumps(arguments)}}]}}],
            "usage": {"prompt_tokens": self._prompt_tokens(payload), "completion_tokens": 8,
                      "total_tokens": self._prompt_tokens(payload) + 8},
        })

    def _stream(self, payload):
        self.send_response(200)
        self.send_header("Content-Type", "text/event-stream")
        self.send_header("Cache-Control", "no-cache")
        self.send_header("Connection", "close")
        self.end_headers()
        # A visible prefill, so time to first token and the 0.25s footprint
        # sampler both have something to measure.
        time.sleep(0.35)
        tokens = ["ok", " then", " more", " text"]
        for index, token in enumerate(tokens):
            frame = {"id": "cmpl-fake", "object": "chat.completion.chunk", "model": MODEL,
                     "choices": [{"index": 0, "delta": {"content": token},
                                  "finish_reason": "stop" if index == len(tokens) - 1 else None}]}
            self.wfile.write(f"data: {json.dumps(frame)}\n\n".encode())
            self.wfile.flush()
            time.sleep(0.05)
        usage = {"id": "cmpl-fake", "object": "chat.completion.chunk", "model": MODEL,
                 "choices": [],
                 "usage": {"prompt_tokens": self._prompt_tokens(payload),
                           "completion_tokens": len(tokens),
                           "total_tokens": self._prompt_tokens(payload) + len(tokens)}}
        self.wfile.write(f"data: {json.dumps(usage)}\n\n".encode())
        self.wfile.write(b"data: [DONE]\n\n")
        self.wfile.flush()

    def log_message(self, *args):
        pass


ThreadingHTTPServer(("127.0.0.1", PORT), Handler).serve_forever()
PY

cat > "$OUT/prompts.json" <<'JSON'
{
  "version": "gate-smoke-1",
  "comment": "A tiny suite for the production-entry gate smoke, not the pinned benchmark suite.",
  "filler_paragraph": "Coolant loop maintenance notes. ",
  "system_prompt": "You are a maintenance assistant.",
  "scenarios": {
    "short_prompt": {"kind": "single", "prompt": "List two checks.", "max_tokens": 32},
    "tool_call": {
      "kind": "tool",
      "prompt": "What is the coolant pressure on vehicle 7? Use the tool.",
      "max_tokens": 32,
      "tools": [{"type": "function", "function": {"name": "read_pressure",
        "description": "Read coolant pressure.",
        "parameters": {"type": "object", "properties": {"vehicle": {"type": "integer"}},
                       "required": ["vehicle"]}}}]
    },
    "stability_soak": {"kind": "soak", "iterations": 3,
      "prompt_template": "Summarise inspection {index}.", "max_tokens": 32}
  }
}
JSON

# Wide bands. Two different Python interpreters are not two runtimes and their
# ratios mean nothing; what this file is for is letting an honestly measured
# pair reach `accepted=true` so the acceptance path is exercised at all. The
# strict copy below exercises the other branch.
cat > "$OUT/thresholds.json" <<'JSON'
{
  "maxTimeToFirstTokenRatio": 3.0,
  "minPrefillThroughputRatio": 0.3,
  "minDecodeThroughputRatio": 0.3,
  "maxPeakFootprintRatio": 3.0,
  "maxPromptTokenSkewRatio": 1.5,
  "paritySuccessScenarios": ["short_prompt", "tool_call", "stability_soak"],
  "scoredScenarios": ["short_prompt"]
}
JSON
cat > "$OUT/thresholds-strict.json" <<'JSON'
{
  "maxTimeToFirstTokenRatio": 1.0000001,
  "minPrefillThroughputRatio": 0.9999999,
  "minDecodeThroughputRatio": 0.9999999,
  "maxPeakFootprintRatio": 1.0000001,
  "maxPromptTokenSkewRatio": 1.5,
  "paritySuccessScenarios": ["short_prompt", "tool_call", "stability_soak"],
  "scoredScenarios": ["short_prompt"]
}
JSON

write_config() {
    local path="$1" mode="$2"
    cat > "$path" <<TOML
[profiles.gate-smoke-baseline]
mode = "local"
executable = "$BASELINE_PYTHON"
argv = [
    "$OUT/fake-runtime.py", "{port}", "$MODEL", "$mode",
    "--host", "{host}",
    "--model", "$MODEL",
    "--prefill-step-size", "2048",
    "--reasoning-effort", "medium",
]

[profiles.gate-smoke-candidate]
mode = "local"
executable = "$CANDIDATE_PYTHON"
argv = [
    "$OUT/fake-runtime.py", "{port}", "$MODEL", "$mode",
    "--host", "{host}",
    "--model", "$MODEL",
    "--prefill-step-size", "2048",
    "--reasoning-effort", "medium",
]
TOML
}
write_config "$OUT/serving.toml" serving
write_config "$OUT/models-only.toml" models-only

run_benchmark() {
    local config="$1" session="$2" log="$3" port="$4"
    rm -rf "$session"
    "$BINARY" benchmark-run \
        --config "$config" --model "$MODEL" --prompts "$OUT/prompts.json" \
        --thresholds "$OUT/thresholds.json" --session "$session" --harness "$HARNESS" \
        --baseline-runtime gate-smoke-baseline --baseline-profile gate-smoke-baseline \
        --candidate-runtime gate-smoke-candidate --candidate-profile gate-smoke-candidate \
        --port "$port" --settle-seconds 2 --startup-timeout 90 --request-timeout 90 \
        > "$log" 2>&1
    printf '%s' "$?"
}

# ============================================================================
# 1. The decision path: one invocation launches, drives, measures and judges.
# ============================================================================

info "control: one invocation spawns both runtimes, measures them and judges"
SESSION="$OUT/session"
STATUS="$(run_benchmark "$OUT/serving.toml" "$SESSION" "$OUT/run.log" "$PORT")"
if [ "$STATUS" = "0" ]; then
    pass "a measured pair inside its thresholds is accepted (exit 0)"
else
    fail "the measured control exited $STATUS, not 0"
    tail -30 "$OUT/run.log"
fi
grep -q '"accepted" : true' "$OUT/run.log" \
    && pass "the accepted decision is reported with its deltas" \
    || fail "the control did not report accepted=true"

BASE_RECORD="$SESSION/records/gate-smoke-baseline.json"
CAND_RECORD="$SESSION/records/gate-smoke-candidate.json"
ATTEST="$SESSION/attest"
if [ ! -s "$BASE_RECORD" ] || [ ! -s "$CAND_RECORD" ]; then
    fail "the control produced no records; nothing below can be checked"
    exit 1
fi

# The measurements have to have come from somewhere. This asserts the link the
# previous revision did not have: every scored scenario carries the completions
# it was measured from, and the observation seals exactly that record.
python3 - "$BASE_RECORD" "$CAND_RECORD" "$ATTEST" <<'PY'
import json, sys
base, cand, attest = sys.argv[1:4]
for path in (base, cand):
    record = json.load(open(path))
    served = 0
    for scenario in record["scenarios"]:
        transcript = scenario.get("transcript")
        assert transcript is not None, f"{path}: {scenario['name']} has no transcript"
        for exchange in transcript["exchanges"]:
            if (exchange["path"] == "/v1/chat/completions" and exchange["status"] == 200
                    and exchange["responseByteCount"] > 0):
                served += 1
    assert served > 0, f"{path}: nothing was ever served"
    seal = json.load(open(f"{attest}/{record['runtime']}.attestation.json"))["transcriptDigest"]
    assert seal, f"{path}: the observation seals no transcript"
print("OK")
PY
[ $? -eq 0 ] \
    && pass "every measurement carries the completions it came from, sealed by the observer" \
    || fail "the control's records are not bound to observed completions"

info "review's own reproduction: two live /v1/models placeholders, driven for real"
STATUS="$(run_benchmark "$OUT/models-only.toml" "$OUT/session-placeholder" \
    "$OUT/placeholder.log" "$((PORT + 1))")"
if [ "$STATUS" = "4" ]; then
    pass "a pass whose runtimes only listed models is inadmissible (exit 4)"
else
    fail "the placeholder pass exited $STATUS, not 4"
    tail -20 "$OUT/placeholder.log"
fi
grep -qF "answered other endpoints and served nothing" "$OUT/placeholder.log" \
    && pass "the refusal names what was missing rather than scoring it" \
    || fail "the placeholder refusal does not say that nothing was served"
grep -q '"accepted" : true' "$OUT/placeholder.log" \
    && fail "the placeholder pass still reported accepted=true" \
    || pass "the placeholder pass produced no decision at all"

# And the half of review's reproduction that no longer has a command to use.
info "a pass that really is missing a required scenario"
# Not an edited record — a real pass, driven by the gate, that never ran
# `tool_call` because it was told to skip it. Both records are sealed and
# honest and the thresholds still require the scenario, so the pair is refused.
# This is the one case that drives the *call site*: a subcommand that passed an
# empty required-scenario list into admission would score this pair instead of
# refusing it, and every other case here would still pass.
rm -rf "$OUT/session-skipped"
"$BINARY" benchmark-run \
    --config "$OUT/serving.toml" --model "$MODEL" --prompts "$OUT/prompts.json" \
    --thresholds "$OUT/thresholds.json" --session "$OUT/session-skipped" \
    --harness "$HARNESS" \
    --baseline-runtime gate-smoke-baseline --baseline-profile gate-smoke-baseline \
    --candidate-runtime gate-smoke-candidate --candidate-profile gate-smoke-candidate \
    --port "$((PORT + 2))" --settle-seconds 2 --startup-timeout 90 --request-timeout 90 \
    --skip tool_call > "$OUT/skipped.log" 2>&1
STATUS=$?
if [ "$STATUS" = "4" ]; then
    pass "a pair missing a required parity scenario is refused (exit 4)"
else
    fail "the skipped-scenario pass exited $STATUS, not 4"
    tail -10 "$OUT/skipped.log"
fi
grep -qF "has no scenario" "$OUT/skipped.log" \
    && pass "the refusal names the scenario neither runtime ran" \
    || fail "the skipped-scenario refusal does not name the missing scenario"

info "the subcommand that attested review's placeholders is gone"
"$BINARY" benchmark-attest open --runtime x --pid $$ --profile p \
    --config "$OUT/serving.toml" --directory "$OUT" > "$OUT/attest.log" 2>&1
STATUS=$?
if [ "$STATUS" = "2" ]; then
    pass "benchmark-attest is not a subcommand any more (exit 2)"
else
    fail "benchmark-attest exited $STATUS; a caller can still direct an observation"
fi

info "there is no argument through which a measurement can be supplied"
for FLAG in --baseline-record --candidate-record --scenarios --ttft; do
    "$BINARY" benchmark-run --config "$OUT/serving.toml" --model "$MODEL" \
        --prompts "$OUT/prompts.json" --thresholds "$OUT/thresholds.json" \
        --session "$OUT/nope" --harness "$HARNESS" \
        --baseline-runtime a --baseline-profile gate-smoke-baseline \
        --candidate-runtime b --candidate-profile gate-smoke-candidate \
        "$FLAG" "$OUT/anything.json" > "$OUT/flag.log" 2>&1
    STATUS=$?
    [ "$STATUS" = "2" ] \
        && pass "benchmark-run refuses $FLAG as a usage error" \
        || fail "benchmark-run accepted $FLAG and exited $STATUS"
done

# ============================================================================
# 2. The replay path: it re-derives a verdict and can never grant one.
# ============================================================================

compare() {
    local baseline="$1" candidate="$2" attest="$3" log="$4"
    local thresholds="${5:-$OUT/thresholds.json}"
    "$BINARY" benchmark-compare \
        --baseline "$baseline" --candidate "$candidate" \
        --thresholds "$thresholds" --attestations "$attest" \
        --output "$OUT/replay-decision.json" > "$log" 2>&1
    printf '%s' "$?"
}

# `expect_refusal <label> <needle> <baseline> <candidate> <attest-dir> <log>`
#
# Refusal means exit 4 — inadmissible. Exit 3 would be "the pair was admitted
# and re-scored", which for a tampered pair is the wrong answer for the
# right-looking reason.
expect_refusal() {
    local label="$1" needle="$2" baseline="$3" candidate="$4" attest="$5" log="$6"
    local status
    status="$(compare "$baseline" "$candidate" "$attest" "$log")"
    if [ "$status" != "4" ]; then
        fail "$label: expected exit 4 (inadmissible), got $status"
        return
    fi
    if ! grep -qF "$needle" "$log"; then
        fail "$label: exit 4 but the refusal never mentions $needle"
        cat "$log"
        return
    fi
    pass "$label"
}

info "replay of the real session re-derives the same verdict and grants nothing"
STATUS="$(compare "$BASE_RECORD" "$CAND_RECORD" "$ATTEST" "$OUT/replay.log")"
if [ "$STATUS" = "3" ]; then
    pass "an admitted replay exits 3 even when it recomputes accepted=true"
else
    fail "the replay exited $STATUS, not 3"
    cat "$OUT/replay.log"
fi
grep -q '"accepted" : true' "$OUT/replay.log" \
    && pass "the replay reports what it recomputed" \
    || fail "the replay did not reproduce the control's verdict"
grep -qF "never an acceptance" "$OUT/replay.log" \
    && pass "the replay says in words that it cannot grant an acceptance" \
    || fail "the replay does not state its own limit"

info "the same real pair, scored against bands it cannot meet"
STATUS="$(compare "$BASE_RECORD" "$CAND_RECORD" "$ATTEST" "$OUT/strict.log" \
    "$OUT/thresholds-strict.json")"
[ "$STATUS" = "3" ] \
    && pass "the thresholds a caller names reach the production entry (exit 3)" \
    || fail "strict thresholds exited $STATUS, not 3"
grep -q '"accepted" : false' "$OUT/strict.log" \
    && pass "a pair outside its bands is reported as not accepted" \
    || fail "strict thresholds did not produce blockers"

# Rewrites one field of a copy of the real record.
edit_record() {
    local source="$1" out="$2" patch="$3"
    python3 - "$source" "$out" "$patch" <<'PY'
import json, sys
source, out, patch = sys.argv[1:4]
record = json.load(open(source))


def merge(target, changes):
    for key, value in changes.items():
        if isinstance(value, dict) and value and isinstance(target.get(key), dict):
            merge(target[key], value)
        else:
            target[key] = value


merge(record, json.loads(patch))
json.dump(record, open(out, "w"), indent=2, sort_keys=True)
PY
}

# Copies the real attestation directory and rewrites one field of one document,
# or of both when `runtime` is `both`.
fork_attestations() {
    local directory="$1" runtime="$2" patch="$3"
    rm -rf "$directory"
    cp -R "$ATTEST" "$directory"
    local targets=("$directory/$runtime.attestation.json")
    if [ "$runtime" = "both" ]; then
        targets=(
            "$directory/gate-smoke-baseline.attestation.json"
            "$directory/gate-smoke-candidate.attestation.json"
        )
    fi
    python3 - "$patch" "${targets[@]}" <<'PY'
import json, sys
patch = json.loads(sys.argv[1])
for path in sys.argv[2:]:
    document = json.load(open(path))
    document.update(patch)
    json.dump(document, open(path, "w"), indent=2, sort_keys=True)
PY
}

# ------------------------------------------- the measurements, attacked --
info "a measurement edited after the pass it was taken in"
# One number, and nothing else. The transcript, the provenance, the pins and the
# observation are all the real ones from the control run; the measured time to
# first token becomes 0.001s. The seal covers the measurements as well as the
# exchanges, so this is refused — and it has to be, or sealing the exchanges
# alone would leave exactly this edit free.
python3 - "$CAND_RECORD" "$OUT/edited-candidate.json" <<'EDIT'
import json, sys
record = json.load(open(sys.argv[1]))
for scenario in record["scenarios"]:
    if scenario["name"] == "short_prompt":
        scenario["timeToFirstTokenSeconds"] = 0.001
json.dump(record, open(sys.argv[2], "w"), indent=2, sort_keys=True)
EDIT
expect_refusal "one edited measurement no longer digests to what was sealed" \
    "are not the ones the gate watched being taken" \
    "$BASE_RECORD" "$OUT/edited-candidate.json" "$ATTEST" "$OUT/edited.log"

info "review's attack in its strongest form: real everything, typed numbers"
# The forger copies what they cannot compute — the real provenance, the real
# pins, the real transcript, beside the real observation this binary wrote — and
# types the four numbers that decide the migration. Every clause the previous
# three revisions had is satisfied by this document. It is refused because the
# numbers are not the ones the observation sealed.
python3 - "$CAND_RECORD" "$OUT/typed-candidate.json" <<'TYPED'
import json, sys
record = json.load(open(sys.argv[1]))
for scenario in record["scenarios"]:
    if scenario["name"] != "short_prompt":
        continue
    scenario["timeToFirstTokenSeconds"] = 0.05
    scenario["prefillTokensPerSecond"] = 100000.0
    scenario["decodeTokensPerSecond"] = 5000.0
    scenario["peakPhysicalFootprintBytes"] = 1000000
json.dump(record, open(sys.argv[2], "w"), indent=2, sort_keys=True)
TYPED
expect_refusal "typed measurements beside a real observation are refused" \
    "are not the ones the gate watched being taken" \
    "$BASE_RECORD" "$OUT/typed-candidate.json" "$ATTEST" "$OUT/typed.log"

info "round 1's attack: two hand-minted records with no provenance at all"
for ROLE in baseline candidate; do
    if [ "$ROLE" = "baseline" ]; then START=100; FINISH=200; else START=300; FINISH=400; fi
    cat > "$OUT/minted-$ROLE.json" <<JSON
{
  "runtime": "gate-smoke-$ROLE",
  "revisions": {"mlx": "0.32.2"},
  "command": ["model-harness", "run", "gate-smoke-$ROLE"],
  "pins": {
    "hostIdentity": "gate-smoke", "modelPath": "$MODEL", "modelDigest": "9f2c1a",
    "quantization": "8bit/group64/affine", "promptSuiteDigest": "aa11bb",
    "contextPolicy": "kv=unbounded;prefill-step=2048;reasoning=medium",
    "maxOutputTokens": 256, "temperature": 0.0, "topP": 1.0, "seed": 1234
  },
  "startedAtUnixSeconds": $START,
  "finishedAtUnixSeconds": $FINISH,
  "peakPhysicalFootprintBytes": 29000000000,
  "scenarios": [
    {"name": "short_prompt", "succeeded": true, "promptTokens": 512, "completionTokens": 64,
     "timeToFirstTokenSeconds": 1.0, "prefillTokensPerSecond": 100.0,
     "decodeTokensPerSecond": 10.0, "wallClockSeconds": 7.0,
     "peakPhysicalFootprintBytes": 29000000000, "processPeakSoFarBytes": 29000000000}
  ],
  "declaredAsymmetries": []
}
JSON
done
expect_refusal "hand-minted records with no provenance are inadmissible" \
    "provenance" \
    "$OUT/minted-baseline.json" "$OUT/minted-candidate.json" "$ATTEST" "$OUT/minted.log"

info "a scenario whose measurements carry no transcript"
edit_record "$CAND_RECORD" "$OUT/notranscript-candidate.json" \
    '{"scenarios": [{"name": "short_prompt", "succeeded": true, "promptTokens": 512,
      "completionTokens": 64, "timeToFirstTokenSeconds": 1.0,
      "prefillTokensPerSecond": 100.0, "decodeTokensPerSecond": 10.0,
      "wallClockSeconds": 1.0, "peakPhysicalFootprintBytes": 29000000000,
      "processPeakSoFarBytes": 29000000000, "transcript": null}]}'
expect_refusal "a measurement with no transcript is refused" \
    "no transcript" \
    "$BASE_RECORD" "$OUT/notranscript-candidate.json" "$ATTEST" "$OUT/notranscript.log"

info "an observation that seals no transcript"
fork_attestations "$OUT/attest-unsealed" gate-smoke-candidate '{"transcriptDigest": null}'
expect_refusal "an observation of a process is not an observation of a measurement" \
    "seals no transcript" \
    "$BASE_RECORD" "$CAND_RECORD" "$OUT/attest-unsealed" "$OUT/unsealed.log"

# ------------------------------------------- the observation, attacked --
info "a comparison with nothing that watched it"
mkdir -p "$OUT/attest-empty"
expect_refusal "a pair with no observation at all is refused" \
    "the gate never observed this pass" \
    "$BASE_RECORD" "$CAND_RECORD" "$OUT/attest-empty" "$OUT/unobserved.log"

info "a record whose pid is not the pid that was watched"
edit_record "$CAND_RECORD" "$OUT/pid-candidate.json" \
    '{"provenance": {"runtimeProcessID": 999999}}'
expect_refusal "a record naming a process the gate never watched is refused" \
    "the record describes a run the gate did not watch" \
    "$BASE_RECORD" "$OUT/pid-candidate.json" "$ATTEST" "$OUT/pid.log"

info "an attestation opened and never closed"
fork_attestations "$OUT/attest-open" gate-smoke-candidate \
    '{"closedAtUnixSeconds": null, "servedModelID": null}'
expect_refusal "an unclosed observation is not a watched pass" \
    "opened and never closed" \
    "$BASE_RECORD" "$CAND_RECORD" "$OUT/attest-open" "$OUT/open.log"

info "an observation of a runtime serving some other model"
fork_attestations "$OUT/attest-model" gate-smoke-candidate \
    '{"servedModelID": "/Users/alexis/src/local-models/something-else"}'
expect_refusal "a runtime the gate saw serving another model is refused" \
    "the record describes a run the gate did not watch" \
    "$BASE_RECORD" "$CAND_RECORD" "$OUT/attest-model" "$OUT/model.log"

info "a pair observed end to end by some other build"
# Both documents together, so they agree with each other and with nothing that
# ran this comparison. That is revision 2's defect as a gate clause: `19c54c…`
# served and `3e5fdcc…` judged.
fork_attestations "$OUT/attest-foreign" both \
    "{\"gateBinaryDigest\": \"$(printf 'b2%.0s' {1..32})\"}"
expect_refusal "a pair nothing in this build observed is refused" \
    "cannot certify that it was the one measured" \
    "$BASE_RECORD" "$CAND_RECORD" "$OUT/attest-foreign" "$OUT/foreign.log"

info "an attestation that exists and cannot be decoded"
rm -rf "$OUT/attest-malformed"
cp -R "$ATTEST" "$OUT/attest-malformed"
printf 'not json at all' > "$OUT/attest-malformed/gate-smoke-candidate.attestation.json"
expect_refusal "a failed read is reported as a read failure, not as absence" \
    "is malformed" \
    "$BASE_RECORD" "$CAND_RECORD" "$OUT/attest-malformed" "$OUT/malformed.log"

info "a replay asked for without naming an attestation directory"
"$BINARY" benchmark-compare --baseline "$BASE_RECORD" --candidate "$CAND_RECORD" \
    --thresholds "$OUT/thresholds.json" > "$OUT/noattest.log" 2>&1
STATUS=$?
[ "$STATUS" = "2" ] \
    && pass "omitting --attestations is a usage error, not a comparison (exit 2)" \
    || fail "omitting --attestations exited $STATUS, not 2"

# ------------------------------------------------- the pins, attacked --
info "a declared contextPolicy its launch does not carry"
edit_record "$CAND_RECORD" "$OUT/policy-candidate.json" \
    "{\"provenance\": {\"launchArgv\": [\"--model\", \"$MODEL\", \"--prefill-step-size\", \"512\", \"--reasoning-effort\", \"medium\"]}}"
expect_refusal "a declared contextPolicy the launch does not derive is refused" \
    "the pin is the caller's claim" \
    "$BASE_RECORD" "$OUT/policy-candidate.json" "$ATTEST" "$OUT/policy.log"

info "a launch that left the prefill chunk to the runtime default"
edit_record "$BASE_RECORD" "$OUT/unpinned-baseline.json" \
    "{\"pins\": {\"contextPolicy\": \"kv=unbounded;prefill-step=unpinned;reasoning=medium\"}, \"provenance\": {\"launchArgv\": [\"--model\", \"$MODEL\", \"--reasoning-effort\", \"medium\"]}}"
edit_record "$CAND_RECORD" "$OUT/unpinned-candidate.json" \
    "{\"pins\": {\"contextPolicy\": \"kv=unbounded;prefill-step=unpinned;reasoning=medium\"}, \"provenance\": {\"launchArgv\": [\"--model\", \"$MODEL\", \"--reasoning-effort\", \"medium\"]}}"
expect_refusal "an unpinned prefill chunk is refused rather than defaulted" \
    "prefill-step=unpinned" \
    "$OUT/unpinned-baseline.json" "$OUT/unpinned-candidate.json" "$ATTEST" \
    "$OUT/unpinned.log"

info "a launch that left the reasoning effort to the chat template's default"
edit_record "$BASE_RECORD" "$OUT/noreason-baseline.json" \
    "{\"pins\": {\"contextPolicy\": \"kv=unbounded;prefill-step=2048;reasoning=unpinned\"}, \"provenance\": {\"launchArgv\": [\"--model\", \"$MODEL\", \"--prefill-step-size\", \"2048\"]}}"
edit_record "$CAND_RECORD" "$OUT/noreason-candidate.json" \
    "{\"pins\": {\"contextPolicy\": \"kv=unbounded;prefill-step=2048;reasoning=unpinned\"}, \"provenance\": {\"launchArgv\": [\"--model\", \"$MODEL\", \"--prefill-step-size\", \"2048\"]}}"
expect_refusal "an unstated reasoning effort is refused rather than defaulted" \
    "reasoning=unpinned" \
    "$OUT/noreason-baseline.json" "$OUT/noreason-candidate.json" "$ATTEST" \
    "$OUT/noreason.log"

info "two runtimes rendering the template at different reasoning efforts"
edit_record "$BASE_RECORD" "$OUT/xhigh-baseline.json" \
    "{\"pins\": {\"contextPolicy\": \"kv=unbounded;prefill-step=2048;reasoning=xhigh\"}, \"provenance\": {\"launchArgv\": [\"--model\", \"$MODEL\", \"--prefill-step-size\", \"2048\", \"--chat-template-args\", \"{\\\"reasoning_effort\\\": \\\"xhigh\\\"}\"]}}"
expect_refusal "two different reasoning policies are not a comparison" \
    "pinned condition \"contextPolicy\" differs" \
    "$OUT/xhigh-baseline.json" "$CAND_RECORD" "$ATTEST" "$OUT/xhigh.log"

info "a record that names no revisions"
edit_record "$CAND_RECORD" "$OUT/norev-candidate.json" '{"revisions": {}}'
expect_refusal "a record with empty revisions is refused" \
    "declares no revisions" \
    "$BASE_RECORD" "$OUT/norev-candidate.json" "$ATTEST" "$OUT/norev.log"

info "two runs configured by different launcher documents"
edit_record "$CAND_RECORD" "$OUT/cfg-candidate.json" \
    "{\"provenance\": {\"configDigest\": \"$(printf '99%.0s' {1..32})\"}}"
expect_refusal "a config digest mismatch is refused" \
    "different launcher configurations" \
    "$BASE_RECORD" "$OUT/cfg-candidate.json" "$ATTEST" "$OUT/cfg.log"

info "a launch argv that never mentions the pinned model"
edit_record "$CAND_RECORD" "$OUT/nomodel-candidate.json" \
    '{"provenance": {"launchArgv": ["--model", "/Users/alexis/src/local-models/other", "--prefill-step-size", "2048", "--reasoning-effort", "medium"]}}'
expect_refusal "a launch that never received the pinned model is refused" \
    "never mentions it" \
    "$BASE_RECORD" "$OUT/nomodel-candidate.json" "$ATTEST" "$OUT/nomodel.log"

info "a digest that is not a digest"
edit_record "$CAND_RECORD" "$OUT/digest-candidate.json" \
    '{"provenance": {"driverDigest": "not-a-digest"}}'
expect_refusal "a malformed driver digest is refused" \
    "is not a SHA-256 digest" \
    "$BASE_RECORD" "$OUT/digest-candidate.json" "$ATTEST" "$OUT/digest.log"

info "a required parity scenario missing from a record"
# Drives the *call site*: `benchmark-compare` passes the threshold document's
# parity and scored scenario names into `admit`. A subcommand that passed an
# empty list would still call the gate and still refuse nothing here.
edit_record "$CAND_RECORD" "$OUT/noscenario-candidate.json" '{"scenarios": []}'
expect_refusal "a record missing every scenario is refused" \
    "no scenario that ever completed" \
    "$BASE_RECORD" "$OUT/noscenario-candidate.json" "$ATTEST" "$OUT/noscenario.log"

printf '\n%s\n' "----------------------------------------"
if [ "$FAILURES" -eq 0 ]; then
    printf 'BENCHMARK GATE SMOKE OK (0 failures)\n'
    exit 0
fi
printf 'BENCHMARK GATE SMOKE FAILED (%d failures)\n' "$FAILURES"
exit 1
