package engine

import (
	"context"
	"encoding/json"
	"sort"
	"strings"

	"github.com/channyeintun/chan/internal/agent"
	"github.com/channyeintun/chan/internal/api"
	artifactspkg "github.com/channyeintun/chan/internal/artifacts"
	"github.com/channyeintun/chan/internal/compact"
	"github.com/channyeintun/chan/internal/config"
	"github.com/channyeintun/chan/internal/ipc"
)

const (
	sessionMemoryArtifactSlot         = "active"
	sessionMemoryArtifactTitle        = "Session Memory"
	sessionMemoryArtifactSource       = "session-memory"
	sessionMemoryMinMessages          = 6
	sessionMemoryPeriodicTurnInterval = 4
	sessionMemoryMaxSectionItems      = 5
	sessionMemoryMaxSnippetLen        = 240
	sessionMemoryMaxFileCount         = 8
	sessionMemoryMaxChars             = 3200
)

type sessionMemoryDocument struct {
	Objective string
	State     string
	Files     []string
	Decisions []string
	Errors    []string
	NextSteps []string
}

func loadSessionMemorySnapshot(ctx context.Context, artifactManager *artifactspkg.Manager, sessionID string) (agent.SessionMemorySnapshot, error) {
	if !config.Load().EnableSessionMemory {
		return agent.SessionMemorySnapshot{}, nil
	}
	if artifactManager == nil || strings.TrimSpace(sessionID) == "" {
		return agent.SessionMemorySnapshot{}, nil
	}

	artifact, found, err := artifactManager.FindSessionArtifact(ctx, artifactspkg.KindSessionMemory, artifactspkg.ScopeSession, sessionID, sessionMemoryArtifactSlot)
	if err != nil || !found {
		return agent.SessionMemorySnapshot{}, err
	}

	loaded, content, err := artifactManager.LoadMarkdown(ctx, artifact.ID, 0)
	if err != nil {
		return agent.SessionMemorySnapshot{}, err
	}

	return agent.SessionMemorySnapshot{
		ArtifactID: loaded.ID,
		Title:      loaded.Title,
		Content:    content,
		Version:    loaded.Version,
		UpdatedAt:  loaded.UpdatedAt,
	}, nil
}

func maybeRefreshSessionMemory(ctx context.Context, bridge *ipc.Bridge, artifactManager *artifactspkg.Manager, sessionID string, turnID int, messages []api.Message, fromIndex int) error {
	if !config.Load().EnableSessionMemory {
		return nil
	}
	if artifactManager == nil || strings.TrimSpace(sessionID) == "" {
		return nil
	}
	if !shouldRefreshSessionMemory(ctx, artifactManager, sessionID, turnID, messages, fromIndex) {
		return nil
	}

	previous, err := loadSessionMemorySnapshot(ctx, artifactManager, sessionID)
	if err != nil {
		previous = agent.SessionMemorySnapshot{}
	}

	content := buildSessionMemoryMarkdown(previous, messages, fromIndex)
	if strings.TrimSpace(content) == "" {
		return nil
	}

	artifact, _, created, err := artifactManager.UpsertSessionMarkdown(ctx, artifactspkg.MarkdownRequest{
		Kind:    artifactspkg.KindSessionMemory,
		Scope:   artifactspkg.ScopeSession,
		Title:   sessionMemoryArtifactTitle,
		Source:  sessionMemoryArtifactSource,
		Content: content,
		Metadata: map[string]any{
			"status":                "active",
			"updated_turn":          turnID,
			"updated_message_count": len(messages),
		},
	}, sessionID, sessionMemoryArtifactSlot)
	if err != nil {
		if bridge == nil {
			return nil
		}
		return bridge.Emit(ipc.EventNotice, ipc.NoticePayload{Message: "session memory update skipped: " + err.Error()})
	}

	if bridge == nil {
		return nil
	}
	if created {
		if err := emitArtifactCreated(bridge, artifact); err != nil {
			return err
		}
	}
	return emitArtifactUpdated(bridge, artifact, content)
}

func shouldRefreshSessionMemory(ctx context.Context, artifactManager *artifactspkg.Manager, sessionID string, turnID int, messages []api.Message, fromIndex int) bool {
	if len(messages) < sessionMemoryMinMessages {
		return false
	}
	if turnHasCompactionSummary(messages, fromIndex) || turnHasToolActivity(messages, fromIndex) {
		return true
	}
	if turnID > 0 && turnID%sessionMemoryPeriodicTurnInterval == 0 {
		return true
	}
	_, found, err := artifactManager.FindSessionArtifact(ctx, artifactspkg.KindSessionMemory, artifactspkg.ScopeSession, sessionID, sessionMemoryArtifactSlot)
	return err == nil && !found
}

func turnHasToolActivity(messages []api.Message, fromIndex int) bool {
	if fromIndex < 0 {
		fromIndex = 0
	}
	for index := fromIndex; index < len(messages); index++ {
		message := messages[index]
		if len(message.ToolCalls) > 0 || message.ToolResult != nil {
			return true
		}
	}
	return false
}

