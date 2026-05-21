# Go Engine Maintainability Review Plan

Review date: 2026-05-21

Scope: Go code under `nami/`, with emphasis on the Go engine, core agent loop, API providers, tools, and CLI boundary.

Guidance applied: `AGENTS.md` and `nami/internal/skills/builtins/go-style-guide.md`.

This is a findings and improvement plan only. No implementation has been performed.

## Summary

The Go engine is functional but several areas are trending toward monolithic, hard-to-maintain code. The biggest risks are very large files, packages with broad responsibilities, hidden package-level mutable state, long orchestration functions with many dependencies, provider-specific conditionals leaking through generic layers, and very low test coverage around core behavior.

Current size signals:

| Area | Go LOC | Notes |
| --- | ---: | --- |
| `nami/internal/tools` | 15,428 | Largest package, 54 files, multiple tool files combine adapters and business logic. |
| `nami/internal/engine` | 10,187 | Broad runtime package covering startup, IPC, slash commands, subagents, tool execution, session memory, MCP, and conversation state. |
| `nami/internal/api` | 6,476 | Provider implementations are large and mix request building, streaming, auth, schema conversion, and quirks. |
| `nami/internal/agent` | 5,223 | Core loop is split somewhat, but orchestration state and retrieval graph remain broad. |

Largest files:

| File | Lines | Primary issue |
| --- | ---: | --- |
| `nami/internal/tools/web_fetch.go` | 1,736 | Tool adapter, HTTP, URL policy, readability extraction, markdown conversion, relevance scoring, artifacts, and cache in one file. |
| `nami/internal/engine/slash_command_handlers.go` | 1,397 | Many unrelated slash command domains in one file and one broad context object. |
| `nami/internal/engine/subagent_runtime.go` | 1,156 | Subagent setup, workspace, hooks, query loop, persistence, tool execution, and result handling in one runtime file. |
| `nami/internal/api/gemini.go` | 1,148 | Provider client, request, stream parsing, schema conversion, and provider behavior in one file. |
| `nami/internal/engine/tool_executor.go` | 1,087 | Tool execution pipeline, permissions, hooks, batching, budgeting, event emission, and helpers in one file. |
| `nami/internal/tools/lsp.go` | 1,024 | Tool adapter, LSP server discovery, process lifecycle, JSON-RPC transport, protocol types, and rendering in one file. |
| `nami/internal/agent/retrieval_graph.go` | 1,023 | Graph model, invalidation, language parsers, import resolution, scoring, and test heuristics in one file. |
| `nami/internal/agent/loop.go` | 1,012 | Query orchestration, model recovery, compaction, retrieval telemetry, failure logging, and retry handling in one file. |

Only two Go test files were found: `nami/internal/tools/apply_patch_test.go` and `nami/internal/engine/tool_executor_test.go`.

## Priority 1: Split Monolithic Runtime Orchestration

### Finding: `RunStdioEngine` owns too much startup and runtime wiring

Evidence: `nami/internal/engine/engine.go:31`

`RunStdioEngine` is a long orchestration path that sets up debug logging, IPC, tool globals, provider/client boot, session and artifact stores, MCP, slash routing, user-input routing, background controls, hooks, and mode/session persistence.

Opportunity: introduce an explicit engine runtime type with small setup methods.

Suggested target shape:

| Component | Responsibility |
| --- | --- |
| `Runtime` or `Engine` | Own bridge, router, registry, stores, trackers, model state, and shutdown. |
| `bootstrap` | Load config, create session, initialize client, create stores, emit ready state. |
| `registerTools` | Register built-in and runtime-dependent tools. |
| `dispatchMessage` | Route IPC messages to focused handlers. |
| `shutdown` | Close MCP, clear runtime globals, stop background commands, close debug logging. |

Expected benefit: makes dependencies explicit, reduces parameter lists, and creates test seams for startup behavior.

### Finding: `engine` package has too many domains

