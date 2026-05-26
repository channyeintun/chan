package agent

import (
	"context"
	"fmt"
	"strings"
)

func recallMemoryIndexes(
	ctx context.Context,
	recall func(context.Context, []MemoryFile, string) ([]MemoryRecallResult, error),
	files []MemoryFile,
	currentUserPrompt string,
	pressure ContextPressureDecision,
) ([]MemoryRecallResult, error) {
	if recall == nil || strings.TrimSpace(currentUserPrompt) == "" || pressure.SkipMemoryRecall {
		return nil, nil
	}

	hasMemoryIndexes := false
	for _, file := range files {
		if file.Type == memoryTypeProjectIndex || file.Type == memoryTypeUserIndex {
			hasMemoryIndexes = true
			break
		}
	}
	if !hasMemoryIndexes {
		return nil, nil
	}

	results, err := recall(ctx, files, currentUserPrompt)
	if err != nil {
		return nil, fmt.Errorf("memory recall unavailable: %w", err)
	}
	if len(results) == 0 {
		return nil, nil
	}
	return results, nil
}

type retrievalMeta struct {
	SnippetCount  int
	TokensUsed    int
	AnchorCount   int
	EdgesExpanded int
	Skipped       bool
}

func runLiveRetrieval(state *QueryState, currentUserPrompt string, pressure ContextPressureDecision) (string, retrievalMeta) {
	if pressure.SkipLiveRetrieval {
		return "", retrievalMeta{Skipped: true}
	}

	recentToolOutput := latestToolOutput(state.Messages)

	anchors := ExtractAnchors(currentUserPrompt, state.TurnContext.GitStatus, recentToolOutput, state.Graph)
	candidates, edgesExpanded := ScoreCandidates(anchors, state.TurnContext.CurrentDir, state.TurnContext.GitStatus, state.RetrievalTouched, state.Graph)
	snippets := ReadLiveSnippets(candidates, pressure.RetrievalBudgetTokens)

	section := FormatLiveRetrievalSection(snippets)
	tokensUsed := 0
	for _, snippet := range snippets {
		tokensUsed += len(snippet.Content) / 4
	}
	return section, retrievalMeta{
		SnippetCount:  len(snippets),
		TokensUsed:    tokensUsed,
		AnchorCount:   len(anchors),
		EdgesExpanded: edgesExpanded,
		Skipped:       false,
	}
}
