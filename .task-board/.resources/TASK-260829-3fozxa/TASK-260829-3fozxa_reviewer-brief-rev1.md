# Independent review brief — log rotation CR revision 1

Review immutable `CR-TASK-260829-3fozxa-1`:

- base: `891de4427bb7de6885b8b221f0e2b24a49a8fdc2`
- candidate tree: `4d3ad50d9920a5b179b6bbf702cc720feac21217`
- patch SHA-256:
  `6376a0a3d6e1cf2aec118aaecd5af1faccbf9eefc266c31ce337c010055d3405`

Reconstruct and verify the exact tree in a disposable review worktree. Do not
modify the producer workspace. Do not contact a live local-model process,
service, socket, endpoint, or user-owned harness.

Attack the production writer and configuration boundary, not only helpers:

- missing/zero/overflowing caps must refuse before runtime/provider side
  effects; there must be no numeric code default;
- rotation must occur at the documented exact byte boundary, including an
  existing file exactly at cap, a write crossing the cap, and oversized input;
- `max_segments` semantics must make aggregate retained bytes no greater than
  `max_segment_bytes * max_segments`, counting the active segment correctly;
- oldest-first pruning must be deterministic under equal/coarse timestamps and
  must never prune the active file or unrelated/foreign files;
- repeated simulated days/weeks must stay bounded without sleeps;
- process writer lifetime, close/error handling, permissions, and no-follow
  protections must remain safe across broker/runtime restart paths.

Use at least one negative mutant or independent probe capable of failing a
plausible off-by-one/pruning-order implementation. Run focused tests, full
`go test ./... -count=1`, `go vet ./...`, and `go build ./...`. If accepted,
record `accept_cr` revision 1 with exact evidence; otherwise attach a verdict
and route `to-dev`.

