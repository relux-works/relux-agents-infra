# TASK-260825-39ycg2 Developer Rework Results

## Reviewer finding closed

The reviewer proved that both new bounded model-check documentation sections
could be deleted while the uncached Go suite remained green. This rework adds
`tools/agents-infra/model_check_docs_test.go` with exact-fragment contracts for
the README and installed `relux-agents-infra` skill. The tests pin:

- exact command shape and repeatable expectations;
- the four artifacts, `0700` directory mode, `0600` file mode, and overwrite
  refusal;
- the default/minimum/maximum deadline, cleanup, retained evidence, and all six
  ordered exit outcomes, including exit-5 precedence;
- unattended Pi `--approve` behavior;
- the distinction between a read-tool observation and proof of the exact file,
  including failed/partial read versus legitimate absence;
- the skill-to-README `#bounded-model-behavior-checks` link.

`internal/infra/model_check.go` now names the existing minimum deadline and
success exit as exported production constants. The documentation tests derive
deadline and exit fragments from those constants; runtime behavior is
unchanged.

## Negative mutation evidence

Each mutant used `-count=1`, ran the named production doc-contract test without
a pipe, restored from a task-scoped byte copy in an `EXIT` trap, and was
followed by `cmp -s`.

| Mutant | Command outcome | Proof |
| --- | ---: | --- |
| README exit `5` changed from “takes precedence over unmet expectations” to “follows unmet expectations” | expected red exit `1` | `TestBoundedModelCheckREADMEContractPinsSafetyAndExitSemantics` failed on the exact missing precedence row; restore `cmp -s` exit `0`. |
| Skill failed-read rule changed from “failure or unknown, never absence” to a permissive absence fallback | expected red exit `1` | `TestReluxAgentsInfraSkillPinsBoundedModelCheckerWorkflow` failed on the exact missing evidence rule; restore `cmp -s` exit `0`. |

The unmutated pair was rerun after restoration and exited `0`.

## Validation run by this developer

| Command | Exit | Result |
| --- | ---: | --- |
| `go test . -run '^(TestPiOperatorContractDocumentsCycle10Boundary\|TestReluxAgentsInfraSkillRoutesSafePiWorkflowToSource\|TestBoundedModelCheckREADMEContractPinsSafetyAndExitSemantics\|TestReluxAgentsInfraSkillPinsBoundedModelCheckerWorkflow\|TestModelCheckProductionEntrypoint)$' -count=1` | `0` | Focused operator-doc and production-entrypoint contracts passed in `15.296s`. |
| `go test ./internal/infra -run '^TestModelCheckCleanupAttestationRefusesUnconfirmedStates$' -count=1` | `0` | Cleanup refusal contract passed in `0.549s`. |
| `go test ./... -count=1` | `0` | All three packages passed uncached (`176.335s`, `3.534s`, `216.942s`). |
| `go vet ./...` | `0` | No findings. |
| `go build ./...` | `0` | All packages compiled. |
| `git diff --check -- README.md SKILL.md LOGBOOK.md tools/agents-infra/internal/infra/model_check.go tools/agents-infra/model_check_docs_test.go` | `0` | No whitespace errors. |
| `./setup.sh` | `0` | Built and installed the candidate CLI and verified global setup. |
| `agents-infra verify global` | `0` | Installed global runtime verified. |
| `agents-infra version` | `0` | Installed binary reports `v1.6.1-28-gac759d9`, commit `ac759d9`. |
| `qwen-infra --print-config` | `0` | Canonical entrypoint resolves to a managed Pi Qwen profile, provider `local-qwen`, reasoning `off`, and the reviewed loopback endpoint. Host-specific paths are omitted. |
| `cmp -s SKILL.md $HOME/.agents/skills/relux-agents-infra/SKILL.md` | `0` | Installed skill is byte-identical to source. |

## Existing real Qwen smoke: accepted versus rerun

The five-minute Qwen smoke was **not rerun in this rework**. Reviewer
`RUN-260825-f75a58` explicitly marked the existing real smoke as independently
re-derived and instructed the rework not to repeat it. This developer accepted
that already-attached raw-event correlation and reran only the non-provider
integrity checks below.

The existing smoke was launched through installed `agents-infra model-check`
against `qwen-infra`. Its already-attached sanitized correlation establishes a
completed, non-error `read` of the installed
`$HOME/.agents/skills/relux-agents-infra/SKILL.md`. The current rework reran
SHA-256 calculation for all four local artifacts; every digest still matches
the attached results and reviewer verdict:

```text
events.jsonl  f089f23d617c670b3c4af91a2cbd11a07f0cf1e2fd1003ecee995720c76bcbc6
stderr.log    a9da0b2d7bf5325258199604cc00f0ef40fbaa82990fcb99e012fa32efa4055f
summary.json  c320bf117c109ef4c83ccef265d785dc0f097a08246fcbfcf65ca3500ba60568
summary.txt   89286380f2acffb1555a33fbb4db368a4aa672d7178cbce0521624a2cc696a49
```

Mode checks also reran: the evidence directory is `0700`; all four artifacts
are `0600`. The sanitized summary still reports:

- `status=timed_out`, checker exit `2`, `300000ms` deadline, `300192ms`
  duration;
- valid but incomplete event stream;
- `read` completed and non-failed, with the tool expectation met;
- final marker expectation unmet and no proven final-response behavior;
- managed cleanup true with both process-group cleanup states confirmed.

Accurate conclusion: skill discovery and the installed skill read are proven by
the previously reviewed raw-event correlation. The overall smoke failed on the
five-minute timeout, and final-response behavior remains unproven.

Raw `events.jsonl` and `stderr.log` remain local mode-`0600` evidence and are
not attached. This resource contains only sanitized facts and stable hashes.
