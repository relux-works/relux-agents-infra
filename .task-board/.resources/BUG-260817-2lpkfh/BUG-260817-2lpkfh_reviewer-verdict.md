# BUG-260817-2lpkfh — Reviewer Verdict

Verdict: **accepted**. Route: `done` (reviewer records acceptance evidence only; no `commit_ack`).

Reviewer run: RUN-260817-e5ed53 (not goal-bound). Read-only; no product code modified.
All mutation work was done in a throwaway module copy under
`.temp/BUG-260817-2lpkfh-review/mutant`, since removed. `git diff --stat` on
`internal/infra/pi_config.go` is empty — no reviewer mutation leaked into the tree.

## What the change does

`validatePiRuntimeEndpointArgv` (`tools/agents-infra/internal/infra/pi_config.go:367`)
is called from `parsePiProfile` (`pi_config.go:207`), i.e. during project-config
parsing, before compose diagnostics, state creation, listener preflight, or spawn.
It derives `(host, port)` from `base_url` via `piBaseURLEndpoint` and requires the
runtime argv to carry exactly one spaced `--host <base_url-host>` pair and exactly
one spaced `--port <base_url-port>` pair.

## Gate attacked, not read

### Production entry 1 — `agents-infra compose --mode primary-session --agent pi`

Built from the working tree and driven as a standalone process against generated
project configs. Control accepted, every divergence form refused with
`invalid_project_configuration` on `agents.pi.profiles.profile.runtime.argv`:

| Attack | Exit | Result |
| --- | ---: | --- |
| exact loopback control | 0 | accepted, argv preserved literally |
| `--host 0.0.0.0` | 1 | refused |
| `--port 19021` vs `base_url` 18021 | 1 | refused |
| `--host=0.0.0.0` (attached) | 1 | refused |
| `--port=19021` (attached) | 1 | refused |
| `--host` absent | 1 | refused |
| `--port` absent | 1 | refused |
| duplicate `--host` (loopback then wildcard) | 1 | refused |
| `--host ::` | 1 | refused |
| `--host localhost` | 1 | refused |

### Production entry 2 — `agents-infra pi --print-config`

Control exit 0 / `status: ok`; wildcard exit 1; port drift exit 1. Same resolver,
same error field. Diagnostics cannot report an endpoint the child would not bind.

### Bypass attempts that failed to find a hole

- **Gate unreached when Pi is not resolvable.** Re-ran the wildcard and port-drift
  mutants with `PATH=/usr/bin:/bin` (no Pi asset). Both still refused with
  `invalid_project_configuration`. The gate lives in config parsing, not behind
  managed-profile resolution, so the `managed:false` path does not skip it.
- **Environment-variable override of the CLI bind.** The real runtime
  (`llama-b10470`) does export `LLAMA_ARG_HOST` / `LLAMA_ARG_PORT`
  (confirmed in `libllama-common.0.1.1.dylib` and `llama serve --help`), and
  `runtimeCmd.Env = opts.Environ` (`pi_launch_posix.go:142`) passes the inherited
  environment through unfiltered for these names. Ran the real binary with
  `LLAMA_ARG_HOST=0.0.0.0 LLAMA_ARG_PORT=19099` plus `--host 127.0.0.1 --port 18099`:
  llama emits `LLAMA_ARG_HOST environment variable is set, but will be overwritten
  by command line argument --host` (same for `--port`). CLI wins — **not** a bypass.
  Reported as proven, not inferred.
- **Flag-value swallowing** (`--chat-template --port 18021 ...`) would leave llama on
  its default 8080 while the gate counts one valid pair, but the launch then fails
  closed: `preflightPiListener` holds the declared port and readiness on the declared
  port never succeeds. No exposure, no foreign-backend attach.
- **UNIX-socket bind** (`--host <x>.sock`, supported by this runtime) is excluded by
  the exact `127.0.0.1` equality.

### Narrowing mutants — bounds proven, not just gate presence

Each narrowed one bound while leaving the gate in place (no delete-only mutant):

| Mutant | Test that went red |
| --- | --- |
| port value no longer compared to `base_url` port | `TestRunComposePiRefusesRuntimeEndpointDivergence/runtime_port_drift` — "production compose admitted runtime/base_url endpoint divergence" |
| host value no longer compared to `base_url` host | `TestRunComposePiRefusesRuntimeEndpointDivergence/wildcard_runtime_bind` |
| `hostFlags != 1 \|\| portFlags != 1` relaxed to `< 1` | `TestParsePiPolicyRejectsMalformedUnsafeUnknownAndNarrowedInputs/duplicate_runtime_port` |

