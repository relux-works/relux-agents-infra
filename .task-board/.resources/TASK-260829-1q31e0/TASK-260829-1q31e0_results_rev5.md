# TASK-260829-1q31e0 revision 5 developer outcome

## Result

Revision 5 reconstructs the immutable revision-4 lifecycle-retention semantics
on exact merged-rotation trunk
`6d051f54440d36e3ca3d132f8d9d1e78d46289de`. The source patch digest was
verified as
`2c69dc0a1a412ba53b09b241bbe4bf6aed3520962bfe516e0f30fba88ffa60da`.
No live Pi/model process, service, socket, or endpoint was contacted.

## Implementation

- Every managed profile requires explicit positive `max_count`, `max_bytes`,
  and `max_age_seconds`; there are no numeric defaults.
- One profile lock serializes lifecycle admission, reservation, aggregate scan,
  and deterministic newest-prefix pruning across canonical `logs/` and every
  exact `runs/<run-key>/logs/` tree.
- Launcher ownership is durable and non-self-mintable: every managed file has a
  same-inode hard-link provenance entry in the private mode-0700
  `lifecycle-log-ownership/` namespace. Matching filename/mode alone remains
  foreign and cannot authorize deletion.
- Every active file holds its own lock. Scanner activity detection precedes
  permissive foreign classification; damaged active evidence reports
  `unknown`.
- `piSessionLog.event` revalidates fd, public path, provenance identity, type,
  mode, and link count before reservation and again immediately before append.
  Failure drops the whole record and keeps status fail-closed.
- `--print-config`, primary-session compose, and managed run reports expose
  configured policy, aggregate root, participating directories, managed,
  foreign, active, byte, age, pruning, dropped-record, error, and
  `within_policy` evidence.
- Merged shared-runtime rotation remains intact: no
  `pi_shared_log_rotation*` path changed, and its explicit
  `max_segment_bytes/max_segments` configuration coexists with lifecycle
  retention in `pi_config.go`.

## Adversarial evidence

- Active mode, link-count, and path-identity mutations through the production
  `piSessionLog.event` gate drop all attempted records and report
  `observation=unknown`.
- A self-minted lifecycle filename/mode shape survives production
  `openPiSessionLogAt` pruning and remains foreign.
- A post-reservation path-displacement race writes to neither inode.
- Twelve concurrent production-shaped run IDs against `max_count=4` admit
  exactly four active logs, preserve foreign evidence, and remain bounded.
- Narrowing only the second production ownership check to open-fd validation
  made the named TOCTOU test fail at exit 1 after writing 110 bytes to the
  displaced inode. Exact source SHA-256 matched before/after restoration, and
  the restored test passed at exit 0.

## Deterministic soak

Eight simulated weeks use a fresh production-shaped run ID for every turn:

| Metric | Observed | Bound |
| --- | ---: | ---: |
| Managed files | 5 | 5 |
| Managed bytes | 405 | 420 |
| Expired files | 0 | 0 |
| Foreign files preserved | 1 | 1 |
| Within policy | true | true |

## Validation

Every command ran directly as a foreground process without `tee`. The uncached
full module suite was split into bounded package calls per the headless-run
contract; together the calls cover every package returned by `go list ./...`.

| Command | Exit | Evidence |
| --- | ---: | --- |
| focused lifecycle/config/status suite, `-count=1 -v` | 0 | `focused-adversarial-rev5.log` |
| focused lifecycle ownership/concurrency `-race` suite | 0 | `race-rev5.log` |
| concurrent production admission `-count=50` | 0 | `concurrency-stress-rev5.log` |
| narrowed post-reservation ownership mutant | 1 expected-red; restored rerun 0 | `narrowing-mutant-rev5.log` |
| `go test ./internal/infra -count=1` | 0, 191.885s | `full-infra-rev5.log` |
| `go test . -count=1` | 0, 79.479s | `full-root-rev5.log` |
| remaining three module packages, `-count=1` | 0 | `full-other-packages-rev5.log` |
| `go vet ./...` | 0 | `go-vet-rev5.log` |
| `go build ./...` | 0 | `go-build-rev5.log` |
| Linux amd64 root + infra compile-only | 0 / 0 | `cross-platform-compile-rev5.log` |
| Windows amd64 root + infra compile-only | 0 / 0 | `cross-platform-compile-rev5.log` |
| explicit changed-file `gofmt -l` emptiness | 0, empty | `format-diff-rev5.log` |
| `git diff --check` | 0, empty | `format-diff-rev5.log` |

## Handoff

The current worktree contains 17 repository paths: four new lifecycle files and
13 current-trunk source/document updates. It is ready for a fresh immutable
Change Request and independent review.
