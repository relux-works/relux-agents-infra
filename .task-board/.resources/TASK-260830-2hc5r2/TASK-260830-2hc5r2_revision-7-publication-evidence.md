# TASK-260830-2hc5r2 — revision 7 verification and publication evidence

## Result

The accepted revision-6 repository delta was restored on current trunk without
changing its implementation, measurements, fork pin, provenance, attestation
logic, or deployed default profile. Fresh verification on base
`436760d62f4ea451cf49614ff7e40109d96915b3` passed and the candidate is ready
to publish as Change Request revision 7.

## Fresh-base and byte-identity proof

All commands ran in the managed Story worktree unless an absolute artifact path
is shown.

| Command / gate | Exit | Evidence |
| --- | ---: | --- |
| `git rev-parse HEAD`; `git rev-parse main` | 0 | both `436760d62f4ea451cf49614ff7e40109d96915b3` |
| `git rev-list --count HEAD..main`; `git rev-list --count main..HEAD` | 0 | `0`; `0` — current trunk, no branch commits |
| `shasum -a 256 /Users/alexis/src/relux-works/relux-agents-infra/.temp/kv-rev6-task-delta.patch` | 0 | `350123ecbc83a81a0a8b2c2b71c9f92486aeb7c211810ad0b6e990d7793861f5` |
| `git apply --reverse --check --exclude=LOGBOOK.md /Users/alexis/src/relux-works/relux-agents-infra/.temp/kv-rev6-task-delta.patch` | 0 | all 15 non-LOGBOOK paths are byte-identical to accepted revision 6 |
| 16-path exact-count gate over `git diff --name-only HEAD` | 0 | exactly 16 changed paths |
| `.configs/codex-config.toml` tracked/untracked absence gate | 0 | absent from candidate delta |
| `git diff --check` | 0 | no whitespace errors |

The first artifact lookup used the brief's literal worktree-relative
`.temp/kv-rev6-task-delta.patch` and exited 128 because the managed worktree has
its own nested `.temp`. `git rev-parse --path-format=absolute --git-common-dir`
resolved the control repository; the named artifact was then found at its
control-root absolute path above. No bytes were copied, repaired, or changed.

## Additive LOGBOOK proof

Both pre-merge sides were reconstructed from Git, not from the spawn brief:

- trunk side: `git show 436760d62f4ea451cf49614ff7e40109d96915b3:LOGBOOK.md`
- Story revision-6 side: stash snapshot
  `93c38ad93a128fd69bc1dc98c21951b600965e6c:LOGBOOK.md`; Git records its first
  parent as the original Story base
  `3295c7da7151de128f176cf7560a57d54c8f6c0d`

The exact-line gate extracted each `## 2026-08-30` section and required every
non-blank line to occur verbatim in the merged working section:

| Source | Non-blank lines | Missing from merged |
| --- | ---: | ---: |
| trunk pre-merge | 15 | 0 |
| Story revision 6 pre-merge | 37 | 0 |
| merged candidate | 51 | — |

The source totals share one identical date header, hence 51 merged non-blank
lines rather than 52.

## Fresh validation on the new base

| Command / gate | Exit | Exact result |
| --- | ---: | --- |
| `swift test -c release` | 0 | 287 tests in 24 suites passed |
| `xcrun swift-format lint --strict --recursive Sources Tests` | 0 | 0 findings |
| `xcodebuild build -scheme mlx-swift-runtime-prototype -configuration Release -destination 'platform=macOS,arch=arm64' -derivedDataPath ./DerivedData -skipPackagePluginValidation -skipMacroValidation` | 0 | `BUILD SUCCEEDED`; one pre-existing `quantization` deprecation warning |
| `BINARY="$PWD/DerivedData/Build/Products/Release/mlx-swift-runtime-prototype" OUT="$PWD/../../.temp/TASK-260830-2hc5r2/benchmark-gate-smoke-rev7" PORT=19771 scripts/benchmark-gate-smoke.sh` | 0 | 52 PASS, 0 FAIL |

The production smoke drove the real `benchmark-run` call site. Its negative
matrix includes omitted live KV, omitted runtime configuration, caller-profile
argv rewriting, repeated and abbreviated Python argparse values, decoy runtime
provenance, models-only placeholders, missing scenarios, forged records,
unclosed/malformed/foreign attestations, and replay refusal.

The smoke reproduced the already-recorded cleanup debt and left its own
task-scoped fake runtime group `67737` listening on port `19791` after the gate
had exited successfully. Exact argv inspection proved both processes belonged
to this revision-7 output directory; only that group was terminated with TERM.
No listener remains in the smoke range `19771...19796`.

## Preserved accepted evidence

Per the revision-7 scope, the hour-scale 73k generation was not rerun or
re-derived. The existing accepted task outcome proves a 73,139-token prompt
produced the correct three-system answer with `finish_reason=stop`, 73,111
cached tokens, and post-generation live `meta.n_ctx=76800`. Revision 6 already
verified that signed fork commit
`45a472f2d0cda166b7ffe1a80fe50dd9621f4303` differs from that proven signed
parent only in live model-listing reporting and tests. The reverse-apply proof
above establishes that revision 7 preserves those pins and semantics exactly.

