# Fix Progress

Tracking fixes per plan.md.

---

## Task 26 — Use `parametersJsonSchema` for Gemini tool declarations

**File**: `gocode/internal/api/gemini.go`

Renamed `geminiFunctionDeclaration.Parameters` → `ParametersJsonSchema` with tag
`json:"parametersJsonSchema,omitempty"`. This makes Gemini accept full JSON Schema
(anyOf, oneOf, const, \$defs) for tool input schemas instead of the restricted
OpenAPI 3.03 `parameters` field.

---
