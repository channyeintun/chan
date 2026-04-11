package main

import (
	"context"
	"fmt"
	"io"
	"iter"
	"path/filepath"
	"strings"
	"time"

	"github.com/channyeintun/gocode/internal/agent"
	"github.com/channyeintun/gocode/internal/api"
	artifactspkg "github.com/channyeintun/gocode/internal/artifacts"
	costpkg "github.com/channyeintun/gocode/internal/cost"
	"github.com/channyeintun/gocode/internal/ipc"
	"github.com/channyeintun/gocode/internal/session"
	skillspkg "github.com/channyeintun/gocode/internal/skills"
	"github.com/channyeintun/gocode/internal/timing"
	toolpkg "github.com/channyeintun/gocode/internal/tools"
)

const exploreSubagentType = "explore"

var exploreSubagentTools = []string{
	"think",
	"list_dir",
	"file_read",
	"glob",
	"grep",
	"go_definition",
	"go_references",
	"project_overview",
	"symbol_search",
	"web_search",
	"web_fetch",
	"git",
}

func makeSubagentRunner(
	bridge *ipc.Bridge,
	registry *toolpkg.Registry,
	parentTracker *costpkg.Tracker,
	sessionStore *session.Store,
	artifactManager *artifactspkg.Manager,
	client api.LLMClient,
	activeModelID string,
	cwd string,
) toolpkg.AgentRunner {
	return func(ctx context.Context, req toolpkg.AgentRunRequest) (toolpkg.AgentRunResult, error) {
		if client == nil {
			return toolpkg.AgentRunResult{}, fmt.Errorf("agent tool is unavailable: model client is not initialized")
		}

		subagentType := strings.TrimSpace(req.SubagentType)
		if subagentType == "" {
			subagentType = exploreSubagentType
		}
		if subagentType != exploreSubagentType {
			return toolpkg.AgentRunResult{}, fmt.Errorf("agent subagent_type %q is not supported yet", subagentType)
		}

		childSessionID, err := newSessionID()
		if err != nil {
			return toolpkg.AgentRunResult{}, err
		}
		childStartedAt := time.Now()
		childTracker := costpkg.NewTracker()
		childMessages := []api.Message{{Role: api.RoleUser, Content: req.Prompt}}
		childRegistry := registry.CloneFiltered(exploreSubagentTools)
		childBridge := ipc.NewBridge(strings.NewReader(""), io.Discard)
		childTimingLogger := timing.NewSessionLogger(sessionStore.SessionDir(childSessionID))
		childSkills, _ := skillspkg.LoadAll(cwd)
		childMode := agent.ModeFast
		childPrompt := subagentSystemPrompt(subagentType, childRegistry.Definitions())

		if err := persistSessionState(sessionStore, sessionStateParams{
			SessionID: childSessionID,
			CreatedAt: childStartedAt,
			Mode:      childMode,
			Model:     activeModelID,
			CWD:       cwd,
			Branch:    agent.LoadTurnContext().GitBranch,
			Tracker:   childTracker,
			Messages:  childMessages,
		}); err != nil {
			return toolpkg.AgentRunResult{}, err
		}
		_ = sessionStore.SaveMetadata(session.Metadata{
			SessionID:    childSessionID,
			CreatedAt:    childStartedAt,
			UpdatedAt:    childStartedAt,
			Mode:         string(childMode),
			Model:        activeModelID,
			CWD:          cwd,
			Branch:       agent.LoadTurnContext().GitBranch,
			TotalCostUSD: 0,
			Title:        req.Description,
		})

		childDeps := agent.QueryDeps{
			CallModel: func(callCtx context.Context, modelReq api.ModelRequest) (iter.Seq2[api.ModelEvent, error], error) {
				return trackModelStream(callCtx, childBridge, childTracker, client, modelReq)
			},
			ExecuteToolBatch: func(callCtx context.Context, calls []api.ToolCall) ([]api.ToolResult, error) {
				return executeToolCallsSilently(callCtx, childRegistry, artifactManager, childSessionID, sessionStore.SessionDir(childSessionID), childTracker, client.Capabilities().MaxOutputTokens, calls)
			},
			CompactMessages: func(callCtx context.Context, current []api.Message, reason agent.CompactReason) ([]api.Message, error) {
				result, err := compactWithMetrics(callCtx, childBridge, childTracker, client, childTimingLogger, childSessionID, 0, string(reason), current)
				if err != nil {
					return nil, err
				}
				return result.Messages, nil
			},
			ApplyResultBudget: func(current []api.Message) []api.Message {
				return current
			},
			PersistMessages: func(updated []api.Message) {
				childMessages = updated
				_ = persistSessionState(sessionStore, sessionStateParams{
					SessionID: childSessionID,
					CreatedAt: childStartedAt,
					Mode:      childMode,
					Model:     activeModelID,
					CWD:       cwd,
					Branch:    agent.LoadTurnContext().GitBranch,
					Tracker:   childTracker,
					Messages:  childMessages,
				})
			},
			Clock: time.Now,
		}

		stream := agent.QueryStream(ctx, agent.QueryRequest{
			Messages:      childMessages,
			SystemPrompt:  childPrompt,
			Mode:          childMode,
			SessionID:     childSessionID,
			Skills:        childSkills,
			Tools:         childRegistry.Definitions(),
			Capabilities:  client.Capabilities(),
			ContextWindow: client.Capabilities().MaxContextWindow,
			MaxTokens:     client.Capabilities().MaxOutputTokens,
		}, childDeps)

		for _, streamErr := range stream {
			if streamErr != nil {
				return toolpkg.AgentRunResult{}, streamErr
			}
		}

		if err := persistSessionState(sessionStore, sessionStateParams{
			SessionID: childSessionID,
			CreatedAt: childStartedAt,
			Mode:      childMode,
			Model:     activeModelID,
			CWD:       cwd,
			Branch:    agent.LoadTurnContext().GitBranch,
			Tracker:   childTracker,
			Messages:  childMessages,
		}); err != nil {
			return toolpkg.AgentRunResult{}, err
		}

		parentTracker.MergeSnapshot(childTracker.Snapshot())
		_ = emitCostUpdate(bridge, parentTracker)

		return toolpkg.AgentRunResult{
			Status:         "completed",
			SubagentType:   subagentType,
			SessionID:      childSessionID,
			TranscriptPath: filepath.Join(sessionStore.SessionDir(childSessionID), "transcript.ndjson"),
			Summary:        latestAssistantContent(childMessages),
			Tools:          toolDefinitionNames(childRegistry.Definitions()),
		}, nil
	}
}

