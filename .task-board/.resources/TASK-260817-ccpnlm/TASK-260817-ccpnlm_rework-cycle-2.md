# TASK-260817-ccpnlm — developer rework cycle 2

## Outcome

Closed the reviewer-cycle-1 production argv bypass and replaced helper-only evidence with production-entry coverage.

- Bare unknown Pi long options now fail unless expressed as a self-contained `--name=value` token or a complete pre-delimiter flag/value pair.
- Empty managed `--api-key` values fail before side effects; diagnostics still redact non-empty values.
- Exact readiness does not follow redirects away from the configured loopback URL.
- Managed state uses exact UTF-8 SHA-256 keys, anchored no-follow components, post-open directory revalidation, random atomic catalog temporaries, and single-link regular lock files.
- Production compose now drives the named profile-state and deterministic 217-record Pi catalog narrowing suites.
- Production `RunPi` coverage attacks listener occupancy and indeterminate checks, malformed/mismatched readiness, a ready foreign listener with a dead selected child, runtime path/spawn failures, literal shell metacharacters, Pi spawn failure, point-of-use Pi mutation, SIGINT/SIGTERM forwarding, graceful group cleanup, forced shutdown escalation, lock release, loader/inbound Pi environment refusal, Qwen lifecycle, and Muse exact target/draft argv.
- Global Pi models/settings/auth/trust sentinels remain byte-identical.
- Capability labels remain requested/configured only; `verified` is empty and DFlash is `configured-unverified`.

Explicit non-claims remain unchanged: a malicious reviewed runtime and a malicious same-UID post-preflight bind winner are outside the launcher trust boundary.

## Reviewer bypass reproduction

Source-built production command:

`agents-infra pi --print-config --unknown`

- Exit code: `1` (expected-red refusal).
- Output: `unknown Pi option "--unknown" must use --name=value or a complete flag/value pair`.
- Log: `.temp/TASK-260817-ccpnlm/bare-unknown-probe-01.log`.

Narrowing controls:

- `--custom-extension=value`: exit `0`, preserved exactly in normalized argv.
- `--custom-extension value`: exit `0`, preserved exactly in normalized argv.
- Neither diagnostic created cache state.

## Validation

Every gate below ran as a standalone process without `tee`.

| Command | Exit | Evidence |
| --- | ---: | --- |
| `gofmt -w internal/infra/pi_*.go main.go main_test.go` | 0 | no output |
| `go test ./internal/infra -run 'Test.*Pi' -count=1` | 0 | `.temp/TASK-260817-ccpnlm/go-test-focused-02.log` |
| `go test . -run 'TestRunPi' -count=1` | 0 | `.temp/TASK-260817-ccpnlm/go-test-main-focused-02.log` |
| `go test ./... -count=1` | 0 | `.temp/TASK-260817-ccpnlm/go-test-full-03.log` |
| `go vet ./...` | 0 | `.temp/TASK-260817-ccpnlm/go-vet-02.log` |
| `go build ./...` | 0 | `.temp/TASK-260817-ccpnlm/go-build-02.log` |
| `GOOS=windows GOARCH=amd64 go build ./...` | 0 | `.temp/TASK-260817-ccpnlm/go-build-windows-01.log` |
| `git diff --check` | 0 | `.temp/TASK-260817-ccpnlm/git-diff-check-03.log` |
| untracked Pi source/manifest whitespace scan | 0 | `.temp/TASK-260817-ccpnlm/untracked-whitespace-01.log` |
| `task-board validate` | 0 | `.temp/TASK-260817-ccpnlm/task-board-validate-03.log` |

An earlier formatting invocation used repo-root paths while already inside the Go module and printed `stat ... no such file or directory`; it was not counted. It was rerun correctly with exit `0`. An earlier full-suite wrapper did not return an exit code before its 30-second yield; it was not counted. The recorded `go-test-full-03.log` run was repeated through a retained session and returned explicit exit `0`.
