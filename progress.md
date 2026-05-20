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

## In Progress

- None.

## Pending

- Integrate streaming tool execution for early result delivery.
- Add bounded/cached turn-context collection and atomic persistence improvements.
- Consolidate shared provider transport behavior after differences are documented in debug logs.

## Notes

- Tests are intentionally not added per instruction.
