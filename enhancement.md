# Archived Research Note

The 2026-04-12 VS Code Copilot Chat comparison research that seeded the enhancement roadmap is no longer an active planning baseline.

- Shipped outcome: see `release-note-v2.md`
- Post-ship review: see `review.md`
- Active research baseline: none

If future comparison work starts, create a new research note instead of extending this archived document.

- `file_read` already supports line ranges, which is good
- It still under-communicates what the model should do after a partial read
- Binary-awareness would reduce confusing output on non-text inputs

Recommended direction:

- Keep line-based reads
- Add continuation hints for large/partial reads
- Add binary detection and a safer fallback output mode

### 4. Add path-sensitive edit safety heuristics

Reference files:

- `src/extension/tools/node/editFileToolUtils.tsx`
- `src/extension/tools/node/createFileTool.tsx`

What is worth copying:

- Treating some paths as higher-risk than normal workspace files
- Consistent diff generation before confirmation
- Stronger guardrails around config, dotfiles, and user-home edits

Why it fits `gocode`:

- Current path resolution blocks cwd escape, which is necessary but not sufficient
- Risky local files should trigger stronger approval behavior than ordinary source files

Recommended direction:

- Add a risk tier for dotfiles, editor config, shell rc files, and user-home paths
- Ensure write approvals always include a stable diff preview

### 5. Separate create, overwrite, and edit intent

Reference files:

- `src/extension/tools/node/createFileTool.tsx`
- `src/extension/tools/node/replaceStringTool.tsx`
- `src/extension/tools/node/multiReplaceStringTool.tsx`

What is worth copying:

- Creation is treated as a distinct operation from editing an existing file.
- Replacement tools are not overloaded with file-creation semantics.
- The tool contract makes existence expectations clearer.

Why it fits `gocode`:

- `file_write` currently acts as both create and overwrite.
- `file_edit` can create a file when `old_string` is empty.
- That flexibility is convenient but makes failure and approval policy less explicit.

Recommended direction:

- Either split creation into a separate tool or add an explicit overwrite policy to `file_write`.
- Remove or de-emphasize implicit file creation from `file_edit`.

### 6. Surface post-edit diagnostics

Reference files:

- `src/extension/tools/node/abstractReplaceStringTool.tsx`
- `src/extension/tools/node/editFileToolUtils.tsx`

What is worth copying:

- After an edit, collect changed-file diagnostics when the environment can provide them.
- Return diagnostics as part of the tool result instead of requiring a separate debugging turn.

Why it fits `gocode`:

- Diff previews are useful, but they do not tell the model whether the edit broke the file.
- This is one of the biggest practical robustness wins from the VS Code implementation.

Recommended direction:

- Add optional post-edit diagnostics to `file_write`, `file_edit`, `multi_replace_file_content`, and future `apply_patch`.

## Best subagent orchestration takeaways to adopt

### 1. Use named specialist subagents with fixed tool envelopes

Reference files:

- `src/extension/agents/vscode-node/agentTypes.ts`
- `src/extension/agents/vscode-node/exploreAgentProvider.ts`
- `src/extension/agents/vscode-node/planAgentProvider.ts`

What is worth copying:

- Agents declare role, tool set, and model policy separately from runtime loop code
- Explore-style agents stay read-heavy and cheap by design
- Tool access is part of the agent contract, not an afterthought

Why it fits `gocode`:

- `gocode` already has `explore` and `general-purpose`
- The next step should be stricter specialization, not more agent types

Recommended direction:

- Keep the agent set small
- Pin each subagent type to an explicit tool allowlist and model preference
- Do not add teams, swarm behavior, or remote orchestration

### 2. Propagate one stable invocation id across the whole child trajectory

Reference files:

- `src/extension/tools/node/searchSubagentTool.ts`
- `src/extension/tools/node/executionSubagentTool.ts`
- `src/extension/chatSessions/vscode-node/test/chatHistoryBuilder.spec.ts`

What is worth copying:

- One child invocation id links the parent tool call, child session, and child tool calls
- Nested tool activity stays attributable in logs and UI

Why it fits `gocode`:

