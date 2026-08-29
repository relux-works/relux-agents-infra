## Status
backlog

## Review
required

## Task Class
code

## Estimate
notEstimated

## Blocked By
- TASK-260828-2wcrph
- TASK-260827-2v13w8
- TASK-260829-3k4qrc

## Blocks
- (none)

## Checklist
- [ ] Report states per runtime what was measured under which pinned conditions, with exact revisions and commands
- [ ] Every invalid comparison named with the reason it is invalid, not silently omitted
- [ ] Memory economy scored as a first-class criterion alongside throughput and latency
- [ ] Decision recorded with concrete blockers for each rejected candidate; Python mlx-lm stays default unless a candidate wins on evidence
- [ ] Abstract states the question, the method in one sentence, and the decision
- [ ] Method section lets a reader reproduce every number: host, models, quantization, pins, commands, revisions
- [ ] Results present all three runtimes in comparable tables, with non-comparable cells marked as such rather than blank
- [ ] Threats to validity names every known measurement limitation AND its direction, including the ones that favoured llama.cpp
- [ ] Discussion separates what the numbers show from what they would need to show to justify migration
- [ ] Conclusion is an explicit GO or NO-GO with the thresholds it was judged against
- [ ] Every claim traceable to an attached artifact; no number appears that cannot be sourced
- [ ] Article lives in articles/<YYMMDD>_local-qwen-runtime-comparison-study/ with the date of writing in the directory name
- [ ] Opens with a dated Provenance section: what was measured, on which binaries and host, on which dates, so a later reader can tell an expired finding from a live one
- [ ] ARTICLE.md, README.md, SHA256SUMS, artifacts/ and reproduce.zsh all present, matching the voice-research article convention
- [ ] Every cited number is reproducible from artifacts/ and checksummed in SHA256SUMS

## Notes

## Precondition Resources
(none)

## Outcome Resources
(none)

## Created
2026-08-28T10:12:56Z

## Last Update
2026-08-29T10:42:48Z
