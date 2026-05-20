# Provider and Model Architecture Progress

Plan source: `enhancements.md`

Constraint: Do not add tests.

## Tasks

- [x] 1. Add a canonical model selection type
- [x] 2. Split provider registry from model catalog
- [ ] 3. Centralize selection resolution
- [ ] 4. Make capabilities model-specific
- [ ] 5. Make provider overrides provider-scoped
- [ ] 6. Move curated models into a model catalog
- [ ] 7. Refactor the model picker around models first
- [ ] 8. Decouple `/connect` from static provider presets
- [ ] 9. Fix subagent provider resolution consistency
- [ ] 10. Store recent model selection as structured data
- [ ] 11. Split provider status from selection status
- [ ] 12. Make client factories protocol-based
- [ ] 13. Add provider/model architecture tests (skipped: user constraint forbids adding tests)

## Log

- 2026-05-21: Created progress tracker from `enhancements.md`.
- 2026-05-21: Added `config.ModelSelection` and `config.ResolvedModelSelection`; routed existing active, subagent, and engine model reference parsing through the canonical selection value while preserving current string boundaries.
- 2026-05-21: Split API provider configuration into `ProviderSpec`, `ModelSpec`, and `ProviderModelSupport` registries; generated legacy `api.Presets` from the split metadata and moved client factory lookup to provider specs.
