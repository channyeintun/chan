package engine

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/channyeintun/nami/internal/session"
	"github.com/channyeintun/nami/internal/swarm"
	toolpkg "github.com/channyeintun/nami/internal/tools"
)

type subagentRolePolicy struct {
	role             swarm.ResolvedRole
	allowedToolNames map[string]struct{}
}

func loadSubagentRolePolicy(cwd string, roleName string, registry *toolpkg.Registry, subagentType string) (*subagentRolePolicy, []string, error) {
	resolvedRole, ok, err := resolveSubagentRole(cwd, roleName)
	if err != nil {
		return nil, nil, err
	}
	if !ok {
		return nil, append([]string(nil), subagentToolNames(subagentType)...), nil
	}
	if problems := unknownRoleToolNames(registry, resolvedRole); len(problems) > 0 {
		return nil, nil, fmt.Errorf("swarm role %q references unknown tools: %s", resolvedRole.Name, strings.Join(problems, ", "))
	}
	allowedToolNames := filterRoleToolNames(registry, subagentType, resolvedRole)
	return &subagentRolePolicy{
		role:             resolvedRole,
		allowedToolNames: stringSet(allowedToolNames),
	}, allowedToolNames, nil
}

func resolveSubagentRole(cwd string, roleName string) (swarm.ResolvedRole, bool, error) {
	trimmedRole := strings.TrimSpace(roleName)
	if trimmedRole == "" {
		return swarm.ResolvedRole{}, false, nil
	}
	spec, err := swarm.LoadProjectSpec(cwd)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return swarm.ResolvedRole{}, false, nil
		}
		return swarm.ResolvedRole{}, false, err
	}
	resolvedRole, ok := spec.Role(trimmedRole)
	if !ok {
		return swarm.ResolvedRole{}, false, fmt.Errorf("swarm role %q is not defined in %s", trimmedRole, spec.Path)
	}
	return resolvedRole, true, nil
}

func filterRoleToolNames(registry *toolpkg.Registry, subagentType string, role swarm.ResolvedRole) []string {
	allowSet := stringSet(role.AllowTools)
	denySet := stringSet(role.DenyTools)
	filtered := make([]string, 0, len(subagentToolNames(subagentType)))
	for _, name := range subagentToolNames(subagentType) {
		if len(allowSet) > 0 {
			if _, ok := allowSet[name]; !ok {
				continue
			}
		}
		if _, ok := denySet[name]; ok {
			continue
		}
		tool, err := registry.Get(name)
		if err != nil {
			continue
		}
		if !permissionProfileAllows(role.PermissionProfile, tool.Permission()) {
			continue
		}
		filtered = append(filtered, name)
	}
	return filtered
}

func unknownRoleToolNames(registry *toolpkg.Registry, role swarm.ResolvedRole) []string {
	combined := append(append([]string(nil), role.AllowTools...), role.DenyTools...)
	unknown := make([]string, 0)
	seen := make(map[string]struct{}, len(combined))
	for _, name := range combined {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		if _, err := registry.Get(trimmed); err != nil {
			unknown = append(unknown, trimmed)
		}
	}
	return unknown
}

func permissionProfileAllows(profile swarm.PermissionProfile, permission toolpkg.PermissionLevel) bool {
	switch profile {
	case swarm.PermissionProfileReadOnly:
		return permission == toolpkg.PermissionReadOnly
	case swarm.PermissionProfileExecute:
		return permission == toolpkg.PermissionReadOnly || permission == toolpkg.PermissionExecute
	default:
		return true
	}
}

func (p *subagentRolePolicy) allowsToolName(name string) bool {
	if p == nil {
		return true
	}
	_, ok := p.allowedToolNames[strings.TrimSpace(name)]
	return ok
}

func (p *subagentRolePolicy) toolDeniedMessage(name string) string {
	trimmed := strings.TrimSpace(name)
	if p == nil {
		return fmt.Sprintf("tool %q is not allowed for this child agent", trimmed)
	}
	if len(p.role.AllowTools) > 0 {
		return fmt.Sprintf("tool %q is not allowed for swarm role %q. Allowed tools: %s. Hand this work off or escalate it instead.", trimmed, p.role.Name, strings.Join(p.role.AllowTools, ", "))
	}
	if len(p.role.DenyTools) > 0 {
		return fmt.Sprintf("tool %q is denied for swarm role %q. Denied tools: %s. Hand this work off or escalate it instead.", trimmed, p.role.Name, strings.Join(p.role.DenyTools, ", "))
	}
	return fmt.Sprintf("tool %q is blocked for swarm role %q by the %s permission profile. Hand this work off or escalate it instead.", trimmed, p.role.Name, p.role.PermissionProfile)
}

func (p *subagentRolePolicy) completionBlocked(store *session.Store, sessionID string, startedAt time.Time, stopReason string) (bool, string, string, error) {
	if p == nil || !p.role.Handoff.Required {
		return false, "", "", nil
	}
	switch strings.TrimSpace(stopReason) {
	case "cancelled", "cancelling":
		return false, "", "", nil
	}
	trimmedSessionID := strings.TrimSpace(sessionID)
	if trimmedSessionID == "" || store == nil {
		return false, "", "", fmt.Errorf("swarm runtime session is unavailable for role %q completion policy", p.role.Name)
	}
	handoffs, err := swarm.ListHandoffs(store, trimmedSessionID, p.role.Name, nil)
	if err != nil {
		return false, "", "", err
	}
	for _, handoff := range handoffs {
		if handoff.SourceRole != p.role.Name {
			continue
		}
		if handoff.CreatedAt.Before(startedAt) && handoff.UpdatedAt.Before(startedAt) {
			continue
		}
		return false, "", "", nil
	}
	reason := fmt.Sprintf("role %q must submit a structured swarm handoff before finishing", p.role.Name)
	return true, reason, p.handoffFollowUpMessage(), nil
}

func (p *subagentRolePolicy) handoffFollowUpMessage() string {
	if p == nil {
		return "Submit a structured swarm handoff before finishing."
	}
	parts := []string{fmt.Sprintf("Before finishing, submit a structured swarm handoff for role %q.", p.role.Name)}
	parts = append(parts, fmt.Sprintf("Use swarm_submit_handoff with source_role %q.", p.role.Name))
	if len(p.role.Handoff.Targets) > 0 {
		parts = append(parts, fmt.Sprintf("Allowed target roles: %s.", strings.Join(p.role.Handoff.Targets, ", ")))
	}
	if len(p.role.Handoff.RequiredFields) > 0 {
		fields := make([]string, 0, len(p.role.Handoff.RequiredFields))
		for _, field := range p.role.Handoff.RequiredFields {
			fields = append(fields, string(field))
		}
		parts = append(parts, fmt.Sprintf("Required handoff fields: %s.", strings.Join(fields, ", ")))
	}
	parts = append(parts, "Continue working until the handoff is queued.")
	return strings.Join(parts, " ")
}

func stringSet(values []string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		set[trimmed] = struct{}{}
	}
	return set
}
