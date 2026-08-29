# TASK-260824-2a4gk3 — review verdict, Change Request revision 2

Reviewer run: `RUN-260824-6431f9` (claude-opus-5 / high).
Verdict: **ACCEPTED** (`accept_cr(TASK-260824-2a4gk3, revision=2)`).

One blocking-at-integration finding is recorded below for the orchestrator. It is
not a defect in the reviewed delta and it is not fixable by a producer under the
Story-workspace rules, so it does not route to `to-dev`.

## Candidate identity — proved, not accepted on narrative

The live worktree was hashed into a throwaway index before any gate ran:

```text
GIT_INDEX_FILE=<temp> git add -A && git write-tree
  -> 95d12fb4c2aaf6050ff51f2c74fee7a81041acff   == candidate tree OID
```

Re-hashed after every mutant build and restore: still `95d12fb4…`. Nothing this
review did leaked into the reviewed bytes.

Revision 1 (accepted, tree `d932b661…`) vs revision 2:

```text
git diff --stat d932b661 95d12fb4  ->  LOGBOOK.md | 10 ++++++++++
```

Only `LOGBOOK.md`. Every reviewed production, test, README, and SKILL byte from
the accepted revision 1 is bit-identical.

## The reconciliation itself

```text
git diff --numstat main:LOGBOOK.md 95d12fb4:LOGBOOK.md  ->  66  0  LOGBOOK.md
```

Zero deleted lines. The candidate `LOGBOOK.md` is a **strict superset** of
current `main`'s: trunk's `1815 — Managed Codex Sync Preserves User State` and
`1753 — Fast-Profile Removal Meets Runtime Preservation Boundary` appear
byte-for-byte, in timestamp order between the Story's `1823` and `1751`. Neither
history is dropped. This is exactly what the rework was asked to produce.

## FINDING (integration, orchestrator-owned) — the base was never advanced

Revision 2 carries the **same base OID `cf21665…`** as the refused revision 1.
Content reconciliation does not move a base, so the structural cause of the
original refusal is unchanged. Reproduced:

```text
CAND=$(git commit-tree 95d12fb4 -p cf21665 -m candidate)
git merge-tree --write-tree --name-only main "$CAND"
  -> CONFLICT (content): Merge conflict in LOGBOOK.md      (exit 1)
     Auto-merging README.md / SKILL.md / infra.go / infra_test.go   (clean)
```

The identical command against revision 1's tree `d932b661` produces the identical
conflict. Under the Change Request contract, `main` has advanced (`cf21665` ->
`2f74fb0`) and intersects five paths this Change Request also changes —
`LOGBOOK.md`, `README.md`, `SKILL.md`, `tools/agents-infra/internal/infra/infra.go`,
`tools/agents-infra/internal/infra/infra_test.go` — which is the
`integration_base_moved` -> `stale` route, independent of mergeability.

Why this is not `changes_requested`: the fix is a base refresh (checkpoint the
Story workspace, replay the branch onto `2f74fb0`), and the Story-workspace
contract forbids a spawned producer from rebasing or merging this branch. The
board already records why it did not happen automatically — *"base refresh
SKIPPED: the managed workspace holds uncommitted work, so there was no clean
checkpoint branch to replay onto trunk 2f74fb0c"*. Routing rework here would ask
a producer to fix something it is not permitted to touch.

Resolution is mechanical when the base is refreshed: the only conflicting file is
`LOGBOOK.md` and the candidate side is a proven strict superset, so *take the
candidate* loses nothing from trunk. The other four intersecting paths merge
clean.

## Gate attacked, not read

The task's own source change narrows `managedSkillLinkFailures`
(`tools/agents-infra/internal/infra/skill_link_validation.go:29`) so provider-owned
top-level skill packages are skipped in **both** modes instead of global only.
Narrowing a containment gate is exactly the shape that must be attacked in both
directions. Production entrypoint is the installed binary's `setup`/`verify`.

