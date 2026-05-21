package api

import (
	"encoding/json"
	"net/http"
	"strings"
)

func (c *AnthropicClient) resolveAPIKey() (string, error) {
	if c.apiKeyFunc != nil {
		return c.apiKeyFunc()
	}
	return c.apiKey, nil
}

func (c *AnthropicClient) resolveBaseURL(apiKey string) string {
	if c.provider != "github-copilot" {
		return c.baseURL
	}
	resolved := strings.TrimRight(GetGitHubCopilotBaseURL(apiKey, c.enterpriseDomain), "/")
	if resolved == "" {
		return c.baseURL
	}
	return resolved
}

func classifyAnthropicStatus(statusCode int, body []byte) error {
	var envelope anthropicErrorEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		message := strings.TrimSpace(string(body))
		if message == "" {
			message = http.StatusText(statusCode)
		}
		return &APIError{Type: classifyAnthropicErrorType(statusCode, "", message), StatusCode: statusCode, Message: message}
	}

	message := envelope.Error.Message
	if message == "" {
		message = http.StatusText(statusCode)
	}

	return &APIError{Type: classifyAnthropicErrorType(statusCode, envelope.Error.Type, message), StatusCode: statusCode, Message: message}
}

func classifyAnthropicErrorType(statusCode int, errorType, message string) APIErrorType {
	lowerType := strings.ToLower(errorType)
	lowerMessage := strings.ToLower(message)

	switch {
	case statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden:
		return ErrAuth
	case statusCode == http.StatusTooManyRequests || strings.Contains(lowerType, "rate_limit"):
		return ErrRateLimit
	case statusCode == 529 || statusCode >= http.StatusInternalServerError || strings.Contains(lowerType, "overloaded"):
		return ErrOverloaded
	case strings.Contains(lowerMessage, "prompt is too long") || strings.Contains(lowerMessage, "prompt too long") || strings.Contains(lowerMessage, "context length"):
		return ErrPromptTooLong
	case strings.Contains(lowerMessage, "max tokens"):
		return ErrMaxTokens
	default:
		return ErrUnknown
	}
}
