# Testing & Refactoring

## Testing

* Use **Swift Testing** framework, not XCTest.
* Tests must be in **Swift**, not ObjC.

## Negative Tests Are A Development Baseline

* Every development change must consider both required behavior and forbidden
  outcomes. Add or preserve relevant executable negative tests for affected
  contracts in features, bug fixes, refactoring, and integration. This is not
  limited to security, validators, or explicit rejection gates.
* Define negative expectations while specifying the change, before treating
  happy-path success as completion. Derive cases from requirements and realistic
  failure modes: invalid inputs, boundary values, forbidden transitions, missing
  authority or dependencies, and interruption, retry, or concurrency where
  relevant. Carry these expectations into implementation and review handoffs.
* Assert the intended rejection or invariant, not merely any error, crash, or
  timeout. Check post-failure state and observable effects against the contract:
  no forbidden partial writes, leaked information, duplicate effects, or lost
  recovery guarantees. Permitted rejection telemetry is not a forbidden effect.
* Pair negative cases with nearby valid positive controls. An implementation
  that rejects everything must not satisfy the tests. Two implementations
  agreeing is not proof of correctness: ground expected verdicts in an
  independent requirement and probe both disagreement directions when
  equivalence is required. Document intentional asymmetry instead of demanding
  identical behavior across different contracts.
* For bug fixes, retain a regression witness and test relevant neighboring
  variants so that the fix addresses the failure class, not just one input.
  Preserve existing negative guarantees when refactoring.
* Scale depth to risk and the changed contract, not a universal test quota.
  Existing applicable coverage may be reused with evidence. For non-executable
  changes, use meaningful contract fixtures or counterexamples where applicable.
  Record a specific rationale for an inapplicable check and distinguish checks
  that could not run from passing evidence. Silence or a small diff is not a
  waiver; superficial tests written only to satisfy a checklist are not evidence.
* When acceptance relies on a checker, scanner, gate, or test harness, establish
  that it detects a known violation of the claimed property using a bounded
  control plant where applicable. Require the intended failure: a skipped or
  not-applied plant, unrelated setup failure, or crash is not a successful
  control. Instrument controls supplement product negative tests; they do not
  mandate exhaustive mutation testing for every edit. Reassess reused control
  evidence when the relevant instrument, configuration, or contract changes.
* Run fault injection and destructive negative cases in isolated fixtures or
  explicitly authorized test environments. Negative-testing obligations never
  authorize damaging user data, resetting live devices, weakening permissions,
  or violating the state-preservation rules below.

## Android App State Preservation

* Preserve the currently installed app, its data, granted permissions, active
  sessions, and user-visible state by default during Android build, test, and
  device-debugging workflows.
* Prefer read-only inspection, targeted instrumentation, in-app navigation,
  activity relaunch, and other non-destructive verification paths before
  changing package or process state.
* Do not reinstall, uninstall, clear app data/storage, revoke permissions, or
  force-stop the app merely to obtain a clean baseline or retry a check. Use
  such operations only when evidence shows they are necessary for the exact
  test or fix being validated.
* When a new APK must be installed, install it once with the least disruptive
  supported update path and reuse that installation across related checks.
  Avoid repeated install cycles between test attempts.
* When spawning a developer, tester, or other device worker, carry this state-
  preservation contract into the task instructions explicitly. Do not delegate
  a vague "run the Android tests" request that lets the worker choose a lane
  which may uninstall the package, replace the app, clear its data, revoke its
  permissions, or discard the user's authenticated/session state when those
  mutations are not required by the task.
* On a physical device, avoid Gradle-managed install/test lanes when direct
  instrumentation, an already-installed app, or a test-APK-only update can
  validate the requested behavior. Use a lane that may uninstall or replace the
  app only when that package mutation is necessary for the exact task and its
  state/permission consequences have been accounted for in advance.
* Before an unavoidable state-resetting operation, resolve the exact package,
  capture any needed evidence, state why the reset is required, and account for
  permissions, approvals, authentication, or setup that only the user can
  restore. If the operation would create new human-only interaction and that
  interaction is not already authorized, stop and request direction instead of
  destroying the working state.

---

## Refactoring workflow

When refactoring (e.g., ObjC → Swift):

1. **Write tests first** (if none exist).
   * Test coverage must be **high for the code being refactored** (not the whole project):
     * target **~80%+** at minimum;
     * **prefer 100%** where practical.

2. **Refactor code.**

3. **Run tests** to verify nothing broke.
