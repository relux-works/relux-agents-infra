# TASK-260825-39ycg2 — Review Verdict (RUN-260825-2e91cf, cycle 3)

**Verdict: ACCEPTED** — `CR-TASK-260825-39ycg2-2` revision `2`.

Candidate tree `67b5e4f8ab40ac302cc38911509d5d4018dcae7e` was confirmed
byte-identical to the live Story worktree before and after every probe
(`git write-tree` twice, unchanged). Everything below was executed by this
reviewer against that exact tree.

## Cycle-2 blocking finding is closed and proven closed

Cycle 2 (`RUN-260825-f75a58`) proved the operator contract could be deleted
whole while the full uncached suite stayed green. Revision 2 adds
`tools/agents-infra/model_check_docs_test.go`. This reviewer re-ran the exact
deletion mutant **and** narrowing mutants the rework list demanded. Every
mutant was applied to the real file, the named production test was run
uncached without a pipe, and each file was restored and `cmp`-verified.

| # | Mutant (all applied to the real source, then restored byte-exact) | Test | Exit |
| --- | --- | --- | ---: |
| M1 | Delete entire README `#### Bounded model behavior checks` (lines 672-754) | README contract | `1` |
| M2 | Delete entire SKILL `### Bounded model checker` (lines 416-469) | SKILL contract | `1` |
| M3 | **Narrow:** exit-5 row "takes precedence over" → "follows" unmet expectations | README contract | `1` |
| M4 | **Narrow:** artifact mode `0600` → `0644` | README contract | `1` |
| M5 | **Narrow:** overwrite-refusal sentence removed, section otherwise intact | README contract | `1` |
| M6 | **Narrow:** `--approve` unattended warning softened | README contract | `1` |
| M7 | **Narrow:** skill failed-read rule → permissive absence fallback | SKILL contract | `1` |
| M8 | **Code-side:** `DefaultModelCheckDeadline` 5m → 4m in `model_check.go` | both | `1` / `1` |
| — | Unmutated re-run after full restore | both | `0` / `0` |

M3-M7 matter more than M1-M2: the pin is proven by **narrowing**, not only by
deletion. M8 is the important one — mutating the production constant reddens
the docs, so the deadline and exit fragments are genuinely derived from
`infra.MinimumModelCheckDeadline` / `DefaultModelCheckDeadline` /
`MaximumModelCheckDeadline` / `ModelCheckExit*` rather than hand-copied. The
contract now fails in both drift directions.

`README.md` and `SKILL.md` were confirmed byte-identical to their pre-mutation
copies after the battery; the tracked tree OID is unchanged.

## Docs re-verified against production code (no discrepancies)

Read `internal/infra/model_check.go`, `pi_run_report.go`, `pi_launch_posix.go`,
and `main.go` against every documented claim. Confirmed accurate:

- `0700` dir (`MkdirAll` + explicit `Chmod`) and four `O_EXCL` `0600` artifacts;
  `Lstat`-based refusal on all four names, so a dangling-symlink squat also refuses.
- Exit ordering in `evaluateModelCheckOutcome` matches the README table exactly,
  including tool-failure (`5`) checked **before** expectations (`4`).
- Deadline scope: `context.WithTimeout` wraps `RunPi` only, after target
  resolution and output preparation — as documented.
- `--expect-text` is `strings.Contains` against the final assistant `message_end`
  text only; `--expect-tool` is exact-name.
- `summary.json` carries `deadline_ms`, `duration_ms`, `timed_out`, and
  `managed_runtime.{pi_process_group_cleanup,runtime_process_group_cleanup,cleanup_confirmed}`.
- Terminal output is only `RenderModelCheckSummary`, gated on `SchemaVersion != 0`,
  which is why "early option validation may fail before summary artifacts exist"
  is exact.
- The constants refactor is behavior-neutral: `MinimumModelCheckDeadline`
  renders `1ms` and `MaximumModelCheckDeadline` renders `30m0s`, preserving the
  refusal string that `model_check_main_test.go:279,290` asserts.

## Suites and negative batteries

