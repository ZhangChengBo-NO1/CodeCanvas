package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type ShellTool struct {
	Root        string
	AllowedCmds map[string]bool
	Timeout     time.Duration
}

func NewShellTool(root string) *ShellTool {
	return &ShellTool{
		Root: root,
		// 只允许这些命令，防止 Agent 乱搞
		AllowedCmds: map[string]bool{
			"ls": true, "cat": true, "pwd": true, "echo": true,
			"mkdir": true, "touch": true,
			"go": true, "python": true, "python3": true,
			"node": true, "npm": true, "npx": true,
			"curl": true,
		},
		Timeout: 30 * time.Second,
	}
}

func (t *ShellTool) Name() string { return "shell_exec" }
func (t *ShellTool) Description() string {
	return "在工作区中执行 Shell 命令，返回标准输出和标准错误。仅支持常见的开发命令。"
}
func (t *ShellTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"command": map[string]any{
				"type":        "string",
				"description": "要执行的命令，例如 'ls -la' 或 'python3 main.py'",
			},
		},
		"required": []string{"command"},
	}
}

func (t *ShellTool) Execute(ctx context.Context, raw json.RawMessage) (string, error) {
	var in struct {
		Command string `json:"command"`
	}
	if err := json.Unmarshal(raw, &in); err != nil {
		return "", fmt.Errorf("解析参数失败: %w", err)
	}

	args := strings.Fields(in.Command)
	if len(args) == 0 {
		return "", fmt.Errorf("命令不能为空")
	}

	// 白名单校验
	baseCmd := args[0]
	if !t.AllowedCmds[baseCmd] {
		return fmt.Sprintf("错误：命令 %q 不在白名单中，拒绝执行", baseCmd), nil
	}

	// 创建带超时的 context
	execCtx, cancel := context.WithTimeout(ctx, t.Timeout)
	defer cancel()

	// 构造命令
	cmd := exec.CommandContext(execCtx, args[0], args[1:]...)
	cmd.Dir = t.Root // 限定执行目录

	// 执行并获取输出
	out, err := cmd.CombinedOutput()
	output := string(out)

	// 截断过长的输出，保护上下文（最多保留 4000 字符）
	if len(output) > 4000 {
		output = output[:4000] + "\n... [输出被截断]"
	}

	if err != nil {
		if execCtx.Err() == context.DeadlineExceeded {
			return fmt.Sprintf("命令超时（%v）\n输出：\n%s", t.Timeout, output), nil
		}
		return fmt.Sprintf("命令执行失败 (退出码非0)：%v\n输出：\n%s", err, output), nil
	}

	if output == "" {
		return "命令执行成功（无输出）", nil
	}
	return output, nil
}