# BUG-260817-161m6u — Reviewer Verdict: CHANGES REQUESTED (-> to-dev)

Reviewer run `RUN-260817-f7ff21` (not goal-bound). Source change is correct and
its gate is genuinely bound by tests; the delivery gap is the installed runtime
and one ungated documentation claim.

## What holds up under attack

`ValidatePiExecutionEnvironment` (`tools/agents-infra/internal/infra/pi_catalog.go:284`)
denies the case-insensitive `LLAMA_ARG_*` namespace and names only the variable.
Production call site `RunPi` (`tools/agents-infra/internal/infra/pi_launch_posix.go:94`)
validates before `CreatePiStateTree`/`runtimeCmd.Start()` (llama.cpp spawn, line 148)
and re-validates at line 197 before Pi spawn. Windows `RunPi`
(`pi_platform_windows.go:73`) refuses managed profiles outright, so its
`ValidatePiExecutionEnvironment` no-op stub is not a bypass path. Unmanaged
passthrough (`selected == ""`) spawns no llama.cpp.

Independent mutants, run against a disposable clone of the tool tree with the
gitignored Pi asset cloned in so production-entry negatives actually execute
(`.temp/BUG-260817-161m6u/mutants/`):

| Mutant | Result | Reddened |
| --- | --- | --- |
| Baseline (unmodified source) | green | production negatives RUN, not SKIP |
| Delete `"LLAMA_ARG_"` from prefix list | FAIL | validator MODEL+CTX_SIZE, production `llama model`/`llama absent option` reached runtime spawn (`runtime exited before readiness`) |
| Narrow to `"LLAMA_ARG_MODEL"` | FAIL | CTX_SIZE admitted at validator and production entry |
| Narrow to `"LLAMA_ARG_C"` | FAIL | MODEL admitted at validator and production entry |
| Drop pre-spawn call, keep only post-readiness call | FAIL | all four production shapes reached runtime spawn — ordering is enforced, not incidental |
| Delete `"DYLD_"` | FAIL | existing loader gate still covered |
| Disable inbound `PI_*` name check | FAIL | `inbound agent dir` — existing managed-name gate still covered |
| Over-broad: also deny `"TERM"` | FAIL | clean-environment control has teeth (not vacuous) |
| Restore original | green | — |

Independent full-repo verification: `go test ./... -count=1` in
`tools/agents-infra` exit 0 (main 79.6s, attachments 1.9s, infra 130.4s);
`go vet ./...` clean; `gofmt -l` empty.

## Finding 1 (blocking) — installed global launcher does not carry the gate

`~/.local/bin/pi-infra` is a shim that `exec`s `~/.local/bin/agents-infra`.
That binary is Mach-O, mtime 2026-08-17 21:11 — built before this fix.

| Probe | Fresh `go build -trimpath` of current source | Installed `~/.local/bin/agents-infra` |
| --- | ---: | ---: |
| `LLAMA_ARG_` | 1 | 0 |
| `runtime-affecting` | 1 | 0 |
| `DYLD_` | 1 | 1 |

It does contain `pi_execution_environment_invalid`, `duplicate environment name`,
and `PI_CODING_AGENT_DIR`, so it carries earlier story Pi work — only this gate
is missing. Running `pi-infra` on this machine today still admits
`LLAMA_ARG_MODEL`.

Cause is visible in the attached evidence: `setup-global-01.log` ends with
`Skipping local CLI wrapper install for global setup; bootstrap owns
~/.local/bin/agents-infra`. `agents-infra setup global` syncs source into
`~/.agents` (confirmed: `~/.agents/tools/agents-infra/internal/infra/pi_catalog.go`
contains the prefix) but never rebuilds the bootstrap-owned binary. The path that
rebuilds it is the repo bootstrap `./setup.sh` -> `scripts/setup.sh:182`
(`go -C "$SOURCE_DIR/tools/agents-infra" build -trimpath ... -o "$BUILD_OUTPUT"`).

Evidence shape: the outcome artifact rows `Global setup refresh from source | 0`
and `Installed global verification | 0` are presented as installed-runtime
coverage, but `verify global` inspects the runtime tree, not the executable —
a property inferred from a proxy signal. Task scope explicitly includes "refresh
installed runtime through setup".

Rework:
- run the repo bootstrap `./setup.sh` so `~/.local/bin/agents-infra` is rebuilt,
  and attach evidence that the *installed* launcher enforces the gate — preferred
  behavioral proof: installed `pi-infra` against a managed profile with
  `LLAMA_ARG_MODEL` set returns `pi_execution_environment_invalid` and creates no
  managed state; string parity is the weaker fallback;
- correct the artifact rows so the installed-runtime claim states exactly what was
  refreshed (`~/.agents` source tree) versus what the operator actually executes.

The local-install lane is fine as-is: the generated local wrapper is a shell
script that rebuilds from `AGENTS_INFRA_SOURCE_DIR=/Users/alexis/.agents` on every
invocation, so it picks the fix up.

## Finding 2 (blocking, small) — documented deny contract is not gated

`tools/agents-infra/pi_operator_docs_test.go` pins every other Pi operator
contract element to an exact README fragment, but contains no `LLAMA_ARG_`
fragment. Deleting the new README sentence ("Duplicate or runtime-affecting
`DYLD_*`, `LD_*`, `NODE_*`, `BUN_*`, or `LLAMA_ARG_*` environment names are
refused before llama.cpp starts...", README.md:600-602) leaves the suite green,
so the documented contract can regress silently. Add the fragment to
`TestPiOperatorContractDocumentsCycle10Boundary`.

## Non-blocking notes

- `TestPiLaunchRejectsLoaderAndInboundPiEnvironmentBeforeState` reaches its
  production negatives only when the gitignored
  `.temp/TASK-260817-2h8hn4/pi-standalone-darwin-arm64-0.84.2` asset exists;
  otherwise `officialPiAsset` calls `t.Skipf` and the package still reports `ok`.
  Story-wide fixture convention already logged (LOGBOOK 2026-08-17 2132), not
  introduced here. It did execute in this review and in the developer run.
- Follow-up candidate, unverified against the pinned runtime: llama.cpp also
  honours non-`LLAMA_ARG_` names (`LLAMA_CACHE`, `HF_TOKEN`, `HF_ENDPOINT`).
  Outside this AC; worth a separate bug after checking the actual runtime build.
- Denying lowercase `llama_arg_*` is broader than POSIX requires but harmless.

## AC / DoD status

| Item | State |
| --- | --- |
| Refuses `LLAMA_ARG_*` before spawn, no values leaked | met |
| Focused tests: MODEL, second name, clean control, existing gates | met |
| Shared Pi launch documentation describes the contract | met in README, ungated (Finding 2) |
| Source tests, setup/install refresh, installed verification | source tests met; installed global refresh NOT met (Finding 1) |
| Gate attacked, not read | met — mutants above |
