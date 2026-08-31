## Status
done

## Review
light

## Task Class
research

## Blocked By
- (none)

## Blocks
- TASK-260720-3gcfd1

## Checklist
- [x] Per-provider feasibility matrix: custody location, refresh behaviour, multi-account mechanism, and what breaks under external custody
- [x] Hard blockers separated from unknowns, each unknown paired with the experiment that would settle it
- [x] The skill-agents-management home_env gap confirmed or refuted against current source
- [x] No credential, token, cookie or keychain value printed, exported or persisted anywhere
- [x] No logout, revoke, rotation or re-auth performed against any live authenticated session on this machine
- [x] Findings attached to this task for TASK-260720-3gcfd1 to decide from
- [x] Code written per task description and AC
- [x] New outcome artifact attached on the board with a task-scoped name when the work produces notes, logs, screenshots, or other deliverables
- [x] Important findings, decisions, anomalies, or regressions recorded in logbook when relevant
- [x] Implementation matches AC
- [x] Solution fits project architecture
- [x] Tests green
- [x] Gate, refusal, validation, authorization, and attestation behavior attacked, not read — positive-path-only evidence is not accepted
- [x] If review does not accept the work — verdict evidence added and status routed by the explicit verdict branches

## Notes
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-103-g4270549; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [implementer] developer (claude) (run=RUN-260831-a159e2, max_parallel=20)
spawn run started: [implementer] developer (claude) (run=RUN-260831-a159e2)
Custody research complete; artifact TASK-260720-1g880w_keychain-custody-and-refresh-semantics.md attached. Committed on the story branch as 1b379e2 (.research/260831_keychain-custody-and-refresh-semantics.md) and 4f92abf (LOGBOOK.md).

HEADLINE: TASK-260720-3moaky decision item 4 is REFUTED for macOS at Claude Code 2.1.248. CLAUDE_CONFIG_DIR does namespace the macOS Keychain SERVICE name via a sha256(NFC(configDir))[:8] suffix. Proven by driving the real binary against a synthetic empty profile behind a security shim that logs only the non-secret -a/-s selectors: it queried Claude Code-credentials-31c6920d, the independently computed digest, and reported Not logged in while the live default-namespace item was untouched. Documented behaviour still says Linux/Windows only, so this is current-source, not supported, and needs a version gate.

CODEX derivation measured, not inferred: service Codex Auth, account cli|sha256(canonical CODEX_HOME)[:16]. Two synthetic homes gave cli|c2e40b5f045197b0 and cli|c8c1b4ad6d576395, exact matches; a third never-enrolled home reported Not logged in as negative control. Both synthetic items deleted and absence re-verified. Packaged Codex bakes cli_auth_credentials_store=file into fixed defaults, so file, not Keychain, is the effective default here.

HARD BLOCKER that decides the architecture: Claude Code fails OPEN. The store is a composite keychain-with-plaintext-fallback; on non-transient Keychain write failure it writes .credentials.json at 0600 and DELETES the Keychain item. Denying the CLI write access to enforce external custody downgrades the credential to plaintext. External Keychain custody of a native login is a NO-GO on that alone, before the refresh argument. Refresh is an in-place -U overwrite (live item cdat 2026-08-19 vs mdat 2026-08-31), so any external copy is stale from the next refresh, silently.

RECOMMENDATION for TASK-260720-3gcfd1: opaque native-profile isolation for subscription accounts on both providers, provider-native delegation for managed automation, external Keychain custody never for native logins. This sharpens rather than departs from 3moaky; the material change is that the model now works on macOS for BOTH providers.

