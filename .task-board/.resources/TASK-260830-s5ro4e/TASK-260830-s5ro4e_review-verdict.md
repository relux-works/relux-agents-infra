# TASK-260830-s5ro4e review verdict

Verdict: **changes requested**. Route to `analysis` for a corrected plan and a new Change Request revision.

Reviewed candidate: `CR-TASK-260830-s5ro4e-1` revision 1, base `5c9b4e4f7a88e1eb937b80851af522e4fa4b066f`, candidate tree `57df379d8d08cb3cec6031cd09e22e232b2dc06e`.

## Findings

### F1 — Critical: the measured consumption surface is the wrong graph

The plan reports 17 compiled agents-management packages and says task-board does not call the raw generalized registry or inference-engine contract. That count describes the restored `v0.2.0` scratch source, not the composed `v0.5.0` candidate the release plan relies on.

Evidence:

- `go mod download -json github.com/relux-works/skill-agents-management@v0.5.0` resolves public `v0.5.0` to exact Git commit `b74f758a90a422304d55460422831da80e3d6cc8` and checksum `h1:yy/j8YgKrXLWcb6zzoAld79a7iamTvOI5eZd/9MaAIk=`.
- The producer-built task-board binary reports that same `v0.5.0` dependency and checksum through `go version -m`.
- `go tool nm` on that composed binary contains `pkg/plugin.(*Registry).RegisterAll`, graph resolution symbols, `pkg/inferenceengine.init`, `vendorplugin.productionEngineFactSource.ReadEngineFacts`, and inference-engine refusal symbols. These are linked production artifacts, not test-only source matches.
- Starting from the 13 package families task-board imports, `GOWORK=off go list -deps` at exact agents-management `v0.5.0` yields 20 module packages, including `pkg/plugin`, `pkg/inferenceengine`, and `pkg/inferenceengine/engines/mlx`.
- The production path is concrete: task-board `internal/spawn/launch_plan.go:157-160` creates and registers systems; `pkg/agentic/registry.go:120-128,170-183` turns that compatibility call into `plugin.Registry.Register`. Model construction reads `vendorplugin.Default`; vendor init registration reaches `pkg/vendorplugin/registry.go:252-337`, which syncs/registers the graph. `pkg/vendorplugin/engine.go:9-11,31-54` imports and calls the inference-engine contract. None of these files has a build tag excluding the path.

Direct `pkg/plugin` and `pkg/inferenceengine` imports in task-board remain zero, but that is not the requested actual consumption surface. The negative shape is **bypass path around the check**: a direct-import census misses wrapper and initialization paths. The reported capability/absence claim does not reproduce against the composed production binary.

Required rework:

1. Re-measure the `v0.5.0` candidate, separating direct imports, transitive compiled packages, linked symbols, and actually invoked production paths.
2. Replace the no-reach claim in the plan and in `LOGBOOK.md`; the current logbook entry would persist false institutional memory.
3. Define the `v0.6.0` removal gate from the exact compatibility methods/types still invoked by `0.25.0` and `v1.7.0`, not from absence of direct imports. Prove both released consumers no longer require every removed adapter before allowing the break.
4. Base the no-runtime-mismatch conclusion on immutable pins, tested adapter compatibility, and cutover isolation. Do not base it on an absence that is false.

### F2 — High: Step 5 has no concrete rollback command sequence

Lines 344-347 say to restore `0.25.0` and `v1.7.0` from release-specific backups, but the plan never gives commands that capture those backups or restore and verify both installed surfaces. Forward patch releases are recovery policy, not an immediate rollback command.

This fails the acceptance criterion requiring a concrete rollback sequence for every release step.

Required rework: add exact, safe backup commands before the first `0.25.1` mutation; exact same-filesystem restore commands for task-board, `tb-sessiond`/installed skill/roles where applicable, and agents-infra; version/verify plus live spawn-and-route checks; and explicit failure handling if either half restores but the other does not. Keep the rollback labelled `UNTESTED` until exercised.

### F3 — High: partially applied installer windows are omitted

The unsupported-window table names only a sub-second task-board binary replacement. The actual project-management installer updates multiple artifacts sequentially (`task-board`, `tb-sessiond`, TUI, launchers, skill, roles, install state, then verification; `scripts/setup.sh:904-924`). An interruption after the first mutation can leave a mixed installed contract for longer than the binary `mv`. The agents-infra coherent setup likewise changes more than one executable when runtime/install surfaces are involved.

The plan assumes these windows away and therefore does not state their duration, impact on new spawn/route commands, recovery authority, or the behavior of in-flight monitors/sessions during a partial install.

Required rework: for Steps 2, 3, and 5, name the window from the first installed mutation until installed canary success or rollback completion; list every artifact that may be mixed; state which saved absolute binary remains the routing authority; state what old `tb-sessiond`, primary sessions, and provider children do; and give a fail-closed rollback sequence from each interruption point.

### F4 — Medium: the load-bearing compatibility gate is not green

The producer records the exact spawn-package command as exit 1 and obtains green only by skipping the Git-history guard because the scratch tree came from `git archive`. That is honest reporting, but it does not satisfy the task's `Tests green` gate, and the skipped guard is a bypass around the composed validation command.

Required rework: run the unmodified command with `-count=1` from a clean detached Git worktree at exact task-board commit `063197b1`, pinned to public `v0.5.0`, with no `replace`/workspace override. Record the complete exit result. If it remains red, keep it red and investigate; do not substitute a skip run as the full gate.

## Checks that passed

- Exact candidate files hash to the handed tree; review did not drift the Change Request.
- `git diff --check` over the exact base/candidate range exits 0.
- The plan names the intended repository/version order and gives useful board capability and in-flight-run requirements for Steps 0-4.
- Operational rollbacks that were not exercised are visibly labelled `UNTESTED`; the bounded source repin is labelled only `PARTIALLY TESTED`.
- The Change Request changes only the research plan and logbook; no migration, tag, or release is present in the reviewed delta.

## Review diagnostics

- The first compact board read used an unknown `task(...)` operation and failed closed; schema discovery identified `get(...)`, whose rerun succeeded. No absence was inferred from the failed read.
- One attempt to remove an incidental `/tmp` scratch file was rejected by the command guard before execution; it had no repository or board effect.

