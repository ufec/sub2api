//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// Key 级模型白名单（allowed_models）的判定语义：
//   - 空（nil / 空列表）= 不限制，所有模型放行（与账号 model_mapping 空值语义一致，
//     存量 Key 行为不变）；
//   - 非空 = 白名单精确匹配或末尾 "*" 通配符匹配（与 matchWildcard 语义一致）。

func TestAPIKeyAllowedModels(t *testing.T) {
	tests := []struct {
		name    string
		allowed []string
		model   string
		want    bool
	}{
		{
			name:    "nil list allows everything",
			allowed: nil,
			model:   "gpt-5.2",
			want:    true,
		},
		{
			name:    "empty list allows everything",
			allowed: []string{},
			model:   "gpt-5.2",
			want:    true,
		},
		{
			name:    "exact match",
			allowed: []string{"gpt-5.2", "claude-sonnet-4-5"},
			model:   "gpt-5.2",
			want:    true,
		},
		{
			name:    "exact miss",
			allowed: []string{"gpt-5.2", "claude-sonnet-4-5"},
			model:   "gpt-5.5",
			want:    false,
		},
		{
			name:    "trailing wildcard match",
			allowed: []string{"claude-*"},
			model:   "claude-sonnet-4-5",
			want:    true,
		},
		{
			name:    "trailing wildcard miss",
			allowed: []string{"claude-*"},
			model:   "gpt-5.2",
			want:    false,
		},
		{
			name:    "mid-star pattern does not match (trailing-only semantics)",
			allowed: []string{"claude-*-sonnet"},
			model:   "claude-3-sonnet",
			want:    false,
		},
		{
			name:    "whitespace entries are ignored",
			allowed: []string{"  ", "gpt-5.2"},
			model:   "gpt-5.2",
			want:    true,
		},
		{
			name:    "entries are matched trimmed",
			allowed: []string{" gpt-5.2 "},
			model:   "gpt-5.2",
			want:    true,
		},
		{
			name:    "empty model with non-empty list is rejected",
			allowed: []string{"gpt-5.2"},
			model:   "",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := &APIKey{AllowedModels: tt.allowed}
			require.Equal(t, tt.want, key.IsModelAllowed(tt.model))
		})
	}
}

func TestNormalizeAPIKeyAllowedModels(t *testing.T) {
	tests := []struct {
		name  string
		in    []string
		want  []string
		isNil bool
	}{
		{
			name:  "nil stays nil",
			in:    nil,
			isNil: true,
		},
		{
			name:  "empty stays nil",
			in:    []string{},
			isNil: true,
		},
		{
			name:  "only blanks collapse to nil",
			in:    []string{"  ", "\t"},
			isNil: true,
		},
		{
			name: "trims and dedupes preserving order",
			in:   []string{" gpt-5.2 ", "claude-*", "gpt-5.2", "  "},
			want: []string{"gpt-5.2", "claude-*"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeAPIKeyAllowedModels(tt.in)
			if tt.isNil {
				require.Empty(t, got)
				return
			}
			require.Equal(t, tt.want, got)
		})
	}
}

func TestValidateAPIKeyAllowedModels(t *testing.T) {
	tests := []struct {
		name    string
		in      []string
		wantErr bool
	}{
		{name: "nil ok", in: nil},
		{name: "plain model ok", in: []string{"gpt-5.2"}},
		{name: "trailing star ok", in: []string{"claude-*"}},
		{name: "mid star invalid", in: []string{"claude-*-sonnet"}, wantErr: true},
		{name: "lone star is valid trailing pattern", in: []string{"*"}},
		{name: "one invalid pattern fails", in: []string{"gpt-5.2", "a*b"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAPIKeyAllowedModels(tt.in)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
