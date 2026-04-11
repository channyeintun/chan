# Enhancement Research: VS Code Copilot Chat Patterns

Date: 2026-04-12

This document replaces the old mixed backlog. It is a research-only note and implementation plan seed.

## Scope

- Source repo: `microsoft/vscode-copilot-chat`
- Primary research target: `src/extension/tools`
- Primary research target: `src/extension/agents`
- For subagent orchestration, the actual control flow also extends into:
  - `src/extension/tools/node/searchSubagentTool.ts`
  - `src/extension/tools/node/executionSubagentTool.ts`
  - `src/extension/prompt/node/searchSubagentToolCallingLoop.ts`
  - `src/extension/prompt/node/executionSubagentToolCallingLoop.ts`
  - `src/extension/intents/node/toolCallingLoop.ts`

## Boundaries

- This is not a parity project.
- This is not a team/swarm/remote-agent roadmap.
- This is not an implementation task list yet.
- Only two areas are in scope:
  - file-related tools
  - subagent orchestration

## Current `gocode` baseline

`gocode` already has a stronger local baseline than the old enhancement note assumed.

- File tools already present: `file_read`, `file_write`, `file_edit`, `multi_replace_file_content`, `file_diff_preview`, `file_history`, `file_history_rewind`
- File-adjacent navigation already present: `glob`, `grep`, `go_definition`, `go_references`, `symbol_search`, `project_overview`
- Execution and batching already present: permission levels, schema validation, path resolution, ordered concurrent execution, streaming result delivery
- Child-agent surface already present: `agent`, `agent_status`, `agent_stop`

Conclusion: the best ideas to borrow are reliability and orchestration patterns, not raw tool-count parity. For file tools, the decision standard should be explicit: match VS Code Copilot Chat where it is stronger, keep `gocode` behavior where it is already safer, and avoid copying features that weaken determinism.

## File-tool robustness rule

For file tools, `gocode` should only diverge from VS Code Copilot Chat in two acceptable directions:

- stronger behavior
- intentionally narrower behavior with a clear safety advantage

Anything else is accidental weakness.

This matters because some VS Code Copilot Chat file tooling is genuinely more mature, but some of it optimizes for flexibility inside VS Code rather than strict local determinism. `gocode` should not copy features blindly.

## Where `gocode` is already stronger

These are not gaps to close. They are strengths to preserve.

### 1. Strict working-directory containment

`gocode/internal/tools/path_resolution.go` currently rejects relative paths that escape the working directory.

Why this is stronger:

- It removes an entire class of accidental cross-project edits.
- It is stricter than VS Code Copilot Chat's external-file confirmation model.

Decision:

- Keep strict cwd containment as the default local CLI posture.
- If external-path support is ever added, it should be opt-in and confirmation-heavy rather than replacing the current rule.

### 2. Session-local rollback with snapshots

`gocode/internal/tools/file_history.go` and `gocode/internal/tools/file_history_tools.go` provide snapshot, diff-stats, and rewind behavior.

Why this is stronger:

- VS Code Copilot Chat has strong edit confirmation, but not the same built-in session-local rewind tool surface.
- `gocode` can recover from bad edits without relying on editor undo state.

Decision:

- Keep file history as a differentiator.
- Any future file-edit improvements should integrate with it instead of bypassing it.

### 3. Chunk validation in `multi_replace_file_content`

`gocode/internal/tools/multi_replace_file_content.go` validates line ranges, validates the target content for each chunk, and rejects overlapping edits.

Why this is stronger:

- It is highly deterministic.
- It reduces accidental drift during multi-region edits.

Decision:

- Preserve this exactness.
- Do not replace it with a looser edit mechanism.

## Where VS Code Copilot Chat is stronger

These are the main file-tool areas where `gocode` should either match or intentionally justify staying narrower.

### 1. Edit strategy coverage

VS Code Copilot Chat has a clearer edit ladder:

- `replace_string_in_file` for exact single replacements
- `multi_replace_string_in_file` for batched exact replacements
- `apply_patch` for larger structural edits
- `edit_file` for more flexible edit application

`gocode` currently has strong exact-edit tools, but no patch-grade tool.

### 2. Edit error taxonomy and guided recovery

VS Code Copilot Chat distinguishes failure modes such as:

- no match
- multiple matches
- no-op edit
- content format problems

It also returns suggestions that steer the model toward rereading or narrowing the edit.

