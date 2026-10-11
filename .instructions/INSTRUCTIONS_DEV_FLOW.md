# Development flow (process specifics)

This module is an interim home for process-specific development-flow rules.
These rules will move into the per-project flow configuration alongside task
structure, nesting and types, roles bound to skills, and role bindings to tasks
or individual task states. Keep each rule self-contained and movable.

## Mutation testing

* Catch mutation regressions after landing: run one full mutation sweep on the
  exact trunk commit after each landing. A red sweep is a defect of the landed
  change: file a bug, fix forward, and land nothing else into that domain until
  the sweep is green.
* Mutation suites are not a gate for development, review rounds, or landing.
* During a feature or bug fix, the worker may, at its own discretion, run and
  write mutants scoped to the module, service, or bug it is working on as a
  development tool. Respect host limits: if a host refuses local kill tests,
  write and compile there only, and run them on hosted infrastructure or
  another permitted host.
* As performance guidance, key mutation result caches on what changes a
  verdict: the mutant definition, targets, killer tests, and toolchain, rather
  than a whole tool binary. This guidance is not a gate.
* As performance guidance, when a gate's cost jumps by an order of magnitude,
  stop and check the selection before extending budgets or windows. This
  guidance adds no mutation gate to development, review, or landing.
