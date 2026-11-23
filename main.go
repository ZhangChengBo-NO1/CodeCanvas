package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"CodeCanvas/internal/llm"
	"CodeCanvas/internal/loop"
	"CodeCanvas/internal/mcp"
	"CodeCanvas/internal/memory"
	"CodeCanvas/internal/tools"
)

func main() {
	client, err := llm.New()
	if err != nil {
		log.Fatal(err)
	}

	// 1. 工作区
	workspace, err := os.MkdirTemp("", "codecanvas-ws-*")
	if err != nil {
		log.Fatalf("创建临时工作区失败: %v", err)
	}
	fmt.Printf("📁 工作区已创建: %s\n\n", workspace)

	// 2. 初始化 Agent
	agent := loop.New(client, 15)

	// 3. 注册本地工具
	agent.RegisterTool(&tools.EchoTool{})
	agent.RegisterTool(&tools.WriteFileTool{Root: workspace})
	agent.RegisterTool(tools.NewShellTool(workspace))
	previewServer := &tools.PreviewServer{Root: workspace} // 保留引用用于退出时清理
	agent.RegisterTool(previewServer)
	agent.RegisterTool(&tools.LoadSkillTool{SkillsDir: "skills"})

	// 4. 接入 MCP
	mcpTools, closeMCP, _ := mcp.LoadFromConfig(context.Background(), "mcp.json")
	for _, t := range mcpTools {
		agent.RegisterTool(t)
	}

	// 5. 初始化长记忆
	store := memory.NewStore("memory/entries.jsonl")
	recentMemory := store.LoadRecent(3)
	sysPrompt := loop.DefaultSystemPrompt(workspace)
	if recentMemory != "" {
		sysPrompt += "\n\n## 历史记忆\n以下是你在过去与用户交互中记住的关键信息：\n" + recentMemory
		fmt.Println("🧠 已加载历史记忆")
	}
	session := loop.NewSession(sysPrompt)

	// 6. 交互循环
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("\n你> ")
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// --- 退出逻辑 ---
		if line == "/exit" || line == "/quit" {
			fmt.Println("\n正在提炼本次对话的记忆...")
			text := memory.DialogueToText(session.Messages)
			if text != "" {
				summary, err := client.Summarize(context.Background(), text)
				if err == nil && summary != "" {
					store.Append(summary)
					fmt.Println("✅ 记忆已保存:", summary)
				} else {
					fmt.Println("⚠️ 记忆保存失败:", err)
				}
			}

			fmt.Println("正在清理后台资源...")
			// 清理 MCP 子进程
			for _, closeFn := range closeMCP {
				_ = closeFn()
			}
			// 清理预览服务器
			if previewServer.URL != "" {
				_ = previewServer.Shutdown(context.Background())
			}
			fmt.Println("👋 再见！")
			break
		}

		// --- 正常运行逻辑 ---
		reply, err := agent.Run(context.Background(), session, line)
		if err != nil {
			fmt.Println("❌ 错误：", err)
			continue
		}
		fmt.Println("Agent>", reply)
	}
}