`gocode` currently returns mostly flat error strings.

### 3. Binary and external-file read handling

VS Code Copilot Chat is stronger at:

- detecting image and binary files
- giving continuation hints for partial reads
- separating workspace files from external-file reads with confirmation

`gocode` is currently safer on path containment, but weaker on binary awareness and partial-read guidance.

### 4. Risk-tiered write confirmation

VS Code Copilot Chat has stronger guardrails for:

- dotfiles
- editor config
- home-directory locations
- workspace-external files

`gocode` currently has a coarse permission model and path containment, but not a finer-grained risk tier inside the allowed tree.

### 5. Post-edit diagnostics

VS Code Copilot Chat integrates diagnostics into edit flows so the model can immediately see whether an edit introduced language errors.

`gocode` currently returns diff-style output but not language-aware post-edit diagnostics.

## What not to copy from VS Code Copilot Chat

Not every advanced behavior from VS Code Copilot Chat is automatically better for `gocode`.

### 1. Do not make fuzzy or similarity-based replacement the default

VS Code Copilot Chat includes fuzzy and similarity matching in its edit engine.

Why this should not be copied blindly:

- It trades determinism for convenience.
- In a local CLI workflow, a confident wrong edit is worse than a rejected edit.

Decision:

- Keep exact replacement as the default.
- If any healing exists, it should be explicit, low-confidence-aware, and never the silent default.

### 2. Do not weaken path safety just to match external-file flexibility

VS Code Copilot Chat supports external-file reads and edits through confirmation flows.

Why this should not be copied blindly:

- The editor has richer context for user confirmation than a pure CLI loop.
- `gocode` benefits from a stricter local sandbox by default.

Decision:

- Keep cwd containment unless there is a deliberate product decision to expand it.

## File-tool match-or-exceed bar

Any future file-tool work should be judged against this checklist.

1. Exact edits must remain deterministic and reject ambiguous matches.
2. Larger structural edits must have a dedicated patch-grade path.
3. Create, overwrite, and edit intent must be clearly separated.
4. Every write path must produce a stable diff preview.
5. Every edit failure must be classified and return a recovery suggestion.
6. Reads must detect binary or image-like inputs and fail safely or switch modes.
7. Large or partial reads must tell the model how to continue reading.
8. Risky paths inside the allowed tree must require stronger confirmation than ordinary source files.
9. File-history snapshot and rewind support must continue to work across all write tools.
10. Post-edit diagnostics should be surfaced when available so bad edits are caught immediately.

## Best file-tool takeaways to adopt

### 1. Add a dedicated patch-grade edit tool

Reference files:

- `src/extension/tools/node/applyPatchTool.tsx`
- `src/extension/tools/node/applyPatch/parser.ts`
- `src/extension/prompts/node/panel/editCodePrompt2.tsx`

What is worth copying:

- A distinct tool for multi-hunk and multi-file edits
- A clear contract for when to use patch edits versus exact string replacement
- A patch format that is structured enough to validate before write time

Why it fits `gocode`:

- `file_edit` is good for exact replacements but brittle for larger dispersed edits
- `multi_replace_file_content` is still exact-match driven
- A patch tool would complement, not replace, the current edit family

Recommended direction:

- Add `apply_patch` as the large-edit path
- Keep `file_edit` for small exact replacements
- Keep `multi_replace_file_content` for repeated exact replacements

### 2. Add edit failure taxonomy and guided recovery

Reference files:

- `src/extension/tools/node/editFileToolUtils.tsx`
- `src/extension/tools/node/abstractReplaceStringTool.tsx`

What is worth copying:

- Distinct error classes for no match, multiple matches, and no-op edits
- Tool responses that tell the model what to do next instead of only failing
- A repair path that encourages reread or narrower edits rather than blind retries

Why it fits `gocode`:

- Current edit errors are accurate but mostly flat strings
- Richer failure classes would reduce retry churn and make automated recovery more reliable

Recommended direction:

- Standardize edit failure kinds across `file_edit`, `multi_replace_file_content`, and future `apply_patch`
- Return actionable recovery hints in tool output

### 3. Make file reads more chunk-aware and binary-aware

Reference files:

- `src/extension/tools/node/readFileTool.tsx`

What is worth copying:

- Explicit continuation hints when a read is partial or truncated
- A first-class chunking model for large files
- Binary handling instead of treating every file like text

Why it fits `gocode`:

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
