# Integration instruction — TASK-260916-3q6z2c (bound developer run)

Revision 6 is accepted (story_final, 32 changed paths). This board and code are colocated (relux-agents-infra-main control root), so the Story lands through `worktree integrate` run by YOU (the bound producer role). No board writes before or during the integrate. Run exactly, in the FOREGROUND, from the control root /Users/administrator/Developer/ReluxWorks/relux-agents-infra-main, and wait (the remote gate reruns on GitHub, ~10 minutes):

    task-board worktree integrate STORY-260916-1vt3x2 --cr TASK-260916-3q6z2c --revision 6 --commit-time "$(date -u +%Y-%m-%dT%H:%M:%SZ)" 2>&1 | tee .temp/integrate-3q6z2c.log

ONLY AFTER it exits: attach `.temp/integrate-3q6z2c.log` as outcome resource `TASK-260916-3q6z2c_integration-results.md` and stop. Do not push. If it refuses, attach the exact refusal and stop.
