# STORY-260825-7oqacp — story-final review verdict

- Verdict: **accepted**
- Change Request: `CR-STORY-260825-7oqacp-1` revision `1`
- Reviewer run: `RUN-260825-f68fb3` (claude-opus-5/high)
- Base `8f01e31aabf4b7b2d2b4c24e7b1bba3bf845fe78` → candidate tree `b88203bbcd6293a56a41fa4ad893a0023e68be55`
- Date: 2026-08-25

## Candidate identity

`git rev-parse HEAD` = `5b081ad3ffd002634ed1e992c70a35dba839174f`; `HEAD^{tree}` =
`b88203bbcd6293a56a41fa4ad893a0023e68be55`, byte-identical to the published
candidate tree. `git status --short` empty, `git rev-list --count HEAD..main` = 0,
`main..HEAD` = 1. The attached patch `STORY-260825-7oqacp_change-request_rev1.patch`
hashes to `7c76d3d29587f1fd05740aeeb48f5b97ed57618a483d07042bc68569d60536ae`
(matches the CR record) and `git apply --check --reverse` succeeds, proving the
patch is exactly this working tree and nothing else. 13 changed paths, all
on-scope (Pi config/plan/launch/canonical-target production code, their tests,
README/SKILL/LOGBOOK). No unrelated repository changes.

## Capability claims verified against the pinned Pi binary, not accepted on report

The pinned asset `pi-standalone-darwin-arm64-0.84.2/pi` was inspected directly.

- Native Qwen thinking reproduces: the binary contains
  `} else if (compat.thinkingFormat === "qwen-chat-template" && model.reasoning) {`
  followed by `params.chat_template_kwargs = { enable_thinking: !!options?.reasoningEffort, preserve_thinking: true }`.
  The new gate requires exactly that conjunction (`reasoning = true` **and**
  `thinking_format = "qwen-chat-template"`), so the validation condition is the
  runtime condition, not a proxy for it.
- That branch does **not** gate on `supportsReasoningEffort`, so the canonical
  profile's `supports_reasoning_effort = false` does not suppress thinking. The
  contract is coherent.
- The `--approve` claim reproduces: the binary parses only `--approve`/`-a`, and
  `docs/usage.md:247` / `README.md:614` define it as "Trust project-local files for
  this run". `docs/security.md` states Pi has **no built-in sandbox** and that
  project trust "does not restrict what the model can ask tools to do". A grep for
  `--yolo`, `--dangerously*`, `--auto-approve`, `--full-auto`, `--unattended`,
  `--permission*`, `--sandbox` returns nothing. There is no native unattended
  tool-execution policy to map `yolo_mode = true` onto, so refusal — not silent
  acceptance and not an `--approve` translation — is the correct branch and matches
  the AC's "explicit unsupported-capability diagnostic otherwise".

## Gates attacked, not read

Mutation probes (applied, run, restored byte-exact; tree verified clean afterwards):

| # | Mutant | Expected-red test | Result |
| --- | --- | --- | --- |
| A | Delete the yolo gate from `buildPiPrimarySessionLaunchPlan` (compose call site) | `TestPiYoloTrueFailsClosedBeforeComposeOrLaunchLookup/compose` and `/launch` | FAIL (`error = <nil>`, and launch reached executable lookup) |
| B | Delete the yolo gate from `RunPi` (posix launch call site) | `.../launch` | FAIL — reached `fork/exec /must/not/run`, proving the gate is what stops exec |
| C | Narrow the parse gate `!p.Reasoning && p.Thinking != "off"` → `== "max"` | `TestParsePiPolicyRejectsMalformedUnsafeUnknownAndNarrowedInputs/non-reasoning_medium` | FAIL |
| D | Narrow the thinking-format check to a nil-only test (accept any non-nil value) | `TestCanonicalQwenProfileAssertionsFailClosed/wrong_qwen_thinking_format` | FAIL |
| E | Narrow the capability condition `target.Reasoning != "off"` → `== "max"` | `.../missing_qwen_thinking_format` and `/wrong_qwen_thinking_format` | FAIL |
| F | Narrow the yolo gate itself to fire only when the policy source is empty | `TestPiYoloTrueFailsClosedBeforeComposeOrLaunchLookup` (both subtests) | FAIL |

