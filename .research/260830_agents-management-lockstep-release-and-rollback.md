# Agents-management lockstep release and rollback plan

Task: `TASK-260830-s5ro4e`

Date: 2026-08-30

Numbering: this document keeps its own revision counter, which runs one ahead
of the board's Change Request counter because the first document revision
predates the first CR. **Document revision 7 is `CR-TASK-260830-s5ro4e-6`**, and
the attached patch is named `..._change-request_rev6.patch` to match the board.
The `rev4 … / rev5 …` rows retained in the validation table keep the document
numbering they were run and reviewed under.

Review state: revised after `CR-TASK-260830-s5ro4e-5` changes requested. This
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

## Preconditions that are NOT IMPLEMENTED

Stated so that the gap is visible rather than papered over by an untested guard.

| Requirement | Why it wants automation | Status |
| --- | --- | --- |
| Assert that the interval between the P6 before-snapshot and the install is not raced by a concurrent `brew` invocation | A human cannot observe concurrency; a lockfile check would | **NOT IMPLEMENTED.** Mitigation: run releases when no other Homebrew work is in flight, and treat a P6 difference as authoritative. |
| Mechanically classify a diff produced by P1 into "touches a modelled boundary" versus "does not" | The judgement is currently the operator's, and a large refactor diff is tiring to read | **NOT IMPLEMENTED.** Mitigation: P1's four questions are answered in writing and attached to the step evidence, so the judgement is reviewable after the fact. |
| Verify that the `env -u` list in Steps 2/5 still covers every `TASK_BOARD_*` name P2 reports | Two lists that must agree; today the operator compares them by eye | **NOT IMPLEMENTED.** Mitigation: both lists appear in this document adjacent to each other, and P2's diff catches an added name even though it does not catch a forgotten `-u`. |
| Rehearse any rollback procedure against a real installed pair | See "UNTESTED" throughout Part III | **NOT IMPLEMENTED / UNTESTED.** |

None of these is a substitute for a guard that was removed: P1, P2, P3, P4, P5
and P6 each cover strictly more than the code they replace. These four are
properties no revision ever had.

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
         "$RECOVERY_ROOT/pm/skills" "$RECOVERY_ROOT/ai" "$RECOVERY_ROOT/sources"

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
  for root in "$HOME/.claude/skills" "$HOME/.codex/skills"; do
    if   [ -L "$root/$skill" ]; then line="$line|symlink|$(readlink "$root/$skill")"
    elif [ -d "$root/$skill" ]; then line="$line|directory|"
    elif [ -e "$root/$skill" ]; then exit 77
    else                             line="$line|absent|"
    fi
  done
  printf '%s\n' "$line" >> "$MANIFEST/skill-state.txt"
done < "$MANIFEST/skills.txt"

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

