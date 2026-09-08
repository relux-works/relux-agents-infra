# M2 current-trunk gap audit: long-horizon local Qwen stability

Audit date: 2026-08-29, Europe/Moscow. This was a read-only static audit. No live model process, service, socket, endpoint, board state, Git ref, or worktree was touched. The only writes were the tool-readiness log and this report under `.temp/TASK-260828-3hultd/`.

## Executive verdict

**M2 is not ready on current trunk.** The low-level shared Pi runtime now has a strong landed recovery foundation: persisted restart/quarantine state, bounded exponential retry delay, automatic half-open, manual quarantine, connection-bound leases, heartbeat handling, draining, stale-process attestation/reclaim, and production-seam abrupt-client lease-release tests. However:

1. the local-model component and task-board consumer are not landed;
2. shared broker/runtime logs are still unbounded on trunk;
3. automatically-created per-turn Pi lifecycle logs have no retention policy;
4. the public status surface omits the active backoff deadline even though the internal ledger stores it;
5. memory/resource pressure and inference-busy semantics are not implemented or explicitly tracked as their own deliverable;
6. no deterministic hours/days/weeks proof or controlled real soak proof exists.

The current backlog is directionally correct but incomplete. In particular, `TASK-260829-1kpj01` cannot honestly produce a validator-compliant Backoff/`AvailabilityLimited` verdict from only `restart_count`, `quarantined_until`, and `last_readiness_match`: `AvailabilityLimited` requires an evidence-derived non-zero `Until`, while current infra keeps `restart_not_before` private in the ledger.

## Exact published baselines

`git ls-remote origin refs/heads/main` was run twice. Both reads matched each repository's cached `origin/main`; therefore all code citations below are against locally available objects that exactly match the published branch.

| Repository | Published `origin/main` | Commit time | Subject |
| --- | --- | --- | --- |
| `relux-agents-infra` | `891de4427bb7de6885b8b221f0e2b24a49a8fdc2` | 2026-08-29T17:11:16+03:00 | `Add shared runtime restart supervision (#9)` |
| `skill-agents-management` | `1e9f203201d8317cd55c155104205ce97db38fa7` | 2026-08-23T23:52:00+03:00 | `Record the goal's completion and the follow-up ledger` |
| `skill-project-management` | `4fbb459c3cff89f198dd5f4fcc7ac3a587cbd357` | 2026-08-29T16:55:29+03:00 | `Add audited checklist item removal (#43)` |

Tool readiness was recorded in `.temp/TASK-260828-3hultd/m2-audit-tool-readiness.log`: `git version 2.50.1 (Apple Git-155)` and a functioning `task-board` query surface.

## What is actually landed

### `relux-agents-infra`

At `891de44`:

- `pi_shared_supervision.go` persists schema `agents-infra.pi.shared-runtime.restart-ledger.v1` with `restart_count`, `restart_not_before`, `quarantined_until`, `last_readiness_match`, `manual_quarantine`, and `half_open`.
- `sharedRuntimeRecordFailure` implements capped exponential delay, a restart budget, quarantine, and half-open failure handling; `sharedRuntimeResetStableRun` resets the counter after a configured stable run.
- `SharedRuntimeStatus` publishes `restart_count`, `quarantined_until`, `last_readiness_match`, and `manual_quarantine` alongside broker, runtime, lease, attestation, and path facts.
- The status surface does **not** publish `restart_not_before`, `half_open`, or a last-failure reason/time.
- Heartbeats and connection-bound leases are present. The broker refuses new leases while `draining`; abrupt connection closure releases a completed lease. The merged tests include real production seams for `handleConnection`, `runSharedPiSession`, automatic cross-broker restart, manual quarantine, corrupt records, stale PID/start-time identity, and orphan refusal/reclaim.
- Existing shared-runtime ownership already provides election, reconnect ladders, stale owner attestation, lease mirrors, stop/drain, and host-restart cold recovery. `d3dfde5` also configures Pi compaction for long-lived sessions, but that is context-management configuration, not a sustained-operation proof.
- `broker.log` and `runtime.log` are opened with `O_APPEND`; there is no cap or rotation on trunk.
- Every Pi turn creates a unique mode-0600 JSONL file in the profile `logs/` directory (`pi_session_log.go`). The file contains a bounded number of lifecycle events per turn, but the directory has no count/age/byte retention or pruning path.
- `model-check` is deadline-bounded and writes explicit mode-0600 raw and sanitized artifacts to an operator-selected directory. It is useful for behavior checks, but it is not a long-horizon soak harness and its raw files are not a global retention solution.

