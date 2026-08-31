# TASK-260831-1bt8f4 retention replay result

The accepted retention state-machine delta was replayed onto the exact landed generic Pi adapter base `8caac7f975975724a884bd9ca5b577f075ccc878` in the managed Story workspace.

- Candidate tree: `d2870eba4186ca0bd85b19fa0b4eff688eb88cff`
- Patch SHA-256: `30e40262f96a3fd52743e12156d81dc65dc7015e63c14ae00d31a761fb3437cd`
- Scope: exactly the previously accepted 26 paths; no widened paths
- Adapter overlap: three hand-resolved unions (`tools/agents-infra/main.go`, `README.md`, `LOGBOOK.md`) plus three clean automatic merges
- Round-trip: applying the binary patch to the base reproduces the candidate tree exactly
- No live Pi runtime, model process, socket, endpoint, or user runtime state was contacted

Validation passed:

- `go build ./...`
- `go vet ./...`
- `gofmt -l` over changed Go files
- `git diff --check`
- focused lifecycle and operator-documentation tests
- 44-test legacy, automatic, pressure, lifecycle, and deterministic eight-week soak suite
- complete `go test ./... -count=1`
- complete `CGO_ENABLED=1 go test -race ./... -count=1 -timeout 30m` on the confirming rerun
- expected-red foreign-evidence and legacy-evidence narrowing mutants
- Linux amd64, Linux arm64, and Windows amd64 cross-builds
- isolated setup/verify and installed README/SKILL/LOGBOOK byte parity

The first race run hit one unrelated pre-existing deadline-process-group flake in an unmodified test. That exact test passed in isolation and the identical full race suite passed on the confirming rerun. Raw tree-bound command evidence is retained in `TASK-260831-1bt8f4_change-request_rev1-validation.log`.

The workspace remains intentionally dirty at the candidate tree so task-board can publish the immutable Change Request during the corrected producer handoff. Integration remains an orchestrator action after independent acceptance.

## Publication recovery confirmation — RUN-260831-cfb19a

This run made no repository change and reran only bounded drift checks required to republish the already validated candidate.

- Hard dependency `TASK-260831-26b034` is `done`.
- Fresh `git fetch origin main` exited 0.
- After that fetch, workspace `HEAD`, Story branch tip, local `main`, `origin/main`, and `FETCH_HEAD` all equal base `8caac7f975975724a884bd9ca5b577f075ccc878`.
- `git write-tree` exited 0 and reproduced candidate tree `d2870eba4186ca0bd85b19fa0b4eff688eb88cff`.
- `git diff --name-only HEAD` exited 0 and still reports exactly the recorded 26-path retention candidate; no path was added or dropped.
- `git diff --check HEAD` exited 0 with empty output.
- The immutable accepted input patches still reproduce SHA-256 `1ed35314955527822a6211f11510c10582aa9f588e9558731d1321b837c117ad` (revision 6) and `0c63c3bc5d9ea0496fc2c26c112f9361ee092791681950eff38cbb0023478afb` (old-base replay revision 1).
- The previously attached tree-bound validation transcript remains the validation authority for build, vet, format, full/race, lifecycle mutants, cross-platform, isolated parity, deterministic eight-week soak, and no-live-runtime gates; this recovery did not claim to rerun those long gates.

The candidate is unchanged and ready for publication as a bounded `story_final` Change Request against the exact selected base above.
