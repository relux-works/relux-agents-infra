# TASK-260826-3i0lwe — Reviewer Verdict, Change Request revision 1

- Verdict: **changes requested** (route `to-dev`)
- Reviewer run: `RUN-260826-2449c2`
- Change Request: `CR-TASK-260826-3i0lwe-1` revision `1`, `repository_delta=present`
- Base OID: `e70f953969d46e451892d9f16e7401b879910b6b`
- Candidate tree OID: `dcdfe67ce2e387f798e2fc66037a2bd22e5fd963`

## 0. Candidate-tree identity (verified, not assumed)

The story worktree was byte-identical to the reviewed candidate, so every command
below ran against the exact revision under review, and the tree was unchanged by
the review:

```
GIT_INDEX_FILE=$(mktemp); git read-tree HEAD; git add -A .; git write-tree
  -> dcdfe67ce2e387f798e2fc66037a2bd22e5fd963   (before review)
  -> dcdfe67ce2e387f798e2fc66037a2bd22e5fd963   (after review)
```

The CR base `e70f953` predates the already-committed `b3113e4`
(`STORY-260825-1r7z9o` shared-lease work), so this task's own delta is
`b3113e4..dcdfe67`: 16 files, +1455/-42.

## 1. Blocking finding

### F1 — The standalone stdin-isolation guard has no witness that can fail

`tools/agents-infra/internal/infra/pi_launch_posix.go:275-277` and
`tools/agents-infra/internal/infra/pi_shared_client_darwin.go:629-631` both add:

```go
if opts.Standalone == nil {
    piCmd.Stdin = opts.Stdin
}
```

This is the code that implements the AC clause *"stdin and UI approval are not
required"* and the DoD clause *"require no human prompt or stdin"*. It is a
gate. Nothing in the suite fails when it is removed.

**Reproducer** (both production launch paths mutated back to unconditional stdin
inheritance, i.e. the guard deleted):

```bash
perl -0pi -e 's/\tif opts\.Standalone == nil \{\n\t\tpiCmd\.Stdin = opts\.Stdin\n\t\}/\tpiCmd.Stdin = opts.Stdin/' \
  internal/infra/pi_launch_posix.go internal/infra/pi_shared_client_darwin.go
go test ./... -count=1
```

Result: the set of failing tests under the mutant is **identical** to the
unmutated baseline set — `comm -13 baseline mutant` is empty. Zero tests fail.

**Why the existing assertion cannot catch it.** `pi_standalone_shared_test.go:145-147`
asserts `info.StdinEOF` on both spawned workers, which reads like coverage. But
`TestRunPiStandaloneConcurrentWorkersShareOnlyRuntimeAndCrashCleanupPreservesPeer`
(`pi_standalone_shared_test.go:119-129`) never sets `RunPiOptions.Stdin`. With
`opts.Stdin == nil`, `os/exec` wires the child to `/dev/null` and the helper
(`pi_shared_integration_test.go:45-51`) observes EOF whether or not the guard
exists. The assertion is true for a reason unrelated to the thing it appears to
prove — the "green suite around a vacuous witness" shape.

**Why it matters even though there is no live bypass today.** I confirmed the
only production caller, `runPiStandaloneCLI` (`main.go:539-548`), omits `Stdin`,
so the guard is currently redundant and no reachable path inherits the operator
terminal. It becomes load-bearing for exactly the consumer this task defers: a
task-board adapter calling `RunPi` in-process would naturally pass `os.Stdin`,
and nothing would flag the regression.

**Fix.** In the concurrent-workers test (or a dedicated one), pass a readable
`Stdin` — e.g. an `io.Pipe` / `os.Pipe` read end holding bytes, or
`strings.NewReader("should-not-be-readable")` — and keep asserting
`info.StdinEOF`. Under the guard the child still sees EOF; with the guard
removed the helper's `os.Stdin.Read` returns `count == 1, err == nil` and the
test fails. Cover both launch paths: the exclusive path
(`pi_launch_posix.go:275`) and the shared-runtime path
(`pi_shared_client_darwin.go:629`) — the current shared-mode test only exercises
the latter, so the former needs its own non-shared-profile case.

## 2. Non-blocking observations (no rework required, listed for the record)

- **N1 — untested refusal branch, redundant:** `pi_standalone_entrypoint_invalid`
  (`pi_standalone.go:151-153`) survives removal. It is belt-and-braces: any
  non-`qwen-infra` canonical entrypoint is already stopped by the vendor mapping
  and by `resolved.Target.Profile == nil`, because `profileForbidden()` in
  `project_config.go:387-392` forbids `profile` on hosted targets.
