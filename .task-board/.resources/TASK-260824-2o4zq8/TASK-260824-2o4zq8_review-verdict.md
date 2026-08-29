# TASK-260824-2o4zq8 — Review Verdict: CHANGES REQUESTED

Reviewer run against `CR-TASK-260824-2o4zq8-1` revision 1 (candidate tree
`7e84ffc0ebc6c12c9722e3ceceb58a74ba4bbfc0`, base `0246afcdfcc72f19a7ecfa846b2d7e13a5d3b0ba`).

The Change Request delta is `LOGBOOK.md` only; the task deliverable itself is
the base commit `0246afc "Resolve canonical vendor targets"` (17 files, +2345).
Both were reviewed. The LOGBOOK entry is accurate and is not the reason for this
verdict.

## Verdict

`changes_requested` -> `to-dev`. One CONFIRMED defect defeats contract
Section 3.6 identity locking and falsifies Section 5 provenance output at the
production alias entrypoint; it is not covered by any test.

## What was verified green

| Gate | Command | Result |
| --- | --- | --- |
| Build | `go build ./...` | clean |
| Vet | `go vet ./...` | clean |
| Full suite | `go test ./... -count=1` | ok 84.9s / 2.5s / 129.8s |
| Canonical/Qwen/Target subset | `go test ./... -run 'Canonical\|Qwen\|Target' -v` | 34 PASS, **0 SKIP** |
| Formatting | `gofmt -l .` | clean |
| Whitespace | `git diff --check` | clean |

The LOGBOOK claim that the Qwen acceptance tests now report `PASS`, not `SKIP`,
reproduces on this machine. `officialPiAsset`'s primary-checkout fallback works
from inside the Story worktree.

## Narrowing mutants driven through production entrypoints (all killed)

| Mutant | Narrowing applied | Killed by |
| --- | --- | --- |
| M1 | Pi `--model` lock compares base string only, ignoring `provider/` and `:thinking` | `TestCanonicalQwenCompositeModelAndCoordinatesAreIdentityLocked` |
| M2 | `resolved.profile_provider`/`resolved.endpoint` sourced from the target assertion instead of the selected profile | `TestRunComposeCanonicalQwenUsesProfileDerivedEffectiveCoordinates` (unit-level Qwen test does **not** kill it — only the main test that splits profile and target across ancestor configs does) |
| M3 | Codex `-c profile=` no longer a conflict | `TestCanonicalCodexSelectorsAcceptExactAndRefuseEveryDivergentForm/profile_config` |
| M4 | entrypoint/vendor agreement check disabled | `TestCanonicalEntrypointResolutionNeverInfersOrFallsBack/alias_vendor_mismatch` |
| M5 | canonical aliases not installed in global mode | `TestSetupGlobalLinksCodexConfig`, `TestSetupGlobalDoesNotRequireALauncherBackendBuild`, `TestVerifyInstalledRuntimeRefusesIncorrectGlobalPiInfraTarget` (Setup postcondition) |

Additional bypass probes that correctly **refuse**: Codex `-mother`, `-pwork`,
`-p=work`, `--profile=work`, `--config model="other"`, `-c=model="other"`,
`--config=model="other"`, `-c 'model = "other"'`, `-c "model='other'"`,
`--model-reasoning-effort=low`, `-c 'profile = "work"'`; Claude `--model=other`,
`--effort=low`, `--effort HIGH` (case variant, fail-closed); Pi `--` message
operands. Exact repeats accepted and normalized in every documented form
(`-mgpt-5.6-sol`, `--config model="gpt-5.6-sol"`, `-c=model_reasoning_effort="high"`).
A probe of the composed argv with conflicting `[agents.codex.primary_session]`
and `[agents.claude.primary_session]` tables confirms no legacy model/effort
leaks into the locked argv.

## Finding 1 — CONFIRMED (blocking): a literal `--` disables the Codex and Claude identity lock

**Where:** `tools/agents-infra/internal/infra/canonical_target.go`
`lockCodexTargetArguments` and `lockClaudeTargetArguments`:

```go
if arg == "--" {
    return append(out, args[i:]...), nil
}
```

**Why it is wrong:** the lock treats `--` as an operand boundary. It is not one
at the provider layer. `parseCodexWrapperArgs` (`codex_launch.go:563`) and
`parseClaudeWrapperArgs` (`claude_launch.go:414`) consume that `--` as a
*wrapper* delimiter (`passThrough = true`) and **do not forward it**; every
following token is then parsed as a provider flag by
`normalizeCodexExplicitSelections` / the Claude composer and reaches the
provider argv. The lock has already stopped inspecting, so no
`target_identity_conflict` is ever raised.

**Reproduced through the production entrypoint** — `runTarget`, which is exactly
what the installed `openai-infra` / `anthropic-infra` wrappers invoke
(`exec "$TARGET" target <entrypoint> "$@"`) — with a recording fake provider on
`PATH`. Actual launched provider argv, `err = nil` in every case:

