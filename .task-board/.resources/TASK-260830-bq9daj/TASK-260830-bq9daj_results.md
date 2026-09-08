# TASK-260830-bq9daj: version the local runtime status payload

Commit: 617d3bca27481c0cb456e42ce3bfbfd7b2242da1 (branch task-board/story/STORY-260830-i540ax)

## What changed

1. **Explicit contract version.** `SharedRuntimeStatus` now carries `contract_version` (int).
   `SharedRuntimeStatusContractVersion = 1` and
   `SharedRuntimeStatusMinSupportedContractVersion = 1` (tools/agents-infra/internal/infra/pi_shared_supervision.go).
   There was no prior explicit version field, so the contract starts at 1, not 2.

2. **Fail-closed decode at a real consumer entry point.** `DecodeSharedRuntimeStatus(data []byte)`
   inspects `contract_version` before trusting any other field and refuses anything outside
   `[SharedRuntimeStatusMinSupportedContractVersion, SharedRuntimeStatusContractVersion]` with
   `shared_runtime_status_unsupported_contract_version`, naming both the observed version and the
   supported range in `SharedRuntimeError.Details`. `agents-infra runtime status --json`
   (tools/agents-infra/main.go) round-trips its own freshly-marshaled output through this exact
   function before printing, so the gate runs on every real invocation, not only in tests.

3. **Bounded failure/backoff evidence.** `SharedRuntimeRestartLedger` and `SharedRuntimeStatus`
   both carry `failure_history []SharedRuntimeFailureEvent` (occurred_at, restart_count, and
   either backoff_seconds or quarantined/quarantined_until). `sharedRuntimeRecordFailure`
   (the function the broker calls on every failed restart attempt) appends to it through
   `appendSharedRuntimeFailureEvent`, which enforces `sharedRuntimeFailureHistoryLimit = 20` by
   evicting the oldest retained entry first (FIFO). This supersedes the `last_failure`/
   `last_failure_at` scalar fields that were explicitly deferred in commit 4ca9e0a; those remain
   absent by name.

4. **Docs.** README.md and SKILL.md document the compatibility rule and the bounded-evidence
   shape; their pinned doc-contract tests (`pi_operator_docs_test.go`) were updated to match.

5. **Cross-platform parity.** `contract_version` and `failure_history` fields plus a
   `DecodeSharedRuntimeStatus` mirror were added to the posix (`!darwin && !windows`) and windows
   stub builds so `tools/agents-infra` (which is not platform-gated) still builds on all three
   platforms; those platforms remain otherwise unsupported (`unsupportedSharedRuntimePlatform`).

## Negative-evidence verification (mutation, not just construction)

- Reverted `appendSharedRuntimeFailureEvent`'s trim to a bare append and reran
  `TestSharedRuntimeFailureHistoryIsBoundedAndEvictsOldestFirst` /
  `TestSharedRuntimeStatusReportSurfacesBoundedFailureHistory`: both went red (25/27 retained
  instead of 20). Restored the real code; both pass again.
- Disabled the version-range check in `DecodeSharedRuntimeStatus` (`if false && (...)`) and reran
  `TestDecodeSharedRuntimeStatusRefusesUnsupportedContractVersion`: it went red (a `contract_version: 2`
  payload was silently admitted with its fields parsed). Restored the real code; passes again.

## Test/build evidence

- `go build ./...`: clean on darwin (native), and cross-compiled for
  `GOOS=windows GOARCH=amd64` and `GOOS=linux GOARCH=amd64`.
- `go vet ./...`: clean on all three of the above.
- `gofmt -l .`: no output (all files formatted).
- `go test ./... -v`: **484 passed, 0 failed, 3 skipped** (pre-existing intentional
  subprocess-helper entry points — `TestSharedRuntimeLeaseHelper`, `TestSharedRuntimeLockHolderHelper`,
  `TestPortHolderHelperProcess` — unrelated to this change). Ran twice (before and after the
  cross-platform refactor) with identical counts.

## Scope discipline

- Did not change the meaning of `restart_count`, `restart_not_before`, `quarantined_until`,
  `last_readiness_match`, `manual_quarantine`, or `half_open`.
- Did not touch mode-0600 ledger file scoping or any operator-configured bound
  (`restart_limit`, `restart_initial_backoff_seconds`, `restart_max_backoff_seconds`,
  `stable_run_seconds`, `quarantine_seconds`).
- Did not wire `agents-management` or register `local-qwen`; those are the two sibling tasks per
  the story brief ("do not invent a consumer").
