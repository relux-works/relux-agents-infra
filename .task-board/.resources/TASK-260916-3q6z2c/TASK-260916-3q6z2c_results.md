# TASK-260916-3q6z2c — Slice A implementation results

Ready for review. Deprecation of direct proprietary-provider launchers, removal of
migrated setup responsibilities, residual kept intact, README/SKILL.md/CHANGELOG updated.

Worktree HEAD: `459742ea67e3c6b84169520b92d74fe7f73e3002` (all changes uncommitted, as required).
This session continued from a prior partial implementation already present in the
worktree: I audited it against the accepted inventory (rev 2), completed the missing
SKILL.md migration, and ran the full verification below myself. Nothing below is
accepted from earlier runs.

## What changed (vs HEAD)

- `tools/agents-infra/main.go` — `runCodex`/`runClaude` replaced by exact one-line
  stderr errors via `deprecatedProviderError` (exit 1 through the existing main error
  path, no parsing/resolution/side effects); guards in `runTarget` (openai-infra,
  anthropic-infra) and `runDirectProviderYoloTarget` (dange surface names) before any
  parsing; `parseDirectProviderYoloTargetArgs` deleted; usage text documents the
  deprecated entrypoints and their removal next release. `runCompose`/`runPrepare`
  untouched.
- `tools/agents-infra/internal/infra/infra.go` — setup/refresh split: new
  `setupClaudeSettings`, `setupCodex` (config modes + rules only); deleted
  `ensureRepoSkillLinks`, `materializeRepoSkill`, stale/dangling skill-link repair,
  `shouldSkipSkillFanout`, `setupClaude`, `setupCodexWithConfig`, `setupCodexSurface`;
  `syncRepo` skips `.instructions/`, `.skills/`, `.configs/codex-mcp-servers.toml`
  (never deletes installed copies). Shim lifecycle, wrappers, doctor legacy fields kept.
- `tools/agents-infra/internal/infra/legacy_prepare_instructions.go` (new) — v1
  compatibility renderer moved here, called only by `PreparePrimarySession`
  (plus minimal scaffold creation for fresh residual runtimes).
- `tools/agents-infra/internal/infra/skill_link_validation.go` — deleted.
- `tools/agents-infra/internal/infra/source_dir.go` — usable-source contract drops
  instruction entrypoints, SKILL.md/README markers, and the include-closure walk.
- `tools/agents-infra/internal/infra/runtime_receipt.go` — skill topology removed
  from install verification.
- `tools/agents-infra/internal/infra/primary_session_prepare.go` — prepare-only
  surface refresh (`prepareCodexProjectSurface`/`prepareClaudeProjectSurface`).
- `.configs/codex-mcp-servers.toml` — deleted (composition still reads
  caller-supplied registries and fails closed on missing definitions).
- Tests — new `deprecation_main_test.go` (exact messages, every arg shape,
  compiled-binary + installed-wrapper probes with sentinel/snapshot no-side-effect
  checks); setup/prepare/compose/canonical-target/installed-binary tests rewritten
  for absence/non-mutation and deprecation; goldens retained.
- `README.md` — rewritten around the residual surface (deprecation notice with exact
  Curator commands, removal timeline, caller-supplied registries, compose/prepare
  compatibility exception, LLDB-absent statement).
- `SKILL.md` — migrated by this session: deprecation notice, Curator-owned
  instruction/skill distribution, compose-based diagnostics, no live-launcher or
  fan-out advice remains (all surviving `agents-infra codex|claude` mentions are
  deprecation notes).
- `CHANGELOG.md` (new) — Unreleased breaking entry, proposed v2.0.0. No version file
  exists to bump: version is injected at install time from `git describe --tags`
  (`scripts/setup.sh:91-96`, `scripts/setup.ps1:51-71`); tags end at v1.6.1 and the
  release owner publishes the tag.

## Verification (all run this session, quoted exit codes)

`go build ./...` → exit **0**. `go vet ./...` → exit **0**. `gofmt -l .` → clean.
Full-suite single calls were NOT used (host stalls on bulk fresh-binary exec; see
anomaly). Per-package groups via compiled test binaries, run from the package dir:

