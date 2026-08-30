# TASK-260830-2hc5r2 — revision 5 developer evidence

## Outcome

Revision 5 addresses both reviewer findings without changing the deployed default profile.

- `RuntimeBenchmark` now owns a runtime-specific argv semantics registry. `python-mlx-lm` follows Python argparse store-action behavior: supported long options accept `--flag=value` and the last repeated occurrence wins. `mlx-swift` follows `RuntimeOptions.parse(arguments:)`: only spaced spellings are admitted and duplicates are unresolved because the runtime rejects them. Unknown runtimes and recognized aliases outside the selected runtime grammar derive `unresolved` and are refused.
- The production path `BenchmarkRunCommand.drive` derives `contextPolicy` from the observed process identity, kernel argv, and post-generation `/v1/models` context-window observation. `RuntimeBenchmark.admitProvenance` independently re-derives the same policy with the record's runtime semantics.
- The exact reviewer negative `--prefill-step-size 2048 --prefill-step-size 999` is driven through production `benchmark-run`; the baseline is recorded as effective `999`, the candidate remains `2048`, and the pair exits 4 without an accepted decision.
- The bounded source tree is republished as signed commit `0a0452a9ca64d5b8ee3786fb23d3f828417f9514` on `relux-works/mlx-lm` branch `task/TASK-260830-2hc5r2-bounded-kv`. Its tree `7378b59c77a2add1dce15de9bf099b399867b761` is byte-identical to reviewed unsigned commit `ec9eea0af1d741cd4eb21c8766478a3e79dd44d6`.
- The task-only executable, benchmark profile, README command, and immutable pip `direct_url.json` now pin the signed OID. The installed executable is `/Users/alexis/.local/bin/mlx_lm-kv76800-0a0452a.server`. The deployed default profile is unchanged.
- A real completion from the newly installed signed server returned `finish_reason=stop` and the exact `SIGNED_BASELINE_OK` marker. The same running process then reported `meta.n_ctx=76800`. Before the first generation it reported no context metadata, confirming that live attestation must sample after cache construction and must not infer KV from argv.
- The prior 73,000+ token correctness evidence remains source-valid because signing changed only the commit object, not the tree. Revision 5 additionally exercises the signed installation live and proves its post-generation bound and output correctness.

The original external fork branch backing upstream PR 1791 was restored to its exact prior head `b0a45b8fdd3bb5d6c390d65a6f8521c296f980ec`. An accidentally automated fork PR was closed after the fork's local policy was read; the explicitly required signed task branch remains published and no automated PR remains open.

## Validation

| Validation | Exit | Result |
| --- | ---: | --- |
| Exact pre-fix repeated-Python-prefill unit witness | 1 | Expected red: production semantics returned non-effective `2048` instead of argparse-effective `999` |
| `swift test -c release --filter RuntimeBenchmarkTests` | 0 | 80 tests / 5 suites |
| `swift test -c release` | 0 | 295 tests / 24 suites |
| Fork `/Users/alexis/.local/pipx/venvs/mlx-lm-qwenfix/bin/python -m unittest tests.test_models tests.test_server` | 0 | 115 tests, 1 skipped |
| `xcodebuild build ... -configuration Release -destination platform=macOS,arch=arm64` | 0 | `BUILD SUCCEEDED` |
| `xcrun swift-format lint --strict --recursive Sources Tests` | 0 | No diagnostics after formatting |
| `bash -n scripts/benchmark-gate-smoke.sh` | 0 | Shell syntax valid |
| `shellcheck -S warning scripts/benchmark-gate-smoke.sh` | 0 | No warning/error diagnostics |
| `git diff --check` | 0 | Clean |
| `benchmark-gate-smoke.sh` using the built Release binary | 0 | `BENCHMARK GATE SMOKE OK (0 failures)`; repeated-flag, missing-live-KV, rewritten-argv, decoy-provenance, malformed/absent evidence and replay attacks all refused |
| Signed commit `git verify-commit` with the task allowed-signers file | 0 | Good ECDSA signature for configured human identity `alexis@relux.works` |
| `git ls-remote` signed task branch | 0 | Remote ref equals `0a0452a9ca64d5b8ee3786fb23d3f828417f9514` |
| Installed `direct_url.json` assertion | 0 | Unique `mlx_lm-*.dist-info` record pins both `commit_id` and `requested_revision` to signed OID |
| Signed live server artifact validation | 0 | Observed argv contains `--max-kv-size 76800`; post-generation `n_ctx=76800`; stopped output contains exact marker; listener cleaned |
| Task-owned smoke-child cleanup verification | 0 | No matching fake runtime/model-harness process and no listener on the decoy port |

## Red and non-counted attempts retained honestly

- An invalid Swift filter expression exited 0 while selecting zero tests; it is not counted as evidence.
- An intermediate focused Swift run exited 1 because valid JSON without `reasoning_effort` was initially classified as malformed; the implementation was corrected so valid omission is `unpinned` while malformed JSON is `unresolved`.
- The first fork test invocation used Xcode Python without `mlx` and exited 1; the exact source suite was rerun with the known MLX-capable environment and exited 0.
- The first live signed-server assertion exited 1 because it required `n_ctx` before any cache existed and requested too few output tokens for a reasoned completion. The corrected live run records the absence before generation, proves correct stopped output, and attests `n_ctx=76800` after cache construction.
- The first installed-provenance assertion hard-coded the stale distribution directory `mlx_lm-0.30.7.dist-info` and exited 1. The corrected assertion requires exactly one `mlx_lm-*.dist-info/direct_url.json` and validates the signed OID; it exited 0.
- An initial tree-identity shell assertion nominally exited 0 but zsh treated unquoted `^{tree}` revisions as unmatched globs, so it established nothing and is not counted. The corrected assertion quotes both revisions, resolves both trees, compares them, and exits 0.
- Default-severity `shellcheck` exited 1 on existing info/style diagnostics (`SC2015`, `SC2086`, `SC2181`). The warning/error gate was run separately and exited 0; the red default run is not reported as passing.
- The production smoke's decoy early-abort path left one task-owned child, matching the already recorded cleanup anomaly. Its exact process and parent were terminated explicitly; the cleanup verification exited 0.

## Evidence files

- `.temp/TASK-260830-2hc5r2/benchmark-gate-smoke-rev5.log`
- `.temp/TASK-260830-2hc5r2/benchmark-gate-rev5/duplicate-python-prefill.log`
- `.temp/TASK-260830-2hc5r2/python-tests-rev5.log`
- `../.temp/TASK-260830-2hc5r2/swift-test-release-rev5.log`
- `../.temp/TASK-260830-2hc5r2/xcodebuild-release-rev5.log`
- `.temp/TASK-260830-2hc5r2/fork-signature-rev5.log`
- `.temp/TASK-260830-2hc5r2/fork-commit-metadata-rev5.log`
- `.temp/TASK-260830-2hc5r2/fork-remote-ref-rev5.log`
- `.temp/TASK-260830-2hc5r2/fork-tree-identity-rev5.log`
- `.temp/TASK-260830-2hc5r2/installed-provenance-rev5.log`
- `.temp/TASK-260830-2hc5r2/live-signed-baseline-02/`
