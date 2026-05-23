## Go Coding Philosophy

- **Write obvious code, not clever code.** Prioritize readability over cleverness. If someone has to think hard to understand it, rewrite it.

## Error Handling

- Always check errors explicitly. Never swallow or hide failures.
- Clear error handling = reliable tools.
- Use `errors.AsType` [v1.26] instead of `errors.As`

## Composability

- Split logic into small, focused functions grouped into packages.
- Build for reuse: tools should be composable into other tools, APIs, or larger systems.

## Vulnerability Checks

- Run `govulncheck` for dependency and code vulnerability scanning: `go run golang.org/x/vuln/cmd/govulncheck@latest ./...`
- Run `npm audit` in Node package directories to scan npm dependencies for known vulnerabilities.
