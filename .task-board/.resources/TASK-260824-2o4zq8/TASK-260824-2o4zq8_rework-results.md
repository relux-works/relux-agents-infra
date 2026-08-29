# TASK-260824-2o4zq8 reviewer rework evidence

## Handoff

- Original implementation commit: `0246afc` (`Resolve canonical vendor targets`)
- Reviewer rework commit: `ba0d95d` (`Lock hosted targets across wrapper delimiter`)
- Story branch: `task-board/story/STORY-260824-1yr6m0`
- Preserved unrelated state: the pre-existing staged/unstaged `LOGBOOK.md` split was not reset, staged, or included in the rework commit. The working copy carries the review root cause and the append-only rework evidence entry.

## Reviewer findings addressed

- Codex and Claude identity locks now continue through the first hosted-wrapper `--`, because their wrapper parsers consume that delimiter and keep provider selectors active.
- A second `--` remains the provider-native operand boundary; Pi retains its first-`--` message operand boundary.
- Ordinary hosted alias launches no longer print the full launch plan to stderr. `--print-config` remains the explicit non-launching diagnostics surface.

## Production negative evidence

`TestRunTargetRejectsHostedIdentitySelectorsAfterWrapperDelimiter` drives `runTarget -> BuildCanonicalTargetLaunchPlan -> lockCanonicalTargetArguments` and refuses eight post-delimiter identity classes:

- Codex `--model`
- Codex `--model-reasoning-effort`
- Codex `-c model=...`
- Codex `-c model_reasoning_effort=...`
- Codex `--profile`
- Codex `-c profile=...`
- Claude `--model`
- Claude `--effort`

Every case requires stable `target_identity_conflict` plus the exact identity field, proves the recording provider was not executed, and proves project-config bytes are unchanged. Exact post-delimiter repeats remain accepted. Separate controls preserve the second hosted/provider boundary and Pi message-boundary refusal.

## Narrowing mutant

A disposable copy at `.temp/TASK-260824-2o4zq8/mutant-module.4G46hE` narrowed both locks back to returning at the first delimiter. The production-entry test ran with `-count=1` and exited 1; all eight subtests failed. The source worktree was never mutated. Evidence: `rework-narrowing-mutant-01.log`.

## Validation

Every gate was run as a standalone process; exit codes below are the real process statuses.

| Command | Exit | Evidence |
| --- | ---: | --- |
| Focused production/unit delimiter tests with `-count=1 -v` | 0 | `rework-focused-02.log` |
| Narrowed first-delimiter mutant production test with `-count=1` | 1 | Expected red; `rework-narrowing-mutant-01.log` |
| `go test ./... -count=1` | 0 | main `64.153s`, attachments `2.472s`, infra `102.405s`; `go-test-full-01.log` |
| `go vet ./...` | 0 | `go-vet-01.log` |
| `go build ./...` | 0 | `go-build-01.log` |
| `gofmt -d` plus empty-output assertion | 0 / 0 | `gofmt-diff-01.log` is empty |
| `git diff --check` | 0 | `git-diff-check-01.log` |
| `git diff --cached --check` | 0 | `git-diff-cached-check-01.log` |
| Isolated global setup without bootstrap sibling | 1 | Expected postcondition refusal; `setup-global-01.log` |
| Isolated global setup after installing exact bootstrap sibling | 0 | `setup-global-02.log` |
| Isolated `verify global` | 0 | `verify-global-01.log` |
| Isolated project-local setup | 0 | `setup-local-01.log` |
| Installed project-local `verify local` | 0 | `verify-local-01.log` |

Both verified surfaces contain executable sibling-only `openai-infra`, `anthropic-infra`, and `qwen-infra` aliases beside their exact `agents-infra` target. The initial global failure is not counted as a pass; it proves the clean-HOME bootstrap boundary is enforced.
