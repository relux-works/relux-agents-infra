# TASK-260824-3rl3ws — Review Verdict (CR rev3): ACCEPTED

Reviewer run `RUN-260824-b8cf91`. Change Request `CR-TASK-260824-3rl3ws-3`
revision 3, base `cf21665dde35274cc14e66e26a93574e0c18c15c`, candidate tree
`efc3d3119f3ca7747d4c866e1f08d17ad1f444f2`, repository delta `present`
(1 path: `LOGBOOK.md`).

## Verdict

Accepted. Both rev2 blocking findings are closed, and both were re-verified
against production code and by executing the evidence, not read off the
rework note.

## G1 — hosted `resolved.profile` semantics: CLOSED

Rev2 rejected §5 for forcing `resolved.profile=not_applicable` on every hosted
alias plan, which is false for Codex. Revision 3 now states per-provider
semantics. Verified directly against the composer:

- `tools/agents-infra/internal/infra/primary_session_launch_plan.go:277-281` —
  the Codex branch sets `Profile` to the explicit CLI value/source when
  `plan.ExplicitProfile`, otherwise `Source: "native"`. The contract's
  "Codex uses an explicit CLI source when a profile is selected explicitly and
  `native` when Codex configuration decides" is exactly this.
- `primary_session_launch_plan.go:460` — the Claude branch sets
  `Source: "not_applicable"`. Matches the contract.
- `primary_session_launch_plan.go:108-112` — `not_applicable` is documented as
  "the field does not exist for the provider", which is why the rev2 blanket
  claim was wrong and why the rev3 split is the correct reading.

The new Codex selector clause was also checked rather than accepted:
`codex_launch.go:798-818` (`normalizeCodexConfigOverride`) records `-c model=`
and `-c model_reasoning_effort=` as explicit selections via
`acceptCodexExplicitValue`, so treating them as identity selectors alongside
`--model` / `--model-reasoning-effort` is grounded. `-c profile=` is *not* in
that switch, so §3's requirement that an alias reject it is a new obligation on
the alias parser rather than a claim about current behavior — the contract does
not misstate current code, and §5's "alias plan reports the Codex profile
coordinate as `native`" follows from the `ExplicitProfile == false` branch.

## G2 — non-failing narrowing controls: CLOSED

Rev2 rejected the seven "narrowing controls" because
`grep -Fv M f | grep -Fq M` is tautologically empty and could not fail. The
dead pipeline is gone. I ran `.temp/TASK-260824-3rl3ws/validate-contract.sh`
myself and then attacked it. Three gates fire:

| Attack | Expectation | Result |
| --- | --- | --- |
| Baseline run against the four live board copies | pass | pass, exit 0, hashes match `c0db2515` |
| A1: feed a contract with the `none edits it` clause removed as the *real* contract | reject | exit 1, `missing marker: none edits it` |
| A2: append one byte to the implementation copy | reject | exit 1 (`cmp -s` under `set -e`) |
| A3: delete `|| return 1` from a mutant-covered marker (`error.remediation`) in `check_contract` | mutant survives → loop fails | exit 1, `mutant survived contract validation` |

A3 is the decisive one: it proves the mutant loop is capable of failing and is
a real control over the exact regression rev2 named — that `check_contract`
must propagate an intermediate marker failure, which a shell function used as
an `if` condition does not do under `set -e` without the explicit `|| return 1`.

Byte identity independently reproduced. The primary outcome and all three
downstream precondition resources
(`TASK-260824-2o4zq8_`, `TASK-260824-1jjze0_`, `TASK-260824-2a4gk3_vendor-target-contract.md`)
hash to `c0db2515ed3a1055d34d0f9384889deea51268b8f06b8ad5dd402d3bb498717b`,
matching the claimed value. Downstream tasks cite and receive the contract, as
the AC requires.

## Rev1 findings F1-F5 re-checked at rev3

Revision 3 edited §3, §5 and §7, so the rev1 anchors were re-verified rather
than inherited:

- F1 (Pi composite `--model` bypass): `pi_args.go:307-324`
  `validateManagedModelSelection` still accepts `model:thinking` and
  `provider/model:thinking`, and `pi_args.go:268-270` still lets `modelThinking`
  override `effectiveThinking`. §3 rule 6 locks the decoded grammar coordinate
  by coordinate and §7.6 requires the divergent-suffix negative. Still closed.
