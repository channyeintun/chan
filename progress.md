# Progress

## Completed

### Phase 1: Add models.dev Snapshot Loader

- Added `nami/internal/modelsdev` with a `Client` that can `Load` and `Refresh` the `https://models.dev/api.json` snapshot.
- Added schema structs for providers, models, limits, modalities, cost, and experimental metadata.
- Added a local cache path helper at `nami/internal/config/paths.go` via `config.CacheDir()`.
- Implemented 24 hour cache freshness handling.
- Implemented stale-cache fallback when remote fetch fails.
- Implemented raw JSON cache writes under the user cache directory.
- Verified the task with `gofmt -w internal/config/paths.go internal/modelsdev/client.go` and `go build ./...`.

### Phase 2: Add Local Catalog Normalization

- Added `nami/internal/catalog` with a `Service` exposing `Snapshot`, `Provider`, and `Model` lookups.
- Normalized `models.dev` provider data into Nami catalog providers and models.
- Mapped known runtime providers onto the local runtime registry, including provider ID translation such as `google -> gemini`, `alibaba -> qwen`, and `zhipuai -> glm`.
- Filtered deprecated models during normalization.
- Preserved deterministic provider ordering through the existing provider priority metadata.
- Preserved deterministic model ordering by normalized display name and ID.
- Ensured provider defaults remain deterministic and can include config-selected local/custom models as fallback catalog entries.
- Verified the task with `gofmt -w internal/catalog/service.go` and `go build ./...`.

### Phase 3: Merge Config and Auth State

- Extended catalog providers with merged auth/config usability state.
- Merged environment-based API key availability, stored GitHub Copilot credentials, stored Codex credentials, `NAMI_API_KEY` active-provider overrides, and Ollama local usability into catalog provider status.
- Kept provider setup guidance and expired-token handling in the catalog snapshot so commands can consume a single source of truth.
- Updated `/providers` discovery to prefer the catalog-backed snapshot and fall back to the previous static path if catalog loading fails.
- Verified the task with `gofmt -w internal/catalog/service.go internal/commands/providers.go` and `go build ./...`.

### Phase 4: Replace Model Selection Source

- Updated model selection option building to prefer catalog-backed provider/model entries.
- `/model` and subagent model selection now enumerate real catalog models instead of only provider default models when the catalog is available.
- Preserved the existing provider-default option builder as a fallback when catalog loading fails.
- Kept curated presets prioritized ahead of the general catalog list.
- Kept custom model entry support and preserved current selection visibility when the active model is outside the catalog.
- Preserved explicit provider/model pairs in picker payloads for catalog-backed selections.
- Verified the task with `gofmt -w internal/commands/providers.go internal/engine/slash_command_model.go` and `go build ./...`.

## Deferred

- No tests were added because the current execution constraint says to never add tests.

## Next

- Phase 4.5: add a clearer catalog-to-runtime adapter layer for provider runtime wiring.
