# Reviewer Verdict — Cycle 8

Task: `TASK-260817-2h8hn4`

Verdict: **changes requested**

Route: `analysis`

## Scope reviewed

- Cycle-8 decision outcome and `.research` source.
- Cycle-8 mandatory practical trust-boundary directive.
- Identical downstream precondition copies on `TASK-260817-ccpnlm` and `TASK-260817-3a0zr3`.
- Official Pi custom-model, settings, usage, parser, configuration, and v0.84.2 release evidence.
- Story decomposition, dependencies, downstream descriptions/scopes/acceptance criteria, task resources, and board validation.

## Passing evidence

- The outcome, `.research` source, and both downstream copies are byte-identical at SHA-256 `ef14211dacbc627260c0e4dfc40cc4eea87b04d01c1d5ddf523f75628624a47b`.
- The three official pinned Pi documents reproduce the hashes recorded in section 3: models `3ab68dd...`, settings `f36d3a91...`, usage `a6a76e73...`.
- Official Pi docs support the recorded `models.json`, auth visibility, project-settings merge, trust flags, native model options, and session-directory facts.
- Pinned production `parseArgs()` confirms that literal `--` is not an end-of-options marker and that recognized provider/model/API-key/thinking options use spaced values; recognized-looking equals forms fall through to unknown flags.
- Official v0.84.2 `SHA256SUMS` confirms `pi-darwin-arm64.tar.gz` SHA-256 `c996e888b7f7dce44bcf24f69176ac646c44139d3916bd49a6b28e5a8c5e3a65`.
- Cycle 8 correctly removes the rejected backend catalog/observer/proxy/runtime-attestation design, keeps both requested profiles launchable, and labels runtime/DFlash claims as requested or configured-unverified.
- The three-task dependency chain remains proportional and complete. `task-board validate` and `git diff --check` pass.

## Finding F1 — managed Pi tree identity is no longer reproducible

Severity: high

Negative shape: **capability claim that does not reproduce**.

The current authoritative cycle-8 artifact retains:

- regular-file count `217`;
- one opaque canonical tree-manifest SHA-256;
- the requirement to verify the “complete compiled-catalog release tree” before side effects and again before Pi spawn.

It does not define or attach the canonical catalog that makes this check deterministic: release-root selection, complete path inventory, byte ordering, manifest record encoding, allowed entry types, path/name normalization and rejection rules, executable-mode treatment, or the exact extra/missing/symlink/hard-link policy. The downstream task-scoped copies have the same omission, and no task-scoped outcome contains the 217-record manifest.

This is a regression from the earlier cycle-5 evidence still visible only in non-authoritative `.temp` material. That version specified bytewise `C`-locale path order and records of `<lowercase-sha256><two spaces>./<relative-path><LF>`, plus explicit entry-type and path rules. Its 217-record manifest hashes to the cycle-8 value `2f68ab1b...`, proving the recorded digest is plausible but not making the current contract reproducible.

An implementation can now choose a narrower manifest procedure—such as hashing only regular files reached by a locale-dependent walk, omitting extra-entry/type checks, or applying different path normalization—and still claim scenario 14 and the execution-closure gate are implemented. The artifact supplies no authoritative rule against which review can reject that mutant.

## Required rework

1. Restore the complete deterministic Pi release-tree catalog contract in the current cycle-8 decision and both downstream copies. Define the release root, exhaustive inventory, byte ordering, record encoding, path/name rules, entry-type/link policy, executable mode, extra/missing handling, and both initial and point-of-use identity requirements.
2. Persist the authoritative 217-record manifest as a new task-scoped outcome/precondition, or persist an equivalently complete generated-catalog payload plus the exact deterministic generation procedure and its input asset identity. Do not leave required catalog content only under `.temp`.
3. Add a named production-entry/narrowing acceptance case that changes or narrows canonicalization while keeping the official asset checksum and entrypoint hash controls intact; it must fail the execution-closure gate.
4. Re-run outcome/downstream parity, TOML parse, official checksum/source checks, `task-board validate`, and `git diff --check`, then hand the revised artifact to a fresh reviewer cycle.

No code implementation or new board element is required for this rework.
