# Review verdict — TASK-260824-3rl3ws (revision 2)

Verdict: **changes requested** → `analysis`
Change Request: `CR-TASK-260824-3rl3ws-2` revision 2 (not accepted)
Reviewer run: `RUN-260824-444ecb` (reviewer archetype, read-only)
Reviewed artifact: `TASK-260824-3rl3ws_vendor-target-contract.md`
(sha256 `f96e1b3f93e45662100ddd4e624210eaa4f965f1c6eab9fbfaf2cff38f552a02`)
Reviewed delta: `cf21665..684b497` — `LOGBOOK.md` only, +19 lines.

## Revision 1 findings: all five closed, verified against code

Each closure was re-checked against the worktree, not read off the rework note.

| Finding | Contract clause in rev2 | Code anchor re-verified | Closed |
| --- | --- | --- | --- |
| F1 composite Pi `--model` bypass | §3.6 locks the *decoded* token: model / optional provider / optional `:thinking` suffix each compared; §7.6 requires the `model:thinking` and `provider/model:thinking` negatives | `pi_args.go:307-324` accepts all four forms; `pi_args.go:268-270` still assigns `effectiveThinking = modelThinking` — the bypass the clause now closes | yes |
| F2 reasoning domain | §2.3 Codex reasoning is an open non-empty provider-owned string, Claude closed to composer-proven efforts, Pi closed to its owned enum; §6 discloses the Claude migration consequence; §7.2 adds the value-domain negatives incl. "a new non-empty Codex token is not locally rejected" | `project_config.go:235` non-empty string only; `claude_launch.go:44` `claudeEffortValues`; `claude_launch.go:377-384` + `primary_session_launch_plan.go:448-456` unknown effort → `source: native`; `pi_args.go:69` `piThinkingLevels` | yes |
| F3 `target.model` vs `resolved.model` | §5 states target model is unqualified, `resolved.model` stays provider-qualified, with the explicit `resolved.model.value == resolved.profile_provider.value + "/" + target.model` invariant | `pi_plan.go:185` `profile.Provider+"/"+profile.Model` | yes |
| F4 `local-qwen` magic literal | §2.3 makes profile `provider` an operator namespace, `profile_provider` an optional assertion, no literal inferred; §6 keeps the legacy label unedited; §7.5 requires a non-`local-qwen` label to be accepted | `pi_config.go:397-407` free-form label; `pi_args.go:275`, `pi_plan.go:99` consume it | yes |
| F5 `resolved.endpoint` scope/invariant | §5 scopes it alias-only, legacy `--agent` plans do not gain it, and requires `resolved.endpoint.value == selected_profile.base_url == pi.runtime.endpoint` | `pi_plan.go:193` managed path always sets `PiRuntimePlan.Endpoint = profile.BaseURL`, so the invariant is structurally satisfiable | yes |

Also re-verified independently: all four contract copies (architecture,
implementation, rollout, deployment preconditions) are byte-identical at
`f96e1b3f…`; Story children are exactly the four tasks in §8; the dependency
chain `3rl3ws → 2o4zq8 → 1jjze0 → 2a4gk3` is linked both directions with
deployment keeping its implementation prerequisite; `target` is still a free
subcommand name in `main.go:40-62`; `compose` today requires `--agent`
(`main.go:426-438`), so the §4 one-selector rule is an additive change, not a
contradiction; post-`--` Pi operands beginning with `-` are already refused
(`pi_args.go:85-89`), so §3.7's operand-boundary claim holds and there is no
passthrough bypass of the Pi identity lock. `go build ./...`, `go vet ./...`
and `go test ./... -count=1` were rerun in this worktree by this reviewer: all
green (`66.197s` / `1.965s` / `97.009s`).

Two findings block acceptance.

## G1 — §5 forces `resolved.profile = not_applicable` on hosted alias plans; that is false for Codex

§5: *"For hosted alias plans, resolved model/reasoning sources are the target
definition and profile/profile-provider/endpoint use `not_applicable`."*