| Group | Command (tools/agents-infra) | Exit |
|---|---|---|
| infra A (compose/prepare/launch-plan/policy/shipped) | `-test.run 'TestBuildChildLaunchComposition\|…\|TestShippedCodexPolicy'` | **0** |
| infra B (setup/verify/source/doctor/wrappers/canonical) | `-test.run 'TestSetup\|…\|TestLegacyPrimarySession'` | **0** |
| infra C (lifecycle/KV/contract/planes) | `-test.run 'TestPiLifecycle\|…\|TestLaunchable'` | **0** |
| infra E/E2 (Pi launch/standalone/turn/run) | two `-test.run` masks | **0**/**0** |
| infra F (retirement/retention/catalog/remainder) | `-test.run 'TestValidateProjectTarget…\|…'` 18 tests | **0** |
| infra D1b (shared-runtime/broker, full) | 95-test `-test.run` mask | **0** (95/95) |
| main M1+M1x (deprecation) | 8 tests, 87 subtests | **0** |
| main M2 (compose/prepare/doctor CLI) | 40 tests | **0** (37 pass, 3 env-skips) |
| main M3-family (installed-binary/setup CLI) | 23 tests incl. solo rerun | **0** (21 pass, 2 env-skips) |
| main M4a (README/skill/model-check docs) | 13 tests | **0** (12 pass, 1 env-skip) |
| main M4b (Pi turn/runtime CLI) | 22 tests | **0** (18 pass, 4 env-skips) |
| attachments + modelharness | `go test -count=1 ./internal/attachments/ ./internal/modelharness/` | **0** (ok/ok) |
| cmd/model-harness | `go test -count=1 ./cmd/model-harness/` | **0** (no test files) |
| post-mutant revert confirm | guard tests + `TestBuildChildLaunchComposition*` | **0**/**0** |

Totals: main package 106/106 run → 96 pass, 10 env-skips, 0 fail. infra package
493/493 run → all green (D1b 95/95; every other test green in its group).
All 10 main skips are pre-existing environmental conditions, each with an explicit
reason in the log: official Pi asset missing from `.temp/TASK-260817-2h8hn4/…`
(9 tests) or pinned Pi production identity is darwin/arm64 while this host is
darwin/amd64 (1 test). No skip is related to this change.

## Mutation probes (2 narrowing mutants, both killed)

1. Deprecation guard narrowed: `runTarget` guard reduced to `openai-infra` only.
   Result: **killed** — `TestRunTargetDeprecatedCanonicalAliasesExitBeforeResolution`
   and `TestRunTargetDeprecatedAliasesIgnoreArgumentShape` fail (exit 1). Reverted.
2. Compose fail-closed narrowed: missing-definition refusal in
   `BuildChildLaunchComposition` replaced with silent `continue`.
   Result: **killed** — exactly `TestBuildChildLaunchCompositionRejectsEnabledServerWithoutDefinition`
   fails; the other 4 composition tests still pass, proving the mutant was narrow
   and the negative test is load-bearing. Reverted; post-revert reruns green.

## Deprecation coverage (measured)

Executable probes through the compiled binary and installed wrappers
(`TestDeprecatedEntrypointsExitOneWithNoSideEffects`): 6 direct argv cases × 6 arg
shapes (no args, `--print-config`, `--help`, danger flags, malformed provider
args, `spawn`) + 5 installed wrappers (openai-infra, anthropic-infra,
openai-dange, anthropic-dange, codex-local) × 2 shapes = **46/46 green**, each
asserting exit 1, exact one-line stderr, empty stdout, no provider/Curator
sentinel, and byte-identical filesystem snapshots. Plus in-process message/shape
tests (87 subtests total across 8 deprecation tests), all green.

## Dead code / contract checks

- `grep` over `tools/agents-infra/**/*.go`: no references remain to
  `ensureRepoSkillLinks`, `materializeRepoSkill`, `shouldSkipSkillFanout`,
  `removeDanglingManagedSkillLinks`, `setupCodexWithConfig`, `setupCodexSurface`,
  `validateSourceSkillLinks`, `managedSkillLinkFailures`,
  `missingInstructionIncludes`, `parseDirectProviderYoloTargetArgs`; the renderer
  (`writeClaudeEntrypoint`, `writeCodexEntrypoints`, include helpers) exists only
  in `legacy_prepare_instructions.go` plus one pointer comment. `repoSkillName`,
  marker consts, and `isRenderedInstructionsFile` all have live readers (doctor /
  prepare). `runtime_receipt.go` has zero skill references; `main.go` prints
  `infra_skill_link` as a bare legacy boolean with no health claim.