| Probe | Expectation | Result |
| --- | --- | --- |
| `TestInstalledBinaryVerifyLocalInspectsEveryManagedSkillSurface` (unmutated) | green | PASS 7.91s |
| `TestInstalledBinarySetupAndVerifyLocalPreserveUnmanagedProviderSkillLinks` (unmutated) | green | PASS 2.82s |
| **Mutant A** — drop `.claude/skills` from the validated roots | red | FAIL, names the un-inspected `.claude/skills` surface |
| **Mutant B** — populate `managedNames` as an empty set (skip every top-level package) | red | FAIL, names the un-inspected `.agents/skills` surface |
| **Mutant C** — restore the pre-fix `ModeGlobal`-only asymmetry | red | FAIL, refuses the external `mac-infra` links |

File SHA-256 re-checked as `366ecf7d…` after every restore.

Mutant C was also built as a **real binary and pointed at the real deployment**,
which reproduces the actual bug rather than a fixture of it:

```text
agents-infra-prefix  verify local /Users/alexis/src/casual-talks  -> exit 1
  managed skill link escapes runtime containment:
    /Users/alexis/src/casual-talks/.claude/skills/mac-infra -> /Users/alexis/.agents/skills/mac-infra
    /Users/alexis/src/casual-talks/.codex/skills/mac-infra  -> /Users/alexis/.agents/skills/mac-infra
agents-infra-prefix  verify global -> exit 0        <- the mode asymmetry, live
agents-infra-candidate verify local /Users/alexis/src/casual-talks -> exit 0
agents-infra-candidate verify global -> exit 0
```

The narrowing is bounded by ownership, not by convenience: setup's fan-out set is
exactly the entries of `.agents/.skills` (`ensureRepoSkillLinks`, `infra.go:584`,
which also materializes the repo skill *into* `.skills` before linking it out), so
`managedNames` names precisely what setup owns. An unreadable `.agents/.skills`
appends a failure and returns — it does not degrade to an empty managed set, so a
failed read is not laundered into an absence.

## Identity locks attacked at the installed aliases

Run from `/Users/alexis/src/casual-talks` against `~/.local/bin/*-infra`:

| Probe | Exit | Code |
| --- | ---: | --- |
| `openai-infra -- --model divergent-model` | 1 | `target_identity_conflict` |
| `openai-infra -- --model gpt-5.6-sol` (exact repeat) | 0 | accepted |
| `openai-infra -- exec -- --profile work` (post-delimiter) | 1 | `target_identity_conflict` |
| `openai-infra -- -c model="other"` | 1 | `target_identity_conflict` |
| `anthropic-infra -- -- --model other` (post-delimiter) | 1 | `target_identity_conflict` |
| `anthropic-infra -- --effort low` | 1 | `target_identity_conflict` |
| `anthropic-infra -- --effort high` (exact repeat) | 0 | accepted |
| `qwen-infra -- --model <path>:high` (divergent thinking suffix) | 1 | `target_identity_conflict` |
| `qwen-infra -- --model other/<path>` (divergent provider) | 1 | `target_identity_conflict` |
| `agents-infra target bogus-infra --print-config` | 1 | `unknown_entrypoint` + field path + remediation |
| `compose --entrypoint qwen-infra --agent codex` | 1 | refuses both selectors |

Project-config SHA-256 stayed `464c699f4dfe505203bc0ac80abb05238f6e04ef645ff36dba65e86b6f26b7b6`
before and after every probe.

## Section 2 tuples and Section 5 invariants — recomputed from production compose

All three `compose --mode primary-session --entrypoint … --schema-version 1 --json`
exited 0 against `/Users/alexis/src/casual-talks`.

| Alias | vendor / environment | model | reasoning |
| --- | --- | --- | --- |
| `openai-infra` | openai / codex | `gpt-5.6-sol` | `high` |
| `anthropic-infra` | anthropic / claude-code | `claude-opus-5` | `high` |
| `qwen-infra` | qwen / pi | `/Users/alexis/src/local-models/Qwen3.8-27B-Uncensored-MLX-8bit` | `off` |

Target and effective provenance for all three resolve to the `casual-talks`
project config. Hosted plans report `profile_provider`/`endpoint` as
`not_applicable`; Codex reports `effective_profile_source: native`.

