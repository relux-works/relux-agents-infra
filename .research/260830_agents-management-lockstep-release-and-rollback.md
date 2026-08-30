# Agents-management lockstep release and rollback plan

Task: `TASK-260830-s5ro4e`

Date: 2026-08-30

Numbering: this document keeps its own revision counter, which runs one ahead
of the board's Change Request counter because the first document revision
predates the first CR. **Document revision 8 is `CR-TASK-260830-s5ro4e-7`**, and
the attached patch is named `..._change-request_rev7.patch` to match the board.
The `rev4 … / rev5 …` rows retained in the validation table keep the document
numbering they were run and reviewed under.

Review state: revised after `CR-TASK-260830-s5ro4e-6` changes requested. This
plan must receive a new independent acceptance before any breaking change,
release, installation, or tag in this sequence.

## Revision 7 is a change of form, not a retreat of scope

Revisions 3 through 6 answered each review by growing an executable harness
inside this document. By revision 6 that harness was twelve shell functions and
about 700 of the document's 1990 lines, and each review found new fail-open
defects in the **newly added** machinery rather than in the migration it was
written to protect:

| Revision | Harness defect the review found |
| ---: | --- |
| 3 | guards without `errexit` in an `if` context accepted failed reads |
| 4 | a hard-coded recovery baseline that had already gone stale |
| 5 | reader fail-opens: three failed `brew` calls produced a fabricated snapshot |
| 6 | `extract_function_body` stops at the first column-0 `}`, so the "byte-identical" mirror compared a truncated prefix and admitted a changed `write_install_state`; `derive_setup_env_override_names` recognised one of at least four override shapes |

Severity did not fall across those cycles. The harness had become unreviewed,
untested software living in a planning document, and its own correctness was a
larger risk than the migration.

**I agree with the orchestrator's call and revision 7 removes it.** The
argument for keeping it — that a program checks more consistently than a human —
is real but does not survive the evidence: this particular program has been
wrong at every audit, the operator cannot see what it actually compares, and
its failure mode is a green result rather than a stop. A human executing a
release can be relied on to run a `diff` and read it. They cannot be relied on
to audit an awk brace-matcher embedded in a markdown fence.

What revision 7 keeps, unchanged in substance:

- values are still **derived at execution time, never restated as literals**;
- every window is still enumerated with duration and impact;
- untested rollbacks are still labelled **UNTESTED**;
- `install_skills` is still modelled, not declared out of scope;
- exact install invocations are still given per step;
- the corrected consumption census is unchanged.

What changed: every guard function became a **precondition the operator
verifies with one named command and one stated expected result**. Where a
property genuinely wants automation, it is stated as a requirement and marked
**NOT IMPLEMENTED** rather than shipped as an untested guard.

Three of the replacements are strictly stronger than the code they replace:

- The mirror guard compared six named function bodies through a broken
  extractor. It is replaced by `diff -ru` over the whole installer tree, which
  cannot truncate, and which additionally covers the top-level
  `AGENTS_SKILLS_DIR`/`CLAUDE_SKILLS_DIR`/`CODEX_SKILLS_DIR` assignments and
  the `install_go` stage that no mirror covered (revision-6 F1 and F2).
- The override classifier recognised `${X:-}` only. It is replaced by a
  reference-set `diff` that catches `$X`, `${X}`, `${X:-}`, `${X-}` and
  `${X:=}` alike, because it matches the `$` sigil rather than an expansion
  operator (revision-6 F3).
- The LLDB identity guard snapshotted four file paths and the llvm formula. It
  is replaced by a snapshot of the **entire** Homebrew formula list, which is
  what makes a `brew install go` visible (revision-6 F1).

## Revision 9 — what the revision-8 review found, and what changed

Two findings, High and Low, and the High one invalidates a load-bearing claim
revision 8 introduced rather than a mechanism this document built. Severity keeps
falling and the subject keeps moving toward the migration domain, which is the
process working. It is also the fourth time this project has met the same defect
shape.

| Finding | What changed |
| --- | --- |
| F1 — the "daemon newer than CLI" refusal that Part I, P7 and R1 step 8 all rest on is **unreachable from every production `ConnectOrStart` call site**. The plan asserted it as fact | Part I's "Daemon newer than CLI" paragraph is rewritten to what production does: the newer branch returns an untyped `fmt.Errorf`, `ConnectOrStart`'s single `errors.As` discards it, the `launch == nil` arm is dead in this binary, and control reaches `takeOverUnresponsiveManager`, which runs no compatibility check and either returns an **unchecked live client** or **SIGTERM/SIGKILLs** the newer daemon — decided by its `startup_healthy`, which is unknown. "The point of irreversibility" keeps its conclusion and replaces its reason. P7's "Different" disposition and R1 step 8 widen from "do not run the spawn canary" to **"run no `ConnectOrStart`-class command against that board"**. Part V's `W2d` impact cell is corrected in the other direction too — in that window nothing fails; the impact is silence. Section **L** adds the reachability probes that should have accompanied I5-I7. The missing guard is recorded as a **named prerequisite for `skill-project-management`**, not written into this plan's prose. |
| F2 — Low: the attached validator's section J read revision 7 from `HEAD`, so it went red the moment revision 8 was committed and could not be reproduced from the shipped artifact | Section J now pins the revision-7 document by SHA (`0425fb7`) instead of reading `HEAD`, so the comparison is reproducible from any later checkout. |

**The shape, named once more.** *The check is present but uncalled from
production.* The engine-kind contract, the External-CI policy gate, the
comparison instrumentation, and now the wrapper's own protocol refusal — a guard
that exists, reads correctly, and never runs on the path that matters. Revision
8's section I asked whether the newer-daemon branch *exists* (I5-I7) and whether
the *older* takeover is *reached* (I9), and never crossed the two. That is the
whole defect, and it produced a plan that told an operator a wrong command would
be refused when in fact it proceeds.

**What the correction changes operationally.** The conclusion "the downgrade
direction has no rollback" survives. What does not survive is the implied safety
underneath it. Revision 8 said the restored CLI cannot touch that board, so the
state would sit still until somebody decided what to do. It will not sit still:
the next `ConnectOrStart`-class command from any agent or human on the host
either drives that daemon unchecked or kills it. So the mitigation is no longer
"the session plane is unavailable, avoid it" but an explicit prohibition on a
named command family, and a downgrade of such a board requires its owner to stop
the newer daemon **first**, out of band, with the session loss accepted in
advance.

**On convergence, asked directly by the review and answered directly.** Two
properties this plan depends on cannot be reduced by more writing: whether a
protocol-5 `tb-sessiond` populates `startup_healthy` (it decides which of F1's
two branches fires, and `0.25.0` does not exist yet), and what a terminated
daemon's provider children do. Both are settled only by building the release and
trying it on a disposable board — which this task is forbidden to do. They are
recorded with the other three such cases in **"The limit of what more writing can
fix"** at the end of this document, which revision 9 extends rather than
duplicates. Naming the limit is the result; a tenth revision that adds another
section without a rehearsal would be adding words, not confidence.

## Revision 8 — what the revision-7 review found, and what changed

Revision 7's form held. The harness is gone, the preconditions are commands an
operator can read, and the review reproduced every derivation rather than
accepting it. The three findings were about the **migration domain** rather than
about this document's own machinery, and severity fell again — High, Medium,
Low.

| Finding | What changed |
| --- | --- |
| F1 — `tb-sessiond` is a long-lived daemon carrying its own CLI-to-daemon protocol version; the plan modelled neither the daemon's lifecycle nor the skew a staged release creates, and R1 could not recover the downgrade direction | Part I gains a section that models the daemon and its control protocol from production text. **P7** enumerates the live daemons and derives `controlProtocolVersion` on both sides *before* the window. Part V names `W2d`, `W5d` and `Wimg`. R1 gains a daemon census and a canary that reaches the session manager instead of only a config reader. The downgrade direction is stated as **unrecoverable by any command this plan or the product provides**, with the exact point of irreversibility named. **Partly SUPERSEDED by revision 9 F1:** the conclusion stands, the stated reason did not — revision 8 believed a refusal protected the downgrade direction, and no production `ConnectOrStart` path performs one. |
| F2 — the snapshot records a `directory` kind for the `~/.claude` and `~/.codex` link trees, stages no tree for it, and R1 then aborts mid-rollback | The snapshot stages link-tree directories symmetrically with `~/.agents`, and **R1 gains a pre-mutation admissibility pass** over the whole snapshot — kinds, link targets, staged trees, saved executables, saved roles and the saved install state. A snapshot R1 cannot restore is refused before the first `mv`, not discovered halfway through. |
| F3 — two of six evidence sections described running the document's own commands when they ran a reimplementation | Section D is retitled as what it is, a fixture-level illustration, with the real P6 run pointed at section H. Section E now drives **the document's own P4 fence, verbatim**, including the manual append the document requires of the operator. Section G's prose matches how the suite actually locates the block. |

Working F1 produced a fact the finding did not have. **The control protocol has
been bumped four times in five days**: `df4201dc` (2026-07-24) introduced the v1
identifier, `445b391e` (2026-07-26) made it the integer 2, `f8ba3268`
(2026-07-28) 3, and `d66bfea8` (2026-07-28) 4. A bump between the installed pair
and `0.25.0` is the historical norm for this surface, not a hypothesis, which is
why P7 derives it rather than assuming either answer.

It also produced a second skew the finding did not name, and the host is already
in it. Even when the protocol does **not** change, a live daemon keeps executing
the task-board image it was started from — with that image's embedded
agents-management — for an unbounded time after the file on disk is replaced.
The two daemons live on this host right now report `0.24.3-112-ged48781` and
`0.24.3-124-g2a527a8` while the installed file is `0.24.3-172-g063197b1`: sixty
and forty-eight commits of drift, eight and five days old, holding 31 and 2
sessions. That window is `Wimg` in Part V, and it exists today, before this
migration starts.

## Decision

Use public `skill-agents-management v0.5.0` as the compatibility bridge. Release
task-board first, then agents-infra, while both compile exact `v0.5.0`. Permit
`v0.6.0` to remove a compatibility adapter only after exact released-source and
linked-binary inventories prove that neither released consumer needs that
adapter. Repin task-board first, then agents-infra, to the accepted immutable
`v0.6.0` tag.

The running programs do not dynamically load the Go module, but that fact alone
does not remove migration risk. `v0.5.0` compatibility calls reach the general
plugin graph and inference-engine code transitively. The safety argument is:

1. immutable module versions and checksums in each released binary;
2. unmodified compatibility/refusal tests against the exact public module;
3. an exact removal manifest proved against both released consumers;
4. out-of-place candidate canaries; and
5. cutover isolation with the previous pair kept as an absolute-path routing
   authority until the installed canary succeeds.

There is no planned *dynamic module-version* mismatch. There are four real
mixed-install windows, named below, during which the default installed surface
may contain artifacts from two releases. Those windows are not called safe or
zero-duration; new spawn/route work uses the saved old pair or waits for the
installed canary.

Reserved versions:

- `skill-agents-management v0.5.0` at
  `b74f758a90a422304d55460422831da80e3d6cc8`, checksum
  `h1:yy/j8YgKrXLWcb6zzoAld79a7iamTvOI5eZd/9MaAIk=` — already released.
- `skill-project-management 0.25.0` — first task-board release compiled with
  exact `v0.5.0`.
- `relux-agents-infra v1.7.0` — first agents-infra release compiled with exact
  `v0.5.0`.
- `skill-agents-management v0.6.0` — conditional breaking release.
- `skill-project-management 0.25.1` — task-board repin to exact `v0.6.0`.
- `relux-agents-infra v1.7.1` — agents-infra repin to exact `v0.6.0`.

If any reserved version is occupied before execution, stop and revise this
reviewed plan. Never move or replace a published tag.

## Corrected task-board consumption measurement

This section is unchanged since document revision 3 and was independently
reproduced by the revision-2 review.

Measurement source:

- clean detached Git worktree at exact task-board commit
  `063197b10e02cbacabfba4c192d16fa310f70eb7`;
- only `tools/board-cli/go.mod` and `go.sum` changed in scratch, pinning public
  `skill-agents-management v0.5.0`;
- `GOWORK=off`, `-mod=mod`, no agents-management `replace`, no workspace
  override;
- candidate binary built by Go 1.25.5 for Darwin arm64.

The four instruments answer different questions:

| Instrument | Exact result | What it establishes |
| --- | ---: | --- |
| Direct tracked-source import census | 25 production files, 36 test files, 13 package families | The written task-board source imports compatibility packages but directly imports neither `pkg/plugin` nor `pkg/inferenceengine`. The earlier 26/38 count did not reproduce and is not used as a gate. |
| `GOWORK=off go list -mod=mod -deps ./...` | 20 compiled agents-management packages | The composed `v0.5.0` graph includes `pkg/plugin`, `pkg/inferenceengine`, and `pkg/inferenceengine/engines/mlx` despite zero direct imports. |
| `go tool nm` on the candidate binary | 555 module symbols | 161 `pkg/agentic`, 32 `pkg/plugin`, 153 `pkg/vendorplugin`, 7 `pkg/inferenceengine`, 190 `pkg/providerlimits`; critical symbols include `plugin.(*Registry).RegisterAll`, `inferenceengine.init`, and `vendorplugin.productionEngineFactSource.ReadEngineFacts`. |
| Production-path inspection plus candidate preflight | real init/registration/build path invoked; preflight exit 0 | The compile/link reach is operational, not test-only or dead-source evidence. |

The 13 direct package families are `pkg/agentic`, six System packages (`agy`,
`claude`, `codex`, `gemini`, `muse`, `qwen`), `pkg/providerlimits`,
`pkg/vendorplugin`, and four vendor packages (`alibaba`, `anthropic`, `google`,
`openai`). Direct raw graph imports remain zero. That is true but not evidence
that the graph is unreachable.

The production path is unguarded by build tags:

1. `internal/spawn/launch_plan.go:157-164` calls `agentic.NewRegistry` and
   `(*agentic.Registry).Register`; `:332-342` dispatches through
   `agentic.BuildPlan`.
2. `pkg/agentic/registry.go:120-128,170-183` maps that unchanged Register call
   through `RegisterWithDependencies` into `plugin.Registry.Register`.
3. task-board blank-imports four vendor packages. Their package `init`
   functions call `vendorplugin.Register`; for example
   `pkg/vendorplugin/vendors/openai/openai.go:59-65`.
4. `pkg/vendorplugin/registry.go:252-337` validates models, syncs the agentic
   graph and registers the vendor graph node; `:363-388` copies every visible
   dependency-first graph node.
5. `internal/spawn/models.go:307-332` reads `vendorplugin.Default` during
   task-board package init. It declares the qwen-codex runtime and builds every
   registered model before any CLI command runs.
6. `pkg/vendorplugin/engine.go:9-11,31-54` imports and calls the concrete
   inference-engine contract. The linked binary contains that implementation
   and its refusal symbols.

The reviewer-required unmodified gate passes from the detached worktree:

```text
$ GOWORK=off go test -mod=mod ./internal/spawn -count=1
ok  github.com/aagrigore/task-board/internal/spawn  10.341s
exit 0
```

The earlier archive-only skip is withdrawn as gate evidence. The Git-history
guard ran normally in this real worktree.

## Exact compatibility-removal gate for `v0.6.0`

Absence of direct imports is not an adapter-removal gate. `v0.5.0` explicitly
labels these load-bearing bridges, all reached by the composed task-board:

- `agentic.Registry`, `NewRegistry`, `Register`,
  `RegisterWithDependencies`, `Lookup`, `Graph`, and `BuildPlan` plus the
  `System`, `LaunchRequest`, `LaunchMode`, and `Plan` shapes;
- `vendorplugin.Registry`, `Default`, `Register`, `DeclareRuntime`,
  `RuntimeDeclarationOf`, `Lookup`, `Graph`, and the `Vendor`, `Model`,
  `RuntimeDeclaration`, `BrokerProvenance`, pricing/lifecycle/effort shapes;
- the internal adapter calls those surfaces currently trigger:
  `plugin.Registry.Register`, graph topological sync, inference-engine init and
  `productionEngineFactSource.ReadEngineFacts`.

`0.25.0` and `v1.7.0` do not yet exist, so their exact final inventories cannot
be guessed in this plan. Before a `v0.6.0` release candidate may be approved,
the release owner must create a versioned removal manifest listing every
method, exported type/field, init registration, linked symbol family and
production call path proposed for removal. Then, for *both* exact released
consumer tags:

1. create clean detached worktrees of `0.25.0` and `v1.7.0`;
2. prove their `go.mod`/`go.sum` use public exact `v0.5.0`, with no
   agents-management replace or workspace override;
3. record direct imports, transitive packages and `go tool nm` symbols;
4. use Go type-check/build failures and production-entry tests to map every
   manifest item to a consumer call site or prove it unused;
5. repin scratch consumers to the exact `v0.6.0` candidate and run full tests,
   builds and real launch/compose canaries;
6. negatively narrow each removed adapter in a private candidate: both
   released consumers must remain green because they no longer require it;
   a delete-only result is insufficient;
7. require the manifest result `required_by_0.25.0=false` and
   `required_by_v1.7.0=false` for every item. `unknown`, failed read, partial
   inventory, surviving symbol, init-only reach or production-path reach blocks
   `v0.6.0`.

This gate covers wrappers and init paths. A zero direct-import count cannot
satisfy it.

## Compatibility matrix

| task-board | Embedded agents-management | agents-infra | Status |
| --- | --- | --- | --- |
| `0.24.3-172-g063197b1` | `v0.2.0` | `v1.6.1-103-g4270549` | Pair observed installed while planning; the recovery baseline is derived from the installed artifact at execution time, not from this row. |
| scratch `063197b1` | `v0.5.0` | unchanged | Composed spawn tests/build/preflight green; not released. |
| `0.25.0` | `v0.5.0` | old installed version | Required Step-2 transition; must pass installed canary. |
| `0.25.0` | `v0.5.0` | `v1.7.0` / `v0.5.0` | Stable compatibility-bridge pair. |
| `0.25.0` | `v0.5.0` | `v1.7.0` while `v0.6.0` exists | Still supported: immutable embedded pins. |
| `0.25.1` | `v0.6.0` | `v1.7.0` / `v0.5.0` | Required Step-5 transition; external compose seam must be canaried before cutover. |
| `0.25.1` | `v0.6.0` | `v1.7.1` / `v0.6.0` | Target pair after breaking cleanup. |

No release build may use an untagged module commit, mutable tag, `@latest`, an
agents-management `replace`, or a Go workspace override.

## Capabilities that must remain working

Every step requires an authoritative board path that can perform:

- spawn preflight and model/provider selection;
- run-record creation, child launch, observe/status/directives and terminal
  outcome handling;
- outcome resource add/update/read;
- producer handoff, reviewer spawn, reviewer verdict, `accept_cr`, rework
  routing and Story-workspace/Change-Request reads.

Goal-bound and writable-context spawn additionally requires a **usable session
plane**: `launchTrackedSpawnRun` routes those runs through the board's
`tb-sessiond`, so a board whose daemon speaks a protocol the CLI does not accept
has lost `spawn` even though every file on disk is correct. P7 and the R1 canary
cover that surface; a `project_config` read does not reach it.

During a mixed-install window, the default PATH surface is not authoritative.
The absolute saved old pair, with the recovery bin prepended to PATH, is:

```bash
PATH="$RECOVERY_ROOT/bin:$PATH" \
  "$RECOVERY_ROOT/bin/task-board" q --format compact \
  'project_config(view=spawn-preflight, role=developer, agent=codex, task_class=research)'
```

That same prefix makes a saved task-board resolve saved `tb-sessiond` and
saved `agents-infra`, not a partially installed sibling.

# Part I — What the two installers actually do

Both installers are `#!/usr/bin/env zsh`. Everything in this part was read from
production text on 2026-08-30 at `skill-project-management` `f1319eff` and
`relux-agents-infra` `bb857fe5`. **Nothing here is a comparison operand.** It is
what the operator should expect to see, so that a release which changed it is
visible as a difference rather than as a surprise mid-window.

## The agents-infra install sequence

`relux-agents-infra/scripts/setup.sh` is 271 lines and has no `setup_main`; its
stages are top-level calls. Read them with:

```bash
sed -n '255,271p' "$AI_RELEASE/scripts/setup.sh"
```

Observed at planning time:

```zsh
print ""
green "=== relux-agents-infra setup ==="
print ""
install_go
install_lldb_mcp
compute_ldflags
build_cli
install_binary
write_install_state
if [[ "$WITH_PDF_TOOLS" == "1" ]]; then
  green "Installing optional PDF toolchain"
  "$SOURCE_DIR/.scripts/setup-pdf-tools.sh"
fi
ensure_user_path
verify_install
print ""
green "=== Done ==="
```

**There are two Homebrew-capable stages before any managed binary is replaced,
not one.** Revision 6 stated one; that was wrong, and it is corrected here.

| Stage | Definition | Can mutate Homebrew | Suppressible |
| --- | --- | --- | --- |
| `install_go` | `scripts/setup.sh:75-91` | `brew install go` when `go` is not on `PATH` and the host is Darwin with `brew`; otherwise `exit 1` | **No flag exists.** Removed only by the P3 precondition below. |
| `install_lldb_mcp` | `scripts/setup.sh:93-171` | `brew install llvm`, then rewrites `$(brew --prefix)/bin/lldb-mcp` and may write `$(brew --prefix)/bin/lldb-mcp.agents-infra.bak` | `AGENTS_INFRA_SKIP_LLDB_MCP=1` returns at `:94-97` before any read |

`install_go` at `:75-91`, verbatim:

```zsh
install_go() {
  if command -v go >/dev/null 2>&1; then
    green "Go already installed: $(go version)"
    return
  fi

  if [[ "$(uname -s)" == "Darwin" ]] && command -v brew >/dev/null 2>&1; then
    yellow "Go not found. Installing via Homebrew..."
    brew install go
    green "Go installed: $(go version)"
    return
  fi

  red "Go is missing and automatic install is unavailable on this platform."
  red "Install Go manually and rerun setup."
  exit 1
}
```

