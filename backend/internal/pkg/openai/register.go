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