func subagentSystemPrompt(subagentType string, defs []api.ToolDefinition) string {
	names := toolDefinitionNames(defs)
	toolList := strings.Join(names, ", ")
	return strings.TrimSpace(fmt.Sprintf(`You are Go CLI %s, a bounded subagent running in a fresh context.

IMPORTANT: Always use absolute paths with file tools. The working directory is provided in the environment context below.
Use only the tools exposed to you in this session. The exact runtime tool names available are: %s.
This subagent is read-only and artifact-safe: do not modify files, do not create or update session artifacts, and do not attempt background process control.
Work only on the delegated task. Keep the final response concise and report concrete findings with file paths and next steps when useful.`, strings.Title(subagentType), toolList))
}

func toolDefinitionNames(defs []api.ToolDefinition) []string {
	names := make([]string, 0, len(defs))
	for _, def := range defs {
		names = append(names, def.Name)
	}
	return names
}

func latestAssistantContent(messages []api.Message) string {
	for index := len(messages) - 1; index >= 0; index-- {
		msg := messages[index]
		if msg.Role != api.RoleAssistant {
			continue
		}
		if strings.TrimSpace(msg.Content) == "" {
			continue
		}
		return strings.TrimSpace(msg.Content)
	}
	return "Subagent completed without a final text response. See the child transcript for details."
}

func executeToolCallsSilently(
	ctx context.Context,
	registry *toolpkg.Registry,
	artifactManager *artifactspkg.Manager,
	sessionID string,
	sessionDir string,
	tracker *costpkg.Tracker,
	maxOutputTokens int,
	calls []api.ToolCall,
) ([]api.ToolResult, error) {
	results := make([]api.ToolResult, len(calls))
	approved := make([]toolpkg.PendingCall, 0, len(calls))
	budget := toolpkg.DefaultResultBudgetForModel(sessionDir, maxOutputTokens)
	aggregateBudget := toolpkg.NewAggregateResultBudget(budget)

	for index, call := range calls {
		normalized, err := normalizeToolCall(call)
		if err != nil {
			results[index] = api.ToolResult{ToolCallID: call.ID, Output: err.Error(), IsError: true}
			continue
		}
		tool, err := registry.Get(normalized.Name)
		if err != nil {
			results[index] = api.ToolResult{ToolCallID: normalized.ID, Output: err.Error(), IsError: true}
			continue
		}
		input, err := decodeToolInput(normalized)
		if err != nil {
			results[index] = api.ToolResult{ToolCallID: normalized.ID, Output: err.Error(), IsError: true}
			continue
		}
		if err := toolpkg.ValidateToolCall(tool, input); err != nil {
			results[index] = api.ToolResult{ToolCallID: normalized.ID, Output: err.Error(), IsError: true}
			continue
		}
		if tool.Permission() != toolpkg.PermissionReadOnly {
			results[index] = api.ToolResult{ToolCallID: normalized.ID, Output: fmt.Sprintf("tool %q is not allowed in the explore subagent", tool.Name()), IsError: true}
			continue
		}
		approved = append(approved, toolpkg.PendingCall{Index: index, Tool: tool, Input: input})
	}

	for _, batch := range toolpkg.PartitionBatches(approved) {
		batchStart := time.Now()
		batchResults := toolpkg.ExecuteBatch(ctx, batch)
		if tracker != nil {
			tracker.RecordToolDuration(time.Since(batchStart))
		}
		for _, result := range batchResults {
			call := calls[result.Index]
			toolResult := api.ToolResult{ToolCallID: call.ID}
			if result.Err != nil {
				toolResult.Output = result.Err.Error()
				toolResult.IsError = true
				results[result.Index] = toolResult
				continue
			}

			output := result.Output.Output
			if !result.Output.IsError {
				budgetedOutput, _, _, err := budgetToolOutput(ctx, artifactManager, sessionID, budget, aggregateBudget, call, output)
				if err == nil {
					output = budgetedOutput
				}
			}
			toolResult.Output = output
			toolResult.IsError = result.Output.IsError
			results[result.Index] = toolResult
		}
	}

	return results, nil
}
