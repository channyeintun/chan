package agent

import (
	"strings"
	"testing"

	"github.com/channyeintun/nami/internal/api"
)

func TestBuildRequestMessagesMergesPromptInjectionIntoLastUserMessage(t *testing.T) {
	messages := []api.Message{
		{Role: api.RoleAssistant, Content: "Previous reply"},
		{Role: api.RoleUser, Content: "Refactor the loop"},
	}

	built := buildRequestMessages(messages, "focused context")
	if len(built) != len(messages) {
		t.Fatalf("expected %d messages, got %d", len(messages), len(built))
	}
	if built[1].Content == messages[1].Content {
		t.Fatal("expected final user message to be rewritten with prompt injection")
	}
	if built[1].Role != api.RoleUser {
		t.Fatalf("expected final role %q, got %q", api.RoleUser, built[1].Role)
	}
	if !strings.Contains(built[1].Content, "<current_turn_context>") {
		t.Fatalf("expected prompt injection wrapper, got %q", built[1].Content)
	}
	if !strings.Contains(built[1].Content, "<user_request>") {
		t.Fatalf("expected user request wrapper, got %q", built[1].Content)
	}
	if !strings.Contains(built[1].Content, "focused context") {
		t.Fatalf("unexpected merged content: %q", built[1].Content)
	}
	if messages[1].Content != "Refactor the loop" {
		t.Fatalf("expected input slice to remain unchanged, got %q", messages[1].Content)
	}
}

func TestBuildRequestMessagesAppendsPromptInjectionAfterToolMessage(t *testing.T) {
	messages := []api.Message{
		{Role: api.RoleTool, ToolResult: &api.ToolResult{Output: "done"}},
	}

	built := buildRequestMessages(messages, "focused context")
	if len(built) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(built))
	}
	if built[1].Role != api.RoleUser {
		t.Fatalf("expected appended role %q, got %q", api.RoleUser, built[1].Role)
	}
	if !strings.Contains(built[1].Content, "focused context") {
		t.Fatalf("expected appended prompt injection content, got %q", built[1].Content)
	}
}

func TestShouldRetryWithoutToolUseDetectsClarificationLoop(t *testing.T) {
	state := &QueryState{
		Capabilities: api.ModelCapabilities{SupportsToolUse: true},
		Tools: []api.ToolDefinition{
			{Name: "read_file"},
		},
	}
	turn := modelTurn{
		assistantText: "Could you tell me what content it should include?",
		stopReason:    "end_turn",
	}

	if !shouldRetryWithoutToolUse(state, "Refactor the loop implementation", turn) {
		t.Fatal("expected clarification response to trigger a retry without tool use")
	}
}

func TestRecordFailedAttemptsCountsRepeatedSignatures(t *testing.T) {
	log := NewAttemptLog(t.TempDir())
	existing := AttemptEntry{
		Command:        "apply_patch",
		ErrorSignature: "permission denied",
	}
	if err := log.Record(existing); err != nil {
		t.Fatalf("record existing attempt: %v", err)
	}

	calls := []api.ToolCall{
		{ID: "call-1", Name: "apply_patch"},
	}
	results := []api.ToolResult{
		{
			ToolCallID: "call-1",
			Output:     "permission denied\nextra detail",
			IsError:    true,
		},
	}

	repeated, err := recordFailedAttempts(log, calls, results)
	if err != nil {
		t.Fatalf("record failed attempts: %v", err)
	}
	if repeated != 1 {
		t.Fatalf("expected 1 repeated signature, got %d", repeated)
	}

	entries, err := log.Load()
	if err != nil {
		t.Fatalf("load attempt log: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 attempt log entries, got %d", len(entries))
	}
}