The first branch is a pure read and returns before anything mutates. That is
the only acceptable branch inside a release window, and P3 makes it the only
reachable one.

`install_lldb_mcp`'s complete mutation set, read out of `:93-171`:
`brew install llvm`; `mkdir -p $(dirname "$brew_prefix/bin/lldb-mcp")`;
`cp "$wrapper" "$wrapper.agents-infra.bak"` when the wrapper exists and is not
a symlink and no backup exists; `rm -f "$wrapper"`; a heredoc rewrite of
`"$wrapper"`; `chmod +x "$wrapper"`. It also exits 1 if
`$(brew --prefix llvm)/bin/lldb-mcp` or `.../bin/lldb` is not executable.

## The task-board install sequence

`skill-project-management/scripts/setup.sh` is 935 lines and its stages are
`setup_main`, defined at `:891`. Read it with:

```bash
sed -n '891,930p' "$PM_RELEASE/scripts/setup.sh"
```

Observed at planning time:

```zsh
setup_main() {
  local parse_status=0
  parse_args "$@" || parse_status=$?
  if [[ "$parse_status" == "2" ]]; then
    return 0
  fi
  if [[ "$parse_status" != "0" ]]; then
    return "$parse_status"
  fi

  print ""
  green "=== project-management skill setup ==="
  print ""
  install_go
  build_cli
  build_sessiond
  build_tui
  install_binary "task-board" "$CLI_BINARY"
  install_binary "tb-sessiond" "$SESSIOND_BINARY"
  install_binary "task-board-tui" "$TUI_BINARY"
  install_binary "openai-board" "$BOARD_LAUNCHER_DIR/openai-board"
  install_binary "anthropic-board" "$BOARD_LAUNCHER_DIR/anthropic-board"
  register_skill
  install_roles
  write_install_state
  check_path
  if ! configure_project; then
    return 1
  fi
  if [[ "$CONFIGURE_PROJECT" == "1" ]]; then
    check_agents_infra_compose "$PROJECT_DIR"
  fi
  install_skills
  verify
  if [[ "$CONFIGURE_PROJECT" == "1" ]]; then
    bootstrap_local_agents
    print_local_agents_hint
  fi
  print ""
  green "=== Done ==="
}
```

The stage list is *readable directly*. Nothing needs to parse it. Three facts
follow by inspection, and they are what `W2` and `W5` are defined from:

1. `install_go` (`:131-146`, called at `:904`) is the **first** stage and runs
   before every `install_binary`. Its body is the same shape as the
   agents-infra one except that a missing `brew` is a hard `exit 1` rather than
   a platform check. There is no skip flag here either.
2. `check_agents_infra_compose`, `bootstrap_local_agents` and
   `print_local_agents_hint` are gated on `CONFIGURE_PROJECT == 1`, and
   `configure_project` (`:739-790`) returns immediately at its own head guard
   when it is `0`. The mandatory bare invocation in Steps 2 and 5 sets none of them.
3. `install_skills` (`:840-888`, called at `:923`) is **not** gated. It runs on
   every bare machine-scoped install, between `check_agents_infra_compose` and
   `verify`.

`install_binary` (`:195-210`) copies to `$BIN_DIR/$name.tmp.$$`,
`chmod +x`, `xattr -c`, then `mv -f` — a new-inode atomic replace, so there is
no window where the destination is missing.

`setup.sh` sources exactly one file at `:35`,
`scripts/lib/agents-infra-compose.zsh`. Confirm that is still true for a
release with:

```bash
grep -n '^source \|^\. ' "$PM_RELEASE/scripts/setup.sh"
```

Expected: one line, naming that file. More lines mean a larger execution
surface than P1/P2 were scoped against.

## What `install_skills` does, and to what

`install_skills` iterates the `SKILL_REPOS` map. For each entry it selects one
of two branches on `~/.agents/skills/<skill>`:

- **already-installed branch** — the path is a real directory and not a
  symlink. Nothing is cloned. `scrub_git_metadata` then removes `.git`,
  `.gitignore`, `.gitattributes` and `.gitmodules` from the installed tree if
  present, and both `~/.claude/skills/<skill>` and `~/.codex/skills/<skill>`
  are removed and recreated as symlinks (`rm -rf` when they are real
  directories).
- **clone branch** — the path is absent or a symlink. The symlink is removed
  first, then `git clone` runs against a remote GitHub/SSH URL. **A failed
  clone hits `continue`, which leaves the skill removed** and, on the next loop
  iteration for that skill, nothing to relink.

The clone branch reaches the network from inside the release window, installs an
unrelated skill at whatever its remote HEAD happens to be, and its failure mode
deletes an artifact. Neither belongs in a lockstep contract migration. P4
removes the branch before the window opens.

Two of the eight (`core-data`, `swiftui`) additionally get a root `SKILL.md`
symlink into their nested directory via the `SKILL_NESTED` map when the root
file is absent.

## `tb-sessiond` is a daemon, and its control protocol is part of this migration

`install_binary "tb-sessiond"` replaces a **file**. What that file becomes when
it runs is a long-lived, per-board daemon whose lifetime is independent of the
file's, which owns that board's provider hosts and session records, and which
speaks a **versioned control protocol** to the CLI. Revisions 1-7 of this plan
modelled the file and not the daemon. Everything below is read from production
text at `skill-project-management` `f1319eff`.

### The contract surface

- `tools/board-cli/internal/sessionmanager/types.go:11-13` defines
  `controlProtocolName = "task-board-session-manager"` and
  `controlProtocolVersion = 4`. It is a compile-time constant, so a CLI's
  supported version is a property of the **build**, not of anything the binary
  prints: `--version` does not report it, and it can only be derived from the
  source tree that build came from. That is why P7 reads it out of
  `$SAVED_SOURCE` and `$PM_RELEASE` rather than out of an installed artifact.
- The daemon reports its own version over its status endpoint, as
  `protocol` (`task-board-session-manager-v4`) and `protocol_version` (`4`),
  alongside `daemon_version`, `pid`, `board_fingerprint` and `session_count`
  (`types.go:114-128`).
- The daemon executable is resolved **through `PATH`**, not from a path beside
  the CLI: `cmd/session.go:26,480-484` looks up the bare name `tb-sessiond`
  with `exec.LookPath`. This is what makes the recovery-PATH prefix used
  throughout this plan work — a saved CLI launched with
  `PATH="$RECOVERY_ROOT/bin:$PATH"` starts the **saved** daemon. It is also why
  a saved CLI run *without* that prefix would start the **release's** daemon.

### Two ways the CLI reaches a live daemon, and only one of them restarts it

| Entry point | Production call sites | Compatibility check | May restart the daemon |
| --- | --- | --- | --- |
| `sessionmanager.Dial` (`client.go:70`) | `cmd/session.go:42,75,100` (`session status`, `list`, `stop`), `cmd/spawn_goal.go:283`, `cmd/primary_goal.go:356`, `cmd/goal_reflect.go:26` | **none** | no — it never launches one |
| `sessionmanager.ConnectOrStart` (`client.go:128`) and `ConnectOrStartAndAttach` (`client.go:228`), both through `connectOrStartSessionManager` (`cmd/session.go:360-372`) | `cmd/session.go:151,199,245` (`logs`, `doctor`, `reclaim-contexts`), `cmd/session_context_economics.go:58`, `cmd/session_context_retention.go:50`, `cmd/spawn_context_lifecycle.go:292`, `cmd/codex_goal_spawn.go:43`, `cmd/codex_manager.go:92`, `cmd/claude_manager.go:115` | **yes** | **yes** |

**Spawn is in the second row.** `launchTrackedSpawnRun`
(`cmd/codex_goal_spawn.go:50-56`) sends every goal-bound or writable-context
Codex/Claude run through `launchManagedGoalBoundRun` (`:81`), which reaches the
daemon through `connectManagedSpawnSession` (`:35-44`) and therefore through
`ConnectOrStart`. The capability this plan says must keep working is the
capability that triggers the restart.

### What happens in each direction

**CLI newer than daemon** — `actual < controlProtocolVersion`
(`client.go:719-724`) returns `ManagerProtocolUpgradeError`. `ConnectOrStart`
does not surface it to the operator: at `client.go:149-163` it calls
`takeOverOutdatedManager` (`stale_takeover.go:71-79`), which fences the running
daemon and replaces it. The takeover first asks the old daemon to `Drain`
(`stale_takeover.go:139`, `client.go:340-346`), which releases only the manager
transport and **deliberately leaves durable provider hosts running**; if the
daemon predates that endpoint or does not release its instance lock within the
grace period, it falls through to a fenced `terminateStaleInstance`
(`stale_takeover.go:162`), i.e. SIGTERM then SIGKILL. Both live daemons on this
host are v4 and can drain, so on this host the automatic path is the gentler
one — but that is a measurement of two processes today, not a property of the
step.

**Daemon newer than CLI** — this is the direction revision 8 got wrong, and the
correction changes the shape of the risk rather than its size.

The refusal text exists. `actual > controlProtocolVersion` (`client.go:725-731`)
returns *"Session Manager protocol v%d is newer than this wrapper supports
(v%d); update task-board before retrying"*. Revisions 1-8 asserted that an older
CLI therefore **cannot** use that board's session plane. That assertion is
false for every `ConnectOrStart` call site in this binary, and `ConnectOrStart`
is the family this plan cares about — goal-bound `spawn`, `session doctor`,
`logs`, `reclaim-contexts`, the managed `codex`/`claude` wrappers.

The reason is that the refusal is returned as a **plain `fmt.Errorf`**, while the
older-daemon branch one `if` above it returns the typed
`*ManagerProtocolUpgradeError`. `ConnectOrStart` inspects the error with exactly
one `errors.As(err, &upgrade)` against that typed value (`client.go:145-166`).
For the newer-daemon error the match fails, the `else` block falls through
without returning, and **the error is discarded**. The only arm that would
surface it is `if launch == nil { return nil, ErrManagerNotRunning }` — which
does not surface it either, and is unreachable regardless: all three production
callers pass a non-nil launch closure (`cmd/session.go:366`,
`cmd/codex_manager.go:92`, `cmd/claude_manager.go:115`). Control therefore
reaches `takeOverUnresponsiveManager(ctx, layout)`
(`stale_takeover.go:168-266`), which dials with `dialManagerStatus`
(`stale_takeover.go:25-41`) — the **unchecked** variant. The compatibility check
lives only in `dialHealthyManager` (`:43-59`), which the takeover never calls.
So the older CLI meets the newer daemon with no compatibility check at all, and
takes one of three branches:

| Live newer daemon's state | What the older CLI does | Operator sees |
| --- | --- | --- |
| holds the singleton lock, second status round-trip reports `startup_healthy: true` | `takeOverUnresponsiveManager` returns that client; `ConnectOrStart` returns it unchanged | **success.** A live client to a daemon speaking a protocol this build does not support, and v4 control calls are then issued against it |
| holds the lock, `startup_healthy: false`, past the 3 s `startupHealthyDeadline` from its recorded `StartedAt` | falls through the recorded/alive/instance re-checks to `terminateStaleInstance` (`:265`) → SIGTERM, 6 s grace, then SIGKILL — then `launch()` starts a daemon from whatever `tb-sessiond` `PATH` resolves to | **success**, after the newer daemon and its sessions were killed out of band |
| does not hold the lock | `(nil, nil, nil)`; `launch()` runs and the poll loop re-dials through `dialHealthyManager`, which *does* check compatibility and keeps failing | `waiting for session manager: context deadline exceeded` after 15 s (`session`) or up to 4 min (managed wrappers). The message names no protocol |

Both of the first two contradict what revisions 1-8 wrote. The first is worse in
kind: the plan told the operator the session plane was *lost*, so a wrong command
would produce a clean refusal. It does not; it proceeds. The second is worse in
blast radius: it is exactly the **out-of-band termination** this plan's own NOT
IMPLEMENTED table calls *"the only remaining move … its blast radius decides
whether that state is recoverable at all … UNKNOWN"* — arriving automatically,
on a daemon that may hold dozens of sessions, instead of as a decision somebody
made.

Which branch fires is decided by one field, `StartupHealthy` / `startup_healthy`
(`types.go:114-128`). Whether a protocol-5 daemon still populates it is
**unknown**: the status shape is precisely what a protocol bump changes, and the
decoder is lenient (`client.go:677`, a plain `json.Decoder.Decode` with no
`DisallowUnknownFields`), so a renamed or dropped field decodes as the zero value
`false` and selects the terminate branch. Stated as unknown, not predicted.

**Where the refusal *is* reachable, and why neither place helps.** Two production
paths do run the check against a newer daemon:

- `sessionmanager.Connect` (`client.go:112`) is the launch-less, compat-checked
  entry. It has zero direct call sites under `cmd/`; it reaches production once,
  as a function value at `cmd/context_security.go:75`, invoked inside
  `FinalizeAndRequireBuilderCoverageEvidence`
  (`builder_gateway_manager.go:319-352`) on the `task-board context publish`
  path. There the refusal is not `ErrManagerNotRunning`, so it takes the
  `default:` arm and surfaces as a `ContextError` with code
  `builder_trace_unavailable` — a message about an unreadable trace, not about a
  protocol.
- `ConnectOrStartAndAttach` (`client.go:228-250`) — but only *after*
  `ConnectOrStart` has already handed back the unchecked client and `Attach` was
  rejected by the newer daemon with an unknown-field 400 (`client.go:669-671`).
  Only then does `takeOverManagerForProtocolUpgrade(ctx, layout, true)` re-dial,
  run `managerProtocolCompatibility`, fail its `errors.As`, and return the
  refusal — wrapped in *"automatic Session Manager protocol upgrade failed after
  a rejected request; retry the wrapper"*, advice that cannot succeed. Whether a
  v5 daemon would reject a v4 attach body at all is unknown; an older client
  sends fewer fields, so it may simply be accepted.

Neither path is a spawn or route command, and neither prevents the unchecked
client from being issued first. **The downgrade direction is unprotected, not
protected-by-refusal.**

`takeOverOutdatedManager` does still refuse explicitly at
`stale_takeover.go:76-78` — *"refusing protocol takeover without an older daemon
version"* — and no CLI subcommand exposes `Drain`, so no operator can drain a
newer daemon by hand. Both remain true. What they do **not** establish, and what
revision 8 read them as establishing, is that anything stops the older CLI from
driving or killing that daemon. `session status` still answers, because `Dial`
performs no compatibility check, but that is a **read**, not a repair.

**A named prerequisite for the owning repository, not for this plan.** If the
`ConnectOrStart` path is to fail closed in the downgrade direction, that is a
change to `skill-project-management`: either `managerProtocolCompatibility`
returns a typed error for the newer branch too and `ConnectOrStart` surfaces it,
or `takeOverUnresponsiveManager` runs a compatibility check before returning a
client or signalling a PID. This plan cannot add behaviour to `tb-sessiond` or
to the wrapper; it can only describe what those binaries do today. The gap is
recorded in the NOT IMPLEMENTED table and, until it is closed, the plan's
response is prevention in P7 and a prohibition in R1 — not a refusal it does not
have.

### Why a bump is likely rather than hypothetical

The constant's whole history, from `git log -L` on its own line:

| Commit | Date | Value |
| --- | --- | --- |
| `df4201dc` | 2026-07-24 | `"task-board-session-manager-v1"` |
| `445b391e` | 2026-07-26 | `2` |
| `f8ba3268` | 2026-07-28 | `3` |
| `d66bfea8` | 2026-07-28 | `4` |

Four values in five days. This plan therefore treats "does `0.25.0` change
`controlProtocolVersion`?" as a question P7 must answer from the release tree,
and a change as a condition that stops the step for a decision — not as
something to discover when a daemon is fenced mid-window.

### The point of irreversibility

If `0.25.0` bumps the constant, the harm is not done by the install. It is done
by **the first `ConnectOrStart`-class command the newly installed CLI runs
against a board whose daemon is older** — a `task-board spawn` with a goal, a
`session doctor`, a managed `codex`/`claude` wrapper. At that moment that board
gets a daemon speaking the new protocol. From then on:

- R1 restores every file. The restored CLI then meets a daemon newer than it
  supports and, on every `ConnectOrStart` path, **does not refuse it**: it
  either receives an unchecked live client and issues v4 control calls against a
  v5 daemon, or it SIGTERM/SIGKILLs that daemon and starts a replacement. Which
  one happens is decided by the newer daemon's `startup_healthy`, a field this
  plan cannot predict.
- The plan does not get to choose between those two. There is no flag, no
  environment variable and no subcommand that selects one, and no operator
  action between the install and the first command that changes it.
- `takeOverOutdatedManager` refuses the downgrade explicitly, and no CLI
  subcommand reaches `Drain`, so nothing in the older build performs a *graceful*
  replacement. What it does instead is a fenced termination, and a termination is
  **not** a drain. What that does to the daemon's provider children is stated as
  **UNKNOWN** in the NOT IMPLEMENTED table, and it stays unknown here.

**The conclusion "no rollback" is unchanged; the reason is different, and the
difference matters operationally.** Revisions 1-8 said the restored CLI cannot
touch that daemon, which implied a wrong command would produce a clean refusal
and leave the daemon intact. It will not. The restored CLI will drive that
daemon or fence it, without a compatibility check, and neither outcome is a
rollback. So the instruction is not "the session plane is unavailable, avoid it
and it will keep" but **"run no `ConnectOrStart`-class command against that
board at all"** — because running one is what causes the next irreversible thing,
and its blast radius is UNKNOWN.

A downgrade of that board therefore requires stopping the newer daemon
**first**, by its owner, out of band, with the session loss accounted for
beforehand — not relying on a refusal that this binary does not perform. That is
a decision for the board's owner, and this plan has no authority to make it.

The response is prevention, in P7: either the release does not change the
constant, or every live daemon is quiesced and stopped by its owner **before**
Step 2 opens. Prevention was already the right answer; it is now the *only*
answer, because the fallback revision 8 believed in — a refusal that keeps the
state frozen until somebody decides — does not exist.

### Image skew, which exists even when the protocol does not change

An unchanged `controlProtocolVersion` makes the step *protocol*-transparent. It
does not make it *image*-transparent. A daemon started before the install keeps
running its own task-board image, with that image's embedded
agents-management, and there is no mechanism that replaces it until it is
restarted. Measured on this host on 2026-08-30, read-only, through
`task-board --board-dir <board> --json session status`:

