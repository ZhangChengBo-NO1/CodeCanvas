package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type LoadSkillTool struct {
	SkillsDir string
}

func (t *LoadSkillTool) Name() string { return "load_skill" }
func (t *LoadSkillTool) Description() string {
	return "加载一个指定技能的详细使用说明。在使用某些特殊能力前，需先调用此工具阅读说明。"
}
func (t *LoadSkillTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{
				"type":        "string",
				"description": "技能名称，例如 web-dev",
			},
		},
		"required": []string{"name"},
	}
}

func (t *LoadSkillTool) Execute(ctx context.Context, raw json.RawMessage) (string, error) {
	var in struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return "", fmt.Errorf("解析参数失败: %w", err)
	}

	path := filepath.Join(t.SkillsDir, in.Name, "SKILL.md")
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Sprintf("错误：找不到技能 '%s'", in.Name), nil
	}

	return string(content), nil
}