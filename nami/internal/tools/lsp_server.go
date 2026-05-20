package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type lspServerConfig struct {
	Name       string
	Command    string
	Args       []string
	LanguageID string
	Extensions map[string]struct{}
	Markers    []string
}

var lspServerConfigs = []lspServerConfig{
	{
		Name:       "gopls",
		Command:    "gopls",
		LanguageID: "go",
		Extensions: map[string]struct{}{".go": {}},
		Markers:    []string{"go.work", "go.mod", ".git"},
	},
	{
		Name:       "typescript-language-server",
		Command:    "typescript-language-server",
		Args:       []string{"--stdio"},
		LanguageID: "typescript",
		Extensions: map[string]struct{}{".ts": {}, ".tsx": {}, ".js": {}, ".jsx": {}, ".mjs": {}, ".cjs": {}},
		Markers:    []string{"tsconfig.json", "jsconfig.json", "package.json", ".git"},
	},
}

func resolveLSPServerConfig(request lspRequest) (lspServerConfig, string, error) {
	if request.FilePath != "" {
		extension := strings.ToLower(filepath.Ext(request.FilePath))
		for _, server := range lspServerConfigs {
			if _, ok := server.Extensions[extension]; ok {
				workspaceDir := detectWorkspaceRoot(filepath.Dir(request.FilePath), server.Markers)
				return server, workspaceDir, nil
			}
		}
		return lspServerConfig{}, "", fmt.Errorf("no LSP server configured for files with extension %q", extension)
	}

	searchPath := request.SearchPath
	if searchPath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return lspServerConfig{}, "", fmt.Errorf("get working directory: %w", err)
		}
		searchPath = cwd
	}
	for _, server := range lspServerConfigs {
		workspaceDir := detectWorkspaceRoot(searchPath, server.Markers)
		if containsWorkspaceMarker(workspaceDir, server.Markers) {
			return server, workspaceDir, nil
		}
	}
	return lspServerConfig{}, "", fmt.Errorf("unable to determine an LSP server for workspace_symbols; provide filePath or use a workspace with known markers")
}

func detectWorkspaceRoot(start string, markers []string) string {
	current := start
	for {
		if containsWorkspaceMarker(current, markers) {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			return start
		}
		current = parent
	}
}

func containsWorkspaceMarker(dir string, markers []string) bool {
	for _, marker := range markers {
		if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
			return true
		}
	}
	return false
}

func languageIDForPath(path string, server lspServerConfig) string {
	extension := strings.ToLower(filepath.Ext(path))
	if server.Name == "typescript-language-server" {
		switch extension {
		case ".js", ".jsx", ".mjs", ".cjs":
			return "javascript"
		default:
			return "typescript"
		}
	}
	return server.LanguageID
}