| Board | PID | `daemon_version` | `protocol_version` | Sessions | Started |
| --- | ---: | --- | ---: | ---: | --- |
| `~/src/mac-infra/.task-board` | 42553 | `0.24.3-112-ged48781` | 4 | 31 | 2026-08-22 |
| `~/src/casual-talks/.task-board` | 61270 | `0.24.3-124-g2a527a8` | 4 | 2 | 2026-08-25 |
| `~/src/relux-works/relux-agents-infra/.task-board` (this migration's board) | — | not running | — | 0 | — |

The installed file is `0.24.3-172-g063197b1`. Two daemons are therefore already
sixty and forty-eight commits behind the binary that would be replaced, and have
been for eight and five days. That is the honest shape of the surface this plan
sequences, and it has two consequences the earlier revisions got wrong:

1. **The instruction "do not restart or kill them" is not the operator's to
   give for `tb-sessiond`.** For provider children and already-exec'd CLI
   processes it is correct. For a daemon whose protocol is older than the
   installed CLI's, the CLI restarts it automatically on the next
   `ConnectOrStart`, whatever this document says.
2. **The migration's own board has no daemon.** An installed canary run here
   meets no session manager at all, so it says nothing about the two boards that
   do have one. That is why P7 is a census of the host, not a check on one
   board.

## Measured host state of the surfaces the installers touch

Read-only, taken 2026-08-30 on the target host. Every row is a completed read;
a failed read would be recorded as `unknown` and is not present. **These are
observations of one host at one time, not properties of the plan, and not
comparison operands.** The preconditions below re-measure them at execution
time.

| Surface | Measured | Consequence |
| --- | --- | --- |
| `command -v go` | `/opt/homebrew/bin/go` | `install_go`'s non-mutating first branch is selected today in both installers |
| `go version` | `go1.25.5 darwin/arm64` | |
| Homebrew formula `go` | installed | |
| Homebrew formula count | 228 | The P6 snapshot compares this whole list, so any `brew install` inside a window is visible |
| `brew list --versions llvm` | `llvm 23.1.0` | `install_lldb_mcp`'s mutating branch is *reachable*; absence is not assumed |
| `brew --prefix llvm` | `/opt/homebrew/opt/llvm` | The prefix read succeeding is what makes the next two rows facts rather than artifacts |
| `/opt/homebrew/bin/lldb-mcp` | present, 538 bytes, `-rwxr-xr-x` | |
| `/opt/homebrew/bin/lldb-mcp.agents-infra.bak` | present, 123 bytes | Written by a previous `install_lldb_mcp` run; part of its mutation set |
| `/opt/homebrew/opt/llvm/bin/lldb-mcp` | **absent** | Pre-existing baseline anomaly. `install_lldb_mcp` would `exit 1` here |
| `/opt/homebrew/opt/llvm/bin/lldb` | **absent** | Same |
| Installed `task-board` / `tb-sessiond` | `0.24.3-172-g063197b1 (commit 063197b1)` | Observation only; Step 0 derives the baseline from the artifact |
| Installed `agents-infra` | `v1.6.1-103-g4270549 commit=4270549` | Same |
| `pgrep -fl tb-sessiond` | 2 live daemons, pids 42553 and 61270, both launched from `~/.local/bin/tb-sessiond` | The file Step 2 replaces is the file both running daemons came from |
| Their `protocol_version` | 4 and 4 | Equal to the installed build's `controlProtocolVersion`, so no skew today |
| Their `daemon_version` | `0.24.3-112-ged48781`, `0.24.3-124-g2a527a8` | **Both differ from the installed file's `0.24.3-172-g063197b1`**: image skew already exists |
| Their `session_count` | 31 and 2 | What an automatic protocol takeover would drain or fence |
| This migration's board | `running:false`, exit 0 | A legitimate absence, not a failed read — and the reason an installed canary here cannot observe the other two |
| `~/.roles` | 9 entries | |
| The eight `SKILL_REPOS` trees | all real directories, no git metadata, both link trees symlinks to the agents copy | The destructive clone branch is not selected today |

The LLDB anomaly is pre-existing, not a migration result. The lockstep steps
preserve the exact presence/absence shape and do **not** claim to repair LLDB.

# Part II — Preconditions the operator verifies

Each precondition is one command with a stated expected result. The operator
runs it, reads the output, and stops if it does not match. There is no code in
this document that decides on the operator's behalf.

A rule that applies to all of them: **the comparison operand is never a value
written in this document.** It is always produced at execution time, either
from the saved installed pair's source or from the host. Recorded outputs above
are dated observations, so nothing here goes stale when a repository moves.

## The derived-versus-restated census, carried forward

Revision 6 enumerated fourteen places where this document had restated a set
that something else owns, and derived all of them. That property is preserved
under the new form — the difference is that the derivation is now a command the
operator runs and reads, not a function this document defines. The census is
kept so the property is checkable rather than asserted.

| # | Place | Was stated as | Derived in revision 7 by |
| ---: | --- | --- | --- |
| 1 | Snapshot managed-executable loop | Seven literal names | the two `sed -nE` commands in P5, run against the release |
| 2 | Snapshot role loop | Nine literal role names | `ls -1A "$HOME/.roles"` into `roles.txt` |
| 3 | Rollback role loop | The same nine names | `roles.txt` plus `ls -1A "$PM_RELEASE/.roles"` for the removal branch |
| 4-6 | Step-2/Step-5/agents-infra executable restore | Literal `restore_binary` calls | R1/R2 loop over `binaries.txt` and the release's own `sed` output |
| 7 | `PM_REPO`, `AI_REPO` | Absolute host paths | `repoPath` in each installer's machine-scoped install state |
| 8 | Install directory | `$HOME/.local/bin` | `binDir` from both install states, required equal |
| 9 | `W2`/`W5` stage sequence | Prose ending at "roles and install state" | `sed -n '/^setup_main() {/,/^}/p'` on the release, saved to `w2-setup-main.txt` / `w5-setup-main.txt` |
| 10 | The managed skill set | Absent | `register_skill`'s `skill_name` plus every `SKILL_REPOS` key, read from the release |
| 11 | The `~/.claude` / `~/.codex` link set | Two literal links | the same derived skill set, each entry's observed kind and target recorded in `skill-state.txt` |
| 12 | agents-infra install state | Absent — never snapshotted | snapshotted and restored by R2; the config-dir rule is quoted from `resolve_config_dir` and covered by P1 |
| 13 | LLDB guard target set | Five LLDB-specific values | P6 records the **whole** `brew list --formula` output plus the four target paths |
| 14 | The installers' environment-override surface | Absent | P2's sigil-matched reference set, diffed against the saved source |

Revision 7 adds a fifteenth site the census did not have: the `W3`/`W6` stage
sequence, which revision 6 described in prose. It is now `sed -n '/^install_go$/,$p'`
on the release, saved to `w3-call-sequence.txt`.

Revision 8 adds two more, both from the revision-7 review:

| # | Place | Was stated as | Derived in revision 8 by |
| ---: | --- | --- | --- |
| 16 | The link trees' restorable content | A recorded kind with nothing staged for it | the snapshot stages `~/.claude`/`~/.codex` directory entries into `pm/claude-skills` and `pm/codex-skills`, and both the snapshot and R1 assert that every recorded kind has what its restore needs |
| 17 | The CLI-to-daemon control protocol | Absent — the daemon was modelled as a file | `grep -n 'controlProtocolVersion *=' …/sessionmanager/types.go` against `$SAVED_SOURCE` and the release, plus `pgrep -fl tb-sessiond` and a per-board `session status --json` read (P7) |

### What is still a literal, and why that is safe

Four kinds of value remain written here. None can drift silently.

- **Release identities** — `0.25.0`, `v1.7.0`, `v0.5.0`, `v0.6.0`, `0.25.1`,
  `v1.7.1`. These are the decision this plan exists to make, not a set derived
  from something else, and each is asserted with `git describe --exact-match`
  before use.
- **Paths this plan owns** — `$HOME/.local/state/TASK-260830-s5ro4e/...`. No
  production code reads them.
- **Paths production writes as its own literals** — `$HOME/.config/task-board`,
  `$HOME/.roles`, `$HOME/.agents/skills`, `$HOME/.claude/skills`,
  `$HOME/.codex/skills`, and the Darwin `~/Library/Application Support/agents-infra`.
  These are quoted from production text and covered by P1: a release that moves
  one of them produces a non-empty `diff -ru`, which stops the step.
- **The observed outputs recorded in Part I and Part II.** These are dated
  measurements, explicitly *not* comparison operands. Every check compares two
  live values.

The distinction that matters is fail-open versus fail-closed. A restated set
that a release extends fails open: the extra role, skill or executable is
installed and never rolled back, and nothing reports it. A value compared
against a live saved source fails closed: both operands exist when the step
runs, so there is no expected digest in this document to go stale.

## P1 — the installer text is unchanged from the source this plan analysed

Replaces `require_mirrored_function_unchanged` and every `extract_function_body`
call. The revision-6 verdict showed that extractor compared a truncated prefix,
admitted a changed `write_install_state`, and covered no top-level assignment at
all. A whole-tree diff has none of those failure modes.

```bash
diff -ru "$SAVED_SOURCE/scripts" "$RELEASE/scripts"; echo "exit=$?"
```

**Expected: no output, `exit=0`.**

If the output is non-empty, the operator reads the diff and answers, in
writing, before the step may proceed:

- does it change `setup_main` or the top-level call sequence — i.e. the stage
  list `W2`/`W5`/`W3`/`W6` are defined from?
- does it change `install_go`, `install_lldb_mcp` or `install_skills` — the
  three stages this plan models by behaviour?
- does it change `AGENTS_SKILLS_DIR`, `CLAUDE_SKILLS_DIR`, `CODEX_SKILLS_DIR`
  (`skill-project-management/scripts/setup.sh:816-818`), `BIN_DIR` (`:16`), or
  the literal paths in `register_skill`, `install_roles` and
  `write_install_state` — i.e. any destination the snapshot and rollback aim at?
- does it add a `source` line, adding files P1 and P2 were not scoped over?

**Any "yes" stops the step.** Part I is then re-derived against the new text and
this plan gets a new review before the release proceeds. A release that changes
the installer's boundary needs a plan that describes the new boundary; it does
not need a cleverer parser.

`$SAVED_SOURCE` is the recovery worktree created in Step 0 at the commit the
*installed* binary reports — so both operands are live trees, and there is no
digest here to maintain.

## P2 — the installer reads no environment variable this plan has not decided

Replaces `derive_setup_env_override_names`, `classify_*_env_override` and
`require_env_overrides_classified`. The old derivation matched `${X:-` only, so
`$X`, `${X-}`, `${X:=}` and `: ${X:=}` all passed unnoticed — which is the class
the guard existed to close. This command matches the `$` sigil instead, so it
sees every expansion shape.

For task-board, over its exact execution surface (`setup.sh` plus the one file
it sources):

```bash
grep -ohE '\$\{?[A-Z][A-Z0-9_]*' \
  "$RELEASE/scripts/setup.sh" \
  "$RELEASE/scripts/lib/agents-infra-compose.zsh" \
  | sed -E 's/^\$\{?//' | sort -u
```

Observed at planning time — 38 names:

```text
AGENTS_INFRA_COMPOSE_CONTRACT AGENTS_INFRA_COMPOSE_SCHEMA_VERSION
AGENTS_SKILLS_DIR BIN_DIR BOARD_LAUNCHER_DIR BOARD_MODE
BOOTSTRAP_LOCAL_AGENTS CLAUDE_SKILLS_DIR CLI_BINARY CLI_DIR CODEX_SKILLS_DIR
CONFIGURE_PROJECT HOME LOCAL_BOARD_DIR PATH PROJECT_DIR PROJECT_OPTION_USED
PWD REMOTE_BOARD REMOTE_INSECURE REMOTE_URL SESSIOND_BINARY SIDECAR_CONFIG
SKILL_DIR SKILL_NESTED TASK_BOARD_BOOTSTRAP_LOCAL_AGENTS
TASK_BOARD_CONFIGURE_PROJECT TASK_BOARD_DIR TASK_BOARD_ID TASK_BOARD_MODE
TASK_BOARD_PROJECT_DIR TASK_BOARD_REMOTE TASK_BOARD_SETUP_SOURCE_ONLY
TASK_BOARD_TLS_NO_VERIFY TASK_BOARD_USER TMPDIR TUI_BINARY TUI_DIR
```

For agents-infra:

```bash
grep -ohE '\$\{?[A-Z][A-Z0-9_]*' "$RELEASE/scripts/setup.sh" \
  | sed -E 's/^\$\{?//' | sort -u
```

Observed at planning time — 19 names:

```text
AGENTS_INFRA_CONFIG_DIR AGENTS_INFRA_SKIP_LLDB_MCP BINARY_NAME BIN_DIR
BUILD_COMMIT BUILD_DATE BUILD_LDFLAGS BUILD_OUTPUT BUILD_VERSION CONFIG_DIR
HOME INSTALL_STATE_PATH LLDB_MCP_WRAPPER_MARKER MODEL_HARNESS_BINARY_NAME
MODEL_HARNESS_BUILD_OUTPUT PATH SOURCE_DIR WITH_PDF_TOOLS XDG_CONFIG_HOME
```

**The check is a diff against the saved source, not against those blocks:**

```bash
diff <(grep -ohE '\$\{?[A-Z][A-Z0-9_]*' "$SAVED_SOURCE/scripts/setup.sh" ... \
        | sed -E 's/^\$\{?//' | sort -u) \
     <(grep -ohE '\$\{?[A-Z][A-Z0-9_]*' "$RELEASE/scripts/setup.sh" ... \
        | sed -E 's/^\$\{?//' | sort -u)
```

**Expected: no output.** A name only in the release is an input nobody decided
about, and the step stops until this plan classifies it.

The set is deliberately unfiltered — it contains the installer's own internal
variables as well as external inputs. Filtering "names the script assigns
itself" is exactly where the old classifier failed: `BIN_DIR` is assigned at
`relux-agents-infra/scripts/setup.sh:10` as `BIN_DIR="${BIN_DIR:-$HOME/.local/bin}"`,
which is *both* a self-assignment and an external override, and
`AGENTS_INFRA_SKIP_LLDB_MCP=1` appears inside the usage text at `:38`, which a
naive assignment filter reads as a self-assignment. An unfiltered set diffed
against the saved source needs no such judgement.

Known limits of the sigil match, stated rather than hidden: it does not see
`printenv NAME`, `eval`, `${(P)var}`, `${!var}` or `[[ -v NAME ]]`. Probe for
those separately:

```bash
grep -nE 'printenv|(^|[^_[:alnum:]])eval([^_[:alnum:]]|$)|\$\{\(P\)|\$\{!|\[\[ -v ' \
  "$RELEASE/scripts/setup.sh" "$RELEASE/scripts/lib/"*.zsh
```

**Expected: no output** (exit 1 from `grep` finding nothing). Observed empty on
both installers at planning time. A match means the environment surface is no
longer readable by inspection and the step stops.

The decided disposition for each external name is in the install invocation
itself: Steps 2 and 5 `env -u` every `TASK_BOARD_*` name, and Steps 3 and 6 set
`AGENTS_INFRA_SKIP_LLDB_MCP=1` and leave `BIN_DIR` and `AGENTS_INFRA_CONFIG_DIR`
unset. `HOME`, `PATH`, `PWD`, `TMPDIR` and `XDG_CONFIG_HOME` are kept as the
operator's real environment; `SKILL_DIR` and `SOURCE_DIR` are computed by the
scripts from `$0` and are not inputs.

## P3 — `go` already resolves, so no installer can install it inside a window

Revision-6 F1. `install_go` runs first in both installers, before every managed
binary, and neither offers a skip flag. A guard placed *around* the install
cannot prevent it — only making its non-mutating branch the reachable one can.

```bash
command -v go && go version && brew list --formula | grep -qx go \
  && echo "PRECONDITION-P3-OK"
```

**Expected: an absolute path to a real `go`, a version line, and
`PRECONDITION-P3-OK`.**

If `go` does not resolve, the operator installs it **outside every window**,
under their own control, and re-runs this command. Only then may Step 2, 3, 5 or
6 begin. Note `command -v go` must succeed in the *same environment the
installer will run in* — Steps 2 and 5 run under `env -u …`, which does not
touch `PATH`, so the check and the run agree.

If `brew` is absent entirely: the task-board installer's `install_go` reaches
`exit 1` when `go` is missing, and the agents-infra one reaches `exit 1` on any
non-Darwin or brewless host. With P3 satisfied neither is reachable, because
both return at their first branch.

In-flight and recovery behaviour for an interruption inside `install_go`: it
mutates nothing when P3 holds, so an interruption there leaves no installed
artifact changed and the step is simply re-run from the top. If P3 was **not**
verified and a `brew install go` was interrupted, Homebrew's own formula state
is the damaged surface; that is outside this plan's authority to restore, the
step is reported `unknown/failed`, and the migration does not resume until the
Homebrew owner repairs it. The P6 snapshot is what makes that damage visible
rather than silent.

## P4 — every managed skill is already a real directory

Prevents `install_skills` from taking its network clone branch, whose failure
mode deletes the skill, inside the window. The skill list is read from the
release's own map, so a ninth skill added by `0.25.0` or `0.25.1` is covered
without editing this document.

```bash
sed -n '/^SKILL_REPOS=(/,/^)/p' "$PM_RELEASE/scripts/setup.sh" \
  | sed -nE 's/^[[:space:]]+([A-Za-z0-9._-]+)[[:space:]]+".*/\1/p' \
  | tee "$RECOVERY_ROOT/release-skill-repos.txt"
```

Observed at planning time — 8 keys: `agent-facing-api`,
`android-testing-tools`, `architecture-diagrams`, `go-testing-tools`,
`ios-testing-tools`, `product-appraisal`, `core-data`, `swiftui`. An empty
result is a failed read, not an empty map, and stops the step.

Then, for each name plus the registered `project-management`:

```bash
while IFS= read -r skill; do
  target="$HOME/.agents/skills/$skill"
  if   [ -L "$target" ]; then printf 'REFUSE %-24s symlink\n'   "$skill"
  elif [ -d "$target" ]; then printf 'OK     %-24s directory\n' "$skill"
  elif [ -e "$target" ]; then printf 'REFUSE %-24s not a directory\n' "$skill"
  else                        printf 'REFUSE %-24s absent\n'    "$skill"
  fi
done < "$RECOVERY_ROOT/release-skill-repos.txt"
```

**Expected: nine `OK` lines and no `REFUSE` line** (eight dependencies plus
`project-management`, which the operator appends to the file). A `REFUSE` means
the installer would clone from the network inside the window. The operator
materialises that skill from its own repository beforehand, with their own SSH
agent, and re-runs the check. **Do not let the release installer do it.**

## P5 — the release's artifact sets match the sets the snapshot holds

The snapshot in Part III records the executables, roles and skills of the
*installed* pair. A release that adds one must have the addition removed on
rollback, so the two sets are compared before the window opens.

Managed executables, task-board:

```bash
sed -nE 's/^[[:space:]]*install_binary "([^"]+)".*/\1/p' \
  "$PM_RELEASE/scripts/setup.sh"
```

Observed at planning time: `task-board`, `tb-sessiond`, `task-board-tui`,
`openai-board`, `anthropic-board` (5).

Managed executables, agents-infra:

```bash
sed -nE 's/^(BINARY_NAME|MODEL_HARNESS_BINARY_NAME)="([^"]+)".*/\2/p' \
  "$AI_RELEASE/scripts/setup.sh"
```

Observed at planning time: `agents-infra`, `model-harness` (2).

Roles the task-board release owns — `install_roles` installs whatever is in
`$SKILL_DIR/.roles`:

```bash
ls -1A "$PM_RELEASE/.roles"
```

Observed at planning time: 9 entries, matching the installed `~/.roles`.

For each of the three sets:

```bash
diff <(sort "$RECOVERY_ROOT/manifest/binaries.txt") <(sort release-set.txt)
```

**Expected: no output.** A name only in the release is an artifact the rollback
must quarantine; a name only in the snapshot is one the release stopped
installing. Both are permitted, both must be *seen* before the window, and the
rollback procedure in Part III handles each explicitly. What is not permitted is
discovering the difference afterwards.

## P6 — Homebrew and the LLDB surface are unchanged across the window

Replaces `snapshot_lldb_surface`, `capture_lldb_surface` and
`guarded_agents_infra_install`. Revision-6 F1 showed the old target set — five
LLDB-specific values — could not see a `brew install go`. This snapshot takes
the **entire installed formula list**, so any Homebrew mutation by any stage is
visible.

Save this as `"$RECOVERY_ROOT/snapshot-brew.sh"` and run it immediately before
and immediately after each agents-infra install:

```text
bash "$RECOVERY_ROOT/snapshot-brew.sh" "$RECOVERY_ROOT/brew-before-step3.txt"
…the install…
bash "$RECOVERY_ROOT/snapshot-brew.sh" "$RECOVERY_ROOT/brew-after-step3.txt"
```

It is a top-level script, not a function. Under `set -euo pipefail` at script
scope a failed read aborts it and no file is published, so a failed read can
never be published as a measured absence. That is the same fail-closed property
the old 60-line reader chased through explicit per-command status tests, and it
is obtained here by *not* being a function whose caller can suppress `errexit`
by invoking it as an `if` condition — the exact defect the revision-5 review
found.

```bash
set -euo pipefail
out="$1"                              # destination path, given by the caller
# Refuse to overwrite: a failed re-run must never destroy an earlier good
# snapshot, and a stale partial must never survive to be mistaken for one.
test ! -e "$out"
rm -f "$out.partial"
trap 'rm -f "$out.partial"' EXIT
{
  brew --prefix
  brew list --formula                 # complete formula set: catches install_go
  brew list --versions llvm
  brew --prefix llvm
} > "$out.partial"
prefix="$(brew --prefix)"
llvm_prefix="$(brew --prefix llvm)"
for target in "$prefix/bin/lldb-mcp" "$prefix/bin/lldb-mcp.agents-infra.bak" \
              "$llvm_prefix/bin/lldb-mcp" "$llvm_prefix/bin/lldb"; do
  if [ -e "$target" ] || [ -L "$target" ]; then
    stat -f '%N|%HT|%z|%m|%Sp' "$target"
    shasum -a 256 "$target"
  else
    printf 'ABSENT|%s\n' "$target"
  fi
done >> "$out.partial"
mv -f "$out.partial" "$out"
```

Notes the operator needs:

- `brew list --versions llvm` exits 1 when llvm is genuinely absent. Under
  `pipefail`/`errexit` that aborts the block, which is the correct conservative
  behaviour: **this plan's target host has llvm installed, and a host without it
  must get a reviewed variant of this block rather than a silently degraded
  snapshot.** Do not "fix" it by appending `|| true` — that is precisely the
  fail-open the revision-5 review found.
- The two llvm-prefixed paths are derived from a `brew --prefix llvm` that
  already succeeded, because the block aborts otherwise. They can never collapse
  to `/bin/lldb-mcp`.
- An `ABSENT` line here is a measured fact. On this host, the last two targets
  are `ABSENT` and that is the pre-existing anomaly recorded in Part I.
- The script refuses an existing `$out` rather than overwriting it. Each
  snapshot gets its own filename (`brew-before-step3.txt`,
  `brew-after-step3.txt`, and the Step-6 pair), so a failed re-run cannot
  destroy the good snapshot it was meant to be compared against.

Comparison:

```bash
diff "$RECOVERY_ROOT/brew-before.txt" "$RECOVERY_ROOT/brew-after.txt"
echo "exit=$?"
```

**Expected: no output, `exit=0`.** Both files must exist and be non-empty; if
either is missing the result is `unknown/failed`, not "identical".

A difference means an installer stage mutated Homebrew inside the window. Report
the step `unknown/failed`, withhold every new default-PATH composition, and do
not claim `W3`/`W6` rollback success. **This plan does not restore Homebrew and
does not claim to.** That surface has a separate owner; the migration resumes
only after they have repaired it or accepted the change.

The before-snapshot-to-after-comparison interval is a bypass guard, not an
accepted window: its required outcome is zero Homebrew delta.

## P7 — the live session-manager daemons, and the protocol both sides speak

Applies to **Steps 2 and 5**, the two that replace `tb-sessiond`. Part I
establishes why: `install_binary` replaces a file, the daemon is a process, and
the CLI decides on its own schedule what to do about a process whose protocol
differs from its own. This precondition makes that decision visible before the
window rather than during it.

### P7a — enumerate the live daemons and the boards they hold

```bash
rc=0
pgrep -fl tb-sessiond > "$RECOVERY_ROOT/daemons-before.txt" || rc=$?
case "$rc" in
  0) : ;;                                     # one line per live daemon
  1) printf 'NO-LIVE-DAEMON\n' > "$RECOVERY_ROOT/daemons-before.txt" ;;
  *) printf 'DAEMON-CENSUS-FAILED rc=%s\n' "$rc" >&2; exit 1 ;;
esac
cat "$RECOVERY_ROOT/daemons-before.txt"
```

**Expected: exit 0 and one line per daemon, or exit 1 and `NO-LIVE-DAEMON`.**
`pgrep` returns 1 for "no match" and 2 or 3 for a usage or fatal error; the
`case` keeps those apart, because a failed process census is `unknown` and must
stop the step, never be read as "no daemons are running".

Each line carries the daemon's own arguments, which is where the board comes
from — `--board-dir <path>` for a local board and
`--remote-origin`/`--remote-board-id` for a remote one, per
`cmd/session.go:486-498`. Extract the local boards:

```bash
sed -nE 's/.*--board-dir[ =]([^ ]+).*/\1/p' "$RECOVERY_ROOT/daemons-before.txt" \
  | sort -u > "$RECOVERY_ROOT/daemon-boards.txt"
grep -c -- '--remote-origin' "$RECOVERY_ROOT/daemons-before.txt" || true
```

A non-zero count of remote-origin daemons is not covered by the per-board read
below and must be dispositioned by hand; this plan does not model a remote
board's session plane.

### P7b — read each live daemon's protocol, without starting or restarting one

```bash
: > "$RECOVERY_ROOT/daemon-status-before.jsonl"
while IFS= read -r board; do
  test -n "$board" || continue
  task-board --board-dir "$board" --json session status \
    >> "$RECOVERY_ROOT/daemon-status-before.jsonl"
done < "$RECOVERY_ROOT/daemon-boards.txt"
cat "$RECOVERY_ROOT/daemon-status-before.jsonl"
```

**Expected: one JSON object per board, exit 0.** `session status` uses
`sessionmanager.Dial` (`cmd/session.go:42`), which never launches a daemon and
performs no compatibility check, so this is a read that reports skew instead of
repairing it. A `"running":false` object with exit 0 is a **legitimate absence**;
a non-zero exit, an empty line or unparseable output is `unknown` and stops the
step. Record `protocol_version`, `daemon_version`, `pid` and `session_count` per
board — `session_count` is what an automatic takeover would drain or fence.

Observed on the target host at planning time: two live daemons, both
`protocol_version` 4, `daemon_version` `0.24.3-112-ged48781` and
`0.24.3-124-g2a527a8`, holding 31 and 2 sessions. That is an observation, not a
comparison operand; P7c below compares two live values.

### P7c — derive `controlProtocolVersion` on both sides

```bash
grep -n 'controlProtocolVersion *=' \
  "$SAVED_SOURCE/tools/board-cli/internal/sessionmanager/types.go"
grep -n 'controlProtocolVersion *=' \
  "$PM_RELEASE/tools/board-cli/internal/sessionmanager/types.go"
```

**Expected: exactly one line from each, and the two values equal.** An empty
result from either side is a failed read — the constant may have been renamed or
moved — and is `unknown`, not "no protocol". Neither operand is written in this
document: one comes from the saved installed pair's own source worktree, the
other from the release tree.

### The two dispositions

**Equal — the step is protocol-transparent.** No takeover fires in either
direction, and R1 stays recoverable. It is **not** image-transparent: every live
daemon keeps running its old task-board image, and its old embedded
agents-management, until it is restarted. Protocol equality proves the transport
is compatible; it proves nothing about how two different embedded
agents-management versions compose across it, and this plan makes no such claim.
Required disposition, recorded per board before the window:

- either the board is **quiesced** — `session_count` 0, and the daemon stopped
  by its owner, outside every window, so the next command starts a daemon from
  the new image;
- or it is **explicitly excluded** from goal-bound and writable-context spawn
  until its daemon next exits on its own.

Neither disposition may be "we will see". A board with a live daemon and no
recorded disposition stops the step.

**Different — stop, before the window, for a decision.** The consequences are
laid out in Part I and are not survivable by R1:

1. The first `ConnectOrStart`-class command from the new CLI against each board
   with an older daemon restarts that daemon automatically, draining or fencing
   the sessions it holds. The operator does not choose when: any goal-bound
   `spawn`, `session doctor`, `session logs`, or managed `codex`/`claude`
   wrapper does it.
2. From that moment the rollback is one-way, and **not because the restored CLI
   refuses the newer daemon — it does not.** On every `ConnectOrStart` path the
   restored older CLI meets that daemon with no compatibility check
   (`client.go:145-166` discards the untyped newer-daemon error;
   `stale_takeover.go:168-266` dials with the unchecked `dialManagerStatus`) and
   either receives a live client and issues v4 calls against a v5 daemon, or
   SIGTERM/SIGKILLs it and starts a replacement. Which one is decided by that
   daemon's `startup_healthy`, which is unknown for a protocol it does not yet
   define. Neither is a rollback, and the plan cannot select between them.
3. Therefore the prohibition after a bump is not "avoid the session plane, it
   will refuse you" but **run no `ConnectOrStart`-class command against that
   board, recovery-prefixed or not**: goal-bound `spawn`, `session doctor`,
   `session logs`, `session reclaim-contexts`, `session context` and the managed
   `codex`/`claude` wrappers. `Dial`-only reads (`session status`, `list`) remain
   safe, and are the only thing that is. A downgrade of such a board requires its
   owner to stop the newer daemon first, out of band, with the session loss
   accepted in advance.
4. The migration board itself has no daemon, so its installed canary cannot
   observe any of this.

The only forms of this step that are safe are: **(a)** the release does not
change the constant, or **(b)** every live daemon is quiesced and stopped by its
owner before Step 2 opens, so that no board carries an older daemon into the
window and there is nothing for the new CLI to take over. Option (b) is a
cross-board operation this plan does not have authority to perform; it is a
decision for the owners of `mac-infra` and `casual-talks`, obtained in writing
before the step, and recorded with the step evidence.

## Preconditions that are NOT IMPLEMENTED

Stated so that the gap is visible rather than papered over by an untested guard.

| Requirement | Why it wants automation | Status |
| --- | --- | --- |
| Assert that the interval between the P6 before-snapshot and the install is not raced by a concurrent `brew` invocation | A human cannot observe concurrency; a lockfile check would | **NOT IMPLEMENTED.** Mitigation: run releases when no other Homebrew work is in flight, and treat a P6 difference as authoritative. |
| Mechanically classify a diff produced by P1 into "touches a modelled boundary" versus "does not" | The judgement is currently the operator's, and a large refactor diff is tiring to read | **NOT IMPLEMENTED.** Mitigation: P1's four questions are answered in writing and attached to the step evidence, so the judgement is reviewable after the fact. |
| Verify that the `env -u` list in Steps 2/5 still covers every `TASK_BOARD_*` name P2 reports | Two lists that must agree; today the operator compares them by eye | **NOT IMPLEMENTED.** Mitigation: both lists appear in this document adjacent to each other, and P2's diff catches an added name even though it does not catch a forgotten `-u`. |
| Rehearse any rollback procedure against a real installed pair | See "UNTESTED" throughout Part III | **NOT IMPLEMENTED / UNTESTED.** |
| Prevent a `ConnectOrStart`-class command from reaching a board with an older daemon during `W2`/`W5` | The restart is automatic and any wrapper, agent or human on the host can trigger it; only quiescing the daemons beforehand removes it | **NOT IMPLEMENTED.** Mitigation: P7's per-board disposition, and a release that does not change `controlProtocolVersion`. There is no host-wide interlock, and this plan does not pretend to one. |
| Establish what a live daemon's provider children do when the daemon is terminated out of band rather than drained | Revision 8 called this "the only remaining move after a protocol downgrade", implying somebody chooses it. Part I now shows the restored older CLI performs it **automatically** on one of the two branches of `takeOverUnresponsiveManager`. Its blast radius decides whether that state is recoverable at all | **NOT IMPLEMENTED / UNKNOWN.** Not measured here, and not guessed. Raised in importance by the F1 correction: it is no longer a consequence of an operator decision but of a command any agent or human on the host can run. |
| Fail closed when an older CLI meets a newer daemon on a `ConnectOrStart` path | The refusal text exists (`client.go:725-731`) and is unreachable from every production `ConnectOrStart` call site: the newer branch returns an untyped `fmt.Errorf`, `ConnectOrStart`'s single `errors.As` matches only the typed older-daemon error, and `takeOverUnresponsiveManager` runs no compatibility check. So the downgrade direction is unprotected, not protected | **NOT IMPLEMENTED, and not this plan's to implement.** This is a change to `skill-project-management` — a typed error for the newer branch surfaced by `ConnectOrStart`, or a compatibility check inside `takeOverUnresponsiveManager` before it returns a client or signals a PID — and a **named prerequisite** if the downgrade direction is ever to be survivable. The plan cannot add behaviour to a binary it only describes. Mitigation until then: P7's prevention, and R1 step 8's prohibition on `ConnectOrStart`-class commands. |

None of these is a substitute for a guard that was removed: P1, P2, P3, P4, P5
and P6 each cover strictly more than the code they replace. These are properties
no revision ever had, and the last one is a property the *product* does not have.

# Part III — Recovery snapshot and rollback

Run the snapshot before Step 2, and repeat it into a fresh `bridge-pair`
snapshot after both `0.25.0` and `v1.7.0` pass installed canaries. These
commands copy no credential, cookie, token, keychain data or environment value.
**Never overwrite a prior snapshot.**

Nothing in this section names a binary, role, skill, repository path or install
directory that this plan decided. Every set is read from the artifact that owns
it, at execution time.

## Snapshot

```bash
set -euo pipefail

# Host paths come from the install state the installers themselves wrote.
PM_STATE="$HOME/.config/task-board/install.json"
AI_STATE="$HOME/Library/Application Support/agents-infra/install.json"   # Darwin
# Non-Darwin: "${XDG_CONFIG_HOME:-$HOME/.config}/agents-infra/install.json",
# per relux-agents-infra/scripts/setup.sh:64-73. P1 covers a release that
# changes that rule.

PM_REPO="$(python3 -c 'import json,sys;print(json.load(open(sys.argv[1]))["repoPath"])' "$PM_STATE")"
AI_REPO="$(python3 -c 'import json,sys;print(json.load(open(sys.argv[1]))["repoPath"])' "$AI_STATE")"
PM_BIN_DIR="$(python3 -c 'import json,sys;print(json.load(open(sys.argv[1]))["binDir"])' "$PM_STATE")"
AI_BIN_DIR="$(python3 -c 'import json,sys;print(json.load(open(sys.argv[1]))["binDir"])' "$AI_STATE")"
test "$PM_BIN_DIR" = "$AI_BIN_DIR"
BIN_DIR="$PM_BIN_DIR"
test -d "$PM_REPO" && test -d "$AI_REPO"

RECOVERY_ROOT="$HOME/.local/state/TASK-260830-s5ro4e/current-pair"
MANIFEST="$RECOVERY_ROOT/manifest"
test ! -e "$RECOVERY_ROOT"
mkdir -p "$RECOVERY_ROOT/bin" "$MANIFEST" "$RECOVERY_ROOT/pm/roles" \
         "$RECOVERY_ROOT/pm/skills" "$RECOVERY_ROOT/pm/claude-skills" \
         "$RECOVERY_ROOT/pm/codex-skills" "$RECOVERY_ROOT/ai" "$RECOVERY_ROOT/sources"

printf '%s\n' "$BIN_DIR"  > "$MANIFEST/bin-dir.txt"
printf '%s\n' "$PM_STATE" > "$MANIFEST/pm-state-path.txt"
printf '%s\n' "$AI_STATE" > "$MANIFEST/ai-state-path.txt"

# Same filesystem, so every restore is an atomic rename rather than a copy that
# can fail half way.
device="$(stat -f %d "$RECOVERY_ROOT")"
for target in "$BIN_DIR" "$HOME/.agents/skills" "$HOME/.roles" \
              "$HOME/.claude/skills" "$HOME/.codex/skills" \
              "$(dirname "$PM_STATE")" "$(dirname "$AI_STATE")"; do
  test -d "$target"
  test "$(stat -f %d "$target")" = "$device"
done

# --- executables: the set is read from the checked-out installers ---
{
  sed -nE 's/^[[:space:]]*install_binary "([^"]+)".*/\1/p' "$PM_REPO/scripts/setup.sh"
  sed -nE 's/^(BINARY_NAME|MODEL_HARNESS_BINARY_NAME)="([^"]+)".*/\2/p' "$AI_REPO/scripts/setup.sh"
} > "$MANIFEST/binaries.txt"
test -s "$MANIFEST/binaries.txt"

while IFS= read -r name; do
  test -n "$name" || continue
  test -f "$BIN_DIR/$name" && test ! -L "$BIN_DIR/$name"
  cp -p "$BIN_DIR/$name" "$RECOVERY_ROOT/bin/$name"
  cmp -s "$BIN_DIR/$name" "$RECOVERY_ROOT/bin/$name"
done < "$MANIFEST/binaries.txt"

ls -1A "$BIN_DIR" > "$MANIFEST/bin-inventory.txt"
shasum -a 256 "$RECOVERY_ROOT/bin"/* > "$RECOVERY_ROOT/bin.sha256"

# --- recovery sources, at the commit each saved executable reports ---
TB_VERSION="$("$RECOVERY_ROOT/bin/task-board" --version)"
SESSIOND_VERSION="$("$RECOVERY_ROOT/bin/tb-sessiond" --version)"
AI_VERSION="$("$RECOVERY_ROOT/bin/agents-infra" version)"
printf '%s\n%s\n%s\n' "$TB_VERSION" "$SESSIOND_VERSION" "$AI_VERSION" \
  > "$RECOVERY_ROOT/versions.txt"

# task-board prints "… (commit <sha>, built …)"; agents-infra prints
# "… commit=<sha> build_date=…". Both are extracted by the same pattern.
extract_commit() { printf '%s\n' "$1" | sed -nE \
  -e 's/.*\(commit ([0-9a-f]{7,40}),.*/\1/p' \
  -e 's/.* commit=([0-9a-f]{7,40})( .*)?$/\1/p'; }

TB_COMMIT="$(extract_commit "$TB_VERSION")"
SD_COMMIT="$(extract_commit "$SESSIOND_VERSION")"
AI_COMMIT="$(extract_commit "$AI_VERSION")"
test -n "$TB_COMMIT" && test -n "$SD_COMMIT" && test -n "$AI_COMMIT"
# One repository ships task-board and tb-sessiond. A divergence means the
# installed pair is incoherent and blocks Step 2 rather than picking one.
test "$TB_COMMIT" = "$SD_COMMIT"

TB_SOURCE="$RECOVERY_ROOT/sources/skill-project-management"
AI_SOURCE="$RECOVERY_ROOT/sources/relux-agents-infra"
# Absolute destinations: `git -C repo worktree add` resolves a relative path
# against the repository, not the caller, and still exits zero.
git -C "$PM_REPO" worktree add --detach "$TB_SOURCE" \
  "$(git -C "$PM_REPO" rev-parse --verify "${TB_COMMIT}^{commit}")"
git -C "$AI_REPO" worktree add --detach "$AI_SOURCE" \
  "$(git -C "$AI_REPO" rev-parse --verify "${AI_COMMIT}^{commit}")"
test -e "$TB_SOURCE/.git" && test -e "$AI_SOURCE/.git"
test -z "$(git -C "$TB_SOURCE" status --porcelain)"
test -z "$(git -C "$AI_SOURCE" status --porcelain)"
git -C "$TB_SOURCE" rev-parse HEAD > "$MANIFEST/pm-source-commit.txt"
git -C "$AI_SOURCE" rev-parse HEAD > "$MANIFEST/ai-source-commit.txt"

# The artifact set must be identical when re-read from the exact sources the
# saved binaries were built from. If a repository HEAD moved and changed it,
# the copy above is not the installed release's artifact set.
{
  sed -nE 's/^[[:space:]]*install_binary "([^"]+)".*/\1/p' "$TB_SOURCE/scripts/setup.sh"
  sed -nE 's/^(BINARY_NAME|MODEL_HARNESS_BINARY_NAME)="([^"]+)".*/\2/p' "$AI_SOURCE/scripts/setup.sh"
} > "$MANIFEST/binaries-from-source.txt"
diff "$MANIFEST/binaries.txt" "$MANIFEST/binaries-from-source.txt"

# --- roles: the whole installed surface, saved and named ---
ls -1A "$HOME/.roles" > "$MANIFEST/roles.txt"
while IFS= read -r role; do
  test -n "$role" || continue
  test -d "$HOME/.roles/$role"
  rsync -a --delete "$HOME/.roles/$role/" "$RECOVERY_ROOT/pm/roles/$role/"
done < "$MANIFEST/roles.txt"

# --- skills: the registered skill plus every SKILL_REPOS dependency ---
{
  sed -nE 's/^[[:space:]]*local skill_name="([^"]+)".*/\1/p' "$TB_SOURCE/scripts/setup.sh"
  sed -n '/^SKILL_REPOS=(/,/^)/p' "$TB_SOURCE/scripts/setup.sh" \
    | sed -nE 's/^[[:space:]]+([A-Za-z0-9._-]+)[[:space:]]+".*/\1/p'
} > "$MANIFEST/skills.txt"
test -s "$MANIFEST/skills.txt"

: > "$MANIFEST/skill-state.txt"
while IFS= read -r skill; do
  test -n "$skill" || continue
  case "$skill" in *"|"*) exit 76 ;; esac
  agents_target="$HOME/.agents/skills/$skill"
  if   [ -L "$agents_target" ]; then kind=symlink;   link="$(readlink "$agents_target")"
  elif [ -d "$agents_target" ]; then kind=directory; link=""
       rsync -a --delete "$agents_target/" "$RECOVERY_ROOT/pm/skills/$skill/"
  elif [ -e "$agents_target" ]; then exit 77
  else                               kind=absent;    link=""
  fi
  line="$skill|$kind|$link"
  # Each link tree gets its own staging area. Revision 7 recorded a `directory`
  # kind here and staged nothing for it, so R1 met an empty tree and aborted
  # halfway through. A kind is only worth recording if its restore has an
  # operand.
  for spec in "$HOME/.claude/skills|claude-skills" "$HOME/.codex/skills|codex-skills"; do
    root="${spec%%|*}"; stage="${spec##*|}"
    if   [ -L "$root/$skill" ]; then line="$line|symlink|$(readlink "$root/$skill")"
    elif [ -d "$root/$skill" ]; then line="$line|directory|"
         rsync -a --delete "$root/$skill/" "$RECOVERY_ROOT/pm/$stage/$skill/"
    elif [ -e "$root/$skill" ]; then exit 77
    else                             line="$line|absent|"
    fi
  done
  printf '%s\n' "$line" >> "$MANIFEST/skill-state.txt"
done < "$MANIFEST/skills.txt"

# --- the snapshot must be restorable, asserted here rather than during R1 ---
# Refusing at the door costs nothing. Aborting mid-rollback leaves the host in a
# state neither the release nor the rollback planned for.
snapshot_restorable() {   # $1 label  $2 kind  $3 recorded link  $4 staged tree
  case "$2" in
    symlink)   test -n "$3" || { printf 'UNRESTORABLE %s symlink with no recorded target\n' "$1" >&2; return 1; } ;;
    directory) test -d "$4" || { printf 'UNRESTORABLE %s directory with no staged tree\n'   "$1" >&2; return 1; } ;;
    absent)    : ;;
    *)         printf 'UNRESTORABLE %s unrecognised kind %s\n' "$1" "$2" >&2; return 1 ;;
  esac
}
while IFS='|' read -r skill a_kind a_link c_kind c_link x_kind x_link; do
  test -n "$skill" || continue
  snapshot_restorable "$skill:.agents" "$a_kind" "$a_link" "$RECOVERY_ROOT/pm/skills/$skill"
  snapshot_restorable "$skill:.claude" "$c_kind" "$c_link" "$RECOVERY_ROOT/pm/claude-skills/$skill"
  snapshot_restorable "$skill:.codex"  "$x_kind" "$x_link" "$RECOVERY_ROOT/pm/codex-skills/$skill"
done < "$MANIFEST/skill-state.txt"

# --- both machine-scoped install states ---
# The agents-infra one records repoPath, which the installed binary uses to
# resolve its own source tree, so a rollback that leaves it naming an abandoned
# release candidate leaves a binary that cannot find its source.
install -m 0644 "$PM_STATE" "$RECOVERY_ROOT/pm/install.json"
install -m 0644 "$AI_STATE" "$RECOVERY_ROOT/ai/install.json"
```

Expected at planning time: `binaries.txt` 7 lines, `roles.txt` 9,
`skills.txt` 9, `skill-state.txt` 9. Those counts are observations; the
`diff` above, not a count written here, is the check.

The `snapshot_restorable` pass is deliberately duplicated as R1's step 0. The
snapshot proves the artifact it just wrote is restorable; R1 proves the artifact
it is about to consume is restorable, whoever wrote it and however long ago.
Neither may rely on the other having run.

A parse failure, an unresolvable commit, a dirty source worktree, a divergent
task-board/tb-sessiond commit, a failed `cmp`, a changed derived set or any
missing executable blocks Step 2. None of them is treated as an absent identity
or a usable rollback baseline.

**Property preserved from revision 5: the recovery baseline and the recovery
binary are the same pair by construction.** Each source worktree is created at
the commit its own saved executable reports, resolved in the owning repository
at execution time. No revision, executable name, role name, skill name,
repository path or install directory is written into this document that has to
be refreshed when either repository moves.

For the bridge snapshot use a fresh
`$HOME/.local/state/TASK-260830-s5ro4e/bridge-pair`; the block is identical.

## Rollback R1 — task-board surface (Steps 2 and 5)

Set `RECOVERY_ROOT` to `current-pair` for Step 2 and `bridge-pair` for Step 5.
`PM_RELEASE` is the release tree that was installed.

```bash
set -euo pipefail
MANIFEST="$RECOVERY_ROOT/manifest"
BIN_DIR="$(cat "$MANIFEST/bin-dir.txt")"

# 0. Admissibility. R1 refuses a snapshot it cannot restore BEFORE it mutates
#    anything: every recorded kind must be one it can restore, every symlink
#    must have a target, every directory must have a staged tree, and every
#    saved executable, role and install state must be present. Revision 7
#    discovered these one at a time, mid-rollback, after executables had already
#    been replaced — a half-restore is a state neither side planned for.
snapshot_restorable() {   # $1 label  $2 kind  $3 recorded link  $4 staged tree
  case "$2" in
    symlink)   test -n "$3" || { printf 'UNRESTORABLE %s symlink with no recorded target\n' "$1" >&2; return 1; } ;;
    directory) test -d "$4" || { printf 'UNRESTORABLE %s directory with no staged tree\n'   "$1" >&2; return 1; } ;;
    absent)    : ;;
    *)         printf 'UNRESTORABLE %s unrecognised kind %s\n' "$1" "$2" >&2; return 1 ;;
  esac
}
while IFS='|' read -r skill a_kind a_link c_kind c_link x_kind x_link; do
  test -n "$skill" || continue
  snapshot_restorable "$skill:.agents" "$a_kind" "$a_link" "$RECOVERY_ROOT/pm/skills/$skill"
  snapshot_restorable "$skill:.claude" "$c_kind" "$c_link" "$RECOVERY_ROOT/pm/claude-skills/$skill"
  snapshot_restorable "$skill:.codex"  "$x_kind" "$x_link" "$RECOVERY_ROOT/pm/codex-skills/$skill"
done < "$MANIFEST/skill-state.txt"
while IFS= read -r name; do
  test -n "$name" || continue
  test -f "$RECOVERY_ROOT/bin/$name" || \
    { printf 'UNRESTORABLE bin/%s missing from the snapshot\n' "$name" >&2; exit 1; }
done < "$MANIFEST/binaries.txt"
while IFS= read -r role; do
  test -n "$role" || continue
  test -d "$RECOVERY_ROOT/pm/roles/$role" || \
    { printf 'UNRESTORABLE roles/%s missing from the snapshot\n' "$role" >&2; exit 1; }
done < "$MANIFEST/roles.txt"
test -f "$RECOVERY_ROOT/pm/install.json"

# 1. Executables the snapshot holds -> restored.
sed -nE 's/^[[:space:]]*install_binary "([^"]+)".*/\1/p' \
  "$PM_RELEASE/scripts/setup.sh" | sort -u > /tmp/r1-release-bins.$$
sort -u "$MANIFEST/binaries.txt" > /tmp/r1-snapshot-bins.$$

while IFS= read -r name; do
  test -n "$name" || continue
  if grep -qxF "$name" /tmp/r1-snapshot-bins.$$; then
    install -m 0755 "$RECOVERY_ROOT/bin/$name" "$BIN_DIR/.$name.rollback.$$"
    mv -f "$BIN_DIR/.$name.rollback.$$" "$BIN_DIR/$name"
  elif [ -e "$BIN_DIR/$name" ]; then
    # 2. An executable this release added is not part of the previous state.
    mv -f "$BIN_DIR/$name" "$BIN_DIR/.$name.superseded.$$"
  fi
done < /tmp/r1-release-bins.$$

# 3. Anything else that appeared in BIN_DIR during the window is reported, not
#    removed: this plan owns the release's artifacts, not the operator's PATH.
ls -1A "$BIN_DIR" | grep -vxF -f "$MANIFEST/bin-inventory.txt" \
  > "$RECOVERY_ROOT/unexpected-bin-entries.txt" || true

# 4. Skills: restore each snapshot entry to its exact recorded kind. Step 0
#    has already proved every kind restorable, so the arms below cannot meet a
#    missing operand. The `*)` arm and the symlink arm's `test -n` stay as a
#    second line of defence; step 0 makes both unreachable, so neither is
#    exercised by any probe, and Part VI says so.
restore_entry() {   # $1 path  $2 kind  $3 link-target  $4 staged tree
  d="$(dirname "$1")"; b="$(basename "$1")"
  case "$2" in
    symlink)
      test -n "$3"
      rm -rf "$1"
      ln -sfn "$3" "$1"
      ;;
    directory)
      test -d "$4"
      rsync -a --delete "$4/" "$d/.$b.rollback.$$/"
      if [ -e "$1" ] || [ -L "$1" ]; then mv "$1" "$d/.$b.failed.$$"; fi
      mv "$d/.$b.rollback.$$" "$1"
      ;;
    absent)
      if [ -e "$1" ] || [ -L "$1" ]; then mv "$1" "$d/.$b.superseded.$$"; fi
      ;;
    *)
      printf 'UNKNOWN-KIND %s %s\n' "$1" "$2" >&2
      return 1
      ;;
  esac
}

while IFS='|' read -r skill a_kind a_link c_kind c_link x_kind x_link; do
  test -n "$skill" || continue
  restore_entry "$HOME/.agents/skills/$skill" "$a_kind" "$a_link" \
                "$RECOVERY_ROOT/pm/skills/$skill"
  restore_entry "$HOME/.claude/skills/$skill" "$c_kind" "$c_link" \
                "$RECOVERY_ROOT/pm/claude-skills/$skill"
  restore_entry "$HOME/.codex/skills/$skill"  "$x_kind" "$x_link" \
                "$RECOVERY_ROOT/pm/codex-skills/$skill"
done < "$MANIFEST/skill-state.txt"

# 5. A skill this release's SKILL_REPOS added is removed from all three trees.
{
  sed -nE 's/^[[:space:]]*local skill_name="([^"]+)".*/\1/p' "$PM_RELEASE/scripts/setup.sh"
  sed -n '/^SKILL_REPOS=(/,/^)/p' "$PM_RELEASE/scripts/setup.sh" \
    | sed -nE 's/^[[:space:]]+([A-Za-z0-9._-]+)[[:space:]]+".*/\1/p'
} | sort -u > /tmp/r1-release-skills.$$
comm -23 /tmp/r1-release-skills.$$ <(sort -u "$MANIFEST/skills.txt") \
  > /tmp/r1-added-skills.$$
while IFS= read -r skill; do
  test -n "$skill" || continue
  restore_entry "$HOME/.agents/skills/$skill" absent "" ""
  restore_entry "$HOME/.claude/skills/$skill" absent "" ""
  restore_entry "$HOME/.codex/skills/$skill"  absent "" ""
done < /tmp/r1-added-skills.$$

# 6. Roles: restore every snapshot role, remove every role this release added.
while IFS= read -r role; do
  test -n "$role" || continue
  rsync -a --delete "$RECOVERY_ROOT/pm/roles/$role/" "$HOME/.roles/.$role.rollback.$$/"
  if [ -e "$HOME/.roles/$role" ]; then
    mv "$HOME/.roles/$role" "$HOME/.roles/.$role.failed.$$"
  fi
  mv "$HOME/.roles/.$role.rollback.$$" "$HOME/.roles/$role"
done < "$MANIFEST/roles.txt"

comm -23 <(ls -1A "$PM_RELEASE/.roles" | sort -u) <(sort -u "$MANIFEST/roles.txt") \
  > /tmp/r1-added-roles.$$
while IFS= read -r role; do
  test -n "$role" || continue
  if [ -e "$HOME/.roles/$role" ]; then
    mv "$HOME/.roles/$role" "$HOME/.roles/.$role.superseded.$$"
  fi
done < /tmp/r1-added-roles.$$

# 7. Install state last.
install -m 0644 "$RECOVERY_ROOT/pm/install.json" \
  "$(dirname "$(cat "$MANIFEST/pm-state-path.txt")")/.install.json.rollback.$$"
mv -f "$(dirname "$(cat "$MANIFEST/pm-state-path.txt")")/.install.json.rollback.$$" \
  "$(cat "$MANIFEST/pm-state-path.txt")"

# 8. Session-manager daemons. R1 restores files. It does not restore a running
#    daemon and it cannot downgrade one, so this step REPORTS rather than
#    repairs. A board whose daemon is newer than the restored CLI supports has
#    not been rolled back, whatever the file system says -- and the restored CLI
#    does NOT refuse such a daemon: on every ConnectOrStart path it either drives
#    it unchecked or SIGTERM/SIGKILLs it. So the census below is followed by a
#    prohibition, not by a safe idle state. Read the canary notes before running
#    anything else against a board this step names.
rc=0
pgrep -fl tb-sessiond > "$RECOVERY_ROOT/daemons-after-r1.txt" || rc=$?
case "$rc" in
  0) : ;;
  1) printf 'NO-LIVE-DAEMON\n' > "$RECOVERY_ROOT/daemons-after-r1.txt" ;;
  *) printf 'DAEMON-CENSUS-FAILED rc=%s\n' "$rc" >&2; exit 1 ;;
esac
sed -nE 's/.*--board-dir[ =]([^ ]+).*/\1/p' "$RECOVERY_ROOT/daemons-after-r1.txt" \
  | sort -u > "$RECOVERY_ROOT/daemon-boards-after-r1.txt"

rm -f /tmp/r1-release-bins.$$ /tmp/r1-snapshot-bins.$$ /tmp/r1-release-skills.$$ \
      /tmp/r1-added-skills.$$ /tmp/r1-added-roles.$$
```

