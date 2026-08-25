# TASK-260826-h934tg — Revision 4 Evidence

## Scope

- Review blocker addressed: rev3 F1 only.
- Rev3 candidate tree: `b59f8c9418476b62799c9653f5d23ee7b535329e`.
- Revision 4 candidate tree: `8c8096878aa5a7b9bb2fefa129d2407dd620e009`.
- Rev3-to-rev4 paths: `LOGBOOK.md` and `tools/agents-infra/internal/infra/pi_standalone_shared_test.go` only.
- Production, README, skill documentation, and every other test are byte-identical to rev3.
- Preserved production hashes:
  - `pi_launch_posix.go`: `84a099004458566137bb9fe4b33f2944533b910c86aef7b47bcd85bab1df1e54`
  - `pi_shared_client_darwin.go`: `b5e1adacec06c1d31a62b4c1e7557cc6af1bc66970fc797df37073fb67745391`

## Shared-path witness

`TestRunPiStandaloneConcurrentWorkersShareOnlyRuntimeAndCrashCleanupPreservesPeer`
drives the production entry `RunPi -> runSharedPiSession` with readable stdin and
a forced-positive `piTerminalFDProbe`. It requires the probe to be called zero
times while both shared standalone workers observe stdin EOF.

The reviewer M4 narrowing was reproduced by calling
`configurePiProcessTerminal(piCmd, opts.Stdin)` only in the shared standalone
branch while stdin remained closed:

| Gate | Exit | Result |
| --- | ---: | --- |
| Clean focused shared witness | 0 | `ok`, 2.580s |
| M4 focused shared witness, `-count=1` | 1 | Expected red: worker launch refused with `operation not supported by device` |
| Restored focused shared witness | 0 | `ok`, 1.993s |
| Focused shared + exclusive + primary-yolo witnesses | 0 | `ok`, 3.981s |

Production was restored from a pre-mutant copy. `cmp` returned exit 0 and the
post-restoration shared production hash is the preserved hash above.

## Landing validation

Every command was run directly on the restored revision-4 tree.

| Command | Exit | Result |
| --- | ---: | --- |
| `go test ./... -count=1` | 0 | root 85.771s; attachments 0.962s; infra 133.631s |
| `go vet ./...` | 0 | Darwin clean |
| `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go vet ./...` | 0 | clean |
| `CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go vet ./...` | 0 | clean |
| `GOOS=darwin GOARCH=arm64 go build ./...` | 0 | clean |
| `GOOS=darwin GOARCH=amd64 go build ./...` | 0 | clean |
| `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ./...` | 0 | clean |
| `CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build ./...` | 0 | clean |
| `CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ./...` | 0 | clean |
| `gofmt -l .` | 0 | empty output |
| `git diff --check` | 0 | clean |

## Ancestry and evidence corrections

- HEAD: merge leaf `eaefc6f4f41a9e2c2c0f0464a8fb632ab13910ff`.
- Current main: `355a156276080b6994080f8e9a767e7416a5b357`.
- `git merge-base --is-ancestor main HEAD`: exit 0.
- `git rev-list --count HEAD..main`: `0`.
- LOGBOOK entries 1528 and 1411 now name both discriminating production-launch
  witnesses: the exclusive worker test and shared concurrency test.

Revision 4 changes no terminal behavior, standalone behavior, authorization,
allowlist, runtime lifecycle, or other production behavior.
