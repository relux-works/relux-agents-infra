# Independent review scope: pressure revision 2

Review `CR-TASK-260829-ivybt9-2` independently. This revision must be bound to
base `6d051f54440d36e3ca3d132f8d9d1e78d46289de`, candidate tree
`eb94a41aefe919622f6031450c936df8ffe20ac1`, patch SHA-256
`b5d3ffd6b9123d459716bcd336300b06d5b0dc4e222bc328b7918c207e79f6d5`,
and exactly 22 changed paths.

Revision 2 is expected to differ from revision 1 only by the new production
entry negative test in `pi_shared_resources_test.go`; production bytes must be
unchanged.

Required review:

1. Verify the exact CR identity and revision-1-to-revision-2 delta.
2. Reapply the permissive absent-`resources` mutant to
   `SharedRuntimeStatusReport` in an isolated scratch copy and prove the new
   named test fails; restore and prove the exact candidate bytes.
3. Re-run the focused production/race slice, full uncached suite, vet,
   formatting/diff checks, and Darwin/Linux/Windows builds in proportion to the
   complete 22-path candidate.
4. Confirm prior concurrency, policy, provenance, ownership, restart/quarantine,
   and unsupported-platform evidence remains valid.
5. Publish a new reviewer-owned verdict and explicitly accept revision 2 only
   if every gate passes.

Static fixtures and fake task-owned transports only. Do not contact any live
model/runtime/daemon/service/endpoint/socket or modify root dirty files and
historical lanes.
