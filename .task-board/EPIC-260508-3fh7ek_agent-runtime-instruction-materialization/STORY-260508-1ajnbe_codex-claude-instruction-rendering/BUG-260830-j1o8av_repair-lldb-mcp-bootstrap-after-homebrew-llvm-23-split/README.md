# BUG-260830-j1o8av: repair-lldb-mcp-bootstrap-after-homebrew-llvm-23-split

## Description
Default ./setup.sh fails on current Homebrew after LLVM 23 no longer provides /opt/homebrew/opt/llvm/bin/lldb-mcp. Detect the supported LLDB package/helper location or produce a typed actionable refusal without breaking unrelated global setup. Preserve AGENTS_INFRA_SKIP_LLDB_MCP=1 as the explicit bypass.

## Scope
relux-agents-infra setup/bootstrap source, Homebrew package detection, generated wrapper verification, docs and fake/static tests. Do not require a live LLDB target.

## Acceptance Criteria
On current Homebrew macOS the default setup either installs and verifies a real supported lldb-mcp helper from the correct package or fails before partial wrapper publication with a precise remediation; LLVM 22 compatibility and the skip flag remain covered; no unrelated global runtime files drift.