Qwen Section 5 equalities, recomputed against
`[agents.pi.profiles."qwen-3.8-27b-mlx-8bit"]`:

```text
target.model            == profile.model      (absolute weights path)   PASS
target.reasoning "off"  == profile.thinking   "off"                     PASS
target.endpoint         == profile.base_url   http://127.0.0.1:18011/v1 PASS
resolved.profile_provider.value == profile.provider  "local-qwen"       PASS
resolved.model.value == "local-qwen" + "/" + target.model               PASS
resolved.endpoint.value == pi.runtime.endpoint                          PASS
profile api == "openai-completions"                                     PASS
pi.runtime.argv contains --host 127.0.0.1 --port 18011 (loopback only)  PASS
```

No secret-shaped token and no `env` node appears anywhere in the three compose
documents.

## Live Qwen/Pi smoke — independently re-run this session

Not accepted on the producer's narrative and not accepted on revision 1's
transcript. Re-run against the real runtime with a distinct reviewer-owned
marker and file:

```text
qwen-infra -- -p --mode json --no-session -nc -ne -ns -np --no-themes \
  -t write,read -- "<REV2 three-step prompt>"
exit 0, 19:48:23 -> 19:49:18 (55s), 69,954-byte / 246-event JSON transcript
```

Observed: `provider = local-qwen`; `model` = the exact absolute weights path;
assistant text `REV2_TEXT_OK`; `tool_execution_start`/`_end` for `write`
(31 bytes to `.temp/TASK-260824-2a4gk3-rev2-review/rev2-roundtrip.txt`, a path no
prior run used) and for `read` (returns exactly `rev2-reviewer-RUN-260824-6431f9`,
`isError: false`); assistant text `REV2_ROUNDTRIP_OK`; `usage.reasoning == 0` on
every assistant message, which reproduces the `off` claim rather than inferring it.

**Reported as unknown, not as a failure:** a first attempt with a longer prompt was
killed by *this reviewer's own* 560s tool timeout (`signal: terminated`, empty
transcript). That is a failed read on my side, not a product failure, and it is
not counted as evidence either way. Its one useful by-product: even when the
wrapper was SIGTERMed mid-generation, the post-kill listener, `mlx_lm.server`
process, and lock-holder checks all returned absent — the managed Pi lifecycle
reaped the runtime it started.

Post-run cleanup (failing commands, absence expected):

| Probe | Exit | Meaning |
| --- | ---: | --- |
| `lsof -nP -iTCP:18011 -sTCP:LISTEN` | 1 | no listener |
| `pgrep -af 'mlx_lm\.server'` | 1 | no runtime process |
| `lsof <profile session.lock>` | 1 | no lock holder |

Project config SHA-256 still `464c699f…`. The reviewer's scratch directory under
`casual-talks/.temp/` was removed; `casual-talks` carries only its own
pre-existing unrelated modifications.

## Suites

Run in this session, in the reviewed worktree:

| Command | Result |
| --- | --- |
| `go test -count=1 ./...` | ok 72.690s / 1.508s / 109.468s |
| `go build ./...` | clean |
| `go vet ./...` | clean |
| `gofmt -l .` | empty |
| `git diff --check` | clean |
| `agents-infra verify global` | exit 0 |
| `agents-infra verify local /Users/alexis/src/casual-talks` | exit 0 |

## Non-blocking, carried forward

1. **Contract and README still print `Qwen3.8-27B-MLX-8bit`.** The deployed and
   only working identity is the absolute weights path — `LOGBOOK 2051` records
   that the literal ID makes `mlx_lm.server` treat it as a Hugging Face repo,
   fail `RepositoryNotFound`, and leave an empty HTTP shell alive. The README
   canonical-target example (`README.md:755`) is therefore a copy-paste that does
   not start. Story-level doc correction, not a defect in this delta. This is the
   same finding revision 1's review raised; it is still open.
2. Pi operands need two `--` delimiters (`qwen-infra -- <pi flags> -- "message"`).
   Worth one line in the README alias section.
