#!/usr/bin/env python3
"""Recompute every figure cited in ARTICLE.md from the raw run records.

Reads only the sealed records, session summaries and decision documents that
sit beside it under ``artifacts/``. Writes one JSON object to stdout, or to
``--output``. ``reproduce.zsh`` diffs that object against the checked-in
``expected-figures.json``; any number in the article that this script cannot
reproduce is a number the article may not carry.

No network, no runtime, no measurement. This is arithmetic over sealed files.
"""

from __future__ import annotations

import argparse
import gzip
import itertools
import json
import statistics
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent  # artifacts/
LLAMA = ROOT / "llamacpp-pair"
SWIFT = ROOT / "mlx-swift-pair"

# The two coverage bounds the shipped sampler enforces, quoted from the
# records' own `mappedFileObservationLimitSeconds` and the Mach bound the
# contract documents. Both are inputs to the refusal arithmetic, not results.
MAPPED_BOUND_SECONDS = 7.0
MACH_BOUND_SECONDS = 0.125


def load_json(path: Path) -> dict:
    if path.suffix == ".gz":
        with gzip.open(path, "rt", encoding="utf-8") as handle:
            return json.load(handle)
    return json.loads(path.read_text(encoding="utf-8"))


def r6(value):
    return None if value is None else round(float(value), 6)


def gaps(stamps: list[float]) -> list[float]:
    ordered = sorted(stamps)
    return [b - a for a, b in itertools.pairwise(ordered)]


def window_summary(window: dict) -> dict:
    """Status, score and measured sampling coverage of one memory window."""
    samples = window.get("rawSamples") or []
    mach = [
        s["machSampledAtUnixSeconds"]
        for s in samples
        if "machSampledAtUnixSeconds" in s
    ]
    mapped = sorted(
        {
            s["mappedFileSampledAtUnixSeconds"]
            for s in samples
            if "mappedFileSampledAtUnixSeconds" in s
        }
    )
    mach_gaps = gaps(mach)
    mapped_gaps = gaps(mapped)
    out = {
        "status": window.get("status"),
        "issues": window.get("issues", []),
        "scoredBytes": window.get("scoredBytes"),
        "readFailureCount": window.get("readFailureCount"),
        "successfulSampleCount": window.get("successfulSampleCount"),
        "machStamps": len(mach),
        "mappedStamps": len(mapped),
        "machGapsOverBound": sum(1 for g in mach_gaps if g > MACH_BOUND_SECONDS),
        "machGapCount": len(mach_gaps),
        "mappedGapsOverBound": sum(1 for g in mapped_gaps if g > MAPPED_BOUND_SECONDS),
        "mappedGapCount": len(mapped_gaps),
    }
    if mapped_gaps:
        out["mappedGapMin"] = r6(min(mapped_gaps))
        out["mappedGapMedian"] = r6(statistics.median(mapped_gaps))
        out["mappedGapMax"] = r6(max(mapped_gaps))
    if mach_gaps:
        out["machGapMax"] = r6(max(mach_gaps))
    return out


def scenario_rows(record: dict) -> dict:
    rows = {}
    for scenario in record["scenarios"]:
        rows[scenario["name"]] = {
            "succeeded": scenario.get("succeeded"),
            "promptTokens": scenario.get("promptTokens"),
            "completionTokens": scenario.get("completionTokens"),
            "ttftSeconds": r6(scenario.get("timeToFirstTokenSeconds")),
            "prefillTokensPerSecond": r6(scenario.get("prefillTokensPerSecond")),
            "decodeTokensPerSecond": r6(scenario.get("decodeTokensPerSecond")),
            "wallClockSeconds": r6(scenario.get("wallClockSeconds")),
            "hostLoadAverageMax": r6(scenario.get("hostLoadAverageMax")),
            "cacheReuse": scenario.get("cacheReuse"),
            "scenarioMemory": window_summary(scenario["peakResidentMemory"])
            if "peakResidentMemory" in scenario
            else None,
            "processPeakSoFarMemory": window_summary(
                scenario["processResidentMemoryPeakSoFar"]
            )
            if "processResidentMemoryPeakSoFar" in scenario
            else None,
            "peakPhysicalFootprintBytes": scenario.get("peakPhysicalFootprintBytes"),
        }
    return rows


def break_even(ttft_base, ttft_cand, rate_base, rate_cand):
    """L* = (TTFT_b - TTFT_c) / (1/r_c - 1/r_b), in output tokens.

    A crossover exists only when one runtime leads TTFT and the other leads
    decode. When one leads both, L* comes out non-positive and there is no
    crossover at any positive response length; the sign is reported rather
    than the magnitude being presented as a length.
    """
    if None in (ttft_base, ttft_cand, rate_base, rate_cand):
        return {"exists": None, "reason": "a required term is unmeasured"}
    numerator = ttft_base - ttft_cand
    denominator = (1.0 / rate_cand) - (1.0 / rate_base)
    if denominator == 0:
        return {
            "exists": False,
            "reason": "identical decode rates; the curves never cross",
        }
    crossover = numerator / denominator
    return {
        "deltaTtftSeconds": r6(numerator),
        "deltaInverseRateSecondsPerToken": r6(denominator),
        "crossoverOutputTokens": r6(crossover),
        "exists": crossover > 0,
        "reason": (
            "one runtime leads both TTFT and decode, so the candidate curve is "
            "below the baseline curve at every positive response length"
            if crossover <= 0
            else "the leaders on TTFT and decode differ, so the curves cross"
        ),
    }