Every mutant is caught, including four true narrowings (C–F), not only deletions.
`git status --short` is empty after restoration.

Production call-site coverage was checked by grep, not assumed: every consumer of
`composite.PiPrimarySession` on a launch or compose path is gated —
`RunPi` (posix and windows) and `buildPiPrimarySessionLaunchPlan`, which is the
single funnel for `BuildPrimarySessionLaunchPlan`, `BuildCanonicalTargetLaunchPlan`,
and `model_check`. `main.go` exposes no `-d`/full-trust escape hatch for `pi`.
`agents.targets` with `environment = "pi"` is admitted only for `vendor = "qwen"`
(`project_config.go:400`), so the hardcoded `qwen-chat-template` requirement is
exactly scoped and not an over-broad gate.

## Real-binary attack against the canonical config shape

A binary built from the exact candidate (`go build -o .../agents-infra-candidate .`)
was run against a temp copy of `/Users/alexis/src/.agents/.configs/project-config.toml`
with `yolo_mode` flipped to `true`. All four production entry points refused with
exit 1, code `pi_yolo_mode_unsupported`, naming the exact source file:

- `compose --mode primary-session --entrypoint qwen-infra`
- `compose --mode primary-session --agent pi`
- `target qwen-infra --print-config`
- `pi --version` (the real `RunPi` exec path)

`pgrep -fl 'mlx_lm.server|llama-server'` exited 1 with empty output afterwards — the
refusal happens before executable lookup, so no local model runtime was started.

## Canonical /Users/alexis/src rollout, independently reproduced

Non-launching compose from the candidate binary against the real canonical config
exited 0 and reported:

| Field | Value | Source |
| --- | --- | --- |
| target reasoning | `medium` | `/Users/alexis/src/.agents/.configs/project-config.toml` |
| resolved reasoning | `medium` | same canonical config |
| resolved yolo | `false` (explicit, not defaulted) | same canonical config |
| interactive argv | `--provider local-qwen --model <qwen path> --thinking medium` | — |

The plan reports the models.json it would write as sha256
`0b9cc9f4523183c09884603aaaaaf62cb867adcedddd076d1ca11007174dd8c2`. That catalog was
**not** taken on trust: the expected document was reconstructed independently from
the canonical profile and hashed to the same value byte-for-byte, confirming the
catalog Pi consumes carries `"reasoning": true` and
`"compat": {"thinkingFormat": "qwen-chat-template", ...}` — the exact pair the pinned
binary requires. `pgrep` for local model servers exited 1 before and after; nothing
was launched.

## Validation rerun from the exact candidate by this reviewer

| Command | Result |
| --- | --- |
| `gofmt -l tools/agents-infra` | 0 lines |
| `go vet ./...` | ok |
| `go build ./...` | ok |
| `GOOS=windows GOARCH=amd64 go build ./...` | ok |
| `GOOS=linux GOARCH=amd64 go build ./...` | ok |
| `go test ./internal/infra -count=1` | ok, 83.950s |
| `go test . -count=1` | ok, 73.543s |
| `go test ./internal/attachments -count=1` | ok, 0.999s |

`go list ./...` shows exactly those three packages, so the module is fully covered.
The Qwen/Pi acceptance tests reported **0 skips** — the official Pi asset was present,
so `officialPiAsset`'s `t.Skipf` did not silently hollow out the suite. Docs are
pinned executably by `pi_operator_docs_test.go` and `model_check_docs_test.go`, which
now assert the corrected `--approve` wording and the new `yolo_mode`/thinking-format
fragments.

## Non-blocking observation

In `validateResolvedPiTarget`, the `!profile.Reasoning` branch is unreachable in
practice: the new parse gate already rejects `reasoning = false` with a non-`off`
thinking level, and `target.Reasoning != profile.Thinking` is checked earlier. It is
harmless defense-in-depth that fails closed either way, and the corresponding test
correctly asserts the parse-gate error code. No change requested.

## Definition of Done

All Story AC and DoD items are satisfied and independently re-verified above.
Acceptance recorded with `accept_cr`; the element parks at `to-review` for the
orchestrator, which commits its scope and makes the `done` transition with
`commit_ack=scope_committed`. This reviewer supplied no `commit_ack`.