# 4. Skills: restore each snapshot entry to its exact recorded kind.
#    An unrecognised kind returns 1, which aborts the block under errexit.
restore_entry() {   # $1 path  $2 kind  $3 link-target  $4 staged tree
  d="$(dirname "$1")"; b="$(basename "$1")"
  case "$2" in
    symlink)
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
  restore_entry "$HOME/.claude/skills/$skill" "$c_kind" "$c_link" ""
  restore_entry "$HOME/.codex/skills/$skill"  "$x_kind" "$x_link" ""
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
# then a live spawn -> outcome -> handoff -> reviewer route canary, recovery-prefixed
```

Rollback status: **UNTESTED operationally.** R1's logic was exercised against a
disposable `HOME` with fake artifacts (Part VI section G: restore, quarantine of
release-added artifacts, preservation of unrelated ones, and refusal on a
malformed snapshot or a missing saved executable). That is not a rehearsal
against a real installed pair and does not make R1 assumed-reversible.

A partial run of R1 is a rollback failure, not a successful half-restore.

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
| R1 (task-board surface) | yes — Part VI section G, disposable `HOME`, fake artifacts, including four refusal probes with narrowing controls | no | **UNTESTED operationally** |
| R2 (agents-infra runtime) | **no** | no | **UNTESTED** |
| Snapshot block | derivations and version parsing only | no | **UNTESTED** |

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

Provider children and already-exec'd CLI/sessiond processes continue. A run that
reaches a new CLI mutation during the mixed window may see a partial default
install; retry its mutation through the recovery-prefixed old pair. Do not
reparent, rewrite or kill the run.

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

P1 through P5 run exactly as in Step 2, and the install command is byte-for-byte
the Step-2 one with `$PM_RELEASE` repointed. The stage list goes to
`"$RECOVERY_ROOT/w5-setup-main.txt"`.

Because `$SAVED_SOURCE` here is the `bridge-pair` source — that is, `0.25.0` —
P1 compares `0.25.1`'s installer against `0.25.0`'s. A boundary change
introduced by `0.25.1` stops Step 5 the same way `0.25.0`'s would have stopped
Step 2.

In-flight runs: same process-image behaviour as Step 2. Their next CLI mutation
may require replay through `bridge-pair`; no run is killed or rewritten. An
interruption inside `install_skills` has the same recovery authority as in
Step 2, against the `bridge-pair` snapshot.

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
| `W5`: task-board `0.25.1` install | First `install_binary` call to installed or R1 canary; measured | Same task-board surface as `W2`, including the unconditional `install_skills` stage; new default spawn/route withheld. | Saved bridge-pair authority, the same P3/P4 preconditions, and the full R1 rollback. |
| `0.25.1/v0.6.0` with agents-infra `v1.7.0/v0.5.0` | Until Step 6 canary | Cross-version embedded libraries behind one external seam. | Allowed only after the Step-5 real spawn/compose/route canary. |
| Step-6 Homebrew/LLDB bypass guard | P6 before-snapshot through a successful after-comparison; measured | As Step 3. | As Step 3. |
| `W6`: agents-infra `v1.7.1` install | First managed binary replacement to installed or R2 canary; measured | Same mixed runtime surface as `W3`; new default composition withheld. | Saved bridge-pair; restore one half, canary, then restore both if the intermediate pair is red. |

The forbidden window is running an installer from a breaking consumer candidate
before its exact tests, removal manifest, out-of-place canary and recovery
snapshot are green. Its impact would be loss of new spawn/route authority for an
unbounded time. This plan never accepts that mode.

Two windows that earlier revisions did not name, and where they are now:

- **`install_go`, both installers, before every managed binary.** Not a window
  under P3, because its only reachable branch is a read. Without P3 it is an
  unbounded Homebrew mutation inside the release window with no snapshot and no
  restore — which is what revision 6 shipped.
- **The `install_skills` clone branch, task-board.** Not a window under P4,
  because every managed skill is already a real directory. Without P4 it is a
  network fetch of an unrelated skill at an arbitrary remote HEAD, whose failure
  mode deletes the skill.

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

Validator: `.temp/TASK-260830-s5ro4e/validate-revision7.sh`, run under both
`bash` and `zsh`. **74 probes, 0 failures, exit 0 in both shells.** Logs:
`.temp/TASK-260830-s5ro4e/val-bash.log`, `val-zsh.log`.

The suite locates the R1 and P6 blocks inside this document **by content**, not
by line number, so a shifted offset cannot silently point it at the wrong block
or at nothing.

Every one of the 39 shell fences in this document was extracted mechanically
(`.temp/TASK-260830-s5ro4e/extract-fences.py`) and syntax-checked: `bash -n` and
`zsh -n` on every `bash` fence, `zsh -n` on every verbatim-production `zsh`
fence. All 39 parse in every applicable shell. The `zsh` fences parsing is what
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

### B, C, D — each replacement attacked against the guard it replaced

Every row runs the **revision-6 guard verbatim** and the **revision-7
precondition** against the same mutant. A row is only meaningful because the
revision-6 column is a fail-open on production text.

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
| D1/D2 | a formula appears in `brew list --formula` — what `install_go`'s `brew install go` does | five-value LLDB target set **compares identical, blind** | whole formula list **differs, refused** |
| D3 | *no mutation* (narrowing control) | — | **identical** |

B5, C5 and D3 are the narrowing controls. Without them B/C/D would prove only
that the new checks can fail, not that they discriminate.

### E — P3 and P4 driven against a disposable HOME

| # | Condition | Result |
| --- | --- | --- |
| E1 | `go` absent from `PATH` | P3 **refuses** (exit 1) |
| E2 | the real target host | P3 **satisfied** (exit 0) |
| E3 | all nine managed skills are real directories | P4 **OK** |
| E4 | one skill is a symlink — the installer would `rm` it and clone | P4 **REFUSE** |
| E5 | one skill is absent — the installer would clone from the network | P4 **REFUSE** |
| E6 | the same skill re-materialised as a directory | P4 **OK** — narrowing control: P4 refuses the clone branch without refusing a legitimate installed skill |

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

The R1 block was extracted from this markdown file by line offset and run with
`bash`, not retyped, against a disposable `HOME` holding fake artifacts. **No
installer was executed.** The simulated release adds an executable, a role and a
skill, and rewrites an existing dependency skill and an existing role.

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
| G18-G19 | a **missing snapshot executable** aborts R1 non-zero, and R1 does not report success while leaving the release binary in place | refused; `BIN_DIR` still holds the release binary, correctly, because R1 stopped |
| G20-G21 | R1 green again once the snapshot is complete, and the executable is restored — narrowing control | pass |

G17 and G20 are what make G16 and G18 evidence rather than noise: without them a
non-zero exit could be R1 being generally broken.

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

### Review lessons carried forward

1. A guard suite that stubs the component under attack proves only that the
   orchestration around it works. Revision 4's five green probes all replaced
   the reader, which is why a reader that returned zero on three failed `brew`
   calls survived them.
2. A fix applied to a finding is not a fix applied to a class. Revision 4
   removed one maintained literal; revision 5 reintroduced the same defect one
   line away.
3. **New, and the reason for this revision's form:** a fix that adds machinery
   moves the defect into the machinery. Revisions 3-6 each closed a finding and
   each introduced a new fail-open in the code that closed it. Severity did not
   fall. When the mechanism protecting a plan needs its own adversarial review
   every cycle, the mechanism is the risk, and the correct move is to replace it
   with something an operator can audit in one reading.

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
  **UNTESTED** and cannot be assumed to work.

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
- `TASK-260830-s5ro4e_review-verdict.md`
- `TASK-260830-s5ro4e_review-verdict-rev3.md`
- `TASK-260830-s5ro4e_review-verdict-rev4.md`
- `TASK-260830-s5ro4e_review-verdict-rev5.md`
