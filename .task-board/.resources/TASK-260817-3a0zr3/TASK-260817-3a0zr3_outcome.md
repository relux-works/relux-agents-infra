# TASK-260817-3a0zr3 developer outcome

## Delivered scope

- Global and local setup install an exact sibling-only `pi-infra` launcher that delegates to `agents-infra pi`, preserves caller cwd and argv order/bytes, and refuses a missing sibling target.
- Setup postconditions and `verify` reject missing/drifted alias bytes or mode, a missing/non-regular/non-executable target, an incorrect global target identity, and missing/drifted authoritative Pi manifest bytes.
- README documents the exact cycle-10 Qwen/Muse TOML, precedence and argv bridge, non-launching diagnostics, exact-UTF-8 SHA-256 state keys, anchored containment, 217-record catalog, lifecycle, requested/unverified capabilities, operator smoke/telemetry checks, error codes, exclusions, and practical trust-boundary non-claims.
- `SKILL.md` routes Pi changes to this source repository and requires `--print-config` plus setup/verify before safe launch.
- Production-entry tests cover global setup delegation, caller cwd, post-separator argv, missing target, alias drift, incorrect target identity, setup repair, catalog drift, and documentation contract presence.
- Important setup/catalog integrity finding recorded in `LOGBOOK.md` at `2026-08-17 1829`.

## Validation evidence

Every command below ran directly; exit codes are the real process results.

| Command | Exit | Evidence |
| --- | ---: | --- |
| `go test . -run 'TestInstalledBinarySetupLocal|TestInstalledBinaryVerifyLocal|TestInstalledBinarySetupGlobalPiInfra' -count=1` | 0 | `.temp/TASK-260817-3a0zr3/go-test-installed-03.log` |
| `go test . -run 'TestPiOperatorContractDocumentsCycle10Boundary|TestReluxAgentsInfraSkillRoutesSafePiWorkflowToSource' -count=1` | 0 | `.temp/TASK-260817-3a0zr3/go-test-docs-02.log` |
| Focused alias/catalog/docs `go test -race ./... -count=1` | 0 | `.temp/TASK-260817-3a0zr3/go-test-race-focused-01.log` |
| `go test ./... -count=1` | 0 | `.temp/TASK-260817-3a0zr3/go-test-full-05.log` |
| `go vet ./...` | 0 | `.temp/TASK-260817-3a0zr3/go-vet-02.log` |
| `go build ./...` | 0 | `.temp/TASK-260817-3a0zr3/go-build-02.log` |
| `gofmt -l .` plus empty-output assertion | 0 / 0 | `.temp/TASK-260817-3a0zr3/gofmt-list-02.log` |
| `./setup.sh` | 0 | `.temp/TASK-260817-3a0zr3/setup-global-02.log` |
| `agents-infra verify global` | 0 | `.temp/TASK-260817-3a0zr3/verify-global-02.log` |
| `agents-infra setup local /Users/alexis/src/relux-works/relux-agents-infra` | 0 | `.temp/TASK-260817-3a0zr3/setup-local-02.log` |
| `agents-infra verify local /Users/alexis/src/relux-works/relux-agents-infra` | 0 | `.temp/TASK-260817-3a0zr3/verify-local-02.log` |
| `git diff --check` | 0 | `.temp/TASK-260817-3a0zr3/git-diff-check-02.log` |
| `task-board validate` after evidence/checklist updates | 0 | `.temp/TASK-260817-3a0zr3/task-board-validate-02.log` |

Expected development failures retained rather than reported green:

- `go-test-full-01.log`: exit 1; test fixture resolved the authoritative manifest from repo root instead of the Go module path.
- `go-test-installed-02.log`: exit 1; the pre-existing negative asserted the entire `internal/` directory was absent, while the new source contract correctly retains the manifest there. The negative now asserts the imported Go package is absent.
- `go-test-docs-01.log`: exit 1; documentation assertion failed on Markdown line wrapping. It now normalizes whitespace while retaining exact contract tokens.

## Installed state

- Global: `/Users/alexis/.local/bin/pi-infra`
- Local: `/Users/alexis/src/relux-works/relux-agents-infra/.local/bin/pi-infra`
- Both aliases were installed from the same managed body and verified successfully.
- Authoritative source manifest SHA-256: `2f68ab1b3f28a9c4b8995f91984f8f47001a79735da7e57aa7fe6d223f90378b`.

## Scope notes

- No model/runtime acquisition, conversion, benchmark automation, secure distribution, backend catalog, compiled observer, proxy adapter/private-pipe authority, or runtime/DFlash attestation was added.
- Qwen and Muse capabilities remain requested/configured and unverified.
- No files were staged or committed.
