# TASK-260830-2hc5r2 review verdict — revision 4

## Verdict

Changes requested. Route to `to-dev` for revision 5.

Reviewed Change Request `CR-TASK-260830-2hc5r2-4`, revision 4, base
`5c9b4e4f7a88e1eb937b80851af522e4fa4b066f`, candidate tree
`1e59f8b7344124dae1c394921e2ead5da2a95af6`.

## F1 — observed argv still has a duplicate-flag bypass

Severity: blocking. Negative shape: **bypass path around the check**.

`BenchmarkRunCommand.drive` correctly records `ProcessObservation.arguments`,
but `RuntimeBenchmark.contextPolicy(derivedFrom:observing:)` returns the first
matching prefill/reasoning flag. Python `argparse`, including `mlx_lm.server`,
uses the last repeated value. The gate can therefore attest a prefill value the
serving process did not use.

I drove the composed production entry point with a candidate process that
re-execed before serving. Its kernel-observed argv contained:

```text
--prefill-step-size 2048 --prefill-step-size 999
```

The exact `benchmark-run` invocation exited `0` with `accepted=true`. Its
candidate record preserved both observed values but pinned:

```text
kv=76800;prefill-step=2048;reasoning=medium
```

A direct control over the same argument sequence reports
`argparse_effective_prefill=999`. Thus the new observed-argv source is in use,
but its interpretation is permissive and the comparison can still score unlike
runtime conditions as equal.

Production call path: `BenchmarkRunCommand.drive` ->
`RuntimeBenchmark.contextPolicy(derivedFrom:observing:)` ->
`RuntimeBenchmark.admit` -> `admitProvenance`.

Required rework: express prefill and reasoning launch observations as one
fail-closed normalization rule. Multiple occurrences, conflicting aliases, or
both reasoning spellings must not quietly choose a caller-favourable value.
Either prove and encode each runtime parser's effective semantics or, preferably
for this comparison gate, refuse ambiguous/repeated representations. Add a
production-entry negative using the exact duplicate argv above and narrowing
coverage for prefill and reasoning aliases.

## F2 — source fork commit is unsigned

Severity: blocking delivery-policy violation.

`ec9eea0af1d741cd4eb21c8766478a3e79dd44d6` is a real commit, the fork worktree
is clean at that OID, and the isolated install's `direct_url.json` pins both
`commit_id` and `requested_revision` to it. However, `git verify-commit` exits
`1` and `git show --format=%G?` reports `N`. The task's source change therefore
does not satisfy the repository-wide signed-delivery requirement.

Required rework: publish the external fork change as a cryptographically signed
human-authored commit, update the benchmark-only executable/profile and
immutable `direct_url.json` pin to the replacement OID, reinstall it, and rerun
the source-bound tests and live provenance checks.

## Earlier findings and acceptance evidence

- Omitted live `meta.n_ctx` with finite `--max-kv-size 76800`: production smoke
  exits `4`, reports `contextBoundNotHonoured` / no bound reported, and emits no
  accepted decision. No argv fallback reproduced.
- Re-exec from profile prefill `2048` to serving-process prefill `999`: production
  smoke exits `4` and names the observed `999` mismatch.
- Decoy baseline with unrelated `--python-bin`: production smoke exits `5`,
  cannot attribute the revision, and writes no decision.
- Mixed bounded/unbounded target/draft caches: source and production-path tests
  derive no active bound; the rerun of `tests.test_models tests.test_server`
  passed 115 tests with one skip.
- Qwen3.5 cache construction at the pinned source uses `RotatingKVCache(76800)`
  for full-attention layers and preserves `ArraysCache` for linear layers.
- The deployed default profile remains untouched; only benchmark profiles carry
  the 76,800 bound.
- The 73,139-token correctness proof was not rerun during this bounded reviewer
  session. I accepted the already-attached source-bound evidence: 73,111 cached
  tokens, correct three-system public answer, `finish_reason=stop`, and live
  `meta.n_ctx=76800` after the response.

## Reviewer validation

| Check | Result |
| --- | --- |
| `swift test -c release` | 290 tests / 24 suites passed |
| macOS arm64 Release `xcodebuild build` | `BUILD SUCCEEDED` |
| `benchmark-gate-smoke.sh` | zero failures; all required rev4 attacks passed |
| Python `tests.test_models tests.test_server` | 115 tests passed, one skipped |
| Exact duplicate-argv production attack | forbidden acceptance reproduced, exit `0` |
| Candidate `git diff --check` | passed |

The first Python rerun used the isolated immutable runtime environment without
its test-only `requests` dependency and exited `1` after 83 tests with an import
error. The subsequent existing test environment supplied the dependency and ran
the complete source suite above; the failed invocation is not counted as green.

## Logbook handoff

The duplicate-argv bypass is an important regression. This reviewer did not
modify tracked `LOGBOOK.md`, because doing so would move the immutable candidate
tree under review. Revision 5 must add the append-only logbook entry at source.
