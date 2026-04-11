package compact

import "github.com/channyeintun/gocode/internal/api"

// CompactableTools lists tools whose old results can be safely truncated.
var CompactableTools = map[string]bool{
	"FileRead":  true,
	"Bash":      true,
	"Grep":      true,
	"Glob":      true,
	"WebSearch": true,
	"WebFetch":  true,
	"FileEdit":  true,
	"FileWrite": true,
}

const truncatedMarker = "[Old tool result content cleared]"

// TruncateToolResults replaces old tool results with a short marker.
// Only truncates results from compactable tools, preserving the most recent
// tool result of each type.
func TruncateToolResults(messages []api.Message) []api.Message {
	toolNamesByCallID := make(map[string]string)
	for _, msg := range messages {
		for _, toolCall := range msg.ToolCalls {
			toolNamesByCallID[toolCall.ID] = toolCall.Name
		}
	}

	// Find the last occurrence index for each compactable tool type.
	lastSeen := make(map[string]int)
	for i, msg := range messages {
		if msg.ToolResult == nil {
			continue
		}
		toolName := toolNamesByCallID[msg.ToolResult.ToolCallID]
		if !CompactableTools[toolName] {
			continue
		}
		lastSeen[toolName] = i
	}

	result := make([]api.Message, len(messages))
	copy(result, messages)

	for i, msg := range result {
		if msg.Role != api.RoleTool || msg.ToolResult == nil {
			continue
		}
		toolName := toolNamesByCallID[msg.ToolResult.ToolCallID]
		if !CompactableTools[toolName] {
			continue
		}
		// Don't truncate the most recent results
		if i == lastSeen[toolName] {
			continue
		}
		if len(msg.Content) > 200 || len(msg.ToolResult.Output) > 200 {
			toolResultCopy := *result[i].ToolResult
			toolResultCopy.Output = truncatedMarker
			result[i].Content = truncatedMarker
			result[i].ToolResult = &toolResultCopy
		}
	}

	return result
}
