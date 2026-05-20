package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

const defaultLSPMaxResults = 100

type LSPTool struct{}

type lspRequest struct {
	Operation          string
	FilePath           string
	Line               int
	Column             int
	Query              string
	IncludeDeclaration bool
	MaxResults         int
	SearchPath         string
}

type lspResponseEnvelope struct {
	JSONRPC string          `json:"jsonrpc,omitempty"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *lspRPCError    `json:"error,omitempty"`
}

type lspRPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

type lspPosition struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type lspRange struct {
	Start lspPosition `json:"start"`
	End   lspPosition `json:"end"`
}

type lspLocation struct {
	URI   string   `json:"uri"`
	Range lspRange `json:"range"`
}

type lspLocationLink struct {
	TargetURI            string   `json:"targetUri"`
	TargetRange          lspRange `json:"targetRange"`
	TargetSelectionRange lspRange `json:"targetSelectionRange"`
}

type lspMarkupContent struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

type lspMarkedString struct {
	Language string `json:"language"`
	Value    string `json:"value"`
}

type lspHover struct {
	Contents any       `json:"contents"`
	Range    *lspRange `json:"range,omitempty"`
}

type lspDocumentSymbol struct {
	Name           string              `json:"name"`
	Detail         string              `json:"detail,omitempty"`
	Kind           int                 `json:"kind"`
	Range          lspRange            `json:"range"`
	SelectionRange lspRange            `json:"selectionRange"`
	Children       []lspDocumentSymbol `json:"children,omitempty"`
}

type lspSymbolInformation struct {
	Name          string      `json:"name"`
	Kind          int         `json:"kind"`
	Location      lspLocation `json:"location"`
	ContainerName string      `json:"containerName,omitempty"`
}

type lspWorkspaceFolder struct {
	URI  string `json:"uri"`
	Name string `json:"name"`
}

type lspOutput struct {
	Operation   string         `json:"operation"`
	FilePath    string         `json:"filePath,omitempty"`
	Workspace   string         `json:"workspace,omitempty"`
	ResultCount int            `json:"resultCount"`
	Results     []lspResultRow `json:"results"`
}

type lspResultRow struct {
	Kind          string `json:"kind"`
	Name          string `json:"name,omitempty"`
	Detail        string `json:"detail,omitempty"`
	SymbolKind    string `json:"symbolKind,omitempty"`
	FilePath      string `json:"filePath,omitempty"`
	Line          int    `json:"line,omitempty"`
	Column        int    `json:"column,omitempty"`
	EndLine       int    `json:"endLine,omitempty"`
	EndColumn     int    `json:"endColumn,omitempty"`
	ContainerName string `json:"containerName,omitempty"`
	Contents      string `json:"contents,omitempty"`
	Signature     string `json:"signature,omitempty"`
}

func NewLSPTool() *LSPTool {
	return &LSPTool{}
}

func (t *LSPTool) Name() string {
	return "lsp"
}

func (t *LSPTool) Description() string {
	return "Use a local Language Server Protocol server for semantic code intelligence including definitions, references, hover, document symbols, workspace symbols, and implementations."
}

func (t *LSPTool) InputSchema() any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"operation": map[string]any{
				"type": "string",
				"enum": []string{
					"go_to_definition",
					"find_references",
					"hover",
					"document_symbols",
					"workspace_symbols",
					"go_to_implementation",
				},
				"description": "The LSP operation to perform.",
			},
			"filePath": map[string]any{
				"type":        "string",
				"description": "Absolute or workspace-relative path for file-based operations.",
			},
			"path": map[string]any{
				"type":        "string",
				"description": "Optional workspace path hint for workspace_symbols.",
			},
			"line": map[string]any{
				"type":        "integer",
				"minimum":     1,
				"description": "1-based line number for position-based operations.",
			},
			"column": map[string]any{
				"type":        "integer",
				"minimum":     1,
				"description": "1-based column number for position-based operations.",
			},
			"character": map[string]any{
				"type":        "integer",
				"minimum":     1,
				"description": "Compatibility alias for column.",
			},
			"query": map[string]any{
				"type":        "string",
				"description": "Workspace symbol query for workspace_symbols.",
			},
			"includeDeclaration": map[string]any{
				"type":        "boolean",
				"description": "Whether declaration sites are included in find_references.",
			},
			"include_declaration": map[string]any{
				"type":        "boolean",
				"description": "Snake_case alias for includeDeclaration.",
			},
			"maxResults": map[string]any{
				"type":        "integer",
				"minimum":     1,
				"description": "Optional maximum number of results to return.",
			},
			"max_results": map[string]any{
				"type":        "integer",
				"minimum":     1,
				"description": "Snake_case alias for maxResults.",
			},
		},
		"required": []string{"operation"},
	}
}

func (t *LSPTool) Permission() PermissionLevel {
	return PermissionReadOnly
}

func (t *LSPTool) Concurrency(input ToolInput) ConcurrencyDecision {
	return ConcurrencyParallel
}

func (t *LSPTool) Validate(input ToolInput) error {
	_, err := parseLSPRequest(input.Params)
	return err
}

func (t *LSPTool) Execute(ctx context.Context, input ToolInput) (ToolOutput, error) {
	request, err := parseLSPRequest(input.Params)
	if err != nil {
		return ToolOutput{}, err
	}
	client, err := newLSPClient(ctx, request)
	if err != nil {
		return ToolOutput{}, err
	}
	defer client.Close()

	rows, err := client.Run(ctx, request)
	if err != nil {
		return ToolOutput{}, err
	}

	output := lspOutput{
		Operation:   request.Operation,
		FilePath:    request.FilePath,
		Workspace:   client.workspaceDir,
		ResultCount: len(rows),
		Results:     rows,
	}
	encoded, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return ToolOutput{}, fmt.Errorf("marshal lsp output: %w", err)
	}
	return ToolOutput{Output: string(encoded)}, nil
}

func parseLSPRequest(params map[string]any) (lspRequest, error) {
	operation, ok := firstStringParam(params, "operation")
	if !ok || strings.TrimSpace(operation) == "" {
		return lspRequest{}, fmt.Errorf("lsp requires operation")
	}
	request := lspRequest{
		Operation:          normalizeLSPOperation(operation),
		MaxResults:         firstPositiveIntOrDefault(params, defaultLSPMaxResults, "maxResults", "max_results"),
		IncludeDeclaration: firstBoolParam(params, "includeDeclaration", "include_declaration"),
	}
	if filePath, ok := firstStringParam(params, "filePath"); ok && strings.TrimSpace(filePath) != "" {
		request.FilePath = strings.TrimSpace(filePath)
	}
	if pathHint, ok := stringParam(params, "path"); ok && strings.TrimSpace(pathHint) != "" {
		trimmedPath := strings.TrimSpace(pathHint)
		if request.Operation == "workspace_symbols" {
			request.SearchPath = trimmedPath
		} else if request.FilePath == "" {
			request.FilePath = trimmedPath
		}
	}
	if query, ok := stringParam(params, "query"); ok {
		request.Query = strings.TrimSpace(query)
	}
	if line, ok := firstIntParam(params, "line"); ok {
		request.Line = line
	}
	if column, ok := firstIntParam(params, "column", "character"); ok {
		request.Column = column
	}

	switch request.Operation {
	case "go_to_definition", "find_references", "hover", "go_to_implementation":
		if request.FilePath == "" {
			return lspRequest{}, fmt.Errorf("lsp %s requires filePath", request.Operation)
		}
		if request.Line < 1 || request.Column < 1 {
			return lspRequest{}, fmt.Errorf("lsp %s requires positive line and column", request.Operation)
		}
	case "document_symbols":
		if request.FilePath == "" {
			return lspRequest{}, fmt.Errorf("lsp document_symbols requires filePath")
		}
	case "workspace_symbols":
		if request.Query == "" {
			return lspRequest{}, fmt.Errorf("lsp workspace_symbols requires query")
		}
	default:
		return lspRequest{}, fmt.Errorf("unsupported lsp operation %q", operation)
	}

	if request.FilePath != "" {
		resolvedPath, err := resolveToolPath(request.FilePath)
		if err != nil {
			return lspRequest{}, err
		}
		info, err := os.Stat(resolvedPath)
		if err != nil {
			return lspRequest{}, fmt.Errorf("stat file %q: %w", resolvedPath, err)
		}
		if info.IsDir() {
			return lspRequest{}, fmt.Errorf("%q is a directory", resolvedPath)
		}
		request.FilePath = resolvedPath
	}
	if request.SearchPath != "" {
		resolvedSearchPath, err := resolveToolPath(request.SearchPath)
		if err != nil {
			return lspRequest{}, err
		}
		request.SearchPath = resolvedSearchPath
	}
	return request, nil
}

func normalizeLSPOperation(value string) string {
	switch strings.TrimSpace(value) {
	case "go_to_definition", "goToDefinition":
		return "go_to_definition"
	case "find_references", "findReferences":
		return "find_references"
	case "hover":
		return "hover"
	case "document_symbols", "documentSymbol":
		return "document_symbols"
	case "workspace_symbols", "workspaceSymbol":
		return "workspace_symbols"
	case "go_to_implementation", "goToImplementation":
		return "go_to_implementation"
	default:
		return strings.TrimSpace(value)
	}
}

func firstPositiveIntOrDefault(params map[string]any, fallback int, keys ...string) int {
	for _, key := range keys {
		if value, ok := intParam(params, key); ok && value > 0 {
			return value
		}
	}
	return fallback
}
