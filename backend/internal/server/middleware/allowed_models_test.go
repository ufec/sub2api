//go:build unit

package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// RequireAllowedModels 校验 Key 级模型白名单（api_keys.allowed_models）：
//   - 空白名单 = 不限制，直接放行（含无 model 的请求，如 Gemini model-in-path 由
//     后续 handler 校验）；
//   - 非空白名单：JSON body 的 model 字段必须命中；不命中返回 404 model_not_found
//     （与调度层 model_not_found 语义一致），并标记 ingress reject。
//
// 错误响应按协议分流：messages 走 Anthropic 格式，其余走 OpenAI 格式。
func newAllowedModelsRouter(apiKey *service.APIKey, cap *capture) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(ContextKeyAPIKey), apiKey)
		c.Next()
		cap.reason, cap.rejected = GetIngressRejectReason(c)
	})
	router.Use(RequireAllowedModels())
	router.POST("/v1/messages", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	router.POST("/v1/chat/completions", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	router.POST("/v1/responses", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	return router
}

type capture struct {
	rejected bool
	reason   IngressRejectReason
}

func do(router *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	return w
}

func TestRequireAllowedModelsEmptyWhitelistPasses(t *testing.T) {
	apiKey := &service.APIKey{ID: 1, Status: service.StatusActive}
	router := newAllowedModelsRouter(apiKey, &capture{})

	for _, body := range []string{"", `{"model":"gpt-5.2"}`, `{"model":""}`} {
		w := do(router, http.MethodPost, "/v1/messages", body)
		require.Equal(t, http.StatusOK, w.Code, "body: %s", body)
	}
}

func TestRequireAllowedModelsBlocksDisallowedModel(t *testing.T) {
	apiKey := &service.APIKey{
		ID:            1,
		Status:        service.StatusActive,
		AllowedModels: []string{"claude-sonnet-4-5", "claude-*"},
	}
	cap := &capture{}
	router := newAllowedModelsRouter(apiKey, cap)

	t.Run("anthropic protocol error shape and ingress reject", func(t *testing.T) {
		w := do(router, http.MethodPost, "/v1/messages", `{"model":"gpt-5.2"}`)
		require.Equal(t, http.StatusNotFound, w.Code)
		require.Contains(t, w.Body.String(), "model_not_found")
		require.Contains(t, w.Body.String(), "gpt-5.2")
		require.True(t, cap.rejected)
		require.Equal(t, IngressRejectModelNotAllowed, cap.reason)
	})

	t.Run("openai protocol error shape", func(t *testing.T) {
		w := do(router, http.MethodPost, "/v1/chat/completions", `{"model":"gpt-5.2","messages":[]}`)
		require.Equal(t, http.StatusNotFound, w.Code)
		require.Contains(t, w.Body.String(), "model_not_found")
	})

	t.Run("whitelisted model passes", func(t *testing.T) {
		for _, body := range []string{`{"model":"claude-sonnet-4-5"}`, `{"model":"claude-opus-4"}`} {
			w := do(router, http.MethodPost, "/v1/messages", body)
			require.Equal(t, http.StatusOK, w.Code, "body: %s", body)
		}
	})
}

func TestRequireAllowedModelsWildcardOnly(t *testing.T) {
	apiKey := &service.APIKey{
		ID:            1,
		Status:        service.StatusActive,
		AllowedModels: []string{"gpt-*"},
	}
	router := newAllowedModelsRouter(apiKey, &capture{})

	w := do(router, http.MethodPost, "/v1/chat/completions", `{"model":"gpt-5.2"}`)
	require.Equal(t, http.StatusOK, w.Code)

	w = do(router, http.MethodPost, "/v1/chat/completions", `{"model":"claude-sonnet-4-5"}`)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestRequireAllowedModelsMissingBodyOrModel(t *testing.T) {
	apiKey := &service.APIKey{
		ID:            1,
		Status:        service.StatusActive,
		AllowedModels: []string{"gpt-5.2"},
	}
	router := newAllowedModelsRouter(apiKey, &capture{})

	// body 缺失 / model 缺省：无法判定，交由后续 handler 的必填校验兜底，不在此拦截
	for _, body := range []string{"", `{"messages":[]}`} {
		w := do(router, http.MethodPost, "/v1/chat/completions", body)
		require.Equal(t, http.StatusOK, w.Code, "body: %s", body)
	}
}
