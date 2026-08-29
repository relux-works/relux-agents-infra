# Reviewer Verdict — CR-TASK-260826-3i0lwe-2 revision 2

- Element: `TASK-260826-3i0lwe` (implement-standalone-pi-yolo-spawn)
- Change Request: `CR-TASK-260826-3i0lwe-2` revision `2`, state `ready`
- Base OID: `e70f953969d46e451892d9f16e7401b879910b6b`
- Candidate tree OID: `47bc47a3beaf019e46e18c0ae1ab581b7d8e951e`
- Repository delta: `present`
- Reviewer run: `RUN-260826-292d0a` (claude, darwin/arm64)
- Review date: 2026-08-26

**Verdict: accepted.**

## 0. Candidate identity actually reviewed

The Change Request base OID predates the merged shared-runtime commit
`b3113e4`, so `git diff <base> <candidate>` shows 35 paths. The delta belonging
to this task is `b3113e4..47bc47a` — 16 paths, +1578/-42. `.research/*`,
`pi_shared_broker_darwin.go`, `pi_shared_protocol.go`, `pi_state.go`,
`pi_plan.go` and the shared attestation/launcher/oracle suites are unchanged by
this revision; they are prior committed work carried in by the base choice.

Working tree was verified byte-identical to the candidate tree before and after
every experiment:

```
MATCH pi_standalone.go / pi_standalone_real_pi_test.go /
      pi_standalone_shared_test.go / pi_standalone_test.go /
      pi_standalone_main_test.go
git diff 47bc47a -- <all tracked changed paths>   # empty
```

All mutants below were applied to a copy-backed file and reverted; the closing
sha256 comparison against the candidate blobs is clean for every production file
touched.

## 1. Prior blocking defect (revision 1 F1) — fixed and independently proven

Revision 1 was rejected because both standalone launch tests left
`RunPiOptions.Stdin` nil, so deleting the stdin guards changed nothing:
`os/exec` already gives a nil `Stdin` the `/dev/null` EOF the test asserted.

Revision 2 supplies a readable non-empty witness on both production paths
(`strings.NewReader("exclusive-readable-stdin-witness")`,
`"shared-readable-stdin-witness-<name>"`) and the helper reports
`stdin_eof = count == 0 && errors.Is(err, io.EOF)`.

I reproduced the discrimination myself, one guard at a time:

| Mutant (production narrowing) | Exclusive stdin test | Shared concurrent test |
| --- | --- | --- |
| none (candidate) | PASS | PASS |
| `pi_launch_posix.go:276` guard removed → `piCmd.Stdin = opts.Stdin` | **FAIL** `StdinEOF:false` | PASS |
| `pi_shared_client_darwin.go:630` guard removed | PASS | **FAIL** `worker 0 inherited readable stdin … StdinEOF:false` |

Each guard is independently load-bearing — neither test masks the other — and
exact restoration returns both to green. Production call sites are named in the
test comments (`RunPi` exclusive branch; `runSharedPiSession`).

## 2. Gates attacked, not read

Every gate below was narrowed (not merely deleted) and the named test failed.

| Gate | Narrowing mutant applied | Test(s) killed |
| --- | --- | --- |
| Managed extension-discovery ownership | drop `--no-extensions` from the managed argv only | `TestBuildStandalonePiArgumentsOwnsExactAuthorizationAndMediumReasoning`, `TestRunPiStandaloneConcurrentWorkers…`, `TestPinnedPiNoModelDirectRPCBash…`, `TestRunTargetQwenStandalonePrintConfig…` |
| Caller-argument refusal | narrow `len(callerArgs) != 0` to a reserved-spelling denylist (the 16 tool/trust/extension spellings) | `…RefusesCallerAuthorizationAndRPCFlags…/--mode_rpc` |
| Tool allowlist membership | drop `!piStandaloneAllowedTools[tool]`, keep only the empty-string check | `…RejectsNarrowedAndInvalidAllowlists/{future_builtin_narrowing,wildcard}`, `…AuthorizationFailurePrecedesExecutableAndState/unknown` |
| Standalone client-state identity | remove the standalone run-ID override so shared state falls back to `TASK_BOARD_RUN_ID` | `TestRunPiStandaloneConcurrentWorkers…` |

