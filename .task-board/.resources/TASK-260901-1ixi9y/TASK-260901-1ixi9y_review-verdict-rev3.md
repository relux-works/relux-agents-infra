# Review verdict: changes_requested (revision 3)

## Scope reviewed

CR-TASK-260901-1ixi9y-3, base `08a094c` -> candidate tree `59bf266...`, 12
changed paths. AC1 (install) and AC4 (drift/repair) remain solid, unchanged
from revision 2. The revision-2 defect (caller-leading flags rejected by
`fs.Parse` before `BuildCanonicalTargetLaunchPlan`) is genuinely fixed for the
alias chain itself: `openai-dange`/`anthropic-dange` now forward
caller-leading `-d`, `--danger`, `--yolo`, and ordinary flags like `--model`
and reach the provider-native dedup.

## Confirmed defect (production call site, reproduced against the candidate)

The fix widened the shared `runTarget` dispatcher, not just the new alias
chain. `runTarget` cannot actually tell an alias-injected leading `-d` apart
from a caller-typed one — both routes produce byte-identical argv by the time
`runTarget` sees it, because the `openai-dange`/`anthropic-dange` wrapper
scripts are plain `exec "$TARGET" -d "$@"` and the canonical `openai-infra`/
`anthropic-infra` wrapper is plain `exec "$TARGET" target <entrypoint> "$@"`.
So `runTarget`'s new branch (`tools/agents-infra/main.go:864`,
`if len(args) > 1 && args[1] == "-d"`) fires identically whether the `-d` came
from the dange wrapper or from a human typing `openai-infra -d ...` /
`anthropic-infra -d ...` directly — bypassing the new alias entirely.

Reproduced by calling `runTarget` (the exact function `run()` dispatches to
for `agents-infra target <entrypoint> ...`, which is exactly what the
pre-existing `openai-infra`/`anthropic-infra` wrapper scripts `exec` into)
with the argv a direct `openai-infra -d --model gpt-5.6-sol --print-config`
invocation produces — no `*-dange` involved:

```go
runTarget([]string{"openai-infra", "-d", "--model", "gpt-5.6-sol", "--print-config"})
```

Result: succeeds and prints a plan whose `provider_argv` includes
`--dangerously-bypass-approvals-and-sandbox` — i.e. Codex YOLO/no-sandbox is
silently selected — plus `--model gpt-5.6-sol` forwarded around any flag
validation. Before this CR (and still true today for any other unregistered
flag, e.g. `openai-infra --danger` — covered by
`TestRunTargetCanonicalAliasStillRequiresBoundaryForUnknownWrapperFlag`),
`openai-infra -d ...` failed closed with `flag provided but not defined: -d`.
That fail-closed contract is now silently gone for exactly the token `-d` as
the caller's first argument to the pre-existing canonical entrypoints.

The README documents this as intentional, which confirms it's not a fluke:

> "the canonical provider launcher accepts caller-leading `-d`, `--danger`,
> `--yolo`, and ordinary provider flags such as `--model` without an explicit
> `--`" (README.md:172-174)

That sentence is about the *canonical provider launcher* (`openai-infra`/
`anthropic-infra`), not about `openai-dange`/`anthropic-dange`.

## Why this is in scope, not a pre-existing/unrelated limitation

The task's own scope line says: "Preserve `openai-infra` and `anthropic-infra`
behavior; the new names are convenience aliases only," and the DoD contains
the exact bullet this breaks:

> "Without the direct-alias-owned leading -d, openai-infra and anthropic-infra
> preserve their pre-existing wrapper delimiter and unknown-flag refusal
> behavior; implicit provider-flag forwarding is scoped only to the dange
> chain"

There is no such thing as "the direct-alias-owned leading -d" once the
process is running — `runTarget` has no marker, env var, or any other signal
that distinguishes "this `-d` was injected by `*-dange`" from "this `-d` is
the caller's own first byte." The implementation comment
(`tools/agents-infra/main.go:912`, "Direct provider YOLO aliases own this
leading -d") asserts an invariant the code cannot enforce. The revision-2
verdict already flagged that an operator might type `-d`/`--danger`/`--yolo`
"out of habit" — that habit now silently escalates a direct `openai-infra`/
`anthropic-infra` invocation to bypass-sandbox/bypass-permissions mode with no
`--` boundary, where it used to fail loudly.

## Why the new tests didn't catch it

`TestRunTargetCanonicalAliasStillRequiresBoundaryForUnknownWrapperFlag` (the
revision-3 "narrowing negative") only drives `runTarget([]string{"openai-infra",
"--danger"})` — no leading `-d`. Every other new/updated test that exercises
`runTarget` with a leading `-d` is testing the intended dange-alias shape and
never asserts that the *same argv, arrived at without going through
`*-dange`*, should be rejected. Nothing in the new test matrix drives
`openai-infra`/`anthropic-infra` (not `*-dange`) with a leading `-d` and
asserts the old fail-closed behavior survives — which is exactly the gap: the
"narrowing negative" narrowed the wrong axis (unknown flag) instead of the
one the DoD explicitly named (leading `-d` without the alias).

## Requested fix

`runTarget` needs an actual signal that distinguishes the `*-dange` call site
from a direct `openai-infra`/`anthropic-infra` invocation before it can grant
the permissive parsing path — for example, having the `*-dange` wrapper pass
a distinct wrapper-owned sentinel (e.g. a dedicated `target` subcommand
argument, an internal env var scoped to the exec chain, or a distinct
entrypoint name plumbed through `BuildCanonicalTargetLaunchPlan`) rather than
inferring intent from the caller-visible token `-d`, which a direct caller can
type themselves. Whatever the mechanism, add a regression test that drives
`openai-infra`/`anthropic-infra` directly (not through `*-dange`) with a
leading `-d` and asserts the pre-existing fail-closed
`flag provided but not defined: -d` behavior is preserved, alongside the
existing dange-chain positive coverage. Then re-verify the README claim
("the canonical provider launcher accepts caller-leading `-d` ... without an
explicit `--`") is scoped correctly, since as written it currently describes
`openai-infra`/`anthropic-infra` themselves, not just the two new aliases.

## Verification note

The probe above was added temporarily to
`tools/agents-infra/canonical_target_main_test.go`, run, and then reverted;
the working tree was restored byte-for-byte from
`TASK-260901-1ixi9y_change-request_rev3.patch` (verified with `diff` against
the CR patch resource, which reported no differences) before this verdict was
recorded. No repository file was left modified by this review.
