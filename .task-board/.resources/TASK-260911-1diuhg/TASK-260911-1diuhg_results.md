# TASK-260911-1diuhg: pin mlx-lm-relux to an explicit fork commit — results

## What changed

- **Host state (real, not simulated):** `pipx uninstall mlx-lm-relux` then
  `pipx install --suffix=-relux --python python3.14
  'git+https://github.com/relux-works/mlx-lm.git@45a472f2d0cda166b7ffe1a80fe50dd9621f4303'`.
  No qwen server process was running under this venv before the reinstall
  (`ps aux` confirmed empty), so nothing live was restarted. Verified after
  reinstall: `direct_url.json` for the installed `mlx_lm-0.32.0.dist-info` now
  reads
  `{"url": "https://github.com/relux-works/mlx-lm.git", "vcs_info": {"commit_id": "45a472f2d0cda166b7ffe1a80fe50dd9621f4303", ...}}`
  — non-editable, pinned, no `dir_info.editable`.
- **`tools/agents-infra/internal/modelharness`:** new `PinnedDistribution`
  profile config (`package`, `site_packages`, `commit`) and
  `VerifyPinnedDistribution` (`pinned_distribution.go`), wired into the
  production `Doctor()` function that backs `model-harness doctor PROFILE`
  (`run.go`). It reads the one `<package>-*.dist-info/direct_url.json` PEP 610
  record under the configured venv and refuses `status=ok` when the install is
  editable, has no git `vcs_info`, or its `commit_id` does not exactly equal
  the configured 40-character pin. Config validation rejects a short/abbreviated
  commit, a relative `site_packages`, and use on `mode = "ssh"` profiles.
- **README.md:** new "Pinned fork distribution" subsection under the
  model-harness config docs — the config shape, the exact non-editable
  reinstall command, and the retirement condition (drop the fork once a
  PyPI `mlx-lm` release carries upstream PR #1791 and the bounded-KV
  behavior).

## AC coverage — 5 of 6 rows driven, 1 stated bound (rev2: driven against the real live config)

| # | Story AC clause | Driven by | Production call site |
|---|---|---|---|
| 1 | installed non-editable from an explicit fork commit | real `pipx install`, verified via `direct_url.json` on this host | operator `pipx` step (documented in README) |
| 2 | tree carries bounded KV, generation-loop recovery (9150698), effective-config report (45a472f) | **partially false — see finding below** | n/a |
| 3a | pipx metadata agrees with the pin | `TestVerifyPinnedDistributionAcceptsNonEditablePinnedCommit`, `TestDoctorPassesPinnedNonEditableInstall`, **rev2:** live `model-harness doctor` run against the real, live `qwen-local` config at `/Users/alexis/src/.agents/.configs/model-harness.toml` (not a synthetic stand-in) | `modelharness.VerifyPinnedDistribution` via `modelharness.Doctor` via `model-harness doctor` |
| 3b | **running server** agrees with the pin | **not implemented — stated bound** (see below) | — |
| 4 | setup/verify refuses an editable install | `TestVerifyPinnedDistributionRefusesEditableInstall`, `TestDoctorRefusesEditablePinnedInstall`, **rev2:** live `model-harness doctor` run against the real config with only `site_packages` redirected to a temp copy of the real dist-info carrying a perturbed `dir_info.editable=true` | same call site as above |
| 4 | setup/verify refuses a drifted commit | `TestVerifyPinnedDistributionRefusesDriftedCommit`, `TestVerifyPinnedDistributionRefusesCommitSharingOnlyAPrefix`, **rev2:** live `model-harness doctor` run against the real config with only `site_packages` redirected to a temp copy of the real dist-info carrying a mismatched `commit_id` | same call site as above |
| 5 | retirement condition documented | README "Pinned fork distribution" section | — |
| 6 | negative test: editable or mismatched commit fails verify; pinned install passes | all of the above, **rev2:** now against the real config/profile/package/commit, not a synthetic stand-in (see "Rev 2 rework" below) | — |

