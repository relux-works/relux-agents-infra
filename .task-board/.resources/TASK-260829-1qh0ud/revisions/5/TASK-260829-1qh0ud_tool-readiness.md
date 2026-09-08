# Tool readiness

Checked 2026-08-29 in the Story worktree.

- `git --version`: exit 0, `git version 2.53.0`
- `go version`: exit 0, `go version go1.25.5 darwin/arm64`
- `task-board --version`: exit 0, `task-board version 0.24.3-167-g42c7d2b7 (commit 42c7d2b7, built 2026-08-29T12:17:19Z)`
- `rg --version`: exit 0, `ripgrep 15.2.0 (rev e89fff89ac)`
- shell `apply_patch --help`: exit 127, `command not found`; expected because patching is exposed as an agent tool, not a shell binary. Project edits use the available agent `apply_patch` tool.
