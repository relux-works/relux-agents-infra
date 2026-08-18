# BUG-260818-jreo1p — managed Pi readiness 503 fix

## Reproduction

- Production `pi-infra` with fully cached Qwen weights: exit 1 in `4.16s`.
- llama.cpp logged `load_model`; exact readiness endpoint returned HTTP 503.
- `waitPiRuntimeReady` returned `runtime_readiness_invalid` immediately and cleaned the owned process/listener.

## Change

- `tools/agents-infra/internal/infra/pi_launch_posix.go`: retry exact HTTP 503 while the owned child remains alive and the configured deadline remains active. Connection errors retain their existing retry behavior. Read failures, every other non-200, malformed JSON, missing exact model, child exit, and timeout remain failures.
- `tools/agents-infra/internal/infra/pi_test.go`: production `RunPi` regression covers 503,503,200 and verifies Pi is reached only after request 3; 502 fails after request 1.
- `README.md` and `SKILL.md`: document the exact retry boundary.

## Validation

- Focused regression: exit 0 (`1.783s`).
- Narrowing/broadening mutant (`503` widened to all `5xx`): expected-red exit 1; 502 was incorrectly admitted and the named production-entry test failed. Original restored byte-for-byte (`cmp` exit 0).
- `go test ./internal/infra -count=1`: exit 0 (`70.890s`).
- `go test ./... -count=1`: exit 0 (root `54.145s`, attachments `0.957s`, infra `81.780s`).
- `go build ./...`: exit 0.
- `go vet ./...`: exit 0.
- `gofmt -d` on changed Go files: exit 0, no output.
- `./setup.sh`: exit 0; global binary/runtime verified.
- `agents-infra setup local /Users/alexis/src/local-models`: exit 0.
- `agents-infra verify local /Users/alexis/src/local-models`: exit 0.
- Installed Qwen text and tool live smokes subsequently exited 0 through the fixed production entry.

No files were staged or committed. The source checkout already contained unrelated tracked and untracked work; only the named readiness function, regression test, docs, and log entry were touched for this bug.
