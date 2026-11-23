package llm

import (
	"context"
	"fmt"
	"os"

	openai "github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

const defaultModel = "deepseek-ai/DeepSeek-V3"

type Client struct {
	api openai.Client
}


func New() (*Client, error) {
	apiKey := os.Getenv("SILICONFLOW_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("请先设置环境变量 SILICONFLOW_API_KEY")
	}

	opts := []option.RequestOption{
		option.WithAPIKey(apiKey),
		option.WithBaseURL("https://api.siliconflow.cn/v1"),
	}
	if base := os.Getenv("SILICONFLOW_BASE_URL"); base != "" {
		opts = append(opts, option.WithBaseURL(base))
	}

	return &Client{api: openai.NewClient(opts...)}, nil
}


func (c *Client) Chat(
	ctx context.Context,
	messages []openai.ChatCompletionMessageParamUnion,
	tools []openai.ChatCompletionToolUnionParam,
) (openai.ChatCompletionMessage, error) {
	params := openai.ChatCompletionNewParams{
		Model:    defaultModel,
		Messages: messages,
	}
	if len(tools) > 0 {
		params.Tools = tools
	}

	resp, err := c.api.Chat.Completions.New(ctx, params)
	if err != nil {
		return openai.ChatCompletionMessage{}, fmt.Errorf("chat completion: %w", err)
	}
	if len(resp.Choices) == 0 {
		return openai.ChatCompletionMessage{}, fmt.Errorf("empty choices")
	}
	return resp.Choices[0].Message, nil
}

// 长期记忆总结
// 在 client.go 追加这个辅助方法
func (c *Client) Summarize(ctx context.Context, dialogue string) (string, error) {
	prompt := fmt.Sprintf(`请用一句话（不超过50个字）总结以下对话的核心内容、用户偏好或完成的任务：
%s`, dialogue)

	params := openai.ChatCompletionNewParams{
		Model: defaultModel,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt),
		},
	}

	resp, err := c.api.Chat.Completions.New(ctx, params)
	if err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("empty choices")
	}
	return resp.Choices[0].Message.Content, nil
}