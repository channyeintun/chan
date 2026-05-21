package commands

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/channyeintun/nami/internal/api"
	"github.com/channyeintun/nami/internal/config"
	"github.com/channyeintun/nami/internal/ipc"
	"github.com/channyeintun/nami/internal/modelselection"
)

const (
	ansiReset = "\x1b[0m"
	ansiBold  = "\x1b[1m"
)

type ProviderStatus struct {
	ID           string
	Label        string
	DefaultModel string
	AuthSource   string
	Configured   bool
	Usable       bool
	SetupHint    string
	LastError    string
	Current      bool
}

type ProviderSnapshot struct {
	ActiveProvider string
	ActiveModel    string
	Selection      SelectionStatus
	Providers      []ProviderStatus
}

type SelectionStatus struct {
	Requested      config.ModelSelection
	Resolved       config.ModelSelection
	ProviderUsable bool
	ModelSupported bool
	Reason         string
}

func FormatProviderSnapshot(snapshot ProviderSnapshot) string {
	lines := make([]string, 0, len(snapshot.Providers)*2+4)
	if snapshot.ActiveProvider != "" && snapshot.ActiveModel != "" {
		lines = append(lines, fmt.Sprintf("Active selection: %s/%s", colorProviderName(snapshot.ActiveProvider), snapshot.ActiveModel))
	} else if snapshot.ActiveModel != "" {
		lines = append(lines, fmt.Sprintf("Active selection: %s", snapshot.ActiveModel))
	}
	if snapshot.Selection.Reason != "" {
		lines = append(lines, fmt.Sprintf("Selection status: provider usable %t · model supported %t · %s", snapshot.Selection.ProviderUsable, snapshot.Selection.ModelSupported, snapshot.Selection.Reason))
	}

	if firstUsable, ok := snapshot.FirstUsable(); ok {
		lines = append(lines, fmt.Sprintf("First usable: %s/%s", colorProviderName(firstUsable.ID), firstUsable.DefaultModel))
	} else {
		lines = append(lines, "First usable: none")
	}

	lines = append(lines, "", "Providers:")
	for _, status := range snapshot.Providers {
		marker := "  "
		if status.Current {
			marker = "* "
		}

		providerName := colorProviderName(padRight(status.ID, 16))
		line := fmt.Sprintf(
			"%s%s %-24s default %s · source %s",
			marker,
			providerName,
			ProviderStateLabel(status),
			status.DefaultModel,
			status.AuthSource,
		)
		lines = append(lines, strings.TrimRight(line, " "))
		if status.LastError != "" {
			lines = append(lines, "  Last error: "+status.LastError)
		}
		if !status.Usable && status.SetupHint != "" {
			lines = append(lines, "  Next: "+status.SetupHint)
		}
	}

	return strings.Join(lines, "\n")
}

func colorProviderName(name string) string {
	trimmed := normalizeProviderID(name)
	color := "\x1b[36m"
	switch trimmed {
	case "github-copilot":
		color = "\x1b[96m"
	case "codex":
		color = "\x1b[36m"
	case "openai":
		color = "\x1b[92m"
	case "anthropic":
		color = "\x1b[95m"
	case "gemini":
		color = "\x1b[93m"
	case "deepseek":
		color = "\x1b[94m"
	case "mistral":
		color = "\x1b[35m"
	case "groq":
		color = "\x1b[32m"
	case "qwen":
		color = "\x1b[96m"
	case "glm":
		color = "\x1b[94m"
	case "ollama":
		color = "\x1b[33m"
	}
	return ansiBold + color + name + ansiReset
}

func padRight(value string, width int) string {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) >= width {
		return trimmed
	}
	return trimmed + strings.Repeat(" ", width-len(trimmed))
}

