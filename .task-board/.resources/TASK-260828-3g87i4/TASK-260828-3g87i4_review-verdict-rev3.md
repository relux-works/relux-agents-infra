# TASK-260828-3g87i4 — Change Request revision 3 review verdict

Verdict: **ACCEPTED**.

Review scope was the only finding left open after revision 2: F1, whether the
negative suite witnesses the production CLI decision rather than merely finding
source text. F2–F5 remain accepted and were not reopened, as required by the
Round 3 brief.

## Candidate integrity

- Change Request: `CR-TASK-260828-3g87i4-3`, revision 3.
- Base OID: `132c5997f9ad8a82358d03d7a08a23eff46bcf9d`.
- Candidate tree OID: `2e6b2884e069169838461a815a206ed8a4f7d1a7`.
- The board-materialized patch has SHA-256
  `edda55429904a6be23944922cbb8b9e4cb7bf68bede167a262481ca4c4c33fa1`
  and is byte-identical to an independent `git diff --binary` of those OIDs.
- The repository delta is one path, `LOGBOOK.md`, adding the Round 3 evidence;
  `git diff --check` passes. The task's executable analysis and test artifacts
  remain task-scoped board resources, as in the already-accepted prior rounds.
- The exact board-materialized artifacts reviewed were:
  - `quant_equivalence.py` SHA-256
    `273a60bf69b72ed1e10709804d17e41b7262d4af2c34a863a1218364e69e6373`;
  - `test_alignment_guard.py` SHA-256
    `f8dc40b29ef059b84cb5b9abc1e2ac7d9393b638332129b6a8aa8b52d9396412`.

## Production-path negative evidence

Production call site: `quant_equivalence.main()` obtains `(rows, skipped)` from
`collect_rows()`, invokes `comparability_verdict(rows)`, prints the resulting
verdict, and selects exit 0/1/2. The test replaces only the row-collection seam,
runs that real `main()`, and asserts both emitted verdict and exit status.

Independent baseline run on the exact board artifacts:

- exit `0`;
- `16 checks, 0 failure(s)`;
- FP8 ratio `3.889` was reported as `NOT COMPARABLE`, exit `1`;
- unreadable input was reported as `INCOMPLETE`, exit `2`.

I then copied the same two artifacts into two separate reviewer scratch
directories. In both, `test_alignment_guard.py` remained byte-identical to the
baseline (`f8dc40b2…`). Only `quant_equivalence.py` was mutated.

1. **Dead-call call-site bypass.** I retained a syntactic
   `if False: comparability_verdict(rows)` and made the live CLI path decide
   inline at `RATIO_CEIL + 1.0`. The unchanged suite exited `1`. Its production
   Gate 2 assertion failed because `main()` emitted `COMPARABLE`, exit `0`, for
   the FP8 row where the test required `NOT COMPARABLE`, exit `1`. The mixed
   one-FP8-among-good-rows case failed for the same reason. This kills the exact
   `check present but uncalled from production` / bypass shape left open in
   revision 2.

2. **Threshold narrowing mutant.** I changed only `RATIO_CEIL = 3.0` to `4.0`.
   The same unchanged suite exited `1`, again failing the production Gate 2 FP8
   assertion because `main()` emitted `COMPARABLE`, exit `0`. Thus the new
   production-path binding did not trade away the original threshold bound.

The external mutant runs also report their corresponding in-suite mutation site
as already changed (or the double-bypass mutant as unloadable). Those additional
self-harness failures are expected after pre-mutating the production artifact;
the acceptance evidence is the independently observed production Gate 2 failure
in each run.

## Count and lint checks

- The equivalence report claims `16 checks, 0 failures`; the independent baseline
  suite prints exactly `16 checks, 0 failure(s)`.
- Ruff passes on the producer-mode (`0755`) copies of the exact reviewed bytes.
  A first Ruff attempt on `task-board resource get` materializations reported only
  `EXE001`, because resource download materialized the shebang-bearing files as
  `0644`; hashes confirmed the content was identical. This is a resource transport
  mode effect, not a source lint defect, and both logs are attached.

## Conclusion

F1 is closed. The negative suite now observes the production decision through
`main()` and fails under both the reviewer's dead-call bypass and the widened
ratio threshold. Together with the F2–F5 acceptance from revision 2, the staged
artifacts and equivalence judgement satisfy this task's acceptance criteria.

