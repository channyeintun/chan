package api

import "strings"

func (c *OpenAIResponsesClient) resolveAPIKey() (string, error) {
	if c.apiKeyFunc != nil {
		return c.apiKeyFunc()
	}
	return c.apiKey, nil
}

func (c *OpenAIResponsesClient) resolveCodexAccountID() string {
	if c.codexAccountIDFunc != nil {
		if accountID := strings.TrimSpace(c.codexAccountIDFunc()); accountID != "" {
			return accountID
		}
	}
	return c.codexAccountID
}

func (c *OpenAIResponsesClient) resolveBaseURL(apiKey string) string {
	if c.provider != "github-copilot" {
		return c.baseURL
	}
	resolved := strings.TrimRight(GetGitHubCopilotBaseURL(apiKey, c.enterpriseDomain), "/")
	if resolved == "" {
		return c.baseURL
	}
	return resolved
}

func openAIResponsesUsesDeveloperRole(model string) bool {
	lower := strings.ToLower(strings.TrimSpace(model))
	prefixes := []string{"gpt-5", "o1", "o3", "o4"}
	for _, p := range prefixes {
		if strings.HasPrefix(lower, p) {
			return true
		}
	}
	return false
}

func openAIResponsesStopReason(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "completed", "in_progress", "queued", "":
		return "end_turn"
	case "incomplete":
		return "max_tokens"
	default:
		return "end_turn"
	}
}
