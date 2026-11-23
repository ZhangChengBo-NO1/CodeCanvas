// internal/loop/prompt.go
package loop

import (
	"strings"
	"fmt"
)

// DefaultSystemPrompt 是这个 Agent 的基础人格和工具使用协议。
// 后续 Memory 摘要、Skill 目录会拼接在它后面。
// 修改 DefaultSystemPrompt 接收工作区路径
func DefaultSystemPrompt(workspaceDir string) string {
	return fmt.Sprintf(`你是 CodeCanvas，一个运行在用户终端中的通用 Agent，以代码能力见长。

## 工作环境
- 你的当前工作区（Workspace）是: %s
- 你所有的文件读写和 Shell 命令执行都会且只能在这个目录下进行。
- 在操作文件时，请始终使用相对路径（例如 'index.html'），不要使用绝对路径。

## 行为准则
- 需要执行具体操作时，调用工具，不要假装执行。
- 工具返回错误时，阅读错误信息并尝试修复。
- 不要用相同参数重复调用同一个工具。

## 工具使用
- 用户要求写文件时，使用 write_file。
- 用户要求运行代码、测试或查看目录时，使用 shell_exec。
- 用户要求"重复/echo"文本时，使用 echo 工具。

## 可用技能
- **web-dev**: 前端网页开发与实时预览。
触发条件：当用户要求写网页、设计主页、修改前端界面时。
⚠️ 强制要求：一旦触发该条件，第一件事必须调用 load_skill("web-dev") 获取高级前端工程师的开发规范，然后再开始行动。`, workspaceDir)


}


// BuildSystemPrompt 把多个片段拼成一个完整的 system prompt。
// 后面 Memory 摘要、Skill 目录都会作为片段传进来。
func BuildSystemPrompt(parts ...string) string {
	var nonEmpty []string
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			nonEmpty = append(nonEmpty, p)
		}
	}
	return strings.Join(nonEmpty, "\n\n")
}