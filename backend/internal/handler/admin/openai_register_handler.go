package admin

import (
	"github.com/gin-gonic/gin"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

// OpenAIRegisterHandler handles OpenAI registration APIs (admin only)
type OpenAIRegisterHandler struct {
	registerService *service.OpenAIRegisterService
	adminService    service.AdminService
}

// NewOpenAIRegisterHandler creates a new handler
func NewOpenAIRegisterHandler(
	registerService *service.OpenAIRegisterService,
	adminService service.AdminService,
) *OpenAIRegisterHandler {
	return &OpenAIRegisterHandler{
		registerService: registerService,
		adminService:    adminService,
	}
}

// AutoRegisterRequest request for auto registration
type AutoRegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	ProxyID  *int64 `json:"proxy_id"`
}

// SessionToRTRequest request for session to RT conversion
type SessionToRTRequest struct {
	SessionToken  string  `json:"session_token" binding:"required"`
	ProxyID       *int64  `json:"proxy_id"`
	CreateAccount bool    `json:"create_account"` // 是否自动创建账号
	Name          string  `json:"name"`           // 账号名称
	Concurrency   int     `json:"concurrency"`
	Priority      int     `json:"priority"`
	GroupIDs      []int64 `json:"group_ids"`
}

// AutoRegister handles automatic OpenAI account registration
// POST /api/v1/admin/openai/auto-register
func (h *OpenAIRegisterHandler) AutoRegister(c *gin.Context) {
	var req AutoRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	result, err := h.registerService.AutoRegister(c.Request.Context(), &service.AutoRegisterInput{
		Email:    req.Email,
		Password: req.Password,
		ProxyID:  req.ProxyID,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, result)
}

// SessionToRT converts session token to refresh token
// POST /api/v1/admin/openai/session-to-rt
func (h *OpenAIRegisterHandler) SessionToRT(c *gin.Context) {
	var req SessionToRTRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	result, err := h.registerService.SessionToRT(c.Request.Context(), &service.SessionToRTInput{
		SessionToken: req.SessionToken,
		ProxyID:      req.ProxyID,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	// 如果需要自动创建账号
	if req.CreateAccount && result.Success {
		credentials := h.registerService.BuildAccountCredentials(result)

		name := req.Name
		if name == "" && result.Email != "" {
			name = result.Email
		}
		if name == "" {
			name = "OpenAI Codex Account"
		}

		account, err := h.adminService.CreateAccount(c.Request.Context(), &service.CreateAccountInput{
			Name:        name,
			Platform:    "openai",
			Type:        "oauth",
			Credentials: credentials,
			ProxyID:     req.ProxyID,
			Concurrency: req.Concurrency,
			Priority:    req.Priority,
			GroupIDs:    req.GroupIDs,
		})
		if err != nil {
			response.ErrorFrom(c, err)
			return
		}

		response.Success(c, gin.H{
			"token_info": result,
			"account":    account,
		})
		return
	}

	response.Success(c, result)
}

// FetchEmailCodeRequest request for fetching verification code from email
type FetchEmailCodeRequest struct {
	ToEmail       string `json:"to_email" binding:"required,email"` // 收件人邮箱
	AdminEmail    string `json:"admin_email" binding:"required"`    // 邮局管理员邮箱
	AdminPassword string `json:"admin_password" binding:"required"` // 邮局管理员密码
	BaseURL       string `json:"base_url"`                          // 邮局API地址，默认 https://cloud-mail.enrun.ggff.net
}

// FetchEmailCode fetches OpenAI verification code from email via CloudMail API
// POST /api/v1/admin/openai/fetch-email-code
func (h *OpenAIRegisterHandler) FetchEmailCode(c *gin.Context) {
	var req FetchEmailCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	// 创建邮局客户端
	client := NewCloudMailClient(CloudMailConfig{
		BaseURL: req.BaseURL,
	})

	// 登录获取token
	if err := client.Login(c.Request.Context(), req.AdminEmail, req.AdminPassword); err != nil {
		response.BadRequest(c, "邮局登录失败: "+err.Error())
		return
	}

	// 获取验证码
	code, err := client.FetchOpenAICode(c.Request.Context(), req.ToEmail)
	if err != nil {
		response.BadRequest(c, "获取验证码失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"code": code,
	})
}

// CreateMailUserRequest 创建邮箱用户请求
type CreateMailUserRequest struct {
	Email         string `json:"email" binding:"required"`       // 新邮箱地址（不含域名）
	Domain        string `json:"domain" binding:"required"`      // 域名，如 enrun.ggff.net
	Password      string `json:"password"`                       // 邮箱密码，不填自动生成
	AdminEmail    string `json:"admin_email" binding:"required"` // 邮局管理员邮箱
	AdminPassword string `json:"admin_password" binding:"required"`
	BaseURL       string `json:"base_url"`
}

// CreateMailUser creates a new mail user
// POST /api/v1/admin/openai/create-mail-user
func (h *OpenAIRegisterHandler) CreateMailUser(c *gin.Context) {
	var req CreateMailUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	fullEmail := req.Email + "@" + req.Domain

	client := NewCloudMailClient(CloudMailConfig{
		BaseURL: req.BaseURL,
	})

	if err := client.Login(c.Request.Context(), req.AdminEmail, req.AdminPassword); err != nil {
		response.BadRequest(c, "邮局登录失败: "+err.Error())
		return
	}

	password := req.Password
	if password == "" {
		password = generateRandomPassword(12)
	}

	if err := client.AddUser(c.Request.Context(), fullEmail, password); err != nil {
		response.BadRequest(c, "创建邮箱用户失败: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"email":    fullEmail,
		"password": password,
	})
}

// generateRandomPassword 生成随机密码
func generateRandomPassword(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[i%len(charset)]
	}
	return string(b)
}
