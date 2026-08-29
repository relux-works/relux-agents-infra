# TASK-260828-3fgca3 — mutant log, G4

Eight mutants over the G4 delta, applied to the shipped source, built, run and
reverted by `.temp/TASK-260828-3fgca3/mutant.sh`. Patches are `m9.py`..`m16.py`
in the same directory. `swift test` is the 324-test contract suite; the smoke is
`scripts/benchmark-gate-smoke.sh`, 59 checks driving the shipped subcommands.

**8/8 killed, 0 survivors.** M1-M8 (G1/G2) were run in the previous revision and
are recorded in the research note.

| Mutant | Kind | What it does | `swift test` | smoke |
| --- | --- | --- | --- | --- |
| M9  | widening | the cross-format arm returns before checking anything the verdict says | exit 1, 12 red | — |
| M10 | widening, call site | `equivalenceReading` returns `.noneDeclared` on a failed read | **exit 0 — blind** | exit 1, 1 FAIL |
| M11 | **narrowing** | `modelDigest` restored to `firstMismatch`, so no cross-format pair can ever be admitted | exit 1, **22 red** | — |
| M12 | **narrowing** | every `reported` speculation state read as `on`, so llama.cpp itself becomes inadmissible | exit 1, **20 red** | — |
| M13 | widening, call site | the driver stops asking `GET /slots` | **exit 0 — blind** | exit 1 — **false acceptance, exit 0** |
| M14 | widening, call site | the driver stops copying the verdict's non-equivalences into the records | **exit 0 — blind** | exit 1, 3 FAIL |
| M15 | widening | `speculation=unread` spent as `off` | exit 1, 2 red | — |
| M16 | widening | the record may declare its own `modelOfRecord` | exit 1, 4 red | — |

## The three that carry the argument

**M13 — the acceptance question for the MTP condition, answered at the
production entry.** With the `/slots` read deleted from
`BenchmarkRunCommand.servingAnswer`, all 324 contract tests still pass — they
hand the reading in directly — and the smoke's speculating pair, two real
spawned processes both reporting `params.speculative: true`, driven and measured
and judged by the shipped `benchmark-run`, comes out:

```
FAIL  a runtime reporting speculative decoding is refused: expected exit 4, got 0
```

Exit 0 is `accepted=true`. Two runtimes drafting, scored against each other as
if the difference were the runtime.

**M11 and M12 — the narrowing pair.** Both make the gate *stricter* and both
redden the tests that pin the *admitted* class rather than the refused one:
M11 breaks `admitsACrossFormatPairUnderEvidence` and the smoke's G4 acceptance;
M12 breaks `admitsARuntimeThatReportsNoSpeculation`,
`admitsAnExplicitlyDisabledSpeculation` and the smoke's `--slots false` case. A
delete-only mutant would have shown the clauses exist and said nothing about
what they cover.

## Logs

`mutant-m9..m16-{build,test,smoke}.log` in this directory.
