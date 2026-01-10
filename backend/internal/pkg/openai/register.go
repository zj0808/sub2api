package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/imroc/req/v3"
)

// OpenAI Auth Endpoints
const (
	// 两套注册 API
	RegisterURLChatGPT = "https://auth.openai.com/api/accounts/user/register" // chatgpt.com 入口，不检验 sentinel
	RegisterURLOpenAI  = "https://auth.openai.com/create-account/password"    // openai.com 入口

	// 其他端点
	SentinelURL = "https://chatgpt.com/backend-api/sentinel/req"
	LoginURL    = "https://auth.openai.com/oauth/token"
	CSRFCookie  = "__Host-next-auth.csrf-token"
)

// RegisterRequest represents registration request
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name,omitempty"`
}

// RegisterResponse represents registration response
type RegisterResponse struct {
	Success bool   `json:"success"`
	UserID  string `json:"user_id,omitempty"`
	Email   string `json:"email,omitempty"`
	Error   string `json:"error,omitempty"`
}

// SentinelResponse from /backend-api/sentinel/req
type SentinelResponse struct {
	TurnstileRequired bool          `json:"turnstile_required"`
	PowChallenge      *PowChallenge `json:"pow,omitempty"`
	Token             string        `json:"token,omitempty"`
}

// RegisterClient handles OpenAI registration
type RegisterClient struct {
	client   *req.Client
	proxyURL string
}

// NewRegisterClient creates a new registration client
func NewRegisterClient(proxyURL string) *RegisterClient {
	client := req.C().
		SetTimeout(30 * time.Second).
		SetUserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	if proxyURL != "" {
		client.SetProxyURL(proxyURL)
	}

	return &RegisterClient{
		client:   client,
		proxyURL: proxyURL,
	}
}

// GetSentinelChallenge gets PoW challenge from sentinel endpoint
func (c *RegisterClient) GetSentinelChallenge(ctx context.Context) (*SentinelResponse, error) {
	var resp SentinelResponse
	r, err := c.client.R().
		SetContext(ctx).
		SetSuccessResult(&resp).
		Get(SentinelURL)

	if err != nil {
		return nil, fmt.Errorf("sentinel request: %w", err)
	}
	if r.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sentinel status: %d", r.StatusCode)
	}
	return &resp, nil
}

// Register registers a new OpenAI account
// Uses chatgpt.com entry point which doesn't verify sentinel token
func (c *RegisterClient) Register(ctx context.Context, req *RegisterRequest) (*RegisterResponse, error) {
	if req.Name == "" {
		req.Name = strings.Split(req.Email, "@")[0]
	}

	body, _ := json.Marshal(req)

	r, err := c.client.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetHeader("Origin", "https://chatgpt.com").
		SetHeader("Referer", "https://chatgpt.com/").
		SetBody(body).
		Post(RegisterURLChatGPT)

	if err != nil {
		return nil, fmt.Errorf("register request: %w", err)
	}

	var resp RegisterResponse
	if err := json.Unmarshal(r.Bytes(), &resp); err != nil {
		// Try to extract error from response
		resp.Error = r.String()
		resp.Success = false
	}

	if r.StatusCode >= 400 {
		return &resp, fmt.Errorf("register failed: %s", resp.Error)
	}

	resp.Success = true
	return &resp, nil
}

// GetCSRFToken extracts CSRF token from cookies
func (c *RegisterClient) GetCSRFToken(cookies []*http.Cookie) string {
	for _, cookie := range cookies {
		if cookie.Name == CSRFCookie {
			// Format: token|hash
			parts := strings.Split(cookie.Value, "|")
			if len(parts) > 0 {
				if decoded, err := url.QueryUnescape(parts[0]); err == nil {
					return decoded
				}
				return parts[0]
			}
		}
	}
	return ""
}

// VerifyEmailRequest represents email verification request
type VerifyEmailRequest struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

// VerifyEmailResponse represents email verification response
type VerifyEmailResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// VerifyEmail submits verification code to OpenAI
// POST https://auth.openai.com/api/accounts/user/verify
func (c *RegisterClient) VerifyEmail(ctx context.Context, email, code string) (*VerifyEmailResponse, error) {
	body, _ := json.Marshal(VerifyEmailRequest{
		Email: email,
		Code:  code,
	})

	r, err := c.client.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetHeader("Origin", "https://chatgpt.com").
		SetHeader("Referer", "https://chatgpt.com/").
		SetBody(body).
		Post("https://auth.openai.com/api/accounts/user/verify")

	if err != nil {
		return nil, fmt.Errorf("verify request: %w", err)
	}

	var resp VerifyEmailResponse
	if err := json.Unmarshal(r.Bytes(), &resp); err != nil {
		resp.Error = r.String()
		resp.Success = false
	}

	if r.StatusCode >= 400 {
		return &resp, fmt.Errorf("verify failed: %s", resp.Error)
	}

	resp.Success = true
	return &resp, nil
}

// LoginRequest represents login request
type LoginRequest struct {
	GrantType string `json:"grant_type"`
	ClientID  string `json:"client_id"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	Scope     string `json:"scope"`
}

// LoginResponse represents login response with tokens
type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	Error        string `json:"error,omitempty"`
	ErrorDesc    string `json:"error_description,omitempty"`
}

// LoginWithPassword performs password-based login to get refresh token
// Uses Resource Owner Password flow
func (c *RegisterClient) LoginWithPassword(ctx context.Context, email, password string) (*LoginResponse, error) {
	formData := url.Values{}
	formData.Set("grant_type", "password")
	formData.Set("client_id", ClientID)
	formData.Set("username", email)
	formData.Set("password", password)
	formData.Set("scope", DefaultScopes)

	r, err := c.client.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetBody(formData.Encode()).
		Post(LoginURL)

	if err != nil {
		return nil, fmt.Errorf("login request: %w", err)
	}

	var resp LoginResponse
	if err := json.Unmarshal(r.Bytes(), &resp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if r.StatusCode >= 400 || resp.Error != "" {
		errMsg := resp.ErrorDesc
		if errMsg == "" {
			errMsg = resp.Error
		}
		return &resp, fmt.Errorf("login failed: %s", errMsg)
	}

	return &resp, nil
}
