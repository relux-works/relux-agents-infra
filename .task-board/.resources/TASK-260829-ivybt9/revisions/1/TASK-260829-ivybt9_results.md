# TASK-260829-ivybt9 developer replay evidence

## Replay identity

- Protected base: `6d051f54440d36e3ca3d132f8d9d1e78d46289de`.
- `HEAD`, selected Story branch tip, local `main`, freshly fetched `origin/main`, and `git ls-remote origin refs/heads/main` all resolved to that exact OID before replay.
- Canonical source: `.task-board/.resources/TASK-260829-1qh0ud/TASK-260829-1qh0ud_change-request_rev7.patch` in the authoritative control root.
- Source SHA-256: `7e377be3bdbe65516820fcfa39cec620f0ca7afed60d1dcb72d8638410d475f5`.
- `git apply --check` exit `0`; `git apply` exit `0`.
- Alternate-index snapshot after replay and again after all validation: regenerated binary patch SHA-256 `7e377be3bdbe65516820fcfa39cec620f0ca7afed60d1dcb72d8638410d475f5`, 22 paths, candidate tree `dabd04a99420aceb21005de65221426bba252c37`; both corrected identity gates exited `0`.

The task precondition named `accepted-pressure-revision7.patch` is a locator file whose bytes are the relative canonical path, so its own SHA-256 is not the payload SHA-256. The locator was resolved against the control root; the canonical payload above is byte-identical to historical CR revision 7.

## Exact 22-path scope

1. `LOGBOOK.md`
2. `README.md`
3. `SKILL.md`
4. `tools/agents-infra/internal/infra/pi_config.go`
5. `tools/agents-infra/internal/infra/pi_shared_attestation_test.go`
6. `tools/agents-infra/internal/infra/pi_shared_broker_darwin.go`
7. `tools/agents-infra/internal/infra/pi_shared_client_darwin.go`
8. `tools/agents-infra/internal/infra/pi_shared_integration_test.go`
9. `tools/agents-infra/internal/infra/pi_shared_launcher_test.go`
10. `tools/agents-infra/internal/infra/pi_shared_operator_darwin.go`
11. `tools/agents-infra/internal/infra/pi_shared_protocol.go`
12. `tools/agents-infra/internal/infra/pi_shared_resources.go`
13. `tools/agents-infra/internal/infra/pi_shared_resources_test.go`
14. `tools/agents-infra/internal/infra/pi_shared_shape_oracle_test.go`
15. `tools/agents-infra/internal/infra/pi_shared_unsupported_posix.go`
16. `tools/agents-infra/internal/infra/pi_shared_unsupported_windows.go`
17. `tools/agents-infra/internal/infra/pi_standalone_shared_test.go`
18. `tools/agents-infra/internal/infra/pi_standalone_test.go`
19. `tools/agents-infra/internal/infra/pi_test.go`
20. `tools/agents-infra/internal/infra/testdata/shared-runtime-resource-observation-v1.json`
21. `tools/agents-infra/internal/infra/testdata/shared-runtime-resource-status-v1.json`
22. `tools/agents-infra/runtime_main_darwin_test.go`

No product bytes were authored beyond the immutable patch. Its `LOGBOOK.md` entries are the task's institutional-memory update; no additional logbook edit was added because that would change the accepted tree.

## Production negative evidence

The focused race gate drives the real `sharedBrokerServer.handleConnection` status/acquire entry points through `observeResourceStatus`, `acquireLease`, and `resourceStatusSnapshot`, using deterministic fake provider observations and task-owned local connection pairs. It covers:

- both revision-6 schedules: pressure between observation/publication, and repeated healthy status polling racing healthy acquisition;
- pressured-status invalidation of pending admission;
- configured/effective policy mismatch in every field, disabled/absent policy, record-derived provenance, unknown reads, stale completion order, draining precedence, pressure refusal/recovery;
- preservation of existing leases and the broker-owned runtime, including no duplicate runtime start.

The full uncached suite also re-runs restart/quarantine/status composition, protocol/fixture compatibility, authorization/attestation shape, standalone composition, and all touched-package negative cases. Historical reviewer narrowed mutants remain applicable because the replayed source and test tree are byte-identical; fresh independent reviewer authority is still required for the replacement CR.

No live model, user runtime, daemon, external service, external endpoint, or user-owned socket was contacted or mutated.

## Validation exits

| Gate | Exit | Evidence |
| --- | ---: | --- |
| `git fetch origin refs/heads/main:refs/remotes/origin/main` | 0 | Fresh authority fetch. |
| `git ls-remote --exit-code origin refs/heads/main` | 0 | GitHub main returned exact protected OID. |
| `git apply --check <canonical-rev7.patch>` | 0 | Patch applies to exact base. |
| `git apply <canonical-rev7.patch>` | 0 | Immutable replay. |
| Initial identity comparison using ordinary `git diff --name-only` | 1 | Honest method failure: ordinary worktree diff omitted four new/untracked files; no product bytes changed. |
| Corrected alternate-index identity gate | 0 | 22 paths, exact patch digest, exact candidate tree. |
| Focused 14-test production resource/status slice under `-race -count=1 -v` | 0 | `internal/infra` passed in `3.522s`. Raw log attached. |
| `go test ./... -count=1` | 0 | Root `82.304s`, attachments `1.271s`, infra `143.815s`, modelharness `0.991s`. Raw log attached. |
| `go vet ./...` | 0 | No diagnostics. |
| `go build ./...` | 0 | Darwin host build. |
| `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ./...` | 0 | Linux build. |
| `CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ./...` | 0 | Windows build. |
| `gofmt -d internal/infra/*.go *.go` plus empty-output assertion | 0 | No formatting diff. |
| `git diff --check` | 0 | No whitespace errors. |
| Post-validation alternate-index identity gate | 0 | Identity remains `7e377… / 22 / dabd04a9…`. |
| Historical/root preservation manifest comparison | 0 | 142-line before/after manifests byte-identical. Manifest attached. |

A preliminary preservation snapshot diagnostic exited `0` while printing missing-relative-path errors because its untracked-file hashes were resolved from the replacement worktree. It was discarded and regenerated from the historical worktree with `pipefail`; only the corrected 142-line manifest is used as evidence.

## Preservation boundary

The compared manifest covers:

- historical `STORY-260829-26nbbv` managed branch OID, worktree status, tracked binary diff, and untracked candidate bytes;
- historical `TASK-260829-1qh0ud` board subtree and every resource/CR/review artifact;
- `.task-board/.element-move-journal.json`;
- control-root dirty `LOGBOOK.md` and every file under `.instructions`.

The before/after comparison exited `0`. Board mutations in this replacement lane were limited to `TASK-260829-ivybt9`, its derived parent aggregation/activity, and its own outcome resources.

## Handoff boundary

Developer evidence is ready. The required final developer handoff must run the configured publication suite (`go test ./... -count=1`, then `go vet ./...`) and publish a new immutable `story_final` Change Request. Independent review and managed Story integration remain Orchestrator-owned; historical CR acceptance is semantic evidence only and is not reused as replacement acceptance authority.
