package openai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/imroc/req/v3"
)

// Codex RT 相关常量
const (
	// Codex OAuth 端点
	CodexTokenURL     = "https://auth.openai.com/oauth/token"
	CodexAuthorizeURL = "https://auth.openai.com/oauth/authorize"

	// Session 相关
	SessionCookie     = "__Secure-next-auth.session-token"
	AuthSessionCookie = "oai-client-auth-session"
)

// CodexClient handles Codex RT operations
type CodexClient struct {
	client   *req.Client
	proxyURL string
}

// SessionInfo extracted from oai-client-auth-session
type SessionInfo struct {
	WorkspaceID string `json:"workspace_id"`
	UserID      string `json:"user_id"`
	Email       string `json:"email"`
}

// NewCodexClient creates a new Codex client
func NewCodexClient(proxyURL string) *CodexClient {
	client := req.C().
		SetTimeout(30 * time.Second).
		SetUserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	if proxyURL != "" {
		client.SetProxyURL(proxyURL)
	}

	return &CodexClient{
		client:   client,
		proxyURL: proxyURL,
	}
}

// ExchangeSessionForRT exchanges session token for refresh token via Codex flow
func (c *CodexClient) ExchangeSessionForRT(ctx context.Context, sessionToken string) (*TokenResponse, error) {
	// Step 1: Generate PKCE
	codeVerifier, err := GenerateCodeVerifier()
	if err != nil {
		return nil, fmt.Errorf("generate verifier: %w", err)
	}
	codeChallenge := GenerateCodeChallenge(codeVerifier)
	state, _ := GenerateState()

	// Step 2: Authorization request with session cookie
	authParams := url.Values{}
	authParams.Set("response_type", "code")
	authParams.Set("client_id", ClientID)
	authParams.Set("redirect_uri", DefaultRedirectURI)
	authParams.Set("scope", DefaultScopes)
	authParams.Set("state", state)
	authParams.Set("code_challenge", codeChallenge)
	authParams.Set("code_challenge_method", "S256")
	authParams.Set("codex_cli_simplified_flow", "true")

	authURL := fmt.Sprintf("%s?%s", CodexAuthorizeURL, authParams.Encode())

	// Make auth request with session cookie
	r, err := c.client.R().
		SetContext(ctx).
		SetCookies(&http.Cookie{
			Name:  SessionCookie,
			Value: sessionToken,
		}).
		Get(authURL)

	if err != nil && !strings.Contains(err.Error(), "redirect") {
		return nil, fmt.Errorf("auth request: %w", err)
	}

	// Extract code from redirect location
	location := r.Header.Get("Location")
	if location == "" {
		return nil, fmt.Errorf("no redirect location, status: %d", r.StatusCode)
	}

	parsedURL, err := url.Parse(location)
	if err != nil {
		return nil, fmt.Errorf("parse redirect: %w", err)
	}

	code := parsedURL.Query().Get("code")
	if code == "" {
		errMsg := parsedURL.Query().Get("error")
		return nil, fmt.Errorf("no code in redirect, error: %s", errMsg)
	}

	// Step 3: Exchange code for tokens
	return c.exchangeCode(ctx, code, codeVerifier)
}

// exchangeCode exchanges authorization code for tokens
func (c *CodexClient) exchangeCode(ctx context.Context, code, codeVerifier string) (*TokenResponse, error) {
	formData := url.Values{}
	formData.Set("grant_type", "authorization_code")
	formData.Set("client_id", ClientID)
	formData.Set("code", code)
	formData.Set("redirect_uri", DefaultRedirectURI)
	formData.Set("code_verifier", codeVerifier)

	var tokenResp TokenResponse
	r, err := c.client.R().
		SetContext(ctx).
		SetFormDataFromValues(formData).
		SetSuccessResult(&tokenResp).
		Post(CodexTokenURL)

	if err != nil {
		return nil, fmt.Errorf("token request: %w", err)
	}
	if r.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token failed: %d - %s", r.StatusCode, r.String())
	}

	return &tokenResp, nil
}

// ParseAuthSession parses oai-client-auth-session cookie (base64 JSON)
func ParseAuthSession(cookieValue string) (*SessionInfo, error) {
	decoded, err := base64.StdEncoding.DecodeString(cookieValue)
	if err != nil {
		// Try URL decoding first
		unescaped, _ := url.QueryUnescape(cookieValue)
		decoded, err = base64.StdEncoding.DecodeString(unescaped)
		if err != nil {
			return nil, fmt.Errorf("decode base64: %w", err)
		}
	}

	var info SessionInfo
	if err := json.Unmarshal(decoded, &info); err != nil {
		return nil, fmt.Errorf("parse json: %w", err)
	}
	return &info, nil
}