- `codex-mcp-servers.toml` references remain only in the retained registry-reader /
  compose path and its tests — the intended compatibility backend.
- Compose/prepare schema-1 goldens pass unchanged (infra A, main M2).

## Blind spots (stated bounds)

- Windows: the executable deprecation test skips on Windows by design; in-process
  guard tests cover dispatch on all platforms. No Windows runtime validation.
- No live provider launch, installer run, or Curator cross-check (out of scope;
  Curator evidence is the accepted B5 report, not rerun).
- `go test ./...` single-call never run (host limitation); equivalent coverage via
  the per-package groups above.

## Host anomaly (finding, not a code defect)

Fresh-binary first exec on this host intermittently stalls in `_dyld_start` for
1–6+ minutes (no `syspolicyd` observed; host uptime 215 days; affects unsigned and
ad-hoc-signed binaries alike; `sample` evidence captured). This caused: one
`go test -list` stall (recovered), the first full-infra timeout (broker children
stuck pre-main), one M3 group timeout (recovered; solo rerun passed in 10.46s),
and one D1 broker-group timeout. D1's 4 failures + 1 hang were proven transient:
the same 5 tests pass isolated in this tree (exit 0) and at pristine HEAD
(detached worktree, exit 0), and the full 95-test broker group passes clean
(D1b exit 0). No code was changed in response; broker/Pi sources are untouched by
this task. Stuck processes I started were killed; other sessions' long-lived
processes were left alone.

## Revision 2 — remote landing gate (rework 1, this session)

Revision 1's configured validation (`go test ./... -count=1` on this host)
failed: rev1 log shows the main package stuck 600s in
`TestDeprecatedEntrypointsExitOneWithNoSideEffects`, infra stuck 601s in a
child-spawn wait, and the two modelharness shutdown tests failing at ~20s
with "no pid was written". All three are this host's documented fresh-binary
/ process-spawn stall pathology, not code defects. Rev2 validation then
failed exit 127 on `sh scripts/remote-gate.sh` (file absent). This revision
adds the gate so validation runs on GitHub instead of this host. No Go
sources were changed in revision 2; the Slice A implementation from revision
1 stands as verified above.

Added (uncommitted, as required; both untracked-but-not-ignored, so the
gate snapshot picks them up):
- `scripts/remote-gate.sh` (new, +x) — POSIX snapshot/push/wait/cleanup
  landing gate, transcribed from the attached `remote-gate-curator.sh` with
  only the header comment adapted to this repo's hosted ubuntu+macOS matrix;
  `WORKFLOW` default stays `ci.yml`, `GODEBUG=netdns=go` kept, Story
  worktree naming (`STORY-*` parent dir) applies unchanged.
- `.github/workflows/ci.yml` (new) — triggers `push` on `main` + `gate/**`
  and `pull_request`; one `test` job on ubuntu-latest + macos-latest with
  checkout, setup-go@v5 (`go-version-file: tools/agents-infra/go.mod`),
  a gofmt check, then exactly
  `cd tools/agents-infra && go build ./... && go vet ./... && go test ./...
  -count=1 -timeout 20m`. No rose-air lane, no platform-case gate.

Cheap local verification (this session, quoted exit codes; whole suite NOT
run here per the rework brief):
- `cd tools/agents-infra && go build ./...` → exit **0**.
- `go vet ./...` → exit **0**. `gofmt -l .` → clean (no output).
- `sh -n scripts/remote-gate.sh` → exit **0**; full read-back of the script
  confirms a faithful transcription (only the header differs from source).
