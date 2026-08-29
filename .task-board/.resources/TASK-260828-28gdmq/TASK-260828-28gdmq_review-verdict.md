# TASK-260828-28gdmq review verdict

Change Request: `CR-TASK-260828-28gdmq-1`, revision 1  
Base: `3f313d9175f2ada9b9ab3320ab524c0918f9daac`  
Candidate tree: `52077c3859885104e89cf6bb6bc50cd8022a026f`  
Patch SHA-256: `d8e69d57e67497ac142350d9efa3d57fbdd03d6094c56a280ee0041f5f2370bd`

## Verdict

**Changes requested; route to `analysis`.**

The central negative result is independently confirmed: `model-harness run`
forwards the engine child's stdout/stderr but does not observe the HTTP request
or response body between Pi and the engine. No feature implementation was
required by this investigation, so the deliberately empty product-code delta is
not a review failure. B4 and B7 both reproduce. The revision is not acceptable
yet because the investigation's deliverable is exact reporting, while several
claims contradict the attached evidence or the host state.

## Independent attack evidence

All reviewer processes were terminated before this verdict was recorded.

| Check | Result |
| --- | --- |
| Live managed runtime | Candidate-built `model-harness` launched installed `mlx_lm.server` 0.31.3 with the 261 MB fixture on `127.0.0.1:18128`. |
| HTTP vs harness streams | HTTP response was 620 bytes and contained `chatcmpl-77032848-bd75-4070-889e-7e9017ace9f5` plus completion text. Harness stdout was 0 bytes; prompt, caller request-id, tool-schema, response-id and completion nonces had 0 hits in both harness streams. Harness stderr contained runtime metadata and the access line only. |
| B4, mid-body crash | After the runtime logged `POST /v1/chat/completions ... 200 -`, PID 41829 was killed while processing a 24,032-token prompt. Curl exited 52 with `http_code=000`, `bytes=0`, `Empty reply from server`; the harness record still said 200. Reproduced. |
| B7, directed SIGTERM | Directed SIGTERM to harness PID 43473 produced exit 143 and zero-byte harness stdout/stderr. Runtime shell PID 43482 remained alive with PPID 1, with its `sleep` child still running. Reproduced in a wrapper that stayed alive so runner PTY teardown could not kill descendants. |
| Shared broker | Producer archive contains two distinct Pi response IDs and two body-less POST lines in one `shared-runtime.log`; no prompt nonce, response ID, lease or run marker appears in that log. Consistent with B3. |
| Compaction restraint | The persisted four-turn transcript has 8 message records, no `compaction` record, and assistant input counts 4823, 4842, 4861 and 4880 around the configured 4870 threshold. Each attached turn stderr has exactly one POST. Not calling the non-firing a defect is correct. |
| Retention | Production opens `runtime.log` with `O_APPEND`; cleanup removes lease mirrors, broker state and socket but not the log. No rotation/cap/pruning call site was found. Sustained-load behavior remains honestly untested. |
| llama-server availability | `llama-server` and `llama-cli` are absent. `llama.cpp` is not installed, but Homebrew does offer the formula. Installing software/model fixtures is outside this read-only reviewer run. |
| Validation | `go test ./internal/modelharness -count=1` passed (0.707s); `go vet ./...` passed; `go test ./... -count=1` passed (root 85.090s, infra 143.753s, modelharness 1.349s); `gofmt -l .` and `git diff --check` were clean. |

Raw reviewer evidence is attached separately as
`TASK-260828-28gdmq_review-evidence.tar.gz`.

## Required corrections

1. **Capability claim does not reproduce: B8's Homebrew statement is false.**
   `.research/260828_model-io-observability-through-harness.md:436-438` says
   Homebrew has only `ollama` and `whisper-talk-llama`. On this host,
   `brew search llama` returns `llama.cpp`, and `brew info --json=v2 llama.cpp`
   reports the formula available but uninstalled. Keep the honest result
   (`llama-server` is absent and behavior is unknown), but state the real
   boundary: exercising it needs a scoped install and a compatible GGUF fixture,
   or a decision that llama-server is not an active deployment target.

2. **The Pi trace event count is wrong.**
   `.research/260828_model-io-observability-through-harness.md:251-255` says 77
   events. The attached `pi-turn-stdout.json` contains 78 valid JSON records;
   the report's own census also sums to 78. Correct the count so the end-to-end
   trace is exact.

3. **The test comment contradicts B8.**
   `tools/agents-infra/internal/modelharness/run_observability_test.go:88-91`
   says prompt/completion bodies use HTTP on “all three candidate runtimes,”
   while the report correctly calls llama-server unknown and untested. Narrow
   the comment to the two audited runtimes, or to the channel-independent fact
   the test actually establishes: bytes a child does not write to stdout/stderr
   cannot be recovered from those streams.

4. **Scope the headline to full wire bodies, not all managed-path output.**
   The report's verdict says nothing captured on the managed path contains a
   prompt or completion, but its own Pi artifact contains the user-visible
   message and assistant completion. The established gap is narrower and more
   important: model-harness-captured engine streams contain neither the fully
   assembled HTTP request nor the HTTP response body, and Pi-side records cannot
   be joined to engine-side requests. Rewrite the headline/result so it does not
   erase the partial Pi-side evidence it later documents.

5. **Make the closure decisions explicit for every blocker.**
   B1 names its decision, but B3-B8 mostly list implementation options. Record
   the owning decision/input explicitly: B3 per-lease segregation vs durable
   lease log; B4 proxy vs runtime cooperation; B5 Pi attempt events vs proxy
   records; B6 concrete size/rotation/retention policy and load protocol; B7
   whether standalone harness owns supervisor-signal cleanup; B8 whether
   llama-server is an active managed target and, if so, who supplies the
   install/GGUF fixture. B2 may remain explicitly dependent on B1.

After these reporting corrections, rerun the narrow modelharness tests and
refresh both the board report resource and repository research copy so they
remain byte-identical.
