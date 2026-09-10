Rework rev 3 of CR-BUG-260830-5pmaiz per BUG-260830-5pmaiz_review-verdict-rev2.md (changes_requested).

Scope of this rev: LOGBOOK.md only. Product delta (project_config_field_gaps.go, infra.go, main.go, pi_test.go) left byte-identical to rev1/rev2.

Defect fixed: rev2 LOGBOOK.md recombination dropped the trunk entry "### 1345 — Environment Admission Has a Coincidental Second Gate" (TASK-260911-39bag5, 11 lines) that exists in trunk b138ebc:LOGBOOK.md and in the Story branch head 1688f0c:LOGBOOK.md. Restored verbatim from git show b138ebc:LOGBOOK.md, in original position (newest-first under 2026-09-11), between the 1730 candidate entry and the 1631 candidate entry.

Proof:
- diff <(git show b138ebc:LOGBOOK.md) LOGBOOK.md shows ONLY additions (>) — the two candidate entries (1730 BUG-260830-5pmaiz, 1631 BUG-260830-1rths2). Zero deletion (<) lines.
- grep -c "Coincidental Second Gate" LOGBOOK.md == 1
- git status --short: only LOGBOOK.md, infra.go, pi_test.go, main.go modified + project_config_field_gaps.go untracked — same file set as rev1/rev2, no other file touched.

Validation re-run (bounded):
- go build ./... (tools/agents-infra): clean, exit 0.
- go test ./internal/infra/... -run TestDoctorReportsEveryMissingSharedRuntimeFieldInOneInvocation|TestDoctorReportsNoFieldGapsForValidSharedRuntimeProfile|TestParsePiRuntimeSharingIsStrictAndOptIn -v: all PASS, exit 0.