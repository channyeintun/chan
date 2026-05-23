# Models.dev Provider Catalog Implementation Plan

## Goal

Redesign Nami's provider and model catalog around `models.dev` as the base source of truth, while keeping provider execution, authentication, and UI selection local to Nami.

The target design should avoid stretching the current static `ProviderSpecs` / `ModelCatalog` tables. Instead, introduce a first-class catalog pipeline:

```text
models.dev fetch/cache -> normalized catalog -> local config/auth/runtime merge -> commands/TUI/engine usage
```

No custom remote server is planned. Nami should fetch directly from `https://models.dev/api.json` and cache locally.

## Current Problems

- Provider and model data is mostly static in `nami/internal/api/provider_config.go`.
- Model selection is built around one default model per provider instead of a provider with many available models.
- Unknown model capability handling relies on heuristics.
- `/providers` mixes provider availability, defaults, and static provider definitions in one snapshot path.
- There is no single local catalog that can answer:
  - what providers exist
  - what models each provider exposes
  - which providers are locally usable
  - what the default model is for each provider
  - what capabilities and limits each model has

## Design Principles

- Treat `models.dev` as base catalog data, not as runtime authority.
- Keep network fetching isolated from provider runtime code.
- Keep provider execution independent from catalog listing.
- Preserve provider auth flows and provider clients.
- Keep curated model presets as UX shortcuts layered on top of the catalog.
- Prefer small Go packages with clear responsibilities.
- Do not add backward-compatibility shims unless a current config or session format needs them.
- Follow the opencode pattern: base catalog from `models.dev`, then local auth/runtime adapters and config-backed provider/model extensions.

## Proposed Package Layout

### `internal/modelsdev`

Responsible only for fetching, validating, caching, and refreshing the `models.dev` snapshot.

Suggested responsibilities:

- Define Go structs matching the `models.dev` provider/model schema.
- Fetch `https://models.dev/api.json` with timeout and user agent.
- Cache the raw JSON snapshot under Nami's cache directory.
- Load from cache when fresh.
- Fall back to stale cache if fetch fails and stale cache exists.
- Expose explicit refresh for future CLI support.

Suggested public API:

```go
type Client struct {
    // cache path, source URL, HTTP client, TTL
}

func (c *Client) Load(ctx context.Context) (Snapshot, error)
func (c *Client) Refresh(ctx context.Context) (Snapshot, error)
```

### `internal/catalog`

Responsible for converting `models.dev` data plus local Nami state into the catalog used by commands, TUI, and engine code.

Suggested responsibilities:

- Normalize provider IDs and model IDs.
- Convert models.dev providers/models into Nami catalog structs.
- Merge Nami config provider overrides.
- Merge local auth and environment availability.
- Support config-backed providers and local models as normal catalog entries.
- Filter deprecated models by default.
- Optionally include alpha/beta models behind a config flag later.
- Compute default model per provider.
- Produce provider status for `/providers` and model selection.

Suggested public API:

```go
type Service struct {
    // modelsdev loader, config loader, env reader, auth readers
}

func (s *Service) Snapshot(ctx context.Context, cfg config.Config) (Snapshot, error)
func (s *Service) Provider(ctx context.Context, cfg config.Config, providerID string) (Provider, bool, error)
func (s *Service) Model(ctx context.Context, cfg config.Config, providerID string, modelID string) (Model, bool, error)
```

Suggested domain types:

```go
type Snapshot struct {
    Providers []Provider
    Defaults  map[string]string
    Active    ModelRef
}

type Provider struct {
    ID           string
    Name         string
    BaseURL      string
    EnvKeys      []string
    Protocol     api.ClientType
    Source       ProviderSource
    Auth         AuthStatus
    DefaultModel string
    Models       []Model
}

type Model struct {
    ID           string
    Name         string
    Family       string
    Status       string
    Capabilities api.ModelCapabilities
    Limits       ModelLimits
    Cost         ModelCost
    API          ModelAPI
}

type ModelRef struct {
    ProviderID string
    ModelID    string
}
```

## Implementation Phases

### Phase 1: Add models.dev Snapshot Loader

Files likely involved:

- `nami/internal/modelsdev/...`
- cache path helper, either existing config/cache location or a new small helper

