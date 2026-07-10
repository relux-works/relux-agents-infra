## Status
done

## Assigned To
codex-inline

## Created
2026-07-10T08:38:16Z

## Last Update
2026-07-10T08:54:13Z

## Blocked By
- (none)

## Blocks
- (none)

## Checklist
- [x] Audit Codex runtime config and source for model fallback chooser controls
- [x] Implement source-managed runtime config when supported
- [x] Add durable global autonomous retry and fallback instructions
- [x] Update tests and generated instruction snapshots
- [x] Run tests setup install and drift verification

## Notes
Implementation started inline by explicit human request. Existing unrelated worktree changes will be preserved.
Audited Codex CLI 0.144.1 source. Supported rate-limit nudge suppression was already source-managed; safety-buffering chooser has no supported config key. Added autonomous retry/fallback instructions, README limitation, and renderer/config regression assertions. go test ./... and go vet ./... pass; temp-home setup/doctor/render checks pass.
Canonical ./setup.sh completed successfully. Installed doctor is healthy; source/runtime cmp is drift-free for workflow instructions, Codex config, and README; installed Codex prompt-input contains policy and no raw includes. Ready for review.
Independent review accepted. Verified Codex rust-v0.144.1 source and schema: notice.hide_rate_limit_model_nudge is supported; the separate safety-buffering chooser has no supported suppression or auto-wait config and no terminal automation or binary patch was added. Verified no human wait-versus-weaker-model handoff, at least three preferred-model retries with backoff, autonomous best fallback, fresh temp-home materialization, current installed runtime, uncached full and focused tests, vet, board validation, and diff hygiene. Review logs: .temp/review-TASK-260710-381t1q-TASK-260625-uwtoga/.

## Precondition Resources
- [agents-model-fallback-policy-audit.md](file://TASK-260710-381t1q/agents-model-fallback-policy-audit.md) — Codex 0.144.1 runtime/config audit and implementation plan

## Outcome Resources
- [model-fallback-policy-outcome.md](file://TASK-260710-381t1q/model-fallback-policy-outcome.md) — Implementation, runtime limitation, tests, reinstall, and drift evidence
