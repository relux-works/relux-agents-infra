# TASK-260830-s5ro4e review verdict — revision 5 (document revision 6)

Verdict: **changes requested**. Route to `analysis` for a corrected plan and a
new Change Request revision.

Reviewed candidate: `CR-TASK-260830-s5ro4e-5` revision 5, base
`5c9b4e4f7a88e1eb937b80851af522e4fa4b066f`, candidate tree
`b1bc6fd1ef94ace92ea21077ca81d9c9a608d8f2`, patch SHA-256
`fec721791ef587fac2ec7db682ddc7e29d92121b4b474d1641062c72499fa70e` (matches the
handed value), plan SHA-256 `083a602386b78ed8b6e5542b5aa4422e47e603d34cffd9a2b0c2bb9b28cddbaa`.
Reviewer run `RUN-260830-17a261`.

Revision 6 does what the last verdict asked: it stopped patching findings and
enumerated a class. I reproduced all five derivations against the real
production installers and they return exactly what the document claims. The
rejection is not a regression — it is that the enumeration missed a stage, and
that the mechanism the census leans on when a value *cannot* be derived is
itself defeatable on today's production text.

Production sources read at: `skill-project-management` `f1319eff`,
`relux-agents-infra` `bb857fe5`.

## Findings

### F1 — High: `install_go` is a second Homebrew mutation in both installers, before every managed binary, and the plan states there is only one

The plan opens its installer-boundary analysis with a factual claim
(`:210-212`):

> The agents-infra bootstrap has **one** optional mutation before either managed
> binary is replaced: `scripts/setup.sh:93-170,258-263` normally runs
> `install_lldb_mcp` …

There are two. Production `relux-agents-infra/scripts/setup.sh` defines
`install_go` at `:75-91` and calls it at `:258` — one line *before* the
`install_lldb_mcp` call at `:259` that the plan's cited range `258-263` begins
at. When `go` is not on the installer's `PATH` it runs `brew install go` on
Darwin, and `exit 1`s otherwise. The same stage exists in the task-board
installer (`skill-project-management/scripts/setup.sh:131-146`, called at
`:904`) with the same `brew install go` branch. Neither installer offers a skip
flag for it, so there is no `AGENTS_INFRA_SKIP_LLDB_MCP` equivalent to mandate.

Three consequences, each load-bearing:

1. **The LLDB guard cannot see it, and would authorize the step anyway.**
   `guarded_agents_infra_install` snapshots, runs
   `AGENTS_INFRA_SKIP_LLDB_MCP=1 "$release_source/scripts/setup.sh"`, snapshots
   again and `cmp`s. `install_go` executes *inside* that before/after window,
   at the very first top-level call. The plan states the guard's target set is
   "the mutation set of production `install_lldb_mcp`, read out of that function
   rather than chosen" (`:230-238`), and `snapshot_lldb_surface` accordingly
   records only `brew --prefix`, llvm's installed state and version, the llvm
   prefix, and four file paths — `$prefix/bin/lldb-mcp`,
   `$prefix/bin/lldb-mcp.agents-infra.bak`, `$llvm_prefix/bin/lldb-mcp`,
   `$llvm_prefix/bin/lldb`. A `brew install go` changes none of them. The
   `cmp` returns identical, `passed` is created, and the step proceeds — after
   an unmodelled Homebrew mutation inside the guard whose stated required
   outcome is "zero LLDB/Homebrew delta" (`:400-404`). The guard is present,
   called from the production step sequence, and admits what it must reject.

2. **On the task-board side it falls outside every named window by
   construction.** `derive_pm_setup_stages` — the plan's own instrument, which I
   ran against the real installer — places `install_go` at position 2 of
   sixteen, before all five `install_binary` calls. `W2` and `W5` are defined as
   opening "at the first `install_binary` call" (`:1523`, `:1731`, and the
   window table at `:1846`, `:1852`). That definition asserts nothing before it
   mutates the host. `install_go` falsifies the assertion, and the resulting
   mutation is in no window, no snapshot, and no restore.

3. **It received none of the four treatments its twin received.** For
   `install_lldb_mcp` the plan measured the target host, mandated a skip flag,
   built a before/after identity guard, and mirror-guarded the function. For
   `install_go` there is no measurement, no precondition, no guard and no
   mirror. The document mentions the name exactly once, in the derived stage
   list at `:573`. The stage sequence was derived — and then only
   `install_skills` was analysed out of it.

