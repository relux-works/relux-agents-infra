# Review verdict: TASK-260901-1ixi9y (CR-TASK-260901-1ixi9y-5, revision 5)

## Verdict: changes_requested

## What was checked
- Full diff (base 08a094c -> candidate dae99b3) across all 12 changed paths.
- `go build ./...` and `go vet ./...` clean in `tools/agents-infra`.
- Ran the full new/changed test surface focused on this feature:
  `TestRunTargetDirectProviderYoloMarkerAcceptsLeadingProviderFlagsWithoutDelimiter`,
  `TestParseDirectProviderYoloTargetArgsPreservesProviderBytesAndConsumesOnlyWrapperSyntax`,
  `TestRunTargetCanonicalAliasesStillRefuseLeadingDangerWithoutDangeMarker`,
  `TestInstalledLocalProviderAliasesScopeImplicitFlagForwardingToDangeChain` (installed-binary
  matrix, both openai-dange/anthropic-dange and the openai-infra/anthropic-infra negative case),
  `TestDirectProviderYoloAliasesDelegateOnceAndPreserveCWDArgv`,
  `TestDirectProviderYoloDelegationResolvesExactlyOneNativeDangerFlag`,
  `TestDirectProviderYoloWrapperBodyForWindowsUsesCanonicalSiblingOnce`,
  `TestSetupRepairsAndVerifyRejectsDirectProviderYoloAliasDrift` (missing/drifted/symlinked/non-executable + repair).
  All pass. Coverage of AC1-AC5 and the DoD's positive-path matrix is solid and drives real
  production call sites (installed binaries, `runTarget`, `Setup`/`VerifyInstalledRuntime`).
- Adversarial probe against the marker gate itself (temporary local test, removed after
  verification, working tree confirmed clean/matches the 12 CR paths before this verdict).

## Confirmed defect: the "dange call-site marker" is not an authentication boundary, it's a
literal argv string comparison, and it is forgeable by anyone who can invoke `openai-infra` /
`anthropic-infra` directly (or `agents-infra target <entrypoint> ...`) without ever going
through `openai-dange` / `anthropic-dange`.

`runTarget` (main.go) treats `args[1] == directProviderYoloCallSiteMarker` as proof the call
came from the dange wrapper chain:

```go
if len(args) > 1 && args[1] == directProviderYoloCallSiteMarker {
    // Only the direct-provider YOLO wrapper chain supplies this internal
    // marker. Consume it here, add the wrapper-owned danger selection, ...
    printConfig, providerArgs = parseDirectProviderYoloTargetArgs(args[2:])
}
```

That comment's premise is false: `DirectProviderYoloCallSiteMarker` is an exported Go constant
(`internal/infra/infra.go`), a fixed, discoverable, non-secret string
(`--agents-infra-direct-provider-yolo-call-site`). Nothing ties its presence to having actually
run through the `openai-dange`/`anthropic-dange` wrapper scripts — no env var set only by the
wrapper, no ancestry check, no per-install token. Anyone who reads this source file (this repo),
or runs `strings` on the installed binary, can reproduce it.

### Reproduction (verified locally, `go test` against `runTarget` directly)

```go
runTarget([]string{"openai-infra", "--agents-infra-direct-provider-yolo-call-site",
    "--print-config", "exec", "rm -rf everything"})
```

Result: `openai-infra` — invoked directly, `openai-dange` never in the picture — emits
`--dangerously-bypass-approvals-and-sandbox` in `provider_argv` and forwards `"exec"`,
`"rm -rf everything"` with **no `--` delimiter**. The exact same call with a caller-leading
`-d` instead of the marker correctly fails closed with `flag provided but not defined: -d`
(this is the behavior `TestRunTargetCanonicalAliasesStillRefuseLeadingDangerWithoutDangeMarker`
asserts) — but the marker string trivially reaches the identical implicit-YOLO,
no-delimiter-required code path that AC/DoD say must be exclusive to the dange chain.

Confirmed identically for `anthropic-infra` (same `runTarget` code path, same marker constant).

### Why this fails the task's own bar
- AC2/AC3 promise `openai-dange`/`anthropic-dange` as the exclusive entry to "exactly one
  danger/YOLO selection" launches; this shows the canonical `openai-infra`/`anthropic-infra`
  aliases — which the scope explicitly requires to "preserve ... behavior" — grant the identical
  privilege to anyone who knows one constant string, silently, with no distinguishing signal in
  the resulting launch.
- The DoD explicitly enumerates: "Direct openai-infra and anthropic-infra invocations with
  leading -d and no dange call-site marker retain the pre-existing flag provided but not defined
  refusal" — this is tested — but never tests the complementary, more dangerous case: a direct
  invocation *with* the call-site marker and no actual dange wrapper in the call chain. That gap
  in the negative-test matrix is exactly where the bypass lives.
- This is the same defect *class* the LOGBOOK entry says revision 3 already introduced and
  revision 4 was supposed to fix ("Revision 3 inferred dange origin from caller-visible leading
  `-d` ... Revision 4 replaces that inference with a dange-wrapper-owned internal call-site
  marker"). Swapping the inferred token for a differently-shaped but equally public/stable token
  does not close the underlying gap: the dispatcher still authenticates the call site by argv
  content alone, not by anything only the wrapper can produce/prove.

## Requested fix (not prescriptive, but the shape that would close this)
Give the wrapper-to-dispatcher handoff an actual capability the dispatcher can verify was
produced by the wrapper and not typed by a caller — e.g., a value only the generated wrapper
script can supply (a per-install/per-run generated secret, an fd/env-var channel a normal shell
invocation of `openai-infra` wouldn't set, or dropping the "trust argv" model entirely and
having the two dange wrappers invoke a distinct, separately dispatched target name that carries
its own resolution instead of tunneling through `openai-infra`'s target dispatch with a
smuggled flag). Add a negative test that forges the marker directly against `openai-infra` /
`anthropic-infra` (bypassing `openai-dange`/`anthropic-dange` entirely) and asserts it still
fails closed the same way a bare `-d` does.

## Scope note
Everything else in this revision — install/repair/drift-detection for both aliases (global and
local), byte/arg preservation, Windows wrapper body, dedup of redundant caller danger flags, and
docs — is correct and well-tested. This is the sole blocker.
