package tools

import (
	"fmt"
	"strings"
	"unicode"
)

func buildWebFetchResult(rawURL, prompt string, respondMode webFetchRespondMode, content webFetchContent) string {
	truncatedContent := truncateWebFetchMarkdown(content.Markdown)
	if respondMode == webFetchRespondModeMarkdown {
		if truncatedContent == "" {
			return "No readable content returned."
		}
		return truncatedContent
	}

	passages := extractRelevantPassages(truncatedContent, prompt)

	var builder strings.Builder
	fmt.Fprintf(&builder, "Fetched: %s\n", rawURL)
	fmt.Fprintf(&builder, "Status: %s\n", content.StatusText)
	if content.ContentType != "" {
		fmt.Fprintf(&builder, "Content-Type: %s\n", content.ContentType)
	}
	fmt.Fprintf(&builder, "Bytes: %d\n", content.Bytes)
	fmt.Fprintf(&builder, "Prompt: %s\n", strings.TrimSpace(prompt))

	if len(passages) > 0 {
		builder.WriteString("\nRelevant excerpts:\n")
		for index, passage := range passages {
			fmt.Fprintf(&builder, "\n%d. %s\n", index+1, passage)
		}
	} else if truncatedContent != "" {
		builder.WriteString("\nContent:\n\n")
		builder.WriteString(truncatedContent)
	} else {
		builder.WriteString("\nNo readable content returned.\n")
	}

	return strings.TrimSpace(builder.String())
}

func truncateWebFetchMarkdown(markdown string) string {
	markdown = strings.TrimSpace(markdown)
	if len(markdown) <= maxWebFetchMarkdownChars {
		return markdown
	}
	return markdown[:maxWebFetchMarkdownChars] + "\n\n[Content truncated due to length...]"
}

func parseWebFetchRespondMode(params map[string]any) (webFetchRespondMode, error) {
	value, ok := firstStringParam(params, "respond_with", "respondWith")
	if !ok {
		return webFetchRespondModeReport, nil
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", string(webFetchRespondModeReport):
		return webFetchRespondModeReport, nil
	case string(webFetchRespondModeMarkdown):
		return webFetchRespondModeMarkdown, nil
	default:
		return "", fmt.Errorf("web_fetch respond_with must be report or markdown")
	}
}

func parseWebFetchPrompt(params map[string]any, respondMode webFetchRespondMode) (string, error) {
	prompt, _ := stringParam(params, "prompt")
	prompt = strings.TrimSpace(prompt)
	if respondMode == webFetchRespondModeMarkdown {
		return prompt, nil
	}
	if prompt == "" {
		return "", fmt.Errorf("web_fetch requires prompt")
	}
	return prompt, nil
}

func extractRelevantPassages(markdown string, prompt string) []string {
	sections := splitWebFetchSections(markdown)
	if len(sections) == 0 {
		return nil
	}

	keywords := keywordSet(prompt)
	if len(keywords) == 0 {
		return firstNonEmptySections(sections, 3)
	}

	type scoredSection struct {
		text  string
		score int
	}

	scored := make([]scoredSection, 0, len(sections))
	for _, section := range sections {
		sectionKeywords := keywordSet(section)
		score := 0
		for keyword := range keywords {
			if _, ok := sectionKeywords[keyword]; ok {
				score++
			}
		}
		if score > 0 {
			scored = append(scored, scoredSection{text: section, score: score})
		}
	}

	if len(scored) == 0 {
		return firstNonEmptySections(sections, 3)
	}

	for left := 0; left < len(scored)-1; left++ {
		for right := left + 1; right < len(scored); right++ {
			if scored[right].score > scored[left].score {
				scored[left], scored[right] = scored[right], scored[left]
			}
		}
	}

	result := make([]string, 0, min(3, len(scored)))
	for _, item := range scored {
		result = append(result, item.text)
		if len(result) == 3 {
			break
		}
	}
	return result
}

func splitWebFetchSections(markdown string) []string {
	rawSections := strings.FieldsFunc(markdown, func(r rune) bool {
		return r == '\n' || r == '\r'
	})
	sections := make([]string, 0, len(rawSections))
	for _, section := range rawSections {
		section = strings.Join(strings.Fields(section), " ")
		if section != "" {
			sections = append(sections, section)
		}
	}
	return sections
}

func firstNonEmptySections(sections []string, limit int) []string {
	result := make([]string, 0, min(limit, len(sections)))
	for _, section := range sections {
		if strings.TrimSpace(section) == "" {
			continue
		}
		result = append(result, section)
		if len(result) == limit {
			break
		}
	}
	return result
}

func keywordSet(text string) map[string]struct{} {
	text = strings.ToLower(text)
	parts := strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	keywords := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		if len(part) < 3 {
			continue
		}
		keywords[part] = struct{}{}
	}
	return keywords
}
