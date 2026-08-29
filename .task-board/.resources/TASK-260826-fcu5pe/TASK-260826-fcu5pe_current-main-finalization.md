# TASK-260826-fcu5pe current-main finalization evidence

## Candidate identity

- Accepted revision 3 commit: `7ef425820de3174b535907fbc085a091be1de8c0`
- Accepted revision 3 tree: `e3b84df47ed9c1dec2404fe4146dac12ed56a509`
- Reconciled candidate commit: `727a45b019ba2ee98a75577565e6e8f5a1f212a1`
- Reconciled candidate tree: `e2951b27d3e2863cde557887a9c0614f3d1ea0c0`
- Current main: `e70f953969d46e451892d9f16e7401b879910b6b`
- Story fork point: `b3cb84550a60f7f4df92a287c573bfc692cd26e0`
- Story/main divergence after reconciliation: `24 / 4` unique commits.
- Commit author and committer time: `2026-08-25T20:01:00+03:00`, matching repository policy.

## Reconciliation

The accepted security-sensitive broker candidate was treated as fixed input.
Current main's two product candidates were already composed on the Story branch
as `3d203b1` and `5e3bbe0`; main's separate board-state commits remain outside
the Story repository candidate.

The integration-base movement overlapped only `LOGBOOK.md`. The accepted tree
already retained every current-main Qwen/model-check entry, but its manual
composition omitted one blank line between the `1210` entry and the
`2026-08-24` heading. Commit `727a45b` restores that byte boundary.

Scope proofs:

- `git diff --exit-code 7ef4258..HEAD -- . ':(exclude)LOGBOOK.md'` exited 0.
- `git diff --name-status 7ef4258..HEAD` reports only `M LOGBOOK.md`.
- `git diff --unified=0 main..HEAD -- LOGBOOK.md` contains no deleted main
  content; the Story's shared-runtime history is additive.
- Worktree status is clean after the reconciliation commit.
- Therefore every non-LOGBOOK product blob, including all shared-runtime,
  13-gate attestation, launcher authorization, lease, CLI, mutation, Qwen
  thinking, model-check, and standalone-yolo files, is byte-identical to the
  independently accepted revision 3 candidate.
- No task-board Pi adapter source or standalone Pi yolo policy changed.

## Validation

Every gate ran directly as a standalone foreground process on candidate tree
`e2951b27d3e2863cde557887a9c0614f3d1ea0c0`. No gate used `tee`, no gate was
backgrounded, and the exits below are the real command exits.

| Command | Exit | Result |
| --- | ---: | --- |
| fail if `gofmt -l .` is non-empty | 0 | no unformatted Go files |
| `git diff --check` | 0 | clean |
| named production-entry mutant/oracle/attestation/broker-reclaim suite | 0 | `21.799s` |
| `go test ./internal/infra -count=1 -run '^(TestShared\|TestConnectAndAttestSharedRuntime\|TestRunSharedRuntimeBroker\|TestReclaimSharedRuntime)'` | 0 | `30.686s` |
| race form of the same focused shared-runtime suite | 0 | `48.158s` |
| `go test ./... -count=1` | 0 | root `129.350s`; attachments `3.379s`; infra `184.868s` |
| `go vet ./...` | 0 | configured landing gate |
| `go build ./...` | 0 | Darwin |
| `GOOS=linux GOARCH=amd64 go build ./...` | 0 | Linux amd64 |
| `GOOS=windows GOARCH=amd64 go build ./...` | 0 | Windows amd64 |

The exact configured landing commands from `task-board.config.json` are the
full uncached Go suite and `go vet ./...`; both exited 0 after the commit.

The named production-entry command includes the shape oracle, mutant
calibration and reject-all harness negative, launcher guard refusals, all client
attestation/report gates, broker admission delete/narrow witnesses, draining,
wire bound, recomputed runtime key, and reclaim UID/PGID authorization. The
production call sites remain the accepted revision 3 sites; this reconciliation
did not change them.

## Attached logs

- `TASK-260826-fcu5pe_go-test-all-current-main-final.log`
- `TASK-260826-fcu5pe_go-test-race-current-main-final.log`
- `TASK-260826-fcu5pe_go-test-production-entry-current-main-final.log`
- `TASK-260826-fcu5pe_go-test-focused-current-main-final.log`

## Handoff

The reconciled candidate is suitable for a fresh `story_final` Change Request
against current main and independent review. The spawn runtime publishes that
Change Request after the required developer handoff.
