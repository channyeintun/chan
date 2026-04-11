# Enhancement Opportunities

This file has been reset from the old parity-audit backlog. Those historical items are now resolved or superseded, so the current list focuses on new product opportunities discovered by reviewing these reference prompts:

- `reference/system-prompts/Antigravity/fast.txt`
- `reference/system-prompts/Antigravity/planning.txt`
- `reference/system-prompts/ClaudeCode/prompt.txt`
- `reference/system-prompts/ClaudeCode/tools.txt`

## Highest-Leverage Opportunities

### 1. Task-mode UI with explicit review checkpoints

**Prompt signal**
Antigravity planning mode is built around `task_boundary` / `notify_user`, explicit `PLANNING`, `EXECUTION`, and `VERIFICATION` phases, and approval-style review of plan artifacts before coding.

**Current gap**
`gocode/internal/agent/modes.go` only exposes `plan` and `fast`. `gocode/internal/agent/planner.go` persists implementation-plan artifacts, but the runtime still relies on “save the plan, then tell the user to switch to /fast” instead of a first-class task-mode UI and review gate.

**Scope**
Add task lifecycle events over IPC, render a task-status panel in the TUI, and let the agent pause for explicit review/approval without forcing the user to manage the transition only through slash commands.

### 2. Interactive artifact review instead of plain-text previews

**Prompt signal**
Antigravity treats artifacts as first-class outputs with structured review and rich markdown formatting guidance: alerts, diffs, mermaid, media, carousels, and clickable file links.

**Current gap**
`gocode/tui/src/components/PlanPanel.tsx` and `gocode/tui/src/components/ArtifactView.tsx` currently render artifact bodies as plain text. Artifact IPC only supports `artifact_created` / `artifact_updated`, so there is no artifact-scoped review action or richer rendering contract.

**Scope**
Upgrade artifact rendering to reuse the markdown pipeline for full artifact bodies, then add artifact review actions, revision requests, and version-aware display for plans, walkthroughs, and search reports.

### 3. Workflow library and workflow-aware slash commands

**Prompt signal**
Both Antigravity prompts define `.agent/workflows/*.md` files with YAML frontmatter plus `// turbo` and `// turbo-all` execution hints. They also expect the agent to discover workflow files when a slash command or task matches.

**Current gap**
`gocode/internal/skills/loader.go` only loads `.agents/*.md` skills. Slash commands in `gocode/cmd/gocode/main.go` are fixed to `/plan`, `/fast`, `/model`, `/cost`, `/usage`, `/compact`, `/resume`, and `/help`.

**Scope**
Add workflow discovery, parsing, and safe execution semantics, plus a workflow command surface such as `/workflow <name>` or direct workflow-backed slash commands with approval-aware auto-run behavior.

### 4. Knowledge Item system backed by session history

**Prompt signal**
Antigravity heavily leans on Knowledge Items (KIs): review KI summaries before research, reuse past analysis, and update durable learnings instead of rediscovering them every session.

**Current gap**
`gocode/internal/artifacts/types.go` declares `KindKnowledgeItem`, but there is no KI loader, ranking/indexing, or prompt injection path. `gocode/internal/session/store.go` persists transcripts, but there is no layer that promotes durable knowledge out of those transcripts.

**Scope**
Add KI metadata/index storage, retrieve relevant KIs for new turns, and let the agent promote durable findings from sessions into reusable knowledge artifacts.

### 5. Delegated subagents for bounded research and setup tasks

**Prompt signal**
ClaudeCode’s tool model includes a Task/subagent mechanism with specialized agents and bounded handoff of work.

**Current gap**
`gocode/internal/tools/registry.go` exposes a single-agent tool surface only. There is no way to launch a scoped child agent for read-only exploration, product setup, or focused investigations.

**Scope**
Add a subagent runtime that can launch a child session with restricted tools/context, collect its report, and fold the result back into the parent turn without losing transcript clarity.

### 6. Hook feedback on prompt submission

**Prompt signal**
ClaudeCode treats hook output, including `user-prompt-submit-hook`, as user feedback that should influence the next iteration.

**Current gap**
`gocode/internal/hooks/types.go` defines session/tool/permission/compact/stop hooks, but there is no prompt-submit hook type or first-class path that injects hook feedback into the next user turn before model execution.

**Scope**
Add prompt-submit hooks, merge their messages into turn context as user feedback, and surface blocked or adjusted prompt state clearly in the UI.

### 7. Prompt caching and context-pressure adaptation

**Prompt signal**
These reference prompts assume long-running agentic loops where stable instructions, memory, and repeated context should be handled efficiently.

**Current gap**
`gocode/internal/cost/pricing.go` and `gocode/internal/cost/tracker.go` already account for cache read/write tokens, but `gocode/internal/api/anthropic.go` does not send prompt-cache controls. `gocode/internal/agent/modes.go` also uses fixed read parallelism and summary verbosity instead of adapting to live context pressure.

**Scope**
Enable Anthropic prompt caching for stable prompt sections, surface cache savings in `/cost`, and adapt read parallelism / tool-summary verbosity as the live context window fills.

## Already Implemented

Do not reopen these as new backlog items unless new regressions appear.

- Distinct `plan` and `fast` execution modes.
- Session task-list, implementation-plan, walkthrough, and tool-log artifacts.
- Permission prompts with optional user notes on tool approvals.
- Skill auto-loading from `.agents/` and `~/.config/gocode/agents/`.
- Hook runners for session/tool/permission/compact/stop lifecycle events.
- PTY-backed background commands with `command_status` and `send_command_input`.
- Conversation compaction, cost tracking, and context window warnings.
- Structured directory/file/search tools and path-traversal hardening.