The caller-argument mutant is the important one: the denylist form is the
obvious "reasonable" narrowing a future edit would make, and it is caught,
because the contract is *no caller provider arguments at all*, not *no reserved
spellings*.

## 3. Independent checks against the real pinned binary

The suite's fake `piExecCommand` never hands the composed argv to real Pi, so I
ran the exact production shape against pinned `0.84.2`
(`.temp/TASK-260817-2h8hn4/pi-standalone-darwin-arm64-0.84.2/pi/pi`), offline,
no model runtime, stdin `/dev/null`:

| Probe | Result |
| --- | --- |
| `--offline --provider … --model … --thinking medium --no-approve --no-extensions --tools read,bash,edit,write --mode json --no-session --print "say hi"` | parses; emits `{"type":"session","version":3,…}` then fails only on missing API key |
| control: same plus `--totally-unknown-flag` | `Error: Unknown option: --totally-unknown-flag` |

So the flag set is real, and the control proves pinned Pi would have rejected an
invented spelling. The advertised entry point was also exercised end to end
against a real config with the built binary:

```
agents-infra pi spawn --prompt "secret operator prompt" --print-config
→ exit 0, argv tail  … --no-approve --no-extensions --tools read,bash,edit,write,grep,find,ls
                       --mode json --no-session --print <prompt>
  reasoning=medium, prompt_leaked=False
yolo_mode=false → {"…","error":{"code":"pi_tool_authorization_required"}}  (stderr only)
```

`piStandaloneAllowedTools` is exactly pinned Pi's built-in set — bundled
`docs/extensions.md:2053` names `read, bash, edit, write, grep, find, ls`, and
the later-upstream `powershell` is correctly refused.

I also closed the one leg of the extension test that had no in-test control.
`TestPinnedPiNoModel…ReplacementExtensions` controls the *project* extension
(rewrites it with a fresh marker under `--approve`, marker appears) but the
*global* `$PI_CODING_AGENT_DIR/extensions` leg asserts absence with no positive
control. Probing pinned Pi directly:

| Launch | Global replacement marker |
| --- | --- |
| `--approve --tools bash` | **loaded** |
| `--no-approve --tools bash` | **loaded** (trust does not gate global extensions) |
| `--no-approve --no-extensions --tools bash` | not loaded |

The global assertion is therefore genuinely discriminating today, not a vacuous
absence. See finding F2 for the test-hygiene follow-up.

## 4. Brief checklist

| Item | Verdict | Evidence |
| --- | --- | --- |
| Both paths use readable non-empty stdin witnesses | ✅ | §1 |
| Each production guard independently proven by removal | ✅ | §1 table |
| Restored production passes the focused tests | ✅ | `go test ./internal/infra . -run 'Standalone\|PinnedPiNoModel'` → ok |
| Deadline accepts (0, 30m], rejects otherwise, sanitized typed error | ✅ | `TestStandaloneCLIAcceptsExactDeadlineBounds` (`1ns`, `30m`), `…RefusesOutOfRangeDeadline…` (`-1ns`, `0`, `30m1ns`) asserts the exact JSON body and that neither prompt nor duration leaks; deadline is genuinely enforced — `opts.Context` drives `pi_deadline_exceeded` in both the exclusive (`pi_launch_posix.go:311,343`) and shared (`pi_shared_client_darwin.go:680,684`) waits |
| README example carries the primary Pi compatibility prerequisite | ✅ | `README.md` standalone block emits `[agents.pi.primary_session]` with `pi_compatibility = "github-release:earendil-works/pi@v0.84.2:darwin-arm64#sha256-…"` and `yolo_mode = false`; `BuildPiStandaloneLaunchPlan` refuses `invalid_project_configuration` when it is absent |
| Exact owned Pi arguments retained | ✅ | argv `DeepEqual` in the unit test, argv tail check in the main test, argv membership check in both launch tests |
| No approval prompts / no stdin requirement | ✅ | §1 + `--print --mode json --no-session`; pinned Pi has no approval decision at all (pinned research §3) |
| No implicit extensions | ✅ | §2, §3 |
| No caller-controlled provider arguments | ✅ | 17-case refusal table before executable lookup; CLI-level `flag.ContinueOnError` + positional check → `pi_standalone_cli_invalid` |
| No sudo/root path | ✅ | `grep -niE "sudo\|setuid\|root"` over `pi_standalone.go`, `pi_launch_posix.go`, `pi_shared_client_darwin.go`, `main.go` → no hits; plan reports `privilege_boundary: calling_user` |
| Isolated Pi process, session, state | ✅ | distinct PIDs, distinct `PI_CODING_AGENT_DIR`/`SESSION_DIR`, `Setpgid` with `pgid == pid` and `pgid_a != pgid_b`, random per-run state key |
| Shared runtime reuse, no premature teardown | ✅ | two leases visible, `status.Runtime.PID` equals the runtime's own published pid file; `SIGKILL` on worker A releases only A's lease while the peer runtime stays live (`inspectSharedProcessKernel`); last release reaps it |
| No task-board adapter or unrelated scope | ✅ | no `task-board`/`TASK_BOARD` reference in new production code; the standalone path deliberately *overrides* `TASK_BOARD_RUN_ID` (mutant-proven), and `task_board_adapter: deferred_not_implemented` is stated in plan, README, SKILL and LOGBOOK |