Then the recovery canary:

```bash
PATH="$RECOVERY_ROOT/bin:$PATH" "$RECOVERY_ROOT/bin/task-board" --version
PATH="$RECOVERY_ROOT/bin:$PATH" "$RECOVERY_ROOT/bin/tb-sessiond" --version
PATH="$RECOVERY_ROOT/bin:$PATH" \
  "$RECOVERY_ROOT/bin/task-board" q --format compact \
  'project_config(view=spawn-preflight, role=developer, agent=codex, task_class=research)'

# Reach the session manager, not only the config reader. `project_config` need
# never touch a daemon, so it can go green over exactly the failure this step
# has to detect. `session status` uses Dial: it starts nothing, restarts
# nothing, and reports the protocol the daemon is actually speaking.
while IFS= read -r board; do
  test -n "$board" || continue
  PATH="$RECOVERY_ROOT/bin:$PATH" \
    "$RECOVERY_ROOT/bin/task-board" --board-dir "$board" --json session status
done < "$RECOVERY_ROOT/daemon-boards-after-r1.txt"

# then a live spawn -> outcome -> handoff -> reviewer route canary, recovery-prefixed,
# on a board whose reported protocol_version the restored CLI supports.
#
# For any board reporting a GREATER protocol_version: run NO ConnectOrStart-class
# command against it -- goal-bound spawn, session doctor, session logs, session
# reclaim-contexts, session context, or the managed codex/claude wrappers --
# recovery-prefixed or not. The `session status` reads above are Dial-based and
# are the only safe ones. Escalate to that board's owner instead.
```