Measured host state, read-only, taken now — the measurement the plan does not
contain: `go` resolves to `/opt/homebrew/bin/go`, `go version go1.25.5
darwin/arm64`, and Homebrew reports formula `go` installed. The non-mutating
branch is selected on this host today. That is the same status the eight
`SKILL_REPOS` skills have, and revision 6 correctly treated *their* measurement
as a fact about one host at one time rather than a property of the plan. The
identical standard applies here.

The previous verdict already noted `install_go` in passing — "the plan does not
name it, but it selects the non-mutating branch here" — under the agents-infra
installer. Revision 6's thesis is that a fix applied to a finding is not a fix
applied to a class. This is the fifteenth site of the class, in the stage the
plan's own derivation surfaced.

Required rework:

1. Correct the "one optional mutation" claim, and cite `install_go` at both
   installers with its call position relative to the first managed-binary
   replacement.
2. Add a pre-window precondition on the same fail-closed construction as
   `require_dependency_skills_materialized`: `go` must already resolve to a real
   executable before either window opens, so the `brew install go` branch is
   unreachable inside them. A skip flag is not available, so the precondition is
   the only mechanism that removes the branch rather than guarding after it.
3. Either widen the before/after identity guard beyond the `install_lldb_mcp`
   target set so a Homebrew formula-set change is detected, or state explicitly
   that the guard covers only the LLDB target set and that Homebrew formula
   state is protected by the precondition instead. Do not leave the guard
   claiming a "zero LLDB/Homebrew delta" outcome it does not measure.
4. Mirror-guard `install_go` in Steps 2/3/5/6 alongside the functions already
   mirrored, so a release that changes what it touches stops the step.
5. State the in-flight behaviour and recovery authority for an interruption
   inside `install_go`, and for the `exit 1` branch when neither `go` nor `brew`
   is present.

### F2 — High: `require_mirrored_function_unchanged` compares a truncated prefix, and admits a changed `write_install_state` on today's production text

The census (`:451-476`) concedes four literal classes and argues each is safe
because it fails closed. For production-literal paths the argument is explicit:

> These are mirrored, and Steps 2 and 5 require `write_install_state`,
> `install_roles` and `register_skill` **byte-identical** between the saved
> source and the release source before the mirrored values are used. A release
> that moves one of them stops the step.

`require_mirrored_function_unchanged` does not compare the function. It compares
whatever `extract_function_body` returns, and that `awk` stops at the first line
that is exactly `}` (`:487-497`). Production `write_install_state`
(`skill-project-management/scripts/setup.sh:792-813`) writes its JSON with a
heredoc whose closing brace sits at column 0, four lines before the function
ends. Measured across all six mirrored functions:

| Function | Real lines | Extracted | |
| --- | ---: | ---: | --- |
| `install_skills` | 49 | 49 | |
| `write_install_state` | 22 | **18** | **truncated** |
| `install_roles` | 28 | 28 | |
| `register_skill` | 41 | 41 | |
| `resolve_config_dir` | 10 | 10 | |
| `install_lldb_mcp` | 79 | 79 | |

The dropped lines are the `EOF` terminator, a blank line, the
`green "Install state: $state_path"` call and the real closing brace.

Mutant, run against the real production file with the plan's own function
extracted verbatim from the document: take `write_install_state` and insert two
lines after the heredoc's `}` making the release additionally write a second
install state into `$HOME/.config/task-board-v2`. The files differ byte-wise.
The guard reports:

```text
MIRROR_UNCHANGED|write_install_state
bash exit=0
MIRROR_UNCHANGED|write_install_state
zsh exit=0
```

Narrowing controls, so this is not a delete-only claim: the same guard returns
`73` for a change *before* the truncation point (`config_dir` moved to
`$HOME/.config/task-board-MOVED`), and `73` for a tail mutation in
`install_skills`, which has no column-0 brace. The guard works on a prefix and
is silent about the remainder.

