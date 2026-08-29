package handler

import "github.com/Wei-Shaw/sub2api/internal/service"

// allowedModelsFilterKey 测试替身用的最小 Key 结构（仅白名单字段参与过滤逻辑）。
type allowedModelsFilterKey = service.APIKey

// filterModelsByKeyWhitelist 按 Key 级模型白名单过滤 /v1/models 列表。
// 空白名单（未配置）原样返回；非空时保留精确或末尾通配符命中的模型。
// 无命中时返回空列表（客户端看不到任何可用模型，与白名单语义一致）。
func filterModelsByKeyWhitelist(models []string, key *service.APIKey) []string {
	if key == nil || len(key.AllowedModels) == 0 {
		return models
	}
	filtered := make([]string, 0, len(models))
	for _, model := range models {
		if key.IsModelAllowed(model) {
			filtered = append(filtered, model)
		}
	}
	return filtered
}
