# BUG-260817-2bh9nk — reviewer verdict RUN-260829-1f24ee

## Verdict

**Changes requested.** Route to `to-dev`.

The original managed `RunPi` gate, operator documentation, and installed
global/local normal-path checks are present and pass, but a later production
entry bypasses the shared model-origin boundary.

## Blocking finding: shared runtime launcher admits denied origins

Negative shape: **bypass path around the check**, with caller-minted evidence.

- Production entry: `tools/agents-infra/main.go:628` passes `os.Environ()` to
  `infra.RunSharedRuntimeLauncher` for `runtime runtime-launch`.
- Protected transition: `tools/agents-infra/internal/infra/pi_shared_launcher_darwin.go:32`
  validates an unkeyed authorization frame and then passes `options.Environ`
  directly to `sharedRuntimeExecve` near line 94.
- Missing enforcement: this entry never calls
  `ValidatePiExecutionEnvironment`, although that validator rejects exact
  `HF_ENDPOINT` and `MODEL_ENDPOINT` names.
- The ordinary broker path scrubs these names before starting the launcher,
  but the independently callable launcher entry remains reachable. A caller
  can create descriptor 3 with `os.Pipe`, start the launcher, learn its PID,
  and compose the frame from the readable profile/runtime key and exec-plan
  digest; no secret or keyed attestation is required.

An isolated reviewer test copied the package under `.temp/` and drove both the
test harness entry and a freshly built production `main` binary. For each exact
name it supplied a valid authorization frame, observed execve on the launcher
PID, and read the canary back inside the runtime target. Both cases passed,
which is a failure of the required denial gate. No environment values are
included in this artifact or the CLI diagnostics.

## Evidence

Baseline: `5c9b4e4f7a88e1eb937b80851af522e4fa4b066f` on
`task-board/story/STORY-260817-1on8ex`, equal to current `main` at review start.
Tool readiness: `/opt/homebrew/bin/go`, `go1.25.5 darwin/arm64`; task-board CLI
resolved to `/Users/alexis/.local/bin/task-board`.

| Command / gate | Exit | Result |
| --- | ---: | --- |
| Isolated launcher probe through package TestMain entry, both names | 0 | Bypass reproduced; runtime target received each denied variable. |
| Fresh `go build` production binary plus the same child-exec probe, both names | 0 | Bypass reproduced through `main.go` routing. |
| Original validator, managed `RunPi` denial, and clean-control tests | 0 | Pass; demonstrates the older path is protected while the new path bypasses it. |
| Installed bootstrap-global/project-local normal launcher test and both docs gates | 0 | Pass; these gates do not exercise direct shared `runtime-launch`. |

Attached logs contain only test names/results and bounded diagnostics:

- `BUG-260817-2bh9nk_RUN-260829-1f24ee_production-binary-bypass.log`
- `BUG-260817-2bh9nk_RUN-260829-1f24ee_original-gates.log`
- `BUG-260817-2bh9nk_RUN-260829-1f24ee_installed-docs-gates.log`

## Required rework and proof

1. Enforce the exact model-origin environment policy inside
   `RunSharedRuntimeLauncher` before `sharedRuntimeExecve`, including the nil
   environment fallback, with name-only diagnostics.
2. Add production-entry negatives for both `HF_ENDPOINT` and `MODEL_ENDPOINT`
   using a valid authorization frame; require no target exec/marker and no
   value leakage. Keep a clean control that reaches execve.
3. Run independent HF-only and MODEL-only narrowing mutants with `-count=1`;
   the opposite-name production test must redden in each case.
4. Keep the existing normal broker/global/local wrapper controls green and add
   composed installed-binary coverage for the shared launcher boundary where
   practical, so a future internal-entry bypass cannot hide behind broker
   scrubbing.

No repository source was modified by this review. Scratch probes live only
under `.temp/BUG-260817-2bh9nk/reviewer-RUN-260829-1f24ee/`. The pre-existing
dirty `LOGBOOK.md` change was preserved untouched.
