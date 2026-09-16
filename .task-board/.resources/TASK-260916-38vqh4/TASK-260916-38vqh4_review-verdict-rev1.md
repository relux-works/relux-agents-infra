# TASK-260916-38vqh4 — review verdict, revision 1

Verdict: CHANGES_REQUESTED. Route to analysis for research-report corrections, then another reviewer cycle. No external blocker or human decision is needed.

Reviewed CR-TASK-260916-38vqh4-1, base 459742ea67e3c6b84169520b92d74fe7f73e3002, candidate 36293029bf009df16e5cf1bf38a1757019c0b812, and TASK-260916-38vqh4_report.md. The repository delta is empty, as verified by git diff --stat. That is the correct delivery shape for this read-only research leaf: the required product is an attached report, not source changes. The rejection concerns report accuracy/completeness, not the absence of code changes.

## Required corrections

1. **P2 — four executable runtime subcommands are missing from the classification table.** The runtime row inventories only status/stop. tools/agents-infra/main.go:854 dispatches quarantine/unquarantine to SetSharedRuntimeManualQuarantine; :870–885 dispatches broker/runtime-launch to RunSharedRuntimeBroker/RunSharedRuntimeLauncher. These are real residual Pi operations, even though usageText does not list all of them. Add explicit KEEP classification, source references, rationale and retained-test coverage for all four. The current broad promise to retain Pi implementation files does not satisfy the acceptance criterion to classify every command. Derive the command inventory from all dispatcher branches as well as README/usage so hidden commands are counted.

2. **P2 — the local Codex shim is named at a nonexistent path.** The setup table and exact deprecation section name `.codex/bin/codex`. LocalLayout actually sets BinDir to `<project>/.local/bin` (internal/infra/infra.go:116–132); setupCodexLocalLauncher writes `codex-local` there (:1414–1437), conditionally on a nonempty enabled MCP list. Its body delegates to agents-infra codex (:1441–1452). README.md:2588 also names `.local/bin/codex-local`. Correct all occurrences to that actual surface, retain the planned codex stderr/exit-1 behavior, and specify how setup preserves/creates/removes the shim when MCP opt-ins are present/absent after bundled-registry distribution stops. Point the executable deprecation tests at the installed shim and retain/adapt TestSetupLocalProjectMCPOptInInstallsCodexLocalLauncher (infra_test.go:1069), plus the no-opt-in/removal tests around :1122–1133. A test aimed at `.codex/bin/codex` would miss the actual installed entrypoint.

No repository edits are requested in this leaf; update the attached report and republish through the producer lifecycle.

## Verified aspects

- All five local Go packages are represented; go list ./... returned exactly the report's five packages. scripts/setup.sh and setup.ps1 step inventory matches their source. All 15 .instructions files and four .configs files are covered by individual or explicitly grouped rows. The command coverage has the four omissions above.
- The v1 compatibility exception is necessary. In the task-board consumer at /Users/administrator/Developer/ReluxWorks/skill-project-management (HEAD b2e26c038ddfe38392aa86412eb066fd12d45462), tools/board-cli/internal/spawn/launch_composition.go:735 invokes child compose; cmd/infra_prepare.go:58–93 invokes and decodes prepare; internal/sessionmanager/launch_plan.go:244–313 requires honest runtime state plus Codex rendered instructions or Claude entrypoint/instruction/settings artifacts. cmd/codex_manager.go:184 and cmd/claude_manager.go:219 invoke primary-session compose. In this repository, primary_session_prepare.go:114,142 calls rendering setup, and child_launch_composition.go:140–155 loads the MCP registry and rejects missing configured definitions. Removing these implementations outright would violate the retained contracts.
- Direct launcher deprecation is otherwise coherent: stderr, exit 1, empty stdout, no provider execution, no print-config exception; guards before target spawn/argument parsing cover canonical aliases and target-yolo. Pure compose/prepare paths remain live. Proposed negative tests exercise executable entrypoints and include narrowing a guard rather than merely deleting it.
- The residual list retains settings linking, config merge, rules, Pi runtime/harness, attachments and task-board APIs. The bounded LLDB absence finding is correct: production search in tools/agents-infra, scripts, .scripts and .configs found no wrapper; matches are test fixtures, while README.md:2547 documents no bootstrap. This is not an independent claim about external LLDB availability.
- B5 evidence was read directly at the cited Curator board resource, including :318–347, :655–732, :751–790 and :1052–1104. It supports profile activation, managed-home MCP/skills and first-run provisioning. The report correctly limits instruction-injection parity and does not claim successful authenticated Claude/Pi prompts.
- Slice A's separation of ordinary installation from v1 compatibility is implementable in principle while preserving compose/prepare expected outputs. This is a design assessment, not proof of future code. No golden or source changes were made. Full implementation validation remains the implementation task's responsibility.

## Validation actually run

- PASS: go list ./... — exit 0, five packages.
- PASS: go test ./internal/infra -run 'TestBuildChildLaunchComposition|TestPreparePrimarySession|TestSetupLocalProjectMCPOptInInstallsCodexLocalLauncher' -count=1 -timeout 60s — exit 0, 3.917s. This reran existing contract and real shim-installation tests, not future migration behavior.
- INCOMPLETE: go test ./... -timeout 4m — explicitly terminated by reviewer before completion, exit 143, no package result output. No full-suite green claim is made. All started test processes were terminated before ending this lifecycle.
- git diff --stat base candidate and git status --porcelain — empty. No code edits, installer invocation, provider launch, Windows validation, or mutation testing performed.
- task-board spawn goal reported this run is not goal-bound; directives reported none.

## Review logbook

2026-09-16: Confirmed mandatory v1 renderer/registry compatibility and bounded absence of an LLDB wrapper. Found four omitted runtime subcommands and incorrect installed Codex shim path. These are recoverable research rework, not stop-the-line conditions. Persisted here and in board notes; repository remains read-only.
