# TASK-260830-2hc5r2 — revision 3 review verdict

## Verdict

**Changes requested.** Route Change Request `CR-TASK-260830-2hc5r2-3`
revision 3 to `to-dev`.

Reviewed base `5c9b4e4f7a88e1eb937b80851af522e4fa4b066f`, candidate tree
`46b7c44340b553654cce849c71d45f4c7defa095`, and supplied patch SHA-256
`81a1e986c560b6504a7ec6969b1e07796fa1fdbf31454f990b33afa190b9c3e3`.
The external source fork was reviewed separately at clean commit
`ec9eea0af1d741cd4eb21c8766478a3e79dd44d6`.

## Blocking finding

### F1 — an absent live context bound is accepted as the argv claim

Revision 3 distinguishes malformed live metadata as `.unread`, but an answered
`/v1/models` entry that omits `meta.n_ctx` becomes `.notReported` and still
falls back to caller/profile argv:

- `RuntimeBenchmark.contextPolicy` maps `.notReported` to
  `--max-kv-size` (`RuntimeBenchmark.swift:733`).
- Admission explicitly permits `.notReported` when the flag is
  `--max-kv-size` (`RuntimeBenchmark.swift:830`).
- The new negative test covers `.unread`; it does not remove `meta.n_ctx` while
  retaining a finite `--max-kv-size` launch.

The reviewer drove the exact candidate Release binary's production
`benchmark-run` entry point. The only attack change was to make both live
`/v1/models` responses omit `meta` while leaving the finite
`--max-kv-size 76800` argv, package-owned baseline entry point, RECORD hashes,
immutable `direct_url.json`, real scenario driver, records, attestations, and
admission path intact.

Actual result: exit 0 and `decision.accepted=true`. The baseline evidence says:

```text
attestation.observedContextWindow.state = notReported
record.pins.contextPolicy = kv=76800;prefill-step=2048;reasoning=medium
record.provenance.launchArgv = ... --max-kv-size 76800 ...
decision.accepted = true
```

Thus the fact that the runtime honoured 76,800 is absent, yet the record
manufactures the same fact from the requested flag. This is **absent evidence
treated as satisfied**, specifically the partial-live-read variant. It violates
the acceptance requirement that `kv=76800` be derived from the running
server rather than argv alone. The producer's supplied-fact audit happened,
but missed this finite-bound fallback.

Required rework:

1. When argv declares any finite context bound, require a positive live
   `.reported(value)` observation and refuse `.notReported`; never derive the
   finite KV pin from argv alone.
2. Keep malformed, failed, partial, non-positive, and mismatched live reads
   inadmissible. An unbounded runtime may use an explicit no-bound state, but
   absence must not attest a requested finite bound.
3. Add a production-entry negative that removes `meta.n_ctx` while retaining
   `--max-kv-size 76800` and requires exit 4 with no `decision.json`.
4. Add a contract negative for `.notReported` plus a declared finite bound.
   Narrow only this case and prove the named test fails.
5. Add an append-only `LOGBOOK.md` closure for this review finding during
   producer rework; the reviewer did not alter the candidate.

## Confirmed closures and evidence

- The requested decoy attack is closed. A baseline launching `fake-runtime.py`
  while `--python-bin` names the valid isolated interpreter exited 5, named the
  entry-point mismatch, and wrote no decision. This drove
  `BenchmarkRunCommand.runPass`/`drive`, not a helper.
- A malformed live `meta.n_ctx` string becomes `kv=unread` and is inadmissible
  at the production entry point (exit 4). The defect is the missing/partial
  value path, not the malformed-value path.
- The fork is clean at immutable commit `ec9eea0...`; its installed
  `direct_url.json` names that exact commit as both `commit_id` and
  `requested_revision`.
- Direct construction through the pinned interpreter returns three
  `ArraysCache` entries and one `RotatingKVCache(max_size=76800)` for Qwen3.5.
- Mixed bounded/unbounded caches and differently bounded caches both derive no
  active bound (`None`), so the revision-1 mixed-cache finding remains closed.
- The benchmark-only Python and Swift profiles carry 76,800. The deployed
  `qwen-local` profile is not changed by this candidate.
- The previously attached 73,139-token live correctness evidence is tied to
  source commit `ec9eea0...`: 73,111 cached tokens, `finish_reason=stop`, and
  the correct three-system answer. This hour-scale generation was not rerun in
  revision 3; the reviewer accepted the already-attached raw evidence and
  rechecked the exact source commit and cache construction.
- Reviewer Release build passed. Focused `RuntimeBenchmarkTests` passed 75
  tests in 5 suites. The no-meta production attack completed the full smoke
  with zero failures, which is the defect evidence: the suite currently
  expects that forbidden acceptance. The malformed-meta variant refused at the
  production admission path.

## Reviewer artifacts

- `TASK-260830-2hc5r2_review-no-meta-production-rev3.log`
- `TASK-260830-2hc5r2_review-no-meta-decision-rev3.json`
- `TASK-260830-2hc5r2_review-no-meta-baseline-record-rev3.json`
- `TASK-260830-2hc5r2_review-no-meta-baseline-attestation-rev3.json`
- `TASK-260830-2hc5r2_review-decoy-production-rev3.log`
- `TASK-260830-2hc5r2_review-malformed-meta-production-rev3.log`
- `TASK-260830-2hc5r2_review-swift-focused-tests-rev3.log`
- `TASK-260830-2hc5r2_review-pinned-python-cache-rev3.log`
- `TASK-260830-2hc5r2_review-mixed-cache-rev3.log`

The reviewer made no repository code changes.