Evidence: `nami/internal/engine` is about 10,187 LOC across 27 files.

The package currently owns startup/provider auth, slash commands, subagents, background agents, session memory, tool execution, MCP status, conversation hydration/timeline, Codex connect, and plan review gates.

Opportunity: split by responsibility while keeping `internal` visibility.

Candidate package boundaries:

| Candidate | Move from | Purpose |
| --- | --- | --- |
| `internal/engine/runtime` | `engine.go`, loop wiring | Engine lifecycle and IPC routing. |
| `internal/engine/slash` or `internal/slash` | `slash_*` | Slash registry, command context, handlers by group. |
| `internal/engine/toolrunner` | `tool_executor.go`, subagent execution subset | Shared tool execution pipeline with parent/subagent policy hooks. |
| `internal/subagent` | `subagent_*`, background agent logic | Subagent runtime, worktrees, role policy, background state. |
| `internal/sessionmemory` | `session_memory.go` | Session-memory extraction and summarization independent of engine wiring. |

Expected benefit: smaller packages with clearer ownership and less accidental access to unexported engine state.

## Priority 2: Make Runtime Dependencies Explicit

### Finding: hidden mutable global state couples `engine` and `tools`

Evidence:

| Location | Coupling |
| --- | --- |
| `nami/internal/engine/engine.go:43` | Sets background command notifier in `tools`. |
| `nami/internal/engine/engine.go:117` | Sets global file history in `tools`. |
| `nami/internal/engine/engine.go:118` | Sets global file read state in `tools`. |
| `nami/internal/engine/engine.go:119` | Sets global session artifacts in `tools`. |
| `nami/internal/engine/engine.go:128` | Sets global swarm runtime in `tools`. |
| `nami/internal/engine/engine.go:143` | Sets session control runtime in `tools`. |
| `nami/internal/engine/engine.go:144` | Sets tool search runtime in `tools`. |
| `nami/internal/engine/engine.go:203` | Sets ask-user runtime in `tools`. |
| `nami/internal/tools/background_commands.go:100` | Package-level background command registry and notifier. |

Opportunity: replace package globals with explicit managers or runtime dependencies.

Suggested path:

1. Define a `tools.Runtime` or small manager interfaces for file history, read state, artifacts, background commands, ask-user, and session controls.
2. Pass runtime dependencies to tool constructors or registry registration groups.
3. Keep temporary compatibility wrappers only if needed during an incremental refactor.

Expected benefit: fewer cross-session hazards, easier tests, and clearer ownership of lifecycle and shutdown.

## Priority 3: Thin Tool Adapters, Extract Tool Services

### Finding: `tools` package is the largest package and mixes adapters with business logic

Evidence: `nami/internal/tools` is about 15,428 LOC across 54 files.

Several tool files combine JSON schema, permission behavior, parsing, I/O, domain algorithms, persistence, and rendering. This conflicts with the guide's package-design advice to keep each package focused and core logic independent of transport or UI.

Primary split targets:

| File | Evidence | Opportunity |
| --- | --- | --- |
| `nami/internal/tools/web_fetch.go` | 1,736 lines; tool/schema at `:186`, HTTP at `:267`, readability at `:433`, cache at `:1651`. | Move HTTP fetch, URL policy, readability, markdown conversion, passage selection, and cache into focused `webfetch` services. Keep `WebFetchTool` as schema/parse/call/render. |
| `nami/internal/tools/lsp.go` | 1,024 lines; schema at `:182`, client setup at `:266`, JSON-RPC/client code later in file. | Move LSP process and JSON-RPC transport into `internal/lspclient`; keep operation handlers small. |
| `nami/internal/tools/background_commands.go` | 803 lines; globals at `:100`, process lifecycle at `:144`, output streaming at `:242`. | Introduce `BackgroundCommandManager`; split process lifecycle, output buffer, and inspection/stop APIs. |
| `nami/internal/tools/apply_patch.go` | 596 lines; `Execute` combines parse, file writes, history, diff preview, diagnostics. | Separate pure patch parsing/planning from filesystem application and rendering. |
| `nami/internal/tools/file_read.go` | 548 lines; text reads, notebook rendering, binary detection, pagination, read-state, metrics. | Split text reader, notebook reader, binary detection, and result rendering. |
| `nami/internal/tools/dependency_overview.go` | 552 lines; discovery plus parsers for multiple ecosystems. | Move parsers by ecosystem into a dependency package or separate files. |

