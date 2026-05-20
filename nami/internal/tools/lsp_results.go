package tools

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

func limitLSPResults(rows []lspResultRow, maxResults int) []lspResultRow {
	if maxResults <= 0 || len(rows) <= maxResults {
		return rows
	}
	return rows[:maxResults]
}

func locationRowsFromAny(raw any, kind string) []lspResultRow {
	data, err := json.Marshal(raw)
	if err != nil {
		return nil
	}
	var links []lspLocationLink
	if err := json.Unmarshal(data, &links); err == nil && len(links) > 0 {
		rows := make([]lspResultRow, 0, len(links))
		for _, link := range links {
			row := lspResultRow{Kind: kind, FilePath: fileURIToPath(link.TargetURI)}
			applyRange(&row, &link.TargetSelectionRange)
			if row.Line == 0 {
				applyRange(&row, &link.TargetRange)
			}
			rows = append(rows, row)
		}
		return sortLSPRows(rows)
	}
	var locations []lspLocation
	if err := json.Unmarshal(data, &locations); err == nil && len(locations) > 0 {
		return rowsFromLocations(locations, kind)
	}
	var location lspLocation
	if err := json.Unmarshal(data, &location); err == nil && location.URI != "" {
		return rowsFromLocations([]lspLocation{location}, kind)
	}
	var link lspLocationLink
	if err := json.Unmarshal(data, &link); err == nil && link.TargetURI != "" {
		row := lspResultRow{Kind: kind, FilePath: fileURIToPath(link.TargetURI)}
		applyRange(&row, &link.TargetSelectionRange)
		if row.Line == 0 {
			applyRange(&row, &link.TargetRange)
		}
		return []lspResultRow{row}
	}
	return nil
}

func rowsFromLocations(locations []lspLocation, kind string) []lspResultRow {
	rows := make([]lspResultRow, 0, len(locations))
	for _, location := range locations {
		row := lspResultRow{Kind: kind, FilePath: fileURIToPath(location.URI)}
		applyRange(&row, &location.Range)
		rows = append(rows, row)
	}
	return sortLSPRows(rows)
}

func rowsFromWorkspaceSymbols(symbols []lspSymbolInformation) []lspResultRow {
	rows := make([]lspResultRow, 0, len(symbols))
	for _, symbol := range symbols {
		row := lspResultRow{
			Kind:          "symbol",
			Name:          symbol.Name,
			SymbolKind:    lspSymbolKindName(symbol.Kind),
			FilePath:      fileURIToPath(symbol.Location.URI),
			ContainerName: symbol.ContainerName,
		}
		applyRange(&row, &symbol.Location.Range)
		rows = append(rows, row)
	}
	return sortLSPRows(rows)
}

func documentSymbolRows(raw json.RawMessage, filePath string) []lspResultRow {
	var documentSymbols []lspDocumentSymbol
	if err := json.Unmarshal(raw, &documentSymbols); err == nil && len(documentSymbols) > 0 {
		rows := make([]lspResultRow, 0, len(documentSymbols))
		flattenDocumentSymbols(&rows, documentSymbols, filePath, "")
		return sortLSPRows(rows)
	}
	var symbolInfos []lspSymbolInformation
	if err := json.Unmarshal(raw, &symbolInfos); err == nil && len(symbolInfos) > 0 {
		return rowsFromWorkspaceSymbols(symbolInfos)
	}
	return nil
}

func flattenDocumentSymbols(rows *[]lspResultRow, symbols []lspDocumentSymbol, filePath, container string) {
	for _, symbol := range symbols {
		row := lspResultRow{
			Kind:          "symbol",
			Name:          symbol.Name,
			Detail:        symbol.Detail,
			SymbolKind:    lspSymbolKindName(symbol.Kind),
			FilePath:      filePath,
			ContainerName: container,
		}
		applyRange(&row, &symbol.SelectionRange)
		if row.Line == 0 {
			applyRange(&row, &symbol.Range)
		}
		*rows = append(*rows, row)
		nextContainer := symbol.Name
		if container != "" {
			nextContainer = container + "." + symbol.Name
		}
		flattenDocumentSymbols(rows, symbol.Children, filePath, nextContainer)
	}
}

func sortLSPRows(rows []lspResultRow) []lspResultRow {
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].FilePath != rows[j].FilePath {
			return rows[i].FilePath < rows[j].FilePath
		}
		if rows[i].Line != rows[j].Line {
			return rows[i].Line < rows[j].Line
		}
		if rows[i].Column != rows[j].Column {
			return rows[i].Column < rows[j].Column
		}
		return rows[i].Name < rows[j].Name
	})
	return rows
}

func applyRange(row *lspResultRow, source *lspRange) {
	if row == nil || source == nil {
		return
	}
	row.Line = source.Start.Line + 1
	row.Column = source.Start.Character + 1
	row.EndLine = source.End.Line + 1
	row.EndColumn = source.End.Character + 1
}

func extractHoverContents(value any) string {
	if value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	case map[string]any:
		if raw, ok := typed["value"].(string); ok {
			return raw
		}
		if language, ok := typed["language"].(string); ok {
			if raw, ok := typed["value"].(string); ok && strings.TrimSpace(raw) != "" {
				return language + "\n" + raw
			}
		}
	case []any:
		parts := make([]string, 0, len(typed))
		for _, part := range typed {
			if text := strings.TrimSpace(extractHoverContents(part)); text != "" {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, "\n\n")
	}
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprint(value)
	}
	var markup lspMarkupContent
	if err := json.Unmarshal(data, &markup); err == nil && strings.TrimSpace(markup.Value) != "" {
		return markup.Value
	}
	var marked lspMarkedString
	if err := json.Unmarshal(data, &marked); err == nil && strings.TrimSpace(marked.Value) != "" {
		if strings.TrimSpace(marked.Language) == "" {
			return marked.Value
		}
		return marked.Language + "\n" + marked.Value
	}
	return string(data)
}

func lspSymbolKindName(kind int) string {
	switch kind {
	case 1:
		return "file"
	case 2:
		return "module"
	case 3:
		return "namespace"
	case 4:
		return "package"
	case 5:
		return "class"
	case 6:
		return "method"
	case 7:
		return "property"
	case 8:
		return "field"
	case 9:
		return "constructor"
	case 10:
		return "enum"
	case 11:
		return "interface"
	case 12:
		return "function"
	case 13:
		return "variable"
	case 14:
		return "constant"
	case 15:
		return "string"
	case 16:
		return "number"
	case 17:
		return "boolean"
	case 18:
		return "array"
	case 19:
		return "object"
	case 20:
		return "key"
	case 21:
		return "null"
	case 22:
		return "enum_member"
	case 23:
		return "struct"
	case 24:
		return "event"
	case 25:
		return "operator"
	case 26:
		return "type_parameter"
	default:
		return fmt.Sprintf("kind_%d", kind)
	}
}
