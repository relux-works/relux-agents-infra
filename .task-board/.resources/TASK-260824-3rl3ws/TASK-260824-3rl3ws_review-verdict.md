# Review verdict — TASK-260824-3rl3ws

Verdict: **changes requested** → `analysis`
Change Request: `CR-TASK-260824-3rl3ws-1` revision 1 (not accepted)
Reviewer run: `RUN-260824-a5dafb` (reviewer archetype, read-only)
Reviewed artifact: `TASK-260824-3rl3ws_vendor-target-contract.md`
(sha256 `a49b07dba180e65bb2006d794ef4235dec639b0d99528b8a80e773266206d81c`)
Reviewed delta: `cf21665..3d7ee31` — `LOGBOOK.md` only, +8 lines.

## What was verified against the codebase, not just read

The contract's factual anchors were checked against the worktree, and the
majority hold:

| Contract claim | Evidence | Verdict |
| --- | --- | --- |
| Pi profile fields `provider/model/base_url/api/reasoning/thinking` | `internal/infra/pi_config.go:134-183` | accurate |
| Qwen reasoning domain `off..max` equals Pi thinking levels | `internal/infra/pi_args.go:69` | accurate |
| `api` must equal `openai-completions` | `internal/infra/pi_config.go:154-157` | accurate |
| `pi_compatibility` under `[agents.pi.primary_session]` | `internal/infra/pi_config.go:90-99` | accurate |
| Anthropic domain `low..max` | `internal/infra/claude_launch.go:44` | accurate |
| Plan contract + schema version 1, `resolved.*`, `not_applicable` | `internal/infra/primary_session_launch_plan.go:11-115` | accurate |
| Sibling alias model, drift repaired by setup, rejected by verify | `internal/infra/infra.go:1041-1065`, `internal/infra/runtime_receipt.go:185-215` | accurate |
| `target` is a free subcommand name | `main.go:40-62` | accurate |
| Legacy Pi validates config/profile/args/env/identity before runtime spawn | `internal/infra/pi_launch_posix.go:59-115` | accurate |
| `provider = "local-qwen"` appears in the reference policy | `README.md:399` | present, but see F4 |
| Build/vet unaffected by the doc-only delta | `go build ./...`, `go vet ./...` — both clean | accurate |

Board state is also sound: three tasks, dependencies `3rl3ws -> 2o4zq8 -> 2a4gk3`
linked both directions, and the contract is byte-identical in both downstream
precondition resources (`diff` clean, hashes match the validation log). Section 7
is a genuinely strong negative-evidence list — it names narrowing mutants (7.8),
read-failure-vs-absence (7.7), the legacy-table bypass path (7.9), and production
entrypoints in its preamble.

The findings below are why this is not accepted. The contract is copied into two
downstream tasks as a **frozen precondition**, so anything unresolved here never
reaches the implementer — and F1/F2 will be frozen into Go gates with locking
negative tests, where they become expensive to revisit.

## F1 — Identity lock has a bypass path through Pi's composite `--model` token

§3.6 locks alias identity per selector: an explicit model, reasoning, profile,
endpoint, or provider argument "may repeat the resolved value" and a conflicting
value fails. §7.6 requires only "conflicting explicit identity CLI selectors are
refused; equal repeats are accepted and normalized".

Pi's real `--model` grammar is not one coordinate. `validateManagedModelSelection`
(`internal/infra/pi_args.go:307-320`) accepts four forms — `model`,
`provider/model`, `model:thinking`, `provider/model:thinking` — and the
`:thinking` suffix is not decorative: `pi_args.go:265-269` assigns
`effectiveThinking = modelThinking`, i.e. a **model** token silently overrides
the reasoning coordinate.

Failure scenario: target `qwen-mlx-8bit` declares `reasoning = "off"`. Caller runs
`qwen-infra --model Qwen3.8-27B-MLX-8bit:high`. An implementation that reads §3.6
as "compare the model selector against `target.model`" and normalizes the base
(the natural reading, given §7.6 says equal repeats are "normalized") sees a
matching model, admits the token, and Pi launches at `thinking=high` under a
target whose declared identity is `off`. No negative in §7 would fail: 7.6 is
satisfied by a plain conflicting `--model`/`--thinking` pair, and the composite
token is never mentioned.

Required: state explicitly that the identity lock is evaluated over the *decoded*
Pi model token — provider part, model part, and thinking suffix each compared
against the target's corresponding coordinate — and add a §7 negative that fails
when a `model:thinking` (and `provider/model:thinking`) token whose base matches
the target is admitted with a divergent suffix.

## F2 — Reasoning domain is declared closed with no grounding and no required negative

§2.3 fixes a closed per-vendor reasoning domain (`minimal|low|medium|high|xhigh|max`
for Codex, `low..max` for Claude). Two problems.

