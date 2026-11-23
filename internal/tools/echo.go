// internal/tools/echo.go
package tools

import (
	"context"
	"encoding/json"
	"fmt"
)

type EchoTool struct{}

func (t *EchoTool) Name() string        { return "echo" }
func (t *EchoTool) Description() string { return "把输入文本原样返回，用于测试 Agent Loop" }
func (t *EchoTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"text": map[string]any{
				"type":        "string",
				"description": "要返回的文本",
			},
		},
		"required": []string{"text"},
	}
}
func (t *EchoTool) Execute(ctx context.Context, raw json.RawMessage) (string, error) {
	var in struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return "", fmt.Errorf("解析参数失败: %w", err)
	}
	return in.Text, nil
}