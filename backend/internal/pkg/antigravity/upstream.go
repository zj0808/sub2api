// Package antigravity provides upstream utilities for Antigravity API.
package antigravity

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/tidwall/gjson"
)

// BaseURLs 返回 Antigravity API 端点回退顺序（sandbox-daily → daily → prod）
// 参考 CLIProxyAPI 的实现，提高可用性
func BaseURLs() []string {
	return []string{
		BaseURLSandboxDaily,
		BaseURLDaily,
		BaseURL, // prod
	}
}

// NewAPIRequestWithBaseURL 创建 Antigravity API 请求（指定 Base URL）
func NewAPIRequestWithBaseURL(ctx context.Context, baseURL, action, accessToken string, body []byte) (*http.Request, error) {
	apiURL := fmt.Sprintf("%s/v1internal:%s", strings.TrimSuffix(baseURL, "/"), action)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("User-Agent", UserAgent)
	return req, nil
}

// GenerateStableSessionID 基于请求内容生成稳定的 sessionId
// 参考 CLIProxyAPI 的实现：从第一条 user 消息的文本内容生成 hash
// 这样相同的对话上下文会产生相同的 sessionId
func GenerateStableSessionID(payload []byte) string {
	contents := gjson.GetBytes(payload, "request.contents")
	if contents.IsArray() {
		for _, content := range contents.Array() {
			if content.Get("role").String() == "user" {
				text := content.Get("parts.0.text").String()
				if text != "" {
					h := sha256.Sum256([]byte(text))
					n := int64(binary.BigEndian.Uint64(h[:8])) & 0x7FFFFFFFFFFFFFFF
					return "-" + strconv.FormatInt(n, 10)
				}
			}
		}
	}
	// 回退：生成随机 sessionId
	return GenerateRandomSessionID()
}

// GenerateRandomSessionID 生成随机 sessionId
func GenerateRandomSessionID() string {
	randBytes, err := GenerateRandomBytes(8)
	if err != nil {
		return "-1234567890123456789" // 回退值
	}
	n := int64(binary.BigEndian.Uint64(randBytes)) & 0x7FFFFFFFFFFFFFFF
	return "-" + strconv.FormatInt(n, 10)
}

// IsRetryableStatusCode 判断是否应该重试到下一个 Base URL
// 参考 CLIProxyAPI：连接错误、429 应该尝试下一个端点
func IsRetryableStatusCode(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests // 429
}

// IsRetryableError 判断错误是否可以重试到下一个 Base URL
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}
	// 网络错误通常可以重试
	errStr := err.Error()
	return strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "no such host") ||
		strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "EOF")
}

