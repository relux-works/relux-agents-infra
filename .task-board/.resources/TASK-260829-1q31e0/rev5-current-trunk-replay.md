# Revision 5 current-trunk replay requirements

Revision 4 is preserved as the immutable board resource
`TASK-260829-1q31e0_change-request_rev4.patch` with SHA-256
`2c69dc0a1a412ba53b09b241bbe4bf6aed3520962bfe516e0f30fba88ffa60da`.
Its Story workspace was intentionally discarded only after that digest was
verified, because its exact base `91356833949cb6a30958265514fe5852d97eec1b`
predates merged lifecycle-log rotation on current `origin/main`
`6d051f54440d36e3ca3d132f8d9d1e78d46289de`.

Reconstruct the revision-4 ownership-bound retention behavior semantically on
the exact fetched current trunk. Reconcile all overlapping lifecycle-log files
with the already-merged rotation implementation; do not replace current trunk
files wholesale and do not regress rotation's byte/count/age bounds, secure
file handling, status fields, or tests.

Preserve and independently prove the revision-4 fixes:

- an active log cannot bypass the aggregate byte cap after path mode,
  ownership, identity, or link-count mutation;
- a foreign file cannot self-mint deletion authority merely by matching a
  managed filename and mode;
- admission and pruning remain deterministic and race-safe;
- explicit positive configuration, aggregate profile-wide retention, bounded
  simulated-week footprint, diagnostics, and operator documentation remain
  intact;
- status is fail-closed or explicitly unknown when ownership/provenance cannot
  be established.

Run the focused adversarial tests, race detector, uncached full Go suite, vet,
format/diff checks, cross-platform compile checks, and the deterministic
multi-week soak. Publish a fresh Change Request from the current-trunk
workspace. Do not contact a live model runtime, service, socket, or endpoint.