Expected benefit: tool behavior becomes testable without running through model-tool plumbing, and new tools do not expand one large package surface indefinitely.

### Finding: `ToolOutput` is a broad shared result object

Evidence: `nami/internal/tools/interface.go:33`

`ToolOutput` contains generic output plus file-read fields, diff metrics, diagnostics, artifacts, spill behavior, error kind, and error hint. This encourages unrelated tools to share fields they do not own.

Opportunity: keep a minimal common output and move domain-specific metadata behind typed metadata or renderer-specific structs.

Expected benefit: reduces hidden coupling between tools and engine rendering paths.

### Finding: `NewRegistry` centralizes all built-in tool creation

Evidence: `nami/internal/tools/registry.go:19`

Opportunity: group registrations by domain, such as file tools, search tools, process tools, LSP tools, swarm tools, and MCP tools.

Expected benefit: less merge conflict risk and clearer dependency ownership.

## Priority 4: Split Provider Implementations and Provider Policies

### Finding: provider files are large and mix unrelated concerns

Evidence:

| File | Lines | Issue |
| --- | ---: | --- |
| `nami/internal/api/gemini.go` | 1,148 | Client setup, request shape, stream parsing, usage handling, tool calls, and safety/block reasons. |
| `nami/internal/api/anthropic.go` | 978 | Client setup, request conversion, SSE dispatch, error handling, usage, and provider quirks. |
| `nami/internal/api/openai_responses.go` | 892 | Responses API request building, stream state, event handling, errors, and provider-specific behavior. |
| `nami/internal/api/openai_compat.go` | 758 | OpenAI-compatible request, stream handling, tool call assembly, and compatibility quirks. |

Opportunity: split each provider into focused files.

Suggested file shape:

| File suffix | Responsibility |
| --- | --- |
| `_client.go` | Client struct, constructor, base transport. |
| `_request.go` | Model request conversion. |
| `_stream.go` | SSE/chunk parsing and event emission. |
| `_types.go` | Provider wire types. |
| `_errors.go` | Error classification and response decoding. |
| `_tools.go` | Tool schema and tool-call conversion if large enough. |

Expected benefit: provider changes become localized and easier to review.

### Finding: provider-specific string checks leak into generic provider clients

Evidence:

| File | Examples |
| --- | --- |
| `nami/internal/api/anthropic.go` | Special cases `github-copilot` around lines `:54`, `:87`, `:242`, `:288`, `:304`. |
| `nami/internal/api/openai_compat.go` | Special cases `github-copilot` around lines `:46`, `:76`, `:117`, `:228`. |
| `nami/internal/api/openai_responses.go` | Special cases `github-copilot` and `codex` around lines `:66`, `:92`, `:130`, `:200`, `:250`, `:262`, `:267`. |

Opportunity: introduce provider policy/adapters for auth headers, dynamic headers, base URL resolution, schema quirks, token refresh, and max-token behavior.

Expected benefit: generic protocol clients stay generic, and provider quirks become explicit configuration instead of scattered string checks.

### Finding: model capability fallback is permissive and implicit

Evidence: `nami/internal/api/provider_config.go:296`

`ResolveModelCapabilities` falls back to model-family heuristics and then to default capabilities with tool use enabled at `:308`.

Opportunity: return capability source/confidence or `(ModelCapabilities, bool)` so unknown-model behavior is explicit at call sites.

