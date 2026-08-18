# Reviewer Verdict — Cycle 10

Task: `TASK-260817-2h8hn4`
Verdict: **accepted**
Route: `done`

## Acceptance evidence

- Re-fetched the three official Pi documents from immutable revision `10acee6045e9025a22dff7e5220ed0d7538f12aa`. Their SHA-256 values reproduce the contract: `models.md` `3ab68dd...0f3`, `settings.md` `f36d3a9...973b`, and `usage.md` `a6a76e7...abca`. The linked current official pages also confirm `models.json` custom providers, project-over-global settings, trust behavior, model CLI options, and `--session-dir`.
- The English decision artifact specifies the exact project TOML, atomic nearest-profile composition, wrapper/profile precedence, exact managed provider/model override rules, equal-form handling, fake-separator refusal, non-launching diagnostics, and explicit absence-versus-read-failure behavior.
- The cycle-8 directive is preserved: reviewed absolute `runtime.executable` plus literal argv is trusted policy; launch guarantees are limited to reproducible config, no-shell absolute spawn, loopback preflight, direct-child liveness, exact model discovery, process-group cleanup, isolated Pi state, and no intentional attach/fallback. Qwen and Muse/DFlash capability labels remain requested/configured and unverified.
- The cycle-10 state boundary uses SHA-256 of exact decoded UTF-8 profile bytes and hash-only contained paths. It requires collision detection, anchored no-follow operations, complete read/revalidation, and pre-side-effect refusal. The named production-entry scenario covers traversal, separators, dot names, absolute-looking names, Unicode lookalikes, case/NFC/NFD variants, symlinks, partial reads, simulated collisions, and raw/lossy narrowing mutants.
- Gate-defeat review covered the standard shapes: exact-identity mismatch and separator lookalikes; post-delimiter provider/model/api-key/trust bypass attempts; absent/malformed/partial reads; foreign ready listener with dead selected child; runtime self-reported DFlash; raw/lossy profile-state aliases; canonicalization narrowing; direct-PID-only cleanup; and bypass paths through diagnostics versus launch. The contract requires real production-entry evidence for implementation and does not accept helper-only or positive-only proof.
- Independently regenerated the official standalone tree manifest from the retained release extraction. It matches the attached 217-record manifest byte-for-byte; the tree has 217 regular files and 34 directories, the entrypoint is native arm64 Mach-O with SHA-256 `d5de3fe...f044`, and all three task-scoped manifest copies have SHA-256 `2f68ab1...378b`.
- The decision source, task outcome, and both downstream precondition copies have identical SHA-256 `b9d9259...fb2d`. Downstream descriptions, scopes, and AC exclude cycle-7 observer/catalog/attestation work and carry the cycle-8 practical boundary plus cycle-10 state-key requirements. Historical cycle-7 checklist items are explicitly recorded as retired, while current AC and replacement checklist items are unambiguous.
- The three-task chain remains the smallest complete decomposition: decision -> launcher implementation -> alias/operator documentation. Dependencies are present, planning evidence is linked, `task-board validate` passes, and `git diff --check` passes.

No implementation exists in this task by scope, so product test execution belongs to `TASK-260817-ccpnlm`; this verdict accepts the implementation-ready decision and its executable negative acceptance contract.
