package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"CodeCanvas/internal/llm"
	"CodeCanvas/internal/loop"
	"CodeCanvas/internal/mcp"
	"CodeCanvas/internal/memory"
	"CodeCanvas/internal/tools"
	"CodeCanvas/tui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// 1. 初始化 LLM
	client, err := llm.New()
	if err != nil {
		log.Fatal(err)
	}

	// 2. 工作区
	workspace := "./workspace"
	if err = os.MkdirAll(workspace, 0755); err != nil {
		log.Fatal("创建工作区失败：%v", err)
	}
	absWorkspace, _ := filepath.Abs(workspace)
	fmt.Printf("📁 工作区: %s\n", absWorkspace)

	// 3. 初始化 Agent 及工具
	agent := loop.New(client, 15)
	agent.RegisterTool(&tools.EchoTool{})
	agent.RegisterTool(&tools.WriteFileTool{Root: absWorkspace})
	agent.RegisterTool(tools.NewShellTool(absWorkspace))
	previewServer := &tools.PreviewServer{Root: absWorkspace}
	agent.RegisterTool(previewServer)
	agent.RegisterTool(&tools.LoadSkillTool{SkillsDir: "skills"})

		// 4. 接入 MCP
	fmt.Println("⏳ 正在连接 MCP Servers (npx 启动较慢，请稍候)...")
	mcpTools, closeMCP, _ := mcp.LoadFromConfig(context.Background(), "mcp.json")
	for _, t := range mcpTools {
		agent.RegisterTool(t)
	}
	fmt.Println("✅ MCP 连接完成，正在启动 TUI...")

	// 5. Memory & Session
	store := memory.NewStore("memory/entries.jsonl")
	recentMemory := store.LoadRecent(3)
	sysPrompt := loop.DefaultSystemPrompt(workspace)
	if recentMemory != "" {
		sysPrompt += "\n\n## 历史记忆\n以下是你在过去与用户交互中记住的关键信息：\n" + recentMemory
	}
	session := loop.NewSession(sysPrompt)

	// 6. 启动 TUI
	p := tea.NewProgram(
		tui.NewModel(agent, session),
		tea.WithAltScreen(),       // 使用全屏模式
		tea.WithMouseCellMotion(), // 支持鼠标滚轮
	)

	if _, err := p.Run(); err != nil {
		fmt.Printf("TUI 运行出错: %v", err)
		os.Exit(1)
	}

	// 7. 退出时的清理与保存记忆
	fmt.Println("\n正在保存记忆并清理资源...")
	
	// 保存记忆
	text := memory.DialogueToText(session.Messages)
	if text != "" {
		summary, err := client.Summarize(context.Background(), text)
		if err == nil && summary != "" {
			store.Append(summary)
			fmt.Println("✅ 记忆已保存:", summary)
		}
	}

	// 关闭 MCP
	for _, closeFn := range closeMCP {
		_ = closeFn()
	}
	// 关闭预览服务器
	_ = previewServer.Shutdown(context.Background())

	fmt.Println("👋 再见！")
}