Reading the canary:

- `"running":false` with exit 0 is a **legitimate absence**: no daemon holds
  that board, and the next command will start one from the restored image. A
  non-zero exit or unparseable output is `unknown`, and R1 is not green.
- `protocol_version` **equal** to the restored build's `controlProtocolVersion`
  (P7c, read from `$RECOVERY_ROOT/sources/skill-project-management`) is the
  rolled-back state.
- `protocol_version` **greater** is the one state R1 cannot fix, and it is more
  dangerous than revision 8 wrote. The restored CLI does **not** refuse that
  daemon on any `ConnectOrStart` path: `client.go:145-166` matches only the
  typed older-daemon error and discards the untyped newer one, and
  `takeOverUnresponsiveManager` (`stale_takeover.go:168-266`) dials with the
  unchecked `dialManagerStatus`. So the next such command either gets a live
  client and issues v4 calls against a newer daemon, or SIGTERM/SIGKILLs a daemon
  holding live sessions and starts a replacement — decided by that daemon's
  `startup_healthy`, which is unknown. Report the step `unknown/failed`, name the
  board, and escalate to that board's owner.

  **From that point, run no `ConnectOrStart`-class command against that board —
  recovery-prefixed or not.** That is: goal-bound `spawn`, `session doctor`,
  `session logs`, `session reclaim-contexts`, `session context`, and the managed
  `codex`/`claude` wrappers. The prohibition is wider than "do not run the live
  spawn canary", because the canary is not the only command that fires the
  takeover; any of the above does, including one an agent or another human on the
  host runs. The `session status` reads above are safe and are the only reads that
  are: `Dial` starts nothing and checks nothing.

The last bullet is why P7 exists. R1 can detect this state; it was never able to
recover it, revision 7 was wrong to imply otherwise by pairing R1 with a canary
that could not see it, and revision 8 was wrong in the other direction — it
believed a refusal stood between the restored CLI and the newer daemon. Nothing
does. Detection is followed by a prohibition, not by a safe idle state.

Rollback status: **UNTESTED operationally.** R1's logic was exercised against a
disposable `HOME` with fake artifacts (Part VI section G: restore, quarantine of
release-added artifacts, preservation of unrelated ones, and refusal on a
malformed snapshot or a missing saved executable). That is not a rehearsal
against a real installed pair and does not make R1 assumed-reversible.

A partial run of R1 is a rollback failure, not a successful half-restore — and
after step 0, a snapshot R1 cannot fully consume produces no partial run at all.

**R1's scope, stated plainly.** R1 restores files: executables, skills, roles and
the machine-scoped install state. It does not restore processes. For provider
children and already-exec'd CLI processes that is correct and intended. For
`tb-sessiond` it is a limit: a daemon started from the release during the window
survives R1, and if the release changed `controlProtocolVersion`, R1 cannot make
that board usable again. Step 8 detects it; nothing in this plan repairs it.

## Rollback R2 — agents-infra managed runtime (Steps 3 and 6)

```bash
set -euo pipefail
MANIFEST="$RECOVERY_ROOT/manifest"
BIN_DIR="$(cat "$MANIFEST/bin-dir.txt")"

sed -nE 's/^(BINARY_NAME|MODEL_HARNESS_BINARY_NAME)="([^"]+)".*/\2/p' \
  "$AI_RELEASE/scripts/setup.sh" | sort -u > /tmp/r2-release-bins.$$

while IFS= read -r name; do
  test -n "$name" || continue
  if grep -qxF "$name" "$MANIFEST/binaries.txt"; then
    install -m 0755 "$RECOVERY_ROOT/bin/$name" "$BIN_DIR/.$name.rollback.$$"
    mv -f "$BIN_DIR/.$name.rollback.$$" "$BIN_DIR/$name"
  elif [ -e "$BIN_DIR/$name" ]; then
    mv -f "$BIN_DIR/$name" "$BIN_DIR/.$name.superseded.$$"
  fi
done < /tmp/r2-release-bins.$$

# Re-mint the runtime from the exact saved source. AGENTS_INFRA_SOURCE_DIR must
# not be set: it would override --source-dir's resolution order.
env -u AGENTS_INFRA_SOURCE_DIR -u AGENTS_INFRA_CONFIG_DIR \
  "$RECOVERY_ROOT/bin/agents-infra" setup global \
  --source-dir "$RECOVERY_ROOT/sources/relux-agents-infra"
env -u AGENTS_INFRA_SOURCE_DIR "$RECOVERY_ROOT/bin/agents-infra" verify global
env -u AGENTS_INFRA_SOURCE_DIR "$RECOVERY_ROOT/bin/agents-infra" doctor global

# Install state last: the installed binary resolves its own source tree
# through this file.
install -m 0644 "$RECOVERY_ROOT/ai/install.json" \
  "$(dirname "$(cat "$MANIFEST/ai-state-path.txt")")/.install.json.rollback.$$"
mv -f "$(dirname "$(cat "$MANIFEST/ai-state-path.txt")")/.install.json.rollback.$$" \
  "$(cat "$MANIFEST/ai-state-path.txt")"

rm -f /tmp/r2-release-bins.$$
```

R2 deliberately does not copy `~/.agents` wholesale and does not print config
contents.

**R2 has no execution evidence of any kind in this revision.** Unlike R1 it was
not exercised even in a sandbox, because `setup global` mutates an installed
runtime. Its risk is correspondingly higher and a rehearsal on a disposable
home is the first thing an operator should do before Step 3.

The restored install state names the original repository path, while the runtime
receipt written by `setup global` names the recovery source worktree. That
divergence is deliberate and must be recorded with the rollback evidence: the
recovery worktree is retained until a clean reinstall from the original
repository re-mints both. **Deleting `$RECOVERY_ROOT` after a rollback leaves a
runtime whose receipt names a path that no longer exists.**

Rollback status: **UNTESTED**.

All rollback procedures in this plan are **UNTESTED operationally** until a
release operator rehearses them on a disposable home/runtime and records the
real exits. They are not assumed reversible. Earlier revisions attacked the
removed harness in a sandbox; that evidence concerned code this revision no
longer contains and is not carried forward as rehearsal evidence for R1 or R2.

The two rollbacks are not equally unknown, and the difference matters when
sequencing the rehearsal:

| Procedure | Logic exercised? | Run against a real pair? | Status |
| --- | --- | --- | --- |
| R1 (task-board surface) | yes — Part VI sections G and J, disposable `HOME`, fake artifacts, six refusal probes with narrowing controls | no | **UNTESTED operationally** |
| R1 step 8, daemon census | yes — the `pgrep` status discrimination, section J | no | **UNTESTED operationally** |
| R2 (agents-infra runtime) | **no** | no | **UNTESTED** |
| Snapshot block | derivations, version parsing and the restorability pass only | no | **UNTESTED** |
| Recovery from a newer-protocol daemon | **there is no procedure** | n/a | **NOT RECOVERABLE.** Prevented by P7 or not prevented at all. |

# Part IV — Release order

## Step 0 — freeze the current working pair

Repository/version: whatever pair is actually installed when this step runs.
The operator does not read a version out of this document.

At planning time the installed pair measured `0.24.3-172-g063197b1` embedding
`v0.2.0` and `v1.6.1-103-g4270549`. That is an observation, not a pin: both
repositories moved during this planning task alone.

Required board capability: run read, preflight, one real spawn, outcome resource
handoff and reviewer route using the installed pair, then capture the
`current-pair` snapshot. The snapshot is admissible only when each saved
executable's reported commit resolves to the HEAD of its clean detached source
worktree. In a disposable HOME, the saved agents-infra binary must run
`setup global --source-dir` against its own derived source worktree and then
`verify global`, both exit 0.

An installed executable whose reported commit does not resolve in its repository
blocks Step 0. The operator repairs or reinstalls a resolvable pair rather than
substituting a nearby revision.

In-flight runs: remain on their existing process images. Do not restart or kill
them. Do not begin Step 2 while a run is at a handoff/acceptance mutation
boundary.

`tb-sessiond` is the exception to "existing process images", and Step 0 is where
it is written down rather than assumed. Run **P7a and P7b here**, before the
snapshot, and record the census with it: which boards hold a live daemon, each
daemon's `protocol_version`, `daemon_version`, `pid` and `session_count`. Two
daemons on this host already run images older than the installed file, so the
census is a real measurement and not a formality. Step 0 mutates nothing and the
census is a read; its purpose is that Step 2's disposition has a baseline to be
a disposition *about*.

Mismatch window: none; this step only copies explicitly named non-secret
artifacts and creates detached source worktrees.

Rollback: no installed artifact changes. On snapshot failure, stop and discard
only the newly created task-scoped snapshot after inspecting it; no release step
may start.

Rollback status: **UNTESTED**.

## Step 1 — hold agents-management at `v0.5.0`

Repository/version: `skill-agents-management v0.5.0` at the exact
commit/checksum named above. The tag already exists; this task publishes
nothing.

Required board capability: the current installed board continues to spawn and
route with embedded `v0.2.0`. The exact detached consumer candidate must pass:

```bash
cd "$PM_CANDIDATE/tools/board-cli"
GOWORK=off go test -mod=mod ./internal/spawn -count=1
GOWORK=off go build -mod=mod ./...
```

In-flight runs: unaffected; no executable changes.

Mismatch window: none at runtime. A source-only scratch repin exists until its
worktree is removed, but the installed board does not read it.

Rollback commands for a failed scratch candidate:

```bash
cd "$PM_CANDIDATE/tools/board-cli"
GOWORK=off go mod edit \
  -require=github.com/relux-works/skill-agents-management@v0.2.0
GOFLAGS=-mod=mod GOWORK=off go mod tidy
GOWORK=off go test -mod=mod ./internal/spawn -count=1
GOWORK=off go build -mod=mod ./...
```

Rollback status: **PARTIALLY TESTED** for source build/read/preflight only;
installed restoration remains untested.

## Step 2 — release and install task-board `0.25.0`

Repository/version: `skill-project-management 0.25.0`, exact public
agents-management `v0.5.0`. agents-infra remains the old installed release.

Required board capability: the saved current-pair remains able to perform all
spawn/route/resource/CR operations. Before install, the reviewed candidate runs
the full suite/build, graph inventory, binary symbol inventory and an
out-of-place real spawn-and-review canary. After install, repeat the same live
canary through installed `task-board`.

### Preconditions, all outside the window

```bash
RECOVERY_ROOT="$HOME/.local/state/TASK-260830-s5ro4e/current-pair"
PM_RELEASE="$HOME/.local/state/TASK-260830-s5ro4e/release-candidates/skill-project-management-0.25.0"
SAVED_SOURCE="$RECOVERY_ROOT/sources/skill-project-management"

test "$(git -C "$PM_RELEASE" describe --tags --exact-match)" = "0.25.0"
```

Then, in order, each with the expected result stated in Part II:

| # | Precondition | Command | Expected |
| --- | --- | --- | --- |
| P1 | installer text unchanged | `diff -ru "$SAVED_SOURCE/scripts" "$PM_RELEASE/scripts"` | no output |
| P2 | environment surface unchanged | the two `grep`/`sed`/`sort -u` sets diffed; plus the indirect-expansion probe | no output, both |
| P3 | `go` already resolves | `command -v go && go version && brew list --formula \| grep -qx go` | path, version, exit 0 |
| P4 | every managed skill is a real directory | the `SKILL_REPOS` loop | nine `OK`, no `REFUSE` |
| P5 | artifact sets match the snapshot | three `diff`s (executables, roles, skills) | no output, or a difference the operator has recorded and R1 handles |
| P7 | live daemons censused, control protocol unchanged | `pgrep -fl tb-sessiond`, per-board `session status --json`, and `grep -n 'controlProtocolVersion *='` on both source trees | a census with a recorded disposition per board, and **two equal protocol values**; a difference stops the step for a decision, and a failed census is `unknown` |

Record the `setup_main` stage list the release will actually run:

```bash
sed -n '/^setup_main() {/,/^}/p' "$PM_RELEASE/scripts/setup.sh" \
  > "$RECOVERY_ROOT/w2-setup-main.txt"
```

This is what `W2` is defined from. It is a copy of the release's own text, not a
list restated here.

### The install command

```bash
env -u TASK_BOARD_PROJECT_DIR -u TASK_BOARD_CONFIGURE_PROJECT \
    -u TASK_BOARD_BOOTSTRAP_LOCAL_AGENTS -u TASK_BOARD_MODE -u TASK_BOARD_DIR \
    -u TASK_BOARD_REMOTE -u TASK_BOARD_USER -u TASK_BOARD_ID \
    -u TASK_BOARD_TLS_NO_VERIFY -u TASK_BOARD_SETUP_SOURCE_ONLY \
  "$PM_RELEASE/scripts/setup.sh"
```

