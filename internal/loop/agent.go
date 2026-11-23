package loop

import (
	"context"
	"encoding/json"
	"fmt"

	openai "github.com/openai/openai-go/v3"
)

// LLM 依赖接口，参数类型对齐官方 SDK
type LLM interface {
	Chat(
		ctx context.Context,
		messages []openai.ChatCompletionMessageParamUnion,
		tools []openai.ChatCompletionToolUnionParam,
	) (openai.ChatCompletionMessage, error)
}

type Agent struct {
	LLM     LLM
	Tools   map[string]Tool
	MaxIter int
}

func New(llm LLM, maxIter int) *Agent {
	return &Agent{LLM: llm, Tools: map[string]Tool{}, MaxIter: maxIter}
}

func (a *Agent) RegisterTool(t Tool) {
	a.Tools[t.Name()] = t
}

// toolDefs 把自定义 Tool 转成官方 SDK 的函数工具定义
func (a *Agent) toolDefs() []openai.ChatCompletionToolUnionParam {
	defs := make([]openai.ChatCompletionToolUnionParam, 0, len(a.Tools))
	for _, t := range a.Tools {
		defs = append(defs, openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
			Name:        t.Name(),
			Description: openai.String(t.Description()), // Description 是指针类型
			Parameters:  openai.FunctionParameters(t.Parameters()),
		}))
	}
	return defs
}

// Run 在 Session 上执行一轮对话
func (a *Agent) Run(ctx context.Context, sess *Session, userMsg string) (string, error) {
	sess.AddMessage(openai.UserMessage(userMsg))

	for i := 0; i < a.MaxIter; i++ {
		reply, err := a.LLM.Chat(ctx, sess.Messages, a.toolDefs())
		if err != nil {
			return "", err
		}

		// 把 assistant 回复加入历史。官方 SDK 提供 ToParam() 做类型转换。
		sess.AddMessage(reply.ToParam())

		if len(reply.ToolCalls) == 0 {
			return reply.Content, nil
		}

		for _, tc := range reply.ToolCalls {
			out := a.executeTool(ctx, tc)
			sess.AddMessage(openai.ToolMessage(out, tc.ID))
		}
	}

	return "", fmt.Errorf("超过最大迭代次数 %d", a.MaxIter)
}

func (a *Agent) executeTool(ctx context.Context, tc openai.ChatCompletionMessageToolCallUnion) string {
	tool, ok := a.Tools[tc.Function.Name]
	if !ok {
		return fmt.Sprintf("错误：未知工具 %q", tc.Function.Name)
	}
	out, err := tool.Execute(ctx, json.RawMessage(tc.Function.Arguments))
	if err != nil {
		return fmt.Sprintf("错误：%v", err)
	}
	return out
}