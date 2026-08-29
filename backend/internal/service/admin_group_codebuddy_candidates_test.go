//go:build unit

package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/codebuddy"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/stretchr/testify/require"
)

type codeBuddyCandidatesAccountRepo struct {
	accountRepoStub
	accounts []Account
}

func (s *codeBuddyCandidatesAccountRepo) ListSchedulableByGroupID(_ context.Context, _ int64) ([]Account, error) {
	return s.accounts, nil
}

// fetchOnlyOnceTokenProvider 第一次成功返回 token，之后返回错误（模拟 token 提供失败回落）。
type fetchOnlyOnceTokenProvider struct {
	token    string
	failNth  int
	calls    int
	getCalls int
}

func (p *fetchOnlyOnceTokenProvider) GetAccessToken(_ context.Context, _ *Account) (string, error) {
	p.getCalls++
	p.calls++
	if p.calls <= p.failNth {
		return "", errors.New("token unavailable")
	}
	return p.token, nil
}

func TestAdminService_CodeBuddyModelsListCandidatesUseLiveUpstream(t *testing.T) {
	logger.Init(logger.InitOptions{Level: "error"})

	var configCalls int
	var gotAuthHeader string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		configCalls++
		gotAuthHeader = r.Header.Get("authorization")
		if got := r.Header.Get("authorization"); got != "Bearer live-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		// 顶层包装结构（data 信封由 FetchEnabledModels 解包）。
		// ParseModels 只读顶层 models（与同步上游路径一致），不含 agents[].models。
		_, _ = w.Write([]byte(`{"code":0,"msg":"ok","data":{"models":[{"id":"glm-6.0","name":"GLM-6"},{"id":"auto","name":"auto"},{"id":"kimi-k9","name":"Kimi"},{"id":"deepseek-v5","name":"DS"}],"agents":[{"models":["agent-only-model"]}]}}`))
	}))
	defer upstream.Close()

	accountRepo := &codeBuddyCandidatesAccountRepo{
		accounts: []Account{
			{
				ID:       1,
				Platform: PlatformCodeBuddy,
				Type:     AccountTypeOAuth,
				Credentials: map[string]any{
					"access_token":  "stale-token",
					"base_url":      upstream.URL,
					"model_mapping": map[string]any{"glm-5.2": "glm-5.2"},
				},
			},
		},
	}
	svc := &adminServiceImpl{
		accountRepo: accountRepo,
		groupRepo: &groupRepoStubForAdmin{
			getByIDByID: map[int64]*Group{
				8: {ID: 8, Platform: PlatformCodeBuddy},
			},
		},
	}
	provider := &fetchOnlyOnceTokenProvider{token: "live-token"}
	svc.SetCodeBuddyTokenProvider(provider)

	candidates, err := svc.GetGroupModelsListCandidates(context.Background(), 8, PlatformCodeBuddy)

	require.NoError(t, err)
	require.Equal(t, "Bearer live-token", gotAuthHeader)
	// 内置默认列表始终在候选里。
	for _, def := range codebuddy.DefaultModels() {
		require.Contains(t, candidates, def)
	}
	// model_mapping 键回落纳入候选。
	require.Contains(t, candidates, "glm-5.2")
	// 实时拉取结果（过滤 auto 占位项；ParseModels 不含 agents[].models，与同步路径一致）。
	require.Contains(t, candidates, "glm-6.0")
	require.Contains(t, candidates, "kimi-k9")
	require.Contains(t, candidates, "deepseek-v5")
	require.NotContains(t, candidates, "auto")
	require.NotContains(t, candidates, "agent-only-model")
	require.Equal(t, 1, configCalls)
	require.Equal(t, 1, provider.getCalls)
}

func TestAdminService_CodeBuddyModelsListCandidatesFallbackOnUpstreamError(t *testing.T) {
	logger.Init(logger.InitOptions{Level: "error"})

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// 上游业务错误码。
		_, _ = w.Write([]byte(`{"code":401,"msg":"unauthorized"}`))
	}))
	defer upstream.Close()

	accountRepo := &codeBuddyCandidatesAccountRepo{
		accounts: []Account{
			{
				ID:       1,
				Platform: PlatformCodeBuddy,
				Type:     AccountTypeOAuth,
				Credentials: map[string]any{
					"base_url":      upstream.URL,
					"model_mapping": map[string]any{"glm-5.2": "glm-5.2", "custom-model": "custom-model"},
				},
			},
		},
	}
	svc := &adminServiceImpl{
		accountRepo: accountRepo,
		groupRepo: &groupRepoStubForAdmin{
			getByIDByID: map[int64]*Group{
				8: {ID: 8, Platform: PlatformCodeBuddy},
			},
		},
	}
	provider := &fetchOnlyOnceTokenProvider{token: "live-token"}
	svc.SetCodeBuddyTokenProvider(provider)

	candidates, err := svc.GetGroupModelsListCandidates(context.Background(), 8, PlatformCodeBuddy)

	require.NoError(t, err)
	for _, def := range codebuddy.DefaultModels() {
		require.Contains(t, candidates, def)
	}
	require.Contains(t, candidates, "glm-5.2")
	require.Contains(t, candidates, "custom-model")
}

