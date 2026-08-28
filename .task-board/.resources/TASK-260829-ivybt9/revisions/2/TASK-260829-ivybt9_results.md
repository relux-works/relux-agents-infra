# TASK-260829-ivybt9 revision 2 developer evidence

## Rework delta

- Added one production-entry negative test in `tools/agents-infra/internal/infra/pi_shared_resources_test.go`.
- The test drives `SharedRuntimeStatusReport`, completes the fake-backed attestation handshake, serves an otherwise valid status response with the `resources` field absent, and requires typed `protocol_violation`.
- No production file changed relative to immutable revision 1. Tree diff from revision 1 is one test file, 60 inserted lines.
- Static fake runtime/provider and task-owned local test transports only; no user model runtime, daemon, service, endpoint, or socket was contacted.

## Negative evidence

| Gate | Exit | Result |
| --- | ---: | --- |
| Named test, production guard intact | 0 | PASS |
| Named test, permissive absent-resources mutant | 1 | Expected red: `err=nil` was rejected by the assertion |
| Restore comparison (`cmp -s`) | 0 | Production file restored byte-for-byte |
| Named test after restore | 0 | PASS |

Production call site: `SharedRuntimeStatusReport` in `pi_shared_operator_darwin.go`. The mutant replaced the refusal with a branch that retained the configured fallback when `message.Resources == nil`; the new test failed exactly on that silent admission.

## Validation

| Command | Exit | Evidence |
| --- | ---: | --- |
| `go test ./internal/infra -race -count=1 -v -run <prior 14-test production slice>` | 0 | `revision2-focused-14-race.log` |
| `go test ./internal/infra -race -count=20 -v -run <two revision-6 schedules>` | 0 | `revision2-revision6-schedules-race.log` |
| `go test ./... -count=1` | 0 | Root 82.553s; infra 139.461s; attachments/modelharness green; `revision2-go-test-all.log` |
| `go vet ./...` | 0 | No diagnostics |
| `gofmt -d internal/infra/*.go *.go` and empty-output assertion | 0 | No formatting delta |
| `git diff --check` | 0 | No whitespace errors |
| `go build ./...` | 0 | Darwin host build |
| `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ./...` | 0 | Linux amd64 build |
| `CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ./...` | 0 | Windows amd64 build |

## Identity and preservation

- Fetched upstream and remote authority: `HEAD`, local `main`, `origin/main`, and `git ls-remote origin refs/heads/main` all equal `6d051f54440d36e3ca3d132f8d9d1e78d46289de`.
- Immutable source revision 7 patch SHA-256: `7e377be3bdbe65516820fcfa39cec620f0ca7afed60d1dcb72d8638410d475f5`; reverse apply check on the replayed tree exited 0 before rework.
- Immutable revision 1 replay identity before rework: candidate tree `dabd04a99420aceb21005de65221426bba252c37`, exactly 22 paths.
- Revision 2 candidate identity: tree `eb94a41aefe919622f6031450c936df8ffe20ac1`, generated patch SHA-256 `a8870cdca9f73a148e508c97d8ac2b8919fe2e401c5d4db1d7aab98195ac8985`, exactly the same 22 paths.
- Historical Story `STORY-260829-26nbbv` worktree HEAD and branch remain `6d051f54440d36e3ca3d132f8d9d1e78d46289de`; its tracked binary diff digest remains `8c6802ec737f3ede9fdc502f11f420e844c33fcfb63ca93c85ee03bf22238a39`.
- 114 hashes covering the historical Task/Story resources and CRs, move journal, root dirty `LOGBOOK.md`, and `.instructions` matched revision 1 preservation evidence (`shasum -c` exit 0).

Revision 2 is ready for a fresh independent review. Historical revision 1 remains changes-requested and is not acceptance authority.
