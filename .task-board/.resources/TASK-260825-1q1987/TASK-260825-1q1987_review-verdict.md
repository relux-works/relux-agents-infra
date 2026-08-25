# Review Verdict — TASK-260825-1q1987

- Verdict: **accepted**
- Change Request: `CR-TASK-260825-1q1987-1` revision `1`
- Reviewer run: `RUN-260825-58ee90` (claude-opus-5)
- Reviewed tree: `1d2845fb6d9c0a186491a523947fc5b3a50a1a41` (base `e70f953`)
- Reviewed at: 2026-08-26

## Scope of the delta

`git diff --stat e70f953 1d2845f` = 2 files, 649 insertions, 0 deletions:
`.research/260825_pi-unattended-tool-authorization.md` (new, 637 lines) and
`LOGBOOK.md` (+12). No code, config, test, or build file changed. The reviewer
worktree's `HEAD^{tree}` equals the candidate tree OID and `git status --short`
is empty, so every artifact inspected below is the exact reviewed candidate and
not a later mutation.

Because the delta is documentation only, no project suite is implicated by the
change itself. A regression check was nevertheless run in this worktree to make
the claim a measurement rather than an inference:
`go build ./...` in `tools/agents-infra` exited `0`, and
`go test ./internal/infra/...` reported `ok ... 84.0s`. Nothing in the delta
could have affected either, and both are reported as observed results, not as
proof that the research is correct.

## What was attacked, not read

This is a research deliverable, so the reviewable gate is the claim set. Every
load-bearing claim was independently re-derived from the pinned upstream tree at
`914cf1472e715297caa30db4b9535d534a9eb718` (verified `git log -1` = tag
`v0.84.2`) and the current snapshot `8fa7eebd235355522c8104166b4f1f959b4e2f10`
(verified `packages/coding-agent/package.json` = `0.84.3`), plus the retained
probe logs. Independent re-verification, not acceptance of the report:

| Claim (doc section) | Independent check | Result |
| --- | --- | --- |
| No default approval decision; absent hook means execute (§3.1) | Read `agent-loop.ts:600-668`; `prepareToolCall` calls `config.beforeToolCall` only `if (config.beforeToolCall)`, block short-circuits to `createErrorToolResult` before `executePreparedToolCall` | Confirmed |
| Sequential/parallel preflight at `agent-loop.ts:452` / `:507` | Read both call sites; both `await prepareToolCall(...)` before execution | Confirmed |
| `tool_execution_start` precedes the hook and is not authorization (§3.2) | Emitted immediately before `prepareToolCall` in both lanes | Confirmed |
| `--approve` is project trust only, never a tool field (§4) | `args.ts:205-208` sets `projectTrustOverride` only; `grep projectTrustOverride args.ts` returns exactly 3 hits, none tool-related | Confirmed |
| `--tools` is strict across built-in **and** extension/custom tools (§5.1) | `agent-session.ts:_refreshToolRegistry` applies `isAllowedTool` to `allCustomTools` *and* `_baseToolDefinitions`; custom tools then `set()` over the same name | Confirmed |
| `--no-extensions` drops global **and** project discovery, keeps explicit `-e` (§5.2) | `resource-loader.ts:451` and `:555`: `noExtensions ? cliEnabledExtensions : merge(cli, enabled)` — all resolved package extensions dropped regardless of scope | Confirmed |
| Skill `allowed-tools` has no consumer (§5.3) | Repo-wide grep over `packages/**` for `allowed-tools`/`allowedTools`, excluding `node_modules`: exactly one hit, `docs/skills.md:148` | Confirmed |
| RPC has no tool-authorization or tool-mutating command (§5.4) | Enumerated the whole `RpcCommand` union and every `case` in `rpc-mode.ts`; 31 commands, none touches the registry or allowlist | Confirmed |
| Direct RPC `bash` bypasses `tool_call` (§5.4) | `rpc-mode.ts:559-580` calls `emitUserBash` then `session.executeBash`; no `beforeToolCall` on this path | Confirmed |
| `user_bash` interception is fail-**open** (§5.4) | `runner.ts:955-980`: a throwing handler is caught, reported via `emitError`, and the loop continues to `undefined` → rpc-mode executes anyway. `UserBashEventResult` (`types.ts:1083-1088`) has only `operations` and `result`; no `block` | Confirmed |
| Agent-core hook files byte-identical pinned → current (§6) | `diff` of `agent.ts`, `agent-loop.ts`, `types.ts` across both checkouts | Identical |
| No new native approval flag in current (§6) | Diffed the tool/extension/trust option surface of `args.ts` across snapshots: line-number shift only, identical option set; `--approve`/`-a` still trust-only at `:219` | Confirmed |
| `clear_queue` added; `powershell` tool added (§6) | `rpc-types.ts:26` in current; `core/tools/powershell.ts` exists only in current | Confirmed |
| agents-infra anchors (§4, §9.2, §10) | `pi_config.go:583` = `validatePiPrimarySessionYolo`; `pi_args.go:51-68` = the known-option tables; `pi_args.go:180-228` = the pass-through composition the doc says must be reserved | Confirmed |

