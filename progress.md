# Maintainability Execution Progress

Last updated: 2026-05-21

Constraints:

- Never add tests.

## Status

| Step | Task | Status | Notes |
| ---: | --- | --- | --- |
| 1 | Add characterization tests for provider streams, tool execution, and persistence error paths. | Skipped | Conflicts with the explicit `Never add tests` constraint. |
| 2 | Introduce explicit runtime dependency structs for engine/tools without changing behavior. | Completed | Added grouped interaction/session runtime config helpers and rewired `engine.go`; `gofmt` and `go build ./...` passed. |
| 3 | Split `tool_executor.go` into executor, authorization, batching, rendering, and normalization files. | Pending | Not started. |
| 4 | Split the largest standalone tools: `web_fetch.go`, `lsp.go`, and `background_commands.go`. | Pending | Not started. |
| 5 | Split slash command handlers by domain and narrow `slashCommandContext`. | Pending | Not started. |
| 6 | Split provider implementations by request, stream, types, errors, and policy. | Pending | Not started. |
| 7 | Split retrieval graph and agent loop helpers after tests are in place. | Blocked | Depends on the skipped test step from the original plan. |
| 8 | Move MCP CLI business logic out of `cmd`. | Pending | Not started. |