This matters beyond the one function. `extract_function_body` is also what
`derive_pm_setup_stages` uses to read `setup_main` (`:559`). Today's `setup_main`
has no column-0 brace inside it, so the sixteen-stage derivation is correct —
but the same truncation would silently shorten the derived stage list, which is
the document's definition of the `W2`/`W5` window. A defect that currently only
narrows a guard would then also narrow a named window, with nothing failing.

A related gap with the same root cause: `install_skills` resolves its
destinations through `AGENTS_SKILLS_DIR`, `CLAUDE_SKILLS_DIR` and
`CODEX_SKILLS_DIR`, which are **top-level assignments** at
`skill-project-management/scripts/setup.sh:816-818`, outside any function.
`require_mirrored_function_unchanged` only ever reads function bodies, so no
mirror covers those three assignments. `register_skill` happens to carry the
same three literals in its own locals, so a *coherent* relocation is caught by
that function's mirror; an incoherent one — moving the top-level variables only,
which changes exactly the eight-dependency-skill surface revision 6 was written
to close — is not.

Required rework: make the extractor bound the function correctly (track brace
depth, or terminate on the next top-level definition rather than the first
column-0 `}`), and prove it with a probe that is red on the current extractor
against real `write_install_state` text. Then either bring the three
`*_SKILLS_DIR` assignments under a guard of the same fail-closed shape, or state
plainly that the mirror covers function bodies only and name what that leaves
uncovered.

### F3 — Medium: `derive_setup_env_override_names` recognises one of four override shapes, so the classifier's fail-closed claim does not hold for the class it names

The plan states (`:900-906`):

> An environment override that a release adds and this plan has not classified
> makes `require_env_overrides_classified` exit 75. The step stops rather than
> running an installer whose behaviour is partly controlled by an input nobody
> decided about.

The derivation is `grep -oE '\$\{[A-Z_]+:-'`. It sees only that one shape.
Injected into a copy of the real production task-board installer, one shape at a
time, cumulatively:

| Injected into real production text | `require_env_overrides_classified` exit |
| --- | ---: |
| `if [ -n "$TASK_BOARD_FUTURE_FLAG" ]; then …` | 0 — admitted |
| `OTHER_FLAG=${OTHER_FLAG-0}` | 0 — admitted |
| `: ${THIRD_FLAG:=on}` | 0 — admitted |
| `FOURTH=${FOURTH_FLAG:-x}` (control) | 75 — refused |

The internal evidence is already visible in the plan: `classify_ai_env_override`
classifies `HOME` because agents-infra reads it as `${HOME:-}`, while the
task-board installer reads `$HOME` plainly to construct `BIN_DIR`,
`AGENTS_SKILLS_DIR` and every destination it writes — and `HOME` never appears
in the derived task-board list or in `classify_pm_env_override`. Two installers,
the same variable, opposite treatment, decided by punctuation rather than by
what the installer reads.

No current release is misclassified; both installers happen to use `${X:-}`
throughout for their real flags. The defect is that the guard's stated property
— a release that adds an override stops the step — holds for one shape and fails
open for the others, which is the class the guard exists to close.

Required rework: widen the derivation to the parameter-expansion forms plus bare
`$NAME`/`${NAME}` references, filter to names the installer does not itself
assign, and classify the resulting set. Prove it with a probe that is red on the
current derivation for at least the plain-`$VAR` shape.

## Confirmed independently — reproduced, not accepted from the producer

I extracted the fifteen derivation/guard functions verbatim from the plan
document (all `bash` code fences, function bodies sliced by definition line and
matching close) and ran them against the real production `scripts/setup.sh` of
both repositories. Both extractions parse clean under `bash` and `zsh`.

| Derivation | Result against production text | Matches the plan's claim |
| --- | --- | --- |
| `derive_pm_setup_stages` | 16 stages, `install_skills` at position 13, between `check_agents_infra_compose` and `verify` | yes |
| `derive_pm_binary_names` | `task-board tb-sessiond task-board-tui openai-board anthropic-board` | yes (5) |
| `derive_ai_binary_names` | `agents-infra model-harness` | yes (2) |
| `derive_pm_managed_skill_names` | `project-management` + all 8 `SKILL_REPOS` keys | yes (9) |
| `derive_pm_role_names` | 9 names, matching `.roles/*` | yes |

