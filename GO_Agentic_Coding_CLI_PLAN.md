# Go CLI Agent — Architecture & Patterns to Adopt

## Verified Feature Mapping (from conversation → actual source files)

| Your feature | Source file(s) verified |
|---|---|
| Compact conversation | `services/compact/compact.ts`, `autoCompact.ts`, `microCompact.ts` |
| Multi-model support | `utils/model/providers.ts`, `utils/model/model.ts` |
| Skill system | `skills/loadSkillsDir.ts`, `skills/bundled/` |
| Tool executor | `tools/BashTool/`, `tools/GrepTool/`, `tools/FileReadTool/` etc. |
| Web research | `tools/WebSearchTool/`, `tools/WebFetchTool/` |
| Chain of thought | `utils/thinking.ts` (`ultrathink` + adaptive thinking budget) |
| Context injection | `context.ts` (`getSystemContext`, `getGitStatus`) |
| Commands | `commands.ts`, slash-command parsing in `utils/slashCommandParsing.ts` |

---

## Go Project Structure

```
cmd/agentcli/main.go       ← cobra entrypoint
internal/
  agent/
    loop.go                ← core dispatch loop
    context_inject.go      ← env context injection
  api/
    client.go              ← provider interface
    claude.go              ← Anthropic streaming
    openai.go              ← OpenAI/OpenAI-compat streaming
  tools/
    interface.go           ← Tool interface + PermissionLevel
    registry.go            ← Tool registry
    bash.go
    file_read.go
    file_write.go
    file_edit.go
    glob.go
    grep.go                ← rg wrapper
    web_search.go
    web_fetch.go
    git.go
  compact/
    pipeline.go            ← CompactionPipeline
    tool_truncate.go       ← strategy 1 (microcompact)
    sliding_window.go      ← strategy 2
    summarize.go           ← strategy 3 (calls LLM or local model)
    tokens.go              ← token estimation
  localmodel/
    runner.go              ← local model interface (ollama/llama.cpp)
    router.go              ← decides local vs. remote per task
  skills/
    loader.go              ← load from ~/.config/agentcli/agents/*.md & .agents/
    frontmatter.go         ← YAML frontmatter parser
  permissions/
    gating.go              ← ask/allow/deny per tool
    bash_rules.go          ← command-level rules
  tui/
    app.go                 ← bubbletea App
    input.go
    output.go
  utils/
    tokens.go              ← token counting
    messages.go            ← message normalization for API
  config/
    config.go
```

---

## Key Patterns to Adopt (Clean-Room Rewrite)

### 1. The Dispatch Loop — `query.ts` + `QueryEngine.ts`

The actual pattern from the source:

```go
// internal/agent/loop.go

// Per user turn (called once when user submits input):
injectEnvContext(&messages)               // ← fresh git status, cwd, etc. (pattern from context.ts)
messages = compactionPipeline.Run(messages) // ← compact BEFORE first LLM call

// Tool-call loop (runs until model stops calling tools):
for {
    response, err := api.StreamQuery(messages, tools)

    if len(response.ToolCalls) == 0 {
        break // final answer — exit loop
    }
    for _, call := range response.ToolCalls {
        result := toolRegistry.Execute(call, permCtx)
        messages = append(messages, toolResultMsg(result))
    }
    messages = compactionPipeline.Run(messages) // ← compact again if tool results pushed over limit
}
```

Key insights verified in source:
- Context injection happens **once per user turn**, not inside the tool-call loop
- Compaction fires **before** each LLM call (both first and subsequent)
- `autoCompact.ts` triggers at `effectiveContextWindow - AUTOCOMPACT_BUFFER_TOKENS` (13,000 token buffer)

---

### 2. Compaction Pipeline — `services/compact/`

Implement **three core strategies** (verified from source):

**Strategy A — Tool Result Truncation** (`microCompact.ts`)
Run first, zero API calls. Compactable tools: `FileRead`, shell tools, `Grep`, `Glob`, `WebSearch`, `WebFetch`, `FileEdit`, `FileWrite`. Truncates old tool results to `[Old tool result content cleared]`. This alone recovers 30-50% of space in code-heavy sessions.

**Strategy B — Summarization** (`compact.ts` + `prompt.ts`)
**Most valuable reference.** The compaction prompt in `services/compact/prompt.ts` is a production-tested string constant with 9 sections: Primary Request, Technical Concepts, Files/Code, Errors/Fixes, Problem Solving, All User Messages, Pending Tasks, Current Work, Optional Next Step. Adapt this format for your `summarize.go`.

