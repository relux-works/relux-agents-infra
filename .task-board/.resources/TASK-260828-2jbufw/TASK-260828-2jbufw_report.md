# llama.cpp under the agents-infra managed harness

Task: TASK-260828-2jbufw (story STORY-260828-2faxgm)
Date: 2026-08-28
Author role: developer
Status: profile and lifecycle delivered; the benchmark-gate half is **not**
delivered and the reason is a named blocker, not an omission

## Verdict, in three parts

1. **`llama-server` runs as a managed child under a `model-harness` local
   profile with the full lifecycle.** Startup, readiness, health and a directed
   `SIGTERM` group shutdown with port release are all evidenced live
   ([§1](#1-the-profile), [§2](#2-lifecycle)). B7 of
   [TASK-260828-28gdmq][obs] — a directed `SIGTERM` orphaning the runtime — is
   reproduced against `llama-server` on the harness as it was at HEAD and is
   closed by this task.
2. **B8 is answered on evidence** ([§3](#3-b8-what-llama-server-emits)).
   `llama-server` is *not* like the other two runtimes. It emits no HTTP access
   line at any verbosity, so it is strictly less observable than
   `mlx_lm.server` on the URL channel; it emits per-request engine records
   keyed by a `task` id, so it is strictly *more* attributable than either MLX
   runtime; and from `-lv 5` upward it puts the **completion and tool calls**
   on stderr, which no previously audited runtime does. The prompt never
   appears at any tested verbosity. B4's shape — a status logged at
   header-write time and therefore reported as `200` for a request that died
   mid-body — **does not apply**, because there is no status line to be wrong;
   the outcome is unknown rather than misreported.
3. **The benchmark-compare gate cannot admit a llama.cpp candidate as it
   stands, and no clause was relaxed to make it.** Two clauses do not fit, one
   loudly and one silently ([§4](#4-the-benchmark-gate)). This is recorded as
   blocker **G1** and **G2** for the owner of TASK-260827-2v13w8 rather than
   patched around, and the cost of the obvious relaxation is measured with the
   gate's real production function.

[obs]: 260828_model-io-observability-through-harness.md

## Host constraint and fixture

TASK-260827-2v13w8 held this host's memory with the 28 GB model on port 18031
for the whole of this task, and was still running at the end of it. The 28 GB
GGUF staged by TASK-260828-3g87i4 was therefore **never loaded here**; every
measurement below uses a 676 MB `Qwen2.5-0.5B-Instruct` Q8_0 fixture
(`sha256 ca59ca7f13d0e15a8cfa77bd17e65d24f6844b554a7b6c12e07a5f89ff76844e`,
downloaded from the ungated `Qwen/Qwen2.5-0.5B-Instruct-GGUF`). Every probe
binds an OS-assigned ephemeral port, deliberately outside `18000-18999`, so
that the neighbouring run's host-contention guard is never tripped by this
task's listeners. No process belonging to another run was signalled.

Observability, readiness and shutdown are properties of the harness path and
of the runtime's logging, not of model size, so the small fixture carries every
claim below **except** retention under sustained load, which is reported as
untested for the same reason B6 reports it untested for the other runtimes.

The runtime under test is the pinned Homebrew `llama.cpp 0.3.0`, upstream
`v0.3.0`, commit `c1d0e7a004015f23bc0233470b747b596f29b264`, build 10621 —
the build TASK-260828-3g87i4 installed and pinned.

## 1. The profile

Additive. `profiles.qwen-local` is untouched and still points at Python
`mlx_lm.server`.

```toml
[profiles.llamacpp-local]
mode = "local"
executable = "/opt/homebrew/bin/llama-server"
argv = [
    "--model", "/absolute/path/to/model.gguf",
    "--host", "{host}",
    "--port", "{port}",
    "--ctx-size", "8192",
    "--batch-size", "2048",
    "--ubatch-size", "2048",
    "--reasoning-effort", "medium",
    "--no-webui",
]
```

No change to `internal/modelharness/config.go` was needed: the existing local
profile shape already expresses this launch, and the exact-token `{host}` /
`{port}` substitution renders the argv `llama-server` receives. The profile is
documented in `README.md` and the documented bytes are handed to the production
resolver by `TestREADMELlamaCppProfileResolvesAsDocumented`, so it is an
artifact the build checks rather than prose beside the code.

The four pinned flags are conditions the three runtimes do **not** default to
the same way:

| Condition | `mlx_lm.server` | mlx-swift | `llama-server` |
| --- | --- | --- | --- |
| KV bound | no flag; unbounded | `--max-kv-size`, absent by default | `--ctx-size`, **always finite** |
| prompt-eval chunk | `--prefill-step-size`, default 2048 | `prefillStepSize`, default 512 | `--ubatch-size`, default 512 (`--batch-size` is the logical batch, default 2048) |
| reasoning effort | `--chat-template-args '{"reasoning_effort": …}'` | `--reasoning-effort` | `--reasoning-effort` — *the same spelling* |

`--no-webui` is not cosmetic: the bundled UI is enabled by default and a
managed runtime should not serve one.

### The KV row is the one that matters, and it is measured

`llama.cpp` is never unbounded. With no `--ctx-size` the fixture reported
`n_ctx = 32768` (taken from the model); with `--ctx-size 8192` it reported
`n_ctx = 8192`, against `n_ctx_train = 32768` in both cases. Both figures are
read off the runtime's own `/v1/models` metadata, not off the help text.

## 2. Lifecycle

### 2.1 Readiness and health match the contract

`model-harness stress` polls `GET <endpoint>/models` and the Pi profile
declares `readiness_path = "/models"`; the endpoint is `http://host:port/v1`,
so readiness is `GET /v1/models` answering `200`. `llama-server` satisfies that
unchanged:

| Property | `llama-server` (fixture) |
| --- | --- |
| `GET /v1/models` before the weights are resident | `503` |
| `GET /v1/models` once resident | `200`, after 1.10 s and 1.12 s on two runs |
| model advertised | the configured file only, not a cache listing |
| `GET /health` | `200`, `{"status":"ok"}` |

That is the mlx-swift shape (readiness gated on a resident model), not the
`mlx_lm.server` shape (`/v1/models` answers `200` about a second after launch
with no weights resident, and lists every model in the local cache).
`llama-server` additionally offers a real `/health`, which `mlx_lm.server`
answers unconditionally (BUG-260827-1jhv2g) and the Swift prototype does not
expose at all. **No gap to record here: llama.cpp meets the contract and
exceeds it on `/health`.**

### 2.2 B7, reproduced against `llama-server` and closed

The same directed `SIGTERM` was sent at a standalone `model-harness run` twice
on the same fixture and the same profile: once with the harness built from
`HEAD`, once with the process-group shutdown.

| After `kill -TERM <harness pid>` | harness at HEAD | with group shutdown |
| --- | --- | --- |
| harness exit | `143` | `0` |
| `llama-server` child | **alive, reparented to pid 1** | gone |
| the listening port | **still held** | free |
| lifecycle records on stderr | `0` | `2` |
| runtime process group | the harness's own (`76948`) | its own (`77012`) |

The two records the fixed harness wrote:

```
model-harness: received terminated; stopping profile "llamacpp-fixture" process group 77012
model-harness: profile "llamacpp-fixture" process group 77012 stopped after terminated (child: <nil>)
```

The orphan the HEAD control produced was cleaned up by the probe itself; no
other run's process was signalled.

### 2.3 What the fix is

`internal/modelharness/run.go` now installs a handler for `SIGINT`, `SIGTERM`
and `SIGHUP`, starts the runtime with `Setpgid` so it leads its own process
group, forwards the received signal to `-pgid`, escalates to `SIGKILL` on the
group after ten seconds, writes a lifecycle record in both directions, and
returns `nil` for a completed signalled stop so a supervisor's `stop` is exit
`0`. A signalled attempt is never restarted by `restart_on_failure`. The
fatal-marker path also kills the group now rather than the one process the
harness holds a handle for. Windows has no process group here and stops the
direct child only; that is stated in `README.md` rather than implied.

This picks the first of the two positions B7 left open — `model-harness run`
owns supervisor-signal cleanup — which was B7's own recommendation.

## 3. B8 — what `llama-server` emits

Section 3 of [the observability report][obs] was repeated against
`llama-server`: nonces planted in a prompt body, an `X-Request-Id` header, a
tool schema, a streamed request and a URL path, with both harness streams
captured to separate files and searched. Two positive controls are included so
that a zero is a real absence and not a broken search.

| Nonce planted in | stdout | stderr |
| --- | ---: | ---: |
| POSITIVE CONTROL — `listening on http://127.0.0.1:<port>` | 0 | **1** |
| POSITIVE CONTROL — the model file name | 0 | **1** |
| user message body | 0 | 0 |
| `X-Request-Id` header | 0 | 0 |
| tool schema function name | 0 | 0 |
| streamed request body | 0 | 0 |
| URL path of a 404 | 0 | 0 |
| engine response id (`chatcmpl-…`) | 0 | 0 |
| concurrent prompt bodies | 0 | 0 |
| the string `POST /v1/chat/completions` anywhere | 0 | 0 |

### 3.1 stdout is empty; there is no access log at all

Across every run in this task `llama-server` wrote **0 bytes to stdout**. Every
line is on stderr, as with `mlx_lm.server`.

Unlike `mlx_lm.server`, there is **no HTTP access line at any verbosity**: no
method, no path, no status, no client. The two nonces that survived on
`mlx_lm.server` — the ones placed in a URL, which survived only because the
access line echoes the request line — do **not** survive here. On the URL
channel `llama-server` is strictly less observable than `mlx_lm.server`.

### 3.2 What it does emit, and why that is better

At the **default** verbosity, `llama-server` writes a per-request record set
that neither MLX runtime writes:

```
slot get_availabl: id  3 | task -1 | selected slot by LRU, t_last = -1
slot launch_slot_: id  3 | task 0 | processing task, is_child = 0
slot print_timing: id  3 | task 0 | prompt eval time = 65.53 ms / 187 tokens (0.35 ms per token, 2853.57 tokens per second)
slot print_timing: id  3 | task 0 |        eval time = 430.25 ms /  29 tokens (15.37 ms per token, 65.08 tokens per second)
slot      release: id  3 | task 0 | stop processing: n_tokens = 215, truncated = 0
```

Three concurrent completions produced three **distinct** `task` ids (63, 64,
65) on three slots with three sets of timings. Against `mlx_lm.server` the same
experiment produced three byte-identical access lines. So captured
`llama-server` output *is* internally attributable per request, with token
counts, timings and a truncation flag.

It still does not close B2. The `task` id never leaves the engine: the HTTP
response object carries `choices`, `created`, `id`, `model`, `object`,
`system_fingerprint`, `timings` and `usage`, and no slot or task field, and the
response headers carry only `Server: llama.cpp`. The join key exists on the
engine side alone, which is the mirror image of the `mlx_lm.server` case where
it existed on the Pi side alone.

### 3.3 Bodies at higher verbosity — a first among the audited runtimes

`-lv 5` and `-lv 10` were tested, not assumed.

| Channel | `-lv 3` (default) | `-lv 5` / `-lv 10` |
| --- | --- | --- |
| user message / assembled prompt | absent | **absent** |
| completion text | absent | **present** |
| tool calls with arguments | absent | **present** |
| tool schema | absent | **present**, as compiled GBNF |
| `X-Request-Id` | absent | absent |
| HTTP request line or status | absent | absent |

The completion arrives as exactly one line — the parsed assistant message:

```
D Parsed message: {"role":"assistant","content":"","tool_calls":[{"type":"function","function":{"name":"get_weather_TOOLNONCE2jbufw","arguments":"{\"city\": \"Yerevan\"}"}}]}
```

That is the first audited runtime where a model **output** is recoverable from
harness-captured bytes. The prompt is not: the user message text
(`Reply with exactly: …`, `What is the weather in Yerevan?`) had zero hits at
every tested level. The `<|im_start|>` fragments that do appear at `-lv 5` are
chat-template and parser machinery, not the assembled request.

Reasoning content was not separately exercised — the 0.5B fixture emits none —
but it is a field of the same `common_chat_msg` that `Parsed message`
serialises, alongside `content` and `tool_calls`. That is a structural reading
of the record shape, not a measurement, and it is reported as such.

`-lv 5` is not proposed as a default. One startup plus one 24-token request
produced 158,391 bytes of stderr, dominated by one-time load lines (1,304
`create_tensor:` lines alone), and steady-state per-request growth was not
measured. It would also route completions and tool calls into whatever sink the
caller pointed the harness at, and `runtime.log` is still unbounded and
unrotated (B6).

### 3.4 B4 does not apply, and the difference is in our favour

The B4 shape is: a status logged at header-write time, so a request that died
mid-body is recorded as `200`. Reproduced against `llama-server` — a 900-token
completion started, then the runtime `SIGKILL`ed with the body in flight — the
client got `http=000` (empty reply) and the captured stream contains **no
status line at all**, because there is none to write.

What it does contain is a signature: the in-flight request's
`launch_slot_: id 1 | task 91 | processing task` line with **no matching
`print_timing` or `release`**, where every completed request in the same
capture has both.

So the outcome is **unknown**, not misreported. The interim rule B4 states for
`mlx_lm.server` — "a `200` in a runtime log is not evidence that a request
succeeded" — has no `llama-server` counterpart to warn about. Anything
reasoning over these logs should read a `launch_slot_` without a `release` as
an unfinished request rather than as an absence of a request.

The unsupervised harness followed the child out and reported exit `1`, which is
the existing documented behaviour.

## 4. The benchmark gate

### 4.1 Why the gate half is not delivered

TASK-260827-2v13w8 is mid-flight redesigning the comparison gate to put runtime
launch, scenario driving, measurement, record construction and judgement under
one trusted production invocation. That work is:

- on branch `task-board/story/STORY-260827-m30k8z`, which this branch does not
  contain (`tools/mlx-swift-runtime-prototype` does not exist on `main` or on
  this story's branch), and
- **uncommitted** in that story's worktree — `BenchmarkRunCommand.swift`,
  `BenchmarkCompareCommand.swift`, `RuntimeBenchmark.swift` and the rest are
  staged-but-unlanded.

There is no way to add a third runtime's `RunRecord` production to that driver
from here without forking 4,500 lines of another run's in-flight source into
this branch, which would guarantee a conflict and would be exactly the
"caller-authored record" shape being removed. Per the task's own sequencing
instruction, the profile half is delivered and this is said plainly instead of
guessed at.

What *is* delivered on this side is the finding the gate owner needs, produced
by compiling their real `RuntimeBenchmark.swift`
(`sha256 d8377708ae4e893cb4f65b8aa4c524a9929c72f032992403b0bc808dd2291e18`)
read-only out of that worktree and calling the production function.

### 4.2 What fits without any change

`RuntimeBenchmark.contextPolicy(derivedFrom:)` reads three conditions off the
recorded argv. One of the three fits llama.cpp with **no change at all**:
`llama-server` spells reasoning effort `--reasoning-effort LEVEL`, which is the
identical spelling the Swift runtime uses and which the derivation already
reads. A llama.cpp launch pinning `--reasoning-effort medium` derives
`reasoning=medium` exactly as the other two do.

### 4.3 G1 — the prefill pin refuses llama.cpp (loud)

```
llama.cpp, flags llama-server accepts
  derived : kv=unbounded;prefill-step=unpinned;reasoning=medium
  admitted: NO -- refused on ["prefill-step=unpinned"]
```

The derivation reads `--prefill-step-size`. `llama-server` has no such flag;
its physical prompt-evaluation chunk is `-ub` / `--ubatch-size`. So every
llama.cpp record is refused by `AdmissionError.unpinnedLaunchCondition`,
whether or not the launch actually pinned the chunk. Loud, and safe.

**The obvious relaxation is not safe, and the cost is measured.** Dropping
`prefill-step=unpinned` from `RuntimeBenchmark.unpinnableConditions` is the
smallest edit that admits llama.cpp. Run against the real function:

| Launch | as shipped | with the clause relaxed |
| --- | --- | --- |
| llama.cpp (the launch the relaxation is for) | REFUSED | admitted |
| mlx-swift **without** `--prefill-step-size` (defaults to 512) | REFUSED | **admitted** |
| python-mlx-lm **without** `--prefill-step-size` (defaults to 2048) | REFUSED | **admitted** |

All three derive the byte-identical string
`kv=unbounded;prefill-step=unpinned;reasoning=medium`, so there is no way to
relax the clause "only for llama.cpp" from the derivation alone. The relaxation
admits a 512-token prefill chunk measured against a 2048-token one and calls
the difference a runtime difference — the exact comparison the clause exists to
prevent, and the one review previously measured as a 1.93x skew.

**Recommendation, for the gate owner to decide:** widen the *derivation*, not
the clause — read `-ub` / `--ubatch-size` as a third spelling of the same
condition, exactly as `--chat-template-args` was added as a second spelling of
reasoning effort. That keeps `unpinnableConditions` untouched and still refuses
a llama.cpp launch that left the chunk to the default. It is a change to
another run's in-flight file and is not made here.

### 4.4 G2 — the KV pin is silently false for llama.cpp (dangerous)

This is the more serious one because it *passes*.

`contextPolicy` derives the KV bound from `--max-kv-size`, and reads its
absence as `unbounded` on the stated grounds that "absent on both sides means
the same thing". That justification holds for two runtimes and breaks on the
third. `llama.cpp` has **no unbounded mode**: measured above, `n_ctx` is 32768
from the model with no flag and 8192 with `--ctx-size 8192`. A llama.cpp record
would therefore carry `kv=unbounded` while running against a finite KV cache,
and — because `Pins.firstMismatch` requires the baseline and candidate
`contextPolicy` to be **equal** — it would *match* an MLX baseline's genuinely
unbounded reading. A false equality on a pin is worse than a refusal: the pin
exists so a declared condition is a reading of the launch, and here the reading
would be wrong while the gate stayed green.

Note that this is not fixed by reading `-c` / `--ctx-size` alone. Absence of
`-c` still means a finite bound for llama.cpp and no bound for `mlx_lm.server`,
so the "absence means the same thing" premise cannot be restored by adding a
spelling. Either the derivation becomes runtime-aware, or the bound is read
from the running server's `/v1/models` metadata (which reports `n_ctx`
directly) rather than from argv.

**This is a decision about the gate's trust model — whether a pin may be
derived from anything other than the recorded argv — and it belongs to
TASK-260827-2v13w8.** It is recorded, not patched.

### 4.5 No clause was relaxed

This task changed nothing in `benchmark-compare`, `RuntimeBenchmark.swift`, or
anything else in `tools/mlx-swift-runtime-prototype`; that tree is not on this
branch and this branch's diff does not touch it. The narrowing evidence for the
clause is §4.3's table, produced by running the shipped function against a
relaxed copy of its own clause list and showing what else the relaxation lets
through.

## 5. Blockers

<a id="g1"></a>
### G1 — the prefill pin has no llama.cpp spelling
`contextPolicy(derivedFrom:)` reads only `--prefill-step-size`; llama.cpp
spells it `-ub`/`--ubatch-size`. Every llama.cpp record is refused.
**Owner:** TASK-260827-2v13w8.
**Recommendation:** add the spelling to the derivation. Do **not** relax
`unpinnableConditions`; §4.3 measures what that admits.

<a id="g2"></a>
### G2 — the KV pin reads `unbounded` for a runtime that never is
Absence of `--max-kv-size` is read as `unbounded`. llama.cpp is always bounded
by `n_ctx`. The record would pass while asserting something false, and would
falsely match an MLX baseline.
**Owner:** TASK-260827-2v13w8.
**Needed:** a decision on whether a pin may be derived from the running
server's reported `n_ctx` rather than from argv alone. There is no additive
argv spelling that restores the "absence means the same thing" premise.

<a id="g3"></a>
### G3 — the single-invocation driver does not exist on any shared branch yet
The benchmark driver that would produce a llama.cpp `RunRecord` with the same
`LaunchProvenance` binding is uncommitted in another story's worktree. Nothing
here can be built against it without forking it.
**Resolution path:** once TASK-260827-2v13w8 lands, adding a llama.cpp pass is
a profile plus a `--candidate-runtime llama-cpp` argument, provided G1 and G2
are settled first.

<a id="b8-closed"></a>
### B8 — closed
`llama-server` is no longer untested. §3 answers it on evidence for the pinned
build 10621. The one part deliberately left open is reasoning content, reported
as a structural reading of the record shape rather than a measurement, and
retention under sustained load, which stays untested for the same reason B6
reports it untested for the other runtimes.

## 6. Tests added

`tools/agents-infra/internal/modelharness/run_shutdown_test.go`

| Test | What it bounds |
| --- | --- |
| `TestRunSignalledShutdownStopsTheWholeProcessGroup` | the forwarded signal reaches the runtime's grandchildren, and does so **without** the SIGKILL escalation behind it |
| `TestRunSignalledShutdownEscalatesToKill` | a runtime that ignores `SIGTERM` is still stopped |
| `TestRunSupervisedShutdownDoesNotRestart` | an operator stop is not a restartable failure, on a child that exits non-zero under `SIGTERM` |
| `TestModelHarnessRunReleasesPortOnDirectedSIGTERM` | the **shipped binary**, a **real** signal, and the port released — the only one bound to `signal.Notify` actually being installed |

`tools/agents-infra/llamacpp_profile_docs_test.go` resolves the README profile
through the production resolver and pins the documented stop contract.

Every test was verified against a mutant that should kill it. Production call
sites: `Run` → `run` (`signal.Notify`) → `runWithSignals` → `runOnce` /
`runSupervised` → `runSupervisedAttempt` → `shutdownRuntime` →
`signalRunProcessGroup` / `killRunProcessGroup`.

| # | Mutant | Killed by | Result |
| --- | --- | --- | ---: |
| N1 | `signalRunProcessGroup` signals `pid` instead of `-pid` | `…StopsTheWholeProcessGroup` | exit 1 |
| N2 | `killRunProcessGroup` kills `pid` instead of `-pid` | `…EscalatesToKill` | exit 1 |
| N3 | the signalled-attempt early return removed, so `restart_on_failure` sees the stop | `…SupervisedShutdownDoesNotRestart` | exit 1 |
| D1 | `signal.Notify` removed entirely — the behaviour at HEAD | `…ReleasesPortOnDirectedSIGTERM` **only** | exit 1 |
| D2 | `configureRunProcess` removed from `runOnce` | `…StopsTheWholeProcessGroup`, `…EscalatesToKill`, `…ReleasesPortOnDirectedSIGTERM` | exit 1 |

Two of these are worth recording rather than just counting.

**N1 survived the first time it was run**, and the reason is a real trap. With
the child signalled directly, `exec.Cmd.Wait` still blocks on the pipes the
grandchild inherited, so the grace period expires, and the *unmutated*
`killRunProcessGroup` escalation then cleans up the whole group. The test saw
a dead group and passed. A group that is SIGKILLed on every stop is not the
contract, so the test now asserts that the group stopped **on the forwarded
signal** — the absence of the escalation record — and N1 dies. A delete-only
mutant, or an assertion on the end state alone, would have proved nothing here.

**D1 is killed by exactly one test**, and that is the point: the three seam
tests feed the signal channel by hand and cannot see whether a handler is
installed at all. Only the shipped-binary test can, and under D1 it fails with
`output=""` — literally B7's "zero bytes written".

`run_shutdown_test.go` also reaps its fixture process groups in `t.Cleanup`.
Two mutation runs in this task leaked fixture shells that ignore `SIGTERM`
after the mutated harness failed to stop them; they were cleaned up by hand and
the reaper prevents a repeat.

## 7. Commands run and exit codes

Each as a standalone process, exit code as reported by the shell.

| Command | Exit |
| --- | ---: |
| `gofmt -l .` (in `tools/agents-infra`) | 0, no output |
| `go vet ./...` | 0 |
| `go build ./...` | 0 |
| `go test ./internal/modelharness/ -count=1` | 0 |
| `go test ./... -count=1` (declared board validation) | 0 — root 100.3 s, infra 187.2 s, modelharness 2.9 s, attachments 2.0 s |
| N1, N2, N3, D1, D2 mutation runs | 1 each, as intended |
| `llamacpp-observability-probe.sh` | 0 |
| `llamacpp-verbosity-probe.sh` | 0 |
| `llamacpp-lifecycle-probe.sh` | 0 |

Both commands declared in `task-board.config.json ->
spawn.worktree_isolation.validation` were run as declared, each in
`tools/agents-infra`, each exit 0.

## 8. Artifacts

Raw evidence under `.temp/TASK-260828-2jbufw/` (gitignored), archived to the
board as `TASK-260828-2jbufw_evidence.tar.gz`: the three probe scripts and
their captured output, the harness stdout/stderr captures at three verbosity
levels, the pre-fix/with-fix lifecycle report, the five mutation logs, the full
`go test` log, and the two compiled gate probes with the digest of the source
they were built from.
