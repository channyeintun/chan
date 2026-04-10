# Progress

## 2026-04-10

- Referenced upstream sourcecode input behavior before implementation, primarily `sourcecode/hooks/useArrowKeyHistory.tsx` and `sourcecode/components/PromptInput/PromptInputFooter.tsx`.
- Ported a sourcecode-inspired TUI input improvement into `go-cli/tui`: prompt history on Up/Down, persistent draft restore, placeholder text, and a footer hint row for primary shortcuts.
- Kept history state in the app layer via `usePromptHistory` so it survives temporary input unmounts such as permission prompts.
- Validated the TypeScript TUI build after the change.

# go-cli — Implementation Progress

## Project Setup

- [x] Go module initialized (go1.26.1, `github.com/channyeintun/go-cli`)
- [x] Full directory structure created
- [x] Cobra dependency added
- [x] `.gitignore` configured
- [x] Build + vet passing clean

---

## Week 1–2: MVP Core

### `internal/api/` — LLM Client + Streaming

- [x] `client.go` — LLMClient interface, ModelRequest, ModelEvent, Usage types
- [x] `provider_config.go` — 9 provider presets (Anthropic, OpenAI, Gemini, DeepSeek, Qwen, GLM, Mistral, Groq, Ollama)
- [x] `retry.go` — APIError classification, exponential backoff, RetryWithBackoff
- [x] `anthropic.go` — Anthropic Messages API streaming client
- [x] `openai_compat.go` — OpenAI-compatible streaming client
- [x] `openai_compat.go` — Surface OpenRouter upstream provider messages from nested error metadata instead of showing only generic wrapper errors
- [x] `gemini.go` — Gemini native streaming client
- [x] `ollama.go` — Ollama local model client

### `internal/agent/` — Query Engine

- [x] `query_stream.go` — iter.Seq2-based QueryStream, QueryDeps, QueryState, 5-phase runIteration skeleton
- [x] `modes.go` — ExecutionMode (plan/fast), ExecutionProfile with ProfileForMode
- [x] `token_budget.go` — ContinuationTracker with diminishing returns logic
- [x] `context_inject.go` — SystemContext (session-stable) + TurnContext (per-turn refresh)
- [x] `loop.go` — Wire real model calls into the 5-phase iteration
- [x] `planner.go` — Plan creation + enforcement before writes
- [x] `planner.go` — Persist implementation-plan artifacts only for actual planning turns (skip simple explanatory questions in plan mode)

### `internal/tools/` — Tool Execution

- [x] `interface.go` — Tool interface, PermissionLevel, ToolInput/ToolOutput
- [x] `registry.go` — Tool registry with Get/List/Definitions
- [x] `orchestration.go` — Dynamic concurrency classification, PartitionBatches, ExecuteBatch
- [x] `budgeting.go` — ResultBudget, ApplyBudget with disk spillover
- [x] `bash.go` — Bash tool with security validation
- [x] `file_read.go` — File read tool
- [x] `file_write.go` — File write tool
- [x] `file_edit.go` — File edit tool
- [x] `glob.go` — Glob tool
- [x] `grep.go` — Ripgrep wrapper tool
- [x] `web_search.go` — Web search tool (DuckDuckGo-backed with domain filters)
- [x] `web_fetch.go` — Web fetch tool (URL validation, HTTPS upgrade, redirect limits, HTML→markdown, in-memory cache)
- [x] `git.go` — Structured read-only git tool (`status`/`diff`/`log`/`show`/`branch`/`blame`)
- [x] `streaming_executor.go` — Start read-safe tools early, enforce exclusive barriers, deliver results in original order

### `internal/utils/`

- [x] `tokens.go` — Token estimation (~4 chars/token)
- [x] `messages.go` — Message normalization (consolidate consecutive, strip whitespace)

---

## Week 3: Security & Awareness

### `internal/permissions/`

- [x] `gating.go` — Rule-based permission context (allow/deny/ask), Decision check
- [x] `bash_rules.go` — ZSH dangerous commands blocklist, destructive command patterns, read-only classifier
- [x] Wire permissions into tool executor

### `internal/agent/`

- [x] `context_inject.go` — Two-layer injection implemented
- [x] Wire context injection into query loop (per-turn refresh)

### `internal/cost/`

- [x] `tracker.go` — Per-model token/cost/duration tracking, thread-safe Snapshot
- [x] Wire into API client (record after every call)
- [x] Wire into tool executor (record tool duration)

---

## Week 4–5: Compaction

### `internal/compact/`

- [x] `tokens.go` — Thresholds (autocompact 13k buffer, warning 20k, circuit breaker 3)
- [x] `pipeline.go` — Pipeline skeleton with 3-strategy cascade
- [x] `tool_truncate.go` — Strategy A: tool result truncation (microcompact)
- [x] `summarize.go` — 9-section compaction prompt template
- [x] Strategy B implementation: call LLM/local model for summarization
- [x] Strategy C implementation: partial compaction scoped to recent messages
- [x] `sliding_window.go` — Sliding window strategy for preserving prior summaries
- [x] Auto-compaction trigger logic wired into query loop
- [ ] Tests for each strategy

