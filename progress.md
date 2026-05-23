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

## Deferred

- No tests were added because the current execution constraint says to never add tests.

## Next

- Phase 3: merge config and auth state into the catalog-backed provider view.
