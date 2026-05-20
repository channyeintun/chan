package engine

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/channyeintun/nami/internal/api"
	toolpkg "github.com/channyeintun/nami/internal/tools"
)

func stringParamFromMap(params map[string]any, key string) (string, bool) {
	value, ok := params[key]
	if !ok {
		return "", false
	}
	stringValue, ok := value.(string)
	return stringValue, ok
}

func firstStringParamFromMap(params map[string]any, keys ...string) (string, bool) {
	for _, key := range keys {
		if value, ok := stringParamFromMap(params, key); ok {
			return value, true
		}
	}
	return "", false
}

func decodeToolInput(call api.ToolCall) (toolpkg.ToolInput, error) {
	params := make(map[string]any)
	raw := strings.TrimSpace(call.Input)
	if raw != "" {
		if err := json.Unmarshal([]byte(raw), &params); err != nil {
			return toolpkg.ToolInput{}, fmt.Errorf("decode tool input for %q: %w", call.Name, err)
		}
	}
	return toolpkg.ToolInput{
		Name:   call.Name,
		Params: params,
		Raw:    call.Input,
	}, nil
}

func normalizeToolCall(call api.ToolCall) (api.ToolCall, error) {
	alias := strings.TrimSpace(call.Name)
	switch alias {
	case "file_search", "grep_search", "read_file", "replace_string_in_file", "glob", "grep", "file_read", "file_edit", "google:search", "google_search", "google.search":
	default:
		return call, nil
	}

	params := make(map[string]any)
	raw := strings.TrimSpace(call.Input)
	if raw != "" {
		if err := json.Unmarshal([]byte(raw), &params); err != nil {
			return api.ToolCall{}, fmt.Errorf("decode tool input for %q: %w", call.Name, err)
		}
	}

	normalized := call
	normalizedParams := cloneToolParams(params)

	switch alias {
	case "file_search", "glob":
		normalized.Name = "file_search"
		if pattern, ok := stringParamFromMap(normalizedParams, "pattern"); !ok || strings.TrimSpace(pattern) == "" {
			if query, ok := stringParamFromMap(normalizedParams, "query"); ok && strings.TrimSpace(query) != "" {
				normalizedParams["pattern"] = normalizeFileSearchPattern(query)
			}
		}
		renameToolParam(normalizedParams, "pattern", "query")
		if _, ok := stringParamFromMap(normalizedParams, "path"); !ok {
			if includePattern, ok := stringParamFromMap(normalizedParams, "includePattern"); ok && strings.TrimSpace(includePattern) != "" && !looksLikeGlob(includePattern) {
				normalizedParams["path"] = includePattern
			}
		}
	case "grep_search", "grep":
		normalized.Name = "grep_search"
		if pattern, ok := stringParamFromMap(normalizedParams, "pattern"); !ok || strings.TrimSpace(pattern) == "" {
			if query, ok := stringParamFromMap(normalizedParams, "query"); ok && strings.TrimSpace(query) != "" {
				isRegexp, hasRegexpFlag := normalizedParams["isRegexp"].(bool)
				if hasRegexpFlag && isRegexp {
					normalizedParams["pattern"] = query
				} else {
					normalizedParams["pattern"] = regexp.QuoteMeta(query)
				}
			}
		}
		renameToolParam(normalizedParams, "pattern", "query")
		if _, ok := stringParamFromMap(normalizedParams, "path"); !ok {
			if includePattern, ok := stringParamFromMap(normalizedParams, "includePattern"); ok && strings.TrimSpace(includePattern) != "" {
				if looksLikeGlob(includePattern) {
					normalizedParams["glob"] = includePattern
				} else {
					normalizedParams["path"] = includePattern
				}
			}
		}
		if _, ok := normalizedParams["head_limit"]; !ok {
			if maxResults, ok := intParamFromMap(normalizedParams, "maxResults"); ok && maxResults > 0 {
				normalizedParams["head_limit"] = maxResults
			}
		}
	case "read_file", "file_read":
		normalized.Name = "read_file"
		renameToolParam(normalizedParams, "filePath", "file_path")
	case "replace_string_in_file", "file_edit":
		normalized.Name = "replace_string_in_file"
		renameToolParam(normalizedParams, "filePath", "file_path")
		renameToolParam(normalizedParams, "oldString", "old_string")
		renameToolParam(normalizedParams, "newString", "new_string")
		renameToolParam(normalizedParams, "replaceAll", "replace_all")
	case "google:search", "google_search", "google.search":
		normalized.Name = "web_search"
		if query, ok := stringParamFromMap(normalizedParams, "query"); !ok || strings.TrimSpace(query) == "" {
			if firstQuery, ok := firstStringInArrayParamFromMap(normalizedParams, "queries"); ok {
				normalizedParams["query"] = firstQuery
			}
		}
		renameToolParam(normalizedParams, "max_results", "limit")
	}

	encoded, err := json.Marshal(normalizedParams)
	if err != nil {
		return api.ToolCall{}, fmt.Errorf("encode normalized tool input for %q: %w", call.Name, err)
	}
	normalized.Input = string(encoded)
	return normalized, nil
}

func cloneToolParams(params map[string]any) map[string]any {
	cloned := make(map[string]any, len(params))
	for key, value := range params {
		cloned[key] = value
	}
	return cloned
}

func renameToolParam(params map[string]any, from, to string) {
	if _, exists := params[to]; exists {
		return
	}
	value, ok := params[from]
	if !ok {
		return
	}
	params[to] = value
}

func normalizeFileSearchPattern(query string) string {
	trimmed := strings.TrimSpace(query)
	if trimmed == "" || filepath.IsAbs(trimmed) || looksLikeGlob(trimmed) {
		return trimmed
	}

	normalized := strings.TrimPrefix(filepath.ToSlash(trimmed), "./")
	if normalized == "" {
		return trimmed
	}
	if strings.HasSuffix(normalized, "/") {
		return "**/" + strings.TrimSuffix(normalized, "/") + "/**"
	}
	if strings.Contains(normalized, "/") {
		return "**/" + normalized + "*"
	}
	return "**/*" + normalized + "*"
}

func looksLikeGlob(value string) bool {
	return strings.ContainsAny(value, "*?[]{}")
}

func intParamFromMap(params map[string]any, key string) (int, bool) {
	value, ok := params[key]
	if !ok {
		return 0, false
	}
	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	default:
		return 0, false
	}
}

func firstStringInArrayParamFromMap(params map[string]any, key string) (string, bool) {
	value, ok := params[key]
	if !ok {
		return "", false
	}
	items, ok := value.([]any)
	if !ok {
		return "", false
	}
	for _, item := range items {
		text, ok := item.(string)
		if ok && strings.TrimSpace(text) != "" {
			return text, true
		}
	}
	return "", false
}