---

## Week 6: Interface & Configuration

### `internal/ipc/`

- [x] `protocol.go` — StreamEvent (18 event types), ClientMessage (6 message types), all typed payloads
- [x] `bridge.go` — NDJSON reader/writer, EmitEvent, EmitReady, EmitError

### `cmd/go-cli/`

- [x] `main.go` — Cobra entrypoint, `--stdio`/`--model`/`--mode` flags, NDJSON event loop
- [x] Wire query engine into the event loop (replace stub response)
- [x] Slash command dispatch (`/plan`, `/fast`, `/compact`, `/model`, `/cost`, `/resume`)
  - Also implemented: `/usage`, `/plan-mode`, `/model default`
  - Also implemented: `/clear`, `/help`, `/status`, `/sessions`, `/diff`

### `internal/config/`

- [x] `config.go` — File + env config loading, ParseModel, Save

### `internal/skills/`

- [x] `loader.go` — Two-directory discovery (~/.config/go-cli/agents/ + .agents/)
- [x] `frontmatter.go` — YAML frontmatter parser
- [x] Wire skills into system prompt injection

### `internal/hooks/`

- [x] `types.go` — 9 hook types, Payload, Response
- [x] `runner.go` — Shell script hook executor (~/.config/go-cli/hooks/)
- [x] Wire hooks into tool execution lifecycle (pre_tool_use / post_tool_use / session_start)
- [ ] Wire hooks into compaction lifecycle

### `internal/session/`

- [x] `store.go` — NDJSON transcript persistence, metadata save/load, ListSessions
- [x] `restore.go` — Resume conversation/model/mode state from transcript + metadata
- [x] Wire session save into query loop
- [x] `title.go` — Session title generation via local model (async after first query)

### `internal/artifacts/`

- [x] `types.go` — 10 artifact kinds, Scope (session/user), Artifact/ArtifactVersion/ArtifactRef
- [x] `service.go` — Service interface (Save/Load/List/Delete/Versions)
- [x] `store.go` — LocalStore filesystem implementation with markdown version history
- [x] `manager.go` — Markdown artifact lifecycle manager for implementation plans and tool logs
- [x] Wire artifacts into tool budgeting spillover
- [x] Wire artifacts into planning mode

### `tui/` — Ink Frontend

- [x] `package.json` — React 19, Ink 7, TypeScript 6
- [x] `tsconfig.json`
- [x] `src/index.tsx` — Entry point
- [x] `src/App.tsx` — Top-level layout + event dispatch
- [x] `src/components/Input.tsx` — Text input + Tab toggle + slash commands
- [x] `src/components/StreamOutput.tsx` — Streaming text output
- [x] `src/components/StatusBar.tsx` — Mode, model, cost display
- [x] `src/components/PermissionPrompt.tsx` — y/n/a approval
- [x] `src/components/ToolProgress.tsx` — Tool execution indicator
- [x] `src/hooks/useEngine.ts` — Spawn Go child, NDJSON I/O
- [x] `src/hooks/useEvents.ts` — StreamEvent → React state
- [x] `src/protocol/types.ts` — Mirrors Go IPC types
- [x] `src/protocol/codec.ts` — NDJSON parser/serializer
- [x] `src/components/PlanPanel.tsx` — Render implementation-plan artifact
- [x] `src/components/ArtifactView.tsx` — Render artifact content
- [x] Conversation transcript retained after submit; live assistant row now shows thinking/responding state instead of clearing the visible prompt
- [x] Status bar labels mode/model explicitly and tool progress uses a real spinner
- [x] Assistant spinner now appears immediately on submit, before the first streamed token or thinking delta arrives
- [x] Install guide now documents `install -m 755` for manual binary installation and explains why it is used
- [x] `npm install` + TypeScript build verification

---

## Phase 2a: Local Model (Post-MVP)

### `internal/localmodel/`

- [x] `runner.go` — Ollama auto-detection, NewLocalModel
- [x] `router.go` — Task-based routing (compaction/scoring/title → local, reasoning → remote)
- [x] Implement Query() method (POST to Ollama /api/generate)
- [x] Wire into compact/summarize.go
- [ ] Wire into session title generation

---

## Phase 2b: Multi-Model Support (Post-MVP)

- [x] Finalize LLMClient with Capabilities()
- [x] `anthropic.go` — Full streaming implementation
- [x] `openai_compat.go` — SSE parser, function calling
- [x] `gemini.go` — Native streaming, function declarations
- [x] `ollama.go` — Local chat streaming implementation
- [x] `/model` runtime switching
- [x] Capability-aware engine adjustments

---

## Phase 3: Enhancements (Source Code Parity)

### `internal/agent/` — CLAUDE.md / Memory File Loading

