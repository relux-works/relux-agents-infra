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
# The one equivalence decision this repository took, as versioned source beside
# the package. `TrustedEquivalenceDecisions.shipped` carries its SHA-256, so
# this is the only document the gate will accept as evidence.
PACKAGE_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TRUSTED_VERDICT="$PACKAGE_ROOT/equivalence/qwen3-8-27b-uncensored.equivalence.json"
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
import argparse, json, mmap, os, sys, time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

PORT = int(sys.argv[1])
MODEL = sys.argv[2]
MODE = sys.argv[3] if len(sys.argv) > 3 else "serving"
# `llama-server` answers `/v1/models` with a `meta` block carrying `n_ctx`;
# the corrected bounded Python baseline emits the effective cache bound after
# cache construction. Measured on build b10621-c1d0e7a00:
# n_ctx is 8192 under `--ctx-size 8192` and 32768 -- the model's n_ctx_train --
# with no context flag. Absent here, this stand-in is MLX-shaped and reports
# nothing.
N_CTX = None
if "--n-ctx" in sys.argv:
    N_CTX = int(sys.argv[sys.argv.index("--n-ctx") + 1])
elif "--max-kv-size" in sys.argv:
    N_CTX = int(sys.argv[sys.argv.index("--max-kv-size") + 1])
# F2. A runtime with a real, finite window that answers `meta.n_ctx` as
# something the gate cannot read a bound out of. Review used the JSON *string*
# "32768" against a genuinely unbounded baseline and got exit 0 with
# accepted=true, because a malformed field collapsed into "the runtime named no
# bound" and that is the reading the argv fallback is allowed to speak to.
N_CTX_STRING = None
if "--n-ctx-string" in sys.argv:
    N_CTX_STRING = sys.argv[sys.argv.index("--n-ctx-string") + 1]
# `GET /slots` is llama-server's, and it is the only endpoint on build
# b10621-c1d0e7a00 whose answer moves with the launch's speculative
# configuration: measured, `--spec-type ngram-mod` flips `params.speculative`
# to true there while `/props` still reports "none". Neither MLX runtime serves
# the route at all, so without this flag the stand-in 404s it and is read as a
# runtime that answered and named nothing.
SLOTS = None
if "--slots" in sys.argv:
    SLOTS = sys.argv[sys.argv.index("--slots") + 1] == "true"
# F3. The route is there and the answer is a failure. Review configured a
# fixture as speculative, made `/slots` answer HTTP 500, and the pass was scored
# as MTP-off: a failed observation spent as a negative one.
SLOTS_STATUS = None
if "--slots-status" in sys.argv:
    SLOTS_STATUS = int(sys.argv[sys.argv.index("--slots-status") + 1])
# The two field spellings this task has to put on one clock boundary. The
# default remains ordinary content so every pre-existing smoke case keeps its
# response shape.
REASONING_FIELD = None
if "--reasoning-field" in sys.argv:
    REASONING_FIELD = sys.argv[sys.argv.index("--reasoning-field") + 1]
    assert REASONING_FIELD in ("reasoning", "reasoning_content")
CACHE_REUSE_HIT = "--cache-reuse-hit" in sys.argv

# Exercise each runtime shape's parser semantics and defaults rather than making
# the two stand-ins identical by accident. Candidate is a role, not a runtime:
# most controls emulate MLX Swift, while the context/speculation/mmap cases use
# llama.cpp-specific flags. Python argparse permits a unique option abbreviation
# and the last repeated value wins. llama.cpp accepts -ub/--ubatch-size and
# defaults that physical chunk to 512, while both MLX servers spell it
# --prefill-step-size and the Python baseline defaults to 2048. A shared
# hand-rolled first-index lookup would turn the no-flag, abbreviation, and
# repeated-value attacks below into fixture lies.
_generation_parser = argparse.ArgumentParser(add_help=False, allow_abbrev=True)
_llama_generation_flags = {
    "--ctx-size", "-c", "--ubatch-size", "-ub", "--n-ctx", "--slots",
    "--slots-status", "--spec-type", "--mmap-artifact",
}
if os.path.basename(sys.argv[0]) != "mlx_lm.server" and any(
    flag in sys.argv for flag in _llama_generation_flags
):
    _generation_parser.add_argument("-ub", "--ubatch-size", type=int, default=512)
else:
    _generation_parser.add_argument("--prefill-step-size", type=int, default=2048)
_generation_args, _ = _generation_parser.parse_known_args(sys.argv[4:])
PREFILL_STEP = (
    _generation_args.prefill_step_size
    if hasattr(_generation_args, "prefill_step_size")
    else _generation_args.ubatch_size
)
REASONING = None
if "--reasoning-effort" in sys.argv:
    REASONING = sys.argv[sys.argv.index("--reasoning-effort") + 1]
if "--chat-template-args" in sys.argv:
    REASONING = json.loads(sys.argv[sys.argv.index("--chat-template-args") + 1]).get(
        "reasoning_effort")