A **bare** machine-scoped install is mandatory. No flag is permitted:
`--configure-project` additionally enables `configure_project`,
`check_agents_infra_compose` and `bootstrap_local_agents` inside the same window
and writes a project `task-board.config.json` this plan does not model.

The `-u` list is every `TASK_BOARD_*` name in P2's task-board reference set.
`HOME`, `PATH`, `PWD` and `TMPDIR` are kept — the installer needs them, and
`BIN_DIR` is unconditionally assigned at `scripts/setup.sh:16` so it is not an
external input for this installer.

### In-flight runs

Provider children and already-exec'd CLI processes continue on their existing
images. A run that reaches a new CLI mutation during the mixed window may see a
partial default install; retry its mutation through the recovery-prefixed old
pair. Do not reparent, rewrite or kill the run.

**`tb-sessiond` does not follow that rule, and the operator does not decide it.**
Replacing the file leaves every running daemon alive, on its old image, holding
its own sessions. What happens next depends on P7c:

- **Protocol unchanged** — no daemon is restarted by anything. Each live daemon
  keeps serving its board from the *old* task-board image with the old embedded
  agents-management until it exits, which may be never within this migration.
  That is window `Wimg`, and P7's per-board disposition is what bounds it.
- **Protocol changed** — the first `ConnectOrStart`-class command the installed
  `0.25.0` runs against a board with an older daemon **restarts that daemon
  automatically**: `Drain` first, so a v2+ daemon's provider hosts survive, and a
  fenced SIGTERM/SIGKILL if it does not release its lock. Any goal-bound
  `task-board spawn`, `session doctor`, `session logs` or managed
  `codex`/`claude` wrapper is such a command, from any process on the host. P7
  refuses to open the window in this state, because from the first such command
  the rollback is one-way.

Note the asymmetry with the rest of this step: withholding the *default PATH*
protects new work this operator starts. It does not protect against another
agent, wrapper or human on the same host invoking the installed `task-board` by
its plain name. There is no interlock for that, and it is listed under
"Preconditions that are NOT IMPLEMENTED".

An interruption inside `install_go` changes nothing when P3 holds (see P3 for
the case where it does not). An interruption inside `install_skills` leaves a
managed skill tree or one of its two links in a partial state; recovery
authority for that surface is R1 steps 4-5, and the run is not resumed until the
R1 canary is green.

### Mixed-install window `W2`

Opens at the first `install_binary` call — **not** at the start of the script:
`install_go` precedes it and, under P3, mutates nothing. Closes when installed
verification plus the live canary succeeds, or when R1 plus the recovery canary
succeeds.

The stages inside it are the sequence in `w2-setup-main.txt`. For the source
this plan was derived against, that is: five `install_binary` calls,
`register_skill`, `install_roles`, `write_install_state`, `check_path`,
`configure_project` (returns immediately under the bare invocation),
**`install_skills`** (the registered skill plus every `SKILL_REPOS` dependency,
each with its `~/.claude` and `~/.codex` link) and `verify`.

Duration is measured as `W2_end - W2_start`, not assumed sub-second. Default-PATH
new spawn/route is withheld during `W2`; the absolute saved old pair remains the
routing authority.

### Rollback

Run **R1** with `RECOVERY_ROOT=…/current-pair` and
`PM_RELEASE=…/skill-project-management-0.25.0`, then the R1 canary.

Rollback status: **UNTESTED**.

## Step 3 — release and install agents-infra `v1.7.0`

Repository/version: `relux-agents-infra v1.7.0`, exact public
agents-management `v0.5.0`. task-board remains installed `0.25.0`.

Required board capability: `0.25.0` must preflight, spawn, compose through
agents-infra, attach outcome and route to a reviewer before and after install.
agents-infra full tests/build/verify plus real target/compose/Pi refusal gates
must pass on the exact release head.

### Preconditions, all outside the window

```bash
RECOVERY_ROOT="$HOME/.local/state/TASK-260830-s5ro4e/current-pair"
AI_RELEASE="$HOME/.local/state/TASK-260830-s5ro4e/release-candidates/relux-agents-infra-v1.7.0"
SAVED_SOURCE="$RECOVERY_ROOT/sources/relux-agents-infra"

test "$(git -C "$AI_RELEASE" describe --tags --exact-match)" = "v1.7.0"
```

| # | Precondition | Command | Expected |
| --- | --- | --- | --- |
| P1 | installer text unchanged | `diff -ru "$SAVED_SOURCE/scripts" "$AI_RELEASE/scripts"` | no output |
| P2 | environment surface unchanged | the agents-infra reference set diffed; plus the indirect-expansion probe | no output, both |
| P3 | `go` already resolves | as above | path, version, exit 0 |
| P5 | executable set matches | `diff` of the two `BINARY_NAME`/`MODEL_HARNESS_BINARY_NAME` values against the snapshot's agents-infra rows | no output |
| P6 | Homebrew/LLDB before-snapshot | the P6 block into `"$RECOVERY_ROOT/brew-before-step3.txt"` | non-empty file, exit 0 |

Record the stage sequence the release will actually run:

```bash
sed -n '/^install_go$/,$p' "$AI_RELEASE/scripts/setup.sh" \
  > "$RECOVERY_ROOT/w3-call-sequence.txt"
```

### The install command

```bash
AGENTS_INFRA_SKIP_LLDB_MCP=1 \
  env -u BIN_DIR -u AGENTS_INFRA_CONFIG_DIR \
  "$AI_RELEASE/scripts/setup.sh"
```

That is the whole invocation. No arguments: `--with-pdf-tools` is forbidden
because it runs a separate toolchain installer this plan does not model.
`BIN_DIR` must be unset so the install directory is the one both install-state
files record; `AGENTS_INFRA_CONFIG_DIR` must be unset so `resolve_config_dir`
picks the same path the snapshot saved.

`AGENTS_INFRA_SKIP_LLDB_MCP=1` is mandatory, not an operator option. Production
`install_lldb_mcp` returns at `:94-97` before `brew --prefix`, `brew install
llvm`, wrapper backup/removal or helper resolution. LLDB bootstrap is unrelated
to this contract migration and is deferred to a separate reviewed maintenance
operation.

### Immediately after the installer exits

```bash
# P6 after-snapshot and comparison
# … the P6 block into "$RECOVERY_ROOT/brew-after-step3.txt" …
diff "$RECOVERY_ROOT/brew-before-step3.txt" "$RECOVERY_ROOT/brew-after-step3.txt"
echo "exit=$?"
```

**Expected: no output, `exit=0`.** Retain both files, the installer's exit code
and the measured command duration with the release evidence. A difference, a
missing file or an empty file withholds every new default-PATH composition even
when the installer itself exited 0.

### In-flight runs

Provider children, primary sessions, Pi runtimes and existing LLDB MCP processes
retain old process images and leases. Do not restart them. New compositions,
including LLDB-enabled compositions, use the saved old agents-infra until the P6
comparison, installed verification and the canary all pass.

`tb-sessiond` is not in this release's artifact set, so Steps 3 and 6 neither
replace it nor trigger a protocol takeover, and P7 does not apply to them. What
does carry across: a live daemon resolves the programs it launches at exec time
through the `PATH` it inherited, so a child launched by an old daemon *after*
this step can pick up the new `agents-infra`. That combination is not proved by
anything in this step, which is why new composition is withheld until the canary
passes rather than only until the installer exits.

An interruption inside `install_go` changes nothing when P3 holds. An
interruption inside `install_binary` leaves either `agents-infra` or
`model-harness` replaced and the other not; `install_binary` uses a new-inode
atomic `mv`, so no destination is ever missing, and R2 restores both.

### Mixed-install window `W3`

Starts when `agents-infra` is replaced — `install_go` and the skipped
`install_lldb_mcp` both precede it and mutate nothing under P3 plus the skip
flag. Includes `model-harness`, install state, receipt invalidation,
`.agents` source sync, Claude/Codex links, helper/target launchers, runtime
verification and receipt write. Ends at installed canary success or R2 canary
success.

Record the actual duration. New default-PATH spawn/route is withheld;
recovery-prefixed old task-board plus old agents-infra remains available even if
the new `0.25.0` half must also be abandoned. A failed P6 comparison is not
folded into `W3` and is not treated as a successful agents-infra rollback.

### Rollback

Run **R2** with `RECOVERY_ROOT=…/current-pair` and
`AI_RELEASE=…/relux-agents-infra-v1.7.0`, then:

```bash
PATH="$RECOVERY_ROOT/bin:$PATH" \
  "$RECOVERY_ROOT/bin/task-board" q --format compact \
  'project_config(view=spawn-preflight, role=developer, agent=codex, task_class=research)'
# then the recovery-prefixed live spawn -> outcome -> handoff -> reviewer canary
```

If agents-infra restores but task-board routing is not green, also run R1. If
task-board restores first but agents-infra does not, keep
`PATH="$RECOVERY_ROOT/bin:$PATH"` and do not admit new default-PATH spawns until
R2 succeeds. **A half-restored pair is a recovery state, never a release
success.**

Rollback status: **UNTESTED**.

After Step 3 is green, capture the fresh immutable `bridge-pair` snapshot of
installed `0.25.0` and `v1.7.0` before any Step-5 mutation.

## Step 4 — allow conditional agents-management `v0.6.0`

Repository/version: `skill-agents-management v0.6.0` candidate.

Required board capability: installed `0.25.0`/`v1.7.0` continues all spawn and
route operations with embedded `v0.5.0`. The exact compatibility-removal gate
above must be green for both release tags and every removed item.

In-flight runs: unaffected; no installed binary resolves the new tag.

Mismatch window: none at runtime. Publishing `v0.6.0` does not repin either
consumer. The candidate is prohibited if any manifest item remains required or
unknown.

Rollback commands before publication:

```bash
cd "$CONSUMER_CANDIDATE"
GOWORK=off go mod edit \
  -require=github.com/relux-works/skill-agents-management@v0.5.0
GOFLAGS=-mod=mod GOWORK=off go mod tidy
GOWORK=off go test -mod=mod ./... -count=1
```

After a bad immutable tag is published, do not delete or move it. Keep consumers
on `v0.5.0` and publish only a separately reviewed forward `v0.6.1` that
restores the required contract; no consumer repin begins first.

Rollback status: **UNTESTED**; `v0.6.0` does not yet exist.

## Step 5 — release and install task-board `0.25.1`

Repository/version: `skill-project-management 0.25.1`, exact released
agents-management `v0.6.0`. agents-infra remains `v1.7.0` embedding `v0.5.0`.

Required board capability: full candidate and installed spawn/route canaries,
including the external agents-infra compose seam. The Step-5 intermediate pair
must be explicitly accepted; independent embedded module versions do not prove
the external seam by themselves.

Identical in shape to Step 2, with two substitutions:

```bash
RECOVERY_ROOT="$HOME/.local/state/TASK-260830-s5ro4e/bridge-pair"
PM_RELEASE="$HOME/.local/state/TASK-260830-s5ro4e/release-candidates/skill-project-management-0.25.1"
SAVED_SOURCE="$RECOVERY_ROOT/sources/skill-project-management"

test "$(git -C "$PM_RELEASE" describe --tags --exact-match)" = "0.25.1"
```

P1 through P5 and **P7** run exactly as in Step 2, and the install command is
byte-for-byte the Step-2 one with `$PM_RELEASE` repointed. The stage list goes to
`"$RECOVERY_ROOT/w5-setup-main.txt"`.

Because `$SAVED_SOURCE` here is the `bridge-pair` source — that is, `0.25.0` —
P1 compares `0.25.1`'s installer against `0.25.0`'s. A boundary change
introduced by `0.25.1` stops Step 5 the same way `0.25.0`'s would have stopped
Step 2.

In-flight runs: same process-image behaviour as Step 2, **including P7 and the
`tb-sessiond` exception in full**. Step 5 replaces `tb-sessiond` again, so the
census, the protocol derivation (this time `bridge-pair`'s saved `0.25.0` source
against the `0.25.1` release tree) and the per-board disposition are repeated,
not inherited from Step 2. A protocol that was unchanged at Step 2 says nothing
about Step 5. Their next CLI mutation may require replay through `bridge-pair`;
no run is killed or rewritten. An interruption inside `install_skills` has the
same recovery authority as in Step 2, against the `bridge-pair` snapshot.

Mixed-install window `W5`: same construction as `W2`, from the first
`install_binary` call through installed canary or R1 canary, including the
unconditional `install_skills` stage. Duration is recorded. Default-PATH new
spawn/route is withheld; `bridge-pair` is authority.

Rollback: run **R1** with `RECOVERY_ROOT=…/bridge-pair` and
`PM_RELEASE=…/skill-project-management-0.25.1`, then the R1 canary plus:

```bash
PATH="$RECOVERY_ROOT/bin:$PATH" \
  env -u AGENTS_INFRA_SOURCE_DIR "$RECOVERY_ROOT/bin/agents-infra" verify global
```

If only task-board restores, force the recovery PATH so it also resolves saved
`v1.7.0` agents-infra. If agents-infra verification is red, run R2 before
admitting new work. Do not claim rollback success from only one green half.

Rollback status: **UNTESTED**.

## Step 6 — release and install agents-infra `v1.7.1`

Repository/version: `relux-agents-infra v1.7.1`, exact released
agents-management `v0.6.0`. task-board remains `0.25.1`.

Required board capability: agents-infra full gates and exact installed
spawn/compose/outcome/handoff/reviewer canary, including successful reviewer
route. The accepted Step-5 pair remains available until the new pair passes.

Identical in shape to Step 3:

```bash
RECOVERY_ROOT="$HOME/.local/state/TASK-260830-s5ro4e/bridge-pair"
AI_RELEASE="$HOME/.local/state/TASK-260830-s5ro4e/release-candidates/relux-agents-infra-v1.7.1"
SAVED_SOURCE="$RECOVERY_ROOT/sources/relux-agents-infra"

test "$(git -C "$AI_RELEASE" describe --tags --exact-match)" = "v1.7.1"
```

P1, P2, P3, P5 and the P6 before-snapshot run exactly as in Step 3, into
`"$RECOVERY_ROOT/brew-before-step6.txt"`. The install command is the Step-3 one
with `$AI_RELEASE` repointed. The P6 after-comparison runs immediately after the
installer exits, into `"$RECOVERY_ROOT/brew-after-step6.txt"`.

Retain both files, the installer exit code and the measured duration. Any failed
read, missing file or difference is a failed step and preserves the bridge-pair
as routing authority.

In-flight runs: same as Step 3. Existing sessions, runtimes and LLDB MCP
processes are not restarted; new composition is withheld from the partial
installed surface until the P6 comparison and full canary pass.

Mixed-install window `W6`: same artifact sequence as `W3`, from the first
managed binary replacement through installed canary or R2 canary. Record actual
duration. `bridge-pair` remains the fail-closed authority.

Rollback: run **R2** with `RECOVERY_ROOT=…/bridge-pair` and
`AI_RELEASE=…/relux-agents-infra-v1.7.1`, then the recovery-prefixed canary.

If `0.25.1` plus restored `v1.7.0` fails the Step-5 seam canary, run **R1**
against `bridge-pair` too. If task-board restores first while R2 is red, use
only the full recovery-prefixed pair. Forward fixes use new immutable patch
releases (`0.25.2`, `v1.7.2` or later); never rewrite `0.25.1`, `v1.7.1` or
`v0.6.0`.

Rollback status: **UNTESTED**.

# Part V — Named unsupported and mixed windows

| Window | Duration | Impact | Survival/avoidance |
| --- | --- | --- | --- |
| Public `v0.5.0` while installed board embeds `v0.2.0` | `0s` dynamic mismatch | None from publication alone. | Exact immutable embedded pin. |
| `W2`: task-board `0.25.0` install | First `install_binary` call to installed canary or R1 canary; measured, unbounded in advance | Default surface may mix every executable, the registered skill, every `SKILL_REPOS` dependency with both its `~/.claude` and `~/.codex` links, roles and install state that the release's own `setup_main` installs; new default spawn/route may fail. | Withhold default surface; P4 materialises dependency skills before the window so no clone branch is reachable inside it; P3 removes the `install_go` brew branch; saved current-pair with recovery PATH remains authority. |
| Step-3 Homebrew/LLDB bypass guard | P6 before-snapshot through a successful after-comparison; measured | No accepted mutation. A failed read or difference makes Homebrew/LLDB state unknown and may affect LLDB-enabled composition. | Mandatory `AGENTS_INFRA_SKIP_LLDB_MCP=1` plus P3; the snapshot covers the whole formula list, so a `brew install` by any stage is visible; stop on any difference. |
| `W3`: agents-infra `v1.7.0` install | First managed binary replacement to installed canary or R2 canary; measured | Every executable the release's `BINARY_NAME`/`MODEL_HARNESS_BINARY_NAME` name, the machine-scoped install state in the resolved config dir, `.agents`, links/helpers and receipt may mix; new composition may fail. | Saved current-pair; existing children/sessions keep process images; retry later mutations through the saved pair. |
| `0.25.0` installed while `v1.7.0` is not yet installed | Until Step 3 canary | Supported only if the Step-2 live external-seam canary passed. | An explicit tested pair, not inferred independence. |
| Public `v0.6.0` while both installed consumers embed `v0.5.0` | `0s` dynamic mismatch | None from publication alone. | Exact pins; no `@latest`. |
| `W2d`: daemon protocol skew after `0.25.0` is installed, **only if `0.25.0` changes `controlProtocolVersion`** | Opens when the new `tb-sessiond`/`task-board` pair is on disk. Closes **per board**, when that board's daemon is next reached by a `ConnectOrStart`-class command — an interval with no upper bound, since a board nobody touches keeps its old daemon indefinitely | Each such board's daemon is restarted without operator action: drained if it is v2+, fenced with SIGTERM/SIGKILL otherwise. **Until then nothing visibly fails** — this cell said the opposite in revision 8 and contradicted its own "restarted without operator action" in the same sentence. In this direction the CLI is the newer side, `managerProtocolCompatibility` returns the *typed* `ManagerProtocolUpgradeError`, and `ConnectOrStart` handles it by taking over and succeeding; the `Dial`-based commands (`session status`, `list`, `stop`) never check compatibility at all. The window's impact is **silence**: a takeover the operator did not ask for and is not told about. **After the takeover, R1 cannot restore that board** — and not because the restored CLI refuses the newer daemon. It does not refuse it: it drives it unchecked or fences it, decided by that daemon's `startup_healthy`. See Part I. | **Not survived — avoided.** P7c derives the constant on both sides before the window and stops the step if it changed. If it must change, every live daemon is quiesced and stopped by its owner before the window opens, so no board carries an older daemon into it. There is no third option, and there is no rollback. |
| `Wimg`: daemon image skew, **whether or not the protocol changes** | Opens when `tb-sessiond` is replaced. Closes per board when that board's daemon next exits. Unbounded; on this host it is already open, with daemons 60 and 48 commits behind the installed file, 8 and 5 days old | Each live daemon keeps executing the old task-board image and its old embedded agents-management while the CLI on disk executes the new one. Protocol equality proves the transport is compatible and proves nothing about how the two embedded contract versions compose across it. Sessions held: 31 and 2 at planning time. | P7's recorded per-board disposition: the board is quiesced and its daemon stopped by its owner outside the window, or it is explicitly excluded from goal-bound and writable-context spawn until its daemon exits. A board with a live daemon and no recorded disposition stops the step. |
| `W5`: task-board `0.25.1` install | First `install_binary` call to installed or R1 canary; measured | Same task-board surface as `W2`, including the unconditional `install_skills` stage; new default spawn/route withheld. | Saved bridge-pair authority, the same P3/P4 preconditions, and the full R1 rollback. |
| `0.25.1/v0.6.0` with agents-infra `v1.7.0/v0.5.0` | Until Step 6 canary | Cross-version embedded libraries behind one external seam. | Allowed only after the Step-5 real spawn/compose/route canary. |
| Step-6 Homebrew/LLDB bypass guard | P6 before-snapshot through a successful after-comparison; measured | As Step 3. | As Step 3. |
| `W6`: agents-infra `v1.7.1` install | First managed binary replacement to installed or R2 canary; measured | Same mixed runtime surface as `W3`; new default composition withheld. | Saved bridge-pair; restore one half, canary, then restore both if the intermediate pair is red. |

The forbidden window is running an installer from a breaking consumer candidate
before its exact tests, removal manifest, out-of-place canary and recovery
snapshot are green. Its impact would be loss of new spawn/route authority for an
unbounded time. This plan never accepts that mode.

`W2d` and `Wimg` repeat identically for Step 5, against the `bridge-pair`
snapshot and the `0.25.0`-to-`0.25.1` protocol comparison. They are not listed
twice; the Step-5 section says P7 is re-run rather than inherited.

Three windows that earlier revisions did not name, and where they are now:

- **`install_go`, both installers, before every managed binary.** Not a window
  under P3, because its only reachable branch is a read. Without P3 it is an
  unbounded Homebrew mutation inside the release window with no snapshot and no
  restore — which is what revision 6 shipped.
- **The `install_skills` clone branch, task-board.** Not a window under P4,
  because every managed skill is already a real directory. Without P4 it is a
  network fetch of an unrelated skill at an arbitrary remote HEAD, whose failure
  mode deletes the skill.
- **`tb-sessiond`'s lifetime, both task-board steps.** Named above as `W2d` and
  `Wimg`. Revisions 1-7 mentioned `tb-sessiond` only as a name in a derived
  executable list and a `--version` line, and asserted that its processes
  "continue" — which production overrides. It is the one surface in this plan
  where a window is opened by a *process*, not by a file, and where one of the
  two directions has no rollback at all.

# Part VI — Validation evidence from this planning task

No migration, install, release or tag was performed. Both installers were read,
never executed.

## Evidence retained from earlier revisions

