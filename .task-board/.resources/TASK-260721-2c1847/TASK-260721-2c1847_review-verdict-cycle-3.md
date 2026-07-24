# Review verdict — changes requested (cycle 3)

## Passing evidence

- `go test ./... -count=1`, `go vet ./...`, and `go test -cover ./... -count=1` pass. Attachment-package coverage is 71.7%.
- Current source cross-compiles the infra test package for `windows/amd64`.
- Global `agents-infra attachments` and backwards-compatible `agents-attachments` both show usage and exit 2. The installed launcher stages an image through Go successfully.
- `agents-infra doctor global` and `agents-infra doctor local /Users/alexis/src/casual-talks` both report `helpers_linked: true`.
- Active attachment docs and implementation contain no Python runtime dependency; setup removes the legacy helper and installs a Go launcher.

## Regression

`findRolloutPath` in `tools/agents-infra/internal/attachments/attachments.go:733` selects every `rollout-*.jsonl` basename that *contains* the requested thread ID. The legacy Python helper used `rglob(f"rollout-*{thread_id}.jsonl")`, which only accepts names ending in that thread ID.

Reproduction used two rollout files under an isolated `HOME/.codex/sessions` directory:

- expected (older): `rollout-run-needle.jsonl`
- unrelated (newer): `rollout-run-needle-unrelated.jsonl`

`agents-infra attachments materialize --thread-id needle` wrote a manifest whose attachment and top-level `session_path` both reference the unrelated, newer file. Materialization can therefore ingest another session's images when its filename merely contains the target ID.

Evidence and executable smoke artifacts: `.temp/TASK-260721-2c1847/attachment-review-VTwGfx/`.

## Required rework

1. Preserve legacy filename selection semantics: require a `rollout-` basename ending in `threadID + ".jsonl"` before newest-mtime selection.
2. Add a Go regression test with an expected rollout and a newer same-substring-but-unrelated rollout; it must select only the expected session.
3. Re-run the Go suite, vet, coverage, and the installed-launcher smoke before another reviewer cycle.

## Verdict

Changes requested. Route `TASK-260721-2c1847` to `to-dev` for focused implementation rework.