func TestAdminService_CodeBuddyModelsListCandidatesFallbackOnTokenFailure(t *testing.T) {
	logger.Init(logger.InitOptions{Level: "error"})

	accountRepo := &codeBuddyCandidatesAccountRepo{
		accounts: []Account{
			{
				ID:       1,
				Platform: PlatformCodeBuddy,
				Type:     AccountTypeOAuth,
				Credentials: map[string]any{
					"model_mapping": map[string]any{"glm-5.2": "glm-5.2", "custom-model": "custom-model"},
				},
			},
		},
	}
	svc := &adminServiceImpl{
		accountRepo: accountRepo,
		groupRepo: &groupRepoStubForAdmin{
			getByIDByID: map[int64]*Group{
				8: {ID: 8, Platform: PlatformCodeBuddy},
			},
		},
	}
	provider := &fetchOnlyOnceTokenProvider{failNth: 1}
	svc.SetCodeBuddyTokenProvider(provider)

	candidates, err := svc.GetGroupModelsListCandidates(context.Background(), 8, PlatformCodeBuddy)

	require.NoError(t, err)
	for _, def := range codebuddy.DefaultModels() {
		require.Contains(t, candidates, def)
	}
	require.Contains(t, candidates, "glm-5.2")
	require.Contains(t, candidates, "custom-model")
}

func TestAdminService_CodeBuddyModelsListCandidatesSkipsAPIKeyAccountsAndOtherPlatforms(t *testing.T) {
	logger.Init(logger.InitOptions{Level: "error"})

	accountRepo := &codeBuddyCandidatesAccountRepo{
		accounts: []Account{
			{
				ID:       1,
				Platform: PlatformCodeBuddy,
				Type:     AccountTypeAPIKey,
				Credentials: map[string]any{
					"model_mapping": map[string]any{"mapping-key": "mapping-key"},
				},
			},
			{
				ID:       2,
				Platform: PlatformAnthropic,
				Type:     AccountTypeAPIKey,
				Credentials: map[string]any{
					"model_mapping": map[string]any{"claude-custom": "claude-x"},
				},
			},
		},
	}
	svc := &adminServiceImpl{
		accountRepo: accountRepo,
		groupRepo: &groupRepoStubForAdmin{
			getByIDByID: map[int64]*Group{
				8: {ID: 8, Platform: PlatformCodeBuddy},
			},
		},
	}
	provider := &fetchOnlyOnceTokenProvider{token: "live-token"}
	svc.SetCodeBuddyTokenProvider(provider)

	candidates, err := svc.GetGroupModelsListCandidates(context.Background(), 8, PlatformCodeBuddy)

	require.NoError(t, err)
	// 非 OAuth CodeBuddy 账号：model_mapping 键仍纳入，但不会触发实时拉取。
	require.Contains(t, candidates, "mapping-key")
	// 非 CodeBuddy 平台账号不参与 codebuddy 平台候选。
	require.NotContains(t, candidates, "claude-custom")
	require.Equal(t, 0, provider.getCalls)
}

func TestAdminService_CodeBuddyCandidatesWithoutProviderStillReturnMappings(t *testing.T) {
	logger.Init(logger.InitOptions{Level: "error"})

	accountRepo := &codeBuddyCandidatesAccountRepo{
		accounts: []Account{
			{
				ID:       1,
				Platform: PlatformCodeBuddy,
				Type:     AccountTypeOAuth,
				Credentials: map[string]any{
					"model_mapping": map[string]any{"mapping-key": "mapping-key"},
				},
			},
		},
	}
	// 未注入 token provider（如单测直构 adminServiceImpl）。
	svc := &adminServiceImpl{
		accountRepo: accountRepo,
		groupRepo: &groupRepoStubForAdmin{
			getByIDByID: map[int64]*Group{
				8: {ID: 8, Platform: PlatformCodeBuddy},
			},
		},
	}

	candidates, err := svc.GetGroupModelsListCandidates(context.Background(), 8, PlatformCodeBuddy)

	require.NoError(t, err)
	require.Contains(t, candidates, "mapping-key")
}