### `skill-agents-management`

At `1e9f203`:

- The generic `agentic` and `vendorplugin` registries and `vendorplugin.BuildLaunch` exist.
- There is no landed `pkg/localruntime`, Pi local-model system plugin, `local-models` vendor, or `local-qwen` runtime declaration. The shipped-state documentation explicitly says the local-model plane is not built.
- `TASK-260829-188jcf` is still `development`. Its revision-4 candidate contains the M1 packages, but independent review requested changes because conflicting system-only model/effort declarations can return success while retaining the first authority. A rejected candidate is not trunk capability.

### `skill-project-management`

At `4fbb459`:

- The board consumes `skill-agents-management v0.2.0` in both `go.mod` and managed vendor state.
- The production `launchRegistry -> buildLaunchPlan` path still builds an `agentic.Registry` and calls `agentic.BuildPlan`; it does not call `vendorplugin.BuildLaunch`.
- No local-Qwen runtime/config/status fields exist on trunk. Local/remote runtime policy parity and task-aware model recommendations have landed, but local Qwen is not among their published runtime rows.
- `TASK-260828-3hultd` remains `development`; its failed CR is an unmerged candidate and cannot be counted as delivered consumer behavior.

## Board-to-capability map

| Capability | Existing owner/task | Current board state | Current-trunk result |
| --- | --- | --- | --- |
| Shared leases, heartbeat, broker election, stale owner/orphan recovery | infra `STORY-260825-1r7z9o` | `done` | Landed before `891de44`; strengthened by the latest production-seam tests |
| Restart/quarantine ledger and status facts | infra `TASK-260829-2t5xmi` | `done` | Landed in `891de44` |
| Shared broker/runtime log rotation | infra `TASK-260829-3fozxa` under `STORY-260829-wk6p9a` | `development` | Not in `origin/main`; current trunk remains append-only |
| M1 Pi/local-models/local-Qwen component | agents `TASK-260829-188jcf` | `development` | Not landed; revision 4 has a declaration-authority blocker |
| Component status widening | agents `TASK-260829-1kpj01` | `backlog` | Not landed; its note still says infra Handoff A is unscheduled even though `891de44` has now landed it |
| Deterministic long-horizon component/consumer proof | agents `TASK-260829-lx3o55` | `backlog`, blocked by `TASK-260829-188jcf` and `TASK-260829-1kpj01` | Not started |
| Generic task-board launch/config consumer | board `TASK-260828-3hultd` | `development` | Not landed; current trunk remains on the old `agentic.BuildPlan` boundary |
| Fresh consumer Story placeholder | board `STORY-260829-bytp8h` | `backlog`, no children | Does not currently own the task in structured board state; `TASK-260828-3hultd` still reports parent `STORY-260829-6sxbuo` |
| Resource pressure / inference-busy admission | only broad prose in board `TASK-260828-3hultd` and the architecture documents | no dedicated implementation task | Not implemented and not independently schedulable |
| Per-turn lifecycle-log retention | none found | untracked | Not implemented |
| Controlled real soak/SLO evidence | broad prose in board `TASK-260828-3hultd`; agents `TASK-260829-lx3o55` is fake/deterministic-only | no explicit real-soak task | Not delivered |

## M2 stability matrix

