package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"

	"CodeCanvas/internal/loop"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ---------------- 配置解析部分 ----------------

type ServerConfig struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

type MCPConfig struct {
	MCPServers map[string]ServerConfig `json:"mcpServers"`
}

// ---------------- MCP 管理器核心逻辑 ----------------

type Manager struct {
	session *mcp.ClientSession
	tools   []loop.Tool
}

// NewManager 连接单个 MCP Server 并发现工具
func NewManager(ctx context.Context, command string, args ...string) (*Manager, error) {
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "CodeCanvas-MCP-Client",
		Version: "v1.0.0",
	}, nil)

	transport := &mcp.CommandTransport{
		Command: exec.CommandContext(ctx, command, args...),
	}

	session, err := client.Connect(ctx, transport, nil)
	if err != nil {
		return nil, fmt.Errorf("连接 MCP Server 失败: %w", err)
	}

	listed, err := session.ListTools(ctx, nil)
	if err != nil {
		session.Close()
		return nil, fmt.Errorf("获取工具列表失败: %w", err)
	}

	var adapted []loop.Tool
	for _, t := range listed.Tools {
		adapted = append(adapted, &mcpToolAdapter{
			session: session,
			name:    t.Name,
			desc:    t.Description,
			schema:  t.InputSchema,
		})
	}

	return &Manager{session: session, tools: adapted}, nil
}

func (m *Manager) Tools() []loop.Tool { return m.tools }
func (m *Manager) Close() error {
	if m.session != nil {
		return m.session.Close()
	}
	return nil
}

// LoadFromConfig 从 mcp.json 读取配置并连接所有的 Server
func LoadFromConfig(ctx context.Context, configPath string) ([]loop.Tool, []func() error, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil // 文件不存在直接跳过
		}
		return nil, nil, fmt.Errorf("读取配置失败: %w", err)
	}

	var cfg MCPConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, nil, fmt.Errorf("解析配置失败: %w", err)
	}

	var allTools []loop.Tool
	var closeFuncs []func() error

	for name, sc := range cfg.MCPServers {
		mgr, err := NewManager(ctx, sc.Command, sc.Args...)
		if err != nil {
			log.Printf("⚠️ MCP Server %q 连接失败（已跳过）: %v", name, err)
			continue
		}
		allTools = append(allTools, mgr.Tools()...)
		closeFuncs = append(closeFuncs, mgr.Close)
		log.Printf("✅ MCP Server %q 已连接，发现 %d 个工具", name, len(mgr.Tools()))
	}

	return allTools, closeFuncs, nil
}

// ---------------- 工具适配器 ----------------

type mcpToolAdapter struct {
	session *mcp.ClientSession
	name    string
	desc    string
	schema  any
}

func (a *mcpToolAdapter) Name() string        { return a.name }
func (a *mcpToolAdapter) Description() string { return a.desc }

func (a *mcpToolAdapter) Parameters() map[string]any {
	if m, ok := a.schema.(map[string]any); ok {
		return m
	}
	return map[string]any{"type": "object", "properties": map[string]any{}}
}

func (a *mcpToolAdapter) Execute(ctx context.Context, raw json.RawMessage) (string, error) {
	var args map[string]any
	if err := json.Unmarshal(raw, &args); err != nil {
		return "", fmt.Errorf("解析 MCP 工具参数失败: %w", err)
	}

	result, err := a.session.CallTool(ctx, &mcp.CallToolParams{
		Name:      a.name,
		Arguments: args,
	})
	if err != nil {
		return "", fmt.Errorf("MCP 工具调用失败: %w", err)
	}
	if result.IsError {
		return "MCP 工具执行返回错误", nil
	}

	var out string
	for _, c := range result.Content {
		if text, ok := c.(*mcp.TextContent); ok {
			out += text.Text
		}
	}
	if out == "" {
		return "MCP 工具执行成功（无文本输出）", nil
	}
	return out, nil
}