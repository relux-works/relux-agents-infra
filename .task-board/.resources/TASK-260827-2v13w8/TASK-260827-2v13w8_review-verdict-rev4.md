# TASK-260827-2v13w8 review verdict — Change Request revision 4

## Verdict

**ACCEPTED.** Accept `CR-TASK-260827-2v13w8-4` revision 4 as adequate evidence
for the conservative migration decision: **REJECT the MLX Swift migration and
keep Python `mlx-lm` as the default runtime.** The decision rests on one
reproduced blocker: the Swift runtime's `long_prompt_8k` scenario-local peak
physical footprint is 38,801,396,520 B versus 33,705,153,440 B for Python,
ratio **1.1512x**, outside the required `<= 1.10x` band. The same axis was
1.144x in revision 3.

Review surface:

- Base OID: `3f313d9175f2ada9b9ab3320ab524c0918f9daac`
- Candidate tree OID: `7f65667945a8087e883e6c82eb9fc8b402cce917`
- Repository delta: present, 69 paths
- The managed worktree had no drift against the candidate tree during review.
- `git diff --check` over the exact base-to-candidate delta passed.

## Gate-defeat result

Round 3's ordinary-caller forgery is closed by the production construction.
The shipped binary has no `benchmark-attest` subcommand and `benchmark-run`
accepts configuration and execution bounds, but no record, scenario result,
TTFT, or other caller-authored measurement. `benchmark-compare` remains a
file-based replay and has no accepting exit: an admitted replay always exits 3.

I reran the production models-only shape using two actual runtime children that
answered `/v1/models` but returned HTTP 404 for completions. The single
`benchmark-run` parent drove them itself and exited 4 with:

> record "reviewer-placeholder-baseline" contains no scenario that ever
> completed a chat completion; the process under observation answered other
> endpoints and served nothing, so there is no benchmark here to judge

No decision was produced. The caller-authored measurement half of the old
attack is now inexpressible through `benchmark-run`; attempts to pass
`--baseline-record`, `--candidate-record`, `--scenarios`, or `--ttft` are usage
errors. The successful full production smoke independently covered the same
shape and all four arguments.

Kernel evidence, not record-only evidence:

- The current Release product hashes to
  `8a517b10e6a74793dd47d33d07b1b08275863f3fb7e8cfb11880a14b71014f91`.
- During the placeholder attack, `ps` showed the live `benchmark-run` parent
  and a live models-only Python runtime child; `lsof -d txt` mapped the parent
  to the Release product above. The production command refused the pair at
  exit 4.
- During an independent short real-model pass, live `benchmark-run` PID 35209
  mapped by `lsof -d txt` to that same Release product; both Python and Swift
  served `short_prompt`, after which the deliberately skipped required
  `tool_call` scenario caused exit 4.
- A separate live Swift `serve` PID 40927 mapped by `lsof -d txt` to the same
  file; hashing the kernel-mapped path returned the same `8a517b10...` digest.
  It loaded the pinned model through `MLXLLM.LLMModelFactory`, reached `ready`,
  and shut down cleanly on SIGINT.
- The independent real-model pass's gate-written candidate attestation reports
  `observedExecutableDigest == gateBinaryDigest == 8a517b10...`, consistent
  with those kernel mappings.

The archived production attestations are internally consistent with this
reproduction: both name `8a517b10...` as `gateBinaryDigest`; the candidate's
`observedExecutableDigest` is also `8a517b10...`, while the baseline correctly
names the kernel Python executable digest instead.

## Judgement on the two disclosures

### L-3 survivor

Acceptable, because it is evidence of an explicitly withdrawn claim rather
than an unwitnessed promised clause. L-3 modifies and rebuilds the measuring
binary so that the trusted measurement code itself invents both the transcript
and the numbers. With no external signature, trusted launcher, or independently
anchored expected digest, a binary cannot prove that its own code was not
changed. Revision 4 does not claim that boundary: it claims resistance to an
ordinary caller of the shipped artifact, identifies the exact artifact by
digest, and demonstrates the modified-build limit with a named surviving
mutant. The shipped-artifact negative shapes that the decision actually relies
on are caught, including narrowing N-P and production bypasses P-1/P-2/P-3.

Treating L-3 as caught because an unrelated fixture made the suite red would
have been false evidence. Recording it as a survivor is the correct result.

### Missing production `benchmark-run` exit status

Acceptable for this revision's conservative rejection, with the gap retained in
the report. The `nohup` wrapper discarded the process status, but the run
completed both passes and wrote `decision.json` with `accepted=false` and the
single 8k footprint blocker. The production source has one direct mapping after
writing that decision: `accepted ? 0 : 3`. Replay of the sealed session exits 3
with the identical decision, and the production smoke independently exercises
`benchmark-run` exits 0, 2, 3, and 4. Repeating an hour-long measurement would
recover missing wrapper telemetry, not change the measurement or the
conservative outcome. A future long-run wrapper should capture `$?`, but that is
not rework required for this evidence packet.

## Decision and reporting review

- The favourable results are not hidden or overstated. The report records Swift
  winning `short_prompt` on TTFT, prefill, decode, and footprint; 8k decode;
  soak median and aggregate throughput; and first served completion.
- Soak is correctly described as a bounded 20/20 pass with zero failures for
  both runtimes, not as proof of unbounded stability.
- Readiness semantics are distinguished: Swift answers 503 until weights are
  resident, while `mlx_lm.server` lists the model in 2.9 s before loading it.
- The revision-3 TTFT and prefill blockers are withdrawn because they did not
  reproduce. The report does not turn a noisy single run into a migration win;
  it requires repeats before any future acceptance.
- The capacity result is honest. The raw 1.12 tok/s observation remains in the
  table, but the report identifies its weak basis (16 decode tokens) and the
  self-induced pressure at load average 35.21 and 49.4 GiB resident on a 64 GiB
  host. It is framed as evidence that this capacity run was much slower, not as
  a stable rate estimate.
- Exact Python, MLX, Swift package, model-harness, host, model, prompt,
  thresholds, config, transcript, and executable revisions/digests are recorded,
  along with the verbatim benchmark command and replay command.
- The 75k capacity, tool-call parity, readiness behavior, and bounded soak
  outcomes are all recorded for both runtimes.
- No installed runtime profile was changed; Python remains the default.

## Independent validation

All commands below were run in the candidate worktree after confirming it
matched candidate tree `7f656679...`:

| Validation | Result |
| --- | --- |
| Release artifact SHA-256 | `8a517b10...`, matches report |
| Placeholder production attack | exit 4, no decision |
| `scripts/benchmark-gate-smoke.sh` with absolute `BINARY` and `OUT` | 39 checks, 0 failures, exit 0 |
| Short real-model single-invocation identity probe | both runtimes served; deliberate missing-scenario refusal exit 4 |
| Live Swift kernel executable mapping | Release product, SHA-256 `8a517b10...` |
| `swift test` | 285 tests in 24 suites, exit 0 |
| `xcrun swift-format lint --strict --recursive Sources Tests` | exit 0 |
| `shellcheck -S warning scripts/*.sh` | exit 0 |
| exact base-to-candidate `git diff --check` | exit 0 |

An initial reviewer smoke attempt passed relative `BINARY`/`OUT` paths into a
config whose launcher runs from another directory and therefore aborted before
serving (`launcher status 256`). The rerun used absolute paths, as the shipped
script's contract requires, and completed 39/39. This was an invocation error
in the review probe, not a product failure.

No code was modified by review. The temporary live runtimes were waited to
completion or stopped cleanly before this verdict.
