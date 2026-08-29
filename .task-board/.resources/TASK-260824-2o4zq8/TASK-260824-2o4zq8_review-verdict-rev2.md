# TASK-260824-2o4zq8 — Review Verdict (CR rev2): ACCEPTED

Reviewer run `RUN-260824-274367` against `CR-TASK-260824-2o4zq8-2` revision 2
(base `ba0d95d`, candidate tree `0855f9c`, `repository_delta=present`).

Reviewed surface: the CR delta (`LOGBOOK.md`, +19) **plus** the task deliverable
itself, which lives in the branch commits `0246afc` "Resolve canonical vendor
targets" (17 files, +2345) and `ba0d95d` "Lock hosted targets across wrapper
delimiter" (6 files, +159/-5). `git diff 0855f9c -- . ':!.temp'` is empty, so the
tree every probe below ran against is exactly the reviewed candidate.

## Verdict

`accepted`. The single blocking finding from the rev1 verdict is fixed, the fix
is correct for the right reason, and the bound is proven by a narrowing mutant I
built and ran myself. The low finding is fixed too. No new blocking defect
survived the attack pass.

## Finding 1 (rev1, blocking) — FIXED and independently verified

`lockCodexTargetArguments` / `lockClaudeTargetArguments` now emit the first `--`
and keep inspecting; only a second `--` terminates identity selection
(`canonical_target.go:419-429`, `:526-536`). Pi is untouched.

The fix rests on a factual claim — *the second delimiter is the provider's real
operand boundary* — so I did not take it on trust. I drove the real provider
CLIs installed on this machine:

| Probe | Result | Meaning |
| --- | --- | --- |
| `claude --model claude-opus-5 -- --model NOTAREALMODEL-xyz` | replies "I am on **Opus 5** (`claude-opus-5`)" | post-`--` `--model` is prompt text, not an option |
| `claude --model claude-opus-5 -- --nonexistent-flag-xyz` | treated as chat text, no `unknown option` error | commander honours `--` |
| control: `claude --nonexistent-flag-xyz` | `error: unknown option` | the error shape exists when parsing does happen |
| `codex exec -- --profile NOPE-XYZ "hi"` | `error: unexpected argument 'NOPE-XYZ' found` | `--profile` became a positional; clap honours `--` |
| `codex --cd -- --nonexistent-flag-xyz` | `error: a value is required for '--cd <DIR>'` | no unlocked option can swallow the delimiter and re-open parsing |

