package service

import (
	"fmt"
	"strings"
)

// Key 级模型白名单（api_keys.allowed_models）。
//
// 语义约定：
//   - 空（nil / 空列表）= 不限制，放行所有模型。这与账号 model_mapping 的空值
//     语义一致，存量 Key 行为完全不变；
//   - 非空 = 白名单模式，请求模型必须精确命中某个条目，或命中末尾 "*" 通配符
//     （与账号 model_mapping 的 matchWildcard 语义一致，例如 "claude-*"）。
//
// 校验在写入口完成（NormalizeAPIKeyAllowedModels / ValidateAPIKeyAllowedModels），
// 热路径 IsModelAllowed 只做纯匹配，不清洗、不分配。

// NormalizeAPIKeyAllowedModels 清洗用户输入的 allowed_models：去空白、去重、
// 保持输入顺序；清洗后为空时返回 nil（= 不限制）。
func NormalizeAPIKeyAllowedModels(models []string) []string {
	if len(models) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(models))
	out := make([]string, 0, len(models))
	for _, model := range models {
		trimmed := strings.TrimSpace(model)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// ValidateAPIKeyAllowedModels 校验通配符格式：与前端 isValidWildcardPattern 一致，
// "*" 只能出现在末尾且仅一个。
func ValidateAPIKeyAllowedModels(models []string) error {
	for _, model := range models {
		starIndex := strings.Index(model, "*")
		if starIndex == -1 {
			continue
		}
		if starIndex != len(model)-1 || strings.LastIndex(model, "*") != starIndex {
			return fmt.Errorf("invalid allowed model pattern %q: '*' is only allowed as a trailing wildcard", model)
		}
	}
	return nil
}

// IsModelAllowed 判断请求模型是否被该 Key 的白名单放行。
// 空白名单 = 不限制；空模型名在白名单模式下视为拒绝。
func (k *APIKey) IsModelAllowed(requestedModel string) bool {
	if k == nil || len(k.AllowedModels) == 0 {
		return true
	}
	model := strings.TrimSpace(requestedModel)
	if model == "" {
		return false
	}
	for _, rawPattern := range k.AllowedModels {
		pattern := strings.TrimSpace(rawPattern)
		if pattern == "" {
			continue
		}
		if pattern == model || matchWildcard(pattern, model) {
			return true
		}
	}
	return false
}
