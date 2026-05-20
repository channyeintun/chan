package api

import (
	"fmt"
	"strings"
)

type ClientFactory func(provider, model, apiKey, baseURL string) (LLMClient, error)

type ProviderRoute struct {
	ProviderID   string
	ModelID      string
	Protocol     ClientType
	BaseURL      string
	APIKey       string
	Capabilities ModelCapabilities
}

var clientFactories = map[ClientType]ClientFactory{
	AnthropicAPI: func(provider, model, apiKey, baseURL string) (LLMClient, error) {
		return NewAnthropicClientForProvider(provider, model, apiKey, baseURL)
	},
	GeminiAPI: func(_ string, model, apiKey, baseURL string) (LLMClient, error) {
		return NewGeminiClient(model, apiKey, baseURL)
	},
	OpenAICompatAPI: func(provider, model, apiKey, baseURL string) (LLMClient, error) {
		return NewOpenAICompatClient(provider, model, apiKey, baseURL)
	},
	OpenAIResponsesAPI: func(provider, model, apiKey, baseURL string) (LLMClient, error) {
		return NewOpenAIResponsesClient(provider, model, apiKey, baseURL)
	},
	OllamaAPI: func(_ string, model, apiKey, baseURL string) (LLMClient, error) {
		return NewOllamaClient(model, apiKey, baseURL)
	},
}

func NewClientForProvider(provider, model, apiKey, baseURL string) (LLMClient, error) {
	provider = strings.TrimSpace(provider)
	if provider == "" {
		return nil, fmt.Errorf("provider is required")
	}

	spec, ok := ProviderSpecFor(provider)
	if !ok {
		return nil, fmt.Errorf("unsupported provider %q", provider)
	}
	if strings.TrimSpace(model) == "" {
		model = spec.DefaultModel
	}
	if strings.TrimSpace(baseURL) == "" {
		baseURL = spec.BaseURL
	}
	return NewClientForRoute(ProviderRoute{
		ProviderID:   provider,
		ModelID:      model,
		Protocol:     spec.Protocol,
		BaseURL:      baseURL,
		APIKey:       apiKey,
		Capabilities: ResolveModelCapabilities(provider, model),
	})
}

func NewClientForRoute(route ProviderRoute) (LLMClient, error) {
	provider := strings.TrimSpace(route.ProviderID)
	if provider == "" {
		return nil, fmt.Errorf("provider is required")
	}
	factory, ok := clientFactories[route.Protocol]
	if !ok {
		return nil, fmt.Errorf("unsupported client type %d for provider %q", route.Protocol, provider)
	}

	client, err := factory(provider, route.ModelID, route.APIKey, route.BaseURL)
	if err != nil {
		return nil, err
	}
	if route.Capabilities != (ModelCapabilities{}) {
		client = WithCapabilities(client, route.Capabilities)
	}
	return client, nil
}