Tasks:

- Add models.dev schema structs.
- Add fetch with timeout.
- Add disk cache with TTL.
- Add stale-cache fallback.
- Add tests with local fixture JSON.

Acceptance criteria:

- Loader can parse a representative `models.dev/api.json` fixture.
- Loader does not require network in tests.
- Fetch failures return stale cache when available.
- Fetch failures without cache return a clear error.

### Phase 2: Add Local Catalog Normalization

Files likely involved:

- `nami/internal/catalog/...`
- `nami/internal/api/provider_config.go` only as temporary source of provider runtime protocol metadata

Tasks:

- Convert `modelsdev.Snapshot` into `catalog.Snapshot`.
- Map models.dev provider fields:
  - `provider.id` -> provider ID
  - `provider.name` -> display name
  - `provider.api` -> base URL
  - `provider.env` -> env keys
  - `provider.models` -> model list
- Map models.dev model fields into Nami capabilities and limits.
- Derive `api.ClientType` for known providers using a local provider runtime registry.
- Keep auth/runtime adapters separate from catalog normalization when models.dev does not fully describe runtime wiring.
- Support config-backed providers and local models in the same normalized catalog.
- Choose provider defaults deterministically.

Acceptance criteria:

- Catalog contains multiple models per provider.
- Known providers retain correct Nami runtime protocol.
- Deprecated models are hidden by default.
- Catalog has deterministic ordering.

### Phase 3: Merge Config and Auth State

Files likely involved:

- `nami/internal/catalog/...`
- `nami/internal/config/config.go`
- existing auth-related config structs

Tasks:

- Preserve provider overrides currently represented by `cfg.Providers`.
- Merge configured default model and base URL overrides into catalog providers.
- Mark providers usable when appropriate:
  - env API key exists
  - stored OAuth/device auth exists
  - local provider is assumed usable, such as Ollama
  - active provider has `NAMI_API_KEY` override
- Treat providers with login-based auth as normal catalog entries whose usability comes from their stored credentials or configured auth.
- Keep status text generation outside the catalog where possible, but expose enough state for commands/UI.

Acceptance criteria:

- `/providers` can be rebuilt from catalog data.
- Existing configured providers still resolve to the same active model unless explicitly changed.
- Existing auth sources are represented in provider status.

### Phase 4: Replace Model Selection Source

Files likely involved:

- `nami/internal/commands/providers.go`
- `nami/internal/engine/slash_command_model.go`
- `nami/internal/engine/slash_command_context.go`
- TUI model selection types only if payload shape needs expansion

Tasks:

- Replace `DiscoverProviderSnapshot` internals with catalog-backed data.
- Build model picker options from real catalog models, not just provider defaults.
- Keep curated presets as prioritized shortcuts.
- Keep custom model entry.
- Preserve active/current selection display.
- Support provider/model IDs from catalog without heuristic provider inference where possible.

Acceptance criteria:

- `/model` can list more than one model per provider.
- Curated models still appear near the top.
- Current model remains selectable even if not in catalog.
- Selection returns an explicit provider/model pair when selected from catalog.

### Phase 4.5: Add Auth/Runtime Adapter Layer

Files likely involved:

- provider client initialization paths
- auth/login flows
- catalog-to-runtime resolution code

Tasks:

- Keep provider runtime behavior separate from catalog listing.
- Model GitHub Copilot as a normal provider with login/auth and optional runtime model refresh.
- Model Codex as an `openai` auth/runtime mode, matching opencode's architecture.
- Keep local models and config-backed providers resolvable through the same catalog path.
- Ensure provider-specific request wiring does not leak into base catalog normalization.

Acceptance criteria:

- Catalog listing works independently of auth/runtime request logic.
- Login-capable providers resolve through the same provider/model selection flow as API-key providers.
- Local/config-backed models resolve through the same catalog path as remote catalog models.

### Phase 5: Move Capability Resolution to Catalog

Files likely involved:

- `nami/internal/api/provider_config.go`
- code paths calling `ResolveModelCapabilities`
- model/client initialization paths

Tasks:

- Use catalog model metadata for context windows, output limits, tool support, reasoning, vision, and caching when available.
- Keep family heuristics only as fallback for custom models.
- Remove duplicated static model capabilities once call sites are migrated.

