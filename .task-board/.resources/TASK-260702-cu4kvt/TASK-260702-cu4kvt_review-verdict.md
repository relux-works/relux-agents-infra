# TASK-260702-cu4kvt Review Verdict

Verdict: accepted.

## Scope reviewed

- Source implementation introduced by commit `d1c8d7d5649c37df394d3401101a9650491b4893`.
- Review worktree HEAD: `cf21665dde35274cc14e66e26a93574e0c18c15c`.
- The task-owned instruction files are clean. The worktree also contains an unrelated eight-path Codex-config Change Request; those changes were not attributed to or modified by this review.

## Acceptance evidence

- `.instructions/INSTRUCTIONS_BROWSER_AUTOMATION.md` states no-focus-by-default behavior, names `mac-safari-session open-bg`, `snapshot`, and `run-js`, forbids Safari activation/frontmost-window/UI-click patterns, identifies passkey/SSO/CAPTCHA/file-picker/permission cases for human takeover, and preserves same-origin plus browser-secret handling rules.
- `.instructions/AGENTS.md` and `.instructions/INSTRUCTIONS.md` each include the module exactly once.
- `README.md` lists the module as the no-focus and authenticated-session policy.
- Installed source and entrypoint bytes match the reviewed source. Claude reaches the installed entrypoint through `~/.claude/CLAUDE.md -> @instructions/INSTRUCTIONS.md` and the `~/.claude/instructions` link. The composed `~/.codex/AGENTS.md` contains the required clauses.
- `agents-infra verify global` passed for `/Users/alexis/.agents`.

## Negative evidence

- Shape attacked: `bypass path around the check`. Both source entrypoints, the installed Claude include-chain, and the composed Codex artifact were checked independently; no runtime-specific omission was found.
- Shape attacked: `absent evidence treated as satisfied`. `TestInstalledBinarySetupLocalRefusesSourceWithUnshippedInstructionInclude` drove the installed-binary setup entrypoint with a missing included module and passed with `-count=1`, proving fail-closed behavior rather than relying on a positive setup path.
- This task adds policy delivery, not an authorization or attestation API; no additional gate exists to forge or self-mint.

## Validation

- `go test -count=1 ./internal/infra` — PASS (`343.969s`).
- `go test -count=1 ./internal/attachments` — PASS (`5.174s`).
- `go test -count=1 .` — PASS (`194.303s`).
- `go test -count=1 -run '^TestInstalledBinarySetupLocalRefusesSourceWithUnshippedInstructionInclude$' .` — PASS (`2.699s`).
- `go vet ./...` — PASS.
- `git diff --check main...HEAD` — PASS.

No code changes were made by the reviewer.
