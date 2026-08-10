# Review verdict: accepted

## Scope reviewed

- Shared source: `.instructions/INSTRUCTIONS_TESTING.md`.
- Installed global runtime: `/Users/alexis/.agents/.instructions/INSTRUCTIONS_TESTING.md` and the rendered global Codex entrypoint.
- Installed project runtime: `/Users/alexis/src/voice/.agents/.instructions/INSTRUCTIONS_TESTING.md`, rendered project Codex entrypoints, and the Claude include/symlink path.
- Producer outcome resources and focused setup/verification evidence.

## Acceptance evidence

- The policy preserves the installed package, app data, permissions, sessions, and user-visible state by default; requires non-destructive verification first; refuses reinstall, uninstall, clear-data/storage, permission revocation, and force-stop without demonstrated necessity for the exact validation; and accounts for exact package scope plus human-only re-interaction before an unavoidable reset.
- Developer/tester/device-worker delegation explicitly carries the preservation contract. Physical-device work explicitly avoids Gradle-managed replacement lanes when direct instrumentation, an already-installed app, or a test-APK-only update is sufficient.
- Source, global runtime, and `/Users/alexis/src/voice` project runtime modules are byte-identical at SHA-256 `9f7a9d8702f017154457e5664bbfad8bb5a0ed74de727e889cdf953e7ac65e6a`.
- `/Users/alexis/.codex/AGENTS.md`, `/Users/alexis/src/voice/AGENTS.md`, and `/Users/alexis/src/voice/.codex/AGENTS.md` each contain all four probed preservation clauses. Global and project Claude entrypoints resolve through their installed `.agents/.instructions` trees; the project entrypoint contains `@instructions/INSTRUCTIONS.md`.
- `agents-infra verify global`, `agents-infra verify local /Users/alexis/src/voice`, and `git diff --check` passed.
- `go test ./internal/infra -count=1` passed from `tools/agents-infra` in 46.156s.

## Gate attack

Negative-evidence shape attacked: **bypass path around the check**. The prior cycle proved `verify local` alone could pass with a stale project runtime. This cycle used direct source-to-installed byte parity and asserted the clauses on every applicable composed delivery surface. The same four-clause probe returns nonzero against the known-stale, non-installed active-repository `AGENTS.md`, proving the probe does not default-pass when evidence is absent. The active repository `.agents` tree is not an installed runtime (`verify local` refuses it because no completed-install receipt exists), so it is not an acceptance surface.

## Verdict

Accepted. The implementation matches the acceptance criteria, fits the shared instruction-module/rendering architecture, and closes the previously demonstrated freshness bypass.
