# Review verdict: accepted

Task: `TASK-260810-3833d1`

## Verdict

Accepted. The implementation satisfies the task scope and acceptance criteria.
No code changes are requested.

## Production-path evidence

- Reviewed commits `a51cb4b` and `57a6148`; `git show --check` passes.
- Confirmed the production call chain is `Setup -> syncRepo -> ensureLocalInstructionScaffold -> RefreshLinks`, rather than an isolated helper-only path.
- A real `agents-infra setup local` into an external `/tmp` project created exactly two project-owned instruction files: `AGENTS.md` and `INSTRUCTIONS.md`.
- The fresh local instruction tree and both rendered Codex surfaces contained none of the shared workflow, attachment, or dirty-checkout policy markers.
- Two repeated real local setup runs preserved four handwritten local instruction files byte-for-byte and rendered their project marker into both Codex entrypoints.
- A handwritten project-root `AGENTS.md` was preserved as `AGENTS.project.md`, rendered through both Codex surfaces, and remained byte-identical after resync.
- Real `prepare` calls for Codex and Claude resolved the external project runtime, rendered/linked the correct provider surfaces, and did not reintroduce shared policy.
- Real `codex --print-config` and `claude --print-config` calls completed against the external project runtime.
- A real isolated global setup retained shared instruction modules and rendered shared policy, proving global behavior was unchanged.

## Negative evidence

- **Bypass path around the check:** exercised fresh setup, repeated setup, `prepare` for both providers, and both primary-session print-config entrypoints. No local path copied or rendered shared global policy.
- **Absent evidence treated as satisfied:** removed the local Codex instruction entrypoint and confirmed setup recreated only that missing scaffold while preserving the handwritten Claude entrypoint.
- **Failure to read treated as absence:** supplied a local Codex entrypoint with a missing include. Setup failed, removed the completed-install receipt, and `verify local` also failed; no permissive fallback was taken.
- **Check present but uncalled from production:** defeated by driving the compiled CLI and inspecting the composed runtime artifacts, not by calling the helper directly.
- The first in-repository temp probe was explicitly discarded because ancestor runtime discovery made it invalid. The accepted evidence comes from the rerun under external `/tmp`, whose JSON reports the exact external `runtime_project_dir`.

## Verification

- Focused setup tests with `-count=1`: passed.
- `go test ./... -count=1`: passed, including the full infra package.
- `go vet ./...`: passed.
- `go build ./...`: passed.
- `git diff --check`: passed.
- Pre-existing changes were separated into coherent commits `837e8df` and `b10d3b7`; their commit checks pass and the full suite passes on the composed checkout.

## Compatibility note

Existing local instruction files are intentionally preserved, including files
left by an older setup. This is consistent with the preserve-all project-owned
overlay contract. The reviewed change prevents every new/resync production path
from copying source `.instructions`; it does not attempt an unsafe provenance-
free deletion of possible handwritten overrides.
