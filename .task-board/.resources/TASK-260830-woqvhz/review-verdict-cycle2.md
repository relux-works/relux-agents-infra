# TASK-260830-27u51n review verdict

## Verdict

Changes requested for `CR-TASK-260830-27u51n-4` revision 4. Route: `to-dev`.

## Blocking finding

F1 — `prove a bound by narrowing, not by deleting`: the Setup-path parity test
does not protect the acceptance criterion that the fallback applies **only**
for an externally caused hosted-CI execution failure the agent cannot repair.

Reproduction against a disposable copy of candidate tree
`aeef4b0d68e2c938360798c4fb48d382dd2e4ddd`:

1. The unmodified production-path control
   `go test ./internal/infra -run '^TestSetupGlobalPublishesExternalCILocalMirrorPolicyToClaudeAndCodex$' -count=1`
   exited 0.
2. Broaden only the first policy sentence from `Use a local mirror only when ...`
   to `Use a local mirror for any repairable or external CI disruption; this
   includes cases where ...`. This expressly admits repairable disruption and
   violates the task's exclusivity and unrepairability boundary while retaining
   every substring in `externalCILocalMirrorPolicyClauses`.
3. The same production Setup-path test still exited 0.

The producer's earlier expected-red policy mutant removed the entire asserted
`hosted CI cannot execute ...` phrase, so it proved phrase presence but not the
claimed bound. The surviving broadened-trigger mutant is a materially different
negative shape and is blocking under the task's Evidence That Counts contract.

## Required rework

- Make the focused Setup-path test assert the complete exclusive trigger,
  including `only when` and `that the agent cannot repair`, rather than an
  interior substring that broader policy can preserve.
- Add a named broadened-trigger negative proof: admit repairable or merely
  inconvenient CI disruption while retaining the hosted-external-cause phrase,
  and require the focused production Setup test to fail.
- Record this surviving-mutant regression in `LOGBOOK.md` during producer
  rework. The reviewer did not edit the immutable candidate tree.
- Rerun the focused gate and the full relevant Go suite uncached. Attach a
  genuinely green full-suite run; do not replace a red broad run with isolated
  passing subsets.

## Additional test evidence

- Candidate diff: four declared paths, 137 insertions, `git diff --check` exit 0.
- Current Claude chain repair is effective: removing the workflow include from
  `.instructions/INSTRUCTIONS.md` makes the focused test red in producer
  evidence; the unmodified focused test passed independently twice.
- Reviewer full run `go test ./... -count=1` exited 1 after 353.815s in
  `TestPiLaunchReadinessServiceUnavailableStillHonorsRuntimeBoundsAtProductionEntry/times_out_while_runtime_remains_alive`
  because `readiness-count` was not created. Other packages passed.
- The exact failing production-entry test then passed 5/5 uncached in isolation
  (9.235s). This establishes a timing-sensitive failure rate for the bounded
  rerun but does not turn the failed full suite into success.
- `task-board` 0.24.3-169-ge4022da4, Git 2.53.0, and Go 1.25.5 darwin/arm64 were
  checked before review work. This run is not goal-bound and had no directives.

## Evidence files

- `TASK-260830-27u51n_reviewer-focused-01.log`
- `TASK-260830-27u51n_reviewer-broadened-policy-mutant-01.log`
- `TASK-260830-27u51n_reviewer-go-test-all-01.log`
- `TASK-260830-27u51n_reviewer-readiness-repro-01.log`