func BuildModelSelectionOptions(snapshot ProviderSnapshot, currentSelection string) []ipc.ModelSelectionOptionPayload {
	currentProvider, currentModel := ResolveModelSelection(currentSelection)
	currentRef := providerModelRef(currentProvider, currentModel)
	options := make([]ipc.ModelSelectionOptionPayload, 0, len(snapshot.Providers)+1)
	seen := make(map[string]struct{}, len(snapshot.Providers)+1)

	if currentModel != "" && !matchesProviderDefault(snapshot, currentProvider, currentModel) {
		label := "Current selection"
		description := "Current session model"
		if status, ok := snapshot.LookupProvider(currentProvider); ok {
			label = fmt.Sprintf("Current: %s (%s) · %s", currentModel, status.Label, ProviderStateLabel(status))
			description = formatModelSelectionDescription(status)
		}
		options = append(options, ipc.ModelSelectionOptionPayload{
			Label:       label,
			Model:       currentModel,
			Provider:    currentProvider,
			Description: description,
			Active:      true,
		})
		seen[currentRef] = struct{}{}
	}

	appendProviderOptions := func(match func(ProviderStatus) bool) {
		for _, status := range snapshot.Providers {
			if !match(status) {
				continue
			}
			if !providerOwnsDefaultModel(status.ID, status.DefaultModel) {
				continue
			}
			ref := providerModelRef(status.ID, status.DefaultModel)
			if _, exists := seen[ref]; exists {
				continue
			}
			options = append(options, ipc.ModelSelectionOptionPayload{
				Label:       fmt.Sprintf("%s (%s Default) · %s", status.DefaultModel, status.Label, ProviderStateLabel(status)),
				Model:       status.DefaultModel,
				Provider:    status.ID,
				Description: formatModelSelectionDescription(status),
				Active:      strings.EqualFold(ref, currentRef),
			})
		}
	}

	appendProviderOptions(func(status ProviderStatus) bool { return status.Usable })
	appendProviderOptions(func(status ProviderStatus) bool { return !status.Usable })

	return options
}

func providerOwnsDefaultModel(providerID string, model string) bool {
	owner := InferProviderFromModel(model)
	return owner == "" || owner == normalizeProviderID(providerID)
}

func ResolveActiveSelection(cfg config.Config) (provider, model string) {
	selection := ResolveActiveModelSelection(cfg)
	return selection.ProviderID, selection.ModelID
}

func ResolveActiveModelSelection(cfg config.Config) config.ModelSelection {
	p := strings.TrimSpace(cfg.Provider)
	m := strings.TrimSpace(cfg.Model)
	if p != "" {
		if m == "" {
			if preset, ok := api.Presets[normalizeProviderID(p)]; ok {
				m = cfg.ProviderDefaultModel(normalizeProviderID(p), preset.DefaultModel)
			}
		}
		return config.NewModelSelection(normalizeProviderID(p), m, cfg.ModelSource, true)
	}
	return ResolveModelSelectionValue(m, cfg.ModelSource)
}

func ResolveSubagentSelection(cfg config.Config) (provider, model string) {
	selection := ResolveSubagentModelSelection(cfg)
	return selection.ProviderID, selection.ModelID
}

func ResolveSubagentModelSelection(cfg config.Config) config.ModelSelection {
	p := strings.TrimSpace(cfg.SubagentProvider)
	m := strings.TrimSpace(cfg.SubagentModel)
	if p != "" {
		if m == "" {
			if preset, ok := api.Presets[normalizeProviderID(p)]; ok {
				m = cfg.ProviderDefaultModel(normalizeProviderID(p), preset.DefaultModel)
			}
		}
		return config.NewModelSelection(normalizeProviderID(p), m, "subagent", true)
	}
	return ResolveModelSelectionValue(m, "subagent")
}

func DiscoverProviderSnapshot(cfg config.Config) ProviderSnapshot {
	activeProvider, activeModel := ResolveActiveSelection(cfg)
	snapshot := ProviderSnapshot{
		ActiveProvider: activeProvider,
		ActiveModel:    activeModel,
		Selection: SelectionStatus{
			Requested: config.NewModelSelection(activeProvider, activeModel, cfg.ModelSource, activeProvider != ""),
			Resolved:  config.NewModelSelection(activeProvider, activeModel, cfg.ModelSource, activeProvider != ""),
			Reason:    "active selection",
		},
		Providers: make([]ProviderStatus, 0, len(api.Presets)),
	}

	for _, providerID := range orderedProviderIDs() {
		preset := api.Presets[providerID]
		defaultModel := cfg.ProviderDefaultModel(providerID, preset.DefaultModel)
		envKey := cfg.ProviderAPIKeyEnv(providerID, preset.EnvKeyVar)
		status := ProviderStatus{
			ID:           providerID,
			Label:        providerDisplayLabel(providerID),
			DefaultModel: defaultModel,
			AuthSource:   "none",
			SetupHint:    providerSetupHint(providerID, envKey),
			Current:      providerID == activeProvider,
		}
		preset.EnvKeyVar = envKey
		preset.DefaultModel = defaultModel
		populateProviderStatus(&status, cfg, activeProvider, preset)
		if providerID == activeProvider {
			snapshot.Selection.ProviderUsable = status.Usable
			snapshot.Selection.ModelSupported = activeModel == "" || modelselection.IsModelCompatibleWithProvider(activeModel, activeProvider) || strings.EqualFold(activeModel, status.DefaultModel)
		}
		snapshot.Providers = append(snapshot.Providers, status)
	}

	return snapshot
}

