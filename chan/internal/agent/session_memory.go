package agent

import "strings"

// SessionMemorySnapshot holds the current extracted session working state.
type SessionMemorySnapshot struct {
	ArtifactID string
	Title      string
	Content    string
	Version    int
}

// FormatSessionMemorySection renders extracted session continuity into the prompt.
func FormatSessionMemorySection(snapshot SessionMemorySnapshot) string {
	content := strings.TrimSpace(snapshot.Content)
	if content == "" {
		return ""
	}

	var b strings.Builder
	b.WriteString("<session_memory>\n")
	if title := strings.TrimSpace(snapshot.Title); title != "" {
		b.WriteString("Title: ")
		b.WriteString(title)
		b.WriteString("\n\n")
	}
	b.WriteString("This is extracted session working state for continuity across long turns and compaction. Treat it as session-scoped working memory, not durable project or user memory. Prefer it when reconstructing the active objective, current state, and pending work.\n\n")
	b.WriteString(content)
	if !strings.HasSuffix(content, "\n") {
		b.WriteString("\n")
	}
	b.WriteString("</session_memory>")
	return b.String()
}
