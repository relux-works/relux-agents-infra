# TASK-260707-xx9bdv clean-workspace rework outcome

## Outcome

The global source and installed runtime contain no x-platform-airdrop,
Tap2Cash, Swipe2Cash, XPAirDrop, or T2C-specific instruction material. The
remaining generic Research & Knowledge Persistence section reads continuously
without the removed project rule.

The installed `~/.agents/.instructions` directory was moved intact into the
task artifact, leaving the runtime path absent, and was recreated from this
clean Story workspace by `agents-infra setup global --source-dir <workspace>`.
`agents-infra verify global` and `agents-infra doctor global` both passed.

## Exact preservation statement

The new preservation bundle contains:

- the full source workflow module containing the one removed Tap2Cash global
  workflow bullet;
- the full installed workflow module containing that same bullet;
- all four stale project-local instruction files previously removed from the
  global runtime, preserved in full;
- complete current source, installed pre-refresh, installed post-refresh, and
  rendered pre/post snapshots;
- source and installed removal diffs, checksums, provenance, and validation
  results.

This outcome claims preservation only for those explicitly listed bytes. It
does not claim the four project-local files were all part of the versioned
global source tree.

Artifact:
`TASK-260707-xx9bdv_clean-workspace-preservation-v2.tar.gz`

- Size: `249715 bytes`
- SHA-256:
  `7789747b840860a25db222c9ca63ac7e7db20e13dba900dc7ec3da51b620f13e`
- Internal checksum verification: exit 0 for all six claimed preserved files.

## Validation

- Strict negative alias gate across source, installed, rendered Codex, and
  rendered Claude surfaces: exit 0.
- Separator/case-flexible negative alias gate across the same surfaces:
  exit 0.
- Source/installed recursive diff after setup: exit 0.
- `go test ./... -count=1`: exit 0.
- `go vet ./...`: exit 0.
- `go build ./...`: exit 0.
- `gofmt -l .` empty-output assertion: exit 0.
- `git diff --check`: exit 0.
- Repository candidate delta: empty; the clean workspace has no task-unrelated
  changes.

Validation was run at
`71230fc187e6b11eb9aea5520616b20967e223e3`. `origin/main` advanced by two
non-instruction commits during testing; the managed Story branch was left
untouched as required, and a read-only alias gate against the newer upstream
instruction tree also passed.

No file from the old Story worktree was accessed.
