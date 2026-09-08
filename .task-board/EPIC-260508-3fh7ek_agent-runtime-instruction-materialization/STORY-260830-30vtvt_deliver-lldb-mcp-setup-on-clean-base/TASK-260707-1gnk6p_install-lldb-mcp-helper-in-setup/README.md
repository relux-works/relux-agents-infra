# TASK-260707-1gnk6p: install-lldb-mcp-helper-in-setup

## Description
Install the LLDB MCP helper prerequisites from the agents-infra setup script on macOS so project-local lldb MCP opt-in works after render/restart without manual Homebrew or wrapper steps.

## Scope
(define task scope)

## Acceptance Criteria
The agents-infra setup script installs the LLDB MCP helper prerequisites on macOS so a project-local lldb MCP opt-in works after render and restart, with the install documented and idempotent, and with a clear failure when the prerequisites cannot be installed rather than a silently broken opt-in.
