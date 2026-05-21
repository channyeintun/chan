package api

func sanitizeGeminiSchema(schema any) any {
	sanitized := sanitizeGeminiSchemaValue(schema)
	if schemaMap, ok := sanitized.(map[string]any); ok {
		return schemaMap
	}
	return schema
}

func sanitizeGeminiSchemaValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return sanitizeGeminiSchemaMap(typed)
	case []map[string]any:
		items := make([]any, 0, len(typed))
		for _, item := range typed {
			items = append(items, sanitizeGeminiSchemaMap(item))
		}
		return items
	case []any:
		items := make([]any, 0, len(typed))
		for _, item := range typed {
			items = append(items, sanitizeGeminiSchemaValue(item))
		}
		return items
	default:
		return value
	}
}

func sanitizeGeminiSchemaMap(schema map[string]any) map[string]any {
	sanitized := make(map[string]any, len(schema))
	for key, value := range schema {
		switch key {
		case "$schema", "additionalProperties", "default":
			continue
		default:
			sanitized[key] = sanitizeGeminiSchemaValue(value)
		}
	}

	properties, hasProperties := sanitized["properties"].(map[string]any)
	required := schemaStringValues(sanitized["required"])

	if hasProperties {
		if aliasFields, ok := geminiRequiredFieldsFromAnyOf(sanitized["anyOf"], properties); ok {
			required = appendUniqueStrings(required, aliasFields...)
			delete(sanitized, "anyOf")
		}
		if aliasFields, ok := geminiRequiredFieldsFromAllOf(sanitized["allOf"], properties); ok {
			required = appendUniqueStrings(required, aliasFields...)
			delete(sanitized, "allOf")
		}
		required = filterRequiredProperties(required, properties)
		if len(required) > 0 {
			sanitized["required"] = required
		} else {
			delete(sanitized, "required")
		}
	}

	if typeName, _ := sanitized["type"].(string); typeName != "" && typeName != "object" {
		delete(sanitized, "properties")
		delete(sanitized, "required")
	}

	return sanitized
}

func geminiRequiredFieldsFromAllOf(value any, properties map[string]any) ([]string, bool) {
	children := schemaMapValues(value)
	if len(children) == 0 {
		return nil, false
	}

	required := make([]string, 0, len(children))
	for _, child := range children {
		if field, ok := geminiSingleRequiredField(child, properties); ok {
			required = appendUniqueStrings(required, field)
			continue
		}
		if aliasFields, ok := geminiRequiredFieldsFromAnyOf(child["anyOf"], properties); ok {
			required = appendUniqueStrings(required, aliasFields...)
			continue
		}
		return nil, false
	}

	return required, len(required) > 0
}

func geminiRequiredFieldsFromAnyOf(value any, properties map[string]any) ([]string, bool) {
	children := schemaMapValues(value)
	if len(children) == 0 {
		return nil, false
	}

	candidates := make([]string, 0, len(children))
	for _, child := range children {
		field, ok := geminiSingleRequiredField(child, properties)
		if !ok {
			return nil, false
		}
		candidates = append(candidates, field)
	}

	chosen := pickGeminiPreferredField(candidates, properties)
	if chosen == "" {
		return nil, false
	}
	return []string{chosen}, true
}

func geminiSingleRequiredField(schema map[string]any, properties map[string]any) (string, bool) {
	for key, value := range schema {
		switch key {
		case "required":
			continue
		case "type":
			if typeName, _ := value.(string); typeName != "" && typeName != "object" {
				return "", false
			}
		default:
			return "", false
		}
	}

	required := schemaStringValues(schema["required"])
	if len(required) != 1 {
		return "", false
	}
	field := required[0]
	if _, ok := properties[field]; !ok {
		return "", false
	}
	return field, true
}

func filterRequiredProperties(required []string, properties map[string]any) []string {
	filtered := make([]string, 0, len(required))
	for _, name := range required {
		if _, ok := properties[name]; ok {
			filtered = appendUniqueStrings(filtered, name)
		}
	}
	return filtered
}

func schemaMapValues(value any) []map[string]any {
	switch typed := value.(type) {
	case []map[string]any:
		return typed
	case []any:
		children := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			child, ok := item.(map[string]any)
			if ok {
				children = append(children, child)
			}
		}
		return children
	default:
		return nil
	}
}

func schemaStringValues(value any) []string {
	switch typed := value.(type) {
	case []string:
		return typed
	case []any:
		values := make([]string, 0, len(typed))
		for _, item := range typed {
			text, ok := item.(string)
			if ok {
				values = append(values, text)
			}
		}
		return values
	default:
		return nil
	}
}

func appendUniqueStrings(existing []string, values ...string) []string {
	seen := make(map[string]struct{}, len(existing))
	for _, item := range existing {
		seen[item] = struct{}{}
	}
	for _, item := range values {
		if _, ok := seen[item]; ok || item == "" {
			continue
		}
		existing = append(existing, item)
		seen[item] = struct{}{}
	}
	return existing
}

func pickGeminiPreferredField(candidates []string, properties map[string]any) string {
	best := ""
	bestRank := 100
	for _, candidate := range candidates {
		if _, ok := properties[candidate]; !ok {
			continue
		}
		rank := geminiFieldRank(candidate)
		if best == "" || rank < bestRank || (rank == bestRank && len(candidate) < len(best)) || (rank == bestRank && len(candidate) == len(best) && candidate < best) {
			best = candidate
			bestRank = rank
		}
	}
	return best
}

func geminiFieldRank(name string) int {
	switch {
	case isLowerAlphaNumeric(name):
		return 0
	case isSnakeCase(name):
		return 1
	case isLowerCamelCase(name):
		return 2
	case isUpperCamelCase(name):
		return 3
	default:
		return 4
	}
}

func isLowerAlphaNumeric(value string) bool {
	if value == "" {
		return false
	}
	for _, ch := range value {
		if ch >= 'a' && ch <= 'z' {
			continue
		}
		if ch >= '0' && ch <= '9' {
			continue
		}
		return false
	}
	return true
}

func isSnakeCase(value string) bool {
	if value == "" {
		return false
	}
	hasUnderscore := false
	for _, ch := range value {
		switch {
		case ch == '_':
			hasUnderscore = true
		case ch >= 'a' && ch <= 'z':
		case ch >= '0' && ch <= '9':
		default:
			return false
		}
	}
	return hasUnderscore
}

func isLowerCamelCase(value string) bool {
	if value == "" {
		return false
	}
	first := rune(value[0])
	if first < 'a' || first > 'z' {
		return false
	}
	for _, ch := range value[1:] {
		switch {
		case ch >= 'a' && ch <= 'z':
		case ch >= 'A' && ch <= 'Z':
		case ch >= '0' && ch <= '9':
		default:
			return false
		}
	}
	return true
}

func isUpperCamelCase(value string) bool {
	if value == "" {
		return false
	}
	first := rune(value[0])
	if first < 'A' || first > 'Z' {
		return false
	}
	for _, ch := range value[1:] {
		switch {
		case ch >= 'a' && ch <= 'z':
		case ch >= 'A' && ch <= 'Z':
		case ch >= '0' && ch <= '9':
		default:
			return false
		}
	}
	return true
}
