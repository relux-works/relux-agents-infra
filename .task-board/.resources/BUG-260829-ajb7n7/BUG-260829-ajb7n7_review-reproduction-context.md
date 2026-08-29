# Review reproduction context

The issue was observed during independent review run `RUN-260829-55bcde` for `TASK-260829-3fozxa` on current `origin/main@675f77ed63376320ed1213f46f9462a299c0abaf`.

- The composed infra matrix completed green.
- A root status test then hung in the unchanged stdout-capture path.
- `tools/agents-infra/main_test.go:captureStdout` and `captureStderr` both call the producer while `os.Stdout`/`os.Stderr` points at an `os.Pipe` writer and only begin `io.Copy` after the producer returns.
- Any producer output larger than the platform pipe buffer can block the producer forever, making the reader unreachable.

This is test infrastructure only. Do not start, stop, probe, or connect to a live Pi/local-model runtime, service, socket, or endpoint. Use static/fake output producers and deterministic multi-megabyte byte identity checks.
