package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/cloudmail"
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

// MailConfig 邮局配置
type MailConfig struct {
	BaseURL       string `json:"base_url"`
	AdminEmail    string `json:"admin_email"`
	AdminPassword string `json:"admin_password"`
}

// AutoRegisterInput input for auto registration
type AutoRegisterInput struct {
	Email      string      `json:"email"`
	Password   string      `json:"password"`
	ProxyID    *int64      `json:"proxy_id,omitempty"`
	MailConfig *MailConfig `json:"mail_config,omitempty"` // 邮局配置，用于自动获取验证码
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
// 完整流程: 注册 → 轮询邮箱获取验证码 → 验证邮箱 → 登录获取RT
func (s *OpenAIRegisterService) AutoRegister(ctx context.Context, input *AutoRegisterInput) (*AutoRegisterResult, error) {
	proxyURL := s.getProxyURL(ctx, input.ProxyID)

	log.Printf("[OpenAIRegister] Starting auto registration for: %s", input.Email)

	client := openai.NewRegisterClient(proxyURL)

	// Step 1: Register via chatgpt.com entry (doesn't verify sentinel)
	log.Printf("[OpenAIRegister] Step 1: Calling register API...")
	regResp, err := client.Register(ctx, &openai.RegisterRequest{
		Email:    input.Email,
		Password: input.Password,
	})
	if err != nil {
		return &AutoRegisterResult{
			Success: false,
			Email:   input.Email,
			Error:   fmt.Sprintf("注册失败: %v", err),
		}, err
	}
	log.Printf("[OpenAIRegister] Registration API called successfully for: %s", input.Email)

	// 如果没有提供邮局配置，只完成注册步骤
	if input.MailConfig == nil {
		log.Printf("[OpenAIRegister] No mail config provided, returning after registration")
		return &AutoRegisterResult{
			Success: regResp.Success,
			Email:   input.Email,
			UserID:  regResp.UserID,
		}, nil
	}

	// Step 2: 轮询邮箱获取验证码
	log.Printf("[OpenAIRegister] Step 2: Polling for verification code...")
	code, err := s.pollForVerificationCode(ctx, input.Email, input.MailConfig)
	if err != nil {
		return &AutoRegisterResult{
			Success: false,
			Email:   input.Email,
			Error:   fmt.Sprintf("获取验证码失败: %v", err),
		}, err
	}
	log.Printf("[OpenAIRegister] Got verification code: %s", code)

	// Step 3: 验证邮箱
	log.Printf("[OpenAIRegister] Step 3: Verifying email...")
	_, err = client.VerifyEmail(ctx, input.Email, code)
	if err != nil {
		return &AutoRegisterResult{
			Success: false,
			Email:   input.Email,
			Error:   fmt.Sprintf("验证邮箱失败: %v", err),
		}, err
	}
	log.Printf("[OpenAIRegister] Email verified successfully")

	// Step 4: 登录获取 refresh_token
	log.Printf("[OpenAIRegister] Step 4: Logging in to get refresh token...")
	loginResp, err := client.LoginWithPassword(ctx, input.Email, input.Password)
	if err != nil {
		return &AutoRegisterResult{
			Success: false,
			Email:   input.Email,
			Error:   fmt.Sprintf("登录失败: %v", err),
		}, err
	}
	log.Printf("[OpenAIRegister] Login successful, got refresh token")

	expiresAt := time.Now().Unix() + loginResp.ExpiresIn

	return &AutoRegisterResult{
		Success:      true,
		Email:        input.Email,
		UserID:       regResp.UserID,
		AccessToken:  loginResp.AccessToken,
		RefreshToken: loginResp.RefreshToken,
		ExpiresAt:    expiresAt,
	}, nil
}

// pollForVerificationCode 轮询邮箱获取验证码
func (s *OpenAIRegisterService) pollForVerificationCode(ctx context.Context, email string, mailCfg *MailConfig) (string, error) {
	// 最多轮询 60 秒，每 5 秒检查一次
	maxAttempts := 12
	interval := 5 * time.Second

	for i := 0; i < maxAttempts; i++ {
		code, err := s.fetchCodeFromMail(ctx, email, mailCfg)
		if err == nil && code != "" {
			return code, nil
		}

		log.Printf("[OpenAIRegister] Attempt %d: No code yet, waiting %v...", i+1, interval)

		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(interval):
			// 继续下一次尝试
		}
	}

	return "", fmt.Errorf("超时：60秒内未收到验证码")
}

// fetchCodeFromMail 从邮局获取验证码
func (s *OpenAIRegisterService) fetchCodeFromMail(ctx context.Context, email string, mailCfg *MailConfig) (string, error) {
	client := cloudmail.NewClient(cloudmail.Config{
		BaseURL:       mailCfg.BaseURL,
		AdminEmail:    mailCfg.AdminEmail,
		AdminPassword: mailCfg.AdminPassword,
	})

	// 登录邮局
	if err := client.Login(ctx, mailCfg.AdminEmail, mailCfg.AdminPassword); err != nil {
		return "", fmt.Errorf("邮局登录失败: %w", err)
	}

	// 获取验证码
	code, err := client.FetchOpenAICode(ctx, email)
	if err != nil {
		return "", err
	}

	return code, nil
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
