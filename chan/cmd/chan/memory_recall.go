package main

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/channyeintun/chan/internal/agent"
)

const (
	memoryRecallMaxCandidates = 32
	memoryRecallMaxSelections = 8
	memoryRecallMaxTerms      = 12
)

var memoryRecallTermPattern = regexp.MustCompile(`[a-z0-9][a-z0-9_./\-]{1,}`)

type memoryRecallSelector struct{}

type memoryRecallCandidate struct {
	ID       string
	Scope    string
	FilePath string
	Line     string
	Title    string
	FileType string
	NotePath string
	Updated  time.Time
	Index    int
}

func (s memoryRecallSelector) Select(ctx context.Context, files []agent.MemoryFile, userPrompt string) ([]agent.MemoryRecallResult, error) {
	_ = ctx
	if strings.TrimSpace(userPrompt) == "" {
		return nil, nil
	}

	candidates := buildMemoryRecallCandidates(files)
	if len(candidates) == 0 {
		return nil, nil
	}

	selected := selectMemoryRecallCandidates(candidates, userPrompt)
	if len(selected) == 0 {
		return nil, nil
	}

	return buildMemoryRecallResults(selected, "deterministic preference match"), nil
}

func buildMemoryRecallCandidates(files []agent.MemoryFile) []memoryRecallCandidate {
	candidates := make([]memoryRecallCandidate, 0, memoryRecallMaxCandidates)
	for _, file := range files {
		if file.Type != "project-index" && file.Type != "user-index" {
			continue
		}
		entries := agent.ParseMemoryIndexEntries(file)
		for _, entry := range entries {
			line := strings.TrimSpace(entry.RawLine)
			if line == "" || strings.TrimSpace(entry.Issue) != "" {
				continue
			}
			candidates = append(candidates, memoryRecallCandidate{
				ID:       fmt.Sprintf("m%d", len(candidates)+1),
				Scope:    file.Type,
				FilePath: file.Path,
				Line:     line,
				Title:    entry.Title,
				FileType: memoryRecallFirstNonEmpty(entry.NoteType, file.Type),
				NotePath: entry.NotePath,
				Updated:  file.UpdatedAt,
				Index:    entry.Order,
			})
			if len(candidates) >= memoryRecallMaxCandidates {
				return candidates
			}
		}
	}
	return candidates
}

func selectMemoryRecallCandidates(candidates []memoryRecallCandidate, userPrompt string) []memoryRecallCandidate {
	terms := extractMemoryRecallTerms(userPrompt)
	if len(terms) == 0 {
		return nil
	}

	type scoredCandidate struct {
		candidate memoryRecallCandidate
		score     int
	}

	scored := make([]scoredCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		score := scoreMemoryRecallCandidate(candidate, terms)
		if score <= 0 {
			continue
		}
		scored = append(scored, scoredCandidate{candidate: candidate, score: score})
	}
	if len(scored) == 0 {
		return nil
	}

	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		if memoryRecallScopeRank(scored[i].candidate.Scope) != memoryRecallScopeRank(scored[j].candidate.Scope) {
			return memoryRecallScopeRank(scored[i].candidate.Scope) < memoryRecallScopeRank(scored[j].candidate.Scope)
		}
		if !scored[i].candidate.Updated.Equal(scored[j].candidate.Updated) {
			return scored[i].candidate.Updated.After(scored[j].candidate.Updated)
		}
		if scored[i].candidate.FilePath != scored[j].candidate.FilePath {
			return scored[i].candidate.FilePath < scored[j].candidate.FilePath
		}
		return scored[i].candidate.Index < scored[j].candidate.Index
	})

	limit := min(memoryRecallMaxSelections, len(scored))
	selected := make([]memoryRecallCandidate, 0, limit)
	for _, candidate := range scored[:limit] {
		selected = append(selected, candidate.candidate)
	}
	return selected
}

func scoreMemoryRecallCandidate(candidate memoryRecallCandidate, terms []string) int {
	line := strings.ToLower(candidate.Line)
	title := strings.ToLower(candidate.Title)
	noteType := strings.ToLower(candidate.FileType)
	notePath := strings.ToLower(candidate.NotePath)
	noteBase := strings.ToLower(filepath.Base(candidate.NotePath))

	score := 0
	for _, term := range terms {
		switch {
		case noteBase != "." && strings.Contains(noteBase, term):
			score += 5
		case strings.Contains(title, term):
			score += 4
		case strings.Contains(line, term):
			score += 3
		case strings.Contains(noteType, term) || strings.Contains(notePath, term):
			score += 2
		}
	}
	if score > 0 && candidate.Scope == "project-index" {
		score++
	}
	return score
}

func buildMemoryRecallResults(candidates []memoryRecallCandidate, source string) []agent.MemoryRecallResult {
	if len(candidates) == 0 {
		return nil
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].FilePath != candidates[j].FilePath {
			return candidates[i].FilePath < candidates[j].FilePath
		}
		return candidates[i].Index < candidates[j].Index
	})

	byPath := make(map[string][]string)
	orderedPaths := make([]string, 0, len(candidates))
	seenPaths := make(map[string]struct{})
	for _, candidate := range candidates {
		if _, ok := seenPaths[candidate.FilePath]; !ok {
			seenPaths[candidate.FilePath] = struct{}{}
			orderedPaths = append(orderedPaths, candidate.FilePath)
		}
		byPath[candidate.FilePath] = append(byPath[candidate.FilePath], candidate.Line)
	}

	results := make([]agent.MemoryRecallResult, 0, len(orderedPaths))
	for _, path := range orderedPaths {
		results = append(results, agent.MemoryRecallResult{
			Path:   path,
			Lines:  byPath[path],
			Source: source,
		})
	}
	return results
}

func extractMemoryRecallTerms(prompt string) []string {
	matches := memoryRecallTermPattern.FindAllString(strings.ToLower(prompt), -1)
	if len(matches) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(matches))
	terms := make([]string, 0, min(memoryRecallMaxTerms, len(matches)))
	for _, match := range matches {
		if isLowSignalMemoryRecallTerm(match) {
			continue
		}
		if _, ok := seen[match]; ok {
			continue
		}
		seen[match] = struct{}{}
		terms = append(terms, match)
		if len(terms) >= memoryRecallMaxTerms {
			break
		}
	}
	return terms
}

func isLowSignalMemoryRecallTerm(term string) bool {
	if len(term) < 3 {
		return true
	}
	if strings.Contains(term, "/") || strings.Contains(term, ".") {
		return false
	}
	switch term {
	case "the", "and", "for", "with", "from", "into", "that", "this", "when", "then", "than", "have", "will", "want", "need", "make", "adds", "add", "use", "using", "used", "show", "help", "continue", "please", "user", "request", "current", "repo", "repository", "project", "code", "file", "files":
		return true
	default:
		return false
	}
}

func memoryRecallScopeRank(scope string) int {
	if scope == "project-index" {
		return 0
	}
	return 1
}

func memoryRecallFirstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
