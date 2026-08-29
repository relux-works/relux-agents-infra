# TASK-260824-2a4gk3 — reviewer verdict: ACCEPTED

Reviewer run `RUN-260824-7f9a37` (claude-opus-5 / high).
Change Request `CR-TASK-260824-2a4gk3-1` revision 1.

Candidate identity was proved before any conclusion: the live Story worktree
tree hashes to exactly the candidate tree, so every gate below ran against the
reviewed bytes and not against a drifted checkout.

```text
git stash create -> 78a2658 ; tree = d932b6614b0095958ba356ebb1206ff2289eebac
git diff <live tree> d932b661...  -> empty
```

Repository delta attributable to this task is 4 paths on top of `f81197f`
(`skill_link_validation.go`, `installed_binary_setup_test.go`, `README.md`,
`LOGBOOK.md`). The other 16 paths in the CR set were already committed by
`TASK-260824-3rl3ws` / `TASK-260824-2o4zq8`.

## Gates attacked, not read

The only production behavior change is a **narrowing of a containment gate**:
`managedSkillLinkFailures` now derives the managed-name set from
`.agents/.skills` in local mode too, instead of inspecting every provider-owned
top-level package. A relaxed guard is exactly the shape that must be attacked,
so it was attacked from both directions.

| Attack | Result |
| --- | ---: |
| Pre-fix binary vs. the real deployment target | **Reproduced the defect.** `verify local /Users/alexis/src/casual-talks` refuses both `mac-infra` links; `verify global` on the same runtime already passes. The mode asymmetry the LOGBOOK names is real, not a convenient story. |
| Mutant B — drop `.claude/skills` from the inspected surfaces | **expected-red.** `TestInstalledBinaryVerifyLocalInspectsEveryManagedSkillSurface` fails naming the un-inspected surface. |
| Mutant C — delete the managed-name skip (widen back) | **expected-red.** `TestInstalledBinarySetupAndVerifyLocalPreserveUnmanagedProviderSkillLinks` fails with both external links refused. |
| Mutant D — inspect only directories, skip top-level symlink entries | **expected-red.** Escape probe on `.agents/skills` is admitted; the surface test fails. |
| Byte-for-byte restore after each mutant (`cmp`) | clean; live tree still hashes to the candidate tree. |

Production call site is real: both tests drive the installed binary through
`setup local` / `verify local`, which reach `managedSkillLinkFailures` via the
setup postcondition and the verify path — not a helper called from nowhere.

The narrowing is **exactly** scoped, verified by reading what setup creates:
`ensureRepoSkillLinks` populates `.agents/skills` only from `.agents/.skills`
entries plus the materialized `repoSkillName` (which is itself materialized
*into* `.agents/.skills`), and `setupClaude`/`setupCodexWithConfig` fan out only
from `.agents/skills`. Every name setup owns is therefore in the managed set and
still fully validated on all three surfaces.

Read-failure handling was checked separately, because absence and failure to
read are different facts: a failed `os.ReadDir` on `.agents/.skills` returns the
failure and never proceeds with an empty managed-name set, so an unreadable
surface cannot degrade into "nothing is managed, skip everything".

## Deployment and provenance, independently re-run

All three aliases were re-resolved from `/Users/alexis/src/casual-talks` with
the installed binary. Wrapper SHA-256 values match the producer's table; each
wrapper carries only its entrypoint name and the sibling dispatch.

| Entrypoint | vendor / environment | model | reasoning |
| --- | --- | --- | --- |
| `openai-infra` | openai / codex | `gpt-5.6-sol` | `high` |
| `anthropic-infra` | anthropic / claude-code | `claude-opus-5` | `high` |
| `qwen-infra` | qwen / pi | resolved MLX weights path | `off` |

Section 5 Qwen invariants recomputed from the production compose document:

```text
resolved.model.value == resolved.profile_provider.value + "/" + target.model   PASS
resolved.endpoint.value == pi.runtime.endpoint                                 PASS
```

`lsof` and `pgrep` returned exit 1 before and after every diagnostic, so
print/compose stayed non-launching.

## Live Qwen/Pi smoke — independently reproduced

The producer's live-run claim was not accepted on narrative. It was re-run from
this review session against the real runtime:

```text
qwen-infra -- -p --mode json --no-session -nc -ne -ns -np --no-themes \
  -t write,read -- "<three-step prompt>"
exit 0, 22:04:09 -> 22:05:20 (71s), 105,868-byte JSON transcript
```

Observed in the transcript: `provider=local-qwen`, `model=` the exact absolute
weights path, `REVIEW_TEXT_OK`, `tool_execution_start`/`_end` x2 for `write` and
`read`, `REVIEW_ROUNDTRIP_OK`, and `usage.reasoning == 0` on all three turns —
which is the producer's `off`-reasoning claim reproducing exactly. The model's
`<think>` chat-template blocks appear as parsed content but bill zero reasoning
tokens, so they do not contradict `--thinking off`.

The reviewer's own tool round trip wrote and read back a distinct task-scoped
file (`review-roundtrip.txt`, 39 bytes) rather than reusing the producer's
artifact, so the write is this session's, not a replay.

Post-run: no listener on 18011, no `mlx_lm.server`, no `session.lock`, and the
project config SHA-256 is still `464c699f...` — unchanged across three
print-configs, one compose, one full launch, and a verify.

## Suites

`go build ./...`, `go vet ./...`, `gofmt -l`, `git diff --check` all clean.
`go test ./... -count=1` green in this session: `agents-infra` 89.390s,
`internal/attachments` 2.084s, `internal/infra` 149.257s.

## Findings (recorded, non-blocking, for the orchestrator)

1. **Contract Section 2 is stale against deployed reality.** The attached
   revision-3 contract still names `Qwen3.8-27B-MLX-8bit` as the target/profile
   model, while the deployed identity is the resolved absolute weights path.
   That substitution is a legitimate operator architecture decision recorded on
   `TASK-260824-1jjze0` and already accepted there — this task inherited it, so
   it is not rework here. But the contract artifact now disagrees with every
   shipped config, and a future reader will trust the document. Worth a Story-
   level doc correction.
2. **The live-smoke evidence shipped as prose plus one 39-byte file.** The
   claims are true — they reproduced — but nothing in the attached set could
   have failed if they had not been. A launch transcript should be attached, not
   narrated. This review attaches its own
   (`TASK-260824-2a4gk3_reviewer-qwen-smoke.jsonl.gz`) to close the gap.
3. **Alias operand ergonomics are unobvious.** A Pi message needs two
   delimiters (`qwen-infra -- <pi flags> -- "message"`); the first attempt
   without them fails with a Go `flag` usage block that lists only
   `-print-config`. The refusal is correct and the second attempt's message
   ("managed Pi operands require the wrapper -- delimiter") is genuinely
   actionable, so this is documentation polish, not a defect.

## Verdict

Accepted. The narrowed gate is bounded in both directions by production-driven
negative tests, the relaxation matches exactly the set of names setup owns, the
deployment resolves the configured tuples with correct provenance, and the
headline live claim reproduces end to end with clean cleanup and unchanged
config bytes.

No `commit_ack` supplied: the commit-owning mover integrates the accepted
revision.
