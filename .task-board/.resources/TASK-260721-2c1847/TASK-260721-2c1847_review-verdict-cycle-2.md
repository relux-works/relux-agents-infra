# Review verdict — changes requested (cycle 2)

## Finding
The Go attachment helper preserves a usage-code marker internally, but the top-level CLI discards it: tools/agents-infra/main.go prints every error and exits 1. The legacy Python helper returned usage() with code 2.

## Reproduction
Built current source: cd tools/agents-infra && go build -o ../../.temp/TASK-260721-2c1847/agents-infra-review . && ../../.temp/TASK-260721-2c1847/agents-infra-review attachments. It prints the helper Usage text and exit code 2, but the process exit status is 1.

## Positive checks
Go tests and vet pass. The attachment package reports 71.6 percent coverage. Global and casual-talks agents-attachments launchers are Go-backed shell launchers with no Python reference; both doctor commands report helpers_linked true.

## Required rework
1. Preserve usage exit status 2 at the agents-infra process boundary, while keeping operational errors at status 1.
2. Add a Go regression test for that process-boundary or extracted exit-code mapping.
3. Re-run the full Go test, coverage, vet, and installed-launcher smoke gates.

## Verdict
Changes requested. Route to developer rework, then run a fresh reviewer cycle.