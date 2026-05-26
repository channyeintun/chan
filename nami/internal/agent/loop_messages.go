package agent

import (
	"strings"

	"github.com/channyeintun/nami/internal/api"
)

func latestUserPrompt(messages []api.Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == api.RoleUser {
			return messages[i].Content
		}
	}
	return ""
}

func latestAssistantMessage(messages []api.Message) api.Message {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == api.RoleAssistant {
			return messages[i]
		}
	}
	return api.Message{Role: api.RoleAssistant}
}

func latestToolOutput(messages []api.Message) string {
	var builder strings.Builder
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if msg.Role != api.RoleTool {
			break
		}
		if msg.ToolResult != nil && strings.TrimSpace(msg.ToolResult.Output) != "" {
			builder.WriteString(msg.ToolResult.Output)
			builder.WriteString("\n")
		}
	}
	return builder.String()
}
