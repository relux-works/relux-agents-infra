# TASK-260830-tvy8q5 revision-3 developer handoff recovery

## Candidate and repair

- Exact Story worktree HEAD: `40d4e4f8c17387f04fa99cb26585f396e8f07cc0`; the managed branch is seven commits behind local `main`, and this developer run did not switch, rebase, merge, or integrate it.
- The recovery precondition was stale relative to the attached revision-2 reviewer verdict. Exact-diff inspection confirmed the blocking tombstone-only retry bypass remained in `PiLegacyRetire -> resumePiLegacyRetirement`.
- Added `TestPiLegacyRetirementResumesTwoProgressWindowCrashes`, which composes a crash after rename but before progress persistence with a second crash after resumed unlink but before progress persistence.
- Expected-red command: `go test ./internal/infra -run '^TestPiLegacyRetirementResumesTwoProgressWindowCrashes$' -count=1`; exit 1. The odd generation incorrectly retained `candidate_renamed=false` after the second crash.
- Repair: after proving the exact operation-bound tombstone, the retry persists `candidate_renamed=true` before unlink. Unexplained source+tombstone absence without persisted authority remains typed unknown.
- Updated `LOGBOOK.md` with the retry-path root cause and corrected invariant.

Final tracked binary diff SHA-256: `33397162661bbc00bc4b77b1d70c25188035f89e7b3c5e7ae3cab425fdea6f55`.

Untracked candidate SHA-256 values:

- `pi_lifecycle_legacy.go`: `595d0e8d23451dfae1a3db1ab383439790aca3f38e63979a4380f3e8368d1c8d`
- `pi_lifecycle_legacy_test.go`: `5b39f40d6f38bc8b8b49954293341f371af5b9d7daa685a4749b740f522496c8`
- `pi_lifecycle_soak_darwin_test.go`: `1335486eae60dbc2620ce29c73dcb0310633a82eafcb51126b6ec2825dfb6102`
- `pi_lifecycle_main_test.go`: `ef5d1c42b38d54f58300f6cd2df978b8975466cdd1df91cf98aabed2e5d4e3eb`

## Validation

Every command ran directly as a standalone foreground process without `tee`.

- Exact external-absence/two-crash production-entry pair: exit 0 (`0.748s`).
- Focused legacy, automatic-path, and deterministic eight-week crash/lease/reload/pressure suite: exit 0 (`5.757s`).
- Root lifecycle CLI/operator-doc/skill suite: exit 0 (`1.249s`).
- Race gate over external absence, ordinary post-unlink recovery, and two-crash recovery: exit 0 (`2.120s`).
- `go test ./... -count=1`: exit 0; root `113.215s`, attachments `1.865s`, infra `220.160s`, modelharness `1.408s`.
- `go vet ./...`: exit 0.
- `go build ./...`: exit 0.
- Darwin arm64 compile-only gate: exit 0.
- Linux amd64 compile-only gate: exit 0.
- Windows amd64 compile-only gate: exit 0.
- `gofmt -l` over every changed Go file: exit 0 with empty output.
- `git diff --check`: exit 0 with empty output.

## Isolation attestation

Validation used deterministic temporary filesystem fixtures and fake clocks. It did not launch or contact Pi, a model, a runtime, a provider process, a service, a socket, an endpoint, or a network service; it performed no setup/install flow and used no live user-HOME runtime state.

The developer candidate is ready for independent review. Review, Change Request acceptance, and Story integration remain reviewer/orchestrator-owned.
