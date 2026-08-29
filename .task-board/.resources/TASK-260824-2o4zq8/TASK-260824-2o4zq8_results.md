# TASK-260824-2o4zq8 implementation evidence

## Handoff

- Commit: `0246afc` (`Resolve canonical vendor targets`)
- Story branch: `task-board/story/STORY-260824-1yr6m0`
- Base condition: the assigned Story worktree was four commits behind `main`; it was not rebased or merged, per the Story workspace contract.
- Preserved unrelated state: the pre-existing staged/unstaged `LOGBOOK.md` index state was not reset or staged. The required 1941 finding was added append-only to the working copy and deliberately left outside the code commit so the foreign staged snapshot remains untouched.

## Implemented production contract

- Strict parsing for atomic `[agents.targets.<name>]` definitions and exact `[agents.entrypoints]` mappings, composed root-to-cwd without cross-file target-field merging.
- Admitted tuple and reasoning-domain validation for OpenAI/Codex, Anthropic/Claude Code, and Qwen/Pi.
- Qwen resolution against an existing complete managed Pi profile, including exact API/model/thinking/provider/endpoint assertions and profile-derived effective provider/model/endpoint provenance.
- `agents-infra target <entrypoint>` dispatch and primary-session `compose --entrypoint`, with schema-v1 target provenance and safe actionable error envelopes.
- Identity locks for Codex flags/config selectors, Claude model/effort selectors, and decoded Pi model/provider/thinking/profile/endpoint coordinates.
- Sibling-only `openai-infra`, `anthropic-infra`, and `qwen-infra` install, repair, setup-postcondition, verification, cwd/argv preservation, and sibling refusal behavior.
- Canonical target reporting in `doctor`; human print output includes configured assertions and effective values/sources without secrets.
- Direct legacy Codex, Claude, Pi, and `pi-infra` behavior remains separate and retains existing precedence and legacy JSON shape.
- README and relux-agents-infra skill updated; obsolete unsupported `triggers` skill frontmatter removed because current validation accepts discovery metadata only through supported fields such as `description`.
- Flight Logbook 1941 records the worktree-relative Pi asset skip root cause and the common-checkout fallback.

## Production entrypoints bound by negative evidence

- `parseProjectConfig`: every cross-vendor/environment pair; required/empty/wrong-type/unknown fields; hosted profile coordinates; closed Claude/Pi reasoning and open non-empty Codex reasoning.
- `ResolveCanonicalTarget` / `BuildCanonicalTargetLaunchPlan`: missing mapping without inference, unknown target, alias/vendor mismatch, Pi profile/API/assertion divergence, Codex selector forms, Pi composite suffix/provider divergence, and legacy precedence isolation.
- `runCompose`: selector exclusivity, target provenance, Qwen profile-derived invariants, actionable missing/unknown/malformed/unreadable/conflict errors, byte-identical project configs, and no Pi runtime state.
- `runTarget`: provider launch cwd/argv, identity conflict before provider execution, and no legacy-table bypass for an unconfigured alias.
- Generated alias executable: exact sibling dispatch plus missing/non-regular/non-executable sibling refusal.
- `Setup` / `VerifyInstalledRuntime`: invalid config before mutation; all three aliases installed, independently narrowed, refused, repaired, and re-verified.

## Validation evidence

Every gate below ran directly as a standalone process; exit codes are the real process statuses.

| Command | Exit | Evidence |
| --- | ---: | --- |
| `go test ./internal/infra . -run 'Canonical\|Target\|Entrypoint' -count=1` | 0 | Focused parser/resolver/compose/dispatch/setup alias scope passed. |
| `go test . -run '^TestRunComposeCanonicalQwenUsesProfileDerivedEffectiveCoordinates$' -count=1 -v` | 0 | Production Qwen compose test reported `PASS`, not `SKIP`. |
| `go test ./internal/infra -run '^TestCanonicalQwen' -count=1 -v` | 0 | All Qwen invariants, negative assertions, composite identity locks, and direct-Pi compatibility tests passed with the official Pi asset. |
| `go test ./... -count=1` | 0 | Final asset-enabled full suite passed: main `102.298s`, attachments `1.985s`, infra `149.681s`. |
| `go vet ./...` | 0 | Final Go vet clean. |
| `go build ./...` | 0 | Final module build clean. |
| skill `quick_validate.py .` through task-scoped Python environment | 0 | `Skill is valid!` |
| `agents-infra setup global ...` in isolated `/tmp/TASK-260824-2o4zq8-smoke-cHiJtf` | 0 | Installed all three canonical aliases beside the exact global sibling. |
| `agents-infra verify global ...` | 0 | Global runtime verified. |
| `agents-infra setup local ...` | 0 | Installed all three aliases and generated local sibling launcher. |
| installed `agents-infra verify local ...` before and after valid canonical config | 0 / 0 | Local runtime verified on legacy-only and configured-target states. |
| installed `openai-infra --print-config` with a complete legacy Codex table but no mapping | 1 | Expected refusal: `unknown_entrypoint`, exact field, remediation; no legacy fallback. |
| installed configured `openai-infra --print-config -- --model gpt-5.6-sol` | 0 | Printed entrypoint/target/effective provenance without launching. |
| `git diff --check` | 0 | Remaining uncommitted worktree diff has no whitespace errors. |
| `git diff --cached --check` | 0 | Preserved pre-existing staged state has no whitespace errors. |

The first attempted global smoke deliberately placed the destination inside the source tree and exited 1 with the expected self-containment refusal. It was not counted as a pass; the positive global smoke was rerun in the isolated `/tmp` root and exited 0.

Tool readiness initially found no `python` (exit 127), system `python3` lacked PyYAML (exit 1), and `uv` was unavailable (exit 127). A task-scoped venv under `.temp/TASK-260824-2o4zq8/skill-validator-venv` installed exactly `PyYAML==6.0.3`; the unchanged validator then exited 0. Details are retained in `tool-readiness-01.log`.

## Scope note

No live Qwen model inference/deployment smoke was run; that operator deployment is explicitly owned by downstream task `TASK-260824-2a4gk3`. This task validated production Qwen composition and official Pi identity without starting provider/runtime side effects.
