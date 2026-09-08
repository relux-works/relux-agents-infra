# TASK-260707-1gnk6p: install-lldb-mcp-helper-in-setup

## Description
Obsoleted on 2026-09-09 by BUG-260830-j1o8av, landed in #39. There is no supported source to install lldb-mcp from: Homebrew llvm 23 ships neither lldb nor lldb-mcp, and Apple Xcode LLDB has no MCP support. The bootstrap this task would have delivered was removed instead of completed. Closed rather than done: no helper was installed.

## Scope
(define task scope)

## Acceptance Criteria
The agents-infra setup script installs the LLDB MCP helper prerequisites on macOS so a project-local lldb MCP opt-in works after render and restart, with the install documented and idempotent, and with a clear failure when the prerequisites cannot be installed rather than a silently broken opt-in.
