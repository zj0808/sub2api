package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"github.com/imroc/req/v3"
)

// CloudMailConfig 邮局配置
type CloudMailConfig struct {
	BaseURL       string `json:"base_url"`       // 邮局API地址，默认 https://cloud-mail.enrun.ggff.net
	AdminEmail    string `json:"admin_email"`    // 管理员邮箱
	AdminPassword string `json:"admin_password"` // 管理员密码
}

// CloudMailClient 邮局API客户端
type CloudMailClient struct {
	baseURL string
	client  *req.Client
	token   string
}

// EmailItem 邮件项
type EmailItem struct {
	EmailID    int    `json:"emailId"`
	SendEmail  string `json:"sendEmail"`
	SendName   string `json:"sendName"`
	Subject    string `json:"subject"`
	ToEmail    string `json:"toEmail"`
	ToName     string `json:"toName"`
	CreateTime string `json:"createTime"`
	Type       int    `json:"type"`
	Content    string `json:"content"`
	Text       string `json:"text"`
	IsDel      int    `json:"isDel"`
}

// NewCloudMailClient 创建邮局客户端
func NewCloudMailClient(cfg CloudMailConfig) *CloudMailClient {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = "https://cloud-mail.enrun.ggff.net"
	}

	return &CloudMailClient{
		baseURL: baseURL,
		client:  req.C().SetTimeout(30 * time.Second),
	}
}

// Login 登录获取token
func (c *CloudMailClient) Login(ctx context.Context, email, password string) error {
	var resp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Token string `json:"token"`
		} `json:"data"`
	}

	r, err := c.client.R().
		SetContext(ctx).
		SetBodyJsonMarshal(map[string]string{
			"email":    email,
			"password": password,
		}).
		SetSuccessResult(&resp).
		Post(c.baseURL + "/api/public/genToken")

	if err != nil {
		return fmt.Errorf("请求失败: %w", err)
	}
	if r.StatusCode != 200 || resp.Code != 200 {
		return fmt.Errorf("登录失败: %s", resp.Message)
	}

	c.token = resp.Data.Token
	return nil
}

// FetchEmails 获取邮件列表
func (c *CloudMailClient) FetchEmails(ctx context.Context, toEmail, sendEmail string) ([]EmailItem, error) {
	var resp struct {
		Code    int         `json:"code"`
		Message string      `json:"message"`
		Data    []EmailItem `json:"data"`
	}

	body := map[string]interface{}{
		"toEmail":   toEmail,
		"sendEmail": sendEmail,
		"timeSort":  "desc",
		"type":      0,
		"isDel":     0,
		"num":       1,
		"size":      10,
	}

	r, err := c.client.R().
		SetContext(ctx).
		SetHeader("Authorization", c.token).
		SetBodyJsonMarshal(body).
		SetSuccessResult(&resp).
		Post(c.baseURL + "/api/public/emailList")

	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}
	if r.StatusCode != 200 || resp.Code != 200 {
		return nil, fmt.Errorf("获取邮件失败: %s", resp.Message)
	}

	return resp.Data, nil
}

// AddUser 添加邮箱用户
func (c *CloudMailClient) AddUser(ctx context.Context, email, password string) error {
	var resp struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}

	body := map[string]interface{}{
		"list": []map[string]string{
			{"email": email, "password": password},
		},
	}

	r, err := c.client.R().
		SetContext(ctx).
		SetHeader("Authorization", c.token).
		SetBodyJsonMarshal(body).
		SetSuccessResult(&resp).
		Post(c.baseURL + "/api/public/addUser")

	if err != nil {
		return fmt.Errorf("请求失败: %w", err)
	}
	if r.StatusCode != 200 || resp.Code != 200 {
		return fmt.Errorf("添加用户失败: %s", resp.Message)
	}

	return nil
}

// FetchOpenAICode 从邮件中提取OpenAI验证码
func (c *CloudMailClient) FetchOpenAICode(ctx context.Context, toEmail string) (string, error) {
	emails, err := c.FetchEmails(ctx, toEmail, "%openai%")
	if err != nil {
		return "", err
	}

	if len(emails) == 0 {
		return "", fmt.Errorf("未找到OpenAI邮件")
	}

	codeRegex := regexp.MustCompile(`\b(\d{6})\b`)
	for _, email := range emails {
		content := email.Text
		if content == "" {
			content = email.Content
		}
		matches := codeRegex.FindStringSubmatch(content)
		if len(matches) > 1 {
			return matches[1], nil
		}
	}

	return "", fmt.Errorf("未在邮件中找到验证码")
}