func turnHasCompactionSummary(messages []api.Message, fromIndex int) bool {
	if fromIndex < 0 {
		fromIndex = 0
	}
	for index := fromIndex; index < len(messages); index++ {
		if compact.IsSummaryMessage(messages[index]) {
			return true
		}
	}
	return false
}

func buildSessionMemoryMarkdown(previous agent.SessionMemorySnapshot, messages []api.Message, fromIndex int) string {
	current := sessionMemoryDocument{
		Objective: firstNonEmptySnippet(recentUserSnippets(messages, 3)...),
		State:     firstNonEmptySnippet(recentAssistantSnippets(messages, fromIndex, 3)...),
		Files:     recentImportantFiles(messages, fromIndex),
		Decisions: recentDecisionSnippets(messages, fromIndex),
		Errors:    recentErrorSnippets(messages, fromIndex),
	}
	current.NextSteps = deriveNextSteps(current.Objective, current.Decisions, current.Errors)
	merged := mergeSessionMemoryDocuments(parseSessionMemoryMarkdown(previous.Content), current)

	var b strings.Builder
	b.WriteString("# Session Memory\n\n")
	b.WriteString("## Current Objective\n\n")
	b.WriteString(bulletOrFallback(merged.Objective, "Continue the current session objective."))
	b.WriteString("\n\n## Current State\n\n")
	b.WriteString(bulletOrFallback(merged.State, "Implementation work is active."))
	b.WriteString("\n\n## Important Files\n\n")
	b.WriteString(listOrFallback(merged.Files, "- No file focus captured yet."))
	b.WriteString("\n\n## Recent Decisions And Findings\n\n")
	b.WriteString(listOrFallback(merged.Decisions, "- No durable decisions captured yet."))
	b.WriteString("\n\n## Recent Errors And Corrections\n\n")
	b.WriteString(listOrFallback(merged.Errors, "- No recent errors captured."))
	b.WriteString("\n\n## Next Steps\n\n")
	b.WriteString(listOrFallback(merged.NextSteps, "- Continue from the latest user request and current file focus."))
	rendered := strings.TrimSpace(b.String()) + "\n"
	if len(rendered) > sessionMemoryMaxChars {
		return strings.TrimSpace(rendered[:sessionMemoryMaxChars]) + "\n"
	}
	return rendered
}

func parseSessionMemoryMarkdown(content string) sessionMemoryDocument {
	content = strings.TrimSpace(content)
	if content == "" {
		return sessionMemoryDocument{}
	}
	sections := splitMarkdownSections(content)
	return sessionMemoryDocument{
		Objective: strings.TrimPrefix(firstListEntry(sections["Current Objective"]), "- "),
		State:     strings.TrimPrefix(firstListEntry(sections["Current State"]), "- "),
		Files:     parseBulletList(sections["Important Files"]),
		Decisions: parseBulletList(sections["Recent Decisions And Findings"]),
		Errors:    parseBulletList(sections["Recent Errors And Corrections"]),
		NextSteps: parseBulletList(sections["Next Steps"]),
	}
}

func mergeSessionMemoryDocuments(previous, current sessionMemoryDocument) sessionMemoryDocument {
	merged := sessionMemoryDocument{
		Objective: firstNonEmptySnippet(current.Objective, previous.Objective),
		State:     firstNonEmptySnippet(current.State, previous.State),
		Files:     mergeBulletLists(current.Files, previous.Files, sessionMemoryMaxFileCount),
		Decisions: mergeBulletLists(current.Decisions, previous.Decisions, sessionMemoryMaxSectionItems),
		Errors:    mergeBulletLists(current.Errors, previous.Errors, sessionMemoryMaxSectionItems),
	}
	merged.NextSteps = mergeBulletLists(current.NextSteps, previous.NextSteps, sessionMemoryMaxSectionItems)
	if len(merged.NextSteps) == 0 {
		merged.NextSteps = deriveNextSteps(merged.Objective, merged.Decisions, merged.Errors)
	}
	return merged
}

func splitMarkdownSections(content string) map[string]string {
	sections := map[string]string{}
	var current string
	var buffer strings.Builder
	flush := func() {
		if current == "" {
			return
		}
		sections[current] = strings.TrimSpace(buffer.String())
		buffer.Reset()
	}
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			flush()
			current = strings.TrimSpace(strings.TrimPrefix(trimmed, "## "))
			continue
		}
		if current == "" {
			continue
		}
		buffer.WriteString(line)
		buffer.WriteString("\n")
	}
	flush()
	return sections
}

func parseBulletList(section string) []string {
	lines := strings.Split(strings.TrimSpace(section), "\n")
	items := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "- ") {
			items = append(items, trimmed)
			continue
		}
		items = append(items, "- "+trimmed)
	}
	return items
}

func firstListEntry(section string) string {
	items := parseBulletList(section)
	if len(items) == 0 {
		return ""
	}
	return items[0]
}

