# Provider and Model Architecture Enhancements

## Scope

This document covers Nami's current provider/model architecture only. No plugin migration is included.

The current implementation has already started separating `provider` and `model` in config, but the architecture is still partially coupled through provider presets, combined `provider/model` strings, duplicated inference heuristics, provider-scoped capabilities, and static provider-specific command behavior.

## Current Coupling Points

| Area | Current state | Why it still couples provider and model |
|---|---|---|
| Config | `config.Config` has `Provider`, `Model`, `SubagentProvider`, `SubagentModel` in `nami/nami/internal/config/config.go` | Runtime/session paths still frequently collapse selection into a single `provider/model` string. |
| Provider defaults | `api.ProviderPreset` stores `ClientType`, `BaseURL`, `EnvKeyVar`, `DefaultModel`, and `Capabilities` in `nami/nami/internal/api/provider_config.go` | Provider, default model, auth, transport, and capabilities are one object. |
| Resolution | `commands.ResolveModelSelection`, `commands.InferProviderFromModel`, `engine.inferProviderFromModel`, and `engine.isModelCompatibleWithProvider` all infer provider from model text | Compatibility is duplicated string matching instead of shared model/provider metadata. |
| Client creation | `api.NewClientForProvider` chooses a client from `api.Presets` in `nami/nami/internal/api/provider_factory.go` | Client construction depends on provider preset identity rather than a resolved provider transport config. |
| Capabilities | API clients copy `preset.Capabilities` during construction | Different models on the same provider can have different limits/features, but capabilities are mostly provider-level. |
| Slash model picker | Curated model presets live in `nami/nami/internal/engine/slash_command_handlers.go` | Model catalog and provider picker behavior are embedded in UI command code. |
| `/connect` | `connectProviderRegistry` and `ConnectProviderCatalog` are static/provider-first | Auth setup is provider-specific and not derived from a provider registry. |
| Recent model | `recent-model.json` stores one `model` string in `nami/nami/internal/config/recent_model.go` | It cannot distinguish explicit provider selection from derived provider inference. |
| Subagent selection | `ResolveSubagentSelection` exists, but subagent defaults still read `cfg.SubagentModel` directly in `provider_behavior.go` | `SubagentProvider` is not consistently part of subagent resolution. |

## Recommended Enhancements

### 1. Add a Canonical Model Selection Type

Introduce one internal representation and stop passing raw `provider/model` strings through core logic.

Suggested shape:

```go
type ModelSelection struct {
    ProviderID       string
    ModelID          string
    ExplicitProvider bool
    Source           string
}

type ResolvedModelSelection struct {
    Requested ModelSelection
    Resolved  ModelSelection
    Reason    string
}
```

Use strings only at CLI, config-file, session-file, and UI boundaries. Engine state, model switching, recent model selection, subagent selection, and status formatting should use the typed selection.

Primary files:
- `nami/nami/internal/config/config.go`
- `nami/nami/internal/engine/startup_provider.go`
- `nami/nami/internal/engine/slash_command_handlers.go`
- `nami/nami/internal/engine/model_state.go`
- `nami/nami/internal/session/store.go`

### 2. Split Provider Registry from Model Catalog

Replace `api.Presets` as the single source of truth with two concepts:

```go
type ProviderSpec struct {
    ID           string
    DisplayName  string
    Protocol     ClientType
    BaseURL      string
    EnvKeyVar    string
    DefaultModel string
    Priority     int
}

type ModelSpec struct {
    ID           string
    DisplayName  string
    Family       string
    Capabilities api.ModelCapabilities
}

type ProviderModelSupport struct {
    ProviderID string
    ModelID    string
    Protocol   ClientType
}
```

Providers become transports/auth endpoints. Models become model identities and capabilities. The support matrix answers whether a provider can serve a model.

This makes `gpt-5.4` a model that can be served by OpenAI, Codex, or GitHub Copilot without redefining it as three separate provider-owned strings.

Primary files:
- `nami/nami/internal/api/provider_config.go`
- `nami/nami/internal/api/provider_factory.go`
- `nami/nami/internal/commands/providers.go`
- `nami/nami/internal/engine/provider_behavior.go`

### 3. Centralize Selection Resolution

Create one resolver package for all model/provider decisions.

Responsibilities:
- Parse user input into `ModelSelection`.
- Preserve the active provider when it supports the requested model.
- Use explicit provider hints when provided.
- Fall back only through declared provider/model support metadata.
- Return a resolution reason for notices and status output.

