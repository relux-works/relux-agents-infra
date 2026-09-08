# TASK-260830-tvy8q5 review verdict — revision 3

## Verdict

Changes requested. Route to `to-dev`.

Immutable review surface:

- Change Request: `CR-TASK-260830-tvy8q5-3`, revision 3
- Base OID: `b78498bf98c05175db10bb341aee621e53de4881`
- Candidate tree OID: `1f94cdbf728846a3dc35be0a1546075db13e7774`
- Patch SHA-256: `c422fa0679ed78e71ce3976456b29eaffc2a37f7c92c561cece470564711a319`
- A fresh `git archive` of the candidate and the managed worktree matched byte-for-byte outside `.git`, `.temp`, and the non-authoritative `.task-board` checkout artifact (`diff -qr` exit 0).

## F1 — Odd-generation resume bypasses policy-provenance binding

Severity: blocking exact-confirmation/authorization defect.

Negative shape: **bypass path around the check**. Fresh confirmation hashes `PolicySource` as part of the complete `PiLegacyRetirementPlan`, but the odd-generation resume branch validates only the previously stored `PlanHash` and the numeric `PolicyDigest`. The current policy provenance supplied by the production caller is never compared with the confirmed plan.

Production call site: `PiLegacyRetire` odd-generation resume branch in `tools/agents-infra/internal/infra/pi_lifecycle_legacy.go:480`. Fresh execution rebuilds the full plan with current `policySource` at line 485, while resume skips that path and does not otherwise bind the current source. The public contract in `SKILL.md:485` says the exact plan hash covers policy provenance and that resume requires the same exact hash.

Deterministic exact-candidate reproduction through the production entry point:

1. Build a dry-run plan with numeric policy P and `policySource="source-a"`.
2. Confirm it and interrupt after the operation-bound rename, leaving the odd generation.
3. Resume with the same numeric policy P, the same old confirmation hash, and `policySource="source-b"`.

Actual result: resume returns `err=nil`, unlinks the candidate, advances the legacy generation to even, reports `RetiredCount=1`, and publishes `WithinPolicy=true` / `SoakReady=true` under `PolicySource=source-b`.

Expected result: changed current provenance must refuse before mutation with `lifecycle_legacy_confirmation_mismatch` (or an equally stable typed refusal), preserve the tombstone and odd authority, and publish neither retirement nor readiness.

Required revision 4:

- Bind odd-generation resume to the current policy provenance covered by the dry-run hash. Persist and validate the canonical source or another non-self-mintable projection that lets resume prove the current full-plan provenance; update generation validation and bounded control-size preflight accordingly.
- Add a production-entry crash/resume test for identical numeric limits plus changed `policySource`; require refusal and preservation before any unlink.
- Prove the new guard with an uncached narrowing mutant, not deletion only.
- Preserve revision 2 external dual-absence refusal and revision 3 two-progress-window crash liveness.
- Rerun focused legacy/automatic/eight-week tests, race, full repository, vet/build/format/diff, and Darwin/Linux/Windows compile gates.

## Reviewer validation

- Revision-3 patch resource SHA-256 matched the handed-off digest.
- Candidate archive versus managed worktree: exit 0, no drift.
- `git diff --check <base> <candidate>`: exit 0.
- Shipped external-absence, ordinary post-unlink, and two-progress-window crash tests: exit 0 (`1.636s`).
- Focused legacy/automatic-path/eight-week soak suite: exit 0 (`5.529s`).
- Isolated reviewer policy-source drift probe: expected rejection assertion failed because production resumed and published readiness; transcript: `TASK-260830-tvy8q5_reviewer-policy-source-probe.log`.
- Exploratory forged-tombstone and post-first-mutation directory-drift probes were not promoted to findings: both still target the exact confirmed inode, preserve unrelated evidence, and do not bypass candidate/path authority. They are excluded from the required rework.
- No Pi executable, installed runtime, model, provider process, socket, endpoint, setup/install flow, or live user-HOME runtime state was contacted. However, the first reviewer test command used a fresh isolated Go module cache and downloaded `github.com/pelletier/go-toml/v2` and `golang.org/x/sys`; this was external module-service contact and violates the assignment's no-network-service execution constraint. Later probes ran with `GOPROXY=off` and `GOSUMDB=off`. This deviation is recorded rather than hidden.

No tracked repository source, test, documentation, or logbook file was modified by the reviewer. Probe source, overlay, and logs exist only under ignored `.temp/` review evidence.
