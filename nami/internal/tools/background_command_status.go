package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

const backgroundCommandSummaryPreviewBytes = 160
const backgroundCommandNotificationPreviewBytes = 4096

func forgetBackgroundCommand(commandID string) (BackgroundCommandResult, error) {
	backgroundCommandsMu.Lock()
	bg, ok := backgroundCommands[commandID]
	if !ok {
		backgroundCommandsMu.Unlock()
		return BackgroundCommandResult{}, fmt.Errorf("command %q not found", commandID)
	}

	bg.consumeMu.Lock()
	defer bg.consumeMu.Unlock()

	bg.mu.Lock()
	if bg.running {
		bg.mu.Unlock()
		backgroundCommandsMu.Unlock()
		return BackgroundCommandResult{}, fmt.Errorf("command %q is still running; stop it before forgetting it", bg.id)
	}

	result := BackgroundCommandResult{
		CommandID: bg.id,
		Command:   bg.command,
		Cwd:       bg.cwd,
		Running:   false,
		StartedAt: bg.startedAt,
		UpdatedAt: bg.updatedAt,
		Error:     bg.errText,
	}
	if bg.exitCode != nil {
		copied := *bg.exitCode
		result.ExitCode = &copied
	}
	bg.mu.Unlock()

	delete(backgroundCommands, commandID)
	backgroundCommandsMu.Unlock()

	result.Output = bg.output.ReadDelta()
	return result, nil
}

func (bg *backgroundCommand) sendInput(input string, wait time.Duration) (BackgroundCommandResult, error) {
	bg.consumeMu.Lock()
	defer bg.consumeMu.Unlock()

	bg.mu.Lock()
	if !bg.running {
		bg.mu.Unlock()
		return BackgroundCommandResult{}, fmt.Errorf("command %q is not running", bg.id)
	}
	_, err := io.WriteString(bg.stdin, input)
	if err == nil {
		bg.updatedAt = time.Now()
	}
	bg.mu.Unlock()
	if err != nil {
		return BackgroundCommandResult{}, fmt.Errorf("write command input: %w", err)
	}

	if wait > 0 {
		timer := time.NewTimer(wait)
		defer timer.Stop()
		select {
		case <-bg.done:
		case <-timer.C:
		}
	}

	return bg.snapshotDelta(), nil
}

func (bg *backgroundCommand) status(wait time.Duration) BackgroundCommandResult {
	bg.consumeMu.Lock()
	defer bg.consumeMu.Unlock()

	if wait > 0 {
		timer := time.NewTimer(wait)
		defer timer.Stop()
		select {
		case <-bg.done:
		case <-timer.C:
		}
	}
	return bg.snapshotDelta()
}

func (bg *backgroundCommand) stop(wait time.Duration) BackgroundCommandResult {
	bg.consumeMu.Lock()
	defer bg.consumeMu.Unlock()

	bg.shutdown()
	if wait > 0 {
		timer := time.NewTimer(wait)
		defer timer.Stop()
		select {
		case <-bg.done:
		case <-timer.C:
		}
	}
	return bg.snapshotDelta()
}

func (bg *backgroundCommand) snapshotDelta() BackgroundCommandResult {
	bg.mu.Lock()
	running := bg.running
	errText := bg.errText
	var exitCode *int
	if bg.exitCode != nil {
		copied := *bg.exitCode
		exitCode = &copied
	}
	bg.mu.Unlock()

	return BackgroundCommandResult{
		CommandID: bg.id,
		Command:   bg.command,
		Cwd:       bg.cwd,
		Running:   running,
		StartedAt: bg.startedAt,
		UpdatedAt: bg.updatedAt,
		Output:    bg.output.ReadDelta(),
		Error:     errText,
		ExitCode:  exitCode,
	}
}

