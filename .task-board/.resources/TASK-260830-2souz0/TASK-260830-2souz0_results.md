# TASK-260830-2souz0 developer outcome

## Handoff

The current-trunk replay and lifecycle authority hardening are ready for independent review.
No live model, Pi runtime, daemon, service endpoint, or socket was launched, contacted, or inspected.

## Base and replay authority

- Fresh `task-board worktree status`, workspace `HEAD`, fetched `origin/main`, `FETCH_HEAD`, and direct `git ls-remote origin refs/heads/main` all resolved to `3295c7da7151de128f176cf7560a57d54c8f6c0d` before production changes.
- Local `refs/heads/main` remained the stale `5c9b4e4f7a88e1eb937b80851af522e4fa4b066f`; it was neither selected nor moved.
- Preserved revision-2 patch size `184182` and SHA-256 `5c162f2050c288039651599e5f8c7fc252e1455412e03d145a7441d5faece553` were independently verified and used only as replay/review input.
- The patch applied cleanly to current protected trunk. Current-trunk `.configs`, `.instructions`, and `infra_test.go` bytes remain unchanged; the README retains the `gpt-5.6-sol` 272K/245K context policy.

## Production changes

- Close recovery now distinguishes the exact pre-publication record from exact completion and accepts completion only when `record.closed_at == odd.started_at`.
- Odd delete generations persist tombstone plus `record.json`, `log.jsonl`, and `active.lock` filesystem identities. Recovery validates strict mode/trusted UID/type/link rules, holds descriptors, revalidates fd/path device+inode+mode+UID immediately before each bounded unlink, and preserves narrowed, replaced, linked/symlinked, or unreadable evidence as typed unknown.
- One strict generation validator now owns status and next-writer authority: even state has zero residual operation fields; odd state validates canonical timestamp, entry/staging grammar, operation ID, counters, exact create/append/close/delete fields, and delete identities.
- Darwin and Linux directory-cookie parsing no longer fabricates `*unix.Dirent` from a short byte buffer; bounds-checked native-endian decoding is checkptr-safe.
- README, source skill, operator tests, and `LOGBOOK.md` describe the strict lifecycle contract and root causes.

## Negative evidence

The initial production-entry mask exited `1` in `4.948s` on replayed revision-2 bytes. It admitted:

- forged close-looking timestamps;
- a mode-narrowed tombstone;
- a substituted child inode;
- even residual `started_at`/counter authority as healthy status;
- malformed odd timestamp/staging/zero-byte-append authority to the next writer.

After hardening, the same classes pass through real `PiLifecycleStatus` and `openPiSessionLog` entry points. The final tests additionally cover all three delete children: narrowed `record.json`, substituted `log.jsonl`, and symlink-substituted `active.lock`, plus residual delete authority and an untrusted stored child UID.

## Validation history

All commands were direct standalone processes; no output was piped through `tee`.

| Command / gate | Exit | Evidence |
| --- | ---: | --- |
| Fresh-base fetch/ref reads and `git apply --check` | 0 | Exact base `3295c7d`; clean replay check |
| Initial authority negative mask, before fixes | 1 | Expected red; forbidden states admitted, package `4.948s` |
| First post-fix authority mask | 1 | Compile error: out-of-scope local `err` at `pi_session_log.go:906`; corrected |
| Authority mask after compile correction | 0 | Package `1.419s` |
| First full lifecycle mask | 1 | Directory link count changed during partial delete; exact failures preserved |
| Exact delete-resume + eight-week soak rerun | 0 | Package `2.751s` |
| Full lifecycle mask after directory identity correction | 0 | Package `13.982s` |
| Expanded three-child substitution mask | 0 | Package `2.168s` |
| Expanded malformed even/odd mask | 0 | Package `3.916s` |
| First focused lifecycle + pressure race | 1 | Go checkptr abort in unsafe Darwin dirent conversion |
| Focused race after bounds-checked dirent decoding | 0 | Package `6.939s` |
| Pre-final uncached `internal/infra` split | 0 | Main slice `153.812s`; exact timing fixture `2.950s` |
| Pre-final uncached root package | 0 | `114.289s` |
| Final focused authority/close/delete/soak mask | 0 | Package `5.014s` |
| Final focused lifecycle + pressure race | 0 | Package `19.919s` |
| Final `go test -count=1 ./internal/infra -skip ...` | 0 | Package `329.480s` |
| Final exact skipped timing fixture | 0 | Package `2.589s` |
| Final `go test -count=1 .` | 0 | Package `84.732s` |
| Final `go test -count=1 ./cmd/model-harness` | 0 | No test files; package compiled |
| Final `go test -count=1 ./internal/attachments` | 0 | Package `1.213s` |
| Final `go test -count=1 ./internal/modelharness` | 0 | Package `0.887s` |
| Final `go vet ./...` | 0 | Empty output |
| Final `gofmt -l .` | 0 | Empty output |
| Final `git diff --check` | 0 | Empty output |
| Final Darwin `go build ./...` | 0 | Empty output |
| Final Linux/amd64 `go build ./...` | 0 | Empty output |
| Final Windows/amd64 `go build ./...` | 0 | Empty output |

The earlier exit-1 runs remain failures and are not relabeled as passing. The final gates bind to the current landing tree.

## No-live-runtime evidence

Only repository inspection, Git remote authority reads, task-board operations, Go tests/vet/builds, formatting, and filesystem-backed test fixtures were used. No `pi-infra`, `qwen-infra`, model checker, runtime executable, HTTP readiness URL, daemon, broker socket, or live service command was invoked.