This should replace the duplicated logic in:
- `commands.ResolveModelSelection`
- `commands.InferProviderFromModel`
- `engine.resolveModelSelection`
- `engine.inferProviderFromModel`
- `engine.isModelCompatibleWithProvider`
- `engine.retainSelectionProvider`

The resolver should not rely on scattered substring checks. Substring matching can remain only as a last-resort unknown-model heuristic with a visible `Reason`.

### 4. Make Capabilities Model-Specific

Provider-level capabilities are too coarse. `OpenAICompatClient`, `OpenAIResponsesClient`, `AnthropicClient`, `GeminiClient`, and `OllamaClient` currently mostly receive capabilities from provider presets.

Add a `ModelCapabilitiesResolver` that resolves capabilities in this order:
- Explicit model metadata from catalog.
- Provider-discovered remote metadata when available.
- Provider/model-family fallback metadata.
- Conservative default if unknown.

This improves:
- context-window reporting
- max output token selection
- vision support checks
- reasoning effort support
- tool-use support
- compaction thresholds

Primary files:
- `nami/nami/internal/api/client.go`
- `nami/nami/internal/api/provider_config.go`
- `nami/nami/internal/api/openai_reasoning.go`
- `nami/nami/internal/api/github_copilot.go`
- `nami/nami/internal/engine/engine.go`
- `nami/nami/internal/engine/engine_turns.go`

### 5. Make Provider Overrides Provider-Scoped

`Config.BaseURL` and `Config.APIKey` are global fields. That makes provider overrides depend on the currently selected provider and can leak intent across providers.

Add provider-scoped config:

```go
type ProviderOverride struct {
    BaseURL string `json:"base_url,omitempty"`
    APIKeyEnv string `json:"api_key_env,omitempty"`
    DefaultModel string `json:"default_model,omitempty"`
}

Providers map[string]ProviderOverride `json:"providers,omitempty"`
```

Do not persist raw API keys. Use environment variable names or existing credential structs.

Because current config files are persisted data, keep a real migration path from top-level `base_url` to the active provider override when loading existing configs.

Primary files:
- `nami/nami/internal/config/config.go`
- `nami/nami/internal/api/provider_factory.go`
- `nami/nami/internal/engine/provider_behavior.go`
- `nami/nami/internal/commands/providers.go`

### 6. Move Curated Models into a Model Catalog

`curatedModelSelectionPresets` currently lives in the slash command handler. It should be model metadata, not command code.

Move curated entries into the model catalog with fields for:
- display name
- family
- default/preferred providers
- cost warning label
- known capabilities
- aliases

The model picker can then render model-first choices and provider-specific routes under each model.

Primary files:
- `nami/nami/internal/engine/slash_command_handlers.go`
- `nami/nami/internal/commands/providers.go`
- new model catalog package or API subpackage

### 7. Refactor the Model Picker Around Models First

Current picker options are mostly provider defaults plus curated entries. A decoupled picker should show model choices first, then provider routes.

Suggested behavior:
- Show current model as `GPT 5.4 via Codex` when provider matters.
- Show model families grouped by catalog metadata.
- Prefer active provider if it supports the model.
- If multiple usable providers support the same model, choose active provider by default and expose alternates.
- If a provider does not support the selected model, fail with a clear reason instead of silently switching unless fallback is explicitly allowed.

Primary files:
- `nami/nami/internal/commands/providers.go`
- `nami/nami/internal/engine/slash_command_handlers.go`
- `nami/nami/tui/src/components/ModelSelectionPrompt.tsx`

### 8. Decouple `/connect` from Static Provider Presets

`ConnectProviderCatalog` is built directly from `api.Presets`, and engine handlers maintain a separate `connectProviderRegistry`.

Move auth metadata and connect actions into provider specs:
- API key env var setup
- local provider setup
- OAuth/device-flow handlers for special providers
- stored credential status

Special providers like GitHub Copilot and Codex can still have custom connector implementations, but `/connect` should discover those from the provider registry instead of maintaining a parallel static list.

Primary files:
- `nami/nami/internal/commands/connect.go`
- `nami/nami/internal/engine/slash_command_handlers.go`
- `nami/nami/internal/engine/codex_connect.go`
- `nami/nami/internal/api/provider_config.go`

### 9. Fix Subagent Provider Resolution Consistency

`ResolveSubagentSelection` exists, but subagent fallback and validation still use `cfg.SubagentModel` directly in parts of `provider_behavior.go`.

Enhance subagent handling so it always resolves through the same typed selection path as the main model:
- Preserve `SubagentProvider` when configured.
- Preserve active provider only when intended and supported.
- Validate provider/model support independently from the main model.
- Store and display subagent provider separately from subagent model.

