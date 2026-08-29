//go:build unit

package handler

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// /v1/models 按 Key 白名单过滤：空白名单原样返回；非空白名单保留命中的模型，
// 顺序与去重同 filterModelsByCustomList 语义一致。
func TestFilterModelsByKeyWhitelist(t *testing.T) {
	tests := []struct {
		name    string
		models  []string
		allowed []string
		want    []string
	}{
		{
			name:    "nil whitelist returns original",
			models:  []string{"gpt-5.2", "claude-opus-4"},
			allowed: nil,
			want:    []string{"gpt-5.2", "claude-opus-4"},
		},
		{
			name:    "empty whitelist returns original",
			models:  []string{"gpt-5.2", "claude-opus-4"},
			allowed: []string{},
			want:    []string{"gpt-5.2", "claude-opus-4"},
		},
		{
			name:    "exact filter",
			models:  []string{"gpt-5.2", "gpt-5.5", "claude-opus-4"},
			allowed: []string{"gpt-5.2", "claude-opus-4"},
			want:    []string{"gpt-5.2", "claude-opus-4"},
		},
		{
			name:    "wildcard filter",
			models:  []string{"gpt-5.2", "gpt-5.5", "claude-opus-4"},
			allowed: []string{"gpt-*"},
			want:    []string{"gpt-5.2", "gpt-5.5"},
		},
		{
			name:    "no match yields empty list",
			models:  []string{"gpt-5.2"},
			allowed: []string{"claude-*"},
			want:    []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := &allowedModelsFilterKey{AllowedModels: tt.allowed}
			got := filterModelsByKeyWhitelist(tt.models, key)
			require.Equal(t, tt.want, got)
		})
	}
}
