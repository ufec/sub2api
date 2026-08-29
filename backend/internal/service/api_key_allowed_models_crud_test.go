//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

// Key 级模型白名单（allowed_models）的 CRUD 语义：
//   - Create：请求里的列表规范化后入库；
//   - Update：nil = 不修改；非 nil = 覆盖（空切片 = 清空白名单，恢复不限制）；
//     只有真正修改时才登记 AllowedModels 列，避免并发计费路径的整行回写问题
//     （与 quota_used / usage_* 的"只声明要改的列"约定一致）。

func TestAPIKeyUpdate_AllowedModels(t *testing.T) {
	key := &APIKey{ID: 1, UserID: 7, Status: StatusActive}

	t.Run("nil request leaves field and column untouched", func(t *testing.T) {
		svc, repo := newUpdateFieldsAPIKeyService(key)
		updated, err := svc.Update(context.Background(), 1, 7, UpdateAPIKeyRequest{})
		require.NoError(t, err)
		require.False(t, repo.lastFields.AllowedModels, "nil request must not declare the allowed_models column")
		require.Nil(t, updated.AllowedModels)
	})

	t.Run("non-nil overwrites and declares column", func(t *testing.T) {
		svc, repo := newUpdateFieldsAPIKeyService(key)
		list := []string{"gpt-5.2", " claude-* ", "gpt-5.2"}
		updated, err := svc.Update(context.Background(), 1, 7, UpdateAPIKeyRequest{AllowedModels: &list})
		require.NoError(t, err)
		require.Len(t, repo.updateFields, 1)
		require.True(t, repo.updateFields[0].AllowedModels)
		require.Equal(t, []string{"gpt-5.2", "claude-*"}, updated.AllowedModels)
	})

	t.Run("empty slice clears whitelist", func(t *testing.T) {
		existing := []string{"gpt-5.2"}
		keyWithList := &APIKey{ID: 1, UserID: 7, Status: StatusActive, AllowedModels: existing}
		svc, repo := newUpdateFieldsAPIKeyService(keyWithList)
		empty := []string{}
		updated, err := svc.Update(context.Background(), 1, 7, UpdateAPIKeyRequest{AllowedModels: &empty})
		require.NoError(t, err)
		require.Len(t, repo.updateFields, 1)
		require.True(t, repo.updateFields[0].AllowedModels)
		require.Nil(t, updated.AllowedModels)
	})

	t.Run("invalid wildcard rejected", func(t *testing.T) {
		svc, repo := newUpdateFieldsAPIKeyService(key)
		list := []string{"claude-*-sonnet"}
		_, err := svc.Update(context.Background(), 1, 7, UpdateAPIKeyRequest{AllowedModels: &list})
		require.Error(t, err)
		require.False(t, repo.lastFields.AllowedModels)
	})
}

func TestAPIKeyCreate_AllowedModels(t *testing.T) {
	svc, _ := newUpdateFieldsAPIKeyService(&APIKey{})
	svc.cfg = &config.Config{}
	svc.userRepo = &userRepoStub{user: &User{ID: 7, Status: StatusActive}}
	created, err := svc.Create(context.Background(), 7, CreateAPIKeyRequest{
		Name:          "k",
		AllowedModels: []string{" gpt-5.2 ", "gpt-5.2", "claude-*"},
	})
	require.NoError(t, err)
	require.Equal(t, []string{"gpt-5.2", "claude-*"}, created.AllowedModels)
}

func TestAPIKeyCreate_InvalidAllowedModels(t *testing.T) {
	svc, _ := newUpdateFieldsAPIKeyService(&APIKey{})
	svc.cfg = &config.Config{}
	svc.userRepo = &userRepoStub{user: &User{ID: 7, Status: StatusActive}}
	_, err := svc.Create(context.Background(), 7, CreateAPIKeyRequest{
		Name:          "k",
		AllowedModels: []string{"a*b"},
	})
	require.Error(t, err)
}
