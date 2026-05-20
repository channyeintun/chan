# Enhancement Progress

Source plan: `enhancements.md`

## Completed

- Made MCP startup progressive for stdio engine startup.
  - `ready` is emitted before MCP server discovery begins.
  - MCP discovery runs asynchronously and registers connected server tools as they become available.
  - Startup emits MCP status notices so the UI is not silent while MCP discovery continues.
- Moved MCP resource tools to prefer session-scoped manager injection.
  - Stdio engine now registers MCP resource tools with the active session manager.
  - Package-global MCP manager fallback remains for compatibility outside the stdio engine.
- Integrated streaming execution for approved tool calls.
  - Approved tools now run through `StreamingExecutor` so ordered results can be emitted as soon as contiguous results are ready.
  - Streaming parallel execution now respects `NAMI_MAX_TOOL_CONCURRENCY`.

## In Progress

- None.

## Pending

- Add bounded/cached turn-context collection and atomic persistence improvements.
- Consolidate shared provider transport behavior after differences are documented in debug logs.

## Notes

- Tests are intentionally not added per instruction.
