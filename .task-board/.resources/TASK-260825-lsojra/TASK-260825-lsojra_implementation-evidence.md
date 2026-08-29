# TASK-260825-lsojra implementation evidence

## Candidate

- Branch: `task-board/story/STORY-260825-1r7z9o`
- Handoff commit: `fc0c69ed8d898702e3f049490724225ad8ce94f2`
- Core implementation: `2e75099ac9f9a35491fad29047dc15b117d8fc1b`
- Production mutant controls: `8c47c1361bc24fb8ffbccf6d45c40eefcf30b67b`
- Main reconciliation: `3d203b13e59f1abb6eb7cc661a4337a58fc14242`, `5e3bbe03168581a679efe8084d39d24d60d6adbd`, `a94e67e`
- Race-fixture and direct-control stabilization: `6146f24`, `fc0c69e`
- Worktree was clean and `git diff --check` exited 0 at handoff.

## Implemented scope

- Strict optional `[runtime.sharing]` parsing with byte-compatible exclusive behavior when absent and fail-closed complete-field validation when present.
- Terminal-independent broker election, publish-before-run unauthorized launcher, revision-9 protocol v6 authorization, broker/runtime kernel identity verification, 13-gate client attestation, and independent `/v1/models` verification.
- Automatic lease acquisition in the normal managed Pi path, RUN-scoped client state and Pi process groups, same-profile cross-project runtime reuse, stale lease cleanup, first-lease grace, effective starter policy, final-release linger/shutdown, and bounded operator status/stop.
- Fail-closed zombie handling, predecessor record carry-forward, identical broker/operator reclamation evidence, unreachable-owner reporting, and no client signal at acquisition deadline.
- Top-level `agents-infra runtime status|stop|broker|runtime-launch`; `pi --print-config` remains non-connecting and side-effect-free.
- Revision-9 shape oracle, decoder/production wiring evidence, 18 calibrated mutants, measured blindness, reject-all harness negative, and one production runtime-launch process per mutant for the plain-valid control.

## Main reconciliation

At the reconciliation checkpoint, `main` had advanced by four commits. The two product commits were carried into this candidate without merging or rebasing; the two remaining main-only commits are board-state snapshots. At handoff `git rev-list --left-right --count HEAD...main` reported `19 4`.

- The accepted non-overlap model-check/Qwen files compare byte-for-byte with `main`: `git diff --exit-code main -- <non-overlap product paths>` exited 0.
- The overlap in `pi_config.go`, `pi_launch_posix.go`, `pi_plan.go`, `pi_test.go`, `main.go`, README, SKILL and LOGBOOK is recorded in `main-overlap.diff`. It preserves native Qwen `--thinking medium`, the pre-launch `pi_yolo_mode_unsupported` refusal, and model-check while adding only the shared-runtime surface and combined documentation/history.
- The current context-aware readiness and bounded process-group cleanup APIs are used by shared broker/client paths. A cancelled acquisition starts and signals nothing.
- No real Qwen/MLX model was launched; tests use task-local fake executables and loopback listeners.

## Final green gates

All commands were direct standalone processes; values below are real exit codes.

| Command | Exit | Evidence |
| --- | ---: | --- |
| `git diff --check` | 0 | clean output |
| `go build ./...` | 0 | `go-build-03.log` |
| `go vet ./...` | 0 | `go-vet-03.log` |
| `go test ./... -count=1` | 0 | `go-test-03.log`: main `88.935s`, attachments `1.198s`, infra `120.908s` |
| `go test -race ./internal/infra -run 'TestSharedRuntime' -count=1` | 0 | `go-test-race-shared-04.log`: infra `26.125s` |
| revision-9 oracle/mutant/production-control focused suite | 0 | `go-test-mutation-02.log`: `2.317s` |
| Linux amd64 test-binary compile | 0 | `go-cross-linux-02.log` |
| Windows amd64 test-binary compile | 0 | `go-cross-windows-02.log` |

## Mutation calibration

Production decoder/oracle agreement: 398 generated frames, 48 accepted and 350 refused. Revision wiring agreement: 417 frames. The production plain-valid control reached the target under all 18 mutants. Each named mutant was killed by the generated witness below; `unknown_by_wire_form` is the sole declared over-refuser.

| Mutant | Killing frame | Direction | Measured blind baselines |
| --- | --- | --- | --- |
| `unknown_ignored` | `position/unknown_first` | admits | none |
| `dup_ignored` | `occurrence/schema/x2` | admits | none |
| `dup_ignored_first_wins` | `occurrence/schema/x2` | admits | none |
| `trailing_ignored` | `structural/two_objects` | admits | rev7, review_c71188 |
| `missing_ignored` | `occurrence/schema/x0` | admits | rev7, review_c71188 |
| `shape_gate_deleted` | `occurrence/schema/x0` | admits | none |
| `dup_only_if_values_differ` | `occurrence/schema/x2` | admits | rev6 |
| `dup_only_protocol_version` | `occurrence/schema/x2` | admits | rev6, review_c71188 |
| `unknown_only_caller_chosen_field` | `length/unknown_0B` | admits | rev6 |
| `unknown_case_folded` | `identity/upper/schema` | admits | rev6, review_c71188 |
| `unknown_prefix_allowed` | `identity/suffix_v2/schema` | admits | rev6, review_c71188 |
| `dup_only_exactly_two_total` | `occurrence/schema/x3` | admits | rev6, rev7 |
| `unknown_allow_over_32` | `length/unknown_33B` | admits | rev6, rev7 |
| `dup_only_when_separated` | `occurrence/schema/x2` | admits | rev6, review_c71188 |
| `unknown_ascii_only` | `identity/homoglyph/schema` | admits | rev6, rev7, review_c71188 |
| `unknown_nonempty_only` | `length/unknown_0B` | admits | rev6, rev7, review_c71188 |
| `dup_keyed_on_wire_form` | `encoding/escaped_repeat/schema` | admits | rev6, rev7, review_c71188 |
| `unknown_by_wire_form` | `encoding/escaped_schema` | over-refuses | rev6, rev7, review_c71188 |

The non-gate `reject_all_probe` reddens all three harness rules: plain-valid admission, declared disagreement side, and measured-baseline blindness.

## Diagnostic red history

These commands failed during development and are not reported as passing:

- The first post-reconciliation focused compile exited 1 because current main added the readiness context and replaced the old process-group helpers. Shared broker/client code was adapted and the reruns passed.
- `go-test-race-shared-01.log` exited 1 because race-instrumented broker startup exceeded the synthetic 2-second fixture timeout.
- `go-test-race-shared-02.log` exited 1 after narrowing the remaining issue to a 3-second attestation fixture poll. Test-only bounds were corrected without weakening production gates; runs 03 and 04 exited 0.
- `go-test-mutation-01.log` exited 1 because an auxiliary asynchronous process observer missed one valid target under load while the target remained live. The control now uses the target's post-`execve` marker directly; `go-test-mutation-02.log` exited 0.

No required final gate remains red.
