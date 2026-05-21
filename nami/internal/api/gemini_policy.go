package api

import (
	"encoding/json"
	"net/http"
	"strings"
)

func geminiMajorVersion(modelID string) int {
	lower := strings.ToLower(modelID)
	lower = strings.TrimPrefix(lower, "models/")
	for _, prefix := range []string{"gemini-live-", "gemini-"} {
		if !strings.HasPrefix(lower, prefix) {
			continue
		}
		rest := lower[len(prefix):]
		i := 0
		for i < len(rest) && rest[i] >= '0' && rest[i] <= '9' {
			i++
		}
		if i == 0 {
			return 0
		}
		v := 0
		for _, ch := range rest[:i] {
			v = v*10 + int(ch-'0')
		}
		return v
	}
	return 0
}

func geminiModelName(model string) string {
	return strings.TrimPrefix(model, "models/")
}

func classifyGeminiStatus(statusCode int, body []byte) error {
	var envelope geminiErrorEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil || envelope.Error == nil {
		message := strings.TrimSpace(string(body))
		if message == "" {
			message = http.StatusText(statusCode)
		}
		return &APIError{Type: classifyGeminiErrorType(statusCode, "", message), StatusCode: statusCode, Message: message}
	}

	message := envelope.Error.Message
	if message == "" {
		message = http.StatusText(statusCode)
	}
	return &APIError{Type: classifyGeminiErrorType(statusCode, envelope.Error.Status, message), StatusCode: statusCode, Message: message}
}

func classifyGeminiErrorType(statusCode int, status, message string) APIErrorType {
	lowerStatus := strings.ToLower(status)
	lowerMessage := strings.ToLower(message)

	switch {
	case statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden || strings.Contains(lowerStatus, "permission"):
		return ErrAuth
	case statusCode == http.StatusTooManyRequests || strings.Contains(lowerStatus, "resource_exhausted"):
		return ErrRateLimit
	case statusCode >= http.StatusInternalServerError || strings.Contains(lowerStatus, "unavailable"):
		return ErrOverloaded
	case strings.Contains(lowerMessage, "token") && strings.Contains(lowerMessage, "limit"):
		return ErrPromptTooLong
	case strings.Contains(lowerMessage, "max output tokens"):
		return ErrMaxTokens
	default:
		return ErrUnknown
	}
}

func mapGeminiStopReason(reason string) string {
	upper := strings.ToUpper(reason)
	switch upper {
	case "STOP":
		return "end_turn"
	case "MAX_TOKENS":
		return "max_tokens"
	case "MALFORMED_FUNCTION_CALL", "UNEXPECTED_TOOL_CALL", "TOOL_CALL", "FUNCTION_CALL":
		return "tool_use"
	default:
		return strings.ToLower(reason)
	}
}
