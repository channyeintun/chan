# Plan

## Goal

Design a lean retrieval architecture for chan's agent harness that reduces context consumption by selecting fresh, live repository context on demand instead of storing durable code summaries.

## Reference Notes

- `chan/internal/agent/query_stream.go`: the system prompt is assembled from the base prompt, selected skills, durable memory prompt, and environment context.
- `chan/internal/agent/loop.go`: memory recall currently runs once per turn before model invocation and already cooperates with prompt-pressure checks.
- `chan/internal/agent/memory_files.go`: durable memory today is built around `AGENTS.md` plus `MEMORY.md` indexes, with recalled lines and note excerpts injected into the prompt.
- `chan/cmd/chan/memory_recall.go`: the current selector performs a model side-query over durable memory index entries, capped by candidate and selection limits.
- `chan/internal/agent/context_inject.go`: turn context already captures cwd, git branch, git status, recent commits, and a compact directory listing.
- `chan/internal/agent/context_pressure.go`: memory recall is skipped when prompt pressure is high, which should remain the guardrail for any broader retrieval step.
- `chan/cmd/chan/session_helpers.go`: compaction already exists, so retrieval must complement compaction rather than bypass it.
- `chan/internal/session/store.go` and `chan/cmd/chan/engine.go`: each session already has a dedicated session directory that can host short-lived retrieval state such as attempt logs or index caches.

## Design Principles

1. The codebase is the source of truth. Cached code summaries must never outrank live files, tests, compiler output, or current git state.
2. Reframe the problem as `working context + retrieval graph + preference store`, not as a general-purpose memory system.
3. Do not add cross-session episodic task memory in the first design.
4. Keep session scratchpad scope narrow: failed attempts, unresolved blockers, and retry guards for the current session only.
5. Prefer exact anchors and structural links over semantic similarity.
6. Retrieval must be token-budgeted, prompt-pressure-aware, and cheap to invalidate.
7. Any cached spans or snippets must be keyed to live file state so stale code is discarded immediately.

## Scope

### In Scope

1. Per-turn working context assembly.
2. A live retrieval graph over repository structure, tool output, git state, and active files.
3. Persistent preference recall for user conventions the codebase cannot tell us.
4. A session-scoped attempt log to prevent retry loops inside one task.
5. Prompt assembly, token budgeting, telemetry, and invalidation rules.

### Out Of Scope

1. Persistent code summaries.
2. Cross-session episodic memory for prior tasks or old fixes.
3. Embedding-first retrieval as the primary mechanism.
4. Automatic promotion of every session artifact into durable memory.
5. Session snapshot export/import in phase 1. This is future work and should stay separate from the core retrieval design.

## Target Architecture

1. Working context
	Keep the current prompt, recent tool output, active files, current git state, and other volatile turn data as the highest-trust context layer.
2. Retrieval graph
	Build a derived, live index over the current repository. Treat it as a query structure, not as stored truth.
3. Preference store
	Retain durable user and project preferences that are not reliably derivable from source code, such as workflow constraints or style preferences.
4. Session attempt log
	Store short-lived records of failed commands, error signatures, rejected hypotheses, and retry guards under the active session only.

## Retrieval Graph Design

### Candidate Node Types

1. File
2. Symbol
3. Test
4. Error signature
5. Tool-result artifact
6. Preference record

### Strong Edge Types

1. File contains symbol.
2. File imports file.
3. Symbol references symbol.
4. Test covers file or symbol.
5. Tool output mentions file, symbol, or test.
6. File or symbol appears in the current diff or staging area.
7. File or symbol was touched recently in the current session.
8. Preference applies to a language, tool, directory, or action type.

### Retrieval Flow Per Turn

1. Extract exact anchors from the current user request, active tool output, active files, and git diff context.
2. Resolve deterministic matches first: file paths, symbol names, test names, error strings, and known preference tags.
3. Expand one hop through strong structural edges.
4. Expand a second hop only when the first-hop set is too sparse.
5. Rank candidates by utility per token instead of pure similarity.
6. Read the top code candidates live from disk and inject snippets, not cached summaries.
7. Assemble the final prompt section under a fixed token budget.

## Ranking Model

Use a deterministic scoring pass before any optional model-assisted reranking.

1. Highest weight: exact prompt or tool-output matches.
2. High weight: current error messages, test failures, and staged or modified files.
3. Medium weight: structural neighbors such as callers, callees, imports, and adjacent tests.
4. Medium weight: recent edits within the current session.
5. Lower weight: applicable user or project preferences.
6. Penalties: extra hop distance, stale cache markers, and high token cost.

The retrieval objective is not “most related memory.” It is “highest-value live context per token.”

## Prompt Assembly Changes

1. Keep `AGENTS.md` style instructions as durable instructions.
2. Narrow durable `MEMORY.md` usage toward preference-like guidance and verified repository conventions, not code facts.
3. Insert retrieval results as live context excerpts with provenance, such as file path, symbol, or tool-output source.
4. Keep the existing pressure gate so retrieval breadth shrinks automatically when compaction pressure rises.
5. Ensure compaction and retrieval share a common token budget instead of competing independently.

