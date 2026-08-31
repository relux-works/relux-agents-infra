#!/bin/zsh
# Verify the local Qwen runtime comparison study.
#
# This does NOT re-run the benchmark. It verifies that every figure ARTICLE.md
# cites is still derivable from the sealed records in artifacts/, and that the
# structural claims the decision rests on -- the exit-4 refusal, the absent
# memory comparison, the absent break-even -- still hold against those records.
set -euo pipefail

ROOT="${0:A:h}"
cd "$ROOT"

REPRO_TEMP="$(mktemp -d "${TMPDIR:-/tmp}/qwen-runtime-study.XXXXXX")"
trap 'rm -rf "$REPRO_TEMP"' EXIT HUP INT TERM

print 'Verifying retained artifact checksums...'
shasum -a 256 -c SHA256SUMS

print 'Checking the sealed records decompress and parse...'
for record in artifacts/llamacpp-pair/records/*.json.gz; do
  gzip -dc "$record" | python3 -c 'import json,sys; json.load(sys.stdin)'
  print "  ok: $record"
done

print 'Regenerating every cited figure from the sealed records...'
python3 artifacts/analysis/recompute.py --output "$REPRO_TEMP/figures.json"
if ! cmp -s "$REPRO_TEMP/figures.json" artifacts/analysis/expected-figures.json; then
  print -u2 'FAIL: recomputed figures differ from the checked-in expectation'
  diff -u artifacts/analysis/expected-figures.json "$REPRO_TEMP/figures.json" | head -80 >&2
  exit 1
fi
print '  ok: recomputed figures are byte-identical to expected-figures.json'

print 'Checking the structural claims the decision rests on...'
python3 - "$REPRO_TEMP/figures.json" <<'PY'
import json
import sys

figures = json.load(open(sys.argv[1]))
llama = figures["llamacppPair"]
swift = figures["mlxSwiftPair"]
failures = []


def check(condition, message):
    print(("  ok: " if condition else "  FAIL: ") + message)
    if not condition:
        failures.append(message)


# ARTICLE.md 4.1 -- the gate refused the pair and wrote no decision.
check(llama["driverExitCode"] == 4, "llama.cpp campaign driver exit is 4 (inadmissible)")
check(llama["decisionWritten"] is False, "no decision.json was written for the llama.cpp pair")
check(
    llama["pins"]["candidate"]["contextPolicy"]
    == "kv=76800;prefill-step=not-reported;reasoning=not-reported",
    "candidate contextPolicy carries two not-reported terms",
)
check(
    llama["pins"]["baseline"]["contextPolicy"] == "kv=76800;prefill-step=2048;reasoning=medium",
    "baseline contextPolicy reports all three terms",
)
check(
    llama["pins"]["baseline"]["contextPolicy"].split(";")[0]
    == llama["pins"]["candidate"]["contextPolicy"].split(";")[0],
    "the KV term agrees on both sides (STORY-260830-2vrhg1's fix held)",
)

# ARTICLE.md 4.3.1 -- sequential passes, no overlap.
check(llama["interval"]["overlapSeconds"] < 0, "the two passes' sealed intervals do not overlap")

# ARTICLE.md 4.3.2 -- exact prompt-token parity on all six scenarios.
skews = {
    name: row.get("promptTokenSkew")
    for name, row in llama["candidateOverBaselineRatios"].items()
}
check(len(skews) == 6, "all six scenarios carry a prompt-token skew")
check(all(value == 1.0 for value in skews.values()), "prompt-token skew is exactly 1.0 on all six")

# ARTICLE.md 4.3.4 -- capacity, and speculation off on both sides.
check(
    llama["baselineScenarios"]["context_75k"]["promptTokens"] == 73016
    and llama["candidateScenarios"]["context_75k"]["promptTokens"] == 73016
    and llama["baselineScenarios"]["context_75k"]["succeeded"]
    and llama["candidateScenarios"]["context_75k"]["succeeded"],
    "both runtimes served the 73,016-token capacity probe",
)
check(
    llama["pins"]["baseline"]["speculation"] == "off"
    and llama["pins"]["candidate"]["speculation"] == "off",
    "speculation is off in both pins",
)

# ARTICLE.md 4.4 -- the memory axis produced no comparison at all.
check(
    llama["memoryWindowsScoredOnBothSides"] == [],
    "no memory window is scored on both sides (the memory axis produced no comparison)",
)

# ARTICLE.md 4.4.1 -- the four windows that publish a score without facing the gate.
ungated = llama["ungatedMemoryWindows"]
check(len(ungated) == 4, "four memory windows bypass the coverage gate")
check(
    all(window["status"] == "measured" and window["scoredBytes"] for window in ungated.values()),
    "all four publish `measured` with a score",
)
check(
    all(
        window["mappedGapsOverBound"] == window["mappedGapCount"] == 19
        for name, window in ungated.items()
        if name.endswith("soakMemory")
    ),
    "both soakMemory windows are 19/19 outside the mapped bound",
)

# ARTICLE.md 4.5 -- no break-even exists in the measured direction.
check(
    sorted(llama["breakEven"]) == ["long_prompt_8k", "short_prompt"],
    "break-even is computed only on the two decode-admissible scenarios",
)
check(
    all(entry["exists"] is False for entry in llama["breakEven"].values()),
    "no positive crossover exists on either decode-admissible scenario",
)
check(
    all(entry["crossoverOutputTokens"] < 0 for entry in llama["breakEven"].values()),
    "the crossover length is negative on both, so the curves never cross",
)

# ARTICLE.md 4.3.3 -- context_75k is excluded from decode by its own completion budget.
check(
    llama["baselineScenarios"]["context_75k"]["completionTokens"] == 16
    and llama["candidateScenarios"]["context_75k"]["completionTokens"] == 16
    and "context_75k" not in llama["decodeAdmissibleScenarios"],
    "context_75k spends 16 completion tokens and is excluded from every decode claim",
)

# ARTICLE.md 4.3.5 -- the one-sided cache on two scenarios.
one_sided = [
    name
    for name in llama["baselineScenarios"]
    if llama["baselineScenarios"][name]["cacheReuse"]["state"]
    != llama["candidateScenarios"][name]["cacheReuse"]["state"]
]
check(
    sorted(one_sided) == ["multiturn_prefix_reuse", "stability_soak"],
    "exactly two scenarios are non-comparable on one-sided cache telemetry",
)
check(
    all(
        llama["baselineScenarios"][name]["cacheReuse"]["state"] == "miss"
        for name in llama["baselineScenarios"]
    ),
    "the incumbent's configured prompt cache did not fire on any scenario",
)

# ARTICLE.md 4.2 -- the MLX Swift arm was scored and rejected on one blocker.
check(swift["accepted"] is False, "the MLX Swift pair was scored and rejected")
check(len(swift["blockers"]) == 1, "the MLX Swift rejection rests on exactly one blocker")
check(
    "long_prompt_8k/peak_physical_footprint_bytes" in swift["blockers"][0],
    "that blocker is the 8k peak footprint ratio",
)
outside = [d for d in swift["scoredDeltas"] if d["verdict"] == "outside"]
check(len(outside) == 1 and round(outside[0]["ratio"], 4) == 1.1512, "the ratio is 1.1512x")

# ARTICLE.md 4.6 -- the two arms may not be compared with each other.
check(
    figures["crossCampaign"]["comparable"] is False,
    "the two candidate arms are marked non-comparable with each other",
)

if failures:
    print(f"\n{len(failures)} structural check(s) failed")
    raise SystemExit(1)
PY

print 'PASS: local Qwen runtime comparison study reproduced'