- CI YAML parses (Ruby YAML: jobs `test`, push branches main + gate/**,
  pull_request) — the `on:` key reads as YAML-1.1 boolean true, which is
  the normal form GitHub Actions expects.
- `go test -count=1 -timeout 5m -run
  'TestDeprecatedProviderErrorMessagesAreExact|TestRunCodexAndClaudeAreDeprecatedForEveryArgumentShape' .`
  → exit **0** (ok 0.639s).
- `go test -count=1 -timeout 5m -run 'TestBuildChildLaunchComposition'
  ./internal/infra/` → exit **0** (ok 0.542s).
- `TestRunSignalledShutdownWaitsForADetachedGroupMember` isolated → exit
  **0** (PASS 0.48s).
- `TestModelHarnessRunReleasesPortOnDirectedSIGTERM` isolated, attempt 1 →
  **FAIL** (killed after 6m0s, host stall, no test-binary output); attempt 2
  → exit **0** (PASS 1.34s; package wall 81.8s, i.e. ~80s of fresh-binary
  load stall before/after the test). No stray fixture processes left
  (`ps` clean); `nc` present at /usr/bin/nc. The attempt-1 kill and the
  80s load stall are the same dyld-assessment pathology the revision-1
  anomaly section documents (prior session: full modelharness package
  ok/ok) — not a product defect, so no test or product change was made.
- Gate prerequisites for the post-handoff run: `gh auth status` → logged
  in (reluxbot, ssh protocol); `SSH_AUTH_SOCK` set with one ED25519 key
  loaded. `git check-ignore` confirms neither new file is ignored.

SIGTERM/port tests on CI (brief item 3): neither test has failed on CI —
no CI run exists yet (`.github/workflows/` is new in this revision), and
the remote gate runs after handoff, so there is no CI verdict to react to
in this turn. Both tests pass in isolation here (0.48s / 1.34s) and the
full modelharness package was green in the prior session, so no fix or
bound is applied: lengthening the 20s pid/listen windows or softening the
no-poll `assertProcessGoneNow` would weaken load-bearing shutdown
assertions to accommodate a sick operator host, while ephemeral GitHub
runners do not exhibit the dyld-assessment stall. If either test fails on
CI, that failure is real signal (a healthy runner spawns /bin/sh in ms)
and must be investigated as a product/shutdown defect, not bounded away.

## Revision 4 — hosted-CI environment-bound tests (rework 2, this session)

Revision 3 was the first green-gate attempt on hosted CI (run 35061433557,
log in `TASK-260916-3q6z2c_change-request_rev3-validation.log`): both lanes
built, vetted, then failed in pre-existing tests that had never run off the
operator host. No production source is changed in revision 4 — six test
files only — and no deprecation/residual test is weakened (guard + golden
probes re-run green below).

Rev3 failure inventory (all quoted from the log):

- ubuntu main: `TestRunPiLifecycleOperatorIsNonLaunchingAndProjectsExactPlan`
  (:49), `TestRunPiLifecycleStatusRefusesForeignEvidence` (:131),
  `TestRunPiLifecycleStatusPaginatesWithoutLaunching` (:203) — each
  `canonicalize cache root: lstat /tmp/.../002/.cache: no such file or
  directory`.
- both lanes infra: `TestConsumerRecordsCancellationAfterChildStartAndKeepsItAuthoritative`
  (:127) — `result-invalid`, want cancelled.
- ubuntu infra: `TestPiLifecycleDeleteRecoveryPreservesSubstitutedChildAuthority/log-inode-substituted`
  (:253) — `changed log.jsonl authority admitted: next=true err=<nil>`.
- both lanes infra: `TestShippedCodexPolicyAdmitsAstraWithMediumCeilingAndSolFallback`
  (:38) — `exec: "task-board": executable file not found in $PATH`.
- macos modelharness: `TestStressRunsBoundedPrefillAndStopsRuntime` (:60) —
  readiness timeout after 5008 ms, child alive (102 RSS samples, 25 MB peak),
  empty stderr.

Diagnoses and fixes (one per failure; assertions unchanged throughout):

1. Lifecycle CLI cache root (ubuntu): the fixtures seed
   `$HOME/Library/Caches` and the CLI resolves `os.UserCacheDir()`, which is
   `$HOME/Library/Caches` on macOS but `$XDG_CACHE_HOME` else `$HOME/.cache`
   on Linux (verified in GOROOT `src/os/file.go`). On Linux the CLI resolved
   the never-created `.cache` and `EvalSymlinks` refused it. Fix:
   `pi_lifecycle_main_test.go` sets `t.Setenv("XDG_CACHE_HOME", cache)` in all
   three tests — honored on Linux, ignored on macOS. These are the only
   `runPi(["lifecycle", …])` callers in the main package, so the scope is
   exact.
2. Consumer cancellation (both lanes): the fake Process A body was the bare
   `sleep 30`, but the child env carries only the fixture `PATH=<tmpdir>`
   (verified with a temporary probe test: plan env is `PATH=<tmpdir>` plus
   run-context vars; probe deleted after use). On any host fast enough to run
   the child within the 250 ms cancel delay, `sh` exits 127 with no document
   and the classifier reports result-invalid. It passed on the operator host
   only because a trivial spawn takes ~340 ms there (measured), so cancel
   always won the race. Fix: new `blockingProcessABody` helper in
   `agents_management_consumer_test.go` embeds `exec.LookPath("sleep")`
   (ambient PATH) as an absolute path; same fix applied to the identical
   latent `sleep 30` fixture in
   `pi_shared_engine_observation_darwin_test.go:188`, which previously could
   not exercise cancellation on a fast host (it asserts only `err != nil`, so
   it never failed — now it actually drives a cancelled turn).
3. Inode substitution (ubuntu): production records and compares exactly
   (device, inode, mode, uid, links) (`piLifecycleDeleteIdentityMatchesStat`,
   `pi_session_log.go:815`). The fixture's unlink+recreate frees the inode,
   which Linux immediately recycles, so the "replacement" is
   identity-identical and production correctly admits it. Fix: hold the
   original open across the substitution (a live fd pins the inode) and assert
   the before/after (dev, ino) differ, mirroring the tombstone-substitution
   test's own distinctness check. The sibling subtests are deterministic
   already (mode change; symlink O_NOFOLLOW refusal), as is the
   rename-away-and-O_EXCL-create hook test.
4. Shipped policy preflight (both lanes): prescribed bound — skip with reason
   `task-board binary unavailable on PATH` when `exec.LookPath` fails; the
   stale "missing tooling is a failure" comment is updated, and the
   native-fallback half still runs as a top-level test everywhere.
5. Stress readiness (macos, single occurrence): passed locally (0.66 s) and on
   ubuntu CI (package ok 12.5 s); unreproduced. Evidence (living child, empty
   stderr, 5 s of unanswered readiness) is consistent with loopback/port
   transience under the parallel-package CI load, and the bind-`:0` → close →
   rebind-in-child pattern is racy by construction regardless of the exact
   cause. Fix: bounded retry (3 attempts, fresh port + fresh runtime each;
   `Stress` always stops its child before returning), every assertion still
   against one real pass, each failed attempt logged. A systematic Stress
   breakage fails all attempts identically, so no gate property is lost. The
   cause is NOT proven — CI is the verifier; see blind spots.

SIGTERM/port tests (brief item 3): passed on ubuntu (modelharness `ok`
12.530 s includes them; they are not short-gated) and on macos (the lane's
only modelharness failure is the stress test above). No change, as predicted
in revision 2.

Local verification (this session, quoted exit codes; whole suite NOT run
here per the brief):

- `cd tools/agents-infra && go build ./...` → exit **0**.
- `go vet ./...` → exit **0** (after fixing one self-introduced `err`
  redeclaration in the retry loop, caught by vet before any test ran).
- `gofmt -l .` → clean (no output).
- `go test -count=3 -run
  'TestConsumerRecordsCancellationAfterChildStartAndKeepsItAuthoritative'
  ./internal/infra/` → exit **0** (3/3).
- `go test -count=1 -run 'TestPiLifecycleDeleteRecoveryPreservesSubstitutedChildAuthority'
  ./internal/infra/` → exit **0** (all 3 subtests).
- `go test -count=1 -run 'TestShippedCodexPolicy…' ./internal/infra/` →
  exit **0** (PASS, task-board on PATH here); compiled test binary re-run
  with a task-board-less PATH → SKIP with the exact reason, exit **0**.
- `go test -count=1 -run 'TestSharedRuntimeEngineObservationReader…'`
  (darwin) → exit **0**.
- `go test -count=1 -run 'TestRunPiLifecycleOperator…|TestRunPiLifecycleStatusRefuses…|TestRunPiLifecycleStatusPaginates…'
  .` → exit **0** (3/3; XDG branch is Linux-only, darwin behavior unchanged
  by construction).
- `go test -count=2 -run 'TestStressRunsBoundedPrefillAndStopsRuntime'
  ./internal/modelharness/` → exit **0** (2/2, first-attempt passes).
- Consumer siblings (`TestConsumer*`, dry-run, qwen-resolve, metamorphic) →
  exit **0**; recovery siblings (`TestPiLifecycleDeleteRecovery*`,
  `TestPiLifecycleRecovery*`, `TestPiLifecycleMalformed*`) → exit **0**.
- Slice A intact: deprecation message/shape tests → exit **0**; compose
  golden `TestBuildChildLaunchComposition*` → exit **0**. No production file
  touched in revision 4 (`git status`: only the six `_test.go` files are new
  vs the rev1–3 set).

Blind spots (stated bounds): the stress-test root cause is diagnosed as
environmental-transient but not proven — if it fails again on CI with the
same alive-child signature, the retry logs (attempt, port, report) will show
whether fresh ports also hang, which separates a port-squatter from a wedged
runner. The XDG and inode fixes are Linux-targeted and verified locally only
for no-regression plus stdlib-source/fixture-honesty argument; the ubuntu CI
lane is their real verifier. No Windows run (unchanged from prior
revisions).

## Revision 5 — hermetic stress fixture (rework 3, this session)

Rev4 gate (run 35063033520): ubuntu green, macos red on
`TestStressRunsBoundedPrefillAndStopsRuntime` only — all 3 fresh-port attempts
failed with the identical alive-child/unanswered-readiness signature. Per the
rev4 blind-spot protocol, fresh ports hanging identically rules OUT a
port-squatter; the cause had to be fixture-vs-host. No production source is
changed in revision 5 — two test files only — and no assertion is weakened.

Diagnosis (proven, not inferred): a throwaway diagnostic test was pushed to a
`gate/**` branch (run 35063667613, branch deleted after) that logged proxy
state, a Go-httptest loopback control, and staged raw dials around a faithful
replica of the python fixture. Macos results, quoted from the CI log:

- Proxy: all six `*_PROXY`/`NO_PROXY` vars absent, `ProxyFunc` returns DIRECT
  for the loopback URL. The proxy-reroute hypothesis is dead.
- Go httptest control in the same process: GET 200 in 1 ms. Loopback and the
  Go HTTP client are fine.
- Python child (`/opt/homebrew/bin/python3`, 3.14.7): pre-spawn dial refused
  instantly; poll 1 refused; polls 2–8 all `dial tcp …: i/o timeout` (2.0 s
  each, no RST, no SYN-ACK); child reaped via SIGKILL with empty stderr;
  post-kill dial refused instantly again.

So once the Homebrew-python child binds 127.0.0.1, inbound SYNs to that port
are blackholed for the child's whole lifetime — a per-binary loopback filter
on the runner image (suspected macOS app-firewall/per-binary gate; exact
mechanism unproven and not needed). Ubuntu in the same run: the replica
fixture passed (python up by poll 2), confirming the failure is macos-host
specific.

Fix: the stress fixture no longer spawns python. New
`internal/modelharness/stress_helper_process_test.go` adds a `TestMain` that
serves the fixture API (`GET /v1/models`, `POST /v1/chat/completions` with
the exact `repeats+11` token model, 404 elsewhere, loopback-only bind) when
invoked as `stress-helper-child <host> <port>`, and `stress_test.go` spawns
the test binary itself (`os.Executable()`). Every gate property of the
witness is preserved — real separate child process, real readiness polling,
real `ps`-based RSS sampling, real stop, real port-release check — and the
listener is the same Go binary the diagnostic proved reachable on that exact
host class (1 ms/200). The 3-attempt retry is kept for the bind-`:0`/rebind
race only. The `python3` dependency and its skip are gone; the test is now
hermetic. No production change: `Stress` reporting a timeout for a genuinely
unreachable child endpoint is correct behavior, not a product defect.

Local verification (this session, quoted exit codes; whole suite NOT run
here per the brief):

- `cd tools/agents-infra && go build ./...` → exit **0**.
- `go vet ./...` (whole module) → exit **0**. `gofmt -l .` → clean.
- `go test -count=2 -run 'TestStressRunsBoundedPrefillAndStopsRuntime'
  ./internal/modelharness/` → exit **0** (2/2, 152 s wall — host fresh-binary
  stall, as documented in rev1).
- `go test -count=1 ./internal/modelharness/` (full package — warranted
  because `TestMain` touches every test in the package) → exit **0**
  (ok 14.1 s).
- Helper contract probed directly via a compiled test binary + foreground
  probe script: `GET /v1/models` → 200 exact body; POST with 3 repeats →
  `prompt_tokens` 14 (= 3+11, token model intact); bogus path → 404; silent
  stderr; SIGKILL-reaped; post-kill dial refused; bad argv → exit 2 with
  usage. Probe exit **0**.
- Slice A intact: deprecation message/shape tests → exit **0**; compose
  golden `TestBuildChildLaunchComposition*` → exit **0**; calibration +
  synthetic-completion unit tests → exit **0**.
- `git status`: only the rev4 set plus the rewritten `stress_test.go` and
  new `stress_helper_process_test.go`; the diagnostic test file and its gate
  branch are both deleted.

Blind spots (stated bounds): the runner-side filter mechanism is diagnosed
(blackholed SYNs, python-specific, Go unaffected) but its exact component is
unproven — no further gate cycle was spent on it because the fix removes the
dependency instead of accommodating the filter. The macos CI lane remains the
verifier for the rewritten fixture. No Windows run (unchanged).

## Revision 6 — installed wrappers refuse before any build (rework 4, this session)

Reviewer P1 (RUN-260916-c350c0, CHANGES REQUESTED): the installed deprecated
wrappers delegated to the local agents-infra launcher, which runs `mkdir` +
`go build` before the Go dispatcher can refuse. Reproduced on this tree
BEFORE the fix (scratch probe `/tmp/b6-rev6-prefail.sh`, kept out of the
repo): with the cached build output deleted and a failing `go` (exit 73)
shadowing the toolchain, the installed openai-infra wrapper exited **73**
with `unexpected-build` and recreated `.agents-infra-build` — the required
exit-1 migration notice never printed. The old test hid this with
`.agents-infra-build` and `Library` snapshot exclusions.

Fix — the deprecation is now decided by the wrapper itself, first, with no
build step and no filesystem writes (deprecated entrypoints only; live
wrappers unchanged):

- `internal/infra/deprecation.go` (new): single-source exact messages via
  `DeprecatedProviderMessage`, POSIX single-quote and cmd caret escaping
  (`<args>` would otherwise parse as a redirection), complete refusal
  bodies, and cliWrapper guards for both shells.
  `main.deprecatedProviderError` now draws from it, so the compiled binary
  and the generated wrappers cannot drift apart.
- `canonicalTargetWrapperBody`: openai-infra/anthropic-infra return
  direct-refusal stubs (no sibling lookup, build, or exec) on POSIX and
  Windows; qwen-infra still delegates.
- `directProviderYoloWrapperBody`: both dange aliases return
  direct-refusal stubs (delegation branch retained only for a future live
  alias).
- `codexLocalLauncherBody`: keeps its generated marker (the MCP-opt-out
  lifecycle keys on it) but refuses directly with the CLI codex message.
- `cliWrapperBody`: a guard for `codex|claude`, `target
  openai-infra|anthropic-infra`, and `target-yolo
  openai-infra|anthropic-infra` runs before any mkdir/build/exec, mirroring
  the Go dispatcher exactly (bare `target` and live subcommands fall
  through to the normal build path).

Durable tests (kept with the project, no exclusions left):

- `deprecation_main_test.go`: `snapshotTree` hashes everything with no
  exclusions; the fixture deletes the cached `.agents-infra-build` output
  after setup and shadows `go` with an exit-73 `unexpected-build` stub
  during every probe; 12 new cliWrapper probes added. Now **58/58**
  subtests (36 compiled-binary × 6 arg shapes, 5 alias wrappers +
  codex-local × 2 shapes, 6 cliWrapper argv × 2 shapes), each asserting
  exit 1, exact one-line stderr, empty stdout, no provider/Curator
  sentinel, and byte-identical project+home trees.
- `installed_binary_setup_test.go` alias test: same failing-go/absent-build
  fixture (**16/16**).
- `infra_test.go`: new `TestDeprecatedCanonicalWrappersRefuseWithoutDelegation`,
  `TestLiveCanonicalWrapperStillDelegates`,
  `TestDeprecatedDirectProviderYoloWrappersRefuseWithoutDelegation`,
  `TestCodexLocalLauncherRefusesWithoutDelegation`, plus guard-order
  assertions in both cliWrapper tests (guard precedes mkdir/`go build`;
  `qwen-infra` never intercepted).
- `canonical_alias_posix_test.go`: live qwen delegation preserved
  (cwd/argv); deprecated canonical/dange aliases refuse identically with
  the sibling present or missing.
- Codex-local setup lifecycle tests assert refusal content, not delegation.

Docs: README (alias paragraph, dange table row — which still described
launching — shim paragraph), SKILL.md (alias + shim paragraphs), and
CHANGELOG (build/delegation-free guarantee) updated. No test asserts the
old wording (grep-verified).

Local verification (this session, quoted exit codes; whole suite NOT run
here per the brief — the remote gate runs it on GitHub after handoff):

- `gofmt -l .` → clean; `go build ./...` → exit **0**;
  `go vet ./...` → exit **0**.
- In-process deprecation guards
  (`TestDeprecatedProviderErrorMessagesAreExact|TestRunCodexAndClaude…|TestRunTargetDeprecated|TestRunDirectProviderYoloTarget`)
  → exit **0**.
- Compose golden `TestBuildChildLaunchComposition` → exit **0**; lifecycle
  CLI masks → exit **0**.
- Infra wrapper/alias/codex-local masks (string + POSIX executable +
  lifecycle + receipt alias-drift) → exit **0** throughout.
- `TestDeprecatedEntrypointsExitOneWithNoSideEffects` (**58/58**) →
  exit **0**.
- `TestInstalledProviderAliasesReportDeprecationWithoutLaunching`
  (**16/16**) → exit **0**.
- README/SKILL docs group (10 tests) → exit **0**.
- Post-fix manual probes (failing `go`, absent build dir): all 5 alias
  wrappers + codex-local + 6 cliWrapper argv shapes → exit 1, exact
  messages, empty stdout, no build-dir recreation. Narrowness: live
  `version` with failing `go` → 73 (build attempted, guard did not fire);
  with real `go` → 0 + version string; bare `target` with failing `go` →
  73 (not a deprecated shape).

Mutants (both narrowing, both killed, bytes restored and re-verified):

1. Canonical guard narrowed to openai-infra only → string test FAILs
   (exit **1**, anthropic body delegates) and executable
   `…/anthropic-infra` FAILs (exit **1**, delegated exit 0, empty
   stderr); openai sibling still passes. Restored → green (exit **0**).
2. POSIX cliWrapper `target` anthropic branch renamed to
   `anthropic-infra-disabled` → both `cliWrapper/target_anthropic-infra`
   shapes FAIL (exit **1**, exit 73 `unexpected-build` — the exact P1
   signature); `cliWrapper/target_openai-infra` passes under the same
   mutant (exit **0**). Restored → all 12 cliWrapper green (exit **0**).

`git status` vs rev5: new `internal/infra/deprecation.go`; modified
`infra.go`, `main.go`, `deprecation_main_test.go`,
`canonical_alias_posix_test.go`, `infra_test.go`,
`installed_binary_setup_test.go`, `README.md`, `SKILL.md`,
`CHANGELOG.md`. Nothing else touched; compose/prepare goldens unchanged.

Blind spots (stated bounds): Windows refusal bodies are string-tested
only — no Windows runtime on this host; the ubuntu+macos CI gate remains
the full-suite verifier. The 217 s wall on one post-restore rerun is the
documented host fresh-binary stall, not test time (test itself <1 s).