Expected benefit: reduces silent behavior changes when model IDs are unknown.

### Finding: OpenAI-compatible tool calls can be emitted in map iteration order

Evidence: `nami/internal/api/openai_compat.go:704`

`emitToolCalls` ranges over `map[int]*openAICompatToolCallState`, which can produce nondeterministic tool-call ordering.

Opportunity: preserve ordered indexes or sort keys before emitting.

Expected benefit: deterministic output and easier debugging.

## Priority 5: Reduce Agent Loop and Retrieval Coupling

### Finding: `QueryState` groups many unrelated concerns

Evidence: `nami/internal/agent/query_stream.go:72`

`QueryState` includes messages, prompts, model settings, execution mode/profile, skills, tools, capabilities, budgets, stop state, continuation, retrieval graph, session memory, and failed-attempt entries.

Opportunity: group fields into focused sub-structs such as prompt state, model state, execution state, retrieval state, memory state, and retry state.

Expected benefit: the loop can pass narrower dependencies to helpers and tests.

### Finding: `loop.go` remains a core orchestration catch-all

Evidence: `nami/internal/agent/loop.go` is 1,012 lines.

Notable mixed areas include model recovery and auto-compaction around `:323`, model streaming around `:411`, retrieval bookkeeping later in the file, and failed-attempt recording around `:941`.

Opportunity: move model recovery, compaction recovery, retrieval bookkeeping, and attempt logging into focused files or small collaborators.

Expected benefit: clearer turn lifecycle and lower regression risk in the main loop.

### Finding: `retrieval_graph.go` combines graph model, filesystem access, parsers, scoring, imports, and tests

Evidence: `nami/internal/agent/retrieval_graph.go:104` reads file metadata and content directly, then dispatches parser logic by extension.

Opportunity: split into `graph.go`, `index.go`, `score.go`, `parsers_*.go`, `imports_*.go`, and `tests.go`.

Expected benefit: language parsing and scoring can evolve independently and receive targeted tests.

### Finding: agent package still performs direct OS and git I/O in core paths

Evidence:

| Location | Issue |
| --- | --- |
| `nami/internal/agent/context_inject.go:52` | Ignores `os.Getwd` error. |
| `nami/internal/agent/context_inject.go:107` | Runs git commands directly. |
| `nami/internal/agent/retrieval.go:402` | Reads/stats files directly. |
| `nami/internal/agent/retrieval_graph.go:108` | Stats files directly. |
| `nami/internal/agent/retrieval_graph.go:128` | Reads files directly. |

Opportunity: introduce small workspace/git/filesystem interfaces where core orchestration consumes behavior.

Expected benefit: easier tests and less coupling to live OS state.

## Priority 6: Error Handling Cleanup

### Finding: some meaningful errors are intentionally discarded

Evidence:

| Location | Discarded error |
| --- | --- |
| `nami/internal/engine/provider_behavior.go:379` | `config.Save` after token refresh. |
| `nami/internal/engine/provider_behavior.go:435` | `config.Save` after token refresh. |
| `nami/internal/engine/subagent_runtime.go:310` | `sessionStore.SaveMetadata`. |
| `nami/internal/engine/subagent_runtime.go:351` | `persistSessionState` during child message persistence. |
| `nami/internal/engine/engine_turns.go:395` | `sessionStore.SaveMetadata`. |
| `nami/internal/engine/engine_turns.go:478` | `persistSessionState`. |
| `nami/internal/engine/engine_turns.go:489` | `persistConversationHydratedPayload`. |
| `nami/internal/agent/loop.go:949` | `log.Load`. |
| `nami/internal/agent/loop.go:978` | `log.Record`. |
| `nami/internal/agent/context_inject.go:52` | `os.Getwd`. |
| `nami/internal/mcp/config.go:125` | `os.Getwd`. |

Opportunity: classify ignored errors as either safe cleanup, best-effort telemetry, or meaningful state loss.