| Dimension | Landed foundation | Gap before M2 confidence |
| --- | --- | --- |
| Hours | Heartbeat, leases, reconnect/election retry, crash restart, stable-run reset, Pi compaction configuration | No M1 consumer on trunk and no elapsed or accelerated hour-scale SLO proof |
| Days | Persisted restart ledger survives broker lifetimes; stale PID/orphan paths are guarded | Shared logs are unbounded; lifecycle-log file count is unbounded; no day-scale simulation result |
| Weeks | Automatic quarantine/half-open and manual operator quarantine prevent an infinite hot crash loop | No week-scale simulation, no bounded artifact footprint, no pressure model, no controlled real soak |
| Recovery | Cold start, broker/process crash, host-restart stale-owner recovery, abrupt-client lease cleanup are substantially covered in infra | Task ownership/retry/resume behavior is not exercised through a landed task-board local-Qwen consumer |
| Restart/quarantine status widening | Infra publishes restart count, quarantine deadline, readiness match, and manual quarantine | Component task is backlog; public status omits active backoff deadline and half-open state |
| Backoff/circuit breaking | Capped exponential delay plus restart budget, automatic quarantine, half-open, stable reset | No jitter despite the broad board AC; no public `restart_not_before`; no last failure provenance; consumer cannot schedule a truthful retry from current public fields |
| Heartbeat/lease/orphan | Connection-bound lease, heartbeat deadlines, lease mirrors, abrupt-disconnect release, PID/start-time/UID/PGID identity checks, reclaim/refusal | No component/task-board end-to-end recovery run on released dependencies; sleep/suspend and long pause behavior lacks an explicit soak/SLO case |
| Draining | Attested `draining` state refuses acquisition and broker shutdown closes connections | Component mapping exists only in rejected candidate; no landed typed task-board refusal/provenance |
| Resource pressure | `max_leases` gives concurrency capacity and the M1 candidate maps at-capacity to `Unknown` | No memory pressure, loaded-model footprint, inference-busy, eviction/drain sequencing, or typed pressure refusal exists |
| Observability | Infra status exposes broker/runtime identity, uptime, runtime argv/cwd/endpoint, configured/effective sharing, leases, attestation, restart count, quarantine, readiness timestamp, and paths | Missing active backoff deadline, half-open, last failure, health/utilization/memory/inference-busy facts, log footprint/rotation state; no released component/board projection |
| Logs/artifacts | Mode-0600, no-follow, single-link checks; model-check summaries cap final-response text | Append-only broker/runtime logs on trunk; unlimited lifecycle-log file count; no aggregate byte/count retention status |
| Consumer widening | Task-aware recommendations and local/remote config authority foundations landed in board | Released component and board generic runtime consumer absent; M2 status projection has no dedicated board task |
| Soak proof | Deterministic fake seams exist across infra and the proposed component design | No composed hours/days/weeks suite, no SLOs, no controlled real soak, no archived proof tying exact three-repo commits together |

## Critical contract correction before status widening

Current infra keeps `restart_not_before` in `SharedRuntimeRestartLedger`, and `sharedRuntimeRestartDelay` uses it internally, but `SharedRuntimeStatus` does not serialize it. By contrast, `vendorplugin.Availability.Validate` requires every `AvailabilityLimited` verdict to carry a non-zero evidence-derived `Until`.

Therefore `TASK-260829-1kpj01` has two honest choices:

1. **Recommended:** first add additive `restart_not_before` (and preferably `half_open` plus last-failure metadata) to infra public status, then decode it and map active backoff to `AvailabilityLimited{Until: restart_not_before}`.
2. Narrow the component task to quarantine-only widening and report an in-progress retry as `Unknown`; do not claim Backoff is production-visible.

Inferring active backoff from `restart_count > 0` is unsafe: the count remains non-zero after readiness until the configured stable-run timer resets it, so a serving runtime could be mislabeled as backoff.

The same correction should update the stale `TASK-260829-1kpj01` dependency note: infra Handoff A is now landed, but the public deadline needed for truthful Backoff is not.

## Smallest independent landing sequence

The dependency-minimal sequence is:

| Wave | Landing | Dependency | Acceptance boundary |
| --- | --- | --- | --- |
| 1A | Finish, independently review, and merge infra `TASK-260829-3fozxa` | `891de44` only | Both `broker.log` and `runtime.log` have explicit byte/segment bounds, exact-boundary tests, deterministic pruning, and status-visible policy/footprint if operators need to diagnose it |
| 1B | Fix the system-only redeclaration authority blocker in agents `TASK-260829-188jcf`, accept M1, merge, and publish an immutable module release | Current agents main; can run parallel with 1A | One symmetric declaration authority; Pi/local-models/local-Qwen packages and generic `BuildLaunch` contract released |
| 1C | Add a small infra status-contract follow-up | `891de44`; can run parallel with 1A/1B | Publish `restart_not_before`; decide and pin `half_open`, last failure, and jitter semantics. Negative tests prove no serving runtime is mislabeled as backoff |
| 1D | Add bounded retention for auto-created Pi lifecycle logs | Current infra; can run parallel with 1A/1B/1C | Explicit byte/count/age policy, deterministic pruning, active-file preservation, status/diagnostic footprint |
| 2 | Rebuild board `TASK-260828-3hultd` on current board trunk against the immutable agents release, refresh `go.mod` and vendor, complete review, and merge | 1B | The sole production boundary is `vendorplugin.BuildLaunch`; local/remote config, selection, cancellation, recovery, and provenance are released with no unstable workspace dependency |
| 3A | Implement agents `TASK-260829-1kpj01`, then publish the next immutable component release | 1B + 1C | Pre- and post-extension fixtures; quarantine and active backoff are source-aware, validator-compliant, and reached through real `CheckAvailability` |
| 3B | Create and land a dedicated resource-pressure vertical slice | 1B; can overlap 2/3A | Infra exposes bounded, typed loaded/busy/pressure facts; component maps them without guessing; board preflight returns a typed side-effect-free refusal and draining/eviction policy is explicit |
| 4 | Add a dedicated board M2 consumer-widening task and upgrade to the release from 3A | 2 + 3A | Query/manifest/status surfaces preserve restart, quarantine, lease, backoff, and recovery provenance through retry/resume/reconnect |
| 5 | Run agents `TASK-260829-lx3o55` as the deterministic composed proof, and add a separate controlled real-soak evidence task | 1A, 1D, 2, 3A, 3B, 4 | Fake-clock hours/days/weeks matrices plus a bounded disposable-runtime soak with explicit SLOs, restart/pressure injection, aggregate artifact bounds, exact commit/release provenance, and no interaction with a user-owned model |

Waves 1A-1D are independent and should not wait on the component/consumer chain. The deterministic soak must remain last: running it before the status deadline, pressure plane, consumer projection, and retention policy exist would only prove a narrower system than the one M2 claims.

## Required soak evidence

For reviewer acceptance, the final proof should pin exact commits/releases and include:

- accelerated hour/day/week clock progression with no wall-clock sleeps;
- crash before readiness, crash after readiness but before stable reset, repeated crash to quarantine, quarantine expiry/half-open success and failure, manual quarantine/unquarantine;
- broker restart and host-restart fixtures with valid stale identity, reused PID, malformed ledger, missing ledger, stale lease mirror, abrupt wrapper death, heartbeat loss, and cancellation during backoff;
- drain while leased, new-acquisition refusal, final-release shutdown, and no duplicate lease/task ownership after retry/resume;
- exact log-segment boundaries, pruning order, lifecycle-log retention, and aggregate maximum bytes/files after simulated weeks;
- memory-pressure/inference-busy admission, recovery after pressure clears, and refusal when pressure cannot be observed;
- a landed task-board path proving initial spawn, retry, resume, reconnect, and recovery all use the same generic launch boundary and preserve public provenance;
- a bounded real soak on a disposable, explicitly owned test runtime, with SLOs for successful turns, recovery time, duplicate ownership, leaked processes/leases, peak and final disk footprint, and zero unbounded retry loops. A literal week-long CI job is unnecessary, but fake-only evidence cannot by itself demonstrate real long-running runtime behavior.

## Commands used

Representative read-only commands:

```text
git -C <repo> ls-remote origin refs/heads/main
git -C <repo> rev-parse refs/remotes/origin/main
git -C <repo> show -s --format='%H %cI %s' <exact-sha>
git -C <repo> show <exact-sha>:<path>
git -C <repo> grep -n <pattern> <exact-sha> -- <source paths>
task-board q --format compact 'get(<id>) { id name status parent children blockedBy outcomeResources }'
task-board grep <pattern> -i --file '*.md'
```

No build or test command was run: the requested audit was static/read-only, and rerunning broad green suites would not add evidence for the identified contract and backlog gaps.