- **N2 — untested refusal branch, redundant:** the `resolved.Target.Environment != "pi"`
  clause (`pi_standalone.go:213`) survives removal for the same reason as N1;
  `Profile == nil` already covers every reachable non-Pi target.
- **N3 — untested validation:** `pi_standalone_deadline_invalid` (`main.go:524-526`)
  survives removal. Not a security gate; a CLI bound.
- **N4 — doc nit:** the README standalone snippet shows only
  `[agents.pi.standalone_session]`, but `BuildPiStandaloneLaunchPlan`
  (`pi_standalone.go:254-256`) also requires
  `agents.pi.primary_session.pi_compatibility`. Following the snippet alone
  yields `invalid_project_configuration`.

## 3. What I verified and accept (do not redo this work)

### 3.1 Suite, build, lint — on the exact candidate tree

| Check | Result |
| --- | --- |
| `go test ./... -count=1` | ok (main 81.7s, attachments 2.6s, infra 126.5s) |
| `go vet ./...` (darwin/arm64, GOOS=windows, GOOS=linux) | clean |
| `go build ./...` (darwin/arm64, darwin/amd64, windows/amd64, linux/amd64) | clean |
| `gofmt -l tools/agents-infra` | clean |

### 3.2 Gates I attacked and that killed their mutants

Every mutant below was applied to the real production source and run through
`go test ./internal/infra/ ./ -run 'Standalone|PinnedPiNoModel'`.

| # | Mutant (narrowing unless noted) | Killed by |
| --- | --- | --- |
| M1 | drop `--no-extensions` from the managed set | 4 tests incl. `TestRunTargetQwenStandalonePrintConfigOwnsAuthorizationAndPreservesReasoning` |
| M2 | narrow caller-arg ownership from "all caller args" to "`--tools`/`-t` only" | 15 of 17 subtests of `TestRunPiStandaloneRefusesCallerAuthorizationAndRPCFlagsBeforeExecutableLookup` (`-e`, `--extension`, `--approve`, `-na`, `--mode rpc`, `-nbt`, …) |
| M3 | allow duplicate allowlist entries | `…RejectsNarrowedAndInvalidAllowlists/duplicate_allowlist` |
| M4 | `yolo_mode` presence alone satisfies authorization (false stops masking) | 3 tests incl. `TestStandalonePiNearestFalseMasksInheritedAuthorization` |
| M5 | `--mode json` → `--mode rpc` | 4 tests incl. `…DirectRPCBashBypassesToolCallHookWhileStandaloneExcludesRPC` |
| M7 | standalone shared-client identity falls back to `TASK_BOARD_RUN_ID` | `…ConcurrentWorkersShareOnlyRuntimeAndCrashCleanupPreservesPeer` |
| M8 | allowlist accepts unknown / `*` / future-builtin names | 3 tests incl. `…AuthorizationFailurePrecedesExecutableAndState/unknown` |
| M9 | prompt may look like a flag or `@file` operand | `TestStandalonePiPromptCannotBecomeAFlagOrFileOperand` |
| M10 | drop `<prompt>` redaction from the diagnostic argv | 3 tests incl. the `runTarget` print-config test |
| M13 | drop `--print` | 3 tests |
| M14 | drop `--no-session` | 3 tests |
| M16 | drop the CLI positional-argument refusal (correct call site, `main.go:521`) | `TestStandaloneCLIRefusesCallerOverridesWithSanitizedTypedFailure/direct_pi_positional_override` |

### 3.3 Real pinned-Pi 0.84.2 proofs — reproduced, with live controls

- `pi --version` on the pinned asset returns `0.84.2`.
- `TestPinnedPiNoModelManagedFlagsDisableProjectAndGlobalReplacementExtensions`
  passes and is **not** vacuous: its control leg (`--approve` without
  `--no-extensions`) writes the project-replacement marker, proving discovery is
  genuinely live in the un-gated configuration and genuinely suppressed under
  the managed set. Both project `.pi/extensions` and global agent-dir extensions
  are covered.
- `TestPinnedPiNoModelDirectRPCBashBypassesToolCallHookWhileStandaloneExcludesRPC`
  passes and writes a **real on-disk sentinel** through direct RPC `bash` while a
  `tool_call` handler blocks everything — confirming the research claim that the
  RPC `bash` command never traverses the model hook, and that production is right
  to expose JSON mode only.