### Local probe evidence re-checked, not taken on trust

The probe logs under `.temp/TASK-260825-1q1987/` exist and their recorded argv
and stdout match every row of the doc's §7 matrix. The decisive negative was
re-read end to end: `rpc-probe/policy-probe.ts` registers
`pi.on("tool_call", () => ({ block: true, terminate: true }))`, and
`rpc-direct-bash.log` nevertheless shows
`{"command":"bash","success":true,"data":{"output":"rpc-direct-bash-ok","exitCode":0}}`.
The bypass is real and reproduces from the retained artifacts, not merely
asserted. `approve-override.log` likewise shows `bash` with `"source":"auto"`,
proving the project-extension replacement of an allowed built-in name.

### Negative-shape audit of the deliverable itself

- **Failed read reported as absence:** not present. The `allowed-tools` search is
  reported as "found only in documentation, no consumer" and independently
  reproduces; it is not inflated into "no such feature exists".
- **Failure laundered into a pass:** not present. The coding-agent test's initial
  exit `1` is retained in §7 as a genuine setup failure with the cause (missing
  generated provider JSON), and the exit-`0` rerun is reported separately. Both
  logs exist and match.
- **Positive-path-only evidence:** not present. The doc's own §11 test plan is
  built on refusals and includes two *narrowing* mutants (items 14 and 15) rather
  than delete-only mutants, which is the standard this project requires.
- **Capability claim that does not reproduce:** actively guarded. §6.1 refuses to
  claim end-to-end tracked Pi spawn, names `spawn.runtimes` as vocabulary rather
  than behavior, and cites the task-board sources showing built-in `qwen` binds
  to the `qwen-code` harness. §11.2 item 17 turns that into a required negative.
- **Unknown reported as unknown:** §9.6 explicitly forbids claiming absence of
  passwordless elevation unless OS policy proves it, and §9.5 refuses to call the
  contract a sandbox.

## Assessment against acceptance criteria

- *Names the exact authorization call path* — §3, re-derived above, line-accurate.
- *Distinguishes project trust from tool execution permission* — §4, proven both
  in source (`projectTrustOverride`) and by the four-row probe table.
- *Tests candidate mechanisms against the pinned binary/source* — §7; nine
  no-model probes plus two upstream regression tests, all logs retained.
- *Compares security and maintenance costs* — §8, six mechanisms; §8.1 adds the
  privilege axis with sudoers and Apple Service Management primary sources.
- *Recommends one concrete implementation with defaults, opt-in scope,
  diagnostics, and negative tests* — §9 (fail-closed defaults, reserved argv,
  diagnostic shape), §10 (five patch points), §11 (20 negatives).

The AC is met in full. The recommendation is also correctly *smaller* than the
available mechanisms: it uses native launch-time controls and explicitly declines
a Pi fork or a policy extension until argument/path policy is actually required.

## Advisory notes for the implementation task (non-blocking)

1. §9.1 item 3 requires unknown allowlist names to fail before spawn but does not
   name the source of truth for the known Pi tool-name set. Upstream adds tools
   (`powershell` landed in `0.84.3`), so the implementing task must pin that
   catalog to the Pi identity it already revalidates. The failure direction is
   safe — a stale catalog rejects a legitimate new name rather than admitting an
   unknown one — which is why this is advisory.
2. The §5.1 caveat that an allowed *name* can resolve to an extension
   *replacement* is derived from `_refreshToolRegistry`, not from a probe that
   combined `--tools bash` with a project `bash` replacement. The code path is
   unambiguous and §11.2 item 8 already schedules the missing negative, but the
   distinction between code-derived and probe-derived is worth carrying forward.

Neither note changes the recommendation, and neither is grounds for rework.

## Verdict

Accepted. Every load-bearing claim reproduces from primary source or retained
artifacts, the negative findings are stated as findings rather than smoothed
over, and the deliverable stops short of claims it cannot support.
