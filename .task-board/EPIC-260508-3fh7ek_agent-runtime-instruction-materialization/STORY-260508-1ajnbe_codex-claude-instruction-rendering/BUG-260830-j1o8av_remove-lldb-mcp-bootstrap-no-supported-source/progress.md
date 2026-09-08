## Status
done

## Review
required

## Task Class
code

## Estimate
notEstimated

## Blocked By
- (none)

## Blocks
- (none)

## Checklist
- [x] install_lldb_mcp and its call site are removed from scripts/setup.sh, along with the wrapper marker and the AGENTS_INFRA_SKIP_LLDB_MCP usage line
- [x] The shared servers.lldb definition is removed from .configs/codex-mcp-servers.toml
- [x] README.md, SKILL.md, and .instructions/INSTRUCTIONS_TOOLS.md no longer claim setup installs Homebrew llvm or an lldb-mcp wrapper, and no longer list lldb as a shared opt-in server
- [x] Regression witness: a full ./setup.sh run with no lldb-mcp present and no skip flag installs both binaries and reports Done
- [x] Negative check: the removed skip flag is gone from usage output, and setup does not consult brew for LLDB at all
- [x] Generic MCP wiring still works: a project declaring its own server in its local registry composes unchanged
- [x] go vet ./... and go test ./... -count=1 pass in tools/agents-infra

## Notes

## Precondition Resources
(none)

## Outcome Resources
(none)

## Created
2026-08-30T00:25:53Z

## Last Update
2026-09-09T15:26:22Z
