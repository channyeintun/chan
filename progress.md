# Provider and Model Architecture Progress

Plan source: `enhancements.md`

Constraint: Do not add tests.

## Tasks

- [x] 1. Add a canonical model selection type
- [x] 2. Split provider registry from model catalog
- [x] 3. Centralize selection resolution
- [x] 4. Make capabilities model-specific
- [x] 5. Make provider overrides provider-scoped
- [x] 6. Move curated models into a model catalog
- [x] 7. Refactor the model picker around models first
- [x] 8. Decouple `/connect` from static provider presets
- [x] 9. Fix subagent provider resolution consistency
- [x] 10. Store recent model selection as structured data
- [x] 11. Split provider status from selection status
- [x] 12. Make client factories protocol-based
- [x] 13. Add provider/model architecture tests (skipped: user constraint forbids adding tests)

## Log

- 2026-05-21: Created progress tracker from `enhancements.md`.
- 2026-05-21: Added `config.ModelSelection` and `config.ResolvedModelSelection`; routed existing active, subagent, and engine model reference parsing through the canonical selection value while preserving current string boundaries.
- 2026-05-21: Split API provider configuration into `ProviderSpec`, `ModelSpec`, and `ProviderModelSupport` registries; generated legacy `api.Presets` from the split metadata and moved client factory lookup to provider specs.
- 2026-05-21: Added a shared `internal/modelselection` resolver for parsing, provider inference, compatibility checks, and resolution reasons; rewired command and engine selection paths away from duplicated inference logic.
- 2026-05-21: Added `api.ResolveModelCapabilities` with catalog, family, provider-default, and conservative fallback resolution; updated API clients to bind capabilities from the selected model instead of directly copying provider presets.
- 2026-05-21: Added provider-scoped config overrides for base URL, API key environment variable, and default model; migrated legacy top-level `base_url` into the active provider override at load time and applied overrides to provider status/client creation.
- 2026-05-21: Moved curated slash-command model entries into `api.CuratedModelCatalog` with model/provider IDs, family, cost warning, and alias metadata; updated the model picker to read curated entries from the catalog.
- 2026-05-21: Changed curated model picker construction to emit catalog model entries before provider-default routes, labeling each model with its provider route and provider usability while preserving fallback provider default options.
- 2026-05-21: Moved `/connect` provider catalog discovery to `api.ProviderSpecs`, changed provider ordering to use provider specs, and replaced the static engine connect registry with special-case handlers plus provider-spec-backed static handlers.
- 2026-05-21: Updated subagent fallback/configured model resolution to use `ResolveActiveSelection` and `ResolveSubagentSelection`, preserving explicit `SubagentProvider` and validating the resolved subagent route independently.
- 2026-05-21: Changed `recent-model.json` persistence to structured provider/model/explicit-provider fields with backward-compatible loading of legacy combined model strings; startup now restores recent selections from structured fields.
- 2026-05-21: Added `SelectionStatus` to provider snapshots, including requested/resolved selections plus separate provider usability and model support diagnostics; formatted provider output now shows selection status independently from provider setup state.
- 2026-05-21: Added `api.ProviderRoute` and `api.NewClientForRoute`; `NewClientForProvider` now resolves provider specs into a concrete route and dispatches client construction by protocol with route-level capabilities.
- 2026-05-21: Skipped provider/model architecture tests because the user explicitly forbids adding tests.
