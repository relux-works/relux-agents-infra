# TASK-260825-kpky8f implementation evidence

Date: 2026-08-25 (Europe/Moscow)
Commit: `5b081ad` (`Support native Qwen thinking and reject Pi yolo`)

## Outcome

- Canonical Qwen non-`off` reasoning now requires a managed Pi profile with
  `reasoning = true` and `compat.thinking_format = "qwen-chat-template"`.
- A source-backed `reasoning = "medium"` resolves to `medium`, generates a
  reasoning-capable Pi model entry, and emits native Pi argv
  `--thinking medium`.
- `agents.pi.primary_session.yolo_mode` composes root-to-leaf with a safe false
  default and explicit-child-false masking. Explicit true fails before
  executable lookup or launch with `pi_yolo_mode_unsupported`; it is never
  translated to Pi's unrelated `--approve` project-trust flag.
- README and the source `relux-agents-infra` skill document the native contract,
  safe default, refusal, and operator verification workflow.

The sanitized pinned-runtime investigation is attached separately as
`TASK-260825-kpky8f_pi-native-contract-probe.md`.

## Validation run directly

| Command | Exit | Evidence |
| --- | ---: | --- |
| `go test ./internal/infra -run 'TestParsePi|TestPiPrimaryYolo|TestPiYoloTrue|TestCanonicalQwen|TestGeneratedPiCatalog' -count=1` | 0 | Focused config, safe-default, refusal, Qwen, and catalog coverage. |
| `go test . -run 'TestRunComposeCanonicalQwen|TestPiOperatorContract|TestReluxAgentsInfraSkill|TestBoundedModelCheckREADME|TestReluxAgentsInfraSkillPinsBounded' -count=1` | 0 | Production compose and documentation contracts. |
| `go test ./... -count=1` | 0 | Full suite: root package `78.693s`, attachments `1.646s`, infra `93.825s`. |
| `go vet ./...` | 0 | Go vet clean. |
| `go build ./...` | 0 | Native Darwin/arm64 build clean. |
| `env GOOS=windows GOARCH=amd64 go build ./...` | 0 | Windows build clean after the platform launch-gate change. |
| `git diff --check` / `git diff --cached --check` | 0 / 0 | Diff hygiene clean before commit. |
| Source-built `agents-infra compose --mode primary-session --entrypoint qwen-infra ... --json` | 0 | Reported target and resolved reasoning `medium`, profile-config provenance, yolo false provenance, and argv `--thinking medium`; verified pinned Pi `0.84.2`. No provider/runtime launch. |
| Source-built `agents-infra target qwen-infra --print-config` | 0 | Reported effective reasoning `medium` with profile-config source and provider argv `--thinking medium`. No provider/runtime launch. |
| Source-built Pi compose with `yolo_mode = true` | 1 | Expected refusal: JSON code `pi_yolo_mode_unsupported`; stderr states that `--approve` controls project-local input trust and recommends false/omission. This is an expected-red gate, not a passing command. |

After temporary narrowing mutants were exactly restored, the four focused
production/gate tests were rerun together and exited 0.

## Negative evidence

- Yolo narrowing mutant: changed the production gate to reject true only when
  a profile was also selected. `TestPiYoloTrueFailsClosedBeforeComposeOrLaunchLookup`
  exited 1: compose admitted the unsupported policy and direct `RunPi` reached
  executable lookup. Exact restoration made the test exit 0.
- Qwen narrowing mutant: limited the production capability gate to `high`.
  `TestCanonicalQwenProfileAssertionsFailClosed` exited 1 because medium
  configurations with missing and wrong `thinking_format` were admitted.
  Exact restoration made the test exit 0.

Production call sites covered by the refusal test are
`BuildPrimarySessionLaunchPlan` and `RunPi`; the Qwen validation is driven via
`ResolveCanonicalTarget` and the production compose entry point.

## Scope notes

- No Pi provider session or local-model runtime was launched.
- No external project policy was modified.
- Working tree was clean after commit; no push was performed.
