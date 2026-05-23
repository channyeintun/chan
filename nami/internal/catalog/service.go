package catalog

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"github.com/channyeintun/nami/internal/api"
	"github.com/channyeintun/nami/internal/config"
	"github.com/channyeintun/nami/internal/modelsdev"
	"github.com/channyeintun/nami/internal/modelselection"
)

type SnapshotLoader interface {
	Load(ctx context.Context) (modelsdev.Snapshot, error)
}

type Service struct {
	Loader SnapshotLoader
}

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
	DefaultModel string
	Models       []Model
}

type ProviderSource string

const (
	ProviderSourceModelsDev ProviderSource = "models.dev"
	ProviderSourceConfig    ProviderSource = "config"
)

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

type ModelLimits struct {
	ContextWindow int
	PromptTokens  int
	OutputTokens  int
}

type ModelCost struct {
	Input      float64
	Output     float64
	CacheRead  float64
	CacheWrite float64
}

type ModelAPI struct {
	Attachment       bool
	Reasoning        bool
	ToolCall         bool
	Temperature      bool
	StructuredOutput bool
	OpenWeights      bool
	InputModalities  []string
	OutputModalities []string
}

type ModelRef struct {
	ProviderID string
	ModelID    string
}

type runtimeProvider struct {
	CatalogID string
	SourceID  string
	Spec      api.ProviderSpec
	Source    ProviderSource
}

var runtimeProviders = []runtimeProvider{
	{CatalogID: "github-copilot", SourceID: "github-copilot", Source: ProviderSourceModelsDev, Spec: mustProviderSpec("github-copilot")},
	{CatalogID: "codex", SourceID: "openai", Source: ProviderSourceConfig, Spec: mustProviderSpec("codex")},
	{CatalogID: "openai", SourceID: "openai", Source: ProviderSourceModelsDev, Spec: mustProviderSpec("openai")},
	{CatalogID: "anthropic", SourceID: "anthropic", Source: ProviderSourceModelsDev, Spec: mustProviderSpec("anthropic")},
	{CatalogID: "gemini", SourceID: "google", Source: ProviderSourceModelsDev, Spec: mustProviderSpec("gemini")},
	{CatalogID: "deepseek", SourceID: "deepseek", Source: ProviderSourceModelsDev, Spec: mustProviderSpec("deepseek")},
	{CatalogID: "mistral", SourceID: "mistral", Source: ProviderSourceModelsDev, Spec: mustProviderSpec("mistral")},
	{CatalogID: "groq", SourceID: "groq", Source: ProviderSourceModelsDev, Spec: mustProviderSpec("groq")},
	{CatalogID: "qwen", SourceID: "alibaba", Source: ProviderSourceModelsDev, Spec: mustProviderSpec("qwen")},
	{CatalogID: "glm", SourceID: "zhipuai", Source: ProviderSourceModelsDev, Spec: mustProviderSpec("glm")},
	{CatalogID: "ollama", SourceID: "", Source: ProviderSourceConfig, Spec: mustProviderSpec("ollama")},
}

func NewService(loader SnapshotLoader) *Service {
	return &Service{Loader: loader}
}

func (s *Service) Snapshot(ctx context.Context, cfg config.Config) (Snapshot, error) {
	if s == nil || s.Loader == nil {
		return Snapshot{}, nil
	}

	base, err := s.Loader.Load(ctx)
	if err != nil {
		return Snapshot{}, err
	}

	providers := make(map[string]Provider, len(runtimeProviders))
	for _, runtime := range runtimeProviders {
		provider := buildProvider(base, cfg, runtime)
		if provider.ID == "" {
			continue
		}
		providers[provider.ID] = provider
	}

	ordered := make([]Provider, 0, len(providers))
	defaults := make(map[string]string, len(providers))
	for _, provider := range providers {
		ordered = append(ordered, provider)
		defaults[provider.ID] = provider.DefaultModel
	}

	slices.SortFunc(ordered, func(a, b Provider) int {
		return cmp.Or(providerPriority(a.ID)-providerPriority(b.ID), strings.Compare(a.ID, b.ID))
	})

	return Snapshot{
		Providers: ordered,
		Defaults:  defaults,
		Active:    resolveActive(cfg, defaults),
	}, nil
}

