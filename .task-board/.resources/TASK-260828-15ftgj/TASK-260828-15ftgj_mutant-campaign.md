# Mutant campaign against this snapshot's own verifier

`reproduce.zsh` refuses, validates and attests: it is a gate, and a gate that has only been
seen to pass has not been shown to work. This campaign attacks it.

**Method.** Each mutant is applied to a full copy of this directory. The mutant then does
what a careless or dishonest editor would do next: it **regenerates
`artifacts/analysis/expected-figures.json` from its own mutated artifacts and re-checksums
`SHA256SUMS`**. That deliberately defeats both the checksum layer and the expected-figures
diff, so only `reproduce.zsh`'s structural block can catch it. Every mutant is applied to a
copy and reverted by deletion; the published snapshot is untouched.

**Production call site.** The block that catches these is the inline `python3 - "$REPRO_TEMP/figures.json"`
heredoc in `reproduce.zsh`, reached unconditionally on every green path after the checksum
verification and the recomputation. There is no other caller and no flag that skips it.

## Results — 13 narrowing mutants, all caught

| ID | Mutation | Claim it attacks | Outcome |
|---|---|---|---|
| M1 | `run-rev4.exit` rewritten `4` → `0` | §4.1 the gate refused the pair | **caught** — `driver exit is 4 (inadmissible)` |
| M2 | a `decision.json` planted beside the refused pair | §4.1 no decision was written | **caught** — `no decision.json was written` |
| M3 | candidate `contextPolicy` narrowed to `prefill-step=2048;reasoning=medium` | §4.1 two terms are `not-reported` | **caught** — `candidate contextPolicy carries two not-reported terms` |
| M4 | baseline `short_prompt` memory window forced to `measured` with a score | §4.4 no window scored on both sides | **caught** — `no memory window is scored on both sides` |
| M5 | candidate decode halved, so llama.cpp leads TTFT and loses decode | §4.5 no break-even exists | **caught** — `no positive crossover exists`, `the crossover length is negative on both` |
| M6 | candidate `long_prompt_8k` prompt tokens 7,784 → 7,900 | §4.3.2 exact parity | **caught** — `prompt-token skew is exactly 1.0 on all six` |
| M7 | candidate pass start moved 600 s earlier, overlapping the baseline | §4.3.1 sequential passes | **caught** — `the two passes' sealed intervals do not overlap` |
| M8 | MLX Swift `decision.json` flipped to `accepted=true`, blockers emptied | §4.2 scored and rejected on one blocker | **caught** — `the MLX Swift pair was scored and rejected` |
| M9 | baseline `multiturn_prefix_reuse` cache state faked to `hit` | §4.3.5 the incumbent's cache never fired | **caught** — `exactly two scenarios are non-comparable`, `the incumbent's prompt cache did not fire` |
| M10 | `context_75k` completion budget inflated 16 → 256 on both sides | §4.3.3 that scenario is excluded from decode | **caught** — `context_75k spends 16 completion tokens and is excluded` |
| M11 | candidate `context_75k` marked `succeeded: false` | §4.3.4 both met the capacity probe | **caught** — `both runtimes served the 73,016-token capacity probe` |
| M12 | candidate `speculation` pin `off` → `draft-mtp` | §3.2 / §5 T5 speculation off on both | **caught** — `speculation is off in both pins` |
| M13 | ungated `soakMemory` timestamps rewritten to a compliant 0.05 s cadence | §4.4.1 those windows are 19/19 outside both bounds | **caught** — `both soakMemory windows are 19/19 outside the mapped bound` |

Each mutant is caught by the check that names the specific claim it attacks, not by a
generic tripwire. Run without the expectation-regeneration step, all 13 are additionally
caught one layer earlier by the `expected-figures.json` diff.

## Survivors — reported, not hidden

| ID | Class | Mutation | Outcome |
|---|---|---|---|
| **B1** | bypass | `recompute.py` replaced by a stub that copies `expected-figures.json` to the output path | **survived, exit 0** |
| **D1** | delete-only control | the structural block removed from `reproduce.zsh`, artifacts untouched | **survived, exit 0** |

**B1 is the honest statement of this verifier's boundary.** Verification here is a
*recomputation*, and whoever can edit the recomputation can make it agree with anything. What
`reproduce.zsh` establishes is that the article's figures follow arithmetically from the
sealed records **as shipped**; it does not establish that a modified snapshot is honest. That
is the same class boundary the MLX Swift arm reported for its own gate: the ordinary-editor
forgery class is closed, the modified-tooling class is not, and no further hardening was
attempted because every additional clause would raise the cost of the same attack without
changing its class.

**D1 is a control, and its survival is the expected result rather than a finding.** Deleting
a gate proves the gate exists; it says nothing about the class the gate bounds. That is what
M1–M13 are for — each narrows a specific claim rather than removing the check.

## Reproducing the campaign

The campaign harness is not shipped in this snapshot, because it exists to attack the
snapshot rather than to describe it. It is a loop that, per mutant: copies this directory,
applies the mutation above, runs `python3 artifacts/analysis/recompute.py --output
artifacts/analysis/expected-figures.json`, regenerates `SHA256SUMS` with
`find . -type f ! -name SHA256SUMS -print0 | sort -z | xargs -0 shasum -a 256`, and records
the exit status of `zsh ./reproduce.zsh`. Each mutation is one line and is stated in full in
the table above.
