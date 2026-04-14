# Lean Retrieval Architecture

This document describes the lean retrieval architecture now implemented in `chan`.

It replaces the old idea of storing durable code summaries with a repo-first retrieval path that selects fresh, live context per turn.

## Summary

Chan now uses four cooperating context layers:

1. **Working context** — current prompt, recent tool output, active session state, git state.
2. **Live retrieval graph** — a session-scoped, in-memory graph over repository structure.
3. **Preference memory** — durable user/project guidance for non-derivable conventions only.
4. **Session attempt log** — short-lived failure history to avoid repeated mistakes in one session.

The codebase remains the source of truth. Retrieval reads live files from disk and injects excerpts with provenance instead of relying on cached code summaries.

## Why this architecture

The old memory-heavy shape risks wasting tokens on stale or low-value code facts.

The new design optimizes for:

- exact anchors over vague recall
- live repository reads over cached summaries
- structural expansion over embedding-first search
- token budget discipline
- fast invalidation when files change
- session-local retry prevention

## Core design rules

1. **Repo first.** Source files, tests, tool output, and git state outrank any stored note.
2. **No durable code summaries.** Persistent memory is for preferences and conventions, not repo facts.
3. **Session-scoped graph.** Structural retrieval state lives in memory for the active session only.
4. **Pressure aware.** Retrieval shrinks or skips when prompt pressure rises.
5. **Cheap invalidation.** Touched files are re-parsed when their mod time changes.

## Main components

### 1. Query state owns a session graph

`QueryState` now carries a session-scoped `RetrievalGraph`.

- `chan/internal/agent/query_stream.go`
- `NewQueryState(...)` initializes `Graph: NewRetrievalGraph(ctx.CurrentDir)`

This means the graph persists across turns within one session instead of being rebuilt from scratch every turn.

### 2. Per-turn live retrieval

The retrieval pass runs during the query loop before model invocation.

- `chan/internal/agent/loop.go`
- `runLiveRetrieval(...)`

Per turn it does this:

1. gather recent tool output
2. extract exact anchors from user prompt, git status, and tool output
3. score candidate files through the retrieval graph
4. read top snippets live from disk
5. inject a `<live_context>` section into the prompt

The injected content is file-backed and attributed:

```xml
<live_context>
<file path="/abs/path/to/file.go" source="exact anchor">
...
</file>
</live_context>
```

### 3. Retrieval graph

The graph is implemented in:

- `chan/internal/agent/retrieval_graph.go`

It is a session-scoped in-memory index over repository structure.

#### Node types

- file
- symbol
- test
- error signature
- tool-result artifact
- preference record

#### Edge types

- file contains symbol
- file imports file
- symbol references symbol
- test covers file/symbol
- tool output mentions file/symbol
- diff/staging overlay
- session touched file
- preference applies

#### Supported languages

The graph lazily parses multiple languages with lightweight regex-based structure extraction:

- Go
- TypeScript / JavaScript
- Python
- Rust
- Ruby
- Java
- C / C++

It extracts symbols, import-like edges, and common test relationships.

## Retrieval flow

### Anchor extraction

`chan/internal/agent/retrieval.go`

Anchors come from exact signals, especially:

- file paths in the prompt or tool output
- symbol names recognized from known graph nodes
- error signatures from failing commands/tool output
- modified files from git status
- files touched earlier in the session

### Graph scoring

The scorer prefers deterministic, high-signal context.

Strong signals include:

- exact file anchors
- staged or modified files
- recent session-touched files
- current error context
- structural 1-hop neighbors

If the first hop is sparse, retrieval expands a second hop with a penalty instead of exploding context size.

### Live snippet reads

After ranking, Chan reads only the top candidate files live from disk and injects small excerpts.

Key limits in `retrieval.go`:

- soft retrieval budget: about `3000` tokens
- max scored candidates: `24`
- max injected snippets: `4`
- max snippet bytes per file: `2000`

## Durable memory is narrowed to preferences

Durable memory still exists, but its role is narrower.