# Keep a real file mapping resident for the life of the process. This is the
# component `ri_phys_footprint` misses and the production memory sampler must
# carry beside it. Touch every page so a mere virtual mapping cannot satisfy the
# assertion on resident mapped-file bytes below.
MAPPED_FILE = None
MAPPED_BYTES = None
if "--mmap-artifact" in sys.argv:
    path = sys.argv[sys.argv.index("--mmap-artifact") + 1]
    MAPPED_FILE = open(path, "rb")
    MAPPED_BYTES = mmap.mmap(MAPPED_FILE.fileno(), 0, access=mmap.ACCESS_READ)
    _mapped_checksum = 0
    for offset in range(0, len(MAPPED_BYTES), os.sysconf("SC_PAGE_SIZE")):
        _mapped_checksum ^= MAPPED_BYTES[offset]


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
            entry = {"id": MODEL, "object": "model"}
            meta = {}
            if MODE != "serving-no-meta" and N_CTX is not None:
                meta["n_ctx"] = N_CTX
                meta["n_ctx_train"] = 32768
            elif MODE != "serving-no-meta" and N_CTX_STRING is not None:
                meta["n_ctx"] = N_CTX_STRING
                meta["n_ctx_train"] = 32768
            elif MODE != "serving-no-meta":
                meta["n_ctx"] = 76800
            if MODE != "serving-no-config":
                runtime_config = {"prefill_step_size": PREFILL_STEP}
                if REASONING is not None:
                    runtime_config["reasoning_effort"] = REASONING
                meta["runtime_config"] = runtime_config
            if meta:
                entry["meta"] = meta
            self._json({"data": [entry]})
            return
        if self.path.rstrip("/").endswith("/slots"):
            if SLOTS_STATUS is not None:
                self._json({"error": "slot reporting failed"}, status=SLOTS_STATUS)
                return
            if SLOTS is None:
                self._json({"error": "this build serves no slots"}, status=404)
                return
            self._json([{"id": 0, "params": {"speculative": SLOTS}}])
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

    def _usage(self, payload, completion_tokens):
        prompt_tokens = self._prompt_tokens(payload)
        cached_tokens = prompt_tokens if CACHE_REUSE_HIT else 0
        return {"prompt_tokens": prompt_tokens, "completion_tokens": completion_tokens,
                "total_tokens": prompt_tokens + completion_tokens,
                "prompt_tokens_details": {"cached_tokens": cached_tokens}}

    def _completion(self, payload):
        return {
            "id": "cmpl-fake", "object": "chat.completion", "model": MODEL,
            "choices": [{"index": 0, "finish_reason": "stop",
                         "message": {"role": "assistant", "content": "ok"}}],
            "usage": self._usage(payload, 4),
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
            "usage": self._usage(payload, 8),
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
            # Three reasoning events and one content event. A reader that knows
            # only the other runtime's spelling starts on the final event,
            # leaving no first-to-last decode interval and making the real gate
            # refuse the run below.
            field = REASONING_FIELD if REASONING_FIELD and index < 3 else "content"
            frame = {"id": "cmpl-fake", "object": "chat.completion.chunk", "model": MODEL,
                     "choices": [{"index": 0, "delta": {field: token},
                                  "finish_reason": "stop" if index == len(tokens) - 1 else None}]}
            self.wfile.write(f"data: {json.dumps(frame)}\n\n".encode())
            self.wfile.flush()
            time.sleep(0.05)
        usage = {"id": "cmpl-fake", "object": "chat.completion.chunk", "model": MODEL,
                 "choices": [],
                 "usage": self._usage(payload, len(tokens))}
        self.wfile.write(f"data: {json.dumps(usage)}\n\n".encode())
        self.wfile.write(b"data: [DONE]\n\n")
        self.wfile.flush()

    def log_message(self, *args):
        pass


ThreadingHTTPServer(("127.0.0.1", PORT), Handler).serve_forever()
PY

# --------------------------------------------- the G4 fixture writers --
#
# Written out as files rather than inlined as heredocs because section 4 has to
# recompute the gate's own model digest, and a digest a shell script hardcodes
# is a digest that stops describing the fixture the first time the fixture
# changes.

cat > "$OUT/mlx-digest.py" <<'PY'
"""The gate's weight-directory digest, recomputed independently.

`BenchmarkRunCommand.modelDigest(directory:)` accumulates each file's name
followed by its bytes and SHA-256s the result. Written out here so the verdict
this script authors names the number the gate will compute, rather than a number
this script asserted and the gate then had to agree with.
"""
import hashlib, os, sys

accumulated = bytearray()
for name in ("config.json", "model.safetensors.index.json"):
    accumulated += name.encode()
    accumulated += open(os.path.join(sys.argv[1], name), "rb").read()
print(hashlib.sha256(bytes(accumulated)).hexdigest())
PY

cat > "$OUT/write-verdict.py" <<'PY'
"""A verdict the CALLER authored, in whichever shape the case under test needs.

Nothing this file writes is evidence. F1 was exactly this: `--equivalence` named
a JSON path, the gate read it, digested it, carried the digest onto both
attestations and sealed it, and review minted a document naming an arbitrary
source of record, the two artifact digests the gate had itself computed,
`comparable` and one generic note -- and got exit 0 with accepted=true. Every
document written here is well-shaped on purpose, so the refusals below are about
who decided rather than about the JSON being wrong.
"""
import json, sys

(path, verdict, covers_gguf, declares_notes, source, mlx, mlx_digest, gguf,
 gguf_digest, mtp, vision, norm) = sys.argv[1:13]
artifacts = [{"path": mlx, "digest": mlx_digest, "quantization": "8bit/group64/affine"}]
if covers_gguf == "true":
    artifacts.append({"path": gguf, "digest": gguf_digest, "quantization": "Q8_0"})
else:
    # A comparable verdict about some other file entirely, which the gate must
    # refuse to accept as describing the artifact under test.
    artifacts.append(
        {"path": "/some/other-model.gguf", "digest": "0" * 64, "quantization": "Q8_0"})
json.dump(
    {
        "sourceOfRecord": source,
        "verdict": verdict,
        "artifacts": artifacts,
        "declaredNonEquivalences": [mtp, vision, norm] if declares_notes == "true" else [],
    },
    open(path, "w"),
    indent=2)
PY

cat > "$OUT/assert-g4.py" <<'PY'
"""What the accepted cross-format pass must have pinned.

The exit code above says the gate admitted the pair; this says it admitted it on
the right grounds. A gate that kept comparing local artifacts could not have
reached an acceptance at all, but a gate that dropped the pin rather than
replacing it would also exit 0 -- and would write something else here.
"""
import json, os, sys

session, mlx_digest, gguf_digest, source, mtp = sys.argv[1:6]
records = os.path.join(session, "records")
baseline = json.load(open(os.path.join(records, "gate-smoke-baseline.json")))
candidate = json.load(open(os.path.join(records, "gate-smoke-candidate.json")))

for record in (baseline, candidate):
    assert record["pins"]["modelOfRecord"] == f"source:{source}", record["pins"]
    assert record["pins"]["speculation"] == "off", record["pins"]
    # The differences the verdict did not dissolve travel with the record, so no
    # report of this decision can be written without stating them.
    assert mtp in record["declaredAsymmetries"], record["declaredAsymmetries"]
    assert any(
        "direction is against llama.cpp" in note and "MTP/speculative decoding" in note
        for note in record["declaredAsymmetries"]
    ), record["declaredAsymmetries"]

decision = json.load(open(os.path.join(session, "decision.json")))
assert any(
    "direction is against llama.cpp" in note and "MTP/speculative decoding" in note
    for note in decision["declaredAsymmetries"]
), decision["declaredAsymmetries"]

# The two artifacts really are different, which is what makes the shared source
# of record the only thing they could have been compared on.
assert baseline["pins"]["modelDigest"] == mlx_digest, baseline["pins"]["modelDigest"]
assert candidate["pins"]["modelDigest"] == gguf_digest, candidate["pins"]["modelDigest"]
assert baseline["pins"]["modelDigest"] != candidate["pins"]["modelDigest"]
assert baseline["pins"]["quantization"] == "8bit/group64/affine", baseline["pins"]
assert candidate["pins"]["quantization"] == "Q8_0", candidate["pins"]

# And the verdict is on the document the gate wrote, not on the one the record
# did, which is what stops a record minting its own model identity.
attest = os.path.join(session, "attest")
for runtime in ("gate-smoke-baseline", "gate-smoke-candidate"):
    document = json.load(open(os.path.join(attest, f"{runtime}.attestation.json")))
    assert document["observedModelEquivalence"]["state"] == "read", document
print("OK")
PY

# Re-exec the serving process with a different prefill flag than the profile
# supplied. This distinguishes kernel-observed argv from caller configuration.
cat > "$OUT/argv-rewriter.py" <<PY
import os, sys
port, model, mode = sys.argv[1:4]
os.execv(sys.executable, [
    sys.executable, "$OUT/fake-runtime.py", port, model, mode,
    "--host", "127.0.0.1", "--model", model,
    "--max-kv-size", "76800", "--prefill-step-size", "999",
    "--reasoning-effort", "medium",
])
PY

# A minimal immutable Python distribution for the baseline control. It is not
# a shortcut around provenance: the production gate verifies this distribution
# and its console script against RECORD through the interpreter behind the live
# baseline process. The decoy negative below deliberately bypasses this wrapper.
BASELINE_VENV="$OUT/baseline-venv"
"$BASELINE_PYTHON" -m venv --without-pip "$BASELINE_VENV"
SITE="$($BASELINE_VENV/bin/python -c 'import sysconfig; print(sysconfig.get_path("purelib"))')"
mkdir -p "$SITE/mlx_lm" "$SITE/mlx_lm-0.0.1.dist-info"
cat > "$SITE/mlx_lm/__init__.py" <<'PY'
PY
cat > "$SITE/mlx_lm/server.py" <<PY
def main():
    path = "$OUT/fake-runtime.py"
    namespace = {"__name__": "__main__", "__file__": path}
    with open(path, "rb") as handle:
        code = compile(handle.read(), path, "exec")
    exec(code, namespace)
PY
cat > "$BASELINE_VENV/bin/mlx_lm.server" <<PY
#!$BASELINE_VENV/bin/python
import sys
from mlx_lm.server import main
if __name__ == '__main__':
    sys.exit(main())
PY
chmod +x "$BASELINE_VENV/bin/mlx_lm.server"
cat > "$SITE/mlx_lm-0.0.1.dist-info/METADATA" <<'META'
Metadata-Version: 2.1
Name: mlx-lm
Version: 0.0.1
META
cat > "$SITE/mlx_lm-0.0.1.dist-info/entry_points.txt" <<'ENTRY'
[console_scripts]
mlx_lm.server = mlx_lm.server:main
ENTRY
cat > "$SITE/mlx_lm-0.0.1.dist-info/direct_url.json" <<'JSON'
{"url":"file:///benchmark-gate-smoke/mlx-lm","vcs_info":{"vcs":"git","commit_id":"1111111111111111111111111111111111111111","requested_revision":"1111111111111111111111111111111111111111"}}
JSON
for DIST in mlx mlx_metal transformers; do
    mkdir -p "$SITE/${DIST}-0.0.1.dist-info"
    NORMALIZED="${DIST//_/-}"
    printf 'Metadata-Version: 2.1\nName: %s\nVersion: 0.0.1\n' "$NORMALIZED" \
        > "$SITE/${DIST}-0.0.1.dist-info/METADATA"
done
"$BASELINE_VENV/bin/python" - "$SITE" "$BASELINE_VENV/bin/mlx_lm.server" <<'PY'
import base64, csv, hashlib, os, pathlib, sys
site, wrapper = map(pathlib.Path, sys.argv[1:])
dist = site / "mlx_lm-0.0.1.dist-info"
paths = [site / "mlx_lm/__init__.py", site / "mlx_lm/server.py",
         dist / "METADATA", dist / "entry_points.txt", dist / "direct_url.json", wrapper]
rows = []
for path in paths:
    data = path.read_bytes()
    digest = base64.urlsafe_b64encode(hashlib.sha256(data).digest()).decode().rstrip("=")
    rows.append((os.path.relpath(path, site), f"sha256={digest}", str(len(data))))
rows.append((os.path.relpath(dist / "RECORD", site), "", ""))
with (dist / "RECORD").open("w", newline="") as handle:
    csv.writer(handle).writerows(rows)
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
#
# The control below requires exit 0 rather than tolerating exit 3. In revision 3
# every admitted-path check in this file exited 3 while the suite reported 118
# PASS / 0 FAIL, because the memory dimension could not be scored at all and the
# tolerance absorbed a total loss of it. A suite where the acceptance path is
# never taken cannot report that the acceptance path broke.
cat > "$OUT/thresholds.json" <<'JSON'
{
  "maxTimeToFirstTokenRatio": 3.0,
  "minPrefillThroughputRatio": 0.1,
  "minDecodeThroughputRatio": 0.1,
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
executable = "$BASELINE_VENV/bin/mlx_lm.server"
argv = [
    "{port}", "$MODEL", "$mode",
    "--host", "{host}",
    "--model", "$MODEL",
    "--max-kv-size", "76800",
    "--prefill-step-size", "2048",
    "--chat-template-args", "{\"reasoning_effort\":\"medium\"}",
]

[profiles.gate-smoke-candidate]
mode = "local"
executable = "$CANDIDATE_PYTHON"
argv = [
    "$OUT/fake-runtime.py", "{port}", "$MODEL", "$mode",
    "--host", "{host}",
    "--model", "$MODEL",
    "--max-kv-size", "76800",
    "--prefill-step-size", "2048",
    "--reasoning-effort", "medium",
]
TOML
}
write_config "$OUT/serving.toml" serving
write_config "$OUT/models-only.toml" models-only
write_config "$OUT/no-meta.toml" serving-no-meta
write_config "$OUT/no-runtime-config.toml" serving-no-config

cat > "$OUT/cache-one-sided.toml" <<TOML
[profiles.gate-smoke-baseline]
mode = "local"
executable = "$BASELINE_VENV/bin/mlx_lm.server"
argv = [
    "{port}", "$MODEL", "serving", "--host", "{host}", "--model", "$MODEL",
    "--max-kv-size", "76800", "--prefill-step-size", "2048",
    "--chat-template-args", "{\"reasoning_effort\":\"medium\"}",
]
[profiles.gate-smoke-candidate]
mode = "local"
executable = "$CANDIDATE_PYTHON"
argv = [
    "$OUT/fake-runtime.py", "{port}", "$MODEL", "serving",
    "--host", "{host}", "--model", "$MODEL", "--cache-reuse-hit",
    "--max-kv-size", "76800", "--prefill-step-size", "2048",
    "--reasoning-effort", "medium",
]
TOML

cat > "$OUT/rewritten-argv.toml" <<TOML
[profiles.gate-smoke-baseline]
mode = "local"
executable = "$BASELINE_VENV/bin/mlx_lm.server"
argv = [
    "{port}", "$MODEL", "serving", "--host", "{host}", "--model", "$MODEL",
    "--max-kv-size", "76800", "--prefill-step-size", "2048",
    "--chat-template-args", "{\"reasoning_effort\":\"medium\"}",
]
[profiles.gate-smoke-candidate]
mode = "local"
executable = "$CANDIDATE_PYTHON"
argv = [
    "$OUT/argv-rewriter.py", "{port}", "$MODEL", "serving",
    "--host", "{host}", "--model", "$MODEL",
    "--max-kv-size", "76800", "--prefill-step-size", "2048",
    "--reasoning-effort", "medium",
]
TOML

cat > "$OUT/duplicate-python-prefill.toml" <<TOML
[profiles.gate-smoke-baseline]
mode = "local"
executable = "$BASELINE_VENV/bin/mlx_lm.server"
argv = [
    "{port}", "$MODEL", "serving", "--host", "{host}", "--model", "$MODEL",
    "--max-kv-size", "76800",
    "--prefill-step-size", "2048", "--prefill-step-size", "999",
    "--chat-template-args", "{\"reasoning_effort\":\"medium\"}",
]
[profiles.gate-smoke-candidate]
mode = "local"
executable = "$CANDIDATE_PYTHON"
argv = [
    "$OUT/fake-runtime.py", "{port}", "$MODEL", "serving",
    "--host", "{host}", "--model", "$MODEL",
    "--max-kv-size", "76800", "--prefill-step-size", "2048",
    "--reasoning-effort", "medium",
]
TOML

cat > "$OUT/abbreviated-python-prefill.toml" <<TOML
[profiles.gate-smoke-baseline]
mode = "local"
executable = "$BASELINE_VENV/bin/mlx_lm.server"
argv = [
    "{port}", "$MODEL", "serving", "--host", "{host}", "--model", "$MODEL",
    "--max-kv-size", "76800",
    "--prefill-step-size", "2048", "--prefill-step-siz", "999",
    "--chat-template-args", "{\"reasoning_effort\":\"medium\"}",
]
[profiles.gate-smoke-candidate]
mode = "local"
executable = "$CANDIDATE_PYTHON"
argv = [
    "$OUT/fake-runtime.py", "{port}", "$MODEL", "serving",
    "--host", "{host}", "--model", "$MODEL",
    "--max-kv-size", "76800", "--prefill-step-size", "2048",
    "--reasoning-effort", "medium",
]
TOML

run_benchmark() {
    local config="$1" session="$2" log="$3" port="$4"
    rm -rf "$session"
    "$BINARY" benchmark-run \
        --config "$config" --model "$MODEL" --prompts "$OUT/prompts.json" \
        --thresholds "$OUT/thresholds.json" --session "$session" --harness "$HARNESS" \
        --baseline-runtime python-mlx-lm --baseline-profile gate-smoke-baseline \
        --candidate-runtime mlx-swift --candidate-profile gate-smoke-candidate \
        --python-bin "$BASELINE_VENV/bin/python" \
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
if [ "$STATUS" = "0" ] && [ -s "$SESSION/decision.json" ]; then
    pass "a measured pair reaches an accepted decision (exit 0)"
else
    fail "the measured control exited $STATUS without an accepted decision"
    tail -30 "$OUT/run.log"
fi
# This is the suite's one unconditional acceptance-path check, and it is the
# check that fails when the resident-memory dimension cannot be scored for a
# realistic scenario window. `accepted` is `blockers.isEmpty`, so an unmeasured
# memory delta lands here as a blocker and the pass turns red rather than
# quietly downgrading to "refused correctly".
CONTROL_STATUS="$STATUS" python3 - "$SESSION/decision.json" <<'CONTROLPY'
import json, os, sys
decision = json.load(open(sys.argv[1]))
status = int(os.environ["CONTROL_STATUS"])
assert status == 0, (status, decision)
assert decision["accepted"] is True, decision
assert not decision["blockers"], decision
memory = [delta for delta in decision["deltas"]
          if delta["metric"] == "peak_resident_memory_upper_bound_bytes"]
assert memory, decision["deltas"]
assert any(delta["scenario"] == "process" for delta in memory), memory
assert any(delta["scenario"] != "process" for delta in memory), memory
for delta in memory:
    assert delta["verdict"] not in ("unmeasured", "non-comparable"), delta
    assert delta["baseline"] > 0, delta
    assert delta["candidate"] > 0, delta
ttft = next(delta for delta in decision["deltas"]
            if delta["scenario"] == "short_prompt"
            and delta["metric"] == "time_to_first_token_seconds")
assert ttft["verdict"] != "non-comparable", ttft
print("OK")
CONTROLPY
CONTROL_DECISION_STATUS=$?
if [ "$CONTROL_DECISION_STATUS" -eq 0 ]; then
    pass "the accepted control scores resident memory on every delta it reports"
else
    fail "the control was accepted without a scored resident-memory dimension"
fi

info "F2: identical prompts with a one-sided sealed cache hit are outside evidence"
CACHE_SESSION="$OUT/session-cache-one-sided"
STATUS="$(run_benchmark "$OUT/cache-one-sided.toml" "$CACHE_SESSION" \
    "$OUT/cache-one-sided.log" "$((PORT + 27))")"
if [ "$STATUS" = "3" ] && [ -s "$CACHE_SESSION/decision.json" ]; then
    pass "one-sided cache reuse reaches a rejected decision (exit 3)"
else
    fail "one-sided cache reuse exited $STATUS without a rejected decision"
    tail -30 "$OUT/cache-one-sided.log"
fi
python3 - "$CACHE_SESSION" <<'CACHEPY'
import json, pathlib, sys
session = pathlib.Path(sys.argv[1])
baseline = json.load(open(session / "records/python-mlx-lm.json"))
candidate = json.load(open(session / "records/mlx-swift.json"))
decision = json.load(open(session / "decision.json"))
baseline_short = next(s for s in baseline["scenarios"] if s["name"] == "short_prompt")
candidate_short = next(s for s in candidate["scenarios"] if s["name"] == "short_prompt")
assert baseline_short["promptTokens"] == candidate_short["promptTokens"], (
    baseline_short, candidate_short)
assert baseline_short["cacheReuse"]["state"] == "miss", baseline_short["cacheReuse"]
assert candidate_short["cacheReuse"]["state"] == "hit", candidate_short["cacheReuse"]
assert any("short_prompt/cache_reuse is one-sided" in b for b in decision["blockers"]), decision
for metric in ("time_to_first_token_seconds", "prefill_tokens_per_second",
               "decode_tokens_per_second"):
    delta = next(d for d in decision["deltas"]
                 if d["scenario"] == "short_prompt" and d["metric"] == metric)
    assert delta["verdict"] == "non-comparable", delta
print("OK")
CACHEPY
CACHE_STATUS=$?
if [ "$CACHE_STATUS" -eq 0 ]; then
    pass "production decision consumes the sealed cache fact and keeps the no-hit control scoreable"
else
    fail "the production cache-reuse gate did not distinguish one-sided and symmetric facts"
fi

# Exact revision-3 reviewer attack: both live model listings answer, but omit
# meta.n_ctx while both launch profiles still request --max-kv-size 76800.
# Requested configuration is not observed enforcement, so this must never
# produce a decision.
STATUS="$(run_benchmark "$OUT/no-meta.toml" "$OUT/session-no-meta" \
    "$OUT/no-meta.log" "$((PORT + 21))")"
if [ "$STATUS" = "4" ]; then
    pass "an omitted live KV bound stays inadmissible despite a finite launch flag (exit 4)"
else
    fail "the omitted live KV bound exited $STATUS, not 4"
    tail -20 "$OUT/no-meta.log"
fi
grep -qF "kv=not-reported" "$OUT/no-meta.log" \
    && pass "the production refusal names absent live KV evidence" \
    || fail "the omitted-bound refusal did not name the absent observation"
grep -q '"accepted" : true' "$OUT/no-meta.log" \
    && fail "the omitted live KV bound still reported accepted=true" \
    || pass "the omitted live KV bound cannot mint an accepted record"

STATUS="$(run_benchmark "$OUT/no-runtime-config.toml" "$OUT/session-no-runtime-config" \
    "$OUT/no-runtime-config.log" "$((PORT + 24))")"
if [ "$STATUS" = "4" ]; then
    pass "an omitted live generation configuration is inadmissible (exit 4)"
else
    fail "the omitted live generation configuration exited $STATUS, not 4"
    tail -20 "$OUT/no-runtime-config.log"
fi
grep -qF 'prefill-step=not-reported' "$OUT/no-runtime-config.log" \
    && pass "the production refusal names the first absent live generation parameter" \
    || fail "the omitted-generation-config refusal did not name absent live prefill evidence"

rm -rf "$OUT/session-rewritten-argv"
"$BINARY" benchmark-run \
    --config "$OUT/rewritten-argv.toml" --model "$MODEL" --prompts "$OUT/prompts.json" \
    --thresholds "$OUT/thresholds.json" --session "$OUT/session-rewritten-argv" \
    --harness "$HARNESS" \
    --baseline-runtime python-mlx-lm --baseline-profile gate-smoke-baseline \
    --candidate-runtime mlx-swift --candidate-profile gate-smoke-candidate \
    --python-bin "$BASELINE_VENV/bin/python" --port "$((PORT + 22))" \
    --settle-seconds 0 --startup-timeout 90 --request-timeout 90 \
    > "$OUT/rewritten-argv.log" 2>&1
STATUS=$?
if [ "$STATUS" = "4" ]; then
    pass "kernel argv overrides a caller profile rewritten before serving (exit 4)"
else
    fail "the rewritten argv attack exited $STATUS, not 4"
    tail -20 "$OUT/rewritten-argv.log"
fi
grep -qF 'prefill-step=2048;reasoning=medium" vs candidate "kv=76800;prefill-step=999' \
    "$OUT/rewritten-argv.log" \
    && pass "the production record derives prefill from the serving process report" \
    || fail "the rewritten argv refusal did not expose the live-reported prefill value"
grep -q '"accepted" : true' "$OUT/rewritten-argv.log" \
    && fail "caller profile argv still minted an accepted record" \
    || pass "caller profile argv cannot substitute for observed process argv"

# Exact revision-4 reviewer shape, now applied to the actual argparse-backed
# baseline: its observed argv repeats the pinned flag and argparse uses 999.
# The running process must report that effective last value rather than the
# gate trying to decode it independently.
STATUS="$(run_benchmark "$OUT/duplicate-python-prefill.toml" \
    "$OUT/session-duplicate-python-prefill" "$OUT/duplicate-python-prefill.log" \
    "$((PORT + 23))")"
if [ "$STATUS" = "4" ]; then
    pass "repeated mlx_lm prefill flags are attested by the process (exit 4)"
else
    fail "the repeated mlx_lm prefill attack exited $STATUS, not 4"
    tail -20 "$OUT/duplicate-python-prefill.log"
fi
grep -qF 'baseline "kv=76800;prefill-step=999;reasoning=medium" vs candidate "kv=76800;prefill-step=2048' \
    "$OUT/duplicate-python-prefill.log" \
    && pass "the production record pins the process-reported repeated value" \
    || fail "the repeated prefill refusal did not expose the process-reported value"
grep -q '"accepted" : true' "$OUT/duplicate-python-prefill.log" \
    && fail "the repeated prefill attack still reported accepted=true" \
    || pass "a non-effective repeated value cannot be attested"

# Exact revision-5 reviewer attack. Python argparse accepts the unique
# abbreviation and resolves 999; the gate must learn that from /v1/models,
# never from an incomplete imitation of argparse.
STATUS="$(run_benchmark "$OUT/abbreviated-python-prefill.toml" \
    "$OUT/session-abbreviated-python-prefill" "$OUT/abbreviated-python-prefill.log" \
    "$((PORT + 25))")"
if [ "$STATUS" = "4" ]; then
    pass "argparse abbreviation is attested by the running baseline (exit 4)"
else
    fail "the argparse abbreviation attack exited $STATUS, not 4"
    tail -20 "$OUT/abbreviated-python-prefill.log"
fi
grep -qF 'baseline "kv=76800;prefill-step=999;reasoning=medium" vs candidate "kv=76800;prefill-step=2048' \
    "$OUT/abbreviated-python-prefill.log" \
    && pass "the production record carries argparse's live effective abbreviation" \
    || fail "the abbreviation refusal did not expose the server-reported value"
grep -q '"accepted" : true' "$OUT/abbreviated-python-prefill.log" \
    && fail "the argparse abbreviation attack still reported accepted=true" \
    || pass "argv abbreviation cannot mint a false effective prefill value"

# Exact revision-2 reviewer attack: the baseline launches a decoy program under
# one interpreter while --python-bin points at the immutable distribution above.
cat > "$OUT/decoy.toml" <<TOML
[profiles.gate-smoke-baseline]
mode = "local"
executable = "$BASELINE_PYTHON"
argv = [
    "$OUT/fake-runtime.py", "{port}", "$MODEL", "serving",
    "--host", "{host}", "--model", "$MODEL",
    "--max-kv-size", "76800", "--prefill-step-size", "2048",
    "--chat-template-args", "{\"reasoning_effort\":\"medium\"}",
]
[profiles.gate-smoke-candidate]
mode = "local"
executable = "$CANDIDATE_PYTHON"
argv = [
    "$OUT/fake-runtime.py", "{port}", "$MODEL", "serving",
    "--host", "{host}", "--model", "$MODEL",
    "--max-kv-size", "76800", "--prefill-step-size", "2048",
    "--reasoning-effort", "medium",
]
TOML
rm -rf "$OUT/session-decoy"
"$BINARY" benchmark-run \
    --config "$OUT/decoy.toml" --model "$MODEL" --prompts "$OUT/prompts.json" \
    --thresholds "$OUT/thresholds.json" --session "$OUT/session-decoy" \
    --harness "$HARNESS" \
    --baseline-runtime python-mlx-lm --baseline-profile gate-smoke-baseline \
    --candidate-runtime mlx-swift --candidate-profile gate-smoke-candidate \
    --python-bin "$BASELINE_VENV/bin/python" --port "$((PORT + 20))" \
    --settle-seconds 0 --startup-timeout 90 --request-timeout 90 \
    > "$OUT/decoy.log" 2>&1
STATUS=$?
if [ "$STATUS" = "5" ]; then
    pass "a decoy baseline plus an unrelated --python-bin produces no decision (exit 5)"
else
    fail "the decoy provenance attack exited $STATUS, not 5"
    tail -20 "$OUT/decoy.log"
fi
grep -qF "runtime revision cannot be attributed to the process that served" "$OUT/decoy.log" \
    && pass "the production refusal names the unowned runtime revision" \
    || fail "the decoy refusal did not name the provenance mismatch"
grep -q '"accepted" : true' "$OUT/decoy.log" \
    && fail "the decoy provenance attack still reported accepted=true" \
    || pass "the decoy provenance attack cannot mint an accepted record"
if pgrep -f "$OUT/decoy.toml" >/dev/null 2>&1; then
    fail "the decoy provenance refusal left its model-harness process group alive"
else
    pass "the decoy provenance refusal terminates its owned process group"
fi

BASE_RECORD="$SESSION/records/python-mlx-lm.json"
CAND_RECORD="$SESSION/records/mlx-swift.json"
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
BOUND_STATUS=$?
if [ "$BOUND_STATUS" -eq 0 ]; then
    pass "every measurement carries the completions it came from, sealed by the observer"
else
    fail "the control's records are not bound to observed completions"
fi

info "review's own reproduction: two live /v1/models placeholders, driven for real"
STATUS="$(run_benchmark "$OUT/models-only.toml" "$OUT/session-placeholder" \
    "$OUT/placeholder.log" "$((PORT + 1))")"
if [ "$STATUS" = "4" ]; then
    pass "a pass whose runtimes only listed models is inadmissible (exit 4)"
else
    fail "the placeholder pass exited $STATUS, not 4"
    tail -20 "$OUT/placeholder.log"
fi
if grep -qF "answered other endpoints and served nothing" "$OUT/placeholder.log"; then
    pass "the refusal names what was missing rather than scoring it"
else
    fail "the placeholder refusal does not say that nothing was served"
fi
if grep -q '"accepted" : true' "$OUT/placeholder.log"; then
    fail "the placeholder pass still reported accepted=true"
else
    pass "the placeholder pass produced no decision at all"
fi

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
    --baseline-runtime python-mlx-lm --baseline-profile gate-smoke-baseline \
    --candidate-runtime mlx-swift --candidate-profile gate-smoke-candidate \
    --port "$((PORT + 2))" --settle-seconds 2 --startup-timeout 90 --request-timeout 90 \
    --skip tool_call > "$OUT/skipped.log" 2>&1
STATUS=$?
if [ "$STATUS" = "4" ]; then
    pass "a pair missing a required parity scenario is refused (exit 4)"
else
    fail "the skipped-scenario pass exited $STATUS, not 4"
    tail -10 "$OUT/skipped.log"
fi
if grep -qF "has no scenario" "$OUT/skipped.log"; then
    pass "the refusal names the scenario neither runtime ran"
else
    fail "the skipped-scenario refusal does not name the missing scenario"
fi

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
    if [ "$STATUS" = "2" ]; then
        pass "benchmark-run refuses $FLAG as a usage error"
    else
        fail "benchmark-run accepted $FLAG and exited $STATUS"
    fi
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
python3 - "$SESSION/decision.json" "$OUT/replay-decision.json" <<'REPLAYPY'
import json, sys
control = json.load(open(sys.argv[1]))
replay = json.load(open(sys.argv[2]))
assert replay == control, (control, replay)
print("OK")
REPLAYPY
REPLAY_STATUS=$?
if [ "$REPLAY_STATUS" -eq 0 ]; then
    pass "the replay reproduces the control's exact threshold outcome"
else
    fail "the replay did not reproduce the control's verdict"
fi
if grep -qF "never an acceptance" "$OUT/replay.log"; then
    pass "the replay says in words that it cannot grant an acceptance"
else
    fail "the replay does not state its own limit"
fi

info "the same real pair, scored against bands it cannot meet"
STATUS="$(compare "$BASE_RECORD" "$CAND_RECORD" "$ATTEST" "$OUT/strict.log" \
    "$OUT/thresholds-strict.json")"
if [ "$STATUS" = "3" ]; then
    pass "the thresholds a caller names reach the production entry (exit 3)"
else
    fail "strict thresholds exited $STATUS, not 3"
fi
if grep -q '"accepted" : false' "$OUT/strict.log"; then
    pass "a pair outside its bands is reported as not accepted"
else
    fail "strict thresholds did not produce blockers"
fi

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
            "$directory/python-mlx-lm.attestation.json"
            "$directory/mlx-swift.attestation.json"
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
python3 - "$CAND_RECORD" "$OUT/notranscript-candidate.json" <<'PY'
import json, sys
record = json.load(open(sys.argv[1]))
record["scenarios"][0]["transcript"] = None
json.dump(record, open(sys.argv[2], "w"), indent=2, sort_keys=True)
PY
expect_refusal "a measurement with no transcript is refused" \
    "no transcript" \
    "$BASE_RECORD" "$OUT/notranscript-candidate.json" "$ATTEST" "$OUT/notranscript.log"

info "an observation that seals no transcript"
fork_attestations "$OUT/attest-unsealed" mlx-swift '{"transcriptDigest": null}'
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
fork_attestations "$OUT/attest-open" mlx-swift \
    '{"closedAtUnixSeconds": null, "servedModelID": null}'
expect_refusal "an unclosed observation is not a watched pass" \
    "opened and never closed" \
    "$BASE_RECORD" "$CAND_RECORD" "$OUT/attest-open" "$OUT/open.log"

info "an observation of a runtime serving some other model"
fork_attestations "$OUT/attest-model" mlx-swift \
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
printf 'not json at all' > "$OUT/attest-malformed/mlx-swift.attestation.json"
expect_refusal "a failed read is reported as a read failure, not as absence" \
    "is malformed" \
    "$BASE_RECORD" "$CAND_RECORD" "$OUT/attest-malformed" "$OUT/malformed.log"

info "a replay asked for without naming an attestation directory"
"$BINARY" benchmark-compare --baseline "$BASE_RECORD" --candidate "$CAND_RECORD" \
    --thresholds "$OUT/thresholds.json" > "$OUT/noattest.log" 2>&1
STATUS=$?
if [ "$STATUS" = "2" ]; then
    pass "omitting --attestations is a usage error, not a comparison (exit 2)"
else
    fail "omitting --attestations exited $STATUS, not 2"
fi

# ------------------------------------------------- the pins, attacked --
# Effective generation parameters are attacked above through production
# `benchmark-run`: missing reports, rewritten argv, repeated argparse values,
# and argparse abbreviation all reach the real live-attestation call site.
# Replay tests below continue attacking record and attestation integrity, but
# deliberately do not infer effective configuration from edited launch argv.

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


# ============================================================================
# 3. The KV bound, at the production entry, for a runtime that is never
#    unbounded.
#
# `llama-server` has no unbounded mode. The derivation used to read the absence
# of `--max-kv-size` as `unbounded`, which for that runtime is simply false --
# and because the pin comparison demands equality, a false `kv=unbounded` would
# have MATCHED a genuinely unbounded MLX baseline and the gate would have stayed
# green over a 32k window compared against no window. The bound is now read from
# the running server inside the gate's own observation window.
#
# Every case below is a real `benchmark-run`: two spawned processes, driven,
# measured, sealed and judged by the shipped subcommand. No record and no
# attestation is edited here, so a gate that read the bound off argv instead of
# off the process cannot pass any of them.
# ============================================================================

# Two profiles whose KV and prefill conditions are set per case. `--n-ctx` is a
# flag of the stand-in, not of the derivation: it decides what the *process*
# reports, which is exactly the input argv cannot supply.
write_kv_config() {
    local path="$1" base_extra="$2" cand_extra="$3" cand_nctx="$4"
    # Argv fragments handed straight to the stand-in, not to the derivation:
    # they decide what the *process* answers, which is exactly the input argv
    # cannot supply. `--n-ctx-string` and `--slots-status` are the two failure
    # shapes review drove through this entry and got exit 0 from.
    local cand_probe="${5:-}" base_probe="${6:-}"
    local cand_meta=""
    [ -n "$cand_nctx" ] && cand_meta="\"--n-ctx\", \"$cand_nctx\","
    cat > "$path" <<TOML
[profiles.gate-smoke-baseline]
mode = "local"
executable = "$BASELINE_VENV/bin/mlx_lm.server"
argv = [
    "{port}", "$MODEL", "serving",
    $base_probe
    "--host", "{host}",
    "--model", "$MODEL",
    "--reasoning-effort", "medium",
    $base_extra
]

[profiles.gate-smoke-candidate]
mode = "local"
executable = "$CANDIDATE_PYTHON"
argv = [
    "$OUT/fake-runtime.py", "{port}", "$MODEL", "serving",
    $cand_meta
    $cand_probe
    "--host", "{host}",
    "--model", "$MODEL",
    "--reasoning-effort", "medium",
    $cand_extra
]
TOML
}

# `expect_run <label> <expected-exit> <needle|-> <config> <session> <log> <port>`
expect_run() {
    local label="$1" expected="$2" needle="$3" config="$4" session="$5" log="$6" port="$7"
    local status
    status="$(run_benchmark "$config" "$session" "$log" "$port")"
    if [ "$status" != "$expected" ]; then
        fail "$label: expected exit $expected, got $status"
        tail -15 "$log"
        return
    fi
    if [ "$needle" != "-" ] && ! grep -qF "$needle" "$log"; then
        fail "$label: exit $expected but the output never mentions $needle"
        tail -15 "$log"
        return
    fi
    pass "$label"
}

# `expect_admitted_run <label> <config> <session> <log> <port>`
#
# Admission and threshold acceptance are different claims. Exit 0 means the
# measured pair was admitted and inside its bands; exit 3 means it was admitted
# and outside at least one band. These cases attack an admission predicate, so
# scheduler noise in the fake runtimes must not turn them into a flaky speed
# test. A real admission refusal remains exit 4 and still fails this helper.
expect_admitted_run() {
    local label="$1" config="$2" session="$3" log="$4" port="$5"
    local status
    status="$(run_benchmark "$config" "$session" "$log" "$port")"
    if { [ "$status" != "0" ] && [ "$status" != "3" ]; } \
        || [ ! -s "$session/decision.json" ]; then
        fail "$label: expected an admitted decision (exit 0 or 3), got $status"
        tail -15 "$log"
        return
    fi
    pass "$label (admitted, exit $status)"
}

info "THE ACCEPTANCE QUESTION: two live runtimes reporting different bounds"
# Both runtimes report the cache they actually constructed. This is the merged
# bounded-Python path: 76800 and 32768 must remain different pins even though
# the candidate names no context flag at all.
write_kv_config "$OUT/kv-false-match.toml" \
    '"--prefill-step-size", "2048", "--max-kv-size", "76800",' \
    '"--ubatch-size", "2048",' 32768
expect_run "a 32k context window cannot match a 76800-token baseline" \
    4 'pinned condition "contextPolicy" differs' \
    "$OUT/kv-false-match.toml" "$OUT/session-kv-false" "$OUT/kv-false.log" "$((PORT + 3))"
if grep -qF "kv=76800" "$OUT/kv-false.log" && grep -qF "kv=32768" "$OUT/kv-false.log"; then
    pass "the refusal names both readings rather than one derived string"
else
    fail "the false-match refusal does not show the two bounds it compared"
fi

info "a llama.cpp-shaped candidate that genuinely shares the bound is admitted"
# The other half of the question: a gate that refused every reporting runtime
# would satisfy the case above and be useless. The bounded baseline and the
# llama.cpp-shaped candidate both report an effective 8192-token cache, so they
# compare equal -- with no clause relaxed and no argv value spent as evidence.
write_kv_config "$OUT/kv-match.toml" \
    '"--prefill-step-size", "2048", "--max-kv-size", "8192",' \
    '"--ctx-size", "8192", "--ubatch-size", "2048",' 8192
expect_admitted_run "a bound reported by the process and a bound pinned in argv compare equal" \
    "$OUT/kv-match.toml" "$OUT/session-kv-match" "$OUT/kv-match.log" "$((PORT + 4))"

# And the pin the accepted pass carries is the number the process reported, in
# both documents the gate wrote. A driver that kept deriving from argv would
# still accept this pair and would write `kv=unbounded` on the candidate.
python3 - "$OUT/session-kv-match" <<'KVPY'
import json, sys, os
session = sys.argv[1]
records = os.path.join(session, "records")
attest = os.path.join(session, "attest")
candidate = json.load(open(os.path.join(records, "mlx-swift.json")))
baseline = json.load(open(os.path.join(records, "python-mlx-lm.json")))
assert candidate["pins"]["contextPolicy"] == "kv=8192;prefill-step=2048;reasoning=medium", \
    candidate["pins"]["contextPolicy"]
assert baseline["pins"]["contextPolicy"] == "kv=8192;prefill-step=2048;reasoning=medium", \
    baseline["pins"]["contextPolicy"]
window = json.load(
    open(os.path.join(attest, "mlx-swift.attestation.json")))["observedContextWindow"]
assert window == {"state": "reported", "length": 8192}, window
silent = json.load(
    open(os.path.join(attest, "python-mlx-lm.attestation.json")))["observedContextWindow"]
assert silent == {"state": "reported", "length": 8192}, silent
print("OK")
KVPY
KV_STATUS=$?
if [ "$KV_STATUS" -eq 0 ]; then
    pass "the pin and the attestation both carry the bound the process reported"
else
    fail "the accepted pass did not record the bound it read off the runtime"
fi

info "a --ctx-size the process did not honour"
# Asked for 8192, running 4096. The pin takes the process's number, so the two
# records still agree and every clause above this one is satisfied.
write_kv_config "$OUT/kv-not-honoured.toml" \
    '"--prefill-step-size", "2048", "--max-kv-size", "4096",' \
    '"--ctx-size", "8192", "--ubatch-size", "2048",' 4096
expect_run "a launch that pinned a bound the process did not run is refused" \
    4 "is not the one the process ran" \
    "$OUT/kv-not-honoured.toml" "$OUT/session-kv-honour" "$OUT/kv-honour.log" "$((PORT + 5))"

info "F2 THE ACCEPTANCE QUESTION: a finite window answered as a malformed field"
# Review's attack, verbatim. The candidate runs a real 32768-token window and
# answers `meta.n_ctx` as the JSON *string* "32768". Under the previous reading
# that collapsed into "the runtime named no bound", the argv fallback derived
# `kv=unbounded`, the two pins were byte-identical, and the shipped entry
# returned exit 0 with accepted=true against a genuinely unbounded baseline.
# A malformed field is not an absent field: it is a failed read, and `kv=unread`
# is unpinnable.
write_kv_config "$OUT/kv-malformed.toml" \
    '"--prefill-step-size", "2048", "--max-kv-size", "76800",' \
    '"--ubatch-size", "2048",' '' \
    '"--n-ctx-string", "32768",'
expect_run "a malformed n_ctx cannot buy an unbounded pin" \
    4 "kv=unread" \
    "$OUT/kv-malformed.toml" "$OUT/session-kv-malformed" "$OUT/kv-malformed.log" \
    "$((PORT + 8))"
if grep -qF '"accepted" : true' "$OUT/kv-malformed.log"; then
    fail "the malformed-window pair was still scored"
else
    pass "the malformed-window pair produced no decision at all"
fi

# And the attestation says which of the two facts happened. A gate that kept
# folding the malformed field into the absence would write `notReported` here
# and derive `unbounded` from it.
python3 - "$OUT/session-kv-malformed" <<'MALPY'
import json, os, sys
attest = os.path.join(sys.argv[1], "attest")
window = json.load(
    open(os.path.join(attest, "mlx-swift.attestation.json")))["observedContextWindow"]
assert window == {"state": "unread"}, window
print("OK")
MALPY
MAL_STATUS=$?
if [ "$MAL_STATUS" -eq 0 ]; then
    pass "the attestation records a failed read, not a runtime that named no bound"
else
    fail "the malformed window was recorded as something other than unread"
fi

info "G1: a launch that left the prompt-evaluation chunk to the runtime default"
# `llama-server` defaults `--ubatch-size` to 512 where `mlx_lm.server` defaults
# `--prefill-step-size` to 2048. Both sides here state neither, and the live
# runtime reports make those different effective values visible. `--batch-size`
# is the logical batch and deliberately does not change the prompt-evaluation
# chunk.
write_kv_config "$OUT/kv-unpinned-chunk.toml" \
    '"--max-kv-size", "8192",' '"--ctx-size", "8192", "--batch-size", "2048",' 8192
expect_run "an unstated prompt-evaluation chunk is refused for llama.cpp too" \
    4 'pinned condition "contextPolicy" differs' \
    "$OUT/kv-unpinned-chunk.toml" "$OUT/session-kv-chunk" "$OUT/kv-chunk.log" "$((PORT + 6))"
if grep -qF 'prefill-step=2048' "$OUT/kv-chunk.log" \
    && grep -qF 'prefill-step=512' "$OUT/kv-chunk.log"; then
    pass "the refusal names both runtime defaults rather than one shared fixture default"
else
    fail "the unstated-chunk refusal did not expose both effective defaults"
fi

info "G1: -ub is the same condition as --prefill-step-size, not a third one"
write_kv_config "$OUT/kv-short-flag.toml" \
    '"--prefill-step-size", "2048", "--max-kv-size", "8192",' \
    '"--ctx-size", "8192", "-ub", "2048",' 8192
expect_admitted_run "the -ub spelling pins the chunk at the same value" \
    "$OUT/kv-short-flag.toml" "$OUT/session-kv-ub" "$OUT/kv-ub.log" "$((PORT + 7))"


# ============================================================================
# 4. G4: the model pin, and speculative decoding, at the production entry.
#
# `mlx_lm.server` serves a weight DIRECTORY and `llama-server` serves a single
# `.gguf` FILE. Two files, two paths, two digests, one upstream model. While
# `modelPath` and `modelDigest` were pins compared for equality, that comparison
# was refused forever -- not because it was unsound, but because the pin had
# been written about the local file rather than about the model.
#
# The replacement pins the shared source of record and demands digest-bound
# equivalence evidence for anything else. Absence of that evidence is a refusal,
# a FAILED READ of it is a different refusal, and neither is a default pass.
# Every case below is a real `benchmark-run`: no record and no attestation is
# edited, so a gate that took the caller's word for any of it fails them.
#
# The candidate artifact here is a real file on disk, digested whole by the gate
# exactly as the 29 GB GGUF would be. Every digest in every verdict below is
# recomputed by this script the way the gate computes it, so nothing here is
# bound to a hardcoded hash.
# ============================================================================

GGUF="$OUT/candidate-weights.gguf"
printf 'GGUF\0\0\0not-a-real-model-just-bytes-the-gate-digests\n' > "$GGUF"
GGUF_DIGEST="$(shasum -a 256 "$GGUF" | cut -d' ' -f1)"

# The gate digests a weight *directory* over its config.json and safetensors
# index, each prefixed by its file name. Recomputed the same way here so the
# verdict names the number the gate will compute rather than one this script
# asserted.
MLX_DIGEST="$(python3 "$OUT/mlx-digest.py" "$MODEL")"

MTP_NOTE="the MLX build drops the MTP head the GGUF carries"
VISION_NOTE="the vision tower is in both files and resident in neither on the text path"
NORM_NOTE="GGUF norms are F32 where the MLX build keeps bf16"
SOURCE_OF_RECORD="hf:orcarouter/Qwen3.8-27B-Uncensored-BF16@a855f377"

# `write_verdict <path> <verdict> <covers-the-gguf> <declares-non-equivalences>`
write_verdict() {
    python3 "$OUT/write-verdict.py" "$1" "$2" "$3" "$4" \
        "$SOURCE_OF_RECORD" "$MODEL" "$MLX_DIGEST" "$GGUF" "$GGUF_DIGEST" \
        "$MTP_NOTE" "$VISION_NOTE" "$NORM_NOTE"
}

write_verdict "$OUT/equivalence.json" comparable true true
write_verdict "$OUT/equivalence-not-comparable.json" notComparable true true
write_verdict "$OUT/equivalence-other-artifact.json" comparable false true
write_verdict "$OUT/equivalence-no-notes.json" comparable true false
printf 'this is not JSON at all\n' > "$OUT/equivalence-broken.json"

# Two profiles serving two different artifacts. The candidate's argv carries the
# `.gguf` path, so the "the launch has to mention the artifact it claims to have
# served" clause is satisfied by the file rather than by the directory.
write_crossformat_config() {
    local path="$1"
    cat > "$path" <<TOML
[profiles.gate-smoke-baseline]
mode = "local"
executable = "$BASELINE_VENV/bin/mlx_lm.server"
argv = [
    "{port}", "$MODEL", "serving",
    "--host", "{host}",
    "--model", "$MODEL",
    "--max-kv-size", "8192",
    "--prefill-step-size", "2048",
    "--reasoning-effort", "medium",
]

[profiles.gate-smoke-candidate]
mode = "local"
executable = "$CANDIDATE_PYTHON"
argv = [
    "$OUT/fake-runtime.py", "{port}", "$GGUF", "serving",
    "--n-ctx", "8192",
    "--host", "{host}",
    "--model", "$GGUF",
    "--ctx-size", "8192",
    "--ubatch-size", "2048",
    "--reasoning-effort", "medium",
]
TOML
}
write_crossformat_config "$OUT/crossformat.toml"

# `run_crossformat <config> <session> <log> <port> [extra flags...]`
run_crossformat() {
    local config="$1" session="$2" log="$3" port="$4"
    shift 4
    rm -rf "$session"
    "$BINARY" benchmark-run \
        --config "$config" --model "$MODEL" --candidate-model "$GGUF" \
        --prompts "$OUT/prompts.json" \
        --thresholds "$OUT/thresholds.json" --session "$session" --harness "$HARNESS" \
        --baseline-runtime gate-smoke-baseline --baseline-profile gate-smoke-baseline \
        --candidate-runtime gate-smoke-candidate --candidate-profile gate-smoke-candidate \
        --port "$port" --settle-seconds 2 --startup-timeout 90 --request-timeout 90 \
        "$@" > "$log" 2>&1
    printf '%s' "$?"
}

# `expect_crossformat <label> <exit> <needle|-> <config> <session> <log> <port> [extra...]`
expect_crossformat() {
    local label="$1" expected="$2" needle="$3" config="$4" session="$5" log="$6" port="$7"
    shift 7
    local status
    status="$(run_crossformat "$config" "$session" "$log" "$port" "$@")"
    if [ "$status" != "$expected" ]; then
        fail "$label: expected exit $expected, got $status"
        tail -15 "$log"
        return
    fi
    if [ "$needle" != "-" ] && ! grep -qF "$needle" "$log"; then
        fail "$label: exit $expected but the output never mentions $needle"
        tail -15 "$log"
        return
    fi
    pass "$label"
}

info "F1 THE ACCEPTANCE QUESTION: a verdict the caller wrote for itself"
# Review's attack, verbatim and then some. This document is `comparable`, names
# the real upstream model, names BOTH artifacts at the digests the gate computed
# for them itself, carries the correct quantization labels, and declares all
# three measured non-equivalences word for word. Every clause the previous
# revision checked is satisfied. It was authored by the invocation asking to be
# believed, so it is not evidence, and the gate says so before it launches
# anything.
expect_crossformat "a caller-authored verdict is refused however well-shaped it is" \
    5 "is not an equivalence decision this repository took" \
    "$OUT/crossformat.toml" "$OUT/session-g4-minted" "$OUT/g4-minted.log" \
    "$((PORT + 10))" --equivalence "$OUT/equivalence.json"
if grep -qF '"accepted" : true' "$OUT/g4-minted.log"; then
    fail "the minted-verdict pair was still scored"
else
    pass "the minted-verdict pair produced no decision at all"
fi

info "the decision this repository DID take reaches the clauses beyond the trust lookup"
# The other half, and it is what makes the check above a gate rather than a wall.
# This is the shipped document, byte for byte, so the trust lookup finds it --
# and the refusal that follows is a *different* one, from a clause further in:
# the decision is about the real 29 GB pair and says nothing about these fixture
# bytes. A gate that trusted nothing would answer the untrusted refusal here too.
expect_crossformat "the trusted decision is admitted as evidence and bound to its own artifacts" \
    5 "no equivalence verdict names an artifact at digest" \
    "$OUT/crossformat.toml" "$OUT/session-g4-trusted" "$OUT/g4-trusted.log" \
    "$((PORT + 22))" --equivalence "$TRUSTED_VERDICT"
if grep -qF "is not an equivalence decision this repository took" "$OUT/g4-trusted.log"; then
    fail "the shipped decision was refused as untrusted by the gate that ships it"
else
    pass "the shipped decision passes the trust lookup at the production entry"
fi

info "the same two artifacts with no equivalence evidence at all"
# Absence is a refusal, and it lands before anything is launched. For a single
# weight file it lands one clause earlier than the model-of-record check: a
# `.gguf` has no `config.json`, so with no verdict there is nowhere to read its
# quantization from and the run is refused for that, by name.
expect_crossformat "a GGUF candidate without a verdict cannot even be pinned" \
    5 "no equivalence verdict names an artifact at digest" \
    "$OUT/crossformat.toml" "$OUT/session-g4-none" "$OUT/g4-none.log" "$((PORT + 11))"

info "two different weight directories with no equivalence evidence"
# The same refusal where nothing else can fire first: both artifacts are
# directories that declare their own quantization, so the run gets all the way
# to the model-of-record check and is refused there for serving two different
# models under no declared equivalence.
OTHER_MODEL="$OUT/other-model"
mkdir -p "$OTHER_MODEL"
cat > "$OTHER_MODEL/config.json" <<'JSON'
{"model_type": "gate-smoke", "quantization": {"bits": 6, "group_size": 64, "mode": "affine"}}
JSON
cat > "$OTHER_MODEL/model.safetensors.index.json" <<'JSON'
{"metadata": {"total_size": 2}, "weight_map": {"a": "model-00001.safetensors"}}
JSON
cat > "$OUT/two-directories.toml" <<TOML
[profiles.gate-smoke-baseline]
mode = "local"
executable = "$BASELINE_VENV/bin/mlx_lm.server"
argv = [
    "{port}", "$MODEL", "serving",
    "--host", "{host}",
    "--model", "$MODEL",
    "--max-kv-size", "8192",
    "--prefill-step-size", "2048",
    "--reasoning-effort", "medium",
]

[profiles.gate-smoke-candidate]
mode = "local"
executable = "$CANDIDATE_PYTHON"
argv = [
    "$OUT/fake-runtime.py", "{port}", "$OTHER_MODEL", "serving",
    "--host", "{host}",
    "--model", "$OTHER_MODEL",
    "--max-kv-size", "8192",
    "--prefill-step-size", "2048",
    "--reasoning-effort", "medium",
]
TOML
rm -rf "$OUT/session-g4-two-dirs"
"$BINARY" benchmark-run \
    --config "$OUT/two-directories.toml" --model "$MODEL" \
    --candidate-model "$OTHER_MODEL" --prompts "$OUT/prompts.json" \
    --thresholds "$OUT/thresholds.json" --session "$OUT/session-g4-two-dirs" \
    --harness "$HARNESS" \
    --baseline-runtime gate-smoke-baseline --baseline-profile gate-smoke-baseline \
    --candidate-runtime gate-smoke-candidate --candidate-profile gate-smoke-candidate \
    --port "$((PORT + 21))" --settle-seconds 2 --startup-timeout 90 --request-timeout 90 \
    > "$OUT/g4-two-dirs.log" 2>&1
STATUS=$?
if [ "$STATUS" = "5" ] \
        && grep -qF "pass --equivalence with a verdict naming the upstream model" \
            "$OUT/g4-two-dirs.log"; then
    pass "two different weight directories without a verdict are refused"
else
    fail "two different directories with no verdict exited $STATUS"
    tail -15 "$OUT/g4-two-dirs.log"
fi

info "a verdict the gate cannot read"
# A failed read is not an absence. Reading it as one would turn an unreadable
# file into a same-format pass over two different models.
expect_crossformat "an unreadable verdict is refused, not spent as an absence" \
    5 "could not be read or decoded" \
    "$OUT/crossformat.toml" "$OUT/session-g4-broken" "$OUT/g4-broken.log" "$((PORT + 12))" \
    --equivalence "$OUT/equivalence-broken.json"

info "the trust lookup runs before anything the document says about itself"
# What the verdict claims stops mattering once the caller wrote it. These three
# are `notComparable`, about some other pair of files, and declaring nothing --
# each of which had its own refusal, and each of which is now refused one clause
# earlier and for the reason that dominates all of them. Those content clauses
# are still there and still fire; they are exercised against a trusted document
# by `RuntimeBenchmarkModelOfRecordTests`, which is the only place they are
# reachable now.
PORT_OFFSET=13
for MINTED in equivalence-not-comparable equivalence-other-artifact equivalence-no-notes; do
    expect_crossformat "a caller-authored $MINTED is refused as untrusted, not on its contents" \
        5 "is not an equivalence decision this repository took" \
        "$OUT/crossformat.toml" "$OUT/session-g4-$MINTED" "$OUT/g4-$MINTED.log" \
        "$((PORT + PORT_OFFSET))" --equivalence "$OUT/$MINTED.json"
    PORT_OFFSET=$((PORT_OFFSET + 1))
done

info "evidence beside a pair that serves one artifact"
# Nothing for the two to be equivalent to, so the verdict is a claim nobody
# checked. Refused rather than ignored -- and this is the second production
# proof that the trusted document is accepted as evidence, because reaching this
# clause at all means the trust lookup found it and every clause between here
# and there let it through.
rm -rf "$OUT/session-g4-unused"
"$BINARY" benchmark-run \
    --config "$OUT/serving.toml" --model "$MODEL" --prompts "$OUT/prompts.json" \
    --thresholds "$OUT/thresholds.json" --session "$OUT/session-g4-unused" \
    --harness "$HARNESS" \
    --baseline-runtime gate-smoke-baseline --baseline-profile gate-smoke-baseline \
    --candidate-runtime gate-smoke-candidate --candidate-profile gate-smoke-candidate \
    --port "$((PORT + 16))" --settle-seconds 2 --startup-timeout 90 --request-timeout 90 \
    --equivalence "$TRUSTED_VERDICT" > "$OUT/g4-unused.log" 2>&1
STATUS=$?
if [ "$STATUS" = "4" ] && grep -qF "while both passes served the same artifact" \
        "$OUT/g4-unused.log"; then
    pass "an unused verdict beside a same-artifact pair is refused"
else
    fail "a same-artifact pair citing a verdict exited $STATUS"
    tail -15 "$OUT/g4-unused.log"
fi

# ---------------------------------------------- speculative decoding --
#
# `llama-server` can draft off this model's MTP head; the MLX baseline has no
# MTP head at all. A tokens/s measured with speculation on is a different
# decoding algorithm, not a faster runtime, and TASK-260828-3g87i4's verdict
# names it as the one way this pair genuinely stops being comparable. So the
# gate REFUSES the reading rather than requiring the two sides to match: two
# speculating runtimes would agree on the pin and still not be a result.

# These run on a SAME-FORMAT pair: both passes serve the fixture weight
# directory, so no equivalence evidence is involved and every refusal below is
# about speculation alone. That separation is the point -- the previous revision
# ran them through the cross-format path, where a change to the model pin could
# have masked a change to this one.

info "a candidate the gate measured to be speculating"
# BOTH sides drafting. A pair in which only the candidate drafts is already
# refused by the `speculation` equality pin, which is the more general refusal;
# the case that needs a clause of its own is the one the pin comparison cannot
# see -- two runtimes agreeing on every pin and still not a migration result.
write_kv_config "$OUT/spec-on.toml" \
    '"--prefill-step-size", "2048", "--max-kv-size", "8192",' \
    '"--ctx-size", "8192", "--ubatch-size", "2048",' 8192 \
    '"--slots", "true",' '"--slots", "true",'
expect_run "a runtime reporting speculative decoding is refused" \
    4 'only "off" is comparable' \
    "$OUT/spec-on.toml" "$OUT/session-spec-on" "$OUT/spec-on.log" "$((PORT + 17))"

info "a candidate that answers /slots and is not speculating"
# The other half. A gate that refused every runtime able to answer the question
# would refuse llama.cpp itself, which is the runtime this task exists to admit.
write_kv_config "$OUT/spec-off.toml" \
    '"--prefill-step-size", "2048", "--max-kv-size", "8192",' \
    '"--ctx-size", "8192", "--ubatch-size", "2048",' 8192 \
    '"--slots", "false",' '"--slots", "false",'
expect_admitted_run "a runtime reporting no speculation is admitted" \
    "$OUT/spec-off.toml" "$OUT/session-spec-off" "$OUT/spec-off.log" "$((PORT + 18))"

info "F3 THE ACCEPTANCE QUESTION: /slots answers HTTP 500 and the launch says nothing"
# Review's attack, verbatim. The route is there, the answer is a failure, and
# neither launch declares a speculative flag. Under the previous reading every
# non-200 collapsed into "the runtime named no speculation state", the argv
# fallback then derived `off`, and a fixture configured as speculative was
# scored as MTP-off with exit 0 and accepted=true. Both sides fail the same way
# here on purpose: their pins agree, so nothing above the refusal can see it,
# and the gate has to establish MTP is off rather than fail to establish it is
# on.
write_kv_config "$OUT/spec-unread.toml" \
    '"--prefill-step-size", "2048", "--max-kv-size", "8192",' \
    '"--ctx-size", "8192", "--ubatch-size", "2048",' 8192 \
    '"--slots-status", "500",' '"--slots-status", "500",'
expect_run "a failed /slots observation cannot be spent as MTP-off" \
    4 'reads "unread" for speculative decoding' \
    "$OUT/spec-unread.toml" "$OUT/session-spec-unread" "$OUT/spec-unread.log" \
    "$((PORT + 23))"
if grep -qF '"accepted" : true' "$OUT/spec-unread.log"; then
    fail "the failed-observation pair was still scored"
else
    pass "the failed-observation pair produced no decision at all"
fi

# And the attestation says which of the two facts happened, on both sides.
python3 - "$OUT/session-spec-unread" <<'SPECPY'
import json, os, sys
attest = os.path.join(sys.argv[1], "attest")
for runtime in ("python-mlx-lm", "mlx-swift"):
    state = json.load(
        open(os.path.join(attest, f"{runtime}.attestation.json")))["observedSpeculation"]
    assert state == {"state": "unread"}, (runtime, state)
print("OK")
SPECPY
SPEC_STATUS=$?
if [ "$SPEC_STATUS" -eq 0 ]; then
    pass "a 500 is recorded as a failed read, not as a runtime that named no state"
else
    fail "the failed /slots observation was recorded as something other than unread"
fi

info "a 404 from /slots is still an absence, and still reaches the argv reading"
# The narrowing question. `mlx_lm.server` and this prototype 404 the route and
# `llama-server --no-slots` answers 501; a fix that read every non-200 as a
# failure would refuse the incumbent baseline outright. The `spec-off` case
# above already admits a reporting runtime; this one admits a silent one.
write_kv_config "$OUT/spec-silent.toml" \
    '"--prefill-step-size", "2048", "--max-kv-size", "8192",' \
    '"--ctx-size", "8192", "--ubatch-size", "2048",' 8192
expect_admitted_run "a runtime that does not serve /slots at all is still admitted" \
    "$OUT/spec-silent.toml" "$OUT/session-spec-silent" "$OUT/spec-silent.log" \
    "$((PORT + 24))"

info "a launch that asks for speculation from a runtime that will not answer"
# The /slots-less bypass: no `--slots` flag on this stand-in, so it 404s the
# route exactly as an MLX runtime does, and the launch asks for drafting anyway.
# The argv reading closes it.
write_kv_config "$OUT/spec-declared.toml" \
    '"--prefill-step-size", "2048", "--max-kv-size", "8192",' \
    '"--ctx-size", "8192", "--ubatch-size", "2048", "--spec-type", "ngram-mod",' 8192
expect_run "a declared --spec-type against a silent runtime is refused" \
    4 "declared:--spec-type=ngram-mod" \
    "$OUT/spec-declared.toml" "$OUT/session-spec-declared" "$OUT/spec-declared.log" \
    "$((PORT + 19))"

info "speculation configured through the environment rather than the argv"
# `llama-server` reads LLAMA_ARG_SPEC_TYPE, and this process's environment is
# what it hands the launcher, so an inherited variable would put the runtime
# under test into drafting while every recorded argv showed nothing. Refused
# before launch, and refused rather than silently scrubbed.
rm -rf "$OUT/session-g4-env"
LLAMA_ARG_SPEC_TYPE=ngram-mod "$BINARY" benchmark-run \
    --config "$OUT/crossformat.toml" --model "$MODEL" --candidate-model "$GGUF" \
    --prompts "$OUT/prompts.json" \
    --thresholds "$OUT/thresholds.json" --session "$OUT/session-g4-env" \
    --harness "$HARNESS" \
    --baseline-runtime gate-smoke-baseline --baseline-profile gate-smoke-baseline \
    --candidate-runtime gate-smoke-candidate --candidate-profile gate-smoke-candidate \
    --port "$((PORT + 20))" --settle-seconds 2 --startup-timeout 90 --request-timeout 90 \
    --equivalence "$TRUSTED_VERDICT" > "$OUT/g4-env.log" 2>&1
STATUS=$?
if [ "$STATUS" = "5" ] && grep -qF "LLAMA_ARG_SPEC_TYPE" "$OUT/g4-env.log"; then
    pass "an inherited LLAMA_ARG_SPEC_TYPE refuses the run by name"
else
    fail "an environment that configures drafting exited $STATUS"
    tail -15 "$OUT/g4-env.log"
fi


# ============================================================================
# 5. F5: the suite is validated before anything is launched.
#
# Review drove the shipped `benchmark-run` with a required `context_75k`
# scenario carrying `"prefix_repeats": "2027"` -- the count as a JSON *string*.
# `spec["prefix_repeats"] as? Int` is nil for a string, so the branch was
# skipped, the 16,232-token prefix was never built, a 15-TOKEN prompt was
# measured, and the invocation exited 0 with `accepted: true`. Both records
# sealed honestly and both transcripts faithfully recorded the wrong request:
# the gate scored a hollow capacity scenario perfectly.
#
# Every case below drives the shipped subcommand with a malformed suite and
# asserts three things, because a refusal that arrives after an hour of model
# loading is not the fix:
#
#   * a nonzero exit,
#   * NO decision emitted -- not a rejection, not an inadmissibility, nothing,
#   * NO session directory, which is created after the suite is read and before
#     the first launch, so its absence is evidence that no runtime was started.
# ============================================================================

SUITE_SESSION="$OUT/session-f5"
make_suite() {
    # $1 destination, $2 python expression mutating `doc`
    SRC="$OUT/prompts.json" DEST="$1" MUTATION="$2" python3 - <<'MUTATE'
import json, os
doc = json.load(open(os.environ["SRC"]))
exec(os.environ["MUTATION"])
json.dump(doc, open(os.environ["DEST"], "w"), indent=2)
MUTATE
}

refuse_suite() {
    local label="$1" suite="$2" fragment="$3"
    rm -rf "$SUITE_SESSION"
    "$BINARY" benchmark-run \
        --config "$OUT/serving.toml" --model "$MODEL" --prompts "$suite" \
        --thresholds "$OUT/thresholds.json" --session "$SUITE_SESSION" \
        --harness "$HARNESS" \
        --baseline-runtime gate-smoke-baseline --baseline-profile gate-smoke-baseline \
        --candidate-runtime gate-smoke-candidate --candidate-profile gate-smoke-candidate \
        --port "$((PORT + 22))" --settle-seconds 2 --startup-timeout 90 \
        --request-timeout 90 > "$OUT/f5.log" 2>&1
    local status=$?
    if [ "$status" = "0" ]; then
        fail "$label was measured and accepted (exit 0)"
        tail -15 "$OUT/f5.log"
        return
    fi
    if grep -qF "$fragment" "$OUT/f5.log"; then
        pass "$label is refused by name before any launch (exit $status)"
    else
        fail "$label exited $status but the refusal does not name $fragment"
        tail -15 "$OUT/f5.log"
    fi
    if grep -q '"accepted"' "$OUT/f5.log"; then
        fail "$label emitted a decision"
    else
        pass "$label emitted no decision at all"
    fi
    if [ -e "$SUITE_SESSION" ]; then
        fail "$label created a session directory, so the run got past the suite"
    else
        pass "$label started no runtime (no session directory)"
    fi
}

info "the finding itself: a required scenario's prefix count written as a string"
make_suite "$OUT/prompts-f5-string.json" \
    'doc["scenarios"]["short_prompt"]["prefix_repeats"] = "2027"'
refuse_suite "a string prefix count on a required scenario" \
    "$OUT/prompts-f5-string.json" 'scenarios.short_prompt.prefix_repeats'

info "the same 15-token prompt reached by a typo instead of by a type error"
make_suite "$OUT/prompts-f5-typo.json" \
    'doc["scenarios"]["short_prompt"]["prefix_repeat"] = 2027'
refuse_suite "a misspelled scenario field" \
    "$OUT/prompts-f5-typo.json" 'scenarios.short_prompt.prefix_repeat'

info "a count that would drive no work and still report a result"
make_suite "$OUT/prompts-f5-zero.json" \
    'doc["scenarios"]["stability_soak"]["iterations"] = 0'
refuse_suite "a soak with zero iterations" \
    "$OUT/prompts-f5-zero.json" 'scenarios.stability_soak.iterations'

info "a filler paragraph that hollows out every prefix without a malformed field"
make_suite "$OUT/prompts-f5-filler.json" 'doc["filler_paragraph"] = ""'
refuse_suite "an empty filler paragraph" \
    "$OUT/prompts-f5-filler.json" 'filler_paragraph'

info "a scenario kind this driver cannot run is refused, not silently skipped"
make_suite "$OUT/prompts-f5-kind.json" \
    'doc["scenarios"]["short_prompt"]["kind"] = "singel"'
refuse_suite "an unrecognised scenario kind" \
    "$OUT/prompts-f5-kind.json" 'scenarios.short_prompt.kind'

# ============================================================================
# 6. F6: the same rule one level deeper, inside the tool declaration.
#
# Review misspelled `parameters.required` as `parameters.require` in the
# `tool_call` scenario. The gate read that as the *supported absence* of
# `required`, so `requiredArguments` came out `[]`, the parity check in
# `BenchmarkScenarios.tool` had no argument key to demand back, a runtime that
# called the tool with an empty argument object satisfied it, and the shipped
# `benchmark-run` exited 0 with `accepted: true`.
#
# `required` is mandatory now. An explicit `[]` still means "this tool takes no
# mandatory arguments" -- it just has to be written down, because a key that is
# not there and a key that is misspelled are the same bytes from here.
#
# Scope of the fix, stated because revision 4 overstated its own: only the tool
# fields this benchmark reads are validated -- `type`, `function`,
# `function.name`, `function.parameters`, `function.parameters.required`. The
# rest of the JSON-Schema parameter block is forwarded to the runtime verbatim
# and is NOT inspected. A misspelling elsewhere inside it still reaches the
# runtime unremarked; that is a deliberate boundary, not a claim of coverage.
# ============================================================================

info "F6 the finding itself: required misspelled as require inside the tool schema"
make_suite "$OUT/prompts-f6-typo.json" \
    'p = doc["scenarios"]["tool_call"]["tools"][0]["function"]["parameters"];
p["require"] = p.pop("required")'
refuse_suite "a misspelled parameters.require" \
    "$OUT/prompts-f6-typo.json" 'scenarios.tool_call.tools[0].function.parameters.required'

info "the absence itself, now that it is not a supported shape"
make_suite "$OUT/prompts-f6-absent.json" \
    'del doc["scenarios"]["tool_call"]["tools"][0]["function"]["parameters"]["required"]'
refuse_suite "a tool declaring no required array at all" \
    "$OUT/prompts-f6-absent.json" 'scenarios.tool_call.tools[0].function.parameters.required'

info "a declaration whose type is not a function the parity check can read back"
make_suite "$OUT/prompts-f6-type.json" \
    'doc["scenarios"]["tool_call"]["tools"][0]["type"] = "code_interpreter"'
refuse_suite "a non-function tool declaration" \
    "$OUT/prompts-f6-type.json" 'scenarios.tool_call.tools[0].type'

# The narrowing control, and it is the one that stops "require an explicit
# array" from turning into "refuse every tool": a tool that genuinely takes no
# mandatory arguments writes `"required": []` and the whole pass reaches an
# admitted decision. The non-empty control is the smoke's own suite, driven
# through the same production entry with `"required": ["vehicle"]`.
info "an explicit empty required array is admitted, and the pass still runs"
make_suite "$OUT/prompts-f6-empty.json" \
    'doc["scenarios"]["tool_call"]["tools"][0]["function"]["parameters"]["required"] = []'
rm -rf "$OUT/session-f6-empty"
"$BINARY" benchmark-run \
    --config "$OUT/serving.toml" --model "$MODEL" --prompts "$OUT/prompts-f6-empty.json" \
    --thresholds "$OUT/thresholds.json" --session "$OUT/session-f6-empty" \
    --harness "$HARNESS" \
    --baseline-runtime gate-smoke-baseline --baseline-profile gate-smoke-baseline \
    --candidate-runtime gate-smoke-candidate --candidate-profile gate-smoke-candidate \
    --port "$((PORT + 23))" --settle-seconds 2 --startup-timeout 90 \
    --request-timeout 90 > "$OUT/f6-empty.log" 2>&1
STATUS=$?
if { [ "$STATUS" = "0" ] || [ "$STATUS" = "3" ]; } \
    && [ -s "$OUT/session-f6-empty/decision.json" ]; then
    pass "a tool that deliberately requires no arguments is measured and admitted (exit $STATUS)"
else
    fail "the explicit empty required array produced no admitted decision (exit $STATUS)"
    tail -15 "$OUT/f6-empty.log"
fi

# And the parity check is still the thing being satisfied, not bypassed: the
# tool_call scenario has to have actually succeeded in both records.
python3 - "$OUT/session-f6-empty" <<'F6PY'
import json, pathlib, sys
session = pathlib.Path(sys.argv[1])
records = sorted((session / "records").glob("*.json"))
assert records, f"no records under {session}/records"
for path in records:
    record = json.load(open(path))
    named = [s for s in record["scenarios"] if s["name"] == "tool_call"]
    assert named, f"{path}: no tool_call scenario"
    assert named[0]["succeeded"], f"{path}: tool_call did not succeed"
print("OK")
F6PY
F6_STATUS=$?
if [ "$F6_STATUS" -eq 0 ]; then
    pass "both records show tool_call actually succeeding under an empty demand list"
else
    fail "the empty-required pass did not really drive the tool scenario"
fi

# The other direction, and it is what stops the rule above from being satisfied
# by refusing everything: the suite this repository actually ships has to get
# PAST validation. Driven with an unreadable thresholds file, which is read
# immediately after the suite, so the refusal names the thresholds rather than
# the prompts exactly when the pinned suite is sound.
info "the pinned suite this repository ships still validates"
rm -rf "$SUITE_SESSION"
"$BINARY" benchmark-run \
    --config "$OUT/serving.toml" --model "$MODEL" \
    --prompts "$PACKAGE_ROOT/examples/benchmark-prompts.json" \
    --thresholds "$OUT/no-such-thresholds.json" --session "$SUITE_SESSION" \
    --harness "$HARNESS" \
    --baseline-runtime gate-smoke-baseline --baseline-profile gate-smoke-baseline \
    --candidate-runtime gate-smoke-candidate --candidate-profile gate-smoke-candidate \
    --port "$((PORT + 22))" > "$OUT/f5-shipped.log" 2>&1
STATUS=$?
if [ "$STATUS" != "0" ] && grep -qF "thresholds" "$OUT/f5-shipped.log" \
    && ! grep -qF "prompt suite" "$OUT/f5-shipped.log"; then
    pass "the shipped six-scenario suite passes validation and stops on the thresholds"
else
    fail "the shipped suite was refused by the schema (exit $STATUS)"
    tail -15 "$OUT/f5-shipped.log"
fi

# ============================================================================
# 7. TASK-260829-3cwcb6: cross-runtime timing and mmap memory, through the
#    production benchmark-run entry point.
# ============================================================================

info "reasoning and reasoning_content must start and end the same clock"
cat > "$OUT/reasoning-fields.toml" <<TOML
[profiles.gate-smoke-baseline]
mode = "local"
executable = "$BASELINE_VENV/bin/mlx_lm.server"
argv = [
    "{port}", "$MODEL", "serving",
    "--reasoning-field", "reasoning",
    "--host", "{host}",
    "--model", "$MODEL",
    "--max-kv-size", "8192",
    "--prefill-step-size", "2048",
    "--reasoning-effort", "medium",
]

[profiles.gate-smoke-candidate]
mode = "local"
executable = "$CANDIDATE_PYTHON"
argv = [
    "$OUT/fake-runtime.py", "{port}", "$MODEL", "serving",
    "--reasoning-field", "reasoning_content",
    "--host", "{host}",
    "--model", "$MODEL",
    "--max-kv-size", "8192",
    "--prefill-step-size", "2048",
    "--reasoning-effort", "medium",
]
TOML
expect_admitted_run "both reasoning spellings produce an admitted production decision" \
    "$OUT/reasoning-fields.toml" "$OUT/session-reasoning-fields" \
    "$OUT/reasoning-fields.log" "$((PORT + 24))"

# The decision alone proves the fields were non-nil; inspect the records to
# prove where the production clock actually landed. Under the old reader the
# candidate starts on the fourth/final event, so decode is unmeasured. The
# corrected reader starts on the first reasoning event and ends on the final
# content event for both spellings. Absolute and cross-process timing are not
# asserted here because concurrent vmmap sampling can delay either fixture;
# the production property is that both spellings yield the same four generated
# events and a first-to-last decode interval.
python3 - "$OUT/session-reasoning-fields" <<'REASONPY'
import json, pathlib, sys
records = pathlib.Path(sys.argv[1]) / "records"
base = json.load(open(records / "python-mlx-lm.json"))["scenarios"]
cand = json.load(open(records / "mlx-swift.json"))["scenarios"]
base = next(s for s in base if s["name"] == "short_prompt")
cand = next(s for s in cand if s["name"] == "short_prompt")
assert base["timeToFirstTokenSeconds"] is not None, base
assert cand["timeToFirstTokenSeconds"] is not None, cand
assert base["completionTokens"] == 4, base
assert cand["completionTokens"] == 4, cand
assert base["decodeTokensPerSecond"] is not None, base
assert cand["decodeTokensPerSecond"] is not None, cand
print("OK")
REASONPY
REASON_STATUS=$?
if [ "$REASON_STATUS" -eq 0 ]; then
    pass "production records clock first-to-last generated events identically"
else
    fail "production timing retained the field-name asymmetry"
fi

info "both runtime shapes score the same resident-memory upper bound"
python3 - "$OUT/fake-weights.gguf" <<'MAPPEDFIXTURE'
import pathlib, sys
pathlib.Path(sys.argv[1]).write_bytes(b"mapped-weight-fixture" * 524288)
MAPPEDFIXTURE
cat > "$OUT/mmap-memory.toml" <<TOML
[profiles.gate-smoke-baseline]
mode = "local"
executable = "$BASELINE_VENV/bin/mlx_lm.server"
argv = [
    "{port}", "$MODEL", "serving",
    "--host", "{host}",
    "--model", "$MODEL",
    "--max-kv-size", "8192",
    "--prefill-step-size", "2048",
    "--reasoning-effort", "medium",
]

[profiles.gate-smoke-candidate]
mode = "local"
executable = "$CANDIDATE_PYTHON"
argv = [
    "$OUT/fake-runtime.py", "{port}", "$MODEL", "serving",
    "--mmap-artifact", "$OUT/fake-weights.gguf",
    "--n-ctx", "8192",
    "--slots", "false",
    "--host", "{host}",
    "--model", "$MODEL",
    "--ctx-size", "8192",
    "--ubatch-size", "2048",
    "--reasoning-effort", "medium",
    "--spec-type", "none",
]
TOML
expect_admitted_run "mmap-loaded and anonymous runtime shapes carry the same scored quantity" \
    "$OUT/mmap-memory.toml" "$OUT/session-mmap-memory" "$OUT/mmap-memory.log" \
    "$((PORT + 25))"

python3 - "$OUT/session-mmap-memory" <<'MEMPY'
import json, pathlib, sys
session = pathlib.Path(sys.argv[1])
records = session / "records"
baseline = json.load(open(records / "python-mlx-lm.json"))
candidate = json.load(open(records / "mlx-swift.json"))
summary = json.load(open(session / "session.json"))
decision = json.load(open(session / "decision.json"))

def assert_peak(peak):
    assert peak["accounting"] == "mach-physical-footprint-plus-vmmap-resident-mapped-file-upper-bound", peak
    assert peak["scoreSemantics"] == "conservative-upper-bound", peak
    assert peak["status"] == "measured", peak
    assert peak["scoredBytes"] > 0, peak
    sample = peak["peakSample"]
    assert sample["machPhysicalFootprintBytes"] > 0, sample
    assert sample["residentMappedFileBytesUpperBound"] >= 0, sample
    composite = (
        sample["machPhysicalFootprintBytes"]
        + sample["residentMappedFileBytesUpperBound"]
    )
    assert sample["residentMemoryUpperBoundBytes"] == composite, sample
    assert peak["scoredBytes"] == composite, peak
    raw = peak["rawSamples"]
    assert raw, peak
    assert all(sample.get("sampledAtUnixSeconds", 0) > 0 for sample in raw), raw
    assert all(sample.get("machSampledAtUnixSeconds", 0) > 0 for sample in raw), raw
    assert all(sample.get("mappedFileSampledAtUnixSeconds", 0) > 0 for sample in raw), raw
    assert [sample["sampledAtUnixSeconds"] for sample in raw] == sorted(
        sample["sampledAtUnixSeconds"] for sample in raw
    ), raw
    # Every scored peak states the mapped-file observation cadence it was
    # produced under, and that stated cadence is not narrower than what the
    # series actually delivered. Tightening the contract constant back toward
    # the unreachable 125 ms claim fails right here.
    limit = peak["mappedFileObservationLimitSeconds"]
    assert limit > 0, peak
    assert "not observable" in peak["mappedFileObservabilityNote"], peak
    stamps = sorted({sample["mappedFileSampledAtUnixSeconds"] for sample in raw})
    # A sampled *window* always carries at least two mapped observations -- the
    # sampler's own coverage gate refuses one with fewer. `warmupMemory` is a
    # single synchronous point reading taken by `BenchmarkPass.recordWarmupMemory`
    # rather than a window, so it has one mapped observation and no gap to judge.
    # Neither warmup nor soak memory is read by `RuntimeBenchmark.decide`.
    if len(raw) > 1:
        assert len(stamps) >= 2, raw
    if len(stamps) >= 2:
        assert max(b - a for a, b in zip(stamps, stamps[1:])) <= limit, (limit, stamps)
    return composite

# Strict, on purpose. Revision 3 tolerated a coverage refusal here, which made a
# dimension that could never be scored indistinguishable from one refused for
# cause. The refusal branch has its own fixture now:
# `benchmark-memory-coverage-refusal-probe` drives the same production sampler
# class with the bound narrowed and requires it to refuse.
for record in (baseline, candidate):
    assert record["peakPhysicalFootprintBytes"] > 0, record["peakPhysicalFootprintBytes"]
    assert_peak(record["peakResidentMemory"])
    for scenario in record["scenarios"]:
        assert_peak(scenario["peakResidentMemory"])
        assert_peak(scenario["processResidentMemoryPeakSoFar"])
    notes = record["declaredAsymmetries"]
    assert any(note.startswith("baseline cache policy:") for note in notes), notes
    assert any(note.startswith("candidate cache policy:") for note in notes), notes
    assert any(note.startswith("cache comparability:") for note in notes), notes
for role in ("baseline", "candidate"):
    assert_peak(summary[role]["warmupMemory"])
    assert_peak(summary[role]["soakMemory"])
    assert summary[role]["memorySamplesReadFailed"] == 0, summary[role]
    assert summary[role]["memorySamplesMalformed"] == 0, summary[role]

assert candidate["peakResidentMemory"]["peakSample"]["residentMappedFileBytesUpperBound"] > 0, candidate["peakResidentMemory"]
memory_deltas = [
    delta for delta in decision["deltas"]
    if delta["metric"] == "peak_resident_memory_upper_bound_bytes"
]
assert memory_deltas, decision["deltas"]
assert not any(
    delta["metric"] == "peak_physical_footprint_bytes"
    for delta in decision["deltas"]
), decision["deltas"]

# Bind the generated evidence to the exact values the production decision
# consumed. Keeping the composite in the records while narrowing
# RuntimeMemoryPeak.validatedScoredBytes to Mach-only must fail here for every
# scenario delta and for the whole-process delta, on both runtimes.
baseline_scenarios = {item["name"]: item for item in baseline["scenarios"]}
candidate_scenarios = {item["name"]: item for item in candidate["scenarios"]}
seen = set()
for delta in memory_deltas:
    name = delta["scenario"]
    assert name not in seen, memory_deltas
    seen.add(name)
    if name == "process":
        baseline_peak = baseline["peakResidentMemory"]
        candidate_peak = candidate["peakResidentMemory"]
    else:
        assert name in baseline_scenarios, (name, baseline_scenarios)
        assert name in candidate_scenarios, (name, candidate_scenarios)
        baseline_peak = baseline_scenarios[name]["peakResidentMemory"]
        candidate_peak = candidate_scenarios[name]["peakResidentMemory"]
    assert delta["verdict"] not in ("unmeasured", "non-comparable"), delta
    assert delta["baseline"] == assert_peak(baseline_peak), delta
    assert delta["candidate"] == assert_peak(candidate_peak), delta
assert "process" in seen, seen
assert any(name != "process" for name in seen), seen
print("OK")
MEMPY
MEM_STATUS=$?
if [ "$MEM_STATUS" -eq 0 ] && [ -s "$OUT/session-mmap-memory/decision.json" ]; then
    pass "both records carry scored resident upper bounds or an explicit coverage refusal"
else
    fail "cross-runtime resident-memory accounting was absent, partial, or scored under the old name"
fi

# The three memory-instrumentation probes drive the production
# `BenchmarkFootprintSampler` class directly. Two are positives that fail when
# the dimension cannot be scored; the third is the negative that fails when the
# gate admits a series it must reject. Revision 3 shipped only the refusal, so
# an instrument that could never score anything looked identical to one
# refusing for cause.
if "$BINARY" benchmark-memory-sampler-probe >"$OUT/sub-cadence-memory.json"; then
    pass "the 20 Hz Mach series catches a 150 ms anonymous transient and the window scores"
else
    fail "the anonymous transient was missed, or its window could not be scored at all"
    tail -5 "$OUT/sub-cadence-memory.json"
fi

if "$BINARY" benchmark-mapped-file-sampler-probe >"$OUT/sub-cadence-mapped-file.json"; then
    pass "a sustained file-backed region reaches the scored mapped-file component"
else
    fail "the mapped-file component did not reach the score for a resident 256 MiB region"
    tail -5 "$OUT/sub-cadence-mapped-file.json"
fi

# The narrowing mutant, executed rather than described: the same production
# sampler on the same workload, with the mapped-file coverage bound narrowed to
# the 125 ms claim revisions 1-3 shipped, must refuse -- and the unnarrowed
# control on the same shape must score. A delete-only check would prove the gate
# exists; this one bounds the class it covers.
if "$BINARY" benchmark-memory-coverage-refusal-probe >"$OUT/coverage-refusal.json"; then
    pass "a narrowed mapped-file coverage bound refuses while the contract bound scores"
else
    fail "the narrowed coverage bound was admitted, or the control could not be scored"
    tail -5 "$OUT/coverage-refusal.json"
fi

# Teardown, which is where `BenchmarkRunCommand` finalises the whole-process
# series it then hands to the decision. This check was added because the strict
# assertions above caught it: stopping the 20 Hz Mach loop while the 5 s vmmap
# loop still had a read in flight left a hole one reader-cost wide at the tail of
# every process peak, so `peakResidentMemory` came back `partial` for the larger
# runtime in every session of the run.
if "$BINARY" benchmark-memory-stop-coverage-probe >"$OUT/stop-coverage.json"; then
    pass "the process peak finalised by stop() is still scoreable with a read in flight"
else
    fail "stopping the sampler mid-read left the process-wide series unscoreable"
    tail -5 "$OUT/stop-coverage.json"
fi

printf '\n%s\n' "----------------------------------------"
if [ "$FAILURES" -eq 0 ]; then
    printf 'BENCHMARK GATE SMOKE OK (0 failures)\n'
    exit 0
fi
printf 'BENCHMARK GATE SMOKE FAILED (%d failures)\n' "$FAILURES"
exit 1
