# Progress

## Completed

- Added the `OpenAIResponsesAPI` client type and registered it in provider factory wiring.
- Added the `codex` provider preset with Responses API routing, `gpt-5.5`, `CODEX_ACCESS_TOKEN`, and Codex model limits.
- Restored active GPT defaults to `gpt-5.4` and kept `gpt-5.5` as an additional curated model selection.
- Added Codex Responses headers, account-id header support, and Codex payload behavior that omits `max_output_tokens`.
- Added Codex auth config storage, OAuth/device-flow token helpers, JWT account-id extraction, and a token refresher.
- Wired Codex into provider discovery, `/connect codex`, model switching, stored auth loading, and token refresh.
- Added `gpt-5.5` to `xhigh` reasoning support.
- Updated docs for Codex setup, `codex/gpt-5.5`, and the `gpt-5.4` default with `gpt-5.5` as another GPT option.
- Replaced the TUI Silvery local file dependency with registry `silvery@^0.19.2`, refreshed `bun.lock`, and removed the vendored copy.
- Explored how opencode sources DeepSeek model metadata from `models.dev` and updated `plan.md` for DeepSeek V4 Flash/Pro support.
- Updated the DeepSeek provider preset, curated model selection, and docs for DeepSeek V4 Flash/Pro.
- Ran focused formatting, diagnostics, stale-default search, and Go engine build checks for DeepSeek V4 Flash/Pro support.

## In Progress

- None.

## Pending

- None.