Suggested rule:

| Error kind | Handling |
| --- | --- |
| Cleanup close after another failure | Ignore with a short comment if truly irrelevant. |
| Telemetry/debug logging | Best-effort is acceptable, but use explicit helper naming or comments. |
| Session/config/artifact persistence | Return, log, or emit a recoverable warning. |
| Runtime context discovery such as `Getwd` | Return a meaningful error unless fallback behavior is explicitly safe. |

Expected benefit: aligns with `AGENTS.md` guidance to avoid swallowing failures and improves recoverability when state persistence breaks.

## Priority 7: Keep CLI Boundary Thin

### Finding: `cmd/nami-engine/mcp.go` contains substantial business logic

Evidence: `nami/cmd/nami-engine/mcp.go` is 680 lines and includes command construction, config mutation, validation, transport parsing, listing, status loading, remove-scope resolution, and rendering.

Opportunity: move MCP command business logic into `internal/commands` or `internal/mcpconfig`, keeping `cmd` responsible for Cobra wiring and process exit behavior.

Expected benefit: matches `cmd/toolname` guidance and makes MCP config behavior testable without Cobra.

## Priority 8: Add Tests Before or During Refactors

### Finding: test coverage is very thin for core engine behavior

Evidence: only two Go test files were found.

High-value test targets before structural refactors:

| Area | Suggested tests |
| --- | --- |
| Provider stream parsing | Golden event sequences for Anthropic, Gemini, OpenAI Responses, and OpenAI-compatible streams. |
| Tool runner | Permission, batching, budget truncation, hook failure, and plan-review pause behavior. |
| Web fetch service | URL normalization, redirect policy, content-type handling, readability extraction, cache behavior. |
| LSP client | JSON-RPC framing, request validation, result normalization. |
| Retrieval graph | Parser extraction by language, import edges, test-source pairing, invalidation. |
| Session persistence | Save failures produce visible warnings or returned errors. |
| MCP CLI logic | Add/list/get/remove behavior independent of Cobra rendering. |

Expected benefit: enables incremental refactors without changing behavior accidentally.

## Suggested Execution Order

| Step | Work | Reason |
| ---: | --- | --- |
| 1 | Add characterization tests for provider streams, tool execution, and persistence error paths. | Protects high-risk behavior before moving code. |
| 2 | Introduce explicit runtime dependency structs for engine/tools without changing behavior. | Reduces global coupling and parameter explosion. |
| 3 | Split `tool_executor.go` into executor, authorization, batching, rendering, and normalization files. | Central path with high leverage and existing tests. |
| 4 | Split the largest standalone tools: `web_fetch.go`, `lsp.go`, and `background_commands.go`. | Biggest file-size wins and clearer service boundaries. |
| 5 | Split slash command handlers by domain and narrow `slashCommandContext`. | Reduces a major engine monolith and merge-conflict area. |
| 6 | Split provider implementations by request, stream, types, errors, and policy. | Makes provider behavior easier to modify safely. |
| 7 | Split retrieval graph and agent loop helpers after tests are in place. | Reduces core loop risk after coverage improves. |
| 8 | Move MCP CLI business logic out of `cmd`. | Brings CLI boundary in line with project guidance. |

## Definition of Done for This Maintainability Initiative

| Goal | Target |
| --- | --- |
| File size | No ordinary implementation file over 700 lines without an explicit reason. Prefer under 400 lines for new files. |
| Function shape | Long orchestration functions replaced by small named steps with explicit dependencies. |
| Package ownership | `engine`, `tools`, `api`, and `agent` each have narrower responsibilities and fewer hidden global interactions. |
| Error handling | Meaningful persistence/config/session errors are surfaced rather than silently discarded. |
| Tests | Core provider streaming, tool execution, retrieval graph, and persistence paths have targeted tests. |
| Behavior | Refactors are behavior-preserving unless a specific bug fix is intentionally included in a separate change. |
