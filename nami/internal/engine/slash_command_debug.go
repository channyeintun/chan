package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/channyeintun/nami/internal/debuglog"
)

func handleDebugSlashCommand(cmd *slashCommandContext) error {
	parts := strings.Fields(strings.TrimSpace(cmd.args))
	subcommand := ""
	if len(parts) > 0 {
		subcommand = strings.ToLower(parts[0])
	}

	switch subcommand {
	case "", "on":
		if err := rebindDebugSession(cmd); err != nil {
			return emitTextResponse(cmd.bridge, fmt.Sprintf("Enable debug logging failed: %v", err))
		}
		path, err := debuglog.Enable()
		if err != nil {
			return emitTextResponse(cmd.bridge, fmt.Sprintf("Enable debug logging failed: %v", err))
		}
		if err := debuglog.OpenMonitorPopup(path); err != nil {
			return emitTextResponse(cmd.bridge, fmt.Sprintf("Debug logging enabled at %s\n\nAutomatic monitor launch failed: %v\nRun manually: %s debug-view --file %s", path, err, os.Args[0], path))
		}
		return emitTextResponse(cmd.bridge, fmt.Sprintf("Debug logging enabled. Opened live monitor for %s", path))
	case "status":
		if err := rebindDebugSession(cmd); err != nil && debuglog.IsEnabled() {
			return emitTextResponse(cmd.bridge, fmt.Sprintf("Debug status unavailable: %v", err))
		}
		status := debuglog.CurrentStatus()
		path := status.Path
		if strings.TrimSpace(path) == "" {
			path = filepath.Join(cmd.store.SessionDir(cmd.state.SessionID), debuglog.DefaultPath)
		}
		state := "disabled"
		if status.Enabled {
			state = "enabled"
		}
		return emitTextResponse(cmd.bridge, fmt.Sprintf("Debug logging is %s\nSession: %s\nPath: %s\nEntries: %d", state, cmd.state.SessionID, path, status.Seq))
	case "path":
		if err := rebindDebugSession(cmd); err != nil && debuglog.IsEnabled() {
			return emitTextResponse(cmd.bridge, fmt.Sprintf("Debug path unavailable: %v", err))
		}
		path := debuglog.CurrentPath()
		if strings.TrimSpace(path) == "" {
			path = filepath.Join(cmd.store.SessionDir(cmd.state.SessionID), debuglog.DefaultPath)
		}
		return emitTextResponse(cmd.bridge, path)
	case "off":
		if err := debuglog.Disable(); err != nil {
			return emitTextResponse(cmd.bridge, fmt.Sprintf("Disable debug logging failed: %v", err))
		}
		return emitTextResponse(cmd.bridge, "Debug logging disabled.")
	default:
		return emitTextResponse(cmd.bridge, "usage: /debug [status|path|off]")
	}
}

func rebindDebugSession(cmd *slashCommandContext) error {
	return debuglog.ConfigureSession(cmd.state.SessionID, cmd.store.SessionDir(cmd.state.SessionID))
}
