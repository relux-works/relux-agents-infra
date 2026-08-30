# TASK-260831-26b034 fresh-trunk replay results

## Authority and immutable preconditions

- Fresh fetch before editing: `git fetch origin main` exit 0; Story `HEAD`, `origin/main`, and the selected base were all `0d1641a0ab8fe47a98d6a54a81524a37e1cc6ead`.
- Fresh fetch immediately before handoff: exit 0; `HEAD`, `origin/main`, and `FETCH_HEAD` still all equal `0d1641a0ab8fe47a98d6a54a81524a37e1cc6ead`.
- Accepted patch: 178058 bytes, SHA-256 `2a89bafb752b5103f33329d4e77a386255c696d1f13a732397f3f1a71d651702`.
- Alternate-index reconstruction from accepted base `4270549dd17c010599e2083bf3ec7672af60ea29`: `git apply --cached` exit 0 and `git write-tree` reproduced accepted tree `16fc3dc61ba89edbc88dbf5cc236bd011ab0c151` exactly.
- The immutable pseudo-version remains `github.com/relux-works/skill-agents-management v0.5.1-0.20260830114459-046baef11790`; `go list -m -json` exit 0 and no `replace` directive is present.

## Fresh-trunk composition

- The accepted patch contains 30 repository paths; the fresh-trunk candidate contains the same 30 task paths.
- Non-mutating `git apply --3way --check` exited 0 and identified one conflict path, `LOGBOOK.md`.
- Mutating `git apply --3way` exited 1 because `LOGBOOK.md` was unmerged; the other paths applied. This expected-red result is reported as a failure, not a pass.
- `LOGBOOK.md` was the only conflict. The resolution is a lossless union: set-difference checks over nonblank lines found 0 missing trunk lines, 0 missing accepted-revision lines, and 0 conflict markers (exit 0). Two current-task logbook entries were then added for this replay and the mutation-count anomaly.
- Blob comparison against accepted tree `16fc3dc6...` found 28/30 reviewed paths byte-identical. The only widened paths are:
  - `LOGBOOK.md`: lossless union of current trunk and accepted revision 3, plus this task's two required logbook entries.
  - `README.md`: clean three-way composition preserving current-trunk model-harness/benchmark/article documentation while replacing the overlapping Pi tool/alias lines with the accepted schema-1 Process-A contract.
- An intentionally strict README nonblank-line superset oracle exited 1 (3 trunk long lines and 1 accepted long line were replaced by composed long lines). Bounded inspection showed the replacements retain both semantic sides; this was not reported as a green line-union check.
- Fresh-trunk candidate tree before CR publication: `26b2b9ca950254e609aa5e9f2307786c95eb0196`.
- Generic adapter architecture, Process-B ownership, and every production Go blob are unchanged from accepted revision 3 except for clean composition with current trunk; no stable agents-management tag was introduced.

## Validation

| Gate | Exit | Result |
| --- | ---: | --- |
| `gofmt -l tools/agents-infra` with zero-output assertion | 0 | Pass, 0 output bytes |
| `git diff --cached --check` | 0 | Pass |
| `env -u TASK_BOARD_DIR go test -mod=mod ./... -count=1` | 0 | Pass; root, attachments, infra, and modelharness packages |
| Focused `go test ./internal/infra -race -count=1 -run ...` | 0 | Pass |
| `go vet ./...` | 0 | Pass |
| Native `go build ./...` | 0 | Pass |
| `GOOS=linux GOARCH=amd64 go build ./...` | 0 | Pass |
| `GOOS=windows GOARCH=amd64 go build ./...` | 0 | Pass |
| `GOOS=darwin GOARCH=amd64 go build ./...` | 0 | Pass |
| Focused no-live/runtime-boundary AST test set | 0 | Pass |
| Accepted rev3 mutation harness | 0 | All 17 executable `run_mutant` attacks killed with real exit 1; discovery-narrowing control admitted with real exit 0; no survivor/unexpected record |
| Mutation restore assertion | 0 | Every Go file byte-identical before/after; no unstaged product diff |

The mutation source and its published rev3 log both contain 17 killed attacks plus one admitted discovery control. The producer/reviewer prose says 18 killed plus one control. This is an immutable-precondition evidence-count anomaly, recorded in `LOGBOOK.md`; no unreviewed mutant was invented. An initial post-run wrapper printed `killed=17` but accidentally returned exit 0 because its final independent `test` passed. That wrapper is not counted as a gate. The corrected aggregate assertion explicitly fails on any count/restore/survivor mismatch and exited 0 for the executable 17 + 1 contract.

## Negative evidence and production call sites

- `parsePiTurnJSONL` in `tools/agents-infra/internal/infra/pi_turn_result.go` is driven by `TestPiTurnTranslatorRefusesTurnAndMessageLifecycleViolations`; narrowing the turn/message lifecycle guards makes each named negative test fail.
- `BuildPiPluginGraph` in `agents_management_registry.go` and `BuildAndRunPiTurn` in `agents_management_process_a.go` are driven with the real broker-backed `SharedRuntimeSanitizedEngineObservationReader` by `TestSharedRuntimeEngineObservationReaderReadsRealBrokerAndProcessASpawnsNeverTouchProcessB`.
- The sole classifier call and no-live/generic-boundary constraints are driven through production-reachable AST guards. The exact mutation harness kills second-classifier, live-read helper, identity-dispatch, Process-B ownership, exit-laundering, stale-observation, identity-drift, profile, tool-failure, and lifecycle narrowings.

## Runtime boundary

No live model, user configuration, HOME state, production socket, endpoint, external service, or stable-tag publication was used. The full and focused test suites use only hermetic test fixtures and test-local fake-backed runtime/broker processes.

## Evidence logs

- `.temp/TASK-260831-26b034/go-test-full-01.log`
- `.temp/TASK-260831-26b034/go-test-race-focused-01.log`
- `.temp/TASK-260831-26b034/go-test-no-live-boundary-01.log`
- `.temp/TASK-260831-26b034/mutants.log`
- `.temp/TASK-260831-26b034/mutation-run-01.stdout.log`
- `.temp/TASK-260831-26b034/mutation-restore-diff-02.log`
- `.temp/TASK-260831-26b034/go-module-pin-01.log`

Signed commit, PR publication, independent acceptance, and integration are the owning reviewer/orchestrator stages after this developer Change Request handoff.
