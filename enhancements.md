# Nami Enhancement Findings

Scope: review focused on the Go engine side: `cmd/nami-engine`, `internal/engine`, `internal/agent`, `internal/tools`, `internal/api`, `internal/session`, `internal/artifacts`, and MCP/subagent runtime paths. No implementation changes were made.

## Highest-impact opportunities

### 1. Replace process-global tool runtime state with session-scoped dependencies

Impact: high correctness and maintainability improvement. Nami currently appears built for one active engine process, but several tool runtimes are process-global. That makes runtime behavior harder to reason about, makes concurrent embedded use risky, and creates subtle coupling between parent sessions, restored sessions, subagents, and background work.

Evidence:

- `internal/engine/engine.go` installs global state with `SetGlobalFileHistory`, `SetGlobalFileReadState`, `SetGlobalSessionArtifacts`, `SetGlobalSwarmRuntime`, `SetGlobalMCPManager`, `SetSessionControlRuntime`, `SetToolSearchRuntime`, and background command notifier hooks.
- `internal/tools/background_commands.go` and `internal/engine/subagent_background.go` keep process-global maps for background commands, agents, and teams.
- `internal/engine/subagent_runtime.go` creates child registries from the parent registry but tools still have access to global runtime hooks.

Suggested path:

- Introduce an explicit `EngineRuntime` or `ToolRuntime` struct containing session ID, artifact manager, MCP manager, file history, file-read state, command/agent registries, and notification sinks.
- Attach runtime to the registry or tool execution context instead of package-level setters/getters.
- Keep narrow package-level fallbacks only where needed for CLI compatibility, then migrate built-in tools to use injected runtime first.

### 2. Make startup progressive instead of blocking on all MCP discovery

Impact: high perceived performance improvement, especially for projects with slow or unavailable MCP servers.

Evidence:

- `RunStdioEngine` starts `mcpManager.Start(ctx)` before emitting ready.
- `mcp.Manager.Start` launches server connects concurrently but waits for all enabled servers to finish before returning.
- Default MCP discovery and stdio startup timeouts are seconds-long, so one slow server can delay engine readiness even though core chat/tools could already be available.

Suggested path:

- Emit `ready` after core tools/session state are initialized.
- Start MCP discovery asynchronously and emit capability/tool updates as servers connect or fail.
- Preserve a deterministic status event so the TUI can show "MCP still connecting" rather than making the whole engine feel cold.

### 3. Use true streaming tool execution for faster first results

Impact: high latency improvement for multi-tool turns. The code has a `StreamingExecutor`, but the active path still resolves/approves all calls first, partitions batches, waits for each batch, and only then handles results in order.

Evidence:

- `internal/tools/streaming_executor.go` supports adding calls as they arrive and yielding contiguous completed results.
- `internal/engine/tool_executor.go` currently calls `prepareToolCalls` for the whole model tool-call list, then `executeApprovedToolBatches`, then processes batch results.
- `internal/tools/orchestration.go` has parallel batches, but result emission still waits for the batch call to return.

Suggested path:

- Integrate `StreamingExecutor` into `executeToolCalls` so approved read-only calls can start while later calls are still being validated or awaiting permission.
- Emit `EventToolResult` as each ordered result becomes available.
- Keep serial/write tools as ordering barriers, preserving current safety semantics.

### 4. Bound and cache turn-context collection on the hot path

Impact: medium-high latency and reliability improvement. Every query iteration refreshes git status, branch, recent log, and a directory listing. Those commands have no timeout and silently degrade on errors.

Evidence:

- `agent.NewQueryState` calls `LoadTurnContext`.
- `runIteration` refreshes `state.TurnContext = LoadTurnContext()` on every iteration.
- `LoadTurnContext` shells out to `git branch`, `git status`, and `git log`, then scans the working directory.
- `gitCommand` uses `exec.Command(...).Output()` without a context deadline.

Suggested path:

- Add a small context collector with timeouts, caching, and invalidation keyed by cwd plus recent tool writes.
- Reuse the same turn context within one user turn unless a write/execute tool marks it stale.
- Surface context collection latency/errors in timing logs rather than silently dropping fields.

### 5. Make persistence atomic and less write-heavy during active turns

Impact: medium-high durability and responsiveness improvement. Session persistence currently rewrites multiple files repeatedly during a turn, and writes are direct rather than atomic.

Evidence:

- `PersistMessages` in `engine_turns.go` calls `persistCurrentMessages` after query-loop updates.
- `persistCurrentMessages` saves metadata, transcript, and hydrated timeline.
- `session.Store.SaveTranscript`, `SaveMetadata`, and `SaveConversationTimeline` write directly to final paths.
- Artifact versions in `artifacts.LocalStore.Save` write content and metadata directly to final files.

Suggested path:

- Use temp-file-plus-rename for metadata, transcripts, timelines, and artifact metadata/content.
- Consider an append-only transcript journal during active turns, then compact/rewrite at stable checkpoints.
- Batch timeline hydration writes around user-visible checkpoints instead of every persisted message update.

### 6. Consolidate provider transport behavior

Impact: medium-high provider correctness improvement. Provider adapters share many responsibilities: request construction, retry classification, streaming SSE parsing, usage tracking, tool-call extraction, and fallback stop behavior.

Evidence:

- `internal/api/anthropic.go`, `openai_responses.go`, `openai_compat.go`, `gemini.go`, and `ollama.go` each implement stream opening and parsing behavior.
- `internal/api/retry.go` centralizes retry policy, but provider classification and stream completion semantics still depend on each adapter.
- Provider stream behavior is central to engine correctness because missing stop events or malformed tool calls can leave the agent loop in the wrong state.

Suggested path:

- Extract shared request setup, HTTP retry handling, SSE read behavior, and terminal-stop normalization where provider behavior is genuinely identical.
- Keep provider-specific translation isolated in small parser/mapper functions so differences stay explicit.
- Add richer debuglog metadata around provider stream lifecycle: request opened, first event, usage received, terminal stop reason, and stream end without stop.

### 7. Expand timing analysis into actionable latency budgets

Impact: medium operational visibility improvement. Nami already has timing logs and a compaction summary, but the most valuable engine bottlenecks are broader than compaction.

Evidence:

- `internal/timing/analysis.go` currently focuses recommendations on compaction outcomes.
- `engine.go` and `engine_turns.go` record startup and turn checkpoints, plus tool budget metrics.
- Startup, context collection, model warmup, MCP discovery, first-token latency, first-tool-result latency, persistence latency, and title generation all affect perceived responsiveness.

Suggested path:

- Extend timing summaries to include p50/p95 for startup-to-ready, MCP discovery, first token, first tool result, tool batch duration, persistence, and compaction.
- Emit recommendations tied to thresholds, for example slow MCP startup, repeated context collection cost, or low-value compactions.
- Add a `nami-engine timing summary --latest` convenience path so these metrics are easy to inspect while tuning.

## Recommended order

1. Make MCP startup progressive to improve first-run responsiveness with limited risk.
2. Move process-global tool runtime state toward session-scoped runtime injection.
3. Integrate streaming tool execution for early result delivery.
4. Add bounded/cached turn-context collection and atomic persistence improvements.
5. Consolidate shared provider transport behavior after differences are documented in debug logs.
