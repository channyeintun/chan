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

## Deferred

- No tests were added because the current execution constraint says to never add tests.

## Next

- Phase 2: add local catalog normalization on top of the new `modelsdev` snapshot loader.
