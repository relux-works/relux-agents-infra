# TASK-260707-1gnk6p clean-base reimplementation

## Scope

- Base and current `HEAD`: `436760d62f4ea451cf49614ff7e40109d96915b3`, equal to `origin/main` when inspected.
- Repository delta contains only LLDB MCP setup scope: `scripts/setup.sh`, `tools/agents-infra/setup_lldb_mcp_test.go`, `README.md`, `SKILL.md`, and the task findings in `LOGBOOK.md`.
- No files or patches were read from the legacy Story worktree.

## Implementation

- `scripts/setup.sh` now fails non-zero when macOS has neither a usable external `lldb-mcp` nor working Homebrew installation prerequisites.
- Homebrew `llvm` is installed only when its `lldb-mcp` helper or adjacent `lldb` is missing.
- One renderer owns the managed wrapper bytes. A byte-identical executable wrapper is preserved without inode replacement; divergent or forged managed bytes are atomically replaced.
- The real generated wrapper is executed in tests to prove argument delegation to the Homebrew helper.
- `README.md` documents `./setup.sh`, `brew install llvm`, the managed wrapper path, explicit skip command, local render, config inspection, and managed Codex restart commands. `SKILL.md` mirrors the operator contract.

## Negative evidence

Pre-fix production-entry command:

```text
go test -count=1 -run '^TestBootstrapSetup' .
exit 1
```

The red run proved all three review findings: missing Homebrew reached `=== Done ===`; the second setup invoked `brew install llvm` again; wrapper repair also reinstalled LLVM. The forged wrapper fixture placed `exit 0` before the expected `exec` line.

Post-fix production-entry command:

```text
go test -count=1 -v -run '^TestBootstrapSetup' .
exit 0
```

Passing real-entry tests:

- `TestBootstrapSetupRefusesMissingLLDBMCPPrerequisites`
- `TestBootstrapSetupInstallsLLDBMCPOnceAndWrapperDelegates`
- `TestBootstrapSetupRepairsManagedLLDBMCPWrapperWithUnreachableExpectedTarget`
- `TestBootstrapSetupRefusesLLVMInstallWithoutExpectedHelper`

## Validation

| Command | Exit | Result |
| --- | ---: | --- |
| `go test -count=1 ./...` | 0 | All Go packages green; root 131.201s, infra 195.190s |
| `go vet ./...` | 0 | Clean |
| `go build ./...` | 0 | Build succeeds |
| `zsh -n setup.sh scripts/setup.sh` | 0 | Shell syntax clean |
| `test -z "$(gofmt -l setup_lldb_mcp_test.go)"` | 0 | Go formatting clean |
| `git diff --check` | 0 | No whitespace errors |