## 5. Validation run by this reviewer

| Command | Result |
| --- | --- |
| `go build ./...` | ok |
| `go vet ./...` | clean |
| `gofmt -l .` | empty |
| `GOOS=windows GOARCH=amd64 go build ./...` / `go vet ./...` | ok / clean |
| `GOOS=linux GOARCH=amd64 go build ./...` | ok |
| `go test ./internal/infra -count=1` | **ok** 121.090s |
| `go test . -count=1` | **ok** 75.656s |
| `go test ./internal/attachments -count=1` | ok |
| focused `-run 'Standalone\|PinnedPiNoModel'` after restoring every mutant | ok (infra 5.9s, main 2.2s) |

Windows standalone is fail-safe rather than untested-by-omission: `opts.Standalone != nil`
forces `managed = true`, which returns `pi_compatibility_unsupported` before any
`LookPath`/`exec`, so the standalone request can never reach the unmanaged
pass-through launch.

## 6. Findings (non-blocking, recorded for follow-up)

**F1 — standalone client run-state directories are never reclaimed.**
`RunPi` creates `…/runs/<random>/{agent,sessions,models.json,lock}` per spawn
and nothing removes it (`grep os.RemoveAll` over `internal/infra` non-test code
shows no state-root cleanup). Every `qwen-infra spawn` therefore leaves a
directory behind forever. The plan's `state.persistence: "disabled"` is a
hardcoded label; it is defensible as "never reused across runs" (the key is
fresh random and Pi runs `--no-session`), which is how README words it, but it
reads as a stronger claim than the disk behavior supports. Not an authorization
defect and the accumulation pattern predates this change (the shared-runtime
task already keyed client state by run id), so it does not block. Recommend a
follow-up that reaps the run directory on exit, or tightens the diagnostic to
say `session_persistence: "disabled"`.

**F2 — one leg of the pinned-Pi extension test has no in-test control.**
The project-extension leg proves discovery is real by rewriting the extension
and re-running under `--approve`; the global `$PI_CODING_AGENT_DIR/extensions`
leg has no such control, so it would silently go vacuous if a future Pi stopped
discovering global extensions. I verified externally (§3) that it is
discriminating against pinned 0.84.2 today. Recommend mirroring the project
control for the global directory.

**F3 — `tool_authorization` diagnostics are constants, not derivations.**
`piStandaloneToolAuthorization` hardcodes `extension_discovery: "disabled"`,
`project_trust: "declined"`, `rpc_direct_bash: "not_exposed"` rather than
deriving them from the composed argv. Today the argv is exactly bound by
`DeepEqual` and tail assertions, so the `--no-extensions` narrowing mutant is
caught — but by the *argv* tests, not by the diagnostic, which kept reporting
`"disabled"` throughout that mutant run. If the diagnostic is ever consumed as
an attestation rather than as documentation, derive it from
`plan.Argv`/`argsPlan` instead.

None of F1–F3 admits what the gate must reject, and none contradicts an
acceptance criterion. They are follow-ups, not rework.
