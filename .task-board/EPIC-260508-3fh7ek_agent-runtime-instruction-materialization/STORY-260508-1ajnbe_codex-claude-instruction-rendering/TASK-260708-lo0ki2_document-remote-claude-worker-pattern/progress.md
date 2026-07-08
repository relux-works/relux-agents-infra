## Status
done

## Assigned To
codex

## Created
2026-07-08T10:03:44Z

## Last Update
2026-07-08T10:06:01Z

## Blocked By
- (none)

## Blocks
- (none)

## Checklist
- [x] Add a dedicated remote agent worker instruction module with host-agnostic variables and workflow
- [x] Document safe local-to-remote project transfer, remote Claude launch, patch/result return, verification, and cleanup
- [x] Register the module in instruction entrypoints and README module lists
- [x] Run setup/render verification after instruction changes

## Notes
Added .instructions/INSTRUCTIONS_REMOTE_AGENTS.md with host-agnostic remote worker guidance. Registered it in .instructions/INSTRUCTIONS.md and .instructions/AGENTS.md, updated README module/tool docs, ran ./setup.sh, verified ~/.agents/.instructions/INSTRUCTIONS_REMOTE_AGENTS.md, ~/.claude/instructions/INSTRUCTIONS_REMOTE_AGENTS.md, and rendered ~/.codex/AGENTS.md contain the new Remote Agent Workers section.

## Precondition Resources
(none)

## Outcome Resources
(none)
