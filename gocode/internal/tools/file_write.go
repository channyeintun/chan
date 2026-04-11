package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileWriteTool creates or overwrites a file on disk.
type FileWriteTool struct{}

// NewFileWriteTool constructs the file write tool.
func NewFileWriteTool() *FileWriteTool {
	return &FileWriteTool{}
}

func (t *FileWriteTool) Name() string {
	return "file_write"
}

func (t *FileWriteTool) Description() string {
	return "Create a text file, or overwrite an existing text file only when overwrite is true."
}

func (t *FileWriteTool) InputSchema() any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"file_path": map[string]any{
				"type":        "string",
				"description": "The absolute path to the file to write.",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "The full text content to write to the file.",
			},
			"overwrite": map[string]any{
				"type":        "boolean",
				"description": "When true, allow overwriting an existing file. Defaults to false.",
			},
		},
		"required": []string{"file_path", "content"},
	}
}

func (t *FileWriteTool) Validate(input ToolInput) error {
	filePath, ok := stringParam(input.Params, "file_path")
	if !ok || strings.TrimSpace(filePath) == "" {
		return fmt.Errorf("file_write requires file_path")
	}
	resolvedPath, err := resolveToolPath(filePath)
	if err != nil {
		return err
	}
	info, err := os.Stat(resolvedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat file %q: %w", resolvedPath, err)
	}
	if info.IsDir() {
		return fmt.Errorf("%q is a directory", resolvedPath)
	}
	if !boolParam(input.Params, "overwrite") {
		return fmt.Errorf("file already exists: %s (set overwrite=true to replace it, or use file_edit for in-place changes)", resolvedPath)
	}
	return nil
}

func (t *FileWriteTool) Permission() PermissionLevel {
	return PermissionWrite
}

func (t *FileWriteTool) Concurrency(input ToolInput) ConcurrencyDecision {
	return ConcurrencySerial
}

func (t *FileWriteTool) Execute(ctx context.Context, input ToolInput) (ToolOutput, error) {
	select {
	case <-ctx.Done():
		return ToolOutput{}, ctx.Err()
	default:
	}

	filePath, ok := stringParam(input.Params, "file_path")
	if !ok || strings.TrimSpace(filePath) == "" {
		return ToolOutput{}, fmt.Errorf("file_write requires file_path")
	}
	filePath, err := resolveToolPath(filePath)
	if err != nil {
		return ToolOutput{}, err
	}

	content, ok := stringParam(input.Params, "content")
	if !ok {
		return ToolOutput{}, fmt.Errorf("file_write requires content")
	}
	overwrite := boolParam(input.Params, "overwrite")

	writeType := "created"
	oldContent := ""
	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
		} else {
			return ToolOutput{}, fmt.Errorf("stat file %q: %w", filePath, err)
		}
	} else {
		if !overwrite {
			return ToolOutput{}, fmt.Errorf("file already exists: %s (set overwrite=true to replace it, or use file_edit for in-place changes)", filePath)
		}
		writeType = "overwritten"
		previousBytes, err := os.ReadFile(filePath)
		if err != nil {
			return ToolOutput{}, fmt.Errorf("read existing file %q: %w", filePath, err)
		}
		oldContent = string(previousBytes)
	}

	trackFileBeforeWrite(filePath)

	parentDir := filepath.Dir(filePath)
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		return ToolOutput{}, fmt.Errorf("create parent directory %q: %w", parentDir, err)
	}

	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		return ToolOutput{}, fmt.Errorf("write file %q: %w", filePath, err)
	}

	preview, insertions, deletions := buildFileDiffPreview(oldContent, content)

	return ToolOutput{
		Output:     fmt.Sprintf("File %s successfully: %s", writeType, filePath),
		FilePath:   filePath,
		Preview:    preview,
		Insertions: insertions,
		Deletions:  deletions,
	}, nil
}
