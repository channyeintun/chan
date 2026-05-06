# Plan: Provider and Dependency Updates

This is a planning-only document. No implementation changes should be made until explicitly approved.

## Goals

- Add `codex` as a first-class provider in the Go engine, with `gpt-5.5` as its default model.
- Use the opencode Codex reference at `reference/opencode/packages/opencode/src/plugin/codex.ts` for auth, endpoint, headers, model limits, and request behavior.
- Keep `gpt-5.4` as Nami's built-in GPT default while adding `gpt-5.5` as another surfaced option for that GPT generation.
- Update `web/docs.html` to document the new Codex provider and GPT 5.5 model selection.
- Replace the TUI's local Silvery dependency with a registry npm dependency instead of `file:vendor/silvery-local`.
- Add DeepSeek V4 Flash and DeepSeek V4 Pro to Nami's built-in DeepSeek support, following how opencode surfaces the official DeepSeek provider models from `models.dev`.

## Current Context

- Provider presets live in `nami/internal/api/provider_config.go` and are selected through `nami/internal/api/provider_factory.go`.
- `OpenAIResponsesClient` already exists in `nami/internal/api/openai_responses.go`, but normal presets currently route through chat completions unless provider behavior selects Responses explicitly.
- The opencode Codex plugin routes model requests to `https://chatgpt.com/backend-api/codex/responses` and allows `gpt-5.5`, setting larger limits for that model: context `400000`, input `272000`, output `128000`.
- The opencode Codex plugin authenticates with ChatGPT OAuth, stores `access`, `refresh`, `expires`, and optional `accountId`, refreshes expired tokens, and sends `ChatGPT-Account-Id` when present.
- Nami currently has built-in `gpt-5.4` references, including OpenAI and GitHub Copilot defaults; these should remain the defaults while `gpt-5.5` is added as another surfaced GPT option.
- The TUI currently declares `"silvery": "file:vendor/silvery-local"` in `nami/tui/package.json`, and `nami/tui/bun.lock` still records the same local file dependency.
- The vendored Silvery package metadata reports version `0.19.2`, so the registry dependency should target that version unless a newer compatible published version is intentionally chosen.
- opencode does not hard-code DeepSeek model IDs in provider source. Its `reference/opencode/packages/opencode/src/provider/models.ts` loads provider/model metadata from `https://models.dev/api.json`, with a cache and bundled snapshot fallback.
- The current `models.dev` official `deepseek` provider exposes `deepseek-v4-flash` and `deepseek-v4-pro` with names `DeepSeek V4 Flash` and `DeepSeek V4 Pro`, context limit `1000000`, output limit `384000`, reasoning enabled, tool calls enabled, and the same `DEEPSEEK_API_KEY` provider env var.
- Nami currently has a static `deepseek` preset in `nami/internal/api/provider_config.go` with default model `deepseek-chat`, context `64000`, and output `8192`; model inference already maps `deepseek` model IDs to the DeepSeek provider.

## DeepSeek V4 Plan

1. Update the DeepSeek provider preset.
   - Keep the provider ID `deepseek`, base URL `https://api.deepseek.com/v1`, env var `DEEPSEEK_API_KEY`, and OpenAI-compatible client path.
   - Change the default model to `deepseek-v4-flash` so `/connect deepseek` and provider defaults land on the faster v4 model.
   - Update static capabilities to match opencode/models.dev for the v4 official provider models: tool use enabled, extended reasoning enabled, context `1000000`, and output `384000`.

2. Surface both v4 model choices.
   - Audit curated model-selection suggestions and provider setup copy for DeepSeek model IDs.
   - Add `deepseek/deepseek-v4-flash` and `deepseek/deepseek-v4-pro` anywhere Nami lists built-in model choices.
   - Keep `deepseek-chat` and `deepseek-reasoner` compatibility only if they remain useful as explicit user-provided model IDs; do not leave them as the surfaced default.

3. Update documentation.
   - Update `README.md` and `web/docs.html` if they mention DeepSeek defaults or example model names.
   - Document `deepseek/deepseek-v4-flash` as the default fast path and `deepseek/deepseek-v4-pro` as the higher-capability option.

4. Run focused checks.
   - Run `gofmt` on edited Go files.
   - Run the Go engine build from `nami`.
   - Search for stale surfaced `deepseek-chat` defaults and confirm the new v4 IDs are present in provider/model selection paths.

## Codex Provider Plan

1. Add a Responses client type path.
   - Add an `OpenAIResponsesAPI` `ClientType` in `nami/internal/api/client.go`.
   - Register it in `clientFactories` in `nami/internal/api/provider_factory.go` with `NewOpenAIResponsesClient`.
   - Keep existing OpenAI-compatible chat completions providers unchanged.

2. Add the `codex` provider preset.
   - Add `codex` to `Presets` in `nami/internal/api/provider_config.go`.
   - Use `ClientType: OpenAIResponsesAPI`.
   - Use base URL `https://chatgpt.com/backend-api/codex`, because `OpenAIResponsesClient` appends `/responses`.
   - Use default model `gpt-5.5`.
   - Use an env fallback such as `CODEX_ACCESS_TOKEN` for manual bearer-token setup.
   - Set capabilities from the reference: tool use enabled, JSON mode if Responses schema behavior remains compatible, context `400000`, prompt/input `272000`, output `128000`.