## Session Attempt Log

1. Persist only session-local, observable facts: failed commands, error messages, blocked paths, and “do not retry until changed” markers.
2. Do not store hidden chain-of-thought or broad reasoning summaries.
3. Expire the entire attempt log with the session.
4. Surface attempt-log entries only when they directly prevent repeated failure inside the current task.

## Invalidation Strategy

1. On file change, rebuild graph nodes and edges only for the touched file and invalidate any cached spans keyed to the old file state.
2. On git diff or staging change, recompute the git overlay without rebuilding the entire graph.
3. On branch switch or resume into a changed worktree, lazily revalidate candidates against current file metadata or hashes before injection.
4. On session end, delete the attempt log and any session-scoped retrieval caches.
5. On preference updates, reload the small preference store directly rather than trying to merge stale entries.

## Integration Plan

1. Refactor terminology so the current durable memory path is separated into instructions, preferences, and retrieval inputs.
2. Replace the current model side-query over `MEMORY.md` index lines with a deterministic retrieval stage over live anchors and graph edges.
3. Add a session-scoped attempt log under the existing session directory layout.
4. Extend prompt assembly so live retrieval excerpts and preference matches are injected separately.
5. Add telemetry for retrieval hit sources, token cost, skipped retrieval under pressure, and repeated-failure prevention.
6. Keep model-assisted reranking as optional future work, not as the default retrieval path.

## Success Criteria

1. Lower average prompt token usage for coding turns without increasing compaction frequency.
2. Fewer irrelevant durable memory injections.
3. Faster convergence on the right files or symbols after compiler or test failures.
4. Fewer repeated failed tool attempts within a single session.
5. No known cases where stale cached code outranks live repository reads.

## Open Questions (Resolved)

1. Symbol-level indexing is included in phase 1. For most languages chan supports (Python, Rust, TS, Ruby, Java, C...) there are no dedicated symbol navigation tools. The retrieval graph provides cheap cross-references that would otherwise require expensive grep + read_file tool turns.
2. `git blame` and deeper history remain explicitly on-demand. Not a first-class retrieval source.
3. Preferences continue in `MEMORY.md` for now with preference-framed injection. A dedicated format is future work.
4. The retrieval graph is session-scoped in memory, built lazily on first file access, and invalidated per file on mod-time change. No noticeable per-turn latency for typical file counts.

## Future Task: Session Snapshot Export/Import

This is a separate future task for sharing the current session state with another person or machine. It should not be treated as memory retrieval and should not invent a synthesized handoff artifact by default.

### Purpose

1. Allow explicit sharing of the real current session state so a teammate can continue the work.
2. Preserve conversational and operational state without pretending that old session data is durable memory.
3. Keep repository truth and session truth separate: the repo still defines the code, while the snapshot defines what happened in the session.

### Core Rules

1. Export the current state as it exists; do not require the system to generate a custom handoff summary.
2. Import should create or seed a new local session rather than mutating an unrelated existing session in place.
3. Imported session data must guide retrieval and continuation, but it must never outrank live repository reads.
4. The feature should be explicit and user-invoked, not an automatic background sync.

### Candidate Snapshot Contents

1. Session metadata such as session id, timestamps, cwd, branch, and model.
2. Full transcript or a transcript manifest pointing to the session transcript file.
3. Current plan and progress artifacts when present.
4. Session-scoped attempt log entries, including failed commands and retry guards.
5. References to session artifacts such as tool logs or generated outputs.
6. Repository state metadata such as HEAD commit, dirty status, and a compact diff summary.
7. A manifest file that declares what was exported and what was intentionally omitted.

### Exclusions

1. Do not convert exported session data into durable preference memory automatically.
2. Do not export hidden chain-of-thought or internal reasoning traces.
3. Do not assume the snapshot is portable across different repository states without validation.

### Import Validation

1. Validate repository identity before restore, including project root, branch, commit, or a comparable repo fingerprint.
2. Warn clearly when the receiving worktree has diverged from the exported state.
3. Allow partial restore when artifacts are missing, but record the missing pieces in the imported manifest.
4. Keep imported state inspectable so the user can see exactly what was restored.

### Security And Redaction

1. Support redaction of secrets, tokens, and machine-specific paths before export.
2. Make it clear which files or artifacts may contain sensitive tool output.
3. Prefer a manifest-driven bundle format so users can audit contents before sharing.

### Relationship To This Design

1. Retrieval remains a per-turn mechanism for selecting live repo context.
2. Session snapshot export/import is a collaboration and continuity feature layered on top of the session store.
3. This feature should reuse existing session artifacts where possible instead of introducing a second parallel persistence model.

## Constraints

1. Do not begin implementation until this plan is accepted.
2. Do not add new long-term memory categories beyond what is described here without revisiting the trust model.
3. Any implementation must preserve the repo-first source-of-truth rule.