These gates concern the consumption census and the module graph, which this
revision did not change. They were run in earlier revisions and, where noted,
independently reproduced by review.

| Command/gate | Exit | Result |
| --- | ---: | --- |
| Tool readiness (`task-board`, Git, Go, `rg`, `nm`) | 0 | Required tools executable. |
| `go mod download -json ...@v0.5.0` | 0 | Exact origin commit and checksum reproduced. |
| `GOFLAGS=-mod=mod GOWORK=off go mod tidy` | 0 | Correct scratch pin completed. |
| `GOWORK=off go list -mod=mod -deps ... ./...` | 0 | 20 compiled module packages recorded. |
| `GOWORK=off go build -mod=mod -o ... .` | 0 | Candidate binary built. |
| `go version -m` / `go tool nm` | 0 / 0 | Exact `v0.5.0` checksum and 555-symbol linked surface recorded. |
| `GOWORK=off go test -mod=mod ./internal/spawn -count=1` | 0 | Unmodified reviewer-required gate green in a real detached Git worktree. Reproduced by the revision-2 review. |
| Candidate production `project_config(view=spawn-preflight, ...)` | 0 | Real candidate init/model/preflight path reached the authoritative board. |
| Targeted child/Codex/Claude stdio-MCP composition and invalid-config refusal tests | 0 | Five production-surface tests green with `-count=1`. |
| `go test ./internal/infra -run 'TestSetupGlobal\|TestVerifyInstalledRuntimeRefusesIncorrectGlobalPiInfraTarget' -count=1` | 0 | Exact setup/verify and negative runtime-target package tests green. |
| `go build ./...` in `tools/agents-infra` | 0 | Go CLI still compiles on the supported host platform. |
| Production setup without skip against a fake Homebrew (revision 3) | 0 | Negative probe changed the fake wrapper SHA-256 `0a5fb2d3…` to `1de8aa42…`, proving the pre-binary LLDB mutation path is reachable. |
| Byte-identical `4270549` binary + exact-source sandbox `setup global` / `verify global` (revision 4) | 0 / 0 | A coherent recovery pair installs and verifies in a disposable HOME; the real installed runtime was untouched. |

## Evidence for this revision

Every command below was run read-only against the real production installers at
`skill-project-management` `f1319eff` and `relux-agents-infra` `bb857fe5`, and
against the real host. Each row is the exact command this plan tells the
operator to run.

Validator: `.temp/TASK-260830-s5ro4e-rev9/validate-revision9.sh`, run under both
`bash` and `zsh`. Logs: `.temp/TASK-260830-s5ro4e-rev9/val-bash.log`,
`val-zsh.log`, attached as
`TASK-260830-s5ro4e_validate-revision9-{bash,zsh}.log`. Probe counts are in the
logs and in the board notes, not restated here where they would go stale.

The suite locates every block it drives — R1, P4a, P4b, P6, P7a and the snapshot
— inside this document **by content**, not by line number, so a shifted offset
cannot silently point it at the wrong block or at nothing. Where a section
compares against a *previous* revision's text, that text is recovered **by
commit SHA** and extracted the same way, so "red on the shipped text" is measured
rather than asserted. Revision 8's validator read it from `HEAD`, which went red
the moment revision 8 was committed and could not be re-run by a reviewer — the
revision-8 F2 finding. Section J now pins `0425fb7` and fails closed if that
object cannot be read.

Every shell fence in this document was extracted mechanically
(`.temp/TASK-260830-s5ro4e-rev9/extract-fences.py`) and syntax-checked: `bash -n`
and `zsh -n` on every `bash` fence, `zsh -n` on every verbatim-production `zsh`
fence. All of them parse in every applicable shell. The `zsh` fences parsing is what
confirms the production excerpts in Part I are quoted faithfully rather than
paraphrased.

### A — the plan's recorded derivations, reproduced against production text

Each row is the exact command Part I or Part II gives the operator, run against
the real `scripts/setup.sh` of both repositories.

| # | Check | Result |
| --- | --- | --- |
| A1-A2 | task-board managed executables | 5: `task-board tb-sessiond task-board-tui openai-board anthropic-board` |
| A3 | agents-infra managed executables | 2: `agents-infra model-harness` |
| A4 | `SKILL_REPOS` keys | 8 |
| A5 | roles in the release tree | 9, equal to installed `~/.roles` |
| A6 | task-board environment reference set | 38 names |
| A7 | agents-infra environment reference set | 19 names |
| A8 | that set contains `HOME`, which the task-board installer reads only as bare `$HOME` | yes |
| A9 | the agents-infra set contains `AGENTS_INFRA_SKIP_LLDB_MCP` (only ever `${X:-0}`) and `BIN_DIR` (only ever `${BIN_DIR:-…}`) | yes |
| A10-A11 | indirect-expansion probe, both installers | exit 1, no match — the sigil match is complete for today's text |
| A12 | `source` lines in the task-board installer | exactly 1 |
| A13 | `install_skills` appears ungated in `setup_main` | yes |
| A14 | the first stage in `setup_main` is `install_go`, not `install_binary` | yes |
| A15 | the agents-infra call sequence is `258:install_go` then `259:install_lldb_mcp` | yes |

A14 and A15 are the measurements revision 6 did not take, and they are why `W2`,
`W3`, `W5` and `W6` can honestly open at the first managed-binary replacement:
the stage that precedes it mutates nothing once P3 holds.

### B, C — each replacement attacked against the guard it replaced

Every row in B and C runs the **revision-6 guard verbatim** and the
**revision-7 precondition** against the same mutant, on real production text. A
row is only meaningful because the revision-6 column is a fail-open there.

**D is different, and revision 7 described it wrongly.** Its three rows compare
`printf`-written fixtures, not the document's P6 script: they show what the two
*shapes* of snapshot can and cannot see, which is an illustration rather than a
measurement of anything that ships. The substantive property — that the block in
this document sees a formula-set change on the real host — is established in
section H, where the P6 fence is extracted from this markdown and executed
against the real Homebrew installation, and H9 is the row that measures it. The
rows below are relabelled accordingly.

| # | Mutant applied to real production text | rev-6 guard | P-check |
| --- | --- | --- | --- |
| B1/B2 | `write_install_state` extended *after* the heredoc's column-0 `}` — revision-6 F2's own example | `MIRROR_UNCHANGED`, **exit 0, admitted** | `diff -ru` prints the hunk, **refused** |
| B3/B4 | top-level `AGENTS_SKILLS_DIR` relocated; no function body changed | all four mirrors **exit 0, admitted** | **refused** |
| B5 | *no mutation* (narrowing control) | — | **exit 0, admitted** |
| C1 | `if [ -n "$TASK_BOARD_FUTURE_FLAG" ]` — bare `$VAR` | **admitted** | **refused** |
| C2 | `OTHER_FLAG=${OTHER_FLAG-0}` — `${X-}` | **admitted** | **refused** |
| C3 | `: ${THIRD_FLAG:=on}` — `${X:=}` | **admitted** | **refused** |
| C4 | `FOURTH=${FOURTH_FLAG:-x}` — `${X:-}` (the one shape rev 6 saw) | **refused** | **refused** |
| C5 | *no mutation* (narrowing control) | — | **admitted** |
| C6 | injected `eval "$SOMETHING"` | n/a | indirect probe **fires** |
| D1/D2 | **fixture**: a formula appears in a `printf`-written formula list | a rev-6-*shaped* five-value record **compares identical** | a formula-list-*shaped* record **differs** |
| D3 | *no mutation* (narrowing control) | — | **identical** |

B5, C5 and D3 are the narrowing controls. Without them B and C would prove only
that the new checks can fail, not that they discriminate. D's control makes the
same point about the two shapes, and nothing more.

### E — P3, and P4 driven as this document's own fences

Revision 7 titled this "P3 and P4 driven" while P4 was a `p4_check` function the
validator defined itself — a mirror of the classification, not the print loop
this document actually gives the operator, and the manual step the document
requires was never exercised. Revision 8 locates **both** P4 fences by content
in the extracted fences and runs them verbatim.

| # | Condition | Result |
| --- | --- | --- |
| E1 | `go` absent from `PATH` | P3 **refuses** (exit 1) |
| E2 | the real target host | P3 **satisfied** (exit 0) |
| E0a/E0b | both P4 fences located by content | located |
| E3a-E3b | the **P4a fence**, run verbatim against the real release tree | exit 0, 8 `SKILL_REPOS` keys derived |
| E3c | **the operator's manual append** of `project-management`, which revision 7 never exercised | 9 names |
| E4-E5 | the **P4b fence**, run verbatim, all nine skills real directories | 9 `OK`, **0** `REFUSE` |
| E6-E7 | one skill is a symlink — the installer would `rm` it and clone | 1 `REFUSE`, and it names the shape it saw (`symlink`) |
| E8-E9 | one skill is absent — the installer would clone from the network | 1 `REFUSE`, named `absent` |
| E10 | one skill is a plain file | 1 `REFUSE` |
| E11-E12 | the same skill re-materialised as a directory | 0 `REFUSE`, 9 `OK` — narrowing control: P4 refuses the clone branch without refusing a legitimate installed skill |

### F — recovery-baseline identity

| # | Input | Result |
| --- | --- | --- |
| F1 | `task-board version 0.24.3-172-g063197b1 (commit 063197b1, built …)` | `063197b1` |
| F2 | `agents-infra v1.6.1-103-g4270549 commit=4270549 build_date=…` | `4270549` |
| F3 | an unparseable version string | empty — the snapshot's `test -n` then refuses rather than materialising a guessed worktree |

### H — the P6 script, run read-only against the real Homebrew installation

Extracted from this document and executed. It only reads: `brew --prefix`,
`brew list --formula`, `brew list --versions llvm`, `brew --prefix llvm`,
`stat` and `shasum`. Nothing was installed, upgraded or removed.

| # | Condition | Result |
| --- | --- | --- |
| H1 | fresh run on the target host | exit 0 |
| H2 | the snapshot covers the whole formula list, not five LLDB values | 237 lines (228 formulae + prefix + llvm version + prefix + 4 target records) |
| H3 | two consecutive clean runs | compare identical — narrowing control, so H9 is a real signal |
| H4-H5 | run against an existing destination | **refused**, and the earlier good snapshot survived |
| H6-H8 | run with `brew` unreachable on `PATH` | **refused** (127), published nothing, left no `.partial` that could be mistaken for a snapshot |
| H9 | one formula removed from the recorded set | visible difference |

H6-H8 are the revision-5 lesson re-checked on the new construction: a failed
read must not become a measured absence, and a partial must never be comparable.
H4-H5 close a hazard the block had in its first draft — it deleted `$out`
before reading, so a failed re-run destroyed the snapshot it was meant to be
compared against.

Measured on the real host during H1: wrapper `/opt/homebrew/bin/lldb-mcp`
SHA-256 `1ea8dd5a…`, backup `.agents-infra.bak` SHA-256 `0a0c5f2c…`, both
llvm-prefixed targets `ABSENT`. The wrapper hash matches the value recorded in
document revision 3, which is independent confirmation that this surface has not
drifted during the planning task.

### G — R1 executed verbatim from this document

The R1 block is located in the extracted fences **by content** — `grep -l
'r1-release-bins'` — and run with `bash`, not retyped, against a disposable
`HOME` holding fake artifacts. Locating by content is what stops a line-number
shift in this document from pointing the suite at the wrong block or at nothing;
revision 7's prose said "by line offset", which was not what its own validator
did. **No installer was executed.** The simulated release adds an executable, a
role and a skill, and rewrites an existing dependency skill and an existing
role.

| # | Assertion | Result |
| --- | --- | --- |
| G1 | R1 exits 0 on a well-formed snapshot | pass |
| G2-G3 | overwritten `task-board` and `tb-sessiond` restored to the snapshot bytes | pass |
| G4 | the executable the release **added** is quarantined | absent from `BIN_DIR` |
| G5-G6 | an unrelated operator-owned binary is **not** removed, and is reported in `unexpected-bin-entries.txt` | pass — the rollback owns the release's artifacts, not the operator's PATH |
| G7 | a dependency skill the release **rewrote** is restored to `v1` | pass |
| G8-G10 | the skill the release **added** is removed from `~/.agents`, `~/.claude` and `~/.codex` | pass |
| G11 | a role the release rewrote is restored | pass |
| G12 | an untouched role survives — narrowing control | pass |
| G13 | the role the release added is removed | pass |
| G14 | the machine-scoped install state's `repoPath` returns to the saved value | pass |
| G15 | a snapshot skill recorded as `symlink` is restored as a symlink, not a copy | pass |
| G16 | an unrecognised kind in `skill-state.txt` **aborts R1 non-zero** | refused |
| G17 | R1 re-run on an already-restored state exits 0 — so G16's non-zero is attributable to the bad kind, not to non-idempotence | pass |
| G18-G19 | a **missing snapshot executable** stops R1 non-zero, and R1 does not report success while leaving the release binary in place | refused; `BIN_DIR` still holds the release binary, correctly, because R1 stopped. In revision 8 it stops in step 0, before any mutation — section J measures that difference directly |
| G20-G21 | R1 green again once the snapshot is complete, and the executable is restored — narrowing control | pass |

G17 and G20 are what make G16 and G18 evidence rather than noise: without them a
non-zero exit could be R1 being generally broken.

### I — the `tb-sessiond` contract, derived from production text and the live host

Every row is a read. No process was signalled, started or stopped; the two live
daemons were observed and left alone.

| # | Check | Result |
| --- | --- | --- |
| I1-I2 | `controlProtocolVersion` is a compile-time constant, and its value | one match, `4` |
| I3 | `task-board --version` does not report it — so it can only come from source | 0 mentions |
| I4 | the daemon publishes it as `protocol_version` on its status endpoint | present in `ManagerStatus` |
| I5-I7 | `managerProtocolCompatibility` has an older-daemon branch, a newer-daemon branch, and the newer branch is a bare refusal | all three present — **presence only; see section L for whether it runs** |
| I8 | `takeOverOutdatedManager` refuses the downgrade direction in words | *"refusing protocol takeover without an older daemon version"* |
| I9 | the takeover is reached from `ConnectOrStart` itself, not from a helper | present in `ConnectOrStart`'s body |
| I10-I11 | the takeover drains first and falls through to a fenced terminate | both present |
| I12 | **no cobra command reaches `Drain`** — so no operator can drain a daemon by hand | 0 files under `cmd/` |
| I13-I14 | `Dial` performs no compatibility check, and `session status` uses it | 0 / 1 |
| I15 | the response decoder is lenient — no `DisallowUnknownFields` | 0 |
| I16-I18 | the daemon is resolved through `PATH` by the bare name `tb-sessiond`, and its board comes from its own argv | all three |
| I19-I20 | goal-bound spawn reaches the daemon through `ConnectOrStart` | both present |
| I21 | the constant's four historical values, from `git show` at each touching commit | `4 3 2 "task-board-session-manager-v1"` |
| I22 | every live daemon answered `session status` | 2 of 2 |
| I23 | both report `protocol_version` 4 | 2 |
| I24 | **neither daemon's image equals the installed file's version** | 0 matches |
| I25 | both run some older `0.24.3-1…` image | 2 |
| I26 | and they hold sessions a takeover would drain or fence | 33 total |
| I27-I29 | this migration's own board: exit 0, `running:false`, no `protocol_version` | a **legitimate absence**, distinguishable from a failed read — and the reason an installed canary here observes neither live daemon |

I24 and I27-I29 are the two rows that make this section evidence rather than
description. The first says the skew is not hypothetical on this host; the
second says the canary the plan already had could not have seen it.

**What this section did wrong in revision 8, stated plainly.** I5-I7 asserted
that the newer-daemon branch *exists*. I9 asserted that the *older*-daemon
takeover is reached from `ConnectOrStart`. Nothing asked whether the newer branch
is reached from anything — and the plan then used I5-I7 as if it were a
guarantee that an older CLI cannot touch a newer daemon. That is the standard
negative shape **"the check is present but uncalled from production"**, and this
document has now hit it in three separate places (the engine-kind contract, the
External-CI policy gate, the comparison instrumentation) before hitting it here.
A presence probe for a guard is not evidence the guard runs. Section L is the
reachability probe that should have accompanied I5-I7 from the start.

The same asymmetry exists in the product's own tests:
`TestConnectOrStartUpgradesHealthyOutdatedProtocolHolder` and
`TestConnectOrStartTakesOverHolderWithoutStartupHealthy` cover the older
direction and the terminate branch; **no test file under
`internal/sessionmanager` names the newer-than-wrapper refusal at all** (L21).

### L — is the newer-daemon refusal reachable from production? Measured, not assumed

Every row is a read of production text at `skill-project-management` `f1319eff`.
No process was signalled, no protocol mismatch staged, no second image built.
The conclusion rests on call sites and their arguments, not on an observed
takeover — the same standard section I uses, and the limit is stated in "What
this revision did not do".

| # | Check | Result |
| --- | --- | --- |
| L1 | the newer-daemon branch returns a bare `fmt.Errorf`, **not** a typed error | 1 |
| L2 | while the older-daemon branch returns `*ManagerProtocolUpgradeError` | 1 |
| L3 | `ConnectOrStart` inspects the compat error with exactly one `errors.As` | 1 |
| L4 | and performs no other `errors.As`/`errors.Is` on it | 1 total |
| L5 | so the untyped newer error matches nothing and the `else` block returns nothing for it | 2 `return nil, upgrade`, both inside the typed arm |
| L6 | the only `launch == nil` early return is `ErrManagerNotRunning`, which is not the refusal | 1 |
| L7-L9 | every production `ConnectOrStart`/`…AndAttach` caller passes a **non-nil** launch closure — `cmd/session.go:366`, `cmd/codex_manager.go:92`, `cmd/claude_manager.go:115` | 3 of 3 |
| L10 | so that arm is dead in this binary and control reaches `takeOverUnresponsiveManager` | 2 call sites in `ConnectOrStart` |
| L11 | `managerProtocolCompatibility` has exactly two production call sites | 2 |
| L12 | **neither is inside `takeOverUnresponsiveManager`** | 0 |
| L13 | which dials with `dialManagerStatus`, the unchecked variant | 1 |
| L14 | and `dialManagerStatus` contains no compatibility check | 0 |
| L15 | a `StartupHealthy` holder is returned to the caller as a live client | 1 |
| L16 | a non-`StartupHealthy` holder falls through to `terminateStaleInstance` | 1 |
| L17 | which signals the process: SIGTERM then SIGKILL | 2 |
| L18 | and `ConnectOrStart` returns that unchecked healthy client directly | 2 |
| L19 | `sessionmanager.Connect`, the launch-less compat-checked entry, has **zero** direct call sites under `cmd/` | 0 |
| L20 | it reaches production once as a function value, at `cmd/context_security.go:75` | 1 |
| L20a | that value is invoked inside `FinalizeAndRequireBuilderCoverageEvidence`, whose `default:` arm turns the refusal into a `builder_trace_unavailable` `ContextError` | both present |
| L20b | reached only from `context publish` / the mutate enforcement hook, not from any spawn or route command | 2 callers, neither a session command |
| L21 | test files naming the newer-than-wrapper refusal | **0** |
| L22-L23 | while the older direction and the terminate branch each have a dedicated test — narrowing control, so L21 is a real gap and not a grep artefact | 1 and 1 |
| L24 | `takeOverManagerForProtocolUpgrade` **does** run the check and **does** surface the untyped refusal — the one `ConnectOrStart`-adjacent place it is reachable | 1 |
| L25 | but only from `ConnectOrStartAndAttach`, after `ConnectOrStart` already returned the unchecked client and `Attach` was rejected | reached via `errors.Is(err, ErrManagerProtocolUpgradeRequired)` |
| L26 | and it is wrapped in *"…retry the wrapper"*, advice that cannot succeed | 1 |
| L27 | the plan now names the mechanism it previously did not: `takeOverUnresponsiveManager`, `StartupHealthy`, `startup_healthy`, `dialManagerStatus`, `launch == nil`; drops both false claims; states the prohibition as the whole `ConnectOrStart` family; records the owning-repo prerequisite | 9 of 9 |
| L28-L28f | **narrowing control, pinned at `cbe69f3`:** every one of those nine rows run against **revision 8's shipped text** | **9 of 9 red** — five terms absent, both false claims present, no prohibition, no prerequisite. So L27 measures the correction and not the grep |

L12 is the row the whole correction turns on. L15 and L16 are the two outcomes,
L21-L23 are why neither was caught, L24-L26 say where the refusal *is* reachable
so that "unreachable" is not overclaimed, and L28 is the control that makes L27
attributable — the row revision 8's own section I never had for I5-I7.

### J — the recorded-but-unrestorable kind, attacked against the shipped text

The control is **revision 7's own R1**, recovered from the committed document at
`HEAD` and extracted the same way as revision 8's, so "red on the text that
shipped" is measured. Every fixture differs from its control in exactly one
recorded field.

| # | Input | Revision-7 R1 | Revision-8 R1 |
| --- | --- | --- | --- |
| J1-J4 | `~/.claude` kind recorded `directory`, nothing staged | **aborts non-zero — after mutating.** Executables already restored; steps 6 and 7 never ran, so the role and the install state are still the abandoned release's | — |
| J5-J8 | the same snapshot | — | **refused in step 0**, executables and role untouched, and it names the entry: `UNRESTORABLE swiftui:.claude directory with no staged tree` |
| J9-J12 | the same kind, with the tree staged by the revision-8 snapshot | — | **exit 0**; the link-tree entry is a directory again, not a symlink, with the snapshot's content, and the rest of the rollback completed — narrowing control |
| J13-J17 | `symlink` kind with an **empty** recorded target | **exit 0 — reports success.** `ln -sfn ""` succeeds, so R1 `rm -rf`s the real tree and leaves a dangling symlink to nowhere | **refused in step 0**, nothing mutated |
| J18-J20 | a **saved role missing** from the snapshot | aborts non-zero after restoring the executables | **refused in step 0**, executables untouched |
| J21-J22 | a well-formed snapshot | — | **exit 0** and restored — narrowing control, so the four refusals are attributable to the input |
| J23-J24 | the snapshot block and R1 both define `snapshot_restorable` and both handle the two link trees | text check, not an execution | both |

