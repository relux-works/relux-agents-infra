# TASK-260830-s5ro4e revision 3 validation

Date: 2026-08-30

Scope: reviewer findings F1 (pre-binary LLDB/Homebrew mutation) and F2
(contradictory institutional-memory entry). No release, tag, migration, or
real installed-runtime mutation was performed.

## Tool readiness

- `git --version`: exit 0, Git 2.53.0.
- `rg --version`: exit 0, ripgrep 15.2.0.
- `task-board --version`: exit 0, task-board
  `0.24.3-172-g063197b1`.
- `go version`: exit 0, Go 1.25.5 darwin/arm64.

## Production boundary measurement

- Source inspection: exit 0. `scripts/setup.sh:258-263` calls
  `install_lldb_mcp` before managed binary installation; lines 94-96 return
  immediately only when `AGENTS_INFRA_SKIP_LLDB_MCP=1`.
- Host baseline read: exit 0 for Homebrew/version and corrected identity guard.
  `llvm 23.1.0`; `/opt/homebrew/bin/lldb-mcp` is a 538-byte regular file with
  SHA-256
  `1ea8dd5ab2f944cca608d6facd14e590769879c821cfa9a4f85f7e89a9b86d2d`;
  `/opt/homebrew/opt/llvm/bin/lldb-mcp` and
  `/opt/homebrew/opt/llvm/bin/lldb` are explicitly absent.
- Direct existing-wrapper invocation: exit 1 because its configured helper is
  absent. This is a measured pre-existing anomaly, not a migration result.

## Gate attacks and real exits

1. Sandboxed production setup with fake HOME nested in the source: exit 1,
   expected refusal (`would sync into itself`). Not a pass.
2. Sandboxed production setup with a noncanonical global bin directory: exit
   1, expected refusal (`pi-infra launcher target is missing`). Not a pass.
3. Sandboxed production setup with canonical fake `$HOME/.local/bin` and
   `AGENTS_INFRA_SKIP_LLDB_MCP=1`: exit 0; setup and global verification green.
4. Exact plan-shaped gate — corrected before snapshot, real sandboxed setup
   with skip, corrected after snapshot, `cmp -s`, and real setup-exit check:
   exit 0. System LLDB/Homebrew identity was byte-identical.
5. Initial identity helper draft: shell exit 0 but output contained
   `command not found` for `stat` and `shasum`; zsh special loop variable
   `path` had erased command lookup, and the last `ABSENT` print hid the
   failures. This result is explicitly discarded as a false green. The plan
   now uses `target` and checks all three commands before reading state.
6. Negative production-entry probe without skip against a fully fake Homebrew
   root: setup exit 0 and rewrote the fake wrapper from SHA-256 `0a5fb2d3...`
   to `1de8aa42...`, retaining the original backup. This proves the mutation
   path is reachable and the skip flag closes a real path.

## Document checks

- `git diff --check`: exit 0.
- Positive corrected-logbook search for the 25/36 direct census, 20-package
  composed graph, 555 linked symbols, and mixed-window decision: exit 0.
- Search of `LOGBOOK.md` for the withdrawn 26/38 census, archive-only skip,
  and old no-window headline: exit 1, expected no-match result. It is reported
  as non-zero, not presented as a passing command.
- Search for both mandatory skip invocations, Step-3/Step-6 LLDB guard rows,
  the false-green disclosure, and the negative probe: exit 0.
- `go -C tools/agents-infra test ./internal/infra -run
  'Test(BuildChildLaunchCompositionCodexIsDeterministicMCPOnlyAndSecretSafe|BuildChildLaunchCompositionClaudeUsesOneMCPConfigArgument|BuildChildLaunchCompositionRejectsInvalidProjectConfiguration|BuildCodexLaunchPlanSupportsStdioMCPServers|BuildClaudeLaunchPlanSupportsStdioMCPServers)$'
  -count=1`: exit 0; five production composition/refusal tests green in
  0.609s.

## Artifact identities

- Plan: `.research/260830_agents-management-lockstep-release-and-rollback.md`,
  755 lines, SHA-256
  `7963d1da078f738ee7c1d16de93a97e97233ca5b95829863824b0e0cf638a872`.
- Logbook: `LOGBOOK.md`, SHA-256
  `1652e3e157d7c856a9834f206f6926e79e95ab25a7a4332ebe53e1178bea91b0`.

All operational rollback commands remain labelled `UNTESTED`; sandboxed setup
and identity probing do not rehearse a real target-home rollback.
