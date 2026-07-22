# Workflow

## Priority: Scope Control Over Speed

* **Task tracking and scope control always come first.** Never skip creating a task "because it's small" or "just a quick fix".
* If the repo already has an established task tracker, board, or workflow artifact, **every code change must be tracked there BEFORE implementation starts.**
* The workflow is: **trigger → skill → task tracking → implement → review → reopen or close.** Never jump straight to implementation because it feels faster.
* Speed is a side effect of good process, not a substitute for it. A change done fast but untracked is worse than a change done properly.

## Proportional Process (Mandatory)

* Match process overhead to the current task's scope, risk, reversibility, and requested outcome. Prefer the smallest workflow that can complete the work correctly and leave enough evidence to trust it.
* A repository having a task tracker does **not** by itself trigger the `project-management` skill, board decomposition, planning artifacts, spawned agents, producer/reviewer chains, or exhaustive validation. Use those only when the user asks, the repository or an applicable skill explicitly requires them, or the work materially benefits from specialization, parallelism, context isolation, or independent review.
* For localized, mechanical, and readily reversible work—such as copying files into an existing package, making a small documentation or configuration edit, formatting text, or rerendering an existing artifact—reuse an existing task or create the minimum required tracker entry, work inline, run narrow checks that can catch regressions caused by that exact delta, and finish.
* Self-review is sufficient for low-risk changes unless an independent review is explicitly required or justified by meaningful failure impact, ambiguity, security, legal sensitivity, or irreversible external effects. Do not spawn a reviewer merely to satisfy ceremony.
* Re-evaluate the workflow whenever scope narrows. Do not carry forward earlier research, review, rendering, or validation gates unless they still protect against a plausible failure in the current delta.
* Skills and workflow systems are constraints, not deliverables. If a mandatory repository or skill contract requires a heavier process, follow it, but do not add extra layers beyond that contract. Ceremony that costs more than the work without increasing confidence is a process failure.

## Model Availability and Fallback

* Treat temporary model unavailability as an operational condition, not as a decision to hand back to the human.
* Do not ask the human whether to keep waiting for the preferred or top-tier model or switch to a faster, cheaper, or less capable model.
* Keep the requested or preferred model and retry it at least three times with a reasonable delay or backoff when the runtime or orchestration tool supports retries. Avoid parallel retry storms.
* If the preferred model remains unavailable after those retries, autonomously choose the best viable execution path for the task: the next-best available model at the highest appropriate reasoning level, another configured agent or provider, or local/inline execution when permitted.
* Preserve the task scope and quality gates when falling back. Record the fallback and its evidence in the task tracker or final handoff, but do not request permission merely to continue.
* Escalate only when no viable execution path remains, or when progress requires human-only authentication, access, approval, or a genuine product/architecture decision. State the exact blocker and the retries or alternatives already attempted.
* This fallback policy does not permit bypassing safety, usage, access, or approval controls.

## Stop-The-Line: No Forced Fits

* Autonomous completion is the default. Work through difficult, slow, or uncertain implementation problems without escalating merely because they require more investigation, retries, or engineering effort.
* Stop only when evidence shows that an external platform/API constraint makes the requested behavior objectively impossible or unsafe, or when progress genuinely requires a human product, architecture, ownership, access, or approval decision.
* A forced fit is visible when each attempt adds more flags, stubs, mocks, priority rules, mock-only behavior, special cases, or tests that avoid the real behavior. Do not use tests or compensating code to make an invalid product or platform model look implemented.
* Canonical example: before Bluetooth permission, iOS cannot reliably reveal whether Bluetooth is powered off without initializing or otherwise touching CoreBluetooth, which may trigger the permission prompt. A product requirement that prioritizes powered-off Bluetooth state before permission has no clean implementation. Surface permission-first, unknown-state, or requirement-change options instead of adding booleans, stubs, mocks, or priority hacks.
* Before escalating, persist the exact constraint, evidence, failed assumption, clean approaches attempted, viable alternatives, tradeoffs, and the precise human decision or external input needed in the repo's task/documentation flow.
* Then present the options or mark the task `blocked` with that exact decision/input. Once the constraint is resolved, resume autonomous execution and finish the task.

---

## Version Control

* **Never commit or stage files automatically.**
* When work is ready to commit, stop and ask for review.
* **Never add Co-Authored-By lines** or any AI attribution to commits.
* When you need to work on multiple revisions, parallel fixes, or isolated experiments in the same repo, prefer **`git worktree`** over juggling branches in one checkout.
* Place temporary worktrees under the project's **`.temp/`** directory, not next to the main checkout.
* If the worktree is for a tracked task, place it under a task-scoped temp path using the task ID:
  * `.temp/<TASK-ID>/worktree/`
  * or `.temp/<TASK-ID>/<repo-name>-worktree/`
* This keeps the main checkout stable while making task-local scratch state easy to find and clean up.

---

## Task Tracking

* If the active repo or skill already defines a task system, use it. Don't create a parallel `.temp/tasks.md`.
* If no project-native task system exists — create a task plan in `.temp/tasks.md` before starting work.
* Track progress in the same file.
* Update/append to the existing plan — **don't create new task files each session**.
* Purpose: resume smoothly if the session breaks.

---

## Primary Parent Goal Actualization

* Apply this policy only in a primary parent session when `TASK_BOARD_RUN_ID` is absent and an active task-board is available.
* On the first materially actionable user requirement, read `task-board goal get`. If no primary goal is active, create it with `task-board goal set-primary --objective TEXT --reason TEXT`. If one is active, update it with `task-board goal update --if-revision N --objective TEXT --reason TEXT` only when the concise, complete objective materially changed.
* On every later user turn that materially adds, removes, or redirects requirements, recompute one complete objective that preserves every unresolved prior requirement and incorporates the changed intent. Update using the revision observed from the latest read.
* Perform at most one successful primary-goal write per user turn. Skip status questions, confirmations, tool chatter, wording-only corrections, semantic no-ops, and other turns that do not change the goal.
* On `primary_goal_revision_conflict`, re-read the active goal, merge the complete objective, and retry once with the new observed revision. Do not narrate routine successful synchronization; report only a persistent failure or conflict, or answer an explicit user request for goal state.
* Never mutate the primary goal from a spawned run. Spawned owners use `task-board spawn goal`; a primary-goal update never silently expands, cancels, completes, or clears a spawned goal. When materially changed delivery scope must reach an existing owner, use the explicit spawn-goal upsert or reroute contract.
* Version 1 does not invoke native Codex or Claude goal APIs and never clears the primary goal automatically when a session exits.
* This is an instruction-only integration: agents-infra stores no task-board state and adds no task-board library dependency. The eligible primary parent calls the external `task-board` CLI.

---

## Research & Knowledge Persistence

* **All research must go through the repo's established task/documentation flow.** Never keep research only in conversation context.
* If the active repo or skill defines an artifact location, store research findings there and link them from the relevant task/doc/worklog.
* If no project-specific convention exists — store in `.temp/` with descriptive names (`research-auth-flow.md`, `analysis-performance.md`).
* **Why:** Context windows collapse. If research lives only in the conversation, it's lost forever when the session resets. Files persist.
* Sub-agents doing research/analysis **must** write their findings to files before finishing.
* Reference research artifacts from the relevant task/doc/worklog so the next session can find them quickly.

---

## Logging

* Store logs in `.temp/` with numbered naming:
  * `pod-lint-01.log`, `spm-build-01.log`, etc.
* Document log locations in your notes/tasks.