| Invocation | Launched provider argv | Contract violated |
| --- | --- | --- |
| `openai-infra -- exec -- --profile work` | `--model gpt-5.6-sol -c model_reasoning_effort="high" exec --profile work` | S3.6: Codex `--profile` must **always** fail `target_identity_conflict` on an alias invocation |
| `openai-infra -- -- --profile work` | `--model gpt-5.6-sol -c model_reasoning_effort="high" --profile work` | same |
| `openai-infra -- -- --model-reasoning-effort low` | `--model gpt-5.6-sol -c model_reasoning_effort="high" --model-reasoning-effort low` | S3.6: `--model-reasoning-effort` is a named identity selector; divergence must fail |
| `anthropic-infra -- -- --model other` | `--model claude-opus-5 --effort high --model other` | S3.6 model lock; Claude's parser is last-wins (documented at `claude_launch.go:368-375` for `--effort`), so the launched model is `other` |
| `anthropic-infra -- -- --effort low` | `--model claude-opus-5 --effort high --effort low` | S3.6 reasoning lock, same last-wins |

Section 5 is falsified at the same time: `--print-config` on the first row
prints `effective_profile: <not-configured>` / `effective_profile_source: native`
while the process is launched under Codex profile `work`; the Claude rows print
`effective_model: claude-opus-5` while `other` is what launches. The diagnostic
surface asserts an identity the launched process does not have.

**Not affected:** Pi. Its `--` is a genuine message-operand boundary and the Pi
composer refuses flag-like operands (`unsafe Pi message operand "--model"`), so
`qwen-infra -- -- --thinking high` fails closed.

**Why the suite misses it:** no test in `canonical_target_test.go`,
`canonical_target_pi_test.go`, `canonical_target_main_test.go`, or
`canonical_target_pi_main_test.go` passes a literal `--` through any lock. The
`--` branch of all three lock functions is entirely uncovered. This is the
"bypass path around the check" negative shape from
`references/negative-evidence.md`: the gate exists, is called from production,
and has a documented route around it.

**Note on reachability:** a single `--` typed straight after the alias name is
consumed by `runTarget`'s `flag.Parse`, so `openai-infra -- --profile work`
still refuses correctly. The bypass needs a `--` that survives into `fs.Args()`
— a second `--`, or any `--` after a positional. The first row above
(`-- exec -- ...`) is an ordinary provider-subcommand-plus-delimited-prompt
shape, not a contrived one.

**Required rework:**

1. Make the Codex and Claude locks keep inspecting across a wrapper `--`, or
   otherwise reconcile the lock's boundary with the wrapper parsers' — whatever
   the wrapper parser will still parse as a flag must still be identity-locked.
   Section 3.7's "existing operand-boundary behavior" is about tokens the
   provider treats as operands; these are not.
2. Add negative tests that drive `--` through the production alias path for
   each affected selector — Codex `--profile`, `-c profile=`, `--model`,
   `--model-reasoning-effort`, `-c model=`, `-c model_reasoning_effort=`, and
   Claude `--model` / `--effort` — asserting `target_identity_conflict` and no
   provider execution, plus a narrowing mutant that proves the new bound.
3. Keep the Pi behavior as is and add a test pinning it, so the divergence
   between Pi's real operand boundary and the hosted wrappers' delimiter is
   recorded rather than rediscovered.

## Finding 2 — low: hosted alias launches dump the full launch plan to stderr

`tools/agents-infra/main.go` `runTarget` writes
`RenderCanonicalTargetLaunchPlan(plan)` to stderr on **every** Codex/Claude
launch, not only under `--print-config`, and the Pi branch returns before that
line so it never does. Every `openai-infra` / `anthropic-infra` session opens
with ~20 lines of config noise on stderr, and the three aliases behave
inconsistently. Section 5 ties that render to `--print-config`. Not blocking on
its own; fold it into the rework.

## Definition-of-Done status

Met: canonical parsing and atomic root-to-cwd composition; tuple and Qwen
profile-assertion validation with the Section 5 qualified-model /
profile-provider / endpoint invariants (including the endpoint-equals-runtime
copy); alias install / repair / verify for all three sibling-only wrappers on
global and local paths with per-alias narrowing; no-config-rewrite and
no-side-effect guarantees across parse, compose, setup, verify and dispatch;
unchanged direct Codex, Claude, Pi and pi-infra precedence including Claude's
unknown-effort provider-native fallback; README and SKILL.md operator docs;
lint, vet, build and the full suite green with no skipped acceptance tests.

Not met: "Drive every Section 7 refusal through production parse/compose/alias/
setup/verify paths with narrowing controls" and "Gating, refusing, validating,
authorizing, or attesting behavior covered by negative tests that fail when the
gate admits what it must reject" — Section 7.6 identity-conflict refusal has an
uncovered live bypass, per Finding 1.
