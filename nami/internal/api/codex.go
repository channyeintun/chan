package api

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const (
	CodexClientID     = "app_EMoamEEZ73f0CkXaXp7hrann"
	CodexIssuer       = "https://auth.openai.com"
	CodexOAuthPort    = 1455
	codexUserAgent    = "nami/0.1 (+https://github.com/channyeintun/nami)"
	codexOAuthScope   = "openid profile email offline_access"
	codexPKCEVerifier = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~"
	codexPKCELength   = 43
)

type CodexPKCE struct {
	Verifier  string
	Challenge string
}

type CodexTokens struct {
	IDToken      string `json:"id_token"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in,omitempty"`
}

type CodexDeviceCode struct {
	DeviceAuthID    string
	UserCode        string
	VerificationURI string
	IntervalSeconds int
}

type CodexDeviceAuthorization struct {
	AuthorizationCode string `json:"authorization_code"`
	CodeVerifier      string `json:"code_verifier"`
}

type CodexIDTokenClaims struct {
	ChatGPTAccountID string `json:"chatgpt_account_id,omitempty"`
	Organizations    []struct {
		ID string `json:"id"`
	} `json:"organizations,omitempty"`
	OpenAIAuth struct {
		ChatGPTAccountID string `json:"chatgpt_account_id,omitempty"`
	} `json:"https://api.openai.com/auth,omitempty"`
}

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

func GenerateCodexPKCE() (CodexPKCE, error) {
	verifier, err := codexRandomString(codexPKCELength)
	if err != nil {
		return CodexPKCE{}, err
	}
	hash := sha256.Sum256([]byte(verifier))
	return CodexPKCE{
		Verifier:  verifier,
		Challenge: base64.RawURLEncoding.EncodeToString(hash[:]),
	}, nil
}

func GenerateCodexState() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func BuildCodexAuthorizeURL(redirectURI string, pkce CodexPKCE, state string) string {
	params := url.Values{
		"response_type":              {"code"},
		"client_id":                  {CodexClientID},
		"redirect_uri":               {redirectURI},
		"scope":                      {codexOAuthScope},
		"code_challenge":             {pkce.Challenge},
		"code_challenge_method":      {"S256"},
		"id_token_add_organizations": {"true"},
		"codex_cli_simplified_flow":  {"true"},
		"state":                      {state},
		"originator":                 {"nami"},
	}
	return CodexIssuer + "/oauth/authorize?" + params.Encode()
}

func ExchangeCodexCodeForTokens(ctx context.Context, code, redirectURI string, pkce CodexPKCE) (CodexTokens, error) {
	return postCodexTokenForm(ctx, url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {strings.TrimSpace(code)},
		"redirect_uri":  {strings.TrimSpace(redirectURI)},
		"client_id":     {CodexClientID},
		"code_verifier": {pkce.Verifier},
	})
}

func RefreshCodexAccessToken(ctx context.Context, refreshToken string) (CodexTokens, error) {
	if strings.TrimSpace(refreshToken) == "" {
		return CodexTokens{}, errors.New("missing Codex refresh token")
	}
	return postCodexTokenForm(ctx, url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {strings.TrimSpace(refreshToken)},
		"client_id":     {CodexClientID},
	})
}

