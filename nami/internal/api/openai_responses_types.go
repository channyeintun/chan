package api

import "strings"

type openAIResponsesRequest struct {
	Model           string                          `json:"model"`
	Input           []map[string]any                `json:"input"`
	Tools           []openAIResponsesToolDefinition `json:"tools,omitempty"`
	MaxOutputTokens int                             `json:"max_output_tokens,omitempty"`
	Temperature     *float64                        `json:"temperature,omitempty"`
	Reasoning       *openAIResponsesReasoning       `json:"reasoning,omitempty"`
	Include         []string                        `json:"include,omitempty"`
	Stream          bool                            `json:"stream"`
	Store           bool                            `json:"store"`
}

type openAIResponsesReasoning struct {
	Effort  string `json:"effort,omitempty"`
	Summary string `json:"summary,omitempty"`
}

type openAIResponsesToolDefinition struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Parameters  any    `json:"parameters"`
	Strict      bool   `json:"strict"`
}

type openAIResponsesEnvelope struct {
	Type string `json:"type"`
}

type openAIResponsesOutputItemEvent struct {
	Item openAIResponsesOutputItem `json:"item"`
}

type openAIResponsesOutputItem struct {
	Type      string                          `json:"type"`
	ID        string                          `json:"id,omitempty"`
	CallID    string                          `json:"call_id,omitempty"`
	Name      string                          `json:"name,omitempty"`
	Arguments string                          `json:"arguments,omitempty"`
	Content   []openAIResponsesMessageContent `json:"content,omitempty"`
	Summary   []openAIResponsesSummaryPart    `json:"summary,omitempty"`
}

type openAIResponsesMessageContent struct {
	Type    string `json:"type,omitempty"`
	Text    string `json:"text,omitempty"`
	Refusal string `json:"refusal,omitempty"`
}

type openAIResponsesSummaryPart struct {
	Text string `json:"text,omitempty"`
}

type openAIResponsesReasoningDeltaEvent struct {
	Delta string `json:"delta"`
}

type openAIResponsesTextDeltaEvent struct {
	Delta string `json:"delta"`
}

type openAIResponsesToolArgumentsDeltaEvent struct {
	Delta string `json:"delta"`
}

type openAIResponsesToolArgumentsDoneEvent struct {
	Arguments string `json:"arguments"`
}

type openAIResponsesOutputTextDoneEvent struct {
	Text string `json:"text"`
}

type openAIResponsesCompletedEvent struct {
	Response struct {
		Status string                      `json:"status"`
		Usage  openAIResponsesUsage        `json:"usage"`
		Output []openAIResponsesOutputItem `json:"output,omitempty"`
	} `json:"response"`
}

type openAIResponsesUsage struct {
	InputTokens        int `json:"input_tokens,omitempty"`
	OutputTokens       int `json:"output_tokens,omitempty"`
	TotalTokens        int `json:"total_tokens,omitempty"`
	InputTokensDetails struct {
		CachedTokens int `json:"cached_tokens,omitempty"`
	} `json:"input_tokens_details"`
}

func (u openAIResponsesUsage) toUsage() *Usage {
	if u.InputTokens == 0 && u.OutputTokens == 0 && u.TotalTokens == 0 && u.InputTokensDetails.CachedTokens == 0 {
		return nil
	}
	inputTokens := max(u.InputTokens-u.InputTokensDetails.CachedTokens, 0)
	return &Usage{
		InputTokens:     inputTokens,
		OutputTokens:    u.OutputTokens,
		CacheReadTokens: u.InputTokensDetails.CachedTokens,
	}
}

type openAIResponsesFailedEvent struct {
	Response struct {
		Error struct {
			Code    string `json:"code,omitempty"`
			Message string `json:"message,omitempty"`
		} `json:"error"`
		IncompleteDetails struct {
			Reason string `json:"reason,omitempty"`
		} `json:"incomplete_details"`
	} `json:"response"`
}

type openAIResponsesErrorEvent struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

type openAIResponsesStreamState struct {
	currentTool      *openAIResponsesToolCallState
	currentText      strings.Builder
	sawReasoningText bool
	sawContentText   bool
	sawToolCall      bool
	sentStop         bool
}

type openAIResponsesToolCallState struct {
	ID        string
	Name      string
	Arguments strings.Builder
}
