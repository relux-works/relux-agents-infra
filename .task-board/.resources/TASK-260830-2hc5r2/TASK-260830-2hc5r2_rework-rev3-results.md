# TASK-260830-2hc5r2 — Rework Revision 3 Developer Outcome

## Result

The benchmark baseline revision is now derived from the process that served the
pass. `--python-bin` is an assertion only. A baseline whose observed process
cannot be tied to its package-owned `mlx_lm.server` entry point, interpreter,
RECORD hashes, and immutable `direct_url.json` revision aborts without a
decision.

The benchmark-only Python and Swift profiles both use a 76,800-token KV bound.
The running server reports that bound through `/v1/models` as `meta.n_ctx`, and
the attestation derives `kv=76800` from that live value. A failed or malformed
live context read becomes `kv=unread` and is inadmissible; it never falls back
to argv. The deployed default profile is unchanged.

The source implementation remains the clean `relux-works/mlx-lm` fork commit
`ec9eea0af1d741cd4eb21c8766478a3e79dd44d6`. Its Qwen3.5 cache construction was
rechecked directly through the pinned environment and returned
`RotatingKVCache 76800 ArraysCache` (exit 0).

## Production Attack Evidence

The final smoke used the Xcode Release binary and drove `benchmark-run` itself.
The accepted control produced:

- baseline context: `kv=76800;prefill-step=2048;reasoning=medium`
- candidate context: `kv=76800;prefill-step=2048;reasoning=medium`
- `accepted=true`

The revision-2 reviewer reproduction launched `fake-runtime.py` under
`/usr/bin/python3` while `--python-bin` pointed at the isolated immutable
baseline environment. It exited 5, wrote no `decision.json`, and reported that
the runtime revision could not be attributed to the process that served. The
production call site is `BenchmarkRunCommand.runPass`, which settles the
observed serving process and calls `BenchmarkRunCommand.pythonRevisions` before
constructing the record pins.

Darwin retains the interpreter image in kernel `argv[0]` after a shebang exec
and the executed console script in `argv[1]`. The final gate requires the
canonical package-owned entry point in that exact slot, so the expected path
cannot be smuggled as an unused later argument.

## Supplied-Fact Audit

- `--candidate-binary` is now only an assertion against the candidate's
  observed executable; Swift revisions come from that executable.
- The model-harness revision now comes from the observed harness executable.
- Thresholds, prompts, and profile declarations are explicit policy/workload
  inputs, not runtime-identity attestations.
- Served model and context remain live runtime reports under the documented
  non-malicious-runtime trust boundary. No other acceptance fact is pinned from
  a separately caller-selected executable.

## Validation Run In This Revision

| Command | Exit | Evidence |
| --- | ---: | --- |
| pinned Python direct Qwen3.5 `make_cache(..., 76800)` check | 0 | `RotatingKVCache 76800 ArraysCache` |
| `swift test -c release` | 0 | 289 tests in 24 suites passed |
| `swift build -c release` | 0 | Release SwiftPM binary linked |
| `xcodebuild build -scheme mlx-swift-runtime-prototype -configuration Release -destination 'platform=macOS,arch=arm64' -derivedDataPath ./DerivedData -skipPackagePluginValidation -skipMacroValidation` | 0 | `BUILD SUCCEEDED`; only the pre-existing `Preflight.swift` quantization deprecation warning remains |
| Xcode Release `scripts/benchmark-gate-smoke.sh`, output `benchmark-smoke-10` | 0 | 0 failures, including the exact decoy provenance attack and all existing refusal attacks |
| `xcrun swift-format lint --strict --recursive Sources Tests` | 0 | no findings |
| `bash -n scripts/benchmark-gate-smoke.sh` | 0 | syntax valid |
| `git diff --check` | 0 | no whitespace errors |

The pinned Python environment does not contain the development test
dependencies: `python -m pytest --version` exited 1 (`No module named pytest`),
and stdlib unittest discovery exited 1 because `requests` is absent. Those are
reported as red and are not claimed as test passes. This revision did not rerun
the expensive 73,000+ token live generation; it relies on the already-attached
source-fork evidence for commit `ec9eea0`, while rerunning the source cache
construction and every agents-infra gate changed here.

## Red Attempts Preserved

- Initial strict Swift format lint: exit 1; formatting findings were corrected,
  and the exact lint reran green.
- `benchmark-smoke-07`: exit 1 because port 19291 was already in use.
- `benchmark-smoke-08`: exit 1 after an intentionally over-narrow self-review
  change required the console script at kernel `argv[0]`; live evidence showed
  Darwin places it at `argv[1]`. The gate was corrected to that exact slot and
  rerun green in `benchmark-smoke-09` and the final Xcode run
  `benchmark-smoke-10`.

