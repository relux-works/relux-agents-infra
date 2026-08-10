# Android app-state preservation instruction

Added `Android App State Preservation` to
`.instructions/INSTRUCTIONS_TESTING.md`.

The shared policy now requires agents to preserve the installed Android app,
data, permissions, sessions, and user-visible state by default; prefer
non-destructive verification; avoid unnecessary reinstall, uninstall,
clear-data, permission revocation, and force-stop operations; reuse one
least-disruptive APK installation; and stop before an unavoidable reset that
would create unauthorized human-only recovery work.

Validation:

- `git diff --check -- .instructions/INSTRUCTIONS_TESTING.md` passed.
- `agents-infra setup global --source-dir /Users/alexis/src/relux-works/relux-agents-infra` passed.
- `agents-infra verify global` passed.
- `agents-infra setup local /Users/alexis/src/voice --source-dir /Users/alexis/src/relux-works/relux-agents-infra --codex-config preserve` passed.
- `agents-infra verify local /Users/alexis/src/voice` passed.
- The new section is present in the source module, global installed module,
  global Codex render, voice local installed module, and voice project Codex
  render.

Logs are under `.temp/instruction-state-preservation/`.