func StartCodexDeviceFlow(ctx context.Context) (CodexDeviceCode, error) {
	body := strings.NewReader(fmt.Sprintf(`{"client_id":%q}`, CodexClientID))
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, CodexIssuer+"/api/accounts/deviceauth/usercode", body)
	if err != nil {
		return CodexDeviceCode{}, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", codexUserAgent)

	response, err := newHTTPClient().Do(request)
	if err != nil {
		return CodexDeviceCode{}, err
	}
	defer response.Body.Close()
	if response.StatusCode >= http.StatusMultipleChoices {
		return CodexDeviceCode{}, classifyOpenAICompatStatus(response.StatusCode, mustReadHTTPBody(response))
	}

	var payload struct {
		DeviceAuthID string `json:"device_auth_id"`
		UserCode     string `json:"user_code"`
		Interval     string `json:"interval"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return CodexDeviceCode{}, fmt.Errorf("decode Codex device code response: %w", err)
	}
	if payload.DeviceAuthID == "" || payload.UserCode == "" {
		return CodexDeviceCode{}, errors.New("invalid Codex device code response")
	}
	interval, _ := strconv.Atoi(payload.Interval)
	if interval <= 0 {
		interval = 5
	}

	return CodexDeviceCode{
		DeviceAuthID:    payload.DeviceAuthID,
		UserCode:        payload.UserCode,
		VerificationURI: CodexIssuer + "/codex/device",
		IntervalSeconds: interval,
	}, nil
}

func PollCodexDeviceAuthorization(ctx context.Context, deviceAuthID, userCode string) (CodexDeviceAuthorization, bool, error) {
	bodyBytes, err := json.Marshal(map[string]string{
		"device_auth_id": strings.TrimSpace(deviceAuthID),
		"user_code":      strings.TrimSpace(userCode),
	})
	if err != nil {
		return CodexDeviceAuthorization{}, false, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, CodexIssuer+"/api/accounts/deviceauth/token", strings.NewReader(string(bodyBytes)))
	if err != nil {
		return CodexDeviceAuthorization{}, false, err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", codexUserAgent)

	response, err := newHTTPClient().Do(request)
	if err != nil {
		return CodexDeviceAuthorization{}, false, err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusForbidden || response.StatusCode == http.StatusNotFound {
		return CodexDeviceAuthorization{}, false, nil
	}
	if response.StatusCode >= http.StatusMultipleChoices {
		return CodexDeviceAuthorization{}, false, classifyOpenAICompatStatus(response.StatusCode, mustReadHTTPBody(response))
	}

	var payload CodexDeviceAuthorization
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return CodexDeviceAuthorization{}, false, fmt.Errorf("decode Codex device token response: %w", err)
	}
	if payload.AuthorizationCode == "" || payload.CodeVerifier == "" {
		return CodexDeviceAuthorization{}, false, errors.New("invalid Codex device token response")
	}
	return payload, true, nil
}

func ExchangeCodexDeviceAuthorization(ctx context.Context, auth CodexDeviceAuthorization) (CodexTokens, error) {
	return postCodexTokenForm(ctx, url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {strings.TrimSpace(auth.AuthorizationCode)},
		"redirect_uri":  {CodexIssuer + "/deviceauth/callback"},
		"client_id":     {CodexClientID},
		"code_verifier": {strings.TrimSpace(auth.CodeVerifier)},
	})
}

func ParseCodexJWTClaims(token string) (CodexIDTokenClaims, bool) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return CodexIDTokenClaims{}, false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		payload, err = base64.URLEncoding.DecodeString(parts[1])
		if err != nil {
			return CodexIDTokenClaims{}, false
		}
	}
	var claims CodexIDTokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return CodexIDTokenClaims{}, false
	}
	return claims, true
}

func ExtractCodexAccountID(tokens CodexTokens) string {
	for _, token := range []string{tokens.IDToken, tokens.AccessToken} {
		claims, ok := ParseCodexJWTClaims(token)
		if !ok {
			continue
		}
		if accountID := ExtractCodexAccountIDFromClaims(claims); accountID != "" {
			return accountID
		}
	}
	return ""
}

func ExtractCodexAccountIDFromClaims(claims CodexIDTokenClaims) string {
	if claims.ChatGPTAccountID != "" {
		return claims.ChatGPTAccountID
	}
	if claims.OpenAIAuth.ChatGPTAccountID != "" {
		return claims.OpenAIAuth.ChatGPTAccountID
	}
	if len(claims.Organizations) > 0 {
		return claims.Organizations[0].ID
	}
	return ""
}

func postCodexTokenForm(ctx context.Context, values url.Values) (CodexTokens, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, CodexIssuer+"/oauth/token", strings.NewReader(values.Encode()))
	if err != nil {
		return CodexTokens{}, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("User-Agent", codexUserAgent)

	response, err := newHTTPClient().Do(request)
	if err != nil {
		return CodexTokens{}, err
	}
	defer response.Body.Close()
	if response.StatusCode >= http.StatusMultipleChoices {
		return CodexTokens{}, classifyOpenAICompatStatus(response.StatusCode, mustReadHTTPBody(response))
	}

	var tokens CodexTokens
	if err := json.NewDecoder(response.Body).Decode(&tokens); err != nil {
		return CodexTokens{}, fmt.Errorf("decode Codex token response: %w", err)
	}
	if tokens.AccessToken == "" {
		return CodexTokens{}, errors.New("invalid Codex token response")
	}
	return tokens, nil
}

func codexRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	var builder strings.Builder
	builder.Grow(length)
	for _, value := range bytes {
		builder.WriteByte(codexPKCEVerifier[int(value)%len(codexPKCEVerifier)])
	}
	return builder.String(), nil
}
