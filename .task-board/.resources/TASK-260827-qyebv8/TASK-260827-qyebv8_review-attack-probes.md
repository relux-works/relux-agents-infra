# TASK-260827-qyebv8 reviewer attack probes

These probes were compiled outside the candidate from the exact production
contract source files. No source under review was modified.

## Out-of-range `max_tokens`

Payload:

```json
{"model":"configured","messages":[{"role":"user","content":"hello"}],"max_tokens":1e300}
```

Command shape:

```bash
swiftc -parse-as-library \
  tools/mlx-swift-runtime-prototype/Sources/MLXSwiftRuntimeContract/*.swift \
  .temp/TASK-260827-qyebv8-review/CrashProbe.swift \
  -o .temp/TASK-260827-qyebv8-review/crash-probe
.temp/TASK-260827-qyebv8-review/crash-probe
```

Observed:

```text
Swift/arm64e-apple-macos.swiftinterface:39435: Fatal error: Double value cannot be converted to Int because the result would be greater than Int.max
PROBE_EXIT=133
```

## Forged Metal-library directory

Helper-level observation after creating a directory at
`mlx-swift_Cmlx.bundle/Contents/Resources/default.metallib`:

```text
OBSERVATION=present(path: ".../mlx-swift_Cmlx.bundle/Contents/Resources/default.metallib")
ADMITTED_FAKE_LIBRARY_DIRECTORY
```

Composed executable probe:

1. Copy `.build/release/mlx-swift-runtime-prototype` into a task-scoped temp
   directory.
2. Create a directory, not a file, at the expected `default.metallib` path next
   to it.
3. Run `serve` with the existing zero-weight unsupported-architecture fixture on
   loopback port 28117.
4. Observe the production `listening` event, then SIGTERM and wait for exit.

Observed:

```text
LISTENING_WITH_FAKE_METALLIB_DIRECTORY=1
PROCESS_EXIT=0
{"event":"listening","host":"127.0.0.1",...,"port":28117}
{"event":"model_load_failed",...}
{"event":"shutting_down","signal":"SIGTERM"}
{"event":"stopped"}
```

The temporary process was synchronously terminated and reaped before the probe
command ended; no background process was left behind.
