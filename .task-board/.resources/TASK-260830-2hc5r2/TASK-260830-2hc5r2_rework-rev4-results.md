# TASK-260830-2hc5r2 rework revision 4

## Outcome

- `RuntimeContextWindow` now routes the live KV fact through one three-state rule: `observed(value)`, `observedAbsent`, or `notObserved`.
- `RuntimeBenchmark.contextPolicy` derives `kv` only from that live observation. `notReported` becomes `kv=not-reported`; `unread` remains `kv=unread`; both are inadmissible and neither can fall back to `--max-kv-size`.
- A finite declared bound is accepted only when the live value is observed and equal. An answered omission with `--max-kv-size 76800` is refused as `contextBoundNotHonoured`.
- Prefill and reasoning derive from `ProcessObservation.arguments`, the kernel argv of the process that served, rather than caller-supplied profile argv.
- `benchmark-gate-smoke.sh` drives both production attacks: omitted `meta.n_ctx` with a finite launch flag, and a process that re-execs with `prefill-step=999` despite profile argv declaring `2048`.
- README and append-only `LOGBOOK.md` document the contract and the observed cleanup anomaly.

Production call path: `BenchmarkRunCommand.drive` reads `/v1/models` plus kernel argv, constructs the record and attestation, then `RuntimeBenchmark.admit -> admitProvenance` derives and enforces the policy.

The source fork remains clean at `ec9eea0af1d741cd4eb21c8766478a3e79dd44d6`. This revision does not alter the fork or deployed default profile; the previously attached 73k generation and Python-suite evidence remains the source-bound proof for that unchanged commit.

## Validation

| Command / evidence | Exit | Result |
| --- | ---: | --- |
| Focused absent-bound test before production fix | 1 | Expected red: admission returned `Comparison`; no error was thrown |
| Same focused test after fix | 0 | Exact absent-live-evidence regression passed |
| `swift test -c release` | 0 | 290 tests in 24 suites passed |
| macOS arm64 Release `xcodebuild build` | 0 | `BUILD SUCCEEDED` |
| `benchmark-gate-smoke.sh`, fresh port 30171 | 0 | Control accepted; no-meta and rewritten-argv attacks exit 4; zero smoke failures |
| `swift format lint --strict --recursive Sources Tests` | 0 | Clean |
| `shellcheck -S warning scripts/benchmark-gate-smoke.sh` | 0 | Clean |
| `bash -n scripts/benchmark-gate-smoke.sh` | 0 | Clean |
| `git diff --check` | 0 | Clean |

Two intermediate smoke reruns exited 1 because old task-owned fake-runtime processes still held their ports. They are preserved as failed evidence in `benchmark-gate-smoke-rev4-02.log` and `benchmark-gate-smoke-rev4-03.log`, were not counted as passing, and the exact task-scoped PIDs were terminated before the fresh-port run. The successful decoy branch again left one listener, which was explicitly cleaned and recorded as separate cleanup debt in `LOGBOOK.md` rather than folded into this attestation rework.

## Evidence files

- `red-absent-bound-01.log`
- `green-absent-bound-01.log`
- `swift-tests-rev4-02.log`
- `xcode-release-build-rev4-02.log`
- `benchmark-gate-smoke-rev4-04.log`
- `swift-format-lint-03.log`
- `shellcheck-rev4-02.log`
- `bash-n-rev4-02.log`
- `git-diff-check-rev4-02.log`
