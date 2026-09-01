# TASK-260707-xx9bdv Rework Summary

## Outcome

The reviewer-identified preservation gap is repaired. The four project-local
instruction files previously named but absent from the task artifact are now
preserved byte-for-byte in a self-contained task-scoped archive. The global
runtime was rebuilt after its installed instruction directory was moved into
that recoverable artifact.

The source instruction tree, installed instruction tree, rendered Codex
instructions, and rendered Claude instructions contain no strict or
separator/case-flexible x-platform-airdrop, Tap2Cash, Swipe2Cash, XPAirDrop, or
T2C aliases.

## Preservation evidence

- Archive: `TASK-260707-xx9bdv_global-instructions-reset-artifact.tar.gz`
- Archive size: 115880 bytes
- Archive SHA-256:
  `2611265e4fb08d9c5e4707235106269a484350bc75ae5567b687c211c91a0d4c`
- Internal `SHA256SUMS`: 78 preserved files
- `shasum -a 256 -c SHA256SUMS`: exit 0
- Archive listing/read validation: exit 0
- Required four-file archive presence gate: exit 0
- Four source-to-artifact `cmp` checks: exit 0

The archive contains the original source worktree diffs, complete pre-cleanup
source and installed workflow modules, the installed tree moved away before
refresh, the post-refresh installed tree, both rendered agent files, validation
logs, and the four recovered project-local instruction files.

## Runtime refresh and validation

- `agents-infra setup global --source-dir "$PWD"`: exit 0
- `agents-infra verify global`: exit 0
- `agents-infra doctor global`: exit 0
- Strict project-specific alias search across all four delivery surfaces:
  exit 0, no matches
- Flexible separator/case alias search across all four delivery surfaces:
  exit 0, no matches
- Source/installed instruction basename comparison: exit 0, no extra installed
  files

The first strict-search wrapper attempt exited 1 before evaluating the search
because it assigned zsh's read-only `status` variable. It was not counted as a
product result; the corrected wrapper was rerun and exited 0.

## Build and test evidence

Evidence was gathered at Story worktree HEAD
`cf21665dde35274cc14e66e26a93574e0c18c15c`, 63 commits behind local `main`,
with pre-existing unrelated dirty Story changes left untouched.

- `cd tools/agents-infra && go test ./... -count=1`: exit 0
  - root package: 180.540s
  - `internal/attachments`: 1.965s
  - `internal/infra`: 306.542s
- `cd tools/agents-infra && go vet ./...`: exit 0
- `cd tools/agents-infra && go build ./...`: exit 0
- `cd tools/agents-infra && gofmt -l .` plus empty-output assertion: exit 0
- `git diff --check`: exit 0

The rework added only ignored task artifacts and refreshed the installed global
runtime; it introduced no new tracked worktree changes. The source cleanup
already exists in commit `d1c8d7d5649c37df394d3401101a9650491b4893`
(`feat: harden agent instruction setup`).

