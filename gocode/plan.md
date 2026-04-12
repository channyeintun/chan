# Implementation Plan

## Objective

Add slash-command preview to the TUI and restyle the prompt chrome to match the screenshot:

- prompt border should show only the top and bottom edges
- prompt area should feel more padded and block-like
- slash commands should preview while the user is still typing the leading `/command` token
- preview rows should show command names plus descriptions
- do not add tests

## Current State

- `tui/src/components/Input.tsx` renders a fully rounded border on all four sides and has no slash preview UI.
- `tui/src/App.tsx` only recognizes slash commands at submit time via `text.startsWith("/")`; there is no pre-submit slash state.
- `tui/src/hooks/usePromptHistory.ts` already exposes the value and cursor offset needed for preview logic, but it has no notion of suggestion state or token replacement.
- The real built-in slash commands currently live in `cmd/gocode/slash_commands.go` and are mirrored in `README.md`; the TUI protocol does not currently expose that catalog.
- The existing `plan.md` and `progress.md` were stale and unrelated to this task, so they must be replaced rather than followed as-is.

## Original Source Reference

- `sourcecode/components/PromptInput/PromptInput.tsx`
	- This is the most relevant visual reference for the requested input chrome.
	- It uses `borderStyle="round"` with `borderLeft={false}` and `borderRight={false}`, which produces the top-and-bottom-only border treatment requested in the screenshot.
	- It keeps the prompt row separate from the suggestion rendering path.
- `sourcecode/components/PromptInput/PromptInputFooter.tsx`
	- This is the clearest reference for where suggestions should render relative to the prompt.
	- In non-fullscreen mode it renders suggestions outside the input body with `paddingX={2}`.
	- In fullscreen mode it portals suggestions only to escape clipping, not because the suggestion list fundamentally belongs inside the input box.
- `sourcecode/components/PromptInput/PromptInputFooterSuggestions.tsx`
	- This provides the useful suggestion row contract: a small `SuggestionItem` model, selected-row highlighting, width stabilization, and a capped visible list.
- `sourcecode/utils/suggestions/commandSuggestions.ts`
	- This is the most relevant behavioral reference for slash-command preview.
	- It only shows command suggestions while the user is still editing the command token.
	- It stops suggesting once arguments begin.
	- It formats accepted commands as `/<command> ` and immediately executes only no-argument commands on Enter.
- `sourcecode/context/promptOverlayContext.tsx` and `sourcecode/components/FullscreenLayout.tsx`
	- These explain the upstream fullscreen portal mechanism.
	- They are reference material, not a required first-step dependency for gocode.

## Source Code Explanation

The upstream implementation splits this feature into three separate concerns:

1. prompt chrome
2. suggestion state generation and ranking
3. suggestion presentation and layout

That split matters because gocode currently collapses most prompt behavior into `Input.tsx`. A direct port of the full upstream prompt stack would be larger than necessary for this task. The useful part to copy is the contract:

- the input row uses horizontal borders only
- suggestions are derived from the in-progress slash token, not from submit-time parsing
- suggestion rendering is handled by a separate component with stable row layout
- fullscreen portal logic is only needed if inline rendering hits clipping or layout problems

## Recommended Design

### 1. Keep slash command metadata authoritative in Go

Preferred approach:

- extend the startup payload emitted by the engine so the TUI receives a slash command catalog
- include at least:
	- `name`
	- `description`
	- `takes_arguments`
	- optional `usage`

Why:

- avoids drift between `cmd/gocode/slash_commands.go`, `README.md`, and the preview UI
- keeps the TUI synced when the built-in slash command list changes

Fallback if scope must stay smaller:

- create a TUI-local catalog that mirrors the current built-in commands
- acceptable only as a deliberate tradeoff, not the preferred long-term shape

### 2. Add a focused TUI hook for slash preview state

Add a small hook, for example `tui/src/hooks/useSlashCommandPreview.ts`, responsible for:

- detecting whether the cursor is still inside the leading slash token
- filtering and ranking commands
- tracking `selectedIndex`
- exposing handlers for `up`, `down`, `tab`, `enter`, and `escape`
- returning a compact list of display-ready items for rendering

Behavior contract:

- show suggestions only when the prompt starts with `/` and the cursor is still in the command token
- hide suggestions once the user starts typing arguments after a space
- rank exact and prefix matches above weaker fuzzy matches
- default selection to the first result
- `Tab`: insert the selected command plus trailing space, do not submit
- `Enter`: execute immediately only for commands that take no arguments; otherwise insert the command plus trailing space and keep focus in the input
- `Up` and `Down`: cycle suggestions before history navigation
- `Escape`: clear preview first, then fall back to the current cancel behavior

### 3. Keep the prompt row and the preview list visually separate

Recommended first implementation:

- keep `Input.tsx` as the prompt row renderer and primary keyboard owner
- add a dedicated preview renderer such as `tui/src/components/SlashCommandPreview.tsx`
- render the preview list immediately below the prompt row in the bottom prompt area

Recommendation:

- start with inline rendering below the prompt row to match the screenshot
- do not import the full upstream overlay/provider system in phase 1
- only introduce portal or overlay plumbing if inline rendering causes transcript clipping or obvious layout regressions

### 4. Restyle the prompt chrome to match the screenshot

Prompt row changes in `tui/src/components/Input.tsx`:

- switch from all-sides rounded border to `borderStyle="round"` with `borderLeft={false}` and `borderRight={false}`
- keep top and bottom borders enabled
- increase padding so the prompt feels closer to the screenshot’s block spacing
- preserve current multi-line editor behavior and cursor rendering

Visual guardrails:

- suggestions should feel like a separate list, not content inside the bordered prompt row
- prompt row height should remain stable with and without suggestions
- narrow terminals must still wrap safely

## Planned File Touches

- `cmd/gocode/slash_commands.go`
	- expose slash command descriptors from the current built-in command source of truth
- `cmd/gocode/engine.go`
	- emit the slash command catalog during startup
- `tui/src/protocol/types.ts`
	- add typed payload support for slash command metadata in the startup protocol
- `tui/src/hooks/useEvents.ts`
	- store the received slash command catalog in UI state
- `tui/src/hooks/useSlashCommandPreview.ts`
	- new hook for detection, ranking, selection, and apply logic
- `tui/src/components/Input.tsx`
	- prompt chrome restyle and preview-aware keyboard routing
- `tui/src/components/SlashCommandPreview.tsx`
	- new preview list renderer
- `tui/src/App.tsx`
	- pass slash metadata into the input, or render the preview component directly below it depending on the final component split
- `README.md`
	- update only if the final implementation adds user-visible keybindings or behavior worth documenting

## Implementation Phases

### Phase 1: Planning and scope lock

- replace the stale `plan.md` and `progress.md`
- document the upstream references and the implementation shape
- do not implement anything in this phase

### Phase 2: Backend slash catalog exposure

- define a Go-side slash command descriptor list from the existing built-in commands
- emit it to the TUI during startup
- keep slash command execution and help text behavior unchanged

### Phase 3: TUI preview state

- add the slash preview hook
- wire preview state to the current prompt value and cursor position
- make suggestion navigation take priority over history only while preview is active

### Phase 4: Prompt chrome restyle

- update the input border to top and bottom only
- add padding that matches the screenshot more closely
- render the preview list as a separate block below the prompt row

### Phase 5: Integration and manual verification

- verify `/` shows the command list
- verify `/pl` narrows to matching commands
- verify typing a space after a slash command hides the preview
- verify `Up` and `Down` navigate suggestions before history
- verify `Tab` inserts the selected command without submitting
- verify `Enter` handles no-argument and argument-taking commands correctly
- verify non-slash prompts behave exactly as before
- verify narrow terminals still render safely
- do not add tests

## Risks And Guardrails

- do not port the full upstream typeahead stack; gocode only needs slash-command preview for this task
- do not duplicate slash command truth in multiple places if backend emission is feasible
- do not break current history navigation, transcript search, or cancel behavior
- do not change slash command execution semantics
- do not add tests

## Exit Criteria

- the prompt row matches the screenshot’s top-and-bottom-only border treatment and roomier padding
- slash commands preview while the user is still typing the initial slash token
- preview rows show command name and description
- selection and apply behavior is predictable and does not break existing submit or history flows
- `plan.md` and `progress.md` reflect this task rather than the old refactor work

## Current Status

Planning only. No implementation has started.
