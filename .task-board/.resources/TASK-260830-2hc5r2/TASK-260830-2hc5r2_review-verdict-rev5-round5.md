# TASK-260830-2hc5r2 — revision 5 / Round 5 review verdict

## Verdict

**Changes requested (`to-dev`).** Candidate tree
`4d8de4c986123907cc4878eb1422fc397923024d` closes the exact repeated-full-flag
attack from revision 4, but the production gate still admits a Python argv that
the observed runtime parses to a different effective prefill value.

## Finding F8 — argparse long-option abbreviation bypass

Negative shape: **bypass path around the check**.

Production call sites:

- `BenchmarkRunCommand.drive` obtains `ProcessObservation.arguments` from
  `KERN_PROCARGS2` and writes it as `RunRecord.provenance.launchArgv`.
- `RuntimeBenchmark.contextPolicy(for:derivedFrom:observing:)` calls
  `observedValue` to decode that observed argv.
- `RuntimeBenchmark.admitProvenance` independently re-derives the policy from
  the recorded argv and live context-window attestation.

The new registry models `python-mlx-lm` as argparse last-wins, but recognizes
only exact `--prefill-step-size` tokens. Python `argparse.ArgumentParser` has
`allow_abbrev=true` by default. Therefore this observed argv is effective
prefill `999` in the pinned signed `mlx_lm.server`, while the decoder ignores
the abbreviated second occurrence and records `2048`:

```text
--prefill-step-size 2048 --prefill-step-siz 999
```

Independent reproduction against the installed signed server's actual parser:

```text
actual_mlx_lm_prefill_step_size=999
```

Production reproduction used the shipped Release binary and real
`benchmark-run` entry point with that exact observed baseline argv. It exited
`0`, emitted `"accepted" : true`, and persisted:

```json
{
  "runtime": "python-mlx-lm",
  "contextPolicy": "kv=76800;prefill-step=2048;reasoning=medium",
  "launchArgv": [
    "...",
    "--prefill-step-size", "2048",
    "--prefill-step-siz", "999",
    "--chat-template-args", "{\"reasoning_effort\":\"medium\"}"
  ]
}
```

Expected result is an inadmissible comparison (exit `4`) with no accepted
decision, because the effective Python prefill `999` does not match the Swift
prefill `2048`.

Reviewer evidence:

- `.temp/review-round5/argparse-abbrev-bypass-04.log`
- `.temp/review-round5/argparse-abbrev-bypass.toml`
- `.temp/review-round5/session-argparse-abbrev-04/records/python-mlx-lm.json`
- signed-parser witness was run through the shebang interpreter behind
  `/Users/alexis/.local/bin/mlx_lm-kv76800-0a0452a.server`

## Required rework

1. Close argparse long-option abbreviations once for every context-policy flag,
   including prefill, context bound, and reasoning. A conservative decoder may
   classify any recognized unique/potential long-option prefix as unresolved;
   alternatively the pinned fork may disable argparse abbreviations and the
   observation contract must prove that parser setting. Do not patch only the
   one spelling above.
2. Add a production `benchmark-run` negative using
   `--prefill-step-size 2048 --prefill-step-siz 999`; require exit `4`, effective
   `999` or explicit unresolved in the refusal, and no `accepted=true`.
3. Add unit narrowing for abbreviated prefill, context-bound, and reasoning
   options, including prefixes that are ambiguous with another known option.
4. Append this regression and closure to `LOGBOOK.md` during producer rework.
5. If the fork commit changes, publish and pin a new signed immutable commit in
   the benchmark-only executable/profile/direct-url chain; keep the deployed
   default profile unchanged.

## Checks that passed

- Exact revision-4 duplicate attack
  `--prefill-step-size 2048 --prefill-step-size 999`: production exit `4`,
  effective `999` recorded.
- Focused Swift gate suite rerun by this reviewer: 81 tests / 5 suites passed.
- Full production smoke rerun by this reviewer: 0 failures, including mixed
  live-KV absence, kernel-argv rewrite, decoy `--python-bin`, malformed/absent
  evidence, replay, and duplicate-full-flag attacks.
- Fork commit `0a0452a9ca64d5b8ee3786fb23d3f828417f9514`: `git verify-commit` reports a
  good SSH/ECDSA signature for `alexis@relux.works`; remote task branch points
  to the same OID; its tree is byte-identical to `ec9eea0`.
- `qwen3_5.make_cache(max_kv_size:)` constructs bounded
  `RotatingKVCache(max_size:..., keep:4)` for attention layers. Batch and
  seeded/single server paths pass the configured bound. Mixed bounded/unbounded
  or differently bounded cache families cause `_cache_max_kv_size` to return
  no attested bound.
- Benchmark-only executable and installed `direct_url.json` both pin `0a0452a`.
  The candidate changes only the benchmark example; it does not declare or
  modify `profiles.qwen-local`.
- Attached live signed-server evidence shows `finish_reason=stop`, exact
  `SIGNED_BASELINE_OK`, kernel-observed `--max-kv-size 76800`, and post-cache
  `/v1/models` `meta.n_ctx=76800`.
- The 73,139-token correctness run was not rerun in this bounded reviewer turn.
  It is accepted as already-attached evidence from the byte-identical source
  tree: `finish_reason=stop`, 73,111 cached tokens, exact three-system answer,
  and post-generation `n_ctx=76800`.

The task-owned decoy listener left by the smoke's known early-abort cleanup
defect was terminated after evidence capture; port `19391` was clear.
