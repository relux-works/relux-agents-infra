# llama.cpp as a benchmark-gate candidate

Task: TASK-260828-3fgca3 (story STORY-260828-2faxgm)
Date: 2026-08-29 (revision 3; revisions 1-2 dated 2026-08-28)
Author role: developer
Built against: `main` at `fb85963`, merged into this story branch. The gate is
the single-invocation construction landed by STORY-260827-m30k8z; the
caller-authored record shape does not exist here and was not reintroduced.

**Revision 3 is the rework of review's F1-F4.** It rewrites the front of this
document as §0 and replaces §5's mutant table; §1-§4 are left as the record of
how the design got here, and where their text is now wrong §0 says so rather
than quietly editing it. The three blocking findings were one defect in three
costumes -- something the gate could not establish, spent as if it had been
established -- and they are fixed as one principle. Read §0 first.

## Verdict, in four parts (revision 2; see §0 for revision 3)

1. **G2 is closed, and the fix is that the gate asks the process.** The KV
   bound is no longer derived from argv at all for a runtime that reports one.
   `llama-server` reports `n_ctx` on `GET /v1/models`, measured, so its pin is
   that number and can never be `unbounded` ([§2](#2-g2--the-kv-bound-comes-from-the-running-process)).
2. **G1 is closed additively.** `-ub` / `--ubatch-size` are read as the second
   and third spellings of the prefill pin, exactly as `--chat-template-args`
   was added as a second spelling of reasoning effort.
   `unpinnableConditions` was **not** relaxed; it grew by one entry
   ([§3](#3-g1--a-third-spelling-of-the-prefill-pin)).
3. **The acceptance question is answered by a killed mutant, at the production
   entry.** With the pre-fix derivation restored, `benchmark-run` returns
   **exit 0, `accepted=true`** for a candidate running a 32,768-token context
   window compared against a genuinely unbounded baseline. With the fix, the
   same two processes are refused **exit 4** naming both readings
   ([§5](#5-mutants)).
4. **G4 is closed by a decision, not by a format branch.** `--model` demanded
   an MLX weight *directory* and `modelPath`/`modelDigest` were pins compared
   for equality, so a GGUF candidate could never satisfy them. The pin now
   identifies the **shared source of record**, and two different artifacts are
   admitted only under a `comparable` equivalence verdict bound to both of
   them by the digests the gate computed itself. Absence of that verdict is a
   refusal, a failed read of it is a different refusal, and MTP must be off for
   any scored comparison ([§4](#4-g4--the-model-pin-identifies-the-model-not-the-file)).

## 0. Revision 3 — the three bypasses, and the one rule

Review drove the shipped `benchmark-run` three times and got `exit 0`,
`accepted=true` three times. Every one of them is the same sentence: **absence
of evidence spent as evidence of the favourable case.** This gate has now been
rebuilt four times around that sentence, so revision 3 fixes the three sites as
one rule and then audits every other reading for the same shape (§0.4).

### 0.1 F1 — the verdict the caller wrote for itself

Revision 2's `--equivalence` named a JSON path. The gate read it, decoded it,
computed its SHA-256, carried that digest onto both attestations and folded it
into the observer's seal. Every one of those steps is correct and **none of them
authenticates anything**: the caller wrote the bytes. Review minted a document
naming an arbitrary source of record, both artifact digests *the gate had itself
computed*, `comparable`, and one generic note — and got `exit 0` with both
records carrying only that invented note, so the three measured non-equivalences
could be omitted from a decision this gate produced.

My pinning decision was right in shape and wrong in trust. The reviewer did not
reject pinning on a shared source of record under digest-bound evidence; it
rejected reading that evidence from a path the caller supplies.

**The fix: admission is bound to a decision the invocation cannot author for
itself.** `TrustedEquivalenceDecisions.shipped` is a fixed list, compiled into
the gate binary from versioned repository source, of the equivalence decisions
that have actually been taken. It holds exactly one entry, TASK-260828-3g87i4's:

| field | value |
| --- | --- |
| `sourceOfRecord` | `hf:orcarouter/Qwen3.8-27B-Uncensored-BF16` |
| `documentDigest` | `106edbf472177b055a149dda5cff3c8c86e13d1278a8ec508789f1197f09f962` |
| `requiredNonEquivalences` | the dropped MTP head, the vision-tower placement, F32-versus-bf16 norms — stated in full, in the store, where a reviewer reads them |
| `provenance` | the report and section the decision came from |

The document is `equivalence/qwen3-8-27b-uncensored.equivalence.json`, versioned
beside the package. Its artifact digests were **recomputed on this host the way
the gate computes them**: the MLX directory over `config.json` plus the
safetensors index, `1b10f3fe…88460b`; the GGUF over all 29 047 084 416 of its
bytes, `31756fca…24f8d6` — 95 s of streamed SHA-256, matching both
TASK-260828-3g87i4's number and the first-party LFS metadata.

The caller still supplies the document, because its contents have to travel into
both records. What the caller cannot do is decide that it counts. The reading
gained a fourth case, and the four are kept apart because collapsing any two of
them is this defect in another shape:

| reading | means | model-of-record |
| --- | --- | --- |
| `noneDeclared` | no verdict named — the same-format case | `artifact:<digest>` |
| `read(_, digest:)` | a **trusted** document, matched by SHA-256 | `source:<upstream>` |
| `unread(path:)` | named and unreadable — a failed read | `unread`, refused |
| `untrusted(path:digest:)` | read and decoded perfectly, and nobody decided it | `untrusted`, refused |

Refused twice on purpose. `BenchmarkRunCommand.execute` refuses an untrusted
reading before any launch; `admitProvenance` refuses it again off the
attestation, so a hand-authored attestation cannot reintroduce it. Then
`admitModelIdentity` repeats the lookup against the digest the attestation
carries, **before** every clause that reads a field out of the document, because
none of those clauses means anything until the document is evidence rather than
something the invocation wrote.

**What the trust store deliberately does not hold: a fixture entry.** An anchor
written so that `scripts/benchmark-gate-smoke.sh` could reach a cross-format
*acceptance* would be a decision nobody took, sitting in the production trust
store — F1 with a test's name on it. The consequence is paid rather than worked
around: the smoke proves the refusals and proves the trusted path is live
(§0.5), and the admitted class is proven by the contract suite against the
unstubbed shipped store and by one run against the real 29 GB pair (§0.5).

**The honest limit of the anchor's `requiredNonEquivalences`.** The digest fixes
the document's contents, so while the store and its file agree, demanding those
entries *of the document* is an equality that holds. What they guard is drift —
a later decision added with its digest updated and one measured difference
replaced by a generic note, which is F1's substitution exactly — and that is
reachable only by editing the store. It is covered by
`trustedDecisionMatchesItsDocument`, which reads the file off disk and reddens on
any drift, and by mutant **M-A4**. A deletion mutant of the record-side anchor
term alone would survive the suite by construction, because the document's own
list catches the same records. That is stated here rather than dressed up as a
kill.

### 0.2 F2 — a malformed field is not an absent field

`servingAnswer` mapped every missing, wrongly typed or non-positive `meta.n_ctx`
to `.notReported`, and `contextPolicy` spends `.notReported` as permission to
derive `unbounded`. Review ran a candidate with a real, finite 32 768-token
window that answered the JSON **string** `"32768"`: the attestation said
`notReported`, the record asserted `kv=unbounded`, and the pair matched a
genuinely unbounded baseline and was scored.

`RuntimeContextWindow.read(fromModelsEntry:)` separates the two facts at the
source. It lives in the contract library rather than in the driver so the suite
can attack every branch directly while the driver stays its only production call
site:

| answer | reading |
| --- | --- |
| no `meta`, or `meta` with no `n_ctx` | `notReported` — the runtime answered and named no bound. The measured MLX case, and the only one that may reach argv |
| `meta` that is not an object; `n_ctx` that is a string, boolean, float, `null`, array or object; a non-positive integer | `unread` — the field is there and the gate could not get a bound out of it. `kv=unread` is in `unpinnableConditions` and is refused |
| a positive JSON integer | `reported(n)` |

`as? Int` alone was not enough: `JSONSerialization` bridges every JSON number
*and* every JSON boolean to `NSNumber`, so `true` reads as 1 and `8192.5`
truncates. Zero is in the failed-read class rather than the absence one — 0 is
not a context window, and reading it as "no bound stated" is the same
substitution one step down.

### 0.3 F3 — a failed observation is not a negative observation

`speculationAnswer` mapped every status other than 200 to `.notReported`,
including 5xx and authorization failures, and the argv fallback then derived
`off`. Review configured a fixture as speculative, made `GET /slots` answer
**HTTP 500**, and the pass was scored as MTP-off. The gate has to *establish*
that MTP is off, not fail to establish that it is on.

`RuntimeSpeculation.read(slotsStatus:body:)` reserves `notReported` for the two
statuses that say **this route is not served here** — 404 for `mlx_lm.server`
and this prototype's router, 501 for `llama-server --no-slots` — and reads
everything else as a failure:

| answer | reading |
| --- | --- |
| 404, 501 | `notReported`; only here does argv speak |
| status 0, any other 4xx/5xx, a 200 that will not parse, a non-empty slot array naming no `speculative` | `unread`; `speculation=unread` is refused |
| 200 with no slots at all | `notReported` — nothing there to be speculating |
| 200 naming the field | `reported(_)`; any single slot drafting settles it |

The narrowing question is answered in both directions: a fix that read every
non-200 as a failure would refuse `mlx_lm.server`, the incumbent baseline, and
mutant **M-A7** proves the 404/501 arm is load-bearing.

### 0.4 The rule applied everywhere else — what the audit found

Every reading in the invocation was walked. Four more sites had the same shape
and are fixed; three are reported as audited and sound, with the reason.

| site | finding | disposition |
| --- | --- | --- |
| `declaredContextBound(inArgv:)` | `--ctx-size abc`, `-c 0` or a trailing `--ctx-size` fell through to "no bound was asked for". That silences `contextBoundNotHonoured`, which is **the only clause keeping a llama.cpp launch away from the argv fallback** when its server names no bound — F2 one level up | **fixed**: three cases (`none` / `pinned` / `unreadable`); `unreadable` refuses by name |
| `declaredSpeculation(inArgv:)` | a speculative flag present with an unreadable value — trailing, or `--spec-type=` — read as "the launch asked for nothing" | **fixed**: reported as a declaration, which refuses. The conservative direction |
| `hostIdentity()` | joined `"unknown-model"` / `"unknown-memsize"` on a failed `sysctl`. `hostIdentity` is compared for **equality**, so two records from two different machines that both failed the read carry the byte-identical pin and compare *equal* | **fixed**: refuses. A placeholder is not a reading |
| launcher version, `--python-bin`, `--candidate-binary` | `capture` merges stderr into the same pipe and ignored the exit status, so a failing `model-harness version` recorded its *error text* as the launcher revision; and revisions asked for and not answered merged as an empty dictionary | **fixed**: all three refuse. Asked-for-and-not-answered is a failed read |
| `appendDelta` | a metric neither pass measured | **already correct**: `verdict: "unmeasured"` plus a blocker, "unknown is not within threshold" |
| `pythonRevisions` / `swiftRevisions` returning `[:]` when the flag was *not* given | nothing is inferred from it, and `missingRevisions` still refuses an empty set | sound |
| `RuntimeContextWindow.notReported` falling back to argv | the residual of §2.4, unchanged: a *bounded* runtime that answers `/v1/models` and declines to name a bound reads as unbounded. Not the llama.cpp case, measured; closing it in general means refusing `mlx_lm.server` | stated, not closed |

**One finding is reported and not fixed, because it is a design decision rather
than a defect of this class.** `ProcessObservation` compares the executable path
at open and at close, and a launcher child that is an exec **stub** changes that
path mid-life: `/opt/homebrew/bin/python3` runs `…/bin/python3.14`, which
re-execs into `…/Python.app/Contents/MacOS/Python`. Whichever side of that exec
the 200 ms child-resolution poll lands on decides, so on a loaded host a healthy
pass is refused with *"the attestation … was opened and never closed"* —
reproduced twice (`.temp/real-pair-01.log`, `.temp/real-pair-03.log`) while the
same pair in the other ordering is accepted. It **fails closed**, so it is not
this rework's class, and the repair is a real tradeoff: `p_starttime` alone is
the pid-recycling defence, and dropping the path comparison would let a process
exec into something the recorded `observedExecutableDigest` does not describe.
It belongs to whoever runs the real 28 GB comparison, and it will make that run
flaky until it is decided.

### 0.5 What proves it, and where

| claim | proof |
| --- | --- |
| F1 refuses at the production entry | smoke: a caller-authored verdict — `comparable`, real upstream, both gate-computed digests, correct quantization labels, all three notes verbatim — exits **5**. Mutant **B3+A1** removes the lookup and the same invocation exits **0, accepted=true** |
| the trusted path is live at the production entry | smoke, two ways: the shipped document gets *past* the trust clause and is refused one clause further in for not covering the fixture artifacts; and beside a same-artifact pair it reaches `equivalenceEvidenceUnused`, which is deeper still |
| the admitted class, on the shipped store unstubbed | `admitsTheShippedDecision` in the contract suite, and one real run: the MLX 8-bit directory against the 29 GB Q8_0 GGUF through `benchmark-run`, **exit 0, `accepted=true`**, `decision.json` carrying all three non-equivalences, both attestations reading the document at `106edbf4…` (`.temp/real-pair-accepted.log`) |
| F2 refuses at the production entry | smoke: a finite 32 768 window answered as `"32768"` exits **4** on `kv=unread`, attestation `{"state":"unread"}`. Mutant **B1+B1b** restores the old reading and the same invocation exits **0, accepted=true** |
| F3 refuses at the production entry | smoke: `/slots` answering **HTTP 500** on both sides exits **4** on `reads "unread" for speculative decoding`, both attestations `{"state":"unread"}`. Mutant **B2** restores the old reading and the same invocation exits **0, accepted=true** |
| nothing was narrowed into uselessness | the 404-`/slots` runtime and the `--slots false` runtime are both still admitted at the production entry; **M-A7** shows what refusing them would cost |

### 0.6 F4 — the smoke script's lint

`shellcheck` exits **0** at its default level, down from 16 findings (12
pre-existing, 4 added by revision 2). Every `A && pass … || fail …` is an
`if`/`else` and no `$?` is read inside a condition.

## 1. Host and fixtures

Nothing else held this host's memory: `ps` showed no `llama-server`,
`mlx_lm`, `model-harness` or `mlx-swift` process before or after, and no
process belonging to another run was signalled. The **28 GB model was never
loaded**. Every llama.cpp measurement below uses the 676 MB
`Qwen2.5-0.5B-Instruct` Q8_0 fixture staged by TASK-260828-2jbufw, on an
OS-assigned ephemeral port outside `18000-18999`. The G4 probes in §5 are
refused *before* any launch, so they loaded nothing either.

Revision 2 adds the `/slots` probes of §4.4 on the same fixture and the same
kind of ephemeral port; both probe servers were killed and verified gone. The
smoke's runtimes are `fake-runtime.py` stand-ins and its weight artifacts are a
two-file fixture directory and a 50-byte file that exists only to be digested.

Runtime under test: pinned Homebrew `llama.cpp 0.3.0`, build **10621**, commit
`c1d0e7a00`, reported by the server itself as `build_info: b10621-c1d0e7a00`.

## 2. G2 — the KV bound comes from the running process

### 2.1 What llama.cpp reports, measured

`GET /v1/models` carries a `meta` block the MLX runtimes do not emit at all:

| Launch | `meta.n_ctx` | `meta.n_ctx_train` | `/props` `default_generation_settings.n_ctx` |
| --- | ---: | ---: | ---: |
| `--ctx-size 8192` | **8192** | 32768 | 8192 |
| no context flag | **32768** | 32768 | 32768 |

Two readings of the same fact, from two endpoints, agreeing. There is no
launch of this runtime that is unbounded.

`ModelsListing.make` — the Swift prototype's own `/v1/models` — emits
`id`/`object`/`created`/`owned_by` and no `meta`. `mlx_lm.server` likewise. So
the two MLX runtimes answer the question and name no bound, which is a
different fact from not being asked.

### 2.2 The decision, and why this one

The brief left the choice open: make the derivation runtime-aware, or read the
bound from the running server. **The bound is read from the running server.**

Runtime-awareness was rejected because it has nowhere trustworthy to get the
runtime's identity. The only anchors in a record are `record.runtime` — a
string the record declares — and the launch executable's path. Keying the KV
semantics off either turns "which runtime is this" into a claim, and the pin
exists precisely so that a declared condition is a *reading*. Worse, it does
not remove the false premise; it forks it into a per-runtime table that the
next runtime has to be added to, correctly, or the same class of defect
returns silently.

Reading the process removes the premise instead of maintaining it. It also
fits the construction that landed: the gate binary already performs a
`GET /v1/models` against the process it spawned, inside its own observation
window, at close, and already refuses a pass whose answer it could not read.
The bound rides on that exact exchange.

### 2.3 What was implemented

`RuntimeContextWindow` (`Sources/MLXSwiftRuntimeContract/RuntimeAttestation.swift`)
is the gate's first-hand reading, with three cases kept apart because
collapsing any two of them is the same defect in another shape:

| Case | Means | KV term |
| --- | --- | --- |
| `.reported(n)` | the runtime answered and named its bound | `kv=n` |
| `.notReported` | the runtime answered and named none | `--max-kv-size` value, else `unbounded` |
| `.unread` | the gate got no answer at all | `kv=unread`, refused |

It is a non-optional field of `RuntimeAttestation`, so it lives in the document
the *gate* authored, not the one the record authored, and
`admitProvenance(_:observing:)` re-derives the pin from it. A record still
cannot declare a policy by writing a string.

`BenchmarkRunCommand.servingAnswer` produces the window and `servedModelID`
from **one** exchange: a failed read yields `nil` model *and* `.unread`, never
`.notReported`. `Pins` moved below that call, because the KV bound is the
runtime's answer and not the launch's.

Three further consequences, each deliberate:

* **The value, not the spelling.** `kv=max-kv-size=4096` became `kv=4096`. A
  llama.cpp `n_ctx` of 8192 and an MLX `--max-kv-size 8192` are one reading of
  one condition; under the old rendering they could never compare equal and
  llama.cpp would have been uncomparable rather than wrong.
* **`kv=unread` joined `unpinnableConditions`.** The list grew; nothing was
  removed. A bound the gate failed to read is exactly as unusable as a prefill
  chunk left to a default.
* **A pinned bound the process did not honour is refused by name.**
  `AdmissionError.contextBoundNotHonoured`. Ask for `--ctx-size 8192`, run
  4096, and the pin still agrees on both sides — because the pin takes the
  process's number — so nothing above this clause can see it.

### 2.4 The residual, stated rather than implied

`.notReported` still falls back to argv, so a *bounded* runtime that answered
`/v1/models` and declined to say so would read as unbounded. That is not the
llama.cpp case — it always reports, measured in §2.1 — and closing it in
general would mean refusing `mlx_lm.server`, which reports nothing and is the
incumbent baseline. What *is* closed is the contradiction: a launch carrying
`--ctx-size` / `-c` whose server will not confirm the bound is refused, so a
llama.cpp launch cannot reach the argv fallback at all. `--max-kv-size` is
deliberately not symmetric there, because it belongs to runtimes measured to
report nothing.

## 3. G1 — a third spelling of the prefill pin

`value(of: "--prefill-step-size") ?? value(of: "--ubatch-size") ?? value(of: "-ub")`.
Additive, and `--batch-size` is deliberately **not** read: it is llama.cpp's
*logical* batch, default 2048, where the physical prompt-evaluation chunk is
`--ubatch-size`, default 512. Reading the first as the second would pin a
condition the launch never stated, at four times the value in effect.

`unpinnableConditions` was not relaxed. TASK-260828-2jbufw measured what the
relaxation admits — an unpinned mlx-swift launch (512) against an unpinned
`mlx_lm.server` one (2048), because all three runtimes derive the
byte-identical string — and the test
`doesNotRelaxTheUnpinnableConditions` asserts the list as a whole so a future
edit has to remove a clause in the open.

## 4. G4 — the model pin identifies the model, not the file

### 4.1 The constraint, restated

`benchmark-run --model` required an MLX weight *directory*:
`BenchmarkRunPins.modelDigest` read `config.json` and
`model.safetensors.index.json`, and the staged GGUF directory has a `.gguf` and
a `PROVENANCE.md` and neither of those. Worse, `modelPath` and `modelDigest`
were pins compared for **equality**, so a comparison in which one side serves
`…-MLX-8bit` and the other `…-OrcaRouter-Q8_0.gguf` could not satisfy them
however the digest was computed. Two production-entry refusals, exit 5 each,
taken before anything was launched.

That is not a bug in the derivation. It is a pin written about the local
artifact when the question is about the model.

### 4.2 The decision, and what it costs

**The pin identifies the shared source of record; byte-identity of the local
file is replaced by declared, digest-bound equivalence evidence — never by
nothing.**

Taken by the story, on the evidence TASK-260828-3g87i4 already produced: both
schemes cost **8.5 bits per weight**, the quantized-tensor sets match apart from
the MTP block, mean relative RMS against the shared BF16 source is **0.766**,
and the verdict is COMPARABLE with three named conditions.

`Pins.modelOfRecord` is what two runs must now agree on. It is not free text and
not the driver's opinion: it is
`RuntimeBenchmark.modelOfRecord(artifactDigest:observing:)` applied to the
reading the *gate* wrote onto `RuntimeAttestation.observedModelEquivalence`, and
`admitProvenance` re-derives it and refuses a record that declares anything
else. Two forms, and only two:

| Reading | Pin | Meaning |
| --- | --- | --- |
| `noneDeclared` | `artifact:<modelDigest>` | no verdict named. The artifact **is** the record, so byte identity is still exactly what two runs must share |
| `read(verdict, digest)` | `source:<sourceOfRecord>` | a verdict was read; `admitModelIdentity` then has to bind it to both artifacts |
| `unread(path)` | `unread` | a verdict was named and could not be read. Refused by name |

**Nothing was relaxed.** `modelPath`, `modelDigest` and `quantization` left
`firstMismatch`, and for the same-format class that is a no-op: with no verdict
the pin *is* the digest, so digest equality is still demanded by an equality
pin, and the other two are demanded by `admitModelIdentity` with the same
`pinMismatch` refusal. The pin-coverage table in `RuntimeBenchmarkTests` now
carries the expected refusal *per field* rather than assuming `pinMismatch` for
all of them, because a table that still claimed `pinMismatch` everywhere would
have passed while saying something false about which clause fires.

What the new class costs, all of it in `admitModelIdentity`:

1. both passes carry a verdict;
2. it is the **same** verdict, matched on the SHA-256 the gate computed over the
   document — so "the same evidence" is a fact about bytes, not two documents
   that happen to agree;
3. its verdict is `comparable`;
4. it names an artifact at **each side's gate-computed digest** — a verdict
   about some other pair of files cannot be aimed at these;
5. the quantization each record pins agrees with what the verdict records for
   that digest;
6. it declares at least one non-equivalence, and **every** one is carried in
   **both** records' `declaredAsymmetries`.

Clause 6 is what makes the three declared non-equivalences — the dropped MTP
head, the vision-tower placement, F32 versus bf16 norms — travel. The driver
copies them into both records before the passes run, so they land in
`decision.json` and in every report taken from it; admission refuses a record
that lost one; and `transcriptDigest` now covers `declaredAsymmetries`, so one
cannot be deleted after the pass that produced it without breaking the
observer's seal.

**Absence refuses structurally, not by a clause.** There is deliberately no
"evidence absent" refusal. With no verdict, `modelOfRecord` derives to
`artifact:<digest>` on both sides, the two digests differ by construction, and
the ordinary pin comparison refuses them. A separate clause there could never
fire, and a clause that cannot fail is not a second opinion — it is a line that
makes a gate look more careful than it is. A **failed read** is a different
fact and keeps its own refusal.

### 4.3 What the gate had to learn to read

* **A `.gguf` digest.** `wholeFileDigest(of:)` streams the whole file at 8 MiB.
  Not a header and not a prefix: a partial digest would let two differently
  quantized files share a pin, and the verdict binds to this number and to
  nothing else.
* **A `.gguf` quantization label.** It has no `config.json` and this gate has no
  GGUF header parser, so the label comes from the verdict entry **matched on the
  digest the gate computed**. A file with no verdict covering its digest is
  refused before launch. A *directory* still reads its own `config.json`, and a
  verdict that disagrees with it is refused — the two are read independently and
  have to say the same thing.
* **`--candidate-model`.** The two passes serve different artifacts, so
  `CommonPins` lost the model and gained `ModelPins` per pass. Omitted, it
  defaults to `--model`, so a same-format run is spelled exactly as before.

### 4.4 MTP — refused, not pinned

Condition 1 of the equivalence verdict: speculative decoding off, because the
MLX baseline has no MTP head to match it. Implemented as a **refusal** rather
than a pin comparison, because two speculating runtimes would agree on the pin
and still not be a migration result.

**The reading comes from the process, and the endpoint was chosen by
measurement.** On `llama.cpp 0.3.0` build `b10621-c1d0e7a00`, `Qwen2.5-0.5B`
Q8_0 fixture, ephemeral ports:

| Launch | `GET /slots` → `[0].params.speculative` | `GET /props` → `…params["speculative.types"]` |
| --- | --- | --- |
| no speculative flag | `false` | `"none"` |
| `--spec-type ngram-mod` | **`true`** | `"none"` |

`/props` reports the compiled default and does **not** move with the launch, so
reading it would report a speculating server as quiet. It is not read. That is
the "prove, or report nothing" rule applied to an endpoint that looked right.

`RuntimeSpeculation` keeps the same three cases apart as `RuntimeContextWindow`:
`reported(Bool)`, `notReported` (neither MLX runtime serves `/slots` at all),
and `unread`, which derives `speculation=unread` and is refused. Only
`notReported` reaches argv, and there `--spec-type` other than `none`, or any of
`--spec-draft-model` / `--model-draft` / `-md`, refuses the pass by name — all
four spellings read off that build's own `--help`, and `--draft` / `--draft-min`
deliberately not, because that build lists them as removed. `--spec-draft-threads`
and friends are *not* read as declarations: a flag that merely configures a
draft path does not by itself say drafting was asked for.

**The environment hole is closed at the entry, not documented away.**
`llama-server` reads `LLAMA_ARG_SPEC_TYPE`, and this gate's own environment is
what it hands the launcher, so an inherited variable would put the runtime under
test into drafting while every recorded argv showed nothing — the G2 defect one
condition along. `benchmark-run` refuses to launch when its environment carries
any `LLAMA_ARG_SPEC_*` or `LLAMA_ARG_DRAFT_*`, and refuses rather than scrubbing:
silently unsetting it would make the gate's environment differ from the
operator's for reasons no record shows.

The residual, stated: a runtime that speculates, serves no `/slots`, and was
configured by a file this gate never reads. That is the same residual class as
`RuntimeContextWindow.notReported`, and it is not the llama.cpp case, measured.

## 5. Mutants

Sixteen across the whole delta, each applied to the shipped source, built, run,
and reverted. **All sixteen killed, zero survivors.** `swift test` is the
324-test contract suite; the smoke is `scripts/benchmark-gate-smoke.sh`, 59
checks driving the shipped subcommands.

M1–M8 are the G1/G2 mutants and M9–M16 are G4's; both sets are revision 2's and
are left as the record of what those changes were bound by. Revision 3's ten are
in §5.1, and three of them are the acceptance questions of §0.

| Mutant | What it does | `swift test` | smoke |
| --- | --- | --- | --- |
| **M1** | the pre-fix derivation: KV off argv, absence as unbounded | exit 1, 10 red | exit 1 — **the false match is accepted, exit 0** |
| **M2** | `.unread` window spent as an absence | exit 1, 2 red | — |
| **M3** | *narrowing*: `kv=unbounded` declared unpinnable | exit 1, 12+ red | — |
| **M4** | *the relaxation this task was told not to make* | exit 1, 3 red | exit 1, 2 FAIL |
| **M5** | G1 undone: `-ub`/`--ubatch-size` unread | exit 1, 7 red | exit 1, 4 FAIL |
| **M6** | *narrowing*: every pinned bound must be confirmed | exit 1, 3 red | exit 1, 2 FAIL |
| **M7** | `--batch-size` read as the prompt-evaluation chunk | exit 1, 1 red | — |
| **M8** | **production call site**: the driver stops reading `meta.n_ctx` | **exit 0 — blind** | exit 1, 6 FAIL |
| **M9** | *widening*: the cross-format arm stops checking the verdict it was handed | exit 1, 12 red | — |
| **M10** | **production call site**: an unreadable verdict returns `noneDeclared` — a failed read spent as an absence | **exit 0 — blind** | exit 1, 1 FAIL |
| **M11** | *narrowing*: byte identity restored on top of the evidence | exit 1, **22 red** | — |
| **M12** | *narrowing*: any runtime that answers `/slots` is treated as speculating | exit 1, **20 red** | — |
| **M13** | **production call site**: the driver stops asking `/slots` | **exit 0 — blind** | exit 1 — **the speculating pair is accepted, exit 0** |
| **M14** | **production call site**: the declared non-equivalences stop travelling into the records | **exit 0 — blind** | exit 1, 3 FAIL |
| **M15** | `speculation=unread` spent as `off` | exit 1, 2 red | — |
| **M16** | *widening*: a record may declare its own model of record | exit 1, 4 red | — |

Four of these carry the G4 argument.

**M13 is the acceptance question for the MTP condition, and it is answered at
the production entry.** Delete the `GET /slots` read from
`BenchmarkRunCommand.servingAnswer` and all **324** contract tests still pass —
they hand the reading in directly — while the smoke's speculating pair, two real
spawned processes both reporting `params.speculative: true`, driven and measured
and judged by the shipped `benchmark-run`, comes out:

```
FAIL  a runtime reporting speculative decoding is refused: expected exit 4, got 0
```

Exit 0 is `accepted=true`. Two runtimes drafting, scored against each other as a
runtime difference, green.

**M10 and M14 are the same seam in the other two directions.** Both leave the
whole contract suite passing and are visible only from the shipped subcommand:
M10 turns an unreadable verdict into an absent one, and the smoke catches it
because the refusal stops naming the failed read; M14 stops the non-equivalences
travelling, and the G4 acceptance check goes from exit 0 to exit 4 because the
records no longer carry what the verdict declared.

**M11 and M12 are the narrowing pair, and they are the ones that say what the
*admitted* class is.** M11 puts `modelDigest` back into `firstMismatch`, so a
cross-format pair can never be admitted however good its verdict is: **22** red,
including `admitsACrossFormatPairUnderEvidence`. M12 reads every `reported`
speculation state as `on`, which makes llama.cpp itself inadmissible: **20** red,
including `admitsARuntimeThatReportsNoSpeculation` and
`admitsAnExplicitlyDisabledSpeculation`. A delete-only mutant would have said
the clauses exist; these say what they cover, in both directions.

### 5.1 Revision 3's mutants

Twelve more, each applied to the shipped source, built, run, and reverted. **All
twelve killed, zero survivors.** `swift test` is the 351-test contract suite; the
smoke is `scripts/benchmark-gate-smoke.sh`, 68 checks driving the shipped
subcommands against the `xcodebuild`-free Release product.

| Mutant | What it does | `swift test` | smoke |
| --- | --- | --- | --- |
| **B3+A1** | **F1 undone at the production entry**: the trust lookup is deleted from `equivalenceReading` and neutered in `admitModelIdentity` | exit 1, 2 red | exit 1, 5 FAIL — **the caller's own verdict is accepted, exit 0** |
| **B1+B1b** | **F2 undone**: a malformed or non-positive `n_ctx` is an absence again | exit 1, **23 red** | exit 1, 3 FAIL — **the 32 768 window pinned `unbounded` and was accepted, exit 0** |
| **B2** | **F3 undone**: every non-200 from `/slots` is an absence again | exit 1, **20 red** | exit 1, 3 FAIL — **the HTTP 500 pair was scored as MTP-off, exit 0** |
| **A1** | the trust guard in `admitModelIdentity` alone stops refusing | exit 1, 2 red | — |
| **A2** *(M11, narrowing)* | byte identity restored on top of the evidence: `modelDigest` back in `firstMismatch` | exit 1, **18 red** | — |
| **A3** *(M12, narrowing)* | any runtime that answers `/slots` is treated as speculating | exit 1, **16 red** | — |
| **A4** *(drift)* | the shipped trust store requires a note its own document does not carry | exit 1, 4 red | — |
| **A5** | the declared non-equivalences stop being demanded of the records | exit 1, 4 red | — |
| **A6** | a context flag the gate cannot read goes back to being an absence | exit 1, **15 red** | — |
| **A7** *(narrowing)* | no status means "route absent", so every MLX runtime reads `unread` | exit 1, 3 red | — |
| **A8** *(narrowing)* | revision 2's M3 re-run here: `kv=unbounded` declared unpinnable, so an MLX-against-MLX pair is refused | exit 1, **73 red** | — |
| **A9** | revision 2's M4 re-run here — **the relaxation this task was told not to make**: `prefill-step=unpinned` dropped from `unpinnableConditions` | exit 1, 3 red | — |

**The three acceptance questions are B3+A1, B1+B1b and B2, and all three are
answered at the production entry.** Each restores exactly the reading review
exploited, and each turns the corresponding smoke check from a refusal into
`exit 0, accepted=true` — the same three results review obtained from the
shipped binary, reproduced here from the shipped binary, and closed by the
delta.

**A2 and A3 are the narrowing pair for G4 and MTP**, unchanged in intent from
revision 2's M11 and M12 and re-run against revision 3's source: 18 and 16 red
respectively, including the positive `admitsACrossFormatPairUnderEvidence` and
`admitsTheShippedDecision`, so the *admitted* class is pinned in both
directions rather than only the refused one. **A7** is the third narrowing
mutant and the one that costs the incumbent: read every `/slots` answer as a
failure and `mlx_lm.server`, which serves no such route, becomes inadmissible.

**A8 and A9 re-answer `unpinnableConditions` against *this* source rather than
citing revision 2.** The list is byte-identical to revision 2's — it was neither
relaxed nor added to by this delta — and both directions still hold: narrowing
it by one entry reddens **73** tests, and making the relaxation the task was
told not to make reddens 3, including `doesNotRelaxTheUnpinnableConditions`,
which asserts the list as a whole so a future edit has to happen in the open.

**A6 is the audit turned into a mutant.** Fifteen red for a one-line
permissiveness in what the gate reads off the *launch* rather than off the
process — the same shape as F2, one level up, and the reason the audit in §0.4
was worth doing rather than asserting.

## 6. Gates

Every command run directly, exit code as reported.

Revision 3's run, each command run directly as its own process.

| Gate | Command | Result |
| --- | --- | --- |
| package build | `swift build --build-tests` | exit **0** |
| contract suite | `swift test` | **351 tests / 29 suites**, exit **0** |
| production-entry smoke | `scripts/benchmark-gate-smoke.sh`, against the Release product | **68 checks, 0 failures**, exit **0** |
| the real pair at the production entry | `benchmark-run` over the MLX 8-bit directory and the 29 GB Q8_0 GGUF under the shipped decision | **exit 0, `accepted=true`** |
| Release product | `swift build -c release` | exit **0** |
| Swift lint | `xcrun swift-format lint --strict --recursive Sources Tests` | exit **0** |
| shell lint | `shellcheck scripts/benchmark-gate-smoke.sh` (default level, not `-S warning`) | exit **0**, from 16 findings |
| whitespace | `git diff --check` | exit **0** |
| mutants | 12 applied, built, run, reverted | **12/12 killed**, 0 survivors |

Revision 2's row for `xcodebuild build -configuration Release` is not repeated:
this delta touches no Metal shader library and the smoke drives the
`swift build -c release` product, which is the binary every check above judged.

Not run, and why: the real 28 GB llama.cpp-vs-MLX comparison. It is now
*expressible* — that is this task's acceptance criterion and it is met at the
production entry — but running it is an hour of exclusive host time against a
28 GB model and a 29 GB GGUF, and it decides a migration rather than an
admissibility question. It belongs to the story's measurement task, not to this
one.

## 7. Files

### 7.1 Revision 3

| File | Change |
| --- | --- |
| `equivalence/qwen3-8-27b-uncensored.equivalence.json` | **new.** The one equivalence decision this repository trusts, with both artifact digests recomputed on this host |
| `Sources/MLXSwiftRuntimeContract/TrustedEquivalenceDecisions.swift` | **new.** `TrustedEquivalenceDecision`, the shipped list, and the lookup by document digest |
| `Sources/MLXSwiftRuntimeContract/ModelEquivalence.swift` | `ModelEquivalenceReading.untrusted(path:digest:)`, its coding, and the four-case doc |
| `Sources/MLXSwiftRuntimeContract/RuntimeAttestation.swift` | `RuntimeContextWindow.read(fromModelsEntry:)` and `RuntimeSpeculation.read(slotsStatus:body:)` — F2 and F3 moved out of the driver so the suite can attack them |
| `Sources/MLXSwiftRuntimeContract/RuntimeBenchmark.swift` | the trust clause and the two drift clauses in `admitModelIdentity`; `modelOfRecordUntrusted`; `contextBoundUnreadable`; `equivalenceEvidenceUntrusted`; `trustedDecisionDisagrees`; `DeclaredContextBound`; the unreadable speculative-flag reading; `trusting:` on `admit` |
| `Sources/mlx-swift-runtime-prototype/BenchmarkRunPins.swift` | `equivalenceReading` performs the trust lookup; `hostIdentity()` refuses a placeholder |
| `Sources/mlx-swift-runtime-prototype/BenchmarkRunCommand.swift` | the untrusted-verdict refusal before launch; the trusted decision's notes folded into `mandated`; the launcher-version and revisions refusals; both readers replaced by their contract-library call sites |
| `Tests/.../RuntimeBenchmarkTrustedEquivalenceTests.swift` | **new.** 11 tests against the **unstubbed** shipped store, two of them positive |
| `Tests/.../RuntimeObservationReadingTests.swift` | **new.** 16 tests over the F2/F3 readings and the two launch-side readings the audit found |
| `Tests/.../RuntimeBenchmarkModelOfRecordTests.swift` | a fixture trust registry, so the clauses *below* the trust lookup stay reachable |
| `scripts/benchmark-gate-smoke.sh` | the three acceptance questions; the speculation section moved to a same-format pair; `--n-ctx-string` and `--slots-status` on the stand-in; all 16 lint findings fixed |
| `examples/model-harness.benchmark.toml`, `README.md` | the trust model, and what `--equivalence` does and does not decide |

### 7.2 Revision 2

| File | Change |
| --- | --- |
| `Sources/MLXSwiftRuntimeContract/ModelEquivalence.swift` | **new.** `ModelEquivalence`, its `Verdict` and `Artifact`, and the three-case `ModelEquivalenceReading` |
| `Sources/MLXSwiftRuntimeContract/RuntimeAttestation.swift` | `RuntimeContextWindow`; `RuntimeSpeculation`; `observedContextWindow`, `observedSpeculation` and `observedModelEquivalence` on the attestation |
| `Sources/MLXSwiftRuntimeContract/RuntimeBenchmark.swift` | `contextPolicy(derivedFrom:observing:)`; `modelOfRecord(artifactDigest:observing:)`; `speculationPolicy(derivedFrom:observing:)`; `declaredSpeculation(inArgv:)`; `admitModelIdentity`; `Pins.modelOfRecord` and `Pins.speculation`; eleven new refusals; `declaredAsymmetries` folded into `transcriptDigest` |
| `Sources/mlx-swift-runtime-prototype/BenchmarkRunPins.swift` | `wholeFileDigest(of:)`; `modelDigest(artifact:)`; `quantizationLabel(artifact:equivalence:digest:)`; `equivalenceReading(path:)` |
| `Sources/mlx-swift-runtime-prototype/BenchmarkRunCommand.swift` | `--candidate-model` and `--equivalence`; per-pass `ModelPins`; the `LLAMA_ARG_SPEC_*` environment refusal; `speculationAnswer` reads `/slots`; the verdict's non-equivalences copied into both records |
| `Sources/mlx-swift-runtime-prototype/BenchmarkPass.swift` | `slots(timeout:)`, at the server root rather than under `/v1` |
| `Tests/.../RuntimeBenchmarkContextBoundTests.swift` | new at G1/G2, 17 tests, mostly negative |
| `Tests/.../RuntimeBenchmarkModelOfRecordTests.swift` | **new.** 22 tests across two suites: the model-of-record clauses and the speculation ones, two of them positive and the rest negative |
| `Tests/.../RuntimeBenchmarkTests.swift` | `variantPins`; the pin-coverage table carries the expected refusal per field |
| `scripts/benchmark-gate-smoke.sh` | section 3 (8 KV checks) and section 4 (13 G4 checks); the stand-in learned `--n-ctx` and `--slots` |
| `examples/model-harness.benchmark.toml` | `profiles.qwen-benchmark-llamacpp`, `--spec-type none`, and the cross-format invocation |
| `README.md` | what the gate observes and refuses; why the model pin is the source of record; why `/slots` and not `/props` |