- I independently confirmed against the pinned binary that Pi does **not**
  validate `--tools` names (`--tools nosuchtool` parses fine and fails later at
  model resolution), so the wrapper's own exact-name allowlist is the only name
  check — and it is the right design. `read,bash,edit,write,grep,find,ls` match
  the pinned help's own documented tool vocabulary.
- I also confirmed pinned Pi rejects a literal `--` operand
  (`Error: Unknown option: --`); `BuildManagedPiArguments` correctly strips the
  wrapper delimiter, so the composed argv is accepted.

### 3.4 Authorization / argument ownership

The public call sites are `runTarget → runPiStandaloneCLI` (`main.go:652-654`) and
`runPi → runPiStandaloneCLI` (`main.go:476-478`), both reaching
`BuildPiStandaloneLaunchPlan` or `RunPi{Standalone}`. Ownership is enforced by
refusing *all* caller provider arguments (`pi_standalone.go:142-144`) rather than
by last-wins composition, which is stronger than the researched contract. The
`flag.ContinueOnError` parser rejects every unknown Pi flag before `RunPi`, and
`fs.Args() != 0` closes the post-`--` positional escape. Authorization failure
provably precedes `LookPath`, runtime start, state, and locks
(`…AuthorizationFailurePrecedesExecutableAndState` asserts `!lookedUp` and an
empty cache root).

`--no-approve` is correctly project-trust separation, not the authorization
mechanism: the diagnostic reports `project_trust: "declined"` and
`native_enforcement: "pi_cli_strict_allowlist"` as independent fields, and
`--approve` is never emitted (asserted in
`TestBuildStandalonePiArgumentsOwnsExactAuthorizationAndMediumReasoning`).

### 3.5 Concurrency, runtime sharing, isolation

`TestRunPiStandaloneConcurrentWorkersShareOnlyRuntimeAndCrashCleanupPreservesPeer`
drives two live `RunPi` calls and verifies distinct PIDs, distinct process-group
IDs each equal to its own PID, distinct `PI_CODING_AGENT_DIR` /
`PI_CODING_AGENT_SESSION_DIR`, exactly two leases on one broker, the runtime PID
matching the runtime's own published pid-file, `SIGKILL` of worker A releasing
only A's lease while the peer runtime stays live (`inspectSharedProcessKernel`),
and the final release reaping the runtime. It also injects
`TASK_BOARD_RUN_ID=RUN-forged-board-id` and proves the workers do not collapse
onto it (M7 confirms this is load-bearing).

### 3.6 Reasoning, environment, identity, privilege, board independence

- `--thinking medium` is present in the **actually spawned** argv (helper capture,
  `pi_standalone_shared_test.go:149`) and in the launch plan's `Reasoning` field
  resolved to the config path. Interactive behavior is untouched: the old logic
  is moved verbatim into the `else` branch and the pre-existing Pi suite is green.
- The existing environment guards apply to standalone unchanged
  (`ValidatePiExecutionEnvironment` plus the inbound `PI_CODING_AGENT_*` refusal,
  `pi_launch_posix.go:129-137`), and Pi identity is re-verified immediately
  before spawn (`pi_launch_posix.go:257-269`).
- No `sudo`, `Setuid`, `launchctl`, or `SMAppService` anywhere in the Go sources;
  the diagnostic states `privilege_boundary: "calling_user"`.
- `go.mod` has three deps (`atomic`, `go-toml/v2`, `x/sys`) — no task-board
  dependency. No task-board import, run identity, or capability claim; the
  diagnostic says `task_board_adapter: "deferred_not_implemented"` and README/
  SKILL.md/LOGBOOK all state the deferral explicitly.
- Windows fails closed: `opts.Standalone != nil` sets `managed = true` and returns
  `pi_compatibility_unsupported` before any exec, never falling through to the
  unmanaged passthrough (`pi_platform_windows.go:102-131`).
- Failures are typed and sanitized: `PiStandaloneFailure.Error()` emits only a
  machine code, the prompt is replaced by `<prompt>` in `DiagnosticArgv`, and the
  `flag` parser names only the offending flag, never its value.
- Diff hygiene: no TODO/FIXME/debug prints/`t.Skip` introduced.

## 4. Routing

One blocking finding (F1). Everything else in a large, security-sensitive change
is proven at the real entry points with mutants that die and controls that show
the negatives are not vacuous. Please add the stdin witness described in F1 for
both launch paths, then resubmit; N1–N4 are informational and need no rework.
