# TASK-260817-3a0zr3 cycle-2 rework evidence

## Scope

- Reject byte-identical symlink replacements for the managed `pi-infra` alias.
- Reject symlink replacements for the exact sibling `agents-infra` target.
- Repair alias type and POSIX mode drift through production `setup local`.
- Preserve the existing caller cwd/argv alias contract.
- Document the regular-file/no-symlink artifact policy in README and the source-owned skill.

## Implementation

- `tools/agents-infra/internal/infra/infra.go`: the alias up-to-date branch now requires `os.Lstat` regular-file identity, exact body, and POSIX mode `0755`.
- `tools/agents-infra/internal/infra/runtime_receipt.go`: verification uses `os.Lstat` for alias and sibling target before reading or executing either path.
- `tools/agents-infra/installed_binary_setup_test.go`: production setup/verify case repairs `0644` and byte-identical symlink alias drift, then refuses a byte-identical symlink sibling target.
- `tools/agents-infra/internal/infra/runtime_receipt_test.go`: negative type-drift coverage for both managed paths.
- `README.md`, `SKILL.md`, `LOGBOOK.md`: exact artifact policy and regression record.

## Validation

| Command | Exit | Result |
| --- | ---: | --- |
| `go test ./internal/infra -run 'TestVerifyInstalledRuntimeRefusesMissingAndDriftedPiInfraAlias|TestSetupRepairsPiInfraAliasDrift' -count=1` | 0 | Focused verifier/setup regressions pass. |
| `go test . -run TestInstalledBinarySetupLocalPiInfraRepairsModeAndSymlinkDrift -count=1` | 0 | Production CLI setup/verify regression passes. |
| `go test ./... -count=1` | 0 | Full Go suite passes; main `28.018s`, attachments `1.805s`, infra `77.265s`. |
| `go vet ./...` | 0 | No findings. |
| `go build ./...` | 0 | All Go packages build. |
| `./setup.sh` | 0 | Source build installed globally; bootstrap reported verified global setup. |
| `agents-infra verify global` | 0 | `/Users/alexis/.agents` verified. |
| `agents-infra setup local /Users/alexis/src/relux-works/relux-agents-infra` | 0 | Local runtime refreshed from source. |
| `agents-infra verify local /Users/alexis/src/relux-works/relux-agents-infra` | 0 | Project `.agents` runtime verified. |
| `git diff --check` | 0 | Diff hygiene passes. |
| `task-board validate` | 0 | Board valid; no issues. |

Tool readiness was established with `command -v`: `rg`, Go, and `task-board` resolved successfully before project work.

## Negative evidence

The named production test replaces the managed alias with a symlink to an external byte-identical copy. `verify local` must refuse it, and `setup local` must replace it with a regular `0755` artifact. It separately replaces the sibling target with a byte-identical symlink and requires refusal. A content-only or follow-link narrowing therefore makes the production test fail.