3. Keep GPT 5.4 as the default while adding GPT 5.5.
   - Preserve built-in `gpt-5.4` defaults, including the OpenAI preset in `nami/internal/api/provider_config.go` and the GitHub Copilot main-model default in `nami/internal/api/github_copilot.go`.
   - Add `gpt-5.5` to curated model-selection options, provider setup copy, docs, web files, and examples without removing surfaced `gpt-5.4` guidance.
   - Keep the surfaced current GPT default on `gpt-5.4` while allowing `gpt-5.5` as an explicit newer alternative.

4. Add Codex-specific request behavior in the Responses client.
   - Add provider-specific headers for `codex`, matching the reference as closely as the current API layer allows: `originator: nami`, a stable `User-Agent`, and bearer `authorization`.
   - Add `ChatGPT-Account-Id` when stored auth has an account id.
   - Keep the request shape Responses-compatible.
   - Omit `max_output_tokens` for Codex requests to match the opencode plugin's `chat.params` override, while still keeping `MaxOutputTokens` in capabilities for budgeting.

5. Add Codex OAuth support.
   - Add `CodexAuth` to `nami/internal/config/config.go` with `AccessToken`, `RefreshToken`, `ExpiresAtUnixMS`, and `AccountID`.
   - Add `nami/internal/api/codex.go` for OAuth constants and token helpers based on the reference:
     - client id `app_EMoamEEZ73f0CkXaXp7hrann`
     - issuer `https://auth.openai.com`
     - browser PKCE flow on localhost port `1455`
     - headless device flow endpoints under the same issuer
     - token refresh via `/oauth/token`
     - JWT claim parsing for account id extraction
   - Add a Codex token refresher similar to the GitHub Copilot refresher so long-running sessions can refresh access tokens without restarting.

6. Wire Codex into provider setup and model selection.
   - Add `codex` to provider ordering, display labels, setup hints, and color handling in `nami/internal/commands/providers.go`.
   - Add Codex auth methods to `nami/internal/commands/connect.go`: browser OAuth, headless OAuth, and manual bearer token via env.
   - Add `connectCodex` to `connectProviderRegistry` in `nami/internal/engine/slash_command_handlers.go`.
   - Add a `codexProviderBehavior` in `nami/internal/engine/provider_behavior.go` so `newLLMClient` can load stored Codex auth, attach a token refresher, and select `codex/gpt-5.5` after `/connect codex`.
   - Keep bare `gpt-5.4` as the default OpenAI inference and allow bare `gpt-5.5` as another OpenAI choice; require `codex/gpt-5.5` or `/connect codex` for Codex to avoid changing existing model inference behavior.

7. Update GPT 5.5 reasoning handling.
   - Confirm whether `gpt-5.5` should allow `xhigh` reasoning.
   - If yes, add `gpt-5.5` to `SupportsXHighReasoningEffort` in `nami/internal/api/openai_reasoning.go`.

8. Update project documentation.
   - Update `web/docs.html` to add Codex provider setup and model-selection documentation.
   - Mention `codex/gpt-5.5` as the Codex model path and keep surfaced `gpt-5.4` guidance for the default OpenAI/Copilot path.
   - Keep the docs aligned with `/connect codex`, the chosen env var name, and any OAuth methods implemented.

9. Run focused compile/build checks after implementation.
   - Run the Go build path for the engine.
   - Exercise `/providers`, `/connect codex`, and model switching paths manually enough to confirm the provider appears and uses `codex/gpt-5.5`.
   - Confirm repository search still shows surfaced `gpt-5.4` defaults and also includes `gpt-5.5` where it is intended as an additional option.

## Silvery Dependency Plan

1. Replace the local file dependency.
   - Change `nami/tui/package.json` from `"silvery": "file:vendor/silvery-local"` to a registry version, preferably `"silvery": "^0.19.2"` to match the vendored package metadata.

2. Refresh the Bun lockfile from the registry.
   - Run the package-manager install from `nami/tui` so `nami/tui/bun.lock` records `silvery` from npm rather than `file:vendor/silvery-local`.
   - Confirm the lockfile also resolves Silvery peer and subdependencies from the registry, especially `@silvery/color` and `@silvery/commander` at compatible `0.19.x` versions.

3. Remove or retire the vendored copy.
   - Search for any remaining references to `vendor/silvery-local`.
   - If only the old dependency reference used it, delete `nami/tui/vendor/silvery-local` so future installs cannot accidentally fall back to the local package.
   - If release tooling unexpectedly relies on that folder, update that tooling to consume `node_modules/silvery` instead.

4. Run focused TUI checks after implementation.
   - Build the TUI from `nami/tui`.
   - Confirm no `file:vendor/silvery-local` or `silvery-local` references remain in tracked dependency files.

## Open Questions Before Implementation

- Should Codex support both browser and headless OAuth immediately, or should the first pass support one OAuth path plus manual bearer-token setup?
- Should the env fallback be named `CODEX_ACCESS_TOKEN`, `CODEX_API_KEY`, or both for convenience?
- Should `gpt-5.5` allow `xhigh` reasoning in Nami, matching the newer GPT 5 family behavior, or stay capped at `high` until confirmed?