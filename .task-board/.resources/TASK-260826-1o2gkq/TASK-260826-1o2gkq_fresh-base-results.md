# TASK-260826-1o2gkq fresh-base merge evidence

## Outcome

The managed Story branch now contains current local `main` as a real merge
parent without changing the accepted CR revision 5 product tree.

- Merge commit: `c40f0c26e071a9b466f1b856bebe91f19fb7390b`
- First parent / accepted candidate: `91adc7328d6a122fbbbb40f42a1d9b6aad5f2ac0`
- Second parent / current local main: `e70f953969d46e451892d9f16e7401b879910b6b`
- Accepted CR rev5 candidate tree: `40a83fe6f3b1544494969edc861f3fe23ffc4757`
- Merge tree: `f92c9b2c2fd0ca11ac00f7fe4d479a7264f6a698`

The merge tree differs from CR rev5 only under `.task-board/**`, where the
second parent contributes current trunk board history. Managed Change Request
publication drops those checkout-artifact paths. Every repository path outside
`.task-board/**`, including `LOGBOOK.md`, is byte-identical to accepted CR rev5.

## Integrity evidence

| Check | Exit | Evidence |
| --- | ---: | --- |
| `git merge-base --is-ancestor main HEAD` | 0 | Current `main` is an ancestor of the Story tip. |
| `git diff --quiet 40a83fe... HEAD -- . ':(exclude).task-board/**'` | 0 | The entire non-board product tree matches accepted CR rev5. |
| CR5 35-path non-LOGBOOK manifest vs post-merge manifest (`cmp -s`) | 0 | Both manifests have SHA-256 `68a2ad65d6831b08d5dd3b127dd68483ed5f969006f4dce92d8499a1965127e1`. |
| `git diff --numstat main 40a83fe... -- LOGBOOK.md` | 0 | `168` additions and `0` deletions: accepted LOGBOOK is an additive superset of main. |
| Final `git status --porcelain=v1 --untracked-files=all`, then empty-output assertion | 0 / 0 | Worktree is clean; task evidence remains ignored under `.temp/`. |

`git merge-tree --write-tree --messages --name-only HEAD main` exited `1` and
the real `git merge --no-ff --no-commit main` exited `1`, both truthfully due to
content conflicts in `LOGBOOK.md`, `README.md`, and `tools/agents-infra/main.go`.
The latter two were mechanical conflicts between equivalent trunk work and the
accepted superset. All 35 non-LOGBOOK CR5 paths were restored from the accepted
tree. `LOGBOOK.md` was resolved to that same accepted tree after proving it
contains every main line with no deletion.

An unscoped pre-commit `git diff --check --cached` exited `2` on whitespace
inside historical Change Request patch payloads newly inherited from main under
`.task-board/.resources/**`. Those payloads are current trunk content and were
not edited. The relevant product-scoped command
`git diff --check --cached -- . ':(exclude).task-board/**'` exited `0`, and the
post-commit product delta check below also exited `0`.

## Fresh validation

Every command below ran as a standalone foreground process after the merge.
Redirection captured output without piping or masking the command status.

| Command | Exit | Result / log |
| --- | ---: | --- |
| `cd tools/agents-infra && gofmt -l .` | 0 | Output empty; `validation/gofmt.log`. |
| Empty-output assertion for the gofmt log | 0 | Module Go files are formatted. |
| `git diff --check main..HEAD -- . ':(exclude).task-board/**'` | 0 | Product delta has no whitespace errors; `validation/git-diff-check-product.log`. |
| `go test ./internal/infra -count=1 -run '^(TestSharedRuntime|TestSharedAuthorization|TestConnectAndAttestSharedRuntime|TestRunSharedRuntimeBroker|TestReclaimSharedRuntime)'` | 0 | `ok` in `43.937s`; `validation/go-test-shared.log`. |
| Same focused suite with `-race` | 0 | `ok` in `65.097s`; `validation/go-test-shared-race.log`. |
| `go test ./... -count=1` | 0 | Root `280.699s`, attachments `14.526s`, infra `397.667s`; `validation/go-test-all.log`. This is the configured landing test gate. |
| `go vet ./...` | 0 | `validation/go-vet.log`. This is the configured landing vet gate. |
| `go build ./...` | 0 | Darwin/arm64 build passed; `validation/go-build-darwin.log`. |

Toolchain evidence: Go `1.25.5` on Darwin/arm64 and Git `2.53.0`.

## Scope and test rationale

This task intentionally changes no product behavior, production gate, test, or
standalone yolo implementation. Therefore it adds no new test case. The
accepted revision 5 negative production-entry witnesses remain byte-identical
and were rerun through the focused suite and its race variant. The production
call sites exercised include shared runtime launch/authorization, client
attestation, broker admission/reclaim, status, and force-stop identity gates.

The only authored repository object is the real merge commit. Its content
preserves current trunk history and the already accepted product candidate,
closing the stale integration-base condition for the next `story_final` Change
Request revision.