**J13-J17 is a finding revision 8 made, not one the review reported.** The
review found that a recorded `directory` kind *aborts* R1. A recorded `symlink`
kind with no target does something worse: it **fails open**. `ln -sfn "" path`
exits 0, so revision 7's R1 destroys the tree it was restoring, creates a
dangling link, and reports a successful rollback. The same defect class, one
arm over, and the reason the fix is an admissibility pass over every kind rather
than a staged tree for one of them.

### K — the daemon census, driven as this document's own P7a fence

P7a's whole point is that a failed process census must not be read as "no
daemons". The fence is located by content and driven with a stub `pgrep` whose
exit code is the only variable.

| # | `pgrep` exit | Result |
| --- | ---: | --- |
| K1-K2 | 0, two daemons | admitted, both lines recorded |
| K3-K4 | 1, no match | **admitted as a legitimate absence**, recorded `NO-LIVE-DAEMON` |
| K5-K7 | 2, usage error | **refused**, no `NO-LIVE-DAEMON` published over the failure, and it names `DAEMON-CENSUS-FAILED rc=2` |
| K8 | 3, fatal | **refused** — the class, not one code |
| K9-K11 | the real `pgrep` on the target host | admitted, 2 live daemons seen, 2 board paths extracted from their own argv — narrowing control |

K3 and K5 are the pair that matters: the same command, one exit code apart, and
the plan treats them as opposite facts.

### The three revision-6 findings, and why the replacements cover them

**F1 — `install_go` unmodelled.** Confirmed against production: defined at
`relux-agents-infra/scripts/setup.sh:75-91` and called at `:258`, one line
before `install_lldb_mcp` at `:259`; defined at
`skill-project-management/scripts/setup.sh:131-146` and called from `setup_main`
at `:904`, before every `install_binary`. Neither has a skip flag. Now covered
three ways: Part I names it and both call positions; **P3** makes its
non-mutating branch the only reachable one; **P6** snapshots the whole
`brew list --formula` output, so a `brew install go` is a visible difference
rather than an invisible one. The old guard's target set — five LLDB-specific
values — could not see it, which is the finding.

**F2 — truncated function comparison.** The extractor is gone. `diff -ru` over
the whole `scripts/` tree cannot truncate, and it covers what no function-body
mirror could: the top-level `AGENTS_SKILLS_DIR`/`CLAUDE_SKILLS_DIR`/
`CODEX_SKILLS_DIR` assignments at `:816-818`, `BIN_DIR` at `:16`, the
`SKILL_REPOS` (`:822-831`) and `SKILL_NESTED` (`:834-838`) maps, and any file
newly added to `scripts/`.
Directly on the finding's own example: revision 6's guard reported
`MIRROR_UNCHANGED|write_install_state` and exit 0 for a mutated
`write_install_state`; `diff -ru` prints the changed hunk, which is what stops
the step.

**F3 — one override shape recognised.** The classifier is gone. `\$\{?[A-Z]…`
matches the `$` sigil, so `$X`, `${X}`, `${X:-}`, `${X-}` and `${X:=}` are all
in the set. Verified on production text: the agents-infra set contains
`AGENTS_INFRA_SKIP_LLDB_MCP`, which reaches the file only as `${X:-0}` at `:94`,
and `BIN_DIR`, which reaches it as `${BIN_DIR:-…}` at `:10` — the exact case
where "filter out names the script assigns itself" produced revision 6's wrong
answer. The reference set is deliberately unfiltered and diffed against the
saved source, so no such judgement is needed. Its residual blind spots
(`printenv`, `eval`, indirect expansion) are named in P2 and probed by a second
command that returns empty on both installers today.

### The three revision-7 findings, and where each is closed

**F1 — `tb-sessiond` modelled as a binary, not a daemon.** Part I now models it:
the constant, both compatibility branches, the automatic takeover reached from
`ConnectOrStart`, the drain-then-fence sequence, the `PATH` resolution, the
protocol's four-value history, and the measured live census. **P7** enumerates
the daemons and derives the constant on both sides before the window, with a
`pgrep` status discrimination proved in section K. **Part V** names `W2d` and
`Wimg` with their real duration semantics — per board, unbounded, closing when
that board's daemon is next reached or next exits. **R1** gains a daemon census
and a canary that reaches the session manager instead of a config reader that
could go green over the failure. The downgrade direction is stated as what
production says it is — but **revision 8 stated it wrongly and revision 9
corrects it**: `managerProtocolCompatibility` does contain the refusal,
`takeOverOutdatedManager` does refuse the downgrade, and no cobra command reaches
`Drain` (I12), yet none of that is reached from any production `ConnectOrStart`
call site, so the direction is *unprotected* rather than refused (section L). It
is still not recoverable, the point of irreversibility is still named, and the
answer is still prevention rather than a rollback step that would not work.

**F2 — a recorded kind with nothing to restore it from.** The snapshot stages
both link trees; R1 refuses before it mutates; section J drives the shipped
revision-7 text and the revision-8 text against the same fixtures and measures
the difference. Working the finding also found the fail-open one arm over
(J13-J17), which is why the fix covers the class.

**F3 — evidence describing a reimplementation as a run.** Section D says what it
is; section E drives the document's own P4 fences including the manual append;
section G's prose matches its validator. Where revision 8 still checks text
rather than execution — J23-J24 — the row says "text check, not an execution".

### The two revision-8 findings, and where each is closed

**F1 — the newer-daemon refusal is unreachable from production.** Confirmed
independently against `f1319eff` and closed in six places, exactly as the review
required. Part I's "Daemon newer than CLI" paragraph now describes the three
branches an older CLI actually takes and names `client.go:145-166`,
`stale_takeover.go:168-266` and the dead `launch == nil` arm. "The point of
irreversibility" keeps its conclusion and replaces its reason: the state is
one-way because the restored CLI will *drive or fence* that daemon without a
check, not because it cannot touch it. P7's "Different" disposition item 2 and
R1 step 8's `protocol_version`-greater bullet are rewritten, and the prohibition
widened from the spawn canary to the whole `ConnectOrStart` family — goal-bound
`spawn`, `session doctor`, `logs`, `reclaim-contexts`, `session context` and the
managed wrappers, recovery-prefixed or not. Part V's `W2d` impact cell is
corrected in the other direction too: in that window nothing fails, and the
impact is silence. Section **L** adds twenty-seven reachability probes next to
I5-I7. And the missing guard is recorded as a **named prerequisite for
`skill-project-management`** in the NOT IMPLEMENTED table, not written into this
document's prose — this plan cannot add behaviour to a binary it only describes.

Working the finding produced two facts the review did not have, both of which
narrow rather than widen the claim. The refusal **is** reachable from exactly two
production paths — `sessionmanager.Connect` as a function value at
`cmd/context_security.go:75`, where it surfaces as a `builder_trace_unavailable`
`ContextError` on the `context publish` path, and
`takeOverManagerForProtocolUpgrade` from `ConnectOrStartAndAttach`, but only
*after* `ConnectOrStart` already handed back the unchecked client. Neither is a
spawn or route command and neither prevents the unchecked client, so the
conclusion holds; "unreachable from every production `ConnectOrStart` call site"
is the accurate form, and L19-L20b and L24-L26 record both.

**F2 — section J read revision 7 from `HEAD`.** Closed by pinning the comparison
to `0425fb7` by SHA. The validator now fails closed if that object cannot be read
rather than silently comparing the document with itself, and section J is
reproducible from any later checkout.

### What this revision did not do, stated as unknown

- **R1 was exercised, not rehearsed.** Section G ran the R1 block against a
  disposable `HOME` holding *fake* artifacts — text files standing in for
  executables, skills and roles. That establishes R1's restore/quarantine
  logic and its refusal behaviour. It establishes **nothing** about a real
  installed pair: no real binary was replaced, no real `rsync` of a multi-
  megabyte skill tree ran, no real filesystem or permission edge was met. R1
  remains **UNTESTED operationally**.
- **R2 was not executed at all.** Its `agents-infra setup global` /
  `verify global` / `doctor global` calls were not run in any form by this
  revision, because doing so would mutate an installed runtime. R2 is
  **UNTESTED** in the stronger sense that even its logic has no execution
  evidence here. Treat the two rollbacks differently in risk terms.
- **The Part III snapshot block was not executed** against the real installed
  pair. Its derivation commands were reproduced individually (section A) and
  its version parsing was probed (section F), but the block as a whole —
  including `git worktree add`, the `cmp` loop and the `stat -f %d` same-device
  assertions — has not been run end to end.
- The revision-6 sandbox suite attacked shell functions this document no longer
  contains; that evidence is not carried forward.
- **The `env -u` invocations in Steps 2/5 and the `env -u`/skip-flag invocation
  in Steps 3/6 were not executed**, so it is not established here that the
  installers complete successfully under exactly those environments. That is a
  rehearsal item, listed under "Preconditions that are NOT IMPLEMENTED".
- **The consumption census was not re-derived.** The 555 symbols, the 20-package
  graph and the `./internal/spawn` gate are unchanged since document revision 3
  and were reproduced by the revision-2 review; this revision makes no
  independent claim about them.
- **No CLI/daemon protocol mismatch was staged end to end.** Section I rests on
  production source, its production call sites, the constant's history, and the
  measured live census. It does **not** rest on a daemon actually refusing a
  CLI, and this revision did not build two task-board images with different
  constants to watch the takeover fire. What a takeover does to a real
  daemon's 31 sessions is therefore **unknown**, not estimated.
- **What a live daemon's provider children do when it is terminated out of band**
  was not measured. The code says `Drain` deliberately leaves them running; it
  says nothing about SIGTERM. Stated as unknown rather than inferred from the
  drain comment. Revision 9 raises its importance: `terminateStaleInstance` is
  not a move somebody chooses after a downgrade, it is one of two branches the
  restored CLI takes **automatically**.
- **Whether a protocol-5 `tb-sessiond` would populate `startup_healthy` is
  unknown**, and it is the single field that decides which of those two branches
  fires. The decoder is lenient, so a renamed or dropped field decodes as `false`
  and selects the terminate branch — that is a mechanism, not a prediction, and
  no prediction is made. See "The limit of what more writing can fix", point 4.
- **Section L rests on call sites and their arguments, not on an observed
  takeover.** No protocol mismatch was staged, no second image with a different
  constant was built, and no `ConnectOrStart` was run against a newer daemon.
  L1-L27 establish that no production `ConnectOrStart` path *calls* the check;
  they do not show the resulting takeover happening. That distinction is the same
  one the F1 finding was about, so it is stated here rather than left implicit.
- **Whether `0.25.0` or `0.25.1` will change `controlProtocolVersion` is not
  established.** Those tags do not exist. That is exactly why P7c derives it at
  execution time instead of this document guessing in either direction.
- **P7 itself was not run as a whole.** P7a was driven verbatim (section K).
  P7b was driven against the real host as three individual commands (I22-I29),
  not as the document's loop. P7c is a two-line `grep` whose operands are two
  source trees, one of which does not exist yet; it was not run.
- **`restore_entry`'s own defences are unreachable and therefore unprobed.**
  R1 step 0 refuses an unrecognised kind and an empty symlink target before the
  restore loop runs, so the `*)` arm and the symlink arm's `test -n "$3"` cannot
  fire. They are kept as a second line of defence, not as tested behaviour. The
  behaviour that *is* tested is step 0's, in section J.
- **The revision-8 snapshot block was still not executed** against the real
  installed pair. Its new restorability pass is proved only by R1's identical
  copy (section J) and by a text check (J23-J24).

### Review lessons carried forward

1. A guard suite that stubs the component under attack proves only that the
   orchestration around it works. Revision 4's five green probes all replaced
   the reader, which is why a reader that returned zero on three failed `brew`
   calls survived them.
2. A fix applied to a finding is not a fix applied to a class. Revision 4
   removed one maintained literal; revision 5 reintroduced the same defect one
   line away.
3. A fix that adds machinery moves the defect into the machinery. Revisions 3-6
   each closed a finding and each introduced a new fail-open in the code that
   closed it. Severity did not fall. When the mechanism protecting a plan needs
   its own adversarial review every cycle, the mechanism is the risk, and the
   correct move is to replace it with something an operator can audit in one
   reading.
4. **New in revision 8:** a plan that models artifacts will miss every risk that
   lives in a process. Seven revisions treated `tb-sessiond` as a name in an
   executable list, and the surface it actually is — a daemon with its own
   versioned protocol, older than the file it came from, holding 33 sessions
   across two boards this migration does not touch — was invisible the whole
   time. The general form: **enumerate what is running, not only what is
   installed.** A rollback restores files; it does not restore processes, and a
   plan that does not say so is claiming a reversibility it does not have.
5. **Also new:** when a review finds a defect in one arm of a `case`, read the
   other arms. F2 reported an abort in the `directory` arm; the `symlink` arm on
   the same input fails *open* (J13-J17), which is worse and was not reported.
   Fixing the reported instance would have left that one shipping.
6. **New in revision 9:** a probe that a guard *exists* is not evidence the guard
   *runs*, and a plan that treats the first as the second is asserting a
   protection it does not have. Revision 8's I5-I7 read the newer-daemon branch
   correctly and never asked what calls it; the answer was "no production
   `ConnectOrStart` path", and an entire mitigation strategy rested on the gap.
   This is the fourth instance of the shape *the check is present but uncalled
   from production* in this project. The rule that follows: **for every guard a
   plan relies on, name the production entry point that reaches it, and probe the
   path — not the text.**
7. **Also new:** when writing about a binary you do not own, the plan may only
   *describe*. A missing guard is a prerequisite for the owning repository, named
   as such, never a sentence added to the plan that reads as though the behaviour
   exists.
8. **Also new:** say where more writing stops helping. Two properties this plan
   depends on can only be settled by a rehearsal it is forbidden to run, and
   naming that limit is a result — not a gap to paper over with another section.

# Stop conditions

Stop and restore the last accepted pair if:

- any precondition P1-P6 does not produce its stated expected result;
- any manifest item proposed for `v0.6.0` is still required or unknown for
  either exact released consumer;
- any read/inventory is failed, partial or malformed — a failed read is never an
  absence;
- a candidate requires `replace`, workspace override, mutable lookup or shadow
  graph;
- a real spawn succeeds but resource/handoff/reviewer routing fails;
- the saved absolute pair cannot route during a cutover failure;
- an in-flight run would need to be killed, reparented or rewritten;
- one restored half is green but the pair canary is red;
- a P1 diff touches a modelled boundary — the plan is re-derived and re-reviewed
  rather than patched at the console;
- a rollback remains unrehearsed in the target environment. It stays labelled
  **UNTESTED** and cannot be assumed to work;
- `controlProtocolVersion` differs between the saved installed source and the
  release, at Step 2 or Step 5. This is not a stop that P7 can clear by being
  re-run: it needs either a release that does not change the constant, or a
  written per-board decision from the owners of every board holding a live
  daemon, obtained before the window;
- a live daemon exists on any board with no recorded P7 disposition;
- the daemon census fails to read — a `pgrep` exit other than 0 or 1, a
  `session status` that exits non-zero or returns unparseable output. A failed
  census is `unknown` and is never "no daemons";
- after a rollback, any board reports a `protocol_version` greater than the
  restored build's `controlProtocolVersion`. That board is not rolled back, and
  no procedure in this plan makes it so.

# The limit of what more writing can fix

The revision-7 review asked for this to be said if it were true, and it is.

Three risks in this plan are now bounded as tightly as a document can bound
them, and the remaining reduction is not another paragraph:

1. **R1 and R2 have never been run against a real installed pair.** R1's logic
   has been driven hard — forty-nine probes across sections G and J, five of
   them revision-8 refusals with narrowing controls — but every artifact it
   moved was a
   text file standing in for a binary, a skill tree or a role. No multi-megabyte
   `rsync`, no real permission edge, no real filesystem. R2 has no execution
   evidence at all. The next useful thing is a rehearsal on a disposable
   home and runtime, recording real exits. Writing more about R1 will not
   produce it.
2. **The protocol takeover has never been observed.** Section I establishes what
   the code does from the code. Nobody in this task has watched a `0.25.0`-class
   CLI fence a v4 daemon holding real sessions, and the honest answer about what
   that does to those 31 sessions is unknown. The next useful thing is two
   throwaway builds with different constants on a disposable board — not a
   sharper description of `takeOverManagerForProtocolUpgrade`.
3. **The installers have never been run under exactly the `env -u` invocations
   this plan specifies.** They are quoted, derived and reviewed; they are not
   executed.

Revision 9 adds two more, both surfaced by correcting F1, and both of the same
kind — questions about a binary that does not exist yet:

4. **Whether a protocol-5 `tb-sessiond` populates `startup_healthy`.** It is the
   single field that decides whether a restored older CLI receives an unchecked
   live client or SIGKILLs a daemon holding live sessions. It cannot be read from
   `f1319eff`, because the answer is a property of an unreleased build. The next
   useful thing is to build `0.25.0`, start its daemon on a disposable board, and
   ask it — not a sharper reading of `takeOverUnresponsiveManager`.
5. **What a terminated daemon's provider children do.** Marked UNKNOWN since
   revision 8 and now more load-bearing, because the termination is automatic
   rather than chosen. It is a process-tree question and is not answerable from
   source with any confidence worth acting on.

Points 2, 4 and 5 are one rehearsal: build `0.25.0`, install it into a disposable
`HOME`, start a daemon on a disposable board, downgrade, and watch. If that
rehearsal is not authorised, then P7's prevention is not one of two options — it
is **mandatory**, because the fallback revision 8 believed in (a refusal that
freezes the state until somebody decides) does not exist, and section L is why.

Everything else in this plan is either derived at execution time or labelled
`UNTESTED`/`unknown`. These five are the cases where the honest answer is
**"this step cannot be made safe without trying it on a disposable copy first"**,
and a reviewer should treat further prose about them as noise. The plan's own
acceptance criterion — reviewed and accepted before the first breaking change —
is met by review; its *operational* readiness is not, and no revision of this
document can supply it. That is the limit, stated as a result rather than as an
apology.

# References

Production sources read at `skill-project-management` `f1319eff` and
`relux-agents-infra` `bb857fe5`, 2026-08-30.

- task-board `internal/spawn/launch_plan.go:157-164,332-342`
- task-board `internal/spawn/models.go:307-332`
- task-board `internal/spawn/model_registry_build.go:9-20,209-306`
- agents-management `pkg/agentic/registry.go:120-128,170-183`
- agents-management `pkg/vendorplugin/registry.go:252-337,363-388`
- agents-management `pkg/vendorplugin/engine.go:9-11,31-54`
- project-management installer `scripts/setup.sh:16` (`BIN_DIR`), `:35` (the one
  `source`), `:131-146` (`install_go`), `:816-818` (skill destination
  assignments), `:822-831` (`SKILL_REPOS`), `:834-838` (`SKILL_NESTED`),
  `:840-888` (`install_skills`), `:195-210` (`install_binary`),
  `:891-930` (`setup_main`)
- agents-infra installer `scripts/setup.sh:5-18` (top-level assignments),
  `:64-73` (`resolve_config_dir`), `:75-91` (`install_go`), `:93-171`
  (`install_lldb_mcp`), `:255-271` (call sequence)
- agents-infra global setup `tools/agents-infra/internal/infra/infra.go:134-215`
- agents-infra source resolution `tools/agents-infra/internal/infra/source_dir.go:12-19`
- task-board session manager `internal/sessionmanager/types.go:11-13` (protocol
  name and version), `:114-128` (`ManagerStatus`)
- task-board session manager `internal/sessionmanager/client.go:70-96` (`Dial`),
  `:112-126` (`Connect`), `:128-165` (`ConnectOrStart` and the upgrade branch),
  `:228-253` (`ConnectOrStartAndAttach`), `:340-346` (`Drain`), `:677`
  (lenient response decoding), `:714-735` (`managerProtocolCompatibility`)
- task-board session manager `internal/sessionmanager/stale_takeover.go:25-41`
  (`dialManagerStatus`, unchecked), `:43-59` (`dialHealthyManager`, the only
  caller of the compatibility check on this path), `:71-79`
  (`takeOverOutdatedManager` and its downgrade refusal), `:86-163`
  (`takeOverManagerForProtocolUpgrade`, drain then fence, and the one production
  place the newer-daemon refusal is surfaced), `:168-266`
  (`takeOverUnresponsiveManager`, no compatibility check), `:265`
  (`terminateStaleInstance`, SIGTERM then SIGKILL)
- task-board session manager `internal/sessionmanager/builder_gateway_manager.go:319-352`
  (`FinalizeAndRequireBuilderCoverageEvidence`, the only production invocation of
  `Connect`)
- task-board `cmd/context_security.go:75` (`sessionmanager.Connect` passed as a
  function value), `cmd/context_publish.go:138` and `cmd/mutate.go:127` (its two
  callers)
- task-board `cmd/codex_manager.go:92` and `cmd/claude_manager.go:115`
  (`ConnectOrStartAndAttach` with a non-nil launch closure)
- task-board `cmd/session.go:26` (`tb-sessiond`), `:34-110` (`Dial`-based
  reads), `:151,199,245` (`ConnectOrStart`-based commands), `:360-372`
  (`connectOrStartSessionManager`), `:480-506` (`launchSessionManagerDaemon`,
  `exec.LookPath`, `--board-dir`/`--remote-origin` argv)
- task-board `cmd/codex_goal_spawn.go:35-44` (`connectManagedSpawnSession`),
  `:50-56` (`launchTrackedSpawnRun`), `:74-79` (`requiresManagedSpawnSession`)
- `TASK-260830-s5ro4e_review-verdict-rev7.md`
- `TASK-260830-s5ro4e_review-verdict-rev6.md`
- `TASK-260830-s5ro4e_review-verdict.md`
- `TASK-260830-s5ro4e_review-verdict-rev3.md`
- `TASK-260830-s5ro4e_review-verdict-rev4.md`
- `TASK-260830-s5ro4e_review-verdict-rev5.md`
