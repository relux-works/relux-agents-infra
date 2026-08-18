# Cycle 10 Verification

Task: `TASK-260817-2h8hn4`
Review finding addressed: cycle-9 F1, raw profile name escapes or aliases the managed state root.

## Contract correction

- Profile names remain exact decoded TOML identities.
- `profile_state_key` is the lowercase 64-hex SHA-256 of the exact UTF-8 profile bytes; no normalization, case folding, cleaning, separator replacement, or lossy sanitization occurs before hashing.
- Managed state is exactly `<canonical-cache-root>/agents-infra/pi/<project_state_key>/<profile_state_key>/`.
- The production entry must use anchored no-follow managed-component operations, prove lexical and physical containment, reject byte-distinct hash collisions, and distinguish absence from failed/partial reads before lock, file, socket, or process side effects.
- `TestPiLaunchProfileStateKeyIsolation` covers slash, backslash, dot, traversal, absolute-looking, Unicode lookalike, case, and NFC/NFD names plus collision, symlink, type, read, and revalidation failures. Raw-name and lossy-sanitizer narrowing mutants must fail the named test.
- New fail-closed codes are `profile_state_key_collision` and `profile_state_path_invalid`.

## Official Pi verification

The task-linked documents were fetched from the official redirected repository at pinned revision `10acee6045e9025a22dff7e5220ed0d7538f12aa` on 2026-08-17.

| Document | SHA-256 |
| --- | --- |
| `models.md` | `3ab68dd46af081d3a11a2d705048f2fbde93a87e29891c677191e24cec2840f3` |
| `settings.md` | `f36d3a918d87d18e13d22ca98f4e429428b5f2bc06316ff7d2e7adace59a973b` |
| `usage.md` | `a6a76e733c50ea8a08701456858d3937e1d60caa8709a7f15125e6f5ba6cabca` |

The hashes match decision section 3. The profile-state correction does not alter Pi-native behavior: custom providers remain `models.json` data; project settings override global settings; native model selection remains `--provider`/`--model`; and `--session-dir` remains documented.

## Board propagation and proportionality

- Updated `TASK-260817-ccpnlm` description, scope, AC, checklist, and both decision preconditions with exact state-key, containment, read-failure, and narrowing requirements.
- Updated `TASK-260817-3a0zr3` description, scope, AC, checklist, and decision precondition with the operator-facing form of the same contract.
- Dependency chain remains `TASK-260817-2h8hn4` -> `TASK-260817-ccpnlm` -> `TASK-260817-3a0zr3`.
- No new story/task/research task or diagram was created. The gap belongs to the existing isolated-state implementation and operator-documentation deliverables.
- Self-verification checked the Story requirements, task scope/AC/DoD, three official Pi documents, downstream task scopes, cycle-8 directive, and all explicit exclusions. The correction introduces no backend catalog, observer, proxy, attestation, acquisition, conversion, benchmark automation, or new runtime product scope.

## Validation evidence

| Check | Result |
| --- | --- |
| Official pinned document hashes | Pass; `.temp/TASK-260817-2h8hn4/official-hashes-cycle10-01.log` |
| TOML example parse | Pass; 2 profiles |
| Adversarial state-key sample | Pass; 13 byte-distinct names produced 13 distinct 64-hex keys |
| Decision/resource parity | Pass; all five copies SHA-256 `b9d92598b5cb92c5d32a434318cfbe056dd37dd1961ba85220b10c785efbfb2d` |
| Markdown trailing whitespace | Pass |
| Required contract terms | Pass |
| `task-board validate` | Pass; no issues |
| `git diff --check` | Pass |

No product build or code test applies: this task explicitly has `No implementation`. Executable production-entry and mutant evidence is required by the downstream implementation task.
