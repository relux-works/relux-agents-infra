# Cycle 8 trust-boundary directive

The user goal is a practical project-aware Pi launcher that can run the two requested local profiles. The architecture must not make Muse or Qwen depend on a future agents-infra backend catalog, compiled observer, internal proxy, or independent DFlash attestation API.

Required decisions:

- Treat `runtime.executable` plus literal argv from reviewed project TOML as explicit trusted executable policy. Selecting it authorizes that local code to run; agents-infra does not claim to prove a trusted runtime's internal honesty.
- Preserve the existing exact managed Pi identity and argv-parser gates. Those protect the wrapper's own policy boundary and are not being weakened.
- For the local runtime, guarantee only what the launcher can reproduce: exact config provenance and validation, absolute argv-only spawn, no shell/interpolation, loopback-only endpoint, preflight refusal when the configured listener is already occupied, direct-child liveness, exact `/v1/models` discovery, process-group ownership/cleanup, isolated Pi state, and no intentional attach/fallback.
- Explicitly exclude a malicious runtime executable and a malicious same-UID process winning the post-preflight bind race from the threat model. Do not describe preflight plus readiness as cryptographic listener ownership.
- Report Qwen `text`/`tools` and Muse `dflash` as requested/configured capabilities. Runtime-reported status may be included as unverified diagnostic provenance, but must never be labeled independently verified or used as a false trust root.
- DFlash launch acceptance is: exact configured target/draft argv, selected runtime child remains alive, exact target appears in readiness, and Pi smoke/benchmark evidence is an operator verification step. No silent fallback is added by agents-infra, but the launcher does not claim it can detect a trusted runtime silently disabling DFlash without an authoritative runtime API.
- Keep model acquisition, conversion, benchmarking automation, secure runtime distribution, backend catalogs, compiled observers, proxy adapters, and cryptographic attestation out of this story.
- Update downstream implementation and operator-documentation acceptance criteria to this practical boundary.

The cycle-7 compiled adapter/backend-observer design is explicitly rejected as scope expansion and because it leaves the requested Muse profile unsupported.