`require_dependency_skills_materialized` is genuinely correct and genuinely
narrowing. Driven against a disposable `HOME`: `0` with all nine as real
directories, `74` for a symlinked dependency skill, `74` for an absent one, `0`
again once re-materialised, `70` for an unreadable installer. The pass-after-
materialise row is the narrowing check — it refuses the clone branch without
refusing a legitimate installed skill.

I confirmed against production `install_skills` that the two branches are as the
plan describes, that a failed clone hits `continue` leaving the skill removed,
and that `setup_main` gates `check_agents_infra_compose`, `bootstrap_local_agents`
and `print_local_agents_hint` on `CONFIGURE_PROJECT == 1` while `install_skills`
and `verify` are unconditional — so the plan's account of which stages the bare
invocation reaches is accurate.

Other checks that passed:

- The working tree matches the candidate tree exactly (`git write-tree` returns
  `b1bc6fd1…`); the plan file is byte-identical to the board plan resource; the
  attached patch hashes to the handed value; `git diff --check` over the exact
  base/candidate range exits 0. The review did not drift the Change Request.
- The delta touches only `.research/…` and `LOGBOOK.md` — no Go, shell, module
  or configuration file. No release, tag, install, migration, Homebrew change or
  installed-runtime mutation is present in the reviewed delta.
- Every step, 0 through 6, states in-flight run behaviour, and Steps 2 and 5
  now state recovery authority for an interruption inside `install_skills`.
- Step 5 now carries a concrete rollback command sequence, which was the
  revision-1 F2 gap; Steps 2 and 5 carry the exact install invocation, which was
  the revision-4 F2 gap.
- Every operational rollback remains explicitly labelled **UNTESTED**, and the
  plan states that the sandbox evidence is not a substitute for an operational
  rehearsal. The Step-1 source repin remains **PARTIALLY TESTED**.
- The `LOGBOOK.md` correction has the right shape: entry `1815` is added, `1528`
  and `1529` carry explicit `SUPERSEDED by …` lines with reasons, and `1245`/`1428`
  carry the corrected 25/36 census, 20-package graph and 555 symbols. No false
  claim stands as a current entry, and the correction replaced rather than
  appended beside.
- The `v0.6.0` removal gate names specific methods, types and adapter call paths
  and requires `required_by_0.25.0=false` / `required_by_v1.7.0=false` per item,
  with `unknown`, failed read, partial inventory, surviving symbol or init-only
  reach all blocking. It is derived from invoked surfaces, not from absence of
  direct imports.

## What I did not re-derive, stated as unknown rather than inferred

- I did **not** rebuild task-board against public `v0.5.0` and did not re-derive
  the 555-symbol count, the 20-package graph or the
  `GOWORK=off go test -mod=mod ./internal/spawn -count=1` exit 0. That section
  is unchanged since document revision 3 and was independently reproduced by the
  revision-2 review; I accepted it from that prior review, not from the producer,
  and I make no independent claim about it in this verdict.
- I did not execute either installer, and the operational rollbacks remain
  unrehearsed. Nothing in this review changes their **UNTESTED** status.

## Review diagnostics

- My first attempt to measure true function length used an `awk` that scanned to
  the last column-0 `}` in the remainder of the file and produced nonsense
  offsets (`install_roles` "ending" at relative line 558). I treated the result
  as a broken instrument rather than a measurement, rewrote the bound to stop at
  the next top-level definition, and re-derived the table in F2 from that.
- A chained `bash -n && echo && zsh -n && echo` swallowed the second echo; I
  re-ran `zsh -n` alone and confirmed exit 0 rather than reporting the missing
  line as a syntax failure.

## Scope and safety

No release, tag, install, migration, Homebrew change or installed-runtime
mutation was performed by this review. Both installers were read, never
executed. All probes ran against copies under
`.temp/TASK-260830-s5ro4e-review5/` and a disposable `HOME` inside it. No
process belonging to another run was signalled. No credential, token, cookie or
keychain value was read, printed or persisted.

## Evidence

- Scratch: `.temp/TASK-260830-s5ro4e-review5/` — `fns.sh` (the fifteen functions
  extracted verbatim from the plan), `mut/` (the `write_install_state` mutant and
  its two narrowing controls), `sbx/` (disposable HOME for the dependency-skill
  probes).
- Production sources read at `skill-project-management` `f1319eff` and
  `relux-agents-infra` `bb857fe5`.