- [x] `memory_files.go` — Discover and load instruction files (CLAUDE.md, .claude/CLAUDE.md, .claude/rules/\*.md, CLAUDE.local.md) walking from cwd to root
- [x] User memory: ~/.claude/CLAUDE.md
- [x] Project memory: directory hierarchy walk with priority ordering
- [x] Local memory: CLAUDE.local.md (untracked project instructions)
- [x] Wire into SystemContext and system prompt composition

### `internal/session/` — Session Title Generation

- [x] `title.go` — Generate 3-7 word titles via local model (Ollama)
- [x] Async title generation after first successful query
- [x] Title persisted into session metadata

### `internal/hooks/` — Lifecycle Hook Wiring

- [x] session_start hook fires at engine boot
- [x] pre_tool_use hook fires before each approved tool call (can deny)
- [x] post_tool_use hook fires after successful tool results

### `cmd/go-cli/` — New Slash Commands

- [x] `/clear` — Clear conversation and start new session
- [x] `/help` — Show all available slash commands
- [x] `/status` — Show session ID, elapsed, mode, model, message count, cost
- [x] `/sessions` — List recent sessions with metadata
- [x] `/diff` — Show git diff (with optional args like --staged)

### `internal/tools/` — File History Tracking

- [x] `file_history.go` — SHA-256 content-addressed backup store, snapshot/rewind support
- [x] Track file state before write/edit operations
- [x] Snapshot creation and rewind to any checkpoint
- [x] Diff stats between snapshot and current state
- [x] Wire into file_write and file_edit tools via global tracker

---

## Summary

| Area           | Scaffolded                 | Wired/Working                                                                                              |
| -------------- | -------------------------- | ---------------------------------------------------------------------------------------------------------- |
| IPC Protocol   | ✅                         | ✅                                                                                                         |
| API Interfaces | ✅                         | ⚠️ (Anthropic + OpenAI-compatible + Gemini + Ollama clients implemented)                                   |
| Agent Loop     | ✅                         | ✅ (live turn loop with model streaming and tool handoff)                                                  |
| Tools          | ✅ (framework)             | ⚠️ (bash + file read/write/edit/glob/grep implemented; remaining tools pending)                            |
| Compaction     | ✅ (Strategies A+B+C done) | ⚠️ (proactive compaction now wired; tests remain pending)                                                  |
| Permissions    | ✅                         | ✅ (stdio permission prompts + session allow rules)                                                        |
| Cost Tracking  | ✅                         | ✅ (API usage, token totals, tool duration, TUI updates)                                                   |
| Hooks          | ✅                         | ✅ (pre/post tool + session_start wired; compaction hooks pending)                                         |
| Artifacts      | ✅                         | ✅ (markdown-backed plan artifacts + tool-log spillover wired)                                             |
| Session        | ✅                         | ✅ (live save + restore + title generation wired)                                                          |
| Config         | ✅                         | ✅                                                                                                         |
| Skills         | ✅                         | ✅ (auto-select matching skills and inject their markdown instructions per turn)                           |
| Local Model    | ✅                         | ✅ (Ollama Query + compaction routing + session title generation wired)                                    |
| Ink TUI        | ✅                         | ✅ (default CLI launches Ink parent, Go child over NDJSON; status/permission/artifact rendering validated) |
| CLI Entrypoint | ✅                         | ✅ (live stdio engine)                                                                                     |
| Memory Files   | ✅                         | ✅ (CLAUDE.md + .claude/rules + CLAUDE.local.md hierarchy loading wired)                                   |
| File History   | ✅                         | ✅ (content-addressed backup + snapshot/rewind wired into file write/edit)                                 |

**Current state:** All four provider clients, the Bash tool, and the file read/write/edit/glob/grep/web_search/web_fetch/git tools are implemented, along with the streaming executor needed to overlap safe tool calls. The default CLI path now launches the Ink frontend as the parent process and runs the Go engine as a stdio child over NDJSON, with status, artifact, compaction, permission/error states, preserved conversation history, and live assistant/tool activity rendered in the TUI while the engine remains recoverable if the configured model is unavailable at startup. The stdio engine persists and restores transcript + session metadata, generates session titles via local model after the first query, supports runtime `/model` switching, exposes `/plan`, `/fast`, `/compact`, `/model`, `/cost`, `/usage`, `/resume`, `/clear`, `/help`, `/status`, `/sessions`, and `/diff` over the stdio command path, emits markdown-backed implementation-plan/tool-log artifacts during planning and oversized tool execution, keeps plan mode read-only through planner enforcement, loads CLAUDE.md/CLAUDE.local.md/.claude/rules/\*.md project instruction files into the system prompt, tracks file edit history for undo/rewind with content-addressed backups, fires pre/post-tool and session lifecycle hooks from ~/.config/go-cli/hooks/, and now shapes requests by model capability: native tool definitions are withheld for text-only models, `ultrathink` only enables extended thinking on supported models, context thresholds already track each model's window, and tool-output budgets scale with model output capacity.
