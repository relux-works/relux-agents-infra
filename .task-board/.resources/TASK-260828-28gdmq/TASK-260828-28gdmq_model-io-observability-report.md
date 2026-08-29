# Model I/O observability through the agents-infra managed harness

Task: TASK-260828-28gdmq (story STORY-260828-2faxgm)
Date: 2026-08-28
Author role: developer
Status: investigation result — no observability feature was implemented

## Verdict

The belief under test was: *because agents-infra proxies child stdin/stdout,
everything that happens with the model can be tracked.*

**That belief is false**, and the exact shape of the failure matters more than
the headline. The harness proxies the child's stdout/stderr faithfully, but on
every audited runtime the prompt and completion bodies never enter that stream —
they travel over the HTTP socket between Pi and the engine, which the harness
does not sit on.

Stated precisely, the established result is two-part:

1. **Model-harness-captured engine streams contain neither the assembled HTTP
   request nor the HTTP response body.** No prompt, completion, tool schema,
   tool call, tool result, reasoning content or engine response id appears in
   harness stdout or stderr on any audited runtime (§3). The only request detail
   that survives is what the access line echoes back from the URL.
2. **Pi-side records cannot be joined to engine-side requests.** No captured
   engine-side record carries any identifier that can be tied to the Pi turn
   that caused it (§4, §8, §9).

What is deliberately **not** claimed: that nothing on the managed path shows
model I/O at all. Pi's own event stream and session transcript do carry the user
message text and the assistant completion text, and this task's own Pi artifact
contains both (§7). That evidence is real and is preserved here rather than
erased by the headline. It is also partial in a way that matters: the
85-character user prompt visible Pi-side became 4966 prompt tokens on the wire,
so what the model actually saw — system prompt, tool schemas, assembled message
array — appears in full on neither side. Pi's transcript is the agent's view of
the conversation, not the request that was sent, and it carries no key that
reaches the engine.

