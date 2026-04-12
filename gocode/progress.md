# Progress

## Current Phase

Planning complete. Implementation has not started.

## Completed

- [x] Compared the current TUI prompt against the requested screenshot.
- [x] Located the current gocode input and submit path in `tui/src/components/Input.tsx`, `tui/src/App.tsx`, and `tui/src/hooks/usePromptHistory.ts`.
- [x] Confirmed that gocode currently has no slash-command preview path; slash commands are only recognized on submit.
- [x] Reviewed the original source references in:
	- `sourcecode/components/PromptInput/PromptInput.tsx`
	- `sourcecode/components/PromptInput/PromptInputFooter.tsx`
	- `sourcecode/components/PromptInput/PromptInputFooterSuggestions.tsx`
	- `sourcecode/utils/suggestions/commandSuggestions.ts`
	- `sourcecode/context/promptOverlayContext.tsx`
	- `sourcecode/components/FullscreenLayout.tsx`
- [x] Determined that the existing `plan.md` and `progress.md` were stale for this task and replaced the planning docs with slash-preview work.
- [x] Wrote a task-specific implementation plan for slash preview and the prompt chrome restyle.

## Pending

- [ ] Expose slash command metadata from the Go engine to the TUI.
- [ ] Add TUI slash preview state and rendering.
- [ ] Restyle the input border and padding to match the screenshot.
- [ ] Manually verify key handling and narrow-terminal behavior.
- [ ] Update user-facing docs only if the final implementation changes visible prompt behavior or keybindings.

## Decisions

- Backend-provided slash command metadata is the preferred source of truth.
- The upstream fullscreen overlay system is reference material, not a required first implementation step for gocode.
- The first implementation should target screenshot parity with a preview list rendered separately from the bordered prompt row.
- No tests will be added.

## Detailed Step Log

### Task 1: Planning Assessment

Status: Completed

Steps completed:

1. Read the current TUI input flow and confirmed the absence of preview behavior.
2. Read the upstream source files that implement the relevant prompt chrome and suggestion behavior.
3. Mapped the visual requirements from the screenshot to the closest upstream prompt/input components.
4. Identified the current single source of truth for built-in slash commands in Go.
5. Replaced the stale planning docs with a task-specific plan for this feature.

Outcome:

- The top-and-bottom-only border treatment should follow the upstream prompt chrome pattern rather than the current all-sides border.
- Slash preview should be a pre-submit TUI concern, not more submit-time branching in `App.tsx`.
- The recommended architecture is a Go-emitted slash command catalog plus a small dedicated TUI preview hook and renderer.

## Working Rules

- Do not proceed to implementation during this planning phase.
- Do not add tests.
- Follow `plan.md` and `progress.md` for the implementation phase.
- After each later implementation task, update `progress.md`, run formatting for code changes, and commit with git commands.
