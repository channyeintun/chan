# Maintainability Execution Progress

Last updated: 2026-05-21

Constraints:

- Never add tests.

## Status

| Step | Task | Status | Notes |
| ---: | --- | --- | --- |
| 1 | Add characterization tests for provider streams, tool execution, and persistence error paths. | Skipped | Conflicts with the explicit `Never add tests` constraint. |
| 2 | Introduce explicit runtime dependency structs for engine/tools without changing behavior. | Completed | Added grouped interaction/session runtime config helpers and rewired `engine.go`; `gofmt` and `go build ./...` passed. |
| 3 | Split `tool_executor.go` into executor, authorization, batching, rendering, and normalization files. | Completed | Split the implementation across focused files; `gofmt` and `go build ./...` passed. |
| 4 | Split the largest standalone tools: `web_fetch.go`, `lsp.go`, and `background_commands.go`. | Completed | All three tool files were split into focused helper files; each substep was verified with `gofmt` and `go build ./...`. |
| 4a | Split `background_commands.go` into lifecycle, output-buffer, and inspection/status files. | Completed | Extracted focused files for command lifecycle, output buffering, and inspect/stop/render helpers; `gofmt` and `go build ./...` passed. |
| 4b | Split `web_fetch.go` into service-focused files behind the tool adapter. | Completed | Separated the tool adapter from HTTP/URL policy, report building, readability extraction, and cache helpers; `gofmt` and `go build ./...` passed. |
| 4c | Split `lsp.go` into transport/client-focused files behind the tool adapter. | Completed | Split the tool adapter, client lifecycle, JSON-RPC transport, server resolution, and result shaping helpers; `gofmt` and `go build ./...` passed. |
| 5 | Split slash command handlers by domain and narrow `slashCommandContext`. | Completed | Split the monolithic handler file into focused command-domain files while preserving the shared context and dispatch flow; `gofmt` and `go build ./...` passed. |
| 6 | Split provider implementations by request, stream, types, errors, and policy. | In progress | Starting with the largest provider files and keeping behavior unchanged. |
| 7 | Split retrieval graph and agent loop helpers after tests are in place. | Blocked | Depends on the skipped test step from the original plan. |
| 8 | Move MCP CLI business logic out of `cmd`. | Pending | Not started. |