func (s *Service) Provider(ctx context.Context, cfg config.Config, providerID string) (Provider, bool, error) {
	snapshot, err := s.Snapshot(ctx, cfg)
	if err != nil {
		return Provider{}, false, err
	}
	providerID = normalizeProviderID(providerID)
	for _, provider := range snapshot.Providers {
		if provider.ID == providerID {
			return provider, true, nil
		}
	}
	return Provider{}, false, nil
}

func (s *Service) Model(ctx context.Context, cfg config.Config, providerID string, modelID string) (Model, bool, error) {
	provider, ok, err := s.Provider(ctx, cfg, providerID)
	if err != nil || !ok {
		return Model{}, ok, err
	}
	modelID = strings.TrimSpace(modelID)
	for _, model := range provider.Models {
		if model.ID == modelID {
			return model, true, nil
		}
	}
	return Model{}, false, nil
}

func buildProvider(base modelsdev.Snapshot, cfg config.Config, runtime runtimeProvider) Provider {
	provider := Provider{
		ID:       runtime.CatalogID,
		Name:     runtime.Spec.DisplayName,
		BaseURL:  runtime.Spec.BaseURL,
		Protocol: runtime.Spec.Protocol,
		Source:   runtime.Source,
	}

	if sourceProvider, ok := base.Providers[runtime.SourceID]; ok {
		provider.Name = firstNonEmpty(sourceProvider.Name, provider.Name)
		provider.BaseURL = firstNonEmpty(sourceProvider.API, provider.BaseURL)
		provider.EnvKeys = append([]string(nil), sourceProvider.Env...)
		provider.Models = buildModels(runtime.CatalogID, sourceProvider.Models)
		provider.Source = ProviderSourceModelsDev
	}

	provider.BaseURL = strings.TrimSpace(cfg.ApplyProviderOverride(runtime.CatalogID).BaseURL)
	if provider.BaseURL == "" {
		provider.BaseURL = runtime.Spec.BaseURL
	}

	if envKey := strings.TrimSpace(cfg.ProviderAPIKeyEnv(runtime.CatalogID, runtime.Spec.EnvKeyVar)); envKey != "" {
		provider.EnvKeys = appendUnique(provider.EnvKeys, envKey)
	}

	provider.Models = ensureModel(provider.Models, runtime.Spec.DefaultModel, fallbackModel(runtime.CatalogID, runtime.Spec.DefaultModel))
	if overrideModel := strings.TrimSpace(cfg.ProviderDefaultModel(runtime.CatalogID, runtime.Spec.DefaultModel)); overrideModel != "" {
		provider.Models = ensureModel(provider.Models, overrideModel, fallbackModel(runtime.CatalogID, overrideModel))
		provider.DefaultModel = overrideModel
	} else {
		provider.DefaultModel = chooseDefaultModel(provider.Models, runtime.Spec.DefaultModel)
	}

	if provider.DefaultModel == "" {
		provider.DefaultModel = chooseDefaultModel(provider.Models, "")
	}

	if provider.Name == "" || (len(provider.Models) == 0 && provider.DefaultModel == "") {
		return Provider{}
	}

	return provider
}

func buildModels(providerID string, source map[string]modelsdev.Model) []Model {
	models := make([]Model, 0, len(source))
	for modelID, sourceModel := range source {
		if strings.EqualFold(strings.TrimSpace(sourceModel.Status), "deprecated") {
			continue
		}
		models = append(models, normalizeModel(providerID, modelID, sourceModel))
	}

	slices.SortFunc(models, func(a, b Model) int {
		return cmp.Or(strings.Compare(a.Name, b.Name), strings.Compare(a.ID, b.ID))
	})
	return models
}

