package loop

import "github.com/openai/openai-go/v3"

type Session struct {
	Messages []openai.ChatCompletionMessageParamUnion
	Len      int
}

func NewSession(systemPrompt string) *Session {
	s := &Session{
		Messages: []openai.ChatCompletionMessageParamUnion{},
		Len:      0,
	}
	if systemPrompt != "" {
		s.AddMessage(openai.SystemMessage(systemPrompt))
	}
	return s
}

func (s *Session) AddMessage(item openai.ChatCompletionMessageParamUnion) {
	s.Messages = append(s.Messages, item)
	s.Len++
}