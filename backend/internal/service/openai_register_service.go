package service

import (
	"context"
	"log"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
)

// OpenAIRegisterService handles OpenAI account registration and RT acquisition
type OpenAIRegisterService struct {
	proxyRepo ProxyRepository
}

// NewOpenAIRegisterService creates a new register service
func NewOpenAIRegisterService(proxyRepo ProxyRepository) *OpenAIRegisterService {
	return &OpenAIRegisterService{
		proxyRepo: proxyRepo,
	}
}

// AutoRegisterInput input for auto registration
type AutoRegisterInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	ProxyID  *int64 `json:"proxy_id,omitempty"`
}

// AutoRegisterResult result of auto registration
type AutoRegisterResult struct {
	Success      bool   `json:"success"`
	Email        string `json:"email"`
	UserID       string `json:"user_id,omitempty"`
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresAt    int64  `json:"expires_at,omitempty"`
	Error        string `json:"error,omitempty"`
}

// SessionToRTInput input for session to RT conversion
type SessionToRTInput struct {
	SessionToken string `json:"session_token"`
	ProxyID      *int64 `json:"proxy_id,omitempty"`
}

// SessionToRTResult result of session to RT conversion
type SessionToRTResult struct {
	Success      bool   `json:"success"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token,omitempty"`
	ExpiresIn    int64  `json:"expires_in"`
	ExpiresAt    int64  `json:"expires_at"`
	Email        string `json:"email,omitempty"`
	Error        string `json:"error,omitempty"`
}

// AutoRegister performs automatic OpenAI account registration
func (s *OpenAIRegisterService) AutoRegister(ctx context.Context, input *AutoRegisterInput) (*AutoRegisterResult, error) {
	proxyURL := s.getProxyURL(ctx, input.ProxyID)

	log.Printf("[OpenAIRegister] Starting auto registration for: %s", input.Email)

	client := openai.NewRegisterClient(proxyURL)

	// Step 1: Try to get sentinel challenge (optional, may not be required)
	sentinel, err := client.GetSentinelChallenge(ctx)
	if err != nil {
		log.Printf("[OpenAIRegister] Sentinel request failed (continuing): %v", err)
	} else if sentinel.PowChallenge != nil {
		log.Printf("[OpenAIRegister] PoW challenge received, solving...")
		powResult, err := openai.SolvePow(sentinel.PowChallenge, 60)
		if err != nil {
			log.Printf("[OpenAIRegister] PoW solve failed: %v", err)
		} else {
			log.Printf("[OpenAIRegister] PoW solved: %s", powResult.Answer[:16]+"...")
		}
	}

	// Step 2: Register via chatgpt.com entry (doesn't verify sentinel)
	regResp, err := client.Register(ctx, &openai.RegisterRequest{
		Email:    input.Email,
		Password: input.Password,
	})
	if err != nil {
		return &AutoRegisterResult{
			Success: false,
			Email:   input.Email,
			Error:   err.Error(),
		}, err
	}

	log.Printf("[OpenAIRegister] Registration completed for: %s", input.Email)

	return &AutoRegisterResult{
		Success: regResp.Success,
		Email:   input.Email,
		UserID:  regResp.UserID,
	}, nil
}

// SessionToRT converts session token to refresh token via Codex flow
func (s *OpenAIRegisterService) SessionToRT(ctx context.Context, input *SessionToRTInput) (*SessionToRTResult, error) {
	proxyURL := s.getProxyURL(ctx, input.ProxyID)

	log.Printf("[OpenAIRegister] Converting session to RT...")

	client := openai.NewCodexClient(proxyURL)

	tokenResp, err := client.ExchangeSessionForRT(ctx, input.SessionToken)
	if err != nil {
		return &SessionToRTResult{
			Success: false,
			Error:   err.Error(),
		}, err
	}

	expiresAt := time.Now().Unix() + tokenResp.ExpiresIn

	// Parse ID token to get email
	var email string
	if tokenResp.IDToken != "" {
		if claims, err := openai.ParseIDToken(tokenResp.IDToken); err == nil {
			email = claims.Email
		}
	}

	log.Printf("[OpenAIRegister] Session to RT successful, email: %s", email)

	return &SessionToRTResult{
		Success:      true,
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		IDToken:      tokenResp.IDToken,
		ExpiresIn:    tokenResp.ExpiresIn,
		ExpiresAt:    expiresAt,
		Email:        email,
	}, nil
}

// getProxyURL retrieves proxy URL from repository
func (s *OpenAIRegisterService) getProxyURL(ctx context.Context, proxyID *int64) string {
	if proxyID == nil || s.proxyRepo == nil {
		return ""
	}
	proxy, err := s.proxyRepo.GetByID(ctx, *proxyID)
	if err != nil || proxy == nil {
		return ""
	}
	return proxy.URL()
}

// BuildAccountCredentials builds credentials map for account creation
func (s *OpenAIRegisterService) BuildAccountCredentials(result *SessionToRTResult) map[string]any {
	expiresAt := time.Unix(result.ExpiresAt, 0).Format(time.RFC3339)
	creds := map[string]any{
		"access_token":  result.AccessToken,
		"refresh_token": result.RefreshToken,
		"expires_at":    expiresAt,
	}
	if result.IDToken != "" {
		creds["id_token"] = result.IDToken
	}
	if result.Email != "" {
		creds["email"] = result.Email
	}
	return creds
}
