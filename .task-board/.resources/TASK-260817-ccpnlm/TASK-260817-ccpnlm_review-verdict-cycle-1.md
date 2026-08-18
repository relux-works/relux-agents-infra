# TASK-260817-ccpnlm reviewer verdict — changes requested

Verdict branch: `changes_requested -> to-dev`.

## Findings

1. **P1 — The production argv bridge admits a forbidden bare unknown long option.**
   Decision section 5 permits unknown long flags only as self-contained `--name=value` or as a complete flag/value pair before the wrapper delimiter. `BuildManagedPiArguments` instead forwards a bare unknown token when it has no suffix operand (`pi_args.go:220-232`). The existing negative test covers only `--unknown -- prompt` (`pi_test.go:167-174`), leaving a bypass path around the suffix-adjacency check. A production-entry attack ran `agents-infra pi --print-config --unknown`; it exited 0 with `status:"ok"` and emitted final argv ending in `--unknown`. Evidence: `.temp/TASK-260817-ccpnlm-bare-unknown-production-probe.json`. Required rework: reject every bare unknown long option; add a real-entry negative test for the no-suffix form and a narrowing test proving only the two permitted complete forms survive.

2. **P1 — Mandatory named production-entry gate evidence is helper-only and cannot prove production invocation.**
   `TestPiLaunchProfileStateKeyIsolation` calls `ResolvePiStatePaths`, `CreatePiStateTree`, `AcquirePiProfileLock`, and `ValidatePiStateKeyCollisions` directly (`pi_test.go:177-224`). `TestPiLaunchRefusesOccupiedListenerBeforeRuntime` calls only `preflightPiListener` (`pi_test.go:367-379`). `TestPiLaunchReadinessRefusesMalformedMismatchAndDeadChild` calls only `waitPiRuntimeReady` (`pi_test.go:381-409`). `TestPiLaunchRejectsCatalogCanonicalizationNarrowing` calls only `VerifyPiExecutionIdentity` (`pi_test.go:411-485`). These tests do not drive `main.runPi` or `RunPi`, so they do not catch removal/reordering/bypass of the guards in the production launch sequence. This is the standard **check present but uncalled from production** shape and violates AC 7 plus decision section 12's explicit real-entry requirement.

3. **P1 — Section-12 negative/narrowing and recovery matrix is materially incomplete.**
   The shipped Pi suite has only 15 focused tests. Missing required production-entry evidence includes `runtime_listener_check_failed`; runtime executable disappearance/literal shell-metacharacter execution; runtime spawn failure; Pi spawn failure; SIGINT/SIGTERM forwarding; graceful shutdown and timeout escalation; lock release on each error path; cache-root resolution/partial stat/open/post-create revalidation failures; raw/slash-replaced/case-folded/Unicode-normalized state-key mutants; and several Pi catalog narrowing cases. The catalog "manifest record ordering" subtest mutates the embedded manifest bytes, but the compiled manifest digest rejects it first, so it does not prove byte-order validation or defeat a narrowed ordering implementation. The named catalog test also lacks the required path case/normalization and record-encoding subcases. Positive normal lifecycle coverage cannot substitute for these gates.

## Validation performed

- `go test ./internal/infra -run 'Test.*Pi' -count=1` — pass.
- `go test ./... -count=1` — pass.
- `go vet ./...` — pass.
- `go build ./...` — pass.
- `git diff --check` — pass.
- `task-board validate` — pass.
- Production defeat probe `agents-infra pi --print-config --unknown` — gate defeated; exit 0 and forbidden token present in composed argv.

Green validation does not accept the task because a concrete production bypass reproduced and the mandatory real-entry negative evidence is absent.
