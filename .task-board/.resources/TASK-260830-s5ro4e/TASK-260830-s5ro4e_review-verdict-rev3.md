# TASK-260830-s5ro4e review verdict — revision 3

Verdict: **changes requested**. Route to `analysis` for a corrected executable
release/rollback plan and a new Change Request revision.

Reviewed candidate: `CR-TASK-260830-s5ro4e-3` revision 3, base
`5c9b4e4f7a88e1eb937b80851af522e4fa4b066f`, candidate tree
`fed6636c9327a8434a5699eb01db0699e4fdd16f`, patch SHA-256
`5b731286c737b98b6738b6830dcb6469fbef2f3fae0dc2bfc333cb63a769d645`.

## Findings

### F1 — High: Step-3/Step-6 LLDB guards accept failed reads

The plan says every failed before/after snapshot must fail the step, but the
executable blocks at lines 493-499 and 637-643 do not enable `errexit` and do
not capture or test either snapshot status or the `cmp` status. Their last
command is only `test "$INSTALL_EXIT" -eq 0`.

An exact-shape zsh attack made both snapshot calls write the same partial line
and return 7. `cmp` then returned 0 and the final install test returned 0, so
the complete block returned 0:

```text
before_exit=7 after_exit=7 cmp_exit=0 install_exit=0 final_block_exit=0
outer_exit=0
```

This is the negative shape **failed read treated as satisfied** (and therefore
a bypass around the identity gate). The attached revision-3 positive probe
only proves the helper succeeds on a readable host; it does not kill this
negative shape. A read can fail after emitting identical partial output and the
plan will admit the installed surface despite its own fail-closed prose.

Required rework:

1. Make each complete Step-3/Step-6 guard one explicitly fail-fast command
   (`set -euo pipefail`) or explicitly retain and require zero for the before
   snapshot, installer, after snapshot, and `cmp`; do not rely on the caller's
   shell options.
2. Prefer writing each snapshot to a temporary file and publishing it only
   after the snapshot command succeeds, so partial evidence cannot look
   complete.
3. Add the negative probe above: identical partial bytes plus non-zero snapshot
   status must make the production plan-shaped block non-zero and must withhold
   default-PATH composition.

### F2 — High: the recovery baseline no longer matches the installed pair

The plan identifies the current agents-infra baseline as
`v1.6.1-100-gd69a435` and creates its recovery source worktree at exact commit
`d69a435945758ea1cd5dfa62395ca32498e712c7` (lines 292-294). The installed
binary measured during this review is instead:

```text
agents-infra v1.6.1-103-g4270549 commit=4270549
```

The snapshot loop copies that installed `-103` executable, while the hard-coded
source worktree remains `-100`. `restore_agents_infra_surface` then runs the
saved `-103` binary against the `-100` source. The gap contains commits
`573d9b4` and `148145c` and changes the managed global instruction surface
(`.instructions/INSTRUCTIONS_WORKFLOW.md`) as well as repository documentation
and tests. The result is not the exact previous working pair claimed by the
rollback.

This is the negative shape **capability claim that does not reproduce**: the
named current/recovery version is stale, and binary/source coherence is
inferred rather than checked.

Required rework:

1. Refresh Step 0, the compatibility matrix, recovery source OID, and later
   references to the exact current installed pair.
2. Add an executable snapshot identity gate that parses the saved executable's
   reported commit and requires it to equal the detached recovery source HEAD;
   failed or unparseable identity is `unknown` and blocks Step 2.
3. Re-run the current-pair spawn/route/resource/CR canary and relevant global
   setup/verify gates against that coherent exact pair. Do not call an older
   source plus newer binary a rollback baseline even if their current production
   Go diff happens to be small.

## Independent checks that passed

- The CR patch resource hash matches the handed SHA-256. The exact delta is
  limited to the release plan and `LOGBOOK.md`; `git diff --check` exits 0.
- The board plan resource is byte-identical to the candidate plan (SHA-256
  `7963d1da078f738ee7c1d16de93a97e97233ca5b95829863824b0e0cf638a872`).
- Revision 3 correctly mandates `AGENTS_INFRA_SKIP_LLDB_MCP=1`, and production
  `scripts/setup.sh:93-96,258-263` confirms the skip returns before the
  Homebrew mutation call.
- The revised `LOGBOOK.md` replaces the old 1245 false census and records the
  corrected 25/36 direct census, 20-package graph, 555 linked symbols, and
  named mixed windows.
- The revision-2 independent review already reproduced the four distinct
  task-board v0.5.0 consumption instruments and the unmodified spawn package
  gate. Revision 3 does not weaken that evidence.
- Operational rollbacks remain explicitly labelled **UNTESTED**. No release,
  tag, migration, Homebrew change, or installed-runtime mutation was performed
  by this review.

## Review evidence

- Guard attack: `.temp/TASK-260830-s5ro4e-review/guard-attack-02.log`.
- Installed baseline/commit gap:
  `.temp/TASK-260830-s5ro4e-review/baseline-drift.log`.
- Candidate/resource identities and cited lines:
  `.temp/TASK-260830-s5ro4e-review/identity-and-lines.log`.

