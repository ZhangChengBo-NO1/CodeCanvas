 package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type WriteFileTool struct {
	Root string // 工作区根目录
}

func (t *WriteFileTool) Name() string { return "write_file" }
func (t *WriteFileTool) Description() string {
	return "在工作区中创建或覆盖一个文件。如果目录不存在会自动创建。"
}
func (t *WriteFileTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "相对于工作区根目录的文件路径，例如 index.html 或 src/main.go",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "要写入的文件内容",
			},
		},
		"required": []string{"path", "content"},
	}
}

func (t *WriteFileTool) Execute(ctx context.Context, raw json.RawMessage) (string, error) {
	var in struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return "", fmt.Errorf("解析参数失败: %w", err)
	}

	// 拼装完整路径
	fullPath := filepath.Join(t.Root, in.Path)

	// 安全检查：确保 Clean 后的路径依然以 Root 开头
	cleanRoot := filepath.Clean(t.Root) + string(os.PathSeparator)
	cleanPath := filepath.Clean(fullPath)
	if !strings.HasPrefix(cleanPath, cleanRoot) && cleanPath != filepath.Clean(t.Root) {
		return "", fmt.Errorf("安全限制：无法在工作区外写文件 (%s)", in.Path)
	}

	// 确保父目录存在
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return "", fmt.Errorf("创建目录失败: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(fullPath, []byte(in.Content), 0644); err != nil {
		return "", fmt.Errorf("写入文件失败: %w", err)
	}

	return fmt.Sprintf("成功写入文件: %s (%d 字节)", in.Path, len(in.Content)), nil
}