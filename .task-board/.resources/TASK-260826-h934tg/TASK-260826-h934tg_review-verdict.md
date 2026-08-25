# TASK-260826-h934tg — Reviewer verdict (CR-TASK-260826-h934tg-1 revision 1)

- Verdict: **changes requested** → `to-dev`
- Reviewer run: `RUN` for TASK-260826-h934tg, `reviewing` → `to-dev`
- Candidate: base `fd80bd8e0c1de3f372fd1a7527613a5135762de4`, tree `02e41e53790b42bfe5cb7cc5c9e19d622a507035`
- Everything below was rerun by this reviewer on the exact candidate tree; nothing
  is accepted from previously attached evidence.

## Candidate identity and ancestry — verified

| Check | Result |
| --- | --- |
| `git rev-parse main` | `fd80bd8e0c1de3f372fd1a7527613a5135762de4` (declared base == current main) |
| `git merge-base --is-ancestor fd80bd8 HEAD` | exit 0 — main **is** an ancestor of HEAD |
| HEAD | `3a52ec7`, a real merge commit, parents `8f81371` (accepted TASK-260826-3i0lwe) + `fd80bd8` (main) |
| `git rev-parse HEAD^{tree}` | `02e41e53790b42bfe5cb7cc5c9e19d622a507035` — matches declared candidate tree |
| Patch resource sha256 | recomputed `79cc18cb…ca7f` == declared == `shasum` of live `git diff base..candidate` |
| Changed paths vs main | exactly the declared 17; `.task-board/` byte-identical to main |

This is a genuine fresh-base merge, not a replay-equivalent tree.

## Accepted implementation preserved — verified

`git diff 8f81371 HEAD -- . ':(exclude).task-board'` touches 11 files. Of those,
`pi_args.go`, `pi_plan.go`, `pi_test.go`, `pi_operator_docs_test.go` are
byte-identical to `fd80bd8` (taken from main verbatim), and `pi_config.go`'s only
impl→candidate delta is main's own removal of `validatePiPrimarySessionYolo`.

The accepted standalone core is **byte-preserved**: `pi_standalone.go`,
`pi_standalone_test.go`, `pi_standalone_shared_test.go`,
`pi_standalone_real_pi_test.go`, `pi_standalone_main_test.go`,
`pi_shared_integration_test.go`, `main.go`, `project_config.go`,
`canonical_target.go` are unchanged from `8f81371`.

`git log -1 --cc` shows the whole hand-resolved surface is 6 files:
`LOGBOOK.md`, `README.md`, `SKILL.md`, `pi_launch_posix.go`,
`pi_platform_windows.go`, `pi_shared_client_darwin.go`.

## Documentation additivity — verified

`### ` entry counts: main 141, accepted impl 145, candidate **146**. Set
difference in both directions is empty — the candidate is the union; no mainline
or Story entry was dropped. New: `1248`, `1335`, `1411`, `2346`, `2352`.

`SKILL.md` loses no line vs main. `README.md` loses no line except two tool-table
rows that are **modified in place** to add the spawn command, not deleted.

## Scope containment — verified

The delta introduces no `sudo`, `setuid`, `seteuid`, `SysProcAttr.Credential`, or
uid manipulation; no task-board adapter, registration, or lifecycle dependency
(`TASK_BOARD_RUN_ID` appears only in the negative test that proves standalone
state never derives from a forged board id); the pinned tool allowlist is
unchanged (`read bash edit write grep find ls`); the owned argument block is
unchanged; `ValidatePiExecutionEnvironment` and the `HF_ENDPOINT`/`MODEL_ENDPOINT`
refusals are untouched.

## Landing suite — rerun on the candidate tree

| Command | Result |
| --- | --- |
| `go vet ./...` | exit 0 |
| `go test ./... -count=1` | exit 0 — root 80.3s, attachments 2.0s, infra 130.2s |
| `GOOS=darwin GOARCH=arm64 go build ./...` | exit 0 |
| `GOOS=darwin GOARCH=amd64 go build ./...` | exit 0 |
| `GOOS=linux GOARCH=amd64 go build ./...` | exit 0 |
| `GOOS=linux GOARCH=arm64 go build ./...` | exit 0 |
| `GOOS=windows GOARCH=amd64 go build ./...` | exit 0 |

