# Progress

## Current

- Phase 5 subagent model selection is complete; optional protocol cleanup is next.

## Completed

- Phase 1 display cleanup completed.
	- Added a shared TUI helper for stripping provider prefixes.
	- Switched StatusBar, ModelSelectionPrompt, ResumeSelectionPrompt, StreamingAssistantMessage, and AssistantTextMessage to the shared helper.
- Phase 2 provider inference updates completed.
	- Added provider hints to curated model presets.
	- Carried provider hints through the model-selection IPC flow.
	- Kept GitHub Copilot as the routing authority when the active provider is Copilot.
- Phase 5 subagent model selection completed.
	- Added a /subagent slash command that reuses the existing model picker.
	- Stored the active subagent model as session-scoped state and persisted it in session metadata.
	- Reset the subagent model on new sessions and restored it on resumed sessions.
	- Surfaced the current subagent model in status output and /subagent help text.

## Next

- Optional Phase 4 cleanup: reduce provider/model exposure in IPC where it is only used for display.