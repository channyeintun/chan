# TUI Parity Plan

## Goal

Bring `go-cli/tui` as close as possible to the interaction model and visual behavior used in `sourcecode`, with priority on:

- main prompt input
- permission prompt input
- markdown rendering
- syntax highlighting
- transcript/message layout
- status/footer behavior

## Comparison Summary

### 1. App shell and layout

Current `go-cli/tui`:

- uses a compact single-column layout in `go-cli/tui/src/App.tsx`
- renders a simple `StatusBar`, `StreamOutput`, optional `PlanPanel`/`ArtifactView`, then either `Input` or `PermissionPrompt`
- has no footer system, no prompt chrome, no list virtualization, no transcript mode split, and no upstream-style message row composition

Reference `sourcecode`:

- uses a richer layout pipeline built from `Messages`, `MessageRow`, `PromptInput`, status notices, footer pills, and multiple modal/dialog overlays
- separates message rendering concerns from prompt/footer concerns
- treats the prompt as a full subsystem, not a single line input widget

Result:

- exact parity is not a styling-only task; the current app structure must be reshaped around upstream interaction boundaries

### 2. Main input

Current `go-cli/tui/src/components/Input.tsx`:

- manual `useInput` handler (~50 lines)
- single-line editing only
- no cursor movement inside the string
- delete only removes from the end
- no selection, no multiline growth, no ghost text, no footer hints, no rich placeholder logic
- prompt history is minimal and separate from editing behavior

Reference `sourcecode/components/PromptInput/PromptInput.tsx`, `sourcecode/components/TextInput.tsx`, and `sourcecode/hooks/useTextInput.ts`:

- `useTextInput` is a 500+ line state machine with a `Cursor` class for positioning (column/line/offset)
- full cursor-aware editing: left/right arrows, home/end, word-jump
- kill ring (Ctrl-W, Ctrl-K, Ctrl-U) and yank ring with yank-pop (Ctrl-Y)
- multiline input with viewport calculation and dynamic growth
- image paste with dimensions detection from clipboard
- prompt history/search and keybinding interactions are integrated with the input model
- dynamic placeholder behavior, ghost text/inline suggestions
- prompt footer composition (hints, suggestions, token warnings) as a sub-component

Result:

- the current input must be replaced, not incrementally restyled
- the replacement requires porting both the text-input state machine and the prompt container/footer layer

### 3. Permission prompt

Current `go-cli/tui/src/components/PermissionPrompt.tsx`:

- static bordered box (~50 lines)
- direct `y/n/a/s` key capture only
- no selection UI, no focus, no tab-to-amend feedback, no cancel path beyond ignoring input

Reference `sourcecode/components/permissions/PermissionPrompt.tsx` and `sourcecode/components/permissions/`:

- selection-based prompt with focus state using `CustomSelect` component
- tab toggle between option selection and feedback input mode
- extensible `PermissionPromptOption` type with ReactNode labels
- dedicated cancel handling and reusable permission option model
- 14+ specialized permission request components (FileWritePermissionRequest, BashPermissionRequest, WebFetchPermissionRequest, NotebookEditPermissionRequest, SkillPermissionRequest, WorkerBadge, etc.)
- each request type renders contextual detail (file paths, command previews, risk explanations)

Result:

- current permission UX is materially different in both behavior and information architecture
- full parity requires both the generic selection prompt and per-tool-type request renderers
- the current engine payload (`tool`, `command`, `risk`) is too sparse for upstream-quality contextual labels

### 4. Markdown rendering and syntax highlighting

Current `go-cli/tui/src/components/MarkdownText.tsx`:

- uses `marked-terminal`
- renders markdown to a plain string and prints it with `Text`
- no streaming markdown boundary optimization
- no syntax highlighter integration beyond whatever `marked-terminal` emits
- no structured table rendering

Reference `sourcecode/components/Markdown.tsx`:

- tokenizes markdown with a cached lexer backed by a module-level LRU cache (500 items)
- `hasMarkdownSyntax()` fast-path skips the lexer entirely for plain text (majority of responses)
- renders non-table content through ANSI formatting helpers
- uses `getCliHighlightPromise()` for async CLI syntax highlighting
- has separate streaming markdown handling to avoid reparsing the stable prefix on every token delta
- renders tables as dedicated `MarkdownTable.tsx` React components
- `stripPromptXMLTags()` for prompt artifact sanitization

Result:

- markdown parity requires a new renderer path, not a tuning of the current one

### 5. Transcript and message rendering

Current `go-cli/tui/src/components/StreamOutput.tsx` and `ToolProgress.tsx`:

- simple role label plus content
- limited grouping for read/search tool calls
- streaming state is rendered inline with a generic spinner label
- tool rendering is summary-oriented and flat

Reference `sourcecode/components/Messages.tsx`, `Message.tsx`, `MessageRow.tsx`, and `sourcecode/components/messages/`:

- message rendering is block-aware and message-type specific
- dedicated renderer files for each block type: `AssistantTextMessage.tsx`, `AssistantToolUseMessage.tsx`, `AssistantThinkingMessage.tsx`, `UserTextMessage.tsx`, `CompactBoundaryMessage.tsx`, `GroupedToolUseContent.tsx`, `CollapsedReadSearchContent.tsx`, etc.
- `MessageRow.tsx` wrapper handles consistent spacing, dot indicators, and layout rules
- `VirtualMessageList.tsx` provides virtualized rendering for long transcripts
- progress messages are a separate streaming-aware concept distinct from completed messages
- collapsed group spinner state tracking is more nuanced than simple show/hide

Result:

- current transcript logic is too coarse for upstream parity; it needs message-block renderers and row-level layout rules
- virtualized rendering is needed for performance in long sessions

### 6. Status and footer

Current `go-cli/tui/src/components/StatusBar.tsx`:

- ~30 lines, single bordered box at top
- shows `[READY/BOOTING] | mode | model | $cost`
- no context window info, no rate limit info, no session name

Reference `sourcecode/components/StatusLine.tsx` and `sourcecode/components/PromptInput/PromptInputFooter*.tsx`:

- **status line** (~100 lines): context window percentage, rate limit info (5-hour and 7-day windows), session name, output style, model display with betas, workspace/directory context
- **prompt footer** (separate layer): command suggestions and hints, permission mode indicator, token warnings, cost thresholds, MCP server status
- these are two distinct rendering layers, not one bar

Result:

- current status bar must be split into an upstream-style status line and a prompt-adjacent footer
- some status fields (rate limits, context window %) require data the engine may not currently emit

### 7. Protocol and state gaps

Current TUI event model in `go-cli/tui/src/protocol/types.ts` and `go-cli/tui/src/hooks/useEvents.ts`:

- stores assistant text as a single accumulated string per turn
- permission requests contain only `tool`, `command`, and `risk`
- no richer prompt/footer state
- no concept of message block types beyond `message` and `tool_call`
- no mechanism to communicate which blocks are still streaming vs complete

Reference behavior requires:

- block-oriented message types: `thinking`, `redacted_thinking`, `tool_use` (with metadata), `tool_result` (with context), `text`, `image`, `CompactBoundary`
- richer permission metadata: explanation text, risk assessment detail, worker context, MCP server identification, sandbox violation context
- streaming markdown rendering that can distinguish stable and unstable sections
- progressive/streaming tool use state (in-progress tracking per block)

Result:

- some parity work can happen entirely in the TUI, but exact permission-dialog and transcript parity may require protocol expansion from the Go engine

## Implementation Plan

### Phase 1: Layout and prompt foundation

1. Replace the current bottom input with a prompt container modeled after the upstream prompt area.
2. Introduce a dedicated text input component with cursor-aware editing and multiline rendering.
3. Add clipboard image paste support with dimension detection.
4. Preserve existing engine command wiring while moving prompt behavior behind the new input abstraction.

