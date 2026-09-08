# TASK-260830-s5ro4e review verdict — revision 2

Verdict: **changes requested**. Route to `analysis` for a corrected plan and a
new Change Request revision.

Reviewed candidate: `CR-TASK-260830-s5ro4e-2` revision 2, base
`5c9b4e4f7a88e1eb937b80851af522e4fa4b066f`, candidate tree
`59847b20212aff9ec245b71d2df289401e50878d`, patch SHA-256
`8727fa3addfcb1191bf114ae7424f5f22059f438d3d5e0d7f12c66fd51fb1393`.

## Findings

### F1 — High: W3/W6 begin after a real installer mutation and cannot roll it back

The plan says W3 starts when `agents-infra` is replaced and the window table
starts at the first binary replacement/receipt invalidation
(`.research/260830_agents-management-lockstep-release-and-rollback.md:435-440,583`).
W6 inherits that artifact sequence (`:553-555`). The production installer does
something earlier: `scripts/setup.sh:258-263` calls `install_lldb_mcp` before
building or replacing either agents-infra binary. On this target host the
preconditions select the mutating branch (`brew_present=true`,
`lldb_mcp_present=true`, and the resolved helper is the Homebrew wrapper path).

That branch runs `brew install llvm`, removes and rewrites
`$BREW_PREFIX/bin/lldb-mcp`, and only then reaches the agents-infra binary
replacement (`scripts/setup.sh:93-170,258-263`). An interruption after the
wrapper mutation but before `install_binary` therefore creates a fifth partial
installer window outside W3/W6. A child composition that enables the LLDB MCP
surface can resolve the changed absolute wrapper even when task-board and
agents-infra are recovery-prefixed. This is the negative shape **bypass path
around the check**: the saved old routing pair does not bound every installer
entry path that can affect a new composition.

The rollback is incomplete for the same reason. The snapshot saves seven
`~/.local/bin` artifacts but not the Homebrew wrapper/helper state (`:217-222`),
and `restore_agents_infra_surface` restores only `agents-infra`,
`model-harness`, and the `.agents` runtime (`:297-308`). It cannot return this
pre-binary mutation to the previous working state.

Required rework: either make Steps 3 and 6 explicitly run the migration with
`AGENTS_INFRA_SKIP_LLDB_MCP=1` and verify that no LLDB/Homebrew path changes, or
extend the named window, snapshot, rollback, in-flight behavior, duration, and
canary to the first possible LLDB/Homebrew mutation. The bounded recommendation
is to skip this unrelated optional install during the lockstep migration and
manage it separately. Keep any newly added operational rollback labelled
`UNTESTED` until rehearsed.

### F2 — Medium: LOGBOOK.md appends a contradiction instead of replacing the false entry

The spawn brief required the revision-1 institutional-memory claim to be
replaced, not contradicted beside a correction. `LOGBOOK.md:8-12` adds a new
“superseded” entry, but `LOGBOOK.md:14-18` leaves the false 26/38 census, the
archive-only skipped-gate rationale, and the “removes the lockstep outage
window” headline intact.

This remains misleading evidence for grep/search consumers and directly misses
the requested correction shape. Required rework: replace or rewrite the 1245
entry so the stored finding itself reports the corrected 25/36 direct census,
20-package composed graph, real unmodified gate, and named mixed-install
windows. Do not retain the disproven conclusion as a second current finding.

## Independent checks that passed

- The exact CR patch resource hashes to the handed SHA-256. The base/candidate
  diff contains only the plan and `LOGBOOK.md`; `git diff --check` exits 0.
- The board plan resource is byte-identical to the candidate plan (SHA-256
  `d7aa02bb99074211d02bd142fbbddff5c7a22d77b6e0dac69ff02f4004d3262a`).
- The four measurements are distinct, not one count relabelled:
  - tracked non-vendored direct census: 25 production files, 36 test files,
    13 package families, zero raw plugin/inference-engine imports;
  - composed `GOWORK=off go list -mod=mod -deps ./...`: 20 module packages;
  - independently built Darwin/arm64 candidate: 555 `go tool nm` lines
    containing the module path, with the reported 161/32/153/7/190 family
    counts and all three named critical symbols;
  - real candidate preflight exits 0; `GODEBUG=inittrace=1` shows agentic,
    inferenceengine, vendorplugin, providerlimits, and all vendor package init
    functions executing. Production callers reach `launchRegistry` and
    `buildLaunchPlan`; the module's `Register` calls `RegisterAll`.
- `GOWORK=off go test -mod=mod ./internal/spawn -count=1` independently exits
  0 in 11.088s from a detached exact-commit worktree pinned only to public
  `v0.5.0`.
- The `v0.6.0` gate names the load-bearing compatibility methods/types, requires
  a per-removal manifest, and refuses release unless every item is proven
  `required_by_0.25.0=false` and `required_by_v1.7.0=false`; failed, partial,
  init-only, linked, and production-path evidence fail closed.
- Step 5 now has a concrete rollback sequence. Operational rollbacks remain
  labelled `UNTESTED`, and in-flight behavior is stated for Steps 0-6.
- No migration, install, release, or tag exists in the reviewed delta.

## Review diagnostics

- An initial direct census accidentally included the tracked vendor tree and
  returned 73/36/17. It was not treated as the source result; excluding
  `tools/board-cli/vendor/**` reproduced the plan's 25/36/13.
- An initial `awk` symbol command failed syntactically. It was recorded as a
  failed read, not converted to zero; fixed-string independent counts then
  reproduced 555 and the named families.
- The review created only task-scoped scratch build evidence under `.temp/` and
  did not modify the Change Request candidate, install anything, publish a tag,
  or kill/restart any process.
