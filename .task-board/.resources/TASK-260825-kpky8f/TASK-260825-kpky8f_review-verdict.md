# TASK-260825-kpky8f review verdict — ACCEPTED

Reviewer: reviewer-archetype tracked run. Date: 2026-08-25.
Change Request `CR-TASK-260825-kpky8f-1` revision 1.

## Why an empty `repository_delta` is the right outcome here

The delta is empty **by snapshot construction, not by absence of work**. The
producer committed its scope as `5b081ad` ("Support native Qwen thinking and
reject Pi yolo") on `task-board/story/STORY-260825-7oqacp`, and that commit is
the CR's own base OID. Candidate tree `b88203b` therefore equals the base tree.

`5b081ad` is the sole commit on this story branch (parent `8f01e31` is the
previous story's board-state commit) and its message carries `TASK-260825-kpky8f`.
It changes 13 files, +276/-39. That commit is the reviewable deliverable, and it
is what I reviewed and attacked. Accepting an empty diff here is accepting a real,
present, verified repository change — not accepting "no change".

## Acceptance criteria, verified by me at production entry points

I built the binary from source (`go build ./`) and drove the real CLI against my
own fixture projects; I did not rely on the producer's reported runs.

| AC | Result | How I proved it |
| --- | --- | --- |
| Non-launching qwen-infra plan reports reasoning medium + source | PASS | `agents-infra target qwen-infra --print-config` → `effective_reasoning: medium`, `effective_reasoning_source: <fixture config path>` |
| Launch argv forwards correct Pi-native reasoning selection | PASS | `provider_argv` = `--provider local-qwen --model <id> --thinking medium` |
| yolo_mode=true rejected with precise unsupported-capability error | PASS | All four production entry points refuse (below) |
| Existing direct Pi behavior and safe defaults compatible | PASS | Default `{value:false, source:"default"}`; direct Pi argv unchanged; child `false` masks ancestor `true` |
| Focused Go tests and documentation checks pass | PASS | Full suite, vet, gofmt, cross-platform builds, doc drift guards |

## Gates attacked, not read

### Qwen native-thinking gate — 5 negative fixtures, all refused with exact fields

| Attack | Outcome |
| --- | --- |
| `compat.thinking_format` removed | `invalid_target`, field `...compat.thinking_format` |
| `thinking_format = "qwen"` (wrong value) | `invalid_target`, field `...compat.thinking_format` |
| profile `reasoning = false` + `thinking = "medium"` | `invalid_project_configuration`, field `...profile.reasoning` |
| target `medium`, profile `thinking = "off"` | `invalid_target`, field `agents.targets.qwen-mlx-8bit.reasoning` |
| profile `medium`, target `reasoning = "off"` | `invalid_target`, field `agents.targets.qwen-mlx-8bit.reasoning` |

Identity lock holds against argv override: `--thinking off`, `--thinking high`,
`--thinking=off`, `--provider other` are all refused with
`target_identity_conflict`. Only the exact configured value is admitted.

### Pi yolo gate — every production entry point, before executable lookup

`target qwen-infra --print-config`, `pi --print-config`, `pi --version` (real
launch path), `compose --mode primary-session --agent pi`, and
`compose --mode primary-session --entrypoint qwen-infra` all exit 1 with
`pi_yolo_mode_unsupported` and the precise message naming the source path,
`--approve`, and the remediation. Ancestor-`true` with no child override still
refuses; child `yolo_mode = false` correctly masks it.

### The contract is real, verified against the pinned Pi binary itself

I did not take the producer's probe on faith. Reading pinned Pi `0.84.2`:

- The dispatch branch is `compat.thinkingFormat === "qwen-chat-template" &&
  model.reasoning` → `chat_template_kwargs = { enable_thinking:
  !!options?.reasoningEffort, preserve_thinking: true }`. Notably this branch
  does **not** consult `supportsReasoningEffort` (unlike the `zai`, `qwen`, and
  `baseten` branches), so the profile's `supports_reasoning_effort = false` is
  correctly irrelevant. The three coordinates the gate enforces are exactly
  necessary and sufficient.
- `reasoningEffort = clampedReasoning === "off" ? undefined : clampedReasoning`,
  so `--thinking medium` yields a truthy effort and `enable_thinking: true`.
- **The gate is load-bearing, not decorative**: `getSupportedThinkingLevels`
  begins `if (!model.reasoning) return ["off"]`, and `clampThinkingLevel` then
  silently downgrades a requested `medium` to `off`. Without the enforced
  `reasoning = true`, an operator configuring `medium` would get *silently
  non-thinking* Pi. The gate prevents precisely that capability-claim-that-does-
  not-reproduce failure.
- `GeneratePiModelsJSON` is the production generator (called from both
  `pi_plan.go:169` and `pi_launch_posix.go:139`) and emits `reasoning` and
  `compat.thinkingFormat`; it emits no `thinkingLevelMap`, so `medium` survives
  clamping.

### The yolo refusal is correct, not lazy

Independently confirmed that pinned Pi has no unattended-execution policy to map
to: `pi --help` documents `--approve, -a` as "Trust project-local files for this
run"; both argument parsers in the binary map `--approve`/`-a` to
`projectTrustOverride` only; and there is no `--yolo`, `--dangerously-*`,
`--auto-approve`, `--skip-permissions` flag, nor any `autoApprove` /
`permissionMode` / `approvalPolicy` settings key. Translating yolo to `--approve`
would have been a security-semantics lie — granting project-file trust while the
operator believed they enabled unattended execution. Rejecting is right.

## My own narrowing mutants (independent of the producer's two)

Delete-only mutants prove nothing, so all three narrow the gate instead:

| Mutant | Result |
| --- | --- |
| M1: remove the yolo gate from the **compose call site only** (`pi_plan.go`), leaving `RunPi` intact | `TestPiYoloTrueFailsClosedBeforeComposeOrLaunchLookup` FAILED both `/compose` and `/launch` subtests |
| M2: narrow Qwen gate `target.Reasoning != "off"` → `== "high"` | `TestCanonicalQwenProfileAssertionsFailClosed` FAILED `missing_qwen_thinking_format` + `wrong_qwen_thinking_format` |
| M3: narrow parse gate `p.Thinking != "off"` → `== "high"` | FAILED `TestParsePiPolicyRejectsMalformedUnsafeUnknownAndNarrowedInputs/non-reasoning_medium` and `TestCanonicalQwenProfileAssertionsFailClosed/profile_cannot_reason` |

M3 also demonstrates defense in depth: narrowing the parse gate is still caught
downstream by the canonical gate, and because the test asserts the exact code
**and** field, the shape change alone fails it. Working tree restored exactly;
`git diff --stat` empty after each restoration.

Production call sites named and driven: `BuildPrimarySessionLaunchPlan`,
`buildPiPrimarySessionLaunchPlan`, `RunPi` (posix + windows),
`BuildCanonicalTargetLaunchPlan`, `ResolveCanonicalTarget`,
`GeneratePiModelsJSON`.

## Suite and hygiene, rerun by me

| Check | Result |
| --- | --- |
| `go test ./internal/infra -count=1` | ok, 82.778s |
| `go test . -count=1` | ok, 71.016s |
| `go test ./internal/attachments -count=1` | ok, 0.958s |
| `go vet ./...` | clean |
| `gofmt -l tools/agents-infra/` | clean |
| `GOOS=windows GOARCH=amd64 go build ./...` | ok |
| `GOOS=linux GOARCH=amd64 go build ./...` | ok |

No lint target exists in the repo; `go vet` + `gofmt` are the lint gate.

## Documentation

README and the source-of-truth `SKILL.md` document the native contract, the safe
default, the refusal, and the operator workflow. Critically, both new sections
are pinned by doc drift-guard tests (`pi_operator_docs_test.go`,
`model_check_docs_test.go`) asserting the exact new fragments — closing the same
class of regression recorded in LOGBOOK entry 1449, where doc sections could be
deleted with a green suite. The producer also corrected pre-existing README/SKILL
text that had wrongly described `--approve` as making tool calls "execute
unattended"; that was a real documentation defect and the fix is in scope.

Logbook entries recorded (2 references to the task ID). Probe artifact is
sanitized — host model/runtime paths omitted, no tokens or key material.

## Non-blocking observation

The producer's results table lists its compose/`--print-config` verifications
without naming the fixture project directory, while the probe records a baseline
of `reasoning off` from the repo's own untracked
`.agents/.configs/project-config.toml` (which still carries `reasoning = false`
/ `thinking = "off"` / target `reasoning = "off"`). Those runs therefore used a
different project dir than the probe baseline, which the table does not state.
An evidence-precision nit only: every claim reproduced exactly on my own
independent fixture, so nothing rests on it. Worth tightening in future evidence
tables by naming the fixture path.

## Verdict

**ACCEPTED.** The implementation matches the AC, fits the existing canonical-
target / primary-session composition architecture, enforces a contract that I
verified against the pinned Pi runtime's actual code rather than its docs, and
is covered by negative tests that fail under three independent narrowing mutants
at named production call sites. Handing to the orchestrator for the `done`
transition with `commit_ack=scope_committed`.