Note on skips: `officialPiAsset` resolves through the repo/gitdir root, so the
standalone production-launch tests only run inside a real checkout. In the
candidate worktree they **run**, not skip:
`TestRunPiStandaloneConcurrentWorkersShareOnlyRuntimeAndCrashCleanupPreservesPeer`
PASS 1.64s, `TestRunPiStandaloneExclusiveWorkerClosesReadableStdin` PASS 1.04s.
All mutation work below was therefore done in a detached worktree of the exact
candidate commit (since removed; candidate tree OID unchanged), never in a flat
copy where those two would have silently skipped.

## Gates attacked, not read

Mutants applied one at a time to a detached worktree of `3a52ec7`.

| # | Mutant | Outcome |
| --- | --- | --- |
| M1 | exclusive standalone stdin guard deleted (reverted to main's unconditional `piCmd.Stdin = opts.Stdin`) | **killed** — `TestRunPiStandaloneExclusiveWorkerClosesReadableStdin` FAIL, `StdinEOF:false` |
| M2 | shared standalone stdin guard deleted | **killed** — `…ConcurrentWorkersShareOnlyRuntimeAndCrashCleanupPreservesPeer` FAIL, `worker 0 inherited readable stdin` |
| M3 | allowlist membership narrowed to "non-empty" (`!piStandaloneAllowedTools[tool]` dropped) | **killed** — `future_builtin_narrowing`, `wildcard` FAIL |
| M4 | standalone yolo gate narrowed to absent-only (explicit `false` admitted) | **killed** — `false_yolo`, `…NearestFalseMasksInheritedAuthorization`, `…AuthorizationFailurePrecedesExecutableAndState/false` FAIL |
| M5 | caller-argument refusal narrowed `!= 0` → `> 1` | **killed** — `…RefusesCallerAuthorizationAndRPCFlags…` FAIL on `--approve`, `-a`, `--no-approve`, `-na`, `-ne` |
| M6 | prompt redaction removed from `DiagnosticArgv` | **killed** — prompt leaked into the launch plan, argument/plan tests FAIL |
| M7 | deadline upper bound widened `30m` → `90m` | **killed** — `…RefusesOutOfRangeDeadlineWithSanitizedFailure/30m1ns` FAIL |
| M8 | primary-session yolo routed into the standalone branch (fails closed) | **SURVIVED** full suite |
| M8b | primary-session `--approve` forwarded **into** the launched standalone argv (fails open) | **SURVIVED** full suite — see F1 |

Positive production probe (built binary, not a unit test): on one composed policy
with `[agents.pi.primary_session] yolo_mode = true` **and**
`[agents.pi.standalone_session] yolo_mode = true`:

- `agents-infra target qwen-infra spawn --print-config` →
  `[… --thinking medium --no-approve --no-extensions --tools read,bash,edit,write,grep,find,ls --mode json --no-session --print <prompt>]`,
  `--approve` absent, `project_trust: declined`, prompt redacted.
- `agents-infra pi --print-config` on the **same** config →
  `[… --thinking medium --approve]`.

So the shipped reconciliation is behaviourally correct: interactive keeps main's
project trust, standalone declines it.

## F1 — the merge's own authorization decision has no negative witness

**Severity: blocking (missing negative coverage on a new, fail-open-capable
authorization branch).**

This leaf created a brand-new authorization branch that exists on neither parent:

```go
// tools/agents-infra/internal/infra/pi_launch_posix.go:71-100
effectiveArgs := opts.Args
if opts.Standalone != nil {
    …                                   // standalone: primary-session yolo never applied
} else {
    effectiveArgs, err = applyPiPrimarySessionYolo(opts.Args, composite.PiPrimarySession)
    …
}
```

On the accepted implementation the composition could not exist (primary
`yolo_mode = true` was a hard `pi_yolo_mode_unsupported` refusal). On main the
standalone mode did not exist. The merge is what decided that interactive project
trust must not reach the unattended worker — and LOGBOOK `1411` states exactly
that as its DECISION, with an EVIDENCE line pointing at `go test ./...`.

That suite cannot see the decision. **Every** standalone test config leaves
`[agents.pi.primary_session] yolo_mode` absent, so `applyPiPrimarySessionYolo` is
a no-op in all of them, and the existing `--approve`-must-not-appear assertions
(`pi_standalone_test.go:67`, `pi_standalone_shared_test.go:155`) cannot
distinguish "standalone excludes primary trust" from "primary trust happens to be
off in this fixture".

Proof, not inference — mutant **M8b**: apply `applyPiPrimarySessionYolo`
unconditionally and thread `effectiveArgs` into `BuildStandalonePiArguments`
(validating caller args against `nil`). The launched worker then receives

```
--provider local-provider --model Model --thinking medium
--approve --no-approve --no-extensions --tools read,bash,edit,write
--mode json --no-session --print "review probe worker"
```

i.e. interactive project trust reaches the unattended worker's argv. `go build`,
`go vet`, and the **entire** configured suite stay green on M8b:

```
ok  …/tools/agents-infra                     80.060s
ok  …/tools/agents-infra/internal/attachments 1.925s
ok  …/tools/agents-infra/internal/infra      130.999s
```

This is the shape "the check is present but its rejection class is never
exercised" — positive-path-only evidence for an authorizing decision.

It is a specific miss, not a declined category: the **sibling** merge branch of
identical shape, `if opts.Standalone == nil { piCmd.Stdin = opts.Stdin }`, *is*
witnessed in both production paths (M1 and M2 both die).

### Required rework

Add a negative witness driving the real entry point. The reviewer wrote and ran
one; it is attached as `review-probe-suggested-witness.go.txt` inside the evidence
archive. Its only substantive difference from the existing exclusive-launch test
is one line:

```go
body = strings.Replace(body, "[agents.pi.primary_session]\n",
    "[agents.pi.primary_session]\nyolo_mode = true\n", 1)
```

followed by asserting `--approve`/`-a` absent and `--no-approve` present in the
argv the child actually received. Measured:

- clean candidate tree → `--- PASS … (1.07s)`
- M8b → `--- FAIL … standalone worker inherited primary-session project trust: [… "--approve", "--no-approve", …]`

Fold it into `pi_standalone_shared_test.go` (production call site: `RunPi` in
`pi_launch_posix.go` → `BuildStandalonePiArguments`). Covering the shared path as
well would be better; the exclusive path alone already kills M8b. Also update the
LOGBOOK `1411` EVIDENCE line to name the witness rather than the general suite.

No product-code change is expected — the code is correct as shipped.

## O1 — non-blocking observation: standalone stdout/stderr now inherit a tty

The merge routes standalone through main's `piProcessWriter`, which returns an
`*os.File` unwrapped. The accepted implementation always wrapped standalone
stdout/stderr in `newPiSynchronizedWriter`, so the child got a pipe. A
terminal-launched worker now inherits the real terminal fd, and its stderr no
longer shares the mutex with the managed runtime's stderr
(`pi_launch_posix.go:205` still wraps the runtime writer).

This is a declared decision in LOGBOOK `1411`, has no authorization or isolation
consequence, and the future board adapter is unaffected because a piped stdout
still falls back to the synchronized writer. Recorded so the adapter task does not
rediscover it; not grounds for rework.

## Definition of Done

| Item | State |
| --- | --- |
| Checkpoint accepted implementation before merging | met — `8f81371` is parent 1 |
| Merge mainline, prove main is ancestor of HEAD | met |
| LOGBOOK/overlaps resolved additively, no feature change | met |
| Full Go test + vet landing suite on merged tree | met — rerun by reviewer, all green |
| Fresh-base merge evidence + published CR | met — patch sha256 verified |
| Negative coverage of gating/authorizing behavior with production call site named | **not met — F1** |
| Lint / build not broken | met — vet + 5 cross builds |
| Implementation matches AC | met on behaviour, blocked on the evidence AC |
| Solution fits project architecture | met |

## Routing

`to-dev` for the F1 witness. Behaviour is correct; the rework is test-only.
