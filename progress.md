# Progress

## 2026-04-15

- Fixed the repeated TUI out-of-memory crash caused by oversized child-agent summaries being duplicated into live tool results, background-agent updates, and replayed UI state.
- Completed Phase 1 MCP client integration for Chan: layered user and workspace config, explicit `stdio`/`sse`/`http`/`ws` transport support, runtime server management, dynamic `mcp__<server>__<tool>` registration, config-driven permission mapping, and MCP status visibility in `/status` and the README.
- Validated the MCP work with `go build ./...`, rebuilt and installed the latest `chan` and `chan-engine` into `~/.local/bin`, and ran a live stdio smoke check against the Go MCP SDK hello server.
- Changed plan mode to `Ultrathink`: removed write blocking, updated the plan-mode system prompt, and updated the `/plan` command description so explicit create/modify requests are allowed in plan mode.