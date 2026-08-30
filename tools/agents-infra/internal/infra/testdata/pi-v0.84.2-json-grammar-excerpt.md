# Pi v0.84.2 JSON grammar excerpt

Source: `packages/coding-agent/docs/json.md` from the pinned
`earendil-works/pi@v0.84.2` Darwin/arm64 release tree. The installed release
path is `./docs/json.md`; its SHA-256 in
`pi-v0.84.2-darwin-arm64-tree-manifest.txt` is
`9c324127fac36eadc781c5222937f3a2b938a5fd671976aab020b27d7c1362a7`.

The following source excerpt is preserved verbatim:

> Each line is a JSON object. The first line is the session header:
>
> `{"type":"session","version":3,"id":"uuid","timestamp":"...","cwd":"/path"}`
>
> `message_update` records are delta-only. They omit both the cumulative `message` field and
> `assistantMessageEvent.partial` to keep stream size linear. The top-level `usage` field contains
> the latest cumulative provider-reported usage and may remain zero when a provider only reports
> usage at completion. Use `contentIndex` and `delta` to assemble live text, thinking, or tool-call
> arguments if needed. `message_end` contains the final authoritative message.

The same pinned source declares the closed tool lifecycle fields used by the
translator: `toolCallId`, `toolName`, and `isError` on
`tool_execution_start`, `tool_execution_update`, and `tool_execution_end`.
