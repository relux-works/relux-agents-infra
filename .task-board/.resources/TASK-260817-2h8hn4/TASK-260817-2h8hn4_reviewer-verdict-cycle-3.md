# Reviewer Verdict — Cycle 3

Task: `TASK-260817-2h8hn4`
Verdict: changes requested
Route: `analysis`

## Finding

### F1 — Managed argv safety is not bound to the Pi parser identity it proves

The revised bridge correctly identifies that Pi commit `a1bc0ec79010887210cc7de28714d72c78577dab` has no end-of-options state, strips the wrapper delimiter, normalizes the recognized equal forms, rejects unsafe suffix operands, and requires production-entry narrowing evidence. The contract nevertheless resolves and launches an arbitrary `pi` executable. It defines no supported Pi version/build identity, no parser-compatibility catalog selected from a verified identity, and no mismatch/unknown refusal before the managed runtime starts.

This is a `capability claim that does not reproduce` and a `bypass path around the check`: all parser-dependent guarantees are established for one source snapshot, while the production path admits another binary whose option arities, aliases, equal forms, delimiter handling, or extension-flag consumption may differ. Section 5 says that updating the supported Pi version requires rerunning conformance, but no supported version exists in the schema or launch contract and no production call site enforces it.

Observed evidence:

- The pinned source identifies `@earendil-works/pi-coding-agent` version `0.84.2` and implements the parser behavior described by the artifact.
- The current official `main` resolved from the task-linked documentation is commit `df018b6020181d4245575fba006361ab69a1408b`, while the artifact pins `a1bc0ec79010887210cc7de28714d72c78577dab`; both currently advertise `0.84.2`, so the version string alone is not a commit/build identity.
- The proposed diagnostics only resolve the Pi executable, and the acceptance scenarios use fake Pi executables without a compatibility-mismatch case.

## Required rework

Define one deterministic production contract before implementation:

1. Bind managed launch to an exact authoritative Pi identity whose parser grammar is the one modeled by the bridge, or define a deterministic generated compatibility catalog keyed by an authoritative executable identity.
2. Specify how direct launch establishes that identity before lock acquisition, file creation, and runtime start; distinguish absent identity, failed/malformed identity reads, unsupported identity, and verified identity. Do not infer compatibility from PATH presence or a non-unique version string.
3. Keep `--print-config` and `compose` non-launching: report the configured/expected compatibility identity and runtime verification state as `unknown` when the executable cannot be authoritatively inspected without execution; `unknown` must not authorize managed launch.
4. Add real-entry negative cases for unsupported, malformed, and spoofed/non-authoritative identity plus a narrowing mutant that accepts the same version string from a different build. The runtime and Pi session must not start on any mismatch or unknown state.
5. Update the exact TOML/diagnostic/error contract and rejected alternatives accordingly. If executable hashing or a signed/package-manager identity is chosen, specify update/relocation semantics and avoid a self-minted repository value authorizing itself.

## Checks completed

- Re-read the complete decision artifact and linked story plan outcome.
- Verified official Pi custom-model, settings, usage, trust, session-directory, and CLI option documentation.
- Inspected pinned production `parseArgs`, model resolution, agent-directory, and settings merge source.
- Reproduced upstream identity drift with `git ls-remote`; compared pinned/current parser and package metadata.
- Confirmed managed identity, DFlash authoritative attestation, separator-lookalike, absent/malformed evidence, production-call-site, and narrowing scenarios otherwise remain explicit.
- Confirmed three-task dependency chain and task traceability are proportional and complete.
- `task-board validate`: pass.
- `git diff --check`: pass.
- Decision artifact/resource SHA-256 parity: pass (`b1cb5ed759b7a3667b2ab190dbe0a7558fd70f9dfc61ef0e0dbfce8fadbc7bcf`).

No product build/test applies to this no-implementation research task. The verdict is ordinary analysis rework, not a stop-the-line blocker.
