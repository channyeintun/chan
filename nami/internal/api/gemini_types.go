package api

type geminiGenerateContentRequest struct {
	Contents          []geminiContent        `json:"contents"`
	SystemInstruction *geminiContent         `json:"systemInstruction,omitempty"`
	Tools             []geminiTool           `json:"tools,omitempty"`
	GenerationConfig  geminiGenerationConfig `json:"generationConfig,omitempty"`
}

type geminiGenerationConfig struct {
	Temperature     *float64 `json:"temperature,omitempty"`
	MaxOutputTokens int      `json:"maxOutputTokens,omitempty"`
	StopSequences   []string `json:"stopSequences,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text             string                  `json:"text,omitempty"`
	Thought          bool                    `json:"thought,omitempty"`
	ThoughtSignature string                  `json:"thoughtSignature,omitempty"`
	InlineData       *geminiInlineData       `json:"inlineData,omitempty"`
	FunctionCall     *geminiFunctionCall     `json:"functionCall,omitempty"`
	FunctionResponse *geminiFunctionResponse `json:"functionResponse,omitempty"`
}

type geminiInlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

type geminiFunctionCall struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name"`
	Args any    `json:"args,omitempty"`
}

type geminiFunctionResponse struct {
	ID       string         `json:"id,omitempty"`
	Name     string         `json:"name"`
	Response map[string]any `json:"response,omitempty"`
}

type geminiTool struct {
	FunctionDeclarations []geminiFunctionDeclaration `json:"functionDeclarations,omitempty"`
}

type geminiFunctionDeclaration struct {
	Name                 string `json:"name"`
	Description          string `json:"description,omitempty"`
	ParametersJsonSchema any    `json:"parametersJsonSchema,omitempty"`
}

type geminiGenerateContentResponse struct {
	Candidates     []geminiCandidate     `json:"candidates,omitempty"`
	PromptFeedback *geminiPromptFeedback `json:"promptFeedback,omitempty"`
	UsageMetadata  *geminiUsageMetadata  `json:"usageMetadata,omitempty"`
	Error          *geminiErrorBody      `json:"error,omitempty"`
}

type geminiCandidate struct {
	Content      geminiContent `json:"content"`
	FinishReason string        `json:"finishReason,omitempty"`
}

type geminiPromptFeedback struct {
	BlockReason string `json:"blockReason,omitempty"`
}

type geminiUsageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount,omitempty"`
	CandidatesTokenCount int `json:"candidatesTokenCount,omitempty"`
	TotalTokenCount      int `json:"totalTokenCount,omitempty"`
}

func (u *geminiUsageMetadata) merge(other *geminiUsageMetadata) {
	if other == nil {
		return
	}
	if other.PromptTokenCount > 0 {
		u.PromptTokenCount = other.PromptTokenCount
	}
	if other.CandidatesTokenCount > 0 {
		u.CandidatesTokenCount = other.CandidatesTokenCount
	}
	if other.TotalTokenCount > 0 {
		u.TotalTokenCount = other.TotalTokenCount
	}
}

func (u geminiUsageMetadata) toUsage() *Usage {
	return &Usage{InputTokens: u.PromptTokenCount, OutputTokens: u.CandidatesTokenCount}
}

type geminiErrorEnvelope struct {
	Error *geminiErrorBody `json:"error,omitempty"`
}

type geminiErrorBody struct {
	Status  string `json:"status,omitempty"`
	Message string `json:"message,omitempty"`
}

type geminiStreamState struct {
	usage      geminiUsageMetadata
	stopReason string
	sentStop   bool
}
