# Retrieval Graph Completion Plan

## Rationale

For most languages (Python, Rust, TS/JS, Ruby, Java, C/C++...) chan has zero structural navigation tools. The only way to find callers, definitions, or related files is grep + read_file — each costing a tool turn that adds 500–2000 tokens to context. A session-scoped graph gives every language cheap cross-references without tool turns, compounding context savings across multi-step tasks.

## Remaining Items from Review

### 1. Persistent Retrieval Graph (queryable structure)
**Current**: stateless per-turn scan — anchors → score → expand → read → discard.
**Target**: session-scoped in-memory graph on QueryState that survives across turns, supports multi-hop traversal, and incrementally updates when files change.

### 2. Missing Node Types
**Current**: File + Error signature only.
**Target**: File, Symbol, Test, Error signature, Tool-result artifact, Preference record.

### 3. Missing Edge Types
**Current**: file-imports-file (Go only), test↔source pairing (Go + TS only).
**Target**: file-contains-symbol, file-imports-file (multi-lang), symbol-references-symbol, test-covers (multi-lang), tool-output-mentions, diff/staging overlay, preference-applies-to.

### 4. Multi-Language Coverage
**Current**: Go imports only, test pairing for Go + TS only.
**Target**: import parsing for Go, TS/JS, Python, Rust, Ruby, Java, C/C++. Test pairing for Go (_test.go), TS/JS (.test.ts, .spec.ts), Python (test_*.py, *_test.py), Ruby (*_spec.rb), Java (*Test.java), Rust (mod tests).

### 5. 2-hop Expansion
**Current**: 1-hop only.
**Target**: expand a second hop when first-hop set is too sparse.

### 6. Retry Prevention Telemetry
**Current**: attempt_log_surfaced event emitted, but no detection of repeated failures matching logged entries.
**Target**: after tool execution, detect when a new failure matches a previously logged signature and emit attempt_repeated telemetry.

## Implementation Phases

### Phase A: Graph Data Structure (new retrieval_graph.go)

Types:
- `NodeKind`: File, Symbol, Test, Error, ToolResult, Preference
- `GraphNode`: kind + key (absolute path or symbol name) + metadata (mod time for files)
- `GraphEdge`: source key → target key + edge type + weight
- `RetrievalGraph`: node map, adjacency list, cwd, go module cache

Core methods:
- `NewRetrievalGraph(cwd)` — create empty graph
- `EnsureFile(path)` — lazily parse file on first access (or if mod time changed), extract symbols/imports/tests, register nodes + edges
- `Invalidate(path)` — remove all nodes/edges sourced from a file
- `Seed(anchors)` — register anchor nodes (files, symbols, errors)
- `Score(budget)` — walk 1-hop from seeds with full weight, optional 2-hop at 50% penalty, return scored candidates

### Phase B: Multi-Language File Parsing (in retrieval_graph.go)

Each parser extracts from file content:
1. Symbol definitions (functions, classes, types, structs, traits, interfaces)
2. Import/require statements → resolved to local file paths
3. Test functions/classes

Supported languages:
- **Go**: func/type/struct/interface declarations; import blocks; Test* functions; _test.go pairing
- **TypeScript/JavaScript**: function/class/const export declarations; import/require statements; .test.ts/.spec.ts/.test.js/.spec.js pairing
- **Python**: def/class declarations; import/from-import; test_*.py and *_test.py pairing
- **Rust**: fn/struct/enum/trait/impl declarations; use statements; #[test] functions; mod tests blocks
- **Ruby**: def/class/module declarations; require/require_relative; *_spec.rb pairing
- **Java**: class/interface/enum declarations; import statements; *Test.java pairing
- **C/C++**: function declarations; #include local headers; no special test convention

All parsers use simple line-by-line regex (no AST). Good enough for structural edges.

### Phase C: Graph-Based Scoring (update retrieval.go)

Replace `ScoreCandidates` body:
1. Ensure graph files for all anchor paths + git-modified + session-touched
2. Seed graph with anchor nodes
3. Walk 1-hop edges, weight by edge type (import=2, contains=2, test-covers=2, tool-mentions=1)
4. If <3 candidates after 1-hop, walk 2nd hop at 50% weights
5. Apply existing modifiers: exact-anchor +3, staged/modified +4, recently-touched +2, error-context +1–4
6. Return sorted candidates as before

### Phase D: Wiring + Invalidation (update loop.go, query_stream.go)

1. Add `Graph *RetrievalGraph` to `QueryState`, initialize in `NewQueryState`
2. In `runLiveRetrieval`, pass `state.Graph` to scoring
3. After `collectTouchedFiles`, call `state.Graph.Invalidate(path)` for each changed file
4. In `runLiveRetrieval`, call `state.Graph.InvalidateGitOverlay()` when git status differs from last known

### Phase E: Retry-Match Telemetry (update loop.go, protocol.go, TUI)

1. In `recordFailedAttempts`, after recording a new failure, check if same command+signature was already in the log → emit `attempt_repeated` event
2. Add `EventAttemptRepeated` + `AttemptRepeatedPayload` to IPC protocol
3. Add to TUI types + handler (no-op, telemetry only)

## File Change Summary

- **New**: `chan/internal/agent/retrieval_graph.go`
- **Edit**: `chan/internal/agent/query_stream.go` — Graph field on QueryState
- **Edit**: `chan/internal/agent/retrieval.go` — delegate scoring to graph
- **Edit**: `chan/internal/agent/loop.go` — graph wiring, invalidation, retry-match telemetry
- **Edit**: `chan/internal/ipc/protocol.go` — EventAttemptRepeated
- **Edit**: `chan/tui/src/protocol/types.ts` — attempt_repeated event
- **Edit**: `chan/tui/src/hooks/useEvents.ts` — handle event

## Constraints

- Session-scoped in memory only — no disk persistence for the graph
- No embedding or vector search
- File parsing is lazy (on first access or after invalidation)
- All parsers use line-by-line regex, not AST parsing
- Language-agnostic: supports any language with a registered parser
- Must stay under ~5ms per turn for graph operations