| Command | Exit | Result |
| --- | ---: | --- |
| `go build ./...` | `0` | — |
| `go vet ./...` | `0` | No findings |
| `go test . -count=1` (uncached) | `0` | `109.231s` |
| `go test ./internal/... -count=1` (uncached) | `0` | `attachments 1.410s`, `infra 135.087s` |
| `go test . -run 'TestModelCheckProductionEntrypoint\|TestMainKeepsProviderChildFailuresAtLegacyExitOne' -v -count=1` | `0` | 14 subtests, all negative-path refusals green (`16.391s`) |
| `agents-infra verify global` | `0` | Installed runtime verified |
| `qwen-infra --print-config` | `0` | Canonical entrypoint resolves |
| `cmp -s SKILL.md $HOME/.agents/skills/relux-agents-infra/SKILL.md` | `0` | Installed skill byte-identical to source |
| `grep -c '^### Bounded model checker' $HOME/.agents/skills/.../SKILL.md` | — | `1` — the new section is actually installed, not only in source |
| `agents-infra version` | `0` | `v1.6.1-28-gac759d9` |

The production battery includes the shapes that matter: forged/absent evidence
(missing tool, missing text, malformed JSONL refused instead of treated as
absence), bypass (non-managed canonical target refused without inventing a
stream failure), bound-by-narrowing (`2h` and `0` deadlines refused with no
artifacts written), and overwrite refusal on both a full and a partial
collision. The happy path asserts two-sidedly that the fixture secret **is**
preserved in raw `events.jsonl` and **is not** present on stdout/stderr or in
the sanitized summary — so "provider output is never mirrored to the terminal"
is a tested behavior, not a doc assertion.

## Qwen smoke independently re-derived — conclusion is accurate

All four artifact digests reproduce exactly as reported; dir `0700`, files `0600`:

```text
events.jsonl  f089f23d617c670b3c4af91a2cbd11a07f0cf1e2fd1003ecee995720c76bcbc6
stderr.log    a9da0b2d7bf5325258199604cc00f0ef40fbaa82990fcb99e012fa32efa4055f
summary.json  c320bf117c109ef4c83ccef265d785dc0f097a08246fcbfcf65ca3500ba60568
summary.txt   89286380f2acffb1555a33fbb4db368a4aa672d7178cbce0521624a2cc696a49
```

Re-parsed the raw stream myself: line 166 `tool_execution_start` `read`
`{"path":"$HOME/.agents/skills/relux-agents-infra/SKILL.md"}`, line 167
`tool_execution_end` `isError=false` on the same `toolCallId`
`1aac3b4a-…`. Skill discovery and the installed-skill read are proven by the
tool event, not by the model's self-report — which is exactly how the docs and
`results.md` frame it.

The run is reported as a **failure**, correctly: `status=timed_out`, exit `2`,
`300192ms` against a `300000ms` deadline, `process_exit_code: unknown`
(context deadline, not an `exec.ExitError`), event stream valid but
**incomplete** (no `agent_end`), final assistant text is `"\n\n"` so the
`RELUX_SKILL_READ_CONFIRMED` marker is genuinely unmet. Both process-group
cleanups `confirmed`. Nothing here is inflated into a passing smoke, and the
absent marker is not spun as "no skill read".

The cycle-2 non-blocking nuance is now stated plainly in `results.md:56-63`:
the smoke read the installed revision immediately before the last prose sync,
42 bytes shorter, rewrapping only, and the smoke was **not** rerun against the
final bytes. Disclosed rather than left to timestamps.

## Non-blocking observation (no rework required)

Three README sentences inside the new section remain outside the pin. Proven,
not assumed — each was mutated for real and the four doc-contract tests stayed
green, while a control mutation of a pinned fragment reddened them (exit `1`),
so the probe harness itself is sound:

- "Provider stdout/stderr is never mirrored to the terminal." (deleted → green)
- "`--expect-text` matches a substring of the final assistant response only."
  (weakened to "anywhere in the stream" → green)
- "It covers managed runtime readiness and the Pi agent run after static target
  resolution and output preparation." (deleted → green)

This is doc-drift exposure only: all three **behaviors** are pinned by
`TestModelCheckProductionEntrypoint` (secret-on-terminal assertion; the
"expected text is limited to the final assistant response" subtest; the
readiness/session deadline subtests). The cycle-2 rework list is fully
satisfied, so this is a note for a future pass, not a blocker.

## Reviewer hygiene

Read-only on product code; no `commit_ack` supplied. Every mutation ran and
restored inside a single shell invocation under an `EXIT` trap, and the tracked
tree OID is unchanged from the candidate. Scratch under
`.temp/REVIEW-39ycg2-r2/` was removed. One disclosure: probing required
`git write-tree`, which needed `git add -A`, so the worktree **index** is now
normalized (5 staged paths) instead of the CR-snapshot delete/untracked
bookkeeping it had on entry. No working-tree file, no content, and no commit
was affected — the tree hashes identically. The orchestrator should still stage
by path when it commits.