- Current `agent` results are usable, but lineage should be first-class across background status, transcripts, and tool events

Recommended direction:

- Assign one invocation id at child launch
- Propagate it through transcript events, tool events, and status polling

### 3. Reuse the same loop abstraction for parent and child agents

Reference files:

- `src/extension/prompt/node/searchSubagentToolCallingLoop.ts`
- `src/extension/prompt/node/executionSubagentToolCallingLoop.ts`
- `src/extension/intents/node/toolCallingLoop.ts`

What is worth copying:

- Parent and child agents share one orchestration model
- Limits, hooks, telemetry, and prompt-building behavior stay aligned
- Child context remains isolated from parent budget accounting

Why it fits `gocode`:

- This reduces feature drift between the main agent loop and child-agent execution
- It keeps compaction, budgeting, and recovery behavior consistent

Recommended direction:

- Make child execution explicitly reuse the `internal/agent` loop contracts
- Keep child message history and token budgets isolated from the parent session

### 4. Add subagent start/stop hooks, including block-stop reasons

Reference files:

- `src/extension/intents/node/toolCallingLoop.ts`
- `src/platform/chat/common/chatHookService.ts`

What is worth copying:

- A start hook can inject additional context into the child run
- A stop hook can block completion if exit criteria are not met
- Block reasons are surfaced back into the next loop turn instead of disappearing

Why it fits `gocode`:

- This adds a policy surface without hardcoding workflow rules into the loop itself
- It gives child agents a cleaner definition of done

Recommended direction:

- Add optional subagent start and stop hooks
- Surface block reasons in transcript state and child status output
- Keep hooks local and synchronous in the first pass

### 5. Return structured child metadata, not only final text

Reference files:

- `src/extension/tools/node/searchSubagentTool.ts`
- `src/extension/tools/node/executionSubagentTool.ts`

What is worth copying:

- Child tools emit readable invocation and completion messages
- Tool metadata carries role, description, and invocation linkage
- The final textual summary is only one layer of the result

Why it fits `gocode`:

- `agent_status` and the TUI can present more useful state than running/done
- Background child work becomes debuggable without dumping full transcripts by default

Recommended direction:

- Extend child result payloads with phase, active tool, and last meaningful event
- Keep the final report concise but preserve structured metadata for the UI

## What not to copy

- agent management UI and wizard flows
- handoff buttons between agents
- teams, swarm, remote agents, or remote execution
- notebook-specific editing work
- broad tool parity for its own sake

## Recommended implementation order

1. File semantics and safety hardening
   - separate create, overwrite, and edit intent
   - improve `file_read` continuation and binary handling
   - add risk-tiered approval without weakening cwd containment
2. Edit engine hardening
   - add `apply_patch`
   - unify edit failure taxonomy and recovery hints
   - add post-edit diagnostics
3. Subagent lineage
   - add invocation ids across child session, tool calls, and status APIs
   - surface structured child metadata to the TUI
4. Shared child lifecycle
   - align child execution with the main loop contracts
   - add start/stop hooks and block-stop reasons

## Likely `gocode` touch points

- `gocode/internal/tools/file_edit.go`
- `gocode/internal/tools/file_write.go`
- `gocode/internal/tools/multi_replace_file_content.go`
- `gocode/internal/tools/file_diff_preview.go`
- `gocode/internal/tools/file_history.go`
- `gocode/internal/tools/file_read.go`
- `gocode/internal/tools/path_resolution.go`
- `gocode/internal/tools/validation.go`
- `gocode/internal/tools/agent.go`
- `gocode/internal/agent/loop.go`
- `gocode/internal/agent/query_stream.go`
- `gocode/internal/ipc/`
- `gocode/tui/src/`

## Decision

- Treat VS Code Copilot Chat as a source of patterns, not a parity target.
- For file tools, the goal is to be more robust than VS Code Copilot Chat where possible, and intentionally narrower where strict local safety is the better tradeoff.
- The best imports for `gocode` are patch-grade editing, classified edit recovery, binary-aware reads, risk-tiered write safety, post-edit diagnostics, and tighter subagent lifecycle plumbing.
- Do not start implementation from this document without picking one narrower slice first.
