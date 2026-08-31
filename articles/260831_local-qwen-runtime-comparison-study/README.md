# Local Qwen Runtime Comparison — Reproducibility Snapshot

This directory is the checksummed snapshot behind `ARTICLE.md`, the three-runtime comparison
study for `TASK-260828-15ftgj` (story `STORY-260828-2faxgm`). It retains the sealed run
records, attestations, session summaries and launch configurations of both measurement
campaigns, the pinned inputs both campaigns ran against, the accepted source reports, and one
analysis script that regenerates every figure the article cites.

The article is published in two places in this repository. `ARTICLE.md` here is the copy that
carries the artifacts; `.research/260831_local-qwen-runtime-comparison-study.md` is the same
document under the repository's dated research convention. They are byte-identical.

## The decision, in one line

**NO-GO on both candidates. Python `mlx-lm` remains the default local Qwen runtime.** MLX
Swift on a reproduced 1.151× 8k footprint blocker against a `≤ 1.10` bar; llama.cpp on an
`exit 4` inadmissibility, an absent memory axis, and a speed advantage of which only TTFT and
prefill are outside this host's noise floor. Reasoning, weighting and reopening conditions:
`ARTICLE.md` §7.

## What `reproduce.zsh` verifies

- every retained artifact matches `SHA256SUMS`;
- the two sealed llama.cpp-campaign records decompress and parse;
- `artifacts/analysis/recompute.py` regenerates every figure the article cites, from those
  records only, and the result is byte-identical to `artifacts/analysis/expected-figures.json`;
- the llama.cpp campaign's driver exit is **4** and **no `decision.json` exists** — the
  refusal the article's §4.1 rests on;
- the two passes' sealed intervals **do not overlap**;
- prompt-token parity is exactly 1.0000 on all six scenarios;
- **no memory window is scored on both sides** — the empty set the article's §4.4 rests on;
- **no positive break-even exists** on either decode-admissible scenario — the absence the
  article's §4.5 rests on;
- the MLX Swift campaign's decision is `accepted=false` with its one recorded blocker.

Any of those turning green-to-red means a claim in the article no longer holds.

## Is the verifier itself any good?

`reproduce.zsh` is a gate, and a gate that has only been seen to pass has not been shown to
work. `artifacts/analysis/mutant-campaign.md` records **13 narrowing mutants** — each one
falsifying a specific claim in the article, then regenerating `expected-figures.json` and
`SHA256SUMS` from its own mutated artifacts so that only the structural block can catch it.
All 13 are caught, each by the check that names the claim it attacks.

It also records **two survivors**, because a campaign that reports only its kills is an
advertisement. The load-bearing one is **B1**: replacing `recompute.py` with a stub that
copies the expectation passes cleanly. Verification here is a recomputation, and whoever can
edit the recomputation can make it agree with anything. What a green run establishes is that
the article's figures follow arithmetically from the sealed records **as shipped** — not that
a modified snapshot is honest.

`reproduce.zsh` does **not** re-run the benchmark. Re-running the measurement needs the model
weights, both runtimes and about an hour of an otherwise-idle M1 Max; `ARTICLE.md` §3.7 gives
the exact command.

## Tools

- `zsh`: run `./reproduce.zsh`.
- Python 3.10+: run `artifacts/analysis/recompute.py` (standard library only — no third-party
  packages, no network; 3.10 is the floor because it uses `itertools.pairwise`).
- `shasum`: verify `SHA256SUMS`.

## Run

```bash
./reproduce.zsh
```

A green run ends with `PASS: local Qwen runtime comparison study reproduced`.

## Layout

| Path | What it is |
|---|---|
| `ARTICLE.md` | the study |
| `SHA256SUMS` | checksums for every file in this directory except itself |
| `reproduce.zsh` | the verification entry point |
| `artifacts/llamacpp-pair/records/*.json.gz` | the two sealed run records of the 2026-08-30 campaign, gzipped (96 MB raw; almost all of it is the raw memory sample series and the sealed transcripts) |
| `artifacts/llamacpp-pair/session.json` | lifecycle, soak and memory-window summaries for both passes |
| `artifacts/llamacpp-pair/attest/` | what the gate observed of each process, including the `notReported` generation-configuration terms behind the exit-4 refusal |
| `artifacts/llamacpp-pair/run-rev4.sh`, `benchmark.toml` | the exact command and the exact launch argv of both runtimes |
| `artifacts/llamacpp-pair/run-rev4.exit`, `run-rev4-interval.txt`, `run-rev4-sweeps.log`, `run-rev4.log` | driver exit 4, the timed interval, the host sweeps that show one model at a time, and the driver's own output |
| `artifacts/llamacpp-pair/probe-llamacpp-*.json` | the live-surface probe behind §4.1 |
| `artifacts/llamacpp-pair/memory-coverage-analysis.txt`, `vmmap-cost-derivation.txt` | per-window gap distributions and the `vmmap` cost derivation behind §4.4 |
| `artifacts/llamacpp-pair/tables.txt` | the campaign's own rendered number set |
| `artifacts/llamacpp-pair/logs/` | both runtimes' stdout for the whole campaign |
| `artifacts/mlx-swift-pair/` | the same shape for the 2026-08-28 campaign, plus its `decision.json` — the one arm that was admitted and scored |
| `artifacts/pinned-inputs/` | the prompt suite, thresholds and model-equivalence declaration both campaigns ran against |
| `artifacts/analysis/recompute.py` | regenerates every cited figure from the records; standard library only |
| `artifacts/analysis/expected-figures.json` | the checked-in expected output |
| `artifacts/analysis/mutant-campaign.md` | 13 narrowing mutants against `reproduce.zsh`, all caught, plus two reported survivors |
| `artifacts/source-documents/` | the accepted producer reports and review verdicts of both arms |

The complete raw evidence archives, spawn logs and change-request patches remain board
outcome resources under `TASK-260827-2v13w8` and `TASK-260829-3k4qrc`. This snapshot keeps
what a clean-checkout reader needs to audit every number in the article.