Removing the attached-form (`--host=`) clause produced no failure, but that is not a
hole: an attached token never matches `token == "--host"`, so the exactly-one rule
still refuses it (verified live through production compose above). The clause only
improves the error message.

## Reviewer validation reruns

| Gate | Exit | Result |
| --- | ---: | --- |
| `go test ./... -count=1` | 0 | Pass (root 52.9s, infra 85.2s, attachments 1.3s) |
| `go vet ./...` | 0 | Pass |
| `gofmt -l tools/agents-infra/` | 0 | Empty |
| `agents-infra verify global` | 0 | Pass |
| `agents-infra verify local` (source repo) | 0 | Pass |
| `agents-infra verify local` (`local-models`) | 0 | Pass |
| Installed `~/.local/bin/agents-infra` carries gate strings | 0 | `--host must be followed by exact base_url host`, `--port must be followed by exact base_url port` present in the 21:11 binary |

The `local-models` `.local/bin/agents-infra` is an `sh` wrapper that exports
`AGENTS_INFRA_SOURCE_DIR=/Users/alexis/src/relux-works/relux-agents-infra`, so the
downstream entry inherits the source-repo fix rather than a divergent copy. This
closes the previous cycle's F1 requirement to fix the reusable source contract.

## Acceptance criteria

- Production compose refuses wildcard bind and runtime-port drift — **met**, attacked through two production entries.
- Accepted argv expresses the exact 127.0.0.1 `base_url` endpoint — **met**, control plan preserves the literal token vector and reports the matching endpoint.
- Tests exercise installed/production entry paths — **met**: `TestRunComposePiRefusesRuntimeEndpointDivergence` (`main_test.go:712`) and `TestInstalledLocalAgentsInfraComposeRefusesPiRuntimeEndpointDivergence` (`installed_binary_setup_test.go:636`), both with a valid narrowing control, both proven to bite under narrowing.
- Documentation states the invariant — **met**: `README.md:486-491`, `SKILL.md:306-307`, `.research/260817_pi-local-model-launch-contract.md:152`. Every example config in the repo satisfies the new rule.
- Focused and full Go validation plus setup/install/verify pass — **met**, rerun independently above.

## Non-blocking follow-ups (do not hold this bug)

1. **Production-entry negatives skip silently without a gitignored asset.**
   `mainTestOfficialPiAsset` calls `t.Skipf` when
   `.temp/TASK-260817-2h8hn4/pi-standalone-darwin-arm64-0.84.2/pi/pi` is absent.
   Proven: with the path removed, `TestRunComposePiRefusesRuntimeEndpointDivergence`
   reports `--- SKIP` and the package reports `ok`. On any checkout without that
   other-task-scoped, gitignored artifact, this bug's production-entry coverage
   disappears with no failure signal. The helper is not in `HEAD` and backs 6 tests,
   so this is a story-wide fixture convention inherited from `TASK-260817-2h8hn4`,
   not a defect introduced here. Worth converting to a hard failure or a durable
   fixture at story level.
2. **The documented invariant is not pinned by the docs gate.**
   `pi_operator_docs_test.go` asserts many README/SKILL fragments but none of the new
   endpoint-binding sentences, so the documented contract can be dropped silently.
3. **`--reuse-port` is not refused.** `preflightPiListener` binds without
   `SO_REUSEPORT`, so an occupied port is still caught; but a profile carrying
   `--reuse-port` would let a later same-port `SO_REUSEPORT` process share the
   declared endpoint. Outside this bug's AC (neither wildcard nor drift) and outside
   the reviewed trusted-policy configs in use; worth a follow-up on the story.

## Reviewer probe hygiene

The only runtime process started was `llama serve -m /nonexistent/model.gguf`, twice,
to settle the env-precedence question. Both exited non-zero on model load before any
socket bind; `lsof` confirmed no listener appeared on the probe ports. No managed Pi
session, no `pi-infra` launch, no model download. Probe logs:
`.temp/BUG-260817-2lpkfh-review/llama-cli-only.log`, `llama-env.log`.
