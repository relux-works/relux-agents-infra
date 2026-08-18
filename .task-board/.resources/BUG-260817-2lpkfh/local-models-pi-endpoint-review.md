# TASK-260817-1q87y4 Reviewer Verdict

Verdict: **changes requested**. Route: `to-dev`.

## Accepted evidence

- GitHub release API independently reports Pi `v0.84.2` `pi-darwin-arm64.tar.gz` as 31,584,437 bytes with SHA-256 `c996e888b7f7dce44bcf24f69176ac646c44139d3916bd49a6b28e5a8c5e3a65`; the retained archive hashes to the same value.
- GitHub release API independently reports llama.cpp `b10470` `llama-b10470-bin-macos-arm64.tar.gz` as 11,079,764 bytes with SHA-256 `75c29cd80a67a8388b8ed08ea4a87531269a737c18945bdf3c3db6d5858024a9`; the retained archive hashes to the same value.
- Installed Pi entrypoint resolves through `/Users/alexis/.local/bin/pi`, returns `0.84.2`, and `pi-infra --print-config` reports the compiled 217-file identity as `verified`.
- Installed llama executable `/Users/alexis/.local/share/llama.cpp/llama-b10470/llama` returns build `10470`, commit `34af94cd9`, Darwin arm64.
- Current Qwen and Muse production plans contain the requested selectors, distinct ports `18011`/`18012`, explicit Muse DFlash path, 64K Muse context, exact declared target/draft subsequences, and requested/unverified capability reporting.
- No GGUF, safetensors, or model binary was found inside the repository.
- Reviewer reruns: `agents-infra verify local` exit 0; `.scripts/test-managed-pi-profiles.sh` exit 0; focused Go negative tests exit 0; `task-board validate` exit 0.

## Blocking review finding

### F1 — loopback/port gate has a production bypass

Negative shape: **bypass path around the check**.

Production call site: `.local/bin/agents-infra compose --mode primary-session --agent pi --project <mutant-project> --schema-version 1 --json`, which reaches `buildPiPrimarySessionLaunchPlan` after `parsePiConfig` / `parsePiRuntime`.

Two narrowed copies of the real project config were composed without launching a runtime:

1. Keep `base_url = http://127.0.0.1:18011/v1`, change only Qwen runtime bind host to `--host 0.0.0.0`. Production compose returned exit 0 and `status: ok`, reporting endpoint `127.0.0.1:18011` while carrying wildcard runtime argv.
2. Keep the same base URL, change only Qwen runtime port to `--port 19011`. Production compose returned exit 0 and `status: ok`, reporting endpoint/readiness on `18011` while carrying runtime port `19011`.

Evidence:

- `.temp/TASK-260817-1q87y4/mutant-wildcard-compose.json`
- `.temp/TASK-260817-1q87y4/mutant-port-drift-compose.json`
- `.agents/tools/agents-infra/internal/infra/pi_config.go`: `validatePiBaseURL` validates only the declared URL; `parsePiRuntime` accepts arbitrary non-empty runtime tokens and does not bind host/port tokens to that URL.
- `.scripts/test-managed-pi-profiles.sh`: `validate_qwen` and `validate_muse` assert the reported endpoint but do not assert the runtime host/port subsequence. Its post-hoc JSON overclaim mutations test the shell predicate, not refusal by the production resolver.

Impact: a profile can claim a loopback endpoint while exposing llama.cpp on all interfaces. Port drift can also make readiness query a different process than the direct child, admitting a foreign pre-existing backend before Pi starts.

## Required rework

1. Fix the reusable agents-infra source contract, not only this installed/project copy: bind managed runtime host and port to the exact `base_url` endpoint, or introduce an equally strict structured runtime contract that cannot express divergence.
2. Add production-entry negative tests that compose the wildcard-bind and port-drift mutants and require refusal. The tests must fail when only the runtime argv check is narrowed; direct helper tests or post-hoc plan JSON validation are insufficient.
3. Extend the project profile test to assert exact host/port argv for both profiles so configuration drift fails locally.
4. Run the source repo setup/install flow, then repeat `agents-infra verify local`, both `pi-infra --print-config` calls, focused/full Go tests, and board validation. Attach updated task-scoped evidence for another reviewer cycle.

## Reviewer probe anomaly

During tool readiness, `pi-infra --help` was forwarded as a normal managed Pi invocation and started the Qwen runtime. The reviewer terminated the exact spawned shell/runtime PIDs; port `18011` had no remaining listener. No recent file appeared under the checked llama.cpp cache roots. This probe is not producer evidence and does not change F1 or the verdict.