func (snapshot ProviderSnapshot) FirstUsable() (ProviderStatus, bool) {
	for _, status := range snapshot.Providers {
		if status.Current && status.Usable {
			return status, true
		}
	}
	for _, status := range snapshot.Providers {
		if status.Usable {
			return status, true
		}
	}
	return ProviderStatus{}, false
}

func (snapshot ProviderSnapshot) LookupProvider(providerID string) (ProviderStatus, bool) {
	providerID = normalizeProviderID(providerID)
	for _, status := range snapshot.Providers {
		if status.ID == providerID {
			return status, true
		}
	}
	return ProviderStatus{}, false
}

func ResolveModelSelection(selection string) (string, string) {
	resolved := ResolveModelSelectionValue(selection, "")
	return resolved.ProviderID, resolved.ModelID
}

func ResolveModelSelectionValue(selection string, source string) config.ModelSelection {
	resolved := modelselection.Resolve(selection, "", source)
	return resolved.Resolved
}

func InferProviderFromModel(model string) string {
	return modelselection.InferProviderFromModel(model)
}

func orderedProviderIDs() []string {
	preferred := []string{
		"github-copilot",
		"codex",
		"openai",
		"anthropic",
		"gemini",
		"deepseek",
		"mistral",
		"groq",
		"qwen",
		"glm",
		"ollama",
	}
	ordered := make([]string, 0, len(api.ProviderSpecs))
	seen := make(map[string]struct{}, len(api.ProviderSpecs))
	for _, providerID := range preferred {
		if _, ok := api.ProviderSpecs[providerID]; !ok {
			continue
		}
		ordered = append(ordered, providerID)
		seen[providerID] = struct{}{}
	}

	extra := make([]string, 0, len(api.ProviderSpecs)-len(ordered))
	for providerID := range api.ProviderSpecs {
		if _, ok := seen[providerID]; ok {
			continue
		}
		extra = append(extra, providerID)
	}
	sort.Strings(extra)
	ordered = append(ordered, extra...)
	return ordered
}

func populateProviderStatus(status *ProviderStatus, cfg config.Config, activeProvider string, preset api.ProviderPreset) {
	if status == nil {
		return
	}

	switch status.ID {
	case "github-copilot":
		populateGitHubCopilotStatus(status, cfg, activeProvider)
	case "codex":
		populateCodexStatus(status, cfg, activeProvider, preset.EnvKeyVar)
	case "ollama":
		populateOllamaStatus(status, cfg, activeProvider)
	default:
		populateAPIKeyProviderStatus(status, cfg, activeProvider, preset.EnvKeyVar)
	}
}

func populateCodexStatus(status *ProviderStatus, cfg config.Config, activeProvider string, envKey string) {
	if activeProvider == status.ID && strings.TrimSpace(cfg.APIKey) != "" {
		status.AuthSource = "env:NAMI_API_KEY"
		status.Configured = true
		status.Usable = true
		status.SetupHint = ""
		return
	}

	if envKey != "" && strings.TrimSpace(os.Getenv(envKey)) != "" {
		status.AuthSource = "env:" + envKey
		status.Configured = true
		status.Usable = true
		status.SetupHint = ""
		return
	}

	creds := cfg.Codex
	if strings.TrimSpace(creds.RefreshToken) != "" {
		status.AuthSource = "stored OAuth"
		status.Configured = true
		status.Usable = true
		status.SetupHint = ""
		return
	}

	if strings.TrimSpace(creds.AccessToken) != "" {
		status.AuthSource = "stored access token"
		status.Configured = true
		if creds.ExpiresAtUnixMS > 0 && time.Now().UnixMilli() > creds.ExpiresAtUnixMS {
			status.LastError = "saved access token expired"
			status.SetupHint = "Run /connect codex to refresh credentials."
			return
		}
		status.Usable = true
		status.SetupHint = ""
	}
}

