# TASK-260828-15ftgj: consolidate-local-runtime-comparison-report

## Description
Consolidate the accumulated Python mlx-lm, MLX Swift and llama.cpp findings into one comparison report and record the resulting backend decision.

## Scope
Synthesis only. No new measurement, no configuration change. Deliverable is a dated article directory articles/<YYMMDD>_local-qwen-runtime-comparison-study/ in this repository, following the voice-research convention — ARTICLE.md, README.md, SHA256SUMS, artifacts/, reproduce.zsh — and the house provenance style of skill-project-management/articles/context-sharing-study.md, which opens with a dated 'Provenance - read this first' snapshot section. The decision is the article's conclusion, not an appendix.

## Acceptance Criteria
One research article in the classic structure — abstract, background, method, results, threats to validity, discussion, conclusion — presents every measured comparison across Python mlx-lm, MLX Swift and llama.cpp with exact revisions and commands, names every invalid comparison and why, states the directional bias of every known limitation, and ends in an explicit GO or NO-GO that selects the best overall compromise rather than a winner on any single axis, stating the weighting it was judged under.
