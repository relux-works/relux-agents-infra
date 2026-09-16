# TASK-260916-3q6z2c — revision 6 review verdict

Verdict: ACCEPTED. Reviewed CR-TASK-260916-3q6z2c-6, candidate tree d91f1a1e3a7fd2420a586a127d774c332e975f18 against base 459742ea67e3c6b84169520b92d74fe7f73e3002. The revision-5 P1 is resolved; no blocking finding remains.

## Findings and scope

Installed canonical/dange aliases and codex-local now print the refusal directly. The generated agents-infra wrapper guards deprecated argv before directory creation, toolchain invocation, or provider resolution. Shared literal messages keep shell and Go dispatch consistent. Live qwen delegation remains covered. The correct local shim is .local/bin/codex-local, as the accepted inventory established.

The real executable test passed all 58/58 cases: 36 compiled CLI cases, 10 alias/shim cases, and 12 generated CLI-wrapper cases. It shadows go with an exit-73 executable, deletes cached wrapper build output, checks exact stderr/exit 1/empty stdout, checks provider/Curator sentinels, and compares project/home snapshots without exclusions. This directly covers the prior failure.

Ordinary setup/refresh exclude .instructions, .skills, skills, and the bundled registry; removed fan-out/validation helpers have no remaining references. The source registry is deleted. Preparation alone reaches the isolated v1 renderer/scaffold; compose retains strict caller-supplied MCP registry handling. No golden/testdata files differ from base. Residual settings, Codex config modes/merge, rules, receipt/backend requirements, Pi runtime subcommands and attachments remain. Read-only source review plus targeted setup/prepare tests support this separation; full residual coverage is accepted from the exact-tree hosted gate.

README/SKILL/CHANGELOG document the residual ownership, compatibility exception, no print-config escape, and next-release removal. Version convention is git-tag injection: CHANGELOG proposes v2.0.0, with publication left to the release owner as the accepted plan explicitly requires; no fabricated version or tag was introduced. LLDB has no bundled implementation to retain.

CI is the requested Ubuntu/macOS build-vet-test matrix. The remote gate differs from its attached source only in header comments; sh -n passes. The task-board absence skip has the requested reason. Cancellation fixtures resolve sleep absolutely; inode-substitution fixtures pin the original inode and assert replacement identity. The stress test uses a separate Go child instead of Python and retains readiness/RSS/prefill/shutdown assertions; it does not introduce a GITHUB_ACTIONS skip. These are fixture changes, not weakened production behavior.

## Independent verification

- go build ./... && go vet ./...: exit 0.
- go test . -run 'TestDeprecated|TestRunCodexAndClaudeAreDeprecated|TestRunCompose|TestRunPrepare' -count=1 -timeout 5m: exit 0, 14.012s.
- go test ./internal/infra -run 'TestDeprecated|TestCLIWrapper|TestLiveCanonicalWrapper|TestCodexLocalLauncher|TestBuildChildLaunchComposition|TestPreparePrimarySession|TestSetupAndRefreshLinksLeaveSkillSurfacesUntouched|TestSetupGlobalDistributesNoInstructionsSkillsOrRegistry|TestSetupLocal.*CodexConfig' -count=1 -timeout 5m: exit 0, 13.854s.
- Every existing candidate file was hash-compared with the candidate tree: zero mismatches. Repository files were never edited during this review.
- Accepted existing full-suite evidence: https://github.com/relux-works/relux-agents-infra/actions/runs/35067793801 — both Ubuntu and macOS jobs success, independently queried. Its head 008bb38d9258c56e4c23bfae98978837f10a3b15 resolves to exactly candidate tree d91f1a1e3a7fd2420a586a127d774c332e975f18. The workflow runs gofmt, build, vet and go test ./... -count=1 -timeout 20m. The full suite was not repeated locally.

## Adversarial evidence

Mutations use Go overlays stored outside the repository, so candidate bytes never require restoration.

1. Narrow canonical-wrapper refusal to openai-infra only: KILLED, exit 1. TestDeprecatedCanonicalAliasesRefuseWithoutSibling/anthropic-infra observed successful delegation instead of exit 1; both platform string cases also failed. OpenAI was not removed from the guard.
2. Narrow compose validation by allowing enabled servers without definitions to continue: first probe INCONCLUSIVE, exit 1 after the Go runner killed the process at 180 seconds without any test assertion output; a shorter retry also produced no assertions and was killed by the runner at 90 seconds (exit 1). Reviewer mutation outcome is therefore 1/2 killed, 1/2 inconclusive, 0 demonstrated survivors; producer records 2/2 killed in prior evidence. This is not a killed or surviving mutant. Producer revision-1 evidence separately records this same negative case killed; it is historical evidence, not a reviewer rerun claim.

Bounds: POSIX execution verified; Windows wrapper bodies are source/string-covered only, with no native Windows runtime execution. No live model/provider/authentication, host installer, or external task-board decoder replay was performed. Existing consumer compatibility evidence and the schema tests are accepted within those limits. Filesystem snapshots cover fixture project/home, not an OS-wide syscall audit.

## Review logbook

2026-09-16: Prior build-before-refusal defect independently verified fixed. Exact-tree hosted evidence confirmed. Local fresh-process startup stalls also affected optional board-help/directive reads and one mutation attempt; these are not passing test evidence. No production workaround or repository edit made. This task-scoped logbook records the anomaly without editing repository LOGBOOK.md. spawn goal reported this run is not goal-bound. Acceptance must be persisted via accept_cr revision=6 after attaching this resource; producer-side integration remains outstanding.

Host-stall evidence: sample of the pending resource-add shell showed all 818 samples at _dyld_start, before board CLI execution. Optional duplicate goal/help/directive probes were terminated; the original successful spawn-goal result remains authoritative.
