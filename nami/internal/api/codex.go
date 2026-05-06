package api

import "strings"

const codexUserAgent = "nami/0.1 (+https://github.com/channyeintun/nami)"

func CodexStaticHeaders(accountID string) map[string]string {
	headers := map[string]string{
		"originator": "nami",
		"User-Agent": codexUserAgent,
	}
	if trimmed := strings.TrimSpace(accountID); trimmed != "" {
		headers["ChatGPT-Account-Id"] = trimmed
	}
	return headers
}