**Strategy C — Partial Compaction** (`compact.ts`)
When full summarization would still overflow, use `getPartialCompactPrompt` to scope summarization to recent messages only, preserving earlier retained context intact. Prevents summary-of-summary recursion.

**Optional (advanced)** — The source also implements:
- **Selective Retention** (`compact.ts`): Score messages by importance, drop low-score ones
- **Hierarchical Compaction** (`compact.ts`): Maintain multiple memory layers (hot/warm/cold)

Start with A+B+C, add selective retention if needed.

**Threshold logic** (adopted from `autoCompact.ts`):
```go
const AutocompactBufferTokens = 13_000        // ~1.8% of 200k window
const WarningThresholdBufferTokens = 20_000   // warn user before hard threshold
const MaxConsecutiveFailures = 3              // circuit breaker

func autocompactThreshold(contextWindow int) int {
    return contextWindow - AutocompactBufferTokens
}
```

**Token counting** — Implement rough estimation (verified from `utils/tokenBudget.ts`):
```go
// Approximate: ~4 chars per token (cl100k encoding)
func estimateTokens(text string) int {
    return len(text) / 4
}
```
For production, integrate a library like `pkoukk/tiktoken-go` for exact counting.

---

### 3. Bash Validation — `tools/BashTool/bashSecurity.ts` + `destructiveCommandWarning.ts`

Two separate concerns, both worth adopting:

**Security validation** (inject into `bash.go`):
- Blocklist of ZSH dangerous commands: `zmodload`, `emulate`, `sysopen`, `syswrite`, `zpty`, `ztcp`, `zsocket`, `zf_rm`, `zf_mv`, `zf_chmod`
- Command substitution patterns to reject: `$()`, `${}`, `<()`, `>()`, `=cmd` (Zsh equals expansion)
- Block `IFS` injection, `HEREDOC_IN_SUBSTITUTION`, unicode whitespace in commands

**Destructive command warnings** (UI hint only, do not block):
- Git: `reset --hard`, `push --force`, `clean -f`, `checkout .`, `commit --amend`, `--no-verify`
- Files: `rm -rf`, `rm -f`
- DB: `DROP TABLE`, `TRUNCATE`, unbounded `DELETE FROM`
- Infra: `kubectl delete`, `terraform destroy`

Adapt the regex patterns from `destructiveCommandWarning.ts` — they are precise and production-tested.

---

### 4. Tool Permission System — `Tool.ts`

The source uses three rule lists, not a simple level enum:

```go
// internal/permissions/gating.go
type PermissionContext struct {
    Mode             string // "default", "bypassPermissions", "autoApprove"
    AlwaysAllowRules []Rule
    AlwaysDenyRules  []Rule
    AlwaysAskRules   []Rule
}
```

Auto-approve reads, prompt for writes/executes — but the actual classifier (`bashPermissions.ts`) checks command-level rules, not just tool-level. For your Go port: implement it as **per-command rules** for bash (e.g. `Bash(git diff:*)` allow, `Bash(rm:*)` ask).

---

### 5. Skill System — `skills/loadSkillsDir.ts`

Skills in the source are **markdown files with YAML frontmatter** loaded from:
- `~/.config/agentcli/agents/` (user-global, platform-generic)
- `.agents/` in project root (project-local)

Frontmatter fields that matter:
```yaml
---
name: git-workflow
description: Git operations and PR workflow guidance
allowed-tools: Bash, FileRead  # optional tool restriction
argument-hint: branch name      # optional
---
Your skill prompt content here...
```

The skill is injected as a system prompt section when invoked via `/skill-name` command or auto-detected on startup. Implement the same two-directory discovery pattern in `skills/loader.go`.

---

### 6. Context Injection — `context.ts`

Two layers, different cache strategies:

**Layer 1 — System context** (cached once per session via `getSystemContext`):
```go
// internal/agent/context_inject.go
type SystemContext struct {
    MainBranch string // default branch for PRs — stable per session
    GitUser    string // git config user.name — stable per session
}
```

**Layer 2 — Environment context** (refreshed every user turn):
```go
type TurnContext struct {
    CurrentDir    string // pwd — may change via cd in bash tool
    GitBranch     string // current branch — may change via checkout
    GitStatus     string // git status --short (truncated at 2000 chars)
    RecentLog     string // git log --oneline -n 5
}
```

The `getUserContext()` also loads AGENTS.md-style config files from ancestor directories up to home (~/.config/agentcli/AGENTS.md). These are loaded once at startup and on `/refresh` command.

---

### 7. Multi-Model Support — `utils/model/providers.ts`