func (bg *backgroundCommand) detail(limit int) BackgroundCommandDetail {
	bg.mu.Lock()
	running := bg.running
	errText := bg.errText
	commandID := bg.id
	command := bg.command
	cwd := bg.cwd
	startedAt := bg.startedAt
	updatedAt := bg.updatedAt
	var exitCode *int
	if bg.exitCode != nil {
		copied := *bg.exitCode
		exitCode = &copied
	}
	bg.mu.Unlock()

	unread := bg.output.unreadSummary(limit)

	return BackgroundCommandDetail{
		CommandID:       commandID,
		Command:         command,
		Cwd:             cwd,
		Status:          backgroundCommandAsyncStatus(running, exitCode, errText),
		Running:         running,
		StartedAt:       startedAt,
		UpdatedAt:       updatedAt,
		Output:          bg.output.tail(limit),
		HasUnreadOutput: unread.HasUnread,
		UnreadBytes:     unread.UnreadBytes,
		ExitCode:        exitCode,
		Error:           errText,
	}
}

func (bg *backgroundCommand) summary() backgroundCommandSummary {
	bg.mu.Lock()
	startedAt := bg.startedAt
	updatedAt := bg.updatedAt
	defer bg.mu.Unlock()

	var exitCode *int
	if bg.exitCode != nil {
		copied := *bg.exitCode
		exitCode = &copied
	}
	unread := bg.output.unreadSummary(backgroundCommandSummaryPreviewBytes)

	return backgroundCommandSummary{
		CommandID:       bg.id,
		Command:         bg.command,
		Cwd:             bg.cwd,
		Running:         bg.running,
		Error:           bg.errText,
		ExitCode:        exitCode,
		StartedAt:       startedAt,
		UpdatedAt:       updatedAt,
		HasUnreadOutput: unread.HasUnread,
		UnreadBytes:     unread.UnreadBytes,
		UnreadPreview:   unread.Preview,
	}
}

func (bg *backgroundCommand) markUpdated(at time.Time) {
	bg.mu.Lock()
	defer bg.mu.Unlock()
	bg.updatedAt = at
}

func (bg *backgroundCommand) asyncUpdate() BackgroundCommandUpdate {
	bg.mu.Lock()
	running := bg.running
	errText := bg.errText
	startedAt := bg.startedAt
	updatedAt := bg.updatedAt
	command := bg.command
	cwd := bg.cwd
	commandID := bg.id
	var exitCode *int
	if bg.exitCode != nil {
		copied := *bg.exitCode
		exitCode = &copied
	}
	bg.mu.Unlock()

	unread := bg.output.unreadSummary(backgroundCommandNotificationPreviewBytes)

	return BackgroundCommandUpdate{
		CommandID:       commandID,
		Command:         command,
		Cwd:             cwd,
		Status:          backgroundCommandAsyncStatus(running, exitCode, errText),
		Running:         running,
		StartedAt:       startedAt,
		UpdatedAt:       updatedAt,
		OutputPreview:   unread.Preview,
		HasUnreadOutput: unread.HasUnread,
		UnreadBytes:     unread.UnreadBytes,
		ExitCode:        exitCode,
		Error:           errText,
	}
}

func backgroundCommandAsyncStatus(running bool, exitCode *int, errText string) string {
	if running {
		return "running"
	}
	if exitCode != nil && *exitCode != 0 {
		return "failed"
	}
	if strings.TrimSpace(errText) != "" {
		return "failed"
	}
	return "completed"
}

func renderBackgroundCommandResult(result BackgroundCommandResult) (string, error) {
	encoded, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func InspectBackgroundCommand(ctx context.Context, commandID string, wait time.Duration, tailBytes int) (BackgroundCommandDetail, error) {
	bg, err := getBackgroundCommand(commandID)
	if err != nil {
		return BackgroundCommandDetail{}, err
	}

	if wait > 0 {
		timer := time.NewTimer(wait)
		defer timer.Stop()
		select {
		case <-bg.done:
		case <-timer.C:
		case <-ctx.Done():
			return BackgroundCommandDetail{}, ctx.Err()
		}
	}

	return bg.detail(tailBytes), nil
}

func StopBackgroundCommand(commandID string, wait time.Duration) (BackgroundCommandResult, error) {
	bg, err := getBackgroundCommand(commandID)
	if err != nil {
		return BackgroundCommandResult{}, err
	}
	return bg.stop(wait), nil
}

func BackgroundCommandUpdateSnapshot(commandID string) (BackgroundCommandUpdate, error) {
	bg, err := getBackgroundCommand(commandID)
	if err != nil {
		return BackgroundCommandUpdate{}, err
	}
	return bg.asyncUpdate(), nil
}
