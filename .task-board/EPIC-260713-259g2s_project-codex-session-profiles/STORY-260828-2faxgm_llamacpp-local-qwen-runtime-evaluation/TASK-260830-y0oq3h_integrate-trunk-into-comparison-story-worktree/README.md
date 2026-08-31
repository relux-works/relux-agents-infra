# TASK-260830-y0oq3h: integrate-trunk-into-comparison-story-worktree

## Description
STORY-260828-2faxgm is 48 commits behind trunk and 7 ahead. Its 7 commits hold the llama.cpp harness integration; trunk now holds the bounded-KV baseline landed as STORY-260830-2vrhg1 (story commit db616f37). The corrected comparison rerun needs both, so trunk must be merged into this Story branch before the rerun runs. A trial merge produced 40 conflict hunks across 10 files, about 1856 conflicted lines, including 509 in RuntimeAttestation.swift and 332 in BenchmarkRunCommand.swift. Merging main into this branch is already an established pattern here: commit 5a287e4. The orchestrator preserved the pre-merge state as .temp/TASK-260829-3k4qrc/rev1-rejected-dirty-delta.patch (worktree net, 12 paths, sha256 51be834687c5c770f5e35f2859f6deb16187637b2ef362533d67a0421562a929) and .temp/TASK-260829-3k4qrc/rev1-rejected-INDEX-state.patch (separate staged state, 20 paths, sha256 418dce3c15a64a53fa26d6c48c82a428554fca7ab577d51ffdaca162986175a5), and restored the worktree to byte-identical state after aborting the trial merge. The orchestrator explicitly authorizes this one named merge operation on this Story branch.

## Scope
(define task scope)

## Acceptance Criteria
Trunk is merged into the Story branch with both sides preserved: the llama.cpp harness integration and the bounded-KV attestation remain functional together. Every bypass closed during the bounded-KV review cycles is proven still closed after the merge by re-running its negative mutants, each of which must be red without its guard. The benchmark gate smoke suite passes. No admission clause of the comparative gate is weakened to resolve a conflict. The resolution states, per conflicted file, which side won and why, and names any conflict resolved by combining rather than choosing.
