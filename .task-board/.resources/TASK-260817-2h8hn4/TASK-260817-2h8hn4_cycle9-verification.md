# Cycle 9 Verification Evidence

Task: `TASK-260817-2h8hn4`

## Reviewer finding closed

Cycle 8 retained an opaque Pi release-tree digest without the canonical catalog payload and generation rules. Cycle 9 restores the exact 217-record manifest, release-root selection, bytewise ordering, record encoding, exhaustive path inventory, entry-type/link policy, permission map, initial check, point-of-use recheck, and a named production-entry narrowing scenario.

## Official source evidence

- Pi docs/source revision: `10acee6045e9025a22dff7e5220ed0d7538f12aa`.
- `models.md`: `3ab68dd46af081d3a11a2d705048f2fbde93a87e29891c677191e24cec2840f3`.
- `settings.md`: `f36d3a918d87d18e13d22ca98f4e429428b5f2bc06316ff7d2e7adace59a973b`.
- `usage.md`: `a6a76e733c50ea8a08701456858d3937e1d60caa8709a7f15125e6f5ba6cabca`.
- Official Pi v0.84.2 darwin-arm64 asset checksum reproduced from release `SHA256SUMS`: `c996e888b7f7dce44bcf24f69176ac646c44139d3916bd49a6b28e5a8c5e3a65`.

## Catalog reproduction

- Independently extracted the official asset and regenerated the canonical manifest.
- Exact record count: `217`.
- Exact manifest SHA-256: `2f68ab1b3f28a9c4b8995f91984f8f47001a79735da7e57aa7fe6d223f90378b`.
- Exact tree: 34 directories including root, 217 regular files, no symlinks or other entry types, every file link count 1.
- Exact permission map: all directories `0755`; four catalogued executable files `0755`; remaining 213 regular files `0644`.
- Regenerated manifest is byte-identical to all three materialized board copies.

## Artifact and board evidence

- Decision source and all four materialized decision resources are byte-identical at SHA-256 `be8c3482ee78685f6b65d5bfa9802893a1666a032bca8df834fa7935688793d3`.
- The implementation and operator-documentation tasks cite the exact manifest and deterministic catalog contract in description, scope, AC, and precondition resources.
- Dependency chain remains `TASK-260817-2h8hn4` -> `TASK-260817-ccpnlm` -> `TASK-260817-3a0zr3`; no additional task or diagram is justified.
- TOML example parses with both profiles.
- `task-board validate`: pass, no issues.
- `git diff --check`: pass.
- Product build/tests are not applicable because this task is architecture research with explicit `No implementation` scope.
