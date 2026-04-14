# Progress

## Active Task

- Completed: lean retrieval architecture implementation for the agent harness.
- Completed: review-driven corrections for retrieval, preference recall, and telemetry wiring.
- Completed: structural edge expansion, test-covers edges, and attempt-log surfaced telemetry.
- Completed: architecture docs sync for structural retrieval edges and surfaced telemetry.
- Completed: session-scoped retrieval graph with multi-language support, 2-hop expansion, and retry-match telemetry.

## Notes

- Implemented session-scoped RetrievalGraph on QueryState with lazy file parsing, mod-time invalidation, and multi-hop scoring.
- Added language parsers for Go, TS/JS, Python, Rust, Ruby, Java, and C/C++ — extracting symbols, imports, and test functions via line-by-line regex.
- Multi-language test pairing: Go (_test.go), TS/JS (.test.ts/.spec.ts), Python (test_*.py/*_test.py), Ruby (_spec.rb), Java (*Test.java).
- Graph-based scoring seeds from exact anchors, walks 1-hop edges at full weight, conditionally expands 2nd hop at 50% penalty when first-hop is sparse.
- ExtractAnchors now also matches symbol names against known graph nodes for symbol-level cross-referencing.
- Graph invalidation wired into tool execution: touched files are invalidated so they re-parse next turn.
- Added attempt_repeated telemetry event emitted when a new tool failure matches a previously logged attempt-log signature.
- Added structural edge expansion (1-hop) to the retrieval pipeline: Go import edges resolve local-package imports to candidate files; test ↔ source edges associate _test.go and .test.ts files with their counterparts.
- Added attempt_log_surfaced telemetry event emitted each turn when attempt-log entries are loaded into the prompt.
- Added edges_expanded field to retrieval telemetry payload and TUI footer display.
- Added a session-scoped attempt log and wired failed tool-attempt recording into the query loop.
- Updated web docs architecture copy to describe exact-anchor retrieval, 1-hop structural expansion, preference-framed durable memory, and surfaced retrieval telemetry.
- Added live retrieval with anchor extraction, candidate scoring, live snippet reads, prompt injection, and retrieval telemetry.
- Narrowed durable memory framing toward preferences and conventions instead of repo facts.
- Shared retrieval token budgeting with context-pressure handling and wired attempt-log creation from the engine session directory.
- Replaced durable-memory model side-query with deterministic preference matching and stopped injecting unrecalled memory index entries.
- Normalized retrieval candidate paths, boosted error-context matches, and expanded touched-file tracking to use tool results and compatibility field names.
- Wired retrieval telemetry into the TUI footer so per-turn retrieval usage is visible instead of being dropped on the frontend.
- Verified the Go module builds successfully with `go build ./...`.