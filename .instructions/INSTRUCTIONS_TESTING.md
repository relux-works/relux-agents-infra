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