Primary files:
- `nami/nami/internal/commands/providers.go`
- `nami/nami/internal/engine/provider_behavior.go`
- `nami/nami/internal/engine/slash_command_handlers.go`
- `nami/nami/internal/config/config.go`

### 10. Store Recent Model Selection as Structured Data

`recent-model.json` currently stores one string. Replace it with structured fields:

```go
type RecentModelSelection struct {
    Provider string `json:"provider,omitempty"`
    Model string `json:"model,omitempty"`
    ExplicitProvider bool `json:"explicit_provider,omitempty"`
    UpdatedAt string `json:"updated_at,omitempty"`
}
```

This avoids re-coupling generic models to whatever provider was inferred at save time. Startup can then decide whether to restore the provider, restore only the model, or preserve configured provider preference.

Primary files:
- `nami/nami/internal/config/recent_model.go`
- `nami/nami/internal/engine/recent_model.go`
- `nami/nami/internal/engine/startup_provider.go`

### 11. Split Provider Status from Selection Status

`ProviderSnapshot` currently mixes active provider/model with provider auth state. Add explicit selection diagnostics:

```go
type SelectionStatus struct {
    Requested ModelSelection
    Resolved ModelSelection
    ProviderUsable bool
    ModelSupported bool
    Reason string
}
```

`/providers` should answer provider setup. `/status` and `/model` should answer whether the current provider can serve the selected model.

This prevents a provider from appearing simply `usable` while the selected model is unsupported for that provider.

Primary files:
- `nami/nami/internal/commands/providers.go`
- `nami/nami/internal/commands/catalog.go`
- `nami/nami/internal/engine/startup_provider.go`
- `nami/nami/internal/engine/slash_command_handlers.go`

### 12. Make Client Factories Protocol-Based

`api.NewClientForProvider` currently looks up a provider preset, then a client factory by `ClientType`. In a cleaner architecture, the resolver should produce a concrete provider route:

```go
type ProviderRoute struct {
    ProviderID string
    ModelID string
    Protocol ClientType
    BaseURL string
    APIKeyEnv string
    Capabilities api.ModelCapabilities
}
```

Client creation then becomes protocol-based and does not need to infer defaults from provider globals.

Primary files:
- `nami/nami/internal/api/provider_factory.go`
- `nami/nami/internal/api/openai_compat.go`
- `nami/nami/internal/api/openai_responses.go`
- `nami/nami/internal/api/anthropic.go`
- `nami/nami/internal/api/gemini.go`
- `nami/nami/internal/api/ollama.go`

### 13. Add Provider/Model Architecture Tests

There are currently no focused provider/model tests under `nami/nami/internal/**/*_test.go`. Add tests before changing behavior.

Recommended tests:
- Model-only selection preserves active provider when supported.
- Explicit `provider/model` always wins over inferred provider.
- `Provider` + empty `Model` resolves to provider default.
- `SubagentProvider` is honored independently from main provider.
- Recent model restoration does not override an explicit CLI/env provider.
- Startup fallback reports the reason and does not pick local-only providers unless policy allows it.
- Provider status separates auth usability from model support.
- Capabilities resolve per model, not only per provider.
- `/model` picker options are model-first and deterministic.

Primary files:
- `nami/nami/internal/commands/providers_test.go`
- `nami/nami/internal/engine/provider_behavior_test.go`
- `nami/nami/internal/engine/startup_provider_test.go`
- `nami/nami/internal/api/provider_registry_test.go`

## Suggested Priority

| Priority | Enhancement | Reason |
|---|---|---|
| P0 | Canonical model selection type | Removes string coupling from core paths. |
| P0 | Centralized resolver | Eliminates duplicated inference and inconsistent provider retention. |
| P0 | Provider/model tests | Locks current intended behavior before refactor. |
| P1 | Split provider registry and model catalog | Main architectural decoupling. |
| P1 | Model-specific capabilities | Prevents wrong context/output/vision/reasoning behavior. |
| P1 | Provider-scoped config overrides | Avoids global base URL/API key coupling. |
| P2 | Model-first picker | Improves UX after architecture is correct. |
| P2 | `/connect` registry integration | Reduces static provider-specific duplication. |

## Expected Outcome

After these enhancements, provider and model become separate concepts:
- A model is selected by model identity and capabilities.
- A provider is selected as a route that can serve that model.
- Provider choice can be explicit, inferred, retained, or fallback-selected with a clear reason.
- Capabilities come from the selected model route, not from provider defaults.
- Config, session state, recent model state, model picker, and provider status all preserve this distinction.
