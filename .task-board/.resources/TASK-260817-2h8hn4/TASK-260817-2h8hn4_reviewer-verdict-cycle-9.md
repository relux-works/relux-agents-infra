# Reviewer Verdict — Cycle 9

Task: `TASK-260817-2h8hn4`
Verdict: changes requested
Route: `analysis`

## Finding

### F1 — Raw profile name escapes or aliases the managed state root

Severity: high
Negative shape: bypass path around the check

The contract validates profile names only as exact non-empty TOML keys (section 4.1), then derives managed state as:

`os.UserCacheDir()/agents-infra/pi/<sha256(canonical-project)>/<profile>/`

The profile is therefore both a configuration identity and an unchecked filesystem path component. A quoted TOML profile such as `../../../../../../.pi/agent` is valid under the written schema and normalizes outside the project/profile cache root. Distinct accepted names such as `qwen`, `./qwen`, and `nested/../qwen` also normalize to the same directory, so they share `models.json`, sessions, and `session.lock` despite the contract saying profiles are independently locked.

This defeats the managed-state isolation and no-unintended-write boundary before runtime or Pi launch. The existing provider-name separator gate and Pi release-tree path canonicalization do not cover profile names. Section 12 has no traversal, separator, containment, or normalized-path collision scenario for this state derivation.

Evidence:

- `.temp/TASK-260817-2h8hn4/review9-profile-path-attack.log` parses the traversal key with Python `tomllib`, demonstrates `contained=False`, and demonstrates the `qwen` / `nested/../qwen` collision.
- `.research/260817_pi-local-model-launch-contract.md:145` permits every non-empty profile key.
- `.research/260817_pi-local-model-launch-contract.md:245` inserts the raw profile into the managed state path.

Required rework:

1. Define one exact safe state-key derivation that is injective for accepted profile names and cannot escape the canonical cache root. A fixed lowercase SHA-256 of the exact UTF-8 profile bytes is the simplest compatible choice; alternatively define and validate a deliberately narrow profile-name alphabet.
2. Specify containment and collision checks before lock acquisition or file creation, including failed/partial path resolution as an error rather than absence.
3. Add production-entry negative cases for `/`, `\\`, `.`, `..`, traversal components, absolute-looking names, Unicode separator/lookalike forms, and normalization/case variants relevant to the supported filesystem.
4. Add a narrowing mutant that replaces the safe state key with the raw profile or a lossy normalized/sanitized name and require a named test to fail.
5. Propagate the corrected state-key/path contract and acceptance cases to both downstream tasks.

## Evidence that passed

- Official Pi `models.md`, `settings.md`, `usage.md`, and `args.ts` at revision `10acee6045e9025a22dff7e5220ed0d7538f12aa` were fetched again; the three documented SHA-256 values match the decision artifact.
- The official v0.84.2 darwin-arm64 release checksum matches `c996e888...`.
- Independent extraction reproduces the attached 217-record manifest byte-for-byte at SHA-256 `2f68ab1b...`; the tree has 34 directories, 217 regular files, no symlinks/other entries, link count one, and the specified permission map.
- Decision source and board outcome are byte-identical at SHA-256 `be8c3482...`.
- Current Story/task dependency chain is decision -> implementation -> alias/documentation; `task-board validate` and `git diff --check` pass.
- Cycle-8 practical trusted-runtime boundary, requested/unverified capability reporting, no fake Pi separator, exact identity mismatch cases, and cycle-9 canonicalization-narrowing scenarios remain present.

No code/build verdict applies because this task's scope is explicitly architecture research with no implementation.
