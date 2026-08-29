# TASK-260828-3fgca3 — revision 5: F6 closed, and the overclaiming corrected

## The finding

Review misspelled `parameters.required` as `parameters.require` in the
`tool_call` scenario. The gate read that as the **supported absence** of
`required`, `requiredArguments` came out `[]`, the parity check in
`BenchmarkScenarios.tool` had no argument key to demand back, a runtime that
called the tool with an empty argument object satisfied it, and the shipped
`benchmark-run` exited **0** with `accepted: true`.

## The fix — the reviewer's recommendation, nothing wider

* `function.parameters.required` is **required**. `[]` means "this tool
  deliberately takes no mandatory arguments"; it just has to be written down.
* `function.parameters` is required and must be an object.
* `type` must be present and equal to `"function"`.
* `tool.parameters.required` removed from `PromptSuiteSchema.supportedAbsences`,
  which now holds **five** entries (pinned by a test).
* `requiredNonEmptyStrings` became `requiredStrings(emptyMeans:)`, so
  `required: []` is legal and `turns: []` is still a fault.

**No allowlist over arbitrary JSON Schema.** The parameter block is an opaque
document this gate forwards to the runtime verbatim.

## The claim, and where it stops

| level | disposition |
| --- | --- |
| document object | unknown keys refused |
| each scenario object | unknown keys refused, kind-scoped |
| tool declaration | exactly `type`, `function`, `function.name`, `function.parameters`, `function.parameters.required` validated (`PromptSuiteSchema.validatedToolFields`, pinned) |
| everything below | **not validated, by design** — forwarded verbatim. `PromptSuiteSchema.unvalidatedByDesign` names it, pinned |

**Residual, stated:** a misspelling inside `parameters` that is not one of the
five named fields still reaches the runtime unremarked. `required` is the only
key in there this gate reads, which is why it is the only one made mandatory.

`RuntimeBenchmark.unpinnableConditions` untouched:
`["kv=unread", "prefill-step=unpinned", "reasoning=unpinned"]`.

## Corrections to revision 4's claims

* LOGBOOK 0135 AUDIT bullet ("All refuse now") — correction bullet added in place.
* Report §0.3 table rows `tool parameters.required` and "any unknown key, at any
  level" — both marked CORRECTED with what was actually true.
* Report §0.3 supported-absence table: six → five.
* README suite-rules table rewritten with the boundary row.
* `PromptSuiteSchemaTests` header: the "at every level" claim replaced with the
  boundary, and the smoke count corrected (4 → 8 malformed suites, §5 and §6).

## Mutants — 3 new, all killed, zero survivors

| Mutant | What it does | `swift test` | production entry |
| --- | --- | --- | --- |
| **M-F6-1** | F6 undone: an absent `required` returns `[]` without a fault again | exit 1, 2 red | exit 1 — the `parameters.require` suite is **measured and accepted, exit 0, `"accepted" : true`**, `tool_call` `succeeded: true` in both records |
| **M-F6-2** *(narrowing)* | explicit `"required": []` becomes a fault | exit 1, 1 red | exit 1 — the empty-demand pass refused, **exit 5** |
| **M-F6-3** *(narrowing)* | rev4's completeness promise: full allowlist over the declaration and function objects | exit 1, 2 red incl. the shipped-suite positive | exit 1 — **the smoke's own control suite refused, exit 5** (its tool carries `description`); nothing downstream runs |

M11 (`modelDigest` back into `Pins.firstMismatch`) and M12 (any `/slots` answer
read as speculating) re-run against this source: exit 1 with **18** and **19**
issues, both killed. Counts not comparable to revision 4's 18/16 — both were
re-authored from their descriptions and the suite grew by six tests.

## Smoke section 6 (production entry, shipped `benchmark-run`)

```
PASS  a misspelled parameters.require is refused by name before any launch (exit 5)
PASS  a misspelled parameters.require emitted no decision at all
PASS  a misspelled parameters.require started no runtime (no session directory)
PASS  a tool declaring no required array at all is refused ... (exit 5) / no decision / no session dir
PASS  a non-function tool declaration is refused ... (exit 5) / no decision / no session dir
PASS  a tool that deliberately requires no arguments is measured and accepted
PASS  both records show tool_call actually succeeding under an empty demand list
```

## Gates — each run directly as its own process

| Gate | Command | Result |
| --- | --- | --- |
| package build | `swift build --build-tests` | exit **0** |
| contract suite | `swift test` | **385 tests / 30 suites**, exit **0** |
| production-entry smoke | `scripts/benchmark-gate-smoke.sh` vs Release product | **95 checks, 0 failures**, exit **0** |
| Release product | `swift build -c release` | exit **0** |
| Swift lint | `xcrun swift-format lint --strict --recursive Sources Tests` | exit **0** |
| shell lint | `shellcheck scripts/benchmark-gate-smoke.sh` | exit **0** |
| whitespace | `git diff --check` | exit **0** |

**Not re-run:** revision 3's real 29 GB cross-format pair through
`benchmark-run`. The 28 GB model was never loaded. This delta changes only tool-
declaration validation; the part of that run it could have broken — the shipped
suite's `tool_call` declaration passing validation — is asserted at the
production entry in smoke §5 and §6.

## Files

* `Sources/MLXSwiftRuntimeContract/PromptSuiteSchema.swift`
* `Tests/MLXSwiftRuntimeContractTests/PromptSuiteSchemaTests.swift`
* `scripts/benchmark-gate-smoke.sh` (new section 6)
* `README.md`
* `.research/260828_llamacpp-in-the-benchmark-gate.md` (revision 5: §0F6, §5.3, §6, §7.0F6, corrections)
* `LOGBOOK.md` (entry 0310, plus correction into 0135)

## Addendum — prior revisions' bound mutants re-run against THIS source

Not cited from earlier rounds; re-applied, built, run, reverted here.

| Mutant | Origin | `swift test` | production entry |
| --- | --- | --- | --- |
| **A8** *(narrowing)* | rev3 | exit 1, 4 red | — |
| **A9** (the relaxation this task was told not to make) | rev3 | exit 1, 3 red incl. the test asserting the list whole | — |
| **M-F5-1** | rev4 | exit 1, 15 red | exit 1 — `"prefix_repeats": "2027"` again **measured and accepted, exit 0** |
| **M-F5-2** *(narrowing)* | rev4 | exit 1, 3 red | — |
| **M-F5-3** *(narrowing)* | rev4 | exit 1, 10 red | — |
| **M11** *(narrowing)* | rev2 | exit 1, 18 issues | — |
| **M12** *(narrowing)* | rev2 | exit 1, 19 issues | — |

Total this session: **10 mutants, 10 killed, 0 survivors.**
`unpinnableConditions` is byte-identical to revision 4's:
`["kv=unread", "prefill-step=unpinned", "reasoning=unpinned"]`.

## Smoke flakiness — reported, not smoothed over

The smoke ran **six times** in this session and went red **twice**, both times
identically: the control pass fails with
`the attestation for "gate-smoke-candidate" was opened and never closed`,
cascading into ~15 downstream FAILs. That is the exec-stub `ProcessObservation`
anomaly first reported in revision 3 and still undecided — `/opt/homebrew/bin/
python3` re-execs and the executable path compared at open and at close differs.
It **fails closed**, it is unrelated to this delta (nothing here touches
observation), and it clears on retry. The 95-checks/0-failures result above is a
run that went green; two of the six did not. On the real 28 GB run a retry costs
an hour rather than two minutes, so this still wants a decision.
