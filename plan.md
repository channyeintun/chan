# TUI Parity Plan

## Goal

Bring `go-cli/tui` as close as possible to the interaction model and visual behavior used in `sourcecode`, with priority on:

- main prompt input
- permission prompt input
- markdown rendering
- syntax highlighting
- transcript/message layout
- status/footer behavior

## Remaining Work

### Phase 6: Protocol follow-up

1. Evaluate whether rate-limit data (5-hour and 7-day windows) should be emitted for the status line.
2. Evaluate whether a cost-threshold setting should be exposed so the footer can show cost-threshold notices.
3. Evaluate whether the permission response payload should carry amendment/feedback text.
4. Evaluate whether assistant message state should become block-oriented instead of single-string per turn.

### Phase 7: Deferred infrastructure

1. Add scroll/fullscreen primitives to the TUI so a real virtualized transcript list can replace the anchored render cap.

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

- permission amendment/feedback parity requires engine protocol expansion
- block-oriented messages would be a significant refactor of both engine emitters and TUI reducers
- virtual transcript list depends on scroll/fullscreen primitives that Ink does not natively provide

## Definition of Done

- remaining protocol evaluation items are resolved (implemented or explicitly deferred)
- virtual transcript list is either implemented or the anchored render cap is confirmed as sufficient