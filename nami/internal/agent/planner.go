package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/channyeintun/nami/internal/api"
	artifactspkg "github.com/channyeintun/nami/internal/artifacts"
	toolpkg "github.com/channyeintun/nami/internal/tools"
)

const (
	planArtifactSlot               = "active"
	planStatusDraft                = "draft"
	planStatusFinal                = "final"
	saveImplementationPlanToolName = "save_implementation_plan"
	saveWalkthroughToolName        = "save_walkthrough"
	planModePromptHint             = "Plan mode: Ultrathink. Not read-only — create/modify if user asks. Non-trivial work: save/update plan via save_implementation_plan as primary review artifact. Choreograph child agents: focused objective, let finish, synthesize. read_project_structure = directory layout. project_overview = semantic summary."
)

// ArtifactUpdate describes an artifact mutation that should be emitted to the UI.
type ArtifactUpdate struct {
	Artifact artifactspkg.Artifact
	Content  string
	Created  bool
}

// Planner coordinates plan artifacts and write-before-plan enforcement.
type Planner struct {
	mode            ExecutionMode
	sessionID       string
	artifactManager *artifactspkg.Manager
}

// PlanReviewRequiredError indicates a write was attempted while a final
// implementation plan is already waiting for user review.
type PlanReviewRequiredError struct {
	ToolName  string
	PlanTitle string
}

// WalkthroughRequiredError indicates the model tried to reopen a finalized
// implementation-plan artifact after execution had already moved past plan review.
type WalkthroughRequiredError struct {
	ToolName  string
	PlanTitle string
}

func (e *PlanReviewRequiredError) Error() string {
	if e == nil {
		return "implementation plan is awaiting user review"
	}
	return fmt.Sprintf("write tool %q blocked in plan mode: implementation plan %q is ready and awaiting user review — do not call write tools until the user approves the plan and the mode switches to fast", e.ToolName, e.PlanTitle)
}

func (e *WalkthroughRequiredError) Error() string {
	if e == nil {
		return "final implementation plan is already complete for review; use save_walkthrough instead"
	}
	return fmt.Sprintf("tool %q blocked: implementation plan %q is already finalized and the session is no longer in plan mode — do not update the implementation-plan artifact again; continue execution and use %s for the completion artifact instead", e.ToolName, e.PlanTitle, saveWalkthroughToolName)
}

// NewPlanner constructs a planner for the current session and mode.
func NewPlanner(mode ExecutionMode, sessionID string, artifactManager *artifactspkg.Manager) *Planner {
	return &Planner{
		mode:            mode,
		sessionID:       strings.TrimSpace(sessionID),
		artifactManager: artifactManager,
	}
}

// BeginTurn creates or refreshes session-scoped planning artifacts for the current turn.
func (p *Planner) BeginTurn(ctx context.Context, userRequest string) ([]ArtifactUpdate, error) {
	return nil, nil
}

// FinalizeTurn persists the plan text produced during the current turn.
func (p *Planner) FinalizeTurn(ctx context.Context, artifactID string, userRequest string, messages []api.Message, fromIndex int) ([]ArtifactUpdate, error) {
	return nil, nil
}

// ValidateTool optionally enforces plan-mode write gating.
func (p *Planner) ValidateTool(ctx context.Context, toolName string, permission toolpkg.PermissionLevel) error {
	if p == nil {
		return nil
	}

	if toolName == saveImplementationPlanToolName && p.mode != ModePlan {
		status, title, err := p.planStatus(ctx)
		if err != nil {
			return fmt.Errorf("tool %q blocked: planner state unavailable: %w", toolName, err)
		}
		if status == planStatusFinal {
			return &WalkthroughRequiredError{ToolName: toolName, PlanTitle: title}
		}
	}

	if permission != toolpkg.PermissionWrite {
		return nil
	}
	if p.mode != ModePlan || !ProfileForMode(p.mode).RequirePlanBeforeWrite {
		return nil
	}

	status, title, err := p.planStatus(ctx)
	if err != nil {
		return fmt.Errorf("write tool %q blocked in plan mode: planner state unavailable: %w", toolName, err)
	}

	if status == planStatusFinal {
		return &PlanReviewRequiredError{ToolName: toolName, PlanTitle: title}
	}

	return fmt.Errorf("write tool %q blocked in plan mode: you must call save_implementation_plan with a complete implementation plan before modifying any files", toolName)
}

// PlanModePromptHint returns the instruction that shapes plan-mode behavior.
func PlanModePromptHint() string {
	return planModePromptHint
}

func (p *Planner) hasSessionArtifacts() bool {
	return p != nil && strings.TrimSpace(p.sessionID) != "" && p.artifactManager != nil
}

func (p *Planner) planStatus(ctx context.Context) (string, string, error) {
	if !p.hasSessionArtifacts() {
		return "", "", nil
	}

	artifact, found, err := p.artifactManager.FindSessionArtifact(ctx, artifactspkg.KindImplementationPlan, artifactspkg.ScopeSession, p.sessionID, planArtifactSlot)
	if err != nil {
		return "", "", err
	}
	if !found {
		return "", "", nil
	}

	if status, ok := artifact.Metadata["status"].(string); ok && strings.TrimSpace(status) != "" {
		return status, artifact.Title, nil
	}

	_, content, err := p.artifactManager.LoadMarkdown(ctx, artifact.ID, 0)
	if err != nil {
		return "", artifact.Title, err
	}
	if strings.Contains(content, "_Planning in progress._") {
		return planStatusDraft, artifact.Title, nil
	}
	return planStatusFinal, artifact.Title, nil
}

var implementationIntentTerms = []string{
	"implement",
	"implementation",
	"fix",
	"add",
	"change",
	"update",
	"refactor",
	"build",
	"create",
	"rename",
	"support",
	"wire",
	"patch",
	"edit",
	"modify",
	"migrate",
	"remove",
	"replace",
}

var questionPrefixes = []string{
	"what",
	"why",
	"how",
	"when",
	"where",
	"which",
	"who",
	"explain",
	"review",
	"analyze",
	"tell me",
	"can you explain",
	"could you explain",
}

func containsAny(text string, terms []string) bool {
	for _, term := range terms {
		if strings.Contains(text, term) {
			return true
		}
	}
	return false
}

func looksLikeQuestion(text string) bool {
	text = normalizeIntentText(text)
	if strings.Contains(text, "?") {
		return true
	}
	for _, prefix := range questionPrefixes {
		if strings.HasPrefix(text, prefix+" ") || text == prefix {
			return true
		}
	}
	return false
}

func normalizeIntentText(text string) string {
	text = strings.ToLower(strings.TrimSpace(text))
	for _, prefix := range []string{"please ", "kindly ", "pls "} {
		text = strings.TrimPrefix(text, prefix)
	}
	return text
}