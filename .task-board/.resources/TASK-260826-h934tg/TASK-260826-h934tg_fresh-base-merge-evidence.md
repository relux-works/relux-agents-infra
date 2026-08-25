# TASK-260826-h934tg Fresh-Base Merge Evidence

## Result

The accepted standalone Pi YOLO checkpoint was merged with current `main` as a
real two-parent merge. No replay or cherry-pick was used.

| Fact | OID |
| --- | --- |
| Accepted implementation commit | `8f81371d93c75552580bb1530281ea5627f429a1` |
| Accepted implementation tree | `47bc47a3beaf019e46e18c0ae1ab581b7d8e951e` |
| Current main commit / merge second parent | `fd80bd8e0c1de3f372fd1a7527613a5135762de4` |
| Current main tree | `1390cca133adc5fd985d88cf33289fb8cb600884` |
| Fresh-base merge commit | `3a52ec762b93149b6db541612f28bf1a6ccef5ed` |
| Fresh-base merge tree | `02e41e53790b42bfe5cb7cc5c9e19d622a507035` |

`git merge-base --is-ancestor main HEAD` exited `0`. The merge commit parents
are exactly the accepted checkpoint followed by current main.

## Reconciliation

Conflicts were limited to additive documentation plus the genuine Pi overlap:

- `LOGBOOK.md` and `README.md` preserve both mainline and standalone content.
- Interactive Pi applies the new mainline primary-session yolo resolution and
  keeps direct terminal writers through `piProcessWriter`.
- Standalone Pi continues to reject caller Pi arguments, bypasses interactive
  yolo rewriting, closes stdin, and uses isolated client state in both shared
  and exclusive production paths.
- The task-board adapter boundary and exact standalone tool allowlist are
  unchanged.

The six standalone implementation/research/test files that did not require an
actual mainline overlap are byte-identical to accepted commit `8f81371` (`git
diff --exit-code ...` exited `0`). The final 17-path product delta relative to
main is a strict subset of the accepted implementation's 35-path delta; the
unexpected-path gate is empty and exited `0`. The missing 18 paths are shared
runtime/mainline paths already carried by current main.

Production call sites exercised by the reconciliation tests:

- `RunPi` in `internal/infra/pi_launch_posix.go` — interactive-yolo selection,
  standalone argument ownership, exclusive stdin split, terminal writer split.
- `runSharedPiSession` in `internal/infra/pi_shared_client_darwin.go` — standalone
  client identity, shared lease reuse, shared stdin split, terminal writer split.
- `runPiStandaloneCLI` / `runTarget` in `main.go` — bounded standalone dispatch
  and typed failure surface.

## Validation

All commands below ran directly on the reconciled tree. Exit codes are literal.

| Command | Exit | Evidence |
| --- | ---: | --- |
| Focused standalone authorization/concurrency/stdin plus interactive writer tests | 0 | `go-test-reconciliation-01.log` |
| `go test ./... -count=1` after the final merge-tree amendment | 0 | `go-test-full-02.log`: root `81.746s`, attachments `1.914s`, infra `127.626s` |
| `go vet ./...` | 0 | `go-vet-full-02.log` (empty) |
| `gofmt -l .` plus empty-output assertion | 0 / 0 | `gofmt-list-01.log` (empty) |
| `go build ./...` | 0 | native Darwin/arm64 |
| `CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build ./...` | 0 | Darwin amd64 cross-build |
| `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ./...` | 0 | Linux amd64 cross-build |
| `CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ./...` | 0 | Windows amd64 cross-build |
| `git merge-base --is-ancestor main HEAD` | 0 | current main is an ancestor |
| Accepted standalone core byte-parity check | 0 | six non-overlap files unchanged from `8f81371` |
| Product changed-path subset gate | 0 | `unexpected-product-paths-01.log` (empty) |
| `git diff --check main..HEAD -- . ':(exclude).task-board/**'` | 0 | product delta clean |
| `git status --porcelain=v1` | 0 | empty output; worktree clean |

One broad pre-commit `git diff --cached --check` exited `2` on historical
whitespace inside board-resource patch payloads imported unchanged from main.
Those artifacts are not part of the Story product delta and were not rewritten.
The scoped final product diff check above exited `0`.

## Scope

Relative to current main, the final candidate changes exactly these 17 product
paths:

```text
.research/260825_pi-unattended-tool-authorization.md
LOGBOOK.md
README.md
SKILL.md
tools/agents-infra/internal/infra/canonical_target.go
tools/agents-infra/internal/infra/pi_config.go
tools/agents-infra/internal/infra/pi_launch_posix.go
tools/agents-infra/internal/infra/pi_platform_windows.go
tools/agents-infra/internal/infra/pi_shared_client_darwin.go
tools/agents-infra/internal/infra/pi_shared_integration_test.go
tools/agents-infra/internal/infra/pi_standalone.go
tools/agents-infra/internal/infra/pi_standalone_real_pi_test.go
tools/agents-infra/internal/infra/pi_standalone_shared_test.go
tools/agents-infra/internal/infra/pi_standalone_test.go
tools/agents-infra/internal/infra/project_config.go
tools/agents-infra/main.go
tools/agents-infra/pi_standalone_main_test.go
```

No task-board integration, authorization-model change, or allowlist expansion
was introduced.
