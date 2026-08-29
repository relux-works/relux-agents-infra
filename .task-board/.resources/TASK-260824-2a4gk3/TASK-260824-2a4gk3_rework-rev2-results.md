# TASK-260824-2a4gk3 — revision 2 rework evidence

## Result

The accepted revision 1 candidate was reconciled with current `main` after
integration refused the overlapping `LOGBOOK.md` path. The combined Story tree
preserves every reviewed source, test, README, deployment, alias, and runtime
behavior from candidate tree `d932b6614b0095958ba356ebb1206ff2289eebac` and
adds both logbook entries landed by concurrent trunk work:

- `1815 — Managed Codex Sync Preserves User State`
- `1753 — Fast-Profile Removal Meets Runtime Preservation Boundary`

`git diff --name-status d932b661` reports only `M LOGBOOK.md`. The two added
blocks are byte-for-byte the current `main` text and are placed in timestamp
order between the Story's `1823` and `1751` entries. No reviewed production,
test, or README byte drifted from revision 1.

## Rework validation

All passing gates below were run directly in this developer run and returned
their real exit status.

| Command | Exit | Result |
| --- | ---: | --- |
| `go test -count=1 -run 'TestInstalledBinary(VerifyLocalInspectsEveryManagedSkillSurface|SetupAndVerifyLocalPreserveUnmanagedProviderSkillLinks)$' .` | 0 | Production installed-binary setup/verify gate checks managed surfaces and preserves unmanaged provider-owned links. |
| `go test -count=1 ./...` | 0 | Packages passed in 63.383s, 1.274s, and 119.304s. |
| `go vet ./...` | 0 | Clean. |
| `go build ./...` | 0 | Build succeeded. |
| `gofmt -d <changed Go files>` plus empty-output assertion | 0 / 0 | No formatting delta. |
| `git diff --check` | 0 | Clean. |
| `agents-infra verify global` | 0 | Verified `/Users/alexis/.agents`. |
| `agents-infra verify local /Users/alexis/src/casual-talks` | 0 | Verified the deployed local runtime. |

Revision 1 already ran source bootstrap, global setup, local setup, and both
verifies with exit 0; those exact commands and results remain attached in
`TASK-260824-2a4gk3_results.md`. They were intentionally not repeated from the
old Story fork after `main` landed a newer managed Codex config migration:
reinstalling from the pre-reparented Story source could undo that newer
installed state. Current global and local verifies were repeated instead.

## Target, alias, and provenance evidence

The installed regular executable aliases remain unchanged:

| Artifact | Size | SHA-256 |
| --- | ---: | --- |
| local `agents-infra` | 475 bytes | `f3982633685fb85d541af53ed1c07b57101f0708197344282c6f00d7ff9b57b8` |
| local `openai-infra` | 292 bytes | `fa0596ed7a039dbe513f49d30fc5a0115cad9f50f84c4e3c7930bbfd7938f173` |
| local `anthropic-infra` | 298 bytes | `d1acbb048939dfcbbbd924f83dcd7cf879b77328e3e61cbcd818ea4fe40c59c7` |
| local `qwen-infra` | 288 bytes | `9ee57860a93ea473c1481029dd34998fa52614eb106c8d5edd6ec0acd349b60a` |

Each alias `--print-config` and each schema-v1 primary-session compose exited
0. Machine assertions exited 0 for:

- OpenAI / Codex / `gpt-5.6-sol` / `high`, with target and effective sources
  equal to the `casual-talks` project config;
- Anthropic / Claude Code / `claude-opus-5` / `high`, with the same exact
  target/effective provenance rule;
- Qwen / Pi / the operator-approved exact MLX weights path / `off`, profile
  `qwen-3.8-27b-mlx-8bit`, provider `local-qwen`, endpoint
  `http://127.0.0.1:18011/v1`, and runtime argv containing exact
  `--host 127.0.0.1 --port 18011` coordinates.

The Qwen Section 5 equalities all passed on the production compose document:

```text
resolved.model.value == resolved.profile_provider.value + "/" + target.model
resolved.endpoint.value == target.endpoint
resolved.endpoint.value == pi.runtime.endpoint
resolved model/provider/endpoint sources == selected profile source
```

Requested capabilities remain `text` and `tools` with
`verification = not-claimed`; no proxy fact was promoted to attestation.

## Refusal, immutability, and cleanup

The real installed production entrypoint was attacked with a divergent model:

```text
.local/bin/openai-infra --print-config -- --model divergent-model
exit 1 (expected-red)
```

It refused with stable code `target_identity_conflict`, named entrypoint,
target, field, source, and remediation before provider launch. The project
config SHA-256 remained
`464c699f4dfe505203bc0ac80abb05238f6e04ef645ff36dba65e86b6f26b7b6`
before and after all print, compose, and refusal probes (`cmp` exit 0).

Post-probe absence checks are reported as failing commands, as required:

| Probe | Exit | Expected result |
| --- | ---: | --- |
| `lsof -nP -iTCP:18011 -sTCP:LISTEN` | 1 | No listener. |
| `pgrep -af 'mlx_lm\\.server.*--port 18011'` | 1 | No runtime process. |
| `lsof <profile session.lock>` | 1 | No lock holder. |

The live Qwen text plus safe write/read tool round trip was not repeated for
this logbook-only reconciliation. It was independently re-run by revision 1's
reviewer with exit 0 and is preserved as
`TASK-260824-2a4gk3_reviewer-qwen-smoke.jsonl.gz`; the accepted verdict records
the text response, distinct 39-byte reviewer file, successful write/read tool
events, zero reasoning usage, loopback-only bind, and cleanup.

No secret or arbitrary environment value was inspected or persisted in this
revision 2 evidence. Logs contain only command output, public target/runtime
coordinates, paths, digests, and environment variable names already admitted
by the managed contract.

