# TASK-260911-1diuhg review verdict — rev2 — accepted

All four rev1 rework items independently re-verified against the REAL live
config `/Users/alexis/src/.agents/.configs/model-harness.toml`, not taken on
the producer's report.

## 1. Gate wired into the real live config — confirmed

`cat /Users/alexis/src/.agents/.configs/model-harness.toml` shows
`[profiles.qwen-local.pinned_distribution]` with `package = "mlx_lm"`,
`site_packages` pointing at the real pipx venv, `commit =
45a472f2d0cda166b7ffe1a80fe50dd9621f4303` — matches README exactly. Backup
`.bak-TASK-260911-1diuhg` present beside it.

## 2. `status=ok` against the real file with the pin actually evaluated — confirmed

Built `model-harness` from this worktree's source myself (not trusting the
producer's build) and ran it directly against the real file:

```
$ go build -o /tmp/mh-rev2-review ./cmd/model-harness
$ /tmp/mh-rev2-review doctor qwen-local --host 127.0.0.1 --port 18011 \
    --config /Users/alexis/src/.agents/.configs/model-harness.toml
profile=qwen-local mode=local executable=/Users/alexis/.local/bin/mlx_lm-relux.server status=ok
```

## 3. Negative control against the real profile/package/commit, live venv untouched — reproduced independently

Built my own temp copies of the real dist-info (not the producer's temp
files) with `site_packages` redirected, `package`/`commit` taken verbatim
from the real config:

```
$ # dir_info.editable=true injected into a temp copy
$ mh doctor qwen-local --config .../config-editable.toml
model-harness: pinned distribution mlx_lm is an editable install; refuse editable installs for a pinned fork
exit=1

$ # commit_id replaced with 40 zeros in a temp copy
$ mh doctor qwen-local --config .../config-drifted.toml
model-harness: pinned distribution mlx_lm is drifted: installed commit 0000...0000 does not match configured pin 45a472f...
exit=1
```

Confirmed the live pipx venv was never touched: `direct_url.json` under
`~/.local/pipx/venvs/mlx-lm-relux/.../mlx_lm-0.32.0.dist-info/` still reads
non-editable, `commit_id = 45a472f2d0cda166b7ffe1a80fe50dd9621f4303`, byte
for byte what rev1 verified. `ps aux | grep -iE 'mlx_lm-relux|model-harness'`
returns nothing — no qwen server or model-harness process was running or
disturbed at any point in this review.

## 4. Operator-owned steps stated explicitly, known gap tracked separately — confirmed

- README's new "Operator follow-up — CLI binary redeploy" section and
  LOGBOOK 2026-09-11/1710 both call out, with the exact rebuild command,
  that the currently-deployed `~/.local/bin/model-harness`
  (`v1.6.1-140-g61d832d`, built 2026-09-09, predates this feature) now
  refuses to parse the live config at all until rebuilt — reproduced myself:
  `model-harness doctor qwen-local --config .../model-harness.toml` against
  the real deployed binary returns `decode config ...: strict mode: fields
  in the document are missing in the target struct`, exit 1. This is a
  real, self-inflicted regression from wiring the live config (which rev1
  explicitly requested), and it is disclosed prominently, not buried — not
  a reason to reject on its own given the task's own DoD already frames
  binary redeploy as a documented operator step, not something the task
  executes.
- `TASK-260911-2zcdqe` exists on the board (`status=backlog`,
  compose-generation-recovery-into-the-pinned-fork-tree) — the known
  `45a472f` vs `9150698` gap is tracked there, not silently fixed inside
  this task, matching LOGBOOK 2026-09-11/1640 and README's "Known gap in
  the pinned commit itself" paragraph.

## Mechanism re-confirmed unchanged from rev1

Repo source is identical to rev1 except README/LOGBOOK prose. Re-ran
`go build ./...`, `go vet ./...`, `gofmt -l .` (clean), and
`go test ./internal/modelharness/... -run 'PinnedDistribution|Doctor' -v`:
all 11 pinned-distribution/doctor tests pass. Code review of
`pinned_distribution.go`: editable check precedes the VCS/commit check
(an editable git-VCS install is refused regardless of commit match),
ambiguous/missing dist-info and non-git/no-vcs_info installs are refused,
exact 40-char commit match required — no gap found.

## Verdict

accepted. All four rev1 rework items are answered with evidence against the
real deployed config, independently reproduced by this review rather than
taken on the producer's word. The stale-deployed-binary side effect is a
genuine, well-disclosed, non-destructive consequence of doing exactly what
rev1 asked for, with an exact remediation step already documented.

repeat-of: none
