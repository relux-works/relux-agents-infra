# BUG-260830-j1o8av: repair-lldb-mcp-bootstrap-after-homebrew-llvm-23-split

## Description
Default ./setup.sh dies before building or installing any binary on current Homebrew:

  Homebrew llvm did not provide expected helper: /opt/homebrew/opt/llvm/bin/lldb-mcp

An optional debugging integration therefore blocks the entire installation, including the agents-infra and model-harness binaries that have nothing to do with LLDB.

Investigation on 2026-09-09 settles the question the original report left open, namely whether a supported helper location exists to detect:

- Homebrew llvm 23.1.1 ships no lldb and no lldb-mcp at all; /opt/homebrew/opt/llvm/bin contains neither.
- The Apple LLDB from Xcode (lldb-2100.0.17.203) has no MCP support; its help lists no mcp command.
- The wrapper this bootstrap previously published, /opt/homebrew/bin/lldb-mcp, execs the now-absent /opt/homebrew/opt/llvm/bin/lldb-mcp, so it is broken rather than usable. install_lldb_mcp does not notice, because it treats an existing lldb-mcp as satisfied only when its path differs from the wrapper it publishes, and here they are the same path.

There is no supported source to install lldb-mcp from. Owner decision on 2026-09-09: remove the integration rather than keep repairing detection for a helper no distribution ships.

The generic MCP mechanism is untouched: a project that obtains an lldb-mcp binary by other means can still declare it in its own registry. What goes away is the bootstrap that installs it and the shared server definition that promises it.

## Scope
scripts/setup.sh bootstrap and its skip flag; the shared servers.lldb definition in .configs/codex-mcp-servers.toml; lldb references in README.md, SKILL.md, and .instructions/INSTRUCTIONS_TOOLS.md. Generic MCP wiring and unrelated test fixtures that merely use lldb as a sample server name stay as they are.

## Acceptance Criteria
setup.sh builds and installs both binaries on a machine with no lldb-mcp anywhere, with no LLDB step and no skip flag needed; no shared registry entry promises an lldb server; docs no longer claim setup installs an lldb-mcp wrapper; go vet and go test pass.