- `chan/internal/agent/memory_files.go`

`AGENTS.md` remains instruction-like context.

`MEMORY.md` recall is now framed as **preferences and conventions** only:

- workflow constraints
- style decisions
- non-derivable project guidance

The prompt text explicitly tells the model to treat these as selective guidance and to verify code facts against the live repository.

That is the key architectural shift: durable memory no longer acts as the main source for code understanding.

## Session attempt log

Chan also keeps a short-lived session attempt log.

Purpose:

- record failed commands and error signatures
- surface relevant failures back into the prompt
- prevent repeated retry loops in the same task

This is session-local state, not durable memory.

It is especially useful when the model keeps retrying the same failing edit or command with stale assumptions.

## Context pressure and shared budget

`chan/internal/agent/context_pressure.go`

Live retrieval shares the same pressure gate used by memory recall.

When context gets hot:

- memory recall can be skipped
- live retrieval can also be skipped
- retrieval budget can be reduced
- compaction remains the higher-priority safety valve

This keeps retrieval and compaction from competing independently for tokens.

## Invalidation model

The graph is designed to be cheap to refresh.

### File invalidation

When a file changes:

- its graph nodes/edges are invalidated
- the next access re-parses the file lazily

### Git overlay invalidation

When git status changes:

- diff/staging edges are rebuilt without rebuilding the whole graph

### Session scope

The graph is in memory only for the active session.

No disk persistence. No durable code cache. No cross-session graph state.

## Telemetry

The retrieval system emits explicit runtime telemetry.

Defined in:

- `chan/internal/ipc/protocol.go`

Key events:

- `retrieval_used`
  - snippet count
  - tokens used
  - anchor count
  - edges expanded
  - skipped flag
- `attempt_log_surfaced`
  - entry count
  - tokens used
  - whether attempt-log content was injected
- `attempt_repeated`
  - emitted when a new tool failure matches a prior logged failure

This telemetry is surfaced to the TUI so retrieval activity is visible, not hidden.

## Relationship to the web docs diagrams

The architecture section in `web/docs.html` already reflects this design.

It shows two complementary views:

1. **Core Harness** — TUI, Go engine, retrieval, memory, planner, tools, and LLM infrastructure.
2. **Per-Turn Retrieval Pipeline** — input signals → anchor extraction → session graph → scoring/budgeting → LLM, with invalidation and pressure gating.

Related assets:

- `web/docs.html`
- `docs/architecture.png`

The important conceptual mapping is:

- **Prompt / Git Diff / Tool Logs** feed anchor extraction
- **Session Graph** persists across turns
- **Scoring** ranks context under a token budget
- **Invalidation** refreshes graph structure after changes
- **Pressure Gate** reduces or skips retrieval under context pressure

## What this architecture is not

This system is intentionally **not**:

- a vector database
- an embedding-first retrieval system
- a durable store of code summaries
- a cross-session episodic task memory system

It is a lightweight, repo-first retrieval layer for selecting the highest-value live context per token.

## Key source files

- `chan/internal/agent/query_stream.go` — session state owns the graph
- `chan/internal/agent/loop.go` — runs retrieval and emits telemetry
- `chan/internal/agent/retrieval.go` — anchor extraction, scoring entrypoint, live snippet reads
- `chan/internal/agent/retrieval_graph.go` — graph structure, parsing, invalidation, scoring expansion
- `chan/internal/agent/context_pressure.go` — pressure gating and retrieval budget
- `chan/internal/agent/memory_files.go` — preference-framed durable memory recall
- `chan/internal/ipc/protocol.go` — retrieval and attempt-log telemetry events

## Final takeaway

Chan’s lean retrieval architecture works by combining:

- live repo reads
- exact anchors
- a session-scoped structural graph
- preference-only durable memory
- session-local retry prevention
- token-aware prompt assembly

The result is lower-trust durable memory, higher-trust live context, and a retrieval path that stays fast, cheap to invalidate, and aligned with the current repository state.