1. **It contradicts the repo's deliberate posture and the contract's own sentence.**
   agents-infra does not enumerate Codex effort anywhere: `project_config.go:235`
   parses `reasoning_effort` as a non-empty string, `project_config_setup.go:119`
   validates only non-emptiness, `codex_launch.go:905` passes it through as
   `-c model_reasoning_effort=`, and `README.md:715` documents it as a non-empty
   TOML string. §2.3 itself says "Provider availability and model availability
   remain provider/runtime facts; the target resolver validates the schema and
   tuple, not a remote model catalog" — and then hardcodes a provider-owned value
   domain one table above that sentence. Consequence: when Codex ships a new
   effort level, `[agents.targets]` cannot express it until agents-infra ships Go
   code, while the legacy `[agents.codex.primary_session]` table it is meant to
   supersede can. Migration to the canonical path becomes a one-way loss of
   expressiveness, which §6 does not disclose. (For Claude the narrowing is more
   defensible — `claudeEffortValues` already exists — but it also silently removes
   the documented native-fallback case at `README.md:1275-1280`, where an unknown
   effort is deliberately *not* rejected and `resolved.reasoning` reports
   `source: native`. Under a closed target domain that state is unreachable; §5
   claims existing field meanings are retained.)

2. **No §7 negative covers it.** 7.1 covers vendor/environment cross-pairs; 7.2
   covers missing/empty/unknown *fields* and wrong TOML *types*. Nothing requires
   a refusal of an out-of-domain reasoning *value*. A declared gate with no
   evidence requirement is not a gate: an implementation that accepts any
   non-empty reasoning string passes every listed negative.

Required: decide the domain deliberately — either keep it open per vendor and say
so, or keep it closed, justify the divergence from the free-string treatment and
from §2.3's own principle, disclose the migration consequence in §6, and add the
missing per-vendor value-domain negative to §7.

## F3 — `target.model` and `resolved.model` cannot both be right on the Pi path

§2.3 requires target `model` to exactly equal profile `model`. §5 says
`resolved.model` reports the effective value. But `pi_plan.go:185` already sets
`Resolved.Model = profile.Provider + "/" + profile.Model` — the Pi plan reports a
**provider-qualified** model today.

So for `qwen-mlx-8bit` the emitted plan carries `target.model =
"Qwen3.8-27B-MLX-8bit"` and `resolved.model = "local-qwen/Qwen3.8-27B-MLX-8bit"`.
A machine consumer doing the obvious identity check — does the resolved model
match the declared target? — sees a mismatch on a correctly configured project.
This also decides F1's open question of what counts as an "equal repeat" for
`--model`, since `pi_args.go:312` accepts both bases.

The contract must state which form each field carries and what the invariant
between them is, rather than leaving an implementer to discover the qualification
at `pi_plan.go:185`.

## F4 — `provider = "local-qwen"` is a magic literal that contradicts §6

§2.3 requires a Qwen target's profile to have `provider = "local-qwen"`. But
`provider` is a free-form operator label: `validatePiProvider`
(`pi_config.go:397-407`) rejects only separators, globs, whitespace and Unicode
lookalikes. It is fed to Pi as `--provider` (`pi_args.go:275`), keys the generated
`models.json` (`pi_plan.go:99`), and participates in the accepted `--model` bases
(`pi_args.go:312`). `local-qwen` is nothing more than the label the README
reference policy happens to use; the sibling example uses `local-muse`.

§6 promises migration is additive and that an operator "reuse[s] the existing
complete Pi profile by name". An operator whose valid, working Qwen profile is
labelled `qwen-mlx` or `local-qwen-mlx` cannot attach a Qwen target without
editing a legacy profile field that also flows into their Pi provider key, model
selector forms and generated catalog. That is exactly the legacy mutation §6 says
is not required. The contradiction is latent only because the documented example
already says `local-qwen`.

Either justify the literal explicitly (and correct §6), or bind vendor↔profile by
a mechanism that does not commandeer a free-form operator field — e.g. make the
binding an assertion the target states, like `endpoint` already is.

## F5 — `resolved.endpoint` is added without an invariant or a scope (minor)

§5 introduces `resolved.endpoint`, but the Pi plan already serializes the same
fact as `pi.runtime.endpoint` (`PiRuntimePlan.Endpoint`, `pi_plan.go:34-43`). Two
fields carrying one fact with no stated must-equal invariant is a future
divergence. §5 also scopes only `target` out of legacy `--agent` output and never
says whether legacy `--agent pi` plans gain `resolved.endpoint` — the AC for this
task is specifically to resolve print/compose.

## What is not being asked for

No rewrite. Sections 1, 2.1-2.2, 2.4, 3 (precedence and the additive-strict
decision), 4, 6 and 8 are sound and factually grounded; the additive-strict
decision in particular is the right call and is correctly reflected in the
LOGBOOK entry. The board decomposition is minimal and traced. Fix F1-F5 in place,
re-copy the contract to the two downstream precondition resources so they do not
drift, and this is acceptable.
