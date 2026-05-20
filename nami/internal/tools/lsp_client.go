package tools

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

type lspClient struct {
	cmd          *exec.Cmd
	stdin        io.WriteCloser
	stdout       *bufio.Reader
	writeMu      sync.Mutex
	nextID       int64
	workspaceDir string
	server       lspServerConfig
	stderr       bytes.Buffer
}

func newLSPClient(ctx context.Context, request lspRequest) (*lspClient, error) {
	server, workspaceDir, err := resolveLSPServerConfig(request)
	if err != nil {
		return nil, err
	}
	if _, err := exec.LookPath(server.Command); err != nil {
		return nil, fmt.Errorf("lsp server %q is not installed or not on PATH", server.Command)
	}

	cmd := exec.CommandContext(ctx, server.Command, server.Args...)
	cmd.Dir = workspaceDir
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("open stdin for %s: %w", server.Name, err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("open stdout for %s: %w", server.Name, err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("open stderr for %s: %w", server.Name, err)
	}

	client := &lspClient{
		cmd:          cmd,
		stdin:        stdin,
		stdout:       bufio.NewReader(stdout),
		workspaceDir: workspaceDir,
		server:       server,
	}
	go func() {
		_, _ = io.Copy(&client.stderr, stderr)
	}()
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start %s: %w", server.Name, err)
	}
	if err := client.initialize(ctx); err != nil {
		_ = client.Close()
		return nil, err
	}
	return client, nil
}

func (c *lspClient) Run(ctx context.Context, request lspRequest) ([]lspResultRow, error) {
	if request.FilePath != "" {
		if err := c.didOpen(request.FilePath); err != nil {
			return nil, err
		}
	}

	switch request.Operation {
	case "go_to_definition":
		var rawResult any
		if err := c.request(ctx, "textDocument/definition", c.textDocumentPositionParams(request), &rawResult); err != nil {
			return nil, err
		}
		return limitLSPResults(locationRowsFromAny(rawResult, "definition"), request.MaxResults), nil
	case "find_references":
		var locations []lspLocation
		params := c.textDocumentPositionParams(request)
		params["context"] = map[string]any{"includeDeclaration": request.IncludeDeclaration}
		if err := c.request(ctx, "textDocument/references", params, &locations); err != nil {
			return nil, err
		}
		return limitLSPResults(rowsFromLocations(locations, "reference"), request.MaxResults), nil
	case "hover":
		var hover lspHover
		if err := c.request(ctx, "textDocument/hover", c.textDocumentPositionParams(request), &hover); err != nil {
			return nil, err
		}
		rows := make([]lspResultRow, 0, 1)
		contents := strings.TrimSpace(extractHoverContents(hover.Contents))
		if contents != "" {
			row := lspResultRow{Kind: "hover", Contents: contents}
			if hover.Range != nil {
				applyRange(&row, hover.Range)
			}
			rows = append(rows, row)
		}
		return rows, nil
	case "document_symbols":
		var rawResult json.RawMessage
		if err := c.request(ctx, "textDocument/documentSymbol", map[string]any{
			"textDocument": map[string]any{"uri": pathToFileURI(request.FilePath)},
		}, &rawResult); err != nil {
			return nil, err
		}
		return limitLSPResults(documentSymbolRows(rawResult, request.FilePath), request.MaxResults), nil
	case "workspace_symbols":
		var symbols []lspSymbolInformation
		if err := c.request(ctx, "workspace/symbol", map[string]any{"query": request.Query}, &symbols); err != nil {
			return nil, err
		}
		return limitLSPResults(rowsFromWorkspaceSymbols(symbols), request.MaxResults), nil
	case "go_to_implementation":
		var rawResult any
		if err := c.request(ctx, "textDocument/implementation", c.textDocumentPositionParams(request), &rawResult); err != nil {
			return nil, err
		}
		return limitLSPResults(locationRowsFromAny(rawResult, "implementation"), request.MaxResults), nil
	default:
		return nil, fmt.Errorf("unsupported lsp operation %q", request.Operation)
	}
}

func (c *lspClient) initialize(ctx context.Context) error {
	var result map[string]any
	params := map[string]any{
		"processId": os.Getpid(),
		"rootUri":   pathToFileURI(c.workspaceDir),
		"rootPath":  c.workspaceDir,
		"clientInfo": map[string]any{
			"name": "nami",
		},
		"workspaceFolders": []lspWorkspaceFolder{{
			URI:  pathToFileURI(c.workspaceDir),
			Name: filepath.Base(c.workspaceDir),
		}},
		"capabilities": map[string]any{
			"textDocument": map[string]any{
				"hover":          map[string]any{"contentFormat": []string{"markdown", "plaintext"}},
				"definition":     map[string]any{"linkSupport": true},
				"implementation": map[string]any{"linkSupport": true},
				"references":     map[string]any{},
				"documentSymbol": map[string]any{"hierarchicalDocumentSymbolSupport": true},
			},
			"workspace": map[string]any{
				"workspaceFolders": true,
			},
		},
	}
	if err := c.request(ctx, "initialize", params, &result); err != nil {
		return fmt.Errorf("initialize %s: %w", c.server.Name, err)
	}
	if err := c.notify("initialized", map[string]any{}); err != nil {
		return fmt.Errorf("notify initialized: %w", err)
	}
	return nil
}

func (c *lspClient) didOpen(filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read file for didOpen %q: %w", filePath, err)
	}
	params := map[string]any{
		"textDocument": map[string]any{
			"uri":        pathToFileURI(filePath),
			"languageId": languageIDForPath(filePath, c.server),
			"version":    1,
			"text":       string(content),
		},
	}
	if err := c.notify("textDocument/didOpen", params); err != nil {
		return fmt.Errorf("notify didOpen: %w", err)
	}
	return nil
}

func (c *lspClient) textDocumentPositionParams(request lspRequest) map[string]any {
	return map[string]any{
		"textDocument": map[string]any{"uri": pathToFileURI(request.FilePath)},
		"position": map[string]any{
			"line":      request.Line - 1,
			"character": request.Column - 1,
		},
	}
}
