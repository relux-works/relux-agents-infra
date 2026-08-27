# TASK-260827-2v13w8 revision-3 reviewer validation

Date: 2026-08-28 15:36 MSK

## Candidate identity

- Base OID: `3f313d9175f2ada9b9ab3320ab524c0918f9daac`
- Candidate tree: `11e765482a7513a1d8e7ef8c78f9313c065cb23c`
- Worktree vs candidate before/after review: identical
- Final binary SHA-256: `ba1c7451a7d035ecfd786cfd93510dc2b28da915a30aba02ea755ece21112695`

## Commands and outcomes

1. Initial smoke invocation: failed before gate execution because the reviewer
   accidentally doubled `tools/mlx-swift-runtime-prototype` in `BINARY`. No
   validation result was inferred from this failure.
2. Corrected production smoke:
   `BINARY=$PWD/DerivedData/Build/Products/Release/mlx-swift-runtime-prototype OUT=$PWD/.temp/review-rev3-forged-observed BASELINE_PORT=18871 CANDIDATE_PORT=18872 ./scripts/benchmark-gate-smoke.sh`
   — exit 0 in 7.2 s. Its observed-pair control used two placeholder HTTP
   processes and arbitrary scenario metrics; `ok.log` says `accepted=true`.
3. Real revision-3 comparison with the current final binary and attached
   attestations — exit 3, `accepted=false`, three reported 8k blockers.
4. `swift test` — exit 0, 271 tests / 23 suites.
5. `xcrun swift-format lint --strict --recursive Sources Tests` — exit 0.
6. `shellcheck -S warning scripts/*.sh` — exit 0.
7. `python3 -m py_compile scripts/runtime-benchmark.py` — exit 0.
8. `git diff --check 3f313d9... 11e7654...` — exit 0.

## Evidence interpretation

The green smoke does not close the forgery class. Its positive control is the
forgery: gate-written attestations over placeholder processes, followed by
caller-authored records with invented metrics, accepted by the production
comparison entry point. See the round-3 verdict for the finding and rework.