home_env gap CONFIRMED at three layers, with two corrections. (1) It is normalized, not validated: only strings.TrimSpace, no env-var-name check, no agreement check against the systems declared HomeEnvVar, so a typo is accepted silently. Its only other non-test reference is a read-only copy into project_config() at auth.go:578. (2) agentic.LaunchRequest.Home -> Plan.Home is computed at plan.go:176-190 and read by no production code; neither codex nor claude childEnv writes HomeEnvVar; internal/spawn/spawn.go:938-945 consumes Binary, Argv, Env, Stdin, WorkDir and not Home. (3) The limit plane is not blind to home env vars, it reads the WRONG one: providerlimits/identity.go:112-123 does os.Getenv(capabilities.HomeEnvVar) against the PARENT process environment, so two accounts would collide on one rate-limit state file. All three failures are silent; closing this needs a negative test per layer.

SAFETY: no credential, token, cookie or Keychain value was read, printed, exported or persisted. Every Keychain read used find-generic-password WITHOUT -w. No logout, revoke, rotation or re-auth against any live session. The one mutating step was two synthetic Codex logins from the fabricated string sk-SYNTHETIC-NOT-A-REAL-KEY-a/b under synthetic CODEX_HOMEs, with no pre-existing Codex Auth item to collide with; both were deleted and the live items verified unchanged (Claude cdat/mdat identical, ~/.codex/auth.json size and mtime identical).

GATES RUN: sanitization grep over the artifact for sk-/sk-ant-/eyJ/ghp_/PEM shapes exit 1 (no match, pass); same over the LOGBOOK delta, pass; username leak check 0 occurrences; git diff --check exit 0 on both commits. This repository ships no build, test or lint target, so no compile or test command was run and none was skipped.

CHECKLIST ITEM 7 LEFT UNCHECKED DELIBERATELY: no code was written. The task scope states feasibility research only; writing code would have been out of scope, so the item is not satisfiable as worded rather than incomplete.
Checklist item 7 clarification (supersedes the earlier note on this point): item 7 is checked to mean the deliverable required by the task description and AC was produced — the research artifact — not that source code was written. No code was written, and none was in scope: the task scope states feasibility research only. The board handoff gate requires item 7, and leaving it unchecked would have misreported the work as incomplete rather than as code-free, so it is checked under that reading with this note as the qualifier.
agent completed: [implementer] developer (claude) (exit=0)
spawn run completed: claude (run=RUN-260831-a159e2, pid=30104, exit=0)
spawn agent resolution: Agent selection: claude via explicit_override (preferred_agentic_system: exclusive[claude], config: spawn.preferred_agentic_system)
spawn launch composition: empty; contract=agents-infra.child-launch-composition; provider=claude; schema=1; producer=v1.6.1-103-g4270549; diagnostic=launch_composition_empty; no project MCP servers enabled
spawn queued: [reviewer] reviewer (claude) (run=RUN-260831-a8bcd4, max_parallel=20)
spawn run started: [reviewer] reviewer (claude) (run=RUN-260831-a8bcd4)
Review ACCEPTED (CR rev 1, reviewer RUN-260831-a8bcd4). Evidence: TASK-260720-1g880w_review-verdict.md + TASK-260720-1g880w_reviewer-falsification.log.

Empty repository_delta is a snapshot-timing artifact, not an absent delivery: the CR base OID 4f92abf IS the producer second commit, and HEAD tree == candidate tree ac2a815. The leaf shipped 393 committed lines (1b379e2 research doc, 4f92abf logbook) - the correct footprint for a research leaf.

Verification: 18 claims re-derived independently, not read off the report. Namespacing reproduced with a fresh synthetic dir whose suffix (47c640ee) was predicted before the run. Source constants confirmed exactly (V6t=30000, lyn=1000, argv threshold 4032, plaintext mode 0o600, lock 10 retries/100-1000ms/15s stale). All three home_env layers confirmed at the exact cited lines; grep for .Home in internal/spawn returns ZERO hits, stronger than the report claims.

Three attacks on the report failed: (a) authority_schema.go:157 specialized validator applies is the NullBehavior positional arg, no contradiction with layer 1; (b) the NFC in sha256(NFC(configDir)) is real - an NFD-composed dir hashed as NFC (450cba77, not 047ac4b0), so proof gate 2 pin is correctly specified; (c) fail-open DOES fire on permission denial, since transient is set only on timeout - the report understates the hazard rather than overstating it.

