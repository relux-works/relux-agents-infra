# TASK-260829-3k4qrc revision 2 review verdict

Verdict: **changes requested**. Route: `to-dev`.

Reviewed Change Request `CR-TASK-260829-3k4qrc-2` revision 2, base
`3272e3ae60b53eb0fb61f6d427a883487711b48f`, candidate tree
`8208c98bc5e7d29479c7d98e3603cffefb878811`. The attached patch reproduces
SHA-256 `339f45d93fbe52768c830b0fe219a2b12ac7c384ca74fa495628b5fc0f514cf4`,
and applying it through an alternate index reconstructs the exact candidate
tree.

## F1 — mmap residency can be stale while the peak is attested `measured`

Severity: blocking for the comparison's memory axis.

`BenchmarkFootprintSampler.physicalLoop` reads Mach footprint every 50 ms but
combines it with `latestMappedFile`, which is refreshed by `vmmap` only every
five seconds (`BenchmarkFootprintSampler.swift:127-186`). It timestamps that
combined value at the fast Mach-read time. `samplingCoverageIssue` then checks
only those composite timestamps (`RuntimeMemoryAccounting.swift:52-73`) and
therefore mistakes a five-second-old mmap component for fresh 50 ms evidence.

Reviewer production-source attack: exact candidate accounting and sampler
sources were compiled in task-scoped scratch. A 256 MiB read-only file mapping
was made resident for less than the five-second refresh interval. A direct call
through the same production `residentMemorySample` observed mapped residency;
the sampler then scored the window after the mapping was gone.

```
directMapped=268645172
scoredMapped=82944
missed=268562228
status=measured
issues=[]
samples=32
```

The gate admitted a score while missing about 256 MiB of the component that
exists specifically for llama.cpp's mmap-loaded weights. This is the standard
negative shape **capability claim that does not reproduce**.

The shipped 150 ms probe does not cover this class: it allocates with
`MAP_ANON` and asserts only the Mach-footprint delta
(`BenchmarkMemorySamplerProbe.swift:15-42`). It passes on revision 2. Disabling
the new fast physical observer makes the same probe exit 1 with two samples,
`status=partial`, and `resident-memory-sampling-gap`, so the anonymous
old/new regression is real; it is simply not evidence for mmap freshness.

Required rework:

- Track freshness/timestamp coverage independently for Mach and mapped-file
  components. Reusing a mapped value must not mint a new mapped observation.
- Score only when the mmap component has the coverage the claim requires, or
  refuse explicitly. A lower-cost reader is fine; silently stretching a stale
  value across the gap is not.
- Add a file-backed transient production-entry fixture that fails on this exact
  revision and passes on the replacement. Keep the anonymous probe as a
  separate capability check.
- Record this regression and closure in `LOGBOOK.md` during producer rework.

## F2 — cache non-comparability is declared but never gates scoring

Severity: blocking for `multiturn_prefix_reuse` and the requirement that every
non-comparable dimension be refused.

Revision 2 correctly enumerates both directions and says their effect is
unknown: baseline-only `--prompt-cache-size 1 --prompt-cache-bytes 8GB`, and
llama.cpp per-slot KV reuse (`BenchmarkRunCommand.swift:292-302`). It also says
that an observed one-sided reuse hit "must be refused rather than scored".

That rule has no structured observation or decision call site. Repository-wide
search finds the rule only in the declaration string and the smoke assertion
that the string exists. `RuntimeBenchmark.decide` computes scenario
comparability solely from prompt-token skew (`RuntimeBenchmark.swift:2067-2151`)
and then scores TTFT, prefill, decode, and memory. No cache-hit fact reaches it.
The smoke at `benchmark-gate-smoke.sh:2216-2219` proves only that the note was
copied into the record.

Revision-1 evidence already reports the dangerous input: baseline reuse did not
fire while llama.cpp reuse did. Under revision 2 that scenario remains
numerically scoreable despite the newly declared rule. This is **check present
but uncalled from production**.

Required rework:

- Carry a structured, sealed reuse observation per runtime/scenario and make a
  one-sided hit set scenario comparability false; or, if reuse cannot be
  established reliably, report it `unknown` and do not score the affected
  scenario.
- Add a production-entry negative with identical prompt tokens and a one-sided
  reuse fact. It must produce `unmeasured`/outside evidence and a blocker, while
  the symmetric/no-hit control remains scoreable.

## Real-runtime scoring evidence remains pending

Revision 2 deliberately did not repeat the hour-scale pair, per the reviewer
brief. Consequently the previous 75k values were produced before this sampler
change and cannot prove that both actual revised runtime configurations land on
the scoring side. The review's fake-runtime mmap session scored every scenario,
but its Python whole-process peak was `partial` with
`resident-memory-sampling-gap`; the decision correctly became `accepted=false`
with unmeasured process memory while the smoke stayed green. The smoke explicitly
allows this branch at `benchmark-gate-smoke.sh:2202-2212,2256-2272`.

After F1 and F2 are fixed, the full sequential, same-host, MTP-off real pair must
be rerun. Both runtimes' `context_75k` memory must be scored or the dimension
must be reported as refused; the task cannot close on the preserved revision-1
numbers.

## Verified closures and validation

- Base note reproduced: Story branch is exactly 7 commits behind both `main`
  and `origin/main`; none of those commits touches benchmark/measurement code.
  No merge/rebase/base movement was performed.
- Bounded-KV merge closure survived the production attacks in the full smoke:
  `--python-bin` decoy exits 5 with no decision; missing live KV stays
  inadmissible despite argv; malformed KV is unread rather than absent;
  rewritten caller argv, repeated argparse flags, and argparse abbreviation all
  expose the running server's effective report; equal live bounds admit. No
  bounded-KV merge regression was found.
- Process-group cleanup survived the exact provenance-refusal path. The smoke
  passed its owned-group absence check, and post-run inspection found zero
  task-config/port-marked survivors with PPID 1.
- Full reviewer smoke: exit 0, `BENCHMARK GATE SMOKE OK (0 failures)`. The
  attached revision-2 transcript contains 115 PASS checks versus the prior 52
  (+63); the suite did not shrink.
- `swift test -c release`: 404 tests / 32 suites, exit 0.
- `swift build -c release`: exit 0 (existing deprecation warning only).
- `xcrun swift-format lint --strict --recursive Sources Tests`: exit 0.
- `shellcheck -S warning scripts/benchmark-gate-smoke.sh`: exit 0.
- `bash -n scripts/benchmark-gate-smoke.sh`: exit 0.
- Candidate `git diff --check`: exit 0.
- Tool readiness: macOS Swift 6.3.3, swift-format 6.3.0, ShellCheck 0.11.0,
  Python 3.14.7, and the configured `model-harness` all executed successfully.

No repository code was modified by the reviewer. Scratch attacks live under
`.temp/TASK-260829-3k4qrc/reviewer-r2/`. `accept_cr` was not called.