The source supports four providers via env vars:
```
CLAUDE_CODE_USE_BEDROCK=1   → AWS Bedrock
CLAUDE_CODE_USE_VERTEX=1    → Google Vertex
ANTHROPIC_BASE_URL=...      → custom/compatible endpoint
ANTHROPIC_MODEL=...         → model override
```

For OpenAI-compat support in your Go build: implement a single `LLMClient` interface, then two concrete types: `AnthropicClient` and `OpenAIClient`. Model selection follows priority: `--model` flag → `ANTHROPIC_MODEL` env → config file → default.

---

### 8. Chain of Thought — `utils/thinking.ts`

Two mechanisms verified in source:

1. **Extended thinking** — API-level `budgetTokens` parameter, triggered by `ultrathink` keyword in user input
2. **System prompt CoT enforcement** — the main prompt explicitly instructs the model to "think before acting, verify assumptions with a tool call before writing code, prefer reading files over assuming content"

For multi-model support (non-Claude models don't have extended thinking API), implement option 2 only: a system prompt section that enforces CoT reasoning regardless of provider.

---

### 9. Local On-Device Model for Token Savings — `getSmallFastModel()` pattern

**Source precedent:** The source already uses a two-tier model strategy. `getSmallFastModel()` (Haiku) is called for cheap internal tasks that don't need the main model's full reasoning. Verified uses:

| Internal Task | Source file | Why it's cheap |
|---|---|---|
| Token counting/estimation | `services/tokenEstimation.ts` | Short input, numeric output |
| API key verification | `services/api/claude.ts` | 1-token response, throwaway |
| Away/session summary | `services/awaySummary.ts` | 30-message window, short output |
| Compaction summarization | `services/compact/compact.ts` | Long input but structured output |
| Bash command classification | `bashPermissions.ts` | Short input, boolean-ish output |

**Your advantage:** Replace all of these with a local on-device model like **Gemma 4 E4B** — zero API cost.

#### Where to apply Gemma 4 E4B in your Go project

**Tier 1 — High-value local tasks (implement first):**

| Task | File | Savings | Feasibility |
|---|---|---|---|
| Compaction summarization | `compact/summarize.go` | **Highest** — every compaction currently costs 1 API call | Good — structured prompt, Gemma handles summarization well |
| Selective retention scoring | `compact/pipeline.go` | High — scores each message for importance | Excellent — classification task, perfect for small models |
| Session title generation | `agent/loop.go` | Medium — 1 call per session | Excellent — trivial task |

**Tier 2 — Medium-value local tasks:**

| Task | File | Savings | Feasibility |
|---|---|---|---|
| User intent detection | `agent/loop.go` | Medium — enhances context | Good — frustration/urgency/continuation detection |
| Bash command risk scoring | `permissions/bash_rules.go` | Medium — 1 call per bash execution | Good — classification into safe/risky/dangerous |
| Context relevance filtering | `agent/context_inject.go` | Medium — reduces what gets injected | Moderate — needs to understand code semantics |

**Tier 3 — Lower-value but still useful:**

| Task | File | Savings | Feasibility |
|---|---|---|---|
| Tool result summarization before truncation | `compact/tool_truncate.go` | Low-medium — summarize before clearing | Good — replaces blind truncation with smart truncation |
| Commit message generation | `tools/git.go` | Low — occasional | Excellent — Gemma handles this well |

#### Architecture: Model Router

```go
// internal/localmodel/router.go
type TaskType int

const (
    TaskCompaction    TaskType = iota  // → prefer local
    TaskScoring                        // → prefer local
    TaskTitleGen                       // → prefer local
    TaskIntentDetect                   // → prefer local
    TaskMainReasoning                  // → always remote (main model)
)

type ModelRouter struct {
    localModel  LocalModel     // Gemma 4 E4B via ollama
    remoteModel LLMClient      // Claude/OpenAI API
    localAvail  bool           // is local model running?
}

func (r *ModelRouter) Route(task TaskType, messages []Message) (LLMClient, error) {
    // Fall back to remote if local model unavailable
    if !r.localAvail {
        return r.remoteModel, nil
    }
    switch task {
    case TaskCompaction, TaskScoring, TaskTitleGen, TaskIntentDetect:
        return r.localModel, nil
    default:
        return r.remoteModel, nil
    }
}
```

#### Local Model Integration via Ollama

```go
// internal/localmodel/runner.go
type LocalModel struct {
    baseURL   string // default: http://localhost:11434
    modelName string // default: gemma4-e4b
}

func (m *LocalModel) Query(prompt string, maxTokens int) (string, error) {
    // POST to Ollama /api/generate endpoint
    // Same interface as remote LLMClient but no streaming needed
}

func DetectLocalModel() (*LocalModel, bool) {
    // Check if ollama is running: GET http://localhost:11434/api/tags
    // Look for gemma4-e4b or similar small model
    // Return (nil, false) if unavailable — graceful fallback
}
```

#### Token savings estimate

For a typical 2-hour coding session with 5 compaction cycles:
- **Without local model:** ~5 summarization API calls × ~4k output tokens = ~20k tokens ($0.30)
- **With Gemma 4 E4B:** 0 API tokens for summarization, scoring, titles = **~$0 for internal tasks**
- **Over a month (20 work days):** saves ~400k tokens (~$6) per developer

The real value is not just cost — it's **latency**. Local inference on Gemma 4 E4B (~4B params) runs in 1-3 seconds on Apple Silicon vs. 3-8 seconds for an API roundtrip. Compaction feels instant.

#### Fallback behavior
Always graceful: if `ollama` is not running or the model isn't pulled, fall back to the remote API silently. The user should never be blocked by the local model being unavailable.

---

### 10. Ripgrep Integration — `utils/ripgrep.ts`

The source uses a priority chain: system `rg` → bundled vendor binary → embedded. For Go, simplify:
1. Check if `rg` is on PATH via `exec.LookPath`
2. Fall back to `grep -r` if not found

This is your `grep.go` tool — do not bundle ripgrep, just shell out to it.

---

## What NOT to Build (verified dead weight for your scope)

| Feature | Source location | Why skip |
|---|---|---|
| KAIROS background agent | `feature('KAIROS')` gated everywhere | You said no |
| Coordinator/Subagent mode | `coordinator/` dir, `feature('COORDINATOR_MODE')` | You said no |
| Undercover mode | `utils/undercover.ts` | Anthropic-internal only |
| BUDDY tamagotchi | `buddy/` dir | Not minimal |
| MCP server orchestration | `services/mcp/` | You said no |
| LSP integration | `tools/LSPTool/` | Not in your feature list |
| React/Ink TUI | `ink.ts`, all `*.tsx` files | Use bubbletea |
| 3,167-line `print.ts` | `cli/print.ts` | SDK mode only |

---

## Effort Estimate (solo Go developer)

| Component | Source reference | Effort |
|---|---|---|
| LLM API client + streaming | `services/api/claude.ts` | 3–5 days |
| Dispatch loop + context injection | `QueryEngine.ts`, `context.ts` | 3–5 days |
| Tool executor (7 tools) | `tools/BashTool/`, `FileReadTool/`, etc. | 6–8 days |
| Bash validation blocklists | `bashSecurity.ts`, `destructiveCommandWarning.ts` | 2–3 days |
| Permission gating | `Tool.ts`, `bashPermissions.ts` | 2–3 days |
| Compaction pipeline (3 strategies) | `compact/` dir | 1–2 weeks |
| Compaction prompt (adapt from reference) | `services/compact/prompt.ts` | 1 day |
| Skill system (markdown loader) | `skills/loadSkillsDir.ts` | 3–4 days |
| TUI (bubbletea) | Nothing to steal — rewrite | 1–2 weeks |
| Token counting + message normalization | `utils/tokens.ts`, `utils/messages.ts` | 2 days |
| Error handling + retry logic | `services/api/errors.ts` | 1–2 days |
| Local model router (Gemma 4 E4B) | `getSmallFastModel()` pattern | 3–4 days *(phase 2)* |
| Multi-model provider interface | `utils/model/providers.ts` | 2–3 days *(phase 2)* |
| Commands + config | `commands.ts` | 3–4 days |
| **MVP (Generic provider-agnostic)** | | **~8–10 weeks solo** |
| **Phase 2a (local model)** | Ollama + Gemma 4 E4B for internal tasks | **+3–4 days** |
| **Phase 2b (multi-model support)** | Add OpenAI compat | **+1 week** |

---

## Recommended Build Order

### Week 1–2 (MVP Core)
1. **`internal/api/`** — Claude streaming client with tool use (day 1–5)
2. **`internal/agent/loop.go`** — dispatch loop with mocked tools (day 5–7)  
   **Checkpoint:** Can chat and mock-execute tools ✓
3. **`internal/tools/`** — real tool execution for 5 essential tools (day 7–11):
   - `bash.go`, `file_read.go`, `file_write.go`, `file_edit.go`, `glob.go`
   - Add `git.go` and `web_search.go` by end of week 2
4. **`internal/utils/tokens.go`** — token counting + message normalization (day 11–12)

### Week 3 (Security & Awareness)
5. **`internal/permissions/`** — permission gating + bash validation (day 12–15)
6. **`internal/agent/context_inject.go`** — git status + working directory injection (day 15–16)

### Week 4–5 (Compaction — your differentiator)
7. **`internal/compact/`** — full 3-strategy pipeline (day 16–28)
   - Strategy A (tool truncation): day 1
   - Strategy B (summarization): day 2–3 (mostly adapted prompt)
   - Strategy C (partial): day 4
   - Threshold logic + tests: day 5

### Week 6 (Interface & Configuration)
8. **`internal/tui/`** — bubbletea wrapping the agent loop (day 28–33)
9. **`internal/skills/`** — markdown skill loader (day 33–36)  
   - Loads `~/.config/agentcli/agents/*.md` and `.agents/*.md` with YAML frontmatter
10. **`internal/config/`** — CLI flags + AGENTS.md config file parsing (day 36–39)

### Phase 2a (Post-MVP, ~3–4 days) — Local Model for Token Savings
11. **`internal/localmodel/runner.go`** — Ollama/Gemma 4 E4B integration
    - Auto-detect if ollama is running at startup
    - Implement `LocalModel` satisfying `LLMClient` interface
12. **`internal/localmodel/router.go`** — Task-based model router
    - Route compaction, scoring, title gen → local model
    - Route main reasoning, tool execution → remote API
    - Graceful fallback if local unavailable
13. **Wire local model into `compact/summarize.go`** — biggest savings  
    **Checkpoint:** Compaction runs offline with zero API cost ✓

### Phase 2b (Post-MVP, ~1 week) — Multi-Model Support
14. **`internal/api/openai_client.go`** — OpenAI/compat provider abstraction
    - Refactor `LLMClient` interface to swap providers

**After week 1:** You have a working single-turn chat loop that can read/write files.  
**After week 5:** Full agent with compaction — production-grade feature set.

---

## Critical Architecture Notes

### Message Normalization (`utils/messages.ts`)
The source normalizes messages before every API call:
- Consolidate consecutive assistant/user messages
- Ensure tool results are paired with tool calls
- Strip trailing whitespace

This is essential for correct API semantics — don't skip it.

### Error Handling (`services/api/errors.ts`)
The source categorizes API errors:
- `prompt_too_long` → trigger compaction immediately
- `rate_limit` → exponential backoff (1s, 2s, 4s)
- `overloaded` → retry with delay

Start with simple retry logic: 3 attempts with exponential backoff.

### Context Injection — The Secret Sauce
Two-layer injection: session-stable data (main branch, git user) cached once + turn-level data (branch, status, cwd) refreshed on each user message. See section 6 for the full struct breakdown.

This is why the source feels "aware" of your project state. Most agents inject once at startup and never update. The key insight: refresh the volatile parts (status, branch, cwd) every turn.

### Key Reference Assets

**1. Compaction prompt** (`services/compact/prompt.ts`)  
Adapt the 9-section summary format — it's production-tested.

**2. ZSH dangerous commands** (`tools/BashTool/bashSecurity.ts`)  
```
zmodload, emulate, sysopen, sysread, syswrite, zpty, ztcp, zsocket,
zf_rm, zf_mv, zf_chmod, zf_mkdir, zf_chown, mapfile
```

**3. Destructive command patterns** (`tools/BashTool/destructiveCommandWarning.ts`)  
Adapt the regex patterns for: git reset, rm -rf, DROP TABLE, kubectl delete, terraform destroy, etc.

**4. Token thresholds** (`services/compact/autoCompact.ts`)  
```
AutocompactBufferTokens = 13_000
WarningThresholdBufferTokens = 20_000
ManualCompactBufferTokens = 3_000
MaxConsecutiveAutocompactFailures = 3
```

---

## Core Insight

> **The real magic is:** Dispatch loop + per-turn context injection + compaction before LLM call. This pattern is what separates a toy agent from a production tool. Most open-source CLI agents miss the per-turn context refresh — they inject once at startup. The source refreshes every turn.

> **Compaction is structural, not optional.** It's not a long-context luxury — it's the difference between working for 5 minutes vs. working for hours. Allocate a full week to it (week 4–5) and get it right.

> **Local model (Gemma 4 E4B) is your unfair advantage.** The source pays for Haiku API calls for every internal task. You can do them for free on-device. Compaction summaries, scoring, session titles — all run locally in 1-3 seconds on Apple Silicon. This is something the original source code doesn't do, and it makes your Go port cheaper and faster than the original for long sessions.