def response_time_table(ttft, rate, lengths):
    return {str(length): r6(ttft + length / rate) for length in lengths}


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--output", type=Path)
    args = parser.parse_args()

    base = load_json(LLAMA / "records" / "python-mlx-lm.json.gz")
    cand = load_json(LLAMA / "records" / "llamacpp.json.gz")
    session = load_json(LLAMA / "session.json")

    swift_base = load_json(SWIFT / "records" / "python-mlx-lm.json")
    swift_cand = load_json(SWIFT / "records" / "mlx-swift.json")
    swift_decision = load_json(SWIFT / "decision.json")

    base_rows = scenario_rows(base)
    cand_rows = scenario_rows(cand)

    llama_interval = {
        "baselineStart": base["startedAtUnixSeconds"],
        "baselineFinish": base["finishedAtUnixSeconds"],
        "baselineDurationSeconds": r6(
            base["finishedAtUnixSeconds"] - base["startedAtUnixSeconds"]
        ),
        "candidateStart": cand["startedAtUnixSeconds"],
        "candidateFinish": cand["finishedAtUnixSeconds"],
        "candidateDurationSeconds": r6(
            cand["finishedAtUnixSeconds"] - cand["startedAtUnixSeconds"]
        ),
        "separationSeconds": r6(
            cand["startedAtUnixSeconds"] - base["finishedAtUnixSeconds"]
        ),
        "overlapSeconds": r6(
            base["finishedAtUnixSeconds"] - cand["startedAtUnixSeconds"]
        ),
    }

    ratios = {}
    for name in base_rows:
        b, c = base_rows[name], cand_rows[name]
        entry = {}
        for key in (
            "ttftSeconds",
            "prefillTokensPerSecond",
            "decodeTokensPerSecond",
            "wallClockSeconds",
        ):
            if b[key] is not None and c[key] is not None and b[key] != 0:
                entry[key] = r6(c[key] / b[key])
        if b["promptTokens"] and c["promptTokens"]:
            entry["promptTokenSkew"] = r6(c["promptTokens"] / b["promptTokens"])
        ratios[name] = entry

    # Decode is only meaningfully measured where the completion budget is not a
    # rounding error. `context_75k` spends 16 completion tokens after a
    # ~950-1280 s prefill, so it is excluded from every decode claim by name.
    decode_scenarios = [
        name
        for name in base_rows
        if base_rows[name]["decodeTokensPerSecond"] is not None
        and cand_rows[name]["decodeTokensPerSecond"] is not None
        and base_rows[name]["completionTokens"] >= 32
    ]

    crossovers = {
        name: break_even(
            base_rows[name]["ttftSeconds"],
            cand_rows[name]["ttftSeconds"],
            base_rows[name]["decodeTokensPerSecond"],
            cand_rows[name]["decodeTokensPerSecond"],
        )
        for name in decode_scenarios
    }

    lengths = [16, 64, 256, 1024]
    response_times = {
        name: {
            "baseline": response_time_table(
                base_rows[name]["ttftSeconds"],
                base_rows[name]["decodeTokensPerSecond"],
                lengths,
            ),
            "candidate": response_time_table(
                cand_rows[name]["ttftSeconds"],
                cand_rows[name]["decodeTokensPerSecond"],
                lengths,
            ),
        }
        for name in decode_scenarios
    }

    gated_memory = {
        "baseline": {
            **{f"scenario:{n}": base_rows[n]["scenarioMemory"] for n in base_rows},
            **{
                f"processPeakSoFar:{n}": base_rows[n]["processPeakSoFarMemory"]
                for n in base_rows
            },
            "wholePassProcessPeak": window_summary(base["peakResidentMemory"]),
        },
        "candidate": {
            **{f"scenario:{n}": cand_rows[n]["scenarioMemory"] for n in cand_rows},
            **{
                f"processPeakSoFar:{n}": cand_rows[n]["processPeakSoFarMemory"]
                for n in cand_rows
            },
            "wholePassProcessPeak": window_summary(cand["peakResidentMemory"]),
        },
    }

    # The four windows BenchmarkPass builds directly, which never reach the
    # coverage gate and publish `measured` with a score regardless of coverage.
    ungated_memory = {
        f"{side}:{window}": window_summary(session[side][window])
        for side in ("baseline", "candidate")
        for window in ("warmupMemory", "soakMemory")
    }

    scored_both_sides = [
        key
        for key in gated_memory["baseline"]
        if gated_memory["baseline"][key]
        and gated_memory["candidate"].get(key)
        and gated_memory["baseline"][key]["status"] == "measured"
        and gated_memory["candidate"][key]["status"] == "measured"
    ]

    swift_base_rows = scenario_rows(swift_base)
    swift_cand_rows = scenario_rows(swift_cand)
    swift_ratios = {}
    for name in swift_base_rows:
        b, c = swift_base_rows[name], swift_cand_rows[name]
        entry = {}
        for key in (
            "ttftSeconds",
            "prefillTokensPerSecond",
            "decodeTokensPerSecond",
            "wallClockSeconds",
            "peakPhysicalFootprintBytes",
        ):
            if b[key] is not None and c[key] is not None and b[key] != 0:
                entry[key] = r6(c[key] / b[key])
        swift_ratios[name] = entry

    # The same incumbent, two campaigns, two configurations. This ratio is why
    # the two candidate arms may not be tabled against each other.
    incumbent_drift = {
        name: {
            key: r6(base_rows[name][key] / swift_base_rows[name][key])
            for key in (
                "ttftSeconds",
                "prefillTokensPerSecond",
                "decodeTokensPerSecond",
            )
            if base_rows[name][key] is not None
            and swift_base_rows[name][key] not in (None, 0)
        }
        for name in base_rows
        if name in swift_base_rows
    }

    figures = {
        "llamacppPair": {
            "driverExitCode": int(
                (LLAMA / "run-rev4.exit").read_text().strip().split("=")[1]
            ),
            "decisionWritten": (LLAMA / "decision.json").exists(),
            "pins": {"baseline": base["pins"], "candidate": cand["pins"]},
            "revisions": {
                "baseline": base["revisions"],
                "candidate": cand["revisions"],
            },
            "declaredAsymmetries": {
                "baseline": base["declaredAsymmetries"],
                "candidate": cand["declaredAsymmetries"],
            },
            "interval": llama_interval,
            "baselineScenarios": base_rows,
            "candidateScenarios": cand_rows,
            "candidateOverBaselineRatios": ratios,
            "decodeAdmissibleScenarios": sorted(decode_scenarios),
            "breakEven": crossovers,
            "totalResponseSecondsByOutputTokens": response_times,
            "gatedMemoryWindows": gated_memory,
            "ungatedMemoryWindows": ungated_memory,
            "memoryWindowsScoredOnBothSides": scored_both_sides,
            "sessionSummary": {
                side: {
                    "hostLoadAverageMax": r6(session[side]["hostLoadAverageMax"]),
                    "memorySamplesSuccessful": session[side]["memorySamplesSuccessful"],
                    "memorySamplesReadFailed": session[side]["memorySamplesReadFailed"],
                    "memorySamplesMalformed": session[side]["memorySamplesMalformed"],
                    "lifecycle": {
                        k: r6(v) for k, v in session[side]["lifecycle"].items()
                    },
                    "soak": {
                        k: (r6(v) if isinstance(v, float) else v)
                        for k, v in session[side]["soak"].items()
                    },
                }
                for side in ("baseline", "candidate")
            },
        },
        "mlxSwiftPair": {
            "accepted": swift_decision["accepted"],
            "blockers": swift_decision["blockers"],
            "declaredAsymmetries": swift_decision["declaredAsymmetries"],
            "scoredDeltas": swift_decision["deltas"],
            "pins": {"baseline": swift_base["pins"], "candidate": swift_cand["pins"]},
            "revisions": {
                "baseline": swift_base["revisions"],
                "candidate": swift_cand["revisions"],
            },
            "interval": {
                "baselineStart": swift_base["startedAtUnixSeconds"],
                "baselineFinish": swift_base["finishedAtUnixSeconds"],
                "candidateStart": swift_cand["startedAtUnixSeconds"],
                "candidateFinish": swift_cand["finishedAtUnixSeconds"],
                "overlapSeconds": r6(
                    swift_base["finishedAtUnixSeconds"]
                    - swift_cand["startedAtUnixSeconds"]
                ),
            },
            "baselineScenarios": swift_base_rows,
            "candidateScenarios": swift_cand_rows,
            "candidateOverBaselineRatios": swift_ratios,
            "wholeProcessPeakPhysicalFootprintBytes": {
                "baseline": swift_base["peakPhysicalFootprintBytes"],
                "candidate": swift_cand["peakPhysicalFootprintBytes"],
            },
        },
        "crossCampaign": {
            "incumbentLlamacppEraOverSwiftEraRatios": incumbent_drift,
            "comparable": False,
            "reason": (
                "different incumbent build and KV pin (kv=unbounded at 9150698 versus "
                "kv=76800 at the 45a472f fork), a different memory instrument "
                "(peakPhysicalFootprintBytes versus the mapped-file-inclusive resident "
                "upper bound), and two campaigns two days apart on a host that was not "
                "idle-locked"
            ),
        },
    }

    text = json.dumps(figures, indent=2, sort_keys=True) + "\n"
    if args.output:
        args.output.write_text(text, encoding="utf-8")
    else:
        print(text, end="")


if __name__ == "__main__":
    main()
