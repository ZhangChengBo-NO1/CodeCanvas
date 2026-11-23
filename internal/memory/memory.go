package memory

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/openai/openai-go/v3"
)

// Entry 是一条记忆
type Entry struct {
	Timestamp time.Time `json:"timestamp"`
	Summary   string    `json:"summary"`
}

type Store struct {
	FilePath string
}

func NewStore(filePath string) *Store {
	return &Store{FilePath: filePath}
}

// LoadRecent 读取最近的 N 条记忆，拼成一个字符串，用于注入 System Prompt
func (s *Store) LoadRecent(n int) string {
	file, err := os.Open(s.FilePath)
	if err != nil {
		return "" // 文件不存在说明没有记忆
	}
	defer file.Close()

	var entries []Entry
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var e Entry
		if err := json.Unmarshal(scanner.Bytes(), &e); err == nil {
			entries = append(entries, e)
		}
	}

	// 取最后 n 条
	start := len(entries) - n
	if start < 0 {
		start = 0
	}

	var sb strings.Builder
	for _, e := range entries[start:] {
		sb.WriteString(fmt.Sprintf("- [%s] %s\n", e.Timestamp.Format("2006-01-02 15:04"), e.Summary))
	}
	return sb.String()
}

// Append 追加一条新记忆
func (s *Store) Append(summary string) error {
	entry := Entry{
		Timestamp: time.Now(),
		Summary:   summary,
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	// 确保目录存在
	dir := "memory"
	os.MkdirAll(dir, 0755)

	f, err := os.OpenFile(s.FilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(string(data) + "\n")
	return err
}

// DialogueToText 把消息历史提取成干净的对话文本用于总结
func DialogueToText(messages []openai.ChatCompletionMessageParamUnion) string {
	var sb strings.Builder
	for _, m := range messages {
		// 忽略 system prompt
		if m.OfSystem != nil {
			continue
		}
		
		// 提取用户消息
		if m.OfUser != nil {
			if m.OfUser.Content.OfString.Value != "" {
				sb.WriteString(fmt.Sprintf("用户: %s\n", m.OfUser.Content.OfString.Value))
			}
		}
		
		// 提取 Assistant 消息
		if m.OfAssistant != nil {
			if m.OfAssistant.Content.OfString.Value != "" {
				sb.WriteString(fmt.Sprintf("AI: %s\n", m.OfAssistant.Content.OfString.Value))
			}
			// 对于工具调用，简单记录一下调用了什么
			if len(m.OfAssistant.ToolCalls) > 0 {
				for _, tc := range m.OfAssistant.ToolCalls {
					sb.WriteString(fmt.Sprintf("[调用工具: %s]\n", tc.GetFunction().Name))
				}
			}
		}
		
		// 提取 Tool 消息
		if m.OfTool != nil {
			if m.OfTool.Content.OfString.Value != "" {
				// 工具输出太长的话截断一下
				out := m.OfTool.Content.OfString.Value
				if len(out) > 200 {
					out = out[:200] + "..."
				}
				sb.WriteString(fmt.Sprintf("[工具结果: %s]\n", out))
			}
		}
	}
	return sb.String()
}