func mergeBulletLists(primary, fallback []string, limit int) []string {
	merged := make([]string, 0, limit)
	seen := make(map[string]struct{}, limit)
	appendItem := func(item string) {
		item = strings.TrimSpace(item)
		if item == "" {
			return
		}
		if !strings.HasPrefix(item, "- ") {
			item = "- " + item
		}
		if _, ok := seen[item]; ok {
			return
		}
		seen[item] = struct{}{}
		merged = append(merged, item)
	}
	for _, item := range primary {
		if limit > 0 && len(merged) >= limit {
			return merged
		}
		appendItem(item)
	}
	for _, item := range fallback {
		if limit > 0 && len(merged) >= limit {
			break
		}
		appendItem(item)
	}
	return merged
}

func recentUserSnippets(messages []api.Message, limit int) []string {
	results := make([]string, 0, limit)
	for index := len(messages) - 1; index >= 0 && len(results) < limit; index-- {
		message := messages[index]
		if message.Role != api.RoleUser {
			continue
		}
		snippet := normalizeSnippet(message.Content)
		if snippet != "" {
			results = append(results, snippet)
		}
	}
	return results
}

func recentAssistantSnippets(messages []api.Message, fromIndex int, limit int) []string {
	results := make([]string, 0, limit)
	if fromIndex < 0 {
		fromIndex = 0
	}
	for index := len(messages) - 1; index >= fromIndex && len(results) < limit; index-- {
		message := messages[index]
		if message.Role != api.RoleAssistant {
			continue
		}
		snippet := normalizeSnippet(message.Content)
		if snippet != "" {
			results = append(results, snippet)
		}
	}
	return results
}

func recentDecisionSnippets(messages []api.Message, fromIndex int) []string {
	items := recentAssistantSnippets(messages, fromIndex, sessionMemoryMaxSectionItems)
	for index := range items {
		items[index] = "- " + items[index]
	}
	return items
}

func recentErrorSnippets(messages []api.Message, fromIndex int) []string {
	if fromIndex < 0 {
		fromIndex = 0
	}
	items := make([]string, 0, sessionMemoryMaxSectionItems)
	for index := len(messages) - 1; index >= fromIndex && len(items) < sessionMemoryMaxSectionItems; index-- {
		message := messages[index]
		if message.ToolResult == nil || !message.ToolResult.IsError {
			continue
		}
		snippet := normalizeSnippet(message.ToolResult.Output)
		if snippet != "" {
			items = append(items, "- "+snippet)
		}
	}
	return items
}

func recentImportantFiles(messages []api.Message, fromIndex int) []string {
	if fromIndex < 0 {
		fromIndex = 0
	}
	seen := make(map[string]struct{}, sessionMemoryMaxFileCount)
	files := make([]string, 0, sessionMemoryMaxFileCount)
	for index := len(messages) - 1; index >= fromIndex && len(files) < sessionMemoryMaxFileCount; index-- {
		message := messages[index]
		if message.ToolResult != nil {
			path := strings.TrimSpace(message.ToolResult.FilePath)
			if path != "" {
				if _, ok := seen[path]; !ok {
					seen[path] = struct{}{}
					files = append(files, "- "+path)
				}
			}
		}
		for _, call := range message.ToolCalls {
			for _, path := range extractPathsFromToolInput(call.Input) {
				if _, ok := seen[path]; ok {
					continue
				}
				seen[path] = struct{}{}
				files = append(files, "- "+path)
				if len(files) >= sessionMemoryMaxFileCount {
					break
				}
			}
		}
	}
	sort.Strings(files)
	return files
}

func extractPathsFromToolInput(raw string) []string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
		return nil
	}
	keys := []string{"filePath", "path", "dirPath", "includePattern"}
	paths := make([]string, 0, len(keys))
	for _, key := range keys {
		value, ok := decoded[key].(string)
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)
		if value == "" || strings.ContainsAny(value, "*?[]") {
			continue
		}
		paths = append(paths, value)
	}
	return paths
}

func deriveNextSteps(objective string, decisions []string, errors []string) []string {
	steps := make([]string, 0, 3)
	if objective != "" {
		steps = append(steps, "- Continue the objective: "+objective)
	}
	if len(errors) > 0 {
		steps = append(steps, "- Resolve the latest error or failed tool path before expanding scope.")
	}
	if len(decisions) > 0 {
		steps = append(steps, "- Build on the recent decisions instead of re-reading the full transcript.")
	}
	return steps
}

func bulletOrFallback(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "- " + fallback
	}
	return "- " + value
}

func listOrFallback(items []string, fallback string) string {
	if len(items) == 0 {
		return fallback
	}
	return strings.Join(items, "\n")
}

func firstNonEmptySnippet(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func normalizeSnippet(value string) string {
	value = strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if value == "" {
		return ""
	}
	if len(value) > sessionMemoryMaxSnippetLen {
		return strings.TrimSpace(value[:sessionMemoryMaxSnippetLen]) + "..."
	}
	return value
}
