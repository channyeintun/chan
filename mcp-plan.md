# MCP Integration Status

## Status

Phase 1 MCP client support is complete.

Chan now loads configured MCP servers at startup, discovers tools, exposes them to the model as namespaced tool calls, routes execution through the existing tool runner, and surfaces basic server status in `/status`.

## What Landed

- User-level MCP config in `~/.config/chan/config.json`
- Workspace-level MCP override in `.chan/mcp.json`
- Explicit transport support for `stdio`, `sse`, `http`, and `ws`
- Runtime MCP manager for startup connection, discovery, execution, and shutdown
- Dynamic tool registration with stable names in the form `mcp__<server>__<tool>`
- Config-driven permission mapping for trusted servers and conservative defaults for untrusted servers
- MCP-aware approval prompt metadata and basic status visibility through `/status`

## Finalized Decisions

1. Chan stays an MCP client only.
2. `.chan/mcp.json` is the canonical project-level MCP override file.
3. MCP servers are loaded at startup; config changes apply on the next fresh session.
4. Transport choice is explicit per server entry; Chan does not probe or fall back across transports.
5. Permission mapping is config-driven; Chan does not infer read/write safety from MCP annotations.
6. Discovered tools are exposed with stable namespaced identifiers instead of raw server tool names.

## Delivered Scope

### Config and runtime

- Layered user and workspace MCP config loading
- Transport-specific runtime validation and env expansion
- Graceful per-server startup failure handling

### Tool integration

- MCP tool adapters registered before the engine emits ready
- JSON argument pass-through to MCP tool calls
- Existing tool output budgeting reused for MCP responses

### Permissions and visibility

- Conservative default approval posture for MCP tools
- Trusted per-tool permission overrides through config
- MCP server connection status included in `/status`

## Verification Completed

- `go build ./...`
- Rebuilt and installed `chan` and `chan-engine` into `~/.local/bin`
- Live stdio smoke check against the Go MCP SDK hello server

## Remaining Follow-Ups

These are optional follow-ups, not blockers for the completed Phase 1 integration:

1. Add a dedicated `/mcp` command with richer per-server and per-tool detail.
2. Add fixture-based smoke checks for `sse`, `http`, and `ws` transports.
3. Evaluate exposing MCP prompts and resources as first-class context.
4. Evaluate broader auth flows if static headers or env-backed secrets are not enough.