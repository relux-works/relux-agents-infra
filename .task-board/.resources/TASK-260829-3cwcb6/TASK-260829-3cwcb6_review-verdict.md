# TASK-260829-3cwcb6 — Reviewer verdict

## Verdict

**Changes requested.** Route to `to-dev`; this is acceptance-blocking implementation
rework, not an external Stop-The-Line boundary.

Reviewed Change Request `CR-TASK-260829-3cwcb6-1` revision 1 at candidate tree
`11edb37e794dc064e41da633843eb398beb7fb7c`, base
`4332e1dddd0164876b4da3ec0340ba9320aec1e9`. The independently exported patch
SHA-256 is `287c98d3d2f84bba0100cca5b183c73208b40ec8da6788d9f0b7602634002a3d`, matching
the Change Request.

## Acceptance-blocking findings

### F1 — The first-class memory criterion is now honest but undecidable

`RuntimeMemoryAccounting`, `BenchmarkFootprintSampler`, and
`BenchmarkRunCommand.drive` correctly prevent Mach `ri_phys_footprint` from being
scored for an exact `llama-server` executable or a reviewed `.gguf` artifact.
Process, scenario, soak, and warm-up routes all use the classified sampler; the
decision fails closed rather than spending `nil` as zero. That correction must
remain.

It is not sufficient for this task. Memory economy is the owner's first-class
criterion, and revision 1 deliberately leaves every llama.cpp memory metric
unmeasured. The next full rerun therefore cannot decide the axis it exists to
decide. This is not a platform-imposed impossibility: the prior `context_75k`
evidence already reproduced a conservative **41.05 GiB** llama.cpp upper bound
against **45.20 GiB** Python, with `ps`/`vmmap` residency evidence and mapped-file
residency accounting. The same prior record explicitly recommends sampling
resident mapped-file bytes beside the process reading.

Required rework:

- add a production memory accounting method that sees mmap-loaded weights and is
  applied to **both** runtimes over the same warm-up, scenario, soak, and
  process-peak windows;
- name the new quantity honestly (do not continue calling a composite or
  resident bound `peakPhysicalFootprintBytes`), preserve raw component readings,
  and state whether the scored value is exact or a conservative upper bound;
- keep failed/partial/malformed reads distinct from absence and fail closed;
- retain the current negative proving the old Mach-only llama.cpp reading is
  refused, then add a production-entry negative proving the new comparable
  reading is present for both runtime shapes and cannot be narrowed back to
  exact executable matching.

The earlier 41.05/45.20 arithmetic proves feasibility; it does not authorize
silently hard-coding that historical value or mixing unlike quantities.

### F2 — MTP's adverse direction is still buried outside the records

The two structural policies are real and correctly directed:

- one-way parity favours the incumbent: baseline success plus candidate failure
  blocks, while the reverse does not;
- MTP-off algorithmic parity is against llama.cpp's product advantage because
  llama.cpp can draft from the GGUF MTP head and the MLX incumbent cannot.

Revision 1 carries the first direction explicitly into both records through the
built-in `parityPolicyDirection`. It does not do the same for MTP. The records
carry `speculation=off` and the trusted non-equivalence that the MLX build drops
the MTP head; the statement that disabling MTP removes a llama.cpp advantage
exists only in code/docs/audit prose. A record consumer must infer the direction,
contrary to the checklist requirement that every remaining known limitation's
direction be explicit in the record.

Required rework: add the MTP-off directional limitation to both production
records and assert it through `benchmark-run`, so a generated decision/report
cannot omit it.

## Corrections accepted in substance

- `content`, `reasoning`, and `reasoning_content` share one generated-event
  definition at `BenchmarkHTTPDriver.stream` via `RuntimeStreamDelta.read`.
- TTFT begins on the first generated event and decode ends on the last generated
  event, not on usage or `[DONE]` tail work.
- The old mmap-invisible Mach reading is refused at the real `benchmark-run`
  entry and all scored footprint fields become unmeasured rather than zero.
- The one-way parity direction and fixed-order limitation are real and correctly
  directed. Fixed order has indeterminate net direction: candidate-second heat
  can hurt llama.cpp while shared host cache can help it.
- No further demonstrated numeric defect biased against llama.cpp was found.
- The thermal claim is honestly left indeterminate: one-minute load average
  cannot attribute a candidate-second delta. The renamed-binary/non-`.gguf`
  limitation is also honest rather than avoided work; the shipped executable and
  real `.gguf` wrapper shape are covered, while the unsupported shape was not
  guessed from a name.
- Producer smoke history is accurate. Attempt 1 ended exit 1 with two fixture
  failures (process observation plus an incorrect optional-key assertion),
  attempt 2 ended exit 1 with the process-observation failure, and attempt 3
  ended exit 0 with `BENCHMARK GATE SMOKE OK (0 failures)`.

## Independent validation

All green checks below ran against an archive of the immutable candidate tree,
not the dirty Story worktree:

| Check | Exit | Result |
| --- | ---: | --- |
| `swift test -c release` | 0 | 398 tests / 32 suites passed |
| `xcrun swift-format lint --strict --recursive Sources Tests` | 0 | clean |
| `shellcheck -S warning scripts/benchmark-gate-smoke.sh` | 0 | clean |
| `benchmark-gate-smoke.sh` with the candidate Release binary | 0 | 0 failures |
| post-mutant `swift test -c release --filter RuntimeStreamDeltaTests` | 0 | 6 tests / 2 suites passed after byte-for-byte restoration |

The producer's canonical Xcode Release build was inspected from its attached
spawn log and accepted as existing evidence; I independently rebuilt the entire
Swift package through the full release test command, but did not repeat the
Xcode shader-bundle build because neither requested rework finding depends on
Metal serving.

## Gate-defeat evidence

Both reported narrowing shapes were reproduced against the candidate's
production artifact, with caches disabled by source changes and the mutated
Release binary rebuilt:

| Mutant | Unit exit | Production smoke exit | Production failure |
| --- | ---: | ---: | --- |
| remove `reasoning_content`, retain `reasoning` only | 1 | 1 | reasoning pair returned decision exit 3; candidate decode became unmeasured; two smoke failures |
| classify mmap only by exact `llama-server` basename | 1 | 1 | `.gguf` candidate returned accepted exit 0 and carried a scored Mach number; two smoke failures |

The second production mutant demonstrates the forbidden old reading directly:
the `.gguf` candidate was accepted and its process footprint was scored when the
artifact arm was narrowed away. Both mutated files were restored from copies and
the focused suites returned green afterwards.

## Evidence notes

One initial clean-archive test launch did not execute because the scratch log
path was wrong and `status` is a reserved zsh variable. A following initial
dependency-resolution session was handed off incorrectly by the reviewer tool
wrapper; its surviving SwiftPM process was observed compiling and was not counted
as a test result. The successful exact-tree commands above were rerun directly
and carry their real exit codes.

The reviewer did not edit `LOGBOOK.md`: changing a repository file would mutate
the immutable candidate under review and violate the reviewer read-only role.
The producer should append F1 and F2 to the Flight Logbook in revision 2.