### Phase 2: Permission UX parity

1. Replace the current keypress-only permission box with a selectable permission prompt.
2. Match upstream option labels and keyboard flow as closely as current payloads allow.
3. Add a TUI-side amendment/feedback path only if the engine can consume it cleanly; otherwise keep the UI shape ready and document the protocol gap.

### Phase 3: Markdown and syntax highlighting parity

1. Replace `marked-terminal` rendering with a token-based markdown renderer.
2. Add ANSI-aware code block rendering with syntax highlighting.
3. Add a streaming markdown renderer that avoids reparsing stable content on every token delta.
4. Render tables and fenced blocks with dedicated components where needed.

### Phase 4: Transcript/message-row parity

1. Refactor `StreamOutput` into message-row style renderers.
2. Split assistant text, streaming text, thinking text, tool rows, and grouped tool rows into distinct renderers.
3. Introduce a `MessageRow` wrapper for consistent spacing, dot indicators, and layout rules.
4. Add virtualized message list rendering for long transcripts.
5. Match upstream spacing, role labeling, and row rhythm more closely.

### Phase 5a: Status line parity

1. Replace the current `StatusBar` with an upstream-style status line.
2. Add context window percentage, session name, and model display formatting.
3. Keep cost/model/mode visible but render in a lighter inline style.

### Phase 5b: Prompt footer parity

1. Add a prompt-adjacent footer layer beneath the input area.
2. Move command hints and mode indicators into the footer.
3. Show token warnings and cost thresholds when relevant.

### Phase 6: Protocol follow-up

1. Evaluate whether the Go engine must emit richer permission metadata (explanation text, risk detail, worker context).
2. Evaluate whether assistant message state should become block-oriented instead of single-string per turn.
3. Evaluate whether context window / rate limit data should be emitted for the status line.
4. Only expand the protocol where TUI parity cannot be reached from existing events.

## Recommended Execution Order

1. Prompt/input foundation
2. Permission prompt
3. Markdown renderer and syntax highlighting
4. Transcript row refactor (including virtualization)
5. Status line
6. Prompt footer
7. Optional protocol expansion if still required

## Out of Scope

The following sourcecode features are explicitly excluded from this parity effort:

- **Voice input** — waveform animation, voice recording integration
- **Vim mode** — `useTextInput` supports vim keybindings but this requires a keybinding subsystem we don't have
- **Feature flag system** — compile-time `feature()` gates
- **Plugin system** — third-party plugin initialization and lifecycle
- **Multi-screen architecture** — separate screens (Doctor, ResumeConversation, AssistantSessionChooser, etc.)
- **Modal/dialog overlays** — dialog launcher system for multiple overlay types
- **Coordinator mode** — separate app flowpath for coordinator sessions
- **Analytics/telemetry** — event tracking in permission prompts and other interactions
- **MDM/keychain** — enterprise prefetch and credential management
- **Suggestion dropdown** — inline suggestion/autocomplete dropdown in prompt input

These can be revisited individually if needed later.

## Risks

- importing upstream components directly is unlikely to work cleanly because `sourcecode` depends on a larger app state, keybinding, and custom Ink stack
- exact visual parity may require pulling over helper utilities instead of only copying JSX
- permission prompt parity is partially limited by current engine payload shape
- transcript parity may expose additional gaps in how tool/thinking state is modeled today
- the text input state machine alone is 500+ lines; porting it is a significant unit of work

## Definition of Done

- prompt area behaves like the upstream prompt instead of the current raw line editor
- permission flow is selection-based and visually aligned with the upstream pattern
- assistant markdown supports upstream-style formatting and syntax-highlighted code blocks
- transcript spacing and message/tool row behavior are substantially aligned with `sourcecode`
- remaining non-parity items are explicitly documented as protocol or architecture gaps