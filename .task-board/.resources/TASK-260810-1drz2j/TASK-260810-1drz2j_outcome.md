# Android app-state preservation instruction — final revision

The shared Android policy now explicitly requires every spawned developer, tester, or device worker to inherit the app-state preservation contract. It rejects vague Android test delegation that could choose a package-destructive lane, and requires physical-device work to prefer direct instrumentation, an already-installed app, or a test-APK-only update when sufficient. Reinstall, uninstall, clear-data/storage, permission revocation, force-stop, or Gradle-managed replacement lanes require demonstrated necessity and advance accounting for user-restored state.

Validation after the final follow-up revision:
- agents-infra setup global from /Users/alexis/src/relux-works/relux-agents-infra passed; agents-infra verify global passed.
- agents-infra setup local for /Users/alexis/src/voice with codex-config preserve passed; agents-infra verify local passed.
- Source, global installed, and voice project-local installed INSTRUCTIONS_TESTING.md are byte-identical with SHA-256 9f7a9d8702f017154457e5664bbfad8bb5a0ed74de727e889cdf953e7ac65e6a.
- All four preservation clauses are directly present in /Users/alexis/src/voice/AGENTS.md and /Users/alexis/src/voice/.codex/AGENTS.md.
- Claude entrypoint /Users/alexis/src/voice/.claude/CLAUDE.md includes @instructions/INSTRUCTIONS.md; its instructions symlink resolves to /Users/alexis/src/voice/.agents/.instructions, where all four clauses are present.
- Direct parity checks and git diff checks passed; verify-local was not used as the sole freshness attestation.

Existing unrelated dirty files were preserved.