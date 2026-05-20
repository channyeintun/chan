package tools

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	defaultWebFetchTimeout         = 60 * time.Second
	maxWebFetchURLLength           = 2000
	maxWebFetchContentBytes  int64 = 10 * 1024 * 1024
	maxWebFetchMarkdownChars       = 100_000
	maxWebFetchCacheBytes    int64 = 50 * 1024 * 1024
	webFetchCacheTTL               = 15 * time.Minute
	maxWebFetchRedirects           = 10
	webFetchUserAgent              = "nami/0.1 (+https://github.com/channyeintun/nami)"
)

type webFetchRespondMode string

const (
	webFetchRespondModeReport   webFetchRespondMode = "report"
	webFetchRespondModeMarkdown webFetchRespondMode = "markdown"
)

// WebFetchTool fetches a URL, converts HTML to markdown, and returns prompt-focused content.
type WebFetchTool struct {
	client *http.Client
	cache  *webFetchCache
}

// NewWebFetchTool constructs the web fetch tool.
func NewWebFetchTool() *WebFetchTool {
	return &WebFetchTool{
		client: &http.Client{
			Timeout: defaultWebFetchTimeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= maxWebFetchRedirects {
					return errors.New("stopped after too many redirects")
				}
				return http.ErrUseLastResponse
			},
		},
		cache: newWebFetchCache(maxWebFetchCacheBytes, webFetchCacheTTL),
	}
}

func (t *WebFetchTool) Name() string {
	return "web_fetch"
}

func (t *WebFetchTool) Description() string {
	return "Fetch content from a URL, convert HTML to markdown, and return either prompt-focused excerpts or markdown."
}

func (t *WebFetchTool) InputSchema() any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"url": map[string]any{
				"type":        "string",
				"description": "The URL to fetch.",
			},
			"prompt": map[string]any{
				"type":        "string",
				"description": "What information to extract from the fetched content. Required unless respond_with is markdown.",
			},
			"respond_with": map[string]any{
				"type":        "string",
				"description": "Optional output mode. Use markdown to return the converted page markdown directly.",
				"enum":        []string{"report", "markdown"},
			},
			"respondWith": map[string]any{
				"type":        "string",
				"description": "CamelCase alias for respond_with.",
				"enum":        []string{"report", "markdown"},
			},
		},
		"required": []string{"url"},
	}
}

func (t *WebFetchTool) Permission() PermissionLevel {
	return PermissionReadOnly
}

func (t *WebFetchTool) Concurrency(input ToolInput) ConcurrencyDecision {
	return ConcurrencyParallel
}

func (t *WebFetchTool) Execute(ctx context.Context, input ToolInput) (ToolOutput, error) {
	rawURL, ok := stringParam(input.Params, "url")
	if !ok || strings.TrimSpace(rawURL) == "" {
		return ToolOutput{}, fmt.Errorf("web_fetch requires url")
	}
	respondMode, err := parseWebFetchRespondMode(input.Params)
	if err != nil {
		return ToolOutput{}, err
	}
	prompt, err := parseWebFetchPrompt(input.Params, respondMode)
	if err != nil {
		return ToolOutput{}, err
	}

	normalizedURL, err := normalizeWebFetchURL(rawURL)
	if err != nil {
		return ToolOutput{}, err
	}

	content, err := t.getMarkdownContent(ctx, normalizedURL)
	if err != nil {
		return ToolOutput{}, err
	}

	result := buildWebFetchResult(normalizedURL, prompt, respondMode, content)

	// Route substantial fetch results to a search-report artifact.
	const searchReportThreshold = 4000
	if respondMode == webFetchRespondModeReport && len(result) >= searchReportThreshold {
		if mutation, ok := saveSearchReportArtifact(ctx, normalizedURL, strings.TrimSpace(prompt), result); ok {
			return ToolOutput{Output: result, Artifacts: []ArtifactMutation{mutation}}, nil
		}
	}

	return ToolOutput{Output: result}, nil
}

type webFetchContent struct {
	URL         string
	StatusCode  int
	StatusText  string
	ContentType string
	Bytes       int
	Markdown    string
}