func populateGitHubCopilotStatus(status *ProviderStatus, cfg config.Config, activeProvider string) {
	if activeProvider == status.ID && strings.TrimSpace(cfg.APIKey) != "" {
		status.AuthSource = "env:NAMI_API_KEY"
		status.Configured = true
		status.Usable = true
		status.SetupHint = ""
		return
	}

	creds := cfg.GitHubCopilot
	if strings.TrimSpace(creds.GitHubToken) != "" {
		status.AuthSource = "stored device auth"
		status.Configured = true
		status.Usable = true
		status.SetupHint = ""
		return
	}

	if strings.TrimSpace(creds.AccessToken) != "" {
		status.AuthSource = "stored access token"
		status.Configured = true
		if creds.ExpiresAtUnixMS > 0 && time.Now().UnixMilli() > creds.ExpiresAtUnixMS {
			status.LastError = "saved access token expired"
			status.SetupHint = "Run /connect github-copilot to refresh credentials."
			return
		}
		status.Usable = true
		status.SetupHint = ""
		return
	}
}

func populateAPIKeyProviderStatus(status *ProviderStatus, cfg config.Config, activeProvider string, envKey string) {
	if activeProvider == status.ID && strings.TrimSpace(cfg.APIKey) != "" {
		status.AuthSource = "env:NAMI_API_KEY"
		status.Configured = true
		status.Usable = true
		status.SetupHint = ""
		return
	}

	if envKey != "" && strings.TrimSpace(os.Getenv(envKey)) != "" {
		status.AuthSource = "env:" + envKey
		status.Configured = true
		status.Usable = true
		status.SetupHint = ""
		return
	}
}

func populateOllamaStatus(status *ProviderStatus, cfg config.Config, activeProvider string) {
	if activeProvider == status.ID && strings.TrimSpace(cfg.APIKey) != "" {
		status.AuthSource = "env:NAMI_API_KEY"
	} else if strings.TrimSpace(os.Getenv("OLLAMA_API_KEY")) != "" {
		status.AuthSource = "env:OLLAMA_API_KEY"
	} else {
		status.AuthSource = "local"
	}
	status.Configured = true
	status.Usable = true
	status.SetupHint = "Ensure Ollama is running on http://localhost:11434."
}

func providerDisplayLabel(providerID string) string {
	switch providerID {
	case "github-copilot":
		return "GitHub Copilot"
	case "codex":
		return "Codex"
	case "openai":
		return "OpenAI"
	case "anthropic":
		return "Anthropic"
	case "gemini":
		return "Gemini"
	case "deepseek":
		return "DeepSeek"
	case "qwen":
		return "Qwen"
	case "glm":
		return "GLM"
	case "mistral":
		return "Mistral"
	case "groq":
		return "Groq"
	case "ollama":
		return "Ollama"
	default:
		return strings.TrimSpace(providerID)
	}
}

func providerSetupHint(providerID string, envKey string) string {
	switch providerID {
	case "github-copilot":
		return "Run /connect github-copilot."
	case "codex":
		return "Run /connect codex or set CODEX_ACCESS_TOKEN."
	case "ollama":
		return "Ensure Ollama is running on http://localhost:11434."
	default:
		if envKey == "" {
			return "Provider setup is required."
		}
		return fmt.Sprintf("Set %s.", envKey)
	}
}

func normalizeProviderID(provider string) string {
	return modelselection.NormalizeProvider(provider)
}

func ProviderStateLabel(status ProviderStatus) string {
	switch {
	case status.Usable:
		return "usable"
	case status.Configured:
		return "configured, needs attention"
	default:
		return "needs setup"
	}
}

func matchesProviderDefault(snapshot ProviderSnapshot, providerID string, model string) bool {
	status, ok := snapshot.LookupProvider(providerID)
	if !ok {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(model), strings.TrimSpace(status.DefaultModel))
}

func formatModelSelectionDescription(status ProviderStatus) string {
	parts := make([]string, 0, 2)
	if status.AuthSource != "" && status.AuthSource != "none" {
		parts = append(parts, status.AuthSource)
	}
	if !status.Usable && status.SetupHint != "" {
		parts = append(parts, status.SetupHint)
	}
	if len(parts) == 0 {
		parts = append(parts, "default provider model")
	}
	return strings.Join(parts, " · ")
}

func providerModelRef(providerID string, model string) string {
	providerID = normalizeProviderID(providerID)
	model = strings.TrimSpace(model)
	if providerID == "" {
		return model
	}
	if model == "" {
		return providerID
	}
	return providerID + "/" + model
}
