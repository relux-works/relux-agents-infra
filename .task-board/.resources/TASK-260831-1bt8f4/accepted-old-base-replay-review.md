# TASK-260830-84z0be review verdict — Change Request revision 1

## Verdict

Accepted. No blocking correctness, architecture, documentation, scope, or
evidence finding remains in `CR-TASK-260830-84z0be-1` revision 1.

Immutable review surface:

- Base OID: `4270549dd17c010599e2083bf3ec7672af60ea29`
- Candidate tree OID: `913168e4ad563edd38551f8d88cdf00665149536`
- Patch SHA-256: `0c63c3bc5d9ea0496fc2c26c112f9361ee092791681950eff38cbb0023478afb`
- Repository delta: present, exactly 26 changed paths
- Accepted revision-6 input patch SHA-256:
  `1ed35314955527822a6211f11510c10582aa9f588e9558731d1321b837c117ad`

Reviewer-owned alternate-index round trips reproduced both immutable trees:

- `b78498b` plus the accepted revision-6 patch produced
  `57e2a9f3e43d75e806d792fe4675a2572345063a`.
- `4270549` plus the revision-1 Change Request patch produced
  `913168e4ad563edd38551f8d88cdf00665149536`.
- A fresh worktree snapshot after all reviewer gates still produced
  `913168e4ad563edd38551f8d88cdf00665149536`; `git status` still named exactly
  the reviewed 26 paths and `git diff --check` passed.

## Semantic replay and scope

The replay preserves the accepted revision-6 functional delta and current-base
content. The only current-base intersection in the original replay was
`LOGBOOK.md` and `README.md`; the delivered documentation preserves the accepted
retention contract, the trunk External-CI section, and the complete newest-first
LOGBOOK union. The 24 code/test paths and `SKILL.md` remain the accepted
functional implementation, with no extra repository path introduced.

The control repository advanced again after the producer's exact-base preflight:
current local/upstream `main` was `4546253d9fa26f62982a5afcb0891f054abedb6c`
at review time, and its path intersection with this Change Request is exactly
`LOGBOOK.md` and `README.md`. This does not alter or invalidate the immutable
revision-1 review surface. It does mean the Orchestrator must use the canonical
`worktree integrate` path: the moved-base guard must decide the now-current
combination and must not be bypassed. A resulting `integration_base_moved` /
`stale` outcome is ordinary replay rework, not permission to land an unreviewed
union.

## Gate-defeat review

The health attestation is exercised through the production call chain
`runPi -> runPiLifecycleCLI -> PiLifecycleOperatorStatus -> PiLifecycleStatus`.

- Clean `^TestRunPiLifecycle` suite: pass. It used temporary HOME/PATH values,
  a nonexistent runtime path, and never launched Pi or a provider.
- Foreign-evidence narrowed mutant: expected exit 1. Removing only
  `status.ForeignCount == 0` from the production `WithinPolicy` predicate made
  the real CLI test observe `ForeignCount:1`, `WithinPolicy:true`, and
  `SoakReady:true`; the external bytes remained the evidence under test.
- Legacy-evidence narrowed mutant: expected exit 1. Removing only
  `status.LegacyCount == 0` made the real CLI test observe `LegacyCount:1`,
  `WithinPolicy:true`, and `SoakReady:true`.
- The focused 47-test lifecycle suite passed. It covers forged/self-minted
  evidence, absent evidence versus failed reads, narrowed tombstone and child
  authority, substituted inode/mode/type paths, odd/even generation recovery,
  stale/caller-minted plan confirmation, two-crash retirement recovery,
  bounded pagination, continuation policy/generation binding, automatic-path
  non-mutation, exact count/byte/control bounds, and the deterministic eight-week
  crash/lease/reload/pressure composition.

The two mutants narrow one independent clause each; they do not delete the
whole gate. Their failures are assertion failures at the composed production
entry, not compilation or setup failures. This closes the narrowed-gate,
production-call-site, forged/foreign evidence, absence-vs-failure, and
non-mutation negative shapes required by the review contract.

## Reviewer validation

- `go test ./... -count=1`: exit 0 (`main` 117.107s, `internal/infra`
  197.522s; all other packages passed or had no tests).
- `CGO_ENABLED=1 go test -race ./... -count=1 -timeout 30m`: exit 0 (`main`
  168.986s, `internal/infra` 291.874s; all other packages passed or had no
  tests).
- Focused lifecycle/legacy/automatic/eight-week suite: exit 0, 47 tests.
- `go build ./...`, `go vet ./...`, changed-file `gofmt -l`, and exact-tree
  `git diff --check`: all exit 0; formatter output was empty.
- Linux/amd64 compile-only tests, Linux/arm64 build, and Windows/amd64
  compile-only tests: all exit 0.
- Isolated installed parity: static `CGO_ENABLED=0` build, `setup global`,
  `verify global`, and `doctor global` all exit 0 in a fresh `/tmp` HOME.
  Installed `README.md`, `SKILL.md`, and `LOGBOOK.md` hashes equal candidate
  bytes.

Two reviewer setup attempts are explicitly excluded from evidence: the first
clean test used an empty temporary module cache with `GOPROXY=off`, and the
first parity command used the repo root as the Go module cwd. Two further
parity harness attempts correctly refused an in-source HOME and a missing
bootstrap-owned `$HOME/.local/bin/agents-infra`. None executed a candidate test
or reached live runtime state. The corrected commands were rerun from scratch
and are the results reported above.

## Runtime boundary

No Pi executable, MLX/Qwen runtime, model/provider process, service, socket,
endpoint, or network service was contacted. Reviewer Go commands used
`GOENV=off`, `GOTELEMETRY=off`, `GOTOOLCHAIN=local`, `GOPROXY=off`, and
`GOSUMDB=off`; application configuration used a temporary HOME and nonexistent
runtime paths. Installed parity used a newly created `/tmp` HOME and did not
touch the user's installed runtime.

The accepted handoff is for the Orchestrator to run the canonical integration
gate. This reviewer supplies no `commit_ack`.
