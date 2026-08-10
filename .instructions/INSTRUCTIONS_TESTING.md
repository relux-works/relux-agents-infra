# Testing & Refactoring

## Testing

* Use **Swift Testing** framework, not XCTest.
* Tests must be in **Swift**, not ObjC.

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