Stated bound (row 3b): `VerifyPinnedDistribution` checks installed package
metadata (`direct_url.json`), not a running server process. The fork's
45a472f commit adds an "effective server generation configuration" self-report
endpoint that a live cross-check could hit, but wiring that requires starting
the actual backend process and is a materially larger, riskier change (spawns
a real model server) than this task's stated scope ("no runtime behaviour
change"). Not implemented; flagged for a follow-up task if the story owner
wants it.

## Finding: commit 45a472f does not actually contain 9150698

The task and story both assert the pinned tree (`45a472f` on
`task/TASK-260830-2hc5r2-bounded-kv`) "contains" the generation-loop-recovery
commit `91506981056172f937e7bdca4ab0d3b7459c7fab` (branch
`runtime/generation-recovery`). Verified against the actual fork checkout at
`/Users/alexis/src/relux-works/mlx-lm`:

```
$ git merge-base 9150698 45a472f
74e7cf931e84ef7c2f63e875adf414e20decc1c5   # common ancestor, NOT 9150698 itself
$ git merge-base --is-ancestor 9150698 45a472f; echo $?
1   # not an ancestor
```

`task/TASK-260830-2hc5r2-bounded-kv` branched from the shared ancestor
`74e7cf9` and only carries a *different* commit (`b0a45b8`, "Tie the health
endpoint status to generation thread liveness") from the health-readiness
side of the work. The actual generation-loop-recovery fix — the `[[]]` vs
`[None]` placeholder fix in `PromptProcessingBatch.extend` and the
per-request exception handling in the batched admission path — is **not**
present at `45a472f`:

```
$ grep -n "logits_processors = \[None\]" mlx_lm/generate.py   # still there at 45a472f
1124:            self.logits_processors = [None] * len(self.uids)
$ grep -n "The generation thread is not running" mlx_lm/server.py   # absent at 45a472f
(no output)
```

This is a real gap, not a hypothesis: the config, `pinned_distribution`
mechanism, and reinstall are all built to work with *whichever* commit is
correct, so fixing this is a one-line config/README update once the right
commit exists — but pinning to `45a472f` as currently instructed means the
installed fork is missing the resilience fix that motivated
`BUG-260827-2tul5n` in the first place. No branch on the fork (local or
`origin`) currently contains both fixes; producing one requires a human
decision/action in that separate repository (its own `AGENTS.md` reserves
commit authorship and pushes to the human), which is out of this task's scope
(pipx spec + agents-infra verify + README only, "no runtime behaviour
change"). Recommend a follow-up: rebase or cherry-pick `9150698` onto
`task/TASK-260830-2hc5r2-bounded-kv` (or vice versa), then update this pin.

## Mutant evidence (narrowing, not delete-only)

| Mutant | Narrows the gate to | Failing test | Bound demonstrated |
|---|---|---|---|
| `meta.DirInfo.Editable && pin.Package != "mlx_lm"` | admits an editable install specifically for package `mlx_lm` (the exact package this task pins) | `TestVerifyPinnedDistributionRefusesEditableInstall`, `TestDoctorRefusesEditablePinnedInstall` | editable-install refusal is real, not vacuous |
| `meta.VCSInfo.CommitID[:7] != pin.Commit[:7]` | admits any installed commit sharing only the first 7 hex characters with the pin | `TestVerifyPinnedDistributionRefusesCommitSharingOnlyAPrefix` | the compare is against the full 40-char SHA, not an ambiguous abbreviated prefix |

Both mutants were applied to `pinned_distribution.go`, confirmed to make the
named test fail (`go test` output captured during the session), then
reverted and confirmed byte-identical to the pre-mutant source via `diff`.

Token-preserving mutant: `VerifyPinnedDistribution` inspects the text content
of `direct_url.json` (a structured-JSON text file, not Go source, but text
data read from disk) and matches specifically on the `vcs_info.commit_id`
token. The `CommitID[:7] != pin.Commit[:7]` mutant above preserves that exact
token/field and the equality-comparison shape, only truncating both operands
to the first 7 characters — it is the token-preserving case: the "commit
matches pin" check still runs and still names `commit_id`, it is just
narrowed to accept any commit sharing a 7-character prefix. Run against the
full behavioral suite (`go test ./internal/modelharness/...`), not a static
checker, and caught by `TestVerifyPinnedDistributionRefusesCommitSharingOnlyAPrefix`.

## Test evidence

```
$ cd tools/agents-infra && go test ./internal/modelharness/... -v
ok  	.../internal/modelharness	17.629s   (all new + existing tests pass)
$ gofmt -l . ; echo exit=$?
exit=0
$ go vet ./...; echo exit=$?
exit=0
$ go test .                      # top-level package, isolated
ok  	.../tools/agents-infra	106.507s
```

`go test ./...` run as one invocation showed a `FAIL` in the top-level
`tools/agents-infra` package under concurrent load: this machine had other
sessions' `go test` processes running against shared global state
(`~/.local`, `~/.agents`) at the same time (confirmed via `ps aux`). Rerunning
every affected package in isolation (`go test .`, `go test
./internal/modelharness/...`, `go test ./internal/infra/...`, `go test
./internal/attachments/...`) passed with exit 0 each time, including the
specific test that appeared near the failure output
(`TestRunSetupRejectsNonBooleanCodexYoloMode`, isolated: PASS). Treating this
as unrun/flaky-under-contention rather than claiming a clean `./...` exit.

## Live CLI smoke (real binary, not just unit tests)

```
$ model-harness doctor qwen-local --host 127.0.0.1 --port 18011 --config .../config.toml
profile=qwen-local mode=local executable=/Users/alexis/.local/bin/mlx_lm-relux.server status=ok
# (pinned_distribution pointed at the real reinstalled venv + real commit)

$ model-harness doctor ... --config .../config-drifted.toml   # commit set to all-zero
model-harness: pinned distribution mlx_lm is drifted: installed commit 45a472f... does not match configured pin 0000...
exit status 1

$ model-harness doctor ... --config .../config-editable.toml  # fabricated dir_info.editable=true
model-harness: pinned distribution mlx_lm is an editable install; refuse editable installs for a pinned fork
exit status 1
```

## Rev 2 rework (answering review verdict rev1: changes_requested)

Rev1's gap: the mechanism (config, `VerifyPinnedDistribution`, `Doctor()` wiring) was
solid and independently reviewed, but the *real* live `qwen-local` config at
`/Users/alexis/src/.agents/.configs/model-harness.toml` was never actually updated —
`model-harness doctor qwen-local` against that real file still reported `status=ok`
without ever declaring, let alone evaluating, `pinned_distribution`. All rev1 negative
controls ran against synthetic/temp configs, not the real target the task's own Scope
line names.

### 1. Wired the gate into the real live config

Backed up, then edited `/Users/alexis/src/.agents/.configs/model-harness.toml` in place
(outside this repository, per the task's explicit authorization) — `.bak-TASK-260911-1diuhg`
kept beside it. Exact diff:

```diff
--- /Users/alexis/src/.agents/.configs/model-harness.toml.bak-TASK-260911-1diuhg
+++ /Users/alexis/src/.agents/.configs/model-harness.toml
@@ -17,6 +17,11 @@
 request_timeout_seconds = 600
 sample_interval_milliseconds = 250

+[profiles.qwen-local.pinned_distribution]
+package = "mlx_lm"
+site_packages = "/Users/alexis/.local/pipx/venvs/mlx-lm-relux/lib/python3.14/site-packages"
+commit = "45a472f2d0cda166b7ffe1a80fe50dd9621f4303"
+
 [profiles.qwen-remote-demo]
 mode = "ssh"
 ssh_executable = "/usr/bin/ssh"
```

`site_packages` was confirmed directly against the real pipx venv before writing it:

```
$ cat /Users/alexis/.local/pipx/venvs/mlx-lm-relux/lib/python3.14/site-packages/mlx_lm-0.32.0.dist-info/direct_url.json
{"url": "https://github.com/relux-works/mlx-lm.git", "vcs_info": {"commit_id": "45a472f2d0cda166b7ffe1a80fe50dd9621f4303", "requested_revision": "45a472f2d0cda166b7ffe1a80fe50dd9621f4303", "vcs": "git"}}
```

`$STORY_TEMP` below is
`/Users/alexis/src/relux-works/relux-agents-infra/.temp/STORY-260911-2lwfr0/`, a
scratch directory *sibling to* the Story worktree checkout (not a path relative to the
worktree root) — it holds validation-only artifacts (a locally built binary, temp
dist-info copies) that are not part of the repository change and are not committed.

### New finding surfaced by wiring the real config: the deployed CLI binary is stale

`model-harness` is a Go binary (`~/.local/bin/model-harness`, `v1.6.1-140-g61d832d`,
built 2026-09-09) installed *before* this task's `pinned_distribution` feature existed.
Config decoding is strict (`toml.NewDecoder(...).DisallowUnknownFields()`,
`config.go:211`), so once the live config declares `pinned_distribution`, the
*currently deployed* binary does not silently skip the new check — it refuses to parse
the config at all:

```
$ model-harness doctor qwen-local --host 127.0.0.1 --port 18011 \
    --config /Users/alexis/src/.agents/.configs/model-harness.toml
model-harness: decode config /Users/alexis/src/.agents/.configs/model-harness.toml: strict mode: fields in the document are missing in the target struct
exit status 1
```

This is worse than rev1's "gate is inert" gap and is a real, immediate regression for
anyone invoking the deployed `model-harness` against this profile before the binary is
rebuilt. The task's own authorization was scoped to the config file, not to redeploying
the global CLI binary (`~/.local/bin/model-harness` is normally refreshed by
`./setup.sh`, a separate operator action), so this rework did **not** overwrite the
deployed binary. Instead, all validation below uses a `model-harness` binary built from
this worktree's current source (`go -C tools/agents-infra build -o
$STORY_TEMP/rev2-build/model-harness ./cmd/model-harness`, not installed
anywhere) to prove the wiring is correct end to end against the real file. Redeploying
`~/.local/bin/model-harness` once this change is accepted is the explicit outstanding
operator follow-up — documented in README's new "Operator follow-up — CLI binary
redeploy" paragraph. It does not touch the qwen model server process (confirmed no
`model-harness` or `mlx_lm-relux.server` process was running throughout this rework —
`ps aux` checked before and after).

### 2. `status=ok` against the real file, pin actually evaluated

```
$ $STORY_TEMP/rev2-build/model-harness doctor qwen-local \
    --host 127.0.0.1 --port 18011 \
    --config /Users/alexis/src/.agents/.configs/model-harness.toml
profile=qwen-local mode=local executable=/Users/alexis/.local/bin/mlx_lm-relux.server status=ok
```

`doctor`'s `status=ok` line is unconditional text, so it alone does not prove the pin
was *evaluated* rather than skipped — proof is in the negative controls in step 3, which
use this identical real config/profile and flip the pin outcome.

### 3. Negative control against the real config, without touching the live pipx venv

Per the spawn brief's explicit safety constraint ("do NOT reinstall or modify the live
mlx-lm-relux venv — other sessions may be using the running qwen server"), the negative
control perturbs a **temp copy** of the real dist-info, referenced by a **temp copy** of
the real (already-edited) config with only `site_packages` repointed — `package` and
`commit` are untouched, taken verbatim from the real file:

```
$ cp -R /Users/alexis/.local/pipx/venvs/mlx-lm-relux/lib/python3.14/site-packages/mlx_lm-0.32.0.dist-info \
    $STORY_TEMP/rev2-negative-control/site-packages/mlx_lm-0.32.0.dist-info
# perturbed the copy's direct_url.json: added "dir_info": {"editable": true}

$ $STORY_TEMP/rev2-build/model-harness doctor qwen-local \
    --host 127.0.0.1 --port 18011 \
    --config $STORY_TEMP/rev2-negative-control/model-harness-editable.toml
model-harness: pinned distribution mlx_lm is an editable install; refuse editable installs for a pinned fork
exit status 1

# second temp copy: commit_id replaced with all-zeros (no editable flag)
$ $STORY_TEMP/rev2-build/model-harness doctor qwen-local \
    --host 127.0.0.1 --port 18011 \
    --config $STORY_TEMP/rev2-negative-control/model-harness-drifted.toml
model-harness: pinned distribution mlx_lm is drifted: installed commit 0000000000000000000000000000000000000000 does not match configured pin 45a472f2d0cda166b7ffe1a80fe50dd9621f4303
exit status 1
```

Then reconfirmed the untouched real file + untouched live venv still pass (nothing live
was disturbed by building the temp copies):

```
$ cat /Users/alexis/.local/pipx/venvs/mlx-lm-relux/lib/python3.14/site-packages/mlx_lm-0.32.0.dist-info/direct_url.json
{"url": "https://github.com/relux-works/mlx-lm.git", "vcs_info": {"commit_id": "45a472f2d0cda166b7ffe1a80fe50dd9621f4303", "requested_revision": "45a472f2d0cda166b7ffe1a80fe50dd9621f4303", "vcs": "git"}}

$ $STORY_TEMP/rev2-build/model-harness doctor qwen-local \
    --host 127.0.0.1 --port 18011 \
    --config /Users/alexis/src/.agents/.configs/model-harness.toml
profile=qwen-local mode=local executable=/Users/alexis/.local/bin/mlx_lm-relux.server status=ok
```

This closes rework item 3: the negative control now runs against the real profile
name, real package/commit values, and a copy of the real dist-info — only
`site_packages` is redirected to an isolated temp copy, so the live venv other sessions
may depend on is never written.

### 4. Operator-owned steps, stated explicitly

- **Done, operator-owned, prior to this task:** `pipx uninstall mlx-lm-relux` +
  non-editable reinstall from `45a472f2d0cda166b7ffe1a80fe50dd9621f4303` (rev1).
- **Done in this rework:** live `qwen-local` config at
  `/Users/alexis/src/.agents/.configs/model-harness.toml` now declares
  `pinned_distribution`; verified against the real file with a worktree-built binary.
- **Outstanding, operator-owned:** redeploy `~/.local/bin/model-harness` (rebuild from
  this change once accepted) so the *currently installed* CLI actually enforces the pin
  instead of refusing to parse the config. README's "Operator follow-up — CLI binary
  redeploy" paragraph states this explicitly, with the exact rebuild command.
- **Known, separately tracked:** `45a472f` does not contain `9150698` (rev1 finding,
  unchanged) — tracked as `TASK-260911-2zcdqe`; not in this task's scope.
- **Not restarted:** the qwen model server itself. No `model-harness`/
  `mlx_lm-relux.server` process was running before, during, or after this rework.

### Full suite re-run (rev2, unchanged source)

Repo source is identical to rev1 (only the external live config changed); re-ran to
confirm no drift since rev1's evidence:

```
$ cd tools/agents-infra && go build ./... && go vet ./...
(both exit 0)
$ go test ./internal/modelharness/... -v
ok  	.../internal/modelharness	14.091s   (all pinned-distribution/doctor tests pass)
$ gofmt -l .; echo exit=$?
exit=0
$ go test . -count=1
ok  	github.com/relux-works/relux-agents-infra/tools/agents-infra	116.613s
```