- F2 (reasoning domains): all three enumerations match code exactly —
  Codex free non-empty string (`project_config.go:235`
  `projectConfigNonEmptyString`), Claude `low|medium|high|xhigh|max`
  (`claude_launch.go:44`), Pi `off|minimal|low|medium|high|xhigh|max`
  (`pi_args.go:69`). The unknown-effort native fallback the contract relies on
  is `primary_session_launch_plan.go:448-456`.
- F3 (`resolved.model` divergence): `pi_plan.go:185` still emits
  `profile.Provider + "/" + profile.Model`; §5's invariant block states the
  provider-qualified relationship explicitly.
- F4 (`local-qwen` magic literal): §2.3 now treats profile `provider` as an
  operator namespace with an optional `profile_provider` assertion; §7.5
  requires a non-`local-qwen` label to be accepted.
- F5 (`resolved.endpoint` duplication): `pi_plan.go:193` sets
  `Runtime.Endpoint = profile.BaseURL`; §5 ties `resolved.endpoint.value ==
  pi.runtime.endpoint` and scopes the field to alias plans only.

## Additional checks

- Alias dispatch shape is consistent with the established wrapper pattern:
  `infra.go:1062-1071` (`piInfraWrapperBody`) already resolves a sibling target
  via `dirname $0`, refuses a missing/non-executable target with exit 127, and
  never consults `PATH`. §4's requirement is the same contract, not a new one.
- Additive fields under schema v1 have prior art in this repo:
  `PrimarySessionResolved.PiCompatibility` is `json:"pi_compatibility,omitempty"`
  (`primary_session_launch_plan.go:105`), the same shape §5 proposes for the
  alias-only `target`, `resolved.profile_provider`, and `resolved.endpoint`.
- Board shape matches §8 exactly: Story `STORY-260824-1yr6m0` has precisely the
  four tasks, and the dependency chain `3rl3ws -> 2o4zq8 -> 1jjze0 -> 2a4gk3`
  is linked in both directions with the extra `2a4gk3 <- 2o4zq8` prerequisite.
- `LOGBOOK.md` delta: four entries, newest-first ordering correct, each claim
  matched to a contract decision. The 1842 FINDING ("creates clause-removal
  mutants and requires the actual validator to reject them; absent markers fail
  before mutation") is accurate as written.
- `go build ./...`, `go vet ./...`, `go test ./... -count=1` rerun by this
  reviewer in the Story worktree: all green
  (`agents-infra` 69.6s, `internal/attachments` 2.6s, `internal/infra` 113.0s).
  No Go source changed in rev3; this is an independent confirmation, not an
  inherited claim.

## Non-blocking observations

These are limits of a scratch validator that lives in gitignored `.temp/` and
is not part of the repository delta. None affects the contract's correctness,
and the rework-validation resource does not overclaim past them — it says
"nine", not "all".

1. The mutant loop covers 9 of the 11 markers `check_contract` enforces.
   `Claude reasoning is deliberately closed` and
   `resolved.model.value == resolved.profile_provider.value + "/" + target.model`
   are enforced against the real contract but are never mutated. Verified:
   deleting `|| return 1` from the Claude marker leaves the validator passing
   (exit 0). The two clauses are still load-bearing for the real contract,
   because a top-level `check_contract "$contract"` failure aborts under
   `set -e`; only the self-test of the propagation mechanism has the gap.
2. Marker checks are substring presence, so a semantic inversion that keeps the
   marker substring passes. Verified: rewriting the clause to "closed to
   nothing: any token is accepted" still exits 0. Inherent to marker-based
   document validation; the real guarantee against clause drift here is the
   four-way byte-identity check, which does fire.
3. `require_marker` assigns the global `marker` without `local`, so
   `check_contract` clobbers the loop variable. The healthy path prints correct
   names only because the mutant's own marker is the last one checked before the
   early return; under A3 the failure message named marker 11 while marker 7's
   mutant was the one that survived. Cosmetic, but it would misdirect the next
   person who has to diagnose a real failure.

If any of these is worth carrying forward, it belongs to the implementation
task's own test harness (§7), not to a rework cycle on this contract.

## Acceptance handoff

`accept_cr` parks `TASK-260824-3rl3ws` at `to-review`. Per the reviewer
constraint this run supplies no `commit_ack`; the orchestrator commits the
`LOGBOOK.md` scope and makes the `done` transition with
`commit_ack=scope_committed`.
