# TASK-260830-bq9daj \u2014 review verdict: ACCEPTED

## Empty repository_delta
The candidate tree equals base (617d3bc) because the implementation was already
committed directly as 617d3bc ("TASK-260830-bq9daj: version SharedRuntimeStatus
and bound failure evidence"), which is already the tip of this branch. There is
nothing left for this CR to add; it correctly carries zero diff. Verified this
is not a stale/no-op claim by reading and testing the actual content of 617d3bc.

## What I attacked (not just read)

1. **Consumer path.** `DecodeSharedRuntimeStatus` is called from exactly one
   production JSON-emission site: `runtime status --json` in main.go:582,
   which marshals the report and round-trips it through Decode before
   printing. `SharedRuntimeStatusReport` (the only producer of a
   `SharedRuntimeStatus` value) has exactly one caller (main.go:570). The
   plain-text path (`printSharedRuntimeStatus`) reads the in-memory struct
   directly, never decodes a wire payload, so it correctly has no gate to
   bypass. No broker/HTTP path emits this type. Confirmed via grep across the
   whole tree, not just the file the test touches.

2. **Bound enforcement at every append site.** `appendSharedRuntimeFailureEvent`
   is the only writer of `ledger.FailureHistory` (grep confirmed); it is
   called from both branches of `sharedRuntimeRecordFailure` (quarantine and
   backoff), which itself has exactly two production call sites, both in
   `pi_shared_broker_darwin.go`. No second append path exists to bypass the
   trim.

3. **Mutation-tested myself, not trusted the producer's report:**
   - Neutered the version-range check (`if false {`) \u2192 
     `TestDecodeSharedRuntimeStatusRefusesUnsupportedContractVersion` went red,
     admitting `contract_version: 2` with fields parsed. Restored, clean diff
     after.
   - Removed the trim in `appendSharedRuntimeFailureEvent` \u2192 both
     `TestSharedRuntimeFailureHistoryIsBoundedAndEvictsOldestFirst` (25 retained
     vs 20 wanted) and `TestSharedRuntimeStatusReportSurfacesBoundedFailureHistory`
     (27 vs 20) went red. Restored, clean diff after.
   - Additionally mutated eviction direction (kept oldest 20, dropped newest 5
     instead of FIFO) to specifically attack claim #4 in the spawn brief \u2014 this
     also went red (`eviction order broken`), confirming the assertion checks
     order, not just count. Restored, clean diff after.
   - Ran the full suite after restoring: `go test ./...` \u2192 all packages `ok`,
     484 `--- PASS` lines, 0 failures, matching the producer's reported count
     exactly.

4. **Older/missing version.** `SharedRuntimeStatusMinSupportedContractVersion`
   is 1, so any payload written before this change (no `contract_version` key,
   unmarshals to 0) is refused, not silently accepted or migrated \u2014 confirmed
   by the `malformed := []int{0, -1}` subtest in
   `TestDecodeSharedRuntimeStatusRefusesUnsupportedContractVersion`. This is
   the correct fail-closed answer for a payload that predates the contract:
   there is no prior explicit version to migrate from (contract_version did
   not exist before 617d3bc), so refusing rather than guessing is right.
   README/SKILL document the version-bump rule (additive fields never bump;
   redefinition/removal does) but don't spell out the zero/missing-field case
   in prose \u2014 the code and test make the answer unambiguous regardless.

5. **Test delta reconciliation.** 617d3bc added 5 new top-level test funcs to
   `pi_shared_supervision_test.go` (10\u219215) and net-strengthened two existing
   tests without weakening them:
   - `runtime_main_darwin_test.go`: added `contract_version`/`failure_history`
     to the required-fields list (additive) plus new assertions that
     `ContractVersion` matches and that emitted JSON decodes through
     `DecodeSharedRuntimeStatus` \u2014 strictly more coverage than before.
   - `pi_operator_docs_test.go`: doc-fixture string literals updated to match
     the new README/SKILL prose (the old text itself changed from "deferred"
     to "remain absent as separate scalar fields"); this tracks doc content,
     it does not loosen a behavioral assertion.
   No existing test was deleted or weakened. The 482\u2192484 delta the producer
   reported is `--- PASS` line count (which includes subtests), not top-level
   func count, and I reproduced 484 exactly on a clean run.

## Minor non-blocking observation
The Windows stub `DecodeSharedRuntimeStatus` (`pi_shared_unsupported_windows.go`)
constructs its unsupported-version error without the `Details` map
(`observed_contract_version`/`supported_contract_version_min/max`) that the
darwin/posix implementation carries \u2014 the message text still names the
version and range, just not structurally. This is inert in practice:
`SharedRuntimeStatusReport` on Windows already fails immediately with
`unsupportedSharedRuntimePlatform()`, so production Windows code never reaches
`DecodeSharedRuntimeStatus`. Not a blocker; noting for anyone hardening the
Windows stub later.

## Verdict
ACCEPT. Contract version field, fail-closed refusal at the real consumer
entry point, and bounded+FIFO failure evidence are all real, wired into
production, and independently confirmed red-without-the-guard by my own
mutations, not just the producer's claims. Full suite green (484/484, 0
failed) on a clean tree.