Eight named blockers are recorded in [Blockers](#blockers). None was fixed here,
deliberately: fixing one in passing would have made this report describe a system
that did not exist when the question was asked.

## Host constraint and fixture

Another tracked run (TASK-260827-2v13w8) was serving the 28 GB Qwen model on this
host throughout. That model was never loaded by this task. All live tracing used
the cached `mlx-community/Qwen1.5-0.5B-Chat-4bit` fixture (261 MB), as
TASK-260827-2h39ya and TASK-260827-2q77g8 did. Observability is a property of the
harness path rather than of model size, so the small fixture is sufficient for
every claim below **except retention under sustained load**, which is reported as
untested.

The 28 GB run was used once, read-only, as a passive natural experiment: its
process file descriptors and its runtime log were inspected without sending it a
single request.

## 1. What the harness proxies today

`model-harness run` (`tools/agents-infra/internal/modelharness/run.go`):

| Channel | Behaviour | Code |
| --- | --- | --- |
| child stdin | inherited from the harness process (`os.Stdin`) | `run.go:32`, `run.go:80` |
| child stdout | forwarded byte-for-byte to the harness's own stdout | `run.go:33`, `run.go:81` |
| child stderr | forwarded byte-for-byte to the harness's own stderr | `run.go:34`, `run.go:82` |
| emitted JSON events | **none authored by the harness**; any JSON seen is the child's own | — |
| supervision markers | literal substring scan over the forwarded bytes; scanning only, never recording | `run.go:137-166` |
| exit handling | child exit status wrapped into the harness's error and exit code 1 | `run.go:36`, `cmd/model-harness/main.go:20-21` |
| signal handling | **none installed** | see [B7](#b7) |

The harness authors exactly two kinds of line of its own, both on stderr, both
only on the supervision path:

```
model-harness: restarting profile "fixture-tiny" after supervised runtime failure (1/2): signal: killed
model-harness: run local profile "fatal-fixture": restart budget exhausted after 2 restarts in 3600s: fatal output "RuntimeError: [metal::malloc] Resource limit (": signal: killed
```

On the shared-broker path the same child streams are redirected into a single
`runtime.log` file (`internal/infra/pi_shared_broker_darwin.go:330,344-345`). On
the exclusive Pi path they go to the launcher's stderr — the operator's terminal
— and are not persisted at all (`internal/infra/pi_launch_posix.go:225-227`).

## 2. Which channel carries prompts and completions, per runtime

Answered from evidence, not from assumption.

| Runtime | Prompt body | Completion body | Tool calls | Basis |
| --- | --- | --- | --- | --- |
| `mlx_lm.server` 0.31.3 | HTTP socket only | HTTP socket only | HTTP socket only | live trace, §3 |
| Swift prototype (`mlx-swift-runtime-prototype`) | HTTP socket only | HTTP socket only | HTTP socket only | source read + live 28 GB runtime log, §2.2 |
| `llama-server` | **unknown — untested** | **unknown — untested** | **unknown — untested** | see [B8](#b8) |

### 2.1 `mlx_lm.server`

Everything the runtime writes goes to **stderr**; its stdout was 0 bytes across
the entire session. Per request it emits one Python `http.server` access line:

```
127.0.0.1 - - [28/Aug/2026 13:19:46] "POST /v1/chat/completions HTTP/1.1" 200 -
```

plus prompt-cache and prefill-progress `INFO` lines. That line contains the
method, the URL path, the protocol, and a status. It contains no body, no
response id, and not even the client's source port — Python's
`BaseHTTPRequestHandler` logs `address_string()`, which is the host alone.

### 2.2 Swift prototype

Read-only inspection of the live 28 GB runtime owned by TASK-260827-2v13w8
(pid 4691 harness, pid 4694 child; both fds 1 and 2 pointing at one log file):

```
{"event":"listening", ...}
{"event":"model_loaded", ...}
{"event":"ready", ...}
```

986 bytes total for a whole serving session — three lifecycle events and nothing
else. The source confirms this is by design: the only per-request event site is
`Router.swift:99` `generation_worker_failed`, on the failure path. There is no
success-path request event, and no request body is ever written to stdout.

## 3. Live recoverability from harness-captured output alone

A supervised `model-harness run` of the fixture was driven through six real
requests with distinctive nonces planted in prompts, headers, tool schemas and
URL paths. Both harness streams were captured to separate files and grepped.

| Nonce planted in | stdout hits | stderr hits |
| --- | ---: | ---: |
| `PROMPTNONCE-7f3a9c21-ALPHA` (user message body) | 0 | 0 |
| `REQIDNONCE-11223344` (`X-Request-Id` header) | 0 | 0 |
| `TOOLNONCE` (tool argument in the prompt) | 0 | 0 |
| `get_weather_TOOLNONCE` (tool schema function name) | 0 | 0 |
| `chatcmpl-` (engine's own response id) | 0 | 0 |
| `"cannot provide information"` (completion text) | 0 | 0 |
| `STREAMNONCE-abc123` (streamed request) | 0 | 0 |
| `REFUSEDNONCE` (**URL path**, 404) | 0 | 1 |
| `MALFORMEDNONCE` (**URL path** of an upstream fetch) | 0 | 1 |

The only nonces that survive are the two that were placed in a **URL**, and they
survive only because the access-log line echoes the request line. Nothing that
travels in a body — prompt, completion, tool schema, tool call, tool result,
reasoning content, or a caller-supplied correlation header — appears anywhere in
harness-captured output. Streaming behaves identically to buffered.

Reasoning content was not separately exercised on the fixture (Qwen1.5-0.5B emits
none), but it travels in the same response body as the completion text, which is
demonstrably absent; the finding covers it by construction rather than by proxy.

## 4. Correlation back to the originating request

**There is no request identity carried end to end.** This is [B2](#b2).

Three concurrent completions were issued against one runtime. The three responses
were distinct — three different `chatcmpl-` ids and three different completions —
and the captured output for them is three byte-identical lines:

```
127.0.0.1 - - [28/Aug/2026 13:20:18] "POST /v1/chat/completions HTTP/1.1" 200 -
127.0.0.1 - - [28/Aug/2026 13:20:18] "POST /v1/chat/completions HTTP/1.1" 200 -
127.0.0.1 - - [28/Aug/2026 13:20:18] "POST /v1/chat/completions HTTP/1.1" 200 -
```

Same second, same text, no ordering guarantee relative to the interleaved
prefill-progress lines. Given N in-flight requests, the captured record cannot
tell you which of them any line belongs to.

A half-key does exist and is worth naming precisely. Pi's own event stream
records the engine's response id:

```json
"responseId":"chatcmpl-03529889-dc3d-41f2-b3b8-21211362087f","rawStopReason":"stop","willRetry":false
```

So Pi knows the engine's identifier for its request. The engine never writes that
identifier anywhere the harness can see. The join key exists on one side of the
boundary only, which is why the correlation fails — not because no identifier
exists, but because it is never emitted on the side that would need it.

## 5. Failure paths

### 5.1 Refused request

`POST /v1/chat/completions-REFUSEDNONCE` → `404`. Captured evidence is one access
line carrying the path and status. The refused body is not captured. A refusal is
attributable only to the extent that its URL differs from a successful one.

### 5.2 Crashed child

`kill -9` of the runtime child while the harness supervised it:

```
model-harness: restarting profile "fixture-tiny" after supervised runtime failure (1/2): signal: killed
```

One line, correctly identifying the profile, the restart counter and the signal.
It carries no timestamp, no PID, and no accounting of what was in flight.

### 5.3 Crashed child with a request in flight — the captured record is *wrong*

The same kill was repeated with a 600-token completion in flight. The client
received nothing (`curl` reported status `000`, "Empty reply from server"). The
captured record for that request says:

```
127.0.0.1 - - [28/Aug/2026 13:21:01] "POST /v1/chat/completions HTTP/1.1" 200 -
```

The status is logged when the response header is written, before the body is
generated. A request that returned zero bytes to its caller is recorded as `200`.
For the failure path the captured evidence is not merely absent — it actively
misreports the outcome. This is [B4](#b4).

### 5.4 Condemned worker

A synthetic child emitting the configured fatal substring exercised the
supervision path end to end:

```
RuntimeError: [metal::malloc] Resource limit (condemned worker)
model-harness: restarting profile "fatal-fixture" after supervised runtime failure (1/2): fatal output "..." : signal: killed
model-harness: restarting profile "fatal-fixture" after supervised runtime failure (2/2): fatal output "..." : signal: killed
model-harness: run local profile "fatal-fixture": restart budget exhausted after 2 restarts in 3600s: fatal output "..." : signal: killed
```

Harness exit code 1. This is the one path where the harness authors genuinely
attributable evidence of its own: marker matched, restart counter, budget
exhaustion, non-zero exit. It still says nothing about which requests died with
the condemned worker.

The same run also proved the pipe itself is honest: the synthetic child's
`SECRETPROMPT-should-not-leak` line appears verbatim in captured stdout on every
attempt. The channel works. The runtimes simply do not write bodies to it.

## 6. Retention, location and size

| Path | Location | Rotation | Size cap | Retention |
| --- | --- | --- | --- | --- |
| `model-harness run` standalone | **none** — inherits the caller's stdout/stderr | n/a | n/a | entirely the operator's |
| Pi exclusive mode | launcher's terminal stderr (`pi_launch_posix.go:225-227`) | n/a | n/a | **not persisted at all** |
| Pi shared-broker mode | `~/Library/Caches/agents-infra/pi-runtimes/<key>/runtime.log` | **none** | **none** | append-only, forever |
| Pi session lifecycle log | `<pi state>/runs/<run>/logs/<ts>-<nonce>.jsonl`, mode 0600 | new file per session | none | accumulates, never pruned |
| Broker lease records | `<runtime key>/leases/` | — | — | **deleted on release** |

`openSharedLog` (`pi_shared_broker_darwin.go:373-387`) opens with
`O_CREAT|O_APPEND|O_WRONLY` and never truncates. A repo-wide search for
truncation, rotation, pruning or size-cap logic in `internal/infra` returns
nothing. `cleanupSharedRuntimeState` (`:490-500`) removes lease mirrors, broker
state and the rendezvous socket, and deliberately leaves `runtime.log` in place —
so the log outlives its runtime while nothing ever bounds it.

Observed on this host: 8 stale `pi-runtimes/*/runtime.log` files from prior days,
147 HTTP request lines across them, largest 72 KB, and 29 accumulated session
JSONL files. Aggregate is small today because these are short sessions; nothing
in the code makes that a property rather than an accident. Retention behaviour
under sustained load was **not tested** — see [B6](#b6).

Content check across all 8 real production runtime logs, for the avoidance of
doubt: `"content"` 0 hits, `"role"` 0 hits, `chatcmpl-` 0 hits, `tool_call`
0 hits.

## 7. One real Pi turn, end to end

A managed `agents-infra pi spawn` turn was run against the fixture through the
harness — the full chain, not a simulation.

**Pi side** (session `01a047e6-f98c-7540-903b-d750fe9c2dbc`): 78 JSON events —
`session`, `agent_start`, `turn_start`, 2 × `message_start`/`message_end`,
68 × `message_update`, `turn_end`, `agent_end`, `agent_settled`. Final record
carries `responseId`, `stopReason`, `willRetry`, and usage
`input=4966, output=67`.

**agents-infra session log** (8 JSONL records): `session_start`,
`runtime_started` (pid/pgid 21012), `runtime_ready`, `pi_started` (pid/pgid
21022), `pi_exited`, `pi_cleanup`, `runtime_cleanup`, `session_end`.

**Engine side**: `GET /v1/models` (readiness probe) then exactly one
`POST /v1/chat/completions`, plus prefill-progress lines for 4966 tokens.

So one Pi turn produced one engine completion request here.

### 7.1 Compaction — attempted in the traced session, did not fire

Compaction was not left inferred. A four-turn session was run on one session id
with compaction enabled and the threshold pushed down to
`compact_at_tokens = 4870` against a context that Pi reported as 4861 input
tokens. agents-infra wrote the policy through correctly — the profile's isolated
`agent/settings.json` shows `{"enabled":true,"reserveTokens":27898,
"keepRecentTokens":512}`, and 27898 = 32768 - 4870 — and the session transcript
accumulated all four turns (8 message records). **No `compaction` record was
produced**, on the one-shot `agents-infra pi --print --session-id` path, within
those four turns.

So compaction is reported here from production transcript evidence — three real
`compaction` records on this host, carrying `tokensBefore`, `firstKeptEntryId`
and their own usage accounting — rather than from the traced session. Why it did
not fire at that threshold on the one-shot path was not chased further; it is out
of this task's scope and is **not** claimed as a defect. What matters for the
observability question is unchanged and was observed on every one of the four
turns: each turn, compacted or not, produces exactly one
`POST /v1/chat/completions` line on the engine side, and a compaction request
would be a fifth line of the same indistinguishable shape.

Retry likewise did not occur in the traced session and is reported from the
transcript's record shape (§8), not from a forced failure.

A detail worth stating plainly: the user prompt was 85 characters, and the engine
processed **4966 prompt tokens**. The Pi-side record shows the 85 characters. The
wire payload — system prompt, tool schemas, assembled message array — is what the
model actually saw, and it appears in full on neither side. Pi's transcript is the
agent's view of the conversation, not the request that was sent.

## 8. Does Pi-side turn identity reach the engine request?

No. This is [B2](#b2), stated from the Pi direction.

Pi's session transcript is rich — `id`/`parentId` chains, session UUID,
millisecond timestamps, full message content, and `compaction` records carrying
`tokensBefore`, `firstKeptEntryId` and usage. Record-type census over all
persisted transcripts on this host: 230 `message`, 4 `session`, 4 `model_change`,
4 `thinking_level_change`, 3 `compaction`, 2 `session_info`.

None of it reaches the engine. The engine-side record has second granularity, no
session id, no run key, no lease id, no response id and no client port. The only
available join is timestamp proximity, which is a proxy, not a key, and which
collapses entirely under concurrency (§4, §9).

Two specific consequences:

- **Compaction** is visible on the Pi side as a typed record with its own usage
  accounting. On the engine side it is an ordinary indistinguishable POST.
- **Retries** are worse: the transcript has no `retry` record type at all. Pi
  carries a `willRetry` flag on the assistant message, so a retried turn can be
  detected, but N engine attempts still collapse to one transcript message with
  nothing tying the attempts to it. This is [B5](#b5).

## 9. The shared-broker case

Two independent Pi sessions — separate project roots, separate run keys, separate
leases — were run concurrently against one broker-owned runtime.

| Session | Pi session id | Pi-side responseId |
| --- | --- | --- |
| A | `01a047e7-abc4-70c0-8cb3-2ae9bc0cf7b4` | `chatcmpl-c6a63791-f20b-4cfd-93d8-edb9d5b2ec20` |
| B | `01a047e7-af88-7d55-964c-ad10122ead59` | `chatcmpl-1c45bc5d-8d9f-4d38-b7be-c54aff308de7` |

The broker's single `runtime.log` contains, for those two sessions:

```
127.0.0.1 - - [28/Aug/2026 13:26:00] "POST /v1/chat/completions HTTP/1.1" 200 -
127.0.0.1 - - [28/Aug/2026 13:26:01] "POST /v1/chat/completions HTTP/1.1" 200 -
```

Grep over that log: `SHAREDNONCE-AAA` 0, `SHAREDNONCE-BBB` 0, `chatcmpl-` 0, both
run-key prefixes 0, `lease` 0.

**Captured engine traffic cannot be attributed to the session that caused it.**
Under the shared broker the situation is strictly worse than the exclusive case,
because the two sessions' traffic is interleaved into one file with no
per-connection marker, and because `broker.log` was 0 bytes and `leases/` was
already empty by the time both sessions ended — the lease records that would at
least establish *who held a lease when* are deleted on release. This is
[B3](#b3).

## Blockers

Recorded as gaps, not implemented.

<a id="b1"></a>
### B1 — HTTP request and response bodies are not on any channel the harness observes

The harness proxies stdout/stderr; prompts and completions travel over the HTTP
socket. No amount of stream capture can recover them.
**To close:** terminate the model connection inside agents-infra — a loopback
reverse proxy between Pi and the engine that records request/response bodies —
or require every managed runtime to emit structured per-request records on
stdout. The first is runtime-agnostic and costs a hop; the second needs
cooperation from `mlx_lm.server`, which is upstream and out of our control.
**Owning decision:** proxy vs. runtime cooperation, and — separately and first —
whether recording prompt bodies at all is acceptable given they carry repository
contents. Every other body-level blocker (B2, B4, and half of B5) waits on this
answer, so it is the one decision that unblocks the rest.
**Recommendation:** the loopback reverse proxy, because it is runtime-agnostic
and does not depend on upstream `mlx_lm.server` accepting a logging change.

<a id="b2"></a>
### B2 — No request identity is carried end to end

Pi knows the engine's `responseId`; the engine never emits it into any captured
stream. There is no agents-infra-minted request id anywhere.
**To close:** mint a request id at the Pi boundary, carry it as a header, and
have whatever terminates the connection log it alongside the Pi session id, run
key and lease id.
**Owning decision: none of its own — explicitly dependent on B1.** There is no
component on the engine side today that could write such a record, so the join
key cannot be specified before B1 decides who terminates the connection. Once B1
is answered, B2 is implementation, not a decision.

<a id="b3"></a>
### B3 — Shared-broker traffic is unattributable, and lease evidence is not durable

One `runtime.log` per runtime key, N leasing sessions, no per-lease marker;
`leases/` is emptied on release and `broker.log` stays 0 bytes.
**Owning decision:** does lease attribution live in the log layout or in a
broker event log? Concretely — segregate captured runtime output per lease (a
file or a tagged stream per leaseholder), or keep one `runtime.log` and add a
durable broker event log recording lease grant/release with run key, project and
PID.
**Recommendation:** the durable broker event log. It is cheap, independently
useful for lease debugging, does **not** depend on B1, and does not require
rewiring the runtime child's file descriptors. It buys *who held a lease when*,
not *which request belonged to whom*; per-request attribution still needs B1+B2.
**Needed from a human:** accept that intermediate scope, or require full
per-lease output segregation instead.

<a id="b4"></a>
### B4 — Captured status misreports requests that died mid-body

An access line reads `200` for a request whose caller received zero bytes,
because the status is logged at header-write time (§5.3).
**Owning decision:** proxy or runtime cooperation — who is permitted to observe
the end of a response? Logging the outcome after the body is flushed cannot be
done inside `mlx_lm.server`, which is upstream and out of our control, so it is
either the B1 reverse proxy recording the real terminal state or a hard
requirement that every managed runtime emit an end-of-response record.
**Recommendation:** fold this into B1 and do not fund it separately; no cheaper
independent path exists.
**Interim rule, which needs no decision and applies today:** a `200` in a runtime
log is not evidence that a request succeeded. Anything reasoning over these logs
must treat the outcome as unknown rather than as success.

<a id="b5"></a>
### B5 — Retries collapse

Pi's transcript has no retry record type; N engine attempts appear as one
message. `willRetry` detects that a retry happened but does not enumerate the
attempts.
**Owning decision:** which side owns retry accounting — Pi or the engine-side
recorder? Either Pi emits a per-attempt record (a `retry` record type, or an
attempt counter on the assistant message), or B1+B2 produce one record per engine
request and attempts are counted there.
**Recommendation:** Pi-side per-attempt records. They are useful before B1
exists, and Pi is the only component that knows an attempt is a retry rather than
a new turn.
**Needed from a human:** Pi is upstream of agents-infra, so the decision is
specifically whether to ask Pi for this record type or to wait for B1.

<a id="b6"></a>
### B6 — `runtime.log` is unbounded and unrotated; behaviour under load untested

`O_APPEND` with no rotation, no size cap, no pruning, and cleanup deliberately
preserves the file. Small today only because sessions have been short.
**Untested:** growth and retention under sustained unattended operation. Not
inferred from the small fixture, because the property depends on request volume
and prefill-progress verbosity, which the fixture does not represent.
**Owning decision:** the concrete retention policy — a size cap per segment, a
number of retained segments, and whether teardown deletes a runtime's log or
preserves it under a maximum age. No policy exists today, so this is a product
decision rather than an implementation choice; there is no defensible default to
pick unilaterally.
**Recommendation:** cap each segment at a stated size with N rotated segments,
and keep the current cleanup-preserves-the-log behaviour only if a maximum age is
attached to it — otherwise a stale log outlives its runtime forever, which is
what happens now.
**Also needed from a human:** the load protocol to measure against — request
rate, session duration, and the prefill verbosity of the real serving runtime.
Growth under sustained unattended operation stays untested until that
measurement is run, and it is not extrapolated from the fixture here.

<a id="b7"></a>
### B7 — A directed SIGTERM to `model-harness run` orphans the runtime silently

Reproduced twice (once with the fixture, once with a trivial child):
`kill -TERM <harness pid>` leaves the harness dead with **zero bytes** written —
no session-end record, no lifecycle marker — while the child is reparented to
pid 1 and keeps holding its port. `runOnce`/`runSupervisedAttempt` install no
signal handler and set no `SysProcAttr`.
**Caveat, stated so the finding is not overclaimed:** an interactive Ctrl-C
sends SIGINT to the whole foreground process group and both processes die; and
the Pi-managed path does clean up correctly — the session log for the traced turn
shows `runtime_cleanup {"state":"confirmed"}`. The gap is specific to a directed
signal at a standalone `model-harness run`, which is exactly what a service
supervisor or a scripted stop would send.
**Owning decision:** does a standalone `model-harness run` own supervisor-signal
cleanup, or is it declared an interactive-only entry point? Today it is neither —
it takes the signal without owning the cleanup, and nothing documents that.
**Recommendation:** it should own cleanup — handle SIGTERM/SIGINT, put the child
in its own process group, and emit a lifecycle record before exiting. The
alternative position (interactive-only, must always be stopped by process group)
is defensible, but it then has to be documented and enforced, because a service
supervisor or a scripted stop sends exactly the signal that breaks today.
**Needed from a human:** pick one of those two positions. The caveat above is why
this is a real choice rather than a self-evident bug fix.

<a id="b8"></a>
### B8 — `llama-server` is untested

Not installed on this host: `llama-server` and `llama-cli` are both absent from
`PATH`. An earlier revision of this report gave the wrong reason — it claimed
Homebrew does not offer it. That is false: `brew search llama` returns the
`llama.cpp` formula, and `brew info --json=v2 llama.cpp` reports it available
with `installed: []`. The real barrier is a scoped install plus a compatible GGUF
fixture, not availability. Its logging behaviour stays **unknown**, and is not
inferred from the other two runtimes.
**Owning decision:** is `llama-server` an active managed runtime target? That
decision now has a home — TASK-260828-3g87i4 in this story installs a pinned
llama.cpp build and stages equivalent-quantization Qwen GGUF weights.
**Resolution path:** if TASK-260828-3g87i4 lands, §3 is repeated against
`llama-server` on that fixture and this blocker closes on evidence. If the story
instead decides llama-server is not a deployment target, it closes as out of
scope. Either way the decision belongs to TASK-260828-3g87i4 and not to an
open-ended "once it is a real deployment target".

## Tests added

`tools/agents-infra/internal/modelharness/run_observability_test.go` — four
characterisation and negative tests. They add no capability; they pin the audited
facts so a later claim that the harness captures model I/O contradicts a test,
and so a silently added capture sink fails the build rather than quietly
invalidating this report.

Every test was verified against a mutant that should kill it. Production call
sites: `Run` → `runSupervised` → `runSupervisedAttempt` → `newFatalOutputWriter`
for the gate tests, `Run` → `runOnce` for the forwarding and sink tests.

| # | Mutant applied to `run.go` | Test | Result |
| --- | --- | --- | ---: |
| M1 | match on `marker[:14]` — keep only the head | `TestSupervisionMarkerGateRejectsNearMissOutput` | exit 1 — killed |
| M2 | match on `marker[14:]` — drop the head | `TestSupervisionMarkerGateRejectsNearMissOutput` | exit 1 — killed |
| M3 | match on `marker[:len(marker)-1]` — drop the trailing `(` | `TestSupervisionMarkerGateRejectsNearMissOutput` | exit 1 — killed |
| M4 | drop the carry in `fatalOutputWriter` (narrows the gate) | `TestSupervisionMarkerGateMatchesAcrossWriteBoundary` | exit 1 — killed |
| M5 | send child stdout to `io.Discard` | `TestRunForwardsChildBytesVerbatimAndAddsNoRecords` | exit 1 — killed |
| M6 | harness opens its own `os.CreateTemp` sink | `TestRunPersistsNothingItself` | exit 1 — killed |

The near-miss gate is bounded at **both ends of the match**, and that is a
correction to an earlier revision of this report, which asserted the property
without a mutant that could have falsified it. M2 — a genuine head truncation —
**survived** against the single "tail differs" near miss, because that input
lacks the marker's trailing `(` and so is rejected by a head-truncated match too.
The test now carries two near misses:

- `RuntimeError: [metal::malloc] Resource limit exceeded` shares the marker's
  whole head and diverges at the trailing `(`, killing M1 and M3.
- `FatalError: [metal::malloc] Resource limit (499000)` shares the marker's whole
  tail including the `(` and diverges in the leading exception name, killing M2.

Each case asserts at run time that it does not contain the full marker, so a near
miss cannot silently degrade into a positive case. M4 covers the opposite
direction — narrowing the gate by chunking — so widening and narrowing are both
bounded, and no result rests on a delete-only mutant.

## Commands run and exit codes

Rerun at this revision, each as a standalone process, exit code as reported by
the shell:

| Command | Exit |
| --- | ---: |
| `gofmt -l .` (in `tools/agents-infra`) | 0, no output |
| `go vet ./...` | 0 |
| `go build ./...` | 0 |
| `go test ./internal/modelharness/ -count=1` | 0 (1.252s) |
| `go test ./internal/modelharness/ -count=1 -run '<the four observability tests>' -v` | 0 — 4 tests, 2 subtests, all PASS |
| `go test ./... -count=1` (declared board validation) | 0 — root 86.887s, infra 146.187s, modelharness 1.689s, attachments 1.685s |
| M1–M6 mutation runs | 1 each, as intended; each named above |

Both commands declared in `task-board.config.json` under
`spawn.worktree_isolation.validation` were run as declared: `go vet ./...` and
`go test ./... -count=1`, each in `tools/agents-infra`, each exit 0.

No production code was modified in this revision — `git diff` on
`internal/modelharness/run.go` is empty after every mutant was reverted. The
tracked delta is `LOGBOOK.md`, `.research/260828_model-io-observability-through-harness.md`
and `tools/agents-infra/internal/modelharness/run_observability_test.go`.

## Artifacts

Raw evidence under `.temp/TASK-260828-28gdmq/` (gitignored), archived to the
board as `TASK-260828-28gdmq_evidence.tar.gz`: harness stdout/stderr captures,
the fatal-marker run, the Pi spawn event stream, the agents-infra session log,
the shared-broker runtime log, both shared-session event streams, and the
harness configs used.