So the lock's boundary is now exactly the wrapper's boundary
(`parseCodexWrapperArgs` consumes delimiter #1; `normalizeCodexExplicitSelections`
breaks at #2; `parseClaudeWrapperArgs` consumes #1 and Claude itself honours #2).

### Narrowing mutant (mine, not the producer's)

Disposable module copy under `/tmp`; both lock bodies narrowed back to
`return append(out, args[i:]...)` at the first delimiter. Source worktree never
mutated.

`go test . -run TestRunTargetRejectsHostedIdentitySelectorsAfterWrapperDelimiter -count=1`
→ **FAIL, all 8 subtests**. Four returned `<nil>` (live launch of the diverging
identity); four returned the Codex wrapper's own conflict error with the wrong
code and field. The test asserts both `target_identity_conflict` and the exact
`provider_args.*` field, so it kills the narrowing, not just a deletion.

### Live bypass matrix through the production binary

Real `agents-infra` built from the candidate, isolated `HOME`/`PATH`, recording
fake `codex`/`claude`, `AGENTS_INFRA_CALLER_CWD` set — i.e. exactly what the
installed wrapper's `exec "$TARGET" target <entrypoint> "$@"` produces.

Refused with `target_identity_conflict`, provider never executed, config bytes
unchanged: Codex `-mother`, `--model=other`, `-p work`, `-pwork`,
`--profile=work`, `--config model="other"`, `-c=profile="work"`,
`--model-reasoning-effort=low`; Claude `--model=other`, `--effort=low`,
`--effort LOW` (case variant, fail-closed) — every one of them placed *after*
the wrapper delimiter.

Accepted and normalised to a single locked selector (no duplicate in argv):
`-- --model gpt-5.6-sol`, `-- -mgpt-5.6-sol`, `-- --model=gpt-5.6-sol`,
`-- -c model="gpt-5.6-sol"`, `-- -c model_reasoning_effort="high"`,
Claude `-- --model claude-opus-5 --effort high`.

### Delimiter-desync attacks (new; not in the rev1 pass)

The one way to defeat this design is to make the lock count two delimiters while
a downstream parser counts one, or to have a parser eat a `--` as an option
value. All fail closed:

| Attack | Result |
| --- | --- |
| `anthropic-infra -- --permission-mode -- -- --model other` | `--permission-mode value "--" is invalid` |
| `anthropic-infra -- --permission-mode acceptEdits -- --model other` | `target_identity_conflict` |
| `anthropic-infra -- --model -- --model other` | `target_identity_conflict` |
| `anthropic-infra -- --effort -- -- --effort low` | `target_identity_conflict` |
| `openai-infra -- --model -- gpt-5.6-sol` | `a value is required for the Codex argument --model` |
| `openai-infra -- -c -- -- --profile work` | `a value is required for the Codex argument -c` |
| `openai-infra -- --sandbox -- -- --model other` | `a value is required for the Codex argument --sandbox` |
| `openai-infra -- --cd -- -- --profile work` | launches; `--profile` lands after Codex's own `--`, and clap refuses to let `--cd` swallow the delimiter (probe above) |

The structural reason it holds: the lock only ever drops identity selectors and
their values, and it refuses `--` as a value on every path, so the Nth `--` in
the lock's input is the Nth `--` in its output. Lock and wrapper stay aligned.

## Finding 2 (rev1, low) — FIXED

The unconditional `RenderCanonicalTargetLaunchPlan` dump to stderr is gone from
`runTarget`. Pinned by an assertion in
`TestRunTargetDispatchPreservesCallerCWDAndLocksBeforeProviderSideEffects`, which
now fails if an ordinary launch writes anything to stderr. `--print-config`
remains the explicit non-launching surface and still writes to stdout.

## Independent checks beyond the rework

**Alias-set narrowing mutant (mine).** `canonicalTargetLauncherNames` narrowed to
`{"openai-infra"}` → `TestSetupRepairsAndVerifyNarrowsEveryCanonicalTargetAlias`
fails for `anthropic-infra` and `qwen-infra`,
`TestCanonicalAliasRefusesMissingAndNonRegularSibling` fails, and
`TestSetupGlobalDoesNotInstallCLIWrapper` fails. Section 7.8's per-alias
narrowing control is real. (Two `TestRunSetupLocalRejectsPrimarySessionFlagsForHomeTarget`
failures in that run are a copy artifact of the isolated module — they do not
reproduce in the clean tree.)

**Fail-closed classes, driven live through the binary:**

| Class | Config | Result |
| --- | --- | --- |
| 7.9 no legacy bypass | valid `[agents.codex.primary_session]`, no mapping | `unknown_entrypoint: ... legacy provider policy is not a fallback`, no launch |
| 7.3 unknown target | `openai-infra = "nope"` | `unknown_target` + entrypoint/target/field context |
| 7.3 alias/vendor mismatch | `openai-infra` → anthropic target | `invalid_target: entrypoint vendor does not match its target vendor` |
| 7.7 malformed source | truncated table header | `invalid_project_configuration ... field project_config: parse` |
| 7.7 unreadable source | `chmod 000` | `invalid_project_configuration ... field project_config: read` — a read failure, **not** reported as an absent mapping |

**Section 5 machine surface:** `compose --mode primary-session --entrypoint`
applies the same lock (post-delimiter `--profile` → structured
`error.code=target_identity_conflict` with `context.{entrypoint,target,field,source}`
and `remediation`); the `target` object carries entrypoint/name/source/vendor/
environment/model/reasoning/profile/profile_provider/endpoint; `--agent` +
`--entrypoint` together is refused; legacy `--agent` plans still have **no**
`target` key.

**Section 3.5 legacy policy still applies:** with an ancestor config carrying
yolo policy, canonical alias launches inherit
`--dangerously-bypass-approvals-and-sandbox` / `--dangerously-skip-permissions`
exactly as the legacy path does; with an isolated config they do not. Correct
per contract — target identity is layered above, not instead of, legacy policy.

## Gates re-run in this session

| Gate | Command | Result |
| --- | --- | ---: |
| Build | `go build ./...` | clean |
| Vet | `go vet ./...` | clean |
| Format | `gofmt -l .` | clean |
| Suite (internal) | `go test ./internal/... -count=1` | ok — infra 126.4s, attachments 0.8s |
| Suite (main) | `go test . -count=1` | ok — 82.6s |
| Focused | `go test ./... -run 'Delimiter\|Canonical\|Target\|Qwen' -count=1` | ok |
| Qwen acceptance | `go test ./... -run Qwen -count=1 -v` | **8 PASS, 0 SKIP** |
| Whitespace | `git diff --check`, `git diff --cached --check` | clean |

The LOGBOOK "1941" claim that the Qwen acceptance tests report `PASS` and not
`SKIP` reproduces here: 8 named tests, zero skips.

## CR rev2 delta

`LOGBOOK.md` +19: entries 2013 (the fix), 2002 (the rev1 root cause), and 1941
(the worktree Pi-fixture resolution). All three are accurate against the code and
against my own reproductions. Nothing in them overstates the evidence.

## Non-blocking note for follow-up

The delimiter semantics now diverge deliberately across the three aliases — for
`openai-infra`/`anthropic-infra` the first `--` is a wrapper delimiter and
identity locking continues through it, while for `qwen-infra` the first `--` is
Pi's message-operand boundary. That is the correct behaviour and it is pinned by
`TestCanonicalQwenDelimiterRemainsMessageOperandBoundary` and recorded in the
LOGBOOK, but README/SKILL do not mention it. Worth a sentence in the operator
docs whenever those files are next touched; not a reason to hold this work.

## Definition of Done

All items met. In particular "Drive every Section 7 refusal through production
parse/compose/alias/setup/verify paths with narrowing controls" and "Gating,
refusing, validating, authorizing, or attesting behavior covered by negative
tests that fail when the gate admits what it must reject" — the two items the
rev1 verdict marked not met — are now satisfied, verified by an independent
narrowing mutant that reddens all eight production-entry cases.

Acceptance evidence is handed to the commit-owning mover. This reviewer supplied
no `commit_ack`.