func normalizeModel(providerID string, modelID string, source modelsdev.Model) Model {
	capabilities := api.ModelCapabilities{
		SupportsToolUse:          source.ToolCall,
		SupportsExtendedThinking: source.Reasoning,
		SupportsVision:           supportsVision(source.Modalities.Input),
		SupportsJsonMode:         source.StructuredOutput,
		SupportsCaching:          source.Cost.CacheRead > 0 || source.Cost.CacheWrite > 0,
		MaxContextWindow:         source.Limit.Context,
		MaxPromptTokens:          source.Limit.Input,
		MaxOutputTokens:          source.Limit.Output,
	}
	if capabilities == (api.ModelCapabilities{}) {
		capabilities = api.ResolveModelCapabilities(providerID, modelID)
	}

	return Model{
		ID:           firstNonEmpty(source.ID, modelID),
		Name:         firstNonEmpty(source.Name, modelID),
		Family:       source.Family,
		Status:       source.Status,
		Capabilities: capabilities,
		Limits: ModelLimits{
			ContextWindow: source.Limit.Context,
			PromptTokens:  source.Limit.Input,
			OutputTokens:  source.Limit.Output,
		},
		Cost: ModelCost{
			Input:      source.Cost.Input,
			Output:     source.Cost.Output,
			CacheRead:  source.Cost.CacheRead,
			CacheWrite: source.Cost.CacheWrite,
		},
		API: ModelAPI{
			Attachment:       source.Attachment,
			Reasoning:        source.Reasoning,
			ToolCall:         source.ToolCall,
			Temperature:      source.Temperature,
			StructuredOutput: source.StructuredOutput,
			OpenWeights:      source.OpenWeights,
			InputModalities:  append([]string(nil), source.Modalities.Input...),
			OutputModalities: append([]string(nil), source.Modalities.Output...),
		},
	}
}

func fallbackModel(providerID string, modelID string) Model {
	capabilities := api.ResolveModelCapabilities(providerID, modelID)
	return Model{
		ID:           modelID,
		Name:         modelID,
		Family:       inferFamily(modelID),
		Capabilities: capabilities,
		Limits: ModelLimits{
			ContextWindow: capabilities.MaxContextWindow,
			PromptTokens:  capabilities.MaxPromptTokens,
			OutputTokens:  capabilities.MaxOutputTokens,
		},
	}
}

func ensureModel(models []Model, modelID string, fallback Model) []Model {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return models
	}
	for _, model := range models {
		if model.ID == modelID {
			return models
		}
	}
	models = append(models, fallback)
	slices.SortFunc(models, func(a, b Model) int {
		return cmp.Or(strings.Compare(a.Name, b.Name), strings.Compare(a.ID, b.ID))
	})
	return models
}

func chooseDefaultModel(models []Model, preferred string) string {
	preferred = strings.TrimSpace(preferred)
	if preferred != "" {
		for _, model := range models {
			if model.ID == preferred {
				return preferred
			}
		}
	}
	if len(models) == 0 {
		return ""
	}
	return models[0].ID
}

func resolveActive(cfg config.Config, defaults map[string]string) ModelRef {
	if providerID := normalizeProviderID(cfg.Provider); providerID != "" {
		modelID := strings.TrimSpace(cfg.Model)
		if modelID == "" {
			modelID = defaults[providerID]
		}
		return ModelRef{ProviderID: providerID, ModelID: modelID}
	}

	resolved := modelselection.Resolve(strings.TrimSpace(cfg.Model), "", cfg.ModelSource)
	providerID := normalizeProviderID(resolved.Resolved.ProviderID)
	modelID := strings.TrimSpace(resolved.Resolved.ModelID)
	if providerID != "" && modelID == "" {
		modelID = defaults[providerID]
	}
	return ModelRef{ProviderID: providerID, ModelID: modelID}
}

func appendUnique(values []string, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return values
	}
	for _, existing := range values {
		if strings.EqualFold(existing, value) {
			return values
		}
	}
	return append(values, value)
}

func supportsVision(modalities []string) bool {
	for _, modality := range modalities {
		switch strings.ToLower(strings.TrimSpace(modality)) {
		case "image", "video":
			return true
		}
	}
	return false
}

func inferFamily(modelID string) string {
	if spec, ok := api.ModelSpecFor(modelID); ok {
		return spec.Family
	}
	return ""
}

func providerPriority(providerID string) int {
	if spec, ok := api.ProviderSpecFor(providerID); ok {
		return spec.Priority
	}
	return 1_000_000
}

func normalizeProviderID(providerID string) string {
	return strings.ToLower(strings.TrimSpace(providerID))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func mustProviderSpec(providerID string) api.ProviderSpec {
	spec, ok := api.ProviderSpecFor(providerID)
	if !ok {
		panic("missing provider spec: " + providerID)
	}
	return spec
}
