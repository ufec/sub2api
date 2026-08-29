package middleware

import (
	"bytes"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

// RequireAllowedModels 校验 Key 级模型白名单（api_keys.allowed_models）。
//
// 语义：
//   - Key 白名单为空（未配置）= 不限制，直接放行；存量 Key 行为不变。
//   - 白名单非空：从请求体 JSON 读取 model 字段做匹配（精确 / 末尾 "*" 通配符，
//     与账号 model_mapping 的 matchWildcard 语义一致）。命中放行；不命中返回
//     404 model_not_found —— 与调度层 model_not_found 的客户端语义一致，客户端
//     会把该模型视为不存在，不会向用户暴露白名单机制。
//   - 请求体缺失或 model 字段缺省时不在此拦截：Gemini 的 model 在 URL path 中、
//     multipart 表单在后续 handler 读取，无法判定时交由后续必填校验兜底。
//     （关键路径已覆盖：/v1/messages、/v1/chat/completions、/v1/responses、
//     /v1/embeddings 等 JSON 请求。）
//
// 错误响应按协议分流：path 含 /messages 走 Anthropic 错误格式，其余走 OpenAI
// 格式。与 RequireGroupAssignment 一样挂在 apiKeyAuth 之后、handler 之前。
func RequireAllowedModels() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey, ok := GetAPIKeyFromContext(c)
		if !ok || apiKey == nil || apiKey.IsModelAllowed("") {
			// IsModelAllowed("") 对空白名单恒为 true，此处仅借道短路空名单。
			c.Next()
			return
		}

		if c.Request == nil || c.Request.Method == http.MethodGet || c.Request.Body == nil {
			c.Next()
			return
		}

		body, err := httputil.ReadRequestBodyWithPrealloc(c.Request)
		if err != nil {
			writeAllowedModelsError(c, http.StatusBadRequest, "invalid_request_error", "Failed to read request body")
			return
		}

		model := strings.TrimSpace(gjson.GetBytes(body, "model").String())
		if model == "" {
			// model 缺省：无法判定，交由后续 handler 必填校验。
			resetRequestBodyForAllowedModels(c, body)
			c.Next()
			return
		}
		if apiKey.IsModelAllowed(model) {
			resetRequestBodyForAllowedModels(c, body)
			c.Next()
			return
		}

		MarkIngressRejected(c, IngressRejectModelNotAllowed)
		writeAllowedModelsError(c, http.StatusNotFound, "model_not_found",
			"Model '"+model+"' is not available for this API key")
	}
}

// writeAllowedModelsError 按请求协议输出错误：/v1/messages 系列（Anthropic 协议）
// 使用 type.error 包裹；其余（OpenAI 兼容）使用 error.type 包裹。
func writeAllowedModelsError(c *gin.Context, status int, errType, message string) {
	if strings.Contains(c.Request.URL.Path, "/messages") {
		c.JSON(status, gin.H{
			"type":  "error",
			"error": gin.H{"type": errType, "message": message},
		})
	} else {
		c.JSON(status, gin.H{
			"error": gin.H{
				"type":    errType,
				"message": message,
				"param":   nil,
				"code":    nil,
			},
		})
	}
	c.Abort()
}

// resetRequestBodyForAllowedModels 把已读出的请求体放回，供后续 handler / 中间件
// 读取（与 compositeTargetPlatformMiddleware 的 resetRequestBody 语义一致）。
func resetRequestBodyForAllowedModels(c *gin.Context, body []byte) {
	c.Request.Body = io.NopCloser(bytes.NewReader(body))
	c.Request.ContentLength = int64(len(body))
	c.Request.Header.Set("Content-Length", strconv.Itoa(len(body)))
}
