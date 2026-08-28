# BUG-260829-ajb7n7 review verdict

## Verdict

Accepted. No blocking findings.

Reviewed Change Request `CR-BUG-260829-ajb7n7-1` revision `1`, base
`675f77ed63376320ed1213f46f9462a299c0abaf`, candidate tree
`3799222b3bfb65d53a9d8d42c43381ad77b577e3`. The downloaded patch SHA-256 is
`3bc0beab9ca6e534365a8cf5ae604c25bc9ce05aefb7bfcfe79d0ed5dfc23514`, matching
the review assignment. The worktree paths under review are byte-equivalent to
the candidate tree.

## Implementation review

- `captureStdout` and `captureStderr` share `capturePipe`, so both surfaces use
  the same ordering and cleanup contract.
- The drain goroutine is launched before the producer is called. A start
  barrier prevents the producer path from being entered before that goroutine
  has started.
- Cleanup is idempotent and restores the process-global descriptor before
  closing the writer, waiting for EOF/drain completion, and closing the reader.
  The same deferred cleanup runs during panic unwinding.
- Separate stdout and stderr regressions write `3 MiB + 17 bytes`, compare the
  captured bytes exactly (including NUL and `0xff`), and assert normal-path
  descriptor restoration. A table-driven panic test asserts restoration for
  both global descriptors.
- Existing production-facing test call sites continue to use these wrappers;
  the exact previously hung call site is
  `TestRunRuntimeStatusJSONIsAbsentAndSideEffectFree -> captureStdout -> capturePipe`.

## Defeat attempt / expected-red proof

In an isolated `.temp` copy of candidate tree revision 1, I narrowed
`capturePipe` back to the prior producer-before-drain ordering while leaving the
new regressions intact. No candidate file was modified.

- `TestCaptureStdoutDrainsLargeOutputConcurrently`: expected `exit 1`; timed out
  after 2 seconds with the producer blocked in `os.File.Write` before the reader
  path.
- `TestCaptureStderrDrainsLargeOutputConcurrently`: expected `exit 1`; same
  blocked writer/read-start ordering.

This is not a gate/refusal/authorization/attestation change, so the standard
forged/absent/bypass gate shapes are not applicable. The ordering mutant is the
negative evidence for this concurrency regression and proves both new tests
fail against the prior implementation.

## Independent validation

- Focused stdout/stderr large-output and panic-restoration tests: `exit 0`
  (`1.030s`).
- Same focused tests under the race detector: `exit 0` (`1.505s`).
- Exact root status test
  `TestRunRuntimeStatusJSONIsAbsentAndSideEffectFree`: `exit 0` (`0.384s`).
- Complete root package, uncached: `exit 0` (`76.807s`).
- Complete root package, uncached JSON evidence: package `exit 0` (`81.425s`),
  `184` test-pass events, `0` skip events, `0` fail events, empty stderr.
- `go vet .`: `exit 0`.
- `go build .` to a `.temp` output: `exit 0`.
- Exact candidate diff: `git diff --check` exit `0`; only `LOGBOOK.md` and
  `tools/agents-infra/main_test.go` changed.

No live Pi/local-model runtime, service, socket, or endpoint was started,
probed, or contacted during review.
