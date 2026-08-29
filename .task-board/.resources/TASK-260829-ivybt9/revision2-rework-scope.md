# Revision 2 rework: prove absent attested resource evidence is refused

Rework `CR-TASK-260829-ivybt9-1` on the same managed Story workspace. Preserve
the existing 22-path candidate exactly except for the smallest test-only delta
needed to close the independent review finding. Do not change production
behavior unless the test exposes a real defect.

Required change:

- Add a production-entry negative test, preferably in
  `tools/agents-infra/internal/infra/pi_shared_resources_test.go`, that serves
  an otherwise valid, fully attested status response with the `resources` JSON
  field absent, invokes `SharedRuntimeStatusReport`, and requires the typed
  `protocol_violation` refusal.
- The test must fail when the production guard in
  `pi_shared_operator_darwin.go` is replaced by a permissive branch that keeps
  fallback resource status when `message.Resources == nil`.
- A helper-only assertion or broker-only test is insufficient.

Validation:

1. Run the new test red against the permissive mutant and green after restore.
2. Re-run the 14-test production resource/status slice under `-race`.
3. Re-run `go test ./... -count=1`, `go vet ./...`, gofmt/diff checks, Darwin
   build, Linux amd64 build, and Windows amd64 build.
4. Recompute the complete candidate identity and publish revision 2.
5. Preserve all historical lanes, root dirty files, board resources, and the
   exact selected base. Use fake/local test transports only; do not contact any
   live model/runtime/daemon/service/endpoint/socket.
