# Review verdict: changes_requested (revision 2)

## Scope reviewed
CR-TASK-260901-1ixi9y-2, base `08a094c` -> candidate tree `9e605e...4738`,
12 changed paths. AC1 (install) and AC4 (drift/repair) verified solid.
AC2/AC3 (byte-for-byte forwarding + exactly one danger selection) fail at the
real production call site whenever the caller's own arguments start with a
flag-shaped token.

## What's good
- `installDirectProviderYoloLaunchers` / `directProviderYoloLauncherFailures`:
  install, drift, repair, missing/symlinked/non-executable coverage in
  `runtime_receipt_test.go` and `infra_test.go` is solid and exercises the
  real `Setup`/`VerifyInstalledRuntime` call sites (AC1, AC4).
- Revision 1's flaw (`runTarget` rejecting the wrapper's own leading `-d`)
  is genuinely fixed for that exact repro.
- `TestInstalledLocalDirectProviderYoloAliasesReachProvidersWithOneNativeDangerFlag`
  is a real production-call-site test (installed binary -> generated alias ->
  `agents-infra target` -> fake provider exec).

## Confirmed defect (production call site, reproduced against the built candidate binary)

`runTarget` (`tools/agents-infra/main.go`) only strips a single leading `-d`
token before calling `fs.Parse`. Every other caller-supplied argument that
looks like a flag (including the exact `-d`/`--danger`/`--yolo` shortcuts
this feature is themed around, or an ordinary flag like `--model`) is fed
straight into a `flag.FlagSet` that registers only `--print-config`, so
`fs.Parse` fails closed with `flag provided but not defined: ...` before
`BuildCanonicalTargetLaunchPlan` — and the provider-native dedup it performs
— is ever reached.

Reproduced end-to-end through a real `setup local` install (`AGENTS_INFRA_SOURCE_DIR`
pointed at the candidate worktree, `go build` from source, real project
config, fake `codex`/`claude` binaries recording argv):

```
$ ./.local/bin/openai-dange -d --print-config
flag provided but not defined: -d
EXIT=1

$ ./.local/bin/openai-dange --danger --print-config
flag provided but not defined: -danger
EXIT=1

$ ./.local/bin/openai-dange --model foo --print-config
flag provided but not defined: -model
EXIT=1

$ ./.local/bin/anthropic-dange --yolo --print-config
flag provided but not defined: -yolo
EXIT=1
```

`--` still works (`openai-dange -- --danger inspect` succeeds), and a bare
`--print-config` or a leading non-flag positional works — but that is a
narrow slice of "byte-for-byte forwarding of caller arguments."

## Why this is in scope, not a pre-existing/unrelated limitation

`openai-infra --danger --print-config` (the pre-existing canonical entrypoint,
no dange alias involved) fails the same way and always has — operators are
expected to use `--` before provider-native flags there, and that's
documented/consistent. That part is not new.

What *is* new and specific to this change: the README this revision adds
promises exactly the opposite of what ships:

> "`openai-dange` and `anthropic-dange` delegate to their matching canonical
> vendor alias with exactly one added `-d` YOLO selection, then forward every
> caller argument unchanged; the canonical provider launcher deduplicates
> danger selections to exactly one provider-native flag."

"forward every caller argument unchanged" and "deduplicates danger
selections" are both false for the most foreseeable case: an operator typing
`-d` (or `--danger`/`--yolo`) themselves on `*-dange`, e.g. out of habit from
using `openai-infra -- --danger`, not realizing the alias already adds it.
The dedup logic in `BuildCanonicalTargetLaunchPlan` (proven by
`TestDirectProviderYoloDelegationResolvesExactlyOneNativeDangerFlag`) is
never reached — `runTarget` crashes first. AC2/AC3 explicitly promise
"exactly one danger/YOLO selection" *and* byte-for-byte forwarding; the
system, not the caller, is supposed to guarantee the "exactly one" part.

## Why the new tests didn't catch it

Every new test that exercises `runTarget`/the installed alias either passes
no caller args, passes only the registered `--print-config` as the first
token, or puts caller args after an explicit `--`:
- `TestDirectProviderYoloAliasesDelegateOnceAndPreserveCWDArgv`: first token
  is `--print-config` (registered), rest after `--`.
- `TestRunTargetAcceptsAliasPrefixedDangerAndPrintsExactlyOneNativeFlag`: same
  shape (`-d --print-config -- ...`).
- `TestInstalledLocalDirectProviderYoloAliasesReachProvidersWithOneNativeDangerFlag`:
  caller args start with `--`.
- `TestDirectProviderYoloDelegationResolvesExactlyOneNativeDangerFlag`: calls
  `BuildCanonicalTargetLaunchPlan` directly, bypassing `runTarget`'s
  `fs.Parse` entirely — the exact class of gap the first review cycle already
  flagged and asked to close for the `-d`-only case; it wasn't generalized to
  other caller-supplied flags including a redundant `-d`.

No test drives `openai-dange`/`anthropic-dange` (or `runTarget` with an
alias-style leading `-d`) with a caller-supplied flag as the next token. This
is exactly the positive-path-only shape: every recorded test result is green
because none of them try the argument shape a real operator relying on the
documented byte-for-byte/dedup guarantee would type.

## Requested fix

In `runTarget`, once the alias-owned leading `-d` is consumed, the remaining
caller args need to reach `BuildCanonicalTargetLaunchPlan` without being
filtered through a `flag.FlagSet` that only knows `--print-config` — e.g.
treat everything after the stripped `-d` as an implicit `--` boundary (still
allow `--print-config` specifically, since that's wrapper-owned and
documented), and add a regression test that drives `openai-dange`/
`anthropic-dange` (or `runTarget` with the alias-injected prefix) with a
caller-supplied `-d`, `--danger`/`--yolo`, and an ordinary flag like
`--model`, asserting exactly one native danger flag reaches the provider and
the caller's other arguments still arrive. Then re-verify README's "forward
every caller argument unchanged" / "deduplicates danger selections" claims
against the fixed binary the same way this verdict did (real setup install,
real project config, fake provider binaries recording argv).
