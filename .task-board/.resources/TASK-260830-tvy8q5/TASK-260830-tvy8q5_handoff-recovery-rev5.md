# TASK-260830-tvy8q5 revision 5 developer evidence

## Change

- Closed the reviewer-proven health bypass in `PiLifecycleStatus`: a fresh complete scan now requires zero legacy and foreign evidence before publishing `within_policy`; `soak_ready` inherits that same gate.
- Strengthened `TestRunPiLifecycleOperatorIsNonLaunchingAndProjectsExactPlan` through the production `runPi -> runPiLifecycleCLI` entry point. It requires pre-retirement `within_policy=false` and `soak_ready=false`, then confirms both only after exact-plan retirement.
- Recorded the root cause, fix, and production-entry evidence in `LOGBOOK.md`.

Exact Story candidate base: `40d4e4f8c17387f04fa99cb26585f396e8f07cc0`.

- `pi_session_log.go`: `d4c2644694644e355293cbed0e20afbd03d269b9060d72de176c279ea8beada1`
- `pi_lifecycle_main_test.go`: `b6e94118165fd1948e97e5566d1d734ce5839d856e08bf5a7a23232f9937677e`
- `LOGBOOK.md`: `1342c54726e32108123fb580690cd5095cb1f8568eb836800ef062126936bc7c`

## Direct validation

All Go commands used task-scoped `HOME`, `GOCACHE`, `GOMODCACHE`, and `GOPATH`, disabled sumdb/telemetry, and a repository-local read-only module proxy. No network proxy was used.

| Gate | Exit | Evidence |
| --- | ---: | --- |
| Test-first production-entry check before the production fix | 1 expected-red | Reported `LegacyCount=1`, `WithinPolicy=true`, `SoakReady=false`. |
| Exact production-entry test after the fix | 0 | Pre-retirement health refused; exact confirmed retirement published health. |
| Narrowed mutant retaining foreign refusal but removing only the legacy-count condition | 1 expected-red | Reported `LegacyCount=1`, `WithinPolicy=true`, `SoakReady=true`; the exact legacy gate is covered. |
| Exact production-entry test after restoring production | 0 | `go test . -run '^TestRunPiLifecycleOperatorIsNonLaunchingAndProjectsExactPlan$' -count=1`. |
| Legacy rev2-rev4 crash/resume, automatic non-mutation, and deterministic eight-week soak suite | 0 | Focused `internal/infra` suite passed in `5.698s`. |
| Lifecycle CLI pagination/retirement and operator docs suite | 0 | Focused root suite passed in `0.636s`. |
| `go build ./...` | 0 | Native module build passed. |
| `gofmt -l` over every changed Go file | 0 | Empty output. |
| `git diff --check` | 0 | Empty output. |

The bounded recovery did not repeat the already attached full repository, race, vet, Linux, or Windows gates. Revision 4 evidence records those gates against the preserved candidate; revision 5 changes one platform-neutral health expression, one production-entry assertion, and `LOGBOOK.md`. This is explicit reliance on prior attached evidence, not a claim that those commands were rerun in this recovery.

## No-live-runtime boundary

No Pi executable, installed runtime, model, provider process, service, socket, endpoint, setup/install flow, network service, or live user-HOME runtime/config state was contacted. Tests used deterministic temporary filesystem fixtures and repository-local caches. Installed skill files were read only as assigned instructions.