`not_applicable` is not a free label. The schema documents it as the source used
*"when the field does not exist for the provider"*
(`primary_session_launch_plan.go:110-111`). For Claude that is exactly right —
`primary_session_launch_plan.go:460` already emits
`Resolved.Profile = {Source: "not_applicable"}`. For Codex it is not:
`primary_session_launch_plan.go:278-280` emits the explicit profile value with
its CLI source, or `native` when none was passed, because Codex genuinely has a
profile concept and its own `config.toml` profile still decides when agents-infra
passes none.

Failure scenario: an `openai-infra` alias plan is emitted for a project whose
Codex config selects a native profile. Under §5 as written the plan reports
`resolved.profile.source = "not_applicable"` — asserting Codex has no profile
coordinate — instead of `native`, which is the field's existing meaning for
"provider configuration decides". A machine consumer diffing an alias plan
against the legacy `--agent codex` plan for the same project sees a provenance
change on a field §5's own opening paragraph promises is unchanged
("Existing fields retain their meanings"). This is the same field-meaning
divergence class as F3/F5, and it gets frozen into three downstream precondition
copies before an implementer can question it.

Required: state the hosted rule per coordinate — `profile_provider` and
`endpoint` are Pi-only and correctly `not_applicable`; `profile` keeps its
existing per-provider meaning (`not_applicable` for Claude, `native` or the
explicit CLI source for Codex). While you are there, say explicitly what an
alias does with a Codex `--profile` / `-c profile=` argument: §2.3 forbids the
target field and §3.6 locks identity selectors, so today the reader must infer
that an explicit Codex profile is a `target_identity_conflict`. Note the Codex
`-c model=` / `-c model_reasoning_effort=` config-override forms already fold
into the same explicit-selection record (`codex_launch.go:798-818`), so naming
them in §3.6 costs one clause and removes the F1-shaped ambiguity for the
hosted path too.

## G2 — The "seven narrowing controls" cannot fail; they are not mutation evidence

`.temp/TASK-260824-3rl3ws/validate-contract.sh` claims a narrowing control per
clause, and `TASK-260824-3rl3ws_rework-validation.md` reports *"Seven narrowing
controls remove the clauses … each removal is detected"*, echoed in the board
note as *"Seven narrowing controls cover F1-F5 plus no-runtime-rewrite and
actionable-remediation clauses"*.

The control is:

```sh
if grep -Fv -- "$marker" "$contract" | grep -Fq -- "$marker"; then fail; fi
printf 'narrowing control detected removal: %s\n' "$marker"
```

`grep -Fv` drops every line containing the marker, so for any single-line
fixed-string marker the second `grep` can never match. Nothing is removed from
anything that is then re-checked; no mutant is built; the branch is dead.
Reproduced against the accepted artifact:

```
narrowing control detected removal: ABSENT MARKER NEVER WRITTEN   <-- marker never present, control still "passes"
narrowing control detected removal: Codex reasoning is deliberately open
```

A marker string that has never appeared in the contract produces the same
"detected removal" line as a real one. This is the fake-gate shape: an
attestation that reports success independent of the property it claims to
establish, presented as the evidence that F1-F5 are locked into the document.

What *is* load-bearing is `check_contract`'s `require_marker` presence checks —
those do fail on a real deletion (verified: stripping the
`Codex reasoning is deliberately open` line makes the presence check fail) — and
the `cmp -s` copies under `set -e`, which really do enforce byte-identity.

Required: either delete the narrowing loop and the claims that rest on it, or
make it a real mutant — write a copy of the contract with the clause removed,
run `check_contract` against that copy, and assert it exits non-zero. Then
correct `TASK-260824-3rl3ws_rework-validation.md` and the board note so the
recorded evidence matches what was actually executed.

## What is not being asked for

No rewrite, and no reopening of F1-F5 — the rework closed all five correctly and
the contract's factual anchors hold against the code. Sections 1, 2, 3, 4, 6, 7
and 8 stand. Fix the one §5 sentence (plus the optional §3.6 clarification),
replace or retract the narrowing-control claim, re-copy the contract byte-for-byte
to the three downstream preconditions, and this is acceptable.
