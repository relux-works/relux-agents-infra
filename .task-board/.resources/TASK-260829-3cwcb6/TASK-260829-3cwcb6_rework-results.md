# TASK-260829-3cwcb6 rework results

Date: 2026-08-29  
Role: developer  
Scope: cross-runtime benchmark timing, resident-memory accounting, record-carried limitations

## Outcome

Revision 1's reasoning-delta correction and Mach-only refusal remain intact. The
rework restores a scored memory-economy axis without reviving the known
llama.cpp underestimate:

- both runtimes are sampled over the same warm-up, scenario, soak, and
  whole-process windows;
- the scored field is `peak_resident_memory_upper_bound_bytes`, never the raw
  legacy `peakPhysicalFootprintBytes`;
- one sample retains exact Mach physical footprint, the literal resident token
  from `vmmap -summary`, the conservative upper edge of that display bucket,
  and their sum;
- records state `scoreSemantics=conservative-upper-bound`;
- complete output with no mapped-file row is a measured zero, while absent,
  OS/tool read failure, incomplete/malformed output, and partial windows remain
  distinct and all refuse scoring;
- every generated record and `decision.json` carries the MTP-off direction:
  **against llama.cpp**, which is prevented from using a capability the MLX
  artifact lacks.

## Production-entry negatives

`scripts/benchmark-gate-smoke.sh` drives the shipped `benchmark-run` entry and:

1. launches one fixture emitting `reasoning` and one emitting
   `reasoning_content`; both must retain four generated events plus a non-null
   first-to-last decode interval. A reader that recognises only one spelling
   leaves the other decode unmeasured and fails the smoke.
2. launches an anonymous-memory fixture and an mmap-backed fixture; both must
   emit the same accounting and bound semantics at warm-up, scenario, soak,
   and process scope. The mmap record must have a positive mapped-file
   component, and the decision must contain the new metric and refuse the old
   metric name. A Mach-only regression therefore fails at the production call
   site rather than passing on its plausible small number.
3. asserts the exact MTP adverse direction in both records and the generated
   decision, so `speculation=off` alone is insufficient.

Contract negatives additionally drive the actual decision entry with absent,
read-failed, malformed, and partial resident-memory peaks. Every shape produces
an unmeasured blocker. The transcript seal changes when either new structured
memory field changes.

## Directional-bias audit

The audit was deliberately repeated looking for defects that make llama.cpp
look worse, not only the known Mach-only defect that made it look better.

| Item | State | Direction for llama.cpp as candidate |
| --- | --- | --- |
| Mach-only memory | corrected; raw field retained but never scored | old reading strongly favoured llama.cpp |
| `reasoning_content` omitted from generated events | corrected | mixed: old TTFT/prefill hurt llama.cpp, old decode helped it |
| response-tail decode endpoint | corrected | runtime/transport dependent |
| fixed baseline-then-candidate order | retained and record-carried | indeterminate: residual heat can hurt candidate; shared cache can help it |
| one-way parity success policy | retained and record-carried | against llama.cpp candidate; it favours the incumbent by policy |
| MTP forced off | retained and now record-carried | against llama.cpp; removes its unique capability |
| conservative resident-memory sum | new and record-carried | runtime-dependent; can double-count where Mach already charges mapped pages |
| finite llama.cpp KV bound vs unbounded MLX cache | admission refusal, not scored | no numeric direction is inferred |

No further unknown production reading was found that silently defaults to a
value adverse to llama.cpp. Failed or unreadable readings on the audited paths
remain refusals, not inferred absences. The new bound is not claimed exact or
direction-neutral.

## Validation and real exit codes

| Command / attempt | Exit | Result |
| --- | ---: | --- |
| `swift test --filter RuntimeMemoryAccountingTests` | 0 | 4 tests / 1 suite passed |
| `swift test --filter RuntimeBenchmarkTests` | 0 | 74 tests / 5 suites passed |
| first rework production smoke (`smoke-01`) | 1 | expected red during development: 13 failures exposed a parser that assumed vmmap's two-line header was one line |
| second production smoke (`smoke-02`) | 1 | 3 failures: mmap path passed; two old admission tests and one timing assertion were coupled to synthetic runtime speed |
| first Release production smoke (`smoke-03`) | 1 | 5 failures: the nominally wide fixture band still rejected a 0.24 synthetic decode ratio; memory record assertions themselves passed |
| final Release production smoke (`smoke-04`) | 0 | `BENCHMARK GATE SMOKE OK (0 failures)`; timing, mmap memory, MTP direction, refusal and attestation paths passed |
| first strict swift-format lint | 1 | 7 formatting findings, corrected mechanically |
| final `xcrun swift-format lint --strict --recursive Sources Tests` | 0 | clean |
| `shellcheck -S warning scripts/benchmark-gate-smoke.sh` | 0 | clean |
| `swift test -c release` | 0 | 400 tests / 32 suites passed |
| Xcode Release build for macOS arm64 | 0 | `BUILD SUCCEEDED`; only existing deprecation warnings |
| `git diff --check` | 0 | clean |

Persisted logs:

- `swift-test-release-01.log`
- `swift-format-lint-01.log`
- `shellcheck-01.log`
- `xcodebuild-release-01.log`
- `git-diff-check-01.log`

Production smoke sessions and per-run records/logs are under sibling directories
`smoke-01` through `smoke-04`; `smoke-04` is the accepted final evidence.

## Tool readiness

Before project work, `task-board`, `git`, `rg`, `go`, Swift 6.3.3 (arm64
macOS), `model-harness v1.6.1-44-gd91d6fc`, `shellcheck 0.11`, `xcrun`, and
`xcodebuild` were invoked successfully. The package declares macOS 15, so the
build and tests targeted macOS rather than an unrelated platform.

## Remaining measurement requirement

This rework validates production accounting with bounded mmap and anonymous
fixtures; it does not claim a fresh 28 GB model result. The migration article
must use a new `benchmark-run` session generated by this revision. Archived
Mach-only records remain historical evidence and are not rescored.