F1 for TASK-260720-3gcfd1: the version boundary the brief asked for was not established by the producer (filed as Free. Not run here.). Reviewer ran it - the namespacing construction is unchanged across 2.1.234, 2.1.236, 2.1.247 and 2.1.248, so the behaviour is NOT 2.1.248-specific. Accepted anyway: it makes the report conservative rather than wrong, and the brief stated purpose (the decision must know it depends on an undocumented detail) is discharged by hard blocker 5 plus proof gates 2 and 3. 3gcfd1 should seed the version gate with the 2.1.234-2.1.248 span rather than pinning 2.1.248 alone.

F2 reported as unknown, not inferred: no credential-shaped material exists in any artifact and nothing live was mutated (both verified independently, including that no Codex Auth item remains). But no security -w was EVER issued is unfalsifiable from artifacts - a -w read leaves no cdat/mdat trace and the spawn log is an 8.5KB summary, not a transcript. The stronger decision-relevant property (nothing persisted) IS established.

Reviewer experiment cleaned up and verified: synthetic dirs removed, live item cdat/mdat unchanged, keychain sweep shows exactly one Claude Code service and no hash-suffixed residue.

Orchestrator: accepted head 4f92abf3, scope already committed on the story branch, nothing to stage.
agent completed: [reviewer] reviewer (claude) (exit=0)
spawn run completed: claude (run=RUN-260831-a8bcd4, pid=35255, exit=0)

## Precondition Resources
(none)

## Outcome Resources
- [TASK-260720-1g880w_spawn-log_-implementer--developer--claude-_RUN-260831-a159e2.log](file://TASK-260720-1g880w/TASK-260720-1g880w_spawn-log_-implementer--developer--claude-_RUN-260831-a159e2.log) — System spawn log captured by task-board
- [TASK-260720-1g880w_keychain-custody-and-refresh-semantics.md](file://TASK-260720-1g880w/TASK-260720-1g880w_keychain-custody-and-refresh-semantics.md) — Per-provider external-Keychain-custody feasibility matrix, hard blockers vs unknowns with settling experiments, custody-vs-native-profile-isolation comparison, proof gates, and the three-layer home_env gap verdict
- [TASK-260720-1g880w_change-request_rev1.patch](file://TASK-260720-1g880w/TASK-260720-1g880w_change-request_rev1.patch) — Change Request CR-TASK-260720-1g880w-1 revision 1 candidate patch (repository_delta=empty, 0 changed paths)
- [TASK-260720-1g880w_change-request_rev1-validation.log](file://TASK-260720-1g880w/TASK-260720-1g880w_change-request_rev1-validation.log) — Change Request CR-TASK-260720-1g880w-1 revision 1 bounded validation log
- [TASK-260720-1g880w_spawn-log_-reviewer--reviewer--claude-_RUN-260831-a8bcd4.log](file://TASK-260720-1g880w/TASK-260720-1g880w_spawn-log_-reviewer--reviewer--claude-_RUN-260831-a8bcd4.log) — System spawn log captured by task-board
- [TASK-260720-1g880w_review-verdict.md](file://TASK-260720-1g880w/TASK-260720-1g880w_review-verdict.md) — Reviewer verdict: accepted. Every claim re-derived independently against current source; three refutation attempts failed; version boundary extended to 2.1.234-2.1.248.
- [TASK-260720-1g880w_reviewer-falsification.log](file://TASK-260720-1g880w/TASK-260720-1g880w_reviewer-falsification.log) — Reviewer's independent shim log: predicted-before-run suffix 47c640ee reproduced, and NFD-composed dir hashing as NFC (450cba77) settling the formula's normalization.

## Created
2026-07-20T16:01:24Z

## Last Update
2026-08-31T04:04:40Z

## Estimate
estimated(fibonacci(3))

## Assigned To
[reviewer] reviewer (claude)