Acceptance criteria:

- Known catalog models use models.dev-derived metadata.
- Custom unknown models still get safe fallback capabilities.
- Existing client behavior remains intact.

### Phase 6: Cleanup Static Catalog Usage

Files likely involved:

- `nami/internal/api/provider_config.go`
- `nami/internal/modelselection/resolver.go`
- `nami/internal/commands/providers.go`

Tasks:

- Reduce `ProviderSpecs` to a provider runtime registry if still needed.
- Remove `ModelCatalog` as primary source of model metadata.
- Replace support matrix logic with catalog lookup.
- Keep only provider runtime metadata, auth/runtime adapters, and curated presets.

Acceptance criteria:

- Static provider table is no longer the primary listing source.
- Static model catalog is either deleted or reduced to test/local fallback data.
- Provider/model selection behavior is catalog-first.

## Testing Plan

### Unit Tests

- Parse models.dev fixture into `modelsdev.Snapshot`.
- Cache freshness and stale fallback behavior.
- Normalize provider and model metadata.
- Merge config overrides.
- Mark auth/env availability correctly.
- Filter deprecated and optionally alpha models.
- Resolve default models deterministically.
- Keep catalog normalization independent from runtime adapter behavior.

### Integration Tests

- `/providers` output with fixture catalog and no credentials.
- `/providers` output with env credential.
- `/model` options include multiple models from provider catalog.
- Selecting a catalog model emits explicit provider/model.
- Existing configured model still resolves.
- Unknown custom model still works through custom model path.
- Login-capable providers still resolve through catalog-backed selection.

### Manual Verification

- Run provider listing with no network and warm cache.
- Run provider listing with no cache and network unavailable.
- Run `/model` with OpenAI, Anthropic, Gemini, Copilot, Codex, and Ollama scenarios.
- Verify model switch still initializes the expected client type.

Suggested commands after implementation:

```sh
go test ./...
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```

## Migration Notes

- Existing config format can remain mostly unchanged at first.
- Existing sessions storing `provider/model` should continue to resolve through catalog lookup.
- Bare model input should become less important, but can still be supported by resolving against current provider first, then catalog suggestions.
- If a model appears under multiple providers, UI selections should carry explicit provider ID to avoid ambiguous inference.
- Config-backed providers and local models should use the same resolution path as models.dev-backed entries.

## Risks

- `models.dev` may not contain every provider entry or the exact runtime protocol Nami needs.
- Provider protocol cannot always be inferred from models.dev alone; Nami still needs a local runtime registry.
- Network access should not block startup or model selection when cache exists.
- Large model catalogs may need search/ranking in the TUI rather than rendering everything flat.
- Some models.dev metadata may not match provider-specific API behavior exactly.
- Moving Codex under `openai` auth/runtime may require a one-time migration for existing `codex/...` session/config references.

## Architecture Decisions

- Cache TTL: use 24 hours for normal freshness, with stale-cache fallback on fetch failure. This avoids startup churn while keeping the catalog current enough for a CLI tool.
- Refresh command: expose `modelsdev.Refresh` immediately, but add user-facing refresh only after the catalog-backed `/providers` and `/model` flows are working. Prefer a non-interactive command-style path such as `/providers refresh` or a CLI `models --refresh`, rather than refreshing implicitly from model selection.
- Model status filtering: include active and beta models by default, hide deprecated models always, and hide alpha models unless an explicit experimental-models config flag is enabled.
- First-run offline fallback: embed a generated snapshot or minimal generated fallback catalog at build time. Use it only when there is no cache and fetching `models.dev` fails. Do not hand-maintain this fallback as a second source of truth.
- Codex: implement as `openai` auth/runtime support, not as a separate catalog provider. Existing `codex/...` references should migrate or resolve to `openai/...` with Codex auth where possible.
- GitHub Copilot: implement as a normal provider with login-based auth. Support provider-owned runtime model refresh later, but keep initial implementation catalog-backed with auth/runtime request wiring separate from catalog normalization.
- Local models: implement as config-backed provider/model entries in the same catalog path as remote catalog entries. Local providers should not bypass catalog resolution.
