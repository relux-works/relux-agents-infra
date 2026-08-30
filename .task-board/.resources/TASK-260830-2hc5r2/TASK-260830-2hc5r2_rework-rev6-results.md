# TASK-260830-2hc5r2 — revision 6 developer evidence

## Result

Revision 6 removes runtime-argv decoding from context-policy attestation. The
running server now reports its effective generation configuration, and the
comparison derives the policy only from the model entry sampled from that
process.

The source fork is published at signed commit
`45a472f2d0cda166b7ffe1a80fe50dd9621f4303` on
`origin/task/TASK-260830-2hc5r2-bounded-kv`. Its parent is the signed bounded-KV
revision `0a0452a9ca64d5b8ee3786fb23d3f828417f9514`; the revision-6 delta changes
only `mlx_lm/server.py` reporting and its server tests. The immutable
benchmark-only pipx install resolves both `commit_id` and `requested_revision`
to `45a472f2d0cda166b7ffe1a80fe50dd9621f4303`.

## Runtime-reported versus refused parameters

- `kv`: reported as `meta.n_ctx` by the active bounded cache after cache
  construction. Missing is `not-reported`; malformed or unread is `unread`.
- `prefill-step`: reported as the parsed effective
  `meta.runtime_config.prefill_step_size`. Missing is `not-reported`; malformed
  or unread is `unread`.
- `reasoning`: reported as the parsed effective
  `meta.runtime_config.reasoning_effort`. Missing is `not-reported`; malformed
  or unread is `unread`.

All six non-value states are inadmissible. None falls back to or is decoded
from argv. Launch argv remains provenance and a requested-model assertion only.
The `RuntimeBenchmark.contextPolicy` API no longer accepts runtime or argv
parameters, preventing accidental parser emulation at the production call
site.

## Exact reviewer attack

The real installed fork was launched with:

```text
--prefill-step-size 2048 --prefill-step-siz 999
```

Its live `/v1/models` response reported:

```json
{"runtime_config":{"prefill_step_size":999,"reasoning_effort":"medium"}}
```

The final production-entry smoke drives the same argparse abbreviation through
`benchmark-run`. The baseline record derives
`kv=76800;prefill-step=999;reasoning=medium`, the candidate derives
`kv=76800;prefill-step=2048;reasoning=medium`, and the gate refuses the pair
with exit 4 and no accepted decision.

## Bound and long-context evidence

The already-attached task evidence
`TASK-260830-2hc5r2_results.md` proves a 73,139-token prompt produced the
correct three-system answer with `finish_reason=stop`, 73,111 cached tokens,
and post-generation live `meta.n_ctx=76800`. Revision 5 additionally attached
`TASK-260830-2hc5r2_signed-live-completion-rev5.json` and
`TASK-260830-2hc5r2_signed-live-models-after-rev5.json` for the signed parent.

Revision 6 did not repeat that hour-scale generation. It directly reran the
fork model/server suite and verified that the new signed commit differs from
the already-proven signed parent only in model-listing reporting and tests. The
bounded Qwen3.5 cache construction and live `n_ctx` derivation are unchanged.

## Validation run directly in revision 6

| Command / gate | Exit | Evidence |
| --- | ---: | --- |
| Fork `python -m unittest tests.test_models tests.test_server` | 0 | 116 tests, 1 skipped |
| Fork Black check | 0 | `mlx_lm/server.py`, `tests/test_server.py` |
| Fork isort check with Black profile | 0 | same paths |
| `swift test -c release --filter RuntimeBenchmarkTests` | 0 | 73 tests in 5 suites |
| `swift test -c release` | 0 | 287 tests in 24 suites |
| macOS arm64 Xcode Release build | 0 | final exact candidate |
| `xcrun swift-format lint --strict --recursive Sources Tests` | 0 | final exact candidate |
| `bash -n scripts/benchmark-gate-smoke.sh` | 0 | syntax gate |
| `shellcheck -S warning scripts/benchmark-gate-smoke.sh` | 0 | warning gate |
| final production `benchmark-gate-smoke.sh` | 0 | 52 passes, 0 failures; exact abbreviation and absent-report attacks included |
| `git diff --check` | 0 | agents-infra story worktree |
| signed fork verification with task allowed-signers file | 0 | good signature for configured human identity |
| live installed-fork abbreviation report | 0 | effective prefill 999; no listener left behind |

The deployed `qwen-local` profile was not edited. Only
`examples/model-harness.benchmark.toml` names the isolated bounded executable
and 76,800-token benchmark pair.

## Honest non-green attempts

- Initial targeted fork test: exit 1 because the test fixture lacked the new
  `chat_template_args` field. The production read was made tolerant and the
  targeted and full suites then exited 0.
- Initial Black-profile-independent isort invocation: exit 1 because it did not
  use the repository's Black-compatible profile. The correct
  `isort --profile black --check-only` invocation exited 0.
- Initial Swift compile after removing the argv registry: exit 1 until obsolete
  parser-registry tests were replaced by live-report tests.
- Next targeted Swift run: exit 1 until reasoning fixtures and one manually
  constructed attestation carried live generation configuration.
- First strict Swift format gate: exit 1 for one missing trailing comma; the
  file was formatted and the repeated strict gate exited 0.
- First production smoke: exit 1 because its assertion expected one error line
  to enumerate two missing fields; admission correctly stopped at the first
  missing field. The assertion was narrowed to the published first cause.
- Second production smoke: exit 1 because the prior aborted decoy case left a
  task-owned listener on port 18791. Exact PID/argv were inspected, only those
  task-owned processes were terminated, and fresh-port full reruns exited 0.
- First plain `git verify-commit`: exit 1 because no SSH allowed-signers file was
  configured for that invocation. Repeating with the task-scoped allowed
  signers file exited 0 and reported a good signature.
- Binary readiness via top-level `--help`: exit 2 because this CLI exposes help
  only through usage errors. Mach-O architecture and executable bits for the
  exact runtime binary and `model-harness` were then verified with exit 0.

Final smoke also exposed the previously recorded separate cleanup debt: the
decoy provenance refusal can leave its model-harness child listening after the
gate exits. The exact final task-owned PIDs were inspected and terminated; no
listener remains on the final smoke port.
