# Independent review scope: fresh resource-pressure replay

Review `CR-TASK-260829-ivybt9-1` as an independent reviewer. This is a static,
fake-backed review only: do not start, stop, probe, or otherwise contact any
live local-model runtime, socket, service, or endpoint.

The change request must remain exactly bound to:

- base `6d051f54440d36e3ca3d132f8d9d1e78d46289de`
- candidate tree `dabd04a99420aceb21005de65221426bba252c37`
- patch SHA-256 `7e377be3bdbe65516820fcfa39cec620f0ca7afed60d1dcb72d8638410d475f5`
- 22 declared changed paths

Review requirements:

1. Verify the exact CR identity and changed-path set before judging behavior.
2. Re-run focused race-enabled tests and the full uncached Go suite, vet,
   gofmt, cross-platform compile/build checks, and diff checks.
3. Re-audit publication revalidation, starvation decoupling, pressure-policy
   boundaries, provenance/ownership correctness, restart/quarantine
   composition, and unsupported-platform behavior.
4. Add reviewer-owned negative tests or temporary mutants for each plausible
   silent regression. Restore all temporary mutations before publishing the
   verdict.
5. Confirm no live runtime was contacted and no unrelated root dirty files were
   incorporated.
6. Publish an explicit accepted or changes-requested verdict against revision 1
   with commands, evidence, and residual risks.

Do not implement product changes in this review run. If a defect is found,
request changes with a minimal reproduction and exact affected paths.
